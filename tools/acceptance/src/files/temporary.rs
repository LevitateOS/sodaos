use std::fs::OpenOptions;
use std::io::Write;
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::Path;

use crate::error::Error;

use super::{c_string, lexical_clean};

/// Create a fresh private directory whose parent has no symlinks.
/// Mirrors `FreshDirectory`.
pub fn fresh_directory(path: &str) -> Result<(), Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute new directory required"));
    }
    let parent = Path::new(path).parent().unwrap_or(Path::new("/"));
    let resolved = std::fs::canonicalize(parent)?;
    if resolved.to_string_lossy() != lexical_clean(&parent.to_string_lossy()) {
        return Err(Error::msg("symlinked parent refused"));
    }
    let raw = c_string(Path::new(path))?;
    let rc = unsafe { libc::mkdir(raw.as_ptr(), 0o700) };
    if rc != 0 {
        return Err(Error::from(std::io::Error::last_os_error()));
    }
    Ok(())
}

/// Admit a new private output path: absolute, unoccupied, below a real
/// private directory. Mirrors `PrivateDestination`.
pub fn private_destination(path: &str) -> Result<(), Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute private output required"));
    }
    let parent = Path::new(path).parent().unwrap_or(Path::new("/"));
    let parent_text = parent.to_string_lossy();
    // Like Go, resolution and stat failures propagate raw; only the
    // shape violations below carry the fixed message.
    let resolved = std::fs::canonicalize(parent)?;
    let meta = std::fs::metadata(parent)?;
    if resolved.to_string_lossy() != parent_text || meta.mode() & 0o077 != 0 {
        return Err(Error::msg("real private output parent required"));
    }
    match std::fs::symlink_metadata(path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        _ => Err(Error::msg("output already exists or cannot be inspected")),
    }
}

/// Private scratch directory, removed on drop.
#[derive(Debug)]
pub struct TempDir {
    path: std::path::PathBuf,
}

impl TempDir {
    /// Create a fresh private scratch directory.
    pub fn new(prefix: &str) -> Result<TempDir, Error> {
        static NEXT: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
        let mut path = std::env::temp_dir();
        path.push(format!(
            "{prefix}-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, std::sync::atomic::Ordering::SeqCst)
        ));
        fresh_directory(&path.to_string_lossy())?;
        Ok(TempDir { path })
    }

    /// Join a name below the directory.
    pub fn join(&self, name: &str) -> std::path::PathBuf {
        self.path.join(name)
    }

    /// Directory path.
    pub fn path(&self) -> &std::path::Path {
        &self.path
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.path);
    }
}

/// Exclusive file creation with exact mode bits. Mirrors `WriteNew`.
pub fn write_new(path: &str, data: &[u8], mode: u32) -> Result<(), Error> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(mode)
        .open(path)?;
    file.write_all(data)?;
    Ok(())
}
