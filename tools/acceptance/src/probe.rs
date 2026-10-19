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
