// Execution fences, extracted from store.rs (A05.M).
use super::store::Store;
use crate::pg::pg_error;
use crate::wire::{Error, Execution};
use tokio_postgres::types::{Json, ToSql};

impl Store {
    // IdentityExecution returns the retained acquisition identity for one
    // (kind, execution_id). The record outlives lease return/deletion.
    pub fn execution(&self, kind: &str, execution_id: &str) -> Result<Execution, Error> {
        let row = self.query_row(
            "SELECT data FROM identity_executions WHERE kind=$1 AND execution_id=$2",
            &[
                &kind as &(dyn ToSql + Sync),
                &execution_id as &(dyn ToSql + Sync),
            ],
        )?;
        Ok(row.try_get::<_, Json<Execution>>(0).map_err(pg_error)?.0)
    }

    // IdentityAdmitExecution records one execution identity. Repeating the
    // same identity returns the recorded row; callers compare digests to
    // detect a changed request for that ID.
    pub fn admit_execution(&self, execution: &Execution) -> Result<(Execution, bool), Error> {
        execution.validate()?;
        let count = self.exec(
            "INSERT INTO identity_executions(kind,execution_id,state,lease_id,data) VALUES($1,$2,$3,$4,$5)\n\t\tON CONFLICT(kind,execution_id) DO NOTHING",
            &[
                &execution.kind as &(dyn ToSql + Sync),
                &execution.execution_id as &(dyn ToSql + Sync),
                &execution.state as &(dyn ToSql + Sync),
                &execution.lease_id as &(dyn ToSql + Sync),
                &Json(execution) as &(dyn ToSql + Sync),
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
        let count = self.exec(
            "UPDATE identity_executions SET state=$1,lease_id=$2,data=$3 WHERE kind=$4 AND execution_id=$5",
            &[
                &execution.state as &(dyn ToSql + Sync),
                &execution.lease_id as &(dyn ToSql + Sync),
                &Json(execution) as &(dyn ToSql + Sync),
                &execution.kind as &(dyn ToSql + Sync),
                &execution.execution_id as &(dyn ToSql + Sync),
            ],
        )?;
        if count != 1 {
            return Err(Error::not_found());
        }
        Ok(())
    }
}
