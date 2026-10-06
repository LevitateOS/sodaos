use crate::domain;
use crate::json::{self, Kind, Spec};

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

const MAPPINGS_SPECS: &[Spec] = &[
    Spec {
        name: "UidMap",
        kind: Kind::StrList,
    },
    Spec {
        name: "GidMap",
        kind: Kind::StrList,
    },
];

const INSPECTION_SPECS: &[Spec] = &[
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
            specs: MAPPINGS_SPECS,
        },
    },
];

impl TerminalInspection {
    /// Strict decode (`strictjson.Decode` in Go).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        let m =
            json::bind_root(&v, "terminalInspection", INSPECTION_SPECS, false).map_err(|e| e.0)?;
        let mappings = m.take_map("mappings");
        Ok(TerminalInspection {
            id: m.take_string("id"),
            running: m.take_bool("running"),
            project: m.take_string("project"),
            owner: m.take_string("owner"),
            privileged: m.take_bool("privileged"),
            userns: m.take_string("userns"),
            uid_map: mappings.take_str_list("UidMap"),
            gid_map: mappings.take_str_list("GidMap"),
        })
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
