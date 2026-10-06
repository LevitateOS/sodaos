use std::fs::{self, File, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::OpenOptionsExt;
use std::os::unix::io::AsRawFd;

use crate::tailnet_domain::{
    valid_container_id, valid_project_id, ERR_CONFLICT, ERR_INVALID, ERR_UNAVAILABLE,
    ERR_UNCONFIRMED,
};
use crate::tailnet_runtime::ProjectRun;

use super::{open_runtime_root, root_directory, runtime_file, same_file, Root};

/// Write the single-use key. It never enters argv, environment, a pipe, journal
/// or the project filesystem. Removal is allowed only after native exec
/// completion is observed.
pub fn write_run_key(root: &Root, run: &ProjectRun, key: &str) -> Result<File, String> {
    if !key.starts_with("tskey-auth-") || key.len() > 1024 || key.contains(['\r', '\n', '\0']) {
        return Err(ERR_INVALID.to_string());
    }
    let mut opts = OpenOptions::new();
    opts.write(true)
        .create_new(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW);
    let mut file = opts
        .open(root.join("input/key"))
        .map_err(|_| ERR_CONFLICT.to_string())?;
    if unsafe { libc::fchown(file.as_raw_fd(), run.uid, run.gid) } != 0 {
        return Err(ERR_UNCONFIRMED.to_string());
    }
    if file.write_all(key.as_bytes()).is_err() {
        return Err(ERR_UNCONFIRMED.to_string());
    }
    if file.sync_all().is_err() {
        return Err(ERR_UNCONFIRMED.to_string());
    }
    Ok(file)
}

/// Zero the exact open key inode and unlink the path. Refuses when the path no
/// longer refers to the same inode, so a later write is never retired.
pub fn retire_run_key(root: &Root, file: &File) -> Result<(), String> {
    let before = file.metadata().map_err(|_| ERR_UNCONFIRMED.to_string())?;
    let after = root
        .lstat("input/key")
        .map_err(|_| ERR_UNCONFIRMED.to_string())?;
    if !same_file(&before, &after) {
        return Err(ERR_UNCONFIRMED.to_string());
    }
    if file.set_len(0).is_err()
        || file.sync_all().is_err()
        || fs::remove_file(root.join("input/key")).is_err()
    {
        return Err(ERR_UNCONFIRMED.to_string());
    }
    Ok(())
}

fn pending_run_key_matches(info: &fs::Metadata, run: &ProjectRun) -> bool {
    runtime_file(info, run.uid, run.gid, 0o600) && info.len() <= 1024
}

fn open_pending_run_key(root: &Root, run: &ProjectRun) -> Result<Option<File>, String> {
    let before = match root.lstat("input/key") {
        Ok(m) => m,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(None),
        Err(_) => return Err(ERR_UNCONFIRMED.to_string()),
    };
    if !pending_run_key_matches(&before, run) {
        return Err(ERR_UNCONFIRMED.to_string());
    }
    let mut opts = OpenOptions::new();
    opts.write(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK);
    let file = opts
        .open(root.join("input/key"))
        .map_err(|_| ERR_UNCONFIRMED.to_string())?;
    let info = file.metadata().map_err(|_| ERR_UNCONFIRMED.to_string())?;
    if !same_file(&before, &info) || !pending_run_key_matches(&info, run) {
        drop(file);
        return Err(ERR_UNCONFIRMED.to_string());
    }
    Ok(Some(file))
}

/// Retire a completed consumer's exact pending input for an explicit retry.
/// Called only under the runtime lock after observing no outstanding native
/// exec. Absent input is fine; substituted input is refused.
pub fn retire_pending_run_key(root: &Root, run: &ProjectRun) -> Result<(), String> {
    let file = match open_pending_run_key(root, run)? {
        Some(f) => f,
        None => return Ok(()),
    };
    let r = retire_run_key(root, &file);
    drop(file);
    r
}

/// Durably record the companion container identity exactly once; replacement
/// is a conflict, never an update.
pub fn write_companion_id(root: &Root, id: &str) -> Result<(), String> {
    if !valid_container_id(id) {
        return Err(ERR_UNCONFIRMED.to_string());
    }
    let mut opts = OpenOptions::new();
    opts.write(true)
        .create_new(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW);
    let mut file = opts
        .open(root.join("companion-id"))
        .map_err(|_| ERR_CONFLICT.to_string())?;
    if file.write_all(id.as_bytes()).is_err() {
        return Err(ERR_UNCONFIRMED.to_string());
    }
    if file.sync_all().is_err() {
        return Err(ERR_UNCONFIRMED.to_string());
    }
    drop(file);
    let dir = File::open(root.join(".")).map_err(|_| ERR_UNCONFIRMED.to_string())?;
    if dir.sync_all().is_err() {
        return Err(ERR_UNCONFIRMED.to_string());
    }
    Ok(())
}

fn read_companion_id_file(root: &Root, path: &str, uid: u32, gid: u32) -> Result<String, String> {
    let mut opts = OpenOptions::new();
    opts.read(true).custom_flags(libc::O_NOFOLLOW);
    let file = opts
        .open(root.join(path))
        .map_err(|_| ERR_UNAVAILABLE.to_string())?;
    let info = file.metadata().map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !runtime_file(&info, uid, gid, 0o600) || info.len() != 64 {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    let mut b = Vec::new();
    file.take(65)
        .read_to_end(&mut b)
        .map_err(|_| ERR_UNAVAILABLE.to_string())?;
    let s = std::str::from_utf8(&b).map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !valid_container_id(s) {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    Ok(s.to_string())
}

/// Read the retained companion identity after re-verifying every ancestor is
/// still appliance-owned.
pub fn read_companion_id(
    base: &str,
    run: &ProjectRun,
    uid: u32,
    gid: u32,
) -> Result<String, String> {
    if !valid_project_id(&run.target.project) || !valid_container_id(&run.target.run) {
        return Err(ERR_INVALID.to_string());
    }
    let root = open_runtime_root(base, uid)?;
    let run_dir = format!("{}/{}", run.target.project, run.target.run);
    for path in [".", run.target.project.as_str(), run_dir.as_str()] {
        let info = root.lstat(path).map_err(|_| ERR_UNAVAILABLE.to_string())?;
        if !root_directory(&info, uid) {
            return Err(ERR_UNAVAILABLE.to_string());
        }
    }
    read_companion_id_file(&root, &format!("{run_dir}/companion-id"), uid, gid)
}
