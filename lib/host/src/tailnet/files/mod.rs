//! Tailnet run files: rooted runtime state, locks, keys, resolver checks.
//! Lane C owns this file.
//!
//! Rust port of `internal/host/tailnet/files.go` plus `current`/`saveCurrent`
//! from `internal/host/tailnet/companion.go` (same Go type, owned by this lane).
//!
//! Containment note: Go uses `os.Root` (openat2-grade containment). This port
//! anchors every operation under an explicit base path and opens leaves with
//! `O_NOFOLLOW`, but joins are plain path joins, so containment under an
//! actively racing attacker is approximated, not proven. Every multi-component
//! relative path is built only from regex-validated IDs (no `.`, `..` or `/`
//! can occur), and tests run in private temp dirs with no attacker races.

use std::fs::{self, File, OpenOptions};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::os::unix::io::AsRawFd;
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};

use crate::tailnet_domain::{valid_project_id, ERR_INVALID, ERR_UNAVAILABLE};

/// Text returned when `current.json` is absent. The companion lane matches on
/// this exact string the way Go matches `errors.Is(err, os.ErrNotExist)`.
const RUNTIME_RECORD_NOT_FOUND: &str = "runtime record not found";

/// Path-anchored root for runtime state. All operations join `rel` under the
/// anchored base path; leaves are opened with `O_NOFOLLOW`.
#[derive(Debug)]
pub struct Root {
    path: PathBuf,
}

/// Locked, owned runtime project directory. The lock releases when the
/// contained `File` is dropped (close), exactly like Go's `f.Close()`.
#[derive(Debug)]
pub struct RunFiles {
    pub(crate) root: Root,
    // Held for close-on-drop lock semantics; never read.
    #[allow(dead_code)]
    lock: File,
    pub(crate) uid: u32,
    pub(crate) gid: u32,
}

impl Root {
    pub(crate) fn join(&self, rel: &str) -> PathBuf {
        self.path.join(rel)
    }

    /// Lstat equivalent: never follows the final component.
    pub(crate) fn lstat(&self, rel: &str) -> std::io::Result<fs::Metadata> {
        fs::symlink_metadata(self.join(rel))
    }

    /// Stat equivalent: follows links, like Go `Root.Stat`.
    fn stat(&self, rel: &str) -> std::io::Result<fs::Metadata> {
        fs::metadata(self.join(rel))
    }
}

fn root_directory(info: &fs::Metadata, uid: u32) -> bool {
    info.is_dir() && info.uid() == uid && info.mode() & 0o777 == 0o700
}

fn runtime_file(info: &fs::Metadata, uid: u32, gid: u32, mode: u32) -> bool {
    info.is_file()
        && info.mode() & 0o777 == mode
        && info.uid() == uid
        && info.gid() == gid
        && info.nlink() == 1
}

fn owned_run_dir(info: &fs::Metadata, uid: u32, gid: u32) -> bool {
    info.is_dir() && info.mode() & 0o777 == 0o700 && info.uid() == uid && info.gid() == gid
}

fn same_file(a: &fs::Metadata, b: &fs::Metadata) -> bool {
    a.dev() == b.dev() && a.ino() == b.ino()
}

/// `mkdir(path, 0o700)`: Go passes the mode to mkdir(2); Rust `create_dir`
/// would use `0o777 & ~umask`, so call libc directly for exact semantics
/// (including `EEXIST` when the path already exists).
fn mkdir_700(path: &Path) -> std::io::Result<()> {
    use std::os::unix::ffi::OsStrExt;
    let c = std::ffi::CString::new(path.as_os_str().as_bytes())
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "path contains nul"))?;
    if unsafe { libc::mkdir(c.as_ptr(), 0o700) } == 0 {
        Ok(())
    } else {
        Err(std::io::Error::last_os_error())
    }
}

