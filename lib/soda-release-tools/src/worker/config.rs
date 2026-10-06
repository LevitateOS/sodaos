//! Worker configuration: paths, identity, admission, and config decode.
use std::os::unix::fs::{MetadataExt, PermissionsExt};

use crate::build_spec::Request;
use crate::digest::hash_file;

pub const WORKER_SOURCE: &str = "/run/soda-build-source";
pub const WORKER_HOME: &str = "/var/lib/soda-build-worker";
pub const WORKER_RUNTIME: &str = "/run/soda-build-worker";
pub const WORKER_TOOLS: &str = "/run/soda-build-tools";
pub const WORKER_FORGEJO: &str = "/run/soda-build-forgejo-source";
pub const PINNED_GO_ROOT: &str = "/usr/local/lib/soda/pinned-go";

#[derive(Debug, Clone, Default)]
pub struct WorkerConfig {
    pub executable: String,
    pub source: String,
    pub forgejo_source: String,
    pub output_parent: String,
    pub storage_root: String,
    pub build_home: String,
    pub runtime: String,
    pub tools: String,
    pub media_authority_directory: String,
}

#[derive(Debug, Clone, Default)]
pub struct Worker {
    pub name: String,
    pub user: String,
    pub executable: String,
    pub directory: String,
    pub read_only: Vec<String>,
    pub writable: Vec<String>,
    pub environment: Vec<String>,
    pub arguments: Vec<String>,
}

pub fn euid() -> u32 {
    unsafe { libc::getuid() }
}

/// `user.Lookup` equivalent over `/etc/passwd` (pure-Go behavior).
pub fn lookup_user(name: &str) -> Result<(u32, u32), String> {
    let data = std::fs::read_to_string("/etc/passwd").map_err(|e| e.to_string())?;
    for line in data.lines() {
        if line.starts_with('#') {
            continue;
        }
        let mut fields = line.split(':');
        let (user, _, uid, gid) = (
            fields.next().unwrap_or_default(),
            fields.next(),
            fields.next().unwrap_or_default(),
            fields.next().unwrap_or_default(),
        );
        if user == name {
            let uid = uid.parse::<u32>().map_err(|e| e.to_string())?;
            let gid = gid.parse::<u32>().map_err(|e| e.to_string())?;
            return Ok((uid, gid));
        }
    }
    Err(format!("user: unknown user {name}"))
}

pub fn build_worker_identity() -> Result<(), String> {
    let (uid, _) = lookup_user("soda-build-worker")?;
    if uid == 0 || euid() != uid {
        return Err("build stage requires the isolated soda-build-worker identity".to_owned());
    }
    Ok(())
}

pub fn worker_runtime_ids() -> Result<(u32, u32), String> {
    lookup_user("soda-build-worker")
}

pub(crate) fn go_clean(path: &str) -> String {
    // `filepath.Clean` equivalent for absolute paths.
    let mut parts: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                parts.pop();
            }
            _ => parts.push(part),
        }
    }
    format!("/{}", parts.join("/"))
}

fn is_abs_clean(path: &str) -> bool {
    path.starts_with('/') && go_clean(path) == path
}

pub fn valid_worker_task_paths(c: &WorkerConfig, r: &Request) -> bool {
    if c.source != r.source || c.forgejo_source != r.forgejo_source {
        return false;
    }
    let parent = parent_dir(&r.out);
    if parent != c.output_parent {
        return false;
    }
    let prefix = format!("{}/.artifacts/releases/", r.source.trim_end_matches('/'));
    parent.starts_with(&prefix) && parent.len() > prefix.len()
}

fn parent_dir(path: &str) -> String {
    match path.rfind('/') {
        Some(0) => "/".to_owned(),
        Some(i) => path[..i].to_owned(),
        None => ".".to_owned(),
    }
}

pub fn worker_executable_matches(path: &str) -> Result<(), String> {
    let current = std::env::current_exe().map_err(|e| e.to_string())?;
    let a = hash_file(current.to_string_lossy().as_ref())?;
    let b = hash_file(path)?;
    if a != b {
        return Err("dispatcher differs from admitted worker executable".to_owned());
    }
    Ok(())
}

fn eval_strict(path: &str) -> Result<String, String> {
    let resolved = std::fs::canonicalize(path).map_err(|e| e.to_string())?;
    Ok(resolved.to_string_lossy().into_owned())
}

pub fn valid_worker_path(path: &str) -> Result<(), String> {
    if !is_abs_clean(path) || path.chars().any(|c| ":\n\r\t %".contains(c)) {
        return Err("explicit safe worker path required".to_owned());
    }
    match eval_strict(path) {
        Ok(resolved) if resolved == path => Ok(()),
        _ => Err("worker paths must exist without symlink traversal".to_owned()),
    }
}

/// `acceptance.TrustedExecutable`: every path component root-owned,
/// non-writable, non-symlink; the leaf a regular executable.
pub fn trusted_executable(path: &str) -> Result<(), String> {
    if !is_abs_clean(path) {
        return Err("absolute admitted executable required".to_owned());
    }
    let mut current = path.to_owned();
    loop {
        let st = std::fs::symlink_metadata(&current).map_err(|e| e.to_string())?;
        let mode = st.permissions().mode();
        if st.uid() != 0 || mode & 0o022 != 0 || st.file_type().is_symlink() {
            return Err(
                "worker executable and parents must be root-owned and not group/world writable"
                    .to_owned(),
            );
        }
        if current == path && (!st.file_type().is_file() || mode & 0o111 == 0) {
            return Err("admitted regular executable required".to_owned());
        }
        if current == "/" {
            break;
        }
        current = parent_dir(&current);
    }
    Ok(())
}

