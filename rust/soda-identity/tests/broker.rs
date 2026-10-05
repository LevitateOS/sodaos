// Broker integration tests over an ephemeral PostgreSQL database. They
// skip when the A10 fixture environment (SODA_PG_HOST, SODA_PG_PORT,
// SODA_PG_SUPER_PASSWORD_FILE) is absent, exactly like the Go suite.
use soda_identity::control::{self, Controller};
use soda_identity::pg::{Client as PgClient, Dsn};
use soda_identity::store::Store;
use soda_identity::wire::*;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};

fn super_dsn() -> Option<String> {
    let host = std::env::var("SODA_PG_HOST").ok()?;
    let port = std::env::var("SODA_PG_PORT").ok()?;
    let password_file = std::env::var("SODA_PG_SUPER_PASSWORD_FILE").ok()?;
    if host.is_empty() || port.is_empty() || password_file.is_empty() {
        return None;
    }
    let raw = std::fs::read_to_string(password_file).ok()?;
    let password = raw.trim();
    if password.is_empty() || password.contains(['\r', '\n', '\0']) {
        return None;
    }
    Some(format!(
        "postgres://postgres:{password}@{host}:{port}/postgres?sslmode=disable"
    ))
}

struct Ephemeral {
    dsn: String,
    super_dsn: String,
    name: String,
}

impl Ephemeral {
    fn create() -> Option<Ephemeral> {
        let super_dsn = super_dsn()?;
        let mut random = [0u8; 8];
        use std::io::Read;
        std::fs::File::open("/dev/urandom")
            .ok()?
            .read_exact(&mut random)
            .ok()?;
        let name = format!("soda_ephem_{}", hex(&random));
        let dsn = Dsn::parse(&super_dsn).ok()?;
        let mut admin = PgClient::connect(&dsn).ok()?;
        admin.simple(&format!("CREATE DATABASE \"{name}\"")).ok()?;
        let dsn = super_dsn.replace("/postgres?sslmode", &format!("/{name}?sslmode"));
        Some(Ephemeral {
            dsn,
            super_dsn,
            name,
        })
    }

    fn store(&self, key: &[u8]) -> Store {
        Store::open_encrypted(&self.dsn, key).expect("open ephemeral store")
    }
}

impl Drop for Ephemeral {
    fn drop(&mut self) {
        if let Ok(dsn) = Dsn::parse(&self.super_dsn) {
            if let Ok(mut admin) = PgClient::connect(&dsn) {
                let _ = admin.simple(&format!("DROP DATABASE IF EXISTS \"{}\"", self.name));
            }
        }
    }
}

fn hex(bytes: &[u8]) -> String {
    bytes.iter().map(|b| format!("{b:02x}")).collect()
}

fn fixture_key() -> Vec<u8> {
    vec![9u8; 32]
}

fn subscription() -> Vec<u8> {
    br#"{"schema_version":1,"providers":{"meta":{"access_token":"synthetic","api_key":"synthetic","api_base_url":"https://api.meta.ai/v1","mechanism":"oauth","obtained_via":"device_code"}}}"#.to_vec()
}

fn connection(provider: &str, owner: i64, id: &str) -> Connection {
    Connection {
        provider_id: provider.to_string(),
        id: id.to_string(),
        owner_id: owner,
        label: "synthetic".to_string(),
        email: "soda-tester@example.invalid".to_string(),
        plan: "plus".to_string(),
        generation: 1,
        state: "ready".to_string(),
    }
}

struct StubSession {
    snapshot: Mutex<Enrollment>,
    connection: Connection,
    credential: Vec<u8>,
}

impl control::EnrollmentSession for StubSession {
    fn snapshot(&self) -> Enrollment {
        self.snapshot.lock().unwrap().clone()
    }
    fn finish(&self) -> Result<(Connection, Vec<u8>), Error> {
        Ok((self.connection.clone(), self.credential.clone()))
    }
    fn close(&self) -> Result<(), Error> {
        Ok(())
    }
}

struct StubProvider {
    enrollment: Enrollment,
    connection: Connection,
    credential: Vec<u8>,
}

impl control::Provider for StubProvider {
    fn start(&self, _owner: i64) -> Result<Box<dyn control::EnrollmentSession>, Error> {
        Ok(Box::new(StubSession {
            snapshot: Mutex::new(self.enrollment.clone()),
            connection: self.connection.clone(),
            credential: self.credential.clone(),
        }))
    }
}

struct StubRuntime {
    credential: Vec<u8>,
    calls: Mutex<Vec<String>>,
}

