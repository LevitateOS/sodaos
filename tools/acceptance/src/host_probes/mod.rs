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
//! Success output is JSON with the same decoded values.

use std::io::Read;
use std::process::{Command, Stdio};

use crate::structured::Value as JsonValue;

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

/// Compact JSON rendering that preserves the constructed field order.
pub fn dumps(value: &JsonValue) -> String {
    serde_json::to_string(value).expect("host-probe JSON serializes")
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
