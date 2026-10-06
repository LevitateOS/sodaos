//! Identity subscription broker (port of `identity_terminal.py`).
//!
//! One JSON request on stdin (`prepare`/`lookup`/`stage`/`start`/
//! `validate`/`finish`/`stop`), one `{"lease","credential"}` document on
//! stdout with `json.dumps` default separators and no trailing newline.
//! Any failure writes `managed Codex operation failed` to stderr and exits
//! 1, with no request bytes or error text in diagnostics. A 45s `SIGALRM`
//! bounds the whole body; like the `.py` (which installs no `SIGALRM`
//! handler) the process dies by signal on timeout.
//!
//! Deltas from the `.py`, all confined to unobservable or absurd inputs:
//! - request bytes must be UTF-8 (`.py` `json.loads(bytes)` also sniffs
//!   UTF-16/32); other encodings fail cleanly either way.
//! - non-object JSON, wrong-typed fields, and duplicate keys fail with the
//!   fixed stderr line instead of a traceback (`.py` raises `TypeError`
//!   outside its `except` list there). Same exit code, quieter stderr.
//! - `EPIPE` on the stdout result exits 0 silently instead of the stderr
//!   line plus exit 1 (the peer is gone; nothing can observe this).
//! - lease `actor_id` accepts JSON ints (i64), floats truncated toward
//!   zero, JSON bools (`True`/`False` → 1/0), and plain numeric strings
//!   (the `.py` `int()` conversions); out-of-range ints and exotic
//!   spellings (underscores, unicode digits) fail closed instead.
//! - `cols`/`rows` must be JSON ints: a string `"80"` fails before the
//!   reservation here, where the `.py` would fail the binding check after
//!   creating the locator. Same exit, same output, no litter.
//! - lease/profile deep equality is order-insensitive with numeric
//!   int/float equivalence (matching `.py` `dict ==`); exotica beyond
//!   that (bool-vs-int excepted) compare by JSON shape.

use std::fs::File;
use std::io::{Read, Write};
use std::os::unix::io::AsRawFd;

use soda_json::JsonValue;

use crate::account::{self, Account};
use crate::fs;
use crate::proto;
use crate::pyemit;
use crate::svc;
use crate::sys;
use crate::term;

/// Stdin cap: `sys.stdin.buffer.read(512 * 1024 + 1)`; longer input is a
/// `request size` failure, never truncation.
pub const REQUEST_LIMIT: usize = 512 * 1024;
/// Credential size bound (`0 < len <= 256KiB`), both directions.
pub const CREDENTIAL_LIMIT: usize = 256 * 1024;
/// Subscription horizon bound (`0 < deadline - now <= 12h`).
pub const HORIZON_SECS: i64 = 12 * 3600;
/// Whole-body broker alarm, in seconds.
pub const BROKER_ALARM_SECS: u32 = 45;

use crate::subscription_prepare::subscription_prepare;
use crate::subscription_wire::{decode_request, empty_result, json_equal, json_int, result_object};

// ---------------------------------------------------------------------------
// Filesystem operations.
// ---------------------------------------------------------------------------

fn subscription_path(identifier: &str) -> Result<String, String> {
    Ok(format!("{}/model", term::terminal_path(identifier)?))
}

/// Missing-path probe under the caller's parent lock (the `String` errors
/// cannot separate `NotFound`; same pattern as `term::terminal_status`).
fn path_missing(path: &str) -> Result<bool, String> {
    match std::fs::symlink_metadata(path) {
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(true),
        Err(err) => Err(err.to_string()),
        Ok(_) => Ok(false),
    }
}

pub(crate) fn lease_execution_id(lease: &JsonValue) -> Result<&str, String> {
    lease
        .get("execution_id")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "subscription lease".to_string())
}

pub(crate) fn lease_actor_id(lease: &JsonValue) -> Result<i64, String> {
    lease
        .get("actor_id")
        .and_then(json_int)
        .ok_or_else(|| "subscription lease".to_string())
}

fn profile_deadline(profile: &JsonValue) -> Result<i64, String> {
    profile
        .get("deadline")
        .and_then(pyemit::as_int)
        .ok_or_else(|| "subscription deadline".to_string())
}

