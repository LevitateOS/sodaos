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

use std::collections::HashMap;
use std::io::Read;
use std::process::{Command, Stdio};

use soda_json::JsonValue;

use crate::sha256::{self, Sha256};

mod content;
mod deployments;
mod listeners;

pub use content::host_content;
pub use deployments::{host_deployments, operator_tailscale};
pub use listeners::host_listeners;

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

/// Core browser origins: the `forgejo-advertisement.sh` `origins()`
/// helper.
pub fn forgejo_origins() -> Result<String, ProbeFailure> {
    let cfg = parse_json(
        &read_text("/etc/soda/dashboard.json", "read dashboard.json")?,
        "parse dashboard.json",
    )?;
    Ok(dumps(&JsonValue::Object(vec![
        (
            "forgejo_internal_url".to_string(),
            JsonValue::Str(
                get_str(&cfg, "forgejo_internal_url", "parse dashboard.json")?.to_string(),
            ),
        ),
        (
            "forgejo_url".to_string(),
            JsonValue::Str(get_str(&cfg, "forgejo_url", "parse dashboard.json")?.to_string()),
        ),
    ])) + "\n")
}

/// Approved running Tailnet gate: the `forgejo-advertisement.sh`
/// inline `python3 -c` over stdin. Silent on success.
pub fn forgejo_tailnet(stdin: &str) -> Result<String, ProbeFailure> {
    let doc = parse_json(stdin, "parse tailscale status")?;
    let running = doc.get("BackendState").and_then(|v| v.as_str()) == Some("Running");
    let expired = match doc.get("Self") {
        None => false,
        Some(value) if !is_truthy(value) => false,
        Some(JsonValue::Object(_)) => doc
            .get("Self")
            .and_then(|v| v.get("Expired"))
            .is_some_and(is_truthy),
        Some(_) => return Err(ProbeFailure::failed("validate tailscale status")),
    };
    if !running || expired {
        return Err(ProbeFailure::exit(
            "approved running Tailnet required; no enrollment is performed",
        ));
    }
    Ok(String::new())
}

