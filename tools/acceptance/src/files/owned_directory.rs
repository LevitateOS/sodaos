use std::ffi::{CStr, CString};
use std::fs::File;
use std::os::fd::{AsRawFd, FromRawFd, OwnedFd};
use std::path::Path;

use crate::error::Error;

use super::{c_string, FileAttr};

/// An open directory handle. All operations resolve beneath it, mirroring
/// Go's `os.Root` (`openat2` with `RESOLVE_BENEATH`): intermediate symlinks
/// and escapes fail instead of resolving outside.
#[derive(Debug)]
pub struct OwnedDir {
    fd: OwnedFd,
    /// Original path, for display and disjointness checks only. All I/O uses
    /// the fd, so a renamed root keeps working like the Go owner.
    path: String,
}

impl OwnedDir {
    /// Open an existing directory, like `os.OpenRoot`.
    pub fn open(path: &str) -> Result<OwnedDir, Error> {
        let raw = c_string(Path::new(path))?;
        let fd = unsafe {
            libc::open(
                raw.as_ptr(),
                libc::O_RDONLY | libc::O_DIRECTORY | libc::O_CLOEXEC,
            )
        };
        if fd < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        Ok(OwnedDir {
            // SAFETY: `open` returned a fresh owned fd.
            fd: unsafe { OwnedFd::from_raw_fd(fd) },
            path: path.to_string(),
        })
    }

    /// Original path this handle was opened from.
    pub fn path(&self) -> &str {
        &self.path
    }

    fn single_component(name: &str) -> Result<CString, Error> {
        if name.is_empty() || name == "." || name == ".." || name.contains('/') {
            return Err(Error::msg("invalid evidence name"));
        }
        c_string(Path::new(name))
    }

    /// Open one child directory, refusing symlinks (`O_NOFOLLOW`).
    fn open_child_dir(fd: &OwnedFd, name: &CStr) -> Result<OwnedFd, Error> {
        let child = unsafe {
            libc::openat(
                fd.as_raw_fd(),
                name.as_ptr(),
                libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            )
        };
        if child < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        // SAFETY: `openat` returned a fresh owned fd.
        Ok(unsafe { OwnedFd::from_raw_fd(child) })
    }

    /// Traverse `parent` (possibly empty) component by component beneath this
    /// root, refusing symlinks and `..`, like Go's beneath-resolving root.
    fn traverse(&self, parent: &str) -> Result<OwnedFd, Error> {
        let mut fd = self.duplicate()?;
        if parent.is_empty() {
            return Ok(fd);
        }
        for component in parent.split('/') {
            let raw = OwnedDir::single_component(component)?;
            fd = OwnedDir::open_child_dir(&fd, &raw)?;
        }
        Ok(fd)
    }

    fn split_parent(name: &str) -> Result<(&str, CString), Error> {
        let (parent, base) = match name.rsplit_once('/') {
            Some((parent, base)) => (parent, base),
            None => ("", name),
        };
        Ok((parent, OwnedDir::single_component(base)?))
    }

    /// `lstat` a path beneath this root without following symlinks.
    pub fn lstat_at(&self, name: &str) -> Result<FileAttr, Error> {
        let (parent, base) = OwnedDir::split_parent(name)?;
        let dir = self.traverse(parent)?;
        let mut st: libc::stat = unsafe { std::mem::zeroed() };
        let rc = unsafe {
            libc::fstatat(
                dir.as_raw_fd(),
                base.as_ptr(),
                &mut st,
                libc::AT_SYMLINK_NOFOLLOW,
            )
        };
        if rc != 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        Ok(FileAttr::from_stat(&st))
    }

    /// Open a file beneath this root for reading, refusing escapes. The
    /// final component is opened `O_NOFOLLOW`; callers re-stat, like the Go
    /// owner's unchanged-regular check.
    pub fn open_file_at(&self, name: &str) -> Result<File, Error> {
        let (parent, base) = OwnedDir::split_parent(name)?;
        let dir = self.traverse(parent)?;
        let fd = unsafe {
            libc::openat(
                dir.as_raw_fd(),
                base.as_ptr(),
                libc::O_RDONLY | libc::O_NONBLOCK | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            )
        };
        if fd < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        // SAFETY: `openat` returned a fresh owned fd.
        Ok(File::from(unsafe { OwnedFd::from_raw_fd(fd) }))
    }

    /// Open a subdirectory beneath this root, refusing escapes.
    pub fn sub_dir(&self, name: &str) -> Result<OwnedDir, Error> {
        let mut fd = self.duplicate()?;
        for component in name.split('/') {
            let raw = OwnedDir::single_component(component)?;
            fd = OwnedDir::open_child_dir(&fd, &raw)?;
        }
        Ok(OwnedDir {
            fd,
            path: format!("{}/{}", self.path, name),
        })
    }

