use super::*;
use std::io::Read;
use std::os::unix::net::UnixListener;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::mpsc;
use std::thread;

static TEST_SOCKET_SEQ: AtomicU64 = AtomicU64::new(0);

fn args(values: &[&str]) -> Vec<String> {
    values.iter().map(|v| v.to_string()).collect()
}

/// Serve one request on a temp Unix socket, capture the raw request bytes,
/// and reply with the given status and body.
fn operator_server(status: u16, body: &[u8]) -> (String, mpsc::Receiver<Vec<u8>>) {
    let seq = TEST_SOCKET_SEQ.fetch_add(1, Ordering::SeqCst);
    let dir = Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(2)
        .expect("workspace root")
        .join(format!(
            ".artifacts/l08-l09/factory-{}-{seq}",
            std::process::id()
        ));
    let _ = std::fs::create_dir_all(&dir);
    let socket = dir.join("operator.sock");
    let _ = std::fs::remove_file(&socket);
    let listener = UnixListener::bind(&socket).expect("bind test socket");
    let (tx, rx) = mpsc::channel();
    let body = body.to_vec();
    thread::spawn(move || {
        let (mut stream, _) = listener.accept().expect("accept test connection");
        stream
            .set_read_timeout(Some(Duration::from_secs(10)))
            .expect("test read timeout");
        let mut raw: Vec<u8> = Vec::new();
        let mut byte = [0u8; 1];
        let mut content_length = 0usize;
        loop {
            let n = stream.read(&mut byte).expect("read test request");
            if n == 0 {
                break;
            }
            raw.push(byte[0]);
            if raw.len() >= 4 && raw[raw.len() - 4..] == *b"\r\n\r\n" {
                let head = String::from_utf8_lossy(&raw);
                for line in head.split("\r\n").skip(1) {
                    if let Some((name, value)) = line.split_once(':') {
                        if name.trim().eq_ignore_ascii_case("content-length") {
                            content_length = value.trim().parse().unwrap_or(0);
                        }
                    }
                }
                break;
            }
        }
        let mut rest = vec![0u8; content_length];
        stream.read_exact(&mut rest).expect("read test body");
        raw.extend_from_slice(&rest);
        let _ = tx.send(raw);
        let reason = match status {
            200 => "OK",
            202 => "Accepted",
            404 => "Not Found",
            409 => "Conflict",
            _ => "Error",
        };
        let response = format!(
            "HTTP/1.1 {} {}\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
            status,
            reason,
            body.len()
        );
        stream
            .write_all(response.as_bytes())
            .expect("write test head");
        stream.write_all(&body).expect("write test body");
    });
    (socket.to_string_lossy().into_owned(), rx)
}

fn request_parts(raw: &[u8]) -> (String, Vec<u8>) {
    let split = raw
        .windows(4)
        .position(|w| w == b"\r\n\r\n")
        .expect("request head terminator");
    (
        String::from_utf8_lossy(&raw[..split]).into_owned(),
        raw[split + 4..].to_vec(),
    )
}

#[test]
fn status_envelope_matches_go_client_bytes() {
    let (socket, rx) = operator_server(200, b"{\"runs\":[]}");
    let mut out: Vec<u8> = Vec::new();
    run(&args(&["--socket", &socket, "status"]), &mut out).expect("status succeeds");
    assert_eq!(out, b"{\"runs\":[]}\n");
    let raw = rx
        .recv_timeout(Duration::from_secs(10))
        .expect("request captured");
    let (head, body) = request_parts(&raw);
    assert!(
        head.starts_with("POST /operator/factory HTTP/1.1\r\n"),
        "{head}"
    );
    let normalized = head.to_ascii_lowercase();
    assert!(normalized.contains("\r\nhost: soda-operator\r\n"), "{head}");
    assert!(
        normalized.contains("\r\ncontent-type: application/json\r\n"),
        "{head}"
    );
    assert_eq!(body, b"{\"type\":\"status\"}\n");
}

