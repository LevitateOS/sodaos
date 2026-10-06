//! Terminal lifecycle (port of the `TERMINALS` half of
//! `project_terminal.py`: reserve/create/attach/inspect/list/end/rename plus
//! `prepare`).
//!
//! Merge-safe: uses only the brief-pinned `sys`/`fs`/`timex` signatures plus
//! the scaffold (`proto`/`pyemit`/`b64`/`sha`), `libc`, and `soda_json`.
//!
//! Two deliberate deltas from the `.py`, both documented at their sites:
//! 1. `reserve` streams the whole `project-terminal` binary for the
//!    `source_hash` check (no 64KB cap — the binary replaces the script).
//! 2. `String` errors cannot separate `NotFound` from other I/O failures, so
//!    absence checks use an explicit existence probe first. Every such site
//!    runs under the TERMINALS parent lock, where concurrent mutation (the
//!    only way the probe could go stale) is impossible.

use std::collections::HashSet;
use std::fs::File;
use std::io;
use std::os::unix::io::AsRawFd;

use soda_json::JsonValue;

use crate::account::{self, Account};
use crate::fs;
use crate::proto;
use crate::pty;
use crate::pyemit;
use crate::sha;
use crate::svc;
use crate::sys;
use crate::timex;

pub const TERMINALS: &str = "/run/soda-terminals";
pub const PROGRAM: &str = "/usr/libexec/soda/project-terminal";

fn s_isreg(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFREG
}

fn s_issock(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFSOCK
}

pub const TMUX_CONFIG: &str = "set -g status off
set -g history-limit 10000
set -s buffer-limit 10
set -s set-clipboard off
set -s escape-time 10
set -g default-terminal screen-256color
set -g update-environment \"\"
set -s exit-unattached off
";

/// Validated terminal locator path.
pub fn terminal_path(identifier: &str) -> Result<String, String> {
    if !proto::valid_identifier(identifier) {
        return Err("invalid terminal identifier".to_string());
    }
    Ok(format!("{TERMINALS}/{identifier}"))
}

/// No-follow descent with the `.py` `root_directory` checks (root-owned,
/// group/other write-free) at every level.
pub(crate) fn checked_chain(top: &str) -> Result<File, String> {
    let mut current = sys::open_root().map_err(|e| format!("open /: {e}"))?;
    for part in top.split('/').filter(|p| !p.is_empty()) {
        let child = sys::open_child_dir(&current, part).map_err(|e| format!("open {part}: {e}"))?;
        drop(current);
        current = child;
        let (uid, mode) = fs::fstat_uid_mode(&current).map_err(|e| format!("stat {part}: {e}"))?;
        if uid != 0 || mode & 0o022 != 0 {
            return Err("unsafe terminal directory".to_string());
        }
    }
    Ok(current)
}

/// Directory entries through the held fd.
fn list_dir_names(dir: &File) -> Result<Vec<String>, String> {
    let mut names = Vec::new();
    let entries = std::fs::read_dir(format!("/proc/self/fd/{}", dir.as_raw_fd()))
        .map_err(|e| e.to_string())?;
    for entry in entries {
        let entry = entry.map_err(|e| e.to_string())?;
        names.push(
            entry
                .file_name()
                .into_string()
                .map_err(|_| "terminal name encoding".to_string())?,
        );
    }
    Ok(names)
}

fn fstatat(dir: &File, name: &str) -> io::Result<libc::stat> {
    let target = std::ffi::CString::new(name)
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidInput, "nul byte in name"))?;
    let mut info: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe {
        libc::fstatat(
            dir.as_raw_fd(),
            target.as_ptr(),
            &mut info,
            libc::AT_SYMLINK_NOFOLLOW,
        )
    } != 0
    {
        return Err(io::Error::last_os_error());
    }
    Ok(info)
}

pub(crate) fn record_exists(dir: &File, name: &str) -> Result<bool, String> {
    match fstatat(dir, name) {
        Ok(_) => Ok(true),
        Err(err) if err.kind() == io::ErrorKind::NotFound => Ok(false),
        Err(err) => Err(err.to_string()),
    }
}

fn write_stdout_best_effort(bytes: &[u8]) {
    let mut rest = bytes;
    while !rest.is_empty() {
        let wrote = unsafe { libc::write(1, rest.as_ptr() as *const libc::c_void, rest.len()) };
        if wrote < 0 {
            if io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            break;
        }
        if wrote == 0 {
            break;
        }
        rest = &rest[wrote as usize..];
    }
}

fn closed_launch_failed() {
    write_stdout_best_effort(&pty::closed_line("launch_failed"));
}

// ---------------------------------------------------------------------------
// Binding + reservation records.
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BindingRecord {
    pub login: String,
    pub uid: u32,
    pub gid: u32,
    pub home: String,
    pub shell: String,
    pub identity: i64,
    pub cols: i64,
    pub rows: i64,
    pub created_at: i64,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Reservation {
    pub expires: i64,
    pub scope: String,
}

/// Validate a `binding` record (dup-tolerant like `json.loads`, unlike the
/// control-frame decoder). Uid/gid widen to `u32` at most: larger JSON ints
/// error here instead of failing later comparisons, a deliberate hardening
/// delta confined to corrupt root-owned files.
pub fn validate_binding(value: &JsonValue) -> Result<BindingRecord, String> {
    let entries = match value {
        JsonValue::Object(entries) => entries,
        _ => return Err("terminal binding".to_string()),
    };
    let keys: HashSet<&str> = entries.iter().map(|(k, _)| k.as_str()).collect();
    let want: HashSet<&str> = ["account", "identity", "cols", "rows", "created_at"]
        .into_iter()
        .collect();
    if keys != want {
        return Err("terminal binding".to_string());
    }
    let get = |key: &str| value.get(key).ok_or_else(|| "terminal binding".to_string());
    let items = match get("account")? {
        JsonValue::Array(items) if items.len() == 5 => items,
        _ => return Err("terminal binding values".to_string()),
    };
    let login = items[0]
        .as_str()
        .ok_or_else(|| "terminal binding values".to_string())?;
    if !account::valid_login(login) || login == "root" {
        return Err("terminal binding values".to_string());
    }
    let uid = items[1]
        .as_integer()
        .and_then(|n| u32::try_from(n).ok())
        .filter(|n| *n > 0)
        .ok_or_else(|| "terminal binding values".to_string())?;
    let gid = items[2]
        .as_integer()
        .and_then(|n| u32::try_from(n).ok())
        .ok_or_else(|| "terminal binding values".to_string())?;
    let home = items[3]
        .as_str()
        .ok_or_else(|| "terminal binding values".to_string())?;
    let shell = items[4]
        .as_str()
        .ok_or_else(|| "terminal binding values".to_string())?;
    if !home.starts_with('/') || !shell.starts_with('/') {
        return Err("terminal binding values".to_string());
    }
    let identity = pyemit::as_int(get("identity")?)
        .filter(|n| *n > 0)
        .ok_or_else(|| "terminal binding values".to_string())?;
    let cols = pyemit::as_int(get("cols")?).ok_or_else(|| "terminal binding values".to_string())?;
    let rows = pyemit::as_int(get("rows")?).ok_or_else(|| "terminal binding values".to_string())?;
    if !proto::dimensions(cols, rows) {
        return Err("terminal binding values".to_string());
    }
    let created_at =
        pyemit::as_int(get("created_at")?).ok_or_else(|| "terminal binding values".to_string())?;
    if !(1..=9007199254740991).contains(&created_at) {
        return Err("terminal binding values".to_string());
    }
    Ok(BindingRecord {
        login: login.to_string(),
        uid,
        gid,
        home: home.to_string(),
        shell: shell.to_string(),
        identity,
        cols,
        rows,
        created_at,
    })
}

pub fn binding_matches_account(record: &BindingRecord, account: &Account) -> bool {
    record.login == account.pw_name
        && record.uid == account.pw_uid
        && record.gid == account.pw_gid
        && record.home == account.pw_dir
        && record.shell == account.pw_shell
}

/// Read and validate the binding, optionally checking the account match.
pub fn binding_record(
    dir: &File,
    account: Option<&Account>,
    identity: i64,
) -> Result<BindingRecord, String> {
    let record = validate_binding(&fs::read_record(dir, "binding")?)?;
    if let Some(held) = account {
        // Exact `.py` comparison: stored vector vs `account_binding`.
        let stored = JsonValue::Array(vec![
            JsonValue::Str(record.login.clone()),
            JsonValue::Number(record.uid.to_string()),
            JsonValue::Number(record.gid.to_string()),
            JsonValue::Str(record.home.clone()),
            JsonValue::Str(record.shell.clone()),
        ]);
        if record.identity != identity || stored != account::account_binding(held) {
            return Err("terminal account changed".to_string());
        }
    }
    Ok(record)
}

/// Validate a `reservation` record.
pub fn validate_reservation(value: &JsonValue) -> Result<Reservation, String> {
    let entries = match value {
        JsonValue::Object(entries) => entries,
        _ => return Err("invalid creation reservation".to_string()),
    };
    let keys: HashSet<&str> = entries.iter().map(|(k, _)| k.as_str()).collect();
    let want: HashSet<&str> = ["expires", "scope"].into_iter().collect();
    if keys != want {
        return Err("invalid creation reservation".to_string());
    }
    let expires = value
        .get("expires")
        .and_then(pyemit::as_int)
        .filter(|n| *n > 0)
        .ok_or_else(|| "invalid creation reservation".to_string())?;
    let scope = value
        .get("scope")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "invalid creation reservation".to_string())?;
    if !proto::valid_scope(scope) {
        return Err("invalid creation reservation".to_string());
    }
    Ok(Reservation {
        expires,
        scope: scope.to_string(),
    })
}

