// Delegated grants, extracted from store.rs (A05.M).
use super::store::{changed, Param, Store};
use crate::wire::{Error, Event, Grant, UnixTime};

impl Store {
    pub fn save_grant(&self, grant: &Grant) -> Result<(), Error> {
        let data = serde_json::to_string(grant)?;
        self.transaction(|tx| {
            tx.exec(
                "INSERT INTO identity_grants(id,connection_id,user_id,project_id,revision,revoked,data) VALUES($1,$2,$3,$4,$5,$6,$7)",
                &[
                    Param::text(&grant.id),
                    Param::text(&grant.connection_id),
                    Param::int(grant.user_id),
                    Param::text(&grant.project_id),
                    Param::int(grant.revision),
                    Param::boolean(grant.revoked),
                    Param::json(&data),
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
            &[Param::text(id)],
        )?;
        Ok(serde_json::from_str(row.text(0)?)?)
    }

    pub fn grants_for(&self, id: &str) -> Result<Vec<Grant>, Error> {
        let (rows, _) = self.query(
            "SELECT data FROM identity_grants WHERE connection_id=$1 ORDER BY id",
            &[Param::text(id)],
        )?;
        rows.iter()
            .map(|r| Ok(serde_json::from_str(r.text(0)?)?))
            .collect()
    }

    pub fn revoke_grant(&self, grant: &Grant) -> Result<(), Error> {
        let mut updated = grant.clone();
        updated.revoked = true;
        updated.revision += 1;
        let data = serde_json::to_string(&updated)?;
        let previous = updated.revision - 1;
        self.transaction(|tx| {
            let count = tx.exec(
                "UPDATE identity_grants SET revoked=TRUE,revision=$1,data=$2 WHERE id=$3 AND revision=$4",
                &[Param::int(updated.revision), Param::json(&data), Param::text(&grant.id), Param::int(previous)],
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
