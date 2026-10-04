// soda-factory is the private operator interface to supervised factory runs:
// status reads recorded state, stop retires one run, and reconcile settles
// outstanding runs. It addresses the dashboard backend's operator endpoint
// over a restricted Unix socket and keeps no database of its own.

use std::env;
use std::io::{self, Read, Write};
use std::os::unix::net::UnixStream;
use std::time::Duration;

const USAGE: &str =
    "usage: soda-factory --socket PATH [--command ID] status [RUN] / stop RUN / reconcile";
const OPERATOR_PATH: &str = "/operator/factory";
const OPERATOR_HOST: &str = "soda-operator";
const RESPONSE_LIMIT: usize = 256 * 1024;
const CLIENT_TIMEOUT: Duration = Duration::from_secs(11 * 60);

fn main() {
    let args: Vec<String> = env::args().skip(1).collect();
    let mut out = io::stdout();
    if let Err(err) = run(&args, &mut out) {
        let _ = out.flush();
        eprintln!("soda-factory: {err}");
        std::process::exit(1);
    }
}

fn run(args: &[String], out: &mut dyn Write) -> Result<(), String> {
    let parsed = parse_args(args)?;
    dispatch(&parsed.socket, &parsed.command, &parsed.positional, out)
}

struct ParsedArgs {
    socket: String,
    command: String,
    positional: Vec<String>,
}

fn print_usage() {
    eprint!(
        "Usage of soda-factory:\n  -command string\n    \tclient-generated command identity for stop and reconcile\n  -socket string\n    \tprivate backend operator Unix socket\n"
    );
}

fn parse_args(args: &[String]) -> Result<ParsedArgs, String> {
    let mut socket = String::new();
    let mut command = String::new();
    let mut positional: Vec<String> = Vec::new();
    let mut i = 0;
    let mut end_of_flags = false;
    while i < args.len() {
        let arg = &args[i];
        if end_of_flags || !arg.starts_with('-') || arg == "-" {
            positional.push(arg.clone());
            i += 1;
            continue;
        }
        if arg == "--" {
            end_of_flags = true;
            i += 1;
            continue;
        }
        let flag = arg.trim_start_matches('-');
        let (name, inline) = match flag.split_once('=') {
            Some((n, v)) => (n, Some(v)),
            None => (flag, None),
        };
        if name == "h" || name == "help" {
            print_usage();
            return Err("flag: help requested".to_string());
        }
        let value = match inline {
            Some(v) => v.to_string(),
            None => {
                i += 1;
                if i >= args.len() {
                    let msg = format!("flag needs an argument: -{name}");
                    eprintln!("{msg}");
                    print_usage();
                    return Err(msg);
                }
                args[i].clone()
            }
        };
        match name {
            "socket" => socket = value,
            "command" => command = value,
            _ => {
                let msg = format!("flag provided but not defined: -{name}");
                eprintln!("{msg}");
                print_usage();
                return Err(msg);
            }
        }
        i += 1;
    }
    if socket.is_empty() {
        return Err(USAGE.to_string());
    }
    Ok(ParsedArgs {
        socket,
        command,
        positional,
    })
}

fn dispatch(
    socket: &str,
    command: &str,
    args: &[String],
    out: &mut dyn Write,
) -> Result<(), String> {
    if args.is_empty() {
        return Err(USAGE.to_string());
    }
    let mut envelope: Vec<(&str, &str)> = Vec::new();
    match args[0].as_str() {
        "status" => {
            if args.len() > 2 || (args.len() == 2 && args[1].is_empty()) {
                return Err("status takes an optional run identity".to_string());
            }
            if !command.is_empty() {
                return Err("status carries no durable command".to_string());
            }
            envelope.push(("type", "status"));
            if args.len() == 2 {
                envelope.push(("target", &args[1]));
            }
        }
        "stop" => {
            if args.len() != 2 || command.is_empty() {
                return Err("stop requires --command ID and one recorded run".to_string());
            }
            envelope.push(("type", "stop"));
            envelope.push(("command_id", command));
            envelope.push(("target", &args[1]));
        }
        "reconcile" => {
            if args.len() != 1 || command.is_empty() {
                return Err("reconcile requires --command ID and no target".to_string());
            }
            envelope.push(("type", "reconcile"));
            envelope.push(("command_id", command));
        }
        _ => return Err("unknown factory command".to_string()),
    }
    send(socket, &envelope, out)
}

