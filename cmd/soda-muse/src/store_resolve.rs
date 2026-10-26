//! Store-authoritative resolution for the Muse native binary.
//!
//! Every invocation re-reads the CURRENT shared-store selection: the staged
//! receipt can never establish currentness on its own. When the store view is
//! visible, the live selection is the sole authority (TTL due-check with a
//! best-effort refresh, known-stale/unknown refusal, full digest
//! verification). The staged binary plus its receipt only run when the store
//! was never admitted (pre-updater hosts); a missing receipt while any store
//! knowledge exists refuses instead of silently launching old bytes.
//!
//! Schema note: the selection and receipt records below are byte-compatible
//! with the shared `soda_provider_tools::store::Selection` JSON (same field
//! names, same `freshness.state` tagged values) and enforce the same gate
//! semantics as `resolve::resolve_new` (Current/UpdateFailed run retained
//! bytes, Stale/Unknown block new work). They are a local mirror only
//! because this crate may not add dependencies in this lane; the requested
//! follow-up is `soda-provider-tools = { path = "../../lib/provider-tools" }`
//! in `cmd/soda-muse/Cargo.toml` (coordinator-owned) with this module then
//! calling `resolve::resolve_new` / `store::read_selection` /
//! `store::verify_bundle` directly instead of mirroring them.

use super::MUSE_NATIVE;
use std::process::Stdio;
use std::time::{Duration, Instant};

const DEFAULT_ROOT: &str = "/var/lib/soda/providers";
const DEFAULT_TTL_SECS: u64 = 300;
const SELECTION_FILE: &str = "current-muse.json";
const DEFAULT_RECEIPT: &str = "/usr/local/libexec/soda/muse-selection.json";

/// Mirror of the shared store's release identity. Validation matches
/// `provider-tools::store::validate_release`.
#[derive(serde::Deserialize)]
struct StoreRelease {
    #[serde(default)]
    provider: String,
    #[serde(default)]
    version: String,
    #[serde(default)]
    digest: String,
    #[serde(default)]
    size: u64,
    #[serde(default)]
    artifact_url: String,
}