#[test]
fn stop_and_reconcile_envelopes_carry_sorted_keys() {
    let (socket, rx) = operator_server(202, b"{}");
    let mut out: Vec<u8> = Vec::new();
    run(
        &args(&["--socket", &socket, "--command", "cmd-1", "stop", "run-9"]),
        &mut out,
    )
    .expect("stop succeeds");
    let raw = rx
        .recv_timeout(Duration::from_secs(10))
        .expect("request captured");
    let (_, body) = request_parts(&raw);
    assert_eq!(
        body,
        b"{\"command_id\":\"cmd-1\",\"target\":\"run-9\",\"type\":\"stop\"}\n"
    );

    let (socket, rx) = operator_server(202, b"{}");
    let mut out: Vec<u8> = Vec::new();
    run(
        &args(&["--socket", &socket, "--command", "cmd-2", "reconcile"]),
        &mut out,
    )
    .expect("reconcile succeeds");
    let raw = rx
        .recv_timeout(Duration::from_secs(10))
        .expect("request captured");
    let (_, body) = request_parts(&raw);
    assert_eq!(body, b"{\"command_id\":\"cmd-2\",\"type\":\"reconcile\"}\n");
}

#[test]
fn responses_map_like_the_go_client() {
    let (socket, _rx) = operator_server(409, b"conflict");
    let mut out: Vec<u8> = Vec::new();
    let err = run(
        &args(&[
            "--socket",
            &socket,
            "--command",
            &"a".repeat(32),
            "stop",
            &"b".repeat(32),
        ]),
        &mut out,
    )
    .expect_err("conflict maps");
    assert!(err.contains("different content"), "{err}");

    let (socket, _rx) = operator_server(404, b"missing");
    let mut out: Vec<u8> = Vec::new();
    let err =
        run(&args(&["--socket", &socket, "status", "gone"]), &mut out).expect_err("not-found maps");
    assert_eq!(err, "factory run not found");

    let (socket, _rx) = operator_server(500, b"boom");
    let mut out: Vec<u8> = Vec::new();
    let err =
        run(&args(&["--socket", &socket, "status"]), &mut out).expect_err("server error maps");
    assert_eq!(err, "operator command failed (HTTP 500)");
}

#[test]
fn success_appends_missing_trailing_newline() {
    let (socket, _rx) = operator_server(200, b"");
    let mut out: Vec<u8> = Vec::new();
    run(&args(&["--socket", &socket, "status"]), &mut out).expect("empty ok");
    assert_eq!(out, b"\n");

    let (socket, _rx) = operator_server(200, b"{\"runs\":[]}\n");
    let mut out: Vec<u8> = Vec::new();
    run(&args(&["--socket", &socket, "status"]), &mut out).expect("newline kept");
    assert_eq!(out, b"{\"runs\":[]}\n");
}

#[test]
fn oversized_response_is_refused() {
    let big = vec![b'x'; RESPONSE_LIMIT + 1];
    let (socket, _rx) = operator_server(200, &big);
    let mut out: Vec<u8> = Vec::new();
    let err = run(&args(&["--socket", &socket, "status"]), &mut out).expect_err("oversize refused");
    assert_eq!(err, "operator response exceeds limit");
}

#[test]
fn unreachable_socket_maps_to_unavailable() {
    let mut out: Vec<u8> = Vec::new();
    let err = run(
        &args(&["--socket", "/tmp/soda-factory-test-missing.sock", "status"]),
        &mut out,
    )
    .expect_err("missing socket maps");
    assert_eq!(err, "operator endpoint unavailable");
}

#[test]
fn local_misuse_is_rejected() {
    for case in [
        vec!["status"],
        vec!["--socket", "/tmp/x.sock"],
        vec!["--socket", "/tmp/x.sock", "run", "attempt"],
        vec!["--socket", "/tmp/x.sock", "--command", "id", "status"],
        vec!["--socket", "/tmp/x.sock", "stop", "run"],
        vec![
            "--socket",
            "/tmp/x.sock",
            "--command",
            "id",
            "reconcile",
            "run",
        ],
        vec!["--socket"],
        vec!["--socket", "/tmp/x.sock", "--bogus", "status"],
    ] {
        let mut out: Vec<u8> = Vec::new();
        assert!(
            run(&args(&case), &mut out).is_err(),
            "misuse accepted: {case:?}"
        );
    }
}

#[test]
fn envelope_escapes_match_go_encoding_json() {
    let body = encode_envelope(&[("type", "a<b>&\"c\"\n\u{2028}é\u{2029}")]);
    assert_eq!(
        body,
        "{\"type\":\"a\\u003cb\\u003e\\u0026\\\"c\\\"\\n\\u2028é\\u2029\"}\n".as_bytes()
    );
}
