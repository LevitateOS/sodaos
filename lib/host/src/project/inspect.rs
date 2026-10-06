use std::collections::HashMap;
use std::time::Instant;

use super::profile;
use super::Executor;
use crate::domain;
use crate::json::{self, Value};
use crate::net;

pub(crate) const INSPECTION_SPECS: &[json::Spec] = &[
    json::Spec {
        name: "id",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "running",
        kind: json::Kind::Bool,
    },
    json::Spec {
        name: "project",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "owner",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "privileged",
        kind: json::Kind::Bool,
    },
    json::Spec {
        name: "userns",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "mappings",
        kind: json::Kind::Object {
            go_type: "struct",
            struct_name: "struct",
            specs: &[
                json::Spec {
                    name: "UidMap",
                    kind: json::Kind::StrList,
                },
                json::Spec {
                    name: "GidMap",
                    kind: json::Kind::StrList,
                },
            ],
        },
    },
];

pub const PROJECT_INSPECT_FORMAT: &str = "{\"id\":{{json .ID}},\"running\":{{json .State.Running}},\"project\":{{json (index .Config.Labels \"org.soda.project\")}},\"owner\":{{json (index .Config.Labels \"org.soda.owner\")}},\"privileged\":{{json .HostConfig.Privileged}},\"userns\":{{json .HostConfig.UsernsMode}},\"mappings\":{{json .HostConfig.IDMappings}}}";

impl<E: Executor> super::Runtime<E> {
    /// `Inspect`: observe one project container by identity.
    pub fn inspect(
        &self,
        id: &str,
        deadline: Instant,
    ) -> Result<(domain::Environment, i64), String> {
        let mut env = domain::Environment {
            image: String::new(),
            profile: None,
            id: id.to_string(),
            ip: String::new(),
            running: false,
        };
        if !domain::valid_id(id) {
            return Err("invalid project id".to_string());
        }
        let out = self.podman(&[], &["inspect", &format!("soda-{id}")], deadline)?;
        let v = json::decode_tolerant(&out).map_err(|_| "invalid native inspection".to_string())?;
        let items = v
            .as_array()
            .filter(|a| a.len() == 1)
            .ok_or_else(|| "invalid native inspection".to_string())?;
        let item = &items[0];
        if item.as_object().is_none() {
            return Err("invalid native inspection".to_string());
        }
        let image = tolerant_str(item, "Image")?;
        if domain::valid_image_ref(&image) {
            env.image = format!("sha256:{}", image.trim_start_matches("sha256:"));
        }
        let empty = Value::Object(Vec::new());
        let config = json::tolerant_get(item, "Config").unwrap_or(&empty);
        let labels = labels_of(config)?;
        if labels
            .get("org.soda.project")
            .map(String::as_str)
            .unwrap_or("")
            != id
        {
            return Err("container is not owned by this project".to_string());
        }
        let owner: i64 = json::parse_go_int64(
            labels
                .get("org.soda.owner")
                .map(String::as_str)
                .unwrap_or(""),
        )
        .ok_or_else(|| "invalid native project owner".to_string())?;
        if owner <= 0 {
            return Err("invalid native project owner".to_string());
        }
        profile::apply_creation_profile(&mut env, &labels, &image)?;
        env.running = match json::tolerant_get(item, "State") {
            None => false,
            Some(state) => match json::tolerant_get(state, "Running") {
                None => false,
                Some(Value::Bool(b)) => *b,
                Some(_) => return Err("invalid native inspection".to_string()),
            },
        };
        let mut ip = String::new();
        if let Some(settings) = json::tolerant_get(item, "NetworkSettings") {
            if settings.as_object().is_none() {
                return Err("invalid native inspection".to_string());
            }
            let empty = Value::Object(Vec::new());
            let networks = json::tolerant_get(settings, "Networks").unwrap_or(&empty);
            if networks.as_object().is_none() {
                return Err("invalid native inspection".to_string());
            }
            if let Some(entry) = networks
                .as_object()
                .unwrap()
                .iter()
                .find(|(k, _)| *k == self.config.network)
                .map(|(_, v)| v)
            {
                if entry.is_null() {
                    // Null decodes as the zero struct: no address.
                } else if entry.as_object().is_none() {
                    return Err("invalid native inspection".to_string());
                }
                match json::tolerant_get(entry, "IPAddress") {
                    None => {}
                    Some(Value::Str(s)) => ip = s.clone(),
                    Some(_) => return Err("invalid native inspection".to_string()),
                }
            }
        }
        env.ip = ip.clone();
        net::admit_ip(&ip, &self.config.subnet).map_err(|e| {
            if e == "project IP outside configured network" {
                e
            } else {
                format!("invalid IP address {ip:?}")
            }
        })?;
        Ok((env, owner))
    }

