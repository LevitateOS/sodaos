// Smoke tests for the PR26 daemon-mux skeleton.
//
// The gmux modules are wired into lib.rs; this harness exercises the actual
// library owners. If these tests pass, admission matches daemon.go and
// every route dispatches.
use soda_host::gmux_admission::{
    body_limit_for, is_admitted_mutation_path, valid_identity_request, valid_terminal_request,
    validate_native_request, validate_tailnet_request, AdmissionGate, RequestHead, TerminalGate,
    ADMITTED_MUTATION_PATHS, BODY_LIMIT_DEFAULT, BODY_LIMIT_IDENTITY, BODY_LIMIT_LARGE,
    IDENTITY_ACTIONS, NATIVE_CLEAN_PATHS, TAILNET_ACTIONS, TERMINAL_FRAME_LIMIT,
    TERMINAL_REQUEST_LIMIT, TERMINAL_STREAM_CAP,
};
use soda_host::gmux_backend::{BackendError, ExecBackend, StubBackend, TerminalSession};
use soda_host::gmux_routes::{
    dispatch, websocket_accept_key, DaemonConfig, RouteOutcome, ROUTE_TABLE,
};
use soda_host::gmux_server::{systemd_listener, Server};
use std::io::{Read, Write};
use std::os::unix::net::{UnixListener, UnixStream};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex};

// -- helpers --

fn head(method: &str, path: &str) -> RequestHead {
    let mut h = RequestHead::post(path);
    h.method = method.to_string();
    h
}

fn status_of(response: &[u8]) -> u16 {
    let text = std::str::from_utf8(response).unwrap();
    assert!(text.starts_with("HTTP/1.1 "), "bad status line: {text:?}");
    text["HTTP/1.1 ".len()..][..3].parse().unwrap()
}

fn text_of(response: &[u8]) -> &str {
    std::str::from_utf8(response).unwrap()
}

fn body_of(response: &[u8]) -> &[u8] {
    let pos = response
        .windows(4)
        .position(|w| w == b"\r\n\r\n")
        .expect("response has no head/body split");
    &response[pos + 4..]
}

fn dispatch_stub(method: &str, path: &str, body: &[u8]) -> Vec<u8> {
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
}

impl ScriptBackend {
    fn err(error: BackendError) -> ScriptBackend {
        ScriptBackend {
            result: Err(error),
            session: 7,
            seen_bodies: Arc::new(Mutex::new(Vec::new())),
            seen_image: Arc::new(Mutex::new(String::new())),
            pumped: Arc::new(AtomicU64::new(u64::MAX)),
        }
    }

    fn ok(json: &[u8]) -> ScriptBackend {
        ScriptBackend {
            result: Ok(json.to_vec()),
            session: 7,
            seen_bodies: Arc::new(Mutex::new(Vec::new())),
            seen_image: Arc::new(Mutex::new(String::new())),
            pumped: Arc::new(AtomicU64::new(u64::MAX)),
        }
    }

    fn replay(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        self.seen_bodies.lock().unwrap().push(body.to_vec());
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
        _stream: UnixStream,
        session: TerminalSession,
    ) -> Result<(), BackendError> {
        self.pumped.store(session.id, Ordering::SeqCst);
        Ok(())
    }
}

// -- admission: native validation --

