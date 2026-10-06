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

use std::io;

use soda_json::JsonValue;

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
use crate::term_status::{status_state, terminal_status};

// ---------------------------------------------------------------------------
// Binding + reservation records.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Status classification.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Inventory + retirement.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Reserve.
// ---------------------------------------------------------------------------

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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sha;
    use crate::term::term_binding_tests::{parse, sample};
    use crate::term_attach::subscription_lifetime;
    use crate::term_collect::collect_finished;
    use crate::term_create::{file_sha256_hex, program_stat_ok};
    use crate::term_paths::TMUX_CONFIG;
    use crate::term_status::parse_ready;

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

#[cfg(test)]
#[path = "term_binding_tests.rs"]
mod term_binding_tests;

#[cfg(test)]
#[path = "term_protocol_tests.rs"]
mod term_protocol_tests;
