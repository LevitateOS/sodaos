//! Correlate one admitted parent selection with its actual native child.
//! Model arguments nominate a task; only immutable native spawn/profile/result
//! observations can bind that task to a child and release its owned tree.

use std::collections::BTreeMap;
use std::time::Instant;

use serde::{Deserialize, Serialize};
use serde_json::{json, Value};

use super::receipt::FactoryReceipt;
use super::{
    ExecutionManifest, Factory, FactoryBroker, FactoryError, FactoryTaskCheckpoint,
    FactoryTerminal, PrivateState, EXECUTION_CORRECT, TASK_EVENT_CHILD_STARTED,
    TASK_EVENT_CHILD_START_INTENT, TASK_EVENT_PARENT_STARTED, TASK_EVENT_TASK_COMMITTED,
};
use crate::project::Executor;

pub(super) const START_TOOL: &str = "factory_start_task";
pub(super) const RELEASE_TOOL: &str = "factory_release_output";

#[derive(Clone, Deserialize)]
struct Plan {
    tasks: Vec<Task>,
    lanes: Vec<Lane>,
}

#[derive(Clone, Deserialize, Serialize)]
struct Task {
    id: String,
    objective: String,
    outputs: Vec<String>,
    #[serde(default, deserialize_with = "nullable_list")]
    acceptance_refs: Vec<String>,
    #[serde(default, deserialize_with = "nullable_list")]
    depends_on: Vec<String>,
    lane: String,
    owned_paths: Vec<String>,
    #[serde(default, deserialize_with = "nullable_list")]
    interface_writes: Vec<String>,
    #[serde(default, deserialize_with = "nullable_list")]
    check_ids: Vec<String>,
}

fn nullable_list<'de, D: serde::Deserializer<'de>>(de: D) -> Result<Vec<String>, D::Error> {
    Ok(Option::<Vec<String>>::deserialize(de)?.unwrap_or_default())
}

fn tasks_conflict(a: &Task, b: &Task) -> bool {
    let paths_overlap = |a: &str, b: &str| {
        a == b
            || a.strip_prefix(b)
                .is_some_and(|tail| b.ends_with('/') || tail.starts_with('/'))
            || b.strip_prefix(a)
                .is_some_and(|tail| a.ends_with('/') || tail.starts_with('/'))
    };
    a.lane == b.lane
        || a.owned_paths
            .iter()
            .any(|a| b.owned_paths.iter().any(|b| paths_overlap(a, b)))
        || a.interface_writes
            .iter()
            .any(|key| b.interface_writes.contains(key))
}

#[derive(Clone, Deserialize)]
struct Lane {
    id: String,
    task_ids: Vec<String>,
}

#[derive(Clone, Default, Deserialize, Serialize)]
struct Selection {
    task: String,
    call: String,
    request: Value,
    nonce: String,
    prompt: String,
    spawn: String,
    child: String,
    started: bool,
    activity_complete: bool,
    result_complete: bool,
    output: String,
}

#[derive(Default, Deserialize, Serialize)]
struct Ledger {
    thread: String,
    invocation: String,
    manifest: String,
    selections: Vec<Selection>,
    profiles: BTreeMap<String, (String, String)>,
    #[serde(default)]
    activity_completed: BTreeMap<String, bool>,
}

pub(super) struct NativeTasks {
    manifest: ExecutionManifest,
    plan: Plan,
    ledger: Ledger,
    name: String,
}

fn invalid() -> FactoryError {
    FactoryError::msg("native task correlation unconfirmed")
}

impl NativeTasks {
    pub(super) fn new(
        receipt: &FactoryReceipt,
        manifest: ExecutionManifest,
        thread: &str,
        private: &PrivateState,
    ) -> Result<Self, FactoryError> {
        let plan: Plan = serde_json::from_slice(&manifest.plan).map_err(|_| invalid())?;
        if plan.tasks.is_empty() || plan.tasks.len() > 64 || plan.lanes.len() > 4 {
            return Err(invalid());
        }
        let name = format!(
            "native-tasks-{}-{}.json",
            receipt.run.project, receipt.run.id
        );
        // An uncertain old native parent is never replaced with another.
        if private
            .read::<Ledger>(&name)
            .map_err(FactoryError::msg)?
            .is_some()
        {
            return Err(FactoryError::Stale);
        }
        let ledger = Ledger {
            thread: thread.into(),
            invocation: receipt
                .binding
                .as_ref()
                .ok_or_else(invalid)?
                .invocation_id
                .clone(),
            manifest: receipt.run.manifest_digest.clone(),
            ..Ledger::default()
        };
        private.write(&name, &ledger).map_err(FactoryError::msg)?;
        Ok(Self {
            manifest,
            plan,
            ledger,
            name,
        })
    }

