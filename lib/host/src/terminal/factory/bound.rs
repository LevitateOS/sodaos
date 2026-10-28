//! Exact bound harness identity for factory reservations (V6 lane A).
//!
//! NEW reservations snapshot the CURRENT shared provider-tools selection
//! (due-check + stale refusal) into one coherent value: provider, version,
//! package digest, staged-executable digest, the verified executable path
//! and the live store reference that keeps the bundle alive across
//! refresh/cleanup. Recovery binds the RECORDED sidecar identity through
//! `resolve_existing` and fails closed when that bundle is gone.
//!
//! The startup-configured harness path/pin is never consulted here: terminal
//! staging uses the bound package path and executable digest only. The
//! package artifact digest (for example the Codex archive digest naming the
//! bundle directory) is separate metadata from the staged executable digest
//! carried by the wire pin; the two are never conflated.
//!
//! Credential, model, effort, contributor, budget and history fields are
//! untouched: this module performs no credential reads and no model calls.

use soda_provider_tools::resolve::RefToken;
use soda_provider_tools::{Provider, Selection};
use std::path::{Path, PathBuf};

use crate::terminal;

/// Durable per-run record of the exact bundle a reservation is bound to.
/// Written to the factory private state at reserve time, read back for
/// recovery or duplicate reserves.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize, PartialEq, Eq)]
pub struct HarnessBind {
    pub family: String,
    pub version: String,
    pub package_digest: String,
    pub executable_sha256: String,
    pub ref_token: String,
}

impl HarnessBind {
    pub fn validate(&self) -> Result<(), String> {
        if provider_for_family(&self.family).is_err() {
            return Err("unsupported factory harness".to_string());
        }
        if !crate::factory::valid_harness_version(&self.version) {
            return Err("unsupported factory harness".to_string());
        }
        for digest in [&self.package_digest, &self.executable_sha256] {
            if digest.len() != 64 || !digest.bytes().all(|b| b.is_ascii_hexdigit()) {
                return Err("invalid harness bind digest".to_string());
            }
        }
        if self.ref_token.is_empty()
            || self.ref_token.len() > 128
            || !self
                .ref_token
                .bytes()
                .all(|b| b.is_ascii_alphanumeric() || b == b'-' || b == b'_')
        {
            return Err("invalid harness bind reference".to_string());
        }
        Ok(())
    }
}

/// Private-state file holding one run's exact harness binding.
pub fn bind_name(project: &str, run: &str) -> Result<String, String> {
    if !crate::domain::valid_id(project) || !crate::factory::valid_factory_run_id(run) {
        return Err("invalid harness bind identity".to_string());
    }
    Ok(format!("harness-{project}-{run}.json"))
}

/// One coherent admitted snapshot for a NEW reservation: the gated
/// selection, its verified executable, the executable digest the wire pin
/// carries, and the active store reference that keeps the exact bundle
/// alive while the host stages it into the container. Provider, version,
/// package digest and executable digest all come from this single value;
/// the moving current record is never re-read afterwards.
pub struct BoundBundle {
    pub provider: Provider,
    pub version: String,
    pub package_digest: String,
    pub executable_sha256: String,
    pub executable: PathBuf,
    pub bundle_dir: PathBuf,
    token: Option<RefToken>,
}

impl BoundBundle {
    /// Transfer the snapshot's active store reference out of the snapshot
    /// into the reservation that now owns the staged bundle. After this
    /// returns `Some`, `release_bound` on this snapshot is a no-op; the new
    /// owner must release after confirmed retirement, and must NOT release
    /// while retirement is uncertain (a conservative leak beats collecting
    /// a live bundle).
    pub fn take_ref_token(&mut self) -> Option<RefToken> {
        self.token.take()
    }

    fn release_token(&mut self) {
        if let Some(token) = self.token.take() {
            let _ = soda_provider_tools::resolve::release(&token);
        }
    }
}

