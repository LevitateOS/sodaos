use std::time::Instant;

use crate::json::{bind_root, decode_strict, parse_go_int64, Kind, Spec};

use crate::project::{Executor, PROJECT_INSPECT_FORMAT};
use crate::tailnet_domain::{valid_container_id, valid_project_id};

use super::ProjectInspection;

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
