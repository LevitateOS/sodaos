//! `soda-build` command flow (Go `tools/soda-build` `main.go`): flag
//! parsing, dispatch admission, source binding, and progress. The isolated
//! worker execution, live-input resolution, and image pipeline stay behind
//! explicit release-pipeline boundary errors.

use std::io::Write;
use std::process::Command;

use crate::build_spec::Request;
use crate::digest::is_revision;
use crate::exitcode::{build_exit_code, note_interrupt, take_interrupt, ToolError};
use crate::goflag::{self, FlagKind, FlagSpec};
use crate::progress::BuildProgress;
use crate::worker;

pub const FLAG_SPECS: &[FlagSpec] = &[
    FlagSpec {
        name: "development",
        kind: FlagKind::Bool,
        usage: "explicit development-only run; never release-qualified",
        default_text: "",
    },
    FlagSpec {
        name: "target",
        kind: FlagKind::Text,
        usage: "development boundary: candidate or media (requires --development)",
        default_text: "",
    },
    FlagSpec {
        name: "media-compression",
        kind: FlagKind::Text,
        usage:
            "fast: development media only; changes host compression metadata (default: upstream)",
        default_text: "",
    },
    FlagSpec {
        name: "arch",
        kind: FlagKind::Text,
        usage: "matching native x86_64",
        default_text: "",
    },
    FlagSpec {
        name: "out",
        kind: FlagKind::Text,
        usage: "fresh absolute output below .artifacts/releases (parent must exist)",
        default_text: "",
    },
    FlagSpec {
        name: "repository-prefix",
        kind: FlagKind::Text,
        usage: "intended immutable image repositories; no publication",
        default_text: "ghcr.io/levitateos/sodaos",
    },
    FlagSpec {
        name: "rootfs-base-url",
        kind: FlagKind::Text,
        usage: "public base URL for the exact hash-named rootfs file",
        default_text: "",
    },
    FlagSpec {
        name: "media-authority",
        kind: FlagKind::Text,
        usage: "worker-local fixture authority; not release custody",
        default_text: "",
    },
    FlagSpec {
        name: "live-inputs",
        kind: FlagKind::Text,
        usage: "internal controller-resolved live inputs file",
        default_text: "",
    },
    FlagSpec {
        name: "forgejo-source",
        kind: FlagKind::Text,
        usage: "explicit clean canonical Forgejo fork checkout",
        default_text: "",
    },
    FlagSpec {
        name: "forgejo-revision",
        kind: FlagKind::Text,
        usage: "internal exact Forgejo source revision",
        default_text: "",
    },
    FlagSpec {
        name: "worker-config",
        kind: FlagKind::Text,
        usage: "root-owned configuration for isolated worker dispatch",
        default_text: "",
    },
    FlagSpec {
        name: "worker-build",
        kind: FlagKind::Bool,
        usage: "internal build stage; requires the isolated build identity",
        default_text: "",
    },
];

#[derive(Debug, Clone, Default)]
pub struct BuildFlags {
    pub request: Request,
    pub worker_build: bool,
    pub worker_config: String,
}

pub fn parse_build_flags(args: &[String]) -> Result<BuildFlags, String> {
    let outcome = goflag::parse(FLAG_SPECS, args)?;
    if !outcome.positionals.is_empty() {
        return Err("unexpected positional arguments".to_owned());
    }
    let request = Request {
        out: outcome.text("out"),
        arch: outcome.text("arch"),
        repository_prefix: outcome.text("repository-prefix"),
        rootfs_base_url: outcome.text("rootfs-base-url"),
        media_authority: outcome.text("media-authority"),
        development: outcome.boolean("development"),
        target: outcome.text("target"),
        media_compression: outcome.text("media-compression"),
        live_inputs: outcome.text("live-inputs"),
        forgejo_source: outcome.text("forgejo-source"),
        forgejo_revision: outcome.text("forgejo-revision"),
        ..Request::default()
    };
    request.validate_target()?;
    Ok(BuildFlags {
        request,
        worker_build: outcome.boolean("worker-build"),
        worker_config: outcome.text("worker-config"),
    })
}

pub fn admit_worker_build(f: &BuildFlags) -> Result<(), String> {
    if !f.worker_config.is_empty() {
        return Err("build stage cannot select worker configuration".to_owned());
    }
    if f.request.live_inputs.is_empty() {
        return Err(
            "isolated worker requires controller-resolved live inputs; it never fetches".to_owned(),
        );
    }
    worker::build_worker_identity()
}

