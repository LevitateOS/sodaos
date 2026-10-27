//! Retained store: immutable bundles, atomic selection, references (T00 frozen).
//!
//! Reads distinguish current, stale, unknown and update-failed. Selection
//! records carry both the last successful check and the last successful
//! admission; a failed attempt updates neither.

use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};

/// Supported provider CLIs.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Provider {
    Codex,
    Muse,
}

impl Provider {
    pub fn as_str(self) -> &'static str {
        match self {
            Provider::Codex => "codex",
            Provider::Muse => "muse",
        }
    }

    pub fn parse(name: &str) -> Option<Provider> {
        match name {
            "codex" => Some(Provider::Codex),
            "muse" => Some(Provider::Muse),
            _ => None,
        }
    }

    /// Official upstream metadata endpoint for the configured channel.
    /// Codex: GitHub releases `latest`; Muse: Meta stable channel manifest.
    pub fn metadata_url(self, channel: &str) -> String {
        match self {
            Provider::Codex => {
                let _ = channel;
                "https://api.github.com/repos/openai/codex/releases/latest".to_string()
            }
            Provider::Muse => {
                if channel == "stable" {
                    "https://api.meta.ai/muse-code/channels/muse-stable".to_string()
                } else {
                    format!("https://api.meta.ai/muse-code/channels/{channel}")
                }
            }
        }
    }

    /// Native executable path inside an admitted bundle directory.
    pub fn executable_relpath(self) -> &'static str {
        match self {
            Provider::Codex => "bin/codex",
            Provider::Muse => "bin/muse",
        }
    }
}

/// One official upstream release: identity of exact bytes, never a
/// compatibility guess. `digest` is lowercase hex sha256 of the admitted
/// artifact; `size` is the exact byte count.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ReleaseIdentity {
    pub provider: Provider,
    pub version: String,
    pub digest: String,
    pub size: u64,
    pub artifact_url: String,
}

/// Freshness of a provider selection as observed by the last check.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(tag = "state", rename_all = "snake_case")]
pub enum Freshness {
    /// Selection matches the latest successful upstream check.
    Current,
    /// Upstream is newer (or the update failed while a newer release is
    /// known): new work is blocked until corrected.
    Stale { latest: String },
    /// No successful check has ever completed for this provider.
    Unknown { reason: String },
    /// The last update attempt failed and no newer release is confirmed;
    /// existing work continues, but fresh qualification evidence is absent.
    UpdateFailed { reason: String },
}

/// The atomically replaced selection record: `current-<provider>.json`.
/// `PartialEq`/`Eq` support exact base comparison for compare-and-swap
/// publication; the serialized shape is unchanged.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Selection {
    pub provider: Provider,
    pub release: ReleaseIdentity,
    /// Seconds since epoch of the last SUCCESSFUL upstream check.
    pub checked_at_secs: u64,
    /// Seconds since epoch of the successful admission of this release.
    pub admitted_at_secs: u64,
    pub freshness: Freshness,
}

impl Selection {
    pub fn bundle_dir(&self, root: &Path) -> PathBuf {
        root.join("bundles")
            .join(self.provider.as_str())
            .join(&self.release.digest)
    }

    pub fn executable(&self, root: &Path) -> PathBuf {
        self.bundle_dir(root)
            .join(self.provider.executable_relpath())
    }
}

/// Overall store status across providers.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Status {
    pub root: PathBuf,
    pub now_secs: u64,
    pub entries: Vec<Selection>,
}

/// Path of the atomically replaced selection record.
pub(crate) fn selection_path(root: &Path, provider: Provider) -> PathBuf {
    root.join(format!("current-{}.json", provider.as_str()))
}

/// Path of the last replaced selection, retained so cleanup keeps it.
pub(crate) fn previous_path(root: &Path, provider: Provider) -> PathBuf {
    root.join(format!("previous-{}.json", provider.as_str()))
}

/// Directory holding active exact-bundle reference tokens.
pub(crate) fn refs_dir(root: &Path) -> PathBuf {
    root.join("refs")
}

