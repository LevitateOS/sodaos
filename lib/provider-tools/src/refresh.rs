//! Bundle admission: download, verify, unpack, atomically select (lane A).
//!
//! First refresh installs; repeated refresh reuses unchanged bytes without a
//! redundant download. An interrupted or failed update never exposes a partial
//! selection: the previous selection stays current until the new bundle is
//! fully verified and the selection record is atomically replaced. Timestamps
//! are refreshed only on success.

use crate::store::{Provider, Selection};

/// Block new work on a known-newer release that could not be admitted, after
/// persisting the discovery so no other process launches the old selection
/// during TTL. Selected bytes and admission time are preserved; only the
/// successful check time advances and freshness becomes `Stale`. Same-release
/// (repair) and first-install failures return the original error untouched.
fn stale_block(
    root: &std::path::Path,
    previous: Option<&Selection>,
    latest: &crate::store::ReleaseIdentity,
    now: u64,
    detail: String,
) -> crate::Error {
    let Some(prev) = previous else {
        return crate::Error::blocked(detail);
    };
    if prev.release.digest == latest.digest {
        return crate::Error::failed(detail);
    }
    let stale = Selection {
        provider: prev.provider,
        release: prev.release.clone(),
        checked_at_secs: now,
        admitted_at_secs: prev.admitted_at_secs,
        freshness: crate::store::Freshness::Stale {
            latest: latest.version.clone(),
        },
    };
    let persisted = (|| -> Result<(), crate::Error> {
        let _lock = crate::store::lock_store(root)?;
        match crate::store::read_selection(root, prev.provider)? {
            Some(current) if current.release.digest != prev.release.digest => Ok(()),
            _ => crate::store::write_selection(root, &stale),
        }
    })();
    match persisted {
        Ok(()) => crate::Error::blocked(format!(
            "known newer {} {} could not be admitted ({detail}); previous {} retained and recorded stale",
            latest.provider.as_str(),
            latest.version,
            prev.release.version
        )),
        Err(e) => crate::Error::uncertain(format!(
            "known newer {} {} could not be admitted ({detail}) and stale state could not be recorded ({e}); retry before launching",
            latest.provider.as_str(),
            latest.version
        )),
    }
}

