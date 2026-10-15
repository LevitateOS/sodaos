//! Test-only SHA-256 helpers pinning the importer's digest behavior.

#[cfg(test)]
use sha2::Digest;
#[cfg(test)]
pub(super) use sha2::Sha256;

#[cfg(test)]
pub(super) fn sha256_hex(data: &[u8]) -> String {
    format!("{:x}", Sha256::digest(data))
}
