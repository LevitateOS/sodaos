//! Pinned SSH key probe, mirroring `probe.go` with OpenSSH delegation.
//!
//! The Go owner performs a TLS-style handshake in-process (`x/crypto/ssh`):
//! TCP dial, key exchange against the pinned `known_hosts`, and no
//! authentication. Reimplementing the SSH transport would mean hand-rolled
//! cryptography, so this port delegates the exchange to the host's `ssh`
//! with authentication disabled (`PreferredAuthentications=none`, no
//! identity offered): the key exchange and pin verification still happen,
//! and the observed `SHA256` fingerprint is parsed from verbose output. No
//! password, agent, private key, proxy, keyscan, or Git authentication is
//! used, and auth rejection after the exchange is not an account denial.

use std::time::Duration;

use crate::command::{CommandSpec, Remote, StdinSpec};
use crate::error::Error;
use crate::process::{start_raw_process, Phase};

/// Exchange bound, like the Go owner's 15-second phase.
const EXCHANGE_TIMEOUT: Duration = Duration::from_secs(15);
/// Verbose output cap: a hostile server cannot fill memory.
const OUTPUT_LIMIT: usize = 256 << 10;
const SSH_SERVICE_ACCEPT: &str = "debug1: SSH2_MSG_SERVICE_ACCEPT received";

/// Parse the first exact OpenSSH host-key diagnostic line.
pub fn parse_server_host_key(stderr: &str) -> Option<String> {
    for line in stderr.lines() {
        let Some(rest) = line.strip_prefix("debug1: Server host key:") else {
            continue;
        };
        let mut fields = rest.split_whitespace();
        let (Some(_key_type), Some(fingerprint)) = (fields.next(), fields.next()) else {
            continue;
        };
        if fingerprint.starts_with("SHA256:") {
            return Some(fingerprint.to_string());
        }
    }
    None
}

/// Probe a pinned Git SSH endpoint and return the observed host-key
/// fingerprint. Mirrors `(Remote).ProbeSSHKey`, including the validation
/// order and the phase check after the exchange.
pub fn probe_ssh_key(phase: &Phase, remote: &Remote) -> Result<String, Error> {
    if remote.user != "git"
        || !(1..=65535).contains(&remote.port)
        || !remote.known_hosts.starts_with('/')
    {
        return Err(Error::msg("explicit Git SSH endpoint and pin required"));
    }
    let meta = std::fs::symlink_metadata(&remote.known_hosts)?;
    use std::os::unix::fs::MetadataExt;
    if !meta.is_file() || meta.mode() & 0o022 != 0 {
        return Err(Error::msg("trusted regular known_hosts required"));
    }
    let inner = phase.child(EXCHANGE_TIMEOUT);
    // `knownhosts.New` equivalent: refuse an unparsable pin file before
    // any connection attempt. Keep this child in the same phase as the
    // exchange so preflight cannot outlive the caller's budget.
    let preflight = CommandSpec {
        name: "ssh-keygen".to_string(),
        args: vec![
            "-l".to_string(),
            "-f".to_string(),
            remote.known_hosts.clone(),
        ],
        dir: None,
        stdin: StdinSpec::Null,
        env: Vec::new(),
    };
    let argv = [
        "-F".to_string(),
        "/dev/null".to_string(),
        "-T".to_string(),
        "-v".to_string(),
        "-o".to_string(),
        "BatchMode=yes".to_string(),
        "-o".to_string(),
        "PreferredAuthentications=none".to_string(),
        "-o".to_string(),
        "PasswordAuthentication=no".to_string(),
        "-o".to_string(),
        "KbdInteractiveAuthentication=no".to_string(),
        "-o".to_string(),
        "ChallengeResponseAuthentication=no".to_string(),
        "-o".to_string(),
        "StrictHostKeyChecking=yes".to_string(),
        "-o".to_string(),
        "GlobalKnownHostsFile=/dev/null".to_string(),
        "-o".to_string(),
        format!("UserKnownHostsFile={}", remote.known_hosts),
        "-o".to_string(),
        "ConnectTimeout=10".to_string(),
        "-o".to_string(),
        "ConnectionAttempts=1".to_string(),
        "-o".to_string(),
        "FingerprintHash=sha256".to_string(),
        "-l".to_string(),
        remote.user.clone(),
        "-p".to_string(),
        remote.port.to_string(),
        remote.host.clone(),
        "true".to_string(),
    ];
    let spec = CommandSpec {
        name: "ssh".to_string(),
        args: argv.to_vec(),
        dir: None,
        stdin: StdinSpec::Null,
        env: Vec::new(),
    };
    let fingerprint = run_pinned_exchange(&inner, &preflight, &spec)?;
    inner.check()?;
    Ok(fingerprint)
}

