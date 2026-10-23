use super::runtime::Runtime;
use super::snapshot::{self, SnapshotInfo};
use super::wire::{hex, id, Spec};
use crate::project::Executor;
use serde::{Deserialize, Serialize};
use std::fs;
use std::os::unix::fs::{MetadataExt, PermissionsExt};
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
    let root = super::home_path(std::path::Path::new("/home/soda-applications"))?;
    let parent =
        fs::symlink_metadata(&root).map_err(|_| "application storage root is unavailable")?;
    if parent.uid() != 0 || !parent.is_dir() || parent.mode() & 0o022 != 0 {
        return Err("application storage root must belong to the operator".into());
    }
    let path = root.join("snapshots");
    let meta = match fs::symlink_metadata(&path) {
        Ok(meta) => meta,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(path),
        Err(_) => return Err("application snapshots are unavailable".into()),
    };
    if !meta.is_dir()
        || meta.file_type().is_symlink()
        || meta.uid() != 0
        || meta.mode() & 0o077 != 0
    {
        return Err("application snapshots must be private operator-owned directories".into());
    }
    Ok(path)
}

impl<E: Executor> Runtime<E> {
    pub fn snapshots(&self, request: &Request, deadline: Instant) -> Result<Inventory, String> {
        request.application.validate()?;
        match request.action.as_str() {
            "list" if request.id.is_empty() && request.expected_container.is_empty() => (),
            "inspect" if id(&request.id) && request.expected_container.is_empty() => (),
            "backup" if id(&request.id) && hex(&request.expected_container, 64) => (),
            _ => return Err("invalid application snapshot operation".into()),
        }
        let root = root()?;
        if request.action == "inspect" {
            let info = snapshot::application_info(&root.join(&request.id))?;
            if info.application.id != request.application.id {
                return Err("snapshot belongs to another application".into());
            }
            return Ok(Inventory {
                snapshots: vec![info],
                complete: true,
            });
        }
        if request.action == "backup" {
            match fs::create_dir(&root) {
                Ok(()) => fs::set_permissions(&root, fs::Permissions::from_mode(0o700))
                    .map_err(|e| e.to_string())?,
                Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => (),
                Err(_) => return Err("application snapshots are unavailable".into()),
            }
            let path = root.join(&request.id);
            if path.exists() {
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
            }
            return Ok(Inventory {
                snapshots: vec![snapshot::application_info(&path)?],
                complete: true,
            });
        }
        let mut snapshots = Vec::new();
        let entries = match fs::read_dir(root) {
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
