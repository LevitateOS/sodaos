// Delegated grants and revocation, extracted from control.rs (A05.M).
use super::control::{new_id, Controller, LeaseFailures};
use super::store_connections::LIST_PAGE_ROWS;
use crate::wire::{Error, Grant, GrantRequest, READY, REVOKED};

impl Controller {
    pub fn grants(&self, owner: i64, id: &str) -> Result<Vec<u8>, Error> {
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
        let mut failures = LeaseFailures::new();
        let mut after: Option<String> = None;
        loop {
            let page = state.store.leases_for_connection_page(id, after.as_deref())
                .map_err(|_| Error::internal("lease scan incomplete"))?;
            if page.is_empty() { break; }
            let count = page.len();
            for lease in page {
                after = Some(lease.id.clone());
                failures.record(Self::end(&state, &lease));
            }
            if count < LIST_PAGE_ROWS as usize { break; }
        }
        failures.finish()
    }

    pub fn revoke_grant(&self, owner: i64, id: &str) -> Result<(), Error> {
        let state = self.lock();
        let grant = state.store.grant(id)?;
        Self::owned(&state, owner, &grant.connection_id)?;
        if !grant.revoked {
            state.store.revoke_grant(&grant)?;
        }
        let mut failures = LeaseFailures::new();
        let mut after: Option<String> = None;
        loop {
            let page = state.store.leases_for_grant_page(id, after.as_deref())
                .map_err(|_| Error::internal("lease scan incomplete"))?;
            if page.is_empty() { break; }
            let count = page.len();
            for lease in page {
                after = Some(lease.id.clone());
                failures.record(Self::finish_lease(&state, &lease));
            }
            if count < LIST_PAGE_ROWS as usize { break; }
        }
        failures.finish()
    }
}
