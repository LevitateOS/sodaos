//! Provider-store identity for Muse executions (UP-20261026-v1, lane B).
//!
//! Every new execution resolves the fresh shared-store selection, so a
//! refreshed bundle is picked up without a daemon restart and no hash is
//! cached across executions. Legacy configured `binary_version` /
//! `binary_sha256` pins are never consulted for new work (mandatory
//! freshness supersedes them); a fully empty provider section is simply the
//! same store path with nothing to ignore. In-flight executions keep the
//! digest they verified against (`MuseExecution.provider_digest`) plus the
//! live store reference (`MuseExecution.provider_ref_token`) transferred
//! from the staging snapshot, so refresh/cleanup cannot collect the exact
//! bundle of a live run; re-resolution of that exact bundle for old-run
//! recovery goes through [`retained_muse_executable`], which fails closed
//! when the bundle is no longer admitted.
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

/// One coherent admitted snapshot for a NEW invocation sequence: the gated
/// selection, its verified executable, and the active store reference that
/// keeps the bundle alive while the host stages it into the running
/// Project. Provider/version/digest/ref all come from this single value;
/// the moving current record is never re-read afterwards. A concurrent
/// admission between the gate and the pin fails this invocation closed
/// (the next invocation snapshots fresh) instead of staging drifted bytes.
pub struct ProviderSnapshot {
    pub selection: soda_provider_tools::store::Selection,
    pub executable: std::path::PathBuf,
    token: Option<soda_provider_tools::resolve::RefToken>,
}

/// Take the single coherent selection snapshot for NEW work and pin its
/// exact bundle: gate the stored selection, pin that same digest under the
/// store lock, then confirm no concurrent admission replaced it in between.
/// `Stale` (a newer release is known but unadmitted) refuses visibly so the
/// operator refreshes; `Unknown` or an unverifiable bundle denies.
pub fn fresh_provider_snapshot() -> Result<ProviderSnapshot, String> {
    // Shared fresh authority: read, refresh when the check is due, gate. An
    // indefinitely old `Current` performs the upstream check here instead of
    // launching stale bytes; `Unknown` or known-newer fails closed. This
    // reuses the same due-check path `expected_muse_identity` uses — no
    // third selection gate is invented.
    let first = fresh_selection().map_err(map_store_error)?;
    let root = soda_provider_tools::config::store_root();
    let (executable, token) = soda_provider_tools::resolve::resolve_pinned(
        soda_provider_tools::Provider::Muse,
        &first.release.digest,
    )
    .map_err(map_store_error)?;
    let snapshot = ProviderSnapshot {
        selection: first,
        executable,
        token: Some(token),
    };
    // Confirm the pin still matches the live record: a concurrent admission
    // (new digest, or the same digest flipped Stale by a failed check while
    // a newer release is known) fails this invocation closed. The next
    // invocation snapshots the new world. Never stage drifted bytes.
    let live =
        soda_provider_tools::store::read_selection(&root, soda_provider_tools::Provider::Muse)
            .map_err(|_| terminal::err_denied())?;
    let current = match live {
        Some(current) => current,
        None => {
            release_provider_snapshot(snapshot);
            return Err(terminal::err_denied());
        }
    };
    if current.release.digest != snapshot.selection.release.digest {
        // A concurrent admission moved the world: deny this invocation so
        // the next one snapshots the new record. No operator refresh can
        // fix a race, so this stays a plain denial, not a stale marker.
        release_provider_snapshot(snapshot);
        return Err(terminal::err_denied());
    }
    if let Err(blocked) = gate_snapshot(&current) {
        release_provider_snapshot(snapshot);
        return Err(map_store_error(blocked));
    }
    // Bind the refreshed authority (`first`), not the re-read: same digest,
    // but `first` carries the outcome of the due check that just ran.
    Ok(snapshot)
}

/// Gate one selection snapshot for NEW work: `Current` and `UpdateFailed`
/// (no newer release confirmed) may run their retained bytes; `Stale` and
/// `Unknown` block.
fn gate_snapshot(
    selection: &soda_provider_tools::store::Selection,
) -> Result<(), soda_provider_tools::Error> {
    use soda_provider_tools::store::Freshness;
    match &selection.freshness {
        Freshness::Current | Freshness::UpdateFailed { .. } => Ok(()),
        Freshness::Stale { latest } => Err(soda_provider_tools::Error::blocked(format!(
            "known newer muse {latest} blocks new muse work; refresh before launching",
        ))),
        Freshness::Unknown { reason } => Err(soda_provider_tools::Error::blocked(format!(
            "no usable muse selection for new work ({reason})",
        ))),
    }
}

/// Release the snapshot's active store reference. The bytes already staged
/// into the guest are independent of store cleanup; in-flight guest
/// processes keep their open inode. Idempotent by consumption.
pub fn release_provider_snapshot(snapshot: ProviderSnapshot) {
    if let Some(token) = snapshot.token {
        let _ = soda_provider_tools::resolve::release(&token);
    }
}

impl ProviderSnapshot {
    /// Transfer the snapshot's active store reference out of the snapshot
    /// into the execution that now owns the staged bundle. After this
    /// returns `Some`, `release_provider_snapshot` on this snapshot is a
    /// no-op; the execution owner must call [`release_provider_ref`] after
    /// confirmed retirement, and must NOT release while retirement is
    /// uncertain (a conservative leak beats collecting a live bundle).
    pub fn take_ref_token(&mut self) -> Option<soda_provider_tools::resolve::RefToken> {
        self.token.take()
    }
}

/// Release one execution-owned store reference by its retained digest and
/// token name (the [`MuseExecution`](super::MuseExecution) pair). This is
/// the only release path for a live execution: call it after
/// `stop_execution` confirms retirement, never when cleanup is
/// unconfirmed. Idempotent (`NotFound` is `Ok`), so an accidental second
/// release is safe; an empty token name is a no-op for unstamped runs.
pub fn release_provider_ref(provider_digest: &str, ref_token: &str) {
    if ref_token.is_empty() {
        return;
    }
    let token = soda_provider_tools::resolve::RefToken {
        token: ref_token.to_string(),
        provider: soda_provider_tools::Provider::Muse,
        digest: provider_digest.to_string(),
    };
    let _ = soda_provider_tools::resolve::release(&token);
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
