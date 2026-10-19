//! SHA-256 helpers for pinned binary checks and provider identifiers.

use sha2::{Digest, Sha256};
use std::io::{self, Read};

pub fn digest(data: &[u8]) -> [u8; 32] {
    Sha256::digest(data).into()
}

pub fn digest_reader(reader: &mut dyn Read) -> io::Result<[u8; 32]> {
    let mut hasher = Sha256::new();
    let mut buf = [0u8; 8192];
    loop {
        let n = reader.read(&mut buf)?;
        if n == 0 {
            return Ok(hasher.finalize().into());
        }
        hasher.update(&buf[..n]);
    }
}

pub fn hex(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(bytes.len() * 2);
    for &byte in bytes {
        out.push(HEX[(byte >> 4) as usize] as char);
        out.push(HEX[(byte & 0x0f) as usize] as char);
    }
    out
}
