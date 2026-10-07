use std::ffi::CString;
use std::fs::File;
use std::os::unix::io::{AsRawFd, BorrowedFd, OwnedFd};

use crate::errors::{self, Error};

fn errno_error() -> Error {
    errors::os_error(std::io::Error::last_os_error())
}

pub(crate) fn fstat_fd(fd: i32) -> Result<libc::stat, Error> {
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(fd, &mut st) } != 0 {
        return Err(errno_error());
    }
    Ok(st)
}

pub(crate) fn open_dir_nofollow(path: &str) -> Result<OwnedFd, Error> {
    use rustix::fs::{Mode, OFlags};
    use std::path::{Component, Path};
    // Root the walk once and retain each parent descriptor. No intermediate
    // symlink can redirect private publication, including after a rename.
    if !Path::new(path).is_absolute() {
        return Err(Error::msg("absolute directory required"));
    }
    let flags = OFlags::RDONLY | OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC;
    let mut dir =
        rustix::fs::open("/", flags, Mode::empty()).map_err(|e| errors::os_error(e.into()))?;
    for component in Path::new(path).components() {
        match component {
            Component::RootDir => {}
            Component::Normal(name) => {
                dir = rustix::fs::openat(&dir, name, flags, Mode::empty())
                    .map_err(|e| errors::os_error(e.into()))?;
            }
            _ => return Err(Error::msg("clean absolute directory required")),
        }
    }
    Ok(dir)
}

pub(crate) fn openat_file(dir_fd: i32, name: &str, flags: i32, mode: u32) -> std::io::Result<File> {
    let name = CString::new(name).map_err(|_| std::io::Error::from_raw_os_error(libc::EINVAL))?;
    // SAFETY: SshDir's held descriptor outlives this scoped borrowed open.
    let dir = unsafe { BorrowedFd::borrow_raw(dir_fd) };
    rustix::fs::openat(
        dir,
        name.as_c_str(),
        rustix::fs::OFlags::from_bits_retain(flags as u32) | rustix::fs::OFlags::CLOEXEC,
        rustix::fs::Mode::from_bits_retain(mode),
    )
    .map(File::from)
    .map_err(Into::into)
}

/// Locked `.ssh` directory: unlocking runs before close, like Go's cleanup.
pub struct SshDir {
    fd: OwnedFd,
}

impl SshDir {
    pub(crate) fn raw(&self) -> i32 {
        self.fd.as_raw_fd()
    }
}

impl Drop for SshDir {
    fn drop(&mut self) {
        let _ = rustix::fs::flock(&self.fd, rustix::fs::FlockOperation::Unlock);
    }
}

pub(crate) fn open_and_lock_ssh_directory(home: &str, uid: u32) -> Result<SshDir, Error> {
    let home_fd = open_dir_nofollow(home)?;
    enrollment_safe_directory(home_fd.as_raw_fd(), uid)?;
    match rustix::fs::mkdirat(&home_fd, ".ssh", rustix::fs::Mode::from_bits_retain(0o700)) {
        Ok(()) | Err(rustix::io::Errno::EXIST) => {}
        Err(e) => return Err(errors::os_error(e.into())),
    }
    let fd = rustix::fs::openat(
        &home_fd,
        ".ssh",
        rustix::fs::OFlags::RDONLY
            | rustix::fs::OFlags::DIRECTORY
            | rustix::fs::OFlags::NOFOLLOW
            | rustix::fs::OFlags::CLOEXEC,
        rustix::fs::Mode::empty(),
    )
    .map_err(|e| errors::os_error(e.into()))?;
    let dir = SshDir { fd };
    enrollment_safe_directory(dir.raw(), uid)?;
    if rustix::fs::flock(
        &dir.fd,
        rustix::fs::FlockOperation::NonBlockingLockExclusive,
    )
    .is_err()
    {
        return Err(Error::msg("authorized keys are busy"));
    }
    Ok(dir)
}

pub(crate) fn validate_authorized_keys_stat(st: &libc::stat, uid: u32) -> Result<(), Error> {
    if st.st_mode & libc::S_IFMT != libc::S_IFREG
        || st.st_uid != uid
        || st.st_nlink != 1
        || st.st_mode & 0o022 != 0
        || st.st_size > 1 << 20
    {
        return Err(Error::msg(
            "existing authorized_keys must be a bounded, singly linked, safely owned regular file",
        ));
    }
    Ok(())
}

pub fn enrollment_safe_directory(fd: i32, uid: u32) -> Result<(), Error> {
    let st = fstat_fd(fd)?;
    if st.st_mode & libc::S_IFMT != libc::S_IFDIR || st.st_uid != uid || st.st_mode & 0o022 != 0 {
        return Err(Error::msg("real safely owned directory required"));
    }
    Ok(())
}

// CoreOS uses a native /root symlink. Resolve that native root-home alias once,
// then require real root-owned non-writable ancestors; never follow .ssh links.
pub fn enrollment_root_home() -> Result<String, Error> {
    let home = crate::pathx::eval_symlinks("/root")?;
    let mut current = home.clone();
    loop {
        let fd = open_dir_nofollow(&current)?;
        let result = enrollment_safe_directory(fd.as_raw_fd(), 0);
        drop(fd);
        result?;
        if current == "/" {
            break;
        }
        current = crate::pathx::dir(&current);
    }
    Ok(home)
}
