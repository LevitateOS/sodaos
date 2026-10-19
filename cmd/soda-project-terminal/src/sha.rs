//! SHA-256 helpers for key revisions and approved setup inputs.

pub use sha2::{Digest, Sha256};

pub fn digest(data: &[u8]) -> [u8; 32] {
    Sha256::digest(data).into()
}

pub fn hex(data: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(data.len() * 2);
    for &byte in data {
        out.push(HEX[(byte >> 4) as usize] as char);
        out.push(HEX[(byte & 0x0f) as usize] as char);
    }
    out
}

pub fn hex_digest(data: &[u8]) -> String {
    hex(&digest(data))
}
