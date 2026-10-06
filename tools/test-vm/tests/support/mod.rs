use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::atomic::{AtomicU64, Ordering};

use super::faccessat;

static COUNTER: AtomicU64 = AtomicU64::new(0);

pub(super) fn kvm_accessible() -> bool {
    use std::ffi::CString;
    use std::os::unix::ffi::OsStrExt;
    let path = CString::new(Path::new("/dev/kvm").as_os_str().as_bytes()).unwrap();
    unsafe {
        faccessat(-100, path.as_ptr(), 4, 0) == 0 && faccessat(-100, path.as_ptr(), 2, 0) == 0
    }
}

pub(super) struct TempDir {
    pub(super) path: PathBuf,
}

impl TempDir {
    pub(super) fn new(tag: &str) -> TempDir {
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

pub(super) fn bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-test-vm"))
}

/// `cmd` builds a hermetic invocation: fixture working directory with a
/// matching logical `$PWD`, and the QEMU override removed.
pub(super) fn cmd(cwd: &Path) -> Command {
    let mut command = Command::new(bin());
    command.current_dir(cwd);
    command.env("PWD", cwd);
    command.env_remove("QEMU");
    command
}

pub(super) fn write_fake(dir: &Path, name: &str, body: &str) {
    let path = dir.join(name);
    fs::write(&path, format!("#!/bin/sh\n{body}\n")).unwrap();
    fs::set_permissions(&path, fs::Permissions::from_mode(0o755)).unwrap();
}

/// `vm_fixture` stages `$root/.artifacts/test-vm` with the four files
/// `start` requires, holding synthetic (never real) key material.
pub(super) fn vm_fixture(root: &Path) -> PathBuf {
    let vm = root.join(".artifacts/test-vm");
    fs::create_dir_all(&vm).unwrap();
    fs::write(vm.join("disk.qcow2"), b"fake-disk").unwrap();
    fs::write(vm.join("soda.ign"), b"{}").unwrap();
    fs::write(vm.join("operator"), b"SYNTHETIC_OPERATOR_KEY\n").unwrap();
    fs::write(vm.join("known_hosts"), b"SYNTHETIC_KNOWN_HOSTS\n").unwrap();
    vm
}

pub(super) const USAGE: &str = "usage: cargo run -p soda-test-vm -- [start|status|ssh [COMMAND...]|tunnel|web-tunnel|console]\n";
