//! Strict JSON input and Go-exact JSON output.
//!
//! [`read_json_file`] mirrors `ReadJSON`: the parent directory stays open,
//! the file must be regular and bounded, and its bytes must not change
//! between stat and read. Decoding rejects unknown fields and trailing
//! data, like Go's `DisallowUnknownFields` plus the second-decode check.
//!
//! Emission mirrors Go's `encoding/json` byte for byte: HTML escaping
//! (`<`, `>`, `&`), `U+2028/2029` escaping, raw number passthrough (the
//! `UseNumber` identity the evidence tests pin), compact separators for
//! reports and two-space indented separators for structured evidence.

use std::fs::File;
use std::io::Read;
use std::path::Path;

use soda_json::JsonValue;

use crate::error::Error;
use crate::files::{self, FileAttr, OwnedDir};
use crate::sha256;

mod timestamps;

pub use timestamps::{now_rfc3339_nano, validate_rfc3339};

#[cfg(test)]
use timestamps::format_unix_nano;

/// Read one JSON value from a bounded regular file, returning the value and
/// the SHA-256 hex of the exact bytes decoded. Mirrors `ReadJSON`.
pub fn read_json_file(path: &str) -> Result<(JsonValue, String), Error> {
    let file = Path::new(path);
    let parent = file.parent().unwrap_or(Path::new("/"));
    let base = file
        .file_name()
        .ok_or_else(|| Error::msg("bounded regular JSON input required"))?;
    let root = OwnedDir::open(&parent.to_string_lossy())?;
    let name = base.to_string_lossy();
    let before = root.lstat_at(&name)?;
    if !before.is_regular || before.size > files::JSON_LIMIT {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    let handle = root.open_file_at(&name)?;
    let after = fstat_attr(&handle)?;
    if !after.is_regular || !files::same_file(before, after) {
        return Err(Error::msg("JSON input changed before reading"));
    }
    let mut data = Vec::new();
    handle.take(files::JSON_LIMIT + 1).read_to_end(&mut data)?;
    if data.len() as u64 > files::JSON_LIMIT {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    let text = std::str::from_utf8(&data).map_err(|_| Error::msg("invalid JSON input"))?;
    let value = JsonValue::parse(text).map_err(|_| Error::msg("invalid JSON input"))?;
    Ok((value, sha256::hex_lower(&sha256::digest(&data))))
}

/// Read one JSON value through an already-open directory. Mirrors
/// `ReadJSONAt`: same bounds and change checks, digest of decoded bytes.
pub fn read_json_at(root: &OwnedDir, name: &str) -> Result<(JsonValue, String), Error> {
    let before = root.lstat_at(name)?;
    if !before.is_regular || before.size > files::JSON_LIMIT {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    let handle = root.open_file_at(name)?;
    let after = fstat_attr(&handle)?;
    if !after.is_regular || !files::same_file(before, after) {
        return Err(Error::msg("JSON input changed before reading"));
    }
    let mut data = Vec::new();
    handle.take(files::JSON_LIMIT + 1).read_to_end(&mut data)?;
    if data.len() as u64 > files::JSON_LIMIT {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    let text = std::str::from_utf8(&data).map_err(|_| Error::msg("invalid JSON input"))?;
    let value = JsonValue::parse(text).map_err(|_| Error::msg("invalid JSON input"))?;
    Ok((value, sha256::hex_lower(&sha256::digest(&data))))
}

fn fstat_attr(file: &File) -> Result<FileAttr, Error> {
    use std::os::fd::AsRawFd;
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::fstat(file.as_raw_fd(), &mut st) };
    if rc != 0 {
        return Err(Error::from(std::io::Error::last_os_error()));
    }
    let mode = st.st_mode;
    Ok(FileAttr {
        is_regular: mode & libc::S_IFMT == libc::S_IFREG,
        is_dir: mode & libc::S_IFMT == libc::S_IFDIR,
        is_symlink: mode & libc::S_IFMT == libc::S_IFLNK,
        perm: mode & 0o7777,
        size: st.st_size.max(0) as u64,
        uid: st.st_uid,
        gid: st.st_gid,
        dev: st.st_dev as u64,
        ino: st.st_ino as u64,
    })
}

/// Reject any object key outside `known`, like `DisallowUnknownFields`.
/// Duplicate keys keep last-wins lookup (matching Go), but every spelling
/// present must be known.
pub fn check_no_unknown(value: &JsonValue, known: &[&str]) -> Result<(), Error> {
    if let JsonValue::Object(entries) = value {
        for (key, _) in entries {
            if !known.contains(&key.as_str()) {
                return Err(Error::msg(format!("json: unknown field {key:?}")));
            }
        }
    }
    Ok(())
}

/// Required string field, like decoding into a Go `string`.
pub fn require_string<'a>(value: &'a JsonValue, field: &str) -> Result<&'a str, Error> {
    value
        .get(field)
        .and_then(|v| v.as_str())
        .ok_or_else(|| Error::msg(format!("invalid {field}: string required")))
}

/// Required strict integer field.
pub fn require_integer(value: &JsonValue, field: &str) -> Result<i128, Error> {
    value
        .get(field)
        .and_then(|v| v.as_integer())
        .ok_or_else(|| Error::msg(format!("invalid {field}: integer required")))
}

/// Required boolean field.
pub fn require_bool(value: &JsonValue, field: &str) -> Result<bool, Error> {
    value
        .get(field)
        .and_then(|v| v.as_bool())
        .ok_or_else(|| Error::msg(format!("invalid {field}: boolean required")))
}

/// Optional string field: missing or null decodes to empty, like Go's zero
/// value; a present mistyped value fails, like Go's unmarshal error.
pub fn opt_string(value: &JsonValue, field: &str) -> Result<String, Error> {
    match value.get(field) {
        None | Some(JsonValue::Null) => Ok(String::new()),
        Some(JsonValue::Str(s)) => Ok(s.clone()),
        Some(_) => Err(Error::msg(format!("invalid {field}: string required"))),
    }
}

/// Optional integer field: missing or null decodes to zero; a present
/// mistyped value fails.
pub fn opt_integer(value: &JsonValue, field: &str) -> Result<i128, Error> {
    match value.get(field) {
        None | Some(JsonValue::Null) => Ok(0),
        Some(v) => v
            .as_integer()
            .ok_or_else(|| Error::msg(format!("invalid {field}: integer required"))),
    }
}

/// Optional boolean field: missing or null decodes to false.
pub fn opt_bool(value: &JsonValue, field: &str) -> Result<bool, Error> {
    match value.get(field) {
        None | Some(JsonValue::Null) => Ok(false),
        Some(JsonValue::Bool(b)) => Ok(*b),
        Some(_) => Err(Error::msg(format!("invalid {field}: boolean required"))),
    }
}

/// Escape a string exactly like Go's `encoding/json`: short escapes, HTML
/// characters, `U+2028/2029`, and `\u00xx` for other controls.
pub fn escape_go(out: &mut String, value: &str) {
    out.push('"');
    for ch in value.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '<' => out.push_str("\\u003c"),
            '>' => out.push_str("\\u003e"),
            '&' => out.push_str("\\u0026"),
            '\u{2028}' => out.push_str("\\u2028"),
            '\u{2029}' => out.push_str("\\u2029"),
            c if (c as u32) < 0x20 => {
                out.push_str(&format!("\\u{:04x}", c as u32));
            }
            c => out.push(c),
        }
    }
    out.push('"');
}

