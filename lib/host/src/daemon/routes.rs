use super::admission::{
    is_admitted_mutation_path, valid_identity_request, valid_terminal_request,
    validate_native_request, validate_tailnet_request, AdmissionGate, AdmissionGuard, RequestHead,
    TerminalGate, TerminalSlot, IDENTITY_ACTIONS,
};
use super::backend::{BackendError, ExecBackend, TerminalSession};
use crate::{tailnet_lifecycle, tcontrol_wire};

use super::response::{
    error_response, json_response, not_found_response, tailnet_json_response, tailnet_response,
    HttpResponse,
};
use super::websocket::websocket_upgrade_response;

/// Every route the daemon serves: 5 identity, 8 tailnet, 1 terminal,
/// 9 project, 6 prepare, and 9 factory.
pub const ROUTE_TABLE: &[(&str, &str)] = &[
    ("POST", "/application"),
    ("POST", "/identity/launch"),
    ("POST", "/identity/validate"),
    ("POST", "/identity/stop"),
    ("POST", "/identity/start"),
    ("POST", "/identity/finish"),
    ("POST", "/tailnet/settings"),
    ("POST", "/tailnet/host"),
    ("POST", "/tailnet/enrollment"),
    ("POST", "/tailnet/options"),
    ("POST", "/tailnet/project"),
    ("POST", "/tailnet/policy"),
    ("POST", "/tailnet/prepare"),
    ("POST", "/tailnet/lifecycle"),
    ("GET", "/terminal"),
    ("POST", "/profile"),
    ("POST", "/create"),
    ("POST", "/inspect"),
    ("POST", "/os"),
    ("POST", "/connection"),
    ("POST", "/lifecycle"),
    ("POST", "/access-keys"),
    ("POST", "/account"),
    ("POST", "/project-access"),
    ("POST", "/terminal-support"),
    ("POST", "/prepare"),
    ("POST", "/prepare-candidate"),
    ("POST", "/prepare-context"),
    ("POST", "/prepare-inspect"),
    ("POST", "/prepare-stop"),
    ("POST", "/prepare-hold"),
    ("POST", "/factory-launch"),
    ("POST", "/factory-inspect"),
    ("POST", "/factory-stop"),
    ("POST", "/factory-takeover"),
    ("POST", "/factory-output"),
    ("POST", "/factory-task-events"),
    ("POST", "/factory-harness"),
    ("POST", "/factory-export"),
    ("POST", "/factory-candidate-inspect"),
];

// -- daemon config surface the mux needs --

/// Minimal config the mux reads. Folds Go's nil-runtime checks
/// (`d.Terminal == nil`, `d.Tailnet == nil`) into booleans and carries the
/// config image Go stamps onto the factory harness pin.
#[derive(Debug, Clone)]
pub struct DaemonConfig {
    /// Appliance project image (`Config.Image`), stamped onto harness pins.
    pub image: String,
    /// False when tailnet management is disabled (Go: `d.Tailnet == nil`).
    /// Trusted tailnet package version from the selected release payload.
    /// Empty means no trusted release was loaded; lifecycle prepare and
    /// apply are then unavailable, never default-open.
    pub tailnet_package_version: String,
    /// Trusted tailnet package archive digest from the selected release.
    pub tailnet_package_sha256: String,
    /// Trusted current release payload path for prepare's fixed importer
    /// ensure. Empty disables prepare; it never defaults.
    pub release_payload: String,
    /// False when no terminal service is wired (Go: `d.Terminal == nil`).
    pub terminal_available: bool,
    /// False when identity has no terminal runtime (Go: `d.Terminal == nil`
    /// in `identityHandler`).
    pub identity_available: bool,
}

impl DaemonConfig {
    /// Test helper: everything enabled, empty image.
    pub fn all_enabled() -> DaemonConfig {
        DaemonConfig {
            image: String::new(),
            tailnet_package_version: String::new(),
            tailnet_package_sha256: String::new(),
            release_payload: String::new(),
            terminal_available: true,
            identity_available: true,
        }
    }
}

// -- dispatch outcome --

/// What a route handler produces: either complete response bytes, or a
/// terminal upgrade (101 bytes already rendered) plus the session and the
/// stream slot the server must hold for the pump lifetime.
pub enum RouteOutcome {
    Respond(HttpResponse),
    TerminalUpgrade {
        response: HttpResponse,
        session: TerminalSession,
        slot: TerminalSlot,
    },
}

