//! Shared provider CLI freshness store (UP-20261026-v1, T00 frozen contract).
//!
//! One retained store serves every consumer. Bundle payloads are immutable by
//! digest; a small atomically replaced selection record is the shared source
//! of truth. Reads distinguish current, stale, unknown and update-failed; a
//! failed check never refreshes timestamps and never presents a fallback as
//! latest.
//!
//! Store layout under the configured root:
//!
//! ```text
//! <root>/.lock                                cross-operation flock(2) guard
//! <root>/bundles/<provider>/<digest-hex>/...   immutable admitted payload
//! <root>/bundles/<provider>/<digest-hex>/release.json   admission record
//! <root>/bundles/<provider>/<digest-hex>/inventory.json per-file digests
//! <root>/current-<provider>.json               atomically replaced selection
//! <root>/previous-<provider>.json              last replaced selection (GC keep)
//! <root>/refs/<token>.json                     active exact-bundle references
//! ```
//!
//! Locking contract (lane B builds on this): [`store::lock_store`] returns the
//! exclusive-access guard. It is held across resolve+acquire
//! ([`resolve::resolve_pinned`]), refresh selection publication, and the whole
//! [`store::cleanup`] pass. Bundle resolution verifies the admission record
//! plus per-file digests via [`store::verify_bundle`]; a failed refresh never
//! downgrades a stored `Stale` selection and a known-newer discovery is
//! persisted before the blocking error returns.
//!
//! Lane ownership: lane A implements every body in this crate. Lanes B and C
//! consume only these exported names; coordinator owns `Cargo.toml` and the
//! workspace membership.

pub mod config;
pub mod discover;
pub mod refresh;
pub mod resolve;
pub mod store;

pub use store::{Freshness, Provider, ReleaseIdentity, Selection, Status};

/// Failure class for freshness operations.
#[derive(Debug)]
pub enum Error {
    /// Invalid configuration, request or on-disk state.
    Denied(String),
    /// Transient failure: network, upstream, download, archive.
    Failed(String),
    /// Teardown/cleanup could not confirm; caller must retry or reconnect.
    Uncertain(String),
    /// Known newer release (or failed update) blocks new provider work.
    /// Existing work keeps its exact retained bytes; only NEW work is blocked.
    Blocked(String),
}

impl Error {
    pub fn denied(message: impl Into<String>) -> Error {
        Error::Denied(message.into())
    }
    pub fn failed(message: impl Into<String>) -> Error {
        Error::Failed(message.into())
    }
    pub fn uncertain(message: impl Into<String>) -> Error {
        Error::Uncertain(message.into())
    }
    pub fn blocked(message: impl Into<String>) -> Error {
        Error::Blocked(message.into())
    }

    /// True when new provider work must not start on this result.
    pub fn blocks_new_work(&self) -> bool {
        matches!(self, Error::Blocked(_))
    }
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Error::Denied(m) | Error::Failed(m) | Error::Uncertain(m) | Error::Blocked(m) => {
                f.write_str(m)
            }
        }
    }
}

impl std::error::Error for Error {}

impl From<std::io::Error> for Error {
    fn from(err: std::io::Error) -> Error {
        Error::Failed(err.to_string())
    }
}

impl From<serde_json::Error> for Error {
    fn from(err: serde_json::Error) -> Error {
        Error::Failed(err.to_string())
    }
}

/// Lane A shared plumbing (crate-internal). No model calls, no credentials.

/// User-Agent sent on all official upstream metadata/artifact requests.
pub(crate) const USER_AGENT: &str = "soda-provider-update (UP-20261026-v1)";

/// Seconds since the Unix epoch; saturates to zero before the epoch.
pub(crate) fn now_secs() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

/// Validate a `sha256:<hex>` or bare hex digest into lowercase hex (64 chars).
pub(crate) fn normalize_digest(raw: &str) -> Result<String, Error> {
    let hex = raw.strip_prefix("sha256:").unwrap_or(raw).trim();
    let lower = hex.to_lowercase();
    if lower.len() == 64 && lower.chars().all(|c| c.is_ascii_hexdigit()) {
        Ok(lower)
    } else {
        Err(Error::denied(format!(
            "invalid sha256 digest (want 64 hex chars): {raw}"
        )))
    }
}

