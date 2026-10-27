// Store-backed provider executable resolution (UP-20261026-v1, lane B).
//
// Coherence protocol (UP-REVIEW-02 finding 2): every store-mode resolve
// captures ONE admitted identity — binary, version, digest, and cleanup
// reference — and every downstream caller uses that immutable snapshot with
// no re-read of the moving current record:
//
// 1. Take a single `Selection` snapshot: read the current record, or
//    refresh when the check is due, then gate it (known-stale or unknown
//    blocks new work). The digest AND the version both come from this one
//    value, so a refresh landing between two reads can never pair an old
//    binary with a new version.
// 2. Cross-check the snapshot against the bundle's own immutable admission
//    record (`release.json` under the digest-addressed bundle directory):
//    version drift fails closed instead of misattributing exact bytes.
// 3. Bind with `resolve_pinned`: bundle verification plus reference
//    publication under a single store-lock hold, so cleanup cannot collect
//    the bundle in the gap between resolving and acquiring.
//
// Explicit `binary` + `sha256` configuration keeps its historical behavior
// byte-for-byte (fixtures and the broker compat test rely on it). Since V2-A
// the broker routes every enabled section to store mode, so legacy pins only
// reach this module as historical attribution, never as new-session bindings.
// Setting `store = true` switches a provider to the shared provider store: every new
// session resolves the fresh selection through `resolve_new`, binds the exact
// admitted bundle, and releases it when the session closes. A refresh
// therefore never corrupts an active enrollment, and new sessions pick up
// the refreshed selection without a broker restart. Recovery of an in-flight
// run re-resolves its bound digest through `resolve_existing`, which binds
// the exact retained bundle (never the current selection) and fails closed
// when that bundle is no longer admitted.
//
// Credential fixtures and model/profile/budget fields are untouched: this
// module performs no credential reads and no model calls.
//
// Lane A owns `lib/provider-tools`: this module only consumes its public
// API (`store::read_selection`, `store::verify_bundle`,
// `resolve::resolve_pinned`, `refresh::refresh`, `config::*`). The small
// refresh-due/gating mirror below exists because no public snapshot helper
// exists yet; see the report note asking lane A for `resolve_snapshot`.

use super::Error;
use std::path::{Path, PathBuf};

/// Which provider store selection to resolve.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProviderKind {
    Codex,
    Muse,
}

impl ProviderKind {
    pub fn tool(self) -> soda_provider_tools::Provider {
        match self {
            ProviderKind::Codex => soda_provider_tools::Provider::Codex,
            ProviderKind::Muse => soda_provider_tools::Provider::Muse,
        }
    }

    pub fn id(self) -> &'static str {
        self.tool().as_str()
    }
}

/// One session's executable binding: the exact path to spawn plus the
/// admitted identity behind it. `token` pins the bundle against cleanup for
/// the session lifetime; it is released when the session closes.
#[derive(Debug, Clone)]
pub struct BoundBinary {
    pub binary: PathBuf,
    pub version: String,
    pub digest: String,
    pub via_store: bool,
    pub token: Option<soda_provider_tools::resolve::RefToken>,
}

/// Take the single coherent selection snapshot for NEW work: read the
/// current record, refresh when the check is due, then gate it. This mirrors
/// the launch semantics of `soda_provider_tools::resolve::resolve_new` but
/// returns the admitted `Selection` itself, so digest and version stay paired
/// from one value. Callers must not re-read the moving current record after
/// this returns.
fn fresh_selection(
    provider: soda_provider_tools::Provider,
) -> Result<soda_provider_tools::Selection, soda_provider_tools::Error> {
    use soda_provider_tools::{config, refresh, store};
    let root = config::store_root();
    let ttl = config::check_ttl();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0);
    let stored = store::read_selection(&root, provider)?;
    let due = match &stored {
        None => true,
        Some(selection) => now.saturating_sub(selection.checked_at_secs) > ttl.as_secs(),
    };
    let selection = if due {
        refresh::refresh(provider)?
    } else {
        stored.expect("selection exists when refresh is not due")
    };
    match &selection.freshness {
        store::Freshness::Current | store::Freshness::UpdateFailed { .. } => Ok(selection),
        store::Freshness::Stale { latest } => Err(soda_provider_tools::Error::blocked(format!(
            "known newer {} {} blocks new {} work; refresh before launching",
            provider.as_str(),
            latest,
            provider.as_str()
        ))),
        store::Freshness::Unknown { reason } => Err(soda_provider_tools::Error::blocked(format!(
            "no usable {} selection for new work ({reason})",
            provider.as_str()
        ))),
    }
}

/// Version recorded in a bundle's own immutable admission record
/// (`release.json` under the digest-addressed bundle directory). The bundle
/// tree is never mutated after admission, so this read cannot race a
/// refresh; it only fails closed on corrupt knowledge.
fn admitted_version(
    root: &Path,
    provider: soda_provider_tools::Provider,
    digest: &str,
) -> Result<String, soda_provider_tools::Error> {
    let path = root
        .join("bundles")
        .join(provider.as_str())
        .join(digest.to_lowercase())
        .join("release.json");
    let bytes = std::fs::read(&path).map_err(|e| {
        soda_provider_tools::Error::denied(format!(
            "admitted {} bundle record unreadable for digest {digest}: {e}",
            provider.as_str()
        ))
    })?;
    let record: soda_provider_tools::ReleaseIdentity =
        serde_json::from_slice(&bytes).map_err(|e| {
            soda_provider_tools::Error::denied(format!(
                "admitted {} bundle record corrupt for digest {digest}: {e}",
                provider.as_str()
            ))
        })?;
    if record.provider != provider
        || !record.digest.eq_ignore_ascii_case(digest)
        || record.version.is_empty()
    {
        return Err(soda_provider_tools::Error::denied(format!(
            "admitted {} bundle identity invalid for digest {digest}",
            provider.as_str()
        )));
    }
    Ok(record.version)
}