/// Read the reservation, if any (missing file → `None`).
fn read_reservation(dir: &File) -> Result<Option<Reservation>, String> {
    if !record_exists(dir, "reservation")? {
        return Ok(None);
    }
    validate_reservation(&fs::read_record(dir, "reservation")?).map(Some)
}

fn permit_live(permit: Option<&Reservation>) -> bool {
    permit
        .map(|p| p.expires > timex::now_secs())
        .unwrap_or(false)
}

// ---------------------------------------------------------------------------
// Status classification.
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum UnitClass {
    Ended,
    Opening,
    Ending,
    ReadyCheck,
}

/// Pure half of `classify_unit_state`: the caller checks cgroup emptiness for
/// stopped units first, preserving the `.py` error order.
pub fn classify_bare_state(
    state: &str,
    permit_present: bool,
    live_permit: bool,
) -> Result<UnitClass, String> {
    if state == "inactive" || state == "failed" {
        return Ok(if live_permit {
            UnitClass::Opening
        } else {
            UnitClass::Ended
        });
    }
    if permit_present {
        return Err("unconsumed reservation has a native unit".to_string());
    }
    match state {
        "active" => Ok(UnitClass::ReadyCheck),
        "activating" => Ok(UnitClass::Opening),
        "deactivating" => Ok(UnitClass::Ending),
        _ => Err("terminal state unavailable".to_string()),
    }
}

/// Status object in `.py` key order: id, name, created_at, ready, attached,
/// state.
pub fn status_object(
    identifier: &str,
    name: &str,
    created_at: i64,
    ready: bool,
    attached: bool,
    state: &str,
) -> JsonValue {
    JsonValue::Object(vec![
        ("id".to_string(), JsonValue::Str(identifier.to_string())),
        ("name".to_string(), JsonValue::Str(name.to_string())),
        (
            "created_at".to_string(),
            JsonValue::Number(created_at.to_string()),
        ),
        ("ready".to_string(), JsonValue::Bool(ready)),
        ("attached".to_string(), JsonValue::Bool(attached)),
        ("state".to_string(), JsonValue::Str(state.to_string())),
    ])
}

fn status_state(value: &JsonValue) -> Result<&str, String> {
    value
        .get("state")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "terminal status".to_string())
}

/// `(pid, dev, ino)` from a `ready` record.
fn parse_ready(value: &JsonValue) -> Result<(i32, u64, u64), String> {
    let pid = value
        .get("pid")
        .and_then(pyemit::as_int)
        .ok_or_else(|| "terminal ready".to_string())?;
    let pid = i32::try_from(pid).map_err(|_| "terminal ready".to_string())?;
    let socket = match value.get("socket") {
        Some(JsonValue::Array(items)) if items.len() == 2 => items,
        _ => return Err("terminal ready".to_string()),
    };
    let dev = socket[0]
        .as_integer()
        .and_then(|n| u64::try_from(n).ok())
        .ok_or_else(|| "terminal ready".to_string())?;
    let ino = socket[1]
        .as_integer()
        .and_then(|n| u64::try_from(n).ok())
        .ok_or_else(|| "terminal ready".to_string())?;
    Ok((pid, dev, ino))
}

/// True while another attach holds the writer lock.
fn writer_attached(directory: &File) -> Result<bool, String> {
    let writer = fs::root_file(directory, "writer", true)?;
    let locked = sys::flock_exclusive_nb(&writer).map_err(|e| e.to_string())?;
    Ok(!locked)
}

fn observe_ready_terminal(path: &str, directory: &File, account: &Account) -> Result<bool, String> {
    let ready = parse_ready(&fs::read_record(directory, "ready")?)?;
    let screen = checked_chain(&format!("{path}/screen"))?;
    drop(screen);
    let sock = format!("{path}/screen/socket");
    if svc::socket_identity(&sock, account, ready.0)? != (ready.1, ready.2) {
        return Err("terminal socket changed".to_string());
    }
    writer_attached(directory)
}

fn classify_unit_state(
    path: &str,
    directory: &File,
    account: &Account,
    identifier: &str,
    state: &str,
    permit: Option<&Reservation>,
) -> Result<(String, bool), String> {
    if (state == "inactive" || state == "failed") && !svc::cgroup_empty(identifier)? {
        return Err("terminal cleanup unconfirmed".to_string());
    }
    match classify_bare_state(state, permit.is_some(), permit_live(permit))? {
        UnitClass::Ended => Ok(("ended".to_string(), false)),
        UnitClass::Opening => Ok(("opening".to_string(), false)),
        UnitClass::Ending => Ok(("ending".to_string(), false)),
        UnitClass::ReadyCheck => {
            let attached = observe_ready_terminal(path, directory, account)?;
            Ok(("ready".to_string(), attached))
        }
    }
}

/// Current status, or `None` for an observed absence.
pub fn terminal_status(
    identifier: &str,
    account: &Account,
    identity: i64,
) -> Result<Option<JsonValue>, String> {
    let path = terminal_path(identifier)?;
    // Absence probe first: `checked_chain` cannot report `NotFound` through
    // `String`, and this always runs under the parent lock.
    match std::fs::symlink_metadata(&path) {
        Err(err) if err.kind() == io::ErrorKind::NotFound => {
            let state = svc::service_state(identifier, None)?;
            if (state != "inactive" && state != "failed") || !svc::cgroup_empty(identifier)? {
                return Err("unidentified native terminal".to_string());
            }
            return Ok(None);
        }
        Err(err) => return Err(err.to_string()),
        Ok(_) => {}
    }
    let directory = checked_chain(&path)?;
    let record = binding_record(&directory, Some(account), identity)?;
    let permit = read_reservation(&directory)?;
    let state = svc::service_state(identifier, Some(account))?;
    let (state, attached) = classify_unit_state(
        &path,
        &directory,
        account,
        identifier,
        &state,
        permit.as_ref(),
    )?;
    let name = fs::read_record(&directory, "name")?;
    let name = name.as_str().ok_or_else(|| "terminal name".to_string())?;
    if !proto::valid_name(name) {
        return Err("terminal name".to_string());
    }
    Ok(Some(status_object(
        identifier,
        name,
        record.created_at,
        state == "ready",
        attached,
        &state,
    )))
}

