use std::fs;
use std::io::{Read, Write};
use std::net::TcpListener;
use std::os::unix::fs::PermissionsExt;
use std::path::Path;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::Duration;

pub(crate) static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

pub(crate) fn test_root() -> std::path::PathBuf {
    let seq = TEST_SEQ.fetch_add(1, Ordering::SeqCst);
    let dir = std::env::temp_dir().join(format!("soda-setup-test-{}-{seq}", std::process::id()));
    fs::create_dir_all(&dir).expect("test root");
    dir
}

pub(crate) struct Stub {
    pub(crate) url: String,
    pub(crate) calls: Arc<Mutex<Vec<String>>>,
}

pub(crate) fn stub_server(admin: bool, deny_current: bool, token: &str) -> Stub {
    let listener = TcpListener::bind("127.0.0.1:0").expect("bind stub");
    let url = format!("http://{}", listener.local_addr().expect("stub addr"));
    let calls: Arc<Mutex<Vec<String>>> = Arc::new(Mutex::new(Vec::new()));
    let calls_clone = Arc::clone(&calls);
    let token = token.to_string();
    let token_clone = token.clone();
    thread::spawn(move || {
        listener.set_nonblocking(false).ok();
        // Serve until the test ends; each connection carries one request.
        for stream in listener.incoming().take(6) {
            let mut stream = match stream {
                Ok(s) => s,
                Err(_) => break,
            };
            stream.set_read_timeout(Some(Duration::from_secs(10))).ok();
            let mut raw: Vec<u8> = Vec::new();
            let mut byte = [0u8; 1];
            let mut head = String::new();
            loop {
                match stream.read(&mut byte) {
                    Ok(0) => break,
                    Ok(_) => raw.push(byte[0]),
                    Err(_) => break,
                }
                if raw.len() >= 4 && raw[raw.len() - 4..] == *b"\r\n\r\n" {
                    head = String::from_utf8_lossy(&raw).into_owned();
                    break;
                }
                if raw.len() > 65536 {
                    break;
                }
            }
            let mut lines = head.split("\r\n");
            let request_line = lines.next().unwrap_or("").to_string();
            let mut authorized = false;
            for line in lines {
                if let Some((name, value)) = line.split_once(':') {
                    if name.trim().eq_ignore_ascii_case("authorization") {
                        authorized = value.trim() == format!("token {token_clone}");
                    }
                }
            }
            let mut parts = request_line.split_whitespace();
            let method = parts.next().unwrap_or("").to_string();
            let path = parts.next().unwrap_or("").to_string();
            calls_clone
                .lock()
                .expect("calls")
                .push(format!("{method} {path}"));
            let (code, body) = if !authorized {
                (401, "unauthorized".to_string())
            } else {
                match format!("{method} {path}").as_str() {
                    "GET /api/v1/user" => {
                        if deny_current {
                            (403, token_clone.clone())
                        } else {
                            (200, format!("{{\"id\": 42, \"login\": \"soda-tester\", \"is_admin\": {admin}}}"))
                        }
                    }
                    "DELETE /api/v1/user/token" => (204, String::new()),
                    _ => (404, "unexpected endpoint".to_string()),
                }
            };
            let reason = match code {
                200 => "OK",
                204 => "No Content",
                401 => "Unauthorized",
                403 => "Forbidden",
                _ => "Not Found",
            };
            let response = format!(
                "HTTP/1.1 {code} {reason}\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
                body.len()
            );
            let _ = stream.write_all(response.as_bytes());
        }
    });
    Stub { url, calls }
}

pub(crate) fn write_token(dir: &Path) -> std::path::PathBuf {
    let path = dir.join("operator-input");
    fs::write(&path, "synthetic-bootstrap-token-not-for-retention\n").expect("token");
    fs::set_permissions(&path, fs::Permissions::from_mode(0o600)).expect("token mode");
    path
}