    pub(super) fn tools(correct: bool) -> Vec<Value> {
        let spec = |name: &str, description: &str| {
            json!({
                "type":"function", "name":name, "description":description,
                "inputSchema":{"type":"object","properties":{"task_id":{"type":"string"}},"required":["task_id"],"additionalProperties":false}
            })
        };
        if correct {
            vec![spec(RELEASE_TOOL, "Release the singleton task's filesystem output while waiting for its trusted checkpoint. Do not write source again after release. The returned digest is a proposal binding, never a check result.")]
        } else {
            vec![spec(START_TOOL,"Select a ready admitted task. Before spawning, call this exactly once for that task. Spawn one native child with the returned prompt verbatim and the returned model and effort. Never reassign or resume a task child.")]
        }
    }

    fn persist(&self, private: &PrivateState) -> Result<(), FactoryError> {
        private
            .write(&self.name, &self.ledger)
            .map_err(FactoryError::msg)
    }

    fn task(&self, id: &str) -> Result<&Task, FactoryError> {
        self.plan
            .tasks
            .iter()
            .find(|task| task.id == id)
            .ok_or_else(invalid)
    }

    fn request(&self, receipt: &FactoryReceipt, task: &str) -> FactoryTaskCheckpoint {
        FactoryTaskCheckpoint {
            project: receipt.run.project.clone(),
            id: receipt.run.id.clone(),
            container: receipt.binding.as_ref().unwrap().project.clone(),
            invocation: self.ledger.invocation.clone(),
            manifest_digest: self.ledger.manifest.clone(),
            task_key: task.into(),
            expected_head: receipt.run.source_commit.clone(),
            ..FactoryTaskCheckpoint::default()
        }
    }

    // Call while holding the run lock, immediately before task admission.
    fn ensure_live<E: Executor, T: FactoryTerminal, B: FactoryBroker>(
        &self,
        factory: &Factory<E, T, B>,
        receipt: &FactoryReceipt,
        private: &PrivateState,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        let (current, exists) = factory.load_receipt(&receipt.run.project, &receipt.run.id)?;
        let binding = current.binding.as_ref().ok_or(FactoryError::Stale)?;
        if !exists
            || current.phase != super::FACTORY_RUNNING
            || binding.invocation_id != self.ledger.invocation
            || private
                .read::<bool>(&format!(
                    "retiring-{}-{}.json",
                    receipt.run.project, receipt.run.id
                ))
                .map_err(FactoryError::msg)?
                .unwrap_or(false)
            || !factory.terminal.live(binding, deadline)
        {
            return Err(FactoryError::Stale);
        }
        Ok(())
    }

    pub(super) fn begin_correction<E: Executor, T: FactoryTerminal, B: FactoryBroker>(
        &self,
        factory: &Factory<E, T, B>,
        receipt: &FactoryReceipt,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        if receipt.run.execution_phase != EXECUTION_CORRECT
            || self.plan.tasks.len() != 1
            || self.manifest.max_children != 0
        {
            return Err(invalid());
        }
        let _lock = factory.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        let private = PrivateState::open(&factory.state_dir).map_err(FactoryError::msg)?;
        self.ensure_live(factory, receipt, &private, deadline)?;
        factory.append_checkpoint_event(
            &self.request(receipt, &self.plan.tasks[0].id),
            TASK_EVENT_PARENT_STARTED,
            "",
            "",
        )
    }

    pub(super) fn final_request<E: Executor, T: FactoryTerminal, B: FactoryBroker>(
        &self,
        factory: &Factory<E, T, B>,
        receipt: &FactoryReceipt,
        deadline: Instant,
    ) -> Result<FactoryTaskCheckpoint, FactoryError> {
        let _lock = factory.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        let journal = factory.load_journal(&receipt.run.project, &receipt.run.id)?;
        if self.plan.tasks.iter().any(|task| {
            !journal.events().iter().any(|event| {
                event.task_key == task.id && event.event_type == TASK_EVENT_TASK_COMMITTED
            })
        }) || self.ledger.selections.len() != self.plan.tasks.len()
            || self
                .ledger
                .selections
                .iter()
                .any(|selection| !selection.started || selection.output.is_empty())
            || self.ledger.profiles.keys().any(|child| {
                !self
                    .ledger
                    .selections
                    .iter()
                    .any(|selection| &selection.child == child)
            })
        {
            return Err(FactoryError::msg(
                "native task checkpoint release remains unconfirmed",
            ));
        }
        let mut request = self.request(receipt, "");
        request.expected_head = factory.integration_head(
            &receipt.run,
            receipt.binding.as_ref().ok_or_else(invalid)?,
            &receipt.run.source_commit,
        )?;
        Ok(request)
    }

