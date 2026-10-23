//! Trusted task output, Git checkpoint and final integration owner.
mod checks;
mod git;
mod state;
mod wire;

use super::deadline::{parse_deadline, seconds_until};
use super::receipt::FactoryReceipt;
use super::{
    ExecutionManifest, Factory, FactoryBroker, FactoryError, FactoryRun, FactoryTerminal,
    TaskEvent, EXECUTION_CORRECT, FACTORY_RUNNING, TASK_EVENT_CHECKPOINT_PREPARED,
    TASK_EVENT_CHILD_REPORTED, TASK_EVENT_CHILD_STARTED, TASK_EVENT_TASK_COMMITTED,
    TASK_EVENT_TASK_FAILED,
};
use crate::project::{Config, Executor, Runtime};
use crate::terminal::Binding;
use git::GitContext;
use state::{
    observation_name, operation_key, operation_name, CheckpointOperation, OutputObservation,
};
use std::time::{Instant, SystemTime, UNIX_EPOCH};
use wire::{CheckpointPlan, CheckpointTask};

pub(crate) use state::PrivateState;
pub use wire::{CheckpointCheckResult, FactoryTaskCheckpoint, FactoryTaskCheckpointState};

struct Admitted {
    run: FactoryRun,
    binding: Binding,
    plan: CheckpointPlan,
    context: GitContext,
    state: PrivateState,
}

fn err(reason: String) -> FactoryError {
    FactoryError::msg(reason)
}

