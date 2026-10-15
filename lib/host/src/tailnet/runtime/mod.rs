//! Tailnet host runtime: /proc identity, podman inspection, run admission.
//! Lane B owns this file.
//!
//! Port of `internal/host/tailnet/runtime.go`. Error texts match the Go
//! domain errors byte for byte; podman argv and output caps are preserved.

use std::time::Instant;

use crate::prepare::path_join;
use crate::project::Executor;
use crate::tailnet_domain::{
    parse_rfc3339_nano, valid_container_id, valid_image_id, valid_project_id, LinkFile, ReadFile,
    RunTarget, ERR_CONFLICT, ERR_INVALID, ERR_UNAVAILABLE, ERR_UNSUPPORTED,
};

/// Podman inspect template for the project container run snapshot.
/// Excludes Config.Env and other credential-bearing full-inspection fields.
pub const TAILNET_RUN_INSPECT: &str = "{\"id\":{{json .ID}},\"running\":{{json .State.Running}},\"pid\":{{json .State.Pid}},\"started\":{{json .State.StartedAt}},\"resolver\":{{json .ResolvConfPath}}}";

/// Admitted native incarnation of one project container run.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProjectRun {
    pub target: RunTarget,
    pub pid: i64,
    pub started: String,
    pub userns: String,
    pub netns: String,
    pub uid: u32,
    pub gid: u32,
    pub resolver: String,
}

/// Root-owned identity of the child, independent of the selected next image.
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct OwnedCompanion {
    #[serde(with = "owned_parent_wire")]
    pub parent: ProjectRun,
    pub id: String,
    pub image: String,
    pub installation_id: String,
    pub generation: String,
}

impl OwnedCompanion {
    pub fn valid(&self) -> bool {
        valid_project_run(&self.parent)
            && valid_container_id(&self.id)
            && self.image.starts_with("sha256:")
            && valid_image_id(&self.image)
            && crate::tcontrol_wire::valid_lifecycle_uuid(&self.installation_id)
            && crate::tcontrol_wire::valid_lifecycle_generation(&self.generation)
    }
}

mod owned_parent_wire {
    use super::ProjectRun;
    use serde::{Deserialize, Deserializer, Serializer};
    pub fn serialize<S: Serializer>(run: &ProjectRun, serializer: S) -> Result<S::Ok, S::Error> {
        use serde::Serialize;
        let raw = serde_json::value::RawValue::from_string(super::marshal_project_run(run))
            .map_err(serde::ser::Error::custom)?;
        raw.serialize(serializer)
    }
    pub fn deserialize<'de, D: Deserializer<'de>>(deserializer: D) -> Result<ProjectRun, D::Error> {
        let raw = Box::<serde_json::value::RawValue>::deserialize(deserializer)?;
        super::decode_project_run(raw.get().as_bytes()).map_err(serde::de::Error::custom)
    }
}

pub fn valid_project_run(run: &ProjectRun) -> bool {
    valid_project_id(&run.target.project)
        && valid_container_id(&run.target.container)
        && valid_container_id(&run.target.run)
        && run.uid != 0
        && run.gid != 0
}

/// /proc-derived process identity bound into the run hash.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProcessIdentity {
    pub start: String,
    pub userns: String,
    pub netns: String,
    pub boot: String,
    pub uid: u32,
    pub gid: u32,
}

/// Minimal podman run snapshot of the project container.
#[derive(Debug, Clone, Default)]
pub struct ProjectRunInspect {
    pub id: String,
    pub running: bool,
    pub pid: i64,
    pub started: String,
    pub resolver: String,
}

/// Podman inspection of the project terminal container.
#[derive(Debug, Clone, Default)]
pub struct ProjectInspection {
    pub id: String,
    pub running: bool,
    pub project: String,
    pub owner: String,
    pub privileged: bool,
    pub userns: String,
    pub uid_map: Vec<String>,
    pub gid_map: Vec<String>,
}

fn unavailable() -> String {
    ERR_UNAVAILABLE.to_string()
}

mod process;

pub use process::{
    parse_namespace_link, parse_process_start_time, parse_stat_fields, process_run_identity,
    read_process_id_map, read_process_namespace, read_system_boot_id,
};

mod wire;

pub use wire::{
    assemble_project_run, decode_project_run, decode_project_run_inspect, marshal_project_run,
};

fn real_read() -> ReadFile {
    Box::new(|path: &str| std::fs::read(path).map_err(|_| unavailable()))
}

fn real_link() -> LinkFile {
    Box::new(|path: &str| {
        std::fs::read_link(path)
            .map_err(|_| unavailable())?
            .into_os_string()
            .into_string()
            .map_err(|_| unavailable())
    })
}

/// Admit one run snapshot against the live native state (real /proc reads).
/// Resolver inode published by the pinned rootful Podman runtime. This is
/// runroot metadata; graphroot still owns the persistent container storage.
pub fn project_resolver_path(cid: &str) -> String {
    format!("/run/containers/storage/overlay-containers/{cid}/userdata/resolv.conf")
}

