use std::io;

use super::process::{exec_diag, signal_name, spawn_diag};
use super::state::{pid_alive, pid_display, ssh_args};
use super::stripped;

#[test]
fn pid_display_keeps_spaces_drops_newlines() {
    assert_eq!(pid_display(b"  3512  \n\n"), "  3512  ");
    assert_eq!(pid_display(b"42"), "42");
    assert_eq!(pid_display(b""), "");
}

#[test]
fn pid_alive_matches_kill_zero_semantics() {
    let mine = std::process::id().to_string();
    assert!(pid_alive(mine.as_bytes()));
    assert!(pid_alive(format!("  {mine}  \n").as_bytes()));
    assert!(pid_alive(format!("0{mine}\n").as_bytes()));
    assert!(pid_alive(b"0"));
    assert!(!pid_alive(b""));
    assert!(!pid_alive(b"\n\n"));
    assert!(!pid_alive(b"abc\n"));
    assert!(!pid_alive(b"12 3\n"));
    assert!(!pid_alive(b"-5\n"));
    assert!(!pid_alive(b"2147483647\n"));
    assert!(!pid_alive(b"9999999999\n"));
}

#[test]
fn ssh_args_pin_key_paths_and_options() {
    assert_eq!(
        ssh_args("/repo/.artifacts/test-vm"),
        vec![
            "-p",
            "22220",
            "-i",
            "/repo/.artifacts/test-vm/operator",
            "-o",
            "IdentitiesOnly=yes",
            "-o",
            "StrictHostKeyChecking=yes",
            "-o",
            "UserKnownHostsFile=/repo/.artifacts/test-vm/known_hosts",
        ]
        .into_iter()
        .map(|part| part.to_string())
        .collect::<Vec<String>>(),
    );
}

#[test]
fn stripped_drops_all_trailing_newlines() {
    assert_eq!(stripped(b"Linux\n"), b"Linux");
    assert_eq!(stripped(b"x\n\n\n"), b"x");
    assert_eq!(stripped(b""), b"");
}

#[test]
fn signal_names_come_from_strsignal() {
    assert_eq!(signal_name(15), "Terminated");
    assert_eq!(signal_name(11), "Segmentation fault");
    assert_eq!(signal_name(1), "Hangup");
    assert_eq!(signal_name(6), "Aborted");
}

#[test]
fn spawn_diag_matrix_matches_bash() {
    // Slashless misses say "command not found" (PATH lookups here run
    // against the ambient PATH, where these names do not exist).
    let (msg, code) = spawn_diag(
        "definitely-missing-soda-vm-probe",
        &io::Error::from_raw_os_error(2),
    );
    assert_eq!((msg.as_str(), code), ("command not found", 127));
    // Direct paths report the raw strerror.
    let (msg, code) = spawn_diag(
        "/nonexistent-soda-vm-probe/foo",
        &io::Error::from_raw_os_error(2),
    );
    assert_eq!((msg.as_str(), code), ("No such file or directory", 127));
    // Directories are reported as such.
    let (msg, code) = spawn_diag("/tmp", &io::Error::from_raw_os_error(13));
    assert_eq!((msg.as_str(), code), ("Is a directory", 126));
}

#[test]
fn exec_diag_matrix_matches_bash() {
    // Slashless misses use the one-line "not found" shape.
    let (lines, code) = exec_diag(
        "definitely-missing-soda-vm-probe",
        &io::Error::from_raw_os_error(2),
    );
    assert_eq!(code, 127);
    assert_eq!(
        lines,
        vec!["test-vm: exec: definitely-missing-soda-vm-probe: not found".to_string()]
    );
    // Direct-path misses use the bare strerror without the exec part.
    let (lines, code) = exec_diag(
        "/nonexistent-soda-vm-probe/foo",
        &io::Error::from_raw_os_error(2),
    );
    assert_eq!(code, 127);
    assert_eq!(
        lines,
        vec!["test-vm: /nonexistent-soda-vm-probe/foo: No such file or directory".to_string()]
    );
    // Directories get the two-line shape reporting "Is a directory".
    let (lines, code) = exec_diag("/tmp", &io::Error::from_raw_os_error(13));
    assert_eq!(code, 126);
    assert_eq!(
        lines,
        vec![
            "test-vm: /tmp: Is a directory".to_string(),
            "test-vm: exec: /tmp: cannot execute: Is a directory".to_string(),
        ]
    );
}
