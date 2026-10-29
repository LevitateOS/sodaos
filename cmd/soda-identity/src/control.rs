// Subscription admission and credential custody, mirroring
// internal/identity/control: enrollment sessions, grants, leases,
// execution fencing and native termination.
use crate::providers::resolution::BoundBinary;
use crate::store::Store;
use crate::wire::{provider_valid, Connection, Enrollment, Error, Lease};
use std::collections::{HashMap, HashSet};
use std::sync::{Arc, Mutex};

pub trait EnrollmentSession: Send {
    fn snapshot(&self) -> Enrollment;
    fn finish(&self) -> Result<(Connection, Vec<u8>), Error>;
    fn close(&self) -> Result<(), Error>;
}

pub trait Provider: Send + Sync {
    fn start(&self, owner: i64) -> Result<Box<dyn EnrollmentSession>, Error>;
    /// Pin the fresh shared selection for NEW lease work (`None` in
    /// explicit mode). A refresh must never corrupt an active run, and new
    /// acquisitions pick up the refreshed selection without a restart.
    fn bind_new(&self) -> Result<Option<BoundBinary>, Error>;
}

pub trait Runtime: Send + Sync {
    fn validate(&self, lease: &Lease) -> Result<(), Error>;
    fn stop(&self, lease: &Lease) -> Result<(), Error>;
    fn finish(&self, lease: &Lease) -> Result<Vec<u8>, Error>;
}

pub fn map_provider_error(err: crate::providers::Error) -> Error {
    use crate::providers::Kind;
    match err.kind() {
        // Only teardown uncertainty escapes with its kind (the broker maps
        // it to `reauth`); every other provider failure is internal, exactly
        // like the Go broker mapping plain provider errors to `unavailable`.
        Kind::Uncertain => Error::uncertain(),
        _ => Error::internal(err.to_string()),
    }
}

pub struct CodexProvider(pub crate::providers::codex::Provider);
pub struct MuseProvider(pub crate::providers::muse::Provider);

struct CodexSession(crate::providers::codex::Session);
struct MuseSession(crate::providers::muse::Session);

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

    fn bind_new(&self) -> Result<Option<BoundBinary>, Error> {
        self.0.bind_new().map_err(map_provider_error)
    }
}

impl Provider for MuseProvider {
    fn start(&self, owner: i64) -> Result<Box<dyn EnrollmentSession>, Error> {
        Ok(Box::new(MuseSession(
            self.0.start(owner).map_err(map_provider_error)?,
        )))
    }

    fn bind_new(&self) -> Result<Option<BoundBinary>, Error> {
        self.0.bind_new().map_err(map_provider_error)
    }
}

pub(crate) struct EnrollmentEntry {
    pub(crate) owner: i64,
    pub(crate) label: String,
    pub(crate) provider_id: String,
    pub(crate) session: Option<Box<dyn EnrollmentSession>>,
    pub(crate) result: Enrollment,
}

pub(crate) struct State {
    pub(crate) store: Store,
    /// Shared so a slow provider start runs without the controller lock.
    pub(crate) providers: HashMap<String, Arc<dyn Provider>>,
    pub(crate) runtime: Box<dyn Runtime>,
    pub(crate) enrollments: HashMap<String, EnrollmentEntry>,
    /// Owner/provider pairs whose provider start is running unlocked; a
    /// second start for the same pair is refused as busy until it settles.
    pub(crate) starting: HashSet<(i64, String)>,
    /// Live lease pins against provider-store cleanup: lease id to the
    /// exact bound bundle. Present only for store-mode providers; the pin
    /// is released once no live lease row references the bundle, so a
    /// refresh never corrupts an active run. Never blocks on this map:
    /// callers hold the controller lock, so the map lock is never taken
    /// in the opposite order.
    pub(crate) lease_refs: Mutex<HashMap<String, BoundBinary>>,
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
                providers: providers
                    .into_iter()
                    .map(|(id, provider)| (id, Arc::from(provider)))
                    .collect(),
                runtime,
                enrollments: HashMap::new(),
                starting: HashSet::new(),
                lease_refs: Mutex::new(HashMap::new()),
            }),
        })
    }

    pub(crate) fn lock(&self) -> std::sync::MutexGuard<'_, State> {
        self.state.lock().unwrap()
    }

    /// Pin the fresh shared selection for a newly admitted lease. Explicit
    /// providers pin nothing. In store mode a known-stale/failed selection
    /// fails acquisition here, so blocked new work surfaces instead of
    /// running on a stale pin; live leases keep their existing pins.
    pub(crate) fn pin_lease(state: &State, lease: &Lease) -> Result<(), Error> {
        let provider = state
            .providers
            .get(&lease.provider_id)
            .ok_or_else(|| Error::internal("identity custody dependencies required"))?;
        if let Some(bound) = provider.bind_new()? {
            state
                .lease_refs
                .lock()
                .unwrap()
                .insert(lease.id.clone(), bound);
        }
        Ok(())
    }

    /// Release a lease's store pin. Callers invoke this only once no live
    /// lease row references the bundle (lease forgotten or returned), so
    /// cleanup may reclaim the bundle once it is otherwise unreferenced.
    /// Release failures keep the pin (fail-safe) and never fail teardown.
    pub(crate) fn unpin_lease(state: &State, lease: &Lease) {
        let held = state.lease_refs.lock().unwrap().remove(&lease.id);
        if let Some(mut bound) = held {
            crate::providers::resolution::release_bound(&mut bound);
        }
    }

    pub(crate) fn owned(state: &State, owner: i64, id: &str) -> Result<Connection, Error> {
        let connection = state.store.connection(id)?;
        if owner <= 0 || connection.owner_id != owner {
            return Err(Error::denied("identity authority denied"));
        }
        Ok(connection)
    }

    pub fn connections(&self, owner: i64) -> Result<Vec<u8>, Error> {
        if owner <= 0 {
            return Err(Error::denied("identity authority denied"));
        }
        self.lock().store.connections(owner)
    }

    pub fn available(&self, actor: i64, project: &str) -> Result<Vec<u8>, Error> {
        if actor <= 0 || project.is_empty() {
            return Err(Error::denied("identity authority denied"));
        }
        self.lock().store.available(actor, project)
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

pub(crate) fn join_errors(failures: Vec<Error>) -> Result<(), Error> {
    if failures.is_empty() {
        return Ok(());
    }
    let messages: Vec<String> = failures.iter().map(|e| e.to_string()).collect();
    Err(Error::internal(messages.join("; ")))
}

pub(crate) struct LeaseFailures(u64);

impl LeaseFailures {
    pub(crate) fn new() -> Self {
        Self(0)
    }
    pub(crate) fn record(&mut self, result: Result<(), Error>) {
        if result.is_err() {
            self.0 = self.0.saturating_add(1);
        }
    }
    pub(crate) fn finish(self) -> Result<(), Error> {
        if self.0 == 0 {
            Ok(())
        } else {
            Err(Error::internal(format!(
                "lease retirement incomplete ({} failures)",
                self.0
            )))
        }
    }
}

pub(crate) fn new_id() -> String {
    new_id_with(|out| getrandom::fill(out).map_err(std::io::Error::other))
}

fn new_id_with(fill: impl FnOnce(&mut [u8]) -> std::io::Result<()>) -> String {
    let mut bytes = [0u8; 16];
    fill(&mut bytes).expect("identity randomness unavailable");
    crate::providers::sha256::hex(&bytes)
}

pub(crate) fn zeroize(data: &mut [u8]) {
    for byte in data.iter_mut() {
        unsafe { std::ptr::write_volatile(byte, 0) };
    }
}
