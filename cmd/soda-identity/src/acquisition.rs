// Execution admission and lease fencing, extracted from control.rs (A05.M).
use super::control::{new_id, Controller, State};
use super::store_connections::LIST_PAGE_ROWS;
use crate::wire::{
    acquisition_digest, AcquireRequest, Connection, Error, Execution, Lease, UnixTime,
    EXECUTION_PENDING, EXECUTION_TERMINAL, FACTORY, READY, TERMINAL,
};

impl Controller {
    pub fn leases(&self, owner: i64, id: &str) -> Result<Vec<u8>, Error> {
        let state = self.lock();
        let connection = state.store.connection(id)?;
        if owner <= 0 {
            return Err(Error::denied("identity authority denied"));
        }
        state.store.leases_for_connection(owner, id, connection.owner_id == owner)
    }

    pub fn acquire(&self, input: &AcquireRequest) -> Result<Lease, Error> {
        let state = self.lock();
        input.validate(UnixTime::now())?;
        let digest = acquisition_digest(input);
        let (existing, created) = Self::admit_execution(&state, input, &digest)?;
        if !created && !existing.lease_id.is_empty() {
            return Self::replay_acquisition(&state, &existing, &digest);
        }
        // A fresh identity, or a pending identity whose earlier reservation
        // never landed, proceeds to reserve exactly one lease for this digest.
        let lease = Self::reserve_execution_lease(&state, input)?;
        state.store.reserve_and_link_execution(&lease, &existing)?;
        Ok(lease)
    }

    // admitExecution records the (kind, execution_id) identity before any
    // lease work. A terminal execution or a changed request for a live
    // identity refuses; the same digest replays the recorded lease instead
    // of reserving another.
    fn admit_execution(
        state: &State,
        input: &AcquireRequest,
        digest: &str,
    ) -> Result<(Execution, bool), Error> {
        let execution = Execution {
            binding: None,
            kind: input.kind.clone(),
            execution_id: input.execution_id.clone(),
            digest: digest.to_string(),
            state: EXECUTION_PENDING.to_string(),
            lease_id: String::new(),
        };
        let (existing, created) = state.store.admit_execution(&execution)?;
        if !created && (existing.state == EXECUTION_TERMINAL || existing.digest != digest) {
            return Err(Error::denied("identity authority denied"));
        }
        Ok((existing, created))
    }

    fn replay_acquisition(
        state: &State,
        existing: &Execution,
        digest: &str,
    ) -> Result<Lease, Error> {
        if existing.digest != digest || existing.state == EXECUTION_TERMINAL {
            return Err(Error::denied("identity authority denied"));
        }
        if existing.lease_id.is_empty() {
            return Err(Error::uncertain());
        }
        let lease = state.store.lease(&existing.lease_id).map_err(|e| {
            if e.is_not_found() {
                Error::uncertain()
            } else {
                e
            }
        })?;
        // Re-check current reservation authority: revocation or withdrawal
        // after the original acquire must not hand the lease out again.
        let connection = state.store.connection(&lease.connection_id).map_err(|e| {
            if e.is_not_found() {
                Error::uncertain()
            } else {
                e
            }
        })?;
        if connection.provider_id != lease.provider_id {
            return Err(Error::denied("identity authority denied"));
        }
        if connection.state != READY || !state.providers.contains_key(&connection.provider_id) {
            return Err(Error::uncertain());
        }
        let mut lease = lease;
        Self::authorize_reservation(state, &connection, &mut lease)?;
        Ok(lease)
    }

    fn reserve_execution_lease(state: &State, input: &AcquireRequest) -> Result<Lease, Error> {
        let connection = state.store.connection(&input.connection_id)?;
        if connection.provider_id != input.provider_id {
            return Err(Error::denied("identity authority denied"));
        }
        if connection.state != READY || !state.providers.contains_key(&connection.provider_id) {
            return Err(Error::uncertain());
        }
        let mut lease = Lease {
            repository_id: input.repository_id,
            provider_id: connection.provider_id.clone(),
            id: new_id(),
            connection_id: connection.id.clone(),
            generation: connection.generation,
            actor_id: input.actor_id,
            project_id: input.project_id.clone(),
            execution_id: input.execution_id.clone(),
            kind: input.kind.clone(),
            role: input.role.clone(),
            deadline: input.deadline,
            grant_id: String::new(),
            grant_revision: 0,
            binding: None,
        };
        Self::authorize_reservation(state, &connection, &mut lease)?;
        Ok(lease)
    }

    // GetExecution returns the immutable acquisition digest, lease/binding
    // metadata and pending/live/terminal state without credentials.
    pub fn get_execution(&self, kind: &str, execution_id: &str) -> Result<Execution, Error> {
        let state = self.lock();
        if (kind != FACTORY && kind != TERMINAL) || execution_id.is_empty() {
            return Err(Error::denied("identity authority denied"));
        }
        state.store.execution(kind, execution_id)
    }

    // CloseExecution creates a terminal acquisition tombstone even before
    // acquisition arrives, prevents subsequent acquisition/registration and
    // reconciles any existing lease.
    pub fn close_execution(&self, kind: &str, execution_id: &str) -> Result<(), Error> {
        let state = self.lock();
        if (kind != FACTORY && kind != TERMINAL) || execution_id.is_empty() {
            return Err(Error::denied("identity authority denied"));
        }
        let tombstone = Execution {
            binding: None,
            kind: kind.to_string(),
            execution_id: execution_id.to_string(),
            digest: String::new(),
            state: EXECUTION_TERMINAL.to_string(),
            lease_id: String::new(),
        };
        let (mut existing, created) = state.store.admit_execution(&tombstone)?;
        if created {
            return Ok(());
        }
        // A terminal execution with a surviving lease still needs
        // retirement; closing twice must not report success while native
        // work continues. The last observed binding stays retained for
        // attribution.
        if !existing.lease_id.is_empty() {
            match state.store.lease(&existing.lease_id) {
                Ok(lease) => {
                    if lease.binding.is_some() {
                        existing.binding = lease.binding.clone();
                    }
                    if Self::end(&state, &lease).is_err() {
                        existing.state = EXECUTION_TERMINAL.to_string();
                        let _ = state.store.observe_execution(&existing);
                        return Err(Error::uncertain());
                    }
                }
                Err(error) if error.is_not_found() => {}
                Err(_) => {
                    existing.state = EXECUTION_TERMINAL.to_string();
                    let _ = state.store.observe_execution(&existing);
                    return Err(Error::uncertain());
                }
            }
            existing.lease_id.clear();
        }
        existing.state = EXECUTION_TERMINAL.to_string();
        state.store.observe_execution(&existing)
    }

    fn authorize_reservation(
        state: &State,
        connection: &Connection,
        lease: &mut Lease,
    ) -> Result<(), Error> {
        if connection.owner_id == lease.actor_id {
            return Ok(());
        }
        if lease.project_id.is_empty() {
            return Err(Error::denied("identity authority denied"));
        }
        let mut after: Option<String> = None;
        loop {
            let grants = state.store.grants_page(&connection.id, after.as_deref())?;
            if grants.is_empty() { break; }
            let count = grants.len();
            for grant in grants {
                after = Some(grant.id.clone());
                if !grant.revoked
                    && grant.user_id == lease.actor_id
                    && grant.project_id == lease.project_id
                {
                    lease.grant_id.clone_from(&grant.id);
                    lease.grant_revision = grant.revision;
                    return Ok(());
                }
            }
            if count < LIST_PAGE_ROWS as usize { break; }
        }
        Err(Error::denied("identity authority denied"))
    }
}