pub fn admit_parent_dispatch(f: &BuildFlags) -> Result<(), String> {
    if f.worker_config.is_empty() || !f.request.media_authority.is_empty() {
        return Err(
            "root-owned --worker-config required; media authority belongs to the isolated worker"
                .to_owned(),
        );
    }
    if !f.request.live_inputs.is_empty() || !f.request.forgejo_revision.is_empty() {
        return Err(
            "controller resolves live inputs per attempt; operator selection refused".to_owned(),
        );
    }
    if !f.request.development {
        return Err("production builds removed; development only".to_owned());
    }
    Ok(())
}

pub fn admit_build_dispatch(f: &BuildFlags) -> Result<(), String> {
    if f.worker_build {
        admit_worker_build(f)
    } else {
        admit_parent_dispatch(f)
    }
}

pub fn usage(argv0: &str) -> String {
    goflag::print_defaults(argv0, FLAG_SPECS)
}

pub fn sanitize_build_env(worker_build: bool) {
    unsafe {
        std::env::remove_var("SODA_BUILD_TIMING_LOG");
        std::env::remove_var("SODA_BUILD_CHILD");
        if !worker_build {
            std::env::remove_var("SODA_BUILD_START_NS");
        }
    }
}

extern "C" fn on_signal(signum: libc::c_int) {
    note_interrupt(128 + signum);
}

pub fn watch_build_signals() {
    unsafe {
        libc::signal(libc::SIGINT, on_signal as *const () as libc::sighandler_t);
        libc::signal(libc::SIGTERM, on_signal as *const () as libc::sighandler_t);
    }
}

pub fn progress_title(r: &Request) -> String {
    if r.development {
        format!("Soda development {} (not release-qualified)", r.target)
    } else {
        "Soda release build".to_owned()
    }
}

pub fn controller_revision() -> Result<String, String> {
    if env!("SODA_BUILD_VCS_MODIFIED") == "1" {
        return Err("controller must be compiled from committed source".to_owned());
    }
    let revision = env!("SODA_BUILD_VCS_REVISION").to_owned();
    if !is_revision(&revision) {
        return Err(
            "controller needs build VCS metadata; compile tools/soda-build from the committed checkout"
                .to_owned(),
        );
    }
    Ok(revision)
}

pub fn forgejo_git_output(source: &str, args: &[&str]) -> Result<Vec<u8>, String> {
    let mut command = Command::new("git");
    command
        .arg("-c")
        .arg(format!("safe.directory={source}"))
        .arg("-C")
        .arg(source)
        .args(args);
    match command.output() {
        Ok(output) if output.status.success() => Ok(output.stdout),
        Ok(output) => Err(format!(
            "git {} failed with status {}",
            args.join(" "),
            output.status.code().unwrap_or(-1)
        )),
        Err(e) => Err(e.to_string()),
    }
}

pub fn validate_forgejo_checkout_root(source: &str) -> Result<(), String> {
    if !source.starts_with('/') || crate::worker::go_clean(source) != source {
        return Err("explicit absolute Forgejo checkout required".to_owned());
    }
    match forgejo_git_output(source, &["rev-parse", "--show-toplevel"]) {
        Ok(root) if String::from_utf8_lossy(&root).trim() == source => {}
        _ => return Err("canonical Forgejo checkout required".to_owned()),
    }
    let dotgit = format!("{source}/.git");
    match std::fs::symlink_metadata(&dotgit) {
        Ok(st) if st.file_type().is_dir() => Ok(()),
        _ => Err("canonical Forgejo checkout required; no worktree".to_owned()),
    }
}

pub fn forgejo_checkout_revision(source: &str) -> Result<String, String> {
    match forgejo_git_output(
        source,
        &["status", "--porcelain", "--untracked-files=normal"],
    ) {
        Ok(status) if status.is_empty() => {}
        _ => return Err("clean committed Forgejo source required".to_owned()),
    }
    match forgejo_git_output(source, &["rev-parse", "HEAD"]) {
        Ok(head) => {
            let revision = String::from_utf8_lossy(&head).trim().to_owned();
            if is_revision(&revision) {
                Ok(revision)
            } else {
                Err("exact Forgejo source revision required".to_owned())
            }
        }
        Err(_) => Err("exact Forgejo source revision required".to_owned()),
    }
}