/// Refresh a provider: check upstream, admit the selected bundle when it is
/// not already stored, and return the resulting selection. A failed check
/// never refreshes timestamps and never downgrades stored knowledge: a stored
/// `Stale` (or `Unknown`) selection is returned unchanged so new work keeps
/// blocking; only a stored `Current`/`UpdateFailed` becomes `UpdateFailed`
/// in memory. When a newer release is known but cannot be admitted, the
/// discovery is persisted as `Stale` and the error blocks new work.
pub fn refresh(provider: Provider) -> Result<Selection, crate::Error> {
    let root = crate::config::store_root();
    std::fs::create_dir_all(&root)?;
    let now = crate::now_secs();
    let previous = crate::store::read_selection(&root, provider)?;

    let latest = match crate::discover::resolve_upstream(provider) {
        Ok(latest) => latest,
        Err(check_err) => match previous {
            Some(sel) => match &sel.freshness {
                crate::store::Freshness::Stale { .. }
                | crate::store::Freshness::Unknown { .. } => {
                    return Ok(sel);
                }
                crate::store::Freshness::Current
                | crate::store::Freshness::UpdateFailed { .. } => {
                    return Ok(Selection {
                        freshness: crate::store::Freshness::UpdateFailed {
                            reason: format!(
                                "upstream check failed ({check_err}); retained {} {}",
                                sel.provider.as_str(),
                                sel.release.version
                            ),
                        },
                        ..sel
                    });
                }
            },
            None => {
                return Err(crate::Error::failed(format!(
                    "upstream check failed and no {} bundle is admitted: {check_err}",
                    provider.as_str()
                )));
            }
        },
    };

    if let Some(sel) = &previous {
        if sel.release.digest == latest.digest {
            if bundle_complete(&root, provider, &sel.release.digest) {
                // Unchanged refresh: no download, only the successful check
                // timestamp advances. Published under the store lock.
                let updated = Selection {
                    checked_at_secs: now,
                    freshness: crate::store::Freshness::Current,
                    ..sel.clone()
                };
                let _lock = crate::store::lock_store(&root)?;
                crate::store::write_selection(&root, &updated)?;
                return Ok(updated);
            }
            // Selection matches upstream but payload is damaged: repair by
            // re-admitting the same digest. No newer release is known, so a
            // repair failure is operational, not blocking.
            admit(&root, provider, &latest)?;
            let repaired = Selection {
                provider,
                release: latest,
                checked_at_secs: now,
                admitted_at_secs: now,
                freshness: crate::store::Freshness::Current,
            };
            let _lock = crate::store::lock_store(&root)?;
            crate::store::write_selection(&root, &repaired)?;
            return Ok(repaired);
        }
        if bundle_complete(&root, provider, &latest.digest) {
            // Payload already on disk (e.g. downloaded before an
            // interruption): verify identity, probe, and swap without a
            // redundant download.
            match crate::discover::probe_executable(
                provider,
                &repaired_exe(&root, provider, &latest.digest),
            ) {
                Ok(()) => {
                    let admitted = Selection {
                        provider,
                        release: latest,
                        checked_at_secs: now,
                        admitted_at_secs: now,
                        freshness: crate::store::Freshness::Current,
                    };
                    publish(&root, previous.as_ref(), &admitted)?;
                    return Ok(admitted);
                }
                Err(probe_err) => {
                    return Err(stale_block(
                        &root,
                        previous.as_ref(),
                        &latest,
                        now,
                        format!(
                            "known newer {} {} is on disk but fails verification ({probe_err})",
                            provider.as_str(),
                            latest.version
                        ),
                    ));
                }
            }
        }
    }

    // Admit the new digest, then atomically select it. Any admission failure
    // against a known-newer release persists the discovery as `Stale` first.
    let created = admit(&root, provider, &latest).map_err(|e| {
        stale_block(&root, previous.as_ref(), &latest, now, e.to_string())
    })?;
    let exe = latest_exe(&root, provider, &latest);
    if let Err(probe_err) = crate::discover::probe_executable(provider, &exe) {
        if created {
            let _ = std::fs::remove_dir_all(bundle_dir(&root, provider, &latest.digest));
        }
        return Err(stale_block(
            &root,
            previous.as_ref(),
            &latest,
            now,
            format!(
                "known newer {} {} admitted but failed verification ({probe_err})",
                provider.as_str(),
                latest.version
            ),
        ));
    }
    let admitted = Selection {
        provider,
        release: latest.clone(),
        checked_at_secs: now,
        admitted_at_secs: now,
        freshness: crate::store::Freshness::Current,
    };
    publish(&root, previous.as_ref(), &admitted)?;
    Ok(admitted)
}

/// Publish a new selection under the store lock: retain the replaced record
/// as `previous`, then atomically replace `current`. A concurrent refresh
/// that already admitted the same digest converges instead of clobbering.
fn publish(
    root: &std::path::Path,
    previous: Option<&Selection>,
    admitted: &Selection,
) -> Result<(), crate::Error> {
    let _lock = crate::store::lock_store(root)?;
    if let Some(sel) = previous {
        if sel.release.digest == admitted.release.digest {
            return Ok(());
        }
    }
    if let Ok(Some(current)) = crate::store::read_selection(root, admitted.provider) {
        if current.release.digest == admitted.release.digest
            && crate::store::verify_bundle(root, admitted.provider, &admitted.release.digest)
                .is_ok()
        {
            return Ok(());
        }
    }
    if let Some(sel) = previous {
        crate::store::write_previous(root, sel)?;
    }
    crate::store::write_selection(root, admitted)?;
    Ok(())
}

