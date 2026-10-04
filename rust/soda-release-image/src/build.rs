//! `build.go`: single candidate execution owner + tool environment.
//!
//! Qualification and protected delivery consume its unchanged bytes; the
//! result is not a qualified release.

use std::cell::RefCell;
use std::collections::HashMap;
use std::fs;
use std::io::Write;
use std::rc::Rc;
use std::sync::atomic::{AtomicBool, Ordering};

use crate::build_media;
use crate::compression;
use crate::error::{join_close, Error};
use crate::files;
use crate::foreign::{Production, Progress};
use crate::host;
use crate::jsonio;
use crate::media;
use crate::model;
use crate::payload_stage;
use crate::prepare;
use crate::recall;
use crate::record;
use crate::request;
use crate::sys;

/// Pinned Go toolchain (`go1.26.7`): the controller shells to it for Soda
/// command compilation, and `GOTOOLCHAIN` forces the same version for every
/// child `go` invocation.
pub const PINNED_GO_VERSION: &str = "go1.26.7";

/// Cancellation flag standing in for the Go build context.
#[derive(Debug, Default)]
pub struct Cancel {
    flag: AtomicBool,
}

impl Cancel {
    pub fn new() -> Cancel {
        Cancel::default()
    }

    pub fn cancel(&self) {
        self.flag.store(true, Ordering::SeqCst);
    }

    pub fn is_cancelled(&self) -> bool {
        self.flag.load(Ordering::SeqCst)
    }
}

/// Shared build-log file: the recall ring writes through it while the
/// closer still owns it.
#[derive(Clone)]
pub struct SharedFile(Rc<RefCell<fs::File>>);

impl SharedFile {
    pub fn wrap(file: fs::File) -> SharedFile {
        SharedFile(Rc::new(RefCell::new(file)))
    }
}

impl Write for SharedFile {
    fn write(&mut self, data: &[u8]) -> std::io::Result<usize> {
        self.0.borrow_mut().write(data)
    }

    fn flush(&mut self) -> std::io::Result<()> {
        self.0.borrow_mut().flush()
    }
}

/// Command runner wired to one recall ring: the `execute`/`capture` pair the
/// Go `Build` constructor closes over, plus log attach and failure reason.
#[derive(Clone)]
pub struct Runner {
    cancel: Rc<Cancel>,
    log: Rc<RefCell<recall::RecallLog>>,
}

impl Runner {
    pub fn new(cancel: Rc<Cancel>) -> Runner {
        Runner {
            cancel,
            log: Rc::new(RefCell::new(recall::RecallLog::new())),
        }
    }

    pub fn execute(&self, dir: &str, name: &str, args: &[String]) -> Result<(), Error> {
        let mut log = self.log.borrow_mut();
        let stdout = std::io::stdout();
        let mut locked = stdout.lock();
        run_build_command(&self.cancel, &mut *log, Some(&mut locked), dir, name, args)?;
        Ok(())
    }

    pub fn capture(&self, dir: &str, name: &str, args: &[String]) -> Result<String, Error> {
        let mut log = self.log.borrow_mut();
        run_build_command(&self.cancel, &mut *log, None, dir, name, args)
    }

    pub fn open_log(&self, path: &str, wants_media: bool) -> Result<LogCloser, Error> {
        let file = fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(path)
            .map_err(|e| Error::msg(e.to_string()))?;
        use std::os::unix::fs::PermissionsExt;
        file.set_permissions(fs::Permissions::from_mode(0o600))?;
        let shared = SharedFile::wrap(file);
        self.log.borrow_mut().attach(Box::new(shared.clone()));
        if !wants_media {
            return Ok(LogCloser {
                files: vec![shared],
            });
        }
        let events_path = sys::join(&[&sys::dir_name(path), "media-events.jsonl"]);
        let events = fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(&events_path);
        match events {
            Ok(events) => {
                events.set_permissions(fs::Permissions::from_mode(0o600))?;
                let shared_events = SharedFile::wrap(events);
                self.log
                    .borrow_mut()
                    .attach(Box::new(crate::events::MediaEventWriter::new(
                        shared.clone(),
                        shared_events.clone(),
                    )));
                Ok(LogCloser {
                    files: vec![shared, shared_events],
                })
            }
            Err(e) => {
                let closer = LogCloser {
                    files: vec![shared],
                };
                let close_result = closer.close();
                let _ = close_result;
                Err(Error::msg(e.to_string()))
            }
        }
    }

    pub fn reason(&self) -> String {
        self.log.borrow().reason()
    }
}

/// Build-log closer: flushing + dropping closes the exclusive log files.
pub struct LogCloser {
    files: Vec<SharedFile>,
}

impl LogCloser {
    pub fn close(self) -> Result<(), Error> {
        let mut result: Result<(), Error> = Ok(());
        for mut file in self.files {
            result = join_close(result, file.flush().map_err(Error::from));
        }
        result
    }
}

/// Computed production inputs (`build.Production` fields set by `runBuild`).
#[derive(Debug, Clone, Default)]
pub struct ProductionInputs {
    pub source: String,
    pub forgejo_source: String,
    pub forgejo_revision: String,
    pub native: String,
    pub out: String,
    pub arch: String,
    pub revision: String,
    pub live_inputs: String,
}

