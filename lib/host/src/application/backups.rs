use super::runtime::Runtime;
use super::snapshot::{self, SnapshotInfo};
use super::wire::{hex, id, Spec};
use crate::project::Executor;
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use std::time::{Instant, SystemTime, UNIX_EPOCH};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Request {
    pub application: Spec,
    pub action: String,
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub expected_container: String,
}

#[derive(Serialize)]
pub struct Inventory {
    pub snapshots: Vec<SnapshotInfo>,
    pub complete: bool,
}

pub(super) fn root() -> Result<PathBuf, String> {
    root_anchored().map(|(path, _)| path)
}

/// Anchor-validated variant: the storage-root descriptor stays open for the
/// caller so every later step can resolve through it instead of bare paths.
pub(super) fn root_anchored() -> Result<(PathBuf, std::os::fd::OwnedFd), String> {
    // Same trusted seam as setup and runtime data: one shared storage root,
    // so the fixture redirect can never split snapshots off to another tree.
    // FJ-847: the root is opened into a validated descriptor and the
    // snapshots child is validated through it; the path hand-off below is
    // rebound to the anchor by dev/ino before the snapshot module reads it.
    let root = super::storage_root()?;
    let anchor = super::home::open_validated_dir(
        &root,
        0,
        0o022,
        "application storage root must belong to the operator",
    )?;
    let path = root.join("snapshots");
    match super::home::open_child_dir_opt(
        &anchor,
        "snapshots",
        "application snapshots are unavailable",
    )? {
        Some(child) => {
            super::home::validate_open_dir(
                &child,
                0,
                0o077,
                "application snapshots must be private operator-owned directories",
            )?;
            super::home::fd_matches_path(&path, &child)
                .map_err(|_| "application snapshots are unavailable".to_string())?;
        }
        None => {}
    }
    Ok((path, anchor))
}

