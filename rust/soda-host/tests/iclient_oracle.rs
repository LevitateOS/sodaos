//! Oracle tests for the PR26 daemon-foundation modules: the identity
//! broker client (`iclient`) and the daemon config loader (`iconfig`).
//!
//! The modules compile via `#[path]` includes (the adapter PR wires them
//! into `lib.rs`); `crate::` paths inside them resolve through the
//! re-exports below. Request bodies are pinned against bytes captured
//! from the live Go client; config decisions against live `loadConfig`.

use soda_host::{domain, json, muse, net, pfactory, ssh, terminal};

#[path = "../src/iclient.rs"]
mod iclient;
#[path = "../src/iconfig.rs"]
mod iconfig;

use std::io::{Read, Write};
use std::os::unix::net::{UnixListener, UnixStream};
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::{mpsc, Arc};
use std::time::{Duration, Instant};

use iclient::{execution_is_terminal, BrokerClient, Execution};
use iconfig::load_config;

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
struct FakeBroker {
    path: String,
    stop: Arc<AtomicBool>,
    handle: Option<std::thread::JoinHandle<()>>,
    captured: mpsc::Receiver<Vec<u8>>,
}

impl FakeBroker {
    fn start<F>(tag: &str, respond: F) -> Self
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

    fn request(&self) -> Vec<u8> {
        self.captured
            .recv_timeout(Duration::from_secs(15))
            .expect("stub: no request arrived")
    }

