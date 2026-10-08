use std::time::Duration;

use super::files::{
    hash_bounded, open_snapshot_file, snapshot_opened_file, snapshot_regular_file,
    MAX_SNAPSHOT_FILE_BYTES,
};
use super::*;
use crate::files::TempDir;
use crate::sha256::{self, Digest, Sha256};

/// Port of `TestEntryHashAndMode` from `test_u08_state.py`.
#[test]
fn entry_hashes_file_and_reports_mode() {
    use std::os::unix::fs::PermissionsExt;
    let dir = TempDir::new("snapshot").unwrap();
    let path = dir.path().join("probe.txt");
    std::fs::write(&path, b"snapshot-me").unwrap();
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o640)).unwrap();
    let json = Entry::snapshot(&path, true).unwrap().json();
    assert_eq!(json.get("mode").unwrap().as_integer().unwrap(), 0o640);
    assert_eq!(
        json.get("uid").unwrap().as_integer().unwrap(),
        unsafe { libc::getuid() } as i128
    );
    assert_eq!(
        json.get("gid").unwrap().as_integer().unwrap(),
        unsafe { libc::getgid() } as i128
    );
    let mut digest = Sha256::new();
    digest.update(b"snapshot-me");
    assert_eq!(
        json.get("sha256").unwrap().as_str().unwrap(),
        sha256::hex_lower(&digest.finalize())
    );
    assert!(json.get("size").is_none());
    assert!(json.get("link").is_none());
}

/// Port of `TestEntrySizeWithoutContent` from `test_u08_state.py`.
#[test]
fn entry_reports_size_without_hashing() {
    let dir = TempDir::new("snapshot").unwrap();
    let path = dir.path().join("secret.bin");
    std::fs::write(&path, b"0123456789abcdef").unwrap();
    let json = Entry::snapshot(&path, false).unwrap().json();
    assert_eq!(json.get("size").unwrap().as_integer().unwrap(), 16);
    assert!(json.get("sha256").is_none());
}

/// Port of `TestEntryRejectsSpecial` from `test_u08_state.py`.
#[test]
fn entry_rejects_special_files() {
    let err = Entry::snapshot(Path::new("/dev/null"), true).unwrap_err();
    assert_eq!(err.kind, SnapshotKind::RuntimeError);
    assert_eq!(err.detail, "Unsupported snapshot file");
}

/// The owner fails closed on links and missing paths instead of
/// following them or substituting emptiness.
#[test]
fn entry_fails_closed_on_links_and_missing_paths() {
    let dir = TempDir::new("snapshot").unwrap();
    let target = dir.path().join("target.txt");
    std::fs::write(&target, b"data").unwrap();
    let link = dir.path().join("link.txt");
    std::os::unix::fs::symlink(&target, &link).unwrap();
    let err = Entry::snapshot(&link, true).unwrap_err();
    assert_eq!(err.kind, SnapshotKind::FileNotFoundError);
    let err = Entry::snapshot(&dir.path().join("missing"), false).unwrap_err();
    assert_eq!(err.kind, SnapshotKind::FileNotFoundError);
    // A changed file changes its hash.
    let first = Entry::snapshot(&target, true).unwrap();
    std::fs::write(&target, b"changed").unwrap();
    let second = Entry::snapshot(&target, true).unwrap();
    assert_ne!(first.json(), second.json());
}

#[test]
fn entry_hash_and_metadata_stay_bound_to_the_opened_file() {
    use std::os::unix::fs::PermissionsExt;

    let dir = TempDir::new("snapshot-opened-file").unwrap();
    let path = dir.path().join("probe.txt");
    let moved = dir.path().join("opened.txt");
    std::fs::write(&path, b"opened bytes").unwrap();
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o640)).unwrap();
    let opened = open_snapshot_file(&path).unwrap();
    std::fs::rename(&path, &moved).unwrap();
    std::fs::write(&path, b"replacement bytes").unwrap();

    let entry = snapshot_opened_file(opened, true).unwrap();
    let mut digest = Sha256::new();
    digest.update(b"opened bytes");
    let json = entry.json();
    assert_eq!(json.get("mode").unwrap().as_integer().unwrap(), 0o640);
    assert_eq!(
        json.get("sha256").unwrap().as_str().unwrap(),
        sha256::hex_lower(&digest.finalize())
    );
}

