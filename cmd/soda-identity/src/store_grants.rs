// Delegated grants, extracted from store.rs (A05.M).
use super::store::{changed, schema_integer, Store};
use crate::pg::pg_error;
use crate::wire::{Error, Event, Grant, UnixTime};
use tokio_postgres::types::{Json, ToSql};

impl Store {
    pub fn save_grant(&self, grant: &Grant) -> Result<(), Error> {
        self.transaction(|tx| {
            tx.exec(
                "INSERT INTO identity_grants(id,connection_id,user_id,project_id,revision,revoked,data) VALUES($1,$2,$3,$4,$5,$6,$7)",
                &[
                    &grant.id as &(dyn ToSql + Sync),
                    &grant.connection_id as &(dyn ToSql + Sync),
                    &schema_integer(grant.user_id)? as &(dyn ToSql + Sync),
                    &grant.project_id as &(dyn ToSql + Sync),
                    &schema_integer(grant.revision)? as &(dyn ToSql + Sync),
                    &grant.revoked as &(dyn ToSql + Sync),
                    &Json(grant) as &(dyn ToSql + Sync),
                ],
            )?;
            tx.append_event(&Event {
                id: 0,
                time: UnixTime::now(),
                action: "grant_created".to_string(),
                owner_id: 0,
                actor_id: 0,
                connection_id: grant.connection_id.clone(),
                lease_id: String::new(),
                project_id: grant.project_id.clone(),
                grant_id: grant.id.clone(),
                execution_id: String::new(),
                kind: String::new(),
                generation: 0,
            })
        })
    }

    pub fn grant(&self, id: &str) -> Result<Grant, Error> {
        let row = self.query_row(
            "SELECT data FROM identity_grants WHERE id=$1",
            &[&id as &(dyn ToSql + Sync)],
        )?;
        Ok(row.try_get::<_, Json<Grant>>(0).map_err(pg_error)?.0)
    }

    pub fn grants_for(&self, id: &str) -> Result<Vec<Grant>, Error> {
        let (rows, _) = self.query(
            "SELECT data FROM identity_grants WHERE connection_id=$1 ORDER BY id",
            &[&id as &(dyn ToSql + Sync)],
        )?;
        rows.iter()
            .map(|r| Ok(r.try_get::<_, Json<Grant>>(0).map_err(pg_error)?.0))
            .collect()
    }

    pub fn revoke_grant(&self, grant: &Grant) -> Result<(), Error> {
        let mut updated = grant.clone();
        updated.revoked = true;
        updated.revision += 1;
        let previous = updated.revision - 1;
        self.transaction(|tx| {
            let count = tx.exec(
                "UPDATE identity_grants SET revoked=TRUE,revision=$1,data=$2 WHERE id=$3 AND revision=$4",
                &[
                    &schema_integer(updated.revision)? as &(dyn ToSql + Sync),
                    &Json(&updated) as &(dyn ToSql + Sync),
                    &grant.id as &(dyn ToSql + Sync),
                    &schema_integer(previous)? as &(dyn ToSql + Sync),
                ],
            )?;
            changed(count)?;
            tx.append_event(&Event {
                id: 0,
                time: UnixTime::now(),
                action: "grant_revoked".to_string(),
                owner_id: 0,
                actor_id: 0,
                connection_id: grant.connection_id.clone(),
                lease_id: String::new(),
                project_id: grant.project_id.clone(),
                grant_id: grant.id.clone(),
                execution_id: String::new(),
                kind: String::new(),
                generation: 0,
            })
        })
    }
}
