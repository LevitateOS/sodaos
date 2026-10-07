use crate::gmux_admission::{
    is_admitted_mutation_path, valid_identity_request, valid_terminal_request,
    validate_native_request, validate_tailnet_request, AdmissionGate, RequestHead, TerminalGate,
    TerminalSlot, IDENTITY_ACTIONS,
};
use crate::gmux_backend::{BackendError, ExecBackend, TerminalSession};

use super::response::{
    error_response, json_response, not_found_response, tailnet_json_response, tailnet_response,
    HttpResponse,
};
use super::websocket::websocket_upgrade_response;

// -- daemon config surface the mux needs --

/// Minimal config the mux reads. Folds Go's nil-runtime checks
/// (`d.Terminal == nil`, `d.Tailnet == nil`) into booleans and carries the
/// config image Go stamps onto the factory harness pin.
#[derive(Debug, Clone)]
pub struct DaemonConfig {
    /// Appliance project image (`Config.Image`), stamped onto harness pins.
    pub image: String,
    /// False when tailnet management is disabled (Go: `d.Tailnet == nil`).
    pub tailnet_management: bool,
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
            tailnet_management: true,
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
    // Read-only observations overlap mutations; admitted mutations and
    // /create take the shared writer gate (ServeHTTP + dispatchCreate).
    let _guard = if is_admitted_mutation_path(&head.path) || head.path == "/create" {
        Some(gate.acquire())
    } else {
        None
    };
    let result = match head.path.as_str() {
        "/profile" => backend.profile(body),
        "/create" => backend.create(body),
        "/inspect" => backend.inspect(body),
        "/os" => backend.observe_os(body),
        "/connection" => backend.connection(body),
        "/lifecycle" => backend.lifecycle(body),
        "/access-keys" => backend.access_keys(body),
        "/account" => backend.account(body),
        "/prepare" => backend.prepare(body),
        "/prepare-candidate" => backend.prepare_candidate(body),
        "/prepare-inspect" => backend.inspect_preparation(body),
        "/prepare-stop" => backend.stop_preparation(body),
        "/prepare-hold" => backend.hold_preparation(body),
        "/factory-launch" => backend.factory_launch(body),
        "/factory-inspect" => backend.factory_inspect(body),
        "/factory-stop" => backend.factory_stop(body),
        "/factory-takeover" => backend.factory_takeover(body),
        "/factory-output" => backend.factory_output(body),
        "/factory-harness" => backend.factory_harness(body, &config.image),
        "/factory-export" => backend.factory_export(body),
        "/factory-candidate-inspect" => backend.factory_candidate_inspect(body),
        _ => Err(BackendError::NotFound),
    };
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
    if !config.tailnet_management {
        return tailnet_response(error_response(503, "Tailnet management disabled"));
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
