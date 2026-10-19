//! Explicit release maintenance. Native OSTree owns staging and activation.
use std::fs::{self, OpenOptions};
use std::io::Write;
use std::os::unix::fs::MetadataExt;
use std::os::unix::fs::OpenOptionsExt;
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};

use serde::{Deserialize, Serialize};
use soda_host::project::admission;

use crate::filesystem::{open_tool, verify_native, Tool};
use crate::release_inventory::{self, Entry};
use crate::release_stage::{self, Deployment};
use crate::support_root::Root;
use crate::terminal_support::{guest, retire_recovery_copies, trusted_read, SCRIPT};

const CLOSED: &str = "release operation incomplete; managed admission remains closed; inspect or recover this operation";
const RECORD: &str = "operation.json";
const LAST: &str = "last-operation";

#[derive(Debug, Clone, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Project {
    project_id: String,
    container_id: String,
    old_sha256: String,
    new_sha256: String,
    previous: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Operation {
    format: u32,
    source_release: String,
    target_release: String,
    source_deployment: String,
    source_serial: u32,
    target_deployment: String,
    source_sha256: String,
    target_sha256: String,
    target_payload_sha256: String,
    source_project_image: String,
    target_project_image: String,
    release_directory: String,
    qualification_fixture: bool,
    rollback: bool,
    phase: String,
    projects: Vec<Project>,
    extensions: crate::release_extensions::Frozen,
}

impl Operation {
    fn expected(&self) -> (&str, &str, &str, &str) {
        if self.rollback {
            (
                &self.source_release,
                &self.source_deployment,
                &self.source_sha256,
                &self.source_project_image,
            )
        } else {
            (
                &self.target_release,
                &self.target_deployment,
                &self.target_sha256,
                &self.target_project_image,
            )
        }
    }
    fn entries(&self) -> Vec<Entry> {
        self.projects
            .iter()
            .map(|p| Entry {
                project_id: p.project_id.clone(),
                container_id: p.container_id.clone(),
            })
            .collect()
    }
    fn validate_extension_binding(&self) -> Result<(), String> {
        self.extensions.validate()?;
        if self.source_release != self.extensions.source.id
            || self.target_release != self.extensions.target.id
            || self.source_sha256 != self.extensions.source.project_support.sha256
            || self.target_sha256 != self.extensions.target.project_support.sha256
            || self.source_project_image != self.extensions.source.project_support.project_image
            || self.target_project_image != self.extensions.target.project_support.project_image
        {
            return Err(CLOSED.into());
        }
        Ok(())
    }

    fn validate_release_shape(&self) -> Result<(), String> {
        use crate::release::is_hex_string;
        if self.source_release == self.target_release
            || self.source_deployment == self.target_deployment
            || ![
                &self.source_deployment,
                &self.target_deployment,
                &self.source_sha256,
                &self.target_sha256,
                &self.target_payload_sha256,
            ]
            .iter()
            .all(|v| is_hex_string(v, 64))
            || ![&self.source_project_image, &self.target_project_image]
                .iter()
                .all(|v| {
                    v.strip_prefix("sha256:")
                        .is_some_and(|s| is_hex_string(s, 64))
                })
            || self.projects.len() > 128
            || !Path::new(&self.release_directory).is_absolute()
            || !matches!(
                self.phase.as_str(),
                "admitted"
                    | "host_locked"
                    | "helpers_verified"
                    | "extensions_prepared"
                    | "activation_requested"
                    | "complete"
            )
        {
            return Err(CLOSED.into());
        }
        Ok(())
    }

    fn validate_project_rows(&self) -> Result<(), String> {
        use crate::release::is_hex_string;
        let mut ids = std::collections::BTreeSet::new();
        let mut containers = std::collections::BTreeSet::new();
        for p in &self.projects {
            if !soda_host::domain::valid_id(&p.project_id)
                || !is_hex_string(&p.container_id, 64)
                || !ids.insert(&p.project_id)
                || !containers.insert(&p.container_id)
                || p.old_sha256 != self.source_sha256
                || p.new_sha256 != self.target_sha256
                || p.previous
                    != format!(
                        "/usr/libexec/soda/.project-terminal.previous-{}",
                        self.source_sha256
                    )
            {
                return Err(CLOSED.into());
            }
        }
        Ok(())
    }

