//! Fixture HTTP server and scratch directories for fetcher unit tests.
//!
//! Each test gets its own loopback server on an ephemeral port, so tests
//! stay hermetic and parallel-safe. Server threads are detached until
//! process exit; each one only ever serves its own test's routes.

use std::collections::HashMap;
use std::io::{Read, Write};
use std::net::TcpListener;
use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex};
use std::thread;

static COUNTER: AtomicU64 = AtomicU64::new(0);

#[derive(Debug, Clone)]
pub struct Observed {
    pub target: String,
    pub user_agent: String,
}

pub struct Server {
    pub base: String,
    seen: Arc<Mutex<Vec<Observed>>>,
}

impl Server {
    /// Serve `routes` (request path without query to status + body) until
    /// the test process exits. Unknown paths get a 404.
    pub fn start(routes: HashMap<String, (u16, Vec<u8>)>) -> Server {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind fixture server");
        let base = format!("http://{}", listener.local_addr().expect("fixture addr"));
        let seen = Arc::new(Mutex::new(Vec::new()));
        let seen_clone = Arc::clone(&seen);
        let routes = Arc::new(routes);
        thread::spawn(move || {
            for stream in listener.incoming() {
                let mut stream = match stream {
                    Ok(stream) => stream,
                    Err(_) => break,
                };
                let mut head = Vec::new();
                let mut buf = [0u8; 1024];
                loop {
                    match stream.read(&mut buf) {
                        Ok(0) => break,
                        Ok(n) => {
                            head.extend_from_slice(&buf[..n]);
                            if head.windows(4).any(|w| w == b"\r\n\r\n") || head.len() > 65536 {
                                break;
                            }
                        }
                        Err(_) => break,
                    }
                }
                let text = String::from_utf8_lossy(&head);
                let mut lines = text.lines();
                let target = lines
                    .next()
                    .unwrap_or("")
                    .split_whitespace()
                    .nth(1)
                    .unwrap_or("")
                    .to_string();
                let mut user_agent = String::new();
                for line in lines {
                    if let Some(value) = line
                        .strip_prefix("User-Agent:")
                        .or_else(|| line.strip_prefix("user-agent:"))
                    {
                        user_agent = value.trim().to_string();
                    }
                }
                seen_clone.lock().expect("seen").push(Observed {
                    target: target.clone(),
                    user_agent,
                });
                let path = target.split('?').next().unwrap_or("").to_string();
                let (status, body) = routes
                    .get(&path)
                    .cloned()
                    .unwrap_or((404, b"no fixture route".to_vec()));
                let reason = match status {
                    200 => "OK",
                    206 => "Partial Content",
                    404 => "Not Found",
                    500 => "Internal Server Error",
                    _ => "Status",
                };
                let response = format!(
                    "HTTP/1.1 {status} {reason}\r\nContent-Length: {}\r\nContent-Type: application/octet-stream\r\nConnection: close\r\n\r\n",
                    body.len()
                );
                let _ = stream.write_all(response.as_bytes());
                let _ = stream.write_all(&body);
                let _ = stream.flush();
            }
        });
        Server { base, seen }
    }

    pub fn seen(&self) -> Vec<Observed> {
        self.seen.lock().expect("seen").clone()
    }
}

/// Scratch directory removed when the guard drops.
pub struct TempDir {
    pub path: PathBuf,
}

impl TempDir {
    pub fn new(tag: &str) -> TempDir {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path =
            std::env::temp_dir().join(format!("soda-fetchers-{tag}-{}-{id}", std::process::id()));
        let _ = std::fs::remove_dir_all(&path);
        std::fs::create_dir_all(&path).expect("scratch dir");
        TempDir { path }
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.path);
    }
}
