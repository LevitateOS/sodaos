//! Terminal lifecycle (port of the `TERMINALS` half of
//! `project_terminal.py`: reserve/create/attach/inspect/list/end/rename plus
//! `prepare`).
//!
//! Merge-safe: uses only the brief-pinned `sys`/`fs`/`timex` signatures plus
//! the scaffold (`proto`/`pyemit`/`b64`/`sha`), `libc`, and Serde JSON.
//!
//! Two deliberate deltas from the `.py`, both documented at their sites:
//! 1. `reserve` streams the whole `project-terminal` binary for the
//!    `source_hash` check (no 64KB cap — the binary replaces the script).
//! 2. `String` errors cannot separate `NotFound` from other I/O failures, so
//!    absence checks use an explicit existence probe first. Every such site
//!    runs under the TERMINALS parent lock, where concurrent mutation (the
//!    only way the probe could go stale) is impossible.

use std::io;

use crate::state_json::StateValue;

use crate::account::Account;
use crate::svc;
use crate::sys;
pub(crate) use crate::term_attach::{attach_terminal, subscription_command};
pub(crate) use crate::term_binding::binding_record;
use crate::term_binding::{binding_matches_account, read_reservation, write_name};
pub(crate) use crate::term_collect::remove_owned_files;
use crate::term_collect::terminal_directories;
pub(crate) use crate::term_create::{create_terminal, file_sha256_hex, reserve_terminal};
use crate::term_paths::list_dir_names;
pub(crate) use crate::term_paths::{checked_chain, record_exists, terminal_path, TERMINALS};
pub(crate) use crate::term_prepare::prepare;
use crate::term_status::terminal_status;

// ---------------------------------------------------------------------------
// List / mutate / control.
// ---------------------------------------------------------------------------

fn list_owned_terminals(account: &Account, identity: i64) -> Result<Vec<StateValue>, String> {
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
                if value.get("state").and_then(|state| state.as_str()) != Some("ended") {
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
) -> Result<Vec<StateValue>, String> {
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
) -> Result<Vec<StateValue>, String> {
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

#[cfg(test)]
#[path = "term_binding_tests.rs"]
mod term_binding_tests;

#[cfg(test)]
#[path = "term_protocol_tests.rs"]
mod term_protocol_tests;

#[cfg(test)]
#[path = "term_native_tests.rs"]
mod term_native_tests;
