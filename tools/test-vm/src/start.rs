use std::fs;
use std::os::unix::io::AsRawFd;

use super::process::{access, is_file, os_error, run};
use super::state::{uname_is, vm_running};
use super::{
    env_or, flock, refuse, umask, Exit, FAIL_PREFIX, LOCK_EX, LOCK_NB, QEMU_DEFAULT, R_OK, W_OK,
    X_OK,
};

pub(super) fn action_start(vm: &str) -> Result<(), Exit> {
    let qemu = env_or("QEMU", QEMU_DEFAULT);
    if !(uname_is("-s", "Linux")
        && uname_is("-m", "x86_64")
        && access("/dev/kvm", R_OK)
        && access("/dev/kvm", W_OK))
    {
        return refuse("Native x86_64 Linux with KVM access required");
    }
    // QEMU selects the executed hypervisor, so refuse anything that is not
    // an absolute executable file instead of execing the override blindly.
    if !(qemu.starts_with('/') && is_file(&qemu) && access(&qemu, X_OK)) {
        return refuse("QEMU must be an absolute executable path (default /usr/libexec/qemu-kvm)");
    }
    for file in ["disk.qcow2", "soda.ign", "operator", "known_hosts"] {
        if !is_file(&format!("{vm}/{file}")) {
            return refuse(format!(
                "Missing {vm}/{file}; see docs/guides/local-testing.md"
            ));
        }
    }
    unsafe {
        umask(0o077);
    }
    // `exec 9>start.lock` truncates and creates at 0666&~umask; the lock is
    // held across the spawn and released at exit.
    let lock_path = format!("{vm}/start.lock");
    let lock = match fs::OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .open(&lock_path)
    {
        Ok(file) => file,
        Err(err) => {
            eprintln!("{FAIL_PREFIX}: {lock_path}: {}", os_error(&err));
            return Err(Exit::Propagate(1));
        }
    };
    if unsafe { flock(lock.as_raw_fd(), LOCK_EX | LOCK_NB) } != 0 {
        return refuse("Another VM start is in progress");
    }
    let pid_path = format!("{vm}/qemu.pid");
    if vm_running(&pid_path)? {
        println!("Test VM is already running");
        return Ok(());
    }
    let drive = format!("if=virtio,format=qcow2,file={vm}/disk.qcow2");
    let fw_cfg = format!("name=opt/com.coreos/config,file={vm}/soda.ign");
    let serial = format!("file:{vm}/console.log");
    // Rust marks its own fds close-on-exec, which is the `9>&-` that keeps
    // the lock out of the QEMU child.
    run(
        &qemu,
        &[
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
            &drive,
            "-fw_cfg",
            &fw_cfg,
            "-nic",
            "user,model=virtio-net-pci,hostfwd=tcp:127.0.0.1:22220-:22",
            "-display",
            "none",
            "-serial",
            &serial,
            "-monitor",
            "none",
            "-daemonize",
            "-pidfile",
            &pid_path,
        ],
    )?;
    println!("Started soda-test (4 vCPU, 8 GiB RAM). SSH: cargo run -p soda-test-vm -- ssh");
    Ok(())
}
