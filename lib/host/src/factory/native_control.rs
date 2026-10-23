//! Native Codex control inside the existing admitted unit and broker lease.

use super::native_tasks::NativeTasks;
use super::receipt::{receipt_stop_owned, FactoryReceipt};
use super::{Factory, FactoryBroker, FactoryError, FactoryTerminal, PrivateState};
use crate::project::Executor;
use crate::terminal::factory::msp::{Connection, Error, Notice};
use crate::terminal::factory::native_io::NativeChannels;
use serde_json::{json, Value};
use std::collections::{BTreeMap, VecDeque};
use std::time::{Duration, Instant};

fn invalid() -> FactoryError {
    FactoryError::msg("native protocol outcome unconfirmed")
}

fn protocol_error(stage: &str, error: Error) -> FactoryError {
    let reason = match error {
        Error::Io(error) => format!("io_{:?}", error.kind()),
        Error::Deadline => "deadline".into(),
        Error::Bounds => "bounds".into(),
        Error::End => "ended".into(),
        Error::Gap => "gap".into(),
        Error::InputRequired => "input_required".into(),
        Error::Invalid(reason) => reason.into(),
        Error::Refused { code, .. } => format!("refused_{code}"),
    };
    FactoryError::msg(format!("native {stage}: {reason}"))
}

impl PrivateState {
    /// The reserve owner retains exact protected native configuration before
    /// broker delivery. There is no public/model profile append route.
    pub(crate) fn retain_native_profile(
        &self,
        run: &super::FactoryRun,
        invocation: &str,
        mut profile: Value,
    ) -> Result<(), String> {
        if !crate::terminal::factory::run::valid_factory_run_id(&run.id)
            || !crate::domain::valid_id(&run.project)
            || invocation.is_empty()
            || invocation.len() > 128
            || profile["model"] != run.model
            || profile["backend"] != "v1"
            || !profile["catalog_sha256"]
                .as_str()
                .is_some_and(crate::preparation::valid_digest)
            || !profile["harness_sha256"]
                .as_str()
                .is_some_and(crate::preparation::valid_digest)
        {
            return Err("native profile binding invalid".into());
        }
        profile["invocation"] = invocation.into();
        profile["run"] = run.id.clone().into();
        profile["manifest_digest"] = run.manifest_digest.clone().into();
        self.write(
            &format!("native-{}-{}-profile.json", run.project, run.id),
            &profile,
        )
    }
}

struct NativeTrace<'a> {
    private: &'a PrivateState,
    name: String,
    counts: BTreeMap<String, u64>,
}

