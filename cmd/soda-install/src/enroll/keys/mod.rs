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
    random_hex_with(bytes, |buffer| {
        getrandom::fill(buffer).map_err(std::io::Error::other)
    })
}

fn random_hex_with(
    bytes: usize,
    mut fill: impl FnMut(&mut [u8]) -> std::io::Result<()>,
) -> Result<String, Error> {
    let mut random = vec![0u8; bytes];
    fill(&mut random).map_err(errors::os_error)?;
    Ok(crate::buildx::hex_encode(&random))
}
