use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::atomic::{AtomicU64, Ordering};

static COUNTER: AtomicU64 = AtomicU64::new(0);

pub struct TempDir {
    pub path: PathBuf,
}

impl TempDir {
    pub fn new(tag: &str) -> TempDir {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path = env::temp_dir().join(format!("soda-setup-it-{tag}-{}-{id}", std::process::id()));
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
    PathBuf::from(env!("CARGO_BIN_EXE_soda-candidate-setup"))
}

/// `cmd` builds a hermetic invocation: the working directory is a fixture,
/// `$PWD` matches it (the binary reads `$PWD` like the script), ambient
/// `SODA_*` inputs are removed, and `SODA_FORGEJO_SOURCE` starts absent.
pub fn cmd(cwd: &Path) -> Command {
    let mut command = Command::new(bin());
    command.current_dir(cwd);
    command.env("PWD", cwd);
    for var in [
        "SODA_FORGEJO_SOURCE",
        "SODA_REPOSITORY_PREFIX",
        "SODA_REFRESH_AUTHORITY",
        "SODA_CANDIDATE_ROOT",
        "SODA_CANDIDATE_HOME",
        "SODA_CANDIDATE_RUN",
        "SODA_CANDIDATE_SCRATCH",
        "SODA_ROTATE_ACK",
        "SUDO_USER",
    ] {
        command.env_remove(var);
    }
    command
}

pub fn write_fake(dir: &Path, name: &str, body: &str) {
    let path = dir.join(name);
    fs::write(&path, format!("#!/bin/sh\n{body}\n")).unwrap();
    fs::set_permissions(&path, fs::Permissions::from_mode(0o755)).unwrap();
}

pub fn real_tool(name: &str) -> Option<PathBuf> {
    let raw = env::var_os("PATH")?;
    for dir in env::split_paths(&raw) {
        let candidate = dir.join(name);
        if let Ok(meta) = fs::metadata(&candidate) {
            if meta.is_file() && meta.permissions().mode() & 0o111 != 0 {
                return Some(candidate);
            }
        }
    }
    None
}

/// Every fake bin dir replaces `ip` with a noisy failure: the script
/// discards `ip` stderr (`2>/dev/null`), so the exact-stderr assertions in
/// every test below also prove the port suppresses it.
pub fn fake_ip(dir: &Path) {
    write_fake(dir, "ip", "echo ip-noise >&2; exit 1");
}

pub fn git_init(dir: &Path) -> bool {
    let git = match real_tool("git") {
        Some(path) => path,
        None => return false,
    };
    for args in [
        vec!["init", "-q"],
        vec!["-c", "user.name=t", "-c", "user.email=t@t", "add", "-A"],
        vec![
            "-c",
            "user.name=t",
            "-c",
            "user.email=t@t",
            "commit",
            "-qm",
            "init",
        ],
    ] {
        let status = Command::new(&git)
            .args(&args)
            .current_dir(dir)
            .env("GIT_CONFIG_NOSYSTEM", "1")
            .env("HOME", dir)
            .status();
        if !matches!(status, Ok(status) if status.success()) {
            return false;
        }
    }
    true
}

extern "C" {
    fn flock(fd: i32, op: i32) -> i32;
}

/// The holder keeps the lock in this process: no subprocess can be
/// orphaned with the file description still open.
pub struct HeldLock {
    _file: fs::File,
}

pub fn hold_flock_lock(path: &Path) -> Option<HeldLock> {
    let file = fs::OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .open(path)
        .ok()?;
    use std::os::unix::io::AsRawFd;
    if unsafe { flock(file.as_raw_fd(), 2 | 4) } != 0 {
        return None;
    }
    Some(HeldLock { _file: file })
}
