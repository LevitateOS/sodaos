//! Ready-task execution around the native child table (D05 / FW14).
//!
//! The runner couples one admitted parent's [`ChildTable`] to its typed
//! event journal: starts record `child_start_intent` + `child_started`,
//! terminal observations record `child_reported` / `task_committed` /
//! `task_failed`, and parent stop retires every child before recording
//! `run_stopped`. Every event binds run, invocation, plan digest, task,
//! sequence and the previous-event digest, so the journal the daemon
//! persists and serves is verifiable end to end. Ordinary prose can never
//! advance task state: there is no prose path here.
//!
//! Scheduling (dependencies, lanes, aggregate choice) stays with the Go
//! control driver, which folds these same events into persisted receipts.
//! This runner enforces the per-task lifecycle, one child per task, the
//! admitted child cap and parent-stop retirement around real native
//! children. Daemon persistence of the journal snapshot and the
//! manifest-to-runner config threading ride the FW24 integration joint.

use std::collections::HashMap;
use std::path::Path;
use std::time::{Duration, Instant};

use crate::preparation;
use crate::terminal::factory::children::{ChildObservation, ChildProfile, ChildSpec, ChildTable};
use crate::terminal::factory::run::{valid_factory_run_id, valid_harness_family};

use super::events::{
    EventJournal, TaskEvent, TASK_EVENT_CHECKPOINT_PREPARED, TASK_EVENT_CHILD_REPORTED,
    TASK_EVENT_CHILD_STARTED, TASK_EVENT_CHILD_START_INTENT, TASK_EVENT_RUN_STOPPED,
    TASK_EVENT_TASK_COMMITTED, TASK_EVENT_TASK_FAILED,
};

/// Reason bound mirroring the Go receipt bound: terminal reasons past it
/// truncate with the same marker instead of refusing the task outcome.
const MAX_TASK_REASON_BYTES: usize = 4 * 1024;

fn truncate_reason(reason: &str) -> String {
    if reason.len() <= MAX_TASK_REASON_BYTES {
        return reason.to_string();
    }
    let mut out = String::with_capacity(MAX_TASK_REASON_BYTES);
    for (index, ch) in reason.char_indices() {
        if index + ch.len_utf8() > MAX_TASK_REASON_BYTES - 12 {
            break;
        }
        out.push(ch);
    }
    out.push_str("…[truncated]");
    out
}

#[derive(Debug, Clone, PartialEq, Eq)]
enum TaskLifecycle {
    Running { child_id: String },
    Reported { child_id: String },
    Completed,
    Failed,
    Cancelled,
}

impl TaskLifecycle {
    fn active(&self) -> bool {
        matches!(
            self,
            TaskLifecycle::Running { .. } | TaskLifecycle::Reported { .. }
        )
    }
}

/// Native task execution for one admitted implement parent run. The run
/// identity rides the child table and the journal scope; the plan digest
/// binds every recorded event.
pub struct TaskRunner {
    invocation: String,
    plan_digest: String,
    harness: String,
    profile: ChildProfile,
    table: ChildTable,
    journal: EventJournal,
    states: HashMap<String, TaskLifecycle>,
}

impl TaskRunner {
    /// Bind one runner to its admitted parent. The scratch root holds
    /// per-child logs and must live under `/home`, outside any
    /// agent-writable checkout.
    pub fn open(
        run: &str,
        invocation: &str,
        plan_digest: &str,
        harness: &str,
        profile: &ChildProfile,
        max_children: i64,
        scratch: &Path,
    ) -> Result<Self, String> {
        if !valid_factory_run_id(run) {
            return Err("invalid factory run identity".to_string());
        }
        if invocation.is_empty()
            || invocation.len() > 128
            || invocation.bytes().any(|b| b < 0x20 || b == 0x7f)
        {
            return Err("task event invocation is invalid".to_string());
        }
        if !preparation::valid_digest(plan_digest) {
            return Err("task event plan digest is invalid".to_string());
        }
        if !valid_harness_family(harness) {
            return Err("unsupported factory harness".to_string());
        }
        profile.validate(harness)?;
        Ok(TaskRunner {
            invocation: invocation.to_string(),
            plan_digest: plan_digest.to_string(),
            harness: harness.to_string(),
            profile: profile.clone(),
            table: ChildTable::open(run, invocation, max_children, scratch)?,
            journal: EventJournal::default(),
            states: HashMap::new(),
        })
    }

