//! Upstream Tea binary + license staging (`scripts/fetch-tea.py`).

use std::path::{Path, PathBuf};
use std::time::Duration;

pub const RELEASES_API: &str = "https://gitea.com/api/v1/repos/gitea/tea/releases/latest";
pub const DL_BASE: &str = "https://dl.gitea.com/tea";
pub const LICENSE_BASE: &str = "https://gitea.com/gitea/tea/raw/tag";
pub const USER_AGENT: &str = "SodaOS-build";
const TIMEOUT: Duration = Duration::from_secs(60);
const BINARY_LIMIT: u64 = 64_000_000;
const META_LIMIT: u64 = 1_000_000;

/// Overridable endpoints so tests can point the fetcher at a fixture
/// server; production uses [`Endpoints::production`].
pub struct Endpoints {
    pub releases_api: String,
    pub dl_base: String,
    pub license_base: String,
}

impl Endpoints {
    pub fn production() -> Endpoints {
        Endpoints {
            releases_api: RELEASES_API.to_string(),
            dl_base: DL_BASE.to_string(),
            license_base: LICENSE_BASE.to_string(),
        }
    }
}

/// The script's default output rooted at the caller's working directory:
/// the release build always runs fetchers from the source root, so this
/// stages exactly where `ROOT/.artifacts/...` did.
pub fn default_out(arch: &str) -> Result<PathBuf, String> {
    let cwd = std::env::current_dir().map_err(|e| e.to_string())?;
    Ok(cwd
        .join(".artifacts/native")
        .join(arch)
        .join("project-tools"))
}

fn download(url: &str, limit: u64) -> Result<Vec<u8>, String> {
    let mut response = crate::fetch::http_get(url, USER_AGENT, TIMEOUT)
        .map_err(|e| format!("fetch {url}: {e}"))?;
    if !(200..300).contains(&response.status) {
        return Err(format!(
            "fetch {url}: unexpected HTTP status {}",
            response.status
        ));
    }
    crate::fetch::read_capped(&mut response.reader, limit)
}

/// `^v[0-9]+\.[0-9]+\.[0-9]+$` without a regex dependency.
fn valid_tag(tag: &str) -> bool {
    let body = match tag.strip_prefix('v') {
        Some(body) => body,
        None => return false,
    };
    let mut parts = body.split('.');
    match (parts.next(), parts.next(), parts.next(), parts.next()) {
        (Some(a), Some(b), Some(c), None) => {
            !a.is_empty()
                && !b.is_empty()
                && !c.is_empty()
                && a.bytes().all(|b| b.is_ascii_digit())
                && b.bytes().all(|b| b.is_ascii_digit())
                && c.bytes().all(|b| b.is_ascii_digit())
        }
        _ => false,
    }
}

fn latest_tag(endpoints: &Endpoints) -> Result<String, String> {
    let url = &endpoints.releases_api;
    let body = download(url, META_LIMIT)?;
    let text = std::str::from_utf8(&body).map_err(|e| format!("fetch {url}: {e}"))?;
    let release = soda_json::JsonValue::parse(text)
        .map_err(|_| format!("fetch {url}: invalid release metadata"))?;
    let tag = release
        .get("tag_name")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    if !valid_tag(tag) {
        return Err("Tea latest release is not a version tag".to_string());
    }
    Ok(tag.to_string())
}

/// The expected digest for `filename`; the last matching line wins, like
/// the owner's loop without a `break`.
fn checksums_for(text: &str, filename: &str) -> Option<String> {
    let mut expected = None;
    for line in text.split('\n') {
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() == 2 && parts[1] == filename {
            expected = Some(parts[0].to_string());
        }
    }
    expected
}

fn valid_elf64(body: &[u8], machine: u16) -> bool {
    if body.len() < 64 || body[..6] != [0x7f, b'E', b'L', b'F', 2, 1] {
        return false;
    }
    u16::from_le_bytes([body[18], body[19]]) == machine
}

/// Stage Tea for `arch` into `out`; returns the stdout line.
pub fn fetch(arch: &str, out: &Path, endpoints: &Endpoints) -> Result<String, String> {
    if arch != "x86_64" {
        return Err("Tea staging supports x86_64 only".to_string());
    }
    let binary = out.join("bin/tea");
    let license_file = out.join("licenses/tea/LICENSE");
    if std::fs::symlink_metadata(&binary).is_ok()
        || std::fs::symlink_metadata(&license_file).is_ok()
    {
        return Err("Tea output already exists; select a fresh output directory".to_string());
    }
    let tag = latest_tag(endpoints)?;
    let version = &tag[1..];
    let filename = format!("tea-{version}-linux-amd64");
    let base = format!("{}/{version}", endpoints.dl_base);
    let sums_url = format!("{base}/checksums.txt");
    let sums = download(&sums_url, META_LIMIT)?;
    let sums_text = std::str::from_utf8(&sums).map_err(|e| format!("fetch {sums_url}: {e}"))?;
    let expected = checksums_for(sums_text, &filename)
        .ok_or_else(|| "Tea checksums omit the requested archive".to_string())?;
    let body = download(&format!("{base}/{filename}"), BINARY_LIMIT)?;
    if body.len() as u64 > BINARY_LIMIT || crate::fetch::sha256_hex(&body) != expected {
        return Err("Tea binary checksum mismatch".to_string());
    }
    if !valid_elf64(&body, 62) {
        return Err("Tea binary is not ELF64 for the requested architecture".to_string());
    }
    let license = download(
        &format!("{}/{tag}/LICENSE", endpoints.license_base),
        META_LIMIT,
    )?;
    if license.is_empty() {
        return Err("Tea license download is empty".to_string());
    }
    std::fs::create_dir_all(out).map_err(|e| e.to_string())?;
    match std::fs::create_dir(out.join("bin")) {
        Ok(()) => {}
        Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => {}
        Err(e) => return Err(e.to_string()),
    }
    std::fs::write(&binary, &body).map_err(|e| e.to_string())?;
    crate::fetch::chmod(&binary, 0o755)?;
    let license_dir = out.join("licenses/tea");
    std::fs::create_dir_all(&license_dir).map_err(|e| e.to_string())?;
    std::fs::write(&license_file, &license).map_err(|e| e.to_string())?;
    crate::fetch::chmod(&license_file, 0o644)?;
    Ok(format!(
        "Upstream Tea {version} ({arch}) staged at {}",
        out.display()
    ))
}

#[cfg(test)]
mod tests;
