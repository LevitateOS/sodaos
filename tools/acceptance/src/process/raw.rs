//! Raw bounded capture on the owned process lifecycle, for callers that
//! need machine stdout — like the project-state snapshot probe.
//! RedactingWriter is file-backed and redacted, which would change those
//! bytes; raw capture keeps bounded in-memory bytes with no transformation
//! while group retirement, reaping, and pump joining stay owned.

use std::sync::{Arc, Mutex};

use super::launch::{lock, start_inner, PumpSink};
use super::owned_process::Process;
use super::Phase;
use crate::command::CommandSpec;
use crate::error::Error;

#[derive(Debug, Default)]
struct RawState {
    bytes: Vec<u8>,
    total: usize,
    cap: usize,
}

/// Shared raw byte sink for pump threads. Bytes past `cap` are counted
/// but discarded, so capture stays bounded while the pipe keeps draining.
#[derive(Debug, Clone)]
pub struct RawCapture {
    state: Arc<Mutex<RawState>>,
}

impl RawCapture {
    pub fn new(cap: usize) -> RawCapture {
        RawCapture {
            state: Arc::new(Mutex::new(RawState {
                bytes: Vec::new(),
                total: 0,
                cap,
            })),
        }
    }

    /// Captured bytes (at most the cap) and whether the stream exceeded it.
    /// Call after [`Process::join_pumps`] so no writer is still pumping.
    pub fn take(&self) -> (Vec<u8>, bool) {
        let mut state = lock(&self.state);
        (std::mem::take(&mut state.bytes), state.total > state.cap)
    }
}

impl PumpSink for RawState {
    fn pump_write(&mut self, bytes: &[u8]) -> Result<(), String> {
        self.total += bytes.len();
        if self.bytes.len() < self.cap {
            let room = self.cap - self.bytes.len();
            self.bytes
                .extend_from_slice(&bytes[..bytes.len().min(room)]);
        }
        Ok(())
    }
}

/// Start an owned process capturing raw stdout/stderr bytes, like
/// [`super::start_process`] but with in-memory bounded sinks instead of
/// redacting writers. The stderr sink discards everything.
pub fn start_raw_process(
    phase: &Phase,
    spec: &CommandSpec,
    cap: usize,
) -> Result<(Arc<Process>, RawCapture, RawCapture), Error> {
    phase.check()?;
    let out = RawCapture::new(cap);
    let err = RawCapture::new(0);
    let process = start_inner(spec, out.state.clone(), err.state.clone())?;
    Ok((process, out, err))
}
