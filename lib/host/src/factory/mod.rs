//! Supervised factory-run execution (PR26).
//!
//! Port of `internal/host/project/{factory,factory_export,factory_output}.go`
//! and the `Factory` half of `factory_candidate.go` (`InspectCandidate`),
//! plus the factory/prepare/candidate slice of the top-level facades in
//! `internal/host/{factory_client,factory_candidate,prepare}.go`.
//!
//! The daemon side is a [`Factory`] over three seams: an [`Executor`] for
//! the podman observation in candidate inspection, a [`FactoryTerminal`]
//! for the supervised native boundary (`internal/host/terminal`, ported
//! separately) and a [`FactoryBroker`] for the identity broker client.
//! Receipt bytes, state mapping, phase machine, reason strings and error
//! strings match the Go implementation; only transport failures (I/O error
//! text, never wire-visible: the daemon maps them to a generic 500) follow
//! Rust formatting.
//!
//! Deadline strings are RFC3339Nano as emitted by Go's `time.Time` JSON
//! encoding. They are stored and re-emitted verbatim, so receipt bytes stay
//! exact; a parsed epoch-nanos copy drives the deadline comparisons. The
//! validator accepts the canonical shape Go emits (`YYYY-MM-DDTHH:MM:SS`
//! with an optional 1-9 digit fraction and a `Z` or numeric offset); exotic
//! inputs Go's own parser might accept or reject beyond that shape are out
//! of scope.
//!
//! Later route layers map [`FactoryError`] variants to statuses exactly
//! like `daemon.go`: `NotFound` to 404, `Stale` to 409, `ExportCandidate`
//! to 422, `ExportBounds` to 413, everything else to a generic 500.

use std::os::unix::fs::PermissionsExt;
use std::time::Instant;

use crate::project::Executor;
use crate::terminal::Binding;

mod artifacts;
mod candidate;
mod confirmation;
mod deadline;
mod events;
mod finish;
mod identity;
mod inspect;
mod launch;
mod manifest;
mod receipt;
mod requests;
mod run;
mod state;
mod stop;
mod task_events;
mod tasks;
#[cfg(test)]
mod tests;

pub use self::candidate::CANDIDATE_INSPECT_SCRIPT;
pub use self::confirmation::{
    confirm_factory_candidate_inspect, confirm_factory_export, confirm_factory_harness,
    confirm_factory_inspect, confirm_factory_launch, confirm_factory_output, confirm_factory_stop,
    confirm_factory_takeover, confirm_factory_task_events, confirm_hold_preparation,
    confirm_inspect_preparation, confirm_prepare, confirm_prepare_candidate,
    confirm_stop_preparation, factory_candidate_status_error, factory_export_status_error,
    factory_not_found_status, factory_output_status_error,
};
pub use self::events::{
    task_event_terminal_type, valid_task_event_type, valid_task_key, EventJournal,
    FactoryTaskEvents, FactoryTaskEventsState, TaskEvent, MAX_TASK_EVENTS_BYTES,
    MAX_TASK_EVENTS_PER_READ, MAX_TASK_EVENT_BYTES, MAX_TASK_JOURNAL_BYTES,
    MAX_TASK_JOURNAL_EVENTS, MIN_TASK_EVENTS_BYTES, TASK_EVENT_CHECKPOINT_PREPARED,
    TASK_EVENT_CHILD_REPORTED, TASK_EVENT_CHILD_STARTED, TASK_EVENT_CHILD_START_INTENT,
    TASK_EVENT_RUN_STOPPED, TASK_EVENT_TASK_COMMITTED, TASK_EVENT_TASK_FAILED,
    TASK_JOURNAL_RESERVED_TERMINAL,
};
pub use self::identity::{AcquireRequest, Lease};
pub use self::manifest::{valid_task_effort, ExecutionManifest};
pub use self::requests::{
    FactoryCandidateInspect, FactoryCandidateState, FactoryExport, FactoryExportState,
    FactoryInspect, FactoryOutput, FactoryOutputState, FactoryStop, FactoryTakeover,
    TakeoverResult,
};
pub use self::run::{
    execution_manifest_required, factory_run_paths, factory_unit_name, max_prompt_for_phase,
    takeover_destination, takeover_source, valid_execution_phase, valid_factory_phase,
    valid_factory_run_id, valid_harness_family, valid_harness_version, FactoryError, FactoryLaunch,
    FactoryRun, DEFAULT_TASK_CHILDREN, ERR_RUN_NOT_FOUND, ERR_RUN_STALE, EXECUTION_CORRECT,
    EXECUTION_DECOMPOSE, EXECUTION_IMPLEMENT, EXECUTION_INVESTIGATE, EXECUTION_REASSESS,
    EXECUTION_REVIEW, EXECUTION_TERMINAL, FACTORY_APPROVED, FACTORY_COMPLETED, FACTORY_FAILED,
    FACTORY_HARNESS_CODEX, FACTORY_HARNESS_MUSE, FACTORY_RUNNING, FACTORY_STATE_ROOT,
    FACTORY_STOPPED, FACTORY_UNCERTAIN, IDENTITY_FACTORY, MAX_EXECUTION_MANIFEST,
    MAX_FACTORY_EXPORT_BUNDLE, MAX_FACTORY_OUTPUT, MAX_FACTORY_OUTPUT_OFFSET,
    MAX_FACTORY_OUTPUT_READ, MAX_FACTORY_OUTPUT_WINDOW, MAX_FACTORY_PROMPT, MAX_REASSESS_PROMPT,
    MAX_TASK_CHILDREN, MIN_TASK_CHILDREN,
};
pub use self::state::{FactoryHarnessPin, FactoryState, OutputSlice};
pub use self::tasks::TaskRunner;

