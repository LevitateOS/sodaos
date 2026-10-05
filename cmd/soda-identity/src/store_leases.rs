// Leases, reservation and credential maintenance, extracted from store.rs (A05.M).
use super::store::{changed, identity_binding, Param, Store, Tx};
use super::store_events::lease_event;
use crate::wire::{credential_valid, provider_valid, Connection, Error, Lease};

impl Store {
    pub fn leases(&self) -> Result<Vec<Lease>, Error> {
        let (rows, _) = self.query("SELECT data FROM identity_leases ORDER BY id", &[])?;
        rows.iter()
            .map(|r| Ok(serde_json::from_str(r.text(0)?)?))
            .collect()
    }

    pub fn lease(&self, id: &str) -> Result<Lease, Error> {
        let row = self.query_row(
            "SELECT data FROM identity_leases WHERE id=?",
            &[Param::text(id)],
        )?;
        Ok(serde_json::from_str(row.text(0)?)?)
    }

    pub fn reserve(&self, lease: &Lease) -> Result<(), Error> {
        if !provider_valid(&lease.provider_id) {
            return Err(Error::denied("identity authority denied"));
        }
        let data = serde_json::to_string(lease)?;
        self.transaction(|tx| {
            let count = tx.exec(
                "INSERT INTO identity_leases(id,connection_id,data) SELECT ?,id,? FROM identity_connections WHERE id=? AND generation=? AND state='ready' AND data->>'provider_id'=? AND (?='muse' OR NOT EXISTS(SELECT 1 FROM identity_leases WHERE connection_id=?)) AND (?='' OR EXISTS(SELECT 1 FROM identity_grants WHERE id=? AND connection_id=? AND user_id=? AND project_id=? AND revision=? AND revoked=FALSE))",
                &[
                    Param::text(&lease.id),
                    Param::text(&data),
                    Param::text(&lease.connection_id),
                    Param::int(lease.generation),
                    Param::text(&lease.provider_id),
                    Param::text(&lease.provider_id),
                    Param::text(&lease.connection_id),
                    Param::text(&lease.grant_id),
                    Param::text(&lease.grant_id),
                    Param::text(&lease.connection_id),
                    Param::int(lease.actor_id),
                    Param::text(&lease.project_id),
                    Param::int(lease.grant_revision),
                ],
            )?;
            if count != 1 {
                return Err(Error::busy());
            }
            tx.append_event(&lease_event(lease, "reserved"))
        })
    }

    pub fn register(&self, lease: &Lease) -> Result<(), Error> {
        let data = serde_json::to_string(lease)?;
        self.transaction(|tx| {
            let count = tx.exec(
                "UPDATE identity_leases SET data=? WHERE id=? AND (data->'binding') IS NULL",
                &[Param::text(&data), Param::text(&lease.id)],
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
                "DELETE FROM identity_leases WHERE id=? AND connection_id=? AND (data->>'generation')::bigint=?",
                &[Param::text(&lease.id), Param::text(&lease.connection_id), Param::int(lease.generation)],
            )?;
            changed(count)?;
            tx.append_event(&lease_event(lease, "returned"))
        })
    }

    pub fn forget_lease(&self, id: &str) -> Result<(), Error> {
        self.transaction(|tx| {
            let row = tx.query_row(
                "SELECT data FROM identity_leases WHERE id=?",
                &[Param::text(id)],
            )?;
            let lease: Lease = serde_json::from_str(row.text(0)?)?;
            tx.exec("DELETE FROM identity_leases WHERE id=?", &[Param::text(id)])?;
            tx.append_event(&lease_event(&lease, "reconciled"))
        })
    }
}

impl<'a> Tx<'a> {
    pub(crate) fn maintain_credential(
        &self,
        lease: &Lease,
        credential: &[u8],
    ) -> Result<(), Error> {
        use crate::wire::CODEX;
        let found = self.query(
            "SELECT data FROM identity_connections WHERE id=? AND generation=? AND state='ready'",
            &[
                Param::text(&lease.connection_id),
                Param::int(lease.generation),
            ],
        )?;
        let Some(row) = found.0.into_iter().next() else {
            return Err(Error::stale());
        };
        let mut connection: Connection = serde_json::from_str(row.text(0)?)?;
        let rotated = |id: &str| id == CODEX;
        if connection.provider_id != lease.provider_id || !rotated(&lease.provider_id) {
            return Err(Error::denied("identity authority denied"));
        }
        connection.generation += 1;
        let data = serde_json::to_string(&connection)?;
        let sealed = self.store.grants()?.seal(
            credential,
            &identity_binding(&connection.id, connection.generation),
        );
        let count = self.exec(
            "UPDATE identity_connections SET generation=?,data=?,credential=? WHERE id=? AND generation=?",
            &[
                Param::int(connection.generation),
                Param::text(&data),
                Param::bytea(&sealed),
                Param::text(&connection.id),
                Param::int(lease.generation),
            ],
        )?;
        changed(count)
    }
}
