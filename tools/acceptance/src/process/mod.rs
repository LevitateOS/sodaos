//! Owned process-group execution, mirroring `process.go`.
//!
//! A started child leads its own process group. A reaper thread observes
//! the leader's exit without reaping it (`waitid` with `WNOWAIT`), then
//! terminates any surviving group members before reaping, so an exited
//! parent can neither leave descendants behind nor let cleanup signal a
//! reused PID. Cancellation and deadlines arrive through [`Phase`], the
//! port of Go's `context.Context` for this crate.

use std::sync::{Arc, Mutex};

use crate::error::Error;
use crate::evidence::RedactingWriter;

mod launch;
mod owned_process;
mod phase;

pub use self::launch::start_process;
pub use self::owned_process::Process;
pub use self::phase::Phase;

/// Shared redacting sink for pump threads.
pub type SharedWriter = Arc<Mutex<RedactingWriter>>;

/// Reaped outcome: exit code plus separated cleanup/wait failures.
#[derive(Debug, Clone)]
pub struct ProcessOutcome {
    /// Exit code, or `-1` for signal termination, like Go's `ExitCode`.
    pub exit_code: Option<i32>,
    /// Wait failure message (non-zero exit or signal).
    pub wait_message: Option<String>,
    /// Group cleanup failure message.
    pub cleanup_message: Option<String>,
}

impl ProcessOutcome {
    /// Combined failure like Go's joined process error.
    pub fn combined(&self) -> Option<Error> {
        Error::join(vec![
            self.cleanup_message.clone().map(Error::msg),
            self.wait_message.clone().map(Error::msg),
        ])
    }
}

#[cfg(test)]
mod tests;
