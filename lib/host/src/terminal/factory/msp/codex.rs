//! Native Codex app-server session methods. Credentials remain in the existing
//! broker-staged private home; this protocol owner never reads an auth file.

use serde_json::{json, Value};
use std::io::{Read, Write};
use std::os::fd::AsRawFd;
use std::time::Instant;

use super::{Connection, Error, Notice};

#[derive(Clone, Debug)]
pub struct CodexThread {
    pub id: String,
    pub model: String,
}

#[derive(Clone, Debug)]
pub struct CodexTurn {
    pub id: String,
    pub status: String,
}

fn text(value: &Value, field: &str) -> Result<String, Error> {
    value
        .get(field)
        .and_then(Value::as_str)
        .filter(|s| super::wire::short(s))
        .map(str::to_owned)
        .ok_or(Error::Invalid("native Codex identity"))
}

impl<W: Write + AsRawFd, R: Read + AsRawFd> Connection<W, R> {
    pub fn initialize_codex(
        &mut self,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<(), Error> {
        if !self.codex {
            return Err(Error::Invalid("Codex transport required"));
        }
        self.call("initialize", json!({"clientInfo":{"name":"soda_factory","version":"1"},"capabilities":{"experimentalApi":true}}), deadline, observe)?;
        self.notify("initialized", deadline)
    }

    pub fn start_codex_thread(
        &mut self,
        workspace: &str,
        model: &str,
        instructions: &str,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<CodexThread, Error> {
        self.start_codex_thread_tools(workspace, model, instructions, &[], deadline, observe)
    }

    pub fn start_codex_thread_tools(
        &mut self,
        workspace: &str,
        model: &str,
        instructions: &str,
        tools: &[Value],
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<CodexThread, Error> {
        if !self.codex
            || self.session.is_some()
            || !workspace.starts_with('/')
            || workspace.len() > 4096
            || workspace.contains('\0')
            || model.len() > 128
            || instructions.len() > super::super::mcp_relay::MAX_INSTRUCTION
            || tools.len() > 4
        {
            return Err(Error::Invalid("native Codex thread input"));
        }
        let mut params = json!({"cwd":workspace,"approvalPolicy":"never","sandbox":"danger-full-access","ephemeral":false,"developerInstructions":instructions});
        if !model.is_empty() {
            params["model"] = model.into();
        }
        if !tools.is_empty() {
            params["dynamicTools"] = tools.into();
        }
        let result = self.call("thread/start", params, deadline, observe)?;
        let thread = result
            .get("thread")
            .ok_or(Error::Invalid("native Codex thread reply"))?;
        if thread.get("cwd").and_then(Value::as_str) != Some(workspace) {
            return Err(Error::Invalid("native Codex workspace"));
        }
        let id = text(thread, "id")?;
        let resolved = text(&result, "model")?;
        if !model.is_empty() && resolved != model {
            return Err(Error::Invalid("native Codex model"));
        }
        self.session = Some(id.clone());
        Ok(CodexThread {
            id,
            model: resolved,
        })
    }

    pub fn read_codex_thread(
        &mut self,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<String, Error> {
        let id = self
            .session
            .clone()
            .ok_or(Error::Invalid("native Codex thread absent"))?;
        let result = self.call(
            "thread/read",
            json!({"threadId":id,"includeTurns":false}),
            deadline,
            observe,
        )?;
        let returned = text(
            result
                .get("thread")
                .ok_or(Error::Invalid("native Codex thread reply"))?,
            "id",
        )?;
        if returned != id {
            return Err(Error::Invalid("native Codex thread binding"));
        }
        Ok(returned)
    }

    pub fn start_codex_turn(
        &mut self,
        prompt: &str,
        effort: &str,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<CodexTurn, Error> {
        let thread = self
            .session
            .clone()
            .ok_or(Error::Invalid("native Codex thread absent"))?;
        if prompt.is_empty()
            || prompt.len() > super::super::run::MAX_REASSESS_PROMPT
            || !crate::factory::valid_task_effort(effort)
        {
            return Err(Error::Invalid("native Codex turn input"));
        }
        let result = self.call(
            "turn/start",
            json!({"threadId":thread,"effort":effort,"input":[{"type":"text","text":prompt}]}),
            deadline,
            observe,
        )?;
        let turn = result
            .get("turn")
            .ok_or(Error::Invalid("native Codex turn reply"))?;
        Ok(CodexTurn {
            id: text(turn, "id")?,
            status: text(turn, "status")?,
        })
    }
}