/// Run a no-model, no-auth executable probe with a hard timeout.
/// Returns stdout on success; kills the child and fails on timeout.
pub(crate) fn run_probe(
    prog: &std::path::Path,
    args: &[&str],
    timeout: std::time::Duration,
) -> Result<String, Error> {
    let mut child = std::process::Command::new(prog)
        .args(args)
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped())
        .spawn()
        .map_err(|e| Error::failed(format!("probe spawn failed: {}", e)))?;
    let deadline = std::time::Instant::now() + timeout;
    loop {
        match child.try_wait() {
            Ok(Some(status)) => {
                let mut out = String::new();
                if let Some(mut stdout) = child.stdout.take() {
                    use std::io::Read as _;
                    let _ = stdout.read_to_string(&mut out);
                }
                if status.success() {
                    return Ok(out);
                } else {
                    return Err(Error::failed(format!(
                        "probe exited with status {status}: {} {}",
                        prog.display(),
                        args.join(" ")
                    )));
                }
            }
            Ok(None) => {
                if std::time::Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(Error::failed(format!(
                        "probe timed out after {}s: {} {}",
                        timeout.as_secs(),
                        prog.display(),
                        args.join(" ")
                    )));
                }
                std::thread::sleep(std::time::Duration::from_millis(50));
            }
            Err(e) => return Err(Error::failed(format!("probe wait failed: {e}"))),
        }
    }
}

/// GET a URL as text. Non-2xx and transport failures are `Failed`, never a
/// stale pin presented as current.
pub(crate) fn http_get_text(url: &str, accept: &str) -> Result<String, Error> {
    let response = ureq::get(url)
        .set("User-Agent", USER_AGENT)
        .set("Accept", accept)
        .timeout(std::time::Duration::from_secs(120))
        .call()
        .map_err(|e| Error::failed(format!("upstream GET failed: {url}: {e}")))?;
    if !(200..300).contains(&response.status()) {
        return Err(Error::failed(format!(
            "upstream GET returned status {}: {url}",
            response.status()
        )));
    }
    response
        .into_string()
        .map_err(|e| Error::failed(format!("upstream body read failed: {url}: {e}")))
}

/// Download a URL to `dest` (parent created), bounded to
/// `expected_size + 1` bytes so an oversized body fails instead of filling
/// disk. Returns the exact byte count written.
pub(crate) fn http_download_bounded(
    url: &str,
    dest: &std::path::Path,
    expected_size: u64,
) -> Result<u64, Error> {
    if let Some(parent) = dest.parent() {
        std::fs::create_dir_all(parent)?;
    }
    let response = ureq::get(url)
        .set("User-Agent", USER_AGENT)
        .set("Accept", "application/octet-stream")
        .timeout(std::time::Duration::from_secs(1800))
        .call()
        .map_err(|e| Error::failed(format!("artifact download failed: {url}: {e}")))?;
    if !(200..300).contains(&response.status()) {
        return Err(Error::failed(format!(
            "artifact download returned status {}: {url}",
            response.status()
        )));
    }
    let file = std::fs::File::create(dest)?;
    use std::io::Read as _;
    let mut reader = response.into_reader().take(expected_size + 1);
    let mut writer = std::io::BufWriter::new(file);
    let written = std::io::copy(&mut reader, &mut writer)
        .map_err(|e| Error::failed(format!("artifact write failed: {e}")))?;
    drop(writer);
    if written > expected_size {
        let _ = std::fs::remove_file(dest);
        return Err(Error::failed(format!(
            "artifact larger than published size {expected_size}: {url}"
        )));
    }
    Ok(written)
}

/// Lowercase hex sha256 of a file plus its length.
pub(crate) fn sha256_hex_file(path: &std::path::Path) -> Result<(String, u64), Error> {
    use sha2::Digest as _;
    let file = std::fs::File::open(path)?;
    let mut reader = std::io::BufReader::new(file);
    let mut hasher = sha2::Sha256::new();
    let len = std::io::copy(&mut reader, &mut hasher)?;
    let hex: String = hasher
        .finalize()
        .iter()
        .map(|b| format!("{b:02x}"))
        .collect();
    Ok((hex, len))
}

/// fsync a directory so renames inside it survive a crash.
pub(crate) fn fsync_dir(dir: &std::path::Path) -> Result<(), Error> {
    let fd = std::fs::File::open(dir)?;
    #[cfg(unix)]
    {
        use std::os::unix::io::AsRawFd as _;
        let rc = unsafe { libc::fsync(fd.as_raw_fd()) };
        if rc != 0 {
            return Err(Error::failed(format!(
                "fsync failed: {}",
                std::io::Error::last_os_error()
            )));
        }
    }
    Ok(())
}
