//! QMP client for owned QEMU guests, mirroring
//! `internal/acceptance/qmp.go`.
//!
//! One command per connection: greet, enable capabilities, send the
//! command, then wait for the matching response id while skipping
//! async events. Deadlines arrive as an [`Instant`] instead of a
//! context; callers pass the tighter of 30s and their own phase
//! deadline, like the Go owner.

use std::io::{BufRead, BufReader, Write};
use std::os::unix::net::UnixStream;
use std::time::Instant;

use soda_json::JsonValue;

use crate::error::Error;

/// QMP message size bound: the Go owner's 4 MiB limit reader.
const MESSAGE_LIMIT: u64 = 4 << 20;

/// QEMU machine-protocol client.
pub struct QmpClient {
    /// Unix socket path.
    pub socket: String,
    /// Test dial hook, like the Go owner's `Dial` field.
    pub dial: Option<fn(&str) -> std::io::Result<UnixStream>>,
}

impl QmpClient {
    fn connect(&self) -> Result<UnixStream, Error> {
        if self.socket.is_empty() {
            return Err(Error::msg("QMP socket path is required"));
        }
        let connection = match self.dial {
            Some(dial) => dial(&self.socket),
            None => UnixStream::connect(&self.socket),
        }
        .map_err(|err| Error::msg(format!("connect QMP socket: {err}")))?;
        Ok(connection)
    }

    /// Run one QMP command, decoding the `return` payload into `result`
    /// when present. A missing result still requires a result-bearing
    /// response, like the Go owner's nil-result check.
    pub fn execute(
        &self,
        command: &str,
        id: &str,
        arguments: Option<&JsonValue>,
        result: Option<&mut JsonValue>,
        deadline: Instant,
    ) -> Result<(), Error> {
        let connection = self.connect()?;
        let timeout = deadline.saturating_duration_since(Instant::now());
        connection
            .set_read_timeout(Some(timeout))
            .and_then(|()| connection.set_write_timeout(Some(timeout)))
            .map_err(|err| Error::msg(format!("set QMP deadline: {err}")))?;
        let mut reader = BufReader::new(connection.try_clone().map_err(|err| Error::msg(format!("connect QMP socket: {err}")))?);
        let mut writer = connection;
        negotiate(&mut reader, &mut writer)?;
        let mut request = vec![("execute".to_string(), JsonValue::Str(command.to_string()))];
        if let Some(args) = arguments {
            request.push(("arguments".to_string(), args.clone()));
        }
        request.push(("id".to_string(), JsonValue::Str(id.to_string())));
        send_message(&mut writer, &JsonValue::Object(request))
            .map_err(|err| Error::msg(format!("send QMP {command}: {err}")))?;
        decode_response(&mut reader, id, result)
    }
}

fn read_message(reader: &mut BufReader<UnixStream>) -> Result<JsonValue, String> {
    let mut line = Vec::new();
    let mut total: u64 = 0;
    loop {
        let done = {
            let chunk = reader.fill_buf().map_err(|err| err.to_string())?;
            if chunk.is_empty() {
                return Err("unexpected end of QMP stream".to_string());
            }
            let mut take = chunk.len();
            let mut done = false;
            if let Some(end) = chunk.iter().position(|b| *b == b'\n') {
                take = end + 1;
                done = true;
            }
            total += take as u64;
            if total > MESSAGE_LIMIT + 1 {
                return Err("QMP message exceeds limit".to_string());
            }
            line.extend_from_slice(&chunk[..take]);
            reader.consume(take);
            done
        };
        if done {
            break;
        }
    }
    let text = std::str::from_utf8(&line).map_err(|_| "invalid QMP message".to_string())?;
    JsonValue::parse(text.trim_end()).map_err(|_| "invalid QMP message".to_string())
}

fn send_message(writer: &mut UnixStream, value: &JsonValue) -> std::io::Result<()> {
    let mut out = String::new();
    crate::jsonio::write_compact(&mut out, value);
    out.push('\n');
    writer.write_all(out.as_bytes())
}

fn negotiate(reader: &mut BufReader<UnixStream>, writer: &mut UnixStream) -> Result<(), Error> {
    let greeting = read_message(reader).map_err(|err| Error::msg(format!("read QMP greeting: {err}")))?;
    if greeting.get("QMP").is_none() {
        return Err(Error::msg("QMP greeting is missing capabilities"));
    }
    let capabilities = JsonValue::Object(vec![
        ("execute".to_string(), JsonValue::Str("qmp_capabilities".to_string())),
        ("id".to_string(), JsonValue::Str("capabilities".to_string())),
    ]);
    send_message(writer, &capabilities).map_err(|err| Error::msg(format!("enable QMP capabilities: {err}")))?;
    decode_response(reader, "capabilities", None)
}

