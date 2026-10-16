//! Typed task-event journal storage and read path (D05 / FW13).
//!
//! The journal lives beside the existing private host receipts
//! (`{state_dir}/{project}-{run}.task-events.json`), written through a
//! 0600 temporary sibling and atomic rename under the existing run lock,
//! exactly like run receipts. Reads are free of checkpoint mutations.

use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::time::Instant;

use crate::project::Executor;

use super::events::{
    EventJournal, FactoryTaskEvents, FactoryTaskEventsState, JournalFile, TaskEvent,
    MAX_TASK_JOURNAL_BYTES,
};
use super::run::execution_manifest_required;
use super::{
    Factory, FactoryBroker, FactoryError, FactoryTerminal, RunLock, FACTORY_STOPPED,
    FACTORY_UNCERTAIN,
};

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    fn journal_path(&self, project: &str, run: &str) -> String {
        format!("{}/{project}-{run}.task-events.json", self.state_dir)
    }

    /// Load and re-verify one journal. Absent reads succeed empty; corrupt
    /// journals fence the run instead of serving partial history.
    pub(in crate::factory) fn load_journal(
        &self,
        project: &str,
        run: &str,
    ) -> Result<EventJournal, FactoryError> {
        let data = match std::fs::read(self.journal_path(project, run)) {
            Ok(data) => data,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                return Ok(EventJournal::default())
            }
            Err(e) => return Err(FactoryError::msg(e.to_string())),
        };
        if data.len() > MAX_TASK_JOURNAL_BYTES + 4096 {
            return Err(FactoryError::msg("task journal exceeds bounds"));
        }
        let file =
            JournalFile::decode(&data).map_err(|_| FactoryError::msg("invalid task journal"))?;
        let mut journal = EventJournal::default();
        for event in &file.events {
            journal
                .append(event)
                .map_err(|_| FactoryError::msg("invalid task journal"))?;
        }
        Ok(journal)
    }

    pub(in crate::factory) fn store_journal(
        &self,
        project: &str,
        run: &str,
        events: &[TaskEvent],
    ) -> Result<(), FactoryError> {
        use std::io::Write;
        let data = JournalFile {
            events: events.to_vec(),
        }
        .encode();
        let path = self.journal_path(project, run);
        let mut counter = 0u32;
        let tmp_path = loop {
            counter += 1;
            let candidate = format!(
                "{}/.task-events-{}-{}-{counter}",
                self.state_dir,
                std::process::id(),
                std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .map(|d| d.as_nanos())
                    .unwrap_or(0)
            );
            match std::fs::OpenOptions::new()
                .create_new(true)
                .write(true)
                .mode(0o600)
                .open(&candidate)
            {
                Ok(_) => break candidate,
                Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists && counter < 100 => {
                    continue
                }
                Err(e) => return Err(FactoryError::msg(e.to_string())),
            }
        };
        let write_result = (|| -> std::io::Result<()> {
            let mut tmp = std::fs::OpenOptions::new().write(true).open(&tmp_path)?;
            tmp.write_all(data.as_bytes())?;
            tmp.set_permissions(std::fs::Permissions::from_mode(0o600))?;
            drop(tmp);
            std::fs::rename(&tmp_path, &path)?;
            Ok(())
        })();
        if let Err(e) = write_result {
            let _ = std::fs::remove_file(&tmp_path);
            return Err(FactoryError::msg(e.to_string()));
        }
        Ok(())
    }

    /// Record one typed child observation under the run lock. The run must
    /// exist, carry an implement manifest, and not be stopped; the event
    /// must bind the run's live native invocation and manifest digest, so
    /// agent-written bytes can never forge accepted events. An exact
    /// duplicate of the journal head is a no-op: a lost reply is looked
    /// up, not replayed.
    pub fn record_task_event(
        &self,
        project: &str,
        run: &str,
        event: &TaskEvent,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        let _lock: RunLock = self.lock_run(project, run, deadline)?;
        let (receipt, exists) = self.load_receipt(project, run)?;
        if !exists {
            return Err(FactoryError::NotFound);
        }
        if matches!(receipt.phase.as_str(), FACTORY_STOPPED | FACTORY_UNCERTAIN) {
            return Err(FactoryError::msg("task run is retired"));
        }
        if !execution_manifest_required(&receipt.run.execution_phase) {
            return Err(FactoryError::msg(
                "non-implement run records no task events",
            ));
        }
        if event.plan_digest != receipt.run.manifest_digest {
            return Err(FactoryError::msg("task event plan digest refuses"));
        }
        match &receipt.binding {
            Some(binding) if event.invocation == binding.invocation_id => {}
            _ => return Err(FactoryError::msg("task event invocation refuses")),
        }
        let mut journal = self.load_journal(project, run)?;
        journal.append(event).map_err(FactoryError::msg)?;
        let events = journal.events().to_vec();
        self.store_journal(project, run, &events)
    }

    /// `FactoryTaskEvents`: page one admitted run's typed journal. The run
    /// must exist; an absent journal reads empty. A response too small for
    /// its next event is a bounded size error without advancing.
    pub fn task_events(
        &self,
        req: &FactoryTaskEvents,
        deadline: Instant,
    ) -> Result<FactoryTaskEventsState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let _lock: RunLock = self.lock_run(&req.project, &req.id, deadline)?;
        let (_, exists) = self.load_receipt(&req.project, &req.id)?;
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let journal = self.load_journal(&req.project, &req.id)?;
        let (events, next_sequence, terminal) = journal
            .query(req.after_sequence, req.byte_limit as usize)
            .map_err(FactoryError::msg)?;
        let state = FactoryTaskEventsState {
            id: req.id.clone(),
            project: req.project.clone(),
            next_sequence,
            terminal,
            events,
        };
        state.validate().map_err(FactoryError::msg)?;
        Ok(state)
    }
}