    /// `ProjectContainer`: bind the exact isolated container identity.
    pub fn project_container(
        &self,
        id: &str,
        require_running: bool,
        deadline: Instant,
    ) -> Result<String, String> {
        if !domain::valid_id(id) {
            return Err("invalid project".to_string());
        }
        let data = self
            .podman(
                &[],
                &[
                    "--remote=false",
                    "inspect",
                    "--format",
                    PROJECT_INSPECT_FORMAT,
                    &format!("soda-{id}"),
                ],
                deadline,
            )
            .map_err(|_| "terminal inspection unavailable".to_string())?;
        if data.len() > 4096 {
            return Err("terminal inspection unavailable".to_string());
        }
        let v =
            json::decode_strict(&data).map_err(|_| "invalid terminal inspection".to_string())?;
        let m = json::bind_root(&v, "projectInspection", INSPECTION_SPECS, false)
            .map_err(|_| "invalid terminal inspection".to_string())?;
        let cid = m.take_string("id");
        let running = m.take_bool("running");
        let project = m.take_string("project");
        let owner = m.take_string("owner");
        let privileged = m.take_bool("privileged");
        let userns = m.take_string("userns");
        let mappings = m.take_map("mappings");
        let uid_map = mappings.take_str_list("UidMap");
        let gid_map = mappings.take_str_list("GidMap");
        let owner_num: i64 = json::parse_go_int64(&owner)
            .ok_or_else(|| "terminal target not ready or isolated".to_string())?;
        if owner_num <= 0 || (require_running && !running) {
            return Err("terminal target not ready or isolated".to_string());
        }
        if !domain::valid_container_id(&cid) || project != id || privileged || userns != "private" {
            return Err("terminal target not ready or isolated".to_string());
        }
        if !project_id_map(&uid_map) || !project_id_map(&gid_map) {
            return Err("terminal target not ready or isolated".to_string());
        }
        Ok(cid)
    }
}

fn tolerant_str(item: &Value, key: &str) -> Result<String, String> {
    match json::tolerant_get(item, key) {
        None => Ok(String::new()),
        Some(Value::Str(s)) => Ok(s.clone()),
        Some(_) => Err("invalid native inspection".to_string()),
    }
}

fn labels_of(config: &Value) -> Result<HashMap<String, String>, String> {
    if config.as_object().is_none() {
        return Err("invalid native inspection".to_string());
    }
    match json::tolerant_get(config, "Labels") {
        None => Ok(HashMap::new()),
        Some(Value::Object(fields)) => {
            let mut map = HashMap::new();
            for (k, val) in fields {
                match val {
                    Value::Str(s) => {
                        map.insert(k.clone(), s.clone());
                    }
                    Value::Null => {}
                    _ => return Err("invalid native inspection".to_string()),
                }
            }
            Ok(map)
        }
        Some(_) => Err("invalid native inspection".to_string()),
    }
}

/// Single 262144-ID mapping with container root shifted off host root.
/// (The Go `base+262144 <= 4294967295` bound is vacuous under uint32
/// wraparound: every canonical non-zero base passes it, so the check is
/// exactly `base > 0`.)
fn project_id_map(values: &[String]) -> bool {
    if values.len() != 1 {
        return false;
    }
    let parts: Vec<&str> = values[0].split(':').collect();
    if parts.len() != 3 || parts[0] != "0" || parts[2] != "262144" {
        return false;
    }
    matches!(parts[1].parse::<u32>(), Ok(base) if base.to_string() == parts[1] && base > 0 && u64::from(base) + 262144 <= 4294967295)
}
