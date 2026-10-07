// Connection custody, extracted from store.rs (A05.M).
use super::store::{changed, identity_binding, schema_integer, Store};
use crate::pg::pg_error;
use crate::wire::{credential_valid, provider_valid, Connection, Error, Event, UnixTime};
use tokio_postgres::types::Json;
use tokio_postgres::types::ToSql;

// This is a query batch size, not a population limit. JSONB rows are screened
// before the application decodes them, then the complete public list is
// admitted against the existing 512 KiB response budget.
pub(crate) const LIST_PAGE_ROWS: i64 = 64;
pub(crate) const LIST_RECORD_BYTES: i32 = 512 << 10;
// success_response appends the final LF, so raw list JSON may use at most
// 512 KiB minus that byte.
pub(crate) const LIST_JSON_BYTES: usize = (512 << 10) - 1;

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

    pub fn connections(&self, owner: i64) -> Result<Vec<u8>, Error> {
        let owner = schema_integer(owner)?;
        let mut output = ListJson::new();
        let mut after: Option<String> = None;
        loop {
            let (rows, _) = self.query(
                "SELECT CASE WHEN octet_length(id)<=$3 THEN id ELSE NULL END, owner_id, state, CASE WHEN octet_length(id)<=$3 AND octet_length(data::text)<=$4 AND data->>'id'=id AND data->>'owner_id'=owner_id::text AND data->>'state'=state THEN data ELSE NULL END FROM identity_connections WHERE owner_id=$1 AND ($2::text IS NULL OR id>$2) ORDER BY id LIMIT $5",
                &[
                    &owner as &(dyn ToSql + Sync),
                    &after as &(dyn ToSql + Sync),
                    &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                    &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                    &LIST_PAGE_ROWS as &(dyn ToSql + Sync),
                ],
            )?;
            if rows.is_empty() { break; }
            for row in &rows {
                let id = row.try_get::<_, Option<String>>(0).map_err(pg_error)?
                    .ok_or_else(|| Error::internal("identity list incomplete"))?;
                let owner = row.try_get::<_, i32>(1).map_err(pg_error)?;
                let state = row.try_get::<_, String>(2).map_err(pg_error)?;
                let value = row.try_get::<_, Option<Json<Connection>>>(3).map_err(pg_error)?
                    .ok_or_else(|| Error::internal("identity list incomplete"))?;
                if value.0.id != id || value.0.owner_id != owner as i64 || value.0.state != state {
                    return Err(Error::internal("identity list incomplete"));
                }
                output.push(&value.0)?;
                after = Some(id);
            }
            if rows.len() < LIST_PAGE_ROWS as usize { break; }
        }
        output.finish()
    }

    pub fn available(&self, actor: i64, project: &str) -> Result<Vec<u8>, Error> {
        let actor = schema_integer(actor)?;
        let mut output = ListJson::new();
        let mut after: Option<String> = None;
        loop {
            let (rows, _) = self.query(
                "SELECT CASE WHEN octet_length(c.id)<=$4 THEN c.id ELSE NULL END, c.owner_id, c.state, CASE WHEN octet_length(c.id)<=$4 AND octet_length(c.data::text)<=$5 AND c.data->>'id'=c.id AND c.data->>'owner_id'=c.owner_id::text AND c.data->>'state'=c.state THEN c.data ELSE NULL END FROM identity_connections c WHERE c.state='ready' AND (c.owner_id=$1 OR EXISTS(SELECT 1 FROM identity_grants g WHERE g.connection_id=c.id AND g.user_id=$2 AND g.project_id=$3 AND NOT g.revoked)) AND ($6::text IS NULL OR c.id>$6) ORDER BY c.id LIMIT $7",
                &[
                    &actor as &(dyn ToSql + Sync),
                    &actor as &(dyn ToSql + Sync),
                    &project as &(dyn ToSql + Sync),
                    &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                    &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                    &after as &(dyn ToSql + Sync),
                    &LIST_PAGE_ROWS as &(dyn ToSql + Sync),
                ],
            )?;
            if rows.is_empty() { break; }
            for row in &rows {
                let id = row.try_get::<_, Option<String>>(0).map_err(pg_error)?
                    .ok_or_else(|| Error::internal("identity list incomplete"))?;
                let owner = row.try_get::<_, i32>(1).map_err(pg_error)?;
                let state = row.try_get::<_, String>(2).map_err(pg_error)?;
                let value = row.try_get::<_, Option<Json<Connection>>>(3).map_err(pg_error)?
                    .ok_or_else(|| Error::internal("identity list incomplete"))?;
                if value.0.id != id || value.0.owner_id != owner as i64 || value.0.state != state {
                    return Err(Error::internal("identity list incomplete"));
                }
                let mut connection = value.0;
                if owner != actor { connection.email.clear(); }
                output.push(&connection)?;
                after = Some(id);
            }
            if rows.len() < LIST_PAGE_ROWS as usize { break; }
        }
        output.finish()
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

pub(crate) struct ListJson { bytes: Vec<u8>, first: bool }

impl ListJson {
    pub(crate) fn new() -> Self { Self { bytes: vec![b'['], first: true } }
    pub(crate) fn push<T: serde::Serialize>(&mut self, value: &T) -> Result<(), Error> {
        if !self.first { self.write_chunk(b",")?; }
        serde_json::to_writer(BoundedListWriter { bytes: &mut self.bytes, limit: LIST_JSON_BYTES - 1 }, value)
            .map_err(|_| Error::internal("identity list exceeds response limit"))?;
        self.first = false;
        Ok(())
    }
    pub(crate) fn finish(mut self) -> Result<Vec<u8>, Error> {
        if self.bytes.len() >= LIST_JSON_BYTES { return Err(Error::internal("identity list exceeds response limit")); }
        self.bytes.push(b']');
        Ok(self.bytes)
    }
    fn write_chunk(&mut self, chunk: &[u8]) -> Result<(), Error> {
        let limit = LIST_JSON_BYTES - 1;
        if self.bytes.len().saturating_add(chunk.len()) > limit {
            return Err(Error::internal("identity list exceeds response limit"));
        }
        self.bytes.extend_from_slice(chunk);
        Ok(())
    }
}

struct BoundedListWriter<'a> { bytes: &'a mut Vec<u8>, limit: usize }
impl std::io::Write for BoundedListWriter<'_> {
    fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
        if self.bytes.len().saturating_add(bytes.len()) > self.limit {
            return Err(std::io::Error::new(std::io::ErrorKind::WriteZero, "list limit"));
        }
        self.bytes.extend_from_slice(bytes);
        Ok(bytes.len())
    }
    fn flush(&mut self) -> std::io::Result<()> { Ok(()) }
}

#[cfg(test)]
mod list_tests {
    use super::{ListJson, LIST_JSON_BYTES};

    #[test]
    fn bounded_list_accepts_exact_json_limit_and_refuses_one_more_byte() {
        let exact_value = "x".repeat(LIST_JSON_BYTES - 4);
        let mut exact = ListJson::new();
        exact.push(&exact_value).unwrap();
        assert_eq!(exact.finish().unwrap().len(), LIST_JSON_BYTES);

        let over_value = "x".repeat(LIST_JSON_BYTES - 3);
        let mut over = ListJson::new();
        assert!(over.push(&over_value).is_err());
    }
}