/// Mirror of the shared store's freshness values (tagged `state`,
/// snake_case), with the same launch semantics as `resolve_new`.
#[derive(serde::Deserialize)]
#[serde(tag = "state", rename_all = "snake_case")]
enum StoreFreshness {
    Current,
    Stale { #[serde(default)] latest: String },
    Unknown { #[serde(default)] reason: String },
    UpdateFailed { #[serde(default)] reason: String },
}

/// Mirror of the shared store's atomically replaced selection record. Both
/// the live host selection and the staged receipt use this one shape.
#[derive(serde::Deserialize)]
struct StoreSelection {
    #[serde(default)]
    provider: String,
    #[serde(default)]
    release: Option<StoreRelease>,
    #[serde(default)]
    checked_at_secs: u64,
    #[serde(default)]
    admitted_at_secs: u64,
    #[serde(default)]
    freshness: Option<StoreFreshness>,
}

impl Default for StoreRelease {
    fn default() -> Self {
        Self {
            provider: String::new(),
            version: String::new(),
            digest: String::new(),
            size: 0,
            artifact_url: String::new(),
        }
    }
}

fn store_root() -> String {
    match std::env::var("SODA_PROVIDERS_ROOT") {
        Ok(root) if !root.is_empty() => root,
        _ => DEFAULT_ROOT.to_string(),
    }
}

fn staged_path() -> String {
    match std::env::var("SODA_MUSE_STAGED_PATH") {
        Ok(path) if !path.is_empty() => path,
        _ => MUSE_NATIVE.to_string(),
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

/// Read and validate one common-schema selection record. `Ok(None)` means
/// the store never admitted this provider (no file); any other failure is
/// `Err` so damaged knowledge fails closed. Pure read: never refreshes.
fn read_store_selection(path: &str) -> Result<Option<StoreSelection>, String> {
    let bytes = match std::fs::read(path) {
        Ok(bytes) => bytes,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(None),
        Err(_) => {
            return Err(String::from("provider store selection unreadable"));
        }
    };
    let selection: StoreSelection = serde_json::from_slice(&bytes)
        .map_err(|_| String::from("provider store selection unreadable"))?;
    validate_selection(&selection)?;
    Ok(Some(selection))
}

/// Mirror of the shared store's selection validation: exact provider,
/// exact release identity, never a fallback presented as usable.
fn validate_selection(selection: &StoreSelection) -> Result<(), String> {
    if selection.provider != "muse" {
        return Err(String::from("provider store selection unreadable"));
    }
    let Some(release) = selection.release.as_ref() else {
        return Err(String::from("provider store selection unreadable"));
    };
    if release.provider != "muse"
        || release.version.is_empty()
        || !is_hex64(&release.digest)
        || release.size == 0
        || !(release.artifact_url.starts_with("https://")
            || release.artifact_url.starts_with("http://localhost")
            || release.artifact_url.starts_with("http://127."))
    {
        return Err(String::from("provider store selection unreadable"));
    }
    if selection.freshness.is_none() {
        return Err(String::from("provider store selection unreadable"));
    }
    let _ = selection.admitted_at_secs;
    Ok(())
}

/// Gate a validated selection the way `resolve_new` does: `Current` and
/// `UpdateFailed` (no newer release known) may run their retained bytes;
/// `Stale` and `Unknown` block new work.
fn gate_selection(selection: &StoreSelection) -> Result<(), String> {
    match selection.freshness.as_ref() {
        Some(StoreFreshness::Current) => Ok(()),
        Some(StoreFreshness::UpdateFailed { reason }) => {
            let _ = reason;
            Ok(())
        }
        Some(StoreFreshness::Stale { latest }) => Err(format!(
            "muse provider is known-stale (latest {latest}); refresh before launching"
        )),
        Some(StoreFreshness::Unknown { reason }) => Err(format!(
            "no usable muse selection for new work ({reason})"
        )),
        None => Err(String::from("no usable muse selection for new work")),
    }
}

/// Resolve the Muse native binary for this invocation.
///
/// Store-authoritative: a usable live selection always wins over the staged
/// binary, so a host that admitted v2 serves v2 to the next invocation with
/// no maintenance step. Falls back to the staged binary only when the store
/// was never admitted. Fails closed on stale/unknown selections, on damaged
/// store knowledge, and on bundle bytes that do not fully verify.
pub(crate) fn resolve_muse_native() -> Result<String, String> {
    let root = store_root();
    let path = format!("{root}/{SELECTION_FILE}");
    let mut selection = match read_store_selection(&path)? {
        Some(selection) => selection,
        None => return Ok(staged_path()),
    };
    if now_secs().saturating_sub(selection.checked_at_secs) > ttl_secs() {
        // Due check: one best-effort refresh where the helper exists. A
        // failed or missing refresh keeps the stored selection, which still
        // gates below; launch never bypasses freshness, it just may run
        // retained bytes whose check is due.
        best_effort_refresh();
        if let Ok(Some(fresh)) = read_store_selection(&path) {
            selection = fresh;
        }
    }
    gate_selection(&selection)?;
    let release = selection.release.as_ref().expect("validated above");
    verify_bundle(&root, &release.digest, release.size)
}

/// Best-effort due-check refresh where the updater helper exists. Any
/// failure (missing helper, timeout, updater error) is ignored: the stored
/// selection still gates above.
fn best_effort_refresh() {
    let mut child = match std::process::Command::new("soda-provider-update")
        .args(["refresh", "--provider", "muse"])
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
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

/// The bundle path is derived from the validated selection digest, and the
/// bytes must fully hash to that digest: digest/inventory coherence, never
/// a size-only check. Rejects symlinks and world-writable files.
fn verify_bundle(root: &str, digest: &str, size: u64) -> Result<String, String> {
    let exe = format!("{root}/bundles/muse/{}/bin/muse", digest.to_lowercase());
    verify_executable(&exe, digest, size)
        .map_err(|_| String::from("provider store executable is unavailable"))?;
    Ok(exe)
}

fn verify_executable(path: &str, digest: &str, size: u64) -> Result<(), String> {
    let meta = std::fs::symlink_metadata(path).map_err(|_| String::from("unavailable"))?;
    use std::os::unix::fs::MetadataExt;
    if meta.file_type().is_symlink()
        || !meta.is_file()
        || meta.mode() & 0o022 != 0
        || meta.len() != size
    {
        return Err(String::from("unavailable"));
    }
    let sum = sha256_of(path)?;
    if sum != digest.to_lowercase() {
        return Err(String::from("unavailable"));
    }
    Ok(())
}

/// Hash one file with the platform checksum tool and return lowercase hex.
/// Fails closed when the helper is missing or the output is unusable.
fn sha256_of(path: &str) -> Result<String, String> {
    let mut child = std::process::Command::new("/usr/bin/sha256sum")
        .arg("--")
        .arg(path)
        .stdin(Stdio::null())
        .stdout(std::process::Stdio::piped())
        .stderr(Stdio::null())
        .spawn()
        .map_err(|_| String::from("muse executable verification unavailable"))?;
    let output = child
        .stdout
        .take()
        .map(|mut out| {
            use std::io::Read as _;
            let mut text = String::new();
            let _ = out.read_to_string(&mut text);
            text
        })
        .unwrap_or_default();
    let deadline = Instant::now() + Duration::from_secs(120);
    loop {
        match child.try_wait() {
            Ok(Some(status)) => {
                if !status.success() {
                    return Err(String::from("muse executable verification unavailable"));
                }
                let sum = output.split_whitespace().next().unwrap_or_default().to_lowercase();
                if !is_hex64(&sum) {
                    return Err(String::from("muse executable verification unavailable"));
                }
                return Ok(sum);
            }
            Ok(None) => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(String::from("muse executable verification unavailable"));
                }
                std::thread::sleep(Duration::from_millis(50));
            }
            Err(_) => return Err(String::from("muse executable verification unavailable")),
        }
    }
}

/// Enforce the staged receipt for retained Projects.
///
/// The live host selection is authoritative whenever it is visible: a
/// receipt that said current at staging time cannot overrule today's
/// selection, and a missing receipt bootstraps through the store path
/// (resolve already returned verified store bytes) instead of launching old
/// bytes. The staged binary runs only when the store was never admitted
/// (pre-updater hosts, with or without a legacy receipt) or when the store
/// view is unavailable and a usable receipt corroborates the staged bytes.
pub(crate) fn enforce_receipt(binary: &str) -> Result<(), String> {
    let root = store_root();
    let staged = staged_path();
    if binary != staged {
        // Store-resolved (or foreign) binary: authority was established and
        // fully verified by this invocation's resolution, not by a copy.
        return Ok(());
    }
    let live = match read_store_selection(&format!("{root}/{SELECTION_FILE}")) {
        Ok(live) => live,
        Err(_) => {
            return Err(String::from("provider store selection unreadable"));
        }
    };
    let bytes = match std::fs::read(receipt_path()) {
        Ok(bytes) => bytes,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
            // No receipt: only a never-admitted store (no trusted path at
            // all) may still run the legacy staged binary. Any existing
            // store knowledge demands the store path instead.
            if live.is_none() {
                return Ok(());
            }
            return Err(String::from(
                "staged muse receipt missing; resolve through the provider store before launching",
            ));
        }
        Err(_) => return Err(String::from("staged muse receipt unreadable")),
    };
    let receipt: StoreSelection = serde_json::from_slice(&bytes)
        .map_err(|_| String::from("staged muse receipt unreadable"))?;
    validate_selection(&receipt).map_err(|_| String::from("staged muse receipt unreadable"))?;
    match receipt.freshness.as_ref() {
        Some(StoreFreshness::Current) => {}
        Some(StoreFreshness::UpdateFailed { reason }) => {
            let _ = reason;
        }
        Some(StoreFreshness::Stale { latest }) => {
            return Err(format!(
                "staged muse is known-stale (latest {latest}); refresh before launching"
            ));
        }
        _ => {
            return Err(String::from(
                "no usable staged muse receipt for new work",
            ));
        }
    }
    match live {
        Some(live) => {
            // Today's authority beats yesterday's copy: a stale or unknown
            // live selection refuses even when the receipt still reads
            // current, and a digest drift refuses the outdated staged bytes
            // instead of running them silently.
            gate_selection(&live)?;
            let live_release = live.release.as_ref().expect("validated above");
            let receipt_release = receipt.release.as_ref().expect("validated above");
            if live_release.digest.to_lowercase() != receipt_release.digest.to_lowercase()
                || live_release.version != receipt_release.version
            {
                return Err(format!(
                    "staged muse {} is older than the current selection {}; refresh before launching",
                    receipt_release.version, live_release.version
                ));
            }
        }
        None => {
            // No store ever admitted: the receipt is the staging record.
            // Digest agreement is still required below.
        }
    }
    let receipt_release = receipt.release.as_ref().expect("validated above");
    verify_executable(&staged, &receipt_release.digest, receipt_release.size).map_err(|_| {
        String::from("staged muse binary does not match the admitted receipt")
    })?;
    Ok(())
}