pub fn bind_forgejo_source(r: &mut Request) -> Result<(), String> {
    validate_forgejo_checkout_root(&r.forgejo_source)?;
    let revision = forgejo_checkout_revision(&r.forgejo_source)?;
    if !r.forgejo_revision.is_empty() && r.forgejo_revision != revision {
        return Err("forgejo source changed after dispatch".to_owned());
    }
    r.forgejo_revision = revision;
    Ok(())
}

pub fn bind_build_source(r: &mut Request) -> Result<(), String> {
    if unsafe { libc::prctl(libc::PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0) } != 0 {
        return Err(std::io::Error::last_os_error().to_string());
    }
    let source = std::env::current_dir().map_err(|e| e.to_string())?;
    let revision = controller_revision()?;
    r.source = source.to_string_lossy().into_owned();
    r.revision = revision;
    bind_forgejo_source(r)
}

pub fn print_build_artifacts(candidate: &str, media: &str, scope: &str) {
    let mut stderr = std::io::stderr().lock();
    let _ = writeln!(stderr, "CANDIDATE {candidate}");
    if !media.is_empty() {
        let _ = writeln!(stderr, "MEDIA {media}");
    }
    let _ = writeln!(stderr, "{scope}");
}

/// Parent dispatch: admit the config, run the isolated worker, and
/// report the produced artifacts.
pub fn run_parent_build(
    worker_config_path: &str,
    r: &Request,
    progress: &mut BuildProgress,
) -> Result<(), worker::WorkerError> {
    let config =
        worker::load_worker_config(worker_config_path, r).map_err(worker::WorkerError::Failed)?;
    let result = worker::run_build_worker(&config, r, progress)?;
    print_build_artifacts(&result.candidate, &result.media, &result.scope);
    Ok(())
}

/// Map the isolated-worker result to the CLI error: failures keep their
/// message and exit 1; cancellation keeps its stop/reap evidence on stderr
/// and takes the recorded signal identity (128+sig) or CANCELLED (130).
pub(crate) fn map_worker_error(e: worker::WorkerError) -> ToolError {
    match e {
        worker::WorkerError::Failed(message) => ToolError::msg(message),
        worker::WorkerError::Cancelled(evidence) => {
            eprintln!("{evidence}");
            take_interrupt()
                .map(ToolError::Interrupted)
                .unwrap_or(ToolError::Cancelled)
        }
    }
}

pub fn run() -> Result<(), ToolError> {
    let argv: Vec<String> = std::env::args().collect();
    let argv0 = argv
        .first()
        .cloned()
        .unwrap_or_else(|| "soda-build".to_owned());
    let args = if argv.len() > 1 { &argv[1..] } else { &[] };
    match goflag::parse(FLAG_SPECS, args) {
        Err(e) if e == goflag::ERR_HELP => {
            eprint!("{}", usage(&argv0));
            std::process::exit(0);
        }
        Err(e) => {
            eprint!("{e}\n{}", usage(&argv0));
            std::process::exit(2);
        }
        Ok(_) => {}
    }
    let mut f = parse_build_flags(args).map_err(ToolError::msg)?;
    if let Err(e) = admit_build_dispatch(&f) {
        return Err(ToolError::msg(e));
    }
    watch_build_signals();
    sanitize_build_env(f.worker_build);
    let mut progress = BuildProgress::new(&progress_title(&f.request)).map_err(ToolError::msg)?;
    let mut err: Option<ToolError> = None;
    if let Err(e) = bind_build_source(&mut f.request) {
        err = Some(ToolError::msg(e));
    } else if f.worker_build {
        // Worker-stage image build: stage completion only.
        if let Err(e) = crate::pipeline::run_worker_stage(&f.request, &mut progress) {
            err = Some(ToolError::msg(e));
        }
    } else if let Err(e) = run_parent_build(&f.worker_config, &f.request, &mut progress) {
        err = Some(map_worker_error(e));
    }
    if err.is_none() {
        err = take_interrupt().map(ToolError::Interrupted);
    }
    if let Err(e) = progress.finish(err.as_ref()) {
        if err.is_none() {
            err = Some(ToolError::msg(e));
        }
    }
    match err {
        Some(e) => Err(e),
        None => Ok(()),
    }
}

pub fn main() {
    if let Err(err) = run() {
        eprintln!("{err}");
        std::process::exit(build_exit_code(Some(&err)));
    }
}

#[cfg(test)]
mod tests;