impl control::Runtime for StubRuntime {
    fn validate(&self, _lease: &Lease) -> Result<(), Error> {
        self.calls.lock().unwrap().push("validate".to_string());
        Ok(())
    }
    fn stop(&self, _lease: &Lease) -> Result<(), Error> {
        self.calls.lock().unwrap().push("stop".to_string());
        Ok(())
    }
    fn finish(&self, _lease: &Lease) -> Result<Vec<u8>, Error> {
        self.calls.lock().unwrap().push("finish".to_string());
        Ok(self.credential.clone())
    }
}

fn stub_provider() -> StubProvider {
    StubProvider {
        enrollment: Enrollment {
            provider_id: "codex".to_string(),
            id: "enrollment-1".to_string(),
            verification_url: "https://auth.openai.com/codex/device".to_string(),
            user_code: "ABCD-1234".to_string(),
            state: "completed".to_string(),
            error: String::new(),
            connection: None,
        },
        connection: Connection {
            provider_id: String::new(),
            id: String::new(),
            owner_id: 0,
            label: String::new(),
            email: "soda-tester@example.invalid".to_string(),
            plan: "plus".to_string(),
            generation: 0,
            state: String::new(),
        },
        credential: subscription(),
    }
}

fn controller(store: Store) -> Controller {
    let mut providers: HashMap<String, Box<dyn control::Provider>> = HashMap::new();
    providers.insert("codex".to_string(), Box::new(stub_provider()));
    Controller::new(
        store,
        providers,
        Box::new(StubRuntime {
            credential: subscription(),
            calls: Mutex::new(Vec::new()),
        }),
    )
    .unwrap()
}

fn binding(kind: &str, generation: i64) -> Binding {
    Binding {
        child_id: String::new(),
        uid: 0,
        gid: 0,
        scope: String::new(),
        credential_root: String::new(),
        invocation_id: String::new(),
        kind: kind.to_string(),
        id: "execution-1".to_string(),
        project: String::new(),
        login: String::new(),
        generation,
    }
}

#[test]
fn store_round_trip() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let store = fixture.store(&fixture_key());
    let conn = connection("codex", 1, "conn-1");
    store.save_connection(&conn, &subscription()).unwrap();
    assert_eq!(
        store.connection("conn-1").unwrap().email,
        "soda-tester@example.invalid"
    );
    assert_eq!(store.credential(&conn).unwrap(), subscription());
    assert_eq!(store.connections(1).unwrap().len(), 1);
    assert_eq!(store.available(1, "project").unwrap().len(), 1);
    // A foreign actor sees the connection only through a grant, without email.
    assert!(store.available(2, "project").unwrap().is_empty());
    let grant = Grant {
        id: "grant-1".to_string(),
        connection_id: "conn-1".to_string(),
        user_id: 2,
        project_id: "project".to_string(),
        revision: 1,
        revoked: false,
    };
    store.save_grant(&grant).unwrap();
    let shared = store.available(2, "project").unwrap();
    assert_eq!(shared.len(), 1);
    assert!(shared[0].email.is_empty());
    store.revoke_grant(&grant).unwrap();
    assert!(store.available(2, "project").unwrap().is_empty());
}

#[test]
fn enrollment_to_lease_lifecycle() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = controller(fixture.store(&fixture_key()));
    let started = broker.start_enrollment(1, "codex", "synthetic").unwrap();
    assert_eq!(started.id, "enrollment-1");
    // A second enrollment while one is unretained is busy.
    assert!(broker.start_enrollment(1, "codex", "other").is_err());
    let read = broker.enrollment(1, "enrollment-1").unwrap();
    let conn = read
        .connection
        .expect("completed enrollment retains a connection");
    assert_eq!(conn.generation, 1);
    assert_eq!(conn.state, "ready");
    // Acquire, register, return.
    let deadline = UnixTime {
        sec: UnixTime::now().sec + 3600,
        nanos: 0,
    };
    let lease = broker
        .acquire(&AcquireRequest {
            repository_id: 0,
            provider_id: "codex".to_string(),
            execution_id: "execution-1".to_string(),
            actor_id: 1,
            connection_id: conn.id.clone(),
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline,
            role: String::new(),
        })
        .unwrap();
    assert_eq!(lease.connection_id, conn.id);
    // Same digest replays the recorded lease instead of reserving another.
    let replay = broker
        .acquire(&AcquireRequest {
            repository_id: 0,
            provider_id: "codex".to_string(),
            execution_id: "execution-1".to_string(),
            actor_id: 1,
            connection_id: conn.id.clone(),
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline,
            role: String::new(),
        })
        .unwrap();
    assert_eq!(replay.id, lease.id);
    let (registered, credential) = broker.register(&lease.id, &binding("factory", 1)).unwrap();
    assert_eq!(credential, subscription());
    assert!(registered.binding.is_some());
    // Owner lease listing strips bindings.
    let listed = broker.leases(1, &conn.id).unwrap();
    assert_eq!(listed.len(), 1);
    assert!(listed[0].binding.is_none());
    broker
        .return_lease(&lease.id, &binding("factory", 1), &subscription())
        .unwrap();
    // Return rotated the credential generation.
    assert_eq!(broker.connections(1).unwrap()[0].generation, 2);
    let execution = broker.get_execution("factory", "execution-1").unwrap();
    assert_eq!(execution.state, "terminal");
}