/// Compact emission, like `json.Marshal`. Object entries keep caller order;
/// callers sort maps first, matching Go's sorted map keys.
pub fn write_compact(out: &mut String, value: &JsonValue) {
    match value {
        JsonValue::Null => out.push_str("null"),
        JsonValue::Bool(true) => out.push_str("true"),
        JsonValue::Bool(false) => out.push_str("false"),
        JsonValue::Number(raw) => out.push_str(raw),
        JsonValue::Str(s) => escape_go(out, s),
        JsonValue::Array(items) => {
            out.push('[');
            for (i, item) in items.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                write_compact(out, item);
            }
            out.push(']');
        }
        JsonValue::Object(entries) => {
            out.push('{');
            for (i, (key, item)) in entries.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                escape_go(out, key);
                out.push(':');
                write_compact(out, item);
            }
            out.push('}');
        }
    }
}

/// Two-space indented emission, like `json.MarshalIndent(v, "", "  ")`.
/// No trailing newline; callers append it like the Go owner does.
pub fn write_indent(out: &mut String, value: &JsonValue) {
    write_indent_at(out, value, 0);
}

fn write_indent_at(out: &mut String, value: &JsonValue, depth: usize) {
    match value {
        JsonValue::Array(items) if !items.is_empty() => {
            out.push_str("[\n");
            for (i, item) in items.iter().enumerate() {
                if i > 0 {
                    out.push_str(",\n");
                }
                write_padding(out, depth + 1);
                write_indent_at(out, item, depth + 1);
            }
            out.push('\n');
            write_padding(out, depth);
            out.push(']');
        }
        JsonValue::Object(entries) if !entries.is_empty() => {
            out.push_str("{\n");
            for (i, (key, item)) in entries.iter().enumerate() {
                if i > 0 {
                    out.push_str(",\n");
                }
                write_padding(out, depth + 1);
                escape_go(out, key);
                out.push_str(": ");
                write_indent_at(out, item, depth + 1);
            }
            out.push('\n');
            write_padding(out, depth);
            out.push('}');
        }
        _ => write_compact(out, value),
    }
}

fn write_padding(out: &mut String, depth: usize) {
    for _ in 0..depth {
        out.push_str("  ");
    }
}

#[cfg(test)]
mod tests;