fn run_pinned_exchange(
    phase: &Phase,
    preflight: &CommandSpec,
    exchange: &CommandSpec,
) -> Result<String, Error> {
    let checked = run_owned_capture(phase, preflight, 0, 0)?;
    if checked.exit_code != Some(0) || checked.wait.is_err() {
        return Err(Error::msg("invalid pinned known_hosts file"));
    }

    run_exchange(phase, exchange)
}

struct OwnedCapture {
    exit_code: Option<i32>,
    wait: Result<(), Error>,
    stdout: Vec<u8>,
    stdout_overflow: bool,
    stderr: Vec<u8>,
    stderr_overflow: bool,
}

/// Run and retire one child through the shared process owner. Exit status and
/// bounded bytes are returned only after cleanup and both pumps are confirmed.
fn run_owned_capture(
    phase: &Phase,
    spec: &CommandSpec,
    stdout_cap: usize,
    stderr_cap: usize,
) -> Result<OwnedCapture, Error> {
    phase.check()?;
    let (process, stdout, stderr) = start_raw_process(phase, spec, stdout_cap, stderr_cap)?;
    let wait = process.wait(phase);
    if !process.is_done() {
        let stop = process.stop();
        let pumps = process.join_pumps();
        let phase_error = phase.check().err();
        if !process.is_done() || stop.is_err() || pumps.is_some() {
            return Err(Error::join(vec![phase_error, Some(exchange_failed())])
                .unwrap_or_else(exchange_failed));
        }
        return Err(phase_error.unwrap_or_else(exchange_failed));
    }

    let outcome = process.outcome().ok_or_else(exchange_failed)?;
    let pump_error = process.join_pumps();
    let stdout_cancelled = stdout.cancelled();
    let stderr_cancelled = stderr.cancelled();
    let (stdout, stdout_overflow) = stdout.take();
    let (stderr, stderr_overflow) = stderr.take();
    let phase_error = phase.check().err();
    if phase_error.is_some()
        || outcome.cleanup_message.is_some()
        || pump_error.is_some()
        || stdout_cancelled
        || stderr_cancelled
    {
        return Err(phase_error.unwrap_or_else(exchange_failed));
    }

    Ok(OwnedCapture {
        exit_code: outcome.exit_code,
        wait,
        stdout,
        stdout_overflow,
        stderr,
        stderr_overflow,
    })
}

fn run_exchange(phase: &Phase, spec: &CommandSpec) -> Result<String, Error> {
    let captured = run_owned_capture(phase, spec, 0, OUTPUT_LIMIT)?;
    let expected_exit = match captured.exit_code {
        Some(0) => captured.wait.is_ok(),
        Some(255) => captured.wait.is_err(),
        _ => false,
    };
    if !expected_exit
        || captured.stdout_overflow
        || captured.stderr_overflow
        || !captured.stdout.is_empty()
    {
        return Err(exchange_failed());
    }

    verified_server_host_key(&captured.stderr).ok_or_else(exchange_failed)
}

fn verified_server_host_key(stderr: &[u8]) -> Option<String> {
    let stderr = std::str::from_utf8(stderr).ok()?;
    let mut fingerprint = None;
    for line in stderr.lines() {
        if fingerprint.is_none() {
            fingerprint = parse_server_host_key(line);
        } else if line == SSH_SERVICE_ACCEPT {
            return fingerprint;
        }
    }
    None
}

fn exchange_failed() -> Error {
    Error::msg("pinned SSH exchange was not verified")
}

#[cfg(test)]
mod tests {
    use super::*;