    fn request_count(&self) -> usize {
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

fn json_reply(status: u16, reason: &str, body: &str) -> Vec<u8> {
    format!(
        "HTTP/1.1 {status} {reason}\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
        body.len()
    )
    .into_bytes()
}

fn error_reply(status: u16, reason: &str, code: &str) -> Vec<u8> {
    format!(
        "HTTP/1.1 {status} {reason}\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{code}",
        code.len()
    )
    .into_bytes()
}

fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

fn full_request(path: &str, body: &str) -> Vec<u8> {
    format!(
        "POST {path} HTTP/1.1\r\nHost: soda-identity\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
        body.len()
    )
    .into_bytes()
}

const LEASE_JSON: &str = r#"{"id":"lease-1","connection_id":"conn","generation":1,"actor_id":"2","project_id":"project","execution_id":"exec","kind":"factory","provider_id":"codex","deadline":"2030-01-01T00:00:00Z","grant_id":"grant-1","grant_revision":3,"binding":{"kind":"factory","id":"container","project":"","login":"","generation":1}}"#;

#[test]
fn acquire_body_matches_go_oracle() {
    let broker = FakeBroker::start("acquire", |_| json_reply(200, "OK", LEASE_JSON));
    let client = BrokerClient::new(&broker.path);
    let req = terminal::AcquireRequest {
        repository_id: 7,
        provider_id: "codex".to_string(),
        execution_id: "exec".to_string(),
        actor_id: 2,
        connection_id: "conn".to_string(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline_secs: 1893553445,
        deadline_nanos: 123456000,
        role: "role-a".to_string(),
    };
    let lease = client.acquire(&req, deadline()).unwrap();
    // Body captured from the live Go client; only the header framing is
    // this client's documented minimal shape.
    let body = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"\",\"label\":\"\",\"project_id\":\"\",\"acquire\":{\"repository_id\":\"7\",\"provider_id\":\"codex\",\"execution_id\":\"exec\",\"actor_id\":\"2\",\"connection_id\":\"conn\",\"project_id\":\"project\",\"kind\":\"factory\",\"deadline\":\"2030-01-02T03:04:05.123456Z\",\"role\":\"role-a\"}}\n";
    assert_eq!(body.len(), 280);
    assert_eq!(broker.request(), full_request("/acquire", body));
    assert_eq!(lease.id, "lease-1");
    assert_eq!(lease.actor_id, 2);
    assert_eq!(lease.grant_id, "grant-1");
    assert_eq!(lease.grant_revision, 3);
    assert_eq!(lease.deadline_raw, "2030-01-01T00:00:00Z");
    assert!(lease.deadline.is_some());
    let binding = lease.binding.expect("binding decoded");
    assert_eq!(binding.id, "container");
    assert_eq!(binding.generation, 1);
}

#[test]
fn acquire_omits_zero_repository_and_role() {
    let broker = FakeBroker::start("acquire2", |_| json_reply(200, "OK", LEASE_JSON));
    let client = BrokerClient::new(&broker.path);
    let req = terminal::AcquireRequest {
        provider_id: "muse".to_string(),
        execution_id: "exec-2".to_string(),
        actor_id: 3,
        connection_id: "conn-2".to_string(),
        project_id: "proj-2".to_string(),
        kind: "terminal".to_string(),
        deadline_secs: 1907050150,
        ..terminal::AcquireRequest::default()
    };
    client.acquire(&req, deadline()).unwrap();
    let body = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"\",\"label\":\"\",\"project_id\":\"\",\"acquire\":{\"provider_id\":\"muse\",\"execution_id\":\"exec-2\",\"actor_id\":\"3\",\"connection_id\":\"conn-2\",\"project_id\":\"proj-2\",\"kind\":\"terminal\",\"deadline\":\"2030-06-07T08:09:10Z\"}}\n";
    assert_eq!(body.len(), 240);
    assert_eq!(broker.request(), full_request("/acquire", body));
}

fn full_binding() -> terminal::Binding {
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

const DELIVERY_JSON: &str = r#"{"lease":{"id":"lease-9","connection_id":"conn","generation":4,"actor_id":"2","project_id":"proj","execution_id":"exec","kind":"terminal","provider_id":"muse","deadline":"2030-01-01T00:00:00Z"},"credential":"e30="}"#;

#[test]
fn register_full_binding_matches_go_oracle() {
    let broker = FakeBroker::start("register", |_| json_reply(200, "OK", DELIVERY_JSON));
    let client = BrokerClient::new(&broker.path);
    let delivery = client
        .register("lease-9", &full_binding(), deadline())
        .unwrap();
    let body = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\",\"binding\":{\"child_id\":\"child\",\"uid\":1000,\"gid\":1000,\"scope\":\"muse-project\",\"credential_root\":\"/run/cred\",\"invocation_id\":\"inv\",\"kind\":\"terminal\",\"id\":\"term\",\"project\":\"proj\",\"login\":\"login\",\"generation\":4}}\n";
    assert_eq!(body.len(), 282);
    assert_eq!(broker.request(), full_request("/register", body));
    assert_eq!(delivery.credential, Some(b"{}".to_vec()));
    assert_eq!(delivery.lease.id, "lease-9");
    assert_eq!(delivery.lease.kind, "terminal");
}

#[test]
fn register_minimal_binding_matches_go_oracle() {
    let broker = FakeBroker::start("register2", |_| json_reply(200, "OK", DELIVERY_JSON));
    let client = BrokerClient::new(&broker.path);
    let binding = terminal::Binding {
        kind: "factory".to_string(),
        id: "container".to_string(),
        generation: 1,
        ..terminal::Binding::default()
    };
    client.register("lease-9", &binding, deadline()).unwrap();
    let body = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\",\"binding\":{\"kind\":\"factory\",\"id\":\"container\",\"project\":\"\",\"login\":\"\",\"generation\":1}}\n";
    assert_eq!(body.len(), 161);
    assert_eq!(broker.request(), full_request("/register", body));
}

#[test]
fn reconcile_end_available_bodies_match_go_oracle() {
    let broker = FakeBroker::start("simple", |raw| {
        let head = String::from_utf8_lossy(raw).into_owned();
        if head.starts_with("POST /available ") {
            // Full connection records, including a lossless >2^53 `,string`
            // owner; the client projects id/provider/state only.
            json_reply(
                200,
                "OK",
                r#"[{"provider_id":"muse","id":"conn-1","owner_id":"9007199254740993","label":"sub","email":"e","plan":"plus","generation":7,"state":"ready"},{"provider_id":"codex","id":"conn-2","owner_id":"2","label":"","email":"","plan":"","generation":1,"state":"reauth"}]"#,
            )
        } else {
            json_reply(200, "OK", "{}")
        }
    });
    let client = BrokerClient::new(&broker.path);
    client.reconcile_lease("lease-9", deadline()).unwrap();
    let reconcile = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\"}\n";
    assert_eq!(reconcile.len(), 76);
    assert_eq!(
        broker.request(),
        full_request("/reconcile-lease", reconcile)
    );
    client.end_lease(2, "lease-9", deadline()).unwrap();
    let end = "{\"provider_id\":\"\",\"owner_id\":\"2\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\"}\n";
    assert_eq!(end.len(), 76);
    assert_eq!(broker.request(), full_request("/lease/end", end));
    let connections = client.available(2, "project", deadline()).unwrap();
    let available =
        "{\"provider_id\":\"\",\"owner_id\":\"2\",\"id\":\"\",\"label\":\"\",\"project_id\":\"project\"}\n";
    assert_eq!(available.len(), 76);
    assert_eq!(broker.request(), full_request("/available", available));
    assert_eq!(
        connections,
        vec![
            muse::MuseConnection {
                id: "conn-1".to_string(),
                provider_id: "muse".to_string(),
                state: "ready".to_string(),
            },
            muse::MuseConnection {
                id: "conn-2".to_string(),
                provider_id: "codex".to_string(),
                state: "reauth".to_string(),
            },
        ]
    );
}

#[test]
fn return_execution_bodies_match_go_oracle() {
    let broker = FakeBroker::start("retexec", |raw| {
        let head = String::from_utf8_lossy(raw).into_owned();
        if head.starts_with("POST /execution/get ") {
            json_reply(
                200,
                "OK",
                r#"{"binding":{"kind":"factory","id":"exec-9","project":"","login":"","generation":1},"kind":"factory","execution_id":"exec-9","digest":"d","state":"terminal","lease_id":"lease-1"}"#,
            )
        } else {
            json_reply(200, "OK", "{}")
        }
    });
    let client = BrokerClient::new(&broker.path);
    let binding = terminal::Binding {
        kind: "factory".to_string(),
        id: "container".to_string(),
        generation: 1,
        ..terminal::Binding::default()
    };
    client
        .return_lease("lease-9", &binding, b"{}", deadline())
        .unwrap();
    let with_cred = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\",\"binding\":{\"kind\":\"factory\",\"id\":\"container\",\"project\":\"\",\"login\":\"\",\"generation\":1},\"credential\":\"e30=\"}\n";
    assert_eq!(with_cred.len(), 181);
    assert_eq!(broker.request(), full_request("/return", with_cred));
    client
        .return_lease("lease-9", &binding, b"", deadline())
        .unwrap();
    let without_cred = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\",\"binding\":{\"kind\":\"factory\",\"id\":\"container\",\"project\":\"\",\"login\":\"\",\"generation\":1}}\n";
    assert_eq!(without_cred.len(), 161);
    assert_eq!(broker.request(), full_request("/return", without_cred));
    let execution = client
        .get_execution("factory", "exec-9", deadline())
        .unwrap();
    let exec_body = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"\",\"label\":\"\",\"project_id\":\"\",\"kind\":\"factory\",\"execution_id\":\"exec-9\"}\n";
    assert_eq!(exec_body.len(), 110);
    assert_eq!(broker.request(), full_request("/execution/get", exec_body));
    assert_eq!(execution.kind, "factory");
    assert_eq!(execution.execution_id, "exec-9");
    assert_eq!(execution.digest, "d");
    assert_eq!(execution.state, "terminal");
    assert_eq!(execution.lease_id, "lease-1");
    assert_eq!(execution.binding.as_ref().unwrap().id, "exec-9");
    assert!(execution_is_terminal(&execution));
    client
        .close_execution("factory", "exec-9", deadline())
        .unwrap();
    assert_eq!(
        broker.request(),
        full_request("/execution/close", exec_body)
    );
}

#[test]
fn execution_is_terminal_matches_factory() {
    // `internal/host/project/factory.go` settles custody exactly when the
    // execution state is `terminal`; every other state (or error) retries.
    for (state, terminal) in [
        ("terminal", true),
        ("live", false),
        ("pending", false),
        ("", false),
        ("Terminal", false),
    ] {
        let e = Execution {
            state: state.to_string(),
            ..Execution::default()
        };
        assert_eq!(execution_is_terminal(&e), terminal, "state {state:?}");
    }
}

#[test]
fn deadline_format_matches_go_time_json() {
    // (secs, nanos, Go `time.Time` JSON): epoch, leap day, year starts,
    // pre-epoch, fraction trimming and nanosecond rollover.
    let cases = [
        (0, 0, "1970-01-01T00:00:00Z"),
        (1582934400, 0, "2020-02-29T00:00:00Z"),
        (1893456000, 0, "2030-01-01T00:00:00Z"),
        (2147483647, 0, "2038-01-19T03:14:07Z"),
        (-1, 0, "1969-12-31T23:59:59Z"),
        (1893456000, 100000000, "2030-01-01T00:00:00.1Z"),
        (1893456000, 123000000, "2030-01-01T00:00:00.123Z"),
        (1893456000, 123456789, "2030-01-01T00:00:00.123456789Z"),
        (1893455999, 1000000000, "2030-01-01T00:00:00Z"),
    ];
    let broker = FakeBroker::start("dates", |_| json_reply(200, "OK", LEASE_JSON));
    let client = BrokerClient::new(&broker.path);
    for (secs, nanos, expected) in cases {
        let req = terminal::AcquireRequest {
            deadline_secs: secs,
            deadline_nanos: nanos,
            ..terminal::AcquireRequest::default()
        };
        client.acquire(&req, deadline()).unwrap();
        let raw = broker.request();
        let text = String::from_utf8(raw).unwrap();
        assert!(
            text.contains(&format!("\"deadline\":\"{expected}\"")),
            "{secs}.{nanos}: {text}"
        );
    }
}

#[test]
fn html_escaping_matches_go_encoder() {
    let broker = FakeBroker::start("escape", |_| json_reply(200, "OK", "[]"));
    let client = BrokerClient::new(&broker.path);
    client.available(2, "a&b<c>d\"q\\", deadline()).unwrap();
    let body = "{\"provider_id\":\"\",\"owner_id\":\"2\",\"id\":\"\",\"label\":\"\",\"project_id\":\"a\\u0026b\\u003cc\\u003ed\\\"q\\\\\"}\n";
    assert_eq!(broker.request(), full_request("/available", body));
}

#[test]
fn error_codes_map_like_go() {
    // (wire code, expected error): the `stale`/`missing` errors carry the
    // adapter's substrings plus the exact Go typed-error text.
    let cases = [
        ("denied", "identity authority denied"),
        ("busy", "subscription is in use"),
        ("stale", "identity generation changed (stale)"),
        ("reauth", "subscription requires reconnection"),
        ("missing", "identity execution missing (not found)"),
        ("nope", "identity operation failed"),
        ("", "identity operation failed"),
        ("  stale\n", "identity generation changed (stale)"),
        ("denied and more", "identity operation failed"),
    ];
    for (code, expected) in cases {
        let owned = code.to_string();
        let broker = FakeBroker::start("err", move |_| error_reply(403, "Forbidden", &owned));
        let client = BrokerClient::new(&broker.path);
        let err = client.available(1, "p", deadline()).unwrap_err();
        assert_eq!(err, expected, "code {code:?}");
    }
    // Go keys on `!= 200`, not on 4xx/5xx: a 201 with a JSON body still
    // fails through the error decoder.
    let broker = FakeBroker::start("err201", |_| json_reply(201, "Created", "{}"));
    let client = BrokerClient::new(&broker.path);
    assert_eq!(
        client.available(1, "p", deadline()).unwrap_err(),
        "identity operation failed"
    );
    // ... and a 200 with a non-list body fails in the data decoder,
    // proving the 200 path was taken.
    let broker = FakeBroker::start("err200", |_| json_reply(200, "OK", "{}"));
    let client = BrokerClient::new(&broker.path);
    let err = client.available(1, "p", deadline()).unwrap_err();
    assert!(err.contains("cannot unmarshal object"), "{err}");
}

#[test]
fn substring_contract_holds() {
    for (code, needle) in [
        ("denied", "denied"),
        ("missing", "not found"),
        ("stale", "stale"),
    ] {
        let owned = code.to_string();
        let broker = FakeBroker::start("substr", move |_| error_reply(403, "Forbidden", &owned));
        let client = BrokerClient::new(&broker.path);
        let err = client.reconcile_lease("l", deadline()).unwrap_err();
        assert!(err.contains(needle), "{code:?} -> {err:?}");
    }
}

/// A valid lease body of exactly `size` bytes, padded in `project_id`.
fn sized_lease(size: usize) -> String {
    let head = r#"{"id":"l","provider_id":"codex","connection_id":"c","generation":1,"actor_id":"2","project_id":""#;
    let tail = r#"","execution_id":"e","kind":"factory","deadline":"2030-01-01T00:00:00Z"}"#;
    assert!(size > head.len() + tail.len());
    format!("{head}{}{tail}", "p".repeat(size - head.len() - tail.len()))
}

#[test]
fn response_limit_is_512kib() {
    const LIMIT: usize = 512 * 1024;
    // Exactly at the limit decodes; one byte over fails like Go's
    // `LimitReader(body, 512<<10+1)` length check.
    for (size, ok) in [(LIMIT, true), (LIMIT + 1, false)] {
        let body = sized_lease(size);
        assert_eq!(body.len(), size);
        let broker = FakeBroker::start("limit", move |_| json_reply(200, "OK", &body));
        let client = BrokerClient::new(&broker.path);
        let req = terminal::AcquireRequest {
            deadline_secs: 1907050150,
            ..terminal::AcquireRequest::default()
        };
        let out = client.acquire(&req, deadline());
        assert_eq!(out.is_ok(), ok, "size {size}");
        if !ok {
            assert_eq!(out.unwrap_err(), "identity response exceeds limit");
        }
    }
    // A hostile Content-Length never allocates: the client reads only
    // limit+1 bytes, then reports the violation without hanging.
    let broker = FakeBroker::start("huge", |_| {
        format!(
            "HTTP/1.1 200 OK\r\nContent-Length: 100000000\r\nConnection: close\r\n\r\n{}",
            "x".repeat(LIMIT + 1)
        )
        .into_bytes()
    });
    let client = BrokerClient::new(&broker.path);
    let req = terminal::AcquireRequest {
        deadline_secs: 1907050150,
        ..terminal::AcquireRequest::default()
    };
    assert_eq!(
        client.acquire(&req, deadline()).unwrap_err(),
        "identity response exceeds limit"
    );
}

#[test]
fn strict_response_decode() {
    // Unknown fields, trailing data and mistyped `,string`/integer fields
    // all fail, like Go's `DisallowUnknownFields` binding.
    let cases = [
        (
            r#"{"id":"l","bogus":1,"connection_id":"c","generation":1,"actor_id":"2","project_id":"p","execution_id":"e","kind":"factory","provider_id":"x","deadline":"2030-01-01T00:00:00Z"}"#,
            "unknown field",
        ),
        (
            r#"{"id":"l","connection_id":"c","generation":1,"actor_id":"2","project_id":"p","execution_id":"e","kind":"factory","provider_id":"x","deadline":"2030-01-01T00:00:00Z"} trailing"#,
            "trailing",
        ),
        (
            r#"{"id":"l","connection_id":"c","generation":1,"actor_id":"x","project_id":"p","execution_id":"e","kind":"factory","provider_id":"x","deadline":"2030-01-01T00:00:00Z"}"#,
            "actor_id",
        ),
        (
            r#"{"id":"l","connection_id":"c","generation":1.5,"actor_id":"2","project_id":"p","execution_id":"e","kind":"factory","provider_id":"x","deadline":"2030-01-01T00:00:00Z"}"#,
            "generation",
        ),
        (
            r#"{"id":"l","connection_id":"c","generation":1,"actor_id":"2","project_id":"p","execution_id":"e","kind":"factory","provider_id":"x","deadline":"not-a-time"}"#,
            "deadline",
        ),
        (r#"[]"#, "object"),
    ];
    for (body, needle) in cases {
        let owned = body.to_string();
        let broker = FakeBroker::start("strict", move |_| json_reply(200, "OK", &owned));
        let client = BrokerClient::new(&broker.path);
        let req = terminal::AcquireRequest {
            deadline_secs: 1907050150,
            ..terminal::AcquireRequest::default()
        };
        let err = client.acquire(&req, deadline()).unwrap_err();
        assert!(err.contains(needle), "{body:?} -> {err:?}");
    }
    // Available: element-level strictness too.
    for (body, needle) in [
        (
            r#"[{"provider_id":"muse","id":"c","owner_id":"1","label":"","email":"","plan":"","generation":1,"state":"ready","bogus":1}]"#,
            "unknown field",
        ),
        (
            r#"[{"provider_id":"muse","id":"c","owner_id":"nope","label":"","email":"","plan":"","generation":1,"state":"ready"}]"#,
            "owner_id",
        ),
        (
            r#"[{"provider_id":"muse","id":"c","owner_id":"1","label":"","email":"","plan":"","generation":"1","state":"ready"}]"#,
            "generation",
        ),
    ] {
        let owned = body.to_string();
        let broker = FakeBroker::start("strict2", move |_| json_reply(200, "OK", &owned));
        let client = BrokerClient::new(&broker.path);
        let err = client.available(1, "p", deadline()).unwrap_err();
        assert!(err.contains(needle), "{body:?} -> {err:?}");
    }
}

#[test]
fn deadline_and_transport_failures() {
    // An expired deadline fails before connecting.
    let broker = FakeBroker::start("expired", |_| json_reply(200, "OK", "{}"));
    let client = BrokerClient::new(&broker.path);
    let err = client
        .reconcile_lease("l", Instant::now() - Duration::from_secs(1))
        .unwrap_err();
    assert_eq!(err, "identity broker unavailable");
    assert_eq!(broker.request_count(), 0);
    // A missing socket fails the same way Go's dial error does.
    let ghost = BrokerClient::new("target/iclient-oracle/no-such-socket.sock");
    assert_eq!(
        ghost.reconcile_lease("l", deadline()).unwrap_err(),
        "identity broker unavailable"
    );
    // A silent broker hits the socket timeout. The stub sleeps past the
    // client deadline but briefly, so the drop-time join stays fast.
    let broker = FakeBroker::start("silent", |_| {
        std::thread::sleep(Duration::from_secs(2));
        json_reply(200, "OK", "{}")
    });
    let client = BrokerClient::new(&broker.path);
    let err = client
        .reconcile_lease("l", Instant::now() + Duration::from_millis(300))
        .unwrap_err();
    assert_eq!(err, "identity broker unavailable");
    // Malformed framing fails as unavailable, like Go's Transport errors.
    for (tag, reply) in [
        ("garbage", b"this is not http\r\n\r\n".to_vec()),
        (
            "badlen",
            b"HTTP/1.1 200 OK\r\nContent-Length: nope\r\n\r\n{}".to_vec(),
        ),
        (
            "duplen",
            b"HTTP/1.1 200 OK\r\nContent-Length: 2\r\nContent-Length: 2\r\n\r\n{}".to_vec(),
        ),
        (
            "short",
            b"HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n{}".to_vec(),
        ),
        ("empty", b"".to_vec()),
    ] {
        let broker = FakeBroker::start(tag, move |_| reply.clone());
        let client = BrokerClient::new(&broker.path);
        assert_eq!(
            client.reconcile_lease("l", deadline()).unwrap_err(),
            "identity broker unavailable",
            "case {tag}",
        );
    }
}

#[test]
fn framing_edge_cases_match_go_transport() {
    // Close-delimited body without Content-Length.
    let broker = FakeBroker::start("close", |_| {
        b"HTTP/1.1 200 OK\r\nConnection: close\r\n\r\n{}".to_vec()
    });
    BrokerClient::new(&broker.path)
        .reconcile_lease("l", deadline())
        .unwrap();
    // Chunked body.
    let broker = FakeBroker::start("chunked", |_| {
        b"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n1\r\n{\r\n1\r\n}\r\n0\r\n\r\n"
            .to_vec()
    });
    BrokerClient::new(&broker.path)
        .reconcile_lease("l", deadline())
        .unwrap();
    // Bare-LF head, which Go's textproto tolerates.
    let broker = FakeBroker::start("barelf", |_| {
        b"HTTP/1.1 200 OK\nContent-Length: 2\n\n{}".to_vec()
    });
    BrokerClient::new(&broker.path)
        .reconcile_lease("l", deadline())
        .unwrap();
    // Chunk extensions and mixed-case framing names.
    let broker = FakeBroker::start("chunkext", |_| {
        b"HTTP/1.1 200 OK\r\ntransfer-encoding: Chunked\r\n\r\n2;ext=1\r\n{}\r\n0\r\n\r\n".to_vec()
    });
    BrokerClient::new(&broker.path)
        .reconcile_lease("l", deadline())
        .unwrap();
}

// ---------- daemon config goldens ----------

const GOLDENS: &str = "tests/data/iconfig";
const PROJECT_IMAGE: &str =
    "sha256:1111111111111111111111111111111111111111111111111111111111111111";
const TAILNET_IMAGE: &str =
    "sha256:2222222222222222222222222222222222222222222222222222222222222222";

fn golden(name: &str) -> String {
    format!("{GOLDENS}/{name}")
}

#[test]
fn golden_valid_configs() {
    let c = load_config(&golden("valid-full.json"), "").unwrap();
    assert_eq!(
        c.muse_sha256,
        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    );
    assert_eq!(c.muse_version, "1.2.3");
    assert_eq!(c.muse_socket, "/run/soda-muse/launch.sock");
    assert_eq!(c.identity_socket, "/run/soda/identity.sock");
    assert_eq!(c.codex_harness, "/usr/libexec/soda/codex");
    assert_eq!(
        c.codex_harness_sha256,
        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
    );
    assert_eq!(c.codex_harness_version, "0.153.4");
    assert!(c.tailnet_management);
    assert_eq!(
        c.tailnet_image,
        "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
    );
    assert_eq!(
        c.image,
        "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
    );
    assert_eq!(c.network, "soda-net");
    assert_eq!(c.subnet, "10.89.0.0/24");
    assert_eq!(c.bridge, "soda-br0");
    let c = load_config(&golden("valid-minimal.json"), "").unwrap();
    assert_eq!(c.image, "img");
    assert_eq!(c.network, "soda-net");
    assert_eq!(c.subnet, "10.89.0.0/24");
    assert_eq!(c.bridge, "soda-br0");
    assert!(!c.tailnet_management);
    assert!(c.muse_sha256.is_empty());
    assert!(c.codex_harness.is_empty());
}

#[test]
fn golden_decode_errors() {
    // Unknown fields rejected; types bound like `encoding/json` (same
    // messages modulo this crate's `decode request: ` prefix).
    let err = load_config(&golden("unknown-field.json"), "").unwrap_err();
    assert!(err.contains(r#"unknown field "bogus""#), "{err}");
    let err = load_config(&golden("bad-type-bool.json"), "").unwrap_err();
    assert!(
        err.contains("cannot unmarshal string")
            && err.contains("Config.tailnet_management")
            && err.contains("of type bool"),
        "{err}"
    );
    let err = load_config(&golden("bad-type-string.json"), "").unwrap_err();
    assert!(
        err.contains("cannot unmarshal number into")
            && err.contains("Config.image")
            && err.contains("of type string"),
        "{err}"
    );
    // Empty input fails exactly like Go's `io.EOF`.
    assert_eq!(load_config(&golden("empty.json"), "").unwrap_err(), "EOF");
    // Missing files fail at the read, carrying the path.
    let err = load_config(&golden("does-not-exist.json"), "").unwrap_err();
    assert!(
        err.starts_with("read tests/data/iconfig/does-not-exist.json: "),
        "{err}"
    );
}

#[test]
fn golden_go_framing_parity() {
    // Go's single `Decode` ignores trailing data: accepted, like Go.
    let c = load_config(&golden("trailing.json"), "").unwrap();
    assert_eq!(c.image, "img");
    // Duplicate fields resolve last-wins, like `encoding/json`.
    let c = load_config(&golden("duplicates.json"), "").unwrap();
    assert_eq!(c.image, "img");
    // A literal `null` decodes as a no-op and reaches validation, like Go
    // (whose subnet error differs only in stdlib text).
    let err = load_config(&golden("null.json"), "").unwrap_err();
    assert_eq!(err, net::parse_prefix("").err().unwrap());
}

#[test]
fn golden_release_overlay() {
    let release = golden("release.json");
    let c = load_config(&golden("overlay-base.json"), &release).unwrap();
    assert_eq!(c.image, PROJECT_IMAGE);
    assert_eq!(c.tailnet_image, TAILNET_IMAGE);
    assert!(c.tailnet_management);
    // Without management the companion image stays empty.
    let c = load_config(&golden("overlay-nomgmt.json"), &release).unwrap();
    assert_eq!(c.image, PROJECT_IMAGE);
    assert!(c.tailnet_image.is_empty());
    // Saved selections conflicting with the release fail loudly.
    for name in ["conflict-image.json", "conflict-tailnet.json"] {
        assert_eq!(
            load_config(&golden(name), &release).unwrap_err(),
            "saved image selection conflicts with appliance release; explicit migration required",
            "{name}"
        );
    }
    // Unreadable or invalid payloads fail alike.
    for release in [golden("bad-release.json"), golden("does-not-exist.json")] {
        assert_eq!(
            load_config(&golden("overlay-base.json"), &release).unwrap_err(),
            "immutable appliance image defaults unavailable",
            "{release}"
        );
    }
    // An empty release path disables the overlay entirely.
    let c = load_config(&golden("valid-minimal.json"), "").unwrap();
    assert_eq!(c.image, "img");
}

#[test]
fn golden_validators() {
    const MUSE: &str = "explicit muse socket, broker socket and release digest required";
    const IDENTITY: &str = "explicit identity runtime socket and verified harness required";
    const STAGED: &str = "invalid staged harness version";
    const TAILNET: &str = "invalid immutable Tailnet companion configuration";
    const NATIVE: &str = "invalid native runtime configuration";
    // (golden, expected error); every message matches Go byte for byte
    // except the subnet case, which carries this crate's prefix message.
    let cases = [
        ("muse-relative-socket.json", MUSE),
        ("muse-bad-digest.json", MUSE),
        ("muse-no-version.json", MUSE),
        ("muse-bad-base.json", MUSE),
        ("muse-relative-identity.json", MUSE),
        ("identity-relative-harness.json", IDENTITY),
        ("identity-bad-sha.json", IDENTITY),
        ("identity-bad-version.json", STAGED),
        ("tailnet-no-mgmt.json", TAILNET),
        ("tailnet-bad-ref.json", TAILNET),
        ("bad-network.json", NATIVE),
        ("bad-bridge.json", NATIVE),
        ("empty-image.json", NATIVE),
        ("dash-image.json", NATIVE),
    ];
    for (name, expected) in cases {
        assert_eq!(
            load_config(&golden(name), "").unwrap_err(),
            expected,
            "{name}"
        );
    }
    assert_eq!(
        load_config(&golden("bad-subnet.json"), "").unwrap_err(),
        net::parse_prefix("nope").err().unwrap()
    );
    // Empty selectors skip their whole stage, like Go.
    load_config(&golden("muse-skipped.json"), "").unwrap();
    load_config(&golden("identity-skipped.json"), "").unwrap();
}

#[test]
fn golden_validation_order() {
    // Each golden fails two stages; the earlier stage's error must win,
    // proving muse < identity < subnet < tailnet < network order.
    assert_eq!(
        load_config(&golden("order-muse-before-subnet.json"), "").unwrap_err(),
        "explicit muse socket, broker socket and release digest required"
    );
    assert_eq!(
        load_config(&golden("order-identity-before-tailnet.json"), "").unwrap_err(),
        "explicit identity runtime socket and verified harness required"
    );
    assert_eq!(
        load_config(&golden("order-subnet-before-network.json"), "").unwrap_err(),
        net::parse_prefix("nope").err().unwrap()
    );
    assert_eq!(
        load_config(&golden("order-tailnet-before-network.json"), "").unwrap_err(),
        "invalid immutable Tailnet companion configuration"
    );
}

#[test]
fn crate_helpers_match_go_validators() {
    // The validators `iconfig` reuses (not reimplements) decide exactly
    // like their Go regexes on boundary inputs.
    assert!(domain::valid_image_ref(&format!(
        "sha256:{}",
        "c".repeat(64)
    )));
    assert!(domain::valid_image_ref(&"c".repeat(64)));
    assert!(!domain::valid_image_ref("sha256:xyz"));
    assert!(domain::valid_container_id(&"a".repeat(64)));
    assert!(!domain::valid_container_id(&"A".repeat(64)));
    assert!(domain::valid_login("soda-net"));
    assert!(!domain::valid_login("9bad"));
    assert!(!domain::valid_login(&"a".repeat(32)));
    assert!(pfactory::valid_harness_version("0.153.4"));
    assert!(!pfactory::valid_harness_version("!!!"));
    assert!(net::parse_prefix("10.89.0.0/24").is_ok());
    assert!(net::parse_prefix("nope").is_err());
    assert!(net::parse_prefix("10.0.0.1/24").is_ok());
    // `,string` decoding behind the connection projection.
    assert_eq!(
        terminal::parse_string_i64("9007199254740993"),
        Some(9007199254740993)
    );
    assert_eq!(terminal::parse_string_i64(""), None);
    // JSON quoting behind request encoding.
    assert_eq!(json::quote("a&b"), "\"a\\u0026b\"");
}
