//! SHA-256 helpers for acceptance evidence and file fingerprints.

pub use sha2::{Digest, Sha256};

pub fn digest(data: &[u8]) -> [u8; 32] {
    Sha256::digest(data).into()
}

/// Lowercase hex, matching Go's `hex.EncodeToString`.
pub fn hex_lower(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(bytes.len() * 2);
    for &byte in bytes {
        out.push(HEX[(byte >> 4) as usize] as char);
        out.push(HEX[(byte & 0x0f) as usize] as char);
    }
    out
}
