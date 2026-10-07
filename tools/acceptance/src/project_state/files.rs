use std::io::Read;
use std::os::unix::ffi::OsStrExt;
use std::path::{Path, PathBuf};

use crate::structured::{self, Value as JsonValue};

use crate::sha256::{self, Digest, Sha256};

use super::{n, obj, s, set, SnapshotFailure, SnapshotKind};

/// Snapshot entry payload beyond uid/gid/mode.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum EntryBody {
    /// File content hash.
    Sha256(String),
    /// File size without content access.
    Size(u64),
    /// Directory: identity only.
    Dir,
}

/// One snapshotted path.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Entry {
    uid: u32,
    gid: u32,
    mode: u32,
    body: EntryBody,
}

impl Entry {
    /// Snapshot one path via `lstat`, like the retired probe's `entry`.
    pub fn snapshot(path: &Path, contents: bool) -> Result<Entry, SnapshotFailure> {
        use std::os::unix::fs::MetadataExt;
        let meta = std::fs::symlink_metadata(path).map_err(SnapshotFailure::io)?;
        let file_type = meta.file_type();
        let entry = Entry {
            uid: meta.uid(),
            gid: meta.gid(),
            mode: meta.mode() & 0o7777,
            body: EntryBody::Dir,
        };
        // Fail closed on links, like the retired probe: `lstat` sees neither a
        // regular file nor a directory, so the snapshot aborts instead of
        // following or recording an administrator's unexpected link.
        if file_type.is_symlink() {
            return Err(SnapshotFailure::bare(SnapshotKind::FileNotFoundError));
        }
        if file_type.is_file() && contents {
            if meta.len() > 512 * 1024 * 1024 {
                return Err(SnapshotFailure::runtime("Snapshot file too large"));
            }
            let mut file = std::fs::File::open(path).map_err(SnapshotFailure::io)?;
            let mut digest = Sha256::new();
            let mut chunk = vec![0u8; 1024 * 1024];
            loop {
                let n = file.read(&mut chunk).map_err(SnapshotFailure::io)?;
                if n == 0 {
                    break;
                }
                digest.update(&chunk[..n]);
            }
            return Ok(Entry {
                body: EntryBody::Sha256(sha256::hex_lower(&digest.finalize())),
                ..entry
            });
        }
        if file_type.is_file() {
            return Ok(Entry {
                body: EntryBody::Size(meta.len()),
                ..entry
            });
        }
        if !file_type.is_dir() {
            return Err(SnapshotFailure::runtime("Unsupported snapshot file"));
        }
        Ok(entry)
    }

    /// Render as the retired probe's JSON object.
    pub fn json(&self) -> JsonValue {
        let mut object = obj();
        set(&mut object, "uid", n(self.uid));
        set(&mut object, "gid", n(self.gid));
        set(&mut object, "mode", n(self.mode));
        match &self.body {
            EntryBody::Sha256(hash) => set(&mut object, "sha256", s(hash.clone())),
            EntryBody::Size(size) => set(&mut object, "size", n(size)),
            EntryBody::Dir => {}
        }
        object
    }
}

/// Python `json.dumps(sort_keys=True)` rendering: sorted keys,
/// `(', ', ': ')` separators, ASCII-only output with lowercase `\u`
/// escapes. Snapshot values are strings, integers, lists, and objects.
pub fn dumps_sorted(value: &JsonValue) -> String {
    let mut out = String::new();
    structured::write_python_sorted(&mut out, value);
    out
}

pub(super) fn list_files(dir: &Path) -> Result<Vec<PathBuf>, SnapshotFailure> {
    let mut files = Vec::new();
    for entry in std::fs::read_dir(dir).map_err(SnapshotFailure::io)? {
        let path = entry.map_err(SnapshotFailure::io)?.path();
        // `Path.is_file` follows symlinks, like the retired probe.
        if path.is_file() {
            files.push(path);
        }
    }
    files.sort();
    Ok(files)
}

pub(super) fn has_git_part(checkout: &Path, path: &Path) -> bool {
    path.strip_prefix(checkout)
        .map(|relative| {
            relative
                .components()
                .any(|part| part.as_os_str().as_bytes() == b".git")
        })
        .unwrap_or(false)
}

pub(super) fn walk_sorted(root: &Path) -> Result<Vec<PathBuf>, SnapshotFailure> {
    let mut found = Vec::new();
    let mut stack = vec![root.to_path_buf()];
    while let Some(dir) = stack.pop() {
        let mut entries: Vec<PathBuf> = Vec::new();
        for entry in std::fs::read_dir(&dir).map_err(SnapshotFailure::io)? {
            entries.push(entry.map_err(SnapshotFailure::io)?.path());
        }
        entries.sort();
        for path in entries {
            found.push(path.clone());
            let file_type = std::fs::symlink_metadata(&path)
                .map_err(SnapshotFailure::io)?
                .file_type();
            if file_type.is_dir() && !file_type.is_symlink() {
                stack.push(path);
            }
        }
    }
    found.sort();
    Ok(found)
}
