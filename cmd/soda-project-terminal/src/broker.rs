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

use std::io::{Read, Write};

use crate::state_json::StateValue;

use crate::pyemit;
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

use crate::subscription_credentials::subscription_stage;
use crate::subscription_prepare::subscription_prepare;
use crate::subscription_profile::{
    live_deadline, subscription_check_unit, subscription_lookup, subscription_resolve,
};
use crate::subscription_retire::subscription_finish;
use crate::subscription_start::subscription_start;
use crate::subscription_wire::{decode_request, result_object};

/// `subscription_dispatch`: op routing in `.py` order (prepare/lookup first,
/// then lease resolution, then stage/start/validate/finish/stop).
pub fn subscription_dispatch(request: &StateValue) -> Result<StateValue, String> {
    let action = request
        .get("action")
        .ok_or_else(|| "subscription request".to_string())?;
    if action == &StateValue::Str("prepare".to_string()) {
        return subscription_prepare(request);
    }
    if action == &StateValue::Str("lookup".to_string()) {
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
