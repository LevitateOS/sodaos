//! `soda-artifacts check-fountain-pin`: answer whether a published Fountain
//! release serves the pinned Fountain source revision before any build input
//! freezes. Read-only; it admits nothing into a build and writes nothing.
//!
//! The pin file names the upstream repository and the exact source revision
//! (`fountain-source.json`). The check lists that repository's releases,
//! downloads each `release.json` asset, and reports the first release whose
//! `source_commit` equals the pin. Plain HTTP is refused everywhere except
//! loopback, so hermetic fixtures keep working without weakening lookups.

use std::io::Read;
use std::time::Duration;

use soda_release_build::http::{HttpTransport, UreqTransport};

const REQUEST_TIMEOUT: Duration = Duration::from_secs(30);
const BODY_CAP: u64 = 262_144;
const MAX_REDIRECTS: usize = 5;

#[derive(serde::Deserialize)]
struct PinFile {
    repository: String,
    revision: String,
}

#[derive(serde::Deserialize)]
struct ReleaseAsset {
    name: String,
    browser_download_url: String,
}

#[derive(serde::Deserialize)]
struct Release {
    tag_name: String,
    assets: Vec<ReleaseAsset>,
}

#[derive(serde::Deserialize)]
struct ReleaseMetadata {
    version: Option<String>,
    source_commit: String,
    image: Option<String>,
}

fn is_hex_revision(value: &str) -> bool {
    value.len() == 40
        && value
            .bytes()
            .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b) || (b'A'..=b'F').contains(&b))
}

fn is_repo_path(value: &str) -> bool {
    let mut parts = value.split('/');
    let ok = |part: &str| {
        !part.is_empty()
            && part
                .bytes()
                .all(|b| b.is_ascii_alphanumeric() || b == b'.' || b == b'-' || b == b'_')
    };
    match (parts.next(), parts.next(), parts.next()) {
        (Some(owner), Some(name), None) => ok(owner) && ok(name),
        _ => false,
    }
}

fn read_pin_file(path: &str) -> Result<(String, String), String> {
    let bytes = std::fs::read(path).map_err(|e| e.to_string())?;
    let pin: PinFile =
        serde_json::from_slice(&bytes).map_err(|_| "invalid Fountain pin file".to_owned())?;
    if !is_repo_path(&pin.repository) {
        return Err("Fountain pin file must name an owner/name repository".to_owned());
    }
    if !is_hex_revision(&pin.revision) {
        return Err("Fountain pin file must pin a 40-character revision".to_owned());
    }
    Ok((pin.repository, pin.revision))
}

fn url_host(url: &str) -> Option<String> {
    let rest = url.split_once("://")?.1;
    let authority = rest.split(['/', '?', '#']).next()?;
    let host = authority.rsplit('@').next()?;
    let host = host
        .strip_prefix('[')
        .and_then(|h| h.split(']').next())
        .unwrap_or_else(|| host.split(':').next().unwrap_or(host));
    Some(host.to_ascii_lowercase())
}

/// Plain HTTP reaches only loopback fixtures; everything else must be HTTPS.
fn admit_url(url: &str) -> Result<(), String> {
    if url.starts_with("https://") {
        return Ok(());
    }
    if url.starts_with("http://") {
        if let Some(host) = url_host(url) {
            if host == "localhost" || host == "127.0.0.1" || host == "::1" {
                return Ok(());
            }
        }
    }
    Err(format!("refusing non-HTTPS Fountain lookup {url}"))
}

fn read_capped(body: Box<dyn Read>, url: &str) -> Result<Vec<u8>, String> {
    let mut bytes = Vec::new();
    body.take(BODY_CAP + 1)
        .read_to_end(&mut bytes)
        .map_err(|e| format!("Fountain lookup failed for {url}: {e}"))?;
    if bytes.len() as u64 > BODY_CAP {
        return Err(format!("Fountain lookup response too large: {url}"));
    }
    Ok(bytes)
}

fn get_document(
    transport: &UreqTransport,
    mut url: String,
    accept: Option<&str>,
) -> Result<Vec<u8>, String> {
    for _ in 0..=MAX_REDIRECTS {
        admit_url(&url)?;
        let response = transport
            .get(&url, accept, REQUEST_TIMEOUT)
            .map_err(|e| format!("Fountain lookup failed for {url}: {e}"))?;
        match response.status {
            200 => return read_capped(response.body, &url),
            301 | 302 | 303 | 307 | 308 => {
                let location = response
                    .location
                    .ok_or_else(|| format!("Fountain lookup redirect without location: {url}"))?;
                url = if location.contains("://") {
                    location
                } else if let Some(base) = url.split_once("//").map(|(_, rest)| rest) {
                    let authority = base.split('/').next().unwrap_or(base);
                    let scheme = if url.starts_with("http://") {
                        "http"
                    } else {
                        "https"
                    };
                    format!("{scheme}://{authority}{location}")
                } else {
                    return Err(format!("Fountain lookup redirect failed: {url}"));
                };
            }
            _ => {
                return Err(format!(
                    "Fountain lookup failed for {url}: {}",
                    response.status_line()
                ))
            }
        }
    }
    Err(format!("Fountain lookup redirected too often: {url}"))
}

fn releases_url(api_base: &str, repository: &str) -> Result<String, String> {
    let base = api_base.trim_end_matches('/');
    let base = if base.is_empty() {
        "https://api.github.com"
    } else {
        base
    };
    let url = format!("{base}/repos/{repository}/releases?per_page=100");
    admit_url(&url)?;
    Ok(url)
}

/// Answer which published release serves `revision`, or explain the gap.
/// Returns the human report on success.
pub fn check_fountain_pin(pin_file: &str, api_base: &str) -> Result<String, String> {
    let transport = UreqTransport;
    let (repository, revision) = read_pin_file(pin_file)?;
    let url = releases_url(api_base, &repository)?;
    let bytes = get_document(&transport, url, Some("application/vnd.github+json"))?;
    let releases: Vec<Release> =
        serde_json::from_slice(&bytes).map_err(|_| "invalid Fountain release list".to_owned())?;
    let mut readable = 0usize;
    for release in &releases {
        let asset = release
            .assets
            .iter()
            .find(|asset| asset.name == "release.json");
        let Some(asset) = asset else { continue };
        let Ok(meta_bytes) = get_document(&transport, asset.browser_download_url.clone(), None)
        else {
            continue;
        };
        let Ok(meta): Result<ReleaseMetadata, _> = serde_json::from_slice(&meta_bytes) else {
            continue;
        };
        readable += 1;
        if meta.source_commit == revision {
            let version = meta.version.unwrap_or_else(|| release.tag_name.clone());
            let mut report = format!(
                "Fountain {version} serves pin {revision}\nrelease.json: {}",
                asset.browser_download_url
            );
            if let Some(image) = meta.image {
                report.push_str(&format!("\nimage: {image}"));
            }
            return Ok(report);
        }
    }
    let detail = match releases.first() {
        None => "repository has no releases".to_owned(),
        Some(newest) => format!(
            "checked {} releases ({} with readable release.json); newest: {}",
            releases.len(),
            readable,
            newest.tag_name
        ),
    };
    Err(format!(
        "no published Fountain release serves pin {revision} ({detail})"
    ))
}

pub fn default_pin_file() -> String {
    "fountain-source.json".to_owned()
}

pub fn default_api_base() -> String {
    "https://api.github.com".to_owned()
}
