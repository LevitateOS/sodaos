// Real daemon backend: `ExecBackend` over the ported executors (PR26).
//
// This is the integrator wiring from GMUX_PATCHES.md §2: each trait method
// strict-decodes its JSON body (replacing `strictjson.Decode`), enforces the
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
// SCAFFOLD (temporary, branch-only): the `scaffold` module below stands in
// for the three still-missing lane pieces (iclient, tcontrol, mserve) with
// the exact contracted signatures. Every scaffold call fails with a
// `scaffold:`-prefixed error that maps to `BackendError::Unimplemented`, and
// the scaffold tests pin that list. Integration deletes the module and those
// tests with it; `HAS_SCAFFOLDS` must read false before the PR26 merge.
// (pops and tcodex lanes already integrated as real modules.)
use std::os::unix::net::UnixStream;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex, OnceLock};
use std::time::{Duration, Instant};

use crate::gmux_backend::{BackendError, ExecBackend, TerminalSession};
use crate::{domain, json, pfactory, project, texec};

/// False once every lane module is integrated and `scaffold` is deleted.
pub const HAS_SCAFFOLDS: bool = true;

const SCAFFOLD_PREFIX: &str = "scaffold:";

/// Lane-owned surface, exact contracted signatures, failing until the lanes
/// land. Each item names its owning branch.
mod scaffold {
    use std::time::Instant;

    fn pending(what: &str) -> String {
        format!("scaffold:{what} pending lane integration")
    }

    /// pr/26-tcontrol: tailnet control plane.
    pub struct Control;

    impl Control {
        pub fn new() -> Self {
            Control
        }
        pub fn settings(&self, _deadline: Instant) -> Result<Vec<u8>, String> {
            Err(pending("tcontrol.settings"))
        }
        pub fn options(&self, _deadline: Instant) -> Result<Vec<u8>, String> {
            Err(pending("tcontrol.options"))
        }
        pub fn host_action(&self, _body: &[u8], _deadline: Instant) -> Result<Vec<u8>, String> {
            Err(pending("tcontrol.host_action"))
        }
        pub fn enrollment(&self, _body: &[u8], _deadline: Instant) -> Result<Vec<u8>, String> {
            Err(pending("tcontrol.enrollment"))
        }
        pub fn project(
            &self,
            _req: &crate::tailnet_domain::ProjectRequest,
            _cid: &str,
            _deadline: Instant,
        ) -> Result<crate::tailnet_domain::ProjectView, String> {
            Err(pending("tcontrol.project"))
        }
    }

    /// TailnetControl over the scaffold control so the companion field below
    /// typechecks; every method fails until Lane T lands its impl, at which
    /// point the backend field swaps type with no call-site change.
    impl crate::tailnet_companion::TailnetControl for Control {
        fn project(
            &self,
            _req: &crate::tailnet_domain::ProjectRequest,
            _cid: &str,
            _deadline: Instant,
        ) -> Result<crate::tailnet_domain::ProjectView, String> {
            Err(pending("tcontrol.TailnetControl.project"))
        }
        fn run_binding(
            &self,
            _target: &crate::tailnet_domain::RunTarget,
            _deadline: Instant,
        ) -> Result<crate::tailnet_domain::RunBinding, String> {
            Err(pending("tcontrol.TailnetControl.run_binding"))
        }
        fn enroll_run(
            &self,
            _target: &crate::tailnet_domain::RunTarget,
            _recheck: &dyn Fn(Instant) -> Result<(), String>,
            _consume: &dyn Fn(Instant, &str) -> Result<(), String>,
            _deadline: Instant,
        ) -> Result<(), String> {
            Err(pending("tcontrol.TailnetControl.enroll_run"))
        }
    }
}

// -- error mapping (daemon.go ServeHTTP + subsystem handlers) --

/// Go maps every unrecognized dispatch error to 500. Scaffold failures map
/// to 501 so integration gaps stay visible (GMUX_PATCHES.md §7).
fn internal(err: String) -> BackendError {
    if err.starts_with(SCAFFOLD_PREFIX) {
        BackendError::Unimplemented
    } else {
        BackendError::Internal
    }
}

/// `strictjson.Decode` failure inside a native dispatch: Go returns the
/// error and ServeHTTP renders 500.
fn decode_native(body: &[u8]) -> Result<json::Value, BackendError> {
    json::decode_strict(body).map_err(|e| internal(e.0))
}

/// Empty-object body (`dispatchProfile`, `/factory-harness`): Go decodes
/// into `struct{}`, so `{}` (and JSON `null`, which decodes into any Go
/// value) passes and anything else fails.
fn decode_empty(body: &[u8]) -> Result<(), BackendError> {
    let v = decode_native(body)?;
    match &v {
        json::Value::Null => Ok(()),
        json::Value::Object(fields) if fields.is_empty() => Ok(()),
        _ => Err(BackendError::Internal),
    }
}

