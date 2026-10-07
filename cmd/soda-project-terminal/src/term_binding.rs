use std::collections::HashSet;
use std::fs::File;
use std::io;

use crate::state_json::StateValue;
use serde::de::{MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;

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

struct RawObject(std::collections::HashMap<String, Box<RawValue>>);

fn raw<'a>(fields: &'a RawObject, key: &str) -> Result<&'a str, String> {
    fields
        .0
        .get(key)
        .map(|value| value.get())
        .ok_or_else(|| "terminal binding".to_string())
}

impl<'de> Deserialize<'de> for RawObject {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct ObjectVisitor;
        impl<'de> Visitor<'de> for ObjectVisitor {
            type Value = RawObject;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("a JSON object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let mut fields = std::collections::HashMap::new();
                while let Some(name) = map.next_key::<String>()? {
                    fields.insert(name, map.next_value::<Box<RawValue>>()?);
                }
                Ok(RawObject(fields))
            }
        }
        deserializer.deserialize_map(ObjectVisitor)
    }
}

/// Validate a `binding` record (dup-tolerant like `json.loads`, unlike the
/// control-frame decoder). Uid/gid widen to `u32` at most: larger JSON ints
/// error here instead of failing later comparisons, a deliberate hardening
/// delta confined to corrupt root-owned files.
pub fn validate_binding(text: &str) -> Result<BindingRecord, String> {
    let fields: RawObject =
        serde_json::from_str(text).map_err(|_| "terminal binding".to_string())?;
    let keys: HashSet<&str> = fields.0.keys().map(String::as_str).collect();
    let want: HashSet<&str> = ["account", "identity", "cols", "rows", "created_at"]
        .into_iter()
        .collect();
    if keys != want {
        return Err("terminal binding".to_string());
    }
    let (login, uid_raw, gid_raw, home, shell): (
        String,
        Box<RawValue>,
        Box<RawValue>,
        String,
        String,
    ) = serde_json::from_str(raw(&fields, "account")?)
        .map_err(|_| "terminal binding values".to_string())?;
    let uid_raw: i128 = uid_raw
        .get()
        .parse()
        .map_err(|_| "terminal binding values".to_string())?;
    let gid_raw: i128 = gid_raw
        .get()
        .parse()
        .map_err(|_| "terminal binding values".to_string())?;
    let uid = u32::try_from(uid_raw)
        .ok()
        .filter(|uid| *uid > 0)
        .ok_or_else(|| "terminal binding values".to_string())?;
    let gid = u32::try_from(gid_raw).map_err(|_| "terminal binding values".to_string())?;
    let integer = |key: &str| -> Result<i64, String> {
        raw(&fields, key)?
            .parse()
            .map_err(|_| "terminal binding values".to_string())
    };
    let identity = integer("identity")?;
    let cols = integer("cols")?;
    let rows = integer("rows")?;
    let created_at = integer("created_at")?;
    if !account::valid_login(&login) || login == "root" {
        return Err("terminal binding values".to_string());
    }
    if !home.starts_with('/') || !shell.starts_with('/') {
        return Err("terminal binding values".to_string());
    }
    if identity <= 0 || !proto::dimensions(cols, rows) {
        return Err("terminal binding values".to_string());
    }
    if !(1..=9007199254740991).contains(&created_at) {
        return Err("terminal binding values".to_string());
    }
    Ok(BindingRecord {
        login,
        uid,
        gid,
        home,
        shell,
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
    let record = validate_binding(&fs::read_record_text_at(dir, "binding")?)?;
    if let Some(held) = account {
        // Exact `.py` comparison: stored vector vs `account_binding`.
        let stored = StateValue::Array(vec![
            StateValue::Str(record.login.clone()),
            StateValue::Number(record.uid.to_string()),
            StateValue::Number(record.gid.to_string()),
            StateValue::Str(record.home.clone()),
            StateValue::Str(record.shell.clone()),
        ]);
        if record.identity != identity || stored != account::account_binding(held) {
            return Err("terminal account changed".to_string());
        }
    }
    Ok(record)
}

/// Validate a `reservation` record.
pub fn validate_reservation(text: &str) -> Result<Reservation, String> {
    let fields: RawObject =
        serde_json::from_str(text).map_err(|_| "invalid creation reservation".to_string())?;
    let keys: HashSet<&str> = fields.0.keys().map(String::as_str).collect();
    let want: HashSet<&str> = ["expires", "scope"].into_iter().collect();
    if keys != want {
        return Err("invalid creation reservation".to_string());
    }
    let expires: i64 = fields
        .0
        .get("expires")
        .ok_or_else(|| "invalid creation reservation".to_string())?
        .get()
        .parse()
        .map_err(|_| "invalid creation reservation".to_string())?;
    let scope: String = serde_json::from_str(
        fields
            .0
            .get("scope")
            .ok_or_else(|| "invalid creation reservation".to_string())?
            .get(),
    )
    .map_err(|_| "invalid creation reservation".to_string())?;
    if expires <= 0 || !proto::valid_scope(&scope) {
        return Err("invalid creation reservation".to_string());
    }
    Ok(Reservation { expires, scope })
}

/// Read the reservation, if any (missing file → `None`).
pub(crate) fn read_reservation(dir: &File) -> Result<Option<Reservation>, String> {
    if !record_exists(dir, "reservation")? {
        return Ok(None);
    }
    validate_reservation(&fs::read_record_text_at(dir, "reservation")?).map(Some)
}

pub(crate) fn permit_live(permit: Option<&Reservation>) -> bool {
    permit
        .map(|p| p.expires > timex::now_secs())
        .unwrap_or(false)
}

pub(crate) fn binding_object(record: &BindingRecord) -> StateValue {
    StateValue::Object(vec![
        (
            "account".to_string(),
            StateValue::Array(vec![
                StateValue::Str(record.login.clone()),
                StateValue::Number(record.uid.to_string()),
                StateValue::Number(record.gid.to_string()),
                StateValue::Str(record.home.clone()),
                StateValue::Str(record.shell.clone()),
            ]),
        ),
        (
            "identity".to_string(),
            StateValue::Number(record.identity.to_string()),
        ),
        (
            "cols".to_string(),
            StateValue::Number(record.cols.to_string()),
        ),
        (
            "rows".to_string(),
            StateValue::Number(record.rows.to_string()),
        ),
        (
            "created_at".to_string(),
            StateValue::Number(record.created_at.to_string()),
        ),
    ])
}

pub(crate) fn reservation_object(expires: i64, scope: &str) -> StateValue {
    StateValue::Object(vec![
        (
            "expires".to_string(),
            StateValue::Number(expires.to_string()),
        ),
        ("scope".to_string(), StateValue::Str(scope.to_string())),
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
        &pyemit::line(&StateValue::Str(name.to_string())),
        0o600,
    )?;
    fs::replace_at(directory, "name.next", "name").map_err(|e| e.to_string())?;
    Ok(())
}
