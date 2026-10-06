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
#[path = "daemon/response.rs"]
mod response;

#[path = "daemon/routes.rs"]
mod routes;

#[path = "daemon/websocket.rs"]
mod websocket;

pub use self::response::error_response;
pub use self::routes::{dispatch, DaemonConfig, RouteOutcome};
pub use self::websocket::websocket_accept_key;

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