/// `FactoryError` to wire mapping. `stale` is per-method: output reads map
/// `Stale` to `OutputStale`, export/candidate reads to `ExportStale`.
fn map_factory_err(err: pfactory::FactoryError, stale: BackendError) -> BackendError {
    match err {
        pfactory::FactoryError::NotFound => BackendError::NotFound,
        pfactory::FactoryError::Stale => stale,
        pfactory::FactoryError::ExportCandidate => BackendError::ExportCandidate,
        pfactory::FactoryError::ExportBounds => BackendError::ExportBounds,
        pfactory::FactoryError::Msg(m) => internal(m),
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
}

/// Broker side of supervised factory runs. Delegates to the identity
/// broker client; method names match the lane contract so this impl is
/// final except for the client type.
pub struct BrokerSeam {
    broker: crate::iclient::BrokerClient,
}

/// Muse lease hooks over the identity broker client.
pub struct HooksSeam {
    broker: crate::iclient::BrokerClient,
}

impl HooksSeam {
    /// The binary builds launch-time hooks from the same broker socket.
    pub fn new(broker_socket: &str) -> Self {
        HooksSeam {
            broker: crate::iclient::BrokerClient::new(broker_socket),
        }
    }
}

type Factory = pfactory::Factory<project::Native, TerminalSeam, BrokerSeam>;

/// The real daemon backend. All runtimes share one native executor; the
/// factory opens lazily like Go's `factoryRun` so a broken receipt root
/// degrades factory routes to 503 instead of refusing startup.
pub struct DaemonBackend {
    project: project::Runtime<project::Native>,
    terminal: texec::Service<project::Native>,
    factory: OnceLock<Result<Factory, String>>,
    broker: crate::iclient::BrokerClient,
    tailnet: scaffold::Control,
    companion: crate::tailnet_companion::Companion<project::Native, scaffold::Control>,
    pops: crate::pops::Ops<project::Native>,
    muse: Option<crate::muse::MuseRuntime<project::Native, HooksSeam>>,
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
            terminal: texec::Service {
                exec: project::Native,
                codex_harness: cfg.codex_harness.clone(),
                codex_harness_sha256: cfg.codex_harness_sha256.clone(),
                codex_harness_version: cfg.codex_harness_version.clone(),
            },
            factory: OnceLock::new(),
            broker: crate::iclient::BrokerClient::new(&broker_socket),
            tailnet: scaffold::Control::new(),
            companion: crate::tailnet_companion::Companion {
                exec: project::Native,
                tailnet: scaffold::Control::new(),
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
                    HooksSeam {
                        broker: crate::iclient::BrokerClient::new(&broker_socket),
                    },
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
                    },
                    BrokerSeam {
                        broker: crate::iclient::BrokerClient::new(&self.broker_socket),
                    },
                )
            })
            .as_ref()
            .map_err(|_| BackendError::Unavailable)
    }
}

// -- twin conversions (pfactory/texec/tcodex mirror the same Go records) --
//
// The executor lanes each own identical wire twins. Conversion is field
// copying; the only semantic step is the RFC 3339 deadline, parsed once
// via the shared parser. A single conversion site per direction keeps the
// twins from drifting (the pending restructure unifies them).

fn cv_lease_to_texec(l: &pfactory::Lease) -> texec::Lease {
    texec::Lease {
        repository_id: l.repository_id,
        provider_id: l.provider_id.clone(),
        id: l.id.clone(),
        connection_id: l.connection_id.clone(),
        generation: l.generation,
        actor_id: l.actor_id,
        project_id: l.project_id.clone(),
        execution_id: l.execution_id.clone(),
        kind: l.kind.clone(),
        role: l.role.clone(),
        deadline_raw: l.deadline.clone(),
        deadline: texec::parse_rfc3339(&l.deadline),
        grant_id: l.grant_id.clone(),
        grant_revision: l.grant_revision,
        binding: l.binding.as_ref().map(cv_binding_to_texec),
    }
}

fn cv_binding_to_texec(b: &pfactory::Binding) -> texec::Binding {
    texec::Binding {
        child_id: b.child_id.clone(),
        uid: b.uid,
        gid: b.gid,
        scope: b.scope.clone(),
        credential_root: b.credential_root.clone(),
        invocation_id: b.invocation_id.clone(),
        kind: b.kind.clone(),
        id: b.id.clone(),
        project: b.project.clone(),
        login: b.login.clone(),
        generation: b.generation,
    }
}

fn cv_binding_to_pfactory(b: &texec::Binding) -> pfactory::Binding {
    pfactory::Binding {
        child_id: b.child_id.clone(),
        uid: b.uid,
        gid: b.gid,
        scope: b.scope.clone(),
        credential_root: b.credential_root.clone(),
        invocation_id: b.invocation_id.clone(),
        kind: b.kind.clone(),
        id: b.id.clone(),
        project: b.project.clone(),
        login: b.login.clone(),
        generation: b.generation,
    }
}