/// Encode a string map as JSON exactly the way Go's encoding/json encodes a
/// map[string]string: keys sorted, `<`, `>`, `&` escaped, short escapes for
/// the common controls, `\u00xx` for the rest, plus a trailing newline.
fn encode_envelope(envelope: &[(&str, &str)]) -> Vec<u8> {
    let mut pairs: Vec<(&str, &str)> = envelope.to_vec();
    pairs.sort_by(|a, b| a.0.cmp(b.0));
    let mut body = String::from("{");
    for (n, (key, value)) in pairs.iter().enumerate() {
        if n > 0 {
            body.push(',');
        }
        push_json_string(&mut body, key);
        body.push(':');
        push_json_string(&mut body, value);
    }
    body.push_str("}\n");
    body.into_bytes()
}

fn push_json_string(out: &mut String, value: &str) {
    out.push('"');
    for ch in value.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '<' => out.push_str("\\u003c"),
            '>' => out.push_str("\\u003e"),
            '&' => out.push_str("\\u0026"),
            c if (c as u32) < 0x20 => {
                out.push_str(&format!("\\u{:04x}", c as u32));
            }
            c => out.push(c),
        }
    }
    out.push('"');
}

fn send(socket: &str, envelope: &[(&str, &str)], out: &mut dyn Write) -> Result<(), String> {
    let body = encode_envelope(envelope);
    let mut stream = UnixStream::connect(socket).map_err(|_| "operator endpoint unavailable")?;
    stream
        .set_read_timeout(Some(CLIENT_TIMEOUT))
        .map_err(|_| "operator endpoint unavailable")?;
    stream
        .set_write_timeout(Some(CLIENT_TIMEOUT))
        .map_err(|_| "operator endpoint unavailable")?;
    let request = format!(
        "POST {} HTTP/1.1\r\nHost: {}\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        OPERATOR_PATH,
        OPERATOR_HOST,
        body.len()
    );
    stream
        .write_all(request.as_bytes())
        .map_err(|_| "operator endpoint unavailable")?;
    stream
        .write_all(&body)
        .map_err(|_| "operator endpoint unavailable")?;
    let (status, data) = read_response(&mut stream)?;
    match status {
        200 | 202 => {
            out.write_all(&data)
                .map_err(|e| format!("write response: {e}"))?;
            if data.is_empty() || data[data.len() - 1] != b'\n' {
                out.write_all(b"\n")
                    .map_err(|e| format!("write response: {e}"))?;
            }
            Ok(())
        }
        404 => Err("factory run not found".to_string()),
        409 => Err("command identity reused for different content".to_string()),
        code => Err(format!("operator command failed (HTTP {code})")),
    }
}

fn read_response(stream: &mut UnixStream) -> Result<(u16, Vec<u8>), String> {
    let mut raw: Vec<u8> = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err("operator endpoint unavailable".to_string()),
            Ok(_) => raw.push(byte[0]),
            Err(_) => return Err("operator endpoint unavailable".to_string()),
        }
        if raw.len() >= 4 && raw[raw.len() - 4..] == *b"\r\n\r\n" {
            break;
        }
    }
    let head = String::from_utf8(raw).map_err(|_| "operator endpoint unavailable")?;
    let mut lines = head.split("\r\n");
    let status_line = lines.next().ok_or("operator endpoint unavailable")?;
    let status: u16 = status_line
        .split_whitespace()
        .nth(1)
        .and_then(|code| code.parse().ok())
        .ok_or("operator endpoint unavailable")?;
    let mut content_length: Option<usize> = None;
    let mut chunked = false;
    for line in lines {
        if line.is_empty() {
            break;
        }
        let (name, value) = line
            .split_once(':')
            .ok_or("operator endpoint unavailable")?;
        if name.trim().eq_ignore_ascii_case("content-length") {
            content_length = value.trim().parse().ok();
        } else if name.trim().eq_ignore_ascii_case("transfer-encoding")
            && value.to_ascii_lowercase().contains("chunked")
        {
            chunked = true;
        }
    }
    let data = if chunked {
        read_chunked(stream)?
    } else if let Some(len) = content_length {
        read_exact_limited(stream, len)?
    } else {
        read_to_end_limited(stream)?
    };
    Ok((status, data))
}

