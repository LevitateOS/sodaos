// Subscription admission and credential custody, mirroring
// internal/identity/control: enrollment sessions, grants, leases,
// execution fencing and native termination.
use crate::store::Store;
use crate::wire::{provider_valid, Connection, Enrollment, Error, Lease};
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
}

impl Provider for MuseProvider {
    fn start(&self, owner: i64) -> Result<Box<dyn EnrollmentSession>, Error> {
        Ok(Box::new(MuseSession(
            self.0.start(owner).map_err(map_provider_error)?,
        )))
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
    pub(crate) providers: HashMap<String, Box<dyn Provider>>,
    pub(crate) runtime: Box<dyn Runtime>,
    pub(crate) enrollments: HashMap<String, EnrollmentEntry>,
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

    pub(crate) fn lock(&self) -> std::sync::MutexGuard<'_, State> {
        self.state.lock().unwrap()
    }

    pub(crate) fn owned(state: &State, owner: i64, id: &str) -> Result<Connection, Error> {
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn identity_id_fails_closed_after_partial_entropy_write() {
        let result = std::panic::catch_unwind(|| {
            new_id_with(|out| {
                out[0] = 1;
                Err(std::io::Error::other("injected entropy failure"))
            });
        });
        assert!(result.is_err());
    }
}