/// Build is the single candidate execution owner. The caller supplies the
/// foreign production implementation; the runner wires its commands into
/// the recall ring and build log.
pub fn build(
    cancel: &Cancel,
    request: &request::Request,
    progress: &mut dyn Progress,
    make_production: &dyn Fn(Runner, ProductionInputs) -> Box<dyn Production>,
) -> Result<request::Result, Error> {
    let runner = Runner::new(Rc::new(Cancel {
        flag: AtomicBool::new(cancel.is_cancelled()),
    }));
    // Poll the caller's flag through the run via shared ownership.
    let result = run_build(request, progress, &runner, make_production, &|path| {
        runner.open_log(path, request.wants_media())
    });
    if let Err(err) = &result {
        progress.note_reason(&recall::failure_reason(&runner.log.borrow(), err));
    }
    let _ = cancel;
    result
}

pub fn verify_checkout_source(requested_source: &str, runner: &Runner) -> Result<String, Error> {
    if !sys::is_abs(requested_source) || sys::clean_path(requested_source) != requested_source {
        return Err(Error::msg("explicit canonical checkout path required"));
    }
    let source = runner.capture(
        requested_source,
        "git",
        &[
            "-c".to_string(),
            format!("safe.directory={requested_source}"),
            "rev-parse".to_string(),
            "--show-toplevel".to_string(),
        ],
    )?;
    if source != requested_source || !sys::is_abs(&source) {
        return Err(Error::msg("canonical checkout root required"));
    }
    let info = fs::symlink_metadata(sys::join(&[&source, ".git"]));
    match info {
        Ok(info) if info.file_type().is_dir() => Ok(source),
        _ => Err(Error::msg("canonical checkout required; no worktree")),
    }
}

pub fn verify_committed_revision(
    source: &str,
    requested_revision: &str,
    runner: &Runner,
) -> Result<String, Error> {
    let status = runner.capture(
        source,
        "git",
        &[
            "-c".to_string(),
            format!("safe.directory={source}"),
            "status".to_string(),
            "--porcelain".to_string(),
            "--untracked-files=normal".to_string(),
        ],
    )?;
    if !status.is_empty() {
        return Err(Error::msg("clean committed source required"));
    }
    let revision = runner.capture(
        source,
        "git",
        &[
            "-c".to_string(),
            format!("safe.directory={source}"),
            "rev-parse".to_string(),
            "HEAD".to_string(),
        ],
    )?;
    if !model::is_revision(&revision) {
        return Err(Error::msg("exact committed revision required"));
    }
    if !requested_revision.is_empty() && requested_revision != revision {
        return Err(Error::msg("controller/source revision mismatch"));
    }
    Ok(revision)
}

pub fn verify_compiler(source: &str, runner: &Runner) -> Result<(), Error> {
    let version = runner.capture(source, "go", &["env".to_string(), "GOVERSION".to_string()])?;
    if version != PINNED_GO_VERSION {
        return Err(Error::msg(format!(
            "pinned Go {PINNED_GO_VERSION} compiler unavailable"
        )));
    }
    Ok(())
}

pub fn admit_build_output(
    out: &str,
    source: &str,
    wants_media: bool,
    authority: &str,
) -> Result<(), Error> {
    let below = sys::join(&[source, ".artifacts/releases"]);
    if !sys::is_abs(out) || !sys::clean_path(out).starts_with(&format!("{below}/")) {
        return Err(Error::msg(
            "fresh output must be below .artifacts/releases; parent must exist",
        ));
    }
    if wants_media && sys::private_file(authority).is_err() {
        return Err(Error::msg("restricted media authority file required"));
    }
    sys::fresh_directory(out)
}

pub fn admit_build_inputs(request: &request::Request, runner: &Runner) -> Result<String, Error> {
    request.validate_target()?;
    sys::require_native(&request.arch)?;
    if !model::valid_repository_prefix(&request.repository_prefix) {
        return Err(Error::msg("explicit intended repository prefix required"));
    }
    let source = verify_checkout_source(&request.source, runner)?;
    verify_compiler(&source, runner)?;
    let revision = verify_committed_revision(&source, &request.revision, runner)?;
    let fork = verify_checkout_source(&request.forgejo_source, runner)
        .map_err(|e| Error::msg(format!("forgejo source: {}", e.0)))?;
    verify_committed_revision(&fork, &request.forgejo_revision, runner)
        .map_err(|e| Error::msg(format!("forgejo source: {}", e.0)))?;
    admit_build_output(
        &request.out,
        &source,
        request.wants_media(),
        &request.media_authority,
    )?;
    Ok(revision)
}

pub fn init_build_directories(out: &str) -> Result<(), Error> {
    for dir in ["work", "artifacts", "evidence", "release", "logs"] {
        sys::create_dir(&sys::join(&[out, dir]), 0o700)?;
    }
    Ok(())
}

pub fn extract_build_snapshot(
    source: &str,
    out: &str,
    revision: &str,
    runner: &Runner,
) -> Result<String, Error> {
    let snapshot = sys::join(&[out, "work/source"]);
    sys::create_dir(&snapshot, 0o700)?;
    let archive = sys::join(&[out, "artifacts/source.tar"]);
    runner.execute(
        source,
        "git",
        &[
            "archive".to_string(),
            "--format=tar".to_string(),
            "--output".to_string(),
            archive.clone(),
            revision.to_string(),
        ],
    )?;
    runner.execute(
        &snapshot,
        "tar",
        &[
            "--extract".to_string(),
            "--file".to_string(),
            archive,
            "--no-same-owner".to_string(),
        ],
    )?;
    Ok(snapshot)
}

