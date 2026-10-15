//! `payload.go`: immutable appliance payload metadata.
//!
//! The canonical Payload v4 wire types and validation live in
//! soda-build-tools; this module keeps only deliver's I/O limits, trust
//! admission and the GHCR repository-policy helper. There is no v3 branch:
//! old documents fail validation instead of being migrated.

use crate::buildx::{read_json_at, Root};
use crate::Error;
pub use soda_build_tools::payload::{ImageBinding as Image, PayloadV4 as Payload, Role, FORMAT};

pub const PATH: &str = "/usr/share/soda/release.json";
pub const IMAGES_PATH: &str = "/usr/share/soda/images";

/// `ValidRepositoryPrefix`: `^ghcr\.io/[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9._-]*$`
/// with length below 200.
pub fn valid_repository_prefix(s: &str) -> bool {
    if s.len() >= 200 {
        return false;
    }
    let rest = match s.strip_prefix("ghcr.io/") {
        Some(rest) => rest,
        None => return false,
    };
    let (owner, repo) = match rest.split_once('/') {
        Some(pair) => pair,
        None => return false,
    };
    if repo.contains('/') {
        return false;
    }
    let mut owner_chars = owner.bytes();
    match owner_chars.next() {
        Some(b @ (b'a'..=b'z' | b'0'..=b'9')) => {
            let _ = b;
        }
        _ => return false,
    }
    if !owner_chars.all(|b| matches!(b, b'a'..=b'z' | b'0'..=b'9' | b'-')) {
        return false;
    }
    let mut repo_chars = repo.bytes();
    match repo_chars.next() {
        Some(b @ (b'a'..=b'z' | b'0'..=b'9')) => {
            let _ = b;
        }
        _ => return false,
    }
    repo_chars.all(|b| matches!(b, b'a'..=b'z' | b'0'..=b'9' | b'.' | b'_' | b'-'))
}

/// `Load`: read and validate a payload file.
pub fn load(path: &str) -> Result<Payload, Error> {
    let parent = match path.rfind('/') {
        Some(0) | None => "/".to_string(),
        Some(i) => path[..i].to_string(),
    };
    let base = match path.rfind('/') {
        Some(i) => &path[i + 1..],
        None => path,
    };
    let root = Root::open(&parent)?;
    let (payload, _) = read_json_at(&root, base)?;
    payload.validate().map_err(|e| Error::msg(e.to_string()))?;
    Ok(payload)
}

#[cfg(test)]
mod tests;
