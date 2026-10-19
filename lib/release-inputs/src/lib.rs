//! Foundation crate for SodaOS Rust build tools (lane R).
//!
//! Read-only release input validators and optional typed trust-key admission.
//! Dependencies resolve through the workspace lock and are cached for offline
//! builds; the `trust-key` feature enables the shared P-256 admission adapter.

/// Complete released rootfs ceiling: retain 128 MiB below the kernel's 2 GiB limit.
pub const MAX_ROOTFS_BYTES: u64 = (2u64 << 30) - (128u64 << 20);

pub fn valid_rootfs_size(bytes: u64) -> bool {
    bytes > 0 && bytes <= MAX_ROOTFS_BYTES
}

pub mod elf;
pub mod payload;
pub mod project_support;
pub mod reader;

#[cfg(feature = "trust-key")]
pub mod trust_key;
