use std::ffi::OsString;

use super::NestedRegistration;
use crate::domain;
use crate::json::{self, SignedInteger};
use crate::terminal::{self, Binding, Delivery};
use serde::de::{self, MapAccess, SeqAccess, Visitor};
use serde::Deserialize;
use std::fmt;

// ---------- pure resolution helpers (muse_linux.go) ----------

/// `museInspect`: exact podman `--format` template.
pub const MUSE_INSPECT: &str = "{\"id\":{{json .ID}},\"pid\":{{json .State.Pid}},\"project\":{{json (index .Config.Labels \"org.soda.project\")}},\"running\":{{json .State.Running}},\"privileged\":{{json .HostConfig.Privileged}},\"userns\":{{json .HostConfig.UsernsMode}}}";
/// Nested-child inspect template.
pub const MUSE_CHILD_INSPECT: &str =
    "{\"id\":{{json .ID}},\"pid\":{{json .State.Pid}},\"running\":{{json .State.Running}}}";

/// `museProjectCgroup`: the complete OCI scope component from
/// `/proc/<pid>/cgroup`, never a substring guess.
pub fn muse_project_cgroup(value: &str) -> Result<String, String> {
    for part in value.trim().split('/') {
        if let Some(id) = part
            .strip_prefix("libpod-")
            .and_then(|s| s.strip_suffix(".scope"))
        {
            if domain::valid_container_id(id) {
                return Ok(id.to_string());
            }
        }
    }
    Err(terminal::err_denied())
}

/// `museMappedUID`: inside-UID for a host UID from a uid/gid map.
pub fn muse_mapped_uid(data: &str, uid: u32) -> Result<i64, String> {
    for line in data.split('\n') {
        let p: Vec<&str> = line.split_whitespace().collect();
        if p.len() != 3 {
            continue;
        }
        let (Some(inside), Some(outside), Some(count)) = (
            terminal::parse_go_uint(p[0], 32),
            terminal::parse_go_uint(p[1], 32),
            terminal::parse_go_uint(p[2], 32),
        ) else {
            continue;
        };
        if outside > 0 && u64::from(uid) >= outside && u64::from(uid) < outside + count {
            return Ok((inside + u64::from(uid) - outside) as i64);
        }
    }
    Err(terminal::err_denied())
}

/// `musePasswdValid`: seven-field passwd row for the mapped UID.
pub fn muse_passwd_valid(account: &[String], uid: i64) -> bool {
    if account.len() != 7 {
        return false;
    }
    domain::valid_login(&account[0])
        && account[0] != "root"
        && account[2] == uid.to_string()
        && account[5].starts_with('/')
}

/// `museAccountNode`: one root-owned account-path node.
pub fn muse_account_node(line: &str, marker: bool) -> bool {
    let parts: Vec<&str> = line.splitn(4, ':').collect();
    if parts.len() != 4 || parts[0] != "0" || parts[1] != "0" {
        return false;
    }
    let Ok(mode) = u32::from_str_radix(parts[2], 8) else {
        return false;
    };
    if mode & 0o022 != 0 {
        return false;
    }
    if marker {
        return parts[3] == "regular file" && mode == 0o600;
    }
    parts[3] == "directory"
}

/// `museAccountModes`: five account-path nodes, marker last.
pub fn muse_account_modes(body: &str) -> bool {
    let lines: Vec<&str> = body.trim().split('\n').collect();
    if lines.len() != 5 {
        return false;
    }
    for (i, line) in lines.iter().enumerate() {
        if !muse_account_node(line, i == 4) {
            return false;
        }
    }
    true
}

/// `museRegistrationValid`: registration shape (Muse launches only).
pub fn muse_registration_valid(input: &NestedRegistration) -> bool {
    domain::valid_container_id(&input.child_id)
        && input.actor_id > 0
        && terminal::valid_terminal_id(&input.registration_id)
        && input.muse
}

/// `museChildPID`: running nested-child PID from its inspect output.
pub fn muse_child_pid(body: &[u8], failed: bool, id: &str) -> Result<i32, String> {
    if failed || body.len() > 4096 {
        return Err(terminal::err_denied());
    }
    let inspection: ChildInspection =
        json::decode_strict_as(body).map_err(|_| terminal::err_denied())?;
    let (Some(got), Some(pid), Some(running)) = (inspection.id, inspection.pid, inspection.running)
    else {
        return Err(terminal::err_denied());
    };
    let pid = pid.0;
    if got != id || !running || pid <= 0 || pid > i32::MAX as i64 {
        return Err(terminal::err_denied());
    }
    Ok(pid as i32)
}

