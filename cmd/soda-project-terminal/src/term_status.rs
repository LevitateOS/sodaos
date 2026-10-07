use std::fs::File;
use std::io;

use crate::state_json::StateValue;
use serde::de::{MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;

use crate::account::Account;
use crate::fs;
use crate::proto;
use crate::svc;
use crate::sys;
use crate::term_binding::{binding_record, permit_live, read_reservation, Reservation};
use crate::term_paths::{checked_chain, terminal_path};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum UnitClass {
    Ended,
    Opening,
    Ending,
    ReadyCheck,
}

/// Pure half of `classify_unit_state`: the caller checks cgroup emptiness for
/// stopped units first, preserving the `.py` error order.
pub fn classify_bare_state(
    state: &str,
    permit_present: bool,
    live_permit: bool,
) -> Result<UnitClass, String> {
    if state == "inactive" || state == "failed" {
        return Ok(if live_permit {
            UnitClass::Opening
        } else {
            UnitClass::Ended
        });
    }
    if permit_present {
        return Err("unconsumed reservation has a native unit".to_string());
    }
    match state {
        "active" => Ok(UnitClass::ReadyCheck),
        "activating" => Ok(UnitClass::Opening),
        "deactivating" => Ok(UnitClass::Ending),
        _ => Err("terminal state unavailable".to_string()),
    }
}

/// Status object in `.py` key order: id, name, created_at, ready, attached,
/// state.
pub fn status_object(
    identifier: &str,
    name: &str,
    created_at: i64,
    ready: bool,
    attached: bool,
    state: &str,
) -> StateValue {
    StateValue::Object(vec![
        ("id".to_string(), StateValue::Str(identifier.to_string())),
        ("name".to_string(), StateValue::Str(name.to_string())),
        (
            "created_at".to_string(),
            StateValue::Number(created_at.to_string()),
        ),
        ("ready".to_string(), StateValue::Bool(ready)),
        ("attached".to_string(), StateValue::Bool(attached)),
        ("state".to_string(), StateValue::Str(state.to_string())),
    ])
}

/// `(pid, dev, ino)` from a `ready` record.
struct ReadyFields(std::collections::HashMap<String, Box<RawValue>>);

impl<'de> Deserialize<'de> for ReadyFields {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct ReadyVisitor;
        impl<'de> Visitor<'de> for ReadyVisitor {
            type Value = ReadyFields;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("a terminal ready record")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let mut fields = std::collections::HashMap::new();
                while let Some(name) = map.next_key::<String>()? {
                    fields.insert(name, map.next_value::<Box<RawValue>>()?);
                }
                Ok(ReadyFields(fields))
            }
        }
        deserializer.deserialize_map(ReadyVisitor)
    }
}

pub(crate) fn parse_ready(text: &str) -> Result<(i32, u64, u64), String> {
    let fields: ReadyFields =
        serde_json::from_str(text).map_err(|_| "terminal ready".to_string())?;
    let pid_raw = fields
        .0
        .get("pid")
        .ok_or_else(|| "terminal ready".to_string())?;
    let pid: i64 = pid_raw
        .get()
        .parse()
        .map_err(|_| "terminal ready".to_string())?;
    let pid = i32::try_from(pid).map_err(|_| "terminal ready".to_string())?;
    let socket_raw = fields
        .0
        .get("socket")
        .ok_or_else(|| "terminal ready".to_string())?;
    let (dev, ino): (Box<RawValue>, Box<RawValue>) =
        serde_json::from_str(socket_raw.get()).map_err(|_| "terminal ready".to_string())?;
    let dev: i128 = dev
        .get()
        .parse()
        .map_err(|_| "terminal ready".to_string())?;
    let ino: i128 = ino
        .get()
        .parse()
        .map_err(|_| "terminal ready".to_string())?;
    let dev = u64::try_from(dev).map_err(|_| "terminal ready".to_string())?;
    let ino = u64::try_from(ino).map_err(|_| "terminal ready".to_string())?;
    Ok((pid, dev, ino))
}

/// True while another attach holds the writer lock.
fn writer_attached(directory: &File) -> Result<bool, String> {
    let writer = fs::root_file(directory, "writer", true)?;
    let locked = sys::flock_exclusive_nb(&writer).map_err(|e| e.to_string())?;
    Ok(!locked)
}

fn observe_ready_terminal(path: &str, directory: &File, account: &Account) -> Result<bool, String> {
    let ready = parse_ready(&fs::read_record_text_at(directory, "ready")?)?;
    let screen = checked_chain(&format!("{path}/screen"))?;
    drop(screen);
    let sock = format!("{path}/screen/socket");
    if svc::socket_identity(&sock, account, ready.0)? != (ready.1, ready.2) {
        return Err("terminal socket changed".to_string());
    }
    writer_attached(directory)
}

fn classify_unit_state(
    path: &str,
    directory: &File,
    account: &Account,
    identifier: &str,
    state: &str,
    permit: Option<&Reservation>,
) -> Result<(String, bool), String> {
    if (state == "inactive" || state == "failed") && !svc::cgroup_empty(identifier)? {
        return Err("terminal cleanup unconfirmed".to_string());
    }
    match classify_bare_state(state, permit.is_some(), permit_live(permit))? {
        UnitClass::Ended => Ok(("ended".to_string(), false)),
        UnitClass::Opening => Ok(("opening".to_string(), false)),
        UnitClass::Ending => Ok(("ending".to_string(), false)),
        UnitClass::ReadyCheck => {
            let attached = observe_ready_terminal(path, directory, account)?;
            Ok(("ready".to_string(), attached))
        }
    }
}

/// Current status, or `None` for an observed absence.
pub fn terminal_status(
    identifier: &str,
    account: &Account,
    identity: i64,
) -> Result<Option<StateValue>, String> {
    let path = terminal_path(identifier)?;
    // Absence probe first: `checked_chain` cannot report `NotFound` through
    // `String`, and this always runs under the parent lock.
    match std::fs::symlink_metadata(&path) {
        Err(err) if err.kind() == io::ErrorKind::NotFound => {
            let state = svc::service_state(identifier, None)?;
            if (state != "inactive" && state != "failed") || !svc::cgroup_empty(identifier)? {
                return Err("unidentified native terminal".to_string());
            }
            return Ok(None);
        }
        Err(err) => return Err(err.to_string()),
        Ok(_) => {}
    }
    let directory = checked_chain(&path)?;
    let record = binding_record(&directory, Some(account), identity)?;
    let permit = read_reservation(&directory)?;
    let state = svc::service_state(identifier, Some(account))?;
    let (state, attached) = classify_unit_state(
        &path,
        &directory,
        account,
        identifier,
        &state,
        permit.as_ref(),
    )?;
    let name = fs::read_record(&directory, "name")?;
    let name = name.as_str().ok_or_else(|| "terminal name".to_string())?;
    if !proto::valid_name(name) {
        return Err("terminal name".to_string());
    }
    Ok(Some(status_object(
        identifier,
        name,
        record.created_at,
        state == "ready",
        attached,
        &state,
    )))
}

pub(crate) fn ready_object(pid: i64, dev: u64, ino: u64) -> StateValue {
    StateValue::Object(vec![
        ("pid".to_string(), StateValue::Number(pid.to_string())),
        (
            "socket".to_string(),
            StateValue::Array(vec![
                StateValue::Number(dev.to_string()),
                StateValue::Number(ino.to_string()),
            ]),
        ),
    ])
}
