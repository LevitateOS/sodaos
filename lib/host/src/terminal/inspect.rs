use crate::domain;
use crate::json;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

// ---------- container inspection ----------

/// `terminalInspect`: the exact podman `--format` template.
pub const TERMINAL_INSPECT: &str = "{\"id\":{{json .ID}},\"running\":{{json .State.Running}},\"project\":{{json (index .Config.Labels \"org.soda.project\")}},\"owner\":{{json (index .Config.Labels \"org.soda.owner\")}},\"privileged\":{{json .HostConfig.Privileged}},\"userns\":{{json .HostConfig.UsernsMode}},\"mappings\":{{json .HostConfig.IDMappings}}}";

/// `terminalInspection`: strict-decoded podman inspect output.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TerminalInspection {
    pub id: String,
    pub running: bool,
    pub project: String,
    pub owner: String,
    pub privileged: bool,
    pub userns: String,
    pub uid_map: Vec<String>,
    pub gid_map: Vec<String>,
}

#[derive(Default)]
struct Mappings {
    uid_map: Vec<String>,
    gid_map: Vec<String>,
}

impl<'de> Deserialize<'de> for Mappings {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct MappingsVisitor;
        impl<'de> Visitor<'de> for MappingsVisitor {
            type Value = Mappings;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an ID mappings object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = Mappings::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("UidMap") {
                        if let Some(v) = map.next_value::<Option<Vec<Option<String>>>>()? {
                            out.uid_map = v.into_iter().map(Option::unwrap_or_default).collect();
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("GidMap") {
                        if let Some(v) = map.next_value::<Option<Vec<Option<String>>>>()? {
                            out.gid_map = v.into_iter().map(Option::unwrap_or_default).collect();
                        }
                        continue;
                    }
                    return Err(de::Error::unknown_field(&key, &["UidMap", "GidMap"]));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(MappingsVisitor)
    }
}

impl<'de> Deserialize<'de> for TerminalInspection {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct InspectionVisitor;
        impl<'de> Visitor<'de> for InspectionVisitor {
            type Value = TerminalInspection;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a terminal inspection object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = TerminalInspection::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.id = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("running") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.running = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("project") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.project = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("owner") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.owner = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("privileged") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.privileged = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("userns") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.userns = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("mappings") {
                        if let Some(v) = map.next_value::<Option<Mappings>>()? {
                            out.uid_map = v.uid_map;
                            out.gid_map = v.gid_map;
                        }
                        continue;
                    }
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
                Ok(out)
            }
        }
        deserializer.deserialize_map(InspectionVisitor)
    }
}

impl TerminalInspection {
    /// Strict decode (`strictjson.Decode` in Go).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

/// `terminalIDMap`: exactly one `0:<base>:262144` mapping with host root
/// shifted away from container root.
pub fn terminal_id_map(values: &[String]) -> bool {
    if values.len() != 1 {
        return false;
    }
    let parts: Vec<&str> = values[0].split(':').collect();
    if parts.len() != 3 || parts[0] != "0" || parts[2] != "262144" {
        return false;
    }
    let Some(base) = parse_go_uint(parts[1], 32) else {
        return false;
    };
    // Canonical re-format check rejects leading zeros, as in Go.
    base.to_string() == parts[1] && base > 0 && base + 262144 <= 4294967295
}

/// `terminalIsolation`: exact container identity, project label,
/// unprivileged private userns, production ID mappings.
pub fn terminal_isolation(v: &TerminalInspection, id: &str) -> bool {
    if !domain::valid_container_id(&v.id)
        || v.project != id
        || v.privileged
        || v.userns != "private"
    {
        return false;
    }
    terminal_id_map(&v.uid_map) && terminal_id_map(&v.gid_map)
}

/// Go `strconv.ParseInt(s, 10, 64)`, probed: one optional sign, ASCII
/// digits, range-checked. Rust's `FromStr` matches exactly (leading `+`
/// and zeros accepted, underscores/spaces rejected).
pub(crate) fn parse_go_int(s: &str) -> Option<i64> {
    s.parse::<i64>().ok()
}

/// Go `strconv.ParseUint(s, 10, bits)`, probed: ASCII digits only (no
/// sign), range-checked.
pub(crate) fn parse_go_uint(s: &str, bits: u32) -> Option<u64> {
    if s.is_empty() || !s.bytes().all(|b| b.is_ascii_digit()) {
        return None;
    }
    let value: u64 = s.parse().ok()?;
    let max = if bits >= 64 {
        u64::MAX
    } else {
        (1u64 << bits) - 1
    };
    if value > max {
        None
    } else {
        Some(value)
    }
}

/// `terminalTargetReady`: positive owner plus isolation, running when required.
pub fn terminal_target_ready(v: &TerminalInspection, id: &str, require_running: bool) -> bool {
    let Some(owner) = parse_go_int(&v.owner) else {
        return false;
    };
    if owner <= 0 {
        return false;
    }
    if require_running && !v.running {
        return false;
    }
    terminal_isolation(v, id)
}

/// Parse the exit code out of a crate-executor error string. Both the real
/// executor (`{cmd} failed: exit status N: {stderr}`) and test fakes (bare
/// `exit status N`) put the true code at the first `exit status ` marker;
/// anything else fails closed to `None`.
pub fn exit_code_of(err: &str) -> Option<i32> {
    let marker = "exit status ";
    let start = err.find(marker)? + marker.len();
    let digits: String = err[start..]
        .chars()
        .take_while(|c| c.is_ascii_digit())
        .collect();
    if digits.is_empty() || digits.len() > 10 {
        return None;
    }
    digits.parse::<i32>().ok()
}
