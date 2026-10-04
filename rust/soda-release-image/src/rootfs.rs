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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_rootfs_chunks_match_and_mismatch() {
        // Oracle: Go TestMediaReadbackBindsNativeIgnitionAndRootfs (chunk half).
        let dir = std::env::temp_dir().join(format!("sri-rf-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("rootfs.img");
        let data = vec![0x5au8; 100];
        std::fs::write(&path, &data).unwrap();
        let hex = sys::hex_sha256(&data);
        let text = format!("stream-hash sha256 2097152\n{hex}\n");
        assert!(verify_rootfs_chunks(path.to_str().unwrap(), &text).is_ok());
        let bad = format!("stream-hash sha256 2097152\n{}\n", "0".repeat(64));
        assert_eq!(
            verify_rootfs_chunks(path.to_str().unwrap(), &bad)
                .unwrap_err()
                .0,
            "rootfs differs from native bootstrap hashes"
        );
        assert!(verify_rootfs_chunks(path.to_str().unwrap(), "bogus").is_err());
        let short = "stream-hash sha256 2097152\n".to_string();
        assert_eq!(
            verify_rootfs_chunks(path.to_str().unwrap(), &short)
                .unwrap_err()
                .0,
            "rootfs differs from native bootstrap hashes"
        );
        // Trailing declared hashes with no more bytes: incomplete.
        let extra = format!("stream-hash sha256 2097152\n{hex}\n{}\n", "0".repeat(64));
        assert_eq!(
            verify_rootfs_chunks(path.to_str().unwrap(), &extra)
                .unwrap_err()
                .0,
            "incomplete rootfs bootstrap hashes"
        );
        let _ = std::fs::remove_dir_all(&dir);
    }
}
