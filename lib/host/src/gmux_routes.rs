// Route handlers for the Rust daemon mux (PR26).
//
// Mirrors `Daemon.routeSubsystem` + `ServeHTTP` + the `dispatch*` family in
// internal/host/daemon.go, `identityHandler` in identity.go, `tailnetHandler`
// in tailnet.go and `terminalHandler` in terminal.go. Handlers enforce the
// transport contract (subsystem routing, admission gate, response envelopes)
// and forward opaque bodies to the `ExecBackend`; input validation and
// execution live behind the trait (see GMUX_PATCHES.md for the division).
//
// Response envelopes mirror Go exactly: `http.Error` writes
// `Content-Type: text/plain; charset=utf-8` + `X-Content-Type-Options:
// nosniff` with a `{message}\n` body; native/identity JSON success appends
// `\n` (json.Encoder); tailnet JSON success has no trailing newline and
// always carries `Cache-Control: no-store`.
use crate::gmux_admission::{
    is_admitted_mutation_path, valid_identity_request, valid_terminal_request,
    validate_native_request, validate_tailnet_request, AdmissionGate, RequestHead, TerminalGate,
    TerminalSlot, IDENTITY_ACTIONS,
};
use crate::gmux_backend::{BackendError, ExecBackend, TerminalSession};

// -- complete route table (method, path) --

/// Every route the mux serves: 5 identity, 6 tailnet, 1 terminal,
/// 8 project, 5 prepare, 8 factory. The smoke test dispatches each one.
pub const ROUTE_TABLE: &[(&str, &str)] = &[
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
    ("GET", "/terminal"),
    ("POST", "/profile"),
    ("POST", "/create"),
    ("POST", "/inspect"),
    ("POST", "/os"),
    ("POST", "/connection"),
    ("POST", "/lifecycle"),
    ("POST", "/access-keys"),
    ("POST", "/account"),
    ("POST", "/prepare"),
    ("POST", "/prepare-candidate"),
    ("POST", "/prepare-inspect"),
    ("POST", "/prepare-stop"),
    ("POST", "/prepare-hold"),
    ("POST", "/factory-launch"),
    ("POST", "/factory-inspect"),
    ("POST", "/factory-stop"),
    ("POST", "/factory-takeover"),
    ("POST", "/factory-output"),
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
    Respond(Vec<u8>),
    TerminalUpgrade {
        response: Vec<u8>,
        session: TerminalSession,
        slot: TerminalSlot,
    },
}

impl RouteOutcome {
    /// Unwrap a plain response (tests use this for non-terminal routes).
    pub fn into_response(self) -> Vec<u8> {
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
) -> Vec<u8> {
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
) -> Vec<u8> {
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
) -> Vec<u8> {
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
        response: websocket_upgrade_response(&key),
        session,
        slot,
    }
}

// -- response envelopes --

fn reason(status: u16) -> &'static str {
    match status {
        101 => "Switching Protocols",
        400 => "Bad Request",
        404 => "Not Found",
        405 => "Method Not Allowed",
        409 => "Conflict",
        413 => "Content Too Large",
        422 => "Unprocessable Entity",
        431 => "Request Header Fields Too Large",
        500 => "Internal Server Error",
        501 => "Not Implemented",
        502 => "Bad Gateway",
        503 => "Service Unavailable",
        _ => "Internal Server Error",
    }
}

/// Go `http.Error` envelope: text/plain + nosniff + `{message}\n`.
/// Exported for the server's parser-level rejections (same envelope).
pub fn error_response(status: u16, message: &str) -> Vec<u8> {
    let mut body = message.as_bytes().to_vec();
    body.push(b'\n');
    let mut head = format!(
        "HTTP/1.1 {status} {}\r\nContent-Type: text/plain; charset=utf-8\r\nX-Content-Type-Options: nosniff\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        reason(status),
        body.len(),
    )
    .into_bytes();
    head.extend_from_slice(&body);
    head
}

fn not_found_response() -> Vec<u8> {
    error_response(404, "404 page not found")
}

/// Native/identity JSON success: application/json + trailing `\n`.
fn json_response(json: &[u8]) -> Vec<u8> {
    let mut head = format!(
        "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        json.len() + 1,
    )
    .into_bytes();
    head.extend_from_slice(json);
    head.push(b'\n');
    head
}

/// Tailnet JSON success: no trailing newline (writeTailnetResponse);
/// `Cache-Control: no-store` is injected by `tailnet_response`.
fn tailnet_json_response(json: &[u8]) -> Vec<u8> {
    let mut head = format!(
        "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nCache-Control: no-store\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        json.len(),
    )
    .into_bytes();
    head.extend_from_slice(json);
    head
}