    pub(super) fn tool<E: Executor, T: FactoryTerminal, B: FactoryBroker>(
        &mut self,
        factory: &Factory<E, T, B>,
        receipt: &FactoryReceipt,
        private: &PrivateState,
        request: &Value,
        params: &Value,
        turn: &str,
        deadline: Instant,
    ) -> Result<String, FactoryError> {
        if params["threadId"] != self.ledger.thread
            || params["turnId"] != turn
            || params
                .get("namespace")
                .is_some_and(|value| !value.is_null())
        {
            return Err(invalid());
        }
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Arguments {
            task_id: String,
        }
        let args: Arguments =
            serde_json::from_value(params["arguments"].clone()).map_err(|_| invalid())?;
        {
            let _lock = factory.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
            self.ensure_live(factory, receipt, private, deadline)?;
        }
        let call = params["callId"]
            .as_str()
            .filter(|s| !s.is_empty() && s.len() <= 128)
            .ok_or_else(invalid)?;
        let task = self.task(&args.task_id)?.clone();
        if receipt.run.execution_phase == EXECUTION_CORRECT {
            if params["tool"] != RELEASE_TOOL || task.id != self.plan.tasks[0].id {
                return Err(invalid());
            }
            if let Some(old) = self.ledger.selections.first() {
                if old.call != call || old.request != *request {
                    return Err(FactoryError::Stale);
                }
            } else {
                self.ledger.selections.push(Selection {
                    task: task.id.clone(),
                    call: call.into(),
                    request: request.clone(),
                    started: true,
                    ..Selection::default()
                });
                self.persist(private)?;
            }
            let digest = factory.observe_task_output(
                &receipt.run.project,
                &receipt.run.id,
                &task.id,
                "",
                deadline,
            )?;
            self.ledger.selections[0].output = digest.clone();
            self.persist(private)?;
            return Ok(json!({"task_id":task.id,"output_digest":digest,"native_child_ref":"","released":true}).to_string());
        }
        if params["tool"] != START_TOOL {
            return Err(invalid());
        }
        if let Some(old) = self
            .ledger
            .selections
            .iter()
            .find(|selection| selection.task == task.id || selection.call == call)
        {
            if old.task != task.id || old.call != call || old.request != *request {
                return Err(FactoryError::Stale);
            }
            return Ok(json!({"admitted":true,"task_id":old.task,"prompt":old.prompt,"model":receipt.run.model,"effort":self.manifest.effort}).to_string());
        }
        let _lock = factory.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        self.ensure_live(factory, receipt, private, deadline)?;
        let journal = factory.load_journal(&receipt.run.project, &receipt.run.id)?;
        let committed = |key: &str| {
            journal
                .events()
                .iter()
                .any(|event| event.task_key == key && event.event_type == TASK_EVENT_TASK_COMMITTED)
        };
        if journal
            .events()
            .iter()
            .any(|event| matches!(event.event_type.as_str(), "task_failed" | "run_stopped"))
            || task.depends_on.iter().any(|id| !committed(id))
        {
            return Ok(
                json!({"admitted":false,"task_id":task.id,"reason":"task_not_ready"}).to_string(),
            );
        }
        let lane = self
            .plan
            .lanes
            .iter()
            .find(|lane| lane.id == task.lane)
            .ok_or_else(invalid)?;
        if !lane.task_ids.contains(&task.id)
            || lane
                .task_ids
                .iter()
                .take_while(|id| **id != task.id)
                .any(|id| !committed(id))
        {
            return Ok(
                json!({"admitted":false,"task_id":task.id,"reason":"lane_not_ready"}).to_string(),
            );
        }
        let active: Vec<_> = self
            .ledger
            .selections
            .iter()
            .filter(|selection| !committed(&selection.task))
            .collect();
        if active.len() >= self.manifest.max_children as usize {
            return Ok(
                json!({"admitted":false,"task_id":task.id,"reason":"child_bound"}).to_string(),
            );
        }
        for selection in active {
            let other = self.task(&selection.task)?;
            if tasks_conflict(other, &task) {
                return Ok(
                    json!({"admitted":false,"task_id":task.id,"reason":"ownership_occupied"})
                        .to_string(),
                );
            }
        }
        for earlier in self
            .plan
            .tasks
            .iter()
            .take_while(|earlier| earlier.id != task.id)
        {
            if committed(&earlier.id)
                || self
                    .ledger
                    .selections
                    .iter()
                    .any(|selection| selection.task == earlier.id)
                || earlier.depends_on.iter().any(|id| !committed(id))
            {
                continue;
            }
            let lane = self
                .plan
                .lanes
                .iter()
                .find(|lane| lane.id == earlier.lane)
                .ok_or_else(invalid)?;
            let ready = lane
                .task_ids
                .iter()
                .take_while(|id| **id != earlier.id)
                .all(|id| committed(id))
                && self
                    .ledger
                    .selections
                    .iter()
                    .filter(|selection| !committed(&selection.task))
                    .all(|selection| {
                        self.task(&selection.task)
                            .is_ok_and(|active| !tasks_conflict(active, earlier))
                    });
            if ready {
                return Ok(json!({"admitted":false,"task_id":task.id,"reason":"earlier_task_ready","ready_task_id":earlier.id}).to_string());
            }
        }
        let nonce = crate::terminal::factory::msp::new_command_id().map_err(|_| invalid())?;
        let child_prompt = format!("Factory native task {}. Admission nonce: {}. Manifest: {}. Parent run: {}. Deadline: {}. Work only on this exact admitted task in the shared checkout: {}. Do not spawn agents, modify Git metadata, perform commits, use credentials, or edit any other task's paths. Return only after releasing your source writes; the host derives and checks the immutable output tree. Do not resume work after completion.",task.id,nonce,receipt.run.manifest_digest,receipt.run.id,receipt.run.deadline,serde_json::to_string(&task).map_err(|_|invalid())?);
        if child_prompt.len() > 24 * 1024 {
            return Err(invalid());
        }
        self.ledger.selections.push(Selection {
            task: task.id.clone(),
            call: call.into(),
            request: request.clone(),
            nonce: nonce.clone(),
            prompt: child_prompt.clone(),
            ..Selection::default()
        });
        // Persist the exact native request and intent before returning a prompt.
        self.persist(private)?;
        factory.append_checkpoint_event(
            &self.request(receipt, &task.id),
            TASK_EVENT_CHILD_START_INTENT,
            "",
            &nonce,
        )?;
        Ok(json!({"admitted":true,"task_id":task.id,"prompt":child_prompt,"model":receipt.run.model,"effort":self.manifest.effort}).to_string())
    }

