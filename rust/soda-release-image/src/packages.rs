//! `prepare.go` package-input readers + `complete.go` RPM inventory guard.

use soda_json::JsonValue;

use crate::error::Error;
use crate::jsonio;

const TAILSCALE_REPO_URL: &str = "https://pkgs.tailscale.com/stable/fedora/tailscale.repo";
const INSTALL_PREFIX: &str = "ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive ";

/// PackageInputs uses the current first-install owner rather than maintaining
/// a second host package list. Reject changes in its command shape for
/// explicit review.
pub fn package_inputs(data: &[u8]) -> Result<(Vec<String>, String), Error> {
    let text = std::str::from_utf8(data).map_err(|e| Error::msg(e.to_string()))?;
    let value =
        jsonio::parse(text).map_err(|e| Error::msg(format!("invalid package inputs: {}", e.0)))?;
    let empty = JsonValue::Null;
    let storage = value.get("storage").unwrap_or(&empty);
    let files = storage.get("files");
    let mut repo = String::new();
    if let Some(JsonValue::Array(entries)) = files {
        for file in entries {
            let path = file.get("path").and_then(|v| v.as_str()).unwrap_or("");
            if path != "/etc/yum.repos.d/tailscale.repo" {
                continue;
            }
            let source = file
                .get("contents")
                .and_then(|c| c.get("source"))
                .and_then(|v| v.as_str())
                .unwrap_or("");
            if !repo.is_empty() || source != TAILSCALE_REPO_URL {
                return Err(Error::msg("unexpected Tailscale repository"));
            }
            repo = source.to_string();
        }
    }
    let systemd = value.get("systemd").unwrap_or(&empty);
    let mut packages: Option<Vec<String>> = None;
    let mut seen: Vec<String> = Vec::new();
    if let Some(JsonValue::Array(units)) = systemd.get("units") {
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
        Some(packages) if !packages.is_empty() && !repo.is_empty() => Ok((packages, repo)),
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

/// validRPMInventory guards the recorded bill-of-materials shape: sorted,
/// unique NAME EPOCH:VERSION-RELEASE.ARCH lines. It validates what the build
/// observed; it never pins what a future build must install.
pub fn valid_rpm_inventory(lines: &[String]) -> Result<(), Error> {
    // slices.IsSorted is non-decreasing; duplicates fail the shape check below.
    if lines.windows(2).any(|w| w[0] > w[1]) {
        return Err(Error::msg("sorted RPM inventory required"));
    }
    let mut seen: Vec<&str> = Vec::new();
    for line in lines {
        if !is_rpm_line(line) || seen.contains(&line.as_str()) {
            return Err(Error::msg("invalid recorded RPM inventory"));
        }
        seen.push(line);
    }
    Ok(())
}

fn is_rpm_line(line: &str) -> bool {
    // `^[a-zA-Z0-9][a-zA-Z0-9+._-]* [0-9]+:[a-zA-Z0-9+._~^-]+$`
    let (name, evr) = match line.split_once(' ') {
        Some(pair) => pair,
        None => return false,
    };
    if evr.contains(' ') {
        return false;
    }
    let mut name_bytes = name.bytes();
    match name_bytes.next() {
        Some(b) if b.is_ascii_alphanumeric() => {}
        _ => return false,
    }
    if !name_bytes.all(|b| b.is_ascii_alphanumeric() || matches!(b, b'+' | b'.' | b'_' | b'-')) {
        return false;
    }
    let (epoch, version) = match evr.split_once(':') {
        Some(pair) => pair,
        None => return false,
    };
    if epoch.is_empty()
        || !epoch.bytes().all(|b| b.is_ascii_digit())
        || version.is_empty()
        || !version.bytes().all(|b| {
            b.is_ascii_alphanumeric() || matches!(b, b'+' | b'.' | b'_' | b'~' | b'^' | b'-')
        })
    {
        return false;
    }
    true
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_package_input_failures() {
        // Oracle: Go TestPackageInputFailures vectors.
        assert!(package_inputs(b"not json").is_err());
        assert_eq!(
            package_inputs(b"{}").unwrap_err().0,
            "missing host package inputs"
        );
        let no_repo = br#"{"storage":{"files":[]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo\n"}]}}"#;
        assert_eq!(
            package_inputs(no_repo).unwrap_err().0,
            "missing host package inputs"
        );
        let bad_cmd = br#"{"storage":{"files":[{"path":"/etc/yum.repos.d/tailscale.repo","contents":{"source":"https://pkgs.tailscale.com/stable/fedora/tailscale.repo"}}]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/bin/false\n"}]}}"#;
        assert_eq!(
            package_inputs(bad_cmd).unwrap_err().0,
            "unexpected package installation command"
        );
        let dup = br#"{"storage":{"files":[{"path":"/etc/yum.repos.d/tailscale.repo","contents":{"source":"https://pkgs.tailscale.com/stable/fedora/tailscale.repo"}}]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo foo\n"}]}}"#;
        assert_eq!(
            package_inputs(dup).unwrap_err().0,
            "invalid or duplicate host package"
        );
        let good = br#"{"storage":{"files":[{"path":"/etc/yum.repos.d/tailscale.repo","contents":{"source":"https://pkgs.tailscale.com/stable/fedora/tailscale.repo"}}]},"systemd":{"units":[{"name":"other.service","contents":"ExecStart=x"},{"name":"soda-extensions.service","contents":"[Service]\nExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo bar-1.0\n"}]}}"#;
        let (packages, repo) = package_inputs(good).unwrap();
        assert_eq!(packages, vec!["foo".to_string(), "bar-1.0".to_string()]);
        assert_eq!(repo, TAILSCALE_REPO_URL);
    }

    #[test]
    fn oracle_rpm_inventory_shape() {
        assert!(valid_rpm_inventory(&[
            "bash 0:5.1.8-1.fc38.x86_64".to_string(),
            "rpm-ostree 0:2023.1-1.fc38.x86_64".to_string(),
        ])
        .is_ok());
        assert_eq!(
            valid_rpm_inventory(&["b 0:1-1.x".to_string(), "a 0:1-1.x".to_string()])
                .unwrap_err()
                .0,
            "sorted RPM inventory required"
        );
        assert_eq!(
            valid_rpm_inventory(&["nope".to_string()]).unwrap_err().0,
            "invalid recorded RPM inventory"
        );
    }
}
