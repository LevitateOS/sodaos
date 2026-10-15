//! Bounded root-Podman inventory. Immutable argv, exact CID/image and every
//! ownership label are checked; neither a name nor a label grants cleanup.
use std::collections::BTreeSet;
use std::fs;
use std::os::unix::fs::{FileTypeExt, MetadataExt};
use std::path::Path;
use std::time::Instant;

use crate::project::Executor;
use crate::tailnet_domain::{
    valid_container_id, valid_project_id, RunTarget, ERR_CONFLICT, ERR_UNCONFIRMED,
};
use crate::tailnet_files::read_companion_record;
use crate::tailnet_runtime::companion_container_args;

use super::{
    companion_command_matches, companion_execs_valid, decode_companion_record,
    match_companion_namespaces, validate_companion_record, Companion, CompanionRecord,
    TailnetControl, COMPANION_INSPECT, RUNTIME_ROOT,
};

pub struct NativeChild {
    pub record: CompanionRecord,
    pub target: RunTarget,
    pub image: String,
    pub installation: String,
    pub generation: String,
}

pub struct ChildInventory {
    pub children: Vec<NativeChild>,
    pub unknown: bool,
}

impl<E: Executor, T: TailnetControl> Companion<E, T> {
    fn inspect_native_child(
        &self,
        cid: &str,
        installation: &str,
        deadline: Instant,
    ) -> Result<NativeChild, String> {
        let data = self.runtime_podman(
            &[
                "inspect".into(),
                "--format".into(),
                COMPANION_INSPECT.into(),
                cid.into(),
            ],
            deadline,
        )?;
        let record = decode_companion_record(&data).map_err(|_| ERR_UNCONFIRMED.to_string())?;
        let label = |key: &str| {
            record
                .labels
                .get(key)
                .cloned()
                .ok_or_else(|| ERR_CONFLICT.to_string())
        };
        let target = RunTarget {
            project: label("project")?,
            run: label("run")?,
            container: label("parent")?,
        };
        let generation = label("generation")?;
        let image = format!(
            "sha256:{}",
            record
                .image
                .strip_prefix("sha256:")
                .unwrap_or(&record.image)
        );
        let args = companion_container_args(&target, &image, installation, &generation)?;
        if !valid_container_id(cid)
            || record.id != cid
            || label("installation")? != installation
            || record.labels.len() != 5
            || !companion_command_matches(&record.command, &args)
        {
            return Err(ERR_CONFLICT.to_string());
        }
        companion_execs_valid(&record.execs)?;
        if record.running
            && (record.pid <= 1
                || !matches!(
                    crate::tailnet_domain::parse_rfc3339_nano(&record.started),
                    Some(false)
                ))
        {
            return Err(ERR_CONFLICT.to_string());
        }
        let path = Path::new(RUNTIME_ROOT)
            .join(&target.project)
            .join(&target.run)
            .join("companion.json");
        match fs::symlink_metadata(&path) {
            Ok(_) => {
                let owned = read_companion_record(RUNTIME_ROOT, &target, 0, 0)?;
                validate_companion_record(&record, &owned.parent, &owned)?;
                match_companion_namespaces(&record, &owned.parent)?;
            }
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {} // Reboot retains the root native metadata above.
            Err(_) => return Err(ERR_UNCONFIRMED.to_string()),
        }
        Ok(NativeChild {
            record,
            target,
            image,
            installation: installation.into(),
            generation,
        })
    }

    pub fn native_children(
        &self,
        installation: &str,
        host: Option<&crate::tailnet_host::HostProcess>,
        deadline: Instant,
    ) -> ChildInventory {
        let mut inventory = ChildInventory {
            children: Vec::new(),
            unknown: false,
        };
        if !crate::tcontrol_wire::valid_lifecycle_uuid(installation) {
            inventory.unknown = true;
            return inventory;
        }
        let data = match self.runtime_podman(
            &[
                "ps".into(),
                "-a".into(),
                "--no-trunc".into(),
                "--format={{.ID}}".into(),
                "--filter".into(),
                format!("label=org.soda.tailnet.installation={installation}"),
            ],
            deadline,
        ) {
            Ok(data) => data,
            Err(_) => {
                inventory.unknown = true;
                return inventory;
            }
        };
        let text = match std::str::from_utf8(&data) {
            Ok(text) => text,
            Err(_) => {
                inventory.unknown = true;
                return inventory;
            }
        };
        let ids: BTreeSet<&str> = text.split_whitespace().collect();
        let host_id = host
            .filter(|host| host.record.installation_id == installation)
            .map(|host| host.record.id.as_str());
        let ids: Vec<_> = ids.into_iter().filter(|id| Some(*id) != host_id).collect();
        if ids.len() > 128 {
            inventory.unknown = true;
        }
        for id in ids.into_iter().take(128) {
            if Instant::now() >= deadline {
                inventory.unknown = true;
                break;
            }
            match self.inspect_native_child(id, installation, deadline) {
                Ok(child) => inventory.children.push(child),
                Err(_) => inventory.unknown = true,
            }
        }
        inventory
    }

    pub fn stop_native_child(&self, child: &NativeChild, deadline: Instant) -> Result<(), String> {
        let before = self.inspect_native_child(&child.record.id, &child.installation, deadline)?;
        if before.image != child.image
            || before.target != child.target
            || before.generation != child.generation
        {
            return Err(ERR_CONFLICT.to_string());
        }
        if before.record.running {
            self.stop_companion(&before.record.id, deadline)?;
        }
        if self
            .inspect_native_child(&child.record.id, &child.installation, deadline)?
            .record
            .running
        {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        // The root-only ancestors and immutable mount recipe establish the
        // socket's scope even when a reboot removed its ephemeral receipt.
        let base = Path::new(RUNTIME_ROOT);
        let project = base.join(&child.target.project);
        let run = project.join(&child.target.run);
        for path in [base, &project, &run] {
            match fs::symlink_metadata(path) {
                Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(()),
                Ok(info)
                    if info.is_dir()
                        && info.uid() == 0
                        && info.gid() == 0
                        && info.mode() & 0o7777 == 0o700 => {}
                _ => return Err(ERR_UNCONFIRMED.to_string()),
            }
        }
        let control = run.join("control");
        match fs::symlink_metadata(&control) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(()),
            Ok(info)
                if info.is_dir()
                    && info.mode() & 0o7777 == 0o700
                    && info.uid() != 0
                    && info.gid() != 0 => {}
            _ => return Err(ERR_UNCONFIRMED.to_string()),
        }
        let socket = control.join("tailscaled.sock");
        match fs::symlink_metadata(&socket) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
            Ok(info) if info.file_type().is_socket() => {
                fs::remove_file(socket).map_err(|_| ERR_UNCONFIRMED.to_string())
            }
            _ => Err(ERR_UNCONFIRMED.to_string()),
        }
    }

    pub fn known_projects(&self) -> Result<BTreeSet<String>, String> {
        let mut projects = BTreeSet::new();
        match fs::read_dir(RUNTIME_ROOT) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(projects),
            Err(_) => return Err(ERR_UNCONFIRMED.to_string()),
            Ok(entries) => {
                for entry in entries {
                    let entry = entry.map_err(|_| ERR_UNCONFIRMED.to_string())?;
                    let id = entry
                        .file_name()
                        .into_string()
                        .map_err(|_| ERR_UNCONFIRMED.to_string())?;
                    if valid_project_id(&id) {
                        projects.insert(id);
                    }
                    if projects.len() > 128 {
                        return Err(ERR_UNCONFIRMED.to_string());
                    }
                }
            }
        }
        Ok(projects)
    }
}
