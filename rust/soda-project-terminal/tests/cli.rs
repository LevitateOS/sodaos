//! CLI contract tests: exact bytes and exit codes for the failure paths
//! that never touch privileged state (bad argv, undecodable stdin).

use std::io::Write;
use std::path::PathBuf;
use std::process::Stdio;

fn bin_path() -> PathBuf {
    if let Some(path) = option_env!("CARGO_BIN_EXE_project_terminal") {
        return PathBuf::from(path);
    }
    let mut dir = std::env::current_exe().expect("current exe");
    dir.pop();
    if dir.file_name().is_some_and(|n| n == "deps") {
        dir.pop();
    }
    dir.join("project-terminal")
}

fn run_stdin(args: &[&str], stdin: &[u8]) -> (i32, Vec<u8>, Vec<u8>) {
    let mut child = std::process::Command::new(bin_path())
        .args(args)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .expect("spawn project-terminal");
    child
        .stdin
        .take()
        .expect("stdin")
        .write_all(stdin)
        .expect("write stdin");
    let output = child.wait_with_output().expect("wait");
    (
        output.status.code().unwrap_or(-1),
        output.stdout,
        output.stderr,
    )
}

const CLOSED: &[u8] = b"{\"type\":\"closed\",\"reason\":\"launch_failed\"}\n";

#[test]
fn prepare_bogus_id_closed() {
    // Invalid identifier: `terminal_path` fails before any privileged touch.
    let (code, out, err) = run_stdin(&["prepare", "bogus"], b"");
    assert_eq!(code, 1);
    assert_eq!(out, CLOSED);
    assert!(err.is_empty());
}

#[test]
fn prepare_missing_locator_closed() {
    // Well-formed but absent: read-only probes, then the closed line.
    let id = "a".repeat(32);
    let (code, out, err) = run_stdin(&["prepare", &id], b"");
    assert_eq!(code, 1);
    assert_eq!(out, CLOSED);
    assert!(err.is_empty());
}

#[test]
fn bad_argv_closed() {
    for args in [
        vec![],
        vec!["frobnicate"],
        vec!["prepare"],
        vec!["broker", "x"],
        vec!["keys", "x", "y"],
    ] {
        let (code, out, err) = run_stdin(&args, b"");
        assert_eq!(code, 1, "{args:?}");
        assert_eq!(out.as_slice(), CLOSED, "{args:?}");
        assert!(err.is_empty(), "{args:?}");
    }
}

#[test]
fn control_bad_identifier_closed() {
    let args = [
        "attach", "ZZZ", "op", "42", "80", "24", "3600", "sh", "term", "",
    ];
    let (code, out, err) = run_stdin(&args, b"");
    assert_eq!(code, 1);
    assert_eq!(out, CLOSED);
    assert!(err.is_empty());
}

const KEYS_ERR: &[u8] = b"native key operation not confirmed\n";

#[test]
fn keys_invalid_stdin() {
    for stdin in [b"".as_slice(), b"{oops", b"[]", b"null"] {
        let (code, out, err) = run_stdin(&["keys"], stdin);
        assert_eq!(code, 1);
        assert!(out.is_empty());
        assert_eq!(err, KEYS_ERR);
    }
}

#[test]
fn keys_duplicate_field_rejected() {
    let stdin = b"{\"login\":\"op\",\"login\":\"x\",\"identity\":7,\"apply\":true,\"revision\":\"\",\"keys\":[]}";
    let (code, out, err) = run_stdin(&["keys"], stdin);
    assert_eq!(code, 1);
    assert!(out.is_empty());
    assert_eq!(err, KEYS_ERR);
}

#[test]
fn keys_wrong_shape_rejected() {
    // Valid JSON but wrong key set: fails before any privileged touch.
    let stdin = b"{\"login\":\"op\"}";
    let (code, out, err) = run_stdin(&["keys"], stdin);
    assert_eq!(code, 1);
    assert!(out.is_empty());
    assert_eq!(err, KEYS_ERR);
}

#[test]
fn keys_valid_shape_unconfirmed_without_identity() {
    // Decodes fine; then fails closed (non-root euid gate, or the missing
    // account marker when root): stdout stays empty either way.
    let stdin = br#"{"login":"zzznosuchuserq","identity":7,"apply":false,"revision":"","keys":[]}"#;
    let (code, out, err) = run_stdin(&["keys"], stdin);
    assert_eq!(code, 1);
    assert!(out.is_empty());
    assert_eq!(err, KEYS_ERR);
}

const BROKER_ERR: &[u8] = b"managed Codex operation failed\n";

#[test]
fn broker_invalid_stdin() {
    for stdin in [b"".as_slice(), b"{oops", b"\"str\""] {
        let (code, out, err) = run_stdin(&["broker"], stdin);
        assert_eq!(code, 1);
        assert!(out.is_empty());
        assert_eq!(err, BROKER_ERR);
    }
}

#[test]
fn broker_oversize_rejected() {
    let big = vec![b'x'; 512 * 1024 + 1];
    let (code, out, err) = run_stdin(&["broker"], &big);
    assert_eq!(code, 1);
    assert!(out.is_empty());
    assert_eq!(err, BROKER_ERR);
}

#[test]
fn broker_unknown_action_without_delivery() {
    // Routes past decode, then fails before any privileged touch.
    let (code, out, err) = run_stdin(&["broker"], b"{\"action\":\"bogus\"}");
    assert_eq!(code, 1);
    assert!(out.is_empty());
    assert_eq!(err, BROKER_ERR);
}

#[test]
fn broker_missing_action_rejected() {
    let (code, out, err) = run_stdin(&["broker"], b"{}");
    assert_eq!(code, 1);
    assert!(out.is_empty());
    assert_eq!(err, BROKER_ERR);
}
