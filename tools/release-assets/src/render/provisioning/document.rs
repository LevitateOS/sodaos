//! Python-compatible JSON emission and bootstrap document construction.

use std::path::Path;

use soda_json::JsonValue;

use super::private_files::read_text;
use super::{ProvError, ProvKind};

/// Escape a string the way Python's `json.dump` with the default
/// `ensure_ascii=True` does: short escapes for the common controls,
/// lowercase `\uXXXX` for everything outside printable ASCII (including
/// DEL), surrogate pairs above the BMP. Unlike Go's escaper, `/`, `<`,
/// `>` and `&` stay raw.
fn escape_python_into(out: &mut String, value: &str) {
    out.push('"');
    for ch in value.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\u{0008}' => out.push_str("\\b"),
            '\u{000c}' => out.push_str("\\f"),
            c if (c as u32) < 0x20 => out.push_str(&format!("\\u{:04x}", c as u32)),
            c if (c as u32) < 0x7f => out.push(c),
            c if (c as u32) <= 0xffff => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => {
                let v = c as u32 - 0x10000;
                out.push_str(&format!(
                    "\\u{:04x}\\u{:04x}",
                    0xd800 + (v >> 10),
                    0xdc00 + (v & 0x3ff)
                ));
            }
        }
    }
    out.push('"');
}

fn indent_into(out: &mut String, level: usize) {
    for _ in 0..level {
        out.push_str("  ");
    }
}

/// Render a value exactly like `json.dump(value, indent=2)`: two-space
/// levels, `": "` after keys, empty containers inline. Number literals are
/// preserved verbatim (Python would renormalize floats, but Butane inputs
/// only carry integers, where both spellings agree).
pub fn dump_python(value: &JsonValue) -> String {
    let mut out = String::new();
    emit_python(&mut out, value, 0);
    out
}

fn emit_python(out: &mut String, value: &JsonValue, level: usize) {
    match value {
        JsonValue::Null => out.push_str("null"),
        JsonValue::Bool(true) => out.push_str("true"),
        JsonValue::Bool(false) => out.push_str("false"),
        JsonValue::Number(raw) => out.push_str(raw),
        JsonValue::Str(text) => escape_python_into(out, text),
        JsonValue::Array(items) => {
            if items.is_empty() {
                out.push_str("[]");
                return;
            }
            out.push_str("[\n");
            for (i, item) in items.iter().enumerate() {
                indent_into(out, level + 1);
                emit_python(out, item, level + 1);
                if i + 1 < items.len() {
                    out.push(',');
                }
                out.push('\n');
            }
            indent_into(out, level);
            out.push(']');
        }
        JsonValue::Object(entries) => {
            if entries.is_empty() {
                out.push_str("{}");
                return;
            }
            out.push_str("{\n");
            for (i, (key, item)) in entries.iter().enumerate() {
                indent_into(out, level + 1);
                escape_python_into(out, key);
                out.push_str(": ");
                emit_python(out, item, level + 1);
                if i + 1 < entries.len() {
                    out.push(',');
                }
                out.push('\n');
            }
            indent_into(out, level);
            out.push('}');
        }
    }
}

pub(crate) fn object_mut(value: &mut JsonValue) -> Option<&mut Vec<(String, JsonValue)>> {
    match value {
        JsonValue::Object(entries) => Some(entries),
        _ => None,
    }
}

fn get_mut<'a>(entries: &'a mut [(String, JsonValue)], key: &str) -> Option<&'a mut JsonValue> {
    entries
        .iter_mut()
        .rev()
        .find(|(k, _)| k == key)
        .map(|(_, v)| v)
}

/// Navigate to the storage file list, reporting `KeyError`/`TypeError`
/// like the script's bare subscripts. A non-list `files` entry reports
/// `AttributeError`, like the script's failed `.append`.
fn files_mut(config: &mut JsonValue) -> Result<&mut JsonValue, ProvError> {
    let entries = object_mut(config)
        .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap must be an object"))?;
    let storage = get_mut(entries, "storage")
        .ok_or_else(|| ProvError::new(ProvKind::Key, "bootstrap has no storage"))?;
    let storage_entries = object_mut(storage)
        .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap storage must be an object"))?;
    let files = get_mut(storage_entries, "files")
        .ok_or_else(|| ProvError::new(ProvKind::Key, "bootstrap storage has no files"))?;
    if matches!(files, JsonValue::Array(_)) {
        Ok(files)
    } else {
        Err(ProvError::new(
            ProvKind::Attribute,
            "bootstrap files have no append",
        ))
    }
}

fn not_a_list() -> ProvError {
    ProvError::new(ProvKind::Attribute, "bootstrap files have no append")
}

pub(crate) fn push_file(config: &mut JsonValue, entry: JsonValue) -> Result<(), ProvError> {
    match files_mut(config)? {
        JsonValue::Array(items) => {
            items.push(entry);
            Ok(())
        }
        _ => Err(not_a_list()),
    }
}

pub(crate) fn clear_files(config: &mut JsonValue) -> Result<(), ProvError> {
    match files_mut(config)? {
        JsonValue::Array(items) => {
            items.clear();
            Ok(())
        }
        _ => Err(not_a_list()),
    }
}

pub(crate) fn str_value(text: &str) -> JsonValue {
    JsonValue::Str(text.to_string())
}

pub(crate) fn file_entry(path: &str, mode: u32, inline: &str) -> JsonValue {
    JsonValue::Object(vec![
        ("path".to_string(), str_value(path)),
        ("mode".to_string(), JsonValue::Number(mode.to_string())),
        (
            "contents".to_string(),
            JsonValue::Object(vec![("inline".to_string(), str_value(inline))]),
        ),
    ])
}

/// One public bootstrap for private provisioning and installer-media
/// conversion: the base document plus the shared branding file.
pub(crate) fn public_config(source: &Path) -> Result<JsonValue, ProvError> {
    let path = source.join("appliance/provisioning/base.json");
    let text = read_text(&path)?;
    let mut config = JsonValue::parse(&text).map_err(|_| {
        ProvError::new(
            ProvKind::JsonDecode,
            format!("cannot parse {}", path.display()),
        )
    })?;
    let svg = read_text(&source.join("assets/branding/source/soda-symbol.svg"))?;
    push_file(
        &mut config,
        file_entry(
            "/var/usrlocal/share/icons/hicolor/scalable/apps/sodaos-icon.svg",
            0o644,
            &svg,
        ),
    )?;
    Ok(config)
}
