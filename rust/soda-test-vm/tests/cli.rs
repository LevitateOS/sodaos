// Integration tests for soda-test-vm. Every case stays hermetic: fixture
// working directories under the temp dir, fake QEMU/ssh/tail on PATH, and
// no real VM is ever started (the fake hypervisor only logs argv and exits).
// Tests past the KVM gate skip cleanly where /dev/kvm is not accessible.

use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::atomic::{AtomicU64, Ordering};

static COUNTER: AtomicU64 = AtomicU64::new(0);

extern "C" {
    fn faccessat(dirfd: i32, path: *const i8, mode: i32, flags: i32) -> i32;
    fn flock(fd: i32, op: i32) -> i32;
}

fn kvm_accessible() -> bool {
    use std::ffi::CString;
    use std::os::unix::ffi::OsStrExt;
    let path = CString::new(Path::new("/dev/kvm").as_os_str().as_bytes()).unwrap();
    unsafe {
        faccessat(-100, path.as_ptr(), 4, 0) == 0 && faccessat(-100, path.as_ptr(), 2, 0) == 0
    }
}

struct TempDir {
    path: PathBuf,
}

impl TempDir {
    fn new(tag: &str) -> TempDir {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path = env::temp_dir().join(format!("soda-vm-it-{tag}-{}-{id}", std::process::id()));
        let _ = fs::remove_dir_all(&path);
        fs::create_dir_all(&path).unwrap();
        TempDir { path }
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.path);
    }
}

fn bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-test-vm"))
}

/// `cmd` builds a hermetic invocation: fixture working directory with a
/// matching logical `$PWD`, and the QEMU override removed.
fn cmd(cwd: &Path) -> Command {
    let mut command = Command::new(bin());
    command.current_dir(cwd);
    command.env("PWD", cwd);
    command.env_remove("QEMU");
    command
}

fn write_fake(dir: &Path, name: &str, body: &str) {
    let path = dir.join(name);
    fs::write(&path, format!("#!/bin/sh\n{body}\n")).unwrap();
    fs::set_permissions(&path, fs::Permissions::from_mode(0o755)).unwrap();
}

/// `vm_fixture` stages `$root/.artifacts/test-vm` with the four files
/// `start` requires, holding synthetic (never real) key material.
fn vm_fixture(root: &Path) -> PathBuf {
    let vm = root.join(".artifacts/test-vm");
    fs::create_dir_all(&vm).unwrap();
    fs::write(vm.join("disk.qcow2"), b"fake-disk").unwrap();
    fs::write(vm.join("soda.ign"), b"{}").unwrap();
    fs::write(vm.join("operator"), b"SYNTHETIC_OPERATOR_KEY\n").unwrap();
    fs::write(vm.join("known_hosts"), b"SYNTHETIC_KNOWN_HOSTS\n").unwrap();
    vm
}

const USAGE: &str = "usage: cargo run -p soda-test-vm -- [start|status|ssh [COMMAND...]|tunnel|web-tunnel|console]\n";

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

#[test]
fn start_refuses_without_kvm_or_with_bad_qemu() {
    let root = TempDir::new("kvmgate");
    if !kvm_accessible() {
        let out = cmd(&root.path).arg("start").output().unwrap();
        assert_eq!(out.status.code(), Some(1));
        assert!(out.stdout.is_empty());
        assert_eq!(
            String::from_utf8_lossy(&out.stderr),
            "Native x86_64 Linux with KVM access required\n"
        );
        return;
    }
    // Past the KVM gate here, so every QEMU shape is refused in turn.
    let empty = TempDir::new("kvmgate-bin");
    for (tag, qemu) in [
        ("relative", "qemu-kvm"),
        ("missing", "/nonexistent-qemu-test-vm"),
        ("directory", "/tmp"),
    ] {
        let out = cmd(&root.path)
            .arg("start")
            .env("QEMU", qemu)
            .output()
            .unwrap();
        assert_eq!(out.status.code(), Some(1), "{tag}");
        assert_eq!(
            String::from_utf8_lossy(&out.stderr),
            "QEMU must be an absolute executable path (default /usr/libexec/qemu-kvm)\n",
            "{tag}"
        );
    }
    // A present but non-executable file is refused the same way.
    let blocked = empty.path.join("blocked-qemu");
    fs::write(&blocked, b"noexec").unwrap();
    fs::set_permissions(&blocked, fs::Permissions::from_mode(0o644)).unwrap();
    let out = cmd(&root.path)
        .arg("start")
        .env("QEMU", &blocked)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "QEMU must be an absolute executable path (default /usr/libexec/qemu-kvm)\n"
    );
}