impl RouteOutcome {
    /// Unwrap a plain response (tests use this for non-terminal routes).
    pub fn into_response(self) -> HttpResponse {
        match self {
            RouteOutcome::Respond(bytes) => bytes,
            RouteOutcome::TerminalUpgrade { .. } => {
                panic!("expected a plain response, got a terminal upgrade")
            }
        }
    }
}

// -- top-level dispatch (daemon.go routeSubsystem + ServeHTTP) --

/// Dispatch one request. `body_complete` reports whether the body arrived
/// within the route limit; a truncated body is treated as a decode failure
/// exactly like Go's `MaxBytesReader` surfacing through strict decode.
pub fn dispatch<B: ExecBackend + ?Sized>(
    backend: &B,
    config: &DaemonConfig,
    gate: &AdmissionGate,
    terminal_gate: &TerminalGate,
    head: &RequestHead,
    body: &[u8],
    body_complete: bool,
) -> RouteOutcome {
    if head.path.starts_with("/identity/") {
        return RouteOutcome::Respond(dispatch_identity(
            backend,
            config,
            head,
            body,
            body_complete,
        ));
    }
    if head.path.starts_with("/tailnet/") {
        return RouteOutcome::Respond(dispatch_tailnet(backend, config, head, body, body_complete));
    }
    if head.path == "/terminal" {
        return dispatch_terminal(backend, config, terminal_gate, head);
    }
    RouteOutcome::Respond(dispatch_native(
        backend,
        config,
        gate,
        head,
        body,
        body_complete,
    ))
}

// -- native operations (daemon.go dispatchOperation + ServeHTTP errors) --

/// Take the shared writer gate for admitted mutations and `/create`.
/// Read-only observations run without it (ServeHTTP + dispatchCreate).
fn acquire_native_writer<'a>(
    gate: &'a AdmissionGate,
    path: &str,
) -> Result<Option<AdmissionGuard<'a>>, HttpResponse> {
    if is_admitted_mutation_path(path) || path == "/create" {
        match gate.try_acquire() {
            Some(guard) => Ok(Some(guard)),
            None => Err(error_response(503, "service busy")),
        }
    } else {
        Ok(None)
    }
}

fn invoke_native_core<B: ExecBackend + ?Sized>(
    backend: &B,
    path: &str,
    body: &[u8],
) -> Option<Result<Vec<u8>, BackendError>> {
    match path {
        "/application" => Some(backend.application(body)),
        "/profile" => Some(backend.profile(body)),
        "/create" => Some(backend.create(body)),
        "/inspect" => Some(backend.inspect(body)),
        "/os" => Some(backend.observe_os(body)),
        "/connection" => Some(backend.connection(body)),
        "/lifecycle" => Some(backend.lifecycle(body)),
        _ => None,
    }
}

fn invoke_native_project<B: ExecBackend + ?Sized>(
    backend: &B,
    path: &str,
    body: &[u8],
) -> Option<Result<Vec<u8>, BackendError>> {
    match path {
        "/access-keys" => Some(backend.access_keys(body)),
        "/account" => Some(backend.account(body)),
        "/project-access" => Some(backend.project_access(body)),
        "/terminal-support" => Some(backend.terminal_support(body)),
        _ => None,
    }
}

fn invoke_native_prepare<B: ExecBackend + ?Sized>(
    backend: &B,
    path: &str,
    body: &[u8],
) -> Option<Result<Vec<u8>, BackendError>> {
    match path {
        "/prepare" => Some(backend.prepare(body)),
        "/prepare-candidate" => Some(backend.prepare_candidate(body)),
        "/prepare-context" => Some(backend.prepare_context(body)),
        "/prepare-inspect" => Some(backend.inspect_preparation(body)),
        "/prepare-stop" => Some(backend.stop_preparation(body)),
        "/prepare-hold" => Some(backend.hold_preparation(body)),
        _ => None,
    }
}

fn invoke_native_factory<B: ExecBackend + ?Sized>(
    backend: &B,
    config: &DaemonConfig,
    path: &str,
    body: &[u8],
) -> Option<Result<Vec<u8>, BackendError>> {
    match path {
        "/factory-launch" => Some(backend.factory_launch(body)),
        "/factory-inspect" => Some(backend.factory_inspect(body)),
        "/factory-stop" => Some(backend.factory_stop(body)),
        "/factory-takeover" => Some(backend.factory_takeover(body)),
        "/factory-output" => Some(backend.factory_output(body)),
        "/factory-task-events" => Some(backend.factory_task_events(body)),
        "/factory-harness" => Some(backend.factory_harness(body, &config.image)),
        "/factory-export" => Some(backend.factory_export(body)),
        "/factory-candidate-inspect" => Some(backend.factory_candidate_inspect(body)),
        _ => None,
    }
}