impl NativeTrace<'_> {
    fn observe(&mut self, notice: &Notice) -> Result<(), Error> {
        let (method, params) = match notice {
            Notice::Codex { method, params } | Notice::CodexRequest { method, params, .. } => {
                (method.as_str(), params)
            }
            _ => return Ok(()),
        };
        // Protocol names and enum values only: no text, tool arguments,
        // provider error messages, source content or account responses.
        let token = |value: Option<&str>| {
            value
                .filter(|value| {
                    value.len() <= 128
                        && value
                            .bytes()
                            .all(|b| b.is_ascii_alphanumeric() || b"_./:-".contains(&b))
                })
                .unwrap_or("")
                .to_owned()
        };
        let key = format!(
            "{}|{}|{}|{}|{}",
            token(Some(method)),
            token(params["item"]["type"].as_str()),
            token(params["item"]["kind"].as_str()),
            token(params["turn"]["status"].as_str()),
            token(params["status"]["type"].as_str())
        );
        if !self.counts.contains_key(&key) && self.counts.len() >= 256 {
            return Err(Error::Bounds);
        }
        let count = self.counts.entry(key).or_default();
        *count = count.saturating_add(1);
        if count.is_power_of_two() {
            self.private
                .write(&self.name, &self.counts)
                .map_err(|_| Error::Invalid("native metadata retention"))?;
        }
        Ok(())
    }
}

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    pub(in crate::factory) fn control_native(
        &self,
        receipt: &FactoryReceipt,
        prompt: &[u8],
        channels: &mut NativeChannels,
        deadline: Instant,
    ) -> Result<(i64, String), FactoryError> {
        let (input, output) = channels.take_pipes().map_err(FactoryError::msg)?;
        let mut connection = Connection::codex(input, output);
        let result = self.drive_codex(receipt, prompt, &mut connection, deadline);
        connection.close_input();
        result
    }

    fn drive_codex(
        &self,
        receipt: &FactoryReceipt,
        prompt: &[u8],
        connection: &mut Connection<std::fs::File, std::fs::File>,
        deadline: Instant,
    ) -> Result<(i64, String), FactoryError> {
        let run = &receipt.run;
        let binding = receipt.binding.as_ref().ok_or_else(invalid)?;
        let private = PrivateState::open(&self.state_dir).map_err(FactoryError::msg)?;
        let mut trace = NativeTrace {
            private: &private,
            name: format!("native-{}-{}-protocol.json", run.project, run.id),
            counts: BTreeMap::new(),
        };
        let mut observer = |notice: Notice| trace.observe(&notice);
        let startup = deadline.min(Instant::now() + Duration::from_secs(30));
        connection
            .initialize_codex(startup, &mut observer)
            .map_err(|error| protocol_error("initialize", error))?;
        let workspace = format!("/home/{}/checkouts/{}", run.role, run.preparation);
        let manifest = if super::execution_manifest_required(&run.execution_phase) {
            Some(
                super::ExecutionManifest::decode(&self.load_execution_manifest(run)?)
                    .map_err(FactoryError::msg)?,
            )
        } else {
            None
        };
        let correct = run.execution_phase == super::EXECUTION_CORRECT;
        let instructions="Follow the admitted Factory phase and immutable manifest. Never write Git metadata, start independent CLI agents, acquire credentials or spawn nested children. Native task output is a proposal; only trusted executor checkpoints authorize completion. For implementation, select ready admitted tasks yourself using factory_start_task, then spawn exactly one native child with the returned prompt verbatim, model and reasoning effort. Never reuse, follow up, resume or replace a task child. Wait for each native result, then read Factory MCP checkpoint context and submit_task_output with the host-observed exact output digest and child reference; wait for its committed checkpoint before dependent or same-lane work. The parent must not edit task source. For correction, do the singleton task yourself, call factory_release_output and never write source afterward; submit that exact output digest and empty native child reference and await the committed checkpoint. Remain alive until all task checkpoints and the final candidate checks are committed.";
        let tools = if manifest.is_some() {
            NativeTasks::tools(correct)
        } else {
            Vec::new()
        };
        let thread = connection
            .start_codex_thread_tools(
                &workspace,
                &run.model,
                instructions,
                &tools,
                startup,
                &mut observer,
            )
            .map_err(|error| protocol_error("thread_start", error))?;
        let effort = manifest
            .as_ref()
            .map(|manifest| manifest.effort.clone())
            .unwrap_or_else(|| "low".into());
        let mut tasks = manifest
            .map(|manifest| NativeTasks::new(receipt, manifest, &thread.id, &private))
            .transpose()?;
        if correct {
            tasks
                .as_ref()
                .ok_or_else(invalid)?
                .begin_correction(self, receipt, deadline)?;
        }
        let text = std::str::from_utf8(prompt).map_err(|_| invalid())?;
        let profile: Option<Value> = private
            .read(&format!("native-{}-{}-profile.json", run.project, run.id))
            .map_err(FactoryError::msg)?;
        if let Some(profile) = &profile {
            if profile["invocation"] != binding.invocation_id
                || profile["run"] != run.id
                || profile["manifest_digest"] != run.manifest_digest
                || profile["model"] != run.model
                || profile["backend"] != "v1"
            {
                return Err(FactoryError::Stale);
            }
        }
        private.write(&format!("native-{}-{}.json",run.project,run.id),&json!({"thread":thread.id,"invocation":binding.invocation_id,"manifest_digest":run.manifest_digest,"native_profile":profile})).map_err(FactoryError::msg)?;
        let mut pending = VecDeque::new();
        let mut observer = |notice| {
            pending.push_back(notice);
            Ok(())
        };
        let turn = connection
            .start_codex_turn(text, &effort, startup, &mut observer)
            .map_err(|error| protocol_error("turn_start", error))?;
        let mut output = String::new();
        let mut evidence_count = 0usize;
        loop {
            self.native_live(receipt, deadline)?;
            let notice = match pending.pop_front() {
                Some(notice) => Ok(notice),
                None => connection.receive(deadline.min(Instant::now() + Duration::from_secs(1))),
            };
            match notice {
                Err(Error::Deadline) => continue,
                Err(error) => return Err(protocol_error("observe", error)),
                Ok(Notice::End) => return Err(protocol_error("observe", Error::End)),
                Ok(Notice::Codex { method, params }) => {
                    trace
                        .observe(&Notice::Codex {
                            method: method.clone(),
                            params: params.clone(),
                        })
                        .map_err(|error| protocol_error("metadata", error))?;
                    self.native_evidence(receipt, &private, &method, &params, &mut evidence_count)?;
                    if let Some(tasks) = tasks.as_mut() {
                        tasks.item(self, receipt, &private, &method, &params, deadline)?;
                    } else if matches!(
                        params["item"]["type"].as_str(),
                        Some("collabAgentToolCall" | "subAgentActivity")
                    ) {
                        return Err(FactoryError::msg(
                            "singleton phase cannot admit native children",
                        ));
                    }
                    if params.get("threadId").and_then(Value::as_str) != Some(&thread.id) {
                        continue;
                    }
                    if method == "item/completed" {
                        let item = &params["item"];
                        if item["type"] == "agentMessage" {
                            if let Some(text) = item["text"].as_str() {
                                if text.len() > super::MAX_FACTORY_OUTPUT {
                                    return Err(FactoryError::msg(
                                        "native output exceeds its bound",
                                    ));
                                }
                                output = text.into();
                            }
                        }
                    }
                    if method == "turn/completed" && params["turn"]["id"].as_str() == Some(&turn.id)
                    {
                        let completed = params["turn"]["status"] == "completed";
                        if completed {
                            if let Some(tasks) = tasks.as_ref() {
                                let request = tasks.final_request(self, receipt, deadline)?;
                                loop {
                                    self.native_live(receipt, deadline)?;
                                    let result = self.task_checkpoint(&request, true, deadline)?;
                                    if result.status == "committed" {
                                        break;
                                    }
                                    if !matches!(result.status.as_str(), "pending" | "busy") {
                                        return Err(FactoryError::msg(
                                            "native final candidate checks failed",
                                        ));
                                    }
                                    std::thread::sleep(Duration::from_millis(100));
                                }
                            }
                        }
                        self.terminal.write_native_output(
                            &receipt
                                .lease
                                .as_ref()
                                .ok_or_else(invalid)?
                                .with_binding(binding),
                            output.as_bytes(),
                            deadline,
                        )?;
                        return Ok((if completed { 0 } else { 1 }, output));
                    }
                }
                Ok(Notice::CodexRequest { id, method, params }) => {
                    trace
                        .observe(&Notice::CodexRequest {
                            id: id.clone(),
                            method: method.clone(),
                            params: params.clone(),
                        })
                        .map_err(|error| protocol_error("metadata", error))?;
                    if method != "item/tool/call" {
                        return Err(invalid());
                    }
                    let result = tasks
                        .as_mut()
                        .ok_or_else(invalid)?
                        .tool(self, receipt, &private, &id, &params, &turn.id, deadline)?;
                    connection
                        .reply_codex_tool(&id, &result, true, deadline)
                        .map_err(|_| invalid())?;
                }
                Ok(_) => return Err(invalid()),
            }
        }
    }

    fn native_live(&self, receipt: &FactoryReceipt, deadline: Instant) -> Result<(), FactoryError> {
        if Instant::now() >= deadline {
            return Err(FactoryError::DeadlineExceeded);
        }
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        let (current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists || receipt_stop_owned(&current.phase) {
            return Err(FactoryError::Stale);
        }
        Ok(())
    }

    fn native_evidence(
        &self,
        receipt: &FactoryReceipt,
        private: &PrivateState,
        method: &str,
        params: &Value,
        count: &mut usize,
    ) -> Result<(), FactoryError> {
        if !matches!(method, "item/started" | "item/completed") {
            return Ok(());
        }
        let item = &params["item"];
        if !matches!(
            item["type"].as_str(),
            Some("collabAgentToolCall" | "subAgentActivity")
        ) {
            return Ok(());
        }
        if *count >= 256 {
            return Err(FactoryError::msg(
                "native lifecycle evidence exceeds its bound",
            ));
        }
        let bytes = serde_json::to_vec(params).map_err(|_| invalid())?;
        if bytes.len() > 64 * 1024 {
            return Err(FactoryError::msg("native lifecycle item exceeds its bound"));
        }
        let key = crate::sha256::hex_lower(&crate::sha256::digest(
            format!("{method}\0{}", item["id"].as_str().ok_or_else(invalid)?).as_bytes(),
        ));
        let name = format!(
            "native-{}-{}-{}.json",
            receipt.run.project, receipt.run.id, key
        );
        let record = json!({"method":method,"invocation":receipt.binding.as_ref().ok_or_else(invalid)?.invocation_id,"params":params});
        if let Some(old) = private.read::<Value>(&name).map_err(FactoryError::msg)? {
            if old != record {
                return Err(FactoryError::Stale);
            }
        } else {
            private.write(&name, &record).map_err(FactoryError::msg)?;
            *count += 1;
        }
        // These are protected native observations only. No task event, child
        // assignment or successful checkpoint is synthesized from this content.
        Ok(())
    }
}
