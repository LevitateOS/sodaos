//! Tailnet host runtime: /proc identity, podman inspection, run admission.
//! Lane B owns this file.
//!
//! Port of `internal/host/tailnet/runtime.go`. Error texts match the Go
//! domain errors byte for byte; podman argv and output caps are preserved.

use std::time::Instant;

use crate::json::{bind_root, decode_strict, parse_go_int64, Kind, Spec};
use crate::prepare::path_join;
use crate::project::{Executor, PROJECT_INSPECT_FORMAT};
use crate::tailnet_domain::{
    go_escape, parse_rfc3339_nano, valid_container_id, valid_image_id, valid_project_id, LinkFile,
    ReadFile, RunTarget, ERR_CONFLICT, ERR_INVALID, ERR_UNAVAILABLE, ERR_UNSUPPORTED,
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

/// Go `strings.Fields` over /proc text: Unicode whitespace when the input is
/// valid UTF-8, ASCII-whitespace byte split otherwise.
fn split_fields(raw: &[u8]) -> Vec<String> {
    match std::str::from_utf8(raw) {
        Ok(s) => s.split_whitespace().map(str::to_string).collect(),
        Err(_) => raw
            .split(|b| matches!(*b, b' ' | b'\t' | b'\n' | b'\r' | b'\x0c' | b'\x0b'))
            .filter(|part| !part.is_empty())
            .map(|part| String::from_utf8_lossy(part).into_owned())
            .collect(),
    }
}

/// Split `/proc/<pid>/stat` into state-first fields, refusing oversized,
/// mismatched-pid and dead (zombie/exited) records.
pub fn parse_stat_fields(pid: i64, data: &[u8]) -> Result<Vec<String>, String> {
    if data.len() > 8192 {
        return Err(unavailable());
    }
    let end = data.iter().rposition(|&b| b == b')');
    let first = data.iter().position(|&b| b == b' ');
    match (end, first) {
        (Some(end), Some(first)) if &data[..first] == pid.to_string().as_bytes() => {
            let fields = split_fields(&data[end + 1..]);
            if fields.len() < 20 || fields[0] == "Z" || fields[0] == "X" {
                return Err(unavailable());
            }
            Ok(fields)
        }
        _ => Err(unavailable()),
    }
}

fn parse_start_time_inner(pid: i64, read: &ReadFile) -> Result<String, String> {
    let path = format!("/proc/{pid}/stat");
    let data = read(&path).map_err(|_| unavailable())?;
    let fields = parse_stat_fields(pid, &data)?;
    let token = &fields[19];
    match token.parse::<u64>() {
        Ok(n) if n != 0 && n.to_string() == *token => Ok(token.clone()),
        _ => Err(unavailable()),
    }
}

/// Canonical start-time token (field 22) of `/proc/<pid>/stat`.
pub fn parse_process_start_time(pid: i64, read: ReadFile) -> Result<String, String> {
    parse_start_time_inner(pid, &read)
}

fn read_id_map_inner(base: &str, name: &str, read: &ReadFile) -> Result<u32, String> {
    let path = format!("{base}/{name}");
    let raw = read(&path).map_err(|_| unavailable())?;
    if raw.len() > 4096 {
        return Err(unavailable());
    }
    let fields = split_fields(&raw);
    if fields.len() != 3 || !id_map(&[fields.join(":")]) {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    match fields[1].parse::<u32>() {
        Ok(n) => Ok(n),
        Err(_) => Ok(0),
    }
}

/// Host UID/GID base of a single-entry id map; anything else is unsupported.
pub fn read_process_id_map(base: &str, name: &str, read: ReadFile) -> Result<u32, String> {
    read_id_map_inner(base, name, &read)
}

/// Validate one `/proc/<pid>/ns/<name>` link target, returning it verbatim.
pub fn parse_namespace_link(name: &str, v: &str) -> Result<String, String> {
    let prefix = format!("{name}:[");
    let inner = match v
        .strip_prefix(prefix.as_str())
        .and_then(|s| s.strip_suffix(']'))
    {
        Some(inner) => inner,
        None => return Err(unavailable()),
    };
    match inner.parse::<u64>() {
        Ok(n) if n != 0 && n.to_string() == inner => Ok(v.to_string()),
        _ => Err(unavailable()),
    }
}

fn read_namespace_inner(base: &str, name: &str, link: &LinkFile) -> Result<String, String> {
    let path = format!("{base}/ns/{name}");
    let v = link(&path).map_err(|_| unavailable())?;
    parse_namespace_link(name, &v)?;
    let host_path = format!("/proc/1/ns/{name}");
    match link(&host_path) {
        Ok(host) if host != v => Ok(v),
        _ => Err(ERR_UNSUPPORTED.to_string()),
    }
}

/// Namespace link of a process, refusing namespaces shared with PID 1.
pub fn read_process_namespace(base: &str, name: &str, link: LinkFile) -> Result<String, String> {
    read_namespace_inner(base, name, &link)
}

fn read_boot_id_inner(read: &ReadFile) -> Result<String, String> {
    let boot = read("/proc/sys/kernel/random/boot_id").map_err(|_| unavailable())?;
    let text = String::from_utf8_lossy(&boot);
    let trimmed = text.trim();
    if trimmed.len() != 36 || trimmed.bytes().filter(|&b| b == b'-').count() != 4 {
        return Err(unavailable());
    }
    let compact: Vec<u8> = trimmed.bytes().filter(|&b| b != b'-').collect();
    if compact.len() != 32 || !compact.iter().all(|b| b.is_ascii_hexdigit()) {
        return Err(unavailable());
    }
    Ok(trimmed.to_string())
}

/// Kernel boot ID binding the start-time token to this boot.
pub fn read_system_boot_id(read: ReadFile) -> Result<String, String> {
    read_boot_id_inner(&read)
}

/// Full native identity of a project container init process.
pub fn process_run_identity(
    pid: i64,
    read: ReadFile,
    link: LinkFile,
) -> Result<ProcessIdentity, String> {
    if pid <= 1 {
        return Err(ERR_CONFLICT.to_string());
    }
    let base = format!("/proc/{pid}");
    Ok(ProcessIdentity {
        start: parse_start_time_inner(pid, &read)?,
        uid: read_id_map_inner(&base, "uid_map", &read)?,
        gid: read_id_map_inner(&base, "gid_map", &read)?,
        userns: read_namespace_inner(&base, "user", &link)?,
        netns: read_namespace_inner(&base, "net", &link)?,
        boot: read_boot_id_inner(&read)?,
    })
}

const RUN_INSPECT_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "running",
        kind: Kind::Bool,
    },
    Spec {
        name: "pid",
        kind: Kind::Int,
    },
    Spec {
        name: "started",
        kind: Kind::Str,
    },
    Spec {
        name: "resolver",
        kind: Kind::Str,
    },
];

