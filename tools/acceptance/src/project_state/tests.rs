use std::time::Duration;

use super::*;
use crate::files::TempDir;
use crate::sha256::{self, Sha256};

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

#[test]
fn dumps_sorted_matches_python_separators() {
    let mut value = obj();
    set(&mut value, "b", n(1));
    set(
        &mut value,
        "a",
        JsonValue::Array(vec![JsonValue::Bool(true), s("x\ny")]),
    );
    assert_eq!(dumps_sorted(&value), r#"{"a": [true, "x\ny"], "b": 1}"#);
    let mut unicode = obj();
    set(&mut unicode, "e", s("é💾"));
    assert_eq!(dumps_sorted(&unicode), "{\"e\": \"\\u00e9\\ud83d\\udcbe\"}");
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