    fn validate(&self) -> Result<(), String> {
        if self.format != 2 {
            return Err("unsupported release operation format; expected 2".into());
        }
        self.validate_extension_binding()?;
        self.validate_release_shape()?;
        self.validate_project_rows()
    }
}

fn path(name: &str) -> PathBuf {
    Path::new(admission::DIRECTORY).join(name)
}
fn load(name: &str) -> Result<Operation, String> {
    load_at(Path::new(admission::DIRECTORY), name)
}
fn load_at(directory: &Path, name: &str) -> Result<Operation, String> {
    let bytes = trusted_read(&directory.join(name), 1 << 20)?;
    #[derive(Deserialize)]
    struct Format {
        format: u32,
    }
    let format: Format = serde_json::from_slice(&bytes).map_err(|_| CLOSED)?;
    if format.format != 2 {
        return Err("unsupported release operation format; expected 2".into());
    }
    let record: Operation = soda_host::json::decode_strict_as(&bytes).map_err(|_| CLOSED)?;
    record.validate()?;
    Ok(record)
}
fn sync_directory() -> Result<(), String> {
    fs::File::open(admission::DIRECTORY)
        .and_then(|f| f.sync_all())
        .map_err(|_| CLOSED.into())
}
fn save(record: &Operation, initial: bool) -> Result<(), String> {
    save_at(Path::new(admission::DIRECTORY), record, initial, 0)
}
fn save_at(directory: &Path, record: &Operation, initial: bool, owner: u32) -> Result<(), String> {
    record.validate()?;
    let destination = directory.join("operation.next");
    match fs::symlink_metadata(&destination) {
        Ok(info) => {
            // This is an uncommitted scratch write, not an operation boundary.
            // It may be partial after interruption; never replace an unsafe entry.
            if !info.is_file()
                || info.uid() != owner
                || info.mode() & 0o022 != 0
                || info.len() > 1 << 20
            {
                return Err(CLOSED.into());
            }
            fs::remove_file(&destination).map_err(|_| CLOSED)?;
        }
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
        Err(_) => return Err(CLOSED.into()),
    }
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC)
        .open(&destination)
        .map_err(|_| CLOSED)?;
    file.write_all(&serde_json::to_vec(record).map_err(|_| CLOSED)?)
        .and_then(|_| file.sync_all())
        .map_err(|_| CLOSED)?;
    if initial {
        fs::hard_link(&destination, directory.join(RECORD)).map_err(|_| CLOSED)?;
        fs::remove_file(destination).map_err(|_| CLOSED)?;
    } else {
        fs::rename(destination, directory.join(RECORD)).map_err(|_| CLOSED)?;
    }
    fs::File::open(directory)
        .and_then(|f| f.sync_all())
        .map_err(|_| CLOSED.into())
}
fn phase(record: &mut Operation, value: &str) -> Result<(), String> {
    record.phase = value.into();
    save(record, false)
}
fn finish(mut record: Operation, deadline: Instant) -> Result<(), String> {
    record
        .extensions
        .verify_packages(record.rollback, deadline)?;
    // The retained native source deployment supplies coordinated rollback.
    // Project-local copies are only needed until this verified finish boundary.
    release_inventory::revalidate(&record.entries(), deadline)?;
    for project in &record.projects {
        let root = Root::open(&project.project_id, deadline)?;
        if root.observation.id != project.container_id {
            return Err(CLOSED.into());
        }
        retire_recovery_copies(
            &root,
            record.expected().2,
            &project.old_sha256,
            &project.new_sha256,
            deadline,
        )?;
        root.close()?;
    }
    phase(&mut record, "complete")?;
    fs::rename(path(RECORD), path(LAST)).map_err(|_| CLOSED)?;
    sync_directory()?;
    println!(
        "Release {} core verified; extension readiness follows maintenance lock release{}",
        record.expected().0,
        if record.qualification_fixture {
            " (development qualification fixture; no supported release claim)"
        } else {
            ""
        }
    );
    Ok(())
}

