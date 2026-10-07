use std::os::fd::AsRawFd;

use crate::native::private_file;
use crate::Error;
use serde::Serialize;

pub(crate) struct StateLock {
    #[allow(dead_code)]
    file: std::fs::File,
}

impl std::fmt::Debug for StateLock {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("StateLock")
    }
}

impl Drop for StateLock {
    fn drop(&mut self) {
        unsafe {
            libc::flock(self.file.as_raw_fd(), libc::LOCK_UN);
        }
    }
}

pub(crate) fn lock_state(path: &str) -> Result<StateLock, Error> {
    private_file(path)?;
    let lock_path = format!("{path}.lock");
    use std::os::unix::fs::OpenOptionsExt;
    let file = std::fs::OpenOptions::new()
        .read(true)
        .write(true)
        .create(true)
        .custom_flags(libc::O_NOFOLLOW)
        .mode(0o600)
        .open(&lock_path)
        .map_err(|e| Error::msg(format!("open {lock_path}: {e}")))?;
    if unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) } != 0 {
        return Err(Error::msg("release operation already active"));
    }
    Ok(StateLock { file })
}

fn create_temp(dir: &str, prefix: &str) -> Result<(std::fs::File, std::path::PathBuf), Error> {
    let staged = tempfile::Builder::new()
        .prefix(prefix)
        .tempfile_in(dir)
        .map_err(|e| Error::msg(format!("create temp: {e}")))?;
    // Incomplete state is operator evidence. Transfer its lifecycle before the
    // first write so a write/sync failure preserves that exact owned attempt.
    staged
        .keep()
        .map_err(|e| Error::msg(format!("retain temp: {}", e.error)))
}

pub(crate) fn save_state<T: Serialize + ?Sized>(path: &str, value: &T) -> Result<(), Error> {
    let data = crate::document::marshal_go_pretty(value)?;
    let parent = match path.rfind('/') {
        Some(0) | None => "/".to_string(),
        Some(i) => path[..i].to_string(),
    };
    let (mut file, temp_path) = create_temp(&parent, ".delivery-state-")?;
    use std::io::Write;
    // Preserve incomplete write attempts rather than repairing them over later state.
    file.write_all(&data)
        .map_err(|e| Error::msg(format!("write state: {e}")))?;
    file.sync_all()
        .map_err(|e| Error::msg(format!("sync state: {e}")))?;
    drop(file);
    std::fs::rename(&temp_path, path).map_err(|e| Error::msg(format!("rename state: {e}")))?;
    let dir =
        std::fs::File::open(&parent).map_err(|e| Error::msg(format!("open {parent}: {e}")))?;
    dir.sync_all()
        .map_err(|e| Error::msg(format!("sync {parent}: {e}")))
}