/// Check a selection record for corrupt or fail-open state.
fn validate_selection(selection: &Selection, provider: Provider) -> Result<(), crate::Error> {
    if selection.provider != provider {
        return Err(crate::Error::denied(format!(
            "selection provider mismatch: record says {}, asked {}",
            selection.provider.as_str(),
            provider.as_str()
        )));
    }
    validate_release(&selection.release, provider)?;
    Ok(())
}

/// Check a release identity for corrupt or fail-open state. Shared by
/// selection reads and bundle admission verification.
fn validate_release(release: &ReleaseIdentity, provider: Provider) -> Result<(), crate::Error> {
    if release.provider != provider {
        return Err(crate::Error::denied(format!(
            "release provider mismatch: record says {}, asked {}",
            release.provider.as_str(),
            provider.as_str()
        )));
    }
    crate::normalize_digest(&release.digest)?;
    if release.size == 0 {
        return Err(crate::Error::denied("release has zero artifact size"));
    }
    if !(release.artifact_url.starts_with("https://")
        || release.artifact_url.starts_with("http://localhost")
        || release.artifact_url.starts_with("http://127."))
    {
        return Err(crate::Error::denied("release artifact URL is not https"));
    }
    if release.version.is_empty() {
        return Err(crate::Error::denied("release has empty version"));
    }
    Ok(())
}

/// Read the current selection for a provider, or `None` when never admitted.
/// A corrupt record fails closed (`Denied`) instead of falling back.
pub fn read_selection(root: &Path, provider: Provider) -> Result<Option<Selection>, crate::Error> {
    let path = selection_path(root, provider);
    let bytes = match std::fs::read(&path) {
        Ok(bytes) => bytes,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(None),
        Err(e) => return Err(crate::Error::from(e)),
    };
    let selection: Selection = serde_json::from_slice(&bytes)?;
    validate_selection(&selection, provider)?;
    Ok(Some(selection))
}

/// Evaluate freshness of a selection against TTL and last outcome.
/// A failed check (`latest == None`) never refreshes timestamps and never
/// presents a fallback as latest: without a known-newer release the outcome
/// is `UpdateFailed`, while a stored `Stale` stays `Stale` so new work keeps
/// blocking until the update succeeds.
pub fn evaluate(
    selection: Option<&Selection>,
    latest: Option<&ReleaseIdentity>,
    now_secs: u64,
    ttl_secs: u64,
) -> Freshness {
    let Some(selection) = selection else {
        match latest {
            Some(latest) => {
                return Freshness::Unknown {
                    reason: format!(
                        "no {} bundle admitted yet; latest upstream is {}",
                        selection_provider_name(selection),
                        latest.version
                    ),
                };
            }
            None => {
                return Freshness::Unknown {
                    reason: "no bundle admitted and upstream check failed".to_string(),
                };
            }
        }
    };
    match latest {
        Some(latest) => {
            if latest.digest == selection.release.digest {
                Freshness::Current
            } else {
                Freshness::Stale {
                    latest: latest.version.clone(),
                }
            }
        }
        None => match &selection.freshness {
            Freshness::Stale { latest } => Freshness::Stale {
                latest: latest.clone(),
            },
            Freshness::Current
                if now_secs.saturating_sub(selection.checked_at_secs) <= ttl_secs =>
            {
                Freshness::Current
            }
            _ => Freshness::UpdateFailed {
                reason: format!(
                    "upstream check failed; retained {} {} without fresh qualification",
                    selection.provider.as_str(),
                    selection.release.version
                ),
            },
        },
    }
}

fn selection_provider_name(selection: Option<&Selection>) -> &'static str {
    match selection {
        Some(selection) => selection.provider.as_str(),
        None => "provider",
    }
}

/// Atomically replace `current-<provider>.json` (write temp + fsync + rename
/// + fsync dir). The record is validated before anything is written.
pub fn write_selection(root: &Path, selection: &Selection) -> Result<(), crate::Error> {
    validate_selection(selection, selection.provider)?;
    std::fs::create_dir_all(root)?;
    let dest = selection_path(root, selection.provider);
    let bytes = serde_json::to_vec_pretty(selection)?;
    atomic_write(&dest, &bytes)
}

