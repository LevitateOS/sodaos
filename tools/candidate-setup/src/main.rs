// soda-candidate-setup prepares this machine so `sudo soda-candidate`
// runs without a coding agent: wrapper on sudo's PATH, admitted controller,
// worker directories, restricted worker config, and a fixture-only media
// authority. Development and fixture scope only: it never creates
// qualification or signing configs, and the generated keys must never
// stand in for release keys.
//
// Rust port of scripts/setup-soda-candidate.sh (plus the
// scripts/candidate-storage.sh defaults it sourced). Messages, exit codes,
// installed paths, file modes, and generated file bytes match the shell.
// Like the script, this binary takes no arguments and ignores any it is
// given. Run from the repository root.

use std::env;
use std::ffi::CString;
use std::fs;
use std::io::{self, Write};
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::OpenOptionsExt;
use std::path::{Path, PathBuf};

mod config;
mod controller;
mod fixture_authority;
mod preflight;
mod process;
mod selinux;
mod storage;
mod worker_caches;
mod worker_tools;

use self::process::{id_un, run};

const FAIL_PREFIX: &str = "setup-soda-candidate";
const PREFIX_DEFAULT: &str = "ghcr.io/levitateos/sodaos";
const STORAGE_ROOT_DEFAULT: &str = "/home/soda-candidate";
const ROOTFS_DIR: &str = "/home/soda-rootfs";
const ADMITTED: &str = "/usr/local/lib/soda/soda-build";
const PINNED_GO: &str = "/usr/local/lib/soda/pinned-go";
const WRAPPER: &str = "/usr/sbin/soda-candidate";
const TOOLS: &str = "/var/lib/soda-candidate-tools";
const WORKER_POLICY_SRC: &str = "system/host/selinux/soda-build-worker.te";
const AUTHORITY: &str = "/var/lib/soda-candidate-authority";
const LEGACY_HOME: &str = "/var/lib/soda-candidate-home";
const LEGACY_RUN: &str = "/var/lib/soda-candidate-run";
const WORKER_USER: &str = "soda-build-worker";

extern "C" {
    fn flock(fd: i32, op: i32) -> i32;
    fn faccessat(dirfd: i32, path: *const i8, mode: i32, flags: i32) -> i32;
    fn umask(mask: u32) -> u32;
    fn sigaction(signum: i32, act: *const Sigaction, oldact: *mut Sigaction) -> i32;
}

/// Matches glibc's `struct sigaction` on Linux (handler, signal mask, flags,
/// restorer). The glibc wrapper fills in the restorer itself.
#[repr(C)]
#[derive(Clone, Copy)]
struct Sigaction {
    handler: usize,
    mask: [u64; 16],
    flags: i32,
    restorer: usize,
}

const SIGPIPE: i32 = 13;
const SIG_DFL: usize = 0;
const LOCK_EX: i32 = 2;
const LOCK_NB: i32 = 4;
const AT_FDCWD: i32 = -100;
const X_OK: i32 = 1;

/// How the process ends: a `fail()` message with exit 1, or a propagated
/// child status with no extra output (the script's `set -e` behavior).
#[derive(Debug)]
enum Exit {
    Fail(String),
    Propagate(i32),
}

fn fail<T>(msg: impl Into<String>) -> Result<T, Exit> {
    Err(Exit::Fail(msg.into()))
}

/// `env_or` mirrors `${VAR:-default}`: unset or empty falls back.
fn env_or(key: &str, default: &str) -> String {
    match env::var(key) {
        Ok(v) if !v.is_empty() => v,
        _ => default.to_string(),
    }
}

/// `current_pwd` mirrors `$PWD`: the inherited logical path when present,
// otherwise the physical working directory.
fn current_pwd() -> String {
    match env::var("PWD") {
        Ok(v) => v,
        Err(_) => env::current_dir()
            .map(|p| p.to_string_lossy().into_owned())
            .unwrap_or_default(),
    }
}

fn is_dir(path: &str) -> bool {
    fs::metadata(path).map(|m| m.is_dir()).unwrap_or(false)
}

fn is_file(path: &str) -> bool {
    fs::metadata(path).map(|m| m.is_file()).unwrap_or(false)
}

/// `stripped` mirrors command substitution: all trailing newlines removed.
fn stripped(out: &[u8]) -> &[u8] {
    let mut end = out.len();
    while end > 0 && out[end - 1] == b'\n' {
        end -= 1;
    }
    &out[..end]
}

fn stripped_string(out: &[u8]) -> String {
    String::from_utf8_lossy(stripped(out)).into_owned()
}

fn current_umask() -> u32 {
    unsafe {
        let mask = umask(0);
        umask(mask);
        mask
    }
}