    fn next_event(
        &self,
        event_type: &str,
        task_key: &str,
        child_id: &str,
        detail: &str,
    ) -> TaskEvent {
        let sequence = self.journal.len() as i64 + 1;
        let prev_digest = self
            .journal
            .events()
            .last()
            .map(|head| head.digest())
            .unwrap_or_default();
        TaskEvent {
            sequence,
            prev_digest,
            event_type: event_type.to_string(),
            task_key: task_key.to_string(),
            child_id: child_id.to_string(),
            invocation: self.invocation.clone(),
            plan_digest: self.plan_digest.clone(),
            detail: detail.to_string(),
        }
    }

    fn record(&mut self, event: &TaskEvent) -> Result<(), String> {
        self.journal.append(event)
    }

    /// Start one ready task's child. A lost start reply is looked up, not
    /// replayed: a task that already owns a live record returns its child
    /// without recording again.
    pub fn start_task(&mut self, spec: &ChildSpec, deadline: Instant) -> Result<String, String> {
        if !crate::factory::valid_task_key(&spec.task_key) {
            return Err("task event task key is invalid".to_string());
        }
        if let Some(TaskLifecycle::Running { child_id }) = self.states.get(&spec.task_key) {
            return Ok(child_id.clone());
        }
        if let Some(state) = self.states.get(&spec.task_key) {
            if !matches!(state, TaskLifecycle::Running { .. }) {
                return Err("task already reached its terminal receipt".to_string());
            }
        }
        let child_id = self
            .table
            .start(spec, &self.profile, &self.harness, deadline)?;
        let intent = self.next_event(TASK_EVENT_CHILD_START_INTENT, &spec.task_key, &child_id, "");
        self.record(&intent)?;
        let started = self.next_event(TASK_EVENT_CHILD_STARTED, &spec.task_key, &child_id, "");
        self.record(&started)?;
        self.states.insert(
            spec.task_key.clone(),
            TaskLifecycle::Running {
                child_id: child_id.clone(),
            },
        );
        Ok(child_id)
    }

    /// Observe one child without blocking.
    pub fn observe_task(&mut self, task_key: &str) -> Result<ChildObservation, String> {
        self.table.observe(task_key)
    }

    /// Record one terminal child observation with its output digest. The
    /// child must have exited; a live child is progress, not a report.
    pub fn report_task(&mut self, task_key: &str, output_digest: &str) -> Result<i32, String> {
        let child_id = match self.states.get(task_key) {
            Some(TaskLifecycle::Running { child_id }) => child_id.clone(),
            _ => return Err("task reported without its running child".to_string()),
        };
        if !preparation::valid_digest(output_digest) {
            return Err("task report lacks its output digest".to_string());
        }
        let observation = self.table.observe(task_key)?;
        let Some(exit_code) = observation.exit_code else {
            return Err("task child still running".to_string());
        };
        let reported = self.next_event(
            TASK_EVENT_CHILD_REPORTED,
            task_key,
            &child_id,
            output_digest,
        );
        self.record(&reported)?;
        self.states
            .insert(task_key.to_string(), TaskLifecycle::Reported { child_id });
        Ok(exit_code)
    }

    /// Record executor-verified check evidence for one executing task. The
    /// detail carries space-separated plan check keys, exactly what the Go
    /// importer parses; unknown keys refuse at import.
    pub fn checkpoint_task(&mut self, task_key: &str, check_ids: &[String]) -> Result<(), String> {
        let child_id = match self.states.get(task_key) {
            Some(TaskLifecycle::Running { child_id })
            | Some(TaskLifecycle::Reported { child_id }) => child_id.clone(),
            _ => return Err("checkpoint precedes its task execution".to_string()),
        };
        for id in check_ids {
            if !crate::factory::valid_task_key(id) {
                return Err("checkpoint names an unknown plan check".to_string());
            }
        }
        let prepared = self.next_event(
            TASK_EVENT_CHECKPOINT_PREPARED,
            task_key,
            &child_id,
            &check_ids.join(" "),
        );
        self.record(&prepared)?;
        Ok(())
    }

