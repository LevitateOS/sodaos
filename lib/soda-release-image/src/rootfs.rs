//! `assemble.go`: rootfs bootstrap-chunk verification.

use std::fs::File;
use std::io::Read;

use sha2::{Digest, Sha256};

use crate::error::Error;
use crate::sys;

const CHUNK_LEN: u64 = 2 << 20;

/// VerifyRootfsChunks checks the bootstrap's native chunk list against the download.
pub fn verify_rootfs_chunks(path: &str, text: &str) -> Result<(), Error> {
    let body = match text.strip_prefix("stream-hash sha256 2097152\n") {
        Some(body) => body,
        None => return Err(Error::msg("unexpected native rootfs hash format")),
    };
    let file = File::open(path)?;
    let want: Vec<&str> = body.split_whitespace().collect();
    let mut index = 0usize;
    let mut reader = file;
    loop {
        let (next, more) = match_rootfs_chunk(&mut reader, &want, index)?;
        index = next;
        if !more {
            break;
        }
    }
    if index == 0 || index != want.len() {
        return Err(Error::msg("incomplete rootfs bootstrap hashes"));
    }
    Ok(())
}

fn match_rootfs_chunk(file: &mut File, want: &[&str], i: usize) -> Result<(usize, bool), Error> {
    let mut hasher = Sha256::new();
    let mut remaining = CHUNK_LEN;
    let mut chunk = [0u8; 8192];
    let mut total: u64 = 0;
    let mut eof = false;
    while remaining > 0 {
        let want_read = remaining.min(chunk.len() as u64) as usize;
        match file.read(&mut chunk[..want_read]) {
            Ok(0) => {
                eof = true;
                break;
            }
            Ok(n) => {
                hasher.update(&chunk[..n]);
                remaining -= n as u64;
                total += n as u64;
            }
            Err(e) => return Err(Error::from(e)),
        }
    }
    if total == 0 {
        return Ok((i, false));
    }
    if i >= want.len() || sys::hex_bytes(&hasher.finalize()) != want[i] {
        return Err(Error::msg("rootfs differs from native bootstrap hashes"));
    }
    Ok((i + 1, !eof))
}
