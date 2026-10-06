//! Error type: plain messages matching the Go owner byte for byte.

/// Pipeline failure; the message matches the Go owner exactly.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Error(pub String);

impl Error {
    pub fn msg(text: impl Into<String>) -> Error {
        Error(text.into())
    }
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.0)
    }
}

impl std::error::Error for Error {}

impl From<std::io::Error> for Error {
    fn from(err: std::io::Error) -> Error {
        Error(err.to_string())
    }
}

impl From<soda_json::ParseError> for Error {
    fn from(_: soda_json::ParseError) -> Error {
        Error("invalid JSON".to_string())
    }
}

/// Join two fallible closes the way Go's `errors.Join` does for log files:
/// both run, and either message survives.
pub fn join_close(first: Result<(), Error>, second: Result<(), Error>) -> Result<(), Error> {
    match (first, second) {
        (Ok(()), Ok(())) => Ok(()),
        (Err(e), Ok(())) | (Ok(()), Err(e)) => Err(e),
        (Err(a), Err(b)) => Err(Error(format!("{}\n{}", a.0, b.0))),
    }
}