#[derive(Default)]
struct ChildInspection {
    id: Option<String>,
    pid: Option<SignedInteger>,
    running: Option<bool>,
}

impl<'de> Deserialize<'de> for ChildInspection {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ChildVisitor;
        impl<'de> Visitor<'de> for ChildVisitor {
            type Value = ChildInspection;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a child inspection object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = ChildInspection::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key == "id" {
                        out.id = map.next_value()?;
                    } else if key == "pid" {
                        out.pid = map.next_value()?;
                    } else if key == "running" {
                        out.running = map.next_value()?;
                    } else {
                        map.next_value::<serde::de::IgnoredAny>()?;
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(ChildVisitor)
    }
}

#[derive(Default)]
struct SoftString(Option<String>);
impl<'de> Deserialize<'de> for SoftString {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct SoftStringVisitor;
        impl<'de> Visitor<'de> for SoftStringVisitor {
            type Value = SoftString;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("any JSON value")
            }
            fn visit_str<E>(self, value: &str) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftString(Some(value.to_string())))
            }
            fn visit_string<E>(self, value: String) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftString(Some(value)))
            }
            fn visit_unit<E>(self) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftString(None))
            }
            fn visit_bool<E>(self, _: bool) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftString(None))
            }
            fn visit_i64<E>(self, _: i64) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftString(None))
            }
            fn visit_u64<E>(self, _: u64) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftString(None))
            }
            fn visit_f64<E>(self, _: f64) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftString(None))
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                while seq.next_element::<serde::de::IgnoredAny>()?.is_some() {}
                Ok(SoftString(None))
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                while map.next_key::<serde::de::IgnoredAny>()?.is_some() {
                    map.next_value::<serde::de::IgnoredAny>()?;
                }
                Ok(SoftString(None))
            }
        }
        deserializer.deserialize_any(SoftStringVisitor)
    }
}

struct SoftBool(Option<bool>);
impl<'de> Deserialize<'de> for SoftBool {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct SoftBoolVisitor;
        impl<'de> Visitor<'de> for SoftBoolVisitor {
            type Value = SoftBool;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("any JSON value")
            }
            fn visit_bool<E>(self, value: bool) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftBool(Some(value)))
            }
            fn visit_unit<E>(self) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftBool(None))
            }
            fn visit_str<E>(self, _: &str) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftBool(None))
            }
            fn visit_string<E>(self, _: String) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftBool(None))
            }
            fn visit_i64<E>(self, _: i64) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftBool(None))
            }
            fn visit_u64<E>(self, _: u64) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftBool(None))
            }
            fn visit_f64<E>(self, _: f64) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(SoftBool(None))
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                while seq.next_element::<serde::de::IgnoredAny>()?.is_some() {}
                Ok(SoftBool(None))
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                while map.next_key::<serde::de::IgnoredAny>()?.is_some() {
                    map.next_value::<serde::de::IgnoredAny>()?;
                }
                Ok(SoftBool(None))
            }
        }
        deserializer.deserialize_any(SoftBoolVisitor)
    }
}

#[derive(Default)]
struct Mount {
    source: Option<String>,
    destination: Option<String>,
    rw: Option<bool>,
}

impl<'de> Deserialize<'de> for Mount {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct MountVisitor;
        impl<'de> Visitor<'de> for MountVisitor {
            type Value = Mount;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a mount object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = Mount::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key == "Source" {
                        out.source = Some(map.next_value::<SoftString>()?.0).flatten();
                    } else if key == "Destination" {
                        out.destination = Some(map.next_value::<SoftString>()?.0).flatten();
                    } else if key == "RW" {
                        out.rw = Some(map.next_value::<SoftBool>()?.0).flatten();
                    } else {
                        map.next_value::<serde::de::IgnoredAny>()?;
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(MountVisitor)
    }
}

struct MountItem(Option<Mount>);
impl<'de> Deserialize<'de> for MountItem {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ItemVisitor;
        impl<'de> Visitor<'de> for ItemVisitor {
            type Value = MountItem;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a JSON value")
            }
            fn visit_map<A>(self, map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                Mount::deserialize(de::value::MapAccessDeserializer::new(map))
                    .map(|m| MountItem(Some(m)))
            }
            fn visit_unit<E>(self) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(MountItem(None))
            }
            fn visit_bool<E>(self, _: bool) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(MountItem(None))
            }
            fn visit_i64<E>(self, _: i64) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(MountItem(None))
            }
            fn visit_u64<E>(self, _: u64) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(MountItem(None))
            }
            fn visit_f64<E>(self, _: f64) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(MountItem(None))
            }
            fn visit_str<E>(self, _: &str) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(MountItem(None))
            }
            fn visit_string<E>(self, _: String) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(MountItem(None))
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                while seq.next_element::<serde::de::IgnoredAny>()?.is_some() {}
                Ok(MountItem(None))
            }
        }
        deserializer.deserialize_any(ItemVisitor)
    }
}

