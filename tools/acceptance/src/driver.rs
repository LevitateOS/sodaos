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

use crate::command::{self, CommandSpec, Remote, StdinSpec};
use crate::error::Error;
use crate::evidence::{create_evidence, Evidence};
use crate::files;
use crate::jsonio;
use crate::process::Phase;
use crate::provisioning;
use crate::remote;
use crate::report::{self, Observation};
use crate::vm;

mod options;

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

fn read_secret_files(files: &[String]) -> Result<Vec<Vec<u8>>, Error> {
    let mut secrets = Vec::new();
    for file in files {
        let bytes = files::private_file(file)?;
        if bytes.is_empty() {
            return Err(Error::msg("empty secret input"));
        }
        let mut trimmed = bytes.clone();
        while trimmed.last().is_some_and(|b| *b == b'\r' || *b == b'\n') {
            trimmed.pop();
        }
        secrets.push(bytes);
        secrets.push(trimmed);
    }
    Ok(secrets)
}

fn load_vm_secrets(
    config_path: &str,
    arch: &str,
    target: &str,
) -> Result<(vm::VmConfig, Vec<Vec<u8>>), Error> {
    let (value, _) = jsonio::read_json_file(config_path)?;
    let vm_config = vm::decode_vm_config(&value)?;
    if vm_config.architecture != arch || vm_config.name != target {
        return Err(Error::msg("VM target/platform mismatch"));
    }
    let private = provisioning::provisioning_secrets(&vm_config.ignition)?;
    Ok((vm_config, private))
}