/// Atomically replace `previous-<provider>.json` with the replaced record so
/// cleanup can retain the previous bundle. Best-effort companion to
/// [`write_selection`]; validated the same way.
pub(crate) fn write_previous(root: &Path, selection: &Selection) -> Result<(), crate::Error> {
    validate_selection(selection, selection.provider)?;
    std::fs::create_dir_all(root)?;
    let dest = previous_path(root, selection.provider);
    let bytes = serde_json::to_vec_pretty(selection)?;
    atomic_write(&dest, &bytes)
}

/// Cross-operation store lock guard. Dropping releases the flock(2).
/// Lane B coherence builds on this: hold across resolve+acquire
/// (`resolve::resolve_pinned`), refresh selection publication, and cleanup.
pub struct StoreLock {
    file: std::fs::File,
}

impl Drop for StoreLock {
    fn drop(&mut self) {
        use std::os::unix::io::AsRawFd as _;
        unsafe {
            libc::flock(self.file.as_raw_fd(), libc::LOCK_UN);
        }
    }
}

/// Acquire the exclusive store lock (`<root>/.lock`). Bounded wait: nonblocking
/// attempts for up to ~30s, then `Failed`. Never silent.
pub fn lock_store(root: &Path) -> Result<StoreLock, crate::Error> {
    std::fs::create_dir_all(root)?;
    let lock_path = root.join(".lock");
    let file = std::fs::OpenOptions::new()
        .create(true)
        .read(true)
        .write(true)
        .open(&lock_path)?;
    use std::os::unix::io::AsRawFd as _;
    let fd = file.as_raw_fd();
    for _ in 0..600 {
        let rc = unsafe { libc::flock(fd, libc::LOCK_EX | libc::LOCK_NB) };
        if rc == 0 {
            return Ok(StoreLock { file });
        }
        let err = std::io::Error::last_os_error();
        if err.kind() != std::io::ErrorKind::WouldBlock {
            return Err(crate::Error::failed(format!("store lock failed: {err}")));
        }
        std::thread::sleep(std::time::Duration::from_millis(50));
    }
    Err(crate::Error::failed(
        "store lock busy after 30s; another resolve/refresh/cleanup holds it",
    ))
}

fn atomic_write(dest: &Path, bytes: &[u8]) -> Result<(), crate::Error> {
    let parent = dest
        .parent()
        .ok_or_else(|| crate::Error::denied(format!("no parent directory: {}", dest.display())))?;
    std::fs::create_dir_all(parent)?;
    let mut tmp = tempfile::NamedTempFile::new_in(parent)?;
    use std::io::Write as _;
    tmp.write_all(bytes)?;
    tmp.flush()?;
    tmp.as_file()
        .sync_all()
        .map_err(|e| crate::Error::failed(format!("fsync failed: {e}")))?;
    tmp.persist(dest)
        .map_err(|e| crate::Error::failed(format!("atomic replace failed: {e}")))?;
    crate::fsync_dir(parent)?;
    Ok(())
}

/// Active reference record: `<root>/refs/<token>.json`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub(crate) struct RefRecord {
    pub token: String,
    pub provider: Provider,
    pub digest: String,
    pub created_at_secs: u64,
    pub pid: u32,
}

fn valid_token(token: &str) -> bool {
    !token.is_empty()
        && token.len() <= 128
        && token
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || c == '-' || c == '_')
}

pub(crate) fn ref_path(root: &Path, token: &str) -> Result<PathBuf, crate::Error> {
    if !valid_token(token) {
        return Err(crate::Error::denied("invalid ref token"));
    }
    Ok(refs_dir(root).join(format!("{token}.json")))
}

/// Create a reference record; fails when the token file already exists.
pub(crate) fn write_ref_record(root: &Path, record: &RefRecord) -> Result<(), crate::Error> {
    crate::normalize_digest(&record.digest)?;
    let dest = ref_path(root, &record.token)?;
    if let Some(parent) = dest.parent() {
        std::fs::create_dir_all(parent)?;
    }
    let bytes = serde_json::to_vec_pretty(record)?;
    let mut tmp = tempfile::NamedTempFile::new_in(refs_dir(root))?;
    use std::io::Write as _;
    tmp.write_all(&bytes)?;
    tmp.flush()?;
    tmp.as_file()
        .sync_all()
        .map_err(|e| crate::Error::failed(format!("fsync failed: {e}")))?;
    match tmp.persist_noclobber(&dest) {
        Ok(_) => {}
        Err(e) if e.error.kind() == std::io::ErrorKind::AlreadyExists => {
            return Err(crate::Error::failed(format!(
                "ref token already exists: {}",
                record.token
            )));
        }
        Err(e) => {
            return Err(crate::Error::failed(format!(
                "ref creation failed: {}",
                e.error
            )));
        }
    }
    crate::fsync_dir(&refs_dir(root))?;
    Ok(())
}

