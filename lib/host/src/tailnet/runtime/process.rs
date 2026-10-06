use crate::tailnet_domain::{LinkFile, ReadFile, ERR_CONFLICT, ERR_UNSUPPORTED};

use super::{id_map, unavailable, ProcessIdentity};

/// Go `strings.Fields` over /proc text: Unicode whitespace when the input is
/// valid UTF-8, ASCII-whitespace byte split otherwise.
fn split_fields(raw: &[u8]) -> Vec<String> {
    match std::str::from_utf8(raw) {
        Ok(s) => s.split_whitespace().map(str::to_string).collect(),
        Err(_) => raw
            .split(|b| matches!(*b, b' ' | b'\t' | b'\n' | b'\r' | b'\x0c' | b'\x0b'))
            .filter(|part| !part.is_empty())
            .map(|part| String::from_utf8_lossy(part).into_owned())
            .collect(),
    }
}

/// Split `/proc/<pid>/stat` into state-first fields, refusing oversized,
/// mismatched-pid and dead (zombie/exited) records.
pub fn parse_stat_fields(pid: i64, data: &[u8]) -> Result<Vec<String>, String> {
    if data.len() > 8192 {
        return Err(unavailable());
    }
    let end = data.iter().rposition(|&b| b == b')');
    let first = data.iter().position(|&b| b == b' ');
    match (end, first) {
        (Some(end), Some(first)) if &data[..first] == pid.to_string().as_bytes() => {
            let fields = split_fields(&data[end + 1..]);
            if fields.len() < 20 || fields[0] == "Z" || fields[0] == "X" {
                return Err(unavailable());
            }
            Ok(fields)
        }
        _ => Err(unavailable()),
    }
}

fn parse_start_time_inner(pid: i64, read: &ReadFile) -> Result<String, String> {
    let path = format!("/proc/{pid}/stat");
    let data = read(&path).map_err(|_| unavailable())?;
    let fields = parse_stat_fields(pid, &data)?;
    let token = &fields[19];
    match token.parse::<u64>() {
        Ok(n) if n != 0 && n.to_string() == *token => Ok(token.clone()),
        _ => Err(unavailable()),
    }
}

/// Canonical start-time token (field 22) of `/proc/<pid>/stat`.
pub fn parse_process_start_time(pid: i64, read: ReadFile) -> Result<String, String> {
    parse_start_time_inner(pid, &read)
}

fn read_id_map_inner(base: &str, name: &str, read: &ReadFile) -> Result<u32, String> {
    let path = format!("{base}/{name}");
    let raw = read(&path).map_err(|_| unavailable())?;
    if raw.len() > 4096 {
        return Err(unavailable());
    }
    let fields = split_fields(&raw);
    if fields.len() != 3 || !id_map(&[fields.join(":")]) {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    match fields[1].parse::<u32>() {
        Ok(n) => Ok(n),
        Err(_) => Ok(0),
    }
}

/// Host UID/GID base of a single-entry id map; anything else is unsupported.
pub fn read_process_id_map(base: &str, name: &str, read: ReadFile) -> Result<u32, String> {
    read_id_map_inner(base, name, &read)
}

/// Validate one `/proc/<pid>/ns/<name>` link target, returning it verbatim.
pub fn parse_namespace_link(name: &str, v: &str) -> Result<String, String> {
    let prefix = format!("{name}:[");
    let inner = match v
        .strip_prefix(prefix.as_str())
        .and_then(|s| s.strip_suffix(']'))
    {
        Some(inner) => inner,
        None => return Err(unavailable()),
    };
    match inner.parse::<u64>() {
        Ok(n) if n != 0 && n.to_string() == inner => Ok(v.to_string()),
        _ => Err(unavailable()),
    }
}

fn read_namespace_inner(base: &str, name: &str, link: &LinkFile) -> Result<String, String> {
    let path = format!("{base}/ns/{name}");
    let v = link(&path).map_err(|_| unavailable())?;
    parse_namespace_link(name, &v)?;
    let host_path = format!("/proc/1/ns/{name}");
    match link(&host_path) {
        Ok(host) if host != v => Ok(v),
        _ => Err(ERR_UNSUPPORTED.to_string()),
    }
}

/// Namespace link of a process, refusing namespaces shared with PID 1.
pub fn read_process_namespace(base: &str, name: &str, link: LinkFile) -> Result<String, String> {
    read_namespace_inner(base, name, &link)
}

fn read_boot_id_inner(read: &ReadFile) -> Result<String, String> {
    let boot = read("/proc/sys/kernel/random/boot_id").map_err(|_| unavailable())?;
    let text = String::from_utf8_lossy(&boot);
    let trimmed = text.trim();
    if trimmed.len() != 36 || trimmed.bytes().filter(|&b| b == b'-').count() != 4 {
        return Err(unavailable());
    }
    let compact: Vec<u8> = trimmed.bytes().filter(|&b| b != b'-').collect();
    if compact.len() != 32 || !compact.iter().all(|b| b.is_ascii_hexdigit()) {
        return Err(unavailable());
    }
    Ok(trimmed.to_string())
}

/// Kernel boot ID binding the start-time token to this boot.
pub fn read_system_boot_id(read: ReadFile) -> Result<String, String> {
    read_boot_id_inner(&read)
}

/// Full native identity of a project container init process.
pub fn process_run_identity(
    pid: i64,
    read: ReadFile,
    link: LinkFile,
) -> Result<ProcessIdentity, String> {
    if pid <= 1 {
        return Err(ERR_CONFLICT.to_string());
    }
    let base = format!("/proc/{pid}");
    Ok(ProcessIdentity {
        start: parse_start_time_inner(pid, &read)?,
        uid: read_id_map_inner(&base, "uid_map", &read)?,
        gid: read_id_map_inner(&base, "gid_map", &read)?,
        userns: read_namespace_inner(&base, "user", &link)?,
        netns: read_namespace_inner(&base, "net", &link)?,
        boot: read_boot_id_inner(&read)?,
    })
}