/// Inject `Cache-Control: no-store` into a rendered tailnet error response
/// (Go sets the header before any tailnet error write).
fn tailnet_response(mut response: Vec<u8>) -> Vec<u8> {
    const NO_STORE: &[u8] = b"Cache-Control: no-store\r\n";
    if let Some(end) = find_header_end(&response) {
        let mut out = Vec::with_capacity(response.len() + NO_STORE.len());
        out.extend_from_slice(&response[..end]);
        out.extend_from_slice(NO_STORE);
        out.extend_from_slice(&response[end..]);
        std::mem::swap(&mut response, &mut out);
    }
    response
}

fn find_header_end(response: &[u8]) -> Option<usize> {
    response
        .windows(2)
        .position(|w| w == b"\r\n")
        .map(|first_end| first_end + 2)
}

/// 101 upgrade response. Header order follows Go's sorted write
/// (Connection, Sec-WebSocket-Accept, Upgrade).
fn websocket_upgrade_response(key: &str) -> Vec<u8> {
    format!(
        "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: {}\r\nUpgrade: websocket\r\n\r\n",
        websocket_accept_key(key),
    )
    .into_bytes()
}

/// RFC 6455 accept key: base64(sha1(key + GUID)). Dependency-free: the
/// soda-host crate vendors no crypto, so SHA-1 and base64 are inline.
/// Exported for the smoke test's known-answer vector.
pub fn websocket_accept_key(key: &str) -> String {
    const GUID: &str = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11";
    let mut input = key.as_bytes().to_vec();
    input.extend_from_slice(GUID.as_bytes());
    base64_encode(&sha1(&input))
}

fn sha1(input: &[u8]) -> [u8; 20] {
    let mut h = [
        0x67452301u32,
        0xEFCDAB89,
        0x98BADCFE,
        0x10325476,
        0xC3D2E1F0,
    ];
    let mut msg = input.to_vec();
    let bit_len = (input.len() as u64) * 8;
    msg.push(0x80);
    while msg.len() % 64 != 56 {
        msg.push(0);
    }
    msg.extend_from_slice(&bit_len.to_be_bytes());
    for block in msg.chunks(64) {
        let mut w = [0u32; 80];
        for i in 0..16 {
            w[i] = u32::from_be_bytes([
                block[4 * i],
                block[4 * i + 1],
                block[4 * i + 2],
                block[4 * i + 3],
            ]);
        }
        for i in 16..80 {
            w[i] = (w[i - 3] ^ w[i - 8] ^ w[i - 14] ^ w[i - 16]).rotate_left(1);
        }
        let (mut a, mut b, mut c, mut d, mut e) = (h[0], h[1], h[2], h[3], h[4]);
        for (i, word) in w.iter().enumerate() {
            let (f, k) = match i {
                0..=19 => ((b & c) | ((!b) & d), 0x5A827999),
                20..=39 => (b ^ c ^ d, 0x6ED9EBA1),
                40..=59 => ((b & c) | (b & d) | (c & d), 0x8F1BBCDC),
                _ => (b ^ c ^ d, 0xCA62C1D6),
            };
            let tmp = a
                .rotate_left(5)
                .wrapping_add(f)
                .wrapping_add(e)
                .wrapping_add(k)
                .wrapping_add(*word);
            e = d;
            d = c;
            c = b.rotate_left(30);
            b = a;
            a = tmp;
        }
        h[0] = h[0].wrapping_add(a);
        h[1] = h[1].wrapping_add(b);
        h[2] = h[2].wrapping_add(c);
        h[3] = h[3].wrapping_add(d);
        h[4] = h[4].wrapping_add(e);
    }
    let mut out = [0u8; 20];
    for (i, word) in h.iter().enumerate() {
        out[4 * i..4 * i + 4].copy_from_slice(&word.to_be_bytes());
    }
    out
}

fn base64_encode(input: &[u8]) -> String {
    const ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity(input.len().div_ceil(3) * 4);
    for chunk in input.chunks(3) {
        let mut n = 0u32;
        for &b in chunk {
            n = (n << 8) | u32::from(b);
        }
        n <<= 8 * (3 - chunk.len());
        for i in 0..4 {
            if i <= chunk.len() {
                out.push(ALPHABET[((n >> (18 - 6 * i)) & 63) as usize] as char);
            } else {
                out.push('=');
            }
        }
    }
    out
}
