// Native binding registration and private delivery, from control.rs (A05.M).
use super::control::{Controller, State};
use crate::wire::{
    credential_valid, Binding, Connection, Error, Lease, UnixTime, EXECUTION_LIVE,
    EXECUTION_PENDING, EXECUTION_TERMINAL, MUSE, READY,
};

impl Controller {
    pub fn register(&self, id: &str, binding: &Binding) -> Result<(Lease, Vec<u8>), Error> {
        let state = self.lock();
        let (mut lease, connection) = Self::registration(&state, id, binding)?;
        Self::registration_execution(&state, &lease)?;
        lease.binding = Some(binding.clone());
        state.store.register(&lease)?;
        if Self::observe_execution_binding(&state, &lease).is_err() {
            let _ = Self::uncertain(&state, &lease);
            return Err(Error::uncertain());
        }
        if state.runtime.validate(&lease).is_err() {
            let _ = Self::uncertain(&state, &lease);
            return Err(Error::denied("identity authority denied"));
        }
        let data = state.store.credential(&connection)?;
        Ok((lease, data))
    }

    // registrationExecution refuses a lease whose execution was closed
    // after acquisition. A closed execution whose lease survived
    // reconciliation stays fenced; it can never deliver credentials again.
    fn registration_execution(state: &State, lease: &Lease) -> Result<(), Error> {
        let execution = state
            .store
            .execution(&lease.kind, &lease.execution_id)
            .map_err(|_| Error::denied("identity authority denied"))?;
        if execution.state == EXECUTION_TERMINAL || execution.lease_id != lease.id {
            return Err(Error::denied("identity authority denied"));
        }
        Ok(())
    }

    fn observe_execution_binding(state: &State, lease: &Lease) -> Result<(), Error> {
        let mut execution = state.store.execution(&lease.kind, &lease.execution_id)?;
        execution.binding = lease.binding.clone();
        state.store.observe_execution(&execution)
    }

    // observeTerminal retains the nonsecret acquisition identity after its
    // lease is gone. A missing execution is ignored; only admitted leases
    // reach it.
    pub(crate) fn observe_terminal(state: &State, lease: &Lease) -> Result<(), Error> {
        let mut execution = match state.store.execution(&lease.kind, &lease.execution_id) {
            Ok(execution) => execution,
            Err(err) if err.is_not_found() => return Ok(()),
            Err(err) => return Err(err),
        };
        execution.state = EXECUTION_TERMINAL.to_string();
        state.store.observe_execution(&execution)
    }

    // releaseExecutionLease detaches a reconciled lease from its execution
    // without tombstoning it: recovery may retry the same identity, gated
    // by current connection authority and native re-attestation. Only
    // Return and CloseExecution make an execution terminal. A tombstone
    // keeps its state.
    pub(crate) fn release_execution_lease(state: &State, lease: &Lease) -> Result<(), Error> {
        let mut execution = match state.store.execution(&lease.kind, &lease.execution_id) {
            Ok(execution) => execution,
            Err(err) if err.is_not_found() => return Ok(()),
            Err(err) => return Err(err),
        };
        if !execution.lease_id.is_empty() && execution.lease_id != lease.id {
            return Ok(());
        }
        execution.lease_id.clear();
        execution.binding = None;
        if execution.state == EXECUTION_LIVE {
            execution.state = EXECUTION_PENDING.to_string();
        }
        state.store.observe_execution(&execution)
    }

    fn registration(
        state: &State,
        id: &str,
        binding: &Binding,
    ) -> Result<(Lease, Connection), Error> {
        let lease = state.store.lease(id)?;
        binding.validate()?;
        if lease.binding.is_some()
            || binding.kind != lease.kind
            || binding.generation != lease.generation
            || !(lease.deadline > UnixTime::now())
        {
            return Err(Error::denied("identity authority denied"));
        }
        let connection = Self::registration_authority(state, &lease)?;
        Ok((lease, connection))
    }

    fn registration_authority(state: &State, lease: &Lease) -> Result<Connection, Error> {
        let connection = state.store.connection(&lease.connection_id)?;
        if connection.state != READY
            || connection.generation != lease.generation
            || connection.provider_id != lease.provider_id
        {
            return Err(Error::stale());
        }
        if !lease.grant_id.is_empty() {
            let grant = state.store.grant(&lease.grant_id)?;
            if grant.revoked || grant.revision != lease.grant_revision {
                return Err(Error::denied("identity authority denied"));
            }
        }
        Ok(connection)
    }

    pub fn return_lease(&self, id: &str, binding: &Binding, data: &[u8]) -> Result<(), Error> {
        let state = self.lock();
        let lease = state.store.lease(id)?;
        if lease.binding.as_ref() != Some(binding) {
            return Err(Error::denied("identity authority denied"));
        }
        // Trusted native termination is mandatory even when the caller reports exit.
        if state.runtime.stop(&lease).is_err() {
            let _ = Self::uncertain(&state, &lease);
            return Err(Error::uncertain());
        }
        if lease.provider_id == MUSE {
            state.store.forget_lease(&lease.id)?;
            return Self::observe_terminal(&state, &lease);
        }
        if !credential_valid(data) {
            let _ = Self::uncertain(&state, &lease);
            return Err(Error::uncertain());
        }
        state.store.return_lease(&lease, data)?;
        Self::observe_terminal(&state, &lease)
    }
}
