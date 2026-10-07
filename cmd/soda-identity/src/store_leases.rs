// Leases, reservation and credential maintenance, extracted from store.rs (A05.M).
use super::store::{changed, identity_binding, schema_integer, Store, Tx};
use super::store_connections::{ListJson, LIST_PAGE_ROWS, LIST_RECORD_BYTES};
use super::store_events::lease_event;
use crate::pg::pg_error;
use crate::wire::{
    credential_valid, provider_valid, Connection, Error, Execution, Lease, EXECUTION_LIVE,
    EXECUTION_PENDING, EXECUTION_TERMINAL,
};
use tokio_postgres::types::{Json, ToSql};

impl Store {
    pub(crate) fn leases_page(&self, after: Option<&str>) -> Result<Vec<Lease>, Error> {
        let (rows, _) = self.query(
            "SELECT CASE WHEN octet_length(id)<=$2 THEN id ELSE NULL END, CASE WHEN octet_length(id)<=$2 AND octet_length(data::text)<=$3 AND jsonb_typeof(data->'connection_id')='string' AND data->>'connection_id'=connection_id THEN data ELSE NULL END FROM identity_leases WHERE ($1::text IS NULL OR id>$1) ORDER BY id LIMIT $4",
            &[
                &after as &(dyn ToSql + Sync),
                &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                &LIST_PAGE_ROWS as &(dyn ToSql + Sync),
            ],
        )?;
        decode_lease_page(&rows)
    }

    pub(crate) fn leases_for_connection_page(
        &self,
        connection_id: &str,
        after: Option<&str>,
    ) -> Result<Vec<Lease>, Error> {
        let (rows, _) = self.query(
            "SELECT CASE WHEN octet_length(id)<=$3 THEN id ELSE NULL END, CASE WHEN octet_length(id)<=$3 AND octet_length(data::text)<=$4 AND jsonb_typeof(data->'connection_id')='string' AND data->>'connection_id'=connection_id THEN data ELSE NULL END FROM identity_leases WHERE connection_id=$1 AND ($2::text IS NULL OR id>$2) ORDER BY id LIMIT $5",
            &[
                &connection_id as &(dyn ToSql + Sync),
                &after as &(dyn ToSql + Sync),
                &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                &LIST_PAGE_ROWS as &(dyn ToSql + Sync),
            ],
        )?;
        decode_lease_page(&rows)
    }

    pub(crate) fn leases_for_grant_page(
        &self,
        grant_id: &str,
        after: Option<&str>,
    ) -> Result<Vec<Lease>, Error> {
        let (rows, _) = self.query(
            "SELECT CASE WHEN octet_length(id)<=$3 THEN id ELSE NULL END, CASE WHEN octet_length(id)<=$3 AND octet_length(data::text)<=$4 AND jsonb_typeof(data->'connection_id')='string' AND data->>'connection_id'=connection_id AND (jsonb_typeof(data->'grant_id') IN ('string','null') OR jsonb_typeof(data->'grant_id') IS NULL) THEN data ELSE NULL END FROM identity_leases WHERE (data->>'grant_id'=$1 OR (jsonb_typeof(data->'grant_id') IS NOT NULL AND jsonb_typeof(data->'grant_id') NOT IN ('string','null'))) AND ($2::text IS NULL OR id>$2) ORDER BY id LIMIT $5",
            &[
                &grant_id as &(dyn ToSql + Sync),
                &after as &(dyn ToSql + Sync),
                &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                &LIST_PAGE_ROWS as &(dyn ToSql + Sync),
            ],
        )?;
        decode_lease_page(&rows)
    }

    pub(crate) fn leases_for_connection(
        &self,
        owner: i64,
        connection_id: &str,
        include_all_actors: bool,
    ) -> Result<Vec<u8>, Error> {
        let owner = schema_integer(owner)?;
        let mut output = ListJson::new();
        let mut after = None;
        loop {
            let (rows, _) = self.query(
                "SELECT CASE WHEN octet_length(id)<=$2 THEN id ELSE NULL END, CASE WHEN octet_length(id)<=$2 AND octet_length(data::text)<=$3 AND jsonb_typeof(data->'connection_id')='string' AND data->>'connection_id'=connection_id AND data->>'connection_id'=$1 THEN data ELSE NULL END FROM identity_leases WHERE connection_id=$1 AND ($4::text IS NULL OR id>$4) ORDER BY id LIMIT $5",
                &[
                    &connection_id as &(dyn ToSql + Sync),
                    &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                    &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                    &after as &(dyn ToSql + Sync),
                    &LIST_PAGE_ROWS as &(dyn ToSql + Sync),
                ],
            )?;
            if rows.is_empty() { break; }
            let page = decode_lease_page(&rows)?;
            for mut lease in page {
                if lease.connection_id != connection_id {
                    return Err(Error::internal("identity lease list incomplete"));
                }
                if !include_all_actors && lease.actor_id != owner as i64 {
                    after = Some(lease.id);
                    continue;
                }
                after = Some(lease.id.clone());
                lease.binding = None;
                output.push(&lease)?;
            }
            if rows.len() < LIST_PAGE_ROWS as usize { break; }
        }
        output.finish()
    }

