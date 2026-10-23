use std::io::{Read, Write};
use std::os::fd::AsRawFd;
use std::time::{Instant, SystemTime, UNIX_EPOCH};

use serde_json::{json, Value};

use super::{Connection, Error, Notice};

#[derive(Clone, Debug)]
pub struct SessionSnapshot {
    pub id: String,
    pub status: String,
    pub active_turn: Option<String>,
    pub turn_count: u64,
}

#[derive(Clone, Debug)]
pub struct TurnAdmission {
    pub command: String,
    pub turn: String,
}

/// Mint once, persist before the command, and reuse only for that exact native
/// operation. UUIDv7 is the native command identity contract, not a tool pin.
pub fn new_command_id() -> Result<String, Error> {
    let mut bytes = [0u8; 16];
    getrandom::fill(&mut bytes).map_err(|_| Error::Invalid("native command entropy"))?;
    let millis = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|_| Error::Invalid("native command clock"))?
        .as_millis();
    if millis > 0xffff_ffff_ffff {
        return Err(Error::Bounds);
    }
    let time = (millis as u64).to_be_bytes();
    bytes[..6].copy_from_slice(&time[2..]);
    bytes[6] = (bytes[6] & 0x0f) | 0x70;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    let hex: String = bytes.iter().map(|b| format!("{b:02x}")).collect();
    Ok(format!(
        "{}-{}-{}-{}-{}",
        &hex[..8],
        &hex[8..12],
        &hex[12..16],
        &hex[16..20],
        &hex[20..]
    ))
}

fn command(value: &str) -> Result<(), Error> {
    let bytes = value.as_bytes();
    if bytes.len() != 36
        || bytes[14] != b'7'
        || !b"89ab".contains(&bytes[19])
        || bytes.iter().enumerate().any(|(n, b)| {
            if [8, 13, 18, 23].contains(&n) {
                *b != b'-'
            } else {
                !b.is_ascii_digit() && !(b'a'..=b'f').contains(b)
            }
        })
    {
        return Err(Error::Invalid("native command identity"));
    }
    Ok(())
}

fn snapshot(value: &Value) -> Result<SessionSnapshot, Error> {
    let text = |key| {
        value
            .get(key)
            .and_then(Value::as_str)
            .filter(|s| super::wire::short(s))
            .map(str::to_owned)
    };
    let active = value
        .get("activeTurnId")
        .ok_or(Error::Invalid("native session turn"))?;
    if !active.is_null() && !active.as_str().is_some_and(super::wire::short) {
        return Err(Error::Invalid("native session turn"));
    }
    Ok(SessionSnapshot {
        id: text("sessionId").ok_or(Error::Invalid("native session identity"))?,
        status: text("status").ok_or(Error::Invalid("native session state"))?,
        active_turn: active.as_str().map(str::to_owned),
        turn_count: value
            .get("turnCount")
            .and_then(Value::as_u64)
            .ok_or(Error::Invalid("native session count"))?,
    })
}

impl<W: Write + AsRawFd, R: Read + AsRawFd> Connection<W, R> {
    pub fn initialize(
        &mut self,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<(), Error> {
        let result = self.call("initialize", json!({"clientInfo":{"name":"soda_factory","version":"1"},"capabilities":{"userInputDialogs":false}}), deadline, observe)?;
        if result.get("platformOs").and_then(Value::as_str) != Some("linux")
            || result.get("sessionDurability").and_then(Value::as_str) != Some("durable")
        {
            return Err(Error::Invalid("durable native Linux session required"));
        }
        self.notify("initialized", deadline)
    }

    pub fn start_session(
        &mut self,
        command_id: &str,
        workspace: &str,
        model: &str,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<SessionSnapshot, Error> {
        command(command_id)?;
        if self.session.is_some()
            || !workspace.starts_with('/')
            || workspace.len() > 4096
            || workspace.contains('\0')
            || model.len() > 128
            || model.contains('\0')
        {
            return Err(Error::Invalid("native session input"));
        }
        let mut params =
            json!({"commandId":command_id,"workspaceRoot":workspace,"providerId":"meta"});
        if !model.is_empty() {
            params["modelId"] = model.into();
        }
        let result = self.call("session/start", params, deadline, observe)?;
        let native = result
            .get("session")
            .ok_or(Error::Invalid("native session reply"))?;
        if native.get("providerId").and_then(Value::as_str) != Some("meta")
            || native.get("workspaceRoot").and_then(Value::as_str) != Some(workspace)
            || (!model.is_empty() && native.get("modelId").and_then(Value::as_str) != Some(model))
        {
            return Err(Error::Invalid("native session binding"));
        }
        let state = snapshot(native)?;
        if state.status != "idle" || state.active_turn.is_some() {
            return Err(Error::Invalid("new native session is not idle"));
        }
        self.session = Some(state.id.clone());
        Ok(state)
    }

    pub fn read_session(
        &mut self,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<SessionSnapshot, Error> {
        let id = self
            .session
            .clone()
            .ok_or(Error::Invalid("native session absent"))?;
        let result = self.call(
            "session/read",
            json!({"sessionId":id,"excludeItems":true}),
            deadline,
            observe,
        )?;
        let state = snapshot(
            result
                .get("session")
                .ok_or(Error::Invalid("native session reply"))?,
        )?;
        if state.id != id {
            return Err(Error::Invalid("native session binding"));
        }
        Ok(state)
    }

    pub fn start_turn(
        &mut self,
        command_id: &str,
        prompt: &str,
        effort: &str,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<TurnAdmission, Error> {
        command(command_id)?;
        let id = self
            .session
            .clone()
            .ok_or(Error::Invalid("native session absent"))?;
        if prompt.is_empty()
            || prompt.len() > super::super::run::MAX_REASSESS_PROMPT
            || !crate::factory::valid_task_effort(effort)
        {
            return Err(Error::Invalid("native turn input"));
        }
        let result = self.call("turn/start", json!({"commandId":command_id,"sessionId":id,"reasoningEffort":effort,"input":[{"type":"text","text":prompt}]}), deadline, observe)?;
        if result.get("commandId").and_then(Value::as_str) != Some(command_id)
            || result.get("status").and_then(Value::as_str) != Some("accepted")
            || result.get("disposition").and_then(Value::as_str) != Some("started")
            || result.get("startedNewTurn").and_then(Value::as_bool) != Some(true)
        {
            return Err(Error::Invalid("native turn admission"));
        }
        let turn = result
            .get("turnId")
            .and_then(Value::as_str)
            .filter(|s| super::wire::short(s))
            .ok_or(Error::Invalid("native turn identity"))?;
        Ok(TurnAdmission {
            command: command_id.into(),
            turn: turn.into(),
        })
    }
}