/// Listener-checked Git SSH advertisement: the final
/// `forgejo-advertisement.sh` Python block.
pub fn forgejo_advertisement() -> Result<String, ProbeFailure> {
    let mut values = HashMap::new();
    for line in read_text("/etc/soda/forgejo.env", "read forgejo.env")?.lines() {
        if line.contains('=') && !line.starts_with('#') {
            let (key, value) = line.split_once('=').unwrap();
            values.insert(key.to_string(), value.to_string());
        }
    }
    let host = values
        .get("FORGEJO__server__SSH_DOMAIN")
        .ok_or_else(|| ProbeFailure::failed("parse forgejo.env"))?;
    let port = values
        .get("FORGEJO__server__SSH_PORT")
        .ok_or_else(|| ProbeFailure::failed("parse forgejo.env"))?;
    let status = parse_json(
        &run_output(
            &["/usr/bin/tailscale", "status", "--json"],
            "read tailscale status",
        )?,
        "parse tailscale status",
    )?;
    let on_tailnet = match status.get("TailscaleIPs") {
        Some(JsonValue::Array(ips)) => ips.iter().any(|ip| ip.as_str() == Some(host.as_str())),
        _ => false,
    };
    if !on_tailnet {
        return Err(ProbeFailure::exit(
            "advertisement does not match a current Tailnet address",
        ));
    }
    let listeners = podman(&["port", "soda-forgejo", "22/tcp"], "read published ports")?;
    if !listeners
        .lines()
        .any(|line| line == format!("{host}:{port}"))
    {
        return Err(ProbeFailure::exit(
            "advertised endpoint is not a real selected private listener",
        ));
    }
    Ok(format!(
        "Listener-checked Git SSH advertisement: {host} {port}\nCore browser origins unchanged. Probe the pinned endpoint from the intended client; this is not routed Git authentication proof.\n"
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dumps_keeps_order_and_separators() {
        let value = JsonValue::Object(vec![
            ("b".to_string(), JsonValue::Number("1".to_string())),
            (
                "a".to_string(),
                JsonValue::Array(vec![JsonValue::Bool(true), JsonValue::Null]),
            ),
        ]);
        assert_eq!(dumps(&value), r#"{"b": 1, "a": [true, null]}"#);
        let value = JsonValue::Object(vec![(
            "e".to_string(),
            JsonValue::Str("é\t\"q\"".to_string()),
        )]);
        assert_eq!(dumps(&value), "{\"e\": \"\\u00e9\\t\\\"q\\\"\"}");
    }

    #[test]
    fn origins_split_hosts_and_ports() {
        assert_eq!(
            split_origin("https://example.test").unwrap(),
            ("example.test".to_string(), 443)
        );
        assert_eq!(
            split_origin("https://Example.Test:8443/x").unwrap(),
            ("example.test".to_string(), 8443)
        );
        assert_eq!(
            split_origin("https://user@[::1]:2222").unwrap(),
            ("::1".to_string(), 2222)
        );
        assert_eq!(
            split_origin("https://[FE80::1]").unwrap(),
            ("fe80::1".to_string(), 443)
        );
        assert_eq!(
            split_origin("https://example.test:0").unwrap(),
            ("example.test".to_string(), 443)
        );
        assert!(split_origin("http://example.test").is_err());
        assert!(split_origin("https://").is_err());
        assert!(split_origin("https://host:notaport").is_err());
    }

    #[test]
    fn private_binds_match_plain_lan_shapes() {
        for ip in [
            "10.1.2.3",
            "100.100.0.1",
            "172.20.0.1",
            "192.168.0.1",
            "fc00::1",
        ] {
            assert!(is_private_bind(ip.parse().unwrap()), "{ip}");
        }
        for ip in ["8.8.8.8", "0.0.0.0", "::", "127.0.0.1", "fe80::1"] {
            assert!(!is_private_bind(ip.parse().unwrap()), "{ip}");
        }
    }

    #[test]
    fn listeners_parse_and_bind() {
        let rows = "LISTEN 0 128 127.0.0.1:9090 0.0.0.0:*\nLISTEN 0 128 [::1]:9090 [::]:*\n";
        let listeners = parse_listeners(rows).unwrap();
        assert_eq!(listeners.len(), 2);
        bound(&listeners, "127.0.0.1", 9090).unwrap();
        assert_eq!(
            bound(&listeners, "127.0.0.1", 3000).unwrap_err(),
            ProbeFailure::exit("required service binding missing")
        );
        let rows = "LISTEN 0 128 0.0.0.0:9090 0.0.0.0:*\n";
        let listeners = parse_listeners(rows).unwrap();
        assert_eq!(
            bound(&listeners, "127.0.0.1", 9090).unwrap_err(),
            ProbeFailure::exit("required service binding missing")
        );
        assert!(parse_listeners("short row\n").is_err());
    }

    #[test]
    fn env_files_skip_empty_and_reject_bare_words() {
        let env = parse_env_file("A=1\n\nB=x=y\n").unwrap();
        assert_eq!(env.get("A").unwrap(), "1");
        assert_eq!(env.get("B").unwrap(), "x=y");
        assert_eq!(
            parse_env_file("BARE\n").unwrap_err(),
            ProbeFailure::failed("parse proxy.env")
        );
    }

    #[test]
    fn tailscale_summaries_shape() {
        let out = operator_tailscale(r#"{"BackendState": "Running", "Self": {"Expired": false}}"#)
            .unwrap();
        assert_eq!(out, "{\"BackendState\": \"Running\", \"Expired\": false}\n");
        let out = operator_tailscale(r#"{"BackendState": "Stopped"}"#).unwrap();
        assert_eq!(out, "{\"BackendState\": \"Stopped\", \"Expired\": null}\n");
        assert!(operator_tailscale(r#"{"BackendState": 7}"#).is_err());
        assert_eq!(
            forgejo_tailnet(r#"{"BackendState": "Running", "Self": {"Expired": false}}"#).unwrap(),
            ""
        );
        assert_eq!(
            forgejo_tailnet(r#"{"BackendState": "NoState"}"#).unwrap_err(),
            ProbeFailure::exit("approved running Tailnet required; no enrollment is performed")
        );
    }

    #[test]
    fn deployments_summarize_in_order() {
        let out = host_deployments(r#"{"deployments": [{"booted": true, "version": "41", "checksum": "abc", "requested-packages": ["x"], "extra": 1}]}"#)
            .unwrap();
        assert_eq!(out, "[{\"booted\": true, \"version\": \"41\", \"checksum\": \"abc\", \"requested-packages\": [\"x\"]}]\n");
        assert!(host_deployments(r#"{"deployments": [7]}"#).is_err());
    }
}