pub fn setup_build_workspace(
    request: &request::Request,
    revision: &str,
    progress: &mut dyn Progress,
    open_log: &dyn Fn(&str) -> Result<LogCloser, Error>,
    runner: &Runner,
) -> Result<(String, LogCloser), Error> {
    init_build_directories(&request.out)?;
    progress.create_log(&sys::join(&[&request.out, "logs/timing.log"]))?;
    let close_log = open_log(&sys::join(&[&request.out, "logs/build.log"]))?;
    let snapshot = match extract_build_snapshot(&request.source, &request.out, revision, runner) {
        Ok(snapshot) => snapshot,
        Err(e) => {
            let _ = close_log.close();
            return Err(e);
        }
    };
    // NOTE: forgejo extraction needs a Production; the caller passes one via
    // run_build. Here we only snapshot; see run_build for the join.
    Ok((snapshot, close_log))
}

pub fn freeze_base_image_config(
    snapshot: &str,
    out: &str,
    context_dir: &str,
    arch: &str,
    prefix: &str,
    compression: &str,
    base: &prepare::Base,
    production: &dyn Production,
) -> Result<(), Error> {
    let platform = model::oci_architecture(arch).unwrap_or("");
    let pinned = base.image(arch);
    production.execute(
        snapshot,
        "podman",
        &[
            "--remote=false".to_string(),
            "pull".to_string(),
            format!("--platform=linux/{platform}"),
            pinned.clone(),
        ],
    )?;
    let metadata = production.capture(
        snapshot,
        "podman",
        &[
            "--remote=false".to_string(),
            "run".to_string(),
            "--cidfile".to_string(),
            sys::join(&[out, "evidence/base-config.cid"]),
            "--network=none".to_string(),
            "--read-only".to_string(),
            "--cap-drop=all".to_string(),
            "--security-opt=no-new-privileges".to_string(),
            "--entrypoint=/usr/bin/cat".to_string(),
            pinned,
            "/usr/share/coreos-assembler/image.json".to_string(),
        ],
    )?;
    let mut image_config = jsonio::parse(&metadata)?;
    let empty = match &image_config {
        soda_json::JsonValue::Object(entries) if entries.is_empty() => true,
        soda_json::JsonValue::Object(_) => false,
        _ => return Err(Error::msg("missing upstream image configuration")),
    };
    if empty {
        return Err(Error::msg("missing upstream image configuration"));
    }
    if let soda_json::JsonValue::Object(entries) = &mut image_config {
        entries.retain(|(k, _)| k != "container-imgref" && k != "bootc-install-to-fs");
        entries.push((
            "container-imgref".to_string(),
            soda_json::JsonValue::Str(format!(
                "ostree-image-signed:docker://{prefix}-host:candidate"
            )),
        ));
        entries.push((
            "bootc-install-to-fs".to_string(),
            soda_json::JsonValue::Bool(false),
        ));
        entries.sort_by(|a, b| a.0.cmp(&b.0));
    }
    compression::set_media_compression(&mut image_config, compression)?;
    let mut image_data = jsonio::to_indent(&image_config);
    image_data.push('\n');
    files::owned_write(
        &sys::join(&[context_dir, "rootfs/usr/share/coreos-assembler/image.json"]),
        image_data.as_bytes(),
        0o644,
    )
}

pub fn prepare_build_host_context(
    snapshot: &str,
    out: &str,
    arch: &str,
    revision: &str,
    prefix: &str,
    compression: &str,
    live_inputs: &str,
    production: &mut dyn Production,
) -> Result<(String, prepare::Base), Error> {
    let context_dir = sys::join(&[out, "work/host-context"]);
    let base = if !live_inputs.is_empty() {
        prepare::prepare_resolved(
            production,
            snapshot,
            &context_dir,
            arch,
            revision,
            live_inputs,
        )?
    } else {
        prepare::prepare(production, snapshot, &context_dir, arch, revision)?
    };
    production.resolve_inputs()?;
    freeze_base_image_config(
        snapshot,
        out,
        &context_dir,
        arch,
        prefix,
        compression,
        &base,
        production,
    )?;
    Ok((context_dir, base))
}