// ---------------------------------------------------------------------------
// Inventory + retirement.
// ---------------------------------------------------------------------------

fn terminal_directories() -> Result<Vec<String>, String> {
    let mut identifiers = Vec::new();
    for entry in std::fs::read_dir(TERMINALS).map_err(|e| e.to_string())? {
        let entry = entry.map_err(|e| e.to_string())?;
        identifiers.push(
            entry
                .file_name()
                .into_string()
                .map_err(|_| "terminal name encoding".to_string())?,
        );
    }
    identifiers.sort();
    if identifiers.len() > 128 {
        return Err("native terminal inventory bound".to_string());
    }
    for identifier in &identifiers {
        terminal_path(identifier)?;
    }
    Ok(identifiers)
}

fn read_single_byte(file: &File) -> Result<Option<u8>, String> {
    let mut byte = [0u8; 1];
    let got = loop {
        let got =
            unsafe { libc::read(file.as_raw_fd(), byte.as_mut_ptr() as *mut libc::c_void, 1) };
        if got < 0 && io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
            continue;
        }
        if got < 0 {
            return Err(io::Error::last_os_error().to_string());
        }
        break got;
    };
    Ok(if got == 1 { Some(byte[0]) } else { None })
}

/// Retire a crash stub (missing/corrupt binding, no unit); the bare-raise
/// sites below return the *original* binding failure, message free-form.
fn stub_retire(path: &str, identifier: &str, directory: &File) -> Result<(), String> {
    let names: HashSet<String> = list_dir_names(directory)?.into_iter().collect();
    if names.iter().any(|n| n != "binding") {
        return Err("terminal binding".to_string());
    }
    if svc::service_state(identifier, None)? != "inactive" {
        return Err("terminal binding".to_string());
    }
    if !svc::cgroup_empty(identifier)? {
        return Err("terminal binding".to_string());
    }
    if names.contains("binding") {
        let fd = fs::root_file(directory, "binding", false)?;
        if read_single_byte(&fd)?.is_some() {
            return Err("unknown terminal binding".to_string());
        }
        drop(fd);
        fs::unlink_at(directory, "binding").map_err(|e| e.to_string())?;
    }
    std::fs::remove_dir(path).map_err(|e| e.to_string())?;
    Ok(())
}

fn remove_screen_contents(directory: &File, account: &Account) -> Result<(), String> {
    let screen = sys::open_child_dir(directory, "screen").map_err(|e| e.to_string())?;
    let (uid, mode) = fs::fstat_uid_mode(&screen).map_err(|e| e.to_string())?;
    // Failed startup may precede the root seal; accept only the original account.
    if (uid != 0 && uid != account.pw_uid) || mode & 0o022 != 0 {
        return Err("unexpected terminal screen".to_string());
    }
    for name in list_dir_names(&screen)? {
        let info = fstatat(&screen, &name).map_err(|e| e.to_string())?;
        let expected = (name == "socket" && s_issock(info.st_mode))
            || (name == "socket.lock" && s_isreg(info.st_mode));
        if !expected || info.st_uid != account.pw_uid || info.st_nlink != 1 {
            return Err("unexpected terminal socket".to_string());
        }
        fs::unlink_at(&screen, &name).map_err(|e| e.to_string())?;
    }
    Ok(())
}

pub(crate) fn remove_owned_files(
    path: &str,
    directory: &File,
    account: &Account,
) -> Result<(), String> {
    // Only exact run-owned files after unit shutdown, never recursive removal.
    let names: HashSet<String> = list_dir_names(directory)?.into_iter().collect();
    let allowed: HashSet<&str> = [
        "binding",
        "reservation",
        "name",
        "name.next",
        "writer",
        "tmux.conf",
        "ready",
        "screen",
    ]
    .into_iter()
    .collect();
    if names.iter().any(|n| !allowed.contains(n.as_str())) {
        return Err("unexpected terminal files".to_string());
    }
    if names.contains("screen") {
        remove_screen_contents(directory, account)?;
    }
    for name in names.iter().filter(|n| n.as_str() != "screen") {
        let info = fstatat(directory, name).map_err(|e| e.to_string())?;
        if !s_isreg(info.st_mode) || info.st_uid != 0 || info.st_nlink != 1 {
            return Err("unexpected terminal file".to_string());
        }
    }
    for name in names.iter().filter(|n| n.as_str() != "screen") {
        fs::unlink_at(directory, name).map_err(|e| e.to_string())?;
    }
    if names.contains("screen") {
        fs::rmdir_at(directory, "screen").map_err(|e| e.to_string())?;
    }
    std::fs::remove_dir(path).map_err(|e| e.to_string())?;
    Ok(())
}

/// Retire exact completed runtimes; returns the live-unit count.
pub fn collect_finished() -> Result<usize, String> {
    let mut active = 0usize;
    for identifier in terminal_directories()? {
        let path = format!("{TERMINALS}/{identifier}");
        let directory = checked_chain(&path)?;
        let record = match binding_record(&directory, None, 0) {
            Ok(record) => record,
            Err(_) => {
                // Missing/corrupt binding with no unit: retire the stub.
                stub_retire(&path, &identifier, &directory)?;
                continue;
            }
        };
        if list_dir_names(&directory)?
            .iter()
            .any(|n| n == "subscription")
        {
            active += 1;
            continue; // broker owns credential capture and retirement
        }
        let account = Account {
            pw_name: record.login.clone(),
            pw_uid: record.uid,
            pw_gid: record.gid,
            pw_dir: record.home.clone(),
            pw_shell: record.shell.clone(),
        };
        let state = svc::service_state(&identifier, Some(&account))?;
        if (state == "inactive" || state == "failed") && svc::cgroup_empty(&identifier)? {
            let permit = read_reservation(&directory)?;
            if !permit_live(permit.as_ref()) {
                remove_owned_files(&path, &directory, &account)?;
            }
        } else {
            active += 1;
        }
    }
    Ok(active)
}

// ---------------------------------------------------------------------------
// Reserve.
// ---------------------------------------------------------------------------

/// Pure `project-terminal` stat rule: regular, uid 0, `mode & 0o022 == 0`.
pub fn program_stat_ok(uid: u32, mode: u32, is_regular: bool) -> bool {
    is_regular && uid == 0 && mode & 0o022 == 0
}

/// Streaming SHA-256 hex over the whole file — DELIBERATE DELTA: the `.py`
/// caps at 64KB because it hashes a script, but this binary replaces it.
pub fn file_sha256_hex(file: &File) -> io::Result<String> {
    let mut hasher = sha::Sha256::new();
    let mut chunk = vec![0u8; 65536];
    loop {
        let got = unsafe {
            libc::read(
                file.as_raw_fd(),
                chunk.as_mut_ptr() as *mut libc::c_void,
                chunk.len(),
            )
        };
        if got < 0 {
            let err = io::Error::last_os_error();
            if err.raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            return Err(err);
        }
        if got == 0 {
            break;
        }
        hasher.update(&chunk[..got as usize]);
    }
    Ok(sha::hex(&hasher.finish()))
}

fn verify_program(source_hash: &str) -> Result<(), String> {
    // Missing/older project support refuses, never installs itself on Open.
    let parent = checked_chain("/usr/libexec/soda")?;
    let program = sys::open_at(
        &parent,
        "project-terminal",
        libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_NONBLOCK,
        0,
    )
    .map_err(|e| format!("open program: {e}"))?;
    let mut info: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(program.as_raw_fd(), &mut info) } != 0 {
        return Err(io::Error::last_os_error().to_string());
    }
    if !program_stat_ok(info.st_uid, info.st_mode, s_isreg(info.st_mode)) {
        return Err("unsafe terminal program".to_string());
    }
    let digest = file_sha256_hex(&program).map_err(|e| e.to_string())?;
    if digest != source_hash {
        return Err("terminal support version differs".to_string());
    }
    Ok(())
}

