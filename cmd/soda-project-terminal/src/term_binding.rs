use std::collections::HashSet;
use std::fs::File;
use std::io;

use soda_json::JsonValue;

use crate::account::{self, Account};
use crate::fs;
use crate::proto;
use crate::pyemit;
use crate::term_paths::record_exists;
use crate::timex;

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
pub(crate) fn read_reservation(dir: &File) -> Result<Option<Reservation>, String> {
    if !record_exists(dir, "reservation")? {
        return Ok(None);
    }
    validate_reservation(&fs::read_record(dir, "reservation")?).map(Some)
}

pub(crate) fn permit_live(permit: Option<&Reservation>) -> bool {
    permit
        .map(|p| p.expires > timex::now_secs())
        .unwrap_or(false)
}

pub(crate) fn binding_object(record: &BindingRecord) -> JsonValue {
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

pub(crate) fn reservation_object(expires: i64, scope: &str) -> JsonValue {
    JsonValue::Object(vec![
        (
            "expires".to_string(),
            JsonValue::Number(expires.to_string()),
        ),
        ("scope".to_string(), JsonValue::Str(scope.to_string())),
    ])
}

pub(crate) fn write_name(directory: &File, name: &str) -> Result<(), String> {
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