fn chown_path(path: &Path, uid: u32, gid: u32) -> std::io::Result<()> {
    use std::os::unix::ffi::OsStrExt;
    let c = std::ffi::CString::new(path.as_os_str().as_bytes())
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "path contains nul"))?;
    if unsafe { libc::chown(c.as_ptr(), uid as libc::uid_t, gid as libc::gid_t) } == 0 {
        Ok(())
    } else {
        Err(std::io::Error::last_os_error())
    }
}

fn open_runtime_root(base: &str, uid: u32) -> Result<Root, String> {
    let info = fs::symlink_metadata(base).map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !root_directory(&info, uid) {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    let root = Root {
        path: PathBuf::from(base),
    };
    let opened = root.stat(".").map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !same_file(&info, &opened) {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    Ok(root)
}

fn ensure_runtime_project_dir(parent: &Root, project: &str, uid: u32) -> Result<(), String> {
    let info = parent.stat(".").map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !root_directory(&info, uid) {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    match mkdir_700(&parent.join(project)) {
        Ok(()) => {}
        Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => {}
        Err(_) => return Err(ERR_UNAVAILABLE.to_string()),
    }
    let info = parent
        .lstat(project)
        .map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !root_directory(&info, uid) {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    Ok(())
}

/// Mirror of `filelock.Acquire` with `LOCK_EX`: poll `flock(LOCK_EX|LOCK_NB)`,
/// retry `EWOULDBLOCK`/`EINTR` on a 25ms tick, report other errors by message,
/// and return `"context deadline exceeded"` (Go `ctx.Err()` text) on expiry.
fn acquire_exclusive(lock: &File, deadline: Instant) -> Result<(), String> {
    loop {
        if Instant::now() >= deadline {
            return Err("context deadline exceeded".to_string());
        }
        let r = unsafe { libc::flock(lock.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
        if r == 0 {
            return Ok(());
        }
        let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
        if errno != libc::EWOULDBLOCK && errno != libc::EINTR {
            return Err(std::io::Error::from_raw_os_error(errno).to_string());
        }
        let now = Instant::now();
        if now >= deadline {
            return Err("context deadline exceeded".to_string());
        }
        std::thread::sleep((deadline - now).min(Duration::from_millis(25)));
    }
}

fn lock_runtime_project(
    root: &Root,
    uid: u32,
    gid: u32,
    deadline: Instant,
) -> Result<File, String> {
    let mut opts = OpenOptions::new();
    opts.read(true)
        .write(true)
        .create(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW);
    let lock = opts
        .open(root.join("lock"))
        .map_err(|_| ERR_UNAVAILABLE.to_string())?;
    let info = lock.metadata().map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !runtime_file(&info, uid, gid, 0o600) {
        drop(lock);
        return Err(ERR_UNAVAILABLE.to_string());
    }
    if let Err(e) = acquire_exclusive(&lock, deadline) {
        drop(lock);
        return Err(e);
    }
    Ok(lock)
}

/// Open (creating if needed) and exclusively lock a runtime project directory.
/// Unsafe existing paths are refused, never repaired.
pub fn open_runtime_project_owned(
    base: &str,
    project: &str,
    uid: u32,
    gid: u32,
    deadline: Instant,
) -> Result<RunFiles, String> {
    if !valid_project_id(project) {
        return Err(ERR_INVALID.to_string());
    }
    let parent = open_runtime_root(base, uid)?;
    ensure_runtime_project_dir(&parent, project, uid)?;
    let root = Root {
        path: parent.join(project),
    };
    let lock = lock_runtime_project(&root, uid, gid, deadline)?;
    Ok(RunFiles {
        root,
        lock,
        uid,
        gid,
    })
}

/// Production open: the runtime tree is appliance-owned (`0:0`).
pub fn open_runtime_project(
    base: &str,
    project: &str,
    deadline: Instant,
) -> Result<RunFiles, String> {
    open_runtime_project_owned(base, project, 0, 0, deadline)
}

mod run;

mod keys;

pub use keys::{
    read_companion_id, retire_pending_run_key, retire_run_key, write_companion_id, write_run_key,
};

mod resolver;

pub use resolver::validate_run_resolver;

#[cfg(test)]
mod tests;
