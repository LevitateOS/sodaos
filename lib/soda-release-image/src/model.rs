//! Exact mirrors of the foreign `release/build` + `release/deliver` model
//! shapes the pipeline decodes, validates, and emits: payloads, candidates,
//! trust, permits, images, CoreOS inputs. JSON field names and emission
//! order match the Go owners; validators return the same messages.

use crate::error::Error;
use crate::sys;

mod candidate;
mod payload;
mod trust;
mod url;

pub use candidate::{valid_candidate_content, Candidate, ForgejoToolchain};
pub use soda_build_tools::reader::Image;

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProducedImage {
    pub manifest: String,
    pub config: String,
    pub archive_sha256: String,
}
pub use payload::{Payload, PayloadImage};
pub use trust::{Permit, SecretFiles, Trust};
pub use url::{https_url, is_loopback_addr, parse_url, UrlParts};

pub const DELIVER_PATH: &str = "/usr/share/soda/release.json";
pub const DELIVER_IMAGES_PATH: &str = "/usr/share/soda/images";
/// `deliver.Names`, in order.
pub const NAMES: [&str; 6] = [
    "dashboard",
    "forgejo",
    "extension",
    "proxy",
    "project-os",
    "tailnet",
];
pub const FORGEJO_COMPILER_IMAGE: &str =
    "docker.io/library/golang@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468";
/// `store.SchemaVersion()`.
pub const SCHEMA_VERSION: i64 = 28;
pub const SODA_SOURCE: &str = "https://github.com/LevitateOS/sodaos";

pub fn is_revision(s: &str) -> bool {
    soda_build_tools::reader::is_revision(s)
}

pub fn is_digest(s: &str) -> bool {
    soda_build_tools::reader::is_digest(s)
}

pub fn oci_architecture(arch: &str) -> Result<&'static str, Error> {
    soda_build_tools::reader::oci_architecture(arch).map_err(|e| Error::msg(e.to_string()))
}

/// `deliver.Digest`: `sha256:`-prefixed hex digest.
pub fn prefixed_digest(s: &str) -> bool {
    match s.strip_prefix("sha256:") {
        Some(hex) => is_digest(hex),
        None => false,
    }
}

/// `deliver.Hash`: `sha256:`-prefixed hex of bytes.
pub fn deliver_hash(data: &[u8]) -> String {
    format!("sha256:{}", sys::hex_sha256(data))
}

fn is_dotted_numbers(value: &str, parts: usize) -> bool {
    let pieces: Vec<&str> = value.split('.').collect();
    pieces.len() == parts
        && pieces
            .iter()
            .all(|p| !p.is_empty() && p.bytes().all(|b| b.is_ascii_digit()))
}

pub fn is_coreos_release(value: &str) -> bool {
    is_dotted_numbers(value, 4)
}

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
    let owner_ok = !owner.is_empty()
        && owner
            .bytes()
            .next()
            .is_some_and(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        && owner
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-');
    let repo_ok = !repo.is_empty()
        && repo
            .bytes()
            .next()
            .is_some_and(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        && repo.bytes().all(|b| {
            b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'.' || b == b'_' || b == b'-'
        });
    owner_ok && repo_ok
}

fn content_get(files: &[(String, String)], name: &str) -> String {
    files
        .iter()
        .find(|(k, _)| k == name)
        .map(|(_, v)| v.clone())
        .unwrap_or_default()
}

#[cfg(test)]
mod tests;