#[test]
fn muse_lease_returns_by_forget() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    // The factory seeds the muse connection from file; enrollment plays
    // no part in the lease cycle under test.
    let store = fixture.store(&fixture_key());
    let conn = connection("muse", 1, "conn-muse");
    store.save_connection(&conn, &subscription()).unwrap();
    let mut providers: HashMap<String, Box<dyn control::Provider>> = HashMap::new();
    providers.insert("muse".to_string(), Box::new(stub_provider()));
    let broker = Controller::new(
        store,
        providers,
        Box::new(StubRuntime {
            credential: subscription(),
            calls: Mutex::new(Vec::new()),
        }),
    )
    .unwrap();
    let deadline = UnixTime {
        sec: UnixTime::now().sec + 3600,
        nanos: 0,
    };
    let lease = broker
        .acquire(&AcquireRequest {
            repository_id: 0,
            provider_id: "muse".to_string(),
            execution_id: "execution-muse".to_string(),
            actor_id: 1,
            connection_id: conn.id.clone(),
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline,
            role: String::new(),
        })
        .unwrap();
    let (registered, credential) = broker.register(&lease.id, &binding("factory", 1)).unwrap();
    assert_eq!(credential, subscription());
    assert!(registered.binding.is_some());
    broker
        .return_lease(&lease.id, &binding("factory", 1), &subscription())
        .unwrap();
    // Borrow, not rotation: the connection generation is untouched, the
    // lease is forgotten, the execution is terminal.
    assert_eq!(broker.connections(1).unwrap()[0].generation, 1);
    assert!(broker.leases(1, &conn.id).unwrap().is_empty());
    let execution = broker.get_execution("factory", "execution-muse").unwrap();
    assert_eq!(execution.state, "terminal");
}

#[test]
fn close_execution_fences_late_registration() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = controller(fixture.store(&fixture_key()));
    broker.start_enrollment(1, "codex", "synthetic").unwrap();
    let read = broker.enrollment(1, "enrollment-1").unwrap();
    let conn = read.connection.unwrap();
    let deadline = UnixTime {
        sec: UnixTime::now().sec + 3600,
        nanos: 0,
    };
    let acquire = AcquireRequest {
        repository_id: 0,
        provider_id: "codex".to_string(),
        execution_id: "execution-9".to_string(),
        actor_id: 1,
        connection_id: conn.id.clone(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline,
        role: String::new(),
    };
    let lease = broker.acquire(&acquire).unwrap();
    broker.close_execution("factory", "execution-9").unwrap();
    assert!(broker.register(&lease.id, &binding("factory", 1)).is_err());
    assert!(broker.acquire(&acquire).is_err());
}

#[test]
fn revoke_retires_live_leases() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = controller(fixture.store(&fixture_key()));
    broker.start_enrollment(1, "codex", "synthetic").unwrap();
    let read = broker.enrollment(1, "enrollment-1").unwrap();
    let conn = read.connection.unwrap();
    let deadline = UnixTime {
        sec: UnixTime::now().sec + 3600,
        nanos: 0,
    };
    broker
        .acquire(&AcquireRequest {
            repository_id: 0,
            provider_id: "codex".to_string(),
            execution_id: "execution-2".to_string(),
            actor_id: 1,
            connection_id: conn.id.clone(),
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline,
            role: String::new(),
        })
        .unwrap();
    broker.revoke(1, &conn.id).unwrap();
    assert_eq!(broker.connections(1).unwrap()[0].state, "revoked");
    assert!(broker.leases(1, &conn.id).unwrap().is_empty());
}

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
    let _guard = Guard {
        shutdown: Arc::clone(&shutdown),
    };
    std::thread::scope(|scope| {
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
        // Runtime paths are denied on the admin socket.
        let (status, _) = call(
            &admin_path,
            &post("/acquire", r#"{"acquire":{"provider_id":"codex"}}"#, ""),
        );
        assert_eq!(status, 403);
        // Unknown paths are denied everywhere.
        let (status, _) = call(&runtime_path, &post("/nope", r#"{}"#, ""));
        assert_eq!(status, 403);
        shutdown.store(true, std::sync::atomic::Ordering::SeqCst);
    });
    let _ = std::fs::remove_file(&admin_path);
    let _ = std::fs::remove_file(&runtime_path);
}

// A dead listener ends serve() promptly for supervisor restart, like Go's
// Serve returning a fatal error, instead of spinning deaf forever.
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
