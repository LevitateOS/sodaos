// Delegated grants, extracted from store.rs (A05.M).
use super::store::{changed, schema_integer, Store};
use super::store_connections::{ListJson, LIST_PAGE_ROWS, LIST_RECORD_BYTES};
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

    pub fn grants_for(&self, connection_id: &str) -> Result<Vec<u8>, Error> {
        let mut output = ListJson::new();
        let mut after: Option<String> = None;
        loop {
            let (rows, _) = self.query(
                "SELECT CASE WHEN octet_length(id)<=$2 THEN id ELSE NULL END, CASE WHEN octet_length(id)<=$2 AND octet_length(data::text)<=$3 AND data->>'id'=id AND data->>'connection_id'=connection_id THEN data ELSE NULL END FROM identity_grants WHERE connection_id=$1 AND ($4::text IS NULL OR id>$4) ORDER BY id LIMIT $5",
                &[
                    &connection_id as &(dyn ToSql + Sync),
                    &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                    &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                    &after as &(dyn ToSql + Sync),
                    &LIST_PAGE_ROWS as &(dyn ToSql + Sync),
                ],
            )?;
            if rows.is_empty() { break; }
            for row in &rows {
                let id = row.try_get::<_, Option<String>>(0).map_err(pg_error)?
                    .ok_or_else(|| Error::internal("identity grant list incomplete"))?;
                let value = row.try_get::<_, Option<Json<Grant>>>(1).map_err(pg_error)?
                    .ok_or_else(|| Error::internal("identity grant list incomplete"))?;
                if value.0.id != id || value.0.connection_id != connection_id {
                    return Err(Error::internal("identity grant list incomplete"));
                }
                output.push(&value.0)?;
                after = Some(id);
            }
            if rows.len() < LIST_PAGE_ROWS as usize { break; }
        }
        output.finish()
    }

    pub(crate) fn grants_page(
        &self,
        connection_id: &str,
        after: Option<&str>,
    ) -> Result<Vec<Grant>, Error> {
        let (rows, _) = self.query(
            "SELECT CASE WHEN octet_length(id)<=$2 THEN id ELSE NULL END, CASE WHEN octet_length(id)<=$2 AND octet_length(data::text)<=$3 AND data->>'id'=id AND data->>'connection_id'=connection_id THEN data ELSE NULL END FROM identity_grants WHERE connection_id=$1 AND ($4::text IS NULL OR id>$4) ORDER BY id LIMIT $5",
            &[
                &connection_id as &(dyn ToSql + Sync),
                &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                &LIST_RECORD_BYTES as &(dyn ToSql + Sync),
                &after as &(dyn ToSql + Sync),
                &LIST_PAGE_ROWS as &(dyn ToSql + Sync),
            ],
        )?;
        let mut grants = Vec::with_capacity(rows.len());
        for row in &rows {
            let id = row.try_get::<_, Option<String>>(0).map_err(pg_error)?
                .ok_or_else(|| Error::internal("identity grant scan incomplete"))?;
            let value = row.try_get::<_, Option<Json<Grant>>>(1).map_err(pg_error)?
                .ok_or_else(|| Error::internal("identity grant scan incomplete"))?;
            if value.0.id != id || value.0.connection_id != connection_id {
                return Err(Error::internal("identity grant scan incomplete"));
            }
            grants.push(value.0);
        }
        Ok(grants)
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
