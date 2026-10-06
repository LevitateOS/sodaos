//! Provisioning renderer (ports `scripts/render-provisioning.py`).
//!
//! Merges the public Butane bootstrap with private per-instance operator
//! inputs. No conversion, installation, account enrollment or reboot is
//! implicit. Rendered documents, secret handling and failure messages
//! match the script; only the argparse envelope carries the new name.

mod document;
mod private_files;
mod render;

#[cfg(test)]
mod tests;

use std::io::ErrorKind;

pub use render::{render, RenderInputs};

#[cfg(test)]
use document::dump_python;
#[cfg(test)]
use private_files::{is_appliance_hostname, is_fixture_hostname};

/// Python exception class names the script's single `except` clause
/// reports; the bin formats the exact failure line from the kind.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProvKind {
    Value,
    Key,
    Type,
    Attribute,
    JsonDecode,
    UnicodeDecode,
    NotFound,
    Exists,
    Permission,
    Os,
    Timeout,
}

impl ProvKind {
    pub fn name(self) -> &'static str {
        match self {
            ProvKind::Value => "ValueError",
            ProvKind::Key => "KeyError",
            ProvKind::Type => "TypeError",
            ProvKind::Attribute => "AttributeError",
            ProvKind::JsonDecode => "JSONDecodeError",
            ProvKind::UnicodeDecode => "UnicodeDecodeError",
            ProvKind::NotFound => "FileNotFoundError",
            ProvKind::Exists => "FileExistsError",
            ProvKind::Permission => "PermissionError",
            ProvKind::Os => "OSError",
            ProvKind::Timeout => "TimeoutExpired",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ProvError {
    pub kind: ProvKind,
    pub detail: String,
}

impl ProvError {
    fn new(kind: ProvKind, detail: impl Into<String>) -> ProvError {
        ProvError {
            kind,
            detail: detail.into(),
        }
    }

    fn value(detail: impl Into<String>) -> ProvError {
        ProvError::new(ProvKind::Value, detail)
    }

    fn io(error: &std::io::Error, detail: impl Into<String>) -> ProvError {
        let kind = match error.kind() {
            ErrorKind::NotFound => ProvKind::NotFound,
            ErrorKind::AlreadyExists => ProvKind::Exists,
            ErrorKind::PermissionDenied => ProvKind::Permission,
            _ => ProvKind::Os,
        };
        ProvError::new(kind, detail)
    }
}