/// List reference records; a missing refs dir means no references. Strict:
/// any unreadable or corrupt `*.json` entry (or a token/filename mismatch) is
/// `Uncertain`, so cleanup fails before deleting anything it cannot see.
/// Non-JSON files are ignored.
pub(crate) fn list_ref_records(root: &Path) -> Result<Vec<RefRecord>, crate::Error> {
    let dir = refs_dir(root);
    let entries = match std::fs::read_dir(&dir) {
        Ok(entries) => entries,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(Vec::new()),
        Err(e) => {
            return Err(crate::Error::uncertain(format!(
                "refs unreadable ({}): {e}",
                dir.display()
            )));
        }
    };
    let mut out = Vec::new();
    for entry in entries {
        let entry = entry.map_err(|e| {
            crate::Error::uncertain(format!("refs listing unreadable ({}): {e}", dir.display()))
        })?;
        if entry.path().extension().and_then(|s| s.to_str()) != Some("json") {
            continue;
        }
        let stem = entry
            .path()
            .file_stem()
            .and_then(|s| s.to_str())
            .unwrap_or_default()
            .to_string();
        let bytes = std::fs::read(entry.path()).map_err(|e| {
            crate::Error::uncertain(format!("ref unreadable ({}): {e}", entry.path().display()))
        })?;
        let record: RefRecord = serde_json::from_slice(&bytes).map_err(|e| {
            crate::Error::uncertain(format!(
                "ref corrupt ({}): {e}; refusing to delete possibly referenced bundles",
                entry.path().display()
            ))
        })?;
        if !valid_token(&record.token) || record.token != stem {
            return Err(crate::Error::uncertain(format!(
                "ref token invalid or mismatched ({}); refusing to delete possibly referenced bundles",
                entry.path().display()
            )));
        }
        crate::normalize_digest(&record.digest).map_err(|e| {
            crate::Error::uncertain(format!(
                "ref digest invalid ({}): {e}; refusing to delete possibly referenced bundles",
                entry.path().display()
            ))
        })?;
        out.push(record);
    }
    Ok(out)
}

/// Read one selection file for cleanup: missing means never admitted; any
/// other IO problem or any corrupt/invalid record is `Uncertain` so cleanup
/// fails before deleting a bundle its knowledge cannot account for.
fn read_selection_file(path: &Path) -> Result<Option<Selection>, crate::Error> {
    let bytes = match std::fs::read(path) {
        Ok(bytes) => bytes,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(None),
        Err(e) => {
            return Err(crate::Error::uncertain(format!(
                "selection unreadable ({}): {e}",
                path.display()
            )));
        }
    };
    let selection: Selection = serde_json::from_slice(&bytes).map_err(|e| {
        crate::Error::uncertain(format!(
            "selection corrupt ({}): {e}; refusing to delete possibly retained bundles",
            path.display()
        ))
    })?;
    validate_selection(&selection, selection.provider).map_err(|e| {
        crate::Error::uncertain(format!(
            "selection invalid ({}): {e}; refusing to delete possibly retained bundles",
            path.display()
        ))
    })?;
    Ok(Some(selection))
}