/// `museReadonlyMount`: the required read-only nested credential mount.
/// Decoded tolerantly (`encoding/json`, not strict) like Go.
pub fn muse_readonly_mount(
    body: &[u8],
    failed: bool,
    source: &str,
    destination: &str,
) -> Result<(), String> {
    if failed || body.len() > 32768 {
        return Err(terminal::err_denied());
    }
    let mounts: Vec<MountItem> =
        json::decode_tolerant_as(body).map_err(|_| terminal::err_denied())?;
    for MountItem(item) in mounts {
        let Some(mount) = item else {
            continue;
        };
        if mount.source.as_deref() == Some(source)
            && mount.destination.as_deref() == Some(destination)
            && mount.rw == Some(false)
        {
            return Ok(());
        }
    }
    Err(terminal::err_denied())
}

/// `museELF`: 64-bit little-endian ELF for the host architecture.
pub fn muse_elf(header: &[u8], arch: &str) -> Result<(), String> {
    if header.len() != soda_build_tools::elf::HEADER_LEN {
        return Err(terminal::err_denied());
    }
    let machine = soda_build_tools::elf::elf64_le_header(header)
        .ok_or_else(terminal::err_denied)?.machine;
    if arch == "amd64" && machine == 62 {
        return Ok(());
    }
    if arch == "arm64" && machine == 183 {
        return Ok(());
    }
    Err(terminal::err_denied())
}

/// Host `GOARCH` spelled the Go way.
pub fn host_go_arch() -> &'static str {
    match std::env::consts::ARCH {
        "x86_64" => "amd64",
        "aarch64" => "arm64",
        _ => "unknown",
    }
}

/// `museSignal`: forwardable unit signals.
pub fn muse_signal(sig: i64) -> bool {
    if !(1..=64).contains(&sig) {
        return false;
    }
    matches!(
        sig as i32,
        libc::SIGINT
            | libc::SIGTERM
            | libc::SIGHUP
            | libc::SIGQUIT
            | libc::SIGTSTP
            | libc::SIGCONT
            | libc::SIGUSR1
            | libc::SIGUSR2
    )
}

/// `museProjectCredentialRoot`: exact `/run/soda-muse/` execution path.
pub fn muse_project_credential_root(binding: &Binding) -> bool {
    if !domain::valid_container_id(&binding.project)
        || !terminal::is_clean_absolute_path(&binding.credential_root)
        || !binding.credential_root.starts_with("/run/soda-muse/")
    {
        return false;
    }
    let parts: Vec<&str> = binding.credential_root["/run/soda-muse/".len()..]
        .split('/')
        .collect();
    if parts.len() == 1 {
        return parts[0] == binding.id;
    }
    parts.len() == 3
        && parts[0] == "nested"
        && terminal::valid_terminal_id(parts[1])
        && parts[2] == binding.id
}

/// `museDeliveryValid`: well-formed Muse delivery for an operation.
pub fn muse_delivery_valid(delivery: &Delivery) -> bool {
    let Some(binding) = &delivery.lease.binding else {
        return false;
    };
    delivery.lease.provider_id == terminal::PROVIDER_MUSE
        && terminal::valid_terminal_id(&delivery.lease.execution_id)
        && binding.id == delivery.lease.execution_id
        && terminal::valid_terminal_id(&binding.invocation_id)
}

/// `museHostEnvironment`: operator environment minus the META key.
pub fn muse_host_environment() -> Vec<(OsString, OsString)> {
    std::env::vars_os()
        .filter(|(name, _)| name.as_os_str() != std::ffi::OsStr::new("META_API_KEY"))
        .collect()
}
