//! Provisioning renderer (ports `scripts/render-provisioning.py`).
//!
//! Merges the public Butane bootstrap with private per-instance operator
//! inputs. No conversion, installation, account enrollment or reboot is
//! implicit. Rendered documents, secret handling and failure messages
//! match the script; only the argparse envelope carries the new name.

use std::io::ErrorKind;
use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::time::{Duration, Instant};

use soda_json::JsonValue;

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

fn read_text(path: &Path) -> Result<String, ProvError> {
    match std::fs::read_to_string(path) {
        Ok(text) => Ok(text),
        Err(e) if e.kind() == ErrorKind::InvalidData => Err(ProvError::new(
            ProvKind::UnicodeDecode,
            format!("cannot decode {}", path.display()),
        )),
        Err(e) => Err(ProvError::io(&e, format!("cannot read {}", path.display()))),
    }
}

/// Bounded regular input, like the script's `regular()`: lstat must show a
/// plain file under 1 MiB, and secret inputs must hide from group/others.
fn regular(path: &Path, private: bool) -> Result<String, ProvError> {
    let meta = std::fs::symlink_metadata(path)
        .map_err(|e| ProvError::io(&e, format!("cannot stat {}", path.display())))?;
    if !meta.file_type().is_file() || meta.len() > 1024 * 1024 {
        return Err(ProvError::value("bounded regular input required"));
    }
    #[cfg(unix)]
    {
        use std::os::unix::fs::MetadataExt;
        if private && meta.mode() & 0o077 != 0 {
            return Err(ProvError::value(
                "secret input must not be accessible to group/others",
            ));
        }
    }
    read_text(path)
}

fn is_label(label: &str, max_middle: usize) -> bool {
    let bytes = label.as_bytes();
    if bytes.is_empty() || bytes.len() > max_middle + 2 {
        return false;
    }
    let edge = |b: u8| b.is_ascii_lowercase() || b.is_ascii_digit();
    if bytes.len() == 1 {
        return edge(bytes[0]);
    }
    if !edge(bytes[0]) || !edge(bytes[bytes.len() - 1]) || bytes.len() - 2 > max_middle {
        return false;
    }
    bytes[1..bytes.len() - 1]
        .iter()
        .all(|b| edge(*b) || *b == b'-')
}

/// Like the script's `appliance_hostname`: 1..=253 chars of dot-separated
/// lowercase alphanumeric labels.
pub fn is_appliance_hostname(value: &str) -> bool {
    let len = value.chars().count();
    (1..=253).contains(&len) && value.split('.').all(|label| is_label(label, 61))
}

/// Like the fixture check: `soda-native-` plus a 1..=42-char label tail.
pub fn is_fixture_hostname(value: &str) -> bool {
    value
        .strip_prefix("soda-native-")
        .is_some_and(|rest| is_label(rest, 40))
}

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

fn object_mut(value: &mut JsonValue) -> Option<&mut Vec<(String, JsonValue)>> {
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

fn push_file(config: &mut JsonValue, entry: JsonValue) -> Result<(), ProvError> {
    match files_mut(config)? {
        JsonValue::Array(items) => {
            items.push(entry);
            Ok(())
        }
        _ => Err(not_a_list()),
    }
}

fn clear_files(config: &mut JsonValue) -> Result<(), ProvError> {
    match files_mut(config)? {
        JsonValue::Array(items) => {
            items.clear();
            Ok(())
        }
        _ => Err(not_a_list()),
    }
}

fn str_value(text: &str) -> JsonValue {
    JsonValue::Str(text.to_string())
}

fn file_entry(path: &str, mode: u32, inline: &str) -> JsonValue {
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
fn public_config(source: &Path) -> Result<JsonValue, ProvError> {
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

/// `Path.absolute()`: join the working directory lexically, resolving
/// nothing, so the key path in argv matches the script's spelling.
fn absolute_lexical(path: &Path) -> Result<PathBuf, ProvError> {
    if path.is_absolute() {
        return Ok(path.to_path_buf());
    }
    let cwd =
        std::env::current_dir().map_err(|e| ProvError::io(&e, "cannot read working directory"))?;
    Ok(cwd.join(path))
}

/// Derive the public half without putting private contents in argv or
/// logs. Encrypted/wrong-type inputs fail without an interactive prompt.
fn derive_host_public(host_key: &Path) -> Result<String, ProvError> {
    let absolute = absolute_lexical(host_key)?;
    let mut child = std::process::Command::new("ssh-keygen")
        .args(["-y", "-P", "", "-f"])
        .arg(&absolute)
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| ProvError::io(&e, "cannot run ssh-keygen"))?;
    let stderr = child.stderr.take().map(|mut pipe| {
        std::thread::spawn(move || {
            let mut sink = Vec::new();
            let _ = std::io::Read::read_to_end(&mut pipe, &mut sink);
        })
    });
    let deadline = Instant::now() + Duration::from_secs(10);
    let status = loop {
        match child
            .try_wait()
            .map_err(|e| ProvError::io(&e, "cannot run ssh-keygen"))?
        {
            Some(status) => break status,
            None => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    if let Some(handle) = stderr {
                        let _ = handle.join();
                    }
                    return Err(ProvError::new(ProvKind::Timeout, "ssh-keygen timed out"));
                }
                std::thread::sleep(Duration::from_millis(5));
            }
        }
    };
    let mut stdout = Vec::new();
    if let Some(mut pipe) = child.stdout.take() {
        std::io::Read::read_to_end(&mut pipe, &mut stdout)
            .map_err(|e| ProvError::io(&e, "cannot run ssh-keygen"))?;
    }
    if let Some(handle) = stderr {
        let _ = handle.join();
    }
    if !status.success() {
        return Err(ProvError::value(
            "unencrypted per-instance Ed25519 host key required",
        ));
    }
    let text = String::from_utf8(stdout)
        .map_err(|_| ProvError::new(ProvKind::UnicodeDecode, "cannot decode ssh-keygen output"))?;
    if !text.starts_with("ssh-ed25519 ") {
        return Err(ProvError::value(
            "unencrypted per-instance Ed25519 host key required",
        ));
    }
    Ok(text)
}