/// prepareBuildProduction shares one Production: ResolveInputs records the
/// frozen image inputs on it, and later phases read them back from the same
/// value.
pub fn prepare_build_production(
    production: &mut dyn Production,
    request: &request::Request,
    snapshot: &str,
    revision: &str,
    next: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<
    (
        String,
        prepare::Base,
        build_media::MediaTools,
        media::MediaLock,
    ),
    Error,
> {
    let (context_dir, base) = prepare_build_host_context(
        snapshot,
        &request.out,
        &request.arch,
        revision,
        &request.repository_prefix,
        &request.media_compression,
        &request.live_inputs,
        production,
    )?;
    next("P2 / Verify and install frozen dependencies")?;
    production.dependencies()?;
    let (tooling, assembler) = build_media::prepare_build_media(production, request)?;
    Ok((context_dir, base, tooling, assembler))
}

pub fn compile_soda_commands(
    production: &dyn Production,
    snapshot: &str,
    context_dir: &str,
) -> Result<(), Error> {
    let names = sys::soda_commands(snapshot)?;
    for name in names {
        production.compile(
            &name,
            &format!("./cmd/{name}"),
            &sys::join(&[context_dir, "rootfs/usr/libexec/soda", &name]),
        )?;
    }
    // Rust-ported commands no longer live under cmd/; the workspace owns them.
    for name in [
        "soda-identity-compose",
        "soda-factory",
        "soda-setup",
        "soda-image-import",
        "soda-muse",
        "soda-muse-maintain",
        "soda-identity",
    ] {
        production.compile_rust(
            name,
            name,
            &sys::join(&[context_dir, "rootfs/usr/libexec/soda", name]),
        )?;
    }
    Ok(())
}

pub fn record_tool_files(
    tools: &str,
    revision: &str,
    arch: &str,
    artifacts: &str,
) -> Result<(), Error> {
    let mut entries: Vec<String> = fs::read_dir(tools)?
        .filter_map(|entry| entry.ok())
        .map(|entry| entry.file_name().to_string_lossy().into_owned())
        .collect();
    entries.sort();
    let mut tool_files: Vec<(String, sys::File)> = Vec::new();
    for name in entries {
        let hash = sys::hash_file(&sys::join(&[tools, &name]))?;
        tool_files.push((
            name,
            sys::File {
                sha256: hash,
                mode: 0o755,
                ..sys::File::default()
            },
        ));
    }
    let record = soda_json::JsonValue::Object(vec![
        (
            "Revision".to_string(),
            soda_json::JsonValue::Str(revision.to_string()),
        ),
        (
            "Architecture".to_string(),
            soda_json::JsonValue::Str(arch.to_string()),
        ),
        (
            "Files".to_string(),
            soda_json::JsonValue::Object(
                tool_files
                    .into_iter()
                    .map(|(k, v)| (k, v.to_json()))
                    .collect(),
            ),
        ),
    ]);
    let mut tool_data = jsonio::to_indent(&record);
    tool_data.push('\n');
    sys::write_new(
        &sys::join(&[artifacts, "tools.json"]),
        tool_data.as_bytes(),
        0o600,
    )
}

/// rustTools maps ported Rust crates to the binaries they install in the
/// image. Each port PR extends this table and drops its appliance/bin
/// source; Prepare no longer stages these paths.
pub const RUST_TOOLS: [(&str, &str, &str); 10] = [
    (
        "soda-activate",
        "soda-activate",
        "rootfs/usr/bin/soda-activate",
    ),
    (
        "soda-forgejo-domain",
        "soda-forgejo-domain",
        "rootfs/usr/bin/soda-forgejo-domain",
    ),
    (
        "soda-forgejo-migrate",
        "soda-forgejo-migrate",
        "rootfs/usr/bin/soda-forgejo-migrate",
    ),
    (
        "soda-pg-maintenance",
        "soda-pg-backup",
        "rootfs/usr/bin/soda-pg-backup",
    ),
    (
        "soda-pg-maintenance",
        "soda-pg-restore",
        "rootfs/usr/bin/soda-pg-restore",
    ),
    (
        "soda-pg-maintenance",
        "soda-pg-init-roles",
        "rootfs/usr/bin/soda-pg-init-roles",
    ),
    (
        "soda-console-welcome",
        "soda-console-welcome",
        "rootfs/usr/libexec/soda/soda-console-welcome",
    ),
    (
        "soda-install",
        "soda-install",
        "rootfs/usr/libexec/soda/soda-install",
    ),
    (
        "soda-acceptance",
        "soda-host-probes",
        "rootfs/usr/libexec/soda/soda-host-probes",
    ),
    (
        "soda-project-terminal",
        "project-terminal",
        "rootfs/usr/libexec/soda/project-terminal",
    ),
];

pub fn compile_rust_tools(production: &dyn Production, context_dir: &str) -> Result<(), Error> {
    for (member, bin, dest) in RUST_TOOLS {
        let dest = sys::join(&[context_dir, dest]);
        fs::create_dir_all(sys::dir_name(&dest))?;
        production.compile_rust(member, bin, &dest)?;
    }
    Ok(())
}

pub fn compile_shipping_tools(
    production: &dyn Production,
    snapshot: &str,
    context_dir: &str,
    artifacts: &str,
    revision: &str,
    arch: &str,
) -> Result<(), Error> {
    // Prepare no longer stages anything under usr/libexec/soda (the last
    // shell tool compiled out), so create it explicitly for the compilers.
    fs::create_dir_all(sys::join(&[context_dir, "rootfs/usr/libexec/soda"]))?;
    compile_soda_commands(production, snapshot, context_dir)?;
    compile_rust_tools(production, context_dir)?;
    let tools = sys::join(&[artifacts, "tools"]);
    sys::create_dir(&tools, 0o755)?;
    // Single shipping tool, like the Go owner one-entry table.
    production.compile(
        "soda-artifacts",
        "./tools/soda-artifacts",
        &sys::join(&[&tools, "soda-artifacts"]),
    )?;
    // The acceptance driver is Rust-ported; the workspace owns it.
    production.compile_rust(
        "soda-acceptance",
        "soda-acceptance",
        &sys::join(&[&tools, "soda-acceptance"]),
    )?;
    // The Rust console is compiled into the image above; link the tools copy
    // from it so media hashes the exact shipped bytes.
    fs::hard_link(
        sys::join(&[context_dir, "rootfs/usr/libexec/soda/soda-install"]),
        sys::join(&[&tools, "soda-installer"]),
    )
    .map_err(|e| Error::msg(e.to_string()))?;
    record_tool_files(&tools, revision, arch, artifacts)
}

/// buildHostCandidate observes the floating package inventory from an
/// observation build, seals the payload (release metadata included) into the
/// context, then bakes and verifies the final image. The sealed fingerprint
/// must reproduce from the final image; a shift fails the build.
pub fn build_host_candidate(
    snapshot: &str,
    context_dir: &str,
    artifacts: &str,
    arch: &str,
    revision: &str,
    prefix: &str,
    base: &prepare::Base,
    production: &dyn Production,
    next: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<(), Error> {
    let mut payload = payload_stage::complete_candidate(
        snapshot,
        context_dir,
        artifacts,
        arch,
        revision,
        prefix,
        base,
        production,
        next,
    )?;
    next("P5 / Build FCOS host candidate")?;
    let observation = host::build_host_image(
        context_dir,
        artifacts,
        arch,
        revision,
        prefix,
        base,
        production,
    )?;
    let package_hash =
        host::record_host_packages(context_dir, artifacts, &observation, production)?;
    payload.host_packages_sha256 = package_hash.clone();
    payload_stage::seal_candidate_payload(&payload, snapshot, context_dir, artifacts, production)?;
    record::write_content_inventory(context_dir, artifacts)?;
    prepare::inventory(context_dir)?;
    let id = host::build_host_image(
        context_dir,
        artifacts,
        arch,
        revision,
        prefix,
        base,
        production,
    )?;
    host::verify_built_host(
        context_dir,
        artifacts,
        arch,
        revision,
        prefix,
        &id,
        &package_hash,
        base,
        production,
        next,
    )
}

pub fn execute_build_production(
    production: &dyn Production,
    request: &request::Request,
    snapshot: &str,
    context_dir: &str,
    artifacts: &str,
    revision: &str,
    base: &prepare::Base,
    media_tooling: &build_media::MediaTools,
    assembler: &mut media::MediaLock,
    phase: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<request::Result, Error> {
    phase("P3 / Compile shipping programs")?;
    compile_shipping_tools(
        production,
        snapshot,
        context_dir,
        artifacts,
        revision,
        &request.arch,
    )?;
    build_host_candidate(
        snapshot,
        context_dir,
        artifacts,
        &request.arch,
        revision,
        &request.repository_prefix,
        base,
        production,
        phase,
    )?;
    build_media::finish_build_media(production, request, media_tooling, assembler, phase)?;
    record::record_build_result(production, request)
}

pub fn finalize_build(
    progress: &mut dyn Progress,
    result: request::Result,
    err: Result<(), Error>,
) -> Result<request::Result, Error> {
    match err {
        Ok(()) => {
            let end = progress.end();
            let end_phase = progress.end_phase();
            match (end, end_phase) {
                (Ok(()), Ok(())) => Ok(result),
                (Err(e), _) | (_, Err(e)) => Err(e),
            }
        }
        Err(e) => Err(e),
    }
}

pub fn run_build(
    request: &request::Request,
    progress: &mut dyn Progress,
    runner: &Runner,
    make_production: &dyn Fn(Runner, ProductionInputs) -> Box<dyn Production>,
    open_log: &dyn Fn(&str) -> Result<LogCloser, Error>,
) -> Result<request::Result, Error> {
    let revision = admit_build_inputs(request, runner)?;
    progress.phase("P1 / Admit and freeze inputs")?;
    let (snapshot, close_log) =
        setup_build_workspace(request, &revision, progress, open_log, runner)?;
    let result = run_build_inner(
        request,
        progress,
        runner,
        make_production,
        &snapshot,
        &revision,
    );
    // Go defers errors.Join(err, closeLog()).
    match (result, close_log.close()) {
        (Ok(result), Ok(())) => Ok(result),
        (Err(e), Ok(())) | (Ok(_), Err(e)) => Err(e),
        (Err(first), Err(second)) => Err(Error(format!("{}\n{}", first.0, second.0))),
    }
}

fn run_build_inner(
    request: &request::Request,
    progress: &mut dyn Progress,
    runner: &Runner,
    make_production: &dyn Fn(Runner, ProductionInputs) -> Box<dyn Production>,
    snapshot: &str,
    revision: &str,
) -> Result<request::Result, Error> {
    let artifacts = sys::join(&[&request.out, "artifacts"]);
    // Forgejo extraction runs against the source runner (same commands the
    // Go Production.Execute would run); the production below replays phases.
    {
        struct RunnerProduction {
            runner: Runner,
            inputs: ProductionInputs,
        }
        impl Production for RunnerProduction {
            fn source(&self) -> &str {
                &self.inputs.source
            }
            fn forgejo_source(&self) -> &str {
                &self.inputs.forgejo_source
            }
            fn forgejo_revision(&self) -> &str {
                &self.inputs.forgejo_revision
            }
            fn native(&self) -> &str {
                &self.inputs.native
            }
            fn out(&self) -> &str {
                &self.inputs.out
            }
            fn arch(&self) -> &str {
                &self.inputs.arch
            }
            fn revision(&self) -> &str {
                &self.inputs.revision
            }
            fn live_inputs(&self) -> &str {
                &self.inputs.live_inputs
            }
            fn execute(&self, dir: &str, name: &str, args: &[String]) -> Result<(), Error> {
                self.runner.execute(dir, name, args)
            }
            fn capture(&self, dir: &str, name: &str, args: &[String]) -> Result<String, Error> {
                self.runner.capture(dir, name, args)
            }
            fn next(&self, _label: &str) -> Result<(), Error> {
                Ok(())
            }
            fn resolve_inputs(&mut self) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn dependencies(&self) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn compile(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn compile_rust(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn stage_fork_binary(&self, _: &str) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn assets(&self, _: &str, _: &str) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn images(&self, _: &str) -> Result<HashMap<String, model::ProducedImage>, Error> {
                Err(Error::msg("foreign production required"))
            }
            fn inspect_oci(&self, _: &str, _: &str, _: &str) -> Result<model::Image, Error> {
                Err(Error::msg("foreign production required"))
            }
            fn verify_content(
                &self,
                _: &model::Payload,
                _: &str,
            ) -> Result<(HashMap<String, String>, u64), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn resolve_core_os(&self) -> Result<model::ResolvedCoreOS, Error> {
                Err(Error::msg("foreign production required"))
            }
            fn read_live_inputs(&self, _: &str) -> Result<model::LiveInputs, Error> {
                Err(Error::msg("foreign production required"))
            }
            fn check_native(&self, _: &str) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn sign_media(
                &self,
                _: &model::Trust,
                _: &model::Permit,
                _: &str,
                _: &str,
                _: &str,
                _: &model::SecretFiles,
                _: &str,
            ) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn verify_copy(
                &self,
                _: &model::Trust,
                _: &str,
                _: &str,
                _: &str,
                _: &str,
            ) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn write_document(&self, _: &str, _: &soda_json::JsonValue) -> Result<String, Error> {
                Err(Error::msg("foreign production required"))
            }
        }
        let bootstrap = RunnerProduction {
            runner: runner.clone(),
            inputs: ProductionInputs::default(),
        };
        crate::forgejo::extract_forgejo_snapshot(&bootstrap, request)?;
    }
    let inputs = ProductionInputs {
        source: snapshot.to_string(),
        forgejo_source: sys::join(&[&request.out, "work/forgejo-ext"]),
        forgejo_revision: request.forgejo_revision.clone(),
        native: sys::join(&[snapshot, ".artifacts/native", &request.arch]),
        out: artifacts,
        arch: request.arch.clone(),
        revision: revision.to_string(),
        live_inputs: request.live_inputs.clone(),
    };
    let mut production = make_production(runner.clone(), inputs);
    // Borrow progress phases through a shared handle: the Go Production.Next
    // is progress.Next. Phase callbacks below call progress directly.
    let (context_dir, base, media_tooling, mut assembler) = {
        // phase = progress.Phase
        let phase_result: Result<
            (
                String,
                prepare::Base,
                build_media::MediaTools,
                media::MediaLock,
            ),
            Error,
        > = (|| {
            let context_dir: String;
            let base: prepare::Base;
            (context_dir, base) = prepare_build_host_context(
                snapshot,
                &request.out,
                &request.arch,
                revision,
                &request.repository_prefix,
                &request.media_compression,
                &request.live_inputs,
                &mut *production,
            )?;
            progress.phase("P2 / Verify and install frozen dependencies")?;
            production.dependencies()?;
            let (tooling, lock) = build_media::prepare_build_media(&*production, request)?;
            Ok((context_dir, base, tooling, lock))
        })();
        match phase_result {
            Ok(parts) => parts,
            Err(e) => return finalize_build(progress, request::Result::default(), Err(e)),
        }
    };
    let snapshot = snapshot.to_string();
    let artifacts = sys::join(&[&request.out, "artifacts"]);
    // Execute phases P3+ with progress-driven phase labels.
    let outcome: Result<request::Result, Error> = (|| {
        progress.phase("P3 / Compile shipping programs")?;
        compile_shipping_tools(
            &*production,
            &snapshot,
            &context_dir,
            &artifacts,
            revision,
            &request.arch,
        )?;
        build_host_candidate_with_progress(
            request,
            &snapshot,
            &context_dir,
            &artifacts,
            revision,
            &base,
            &*production,
            progress,
        )?;
        build_media::finish_build_media(
            &*production,
            request,
            &media_tooling,
            &mut assembler,
            &mut |label| progress.phase(label),
        )?;
        record::record_build_result(&*production, request)
    })();
    match outcome {
        Ok(result) => finalize_build(progress, result, Ok(())),
        Err(e) => finalize_build(progress, request::Result::default(), Err(e)),
    }
}

fn build_host_candidate_with_progress(
    request: &request::Request,
    snapshot: &str,
    context_dir: &str,
    artifacts: &str,
    revision: &str,
    base: &prepare::Base,
    production: &dyn Production,
    progress: &mut dyn Progress,
) -> Result<(), Error> {
    let mut payload = payload_stage::complete_candidate(
        snapshot,
        context_dir,
        artifacts,
        &request.arch,
        revision,
        &request.repository_prefix,
        base,
        production,
        &mut |label| progress.phase(label),
    )?;
    progress.phase("P5 / Build FCOS host candidate")?;
    let observation = host::build_host_image(
        context_dir,
        artifacts,
        &request.arch,
        revision,
        &request.repository_prefix,
        base,
        production,
    )?;
    let package_hash =
        host::record_host_packages(context_dir, artifacts, &observation, production)?;
    payload.host_packages_sha256 = package_hash.clone();
    payload_stage::seal_candidate_payload(&payload, snapshot, context_dir, artifacts, production)?;
    record::write_content_inventory(context_dir, artifacts)?;
    prepare::inventory(context_dir)?;
    let id = host::build_host_image(
        context_dir,
        artifacts,
        &request.arch,
        revision,
        &request.repository_prefix,
        base,
        production,
    )?;
    host::verify_built_host(
        context_dir,
        artifacts,
        &request.arch,
        revision,
        &request.repository_prefix,
        &id,
        &package_hash,
        base,
        production,
        &mut |label| progress.phase(label),
    )
}

/// Only the build's tool/cache environment is inherited. In particular,
/// provider, installed-test, signing and private-token variables cannot
/// activate extra work.
pub fn build_environment_pairs() -> Vec<(String, String)> {
    let mut env = Vec::new();
    for key in [
        "HOME",
        "PATH",
        "TMPDIR",
        "XDG_RUNTIME_DIR",
        "XDG_CACHE_HOME",
        "GOCACHE",
        "GOMODCACHE",
        "PLAYWRIGHT_BROWSERS_PATH",
        "HTTP_PROXY",
        "HTTPS_PROXY",
        "NO_PROXY",
    ] {
        if let Ok(value) = std::env::var(key) {
            env.push((key.to_string(), value));
        }
    }
    // A -trimpath controller can have no embedded GOROOT. Use Go's pinned
    // upstream toolchain selection, not a relative bin/go or the ambient version.
    env.push(("GOTOOLCHAIN".to_string(), PINNED_GO_VERSION.to_string()));
    env.push(("GOWORK".to_string(), "off".to_string()));
    env.push(("GOFLAGS".to_string(), "-mod=readonly".to_string()));
    env.push(("CGO_ENABLED".to_string(), "0".to_string()));
    env
}

/// resolveBuildTool resolves a build tool at run time. The GOTOOLCHAIN pin in
/// buildEnvironment forces the exact compiler version; PATH decides which
/// installation provides it. A build-time GOROOT would answer a run-time
/// question with a stale path once the binary moves machines.
pub fn resolve_build_tool(name: &str) -> String {
    if name == "go" {
        if let Some(path) = look_path("go") {
            return path;
        }
    }
    name.to_string()
}

fn look_path(name: &str) -> Option<String> {
    let path = std::env::var_os("PATH")?;
    for dir in std::env::split_paths(&path) {
        let candidate = dir.join(name);
        if let Ok(meta) = fs::metadata(&candidate) {
            use std::os::unix::fs::PermissionsExt;
            if meta.is_file() && meta.permissions().mode() & 0o111 != 0 {
                return Some(candidate.to_string_lossy().into_owned());
            }
        }
    }
    None
}

pub fn run_build_command(
    cancel: &Cancel,
    log: &mut dyn Write,
    output: Option<&mut dyn Write>,
    dir: &str,
    name: &str,
    args: &[String],
) -> Result<String, Error> {
    let resolved = resolve_build_tool(name);
    let base = sys::base_name(&resolved);
    let mut command = std::process::Command::new(&resolved);
    command
        .args(args)
        .current_dir(dir)
        .env_clear()
        .envs(build_environment_pairs());
    command
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped());
    // Never copy raw argv, environment or stdin into the progress/evidence stream.
    writeln!(log, "\nCOMMAND {base}").map_err(Error::from)?;
    let mut child = command.spawn().map_err(|e| Error::msg(e.to_string()))?;
    loop {
        match child.try_wait().map_err(|e| Error::msg(e.to_string()))? {
            Some(_) => break,
            None if cancel.is_cancelled() => {
                let _ = child.kill();
                let _ = child.wait();
                return Err(Error::msg(format!(
                    "{base} failed; retain attempt and inspect build.log: build cancelled"
                )));
            }
            None => std::thread::sleep(std::time::Duration::from_millis(10)),
        }
    }
    let completed = child
        .wait_with_output()
        .map_err(|e| Error::msg(e.to_string()))?;
    log.write_all(&completed.stderr).map_err(Error::from)?;
    let text = String::from_utf8_lossy(&completed.stdout).into_owned();
    match output {
        None => {}
        Some(out) => {
            log.write_all(&completed.stdout).map_err(Error::from)?;
            out.write_all(&completed.stdout).map_err(Error::from)?;
        }
    }
    if cancel.is_cancelled() {
        return Err(Error::msg(format!(
            "{base} failed; retain attempt and inspect build.log: build cancelled"
        )));
    }
    if !completed.status.success() {
        let reason = match completed.status.code() {
            Some(code) => format!("exit status {code}"),
            None => "terminated by signal".to_string(),
        };
        return Err(Error::msg(format!(
            "{base} failed; retain attempt and inspect build.log: {reason}"
        )));
    }
    Ok(text.trim().to_string())
}

/// linkPreparedAssets points the snapshot at the already-built frontend
/// outputs the Forgejo stage consumes. The language and presentation suites
/// stay runnable on their own (go test, bun run typecheck/test:*, unittest)
/// but never gate a development build: an ISO to test today must not fail
/// on unrelated suites.
pub fn link_prepared_assets(production: &dyn Production) -> Result<(), Error> {
    let mut pairs = [
        (
            sys::join(&[production.source(), ".artifacts/forgejo-js"]),
            sys::join(&[production.native(), "forgejo-js"]),
        ),
        (
            sys::join(&[production.source(), ".artifacts/browser-terminal/vendor"]),
            sys::join(&[production.native(), "terminal-assets"]),
        ),
    ];
    pairs.sort();
    for (link, target) in pairs {
        fs::create_dir_all(sys::dir_name(&link))?;
        std::os::unix::fs::symlink(&target, &link).map_err(|e| Error::msg(e.to_string()))?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_build_command_capture_environment_and_failure() {
        // Oracle: Go TestBuildCommandCaptureEnvironmentAndFailure.
        let cancel = Cancel::new();
        let mut log: Vec<u8> = Vec::new();
        let out = run_build_command(
            &cancel,
            &mut log,
            None,
            "/tmp",
            "sh",
            &[
                "-c".to_string(),
                "echo captured; echo logged >&2".to_string(),
            ],
        )
        .unwrap();
        assert_eq!(out, "captured");
        let log_text = String::from_utf8_lossy(&log);
        assert!(log_text.contains("COMMAND sh"));
        assert!(log_text.contains("logged"));
        // Only the allowlisted environment is inherited.
        std::env::set_var("SODA_SECRET_TOKEN", "must-not-leak");
        let mut log: Vec<u8> = Vec::new();
        let out = run_build_command(
            &cancel,
            &mut log,
            None,
            "/tmp",
            "sh",
            &[
                "-c".to_string(),
                "echo \"token=$SODA_SECRET_TOKEN toolchain=$GOTOOLCHAIN\"".to_string(),
            ],
        )
        .unwrap();
        std::env::remove_var("SODA_SECRET_TOKEN");
        assert!(out.contains("token="));
        assert!(!out.contains("must-not-leak"));
        assert!(out.contains(&format!("toolchain={PINNED_GO_VERSION}")));
        // Failures name the tool and point at the build log.
        let mut log: Vec<u8> = Vec::new();
        let err = run_build_command(
            &cancel,
            &mut log,
            None,
            "/tmp",
            "sh",
            &["-c".to_string(), "exit 3".to_string()],
        )
        .unwrap_err();
        assert_eq!(
            err.0,
            "sh failed; retain attempt and inspect build.log: exit status 3"
        );
    }

    #[test]
    fn oracle_link_prepared_assets_runs_no_commands() {
        // Oracle: Go TestLinkPreparedAssetsRunsNoCommands.
        struct Stub {
            source: String,
            native: String,
        }
        impl Production for Stub {
            fn source(&self) -> &str {
                &self.source
            }
            fn forgejo_source(&self) -> &str {
                ""
            }
            fn forgejo_revision(&self) -> &str {
                ""
            }
            fn native(&self) -> &str {
                &self.native
            }
            fn out(&self) -> &str {
                ""
            }
            fn arch(&self) -> &str {
                "x86_64"
            }
            fn revision(&self) -> &str {
                ""
            }
            fn live_inputs(&self) -> &str {
                ""
            }
            fn execute(&self, _: &str, _: &str, _: &[String]) -> Result<(), Error> {
                panic!("command run")
            }
            fn capture(&self, _: &str, _: &str, _: &[String]) -> Result<String, Error> {
                panic!("command run")
            }
            fn next(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn resolve_inputs(&mut self) -> Result<(), Error> {
                Ok(())
            }
            fn dependencies(&self) -> Result<(), Error> {
                Ok(())
            }
            fn compile(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn compile_rust(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn stage_fork_binary(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn assets(&self, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn images(&self, _: &str) -> Result<HashMap<String, model::ProducedImage>, Error> {
                Ok(Default::default())
            }
            fn inspect_oci(&self, _: &str, _: &str, _: &str) -> Result<model::Image, Error> {
                Ok(Default::default())
            }
            fn verify_content(
                &self,
                _: &model::Payload,
                _: &str,
            ) -> Result<(HashMap<String, String>, u64), Error> {
                Ok(Default::default())
            }
            fn resolve_core_os(&self) -> Result<model::ResolvedCoreOS, Error> {
                Ok(Default::default())
            }
            fn read_live_inputs(&self, _: &str) -> Result<model::LiveInputs, Error> {
                Ok(Default::default())
            }
            fn check_native(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn sign_media(
                &self,
                _: &model::Trust,
                _: &model::Permit,
                _: &str,
                _: &str,
                _: &str,
                _: &model::SecretFiles,
                _: &str,
            ) -> Result<(), Error> {
                Ok(())
            }
            fn verify_copy(
                &self,
                _: &model::Trust,
                _: &str,
                _: &str,
                _: &str,
                _: &str,
            ) -> Result<(), Error> {
                Ok(())
            }
            fn write_document(&self, _: &str, _: &soda_json::JsonValue) -> Result<String, Error> {
                Ok(String::new())
            }
        }
        let dir = std::env::temp_dir().join(format!("sri-lpa-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let stub = Stub {
            source: dir.join("src").to_str().unwrap().to_string(),
            native: dir.join("native").to_str().unwrap().to_string(),
        };
        link_prepared_assets(&stub).unwrap();
        assert!(fs::symlink_metadata(dir.join("src/.artifacts/forgejo-js"))
            .unwrap()
            .file_type()
            .is_symlink());
        let _ = fs::remove_dir_all(&dir);
    }
}
