use std::collections::HashMap;
use std::time::Instant;

use super::profile;
use super::Executor;
use crate::domain;
use crate::json;
use crate::net;
use serde::de::{self, MapAccess, SeqAccess, Visitor};
use serde::Deserialize;
use std::fmt;

pub(crate) struct GoStringList(pub(crate) Vec<String>);

impl<'de> Deserialize<'de> for GoStringList {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ListVisitor;
        impl<'de> Visitor<'de> for ListVisitor {
            type Value = GoStringList;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an array of strings")
            }
            fn visit_unit<E>(self) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(GoStringList(Vec::new()))
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                let mut out = Vec::new();
                while let Some(value) = seq.next_element::<Option<String>>()? {
                    out.push(value.unwrap_or_default());
                }
                Ok(GoStringList(out))
            }
        }
        deserializer.deserialize_any(ListVisitor)
    }
}

#[derive(Default)]
pub(crate) struct ProjectIdMappings {
    pub(crate) uid_map: Vec<String>,
    pub(crate) gid_map: Vec<String>,
}

impl<'de> Deserialize<'de> for ProjectIdMappings {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct MappingsVisitor;
        impl<'de> Visitor<'de> for MappingsVisitor {
            type Value = ProjectIdMappings;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("ID mappings object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = ProjectIdMappings::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("UidMap") {
                        if let Some(v) = map.next_value::<Option<GoStringList>>()? {
                            out.uid_map = v.0;
                        }
                    } else if key.eq_ignore_ascii_case("GidMap") {
                        if let Some(v) = map.next_value::<Option<GoStringList>>()? {
                            out.gid_map = v.0;
                        }
                    } else {
                        return Err(de::Error::unknown_field(&key, &["UidMap", "GidMap"]));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(MappingsVisitor)
    }
}

#[derive(Default)]
pub(crate) struct ProjectContainerInspection {
    pub(crate) id: String,
    pub(crate) running: bool,
    pub(crate) project: String,
    pub(crate) owner: String,
    pub(crate) privileged: bool,
    pub(crate) userns: String,
    pub(crate) mappings: ProjectIdMappings,
}

impl<'de> Deserialize<'de> for ProjectContainerInspection {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct InspectionVisitor;
        impl<'de> Visitor<'de> for InspectionVisitor {
            type Value = ProjectContainerInspection;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("project container inspection object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = ProjectContainerInspection::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.id = v;
                        }
                    } else if key.eq_ignore_ascii_case("running") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.running = v;
                        }
                    } else if key.eq_ignore_ascii_case("project") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.project = v;
                        }
                    } else if key.eq_ignore_ascii_case("owner") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.owner = v;
                        }
                    } else if key.eq_ignore_ascii_case("privileged") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.privileged = v;
                        }
                    } else if key.eq_ignore_ascii_case("userns") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.userns = v;
                        }
                    } else if key.eq_ignore_ascii_case("mappings") {
                        if let Some(v) = map.next_value::<Option<ProjectIdMappings>>()? {
                            out.mappings = v;
                        }
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &[
                                "id",
                                "running",
                                "project",
                                "owner",
                                "privileged",
                                "userns",
                                "mappings",
                            ],
                        ));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(InspectionVisitor)
    }
}

pub(crate) fn decode_project_container_inspection(
    body: &[u8],
) -> Result<ProjectContainerInspection, String> {
    json::decode_strict_as(body).map_err(|e| e.0)
}

#[derive(Default)]
struct ImageConfig {
    labels: Option<HashMap<String, Option<String>>>,
}
#[derive(Default)]
struct ContainerState {
    running: Option<bool>,
}
#[derive(Default)]
struct Network {
    ip_address: Option<String>,
}
#[derive(Default)]
struct NetworkSettings {
    networks: Option<HashMap<String, Option<Network>>>,
}
#[derive(Default)]
struct ContainerInspect {
    image: Option<String>,
    config: Option<ImageConfig>,
    state: Option<ContainerState>,
    network_settings: Option<NetworkSettings>,
}