/// The script's `dest.parent.resolve() != dest.parent` gate: no symlinked
/// ancestor and no `.`/`..` components (both would resolve away from the
/// given spelling). Missing parents pass here and fail at `stat`, like
/// the script's non-strict resolve.
fn is_real_parent(parent: &Path) -> bool {
    if parent.components().any(|c| {
        matches!(
            c,
            std::path::Component::CurDir | std::path::Component::ParentDir
        )
    }) {
        return false;
    }
    !parent.ancestors().any(|ancestor| {
        std::fs::symlink_metadata(ancestor).is_ok_and(|m| m.file_type().is_symlink())
    })
}

fn write_exclusive(dest: &Path, document: &str) -> Result<(), ProvError> {
    if !dest.is_absolute() {
        return Err(ProvError::value(
            "absolute output under a real private parent required",
        ));
    }
    let parent = dest.parent().unwrap_or(dest);
    if !is_real_parent(parent) {
        return Err(ProvError::value(
            "absolute output under a real private parent required",
        ));
    }
    let mode = {
        #[cfg(unix)]
        {
            use std::os::unix::fs::MetadataExt;
            std::fs::metadata(parent)
                .map_err(|e| ProvError::io(&e, format!("cannot stat {}", parent.display())))?
                .mode()
        }
        #[cfg(not(unix))]
        {
            let _ = std::fs::metadata(parent)
                .map_err(|e| ProvError::io(&e, format!("cannot stat {}", parent.display())))?;
            0u32
        }
    };
    if mode & 0o077 != 0 {
        return Err(ProvError::value("per-instance parent must be private"));
    }
    let mut options = std::fs::OpenOptions::new();
    options.write(true).create_new(true);
    #[cfg(unix)]
    {
        use std::os::unix::fs::OpenOptionsExt;
        options.mode(0o600);
    }
    let mut file = options
        .open(dest)
        .map_err(|e| ProvError::io(&e, format!("cannot create {}", dest.display())))?;
    std::io::Write::write_all(&mut file, document.as_bytes())
        .map_err(|e| ProvError::io(&e, format!("cannot write {}", dest.display())))?;
    std::io::Write::write_all(&mut file, b"\n")
        .map_err(|e| ProvError::io(&e, format!("cannot write {}", dest.display())))?;
    Ok(())
}

pub struct RenderInputs<'a> {
    pub operator_key: &'a Path,
    pub password_hash: &'a Path,
    pub out: &'a Path,
    pub hostname: Option<&'a str>,
    pub host_key: Option<&'a Path>,
    pub bootstrap: &'a str,
    pub product_hostname: Option<&'a str>,
}

