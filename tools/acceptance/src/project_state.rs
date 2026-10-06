//! Bounded, read-only U08 state snapshot. This module owns the probe (it
//! replaces the retired `tests/installed/project-state.py`).
//!
//! Run as root INSIDE a selected project. No
//! environment/secret/DB-credential/shadow/private-key contents are
//! exported. The driver supplies the compiled probe over the selected
//! administrator SSH session.
//!
//! Failure taxonomy mirrors the retired probe: `RuntimeError` and
//! `AssertionError` keep their detail text, every other kind renders
//! with an empty detail.

use std::collections::BTreeSet;
use std::os::unix::ffi::OsStrExt;
use std::path::{Path, PathBuf};

use soda_json::JsonValue;

mod command;
mod files;

pub use command::{command, command_with_timeout, output_lines};
pub use files::{dumps_sorted, Entry, EntryBody};

use files::{has_git_part, list_files, walk_sorted};

/// Snapshot failure kind, mirroring the Python exception taxonomy.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SnapshotKind {
    /// Explicit snapshot violation.
    RuntimeError,
    /// Failed assertion.
    AssertionError,
    /// Missing path.
    FileNotFoundError,
    /// Unreadable path.
    PermissionError,
    /// Other OS failure.
    OSError,
    /// Snapshot command timed out.
    TimeoutExpired,
    /// Snapshot command output was not UTF-8.
    UnicodeDecodeError,
    /// Malformed project IP.
    ValueError,
    /// Missing required environment variable.
    KeyError,
}

impl SnapshotKind {
    /// Exception name.
    pub fn name(self) -> &'static str {
        match self {
            SnapshotKind::RuntimeError => "RuntimeError",
            SnapshotKind::AssertionError => "AssertionError",
            SnapshotKind::FileNotFoundError => "FileNotFoundError",
            SnapshotKind::PermissionError => "PermissionError",
            SnapshotKind::OSError => "OSError",
            SnapshotKind::TimeoutExpired => "TimeoutExpired",
            SnapshotKind::UnicodeDecodeError => "UnicodeDecodeError",
            SnapshotKind::ValueError => "ValueError",
            SnapshotKind::KeyError => "KeyError",
        }
    }
}

/// Snapshot failure. Only `RuntimeError` and `AssertionError` carry
/// detail, like the retired probe's handler.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SnapshotFailure {
    /// Failure kind.
    pub kind: SnapshotKind,
    /// Detail text (empty unless RuntimeError/AssertionError).
    pub detail: String,
}

impl SnapshotFailure {
    fn runtime(detail: impl Into<String>) -> SnapshotFailure {
        SnapshotFailure {
            kind: SnapshotKind::RuntimeError,
            detail: detail.into(),
        }
    }

    fn assertion(detail: impl Into<String>) -> SnapshotFailure {
        SnapshotFailure {
            kind: SnapshotKind::AssertionError,
            detail: detail.into(),
        }
    }

    fn bare(kind: SnapshotKind) -> SnapshotFailure {
        SnapshotFailure {
            kind,
            detail: String::new(),
        }
    }

    fn io(error: std::io::Error) -> SnapshotFailure {
        use std::io::ErrorKind;
        match error.kind() {
            ErrorKind::NotFound => SnapshotFailure::bare(SnapshotKind::FileNotFoundError),
            ErrorKind::PermissionDenied => SnapshotFailure::bare(SnapshotKind::PermissionError),
            // `read_to_string` is the only `InvalidData` source here, like
            // the retired probe's strict `read_text`/`decode` calls.
            ErrorKind::InvalidData => SnapshotFailure::bare(SnapshotKind::UnicodeDecodeError),
            _ => SnapshotFailure::bare(SnapshotKind::OSError),
        }
    }
}

impl std::fmt::Display for SnapshotFailure {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "Project snapshot failed: {} {}",
            self.kind.name(),
            self.detail
        )
    }
}

impl std::error::Error for SnapshotFailure {}

/// Empty JSON object.
fn obj() -> JsonValue {
    JsonValue::Object(Vec::new())
}

/// JSON string.
fn s(text: impl Into<String>) -> JsonValue {
    JsonValue::Str(text.into())
}

/// JSON integer.
fn n(value: impl std::fmt::Display) -> JsonValue {
    JsonValue::Number(value.to_string())
}

/// Dict-style assignment: replace an existing key, else append. The
/// The retired probe overwrites `data['files']` entries when the final
/// hashed pass covers a path recorded earlier without contents.
fn set(object: &mut JsonValue, key: &str, value: JsonValue) {
    if let JsonValue::Object(entries) = object {
        if let Some(slot) = entries.iter_mut().find(|(k, _)| k == key) {
            slot.1 = value;
            return;
        }
        entries.push((key.to_string(), value));
    }
}