/// Strict-decode one run snapshot, binding it to the expected container.
pub fn decode_project_run_inspect(data: &[u8], cid: &str) -> Result<ProjectRunInspect, String> {
    let conflict = || ERR_CONFLICT.to_string();
    let value = decode_strict(data).map_err(|_| conflict())?;
    let bound =
        bind_root(&value, "projectRunInspect", RUN_INSPECT_SPECS, false).map_err(|_| conflict())?;
    let raw = ProjectRunInspect {
        id: bound.take_string("id"),
        running: bound.take_bool("running"),
        pid: bound.take_i64("pid"),
        started: bound.take_string("started"),
        resolver: bound.take_string("resolver"),
    };
    if raw.id != cid || !raw.running || raw.pid <= 1 {
        return Err(conflict());
    }
    Ok(raw)
}

/// `json.Marshal` of a run record: fixed field order, Go string escaping.
pub fn marshal_project_run(run: &ProjectRun) -> String {
    format!(
        "{{\"Target\":{{\"Project\":{},\"Container\":{},\"Run\":{}}},\"PID\":{},\"Started\":{},\"UserNS\":{},\"NetNS\":{},\"UID\":{},\"GID\":{},\"Resolver\":{}}}",
        go_escape(&run.target.project),
        go_escape(&run.target.container),
        go_escape(&run.target.run),
        run.pid,
        go_escape(&run.started),
        go_escape(&run.userns),
        go_escape(&run.netns),
        run.uid,
        run.gid,
        go_escape(&run.resolver),
    )
}

