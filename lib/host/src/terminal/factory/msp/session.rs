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

/// Observe-first decision table for the MCP registration probe. Empty
/// results and completed structured refusals record their summary and let
/// the admitted turn proceed. Transport, deadline and protocol uncertainty
/// (slow lists, timeouts, framing gaps, EOF, unexpected input, malformed
/// replies) records an unavailable marker and stops before any turn: a
/// late reply could otherwise break the strict transport correlation of
/// the next call.
pub fn mcp_probe_outcome(outcome: Result<Value, Error>) -> (Value, Option<Error>) {
    match outcome {
        Ok(result) => (sanitize_mcp_servers(&result), None),
        Err(Error::Refused { code, .. }) => (
            json!({"factory_present": false, "total": 0, "refused": code}),
            None,
        ),
        Err(error) => (
            json!({"factory_present": false, "total": 0, "unavailable": true}),
            Some(error),
        ),
    }
}

/// Sanitized MCP registration summary. Only the factory-present boolean
/// and the total row count are retained; no daemon string, command, URL,
/// environment, credential or free-text message enters stored summaries.
/// A missing or malformed list is unknown, not success.
pub fn sanitize_mcp_servers(list: &Value) -> Value {
    let rows = list.get("servers").and_then(Value::as_array);
    let total = rows.map(|rows| rows.len()).unwrap_or(0);
    let factory_present = rows
        .map(|rows| {
            rows.iter()
                .take(1024)
                .any(|row| row.get("name").and_then(Value::as_str) == Some("factory"))
        })
        .unwrap_or(false);
    json!({"factory_present": factory_present, "total": total})
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
        let result = self.call("initialize", json!({"clientInfo":{"name":"soda_factory","version":"1"},"capabilities":{"userInputDialogs":false,"experimentalApi":true,"requestedCapabilities":["rawLog","sessionMcp"]}}), deadline, observe)?;
        if result.get("platformOs").and_then(Value::as_str) != Some("linux")
            || result.get("sessionDurability").and_then(Value::as_str) != Some("durable")
        {
            return Err(Error::Invalid("durable native Linux session required"));
        }
        // The Factory MCP relay comes from the staged per-run settings, not
        // from session-added config, so the grant list needs no gate: the
        // request stays on the wire for peers that key behavior off it, but
        // a missing, malformed, or denying list never refuses a peer whose
        // staged relay is otherwise usable. A relay the daemon cannot start
        // still fails truthfully at the daemon, which this client
        // propagates unchanged.
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
        self.start_session_config(command_id, workspace, model, None, deadline, observe)
    }

    /// The fixed root-staged Factory relay joins the existing session. No
    /// caller supplied executable, network endpoint or credential is accepted.
    /// The per-run staged settings.json is the single authoritative Factory
    /// MCP registration, so session/start carries no mcpServers: sending the
    /// staged `factory` name again makes the daemon refuse the start as a
    /// session MCP name conflict.
    pub fn start_factory_session(
        &mut self,
        command_id: &str,
        workspace: &str,
        model: &str,
        home: &str,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<SessionSnapshot, Error> {
        if !home.starts_with(&format!("{workspace}/.soda-home/runs/"))
            || !home.ends_with("/home")
            || home.contains('\0')
        {
            return Err(Error::Invalid("native Factory home"));
        }
        // No grant gate: the staged settings carry the required relay, so
        // session/start proceeds regardless of the recorded sessionMcp
        // grant. A relay the daemon cannot start fails at the daemon with
        // its structured reason, which the caller propagates unchanged.
        self.start_session_config(command_id, workspace, model, None, deadline, observe)
    }

    fn start_session_config(
        &mut self,
        command_id: &str,
        workspace: &str,
        model: &str,
        config: Option<Value>,
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
        let mut params = json!({"commandId":command_id,"workspaceRoot":workspace,
            "providerId":"meta","approvalMode":"allowAll"});
        if !model.is_empty() {
            params["modelId"] = model.into();
        }
        if let Some(config) = config {
            params["config"] = config;
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

    /// Read-only inspection of a child already reported by the native parent.
    /// Read-only MCP registration observation for one session. This never
    /// resumes a session or admits a model turn; the caller records the
    /// sanitized summary and proceeds on every outcome.
    pub fn mcp_server_list(
        &mut self,
        id: &str,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<Value, Error> {
        if !super::wire::short(id) {
            return Err(Error::Invalid("native session identity"));
        }
        self.call(
            "mcpServer/list",
            json!({"sessionId": id}),
            deadline,
            observe,
        )
    }

    /// This never resumes a session or admits another model turn.
    pub fn inspect_session(
        &mut self,
        id: &str,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<Value, Error> {
        if !super::wire::short(id) {
            return Err(Error::Invalid("native session identity"));
        }
        let result = self.call(
            "session/read",
            json!({"sessionId":id,"excludeItems":true}),
            deadline,
            observe,
        )?;
        if result["session"]["sessionId"] != id {
            return Err(Error::Invalid("native session binding"));
        }
        Ok(result)
    }

    pub fn log_streams(
        &mut self,
        id: &str,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<Vec<Value>, Error> {
        let result = self.call("log/streams", json!({"sessionId":id}), deadline, observe)?;
        let streams = result["streams"]
            .as_array()
            .filter(|rows| rows.len() <= 256)
            .ok_or(Error::Invalid("native log streams"))?;
        Ok(streams.clone())
    }

    pub fn log_page(
        &mut self,
        id: &str,
        stream: &Value,
        after: Option<&Value>,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<Value, Error> {
        let mut params = json!({"sessionId":id,"stream":stream,"limit":64});
        if let Some(after) = after {
            params["from"] = after.clone();
        }
        self.call("log/page", params, deadline, observe)
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
