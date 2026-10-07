use crate::structured::Value as JsonValue;

use crate::command::Remote;
use crate::error::Error;
use crate::files;

use super::{decode_ignition_files, inline_data};

/// Lenient Ignition parse failure. Callers map it to their own input
/// message, like Go's per-caller `Unmarshal` checks.
#[derive(Debug)]
pub struct IgnitionError;

/// Parse a lenient Ignition document. Callers map failure to their own
/// input message, like Go's per-caller `Unmarshal` checks.
pub fn parse_ignition(data: &[u8]) -> Result<JsonValue, IgnitionError> {
    let text = std::str::from_utf8(data).map_err(|_| IgnitionError)?;
    JsonValue::parse(text).map_err(|_| IgnitionError)
}

fn is_fixture_trust_file(path: &str) -> bool {
    path == "/etc/hostname" || path == "/etc/ssh/ssh_host_ed25519_key"
}

/// Go's `strings.TrimSpace` is Unicode White Space, which is exactly
/// Rust's `str::trim`.
fn trim_space(text: &str) -> &str {
    text.trim()
}

fn run_ssh_keygen(args: &[&str]) -> Result<String, Error> {
    let output = std::process::Command::new("ssh-keygen")
        .args(args)
        .stdin(std::process::Stdio::null())
        .output()
        .map_err(Error::from)?;
    if !output.status.success() {
        return Err(Error::msg("ssh-keygen failed"));
    }
    String::from_utf8(output.stdout).map_err(|_| Error::msg("ssh-keygen failed"))
}

/// `known_hosts` lookup form for an explicit host and port.
fn known_hosts_query(host: &str, port: i64) -> String {
    if port == 22 {
        return host.to_string();
    }
    if host.starts_with('[') {
        return format!("{host}:{port}");
    }
    format!("[{host}]:{port}")
}

/// Verify the supplied trust root against the selected private bootstrap
/// before starting anything. Mirrors `VerifyFixtureTrust`: the fixture
/// hostname must match and the embedded host key must match the pinned
/// machine `known_hosts`. No keyscan, trust replacement, or product key
/// API is used.
pub fn verify_fixture_trust(path: &str, name: &str, remote: &Remote) -> Result<(), Error> {
    let data = files::private_file(path)?;
    let parsed = parse_ignition(&data).map_err(|_| Error::msg("invalid private Ignition"))?;
    let trust_parse = std::process::Command::new("ssh-keygen")
        .args(["-l", "-f", &remote.known_hosts])
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .status()
        .map_err(Error::from)?;
    if !trust_parse.success() {
        return Err(Error::msg("invalid pinned known_hosts"));
    }
    let files =
        decode_ignition_files(&parsed).map_err(|_| Error::msg("invalid private Ignition"))?;
    let mut matched = false;
    let mut hostname = false;
    for file in &files {
        if !is_fixture_trust_file(&file.path) {
            continue;
        }
        let body = inline_data(&file.source, &file.compression)?;
        if file.path == "/etc/hostname" {
            hostname = trim_space(&String::from_utf8_lossy(&body)) == name;
            continue;
        }
        verify_pinned_host_key(
            &body,
            &known_hosts_query(&remote.host, remote.port),
            &remote.known_hosts,
        )?;
        matched = true;
    }
    if !matched || !hostname {
        return Err(Error::msg(
            "matching fixture hostname and pinned Ed25519 host key required in Ignition",
        ));
    }
    Ok(())
}

fn verify_pinned_host_key(body: &[u8], query: &str, known_hosts: &str) -> Result<(), Error> {
    let scratch = files::TempDir::new("soda-trust")?;
    let priv_path = scratch.join("host_key");
    std::fs::write(&priv_path, body).map_err(Error::from)?;
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(&priv_path, std::fs::Permissions::from_mode(0o600))
        .map_err(Error::from)?;
    let derived = run_ssh_keygen(&["-y", "-P", "", "-f", &priv_path.to_string_lossy()])
        .map_err(|_| Error::msg("invalid per-instance host key"))?;
    let mut parts = derived.split_whitespace();
    let (key_type, key_b64) = (parts.next().unwrap_or(""), parts.next().unwrap_or(""));
    let found = run_ssh_keygen(&["-F", query, "-f", known_hosts]).unwrap_or_default();
    for line in found.lines() {
        if line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split_whitespace().collect();
        if fields.len() >= 3 && fields[1] == key_type && fields[2] == key_b64 {
            return Ok(());
        }
    }
    Err(Error::msg(
        "ignition host key does not match pinned management trust",
    ))
}
