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
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::os::unix::net::UnixStream;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, OnceLock};
use std::time::{Duration, Instant};

use crate::daemon::backend::{BackendError, ExecBackend, TerminalSession};
use crate::daemon::broker::cv_lease_to_texec;
use crate::{
    domain, factory, json, project, tcontrol,
    terminal::{self, factory::tcodex},
};

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

fn cv_run_to_tcodex(r: &factory::FactoryRun) -> crate::terminal::factory::tcodex::FactoryRun {
    crate::terminal::factory::tcodex::FactoryRun {
        deadline_raw: r.deadline.clone(),
        actor: r.actor,
        id: r.id.clone(),
        project: r.project.clone(),
        role: r.role.clone(),
        preparation: r.preparation.clone(),
        harness: r.harness.clone(),
        harness_vers: r.harness_vers.clone(),
        model: r.model.clone(),
        assignment: r.assignment.clone(),
        source_commit: r.source_commit.clone(),
        connection: r.connection.clone(),
    }
}

fn cv_slice_to_factory(
    s: &crate::terminal::factory::tcodex::FactoryCodexOutputSlice,
) -> factory::OutputSlice {
    factory::OutputSlice {
        data: s.data.clone(),
        total: s.total,
        offset: s.offset,
        truncated: s.truncated,
        gap: s.gap,
    }
}

// -- seam implementations --