/// Resolve the fresh selected executable for NEW work (T00 `resolve_new`)
/// as ONE coherent admitted identity. Blocks fail-closed when the provider
/// is known-stale or its update failed while a newer release is known: the
/// broker surfaces that as unavailable instead of running a stale pin.
pub fn resolve_new(provider: ProviderKind) -> Result<BoundBinary, Error> {
    let tool = provider.tool();
    let root = soda_provider_tools::config::store_root();
    // ONE snapshot: digest and version are paired here, never re-read.
    let selection = fresh_selection(tool).map_err(map_store_error)?;
    let admitted = admitted_version(&root, tool, &selection.release.digest)
        .map_err(map_store_error)?;
    if admitted != selection.release.version {
        return Err(Error::failed(format!(
            "{} selection version drifted from its admitted bundle; refresh before launching",
            provider.id()
        )));
    }
    let digest = selection.release.digest.clone();
    let version = selection.release.version.clone();
    // Atomic bind under a single store-lock hold (verify + reference
    // publish): cleanup cannot collect the bundle in between.
    let (binary, token) =
        soda_provider_tools::resolve::resolve_pinned(tool, &digest).map_err(map_store_error)?;
    debug_assert_eq!(
        digest_of_executable(&root, tool, &binary).as_deref(),
        Some(digest.as_str())
    );
    Ok(BoundBinary {
        binary,
        version,
        digest,
        via_store: true,
        token: Some(token),
    })
}

/// Bind the exact retained bundle for EXISTING work (T00 `resolve_existing`)
/// as ONE coherent admitted identity. Digest-addressed only: the current
/// selection is never consulted, so recovery keeps its exact retained bytes
/// across refreshes. The returned token pins the bundle for the caller's
/// session lifetime; release it with [`release_bound`]. Fails closed when
/// that bundle is no longer admitted; callers must not fall back to the
/// current selection here.
pub fn resolve_existing(provider: ProviderKind, digest: &str) -> Result<BoundBinary, Error> {
    if digest.len() != 64 || !digest.bytes().all(|b| b.is_ascii_hexdigit()) {
        return Err(Error::denied("invalid retained provider bundle"));
    }
    let tool = provider.tool();
    let root = soda_provider_tools::config::store_root();
    let digest = digest.to_lowercase();
    // Atomic bind under a single store-lock hold, same as new work: the
    // bundle cannot be collected between verification and pinning.
    let (binary, token) =
        soda_provider_tools::resolve::resolve_pinned(tool, &digest).map_err(map_store_error)?;
    // The version comes from the bundle's own immutable admission record,
    // never from the moving current selection.
    let version = admitted_version(&root, tool, &digest).map_err(map_store_error)?;
    debug_assert_eq!(
        digest_of_executable(&root, tool, &binary).as_deref(),
        Some(digest.as_str())
    );
    Ok(BoundBinary {
        binary,
        version,
        digest,
        via_store: true,
        token: Some(token),
    })
}

/// Release a session binding. A release failure keeps the bundle pinned
/// (fail-safe against cleanup); it never fails session teardown.
pub fn release_bound(bound: &mut BoundBinary) {
    if let Some(token) = bound.token.take() {
        if let Err(err) = soda_provider_tools::resolve::release(&token) {
            eprintln!(
                "provider ref release for bundle {} failed: {err}",
                bound.digest
            );
        }
    }
}

/// Extract the bundle digest from an admitted executable path
/// `<root>/bundles/<provider>/<digest>/...`. Returns `None` when the path
/// is not inside the configured store layout, so explicit (non-store)
/// binaries are never mistaken for admitted bundles.
pub fn digest_of_executable(
    root: &Path,
    provider: soda_provider_tools::Provider,
    exe: &Path,
) -> Option<String> {
    let rel = exe.strip_prefix(root).ok()?;
    let mut parts = rel.components();
    let bundles = parts.next()?.as_os_str().to_str()?;
    let name = parts.next()?.as_os_str().to_str()?;
    let digest = parts.next()?.as_os_str().to_str()?;
    if bundles != "bundles" || name != provider.as_str() {
        return None;
    }
    if digest.len() != 64 || !digest.bytes().all(|b| b.is_ascii_hexdigit()) {
        return None;
    }
    // The digest must be followed by the provider-relative payload path
    // (e.g. `bin/codex`), never the bundle root itself.
    if parts.next().is_none() {
        return None;
    }
    Some(digest.to_string())
}

fn map_store_error(err: soda_provider_tools::Error) -> Error {
    use soda_provider_tools::Error as Store;
    match err {
        // Known-newer-blocked stays visible as a failure (the broker maps
        // plain provider failures to `unavailable`); existing runs keep
        // their bound exact bundle.
        Store::Blocked(message) => Error::failed(message),
        Store::Denied(message) => Error::denied(message),
        Store::Uncertain(message) => Error::uncertain(message),
        Store::Failed(message) => Error::failed(message),
    }
}
