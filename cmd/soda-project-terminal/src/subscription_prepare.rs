use soda_json::JsonValue;

use crate::account::{self, Account};
use crate::fs;
use crate::pyemit;
use crate::subscription_profile::{lease_actor_id, lease_execution_id, subscription_check_unit};
use crate::subscription_retire::subscription_retire;
use crate::subscription_wire::{
    deadline_ok, lease_with_binding, native_binding, profile_object, result_object,
};
use crate::svc;
use crate::sys;
use crate::term;
use crate::timex;

pub(crate) fn mount_argv(path: &str, options: &str) -> Vec<String> {
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
