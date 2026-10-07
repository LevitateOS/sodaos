use std::time::Instant;

use crate::json::{self, parse_go_int64};
use crate::project::GoStringList;
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};

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

#[derive(Default)]
struct MappingWire {
    uid_map: Option<GoStringList>,
    gid_map: Option<GoStringList>,
}

impl<'de> Deserialize<'de> for MappingWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = MappingWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("container mapping")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = MappingWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    match k.to_ascii_lowercase().as_str() {
                        "uidmap" => {
                            if let Some(v) = map.next_value::<Option<GoStringList>>()? {
                                o.uid_map = Some(v)
                            }
                        }
                        "gidmap" => {
                            if let Some(v) = map.next_value::<Option<GoStringList>>()? {
                                o.gid_map = Some(v)
                            }
                        }
                        _ => return Err(de::Error::unknown_field(&k, &["UidMap", "GidMap"])),
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct InspectionWire {
    id: Option<String>,
    running: Option<bool>,
    project: Option<String>,
    owner: Option<String>,
    privileged: Option<bool>,
    userns: Option<String>,
    mappings: Option<MappingWire>,
}
impl<'de> Deserialize<'de> for InspectionWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = InspectionWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("terminal project inspection")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = InspectionWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    match k.to_ascii_lowercase().as_str() {
                        "id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.id = Some(v)
                            }
                        }
                        "running" => {
                            if let Some(v) = map.next_value::<Option<bool>>()? {
                                o.running = Some(v)
                            }
                        }
                        "project" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.project = Some(v)
                            }
                        }
                        "owner" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.owner = Some(v)
                            }
                        }
                        "privileged" => {
                            if let Some(v) = map.next_value::<Option<bool>>()? {
                                o.privileged = Some(v)
                            }
                        }
                        "userns" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.userns = Some(v)
                            }
                        }
                        "mappings" => {
                            if let Some(v) = map.next_value::<Option<MappingWire>>()? {
                                o.mappings = Some(v)
                            }
                        }
                        _ => {
                            return Err(de::Error::unknown_field(
                                &k,
                                &[
                                    "id",
                                    "running",
                                    "project",
                                    "owner",
                                    "privileged",
                                    "userns",
                                    "mappings",
                                ],
                            ))
                        }
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

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
    let bound: InspectionWire =
        json::decode_strict_as(&data).map_err(|_| "invalid terminal inspection".to_string())?;
    let mappings = bound.mappings.unwrap_or_default();
    let owner = bound.owner.unwrap_or_default();
    let owner_ok = matches!(parse_go_int64(&owner), Some(n) if n > 0);
    let out = ProjectInspection {
        id: bound.id.unwrap_or_default(),
        running: bound.running.unwrap_or_default(),
        project: bound.project.unwrap_or_default(),
        owner,
        privileged: bound.privileged.unwrap_or_default(),
        userns: bound.userns.unwrap_or_default(),
        uid_map: mappings.uid_map.map(|v| v.0).unwrap_or_default(),
        gid_map: mappings.gid_map.map(|v| v.0).unwrap_or_default(),
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