fn live_deadline(profile: &JsonValue) -> Result<(), String> {
    if profile_deadline(profile)? <= sys::now_secs() {
        return Err("session expired".to_string());
    }
    Ok(())
}

/// `subscription_profile` outcome: `Missing` replays the `.py`
/// `FileNotFoundError` path (absent locator, subscription, or binding).
pub enum ProfileHit {
    Found {
        profile: JsonValue,
        account: Account,
    },
    Missing,
}

pub fn subscription_profile(lease: &JsonValue) -> Result<ProfileHit, String> {
    let identifier = lease_execution_id(lease)?.to_string();
    let path = term::terminal_path(&identifier)?;
    if path_missing(&path)? {
        return Ok(ProfileHit::Missing);
    }
    let directory = term::checked_chain(&path)?;
    if !term::record_exists(&directory, "subscription")? {
        return Ok(ProfileHit::Missing);
    }
    let profile = fs::read_record(&directory, "subscription")?;
    let stored_lease = profile
        .get("lease")
        .ok_or_else(|| "subscription lease".to_string())?;
    let stored_binding = profile
        .get("binding")
        .ok_or_else(|| "subscription lease".to_string())?;
    let lease_binding = lease
        .get("binding")
        .ok_or_else(|| "subscription lease".to_string())?;
    if !json_equal(stored_lease, lease) || !json_equal(stored_binding, lease_binding) {
        return Err("lease changed".to_string());
    }
    let actor_id = lease_actor_id(lease)?;
    let login = lease_binding
        .get("login")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "subscription lease".to_string())?;
    let account = account::account_for(login, actor_id)?;
    if !term::record_exists(&directory, "binding")? {
        return Ok(ProfileHit::Missing);
    }
    term::binding_record(&directory, Some(&account), actor_id)?;
    Ok(ProfileHit::Found { profile, account })
}

fn subscription_invocation(identifier: &str) -> Result<String, String> {
    let output = sys::run_output(&svc::invocation_show_argv(&svc::unit_name(identifier)), 2)
        .map_err(|e| format!("unit invocation: {e}"))?;
    if !output.status.success() {
        return Err("unit invocation".to_string());
    }
    let text = std::str::from_utf8(&output.stdout).map_err(|_| "unit invocation".to_string())?;
    Ok(text.trim().to_string())
}

/// `subscription_check_unit`: incarnation pin plus liveness classification.
/// Returns `false` for a confirmed-absent unit when `running` is false.
pub fn subscription_check_unit(
    lease: &JsonValue,
    account: &Account,
    running: bool,
) -> Result<bool, String> {
    let identifier = lease_execution_id(lease)?.to_string();
    let state = svc::service_state(&identifier, Some(account))?;
    let path = term::terminal_path(&identifier)?;
    let directory = term::checked_chain(&path)?;
    if !term::record_exists(&directory, "subscription-unit")? {
        if state != "inactive" || !svc::cgroup_empty(&identifier)? {
            return Err("unexpected unit".to_string());
        }
        if running {
            return Err("session not started".to_string());
        }
        return Ok(false);
    }
    let record = fs::read_record(&directory, "subscription-unit")?;
    drop(directory);
    let expected = record
        .get("invocation_id")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "unit incarnation changed".to_string())?;
    let observed = subscription_invocation(&identifier)?;
    // Invocation IDs are lowercase hex-32, like terminal identifiers.
    if !proto::valid_identifier(expected) || (!observed.is_empty() && observed != expected) {
        return Err("unit incarnation changed".to_string());
    }
    if (state == "inactive" || state == "failed") && svc::cgroup_empty(&identifier)? {
        if running {
            return Err("session ended".to_string());
        }
        return Ok(false);
    }
    if observed != expected {
        return Err("unit incarnation absent".to_string());
    }
    if running && state != "active" {
        return Err("session unavailable".to_string());
    }
    Ok(true)
}

/// `fchownat(dir, name, uid, gid, AT_SYMLINK_NOFOLLOW)`.
fn fchownat_no_follow(dir: &File, name: &str, uid: u32, gid: u32) -> std::io::Result<()> {
    let target = std::ffi::CString::new(name)
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "nul byte in name"))?;
    if unsafe {
        libc::fchownat(
            dir.as_raw_fd(),
            target.as_ptr(),
            uid as libc::uid_t,
            gid as libc::gid_t,
            libc::AT_SYMLINK_NOFOLLOW,
        )
    } != 0
    {
        return Err(std::io::Error::last_os_error());
    }
    Ok(())
}

