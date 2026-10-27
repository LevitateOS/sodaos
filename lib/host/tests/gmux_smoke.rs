// Smoke tests for the Rust daemon transport.
//
// The daemon modules are wired into lib.rs; this harness exercises the actual
// library owners. If these tests pass, admission matches daemon.go and
// every route dispatches.
use soda_host::daemon::admission::{
    body_limit_for, is_admitted_mutation_path, valid_identity_request, valid_terminal_request,
    validate_native_request, validate_tailnet_request, AdmissionGate, RequestHead, TerminalGate,
    ADMITTED_MUTATION_PATHS, BODY_LIMIT_DEFAULT, BODY_LIMIT_IDENTITY, BODY_LIMIT_LARGE,
    IDENTITY_ACTIONS, NATIVE_CLEAN_PATHS, TAILNET_ACTIONS, TERMINAL_FRAME_LIMIT,
    TERMINAL_REQUEST_LIMIT, TERMINAL_STREAM_CAP,
};
use soda_host::daemon::backend::{BackendError, ExecBackend, StubBackend, TerminalSession};
use soda_host::daemon::response::HttpResponse;
use soda_host::daemon::routes::{dispatch, DaemonConfig, RouteOutcome, ROUTE_TABLE};
use soda_host::daemon::server::{systemd_listener, Server};
use std::io::{Read, Write};
use std::os::unix::net::{UnixListener, UnixStream};
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::{Arc, Mutex};

// -- helpers --

fn head(method: &str, path: &str) -> RequestHead {
    let mut h = RequestHead::post(path);
    h.method = method.to_string();
    h
}

trait ResponseView {
    fn status(&self) -> u16;
    fn headers_and_body(&self) -> String;
    fn body_bytes(&self) -> Vec<u8>;
    fn header_value(&self, name: &str) -> Option<String>;
}
impl ResponseView for Vec<u8> {
    fn status(&self) -> u16 {
        let s = String::from_utf8_lossy(self);
        s["HTTP/1.1 ".len()..][..3].parse().unwrap()
    }
    fn headers_and_body(&self) -> String {
        String::from_utf8_lossy(self).into_owned()
    }
    fn body_bytes(&self) -> Vec<u8> {
        let p = self.windows(4).position(|w| w == b"\r\n\r\n").unwrap();
        self[p + 4..].to_vec()
    }
    fn header_value(&self, name: &str) -> Option<String> {
        let end = self.windows(4).position(|w| w == b"\r\n\r\n")?;
        let head = std::str::from_utf8(&self[..end]).ok()?;
        head.lines().skip(1).find_map(|line| {
            let (key, value) = line.split_once(':')?;
            key.eq_ignore_ascii_case(name)
                .then(|| value.trim().to_string())
        })
    }
}
impl ResponseView for HttpResponse {
    fn status(&self) -> u16 {
        self.status().as_u16()
    }
    fn headers_and_body(&self) -> String {
        let mut s = format!("HTTP/1.1 {}\r\n", self.status());
        for (name, value) in self.headers() {
            let canonical = name
                .as_str()
                .split('-')
                .map(|part| {
                    let mut chars = part.chars();
                    chars
                        .next()
                        .map(|c| c.to_ascii_uppercase().to_string() + chars.as_str())
                        .unwrap_or_default()
                })
                .collect::<Vec<_>>()
                .join("-");
            s.push_str(&canonical);
            s.push_str(": ");
            s.push_str(value.to_str().unwrap_or(""));
            s.push_str("\r\n");
        }
        s.push_str("\r\n");
        s.push_str(&String::from_utf8_lossy(&self.body_bytes()));
        s
    }
    fn body_bytes(&self) -> Vec<u8> {
        use http_body_util::BodyExt;
        tokio::runtime::Builder::new_current_thread()
            .build()
            .unwrap()
            .block_on(self.body().clone().collect())
            .unwrap()
            .to_bytes()
            .to_vec()
    }
    fn header_value(&self, name: &str) -> Option<String> {
        self.headers()
            .get(name)
            .and_then(|value| value.to_str().ok())
            .map(str::to_owned)
    }
}
fn status_of<R: ResponseView>(response: &R) -> u16 {
    response.status()
}
fn text_of<R: ResponseView>(response: &R) -> String {
    response.headers_and_body()
}
fn body_of<R: ResponseView>(response: &R) -> Vec<u8> {
    response.body_bytes()
}

fn dispatch_stub(method: &str, path: &str, body: &[u8]) -> HttpResponse {
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    dispatch(
        &StubBackend,
        &DaemonConfig::all_enabled(),
        &gate,
        &terminal_gate,
        &head(method, path),
        body,
        true,
    )
    .into_response()
}

/// Scriptable backend: every method replays one result, records bodies and
/// the harness image, and records pumped sessions. Proves body passthrough
/// and per-subsystem error mapping without knowing sibling wire types.
struct ScriptBackend {
    result: Result<Vec<u8>, BackendError>,
    session: u64,
    seen_bodies: Arc<Mutex<Vec<Vec<u8>>>>,
    seen_image: Arc<Mutex<String>>,
    pumped: Arc<AtomicU64>,
    pumped_message: Arc<Mutex<Option<String>>>,
    blocked_calls: Option<Arc<(AtomicU64, AtomicBool)>>,
    hold_pump: bool,
    fail_pump: bool,
}

struct ReleaseBlockedCalls(Arc<(AtomicU64, AtomicBool)>);
impl Drop for ReleaseBlockedCalls {
    fn drop(&mut self) {
        self.0 .1.store(true, Ordering::SeqCst);
    }
}

struct StopSmokeServer(Arc<Server<ScriptBackend>>);
impl Drop for StopSmokeServer {
    fn drop(&mut self) {
        self.0.shutdown();
    }
}

impl ScriptBackend {
    fn err(error: BackendError) -> ScriptBackend {
        ScriptBackend {
            result: Err(error),
            session: 7,
            seen_bodies: Arc::new(Mutex::new(Vec::new())),
            seen_image: Arc::new(Mutex::new(String::new())),
            pumped: Arc::new(AtomicU64::new(u64::MAX)),
            pumped_message: Arc::new(Mutex::new(None)),
            blocked_calls: None,
            hold_pump: false,
            fail_pump: false,
        }
    }

    fn ok(json: &[u8]) -> ScriptBackend {
        ScriptBackend {
            result: Ok(json.to_vec()),
            session: 7,
            seen_bodies: Arc::new(Mutex::new(Vec::new())),
            seen_image: Arc::new(Mutex::new(String::new())),
            pumped: Arc::new(AtomicU64::new(u64::MAX)),
            pumped_message: Arc::new(Mutex::new(None)),
            blocked_calls: None,
            hold_pump: false,
            fail_pump: false,
        }
    }

    fn replay(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.seen_bodies.lock().unwrap().push(body.to_vec());
        if let Some(state) = &self.blocked_calls {
            let (entered, release) = state.as_ref();
            entered.fetch_add(1, Ordering::SeqCst);
            while !release.load(Ordering::SeqCst) {
                std::thread::sleep(std::time::Duration::from_millis(2));
            }
        }
        self.result.clone()
    }
}

