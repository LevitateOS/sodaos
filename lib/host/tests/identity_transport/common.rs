use std::io::{Read, Write};
use std::os::unix::net::{UnixListener, UnixStream};
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::{mpsc, Arc};
use std::time::{Duration, Instant};

use crate::terminal;

static SOCK_COUNTER: AtomicU64 = AtomicU64::new(0);

fn sock_path(tag: &str) -> String {
    let id = SOCK_COUNTER.fetch_add(1, Ordering::SeqCst);
    let dir = "target/iclient-oracle";
    std::fs::create_dir_all(dir).unwrap();
    format!("{dir}/{tag}-{}-{id}.sock", std::process::id())
}

fn read_http_request(stream: &mut UnixStream) -> Vec<u8> {
    stream
        .set_read_timeout(Some(Duration::from_secs(10)))
        .unwrap();
    let mut raw = Vec::new();
    let mut buf = [0u8; 4096];
    let mut head_end = None;
    let mut length = 0usize;
    loop {
        let n = stream.read(&mut buf).unwrap();
        assert!(n > 0, "stub: connection closed mid-request");
        raw.extend_from_slice(&buf[..n]);
        if head_end.is_none() {
            if let Some(i) = find_crlf2(&raw) {
                head_end = Some(i + 4);
                let head = String::from_utf8_lossy(&raw[..i + 4]).into_owned();
                for line in head.split("\r\n") {
                    if let Some(v) = line
                        .strip_prefix("Content-Length:")
                        .or_else(|| line.strip_prefix("content-length:"))
                    {
                        length = v.trim().parse().unwrap();
                    }
                }
            }
        }
        if let Some(end) = head_end {
            if raw.len() >= end + length {
                break;
            }
        }
    }
    raw
}

fn find_crlf2(raw: &[u8]) -> Option<usize> {
    raw.windows(4).position(|w| w == b"\r\n\r\n")
}

/// Scripted fake broker: every connection's raw request bytes are captured
/// and answered by `respond`. Stops promptly on drop (no hanging joins).
pub(super) struct FakeBroker {
    pub(super) path: String,
    stop: Arc<AtomicBool>,
    handle: Option<std::thread::JoinHandle<()>>,
    captured: mpsc::Receiver<Vec<u8>>,
}

impl FakeBroker {
    pub(super) fn start<F>(tag: &str, respond: F) -> Self
    where
        F: Fn(&[u8]) -> Vec<u8> + Send + Sync + 'static,
    {
        let path = sock_path(tag);
        let _ = std::fs::remove_file(&path);
        let listener = UnixListener::bind(&path).unwrap();
        listener.set_nonblocking(true).unwrap();
        let respond = Arc::new(respond);
        let (tx, rx) = mpsc::channel();
        let stop = Arc::new(AtomicBool::new(false));
        let stop_clone = Arc::clone(&stop);
        let handle = std::thread::spawn(move || {
            while !stop_clone.load(Ordering::SeqCst) {
                let (mut stream, _) = match listener.accept() {
                    Ok(pair) => pair,
                    Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {
                        std::thread::sleep(Duration::from_millis(5));
                        continue;
                    }
                    Err(_) => break,
                };
                stream
                    .set_read_timeout(Some(Duration::from_secs(10)))
                    .unwrap();
                let raw = read_http_request(&mut stream);
                let reply = respond(&raw);
                let _ = tx.send(raw);
                let _ = stream.write_all(&reply);
            }
        });
        FakeBroker {
            path,
            stop,
            handle: Some(handle),
            captured: rx,
        }
    }

    pub(super) fn request(&self) -> Vec<u8> {
        self.captured
            .recv_timeout(Duration::from_secs(15))
            .expect("stub: no request arrived")
    }

    pub(super) fn request_count(&self) -> usize {
        let mut n = 0;
        while self.captured.try_recv().is_ok() {
            n += 1;
        }
        n
    }
}

impl Drop for FakeBroker {
    fn drop(&mut self) {
        self.stop.store(true, Ordering::SeqCst);
        if let Some(handle) = self.handle.take() {
            let _ = handle.join();
        }
        let _ = std::fs::remove_file(&self.path);
    }
}

pub(super) fn json_reply(status: u16, reason: &str, body: &str) -> Vec<u8> {
    format!(
        "HTTP/1.1 {status} {reason}\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
        body.len()
    )
    .into_bytes()
}

pub(super) fn error_reply(status: u16, reason: &str, code: &str) -> Vec<u8> {
    format!(
        "HTTP/1.1 {status} {reason}\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{code}",
        code.len()
    )
    .into_bytes()
}

pub(super) fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

pub(super) fn full_request(path: &str, body: &str) -> Vec<u8> {
    format!(
        "POST {path} HTTP/1.1\r\nHost: soda-identity\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
        body.len()
    )
    .into_bytes()
}

pub(super) const LEASE_JSON: &str = r#"{"id":"lease-1","connection_id":"conn","generation":1,"actor_id":"2","project_id":"project","execution_id":"exec","kind":"factory","provider_id":"codex","deadline":"2030-01-01T00:00:00Z","grant_id":"grant-1","grant_revision":3,"binding":{"kind":"factory","id":"container","project":"","login":"","generation":1}}"#;

pub(super) fn full_binding() -> terminal::Binding {
    terminal::Binding {
        child_id: "child".to_string(),
        uid: 1000,
        gid: 1000,
        scope: "muse-project".to_string(),
        credential_root: "/run/cred".to_string(),
        invocation_id: "inv".to_string(),
        kind: "terminal".to_string(),
        id: "term".to_string(),
        project: "proj".to_string(),
        login: "login".to_string(),
        generation: 4,
    }
}

pub(super) const DELIVERY_JSON: &str = r#"{"lease":{"id":"lease-9","connection_id":"conn","generation":4,"actor_id":"2","project_id":"proj","execution_id":"exec","kind":"terminal","provider_id":"muse","deadline":"2030-01-01T00:00:00Z"},"credential":"e30="}"#;

/// A valid lease body of exactly `size` bytes, padded in `project_id`.
pub(super) fn sized_lease(size: usize) -> String {
    let head = r#"{"id":"l","provider_id":"codex","connection_id":"c","generation":1,"actor_id":"2","project_id":""#;
    let tail = r#"","execution_id":"e","kind":"factory","deadline":"2030-01-01T00:00:00Z"}"#;
    assert!(size > head.len() + tail.len());
    format!("{head}{}{tail}", "p".repeat(size - head.len() - tail.len()))
}
