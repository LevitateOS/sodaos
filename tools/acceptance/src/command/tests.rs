use std::time::Duration;

use crate::evidence::Evidence;
use crate::process::Phase;

use super::*;

fn fixture_evidence() -> (std::path::PathBuf, Evidence) {
    let mut dir = std::env::temp_dir();
    dir.push(format!(
        "soda-command-{}-{}",
        std::process::id(),
        fresh_id()
    ));
    std::fs::create_dir_all(&dir).unwrap();
    let path = dir.join("evidence").to_string_lossy().into_owned();
    let evidence = crate::evidence::create_evidence(&path, &[]).unwrap();
    (dir, evidence)
}

fn fresh_id() -> u64 {
    use std::sync::atomic::{AtomicU64, Ordering};
    static NEXT: AtomicU64 = AtomicU64::new(0);
    NEXT.fetch_add(1, Ordering::SeqCst)
}

#[test]
fn literal_remote_arguments() {
    let values = ["", "a b", "'quoted'", "a; b", "$(false)", "two\nlines"];
    let mut args = vec!["printf".to_string(), "%s\\0".to_string()];
    args.extend(values.iter().map(|s| s.to_string()));
    let output = std::process::Command::new("/bin/sh")
        .arg("-c")
        .arg(quote(&args))
        .output()
        .unwrap();
    assert!(output.status.success());
    let mut want = values.join("\x00");
    want.push('\x00');
    assert_eq!(String::from_utf8_lossy(&output.stdout), want);
}

#[test]
fn pinned_ssh_options() {
    let mut dir = std::env::temp_dir();
    dir.push(format!("soda-remote-{}-{}", std::process::id(), fresh_id()));
    std::fs::create_dir_all(&dir).unwrap();
    let key = dir.join("key");
    let hosts = dir.join("known_hosts");
    std::fs::write(&key, b"synthetic identity, not used for authentication").unwrap();
    std::fs::write(&hosts, b"synthetic pin, not used for authentication").unwrap();
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(&key, std::fs::Permissions::from_mode(0o600)).unwrap();
    let remote = Remote {
        user: "root".to_string(),
        host: "127.0.0.1".to_string(),
        key: key.to_string_lossy().into_owned(),
        known_hosts: hosts.to_string_lossy().into_owned(),
        port: 22222,
        timeout: Duration::ZERO,
    };
    let command = remote
        .command(
            &["printf".to_string(), "%s".to_string(), "a b".to_string()],
            StdinSpec::Bytes(b"input".to_vec()),
        )
        .unwrap();
    let joined = command.args.join(" ");
    for required in [
        "StrictHostKeyChecking=yes",
        "IdentitiesOnly=yes",
        "GlobalKnownHostsFile=/dev/null",
        "timeout",
        "--kill-after=10s",
    ] {
        assert!(joined.contains(required), "{joined}");
    }
    assert!(!joined.contains("accept-new"), "{joined}");
    assert!(!joined.contains("ProxyJump"), "{joined}");
    std::fs::set_permissions(&key, std::fs::Permissions::from_mode(0o644)).unwrap();
    assert!(remote.args().is_err());
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
#[cfg(target_os = "linux")]
fn command_and_evidence_failures_are_separate() {
    let (_dir, evidence) = secret_evidence();
    let spec = CommandSpec {
        name: "/bin/sh".to_string(),
        args: vec![
            "-c".to_string(),
            "read -r value; printf '%s\\n' \"$value\"; printf 'expected denial\\n' >&2; exit 23"
                .to_string(),
        ],
        dir: None,
        stdin: StdinSpec::Bytes(b"synthetic-password\n".to_vec()),
        env: Vec::new(),
    };
    let (result, err) = execute(&Phase::background(), &evidence, "denied", &spec);
    assert!(err.is_none());
    assert!(result.started);
    assert_eq!(result.exit_code, Some(23));
    assert!(result.err.is_some());
    assert!(!String::from_utf8_lossy(&result.stdout).contains("synthetic-password"));

    evidence.write("collision.stdout", b"").unwrap();
    let idle = CommandSpec {
        name: "/bin/sh".to_string(),
        args: vec!["-c".to_string(), "exit 0".to_string()],
        dir: None,
        stdin: StdinSpec::Null,
        env: Vec::new(),
    };
    let (result, err) = execute(&Phase::background(), &evidence, "collision", &idle);
    assert!(err.is_some());
    assert!(!result.started);
    assert!(result.err.is_none());
}

#[test]
#[cfg(target_os = "linux")]
fn output_limit_failure_survives_nonzero_native_exit() {
    let (_dir, evidence) = fixture_evidence();
    let spec = CommandSpec {
        name: "/bin/sh".to_string(),
        args: vec![
            "-c".to_string(),
            "head -c 16777217 /dev/zero | tr '\\000' '\\n'; exit 23".to_string(),
        ],
        dir: None,
        stdin: StdinSpec::Null,
        env: Vec::new(),
    };

    let (result, evidence_err) = execute(&Phase::background(), &evidence, "too-large", &spec);

    assert!(
        evidence_err.is_some(),
        "capture failure must be evidence failure"
    );
    assert!(result.err.is_some(), "exit 23 must remain a native failure");
    assert_eq!(result.exit_code, Some(23));
    assert!(
        result.stdout.is_empty(),
        "partial capture must not be decoded"
    );
}

#[test]
#[cfg(target_os = "linux")]
fn cancelled_command_is_not_denial_or_success() {
    let (_dir, evidence) = fixture_evidence_pair();
    let phase = Phase::background();
    phase.cancel();
    let spec = CommandSpec {
        name: "/bin/sh".to_string(),
        args: vec!["-c".to_string(), "exit 0".to_string()],
        dir: None,
        stdin: StdinSpec::Null,
        env: Vec::new(),
    };
    let (result, err) = execute(&phase, &evidence, "cancelled", &spec);
    assert!(err.is_none());
    assert!(!result.started);
    assert!(result.err.as_ref().is_some_and(|e| e.is_cancelled()));
}

fn secret_evidence() -> (std::path::PathBuf, Evidence) {
    let mut dir = std::env::temp_dir();
    dir.push(format!(
        "soda-command-secret-{}-{}",
        std::process::id(),
        fresh_id()
    ));
    std::fs::create_dir_all(&dir).unwrap();
    let path = dir.join("evidence").to_string_lossy().into_owned();
    let evidence =
        crate::evidence::create_evidence(&path, &[b"synthetic-password".to_vec()]).unwrap();
    (dir, evidence)
}

fn fixture_evidence_pair() -> (std::path::PathBuf, Evidence) {
    fixture_evidence()
}
