// Real daemon backend: `ExecBackend` over the ported executors (PR26).
//
// Each trait method strict-decodes its JSON body (replacing
// `strictjson.Decode`), enforces the
// daemon-level pre-checks from `dispatch*` in internal/host/daemon.go, calls
// the executor, and encodes the response. Mutation gating lives in the mux,
// which holds the admission guard across backend calls: this backend MUST
// NEVER acquire the gate itself (the gate is a blocking non-reentrant
// mutex; re-acquiring would deadlock).
//
// Per-request horizons mirror Go: 3-minute native timeout, 20s tailnet
// timeout, terminal lifetime from the request expiry. Bodies arrive
// pre-limited by the mux; over-limit bodies fail here as decode errors,
// exactly like Go's `MaxBytesReader` surfacing through strict decode.
//
use std::sync::atomic::AtomicU64;
use std::sync::OnceLock;
use std::time::{Duration, Instant};

use crate::daemon::backend::BackendError;
use crate::{factory, json, project, tcontrol, terminal};

mod backend;
mod factory_terminal;
mod tailnet;
#[cfg(test)]
pub(in crate::dbackend) mod test_support;
mod websocket;

// -- error mapping (daemon.go ServeHTTP + subsystem handlers) --

/// Map an unclassified executor error to Go's default internal failure.
fn internal(_err: String) -> BackendError {
    BackendError::Internal
}

/// `strictjson.Decode` failure inside a native dispatch: Go returns the
/// error and ServeHTTP renders 500.
/// Empty-object body (`dispatchProfile`, `/factory-harness`): Go decodes
/// into `struct{}`, so `{}` (and JSON `null`, which decodes into any Go
/// value) passes and anything else fails.
fn decode_empty(body: &[u8]) -> Result<(), BackendError> {
    decode_empty_or_null(body).map_err(|e| internal(e))
}

fn decode_empty_or_null(body: &[u8]) -> Result<(), String> {
    use serde::de::{self, MapAccess, Visitor};
    use serde::Deserialize;
    struct Empty;
    impl<'de> Deserialize<'de> for Empty {
        fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
        where
            D: serde::Deserializer<'de>,
        {
            struct EmptyVisitor;
            impl<'de> Visitor<'de> for EmptyVisitor {
                type Value = Empty;
                fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                    f.write_str("null or an empty object")
                }
                fn visit_unit<E>(self) -> Result<Self::Value, E>
                where
                    E: de::Error,
                {
                    Ok(Empty)
                }
                fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
                where
                    A: MapAccess<'de>,
                {
                    if map.next_key::<String>()?.is_some() {
                        return Err(de::Error::custom("expected an empty object"));
                    }
                    Ok(Empty)
                }
            }
            deserializer.deserialize_any(EmptyVisitor)
        }
    }
    if body.len() > json::MAXIMUM_REQUEST_BYTES {
        return Err("request exceeds 1 MiB".to_string());
    }
    let mut input = serde_json::Deserializer::from_slice(body);
    Empty::deserialize(&mut input).map_err(|e| e.to_string())?;
    input.end().map_err(|e| e.to_string())
}

/// `FactoryError` to wire mapping. `stale` is per-method: output reads map
/// `Stale` to `OutputStale`, export/candidate reads to `ExportStale`.
fn map_factory_err(err: factory::FactoryError, stale: BackendError) -> BackendError {
    match err {
        factory::FactoryError::NotFound => BackendError::NotFound,
        factory::FactoryError::Stale => stale,
        factory::FactoryError::ExportCandidate => BackendError::ExportCandidate,
        factory::FactoryError::ExportBounds => BackendError::ExportBounds,
        factory::FactoryError::Msg(m) => internal(m),
        // Busy, Denied, Uncertain, DeadlineExceeded: Go renders every other
        // factory error as 500 (or 409 identity-unconfirmed behind the
        // identity routes, which the mux owns).
        _ => BackendError::Internal,
    }
}

fn native_deadline() -> Instant {
    Instant::now() + Duration::from_secs(180)
}

fn tailnet_deadline() -> Instant {
    Instant::now() + Duration::from_secs(20)
}

// -- backend --

/// Inputs the binary derives from the daemon config. Kept minimal: the mux
/// owns transport admission, the runtimes own execution.
pub struct BackendConfig {
    pub project: project::Config,
    pub codex_harness: String,
    pub codex_harness_sha256: String,
    pub codex_harness_version: String,
    pub muse_harness: String,
    pub muse_harness_sha256: String,
    pub muse_harness_version: String,
    pub broker_socket: String,
    pub tailnet_image: String,
    /// `Some` exactly when the operator configured a muse release digest
    /// (Go wires `d.Muse` only then).
    pub muse: Option<MuseConfig>,
}

pub struct MuseConfig {
    pub version: String,
    pub sha256: String,
}

/// Terminal side of supervised factory runs. Delegates to the `tcodex`
/// service operations over the shared native executor shape.
pub struct TerminalSeam {
    harness: String,
    harness_version: String,
    harness_sha256: String,
    muse_harness: String,
    muse_harness_version: String,
    muse_harness_sha256: String,
}

type Factory = factory::Factory<project::Native, TerminalSeam, crate::iclient::BrokerClient>;