macro_rules! tolerant_object {
    ($ty:ident, {$($field:ident => $name:literal : $value:ty),+ $(,)?}) => {
        impl<'de> Deserialize<'de> for $ty {
            fn deserialize<D>(deserializer: D) -> Result<Self, D::Error> where D: serde::Deserializer<'de> {
                struct ObjectVisitor;
                impl<'de> Visitor<'de> for ObjectVisitor {
                    type Value = $ty;
                    fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str("an object") }
                    fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error> where A: MapAccess<'de> {
                        let mut out = <$ty>::default();
                        while let Some(key) = map.next_key::<String>()? {
                            let mut known = false;
                            $(if key.eq_ignore_ascii_case($name) {
                                if let Some(value) = map.next_value::<Option<$value>>()? { out.$field = Some(value); }
                                known = true;
                            })+
                            if !known { map.next_value::<serde::de::IgnoredAny>()?; }
                        }
                        Ok(out)
                    }
                }
                deserializer.deserialize_map(ObjectVisitor)
            }
        }
    };
}

tolerant_object!(ImageConfig, { labels => "Labels": HashMap<String, Option<String>> });
tolerant_object!(ContainerState, { running => "Running": bool });
tolerant_object!(Network, { ip_address => "IPAddress": String });
tolerant_object!(NetworkSettings, { networks => "Networks": HashMap<String, Option<Network>> });
tolerant_object!(ContainerInspect, {
    image => "Image": String,
    config => "Config": ImageConfig,
    state => "State": ContainerState,
    network_settings => "NetworkSettings": NetworkSettings,
});

#[derive(Default)]
struct ProjectInspectArray(Vec<ContainerInspect>);

impl<'de> Deserialize<'de> for ProjectInspectArray {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ArrayVisitor;
        impl<'de> Visitor<'de> for ArrayVisitor {
            type Value = ProjectInspectArray;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an array of container inspections")
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                let mut out = Vec::new();
                while let Some(item) = seq.next_element::<ContainerInspect>()? {
                    out.push(item);
                }
                Ok(ProjectInspectArray(out))
            }
        }
        deserializer.deserialize_seq(ArrayVisitor)
    }
}

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
        let inspections: ProjectInspectArray =
            json::decode_tolerant_as(&out).map_err(|_| "invalid native inspection".to_string())?;
        if inspections.0.len() != 1 {
            return Err("invalid native inspection".to_string());
        }
        let item = &inspections.0[0];
        let image = item.image.as_deref().unwrap_or("").to_string();
        if domain::valid_image_ref(&image) {
            env.image = format!("sha256:{}", image.trim_start_matches("sha256:"));
        }
        let labels: HashMap<String, String> = item
            .config
            .as_ref()
            .and_then(|config| config.labels.as_ref())
            .map(|values| {
                values
                    .iter()
                    .filter_map(|(key, value)| value.clone().map(|v| (key.clone(), v)))
                    .collect()
            })
            .unwrap_or_default();
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
        env.running = item
            .state
            .as_ref()
            .and_then(|state| state.running)
            .unwrap_or(false);
        let mut ip = String::new();
        if let Some(entry) = item
            .network_settings
            .as_ref()
            .and_then(|settings| settings.networks.as_ref())
            .and_then(|networks| networks.get(&self.config.network))
            .and_then(Option::as_ref)
        {
            ip = entry.ip_address.clone().unwrap_or_default();
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
        let inspection = decode_project_container_inspection(&data)
            .map_err(|_| "invalid terminal inspection".to_string())?;
        let cid = inspection.id;
        let running = inspection.running;
        let project = inspection.project;
        let owner = inspection.owner;
        let privileged = inspection.privileged;
        let userns = inspection.userns;
        let uid_map = inspection.mappings.uid_map;
        let gid_map = inspection.mappings.gid_map;
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
