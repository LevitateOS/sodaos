use std::fs;
use std::os::unix::fs::PermissionsExt;

use super::support::{cmd, vm_fixture, write_fake, TempDir};

#[test]
fn ssh_exec_failure_shapes_match_bash() {
    let root = TempDir::new("sshexec");
    vm_fixture(&root.path);
    // Missing from PATH: 127 with the exec "not found" shape.
    let empty = TempDir::new("sshexec-bin");
    let out = cmd(&root.path)
        .arg("ssh")
        .env("PATH", &empty.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(127));
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "test-vm: exec: ssh: not found\n"
    );
    // tail fails the same way; the tunnels print first, then fail.
    let out = cmd(&root.path)
        .arg("console")
        .env("PATH", &empty.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(127));
    let out = cmd(&root.path)
        .arg("tunnel")
        .env("PATH", &empty.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(127));
    assert_eq!(
        String::from_utf8_lossy(&out.stdout),
        "Keep this running: Forgejo http://localhost:23000; Cockpit https://localhost:29090\n"
    );
    // Present but non-executable: 126 with the two-line shape.
    let blocked = empty.path.join("ssh");
    fs::write(&blocked, b"noexec").unwrap();
    fs::set_permissions(&blocked, fs::Permissions::from_mode(0o644)).unwrap();
    let out = cmd(&root.path)
        .arg("ssh")
        .env("PATH", &empty.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(126));
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "test-vm: ssh: Permission denied\ntest-vm: exec: ssh: cannot execute: Permission denied\n"
    );
    // A directory on PATH is skipped like a miss: 127 "not found".
    fs::remove_file(&blocked).unwrap();
    fs::create_dir(&blocked).unwrap();
    let out = cmd(&root.path)
        .arg("ssh")
        .env("PATH", &empty.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(127));
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "test-vm: exec: ssh: not found\n"
    );
}

#[test]
fn ssh_and_tunnels_exec_with_exact_argv() {
    let root = TempDir::new("sshargv");
    let vm = vm_fixture(&root.path);
    let fakes = TempDir::new("sshargv-bin");
    write_fake(&fakes.path, "ssh", "printf '%s\\n' \"$@\"");
    write_fake(&fakes.path, "tail", "printf '%s\\n' \"$@\"");
    let vm_str = vm.to_string_lossy().into_owned();
    let base = vec![
        "-p".to_string(),
        "22220".to_string(),
        "-i".to_string(),
        format!("{vm_str}/operator"),
        "-o".to_string(),
        "IdentitiesOnly=yes".to_string(),
        "-o".to_string(),
        "StrictHostKeyChecking=yes".to_string(),
        "-o".to_string(),
        format!("UserKnownHostsFile={vm_str}/known_hosts"),
    ];
    // Extra ssh arguments pass through after the host, like "$@" post-shift.
    let out = cmd(&root.path)
        .args(["ssh", "echo", "hi"])
        .env("PATH", &fakes.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(0));
    let mut want = base.clone();
    want.push("root@127.0.0.1".to_string());
    want.push("echo".to_string());
    want.push("hi".to_string());
    let got: Vec<String> = String::from_utf8_lossy(&out.stdout)
        .lines()
        .map(|line| line.to_string())
        .collect();
    assert_eq!(got, want);
    // Tunnels ignore extras and pin the loopback forwards.
    let out = cmd(&root.path)
        .args(["tunnel", "ignored"])
        .env("PATH", &fakes.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(0));
    let stdout = String::from_utf8_lossy(&out.stdout);
    let mut lines = stdout.lines();
    assert_eq!(
        lines.next().unwrap(),
        "Keep this running: Forgejo http://localhost:23000; Cockpit https://localhost:29090"
    );
    let mut want = base.clone();
    want.extend(
        [
            "-o",
            "ExitOnForwardFailure=yes",
            "-NT",
            "-L",
            "127.0.0.1:23000:127.0.0.1:3000",
            "-L",
            "127.0.0.1:29090:127.0.0.1:9090",
            "root@127.0.0.1",
        ]
        .into_iter()
        .map(|part| part.to_string()),
    );
    assert_eq!(lines.map(|line| line.to_string()).collect::<Vec<_>>(), want);
    let out = cmd(&root.path)
        .arg("web-tunnel")
        .env("PATH", &fakes.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(0));
    let stdout = String::from_utf8_lossy(&out.stdout);
    assert!(stdout.contains("Forgejo + Sodaspaces https://localhost:24444"));
    assert!(stdout.contains("127.0.0.1:24444:127.0.0.1:24444"));
    assert!(!stdout.contains("24443"));
    // console follows the last 80 lines of the guest console log.
    let out = cmd(&root.path)
        .args(["console", "ignored"])
        .env("PATH", &fakes.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(0));
    let got: Vec<String> = String::from_utf8_lossy(&out.stdout)
        .lines()
        .map(|line| line.to_string())
        .collect();
    assert_eq!(
        got,
        vec![
            "-n".to_string(),
            "80".to_string(),
            "-f".to_string(),
            format!("{vm_str}/console.log"),
        ]
    );
}