fn is_regular(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFREG
}

/// `subscription_auth_directory`: the `model/auth` dir must be owned by the
/// account with no group/other permission bits.
pub fn subscription_auth_directory(lease: &JsonValue, account: &Account) -> Result<File, String> {
    let identifier = lease_execution_id(lease)?.to_string();
    let model = term::checked_chain(&subscription_path(&identifier)?)?;
    let directory = sys::open_child_dir(&model, "auth").map_err(|e| e.to_string())?;
    drop(model);
    let info = fs::fstat_all(&directory).map_err(|e| e.to_string())?;
    if info.st_uid != account.pw_uid || info.st_gid != account.pw_gid || info.st_mode & 0o077 != 0 {
        return Err("unsafe auth directory".to_string());
    }
    Ok(directory)
}

/// `subscription_seed`: validated credential bytes become the account-owned
/// `auth.json` (exclusive create: a second stage fails, like the `.py`).
pub fn subscription_seed(
    lease: &JsonValue,
    account: &Account,
    encoded: &str,
) -> Result<(), String> {
    let state = crate::b64::decode(encoded).ok_or_else(|| "credential encoding".to_string())?;
    if state.is_empty() || state.len() > CREDENTIAL_LIMIT {
        return Err("credential size".to_string());
    }
    let text = std::str::from_utf8(&state).map_err(|_| "credential json".to_string())?;
    JsonValue::parse(text).map_err(|_| "credential json".to_string())?;
    let directory = subscription_auth_directory(lease, account)?;
    fs::new_file(&directory, "auth.json", &state, 0o600)?;
    fchownat_no_follow(&directory, "auth.json", account.pw_uid, account.pw_gid)
        .map_err(|e| e.to_string())?;
    Ok(())
}

/// `subscription_stage`: deadline, liveness, then seed.
pub fn subscription_stage(
    request: &JsonValue,
    lease: &JsonValue,
    profile: &JsonValue,
    account: &Account,
) -> Result<JsonValue, String> {
    live_deadline(profile)?;
    subscription_check_unit(lease, account, true)?;
    let encoded = request
        .get("delivery")
        .and_then(|d| d.get("credential"))
        .and_then(|v| v.as_str())
        .ok_or_else(|| "subscription request".to_string())?;
    subscription_seed(lease, account, encoded)?;
    Ok(result_object(lease, ""))
}

/// `subscription_harness_digest`: the guest `codex` binary must be a
/// root-owned regular file with one link and no group/other write bits.
pub fn subscription_harness_digest(harness: &str) -> Result<String, String> {
    let directory = term::checked_chain(&format!("{harness}/bin"))?;
    let codex = sys::open_at(&directory, "codex", libc::O_RDONLY | libc::O_NONBLOCK, 0)
        .map_err(|e| e.to_string())?;
    drop(directory);
    let info = fs::fstat_all(&codex).map_err(|e| e.to_string())?;
    if !is_regular(info.st_mode)
        || info.st_uid != 0
        || info.st_mode & 0o022 != 0
        || info.st_nlink != 1
    {
        return Err("unsafe harness executable".to_string());
    }
    term::file_sha256_hex(&codex).map_err(|e| e.to_string())
}

/// Exact `respawn-pane` argv for the model session (fixed head plus
/// `term::subscription_command`).
pub fn respawn_argv(path: &str, home: &str) -> Vec<String> {
    let mut argv = vec![
        "respawn-pane".to_string(),
        "-k".to_string(),
        "-t".to_string(),
        "soda:0.0".to_string(),
        "-c".to_string(),
        home.to_string(),
    ];
    argv.extend(term::subscription_command(path));
    argv
}

