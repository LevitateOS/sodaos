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

use std::time::{Duration, Instant};

use crate::command::Remote;
use crate::error::Error;
use crate::process::Phase;

/// Exchange bound, like the Go owner's 15-second phase.
const EXCHANGE_TIMEOUT: Duration = Duration::from_secs(15);
/// Verbose output cap: a hostile server cannot fill memory.
const OUTPUT_LIMIT: u64 = 256 << 10;

/// Parse the first `Server host key:` fingerprint from verbose `ssh`
/// output, like Go's `ssh.FingerprintSHA256` recording.
pub fn parse_server_host_key(stderr: &str) -> Option<String> {
    for line in stderr.lines() {
        let Some(rest) = line.split("Server host key:").nth(1) else {
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
    if remote.user != "git" || !(1..=65535).contains(&remote.port) || !remote.known_hosts.starts_with('/') {
        return Err(Error::msg("explicit Git SSH endpoint and pin required"));
    }
    let meta = std::fs::symlink_metadata(&remote.known_hosts)?;
    use std::os::unix::fs::MetadataExt;
    if !meta.is_file() || meta.mode() & 0o022 != 0 {
        return Err(Error::msg("trusted regular known_hosts required"));
    }
    // `knownhosts.New` equivalent: refuse an unparsable pin file before
    // any connection attempt.
    let parsed = std::process::Command::new("ssh-keygen")
        .args(["-l", "-f", &remote.known_hosts])
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .status()
        .map_err(Error::from)?;
    if !parsed.success() {
        return Err(Error::msg("invalid pinned known_hosts file"));
    }
    let inner = phase.child(EXCHANGE_TIMEOUT);
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
    let stderr = run_exchange(&inner, &argv)?;
    if let Err(e) = inner.check() {
        return Err(e);
    }
    if let Some(fingerprint) = parse_server_host_key(&stderr) {
        return Ok(fingerprint);
    }
    let detail = stderr.lines().rev().find(|line| !line.trim().is_empty()).unwrap_or("ssh failed").to_string();
    Err(Error::join(vec![Some(Error::msg("pinned endpoint key was not observed")), Some(Error::msg(detail))])
        .unwrap_or_else(|| Error::msg("pinned endpoint key was not observed")))
}

fn run_exchange(phase: &Phase, argv: &[String]) -> Result<String, Error> {
    if phase.check().is_err() {
        return Ok(String::new());
    }
    let mut child = std::process::Command::new("ssh")
        .args(argv)
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::piped())
        .spawn()
        .map_err(Error::from)?;
    let deadline = Instant::now() + EXCHANGE_TIMEOUT;
    loop {
        match child.try_wait().map_err(Error::from)? {
            Some(_) => break,
            None => {
                if phase.check().is_err() || Instant::now() >= deadline {
                    let _ = child.kill();
                    break;
                }
                std::thread::sleep(Duration::from_millis(50));
            }
        }
    }
    let _ = child.wait();
    let mut stderr = String::new();
    if let Some(pipe) = child.stderr.take() {
        use std::io::Read;
        let mut buf = Vec::new();
        let _ = pipe.take(OUTPUT_LIMIT + 1).read_to_end(&mut buf);
        stderr = String::from_utf8_lossy(&buf).into_owned();
    }
    Ok(stderr)
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
        let public = public.split_whitespace().take(2).collect::<Vec<_>>().join(" ");
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
        assert!(err.to_string().contains("pinned endpoint key was not observed"), "{err}");

        remote.user = "operator".to_string();
        assert_eq!(
            probe_ssh_key(&Phase::background(), &remote).unwrap_err().to_string(),
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
            probe_ssh_key(&Phase::background(), &remote).unwrap_err().to_string(),
            "trusted regular known_hosts required"
        );

        let garbage = dir.join("garbage_hosts");
        std::fs::write(&garbage, "this is not a known_hosts file at all\n").unwrap();
        remote.known_hosts = garbage.to_string_lossy().into_owned();
        assert_eq!(
            probe_ssh_key(&Phase::background(), &remote).unwrap_err().to_string(),
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
        assert_eq!(parse_server_host_key(stderr).as_deref(), Some("SHA256:abc123+/="));
        assert_eq!(parse_server_host_key("no keys here\n"), None);
        assert_eq!(parse_server_host_key("debug1: Server host key: ssh-rsa MD5:aa:bb\n"), None);
    }
}
