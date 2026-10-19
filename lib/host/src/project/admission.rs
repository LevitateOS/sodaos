//! Cross-process exclusion between managed-work admission and explicit maintenance.
use std::fs::{File, OpenOptions};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::{Path, PathBuf};

pub const DIRECTORY: &str = "/var/lib/soda/project-maintenance";
pub const MAINTENANCE: &str = "project_maintenance";
pub const UNAVAILABLE: &str = "project_admission_unavailable";

pub struct AdmissionGuard(File);

impl Drop for AdmissionGuard {
    fn drop(&mut self) {
        use std::os::fd::AsRawFd;
        unsafe { libc::flock(self.0.as_raw_fd(), libc::LOCK_UN) };
    }
}

pub fn marker_path(project: &str) -> PathBuf {
    Path::new(DIRECTORY).join(format!("{project}.json"))
}

fn lock(directory: &Path, exclusive: bool, owner: u32) -> Result<AdmissionGuard, String> {
    use std::os::fd::AsRawFd;
    // Boot tmpfiles owns creation. Read-only directory access lets the
    // dashboard share this same inode without writable maintenance data.
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC)
        .open(directory)
        .map_err(|_| UNAVAILABLE.to_string())?;
    let info = file.metadata().map_err(|_| UNAVAILABLE.to_string())?;
    if !info.is_dir() || info.uid() != owner || info.mode() & 0o7777 != 0o755 {
        return Err(UNAVAILABLE.to_string());
    }
    let operation = if exclusive {
        libc::LOCK_EX
    } else {
        libc::LOCK_SH
    };
    if unsafe { libc::flock(file.as_raw_fd(), operation | libc::LOCK_NB) } != 0 {
        let code = match std::io::Error::last_os_error().raw_os_error() {
            Some(libc::EWOULDBLOCK) => MAINTENANCE,
            _ => UNAVAILABLE,
        };
        return Err(code.to_string());
    }
    Ok(AdmissionGuard(file))
}

fn admit_at(directory: &Path, project: Option<&str>, owner: u32) -> Result<AdmissionGuard, String> {
    let guard = lock(directory, false, owner)?;
    // Unknown directory entries/metadata failures are blockers, never proof of absence.
    for name in
        std::iter::once("operation.json".to_string()).chain(project.map(|id| format!("{id}.json")))
    {
        match std::fs::symlink_metadata(directory.join(name)) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
            Ok(_) => return Err(MAINTENANCE.to_string()),
            Err(_) => return Err(UNAVAILABLE.to_string()),
        }
    }
    // Creation must not cross a retained Project's interrupted operation.
    if project.is_none() {
        let entries = std::fs::read_dir(directory).map_err(|_| UNAVAILABLE.to_string())?;
        for entry in entries {
            let entry = entry.map_err(|_| UNAVAILABLE.to_string())?;
            if entry.path().extension().is_some_and(|v| v == "json") {
                return Err(MAINTENANCE.to_string());
            }
        }
    }
    Ok(guard)
}

pub fn admit(project: Option<&str>) -> Result<AdmissionGuard, String> {
    if project.is_some_and(|id| !crate::domain::valid_id(id)) {
        return Err(UNAVAILABLE.to_string());
    }
    admit_at(Path::new(DIRECTORY), project, 0)
}

/// Cleanup shares the same inode exclusion but never treats a retained
/// operation marker as authority to skip stopping an owned resource.
pub fn cleanup_guard() -> Result<AdmissionGuard, String> {
    lock(Path::new(DIRECTORY), false, 0)
}

pub fn exclusive() -> Result<AdmissionGuard, String> {
    lock(Path::new(DIRECTORY), true, 0)
}
