//! Private filesystem inputs: exact ports of the `internal/release/build`
//! file helpers plus an fd-confined directory handle mirroring Go's `os.Root`.
//!
//! Constructed messages match the Go owner byte for byte. Raw OS errors keep
//! their [`std::io::ErrorKind`] but render with Rust's `(os error N)` suffix;
//! tests assert kinds and constructed messages, never raw OS text.

use std::ffi::CString;
use std::os::unix::ffi::OsStrExt;
use std::path::Path;

use crate::error::Error;

mod inputs;
mod owned_directory;
mod temporary;

pub use inputs::{hash_at, hash_file, private_file};
pub use owned_directory::OwnedDir;
pub use temporary::{fresh_directory, private_destination, write_new, TempDir};

/// Maximum private-file input: `PrivateFile` bound.
pub const PRIVATE_FILE_LIMIT: u64 = 1 << 20;
/// Maximum decoded JSON input: `ReadJSONAt` bound.
pub const JSON_LIMIT: u64 = 4 << 20;

/// Lexical path clean, matching Go's `filepath.Clean`: duplicate separators
/// collapse, `.` drops out, `..` pops the previous component without touching
/// the filesystem, and a trailing separator is removed except at the root.
pub fn lexical_clean(path: &str) -> String {
    let absolute = path.starts_with('/');
    let mut parts: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if parts.last().is_some_and(|last| *last != "..") {
                    parts.pop();
                } else if !absolute {
                    parts.push("..");
                }
            }
            _ => parts.push(part),
        }
    }
    let mut out = parts.join("/");
    if absolute {
        out.insert(0, '/');
    }
    if out.is_empty() {
        out.push('.');
    }
    out
}

fn c_string(path: &Path) -> Result<CString, Error> {
    CString::new(path.as_os_str().as_bytes()).map_err(|_| {
        Error::from(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "path contains NUL",
        ))
    })
}

/// File identity plus the mode bits callers gate on.
#[derive(Debug, Clone, Copy)]
pub struct FileAttr {
    /// Regular file (never a symlink: always read with `AT_SYMLINK_NOFOLLOW`).
    pub is_regular: bool,
    /// Directory.
    pub is_dir: bool,
    /// Symlink.
    pub is_symlink: bool,
    /// Permission bits (`st_mode & 0o7777`).
    pub perm: u32,
    /// Size in bytes.
    pub size: u64,
    /// Owner UID/GID.
    pub uid: u32,
    /// Owner UID/GID.
    pub gid: u32,
    /// Device + inode for same-file comparison.
    pub dev: u64,
    /// Device + inode for same-file comparison.
    pub ino: u64,
}

impl FileAttr {
    fn from_stat(st: &libc::stat) -> FileAttr {
        let mode = st.st_mode;
        FileAttr {
            is_regular: mode & libc::S_IFMT == libc::S_IFREG,
            is_dir: mode & libc::S_IFMT == libc::S_IFDIR,
            is_symlink: mode & libc::S_IFMT == libc::S_IFLNK,
            perm: mode & 0o7777,
            size: st.st_size.max(0) as u64,
            uid: st.st_uid,
            gid: st.st_gid,
            dev: st.st_dev,
            ino: st.st_ino,
        }
    }
}

/// True when both attributes describe the same file, like `os.SameFile`.
pub fn same_file(a: FileAttr, b: FileAttr) -> bool {
    a.dev == b.dev && a.ino == b.ino
}

#[cfg(test)]
mod tests;
