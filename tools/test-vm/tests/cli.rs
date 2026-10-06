// Integration tests for soda-test-vm. Every case stays hermetic: fixture
// working directories under the temp dir, fake QEMU/ssh/tail on PATH, and
// no real VM is ever started (the fake hypervisor only logs argv and exits).
// Tests past the KVM gate skip cleanly where /dev/kvm is not accessible.

use std::fs;

#[path = "cli/start.rs"]
mod start;
#[path = "cli/status.rs"]
mod status;
mod support;
#[path = "cli/transport.rs"]
mod transport;

use self::support::{cmd, vm_fixture, TempDir};

extern "C" {
    fn faccessat(dirfd: i32, path: *const i8, mode: i32, flags: i32) -> i32;
    fn flock(fd: i32, op: i32) -> i32;
}

#[test]
fn stale_pwd_falls_back_to_working_directory() {
    // A $PWD that does not name the cwd (callers that fix the cwd only)
    // is ignored; an empty one too. The live pidfile proves resolution
    // used the fixture, not the stale value.
    for pwd in ["/nonexistent-stale-soda-vm", ""] {
        let root = TempDir::new("stalepwd");
        let vm = vm_fixture(&root.path);
        fs::write(vm.join("qemu.pid"), format!("{}\n", std::process::id())).unwrap();
        let mut command = cmd(&root.path);
        command.env("PWD", pwd);
        let out = command.arg("status").output().unwrap();
        assert_eq!(out.status.code(), Some(0), "PWD={pwd:?}");
        assert!(
            String::from_utf8_lossy(&out.stdout).contains("is running"),
            "PWD={pwd:?}"
        );
    }
}

#[test]
fn closed_stdout_dies_by_sigpipe_like_shell() {
    use std::os::fd::FromRawFd;
    use std::os::unix::process::ExitStatusExt;
    use std::process::Stdio;

    extern "C" {
        fn pipe(fds: *mut i32) -> i32;
        fn close(fd: i32) -> i32;
    }

    let root = TempDir::new("sigpipe");
    let vm = vm_fixture(&root.path);
    fs::write(vm.join("qemu.pid"), format!("{}\n", std::process::id())).unwrap();
    let mut fds = [0i32; 2];
    assert_eq!(unsafe { pipe(fds.as_mut_ptr()) }, 0);
    // Close the read end before spawn: the status line must raise SIGPIPE,
    // which the binary leaves at SIG_DFL like the shell.
    assert_eq!(unsafe { close(fds[0]) }, 0);
    let stdout = unsafe { Stdio::from_raw_fd(fds[1]) };
    let status = cmd(&root.path)
        .arg("status")
        .stdout(stdout)
        .stderr(Stdio::null())
        .status()
        .unwrap();
    assert_eq!(
        status.signal(),
        Some(13),
        "expected death by SIGPIPE, got {status:?}"
    );
}
