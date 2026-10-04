//! Installer error taxonomy: exact Go messages plus the identities the
//! control flow matches on (cancel, restart, quit, command exit).

use std::fmt;

use crate::fmtx::{sprintf, Arg};

/// All installer failures. Messages match the Go errors byte for byte.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Error {
    /// Plain message error (`errors.New` / `fmt.Errorf` without wrapping).
    Msg(String),
    /// `context.Canceled`: `"context canceled"`.
    Canceled,
    /// `context.DeadlineExceeded`: `"context deadline exceeded"`.
    DeadlineExceeded,
    /// `errBack` sentinel: `"back requested"`.
    Back,
    /// `errRestart` sentinel: `"restart requested"`.
    Restart,
    /// `errCancel` sentinel: `"cancel requested"`.
    Cancel,
    /// `commandExit`: failed child with exit code and interrupt flag.
    CmdExit {
        name: String,
        code: i32,
        interrupted: bool,
    },
    /// `errEnrollmentWriteUncertain`: the authorized-key write may have
    /// completed but could not be confirmed.
    EnrollUncertain,
}

impl Error {
    pub fn msg(text: impl Into<String>) -> Error {
        Error::Msg(text.into())
    }
}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Error::Msg(text) => write!(f, "{text}"),
            Error::Canceled => write!(f, "context canceled"),
            Error::DeadlineExceeded => write!(f, "context deadline exceeded"),
            Error::Back => write!(f, "back requested"),
            Error::Restart => write!(f, "restart requested"),
            Error::Cancel => write!(f, "cancel requested"),
            Error::CmdExit { name, code, interrupted } => write!(
                f,
                "{}",
                sprintf(
                    "%s failed (exit %d, interrupted %t); raw diagnostics suppressed",
                    &[Arg::Str(name), Arg::Int(*code as i64), Arg::Bool(*interrupted)]
                )
            ),
            Error::EnrollUncertain => write!(
                f,
                "authorized-key import may have completed; the file changed or the write could not be confirmed; inspect native access before another import"
            ),
        }
    }
}

impl std::error::Error for Error {}

/// Go `syscall.Errno` text on Linux: the C library message with a lowercase
/// initial, `errno %d` for unknown codes.
pub fn errno_text(errno: i32) -> String {
    let text = unsafe {
        let raw = libc::strerror(errno);
        if raw.is_null() {
            return format!("errno {errno}");
        }
        String::from_utf8_lossy(std::ffi::CStr::from_ptr(raw).to_bytes()).into_owned()
    };
    if text.starts_with("Unknown error") {
        return format!("errno {errno}");
    }
    let mut chars = text.chars();
    match chars.next() {
        Some(first) => first.to_lowercase().collect::<String>() + chars.as_str(),
        None => text,
    }
}

/// Raw OS error, like Go returning a `syscall.Errno` unwrapped.
pub fn os_error(err: std::io::Error) -> Error {
    match err.raw_os_error() {
        Some(errno) => Error::msg(errno_text(errno)),
        None => Error::msg(err.to_string()),
    }
}

/// Go `*PathError`: `op path: errno text`.
pub fn path_error(op: &str, path: &str, err: std::io::Error) -> Error {
    let text = match err.raw_os_error() {
        Some(errno) => errno_text(errno),
        None => err.to_string(),
    };
    Error::msg(format!("{op} {path}: {text}"))
}

/// Go `*LinkError`: `op old new: errno text`.
pub fn link_error(op: &str, old: &str, new: &str, err: std::io::Error) -> Error {
    let text = match err.raw_os_error() {
        Some(errno) => errno_text(errno),
        None => err.to_string(),
    };
    Error::msg(format!("{op} {old} {new}: {text}"))
}
