//! Native control inside the existing admitted unit and broker lease.

use super::native_actions::NativeTaskOwner;
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
        Error::Refused { code, reason } => match reason {
            Some(reason) => format!("refused_{code}_{reason}"),
            None => format!("refused_{code}"),
        },
    };
    FactoryError::msg(format!("native {stage}: {reason}"))
}

struct NativeTrace<'a> {
    private: &'a PrivateState,
    name: String,
    counts: BTreeMap<String, u64>,
}

impl NativeTrace<'_> {
    fn observe(&mut self, notice: &Notice) -> Result<(), Error> {
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
        let fields = match notice {
            Notice::Item { item, .. } => (
                "item",
                Some(item.kind.as_str()),
                None,
                Some(item.status.as_str()),
                None,
            ),
            Notice::TurnStarted { .. } => ("turn/started", None, None, None, None),
            Notice::TurnCompleted { terminal, .. } => {
                ("turn/completed", None, None, Some(terminal.as_str()), None)
            }
            Notice::Other { method, .. } => (method.as_str(), None, None, None, None),
            Notice::End => ("end", None, None, None, None),
        };
        let key = format!(
            "{}|{}|{}|{}|{}",
            token(Some(fields.0)),
            token(fields.1),
            token(fields.2),
            token(fields.3),
            token(fields.4)
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
        if receipt.run.harness != super::FACTORY_HARNESS_MUSE {
            return Err(FactoryError::msg("unsupported factory harness"));
        }
        let mut connection = Connection::new(input, output);
        let result = self.drive_muse(receipt, prompt, &mut connection, deadline);
        connection.close_input();
        result
    }

    /// Observe-first MCP registration probe. The probe runs under its own
    /// short deadline so a slow list cannot exhaust the turn budget. The
    /// sanitized summary is stored by the caller on every outcome; only
    /// transport, deadline and protocol uncertainty stops the run before
    /// any turn.
    fn observe_mcp_registration(
        connection: &mut Connection<std::fs::File, std::fs::File>,
        session_id: &str,
        deadline: Instant,
        observer: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> (Value, Result<(), FactoryError>) {
        use crate::terminal::factory::msp::mcp_probe_outcome;
        let probe = deadline.min(Instant::now() + Duration::from_secs(15));
        let (summary, blocked) =
            mcp_probe_outcome(connection.mcp_server_list(session_id, probe, observer));
        let gate = match blocked {
            None => Ok(()),
            Some(error) => Err(protocol_error("mcp_servers", error)),
        };
        (summary, gate)
    }

    fn drive_muse(
        &self,
        receipt: &FactoryReceipt,
        prompt: &[u8],
        connection: &mut Connection<std::fs::File, std::fs::File>,
        deadline: Instant,
    ) -> Result<(i64, String), FactoryError> {
        let run = &receipt.run;
        let binding = receipt.binding.as_ref().ok_or_else(invalid)?;
        let private = PrivateState::open(&self.state_dir).map_err(FactoryError::msg)?;
        let name = format!("native-{}-{}.json", run.project, run.id);
        if private
            .read::<Value>(&name)
            .map_err(FactoryError::msg)?
            .is_some()
        {
            return Err(FactoryError::Stale);
        }
        let session_command = crate::terminal::factory::msp::new_command_id()
            .map_err(|error| protocol_error("session_identity", error))?;
        let turn_command = crate::terminal::factory::msp::new_command_id()
            .map_err(|error| protocol_error("turn_identity", error))?;
        // Retain identities before any native command. An uncertain native
        // outcome is never replaced by a fresh parent or a second child.
        let mut admission = json!({"session_command":session_command,"turn_command":turn_command,
            "invocation":binding.invocation_id,"manifest_digest":run.manifest_digest,
            "model":run.model,"effort":run.effort,"contributor":run.contributor,
            "harness_version":run.harness_vers});
        private
            .write(&name, &admission)
            .map_err(FactoryError::msg)?;
        let mut trace = NativeTrace {
            private: &private,
            name: format!("native-{}-{}-protocol.json", run.project, run.id),
            counts: BTreeMap::new(),
        };
        let mut pending = VecDeque::new();
        let startup = deadline.min(Instant::now() + Duration::from_secs(30));
        let mut observer = |notice| {
            pending.push_back(notice);
            Ok(())
        };
        connection
            .initialize(startup, &mut observer)
            .map_err(|error| protocol_error("initialize", error))?;
        let workspace = format!("/home/{}/checkouts/{}", run.role, run.preparation);
        let home = format!("{workspace}/.soda-home/runs/{}/home", run.id);
        let session = connection
            .start_factory_session(
                &session_command,
                &workspace,
                &run.model,
                &home,
                startup,
                &mut observer,
            )
            .map_err(|error| protocol_error("session_start", error))?;
        if session.turn_count != 0 {
            return Err(invalid());
        }
        admission["session"] = session.id.clone().into();
        // Persist the created native session identity before any optional
        // read: a crash or stop during the probe must not lose it.
        private
            .write(&name, &admission)
            .map_err(FactoryError::msg)?;
        // Observe-first MCP registration: record only the factory-present
        // boolean, the total row count and numeric error codes. An empty
        // list is unknown, not success. Transport, deadline and protocol
        // uncertainty persists its marker, then fails truthfully before
        // any turn so a late probe reply cannot corrupt the next call.
        let (servers, probe) =
            Self::observe_mcp_registration(connection, &session.id, startup, &mut observer);
        admission["mcp_servers"] = servers;
        private
            .write(&name, &admission)
            .map_err(FactoryError::msg)?;
        probe?;
        let manifest = if super::execution_manifest_required(&run.execution_phase) {
            Some(
                super::ExecutionManifest::decode(&self.load_execution_manifest(run)?)
                    .map_err(FactoryError::msg)?,
            )
        } else {
            None
        };
        let effort = manifest
            .as_ref()
            .map(|manifest| manifest.effort.as_str())
            .unwrap_or(if run.effort.is_empty() {
                "low"
            } else {
                &run.effort
            })
            .to_owned();
        let mut tasks = manifest
            .map(|manifest| NativeTasks::new(receipt, manifest, &session.id, &private))
            .transpose()?;
        let owner = if tasks.is_some() {
            Some(NativeTaskOwner::register(&self.state_dir, receipt)?)
        } else {
            None
        };
        if run.execution_phase == super::EXECUTION_CORRECT {
            tasks
                .as_ref()
                .ok_or_else(invalid)?
                .begin_correction(self, receipt, deadline)?;
        }
        let text = std::str::from_utf8(prompt).map_err(|_| invalid())?;
        let turn = connection
            .start_turn(&turn_command, text, &effort, startup, &mut observer)
            .map_err(|error| protocol_error("turn_start", error))?;
        admission["turn"] = turn.turn.clone().into();
        private
            .write(&name, &admission)
            .map_err(FactoryError::msg)?;
        let mut cursors = BTreeMap::new();
        let mut next_observation = Instant::now();
        let mut output = String::new();
        loop {
            self.native_live(receipt, deadline)?;
            if let (Some(owner), Some(tasks)) = (owner.as_ref(), tasks.as_mut()) {
                owner.poll(self, receipt, &private, tasks, &session.id, &turn.turn);
                if Instant::now() >= next_observation {
                    self.observe_muse_tasks(
                        receipt,
                        &private,
                        tasks,
                        connection,
                        &session.id,
                        &turn.turn,
                        &mut cursors,
                        &mut pending,
                        deadline,
                    )?;
                    next_observation = Instant::now() + Duration::from_secs(1);
                }
            }
            let notice = match pending.pop_front() {
                Some(notice) => notice,
                None => match connection
                    .receive(deadline.min(Instant::now() + Duration::from_millis(250)))
                {
                    Err(Error::Deadline) => continue,
                    Err(error) => return Err(protocol_error("observe", error)),
                    Ok(notice) => notice,
                },
            };
            trace
                .observe(&notice)
                .map_err(|error| protocol_error("metadata", error))?;
            match notice {
                Notice::Item {
                    session: observed,
                    item,
                    ..
                } => {
                    if observed != session.id {
                        continue;
                    }
                    if item.kind == "subagent" {
                        if tasks.is_none() || run.execution_phase == super::EXECUTION_CORRECT {
                            return Err(FactoryError::msg(
                                "singleton phase cannot admit native children",
                            ));
                        }
                        next_observation = Instant::now();
                    }
                    if item.kind == "agentMessage"
                        && item.status == "completed"
                        && item.turn_id.as_deref() == Some(&turn.turn)
                    {
                        if let Some(text) = item.text {
                            if text.len() > super::MAX_FACTORY_OUTPUT {
                                return Err(FactoryError::msg("native output exceeds its bound"));
                            }
                            output = text;
                        }
                    }
                }
                Notice::TurnCompleted {
                    session: observed,
                    turn: observed_turn,
                    terminal,
                    failure,
                    ..
                } if observed == session.id && observed_turn == turn.turn => {
                    let completed = terminal == "completed" && failure.is_none();
                    if completed {
                        if let Some(tasks) = tasks.as_mut() {
                            self.observe_muse_tasks(
                                receipt,
                                &private,
                                tasks,
                                connection,
                                &session.id,
                                &turn.turn,
                                &mut cursors,
                                &mut pending,
                                deadline,
                            )?;
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
                Notice::End => return Err(protocol_error("observe", Error::End)),
                _ => {}
            }
        }
    }

    fn observe_muse_tasks(
        &self,
        receipt: &FactoryReceipt,
        private: &PrivateState,
        tasks: &mut NativeTasks,
        connection: &mut Connection<std::fs::File, std::fs::File>,
        parent: &str,
        turn: &str,
        cursors: &mut BTreeMap<String, Value>,
        pending: &mut VecDeque<Notice>,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        let mut sessions = vec![parent.to_owned()];
        let mut index = 0;
        while index < sessions.len() {
            self.native_live(receipt, deadline)?;
            let session = sessions[index].clone();
            let mut observer = |notice| {
                pending.push_back(notice);
                Ok(())
            };
            // The session stream durably mirrors native control and run facts;
            // reading it once avoids counting the separate run mirror twice.
            let stream = json!({"kind":"session","id":session});
            for _ in 0..256 {
                let page = connection
                    .log_page(
                        &session,
                        &stream,
                        cursors.get(&session),
                        deadline.min(Instant::now() + Duration::from_secs(10)),
                        &mut observer,
                    )
                    .map_err(|error| protocol_error("native_log", error))?;
                let records = page["records"]
                    .as_array()
                    .filter(|records| records.len() <= 64)
                    .ok_or_else(invalid)?;
                if records.is_empty() {
                    break;
                }
                for record in records {
                    if record.get("retained_marker").is_some() {
                        return Err(FactoryError::msg("native child evidence was not retained"));
                    }
                    tasks.muse_record(self, receipt, private, &session, turn, record, deadline)?;
                }
                let cursor = &page["nextCursor"];
                let after = &cursor["after"];
                if cursor["stream"] != stream
                    || !after["id"].is_string()
                    || after["sequence"].as_u64().is_none()
                    || cursors
                        .get(&session)
                        .is_some_and(|old| old["sequence"].as_u64() >= after["sequence"].as_u64())
                {
                    return Err(invalid());
                }
                cursors.insert(session.clone(), after.clone());
                if records.len() < 64 {
                    break;
                }
            }
            if session != parent {
                let snapshot = connection
                    .inspect_session(
                        &session,
                        deadline.min(Instant::now() + Duration::from_secs(10)),
                        &mut observer,
                    )
                    .map_err(|error| protocol_error("child_session", error))?;
                tasks.muse_session(self, receipt, private, &snapshot["session"], deadline)?;
            } else {
                sessions.extend(tasks.muse_children());
            }
            index += 1;
        }
        Ok(())
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
}