fn bundle_dir(root: &std::path::Path, provider: Provider, digest: &str) -> std::path::PathBuf {
    root.join("bundles").join(provider.as_str()).join(digest)
}

fn latest_exe(
    root: &std::path::Path,
    provider: Provider,
    latest: &crate::store::ReleaseIdentity,
) -> std::path::PathBuf {
    bundle_dir(root, provider, &latest.digest).join(provider.executable_relpath())
}

fn repaired_exe(root: &std::path::Path, provider: Provider, digest: &str) -> std::path::PathBuf {
    bundle_dir(root, provider, digest).join(provider.executable_relpath())
}

fn bundle_complete(root: &std::path::Path, provider: Provider, digest: &str) -> bool {
    crate::store::verify_bundle(root, provider, digest).is_ok()
}

/// Download an artifact to a bounded staging file and verify its sha256 and
/// exact size against the release identity. No symlinks or absolute paths may
/// escape the bundle directory on unpack; native arch is enforced.
pub fn fetch_verified(
    release: &crate::store::ReleaseIdentity,
    staging: &std::path::Path,
) -> Result<(), crate::Error> {
    if !(release.artifact_url.starts_with("https://")) {
        return Err(crate::Error::denied(format!(
            "refusing non-https artifact URL: {}",
            release.artifact_url
        )));
    }
    let digest = crate::normalize_digest(&release.digest)?;
    if release.size == 0 {
        return Err(crate::Error::denied("refusing zero-size artifact"));
    }
    if let Some(parent) = staging.parent() {
        std::fs::create_dir_all(parent)?;
    }
    let _ = std::fs::remove_file(staging);
    let written = crate::http_download_bounded(&release.artifact_url, staging, release.size)
        .map_err(|e| {
            let _ = std::fs::remove_file(staging);
            e
        })?;
    if written != release.size {
        let _ = std::fs::remove_file(staging);
        return Err(crate::Error::failed(format!(
            "artifact size {written} != published {}",
            release.size
        )));
    }
    let (actual, _) = crate::sha256_hex_file(staging)?;
    if actual != digest {
        let _ = std::fs::remove_file(staging);
        return Err(crate::Error::failed(format!(
            "artifact sha256 mismatch for {} {}: got {actual}",
            release.provider.as_str(),
            release.version
        )));
    }
    Ok(())
}

/// Fetch, verify, unpack and publish the bundle for `release` under
/// `bundles/<provider>/<digest>/`. Returns true when this call created the
/// directory (callers only remove what they created).
fn admit(
    root: &std::path::Path,
    provider: Provider,
    release: &crate::store::ReleaseIdentity,
) -> Result<bool, crate::Error> {
    let dest = bundle_dir(root, provider, &release.digest);
    if bundle_complete(root, provider, &release.digest) {
        return Ok(false);
    }
    let pid = std::process::id();
    let staging_file = root.join(format!(".staging-{}-{pid}.artifact", provider.as_str()));
    let staging_dir = root.join(format!(".staging-{}-{pid}.d", provider.as_str()));
    let _ = std::fs::remove_file(&staging_file);
    let _ = std::fs::remove_dir_all(&staging_dir);
    let result = admit_inner(root, provider, release, &staging_file, &staging_dir, &dest);
    let _ = std::fs::remove_file(&staging_file);
    if result.is_err() {
        let _ = std::fs::remove_dir_all(&staging_dir);
    }
    result
}

