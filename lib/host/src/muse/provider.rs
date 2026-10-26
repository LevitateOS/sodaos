//! Provider-store identity for Muse executions (UP-20261026-v1, lane B).
//!
//! Pinned `binary_version` / `binary_sha256` configuration keeps its
//! historical meaning: the guest bundle is verified against the configured
//! pin. Empty pins switch to the shared provider store instead: every new
//! execution resolves the fresh selection, so a refreshed bundle is picked
//! up without a daemon restart and no hash is cached across executions.
//! In-flight executions keep the digest they verified against
//! (`MuseExecution.provider_digest`); re-resolution of that exact bundle
//! for old-run recovery goes through [`retained_muse_executable`], which
//! fails closed when the bundle is no longer admitted.
//!
//! Credential fixtures and model/profile/budget fields are untouched: this
//! module performs no credential reads and no model calls.

use crate::terminal;
use std::path::PathBuf;

/// Take the single coherent selection snapshot for NEW work: read the
/// current record, refresh when the check is due, then gate it. This mirrors
/// the launch semantics of `soda_provider_tools::resolve::resolve_new` but
/// returns the admitted `Selection` itself, so digest and version stay paired
/// from one value. The moving current record is never re-read afterwards.
fn fresh_selection() -> Result<soda_provider_tools::Selection, soda_provider_tools::Error> {
    use soda_provider_tools::{config, refresh, store};
    let root = config::store_root();
    let ttl = config::check_ttl();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0);
    let stored = store::read_selection(&root, soda_provider_tools::Provider::Muse)?;
    let due = match &stored {
        None => true,
        Some(selection) => now.saturating_sub(selection.checked_at_secs) > ttl.as_secs(),
    };
    let selection = if due {
        refresh::refresh(soda_provider_tools::Provider::Muse)?
    } else {
        stored.expect("selection exists when refresh is not due")
    };
    match &selection.freshness {
        store::Freshness::Current | store::Freshness::UpdateFailed { .. } => Ok(selection),
        store::Freshness::Stale { latest } => Err(soda_provider_tools::Error::blocked(format!(
            "known newer muse {latest} blocks new muse work; refresh before launching",
        ))),
        store::Freshness::Unknown { reason } => Err(soda_provider_tools::Error::blocked(format!(
            "no usable muse selection for new work ({reason})",
        ))),
    }
}

/// Expected `(version, sha256)` for one NEW Muse execution, captured as ONE
/// coherent admitted identity: both halves come from the same selection
/// snapshot, and the selected bundle is verified before anything is
/// returned. No reference is pinned here: the guest bytes are verified
/// inside the container per execution, and old-run recovery re-resolves the
/// exact retained digest (fail closed) through [`retained_muse_executable`].
pub fn expected_muse_identity(
    configured_version: &str,
    configured_sha256: &str,
) -> Result<(String, String), String> {
    if !configured_sha256.is_empty() {
        if configured_version.is_empty() {
            return Err(terminal::err_denied());
        }
        return Ok((
            configured_version.to_string(),
            configured_sha256.to_string(),
        ));
    }
    if !configured_version.is_empty() {
        // A version without a digest cannot identify exact bytes; this is a
        // misconfiguration, never a reason to fall back to a stale pin.
        return Err(terminal::err_denied());
    }
    // ONE snapshot: digest and version are paired here, never re-read.
    let selection = fresh_selection().map_err(map_store_error)?;
    let digest = selection.release.digest.clone();
    let version = selection.release.version.clone();
    // Fail closed when the selected bundle no longer verifies, matching the
    // gate the shared resolver applies before returning an executable.
    let root = soda_provider_tools::config::store_root();
    soda_provider_tools::store::verify_bundle(&root, soda_provider_tools::Provider::Muse, &digest)
        .map_err(|_| terminal::err_denied())?;
    debug_assert_eq!(
        bundle_digest(&root, &selection.executable(&root)).as_deref(),
        Some(digest.as_str())
    );
    Ok((version, digest))
}

/// Exact retained executable for EXISTING work by bundle digest (T00
/// `resolve_existing`). Fails closed when that bundle is no longer
/// admitted; callers must not fall back to the current selection here.
pub fn retained_muse_executable(digest: &str) -> Result<PathBuf, String> {
    if digest.len() != 64 || !digest.bytes().all(|b| b.is_ascii_hexdigit()) {
        return Err(terminal::err_denied());
    }
    soda_provider_tools::resolve::resolve_existing(soda_provider_tools::Provider::Muse, digest)
        .map_err(|_| terminal::err_denied())
}

/// Bundle digest for an admitted executable
/// `<root>/bundles/muse/<digest>/...`. Returns `None` when the path is not
/// inside the configured store layout, mirroring the broker's binding so a
/// non-store path is never mistaken for an admitted bundle.
fn bundle_digest(root: &std::path::Path, exe: &std::path::Path) -> Option<String> {
    let rel = exe.strip_prefix(root).ok()?;
    let mut parts = rel.components();
    let bundles = parts.next()?.as_os_str().to_str()?;
    let name = parts.next()?.as_os_str().to_str()?;
    let digest = parts.next()?.as_os_str().to_str()?;
    if bundles != "bundles" || name != soda_provider_tools::Provider::Muse.as_str() {
        return None;
    }
    if digest.len() != 64 || !digest.bytes().all(|b| b.is_ascii_hexdigit()) {
        return None;
    }
    if parts.next().is_none() {
        return None;
    }
    Some(digest.to_string())
}

fn map_store_error(err: soda_provider_tools::Error) -> String {
    use soda_provider_tools::Error as Store;
    match err {
        // Known-newer-blocked stays visible as stale (the serve loop maps
        // failures to denied, but the stale marker tells the operator a
        // refresh is due); existing runs keep their bound exact bundle.
        Store::Blocked(_) => terminal::err_stale(),
        _ => terminal::err_denied(),
    }
}
