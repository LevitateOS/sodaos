//! SHA-256 helpers used by the bounded OCI layout reader.

#[cfg(test)]
use sha2::Digest;
pub(super) use sha2::Sha256;

pub(super) fn hex_lower(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(bytes.len() * 2);
    for &byte in bytes {
        out.push(HEX[(byte >> 4) as usize] as char);
        out.push(HEX[(byte & 0x0f) as usize] as char);
    }
    out
}

#[cfg(test)]
pub(super) fn sha256_hex(data: &[u8]) -> String {
    hex_lower(&Sha256::digest(data))
}
