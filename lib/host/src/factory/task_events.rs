//! Typed task-event journal storage and read path (D05 / FW13).
//!
//! The journal lives beside the existing private host receipts
//! (`{state_dir}/{project}-{run}.task-events.json`), written through a
//! 0600 temporary sibling and atomic rename under the existing run lock,
//! exactly like run receipts. Reads are free of checkpoint mutations.

use std::time::Instant;

use crate::project::Executor;

use super::events::{
    EventJournal, FactoryTaskEvents, FactoryTaskEventsState, JournalFile, TaskEvent,
};
use super::{Factory, FactoryBroker, FactoryError, FactoryTerminal, RunLock};

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    /// Load and re-verify one journal. Absent reads succeed empty; corrupt
    /// journals fence the run instead of serving partial history.
    pub(in crate::factory) fn load_journal(
        &self,
        project: &str,
        run: &str,
    ) -> Result<EventJournal, FactoryError> {
        let state =
            super::checkpoint::PrivateState::open(&self.state_dir).map_err(FactoryError::msg)?;
        let file = match state
            .read_bytes(&format!("{project}-{run}.task-events.json"))
            .map_err(FactoryError::msg)?
        {
            Some(bytes) => JournalFile::decode(&bytes).map_err(FactoryError::msg)?,
            None => JournalFile { events: Vec::new() },
        };
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
        let state =
            super::checkpoint::PrivateState::open(&self.state_dir).map_err(FactoryError::msg)?;
        let data = JournalFile {
            events: events.to_vec(),
        }
        .encode();
        state
            .write_bytes(
                &format!("{project}-{run}.task-events.json"),
                data.as_bytes(),
            )
            .map_err(FactoryError::msg)
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