#[test]
fn snapshot_file_open_is_nonblocking_and_rejects_fifo() {
    use std::os::fd::AsRawFd;
    use std::os::unix::ffi::OsStrExt;
    use std::sync::mpsc;
    use std::time::Duration;

    let dir = TempDir::new("snapshot-fifo").unwrap();
    let path = dir.path().join("pipe");
    let name = std::ffi::CString::new(path.as_os_str().as_bytes()).unwrap();
    assert_eq!(unsafe { libc::mkfifo(name.as_ptr(), 0o600) }, 0);

    let (opened_tx, opened_rx) = mpsc::channel();
    let open_path = path.clone();
    let opener = std::thread::spawn(move || {
        let nonblocking = open_snapshot_file(&open_path).map(|file| {
            let flags = unsafe { libc::fcntl(file.as_raw_fd(), libc::F_GETFL) };
            flags >= 0 && flags & libc::O_NONBLOCK != 0
        });
        let _ = opened_tx.send(nonblocking);
    });
    let nonblocking = match opened_rx.recv_timeout(Duration::from_millis(100)) {
        Ok(result) => result.unwrap(),
        Err(mpsc::RecvTimeoutError::Timeout) => {
            // Pair with an accidentally blocking FIFO open so the regression
            // fails without leaving a stuck test thread behind.
            let writer = std::fs::OpenOptions::new().write(true).open(&path).unwrap();
            let _ = opened_rx.recv_timeout(Duration::from_secs(1));
            drop(writer);
            opener.join().unwrap();
            panic!("snapshot FIFO open blocked");
        }
        Err(error) => panic!("snapshot FIFO open failed: {error}"),
    };
    opener.join().unwrap();
    assert!(nonblocking);
    let error = snapshot_regular_file(&path, true).unwrap_err();
    assert_eq!(error.kind, SnapshotKind::RuntimeError);
    assert_eq!(error.detail, "Unsupported snapshot file");
}

#[test]
fn snapshot_hash_reads_at_most_cap_plus_one_to_detect_growth() {
    use std::io::Cursor;

    let mut reader = Cursor::new(b"12345".to_vec());
    let error = hash_bounded(&mut reader, 4).unwrap_err();
    assert_eq!(error.kind, SnapshotKind::RuntimeError);
    assert_eq!(error.detail, "Snapshot file too large");
    assert_eq!(reader.position(), 5);
}

#[test]
fn size_only_snapshot_allows_sparse_file_over_hash_cap() {
    let dir = TempDir::new("snapshot-size-cap").unwrap();
    let path = dir.path().join("large.bin");
    let file = std::fs::File::create(&path).unwrap();
    file.set_len(MAX_SNAPSHOT_FILE_BYTES + 1).unwrap();
    drop(file);

    let size_only = Entry::snapshot(&path, false).unwrap().json();
    assert_eq!(
        size_only.get("size").unwrap().as_integer().unwrap(),
        (MAX_SNAPSHOT_FILE_BYTES + 1) as i128
    );
    let hashed = Entry::snapshot(&path, true).unwrap_err();
    assert_eq!(hashed.kind, SnapshotKind::RuntimeError);
    assert_eq!(hashed.detail, "Snapshot file too large");
}

