//! Rust port of `internal/release/deliver`: native Sigstore release and
//! channel semantics (admission, prepare, sign, fetch, publish, finalize).
//!
//! The Go package remains the owner of record; this crate mirrors its exact
//! validation behavior, error text, and JSON shapes. It never installs an
//! image, changes host trust, or reboots.

use sha2::{Digest as _, Sha256};

pub mod admission;
pub mod buildx;
pub mod check;
pub mod content;
pub mod document;
pub mod fetch;
pub mod finalize;
pub mod import;
pub(crate) mod json_serde;
pub mod model;
pub mod native;
pub mod oci;
pub mod payload;
pub mod prepare;
pub mod publish;

pub use model::{
    admit_channel, admit_release, empty_state, Channel, Highwater, Permit, Release, Seen, Trust,
};
pub use payload::{Image, Payload, NAMES};

/// Validation/transport failure; the message matches the Go owner exactly.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Error(pub String);

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.0)
    }
}

impl std::error::Error for Error {}

impl Error {
    pub fn msg(text: impl Into<String>) -> Error {
        Error(text.into())
    }
    /// `ErrRefused`: release authority or completeness refused.
    pub fn refused() -> Error {
        Error("release authority or completeness refused".to_string())
    }
    /// `ErrUnavailable`: transport failure; preserve attempt and observe.
    pub fn unavailable() -> Error {
        Error(
            "release transport unavailable; preserve attempt and observe before retrying publication"
                .to_string(),
        )
    }
    /// `errorAt`: operation-scoped refusal.
    pub fn at(operation: &str) -> Error {
        Error(format!(
            "{operation}: release authority or completeness refused"
        ))
    }
}

/// `Hash`: sha256 digest reference of bytes.
pub fn hash_bytes(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    format!("sha256:{:x}", hasher.finalize())
}

/// `Digest`: `sha256:`-prefixed digest shape.
pub fn is_digest_ref(s: &str) -> bool {
    s.len() == 7 + 64 && s.starts_with("sha256:") && soda_build_tools::reader::is_digest(&s[7..])
}

/// `channel`: known channel names.
pub fn is_channel(s: &str) -> bool {
    s == "candidate" || s == "preview" || s == "stable"
}

/// Current UTC time as a Unix timestamp (`nowUTC().Unix()`).
pub fn now_unix() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}
