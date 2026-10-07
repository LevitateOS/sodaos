// Immutable audit events, extracted from store.rs (A05.M).
use super::store::{Param, Store, Tx};
use crate::wire::{Error, Event, Lease, UnixTime};

impl Store {
    pub fn events(&self, owner: i64, id: &str) -> Result<Vec<Event>, Error> {
        if owner <= 0 {
            return Err(Error::denied("identity authority denied"));
        }
        let (rows, _) = self.query(
            "SELECT id,data FROM identity_events WHERE owner_id=$1 AND connection_id=$2 ORDER BY id DESC LIMIT 200",
            &[Param::int(owner), Param::text(id)],
        )?;
        let mut out = Vec::with_capacity(rows.len());
        for row in &rows {
            let id = row.integer(0)?;
            let mut event: Event = serde_json::from_str(row.text(1)?)?;
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
                &[Param::text(&event.connection_id)],
            )?;
            event.owner_id = row.integer(0)?;
            if event.generation == 0 {
                event.generation = row.integer(1)?;
            }
        }
        if event.actor_id == 0 {
            event.actor_id = event.owner_id;
        }
        event.time = UnixTime::now();
        let data = serde_json::to_string(&event)?;
        self.exec(
            "INSERT INTO identity_events(owner_id,connection_id,data) VALUES($1,$2,$3)",
            &[
                Param::int(event.owner_id),
                Param::text(&event.connection_id),
                Param::json(&data),
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
