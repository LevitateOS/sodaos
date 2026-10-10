// Admin/runtime HTTP admission and dead-listener integration scenarios,
// split from tests/broker.rs (A-R02c). Fixture support lives in
// tests/common.
mod common;

use common::{controller, fixture_key, Ephemeral};
use soda_identity::wire::{AcquireRequest, Request, UnixTime};
use std::sync::Arc;

#[test]
fn http_admission_matches_go() {
    use soda_identity::http::Server;
    use std::io::{Read, Write};
    use std::os::unix::net::{UnixListener, UnixStream};

    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = Arc::new(controller(fixture.store(&fixture_key())));
    let dir = std::env::temp_dir().join(format!("soda-broker-test-{}", std::process::id()));
    let _ = std::fs::create_dir_all(&dir);
    let admin_path = dir.join("admin.sock");
    let runtime_path = dir.join("runtime.sock");
    let _ = std::fs::remove_file(&admin_path);
    let _ = std::fs::remove_file(&runtime_path);
    let admin = UnixListener::bind(&admin_path).unwrap();
    let runtime = UnixListener::bind(&runtime_path).unwrap();
    let shutdown = Arc::new(std::sync::atomic::AtomicBool::new(false));
    let inflight = Arc::new(std::sync::atomic::AtomicUsize::new(0));
    let admin_server = Server::new(
        Arc::clone(&broker),
        false,
        Arc::clone(&shutdown),
        Arc::clone(&inflight),
    );
    let runtime_server = Server::new(
        Arc::clone(&broker),
        true,
        Arc::clone(&shutdown),
        Arc::clone(&inflight),
    );
    struct Guard {
        shutdown: Arc<std::sync::atomic::AtomicBool>,
    }
    impl Drop for Guard {
        fn drop(&mut self) {
            // A failed assertion must still release the server threads.
            self.shutdown
                .store(true, std::sync::atomic::Ordering::SeqCst);
        }
    }
    std::thread::scope(|scope| {
        let _guard = Guard {
            shutdown: Arc::clone(&shutdown),
        };
        scope.spawn(|| admin_server.serve(&admin));
        scope.spawn(|| runtime_server.serve(&runtime));
        let call = |path: &std::path::Path, raw: &[u8]| -> (u16, Vec<u8>) {
            let mut stream = UnixStream::connect(path).unwrap();
            stream.write_all(raw).unwrap();
            let mut out = Vec::new();
            stream.read_to_end(&mut out).unwrap();
            let head = out.split(|b| *b == b'\n').next().unwrap_or(b"").to_vec();
            let status = head
                .split(|b| *b == b' ')
                .nth(1)
                .and_then(|c| std::str::from_utf8(c).ok()?.parse::<u16>().ok())
                .unwrap_or(0);
            let body = out
                .windows(4)
                .position(|w| w == b"\r\n\r\n")
                .map(|i| out[i + 4..].to_vec())
                .unwrap_or_default();
            (status, body)
        };
        let post = |target: &str, body: &str, extra: &str| -> Vec<u8> {
            format!(
                "POST {target} HTTP/1.1\r\nHost: x\r\nContent-Type: application/json\r\nContent-Length: {}\r\n{extra}\r\n{body}",
                body.len()
            )
            .into_bytes()
        };
        // Happy path: no connections yet.
        let (status, body) = call(
            &admin_path,
            &post("/connections", r#"{"owner_id":"1"}"#, ""),
        );
        assert_eq!((status, body.as_slice()), (200, b"[]\n".as_slice()));
        // Admission gates.
        let (status, _) = call(&admin_path, b"GET /connections HTTP/1.1\r\nHost: x\r\n\r\n");
        assert_eq!(status, 403);
        let (status, body) = call(
            &admin_path,
            &post("/connections?x=1", r#"{"owner_id":"1"}"#, ""),
        );
        assert_eq!(status, 403);
        assert_eq!(body, b"denied\n");
        let (status, _) = call(
            &admin_path,
            &post(
                "/connections",
                r#"{"owner_id":"1"}"#,
                "Origin: https://x\r\n",
            ),
        );
        assert_eq!(status, 403);
        // Strict bodies.
        let (status, body) = call(&admin_path, &post("/connections", "not json", ""));
        assert_eq!(status, 400);
        assert_eq!(body, b"invalid request\n");
        let (status, _) = call(
            &admin_path,
            &post("/connections", r#"{"owner_id":"1","bogus":true}"#, ""),
        );
        assert_eq!(status, 400);
        // The route uses the canonical Go-shaped names while preserving the
        // existing integer-string, null-scalar and base64 wire codecs.
        let (status, _) = call(
            &runtime_path,
            &post(
                "/register",
                r#"{"id":"missing-lease","binding":{"child_id":null,"uid":17,"gid":18,"scope":"terminal","credential_root":"root","invocation_id":"invocation","kind":"factory","generation":1}}"#,
                "",
            ),
        );
        assert_eq!(status, 404, "canonical binding reached the broker route");
        for (path, body) in [
            (
                "/register",
                r#"{"id":"missing-lease","binding":{"child_id":"x","future":true}}"#,
            ),
            (
                "/grant/create",
                r#"{"owner_id":"1","grant":{"connection_id":"missing","user_id":"2","project_id":"project","confirm_subscription":true,"confirm_credential_exposure":true,"future":true}}"#,
            ),
            (
                "/acquire",
                r#"{"acquire":{"provider_id":"codex","future":true}}"#,
            ),
            ("/connections", r#"{"Owner_id":"1"}"#),
        ] {
            let (status, response) = call(&runtime_path, &post(path, body, ""));
            assert_eq!(
                (status, response.as_slice()),
                (400, b"invalid request\n".as_slice())
            );
        }
        for body in [
            r#"{"owner_id":"1","owner_id":"1"}"#.to_string(),
            format!("{{\"id\":{}0{}}}", "[".repeat(101), "]".repeat(101)),
        ] {
            let (status, _) = call(&runtime_path, &post("/connections", &body, ""));
            assert_eq!(status, 400, "strict body accepted: {body}");
        }
        // Lifecycle administration fences acquisition without credential delivery.
        broker.start_enrollment(1, "codex", "synthetic").unwrap();
        let connection = broker
            .enrollment(1, "enrollment-1")
            .unwrap()
            .connection
            .unwrap();
        let mut acquire = AcquireRequest {
            repository_id: 0,
            provider_id: "codex".to_string(),
            execution_id: "admin-preclosed".to_string(),
            actor_id: 1,
            connection_id: connection.id,
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline: UnixTime {
                sec: UnixTime::now().sec + 3600,
                nanos: 0,
            },
            role: String::new(),
        };
        let selector = r#"{"kind":"factory","execution_id":"admin-preclosed"}"#;
        let (status, response) = call(&admin_path, &post("/execution/close", selector, ""));
        assert_eq!((status, response.as_slice()), (200, b"{}\n".as_slice()));
        let (status, response) = call(&admin_path, &post("/execution/get", selector, ""));
        assert_eq!(status, 200);
        assert_eq!(
            serde_json::from_slice::<serde_json::Value>(&response).unwrap(),
            serde_json::json!({
                "kind": "factory", "execution_id": "admin-preclosed", "digest": "", "state": "terminal",
            })
        );
        let payload = serde_json::to_string(&Request {
            acquire: Some(acquire.clone()),
            ..Request::default()
        })
        .unwrap();
        let (status, _) = call(&runtime_path, &post("/acquire", &payload, ""));
        assert_eq!(
            status, 403,
            "admin closure must fence a valid later acquisition"
        );
        // A live execution's administration response carries only its metadata.
        acquire.execution_id = "admin-metadata".to_string();
        let lease = broker.acquire(&acquire).unwrap();
        let (status, response) = call(
            &admin_path,
            &post(
                "/execution/get",
                r#"{"kind":"factory","execution_id":"admin-metadata"}"#,
                "",
            ),
        );
        assert_eq!(status, 200);
        let metadata: serde_json::Value = serde_json::from_slice(&response).unwrap();
        assert_eq!(metadata["kind"], "factory");
        assert_eq!(metadata["execution_id"], "admin-metadata");
        assert_eq!(metadata["lease_id"], lease.id);
        assert_eq!(metadata["state"], "live");
        assert!(!metadata["digest"].as_str().unwrap().is_empty());
        let mut fields = metadata
            .as_object()
            .unwrap()
            .keys()
            .map(String::as_str)
            .collect::<Vec<_>>();
        fields.sort_unstable();
        assert_eq!(
            fields,
            ["digest", "execution_id", "kind", "lease_id", "state"]
        );
        for path in ["/execution/get", "/execution/close"] {
            for selector in [
                r#"{"kind":"unknown","execution_id":"admin-metadata"}"#,
                r#"{"kind":"factory"}"#,
            ] {
                let (status, _) = call(&admin_path, &post(path, selector, ""));
                assert_eq!(status, 403, "invalid lifecycle selector on {path}");
            }
        }
        // Credential custody and lease recovery remain runtime-only.
        for (path, body) in [
            ("/acquire", payload.as_str()),
            (
                "/register",
                r#"{"id":"missing-lease","binding":{"kind":"factory","id":"admin-metadata","generation":1}}"#,
            ),
            (
                "/return",
                r#"{"id":"missing-lease","binding":{"kind":"factory","id":"admin-metadata","generation":1},"credential":"e30="}"#,
            ),
            (
                "/reject",
                r#"{"id":"missing-lease","binding":{"kind":"factory","id":"admin-metadata","generation":1}}"#,
            ),
            ("/reconcile-lease", r#"{"id":"missing-lease"}"#),
        ] {
            let (status, response) = call(&admin_path, &post(path, body, ""));
            assert_eq!(
                (status, response.as_slice()),
                (403, b"denied\n".as_slice()),
                "runtime-only route {path}"
            );
        }
        // Unknown paths are denied everywhere.
        let (status, _) = call(&runtime_path, &post("/nope", r#"{}"#, ""));
        assert_eq!(status, 403);
        shutdown.store(true, std::sync::atomic::Ordering::SeqCst);
    });
    let _ = std::fs::remove_file(&admin_path);
    let _ = std::fs::remove_file(&runtime_path);
}

// An unusable listener ends serve() promptly for supervisor restart.
// This exercises initial listener setup failure, not the fatal accept counter.
#[test]
#[cfg(target_os = "linux")]
fn dead_listener_fails_fast() {
    use soda_identity::http::Server;
    use std::os::unix::io::AsRawFd;
    use std::os::unix::net::UnixListener;

    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = Arc::new(controller(fixture.store(&fixture_key())));
    let dir = std::env::temp_dir().join(format!("soda-broker-test-{}", std::process::id()));
    let _ = std::fs::create_dir_all(&dir);
    let path = dir.join("dead.sock");
    let _ = std::fs::remove_file(&path);
    let listener = UnixListener::bind(&path).unwrap();
    // Close the fd behind serve's back: every accept fails fatally. Pin
    // the number with /dev/null guards so no parallel test thread reuses
    // it mid-test; skip if the race is lost instead of serving a live fd.
    let fd = listener.as_raw_fd();
    unsafe { libc::close(fd) };
    let mut guards = Vec::new();
    for _ in 0..64 {
        let guard = std::fs::File::open("/dev/null").unwrap();
        let pinned = guard.as_raw_fd() == fd;
        guards.push(guard);
        if pinned {
            break;
        }
    }
    if guards.iter().all(|f| f.as_raw_fd() != fd) {
        eprintln!("could not pin dead listener fd; skipping");
        std::mem::forget(listener);
        return;
    }
    let server = Server::new(
        Arc::clone(&broker),
        false,
        Arc::new(std::sync::atomic::AtomicBool::new(false)),
        Arc::new(std::sync::atomic::AtomicUsize::new(0)),
    );
    let start = std::time::Instant::now();
    server.serve(&listener);
    assert!(
        start.elapsed() < std::time::Duration::from_secs(10),
        "serve spun on a dead listener"
    );
    std::mem::forget(listener);
    drop(guards);
    let _ = std::fs::remove_file(&path);
}

#[test]
fn shutdown_joins_admitted_provider_work_without_blocking_http_runtime() {
    use soda_identity::control::{Controller, EnrollmentSession, Provider, Runtime};
    use soda_identity::http::Server;
    use soda_identity::wire::{Connection, Enrollment, Error, Lease};
    use std::collections::HashMap;
    use std::io::{Read, Write};
    use std::os::unix::net::{UnixListener, UnixStream};
    use std::sync::atomic::{AtomicBool, Ordering};
    use std::sync::{mpsc, Mutex};
    use std::time::Duration;

    struct BlockingProvider {
        entered: mpsc::SyncSender<()>,
        release: Mutex<mpsc::Receiver<()>>,
        finished: Arc<AtomicBool>,
    }
    struct Session(Enrollment);
    impl EnrollmentSession for Session {
        fn snapshot(&self) -> Enrollment {
            self.0.clone()
        }
        fn finish(&self) -> Result<(Connection, Vec<u8>), Error> {
            Err(Error::internal("unused test session"))
        }
        fn close(&self) -> Result<(), Error> {
            Ok(())
        }
    }
    impl Provider for BlockingProvider {
        fn start(&self, _owner: i64) -> Result<Box<dyn EnrollmentSession>, Error> {
            self.entered.send(()).unwrap();
            self.release.lock().unwrap().recv().unwrap();
            self.finished.store(true, Ordering::SeqCst);
            Ok(Box::new(Session(Enrollment {
                provider_id: "codex".into(),
                id: "blocked-test".into(),
                verification_url: String::new(),
                user_code: String::new(),
                state: "pending".into(),
                error: String::new(),
                connection: None,
            })))
        }
    }
    struct EmptyRuntime;
    impl Runtime for EmptyRuntime {
        fn validate(&self, _: &Lease) -> Result<(), Error> {
            Ok(())
        }
        fn stop(&self, _: &Lease) -> Result<(), Error> {
            Ok(())
        }
        fn finish(&self, _: &Lease) -> Result<Vec<u8>, Error> {
            Ok(Vec::new())
        }
    }

    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let (entered_tx, entered_rx) = mpsc::sync_channel(1);
    let (release_tx, release_rx) = mpsc::channel();
    let finished = Arc::new(AtomicBool::new(false));
    let mut providers: HashMap<String, Box<dyn Provider>> = HashMap::new();
    providers.insert(
        "codex".into(),
        Box::new(BlockingProvider {
            entered: entered_tx,
            release: Mutex::new(release_rx),
            finished: Arc::clone(&finished),
        }),
    );
    let broker = Arc::new(
        Controller::new(
            fixture.store(&fixture_key()),
            providers,
            Box::new(EmptyRuntime),
        )
        .unwrap(),
    );
    let dir = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(2)
        .unwrap()
        .join(".artifacts/l08-l09");
    let _ = std::fs::create_dir_all(&dir);
    let path = dir.join(format!("http-drain-{}.sock", std::process::id()));
    let _ = std::fs::remove_file(&path);
    let listener = UnixListener::bind(&path).unwrap();
    let shutdown = Arc::new(AtomicBool::new(false));
    struct ShutdownOnDrop(Arc<AtomicBool>);
    impl Drop for ShutdownOnDrop {
        fn drop(&mut self) {
            self.0.store(true, Ordering::SeqCst);
        }
    }
    let _shutdown_guard = ShutdownOnDrop(Arc::clone(&shutdown));
    let server = Server::new(
        Arc::clone(&broker),
        false,
        Arc::clone(&shutdown),
        Arc::new(std::sync::atomic::AtomicUsize::new(0)),
    );
    let done = Arc::new(AtomicBool::new(false));
    let thread_done = Arc::clone(&done);
    let server_thread = std::thread::spawn(move || {
        server.serve(&listener);
        thread_done.store(true, Ordering::SeqCst);
    });

    let start_request = |request_path: std::path::PathBuf, label: String| {
        std::thread::spawn(move || {
            let body = format!(r#"{{"owner_id":"1","provider_id":"codex","label":"{label}"}}"#);
            let mut stream = UnixStream::connect(request_path).unwrap();
            write!(
                stream,
                "POST /enrollment/start HTTP/1.1\r\nHost: x\r\nContent-Length: {}\r\n\r\n{}",
                body.len(),
                body
            )
            .unwrap();
            let mut response = Vec::new();
            let _ = stream.read_to_end(&mut response);
            response
        })
    };
    let request = start_request(path.clone(), "test".to_string());
    entered_rx
        .recv_timeout(Duration::from_secs(5))
        .expect("provider callback admitted");

    // A request that needs no Controller lock must still be serviced while
    // the synchronous provider callback occupies a blocking backend worker.
    let mut probe = UnixStream::connect(&path).unwrap();
    probe
        .set_read_timeout(Some(Duration::from_secs(2)))
        .unwrap();
    probe
        .write_all(b"GET /connections HTTP/1.1\r\nHost: x\r\n\r\n")
        .unwrap();
    let mut response = Vec::new();
    probe.read_to_end(&mut response).unwrap();
    assert!(response.starts_with(b"HTTP/1.1 403"));

    // Eight backend slots are available. The first provider callback holds
    // the Controller lock, so the next seven admitted calls wait behind it;
    // the ninth call must be rejected instead of accumulating another job.
    let queued: Vec<_> = (0..7)
        .map(|index| start_request(path.clone(), format!("queued-{index}")))
        .collect();
    std::thread::sleep(Duration::from_millis(150));
    let mut excess = UnixStream::connect(&path).unwrap();
    excess
        .set_read_timeout(Some(Duration::from_secs(2)))
        .unwrap();
    let body = r#"{"owner_id":"1","provider_id":"codex","label":"excess"}"#;
    write!(
        excess,
        "POST /enrollment/start HTTP/1.1\r\nHost: x\r\nContent-Length: {}\r\n\r\n{}",
        body.len(),
        body
    )
    .unwrap();
    let mut excess_response = Vec::new();
    excess.read_to_end(&mut excess_response).unwrap();
    assert!(excess_response.starts_with(b"HTTP/1.1 503"));

    let mut partial_head = UnixStream::connect(&path).unwrap();
    partial_head
        .write_all(b"POST /connections HTTP/1.1\r\nHost:")
        .unwrap();
    let mut partial_body = UnixStream::connect(&path).unwrap();
    partial_body
        .write_all(b"POST /connections HTTP/1.1\r\nHost: x\r\nContent-Length: 100\r\n\r\n{")
        .unwrap();

    shutdown.store(true, Ordering::SeqCst);
    std::thread::sleep(Duration::from_millis(100));
    assert!(
        !done.load(Ordering::SeqCst),
        "server returned before admitted backend work finished"
    );
    release_tx.send(()).unwrap();
    server_thread.join().unwrap();
    assert!(done.load(Ordering::SeqCst));
    let _ = request.join().unwrap();
    for request in queued {
        let _ = request.join().unwrap();
    }
    assert!(finished.load(Ordering::SeqCst));
    let _ = std::fs::remove_file(path);
}
