//! Bounded, read-only U08 state snapshot. This module owns the probe (it
//! replaces the retired `tests/installed/project-state.py`).
//!
//! Run as root INSIDE a selected project. No
//! environment/secret/DB-credential/shadow/private-key contents are
//! exported. The driver supplies the compiled probe over the selected
//! administrator SSH session.
//!
//! Failure taxonomy mirrors the retired probe: `RuntimeError` and
//! `AssertionError` keep their detail text, every other kind renders
//! with an empty detail.

use std::os::unix::ffi::OsStrExt;
use std::path::{Path, PathBuf};

use soda_json::JsonValue;

mod command;
mod files;
mod snapshot;
mod workloads;

pub use command::{command, command_with_timeout, output_lines};
pub use files::{dumps_sorted, Entry, EntryBody};

pub use snapshot::run_snapshot;

#[cfg(test)]
use workloads::check_project_ip;

/// Snapshot failure kind, mirroring the Python exception taxonomy.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SnapshotKind {
    /// Explicit snapshot violation.
    RuntimeError,
    /// Failed assertion.
    AssertionError,
    /// Missing path.
    FileNotFoundError,
    /// Unreadable path.
    PermissionError,
    /// Other OS failure.
    OSError,
    /// Snapshot command timed out.
    TimeoutExpired,
    /// Snapshot command output was not UTF-8.
    UnicodeDecodeError,
    /// Malformed project IP.
    ValueError,
    /// Missing required environment variable.
    KeyError,
}

impl SnapshotKind {
    /// Exception name.
    pub fn name(self) -> &'static str {
        match self {
            SnapshotKind::RuntimeError => "RuntimeError",
            SnapshotKind::AssertionError => "AssertionError",
            SnapshotKind::FileNotFoundError => "FileNotFoundError",
            SnapshotKind::PermissionError => "PermissionError",
            SnapshotKind::OSError => "OSError",
            SnapshotKind::TimeoutExpired => "TimeoutExpired",
            SnapshotKind::UnicodeDecodeError => "UnicodeDecodeError",
            SnapshotKind::ValueError => "ValueError",
            SnapshotKind::KeyError => "KeyError",
        }
    }
}

/// Snapshot failure. Only `RuntimeError` and `AssertionError` carry
/// detail, like the retired probe's handler.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SnapshotFailure {
    /// Failure kind.
    pub kind: SnapshotKind,
    /// Detail text (empty unless RuntimeError/AssertionError).
    pub detail: String,
}

impl SnapshotFailure {
    fn runtime(detail: impl Into<String>) -> SnapshotFailure {
        SnapshotFailure {
            kind: SnapshotKind::RuntimeError,
            detail: detail.into(),
        }
    }

    fn assertion(detail: impl Into<String>) -> SnapshotFailure {
        SnapshotFailure {
            kind: SnapshotKind::AssertionError,
            detail: detail.into(),
        }
    }

    fn bare(kind: SnapshotKind) -> SnapshotFailure {
        SnapshotFailure {
            kind,
            detail: String::new(),
        }
    }

    fn io(error: std::io::Error) -> SnapshotFailure {
        use std::io::ErrorKind;
        match error.kind() {
            ErrorKind::NotFound => SnapshotFailure::bare(SnapshotKind::FileNotFoundError),
            ErrorKind::PermissionDenied => SnapshotFailure::bare(SnapshotKind::PermissionError),
            // `read_to_string` is the only `InvalidData` source here, like
            // the retired probe's strict `read_text`/`decode` calls.
            ErrorKind::InvalidData => SnapshotFailure::bare(SnapshotKind::UnicodeDecodeError),
            _ => SnapshotFailure::bare(SnapshotKind::OSError),
        }
    }
}

impl std::fmt::Display for SnapshotFailure {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "Project snapshot failed: {} {}",
            self.kind.name(),
            self.detail
        )
    }
}

