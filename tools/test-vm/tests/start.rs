use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};

use super::flock;
use super::support::{cmd, kvm_accessible, vm_fixture, write_fake, TempDir};

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
        ("directory", root.path.to_str().unwrap()),
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
    // Child signals map to shell-compatible statuses without Bash's
    // human-readable signal notices.
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
    assert!(out.stderr.is_empty());

    let root = TempDir::new("qsegv");
    vm_fixture(&root.path);
    let fakes = TempDir::new("qsegv-bin");
    write_fake(&fakes.path, "qemu", "kill -SEGV $$");
    let out = cmd(&root.path)
        .arg("start")
        .env("QEMU", fakes.path.join("qemu"))
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(139));
    assert!(out.stdout.is_empty());
    assert!(out.stderr.is_empty());

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
            "test-vm: {}/.artifacts/test-vm/qemu.pid: Permission denied (os error 13)\n",
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
            "test-vm: {}/.artifacts/test-vm/qemu.pid: Permission denied (os error 13)\n",
            root.path.display()
        )
    );
    assert!(!argv_log.exists());
}

#[test]
fn missing_uname_keeps_native_error_and_refuses() {
    // uname is the first gate: a PATH without it emits the native spawn
    // error before the platform refusal. No KVM access is needed here.
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
    let stderr = String::from_utf8_lossy(&out.stderr);
    assert!(stderr.starts_with("test-vm: uname: "), "{stderr:?}");
    assert!(stderr.contains("No such file or directory"), "{stderr:?}");
    assert!(
        stderr.ends_with("Native x86_64 Linux with KVM access required\n"),
        "{stderr:?}"
    );
}