struct Options {
    directory: Option<PathBuf>,
    check: bool,
    recover: bool,
    rollback: bool,
    fixture: bool,
}
fn options(args: &[String]) -> Result<Options, String> {
    let mut out = Options {
        directory: None,
        check: false,
        recover: false,
        rollback: false,
        fixture: false,
    };
    let mut args = args.iter();
    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--release-dir"=>out.directory=Some(PathBuf::from(args.next().ok_or("--release-dir needs a path")?)),
            "--check"=>out.check=true,
            "--recover"=>out.recover=true,
            "--rollback"=>out.rollback=true,
            "--qualification-fixture"=>out.fixture=true,
            _=>return Err("release accepts --release-dir PATH, --check, --recover, --rollback and --qualification-fixture".into()),
        }
    }
    if out.check && (out.recover || out.rollback)
        || out.rollback && (out.directory.is_some() || out.fixture)
    {
        return Err("select one release operation".into());
    }
    Ok(out)
}

fn no_other_repairs() -> Result<(), String> {
    for entry in fs::read_dir(admission::DIRECTORY).map_err(|_| CLOSED)? {
        let entry = entry.map_err(|_| CLOSED)?;
        if entry.file_name() != RECORD && entry.path().extension().is_some_and(|v| v == "json") {
            return Err("a Project repair must be reconciled before release maintenance".into());
        }
    }
    Ok(())
}

fn observe(
    entries: &[Entry],
    source: &str,
    target: &str,
    idle: bool,
    deadline: Instant,
) -> Result<(), String> {
    release_inventory::revalidate(entries, deadline)?;
    for entry in entries {
        let root = Root::open(&entry.project_id, deadline)?;
        if root.observation.id != entry.container_id {
            return Err(CLOSED.into());
        }
        let actual = guest(
            &root,
            if idle { "idle" } else { "observe" },
            target,
            deadline,
        )
        .map_err(|_| {
            format!(
                "Project {} has active or uncertain managed work or unreadable support",
                entry.project_id
            )
        })?;
        if actual != source && actual != target {
            return Err(format!(
                "Project {} support is outside the admitted source/target pair",
                entry.project_id
            ));
        }
        root.close()?;
    }
    Ok(())
}

fn replace(record: &Operation, tool: &Tool, deadline: Instant) -> Result<(), String> {
    let entries = record.entries();
    observe(
        &entries,
        &record.source_sha256,
        &record.target_sha256,
        true,
        deadline,
    )?;
    let expected = record.expected().2;
    for p in &record.projects {
        let root = Root::open(&p.project_id, deadline)?;
        if root.observation.id != p.container_id {
            return Err(CLOSED.into());
        }
        let actual = guest(&root, "idle", expected, deadline)?;
        if actual != expected {
            root.install(SCRIPT, expected, &actual, tool, deadline)?;
        }
        if guest(&root, "observe", expected, deadline)? != expected {
            return Err(CLOSED.into());
        }
        root.close()?;
    }
    observe(&entries, expected, expected, true, deadline)
}

fn tool(path: &str, digest: &str) -> Result<Tool, String> {
    let (dir, name) = path.rsplit_once('/').ok_or("invalid native helper path")?;
    let source = open_tool(dir, name)?;
    if source.size == 0 || source.size > 32 << 20 {
        return Err("invalid native helper size".into());
    }
    verify_native(&source, digest)?;
    Ok(source)
}

fn future_image(image: &str, digest: &str, deadline: Instant) -> Result<(), String> {
    let out = crate::command::podman(
        &[
            "run",
            "--rm",
            "--pull=never",
            "--network=none",
            "--read-only",
            "--cap-drop=all",
            "--security-opt=no-new-privileges",
            "--entrypoint=/usr/bin/sha256sum",
            image,
            "/usr/libexec/soda/project-terminal",
        ],
        deadline,
    )
    .map_err(|_| "future-Project image support unavailable")?;
    let text = std::str::from_utf8(&out).map_err(|_| "invalid image support observation")?;
    if text.split_whitespace().collect::<Vec<_>>() != [digest, "/usr/libexec/soda/project-terminal"]
    {
        return Err("future-Project image helper differs from release".into());
    }
    Ok(())
}