/// Remove unreferenced non-current bundles; returns bytes reclaimed.
/// Current, previous and actively referenced bundles are always retained.
/// The whole pass holds the exclusive store lock, and every piece of
/// protection knowledge (current/previous selections, all refs) is validated
/// BEFORE any deletion: corrupt or unreadable knowledge fails `Uncertain`
/// with nothing deleted. Leftover `.staging-*` paths older than one hour are
/// reclaimed too; fresher ones may belong to a concurrent refresh and are
/// left alone.
pub fn cleanup(root: &Path) -> Result<u64, crate::Error> {
    use std::collections::HashSet;
    let _lock = lock_store(root)?;
    let mut protected: HashSet<(String, String)> = HashSet::new();
    for provider in [Provider::Codex, Provider::Muse] {
        for path in [
            selection_path(root, provider),
            previous_path(root, provider),
        ] {
            if let Some(selection) = read_selection_file(&path)? {
                protected.insert((provider.as_str().to_string(), selection.release.digest));
            }
        }
    }
    for record in list_ref_records(root)? {
        protected.insert((record.provider.as_str().to_string(), record.digest));
    }

    let mut reclaimed: u64 = 0;
    let mut failures: Vec<String> = Vec::new();
    let bundles = root.join("bundles");
    let providers = match std::fs::read_dir(&bundles) {
        Ok(entries) => entries,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(0),
        Err(e) => return Err(crate::Error::from(e)),
    };
    for provider_entry in providers {
        let provider_entry = match provider_entry {
            Ok(entry) => entry,
            Err(e) => {
                failures.push(e.to_string());
                continue;
            }
        };
        let provider_name = provider_entry.file_name().to_string_lossy().into_owned();
        if provider_entry
            .file_type()
            .map(|t| !t.is_dir())
            .unwrap_or(true)
        {
            continue;
        }
        let dirs = match std::fs::read_dir(provider_entry.path()) {
            Ok(dirs) => dirs,
            Err(e) => {
                failures.push(e.to_string());
                continue;
            }
        };
        for dir in dirs {
            let dir = match dir {
                Ok(dir) => dir,
                Err(e) => {
                    failures.push(e.to_string());
                    continue;
                }
            };
            if !dir.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                continue;
            }
            let digest = dir.file_name().to_string_lossy().into_owned();
            if protected.contains(&(provider_name.clone(), digest.clone())) {
                continue;
            }
            let looks_like_digest =
                digest.len() == 64 && digest.chars().all(|c| c.is_ascii_hexdigit());
            if !looks_like_digest {
                continue;
            }
            reclaimed += dir_size(&dir.path());
            if let Err(e) = std::fs::remove_dir_all(dir.path()) {
                failures.push(format!("{}: {e}", dir.path().display()));
            }
        }
    }
    // Reclaim stale staging leftovers from interrupted refreshes.
    if let Ok(entries) = std::fs::read_dir(root) {
        for entry in entries.flatten() {
            let name = entry
                .path()
                .file_name()
                .and_then(|s| s.to_str())
                .unwrap_or_default()
                .to_string();
            if !(name.starts_with(".staging-") || name.ends_with(".staging")) {
                continue;
            }
            let old = entry
                .metadata()
                .and_then(|m| m.modified())
                .map(|t| {
                    std::time::SystemTime::now()
                        .duration_since(t)
                        .map(|d| d.as_secs() > 3600)
                        .unwrap_or(true)
                })
                .unwrap_or(true);
            if !old {
                continue;
            }
            let path = entry.path();
            if path.is_dir() {
                reclaimed += dir_size(&path);
                if let Err(e) = std::fs::remove_dir_all(&path) {
                    failures.push(format!("{}: {e}", path.display()));
                }
            } else {
                reclaimed += entry.metadata().map(|m| m.len()).unwrap_or(0);
                if let Err(e) = std::fs::remove_file(&path) {
                    failures.push(format!("{}: {e}", path.display()));
                }
            }
        }
    }
    if !failures.is_empty() {
        return Err(crate::Error::uncertain(format!(
            "cleanup incomplete: {}",
            failures.join("; ")
        )));
    }
    Ok(reclaimed)
}

fn dir_size(dir: &Path) -> u64 {
    let mut total = 0u64;
    let mut stack = vec![dir.to_path_buf()];
    while let Some(current) = stack.pop() {
        let Ok(entries) = std::fs::read_dir(&current) else {
            continue;
        };
        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_symlink() {
                total += entry.metadata().map(|m| m.len()).unwrap_or(0);
                continue;
            }
            if path.is_dir() {
                stack.push(path);
            } else {
                total += entry.metadata().map(|m| m.len()).unwrap_or(0);
            }
        }
    }
    total
}

