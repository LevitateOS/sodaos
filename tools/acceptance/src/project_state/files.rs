use std::io::Read;
use std::os::unix::ffi::OsStrExt;
use std::path::{Path, PathBuf};

use soda_json::JsonValue;

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
    fn escape(text: &str, out: &mut String) {
        out.push('"');
        for ch in text.chars() {
            match ch {
                '"' => out.push_str("\\\""),
                '\\' => out.push_str("\\\\"),
                '\n' => out.push_str("\\n"),
                '\r' => out.push_str("\\r"),
                '\t' => out.push_str("\\t"),
                '\u{08}' => out.push_str("\\b"),
                '\u{0c}' => out.push_str("\\f"),
                c if (c as u32) < 0x20 => {
                    out.push_str(&format!("\\u{:04x}", c as u32));
                }
                c if (c as u32) < 0x7f => out.push(c),
                c if (c as u32) < 0x10000 => {
                    out.push_str(&format!("\\u{:04x}", c as u32));
                }
                c => {
                    // ensure_ascii astral planes as lowercase surrogate pairs.
                    let code = c as u32 - 0x10000;
                    out.push_str(&format!(
                        "\\u{:04x}\\u{:04x}",
                        0xd800 + (code >> 10),
                        0xdc00 + (code & 0x3ff)
                    ));
                }
            }
        }
        out.push('"');
    }

    fn render(value: &JsonValue, out: &mut String) {
        match value {
            JsonValue::Null => out.push_str("null"),
            JsonValue::Bool(true) => out.push_str("true"),
            JsonValue::Bool(false) => out.push_str("false"),
            JsonValue::Number(raw) => out.push_str(raw),
            JsonValue::Str(text) => escape(text, out),
            JsonValue::Array(items) => {
                out.push('[');
                for (index, item) in items.iter().enumerate() {
                    if index > 0 {
                        out.push_str(", ");
                    }
                    render(item, out);
                }
                out.push(']');
            }
            JsonValue::Object(entries) => {
                let mut order: Vec<&(String, JsonValue)> = entries.iter().collect();
                order.sort_by(|a, b| a.0.cmp(&b.0));
                out.push('{');
                for (index, (key, item)) in order.iter().enumerate() {
                    if index > 0 {
                        out.push_str(", ");
                    }
                    escape(key, out);
                    out.push_str(": ");
                    render(item, out);
                }
                out.push('}');
            }
        }
    }

    let mut out = String::new();
    render(value, &mut out);
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
