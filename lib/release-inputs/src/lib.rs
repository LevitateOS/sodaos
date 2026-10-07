//! Foundation crate for SodaOS Rust build tools (lane R).
//!
//! Read-only release input validators and optional typed trust-key admission.
//! Dependencies resolve through the workspace lock and are cached for offline
//! builds; the `trust-key` feature enables the shared P-256 admission adapter.

pub mod elf;
pub mod reader;

#[cfg(feature = "trust-key")]
pub mod trust_key;

#[cfg(test)]
mod tests {
    #[test]
    fn package_metadata_present() {
        assert_eq!(env!("CARGO_PKG_NAME"), "soda-build-tools");
        assert!(!env!("CARGO_PKG_VERSION").is_empty());
    }
}
