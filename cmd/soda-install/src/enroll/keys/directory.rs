use std::ffi::CString;
use std::fs::File;
use std::os::unix::io::{AsRawFd, FromRawFd, OwnedFd};

use crate::errors::{self, Error};

fn errno_error() -> Error {
    errors::os_error(std::io::Error::last_os_error())
}

fn path_cstr(path: &str) -> Result<CString, Error> {
    // Go's string-to-pointer conversion fails closed with EINVAL on NUL.
    CString::new(path)
        .map_err(|_| errors::os_error(std::io::Error::from_raw_os_error(libc::EINVAL)))
}

pub(crate) fn fstat_fd(fd: i32) -> Result<libc::stat, Error> {
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(fd, &mut st) } != 0 {
        return Err(errno_error());
    }
    Ok(st)
}

pub(crate) fn open_dir_nofollow(path: &str) -> Result<OwnedFd, Error> {
    let path = path_cstr(path)?;
    let fd = unsafe {
        libc::open(
            path.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if fd < 0 {
        return Err(errno_error());
    }
    Ok(unsafe { OwnedFd::from_raw_fd(fd) })
}

pub(crate) fn openat_file(dir_fd: i32, name: &str, flags: i32, mode: u32) -> std::io::Result<File> {
    let name = CString::new(name).map_err(|_| std::io::Error::from_raw_os_error(libc::EINVAL))?;
    let fd = unsafe { libc::openat(dir_fd, name.as_ptr(), flags | libc::O_CLOEXEC, mode) };
    if fd < 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(unsafe { File::from_raw_fd(fd) })
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
        unsafe {
            libc::flock(self.fd.as_raw_fd(), libc::LOCK_UN);
        }
    }
}

pub(crate) fn open_and_lock_ssh_directory(home: &str, uid: u32) -> Result<SshDir, Error> {
    let home_fd = open_dir_nofollow(home)?;
    enrollment_safe_directory(home_fd.as_raw_fd(), uid)?;
    let ssh = CString::new(".ssh").map_err(|_| Error::msg("invalid name"))?;
    if unsafe { libc::mkdirat(home_fd.as_raw_fd(), ssh.as_ptr(), 0o700) } != 0 {
        let errno = unsafe { *libc::__errno_location() };
        if errno != libc::EEXIST {
            return Err(errors::os_error(std::io::Error::from_raw_os_error(errno)));
        }
    }
    let dir_fd = unsafe {
        libc::openat(
            home_fd.as_raw_fd(),
            ssh.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if dir_fd < 0 {
        return Err(errno_error());
    }
    let dir = SshDir {
        fd: unsafe { OwnedFd::from_raw_fd(dir_fd) },
    };
    enrollment_safe_directory(dir.raw(), uid)?;
    if unsafe { libc::flock(dir.raw(), libc::LOCK_EX | libc::LOCK_NB) } != 0 {
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
