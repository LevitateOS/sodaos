//! Private exclusive evidence with streaming redaction, mirroring
//! `internal/acceptance/evidence.go`.
//!
//! An [`Evidence`] root is a fresh private directory held open; every entry
//! is created exclusively and never overwritten. Streamed bytes pass through
//! [`RedactingWriter`], which withholds trailing matches across writes so a
//! split secret never reaches the file, then strips URL queries/fragments.
//! Structured values are scrubbed before encoding, and
//! [`Evidence::check_secrets`] re-scans every retained byte as defense in
//! depth.

use crate::files::OwnedDir;

mod redaction;
mod store;

pub use redaction::{redact_urls, RedactingWriter};
pub use store::create_evidence;

/// Structured/streamed evidence size bound: 16 MiB of input bytes.
pub const EVIDENCE_LIMIT: u64 = 16 << 20;

/// Private exclusive evidence root with a redaction secret set.
pub struct Evidence {
    root: OwnedDir,
    secrets: Vec<Vec<u8>>,
}

fn contains_slice(haystack: &[u8], needle: &[u8]) -> bool {
    if needle.is_empty() || needle.len() > haystack.len() {
        return false;
    }
    haystack.windows(needle.len()).any(|w| w == needle)
}

fn longest_secret(secrets: &[Vec<u8>]) -> usize {
    secrets.iter().map(|s| s.len()).max().unwrap_or(0).max(1)
}

#[cfg(test)]
mod tests;