fn response(op: &CheckpointOperation) -> FactoryTaskCheckpointState {
    let bytes = serde_json::to_vec(op).expect("serializable checkpoint record");
    FactoryTaskCheckpointState {
        request: op.request.clone(),
        final_candidate: op.final_candidate,
        status: match op.phase.as_str() {
            "committed" => "committed",
            "failed" => "failed",
            _ => "pending",
        }
        .into(),
        candidate: op.candidate.clone(),
        tree: op.tree.clone(),
        receipt_digest: crate::sha256::hex_lower(&crate::sha256::digest(&bytes)),
        checks: op.checks.clone(),
        reason: op.reason.clone(),
    }
}

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    fn admit_checkpoint(
        &self,
        req: &FactoryTaskCheckpoint,
        final_candidate: bool,
        live: bool,
        deadline: Instant,
    ) -> Result<Admitted, FactoryError> {
        req.validate(final_candidate).map_err(err)?;
        let state = PrivateState::open(&self.state_dir).map_err(err)?;
        let (receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let binding = receipt.binding.as_ref().ok_or(FactoryError::Stale)?;
        if receipt.run.manifest_digest != req.manifest_digest
            || binding.project != req.container
            || binding.invocation_id != req.invocation
            || !super::execution_manifest_required(&receipt.run.execution_phase)
        {
            return Err(FactoryError::Stale);
        }
        if live
            && (state
                .read::<bool>(&format!("retiring-{}-{}.json", req.project, req.id))
                .map_err(err)?
                .unwrap_or(false)
                || receipt.phase != FACTORY_RUNNING
                || !self.terminal.live(binding, deadline)
                || seconds_until(parse_deadline(&receipt.run.deadline).unwrap_or_default()) <= 0)
        {
            return Err(FactoryError::Stale);
        }
        let runtime = Runtime {
            exec: &self.exec,
            config: Config::default(),
        };
        if runtime
            .prepare_container(&req.project, true, deadline)
            .map_err(err)?
            != req.container
        {
            return Err(FactoryError::Stale);
        }
        let bytes = self.load_execution_manifest(&receipt.run)?;
        let manifest = ExecutionManifest::decode(&bytes).map_err(err)?;
        let plan = CheckpointPlan::decode(&manifest).map_err(err)?;
        if receipt.run.execution_phase == EXECUTION_CORRECT
            && (plan.tasks.len() != 1 || manifest.max_children != 0)
        {
            return Err(FactoryError::msg(
                "correction checkpoint must remain a singleton",
            ));
        }
        let context = GitContext::new(
            &receipt.run,
            req,
            binding.uid,
            binding.gid,
            &operation_key(req, final_candidate),
        );
        Ok(Admitted {
            run: receipt.run,
            binding: binding.clone(),
            plan,
            context,
            state,
        })
    }

    pub(in crate::factory) fn guard_checkpoint_checkout(
        &self,
        receipt: &FactoryReceipt,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        if !super::execution_manifest_required(&receipt.run.execution_phase) {
            return Ok(());
        }
        PrivateState::open(&self.state_dir).map_err(err)?;
        let binding = receipt.binding.as_ref().ok_or(FactoryError::Stale)?;
        let req = FactoryTaskCheckpoint {
            project: receipt.run.project.clone(),
            id: receipt.run.id.clone(),
            container: binding.project.clone(),
            invocation: binding.invocation_id.clone(),
            manifest_digest: receipt.run.manifest_digest.clone(),
            expected_head: receipt.run.source_commit.clone(),
            ..FactoryTaskCheckpoint::default()
        };
        GitContext::new(&receipt.run, &req, binding.uid, binding.gid, "guard")
            .guard(&self.exec, &receipt.run.source_commit, deadline)
            .map_err(err)
    }

    fn integration_head(&self, admitted: &Admitted) -> Result<String, FactoryError> {
        let journal = self.load_journal(&admitted.run.project, &admitted.run.id)?;
        let mut head = admitted.plan.source_commit.clone();
        for event in journal.events() {
            if event.invocation != admitted.binding.invocation_id
                || event.plan_digest != admitted.run.manifest_digest
            {
                return Err(FactoryError::Stale);
            }
            if event.event_type == TASK_EVENT_TASK_COMMITTED {
                for field in event.detail.split_whitespace() {
                    if let Some(commit) = field.strip_prefix("commit=") {
                        if !crate::preparation::valid_commit(commit) {
                            return Err(FactoryError::Stale);
                        }
                        head = commit.into();
                    }
                }
            }
        }
        Ok(head)
    }

    /// Only native transport calls this after observing completion and release.
    /// There is no HTTP/model entry point. The digest names the owned tree.
    pub fn observe_task_output(
        &self,
        project: &str,
        run: &str,
        task_key: &str,
        child: &str,
        deadline: Instant,
    ) -> Result<String, FactoryError> {
        if !crate::domain::valid_id(project)
            || !super::valid_factory_run_id(run)
            || !super::valid_task_key(task_key)
        {
            return Err(FactoryError::Stale);
        }
        let _lock = self.lock_run(project, run, deadline)?;
        let (receipt, exists) = self.load_receipt(project, run)?;
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let binding = receipt.binding.as_ref().ok_or(FactoryError::Stale)?;
        let mut req = FactoryTaskCheckpoint {
            project: project.into(),
            id: run.into(),
            container: binding.project.clone(),
            invocation: binding.invocation_id.clone(),
            manifest_digest: receipt.run.manifest_digest.clone(),
            task_key: task_key.into(),
            output_digest: "0".repeat(64),
            expected_head: receipt.run.source_commit.clone(),
        };
        let mut admitted = self.admit_checkpoint(&req, false, true, deadline)?;
        let task = admitted
            .plan
            .tasks
            .iter()
            .find(|task| task.id == task_key)
            .ok_or_else(|| FactoryError::msg("native output names an unknown task"))?
            .clone();
        let journal = self.load_journal(project, run)?;
        let correct = admitted.run.execution_phase == EXECUTION_CORRECT;
        let started = journal.events().iter().rev().find(|event| {
            event.task_key == task_key
                && event.event_type
                    == if correct {
                        "parent_task_started"
                    } else {
                        TASK_EVENT_CHILD_STARTED
                    }
        });
        if started.is_none_or(|event| {
            event.invocation != req.invocation
                || event.plan_digest != req.manifest_digest
                || event.child_id != child
                || correct != child.is_empty()
        }) {
            return Err(FactoryError::msg(
                "native output lacks its observed task owner",
            ));
        }
        req.expected_head = self.integration_head(&admitted)?;
        admitted.context = GitContext::new(
            &admitted.run,
            &req,
            binding.uid,
            binding.gid,
            &operation_key(&req, false),
        );
        if let Some(old) = admitted
            .state
            .read::<OutputObservation>(&observation_name(&req))
            .map_err(err)?
        {
            if old.request.project != req.project
                || old.request.id != req.id
                || old.request.invocation != req.invocation
                || old.request.manifest_digest != req.manifest_digest
                || old.request.task_key != req.task_key
                || old.child != child
            {
                return Err(FactoryError::Stale);
            }
            self.append_checkpoint_event(
                &old.request,
                if correct {
                    "parent_output_observed"
                } else {
                    TASK_EVENT_CHILD_REPORTED
                },
                child,
                &old.request.output_digest,
            )?;
            return Ok(old.request.output_digest);
        }
        let (parent_tree, tree) = admitted
            .context
            .observe(&self.exec, &task, &req.expected_head, deadline)
            .map_err(err)?;
        req.output_digest = crate::sha256::hex_lower(&crate::sha256::digest(
            format!(
                "released-output-v1\0{}\0{}\0{}\0{}\0{}\0{tree}",
                req.id, req.manifest_digest, req.invocation, req.task_key, child
            )
            .as_bytes(),
        ));
        let observation = OutputObservation {
            request: req.clone(),
            child: child.into(),
            tree,
            parent_tree,
        };
        admitted
            .state
            .write(&observation_name(&req), &observation)
            .map_err(err)?;
        self.append_checkpoint_event(
            &req,
            if correct {
                "parent_output_observed"
            } else {
                TASK_EVENT_CHILD_REPORTED
            },
            child,
            &req.output_digest,
        )?;
        Ok(req.output_digest)
    }

    pub(in crate::factory) fn append_checkpoint_event(
        &self,
        req: &FactoryTaskCheckpoint,
        event_type: &str,
        child: &str,
        detail: &str,
    ) -> Result<(), FactoryError> {
        let mut journal = self.load_journal(&req.project, &req.id)?;
        if let Some(old) = journal
            .events()
            .iter()
            .find(|event| event.task_key == req.task_key && event.event_type == event_type)
        {
            return if old.invocation == req.invocation
                && old.plan_digest == req.manifest_digest
                && old.child_id == child
                && old.detail == detail
            {
                Ok(())
            } else {
                Err(FactoryError::Stale)
            };
        }
        let event = TaskEvent {
            sequence: journal.len() as i64 + 1,
            prev_digest: journal
                .events()
                .last()
                .map(TaskEvent::digest)
                .unwrap_or_default(),
            event_type: event_type.into(),
            task_key: req.task_key.clone(),
            child_id: child.into(),
            invocation: req.invocation.clone(),
            plan_digest: req.manifest_digest.clone(),
            detail: detail.into(),
        };
        journal.append(&event).map_err(err)?;
        self.store_journal(&req.project, &req.id, journal.events())
    }

    pub fn task_checkpoint(
        &self,
        req: &FactoryTaskCheckpoint,
        final_candidate: bool,
        deadline: Instant,
    ) -> Result<FactoryTaskCheckpointState, FactoryError> {
        req.validate(final_candidate).map_err(err)?;
        let _lock = self.lock_run(&req.project, &req.id, deadline)?;
        let admitted = self.admit_checkpoint(req, final_candidate, false, deadline)?;
        if let Some(old) = admitted
            .state
            .read::<CheckpointOperation>(&operation_name(req, final_candidate))
            .map_err(err)?
        {
            if old.request != *req || old.final_candidate != final_candidate {
                return Err(FactoryError::Stale);
            }
            if matches!(old.phase.as_str(), "committed" | "failed") {
                if old.phase == "failed" && !final_candidate {
                    let observation = admitted
                        .state
                        .read::<OutputObservation>(&observation_name(req))
                        .map_err(err)?
                        .ok_or(FactoryError::Stale)?;
                    self.append_checkpoint_event(
                        req,
                        TASK_EVENT_TASK_FAILED,
                        &observation.child,
                        &old.reason,
                    )?;
                }
                return Ok(response(&old));
            }
        }
        let admitted = self.admit_checkpoint(req, final_candidate, true, deadline)?;
        for task in &admitted.plan.tasks {
            let mut other = req.clone();
            other.task_key = task.id.clone();
            if let Some(op) = admitted
                .state
                .read::<CheckpointOperation>(&operation_name(&other, false))
                .map_err(err)?
            {
                if op.request.task_key != req.task_key
                    && matches!(op.phase.as_str(), "preparing" | "checking" | "checked")
                {
                    return Ok(FactoryTaskCheckpointState {
                        request: req.clone(),
                        final_candidate,
                        status: "busy".into(),
                        reason: "another_checkpoint_active".into(),
                        ..FactoryTaskCheckpointState::default()
                    });
                }
                if op.phase == "failed" {
                    return Err(FactoryError::msg(
                        "a failed task closes checkpoint admission",
                    ));
                }
            }
        }
        let mut op = match admitted
            .state
            .read::<CheckpointOperation>(&operation_name(req, final_candidate))
            .map_err(err)?
        {
            Some(op) => op,
            None => self.begin_checkpoint(&admitted, req, final_candidate, deadline)?,
        };
        if op.phase == "preparing" {
            op.candidate = match admitted.context.prepare(&self.exec, &op, deadline) {
                Ok(candidate) => candidate,
                Err(_) => {
                    return self.fail_checkpoint(
                        &admitted,
                        op,
                        "checkpoint_preparation_failed",
                        deadline,
                    )
                }
            };
            op.phase = "checking".into();
            admitted
                .state
                .write(&operation_name(req, final_candidate), &op)
                .map_err(err)?;
        }
        if op.phase == "checking" {
            if op.check_index < op.selected_checks.len() {
                let check = op.selected_checks[op.check_index].clone();
                if !op.check_launch_intent {
                    op.check_launch_intent = true;
                    admitted
                        .state
                        .write(&operation_name(req, final_candidate), &op)
                        .map_err(err)?;
                    let remaining =
                        seconds_until(parse_deadline(&admitted.run.deadline).unwrap_or_default())
                            .max(0) as u64;
                    let _ = checks::start(
                        &self.exec,
                        &admitted.context,
                        &op,
                        &check,
                        remaining,
                        deadline,
                    );
                    return Ok(response(&op));
                }
                match checks::observe(&self.exec, &admitted.context, &op, &check, deadline) {
                    Ok(checks::Observation::Pending(invocation)) => {
                        op.check_invocation = invocation;
                        admitted
                            .state
                            .write(&operation_name(req, final_candidate), &op)
                            .map_err(err)?;
                        return Ok(response(&op));
                    }
                    Ok(checks::Observation::Complete(invocation, result)) => {
                        op.check_invocation = invocation;
                        let failed = !result.passed && check.required;
                        op.checks.push(result);
                        op.check_index += 1;
                        op.check_launch_intent = false;
                        op.check_invocation.clear();
                        admitted
                            .state
                            .write(&operation_name(req, final_candidate), &op)
                            .map_err(err)?;
                        if failed {
                            return self.fail_checkpoint(
                                &admitted,
                                op,
                                "required_check_failed",
                                deadline,
                            );
                        }
                        return Ok(response(&op));
                    }
                    Err(_) => {
                        return self.fail_checkpoint(
                            &admitted,
                            op,
                            "check_outcome_unconfirmed",
                            deadline,
                        )
                    }
                }
            }
            if admitted
                .context
                .verify_export(&self.exec, &op, deadline)
                .is_err()
            {
                return self.fail_checkpoint(&admitted, op, "checked_export_changed", deadline);
            }
            if !final_candidate {
                let observation = admitted
                    .state
                    .read::<OutputObservation>(&observation_name(req))
                    .map_err(err)?
                    .ok_or(FactoryError::Stale)?;
                let task = admitted
                    .plan
                    .tasks
                    .iter()
                    .find(|task| task.id == req.task_key)
                    .ok_or(FactoryError::Stale)?;
                if admitted
                    .context
                    .observe(&self.exec, task, &req.expected_head, deadline)
                    .map_err(err)?
                    .1
                    != observation.tree
                {
                    return self.fail_checkpoint(
                        &admitted,
                        op,
                        "released_output_changed",
                        deadline,
                    );
                }
            } else if admitted
                .context
                .inspect_final(&self.exec, &req.expected_head, &[], deadline)
                .is_err()
            {
                return self.fail_checkpoint(&admitted, op, "final_source_changed", deadline);
            }
            op.phase = "checked".into();
            admitted
                .state
                .write(&operation_name(req, final_candidate), &op)
                .map_err(err)?;
        }
        if op.phase == "checked" {
            self.admit_checkpoint(req, final_candidate, true, deadline)?;
            if !admitted
                .context
                .commit_binding_current(&self.exec, &op, deadline)
                .map_err(err)?
            {
                return self.fail_checkpoint(
                    &admitted,
                    op,
                    "integration_head_or_index_changed",
                    deadline,
                );
            }
            let child = if final_candidate {
                String::new()
            } else {
                admitted
                    .state
                    .read::<OutputObservation>(&observation_name(req))
                    .map_err(err)?
                    .ok_or(FactoryError::Stale)?
                    .child
            };
            if !final_candidate {
                let checks = op
                    .checks
                    .iter()
                    .filter(|r| r.passed)
                    .map(|r| r.id.as_str())
                    .collect::<Vec<_>>()
                    .join(" ");
                self.append_checkpoint_event(req, TASK_EVENT_CHECKPOINT_PREPARED, &child, &checks)?;
            }
            if admitted.context.commit(&self.exec, &op, deadline).is_err() {
                return Ok(response(&op));
            }
            if !final_candidate {
                let detail = if op.candidate == req.expected_head {
                    format!("tree={}", op.tree)
                } else {
                    format!("tree={} commit={}", op.tree, op.candidate)
                };
                self.append_checkpoint_event(req, TASK_EVENT_TASK_COMMITTED, &child, &detail)?;
            }
            op.phase = "committed".into();
            admitted
                .state
                .write(&operation_name(req, final_candidate), &op)
                .map_err(err)?;
        }
        Ok(response(&op))
    }

    /// Observe an already retained operation after parent retirement. This
    /// path never launches a check, changes Git or completes an unfinished op.
    pub fn checkpoint_receipt(
        &self,
        req: &FactoryTaskCheckpoint,
        final_candidate: bool,
        deadline: Instant,
    ) -> Result<FactoryTaskCheckpointState, FactoryError> {
        req.validate(final_candidate).map_err(err)?;
        let _lock = self.lock_run(&req.project, &req.id, deadline)?;
        let admitted = self.admit_checkpoint(req, final_candidate, false, deadline)?;
        match admitted
            .state
            .read::<CheckpointOperation>(&operation_name(req, final_candidate))
            .map_err(err)?
        {
            Some(op) if op.request == *req && op.final_candidate == final_candidate => {
                Ok(response(&op))
            }
            Some(_) => Err(FactoryError::Stale),
            None => Ok(FactoryTaskCheckpointState {
                request: req.clone(),
                final_candidate,
                status: "pending".into(),
                reason: "checkpoint_receipt_unavailable".into(),
                ..FactoryTaskCheckpointState::default()
            }),
        }
    }

    fn begin_checkpoint(
        &self,
        admitted: &Admitted,
        req: &FactoryTaskCheckpoint,
        final_candidate: bool,
        deadline: Instant,
    ) -> Result<CheckpointOperation, FactoryError> {
        if self.integration_head(admitted)? != req.expected_head {
            return Err(FactoryError::Stale);
        }
        let (parent_tree, tree, selected_checks) = if final_candidate {
            let mut commits = Vec::new();
            for task in &admitted.plan.tasks {
                let mut task_req = req.clone();
                task_req.task_key = task.id.clone();
                let op = admitted
                    .state
                    .read::<CheckpointOperation>(&operation_name(&task_req, false))
                    .map_err(err)?
                    .ok_or_else(|| FactoryError::msg("final candidate has incomplete tasks"))?;
                if op.phase != "committed"
                    || op.request.invocation != req.invocation
                    || op.request.container != req.container
                    || op.request.manifest_digest != req.manifest_digest
                {
                    return Err(FactoryError::msg(
                        "final candidate has unconfirmed task evidence",
                    ));
                }
                commits.push(op.candidate);
            }
            let tree = admitted
                .context
                .inspect_final(&self.exec, &req.expected_head, &commits, deadline)
                .map_err(err)?;
            (
                tree.clone(),
                tree,
                admitted
                    .plan
                    .checks
                    .iter()
                    .filter(|check| check.scope == "integration" && check.required)
                    .cloned()
                    .collect(),
            )
        } else {
            let observation = admitted
                .state
                .read::<OutputObservation>(&observation_name(req))
                .map_err(err)?
                .ok_or_else(|| FactoryError::msg("checkpoint lacks observed native output"))?;
            let observed = &observation.request;
            if observed.project != req.project
                || observed.id != req.id
                || observed.container != req.container
                || observed.invocation != req.invocation
                || observed.manifest_digest != req.manifest_digest
                || observed.task_key != req.task_key
                || observed.output_digest != req.output_digest
            {
                return Err(FactoryError::Stale);
            }
            let task: &CheckpointTask = admitted
                .plan
                .tasks
                .iter()
                .find(|task| task.id == req.task_key)
                .ok_or(FactoryError::Stale)?;
            let (parent_tree, released) = admitted
                .context
                .observe(&self.exec, task, &req.expected_head, deadline)
                .map_err(err)?;
            if released != observation.tree {
                return Err(FactoryError::msg("released task output changed"));
            }
            let tree = admitted
                .context
                .integrate(&self.exec, task, &req.expected_head, &released, deadline)
                .map_err(err)?;
            let selected = admitted
                .plan
                .checks
                .iter()
                .filter(|check| task.check_ids.contains(&check.id))
                .cloned()
                .collect();
            (parent_tree, tree, selected)
        };
        let op = CheckpointOperation {
            request: req.clone(),
            final_candidate,
            phase: "preparing".into(),
            parent_tree,
            tree,
            candidate: String::new(),
            created_unix: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .map_err(|_| FactoryError::Stale)?
                .as_secs(),
            checks: Vec::new(),
            reason: String::new(),
            context: admitted.context.clone(),
            selected_checks,
            check_index: 0,
            check_invocation: String::new(),
            check_launch_intent: false,
        };
        admitted
            .state
            .write(&operation_name(req, final_candidate), &op)
            .map_err(err)?;
        Ok(op)
    }

    fn fail_checkpoint(
        &self,
        admitted: &Admitted,
        mut op: CheckpointOperation,
        reason: &str,
        deadline: Instant,
    ) -> Result<FactoryTaskCheckpointState, FactoryError> {
        if op.check_launch_intent {
            if let Some(check) = op.selected_checks.get(op.check_index) {
                checks::stop(&self.exec, &admitted.context, &op, check, deadline).map_err(err)?;
            }
        }
        op.phase = "failed".into();
        op.reason = reason.into();
        admitted
            .state
            .write(&operation_name(&op.request, op.final_candidate), &op)
            .map_err(err)?;
        if !op.final_candidate {
            let observation = admitted
                .state
                .read::<OutputObservation>(&observation_name(&op.request))
                .map_err(err)?
                .ok_or(FactoryError::Stale)?;
            self.append_checkpoint_event(
                &op.request,
                TASK_EVENT_TASK_FAILED,
                &observation.child,
                reason,
            )?;
        }
        Ok(response(&op))
    }

    pub(in crate::factory) fn retire_checkpoint_checks(
        &self,
        receipt: &FactoryReceipt,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        if !super::execution_manifest_required(&receipt.run.execution_phase) {
            return Ok(());
        }
        let Some(binding) = &receipt.binding else {
            return Ok(());
        };
        let state = PrivateState::open(&self.state_dir).map_err(err)?;
        let bytes = self.load_execution_manifest(&receipt.run)?;
        let manifest = ExecutionManifest::decode(&bytes).map_err(err)?;
        let plan = CheckpointPlan::decode(&manifest).map_err(err)?;
        for key in plan
            .tasks
            .iter()
            .map(|task| task.id.as_str())
            .chain(std::iter::once(""))
        {
            let req = FactoryTaskCheckpoint {
                project: receipt.run.project.clone(),
                id: receipt.run.id.clone(),
                container: binding.project.clone(),
                invocation: binding.invocation_id.clone(),
                manifest_digest: receipt.run.manifest_digest.clone(),
                task_key: key.into(),
                expected_head: receipt.run.source_commit.clone(),
                ..FactoryTaskCheckpoint::default()
            };
            if let Some(op) = state
                .read::<CheckpointOperation>(&operation_name(&req, key.is_empty()))
                .map_err(err)?
            {
                if op.check_launch_intent && !matches!(op.phase.as_str(), "committed" | "failed") {
                    if let Some(check) = op.selected_checks.get(op.check_index) {
                        checks::stop(&self.exec, &op.context, &op, check, deadline).map_err(err)?;
                    }
                }
            }
        }
        Ok(())
    }

    pub(in crate::factory) fn fence_checkpoint_retirement(
        &self,
        receipt: &FactoryReceipt,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        if !super::execution_manifest_required(&receipt.run.execution_phase) {
            return Ok(());
        }
        // Serialize against a check launch, then prohibit any subsequent one
        // before observing and retiring all retained native check units.
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        PrivateState::open(&self.state_dir)
            .map_err(err)?
            .write(
                &format!("retiring-{}-{}.json", receipt.run.project, receipt.run.id),
                &true,
            )
            .map_err(err)
    }
}