#[test]
fn native_clean_paths_reject_query_and_escapes() {
    assert_eq!(NATIVE_CLEAN_PATHS.len(), 18);
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

#[test]
#[cfg(target_os = "linux")]
fn peer_credentials_attest_self() {
    let (a, b) = UnixStream::pair().unwrap();
    let cred = soda_host::gmux_admission::peer_cred(&a).unwrap();
    assert_eq!(cred.pid, std::process::id() as i32);
    assert_eq!(cred.uid, unsafe { libc::geteuid() });
    assert_eq!(cred.gid, unsafe { libc::getegid() });
    drop(b);
}

#[test]
#[cfg(target_os = "linux")]
fn muse_peer_carries_pidfd_pin() {
    use std::os::unix::io::AsRawFd;
    let (a, _b) = UnixStream::pair().unwrap();
    let peer = soda_host::gmux_admission::muse_peer(a.as_raw_fd()).unwrap();
    assert_eq!(peer.pid, std::process::id() as i32);
    assert!(peer.pidfd >= 0, "pidfd must be pinned");
    soda_host::gmux_admission::close_pidfd(&peer);
}

// -- routes: table completeness --

#[test]
fn route_table_has_every_go_route() {
    assert_eq!(ROUTE_TABLE.len(), 33);
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
        assert!(text_of(&response).contains("Content-Type: text/plain; charset=utf-8"));
        assert!(text_of(&response).contains("X-Content-Type-Options: nosniff"));
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
    assert!(text_of(&response).contains("Content-Type: application/json"));
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
fn create_and_mutations_take_the_gate_reads_do_not() {
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
    // A mutation blocks while the gate is held elsewhere.
    let (tx, rx) = std::sync::mpsc::channel();
    std::thread::scope(|s| {
        s.spawn(|| {
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
            tx.send(status_of(&response)).unwrap();
        });
        assert!(
            rx.recv_timeout(std::time::Duration::from_millis(200))
                .is_err(),
            "mutation must block on the held gate"
        );
        drop(held);
        assert_eq!(
            rx.recv_timeout(std::time::Duration::from_secs(5)).unwrap(),
            200
        );
    });
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
        assert!(text_of(&response).contains("Cache-Control: no-store\r\n"));
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
    assert!(text_of(&response).contains("Cache-Control: no-store\r\n"));

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
    assert!(text_of(&response).contains("Cache-Control: no-store\r\n"));
}

// -- routes: terminal --

fn terminal_head() -> RequestHead {
    let mut h = head("GET", "/terminal");
    h.upgrade_websocket = true;
    h.ws_key = Some("dGhlIHNhbXBsZSBub25jZQ==".to_string());
    h
}

#[test]
fn websocket_accept_key_matches_rfc6455_vector() {
    assert_eq!(
        websocket_accept_key("dGhlIHNhbXBsZSBub25jZQ=="),
        "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
    );
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
            assert!(text_of(&response).contains("Connection: Upgrade\r\n"));
            assert!(text_of(&response).contains("Upgrade: websocket\r\n"));
            assert!(text_of(&response)
                .contains("Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=\r\n"));
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
        StubBackend.pump_terminal(stream, TerminalSession { id: 0 }),
        Err(BackendError::Unimplemented)
    );
}

// -- server: end to end over a real socket --

fn read_all(stream: &mut UnixStream) -> Vec<u8> {
    let mut out = Vec::new();
    stream.read_to_end(&mut out).unwrap();
    out
}

#[test]
fn server_serves_stub_routes_and_parser_rejections() {
    let path = std::env::temp_dir().join(format!("gmux-smoke-{}.sock", std::process::id()));
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
    assert!(text_of(&response).contains("Cache-Control: no-store\r\n"));

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
    assert_eq!(status_of(&read_all(&mut client)), 431);

    // Chunked bodies are refused (single Content-Length only).
    let mut client = UnixStream::connect(&path).unwrap();
    client
        .write_all(b"POST /create HTTP/1.1\r\nHost: x\r\nTransfer-Encoding: chunked\r\n\r\n")
        .unwrap();
    assert_eq!(status_of(&read_all(&mut client)), 400);

    server.shutdown();
    handle.join().unwrap();
    assert_eq!(server.inflight(), 0);
    let _ = std::fs::remove_file(&path);
}

#[test]
fn server_runs_terminal_upgrade_and_pump() {
    let path = std::env::temp_dir().join(format!("gmux-term-{}.sock", std::process::id()));
    let _ = std::fs::remove_file(&path);
    let listener = UnixListener::bind(&path).unwrap();
    let backend = ScriptBackend::ok(b"{}");
    let pumped = Arc::clone(&backend.pumped);
    let server = Arc::new(Server::new(Arc::new(backend), DaemonConfig::all_enabled()));
    let serving = Arc::clone(&server);
    let handle = std::thread::spawn(move || serving.serve(&listener));

    let mut client = UnixStream::connect(&path).unwrap();
    client
        .write_all(
            b"GET /terminal HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n",
        )
        .unwrap();
    // The pump records the session and returns, closing the stream.
    let response = read_all(&mut client);
    assert_eq!(status_of(&response), 101);
    assert!(text_of(&response).contains("Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo="));
    assert_eq!(pumped.load(Ordering::SeqCst), 7);

    server.shutdown();
    handle.join().unwrap();
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