/// `subscription_start`: deadline, root-seal the harness, digest pin,
/// binding recheck, liveness, one-shot `subscription-started` marker,
/// respawn the pane into the harness, liveness again.
pub fn subscription_start(
    request: &JsonValue,
    lease: &JsonValue,
    profile: &JsonValue,
    account: &Account,
) -> Result<JsonValue, String> {
    live_deadline(profile)?;
    let identifier = lease_execution_id(lease)?.to_string();
    let actor_id = lease_actor_id(lease)?;
    let path = term::terminal_path(&identifier)?;
    let harness = format!("{}/harness", subscription_path(&identifier)?);
    sys::run_checked(
        &[
            "/usr/bin/chown".to_string(),
            "-R".to_string(),
            "0:0".to_string(),
            harness.clone(),
        ],
        10,
    )
    .map_err(|e| format!("seal harness: {e}"))?;
    sys::run_checked(
        &[
            "/usr/bin/chmod".to_string(),
            "-R".to_string(),
            "go-w".to_string(),
            harness.clone(),
        ],
        10,
    )
    .map_err(|e| format!("seal harness: {e}"))?;
    let digest = subscription_harness_digest(&harness)?;
    let want = request.get("harness_sha256");
    let matches = want.and_then(|v| v.as_str()) == Some(digest.as_str());
    if !matches {
        return Err("guest harness digest differs".to_string());
    }
    let directory = term::checked_chain(&path)?;
    term::binding_record(&directory, Some(account), actor_id)?;
    drop(directory);
    subscription_check_unit(lease, account, true)?;
    let directory = term::checked_chain(&path)?;
    fs::new_file(&directory, "subscription-started", b"1", 0o600)?;
    drop(directory);
    svc::tmux_control(
        account,
        &format!("{path}/screen/socket"),
        &respawn_argv(&path, &account.pw_dir),
    )?;
    subscription_check_unit(lease, account, true)?;
    Ok(result_object(lease, ""))
}

/// `subscription_cgroup`: the owned service cgroup, root-owned and
/// group/other write-free.
pub fn subscription_cgroup(identifier: &str) -> Result<File, String> {
    let parent = svc::cgroup_parent()?;
    let group =
        sys::open_child_dir(&parent, &svc::unit_name(identifier)).map_err(|e| e.to_string())?;
    drop(parent);
    let (uid, mode) = fs::fstat_uid_mode(&group).map_err(|e| e.to_string())?;
    if uid != 0 || mode & 0o022 != 0 {
        return Err("unsafe cgroup".to_string());
    }
    Ok(group)
}

/// `subscription_kernel_write`: one full write to a cgroup control file.
pub fn subscription_kernel_write(directory: &File, name: &str, value: &[u8]) -> Result<(), String> {
    let fd = sys::open_at(directory, name, libc::O_WRONLY, 0).map_err(|e| e.to_string())?;
    let wrote = loop {
        let wrote = unsafe {
            libc::write(
                fd.as_raw_fd(),
                value.as_ptr() as *const libc::c_void,
                value.len(),
            )
        };
        if wrote < 0 {
            let err = std::io::Error::last_os_error();
            if err.raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            return Err(err.to_string());
        }
        break wrote as usize;
    };
    if wrote != value.len() {
        return Err("short cgroup write".to_string());
    }
    Ok(())
}

/// `KEY value` rows of a `cgroup.events` body, with the `.py`
/// `dict(row.split() ...)` shape (anything but a pair raises).
pub fn parse_cgroup_events(content: &[u8]) -> Result<Vec<(String, String)>, String> {
    if !content.is_ascii() {
        return Err("cgroup events encoding".to_string());
    }
    let text = std::str::from_utf8(content).map_err(|_| "cgroup events encoding".to_string())?;
    let body = text.strip_suffix('\n').unwrap_or(text);
    let mut values = Vec::new();
    if !body.is_empty() {
        for row in body.split('\n') {
            let row = row.strip_suffix('\r').unwrap_or(row);
            let parts: Vec<&str> = row.split_whitespace().collect();
            if parts.len() != 2 {
                return Err("cgroup events shape".to_string());
            }
            values.push((parts[0].to_string(), parts[1].to_string()));
        }
    }
    Ok(values)
}

/// `subscription_freeze`: freeze the cgroup and confirm `frozen 1` within 5s.
pub fn subscription_freeze(group: &File) -> Result<(), String> {
    subscription_kernel_write(group, "cgroup.freeze", b"1\n")?;
    let until = sys::monotonic() + 5.0;
    loop {
        if sys::monotonic() >= until {
            return Err("freeze unconfirmed".to_string());
        }
        let fd =
            sys::open_at(group, "cgroup.events", libc::O_RDONLY, 0).map_err(|e| e.to_string())?;
        let content = fs::read_up_to(&fd, 4096).map_err(|e| e.to_string())?;
        drop(fd);
        let values = parse_cgroup_events(&content)?;
        if values.iter().any(|(k, v)| k == "frozen" && v == "1") {
            return Ok(());
        }
        std::thread::sleep(std::time::Duration::from_millis(20));
    }
}

