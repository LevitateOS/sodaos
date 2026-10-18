//! Bounded root-Podman inventory. Immutable argv, exact CID/image and every
//! ownership label are checked; neither a name nor a label grants cleanup.
use std::collections::BTreeSet;
use std::fs;
use std::os::unix::fs::{FileTypeExt, MetadataExt, OpenOptionsExt};
use std::path::Path;
use std::time::Instant;

use crate::project::Executor;
use crate::tailnet_domain::{
    valid_container_id, valid_project_id, RunTarget, ERR_CONFLICT, ERR_INVALID, ERR_UNAVAILABLE,
    ERR_UNCONFIRMED,
};
use crate::tailnet_files::read_companion_record;
use crate::tailnet_runtime::{companion_container_args, decode_project_run, valid_project_run};

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

/// Bounded effect batch for one obsolete-retirement pass. A pass may leave
/// unknown remainder; repeated authorized passes drain the backlog instead of
/// one unbounded sweep.
pub const OBSOLETE_RETIRE_BATCH: usize = 16;

/// Deterministic FNV-1a 64-bit mix over the container ID bytes. Stable across
/// processes (unlike `RandomState`) and dependency-free; paired with the ID
/// itself it is a total order.
pub(crate) fn scan_mix(id: &str) -> u64 {
    let mut hash: u64 = 0xcbf29ce484222325;
    for byte in id.bytes() {
        hash ^= u64::from(byte);
        hash = hash.wrapping_mul(0x100000001b3);
    }
    hash
}

/// Deterministic mixed-byte exploration order for the bounded inventory scan.
/// A fixed lexicographic prefix (for example many live current children) must
/// not monopolize the 128-item inspection bound and starve trailing obsolete
/// candidates, so IDs are explored in hash-mixed order instead of sorted
/// order. Pure: every input key appears exactly once, in full.
pub(crate) fn scan_order(mut ids: Vec<String>) -> Vec<String> {
    ids.sort_by(|a, b| scan_mix(a).cmp(&scan_mix(b)).then_with(|| a.cmp(b)));
    ids
}

/// Check the root-only runtime scope dirs and return the control dir.
/// A missing path means the runtime is already gone: the caller is done.
fn scope_control_dir(child: &NativeChild) -> Result<Option<std::path::PathBuf>, String> {
    let base = Path::new(RUNTIME_ROOT);
    let project = base.join(&child.target.project);
    let run = project.join(&child.target.run);
    for path in [base, &project, &run] {
        match fs::symlink_metadata(path) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(None),
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
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(None),
        Ok(info)
            if info.is_dir()
                && info.mode() & 0o7777 == 0o700
                && info.uid() != 0
                && info.gid() != 0 =>
        {
            Ok(Some(control))
        }
        _ => Err(ERR_UNCONFIRMED.to_string()),
    }
}

/// Remove the control socket when present; absence is already clean.
fn remove_control_socket(control: &Path) -> Result<(), String> {
    let socket = control.join("tailscaled.sock");
    match fs::symlink_metadata(&socket) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Ok(info) if info.file_type().is_socket() => {
            fs::remove_file(socket).map_err(|_| ERR_UNCONFIRMED.to_string())
        }
        _ => Err(ERR_UNCONFIRMED.to_string()),
    }
}

/// Decode an inspected container into its record, run target, image and
/// generation. Every missing label conflicts, like the inline lookup.
fn decode_inspected_child(
    data: &[u8],
) -> Result<(CompanionRecord, RunTarget, String, String), String> {
    let record = decode_companion_record(data).map_err(|_| ERR_UNCONFIRMED.to_string())?;
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
    Ok((record, target, image, generation))
}

/// Check the inspected execs and the running-state proof.
fn check_inspected_running(record: &CompanionRecord) -> Result<(), String> {
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
    Ok(())
}