/// FJ-855 honest snapshot scope (FJ-854 follow-up): this file does NOT claim
/// every privileged snapshot mutation is descriptor-relative end to end.
/// The snapshot module (`snapshot.rs`, frozen — not owned by this lane)
/// resolves its own interior paths (`File::open`, `OpenOptions::open`,
/// `read_dir`, `copy`) from the paths handed to it. What this seam owns is
/// the handoff bound: every leaf handed to the snapshot module is opened
/// through the storage-root anchor and rebound by dev/ino immediately
/// before the call, and re-opened and rebound immediately after, so a leaf
/// swapped across the call fails closed (refusal, or the leaf skipped from
/// inventory) instead of acting on foreign bytes. The bound holds because
/// the interior opens are confined to the id-bound leaf itself inside the
/// operator-owned mode-0700 snapshots tree, in the same host process (no
/// privilege boundary crossed), with unsealed/unreadable snapshots never
/// selectable. Bringing the snapshot interior fully descriptor-relative is
/// explicitly out of scope for this lane.
impl<E: Executor> Runtime<E> {
    pub fn snapshots(&self, request: &Request, deadline: Instant) -> Result<Inventory, String> {
        request.application.validate()?;
        match request.action.as_str() {
            "list" if request.id.is_empty() && request.expected_container.is_empty() => (),
            "inspect" if id(&request.id) && request.expected_container.is_empty() => (),
            "backup" if id(&request.id) && hex(&request.expected_container, 64) => (),
            _ => return Err("invalid application snapshot operation".into()),
        }
        let (root, anchor) = root_anchored()?;
        if request.action == "inspect" {
            // FJ-847: the snapshot leaf is opened through the storage-root
            // anchor and rebound by dev/ino before the snapshot module reads
            // it — a swapped leaf fails closed here.
            let child = super::home::open_child_dir_opt(
                &anchor,
                "snapshots",
                "application snapshots are unavailable",
            )?
            .ok_or("application snapshots are unavailable")?;
            let leaf = super::home::open_child_dir_opt(
                &child,
                &request.id,
                "application snapshots are unavailable",
            )?
            .ok_or("application snapshots are unavailable")?;
            let leaf_path = root.join(&request.id);
            super::home::fd_matches_path(&leaf_path, &leaf)
                .map_err(|_| "application snapshots are unavailable".to_string())?;
            let info = snapshot::application_info(&leaf_path)?;
            // FJ-855: the leaf must still be the anchored child after the
            // read — a replacement across the read fails closed here
            // instead of returning bytes from a swapped tree.
            let rebound = super::home::open_child_dir_opt(
                &child,
                &request.id,
                "application snapshots are unavailable",
            )?
            .ok_or("application snapshots are unavailable")?;
            super::home::fd_matches_path(&leaf_path, &rebound)
                .map_err(|_| "application snapshots are unavailable".to_string())?;
            if info.application.id != request.application.id {
                return Err("snapshot belongs to another application".into());
            }
            return Ok(Inventory {
                snapshots: vec![info],
                complete: true,
            });
        }
        if request.action == "backup" {
            // FJ-847: the snapshots directory is created through the
            // re-validated storage-root anchor — never by a bare create_dir
            // — and only a directory this call just created is assigned its
            // mode; an existing tree is re-validated, never re-owned.
            let snapshots_anchor = super::home::make_child_dir(
                &anchor,
                "snapshots",
                0,
                "application snapshots are unavailable",
            )?;
            super::home::fd_matches_path(&root, &snapshots_anchor)
                .map_err(|_| "application snapshots are unavailable".to_string())?;
            let path = root.join(&request.id);
            let bound = super::home::open_child_dir_opt(
                &snapshots_anchor,
                &request.id,
                "application snapshots are unavailable",
            )?;
            if let Some(existing_dir) = bound {
                super::home::fd_matches_path(&path, &existing_dir)
                    .map_err(|_| "application snapshots are unavailable".to_string())?;
                let existing = snapshot::application_info(&path)?;
                if existing.id != request.id || existing.application != request.application {
                    return Err("snapshot identity is already bound to another operation".into());
                }
            } else {
                let spec = &request.application;
                let native = self.inspect(spec, &request.expected_container, deadline)?;
                if native.running || native.image != spec.image {
                    return Err("stop the admitted application before backing up".into());
                }
                snapshot::backup_application(
                    &super::runtime::data_directory(&spec.data)?,
                    &path,
                    SnapshotInfo {
                        id: request.id.clone(),
                        created_at: SystemTime::now()
                            .duration_since(UNIX_EPOCH)
                            .map_err(|_| "snapshot time unavailable")?
                            .as_secs(),
                        application: spec.clone(),
                    },
                    native
                        .labels
                        .get("io.canlang.storage")
                        .ok_or("application image has no admitted storage fingerprint")?,
                )?;
                // Guard the handoff to the snapshot reader: the fresh tree
                // must still be the anchored child.
                let fresh = super::home::open_child_dir_opt(
                    &snapshots_anchor,
                    &request.id,
                    "application snapshots are unavailable",
                )?
                .ok_or("application snapshots are unavailable")?;
                super::home::fd_matches_path(&path, &fresh)
                    .map_err(|_| "application snapshots are unavailable".to_string())?;
            }
            return Ok(Inventory {
                snapshots: vec![snapshot::application_info(&path)?],
                complete: true,
            });
        }
        // FJ-855: the inventory scan resolves through the storage-root
        // anchor. An absent snapshots tree is still an empty inventory; a
        // present one is rebound by dev/ino before the scan, and every
        // selected leaf is opened through the anchor and rebound again
        // before the snapshot module reads it — a leaf swapped between the
        // scan and the read is skipped, never selected.
        let child = super::home::open_child_dir_opt(
            &anchor,
            "snapshots",
            "application snapshots are unavailable",
        )?;
        let child = match child {
            Some(child) => child,
            None => {
                return Ok(Inventory {
                    snapshots: Vec::new(),
                    complete: true,
                })
            }
        };
        super::home::fd_matches_path(&root, &child)
            .map_err(|_| "snapshot inventory is unavailable".to_string())?;
        let mut snapshots = Vec::new();
        let entries = match fs::read_dir(&root) {
            Ok(entries) => entries,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                return Ok(Inventory {
                    snapshots,
                    complete: true,
                })
            }
            Err(_) => return Err("snapshot inventory is unavailable".into()),
        };
        for entry in entries {
            let entry = entry.map_err(|_| "snapshot inventory is unavailable")?;
            if !entry.file_name().to_str().is_some_and(id) {
                continue;
            }
            let name = entry.file_name();
            let leaf = match super::home::open_child_dir_opt(
                &child,
                name.to_str().ok_or("snapshot inventory is unavailable")?,
                "snapshot inventory is unavailable",
            )? {
                Some(leaf) => leaf,
                None => continue,
            };
            if super::home::fd_matches_path(&entry.path(), &leaf).is_err() {
                continue;
            }
            // An interrupted, unsealed copy is retained but is never selectable.
            if let Ok(info) = snapshot::application_info(&entry.path()) {
                if info.application.id == request.application.id {
                    snapshots.push(info);
                }
            }
        }
        snapshots.sort_by(|a, b| {
            b.created_at
                .cmp(&a.created_at)
                .then_with(|| a.id.cmp(&b.id))
        });
        let complete = snapshots.len() <= 64;
        snapshots.truncate(64);
        Ok(Inventory {
            snapshots,
            complete,
        })
    }
}