    fn fixture_dir(prefix: &str) -> std::path::PathBuf {
        let mut dir = std::env::temp_dir();
        dir.push(format!("soda-probe-{prefix}-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    fn remote_for(dir: &std::path::Path, known_hosts: &str) -> Remote {
        let _ = dir;
        Remote {
            user: "git".to_string(),
            host: "127.0.0.1".to_string(),
            key: String::new(),
            known_hosts: known_hosts.to_string(),
            port: 1,
            timeout: Duration::ZERO,
        }
    }

    fn exchange_command(script: &str) -> CommandSpec {
        CommandSpec {
            name: "/bin/sh".to_string(),
            args: vec!["-c".to_string(), script.to_string()],
            dir: None,
            stdin: StdinSpec::Null,
            env: Vec::new(),
        }
    }

    fn pinned_hosts(dir: &std::path::Path, name: &str) -> String {
        // Real key material: both ssh-keygen and Go's knownhosts parser
        // reject mock blobs.
        let key = dir.join("probe-key");
        let status = std::process::Command::new("ssh-keygen")
            .args(["-q", "-t", "ed25519", "-N", "", "-f"])
            .arg(&key)
            .stdin(std::process::Stdio::null())
            .stdout(std::process::Stdio::null())
            .stderr(std::process::Stdio::null())
            .status()
            .unwrap();
        assert!(status.success());
        let public = std::fs::read_to_string(dir.join("probe-key.pub")).unwrap();
        let public = public
            .split_whitespace()
            .take(2)
            .collect::<Vec<_>>()
            .join(" ");
        let hosts = dir.join(name);
        std::fs::write(&hosts, format!("[127.0.0.1]:1 {public}\n")).unwrap();
        hosts.to_string_lossy().into_owned()
    }

    #[test]
    fn endpoint_and_pin_gates() {
        let dir = fixture_dir("gates");
        let good = pinned_hosts(&dir, "known_hosts");
        let mut remote = remote_for(&dir, &good);
        // Refused instantly: no server on port 1, so no fingerprint.
        let err = probe_ssh_key(&Phase::background(), &remote).unwrap_err();
        assert!(
            err.to_string()
                .contains("pinned SSH exchange was not verified"),
            "{err}"
        );

        remote.user = "operator".to_string();
        assert_eq!(
            probe_ssh_key(&Phase::background(), &remote)
                .unwrap_err()
                .to_string(),
            "explicit Git SSH endpoint and pin required"
        );
        remote.user = "git".to_string();
        remote.port = 70000;
        assert!(probe_ssh_key(&Phase::background(), &remote).is_err());
        remote.port = 1;
        remote.known_hosts = "relative".to_string();
        assert!(probe_ssh_key(&Phase::background(), &remote).is_err());

        let open = dir.join("open_hosts");
        std::fs::write(&open, "x").unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&open, std::fs::Permissions::from_mode(0o666)).unwrap();
        remote.known_hosts = open.to_string_lossy().into_owned();
        assert_eq!(
            probe_ssh_key(&Phase::background(), &remote)
                .unwrap_err()
                .to_string(),
            "trusted regular known_hosts required"
        );

        let garbage = dir.join("garbage_hosts");
        std::fs::write(&garbage, "this is not a known_hosts file at all\n").unwrap();
        remote.known_hosts = garbage.to_string_lossy().into_owned();
        assert_eq!(
            probe_ssh_key(&Phase::background(), &remote)
                .unwrap_err()
                .to_string(),
            "invalid pinned known_hosts file"
        );

        remote.known_hosts = dir.join("missing").to_string_lossy().into_owned();
        let err = probe_ssh_key(&Phase::background(), &remote).unwrap_err();
        assert_eq!(err.io_kind(), Some(std::io::ErrorKind::NotFound));
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn cancelled_phase_wins() {
        let dir = fixture_dir("cancel");
        let good = pinned_hosts(&dir, "known_hosts");
        let remote = remote_for(&dir, &good);
        let phase = Phase::background();
        phase.cancel();
        assert!(probe_ssh_key(&phase, &remote).unwrap_err().is_cancelled());
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn fingerprint_lines_parse() {
        let stderr = "debug1: Connecting to 127.0.0.1 [127.0.0.1] port 22.\ndebug1: Server host key: ssh-ed25519 SHA256:abc123+/=\ndebug1: Authenticated.\n";
        assert_eq!(
            parse_server_host_key(stderr).as_deref(),
            Some("SHA256:abc123+/=")
        );
        assert_eq!(parse_server_host_key("no keys here\n"), None);
        assert_eq!(
            parse_server_host_key("debug1: Server host key: ssh-rsa MD5:aa:bb\n"),
            None
        );
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn exchange_requires_ordered_verified_key_and_service_accept() {
        let key = "debug1: Server host key: ssh-ed25519 SHA256:fixture-pin";
        let marker = SSH_SERVICE_ACCEPT;
        for (script, accepted) in [
            (
                format!("printf '%s\\n%s\\n' '{key}' '{marker}' >&2; exit 255"),
                true,
            ),
            (
                format!("printf '%s\\n%s\\n' '{key}' '{marker}' >&2; exit 0"),
                true,
            ),
            (format!("printf '%s\\n' '{key}' >&2; exit 255"), false),
            (
                format!("printf '%s\\n%s\\n' '{marker}' '{key}' >&2; exit 255"),
                false,
            ),
            (
                format!("printf '%s\\n' 'remote banner {key}' '{marker}' >&2; exit 255"),
                false,
            ),
        ] {
            let result = run_exchange(
                &Phase::timeout(Duration::from_secs(3)),
                &exchange_command(&script),
            );
            assert_eq!(
                result.as_deref().ok(),
                accepted.then_some("SHA256:fixture-pin"),
                "script result: {result:?}"
            );
        }
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn exchange_rejects_changed_key_and_unexpected_status() {
        let key = "debug1: Server host key: ssh-ed25519 SHA256:changed-pin";
        let marker = SSH_SERVICE_ACCEPT;
        let changed_pin = format!("printf '%s\\n' '{key}' >&2; exit 255");
        let unexpected_status = format!("printf '%s\\n%s\\n' '{key}' '{marker}' >&2; exit 1");
        for script in [changed_pin, unexpected_status] {
            assert!(run_exchange(
                &Phase::timeout(Duration::from_secs(3)),
                &exchange_command(&script),
            )
            .is_err());
        }
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn exchange_refuses_output_overflow_and_deadline() {
        let key = "debug1: Server host key: ssh-ed25519 SHA256:fixture-pin";
        let marker = SSH_SERVICE_ACCEPT;
        let prefix = format!("{key}\n{marker}\n");
        let filler = OUTPUT_LIMIT - prefix.len();
        let exact = format!(
            "printf '%s' '{prefix}' >&2; head -c {filler} /dev/zero | tr '\\000' x >&2; exit 255"
        );
        let exact_result = run_exchange(
            &Phase::timeout(Duration::from_secs(5)),
            &exchange_command(&exact),
        );
        assert!(matches!(exact_result.as_deref(), Ok("SHA256:fixture-pin")));

        let over = format!(
            "printf '%s' '{prefix}' >&2; head -c {} /dev/zero | tr '\\000' x >&2; exit 255",
            filler + 1
        );
        assert!(run_exchange(
            &Phase::timeout(Duration::from_secs(5)),
            &exchange_command(&over),
        )
        .is_err());

        let stdout = format!("printf x; printf '%s\\n%s\\n' '{key}' '{marker}' >&2; exit 255");
        assert!(run_exchange(
            &Phase::timeout(Duration::from_secs(3)),
            &exchange_command(&stdout),
        )
        .is_err());

        let start = std::time::Instant::now();
        let timed_out = run_exchange(
            &Phase::timeout(Duration::from_millis(100)),
            &exchange_command("sleep 5"),
        );
        assert!(timed_out.is_err());
        assert!(start.elapsed() < Duration::from_secs(3));
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn preflight_uses_the_exchange_phase_and_must_be_owned() {
        let key = "debug1: Server host key: ssh-ed25519 SHA256:fixture-pin";
        let marker = SSH_SERVICE_ACCEPT;
        let result = run_pinned_exchange(
            &Phase::timeout(Duration::from_secs(3)),
            &exchange_command("printf keygen-output; printf keygen-diagnostic >&2; exit 0"),
            &exchange_command(&format!(
                "printf '%s\\n%s\\n' '{key}' '{marker}' >&2; exit 255"
            )),
        );
        assert_eq!(result.as_deref().ok(), Some("SHA256:fixture-pin"));

        assert_eq!(
            run_pinned_exchange(
                &Phase::timeout(Duration::from_secs(3)),
                &exchange_command("exit 1"),
                &exchange_command("exit 0"),
            )
            .unwrap_err()
            .to_string(),
            "invalid pinned known_hosts file"
        );

        let dir = fixture_dir("preflight-not-started");
        let marker_file = dir.join("started");
        let script = format!("touch '{}'", marker_file.display());
        for phase in [Phase::timeout(Duration::ZERO), {
            let phase = Phase::background();
            phase.cancel();
            phase
        }] {
            assert!(run_pinned_exchange(
                &phase,
                &exchange_command(&script),
                &exchange_command("exit 0"),
            )
            .is_err());
            assert!(!marker_file.exists());
        }
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn preflight_stall_consumes_the_same_absolute_deadline() {
        let start = std::time::Instant::now();
        let result = run_pinned_exchange(
            &Phase::timeout(Duration::from_millis(150)),
            &exchange_command("sleep 5"),
            &exchange_command("exit 0"),
        );
        assert!(result.is_err());
        assert!(start.elapsed() < Duration::from_secs(3));

        let start = std::time::Instant::now();
        let result = run_pinned_exchange(
            &Phase::timeout(Duration::from_millis(400)),
            &exchange_command("sleep 0.2"),
            &exchange_command("sleep 5"),
        );
        assert!(result.is_err());
        assert!(start.elapsed() < Duration::from_secs(3));
    }
}
