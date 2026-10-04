//! Foundation crate for SodaOS Rust build tools (lane R).
//!
//! PR03 establishes the offline Rust build (workspace, pinned toolchain,
//! vendored sources). Later lane-R PRs add the asset-script and
//! release/build ports on top of this crate. Dependency-free by policy:
//! the tree must build with zero network.

#[cfg(test)]
mod tests {
    #[test]
    fn package_metadata_present() {
        assert_eq!(env!("CARGO_PKG_NAME"), "soda-build-tools");
        assert!(!env!("CARGO_PKG_VERSION").is_empty());
    }
}