/// Release the snapshot's active store reference. The bytes already staged
/// into the container are independent of store cleanup; in-flight container
/// processes keep their open inode. Idempotent by consumption.
pub fn release_bound(mut bound: BoundBundle) {
    bound.release_token();
}

/// Release one reservation-owned store reference by its retained package
/// digest and token name (the [`HarnessBind`] pair). This is the only
/// release path for a bound run: call it after confirmed retirement, never
/// when cleanup is unconfirmed. Idempotent (`NotFound` is `Ok`), so an
/// accidental second release is safe; an empty token name is a no-op.
pub fn release_harness_ref(provider: Provider, package_digest: &str, ref_token: &str) {
    if ref_token.is_empty() {
        return;
    }
    let token = RefToken {
        token: ref_token.to_string(),
        provider,
        digest: package_digest.to_string(),
    };
    let _ = soda_provider_tools::resolve::release(&token);
}

pub fn provider_for_family(family: &str) -> Result<Provider, String> {
    match family {
        crate::factory::FACTORY_HARNESS_MUSE => Ok(Provider::Muse),
        _ => Err("unsupported factory harness".to_string()),
    }
}

/// Take the single coherent selection snapshot for NEW work: read the
/// current record, refresh when the check is due, then gate it. This
/// mirrors the launch semantics of
/// `soda_provider_tools::resolve::resolve_new` but returns the admitted
/// `Selection` itself, so package digest and version stay paired from one
/// value. The moving current record is never re-read afterwards.
fn fresh_selection_for(provider: Provider) -> Result<Selection, soda_provider_tools::Error> {
    let root = soda_provider_tools::config::store_root();
    let ttl = soda_provider_tools::config::check_ttl();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0);
    let stored = soda_provider_tools::store::read_selection(&root, provider)?;
    let due = match &stored {
        None => true,
        Some(selection) => now.saturating_sub(selection.checked_at_secs) > ttl.as_secs(),
    };
    let selection = if due {
        soda_provider_tools::refresh::refresh(provider)?
    } else {
        stored.expect("selection exists when refresh is not due")
    };
    gate_new(&selection)?;
    Ok(selection)
}

/// Gate one selection snapshot for NEW work: `Current` and `UpdateFailed`
/// (no newer release confirmed) may run their retained bytes; `Stale` and
/// `Unknown` block new work visibly instead of falling back.
fn gate_new(selection: &Selection) -> Result<(), soda_provider_tools::Error> {
    match &selection.freshness {
        soda_provider_tools::Freshness::Current
        | soda_provider_tools::Freshness::UpdateFailed { .. } => Ok(()),
        soda_provider_tools::Freshness::Stale { latest } => {
            Err(soda_provider_tools::Error::blocked(format!(
                "known newer {} {} blocks new {} work; refresh before launching",
                selection.provider.as_str(),
                latest,
                selection.provider.as_str(),
            )))
        }
        soda_provider_tools::Freshness::Unknown { reason } => {
            Err(soda_provider_tools::Error::blocked(format!(
                "no usable {} selection for new work ({reason})",
                selection.provider.as_str(),
            )))
        }
    }
}

fn map_store_error(err: soda_provider_tools::Error) -> String {
    match err {
        // Known-newer-blocked stays visible as stale; existing runs keep
        // their bound exact bundle.
        soda_provider_tools::Error::Blocked(_) => terminal::err_stale(),
        _ => terminal::err_denied(),
    }
}

/// Lowercase hex sha256 of one verified bundle file.
fn file_digest(path: &Path) -> Result<String, String> {
    let data = std::fs::read(path).map_err(|_| terminal::err_denied())?;
    Ok(crate::sha256::hex_lower(&crate::sha256::digest(&data)))
}

