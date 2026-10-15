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

#[cfg(test)]
mod tests {
    #[test]
    fn rootfs_retains_the_exact_kernel_margin() {
        assert_eq!(super::MAX_ROOTFS_BYTES, 2_013_265_920);
        assert!(super::valid_rootfs_size(2_013_265_920));
        assert!(!super::valid_rootfs_size(2_013_265_921));
        assert!(!super::valid_rootfs_size(0));
    }
    #[test]
    fn package_metadata_present() {
        assert_eq!(env!("CARGO_PKG_NAME"), "soda-build-tools");
        assert!(!env!("CARGO_PKG_VERSION").is_empty());
    }
}
