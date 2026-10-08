//! Read-only release/build validators (lane R, PR04 release-reader).
//!
//! Mostly dependency-free port of the pure read/validate surface of
//! `internal/release/build`: architecture names, digest shapes, strict HTTPS
//! metadata URLs, GnuPG status parsing, recipe/unit-file readers, and the
//! CoreOS/Tailnet/Muse/Forgejo metadata shapes. It performs no network
//! fetches, builds, or writes beyond reading the caller's files, and every
//! error message matches the Go owner byte for byte.
//!
//! The Go package remains the broader release-domain owner. The selected
//! `stream` records are the canonical Rust wire and validation owner for the
//! controller-resolved live-input handoff consumed by the isolated build.

pub mod forgejo;
pub mod image;
pub mod muse;
pub mod settings;
pub mod signature;
pub mod stream;
pub mod url;

pub use image::Image;

/// Validation failure; the message matches the Go owner exactly.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Error(pub String);

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.0)
    }
}

impl std::error::Error for Error {}

/// Maps a Soda architecture name to its OCI form (`files.go`).
pub fn oci_architecture(arch: &str) -> Result<&'static str, Error> {
    if arch == "x86_64" {
        Ok("amd64")
    } else {
        Err(Error("expected x86_64".to_string()))
    }
}

/// Lowercase hex SHA-256 digest shape (`files.go Digest`).
pub fn is_digest(s: &str) -> bool {
    s.len() == 64 && s.bytes().all(|b| matches!(b, b'0'..=b'9' | b'a'..=b'f'))
}

/// Lowercase hex 40-byte revision shape (`files.go Revision`).
pub fn is_revision(s: &str) -> bool {
    s.len() == 40 && s.bytes().all(|b| matches!(b, b'0'..=b'9' | b'a'..=b'f'))
}

fn non_empty_digits(s: &str) -> bool {
    !s.is_empty() && s.bytes().all(|b| b.is_ascii_digit())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn architecture_names() {
        assert_eq!(oci_architecture("x86_64").unwrap(), "amd64");
        assert_eq!(
            oci_architecture("aarch64").unwrap_err(),
            Error("expected x86_64".to_string())
        );
        assert_eq!(
            oci_architecture("amd64").unwrap_err(),
            Error("expected x86_64".to_string())
        );
    }

    #[test]
    fn digest_and_revision_shapes() {
        assert!(is_digest(&"a".repeat(64)));
        assert!(is_digest(&"0123456789abcdef".repeat(4)));
        assert!(!is_digest(&"a".repeat(63)));
        assert!(!is_digest(&"a".repeat(65)));
        assert!(!is_digest(&"A".repeat(64)));
        assert!(!is_digest(&"g".repeat(64)));
        assert!(!is_digest(""));
        assert!(is_revision(&"b".repeat(40)));
        assert!(!is_revision(&"b".repeat(39)));
        assert!(!is_revision(&"B".repeat(40)));
        assert!(!is_revision(&"z".repeat(40)));
    }
}