/// Take the single coherent snapshot for a NEW reservation and pin its
/// exact bundle: gate the stored selection, pin that same digest under the
/// store lock, then confirm no concurrent admission replaced it in between.
/// `Stale` (a newer release is known but unadmitted) refuses visibly;
/// `Unknown` or an unverifiable bundle denies.
pub fn resolve_current(family: &str) -> Result<BoundBundle, String> {
    let provider = provider_for_family(family)?;
    // Shared fresh authority: read, refresh when the check is due, gate.
    let first = fresh_selection_for(provider).map_err(map_store_error)?;
    let root = soda_provider_tools::config::store_root();
    let (executable, token) =
        soda_provider_tools::resolve::resolve_pinned(provider, &first.release.digest)
            .map_err(map_store_error)?;
    let version = first.release.version.clone();
    let package_digest = first.release.digest.clone();
    let mut bound = BoundBundle {
        provider,
        version,
        package_digest,
        executable_sha256: String::new(),
        executable,
        bundle_dir: first.bundle_dir(&root),
        token: Some(token),
    };
    // Confirm the pin still matches the live record: a concurrent admission
    // (new digest, or the same digest flipped Stale by a failed check while
    // a newer release is known) fails this reservation closed. The next
    // reservation snapshots the new world. Never stage drifted bytes.
    let live = soda_provider_tools::store::read_selection(&root, provider)
        .map_err(|_| terminal::err_denied())?;
    let current = match live {
        Some(current) => current,
        None => {
            release_bound(bound);
            return Err(terminal::err_denied());
        }
    };
    if current.release.digest != bound.package_digest {
        release_bound(bound);
        return Err(terminal::err_denied());
    }
    if let Err(blocked) = gate_new(&current) {
        release_bound(bound);
        return Err(map_store_error(blocked));
    }
    // Bind the refreshed authority (`first`), not the re-read: same digest,
    // but `first` carries the outcome of the due check that just ran.
    match file_digest(&bound.executable) {
        Ok(digest) => {
            bound.executable_sha256 = digest;
            Ok(bound)
        }
        Err(err) => {
            release_bound(bound);
            Err(err)
        }
    }
}

/// Expected `(version, executable digest)` for the CURRENT shared selection,
/// without pinning a reference: for pin queries, which reserve nothing.
/// Fails closed exactly like [`resolve_current`].
pub fn current_pin(family: &str) -> Result<(String, String), String> {
    let provider = provider_for_family(family)?;
    let selection = fresh_selection_for(provider).map_err(map_store_error)?;
    let executable =
        soda_provider_tools::resolve::resolve_existing(provider, &selection.release.digest)
            .map_err(map_store_error)?;
    let digest = file_digest(&executable)?;
    Ok((selection.release.version.clone(), digest))
}

/// Exact retained bundle for EXISTING work by recorded package digest
/// (plus the recorded executable digest for byte-exact confirmation).
/// Fails closed when that bundle is no longer admitted; callers must not
/// fall back to the current selection here. Acquires a fresh live
/// reference for the recovered reservation; the caller owns it.
pub fn bind_recorded(bind: &HarnessBind) -> Result<BoundBundle, String> {
    bind.validate()?;
    let provider = provider_for_family(&bind.family)?;
    let executable = soda_provider_tools::resolve::resolve_existing(provider, &bind.package_digest)
        .map_err(|_| terminal::err_denied())?;
    if file_digest(&executable)? != bind.executable_sha256 {
        return Err(terminal::err_denied());
    }
    let token = soda_provider_tools::resolve::acquire(provider, &bind.package_digest)
        .map_err(|_| terminal::err_denied())?;
    // The bundle directory is the verified executable's grandparent:
    // `<root>/bundles/<provider>/<package>/bin/<cli>`. Deriving it from
    // the verified path keeps it exact instead of rebuilding it by hand.
    let bundle_dir = executable
        .parent()
        .and_then(|bin| bin.parent())
        .ok_or_else(terminal::err_denied)?
        .to_path_buf();
    Ok(BoundBundle {
        provider,
        version: bind.version.clone(),
        package_digest: bind.package_digest.clone(),
        executable_sha256: bind.executable_sha256.clone(),
        executable,
        bundle_dir,
        token: Some(token),
    })
}
