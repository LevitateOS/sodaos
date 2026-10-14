//! Cross-process exclusion between managed-work admission and explicit maintenance.
use std::fs::{File, OpenOptions};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt, PermissionsExt};
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

fn lock(directory: &Path, exclusive: bool) -> Result<AdmissionGuard, String> {
    use std::os::fd::AsRawFd;
    std::fs::create_dir_all(directory).map_err(|_| UNAVAILABLE.to_string())?;
    let info = std::fs::symlink_metadata(directory).map_err(|_| UNAVAILABLE.to_string())?;
    if !info.is_dir() || info.uid() != unsafe { libc::geteuid() } || info.mode() & 0o022 != 0 {
        return Err(UNAVAILABLE.to_string());
    }
    let file = OpenOptions::new()
        .read(true)
        .write(true)
        .create(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC)
        .open(directory.join("admission.lock"))
        .map_err(|_| UNAVAILABLE.to_string())?;
    let info = file.metadata().map_err(|_| UNAVAILABLE.to_string())?;
    if !info.is_file()
        || info.uid() != unsafe { libc::geteuid() }
        || info.permissions().mode() & 0o077 != 0
    {
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

fn admit_at(directory: &Path, project: Option<&str>) -> Result<AdmissionGuard, String> {
    let guard = lock(directory, false)?;
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
    admit_at(Path::new(DIRECTORY), project)
}

pub fn exclusive() -> Result<AdmissionGuard, String> {
    lock(Path::new(DIRECTORY), true)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn exclusion_and_interrupted_marker_close_admission() {
        let dir = std::env::temp_dir().join(format!("soda-admission-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let shared = admit_at(&dir, Some("p0123456789abcdef01234567")).unwrap();
        assert!(lock(&dir, true).is_err());
        drop(shared);
        let exclusive = lock(&dir, true).unwrap();
        assert!(admit_at(&dir, None).is_err());
        std::fs::write(dir.join("p0123456789abcdef01234567.json"), b"interrupted").unwrap();
        drop(exclusive);
        assert!(admit_at(&dir, Some("p0123456789abcdef01234567")).is_err());
        assert!(admit_at(&dir, None).is_err());
        std::fs::remove_file(dir.join("p0123456789abcdef01234567.json")).unwrap();
        drop(admit_at(&dir, None).unwrap());
        // A malformed/unreadable lock is an unavailable observation, not
        // evidence that maintenance is running.
        std::fs::set_permissions(
            dir.join("admission.lock"),
            std::fs::Permissions::from_mode(0o666),
        )
        .unwrap();
        assert!(matches!(admit_at(&dir, None), Err(reason) if reason == UNAVAILABLE));
        std::fs::remove_dir_all(dir).unwrap();
    }
}
