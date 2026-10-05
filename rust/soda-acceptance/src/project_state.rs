//! Bounded, read-only U08 state snapshot, mirroring
//! `tests/installed/project-state.py`.
//!
//! Run as root INSIDE a selected project. No
//! environment/secret/DB-credential/shadow/private-key contents are
//! exported. The driver supplies the compiled probe over the selected
//! administrator SSH session.
//!
//! Failure taxonomy mirrors the Python owner: `RuntimeError` and
//! `AssertionError` keep their detail text, every other kind renders
//! with an empty detail.

use std::collections::BTreeSet;
use std::io::Read;
use std::os::unix::ffi::OsStrExt;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

use soda_json::JsonValue;

use crate::sha256::{self, Sha256};

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
/// detail, like the Python owner's handler.
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
            // the owner's strict `read_text`/`decode` calls.
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
/// Python owner overwrites `data['files']` entries when the final
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

/// Run a snapshot command with piped output and a timeout, returning
/// stripped stdout. Stderr is drained and discarded, like the Python
/// owner capturing it and never reading it.
pub fn command_with_timeout(
    argv: &[String],
    extra_env: &[(&str, &str)],
    timeout: Duration,
) -> Result<String, SnapshotFailure> {
    let mut child = Command::new(&argv[0])
        .args(&argv[1..])
        .envs(extra_env.iter().copied())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(SnapshotFailure::io)?;
    // Both pipes drain on threads while the parent polls, like
    // `communicate()`: waiting before reading would deadlock once a pipe
    // fills.
    let mut stdout_pipe = child.stdout.take();
    let out_reader = std::thread::spawn(move || {
        let mut stdout = Vec::new();
        if let Some(pipe) = stdout_pipe.take() {
            let _ = std::io::BufReader::new(pipe).read_to_end(&mut stdout);
        }
        stdout
    });
    let mut stderr = child.stderr.take();
    let drain = std::thread::spawn(move || {
        if let Some(stderr) = stderr.take() {
            let mut sink = Vec::new();
            let _ = std::io::BufReader::new(stderr).read_to_end(&mut sink);
        }
    });
    let deadline = Instant::now() + timeout;
    loop {
        match child.try_wait().map_err(SnapshotFailure::io)? {
            Some(status) => {
                let stdout = out_reader.join().unwrap_or_default();
                let _ = drain.join();
                if !status.success() {
                    let shown: Vec<&str> = argv.iter().take(5).map(String::as_str).collect();
                    return Err(SnapshotFailure::runtime(format!(
                        "Required snapshot command failed: {}",
                        shown.join(" ")
                    )));
                }
                if stdout.len() > 4 * 1024 * 1024 {
                    return Err(SnapshotFailure::runtime("Snapshot output exceeded bound"));
                }
                let text = String::from_utf8(stdout)
                    .map_err(|_| SnapshotFailure::bare(SnapshotKind::UnicodeDecodeError))?;
                return Ok(text.trim().to_string());
            }
            None if Instant::now() >= deadline => {
                let _ = child.kill();
                let _ = child.wait();
                let _ = drain.join();
                return Err(SnapshotFailure::bare(SnapshotKind::TimeoutExpired));
            }
            None => std::thread::sleep(Duration::from_millis(10)),
        }
    }
}

/// Snapshot command with the owner's 30s timeout and inherited environment.
pub fn command(argv: &[String], extra_env: &[(&str, &str)]) -> Result<String, SnapshotFailure> {
    command_with_timeout(argv, extra_env, Duration::from_secs(30))
}

/// Split stripped command output into lines. Empty output yields no
/// lines, like the Python owner's `strip().splitlines()`.
pub fn output_lines(text: &str) -> Vec<String> {
    if text.is_empty() {
        return Vec::new();
    }
    text.lines().map(|line| line.to_string()).collect()
}

/// Snapshot entry payload beyond uid/gid/mode.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum EntryBody {
    /// File content hash.
    Sha256(String),
    /// File size without content access.
    Size(u64),
    /// Directory: identity only.
    Dir,
}

/// One snapshotted path.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Entry {
    uid: u32,
    gid: u32,
    mode: u32,
    body: EntryBody,
}