impl std::error::Error for SnapshotFailure {}

/// Empty JSON object.
fn obj() -> JsonValue {
    JsonValue::Object(Vec::new())
}

/// JSON string.
fn s(text: impl Into<String>) -> JsonValue {
    JsonValue::Str(text.into())
}

/// JSON integer.
fn n(value: impl std::fmt::Display) -> JsonValue {
    JsonValue::Number(value.to_string())
}

/// Dict-style assignment: replace an existing key, else append. The
/// The retired probe overwrites `data['files']` entries when the final
/// hashed pass covers a path recorded earlier without contents.
fn set(object: &mut JsonValue, key: &str, value: JsonValue) {
    if let JsonValue::Object(entries) = object {
        if let Some(slot) = entries.iter_mut().find(|(k, _)| k == key) {
            slot.1 = value;
            return;
        }
        entries.push((key.to_string(), value));
    }
}

/// Mutable access to a named child object, creating it when missing.
fn sub_mut<'a>(object: &'a mut JsonValue, key: &str) -> &'a mut JsonValue {
    if object.get(key).is_none() {
        set(object, key, obj());
    }
    if let JsonValue::Object(entries) = object {
        let index = entries.iter().position(|(k, _)| k == key).unwrap();
        return &mut entries[index].1;
    }
    unreachable!("snapshot document nodes are objects");
}

/// Snapshot one home's `.ssh` entries in sorted name order. Only
/// `config` and `known_hosts` export hashes; every other file exports
/// size, and `authorized_keys` is excluded (it is covered by the
/// accounts dump). `is_file`/`is_dir` follow symlinks exactly like the
/// owner's checks, and the entry itself fails the snapshot on a link.
fn snapshot_ssh_files(home: &Path) -> Result<Vec<(String, JsonValue)>, SnapshotFailure> {
    let ssh = home.join(".ssh");
    let mut names = Vec::new();
    for dir_entry in std::fs::read_dir(&ssh).map_err(SnapshotFailure::io)? {
        names.push(dir_entry.map_err(SnapshotFailure::io)?.file_name());
    }
    names.sort();
    let mut out = Vec::new();
    for name in names {
        let path = ssh.join(&name);
        let name = name.to_string_lossy().into_owned();
        if path.is_file() && name != "authorized_keys" {
            let contents = name == "config" || name == "known_hosts";
            out.push((name, Entry::snapshot(&path, contents)?.json()));
        } else if path.is_dir()
            && !std::fs::symlink_metadata(&path)
                .map_err(SnapshotFailure::io)?
                .file_type()
                .is_symlink()
            && name != "u08-personal-git"
        {
            out.push((name, Entry::snapshot(&path, false)?.json()));
        }
    }
    Ok(out)
}

fn glob_prefix(dir: &Path, prefix: &str) -> Result<Vec<PathBuf>, SnapshotFailure> {
    let mut hits = Vec::new();
    // A missing directory yields no matches, like `Path.glob`; other
    // read failures propagate like the retired probe's `OSError`.
    let read = match std::fs::read_dir(dir) {
        Ok(read) => read,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(hits),
        Err(error) => return Err(SnapshotFailure::io(error)),
    };
    for entry in read {
        let entry = entry.map_err(SnapshotFailure::io)?;
        if entry.file_name().as_bytes().starts_with(prefix.as_bytes()) {
            hits.push(entry.path());
        }
    }
    hits.sort();
    Ok(hits)
}

fn arg_list(args: &[&str]) -> Vec<String> {
    args.iter().map(|arg| arg.to_string()).collect()
}

/// Entry gate: root inside a container.
pub fn check_snapshot_gate(euid: u32, containerenv: &Path) -> Result<(), SnapshotFailure> {
    if euid != 0 || !containerenv.exists() {
        return Err(SnapshotFailure::assertion(""));
    }
    Ok(())
}

#[cfg(test)]
mod tests;