    /// `mkdirat` beneath this root with exact mode bits, traversing
    /// intermediate components like Go's `Root.Mkdir` on nested names.
    pub fn mkdir_at(&self, name: &str, mode: u32) -> Result<(), Error> {
        let (parent, base) = OwnedDir::split_parent(name)?;
        let dir = self.traverse(parent)?;
        let rc = unsafe { libc::mkdirat(dir.as_raw_fd(), base.as_ptr(), mode) };
        if rc != 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        Ok(())
    }

    /// Exclusive `O_CREAT|O_EXCL` file creation beneath this root.
    pub fn create_new_at(&self, name: &str, mode: u32) -> Result<File, Error> {
        let (parent, base) = OwnedDir::split_parent(name)?;
        let dir = self.traverse(parent)?;
        let fd = unsafe {
            libc::openat(
                dir.as_raw_fd(),
                base.as_ptr(),
                libc::O_WRONLY | libc::O_CREAT | libc::O_EXCL | libc::O_CLOEXEC,
                mode,
            )
        };
        if fd < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        // SAFETY: `openat` returned a fresh owned fd.
        Ok(File::from(unsafe { OwnedFd::from_raw_fd(fd) }))
    }

    /// Exclusive hard link beneath this root, like `Root.Link`.
    pub fn link_at(&self, old: &str, new: &str) -> Result<(), Error> {
        let (old_parent, old_base) = OwnedDir::split_parent(old)?;
        let (new_parent, new_base) = OwnedDir::split_parent(new)?;
        let old_dir = self.traverse(old_parent)?;
        let new_dir = self.traverse(new_parent)?;
        let rc = unsafe {
            libc::linkat(
                old_dir.as_raw_fd(),
                old_base.as_ptr(),
                new_dir.as_raw_fd(),
                new_base.as_ptr(),
                0,
            )
        };
        if rc != 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        Ok(())
    }

    fn duplicate(&self) -> Result<OwnedFd, Error> {
        let fd = unsafe { libc::fcntl(self.fd.as_raw_fd(), libc::F_DUPFD_CLOEXEC, 0) };
        if fd < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        // SAFETY: `fcntl` returned a fresh owned fd.
        Ok(unsafe { OwnedFd::from_raw_fd(fd) })
    }

    /// Walk entries beneath this root, yielding root-relative paths with a
    /// regularity flag. Directories recurse; callers report their own error
    /// for non-regular entries like the Go owner does.
    pub fn walk_files(
        &self,
        visit: &mut dyn FnMut(&str, bool) -> Result<(), Error>,
    ) -> Result<(), Error> {
        self.walk_recursion("", visit)
    }

    fn walk_recursion(
        &self,
        prefix: &str,
        visit: &mut dyn FnMut(&str, bool) -> Result<(), Error>,
    ) -> Result<(), Error> {
        let dir = if prefix.is_empty() {
            OwnedDir {
                fd: self.duplicate()?,
                path: self.path.clone(),
            }
        } else {
            self.sub_dir(prefix)?
        };
        // `fdopendir` takes ownership, and a duplicate would share this
        // handle's directory offset across walks. Open "." beneath the
        // pinned fd instead: an independent description rooted at the same
        // inode, without reopening the public pathname.
        let dot = c_string(Path::new("."))?;
        let raw = unsafe {
            libc::openat(
                dir.fd.as_raw_fd(),
                dot.as_ptr(),
                libc::O_RDONLY | libc::O_DIRECTORY | libc::O_CLOEXEC,
            )
        };
        if raw < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        let stream = unsafe { libc::fdopendir(raw) };
        if stream.is_null() {
            unsafe { libc::close(raw) };
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        let mut result = Ok(());
        loop {
            unsafe { *libc::__errno_location() = 0 };
            let entry = unsafe { libc::readdir(stream) };
            if entry.is_null() {
                let errno = unsafe { *libc::__errno_location() };
                if errno != 0 {
                    result = Err(Error::from(std::io::Error::from_raw_os_error(errno)));
                }
                break;
            }
            let name = unsafe { CStr::from_ptr((*entry).d_name.as_ptr()) }.to_string_lossy();
            if name == "." || name == ".." {
                continue;
            }
            let rel = if prefix.is_empty() {
                name.to_string()
            } else {
                format!("{prefix}/{name}")
            };
            let attr = match dir.lstat_at(&name) {
                Ok(attr) => attr,
                Err(e) => {
                    result = Err(e);
                    break;
                }
            };
            if attr.is_dir {
                if let Err(e) = self.walk_recursion(&rel, visit) {
                    result = Err(e);
                    break;
                }
            } else if let Err(e) = visit(&rel, attr.is_regular) {
                result = Err(e);
                break;
            }
        }
        unsafe { libc::closedir(stream) };
        result
    }
}