/// `subscription_capture`: validated read-back of the account-owned
/// credential file (regular, account uid, one link, `0o077`-clean,
/// bounded JSON).
pub fn subscription_capture(lease: &JsonValue, account: &Account) -> Result<Vec<u8>, String> {
    let directory = subscription_auth_directory(lease, account)?;
    let fd = sys::open_at(
        &directory,
        "auth.json",
        libc::O_RDONLY | libc::O_NONBLOCK,
        0,
    )
    .map_err(|e| e.to_string())?;
    drop(directory);
    let info = fs::fstat_all(&fd).map_err(|e| e.to_string())?;
    if !is_regular(info.st_mode)
        || info.st_uid != account.pw_uid
        || info.st_nlink != 1
        || info.st_mode & 0o077 != 0
    {
        return Err("unsafe credential file".to_string());
    }
    let state = fs::read_up_to(&fd, CREDENTIAL_LIMIT + 1).map_err(|e| e.to_string())?;
    if state.is_empty() || state.len() > CREDENTIAL_LIMIT {
        return Err("credential size".to_string());
    }
    let text = std::str::from_utf8(&state).map_err(|_| "credential json".to_string())?;
    JsonValue::parse(text).map_err(|_| "credential json".to_string())?;
    Ok(state)
}

/// Unlink one name, ignoring absence (other errors propagate).
fn unlink_missing_ok(dir: &File, name: &str) -> Result<(), String> {
    match fs::unlink_at(dir, name) {
        Ok(()) => Ok(()),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Err(err) => Err(err.to_string()),
    }
}

fn rmdir_missing_ok(dir: &File, name: &str) -> Result<(), String> {
    match fs::rmdir_at(dir, name) {
        Ok(()) => Ok(()),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Err(err) => Err(err.to_string()),
    }
}

/// `subscription_remove_model`: subscription records plus the model tree.
/// The `.py` removes `model/auth` and `model/harness` through slash paths
/// relative to the held dir fd; the equivalent here opens `model` once.
pub fn subscription_remove_model(directory: &File) -> Result<(), String> {
    for name in ["subscription", "subscription-started", "subscription-unit"] {
        unlink_missing_ok(directory, name)?;
    }
    match sys::open_child_dir(directory, "model") {
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(()),
        Err(err) => return Err(err.to_string()),
        Ok(model) => {
            rmdir_missing_ok(&model, "auth")?;
            rmdir_missing_ok(&model, "harness")?;
        }
    }
    rmdir_missing_ok(directory, "model")
}

/// `os.path.ismount`: device differs from the parent, or same file (binds
/// and filesystem roots); every stat failure is "not a mount".
pub fn is_mount(path: &str) -> bool {
    let target = match std::ffi::CString::new(path) {
        Ok(target) => target,
        Err(_) => return false,
    };
    let mut first: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::lstat(target.as_ptr(), &mut first) } != 0 {
        return false;
    }
    let parent = match path.rsplit_once('/') {
        Some((head, _)) if !head.is_empty() => head,
        _ => "/",
    };
    let parent = match std::ffi::CString::new(parent) {
        Ok(parent) => parent,
        Err(_) => return false,
    };
    let mut second: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::lstat(parent.as_ptr(), &mut second) } != 0 {
        return false;
    }
    first.st_dev != second.st_dev || first.st_ino == second.st_ino
}

/// `subscription_retire`: unmount the model mounts, remove the model tree
/// and the exact run-owned files.
pub fn subscription_retire(lease: &JsonValue, account: &Account) -> Result<(), String> {
    let identifier = lease_execution_id(lease)?.to_string();
    let path = term::terminal_path(&identifier)?;
    let model = subscription_path(&identifier)?;
    for name in ["auth", "harness"] {
        let mount = format!("{model}/{name}");
        if is_mount(&mount) {
            sys::run_checked(&["/usr/bin/umount".to_string(), mount], 5)
                .map_err(|e| format!("umount: {e}"))?;
        }
    }
    let directory = term::checked_chain(&path)?;
    subscription_remove_model(&directory)?;
    term::remove_owned_files(&path, &directory, account)?;
    Ok(())
}