/// Cross-check the on-disk companion receipt when present. A reboot may
/// have removed it; the root native metadata above still stands.
fn check_companion_receipt(record: &CompanionRecord, target: &RunTarget) -> Result<(), String> {
    let path = Path::new(RUNTIME_ROOT)
        .join(&target.project)
        .join(&target.run)
        .join("companion.json");
    match fs::symlink_metadata(&path) {
        Ok(_) => {
            let owned = read_companion_record(RUNTIME_ROOT, target, 0, 0)?;
            validate_companion_record(record, &owned.parent, &owned)?;
            match_companion_namespaces(record, &owned.parent)?;
        }
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {} // Reboot retains the root native metadata above.
        Err(_) => return Err(ERR_UNCONFIRMED.to_string()),
    }
    Ok(())
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
        let (record, target, image, generation) = decode_inspected_child(&data)?;
        let args = companion_container_args(&target, &image, installation, &generation)?;
        let installation_label = |key: &str| {
            record
                .labels
                .get(key)
                .cloned()
                .ok_or_else(|| ERR_CONFLICT.to_string())
        };
        if !valid_container_id(cid)
            || record.id != cid
            || installation_label("installation")? != installation
            || record.labels.len() != 5
            || !companion_command_matches(&record.command, &args)
        {
            return Err(ERR_CONFLICT.to_string());
        }
        check_inspected_running(&record)?;
        check_companion_receipt(&record, &target)?;
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
        let host_id = host
            .filter(|host| host.record.installation_id == installation)
            .map(|host| host.record.id.as_str());
        let mut seen = BTreeSet::new();
        let ids: Vec<String> = text
            .split_whitespace()
            .filter(|id| Some(*id) != host_id)
            .filter(|id| seen.insert((*id).to_string()))
            .map(str::to_string)
            .collect();
        // Mixed-byte exploration order: no fixed sorted prefix may consume
        // every inspection slot while trailing obsolete candidates wait.
        let ids = scan_order(ids);
        if ids.len() > 128 {
            inventory.unknown = true;
        }
        for id in ids.into_iter().take(128) {
            if Instant::now() >= deadline {
                inventory.unknown = true;
                break;
            }
            match self.inspect_native_child(&id, installation, deadline) {
                Ok(child) => inventory.children.push(child),
                Err(_) => inventory.unknown = true,
            }
        }
        inventory
    }

    /// Read-only current-run lookup for one project. Mirrors the validation in
    /// `RunFiles::current` (owner/mode/link/size/parse/run checks) without
    /// creating directories or taking locks; retirement callers already hold
    /// lifecycle exclusion. A missing or unreadable record is refusal, never
    /// absence: callers preserve those children.
    pub(crate) fn current_run_id(&self, project: &str) -> Result<String, String> {
        if !valid_project_id(project) {
            return Err(ERR_INVALID.to_string());
        }
        let path = Path::new(RUNTIME_ROOT).join(project).join("current.json");
        let mut opts = std::fs::OpenOptions::new();
        opts.read(true).custom_flags(libc::O_NOFOLLOW);
        let mut file = match opts.open(&path) {
            Ok(file) => file,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                return Err("runtime record not found".to_string());
            }
            Err(_) => return Err(ERR_UNAVAILABLE.to_string()),
        };
        let info = file.metadata().map_err(|_| ERR_UNAVAILABLE.to_string())?;
        if !info.is_file()
            || info.mode() & 0o777 != 0o600
            || info.uid() != 0
            || info.gid() != 0
            || info.nlink() != 1
            || info.len() > 8192
        {
            return Err(ERR_UNAVAILABLE.to_string());
        }
        use std::io::Read;
        let mut buf = Vec::new();
        file.read_to_end(&mut buf)
            .map_err(|_| ERR_UNAVAILABLE.to_string())?;
        let run = decode_project_run(&buf).map_err(|_| ERR_UNAVAILABLE.to_string())?;
        if !valid_project_run(&run) || run.target.project != project {
            return Err(ERR_UNAVAILABLE.to_string());
        }
        Ok(run.target.run)
    }

    /// Remove one verified stopped owned child. Stops the verified child
    /// first, reinspects by exact ID immediately before removal, and only then
    /// removes without force. Identity drift or a still-running child refuses;
    /// nothing foreign is ever named: no prune, no name-only removal, no
    /// forced removal, no credential/volume/image effects.
    pub(crate) fn retire_verified_stopped_child(
        &self,
        child: &NativeChild,
        deadline: Instant,
    ) -> Result<(), String> {
        self.stop_native_child(child, deadline)?;
        let after = self
            .inspect_native_child(&child.record.id, &child.installation, deadline)
            .map_err(|_| ERR_UNCONFIRMED.to_string())?;
        if after.record.running
            || after.image != child.image
            || after.target != child.target
            || after.generation != child.generation
        {
            return Err(ERR_CONFLICT.to_string());
        }
        self.runtime_podman(&["rm".into(), child.record.id.clone()], deadline)
            .map(|_| ())
            .map_err(|_| ERR_UNCONFIRMED.to_string())
    }

    /// Bounded retirement of verified stopped obsolete companions selected by
    /// the caller. The caller establishes obsolescence (for example a different
    /// verified current run for the same project); every candidate is still
    /// re-verified by exact ID immediately before non-forced removal. Running
    /// children are never selected here, and an unknown remainder beyond the
    /// inspected bound stays unconfirmed while the known subset still makes
    /// progress. Returns the number retired.
    pub(crate) fn retire_stopped_where(
        &self,
        installation: &str,
        host: Option<&crate::tailnet_host::HostProcess>,
        should_retire: &dyn Fn(&NativeChild) -> bool,
        limit: usize,
        deadline: Instant,
    ) -> Result<usize, String> {
        let inventory = self.native_children(installation, host, deadline);
        let mut retired = 0;
        for child in &inventory.children {
            if retired >= limit {
                break;
            }
            if child.record.running || !should_retire(child) {
                continue;
            }
            self.retire_verified_stopped_child(child, deadline)?;
            retired += 1;
        }
        Ok(retired)
    }

    /// Reject a child whose inspected identity drifted from the stop request.
    fn check_stop_identity(before: &NativeChild, child: &NativeChild) -> Result<(), String> {
        if before.image != child.image
            || before.target != child.target
            || before.generation != child.generation
        {
            return Err(ERR_CONFLICT.to_string());
        }
        Ok(())
    }

    /// Prove the child stayed stopped after the stop call.
    fn confirm_child_stopped(&self, child: &NativeChild, deadline: Instant) -> Result<(), String> {
        if self
            .inspect_native_child(&child.record.id, &child.installation, deadline)?
            .record
            .running
        {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        Ok(())
    }

    pub fn stop_native_child(&self, child: &NativeChild, deadline: Instant) -> Result<(), String> {
        let before = self.inspect_native_child(&child.record.id, &child.installation, deadline)?;
        Self::check_stop_identity(&before, child)?;
        if before.record.running {
            self.stop_companion(&before.record.id, deadline)?;
        }
        self.confirm_child_stopped(child, deadline)?;
        // The root-only ancestors and immutable mount recipe establish the
        // socket's scope even when a reboot removed its ephemeral receipt.
        let Some(control) = scope_control_dir(child)? else {
            return Ok(());
        };
        remove_control_socket(&control)
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
