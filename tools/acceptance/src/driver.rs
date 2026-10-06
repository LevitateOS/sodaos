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
mod tests {
    use super::*;
    use crate::files::TempDir;

    #[test]
    fn durations_match_go_terms() {
        assert_eq!(parse_duration("0").unwrap(), 0);
        assert_eq!(parse_duration("30m").unwrap(), 1_800_000_000_000);
        assert_eq!(parse_duration("1h30m").unwrap(), 5_400_000_000_000);
        assert_eq!(parse_duration("1.5h").unwrap(), 5_400_000_000_000);
        assert_eq!(parse_duration("300ms").unwrap(), 300_000_000);
        assert_eq!(parse_duration("-5s").unwrap(), -5_000_000_000);
        assert_eq!(parse_duration("2µs").unwrap(), 2_000);
        assert_eq!(parse_duration("2μs").unwrap(), 2_000);
        assert!(parse_duration("").is_err());
        assert!(parse_duration("1").is_err());
        assert!(parse_duration("1x").is_err());
        assert!(parse_duration("99999999999999999999h").is_err());
    }

    #[test]
    fn flags_mirror_go_forms() {
        let args = [
            "--owner",
            "P02",
            "-revision=a",
            "--hold",
            "-secret-file",
            "one",
            "--secret-file=two",
            "--",
            "pos",
        ]
        .iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>();
        let parsed = parse_flags(
            &args,
            &[
                ("owner", FlagKind::Str),
                ("revision", FlagKind::Str),
                ("hold", FlagKind::Bool),
                ("secret-file", FlagKind::Repeat),
            ],
        )
        .unwrap();
        assert_eq!(flag_string(&parsed, "owner"), "P02");
        assert_eq!(flag_string(&parsed, "revision"), "a");
        assert_eq!(parsed.bools.get("hold"), Some(&true));
        assert_eq!(
            parsed.repeats.get("secret-file").unwrap(),
            &vec!["one".to_string(), "two".to_string()]
        );
        assert_eq!(parsed.positionals, vec!["pos".to_string()]);
        // Bools take no separate value; unknown flags and missing values fail.
        let args = ["-hold", "false"]
            .iter()
            .map(|s| s.to_string())
            .collect::<Vec<_>>();
        let parsed = parse_flags(&args, &[("hold", FlagKind::Bool)]).unwrap();
        assert_eq!(parsed.positionals, vec!["false".to_string()]);
        let args = ["-bogus"].iter().map(|s| s.to_string()).collect::<Vec<_>>();
        assert!(parse_flags(&args, &[("hold", FlagKind::Bool)]).is_err());
        let args = ["-owner"].iter().map(|s| s.to_string()).collect::<Vec<_>>();
        assert!(parse_flags(&args, &[("owner", FlagKind::Str)]).is_err());
        let args = ["-hold=maybe"]
            .iter()
            .map(|s| s.to_string())
            .collect::<Vec<_>>();
        assert!(parse_flags(&args, &[("hold", FlagKind::Bool)]).is_err());
    }

    #[test]
    fn option_validation_matches_go_messages() {
        let base = [
            "exec",
            "--owner",
            "P02",
            "--revision",
            &"a".repeat(40),
            "--arch",
            "x86_64",
            "--target",
            "fixture",
            "--evidence",
            "/tmp/x",
        ]
        .iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>();
        let mut with_cmd = base.clone();
        with_cmd.extend(["--".to_string(), "true".to_string()]);
        assert!(parse_run_options(&with_cmd).is_ok());
        assert_eq!(
            parse_run_options(&base).err().unwrap().to_string(),
            "an existing owned check command is required"
        );
        let mut bad_owner = with_cmd.clone();
        bad_owner[2] = "P07".to_string();
        assert_eq!(
            parse_run_options(&bad_owner).err().unwrap().to_string(),
            "explicit owner required; P07/P08 are not independent tasks"
        );
        let mut bad_timeout = base.clone();
        bad_timeout.extend([
            "--timeout".to_string(),
            "25h".to_string(),
            "--".to_string(),
            "true".to_string(),
        ]);
        assert_eq!(
            parse_run_options(&bad_timeout).err().unwrap().to_string(),
            "revision, non-secret target and bounded timeout required"
        );
        let publish = ["publish"]
            .iter()
            .map(|s| s.to_string())
            .collect::<Vec<_>>();
        assert_eq!(
            parse_run_options(&publish).err().unwrap().to_string(),
            "no product/media/release workflow is implemented by this support tool"
        );
    }

    fn run_args(action: &str, evidence: &str) -> Vec<String> {
        [
            action,
            "--owner",
            "P07",
            "--revision",
            &"a".repeat(40),
            "--arch",
            "x86_64",
            "--target",
            "fixture",
            "--evidence",
            evidence,
        ]
        .iter()
        .map(|s| s.to_string())
        .collect()
    }

    /// Port of `TestInvalidActionsDoNotCreateEvidence` from `main_test.go`.
    #[test]
    fn invalid_actions_do_not_create_evidence() {
        for action in ["publish", "exec", "native", "vm", "probe-ssh"] {
            let scratch = TempDir::new("driver").unwrap();
            let path = scratch.join("evidence").to_string_lossy().into_owned();
            let args = run_args(action, &path);
            assert!(run(&Phase::background(), &args).is_err(), "{action}");
            assert!(std::fs::symlink_metadata(&path).is_err(), "{action}");
        }
    }

    /// Port of `TestCancelledExecutionRecordsFailure` from `main_test.go`.
    #[test]
    fn cancelled_execution_records_failure() {
        let scratch = TempDir::new("driver").unwrap();
        let path = scratch.join("evidence").to_string_lossy().into_owned();
        let root = Phase::background();
        root.cancel();
        let args = [
            "exec",
            "--owner",
            "P02",
            "--revision",
            &"a".repeat(40),
            "--arch",
            "x86_64",
            "--target",
            "fixture",
            "--evidence",
            &path,
            "--",
            "must-not-be-started",
        ]
        .iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>();
        assert!(run(&root, &args).is_err());
        let raw = std::fs::read(format!("{path}/observation.json")).unwrap();
        let text = String::from_utf8(raw).unwrap();
        assert!(text.contains("\"Outcome\": \"cancelled\""), "{text}");
        assert!(text.contains("\"Execution\": \"not-started\""), "{text}");
    }
}
