//! Store-authoritative resolution for the Muse native binary.
//!
//! Every invocation re-reads the CURRENT shared-store selection: the staged
//! receipt can never establish currentness on its own. When the store view is
//! visible, the live selection is the sole authority (TTL due-check with a
//! real refresh, known-stale/unknown refusal, full digest verification).
//! The staged binary plus its receipt only run when the store was never
//! admitted (pre-updater hosts); a missing receipt while any store knowledge
//! exists refuses instead of silently launching old bytes.
//!
//! Admission and freshness truth lives in the shared `soda_provider_tools`
//! crate: selection records parse as its `Selection` type and new work
//! resolves through `resolve::resolve_new` (which refreshes when the check
//! is due and gates `Current`/`UpdateFailed` vs `Stale`/`Unknown`). This
//! module holds no mirror of those records or that gate.

use super::MUSE_NATIVE;
use std::process::Stdio;
use std::time::{Duration, Instant};

use soda_provider_tools::{store, Provider};

const DEFAULT_RECEIPT: &str = "/usr/local/libexec/soda/muse-selection.json";

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

fn is_hex64(s: &str) -> bool {
    s.len() == 64 && s.bytes().all(|b| b.is_ascii_hexdigit())
}

/// Validate one shared-schema receipt record the same way the store does:
/// exact provider, exact release identity, never a fallback presented as
/// usable. Pure check: never refreshes.
fn validate_receipt(selection: &store::Selection) -> Result<(), String> {
    if selection.provider != Provider::Muse {
        return Err(String::from("staged muse receipt unreadable"));
    }
    let release = &selection.release;
    if release.provider != Provider::Muse
        || release.version.is_empty()
        || !is_hex64(&release.digest)
        || release.size == 0
        || !(release.artifact_url.starts_with("https://")
            || release.artifact_url.starts_with("http://localhost")
            || release.artifact_url.starts_with("http://127."))
    {
        return Err(String::from("staged muse receipt unreadable"));
    }
    Ok(())
}

/// Gate one shared-schema selection the way `resolve_new` does: `Current`
/// and `UpdateFailed` (no newer release known) may run retained bytes;
/// `Stale` and `Unknown` block new work.
fn gate_live(selection: &store::Selection) -> Result<(), String> {
    match &selection.freshness {
        store::Freshness::Current => Ok(()),
        store::Freshness::UpdateFailed { reason } => {
            let _ = reason;
            Ok(())
        }
        store::Freshness::Stale { latest } => Err(format!(
            "muse provider is known-stale (latest {latest}); refresh before launching"
        )),
        store::Freshness::Unknown { reason } => {
            Err(format!("no usable muse selection for new work ({reason})"))
        }
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
    let root = soda_provider_tools::config::store_root();
    match store::read_selection(&root, Provider::Muse) {
        Ok(None) => Ok(staged_path()),
        Err(_) => Err(String::from("provider store selection unreadable")),
        Ok(Some(_)) => {
            // Store knowledge exists: the shared resolver is authoritative
            // (due-check refresh, staleness gate, full bundle verification).
            // Its refusal message stays visible to the operator.
            let executable = soda_provider_tools::resolve::resolve_new(Provider::Muse)
                .map_err(|e| e.to_string())?;
            Ok(executable.to_string_lossy().into_owned())
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
    let staged = staged_path();
    if binary != staged {
        // Store-resolved (or foreign) binary: authority was established and
        // fully verified by this invocation's resolution, not by a copy.
        return Ok(());
    }
    let root = soda_provider_tools::config::store_root();
    let live = match store::read_selection(&root, Provider::Muse) {
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
    let receipt: store::Selection = serde_json::from_slice(&bytes)
        .map_err(|_| String::from("staged muse receipt unreadable"))?;
    validate_receipt(&receipt)?;
    match receipt.freshness {
        store::Freshness::Current => {}
        store::Freshness::UpdateFailed { .. } => {}
        store::Freshness::Stale { latest } => {
            return Err(format!(
                "staged muse is known-stale (latest {latest}); refresh before launching"
            ));
        }
        _ => {
            return Err(String::from("no usable staged muse receipt for new work"));
        }
    }
    match live {
        Some(live) => {
            // Today's authority beats yesterday's copy: a stale or unknown
            // live selection refuses even when the receipt still reads
            // current, and a digest drift refuses the outdated staged bytes
            // instead of running them silently.
            gate_live(&live)?;
            if live.release.digest.to_lowercase() != receipt.release.digest.to_lowercase()
                || live.release.version != receipt.release.version
            {
                return Err(format!(
                    "staged muse {} is older than the current selection {}; refresh before launching",
                    receipt.release.version, live.release.version
                ));
            }
        }
        None => {
            // No store ever admitted: the receipt is the staging record.
            // Digest agreement is still required below.
        }
    }
    verify_executable(&staged, &receipt.release.digest, receipt.release.size)
        .map_err(|_| String::from("staged muse binary does not match the admitted receipt"))?;
    Ok(())
}

/// The staged bytes must fully hash to the admitted digest:
/// digest/inventory coherence, never a size-only check. Rejects symlinks
/// and world-writable files.
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