/// Only the selected native rootful storage resolver is considered.
pub fn admit_project_run_snapshot(
    cid: &str,
    raw: &ProjectRunInspect,
) -> Result<(String, ProcessIdentity), String> {
    match parse_rfc3339_nano(&raw.started) {
        Some(false) => {}
        _ => return Err(unavailable()),
    }
    let resolver = project_resolver_path(cid);
    if raw.resolver != resolver {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    let identity = process_run_identity(raw.pid, real_read(), real_link())?;
    Ok((resolver, identity))
}

/// Re-observe the snapshot bytes and the native identity; any drift conflicts.
pub fn confirm_project_run_identity(
    exec: &dyn Executor,
    cid: &str,
    data: &[u8],
    pid: i64,
    identity: &ProcessIdentity,
    deadline: Instant,
) -> Result<(), String> {
    match inspect_project_run(exec, cid, deadline) {
        Ok(again) if again.as_slice() == data => {}
        _ => return Err(ERR_CONFLICT.to_string()),
    }
    let second = process_run_identity(pid, real_read(), real_link())
        .map_err(|_| ERR_CONFLICT.to_string())?;
    if &second != identity {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(())
}

/// Minimal podman snapshot of one project container.
pub fn inspect_project_run(
    exec: &dyn Executor,
    cid: &str,
    deadline: Instant,
) -> Result<Vec<u8>, String> {
    exec.run(
        &[],
        "/usr/bin/podman",
        &[
            "--remote=false",
            "inspect",
            "--format",
            TAILNET_RUN_INSPECT,
            cid,
        ],
        deadline,
    )
}

/// Resolve, snapshot, admit and re-bind the current run of one project.
pub fn project_run(exec: &dyn Executor, id: &str, deadline: Instant) -> Result<ProjectRun, String> {
    let cid = match project_container(exec, id, true, deadline) {
        Ok(cid) => cid,
        Err(_) => return Err(unavailable()),
    };
    let data = match inspect_project_run(exec, &cid, deadline) {
        Ok(data) if data.len() <= 4096 => data,
        _ => return Err(unavailable()),
    };
    let raw = decode_project_run_inspect(&data, &cid)?;
    let (resolver, identity) = admit_project_run_snapshot(&cid, &raw)?;
    confirm_project_run_identity(exec, &cid, &data, raw.pid, &identity, deadline)?;
    match project_container(exec, id, true, deadline) {
        Ok(current) if current == cid => {}
        _ => return Err(ERR_CONFLICT.to_string()),
    }
    Ok(assemble_project_run(id, &cid, &resolver, &raw, &identity))
}

/// Immutable companion `create` recipe joining the project's namespaces.
/// The image is host configuration (pinned digest), never a caller field.
pub fn companion_create_args(
    run: &ProjectRun,
    owned: &OwnedCompanion,
) -> Result<Vec<String>, String> {
    if !valid_project_run(run) {
        return Err(ERR_INVALID.to_string());
    }
    companion_container_args(
        &run.target,
        &owned.image,
        &owned.installation_id,
        &owned.generation,
    )
}

/// Exact immutable native recipe, also verifiable from root Podman metadata
/// after reboot has discarded ephemeral run files and parent processes.
pub fn companion_container_args(
    target: &RunTarget,
    image: &str,
    installation: &str,
    generation: &str,
) -> Result<Vec<String>, String> {
    if !valid_project_id(&target.project)
        || !valid_container_id(&target.container)
        || !valid_container_id(&target.run)
        || !image.starts_with("sha256:")
        || !valid_image_id(image)
        || !crate::tcontrol_wire::valid_lifecycle_uuid(installation)
        || !crate::tcontrol_wire::valid_lifecycle_generation(generation)
    {
        return Err(ERR_INVALID.to_string());
    }
    let base = path_join(&[
        "/run/soda-tailnet",
        target.project.as_str(),
        target.run.as_str(),
    ]);
    Ok(vec![
        "--remote=false".to_string(),
        "create".to_string(),
        "--name".to_string(),
        format!("soda-tailnet-{}-{}", target.project, target.run),
        "--label".to_string(),
        format!("org.soda.tailnet.project={}", target.project),
        "--label".to_string(),
        format!("org.soda.tailnet.run={}", target.run),
        "--label".to_string(),
        format!("org.soda.tailnet.parent={}", target.container),
        "--label".into(),
        format!("org.soda.tailnet.installation={}", installation),
        "--label".into(),
        format!("org.soda.tailnet.generation={}", generation),
        format!("--userns=container:{}", target.container),
        format!("--network=container:{}", target.container),
        "--pid=private".to_string(),
        "--ipc=private".to_string(),
        "--uts=private".to_string(),
        "--cgroupns=private".to_string(),
        "--user=0:0".to_string(),
        "--cap-drop=ALL".to_string(),
        "--cap-add=NET_ADMIN".to_string(),
        "--device=/dev/net/tun".to_string(),
        "--security-opt=label=disable".to_string(),
        "--no-hosts".to_string(),
        "--log-driver=none".to_string(),
        "--pull=never".to_string(),
        "--volume".to_string(),
        format!("{base}/control:/run/tailscale:rw"),
        "--volume".to_string(),
        format!("{base}/input:/run/soda-enrollment:ro"),
        "--entrypoint=/usr/local/bin/tailscaled".to_string(),
        image.to_string(),
        "--state=mem:".to_string(),
        "--socket=/run/tailscale/tailscaled.sock".to_string(),
        "--tun=tailscale0".to_string(),
        "--no-logs-no-support".to_string(),
    ])
}

/// Re-resolve the project's run and require byte-identical incarnation.
pub fn recheck_project_run(
    exec: &dyn Executor,
    before: &ProjectRun,
    deadline: Instant,
) -> Result<(), String> {
    match project_run(exec, &before.target.project, deadline) {
        Ok(after) if after == *before => Ok(()),
        _ => Err(ERR_CONFLICT.to_string()),
    }
}

mod project;

pub use project::{id_map, inspect_project, project_container, project_isolation, project_running};

#[cfg(test)]
mod tests;

#[cfg(test)]
mod project_tests;

#[cfg(test)]
pub(crate) fn test_ownership(image: &str, parent: &ProjectRun) -> OwnedCompanion {
    OwnedCompanion {
        parent: parent.clone(),
        id: "e".repeat(64),
        image: image.into(),
        installation_id: "550e8400-e29b-41d4-a716-446655440000".into(),
        generation: "7".into(),
    }
}
