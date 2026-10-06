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

use soda_json::JsonValue;

use crate::account::Account;
use crate::fs;
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

use crate::subscription_cgroup::{
    subscription_cgroup, subscription_freeze, subscription_kernel_write,
};
use crate::subscription_credentials::{subscription_capture, subscription_stage};
use crate::subscription_prepare::subscription_prepare;
use crate::subscription_profile::{
    lease_execution_id, live_deadline, subscription_check_unit, subscription_lookup,
    subscription_path, subscription_resolve,
};
use crate::subscription_start::subscription_start;
use crate::subscription_wire::{decode_request, result_object};

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