/// The `.ssh` loop exports hashes only for `config`/`known_hosts`,
/// sizes for the rest, skips `authorized_keys`/`u08-personal-git`,
/// and fails the snapshot on a link.
#[test]
fn ssh_files_export_hashes_sizes_and_reject_links() {
    let dir = TempDir::new("snapshot").unwrap();
    let home = dir.path().join("home");
    let ssh = home.join(".ssh");
    std::fs::create_dir_all(&ssh).unwrap();
    std::fs::write(ssh.join("config"), b"Host x\n").unwrap();
    std::fs::write(ssh.join("identity"), b"PRIVATE").unwrap();
    std::fs::write(ssh.join("authorized_keys"), b"ssh-ed25519 AAAA\n").unwrap();
    std::fs::create_dir_all(ssh.join("u08-personal-git")).unwrap();
    let entries = snapshot_ssh_files(&home).unwrap();
    let names: Vec<&str> = entries.iter().map(|(name, _)| name.as_str()).collect();
    assert_eq!(names, vec!["config", "identity"]);
    let config = entries[0].1.clone();
    assert!(config.get("sha256").is_some());
    assert!(config.get("size").is_none());
    let identity = entries[1].1.clone();
    assert_eq!(identity.get("size").unwrap().as_integer().unwrap(), 7);
    assert!(identity.get("sha256").is_none());
    std::os::unix::fs::symlink(ssh.join("config"), ssh.join("alias")).unwrap();
    let err = snapshot_ssh_files(&home).unwrap_err();
    assert_eq!(err.kind, SnapshotKind::FileNotFoundError);
}

#[test]
fn snapshot_gate_requires_root_container() {
    let dir = TempDir::new("snapshot").unwrap();
    let marker = dir.path().join(".containerenv");
    std::fs::write(&marker, b"").unwrap();
    assert!(check_snapshot_gate(0, &marker).is_ok());
    assert_eq!(
        check_snapshot_gate(1000, &marker).unwrap_err(),
        SnapshotFailure::assertion("")
    );
    assert_eq!(
        check_snapshot_gate(0, &dir.path().join("missing")).unwrap_err(),
        SnapshotFailure::assertion("")
    );
}

#[test]
#[cfg(target_os = "linux")]
fn snapshot_command_reports_failures() {
    let ok = command(&arg_list(&["echo", "  hi  "]), &[]).unwrap();
    assert_eq!(ok, "hi");
    let err = command(&arg_list(&["false"]), &[]).unwrap_err();
    assert_eq!(err.kind, SnapshotKind::RuntimeError);
    assert_eq!(err.detail, "Required snapshot command failed: false");
    let err = command_with_timeout(&arg_list(&["sleep", "5"]), &[], Duration::from_millis(100))
        .unwrap_err();
    assert_eq!(err.kind, SnapshotKind::TimeoutExpired);
    let err = command(&arg_list(&["head", "-c", "5000000", "/dev/zero"]), &[]).unwrap_err();
    assert_eq!(err.kind, SnapshotKind::RuntimeError);
    assert_eq!(err.detail, "Snapshot output exceeded bound");
}

/// Owner rule shared with the process tests: never signal a PID read
/// from disk. A missing /proc entry or a zombie state proves the
/// descendant no longer executes.
#[cfg(target_os = "linux")]
fn assert_descendant_retired(pid_file: &std::path::Path) {
    let deadline = std::time::Instant::now() + Duration::from_secs(5);
    let pid = loop {
        if let Ok(raw) = std::fs::read_to_string(pid_file) {
            let trimmed = raw.trim().to_string();
            if !trimmed.is_empty() {
                break trimmed;
            }
        }
        assert!(
            std::time::Instant::now() < deadline,
            "descendant pid never recorded"
        );
        std::thread::sleep(Duration::from_millis(10));
    };
    loop {
        match std::fs::read_to_string(format!("/proc/{pid}/stat")) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => return,
            Err(e) if e.raw_os_error() == Some(libc::ESRCH) => return,
            Err(e) => panic!("{e}"),
            Ok(raw) => {
                let tail = raw.rsplit(')').next().unwrap_or("");
                if tail.split_whitespace().next() == Some("Z") {
                    return;
                }
                assert!(
                    std::time::Instant::now() < deadline,
                    "descendant {pid} survived"
                );
                std::thread::sleep(Duration::from_millis(10));
            }
        }
    }
}