/// One admitted file: `/`-separated path relative to the bundle dir, exact
/// size, sha256 of the admitted bytes, and whether it must stay executable.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InventoryEntry {
    pub path: String,
    pub size: u64,
    pub sha256: String,
    pub executable: bool,
}

/// Per-file digest inventory of one admitted bundle (`inventory.json`).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Inventory {
    pub provider: Provider,
    pub digest: String,
    pub entries: Vec<InventoryEntry>,
}

/// Write the admission record (`release.json`) into a bundle directory.
/// The digest must match the directory name; the identity itself is validated.
pub fn write_release_record(
    bundle_dir: &Path,
    release: &ReleaseIdentity,
) -> Result<(), crate::Error> {
    validate_release(release, release.provider)?;
    let digest = crate::normalize_digest(&release.digest)?;
    let name = bundle_dir
        .file_name()
        .and_then(|s| s.to_str())
        .unwrap_or_default()
        .to_lowercase();
    if name != digest {
        return Err(crate::Error::denied(format!(
            "release digest does not match bundle directory {}",
            bundle_dir.display()
        )));
    }
    std::fs::create_dir_all(bundle_dir)?;
    let bytes = serde_json::to_vec_pretty(release)?;
    std::fs::write(bundle_dir.join("release.json"), bytes)?;
    Ok(())
}

/// Scan of one bundle tree: every symlink is refused (never follow symlink
/// authority into writable locations), names must be UTF-8 and relative.
struct TreeEntry {
    rel: String,
    is_dir: bool,
    size: u64,
    sha256: Option<String>,
    executable: bool,
}

fn scan_tree(bundle_dir: &Path) -> Result<Vec<TreeEntry>, crate::Error> {
    let mut out = Vec::new();
    let mut stack = vec![bundle_dir.to_path_buf()];
    while let Some(current) = stack.pop() {
        let entries = std::fs::read_dir(&current).map_err(|e| {
            crate::Error::denied(format!("bundle unreadable ({}): {e}", current.display()))
        })?;
        for entry in entries {
            let entry = entry.map_err(|e| {
                crate::Error::denied(format!("bundle listing unreadable ({}): {e}", current.display()))
            })?;
            let path = entry.path();
            if path.is_symlink() {
                return Err(crate::Error::denied(format!(
                    "bundle contains symlink, refusing: {}",
                    path.display()
                )));
            }
            let rel = path
                .strip_prefix(bundle_dir)
                .map_err(|_| {
                    crate::Error::denied(format!(
                        "bundle entry escapes directory: {}",
                        path.display()
                    ))
                })?
                .to_str()
                .ok_or_else(|| {
                    crate::Error::denied(format!(
                        "bundle entry is not UTF-8: {}",
                        path.display()
                    ))
                })?
                .replace('\\', "/");
            if rel.is_empty()
                || rel.starts_with('/')
                || rel.split('/').any(|c| c.is_empty() || c == "." || c == "..")
            {
                return Err(crate::Error::denied(format!(
                    "bundle entry has unsafe name: {rel}"
                )));
            }
            if path.is_dir() {
                out.push(TreeEntry {
                    rel,
                    is_dir: true,
                    size: 0,
                    sha256: None,
                    executable: false,
                });
                stack.push(path);
            } else if path.is_file() {
                #[cfg(unix)]
                let executable = {
                    use std::os::unix::fs::PermissionsExt as _;
                    std::fs::metadata(&path)
                        .map(|m| m.permissions().mode() & 0o111 != 0)
                        .unwrap_or(false)
                };
                #[cfg(not(unix))]
                let executable = false;
                let (sha256, size) = crate::sha256_hex_file(&path)?;
                out.push(TreeEntry {
                    rel,
                    is_dir: false,
                    size,
                    sha256: Some(sha256),
                    executable,
                });
            } else {
                return Err(crate::Error::denied(format!(
                    "bundle entry is not a file or dir: {}",
                    path.display()
                )));
            }
        }
    }
    out.sort_by(|a, b| a.rel.cmp(&b.rel));
    Ok(out)
}