    pub(super) fn item<E: Executor, T: FactoryTerminal, B: FactoryBroker>(
        &mut self,
        factory: &Factory<E, T, B>,
        receipt: &FactoryReceipt,
        private: &PrivateState,
        method: &str,
        params: &Value,
        deadline: Instant,
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
        if receipt.run.execution_phase == EXECUTION_CORRECT
            || params["threadId"] != self.ledger.thread
        {
            return Err(invalid());
        }
        if item["type"] == "collabAgentToolCall" {
            if item["senderThreadId"] != self.ledger.thread {
                return Err(invalid());
            }
            let tool = item["tool"].as_str().ok_or_else(invalid)?;
            if tool == "spawnAgent" {
                let prompt = item["prompt"].as_str().ok_or_else(invalid)?;
                let index = self
                    .ledger
                    .selections
                    .iter()
                    .position(|selection| selection.prompt == prompt)
                    .ok_or_else(invalid)?;
                let spawn = item["id"]
                    .as_str()
                    .filter(|id| !id.is_empty() && id.len() <= 128)
                    .ok_or_else(invalid)?;
                let selection = &mut self.ledger.selections[index];
                if !selection.spawn.is_empty() && selection.spawn != spawn {
                    return Err(FactoryError::Stale);
                }
                selection.spawn = spawn.into();
                if method == "item/completed" {
                    if item["status"] != "completed" {
                        return Err(invalid());
                    }
                    let children = item["receiverThreadIds"]
                        .as_array()
                        .filter(|ids| ids.len() == 1)
                        .ok_or_else(invalid)?;
                    let child = children[0]
                        .as_str()
                        .filter(|id| !id.is_empty() && id.len() <= 128)
                        .ok_or_else(invalid)?
                        .to_owned();
                    if !selection.child.is_empty() && selection.child != child {
                        return Err(FactoryError::Stale);
                    }
                    selection.child = child.clone();
                    let model = item["model"].as_str().ok_or_else(invalid)?;
                    let effort = item["reasoningEffort"].as_str().ok_or_else(invalid)?;
                    if model != receipt.run.model || effort != self.manifest.effort {
                        return Err(invalid());
                    }
                    if self
                        .ledger
                        .selections
                        .iter()
                        .enumerate()
                        .any(|(other, selection)| other != index && selection.child == child)
                    {
                        return Err(FactoryError::Stale);
                    }
                    if self
                        .ledger
                        .profiles
                        .get(&child)
                        .is_some_and(|old| old != &(model.into(), effort.into()))
                    {
                        return Err(FactoryError::Stale);
                    }
                    self.ledger
                        .profiles
                        .insert(child.clone(), (model.into(), effort.into()));
                    if item["agentsStates"][&child]["status"] == "completed" {
                        self.ledger.selections[index].result_complete = true;
                        self.ledger.activity_completed.insert(child, true);
                    }
                }
            } else if matches!(tool, "wait" | "closeAgent") {
                for child in item["receiverThreadIds"].as_array().ok_or_else(invalid)? {
                    let child = child.as_str().ok_or_else(invalid)?;
                    let selection = self
                        .ledger
                        .selections
                        .iter_mut()
                        .find(|selection| selection.child == child)
                        .ok_or_else(invalid)?;
                    if method == "item/completed"
                        && item["agentsStates"][child]["status"] == "completed"
                    {
                        selection.result_complete = true;
                        self.ledger.activity_completed.insert(child.into(), true);
                    }
                }
            } else {
                return Err(invalid());
            }
        } else {
            let child = item["agentThreadId"]
                .as_str()
                .filter(|id| !id.is_empty() && id.len() <= 128)
                .ok_or_else(invalid)?;
            if item["kind"] == "started" {
                let model = item["model"].as_str().ok_or_else(invalid)?;
                let effort = item["reasoningEffort"].as_str().ok_or_else(invalid)?;
                if model != receipt.run.model || effort != self.manifest.effort {
                    return Err(invalid());
                }
                if let Some(old) = self.ledger.profiles.get(child) {
                    if old != &(model.into(), effort.into()) {
                        return Err(FactoryError::Stale);
                    }
                } else {
                    if self.ledger.profiles.len() >= 64 {
                        return Err(invalid());
                    }
                    self.ledger
                        .profiles
                        .insert(child.into(), (model.into(), effort.into()));
                }
            } else if item["kind"] == "completed" {
                if !self.ledger.profiles.contains_key(child) {
                    return Err(invalid());
                }
                self.ledger.activity_completed.insert(child.into(), true);
            } else if item["kind"] == "interrupted" {
                return Err(invalid());
            }
        }
        self.persist(private)?;
        self.release_observed(factory, receipt, private, deadline)
    }