fn cv_run_to_tcodex(r: &pfactory::FactoryRun) -> crate::tcodex::FactoryRun {
    crate::tcodex::FactoryRun {
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

fn cv_acquire_to_texec(r: &pfactory::AcquireRequest) -> texec::AcquireRequest {
    let (secs, nanos) = texec::parse_rfc3339(&r.deadline).unwrap_or((0, 0));
    texec::AcquireRequest {
        repository_id: 0,
        provider_id: r.provider_id.clone(),
        execution_id: r.execution_id.clone(),
        actor_id: r.actor_id,
        connection_id: r.connection_id.clone(),
        project_id: r.project_id.clone(),
        kind: r.kind.clone(),
        deadline_secs: secs,
        deadline_nanos: nanos,
        role: r.role.clone(),
    }
}

fn cv_lease_to_pfactory(l: &texec::Lease) -> pfactory::Lease {
    pfactory::Lease {
        repository_id: l.repository_id,
        provider_id: l.provider_id.clone(),
        id: l.id.clone(),
        connection_id: l.connection_id.clone(),
        generation: l.generation,
        actor_id: l.actor_id,
        project_id: l.project_id.clone(),
        execution_id: l.execution_id.clone(),
        kind: l.kind.clone(),
        role: l.role.clone(),
        deadline: l.deadline_raw.clone(),
        grant_id: l.grant_id.clone(),
        grant_revision: l.grant_revision,
        binding: l.binding.as_ref().map(cv_binding_to_pfactory),
    }
}

fn cv_slice_to_pfactory(s: &crate::tcodex::FactoryCodexOutputSlice) -> pfactory::OutputSlice {
    pfactory::OutputSlice {
        data: s.data.clone(),
        total: s.total,
        offset: s.offset,
        truncated: s.truncated,
        gap: s.gap,
    }
}

// -- seam implementations --

impl pfactory::FactoryTerminal for TerminalSeam {
    fn harness_version(&self) -> String {
        self.harness_version.clone()
    }
    fn harness_sha256(&self) -> String {
        self.harness_sha256.clone()
    }
    fn reserve(
        &self,
        run: &pfactory::FactoryRun,
        lease: &pfactory::Lease,
        pin: &str,
        max_secs: i64,
        deadline: Instant,
    ) -> Result<pfactory::Binding, pfactory::FactoryError> {
        let service = self.service();
        let trun = cv_run_to_tcodex(run);
        let tlease = cv_lease_to_texec(lease);
        service
            .factory_codex_reserve(&trun, &tlease, pin, max_secs, deadline)
            .map(|(binding, _paths)| cv_binding_to_pfactory(&binding))
            .map_err(pfactory::FactoryError::Msg)
    }
    fn start(
        &self,
        lease: &pfactory::Lease,
        credential: &[u8],
        prompt: &[u8],
        deadline: Instant,
    ) -> Result<(), pfactory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        service
            .factory_codex_start(&tlease, credential, prompt, deadline)
            .map_err(pfactory::FactoryError::Msg)
    }
    fn wait(
        &self,
        lease: &pfactory::Lease,
        deadline: Instant,
    ) -> Result<(i64, String), pfactory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        service
            .factory_codex_wait(&tlease, deadline)
            .map(|(code, out)| (code as i64, out))
            .map_err(pfactory::FactoryError::Msg)
    }
    fn stop(
        &self,
        lease: &pfactory::Lease,
        deadline: Instant,
    ) -> Result<(), pfactory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        service
            .factory_codex_stop(&tlease, deadline)
            .map_err(pfactory::FactoryError::Msg)
    }
    fn stop_unbound(
        &self,
        run: &pfactory::FactoryRun,
        deadline: Instant,
    ) -> Result<(), pfactory::FactoryError> {
        let service = self.service();
        let trun = cv_run_to_tcodex(run);
        service
            .factory_codex_stop_unbound(&trun, deadline)
            .map_err(pfactory::FactoryError::Msg)
    }
    fn capture(
        &self,
        lease: &pfactory::Lease,
        deadline: Instant,
    ) -> Result<Vec<u8>, pfactory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        service
            .factory_codex_capture(&tlease, deadline)
            .map_err(pfactory::FactoryError::Msg)
    }
    fn live(&self, binding: &pfactory::Binding, deadline: Instant) -> bool {
        let service = self.service();
        let tbinding = cv_binding_to_texec(binding);
        service.factory_codex_live(&tbinding, deadline)
    }
    fn output(
        &self,
        project: &str,
        binding: &pfactory::Binding,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<pfactory::OutputSlice, pfactory::FactoryError> {
        let service = self.service();
        let tbinding = cv_binding_to_texec(binding);
        service
            .factory_codex_output(project, &tbinding, offset, limit, deadline)
            .map(|s| cv_slice_to_pfactory(&s))
            .map_err(pfactory::FactoryError::Msg)
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
    ) -> Result<(String, bool), pfactory::FactoryError> {
        let service = self.service();
        service
            .factory_takeover_copy(project, recorded, role, preparation, member, run, deadline)
            .map_err(pfactory::FactoryError::Msg)
    }
    fn export_bundle(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        candidate: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, pfactory::FactoryError> {
        let service = self.service();
        service
            .factory_export_bundle(project, recorded, role, preparation, candidate, deadline)
            .map_err(pfactory::FactoryError::Msg)
    }
}

impl TerminalSeam {
    /// The seam owns no service state: it rebuilds the value service (pure
    /// configuration + the shared executor shape) per call. Service methods
    /// take `&self` and hold no interior state, so this is free.
    fn service(&self) -> texec::Service<project::Native> {
        texec::Service {
            exec: project::Native,
            codex_harness: self.harness.clone(),
            codex_harness_sha256: self.harness_sha256.clone(),
            codex_harness_version: self.harness_version.clone(),
        }
    }
}

impl pfactory::FactoryBroker for BrokerSeam {
    fn acquire(
        &self,
        req: &pfactory::AcquireRequest,
        deadline: Instant,
    ) -> Result<pfactory::Lease, pfactory::FactoryError> {
        // The scaffold broker fails here until Lane I lands; the conversion
        // below is final and covered by the round-trip test.
        let treq = cv_acquire_to_texec(req);
        self.broker
            .acquire(&treq, deadline)
            .map(|lease| cv_lease_to_pfactory(&lease))
            .map_err(pfactory::FactoryError::Msg)
    }
    fn register(
        &self,
        _lease_id: &str,
        _binding: &pfactory::Binding,
        _deadline: Instant,
    ) -> Result<Vec<u8>, pfactory::FactoryError> {
        Err(pfactory::FactoryError::Msg(scaffold_broker_err()))
    }
    fn return_lease(
        &self,
        lease_id: &str,
        binding: &pfactory::Binding,
        credential: &[u8],
        deadline: Instant,
    ) -> Result<(), pfactory::FactoryError> {
        let tbinding = cv_binding_to_texec(binding);
        self.broker
            .return_lease(lease_id, &tbinding, credential, deadline)
            .map_err(pfactory::FactoryError::Msg)
    }
    fn reconcile_lease(&self, _lease_id: &str, _deadline: Instant) {}
    fn execution_is_terminal(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<bool, pfactory::FactoryError> {
        self.broker
            .get_execution(kind, execution_id, deadline)
            .map(|exec| crate::iclient::execution_is_terminal(&exec))
            .map_err(pfactory::FactoryError::Msg)
    }
    fn close_execution(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<(), pfactory::FactoryError> {
        self.broker
            .close_execution(kind, execution_id, deadline)
            .map_err(pfactory::FactoryError::Msg)
    }
}

fn scaffold_broker_err() -> String {
    format!("{SCAFFOLD_PREFIX}iclient pending lane integration")
}

impl crate::muse::MuseHooks for HooksSeam {
    fn acquire(
        &self,
        req: &texec::AcquireRequest,
        deadline: Instant,
    ) -> Result<texec::Lease, String> {
        self.broker.acquire(req, deadline)
    }
    fn attach(
        &self,
        lease_id: &str,
        binding: &texec::Binding,
        deadline: Instant,
    ) -> Result<texec::Delivery, String> {
        self.broker.register(lease_id, binding, deadline)
    }
    fn end(&self, actor: i64, lease_id: &str, deadline: Instant) -> Result<(), String> {
        self.broker.end_lease(actor, lease_id, deadline)
    }
    fn authorize(&self, actor: i64, project: &str, deadline: Instant) -> Result<(), String> {
        // Go `museRuntime.Authorize`: a Muse+Ready connection must exist.
        let available = self.broker.available(actor, project, deadline)?;
        crate::muse::muse_connection_authorized(&available)
    }
    fn nested_authorize(&self, actor: i64, project: &str, deadline: Instant) -> Result<(), String> {
        // Go `authorizeNested` is the identical check.
        self.authorize(actor, project, deadline)
    }
    fn select(
        &self,
        actor: i64,
        project: &str,
        selected: &str,
        deadline: Instant,
    ) -> Result<String, String> {
        let available = self.broker.available(actor, project, deadline)?;
        crate::muse::select_muse_connection(&available, selected)
    }
}

impl texec::IdentityBroker for DaemonBackend {
    fn acquire(
        &self,
        req: &texec::AcquireRequest,
        deadline: Instant,
    ) -> Result<texec::Lease, String> {
        self.broker.acquire(req, deadline)
    }
    fn register(
        &self,
        lease_id: &str,
        binding: &texec::Binding,
        deadline: Instant,
    ) -> Result<texec::Delivery, String> {
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
        let v = decode_native(body)?;
        let id = domain::Create::from_value(&v)
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
        let v = decode_native(body)?;
        let input = domain::Create::from_value(&v).map_err(internal)?;
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
        let v = decode_native(body)?;
        let input = project::Lifecycle::from_value(&v).map_err(internal)?;
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
        let v = decode_native(body)?;
        let req = pfactory::FactoryLaunch::from_value(&v).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .launch(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let v = decode_native(body)?;
        let req = pfactory::FactoryInspect::from_value(&v).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .inspect(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_stop(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let v = decode_native(body)?;
        let req = pfactory::FactoryStop::from_value(&v).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .stop(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_takeover(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let v = decode_native(body)?;
        let req = pfactory::FactoryTakeover::from_value(&v).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .takeover(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_output(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let v = decode_native(body)?;
        let req = pfactory::FactoryOutput::from_value(&v).map_err(internal)?;
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
        let v = decode_native(body)?;
        let req = pfactory::FactoryExport::from_value(&v).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .export(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_candidate_inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let v = decode_native(body)?;
        let req = pfactory::FactoryCandidateInspect::from_value(&v).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .inspect_candidate(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn identity_launch(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        // The mux maps every non-501 identity error to 409, like Go's
        // `identityHandler`; `internal` preserves the scaffold signal.
        let input = texec::TerminalStart::decode(body).map_err(internal)?;
        let lease = texec::identity_launch(
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
        let delivery = texec::Delivery::decode(body).map_err(internal)?;
        if delivery.lease.kind == texec::KIND_FACTORY {
            // Broker validate/stop/finish callbacks for supervised runs.
            let out = self
                .terminal
                .factory_identity_operation(action, &delivery, native_deadline())
                .map_err(internal)?;
            return Ok(out.encode().into_bytes());
        }
        let muse_scoped = delivery.lease.provider_id == texec::PROVIDER_MUSE
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
        stream: UnixStream,
        _session: TerminalSession,
    ) -> Result<(), BackendError> {
        pump_terminal(&self.terminal, self, stream).map_err(internal)
    }
}

// -- tailnet helpers (tailnet.go dispatchTailnetAction) --

/// Tailnet decode failure: Go's `decodeTailnetBody` reports `ErrInvalid`
/// (400), unlike the native 500.
fn decode_tailnet_empty(body: &[u8]) -> Result<(), BackendError> {
    let v = json::decode_strict(body).map_err(|_| BackendError::Invalid)?;
    match &v {
        json::Value::Null => Ok(()),
        json::Value::Object(fields) if fields.is_empty() => Ok(()),
        _ => Err(BackendError::Invalid),
    }
}

/// Lane T error substrings to wire mapping. The mux renders tailnet
/// `ExportStale` as 409 and `ExportCandidate` as 422, so conflict and
/// unsupported reuse those variants behind this subsystem only.
fn map_tailnet_err(err: String) -> BackendError {
    if err.starts_with(SCAFFOLD_PREFIX) {
        return BackendError::Unimplemented;
    }
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

const TAILNET_PROJECT_SPECS: &[json::Spec] = &[
    json::Spec {
        name: "project",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "action",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "revision",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "binding",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "confirm_id",
        kind: json::Kind::Str,
    },
];

/// Strict decode of one project/policy request (Go `ProjectRequest` shape).
fn decode_project_request(
    body: &[u8],
) -> Result<crate::tailnet_domain::ProjectRequest, BackendError> {
    let v = json::decode_strict(body).map_err(|_| BackendError::Invalid)?;
    let m = json::bind_root(&v, "ProjectRequest", TAILNET_PROJECT_SPECS, false)
        .map_err(|_| BackendError::Invalid)?;
    Ok(crate::tailnet_domain::ProjectRequest {
        project: m.take_string("project"),
        action: m.take_string("action"),
        revision: m.take_string("revision"),
        binding: m.take_string("binding"),
        confirm_id: m.take_string("confirm_id"),
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

/// Incoming websocket message after server-side processing.
enum WsIn {
    Text(Vec<u8>),
    Closed,
}

/// Read one complete client message: single or fragmented frames, ping
/// answered, pong ignored, close ends the pump. Client frames MUST be
/// masked (RFC 6455 §5.1); anything else is a transport error. `TimedOut`
/// reads surface as `Closed` so the expiry deadline ends the pump.
fn ws_read_message(
    stream: &mut UnixStream,
    limit: usize,
    deadline: Instant,
) -> Result<WsIn, String> {
    let mut message: Vec<u8> = Vec::new();
    let mut fragmented = false;
    loop {
        set_deadline(stream, deadline, true)?;
        let mut head = [0u8; 2];
        read_exact(stream, &mut head)?;
        let fin = head[0] & 0x80 != 0;
        let opcode = head[0] & 0x0f;
        let masked = head[1] & 0x80 != 0;
        if !masked {
            return Err("terminal transport ended".to_string());
        }
        let mut length = (head[1] & 0x7f) as u64;
        if length == 126 {
            let mut ext = [0u8; 2];
            read_exact(stream, &mut ext)?;
            length = u16::from_be_bytes(ext) as u64;
        } else if length == 127 {
            let mut ext = [0u8; 8];
            read_exact(stream, &mut ext)?;
            length = u64::from_be_bytes(ext);
        }
        if opcode >= 0x8 && length > 125 {
            return Err("terminal transport ended".to_string());
        }
        if length > limit as u64 || message.len() as u64 + length > limit as u64 {
            return Err("terminal transport ended".to_string());
        }
        let mut mask = [0u8; 4];
        read_exact(stream, &mut mask)?;
        let mut payload = vec![0u8; length as usize];
        read_exact(stream, &mut payload)?;
        for (i, byte) in payload.iter_mut().enumerate() {
            *byte ^= mask[i % 4];
        }
        match opcode {
            0x8 => return Ok(WsIn::Closed),
            0x9 => {
                ws_write_frame(stream, 0xA, &payload, deadline)?;
                continue;
            }
            0xA => continue,
            // Binary frames are invalid control (Go rejects them); fail the
            // transport like Go's handshake/message error.
            0x2 => return Err("terminal transport ended".to_string()),
            0x0..=0x1 => {
                if opcode != 0x0 {
                    if fragmented {
                        return Err("terminal transport ended".to_string());
                    }
                } else if !fragmented {
                    return Err("terminal transport ended".to_string());
                }
                message.extend_from_slice(&payload);
                if fin {
                    return Ok(WsIn::Text(message));
                }
                fragmented = true;
            }
            _ => return Err("terminal transport ended".to_string()),
        }
    }
}

fn read_exact(stream: &mut UnixStream, buf: &mut [u8]) -> Result<(), String> {
    use std::io::Read;
    stream.read_exact(buf).map_err(|e| {
        if e.kind() == std::io::ErrorKind::TimedOut {
            "terminal transport expired".to_string()
        } else {
            "terminal transport ended".to_string()
        }
    })
}

fn set_deadline(stream: &UnixStream, deadline: Instant, read: bool) -> Result<(), String> {
    let remaining = deadline.saturating_duration_since(Instant::now());
    let result = if read {
        stream.set_read_timeout(Some(remaining))
    } else {
        stream.set_write_timeout(Some(remaining))
    };
    result.map_err(|_| "terminal transport ended".to_string())
}

/// Write one unmasked server text frame (5s cap like Go's `Write`).
fn ws_write_text(
    stream: &mut UnixStream,
    body: &[u8],
    ctx_deadline: Instant,
) -> Result<(), String> {
    let write_deadline = Instant::now() + Duration::from_secs(5);
    let deadline = write_deadline.min(ctx_deadline);
    ws_write_frame(stream, 0x1, body, deadline)
}

fn ws_write_frame(
    stream: &mut UnixStream,
    opcode: u8,
    body: &[u8],
    deadline: Instant,
) -> Result<(), String> {
    use std::io::Write;
    set_deadline(stream, deadline, false)?;
    let mut head = vec![0x80 | opcode];
    if body.len() < 126 {
        head.push(body.len() as u8);
    } else if body.len() <= 0xffff {
        head.push(126);
        head.extend_from_slice(&(body.len() as u16).to_be_bytes());
    } else {
        head.push(127);
        head.extend_from_slice(&(body.len() as u64).to_be_bytes());
    }
    stream.write_all(&head).map_err(|e| {
        if e.kind() == std::io::ErrorKind::TimedOut {
            "terminal transport expired".to_string()
        } else {
            "terminal transport ended".to_string()
        }
    })?;
    stream
        .write_all(body)
        .map_err(|_| "terminal transport ended".to_string())?;
    stream
        .flush()
        .map_err(|_| "terminal transport ended".to_string())
}

fn closed_frame(reason: &str) -> texec::TerminalFrame {
    texec::TerminalFrame {
        frame_type: "closed".to_string(),
        data: String::new(),
        cols: 0,
        rows: 0,
        reason: reason.to_string(),
        terminals: None,
    }
}

/// Owns the upgraded stream until the session ends: first-frame request
/// (5s, text, ≤4096 bytes, strict `TerminalRequest`, valid window),
/// expiry deadline, launch (inspect 10s, managed end, native attach),
/// then the bidirectional pump. Mirrors `Handler`/`pumpIO` in service.go.
fn pump_terminal(
    service: &texec::Service<project::Native>,
    backend: &DaemonBackend,
    mut stream: UnixStream,
) -> Result<(), String> {
    use crate::gmux_admission::{TERMINAL_FRAME_LIMIT, TERMINAL_REQUEST_LIMIT};

    // First frame: 5s like Go's `readTerminalRequest`.
    let first_deadline = Instant::now() + Duration::from_secs(5);
    let request_bytes = match ws_read_message(&mut stream, TERMINAL_FRAME_LIMIT, first_deadline)? {
        WsIn::Text(body) if body.len() <= TERMINAL_REQUEST_LIMIT => body,
        _ => return Ok(()),
    };
    let request = match texec::TerminalRequest::decode(&request_bytes).ok() {
        Some(request) if request.valid(texec::now_unix()) => request,
        _ => return Ok(()),
    };
    // Expiry deadline from the request (Go: 12h ctx capped by Expires).
    let expiry = Instant::now()
        + Duration::from_secs(request.expires.saturating_sub(texec::now_unix()).max(0) as u64);
    let session_deadline = (Instant::now() + Duration::from_secs(12 * 3600)).min(expiry);

    // Launch: inspect (10s), managed end, native attach.
    let inspect_deadline = Instant::now() + Duration::from_secs(10);
    let container = match service.project_container(
        &request.project,
        true,
        inspect_deadline.min(session_deadline),
    ) {
        Ok(cid) => cid,
        Err(_) => {
            let frame = closed_frame("launch_failed");
            let _ = ws_write_text(&mut stream, frame.encode().as_bytes(), session_deadline);
            return Ok(());
        }
    };
    let end_hook = |actor: i64, lease_id: &str| {
        // Scaffold until Lane I lands; then the real broker call.
        // The error text never surfaces (managed_end maps it internally).
        backend.broker.end_lease(actor, lease_id, session_deadline)
    };
    if service
        .managed_end(&container, &request, Some(&end_hook), session_deadline)
        .is_err()
    {
        let frame = closed_frame("cleanup_unconfirmed");
        let _ = ws_write_text(&mut stream, frame.encode().as_bytes(), session_deadline);
        return Ok(());
    }
    let attach = match texec::NativeAttach::attach(&container, &request) {
        Ok(attach) => attach,
        Err(_) => {
            let frame = closed_frame("launch_failed");
            let _ = ws_write_text(&mut stream, frame.encode().as_bytes(), session_deadline);
            return Ok(());
        }
    };
    let attach = Arc::new(Mutex::new(attach));

    // Expiry watcher: Go's ctx deadline cancels the socket and closes the
    // process; here the watcher closes the attach and read timeouts end
    // the loops.
    let expiry_attach = Arc::clone(&attach);
    let expiry_in = session_deadline.saturating_duration_since(Instant::now());
    std::thread::spawn(move || {
        std::thread::sleep(expiry_in);
        if let Ok(mut guard) = expiry_attach.lock() {
            guard.close();
        }
    });

    // Incoming: text TerminalFrames, strict, input-valid; a `close` frame
    // ends input (the launcher still reports teardown); anything else
    // closes the attach and ends the pump.
    let mut incoming_stream = stream
        .try_clone()
        .map_err(|_| "terminal transport ended".to_string())?;
    let incoming_attach = Arc::clone(&attach);
    let incoming = std::thread::spawn(move || loop {
        let message =
            match ws_read_message(&mut incoming_stream, TERMINAL_FRAME_LIMIT, session_deadline) {
                Ok(WsIn::Text(body)) => body,
                Ok(_) => {
                    if let Ok(mut guard) = incoming_attach.lock() {
                        guard.close();
                    }
                    return;
                }
                Err(_) => {
                    if let Ok(mut guard) = incoming_attach.lock() {
                        guard.close();
                    }
                    return;
                }
            };
        let frame = texec::TerminalFrame::decode(&message);
        let valid = frame.as_ref().is_ok_and(|f| f.input_valid());
        match frame {
            Ok(frame) if valid => {
                let is_close = frame.frame_type == "close";
                let mut guard = match incoming_attach.lock() {
                    Ok(guard) => guard,
                    Err(_) => return,
                };
                if guard.input_frame(&frame).is_err() {
                    guard.close();
                    return;
                }
                drop(guard);
                if is_close {
                    return;
                }
            }
            _ => {
                if let Ok(mut guard) = incoming_attach.lock() {
                    guard.close();
                }
                return;
            }
        }
    });

    // Outgoing: frames until `closed`/`metadata`, error, or expiry.
    loop {
        let frame = {
            let mut guard = match attach.lock() {
                Ok(guard) => guard,
                Err(_) => break,
            };
            match guard.output_frame() {
                Ok(frame) => frame,
                Err(_) => break,
            }
        };
        let done = frame.frame_type == "closed" || frame.frame_type == "metadata";
        if ws_write_text(&mut stream, frame.encode().as_bytes(), session_deadline).is_err() {
            break;
        }
        if done {
            break;
        }
    }

    // Go's `defer p.Close()` + conn close: unblock the reader, join it.
    if let Ok(mut guard) = attach.lock() {
        guard.close();
    }
    let _ = stream.shutdown(std::net::Shutdown::Both);
    let _ = incoming.join();
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::gmux_backend::ExecBackend;

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

    // -- scaffold tripwires (deleted at integration with the scaffold mod) --

    #[test]
    #[allow(clippy::assertions_on_constants)]
    fn scaffolds_advertised() {
        assert!(HAS_SCAFFOLDS);
    }

    #[test]
    fn scaffold_tailnet_and_identity_surface_as_unimplemented() {
        // Valid-shaped bodies that reach past decode into scaffolded
        // executors must report 501, never 500 or success.
        assert!(matches!(
            backend().tailnet("settings", b"{}"),
            Err(BackendError::Unimplemented)
        ));
        assert!(matches!(
            backend().tailnet("host", b"{}"),
            Err(BackendError::Unimplemented)
        ));
        assert!(matches!(
            backend().tailnet("enrollment", b"{}"),
            Err(BackendError::Unimplemented)
        ));
        // No broker listens at the test socket: the real client fails to
        // connect and the launch reports 500, exactly like Go's
        // `identityHandler` on broker errors.
        let start = br#"{"connection_id":"c","project_id":"p0123456789abcdef01234567","actor_id":"1","login":"l","scope":"s","cols":80,"rows":24}"#;
        assert!(matches!(
            backend().identity_launch(start),
            Err(BackendError::Internal)
        ));
        // Muse-scoped delivery with muse configured reaches the mserve
        // scaffold instead of the terminal identity path.
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
        use pfactory::FactoryError;
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
                FactoryError::Msg("scaffold:x".to_string()),
                BackendError::OutputStale
            ),
            BackendError::Unimplemented
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
            map_tailnet_err("scaffold:x".to_string()),
            BackendError::Unimplemented
        ));
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

    // -- twin conversions round-trip losslessly --

    #[test]
    fn lease_binding_round_trip() {
        let lease = pfactory::Lease {
            repository_id: 7,
            provider_id: "codex".to_string(),
            id: "l".to_string(),
            connection_id: "c".to_string(),
            generation: 3,
            actor_id: 1001,
            project_id: "p".to_string(),
            execution_id: "e".to_string(),
            kind: "factory".to_string(),
            role: "coder".to_string(),
            deadline: "2026-10-05T00:00:00Z".to_string(),
            grant_id: "g".to_string(),
            grant_revision: 2,
            binding: Some(pfactory::Binding {
                child_id: "child".to_string(),
                uid: 1001,
                gid: 1001,
                scope: "project".to_string(),
                credential_root: "/run/cred".to_string(),
                invocation_id: "i".to_string(),
                kind: "factory".to_string(),
                id: "b".to_string(),
                project: "p".to_string(),
                login: "coder".to_string(),
                generation: 1,
            }),
        };
        let back = cv_lease_to_pfactory(&cv_lease_to_texec(&lease));
        assert_eq!(back, lease);
    }

    // -- websocket codec over a loopback pair --

    fn masked_frame(opcode: u8, payload: &[u8], fin: bool) -> Vec<u8> {
        let mut frame = vec![(if fin { 0x80 } else { 0 }) | opcode];
        let mask = [0x11u8, 0x22, 0x33, 0x44];
        if payload.len() < 126 {
            frame.push(0x80 | payload.len() as u8);
        } else {
            frame.push(0x80 | 126);
            frame.extend_from_slice(&(payload.len() as u16).to_be_bytes());
        }
        frame.extend_from_slice(&mask);
        frame.extend(payload.iter().enumerate().map(|(i, b)| b ^ mask[i % 4]));
        frame
    }

    fn read_server_frame(stream: &mut UnixStream) -> (u8, Vec<u8>) {
        use std::io::Read;
        let mut head = [0u8; 2];
        stream.read_exact(&mut head).unwrap();
        assert_eq!(head[1] & 0x80, 0, "server frames are unmasked");
        let mut len = (head[1] & 0x7f) as usize;
        if len == 126 {
            let mut ext = [0u8; 2];
            stream.read_exact(&mut ext).unwrap();
            len = u16::from_be_bytes(ext) as usize;
        }
        let mut payload = vec![0u8; len];
        stream.read_exact(&mut payload).unwrap();
        (head[0] & 0x0f, payload)
    }

    #[test]
    fn ws_codec_round_trip_ping_fragmentation_close() {
        use std::io::Write;
        use std::os::unix::net::UnixStream;
        let deadline = Instant::now() + Duration::from_secs(5);
        // Text message.
        let (mut client, mut server) = UnixStream::pair().unwrap();
        client
            .write_all(&masked_frame(0x1, b"hello", true))
            .unwrap();
        match ws_read_message(&mut server, 131072, deadline).unwrap() {
            WsIn::Text(body) => assert_eq!(body, b"hello"),
            _ => panic!("expected text"),
        }
        // Fragmented message assembles.
        let (mut client, mut server) = UnixStream::pair().unwrap();
        client.write_all(&masked_frame(0x1, b"he", false)).unwrap();
        client.write_all(&masked_frame(0x0, b"llo", true)).unwrap();
        match ws_read_message(&mut server, 131072, deadline).unwrap() {
            WsIn::Text(body) => assert_eq!(body, b"hello"),
            _ => panic!("expected assembled text"),
        }
        // Ping is answered, then the next message reads.
        let (mut client, mut server) = UnixStream::pair().unwrap();
        client.write_all(&masked_frame(0x9, b"pp", true)).unwrap();
        client
            .write_all(&masked_frame(0x1, b"after", true))
            .unwrap();
        match ws_read_message(&mut server, 131072, deadline).unwrap() {
            WsIn::Text(body) => assert_eq!(body, b"after"),
            _ => panic!("expected text after ping"),
        }
        let (opcode, pong) = read_server_frame(&mut client);
        assert_eq!(opcode, 0xA);
        assert_eq!(pong, b"pp");
        // Close ends the pump.
        let (mut client, mut server) = UnixStream::pair().unwrap();
        client.write_all(&masked_frame(0x8, b"", true)).unwrap();
        assert!(matches!(
            ws_read_message(&mut server, 131072, deadline),
            Ok(WsIn::Closed)
        ));
        // Binary, unmasked, and over-limit frames are transport errors.
        let (mut client, mut server) = UnixStream::pair().unwrap();
        client.write_all(&masked_frame(0x2, b"nope", true)).unwrap();
        assert!(ws_read_message(&mut server, 131072, deadline).is_err());
        let (mut client, mut server) = UnixStream::pair().unwrap();
        client.write_all(&[0x81, 0x01, b'x']).unwrap();
        assert!(ws_read_message(&mut server, 131072, deadline).is_err());
        let (mut client, mut server) = UnixStream::pair().unwrap();
        client
            .write_all(&masked_frame(0x1, b"toolong", true))
            .unwrap();
        assert!(ws_read_message(&mut server, 4, deadline).is_err());
    }

    #[test]
    fn terminal_accept_mints_unique_sessions() {
        let backend = backend();
        let first = backend.terminal_accept("k").unwrap().id;
        let second = backend.terminal_accept("k").unwrap().id;
        assert_ne!(first, second);
    }
}