/// Mutable access to a named child object, creating it when missing.
fn sub_mut<'a>(object: &'a mut JsonValue, key: &str) -> &'a mut JsonValue {
    if object.get(key).is_none() {
        set(object, key, obj());
    }
    if let JsonValue::Object(entries) = object {
        let index = entries.iter().position(|(k, _)| k == key).unwrap();
        return &mut entries[index].1;
    }
    unreachable!("snapshot document nodes are objects");
}

/// Snapshot one home's `.ssh` entries in sorted name order. Only
/// `config` and `known_hosts` export hashes; every other file exports
/// size, and `authorized_keys` is excluded (it is covered by the
/// accounts dump). `is_file`/`is_dir` follow symlinks exactly like the
/// owner's checks, and the entry itself fails the snapshot on a link.
fn snapshot_ssh_files(home: &Path) -> Result<Vec<(String, JsonValue)>, SnapshotFailure> {
    let ssh = home.join(".ssh");
    let mut names = Vec::new();
    for dir_entry in std::fs::read_dir(&ssh).map_err(SnapshotFailure::io)? {
        names.push(dir_entry.map_err(SnapshotFailure::io)?.file_name());
    }
    names.sort();
    let mut out = Vec::new();
    for name in names {
        let path = ssh.join(&name);
        let name = name.to_string_lossy().into_owned();
        if path.is_file() && name != "authorized_keys" {
            let contents = name == "config" || name == "known_hosts";
            out.push((name, Entry::snapshot(&path, contents)?.json()));
        } else if path.is_dir()
            && !std::fs::symlink_metadata(&path)
                .map_err(SnapshotFailure::io)?
                .file_type()
                .is_symlink()
            && name != "u08-personal-git"
        {
            out.push((name, Entry::snapshot(&path, false)?.json()));
        }
    }
    Ok(out)
}

fn glob_prefix(dir: &Path, prefix: &str) -> Result<Vec<PathBuf>, SnapshotFailure> {
    let mut hits = Vec::new();
    // A missing directory yields no matches, like `Path.glob`; other
    // read failures propagate like the retired probe's `OSError`.
    let read = match std::fs::read_dir(dir) {
        Ok(read) => read,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(hits),
        Err(error) => return Err(SnapshotFailure::io(error)),
    };
    for entry in read {
        let entry = entry.map_err(SnapshotFailure::io)?;
        if entry.file_name().as_bytes().starts_with(prefix.as_bytes()) {
            hits.push(entry.path());
        }
    }
    hits.sort();
    Ok(hits)
}

fn arg_list(args: &[&str]) -> Vec<String> {
    args.iter().map(|arg| arg.to_string()).collect()
}

/// Entry gate: root inside a container.
pub fn check_snapshot_gate(euid: u32, containerenv: &Path) -> Result<(), SnapshotFailure> {
    if euid != 0 || !containerenv.exists() {
        return Err(SnapshotFailure::assertion(""));
    }
    Ok(())
}

/// Project IP gate, like `assert ip_address(ip) in ip_network('10.89.0.0/24')`:
/// unparseable input is a `ValueError`, parsed-but-outside (v4 or v6) fails
/// the bare membership assert.
fn check_project_ip(ip: &str) -> Result<(), SnapshotFailure> {
    match ip.parse::<std::net::IpAddr>() {
        Ok(std::net::IpAddr::V4(v4)) => {
            let octets = v4.octets();
            if octets[0] == 10 && octets[1] == 89 && octets[2] == 0 {
                Ok(())
            } else {
                Err(SnapshotFailure::assertion(""))
            }
        }
        Ok(_) => Err(SnapshotFailure::assertion("")),
        Err(_) => Err(SnapshotFailure::bare(SnapshotKind::ValueError)),
    }
}

