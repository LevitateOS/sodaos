#![cfg(unix)]

// Shared across suites; each suite uses a subset.
#[allow(dead_code)]
#[path = "locales_support/mod.rs"]
mod locales_support;

use locales_support::{run, status_of, TempDir};

#[test]
fn native_or_lock_is_required() {
    let dir = TempDir::new("required");
    let output = run(&dir.path, &["--out", "locale.ini"]);
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("one of the arguments --native --lock is required"),
        "{stderr}"
    );
}

#[test]
fn native_and_lock_are_mutually_exclusive() {
    let dir = TempDir::new("exclusive");
    let output = run(&dir.path, &["--native", "a", "--lock", "b", "--out", "c"]);
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("not allowed with argument --native"),
        "{stderr}"
    );
}

#[test]
fn out_is_required() {
    let dir = TempDir::new("out");
    let output = run(&dir.path, &["--native", "a"]);
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("the following arguments are required: --out"),
        "{stderr}"
    );
}

#[test]
fn stray_arguments_are_rejected() {
    let dir = TempDir::new("stray");
    let output = run(&dir.path, &["--native", "a", "--out", "b", "extra"]);
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("unrecognized arguments: extra"), "{stderr}");
}

#[test]
fn unknown_flags_are_rejected() {
    let dir = TempDir::new("unknown");
    let output = run(&dir.path, &["--native", "a", "--out", "b", "--bogus"]);
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("unrecognized arguments: --bogus"),
        "{stderr}"
    );
}

#[test]
fn help_exits_zero() {
    let dir = TempDir::new("help");
    for flag in ["--help", "-h"] {
        let output = run(&dir.path, &[flag]);
        assert_eq!(status_of(&output), 0);
        let stdout = String::from_utf8_lossy(&output.stdout);
        assert!(stdout.contains("--native FILE"), "{stdout}");
        assert!(stdout.contains("--lock FILE"), "{stdout}");
    }
}
