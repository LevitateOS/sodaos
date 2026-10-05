//! Isolated-worker dispatch (Go `tools/soda-build` `worker_linux.go`):
//! config admission, worker argv construction, attempt runtime ownership,
//! and result validation. Sandbox execution and live-input resolution stay
//! behind the release-pipeline boundary in `build_cli`.

pub mod config;
pub mod execution;
pub mod runtime;

pub use config::*;
pub use execution::*;
pub use runtime::*;

#[cfg(test)]
mod tests;
