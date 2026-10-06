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
use crate::timex;

/// Stdin cap: `sys.stdin.buffer.read(512 * 1024 + 1)`; longer input is a
/// `request size` failure, never truncation.
pub const REQUEST_LIMIT: usize = 512 * 1024;
/// Credential size bound (`0 < len <= 256KiB`), both directions.
pub const CREDENTIAL_LIMIT: usize = 256 * 1024;
/// Subscription horizon bound (`0 < deadline - now <= 12h`).
pub const HORIZON_SECS: i64 = 12 * 3600;
/// Whole-body broker alarm, in seconds.
pub const BROKER_ALARM_SECS: u32 = 45;

// ---------------------------------------------------------------------------
// Pure JSON helpers.
// ---------------------------------------------------------------------------

/// Last value wins for duplicate keys, like `.py` `json.loads` dicts.
fn last<'a>(entries: &'a [(String, JsonValue)], key: &str) -> Option<&'a JsonValue> {
    entries.iter().rev().find(|(k, _)| k == key).map(|(_, v)| v)
}

fn number_f64(raw: &str) -> Option<f64> {
    raw.parse::<f64>().ok().filter(|n| n.is_finite())
}

/// Numeric equivalence for JSON numbers: `i128` compare when both are plain
/// ints, else finite-float compare (so `100` equals `1e2`, like `.py`).
fn number_equal(a: &str, b: &str) -> bool {
    let left = JsonValue::Number(a.to_string());
    let right = JsonValue::Number(b.to_string());
    match (left.as_integer(), right.as_integer()) {
        (Some(x), Some(y)) => x == y,
        _ => match (number_f64(a), number_f64(b)) {
            (Some(x), Some(y)) => x == y,
            _ => a == b,
        },
    }
}

/// Order-insensitive deep equality mirroring `.py` `dict ==`:
/// objects compare by key set with last-wins values, arrays pairwise,
/// numbers numerically, `True == 1` / `False == 0`.
pub fn json_equal(a: &JsonValue, b: &JsonValue) -> bool {
    match (a, b) {
        (JsonValue::Null, JsonValue::Null) => true,
        (JsonValue::Bool(x), JsonValue::Bool(y)) => x == y,
        (JsonValue::Bool(x), JsonValue::Number(raw))
        | (JsonValue::Number(raw), JsonValue::Bool(x)) => {
            number_f64(raw) == Some(f64::from(u8::from(*x)))
        }
        (JsonValue::Number(x), JsonValue::Number(y)) => number_equal(x, y),
        (JsonValue::Str(x), JsonValue::Str(y)) => x == y,
        (JsonValue::Array(x), JsonValue::Array(y)) => {
            x.len() == y.len() && x.iter().zip(y.iter()).all(|(u, v)| json_equal(u, v))
        }
        (JsonValue::Object(x), JsonValue::Object(y)) => {
            let mut xkeys: Vec<&str> = x.iter().map(|(k, _)| k.as_str()).collect();
            let mut ykeys: Vec<&str> = y.iter().map(|(k, _)| k.as_str()).collect();
            xkeys.sort_unstable();
            xkeys.dedup();
            ykeys.sort_unstable();
            ykeys.dedup();
            if xkeys != ykeys {
                return false;
            }
            xkeys.iter().all(|k| match (last(x, k), last(y, k)) {
                (Some(u), Some(v)) => json_equal(u, v),
                _ => false,
            })
        }
        _ => false,
    }
}

/// The `.py` `int()` conversions for lease `actor_id`: JSON ints, floats
/// truncated toward zero, and (trimmed, optionally signed) plain numeric
/// strings.
pub fn json_int(value: &JsonValue) -> Option<i64> {
    match value {
        JsonValue::Number(raw) => {
            if let Some(n) = pyemit::as_int(value) {
                return Some(n);
            }
            // `.py` `int()`: floats truncate toward zero.
            let n = number_f64(raw)?.trunc();
            const LO: f64 = -9_223_372_036_854_775_808.0;
            const HI: f64 = 9_223_372_036_854_775_808.0;
            if n < LO || n >= HI {
                return None;
            }
            #[allow(clippy::cast_possible_truncation)]
            Some(n as i64)
        }
        // `.py` `int(True) == 1`, `int(False) == 0`.
        JsonValue::Bool(b) => Some(i64::from(*b)),
        JsonValue::Str(text) => {
            let trimmed = text.trim();
            let (negative, digits) = match trimmed.strip_prefix('-') {
                Some(rest) => (true, rest),
                None => (false, trimmed.strip_prefix('+').unwrap_or(trimmed)),
            };
            if digits.is_empty() || !digits.bytes().all(|b| b.is_ascii_digit()) {
                return None;
            }
            let n: i64 = digits.parse().ok()?;
            Some(if negative { -n } else { n })
        }
        _ => None,
    }
}