const RUN_TARGET_SPECS: &[Spec] = &[
    Spec {
        name: "Project",
        kind: Kind::Str,
    },
    Spec {
        name: "Container",
        kind: Kind::Str,
    },
    Spec {
        name: "Run",
        kind: Kind::Str,
    },
];

const PROJECT_RUN_SPECS: &[Spec] = &[
    Spec {
        name: "Target",
        kind: Kind::Object {
            go_type: "tailnet.RunTarget",
            struct_name: "RunTarget",
            specs: RUN_TARGET_SPECS,
        },
    },
    Spec {
        name: "PID",
        kind: Kind::Int,
    },
    Spec {
        name: "Started",
        kind: Kind::Str,
    },
    Spec {
        name: "UserNS",
        kind: Kind::Str,
    },
    Spec {
        name: "NetNS",
        kind: Kind::Str,
    },
    Spec {
        name: "UID",
        kind: Kind::U32,
    },
    Spec {
        name: "GID",
        kind: Kind::U32,
    },
    Spec {
        name: "Resolver",
        kind: Kind::Str,
    },
];

/// Strict-decode a persisted run record (caller validates identity fields).
pub fn decode_project_run(data: &[u8]) -> Result<ProjectRun, String> {
    let value = decode_strict(data).map_err(|_| unavailable())?;
    let bound =
        bind_root(&value, "projectRun", PROJECT_RUN_SPECS, false).map_err(|_| unavailable())?;
    let target = bound.take_map("Target");
    Ok(ProjectRun {
        target: RunTarget {
            project: target.take_string("Project"),
            container: target.take_string("Container"),
            run: target.take_string("Run"),
        },
        pid: bound.take_i64("PID"),
        started: bound.take_string("Started"),
        userns: bound.take_string("UserNS"),
        netns: bound.take_string("NetNS"),
        uid: bound.take_u32("UID"),
        gid: bound.take_u32("GID"),
        resolver: bound.take_string("Resolver"),
    })
}

