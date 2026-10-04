//! Crate error: exact messages, OS-kind preservation, redaction-safe chains.
//!
//! Messages match the Go owner byte for byte wherever they reach evidence
//! (`failure.txt`, observation records). I/O errors keep their
//! [`std::io::ErrorKind`] so exclusivity checks (`AlreadyExists`) and
//! absence checks (`NotFound`) behave like the Go `os.ErrExist` /
//! `os.IsNotExist` gates. [`Error::redacted`] mirrors Go's `safeError`: the
//! display text is scrubbed while the source chain stays intact for
//! [`Error::is_cancelled`] and kind inspection.

use std::fmt;
use std::io;

/// Crate error.
#[derive(Debug)]
pub enum Error {
    /// OS error with its kind preserved.
    Io(io::Error),
    /// Cancelled phase context. Displays as `context canceled` so retained
    /// `failure.txt` records match the Go owner exactly.
    Cancelled,
    /// Plain message with an optional cause chain.
    Msg {
        message: String,
        source: Option<Box<Error>>,
    },
}

impl Error {
    /// Plain message with no cause.
    pub fn msg(message: impl Into<String>) -> Error {
        Error::Msg {
            message: message.into(),
            source: None,
        }
    }

    /// Scrubbed display text keeping the original chain, like Go's
    /// `Evidence.RedactError`.
    pub fn redacted(message: String, cause: Error) -> Error {
        Error::Msg {
            message,
            source: Some(Box::new(cause)),
        }
    }

    /// Chained message: `message: cause`, like Go's `%w` wrapping.
    pub fn wrap(message: impl Into<String>, cause: Error) -> Error {
        Error::Msg {
            message: format!("{}: {}", message.into(), cause),
            source: Some(Box::new(cause)),
        }
    }

    /// True when this error or any cause is [`Error::Cancelled`].
    pub fn is_cancelled(&self) -> bool {
        match self {
            Error::Cancelled => true,
            Error::Io(_) => false,
            Error::Msg { source, .. } => source.as_ref().is_some_and(|e| e.is_cancelled()),
        }
    }

    /// The I/O kind when this error or its outermost I/O cause has one.
    pub fn io_kind(&self) -> Option<io::ErrorKind> {
        match self {
            Error::Io(e) => Some(e.kind()),
            Error::Cancelled => None,
            Error::Msg { source, .. } => source.as_ref().and_then(|e| e.io_kind()),
        }
    }
}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Error::Io(e) => write!(f, "{e}"),
            Error::Cancelled => f.write_str("context canceled"),
            Error::Msg { message, .. } => f.write_str(message),
        }
    }
}

impl std::error::Error for Error {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Error::Io(e) => Some(e),
            Error::Cancelled => None,
            Error::Msg { source, .. } => source
                .as_ref()
                .map(|e| e as &(dyn std::error::Error + 'static)),
        }
    }
}

impl From<io::Error> for Error {
    fn from(error: io::Error) -> Error {
        Error::Io(error)
    }
}
