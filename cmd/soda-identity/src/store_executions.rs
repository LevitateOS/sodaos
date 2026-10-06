// Execution fences, extracted from store.rs (A05.M).
use super::store::{Param, Store};
use crate::wire::{Error, Execution};

impl Store {
    // IdentityExecution returns the retained acquisition identity for one
    // (kind, execution_id). The record outlives lease return/deletion.
    pub fn execution(&self, kind: &str, execution_id: &str) -> Result<Execution, Error> {
        let row = self.query_row(
            "SELECT data FROM identity_executions WHERE kind=$1 AND execution_id=$2",
            &[Param::text(kind), Param::text(execution_id)],
        )?;
        Ok(serde_json::from_str(row.text(0)?)?)
    }

    // IdentityAdmitExecution records one execution identity. Repeating the
    // same identity returns the recorded row; callers compare digests to
    // detect a changed request for that ID.
    pub fn admit_execution(&self, execution: &Execution) -> Result<(Execution, bool), Error> {
        execution.validate()?;
        let data = serde_json::to_string(execution)?;
        let count = self.exec(
            "INSERT INTO identity_executions(kind,execution_id,state,lease_id,data) VALUES($1,$2,$3,$4,$5)\n\t\tON CONFLICT(kind,execution_id) DO NOTHING",
            &[
                Param::text(&execution.kind),
                Param::text(&execution.execution_id),
                Param::text(&execution.state),
                Param::text(&execution.lease_id),
                Param::text(&data),
            ],
        )?;
        if count == 1 {
            return Ok((execution.clone(), true));
        }
        Ok((
            self.execution(&execution.kind, &execution.execution_id)?,
            false,
        ))
    }

    // IdentityObserveExecution advances an execution's state, lease and
    // binding. The acquisition digest is additionally guarded by a schema
    // trigger, and a terminal execution can never leave that state.
    pub fn observe_execution(&self, execution: &Execution) -> Result<(), Error> {
        execution.validate()?;
        let data = serde_json::to_string(execution)?;
        let count = self.exec(
            "UPDATE identity_executions SET state=$1,lease_id=$2,data=$3 WHERE kind=$4 AND execution_id=$5",
            &[
                Param::text(&execution.state),
                Param::text(&execution.lease_id),
                Param::text(&data),
                Param::text(&execution.kind),
                Param::text(&execution.execution_id),
            ],
        )?;
        if count != 1 {
            return Err(Error::not_found());
        }
        Ok(())
    }
}