fn admit_inner(
    root: &std::path::Path,
    provider: Provider,
    release: &crate::store::ReleaseIdentity,
    staging_file: &std::path::Path,
    staging_dir: &std::path::Path,
    dest: &std::path::Path,
) -> Result<bool, crate::Error> {
    // Raw errors propagate: the caller maps known-newer admission failures
    // through stale_block (persist discovery, block new work). Same-release
    // repair failures stay operational.
    fetch_verified(release, staging_file)?;
    std::fs::create_dir_all(staging_dir)?;
    install_staging(provider, staging_file, staging_dir)?;
    normalize_prefix(provider, staging_dir)?;
    crate::store::write_release_record(staging_dir, release)?;
    chmod_exec(provider, staging_dir)?;
    crate::store::record_inventory(staging_dir, release)?;
    if let Some(parent) = dest.parent() {
        std::fs::create_dir_all(parent)?;
    }
    // Publish atomically: rename wins the race; a concurrently admitted
    // identical digest is reused instead of overwritten. Any rename failure
    // re-checks the destination so a lost race still converges.
    match std::fs::rename(staging_dir, dest) {
        Ok(()) => {
            let _ = crate::fsync_dir(dest);
            if let Some(parent) = dest.parent() {
                let _ = crate::fsync_dir(parent);
            }
            Ok(true)
        }
        Err(rename_err) => {
            if bundle_complete(root, provider, &release.digest) {
                let _ = std::fs::remove_dir_all(staging_dir);
                return Ok(false);
            }
            let _ = std::fs::remove_dir_all(staging_dir);
            Err(crate::Error::failed(format!(
                "bundle publish failed ({}); concurrent bundle incomplete: {rename_err}",
                dest.display()
            )))
        }
    }
}

#[derive(Debug, PartialEq, Eq)]
enum ArtifactKind {
    ZstdTar,
    GzipTar,
    RawExecutable,
}

fn detect_kind(staging_file: &std::path::Path) -> Result<ArtifactKind, crate::Error> {
    use std::io::Read as _;
    let mut magic = [0u8; 8];
    let mut file = std::fs::File::open(staging_file)?;
    file.read_exact(&mut magic)?;
    if magic.starts_with(&[0x28, 0xB5, 0x2F, 0xFD]) {
        Ok(ArtifactKind::ZstdTar)
    } else if magic.starts_with(&[0x1F, 0x8B]) {
        Ok(ArtifactKind::GzipTar)
    } else if magic.starts_with(&[0x7F, b'E', b'L', b'F']) {
        Ok(ArtifactKind::RawExecutable)
    } else {
        Err(crate::Error::failed(format!(
            "unknown artifact format (magic {:02X?}): {}",
            &magic[..4],
            staging_file.display()
        )))
    }
}

fn install_staging(
    provider: Provider,
    staging_file: &std::path::Path,
    staging_dir: &std::path::Path,
) -> Result<(), crate::Error> {
    match detect_kind(staging_file)? {
        ArtifactKind::RawExecutable => {
            // Muse ships a raw native executable, not an archive.
            let bin = staging_dir.join("bin");
            std::fs::create_dir_all(&bin)?;
            let exe_name = provider
                .executable_relpath()
                .rsplit('/')
                .next()
                .expect("relpath has file name");
            std::fs::copy(staging_file, bin.join(exe_name))?;
            Ok(())
        }
        ArtifactKind::GzipTar => {
            let file = std::fs::File::open(staging_file)?;
            let decoder = flate2::read::GzDecoder::new(file);
            unpack_tar_safely(decoder, staging_dir)
        }
        ArtifactKind::ZstdTar => {
            // System zstd only; no new decompression crates.
            let mut child = std::process::Command::new("/usr/bin/zstd")
                .args(["-d", "-c", "--"])
                .arg(staging_file)
                .stdin(std::process::Stdio::null())
                .stdout(std::process::Stdio::piped())
                .stderr(std::process::Stdio::null())
                .spawn()
                .map_err(|e| {
                    crate::Error::failed(format!(
                        "system zstd is required but would not start: {e}"
                    ))
                })?;
            let stdout = child
                .stdout
                .take()
                .ok_or_else(|| crate::Error::failed("system zstd produced no output stream"))?;
            let result = unpack_tar_safely(stdout, staging_dir);
            let status = child.wait().map_err(crate::Error::from)?;
            if !status.success() {
                return Err(crate::Error::failed(format!(
                    "system zstd decompression failed with status {status}"
                )));
            }
            result
        }
    }
}

