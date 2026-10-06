// Admin/runtime HTTP admission and dead-listener integration scenarios,
// split from tests/broker.rs (A-R02c). Fixture support lives in
// tests/common.
mod common;

use common::{controller, fixture_key, Ephemeral};
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
