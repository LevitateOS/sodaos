//! `prepare.go` package-input readers + `complete.go` RPM inventory guard.

use crate::error::Error;
use crate::jsonio;

const INSTALL_PREFIX: &str = "ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive ";

/// PackageInputs uses the current first-install owner rather than maintaining
/// a second host package list. Reject changes in its command shape for
/// explicit review. The thin host carries no Tailscale repository or RPM:
/// either fails closed here.
pub fn package_inputs(data: &[u8]) -> Result<Vec<String>, Error> {
    let text = std::str::from_utf8(data).map_err(|e| Error::msg(e.to_string()))?;
    let value: serde_json::Value =
        jsonio::parse(text).map_err(|e| Error::msg(format!("invalid package inputs: {}", e.0)))?;
    let empty = serde_json::Value::Null;
    let storage = value.get("storage").unwrap_or(&empty);
    let files = storage.get("files");
    if let Some(serde_json::Value::Array(entries)) = files {
        for file in entries {
            let path = file.get("path").and_then(|v| v.as_str()).unwrap_or("");
            if path == "/etc/yum.repos.d/tailscale.repo" {
                return Err(Error::msg("unexpected Tailscale repository"));
            }
        }
    }
    let systemd = value.get("systemd").unwrap_or(&empty);
    let mut packages: Option<Vec<String>> = None;
    let mut seen: Vec<String> = Vec::new();
    if let Some(serde_json::Value::Array(units)) = systemd.get("units") {
        for unit in units {
            let name = unit.get("name").and_then(|v| v.as_str()).unwrap_or("");
            if name != "soda-extensions.service" {
                continue;
            }
            let contents = unit.get("contents").and_then(|v| v.as_str()).unwrap_or("");
            for line in contents.split('\n') {
                if !line.starts_with("ExecStart=") {
                    continue;
                }
                packages = Some(parse_install_packages(line, packages.is_some(), &mut seen)?);
            }
        }
    }
    match packages {
        Some(packages) if !packages.is_empty() => {
            if packages.iter().any(|name| name == "tailscale") {
                return Err(Error::msg("unexpected Tailscale package"));
            }
            Ok(packages)
        }
        _ => Err(Error::msg("missing host package inputs")),
    }
}

fn parse_install_packages(
    line: &str,
    existing: bool,
    seen: &mut Vec<String>,
) -> Result<Vec<String>, Error> {
    if existing || !line.starts_with(INSTALL_PREFIX) {
        return Err(Error::msg("unexpected package installation command"));
    }
    let mut packages = Vec::new();
    for name in line[INSTALL_PREFIX.len()..].split_whitespace() {
        if !is_package_name(name) || seen.contains(&name.to_string()) {
            return Err(Error::msg("invalid or duplicate host package"));
        }
        seen.push(name.to_string());
        packages.push(name.to_string());
    }
    Ok(packages)
}

fn is_package_name(name: &str) -> bool {
    // `^[a-z0-9][a-z0-9+.-]*$`
    let mut bytes = name.bytes();
    match bytes.next() {
        Some(b) if b.is_ascii_lowercase() || b.is_ascii_digit() => {}
        _ => return false,
    }
    bytes.all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || matches!(b, b'+' | b'.' | b'-'))
}

pub use soda_build_tools::reader::rpm::valid_rpm_inventory;
