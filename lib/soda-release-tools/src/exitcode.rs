//! Exit-code mapping (Go `internal/release/build` `BuildExitCode`).

use std::sync::atomic::{AtomicI32, Ordering};

/// Signal-interrupt carrier: `build interrupted`, exiting 128 + signal.
#[derive(Debug, Clone, Copy)]
pub struct Interrupted(pub i32);

impl std::fmt::Display for Interrupted {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "build interrupted")
    }
}

impl Interrupted {
    pub fn exit_code(self) -> i32 {
        self.0
    }
}

static INTERRUPT_CODE: AtomicI32 = AtomicI32::new(0);

pub fn note_interrupt(code: i32) {
    INTERRUPT_CODE.store(code, Ordering::SeqCst);
}

pub fn take_interrupt() -> Option<Interrupted> {
    match INTERRUPT_CODE.swap(0, Ordering::SeqCst) {
        0 => None,
        code => Some(Interrupted(code)),
    }
}

/// Non-consuming interrupt observation for live cancellation predicates.
/// Unlike take_interrupt, the outer CLI still owns reporting afterwards.
pub fn has_interrupt() -> bool {
    INTERRUPT_CODE.load(Ordering::SeqCst) != 0
}

#[cfg(test)]
static INTERRUPT_TEST_LOCK: std::sync::Mutex<()> = std::sync::Mutex::new(());

/// Serializes tests that mutate the global interrupt flag.
#[cfg(test)]
pub(crate) fn interrupt_test_lock() -> std::sync::MutexGuard<'static, ()> {
    INTERRUPT_TEST_LOCK.lock().unwrap()
}

/// Port of `BuildExitCode`: typed exit codes win, cancellation maps to
/// 130, everything else is 1.
pub fn build_exit_code(err: Option<&ToolError>) -> i32 {
    match err {
        None => 0,
        Some(ToolError::Interrupted(i)) => {
            if i.exit_code() > 0 {
                i.exit_code()
            } else {
                1
            }
        }
        Some(ToolError::ExitCode(code)) => {
            if *code > 0 {
                *code
            } else {
                1
            }
        }
        Some(ToolError::Cancelled) => 130,
        Some(ToolError::Message(_)) => 1,
    }
}

/// CLI error with an optional exact exit code.
#[derive(Debug, Clone)]
pub enum ToolError {
    Message(String),
    ExitCode(i32),
    Interrupted(Interrupted),
    Cancelled,
}

impl std::fmt::Display for ToolError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ToolError::Message(msg) => write!(f, "{msg}"),
            ToolError::ExitCode(code) => write!(f, "controller exited {code}"),
            ToolError::Interrupted(i) => write!(f, "{i}"),
            ToolError::Cancelled => write!(f, "build interrupted"),
        }
    }
}

impl ToolError {
    pub fn msg(text: impl Into<String>) -> ToolError {
        ToolError::Message(text.into())
    }
}