/// Unpack a tar stream with archive safety: no absolute paths, no `..`
/// escapes, no symlinks or hardlinks, native payload only. Bounds total
/// unpacked bytes and entry count.
fn unpack_tar_safely(
    reader: impl std::io::Read,
    dest: &std::path::Path,
) -> Result<(), crate::Error> {
    let mut archive = tar::Archive::new(reader);
    let mut total: u64 = 0;
    let mut entries_count: u64 = 0;
    for entry in archive.entries().map_err(crate::Error::from)? {
        let mut entry = entry.map_err(crate::Error::from)?;
        entries_count += 1;
        if entries_count > 200_000 {
            return Err(crate::Error::failed("tar entry count exceeds safety bound"));
        }
        let kind = entry.header().entry_type();
        if kind.is_symlink() || kind.is_hard_link() {
            return Err(crate::Error::failed(format!(
                "tar entry is a link, refusing: {}",
                entry
                    .path()
                    .map(|p| p.display().to_string())
                    .unwrap_or_default()
            )));
        }
        if kind != tar::EntryType::Directory && kind != tar::EntryType::Regular {
            return Err(crate::Error::failed(format!(
                "tar entry has unsafe type {kind:?}, refusing"
            )));
        }
        let path = entry.path().map_err(crate::Error::from)?.into_owned();
        if path.is_absolute()
            || path.components().any(|c| {
                matches!(
                    c,
                    std::path::Component::ParentDir | std::path::Component::Prefix(_)
                )
            })
        {
            return Err(crate::Error::failed(format!(
                "tar entry escapes bundle directory, refusing: {}",
                path.display()
            )));
        }
        total += entry.header().size().unwrap_or(0);
        if total > 4 * 1024 * 1024 * 1024 {
            return Err(crate::Error::failed(
                "tar payload exceeds 4GiB safety bound",
            ));
        }
        entry.unpack_in(dest).map_err(crate::Error::from)?;
    }
    Ok(())
}

/// Archives sometimes nest payload under one top-level dir; hoist it so
/// `bin/<exe>` sits directly under the bundle dir.
fn normalize_prefix(provider: Provider, dir: &std::path::Path) -> Result<(), crate::Error> {
    if dir.join(provider.executable_relpath()).exists() {
        return Ok(());
    }
    let mut subdirs = Vec::new();
    for entry in std::fs::read_dir(dir)? {
        let entry = entry?;
        if entry.file_type()?.is_dir() {
            subdirs.push(entry.path());
        }
    }
    if subdirs.len() == 1 && subdirs[0].join(provider.executable_relpath()).exists() {
        let top = subdirs.remove(0);
        for entry in std::fs::read_dir(&top)? {
            let entry = entry?;
            let dest = dir.join(entry.file_name());
            if dest.exists() {
                return Err(crate::Error::failed(format!(
                    "cannot hoist archive prefix, collision: {}",
                    dest.display()
                )));
            }
            std::fs::rename(entry.path(), &dest)?;
        }
        std::fs::remove_dir(&top)?;
    }
    if !dir.join(provider.executable_relpath()).exists() {
        return Err(crate::Error::failed(format!(
            "admitted payload lacks required executable {}",
            provider.executable_relpath()
        )));
    }
    Ok(())
}

fn chmod_exec(provider: Provider, dir: &std::path::Path) -> Result<(), crate::Error> {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt as _;
        let exe = dir.join(provider.executable_relpath());
        let mut perms = std::fs::metadata(&exe)?.permissions();
        perms.set_mode(0o755);
        std::fs::set_permissions(&exe, perms)?;
        if provider == Provider::Codex {
            let companion = dir.join("bin").join("codex-code-mode-host");
            if companion.exists() {
                let mut perms = std::fs::metadata(&companion)?.permissions();
                perms.set_mode(0o755);
                std::fs::set_permissions(&companion, perms)?;
            }
        }
    }
    let _ = provider;
    Ok(())
}