fn collect_all_secrets(opts: &RunOptions) -> Result<(Vec<Vec<u8>>, vm::VmConfig), Error> {
    let mut secrets = read_secret_files(&opts.secret_files)?;
    let mut vm_config = vm::VmConfig::default();
    if opts.action == "vm" {
        let (config, vm_secrets) = load_vm_secrets(&opts.config, &opts.arch, &opts.target)?;
        vm_config = config;
        secrets.extend(vm_secrets);
    }
    Ok((secrets, vm_config))
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

/// Prebuilt target payload: the `soda-acceptance-remote` binary shipped
/// beside the driver. The Go owner embeds Python source instead, so a
/// missing sibling is a new deployment-shape failure with no Go text.
fn payload_bytes() -> Result<Vec<u8>, Error> {
    let driver = std::env::current_exe().map_err(Error::from)?;
    let payload = driver
        .parent()
        .unwrap_or(std::path::Path::new("."))
        .join("soda-acceptance-remote");
    std::fs::read(&payload).map_err(Error::from)
}

#[allow(clippy::too_many_arguments)]
fn execute_exec_or_native(
    phase: &Phase,
    evidence: &Evidence,
    action: &str,
    args: &[String],
    remote_file: &str,
    request: &str,
    revision: &str,
    arch: &str,
    target: &str,
    remote: &Remote,
    observation: &mut Observation,
) -> (Option<Error>, Option<Error>) {
    let spec = if action == "exec" {
        observation.invocation = Some(args.to_vec());
        let mut spec = CommandSpec {
            name: args[0].clone(),
            args: args[1..].to_vec(),
            dir: None,
            stdin: StdinSpec::Inherit,
            env: Vec::new(),
        };
        if !remote_file.is_empty() {
            spec = match remote.command(args, StdinSpec::Inherit) {
                Ok(spec) => spec,
                Err(err) => return (Some(err), None),
            };
        }
        spec
    } else {
        let payload = match payload_bytes() {
            Ok(payload) => payload,
            Err(err) => return (Some(err), None),
        };
        let spec = match remote::native_phase(remote, request, revision, arch, target, &payload) {
            Ok(spec) => spec,
            Err(err) => return (Some(err), None),
        };
        let raw = match jsonio::read_json_file(request) {
            Ok((value, _)) => value,
            Err(err) => return (Some(err), None),
        };
        let request = match remote::decode_remote_request(&raw) {
            Ok(request) => request,
            Err(err) => return (Some(err), None),
        };
        observation.invocation = Some(vec![
            "embedded native executor".to_string(),
            request.phase.clone(),
            request.revision,
            request.architecture,
            request.target,
            request.work,
        ]);
        observation.action = format!("native-{}", request.phase);
        spec
    };
    let (result, evidence_err) = command::execute(phase, evidence, "check", &spec);
    observation.exit_code = result.exit_code.map(i64::from);
    if result.started {
        observation.execution = "completed".to_string();
        if result.err.is_some() {
            observation.execution = "failed".to_string();
        }
    }
    (result.err, evidence_err)
}

fn execute_probe_ssh(
    phase: &Phase,
    evidence: &Evidence,
    remote: &Remote,
    observation: &mut Observation,
) -> (Option<Error>, Option<Error>) {
    observation.topology = format!(
        "direct client TCP to {}:{}; no proxy, forwarding inference or Git authentication",
        remote.host, remote.port
    );
    observation.invocation = Some(vec![
        "SSH key exchange only".to_string(),
        remote.host.clone(),
        remote.port.to_string(),
        remote.known_hosts.clone(),
    ]);
    observation.execution = "failed".to_string();
    observation.cleanup = "connection closed".to_string();
    let fingerprint = match crate::probe::probe_ssh_key(phase, remote) {
        Ok(fingerprint) => fingerprint,
        Err(err) => return (Some(err), None),
    };
    observation.execution = "completed".to_string();
    let evidence_err = evidence
        .write("ssh-key.txt", format!("{fingerprint}\nPinned SSH endpoint observed, not Git authentication or project routing proof.\n").as_bytes())
        .err();
    (None, evidence_err)
}

fn run_vm_phase(
    phase: &Phase,
    evidence: &Evidence,
    vm: &mut vm::Vm,
    work: &str,
    restart: bool,
    hold: bool,
    observation: &mut Observation,
) -> (Option<Error>, Option<Error>) {
    observation.execution = "completed".to_string();
    let mut evidence_err = evidence
        .write("boot-ready.txt", b"Fresh guest reached pinned SSH readiness. This is fixture evidence, not a product check.\n")
        .err();
    let mut operation_err = None;
    if restart {
        operation_err = vm.restart(phase).err();
        if operation_err.is_none() {
            evidence_err = Error::join(vec![
                evidence_err,
                evidence
                    .write(
                        "restart-ready.txt",
                        b"Same owned disk/NVRAM reached pinned SSH after restart. No project persistence assertion was performed.\n",
                    )
                    .err(),
            ]);
        }
    }
    if operation_err.is_none() && hold {
        println!("Fresh guest ready. Invoke owned checks separately. Interrupt shuts down and retains disks; cancellation is not a pass.");
        operation_err = vm.wait(phase).err();
    }
    if operation_err.is_some() {
        observation.execution = "failed".to_string();
    }
    let cleanup_err = vm.close().err();
    observation.cleanup = format!("completed; private disk/NVRAM retained at {work}");
    if cleanup_err.is_some() {
        observation.cleanup = "failed; private work retained".to_string();
    }
    (Error::join(vec![operation_err, cleanup_err]), evidence_err)
}

fn execute_vm(
    phase: &Phase,
    evidence: &Evidence,
    vm_config: vm::VmConfig,
    restart: bool,
    hold: bool,
    observation: &mut Observation,
) -> (Option<Error>, Option<Error>) {
    observation.topology =
        "matching-native KVM; loopback management SSH only, no routed-client claim".to_string();
    let mut invocation = vec![
        "fresh VM".to_string(),
        vm_config.name.clone(),
        vm_config.work.clone(),
        vm_config.base_receipt.clone(),
        "private single fw_cfg Ignition (not retained)".to_string(),
    ];
    if restart {
        invocation.push("--restart".to_string());
    }
    if hold {
        invocation.push("--hold".to_string());
    }
    observation.invocation = Some(invocation);
    observation.cleanup = "not-reached or failed launch; work retained".to_string();
    let work = vm_config.work.clone();
    let mut guest = match vm::launch_vm(phase, vm_config, evidence) {
        Ok(guest) => guest,
        Err(failure) => {
            let failure = *failure;
            if let Some(mut started) = failure.vm {
                observation.execution = "failed".to_string();
                observation.cleanup =
                    "completed after failed launch; private work retained".to_string();
                if started.close().is_err() {
                    observation.cleanup =
                        "failed after failed launch; private work retained".to_string();
                }
            }
            return (Some(failure.err), None);
        }
    };
    run_vm_phase(
        phase,
        evidence,
        &mut guest,
        &work,
        restart,
        hold,
        observation,
    )
}

fn execute_action(
    phase: &Phase,
    evidence: &Evidence,
    opts: &RunOptions,
    vm_config: vm::VmConfig,
    remote: &Remote,
    observation: &mut Observation,
) -> (Option<Error>, Option<Error>) {
    match opts.action.as_str() {
        "exec" | "native" => execute_exec_or_native(
            phase,
            evidence,
            &opts.action,
            &opts.cmd_args,
            &opts.remote_file,
            &opts.request,
            &opts.revision,
            &opts.arch,
            &opts.target,
            remote,
            observation,
        ),
        "probe-ssh" => execute_probe_ssh(phase, evidence, remote, observation),
        "vm" => execute_vm(
            phase,
            evidence,
            vm_config,
            opts.restart,
            opts.hold,
            observation,
        ),
        _ => (None, None),
    }
}

fn read_build_settings(observation: &mut Observation) {
    observation.tool_revision = option_env!("BUILD_REVISION")
        .unwrap_or("unknown")
        .to_string();
    observation.tool_dirty = option_env!("BUILD_MODIFIED").unwrap_or("false") == "true";
    if observation.tool_revision.is_empty() {
        observation.tool_revision = "unknown".to_string();
    }
}

fn redact_observation_strings(evidence: &Evidence, observation: &mut Observation) {
    if let Some(invocation) = observation.invocation.as_mut() {
        for arg in invocation {
            *arg = evidence.redact_string(arg);
        }
    }
    observation.topology = evidence.redact_string(&observation.topology);
    observation.cleanup = evidence.redact_string(&observation.cleanup);
    if let Some(artifacts) = observation.artifacts.as_mut() {
        let redacted: Vec<(String, String)> = artifacts
            .iter()
            .map(|(name, sum)| (evidence.redact_string(name), sum.clone()))
            .collect();
        *artifacts = redacted.into_iter().collect();
    }
}

/// Join error Displays like [`Error::join`]: single messages pass
/// through, several join with newlines. Used where the Go owner
/// formats a join it also keeps split; owned errors move only once.
fn combine_messages(errors: &[&Option<Error>]) -> Option<String> {
    let parts: Vec<String> = errors
        .iter()
        .filter_map(|e| e.as_ref().map(|e| e.to_string()))
        .collect();
    if parts.is_empty() {
        return None;
    }
    Some(parts.join("\n"))
}

fn finalize_observation(
    evidence: &Evidence,
    mut observation: Observation,
    operation_err: Option<Error>,
    evidence_err: Option<Error>,
) -> Result<(), Error> {
    read_build_settings(&mut observation);
    let mut evidence_err = Error::join(vec![evidence_err, evidence.check_secrets().err()]);
    let operation_failed = operation_err.is_some();
    let operation_cancelled = operation_err.as_ref().is_some_and(|e| e.is_cancelled());
    if operation_failed || evidence_err.is_some() {
        // No success-shaped final record exists until retention has finalized.
        let combined = combine_messages(&[&operation_err, &evidence_err]).unwrap_or_default();
        let redacted = evidence.redact_string(&combined);
        evidence_err = Error::join(vec![
            evidence_err,
            evidence
                .write("failure.txt", format!("{redacted}\n").as_bytes())
                .err(),
        ]);
    }
    let (hashes, hashes_err) = report::hashes(evidence);
    observation.files = Some(hashes);
    evidence_err = Error::join(vec![evidence_err, hashes_err]);
    observation.evidence = if evidence_err.is_some() {
        "failed".to_string()
    } else {
        "completed".to_string()
    };
    observation.outcome = if operation_failed || evidence_err.is_some() {
        "failed".to_string()
    } else {
        "completed".to_string()
    };
    if operation_cancelled {
        observation.outcome = "cancelled".to_string();
    }
    observation.finished = jsonio::now_rfc3339_nano();
    redact_observation_strings(evidence, &mut observation);
    // No success-shaped final record exists until retention has finalized.
    let publish_err = evidence
        .publish_observation(&report::observation_json(&observation))
        .err();
    match combine_messages(&[&operation_err, &evidence_err, &publish_err]) {
        None => Ok(()),
        Some(combined) => Err(Error::msg(evidence.redact_string(&combined))),
    }
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