/// The real daemon backend. All runtimes share one native executor; the
/// factory opens lazily like Go's `factoryRun` so a broken receipt root
/// degrades factory routes to 503 instead of refusing startup.
pub struct DaemonBackend {
    project: project::Runtime<project::Native>,
    terminal: terminal::Service<project::Native>,
    factory: OnceLock<Result<Factory, String>>,
    broker: crate::iclient::BrokerClient,
    tailnet: tcontrol::Control<project::Native>,
    companion:
        crate::tailnet_companion::Companion<project::Native, tcontrol::Control<project::Native>>,
    pops: crate::pops::Ops<project::Native>,
    muse: Option<crate::muse::MuseRuntime<project::Native, crate::iclient::BrokerClient>>,
    codex_harness: String,
    broker_socket: String,
    factory_state_dir: String,
    next_session: AtomicU64,
}

impl DaemonBackend {
    pub fn open(cfg: BackendConfig, factory_state_dir: &str) -> Self {
        let broker_socket = cfg.broker_socket.clone();
        DaemonBackend {
            project: project::Runtime {
                exec: project::Native,
                config: cfg.project.clone(),
            },
            terminal: terminal::Service {
                exec: project::Native,
                codex_harness: cfg.codex_harness.clone(),
                codex_harness_sha256: cfg.codex_harness_sha256.clone(),
                codex_harness_version: cfg.codex_harness_version.clone(),
                muse_harness: cfg.muse_harness.clone(),
                muse_harness_sha256: cfg.muse_harness_sha256.clone(),
                muse_harness_version: cfg.muse_harness_version.clone(),
            },
            factory: OnceLock::new(),
            broker: crate::iclient::BrokerClient::new(&broker_socket),
            tailnet: tcontrol::Control::new(project::Native, tcontrol::Options::default()),
            companion: crate::tailnet_companion::Companion {
                exec: project::Native,
                tailnet: tcontrol::Control::new(project::Native, tcontrol::Options::default()),
                image: cfg.tailnet_image.clone(),
                enabled_check: None,
            },
            pops: crate::pops::Ops {
                exec: project::Native,
                config: cfg.project,
            },
            muse: cfg.muse.map(|m| {
                crate::muse::MuseRuntime::new(
                    project::Native,
                    crate::iclient::BrokerClient::new(&broker_socket),
                    m.version,
                    m.sha256,
                )
            }),
            codex_harness: cfg.codex_harness,
            broker_socket: broker_socket.clone(),
            factory_state_dir: factory_state_dir.to_string(),
            next_session: AtomicU64::new(1),
        }
    }

    /// Go `factoryRun`: lazily open supervised run orchestration over the
    /// protected receipt root. Failures (including a missing root) report
    /// `Unavailable`, matching Go's 503.
    /// Companion phase helpers for the `--tailnet-action` CLI surface
    /// (Go `runTailnetAction` over `NewCompanionDaemon`).
    pub fn companion_start(&self, project: &str, deadline: Instant) -> Result<String, String> {
        self.companion.start_tailnet(project, deadline)
    }

    pub fn companion_wait(&self, cid: &str, deadline: Instant) -> Result<(), String> {
        self.companion.wait_tailnet(cid, deadline)
    }

    pub fn companion_stop(&self, project: &str, deadline: Instant) -> Result<(), String> {
        self.companion.stop_tailnet(project, deadline)
    }

    fn factory(&self) -> Result<&Factory, BackendError> {
        self.factory
            .get_or_init(|| {
                // Go: `MkdirAll(root, 0o700)`. Rust's create_dir_all applies
                // umask modes, so enforce the private mode explicitly before
                // the open check below (which rejects group/other access).
                std::fs::create_dir_all(&self.factory_state_dir).map_err(|e| e.to_string())?;
                std::fs::set_permissions(
                    &self.factory_state_dir,
                    std::os::unix::fs::PermissionsExt::from_mode(0o700),
                )
                .map_err(|e| e.to_string())?;
                Factory::open_factory(
                    &self.factory_state_dir,
                    project::Native,
                    TerminalSeam {
                        harness: self.terminal.codex_harness.clone(),
                        harness_version: self.terminal.codex_harness_version.clone(),
                        harness_sha256: self.terminal.codex_harness_sha256.clone(),
                        muse_harness: self.terminal.muse_harness.clone(),
                        muse_harness_version: self.terminal.muse_harness_version.clone(),
                        muse_harness_sha256: self.terminal.muse_harness_sha256.clone(),
                    },
                    crate::iclient::BrokerClient::new(&self.broker_socket),
                )
            })
            .as_ref()
            .map_err(|_| BackendError::Unavailable)
    }
}

impl terminal::IdentityBroker for DaemonBackend {
    fn acquire(
        &self,
        req: &terminal::AcquireRequest,
        deadline: Instant,
    ) -> Result<terminal::Lease, String> {
        self.broker.acquire(req, deadline)
    }
    fn register(
        &self,
        lease_id: &str,
        binding: &terminal::Binding,
        deadline: Instant,
    ) -> Result<terminal::Delivery, String> {
        self.broker.register(lease_id, binding, deadline)
    }
    fn reconcile_lease(&self, lease_id: &str) -> Result<(), String> {
        // Go passes `context.Background` (no deadline); reconcile failures
        // are ignored by every caller, so a bounded horizon only converts a
        // wedged-broker hang into an ignored error.
        self.broker.reconcile_lease(lease_id, native_deadline())
    }
}