#[test]
fn start_reports_missing_files_in_order() {
    if !kvm_accessible() {
        eprintln!("skipping: /dev/kvm not accessible");
        return;
    }
    let root = TempDir::new("missingfiles");
    let fakes = TempDir::new("missingfiles-bin");
    write_fake(&fakes.path, "qemu", "exit 0");
    let qemu = fakes.path.join("qemu");
    let vm = root.path.join(".artifacts/test-vm");
    fs::create_dir_all(&vm).unwrap();
    for file in ["disk.qcow2", "soda.ign", "operator", "known_hosts"] {
        let out = cmd(&root.path)
            .arg("start")
            .env("QEMU", &qemu)
            .output()
            .unwrap();
        assert_eq!(out.status.code(), Some(1), "{file}");
        assert_eq!(
            String::from_utf8_lossy(&out.stderr),
            format!(
                "Missing {}/.artifacts/test-vm/{file}; see docs/guides/local-testing.md\n",
                root.path.display()
            ),
            "{file}"
        );
        fs::write(vm.join(file), b"staged").unwrap();
    }
}

/// The holder keeps the lock in this process: no subprocess can be
/// orphaned with the file description still open.
struct HeldLock {
    file: fs::File,
}

impl HeldLock {
    fn new(path: &Path) -> HeldLock {
        let file = fs::OpenOptions::new()
            .write(true)
            .create(true)
            .truncate(false)
            .open(path)
            .unwrap();
        use std::os::unix::io::AsRawFd;
        assert_eq!(unsafe { flock(file.as_raw_fd(), 2 | 4) }, 0);
        HeldLock { file }
    }
}

impl Drop for HeldLock {
    fn drop(&mut self) {
        use std::os::unix::io::AsRawFd;
        unsafe {
            flock(self.file.as_raw_fd(), 8);
        }
    }
}

#[test]
fn start_refuses_held_lock_and_running_vm() {
    if !kvm_accessible() {
        eprintln!("skipping: /dev/kvm not accessible");
        return;
    }
    let root = TempDir::new("locked");
    let vm = vm_fixture(&root.path);
    let fakes = TempDir::new("locked-bin");
    write_fake(&fakes.path, "qemu", "exit 0");
    let qemu = fakes.path.join("qemu");
    let _held = HeldLock::new(&vm.join("start.lock"));
    let out = cmd(&root.path)
        .arg("start")
        .env("QEMU", &qemu)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "Another VM start is in progress\n"
    );
    drop(_held);
    fs::write(vm.join("qemu.pid"), format!("{}\n", std::process::id())).unwrap();
    let out = cmd(&root.path)
        .arg("start")
        .env("QEMU", &qemu)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(0));
    assert_eq!(
        String::from_utf8_lossy(&out.stdout),
        "Test VM is already running\n"
    );
}

/// Fake QEMU logs its argv, writes the requested pidfile, and exits as told.
/// No VM is ever started: the binary under test cannot tell the difference.
fn write_fake_qemu(dir: &Path, exit: &str) -> PathBuf {
    write_fake(
        dir,
        "qemu",
        &format!(
            "printf '%s\\n' \"$@\" >\"$QEMU_ARGV_LOG\"\nprev=\"\"\nfor a in \"$@\"; do if [ \"$prev\" = \"-pidfile\" ]; then echo \"$$\" >\"$a\"; fi; prev=\"$a\"; done\nexit {exit}"
        ),
    );
    dir.join("qemu")
}