    pub fn lease(&self, id: &str) -> Result<Lease, Error> {
        let row = self.query_row(
            "SELECT data FROM identity_leases WHERE id=$1",
            &[&id as &(dyn ToSql + Sync)],
        )?;
        Ok(row.try_get::<_, Json<Lease>>(0).map_err(pg_error)?.0)
    }

    /// Reserve one lease and link it to its admitted execution in one Store
    /// transaction. A retry after an uncertain COMMIT must first re-read the
    /// execution mapping; this method never retries the transaction itself.
    pub fn reserve_and_link_execution(
        &self,
        lease: &Lease,
        expected: &Execution,
    ) -> Result<(), Error> {
        if !provider_valid(&lease.provider_id)
            || lease.kind != expected.kind
            || lease.execution_id != expected.execution_id
        {
            return Err(Error::denied("identity authority denied"));
        }
        self.transaction(|tx| {
            let row = tx.query_row(
                "SELECT data FROM identity_executions WHERE kind=$1 AND execution_id=$2 FOR UPDATE",
                &[
                    &expected.kind as &(dyn ToSql + Sync),
                    &expected.execution_id as &(dyn ToSql + Sync),
                ],
            )?;
            let mut current = row.try_get::<_, Json<Execution>>(0).map_err(pg_error)?.0;
            if current.digest != expected.digest {
                return Err(Error::denied("identity authority denied"));
            }
            if current.state == EXECUTION_TERMINAL {
                return Err(Error::denied("identity authority denied"));
            }
            if current.state != EXECUTION_PENDING || !current.lease_id.is_empty() {
                return Err(Error::uncertain());
            }
            tx.reserve_lease(lease)?;
            current.state = EXECUTION_LIVE.to_owned();
            current.lease_id.clone_from(&lease.id);
            let count = tx.exec(
                "UPDATE identity_executions SET state=$1,lease_id=$2,data=$3 WHERE kind=$4 AND execution_id=$5 AND state=$6 AND lease_id='' AND data->>'digest'=$7",
                &[
                    &current.state as &(dyn ToSql + Sync),
                    &current.lease_id as &(dyn ToSql + Sync),
                    &Json(&current) as &(dyn ToSql + Sync),
                    &current.kind as &(dyn ToSql + Sync),
                    &current.execution_id as &(dyn ToSql + Sync),
                    &EXECUTION_PENDING as &(dyn ToSql + Sync),
                    &current.digest as &(dyn ToSql + Sync),
                ],
            )?;
            if count != 1 {
                return Err(Error::uncertain());
            }
            Ok(())
        })
    }

    pub fn register(&self, lease: &Lease) -> Result<(), Error> {
        self.transaction(|tx| {
            let count = tx.exec(
                "UPDATE identity_leases SET data=$1 WHERE id=$2 AND (data->'binding') IS NULL",
                &[
                    &Json(lease) as &(dyn ToSql + Sync),
                    &lease.id as &(dyn ToSql + Sync),
                ],
            )?;
            changed(count)?;
            tx.append_event(&lease_event(lease, "registered"))
        })
    }

    // IdentityReturn atomically accepts maintained bytes and releases one generation.
    pub fn return_lease(&self, lease: &Lease, credential: &[u8]) -> Result<(), Error> {
        if !credential_valid(credential) {
            return Err(Error::uncertain());
        }
        self.transaction(|tx| {
            tx.maintain_credential(lease, credential)?;
            let count = tx.exec(
                "DELETE FROM identity_leases WHERE id=$1 AND connection_id=$2 AND (data->>'generation')::bigint=$3",
                &[
                    &lease.id as &(dyn ToSql + Sync),
                    &lease.connection_id as &(dyn ToSql + Sync),
                    &lease.generation as &(dyn ToSql + Sync),
                ],
            )?;
            changed(count)?;
            tx.append_event(&lease_event(lease, "returned"))
        })
    }

