// Completion, revocation and reconciliation, from control.rs (A05.M).
use super::control::{zeroize, Controller, LeaseFailures, State};
use super::store_connections::LIST_PAGE_ROWS;
use crate::wire::{
    credential_valid, Binding, Error, Lease, UnixTime, MUSE, READY, REAUTH, REVOKED,
};

impl Controller {
    pub(crate) fn uncertain(state: &State, lease: &Lease) -> Result<(), Error> {
        let connection = state.store.connection(&lease.connection_id)?;
        if connection.state == REVOKED {
            return Ok(());
        }
        state.store.set_state(&connection, REAUTH)
    }

    pub(crate) fn end(state: &State, lease: &Lease) -> Result<(), Error> {
        if lease.binding.is_none() {
            state.store.forget_lease(&lease.id)?;
            return Self::observe_terminal(state, lease);
        }
        if lease.provider_id != MUSE {
            Self::uncertain(state, lease)?;
        }
        if lease.binding.is_some() && state.runtime.stop(lease).is_err() {
            let _ = Self::uncertain(state, lease);
            return Err(Error::uncertain());
        }
        state.store.forget_lease(&lease.id)?;
        Self::observe_terminal(state, lease)
    }

    pub fn end_lease(&self, owner: i64, id: &str) -> Result<(), Error> {
        let state = self.lock();
        let lease = state.store.lease(id)?;
        if lease.actor_id != owner {
            Self::owned(&state, owner, &lease.connection_id)?;
        }
        Self::finish_lease(&state, &lease)
    }

    pub(crate) fn finish_lease(state: &State, lease: &Lease) -> Result<(), Error> {
        if lease.binding.is_none() {
            state.store.forget_lease(&lease.id)?;
            return Self::observe_terminal(state, lease);
        }
        if lease.provider_id == MUSE {
            return Self::end(state, lease);
        }
        let captured = state.runtime.finish(lease);
        // Capture failure does not establish retirement; always attempt
        // native stop.
        let stop = state.runtime.stop(lease);
        let valid = captured.is_ok() && captured.as_deref().map(credential_valid).unwrap_or(false);
        let mut data = captured.unwrap_or_default();
        if !valid || stop.is_err() {
            zeroize(&mut data);
            let _ = Self::uncertain(state, lease);
            return Err(Error::uncertain());
        }
        let result = state.store.return_lease(lease, &data);
        zeroize(&mut data);
        if let Err(err) = result {
            let _ = Self::uncertain(state, lease);
            return Err(err);
        }
        Self::observe_terminal(state, lease)
    }

    // ReconcileLease is scoped recovery, never cancellation of another execution.
    // An admitted-but-unreserved lease retires back to pending so the same
    // identity may retry; a bound lease completed native work and stays
    // terminal once retired (I06-F1).
    pub fn reconcile_lease(&self, id: &str) -> Result<(), Error> {
        let state = self.lock();
        let lease = match state.store.lease(id) {
            Ok(lease) => lease,
            Err(err) if err.is_not_found() => return Ok(()),
            Err(err) => return Err(err),
        };
        if lease.binding.is_none() {
            state.store.forget_lease(&lease.id)?;
            return Self::release_execution_lease(&state, &lease);
        }
        Self::end(&state, &lease)
    }

    pub fn reconcile(&self) -> Result<(), Error> {
        let state = self.lock();
        let mut failures = LeaseFailures::new();
        let mut after: Option<String> = None;
        loop {
            let page = state.store.leases_page(after.as_deref())
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

    pub fn sweep(&self) -> Result<(), Error> {
        let state = self.lock();
        let mut failures = LeaseFailures::new();
        let mut after: Option<String> = None;
        loop {
            let page = state.store.leases_page(after.as_deref())
                .map_err(|_| Error::internal("lease scan incomplete"))?;
            if page.is_empty() { break; }
            let count = page.len();
            for lease in page {
                after = Some(lease.id.clone());
                failures.record(Self::sweep_lease(&state, &lease));
            }
            if count < LIST_PAGE_ROWS as usize { break; }
        }
        failures.finish()
    }

    fn sweep_lease(state: &State, lease: &Lease) -> Result<(), Error> {
        let connection = state.store.connection(&lease.connection_id)?;
        if connection.state != READY {
            return Self::end(state, lease);
        }
        if !(lease.deadline > UnixTime::now()) {
            return Self::finish_lease(state, lease);
        }
        Ok(())
    }

    // Reject denies reuse after native subscription rejection, then retires
    // every copy.
    pub fn reject(&self, id: &str, binding: &Binding) -> Result<(), Error> {
        let state = self.lock();
        let lease = state.store.lease(id)?;
        if lease.provider_id != MUSE || lease.binding.as_ref() != Some(binding) {
            return Err(Error::denied("identity authority denied"));
        }
        Self::uncertain(&state, &lease)?;
        Self::retire_connection(&state, &lease.connection_id)
    }

    fn retire_connection(state: &State, id: &str) -> Result<(), Error> {
        let mut failures = LeaseFailures::new();
        let mut after: Option<String> = None;
        loop {
            let page = state.store.leases_for_connection_page(id, after.as_deref())
                .map_err(|_| Error::internal("lease scan incomplete"))?;
            if page.is_empty() { break; }
            let count = page.len();
            for lease in page {
                after = Some(lease.id.clone());
                failures.record(Self::end(state, &lease));
            }
            if count < LIST_PAGE_ROWS as usize { break; }
        }
        failures.finish()
    }
}
