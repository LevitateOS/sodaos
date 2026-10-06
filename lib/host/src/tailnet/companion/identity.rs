use std::os::unix::fs::MetadataExt;

use crate::json::{bind_root, decode_strict, Kind, Spec};

use crate::tailnet_domain::{
    parse_rfc3339_nano, valid_container_id, ERR_CONFLICT, ERR_UNAVAILABLE,
};
use crate::tailnet_runtime::{companion_create_args, ProjectRun};

/// Decoded companion `inspect` record. Field names match the Go struct so
/// binding errors keep Go's `companion.<Field>` paths.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct CompanionRecord {
    pub id: String,
    pub image: String,
    pub command: Vec<String>,
    pub running: bool,
    pub pid: i64,
    pub started: String,
    pub execs: Vec<String>,
}

const COMPANION_SPECS: &[Spec] = &[
    Spec {
        name: "ID",
        kind: Kind::Str,
    },
    Spec {
        name: "Image",
        kind: Kind::Str,
    },
    Spec {
        name: "Command",
        kind: Kind::StrList,
    },
    Spec {
        name: "Running",
        kind: Kind::Bool,
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
        name: "Execs",
        kind: Kind::StrList,
    },
];

pub(crate) fn decode_companion_record(data: &[u8]) -> Result<CompanionRecord, String> {
    let value = decode_strict(data).map_err(|e| e.0)?;
    let bound = bind_root(&value, "companion", COMPANION_SPECS, false).map_err(|e| e.0)?;
    Ok(CompanionRecord {
        id: bound.take_string("ID"),
        image: bound.take_string("Image"),
        command: bound.take_str_list("Command"),
        running: bound.take_bool("Running"),
        pid: bound.take_i64("PID"),
        started: bound.take_string("Started"),
        execs: bound.take_str_list("Execs"),
    })
}

pub fn companion_identity_matches(
    out: &CompanionRecord,
    _run: &ProjectRun,
    image: &str,
    id: &str,
) -> bool {
    valid_container_id(id)
        && out.id == id
        && out.image.trim_start_matches("sha256:") == image.trim_start_matches("sha256:")
}

pub fn companion_command_matches(cmd: &[String], args: &[String]) -> bool {
    cmd.len() == args.len() + 1
        && (cmd[0] == "/usr/bin/podman" || cmd[0] == "podman")
        && cmd[1..] == args[..]
}

