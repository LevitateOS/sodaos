// Runtime contract between the daemon route layer and its executor.
//
// Bodies cross this boundary as opaque JSON bytes: the mux owns
// transport-level admission (method, path, query, origin, body limits, the
// mutation gate) and the backend owns wire-type decoding and execution.
// Success payloads MUST be JSON values without a trailing newline; the
// route layer appends the newline to match Go's json.Encoder envelope.
use std::os::unix::net::UnixStream;
use tungstenite::protocol::WebSocket;

/// Backend failure modes, one per distinct HTTP mapping in daemon.go.
//
// The route layer maps each variant per subsystem; messages never
// carry input or provider text, mirroring the Go handlers.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BackendError {
    /// Unknown run or unknown operation: native 404.
    NotFound,
    /// Runtime missing or disabled: native 503, tailnet 503, identity 409.
    Unavailable,
    /// Factory output incarnation changed: native 409.
    OutputStale,
    /// Factory run incarnation changed: native 409.
    ExportStale,
    /// Factory export of a candidate run: native 422.
    ExportCandidate,
    /// Factory export outside bounds: native 413.
    ExportBounds,
    /// Input failed validation: tailnet 400, otherwise native 500.
    Invalid,
    /// Operation refused: native 500, identity 409.
    Denied,
    /// Stub-only result for intentionally unimplemented calls. Every
    /// subsystem maps it to 501; production backends must not return it.
    Unimplemented,
    /// Anything else: native 500, tailnet 502, identity 409.
    Internal,
}

/// Opaque handle for an admitted terminal session.
//
// The route layer obtains it from `terminal_accept` before emitting the 101
// upgrade; the server loop moves it into the pump thread so the session and
// its stream slot share one lifetime.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct TerminalSession {
    /// Backend-defined session discriminator (0 for the stub).
    pub id: u64,
}

/// Project + terminal + tailnet + identity execution surface for the mux.
//
// One method per Go `Daemon.dispatch*` leaf, with two deliberate folds:
// identity validate/stop/start/finish share `identity_action` (the Go daemon
// routes all four through `Terminal.Identity` / `Muse.Muse` by action name),
// and the six tailnet actions share `tailnet` (Go's `dispatchTailnetAction`).
// Everything takes opaque bodies so sibling wire types stay sibling-owned.
pub trait ExecBackend: Send + Sync {
    // -- project routes (daemon.go dispatchProfile/dispatchCreate/
    //    dispatchCreateTargeted/dispatchMutation) --
    fn profile(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn create(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn observe_os(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn connection(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn lifecycle(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn access_keys(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn account(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    /// Read-only native Project privilege observation.
    fn project_access(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;

    // -- prepare routes (daemon.go dispatchPrepare) --
    fn prepare(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn prepare_candidate(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn inspect_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn stop_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn hold_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;

    // -- factory routes (daemon.go dispatchFactory) --
    fn factory_launch(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn factory_inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn factory_stop(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn factory_takeover(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn factory_output(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    /// `image` is the daemon config image Go stamps onto the harness pin
    /// (`factory.HarnessPin()` + `pin.Image = d.Config.Image`) before
    /// validation; the mux passes it so the backend need not read config.
    fn factory_harness(&self, body: &[u8], image: &str) -> Result<Vec<u8>, BackendError>;
    fn factory_export(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    fn factory_candidate_inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;

    // -- identity routes (identity.go identityOperation) --
    fn identity_launch(&self, body: &[u8]) -> Result<Vec<u8>, BackendError>;
    /// `action` is one of `validate`, `stop`, `start`, `finish`. Lease-kind
    /// routing (factory-run callbacks vs muse leases vs terminal leases) is
    /// backend-internal: the backend decodes the delivery wire envelope and
    /// dispatches like Go's `identityOperation` / `factoryIdentityOperation`.
    /// Unknown actions MUST be denied, never executed.
    fn identity_action(&self, action: &str, body: &[u8]) -> Result<Vec<u8>, BackendError>;

    // -- tailnet routes (tailnet.go dispatchTailnetAction) --
    /// `action` is one of `settings`, `host`, `enrollment`, `options`,
    /// `project`, `policy`. The container-identity fencing around
    /// project/policy observations (Go's `ProjectContainer` before/after
    /// check) is backend-internal: the backend owns both the project
    /// runtime and the tailnet control. Unknown actions MUST be rejected.
    fn tailnet(&self, action: &str, body: &[u8]) -> Result<Vec<u8>, BackendError>;

    // -- terminal route (terminal.go terminalHandler + service.go Handler) --
    /// Admission half of the websocket attach: validates executor-side
    /// preconditions (the Go `attachLauncher` assertion and stream
    /// registration live here) and mints the session the pump loop needs.
    /// The mux performs the HTTP upgrade only after this succeeds.
    /// `key` is the client's `Sec-WebSocket-Key` for logging correlation.
    fn terminal_accept(&self, key: &str) -> Result<TerminalSession, BackendError>;
    /// Transport half: owns the upgraded stream until the session ends
    /// (first-frame request, expiry deadline, launch, bidirectional pump).
    /// Consumes the stream; the mux never touches it after handoff.
    fn pump_terminal(
        &self,
        stream: WebSocket<UnixStream>,
        session: TerminalSession,
        shutdown: std::sync::Arc<std::sync::atomic::AtomicBool>,
    ) -> Result<(), BackendError>;
}

/// Test backend: every operation reports unimplemented.
//
// The route layer maps `Unimplemented` to 501 in every subsystem. Tests use
// this stub to exercise routing without running the native executors.
#[derive(Debug, Default, Clone, Copy)]
pub struct StubBackend;

impl ExecBackend for StubBackend {
    fn profile(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn create(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn inspect(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn observe_os(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn connection(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn lifecycle(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn access_keys(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn account(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn project_access(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn prepare(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn prepare_candidate(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn inspect_preparation(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn stop_preparation(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn hold_preparation(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn factory_launch(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn factory_inspect(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn factory_stop(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn factory_takeover(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn factory_output(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn factory_harness(&self, _body: &[u8], _image: &str) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn factory_export(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn factory_candidate_inspect(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn identity_launch(&self, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn identity_action(&self, _action: &str, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn tailnet(&self, _action: &str, _body: &[u8]) -> Result<Vec<u8>, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn terminal_accept(&self, _key: &str) -> Result<TerminalSession, BackendError> {
        Err(BackendError::Unimplemented)
    }
    fn pump_terminal(
        &self,
        _stream: WebSocket<UnixStream>,
        _session: TerminalSession,
        _shutdown: std::sync::Arc<std::sync::atomic::AtomicBool>,
    ) -> Result<(), BackendError> {
        Err(BackendError::Unimplemented)
    }
}