fn binding_object(record: &BindingRecord) -> JsonValue {
    JsonValue::Object(vec![
        (
            "account".to_string(),
            JsonValue::Array(vec![
                JsonValue::Str(record.login.clone()),
                JsonValue::Number(record.uid.to_string()),
                JsonValue::Number(record.gid.to_string()),
                JsonValue::Str(record.home.clone()),
                JsonValue::Str(record.shell.clone()),
            ]),
        ),
        (
            "identity".to_string(),
            JsonValue::Number(record.identity.to_string()),
        ),
        (
            "cols".to_string(),
            JsonValue::Number(record.cols.to_string()),
        ),
        (
            "rows".to_string(),
            JsonValue::Number(record.rows.to_string()),
        ),
        (
            "created_at".to_string(),
            JsonValue::Number(record.created_at.to_string()),
        ),
    ])
}

fn reservation_object(expires: i64, scope: &str) -> JsonValue {
    JsonValue::Object(vec![
        (
            "expires".to_string(),
            JsonValue::Number(expires.to_string()),
        ),
        ("scope".to_string(), JsonValue::Str(scope.to_string())),
    ])
}

fn write_name(directory: &File, name: &str) -> Result<(), String> {
    if !proto::valid_name(name) {
        return Err("terminal name".to_string());
    }
    // A prior interrupted rename may have left only this exact private
    // staging file. Readers/writers share the parent lock; never overwrite
    // unknown files.
    match record_exists(directory, "name.next")? {
        false => {}
        true => {
            // Unsafe staging content propagates (like the `.py` ValueError);
            // only a missing file is skipped.
            let staging = fs::root_file(directory, "name.next", false)?;
            drop(staging);
            match fs::unlink_at(directory, "name.next") {
                Ok(()) => {}
                Err(err) if err.kind() == io::ErrorKind::NotFound => {}
                Err(err) => return Err(err.to_string()),
            }
        }
    }
    fs::new_file(
        directory,
        "name.next",
        &pyemit::line(&JsonValue::Str(name.to_string())),
        0o600,
    )?;
    fs::replace_at(directory, "name.next", "name").map_err(|e| e.to_string())?;
    Ok(())
}

/// Reserve a terminal locator; the caller holds the parent lock.
#[allow(clippy::too_many_arguments)] // 8-arg shape mandated by the PR25 brief
pub fn reserve_terminal(
    identifier: &str,
    account: &Account,
    identity: i64,
    cols: i64,
    rows: i64,
    name: &str,
    source_hash: &str,
    scope: &str,
) -> Result<JsonValue, String> {
    verify_program(source_hash)?;
    for term in ["xterm-256color", "screen-256color"] {
        sys::run_checked(&svc::infocmp_argv(term), 2)
            .map_err(|e| format!("infocmp {term}: {e}"))?;
    }
    let path = terminal_path(identifier)?;
    // Refuse occupied locators before retiring any completed runtime files.
    // The caller holds the native parent lock across preparation/admission.
    let parent = checked_chain(TERMINALS)?;
    match fstatat(&parent, identifier) {
        Ok(_) => return Err("terminal identifier occupied".to_string()),
        Err(err) if err.kind() == io::ErrorKind::NotFound => {}
        Err(err) => return Err(err.to_string()),
    }
    if svc::service_state(identifier, None)? != "inactive" || !svc::cgroup_empty(identifier)? {
        return Err("terminal unit occupied".to_string());
    }
    collect_finished()?;
    if terminal_directories()?.len() >= 128 {
        return Err("native reservation capacity".to_string());
    }
    fs::mkdir_at(&parent, identifier, 0o711).map_err(|e| e.to_string())?;
    drop(parent);
    let directory = checked_chain(&path)?;
    fs::fchmod(&directory, 0o711).map_err(|e| e.to_string())?;
    let created = BindingRecord {
        login: account.pw_name.clone(),
        uid: account.pw_uid,
        gid: account.pw_gid,
        home: account.pw_dir.clone(),
        shell: account.pw_shell.clone(),
        identity,
        cols,
        rows,
        created_at: timex::now_secs(),
    };
    fs::new_file(
        &directory,
        "binding",
        &pyemit::line(&binding_object(&created)),
        0o600,
    )?;
    fs::new_file(
        &directory,
        "reservation",
        &pyemit::line(&reservation_object(timex::now_secs() + 120, scope)),
        0o600,
    )?;
    write_name(&directory, name)?;
    fs::new_file(&directory, "writer", b"", 0o600)?;
    fs::new_file(&directory, "tmux.conf", TMUX_CONFIG.as_bytes(), 0o644)?;
    fs::mkdir_at(&directory, "screen", 0o700).map_err(|e| e.to_string())?;
    fs::chown_path(
        &format!("{path}/screen"),
        account.pw_uid,
        account.pw_gid,
        false,
    )
    .map_err(|e| e.to_string())?;
    drop(directory);
    terminal_status(identifier, account, identity)?
        .ok_or_else(|| "terminal reservation".to_string())
}

// ---------------------------------------------------------------------------
// Create.
// ---------------------------------------------------------------------------

/// Exact `systemd-run` argv for the supervised tmux server. The privileged
/// hook runs the binary directly: `ExecStartPost=+…/project-terminal prepare
/// <id>` (no `python3`, no `-I`).
pub fn systemd_run_argv(
    identifier: &str,
    account: &Account,
    path: &str,
    lifetime: &[String],
) -> Vec<String> {
    let mut argv = vec![
        "/usr/bin/systemd-run".to_string(),
        "--quiet".to_string(),
        "--collect".to_string(),
        format!("--unit=soda-terminal-{identifier}"),
        format!("--description=Soda terminal {identifier}"),
        "--service-type=exec".to_string(),
        format!("--property=User={}", account.pw_name),
        format!("--property=Group={}", account.pw_gid),
        format!("--property=WorkingDirectory={}", account.pw_dir),
        "--slice=system.slice".to_string(),
        "--property=KillMode=control-group".to_string(),
        "--property=SendSIGKILL=yes".to_string(),
        "--property=Restart=no".to_string(),
        "--property=TimeoutStartSec=10s".to_string(),
        "--property=TimeoutStopSec=3s".to_string(),
        "--property=UMask=0077".to_string(),
        "--property=LimitCORE=0".to_string(),
    ];
    argv.extend(lifetime.iter().cloned());
    argv.extend(
        [
            "--property=StandardInput=null",
            "--property=StandardOutput=null",
            "--property=StandardError=null",
            &format!("--property=ExecStartPost=+{PROGRAM} prepare {identifier}"),
        ]
        .into_iter()
        .map(|s| s.to_string()),
    );
    for (key, value) in account::user_environment(account) {
        argv.push(format!("--setenv={key}={value}"));
    }
    argv.extend(
        [
            "/usr/bin/tmux",
            "-D",
            "-S",
            &format!("{path}/screen/socket"),
            "-f",
            &format!("{path}/tmux.conf"),
        ]
        .into_iter()
        .map(|s| s.to_string()),
    );
    argv
}

/// Consume the reservation and start the supervised server.
pub fn create_terminal(
    identifier: &str,
    account: &Account,
    identity: i64,
    cols: i64,
    rows: i64,
    name: &str,
    scope: &str,
) -> Result<JsonValue, String> {
    let path = terminal_path(identifier)?;
    let directory = checked_chain(&path)?; // no creation on a missing/ended locator
    let record = binding_record(&directory, Some(account), identity)?;
    let permit = read_reservation(&directory)?;
    let usable = match permit.as_ref() {
        Some(permit) => {
            permit.expires > timex::now_secs()
                && permit.scope == scope
                && (cols, rows) == (record.cols, record.rows)
        }
        None => false,
    };
    if !usable {
        return Err("creation reservation expired, consumed or changed".to_string());
    }
    if collect_finished()? >= 64 {
        return Err("native terminal capacity".to_string());
    }
    // Same native lock as End: a late Create cannot run after End removed its
    // one-use permission, even across web/helper restarts or a lost reply.
    fs::unlink_at(&directory, "reservation").map_err(|e| e.to_string())?;
    write_name(&directory, name)?;
    drop(directory);
    if svc::service_state(identifier, None)? != "inactive" {
        return Err("terminal unit occupied".to_string());
    }
    let lifetime = subscription_lifetime(&path)?;
    sys::run_checked(&systemd_run_argv(identifier, account, &path, &lifetime), 15)
        .map_err(|e| format!("systemd-run: {e}"))?;
    // No owner stream or lifetime timer. systemd's start job includes the
    // bounded privileged preparation hook; it cleans the cgroup if it fails.
    terminal_status(identifier, account, identity)?.ok_or_else(|| "terminal creation".to_string())
}

