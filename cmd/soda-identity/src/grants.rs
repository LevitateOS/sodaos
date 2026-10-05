// Delegated grants and revocation, extracted from control.rs (A05.M).
use super::control::{join_errors, new_id, Controller};
use crate::wire::{Error, Grant, GrantRequest, READY, REVOKED};

impl Controller {
    pub fn grants(&self, owner: i64, id: &str) -> Result<Vec<Grant>, Error> {
        let state = self.lock();
        Self::owned(&state, owner, id)?;
        state.store.grants_for(id)
    }

    pub fn create_grant(&self, owner: i64, input: &GrantRequest) -> Result<Grant, Error> {
        let state = self.lock();
        input.validate()?;
        let connection = Self::owned(&state, owner, &input.connection_id)?;
        if connection.state != READY {
            return Err(Error::uncertain());
        }
        let grant = Grant {
            id: new_id(),
            connection_id: input.connection_id.clone(),
            user_id: input.user_id,
            project_id: input.project_id.clone(),
            revision: 1,
            revoked: false,
        };
        state.store.save_grant(&grant)?;
        Ok(grant)
    }

    pub fn revoke(&self, owner: i64, id: &str) -> Result<(), Error> {
        let state = self.lock();
        let connection = Self::owned(&state, owner, id)?;
        state.store.set_state(&connection, REVOKED)?;
        let all = state.store.leases()?;
        let mut failures = Vec::new();
        for lease in all {
            if lease.connection_id == id {
                if let Err(err) = Self::end(&state, &lease) {
                    failures.push(err);
                }
            }
        }
        join_errors(failures)
    }

    pub fn revoke_grant(&self, owner: i64, id: &str) -> Result<(), Error> {
        let state = self.lock();
        let grant = state.store.grant(id)?;
        Self::owned(&state, owner, &grant.connection_id)?;
        if !grant.revoked {
            state.store.revoke_grant(&grant)?;
        }
        let all = state.store.leases()?;
        let mut failures = Vec::new();
        for lease in all {
            if lease.grant_id == id {
                if let Err(err) = Self::finish_lease(&state, &lease) {
                    failures.push(err);
                }
            }
        }
        join_errors(failures)
    }
}