/// `subscription_finish`: freeze a live unit, capture on `finish`, always
/// kill/close/stop, retire on `stop`. The `.py` `finally` order is exact:
/// a failed kill skips the stop, and a `finally` failure replaces the body
/// error.
pub fn subscription_finish(
    action: &str,
    lease: &JsonValue,
    _profile: &JsonValue,
    account: &Account,
) -> Result<JsonValue, String> {
    let identifier = lease_execution_id(lease)?.to_string();
    let live = subscription_check_unit(lease, account, false)?;
    let group = if live {
        Some(subscription_cgroup(&identifier)?)
    } else {
        None
    };
    let body: Result<Vec<u8>, String> = (|| {
        if let Some(group) = group.as_ref() {
            subscription_freeze(group)?;
        }
        if action == "finish" {
            return subscription_capture(lease, account);
        }
        Ok(Vec::new())
    })();
    let mut raised: Result<(), String> = Ok(());
    if let Some(group) = group {
        let killed = subscription_kernel_write(&group, "cgroup.kill", b"1\n");
        drop(group);
        if let Err(err) = killed {
            raised = Err(err);
        }
        // A failed kill aborts the rest of the `.py` finally body.
        if raised.is_ok() {
            if let Err(err) = svc::stop_service(&identifier, account) {
                raised = Err(err);
            }
        }
    } else if let Err(err) = svc::stop_service(&identifier, account) {
        raised = Err(err);
    }
    // A `finally` failure replaces the body error.
    raised?;
    let state = body?;
    if action == "stop" {
        subscription_retire(lease, account)?;
    }
    Ok(result_object(lease, &crate::b64::encode(&state)))
}

/// `subscription_lookup`: binding-checked observation, or the empty lease
/// for an absent locator or a locator without a subscription.
pub fn subscription_lookup(request: &JsonValue) -> Result<JsonValue, String> {
    let lease = request
        .get("delivery")
        .and_then(|d| d.get("lease"))
        .ok_or_else(|| "subscription request".to_string())?;
    let identifier = lease_execution_id(lease)?.to_string();
    let actor_id = lease_actor_id(lease)?;
    let login = request
        .get("login")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "subscription request".to_string())?;
    let account = account::account_for(login, actor_id)?;
    let path = term::terminal_path(&identifier)?;
    if path_missing(&path)? {
        return Ok(empty_result());
    }
    let directory = term::checked_chain(&path)?;
    term::binding_record(&directory, Some(&account), actor_id)?;
    if !term::record_exists(&directory, "subscription")? {
        return Ok(empty_result());
    }
    let profile = fs::read_record(&directory, "subscription")?;
    let stored = profile
        .get("lease")
        .ok_or_else(|| "subscription lease".to_string())?;
    let stored_id = stored
        .get("execution_id")
        .ok_or_else(|| "subscription lease".to_string())?;
    let want_id = lease
        .get("execution_id")
        .ok_or_else(|| "subscription lease".to_string())?;
    // The `.py` `int()` conversions raise on exotic values; unconvertible
    // actor IDs fail the binding like a mismatch does.
    let stored_actor = stored
        .get("actor_id")
        .and_then(json_int)
        .ok_or_else(|| "lookup binding".to_string())?;
    let want_actor = lease
        .get("actor_id")
        .and_then(json_int)
        .ok_or_else(|| "lookup binding".to_string())?;
    if !json_equal(stored_id, want_id) || stored_actor != want_actor {
        return Err("lookup binding".to_string());
    }
    Ok(result_object(stored, ""))
}

/// `subscription_resolve`: the stored profile, or `None` for a confirmed
/// terminated `stop` (inactive/failed unit plus an empty cgroup).
pub fn subscription_resolve(
    action: &str,
    lease: &JsonValue,
) -> Result<Option<(JsonValue, Account)>, String> {
    match subscription_profile(lease)? {
        ProfileHit::Found { profile, account } => Ok(Some((profile, account))),
        ProfileHit::Missing => {
            if action != "stop" {
                return Err("subscription missing".to_string());
            }
            let identifier = lease_execution_id(lease)?.to_string();
            let actor_id = lease_actor_id(lease)?;
            let login = lease
                .get("binding")
                .and_then(|b| b.get("login"))
                .and_then(|v| v.as_str())
                .ok_or_else(|| "subscription lease".to_string())?;
            let account = account::account_for(login, actor_id)?;
            let state = svc::service_state(&identifier, Some(&account))?;
            if (state != "inactive" && state != "failed") || !svc::cgroup_empty(&identifier)? {
                return Err("termination unknown".to_string());
            }
            Ok(None)
        }
    }
}