fn verify_active(record: &Operation, deadline: Instant) -> Result<(), String> {
    let (release, commit, digest, image) = record.expected();
    let state = release_stage::inspect(deadline)?;
    if state.booted.checksum != commit || state.staged.is_some() {
        return Err(CLOSED.into());
    }
    let installed = crate::release_target::installed_source(deadline)?;
    if installed.release_id != release
        || installed.helper_sha256 != digest
        || installed.deployment != commit
    {
        return Err(CLOSED.into());
    }
    // Also bind the selected profile, not merely a previously imported image.
    let installed: soda_release_deliver::payload::Payload =
        serde_json::from_slice(&trusted_read(Path::new(crate::RELEASE_PATH), 4 << 20)?)
            .map_err(|_| CLOSED)?;
    if soda_release_deliver::payload::Role::ProjectOs
        .binding(&installed.images)
        .config
        != image
    {
        return Err(CLOSED.into());
    }
    observe(&record.entries(), digest, digest, true, deadline)?;
    future_image(image, digest, deadline)
}

#[derive(Debug)]
pub(crate) enum Failure {
    BeforeCommit(String),
    CoreCommitted(String),
}
impl std::fmt::Display for Failure {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::BeforeCommit(message) => write!(f, "{message}"),
            Self::CoreCommitted(message) => write!(
                f,
                "core release committed; extension readiness incomplete: {message}"
            ),
        }
    }
}
impl Failure {
    pub(crate) fn exit_code(&self) -> i32 {
        if matches!(self, Self::CoreCommitted(_)) {
            75
        } else {
            1
        }
    }
}

fn post_commit(record: Option<Operation>) -> Result<(), Failure> {
    let deadline = Instant::now() + Duration::from_secs(300);
    let (frozen, rollback) = if let Some(record) = record {
        (record.extensions, record.rollback)
    } else {
        let payload = soda_release_deliver::payload::load(crate::RELEASE_PATH)
            .map_err(|_| Failure::CoreCommitted("current release unavailable".into()))?;
        (
            crate::release_extensions::Frozen::capture(&payload, deadline)
                .map_err(Failure::CoreCommitted)?,
            false,
        )
    };
    crate::release_extensions::reconcile(&frozen, rollback, deadline)
        .map_err(Failure::CoreCommitted)
}

/// Ordinary boot record selection. An active operation keeps its frozen
/// record for verify/finalize under the guard. With no operation active,
/// reconciliation follows current intent: the retained historical record
/// stays intact for explicit rollback and evidence but is never parsed here,
/// so a later legitimate administrator change cannot poison boot.
fn select_boot_record(directory: &Path) -> Result<Option<Operation>, String> {
    match fs::symlink_metadata(directory.join(RECORD)) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(None),
        Err(_) => Err(CLOSED.into()),
        Ok(_) => Ok(Some(load_at(directory, RECORD)?)),
    }
}

pub(crate) fn reconcile_boot(args: &[String]) -> Result<(), Failure> {
    let finalized = (|| -> Result<Option<Operation>, String> {
        if !args.is_empty() {
            return Err("release-reconcile takes no arguments".into());
        }
        let _guard = admission::exclusive()?;
        let record = match select_boot_record(Path::new(admission::DIRECTORY))? {
            Some(record) => record,
            None => return Ok(None),
        };
        no_other_repairs()?;
        let deadline = Instant::now() + Duration::from_secs(120);
        // Boot consumes the same importer custody: an acquisition that
        // outlived its operation retires here or fences the finalize.
        crate::importer_custody::reconcile(Path::new(admission::DIRECTORY), 0, deadline)?;
        verify_active(&record, deadline)?;
        finish(record.clone(), deadline)?;
        Ok(Some(record))
    })()
    .map_err(Failure::BeforeCommit)?;
    // All exclusive guards have been dropped before normal manager reconciliation.
    post_commit(finalized)
}

