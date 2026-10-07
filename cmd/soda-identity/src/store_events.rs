// Immutable audit events, extracted from store.rs (A05.M).
use super::store::{schema_integer, Store, Tx};
use crate::pg::pg_error;
use crate::wire::{Error, Event, Lease, UnixTime};
use tokio_postgres::types::{Json, ToSql};

impl Store {
    pub fn events(&self, owner: i64, id: &str) -> Result<Vec<Event>, Error> {
        if owner <= 0 {
            return Err(Error::denied("identity authority denied"));
        }
        let (rows, _) = self.query(
            "SELECT id,data FROM identity_events WHERE owner_id=$1 AND connection_id=$2 ORDER BY id DESC LIMIT 200",
            &[
                &schema_integer(owner)? as &(dyn ToSql + Sync),
                &id as &(dyn ToSql + Sync),
            ],
        )?;
        let mut out = Vec::with_capacity(rows.len());
        for row in &rows {
            let id = row.try_get::<_, i64>(0).map_err(pg_error)?;
            let mut event = row.try_get::<_, Json<Event>>(1).map_err(pg_error)?.0;
            event.id = id;
            out.push(event);
        }
        Ok(out)
    }
}

impl<'a> Tx<'a> {
    pub(crate) fn append_event(&self, event: &Event) -> Result<(), Error> {
        let mut event = event.clone();
        if event.owner_id == 0 {
            let row = self.query_row(
                "SELECT owner_id,generation FROM identity_connections WHERE id=$1",
                &[&event.connection_id as &(dyn ToSql + Sync)],
            )?;
            event.owner_id = i64::from(row.try_get::<_, i32>(0).map_err(pg_error)?);
            if event.generation == 0 {
                event.generation = i64::from(row.try_get::<_, i32>(1).map_err(pg_error)?);
            }
        }
        if event.actor_id == 0 {
            event.actor_id = event.owner_id;
        }
        event.time = UnixTime::now();
        self.exec(
            "INSERT INTO identity_events(owner_id,connection_id,data) VALUES($1,$2,$3)",
            &[
                &schema_integer(event.owner_id)? as &(dyn ToSql + Sync),
                &event.connection_id as &(dyn ToSql + Sync),
                &Json(&event) as &(dyn ToSql + Sync),
            ],
        )?;
        Ok(())
    }
}

pub(crate) fn lease_event(lease: &Lease, action: &str) -> Event {
    Event {
        id: 0,
        time: UnixTime::now(),
        action: action.to_string(),
        owner_id: 0,
        actor_id: lease.actor_id,
        connection_id: lease.connection_id.clone(),
        lease_id: lease.id.clone(),
        project_id: lease.project_id.clone(),
        grant_id: lease.grant_id.clone(),
        execution_id: lease.execution_id.clone(),
        kind: lease.kind.clone(),
        generation: lease.generation,
    }
}