impl Entry {
    /// Snapshot one path via `lstat`, like the Python owner's `entry`.
    pub fn snapshot(path: &Path, contents: bool) -> Result<Entry, SnapshotFailure> {
        use std::os::unix::fs::MetadataExt;
        let meta = std::fs::symlink_metadata(path).map_err(SnapshotFailure::io)?;
        let file_type = meta.file_type();
        let entry = Entry {
            uid: meta.uid(),
            gid: meta.gid(),
            mode: meta.mode() & 0o7777,
            body: EntryBody::Dir,
        };
        // Fail closed on links, like the owner: `lstat` sees neither a
        // regular file nor a directory, so the snapshot aborts instead of
        // following or recording an administrator's unexpected link.
        if file_type.is_symlink() {
            return Err(SnapshotFailure::bare(SnapshotKind::FileNotFoundError));
        }
        if file_type.is_file() && contents {
            if meta.len() > 512 * 1024 * 1024 {
                return Err(SnapshotFailure::runtime("Snapshot file too large"));
            }
            let mut file = std::fs::File::open(path).map_err(SnapshotFailure::io)?;
            let mut digest = Sha256::new();
            let mut chunk = vec![0u8; 1024 * 1024];
            loop {
                let n = file.read(&mut chunk).map_err(SnapshotFailure::io)?;
                if n == 0 {
                    break;
                }
                digest.update(&chunk[..n]);
            }
            return Ok(Entry {
                body: EntryBody::Sha256(sha256::hex_lower(&digest.finalize())),
                ..entry
            });
        }
        if file_type.is_file() {
            return Ok(Entry {
                body: EntryBody::Size(meta.len()),
                ..entry
            });
        }
        if !file_type.is_dir() {
            return Err(SnapshotFailure::runtime("Unsupported snapshot file"));
        }
        Ok(entry)
    }

    /// Render as the owner's JSON object.
    pub fn json(&self) -> JsonValue {
        let mut object = obj();
        set(&mut object, "uid", n(self.uid));
        set(&mut object, "gid", n(self.gid));
        set(&mut object, "mode", n(self.mode));
        match &self.body {
            EntryBody::Sha256(hash) => set(&mut object, "sha256", s(hash.clone())),
            EntryBody::Size(size) => set(&mut object, "size", n(size)),
            EntryBody::Dir => {}
        }
        object
    }
}

/// Python `json.dumps(sort_keys=True)` rendering: sorted keys,
/// `(', ', ': ')` separators, ASCII-only output with lowercase `\u`
/// escapes. Snapshot values are strings, integers, lists, and objects.
pub fn dumps_sorted(value: &JsonValue) -> String {
    fn escape(text: &str, out: &mut String) {
        out.push('"');
        for ch in text.chars() {
            match ch {
                '"' => out.push_str("\\\""),
                '\\' => out.push_str("\\\\"),
                '\n' => out.push_str("\\n"),
                '\r' => out.push_str("\\r"),
                '\t' => out.push_str("\\t"),
                '\u{08}' => out.push_str("\\b"),
                '\u{0c}' => out.push_str("\\f"),
                c if (c as u32) < 0x20 => {
                    out.push_str(&format!("\\u{:04x}", c as u32));
                }
                c if (c as u32) < 0x7f => out.push(c),
                c if (c as u32) < 0x10000 => {
                    out.push_str(&format!("\\u{:04x}", c as u32));
                }
                c => {
                    // ensure_ascii astral planes as lowercase surrogate pairs.
                    let code = c as u32 - 0x10000;
                    out.push_str(&format!(
                        "\\u{:04x}\\u{:04x}",
                        0xd800 + (code >> 10),
                        0xdc00 + (code & 0x3ff)
                    ));
                }
            }
        }
        out.push('"');
    }

    fn render(value: &JsonValue, out: &mut String) {
        match value {
            JsonValue::Null => out.push_str("null"),
            JsonValue::Bool(true) => out.push_str("true"),
            JsonValue::Bool(false) => out.push_str("false"),
            JsonValue::Number(raw) => out.push_str(raw),
            JsonValue::Str(text) => escape(text, out),
            JsonValue::Array(items) => {
                out.push('[');
                for (index, item) in items.iter().enumerate() {
                    if index > 0 {
                        out.push_str(", ");
                    }
                    render(item, out);
                }
                out.push(']');
            }
            JsonValue::Object(entries) => {
                let mut order: Vec<&(String, JsonValue)> = entries.iter().collect();
                order.sort_by(|a, b| a.0.cmp(&b.0));
                out.push('{');
                for (index, (key, item)) in order.iter().enumerate() {
                    if index > 0 {
                        out.push_str(", ");
                    }
                    escape(key, out);
                    out.push_str(": ");
                    render(item, out);
                }
                out.push('}');
            }
        }
    }

    let mut out = String::new();
    render(value, &mut out);
    out
}