/// `write_staged` mirrors shell redirection into staging: bytes exact, mode
/// 0666 filtered by the operator umask like `>` and `open("w")`.
fn write_staged(path: &Path, bytes: &[u8]) -> io::Result<()> {
    let mode = 0o666 & !current_umask();
    let mut opts = fs::OpenOptions::new();
    opts.write(true).create(true).truncate(true).mode(mode);
    opts.open(path)?.write_all(bytes)
}

fn is_executable(path: &Path) -> bool {
    match CString::new(path.as_os_str().as_bytes()) {
        Ok(c) => unsafe { faccessat(AT_FDCWD, c.as_ptr(), X_OK, 0) == 0 },
        Err(_) => false,
    }
}

/// `command_v` mirrors `command -v`: a PATH search with execute permission.
fn command_v(tool: &str) -> Option<PathBuf> {
    let raw = env::var_os("PATH").unwrap_or_else(|| "/bin:/usr/bin".into());
    command_v_in(tool, &raw)
}

fn command_v_in(tool: &str, path_env: &std::ffi::OsStr) -> Option<PathBuf> {
    if tool.contains('/') {
        let path = PathBuf::from(tool);
        return is_executable(&path).then_some(path);
    }
    for dir in env::split_paths(path_env) {
        let candidate = dir.join(tool);
        if is_executable(&candidate) {
            return Some(candidate);
        }
    }
    None
}

fn run_setup(cleanup: &mut Vec<PathBuf>) -> Result<(), Exit> {
    let pre = preflight::preflight()?;
    let preflight::Preflight {
        prefix,
        refresh,
        forgejo_source,
        storage,
        rootfs_url,
        pwd,
        output_parent,
        worker_json_path,
        pinned,
        pinned_goroot,
        want,
    } = pre;

    storage::prepare_scratch(&storage)?;

    let ctl = controller::admit_controller(&storage, cleanup)?;
    let controller::ControllerPaths { bindir } = ctl;

    let tools = worker_tools::provision_worker_tools(
        &output_parent,
        &storage,
        &pinned,
        &pinned_goroot,
        &want,
    )?;
    let worker_tools::WorkerTools { bun_final, owned } = tools;

    let caches = worker_caches::warm_worker_caches(&storage, &pinned, &owned, &bun_final)?;
    let worker_caches::WorkerCaches {
        go_mod,
        go_build_cache,
    } = caches;

    selinux::install_worker_selinux(&bindir, &storage, &go_mod, &go_build_cache, &owned)?;

    fixture_authority::admit_fixture_authority(
        &pwd,
        &forgejo_source,
        &output_parent,
        &storage,
        &worker_json_path,
        &prefix,
        &refresh,
        &owned,
        cleanup,
    )?;

    run("sudo", &["mkdir", "-p", ROOTFS_DIR])?;
    let rootfs_owner = String::from_utf8_lossy(&id_un()).into_owned();
    run("sudo", &["chown", &rootfs_owner, ROOTFS_DIR])?;
    if command_v("restorecon").is_some() {
        run("sudo", &["restorecon", ROOTFS_DIR])?;
    }

    println!("-- ready. Development candidate command from {pwd}:");
    println!("  sudo {ADMITTED} --worker-config {worker_json_path} --arch x86_64 \\");
    println!("    --out {output_parent}/manual-01 --development --target candidate \\");
    println!("    --forgejo-source {forgejo_source}");
    println!("-- installer rootfs pickup: {ROOTFS_DIR} (served by soda-rootfs-server.service)");
    println!("  after a media build, copy its hash-named rootfs here:");
    println!("  sudo cp <out>/artifacts/media/*-rootfs.img {ROOTFS_DIR}/ && sudo chown root:root {ROOTFS_DIR}/*-rootfs.img && sudo chmod 0644 {ROOTFS_DIR}/*-rootfs.img");
    if !rootfs_url.is_empty() {
        println!("-- guests fetch the filed rootfs from: {rootfs_url}");
    }
    Ok(())
}

fn main() {
    // Rust ignores SIGPIPE at startup (a write to a closed pipe would return
    // EPIPE and println! would panic instead of dying like the shell);
    // restore the default disposition for byte parity.
    unsafe {
        let restore_pipe = Sigaction {
            handler: SIG_DFL,
            mask: [0; 16],
            flags: 0,
            restorer: 0,
        };
        sigaction(SIGPIPE, &restore_pipe, std::ptr::null_mut());
    }
    let mut cleanup: Vec<PathBuf> = Vec::new();
    let result = run_setup(&mut cleanup);
    for path in &cleanup {
        let _ = fs::remove_dir_all(path);
    }
    match result {
        Ok(()) => {}
        Err(Exit::Fail(msg)) => {
            eprintln!("{FAIL_PREFIX}: {msg}");
            std::process::exit(1);
        }
        Err(Exit::Propagate(code)) => std::process::exit(code),
    }
}

#[cfg(test)]
mod tests;