pub fn companion_execs_valid(execs: &[String]) -> Result<(), String> {
    if execs.len() > 16 {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    for id in execs {
        if !valid_container_id(id) {
            return Err(ERR_UNAVAILABLE.to_string());
        }
    }
    Ok(())
}

pub fn validate_companion_record(
    out: &CompanionRecord,
    run: &ProjectRun,
    image: &str,
    id: &str,
) -> Result<(), String> {
    let args = companion_create_args(run, image)?;
    if !companion_identity_matches(out, run, image, id) {
        return Err(ERR_CONFLICT.to_string());
    }
    if out.running {
        match parse_rfc3339_nano(&out.started) {
            Some(false) => {}
            _ => return Err(ERR_CONFLICT.to_string()),
        }
        if out.pid <= 1 {
            return Err(ERR_CONFLICT.to_string());
        }
    }
    // CreateCommand is retained by Podman. Admit exactly our immutable recipe,
    // allowing only the native argv[0] spelling, never labels/name alone.
    if !companion_command_matches(&out.command, &args) {
        return Err(ERR_CONFLICT.to_string());
    }
    companion_execs_valid(&out.execs)
}

/// Live `/proc` identity reduced to the fields a companion incarnation must
/// match. Start-time and boot-id reads only gate validity, as in Go.
struct ProcessIdentity {
    userns: String,
    netns: String,
    uid: u32,
    gid: u32,
}

fn parse_stat_fields(pid: i64, data: &[u8]) -> Result<Vec<String>, String> {
    if data.len() > 8192 {
        return Err(ERR_CONFLICT.to_string());
    }
    let first = data
        .iter()
        .position(|&b| b == b' ')
        .ok_or_else(|| ERR_CONFLICT.to_string())?;
    let end = data
        .iter()
        .rposition(|&b| b == b')')
        .ok_or_else(|| ERR_CONFLICT.to_string())?;
    if data[..first] != *pid.to_string().as_bytes() {
        return Err(ERR_CONFLICT.to_string());
    }
    let rest = std::str::from_utf8(&data[end + 1..]).map_err(|_| ERR_CONFLICT.to_string())?;
    let fields: Vec<String> = rest.split_whitespace().map(str::to_string).collect();
    if fields.len() < 20 || fields[0] == "Z" || fields[0] == "X" {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(fields)
}

fn parse_process_start(fields: &[String]) -> Result<(), String> {
    match fields[19].parse::<u64>() {
        Ok(n) if n != 0 && n.to_string() == fields[19] => Ok(()),
        _ => Err(ERR_CONFLICT.to_string()),
    }
}

fn read_process_id_map(
    base: &str,
    name: &str,
    read: &dyn Fn(&str) -> Result<Vec<u8>, String>,
) -> Result<u32, String> {
    let data = read(&format!("{base}/{name}")).map_err(|_| ERR_CONFLICT.to_string())?;
    if data.len() > 4096 {
        return Err(ERR_CONFLICT.to_string());
    }
    let text = String::from_utf8_lossy(&data);
    let fields: Vec<&str> = text.split_whitespace().collect();
    if fields.len() != 3 {
        return Err(ERR_CONFLICT.to_string());
    }
    if fields[0] != "0" || fields[2] != "262144" {
        return Err(ERR_CONFLICT.to_string());
    }
    match fields[1].parse::<u32>() {
        Ok(n)
            if n != 0
                && n.to_string() == fields[1]
                && u64::from(n) + 262144 <= u64::from(u32::MAX) =>
        {
            Ok(n)
        }
        _ => Err(ERR_CONFLICT.to_string()),
    }
}

fn read_process_namespace(
    base: &str,
    name: &str,
    link: &dyn Fn(&str) -> Result<String, String>,
) -> Result<String, String> {
    let value = link(&format!("{base}/ns/{name}")).map_err(|_| ERR_CONFLICT.to_string())?;
    let prefix = format!("{name}:[");
    let inner = value
        .strip_prefix(&prefix)
        .and_then(|s| s.strip_suffix(']'))
        .ok_or_else(|| ERR_CONFLICT.to_string())?;
    match inner.parse::<u64>() {
        Ok(n) if n != 0 && n.to_string() == inner => {}
        _ => return Err(ERR_CONFLICT.to_string()),
    }
    let host = link(&format!("/proc/1/ns/{name}")).map_err(|_| ERR_CONFLICT.to_string())?;
    if host == value {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(value)
}

fn read_system_boot_id(read: &dyn Fn(&str) -> Result<Vec<u8>, String>) -> Result<(), String> {
    let boot = read("/proc/sys/kernel/random/boot_id").map_err(|_| ERR_CONFLICT.to_string())?;
    let text = String::from_utf8_lossy(&boot);
    let trimmed = text.trim();
    if trimmed.len() != 36 || trimmed.bytes().filter(|&b| b == b'-').count() != 4 {
        return Err(ERR_CONFLICT.to_string());
    }
    let compact: String = trimmed.chars().filter(|&c| c != '-').collect();
    if compact.len() != 32 || !compact.bytes().all(|b| b.is_ascii_hexdigit()) {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(())
}

fn process_run_identity(
    pid: i64,
    read: &dyn Fn(&str) -> Result<Vec<u8>, String>,
    link: &dyn Fn(&str) -> Result<String, String>,
) -> Result<ProcessIdentity, String> {
    if pid <= 1 {
        return Err(ERR_CONFLICT.to_string());
    }
    let data = read(&format!("/proc/{pid}/stat")).map_err(|_| ERR_CONFLICT.to_string())?;
    let fields = parse_stat_fields(pid, &data)?;
    parse_process_start(&fields)?;
    let base = format!("/proc/{pid}");
    let uid = read_process_id_map(&base, "uid_map", read)?;
    let gid = read_process_id_map(&base, "gid_map", read)?;
    let userns = read_process_namespace(&base, "user", link)?;
    let netns = read_process_namespace(&base, "net", link)?;
    read_system_boot_id(read)?;
    Ok(ProcessIdentity {
        userns,
        netns,
        uid,
        gid,
    })
}

pub fn match_companion_namespaces(out: &CompanionRecord, run: &ProjectRun) -> Result<(), String> {
    if !out.running {
        return Ok(());
    }
    let read = |path: &str| std::fs::read(path).map_err(|_| ERR_CONFLICT.to_string());
    let link = |path: &str| {
        std::fs::read_link(path)
            .map(|p| p.to_string_lossy().into_owned())
            .map_err(|_| ERR_CONFLICT.to_string())
    };
    match process_run_identity(out.pid, &read, &link) {
        Ok(identity)
            if identity.userns == run.userns
                && identity.netns == run.netns
                && identity.uid == run.uid
                && identity.gid == run.gid =>
        {
            Ok(())
        }
        _ => Err(ERR_CONFLICT.to_string()),
    }
}

pub(crate) fn stat_metadata(path: &str) -> Result<std::fs::Metadata, String> {
    std::fs::metadata(path).map_err(|_| ERR_CONFLICT.to_string())
}

/// Enrollment/observations use the actual resolver inode, never Podman's
/// generated `ResolvConfPath` metadata.
pub(crate) fn companion_resolver(
    run: &ProjectRun,
    pid: i64,
    stat: &dyn Fn(&str) -> Result<std::fs::Metadata, String>,
) -> Result<(), String> {
    let original = stat(&run.resolver).map_err(|_| ERR_CONFLICT.to_string())?;
    let actual =
        stat(&format!("/proc/{pid}/root/etc/resolv.conf")).map_err(|_| ERR_CONFLICT.to_string())?;
    if !original.is_file() || original.dev() != actual.dev() || original.ino() != actual.ino() {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(())
}
