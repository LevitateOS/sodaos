// Broker error contract, extracted from wire.rs (A05.M).

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ErrorKind {
    Denied,
    Busy,
    Stale,
    Uncertain,
    NotFound,
    Internal,
}

#[derive(Debug)]
pub struct Error {
    kind: ErrorKind,
    message: String,
}

impl Error {
    pub fn denied(message: impl Into<String>) -> Error {
        Error {
            kind: ErrorKind::Denied,
            message: message.into(),
        }
    }
    pub fn busy() -> Error {
        Error {
            kind: ErrorKind::Busy,
            message: "subscription is in use".to_string(),
        }
    }
    pub fn stale() -> Error {
        Error {
            kind: ErrorKind::Stale,
            message: "identity generation changed".to_string(),
        }
    }
    pub fn uncertain() -> Error {
        Error {
            kind: ErrorKind::Uncertain,
            message: "subscription requires reconnection".to_string(),
        }
    }
    pub fn not_found() -> Error {
        Error {
            kind: ErrorKind::NotFound,
            message: "identity execution missing".to_string(),
        }
    }
    pub fn internal(message: impl Into<String>) -> Error {
        Error {
            kind: ErrorKind::Internal,
            message: message.into(),
        }
    }
    pub fn kind(&self) -> ErrorKind {
        self.kind
    }
    pub fn is_denied(&self) -> bool {
        self.kind == ErrorKind::Denied
    }
    pub fn is_not_found(&self) -> bool {
        self.kind == ErrorKind::NotFound
    }
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.message)
    }
}

impl std::error::Error for Error {}

impl From<serde_json::Error> for Error {
    fn from(err: serde_json::Error) -> Error {
        Error::internal(err.to_string())
    }
}

impl From<std::io::Error> for Error {
    fn from(err: std::io::Error) -> Error {
        Error::internal(err.to_string())
    }
}
