//! Shared logic for the Rust ports of `soda-build`, `soda-candidate`,
//! `soda-artifacts`, and `soda-candidate-check`. Each binary owns its Clap
//! command schema and Soda-specific refusal gates; the heavy release pipeline
//! steps (live-input resolution, worker execution,
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
pub mod pipeline;
pub mod progress;
pub mod worker;
