// Subscription admission and credential custody, mirroring
// internal/identity/control: enrollment sessions, grants, leases,
// execution fencing and native termination.
use crate::store::Store;
use crate::wire::{
    acquisition_digest, credential_valid, provider_valid, AcquireRequest, Binding, Connection,
    Enrollment, Error, Execution, Grant, GrantRequest, Lease, UnixTime, EXECUTION_LIVE,
    EXECUTION_PENDING, EXECUTION_TERMINAL, FACTORY, MUSE, READY, REAUTH, REVOKED, TERMINAL,
};
use std::collections::HashMap;
use std::sync::Mutex;

pub trait EnrollmentSession: Send {
    fn snapshot(&self) -> Enrollment;
    fn finish(&self) -> Result<(Connection, Vec<u8>), Error>;
    fn close(&self) -> Result<(), Error>;
}

pub trait Provider: Send + Sync {
    fn start(&self, owner: i64) -> Result<Box<dyn EnrollmentSession>, Error>;
}

pub trait Runtime: Send + Sync {
    fn validate(&self, lease: &Lease) -> Result<(), Error>;
    fn stop(&self, lease: &Lease) -> Result<(), Error>;
    fn finish(&self, lease: &Lease) -> Result<Vec<u8>, Error>;
}

pub fn map_provider_error(err: identity_providers::Error) -> Error {
    use identity_providers::Kind;
    match err.kind() {
        // Only teardown uncertainty escapes with its kind (the broker maps
        // it to `reauth`); every other provider failure is internal, exactly
        // like the Go broker mapping plain provider errors to `unavailable`.
        Kind::Uncertain => Error::uncertain(),
        _ => Error::internal(err.to_string()),
    }
}

pub struct CodexProvider(pub identity_providers::codex::Provider);
pub struct MuseProvider(pub identity_providers::muse::Provider);

struct CodexSession(identity_providers::codex::Session);
struct MuseSession(identity_providers::muse::Session);

impl EnrollmentSession for CodexSession {
    fn snapshot(&self) -> Enrollment {
        self.0.snapshot()
    }
    fn finish(&self) -> Result<(Connection, Vec<u8>), Error> {
        self.0.finish().map_err(map_provider_error)
    }
    fn close(&self) -> Result<(), Error> {
        self.0.close().map_err(map_provider_error)
    }
}

impl EnrollmentSession for MuseSession {
    fn snapshot(&self) -> Enrollment {
        self.0.snapshot()
    }
    fn finish(&self) -> Result<(Connection, Vec<u8>), Error> {
        self.0.finish().map_err(map_provider_error)
    }
    fn close(&self) -> Result<(), Error> {
        self.0.close().map_err(map_provider_error)
    }
}

impl Provider for CodexProvider {
    fn start(&self, owner: i64) -> Result<Box<dyn EnrollmentSession>, Error> {
        Ok(Box::new(CodexSession(
            self.0.start(owner).map_err(map_provider_error)?,
        )))
    }
}

impl Provider for MuseProvider {
    fn start(&self, owner: i64) -> Result<Box<dyn EnrollmentSession>, Error> {
        Ok(Box::new(MuseSession(
            self.0.start(owner).map_err(map_provider_error)?,
        )))
    }
}

struct EnrollmentEntry {
    owner: i64,
    label: String,
    provider_id: String,
    session: Option<Box<dyn EnrollmentSession>>,
    result: Enrollment,
}

struct State {
    store: Store,
    providers: HashMap<String, Box<dyn Provider>>,
    runtime: Box<dyn Runtime>,
    enrollments: HashMap<String, EnrollmentEntry>,
}

pub struct Controller {
    state: Mutex<State>,
}

impl Controller {
    pub fn new(
        store: Store,
        providers: HashMap<String, Box<dyn Provider>>,
        runtime: Box<dyn Runtime>,
    ) -> Result<Controller, Error> {
        if providers.is_empty() {
            return Err(Error::internal("identity custody dependencies required"));
        }
        for id in providers.keys() {
            if !provider_valid(id) {
                return Err(Error::internal("invalid subscription provider"));
            }
        }
        Ok(Controller {
            state: Mutex::new(State {
                store,
                providers,
                runtime,
                enrollments: HashMap::new(),
            }),
        })
    }