/// Pure `RuntimeMaxSec` rule: `0 < deadline - now <= 12h`.
pub fn lifetime_argv(deadline: i64, now: i64) -> Result<Vec<String>, String> {
    let seconds = deadline - now;
    if !(1..=12 * 3600).contains(&seconds) {
        return Err("subscription deadline".to_string());
    }
    Ok(vec![format!("--property=RuntimeMaxSec={seconds}")])
}

/// Extra unit lifetime from the optional `subscription` record.
pub fn subscription_lifetime(path: &str) -> Result<Vec<String>, String> {
    let directory = checked_chain(path)?;
    if !list_dir_names(&directory)?
        .iter()
        .any(|n| n == "subscription")
    {
        return Ok(Vec::new());
    }
    let record = fs::read_record(&directory, "subscription")?;
    let deadline = record
        .get("deadline")
        .and_then(pyemit::as_int)
        .ok_or_else(|| "subscription deadline".to_string())?;
    lifetime_argv(deadline, timex::now_secs())
}

/// Model-harness tmux command for subscription sessions.
pub fn subscription_command(path: &str) -> Vec<String> {
    vec![
        "-e".to_string(),
        format!("CODEX_HOME={path}/model/auth"),
        "-e".to_string(),
        format!("CODEX_SQLITE_HOME={path}/model/auth/state"),
        "-e".to_string(),
        format!("PATH={path}/model/harness/bin:{path}/model/harness/codex-path:/usr/bin:/bin"),
        format!(
            "exec {path}/model/harness/bin/codex --config 'cli_auth_credentials_store=\"file\"' --config 'sqlite_home=\"{path}/model/auth/state\"' --config 'log_dir=\"{path}/model/auth/logs\"' --ask-for-approval never --sandbox danger-full-access"
        ),
    ]
}

// ---------------------------------------------------------------------------
// Attach.
// ---------------------------------------------------------------------------

/// Attach a PTY; returns the process exit code.
pub fn attach_terminal(
    identifier: &str,
    account: &Account,
    identity: i64,
    cols: i64,
    rows: i64,
    seconds: i64,
) -> i32 {
    match attach_inner(identifier, account, identity, cols, rows, seconds) {
        Ok(code) => code,
        Err(_) => {
            closed_launch_failed();
            1
        }
    }
}

fn attach_inner(
    identifier: &str,
    account: &Account,
    identity: i64,
    cols: i64,
    rows: i64,
    seconds: i64,
) -> Result<i32, String> {
    let path = terminal_path(identifier)?;
    let directory = checked_chain(&path)?;
    let writer = fs::root_file(&directory, "writer", true)?;
    if !sys::flock_exclusive_nb(&writer).map_err(|e| e.to_string())? {
        return Err("terminal writer busy".to_string()); // never evict a writer
    }
    binding_record(&directory, Some(account), identity)?;
    let screen = checked_chain(&format!("{path}/screen"))?;
    drop(screen);
    let ready = parse_ready(&fs::read_record(&directory, "ready")?)?;
    let sock = format!("{path}/screen/socket");
    if svc::service_state(identifier, Some(account))? != "active" {
        return Err("terminal absent".to_string());
    }
    if svc::socket_identity(&sock, account, ready.0)? != (ready.1, ready.2) {
        return Err("terminal absent".to_string());
    }
    // The writer lock and directory stay held across the relay.
    let code = pty::run_terminal(account, cols, rows, seconds, &sock);
    drop(writer);
    drop(directory);
    Ok(code)
}

// ---------------------------------------------------------------------------
// List / mutate / control.
// ---------------------------------------------------------------------------

fn list_owned_terminals(account: &Account, identity: i64) -> Result<Vec<JsonValue>, String> {
    let mut values = Vec::new();
    for identifier in terminal_directories()? {
        let path = format!("{TERMINALS}/{identifier}");
        let directory = checked_chain(&path)?;
        let record = binding_record(&directory, None, 0)?;
        let pending = read_reservation(&directory)?;
        drop(directory);
        if pending.is_none()
            && record.identity == identity
            && binding_matches_account(&record, account)
        {
            if let Some(value) = terminal_status(&identifier, account, identity)? {
                if status_state(&value)? != "ended" {
                    values.push(value);
                }
            }
        }
    }
    Ok(values)
}

fn mutate_terminal(
    action: &str,
    identifier: &str,
    account: &Account,
    identity: i64,
    name: &str,
) -> Result<Vec<JsonValue>, String> {
    let path = terminal_path(identifier)?;
    // Absence probe first (always under the parent lock, like above).
    if let Err(err) = std::fs::symlink_metadata(&path) {
        if err.kind() == io::ErrorKind::NotFound {
            if action == "end" && terminal_status(identifier, account, identity)?.is_none() {
                return Ok(Vec::new());
            }
            return Err("terminal missing".to_string());
        }
        return Err(err.to_string());
    }
    let directory = checked_chain(&path)?;
    binding_record(&directory, Some(account), identity)?;
    if action == "end" {
        if list_dir_names(&directory)?
            .iter()
            .any(|n| n == "subscription")
        {
            return Err("subscription requires broker End".to_string());
        }
        svc::stop_service(identifier, account)?;
        remove_owned_files(&path, &directory, account)?;
        return Ok(Vec::new());
    }
    if action != "rename" {
        return Err("terminal action".to_string());
    }
    write_name(&directory, name)?;
    drop(directory);
    match terminal_status(identifier, account, identity)? {
        None => Ok(Vec::new()),
        Some(value) => Ok(vec![value]),
    }
}

/// Dispatch a control action under the TERMINALS parent lock (shared for
/// reads, exclusive for mutations).
#[allow(clippy::too_many_arguments)] // 9-arg shape mandated by the PR25 brief
pub fn control_terminal(
    action: &str,
    identifier: &str,
    account: &Account,
    identity: i64,
    cols: i64,
    rows: i64,
    name: &str,
    source_hash: &str,
    scope: &str,
) -> Result<Vec<JsonValue>, String> {
    let parent = checked_chain(TERMINALS)?;
    if action == "list" || action == "inspect" {
        sys::flock_shared(&parent).map_err(|e| e.to_string())?;
    } else {
        sys::flock_exclusive(&parent).map_err(|e| e.to_string())?;
    }
    let result = match action {
        "reserve" => reserve_terminal(
            identifier,
            account,
            identity,
            cols,
            rows,
            name,
            source_hash,
            scope,
        )
        .map(|v| vec![v]),
        "create" => {
            create_terminal(identifier, account, identity, cols, rows, name, scope).map(|v| vec![v])
        }
        "list" => list_owned_terminals(account, identity),
        "inspect" => {
            terminal_status(identifier, account, identity).map(|v| v.into_iter().collect())
        }
        _ => mutate_terminal(action, identifier, account, identity, name),
    };
    drop(parent);
    result
}

// ---------------------------------------------------------------------------
// Prepare (systemd ExecStartPost).
// ---------------------------------------------------------------------------

