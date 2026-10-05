//! Single failure type. Every operational failure surfaces externally as the
//! fixed stderr line plus exit 1 (the `.py` `sys.exit(...)` arm); the inner
//! message exists for unit tests only and is never printed.

/// Byte-identical to the `.py` `sys.exit(...)` payload (plus `\n`).
pub const FIXED_MESSAGE: &str =
    "factory preparation unconfirmed; inspect native state and managed files";

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Error {
    /// The `.py` `fail(msg)` / `ValueError` path.
    Fail(String),
    /// OS / subprocess failure (`.py` `OSError` / `SubprocessError` paths).
    Io(String),
    /// `write_new` target already exists (`.py` `FileExistsError`).
    Exists,
    /// `read_json` target absent (`.py` `FileNotFoundError`).
    Missing,
}

impl Error {
    pub fn fail(message: impl Into<String>) -> Error {
        Error::Fail(message.into())
    }

    pub fn io(what: &str, err: &std::io::Error) -> Error {
        Error::Io(format!("{what}: {err}"))
    }

    pub fn io_msg(message: impl Into<String>) -> Error {
        Error::Io(message.into())
    }

    /// Classify an `std::io::Error` the way the `.py` distinguishes
    /// `FileNotFoundError` / `FileExistsError` from other `OSError`s.
    /// Symlink-vs-`O_NOFOLLOW` (`ELOOP`) stays a plain IO error, like the
    /// `.py` where it is not a `FileNotFoundError`.
    pub fn classify(err: std::io::Error) -> Error {
        match err.kind() {
            std::io::ErrorKind::NotFound => Error::Missing,
            std::io::ErrorKind::AlreadyExists => Error::Exists,
            _ => Error::Io(err.to_string()),
        }
    }
}

impl From<std::io::Error> for Error {
    fn from(err: std::io::Error) -> Error {
        Error::classify(err)
    }
}

/// The `.py` `fail(message)`: always raises, never returns a value.
pub fn fail<T>(message: impl Into<String>) -> Result<T, Error> {
    Err(Error::fail(message))
}