    fn release_observed<E: Executor, T: FactoryTerminal, B: FactoryBroker>(
        &mut self,
        factory: &Factory<E, T, B>,
        receipt: &FactoryReceipt,
        private: &PrivateState,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        for index in 0..self.ledger.selections.len() {
            if self
                .ledger
                .activity_completed
                .contains_key(&self.ledger.selections[index].child)
            {
                self.ledger.selections[index].activity_complete = true;
            }
            let selection = self.ledger.selections[index].clone();
            if selection.child.is_empty() || !self.ledger.profiles.contains_key(&selection.child) {
                continue;
            }
            if !selection.started {
                let _lock = factory.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
                self.ensure_live(factory, receipt, private, deadline)?;
                factory.append_checkpoint_event(
                    &self.request(receipt, &selection.task),
                    TASK_EVENT_CHILD_STARTED,
                    &selection.child,
                    "",
                )?;
                self.ledger.selections[index].started = true;
                self.persist(private)?;
            }
            if selection.activity_complete
                && selection.result_complete
                && selection.output.is_empty()
            {
                let output = factory.observe_task_output(
                    &receipt.run.project,
                    &receipt.run.id,
                    &selection.task,
                    &selection.child,
                    deadline,
                )?;
                self.ledger.selections[index].output = output;
                self.persist(private)?;
            }
        }
        Ok(())
    }
}