    pub fn forget_lease(&self, id: &str) -> Result<(), Error> {
        self.transaction(|tx| {
            let row = tx.query_row(
                "SELECT data FROM identity_leases WHERE id=$1",
                &[&id as &(dyn ToSql + Sync)],
            )?;
            let lease = row.try_get::<_, Json<Lease>>(0).map_err(pg_error)?.0;
            tx.exec(
                "DELETE FROM identity_leases WHERE id=$1",
                &[&id as &(dyn ToSql + Sync)],
            )?;
            tx.append_event(&lease_event(&lease, "reconciled"))
        })
    }
}

fn decode_lease_page(rows: &[crate::pg::Row]) -> Result<Vec<Lease>, Error> {
    let mut leases = Vec::with_capacity(rows.len());
    for row in rows {
        let id = row.try_get::<_, Option<String>>(0).map_err(pg_error)?
            .ok_or_else(|| Error::internal("identity lease scan incomplete"))?;
        let value = row.try_get::<_, Option<Json<Lease>>>(1).map_err(pg_error)?
            .ok_or_else(|| Error::internal("identity lease scan incomplete"))?;
        if value.0.id != id { return Err(Error::internal("identity lease scan incomplete")); }
        leases.push(value.0);
    }
    Ok(leases)
}

impl<'a> Tx<'a> {
    fn reserve_lease(&self, lease: &Lease) -> Result<(), Error> {
        let count = self.exec(
            "INSERT INTO identity_leases(id,connection_id,data) SELECT $1,id,$2 FROM identity_connections WHERE id=$3 AND generation=$4 AND state='ready' AND data->>'provider_id'=$5 AND ($6='muse' OR NOT EXISTS(SELECT 1 FROM identity_leases WHERE connection_id=$7)) AND ($8='' OR EXISTS(SELECT 1 FROM identity_grants WHERE id=$9 AND connection_id=$10 AND user_id=$11 AND project_id=$12 AND revision=$13 AND revoked=FALSE))",
            &[
                &lease.id as &(dyn ToSql + Sync),
                &Json(lease) as &(dyn ToSql + Sync),
                &lease.connection_id as &(dyn ToSql + Sync),
                &schema_integer(lease.generation)? as &(dyn ToSql + Sync),
                &lease.provider_id as &(dyn ToSql + Sync),
                &lease.provider_id as &(dyn ToSql + Sync),
                &lease.connection_id as &(dyn ToSql + Sync),
                &lease.grant_id as &(dyn ToSql + Sync),
                &lease.grant_id as &(dyn ToSql + Sync),
                &lease.connection_id as &(dyn ToSql + Sync),
                &schema_integer(lease.actor_id)? as &(dyn ToSql + Sync),
                &lease.project_id as &(dyn ToSql + Sync),
                &schema_integer(lease.grant_revision)? as &(dyn ToSql + Sync),
            ],
        )?;
        if count != 1 {
            return Err(Error::busy());
        }
        self.append_event(&lease_event(lease, "reserved"))
    }

    pub(crate) fn maintain_credential(
        &self,
        lease: &Lease,
        credential: &[u8],
    ) -> Result<(), Error> {
        use crate::wire::CODEX;
        let found = self.query(
            "SELECT data FROM identity_connections WHERE id=$1 AND generation=$2 AND state='ready'",
            &[
                &lease.connection_id as &(dyn ToSql + Sync),
                &schema_integer(lease.generation)? as &(dyn ToSql + Sync),
            ],
        )?;
        let Some(row) = found.0.into_iter().next() else {
            return Err(Error::stale());
        };
        let mut connection = row.try_get::<_, Json<Connection>>(0).map_err(pg_error)?.0;
        let rotated = |id: &str| id == CODEX;
        if connection.provider_id != lease.provider_id || !rotated(&lease.provider_id) {
            return Err(Error::denied("identity authority denied"));
        }
        connection.generation += 1;
        let sealed = self.store.grants()?.seal(
            credential,
            &identity_binding(&connection.id, connection.generation),
        );
        let count = self.exec(
            "UPDATE identity_connections SET generation=$1,data=$2,credential=$3 WHERE id=$4 AND generation=$5",
            &[
                &schema_integer(connection.generation)? as &(dyn ToSql + Sync),
                &Json(&connection) as &(dyn ToSql + Sync),
                &sealed as &(dyn ToSql + Sync),
                &connection.id as &(dyn ToSql + Sync),
                &schema_integer(lease.generation)? as &(dyn ToSql + Sync),
            ],
        )?;
        changed(count)
    }
}
