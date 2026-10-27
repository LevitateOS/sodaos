//! Launch-time resolution and active-run references (lane A).
//!
//! New sessions resolve through [`resolve_new`], which refreshes when the
//! check is due and blocks when the provider is known-stale or the update
//! failed while a newer release is known. In-flight sessions and recovery
//! keep their bound exact bundle via [`resolve_existing`] plus an active
//! reference token that [`cleanup`](crate::store::cleanup) must not collect.

use crate::store::Provider;
use std::path::PathBuf;

/// Opaque handle keeping one exact bundle alive across refresh/cleanup.
#[derive(Debug, Clone)]
pub struct RefToken {
    /// Token file name under `<root>/refs/`.
    pub token: String,
    pub provider: Provider,
    pub digest: String,
}

/// Resolve the fresh selected executable for NEW work. Refreshes the provider
/// when its check is due; blocks (Blocked) when known stale, unknown, or when
/// the refresh itself reports a known-newer release. Never falls back silently.
pub fn resolve_new(provider: Provider) -> Result<PathBuf, crate::Error> {
    let root = crate::config::store_root();
    let ttl = crate::config::check_ttl();
    let now = crate::now_secs();
    let selection = crate::store::read_selection(&root, provider)?;
    let due = match &selection {
        None => true,
        Some(sel) => now.saturating_sub(sel.checked_at_secs) > ttl.as_secs(),
    };
    if due {
        let refreshed = crate::refresh::refresh(provider)?;
        return executable_for_refreshed(&root, &refreshed);
    }
    let sel = selection.expect("selection exists when refresh is not due");
    match &sel.freshness {
        crate::store::Freshness::Current | crate::store::Freshness::UpdateFailed { .. } => {
            executable_for(&root, &sel)
        }
        crate::store::Freshness::Stale { latest } => Err(crate::Error::blocked(format!(
            "known newer {} {} blocks new {} work; refresh before launching",
            provider.as_str(),
            latest,
            provider.as_str()
        ))),
        crate::store::Freshness::Unknown { reason } => Err(crate::Error::blocked(format!(
            "no usable {} selection for new work ({reason})",
            provider.as_str()
        ))),
    }
}

/// Gate a freshly refreshed selection: only `Current`/`UpdateFailed` resolve
/// to an executable. A refresh that preserved `Stale` (failed check while a
/// newer release is known) or `Unknown` blocks new work.
fn executable_for_refreshed(
    root: &std::path::Path,
    refreshed: &crate::store::Selection,
) -> Result<PathBuf, crate::Error> {
    match &refreshed.freshness {
        crate::store::Freshness::Current | crate::store::Freshness::UpdateFailed { .. } => {
            executable_for(root, refreshed)
        }
        crate::store::Freshness::Stale { latest } => Err(crate::Error::blocked(format!(
            "known newer {} {} blocks new {} work; refresh before launching",
            refreshed.provider.as_str(),
            latest,
            refreshed.provider.as_str()
        ))),
        crate::store::Freshness::Unknown { reason } => Err(crate::Error::blocked(format!(
            "no usable {} selection for new work ({reason})",
            refreshed.provider.as_str()
        ))),
    }
}

fn executable_for(
    root: &std::path::Path,
    selection: &crate::store::Selection,
) -> Result<PathBuf, crate::Error> {
    crate::store::verify_selected(root, selection).map_err(|e| {
        if e.blocks_new_work() {
            e
        } else {
            crate::Error::failed(format!(
                "selected {} {} bundle failed verification ({}); refresh before launching",
                selection.provider.as_str(),
                selection.release.version,
                e
            ))
        }
    })
}

/// Resolve the exact retained executable for EXISTING work by digest.
/// Verifies the admission record plus per-file digests/permissions/paths
/// (never a bare hash-shaped directory); fails closed when that bundle is
/// not admitted or no longer verifies.
pub fn resolve_existing(provider: Provider, digest: &str) -> Result<PathBuf, crate::Error> {
    let root = crate::config::store_root();
    crate::store::verify_bundle(&root, provider, digest)
}

/// Resolve and pin the exact bundle in one locked transaction: the store lock
/// is held across verification and reference publication, so cleanup cannot
/// collect the bundle between resolution and acquire. Returns the verified
/// executable plus the live reference token. Lane B builds on this.
pub fn resolve_pinned(provider: Provider, digest: &str) -> Result<(PathBuf, RefToken), crate::Error> {
    let root = crate::config::store_root();
    let _lock = crate::store::lock_store(&root)?;
    let exe = crate::store::verify_bundle(&root, provider, digest)?;
    let token = acquire_locked(&root, provider, digest)?;
    Ok((exe, token))
}

/// Create an active reference so cleanup preserves this bundle until release.
/// Holds the store lock across verification and reference publication: a
/// concurrent cleanup cannot delete the bundle in between.
pub fn acquire(provider: Provider, digest: &str) -> Result<RefToken, crate::Error> {
    let root = crate::config::store_root();
    let _lock = crate::store::lock_store(&root)?;
    acquire_locked(&root, provider, digest)
}

fn acquire_locked(
    root: &std::path::Path,
    provider: Provider,
    digest: &str,
) -> Result<RefToken, crate::Error> {
    // Fail closed first: only verified admitted exact bundles can be referenced.
    crate::store::verify_bundle(root, provider, digest)?;
    let digest = crate::normalize_digest(digest)?;
    std::fs::create_dir_all(crate::store::refs_dir(root))?;
    for _ in 0..8 {
        let token = new_token(provider);
        let record = crate::store::RefRecord {
            token: token.clone(),
            provider,
            digest: digest.clone(),
            created_at_secs: crate::now_secs(),
            pid: std::process::id(),
        };
        match crate::store::write_ref_record(root, &record) {
            Ok(()) => {
                return Ok(RefToken {
                    token,
                    provider,
                    digest,
                });
            }
            Err(e) if e.to_string().starts_with("ref token already exists") => continue,
            Err(e) => return Err(e),
        }
    }
    Err(crate::Error::failed(
        "could not allocate a unique ref token",
    ))
}

fn new_token(provider: Provider) -> String {
    let mut rand = [0u8; 8];
    let mut have_rand = false;
    if let Ok(mut f) = std::fs::File::open("/dev/urandom") {
        use std::io::Read as _;
        let mut buf = [0u8; 8];
        if f.read_exact(&mut buf).is_ok() {
            rand = buf;
            have_rand = true;
        }
    }
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.subsec_nanos())
        .unwrap_or(0);
    let suffix: String = rand.iter().map(|b| format!("{b:02x}")).collect();
    format!(
        "rt-{}-{}-{}{}",
        provider.as_str(),
        std::process::id(),
        if have_rand {
            String::new()
        } else {
            format!("{nanos:x}-")
        },
        suffix
    )
}

/// Release a previously acquired reference. Holds the store lock so the
/// removal serializes with cleanup's protection scan.
pub fn release(token: &RefToken) -> Result<(), crate::Error> {
    let root = crate::config::store_root();
    let _lock = crate::store::lock_store(&root)?;
    let path = crate::store::ref_path(&root, &token.token)?;
    match std::fs::remove_file(&path) {
        Ok(()) => Ok(()),
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Err(e) => Err(crate::Error::uncertain(format!(
            "ref release could not be confirmed ({}): {e}",
            path.display()
        ))),
    }
}