fn native_error_response(result: Result<Vec<u8>, BackendError>) -> HttpResponse {
    match result {
        Ok(json) => json_response(&json),
        Err(BackendError::NotFound) => error_response(404, "404 page not found"),
        Err(BackendError::Unavailable) => error_response(503, "factory runtime unavailable"),
        Err(BackendError::OutputStale) => error_response(409, "factory output incarnation changed"),
        Err(BackendError::ExportStale) => error_response(409, "factory run incarnation changed"),
        Err(BackendError::ExportCandidate) => {
            error_response(422, "export candidate is not recorded")
        }
        Err(BackendError::ExportBounds) => error_response(413, "candidate export exceeds bounds"),
        Err(BackendError::Unimplemented) => error_response(501, "not implemented"),
        Err(_) => error_response(500, "native operation failed; inspect operator journal"),
    }
}

fn dispatch_native<B: ExecBackend + ?Sized>(
    backend: &B,
    config: &DaemonConfig,
    gate: &AdmissionGate,
    head: &RequestHead,
    body: &[u8],
    body_complete: bool,
) -> HttpResponse {
    if let Err(reject) = validate_native_request(head) {
        return error_response(reject.status, reject.message);
    }
    if !body_complete {
        // Over-limit bodies fail strict decode in Go -> generic 500.
        return error_response(500, "native operation failed; inspect operator journal");
    }
    let _guard = match acquire_native_writer(gate, &head.path) {
        Ok(guard) => guard,
        Err(response) => return response,
    };
    let result = invoke_native_core(backend, &head.path, body)
        .or(invoke_native_project(backend, &head.path, body))
        .or(invoke_native_prepare(backend, &head.path, body))
        .or(invoke_native_factory(backend, config, &head.path, body))
        .unwrap_or(Err(BackendError::NotFound));
    native_error_response(result)
}

// -- identity operations (identity.go identityHandler) --

fn dispatch_identity<B: ExecBackend + ?Sized>(
    backend: &B,
    config: &DaemonConfig,
    head: &RequestHead,
    body: &[u8],
    body_complete: bool,
) -> HttpResponse {
    if !valid_identity_request(head) || !config.identity_available {
        return error_response(400, "invalid identity operation");
    }
    if !body_complete {
        return error_response(409, "identity operation unconfirmed");
    }
    let action = head.path.strip_prefix("/identity/").unwrap_or("");
    if !IDENTITY_ACTIONS.contains(&action) {
        // Go forwards unknown actions to Terminal.Identity, which denies
        // them; the mux denies directly with the same 409.
        return error_response(409, "identity operation unconfirmed");
    }
    let result = if action == "launch" {
        backend.identity_launch(body)
    } else {
        backend.identity_action(action, body)
    };
    match result {
        Ok(json) => json_response(&json),
        Err(BackendError::Unimplemented) => error_response(501, "not implemented"),
        Err(_) => error_response(409, "identity operation unconfirmed"),
    }
}

// -- tailnet operations (tailnet.go tailnetHandler) --

fn dispatch_tailnet<B: ExecBackend + ?Sized>(
    backend: &B,
    config: &DaemonConfig,
    head: &RequestHead,
    body: &[u8],
    body_complete: bool,
) -> HttpResponse {
    let action = match validate_tailnet_request(head) {
        Ok(action) => action,
        Err(404) => return tailnet_response(not_found_response()),
        Err(_) => return tailnet_response(error_response(400, "invalid Tailnet operation")),
    };
    if action == "prepare" || action == "lifecycle" {
        if let Err(status) = validate_lifecycle_call(config, action, body, body_complete) {
            return tailnet_response(error_response(status, tailnet_unavailable()));
        }
    }
    if !body_complete {
        return tailnet_response(error_response(400, tailnet_unavailable()));
    }
    match backend.tailnet(action, body) {
        Ok(json) => {
            if json.len() > 65536 {
                return tailnet_response(error_response(502, tailnet_unavailable()));
            }
            tailnet_json_response(&json)
        }
        Err(BackendError::Invalid) => tailnet_response(error_response(400, tailnet_unavailable())),
        Err(BackendError::OutputStale) | Err(BackendError::ExportStale) => {
            tailnet_response(error_response(409, tailnet_unavailable()))
        }
        Err(BackendError::ExportCandidate) => {
            tailnet_response(error_response(422, tailnet_unavailable()))
        }
        Err(BackendError::Unavailable) => {
            tailnet_response(error_response(503, tailnet_unavailable()))
        }
        Err(BackendError::Unimplemented) => {
            tailnet_response(error_response(501, "not implemented"))
        }
        Err(_) => tailnet_response(error_response(502, tailnet_unavailable())),
    }
}

