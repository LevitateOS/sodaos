use crate::state_json::StateValue;

use crate::account::{self, Account};
use crate::fs;
use crate::proto;
use crate::pyemit;
use crate::subscription_wire::{empty_result, json_equal, json_int, result_object};
use crate::svc;
use crate::sys;
use crate::term;

// ---------------------------------------------------------------------------
// Filesystem operations.
// ---------------------------------------------------------------------------

pub(crate) fn subscription_path(identifier: &str) -> Result<String, String> {
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

pub(crate) fn lease_execution_id(lease: &StateValue) -> Result<&str, String> {
    lease
        .get("execution_id")
        .and_then(|v| v.as_str())
        .ok_or_else(|| "subscription lease".to_string())
}

pub(crate) fn lease_actor_id(lease: &StateValue) -> Result<i64, String> {
    lease
        .get("actor_id")
        .and_then(json_int)
        .ok_or_else(|| "subscription lease".to_string())
}

fn profile_deadline(profile: &StateValue) -> Result<i64, String> {
    profile
        .get("deadline")
        .and_then(pyemit::as_int)
        .ok_or_else(|| "subscription deadline".to_string())
}

pub(crate) fn live_deadline(profile: &StateValue) -> Result<(), String> {
    if profile_deadline(profile)? <= sys::now_secs() {
        return Err("session expired".to_string());
    }
    Ok(())
}

/// `subscription_profile` outcome: `Missing` replays the `.py`
/// `FileNotFoundError` path (absent locator, subscription, or binding).
pub enum ProfileHit {
    Found {
        profile: StateValue,
        account: Account,
    },
    Missing,
}

pub fn subscription_profile(lease: &StateValue) -> Result<ProfileHit, String> {
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
    lease: &StateValue,
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

/// `subscription_lookup`: binding-checked observation, or the empty lease
/// for an absent locator or a locator without a subscription.
pub fn subscription_lookup(request: &StateValue) -> Result<StateValue, String> {
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
    lease: &StateValue,
) -> Result<Option<(StateValue, Account)>, String> {
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