fn list_files(dir: &Path) -> Result<Vec<PathBuf>, SnapshotFailure> {
    let mut files = Vec::new();
    for entry in std::fs::read_dir(dir).map_err(SnapshotFailure::io)? {
        let path = entry.map_err(SnapshotFailure::io)?.path();
        // `Path.is_file` follows symlinks, like the owner.
        if path.is_file() {
            files.push(path);
        }
    }
    files.sort();
    Ok(files)
}

fn has_git_part(checkout: &Path, path: &Path) -> bool {
    path.strip_prefix(checkout)
        .map(|relative| {
            relative
                .components()
                .any(|part| part.as_os_str().as_bytes() == b".git")
        })
        .unwrap_or(false)
}

fn walk_sorted(root: &Path) -> Result<Vec<PathBuf>, SnapshotFailure> {
    let mut found = Vec::new();
    let mut stack = vec![root.to_path_buf()];
    while let Some(dir) = stack.pop() {
        let mut entries: Vec<PathBuf> = Vec::new();
        for entry in std::fs::read_dir(&dir).map_err(SnapshotFailure::io)? {
            entries.push(entry.map_err(SnapshotFailure::io)?.path());
        }
        entries.sort();
        for path in entries {
            found.push(path.clone());
            let file_type = std::fs::symlink_metadata(&path)
                .map_err(SnapshotFailure::io)?
                .file_type();
            if file_type.is_dir() && !file_type.is_symlink() {
                stack.push(path);
            }
        }
    }
    found.sort();
    Ok(found)
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
    // read failures propagate like the owner's `OSError`.
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

/// Collect the full snapshot document, like the Python owner's `main`.
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
mod tests {
    use super::*;
    use crate::files::TempDir;

    /// Port of `TestEntryHashAndMode` from `test_u08_state.py`.
    #[test]
    fn entry_hashes_file_and_reports_mode() {
        use std::os::unix::fs::PermissionsExt;
        let dir = TempDir::new("snapshot").unwrap();
        let path = dir.path().join("probe.txt");
        std::fs::write(&path, b"snapshot-me").unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o640)).unwrap();
        let json = Entry::snapshot(&path, true).unwrap().json();
        assert_eq!(json.get("mode").unwrap().as_integer().unwrap(), 0o640);
        assert_eq!(
            json.get("uid").unwrap().as_integer().unwrap(),
            unsafe { libc::getuid() } as i128
        );
        assert_eq!(
            json.get("gid").unwrap().as_integer().unwrap(),
            unsafe { libc::getgid() } as i128
        );
        let mut digest = Sha256::new();
        digest.update(b"snapshot-me");
        assert_eq!(
            json.get("sha256").unwrap().as_str().unwrap(),
            sha256::hex_lower(&digest.finalize())
        );
        assert!(json.get("size").is_none());
        assert!(json.get("link").is_none());
    }

    /// Port of `TestEntrySizeWithoutContent` from `test_u08_state.py`.
    #[test]
    fn entry_reports_size_without_hashing() {
        let dir = TempDir::new("snapshot").unwrap();
        let path = dir.path().join("secret.bin");
        std::fs::write(&path, b"0123456789abcdef").unwrap();
        let json = Entry::snapshot(&path, false).unwrap().json();
        assert_eq!(json.get("size").unwrap().as_integer().unwrap(), 16);
        assert!(json.get("sha256").is_none());
    }

    /// Port of `TestEntryRejectsSpecial` from `test_u08_state.py`.
    #[test]
    fn entry_rejects_special_files() {
        let err = Entry::snapshot(Path::new("/dev/null"), true).unwrap_err();
        assert_eq!(err.kind, SnapshotKind::RuntimeError);
        assert_eq!(err.detail, "Unsupported snapshot file");
    }

    /// The owner fails closed on links and missing paths instead of
    /// following them or substituting emptiness.
    #[test]
    fn entry_fails_closed_on_links_and_missing_paths() {
        let dir = TempDir::new("snapshot").unwrap();
        let target = dir.path().join("target.txt");
        std::fs::write(&target, b"data").unwrap();
        let link = dir.path().join("link.txt");
        std::os::unix::fs::symlink(&target, &link).unwrap();
        let err = Entry::snapshot(&link, true).unwrap_err();
        assert_eq!(err.kind, SnapshotKind::FileNotFoundError);
        let err = Entry::snapshot(&dir.path().join("missing"), false).unwrap_err();
        assert_eq!(err.kind, SnapshotKind::FileNotFoundError);
        // A changed file changes its hash.
        let first = Entry::snapshot(&target, true).unwrap();
        std::fs::write(&target, b"changed").unwrap();
        let second = Entry::snapshot(&target, true).unwrap();
        assert_ne!(first.json(), second.json());
    }

    /// The `.ssh` loop exports hashes only for `config`/`known_hosts`,
    /// sizes for the rest, skips `authorized_keys`/`u08-personal-git`,
    /// and fails the snapshot on a link.
    #[test]
    fn ssh_files_export_hashes_sizes_and_reject_links() {
        let dir = TempDir::new("snapshot").unwrap();
        let home = dir.path().join("home");
        let ssh = home.join(".ssh");
        std::fs::create_dir_all(&ssh).unwrap();
        std::fs::write(ssh.join("config"), b"Host x\n").unwrap();
        std::fs::write(ssh.join("identity"), b"PRIVATE").unwrap();
        std::fs::write(ssh.join("authorized_keys"), b"ssh-ed25519 AAAA\n").unwrap();
        std::fs::create_dir_all(ssh.join("u08-personal-git")).unwrap();
        let entries = snapshot_ssh_files(&home).unwrap();
        let names: Vec<&str> = entries.iter().map(|(name, _)| name.as_str()).collect();
        assert_eq!(names, vec!["config", "identity"]);
        let config = entries[0].1.clone();
        assert!(config.get("sha256").is_some());
        assert!(config.get("size").is_none());
        let identity = entries[1].1.clone();
        assert_eq!(identity.get("size").unwrap().as_integer().unwrap(), 7);
        assert!(identity.get("sha256").is_none());
        std::os::unix::fs::symlink(ssh.join("config"), ssh.join("alias")).unwrap();
        let err = snapshot_ssh_files(&home).unwrap_err();
        assert_eq!(err.kind, SnapshotKind::FileNotFoundError);
    }

    #[test]
    fn snapshot_gate_requires_root_container() {
        let dir = TempDir::new("snapshot").unwrap();
        let marker = dir.path().join(".containerenv");
        std::fs::write(&marker, b"").unwrap();
        assert!(check_snapshot_gate(0, &marker).is_ok());
        assert_eq!(
            check_snapshot_gate(1000, &marker).unwrap_err(),
            SnapshotFailure::assertion("")
        );
        assert_eq!(
            check_snapshot_gate(0, &dir.path().join("missing")).unwrap_err(),
            SnapshotFailure::assertion("")
        );
    }

    #[test]
    fn snapshot_command_reports_failures() {
        let ok = command(&arg_list(&["echo", "  hi  "]), &[]).unwrap();
        assert_eq!(ok, "hi");
        let err = command(&arg_list(&["false"]), &[]).unwrap_err();
        assert_eq!(err.kind, SnapshotKind::RuntimeError);
        assert_eq!(err.detail, "Required snapshot command failed: false");
        let err = command_with_timeout(&arg_list(&["sleep", "5"]), &[], Duration::from_millis(100))
            .unwrap_err();
        assert_eq!(err.kind, SnapshotKind::TimeoutExpired);
        let err = command(&arg_list(&["head", "-c", "5000000", "/dev/zero"]), &[]).unwrap_err();
        assert_eq!(err.kind, SnapshotKind::RuntimeError);
        assert_eq!(err.detail, "Snapshot output exceeded bound");
    }

    #[test]
    fn dumps_sorted_matches_python_separators() {
        let mut value = obj();
        set(&mut value, "b", n(1));
        set(
            &mut value,
            "a",
            JsonValue::Array(vec![JsonValue::Bool(true), s("x\ny")]),
        );
        assert_eq!(dumps_sorted(&value), r#"{"a": [true, "x\ny"], "b": 1}"#);
        let mut unicode = obj();
        set(&mut unicode, "e", s("é💾"));
        assert_eq!(dumps_sorted(&unicode), "{\"e\": \"\\u00e9\\ud83d\\udcbe\"}");
    }

    #[test]
    fn project_network_check_is_strict() {
        assert!(check_project_ip("10.89.0.7").is_ok());
        assert_eq!(
            check_project_ip("10.89.1.7").unwrap_err().kind,
            SnapshotKind::AssertionError
        );
        assert_eq!(
            check_project_ip("not-an-ip").unwrap_err().kind,
            SnapshotKind::ValueError
        );
        assert_eq!(
            check_project_ip("10.89.0.256").unwrap_err().kind,
            SnapshotKind::ValueError
        );
        assert_eq!(
            check_project_ip("10.89.0.07").unwrap_err().kind,
            SnapshotKind::ValueError
        );
        assert_eq!(
            check_project_ip("::1").unwrap_err().kind,
            SnapshotKind::AssertionError
        );
    }
}