fn decode_response(reader: &mut BufReader<UnixStream>, id: &str, result: Option<&mut JsonValue>) -> Result<(), Error> {
    loop {
        let response = read_message(reader).map_err(|err| Error::msg(format!("read QMP response {id}: {err}")))?;
        let JsonValue::Object(_) = response else {
            return Err(Error::msg(format!("read QMP response {id}: invalid QMP message")));
        };
        if response.get("id").and_then(|v| v.as_str()) != Some(id) {
            continue;
        }
        if let Some(error) = response.get("error") {
            if !matches!(error, JsonValue::Null) {
                let class = error.get("class").and_then(|v| v.as_str());
                let desc = error.get("desc").and_then(|v| v.as_str());
                match (class, desc) {
                    (Some(class), Some(desc)) => return Err(Error::msg(format!("QMP {id} failed: {class}: {desc}"))),
                    _ => return Err(Error::msg(format!("read QMP response {id}: invalid QMP message"))),
                }
            }
        }
        match response.get("return") {
            None => return Err(Error::msg("QMP response has neither result nor error")),
            Some(payload) => {
                if let Some(out) = result {
                    *out = payload.clone();
                }
                return Ok(());
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::files::TempDir;
    use std::os::unix::net::UnixListener;
    use std::sync::{Mutex, OnceLock};

    fn serve_fixture(listener: UnixListener, payload: &str) -> std::thread::JoinHandle<()> {
        let payload = payload.to_string();
        std::thread::spawn(move || {
            let (connection, _) = listener.accept().unwrap();
            let mut reader = BufReader::new(connection.try_clone().unwrap());
            let mut writer = connection;
            send_message(&mut writer, &JsonValue::Object(vec![("QMP".to_string(), JsonValue::Object(vec![]))])).unwrap();
            for _ in 0..2 {
                let request = read_message(&mut reader).unwrap();
                let id = request.get("id").and_then(|v| v.as_str()).unwrap().to_string();
                let body = if request.get("execute").and_then(|v| v.as_str()) == Some("qmp_capabilities") {
                    "{}".to_string()
                } else {
                    payload.to_string()
                };
                let response = JsonValue::Object(vec![
                    ("return".to_string(), JsonValue::parse(&body).unwrap()),
                    ("id".to_string(), JsonValue::Str(id)),
                ]);
                send_message(&mut writer, &response).unwrap();
            }
        })
    }

    /// Port of `TestQMPClientNegotiatesAndReturnsTypedResult` from `qmp_test.go`.
    #[test]
    fn negotiates_and_returns_result() {
        let dir = TempDir::new("qmp").unwrap();
        let socket = dir.join("qmp.sock").to_string_lossy().into_owned();
        let listener = UnixListener::bind(&socket).unwrap();
        let server = serve_fixture(listener, r#"{"status":"running"}"#);
        let client = QmpClient { socket, dial: None };
        let mut result = JsonValue::Null;
        client
            .execute("query-status", "status", None, Some(&mut result), Instant::now() + std::time::Duration::from_secs(30))
            .unwrap();
        assert_eq!(result.get("status").and_then(|v| v.as_str()), Some("running"));
        server.join().unwrap();
    }

    static PAIR_SERVER: OnceLock<Mutex<Option<UnixStream>>> = OnceLock::new();

    fn pair_dial(_: &str) -> std::io::Result<UnixStream> {
        let mut guard = PAIR_SERVER.get().unwrap().lock().unwrap();
        guard.take().ok_or_else(|| std::io::Error::new(std::io::ErrorKind::NotConnected, "no paired QMP server"))
    }

    /// Port of `TestQMPClientReportsNativeError` from `qmp_test.go`.
    #[test]
    fn reports_native_error() {
        let (client_end, server_end) = UnixStream::pair().unwrap();
        let _ = PAIR_SERVER.set(Mutex::new(None));
        *PAIR_SERVER.get().unwrap().lock().unwrap() = Some(client_end);
        error_over_pair(server_end)
    }

    fn error_over_pair(server_end: UnixStream) {
        let server = std::thread::spawn(move || {
            let mut reader = BufReader::new(server_end.try_clone().unwrap());
            let mut writer = server_end;
            send_message(&mut writer, &JsonValue::Object(vec![("QMP".to_string(), JsonValue::Object(vec![]))])).unwrap();
            let request = read_message(&mut reader).unwrap();
            let id = request.get("id").and_then(|v| v.as_str()).unwrap().to_string();
            let ok = JsonValue::Object(vec![
                ("return".to_string(), JsonValue::parse("{}").unwrap()),
                ("id".to_string(), JsonValue::Str(id)),
            ]);
            send_message(&mut writer, &ok).unwrap();
            let request = read_message(&mut reader).unwrap();
            let id = request.get("id").and_then(|v| v.as_str()).unwrap().to_string();
            let failure = JsonValue::Object(vec![
                (
                    "error".to_string(),
                    JsonValue::Object(vec![
                        ("class".to_string(), JsonValue::Str("GenericError".to_string())),
                        ("desc".to_string(), JsonValue::Str("rejected".to_string())),
                    ]),
                ),
                ("id".to_string(), JsonValue::Str(id)),
            ]);
            send_message(&mut writer, &failure).unwrap();
        });
        let client = QmpClient {
            socket: "ignored".to_string(),
            dial: Some(pair_dial),
        };
        let err = client
            .execute("system_powerdown", "powerdown", None, None, Instant::now() + std::time::Duration::from_secs(30))
            .unwrap_err();
        assert!(err.to_string().contains("GenericError: rejected"), "{err}");
        server.join().unwrap();
    }

    #[test]
    fn missing_greeting_capabilities_fail() {
        let dir = TempDir::new("qmp").unwrap();
        let socket = dir.join("qmp.sock").to_string_lossy().into_owned();
        let listener = UnixListener::bind(&socket).unwrap();
        let server = std::thread::spawn(move || {
            let (mut connection, _) = listener.accept().unwrap();
            send_message(&mut connection, &JsonValue::Object(vec![])).unwrap();
        });
        let client = QmpClient { socket, dial: None };
        let err = client
            .execute("query-status", "status", None, None, Instant::now() + std::time::Duration::from_secs(30))
            .unwrap_err();
        assert_eq!(err.to_string(), "QMP greeting is missing capabilities");
        server.join().unwrap();
    }
}