/// `subscription_dispatch`: op routing in `.py` order (prepare/lookup first,
/// then lease resolution, then stage/start/validate/finish/stop).
pub fn subscription_dispatch(request: &JsonValue) -> Result<JsonValue, String> {
    let action = request
        .get("action")
        .ok_or_else(|| "subscription request".to_string())?;
    if action == &JsonValue::Str("prepare".to_string()) {
        return subscription_prepare(request);
    }
    if action == &JsonValue::Str("lookup".to_string()) {
        return subscription_lookup(request);
    }
    let lease = request
        .get("delivery")
        .and_then(|d| d.get("lease"))
        .ok_or_else(|| "subscription request".to_string())?;
    let name = action.as_str().unwrap_or("");
    let resolved = subscription_resolve(name, lease)?;
    let Some((profile, account)) = resolved else {
        return Ok(result_object(lease, ""));
    };
    if name == "stage" {
        return subscription_stage(request, lease, &profile, &account);
    }
    if name == "start" {
        return subscription_start(request, lease, &profile, &account);
    }
    if name == "validate" {
        live_deadline(&profile)?;
        subscription_check_unit(lease, &account, true)?;
        return Ok(result_object(lease, ""));
    }
    if name == "finish" || name == "stop" {
        return subscription_finish(name, lease, &profile, &account);
    }
    Err("operation".to_string())
}

// ---------------------------------------------------------------------------
// Entry point.
// ---------------------------------------------------------------------------

enum BrokerFail {
    Error,
    Epipe,
}

fn read_stdin() -> Result<Vec<u8>, BrokerFail> {
    let mut body = Vec::new();
    std::io::stdin()
        .lock()
        .take((REQUEST_LIMIT + 1) as u64)
        .read_to_end(&mut body)
        .map_err(|_| BrokerFail::Error)?;
    Ok(body)
}

fn write_stdout_all(bytes: &[u8]) -> Result<(), BrokerFail> {
    let mut out = std::io::stdout().lock();
    match out.write_all(bytes).and_then(|()| out.flush()) {
        Ok(()) => Ok(()),
        Err(err) if err.raw_os_error() == Some(libc::EPIPE) => Err(BrokerFail::Epipe),
        Err(_) => Err(BrokerFail::Error),
    }
}

fn write_stderr_all(bytes: &[u8]) {
    let mut rest = bytes;
    while !rest.is_empty() {
        let wrote = unsafe { libc::write(2, rest.as_ptr() as *const libc::c_void, rest.len()) };
        if wrote < 0 {
            if std::io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
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

fn broker_inner() -> Result<(), BrokerFail> {
    let body = read_stdin()?;
    let request = decode_request(&body).map_err(|_| BrokerFail::Error)?;
    let parent = term::checked_chain(term::TERMINALS).map_err(|_| BrokerFail::Error)?;
    sys::flock_exclusive(&parent).map_err(|_| BrokerFail::Error)?;
    let result = subscription_dispatch(&request).map_err(|_| BrokerFail::Error)?;
    drop(parent);
    // `json.dumps` default separators, no trailing newline.
    write_stdout_all(pyemit::dumps_default(&result).as_bytes())
}

/// `subscription_main`: alarm, bounded read, exclusive dispatch, result.
/// EPIPE on stdout exits 0 silently.
pub fn broker_main() -> i32 {
    unsafe {
        libc::alarm(BROKER_ALARM_SECS);
    }
    match broker_inner() {
        Ok(()) => 0,
        Err(BrokerFail::Epipe) => 0,
        Err(BrokerFail::Error) => {
            write_stderr_all(b"managed Codex operation failed\n");
            1
        }
    }
}

#[cfg(test)]
#[path = "broker_tests.rs"]
mod broker_tests;
