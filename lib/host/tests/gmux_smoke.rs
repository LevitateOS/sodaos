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
    fn factory_stop(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn factory_takeover(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.replay(body)
    }
    fn factory_output(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
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
    assert_eq!(NATIVE_CLEAN_PATHS.len(), 19);
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
    assert_eq!(ADMITTED_MUTATION_PATHS.len(), 7);
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
    assert_eq!(TAILNET_ACTIONS.len(), 6);
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
    assert_eq!(ROUTE_TABLE.len(), 34);
    let count = |prefix: &str| {
        ROUTE_TABLE
            .iter()
            .filter(|(_, p)| p.starts_with(prefix))
            .count()
    };
    assert_eq!(count("/identity/"), 5);
    assert_eq!(count("/tailnet/"), 6);
    assert_eq!(
        ROUTE_TABLE
            .iter()
            .filter(|(_, p)| *p == "/terminal")
            .count(),
        1
    );
    for path in [
        "/profile",
        "/create",
        "/inspect",
        "/os",
        "/connection",
        "/lifecycle",
        "/access-keys",
        "/account",
        "/project-access",
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
fn tailnet_success_has_no_trailing_newline_and_disabled_is_503() {
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

    let mut config = DaemonConfig::all_enabled();
    config.tailnet_management = false;
    let response = dispatch(
        &backend,
        &config,
        &gate,
        &terminal_gate,
        &head("POST", "/tailnet/settings"),
        b"{}",
        true,
    )
    .into_response();
    assert_eq!(status_of(&response), 503);
    assert_eq!(body_of(&response), b"Tailnet management disabled\n");

    let response = dispatch_stub("POST", "/tailnet/bogus", b"{}");
    assert_eq!(status_of(&response), 404);
    assert_eq!(
        response.header_value("cache-control").as_deref(),
        Some("no-store")
    );
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