    /// Record one accepted task output with its optional tree/commit
    /// evidence. Only a reported task can complete: the executor's output
    /// digest must already be on record.
    pub fn complete_task(
        &mut self,
        task_key: &str,
        tree: Option<&str>,
        commit: Option<&str>,
    ) -> Result<(), String> {
        let child_id = match self.states.get(task_key) {
            Some(TaskLifecycle::Reported { child_id }) => child_id.clone(),
            _ => return Err("task committed without its reported output".to_string()),
        };
        let mut detail = String::new();
        if let Some(tree) = tree {
            if !preparation::valid_commit(tree) {
                return Err("task commit evidence lacks its identity".to_string());
            }
            detail.push_str("tree=");
            detail.push_str(tree);
        }
        if let Some(commit) = commit {
            if !preparation::valid_commit(commit) {
                return Err("task commit evidence lacks its identity".to_string());
            }
            if !detail.is_empty() {
                detail.push(' ');
            }
            detail.push_str("commit=");
            detail.push_str(commit);
        }
        let committed = self.next_event(TASK_EVENT_TASK_COMMITTED, task_key, &child_id, &detail);
        self.record(&committed)?;
        self.states
            .insert(task_key.to_string(), TaskLifecycle::Completed);
        Ok(())
    }

    /// Record one terminal task failure. Remaining children stay owned
    /// until the caller stops the run; only the failed task reaches its
    /// terminal receipt here.
    pub fn fail_task(&mut self, task_key: &str, reason: &str) -> Result<(), String> {
        let child_id = match self.states.get(task_key) {
            Some(TaskLifecycle::Running { child_id })
            | Some(TaskLifecycle::Reported { child_id }) => child_id.clone(),
            _ => return Err("task failed without its running child".to_string()),
        };
        let failed = self.next_event(
            TASK_EVENT_TASK_FAILED,
            task_key,
            &child_id,
            &truncate_reason(reason),
        );
        self.record(&failed)?;
        self.states
            .insert(task_key.to_string(), TaskLifecycle::Failed);
        Ok(())
    }

    /// Retire every child for parent stop, then record the terminal event.
    /// No child may outlive confirmed retirement; the journal admits
    /// nothing after this.
    pub fn stop(
        &mut self,
        task_key: &str,
        reason: &str,
    ) -> Result<crate::terminal::factory::children::ChildStopReport, String> {
        if !crate::factory::valid_task_key(task_key) {
            return Err("task event task key is invalid".to_string());
        }
        let report = self.table.stop_all();
        self.states.retain(|_, state| {
            if state.active() {
                *state = TaskLifecycle::Cancelled;
            }
            true
        });
        let stopped = self.next_event(
            TASK_EVENT_RUN_STOPPED,
            task_key,
            "",
            &truncate_reason(reason),
        );
        self.record(&stopped)?;
        Ok(report)
    }

    /// Live native children awaiting retirement after a failure.
    pub fn active_tasks(&self) -> Vec<String> {
        let mut active: Vec<String> = self
            .states
            .iter()
            .filter(|(_, state)| state.active())
            .map(|(key, _)| key.clone())
            .collect();
        active.sort();
        active
    }

    /// Currently live children at the native boundary.
    pub fn active_child_count(&mut self) -> usize {
        self.table.active_count()
    }

    /// Journal snapshot for daemon persistence and the read path. The
    /// daemon stores these bytes atomically beside its private receipts;
    /// this runner never touches the live journal file itself.
    pub fn journal_snapshot(&self) -> Vec<TaskEvent> {
        self.journal.events().to_vec()
    }

    /// Join one child with a bounded wait, like the table.
    pub fn join_task(&mut self, task_key: &str, timeout: Duration) -> Result<i32, String> {
        self.table.join(task_key, timeout)
    }

    /// Cancel one child, like the table.
    pub fn cancel_task(&mut self, task_key: &str) -> Result<Option<i32>, String> {
        self.table.cancel(task_key)
    }
}