/// (Re)build `inventory.json` for an admitted bundle directory. Refuses
/// symlink-containing trees. `release.json` must already be written: it is
/// covered by the inventory like every other file; only `inventory.json`
/// itself is excluded.
pub fn record_inventory(
    bundle_dir: &Path,
    release: &ReleaseIdentity,
) -> Result<Inventory, crate::Error> {
    let digest = crate::normalize_digest(&release.digest)?;
    let tree = scan_tree(bundle_dir)?;
    if !tree.iter().any(|e| !e.is_dir && e.rel == "release.json") {
        return Err(crate::Error::denied(format!(
            "cannot inventory bundle without admission record: {}",
            bundle_dir.display()
        )));
    }
    let entries: Vec<InventoryEntry> = tree
        .iter()
        .filter(|e| !e.is_dir && e.rel != "inventory.json")
        .map(|e| InventoryEntry {
            path: e.rel.clone(),
            size: e.size,
            sha256: e.sha256.clone().unwrap_or_default(),
            executable: e.executable,
        })
        .collect();
    let inventory = Inventory {
        provider: release.provider,
        digest,
        entries,
    };
    let bytes = serde_json::to_vec_pretty(&inventory)?;
    std::fs::write(bundle_dir.join("inventory.json"), bytes)?;
    Ok(inventory)
}

/// Verify an admitted bundle before use: the digest names a real directory
/// (never a symlink), `release.json` parses and matches provider+digest,
/// `inventory.json` is present and matches provider+digest, every inventoried
/// file still matches its digest/size/exec bits with exact coverage (no extra
/// files, none missing), the required executable exists as a real executable
/// file, and (Codex) the companion exists too. A bare hash-shaped directory
/// with a fabricated executable and no admission record is `Denied`, and so
/// is any bundle whose admission-time digest inventory is missing, unreadable
/// or corrupt: there is no legacy bypass. A missing inventory never disables
/// content verification; pre-inventory caches are re-admitted from verified
/// upstream bytes, never re-hashed from arbitrary current contents.
pub fn verify_bundle(
    root: &Path,
    provider: Provider,
    digest: &str,
) -> Result<PathBuf, crate::Error> {
    let digest = crate::normalize_digest(digest)?;
    let bundle = root
        .join("bundles")
        .join(provider.as_str())
        .join(&digest);
    let meta = std::fs::symlink_metadata(&bundle).map_err(|_| {
        crate::Error::denied(format!(
            "no admitted {} bundle for digest {digest}; refusing to substitute another build",
            provider.as_str()
        ))
    })?;
    if meta.file_type().is_symlink() || !meta.is_dir() {
        return Err(crate::Error::denied(format!(
            "admitted {} bundle is not a real directory: {}",
            provider.as_str(),
            bundle.display()
        )));
    }
    let record_bytes = std::fs::read(bundle.join("release.json")).map_err(|_| {
        crate::Error::denied(format!(
            "no admission record for {} bundle {digest}; refusing hash-shaped directory",
            provider.as_str()
        ))
    })?;
    let record: ReleaseIdentity = serde_json::from_slice(&record_bytes).map_err(|e| {
        crate::Error::denied(format!(
            "admission record corrupt for {} bundle {digest}: {e}",
            provider.as_str()
        ))
    })?;
    validate_release(&record, provider).map_err(|e| {
        crate::Error::denied(format!(
            "admission record invalid for {} bundle {digest}: {e}",
            provider.as_str()
        ))
    })?;
    if crate::normalize_digest(&record.digest)? != digest {
        return Err(crate::Error::denied(format!(
            "admission record digest mismatch for {} bundle {digest}",
            provider.as_str()
        )));
    }
    let inventory_path = bundle.join("inventory.json");
    // No legacy bypass: without the admission-time digest inventory there is
    // nothing trustworthy to compare the live tree against, so verification
    // fails closed. Callers re-admit from verified upstream bytes instead of
    // re-hashing arbitrary current contents.
    if !inventory_path.exists() {
        return Err(crate::Error::denied(format!(
            "bundle without digest inventory for {} {digest}; refusing to launch unverified bytes",
            provider.as_str()
        )));
    }
    {
        let inv_bytes = std::fs::read(&inventory_path).map_err(|e| {
            crate::Error::denied(format!(
                "bundle inventory unreadable for {} {digest}: {e}",
                provider.as_str()
            ))
        })?;
        let inventory: Inventory = serde_json::from_slice(&inv_bytes).map_err(|e| {
            crate::Error::denied(format!(
                "bundle inventory corrupt for {} {digest}: {e}",
                provider.as_str()
            ))
        })?;
        if inventory.provider != provider || crate::normalize_digest(&inventory.digest)? != digest
        {
            return Err(crate::Error::denied(format!(
                "bundle inventory identity mismatch for {} {digest}",
                provider.as_str()
            )));
        }
        let tree = scan_tree(&bundle)?;
        let live: Vec<&TreeEntry> = tree
            .iter()
            .filter(|e| !e.is_dir && e.rel != "inventory.json")
            .collect();
        if live.len() != inventory.entries.len() {
            return Err(crate::Error::denied(format!(
                "bundle inventory coverage mismatch for {} {digest}: {} files on disk, {} inventoried",
                provider.as_str(),
                live.len(),
                inventory.entries.len()
            )));
        }
        for entry in &inventory.entries {
            let found = live.iter().find(|e| e.rel == entry.path).ok_or_else(|| {
                crate::Error::denied(format!(
                    "bundle file missing for {} {digest}: {}",
                    provider.as_str(),
                    entry.path
                ))
            })?;
            if found.size != entry.size || found.sha256.as_deref() != Some(entry.sha256.as_str())
            {
                return Err(crate::Error::denied(format!(
                    "bundle file changed for {} {digest}: {} (digest/size mismatch)",
                    provider.as_str(),
                    entry.path
                )));
            }
            if entry.executable && !found.executable {
                return Err(crate::Error::denied(format!(
                    "bundle file lost executable bit for {} {digest}: {}",
                    provider.as_str(),
                    entry.path
                )));
            }
        }
    }
    let exe = bundle.join(provider.executable_relpath());
    require_executable_file(&exe, provider, "executable")?;
    if provider == Provider::Codex {
        let companion = bundle.join("bin").join("codex-code-mode-host");
        require_executable_file(&companion, provider, "companion")?;
    }
    Ok(exe)
}

