//! Shared supervised-run lifecycle for factory CLI families.
//!
//! The codex and muse runners admit different providers, scopes and
//! guests, but drive the same boundary: unit wait plus exit/output
//! files, broker attestation, idempotent stop with pid-change
//! re-retire, bounded credential echo, live invocation match, and
//! windowed stdout slices. Those control flows live here, once, as
//! family-parameterized cores; `tcodex` and `tmuse` keep only their
//! admission policy (binding checks, path structs, staging) and
//! delegate. Every core preserves the exact podman/unit call sequence
//! the family FakeExec suites pin, so either side fails loudly on drift.

// `muse_serve_oracle` compiles this module through a private `#[path]` copy
// that never touches `RunPathsFn`; it serves the real library.
#[allow(unused_imports)]
pub use super::binding::{checked_binding_paths, RunPathsFn};
pub use super::output::{check_output_range, FactoryOutputSlice};