impl factory::FactoryTerminal for TerminalSeam {
    fn harness_family(&self) -> String {
        self.pin_family().to_string()
    }
    fn harness_version(&self) -> String {
        if self.pin_family() == factory::FACTORY_HARNESS_MUSE {
            self.muse_harness_version.clone()
        } else {
            self.harness_version.clone()
        }
    }
    fn harness_sha256(&self) -> String {
        if self.pin_family() == factory::FACTORY_HARNESS_MUSE {
            self.muse_harness_sha256.clone()
        } else {
            self.harness_sha256.clone()
        }
    }
    fn reserve(
        &self,
        run: &factory::FactoryRun,
        lease: &factory::Lease,
        pin: &str,
        max_secs: i64,
        deadline: Instant,
    ) -> Result<terminal::Binding, factory::FactoryError> {
        let service = self.service();
        let trun = cv_run_to_tcodex(run);
        let tlease = cv_lease_to_texec(lease);
        let out = if run.harness == factory::FACTORY_HARNESS_MUSE {
            service
                .factory_muse_reserve(&trun, &tlease, pin, max_secs, deadline)
                .map(|(binding, _paths)| binding)
        } else {
            service
                .factory_codex_reserve(&trun, &tlease, pin, max_secs, deadline)
                .map(|(binding, _paths)| binding)
        };
        out.map_err(factory::FactoryError::Msg)
    }
    fn start(
        &self,
        lease: &factory::Lease,
        credential: &[u8],
        prompt: &[u8],
        deadline: Instant,
    ) -> Result<(), factory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        if muse_scoped_lease(&tlease) {
            service
                .factory_muse_start(&tlease, credential, prompt, deadline)
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_start(&tlease, credential, prompt, deadline)
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn wait(
        &self,
        lease: &factory::Lease,
        deadline: Instant,
    ) -> Result<(i64, String), factory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        if muse_scoped_lease(&tlease) {
            service
                .factory_muse_wait(&tlease, deadline)
                .map(|(code, out)| (code as i64, out))
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_wait(&tlease, deadline)
                .map(|(code, out)| (code as i64, out))
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn stop(&self, lease: &factory::Lease, deadline: Instant) -> Result<(), factory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        if muse_scoped_lease(&tlease) {
            service
                .factory_muse_stop(&tlease, deadline)
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_stop(&tlease, deadline)
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn stop_unbound(
        &self,
        run: &factory::FactoryRun,
        deadline: Instant,
    ) -> Result<(), factory::FactoryError> {
        let service = self.service();
        let trun = cv_run_to_tcodex(run);
        if run.harness == factory::FACTORY_HARNESS_MUSE {
            service
                .factory_muse_stop_unbound(&trun, deadline)
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_stop_unbound(&trun, deadline)
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn capture(
        &self,
        lease: &factory::Lease,
        deadline: Instant,
    ) -> Result<Vec<u8>, factory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        if muse_scoped_lease(&tlease) {
            service
                .factory_muse_capture(&tlease, deadline)
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_capture(&tlease, deadline)
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn live(&self, binding: &terminal::Binding, deadline: Instant) -> bool {
        let service = self.service();
        if binding.scope == tcodex::FACTORY_SCOPE_MUSE {
            service.factory_muse_live(binding, deadline)
        } else {
            service.factory_codex_live(binding, deadline)
        }
    }
    fn output(
        &self,
        project: &str,
        binding: &terminal::Binding,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<factory::OutputSlice, factory::FactoryError> {
        let service = self.service();
        if binding.scope == tcodex::FACTORY_SCOPE_MUSE {
            service
                .factory_muse_output(project, binding, offset, limit, deadline)
                .map(|s| cv_slice_to_factory(&s))
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_output(project, binding, offset, limit, deadline)
                .map(|s| cv_slice_to_factory(&s))
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn takeover_copy(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        member: &str,
        run: &str,
        deadline: Instant,
    ) -> Result<(String, bool), factory::FactoryError> {
        let service = self.service();
        service
            .factory_takeover_copy(project, recorded, role, preparation, member, run, deadline)
            .map_err(factory::FactoryError::Msg)
    }
    fn export_bundle(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        candidate: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, factory::FactoryError> {
        let service = self.service();
        service
            .factory_export_bundle(project, recorded, role, preparation, candidate, deadline)
            .map_err(factory::FactoryError::Msg)
    }
}

/// A lease takes the muse path when its binding carries the factory-muse
/// scope. Anything else (including unbound) stays on the codex path,
/// which denies what it does not recognize.
fn muse_scoped_lease(lease: &terminal::Lease) -> bool {
    lease
        .binding
        .as_ref()
        .is_some_and(|b| b.scope == tcodex::FACTORY_SCOPE_MUSE)
}

impl TerminalSeam {
    /// The seam owns no service state: it rebuilds the value service (pure
    /// configuration + the shared executor shape) per call. Service methods
    /// take `&self` and hold no interior state, so this is free.
    fn service(&self) -> terminal::Service<project::Native> {
        terminal::Service {
            exec: project::Native,
            codex_harness: self.harness.clone(),
            codex_harness_sha256: self.harness_sha256.clone(),
            codex_harness_version: self.harness_version.clone(),
            muse_harness: self.muse_harness.clone(),
            muse_harness_sha256: self.muse_harness_sha256.clone(),
            muse_harness_version: self.muse_harness_version.clone(),
        }
    }

    /// The pin advertises the codex harness when configured, else the
    /// muse harness when configured, else nothing usable.
    fn pin_family(&self) -> &'static str {
        if !self.harness.is_empty() {
            factory::FACTORY_HARNESS_CODEX
        } else if !self.muse_harness.is_empty() {
            factory::FACTORY_HARNESS_MUSE
        } else {
            factory::FACTORY_HARNESS_CODEX
        }
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

// -- project routes (daemon.go dispatchProfile/Create/Targeted/Mutation) --

impl DaemonBackend {
    /// Shared `/inspect`, `/os`, `/connection` prelude: decode a creation
    /// identity but validate the ID only (Go checks `ValidID` alone here,
    /// with the "invalid project identity" message, not full `Validate`).
    fn targeted_id(&self, body: &[u8]) -> Result<String, BackendError> {
        let id = domain::Create::decode(body)
            .map(|c| c.id)
            .map_err(internal)?;
        if !domain::valid_id(&id) {
            return Err(internal("invalid project identity".to_string()));
        }
        Ok(id)
    }
}

impl ExecBackend for DaemonBackend {
    fn profile(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        decode_empty(body)?;
        let out = self
            .project
            .resolve_profile(native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn create(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        // The mux holds the writer gate for /create; see module docs.
        let input = domain::Create::decode(body).map_err(internal)?;
        // Go checks identity before Validate, with its own message.
        if !domain::valid_id(&input.id) || input.owner <= 0 {
            return Err(internal("invalid project identity".to_string()));
        }
        input.validate().map_err(internal)?;
        let out = self
            .project
            .create(&input, native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let id = self.targeted_id(body)?;
        // Go returns the environment and drops the generation counter.
        let (env, _) = self
            .project
            .inspect(&id, native_deadline())
            .map_err(internal)?;
        Ok(env.encode().into_bytes())
    }

    fn observe_os(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let id = self.targeted_id(body)?;
        let out = self
            .project
            .observe_os(&id, native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn connection(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let id = self.targeted_id(body)?;
        let out = self
            .project
            .connection(&id, native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn lifecycle(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let input = project::Lifecycle::decode(body).map_err(internal)?;
        let out = self
            .project
            .lifecycle(&input, native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn access_keys(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::AccessKeysReq::decode(body).map_err(internal)?;
        self.pops
            .access_keys(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn account(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::AccountReq::decode(body).map_err(internal)?;
        self.pops
            .account(&req, native_deadline())
            .map_err(internal)?;
        // Go's `dispatchMutation` synthesizes this; there is no backend value.
        Ok(b"{\"ok\":true}".to_vec())
    }

    fn project_access(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = domain::ProjectAccessRequest::decode(body).map_err(internal)?;
        self.pops
            .project_access(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn prepare(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::PrepareReq::decode(body).map_err(internal)?;
        self.pops
            .prepare(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn prepare_candidate(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::PrepareCandidateReq::decode(body).map_err(internal)?;
        self.pops
            .prepare_candidate(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn inspect_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::InspectPreparationReq::decode(body).map_err(internal)?;
        self.pops
            .inspect_preparation(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn stop_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::StopPreparationReq::decode(body).map_err(internal)?;
        self.pops
            .stop_preparation(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn hold_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::HoldPreparationReq::decode(body).map_err(internal)?;
        self.pops
            .hold_preparation(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn factory_launch(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        // Factory routes stay off the shared mutation gate (Go comment):
        // per-run locks plus broker serialization are the control.
        let factory = self.factory()?;
        let req = factory::FactoryLaunch::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .launch(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryInspect::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .inspect(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_stop(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryStop::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .stop(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_takeover(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryTakeover::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .takeover(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_output(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryOutput::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .output(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::OutputStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_harness(&self, body: &[u8], image: &str) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        decode_empty(body)?;
        // Go stamps the daemon image onto the pin before validation.
        let mut pin = factory.harness_pin();
        pin.image = image.to_string();
        pin.validate().map_err(|_| BackendError::Unavailable)?;
        Ok(pin.encode().into_bytes())
    }

    fn factory_export(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryExport::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .export(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_candidate_inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryCandidateInspect::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .inspect_candidate(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn identity_launch(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        // The mux maps every non-501 identity error to 409, like Go's
        // `identityHandler`.
        let input = terminal::TerminalStart::decode(body).map_err(internal)?;
        let lease = terminal::identity_launch(
            self,
            &self.terminal,
            &input,
            !self.codex_harness.is_empty(),
            native_deadline(),
        )
        .map_err(internal)?;
        Ok(lease.encode().into_bytes())
    }

    fn identity_action(&self, action: &str, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let delivery = terminal::Delivery::decode(body).map_err(internal)?;
        if delivery.lease.kind == terminal::KIND_FACTORY {
            // Broker validate/stop/finish callbacks for supervised runs.
            let out = self
                .terminal
                .factory_identity_operation(action, &delivery, native_deadline())
                .map_err(internal)?;
            return Ok(out.encode().into_bytes());
        }
        let muse_scoped = delivery.lease.provider_id == terminal::PROVIDER_MUSE
            && delivery
                .lease
                .binding
                .as_ref()
                .is_some_and(|b| b.scope == "muse-project");
        if muse_scoped {
            if let Some(m) = &self.muse {
                let out = m
                    .muse(action, &delivery, native_deadline())
                    .map_err(internal)?;
                return Ok(out.encode().into_bytes());
            }
            // Muse-scoped without a muse runtime falls through to the
            // terminal identity path, which denies it (Go parity).
        }
        let out = self
            .terminal
            .identity(action, &delivery, native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn tailnet(&self, action: &str, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let deadline = tailnet_deadline();
        match action {
            "settings" => {
                decode_tailnet_empty(body)?;
                self.tailnet.settings(deadline).map_err(map_tailnet_err)
            }
            "options" => {
                decode_tailnet_empty(body)?;
                self.tailnet.options(deadline).map_err(map_tailnet_err)
            }
            "host" => self
                .tailnet
                .host_action(body, deadline)
                .map_err(map_tailnet_err),
            "enrollment" => self
                .tailnet
                .enrollment(body, deadline)
                .map_err(map_tailnet_err),
            "project" | "policy" => self.tailnet_project_or_policy(action, body, deadline),
            _ => Err(BackendError::Invalid),
        }
    }

    fn terminal_accept(&self, _key: &str) -> Result<TerminalSession, BackendError> {
        // Go asserts the executor provides the attach launcher and
        // registers the stream before the 101. The mux now owns stream
        // registration (TerminalSlot), and native attach is always
        // available in this backend, so admission always succeeds; the
        // session id correlates pump logging.
        Ok(TerminalSession {
            id: self.next_session.fetch_add(1, Ordering::SeqCst),
        })
    }

    fn pump_terminal(
        &self,
        stream: tungstenite::protocol::WebSocket<UnixStream>,
        _session: TerminalSession,
        shutdown: Arc<std::sync::atomic::AtomicBool>,
    ) -> Result<(), BackendError> {
        pump_terminal(&self.terminal, self, stream, shutdown).map_err(internal)
    }
}

// -- tailnet helpers (tailnet.go dispatchTailnetAction) --

/// Tailnet decode failure: Go's `decodeTailnetBody` reports `ErrInvalid`
/// (400), unlike the native 500.
fn decode_tailnet_empty(body: &[u8]) -> Result<(), BackendError> {
    decode_empty_or_null(body).map_err(|_| BackendError::Invalid)
}

/// Lane T error substrings to wire mapping. The mux renders tailnet
/// `ExportStale` as 409 and `ExportCandidate` as 422, so conflict and
/// unsupported reuse those variants behind this subsystem only.
fn map_tailnet_err(err: String) -> BackendError {
    if err.contains("invalid request") {
        return BackendError::Invalid;
    }
    if err.contains("conflict") {
        return BackendError::ExportStale;
    }
    if err.contains("unsupported") {
        return BackendError::ExportCandidate;
    }
    if err.contains("unavailable") {
        return BackendError::Unavailable;
    }
    BackendError::Internal
}

/// `^[0-9a-f]{32}$` or the `"0"` unset marker (Go `validRevision`).
fn valid_revision(v: &str) -> bool {
    v == "0"
        || (v.len() == 32
            && v.bytes()
                .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()))
}

#[derive(Default)]
struct TailnetProjectWire {
    project: Option<String>,
    action: Option<String>,
    revision: Option<String>,
    binding: Option<String>,
    confirm_id: Option<String>,
}
impl<'de> Deserialize<'de> for TailnetProjectWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = TailnetProjectWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("Tailnet project request")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = TailnetProjectWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    match k.to_ascii_lowercase().as_str() {
                        "project" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.project = Some(v)
                            }
                        }
                        "action" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.action = Some(v)
                            }
                        }
                        "revision" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.revision = Some(v)
                            }
                        }
                        "binding" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.binding = Some(v)
                            }
                        }
                        "confirm_id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.confirm_id = Some(v)
                            }
                        }
                        _ => {
                            return Err(de::Error::unknown_field(
                                &k,
                                &["project", "action", "revision", "binding", "confirm_id"],
                            ))
                        }
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

/// Strict decode of one project/policy request (Go `ProjectRequest` shape).
fn decode_project_request(
    body: &[u8],
) -> Result<crate::tailnet_domain::ProjectRequest, BackendError> {
    let m: TailnetProjectWire = json::decode_strict_as(body).map_err(|_| BackendError::Invalid)?;
    Ok(crate::tailnet_domain::ProjectRequest {
        project: m.project.unwrap_or_default(),
        action: m.action.unwrap_or_default(),
        revision: m.revision.unwrap_or_default(),
        binding: m.binding.unwrap_or_default(),
        confirm_id: m.confirm_id.unwrap_or_default(),
    })
}

/// Go `ProjectRequest.Validate` + `validateProjectMutation`, verbatim.
fn validate_project_request(
    req: &crate::tailnet_domain::ProjectRequest,
) -> Result<(), BackendError> {
    if !crate::tailnet_domain::valid_project_id(&req.project) {
        return Err(BackendError::Invalid);
    }
    if req.action == "inspect" {
        if !req.revision.is_empty() || !req.binding.is_empty() || !req.confirm_id.is_empty() {
            return Err(BackendError::Invalid);
        }
        return Ok(());
    }
    if !valid_revision(&req.revision) || req.confirm_id != req.project {
        return Err(BackendError::Invalid);
    }
    match req.action.as_str() {
        "disable" => {
            if !req.binding.is_empty() {
                return Err(BackendError::Invalid);
            }
        }
        "enable" | "retry" => {
            if !(req.binding.len() == 32
                && req
                    .binding
                    .bytes()
                    .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()))
            {
                return Err(BackendError::Invalid);
            }
        }
        _ => return Err(BackendError::Invalid),
    }
    Ok(())
}

/// Go `ProjectView` JSON render with identical `omitempty` behavior.
fn encode_project_view(view: &crate::tailnet_domain::ProjectView) -> String {
    let mut out = String::from("{");
    let mut first = true;
    let field = |out: &mut String, first: &mut bool, name: &str, value: String| {
        if !*first {
            out.push(',');
        }
        *first = false;
        out.push_str(&json::quote(name));
        out.push(':');
        out.push_str(&value);
    };
    if !view.available_binding.is_empty() {
        field(
            &mut out,
            &mut first,
            "available_binding",
            json::quote(&view.available_binding),
        );
    }
    if !view.available_network.is_empty() {
        field(
            &mut out,
            &mut first,
            "available_network",
            json::quote(&view.available_network),
        );
    }
    if !view.addresses.is_empty() {
        let list = view
            .addresses
            .iter()
            .map(|a| json::quote(a))
            .collect::<Vec<_>>()
            .join(",");
        field(&mut out, &mut first, "addresses", format!("[{list}]"));
    }
    if !view.dns_name.is_empty() {
        field(
            &mut out,
            &mut first,
            "dns_name",
            json::quote(&view.dns_name),
        );
    }
    field(
        &mut out,
        &mut first,
        "saved",
        if view.saved {
            "true".to_string()
        } else {
            "false".to_string()
        },
    );
    field(&mut out, &mut first, "project", json::quote(&view.project));
    field(
        &mut out,
        &mut first,
        "revision",
        json::quote(&view.revision),
    );
    field(&mut out, &mut first, "binding", json::quote(&view.binding));
    field(
        &mut out,
        &mut first,
        "enabled",
        if view.enabled {
            "true".to_string()
        } else {
            "false".to_string()
        },
    );
    field(&mut out, &mut first, "state", json::quote(&view.state));
    field(&mut out, &mut first, "outcome", json::quote(&view.outcome));
    out.push('}');
    out
}

impl DaemonBackend {
    /// `executeTailnetProjectOrPolicy`: strict decode, validation, the
    /// container-identity fence (resolve before AND after; mismatch or
    /// error is unconfirmed), then the observation.
    fn tailnet_project_or_policy(
        &self,
        action: &str,
        body: &[u8],
        deadline: Instant,
    ) -> Result<Vec<u8>, BackendError> {
        let req = decode_project_request(body)?;
        validate_project_request(&req)?;
        if action == "policy" && req.action != "inspect" {
            return Err(BackendError::Invalid);
        }
        let cid = self
            .project
            .project_container(&req.project, false, deadline)
            .map_err(|_| BackendError::Internal)?;
        let out = if action == "policy" {
            self.tailnet
                .project(&req, &cid, deadline)
                .map_err(map_tailnet_err)?
        } else {
            self.companion
                .observe_project_tailnet(&req, &cid, deadline)
                .map_err(map_tailnet_err)?
        };
        let after = self
            .project
            .project_container(&req.project, false, deadline)
            .map_err(|_| BackendError::Internal)?;
        if after != cid {
            return Err(BackendError::Internal);
        }
        Ok(encode_project_view(&out).into_bytes())
    }
}

// -- terminal pump (terminal.go terminalHandler + service.go Handler) --

/// Terminal attach loop. Tungstenite owns RFC6455 framing and the upgraded
/// connection; one thread owns the protocol for the entire session.
fn closed_frame(reason: &str) -> terminal::TerminalFrame {
    terminal::TerminalFrame {
        frame_type: "closed".into(),
        data: String::new(),
        cols: 0,
        rows: 0,
        reason: reason.into(),
        terminals: None,
    }
}

fn pump_terminal(
    service: &terminal::Service<project::Native>,
    backend: &DaemonBackend,
    mut ws: tungstenite::protocol::WebSocket<UnixStream>,
    shutdown: Arc<std::sync::atomic::AtomicBool>,
) -> Result<(), String> {
    use crate::daemon::admission::{TERMINAL_FRAME_LIMIT, TERMINAL_REQUEST_LIMIT};
    use std::os::fd::AsRawFd;
    use std::sync::atomic::Ordering;
    use tungstenite::{Error as WsError, Message};

    let socket_fd = ws.get_ref().as_raw_fd();
    let flags = unsafe { libc::fcntl(socket_fd, libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(socket_fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
        return Err("terminal transport ended".to_string());
    }
    ws.set_config(|config| {
        config.max_message_size = Some(TERMINAL_REQUEST_LIMIT);
        config.max_frame_size = Some(TERMINAL_REQUEST_LIMIT);
    });
    let first_deadline = Instant::now() + Duration::from_secs(5);
    let first = loop {
        if shutdown.load(Ordering::Acquire) {
            return Ok(());
        }
        let remaining = first_deadline.saturating_duration_since(Instant::now());
        if remaining.is_zero() {
            return Ok(());
        }
        match ws.read() {
            Ok(Message::Text(text)) if text.len() <= TERMINAL_REQUEST_LIMIT => {
                break text.as_bytes().to_vec()
            }
            Ok(Message::Ping(_)) => {
                let _ = ws.flush();
            }
            Ok(Message::Pong(_)) => {}
            Err(WsError::Io(ref e)) if e.kind() == std::io::ErrorKind::WouldBlock => {}
            Ok(_) | Err(_) => return Ok(()),
        }
        let mut pollfd = libc::pollfd {
            fd: socket_fd,
            events: libc::POLLIN,
            revents: 0,
        };
        let wait = first_deadline
            .saturating_duration_since(Instant::now())
            .as_millis()
            .min(100) as i32;
        if unsafe { libc::poll(&mut pollfd, 1, wait.max(1)) } < 0
            && std::io::Error::last_os_error().kind() != std::io::ErrorKind::Interrupted
        {
            return Ok(());
        }
    };
    let request = match terminal::TerminalRequest::decode(&first).ok() {
        Some(request) if request.valid(terminal::now_unix()) => request,
        _ => return Ok(()),
    };
    ws.set_config(|config| {
        config.max_message_size = Some(TERMINAL_FRAME_LIMIT);
        config.max_frame_size = Some(TERMINAL_FRAME_LIMIT);
    });
    let expiry = Instant::now()
        + Duration::from_secs(request.expires.saturating_sub(terminal::now_unix()).max(0) as u64);
    let session_deadline = expiry;
    let operation_deadline = native_deadline().min(session_deadline);
    let inspect_deadline = (Instant::now() + Duration::from_secs(10)).min(operation_deadline);
    let container = match service.project_container(&request.project, true, inspect_deadline) {
        Ok(cid) => cid,
        Err(_) => {
            let _ = ws.send(Message::Text(closed_frame("launch_failed").encode().into()));
            return Ok(());
        }
    };
    let end_hook = |actor: i64, lease_id: &str| {
        backend
            .broker
            .end_lease(actor, lease_id, operation_deadline)
    };
    if service
        .managed_end(&container, &request, Some(&end_hook), operation_deadline)
        .is_err()
    {
        let _ = ws.send(Message::Text(
            closed_frame("cleanup_unconfirmed").encode().into(),
        ));
        return Ok(());
    }
    if shutdown.load(Ordering::Acquire) {
        return Ok(());
    }
    // NativeAttach::attach is a synchronous spawn outside the shared
    // one-shot deadline; shutdown custody for that child remains separate.
    let attach = match terminal::NativeAttach::attach(&container, &request, Arc::clone(&shutdown)) {
        Ok(attach) => attach,
        Err(_) => {
            let _ = ws.send(Message::Text(closed_frame("launch_failed").encode().into()));
            return Ok(());
        }
    };
    pump_attached(ws, attach, session_deadline, shutdown)
}

fn pump_attached<S>(
    mut ws: tungstenite::protocol::WebSocket<S>,
    mut attach: terminal::NativeAttach,
    deadline: Instant,
    shutdown: Arc<std::sync::atomic::AtomicBool>,
) -> Result<(), String>
where
    S: std::io::Read + std::io::Write + std::os::fd::AsRawFd + Send + 'static,
{
    use crate::daemon::admission::TERMINAL_FRAME_LIMIT;
    use std::io::Read;
    use std::io::Write;
    use std::os::fd::AsRawFd;
    use std::sync::atomic::{AtomicBool, Ordering};
    use std::sync::mpsc::{self, TryRecvError, TrySendError};
    use tungstenite::{Error as WsError, Message};
    let socket_fd = ws.get_ref().as_raw_fd();
    let flags = unsafe { libc::fcntl(socket_fd, libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(socket_fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
        let close_error = attach.close().err();
        if let Some(error) = close_error {
            attach.retain_child_until_exit();
            return Err(format!("terminal transport ended; {error}"));
        }
        return Err("terminal transport ended".to_string());
    }
    let mut reader = match attach.take_reader() {
        Some(reader) => reader,
        None => {
            let close_error = attach.close().err();
            if let Some(error) = close_error {
                attach.retain_child_until_exit();
                return Err(format!("terminal output ended; {error}"));
            }
            return Err("terminal output ended".to_string());
        }
    };

    // A bounded child-output queue keeps a quiet or slow socket from pinning
    // the stdout reader. Cancellation is polled even if a descendant retains
    // the pipe after the direct child exits.
    let (out_tx, out_rx) = mpsc::sync_channel::<terminal::TerminalFrame>(8);
    let (wake_reader, mut wake_writer) =
        UnixStream::pair().map_err(|_| "terminal transport ended")?;
    wake_reader.set_nonblocking(true).ok();
    wake_writer.set_nonblocking(true).ok();
    let cancel_reader = Arc::new(AtomicBool::new(false));
    let worker_cancel = Arc::clone(&cancel_reader);
    let output = std::thread::spawn(move || {
        let mut line = Vec::with_capacity(1024);
        let mut buf = [0u8; 4096];
        while !worker_cancel.load(Ordering::Acquire) {
            let mut pollfd = libc::pollfd {
                fd: reader.get_ref().as_raw_fd(),
                events: libc::POLLIN,
                revents: 0,
            };
            let ready = unsafe { libc::poll(&mut pollfd, 1, 100) };
            if ready < 0 {
                if std::io::Error::last_os_error().kind() == std::io::ErrorKind::Interrupted {
                    continue;
                } else {
                    break;
                }
            }
            if ready == 0 {
                continue;
            }
            match reader.get_mut().read(&mut buf) {
                Ok(0) => break,
                Ok(n) => {
                    for &byte in &buf[..n] {
                        if byte == b'\n' {
                            if line.last() == Some(&b'\r') {
                                line.pop();
                            }
                            if let Ok(frame) = terminal::parse_output_line(&line) {
                                line.clear();
                                let mut frame = frame;
                                loop {
                                    if worker_cancel.load(Ordering::Acquire) {
                                        return;
                                    }
                                    match out_tx.try_send(frame) {
                                        Ok(()) => {
                                            let _ = wake_writer.write(&[1]);
                                            break;
                                        }
                                        Err(TrySendError::Full(returned)) => {
                                            frame = returned;
                                            std::thread::sleep(Duration::from_millis(5));
                                        }
                                        Err(TrySendError::Disconnected(_)) => return,
                                    }
                                }
                            } else {
                                return;
                            }
                        } else {
                            line.push(byte);
                            if line.len() > 131072 {
                                return;
                            }
                        }
                    }
                }
                Err(e)
                    if e.kind() == std::io::ErrorKind::WouldBlock
                        || e.kind() == std::io::ErrorKind::Interrupted =>
                {
                    continue
                }
                Err(_) => break,
            }
        }
    });

    let deadline = deadline;
    let mut pending_send = false;
    let mut pending_message: Option<Message> = None;
    let mut write_deadline = None;
    let mut closed = false;
    let mut close_received = false;
    // `read()` may have pulled later complete frames into tungstenite's own
    // buffer with the first text request. Drain that before sleeping on fd.
    let mut drain_buffered = true;
    while Instant::now() < deadline && !shutdown.load(Ordering::Acquire) {
        if write_deadline.is_some_and(|until| Instant::now() >= until) {
            break;
        }
        let mut pollfds = [
            libc::pollfd {
                fd: socket_fd,
                events: libc::POLLIN
                    | if pending_send || pending_message.is_some() {
                        libc::POLLOUT
                    } else {
                        0
                    },
                revents: 0,
            },
            libc::pollfd {
                fd: wake_reader.as_raw_fd(),
                events: libc::POLLIN,
                revents: 0,
            },
        ];
        let ready = if drain_buffered && !pending_send && pending_message.is_none() {
            1
        } else {
            unsafe { libc::poll(pollfds.as_mut_ptr(), pollfds.len() as _, 100) }
        };
        if ready < 0 && std::io::Error::last_os_error().kind() != std::io::ErrorKind::Interrupted {
            break;
        }
        if pollfds[0].revents & (libc::POLLERR | libc::POLLHUP | libc::POLLNVAL) != 0 {
            break;
        }
        if pollfds[1].revents & libc::POLLIN != 0 {
            let mut wake_bytes = [0u8; 64];
            let _ = wake_reader_read(&wake_reader, &mut wake_bytes);
        }
        if pending_message.is_none() && !pending_send {
            match out_rx.try_recv() {
                Ok(frame) => {
                    closed = frame.frame_type == "closed" || frame.frame_type == "metadata";
                    pending_message = Some(Message::Text(frame.encode().into()));
                    write_deadline = Some(Instant::now() + Duration::from_secs(5));
                }
                Err(TryRecvError::Disconnected) => break,
                Err(TryRecvError::Empty) => {}
            }
        }
        if pending_send {
            if pollfds[0].revents & libc::POLLOUT != 0 {
                match ws.flush() {
                    Ok(()) => {
                        pending_send = false;
                        if pending_message.is_none() {
                            write_deadline = None;
                        }
                    }
                    Err(WsError::Io(ref e)) if e.kind() == std::io::ErrorKind::WouldBlock => {}
                    Err(_) => break,
                }
            }
        }
        if !pending_send {
            if let Some(message) = pending_message.take() {
                write_deadline.get_or_insert_with(|| Instant::now() + Duration::from_secs(5));
                match ws.write(message) {
                    Ok(()) => pending_send = true,
                    Err(WsError::WriteBufferFull(message)) => {
                        pending_message = Some(*message);
                        pending_send = true;
                    }
                    // tungstenite retains the frame in its write buffer after
                    // an I/O WouldBlock; flush it later without resending it.
                    Err(WsError::Io(ref e))
                        if e.kind() == std::io::ErrorKind::WouldBlock
                            || e.kind() == std::io::ErrorKind::TimedOut =>
                    {
                        pending_send = true
                    }
                    Err(_) => break,
                }
            }
        }
        if (closed || close_received) && !pending_send && pending_message.is_none() {
            break;
        }
        if !drain_buffered && pollfds[0].revents & libc::POLLIN == 0 {
            continue;
        }
        match ws.read() {
            Ok(Message::Text(text)) if text.len() <= TERMINAL_FRAME_LIMIT => {
                let Ok(frame) = terminal::TerminalFrame::decode(text.as_bytes()) else {
                    break;
                };
                if !frame.input_valid() {
                    break;
                }
                let is_close = frame.frame_type == "close";
                if attach.input_frame(&frame).is_err() || is_close {
                    break;
                }
                drain_buffered = true;
            }
            Ok(Message::Ping(_)) => {
                pending_send = true;
                write_deadline.get_or_insert_with(|| Instant::now() + Duration::from_secs(5));
                drain_buffered = true;
            }
            Ok(Message::Pong(_)) => {
                drain_buffered = true;
            }
            Ok(Message::Close(_)) => {
                let _ = ws.close(None);
                pending_send = true;
                write_deadline.get_or_insert_with(|| Instant::now() + Duration::from_secs(5));
                close_received = true;
            }
            Ok(_) => break,
            Err(WsError::Io(ref e)) if e.kind() == std::io::ErrorKind::WouldBlock => {
                drain_buffered = false;
            }
            Err(_) => break,
        }
    }
    cancel_reader.store(true, Ordering::Release);
    drop(out_rx);
    let close_result = attach.close();
    let output_result = output
        .join()
        .map_err(|_| "terminal output reader panicked".to_string());
    let _ = ws.close(None);
    let close_deadline = (Instant::now() + Duration::from_millis(250)).min(deadline);
    while Instant::now() < close_deadline {
        let mut pfd = libc::pollfd {
            fd: socket_fd,
            events: libc::POLLOUT,
            revents: 0,
        };
        let ready = unsafe { libc::poll(&mut pfd, 1, 25) };
        if ready <= 0 {
            continue;
        }
        match ws.flush() {
            Ok(()) => break,
            Err(WsError::Io(ref e)) if e.kind() == std::io::ErrorKind::WouldBlock => continue,
            Err(_) => break,
        }
    }
    match (close_result, output_result) {
        (Err(close_error), Err(output_error)) => {
            // Keep the same pump task as Child owner until wait confirms exit.
            // The host's absolute shutdown deadline bounds this retained task.
            attach.retain_child_until_exit();
            Err(format!("{close_error}; {output_error}"))
        }
        (Err(close_error), Ok(())) => {
            attach.retain_child_until_exit();
            Err(close_error)
        }
        (Ok(()), Err(output_error)) => Err(output_error),
        (Ok(()), Ok(())) => Ok(()),
    }
}

fn wake_reader_read(stream: &UnixStream, buf: &mut [u8]) -> usize {
    use std::io::Read;
    let mut stream = stream;
    stream.read(buf).unwrap_or(0)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::daemon::backend::ExecBackend;

    fn test_config() -> BackendConfig {
        BackendConfig {
            project: project::Config {
                muse_socket: "/run/soda-test-muse.sock".to_string(),
                image: "sha256:0000000000000000000000000000000000000000000000000000000000000000"
                    .to_string(),
                network: "soda-test".to_string(),
                subnet: "10.99.0.0/24".to_string(),
                bridge: "soda-test0".to_string(),
            },
            codex_harness: "/run/soda-test-harness".to_string(),
            codex_harness_sha256:
                "0000000000000000000000000000000000000000000000000000000000000000".to_string(),
            codex_harness_version: "v0".to_string(),
            muse_harness: String::new(),
            muse_harness_sha256: String::new(),
            muse_harness_version: String::new(),
            broker_socket: "/run/soda-test-broker.sock".to_string(),
            tailnet_image: "ghcr.io/test/tailnet:latest".to_string(),
            muse: None,
        }
    }

    /// Backend whose factory root can never open: `create_dir_all` on an
    /// existing file always fails, so factory routes deterministically
    /// report `Unavailable` without touching the filesystem. (This source
    /// file is guaranteed to exist wherever the test builds.)
    fn backend() -> DaemonBackend {
        DaemonBackend::open(test_config(), file!())
    }

    fn backend_with_muse() -> DaemonBackend {
        let mut cfg = test_config();
        cfg.muse = Some(MuseConfig {
            version: "v0".to_string(),
            sha256: "0000000000000000000000000000000000000000000000000000000000000000".to_string(),
        });
        DaemonBackend::open(cfg, file!())
    }

    struct ShortIo {
        inner: UnixStream,
        block_first_write: bool,
    }

    impl std::os::fd::AsRawFd for ShortIo {
        fn as_raw_fd(&self) -> std::os::fd::RawFd {
            self.inner.as_raw_fd()
        }
    }
    impl std::io::Read for ShortIo {
        fn read(&mut self, buf: &mut [u8]) -> std::io::Result<usize> {
            std::io::Read::read(&mut self.inner, buf)
        }
    }
    impl std::io::Write for ShortIo {
        fn write(&mut self, buf: &[u8]) -> std::io::Result<usize> {
            if self.block_first_write {
                self.block_first_write = false;
                return Err(std::io::ErrorKind::WouldBlock.into());
            }
            std::io::Write::write(&mut self.inner, &buf[..buf.len().min(3)])
        }
        fn flush(&mut self) -> std::io::Result<()> {
            std::io::Write::flush(&mut self.inner)
        }
    }

    fn synthetic_attach(
        script: &str,
    ) -> (
        terminal::NativeAttach,
        u32,
        Arc<std::sync::atomic::AtomicBool>,
    ) {
        let child = std::process::Command::new("/bin/sh")
            .arg("-c")
            .arg(script)
            .stdin(std::process::Stdio::piped())
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::null())
            .spawn()
            .expect("start synthetic stdio child");
        let pid = child.id();
        let shutdown = Arc::new(std::sync::atomic::AtomicBool::new(false));
        let attach =
            terminal::NativeAttach::from_child_for_test(child, Arc::clone(&shutdown)).unwrap();
        (attach, pid, shutdown)
    }

    #[test]
    fn production_attached_pump_handles_ping_short_writes_and_flushes_closed_once() {
        use tungstenite::protocol::{Role, WebSocket};
        use tungstenite::Message;
        let (server, client) = UnixStream::pair().unwrap();
        client
            .set_read_timeout(Some(Duration::from_secs(3)))
            .unwrap();
        let server_ws = WebSocket::from_raw_socket(
            ShortIo {
                inner: server,
                block_first_write: true,
            },
            Role::Server,
            None,
        );
        let mut client_ws = WebSocket::from_raw_socket(client, Role::Client, None);
        let (attach, _, shutdown) = synthetic_attach(
            "printf '%s\\n' '{\"type\":\"closed\",\"reason\":\"exited\"}'; read line",
        );
        let pump_shutdown = Arc::clone(&shutdown);
        let pump = std::thread::spawn(move || {
            pump_attached(
                server_ws,
                attach,
                Instant::now() + Duration::from_secs(3),
                pump_shutdown,
            )
        });

        client_ws
            .send(Message::Ping(bytes::Bytes::from_static(b"ping")))
            .unwrap();
        let mut got_pong = false;
        let mut text_count = 0;
        for _ in 0..6 {
            match client_ws.read() {
                Err(tungstenite::Error::ConnectionClosed | tungstenite::Error::AlreadyClosed) => {
                    break
                }
                Err(tungstenite::Error::Io(ref e)) if e.kind() == std::io::ErrorKind::TimedOut => {
                    break
                }
                Err(error) => panic!("server websocket read failed: {error}"),
                Ok(message) => match message {
                    Message::Pong(payload) => {
                        assert_eq!(payload, bytes::Bytes::from_static(b"ping"));
                        got_pong = true;
                    }
                    Message::Text(text) => {
                        assert_eq!(
                            terminal::TerminalFrame::decode(text.as_bytes())
                                .unwrap()
                                .frame_type,
                            "closed"
                        );
                        text_count += 1;
                    }
                    Message::Close(_) => break,
                    other => panic!("unexpected websocket message: {other:?}"),
                },
            }
        }
        assert!(got_pong, "same protocol owner answers Ping");
        assert_eq!(
            text_count, 1,
            "WouldBlock retries must not duplicate a terminal frame"
        );
        pump.join().unwrap().unwrap();
    }

    #[test]
    fn production_attached_pump_cancels_full_output_queue_and_reaps_child() {
        use tungstenite::protocol::{Role, WebSocket};
        let (server, _client) = UnixStream::pair().unwrap();
        let server_ws = WebSocket::from_raw_socket(server, Role::Server, None);
        let (attach, pid, shutdown) = synthetic_attach(
            "i=0; while [ $i -lt 20000 ]; do printf '%s\\n' '{\"type\":\"output\",\"data\":\"YQ==\"}'; i=$((i+1)); done; exec /bin/sleep 60",
        );
        let pump_shutdown = Arc::clone(&shutdown);
        let (done_tx, done_rx) = std::sync::mpsc::channel();
        let pump = std::thread::spawn(move || {
            let result = pump_attached(
                server_ws,
                attach,
                Instant::now() + Duration::from_secs(10),
                pump_shutdown,
            );
            let _ = done_tx.send(result);
        });
        std::thread::sleep(Duration::from_millis(200));
        shutdown.store(true, Ordering::Release);
        done_rx
            .recv_timeout(Duration::from_secs(5))
            .expect("cancelled pump joins despite slow peer")
            .unwrap();
        pump.join().unwrap();
        assert_eq!(
            unsafe { libc::kill(pid as i32, 0) },
            -1,
            "NativeAttach must kill and reap its direct child"
        );
    }

    #[test]
    fn production_attached_pump_expires_stalled_websocket_write() {
        use std::os::fd::AsRawFd;
        use tungstenite::protocol::{Role, WebSocket};
        let (server, client) = UnixStream::pair().unwrap();
        let small_buffer: libc::c_int = 4096;
        assert_eq!(
            unsafe {
                libc::setsockopt(
                    server.as_raw_fd(),
                    libc::SOL_SOCKET,
                    libc::SO_SNDBUF,
                    &small_buffer as *const _ as *const libc::c_void,
                    std::mem::size_of_val(&small_buffer) as libc::socklen_t,
                )
            },
            0
        );
        let (attach, pid, shutdown) = synthetic_attach(
            "i=0; while [ $i -lt 20000 ]; do printf '%s\\n' '{\"type\":\"output\",\"data\":\"YQ==\"}'; i=$((i+1)); done; exec /bin/sleep 60",
        );
        let ws = WebSocket::from_raw_socket(server, Role::Server, None);
        let started = Instant::now();
        pump_attached(
            ws,
            attach,
            Instant::now() + Duration::from_secs(9),
            shutdown,
        )
        .unwrap();
        let elapsed = started.elapsed();
        assert!(
            elapsed >= Duration::from_millis(4500),
            "stalled peer should consume the 5s write budget: {elapsed:?}"
        );
        assert!(
            elapsed < Duration::from_secs(10),
            "5s write budget plus 3s child grace and final flush should bound the pump: {elapsed:?}"
        );
        assert_eq!(
            unsafe { libc::kill(pid as i32, 0) },
            -1,
            "deadline cleanup must reap NativeAttach child"
        );
        drop(client); // Intentionally unread until pump expiry.
    }

    #[test]
    fn native_attach_input_frame_expires_when_child_does_not_read() {
        use crate::terminal::TerminalFrame;
        let (mut attach, pid, _) = synthetic_attach("exec /bin/sleep 60");
        let frame = TerminalFrame {
            frame_type: "input".to_string(),
            data: "QUFB".repeat(5461),
            ..Default::default()
        };
        let started = Instant::now();
        let result = loop {
            match attach.input_frame(&frame) {
                Ok(()) => {}
                Err(error) => break error,
            }
        };
        let elapsed = started.elapsed();
        assert_eq!(result, "terminal input deadline exceeded");
        assert!(
            elapsed >= Duration::from_millis(1900),
            "stdin backpressure should use its 2s budget: {elapsed:?}"
        );
        attach.close().unwrap();
        assert_eq!(
            unsafe { libc::kill(pid as i32, 0) },
            -1,
            "input failure cleanup must reap child"
        );
    }

    // -- decode/gate layer: invalid bodies fail before any executor touch --

    #[test]
    fn profile_rejects_unknown_fields() {
        assert!(matches!(
            backend().profile(b"{\"unexpected\":1}"),
            Err(BackendError::Internal)
        ));
        assert!(matches!(
            backend().profile(b"[]"),
            Err(BackendError::Internal)
        ));
    }

    #[test]
    fn create_checks_identity_before_validate() {
        // Bad ID with good owner: Go's "invalid project identity" path.
        assert!(matches!(
            backend().create(b"{\"id\":\"bad id!\",\"owner\":1}"),
            Err(BackendError::Internal)
        ));
        // Good ID with bad owner: same path (never reaches Validate).
        assert!(matches!(
            backend().create(b"{\"id\":\"p0123456789abcdef01234567\",\"owner\":0}"),
            Err(BackendError::Internal)
        ));
        // Unknown fields rejected even with valid identity.
        assert!(matches!(
            backend().create(b"{\"id\":\"p0123456789abcdef01234567\",\"owner\":1,\"x\":1}"),
            Err(BackendError::Internal)
        ));
    }

    #[test]
    fn targeted_routes_validate_id_only() {
        for route in ["inspect", "os", "connection"] {
            let result = match route {
                "inspect" => backend().inspect(b"{\"id\":\"bad\"}"),
                "os" => backend().observe_os(b"{\"id\":\"bad\"}"),
                _ => backend().connection(b"{\"id\":\"bad\"}"),
            };
            assert!(matches!(result, Err(BackendError::Internal)), "{route}");
        }
    }

    #[test]
    fn lifecycle_rejects_bad_action_without_exec() {
        // Unknown action fails in the runtime gate before any container
        // inspection; unknown fields fail at decode.
        assert!(matches!(
            backend()
                .lifecycle(b"{\"project\":\"p0123456789abcdef01234567\",\"action\":\"bogus\"}"),
            Err(BackendError::Internal)
        ));
        assert!(matches!(
            backend().lifecycle(
                b"{\"project\":\"p0123456789abcdef01234567\",\"action\":\"start\",\"x\":1}"
            ),
            Err(BackendError::Internal)
        ));
    }

    #[test]
    fn factory_routes_report_unavailable_root() {
        // Factory opens lazily; the file-as-directory root fails the open,
        // mirroring Go's `factoryRun` 503 before any body decode.
        assert!(matches!(
            backend().factory_launch(b"{}"),
            Err(BackendError::Unavailable)
        ));
        assert!(matches!(
            backend().factory_harness(b"{}", "sha256:0"),
            Err(BackendError::Unavailable)
        ));
    }

    // -- integrated tailnet control plane (Lane T, no scaffold) --

    /// Backend whose tailnet controls observe scratch state paths that are
    /// never created, so the real control plane answers deterministically
    /// (default policy view, unavailable host) without appliance paths.
    fn backend_with_scratch_tailnet(name: &str) -> DaemonBackend {
        let mut backend = backend();
        let root = std::path::PathBuf::from("target/dbackend-test")
            .join(format!("{name}-{}", std::process::id()));
        let opts = tcontrol::Options {
            state_dir: root.join("soda-tailnet"),
            socket: root.join("tailscaled.sock"),
            ..tcontrol::Options::default()
        };
        backend.tailnet = tcontrol::Control::new(project::Native, opts.clone());
        backend.companion.tailnet = tcontrol::Control::new(project::Native, opts);
        backend
    }

    #[test]
    fn tailnet_settings_and_options_answer_without_appliance_state() {
        let backend = backend_with_scratch_tailnet("settings");
        let settings = backend.tailnet("settings", b"{}").expect("settings");
        let body = String::from_utf8(settings).expect("utf8 settings");
        assert!(body.contains("\"host_unavailable\":true"), "{body}");
        assert!(body.contains("\"revision\":\"0\""), "{body}");
        let options = backend.tailnet("options", b"{}").expect("options");
        let obody = String::from_utf8(options).expect("utf8 options");
        assert!(obody.contains("\"revision\":\"0\""), "{obody}");
    }

    #[test]
    fn tailnet_host_and_enrollment_reject_malformed_bodies() {
        let backend = backend_with_scratch_tailnet("reject");
        assert!(matches!(
            backend.tailnet("host", b"{"),
            Err(BackendError::Invalid)
        ));
        assert!(matches!(
            backend.tailnet("enrollment", b"{"),
            Err(BackendError::Invalid)
        ));
    }

    #[test]
    fn identity_launch_without_broker_is_internal() {
        // No broker listens at the test socket: the real client fails to
        // connect and the launch reports 500, exactly like Go's
        // `identityHandler` on broker errors.
        let start = br#"{"connection_id":"c","project_id":"p0123456789abcdef01234567","actor_id":"1","login":"l","scope":"s","cols":80,"rows":24}"#;
        assert!(matches!(
            backend().identity_launch(start),
            Err(BackendError::Internal)
        ));
        // The muse-configured backend constructs (real mserve runtime, no
        // I/O at open).
        let _ = backend_with_muse();
    }

    #[test]
    fn tailnet_rejects_bad_input_as_invalid() {
        assert!(matches!(
            backend().tailnet("settings", b"{\"x\":1}"),
            Err(BackendError::Invalid)
        ));
        assert!(matches!(
            backend().tailnet("bogus", b"{}"),
            Err(BackendError::Invalid)
        ));
        // Bad project id fails validation before any container resolve.
        assert!(matches!(
            backend().tailnet("project", b"{\"project\":\"bad\",\"action\":\"inspect\"}"),
            Err(BackendError::Invalid)
        ));
        // Policy admits inspect only.
        let pid = "p0123456789abcdef01234567";
        let body = format!(
            "{{\"project\":\"{pid}\",\"action\":\"enable\",\"revision\":\"0\",\"binding\":\"{pid}\",\"confirm_id\":\"{pid}\"}}"
        );
        assert!(matches!(
            backend().tailnet("policy", body.as_bytes()),
            Err(BackendError::Invalid)
        ));
    }

    #[test]
    fn tailnet_validate_rules_match_go() {
        use crate::tailnet_domain::ProjectRequest;
        let pid = "p0123456789abcdef01234567".to_string();
        let rev = "0123456789abcdef0123456789abcdef".to_string();
        let base = ProjectRequest {
            project: pid.clone(),
            action: "inspect".to_string(),
            revision: String::new(),
            binding: String::new(),
            confirm_id: String::new(),
        };
        assert!(validate_project_request(&base).is_ok());
        // Inspect forbids mutation fields.
        let mut bad = base.clone();
        bad.revision = rev.clone();
        assert!(validate_project_request(&bad).is_err());
        // Enable needs 32-hex binding + self confirm + valid revision.
        let ok = ProjectRequest {
            action: "enable".to_string(),
            revision: rev.clone(),
            binding: rev.clone(),
            confirm_id: pid.clone(),
            ..base.clone()
        };
        assert!(validate_project_request(&ok).is_ok());
        let mut bad_binding = ok.clone();
        bad_binding.binding = "xyz".to_string();
        assert!(validate_project_request(&bad_binding).is_err());
        let mut bad_confirm = ok.clone();
        bad_confirm.confirm_id = "pffffffffffffffffffffffff".to_string();
        assert!(validate_project_request(&bad_confirm).is_err());
        // Disable forbids binding.
        let mut disable = ok.clone();
        disable.action = "disable".to_string();
        assert!(validate_project_request(&disable).is_err());
        disable.binding.clear();
        assert!(validate_project_request(&disable).is_ok());
        // Unknown action rejected.
        let mut unknown = ok.clone();
        unknown.action = "explode".to_string();
        assert!(validate_project_request(&unknown).is_err());
        assert!(!valid_revision("ABCDEF0123456789abcdef0123456789"));
        assert!(valid_revision("0"));
    }

    #[test]
    fn tailnet_view_encode_matches_go_omitempty() {
        use crate::tailnet_domain::ProjectView;
        let empty = ProjectView::default();
        assert_eq!(
            encode_project_view(&empty),
            "{\"saved\":false,\"project\":\"\",\"revision\":\"\",\"binding\":\"\",\"enabled\":false,\"state\":\"\",\"outcome\":\"\"}"
        );
        let full = ProjectView {
            available_binding: "b".to_string(),
            addresses: vec!["10.0.0.1".to_string()],
            saved: true,
            project: "p".to_string(),
            enabled: true,
            state: "ready".to_string(),
            ..ProjectView::default()
        };
        assert_eq!(
            encode_project_view(&full),
            "{\"available_binding\":\"b\",\"addresses\":[\"10.0.0.1\"],\"saved\":true,\"project\":\"p\",\"revision\":\"\",\"binding\":\"\",\"enabled\":true,\"state\":\"ready\",\"outcome\":\"\"}"
        );
    }

    // -- error mapping --

    #[test]
    fn factory_error_mapping() {
        use crate::factory::FactoryError;
        assert!(matches!(
            map_factory_err(FactoryError::NotFound, BackendError::OutputStale),
            BackendError::NotFound
        ));
        assert!(matches!(
            map_factory_err(FactoryError::Stale, BackendError::OutputStale),
            BackendError::OutputStale
        ));
        assert!(matches!(
            map_factory_err(FactoryError::Stale, BackendError::ExportStale),
            BackendError::ExportStale
        ));
        assert!(matches!(
            map_factory_err(FactoryError::ExportCandidate, BackendError::OutputStale),
            BackendError::ExportCandidate
        ));
        assert!(matches!(
            map_factory_err(FactoryError::ExportBounds, BackendError::OutputStale),
            BackendError::ExportBounds
        ));
        assert!(matches!(
            map_factory_err(
                FactoryError::Msg("boom".to_string()),
                BackendError::OutputStale
            ),
            BackendError::Internal
        ));
    }

    #[test]
    fn tailnet_error_mapping() {
        assert!(matches!(
            map_tailnet_err("tailscale: invalid request".to_string()),
            BackendError::Invalid
        ));
        assert!(matches!(
            map_tailnet_err("revision conflict".to_string()),
            BackendError::ExportStale
        ));
        assert!(matches!(
            map_tailnet_err("action unsupported".to_string()),
            BackendError::ExportCandidate
        ));
        assert!(matches!(
            map_tailnet_err("tailnet unavailable".to_string()),
            BackendError::Unavailable
        ));
        assert!(matches!(
            map_tailnet_err("boom".to_string()),
            BackendError::Internal
        ));
    }

    // -- websocket codec over a loopback pair --

    #[test]
    fn terminal_accept_mints_unique_sessions() {
        let backend = backend();
        let first = backend.terminal_accept("k").unwrap().id;
        let second = backend.terminal_accept("k").unwrap().id;
        assert_ne!(first, second);
    }
}