fn require_executable_file(
    path: &Path,
    provider: Provider,
    what: &str,
) -> Result<(), crate::Error> {
    let meta = std::fs::symlink_metadata(path).map_err(|_| {
        crate::Error::denied(format!(
            "{} bundle missing required {what}: {}",
            provider.as_str(),
            path.display()
        ))
    })?;
    if meta.file_type().is_symlink() || !meta.is_file() {
        return Err(crate::Error::denied(format!(
            "{} bundle {what} is not a real file: {}",
            provider.as_str(),
            path.display()
        )));
    }
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt as _;
        if meta.permissions().mode() & 0o111 == 0 {
            return Err(crate::Error::denied(format!(
                "{} bundle {what} is not executable: {}",
                provider.as_str(),
                path.display()
            )));
        }
    }
    Ok(())
}

/// Verify the exact bundle a selection points at, including that the on-disk
/// admission record carries the selected version and size.
pub fn verify_selected(root: &Path, selection: &Selection) -> Result<PathBuf, crate::Error> {
    let exe = verify_bundle(root, selection.provider, &selection.release.digest)?;
    let record_bytes = std::fs::read(
        root.join("bundles")
            .join(selection.provider.as_str())
            .join(&selection.release.digest)
            .join("release.json"),
    )
    .map_err(|e| {
        crate::Error::denied(format!("selected bundle admission record unreadable: {e}"))
    })?;
    let record: ReleaseIdentity = serde_json::from_slice(&record_bytes).map_err(|e| {
        crate::Error::denied(format!("selected bundle admission record corrupt: {e}"))
    })?;
    if record.version != selection.release.version || record.size != selection.release.size {
        return Err(crate::Error::denied(format!(
            "selected {} {} does not match admitted record ({} {})",
            selection.provider.as_str(),
            selection.release.version,
            record.version,
            record.size
        )));
    }
    Ok(exe)
}