pub fn admit_loaded_worker_config(c: &WorkerConfig, r: &Request) -> Result<(), String> {
    if !valid_worker_task_paths(c, r) {
        return Err("worker source/output differs from approved task paths".to_owned());
    }
    trusted_executable(&c.executable)?;
    worker_executable_matches(&c.executable)
}

pub fn worker_config_paths(c: &WorkerConfig, r: &Request) -> Vec<String> {
    let mut paths = vec![
        c.source.clone(),
        c.forgejo_source.clone(),
        c.output_parent.clone(),
        c.storage_root.clone(),
        c.build_home.clone(),
        c.runtime.clone(),
        c.tools.clone(),
    ];
    if r.wants_media() {
        paths.push(c.media_authority_directory.clone());
    }
    paths
}

pub fn admit_storage_root(c: &WorkerConfig) -> Result<(), String> {
    if c.storage_root.is_empty() || !is_abs_clean(&c.storage_root) {
        return Err("worker storage root required; rerun the setup script".to_owned());
    }
    for (name, p) in [("build home", &c.build_home), ("runtime", &c.runtime)] {
        if p != &c.storage_root && !p.starts_with(&format!("{}/", c.storage_root)) {
            return Err(format!(
                "worker {name} {p:?} is outside the admitted storage root {:?}",
                c.storage_root
            ));
        }
    }
    Ok(())
}

/// `deliver.PrivateFile`: absolute, symlink-free, caller-owned private
/// regular file under a private parent, at most 1 MiB.
pub fn private_file(path: &str) -> Result<(), String> {
    if !path.starts_with('/') {
        return Err("refused".to_owned());
    }
    match eval_strict(path) {
        Ok(resolved) if resolved == path => {}
        _ => return Err("refused".to_owned()),
    }
    let st = std::fs::symlink_metadata(path).map_err(|_| "refused".to_owned())?;
    if !st.file_type().is_file()
        || st.permissions().mode() & 0o077 != 0
        || st.len() > 1 << 20
        || st.uid() != euid()
    {
        return Err("refused".to_owned());
    }
    let parent = std::fs::metadata(parent_dir(path)).map_err(|_| "refused".to_owned())?;
    if parent.permissions().mode() & 0o077 != 0 {
        return Err("refused".to_owned());
    }
    Ok(())
}

fn strict_string_field(
    entries: &[(String, soda_json::JsonValue)],
    key: &str,
) -> Result<String, String> {
    let mut found: Option<String> = None;
    for (k, v) in entries {
        if k == key {
            match v.as_str() {
                Some(s) => found = Some(s.to_owned()),
                None => return Err(format!("invalid {key}")),
            }
        }
    }
    found.ok_or_else(|| format!("missing {key}"))
}

fn decode_worker_config(data: &[u8]) -> Result<WorkerConfig, String> {
    const KNOWN: &[&str] = &[
        "Executable",
        "Source",
        "ForgejoSource",
        "OutputParent",
        "StorageRoot",
        "BuildHome",
        "Runtime",
        "Tools",
        "MediaAuthorityDirectory",
    ];
    let text = std::str::from_utf8(data).map_err(|e| e.to_string())?;
    let value =
        soda_json::JsonValue::parse(text).map_err(|_| "invalid worker configuration".to_owned())?;
    let soda_json::JsonValue::Object(entries) = &value else {
        return Err("invalid worker configuration".to_owned());
    };
    for (k, _) in entries {
        if !KNOWN.contains(&k.as_str()) {
            return Err(format!("unknown worker configuration field {k:?}"));
        }
    }
    Ok(WorkerConfig {
        executable: strict_string_field(entries, "Executable").unwrap_or_default(),
        source: strict_string_field(entries, "Source").unwrap_or_default(),
        forgejo_source: strict_string_field(entries, "ForgejoSource").unwrap_or_default(),
        output_parent: strict_string_field(entries, "OutputParent").unwrap_or_default(),
        storage_root: strict_string_field(entries, "StorageRoot").unwrap_or_default(),
        build_home: strict_string_field(entries, "BuildHome").unwrap_or_default(),
        runtime: strict_string_field(entries, "Runtime").unwrap_or_default(),
        tools: strict_string_field(entries, "Tools").unwrap_or_default(),
        media_authority_directory: strict_string_field(entries, "MediaAuthorityDirectory")
            .unwrap_or_default(),
    })
}

/// `deliver.ReadJSON` equivalent for the worker config: bounded regular
/// input decoded strictly.
pub fn read_worker_config(path: &str) -> Result<WorkerConfig, String> {
    let st = std::fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    if !st.file_type().is_file() || st.len() > 1 << 20 {
        return Err("bounded regular JSON input required".to_owned());
    }
    let data = std::fs::read(path).map_err(|e| e.to_string())?;
    if data.len() > 1 << 20 {
        return Err("bounded regular JSON input required".to_owned());
    }
    decode_worker_config(&data)
}

pub fn load_worker_config(path: &str, r: &Request) -> Result<WorkerConfig, String> {
    if euid() != 0 {
        return Err(
            "run the admitted controller as root; source commands run only as soda-build-worker"
                .to_owned(),
        );
    }
    if private_file(path).is_err() {
        return Err("root-owned restricted worker configuration required".to_owned());
    }
    let c = read_worker_config(path)?;
    admit_storage_root(&c)?;
    admit_loaded_worker_config(&c, r)?;
    for p in worker_config_paths(&c, r) {
        valid_worker_path(&p)?;
    }
    Ok(c)
}
