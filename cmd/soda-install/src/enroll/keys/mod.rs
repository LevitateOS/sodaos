//! Bounded authorized-keys import through validated open inodes: existing
//! files are appended, never replaced; new files are published by hard link.

mod authorized_keys;
mod directory;
#[cfg(test)]
mod tests;

use std::fs::File;
use std::os::unix::io::AsRawFd;

use crate::errors::{self, Error};
use crate::signal::Ctx;

pub use authorized_keys::append_enrollment_key_with_writer;
pub use directory::{enrollment_root_home, enrollment_safe_directory};

/// Single-syscall writer, like Go's `unix.Write`: no looping, so partial
/// writes surface for explicit uncertainty.
fn single_write(file: &File, data: &[u8]) -> std::io::Result<usize> {
    let n = unsafe {
        libc::write(
            file.as_raw_fd(),
            data.as_ptr() as *const libc::c_void,
            data.len(),
        )
    };
    if n < 0 {
        Err(std::io::Error::last_os_error())
    } else {
        Ok(n as usize)
    }
}

pub fn append_enrollment_key(ctx: &Ctx, home: &str, key: &str, uid: u32) -> Result<(), Error> {
    append_enrollment_key_with_writer(ctx, home, key, uid, &single_write)
}

pub fn random_hex(bytes: usize) -> Result<String, Error> {
    let mut random = vec![0u8; bytes];
    let mut filled = 0;
    while filled < bytes {
        let n = unsafe {
            libc::getrandom(
                random[filled..].as_mut_ptr() as *mut libc::c_void,
                (bytes - filled) as libc::size_t,
                0,
            )
        };
        if n < 0 {
            let errno = unsafe { *libc::__errno_location() };
            if errno == libc::EINTR {
                continue;
            }
            return Err(errors::os_error(std::io::Error::from_raw_os_error(errno)));
        }
        filled += n as usize;
    }
    Ok(crate::buildx::hex_encode(&random))
}
