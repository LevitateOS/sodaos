//! Shared-store-first resolution for the Muse native binary.
//!
//! Ordinary launch prefers the shared provider store's selected bundle and
//! refuses known-stale or unusable selections fail-closed, so a new shell
//! Muse call can never silently keep a baked executable once the updater
//! has admitted something newer. The pinned installed binary remains only
//! for hosts where the store was never admitted (before the updater's
//! first refresh); it is never preferred over an admitted selection.
//!
//! When the check is due, one best-effort `soda-provider-update refresh`
//! runs where that helper exists; refresh failure never blocks launch here
//! (the stored selection still gates below). Staged-receipt enforcement
//! covers retained Projects: maintain plants the admitted identity next to
//! the staged binary, and launch refuses a mismatch or an explicit stale
//! marker without hashing 400MB per invocation (digest verification happens
//! at admission and at staging, not at every exec).

use super::MUSE_NATIVE;
use std::time::{Duration, Instant};

const DEFAULT_ROOT: &str = "/var/lib/soda/providers";
const DEFAULT_TTL_SECS: u64 = 300;
const SELECTION_FILE: &str = "current-muse.json";
const DEFAULT_RECEIPT: &str = "/usr/local/libexec/soda/muse-selection.json";

#[derive(Default, serde::Deserialize)]
struct Release {
    #[serde(default)]
    version: String,
    #[serde(default)]
    digest: String,
    #[serde(default)]
    size: u64,
}

#[derive(Default, serde::Deserialize)]
struct Freshness {
    #[serde(default)]
    state: String,
    #[serde(default)]
    latest: Option<String>,
}

#[derive(Default, serde::Deserialize)]
struct Selection {
    #[serde(default)]
    provider: String,
    #[serde(default)]
    release: Release,
    #[serde(default)]
    checked_at_secs: u64,
    #[serde(default)]
    freshness: Freshness,
}

fn store_root() -> String {
    match std::env::var("SODA_PROVIDERS_ROOT") {
        Ok(root) if !root.is_empty() => root,
        _ => DEFAULT_ROOT.to_string(),
    }
}

fn receipt_path() -> String {
    match std::env::var("SODA_MUSE_RECEIPT_PATH") {
        Ok(path) if !path.is_empty() => path,
        _ => DEFAULT_RECEIPT.to_string(),
    }
}

fn ttl_secs() -> u64 {
    std::env::var("SODA_PROVIDERS_CHECK_TTL_SECS")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(DEFAULT_TTL_SECS)
}

fn now_secs() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

fn is_hex64(s: &str) -> bool {
    s.len() == 64 && s.bytes().all(|b| b.is_ascii_hexdigit())
}

/// Resolve the native Muse binary for this invocation.
///
/// Store-first: an admitted selection wins over the pinned path. Fails
/// closed on known-stale/unknown selections and on damaged bundles. Falls
/// back to the pinned binary only when the store was never admitted.
pub(crate) fn resolve_muse_native() -> Result<String, String> {
    let root = store_root();
    let path = format!("{root}/{SELECTION_FILE}");
    let mut selection = match std::fs::read(&path) {
        Ok(bytes) => parse_selection(&bytes)?,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
            return Ok(MUSE_NATIVE.to_string())
        }
        Err(_) => return Err(String::from("provider store selection unreadable")),
    };
    if now_secs().saturating_sub(selection.checked_at_secs) > ttl_secs() {
        best_effort_refresh();
        if let Ok(bytes) = std::fs::read(&path) {
            if let Ok(fresh) = parse_selection(&bytes) {
                selection = fresh;
            }
        }
    }
    let digest = gate_selection(&selection)?;
    verify_bundle(&root, &digest, selection.release.size)
}

fn parse_selection(bytes: &[u8]) -> Result<Selection, String> {
    let selection: Selection = serde_json::from_slice(bytes)
        .map_err(|_| String::from("provider store selection unreadable"))?;
    if selection.provider != "muse"
        || selection.release.version.is_empty()
        || !is_hex64(&selection.release.digest)
        || selection.release.size == 0
    {
        return Err(String::from("provider store selection unreadable"));
    }
    Ok(selection)
}