/// ACCEPTANCE-20-001: the timeout path retires the whole group through
/// the owned lifecycle. A backgrounded grandchild holds the pipes for
/// 60s; the call must report TimeoutExpired promptly with no detached
/// readers and no surviving descendant.
#[test]
#[cfg(target_os = "linux")]
fn snapshot_command_timeout_retires_group_and_readers() {
    let dir = TempDir::new("snapshot").unwrap();
    let pid_file = dir.path().join("descendant.pid");
    let script = format!("sleep 60 & echo $! > {}; wait", pid_file.display());
    let start = std::time::Instant::now();
    let err = command_with_timeout(
        &arg_list(&["sh", "-c", script.as_str()]),
        &[],
        Duration::from_millis(200),
    )
    .unwrap_err();
    assert_eq!(err.kind, SnapshotKind::TimeoutExpired);
    // Latency budget, reported separately: 200ms phase deadline plus the
    // owned stop grace (TERM, KILL after 10s, 5s reap wait — 15s worst
    // case), plus scheduling margin.
    assert!(
        start.elapsed() < Duration::from_secs(20),
        "group retirement took {:?}",
        start.elapsed()
    );
    assert_descendant_retired(&pid_file);
}

/// ACCEPTANCE-20-001 D1: the leader exits at once while a setsid daemon
/// holds the pipes past group retirement (foreign: never signalled).
/// Phased pumps exit at the deadline and the call reports TimeoutExpired
/// — bounded and joined, not partial success. The ready file proves the
/// escape completed before the leader exited.
#[test]
#[cfg(target_os = "linux")]
fn snapshot_command_escaped_pipes_cancel_at_deadline() {
    let dir = TempDir::new("snapshot").unwrap();
    let ready = dir.path().join("escaped");
    let script = format!(
        "setsid sh -c 'echo ready > {}; exec sleep 5' & while [ ! -f {} ]; do sleep 0.01; done; echo hi",
        ready.display(),
        ready.display()
    );
    let start = std::time::Instant::now();
    let err = command_with_timeout(
        &arg_list(&["sh", "-c", script.as_str()]),
        &[],
        Duration::from_millis(300),
    )
    .unwrap_err();
    assert_eq!(err.kind, SnapshotKind::TimeoutExpired);
    assert!(
        start.elapsed() < Duration::from_secs(5),
        "escaped pipes outlived the deadline by {:?}",
        start.elapsed()
    );
}

/// ACCEPTANCE-21-001: the leader prints clean stdout and exits while a
/// setsid daemon holds ONLY stderr past group retirement (its stdout is
/// /dev/null, unsignalled foreign). Stdout completes but stderr never
/// does: either required pipe's cancellation prevents success, so the
/// call reports TimeoutExpired — never Ok("complete") on the clean
/// leader alone. The ready file proves the escape completed before the
/// leader exited.
#[test]
#[cfg(target_os = "linux")]
fn snapshot_command_stderr_only_escape_cancels_at_deadline() {
    let dir = TempDir::new("snapshot").unwrap();
    let ready = dir.path().join("escaped");
    let script = format!(
        "setsid sh -c 'echo ready > {}; exec sleep 5' 1>/dev/null & while [ ! -f {} ]; do sleep 0.01; done; echo complete",
        ready.display(),
        ready.display()
    );
    let start = std::time::Instant::now();
    let err = command_with_timeout(
        &arg_list(&["sh", "-c", script.as_str()]),
        &[],
        Duration::from_millis(300),
    )
    .unwrap_err();
    assert_eq!(err.kind, SnapshotKind::TimeoutExpired);
    assert!(
        start.elapsed() < Duration::from_secs(5),
        "stderr-only escape outlived the deadline by {:?}",
        start.elapsed()
    );
}

/// ACCEPTANCE-21-001: mirror — a setsid daemon holds ONLY stdout past
/// group retirement (its stderr is /dev/null) while the leader's own
/// stderr completes. Stdout cancellation still reports TimeoutExpired;
/// stdout behavior is unchanged while stderr joins the completion
/// contract. The ready file proves the escape completed before the
/// leader exited.
#[test]
#[cfg(target_os = "linux")]
fn snapshot_command_stdout_only_escape_cancels_at_deadline() {
    let dir = TempDir::new("snapshot").unwrap();
    let ready = dir.path().join("escaped");
    let script = format!(
        "setsid sh -c 'echo ready > {}; exec sleep 5' 2>/dev/null & while [ ! -f {} ]; do sleep 0.01; done; echo noise >&2",
        ready.display(),
        ready.display()
    );
    let start = std::time::Instant::now();
    let err = command_with_timeout(
        &arg_list(&["sh", "-c", script.as_str()]),
        &[],
        Duration::from_millis(300),
    )
    .unwrap_err();
    assert_eq!(err.kind, SnapshotKind::TimeoutExpired);
    assert!(
        start.elapsed() < Duration::from_secs(5),
        "stdout-only escape outlived the deadline by {:?}",
        start.elapsed()
    );
}