/// Validate one lifecycle call before dispatch (RCX-2 T01). The Go service
/// forwards the manager-verified SDK call verbatim; this layer re-checks
/// the wire strictly, scopes it to the required controller and its tailnet
/// target, binds the endpoint to its operation (prepare carries activate
/// only), and matches the artifact pin against the trusted release. Any
/// mismatch stops before the backend, so rejected, replayed, wrong-variant
/// or unbound authority causes zero effects. Statuses: 400 malformed,
/// 409 busy or changed identity, 503 unavailable without a trusted pin.
fn validate_lifecycle_call(
    config: &DaemonConfig,
    action: &str,
    body: &[u8],
    body_complete: bool,
) -> Result<(), u16> {
    if !body_complete {
        return Err(400);
    }
    let call = match tcontrol_wire::decode_lifecycle_call(body) {
        Ok(call) => call,
        Err(_) => return Err(400),
    };
    let now_unix = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|elapsed| elapsed.as_secs() as i64)
        .unwrap_or(0);
    if tcontrol_wire::validate_lifecycle_call(&call, now_unix).is_err() {
        return Err(400);
    }
    let scoped = call.controller_id == "soda" && call.package_id == "tailnet";
    let endpoint_ok = if action == "prepare" {
        call.operation == tcontrol_wire::LIFECYCLE_ACTIVATE
    } else {
        call.operation == tcontrol_wire::LIFECYCLE_OBSERVE
            || call.operation == tcontrol_wire::LIFECYCLE_ACTIVATE
            || call.operation == tcontrol_wire::LIFECYCLE_DEACTIVATE
    };
    if !scoped || !endpoint_ok {
        return Err(409);
    }
    let pin = tailnet_lifecycle::ReleasePin {
        package_version: config.tailnet_package_version.clone(),
        package_sha256: config.tailnet_package_sha256.clone(),
    };
    match tailnet_lifecycle::match_release_pin(&call, &pin) {
        Ok(()) => Ok(()),
        Err(message) => {
            if message.starts_with("conflict") {
                Err(409)
            } else {
                Err(503)
            }
        }
    }
}

fn tailnet_unavailable() -> &'static str {
    "Tailnet operation unavailable or unconfirmed; observe before retrying"
}

// -- terminal attach (terminal.go terminalHandler + service.go Handler) --

fn dispatch_terminal<B: ExecBackend + ?Sized>(
    backend: &B,
    config: &DaemonConfig,
    terminal_gate: &TerminalGate,
    head: &RequestHead,
) -> RouteOutcome {
    let invalid = || RouteOutcome::Respond(error_response(400, "invalid private terminal request"));
    if !valid_terminal_request(head) {
        return invalid();
    }
    if !config.terminal_available {
        return RouteOutcome::Respond(error_response(503, "terminal unavailable"));
    }
    // Go's websocket.Accept rejects non-upgrade handshakes; the mux rejects
    // them explicitly with the same 400 family instead of attempting a 101.
    let key = match (&head.upgrade_websocket, &head.ws_key) {
        (true, Some(key)) => key.clone(),
        _ => return invalid(),
    };
    let Some(websocket_request) = head.websocket_request.as_ref() else {
        return invalid();
    };
    let response = websocket_upgrade_response(websocket_request);
    if response.status() != hyper::StatusCode::SWITCHING_PROTOCOLS {
        return RouteOutcome::Respond(response);
    }
    let slot = match terminal_gate.try_register() {
        Some(slot) => slot,
        None => return RouteOutcome::Respond(error_response(503, "terminal unavailable")),
    };
    let session = match backend.terminal_accept(&key) {
        Ok(session) => session,
        Err(BackendError::Unimplemented) => {
            return RouteOutcome::Respond(error_response(501, "not implemented"));
        }
        Err(_) => return RouteOutcome::Respond(error_response(503, "terminal unavailable")),
    };
    RouteOutcome::TerminalUpgrade {
        response,
        session,
        slot,
    }
}