/// Collect the full snapshot document, like the retired probe's `main`.
pub fn run_snapshot() -> Result<JsonValue, SnapshotFailure> {
    check_snapshot_gate(unsafe { libc::geteuid() }, Path::new("/run/.containerenv"))?;
    let mut data = obj();
    set(&mut data, "files", obj());
    set(&mut data, "people", obj());
    set(&mut data, "git", obj());
    set(&mut data, "workloads", JsonValue::Array(Vec::new()));
    set(&mut data, "volumes", JsonValue::Array(Vec::new()));

    let mut paths = vec![
        "/etc/passwd".to_string(),
        "/etc/group".to_string(),
        "/etc/ssh/sshd_config.d/10-soda.conf".to_string(),
        "/etc/ssh/ssh_host_ed25519_key.pub".to_string(),
        "/etc/mise/config.toml".to_string(),
        "/etc/containers/containers.conf".to_string(),
        "/etc/containers/storage.conf".to_string(),
        "/etc/systemd/system/soda-podman.service".to_string(),
        "/usr/libexec/soda/project-init".to_string(),
        "/usr/libexec/soda/project-account".to_string(),
        "/etc/sudoers.d/soda-project".to_string(),
    ];
    if Path::new("/etc/systemd/system/soda-podman.socket").exists() {
        paths.push("/etc/systemd/system/soda-podman.socket".to_string());
    }
    for base in ["/etc/ssh/authorized_keys", "/var/lib/soda/accounts"] {
        if !Path::new(base).is_dir() {
            return Err(SnapshotFailure::assertion(""));
        }
        for file in list_files(Path::new(base))? {
            paths.push(file.to_string_lossy().into_owned());
        }
    }

    for login in ["u08-alice-8417", "u08-bob-8417"] {
        let home = Path::new("/home").join(login);
        if !home.exists() {
            continue;
        }
        let mut person = obj();
        set(
            &mut person,
            "identity",
            s(command(&arg_list(&["id", login]), &[])?),
        );
        set(&mut person, "home", Entry::snapshot(&home, true)?.json());
        set(
            &mut person,
            "shared",
            Entry::snapshot(&home.join("shared"), true)?.json(),
        );
        for (name, value) in snapshot_ssh_files(&home)? {
            set(&mut person, &name, value);
        }
        set(sub_mut(&mut data, "people"), login, person);

        let checkout = home.join("u08-personal-checkout");
        if checkout.exists() {
            // Only the known run-owned repository; export hashes, never content.
            let files = walk_sorted(&checkout)?;
            if files.len() > 10000 {
                return Err(SnapshotFailure::runtime("Checkout snapshot exceeded bound"));
            }
            for file in files {
                if has_git_part(&checkout, &file) {
                    continue;
                }
                let file_type = std::fs::symlink_metadata(&file)
                    .map_err(SnapshotFailure::io)?
                    .file_type();
                if file_type.is_file() || file_type.is_symlink() {
                    paths.push(file.to_string_lossy().into_owned());
                }
            }
            let checkout_arg = checkout.to_string_lossy().into_owned();
            let mut git = obj();
            set(
                &mut git,
                "head",
                s(command(
                    &arg_list(&[
                        "runuser",
                        "-u",
                        login,
                        "--",
                        "git",
                        "-C",
                        &checkout_arg,
                        "rev-parse",
                        "HEAD",
                    ]),
                    &[],
                )?),
            );
            set(
                &mut git,
                "status",
                s(command(
                    &arg_list(&[
                        "runuser",
                        "-u",
                        login,
                        "--",
                        "git",
                        "-C",
                        &checkout_arg,
                        "status",
                        "--porcelain=v1",
                    ]),
                    &[],
                )?),
            );
            set(
                &mut git,
                "refs",
                s(command(
                    &arg_list(&[
                        "runuser",
                        "-u",
                        login,
                        "--",
                        "git",
                        "-C",
                        &checkout_arg,
                        "show-ref",
                    ]),
                    &[],
                )?),
            );
            set(sub_mut(&mut data, "git"), login, git);
        }

        for private in [
            ".ssh/u08-personal-git/identity",
            ".config/soda-u08-workload/database-password",
        ] {
            let target = home.join(private);
            if target.exists() {
                set(
                    sub_mut(&mut data, "files"),
                    &target.to_string_lossy(),
                    Entry::snapshot(&target, false)?.json(),
                );
            }
        }
        for probe in glob_prefix(&home, "u08-access-")? {
            let file_type = std::fs::symlink_metadata(&probe)
                .map_err(SnapshotFailure::io)?
                .file_type();
            if !file_type.is_dir() || file_type.is_symlink() {
                return Err(SnapshotFailure::assertion(""));
            }
            for file in list_files(&probe)? {
                paths.push(file.to_string_lossy().into_owned());
            }
        }
    }

    let shared = Path::new("/srv/project/shared");
    paths.push(shared.to_string_lossy().into_owned());
    for probe in glob_prefix(shared, "u08-shared-")? {
        paths.push(probe.to_string_lossy().into_owned());
        paths.push(probe.join("members").to_string_lossy().into_owned());
    }
    let node = Path::new("/opt/mise/installs/node/24.20.0/bin/node");
    if node.exists() {
        paths.push(node.to_string_lossy().into_owned());
    }
    let unique: BTreeSet<String> = paths.into_iter().collect();
    for path in &unique {
        set(
            sub_mut(&mut data, "files"),
            path,
            Entry::snapshot(Path::new(path), true)?.json(),
        );
    }

    // Fixed project-local API. A second project without initialized workload
    // storage still gets complete account/rootfs coverage, not fabricated lists.
    let expect = std::env::var("SODA_EXPECT_WORKLOADS").unwrap_or_default();
    if expect != "0" && expect != "1" {
        return Err(SnapshotFailure::assertion(
            "Caller must declare required workload observations",
        ));
    }
    if expect == "1" {
        let podman = ["podman", "--url", "unix:///run/soda-podman/podman.sock"];
        let ids = output_lines(&command(
            &arg_list(&[podman[0], podman[1], podman[2], "ps", "-aq", "--no-trunc"]),
            &[],
        )?);
        if ids.len() < 2 {
            return Err(SnapshotFailure::assertion(
                "Missing required workload containers",
            ));
        }
        let mut sorted_ids = ids;
        sorted_ids.sort();
        let mut workloads = Vec::new();
        for identifier in &sorted_ids {
            workloads.push(s(command(
                &arg_list(&[
                    podman[0],
                    podman[1],
                    podman[2],
                    "container",
                    "inspect",
                    "--format",
                    "{{json .ID}} {{json .Name}} {{json .Image}} {{json .HostConfig.NetworkMode}} {{json .Mounts}}",
                    identifier,
                ]),
                &[],
            )?));
        }
        set(&mut data, "workloads", JsonValue::Array(workloads));
        let mut volumes = output_lines(&command(
            &arg_list(&[
                podman[0],
                podman[1],
                podman[2],
                "volume",
                "ls",
                "--format",
                "{{.Name}}",
            ]),
            &[],
        )?);
        volumes.sort();
        set(
            &mut data,
            "volumes",
            JsonValue::Array(volumes.into_iter().map(s).collect()),
        );
        let names = output_lines(&command(
            &arg_list(&[
                podman[0],
                podman[1],
                podman[2],
                "ps",
                "--format",
                "{{.Names}}",
            ]),
            &[],
        )?);
        let databases: Vec<String> = names
            .into_iter()
            .filter(|name| name == "u08-projectnet-database" || name == "workload_database_1")
            .collect();
        if databases.is_empty() {
            return Err(SnapshotFailure::assertion(
                "No running database; restore existing workload before snapshot",
            ));
        }
        let ip = std::env::var("SODA_PROJECT_IP")
            .map_err(|_| SnapshotFailure::bare(SnapshotKind::KeyError))?;
        check_project_ip(&ip)?;
        let mut passfiles =
            glob_prefix(Path::new("/home/u08-alice-8417/.config"), "u08-db-client-")?
                .into_iter()
                .map(|dir| dir.join("pgpass"))
                .filter(|path| path.exists())
                .collect::<Vec<PathBuf>>();
        passfiles.sort();
        if passfiles.is_empty() {
            return Err(SnapshotFailure::assertion(
                "Missing native client credential input",
            ));
        }
        let passfile = passfiles.pop().unwrap();
        let link_type = std::fs::symlink_metadata(&passfile)
            .map_err(SnapshotFailure::io)?
            .file_type();
        if link_type.is_symlink() {
            return Err(SnapshotFailure::assertion(""));
        }
        let pass_meta = std::fs::metadata(&passfile).map_err(SnapshotFailure::io)?;
        use std::os::unix::fs::MetadataExt;
        if pass_meta.mode() & 0o077 != 0 {
            return Err(SnapshotFailure::assertion(""));
        }
        let cached = std::fs::read_to_string(&passfile).map_err(SnapshotFailure::io)?;
        let endpoint = cached.split(':').next().unwrap_or("");
        if endpoint != ip {
            return Err(SnapshotFailure::assertion(
                "Cached credential endpoint differs from live native target",
            ));
        }
        let mut database = obj();
        for name in &databases {
            // Native TCP client, also used by both real developers; no dependency
            // on exec into a different-UID workload and no credential export.
            let rows = command(
                &arg_list(&[
                    "psql",
                    "-X",
                    "-w",
                    "-h",
                    &ip,
                    "-p",
                    "5432",
                    "-U",
                    "developer",
                    "-d",
                    "soda_example",
                    "-At",
                    "-c",
                    "SELECT run_id,value FROM soda_u08_probe ORDER BY run_id",
                ]),
                &[
                    ("PGPASSFILE", &passfile.to_string_lossy()),
                    ("PGCONNECT_TIMEOUT", "5"),
                ],
            )?;
            if rows.is_empty() {
                return Err(SnapshotFailure::assertion(
                    "Empty required database snapshot",
                ));
            }
            set(&mut database, name, s(rows));
        }
        set(&mut data, "database", database);
    }
    Ok(data)
}

#[cfg(test)]
mod tests;