/// Refuse new work on known-stale or unusable selections. `Current` and
/// `UpdateFailed` (no newer release known) may run their retained bytes,
/// matching the shared store's launch semantics.
fn gate_selection(selection: &Selection) -> Result<String, String> {
    match selection.freshness.state.as_str() {
        "current" | "update_failed" => Ok(selection.release.digest.clone()),
        "stale" => {
            let latest = selection.freshness.latest.as_deref().unwrap_or("unknown");
            Err(format!(
                "muse provider is known-stale (latest {latest}); refresh before launching"
            ))
        }
        _ => Err(String::from("no usable muse selection for new work")),
    }
}

/// Best-effort due-check refresh where the updater helper exists. Any
/// failure (missing helper, timeout, updater error) is ignored: the stored
/// selection still gates above, so launch never bypasses freshness, it just
/// may run retained bytes whose check is due.
fn best_effort_refresh() {
    let mut child = match std::process::Command::new("soda-provider-update")
        .args(["refresh", "--provider", "muse"])
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .spawn()
    {
        Ok(child) => child,
        Err(_) => return,
    };
    let deadline = Instant::now() + Duration::from_secs(60);
    loop {
        match child.try_wait() {
            Ok(Some(_)) => return,
            Ok(None) => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    return;
                }
                std::thread::sleep(Duration::from_millis(50));
            }
            Err(_) => return,
        }
    }
}

/// The bundle path is derived from the validated selection digest, so only
/// exact admitted bytes can match. Rejects symlinks, world-writable files
/// and size drift without rehashing at every launch.
fn verify_bundle(root: &str, digest: &str, size: u64) -> Result<String, String> {
    let exe = format!("{root}/bundles/muse/{digest}/bin/muse");
    let meta = std::fs::symlink_metadata(&exe)
        .map_err(|_| String::from("provider store executable is unavailable"))?;
    use std::os::unix::fs::MetadataExt;
    if !meta.is_file() || meta.mode() & 0o022 != 0 || meta.len() != size {
        return Err(String::from("provider store executable is unavailable"));
    }
    Ok(exe)
}

/// Enforce the staged receipt for retained Projects.
///
/// Applies only when launch resolves to the pinned (staged) path: the
/// receipt maintain planted at staging must agree with the staged binary,
/// and an explicit stale marker refuses. A missing receipt grandfathers
/// pre-change stagings, which the next launch-time staging replaces.
pub(crate) fn enforce_receipt(binary: &str) -> Result<(), String> {
    if binary != MUSE_NATIVE {
        return Ok(());
    }
    let bytes = match std::fs::read(receipt_path()) {
        Ok(bytes) => bytes,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(()),
        Err(_) => return Err(String::from("staged muse receipt unreadable")),
    };
    let receipt: Selection = serde_json::from_slice(&bytes)
        .map_err(|_| String::from("staged muse receipt unreadable"))?;
    if receipt.provider != "muse"
        || receipt.release.version.is_empty()
        || !is_hex64(&receipt.release.digest)
        || receipt.release.size == 0
    {
        return Err(String::from("staged muse receipt unreadable"));
    }
    match receipt.freshness.state.as_str() {
        "current" | "update_failed" => {}
        "stale" => {
            let latest = receipt.freshness.latest.as_deref().unwrap_or("unknown");
            return Err(format!(
                "staged muse is known-stale (latest {latest}); refresh before launching"
            ));
        }
        _ => return Err(String::from("no usable staged muse receipt for new work")),
    }
    let meta = std::fs::symlink_metadata(binary)
        .map_err(|_| String::from("staged muse binary is unavailable"))?;
    use std::os::unix::fs::MetadataExt;
    if !meta.is_file() || meta.mode() & 0o022 != 0 || meta.len() != receipt.release.size {
        return Err(String::from(
            "staged muse binary does not match the admitted receipt",
        ));
    }
    Ok(())
}
