//! Confined filesystem custody: descriptor-relative opens, no-follow
//! lstat/open, inode identity checks, and hashing through an open root.

use crate::{io_error, sha256_hex_stream, Error};
use std::fs::File as FsFile;
use std::os::fd::{AsFd, AsRawFd, FromRawFd, IntoRawFd, OwnedFd};
use std::os::unix::ffi::OsStrExt;
use std::path::Path;

/// Confined directory handle emulating Go's `os.Root`: every resolution
/// stays inside the opened directory; intermediate symlinks are refused.
pub struct Root {
    fd: OwnedFd,
}

#[derive(Debug, Clone)]
pub struct FileMeta {
    pub is_regular: bool,
    pub size: u64,
    pub dev: u64,
    pub ino: u64,
}

#[allow(clippy::unnecessary_cast)] // dev_t/ino_t widths are platform-dependent
fn stat_to_meta(st: &libc::stat) -> FileMeta {
    FileMeta {
        is_regular: (st.st_mode & libc::S_IFMT) == libc::S_IFREG,
        size: st.st_size.max(0) as u64,
        dev: st.st_dev as u64,
        ino: st.st_ino as u64,
    }
}

impl Root {
    /// Opens a real directory without following a final symlink.
    pub fn open(dir: &Path) -> Result<Root, Error> {
        let st = std::fs::symlink_metadata(dir).map_err(|e| io_error("lstat", dir, e))?;
        if !st.is_dir() || st.file_type().is_symlink() {
            return Err(Error::msg(format!(
                "open {}: not a directory",
                dir.display()
            )));
        }
        let fd = open_at(
            libc::AT_FDCWD,
            dir,
            libc::O_PATH | libc::O_DIRECTORY | libc::O_NOFOLLOW,
        )?;
        Ok(Root { fd })
    }

    fn resolve_parent(&self, name: &str) -> Result<(OwnedFd, String), Error> {
        use std::os::fd::BorrowedFd;
        if name.is_empty() || name.starts_with('/') {
            return Err(Error::msg("confined path escapes root"));
        }
        let parts: Vec<&str> = name.split('/').collect();
        if parts
            .iter()
            .any(|p| p.is_empty() || *p == "." || *p == "..")
        {
            return Err(Error::msg("confined path escapes root"));
        }
        let mut dir_fd: BorrowedFd<'_> = self.fd.as_fd();
        let mut owned: Vec<OwnedFd> = Vec::new();
        for component in &parts[..parts.len() - 1] {
            let child = open_at_fd(
                dir_fd,
                component,
                libc::O_PATH | libc::O_DIRECTORY | libc::O_NOFOLLOW,
            )
            .map_err(|e| Error::msg(format!("open {component}: {e}")))?;
            owned.push(child);
            dir_fd = owned.last().unwrap().as_fd();
        }
        // Duplicate the final dirfd so the returned fd is independent.
        let raw = dir_fd.as_raw_fd();
        let duped = unsafe { libc::fcntl(raw, libc::F_DUPFD_CLOEXEC, 0) };
        if duped < 0 {
            return Err(Error::msg(format!(
                "open {name}: {}",
                std::io::Error::last_os_error()
            )));
        }
        let out = unsafe { OwnedFd::from_raw_fd(duped) };
        Ok((out, parts[parts.len() - 1].to_string()))
    }

    /// `lstat` confined to the root.
    pub fn lstat(&self, name: &str) -> Result<FileMeta, Error> {
        let (dir, leaf) = self.resolve_parent(name)?;
        let leaf_c = std::ffi::CString::new(leaf.as_str()).unwrap();
        let mut st: libc::stat = unsafe { std::mem::zeroed() };
        let rc = unsafe {
            libc::fstatat(
                dir.as_raw_fd(),
                leaf_c.as_ptr(),
                &mut st,
                libc::AT_SYMLINK_NOFOLLOW,
            )
        };
        if rc != 0 {
            return Err(Error::msg(format!(
                "lstat {name}: {}",
                std::io::Error::last_os_error()
            )));
        }
        Ok(stat_to_meta(&st))
    }

    /// Opens a regular file without following a final symlink.
    pub fn open_file(&self, name: &str) -> Result<FsFile, Error> {
        let (dir, leaf) = self.resolve_parent(name)?;
        let fd = open_at_fd(
            dir.as_fd(),
            &leaf,
            libc::O_RDONLY | libc::O_NONBLOCK | libc::O_NOFOLLOW,
        )
        .map_err(|e| Error::msg(format!("open {name}: {e}")))?;
        Ok(unsafe { FsFile::from(OwnedFd::from_raw_fd(fd.into_raw_fd())) })
    }

    /// Opens `name` only if it is still the same regular file.
    pub fn open_unchanged(&self, name: &str, st: &FileMeta) -> Result<FsFile, Error> {
        let f = self.open_file(name)?;
        let actual = f
            .metadata()
            .map_err(|e| io_error("stat", Path::new(name), e))?;
        if !is_regular_meta(&actual) || !same_file_meta(st, &actual) {
            return Err(Error::msg("JSON input changed before reading"));
        }
        Ok(f)
    }
}

fn open_at(dirfd: libc::c_int, path: &Path, flags: libc::c_int) -> Result<OwnedFd, Error> {
    let c = std::ffi::CString::new(path.as_os_str().as_bytes())
        .map_err(|_| Error::msg(format!("open {}: invalid path", path.display())))?;
    let raw = unsafe { libc::openat(dirfd, c.as_ptr(), flags | libc::O_CLOEXEC, 0) };
    if raw < 0 {
        return Err(Error::msg(format!(
            "open {}: {}",
            path.display(),
            std::io::Error::last_os_error()
        )));
    }
    Ok(unsafe { OwnedFd::from_raw_fd(raw) })
}

fn open_at_fd(dirfd: impl AsFd, leaf: &str, flags: libc::c_int) -> Result<OwnedFd, std::io::Error> {
    let c = std::ffi::CString::new(leaf)
        .map_err(|e| std::io::Error::new(std::io::ErrorKind::InvalidInput, e.to_string()))?;
    let raw = unsafe {
        libc::openat(
            dirfd.as_fd().as_raw_fd(),
            c.as_ptr(),
            flags | libc::O_CLOEXEC,
            0,
        )
    };
    if raw < 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(unsafe { OwnedFd::from_raw_fd(raw) })
}

fn is_regular_meta(meta: &std::fs::Metadata) -> bool {
    meta.is_file() && !meta.file_type().is_symlink()
}

fn same_file_meta(st: &FileMeta, meta: &std::fs::Metadata) -> bool {
    use std::os::unix::fs::MetadataExt;
    st.dev == meta.dev() && st.ino == meta.ino()
}

/// SHA-256 through a confined root, refusing mid-read swaps.
pub fn hash_at(root: &Root, name: &str) -> Result<String, Error> {
    let st = root.lstat(name)?;
    if !st.is_regular {
        return Err(Error::msg("regular non-symlink file required"));
    }
    let mut f = root.open_file(name)?;
    let actual = f
        .metadata()
        .map_err(|e| io_error("stat", Path::new(name), e))?;
    if !is_regular_meta(&actual) || !same_file_meta(&st, &actual) {
        return Err(Error::msg("file changed before hashing"));
    }
    sha256_hex_stream(&mut f).map_err(|e| io_error("read", Path::new(name), e))
}