/// Merge the public bootstrap with the private inputs and write the
/// exclusive output document.
pub fn render(source: &Path, inputs: &RenderInputs<'_>) -> Result<(), ProvError> {
    let key = regular(inputs.operator_key, false)?;
    let password = regular(inputs.password_hash, true)?;
    let key = key.trim();
    let password = password.trim();
    if !(key.starts_with("ssh-") || key.starts_with("ecdsa-") || key.starts_with("sk-"))
        || key.contains('\n')
        || !password.starts_with('$')
        || password.contains('\n')
    {
        return Err(ProvError::value(
            "provide a public SSH key and crypt(3) hash, not plaintext",
        ));
    }
    let mut config = public_config(source)?;
    if let Some(product) = inputs.product_hostname {
        if inputs.hostname.is_some() || !is_appliance_hostname(product) {
            return Err(ProvError::value(
                "valid appliance hostname or fixture hostname required, not both",
            ));
        }
    }
    if inputs.bootstrap == "minimal" {
        let entries = object_mut(&mut config)
            .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap must be an object"))?;
        let position = entries
            .iter()
            .rposition(|(k, _)| k == "systemd")
            .ok_or_else(|| ProvError::new(ProvKind::Key, "bootstrap has no systemd"))?;
        entries.remove(position);
        clear_files(&mut config)?;
    } else if inputs.bootstrap != "extensions" {
        return Err(ProvError::value("unknown bootstrap profile"));
    }
    let entries = object_mut(&mut config)
        .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap must be an object"))?;
    entries.push((
        "passwd".to_string(),
        JsonValue::Object(vec![(
            "users".to_string(),
            JsonValue::Array(vec![JsonValue::Object(vec![
                ("name".to_string(), str_value("root")),
                (
                    "ssh_authorized_keys".to_string(),
                    JsonValue::Array(vec![str_value(key)]),
                ),
                ("password_hash".to_string(), str_value(password)),
            ])]),
        )]),
    ));
    if let Some(hostname) = inputs.hostname {
        if !is_fixture_hostname(hostname) {
            return Err(ProvError::value(
                "fresh soda-native-* fixture hostname required",
            ));
        }
        push_file(
            &mut config,
            file_entry("/etc/hostname", 0o644, &format!("{hostname}\n")),
        )?;
    }
    if let Some(product) = inputs.product_hostname {
        push_file(
            &mut config,
            file_entry("/etc/hostname", 0o644, &format!("{product}\n")),
        )?;
    }
    if let Some(host_key) = inputs.host_key {
        let private_key = regular(host_key, true)?;
        let public = derive_host_public(host_key)?;
        push_file(
            &mut config,
            file_entry("/etc/ssh/ssh_host_ed25519_key", 0o600, &private_key),
        )?;
        push_file(
            &mut config,
            file_entry("/etc/ssh/ssh_host_ed25519_key.pub", 0o644, &public),
        )?;
    }
    write_exclusive(inputs.out, &dump_python(&config))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn hostnames_follow_the_script_regexes() {
        assert!(is_appliance_hostname("factory-01.lab.example"));
        assert!(is_appliance_hostname("a"));
        assert!(!is_appliance_hostname(""));
        assert!(!is_appliance_hostname("-lead.example"));
        assert!(!is_appliance_hostname("trail-.example"));
        assert!(!is_appliance_hostname("UPPER.example"));
        assert!(!is_appliance_hostname("under_score.example"));
        assert!(!is_appliance_hostname(".leading.example"));
        assert!(!is_appliance_hostname("trailing.example."));
        assert!(!is_appliance_hostname("double..dot"));
        assert!(!is_appliance_hostname(&"a".repeat(64)));
        assert!(is_appliance_hostname(&("a".repeat(63) + ".example")));
        assert!(!is_appliance_hostname(&"a".repeat(254)));
        assert!(is_fixture_hostname("soda-native-fixture"));
        assert!(is_fixture_hostname("soda-native-a"));
        assert!(!is_fixture_hostname("soda-native-"));
        assert!(!is_fixture_hostname("soda-native-UPPER"));
        assert!(!is_fixture_hostname("other-fixture"));
        assert!(!is_fixture_hostname(
            "soda-native-a-very-long-tail-that-keeps-going-past-forty-two"
        ));
    }

    #[test]
    fn dump_matches_json_indent_two() {
        let value = JsonValue::parse(
            "{\"b\": [1, {\"x\": true}, [], {}], \"a\": \"q\\\"\\n\\u0001~/\\u007f\\u00e9😀\", \"e\": {}, \"n\": null}",
        )
        .expect("parse");
        assert_eq!(
            dump_python(&value),
            "{\n  \"b\": [\n    1,\n    {\n      \"x\": true\n    },\n    [],\n    {}\n  ],\n  \"a\": \"q\\\"\\n\\u0001~/\\u007f\\u00e9\\ud83d\\ude00\",\n  \"e\": {},\n  \"n\": null\n}"
        );
    }

    #[test]
    fn dump_keeps_number_literals_and_key_order() {
        let value = JsonValue::parse("{\"mode\": 420, \"ratio\": 1e2}").expect("parse");
        assert_eq!(
            dump_python(&value),
            "{\n  \"mode\": 420,\n  \"ratio\": 1e2\n}"
        );
    }
}
