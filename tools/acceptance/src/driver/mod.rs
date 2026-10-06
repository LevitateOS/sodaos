//! Outside support driver, mirroring `tools/soda-acceptance/main.go`.
//!
//! Invokes existing owned checks; no product scenarios. Actions:
//! `exec`, `native`, `vm`, `probe-ssh`, and `report`. Flag parsing
//! mirrors Go's `flag` package (single/double dashes, `=` or separate
//! values, bools without consumption), and durations parse like Go's
//! `time.ParseDuration`. Cancellation arrives through [`Phase`]:
//! the binary forwards SIGINT/SIGTERM into it, like Go's
//! `signal.NotifyContext`.

use std::sync::OnceLock;
use std::time::Duration;

use crate::command::{self, Remote};
use crate::error::Error;
use crate::evidence::create_evidence;
use crate::files;
use crate::jsonio;
use crate::process::Phase;
use crate::report::{self, Observation};

mod actions;
mod finalization;
mod inputs;
mod options;

use actions::execute_action;
use finalization::finalize_observation;
use inputs::collect_all_secrets;
#[cfg(test)]
use options::parse_duration;
use options::{flag_string, parse_flags, parse_run_options, FlagKind, RunOptions};

/// SIGINT/SIGTERM cancel this phase, like `signal.NotifyContext`.
static SIGNAL_PHASE: OnceLock<Phase> = OnceLock::new();

extern "C" fn forward_signal(_signal: libc::c_int) {
    if let Some(phase) = SIGNAL_PHASE.get() {
        phase.cancel();
    }
}

/// Forward SIGINT/SIGTERM cancellations into `phase`. The binary calls
/// this once; tests drive phases directly and never install it.
pub fn install_signal_forwarding(phase: &Phase) {
    let _ = SIGNAL_PHASE.set(phase.clone());
    let handler = forward_signal as extern "C" fn(libc::c_int);
    unsafe {
        libc::signal(libc::SIGINT, handler as libc::sighandler_t);
        libc::signal(libc::SIGTERM, handler as libc::sighandler_t);
    }
}

/// Driver platform in Go `GOOS/GOARCH` vocabulary.
fn client_platform() -> String {
    let os = match std::env::consts::OS {
        "macos" => "darwin",
        other => other,
    };
    let arch = match std::env::consts::ARCH {
        "x86_64" => "amd64",
        "x86" => "386",
        "aarch64" => "arm64",
        other => other,
    };
    format!("{os}/{arch}")
}

fn init_observation(opts: &RunOptions) -> Result<(Observation, Remote), Error> {
    let mut observation = Observation {
        owner: opts.owner.clone(),
        requested_revision: opts.revision.clone(),
        requested_architecture: opts.arch.clone(),
        target: opts.target.clone(),
        client_platform: client_platform(),
        action: opts.action.clone(),
        started: jsonio::now_rfc3339_nano(),
        execution: "not-started".to_string(),
        cleanup: "not-applicable".to_string(),
        topology: "local command".to_string(),
        artifacts: Some(Default::default()),
        ..Observation::default()
    };
    if let Some(artifacts) = observation.artifacts.as_mut() {
        for file in &opts.artifact_files {
            artifacts.insert(file.clone(), files::hash_file(file)?);
        }
    }
    let mut remote = Remote {
        user: String::new(),
        host: String::new(),
        key: String::new(),
        known_hosts: String::new(),
        port: 0,
        timeout: Duration::ZERO,
    };
    if !opts.remote_file.is_empty() {
        let (value, _) = jsonio::read_json_file(&opts.remote_file)?;
        remote = command::decode_remote(&value)?;
        remote.timeout = opts.timeout;
        observation.topology = format!(
            "pinned SSH {}@{}:{}; management transport, not a project client route",
            remote.user, remote.host, remote.port
        );
        observation.cleanup = "remote phase bounded by timeout + 10s; interrupted transport is not remote cleanup proof".to_string();
    }
    Ok((observation, remote))
}

/// Run one driver action. Mirrors `run`.
pub fn run(root: &Phase, args: &[String]) -> Result<(), Error> {
    if args.is_empty() {
        return Err(Error::msg(
            "usage: soda-acceptance exec|native|vm|probe-ssh|report [flags]",
        ));
    }
    if args[0] == "report" {
        return report(&args[1..]);
    }
    let opts = parse_run_options(args)?;
    let (secrets, vm_config) = collect_all_secrets(&opts)?;
    let evidence = create_evidence(&opts.evidence, &secrets)?;
    let phase = root.child(opts.timeout);
    let (mut observation, remote) = init_observation(&opts)?;
    let (operation_err, evidence_err) = execute_action(
        &phase,
        &evidence,
        &opts,
        vm_config,
        &remote,
        &mut observation,
    );
    finalize_observation(&evidence, observation, operation_err, evidence_err)
}

fn report(args: &[String]) -> Result<(), Error> {
    let parsed = parse_flags(
        args,
        &[
            ("arch", FlagKind::Str),
            ("revision", FlagKind::Str),
            ("out", FlagKind::Str),
            ("record", FlagKind::Repeat),
        ],
    )
    .map_err(|_| Error::msg("invalid report flags"))?;
    if !parsed.positionals.is_empty() {
        return Err(Error::msg("invalid report flags"));
    }
    report::handoff(
        &flag_string(&parsed, "out"),
        &flag_string(&parsed, "arch"),
        &flag_string(&parsed, "revision"),
        parsed
            .repeats
            .get("record")
            .cloned()
            .unwrap_or_default()
            .as_slice(),
    )
}

#[cfg(test)]
mod tests;