    fn lock(&self) -> std::sync::MutexGuard<'_, State> {
        self.state.lock().unwrap()
    }

    fn pending_enrollment(state: &State, owner: i64, provider_id: &str) -> bool {
        state.enrollments.values().any(|entry| {
            entry.owner == owner
                && entry.provider_id == provider_id
                && entry.result.connection.is_none()
                && entry.session.as_ref().is_some_and(|session| {
                    matches!(session.snapshot().state.as_str(), "pending" | "completed")
                })
        })
    }

    fn owned(state: &State, owner: i64, id: &str) -> Result<Connection, Error> {
        let connection = state.store.connection(id)?;
        if owner <= 0 || connection.owner_id != owner {
            return Err(Error::denied("identity authority denied"));
        }
        Ok(connection)
    }

    pub fn connections(&self, owner: i64) -> Result<Vec<Connection>, Error> {
        if owner <= 0 {
            return Err(Error::denied("identity authority denied"));
        }
        self.lock().store.connections(owner)
    }

    pub fn available(&self, actor: i64, project: &str) -> Result<Vec<Connection>, Error> {
        if actor <= 0 || project.is_empty() {
            return Err(Error::denied("identity authority denied"));
        }
        self.lock().store.available(actor, project)
    }

    pub fn start_enrollment(
        &self,
        owner: i64,
        provider_id: &str,
        label: &str,
    ) -> Result<Enrollment, Error> {
        let mut state = self.lock();
        if !state.providers.contains_key(provider_id)
            || owner <= 0
            || label.len() > 100
            || label.is_empty()
        {
            return Err(Error::denied("identity authority denied"));
        }
        if Self::pending_enrollment(&state, owner, provider_id) {
            return Err(Error::busy());
        }
        let session = state.providers[provider_id].start(owner)?;
        let mut result = session.snapshot();
        result.provider_id = provider_id.to_string();
        if result.id.is_empty() {
            let _ = session.close();
            return Err(Error::uncertain());
        }
        state.enrollments.insert(
            result.id.clone(),
            EnrollmentEntry {
                owner,
                label: label.to_string(),
                provider_id: provider_id.to_string(),
                session: Some(session),
                result: result.clone(),
            },
        );
        Ok(result)
    }

    pub fn enrollment(&self, owner: i64, id: &str) -> Result<Enrollment, Error> {
        let mut guard = self.lock();
        let state: &mut State = &mut guard;
        let entry = state.enrollments.get_mut(id);
        let Some(entry) = entry else {
            return Err(Error::denied("identity authority denied"));
        };
        if entry.owner != owner {
            return Err(Error::denied("identity authority denied"));
        }
        let Some(session) = entry.session.as_ref() else {
            return Ok(entry.result.clone());
        };
        let mut result = session.snapshot();
        result.provider_id.clone_from(&entry.provider_id);
        if result.state != "completed" {
            entry.result = result.clone();
            return Ok(result);
        }
        let (mut connection, mut data) = match session.finish() {
            Ok(finished) => finished,
            Err(_) => {
                result.state = "failed".to_string();
                result.error = "Provider enrollment could not be retained".to_string();
                let _ = entry.session.as_ref().map(|s| s.close());
                entry.session = None;
                entry.result = result.clone();
                return Err(Error::internal("Provider enrollment could not be retained"));
            }
        };
        connection.provider_id.clone_from(&entry.provider_id);
        connection.id = new_id();
        connection.owner_id = owner;
        connection.label.clone_from(&entry.label);
        connection.generation = 1;
        connection.state = READY.to_string();
        let save = state.store.save_connection(&connection, &data);
        zeroize(&mut data);
        let _ = entry.session.as_ref().map(|s| s.close());
        entry.session = None;
        if let Err(err) = save {
            result.state = "failed".to_string();
            result.error = "Provider enrollment could not be retained".to_string();
            entry.result = result.clone();
            return Err(err);
        }
        result.connection = Some(connection);
        result.user_code.clear();
        result.verification_url.clear();
        entry.result = result.clone();
        Ok(result)
    }

    pub fn cancel_enrollment(&self, owner: i64, id: &str) -> Result<(), Error> {
        let mut state = self.lock();
        let Some(entry) = state.enrollments.get_mut(id) else {
            return Err(Error::denied("identity authority denied"));
        };
        if entry.owner != owner {
            return Err(Error::denied("identity authority denied"));
        }
        if let Some(session) = entry.session.as_ref() {
            session.close()?;
            entry.session = None;
        }
        entry.result.state = "canceled".to_string();
        entry.result.user_code.clear();
        Ok(())
    }

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

    pub fn leases(&self, owner: i64, id: &str) -> Result<Vec<Lease>, Error> {
        let state = self.lock();
        let connection = state.store.connection(id)?;
        if owner <= 0 {
            return Err(Error::denied("identity authority denied"));
        }
        let all = state.store.leases()?;
        let mut out = Vec::new();
        for mut lease in all {
            if lease.connection_id == id
                && (connection.owner_id == owner || lease.actor_id == owner)
            {
                lease.binding = None;
                out.push(lease);
            }
        }
        Ok(out)
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
        let mut existing = existing;
        existing.state = EXECUTION_LIVE.to_string();
        existing.lease_id.clone_from(&lease.id);
        state.store.observe_execution(&existing)?;
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
        state.store.reserve(&lease)?;
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
            if let Ok(lease) = state.store.lease(&existing.lease_id) {
                if lease.binding.is_some() {
                    existing.binding = lease.binding.clone();
                }
                if Self::end(&state, &lease).is_err() {
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
        let grants = state.store.grants_for(&connection.id)?;
        for grant in grants {
            if !grant.revoked
                && grant.user_id == lease.actor_id
                && grant.project_id == lease.project_id
            {
                lease.grant_id.clone_from(&grant.id);
                lease.grant_revision = grant.revision;
                return Ok(());
            }
        }
        Err(Error::denied("identity authority denied"))
    }

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
    fn observe_terminal(state: &State, lease: &Lease) -> Result<(), Error> {
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
    fn release_execution_lease(state: &State, lease: &Lease) -> Result<(), Error> {
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

    fn uncertain(state: &State, lease: &Lease) -> Result<(), Error> {
        let connection = state.store.connection(&lease.connection_id)?;
        if connection.state == REVOKED {
            return Ok(());
        }
        state.store.set_state(&connection, REAUTH)
    }

    fn end(state: &State, lease: &Lease) -> Result<(), Error> {
        if lease.binding.is_none() {
            state.store.forget_lease(&lease.id)?;
            return Self::release_execution_lease(state, lease);
        }
        if lease.provider_id != MUSE {
            Self::uncertain(state, lease)?;
        }
        if lease.binding.is_some() && state.runtime.stop(lease).is_err() {
            let _ = Self::uncertain(state, lease);
            return Err(Error::uncertain());
        }
        state.store.forget_lease(&lease.id)?;
        Self::release_execution_lease(state, lease)
    }

    pub fn end_lease(&self, owner: i64, id: &str) -> Result<(), Error> {
        let state = self.lock();
        let lease = state.store.lease(id)?;
        if lease.actor_id != owner {
            Self::owned(&state, owner, &lease.connection_id)?;
        }
        Self::finish_lease(&state, &lease)
    }

    fn finish_lease(state: &State, lease: &Lease) -> Result<(), Error> {
        if lease.binding.is_none() {
            state.store.forget_lease(&lease.id)?;
            return Self::release_execution_lease(state, lease);
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
        Self::release_execution_lease(state, lease)
    }

    // ReconcileLease is scoped recovery, never cancellation of another execution.
    pub fn reconcile_lease(&self, id: &str) -> Result<(), Error> {
        let state = self.lock();
        let lease = match state.store.lease(id) {
            Ok(lease) => lease,
            Err(err) if err.is_not_found() => return Ok(()),
            Err(err) => return Err(err),
        };
        Self::end(&state, &lease)
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

    pub fn reconcile(&self) -> Result<(), Error> {
        let state = self.lock();
        let all = state.store.leases()?;
        let mut failures = Vec::new();
        for lease in all {
            if let Err(err) = Self::end(&state, &lease) {
                failures.push(err);
            }
        }
        join_errors(failures)
    }

    pub fn sweep(&self) -> Result<(), Error> {
        let state = self.lock();
        let all = state.store.leases()?;
        let mut failures = Vec::new();
        for lease in all {
            if let Err(err) = Self::sweep_lease(&state, &lease) {
                failures.push(err);
            }
        }
        join_errors(failures)
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
        let all = state.store.leases()?;
        let mut failures = Vec::new();
        for other in all {
            if other.connection_id == id {
                if let Err(err) = Self::end(state, &other) {
                    failures.push(err);
                }
            }
        }
        join_errors(failures)
    }

    pub fn close(&self) -> Result<(), Error> {
        let state = self.lock();
        let mut failures = Vec::new();
        for entry in state.enrollments.values() {
            if let Some(session) = entry.session.as_ref() {
                if let Err(err) = session.close() {
                    failures.push(err);
                }
            }
        }
        join_errors(failures)
    }
}

fn join_errors(failures: Vec<Error>) -> Result<(), Error> {
    if failures.is_empty() {
        return Ok(());
    }
    let messages: Vec<String> = failures.iter().map(|e| e.to_string()).collect();
    Err(Error::internal(messages.join("; ")))
}

fn new_id() -> String {
    let mut bytes = [0u8; 16];
    fill_random(&mut bytes).expect("identity randomness unavailable");
    identity_providers::sha256::hex(&bytes)
}

fn fill_random(out: &mut [u8]) -> std::io::Result<()> {
    use std::io::Read;
    std::fs::File::open("/dev/urandom")?.read_exact(out)
}

fn zeroize(data: &mut [u8]) {
    for byte in data.iter_mut() {
        unsafe { std::ptr::write_volatile(byte, 0) };
    }
}