// ---------- supervised-boundary seams ----------

/// Supervised native boundary: the `terminal.Service` methods the factory
/// orchestrator calls. Ported with the terminal lane; tests script fakes.
pub trait FactoryTerminal {
    fn harness_pin(&self, family: &str) -> Result<FactoryHarnessPin, String>;
    fn reserve(
        &self,
        run: &FactoryRun,
        lease: &Lease,
        pin: &str,
        max_secs: i64,
        deadline: Instant,
    ) -> Result<Binding, FactoryError>;
    fn start(
        &self,
        lease: &Lease,
        credential: &[u8],
        prompt: &[u8],
        deadline: Instant,
    ) -> Result<(), FactoryError>;
    fn wait(&self, lease: &Lease, deadline: Instant) -> Result<(i64, String), FactoryError>;
    fn stop(&self, lease: &Lease, deadline: Instant) -> Result<(), FactoryError>;
    fn stop_unbound(&self, run: &FactoryRun, deadline: Instant) -> Result<(), FactoryError>;
    fn capture(&self, lease: &Lease, deadline: Instant) -> Result<Vec<u8>, FactoryError>;
    fn live(&self, binding: &Binding, deadline: Instant) -> bool;
    fn output(
        &self,
        project: &str,
        binding: &Binding,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<OutputSlice, FactoryError>;
    #[allow(clippy::too_many_arguments)]
    fn takeover_copy(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        member: &str,
        run: &str,
        deadline: Instant,
    ) -> Result<(String, bool), FactoryError>;
    fn export_bundle(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        candidate: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, FactoryError>;
}

/// Identity broker: the `identityclient.Client` methods the factory
/// orchestrator calls. Reconciliation failures are ignored by every
/// caller, exactly like the Go `_ =` assignments, so reconcile reports
/// nothing.
pub trait FactoryBroker {
    fn acquire(&self, req: &AcquireRequest, deadline: Instant) -> Result<Lease, FactoryError>;
    fn register(
        &self,
        lease_id: &str,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<Vec<u8>, FactoryError>;
    fn return_lease(
        &self,
        lease_id: &str,
        binding: &Binding,
        credential: &[u8],
        deadline: Instant,
    ) -> Result<(), FactoryError>;
    fn reconcile_lease(&self, lease_id: &str, deadline: Instant);
    fn execution_is_terminal(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<bool, FactoryError>;
    fn close_execution(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<(), FactoryError>;
}

/// Credential bytes that are wiped when they fall out of scope, mirroring
/// Go's `defer clear(...)`.
pub(in crate::factory) struct Secret(Vec<u8>);

impl Drop for Secret {
    fn drop(&mut self) {
        for b in self.0.iter_mut() {
            // Volatile so the wipe survives optimization.
            unsafe { std::ptr::write_volatile(b, 0) };
        }
    }
}

/// Held per-run lock file. Closing releases the flock, exactly like Go.
pub(in crate::factory) struct RunLock {
    pub(in crate::factory) _file: std::fs::File,
}

// ---------- factory orchestrator ----------

/// Supervised factory-run orchestration over a private receipt directory.
/// Launch and stop never hold the per-run lock across broker or native
/// calls; takeover holds it across the copy to serialize duplicates.
pub struct Factory<E, T, B> {
    pub(in crate::factory) exec: E,
    pub(in crate::factory) terminal: T,
    pub(in crate::factory) broker: B,
    pub(in crate::factory) state_dir: String,
}

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    /// `OpenFactory`: wire run orchestration over a private receipt
    /// directory. The directory must already exist with private
    /// permissions. (Go also rejects nil dependencies; Rust's type
    /// system owns them, so there is no such error here.)
    pub fn open_factory(state_dir: &str, exec: E, terminal: T, broker: B) -> Result<Self, String> {
        if !state_dir.starts_with('/') {
            return Err("factory receipt directory must be absolute".to_string());
        }
        let info = std::fs::symlink_metadata(state_dir).map_err(|e| e.to_string())?;
        if !info.is_dir() || info.permissions().mode() & 0o777 & 0o077 != 0 {
            return Err("factory receipts must live in a private directory".to_string());
        }
        Ok(Factory {
            exec,
            terminal,
            broker,
            state_dir: state_dir.to_string(),
        })
    }

    /// `HarnessPin`: the staged-harness identity the executor admits.
    /// `image` stays empty here; the daemon route fills it from its
    /// configuration, exactly like the Go dispatch.
    pub fn harness_pin(&self, family: &str) -> Result<FactoryHarnessPin, String> {
        self.terminal.harness_pin(family)
    }
}
