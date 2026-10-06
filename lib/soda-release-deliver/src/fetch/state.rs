use std::os::fd::AsRawFd;
use std::sync::atomic::{AtomicU64, Ordering};

use crate::jsonx::{marshal, Emit};
use crate::native::private_file;
use crate::Error;

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

static TEMP_COUNTER: AtomicU64 = AtomicU64::new(0);

fn create_temp(dir: &str, prefix: &str) -> Result<(std::fs::File, String), Error> {
    use std::os::unix::fs::OpenOptionsExt;
    for _ in 0..100 {
        let id = TEMP_COUNTER.fetch_add(1, Ordering::Relaxed);
        let name = format!("{dir}/{prefix}{}-{}", std::process::id(), id);
        match std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(&name)
        {
            Ok(file) => return Ok((file, name)),
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(e) => return Err(Error::msg(format!("create temp: {e}"))),
        }
    }
    Err(Error::msg("create temp: too many attempts"))
}

pub(crate) fn save_state<T: Emit + ?Sized>(path: &str, value: &T) -> Result<(), Error> {
    let data = marshal(value);
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