/// Bind an admitted snapshot into a run record; the run token hashes the
/// container, start time, PID and full native identity.
pub fn assemble_project_run(
    id: &str,
    cid: &str,
    resolver: &str,
    raw: &ProjectRunInspect,
    identity: &ProcessIdentity,
) -> ProjectRun {
    let payload = format!(
        "{{\"CID\":{},\"Started\":{},\"PID\":{},\"Identity\":{{\"Start\":{},\"UserNS\":{},\"NetNS\":{},\"Boot\":{},\"UID\":{},\"GID\":{}}}}}",
        go_escape(cid),
        go_escape(&raw.started),
        raw.pid,
        go_escape(&identity.start),
        go_escape(&identity.userns),
        go_escape(&identity.netns),
        go_escape(&identity.boot),
        identity.uid,
        identity.gid,
    );
    let sum = crate::sha256::digest(payload.as_bytes());
    ProjectRun {
        target: RunTarget {
            project: id.to_string(),
            container: cid.to_string(),
            run: crate::sha256::hex_lower(&sum),
        },
        pid: raw.pid,
        started: raw.started.clone(),
        userns: identity.userns.clone(),
        netns: identity.netns.clone(),
        uid: identity.uid,
        gid: identity.gid,
        resolver: resolver.to_string(),
    }
}

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
/// Only the selected native rootful storage resolver is considered.
pub fn admit_project_run_snapshot(
    cid: &str,
    raw: &ProjectRunInspect,
) -> Result<(String, ProcessIdentity), String> {
    match parse_rfc3339_nano(&raw.started) {
        Some(false) => {}
        _ => return Err(unavailable()),
    }
    let resolver =
        format!("/var/lib/containers/storage/overlay-containers/{cid}/userdata/resolv.conf");
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
pub fn companion_create_args(run: &ProjectRun, image: &str) -> Result<Vec<String>, String> {
    if !valid_project_id(&run.target.project)
        || !valid_container_id(&run.target.container)
        || !valid_container_id(&run.target.run)
        || !image.starts_with("sha256:")
        || !valid_image_id(image)
        || run.uid == 0
        || run.gid == 0
    {
        return Err(ERR_INVALID.to_string());
    }
    let base = path_join(&[
        "/run/soda-tailnet",
        run.target.project.as_str(),
        run.target.run.as_str(),
    ]);
    Ok(vec![
        "--remote=false".to_string(),
        "create".to_string(),
        "--name".to_string(),
        format!("soda-tailnet-{}-{}", run.target.project, run.target.run),
        "--label".to_string(),
        format!("org.soda.tailnet.project={}", run.target.project),
        "--label".to_string(),
        format!("org.soda.tailnet.run={}", run.target.run),
        "--label".to_string(),
        format!("org.soda.tailnet.parent={}", run.target.container),
        format!("--userns=container:{}", run.target.container),
        format!("--network=container:{}", run.target.container),
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

/// Exactly one `0:<base>:262144` entry with a positive base that cannot
/// overflow the 32-bit range.
pub fn id_map(values: &[String]) -> bool {
    if values.len() != 1 {
        return false;
    }
    let parts: Vec<&str> = values[0].split(':').collect();
    if parts.len() != 3 || parts[0] != "0" || parts[2] != "262144" {
        return false;
    }
    matches!(parts[1].parse::<u32>(), Ok(base) if base.to_string() == parts[1]
                && base > 0
                && u64::from(base) + 262144 <= 4294967295)
}

/// Terminal container isolation: owned ID, matching project, unprivileged,
/// private userns and exact single-entry ID maps.
pub fn project_isolation(v: &ProjectInspection, id: &str) -> bool {
    valid_container_id(&v.id)
        && v.project == id
        && !v.privileged
        && v.userns == "private"
        && id_map(&v.uid_map)
        && id_map(&v.gid_map)
}

const INSPECTION_MAPPING_SPECS: &[Spec] = &[
    Spec {
        name: "UidMap",
        kind: Kind::StrList,
    },
    Spec {
        name: "GidMap",
        kind: Kind::StrList,
    },
];

const PROJECT_INSPECTION_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "running",
        kind: Kind::Bool,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "owner",
        kind: Kind::Str,
    },
    Spec {
        name: "privileged",
        kind: Kind::Bool,
    },
    Spec {
        name: "userns",
        kind: Kind::Str,
    },
    Spec {
        name: "mappings",
        kind: Kind::Object {
            go_type: "struct",
            struct_name: "struct",
            specs: INSPECTION_MAPPING_SPECS,
        },
    },
];

/// Inspect the terminal container of one project by identity.
pub fn inspect_project(
    exec: &dyn Executor,
    id: &str,
    deadline: Instant,
) -> Result<ProjectInspection, String> {
    if !valid_project_id(id) {
        return Err("invalid project".to_string());
    }
    let target = format!("soda-{id}");
    let data = exec
        .run(
            &[],
            "/usr/bin/podman",
            &[
                "--remote=false",
                "inspect",
                "--format",
                PROJECT_INSPECT_FORMAT,
                &target,
            ],
            deadline,
        )
        .map_err(|_| "terminal inspection unavailable".to_string())?;
    if data.len() > 4096 {
        return Err("terminal inspection unavailable".to_string());
    }
    let value = decode_strict(&data).map_err(|_| "invalid terminal inspection".to_string())?;
    let bound = bind_root(&value, "projectInspection", PROJECT_INSPECTION_SPECS, false)
        .map_err(|_| "invalid terminal inspection".to_string())?;
    let mappings = bound.take_map("mappings");
    let owner = bound.take_string("owner");
    let owner_ok = matches!(parse_go_int64(&owner), Some(n) if n > 0);
    let out = ProjectInspection {
        id: bound.take_string("id"),
        running: bound.take_bool("running"),
        project: bound.take_string("project"),
        owner,
        privileged: bound.take_bool("privileged"),
        userns: bound.take_string("userns"),
        uid_map: mappings.take_str_list("UidMap"),
        gid_map: mappings.take_str_list("GidMap"),
    };
    if !owner_ok || !project_isolation(&out, id) {
        return Err("terminal target not ready or isolated".to_string());
    }
    Ok(out)
}

/// Bound container ID of one project, optionally requiring it running.
pub fn project_container(
    exec: &dyn Executor,
    id: &str,
    require_running: bool,
    deadline: Instant,
) -> Result<String, String> {
    let v = inspect_project(exec, id, deadline)?;
    if require_running && !v.running {
        return Err("terminal target not ready or isolated".to_string());
    }
    Ok(v.id)
}

/// Running state of one project's terminal container.
pub fn project_running(exec: &dyn Executor, id: &str, deadline: Instant) -> Result<bool, String> {
    Ok(inspect_project(exec, id, deadline)?.running)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::RefCell;

    fn stat_bytes(mode: &str) -> Vec<u8> {
        let mut fields: Vec<String> = "S 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 456 1"
            .split_whitespace()
            .map(str::to_string)
            .collect();
        if mode == "dead" {
            fields[0] = "Z".to_string();
        }
        if mode == "missing start" {
            fields[19] = "0".to_string();
        }
        let pid = if mode == "wrong pid" { "124" } else { "123" };
        format!("{pid} (comm with ) spaces) {}", fields.join(" ")).into_bytes()
    }

    fn mode_read(mode: &'static str) -> ReadFile {
        Box::new(move |path: &str| -> Result<Vec<u8>, String> {
            match path {
                "/proc/123/stat" => Ok(stat_bytes(mode)),
                "/proc/123/uid_map" | "/proc/123/gid_map" => {
                    let mut data = "0 524288 262144\n".to_string();
                    if mode == "extra map" {
                        data.push_str("300000 900000 1\n");
                    }
                    if mode == "root map" {
                        data = "0 0 262144\n".to_string();
                    }
                    Ok(data.into_bytes())
                }
                "/proc/sys/kernel/random/boot_id" => {
                    if mode == "bad boot" {
                        return Ok(b"bad".to_vec());
                    }
                    Ok(b"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee\n".to_vec())
                }
                _ => Err("unexpected proc input".to_string()),
            }
        })
    }

    fn mode_link(mode: &'static str) -> LinkFile {
        Box::new(move |path: &str| -> Result<String, String> {
            if mode == "missing namespace" {
                return Err("gone".to_string());
            }
            let kind = if path.ends_with("/net") {
                "net"
            } else {
                "user"
            };
            let mut inode = "11";
            if path.contains("/123/") {
                inode = "22";
            }
            if mode.strip_prefix("host ") == Some(kind) {
                inode = "11";
            }
            Ok(format!("{kind}:[{inode}]"))
        })
    }

    #[test]
    fn process_identity_matrix() {
        for mode in [
            "valid",
            "host user",
            "host net",
            "extra map",
            "root map",
            "wrong pid",
            "dead",
            "missing start",
            "bad boot",
            "missing namespace",
        ] {
            let out = process_run_identity(123, mode_read(mode), mode_link(mode));
            if mode == "valid" {
                let identity = out.expect("valid identity must admit");
                assert_eq!(identity.uid, 524288, "{mode}");
                assert_eq!(identity.gid, 524288, "{mode}");
                assert_eq!(identity.start, "456", "{mode}");
                assert_eq!(identity.userns, "user:[22]", "{mode}");
                assert_eq!(identity.netns, "net:[22]", "{mode}");
                assert_eq!(
                    identity.boot, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
                    "{mode}"
                );
            } else {
                assert!(out.is_err(), "{mode}: unsafe identity accepted");
            }
        }
    }

    #[test]
    fn process_identity_rejects_non_root_pid() {
        let out = process_run_identity(1, mode_read("valid"), mode_link("valid"));
        assert_eq!(out.unwrap_err(), ERR_CONFLICT);
    }

    fn fixture_run() -> ProjectRun {
        ProjectRun {
            target: RunTarget {
                project: format!("p{}", "a".repeat(24)),
                container: "b".repeat(64),
                run: "c".repeat(64),
            },
            uid: 524288,
            gid: 524288,
            ..Default::default()
        }
    }

    #[test]
    fn companion_recipe_keeps_fixed_namespaces() {
        let run = fixture_run();
        let image = format!("sha256:{}", "d".repeat(64));
        let args = companion_create_args(&run, &image).expect("recipe must build");
        let joined = args.join(" ");
        for want in [
            format!("--userns=container:{}", run.target.container),
            format!("--network=container:{}", run.target.container),
            "--pid=private".to_string(),
            "--ipc=private".to_string(),
            "--uts=private".to_string(),
            "--cgroupns=private".to_string(),
            "--cap-drop=ALL".to_string(),
            "--cap-add=NET_ADMIN".to_string(),
            "--device=/dev/net/tun".to_string(),
            "--no-hosts".to_string(),
            "--log-driver=none".to_string(),
            "--pull=never".to_string(),
            image.clone(),
            "--entrypoint=/usr/local/bin/tailscaled".to_string(),
            "--state=mem:".to_string(),
            "--no-logs-no-support".to_string(),
        ] {
            assert!(
                joined.contains(want.as_str()),
                "missing fixed recipe argument {want}"
            );
        }
        for forbidden in [
            "--privileged",
            "--rm",
            "--replace",
            "--env",
            "SYS_MODULE",
            ":U",
            "--network=host",
            "--pid=host",
            "--userns=host",
            "/var/lib/soda-tailnet",
            "/run/podman",
            "/proc/",
            "tskey",
            "client_secret",
        ] {
            assert!(
                !joined.contains(forbidden),
                "unsafe companion argument {forbidden}"
            );
        }
        let mut count = 0;
        for (i, arg) in args.iter().enumerate() {
            if arg == "--volume" {
                count += 1;
                let mount = &args[i + 1];
                let prefix = format!(
                    "/run/soda-tailnet/{}/{}/",
                    run.target.project, run.target.run
                );
                assert!(mount.starts_with(prefix.as_str()), "non-run mount {mount}");
            }
        }
        assert_eq!(count, 2, "unexpected mount count");
        for bad in [
            "latest".to_string(),
            "docker.io/tailscale/tailscale:latest".to_string(),
            "sha256:bad".to_string(),
            "--privileged".to_string(),
        ] {
            assert!(
                companion_create_args(&run, &bad).is_err(),
                "mutable/caller image accepted: {bad}"
            );
        }
        let mut bad_target = run.clone();
        bad_target.target.container = "../other".to_string();
        assert!(
            companion_create_args(&bad_target, &image).is_err(),
            "invalid namespace target accepted"
        );
        let mut no_ids = run.clone();
        no_ids.uid = 0;
        assert!(companion_create_args(&no_ids, &image).is_err());
    }

    #[test]
    fn stat_fields_reject_bad_shapes() {
        assert!(parse_stat_fields(123, &vec![b'x'; 8193]).is_err());
        assert!(parse_stat_fields(123, b"123 (a)").is_err());
        assert!(parse_stat_fields(123, b"124 (a) S 1 2").is_err());
        assert!(parse_stat_fields(123, &stat_bytes("dead")).is_err());
        let fields = parse_stat_fields(123, &stat_bytes("valid")).unwrap();
        assert!(fields.len() >= 20);
        assert_eq!(fields[19], "456");
    }

    #[test]
    fn namespace_link_shapes() {
        assert_eq!(
            parse_namespace_link("user", "user:[22]").unwrap(),
            "user:[22]"
        );
        for bad in [
            "user:22",
            "user:[0]",
            "user:[01]",
            "net:[22]",
            "user:[22",
            "user:[]",
        ] {
            assert!(parse_namespace_link("user", bad).is_err(), "{bad}");
        }
    }

    #[test]
    fn id_map_edges() {
        assert!(id_map(&["0:524288:262144".to_string()]));
        assert!(id_map(&["0:4294705151:262144".to_string()]));
        for bad in [
            vec![],
            vec!["0:524288:262144".to_string(), "0:1:2".to_string()],
            vec!["0:0:262144".to_string()],
            vec!["0:4294705152:262144".to_string()],
            vec!["0:4294967295:262144".to_string()],
            vec!["1:524288:262144".to_string()],
            vec!["0:524288:1".to_string()],
            vec!["0:+524288:262144".to_string()],
            vec!["0:0524288:262144".to_string()],
            vec!["nope".to_string()],
        ] {
            assert!(!id_map(&bad), "{bad:?}");
        }
    }

    #[test]
    fn marshal_and_decode_round_trip() {
        let run = ProjectRun {
            target: RunTarget {
                project: "p0123456789abcdef01234567".to_string(),
                container: "b".repeat(64),
                run: "c".repeat(64),
            },
            pid: 123,
            started: "2026-01-02T03:04:05.678901234Z".to_string(),
            userns: "user:[22]".to_string(),
            netns: "net:[22]".to_string(),
            uid: 524288,
            gid: 524289,
            resolver: "/run/x".to_string(),
        };
        let text = marshal_project_run(&run);
        assert!(text.starts_with("{\"Target\":{\"Project\":\"p0123456789abcdef01234567\""));
        assert!(text.contains("\"PID\":123"));
        assert!(text.contains("\"UID\":524288"));
        assert!(text.ends_with("\"Resolver\":\"/run/x\"}"));
        let back = decode_project_run(text.as_bytes()).unwrap();
        assert_eq!(back, run);
        assert!(decode_project_run(b"{\"Target\":{},\"PID\":1,\"bogus\":true}").is_err());
    }

    #[test]
    fn decode_run_inspect_binds_container() {
        let good = br#"{"id":"abc","running":true,"pid":123,"started":"s","resolver":"r"}"#;
        let raw = decode_project_run_inspect(good, "abc").unwrap();
        assert_eq!(raw.pid, 123);
        assert!(raw.running);
        // Missing optional snapshot fields still bind (admission checks them).
        let sparse = br#"{"id":"abc","running":true,"pid":123}"#;
        assert!(decode_project_run_inspect(sparse, "abc").is_ok());
        for bad in [
            br#"{"id":"other","running":true,"pid":123,"started":"s","resolver":"r"}"#.as_slice(),
            br#"{"id":"abc","running":false,"pid":123,"started":"s","resolver":"r"}"#.as_slice(),
            br#"{"id":"abc","running":true,"pid":1,"started":"s","resolver":"r"}"#.as_slice(),
            br#"{"id":"abc","running":true,"pid":1}"#.as_slice(),
            br#"[]"#.as_slice(),
            br#"{"id":"abc","running":true,"pid":123,"started":"s","resolver":"r","extra":1}"#
                .as_slice(),
        ] {
            assert!(decode_project_run_inspect(bad, "abc").is_err());
        }
    }

    #[test]
    fn admit_snapshot_rejects_before_touching_proc() {
        let cid = "c".repeat(64);
        let resolver =
            format!("/var/lib/containers/storage/overlay-containers/{cid}/userdata/resolv.conf");
        let raw = ProjectRunInspect {
            id: cid.clone(),
            running: true,
            pid: 123,
            started: "not-a-time".to_string(),
            resolver: resolver.clone(),
        };
        assert_eq!(
            admit_project_run_snapshot(&cid, &raw).unwrap_err(),
            ERR_UNAVAILABLE
        );
        let raw = ProjectRunInspect {
            started: "2026-01-02T03:04:05Z".to_string(),
            resolver: "/etc/resolv.conf".to_string(),
            ..raw
        };
        assert_eq!(
            admit_project_run_snapshot(&cid, &raw).unwrap_err(),
            ERR_UNSUPPORTED
        );
    }

    struct FakeExec {
        out: Vec<u8>,
        err: Option<String>,
        seen: RefCell<Vec<String>>,
    }

    impl Executor for FakeExec {
        fn run(
            &self,
            _stdin: &[u8],
            cmd: &str,
            args: &[&str],
            _deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            let mut seen = self.seen.borrow_mut();
            seen.push(cmd.to_string());
            seen.extend(args.iter().map(|s| s.to_string()));
            match &self.err {
                Some(e) => Err(e.clone()),
                None => Ok(self.out.clone()),
            }
        }
    }

    fn inspection_json(
        id: &str,
        cid: &str,
        running: bool,
        owner: &str,
        privileged: bool,
    ) -> Vec<u8> {
        format!(
            "{{\"id\":{cid:?},\"running\":{running},\"project\":{id:?},\"owner\":{owner:?},\"privileged\":{privileged},\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:524288:262144\"],\"GidMap\":[\"0:524288:262144\"]}}}}"
        )
        .into_bytes()
    }

    #[test]
    fn inspect_project_argv_and_gates() {
        let id = "p0123456789abcdef01234567";
        let cid = "b".repeat(64);
        let deadline = Instant::now() + std::time::Duration::from_secs(5);
        let exec = FakeExec {
            out: inspection_json(id, &cid, true, "1000", false),
            err: None,
            seen: RefCell::new(Vec::new()),
        };
        let v = inspect_project(&exec, id, deadline).unwrap();
        assert_eq!(v.id, cid);
        assert!(v.running);
        assert_eq!(project_container(&exec, id, true, deadline).unwrap(), cid);
        assert!(project_running(&exec, id, deadline).unwrap());
        let seen: Vec<String> = exec.seen.borrow().clone();
        let expected = vec![
            "/usr/bin/podman".to_string(),
            "--remote=false".to_string(),
            "inspect".to_string(),
            "--format".to_string(),
            PROJECT_INSPECT_FORMAT.to_string(),
            format!("soda-{id}"),
        ];
        assert_eq!(seen[..6].to_vec(), expected);
        assert_eq!(
            inspect_project(&exec, "bogus", deadline).unwrap_err(),
            "invalid project"
        );
        let exec = FakeExec {
            out: inspection_json(id, &cid, false, "1000", false),
            err: None,
            seen: RefCell::new(Vec::new()),
        };
        assert_eq!(
            project_container(&exec, id, true, deadline).unwrap_err(),
            "terminal target not ready or isolated"
        );
        assert_eq!(project_container(&exec, id, false, deadline).unwrap(), cid);
        for (owner, privileged) in [("0", false), ("-3", false), ("NaN", false), ("1000", true)] {
            let exec = FakeExec {
                out: inspection_json(id, &cid, true, owner, privileged),
                err: None,
                seen: RefCell::new(Vec::new()),
            };
            assert_eq!(
                inspect_project(&exec, id, deadline).unwrap_err(),
                "terminal target not ready or isolated",
                "{owner}/{privileged}"
            );
        }
        let exec = FakeExec {
            out: b"{}".to_vec(),
            err: Some("boom".to_string()),
            seen: RefCell::new(Vec::new()),
        };
        assert_eq!(
            inspect_project(&exec, id, deadline).unwrap_err(),
            "terminal inspection unavailable"
        );
        let exec = FakeExec {
            out: b"[]".to_vec(),
            err: None,
            seen: RefCell::new(Vec::new()),
        };
        assert_eq!(
            inspect_project(&exec, id, deadline).unwrap_err(),
            "invalid terminal inspection"
        );
    }
}