/// `tmux new-session` argv for the owned `soda` session. `-D` starts empty
/// and disables exit-empty; normal empty-server exit is restored AFTER
/// new-session in the same command queue (also supported by 3.2a).
pub fn tmux_new_session_args(cols: i64, rows: i64, home: &str, profile: &[String]) -> Vec<String> {
    let mut argv = vec![
        "new-session".to_string(),
        "-d".to_string(),
        "-s".to_string(),
        "soda".to_string(),
        "-x".to_string(),
        cols.to_string(),
        "-y".to_string(),
        rows.to_string(),
        "-c".to_string(),
        home.to_string(),
    ];
    argv.extend(profile.iter().cloned());
    argv.extend(
        [";", "set-option", "-s", "exit-empty", "on"]
            .into_iter()
            .map(|s| s.to_string()),
    );
    argv
}

fn subscription_session(directory: &File) -> Result<Vec<String>, String> {
    if !list_dir_names(directory)?
        .iter()
        .any(|n| n == "subscription")
    {
        return Ok(Vec::new());
    }
    let profile = fs::read_record(directory, "subscription")?;
    let id = profile
        .get("binding")
        .and_then(|v| v.get("id"))
        .and_then(|v| v.as_str())
        .ok_or_else(|| "terminal subscription".to_string())?;
    let unit = format!("soda-terminal-{id}.service");
    let output =
        sys::run_output(&svc::invocation_show_argv(&unit), 2).map_err(|e| e.to_string())?;
    if !output.status.success() {
        return Err("terminal incarnation".to_string());
    }
    let invocation = std::str::from_utf8(&output.stdout)
        .map_err(|_| "terminal incarnation".to_string())?
        .trim()
        .to_string();
    if !proto::valid_scope(&invocation) {
        return Err("missing terminal incarnation".to_string());
    }
    let record = JsonValue::Object(vec![(
        "invocation_id".to_string(),
        JsonValue::Str(invocation),
    )]);
    fs::new_file(
        directory,
        "subscription-unit",
        &pyemit::line(&record),
        0o600,
    )?;
    Ok(vec!["exec /usr/bin/sleep infinity".to_string()])
}

fn ready_object(pid: i64, dev: u64, ino: u64) -> JsonValue {
    JsonValue::Object(vec![
        ("pid".to_string(), JsonValue::Number(pid.to_string())),
        (
            "socket".to_string(),
            JsonValue::Array(vec![
                JsonValue::Number(dev.to_string()),
                JsonValue::Number(ino.to_string()),
            ]),
        ),
    ])
}

/// Privileged one-shot hook: verify the cgroup/socket, open the session,
/// publish `ready`. Silent success; `closed/launch_failed` + 1 on failure
/// (the `.py` `main` wrapper reports prepare failures exactly so).
pub fn prepare(identifier: &str) -> i32 {
    match prepare_inner(identifier) {
        Ok(()) => 0,
        Err(_) => {
            closed_launch_failed();
            1
        }
    }
}

