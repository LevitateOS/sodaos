use crate::json::{self, SignedInteger};
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};

use crate::tailnet_domain::{go_escape, RunTarget, ERR_CONFLICT};

use super::{unavailable, ProcessIdentity, ProjectRun, ProjectRunInspect};

#[derive(Default)]
struct RunInspectWire {
    id: Option<String>,
    running: Option<bool>,
    pid: Option<SignedInteger>,
    started: Option<String>,
    resolver: Option<String>,
}

impl<'de> Deserialize<'de> for RunInspectWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RunInspectWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("project run inspection")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = RunInspectWire::default();
                while let Some(key) = map.next_key::<String>()? {
                    match key.to_ascii_lowercase().as_str() {
                        "id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.id = Some(v);
                            }
                        }
                        "running" => {
                            if let Some(v) = map.next_value::<Option<bool>>()? {
                                out.running = Some(v);
                            }
                        }
                        "pid" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                out.pid = Some(v);
                            }
                        }
                        "started" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.started = Some(v);
                            }
                        }
                        "resolver" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.resolver = Some(v);
                            }
                        }
                        _ => {
                            return Err(de::Error::unknown_field(
                                &key,
                                &["id", "running", "pid", "started", "resolver"],
                            ))
                        }
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(V)
    }
}

/// Strict-decode one run snapshot, binding it to the expected container.
pub fn decode_project_run_inspect(data: &[u8], cid: &str) -> Result<ProjectRunInspect, String> {
    let conflict = || ERR_CONFLICT.to_string();
    let bound: RunInspectWire = json::decode_strict_as(data).map_err(|_| conflict())?;
    let raw = ProjectRunInspect {
        id: bound.id.unwrap_or_default(),
        running: bound.running.unwrap_or_default(),
        pid: bound.pid.map(Into::into).unwrap_or_default(),
        started: bound.started.unwrap_or_default(),
        resolver: bound.resolver.unwrap_or_default(),
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

#[derive(Default)]
struct RunTargetWire {
    project: Option<String>,
    container: Option<String>,
    run: Option<String>,
}

impl<'de> Deserialize<'de> for RunTargetWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RunTargetWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("run target")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = RunTargetWire::default();
                while let Some(key) = map.next_key::<String>()? {
                    match key.to_ascii_lowercase().as_str() {
                        "project" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.project = Some(v);
                            }
                        }
                        "container" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.container = Some(v);
                            }
                        }
                        "run" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.run = Some(v);
                            }
                        }
                        _ => {
                            return Err(de::Error::unknown_field(
                                &key,
                                &["Project", "Container", "Run"],
                            ))
                        }
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(V)
    }
}

#[derive(Default)]
struct ProjectRunWire {
    target: Option<RunTargetWire>,
    pid: Option<SignedInteger>,
    started: Option<String>,
    userns: Option<String>,
    netns: Option<String>,
    uid: Option<u32>,
    gid: Option<u32>,
    resolver: Option<String>,
}

impl<'de> Deserialize<'de> for ProjectRunWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ProjectRunWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("project run record")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = ProjectRunWire::default();
                while let Some(key) = map.next_key::<String>()? {
                    match key.to_ascii_lowercase().as_str() {
                        "target" => {
                            if let Some(v) = map.next_value::<Option<RunTargetWire>>()? {
                                out.target = Some(v);
                            }
                        }
                        "pid" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                out.pid = Some(v);
                            }
                        }
                        "started" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.started = Some(v);
                            }
                        }
                        "userns" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.userns = Some(v);
                            }
                        }
                        "netns" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.netns = Some(v);
                            }
                        }
                        "uid" => {
                            if let Some(v) = map.next_value::<Option<u32>>()? {
                                out.uid = Some(v);
                            }
                        }
                        "gid" => {
                            if let Some(v) = map.next_value::<Option<u32>>()? {
                                out.gid = Some(v);
                            }
                        }
                        "resolver" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.resolver = Some(v);
                            }
                        }
                        _ => {
                            return Err(de::Error::unknown_field(
                                &key,
                                &[
                                    "Target", "PID", "Started", "UserNS", "NetNS", "UID", "GID",
                                    "Resolver",
                                ],
                            ))
                        }
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(V)
    }
}

/// Strict-decode a persisted run record (caller validates identity fields).
pub fn decode_project_run(data: &[u8]) -> Result<ProjectRun, String> {
    let bound: ProjectRunWire = json::decode_strict_as(data).map_err(|_| unavailable())?;
    let target = bound.target.unwrap_or_default();
    Ok(ProjectRun {
        target: RunTarget {
            project: target.project.unwrap_or_default(),
            container: target.container.unwrap_or_default(),
            run: target.run.unwrap_or_default(),
        },
        pid: bound.pid.map(Into::into).unwrap_or_default(),
        started: bound.started.unwrap_or_default(),
        userns: bound.userns.unwrap_or_default(),
        netns: bound.netns.unwrap_or_default(),
        uid: bound.uid.unwrap_or_default(),
        gid: bound.gid.unwrap_or_default(),
        resolver: bound.resolver.unwrap_or_default(),
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
