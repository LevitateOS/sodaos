//! Shared logic for the Rust ports of `soda-build`, `soda-candidate`,
//! `soda-artifacts`, and `soda-candidate-check`. Each binary keeps the Go
//! owner's exact flags, usage text, refusal messages, and exit codes; the
//! heavy release pipeline steps (live-input resolution, worker execution,
//! image build, OCI inspection, CoreOS fetch) delegate to the release
//! build/deliver/image crates.

pub mod artifacts;
pub mod build_cli;
pub mod build_spec;
pub mod candidate;
pub mod candidate_controller;
pub mod candidate_display;
pub mod candidate_fixture;
pub mod candidate_hints;
pub mod candidate_prompts;
pub mod check_cli;
pub mod digest;
pub mod exitcode;
pub mod goflag;
pub mod pipeline;
pub mod progress;
pub mod worker;