#[test]
fn start_runs_qemu_with_exact_argv_and_releases_lock() {
    if !kvm_accessible() {
        eprintln!("skipping: /dev/kvm not accessible");
        return;
    }
    let root = TempDir::new("startok");
    let vm = vm_fixture(&root.path);
    let fakes = TempDir::new("startok-bin");
    let qemu = write_fake_qemu(&fakes.path, "0");
    let argv_log = fakes.path.join("argv.log");
    let out = cmd(&root.path)
        .arg("start")
        .env("QEMU", &qemu)
        .env("QEMU_ARGV_LOG", &argv_log)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(0));
    assert_eq!(
        String::from_utf8_lossy(&out.stdout),
        "Started soda-test (4 vCPU, 8 GiB RAM). SSH: cargo run -p soda-test-vm -- ssh\n"
    );
    assert!(out.stderr.is_empty());
    let vm_str = vm.to_string_lossy().into_owned();
    let logged = fs::read_to_string(&argv_log).unwrap();
    let logged: Vec<String> = logged.lines().map(|line| line.to_string()).collect();
    let expected: Vec<String> = [
        "-name",
        "soda-test",
        "-machine",
        "q35,accel=kvm",
        "-cpu",
        "host",
        "-smp",
        "4",
        "-m",
        "8192",
        "-drive",
        &format!("if=virtio,format=qcow2,file={vm_str}/disk.qcow2"),
        "-fw_cfg",
        &format!("name=opt/com.coreos/config,file={vm_str}/soda.ign"),
        "-nic",
        "user,model=virtio-net-pci,hostfwd=tcp:127.0.0.1:22220-:22",
        "-display",
        "none",
        "-serial",
        &format!("file:{vm_str}/console.log"),
        "-monitor",
        "none",
        "-daemonize",
        "-pidfile",
        &format!("{vm_str}/qemu.pid"),
    ]
    .into_iter()
    .map(|part| part.to_string())
    .collect();
    assert_eq!(logged, expected);
    // The lock is created 0600 under the start umask and released at exit.
    let mode = fs::metadata(vm.join("start.lock"))
        .unwrap()
        .permissions()
        .mode()
        & 0o777;
    assert_eq!(mode, 0o600);
    let relock = fs::OpenOptions::new()
        .write(true)
        .open(vm.join("start.lock"))
        .unwrap();
    use std::os::unix::io::AsRawFd;
    assert_eq!(unsafe { flock(relock.as_raw_fd(), 2 | 4) }, 0);
    unsafe {
        flock(relock.as_raw_fd(), 8);
    }
}

#[test]
fn start_propagates_qemu_status_silently() {
    if !kvm_accessible() {
        eprintln!("skipping: /dev/kvm not accessible");
        return;
    }
    let root = TempDir::new("code-3");
    let vm = vm_fixture(&root.path);
    let fakes = TempDir::new("qfail-bin");
    write_fake(&fakes.path, "qemu", "exit 3");
    let out = cmd(&root.path)
        .arg("start")
        .env("QEMU", fakes.path.join("qemu"))
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(3));
    assert!(out.stdout.is_empty());
    assert!(out.stderr.is_empty());
    assert!(!vm.join("qemu.pid").exists());
}

