//! SHA-256 recipe for approved factory-role inputs.

use sha2::{Digest, Sha256};

pub fn hex(data: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(data.len() * 2);
    for &byte in data {
        out.push(HEX[(byte >> 4) as usize] as char);
        out.push(HEX[(byte & 0x0f) as usize] as char);
    }
    out
}

/// Canonical approved-inputs digest: names sorted ascending, then
/// `name + NUL + contents` for each file, lowercase hex.
pub fn approved_digest(files: &[(String, Vec<u8>)]) -> String {
    let mut ordered: Vec<&(String, Vec<u8>)> = files.iter().collect();
    ordered.sort_by(|a, b| a.0.cmp(&b.0));
    let mut hasher = Sha256::new();
    for (name, contents) in ordered {
        hasher.update(name.as_bytes());
        hasher.update([0]);
        hasher.update(contents);
    }
    hex(&hasher.finalize())
}

#[cfg(test)]
#[path = "sha_tests.rs"]
mod tests;