impl ExecBackend for ScriptBackend {
    // FJ-874: the setup route reaches the scripted backend so the typed
    // claim codes pin at the wire. Every other application route keeps
    // the trait default (501) exactly as before.
    fn application_setup(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn profile(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn create(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn observe_os(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn connection(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn lifecycle(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn access_keys(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn account(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn project_access(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn prepare(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn prepare_candidate(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn prepare_context(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn inspect_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn stop_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn hold_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn factory_launch(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn factory_inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn factory_verification_checks(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn factory_stop(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn factory_takeover(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn factory_output(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn factory_task_events(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn factory_harness(&self, body: &[u8], image: &str) -> Result<Vec<u8>, BackendError> {
        *self.seen_image.lock().unwrap() = image.to_string();
        self.replay(body)
    }
    fn factory_export(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn factory_candidate_inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn identity_launch(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn identity_action(&self, action: &str, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        assert!(
            ["validate", "stop", "start", "finish"].contains(&action),
            "unknown identity action reached the backend: {action:?}"
        );
        self.replay(body)
    }
    fn tailnet(&self, action: &str, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        assert!(
            TAILNET_ACTIONS.contains(&action),
            "unknown tailnet action reached the backend: {action:?}"
        );
        self.replay(body)
    }
    fn terminal_accept(&self, key: &str) -> Result<TerminalSession, BackendError> {
        assert!(!key.is_empty(), "empty websocket key reached the backend");
        match self.result.clone() {
            Ok(_) => Ok(TerminalSession { id: self.session }),
            Err(e) => Err(e),
        }
    }
    fn pump_terminal(
        &self,
        mut stream: tungstenite::protocol::WebSocket<UnixStream>,
        session: TerminalSession,
        shutdown: Arc<std::sync::atomic::AtomicBool>,
    ) -> Result<(), BackendError> {
        self.pumped.store(session.id, Ordering::SeqCst);
        if self.fail_pump {
            return Err(BackendError::Internal);
        }
        if self.hold_pump {
            while !shutdown.load(Ordering::SeqCst) {
                std::thread::sleep(std::time::Duration::from_millis(5));
            }
            return Ok(());
        }
        loop {
            match stream.read() {
                Ok(tungstenite::Message::Ping(_)) => {
                    let _ = stream.flush();
                }
                Ok(tungstenite::Message::Text(text)) => {
                    *self.pumped_message.lock().unwrap() = Some(text.to_string());
                    break;
                }
                _ => break,
            }
        }
        Ok(())
    }
}

// -- admission: native validation --

#[test]
fn native_clean_paths_reject_query_and_escapes() {
    assert_eq!(NATIVE_CLEAN_PATHS.len(), 33);
    assert!(NATIVE_CLEAN_PATHS.contains(&"/application"));
    assert!(NATIVE_CLEAN_PATHS.contains(&"/terminal-support"));
    for path in NATIVE_CLEAN_PATHS {
        let mut q = head("POST", path);
        q.has_query = true;
        let reject = validate_native_request(&q).unwrap_err();
        assert_eq!(
            (reject.status, reject.message),
            (400, "invalid native operation path")
        );

        let mut e = head("POST", path);
        e.escaped = true;
        let reject = validate_native_request(&e).unwrap_err();
        assert_eq!(
            (reject.status, reject.message),
            (400, "invalid native operation path")
        );

        validate_native_request(&head("POST", path)).unwrap();
    }
}

#[test]
fn native_method_check_runs_after_clean_path_check() {
    // GET on a clean path with a query reports the path (400), not POST (405).
    let mut h = head("GET", "/create");
    h.has_query = true;
    let reject = validate_native_request(&h).unwrap_err();
    assert_eq!(reject.status, 400);

    // GET on a clean path reports POST required.
    let reject = validate_native_request(&head("GET", "/create")).unwrap_err();
    assert_eq!((reject.status, reject.message), (405, "POST required"));

    // Non-clean paths skip the query check (Go's default-true branch) but
    // still require POST.
    validate_native_request(&{
        let mut h = head("POST", "/inspect");
        h.has_query = true;
        h
    })
    .unwrap();
    assert_eq!(
        validate_native_request(&head("GET", "/inspect"))
            .unwrap_err()
            .status,
        405
    );
}

#[test]
fn mutation_gate_covers_exactly_the_go_paths() {
    assert_eq!(ADMITTED_MUTATION_PATHS.len(), 11);
    assert!(is_admitted_mutation_path("/application"));
    assert!(!is_admitted_mutation_path("/terminal-support"));
    for path in ADMITTED_MUTATION_PATHS {
        assert!(is_admitted_mutation_path(path), "{path}");
    }
    for path in [
        "/create",
        "/profile",
        "/inspect",
        "/os",
        "/connection",
        "/project-access",
        "/prepare-inspect",
        "/factory-launch",
        "/factory-verification-checks",
        "/factory-application-build",
        "/factory-harness",
    ] {
        assert!(!is_admitted_mutation_path(path), "{path}");
    }
}

#[test]
fn body_limits_match_go() {
    assert_eq!(BODY_LIMIT_DEFAULT, 65536);
    assert_eq!(BODY_LIMIT_LARGE, 1 << 20);
    assert_eq!(BODY_LIMIT_IDENTITY, 384 << 10);
    assert_eq!(body_limit_for("/prepare"), BODY_LIMIT_LARGE);
    assert_eq!(body_limit_for("/prepare-candidate"), BODY_LIMIT_LARGE);
    assert_eq!(body_limit_for("/factory-launch"), BODY_LIMIT_LARGE);
    assert_eq!(body_limit_for("/identity/finish"), BODY_LIMIT_IDENTITY);
    assert_eq!(body_limit_for("/tailnet/host"), BODY_LIMIT_DEFAULT);
    assert_eq!(body_limit_for("/create"), BODY_LIMIT_DEFAULT);
    assert_eq!(body_limit_for("/factory-harness"), BODY_LIMIT_DEFAULT);
    assert_eq!(body_limit_for("/project-access"), BODY_LIMIT_DEFAULT);
}

#[test]
fn terminal_constants_match_go() {
    assert_eq!(TERMINAL_STREAM_CAP, 128);
    assert_eq!(TERMINAL_REQUEST_LIMIT, 4096);
    assert_eq!(TERMINAL_FRAME_LIMIT, 131072);
    assert_eq!(IDENTITY_ACTIONS.len(), 5);
    assert_eq!(TAILNET_ACTIONS.len(), 8);
}

// -- admission: subsystem validators --

#[test]
fn identity_validator_matches_go() {
    assert!(valid_identity_request(&head("POST", "/identity/launch")));
    assert!(!valid_identity_request(&head("GET", "/identity/launch")));
    let mut q = head("POST", "/identity/launch");
    q.has_query = true;
    assert!(!valid_identity_request(&q));
    let mut e = head("POST", "/identity/launch");
    e.escaped = true;
    assert!(!valid_identity_request(&e));
    let mut o = head("POST", "/identity/launch");
    o.origin_present = true;
    assert!(!valid_identity_request(&o));
}

#[test]
fn tailnet_validator_matches_go() {
    for action in TAILNET_ACTIONS {
        assert_eq!(
            validate_tailnet_request(&head("POST", &format!("/tailnet/{action}"))).unwrap(),
            *action
        );
    }
    assert_eq!(
        validate_tailnet_request(&head("POST", "/tailnet/bogus")).unwrap_err(),
        404
    );
    assert_eq!(
        validate_tailnet_request(&head("GET", "/tailnet/host")).unwrap_err(),
        400
    );
    let mut q = head("POST", "/tailnet/host");
    q.has_query = true;
    assert_eq!(validate_tailnet_request(&q).unwrap_err(), 400);
}

#[test]
fn terminal_validator_matches_go() {
    let mut h = head("GET", "/terminal");
    h.upgrade_websocket = true;
    h.ws_key = Some("k".to_string());
    assert!(valid_terminal_request(&h));
    assert!(!valid_terminal_request(&head("POST", "/terminal")));
    let mut q = head("GET", "/terminal");
    q.has_query = true;
    assert!(!valid_terminal_request(&q));
    let mut e = head("GET", "/terminal");
    e.escaped = true;
    assert!(!valid_terminal_request(&e));
    let mut o = head("GET", "/terminal");
    o.origin_present = true;
    assert!(!valid_terminal_request(&o));
}

// -- admission: gates --

#[test]
fn mutation_gate_serializes() {
    let gate = AdmissionGate::new();
    let held = gate.try_acquire().expect("free gate");
    assert!(gate.try_acquire().is_none(), "gate must serialize");
    drop(held);
    assert!(gate.try_acquire().is_some(), "gate must release on drop");
}

#[test]
fn mutation_gate_blocking_acquire_hands_off() {
    let gate = AdmissionGate::new();
    let held = gate.acquire();
    let (tx, rx) = std::sync::mpsc::channel();
    std::thread::scope(|s| {
        s.spawn(|| {
            tx.send("waiting").unwrap();
            let _guard = gate.acquire();
            tx.send("acquired").unwrap();
        });
        assert_eq!(rx.recv().unwrap(), "waiting");
        drop(held);
        assert_eq!(rx.recv().unwrap(), "acquired");
    });
}

#[test]
fn terminal_gate_caps_and_releases() {
    let gate = TerminalGate::new();
    assert_eq!(TERMINAL_STREAM_CAP, 128);
    let mut slots = Vec::new();
    for _ in 0..TERMINAL_STREAM_CAP {
        slots.push(gate.try_register().expect("slot under cap"));
    }
    assert_eq!(gate.live(), TERMINAL_STREAM_CAP);
    assert!(gate.try_register().is_none(), "cap must hold");
    slots.pop();
    assert_eq!(gate.live(), TERMINAL_STREAM_CAP - 1);
    assert!(gate.try_register().is_some(), "drop must release");
}

// -- routes: table completeness --

#[test]
fn route_table_has_every_go_route() {
    assert_eq!(ROUTE_TABLE.len(), 50);
    let count = |prefix: &str| {
        ROUTE_TABLE
            .iter()
            .filter(|(_, p)| p.starts_with(prefix))
            .count()
    };
    assert_eq!(count("/identity/"), 5);
    assert_eq!(count("/tailnet/"), 8);
    assert_eq!(
        ROUTE_TABLE
            .iter()
            .filter(|(_, p)| *p == "/terminal")
            .count(),
        1
    );
    for path in [
        "/profile",
        "/application",
        "/application/setup",
        "/application/snapshots",
        "/terminal-support",
        "/create",
        "/inspect",
        "/os",
        "/connection",
        "/lifecycle",
        "/access-keys",
        "/account",
        "/project-access",
        "/member-access",
        "/prepare",
        "/prepare-candidate",
        "/prepare-context",
        "/prepare-inspect",
        "/prepare-stop",
        "/prepare-hold",
        "/factory-launch",
        "/factory-verification-checks",
        "/factory-application-build",
        "/factory-inspect",
        "/factory-stop",
        "/factory-takeover",
        "/factory-output",
        "/factory-task-events",
        "/factory-native-task",
        "/factory-task-checkpoint",
        "/factory-final-candidate",
        "/factory-task-checkpoint-receipt",
        "/factory-final-candidate-receipt",
        "/factory-harness",
        "/factory-export",
        "/factory-candidate-inspect",
    ] {
        assert!(
            ROUTE_TABLE.contains(&("POST", path)),
            "missing route POST {path}"
        );
    }
}

#[test]
fn every_post_route_dispatches_through_the_stub() {
    for (method, path) in ROUTE_TABLE {
        if *method != "POST" {
            continue;
        }
        // Lifecycle endpoints validate the verified tuple before the
        // backend: malformed calls stop at 400 without effects.
        if *path == "/tailnet/prepare" || *path == "/tailnet/lifecycle" {
            let response = dispatch_stub(method, path, b"{}");
            assert_eq!(status_of(&response), 400, "route {method} {path}");
            continue;
        }
        let response = dispatch_stub(method, path, b"{}");
        assert_eq!(status_of(&response), 501, "route {method} {path}");
        assert!(
            text_of(&response).ends_with("\r\n\r\nnot implemented\n"),
            "route {method} {path}"
        );
    }
}

#[test]
fn bodies_and_harness_image_pass_through_verbatim() {
    let backend = ScriptBackend::ok(b"{\"ok\":true}");
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let mut config = DaemonConfig::all_enabled();
    config.image = "sha256:abc".to_string();
    let bodies: Arc<Mutex<Vec<Vec<u8>>>> = Arc::clone(&backend.seen_bodies);
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("POST", "/factory-harness"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 200);
    assert_eq!(bodies.lock().unwrap().as_slice(), &[b"{}".to_vec()]);
    assert_eq!(backend.seen_image.lock().unwrap().as_str(), "sha256:abc");
}

// -- routes: native mapping --

#[test]
fn native_error_mapping_matches_go() {
    let cases: &[(BackendError, u16, &str)] = &[
        (BackendError::NotFound, 404, "404 page not found\n"),
        (
            BackendError::Unavailable,
            503,
            "factory runtime unavailable\n",
        ),
        (
            BackendError::OutputStale,
            409,
            "factory output incarnation changed\n",
        ),
        (
            BackendError::ExportStale,
            409,
            "factory run incarnation changed\n",
        ),
        (
            BackendError::ExportCandidate,
            422,
            "export candidate is not recorded\n",
        ),
        (
            BackendError::ExportBounds,
            413,
            "candidate export exceeds bounds\n",
        ),
        // The genuine typed explicit-bounds refusal for /prepare-context:
        // oversize native selections surface 413 (never 500), while every
        // other context failure keeps the generic 500 and fails closed.
        (
            BackendError::ContextBounds,
            413,
            "preparation context exceeds bounds\n",
        ),
        (
            BackendError::Invalid,
            500,
            "native operation failed; inspect operator journal\n",
        ),
        (
            BackendError::Denied,
            500,
            "native operation failed; inspect operator journal\n",
        ),
        (
            BackendError::Internal,
            500,
            "native operation failed; inspect operator journal\n",
        ),
        (BackendError::Unimplemented, 501, "not implemented\n"),
    ];
    for (error, status, body) in cases {
        let backend = ScriptBackend::err(*error);
        let gate = AdmissionGate::new();
        let terminal_gate = TerminalGate::new();
        let response = dispatch(
            &backend,
            &DaemonConfig::all_enabled(),
            &gate,
            &terminal_gate,
            &head("POST", "/lifecycle"),
            b"{}",
            true,
        )
        .into_response();
        assert_eq!(status_of(&response), *status, "error {error:?}");
        assert_eq!(body_of(&response), body.as_bytes(), "error {error:?}");
        assert_eq!(
            response.header_value("content-type").as_deref(),
            Some("text/plain; charset=utf-8")
        );
        assert_eq!(
            response.header_value("x-content-type-options").as_deref(),
            Some("nosniff")
        );
    }
}

#[test]
fn native_unknown_path_is_404_and_terminal_prefix_is_405() {
    let response = dispatch_stub("POST", "/nope", b"{}");
    assert_eq!(status_of(&response), 404);
    // Exact-match routing: only "/terminal" upgrades; siblings want POST.
    let response = dispatch_stub("GET", "/terminal/x", b"");
    assert_eq!(status_of(&response), 405);
    let response = dispatch_stub("GET", "/create", b"");
    assert_eq!(status_of(&response), 405);
    assert!(text_of(&response).ends_with("\r\n\r\nPOST required\n"));
}

#[test]
fn native_success_envelope_matches_go_encoder() {
    let backend = ScriptBackend::ok(b"{\"a\":1}");
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let response = dispatch(
        &backend,
        &DaemonConfig::all_enabled(),
        &gate,
        &terminal_gate,
        &head("POST", "/inspect"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 200);
    assert_eq!(
        response.header_value("content-type").as_deref(),
        Some("application/json")
    );
    assert_eq!(body_of(&response), b"{\"a\":1}\n");
}

#[test]
fn over_limit_bodies_fail_like_go_maxbytesreader() {
    let backend = ScriptBackend::ok(b"{}");
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let config = DaemonConfig::all_enabled();
    for (path, status) in [
        ("/create", 500),
        ("/identity/launch", 409),
        ("/tailnet/host", 400),
    ] {
        let response = dispatch(
            &backend,
            &config,
            &gate,
            &terminal_gate,
            &head("POST", path),
            b"{}",
            false,
        )
        .into_response();
        assert_eq!(status_of(&response), status, "path {path}");
    }
    // Truncated bodies never reach the backend.
    assert!(backend.seen_bodies.lock().unwrap().is_empty());
}

#[test]
fn create_and_mutations_reject_busy_gate_reads_do_not() {
    let backend = ScriptBackend::ok(b"{}");
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let config = DaemonConfig::all_enabled();
    let held = gate.try_acquire().expect("gate free");
    // A read completes while the gate is held elsewhere.
    let (tx, rx) = std::sync::mpsc::channel();
    std::thread::scope(|s| {
        s.spawn(|| {
            let response = dispatch(
                &backend,
                &config,
                &gate,
                &terminal_gate,
                &head("POST", "/inspect"),
                b"{}",
                true,
            )
            .into_response();
            tx.send(status_of(&response)).unwrap();
        });
        assert_eq!(
            rx.recv_timeout(std::time::Duration::from_secs(5)).unwrap(),
            200
        );
    });
    // A mutation refuses immediately while another operation owns the gate.
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("POST", "/lifecycle"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 503);
    assert_eq!(backend.seen_bodies.lock().unwrap().len(), 1);
    // Project privilege is a read-only observation and bypasses the writer gate.
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("POST", "/project-access"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 200);
    assert_eq!(backend.seen_bodies.lock().unwrap().len(), 2);
    drop(held);
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("POST", "/lifecycle"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 200);
    assert_eq!(backend.seen_bodies.lock().unwrap().len(), 3);
}

// -- routes: identity mapping --

#[test]
fn identity_errors_collapse_to_409_like_go() {
    for error in [
        BackendError::NotFound,
        BackendError::Unavailable,
        BackendError::Invalid,
        BackendError::Denied,
        BackendError::Internal,
    ] {
        let backend = ScriptBackend::err(error);
        let gate = AdmissionGate::new();
        let terminal_gate = TerminalGate::new();
        let response = dispatch(
            &backend,
            &DaemonConfig::all_enabled(),
            &gate,
            &terminal_gate,
            &head("POST", "/identity/validate"),
            b"{}",
            true,
        )
        .into_response();
        assert_eq!(status_of(&response), 409, "error {error:?}");
        assert_eq!(body_of(&response), b"identity operation unconfirmed\n");
    }
}

#[test]
fn identity_rejects_bad_shape_unknown_actions_and_missing_runtime() {
    // Unknown actions deny with 409 (Go forwards to Terminal.Identity,
    // which denies); nothing reaches the backend.
    let backend = ScriptBackend::ok(b"{}");
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let response = dispatch(
        &backend,
        &DaemonConfig::all_enabled(),
        &gate,
        &terminal_gate,
        &head("POST", "/identity/bogus"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 409);
    assert!(backend.seen_bodies.lock().unwrap().is_empty());

    // Bad shapes and missing runtime report 400.
    let response = dispatch(
        &backend,
        &DaemonConfig::all_enabled(),
        &gate,
        &terminal_gate,
        &head("GET", "/identity/launch"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 400);
    assert_eq!(body_of(&response), b"invalid identity operation\n");

    let mut config = DaemonConfig::all_enabled();
    config.identity_available = false;
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("POST", "/identity/launch"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 400);
}

// -- routes: tailnet mapping --

#[test]
fn tailnet_error_mapping_matches_go() {
    let cases: &[(BackendError, u16)] = &[
        (BackendError::Invalid, 400),
        (BackendError::OutputStale, 409),
        (BackendError::ExportStale, 409),
        (BackendError::ExportCandidate, 422),
        (BackendError::Unavailable, 503),
        (BackendError::NotFound, 502),
        (BackendError::Denied, 502),
        (BackendError::Internal, 502),
        (BackendError::Unimplemented, 501),
    ];
    for (error, status) in cases {
        let backend = ScriptBackend::err(*error);
        let gate = AdmissionGate::new();
        let terminal_gate = TerminalGate::new();
        let response = dispatch(
            &backend,
            &DaemonConfig::all_enabled(),
            &gate,
            &terminal_gate,
            &head("POST", "/tailnet/host"),
            b"{}",
            true,
        )
        .into_response();
        assert_eq!(status_of(&response), *status, "error {error:?}");
        // Every tailnet response carries no-store, errors included.
        assert_eq!(
            response.header_value("cache-control").as_deref(),
            Some("no-store")
        );
    }
    let failure = "Tailnet operation unavailable or unconfirmed; observe before retrying\n";
    let backend = ScriptBackend::err(BackendError::Internal);
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let response = dispatch(
        &backend,
        &DaemonConfig::all_enabled(),
        &gate,
        &terminal_gate,
        &head("POST", "/tailnet/host"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(body_of(&response), failure.as_bytes());
}

#[test]
fn tailnet_success_has_no_trailing_newline() {
    let backend = ScriptBackend::ok(b"{\"saved\":true}");
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let response = dispatch(
        &backend,
        &DaemonConfig::all_enabled(),
        &gate,
        &terminal_gate,
        &head("POST", "/tailnet/policy"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 200);
    assert_eq!(body_of(&response), b"{\"saved\":true}");
    assert_eq!(
        response.header_value("cache-control").as_deref(),
        Some("no-store")
    );

    // Oversized backend payloads fail closed with 502 like Go.
    let big = ScriptBackend::ok(&vec![b'x'; 65537]);
    let response = dispatch(
        &big,
        &DaemonConfig::all_enabled(),
        &gate,
        &terminal_gate,
        &head("POST", "/tailnet/settings"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 502);

    let response = dispatch_stub("POST", "/tailnet/bogus", b"{}");
    assert_eq!(status_of(&response), 404);
    assert_eq!(
        response.header_value("cache-control").as_deref(),
        Some("no-store")
    );
}

// -- routes: tailnet lifecycle (RCX-2 T01) --

fn lifecycle_call_body(operation: &str, deadline_unix: i64) -> Vec<u8> {
    format!(
        "{{\
        \"controller_id\":\"soda\",\
        \"controller_instance_id\":\"dddddddddddddddddddddddddddddddd\",\
        \"package_id\":\"tailnet\",\
        \"package_version\":\"1.0.0\",\
        \"package_sha256\":\"{}\",\
        \"installation_id\":\"550e8400-e29b-41d4-a716-446655440000\",\
        \"generation\":\"7\",\
        \"operation\":\"{operation}\",\
        \"deadline_unix\":{deadline_unix}}}",
        "b".repeat(64),
    )
    .into_bytes()
}

fn lifecycle_future_deadline() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|elapsed| elapsed.as_secs() as i64)
        .unwrap_or(0)
        + 300
}

fn lifecycle_pinned_config() -> DaemonConfig {
    let mut config = DaemonConfig::all_enabled();
    config.tailnet_package_version = "1.0.0".to_string();
    config.tailnet_package_sha256 = "b".repeat(64);
    config
}

#[test]
fn tailnet_lifecycle_rejects_malformed_before_backend() {
    let backend = ScriptBackend::ok(b"{\"ok\":true}");
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let config = lifecycle_pinned_config();
    let seen = || backend.seen_bodies.lock().unwrap().len();
    // Empty JSON, unknown fields and truncated bodies never reach effects.
    let mut truncated = lifecycle_call_body("observe", lifecycle_future_deadline());
    truncated.pop();
    truncated.push(b'x');
    for body in [
        b"{}".to_vec(),
        b"{\"controller_id\":\"soda\"}".to_vec(),
        truncated,
    ] {
        let response = dispatch(
            &backend,
            &config,
            &gate,
            &terminal_gate,
            &head("POST", "/tailnet/lifecycle"),
            &body,
            true,
        )
        .into_response();
        assert_eq!(status_of(&response), 400);
    }
    // Truncated transport and non-POST/query targets fail the same way.
    let body = lifecycle_call_body("observe", lifecycle_future_deadline());
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("POST", "/tailnet/lifecycle"),
        &body,
        false,
    )
    .into_response();
    assert_eq!(status_of(&response), 400);
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("GET", "/tailnet/lifecycle"),
        &body,
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 400);
    let mut query = head("POST", "/tailnet/lifecycle");
    query.has_query = true;
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &query,
        &body,
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 400);
    assert_eq!(seen(), 0);
}

#[test]
fn tailnet_lifecycle_scope_pin_and_endpoint_mismatch_stop_before_backend() {
    let backend = ScriptBackend::ok(b"{\"ok\":true}");
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let config = lifecycle_pinned_config();
    let seen = || backend.seen_bodies.lock().unwrap().len();
    let deadline = lifecycle_future_deadline();
    // Wrong controller or package is an unbound authority: 409, no effects.
    let mut wrong_controller = lifecycle_call_body("observe", deadline);
    let from = b"\"controller_id\":\"soda\"".to_vec();
    let to = b"\"controller_id\":\"other\"".to_vec();
    let position = wrong_controller
        .windows(from.len())
        .position(|window| window == from.as_slice())
        .unwrap();
    wrong_controller.splice(position..position + from.len(), to.iter().cloned());
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("POST", "/tailnet/lifecycle"),
        &wrong_controller,
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 409);
    // Prepare carries activate only: an observe tuple is endpoint-mismatched.
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("POST", "/tailnet/prepare"),
        &lifecycle_call_body("observe", deadline),
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 409);
    // A drifted pin is a changed identity; a missing pin is unavailable.
    // Both stop before the backend.
    let mut drifted = lifecycle_pinned_config();
    drifted.tailnet_package_sha256 = "c".repeat(64);
    let response = dispatch(
        &backend,
        &drifted,
        &gate,
        &terminal_gate,
        &head("POST", "/tailnet/lifecycle"),
        &lifecycle_call_body("observe", deadline),
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 409);
    let unpinned = DaemonConfig::all_enabled();
    let response = dispatch(
        &backend,
        &unpinned,
        &gate,
        &terminal_gate,
        &head("POST", "/tailnet/lifecycle"),
        &lifecycle_call_body("observe", deadline),
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 503);
    assert_eq!(seen(), 0);
}

#[test]
fn tailnet_lifecycle_verified_calls_reach_backend_once() {
    let observation = b"{\"controller_id\":\"soda\",\"controller_instance_id\":\"dddddddddddddddddddddddddddddddd\",\"package_id\":\"tailnet\",\"package_version\":\"1.0.0\",\"package_sha256\":\"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\",\"installation_id\":\"550e8400-e29b-41d4-a716-446655440000\",\"generation\":\"7\",\"operation\":\"observe\",\"deadline_unix\":1792100010,\"state\":\"stopped\",\"effect_generation\":null,\"admission_ready\":false,\"detail\":\"test\"}";
    let backend = ScriptBackend::ok(observation);
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let config = lifecycle_pinned_config();
    let seen = || backend.seen_bodies.lock().unwrap().len();
    let deadline = lifecycle_future_deadline();
    for (path, operation) in [
        ("/tailnet/lifecycle", "observe"),
        ("/tailnet/lifecycle", "activate"),
        ("/tailnet/lifecycle", "deactivate"),
        ("/tailnet/prepare", "activate"),
    ] {
        let body = lifecycle_call_body(operation, deadline);
        let response = dispatch(
            &backend,
            &config,
            &gate,
            &terminal_gate,
            &head("POST", path),
            &body,
            true,
        )
        .into_response();
        assert_eq!(status_of(&response), 200, "{path} {operation}");
        assert_eq!(body_of(&response), observation.to_vec());
    }
    assert_eq!(seen(), 4);
    // Backend failures map through the shared tailnet statuses.
    let backend = ScriptBackend::err(BackendError::Unavailable);
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("POST", "/tailnet/lifecycle"),
        &lifecycle_call_body("observe", deadline),
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 503);
}

// -- routes: terminal --

fn terminal_head() -> RequestHead {
    let mut h = head("GET", "/terminal");
    h.upgrade_websocket = true;
    h.ws_key = Some("dGhlIHNhbXBsZSBub25jZQ==".to_string());
    let request = hyper::Request::builder()
        .method("GET")
        .uri("http://local/terminal")
        .version(hyper::Version::HTTP_11)
        .header("upgrade", "websocket")
        .header("connection", "Upgrade")
        .header("sec-websocket-version", "13")
        .header("sec-websocket-key", "dGhlIHNhbXBsZSBub25jZQ==")
        .body(())
        .unwrap();
    h.websocket_request = Some(request);
    h
}

#[test]
fn terminal_upgrade_holds_slot_and_renders_101() {
    let backend = ScriptBackend::ok(b"{}");
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let outcome = dispatch(
        &backend,
        &DaemonConfig::all_enabled(),
        &gate,
        &terminal_gate,
        &terminal_head(),
        b"",
        true,
    );
    match outcome {
        RouteOutcome::TerminalUpgrade {
            response,
            session,
            slot,
        } => {
            assert_eq!(status_of(&response), 101);
            assert_eq!(
                response.header_value("connection").as_deref(),
                Some("Upgrade")
            );
            assert_eq!(
                response.header_value("upgrade").as_deref(),
                Some("websocket")
            );
            assert_eq!(
                response.header_value("sec-websocket-accept").as_deref(),
                Some("s3pPLMBiTxaQ9kYGzzhZRbK+xOo=")
            );
            assert_eq!(session.id, 7);
            assert_eq!(terminal_gate.live(), 1, "slot held for the session");
            drop(slot);
            assert_eq!(terminal_gate.live(), 0, "slot released with outcome");
        }
        RouteOutcome::Respond(_) => panic!("expected a 101 upgrade"),
    }
}

#[test]
fn terminal_rejections_match_go() {
    let backend = ScriptBackend::ok(b"{}");
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let config = DaemonConfig::all_enabled();

    // Missing upgrade headers: 400 like a failed websocket.Accept.
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("GET", "/terminal"),
        b"",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 400);
    assert_eq!(body_of(&response), b"invalid private terminal request\n");

    // POST on the exact attach path fails the private-terminal shape
    // check (Go routes by exact path first, then validates): 400, not 405.
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("POST", "/terminal"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 400);
    assert_eq!(body_of(&response), b"invalid private terminal request\n");

    // No runtime: 503.
    let mut down = DaemonConfig::all_enabled();
    down.terminal_available = false;
    let response = dispatch(
        &backend,
        &down,
        &gate,
        &terminal_gate,
        &terminal_head(),
        b"",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 503);
    assert_eq!(body_of(&response), b"terminal unavailable\n");

    // Full house: 503.
    let mut slots = Vec::new();
    for _ in 0..TERMINAL_STREAM_CAP {
        slots.push(terminal_gate.try_register().unwrap());
    }
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &terminal_head(),
        b"",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 503);

    // Backend-side failure before upgrade: 503 (closest Go pre-upgrade signal).
    drop(slots);
    let failing = ScriptBackend::err(BackendError::Unavailable);
    let response = dispatch(
        &failing,
        &config,
        &gate,
        &terminal_gate,
        &terminal_head(),
        b"",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 503);

    // Stub: honest 501.
    let response = dispatch(
        &StubBackend,
        &config,
        &gate,
        &terminal_gate,
        &terminal_head(),
        b"",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 501);
}

#[test]
fn stub_pump_reports_unimplemented() {
    let (stream, _peer) = UnixStream::pair().unwrap();
    assert_eq!(
        StubBackend.pump_terminal(
            tungstenite::protocol::WebSocket::from_raw_socket(
                stream,
                tungstenite::protocol::Role::Server,
                None
            ),
            TerminalSession { id: 0 },
            Arc::new(std::sync::atomic::AtomicBool::new(false)),
        ),
        Err(BackendError::Unimplemented)
    );
}

// -- server: end to end over a real socket --

fn read_all(stream: &mut UnixStream) -> Vec<u8> {
    let mut out = Vec::new();
    stream.read_to_end(&mut out).unwrap();
    out
}

fn read_http_head(stream: &mut UnixStream) -> Vec<u8> {
    let mut out = Vec::new();
    let mut byte = [0u8; 1];
    while !out.ends_with(b"\r\n\r\n") {
        stream.read_exact(&mut byte).unwrap();
        out.push(byte[0]);
    }
    out
}

fn socket_path(name: &str) -> std::path::PathBuf {
    let repo = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .find(|path| path.join("Cargo.toml").is_file() && path.join("lib").is_dir())
        .expect("test crate is inside the workspace");
    let dir = repo.join(".artifacts/l08-l09");
    std::fs::create_dir_all(&dir).unwrap();
    dir.join(format!("{name}-{}.sock", std::process::id()))
}

#[test]
fn server_serves_stub_routes_and_parser_rejections() {
    let path = socket_path("gmux-smoke");
    let _ = std::fs::remove_file(&path);
    let listener = UnixListener::bind(&path).unwrap();
    let server = Arc::new(Server::new(
        Arc::new(StubBackend),
        DaemonConfig::all_enabled(),
    ));
    let serving = Arc::clone(&server);
    let handle = std::thread::spawn(move || serving.serve(&listener));

    // Stub POST route -> 501.
    let mut client = UnixStream::connect(&path).unwrap();
    client
        .write_all(b"POST /profile HTTP/1.1\r\nHost: x\r\nContent-Length: 2\r\n\r\n{}")
        .unwrap();
    let response = read_all(&mut client);
    assert_eq!(status_of(&response), 501);
    assert_eq!(
        response.header_value("connection").as_deref(),
        Some("close")
    );

    // The native verification endpoint uses the same real socket transport,
    // clean-path admission and POST-only dispatch as the supervised run.
    for (method, target, expected) in [
        ("POST", "/factory-verification-checks", 501),
        ("POST", "/factory-verification-checks?run=other", 400),
        ("GET", "/factory-verification-checks", 405),
    ] {
        let mut client = UnixStream::connect(&path).unwrap();
        client
            .write_all(
                format!("{method} {target} HTTP/1.1\r\nHost: x\r\nContent-Length: 2\r\n\r\n{{}}")
                    .as_bytes(),
            )
            .unwrap();
        assert_eq!(
            status_of(&read_all(&mut client)),
            expected,
            "{method} {target}"
        );
    }

    // Tailnet errors still carry no-store over the wire.
    let mut client = UnixStream::connect(&path).unwrap();
    client
        .write_all(b"POST /tailnet/settings HTTP/1.1\r\nHost: x\r\nContent-Length: 2\r\n\r\n{}")
        .unwrap();
    let response = read_all(&mut client);
    assert_eq!(status_of(&response), 501);
    assert_eq!(
        response.header_value("cache-control").as_deref(),
        Some("no-store")
    );

    // Malformed head -> 400.
    let mut client = UnixStream::connect(&path).unwrap();
    client.write_all(b"HELLO\r\n\r\n").unwrap();
    assert_eq!(status_of(&read_all(&mut client)), 400);

    // Oversized head -> 431.
    let mut client = UnixStream::connect(&path).unwrap();
    client
        .write_all(b"POST /create HTTP/1.1\r\nHost: x\r\nX-Pad: ")
        .unwrap();
    client.write_all(&vec![b'a'; 9000]).unwrap();
    client.write_all(b"\r\n\r\n").unwrap();
    // Hyper can reject once the header limit is crossed while excess request
    // bytes remain unread. Its parser-error response is an empty 431, so
    // verify the complete response head and zero body length without treating
    // an unread-input reset as a truncated response body.
    let response = read_http_head(&mut client);
    assert_eq!(status_of(&response), 431);
    assert_eq!(
        response.header_value("content-length").as_deref(),
        Some("0")
    );

    // Hyper decodes legal chunked framing; dispatch then applies the normal
    // route behavior to the empty request body.
    let mut client = UnixStream::connect(&path).unwrap();
    client
        .write_all(
            b"POST /create HTTP/1.1\r\nHost: x\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n",
        )
        .unwrap();
    assert_eq!(status_of(&read_all(&mut client)), 501);

    server.shutdown();
    handle.join().unwrap().unwrap();
    assert_eq!(server.inflight(), 0);
    let _ = std::fs::remove_file(&path);
}

#[test]
fn server_runs_terminal_upgrade_and_pump() {
    let path = socket_path("gmux-term");
    let _ = std::fs::remove_file(&path);
    let listener = UnixListener::bind(&path).unwrap();
    let backend = ScriptBackend::ok(b"{}");
    let pumped = Arc::clone(&backend.pumped);
    let pumped_message = Arc::clone(&backend.pumped_message);
    let server = Arc::new(Server::new(Arc::new(backend), DaemonConfig::all_enabled()));
    let serving = Arc::clone(&server);
    let handle = std::thread::spawn(move || serving.serve(&listener));

    let mut client = UnixStream::connect(&path).unwrap();
    client
        .write_all(
            b"GET /terminal HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n\x89\x84\x01\x02\x03\x04\x71\x6b\x6d\x63\x81\x85\x01\x02\x03\x04\x69\x67\x6f\x68\x6e",
        )
        .unwrap();
    // The pump records the session and returns, closing the stream.
    let response = read_all(&mut client);
    assert_eq!(status_of(&response), 101);
    assert_eq!(
        response.header_value("connection").as_deref(),
        Some("Upgrade"),
        "Connection: close makes the actual Go client discard the upgrade header"
    );
    assert_eq!(
        response.header_value("sec-websocket-accept").as_deref(),
        Some("s3pPLMBiTxaQ9kYGzzhZRbK+xOo=")
    );
    assert_eq!(pumped.load(Ordering::SeqCst), 7);
    assert_eq!(
        pumped_message.lock().unwrap().as_deref(),
        Some("hello"),
        "Hyper read-ahead bytes must reach tungstenite"
    );
    assert!(
        response.windows(6).any(|w| w == b"\x8a\x04ping"),
        "tungstenite should answer Ping from the same protocol owner"
    );

    server.shutdown();
    handle.join().unwrap().unwrap();
    assert_eq!(server.inflight(), 0, "upgrade pump ownership is joined");
    let _ = std::fs::remove_file(&path);
}

#[test]
fn terminal_pump_failure_is_reported_by_server_serve() {
    let path = socket_path("gmux-pump-error");
    let _ = std::fs::remove_file(&path);
    let listener = UnixListener::bind(&path).unwrap();
    let mut backend = ScriptBackend::ok(b"{}");
    backend.fail_pump = true;
    let pumped = Arc::clone(&backend.pumped);
    let server = Arc::new(Server::new(Arc::new(backend), DaemonConfig::all_enabled()));
    let _stop_on_exit = StopSmokeServer(Arc::clone(&server));
    let serving = Arc::clone(&server);
    let handle = std::thread::spawn(move || serving.serve(&listener));

    let mut client = UnixStream::connect(&path).unwrap();
    client.write_all(b"GET /terminal HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n").unwrap();
    assert_eq!(status_of(&read_http_head(&mut client)), 101);
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(2);
    while pumped.load(Ordering::SeqCst) != 7 && std::time::Instant::now() < deadline {
        std::thread::sleep(std::time::Duration::from_millis(5));
    }
    assert_eq!(pumped.load(Ordering::SeqCst), 7);
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(2);
    while !server.is_shutdown() && std::time::Instant::now() < deadline {
        std::thread::sleep(std::time::Duration::from_millis(5));
    }
    assert!(
        server.is_shutdown(),
        "pump failure must initiate server shutdown"
    );
    assert_eq!(
        handle.join().unwrap().unwrap_err(),
        "terminal pump failed: Internal"
    );
    let _ = std::fs::remove_file(&path);
}

#[test]
fn server_keeps_upgrade_inflight_until_pump_shutdown_join() {
    let path = socket_path("gmux-pump-live");
    let _ = std::fs::remove_file(&path);
    let listener = UnixListener::bind(&path).unwrap();
    let mut backend = ScriptBackend::ok(b"{}");
    backend.hold_pump = true;
    let pumped = Arc::clone(&backend.pumped);
    let server = Arc::new(Server::new(Arc::new(backend), DaemonConfig::all_enabled()));
    let _stop_on_exit = StopSmokeServer(Arc::clone(&server));
    let serving = Arc::clone(&server);
    let handle = std::thread::spawn(move || serving.serve(&listener));
    let mut client = UnixStream::connect(&path).unwrap();
    client.write_all(b"GET /terminal HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n").unwrap();
    let response_head = read_http_head(&mut client);
    assert_eq!(status_of(&response_head), 101);
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(2);
    while pumped.load(Ordering::SeqCst) != 7 && std::time::Instant::now() < deadline {
        std::thread::sleep(std::time::Duration::from_millis(5));
    }
    assert_eq!(pumped.load(Ordering::SeqCst), 7);
    assert!(
        server.inflight() > 0,
        "inflight includes the active upgrade pump"
    );
    server.shutdown();
    handle.join().unwrap().unwrap();
    assert_eq!(
        server.inflight(),
        0,
        "inflight drops only after pump cleanup joins"
    );
    let _ = std::fs::remove_file(&path);
}

#[test]
fn server_bounds_backend_work_and_joins_it_during_shutdown() {
    let path = socket_path("gmux-drain");
    let _ = std::fs::remove_file(&path);
    let listener = UnixListener::bind(&path).unwrap();
    let mut backend = ScriptBackend::ok(b"{}");
    let callback_state = Arc::new((AtomicU64::new(0), AtomicBool::new(false)));
    backend.blocked_calls = Some(Arc::clone(&callback_state));
    let server = Arc::new(Server::new(Arc::new(backend), DaemonConfig::all_enabled()));
    let _stop_on_exit = StopSmokeServer(Arc::clone(&server));
    let _release_on_exit = ReleaseBlockedCalls(Arc::clone(&callback_state));
    let serving = Arc::clone(&server);
    let handle = std::thread::spawn(move || serving.serve(&listener));

    let mut admitted = Vec::new();
    for _ in 0..16 {
        let mut client = UnixStream::connect(&path).unwrap();
        client
            .write_all(b"POST /profile HTTP/1.1\r\nHost: x\r\nContent-Length: 2\r\n\r\n{}")
            .unwrap();
        admitted.push(client);
    }
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(3);
    while callback_state.0.load(Ordering::SeqCst) != 16 && std::time::Instant::now() < deadline {
        std::thread::sleep(std::time::Duration::from_millis(5));
    }
    assert_eq!(
        callback_state.0.load(Ordering::SeqCst),
        16,
        "all bounded backend permits are occupied"
    );

    let mut excess = UnixStream::connect(&path).unwrap();
    excess
        .set_read_timeout(Some(std::time::Duration::from_secs(2)))
        .unwrap();
    excess
        .write_all(b"POST /profile HTTP/1.1\r\nHost: x\r\nContent-Length: 2\r\n\r\n{}")
        .unwrap();
    let response = read_all(&mut excess);
    assert_eq!(status_of(&response), 503, "excess backend work is rejected");

    // An admitted partial head remains owned by the server until shutdown
    // cancels its HTTP driver; backend callbacks remain joined separately.
    let mut partial = UnixStream::connect(&path).unwrap();
    partial
        .write_all(b"POST /profile HTTP/1.1\r\nHost: x\r\nContent-Length: 2\r\n")
        .unwrap();
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(2);
    while server.inflight() < 17 && std::time::Instant::now() < deadline {
        std::thread::sleep(std::time::Duration::from_millis(5));
    }
    assert!(
        server.inflight() >= 17,
        "partial request remains tracked alongside callbacks"
    );

    server.shutdown();
    std::thread::sleep(std::time::Duration::from_millis(100));
    assert!(
        !handle.is_finished(),
        "shutdown must join active backend callbacks"
    );
    callback_state.1.store(true, Ordering::SeqCst);
    handle.join().unwrap().unwrap();
    assert_eq!(
        server.inflight(),
        0,
        "HTTP drivers and backend ownership are drained"
    );
    drop(admitted);
    drop(partial);
    let _ = std::fs::remove_file(&path);
}

#[test]
fn systemd_listener_refuses_without_activation() {
    if std::env::var("LISTEN_FDS").as_deref() == Ok("1") {
        eprintln!("skipping: test runs under socket activation");
        return;
    }
    assert!(systemd_listener().is_err());
}

// -- native MSP session/start for the staged Factory MCP relay --
//
// A fake peer stands in for the Muse daemon over a UnixStream pair and speaks
// the exact newline-delimited JSON-RPC framing the MSP transport owns. The
// cases prove: session/start carries no MCP config and succeeds on any
// grant shape, because the staged per-run settings are the single
// authoritative `factory` registration; a start that repeats the staged
// name is refused like the observed daemon -32030. Refusal cases prove only
// strictly validated structured reason codes surface: arbitrary daemon
// message text never becomes a reason.
mod msp_capability {
    use soda_host::terminal::factory::msp::{
        mcp_probe_outcome, new_command_id, sanitize_mcp_servers, Connection, Error,
    };
    use std::io::{BufRead, BufReader, Write};
    use std::os::unix::net::UnixStream;
    use std::sync::{Arc, Mutex};
    use std::time::{Duration, Instant};

    #[derive(Default)]
    struct PeerSeen {
        requested: Vec<String>,
        session_starts: u64,
        mcp_config: bool,
        list_calls: u64,
        list_clean: bool,
    }

    fn deadline() -> Instant {
        Instant::now() + Duration::from_secs(10)
    }

    fn spawn_peer(
        granted: Option<serde_json::Value>,
        start_error: Option<serde_json::Value>,
    ) -> (UnixStream, Arc<Mutex<PeerSeen>>) {
        let (client, peer) = UnixStream::pair().expect("socket pair");
        let seen = Arc::new(Mutex::new(PeerSeen::default()));
        let seen_peer = Arc::clone(&seen);
        std::thread::spawn(move || {
            let mut reader = BufReader::new(peer.try_clone().expect("clone"));
            let mut writer = peer;
            let mut line = String::new();
            loop {
                line.clear();
                if reader.read_line(&mut line).unwrap_or(0) == 0 {
                    return;
                }
                let frame: serde_json::Value = match serde_json::from_str(&line) {
                    Ok(frame) => frame,
                    Err(_) => return,
                };
                let method = frame.get("method").and_then(|m| m.as_str()).unwrap_or("");
                let id = frame.get("id").cloned().unwrap_or(serde_json::Value::Null);
                if method == "initialize" {
                    let mut seen = seen_peer.lock().expect("seen");
                    if let Some(requested) = frame
                        .get("params")
                        .and_then(|p| p.get("capabilities"))
                        .and_then(|c| c.get("requestedCapabilities"))
                        .and_then(|r| r.as_array())
                    {
                        seen.requested = requested
                            .iter()
                            .filter_map(|v| v.as_str().map(str::to_owned))
                            .collect();
                    }
                    let mut result =
                        serde_json::json!({"platformOs": "linux", "sessionDurability": "durable"});
                    if let Some(granted) = &granted {
                        result["grantedCapabilities"] = granted.clone();
                    }
                    let reply = serde_json::json!({"jsonrpc": "2.0", "id": id, "result": result});
                    writeln!(writer, "{reply}").expect("reply");
                } else if method == "session/start" {
                    let mut seen = seen_peer.lock().expect("seen");
                    seen.session_starts += 1;
                    // The staged per-run settings already register the
                    // `factory` relay: like the observed daemon -32030, a
                    // start that repeats the name is a name conflict.
                    seen.mcp_config = frame
                        .get("params")
                        .and_then(|p| p.get("config"))
                        .and_then(|c| c.get("mcpServers"))
                        .and_then(|s| s.get("factory"))
                        .is_some();
                    let reply = if let Some(error) = &start_error {
                        serde_json::json!({"jsonrpc": "2.0", "id": id, "error": error})
                    } else if seen.mcp_config {
                        serde_json::json!({"jsonrpc": "2.0", "id": id,
                            "error": {"code": -32030, "data": {"reason": "sessionMcpNameConflict"}}})
                    } else {
                        let params = frame.get("params").cloned().unwrap_or_default();
                        serde_json::json!({"jsonrpc": "2.0", "id": id, "result": {"session": {
                            "providerId": "meta",
                            "workspaceRoot": params.get("workspaceRoot").cloned().unwrap_or_default(),
                            "modelId": params.get("modelId").cloned().unwrap_or_default(),
                            "sessionId": "fake-session-1",
                            "status": "idle",
                            "activeTurnId": null,
                            "turnCount": 0,
                        }}})
                    };
                    writeln!(writer, "{reply}").expect("reply");
                } else if method == "mcpServer/list" {
                    let mut seen = seen_peer.lock().expect("seen");
                    seen.list_calls += 1;
                    seen.list_clean = frame
                        .get("params")
                        .and_then(|p| p.as_object())
                        .map(|o| {
                            o.len() == 1
                                && frame
                                    .get("params")
                                    .and_then(|p| p.get("sessionId"))
                                    .and_then(|s| s.as_str())
                                    == Some("fake-session-1")
                        })
                        .unwrap_or(false);
                    let long = "x".repeat(2000);
                    let reply = serde_json::json!({"jsonrpc": "2.0", "id": id, "result": {"servers": [
                        {"name": "factory", "source": "session", "transport": "stdio",
                         "command": "/secret/relay", "url": "https://secret.invalid",
                         "env": {"TOKEN": "secret"}, "oauth": "bearer-secret"},
                        {"name": long, "source": "settings"},
                    ]}});
                    writeln!(writer, "{reply}").expect("reply");
                }
                // Notifications (e.g. initialized) carry no id and need no reply.
            }
        });
        (client, seen)
    }

    fn workspace() -> (String, String) {
        let root = "/ws".to_string();
        let home = format!("{root}/.soda-home/runs/a1b2c3d4e5f60718293a4b5c6d7e8f90/home");
        (root, home)
    }

    fn start_once(
        connection: &mut Connection<UnixStream, UnixStream>,
        observe: &mut impl FnMut(soda_host::terminal::factory::msp::Notice) -> Result<(), Error>,
    ) -> Result<String, Error> {
        let (root, home) = workspace();
        connection
            .start_factory_session(
                &new_command_id().expect("command"),
                &root,
                "muse-spark-1.3",
                &home,
                deadline(),
                observe,
            )
            .map(|snapshot| snapshot.id)
    }

    #[test]
    fn mcp_capable_peer_completes_mcp_session_start() {
        let (client, seen) = spawn_peer(Some(serde_json::json!(["rawLog", "sessionMcp"])), None);
        let mut connection = Connection::new(client.try_clone().expect("clone"), client);
        let mut observe = |_notice| Ok(());
        connection
            .initialize(deadline(), &mut observe)
            .expect("initialize");
        let guard = seen.lock().expect("seen");
        assert!(
            guard.requested.iter().any(|c| c == "sessionMcp"),
            "initialize must request sessionMcp, got {:?}",
            guard.requested
        );
        drop(guard);
        let id = start_once(&mut connection, &mut observe).expect("session start");
        assert_eq!(id, "fake-session-1");
        let seen = seen.lock().expect("seen");
        assert_eq!(seen.session_starts, 1);
        assert!(
            !seen.mcp_config,
            "session/start must not duplicate the staged Factory MCP registration"
        );
    }

    #[test]
    fn mcp_list_probe_sends_only_session_id() {
        let (client, seen) = spawn_peer(Some(serde_json::json!(["rawLog", "sessionMcp"])), None);
        let mut connection = Connection::new(client.try_clone().expect("clone"), client);
        let mut observe = |_notice| Ok(());
        connection
            .initialize(deadline(), &mut observe)
            .expect("initialize");
        start_once(&mut connection, &mut observe).expect("session start");
        let result = connection
            .mcp_server_list("fake-session-1", deadline(), &mut observe)
            .expect("list");
        assert_eq!(
            result
                .get("servers")
                .and_then(|s| s.as_array())
                .map(|a| a.len()),
            Some(2)
        );
        let seen = seen.lock().expect("seen");
        assert_eq!(seen.list_calls, 1);
        assert!(
            seen.list_clean,
            "list params must carry only the session id"
        );
    }

    #[test]
    fn sanitize_mcp_servers_keeps_only_presence_and_total() {
        let dirty = serde_json::json!({"servers": [
            {"name": "factory", "source": "session", "transport": "stdio",
             "command": "/secret/relay", "url": "https://secret.invalid",
             "env": {"TOKEN": "v"}, "oauth": "bearer"},
            {"name": "other", "source": "settings"},
            "not-an-object",
            {"command": "no-name"},
        ], "extra": "dropped"});
        let clean = sanitize_mcp_servers(&dirty);
        assert_eq!(
            clean,
            serde_json::json!({"factory_present": true, "total": 4})
        );
        assert!(
            !clean.to_string().contains("secret"),
            "no daemon string, command, URL, env or credential may survive"
        );
        let absent = sanitize_mcp_servers(&serde_json::json!({"servers": [{"name": "other"}]}));
        assert_eq!(
            absent,
            serde_json::json!({"factory_present": false, "total": 1})
        );
    }

    #[test]
    fn sanitize_mcp_servers_treats_missing_or_malformed_lists_as_unknown() {
        for raw in [
            serde_json::json!({}),
            serde_json::json!({"servers": "nope"}),
            serde_json::json!({"servers": null}),
        ] {
            assert_eq!(
                sanitize_mcp_servers(&raw),
                serde_json::json!({"factory_present": false, "total": 0})
            );
        }
    }

    #[test]
    fn mcp_list_with_spent_deadline_fails_fast_without_hang() {
        let (client, seen) = spawn_peer(Some(serde_json::json!(["rawLog", "sessionMcp"])), None);
        let mut connection = Connection::new(client.try_clone().expect("clone"), client);
        let mut observe = |_notice| Ok(());
        connection
            .initialize(deadline(), &mut observe)
            .expect("initialize");
        let spent = Instant::now()
            .checked_sub(Duration::from_secs(1))
            .expect("past deadline");
        let error = connection
            .mcp_server_list("fake-session-1", spent, &mut observe)
            .expect_err("spent deadline must fail fast");
        assert!(
            matches!(error, Error::Deadline | Error::Io(_)),
            "spent deadline must fail locally, got {error:?}"
        );
        assert_eq!(seen.lock().expect("seen").list_calls, 0);
        let (summary, blocked) = mcp_probe_outcome(Err(error));
        assert_eq!(
            summary,
            serde_json::json!({"factory_present": false, "total": 0, "unavailable": true})
        );
        assert!(blocked.is_some());
    }

    #[test]
    fn probe_outcome_records_and_proceeds_on_empty_or_refused_lists() {
        let empty: Vec<(serde_json::Value, serde_json::Value)> = vec![
            (
                serde_json::json!({"servers": []}),
                serde_json::json!({"factory_present": false, "total": 0}),
            ),
            (
                serde_json::json!({}),
                serde_json::json!({"factory_present": false, "total": 0}),
            ),
            (
                serde_json::json!({"servers": [{"name": "factory", "command": "/x"}]}),
                serde_json::json!({"factory_present": true, "total": 1}),
            ),
        ];
        for (raw, expected) in empty {
            let (summary, blocked) = mcp_probe_outcome(Ok(raw));
            assert_eq!(summary, expected);
            assert!(blocked.is_none(), "empty list is unknown, not a failure");
        }
        let (summary, blocked) = mcp_probe_outcome(Err(Error::Refused {
            code: -32030,
            reason: Some("session_mcp_name_conflict".to_string()),
        }));
        assert_eq!(
            summary,
            serde_json::json!({"factory_present": false, "total": 0, "refused": -32030})
        );
        assert!(blocked.is_none(), "structured refusal records and proceeds");
    }

    #[test]
    fn probe_outcome_stops_before_turn_on_transport_uncertainty() {
        let uncertain: Vec<Error> = vec![
            Error::Deadline,
            Error::Bounds,
            Error::End,
            Error::Gap,
            Error::InputRequired,
            Error::Invalid("native reply"),
            Error::Io(std::io::Error::new(std::io::ErrorKind::BrokenPipe, "probe")),
        ];
        for error in uncertain {
            let (summary, blocked) = mcp_probe_outcome(Err(error));
            assert_eq!(
                summary,
                serde_json::json!({"factory_present": false, "total": 0, "unavailable": true})
            );
            assert!(blocked.is_some(), "uncertainty must stop before any turn");
        }
    }

    #[test]
    fn duplicate_factory_name_is_refused_as_name_conflict() {
        // Pins the emulated daemon contract behind the observed -32030:
        // a session/start that repeats the staged `factory` name is
        // refused, so the product must keep that single registration.
        let (client, seen) = spawn_peer(Some(serde_json::json!(["rawLog", "sessionMcp"])), None);
        let mut reader = BufReader::new(client.try_clone().expect("clone"));
        let mut writer = client;
        let mut line = String::new();
        let request = serde_json::json!({"jsonrpc": "2.0", "id": 1,
            "method": "session/start",
            "params": {"commandId": "00".repeat(16), "workspaceRoot": "/ws",
                "providerId": "meta", "approvalMode": "allowAll",
                "config": {"mcpServers": {"factory": {"transport": "stdio"}}}}});
        writeln!(writer, "{request}").expect("write");
        line.clear();
        reader.read_line(&mut line).expect("reply");
        let reply: serde_json::Value = serde_json::from_str(&line).expect("json");
        assert_eq!(reply.get("id"), Some(&serde_json::json!(1)));
        assert_eq!(
            reply.get("error").and_then(|e| e.get("code")),
            Some(&serde_json::json!(-32030))
        );
        assert!(seen.lock().expect("seen").mcp_config);
    }

    #[test]
    fn peer_without_session_mcp_still_starts_staged_relay() {
        // The staged settings carry the required relay, so a denied grant
        // never blocks session/start: only the daemon can report a relay
        // it cannot start.
        let (client, seen) = spawn_peer(Some(serde_json::json!(["rawLog"])), None);
        let mut connection = Connection::new(client.try_clone().expect("clone"), client);
        let mut observe = |_notice| Ok(());
        connection
            .initialize(deadline(), &mut observe)
            .expect("initialize");
        let id = start_once(&mut connection, &mut observe).expect("session start");
        assert_eq!(id, "fake-session-1");
        assert_eq!(seen.lock().expect("seen").session_starts, 1);
    }

    #[test]
    fn peer_missing_grant_list_still_starts_staged_relay() {
        let (client, seen) = spawn_peer(None, None);
        let mut connection = Connection::new(client.try_clone().expect("clone"), client);
        let mut observe = |_notice| Ok(());
        connection
            .initialize(deadline(), &mut observe)
            .expect("initialize");
        let id = start_once(&mut connection, &mut observe).expect("session start");
        assert_eq!(id, "fake-session-1");
        assert_eq!(seen.lock().expect("seen").session_starts, 1);
    }

    #[test]
    fn peer_malformed_grant_list_still_starts_staged_relay() {
        let (client, seen) = spawn_peer(Some(serde_json::json!("sessionMcp")), None);
        let mut connection = Connection::new(client.try_clone().expect("clone"), client);
        let mut observe = |_notice| Ok(());
        connection
            .initialize(deadline(), &mut observe)
            .expect("initialize");
        let id = start_once(&mut connection, &mut observe).expect("session start");
        assert_eq!(id, "fake-session-1");
        assert_eq!(seen.lock().expect("seen").session_starts, 1);
    }

    #[test]
    fn peer_mixed_grant_list_still_starts_staged_relay() {
        let (client, seen) = spawn_peer(Some(serde_json::json!(["sessionMcp", null, 7])), None);
        let mut connection = Connection::new(client.try_clone().expect("clone"), client);
        let mut observe = |_notice| Ok(());
        connection
            .initialize(deadline(), &mut observe)
            .expect("initialize");
        let id = start_once(&mut connection, &mut observe).expect("session start");
        assert_eq!(id, "fake-session-1");
        assert_eq!(seen.lock().expect("seen").session_starts, 1);
    }

    #[test]
    fn refusal_without_structured_reason_carries_no_text() {
        let error_body =
            serde_json::json!({"code": -32010, "message": "leaky /secret/path and prompt text"});
        let (client, _) = spawn_peer(
            Some(serde_json::json!(["rawLog", "sessionMcp"])),
            Some(error_body),
        );
        let mut connection = Connection::new(client.try_clone().expect("clone"), client);
        let mut observe = |_notice| Ok(());
        connection
            .initialize(deadline(), &mut observe)
            .expect("initialize");
        let error = start_once(&mut connection, &mut observe).expect_err("must refuse");
        match error {
            Error::Refused { code, reason } => {
                assert_eq!(code, -32010);
                assert_eq!(reason, None, "free-text message must never surface");
            }
            other => panic!("expected refusal, got {other:?}"),
        }
    }

    #[test]
    fn refusal_with_structured_reason_keeps_validated_code() {
        let error_body =
            serde_json::json!({"code": -32010, "data": {"reason": "capabilityRequired"}});
        let (client, _) = spawn_peer(
            Some(serde_json::json!(["rawLog", "sessionMcp"])),
            Some(error_body),
        );
        let mut connection = Connection::new(client.try_clone().expect("clone"), client);
        let mut observe = |_notice| Ok(());
        connection
            .initialize(deadline(), &mut observe)
            .expect("initialize");
        let error = start_once(&mut connection, &mut observe).expect_err("must refuse");
        match error {
            Error::Refused { code, reason } => {
                assert_eq!(code, -32010);
                assert_eq!(reason.as_deref(), Some("capabilityRequired"));
            }
            other => panic!("expected refusal, got {other:?}"),
        }
    }
}

// -- factory phase prompt bounds (intent decompose headroom) --

use soda_host::factory::{
    max_prompt_for_phase, EXECUTION_DECOMPOSE, EXECUTION_IMPLEMENT, EXECUTION_REASSESS,
    EXECUTION_REPAIR,
};
use soda_host::terminal::factory::run::{
    max_staging_prompt_for_phase, MAX_DECOMPOSE_PROMPT as STAGING_DECOMPOSE_PROMPT,
    MAX_FACTORY_PROMPT as STAGING_FACTORY_PROMPT, MAX_REASSESS_PROMPT as STAGING_REASSESS_PROMPT,
};

#[test]
fn decompose_phase_keeps_intent_headroom_on_both_gates() {
    assert_eq!(STAGING_DECOMPOSE_PROMPT, 128 * 1024);
    assert_eq!(max_prompt_for_phase(EXECUTION_DECOMPOSE), 128 * 1024);
    assert_eq!(max_staging_prompt_for_phase("decompose"), 128 * 1024);
}

// -- executor stdout-cap typed bounds signal (FJ-745) --

/// Genuine executor→route bounds contract: the real native executor's
/// stdout-ceiling overflow is a typed `BoundedError::StdoutLimit`
/// (never a string), exact-cap output fits (the legal max64 boundary),
/// and stderr/exit-status failures stay distinct `Failed` and fail
/// closed. The typed signal then walks the 725 chain to the route's
/// 413 through the real dispatch.
#[test]
fn native_bounded_stdout_overflow_is_typed_bounds() {
    use soda_host::project::{BoundedError, Executor, Native};
    use std::time::{Duration, Instant};
    let deadline = || Instant::now() + Duration::from_secs(30);

    // Production-shaped boundary mirroring collect_approved_files: one
    // 32KiB approved file reads through cap+1. Exact-cap output fits,
    // so the legal max64 singles still confirm.
    let fit = Native
        .run_bounded_typed(
            &[],
            "/usr/bin/head",
            &["-c", "32768", "/dev/zero"],
            deadline(),
            32769,
            1024,
        )
        .expect("exact-cap output must fit");
    assert_eq!(fit.len(), 32768);

    // Exactly cap+1 output still returns: the producing-site
    // `content.len() > cap` check (same Bounds message) catches this
    // exact boundary, while the executor below catches everything past
    // it. Both layers are typed.
    let boundary = Native
        .run_bounded_typed(
            &[],
            "/usr/bin/head",
            &["-c", "32769", "/dev/zero"],
            deadline(),
            32769,
            1024,
        )
        .expect("cap+1 output must return for the site check");
    assert!(boundary.len() > 32768);

    // Past cap+1 the executor itself overflows typed — no string matched.
    let overflow = Native
        .run_bounded_typed(
            &[],
            "/usr/bin/head",
            &["-c", "32770", "/dev/zero"],
            deadline(),
            32769,
            1024,
        )
        .expect_err("cap+2 output must overflow");
    assert_eq!(overflow, BoundedError::StdoutLimit);

    // Stderr-ceiling overflow is NOT bounds: distinct, fail-closed.
    let stderr = Native
        .run_bounded_typed(
            &[],
            "/usr/bin/sh",
            &["-c", "head -c 2048 /dev/zero >&2"],
            deadline(),
            65536,
            1024,
        )
        .expect_err("stderr overflow must fail");
    assert!(matches!(stderr, BoundedError::Failed(_)));
    assert_ne!(stderr, BoundedError::StdoutLimit);

    // A nonzero exit is NOT bounds either.
    let status = Native
        .run_bounded_typed(
            &[],
            "/usr/bin/sh",
            &["-c", "exit 3"],
            deadline(),
            65536,
            1024,
        )
        .expect_err("nonzero exit must fail");
    assert!(matches!(status, BoundedError::Failed(_)));
    assert_ne!(status, BoundedError::StdoutLimit);

    // The legacy String surface is preserved byte-identical for the
    // other native callers (terminal, factory, tailnet, ...).
    let legacy = Native
        .run_bounded(
            &[],
            "/usr/bin/head",
            &["-c", "32770", "/dev/zero"],
            deadline(),
            32769,
            1024,
        )
        .expect_err("cap+2 output must overflow");
    assert_eq!(
        legacy,
        "/usr/bin/head failed: output capture limit exceeded"
    );

    // The typed signal walks the 725 chain to the route's 413: the same
    // match arms as context_git_bounded, dbackend/backend.rs and
    // daemon/routes.rs, through the real dispatch.
    let context_error = match overflow {
        BoundedError::StdoutLimit => soda_host::prepare::ContextReadError::Bounds(
            "approved-base context file exceeds its bound",
        ),
        BoundedError::Failed(message) => soda_host::prepare::ContextReadError::Failed(message),
    };
    let backend_error = match context_error {
        soda_host::prepare::ContextReadError::Bounds(_) => BackendError::ContextBounds,
        soda_host::prepare::ContextReadError::Failed(_) => BackendError::Internal,
    };
    assert_eq!(backend_error, BackendError::ContextBounds);
    let backend = ScriptBackend::err(backend_error);
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let response = dispatch(
        &backend,
        &DaemonConfig::all_enabled(),
        &gate,
        &terminal_gate,
        &head("POST", "/prepare-context"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 413);
    assert_eq!(body_of(&response), b"preparation context exceeds bounds\n");
}

/// Mixed-failure precedence: overflow plus an incomplete transfer, or
/// overflow plus a nonzero exit, stays fail-closed (never
/// `StdoutLimit`), on both Native and NativeStatusOnly. Pure overflow
/// with a live leader still reports the kill status as bounds.
#[test]
fn native_bounded_mixed_failures_stay_fail_closed() {
    use soda_host::project::{BoundedError, Executor, Native, NativeStatusOnly};
    use std::time::{Duration, Instant};
    let deadline = || Instant::now() + Duration::from_secs(30);

    // Overflow plus incomplete stdin capture: the leader exits at once
    // while one background writer overflows stdout after the exit and a
    // second background sleeper holds the stdin read end open without
    // reading (1 MiB, never consumed), so the pump misses the bound.
    // The leader parks the pipe on fd 3 first: dash backgrounds stdin
    // from /dev/null, so only an inherited fd 3 keeps the read end open
    // past the exit (a dead reader would read as Done, not Incomplete).
    // The holder points its own stdout/stderr at /dev/null so the
    // overflow SIGPIPE cannot take it down early. Incomplete beats the
    // flag: Failed with the capture diagnostic, never bounds.
    for executor in ["native", "status-only"] {
        let overflow = if executor == "native" {
            Native
                .run_bounded_typed(
                    &vec![0u8; 1 << 20],
                    "/usr/bin/sh",
                    &[
                        "-c",
                        "exec 3<&0; (sleep 30) 1>/dev/null 2>/dev/null & (sleep 5; head -c 100000 /dev/zero) 2>/dev/null & exit 0",
                    ],
                    deadline(),
                    1024,
                    1024,
                )
                .expect_err("overflow with incomplete capture must fail")
        } else {
            NativeStatusOnly
                .run_bounded_typed(
                    &vec![0u8; 1 << 20],
                    "/usr/bin/sh",
                    &[
                        "-c",
                        "exec 3<&0; (sleep 30) 1>/dev/null 2>/dev/null & (sleep 5; head -c 100000 /dev/zero) 2>/dev/null & exit 0",
                    ],
                    deadline(),
                    1024,
                    1024,
                )
                .expect_err("overflow with incomplete capture must fail")
        };
        assert_ne!(overflow, BoundedError::StdoutLimit);
        match overflow {
            BoundedError::Failed(message) => assert!(
                message.contains("deadline exceeded"),
                "incomplete capture, got: {message}"
            ),
            BoundedError::StdoutLimit => unreachable!("asserted above"),
        }
    }

    // Overflow plus a nonzero exit: the leader exits 3 at once while a
    // background writer overflows stdout after the exit (its stderr is
    // detached so only the stdout flag fires). The reaped status beats
    // the flag: Failed with the exit diagnostic, never bounds.
    let exited = Native
        .run_bounded_typed(
            &[],
            "/usr/bin/sh",
            &[
                "-c",
                "(sleep 5; head -c 100000 /dev/zero) 2>/dev/null & exit 3",
            ],
            deadline(),
            1024,
            1024,
        )
        .expect_err("overflow with nonzero exit must fail");
    assert_ne!(exited, BoundedError::StdoutLimit);
    assert!(
        matches!(exited, BoundedError::Failed(_)),
        "nonzero exit, got: {exited:?}"
    );
    let exited_status_only = NativeStatusOnly
        .run_bounded_typed(
            &[],
            "/usr/bin/sh",
            &[
                "-c",
                "(sleep 5; head -c 100000 /dev/zero) 2>/dev/null & exit 3",
            ],
            deadline(),
            1024,
            1024,
        )
        .expect_err("overflow with nonzero exit must fail");
    assert_eq!(
        exited_status_only,
        BoundedError::Failed("exit status 3".to_string())
    );

    // The legacy String surface carries the same fail-closed exit
    // diagnostic for the other native callers.
    let legacy = Native
        .run_bounded(
            &[],
            "/usr/bin/sh",
            &[
                "-c",
                "(sleep 5; head -c 100000 /dev/zero) 2>/dev/null & exit 3",
            ],
            deadline(),
            1024,
            1024,
        )
        .expect_err("overflow with nonzero exit must fail");
    assert_eq!(legacy, "/usr/bin/sh failed: exit status 3: ");

    // Pure overflow with complete transfers and a clean exit stays
    // bounds on both executors: the kill status of a live leader never
    // leaks into the diagnostic.
    for executor in ["native", "status-only"] {
        let pure = if executor == "native" {
            Native
                .run_bounded_typed(
                    &[],
                    "/usr/bin/head",
                    &["-c", "2000", "/dev/zero"],
                    deadline(),
                    1024,
                    1024,
                )
                .expect_err("pure overflow must refuse")
        } else {
            NativeStatusOnly
                .run_bounded_typed(
                    &[],
                    "/usr/bin/head",
                    &["-c", "2000", "/dev/zero"],
                    deadline(),
                    1024,
                    1024,
                )
                .expect_err("pure overflow must refuse")
        };
        assert_eq!(pure, BoundedError::StdoutLimit);
    }
}

/// Live-leader overflow+nonzero race: the child overflows stdout in the
/// foreground and exits 3 ~simultaneously through the live-leader path.
/// The flag causally precedes the exit (the overflow drain drops the
/// read end, the writer SIGPIPEs, dash exits 3), and the poll loop checks
/// the flag before the waiter each iteration, so the exclusive-waiter
/// branch always runs with the reaped exit-3 status in hand. That status
/// beats the flag: Failed with the exit diagnostic, never `StdoutLimit`,
/// on both executors plus the legacy String surface. A pure overflow
/// whose live leader is SIGKILLed by the branch cleanup (signal death,
/// no exit code) still reports the kill status as bounds.
/// Real native executor, isolated fixtures, true exits.
#[test]
fn native_bounded_live_leader_overflow_honors_nonzero_exit() {
    use soda_host::project::{BoundedError, Executor, Native, NativeStatusOnly};
    use std::time::{Duration, Instant};
    let deadline = || Instant::now() + Duration::from_secs(30);

    // 200 KiB foreground write: the leader blocks on the full pipe while
    // the drain flags on its first read, so the flag strictly precedes
    // the SIGPIPE death and `exit 3`. No background writer, no sleep:
    // the leader is alive when the flag fires (live-leader branch) yet
    // already a natural exit-3 zombie when the branch cleanup reaps.
    let script = "head -c 200000 /dev/zero; exit 3";
    let exited = Native
        .run_bounded_typed(&[], "/usr/bin/sh", &["-c", script], deadline(), 1024, 1024)
        .expect_err("live-leader overflow with nonzero exit must fail");
    assert_ne!(exited, BoundedError::StdoutLimit);
    match &exited {
        BoundedError::Failed(message) => assert!(
            message.contains("exit status 3"),
            "live-leader nonzero exit, got: {message}"
        ),
        BoundedError::StdoutLimit => unreachable!("asserted above"),
    }

    let exited_status_only = NativeStatusOnly
        .run_bounded_typed(&[], "/usr/bin/sh", &["-c", script], deadline(), 1024, 1024)
        .expect_err("live-leader overflow with nonzero exit must fail");
    assert_eq!(
        exited_status_only,
        BoundedError::Failed("exit status 3".to_string())
    );

    // The legacy String surface carries the same fail-closed exit
    // diagnostic for the other native callers.
    let legacy = Native
        .run_bounded(&[], "/usr/bin/sh", &["-c", script], deadline(), 1024, 1024)
        .expect_err("live-leader overflow with nonzero exit must fail");
    assert_eq!(legacy, "/usr/bin/sh failed: exit status 3: ");
}

/// Genuine overflow route: a real Native executor drives
/// `read_preparation_context` against an isolated podman fixture whose
/// candidate diff overflows the bounded path-set read; the typed Bounds
/// signal walks the real DaemonBackend into the real `/prepare-context`
/// dispatch for a 413. No fake peer anywhere on this path: every
/// overflow byte is real `git diff` output through real pipes and
/// limits. The test builds its own fixture container (local wiki image
/// for its git shell, role `soda-coder`, ready coder checkout) and
/// removes it afterwards; the only assumed host state is podman plus
/// the root-owned empty admission directory from production boot.
#[test]
fn native_bounded_overflow_walks_genuine_route_to_413() {
    let project = "p765765765765765765765765";
    let id = "f766766766766766766766766";
    let container = format!("soda-{project}");
    let checkout = format!("/home/soda-coder/checkouts/{id}");
    // 3500 changed paths (~84 KiB of `--name-only -z` output) overflow
    // the 3073-byte bounded path-set read past any pipe buffer, so the
    // leader is still alive and blocked when the flag fires.
    let commits = build_overflow_fixture(&container, &checkout);
    let body = format!(
        "{{\"project\":\"{project}\",\"id\":\"{id}\",\
         \"source_commit\":\"{base}\",\"approved_base\":\"{base}\",\
         \"diff_base\":\"{base}\",\"candidate\":\"{candidate}\",\
         \"paths\":[],\"not_after\":\"2030-01-01T00:00:00Z\"}}",
        base = commits.base,
        candidate = commits.candidate,
    );
    let backend = soda_host::dbackend::DaemonBackend::open(
        soda_host::dbackend::BackendConfig {
            canlang_runtime: None,
            project: soda_host::project::Config {
                muse_socket: String::new(),
                image: String::new(),
                network: String::new(),
                subnet: String::new(),
                bridge: String::new(),
            },
            codex_harness: String::new(),
            codex_harness_sha256: String::new(),
            codex_harness_version: String::new(),
            codex_model_catalog: String::new(),
            codex_model_catalog_sha256: String::new(),
            muse_harness: String::new(),
            muse_harness_sha256: String::new(),
            muse_harness_version: String::new(),
            broker_socket: String::new(),
            tailnet_image: String::new(),
            release_payload: String::new(),
            muse: None,
        },
        "/home/vince/.muse-fj-765-fixture-state",
    );
    match backend.prepare_context(body.as_bytes()) {
        Err(BackendError::ContextBounds) => {}
        other => panic!("genuine overflow must map ContextBounds, got: {other:?}"),
    }
    let gate = AdmissionGate::new();
    let terminal_gate = TerminalGate::new();
    let response = dispatch(
        &backend,
        &DaemonConfig::all_enabled(),
        &gate,
        &terminal_gate,
        &head("POST", "/prepare-context"),
        body.as_bytes(),
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 413);
    assert_eq!(body_of(&response), b"preparation context exceeds bounds\n");
    fixture_run(&["podman", "rm", "-f", &container], &[]);
}

/// Base/candidate commits of the genuine-route fixture checkout.
struct FixtureCommits {
    base: String,
    candidate: String,
}

/// Run one fixture command with true exits: any failure panics with the
/// exact argv, exit status and raw stdout/stderr. Isolated fixtures only.
fn fixture_run(argv: &[&str], stdin_bytes: &[u8]) -> Vec<u8> {
    use std::io::Write;
    use std::process::{Command, Stdio};
    let (program, args) = argv.split_first().expect("fixture argv");
    let mut child = Command::new(program)
        .args(args)
        .stdin(if stdin_bytes.is_empty() {
            Stdio::null()
        } else {
            Stdio::piped()
        })
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .unwrap_or_else(|e| panic!("fixture spawn {argv:?} failed: {e}"));
    if !stdin_bytes.is_empty() {
        child
            .stdin
            .take()
            .expect("fixture stdin")
            .write_all(stdin_bytes)
            .unwrap_or_else(|e| panic!("fixture stdin {argv:?} failed: {e}"));
    }
    let output = child
        .wait_with_output()
        .unwrap_or_else(|e| panic!("fixture wait {argv:?} failed: {e}"));
    if !output.status.success() {
        panic!(
            "fixture command {argv:?} exited {}: stdout={:?} stderr={:?}",
            output.status,
            String::from_utf8_lossy(&output.stdout),
            String::from_utf8_lossy(&output.stderr)
        );
    }
    output.stdout
}

/// Build the isolated overflow fixture: a podman container shaped like a
/// recorded ready preparation (identity labels, private userns with a
/// shifted map, role user, in-container helper answering `inspect`, git
/// checkout whose candidate adds 3500 files). Returns the base and
/// candidate commits. The container is replaced when present and
/// removed by the caller.
fn build_overflow_fixture(container: &str, checkout: &str) -> FixtureCommits {
    let project = container.strip_prefix("soda-").expect("fixture name");
    // Tolerated: the fixture may not exist yet. Every later step uses
    // true exits through fixture_run.
    {
        use std::process::{Command, Stdio};
        let _ = Command::new("podman")
            .args(["rm", "-f", container])
            .stdin(Stdio::null())
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status();
    }
    fixture_run(
        &[
            "podman",
            "run",
            "-d",
            "--name",
            container,
            "--label",
            &format!("org.soda.project={project}"),
            "--label",
            "org.soda.owner=1000",
            "--userns=auto:size=65536",
            "ghcr.io/requarks/wiki:2",
            "sleep",
            "3600",
        ],
        &[],
    );
    // The image entrypoint is slow; the first exec below doubles as the
    // readiness gate and fails loudly when the image is missing.
    fixture_run(
        &[
            "podman",
            "exec",
            "--user",
            "root",
            container,
            "sh",
            "-c",
            &format!(
                "echo 'soda-coder:x:1001:1001::/home/soda-coder:/bin/sh' >> /etc/passwd \
                 && echo 'soda-coder:x:1001:' >> /etc/group \
                 && mkdir -p /home/soda-coder {checkout} /usr/libexec/soda \
                 && chown 1001:1001 /home/soda-coder {checkout}"
            ),
        ],
        &[],
    );
    // Minimal factory helper: reports this checkout as the recorded
    // ready coder preparation. The source commit is filled in below.
    let out = fixture_run(
        &[
            "podman",
            "exec",
            "--user",
            "soda-coder",
            container,
            "sh",
            "-c",
            &format!(
                "cd {checkout} && git init -q -b main . \
                 && git config user.email fixture@invalid \
                 && git config user.name fixture \
                 && echo base > AGENTS.md && git add -A && git commit -qm base \
                 && for i in $(seq 1 3500); do echo \"candidate $i\" \
                 > \"overflow-file-$(printf %05d $i).txt\"; done \
                 && git add -A && git commit -qm candidate \
                 && git rev-parse HEAD~1 && git rev-parse HEAD \
                 && git diff --no-ext-diff --no-textconv --no-renames \
                 --name-only -z HEAD~1 HEAD | wc -c"
            ),
        ],
        &[],
    );
    let lines: Vec<&str> = std::str::from_utf8(&out)
        .expect("fixture hashes")
        .lines()
        .collect();
    assert_eq!(lines.len(), 3, "fixture commits output: {lines:?}");
    let width: usize = lines[2].trim().parse().expect("fixture diff width");
    assert!(
        width > 12 * 256 + 1,
        "fixture diff must overflow the path-set bound, got {width}"
    );
    let helper = format!(
        "#!/bin/sh\nREQUEST=$(cat)\ncase \"$REQUEST\" in *'\"op\":\"inspect\"'*) \
         printf '%s' '{{\"known\":true,\"phase\":\"ready\",\"role\":\"soda-coder\",\
         \"setup_digest\":\"0000000000000000000000000000000000000000000000000000000000000000\",\
         \"source_commit\":\"{}\",\"ready\":true}}' ;; \
         *) printf '%s' '{{\"known\":false}}' ;; esac\n",
        lines[0].trim()
    );
    fixture_run(
        &[
            "podman",
            "exec",
            "-i",
            "--user",
            "root",
            container,
            "sh",
            "-c",
            "cat > /usr/libexec/soda/project-factory-roles \
             && chmod +x /usr/libexec/soda/project-factory-roles",
        ],
        helper.as_bytes(),
    );
    FixtureCommits {
        base: lines[0].trim().to_string(),
        candidate: lines[1].trim().to_string(),
    }
}

#[test]
fn unaffected_phases_keep_their_exact_bounds_on_both_gates() {
    assert_eq!(STAGING_FACTORY_PROMPT, 64 * 1024);
    assert_eq!(STAGING_REASSESS_PROMPT, 320 * 1024);
    assert_eq!(max_prompt_for_phase(EXECUTION_REASSESS), 320 * 1024);
    assert_eq!(max_prompt_for_phase(EXECUTION_REPAIR), 320 * 1024);
    assert_eq!(max_prompt_for_phase(EXECUTION_IMPLEMENT), 64 * 1024);
    assert_eq!(max_staging_prompt_for_phase("reassess"), 320 * 1024);
    assert_eq!(max_staging_prompt_for_phase("repair"), 320 * 1024);
    assert_eq!(max_staging_prompt_for_phase("implement"), 64 * 1024);
    assert_eq!(max_staging_prompt_for_phase("unknown-phase"), 64 * 1024);
}

// -- real-native setup token rotation (FJ-824: folded here from the
// removed tests/setup_rotation.rs so rotation keeps its existing-target
// home; no new test files) --
//
// Real-native setup token rotation (FJ-v3-789 renew schema + FJ-v3-790
// atomic replacement + FJ-v3-792 live-holder semantics, plus the FJ-809
// wrong-generation refusal / reconcile proof).
//
// The running setup process holds its token+expiry in memory from process
// start, exactly like the Canlang setup entry captures both at start: a
// rotated file alone cannot reach it, which is why renewal stops the
// retained generation before swapping the credential. These tests exercise
// the real rotation seams with real files and a real holder process — no
// mocks, no fake Ready:
//
// * the authorized-renewal predicate (`setup_renewal_advance`),
// * the atomic token swap (`publish_setup_file`) with failure injection
//   at every injectable fallible step, proving the old credential always
//   survives,
// * a real child process holding the old token, proving rotation is
//   invisible to a live holder and that a restart captures the new one.
//
// Scratch state lives under `target/setup-rotation-test/` (never /tmp)
// and is removed on success.

use soda_host::application::{
    classify_reconcile_failure, parse_reconcile_outcome, publish_setup_file,
    reconcile_permits_rotation, setup_renewal_advance, ReconcileExpectation, ReconcileOutcome,
    ReconcileRefusal, ReleaseBinding, Runtime, SetupObservation, SetupRecord, SetupRequest,
    SetupSpec, SnapshotRequest, Spec, Token, RECONCILE_REFUSAL_LINE, RECONCILE_UNCLAIMED_LINE,
    SETUP_RENEWAL_WINDOW_SECS,
};
use soda_host::project::Native;
use std::ffi::CString;
use std::io::{BufRead, BufReader};
use std::os::fd::{AsRawFd, FromRawFd, OwnedFd};
use std::os::unix::fs::{FileTypeExt, MetadataExt, PermissionsExt};
use std::path::PathBuf;
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};
fn scratch(name: &str) -> PathBuf {
    let dir = PathBuf::from("target/setup-rotation-test").join(name);
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).expect("scratch dir");
    dir
}

fn euid() -> u32 {
    unsafe { libc::getuid() }
}

fn setup_spec() -> SetupSpec {
    SetupSpec {
        id: format!("a{}", "b".repeat(24)),
        name: "example".into(),
        artifact_sha256: "2".repeat(64),
        bundle_sha256: "3".repeat(64),
        tools_sha256: "4".repeat(64),
        runtime_image: format!("sha256:{}", "5".repeat(64)),
        image: format!("sha256:{}", "6".repeat(64)),
        data: format!("a{}", "c".repeat(24)),
        origin: "https://setup-rotation.example.test".into(),
        port: 20091,
    }
}

fn setup_record(expires_at: i64) -> SetupRecord {
    SetupRecord {
        id: format!("a{}", "d".repeat(24)),
        application: setup_spec(),
        administrator_id: 41,
        expires_at,
        state: "pending".into(),
        observation: None,
    }
}

fn token(operation: &str, value: &str, expires_at: i64) -> Token {
    Token {
        operation: operation.into(),
        token: value.into(),
        expires_at,
    }
}

fn read_token(path: &std::path::Path) -> Token {
    serde_json::from_slice(&std::fs::read(path).expect("token file")).expect("token json")
}

#[test]
fn renewal_advance_accepts_forward_bounded_expiry() {
    let previous = setup_record(1_000);
    let mut identity = previous.clone();
    identity.expires_at = 1_000 + 1_800;
    assert!(setup_renewal_advance(&previous, &identity, 1_001));
    // Edge: exactly one full window ahead of now is still admitted.
    identity.expires_at = 1_001 + SETUP_RENEWAL_WINDOW_SECS;
    assert!(setup_renewal_advance(&previous, &identity, 1_001));
}

#[test]
fn renewal_advance_rejects_stale_or_unbounded_expiry() {
    let previous = setup_record(1_000);
    let same = previous.clone();
    assert!(!setup_renewal_advance(&previous, &same, 1_001));
    let mut backward = previous.clone();
    backward.expires_at = 999;
    assert!(!setup_renewal_advance(&previous, &backward, 1_001));
    let mut beyond = previous.clone();
    beyond.expires_at = 1_001 + SETUP_RENEWAL_WINDOW_SECS + 1;
    assert!(!setup_renewal_advance(&previous, &beyond, 1_001));
}

#[test]
fn renewal_advance_rejects_any_admission_change() {
    let previous = setup_record(1_000);
    let bumped = |mut r: SetupRecord| {
        r.expires_at = 2_800;
        r
    };
    let mut tampered = bumped(previous.clone());
    tampered.id = format!("a{}", "e".repeat(24));
    assert!(!setup_renewal_advance(&previous, &tampered, 1_001));
    let mut tampered = bumped(previous.clone());
    tampered.application.port = 20092;
    assert!(!setup_renewal_advance(&previous, &tampered, 1_001));
    let mut tampered = bumped(previous.clone());
    tampered.administrator_id = 42;
    assert!(!setup_renewal_advance(&previous, &tampered, 1_001));
    let mut tampered = bumped(previous.clone());
    tampered.state = "complete".into();
    assert!(!setup_renewal_advance(&previous, &tampered, 1_001));
    let mut tampered = bumped(previous.clone());
    tampered.observation = Some(SetupObservation {
        container: "f".repeat(64),
        running: false,
        ready: false,
    });
    assert!(!setup_renewal_advance(&previous, &tampered, 1_001));
}

#[test]
fn atomic_publish_replaces_token_bytes() {
    let dir = scratch("publish-ok");
    let path = dir.join("token.json");
    let uid = euid();
    let old = token("op1", &"1".repeat(64), 1_000);
    publish_setup_file(&path, &old, uid).expect("initial publish");
    let fresh = token("op1", &"2".repeat(64), 2_800);
    publish_setup_file(&path, &fresh, uid).expect("rotation publish");
    let stored = read_token(&path);
    assert_eq!(stored.token, "2".repeat(64));
    assert_eq!(stored.expires_at, 2_800);
    assert_eq!(stored.operation, "op1");
    assert_eq!(
        std::fs::metadata(&path).expect("meta").mode() & 0o777,
        0o600
    );
    std::fs::remove_dir_all(&dir).expect("cleanup");
}

#[test]
fn atomic_publish_stage_failure_preserves_old_token() {
    let dir = scratch("publish-stage-fail");
    let path = dir.join("token.json");
    let uid = euid();
    let old = token("op1", &"1".repeat(64), 1_000);
    publish_setup_file(&path, &old, uid).expect("initial publish");
    let before = std::fs::read(&path).expect("old bytes");
    // Read-only directory: staging the replacement cannot even begin.
    std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o555)).expect("lock");
    let fresh = token("op1", &"2".repeat(64), 2_800);
    let err = publish_setup_file(&path, &fresh, uid).expect_err("stage must fail");
    assert!(err.contains("cannot be staged"), "unexpected: {err}");
    std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o755)).expect("unlock");
    assert_eq!(std::fs::read(&path).expect("old bytes"), before);
    assert_eq!(read_token(&path).token, "1".repeat(64));
    std::fs::remove_dir_all(&dir).expect("cleanup");
}

#[test]
fn atomic_publish_ownership_failure_preserves_old_token() {
    if euid() == 0 {
        eprintln!("running as root: chown cannot fail; skipping ownership injection");
        return;
    }
    let dir = scratch("publish-owner-fail");
    let path = dir.join("token.json");
    let uid = euid();
    let old = token("op1", &"1".repeat(64), 1_000);
    publish_setup_file(&path, &old, uid).expect("initial publish");
    let before = std::fs::read(&path).expect("old bytes");
    // Taking another identity's ownership fails for a non-root publisher.
    let fresh = token("op1", &"2".repeat(64), 2_800);
    let err = publish_setup_file(&path, &fresh, uid + 1).expect_err("chown must fail");
    assert!(err.contains("ownership failed"), "unexpected: {err}");
    assert_eq!(std::fs::read(&path).expect("old bytes"), before);
    assert_eq!(read_token(&path).token, "1".repeat(64));
    std::fs::remove_dir_all(&dir).expect("cleanup");
}

#[test]
fn atomic_publish_rename_failure_preserves_old_state() {
    let dir = scratch("publish-rename-fail");
    // The token path is a live non-empty directory: the staged file cannot
    // rename over it, and the directory must survive with contents intact.
    let path = dir.join("token.json");
    std::fs::create_dir(&path).expect("token dir");
    std::fs::write(path.join("held"), b"held").expect("held");
    let fresh = token("op1", &"2".repeat(64), 2_800);
    let err = publish_setup_file(&path, &fresh, euid()).expect_err("rename must fail");
    assert!(err.contains("publication failed"), "unexpected: {err}");
    assert_eq!(std::fs::read(path.join("held")).expect("held"), b"held");
    // No staged sibling may leak beside the target.
    let leftovers: Vec<_> = std::fs::read_dir(&dir).expect("scan").collect();
    assert_eq!(leftovers.len(), 1);
    std::fs::remove_dir_all(&dir).expect("cleanup");
}

const HOLDER_ENV: &str = "SODA_SETUP_ROTATION_HOLDER";
const HOLDER_SOCK_ENV: &str = "SODA_SETUP_ROTATION_SOCK";
const HOLDER_TOKEN_ENV: &str = "SODA_SETUP_ROTATION_TOKEN";

/// Real token consumer: reads the token file once at process start into
/// memory — exactly the capture semantics of the running setup process —
/// then answers probes against the captured bytes over a socket. Rotation
/// of the file is invisible to it until it restarts.
fn run_holder_child() -> ! {
    let sock = std::env::var(HOLDER_SOCK_ENV).expect("sock");
    let path = std::env::var(HOLDER_TOKEN_ENV).expect("token");
    let raw = std::fs::read_to_string(&path).expect("read token");
    let held: Token = serde_json::from_str(&raw).expect("parse token");
    let stream = UnixStream::connect(&sock).expect("connect");
    let mut reader = BufReader::new(stream.try_clone().expect("clone"));
    let mut writer = stream;
    writer.write_all(b"ready\n").expect("ready");
    let mut line = String::new();
    loop {
        line.clear();
        if reader.read_line(&mut line).expect("read") == 0 {
            std::process::exit(0);
        }
        let probe = line.trim_end();
        if probe == "quit" {
            std::process::exit(0);
        }
        let mut parts = probe.splitn(2, ' ');
        let (value, expiry) = (parts.next().unwrap_or(""), parts.next().unwrap_or(""));
        if value == held.token && expiry == held.expires_at.to_string() {
            writer.write_all(b"ok\n").expect("reply");
        } else {
            writer.write_all(b"reject\n").expect("reply");
        }
    }
}

struct Holder {
    child: std::process::Child,
    reader: BufReader<UnixStream>,
    writer: UnixStream,
}

fn spawn_holder(token_path: &std::path::Path, tag: &str) -> Holder {
    let sock = token_path
        .parent()
        .unwrap()
        .join(format!("holder-{tag}.sock"));
    let _ = std::fs::remove_file(&sock);
    let listener = UnixListener::bind(&sock).expect("bind");
    listener.set_nonblocking(true).expect("nonblocking");
    let exe = std::env::current_exe().expect("current exe");
    let child = Command::new(exe)
        .arg("--exact")
        .arg("live_holder_rejects_rotated_token_and_restart_recovers")
        .arg("--nocapture")
        .env(HOLDER_ENV, "1")
        .env(HOLDER_SOCK_ENV, &sock)
        .env(HOLDER_TOKEN_ENV, token_path)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
        .expect("spawn holder");
    let deadline = Instant::now() + Duration::from_secs(30);
    let stream = loop {
        match listener.accept() {
            Ok((stream, _)) => break stream,
            Err(_) if Instant::now() < deadline => std::thread::sleep(Duration::from_millis(20)),
            Err(e) => panic!("holder never connected: {e}"),
        }
    };
    // Accepted sockets inherit the listener's nonblocking mode; the
    // probe protocol below needs blocking reads.
    stream.set_nonblocking(false).expect("blocking");
    let stream_clone = stream.try_clone().expect("clone");
    let mut reader = BufReader::new(stream_clone);
    let mut line = String::new();
    reader.read_line(&mut line).expect("ready");
    assert_eq!(line.trim_end(), "ready");
    Holder {
        child,
        reader,
        writer: stream,
    }
}

impl Holder {
    fn present(&mut self, value: &str, expiry: i64) -> String {
        writeln!(self.writer, "{value} {expiry}").expect("probe");
        let mut line = String::new();
        self.reader.read_line(&mut line).expect("reply");
        line.trim_end().to_string()
    }

    fn stop(mut self) {
        let _ = writeln!(self.writer, "quit");
        let _ = self.child.wait();
    }
}

#[test]
fn live_holder_rejects_rotated_token_and_restart_recovers() {
    if std::env::var(HOLDER_ENV).is_ok() {
        run_holder_child();
    }
    let dir = scratch("live-holder");
    let path = dir.join("token.json");
    let uid = euid();
    let op = format!("a{}", "f".repeat(24));
    let old = token(&op, &"3".repeat(64), 1_000);
    publish_setup_file(&path, &old, uid).expect("initial publish");

    // The live holder captured the old token at (process) start.
    let mut holder = spawn_holder(&path, "first");
    assert_eq!(holder.present(&"3".repeat(64), 1_000), "ok");

    // Renewal rotates the file atomically while the holder lives: the
    // holder cannot see the new credential — the hazard that forces the
    // host to stop the retained generation before swapping.
    let fresh = token(&op, &"4".repeat(64), 2_800);
    publish_setup_file(&path, &fresh, uid).expect("atomic rotation");
    assert_eq!(read_token(&path).token, "4".repeat(64));
    assert_eq!(holder.present(&"4".repeat(64), 2_800), "reject");
    assert_eq!(holder.present(&"3".repeat(64), 1_000), "ok");
    holder.stop();

    // Restarted, the replacement process captures the fresh credential at
    // start: recovery without losing identity, address, or data.
    let mut holder = spawn_holder(&path, "second");
    assert_eq!(holder.present(&"4".repeat(64), 2_800), "ok");
    assert_eq!(holder.present(&"3".repeat(64), 1_000), "reject");
    holder.stop();

    std::fs::remove_dir_all(&dir).expect("cleanup");
}

/// FJ-847: descriptor-relative fixture anchors for the privileged test
/// helpers. The parent is opened without following into an `OwnedFd`; mkdir,
/// chown, staging, publication, and removal all resolve through that
/// descriptor with `O_NOFOLLOW` / `AT_SYMLINK_NOFOLLOW`. A symlink swapped
/// into any path after validation cannot redirect the mutation: the anchored
/// call fails closed on the descriptor.
fn anchor_leaf(name: &std::ffi::OsStr) -> CString {
    CString::new(name.as_encoded_bytes()).expect("FJ-847: leaf name")
}

/// FJ-855: walk the parent chain from the trust root (`/`) one no-follow
/// `openat` per component while holding the parent descriptor — no
/// full-path opens anywhere in the fixture anchors. The single CoreOS
/// `/home` alias traverses its allowlisted target exactly like the library
/// (`var/home`, `/var/home`); any other symlink on the chain fails closed
/// at the `openat`. Returns the walked parent descriptor plus the final
/// leaf name; the caller pins the leaf itself through the parent.
fn anchor_parent_fd(path: &std::path::Path) -> (OwnedFd, CString) {
    assert!(
        path.is_absolute(),
        "FJ-855: anchor path must be absolute: {}",
        path.display()
    );
    let mut comps: Vec<std::ffi::OsString> = Vec::new();
    for part in path.components() {
        match part {
            std::path::Component::RootDir => {}
            std::path::Component::Normal(name) => comps.push(name.to_owned()),
            _ => panic!(
                "FJ-855: anchor path must use only normal components: {}",
                path.display()
            ),
        }
    }
    // CoreOS `/home` alias, same allowlist as the library walk.
    if comps
        .first()
        .is_some_and(|first| first.as_encoded_bytes() == b"home")
    {
        match std::fs::read_link(std::path::Path::new("/home")) {
            Ok(target)
                if target == std::path::Path::new("var/home")
                    || target == std::path::Path::new("/var/home") =>
            {
                let mut resolved = vec![
                    std::ffi::OsString::from("var"),
                    std::ffi::OsString::from("home"),
                ];
                resolved.extend(comps.into_iter().skip(1));
                comps = resolved;
            }
            Ok(_) => panic!("FJ-855: anchor escapes via a non-allowlisted alias"),
            Err(_) => {
                if std::fs::symlink_metadata("/home")
                    .map(|meta| meta.file_type().is_symlink())
                    .unwrap_or(false)
                {
                    panic!("FJ-855: anchor escapes via a non-allowlisted alias");
                }
            }
        }
    }
    let tail = comps.pop().expect("FJ-855: anchor path has no leaf");
    let root = CString::new("/").expect("FJ-855: trust root");
    let root_fd = unsafe {
        libc::open(
            root.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
        )
    };
    assert!(root_fd >= 0, "FJ-855: trust root unavailable");
    let mut parent: OwnedFd = unsafe { OwnedFd::from_raw_fd(root_fd) };
    for comp in &comps {
        let leaf = anchor_leaf(comp);
        let fd = unsafe {
            libc::openat(
                parent.as_raw_fd(),
                leaf.as_ptr(),
                libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            )
        };
        assert!(
            fd >= 0,
            "FJ-855: anchor chain left its descriptor above {}: {}",
            std::path::Path::new(comp).display(),
            path.display()
        );
        parent = unsafe { OwnedFd::from_raw_fd(fd) };
    }
    (parent, anchor_leaf(&tail))
}

fn anchor_fd(path: &std::path::Path) -> OwnedFd {
    // FJ-855: the leaf itself is pinned through the walked parent
    // descriptor — a symlink swapped in at the leaf (or anywhere above)
    // fails here instead of redirecting the anchor.
    let (parent, leaf) = anchor_parent_fd(path);
    let fd = unsafe {
        libc::openat(
            parent.as_raw_fd(),
            leaf.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
        )
    };
    assert!(fd >= 0, "FJ-855: anchor unavailable: {}", path.display());
    let mut stat: libc::stat = unsafe { std::mem::zeroed() };
    assert!(
        unsafe { libc::fstat(fd, &mut stat) } == 0
            && (stat.st_mode as u32 & libc::S_IFMT) == libc::S_IFDIR,
        "FJ-855: anchor is not a directory: {}",
        path.display()
    );
    unsafe { OwnedFd::from_raw_fd(fd) }
}

fn anchor_child_dir(parent: &OwnedFd, name: &std::ffi::OsStr) -> OwnedFd {
    let leaf = anchor_leaf(name);
    let fd = unsafe {
        libc::openat(
            parent.as_raw_fd(),
            leaf.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
        )
    };
    assert!(fd >= 0, "FJ-847: anchored child unavailable");
    unsafe { OwnedFd::from_raw_fd(fd) }
}

fn anchor_child_any(parent: &OwnedFd, name: &std::ffi::OsStr) -> OwnedFd {
    let leaf = anchor_leaf(name);
    let fd = unsafe {
        libc::openat(
            parent.as_raw_fd(),
            leaf.as_ptr(),
            libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
        )
    };
    assert!(fd >= 0, "FJ-847: anchored child unavailable");
    unsafe { OwnedFd::from_raw_fd(fd) }
}

/// FJ-847: `fchownat` with `AT_SYMLINK_NOFOLLOW` through the validated
/// parent descriptor — a symlink at `path` is never followed — then the
/// opened descriptor is re-validated for owner and non-symlink identity.
fn chown_numbered(path: &std::path::Path, uid: u32) {
    let parent = path.parent().expect("FJ-847: chown target has a parent");
    let name = path.file_name().expect("FJ-847: chown target has a name");
    let before = std::fs::symlink_metadata(path).expect("FJ-847: chown target must exist");
    assert!(
        !before.file_type().is_symlink(),
        "FJ-847: chown refuses a symlink: {}",
        path.display()
    );
    let parent_fd = anchor_fd(parent);
    let leaf = anchor_leaf(name);
    assert!(
        unsafe {
            libc::fchownat(
                parent_fd.as_raw_fd(),
                leaf.as_ptr(),
                uid,
                uid,
                libc::AT_SYMLINK_NOFOLLOW,
            )
        } == 0,
        "FJ-847: anchored chown {}",
        path.display()
    );
    let fd = anchor_child_any(&parent_fd, name);
    let mut stat: libc::stat = unsafe { std::mem::zeroed() };
    assert!(
        unsafe { libc::fstat(fd.as_raw_fd(), &mut stat) } == 0 && stat.st_uid == uid,
        "FJ-847: chown target changed under us: {}",
        path.display()
    );
}

fn write_private(path: &std::path::Path, bytes: &[u8], uid: u32) {
    // FJ-847: the parent is proven our own canonical private tree first,
    // then pinned with a no-follow descriptor; staging, ownership, and
    // publication all resolve through it. A redirected parent refuses
    // instead of collecting the secret. Both secrets live directly under
    // the dir owned by the same uid (journal under operator-owned `setups`,
    // token under the runtime-owned control dir).
    let parent = path.parent().expect("FJ-847: secret has a parent");
    let name = path.file_name().expect("FJ-847: secret has a name");
    assert_owned_private_dir(parent, uid);
    let parent_fd = anchor_fd(parent);
    let mut nonce = [0u8; 8];
    getrandom::fill(&mut nonce).expect("FJ-847: nonce");
    let tmp = std::ffi::OsString::from(format!(
        ".{}.{}.pending",
        name.to_string_lossy(),
        hex_lower(&nonce)
    ));
    let tmp_leaf = anchor_leaf(&tmp);
    let fd = unsafe {
        libc::openat(
            parent_fd.as_raw_fd(),
            tmp_leaf.as_ptr(),
            libc::O_WRONLY | libc::O_CREAT | libc::O_EXCL | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0o600,
        )
    };
    assert!(fd >= 0, "FJ-847: anchored stage {}", path.display());
    assert!(
        unsafe { libc::fchown(fd, uid, uid) } == 0,
        "FJ-847: anchored own {}",
        path.display()
    );
    let mut written = 0;
    while written < bytes.len() {
        let step = unsafe {
            libc::write(
                fd,
                bytes[written..].as_ptr() as *const libc::c_void,
                bytes.len() - written,
            )
        };
        assert!(step > 0, "FJ-847: anchored write {}", path.display());
        written += step as usize;
    }
    assert!(
        unsafe { libc::fsync(fd) } == 0,
        "FJ-847: anchored sync {}",
        path.display()
    );
    unsafe { libc::close(fd) };
    let leaf = anchor_leaf(name);
    assert!(
        unsafe {
            libc::renameat(
                parent_fd.as_raw_fd(),
                tmp_leaf.as_ptr(),
                parent_fd.as_raw_fd(),
                leaf.as_ptr(),
            )
        } == 0,
        "FJ-847: anchored publish {}",
        path.display()
    );
    assert!(
        unsafe { libc::fsync(parent_fd.as_raw_fd()) } == 0,
        "FJ-847: anchored dir sync {}",
        path.display()
    );
    chown_numbered(path, uid);
    let meta = std::fs::symlink_metadata(path).expect("FJ-847: secret must exist");
    assert!(
        meta.is_file() && !meta.file_type().is_symlink() && meta.uid() == uid,
        "FJ-847: secret changed under us: {}",
        path.display()
    );
}

fn make_private_dir(path: &std::path::Path, uid: u32) {
    // FJ-847: creation goes through the validated parent descriptor with
    // mkdirat; only a directory this call just created is assigned its
    // owner via fchownat (never following), then the opened descriptor is
    // re-validated. An existing tree is re-validated and never re-owned.
    let parent = path.parent().expect("FJ-847: dir has a parent");
    let name = path.file_name().expect("FJ-847: dir has a name");
    let parent_fd = anchor_fd(parent);
    let leaf = anchor_leaf(name);
    let mut created = false;
    if unsafe {
        libc::openat(
            parent_fd.as_raw_fd(),
            leaf.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
        )
    } < 0
    {
        assert!(
            unsafe { libc::mkdirat(parent_fd.as_raw_fd(), leaf.as_ptr(), 0o700) } == 0,
            "FJ-847: anchored mkdir {}",
            path.display()
        );
        created = true;
    }
    if created {
        assert!(
            unsafe {
                libc::fchownat(
                    parent_fd.as_raw_fd(),
                    leaf.as_ptr(),
                    uid,
                    uid,
                    libc::AT_SYMLINK_NOFOLLOW,
                )
            } == 0,
            "FJ-847: anchored chown {}",
            path.display()
        );
    }
    let fd = anchor_child_dir(&parent_fd, name);
    assert!(
        unsafe { libc::fchmod(fd.as_raw_fd(), 0o700) } == 0,
        "FJ-847: anchored mode {}",
        path.display()
    );
    let mut stat: libc::stat = unsafe { std::mem::zeroed() };
    assert!(
        unsafe { libc::fstat(fd.as_raw_fd(), &mut stat) } == 0
            && stat.st_uid == uid
            && (stat.st_mode as u32 & 0o077) == 0,
        "FJ-847: anchored dir changed under us: {}",
        path.display()
    );
    // Canonical-path parity with the library guards.
    assert_owned_private_dir(path, uid);
}

/// FJ-847/FJ-855: guarded cleanup. The tree is re-validated by the canonical
/// guards, pinned through the descriptor walk, bound by dev/ino, then every
/// child is removed through its HELD parent descriptor (`unlinkat` per name,
/// recursing with the child descriptor carried down) and the emptied root is
/// dropped through its walked parent anchor. FJ-855: path strings are never
/// re-resolved through the descent — the listing itself comes from the
/// kernel's view of the open directory (`/proc/self/fd`), subdirectories
/// are descended via `openat` from the held parent, and symlinks are removed
/// as links (their targets untouched). A swapped tree refuses instead of
/// deleting foreign bytes. All fixtures here are private single-owner trees,
/// so no third party can rename entries between the listing and the anchored
/// removal.
fn remove_tree_anchored(path: &std::path::Path) {
    assert!(path.is_absolute(), "FJ-855: cleanup path must be absolute");
    // Walk the parent, then pin the target itself through the walked
    // descriptor: a root swapped for a symlink (or renamed away) fails here
    // instead of deleting foreign bytes.
    let (parent_fd, leaf) = anchor_parent_fd(path);
    let raw = unsafe {
        libc::openat(
            parent_fd.as_raw_fd(),
            leaf.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
        )
    };
    assert!(
        raw >= 0,
        "FJ-855: cleanup path left its anchor: {}",
        path.display()
    );
    let anchor: OwnedFd = unsafe { OwnedFd::from_raw_fd(raw) };
    let expect = std::fs::symlink_metadata(path).expect("FJ-855: cleanup target must exist");
    let mut stat: libc::stat = unsafe { std::mem::zeroed() };
    assert!(unsafe { libc::fstat(anchor.as_raw_fd(), &mut stat) } == 0);
    assert!(
        expect.dev() == stat.st_dev && expect.ino() == stat.st_ino,
        "FJ-855: cleanup path left its anchor: {}",
        path.display()
    );
    remove_tree_fd(&anchor, path);
    assert!(
        unsafe { libc::unlinkat(parent_fd.as_raw_fd(), leaf.as_ptr(), libc::AT_REMOVEDIR) } == 0,
        "FJ-855: anchored rmdir {}",
        path.display()
    );
}

/// Recursive cleanup with the directory descriptor carried through the
/// descent: the listing is the kernel's view of the held directory, each
/// subdirectory is opened from its held parent and removed through it, and
/// symlinks are unlinked as links — never followed, never re-resolved.
fn remove_tree_fd(dir: &OwnedFd, display: &std::path::Path) {
    let listing = std::fs::read_dir(format!("/proc/self/fd/{}", dir.as_raw_fd()))
        .expect("FJ-855: cleanup scan");
    for entry in listing {
        let entry = entry.expect("FJ-855: cleanup entry");
        let name = entry.file_name();
        let kind = entry.file_type().expect("FJ-855: cleanup kind");
        let leaf = anchor_leaf(&name);
        if kind.is_dir() && !kind.is_symlink() {
            let raw = unsafe {
                libc::openat(
                    dir.as_raw_fd(),
                    leaf.as_ptr(),
                    libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
                )
            };
            assert!(
                raw >= 0,
                "FJ-855: anchored descend {}",
                display.join(&name).display()
            );
            let child: OwnedFd = unsafe { OwnedFd::from_raw_fd(raw) };
            remove_tree_fd(&child, &display.join(&name));
            assert!(
                unsafe { libc::unlinkat(dir.as_raw_fd(), leaf.as_ptr(), libc::AT_REMOVEDIR) } == 0,
                "FJ-855: anchored rmdir {}",
                display.join(&name).display()
            );
        } else {
            assert!(
                unsafe { libc::unlinkat(dir.as_raw_fd(), leaf.as_ptr(), 0) } == 0,
                "FJ-855: anchored unlink {}",
                display.join(&name).display()
            );
        }
    }
}

fn hex_lower(bytes: &[u8]) -> String {
    bytes.iter().map(|b| format!("{b:02x}")).collect()
}

/// FJ-837: process-global setup variables are captured BEFORE mutation and
/// restored on drop — success and panic paths alike — so a failure cannot
/// leak `SODA_APPLICATIONS_ROOT`, `PATH`, or the podman log pointer into
/// sibling tests sharing this harness process.
struct RestoreEnv {
    key: &'static str,
    prior: Option<std::ffi::OsString>,
}
impl RestoreEnv {
    fn capture(key: &'static str) -> Self {
        Self {
            key,
            prior: std::env::var_os(key),
        }
    }
}
impl Drop for RestoreEnv {
    fn drop(&mut self) {
        match &self.prior {
            Some(value) => std::env::set_var(self.key, value),
            None => std::env::remove_var(self.key),
        }
    }
}

/// FJ-837 canonical fixture guard, hardened per FJ-841. The pre-835 lexical
/// `starts_with` check could not see through an ancestor symlink, and the old
/// cwd-relative scratch followed the caller — so a hostile or stale tree
/// could redirect creation/chown/removal outside the workspace BEFORE the
/// production resolver ever ran. This validates the REAL path first and
/// refuses before ANY filesystem mutation:
///
/// * both paths are absolute, the anchor itself is already canonical
///   (`canonicalize == self`, pinning the whole chain above it), and the
///   fixture carries only normal components below the anchor (no `..`);
/// * EVERY existing ancestor from the fixture up to AND INCLUDING the anchor
///   is a real directory, never a symlink, never group/other-writable, and
///   canonicalizes to itself (FJ-841: the 837 loop stopped at the FIRST
///   existing ancestor, so a group-writable or swapped HIGHER ancestor above
///   a clean nearest one escaped — a clean nearest `canonicalize == self`
///   only proves no symlink above it, never the mode bits up the chain);
/// * the resolved fixture stays inside the canonical anchor tree.
///
/// The lexical guard above still runs first and refuses before ANY
/// filesystem mutation. Per FJ-847 every privileged mutation and the final
/// cleanup below additionally resolve through validated directory
/// descriptors (`anchor_fd` / `openat` family with `O_NOFOLLOW` /
/// `AT_SYMLINK_NOFOLLOW`), so a symlink swapped into a path after the guard
/// fails closed on the descriptor instead of acting on foreign bytes
/// (fail-closed refusal is the protection — no adversary here, only
/// misconfiguration or a stale tree).
fn assert_canonical_fixture_dir(path: &std::path::Path, anchor: &std::path::Path) {
    assert!(
        path.is_absolute(),
        "FJ-837: fixture path must be absolute: {}",
        path.display()
    );
    assert!(
        anchor.is_absolute(),
        "FJ-837: fixture anchor must be absolute: {}",
        anchor.display()
    );
    let canonical_anchor =
        std::fs::canonicalize(anchor).expect("FJ-837: fixture anchor must exist and resolve");
    assert_eq!(
        canonical_anchor,
        anchor,
        "FJ-837: fixture anchor escapes via symlink: {}",
        anchor.display()
    );
    let rest = path
        .strip_prefix(anchor)
        .expect("FJ-837: fixture outside the anchor tree");
    assert!(
        !rest.as_os_str().is_empty()
            && rest
                .components()
                .all(|part| matches!(part, std::path::Component::Normal(_))),
        "FJ-837: fixture path must use only normal components below the anchor: {}",
        path.display()
    );
    // FJ-841: walk EVERY ancestor from the fixture up to and including the
    // anchor — never stop at the first existing one. Missing components are
    // skipped (they will be created under a validated parent); each existing
    // one must be a real non-symlink directory with no group/other-writable
    // bit that canonicalizes to itself.
    let mut cursor: &std::path::Path = path;
    loop {
        match std::fs::symlink_metadata(cursor) {
            Ok(meta) => {
                assert!(
                    meta.is_dir() && !meta.file_type().is_symlink(),
                    "FJ-841: fixture ancestor is not a real directory: {}",
                    cursor.display()
                );
                assert_eq!(
                    meta.mode() & 0o022,
                    0,
                    "FJ-841: fixture ancestor is group/other-writable: {}",
                    cursor.display()
                );
                let canonical =
                    std::fs::canonicalize(cursor).expect("FJ-841: fixture ancestor must resolve");
                assert_eq!(
                    canonical,
                    cursor,
                    "FJ-841: fixture ancestor escapes via symlink: {}",
                    cursor.display()
                );
            }
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
            Err(_) => panic!("FJ-841: fixture ancestor unavailable: {}", cursor.display()),
        }
        if cursor == anchor {
            break;
        }
        cursor = cursor
            .parent()
            .expect("FJ-841: fixture escapes the filesystem root");
    }
    assert!(
        path.starts_with(&canonical_anchor),
        "FJ-837: resolved fixture left the anchor tree: {}",
        path.display()
    );
}

/// FJ-837: directories the test itself owns must be canonical private
/// trees (expected uid, 0700) — checked AFTER creation but BEFORE any secret
/// is written, and (for a stale pre-existing tree) BEFORE any removal.
/// Anything else is refused, never removed blind. FJ-841: the uid is a
/// parameter so the runtime-owned control dir (10000) gets the same
/// symlink/TOCTOU re-validation at use as the operator-owned dirs.
fn assert_owned_private_dir(path: &std::path::Path, uid: u32) {
    let meta = std::fs::symlink_metadata(path).expect("FJ-841: owned fixture dir must exist");
    assert!(
        meta.is_dir() && !meta.file_type().is_symlink(),
        "FJ-841: owned fixture dir is not a real directory: {}",
        path.display()
    );
    assert_eq!(
        meta.uid(),
        uid,
        "FJ-841: owned fixture dir has the wrong owner: {}",
        path.display()
    );
    assert_eq!(
        meta.mode() & 0o777,
        0o700,
        "FJ-841: owned fixture dir must be private: {}",
        path.display()
    );
    assert_eq!(
        std::fs::canonicalize(path).expect("FJ-841: owned fixture dir must resolve"),
        path,
        "FJ-841: owned fixture dir escapes via symlink: {}",
        path.display()
    );
}

fn assert_private_root_dir(path: &std::path::Path) {
    assert_owned_private_dir(path, 0);
}

/// FJ-841: meaningful path negatives with NO outside-root writes. Runs as any
/// user (no euid gate): every unsafe shape is refused by the canonical guard
/// before any mutation, and the one safe shape is granted. All fixtures live
/// under this package's `target/` scratch and are removed afterward; nothing
/// outside the anchor is created, chowned, or removed. The root-only proof
/// below reuses the same guards at each real mutation.
#[test]
fn fixture_guard_refuses_unsafe_paths_without_mutation() {
    use std::os::unix::fs::PermissionsExt;
    let anchor = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join(format!("target/fj841-negatives-{}", std::process::id()));
    std::fs::create_dir_all(&anchor).expect("negatives anchor");
    let refused = |label: &str, path: std::path::PathBuf| {
        assert!(
            std::panic::catch_unwind(|| assert_canonical_fixture_dir(&path, &anchor)).is_err(),
            "FJ-841: unsafe shape was granted: {label}"
        )
    };
    // Safe control: missing leaf under a clean chain is granted.
    assert_canonical_fixture_dir(&anchor.join("ok/fixture"), &anchor);
    // Lexical escapes: relative, `..`, and outside-anchor paths.
    refused(
        "relative",
        std::path::PathBuf::from("target/fj841-negatives-relative"),
    );
    refused("dotdot", anchor.join("ok/../evil"));
    refused(
        "outside-anchor",
        anchor.with_extension("other").join("fixture"),
    );
    // Symlink ancestor redirects the whole subtree.
    let real = anchor.join("real");
    std::fs::create_dir_all(&real).expect("real dir");
    std::os::unix::fs::symlink(&real, anchor.join("link")).expect("link");
    refused("symlink-ancestor", anchor.join("link/fixture"));
    // FJ-841 discriminator: the nearest existing ancestor is clean, but a
    // HIGHER one is group-writable — the 837 first-existing loop granted
    // this shape; the every-ancestor walk refuses it.
    let writable = anchor.join("writable");
    let clean = writable.join("clean");
    std::fs::create_dir_all(&clean).expect("nested dirs");
    std::fs::set_permissions(&writable, std::fs::Permissions::from_mode(0o775))
        .expect("writable mode");
    refused("writable-higher-ancestor", clean.join("fixture"));
    std::fs::set_permissions(&writable, std::fs::Permissions::from_mode(0o755))
        .expect("restore mode");
    // Owned-dir negatives: wrong owner, over-permissive mode, symlink —
    // all refused before any chown/removal could act on them.
    let foreign = anchor.join("foreign");
    std::fs::create_dir_all(&foreign).expect("foreign dir");
    std::fs::set_permissions(&foreign, std::fs::Permissions::from_mode(0o700))
        .expect("foreign mode");
    // Expected-owner refusal in both modes: non-root never owns as uid 0,
    // root never owns as 60000 — either way the uid gate must fire.
    let expected_owner: u32 = if euid() == 0 { 60000 } else { 0 };
    assert!(
        std::panic::catch_unwind(|| assert_owned_private_dir(&foreign, expected_owner)).is_err(),
        "FJ-841: wrong-owner dir was granted"
    );
    assert!(
        std::panic::catch_unwind(|| assert_owned_private_dir(&foreign, 60000)).is_err(),
        "FJ-841: wrong-owner dir was granted"
    );
    let loose = anchor.join("loose");
    std::fs::create_dir_all(&loose).expect("loose dir");
    std::fs::set_permissions(&loose, std::fs::Permissions::from_mode(0o755)).expect("loose mode");
    assert!(
        std::panic::catch_unwind(|| assert_owned_private_dir(&loose, euid())).is_err(),
        "FJ-841: over-permissive dir was granted"
    );
    assert!(
        std::panic::catch_unwind(|| assert_owned_private_dir(&anchor.join("link"), euid()))
            .is_err(),
        "FJ-841: symlinked dir was granted"
    );
    remove_tree_anchored(&anchor);
    assert!(!anchor.exists(), "negatives scratch survived");
}

/// FJ-847: the public atomic-publish surface is descriptor-relative. A
/// symlink swapped into the parent path after a successful publish cannot
/// divert the next write: staging fails closed on the descriptor, the link
/// target stays untouched, and no pending residue leaks. Restoring the
/// directory makes publication work again with the old bytes undisturbed.
/// Runs as any user (no euid gate) on the package `target/` scratch;
/// nothing outside the anchor is created, written, or removed.
#[test]
fn anchored_publish_refuses_symlink_swap_without_write_through() {
    let anchor = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join(format!("target/fj847-negatives-{}", std::process::id()));
    std::fs::create_dir_all(&anchor).expect("negatives anchor");
    let real = anchor.join("real");
    let evil = anchor.join("evil");
    std::fs::create_dir_all(&evil).expect("evil dir");
    std::fs::write(evil.join("sentinel"), b"evil-sentinel").expect("sentinel");
    make_private_dir(&real, euid());
    let uid = euid();
    let path = real.join("role.json");
    let old = token("op1", &"1".repeat(64), 1_000);
    publish_setup_file(&path, &old, uid).expect("initial publish");
    assert_eq!(read_token(&path).token, "1".repeat(64));
    // Swap the validated directory for a symlink to the evil tree AFTER the
    // successful publish above: the next publish must fail closed.
    let saved = anchor.join("real-saved");
    std::fs::rename(&real, &saved).expect("swap aside");
    std::os::unix::fs::symlink(&evil, &real).expect("swap link");
    let fresh = token("op1", &"2".repeat(64), 2_800);
    let err = publish_setup_file(&path, &fresh, uid).expect_err("swapped parent must refuse");
    assert!(err.contains("cannot be staged"), "unexpected: {err}");
    // No write-through: the link target holds only its sentinel, and the
    // refused staging left no pending residue in the saved tree either.
    let mut evil_entries: Vec<_> = std::fs::read_dir(&evil)
        .expect("evil scan")
        .map(|entry| entry.expect("evil entry").file_name())
        .collect();
    evil_entries.sort();
    assert_eq!(
        evil_entries,
        vec![std::ffi::OsString::from("sentinel")],
        "write-through past the swap: {evil_entries:?}"
    );
    let mut saved_entries: Vec<_> = std::fs::read_dir(&saved)
        .expect("saved scan")
        .map(|entry| entry.expect("saved entry").file_name())
        .collect();
    saved_entries.sort();
    assert_eq!(
        saved_entries,
        vec![std::ffi::OsString::from("role.json")],
        "refused staging leaked residue: {saved_entries:?}"
    );
    // Restore the directory: publication works again and the refused attempt
    // never disturbed the old bytes.
    std::fs::remove_file(&real).expect("unlink swap");
    std::fs::rename(&saved, &real).expect("restore");
    assert_eq!(read_token(&path).token, "1".repeat(64));
    publish_setup_file(&path, &fresh, uid).expect("publish after restore");
    assert_eq!(read_token(&path).token, "2".repeat(64));
    remove_tree_anchored(&anchor);
    assert!(!anchor.exists(), "negatives scratch survived");
}

/// FJ-862: the packaged reconcile's typed outcomes parse through the fixed
/// allowlist only. The exact `ok:true` success shape bound to this exact
/// generation/operation/app context maps to `Committed`; everything else —
/// garbage, empty, wrong shapes, unbound receipts, extra fields, trailing
/// non-whitespace — is `Uncertain`. A trailing newline (what the CLI
/// prints) is accepted. Unclaimed and refusal outcomes never arrive here:
/// they are nonzero exits classified from the failure diagnostic (see the
/// next test), never parsed from stdout.
fn reconcile_expectation() -> ReconcileExpectation {
    ReconcileExpectation {
        operation: format!("a{}", "d".repeat(24)),
        app: "example".into(),
        artifact_digest: "2".repeat(64),
        bundle_digest: "3".repeat(64),
        storage_fingerprint: "7".repeat(64),
    }
}

fn reconcile_success(operation: &str, app: &str, storage: &str) -> Vec<u8> {
    format!(
        "{{\"ok\":true,\"command\":\"reconcile\",\"schema\":\"canlang.soda.setup.v1\",\
        \"operation\":{operation:?},\"app\":{app:?},\"artifactDigest\":\"{a}\",\
        \"bundleDigest\":\"{b}\",\"storageFingerprint\":{storage:?},\
        \"ownerTeam\":\"12345678-1234-1234-1234-123456789abc\",\"completedAt\":1793118443}}",
        a = "2".repeat(64),
        b = "3".repeat(64),
    )
    .into_bytes()
}

#[test]
fn reconcile_outcome_parsing_is_allowlisted() {
    let expected = reconcile_expectation();
    let mut bound = reconcile_success(
        &expected.operation,
        &expected.app,
        &expected.storage_fingerprint,
    );
    assert_eq!(
        parse_reconcile_outcome(&bound, &expected),
        ReconcileOutcome::Committed
    );
    bound.push(b'\n');
    assert_eq!(
        parse_reconcile_outcome(&bound, &expected),
        ReconcileOutcome::Committed,
        "the CLI's trailing newline must be accepted"
    );
    // A success that names anything else is a confused output, never a
    // permit: wrong operation/app/digests/storage all refuse.
    for (label, stdout) in [
        (
            "operation",
            reconcile_success("a9", &expected.app, &expected.storage_fingerprint),
        ),
        (
            "app",
            reconcile_success(&expected.operation, "other", &expected.storage_fingerprint),
        ),
        (
            "storage",
            reconcile_success(&expected.operation, &expected.app, &"8".repeat(64)),
        ),
    ] {
        assert_eq!(
            parse_reconcile_outcome(&stdout, &expected),
            ReconcileOutcome::Uncertain,
            "unbound success was trusted: {label}"
        );
    }
    let uncertain: &[&[u8]] = &[
        b"",
        b"\n",
        b"{}",
        b"{\"ok\":true}",
        b"{\"ok\":true,\"command\":\"reconcile\"}",
        b"{\"ok\":false,\"command\":\"reconcile\"}",
        b"{\"ok\":true,\"command\":\"setup\"}",
        b"{\"ok\":true,\"command\":\"reconcile\",\"schema\":\"canlang.soda.setup.v1\"}",
        b"{\"claim\":\"unclaimed\"}",
        b"{\"claim\":\"committed\"}",
        b"{\"claim\":\"conflict\"}",
        b"unclaimed",
        b"[]",
        b"\x00\x01",
        b"setup claim is committed",
    ];
    for bad in uncertain {
        assert_eq!(
            parse_reconcile_outcome(bad, &expected),
            ReconcileOutcome::Uncertain,
            "non-allowlisted stdout was trusted: {bad:?}"
        );
    }
    // Wrong types, empty owner, non-positive completion, and extra fields
    // all refuse even when the binding fields match.
    let mut mutated = bound.clone();
    mutated.extend_from_slice(br#"{"extra":1}"#);
    assert_eq!(
        parse_reconcile_outcome(&mutated, &expected),
        ReconcileOutcome::Uncertain
    );
    let no_owner = reconcile_success(
        &expected.operation,
        &expected.app,
        &expected.storage_fingerprint,
    );
    let no_owner = String::from_utf8(no_owner)
        .unwrap()
        .replace("12345678-1234-1234-1234-123456789abc", "");
    assert_eq!(
        parse_reconcile_outcome(no_owner.as_bytes(), &expected),
        ReconcileOutcome::Uncertain
    );
    let zero_completed = reconcile_success(
        &expected.operation,
        &expected.app,
        &expected.storage_fingerprint,
    );
    let zero_completed = String::from_utf8(zero_completed)
        .unwrap()
        .replace("1793118443", "0");
    assert_eq!(
        parse_reconcile_outcome(zero_completed.as_bytes(), &expected),
        ReconcileOutcome::Uncertain
    );
}

/// FJ-862: the packaged CLI's two sanitized nonzero lines classify through
/// the fixed allowlist only. The match is suffix-anchored on the exact
/// literals at the end of the executor diagnostic: podman wrapper text may
/// precede them, but free-form text after them (or anywhere else) never
/// matches and never surfaces.
#[test]
fn reconcile_failure_classification_is_allowlisted() {
    let unclaimed = format!("podman failed: exit status 1: {RECONCILE_UNCLAIMED_LINE}\n");
    assert_eq!(
        classify_reconcile_failure(&unclaimed),
        ReconcileRefusal::Unclaimed
    );
    assert_eq!(
        classify_reconcile_failure(&unclaimed.trim_end()),
        ReconcileRefusal::Unclaimed
    );
    let refused = format!("podman failed: exit status 1: {RECONCILE_REFUSAL_LINE}\n");
    assert_eq!(
        classify_reconcile_failure(&refused),
        ReconcileRefusal::Refused
    );
    // Podman wrapper variants keep the literal last: still classified.
    assert_eq!(
        classify_reconcile_failure(&format!(
            "podman failed: exit status 125: {RECONCILE_UNCLAIMED_LINE}"
        )),
        ReconcileRefusal::Unclaimed
    );
    let unknown = [
        "podman failed: exit status 1: ",
        "podman failed: exit status 1: can-soda: Setup reconciliation refused. trailing junk",
        "podman failed: exit status 1: can-soda: setup reconciliation refused.",
        "podman failed: exit status 1: Setup reconciliation refused.",
        "podman failed: signal: killed",
        "podman failed: deadline exceeded",
        "podman failed: output capture limit exceeded",
        "",
    ];
    for bad in unknown {
        assert_eq!(
            classify_reconcile_failure(bad),
            ReconcileRefusal::Unknown,
            "non-allowlisted diagnostic classified: {bad:?}"
        );
    }
}

/// FJ-855: the rotation gate permits only the proven-unclaimed outcome.
/// Committed (expiry never uncommits), conflict, and uncertain keep the old
/// token — the original credential is never destroyed on unknown/conflict.
#[test]
fn reconcile_gate_permits_only_proven_unclaimed() {
    assert!(reconcile_permits_rotation(ReconcileOutcome::Unclaimed));
    assert!(!reconcile_permits_rotation(ReconcileOutcome::Committed));
    assert!(!reconcile_permits_rotation(ReconcileOutcome::Conflict));
    assert!(!reconcile_permits_rotation(ReconcileOutcome::Uncertain));
}

/// FJ-874: the post-reconcile fence distinguishes RUNNING from
/// stopped/exited holders. The re-check runs default `podman ps`
/// (running containers only): a running holder is listed and still
/// fences, while a correctly stopped retained container is absent (blank
/// output — the branch the fence clears on) and must NOT refuse an
/// unclaimed rotation. This pins the load-bearing platform behavior with
/// real holder fixtures under setup-holder names, executing the fence's
/// exact `ps` argv (no `--all`). Runs as any user with rootless podman;
/// skips cleanly when podman or a fixture image is unavailable. Unique
/// names per process; every created container is force-removed on every
/// exit path.
#[test]
fn fence_ps_lists_running_holder_only() {
    let pid = std::process::id();
    let running = format!("soda-canlang-setup-fj874-{pid}-running");
    let stopped = format!("soda-canlang-setup-fj874-{pid}-stopped");
    struct RemoveOnDrop(Vec<String>);
    impl Drop for RemoveOnDrop {
        fn drop(&mut self) {
            for name in &self.0 {
                let _ = Command::new("podman").args(["rm", "-f", name]).output();
            }
        }
    }
    let _cleanup = RemoveOnDrop(vec![running.clone(), stopped.clone()]);
    // Best-effort clear of stale fixtures from a killed run.
    for name in [&running, &stopped] {
        let _ = Command::new("podman").args(["rm", "-f", name]).output();
    }
    // Fixture image: first locally present candidate; skip when podman or
    // every candidate is unavailable (no pulls, no network).
    let image = [
        "docker.io/louislam/uptime-kuma:2",
        "ghcr.io/requarks/wiki:2",
    ]
    .into_iter()
    .find(|image| {
        Command::new("podman")
            .args(["image", "exists", image])
            .output()
            .is_ok_and(|out| out.status.success())
    });
    let image = match image {
        Some(image) => image,
        None => {
            eprintln!("SKIP podman-fixture: no local fixture image");
            return;
        }
    };
    // The fence's exact `ps` argv (no `--all`): default lists running only.
    let fence_ps = |name: &str| -> Vec<u8> {
        let out = Command::new("podman")
            .args([
                "ps",
                "--no-trunc",
                &format!("--filter=name=^{name}$"),
                "--format={{.ID}}",
            ])
            .output()
            .expect("podman ps");
        assert!(out.status.success(), "podman ps failed");
        out.stdout
    };
    // Stopped holder fixture: created, never started.
    let create = Command::new("podman")
        .args(["create", "--name", &stopped, image])
        .output()
        .expect("podman create");
    if !create.status.success() {
        eprintln!("SKIP podman-fixture: create refused");
        return;
    }
    // Running holder fixture: started and still up (polled, start races).
    let run = Command::new("podman")
        .args(["run", "-d", "--name", &running, image])
        .output()
        .expect("podman run");
    if !run.status.success() {
        eprintln!("SKIP podman-fixture: run refused");
        return;
    }
    let mut listed = Vec::new();
    for _ in 0..25 {
        listed = fence_ps(&running);
        if !listed.iter().all(u8::is_ascii_whitespace) {
            break;
        }
        std::thread::sleep(std::time::Duration::from_millis(200));
    }
    assert!(
        !listed.iter().all(u8::is_ascii_whitespace),
        "running holder must be listed by the fence ps"
    );
    // A stopped retained container is absent from the running-only `ps`,
    // so the unclaimed rotation must NOT refuse on it.
    let absent = fence_ps(&stopped);
    assert!(
        absent.iter().all(u8::is_ascii_whitespace),
        "stopped holder must be absent from the fence ps"
    );
}

/// FJ-874: the `/application/setup` route carries the three claim outcomes
/// as typed wire codes the Go setup route pins — committed and conflict
/// as the two distinct 409 bodies, unconfirmed as the generic 500 — and
/// any other failure keeps the generic 500 (which the Go consumer maps to
/// unconfirmed). Runs as any user: scripted backend, no runtime.
#[test]
fn setup_claim_wire_codes_are_typed() {
    let cases: &[(BackendError, u16, &str)] = &[
        (
            BackendError::ExportStale,
            409,
            "factory run incarnation changed\n",
        ),
        (
            BackendError::OutputStale,
            409,
            "factory output incarnation changed\n",
        ),
        (
            BackendError::Internal,
            500,
            "native operation failed; inspect operator journal\n",
        ),
        // Unknown shape (no claim code): generic 500, never a claim.
        (
            BackendError::Denied,
            500,
            "native operation failed; inspect operator journal\n",
        ),
    ];
    for (error, status, body) in cases {
        let backend = ScriptBackend::err(*error);
        let gate = AdmissionGate::new();
        let terminal_gate = TerminalGate::new();
        let response = dispatch(
            &backend,
            &DaemonConfig::all_enabled(),
            &gate,
            &terminal_gate,
            &head("POST", "/application/setup"),
            b"{}",
            true,
        )
        .into_response();
        assert_eq!(status_of(&response), *status, "error {error:?}");
        assert_eq!(body_of(&response), body.as_bytes(), "error {error:?}");
    }
}

/// FJ-855: the FJ-853 race is a swap ABOVE the final component between
/// check and use. A swapper thread hammers ancestor rename/symlink swaps
/// while this thread publishes through the real library walk: every attempt
/// either lands the exact bytes of exactly one attempt in the pinned tree
/// or fails closed, and the evil tree never gains a byte. Runs as any user
/// (no euid gate); nothing outside the anchor is created or written.
#[test]
fn descriptor_walk_refuses_ancestor_swap_without_write_through() {
    use std::sync::atomic::{AtomicBool, Ordering};
    use std::sync::Arc;
    let anchor = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join(format!("target/fj855-walk-{}", std::process::id()));
    std::fs::create_dir_all(&anchor).expect("walk anchor");
    let evil = anchor.join("evil");
    std::fs::create_dir_all(&evil).expect("evil dir");
    std::fs::write(evil.join("sentinel"), b"evil-sentinel").expect("sentinel");
    let real = anchor.join("real");
    let saved = anchor.join("saved-aside");
    make_private_dir(&real, euid());
    let uid = euid();
    let path = real.join("role.json");
    publish_setup_file(&path, &token("op-walk", &"0".repeat(64), 1_000), uid)
        .expect("initial publish");
    let stop = Arc::new(AtomicBool::new(false));
    let hammer = {
        let stop = Arc::clone(&stop);
        let (real, saved, evil) = (real.clone(), saved.clone(), evil.clone());
        std::thread::spawn(move || {
            while !stop.load(Ordering::Relaxed) {
                let _ = std::fs::rename(&real, &saved);
                let _ = std::os::unix::fs::symlink(&evil, &real);
                let _ = std::fs::remove_file(&real);
                let _ = std::fs::rename(&saved, &real);
            }
        })
    };
    let mut landed: Vec<Vec<u8>> = Vec::new();
    let mut attempts = 0u32;
    // Attempt until at least one publish lands (minimum 50 attempts, cap
    // 2000): an all-refused run would prove nothing about atomicity.
    while (landed.is_empty() || attempts < 50) && attempts < 2000 {
        attempts += 1;
        let value = format!("{attempts:064x}");
        let attempt = token("op-walk", &value, 1_000 + attempts as i64);
        match publish_setup_file(&path, &attempt, uid) {
            Ok(()) => landed.push(serde_json::to_vec(&attempt).expect("attempt bytes")),
            Err(err) => assert!(
                err.contains("setup record"),
                "refusal leaked host text: {err}"
            ),
        }
    }
    assert!(
        !landed.is_empty(),
        "no publish landed in {attempts} attempts"
    );
    stop.store(true, Ordering::Relaxed);
    hammer.join().expect("swapper join");
    // Restore exactly one directory tree: whichever instance the walk pinned
    // last is authoritative; the other (if the hammer left both) is removed.
    if std::fs::symlink_metadata(&real)
        .map(|meta| meta.file_type().is_symlink())
        .unwrap_or(false)
    {
        std::fs::remove_file(&real).expect("unlink swap");
    }
    if saved.exists() {
        if real.exists() {
            remove_tree_anchored(&saved);
        } else {
            std::fs::rename(&saved, &real).expect("restore");
        }
    }
    // No write-through, ever: the evil tree holds only its sentinel, and no
    // staging residue leaked beside either instance.
    let mut evil_entries: Vec<_> = std::fs::read_dir(&evil)
        .expect("evil scan")
        .map(|entry| entry.expect("evil entry").file_name())
        .collect();
    evil_entries.sort();
    assert_eq!(
        evil_entries,
        vec![std::ffi::OsString::from("sentinel")],
        "write-through past the ancestor swap: {evil_entries:?}"
    );
    // Atomicity: the surviving file is the exact bytes of exactly one
    // attempt — never torn, never foreign.
    let final_bytes = std::fs::read(&path).expect("final bytes");
    assert!(
        landed.contains(&final_bytes),
        "final bytes match no single attempt"
    );
    let parsed: Token = serde_json::from_slice(&final_bytes).expect("final token");
    assert_eq!(parsed.operation, "op-walk");
    // The walk works again once the hammer stops.
    let fresh = token("op-walk", &"f".repeat(64), 9_999);
    publish_setup_file(&path, &fresh, uid).expect("publish after hammer");
    assert_eq!(read_token(&path).token, "f".repeat(64));
    remove_tree_anchored(&anchor);
    assert!(!anchor.exists(), "walk scratch survived");
}

/// FJ-855 (FJ-854 FIFO regression): a FIFO planted where a record would be
/// read refuses without hanging, and the publish surface cannot hang on it
/// either. Part (a) is a bounded probe of the exact record-open flag
/// discipline (`O_RDONLY|O_NONBLOCK|O_NOFOLLOW|O_CLOEXEC` through the walked
/// parent): the open must return promptly — without `O_NONBLOCK` it would
/// hang forever holding the mutation gate — and `fstat` shows the
/// non-record type the caller refuses. Part (b) drives the real
/// atomic-publish surface: publication never opens the target name (staged
/// exclusive creation plus atomic rename), so it completes promptly with
/// the exact record bytes in place of the FIFO. Runs as any user; nothing
/// blocks.
#[test]
fn record_open_refuses_fifo_without_hang() {
    let anchor = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join(format!("target/fj855-fifo-{}", std::process::id()));
    std::fs::create_dir_all(&anchor).expect("fifo anchor");
    let fifo = anchor.join("token.json");
    let raw = CString::new(fifo.as_os_str().as_encoded_bytes()).expect("fifo path");
    assert!(
        unsafe { libc::mkfifo(raw.as_ptr(), 0o600) } == 0,
        "FJ-855: mkfifo"
    );
    // (a) Bounded nonblocking-open probe through the walked parent: the open
    // returns (it would hang without O_NONBLOCK) and fstat proves FIFO, so
    // the record type check refuses.
    let (parent_fd, leaf) = anchor_parent_fd(&fifo);
    let (done_tx, done_rx) = std::sync::mpsc::channel();
    std::thread::spawn(move || {
        let fd = unsafe {
            libc::openat(
                parent_fd.as_raw_fd(),
                leaf.as_ptr(),
                libc::O_RDONLY | libc::O_NONBLOCK | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            )
        };
        let refused = if fd < 0 {
            false
        } else {
            let mut stat: libc::stat = unsafe { std::mem::zeroed() };
            let is_fifo = unsafe { libc::fstat(fd, &mut stat) } == 0
                && (stat.st_mode as u32 & libc::S_IFMT) == libc::S_IFIFO;
            unsafe { libc::close(fd) };
            is_fifo
        };
        let _ = done_tx.send(refused);
    });
    assert_eq!(
        done_rx
            .recv_timeout(std::time::Duration::from_secs(10))
            .expect("FJ-855: record open hung on a FIFO"),
        true,
        "FJ-855: FIFO was not refused as a non-record type"
    );
    // (b) The real publish surface never opens the target name, so the FIFO
    // cannot hang it either: the staged record publishes atomically over the
    // FIFO, leaving the exact bytes as a regular record with no residue.
    let fresh = token("op1", &"2".repeat(64), 2_800);
    publish_setup_file(&fifo, &fresh, euid()).expect("publish over FIFO");
    assert_eq!(read_token(&fifo).token, "2".repeat(64));
    let meta = std::fs::symlink_metadata(&fifo).expect("record must exist");
    assert!(
        meta.is_file() && !meta.file_type().is_symlink(),
        "publish left a non-record behind the FIFO name"
    );
    let mut entries: Vec<_> = std::fs::read_dir(&anchor)
        .expect("scan")
        .map(|entry| entry.expect("entry").file_name())
        .collect();
    entries.sort();
    assert_eq!(
        entries,
        vec![std::ffi::OsString::from("token.json")],
        "publish leaked residue: {entries:?}"
    );
    remove_tree_anchored(&anchor);
    assert!(!anchor.exists(), "fifo scratch survived");
}

/// FJ-855 guarded-cleanup negatives: a root swapped for a symlink refuses
/// (nothing foreign is deleted), while a dangling symlink child is removed
/// as a link with its target untouched.
#[test]
fn anchored_cleanup_refuses_swapped_root_and_unlinks_dangling_link() {
    let anchor = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join(format!("target/fj855-cleanup-{}", std::process::id()));
    std::fs::create_dir_all(&anchor).expect("cleanup anchor");
    let evil = anchor.join("evil");
    std::fs::create_dir_all(&evil).expect("evil dir");
    std::fs::write(evil.join("sentinel"), b"evil-sentinel").expect("sentinel");
    let victim = anchor.join("victim");
    make_private_dir(&victim, euid());
    std::fs::write(victim.join("kept"), b"kept").expect("kept");
    // A root swapped for a symlink to the evil tree refuses: the evil tree
    // and the saved victim both survive byte-identical.
    let saved = anchor.join("victim-saved");
    std::fs::rename(&victim, &saved).expect("swap aside");
    std::os::unix::fs::symlink(&evil, &victim).expect("swap link");
    assert!(
        std::panic::catch_unwind(|| remove_tree_anchored(&victim)).is_err(),
        "FJ-855: cleanup followed the swapped root"
    );
    let mut evil_entries: Vec<_> = std::fs::read_dir(&evil)
        .expect("evil scan")
        .map(|entry| entry.expect("evil entry").file_name())
        .collect();
    evil_entries.sort();
    assert_eq!(
        evil_entries,
        vec![std::ffi::OsString::from("sentinel")],
        "cleanup touched the swap target: {evil_entries:?}"
    );
    assert_eq!(
        std::fs::read(saved.join("kept")).expect("victim bytes"),
        b"kept"
    );
    std::fs::remove_file(&victim).expect("unlink swap");
    std::fs::rename(&saved, &victim).expect("restore");
    // A symlink child is removed as a link: the target survives, the tree
    // removes cleanly through the carried descriptors.
    std::os::unix::fs::symlink(evil.join("sentinel"), victim.join("dangling"))
        .expect("dangling link");
    remove_tree_anchored(&victim);
    assert!(!victim.exists(), "victim tree survived");
    assert_eq!(
        std::fs::read(evil.join("sentinel")).expect("target bytes"),
        b"evil-sentinel"
    );
    remove_tree_anchored(&anchor);
    assert!(!anchor.exists(), "cleanup scratch survived");
}

fn wire_spec(s: &SetupSpec) -> Spec {
    Spec {
        id: s.id.clone(),
        revision: 1,
        name: s.name.clone(),
        artifact_sha256: s.artifact_sha256.clone(),
        bundle_sha256: s.bundle_sha256.clone(),
        tools_sha256: s.tools_sha256.clone(),
        runtime_image: s.runtime_image.clone(),
        image: s.image.clone(),
        data: s.data.clone(),
        owner_team: "00000000-0000-0000-0000-000000000000".into(),
        origin: s.origin.clone(),
        port: s.port,
    }
}

/// FJ-v3-809 identity-before-mutation plus FJ-v3-811/812 recovery,
/// sandboxed per FJ-v3-828: the production appliance path is never touched.
///
/// Root-only: the real `Runtime::setup` path requires an operator-owned
/// tree and the real podman transport, so as any other user this test
/// reports SKIP and changes nothing. As root it drives the real setup code
/// (real files, real podman responses) through three ordered phases on one
/// retained setup, rooted at a UNIQUE fixture directory under this package's
/// `target/` scratch area (injected via `SODA_APPLICATIONS_ROOT`; the
/// production default stays `/home/soda-applications` when unset):
///
/// * a wrong-generation begin is refused BEFORE any journal, holder, or
///   credential mutation — token and journal bytes are byte-identical
///   afterward, and the recorded podman invocations are exactly the one
///   binding inspect (no inventory scan, no stop, no swap);
/// * a current begin fences the holder, reconciles proven-unclaimed over the
///   real argv (nonzero sanitized line on stderr), re-proves the fence,
///   rotates the stale token on real files, preserves the operation binding
///   and advances only the journal expiry (owner/claim preservation), then
///   fails honestly at container creation where the test image is absent —
///   rotation committed, outcome unconfirmed;
/// * the retry binds the actual current bytes without minting again;
/// * a committed reconcile verdict (bound `ok:true` shape) refuses rotation
///   with the original credential byte-identical (fence, admission, and
///   reconcile probed, no swap, no mint);
/// * a FIFO planted as the token record cannot hang the mutation gate: the
///   nonblocking record open returns at once, the type check refuses, and
///   begin completes promptly with an honest error.
///
/// FJ-862: the two packaged-binary responses the library must pin before any
/// rotation — image admission and the reconcile verdict — are emulated by the
/// test-owned podman shim (fixed match on the REAL argv: `--bundle` present,
/// `--operation` rejected; nonzero sanitized stderr lines for unclaimed and
/// refusal, the bound `ok:true` shape for committed). Every other
/// invocation still delegates to the real podman binary, so the asserted
/// call order is the real call order with real transport responses for
/// everything else; every file effect and the refusal/rotation decision are
/// the real library code. UNRUN here — no root execution before independent
/// review; the traces above were updated to the real contract by careful
/// tracing, and review must run them as root and confirm.
///
/// Refusal before outside mutation: the REAL paths are canonical-guarded
/// inside the allowed workspace scratch tree (realpath + no-symlink +
/// mode checks on every ancestor, absolute guarded scratch) BEFORE
/// anything is created, and the production resolver additionally refuses
/// any root outside the operator-owned `/home` tree. Process-global
/// variables are captured up front and restored on every exit path, and the
/// snapshot inventory over the same seam proves setup and backups share one
/// root. A logging shim stands first on PATH and delegates every call to
/// the real podman binary, so the asserted invocation order is the real
/// call order with real transport responses — never a fabricated inventory.
#[test]
fn stale_begin_refused_before_mutation_current_begin_reconciles() {
    // FJ-837: capture process-global variables BEFORE any mutation; the
    // guards restore the priors on drop — success and panic paths alike.
    let _restore_root = RestoreEnv::capture("SODA_APPLICATIONS_ROOT");
    let _restore_log = RestoreEnv::capture("SODA_809_PODMAN_LOG");
    let _restore_path = RestoreEnv::capture("PATH");
    let _restore_reconcile = RestoreEnv::capture("SODA_809_RECONCILE");
    let _restore_reconcile_img = RestoreEnv::capture("SODA_809_RECONCILE_IMG");
    let _restore_reconcile_ok = RestoreEnv::capture("SODA_809_RECONCILE_OK");
    if euid() != 0 {
        eprintln!("SKIP root-only: real setup path needs the operator-owned tree");
        return;
    }
    let manifest_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    let anchor = manifest_dir.join("target");
    let root = anchor.join(format!("soda-828-setup-root-{}", std::process::id()));
    // FJ-837: ABSOLUTE guarded scratch, never cwd-relative.
    let scratch = anchor.join(format!("setup-809-rootevidence-{}", std::process::id()));
    // FJ-837: CANONICAL validation of the REAL paths BEFORE any
    // creation/chown/removal — never the old lexical guard. FJ-847: the
    // same validation is re-run immediately before each mutation below,
    // and the stale clear, creation, and final cleanup all resolve through
    // validated descriptors — a swap between check and use fails closed on
    // the descriptor instead of acting outside the fixture.
    assert_canonical_fixture_dir(&root, &anchor);
    assert_canonical_fixture_dir(&scratch, &anchor);
    if root.exists() {
        // FJ-837: a stale tree is only cleared when it re-validates as our
        // own private operator-owned fixture — never removed blind. FJ-847:
        // the removal itself goes through the trusted descriptor.
        assert_canonical_fixture_dir(&root, &anchor);
        assert_private_root_dir(&root);
        remove_tree_anchored(&root);
    }
    if scratch.exists() {
        assert_canonical_fixture_dir(&scratch, &anchor);
        assert_private_root_dir(&scratch);
        remove_tree_anchored(&scratch);
    }
    let production_path = PathBuf::from("/home/soda-applications");
    let production_pre = production_path.exists();
    std::env::set_var("SODA_APPLICATIONS_ROOT", &root);
    std::fs::create_dir_all(&scratch).expect("scratch");
    std::fs::set_permissions(&scratch, std::fs::Permissions::from_mode(0o700))
        .expect("scratch mode");
    assert_private_root_dir(&scratch);
    let podman_log = scratch.join("podman.log");
    std::fs::write(&podman_log, b"").expect("log");
    let shim = scratch.join("podman");
    // FJ-862: the shim additionally emulates the two packaged-binary
    // responses the rotation gate pins — the reconcile verdict (fixed argv
    // match on the REAL `reconcile` contract, outcome word from
    // `SODA_809_RECONCILE` with the bound `ok:true` shape from
    // `SODA_809_RECONCILE_OK`) and image admission (fixed match on
    // `image inspect`, canned JSON from `SODA_809_RECONCILE_IMG`). Anything
    // else still delegates to the real podman binary, and every invocation
    // — emulated or delegated — is recorded in order. The argv pin rejects
    // the removed `--operation` flag and requires the real `--bundle` flag:
    // a regression to the old argv fails the invocation instead of being
    // emulated.
    std::fs::write(
        &shim,
        "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$SODA_809_PODMAN_LOG\"\ncase \"$*\" in\n*'image inspect'*)\nif [ -n \"${SODA_809_RECONCILE_IMG:-}\" ]; then\nprintf '%s' \"$SODA_809_RECONCILE_IMG\"\nexit 0\nfi\n;;\nesac\ncase \"$*\" in\n*reconcile*)\ncase \"$*\" in\n*--bundle*--data*--setup-token-file*--setup-receipt-file*) ;;\n*) exec /usr/bin/podman \"$@\" ;;\nesac\ncase \"$*\" in\n*--operation*) exit 125 ;;\nesac\nif [ \"${SODA_809_RECONCILE:-}\" = \"unclaimed\" ]; then\nprintf '%s\\n' 'can-soda: Setup reconciliation found no committed claim.' >&2\nexit 1\nfi\nif [ \"${SODA_809_RECONCILE:-}\" = \"refused\" ]; then\nprintf '%s\\n' 'can-soda: Setup reconciliation refused.' >&2\nexit 1\nfi\nprintf '%s\\n' \"$SODA_809_RECONCILE_OK\"\nexit 0\n;;\nesac\nexec /usr/bin/podman \"$@\"\n",
    )
    .expect("shim");
    std::fs::set_permissions(&shim, std::fs::Permissions::from_mode(0o755)).expect("shim mode");

    let id = format!("a{}", "8".repeat(24));
    let mut spec = setup_spec();
    spec.id = id.clone();
    spec.data = format!("a{}", "9".repeat(24));
    spec.origin = "https://setup-809.example.test".into();
    spec.port = 20092;
    // FJ-862: canned admitted-image observation matching the mutated spec:
    // the exact image digest, the runtime user, and every label admission
    // checks, plus the storage label the reconcile success shape binds.
    // Built from the spec so the pin stays exact.
    let admitted_image_json = format!(
        "{{\"image\":{:?},\"user\":\"10000:10000\",\"labels\":{{\"io.canlang.app\":\"example\",\"io.canlang.artifact\":{:?},\"io.canlang.bundle\":{:?},\"io.canlang.tools\":{:?},\"io.soda.canlang.runtime\":{:?},\"io.canlang.origin\":{:?},\"io.canlang.storage\":\"{}\"}}}}",
        spec.image,
        spec.artifact_sha256,
        spec.bundle_sha256,
        spec.tools_sha256,
        spec.runtime_image,
        spec.origin,
        "7".repeat(64),
    );
    std::env::set_var("SODA_809_RECONCILE_IMG", &admitted_image_json);
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .expect("clock")
        .as_secs() as i64;
    let journal_record = SetupRecord {
        id: id.clone(),
        application: spec.clone(),
        administrator_id: 41,
        expires_at: now - 120,
        state: "pending".into(),
        observation: None,
    };
    let begin_record = SetupRecord {
        expires_at: now + 1_800,
        ..journal_record.clone()
    };
    let old_token = token(&id, &"1".repeat(64), now - 120);
    // FJ-862: the bound `ok:true` reconcile success the committed phase
    // emulates: it must name this exact operation/app/digests/storage, or
    // the gate refuses it as confused output.
    std::env::set_var(
        "SODA_809_RECONCILE_OK",
        format!(
            "{{\"ok\":true,\"command\":\"reconcile\",\"schema\":\"canlang.soda.setup.v1\",\
            \"operation\":{id:?},\"app\":\"example\",\"artifactDigest\":{:?},\
            \"bundleDigest\":{:?},\"storageFingerprint\":\"{}\",\
            \"ownerTeam\":\"12345678-1234-1234-1234-123456789abc\",\"completedAt\":{}}}",
            spec.artifact_sha256,
            spec.bundle_sha256,
            "7".repeat(64),
            now - 60,
        ),
    );

    // FJ-841: re-validate immediately before creation (closes the
    // check/use gap between the first guard and this mutation).
    assert_canonical_fixture_dir(&root, &anchor);
    make_private_dir(&root, 0);
    let setups = root.join("setups");
    make_private_dir(&setups, 0);
    let journal = setups.join(format!("{id}.json"));
    write_private(
        &journal,
        &serde_json::to_vec(&journal_record).expect("journal"),
        0,
    );
    let journal_before = std::fs::read(&journal).expect("journal bytes");
    let control = setups.join(&id);
    make_private_dir(&control, 10000);
    let token_path = control.join("token.json");
    write_private(
        &token_path,
        &serde_json::to_vec(&old_token).expect("token"),
        10000,
    );
    let token_before = std::fs::read(&token_path).expect("token bytes");

    std::env::set_var("SODA_809_PODMAN_LOG", &podman_log);
    let mut path_value = std::ffi::OsString::from(scratch.as_os_str());
    path_value.push(":");
    path_value.push(std::env::var_os("PATH").unwrap_or_default());
    std::env::set_var("PATH", path_value);

    let calls = || -> Vec<String> {
        std::fs::read_to_string(&podman_log)
            .expect("log")
            .lines()
            .map(str::to_string)
            .collect()
    };
    // FJ-855: the selected release matches the fixture spec, so the
    // creation path proceeds past the release pin to the real (absent)
    // image — the failure under test stays the honest transport error.
    let runtime = || Runtime {
        exec: Native,
        release: Some(ReleaseBinding {
            image: spec.runtime_image.clone(),
            tools_sha256: spec.tools_sha256.clone(),
        }),
    };
    let deadline = Instant::now() + Duration::from_secs(90);
    let begin = |expected: &str| SetupRequest {
        setup: begin_record.clone(),
        action: "begin".into(),
        expected_container: expected.into(),
    };

    // Phase one: a wrong pinned generation is refused with zero mutation.
    let wrong = "e".repeat(64);
    let err = match runtime().setup(&begin(&wrong), deadline) {
        Err(err) => err,
        Ok(_) => panic!("stale generation must be refused"),
    };
    assert!(
        err.contains("setup generation differs"),
        "unexpected refusal: {err}"
    );
    assert_eq!(
        std::fs::read(&token_path).expect("token bytes"),
        token_before,
        "stale begin rotated the credential"
    );
    assert_eq!(
        std::fs::read(&journal).expect("journal bytes"),
        journal_before,
        "stale begin advanced the journal"
    );
    let first = calls();
    assert_eq!(
        first.len(),
        1,
        "stale begin called past the bind: {first:?}"
    );
    assert!(
        first[0].contains("container")
            && first[0].contains("inspect")
            && first[0].contains(&format!("soda-canlang-setup-{id}")),
        "bind did not inspect the retained generation: {first:?}"
    );
    assert!(
        !first.iter().any(|line| line
            .split_whitespace()
            .next()
            .is_some_and(|verb| verb == "stop" || verb == "ps" || verb == "create")),
        "stale begin reached past the bind: {first:?}"
    );

    // Phase two: the current begin fences (no live holder here),
    // reconciles proven-unclaimed through the emulated packaged verdict
    // (real argv, nonzero sanitized line on stderr), re-proves the fence,
    // rotates on real files, then fails honestly at container creation
    // where the test image is absent — rotation committed, outcome
    // unconfirmed. Owner/claim survive: the operation binding is unchanged
    // and only the journal expiry advances.
    std::env::set_var("SODA_809_RECONCILE", "unclaimed");
    let err = match runtime().setup(&begin(""), deadline) {
        Err(err) => err,
        Ok(_) => panic!("absent image must fail creation"),
    };
    assert!(
        err.contains("native application operation failed"),
        "unexpected failure point: {err}"
    );
    let rotated: Token =
        serde_json::from_slice(&std::fs::read(&token_path).expect("token")).expect("rotated token");
    assert_eq!(
        rotated.operation, id,
        "rotation changed the operation binding"
    );
    assert_eq!(
        rotated.expires_at,
        now + 1_800,
        "rotation missed the renewed expiry"
    );
    assert_ne!(
        rotated.token,
        "1".repeat(64),
        "rotation kept the stale credential"
    );
    assert!(
        rotated.token.len() == 64
            && rotated
                .token
                .bytes()
                .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b)),
        "rotation minted a malformed credential"
    );
    let journal_after: SetupRecord =
        serde_json::from_slice(&std::fs::read(&journal).expect("journal")).expect("journal");
    assert_eq!(
        journal_after,
        SetupRecord {
            expires_at: now + 1_800,
            ..journal_record.clone()
        },
        "renewal changed more than the journal expiry"
    );
    // FJ-855: the pre-rotation reconcile prepares the empty
    // application-data home (credential-free, exactly as creation would)
    // but mints nothing itself: only the writer lock exists under the leaf.
    let mut data_entries: Vec<_> = std::fs::read_dir(root.join("data").join(&spec.data))
        .expect("data scan")
        .map(|entry| entry.expect("data entry").file_name())
        .collect();
    data_entries.sort();
    assert_eq!(
        data_entries,
        vec![std::ffi::OsString::from(".writer-lock")],
        "failed begin left unexpected application data: {data_entries:?}"
    );
    let second = calls();
    assert_eq!(
        second.len(),
        first.len() + 8,
        "current begin took an unexpected path: {second:?}"
    );
    let verbs: Vec<&str> = second[first.len()..]
        .iter()
        .map(|line| line.split_whitespace().next().unwrap_or(""))
        .collect();
    assert_eq!(
        verbs,
        vec![
            "ps",
            "image",
            "run",
            "ps",
            "container",
            "ps",
            "image",
            "create"
        ],
        "current begin did not fence-reconcile-then-admit: {second:?}"
    );

    // Phase three: the retry binds the actual current bytes — no second
    // mint, no journal move, same honest failure.
    let rotated_before = std::fs::read(&token_path).expect("token bytes");
    let journal_settled = std::fs::read(&journal).expect("journal bytes");
    let err = match runtime().setup(&begin(""), deadline) {
        Err(err) => err,
        Ok(_) => panic!("missing image must fail admission again"),
    };
    assert!(
        err.contains("native application operation failed"),
        "retry failed at a new point: {err}"
    );
    assert_eq!(
        std::fs::read(&token_path).expect("token bytes"),
        rotated_before,
        "retry minted a second credential over actual state"
    );
    assert_eq!(
        std::fs::read(&journal).expect("journal bytes"),
        journal_settled,
        "retry moved the journal"
    );
    let third = calls();
    assert_eq!(
        third.len(),
        second.len() + 4,
        "retry took an unexpected path: {third:?}"
    );
    let retry_verbs: Vec<&str> = third[second.len()..]
        .iter()
        .map(|line| line.split_whitespace().next().unwrap_or(""))
        .collect();
    assert_eq!(
        retry_verbs,
        vec!["container", "ps", "image", "create"],
        "retry re-ran rotation instead of binding actual state: {third:?}"
    );

    // Phase four: a committed durable claim refuses rotation with the
    // original credential untouched. The expired token below re-arms the
    // renewal rotation branch; the emulated reconcile reports committed;
    // begin must refuse with the fixed message, mint nothing, and leave
    // both the expired token bytes and the settled journal exactly as they
    // were. Only fence, admission, and reconcile are probed — no swap.
    std::env::set_var("SODA_809_RECONCILE", "committed");
    let expired = token(&id, &"5".repeat(64), now - 60);
    write_private(
        &token_path,
        &serde_json::to_vec(&expired).expect("expired token"),
        10000,
    );
    let expired_before = std::fs::read(&token_path).expect("token bytes");
    let err = match runtime().setup(&begin(""), deadline) {
        Err(err) => err,
        Ok(_) => panic!("committed claim must refuse rotation"),
    };
    assert!(
        err.contains("setup claim is committed") && err.contains("retained credential kept"),
        "unexpected refusal: {err}"
    );
    assert_eq!(
        std::fs::read(&token_path).expect("token bytes"),
        expired_before,
        "refused rotation destroyed the original credential"
    );
    assert_eq!(
        std::fs::read(&journal).expect("journal bytes"),
        journal_settled,
        "refused rotation moved the journal"
    );
    let fourth = calls();
    assert_eq!(
        fourth.len(),
        third.len() + 3,
        "committed refusal took an unexpected path: {fourth:?}"
    );
    let refused_verbs: Vec<&str> = fourth[third.len()..]
        .iter()
        .map(|line| line.split_whitespace().next().unwrap_or(""))
        .collect();
    assert_eq!(
        refused_verbs,
        vec!["ps", "image", "run"],
        "committed refusal probed past reconcile: {fourth:?}"
    );

    // Phase five: a FIFO planted where the token record would be read
    // cannot hang the mutation gate. The nonblocking record open returns at
    // once, the type check refuses the non-record, and begin completes
    // promptly with an honest error — the FIFO survives and nothing is
    // minted over it. (Without O_NONBLOCK this phase would hang inside the
    // open; the suite timeout, not an assertion, would be the detector.)
    std::env::set_var("SODA_809_RECONCILE", "unclaimed");
    std::fs::remove_file(&token_path).expect("remove token");
    let fifo_raw =
        CString::new(token_path.as_os_str().as_encoded_bytes()).expect("FJ-855: fifo path");
    assert!(
        unsafe { libc::mkfifo(fifo_raw.as_ptr(), 0o600) } == 0,
        "FJ-855: mkfifo token"
    );
    assert!(
        unsafe { libc::chown(fifo_raw.as_ptr(), 10000, 10000) } == 0,
        "FJ-855: own fifo"
    );
    let err = match runtime().setup(&begin(""), deadline) {
        Err(err) => err,
        Ok(_) => panic!("FIFO token must refuse"),
    };
    assert!(
        err.contains("native application operation failed"),
        "unexpected failure point: {err}"
    );
    let fifo_meta = std::fs::symlink_metadata(&token_path).expect("fifo must survive");
    assert!(
        fifo_meta.file_type().is_fifo(),
        "FJ-855: FIFO token was disturbed"
    );
    let fifth = calls();
    assert_eq!(
        fifth.len(),
        fourth.len() + 4,
        "FIFO begin took an unexpected path: {fifth:?}"
    );
    let fifo_verbs: Vec<&str> = fifth[fourth.len()..]
        .iter()
        .map(|line| line.split_whitespace().next().unwrap_or(""))
        .collect();
    assert_eq!(
        fifo_verbs,
        vec!["container", "ps", "image", "create"],
        "FIFO begin probed past the honest failure: {fifth:?}"
    );

    // FJ-837: the same fixture seam drives backups — the snapshot
    // inventory over the redirected root sees the empty fixture tree, never
    // the fixed production path (which stays absent throughout). This probe
    // makes zero podman calls, so the invocation counts above are untouched;
    // without the shared seam this fails (the fixed path does not exist).
    let inventory = runtime()
        .snapshots(
            &SnapshotRequest {
                application: wire_spec(&spec),
                action: "list".into(),
                id: String::new(),
                expected_container: String::new(),
            },
            deadline,
        )
        .expect("snapshot inventory over the fixture seam");
    assert!(
        inventory.complete && inventory.snapshots.is_empty(),
        "snapshot seam left the fixture root"
    );

    // FJ-847: the final cleanup was path-based — these removals now pass
    // the same canonical + private validation as every earlier mutation AND
    // resolve through the trusted descriptor, so a swapped or stale tree
    // refuses instead of deleting foreign bytes.
    assert_canonical_fixture_dir(&root, &anchor);
    assert_private_root_dir(&root);
    assert_canonical_fixture_dir(&scratch, &anchor);
    assert_private_root_dir(&scratch);
    remove_tree_anchored(&root);
    remove_tree_anchored(&scratch);
    // FJ-837: process-global values restore via the guards on scope exit —
    // success and panic paths alike — so no explicit removal here.
    assert!(!root.exists(), "fixture tree survived");
    assert!(!scratch.exists(), "scratch tree survived");
    assert_eq!(
        production_path.exists(),
        production_pre,
        "production appliance path was touched"
    );
}