/// `lease['binding'] = native`: in-place replace, else append (dict order
/// preserved like the `.py` mutation).
pub fn lease_with_binding(lease: &JsonValue, native: JsonValue) -> Option<JsonValue> {
    match lease {
        JsonValue::Object(entries) => {
            let mut out = entries.clone();
            if let Some(slot) = out.iter_mut().find(|(k, _)| k == "binding") {
                slot.1 = native;
            } else {
                out.push(("binding".to_string(), native));
            }
            Some(JsonValue::Object(out))
        }
        _ => None,
    }
}

/// `0 < deadline - now <= 12h`.
pub fn deadline_ok(deadline: i64, now: i64) -> bool {
    let rest = deadline.saturating_sub(now);
    rest > 0 && rest <= HORIZON_SECS
}

/// `{'lease': lease, 'credential': ...}` in `.py` key order.
pub fn result_object(lease: &JsonValue, credential: &str) -> JsonValue {
    JsonValue::Object(vec![
        ("lease".to_string(), lease.clone()),
        (
            "credential".to_string(),
            JsonValue::Str(credential.to_string()),
        ),
    ])
}

/// Empty-lease lookup result: `{'lease': {}, 'credential': ''}`.
pub fn empty_result() -> JsonValue {
    result_object(&JsonValue::Object(Vec::new()), "")
}

/// Pure request decode: size cap plus JSON parse (last-wins, no shape or
/// duplicate checks — `subscription_main` runs plain `json.loads`).
pub fn decode_request(body: &[u8]) -> Result<JsonValue, String> {
    if body.len() > REQUEST_LIMIT {
        return Err("request size".to_string());
    }
    let text = std::str::from_utf8(body).map_err(|_| "request encoding".to_string())?;
    JsonValue::parse(text).map_err(|_| "request json".to_string())
}

/// Native binding record in `.py` key order.
pub fn native_binding(
    identifier: &str,
    project: &JsonValue,
    login: &str,
    generation: &JsonValue,
) -> JsonValue {
    JsonValue::Object(vec![
        ("kind".to_string(), JsonValue::Str("terminal".to_string())),
        ("id".to_string(), JsonValue::Str(identifier.to_string())),
        ("project".to_string(), project.clone()),
        ("login".to_string(), JsonValue::Str(login.to_string())),
        ("generation".to_string(), generation.clone()),
    ])
}

/// Stored subscription profile in `.py` key order.
pub fn profile_object(
    lease: &JsonValue,
    binding: &JsonValue,
    deadline: i64,
    scope: &str,
) -> JsonValue {
    JsonValue::Object(vec![
        ("lease".to_string(), lease.clone()),
        ("binding".to_string(), binding.clone()),
        (
            "deadline".to_string(),
            JsonValue::Number(deadline.to_string()),
        ),
        ("scope".to_string(), JsonValue::Str(scope.to_string())),
    ])
}

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

fn lease_execution_id(lease: &JsonValue) -> Result<&str, String> {
    lease
        .get("execution_id")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "subscription lease".to_string())
}

fn lease_actor_id(lease: &JsonValue) -> Result<i64, String> {
    lease
        .get("actor_id")
        .and_then(json_int)
        .ok_or_else(|| "subscription lease".to_string())
}

