use std::sync::{Arc, Mutex, MutexGuard};

// -- body limits (daemon.go ServeHTTP, identityHandler, tailnetHandler) --

/// Default native body cap: 64 KiB.
pub const BODY_LIMIT_DEFAULT: usize = 65536;
/// Large native bodies: /prepare, /prepare-candidate, /factory-launch.
pub const BODY_LIMIT_LARGE: usize = 1 << 20;
/// Identity operations carry delivery envelopes: 384 KiB.
pub const BODY_LIMIT_IDENTITY: usize = 384 << 10;

// -- route tables (daemon.go hasNativeCleanPath / dispatchOperation) --

/// Native paths that MUST arrive without query string or escapes
/// (Go: `r.URL.RawQuery == "" && !r.URL.ForceQuery && r.URL.RawPath == ""`).
pub const NATIVE_CLEAN_PATHS: &[&str] = &[
    "/lifecycle",
    "/access-keys",
    "/profile",
    "/create",
    "/os",
    "/prepare",
    "/prepare-candidate",
    "/prepare-inspect",
    "/prepare-stop",
    "/prepare-hold",
    "/factory-launch",
    "/factory-inspect",
    "/factory-stop",
    "/factory-takeover",
    "/factory-output",
    "/factory-harness",
    "/factory-export",
    "/factory-candidate-inspect",
];

/// Native paths serialized through the shared mutation gate
/// (Go: `isAdmittedMutationPath`). `/create` owns the same gate inside its
/// dispatch instead, and factory routes stay off it (per-run locks govern).
pub const ADMITTED_MUTATION_PATHS: &[&str] = &[
    "/lifecycle",
    "/access-keys",
    "/account",
    "/prepare",
    "/prepare-candidate",
    "/prepare-stop",
    "/prepare-hold",
];

/// Identity actions after the `/identity/` prefix (identity.go).
pub const IDENTITY_ACTIONS: &[&str] = &["launch", "validate", "stop", "start", "finish"];

/// Tailnet actions after the `/tailnet/` prefix (tailnet.go).
pub const TAILNET_ACTIONS: &[&str] = &[
    "settings",
    "host",
    "enrollment",
    "options",
    "project",
    "policy",
];

/// Terminal stream cap: 2 * terminalLimit, terminalLimit = 64 (service.go).
pub const TERMINAL_STREAM_CAP: usize = 128;

/// First-frame terminal request cap in bytes (service.go readTerminalRequest).
pub const TERMINAL_REQUEST_LIMIT: usize = 4096;

/// Terminal frame read limit (terminal/types.go FrameLimit).
pub const TERMINAL_FRAME_LIMIT: usize = 131072;

// -- parsed request head --

/// What the server parser extracts from the HTTP head for admission.
///
/// `has_query` folds Go's `RawQuery != "" || ForceQuery` (any `?` in the
/// target); `escaped` folds `RawPath != ""` (any `%` escape in the target,
/// i.e. the raw target is not the canonical encoding of the decoded path);
/// `origin_present` folds `len(Header.Values("Origin")) != 0` (any Origin
/// header line, even empty). `path` is the percent-decoded target.
#[derive(Debug, Clone)]
pub struct RequestHead {
    pub method: String,
    pub path: String,
    pub has_query: bool,
    pub escaped: bool,
    pub origin_present: bool,
    /// True when an `Upgrade: websocket` token is present (case-insensitive,
    /// comma-separated values honored).
    pub upgrade_websocket: bool,
    /// First `Sec-WebSocket-Key` value, if any.
    pub ws_key: Option<String>,
}

impl RequestHead {
    /// Test/dispatch helper: a plain POST with no query, escapes or Origin.
    pub fn post(path: &str) -> RequestHead {
        RequestHead {
            method: "POST".to_string(),
            path: path.to_string(),
            has_query: false,
            escaped: false,
            origin_present: false,
            upgrade_websocket: false,
            ws_key: None,
        }
    }
}

// -- native request validation (daemon.go validateNativeOperationRequest) --

/// Native rejection: status + fixed wire message (daemon.go).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct NativeRejection {
    pub status: u16,
    pub message: &'static str,
}

/// Order matters: Go checks the clean path first (400), then POST (405).
pub fn validate_native_request(head: &RequestHead) -> Result<(), NativeRejection> {
    if NATIVE_CLEAN_PATHS.contains(&head.path.as_str()) && (head.has_query || head.escaped) {
        return Err(NativeRejection {
            status: 400,
            message: "invalid native operation path",
        });
    }
    if head.method != "POST" {
        return Err(NativeRejection {
            status: 405,
            message: "POST required",
        });
    }
    Ok(())
}

