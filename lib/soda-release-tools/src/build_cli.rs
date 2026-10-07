//! `soda-build` command flow (Go `tools/soda-build` `main.go`): flag
//! parsing, dispatch admission, source binding, and progress. The isolated
//! worker execution, live-input resolution, and image pipeline stay behind
//! explicit release-pipeline boundary errors.

use std::io::Write;
use std::process::Command;

use crate::build_spec::Request;
use crate::digest::is_revision;
use crate::exitcode::{build_exit_code, note_interrupt, take_interrupt, ToolError};
use crate::progress::BuildProgress;
use crate::worker;
use clap::{Arg, ArgAction, Command as ClapCommand};

fn value(name: &'static str, help: &'static str) -> Arg {
    Arg::new(name)
        .long(name)
        .help(help)
        .action(ArgAction::Set)
        .num_args(1)
        .allow_hyphen_values(true)
}

fn boolean(name: &'static str, help: &'static str) -> Arg {
    Arg::new(name)
        .long(name)
        .help(help)
        .action(ArgAction::Set)
        .num_args(0..=1)
        .require_equals(true)
        .default_missing_value("true")
        .value_parser([
            "true", "false", "1", "0", "t", "T", "TRUE", "True", "f", "F", "FALSE", "False",
        ])
}

fn command(name: &str) -> ClapCommand {
    ClapCommand::new("soda-build")
        .bin_name(name)
        .args_override_self(true)
        .disable_help_subcommand(true)
        .arg(boolean(
            "development",
            "explicit development-only run; never release-qualified",
        ))
        .arg(value(
            "target",
            "development boundary: candidate or media (requires --development)",
        ))
        .arg(value(
            "media-compression",
            "fast: development media only; changes host compression metadata (default: upstream)",
        ))
        .arg(value("arch", "matching native x86_64"))
        .arg(value(
            "out",
            "fresh absolute output below .artifacts/releases (parent must exist)",
        ))
        .arg(
            value(
                "repository-prefix",
                "intended immutable image repositories; no publication",
            )
            .default_value("ghcr.io/levitateos/sodaos"),
        )
        .arg(value(
            "rootfs-base-url",
            "public base URL for the exact hash-named rootfs file",
        ))
        .arg(value(
            "media-authority",
            "worker-local fixture authority; not release custody",
        ))
        .arg(value(
            "live-inputs",
            "internal controller-resolved live inputs file",
        ))
        .arg(value(
            "forgejo-source",
            "explicit clean canonical Forgejo fork checkout",
        ))
        .arg(value(
            "forgejo-revision",
            "internal exact Forgejo source revision",
        ))
        .arg(value(
            "worker-config",
            "root-owned configuration for isolated worker dispatch",
        ))
        .arg(boolean(
            "worker-build",
            "internal build stage; requires the isolated build identity",
        ))
}

#[derive(Debug, Clone, Default)]
pub struct BuildFlags {
    pub request: Request,
    pub worker_build: bool,
    pub worker_config: String,
}

pub fn parse_build_flags(args: &[String]) -> Result<BuildFlags, String> {
    let matches = command("soda-build")
        .try_get_matches_from(std::iter::once("soda-build").chain(args.iter().map(String::as_str)))
        .map_err(|error| {
            if error.kind() == clap::error::ErrorKind::DisplayHelp {
                "build help requested".to_owned()
            } else {
                "invalid build command flags".to_owned()
            }
        })?;
    let request = Request {
        out: matches
            .get_one::<String>("out")
            .cloned()
            .unwrap_or_default(),
        arch: matches
            .get_one::<String>("arch")
            .cloned()
            .unwrap_or_default(),
        repository_prefix: matches
            .get_one::<String>("repository-prefix")
            .cloned()
            .unwrap_or_else(|| "ghcr.io/levitateos/sodaos".to_owned()),
        rootfs_base_url: matches
            .get_one::<String>("rootfs-base-url")
            .cloned()
            .unwrap_or_default(),
        media_authority: matches
            .get_one::<String>("media-authority")
            .cloned()
            .unwrap_or_default(),
        development: matches.get_one::<String>("development").is_some_and(|v| {
            v == "true" || v == "1" || v == "t" || v == "T" || v == "TRUE" || v == "True"
        }),
        target: matches
            .get_one::<String>("target")
            .cloned()
            .unwrap_or_default(),
        media_compression: matches
            .get_one::<String>("media-compression")
            .cloned()
            .unwrap_or_default(),
        live_inputs: matches
            .get_one::<String>("live-inputs")
            .cloned()
            .unwrap_or_default(),
        forgejo_source: matches
            .get_one::<String>("forgejo-source")
            .cloned()
            .unwrap_or_default(),
        forgejo_revision: matches
            .get_one::<String>("forgejo-revision")
            .cloned()
            .unwrap_or_default(),
        ..Request::default()
    };
    request.validate_target()?;
    Ok(BuildFlags {
        request,
        worker_build: matches.get_one::<String>("worker-build").is_some_and(|v| {
            v == "true" || v == "1" || v == "t" || v == "T" || v == "TRUE" || v == "True"
        }),
        worker_config: matches
            .get_one::<String>("worker-config")
            .cloned()
            .unwrap_or_default(),
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
    command(argv0).render_help().to_string()
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
    let mut f = match parse_build_flags(args) {
        Ok(flags) => flags,
        Err(e) if e == "build help requested" => {
            print!("{}", usage(&argv0));
            std::process::exit(0);
        }
        Err(e) if e == "invalid build command flags" => {
            eprint!("{e}\n{}", usage(&argv0));
            std::process::exit(2);
        }
        Err(e) => return Err(ToolError::msg(e)),
    };
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