fn mount_argv(path: &str, options: &str) -> Vec<String> {
    vec![
        "/usr/bin/mount".to_string(),
        "-t".to_string(),
        "tmpfs".to_string(),
        "-o".to_string(),
        options.to_string(),
        "tmpfs".to_string(),
        path.to_string(),
    ]
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

/// `subscription_prepare`: deadline gate, reserve, provision; provision
/// failures stop the unit and retire before propagating (reserve failures
/// propagate without cleanup, exactly like the `.py`).
pub fn subscription_prepare(request: &JsonValue) -> Result<JsonValue, String> {
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
    let deadline_text = lease
        .get("deadline")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "deadline".to_string())?;
    let deadline =
        timex::parse_iso_deadline(deadline_text).ok_or_else(|| "deadline".to_string())?;
    if !deadline_ok(deadline, sys::now_secs()) {
        return Err("deadline".to_string());
    }
    let cols = request
        .get("cols")
        .and_then(pyemit::as_int)
        .ok_or_else(|| "subscription request".to_string())?;
    let rows = request
        .get("rows")
        .and_then(pyemit::as_int)
        .ok_or_else(|| "subscription request".to_string())?;
    let source_hash = request
        .get("source_hash")
        .ok_or_else(|| "subscription request".to_string())?;
    let source_hash = source_hash
        .as_str()
        .ok_or_else(|| "terminal support version differs".to_string())?;
    let scope = request
        .get("scope")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "subscription request".to_string())?;
    // Held across preparation like the `.py` (no flock here: the broker
    // entry holds the exclusive parent lock across the whole dispatch).
    let parent = term::checked_chain(term::TERMINALS)?;
    let reserved = term::reserve_terminal(
        &identifier,
        &account,
        actor_id,
        cols,
        rows,
        "Codex",
        source_hash,
        scope,
    );
    if let Err(err) = reserved {
        drop(parent);
        return Err(err);
    }
    let provisioned = subscription_provision(request, lease, &account, deadline, cols, rows, scope);
    match provisioned {
        Ok(result) => {
            drop(parent);
            Ok(result)
        }
        Err(original) => {
            // Cleanup failures replace the original error, in `.py` order:
            // a failed stop skips the retire.
            if let Err(err) = svc::stop_service(&identifier, &account) {
                drop(parent);
                return Err(err);
            }
            if let Err(err) = subscription_retire(lease, &account) {
                drop(parent);
                return Err(err);
            }
            drop(parent);
            Err(original)
        }
    }
}

#[allow(clippy::too_many_arguments)]
fn subscription_provision(
    request: &JsonValue,
    lease: &JsonValue,
    account: &Account,
    deadline: i64,
    cols: i64,
    rows: i64,
    scope: &str,
) -> Result<JsonValue, String> {
    let identifier = lease_execution_id(lease)?.to_string();
    let actor_id = lease_actor_id(lease)?;
    let path = term::terminal_path(&identifier)?;
    let directory = term::checked_chain(&path)?;
    let project = request
        .get("container")
        .ok_or_else(|| "subscription request".to_string())?;
    let generation = lease
        .get("generation")
        .ok_or_else(|| "subscription lease".to_string())?;
    let native = native_binding(&identifier, project, &account.pw_name, generation);
    let bound = lease_with_binding(lease, native.clone())
        .ok_or_else(|| "subscription lease".to_string())?;
    let profile = profile_object(&bound, &native, deadline, scope);
    fs::new_file(&directory, "subscription", &pyemit::line(&profile), 0o600)?;
    fs::mkdir_at(&directory, "model", 0o711).map_err(|e| e.to_string())?;
    let model_path = format!("{path}/model");
    fs::chmod_path(&model_path, 0o711, false).map_err(|e| e.to_string())?;
    let model = sys::open_child_dir(&directory, "model").map_err(|e| e.to_string())?;
    fs::mkdir_at(&model, "harness", 0o755).map_err(|e| e.to_string())?;
    fs::mkdir_at(&model, "auth", 0o700).map_err(|e| e.to_string())?;
    drop(model);
    drop(directory);
    let harness = format!("{model_path}/harness");
    let auth = format!("{model_path}/auth");
    sys::run_checked(&mount_argv(&harness, "size=1g,nosuid,nodev,mode=755"), 5)
        .map_err(|e| format!("mount harness: {e}"))?;
    sys::run_checked(&mount_argv(&auth, "size=64m,nosuid,nodev,mode=700"), 5)
        .map_err(|e| format!("mount auth: {e}"))?;
    fs::chown_path(&auth, account.pw_uid, account.pw_gid, false).map_err(|e| e.to_string())?;
    term::create_terminal(&identifier, account, actor_id, cols, rows, "Codex", scope)?;
    subscription_check_unit(&bound, account, true)?;
    Ok(result_object(&bound, ""))
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