/// True for the mutation paths serialized on the shared gate.
pub fn is_admitted_mutation_path(path: &str) -> bool {
    ADMITTED_MUTATION_PATHS.contains(&path)
}

/// Body cap for a native/identity/tailnet path (daemon.go ServeHTTP).
pub fn body_limit_for(path: &str) -> usize {
    if path == "/prepare" || path == "/prepare-candidate" || path == "/factory-launch" {
        BODY_LIMIT_LARGE
    } else if path.starts_with("/identity/") {
        BODY_LIMIT_IDENTITY
    } else {
        BODY_LIMIT_DEFAULT
    }
}

// -- subsystem validators --

/// Identity requests: POST, no query, no escapes, no Origin header
/// (identity.go `validIdentityRequest`).
pub fn valid_identity_request(head: &RequestHead) -> bool {
    head.method == "POST" && !head.has_query && !head.escaped && !head.origin_present
}

/// Tailnet action validation (tailnet.go `validateTailnetRequest`).
/// Returns the action on success, the HTTP status on failure.
pub fn validate_tailnet_request(head: &RequestHead) -> Result<&'static str, u16> {
    if head.method != "POST" || head.has_query || head.escaped {
        return Err(400);
    }
    let action = head.path.strip_prefix("/tailnet/").unwrap_or("");
    match TAILNET_ACTIONS.iter().find(|a| **a == action) {
        Some(found) => Ok(found),
        None => Err(404),
    }
}

/// Terminal attach requests: GET, no query, no escapes, no Origin header
/// (terminal/service.go `validPrivateTerminalRequest`).
pub fn valid_terminal_request(head: &RequestHead) -> bool {
    head.method == "GET" && !head.has_query && !head.escaped && !head.origin_present
}

// -- shared mutation gate (daemon.go acquireAdmission) --

/// Globally serialized mutation gate with a const constructor, so mux
/// structs stay usable without a constructor like Go's lazy `sync.Once`.
///
/// Go lets cancelled waiters leave the queue; the skeleton offers a
/// blocking `acquire` plus a non-blocking `try_acquire` and documents the
/// timeout wiring for the integrator (GMUX_PATCHES.md): the server applies
/// per-request deadlines around dispatch, so a stuck holder cannot wedge
/// the mux past the request horizon.
#[derive(Debug, Default)]
pub struct AdmissionGate {
    held: Mutex<()>,
}

impl AdmissionGate {
    pub const fn new() -> AdmissionGate {
        AdmissionGate {
            held: Mutex::new(()),
        }
    }

    /// Block until the gate is free; guards release on drop.
    /// Poisoning is unrecoverable by construction (holders never panic
    /// while holding it), so a poisoned mutex is treated as free.
    pub fn acquire(&self) -> AdmissionGuard<'_> {
        let guard = self.held.lock().unwrap_or_else(|e| e.into_inner());
        AdmissionGuard { _guard: guard }
    }

    /// Take the gate only if it is free right now.
    pub fn try_acquire(&self) -> Option<AdmissionGuard<'_>> {
        self.held
            .try_lock()
            .map(|guard| AdmissionGuard { _guard: guard })
            .ok()
    }
}

/// Held admission; dropping releases the gate.
pub struct AdmissionGuard<'a> {
    _guard: MutexGuard<'a, ()>,
}

// -- terminal stream cap (terminal/service.go register) --

/// Counts live terminal streams against `TERMINAL_STREAM_CAP`.
///
/// Slots release on drop, mirroring Go's register/unregister cleanup that
/// runs even when the websocket handshake fails.
#[derive(Debug, Default)]
pub struct TerminalGate {
    live: Arc<Mutex<usize>>,
}

impl TerminalGate {
    pub fn new() -> TerminalGate {
        TerminalGate {
            live: Arc::new(Mutex::new(0)),
        }
    }

    /// Reserve a stream slot; None when the cap is reached.
    pub fn try_register(&self) -> Option<TerminalSlot> {
        let mut live = self.live.lock().unwrap_or_else(|e| e.into_inner());
        if *live >= TERMINAL_STREAM_CAP {
            return None;
        }
        *live += 1;
        Some(TerminalSlot {
            live: Arc::clone(&self.live),
        })
    }

    /// Current live count (observability for tests).
    pub fn live(&self) -> usize {
        *self.live.lock().unwrap_or_else(|e| e.into_inner())
    }
}

/// Reserved terminal stream slot; dropping frees it. Plain data over a
/// locked refcount, so the server can move it into the pump thread.
pub struct TerminalSlot {
    live: Arc<Mutex<usize>>,
}

impl Drop for TerminalSlot {
    fn drop(&mut self) {
        let mut live = self.live.lock().unwrap_or_else(|e| e.into_inner());
        *live = live.saturating_sub(1);
    }
}