pub(crate) fn run(args: &[String]) -> Result<(), Failure> {
    run_operation(args).map_err(|error| {
        Failure::BeforeCommit(if fs::symlink_metadata(path(RECORD)).is_ok() {
            format!("{error}; {CLOSED}")
        } else {
            error
        })
    })?;
    if !args.iter().any(|a| a == "--check") {
        match fs::symlink_metadata(path(RECORD)) {
            Ok(_) => {}
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                post_commit(Some(load(LAST).map_err(Failure::CoreCommitted)?))?
            }
            Err(_) => return Err(Failure::BeforeCommit(CLOSED.into())),
        }
    }
    Ok(())
}
fn resume_previous_operation(
    o: &Options,
    previous: &Option<Operation>,
    deadline: Instant,
) -> Result<Option<()>, String> {
    if let Some(record) = previous {
        if record.rollback {
            return rollback(deadline).map(|()| Some(()));
        }
        let state = release_stage::inspect(deadline)?;
        if state.booted.checksum == record.target_deployment {
            verify_active(record, deadline)?;
            return finish(record.clone(), deadline).map(|()| Some(()));
        }
    }
    if o.recover {
        // Recovery consumes the same importer custody the run recorded: an
        // outstanding acquisition retires here or fences recovery.
        crate::importer_custody::reconcile(Path::new(admission::DIRECTORY), 0, deadline)?;
    }
    Ok(None)
}

struct AdmittedOperation {
    target: crate::release_target::Target,
    state: release_stage::State,
    entries: Vec<Entry>,
}

fn check_resumed_record(
    r: &Operation,
    target: &crate::release_target::Target,
    state: &release_stage::State,
    entries: &[Entry],
) -> Result<(), String> {
    if r.source_release != target.source.release_id
        || r.source_sha256 != target.source.helper_sha256
        || r.source_deployment != state.booted.checksum
        || r.target_release != target.payload.id
        || r.target_deployment != target.media.deployment_commit
        || r.target_sha256 != target.payload.project_support.sha256
        || r.target_payload_sha256 != target.candidate.payload_sha256
        || r.extensions.target != target.payload
        || r.entries() != entries
    {
        return Err(CLOSED.into());
    }
    Ok(())
}

fn start_operation_record(
    target: &crate::release_target::Target,
    state: &release_stage::State,
    entries: &[Entry],
    fixture: bool,
    deadline: Instant,
) -> Result<Operation, String> {
    Ok(Operation {
        format: 2,
        extensions: crate::release_extensions::Frozen::capture(&target.payload, deadline)?,
        source_release: target.source.release_id.clone(),
        target_release: target.payload.id.clone(),
        source_deployment: state.booted.checksum.clone(),
        source_serial: state.booted.serial,
        target_deployment: target.media.deployment_commit.clone(),
        source_sha256: target.source.helper_sha256.clone(),
        target_sha256: target.payload.project_support.sha256.clone(),
        target_payload_sha256: target.candidate.payload_sha256.clone(),
        source_project_image: target.source.project_image.clone(),
        target_project_image: target.payload.project_support.project_image.clone(),
        release_directory: target
            .directory
            .to_str()
            .ok_or("invalid release path")?
            .into(),
        qualification_fixture: fixture,
        rollback: false,
        phase: "admitted".into(),
        projects: entries
            .iter()
            .map(|e| Project {
                project_id: e.project_id.clone(),
                container_id: e.container_id.clone(),
                old_sha256: target.source.helper_sha256.clone(),
                new_sha256: target.payload.project_support.sha256.clone(),
                previous: format!(
                    "/usr/libexec/soda/.project-terminal.previous-{}",
                    target.source.helper_sha256
                ),
            })
            .collect(),
    })
}

