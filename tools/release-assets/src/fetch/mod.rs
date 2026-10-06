//! Pinned asset acquisition (ex `soda-asset-fetchers`).
//!
//! Ports `scripts/fetch-tea.py`, `scripts/fetch-terminal.py` and
//! `tools/soda-fetch-muse` (on `internal/release/build` `FetchMuse`) to
//! three binaries. Every success output (file layouts, modes,
//! stdout lines) and every validation error message matches the owner byte
//! for byte; only unpinned failure text (HTTP transport errors, usage
//! errors, malformed-input details) differs.

pub mod muse;
pub mod tea;
pub mod terminal;

#[cfg(test)]
pub(crate) mod test_server;

use std::io::Read;
use std::path::Path;
use std::time::Duration;

/// A completed HTTP GET: the status line plus an owned body reader.
pub struct GetResponse {
    pub status: u16,
    pub reader: Box<dyn Read>,
}

/// Plain GET with an explicit User-Agent and a whole-request timeout.
///
/// Transport failures come back as `Err`; HTTP error statuses come back as
/// `Ok` with an empty reader so each fetcher can apply its owner's status
/// rule (the Python fetchers raise on any non-2xx; the Go Muse fetcher
/// only accepts 200).
pub fn http_get(url: &str, user_agent: &str, timeout: Duration) -> Result<GetResponse, String> {
    let response = ureq::get(url)
        .timeout(timeout)
        .set("User-Agent", user_agent)
        .call();
    match response {
        Ok(ok) => {
            let status = ok.status();
            Ok(GetResponse {
                status,
                reader: Box::new(ok.into_reader()),
            })
        }
        Err(ureq::Error::Status(status, _)) => Ok(GetResponse {
            status,
            reader: Box::new(std::io::empty()),
        }),
        Err(ureq::Error::Transport(transport)) => Err(transport.to_string()),
    }
}

/// Read at most `limit + 1` bytes so callers can tell "exactly at the cap"
/// from "over the cap", like the owners' `read(limit + 1)` idiom.
pub fn read_capped(reader: &mut dyn Read, limit: u64) -> Result<Vec<u8>, String> {
    let mut body = Vec::new();
    reader
        .take(limit.saturating_add(1))
        .read_to_end(&mut body)
        .map_err(|e| e.to_string())?;
    Ok(body)
}

/// Lowercase-hex SHA-256, like `hashlib.sha256().hexdigest()`.
pub fn sha256_hex(data: &[u8]) -> String {
    use sha2::Digest;
    let mut hasher = sha2::Sha256::new();
    hasher.update(data);
    hex_lower(&hasher.finalize())
}

/// Streaming SHA-256 for already-staged files, like Go's `HashFile`.
pub fn sha256_hex_stream(reader: &mut dyn Read) -> Result<String, String> {
    use sha2::Digest;
    let mut hasher = sha2::Sha256::new();
    std::io::copy(reader, &mut hasher).map_err(|e| e.to_string())?;
    Ok(hex_lower(&hasher.finalize()))
}

/// Standard-base64 SHA-512 for `sha512-<...>` integrity pins.
pub fn sha512_base64(data: &[u8]) -> String {
    use base64::Engine as _;
    use sha2::Digest;
    let mut hasher = sha2::Sha512::new();
    hasher.update(data);
    base64::engine::general_purpose::STANDARD.encode(hasher.finalize())
}

fn hex_lower(digest: &[u8]) -> String {
    let mut out = String::with_capacity(digest.len() * 2);
    for byte in digest {
        out.push_str(&format!("{byte:02x}"));
    }
    out
}

pub(crate) fn chmod(path: &Path, mode: u32) -> Result<(), String> {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(path, std::fs::Permissions::from_mode(mode))
            .map_err(|e| e.to_string())
    }
    #[cfg(not(unix))]
    {
        let _ = (path, mode);
        Err("asset staging requires unix file modes".to_string())
    }
}

/// Tokenize one argv element: `--name=value` gives `("name", Some(value))`,
/// `--name` gives `("name", None)`; one or two leading dashes are stripped.
/// Returns `None` for non-flag tokens (including a bare `--`).
pub fn flag_token(arg: &str) -> Option<(String, Option<String>)> {
    if arg == "--" || arg == "-" || !arg.starts_with('-') {
        return None;
    }
    let body = arg.trim_start_matches('-');
    if body.is_empty() {
        return None;
    }
    match body.split_once('=') {
        Some((name, value)) => Some((name.to_string(), Some(value.to_string()))),
        None => Some((body.to_string(), None)),
    }
}
