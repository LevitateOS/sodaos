//! Build-time release asset tooling (C09 consolidation).
//!
//! One package with seven binaries, folded from `soda-asset-fetchers`
//! (fetch), `soda-forgejo-locales` (locales) and `soda-stage-render`
//! (render). Binary names, outputs, modes, messages and exit codes are
//! unchanged; only the package identity and module paths moved.
//!
//! Shared helpers stay per-namespace byte-for-byte (see each `mod.rs`);
//! nothing is deduplicated across namespaces.

pub mod fetch;
pub mod locales;
pub mod render;