fn reuse_or_start_record(
    previous: Option<Operation>,
    target: &crate::release_target::Target,
    state: &release_stage::State,
    entries: &[Entry],
    fixture: bool,
    deadline: Instant,
) -> Result<Operation, String> {
    if let Some(r) = previous {
        check_resumed_record(&r, target, state, entries)?;
        return Ok(r);
    }
    start_operation_record(target, state, entries, fixture, deadline)
}

fn admit_operation_run(
    o: &Options,
    previous: Option<Operation>,
    deadline: Instant,
) -> Result<(AdmittedOperation, Operation), String> {
    let directory = o
        .directory
        .clone()
        .or_else(|| {
            previous
                .as_ref()
                .map(|r| PathBuf::from(&r.release_directory))
        })
        .ok_or("--release-dir PATH required")?;
    let fixture = previous
        .as_ref()
        .map_or(o.fixture, |r| r.qualification_fixture);
    if o.fixture && !fixture {
        return Err("recovery cannot change qualification mode".into());
    }
    let target = crate::release_target::admit(&directory, fixture, deadline)?;
    let state = release_stage::inspect(deadline)?;
    if state.booted.checksum != target.source.deployment
        || state.booted.checksum == target.media.deployment_commit
    {
        return Err("native source/target pair is not an upgrade".into());
    }
    let entries = release_inventory::inventory(deadline)?;
    let record = reuse_or_start_record(previous, &target, &state, &entries, fixture, deadline)?;
    Ok((
        AdmittedOperation {
            target,
            state,
            entries,
        },
        record,
    ))
}

fn preview_operation_content(
    o: &Options,
    admitted: &AdmittedOperation,
    record: &mut Operation,
    deadline: Instant,
) -> Result<release_stage::StagedContent, String> {
    let content = release_stage::StagedContent::open(&admitted.target, deadline)?;
    // Preview is read-only: image acquisition runs under importer custody
    // in the real run, never in preflight.
    if !o.recover && !o.check {
        record.extensions.prefetch(deadline)?;
    }
    record.extensions.retained(false, deadline)?;
    record.extensions.retained(true, deadline)?;
    crate::release_target::verify_project_image(&admitted.target, deadline)?;
    future_image(
        &record.source_project_image,
        &record.source_sha256,
        deadline,
    )?;
    let source = tool(
        &format!(
            "{}/usr/libexec/soda/project-terminal",
            release_stage::deployment_root(&admitted.state.booted)?.display()
        ),
        &record.source_sha256,
    )?;
    drop(source); // Establish retained rollback bytes before recording any mutation.
    observe(
        &admitted.entries,
        &record.source_sha256,
        if o.recover {
            &record.target_sha256
        } else {
            &record.source_sha256
        },
        true,
        deadline,
    )?;
    Ok(content)
}

fn execute_operation_run(
    o: &Options,
    admitted: &AdmittedOperation,
    record: &mut Operation,
    content: release_stage::StagedContent,
    deadline: Instant,
) -> Result<(), String> {
    admitted.target.confirm_source(deadline)?;
    if !o.recover {
        save(record, true)?;
    }
    content.stage(&record.target_deployment, deadline)?;
    phase(record, "host_locked")?;
    let helper = tool(&content.helper_path, &record.target_sha256)?;
    replace(record, &helper, deadline)?;
    phase(record, "helpers_verified")?;
    // The helper's open file pins the mounted release filesystem. Retire the
    // read handle before unmounting, including matching-support no-op repairs.
    drop(helper);
    content.close(deadline)?;
    release_inventory::revalidate(&admitted.entries, deadline)?;
    record.extensions.prepare(false, deadline)?;
    phase(record, "extensions_prepared")?;
    phase(record, "activation_requested")?;
    println!("Projects verified; requesting native activation of {}. Admission remains closed through boot verification.",record.target_release);
    release_stage::activate(&record.target_deployment, deadline)
}