fn prepare_inner(identifier: &str) -> Result<(), String> {
    let path = terminal_path(identifier)?;
    let parent = svc::cgroup_parent()?;
    let group =
        sys::open_child_dir(&parent, &svc::unit_name(identifier)).map_err(|e| e.to_string())?;
    drop(parent);
    let (uid, mode) = fs::fstat_uid_mode(&group).map_err(|e| e.to_string())?;
    if uid != 0 || mode & 0o022 != 0 {
        return Err("unsafe terminal cgroup".to_string());
    }
    let procs = sys::open_at(&group, "cgroup.procs", libc::O_RDONLY | libc::O_NOFOLLOW, 0)
        .map_err(|e| e.to_string())?;
    drop(group);
    let mut content = vec![0u8; 16384];
    let got = loop {
        let got = unsafe {
            libc::read(
                procs.as_raw_fd(),
                content.as_mut_ptr() as *mut libc::c_void,
                content.len(),
            )
        };
        if got < 0 && io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
            continue;
        }
        if got < 0 {
            return Err(io::Error::last_os_error().to_string());
        }
        break got as usize;
    };
    content.truncate(got);
    if !content.is_ascii() {
        return Err("preparation outside owned cgroup".to_string());
    }
    let text = std::str::from_utf8(&content)
        .map_err(|_| "preparation outside owned cgroup".to_string())?;
    let here = std::process::id().to_string();
    if !text
        .split('\n')
        .any(|line| line.strip_suffix('\r').unwrap_or(line) == here)
    {
        return Err("preparation outside owned cgroup".to_string());
    }
    drop(procs);
    let directory = checked_chain(&path)?;
    let record = binding_record(&directory, None, 0)?;
    let account = account::account_for(&record.login, record.identity)?;
    binding_record(&directory, Some(&account), record.identity)?;
    let sock = format!("{path}/screen/socket");
    // MAINPID is supplied by systemd, never by the browser.
    let pid: i64 = std::env::var("MAINPID")
        .map_err(|_| "missing main process".to_string())?
        .parse()
        .map_err(|_| "missing main process".to_string())?;
    if pid <= 0 || pid > i64::from(i32::MAX) {
        return Err("missing main process".to_string());
    }
    let pid = pid as i32;
    let until = sys::monotonic() + 5.0;
    loop {
        if sys::monotonic() >= until {
            return Err("tmux startup failed".to_string());
        }
        if std::path::Path::new(&format!("{sock}.lock")).exists() {
            std::thread::sleep(std::time::Duration::from_millis(20));
            continue;
        }
        match svc::socket_identity_kinded(&sock, &account, pid) {
            Ok(_) => break,
            Err(svc::SocketCheck::Retryable(_)) => {
                std::thread::sleep(std::time::Duration::from_millis(20));
            }
            Err(svc::SocketCheck::Fatal(message)) => return Err(message),
        }
    }
    // Seal the socket's parent before publishing readiness: the project user
    // may connect but cannot replace this socket with another context.
    fs::chown_path(&format!("{path}/screen"), 0, 0, true).map_err(|e| e.to_string())?;
    fs::chmod_path(&format!("{path}/screen"), 0o711, true).map_err(|e| e.to_string())?;
    let (dev, ino) = svc::socket_identity(&sock, &account, pid)?;
    let profile = subscription_session(&directory)?;
    svc::tmux_control(
        &account,
        &sock,
        &tmux_new_session_args(record.cols, record.rows, &account.pw_dir, &profile),
    )?;
    fs::new_file(
        &directory,
        "ready",
        &pyemit::line(&ready_object(i64::from(pid), dev, ino)),
        0o600,
    )?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sample() -> Account {
        Account {
            pw_name: "op".to_string(),
            pw_uid: 1001,
            pw_gid: 1002,
            pw_dir: "/home/op".to_string(),
            pw_shell: "/bin/bash".to_string(),
        }
    }

    fn binding_doc(account: &str, identity: &str, cols: &str, rows: &str, created: &str) -> String {
        format!(
            "{{\"account\":{account},\"identity\":{identity},\"cols\":{cols},\"rows\":{rows},\"created_at\":{created}}}"
        )
    }

    fn good_account() -> String {
        r#"["op",1001,1002,"/home/op","/bin/bash"]"#.to_string()
    }

    fn parse(text: &str) -> JsonValue {
        JsonValue::parse(text).unwrap()
    }

    #[test]
    fn path_matrix() {
        let id = "a".repeat(32);
        assert_eq!(
            terminal_path(&id).unwrap(),
            format!("/run/soda-terminals/{id}")
        );
        assert!(terminal_path("").is_err());
        assert!(terminal_path(&"A".repeat(32)).is_err());
        assert!(terminal_path("short").is_err());
        assert!(terminal_path("../escape").is_err());
    }

    #[test]
    fn binding_matrix() {
        let good = binding_doc(&good_account(), "7", "80", "24", "1700000000");
        let record = validate_binding(&parse(&good)).unwrap();
        assert_eq!(record.login, "op");
        assert_eq!((record.uid, record.gid), (1001, 1002));
        assert_eq!(
            (record.identity, record.cols, record.rows, record.created_at),
            (7, 80, 24, 1700000000)
        );
        // Shape violations.
        assert!(validate_binding(&parse("[1,2]")).is_err());
        assert!(
            validate_binding(&parse(r#"{"account":[],"identity":1,"cols":80,"rows":24}"#)).is_err()
        );
        assert!(validate_binding(&parse(
            &binding_doc(&good_account(), "7", "80", "24", "1700000000").replace("cols", "colz")
        ))
        .is_err());
        // Duplicate keys tolerated (json.loads last-wins), unlike control frames.
        let dup = binding_doc(&good_account(), "7", "80", "24", "1");
        let dup = dup.replace("\"created_at\":1", "\"created_at\":1,\"created_at\":9");
        assert_eq!(validate_binding(&parse(&dup)).unwrap().created_at, 9);
        // Account vector violations.
        for bad in [
            r#"["Root",1001,1002,"/home/op","/bin/bash"]"#, // login case
            r#"["root",1001,1002,"/home/op","/bin/bash"]"#, // root
            r#"["op",0,1002,"/home/op","/bin/bash"]"#,      // uid 0
            r#"["op",-1,1002,"/home/op","/bin/bash"]"#,     // uid negative
            r#"["op",1001,-1,"/home/op","/bin/bash"]"#,     // gid negative
            r#"["op",1001,1002,"home/op","/bin/bash"]"#,    // relative dir
            r#"["op",1001,1002,"/home/op","bin/bash"]"#,    // relative shell
            r#"["op",1001,1002,"/home/op"]"#,               // short
            r#"["op",1001.0,1002,"/home/op","/bin/bash"]"#, // float uid
            r#"["op",true,1002,"/home/op","/bin/bash"]"#,   // bool uid
            r#""op""#,                                      // not a list
        ] {
            assert!(
                validate_binding(&parse(&binding_doc(bad, "7", "80", "24", "1700000000"))).is_err(),
                "{bad}"
            );
        }
        // Scalar violations.
        for (identity, cols, rows, created) in [
            ("0", "80", "24", "1700000000"),       // identity 0
            ("-3", "80", "24", "1700000000"),      // identity negative
            ("7.0", "80", "24", "1700000000"),     // identity float
            ("7", "1", "24", "1700000000"),        // cols small
            ("7", "501", "24", "1700000000"),      // cols big
            ("7", "80", "301", "1700000000"),      // rows big
            ("7", "\"80\"", "24", "1700000000"),   // cols string
            ("7", "80", "24", "0"),                // created 0
            ("7", "80", "24", "-1"),               // created negative
            ("7", "80", "24", "9007199254740992"), // created > 2^53-1
            ("7", "80", "24", "true"),             // created bool
        ] {
            assert!(
                validate_binding(&parse(&binding_doc(
                    &good_account(),
                    identity,
                    cols,
                    rows,
                    created
                )))
                .is_err(),
                "{identity}/{cols}/{rows}/{created}"
            );
        }
        // Upper bound 2^53-1 accepted.
        assert!(validate_binding(&parse(&binding_doc(
            &good_account(),
            "7",
            "80",
            "24",
            "9007199254740991"
        )))
        .is_ok());
    }

    #[test]
    fn account_match_matrix() {
        let record = validate_binding(&parse(&binding_doc(
            &good_account(),
            "7",
            "80",
            "24",
            "1700000000",
        )))
        .unwrap();
        let account = sample();
        assert!(binding_matches_account(&record, &account));
        let mut other = account.clone();
        other.pw_uid = 1003;
        assert!(!binding_matches_account(&record, &other));
        let mut other = account.clone();
        other.pw_shell = "/bin/sh".to_string();
        assert!(!binding_matches_account(&record, &other));
    }

    #[test]
    fn reservation_matrix() {
        let scope = "c".repeat(64);
        let good = format!("{{\"expires\":9999999999,\"scope\":\"{scope}\"}}");
        let permit = validate_reservation(&parse(&good)).unwrap();
        assert_eq!((permit.expires, permit.scope), (9999999999, scope));
        for bad in [
            r#"{"expires":9999999999}"#.to_string(),
            r#"{"expires":0,"scope":""}"#.to_string(),
            format!("{{\"expires\":-1,\"scope\":\"{}\"}}", "c".repeat(64)),
            format!("{{\"expires\":99.5,\"scope\":\"{}\"}}", "c".repeat(64)),
            r#"{"expires":99,"scope":"short"}"#.to_string(),
            format!("{{\"expires\":99,\"scope\":\"{}\"}}", "C".repeat(64)),
            format!(
                "{{\"expires\":99,\"scope\":\"{}\",\"x\":1}}",
                "c".repeat(64)
            ),
            r#"[]"#.to_string(),
        ] {
            assert!(validate_reservation(&parse(&bad)).is_err(), "{bad}");
        }
    }

    #[test]
    fn missing_reservation_is_none() {
        // Absence probe works without root-owned files.
        let dir = std::env::temp_dir().join(format!("soda-pt-term-{}-res", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let fd = std::fs::File::open(&dir).unwrap();
        assert_eq!(read_reservation(&fd).unwrap(), None);
        assert!(!record_exists(&fd, "reservation").unwrap());
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn classify_matrix() {
        use UnitClass::*;
        // Stopped units: live permit → opening, else ended.
        assert_eq!(
            classify_bare_state("inactive", false, false).unwrap(),
            Ended
        );
        assert_eq!(classify_bare_state("failed", true, false).unwrap(), Ended);
        assert_eq!(
            classify_bare_state("inactive", true, true).unwrap(),
            Opening
        );
        // Live units with a permit are corrupt.
        for state in [
            "active",
            "activating",
            "deactivating",
            "reloading",
            "unknown",
        ] {
            assert!(classify_bare_state(state, true, true).is_err(), "{state}");
            assert!(classify_bare_state(state, true, false).is_err(), "{state}");
        }
        // Live states without a permit.
        assert_eq!(
            classify_bare_state("active", false, false).unwrap(),
            ReadyCheck
        );
        assert_eq!(
            classify_bare_state("activating", false, false).unwrap(),
            Opening
        );
        assert_eq!(
            classify_bare_state("deactivating", false, false).unwrap(),
            Ending
        );
        assert!(classify_bare_state("reloading", false, false).is_err());
        assert!(classify_bare_state("", false, false).is_err());
    }

    #[test]
    fn systemd_argv_exact() {
        let id = "a".repeat(32);
        let path = format!("/run/soda-terminals/{id}");
        let account = sample();
        let argv = systemd_run_argv(&id, &account, &path, &[]);
        let head = [
            "/usr/bin/systemd-run",
            "--quiet",
            "--collect",
            &format!("--unit=soda-terminal-{id}"),
            &format!("--description=Soda terminal {id}"),
            "--service-type=exec",
            "--property=User=op",
            "--property=Group=1002",
            "--property=WorkingDirectory=/home/op",
            "--slice=system.slice",
            "--property=KillMode=control-group",
            "--property=SendSIGKILL=yes",
            "--property=Restart=no",
            "--property=TimeoutStartSec=10s",
            "--property=TimeoutStopSec=3s",
            "--property=UMask=0077",
            "--property=LimitCORE=0",
            "--property=StandardInput=null",
            "--property=StandardOutput=null",
            "--property=StandardError=null",
            &format!("--property=ExecStartPost=+{PROGRAM} prepare {id}"),
            "--setenv=HOME=/home/op",
            "--setenv=USER=op",
            "--setenv=LOGNAME=op",
            "--setenv=SHELL=/bin/bash",
            "--setenv=PATH=/usr/local/bin:/usr/bin:/bin",
            "--setenv=TERM=xterm-256color",
            "--setenv=LANG=C.UTF-8",
            "/usr/bin/tmux",
            "-D",
            "-S",
            &format!("{path}/screen/socket"),
            "-f",
            &format!("{path}/tmux.conf"),
        ];
        assert_eq!(
            argv,
            head.into_iter().map(|s| s.to_string()).collect::<Vec<_>>()
        );
        // No python3 anywhere in the hook.
        assert!(!argv.iter().any(|a| a.contains("python")));
        // Lifetime slots in after LimitCORE.
        let argv = systemd_run_argv(
            &id,
            &account,
            &path,
            &["--property=RuntimeMaxSec=60".to_string()],
        );
        let at = argv
            .iter()
            .position(|a| a == "--property=LimitCORE=0")
            .unwrap();
        assert_eq!(argv[at + 1], "--property=RuntimeMaxSec=60");
        assert_eq!(argv[at + 2], "--property=StandardInput=null");
    }

    #[test]
    fn tmux_session_argv_exact() {
        assert_eq!(
            tmux_new_session_args(80, 24, "/home/op", &[]),
            [
                "new-session",
                "-d",
                "-s",
                "soda",
                "-x",
                "80",
                "-y",
                "24",
                "-c",
                "/home/op",
                ";",
                "set-option",
                "-s",
                "exit-empty",
                "on"
            ]
            .into_iter()
            .map(|s| s.to_string())
            .collect::<Vec<_>>()
        );
        let profile = vec!["exec /usr/bin/sleep infinity".to_string()];
        let argv = tmux_new_session_args(100, 30, "/home/op", &profile);
        assert_eq!(argv[10], "exec /usr/bin/sleep infinity");
        assert_eq!(&argv[11..], &[";", "set-option", "-s", "exit-empty", "on"]);
    }

    #[test]
    fn subscription_command_exact() {
        let path = "/run/soda-terminals/abc";
        assert_eq!(
            subscription_command(path),
            vec![
                "-e".to_string(),
                format!("CODEX_HOME={path}/model/auth"),
                "-e".to_string(),
                format!("CODEX_SQLITE_HOME={path}/model/auth/state"),
                "-e".to_string(),
                format!("PATH={path}/model/harness/bin:{path}/model/harness/codex-path:/usr/bin:/bin"),
                format!(
                    "exec {path}/model/harness/bin/codex --config 'cli_auth_credentials_store=\"file\"' --config 'sqlite_home=\"{path}/model/auth/state\"' --config 'log_dir=\"{path}/model/auth/logs\"' --ask-for-approval never --sandbox danger-full-access"
                ),
            ]
        );
    }

    #[test]
    fn lifetime_rule() {
        assert_eq!(
            lifetime_argv(1000, 900).unwrap(),
            vec!["--property=RuntimeMaxSec=100".to_string()]
        );
        assert_eq!(
            lifetime_argv(900 + 43200, 900).unwrap(),
            vec!["--property=RuntimeMaxSec=43200".to_string()]
        );
        assert!(lifetime_argv(900, 900).is_err());
        assert!(lifetime_argv(899, 900).is_err());
        assert!(lifetime_argv(900 + 43201, 900).is_err());
    }

    #[test]
    fn emission_bytes_exact() {
        assert_eq!(
            pyemit::dumps(&status_object("id", "nm", 42, true, false, "ready")),
            r#"{"id":"id","name":"nm","created_at":42,"ready":true,"attached":false,"state":"ready"}"#
        );
        assert_eq!(
            pyemit::dumps(&ready_object(7, 8, 9)),
            r#"{"pid":7,"socket":[8,9]}"#
        );
        assert_eq!(
            pyemit::dumps(&reservation_object(100, &"c".repeat(64))),
            format!("{{\"expires\":100,\"scope\":\"{}\"}}", "c".repeat(64))
        );
        let record = validate_binding(&parse(&binding_doc(
            &good_account(),
            "7",
            "80",
            "24",
            "1700000000",
        )))
        .unwrap();
        assert_eq!(
            pyemit::dumps(&binding_object(&record)),
            r#"{"account":["op",1001,1002,"/home/op","/bin/bash"],"identity":7,"cols":80,"rows":24,"created_at":1700000000}"#
        );
        // Name docs are bare JSON strings.
        assert_eq!(pyemit::line(&JsonValue::Str("nm".to_string())), b"\"nm\"\n");
    }

    #[test]
    fn program_hash_streams_past_64k() {
        // Proves the deliberate no-cap delta: 100KB hashes whole.
        let dir = std::env::temp_dir().join(format!("soda-pt-term-{}-hash", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let big: Vec<u8> = (0..100_000).map(|i| (i * 31 + 7) as u8).collect();
        std::fs::write(dir.join("big"), &big).unwrap();
        let file = std::fs::File::open(dir.join("big")).unwrap();
        assert_eq!(file_sha256_hex(&file).unwrap(), sha::hex_digest(&big));
        assert_eq!(
            sha::hex(&sha::digest(b"abc")),
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
        assert_ne!(
            file_sha256_hex(&file).unwrap(),
            sha::hex_digest(&big[..65536])
        );
        assert!(program_stat_ok(0, 0o100644, true));
        assert!(program_stat_ok(0, 0o100755, true));
        assert!(!program_stat_ok(0, 0o100664, true)); // group write
        assert!(!program_stat_ok(1000, 0o100644, true)); // owner
        assert!(!program_stat_ok(0, 0o100644, false)); // not regular
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn ready_matrix() {
        assert_eq!(
            parse_ready(&parse(r#"{"pid":7,"socket":[8,9]}"#)).unwrap(),
            (7, 8, 9)
        );
        // Extra keys tolerated (direct indexing in the `.py`).
        assert_eq!(
            parse_ready(&parse(r#"{"pid":7,"socket":[8,9],"x":1}"#)).unwrap(),
            (7, 8, 9)
        );
        for bad in [
            r#"{"socket":[8,9]}"#,
            r#"{"pid":7}"#,
            r#"{"pid":"7","socket":[8,9]}"#,
            r#"{"pid":7,"socket":[8]}"#,
            r#"{"pid":7,"socket":[8,-1]}"#,
            r#"{"pid":7.0,"socket":[8,9]}"#,
            r#"[]"#,
        ] {
            assert!(parse_ready(&parse(bad)).is_err(), "{bad}");
        }
    }

    #[test]
    fn entry_point_smoke() {
        let account = sample();
        let id = "e".repeat(32);
        // Deterministic failures (bad shape, absent paths, no privilege).
        // `prepare`/`attach` failures write `closed/launch_failed` lines to
        // the harness-captured stdout.
        assert_eq!(prepare("not-an-id"), 1);
        assert_eq!(prepare(&id), 1);
        assert_eq!(attach_terminal("not-an-id", &account, 1, 80, 24, 60), 1);
        assert_eq!(attach_terminal(&id, &account, 1, 80, 24, 60), 1);
        assert!(control_terminal("bogus", &id, &account, 1, 80, 24, "", "", "").is_err());
        assert!(reserve_terminal(&id, &account, 1, 80, 24, "nm", "x", &"c".repeat(64)).is_err());
        assert!(create_terminal("not-an-id", &account, 1, 80, 24, "nm", &"c".repeat(64)).is_err());
        assert!(subscription_lifetime("/definitely/not/here-9f3c").is_err());
        // Environment-dependent outcomes (systemd host vs container):
        // exercised, not asserted.
        let dir = std::env::temp_dir().join(format!("soda-pt-term-{}-smoke", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let fd = std::fs::File::open(&dir).unwrap();
        let _result = binding_record(&fd, None, 0);
        let _result = terminal_status(&id, &account, 1);
        let _result = collect_finished();
        let _result = create_terminal(&id, &account, 1, 80, 24, "nm", &"c".repeat(64));
        let _result = control_terminal("list", "", &account, 1, 80, 24, "", "", "");
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn tmux_config_exact() {
        assert_eq!(
            TMUX_CONFIG,
            "set -g status off\nset -g history-limit 10000\nset -s buffer-limit 10\nset -s set-clipboard off\nset -s escape-time 10\nset -g default-terminal screen-256color\nset -g update-environment \"\"\nset -s exit-unattached off\n"
        );
    }
}
