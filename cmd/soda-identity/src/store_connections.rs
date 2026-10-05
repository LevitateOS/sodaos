// Connection custody, extracted from store.rs (A05.M).
use super::store::{changed, identity_binding, Param, Store};
use crate::wire::{credential_valid, provider_valid, Connection, Error, Event, UnixTime};

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
        let data = serde_json::to_string(connection)?;
        let sealed = self.grants()?.seal(
            credential,
            &identity_binding(&connection.id, connection.generation),
        );
        self.transaction(|tx| {
            tx.exec(
                "INSERT INTO identity_connections(id,owner_id,generation,state,data,credential) VALUES(?,?,?,?,?,?)",
                &[
                    Param::text(&connection.id),
                    Param::int(connection.owner_id),
                    Param::int(connection.generation),
                    Param::text(&connection.state),
                    Param::text(&data),
                    Param::bytea(&sealed),
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
            "SELECT data FROM identity_connections WHERE id=?",
            &[Param::text(id)],
        )?;
        Ok(serde_json::from_str(row.text(0)?)?)
    }

    pub fn credential(&self, connection: &Connection) -> Result<Vec<u8>, Error> {
        let row = self.query_row(
            "SELECT credential FROM identity_connections WHERE id=? AND generation=? AND state='ready'",
            &[Param::text(&connection.id), Param::int(connection.generation)],
        )?;
        self.grants()?.open(
            &row.bytea(0)?,
            &identity_binding(&connection.id, connection.generation),
        )
    }

    pub fn connections(&self, owner: i64) -> Result<Vec<Connection>, Error> {
        let (rows, _) = self.query(
            "SELECT data FROM identity_connections WHERE owner_id=? ORDER BY id",
            &[Param::int(owner)],
        )?;
        rows.iter()
            .map(|r| Ok(serde_json::from_str(r.text(0)?)?))
            .collect()
    }

    pub fn available(&self, actor: i64, project: &str) -> Result<Vec<Connection>, Error> {
        let (rows, _) = self.query(
            "SELECT c.data FROM identity_connections c WHERE c.state='ready' AND (c.owner_id=? OR EXISTS(SELECT 1 FROM identity_grants g WHERE g.connection_id=c.id AND g.user_id=? AND g.project_id=? AND NOT g.revoked)) ORDER BY c.id",
            &[Param::int(actor), Param::int(actor), Param::text(project)],
        )?;
        let mut out = Vec::with_capacity(rows.len());
        for row in &rows {
            let mut connection: Connection = serde_json::from_str(row.text(0)?)?;
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
        let data = serde_json::to_string(&updated)?;
        self.transaction(|tx| {
            let count = tx.exec(
                "UPDATE identity_connections SET state=?,data=?,credential=CASE WHEN ?='revoked' THEN '\\x'::bytea ELSE credential END WHERE id=? AND generation=?",
                &[
                    Param::text(state),
                    Param::text(&data),
                    Param::text(state),
                    Param::text(&connection.id),
                    Param::int(connection.generation),
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
