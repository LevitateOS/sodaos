//! Shared logic for the Rust ports of `soda-build`, `soda-candidate`,
//! and `soda-artifacts`. Each binary keeps the Go owner's exact flags,
//! usage text, refusal messages, and exit codes; only the heavy release
//! pipeline steps (live-input resolution, worker execution, image build,
//! OCI inspection, CoreOS fetch) remain behind explicit boundary errors
//! until their owning ports land.

pub mod artifacts;
pub mod build_cli;
pub mod build_spec;
pub mod candidate;
pub mod candidate_controller;
pub mod candidate_display;
pub mod candidate_fixture;
pub mod candidate_hints;
pub mod candidate_prompts;
pub mod digest;
pub mod exitcode;
pub mod goflag;
pub mod progress;
pub mod worker;
