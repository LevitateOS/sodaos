//! Host-side answers for the kept installed shell skeletons, mirroring
//! the embedded Python of `tests/installed/host.sh`,
//! `tests/installed/operator.sh`, and
//! `tests/installed/forgejo-advertisement.sh`.
//!
//! The shells keep every gate, package, unit, ownership, and label
//! check; only the `python3` blocks move here, called by absolute
//! path. Verdict messages (`SystemExit` and assertion texts)
//! print verbatim; operational failures (I/O, JSON, subprocess)
//! collapse Python tracebacks to one `host probe failed:` line.
//! Success output is byte-identical.

use std::io::Read;
use std::process::{Command, Stdio};

use soda_json::JsonValue;

use crate::sha256::{self, Digest, Sha256};

mod content;
mod deployments;
mod forgejo;
mod listeners;
mod tailnet;

pub use content::host_content;
pub use deployments::{host_deployments, operator_tailscale};
pub use forgejo::{forgejo_advertisement, forgejo_origins};
pub use listeners::host_listeners;
pub use tailnet::forgejo_tailnet;

#[cfg(test)]
use listeners::{bound, is_private_bind, parse_env_file, parse_listeners, split_origin};

/// Host-probe failure: a verdict message or an operational failure.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ProbeFailure {
    /// Verdict message, printed verbatim like `SystemExit`.
    Exit(String),
    /// Operational failure; the fixed operation name only.
    Failed(&'static str),
}

impl ProbeFailure {
    fn exit(message: impl Into<String>) -> ProbeFailure {
        ProbeFailure::Exit(message.into())
    }

    fn failed(op: &'static str) -> ProbeFailure {
        ProbeFailure::Failed(op)
    }
}

impl std::fmt::Display for ProbeFailure {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ProbeFailure::Exit(message) => f.write_str(message),
            ProbeFailure::Failed(op) => write!(f, "host probe failed: {op}"),
        }
    }
}

impl std::error::Error for ProbeFailure {}

/// Python `json.dumps` rendering: insertion order, `(', ', ': ')`
/// separators, ASCII-only output with lowercase `\u` escapes.
pub fn dumps(value: &JsonValue) -> String {
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
                out.push('{');
                for (index, (key, item)) in entries.iter().enumerate() {
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

fn read_file(path: &str, op: &'static str) -> Result<Vec<u8>, ProbeFailure> {
    std::fs::read(path).map_err(|_| ProbeFailure::failed(op))
}

fn read_text(path: &str, op: &'static str) -> Result<String, ProbeFailure> {
    let bytes = read_file(path, op)?;
    String::from_utf8(bytes).map_err(|_| ProbeFailure::failed(op))
}

fn parse_json(text: &str, op: &'static str) -> Result<JsonValue, ProbeFailure> {
    JsonValue::parse(text).map_err(|_| ProbeFailure::failed(op))
}

fn get_str<'a>(value: &'a JsonValue, key: &str, op: &'static str) -> Result<&'a str, ProbeFailure> {
    value
        .get(key)
        .and_then(|v| v.as_str())
        .ok_or_else(|| ProbeFailure::failed(op))
}

/// Python truthiness for JSON values, for the `(d.get("Self") or {})`
/// shapes.
fn is_truthy(value: &JsonValue) -> bool {
    match value {
        JsonValue::Null => false,
        JsonValue::Bool(b) => *b,
        JsonValue::Number(raw) => raw.parse::<f64>().map(|n| n != 0.0).unwrap_or(true),
        JsonValue::Str(s) => !s.is_empty(),
        JsonValue::Array(items) => !items.is_empty(),
        JsonValue::Object(entries) => !entries.is_empty(),
    }
}

/// Run a probe subprocess to completion, returning stdout. Failures
/// carry the fixed operation name, never argv or output.
fn run_output(argv: &[&str], op: &'static str) -> Result<String, ProbeFailure> {
    let output = Command::new(argv[0])
        .args(&argv[1..])
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::null())
        .output()
        .map_err(|_| ProbeFailure::failed(op))?;
    if !output.status.success() {
        return Err(ProbeFailure::failed(op));
    }
    String::from_utf8(output.stdout).map_err(|_| ProbeFailure::failed(op))
}

fn sha256_file(path: &std::path::Path, op: &'static str) -> Result<String, ProbeFailure> {
    let mut file = std::fs::File::open(path).map_err(|_| ProbeFailure::failed(op))?;
    let mut digest = Sha256::new();
    let mut chunk = vec![0u8; 1024 * 1024];
    loop {
        let n = file
            .read(&mut chunk)
            .map_err(|_| ProbeFailure::failed(op))?;
        if n == 0 {
            break;
        }
        digest.update(&chunk[..n]);
    }
    Ok(sha256::hex_lower(&digest.finalize()))
}

fn podman(args: &[&str], op: &'static str) -> Result<String, ProbeFailure> {
    let mut argv = vec!["podman", "--remote=false"];
    argv.extend(args.iter().copied());
    run_output(&argv, op)
}

#[cfg(test)]
mod tests;
