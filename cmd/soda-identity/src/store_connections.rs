// Connection custody, extracted from store.rs (A05.M).
use super::store::{changed, identity_binding, schema_integer, Store};
use crate::pg::pg_error;
use crate::wire::{credential_valid, provider_valid, Connection, Error, Event, UnixTime};
use tokio_postgres::types::Json;
use tokio_postgres::types::ToSql;

impl Store {
    pub fn save_connection(&self, connection: &Connection, credential: &[u8]) -> Result<(), Error> {
        if self.grants.is_none() {
            return Err(Error::internal(
                "identity credential encryption key missing or incorrect",
            ));
        }
        if !provider_valid(&connection.provider_id)
            || !credential_valid(credential)
            || connection.owner_id <= 0
            || connection.id.is_empty()
            || connection.generation <= 0
        {
            return Err(Error::denied("identity authority denied"));
        }
        let sealed = self.grants()?.seal(
            credential,
            &identity_binding(&connection.id, connection.generation),
        );
        self.transaction(|tx| {
            tx.exec(
                "INSERT INTO identity_connections(id,owner_id,generation,state,data,credential) VALUES($1,$2,$3,$4,$5,$6)",
                &[
                    &connection.id as &(dyn ToSql + Sync),
                    &schema_integer(connection.owner_id)? as &(dyn ToSql + Sync),
                    &schema_integer(connection.generation)? as &(dyn ToSql + Sync),
                    &connection.state as &(dyn ToSql + Sync),
                    &Json(connection) as &(dyn ToSql + Sync),
                    &sealed as &(dyn ToSql + Sync),
                ],
            )?;
            tx.append_event(&Event {
                id: 0,
                time: UnixTime::now(),
                action: "connected".to_string(),
                owner_id: connection.owner_id,
                actor_id: connection.owner_id,
                connection_id: connection.id.clone(),
                lease_id: String::new(),
                project_id: String::new(),
                grant_id: String::new(),
                execution_id: String::new(),
                kind: String::new(),
                generation: connection.generation,
            })
        })
    }

    pub fn connection(&self, id: &str) -> Result<Connection, Error> {
        let row = self.query_row(
            "SELECT data FROM identity_connections WHERE id=$1",
            &[&id as &(dyn ToSql + Sync)],
        )?;
        Ok(row.try_get::<_, Json<Connection>>(0).map_err(pg_error)?.0)
    }

    pub fn credential(&self, connection: &Connection) -> Result<Vec<u8>, Error> {
        let row = self.query_row(
            "SELECT credential FROM identity_connections WHERE id=$1 AND generation=$2 AND state='ready'",
            &[
                &connection.id as &(dyn ToSql + Sync),
                &schema_integer(connection.generation)? as &(dyn ToSql + Sync),
            ],
        )?;
        self.grants()?.open(
            row.try_get::<_, Vec<u8>>(0).map_err(pg_error)?.as_slice(),
            &identity_binding(&connection.id, connection.generation),
        )
    }

    pub fn connections(&self, owner: i64) -> Result<Vec<Connection>, Error> {
        let (rows, _) = self.query(
            "SELECT data FROM identity_connections WHERE owner_id=$1 ORDER BY id",
            &[&schema_integer(owner)? as &(dyn ToSql + Sync)],
        )?;
        rows.iter()
            .map(|r| Ok(r.try_get::<_, Json<Connection>>(0).map_err(pg_error)?.0))
            .collect()
    }

    pub fn available(&self, actor: i64, project: &str) -> Result<Vec<Connection>, Error> {
        let (rows, _) = self.query(
            "SELECT c.data FROM identity_connections c WHERE c.state='ready' AND (c.owner_id=$1 OR EXISTS(SELECT 1 FROM identity_grants g WHERE g.connection_id=c.id AND g.user_id=$2 AND g.project_id=$3 AND NOT g.revoked)) ORDER BY c.id",
            &[
                &schema_integer(actor)? as &(dyn ToSql + Sync),
                &schema_integer(actor)? as &(dyn ToSql + Sync),
                &project as &(dyn ToSql + Sync),
            ],
        )?;
        let mut out = Vec::with_capacity(rows.len());
        for row in &rows {
            let mut connection = row.try_get::<_, Json<Connection>>(0).map_err(pg_error)?.0;
            if connection.owner_id != actor {
                connection.email.clear();
            }
            out.push(connection);
        }
        Ok(out)
    }

    // IdentityState denies admission before callers attempt native termination.
    pub fn set_state(&self, connection: &Connection, state: &str) -> Result<(), Error> {
        let mut updated = connection.clone();
        updated.state = state.to_string();
        self.transaction(|tx| {
            let count = tx.exec(
                "UPDATE identity_connections SET state=$1,data=$2,credential=CASE WHEN $3='revoked' THEN '\\x'::bytea ELSE credential END WHERE id=$4 AND generation=$5",
                &[
                    &state as &(dyn ToSql + Sync),
                    &Json(&updated) as &(dyn ToSql + Sync),
                    &state as &(dyn ToSql + Sync),
                    &connection.id as &(dyn ToSql + Sync),
                    &schema_integer(connection.generation)? as &(dyn ToSql + Sync),
                ],
            )?;
            changed(count)?;
            tx.append_event(&Event {
                id: 0,
                time: UnixTime::now(),
                action: state.to_string(),
                owner_id: connection.owner_id,
                actor_id: connection.owner_id,
                connection_id: connection.id.clone(),
                lease_id: String::new(),
                project_id: String::new(),
                grant_id: String::new(),
                execution_id: String::new(),
                kind: String::new(),
                generation: connection.generation,
            })
        })
    }
}