/// ACCEPTANCE-20-001 D1: the leader stays alive on an escaped daemon
/// past the deadline. Owned stop retires the group (the foreign daemon
/// survives, unsignalled), phased pumps exit, and the call reports
/// TimeoutExpired within phase deadline plus stop grace.
#[test]
#[cfg(target_os = "linux")]
fn snapshot_command_timeout_with_escaped_pipes_stays_bounded() {
    let start = std::time::Instant::now();
    let err = command_with_timeout(
        &arg_list(&["sh", "-c", "setsid sleep 5"]),
        &[],
        Duration::from_millis(300),
    )
    .unwrap_err();
    assert_eq!(err.kind, SnapshotKind::TimeoutExpired);
    assert!(
        start.elapsed() < Duration::from_secs(5),
        "escaped timeout took {:?}",
        start.elapsed()
    );
}

/// ACCEPTANCE-20-001: direct-parent exit with inherited pipes. The
/// leader prints and exits at once while a grandchild holds the pipes;
/// owned group retirement reaps the descendant, drains complete bytes,
/// and reports success — nothing detaches and nothing survives.
#[test]
#[cfg(target_os = "linux")]
fn snapshot_command_exit_path_retires_inherited_pipes() {
    let dir = TempDir::new("snapshot").unwrap();
    let pid_file = dir.path().join("descendant.pid");
    let script = format!("echo hello; sleep 60 & echo $! > {}", pid_file.display());
    let start = std::time::Instant::now();
    let out = command_with_timeout(
        &arg_list(&["sh", "-c", script.as_str()]),
        &[],
        Duration::from_secs(30),
    )
    .unwrap();
    assert_eq!(out, "hello");
    assert!(
        start.elapsed() < Duration::from_secs(10),
        "pipe retirement took {:?}",
        start.elapsed()
    );
    assert_descendant_retired(&pid_file);
}

#[test]
fn dumps_sorted_emits_sorted_json_values() {
    let mut value = obj();
    set(&mut value, "b", n(1));
    set(
        &mut value,
        "a",
        JsonValue::Array(vec![JsonValue::Bool(true), s("x\ny")]),
    );
    let encoded = dumps_sorted(&value);
    assert!(encoded.find("\"a\"").unwrap() < encoded.find("\"b\"").unwrap());
    let decoded: serde_json::Value = serde_json::from_str(&encoded).unwrap();
    assert_eq!(decoded["a"], serde_json::json!([true, "x\ny"]));
    assert_eq!(decoded["b"], 1);
    let mut unicode = obj();
    set(&mut unicode, "e", s("é💾"));
    assert_eq!(
        serde_json::from_str::<serde_json::Value>(&dumps_sorted(&unicode)).unwrap()["e"],
        "é💾"
    );
}

#[test]
fn project_network_check_is_strict() {
    assert!(check_project_ip("10.89.0.7").is_ok());
    assert_eq!(
        check_project_ip("10.89.1.7").unwrap_err().kind,
        SnapshotKind::AssertionError
    );
    assert_eq!(
        check_project_ip("not-an-ip").unwrap_err().kind,
        SnapshotKind::ValueError
    );
    assert_eq!(
        check_project_ip("10.89.0.256").unwrap_err().kind,
        SnapshotKind::ValueError
    );
    assert_eq!(
        check_project_ip("10.89.0.07").unwrap_err().kind,
        SnapshotKind::ValueError
    );
    assert_eq!(
        check_project_ip("::1").unwrap_err().kind,
        SnapshotKind::AssertionError
    );
}
