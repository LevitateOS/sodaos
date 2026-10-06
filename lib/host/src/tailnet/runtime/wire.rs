use crate::json::{bind_root, decode_strict, Kind, Spec};

use crate::tailnet_domain::{go_escape, RunTarget, ERR_CONFLICT};

use super::{unavailable, ProcessIdentity, ProjectRun, ProjectRunInspect};

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