fn read_exact_limited(stream: &mut UnixStream, len: usize) -> Result<Vec<u8>, String> {
    // Mirror Go's io.ReadAll(io.LimitReader(body, RESPONSE_LIMIT + 1)): never
    // allocate past the limit, refuse short bodies (Go surfaces those as a
    // body read error), and refuse anything over the limit.
    let want = len.min(RESPONSE_LIMIT + 1);
    let mut data = vec![0u8; want];
    stream
        .read_exact(&mut data)
        .map_err(|_| "operator response exceeds limit")?;
    if len > RESPONSE_LIMIT {
        return Err("operator response exceeds limit".to_string());
    }
    Ok(data)
}

fn read_to_end_limited(stream: &mut UnixStream) -> Result<Vec<u8>, String> {
    let mut data: Vec<u8> = Vec::new();
    let mut chunk = [0u8; 8192];
    loop {
        match stream.read(&mut chunk) {
            Ok(0) => break,
            Ok(n) => {
                data.extend_from_slice(&chunk[..n]);
                if data.len() > RESPONSE_LIMIT {
                    return Err("operator response exceeds limit".to_string());
                }
            }
            Err(_) => return Err("operator response exceeds limit".to_string()),
        }
    }
    Ok(data)
}

fn read_chunked(stream: &mut UnixStream) -> Result<Vec<u8>, String> {
    let mut data: Vec<u8> = Vec::new();
    loop {
        let line = read_line(stream)?;
        let size = usize::from_str_radix(line.split(';').next().unwrap_or("").trim(), 16)
            .map_err(|_| "operator response exceeds limit")?;
        if size == 0 {
            let _ = read_line(stream);
            break;
        }
        if data.len() + size > RESPONSE_LIMIT {
            return Err("operator response exceeds limit".to_string());
        }
        let mut chunk = vec![0u8; size];
        stream
            .read_exact(&mut chunk)
            .map_err(|_| "operator response exceeds limit")?;
        data.extend_from_slice(&chunk);
        let _ = read_line(stream);
    }
    Ok(data)
}

fn read_line(stream: &mut UnixStream) -> Result<String, String> {
    let mut line: Vec<u8> = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err("operator response exceeds limit".to_string()),
            Ok(_) => line.push(byte[0]),
            Err(_) => return Err("operator response exceeds limit".to_string()),
        }
        if line.len() >= 2 && line[line.len() - 2..] == *b"\r\n" {
            line.truncate(line.len() - 2);
            break;
        }
        if line.len() > 65536 {
            return Err("operator response exceeds limit".to_string());
        }
    }
    String::from_utf8(line).map_err(|_| "operator response exceeds limit".to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
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
        let dir = env::temp_dir().join(format!("soda-factory-test-{}-{}", std::process::id(), seq));
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
        assert!(head.contains("\r\nHost: soda-operator\r\n"), "{head}");
        assert!(
            head.contains("\r\nContent-Type: application/json\r\n"),
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
        let err = run(&args(&["--socket", &socket, "status", "gone"]), &mut out)
            .expect_err("not-found maps");
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
        let err =
            run(&args(&["--socket", &socket, "status"]), &mut out).expect_err("oversize refused");
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
        let body = encode_envelope(&[("type", "a<b>&\"c\"\n")]);
        assert_eq!(
            body,
            b"{\"type\":\"a\\u003cb\\u003e\\u0026\\\"c\\\"\\n\"}\n"
        );
    }
}
