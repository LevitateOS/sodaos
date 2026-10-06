use std::fs;

use super::support::{cmd, vm_fixture, TempDir, USAGE};

#[test]
fn unknown_action_prints_usage_and_exits_two() {
    let root = TempDir::new("usage");
    let out = cmd(&root.path).arg("bogus").output().unwrap();
    assert_eq!(out.status.code(), Some(2));
    assert!(out.stdout.is_empty());
    assert_eq!(String::from_utf8_lossy(&out.stderr), USAGE);
}

#[test]
fn empty_action_defaults_to_status() {
    let root = TempDir::new("empty");
    let out = cmd(&root.path).arg("").output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert_eq!(
        String::from_utf8_lossy(&out.stdout),
        "soda-test is not running\n"
    );
}

#[test]
fn status_reports_not_running_without_pidfile() {
    let root = TempDir::new("nostatus");
    let vm = vm_fixture(&root.path);
    assert!(!vm.join("qemu.pid").exists());
    for args in [vec![], vec!["status".to_string()]] {
        let out = cmd(&root.path).args(&args).output().unwrap();
        assert_eq!(out.status.code(), Some(1), "{args:?}");
        assert_eq!(
            String::from_utf8_lossy(&out.stdout),
            "soda-test is not running\n",
            "{args:?}"
        );
        assert!(out.stderr.is_empty(), "{args:?}");
        assert!(
            !String::from_utf8_lossy(&out.stdout).contains("SYNTHETIC_"),
            "{args:?}"
        );
    }
}

#[test]
fn status_rejects_unparseable_pidfiles_silently() {
    let root = TempDir::new("badpid");
    let vm = vm_fixture(&root.path);
    for (tag, bytes) in [
        ("empty", b"".as_slice()),
        ("newlines", b"\n\n"),
        ("garbage", b"abc\n"),
        ("interior-space", b"12 3\n"),
        ("negative", b"-5\n"),
        ("huge", b"2147483647\n"),
    ] {
        fs::write(vm.join("qemu.pid"), bytes).unwrap();
        let out = cmd(&root.path).arg("status").output().unwrap();
        assert_eq!(out.status.code(), Some(1), "{tag}");
        assert_eq!(
            String::from_utf8_lossy(&out.stdout),
            "soda-test is not running\n",
            "{tag}"
        );
        assert!(out.stderr.is_empty(), "{tag}");
    }
}

#[test]
fn status_reports_live_pid_and_keeps_spacing() {
    let root = TempDir::new("livepid");
    let vm = vm_fixture(&root.path);
    let mine = std::process::id().to_string();
    fs::write(vm.join("qemu.pid"), format!("{mine}\n")).unwrap();
    let out = cmd(&root.path).arg("status").output().unwrap();
    assert_eq!(out.status.code(), Some(0));
    assert_eq!(
        String::from_utf8_lossy(&out.stdout),
        format!("soda-test is running (PID {mine}); SSH at 127.0.0.1:22220\n")
    );
    fs::write(vm.join("qemu.pid"), format!("  {mine}  \n")).unwrap();
    let out = cmd(&root.path).arg("status").output().unwrap();
    assert_eq!(out.status.code(), Some(0));
    assert_eq!(
        String::from_utf8_lossy(&out.stdout),
        format!("soda-test is running (PID   {mine}  ); SSH at 127.0.0.1:22220\n")
    );
}
