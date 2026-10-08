//! SHA-256 helpers used by the bounded OCI layout reader.

#[cfg(test)]
use sha2::Digest;
pub(super) use sha2::Sha256;

#[cfg(test)]
pub(super) fn sha256_hex(data: &[u8]) -> String {
    format!("{:x}", Sha256::digest(data))
}