#[test]
fn start_reports_qemu_killed_by_signal() {
    if !kvm_accessible() {
        eprintln!("skipping: /dev/kvm not accessible");
        return;
    }
    // TERM prints the bare notice; the fake reports its pid so the SEGV
    // line (which carries the pid) is asserted byte-exact.
    let root = TempDir::new("qterm");
    vm_fixture(&root.path);
    let fakes = TempDir::new("qterm-bin");
    write_fake(&fakes.path, "qemu", "kill -TERM $$");
    let out = cmd(&root.path)
        .arg("start")
        .env("QEMU", fakes.path.join("qemu"))
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(143));
    assert!(out.stdout.is_empty());
    assert_eq!(String::from_utf8_lossy(&out.stderr), "Terminated\n");

    let root = TempDir::new("qsegv");
    vm_fixture(&root.path);
    let fakes = TempDir::new("qsegv-bin");
    let pid_file = fakes.path.join("qemu.pid.log");
    write_fake(
        &fakes.path,
        "qemu",
        "echo $$ >\"$QEMU_PID_LOG\"\nkill -SEGV $$",
    );
    let out = cmd(&root.path)
        .arg("start")
        .env("QEMU", fakes.path.join("qemu"))
        .env("QEMU_PID_LOG", &pid_file)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(139));
    assert!(out.stdout.is_empty());
    let fake_pid = fs::read_to_string(&pid_file).unwrap();
    let fake_pid = fake_pid.trim();
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        format!(
            "test-vm: {fake_pid} Segmentation fault      (core dumped) \"$qemu\" -name soda-test -machine q35,accel=kvm -cpu host -smp 4 -m 8192 -drive \"if=virtio,format=qcow2,file=$vm/disk.qcow2\" -fw_cfg \"name=opt/com.coreos/config,file=$vm/soda.ign\" -nic user,model=virtio-net-pci,hostfwd=tcp:127.0.0.1:22220-:22 -display none -serial \"file:$vm/console.log\" -monitor none -daemonize -pidfile \"$vm/qemu.pid\" 9>&-\n"
        )
    );

    // INT and PIPE deaths stay silent.
    for (tag, sig, code) in [("int", "INT", 130), ("pipe", "PIPE", 141)] {
        let root = TempDir::new(tag);
        vm_fixture(&root.path);
        let fakes = TempDir::new("qsigquiet-bin");
        write_fake(&fakes.path, "qemu", &format!("kill -{sig} $$"));
        let out = cmd(&root.path)
            .arg("start")
            .env("QEMU", fakes.path.join("qemu"))
            .output()
            .unwrap();
        assert_eq!(out.status.code(), Some(code), "{tag}");
        assert!(out.stdout.is_empty(), "{tag}");
        assert!(out.stderr.is_empty(), "{tag}");
    }
}

#[test]
fn unreadable_pidfile_reports_and_aborts() {
    let root = TempDir::new("unreadable");
    let vm = vm_fixture(&root.path);
    let pid_path = vm.join("qemu.pid");
    fs::write(&pid_path, format!("{}\n", std::process::id())).unwrap();
    fs::set_permissions(&pid_path, fs::Permissions::from_mode(0o000)).unwrap();
    if fs::read(&pid_path).is_ok() {
        eprintln!("skipping: pidfile still readable (root?)");
        return;
    }
    // The substitution failure aborts under `set -e`: diagnostic only, the
    // not-running line never prints.
    let out = cmd(&root.path).arg("status").output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        format!(
            "test-vm: {}/.artifacts/test-vm/qemu.pid: Permission denied\n",
            root.path.display()
        )
    );
}

#[test]
fn unreadable_pidfile_aborts_start_before_qemu() {
    if !kvm_accessible() {
        eprintln!("skipping: /dev/kvm not accessible");
        return;
    }
    let root = TempDir::new("unreadablestart");
    let vm = vm_fixture(&root.path);
    let pid_path = vm.join("qemu.pid");
    fs::write(&pid_path, format!("{}\n", std::process::id())).unwrap();
    fs::set_permissions(&pid_path, fs::Permissions::from_mode(0o000)).unwrap();
    if fs::read(&pid_path).is_ok() {
        eprintln!("skipping: pidfile still readable (root?)");
        return;
    }
    let fakes = TempDir::new("unreadablestart-bin");
    let argv_log = fakes.path.join("argv.log");
    let qemu = write_fake_qemu(&fakes.path, "0");
    let out = cmd(&root.path)
        .arg("start")
        .env("QEMU", &qemu)
        .env("QEMU_ARGV_LOG", &argv_log)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        format!(
            "test-vm: {}/.artifacts/test-vm/qemu.pid: Permission denied\n",
            root.path.display()
        )
    );
    assert!(!argv_log.exists());
}

#[test]
fn missing_uname_reports_and_refuses() {
    // uname is the first gate: a PATH without it prints the diagnostic,
    // then the KVM refusal. No KVM access needed to reach this.
    let root = TempDir::new("nouname");
    vm_fixture(&root.path);
    let empty = TempDir::new("nouname-bin");
    let out = cmd(&root.path)
        .arg("start")
        .env("PATH", &empty.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "test-vm: uname: command not found\nNative x86_64 Linux with KVM access required\n"
    );
}

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