fn run_operation(args: &[String]) -> Result<(), String> {
    let o = options(args)?;
    let _guard = admission::exclusive()?;
    no_other_repairs()?;
    let deadline = Instant::now() + Duration::from_secs(15 * 60);
    if o.rollback {
        return rollback(deadline);
    }
    let previous = if o.recover {
        Some(load(RECORD)?)
    } else {
        if fs::symlink_metadata(path(RECORD)).is_ok() {
            return Err(CLOSED.into());
        }
        None
    };
    if resume_previous_operation(&o, &previous, deadline)?.is_some() {
        return Ok(());
    }
    let (admitted, mut record) = admit_operation_run(&o, previous, deadline)?;
    println!(
        "Admitted {}: {}",
        admitted.target.payload.id,
        admitted.target.qualification_scope()
    );
    let content = preview_operation_content(&o, &admitted, &mut record, deadline)?;
    if o.check {
        println!("Release {} preflight: {} exact Projects idle; native activation, support replacement and image acquisition have not run{}",record.target_release,admitted.entries.len(),if record.qualification_fixture { "; qualification fixture only" } else { "" });
        return content.close(deadline);
    }
    execute_operation_run(&o, &admitted, &mut record, content, deadline)
}

fn admit_rollback_run(
    deadline: Instant,
) -> Result<(bool, Operation, release_stage::State), String> {
    let pending = fs::symlink_metadata(path(RECORD)).is_ok();
    let record = load(if pending { RECORD } else { LAST })?;
    // Rollback consumes the same importer custody: outstanding acquisition
    // retires here or fences the rollback.
    crate::importer_custody::reconcile(Path::new(admission::DIRECTORY), 0, deadline)?;
    if !pending {
        record
            .extensions
            .same_intent(&crate::release_extensions::snapshot(deadline)?)?;
    }
    record.extensions.retained(true, deadline)?;
    let state = release_stage::inspect(deadline)?;
    if state.booted.checksum != record.source_deployment
        && state.booted.checksum != record.target_deployment
    {
        return Err(CLOSED.into());
    }
    release_inventory::revalidate(&record.entries(), deadline)?;
    observe(
        &record.entries(),
        &record.source_sha256,
        &record.target_sha256,
        true,
        deadline,
    )?;
    Ok((pending, record, state))
}

fn restore_rollback_source(
    record: Operation,
    state: &release_stage::State,
    helper: &Tool,
    deadline: Instant,
) -> Result<(), String> {
    if state.staged.is_some() {
        release_stage::discard_locked(&record.target_deployment, deadline)?;
    }
    replace(&record, helper, deadline)?;
    record.extensions.prepare(true, deadline)?;
    crate::command::run(
        "/usr/bin/systemctl",
        &["start", "forgejo.service"],
        deadline,
    )?;
    verify_active(&record, deadline)?;
    finish(record, deadline)
}

fn restore_rollback_target(
    record: &mut Operation,
    helper: &Tool,
    deadline: Instant,
) -> Result<(), String> {
    release_stage::stage_retained(&record.source_deployment, deadline)?;
    phase(record, "host_locked")?;
    replace(record, helper, deadline)?;
    record.extensions.prepare(true, deadline)?;
    phase(record, "extensions_prepared")?;
    phase(record, "activation_requested")?;
    release_stage::activate(&record.source_deployment, deadline)
}

fn rollback(deadline: Instant) -> Result<(), String> {
    let (pending, mut record, state) = admit_rollback_run(deadline)?;
    // Native redeployment assigns a new serial and can retire the original
    // directory. After rollback boot, read support from that exact booted commit.
    let source_deployment = if state.booted.checksum == record.source_deployment {
        state.booted.clone()
    } else {
        Deployment {
            checksum: record.source_deployment.clone(),
            serial: record.source_serial,
            staged: false,
            finalization_locked: false,
        }
    };
    let helper = tool(
        &format!(
            "{}/usr/libexec/soda/project-terminal",
            release_stage::deployment_root(&source_deployment)?.display()
        ),
        &record.source_sha256,
    )?;
    record.rollback = true;
    record.phase = "admitted".into();
    save(&record, !pending)?;
    if state.booted.checksum == record.source_deployment {
        return restore_rollback_source(record, &state, &helper, deadline);
    }
    restore_rollback_target(&mut record, &helper, deadline)
}
