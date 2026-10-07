//! QMP client for owned QEMU guests, mirroring
//! `internal/acceptance/qmp.go`.
//!
//! One command per connection: greet, enable capabilities, send the
//! command, then wait for the matching response id while skipping
//! async events. Each operation uses one finite [`Phase`] for its
//! deadline and cancellation signal, like the Go owner's context.

use std::io::{BufRead, BufReader, Write};
use std::os::fd::{FromRawFd, OwnedFd};
use std::os::unix::ffi::OsStrExt;
use std::os::unix::io::AsRawFd;
use std::os::unix::net::UnixStream;
use std::time::{Duration, Instant};

use serde::de::{IgnoredAny, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};
use serde_json::value::RawValue;

use crate::structured::Value as JsonValue;

use crate::error::Error;
use crate::process::Phase;

/// QMP message size bound: the Go owner's 4 MiB limit reader.
const MESSAGE_LIMIT: u64 = 4 << 20;

/// QEMU machine-protocol client.
pub struct QmpClient {
    /// Unix socket path.
    pub socket: String,
    /// Test dial hook, like the Go owner's `Dial` field.
    #[cfg(test)]
    pub dial: Option<fn(&str) -> std::io::Result<UnixStream>>,
}

impl QmpClient {
    fn connect(&self, phase: &Phase) -> Result<UnixStream, Error> {
        phase
            .deadline()
            .ok_or_else(|| Error::msg("QMP requires a finite phase deadline"))?;
        phase.check()?;
        if self.socket.is_empty() {
            return Err(Error::msg("QMP socket path is required"));
        }
        #[cfg(test)]
        let connection = match self.dial {
            Some(dial) => dial(&self.socket),
            None => connect_until(&self.socket, phase),
        }
        .map_err(|err| Error::msg(format!("connect QMP socket: {err}")))?;
        #[cfg(not(test))]
        let connection = connect_until(&self.socket, phase)
            .map_err(|err| Error::msg(format!("connect QMP socket: {err}")))?;
        connection
            .set_nonblocking(true)
            .map_err(|err| Error::msg(format!("set QMP nonblocking mode: {err}")))?;
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
        phase: &Phase,
    ) -> Result<(), Error> {
        phase
            .deadline()
            .ok_or_else(|| Error::msg("QMP requires a finite phase deadline"))?;
        phase.check()?;
        let connection = self.connect(phase)?;
        let mut reader = BufReader::new(
            connection
                .try_clone()
                .map_err(|err| Error::msg(format!("connect QMP socket: {err}")))?,
        );
        let mut writer = connection;
        negotiate(&mut reader, &mut writer, phase)?;
        let request = ExecuteRequest {
            execute: command,
            arguments,
            id,
        };
        send_message(&mut writer, &request, phase)
            .map_err(|err| Error::msg(format!("send QMP {command}: {err}")))?;
        decode_response(&mut reader, id, result, phase)
    }
}

fn check_deadline(deadline: Instant) -> std::io::Result<Duration> {
    let remaining = deadline.saturating_duration_since(Instant::now());
    if remaining.is_zero() {
        Err(std::io::Error::new(
            std::io::ErrorKind::TimedOut,
            "deadline expired",
        ))
    } else {
        Ok(remaining)
    }
}

fn check_phase(phase: &Phase) -> std::io::Result<Instant> {
    phase
        .check()
        .map_err(|err| std::io::Error::new(std::io::ErrorKind::TimedOut, err.to_string()))?;
    phase.deadline().ok_or_else(|| {
        std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "QMP requires a finite phase deadline",
        )
    })
}

fn wait_fd(fd: i32, events: i16, phase: &Phase) -> std::io::Result<()> {
    loop {
        let deadline = check_phase(phase)?;
        let remaining = check_deadline(deadline)?;
        let millis = remaining
            .as_millis()
            .saturating_add(u128::from(remaining.subsec_nanos() % 1_000_000 != 0));
        let timeout = millis.min(10).min(i32::MAX as u128) as i32;
        let mut pollfd = libc::pollfd {
            fd,
            events,
            revents: 0,
        };
        let rc = unsafe { libc::poll(&mut pollfd, 1, timeout) };
        if rc > 0 {
            return Ok(());
        }
        if rc == 0 {
            continue;
        }
        let err = std::io::Error::last_os_error();
        if err.kind() != std::io::ErrorKind::Interrupted {
            return Err(err);
        }
    }
}

fn connect_until(path: &str, phase: &Phase) -> std::io::Result<UnixStream> {
    use std::mem::{size_of, zeroed};
    use std::os::fd::IntoRawFd;
    let path = std::ffi::OsStr::new(path).as_bytes();
    let mut address: libc::sockaddr_un = unsafe { zeroed() };
    if path.is_empty() || path.contains(&0) || path.len() >= address.sun_path.len() {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "invalid QMP socket path",
        ));
    }
    address.sun_family = libc::AF_UNIX as libc::sa_family_t;
    for (dst, src) in address.sun_path.iter_mut().zip(path.iter().copied()) {
        *dst = src as libc::c_char;
    }
    let fd = unsafe {
        libc::socket(
            libc::AF_UNIX,
            libc::SOCK_STREAM | libc::SOCK_NONBLOCK | libc::SOCK_CLOEXEC,
            0,
        )
    };
    if fd < 0 {
        return Err(std::io::Error::last_os_error());
    }
    let owned = unsafe { OwnedFd::from_raw_fd(fd) };
    let length = (size_of::<libc::sa_family_t>() + path.len() + 1) as libc::socklen_t;
    loop {
        check_phase(phase)?;
        let rc =
            unsafe { libc::connect(fd, (&address as *const libc::sockaddr_un).cast(), length) };
        if rc == 0 {
            break;
        }
        let err = std::io::Error::last_os_error();
        match err.raw_os_error() {
            Some(libc::EINPROGRESS) | Some(libc::EALREADY) => {
                wait_fd(fd, libc::POLLOUT, phase)?;
                let mut socket_error: libc::c_int = 0;
                let mut size = size_of::<libc::c_int>() as libc::socklen_t;
                if unsafe {
                    libc::getsockopt(
                        fd,
                        libc::SOL_SOCKET,
                        libc::SO_ERROR,
                        (&mut socket_error as *mut libc::c_int).cast(),
                        &mut size,
                    )
                } != 0
                {
                    return Err(std::io::Error::last_os_error());
                }
                if socket_error != 0 {
                    return Err(std::io::Error::from_raw_os_error(socket_error));
                }
                break;
            }
            // Linux reports EAGAIN when the AF_UNIX listener backlog is full;
            // unlike EINPROGRESS, that attempt has not been queued. Retry it
            // after a short deadline-bounded pause.
            Some(libc::EAGAIN) => {
                let remaining = check_deadline(check_phase(phase)?)?;
                let millis = remaining
                    .as_millis()
                    .saturating_add(u128::from(remaining.subsec_nanos() % 1_000_000 != 0))
                    .min(10) as i32;
                unsafe {
                    libc::poll(std::ptr::null_mut(), 0, millis.max(1));
                }
            }
            Some(libc::EINTR) => continue,
            _ => return Err(err),
        }
    }
    check_phase(phase)?;
    let raw = owned.into_raw_fd();
    Ok(unsafe { UnixStream::from_raw_fd(raw) })
}

fn read_message(
    reader: &mut BufReader<UnixStream>,
    phase: &Phase,
) -> Result<Box<RawValue>, String> {
    let mut line = Vec::new();
    let mut total: u64 = 0;
    loop {
        let done = {
            check_phase(phase).map_err(|err| err.to_string())?;
            let chunk = match reader.fill_buf() {
                Ok(chunk) => chunk,
                Err(err) if err.kind() == std::io::ErrorKind::WouldBlock => {
                    wait_fd(reader.get_ref().as_raw_fd(), libc::POLLIN, phase)
                        .map_err(|err| err.to_string())?;
                    continue;
                }
                Err(err) if err.kind() == std::io::ErrorKind::Interrupted => continue,
                Err(err) => return Err(err.to_string()),
            };
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
    let value =
        serde_json::from_str(text.trim_end()).map_err(|_| "invalid QMP message".to_string())?;
    check_phase(phase).map_err(|err| err.to_string())?;
    Ok(value)
}

fn send_message<T: Serialize>(
    writer: &mut UnixStream,
    value: &T,
    phase: &Phase,
) -> std::io::Result<()> {
    check_phase(phase)?;
    let mut out = String::new();
    crate::jsonio::write_compact(&mut out, value);
    out.push('\n');
    check_phase(phase)?;
    let mut bytes = out.as_bytes();
    while !bytes.is_empty() {
        check_phase(phase)?;
        match writer.write(bytes) {
            Ok(0) => {
                return Err(std::io::Error::new(
                    std::io::ErrorKind::WriteZero,
                    "QMP socket closed",
                ))
            }
            Ok(written) => bytes = &bytes[written..],
            Err(err) if err.kind() == std::io::ErrorKind::WouldBlock => {
                wait_fd(writer.as_raw_fd(), libc::POLLOUT, phase)?
            }
            Err(err) if err.kind() == std::io::ErrorKind::Interrupted => continue,
            Err(err) => return Err(err),
        }
    }
    check_phase(phase)?;
    Ok(())
}

#[derive(Serialize)]
struct ExecuteRequest<'a> {
    execute: &'a str,
    #[serde(skip_serializing_if = "Option::is_none")]
    arguments: Option<&'a JsonValue>,
    id: &'a str,
}

#[derive(Serialize)]
struct CapabilitiesRequest<'a> {
    execute: &'a str,
    id: &'a str,
}

#[derive(Default)]
struct QmpEnvelope {
    qmp: Option<Box<RawValue>>,
    id: Option<Box<RawValue>>,
    error: Option<Box<RawValue>>,
    result: Option<Box<RawValue>>,
}

impl<'de> Deserialize<'de> for QmpEnvelope {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct EnvelopeVisitor;
        impl<'de> Visitor<'de> for EnvelopeVisitor {
            type Value = QmpEnvelope;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("a QMP message object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let mut envelope = QmpEnvelope::default();
                while let Some(key) = map.next_key::<String>()? {
                    match key.as_str() {
                        "QMP" => {
                            envelope.qmp = Some(map.next_value()?);
                        }
                        "id" => envelope.id = Some(map.next_value()?),
                        "error" => envelope.error = Some(map.next_value()?),
                        "return" => envelope.result = Some(map.next_value()?),
                        _ => {
                            let _: IgnoredAny = map.next_value()?;
                        }
                    }
                }
                Ok(envelope)
            }
        }
        deserializer.deserialize_map(EnvelopeVisitor)
    }
}

#[cfg(test)]
fn read_dynamic_message(
    reader: &mut BufReader<UnixStream>,
    phase: &Phase,
) -> Result<JsonValue, String> {
    let raw = read_message(reader, phase)?;
    JsonValue::from_raw(&raw).map_err(|_| "invalid QMP message".to_string())
}

fn negotiate(
    reader: &mut BufReader<UnixStream>,
    writer: &mut UnixStream,
    phase: &Phase,
) -> Result<(), Error> {
    let greeting = read_message(reader, phase)
        .map_err(|err| Error::msg(format!("read QMP greeting: {err}")))?;
    let greeting: QmpEnvelope = serde_json::from_str(greeting.get())
        .map_err(|_| Error::msg("read QMP greeting: invalid QMP message"))?;
    if greeting.qmp.is_none() {
        return Err(Error::msg("QMP greeting is missing capabilities"));
    }
    let capabilities = CapabilitiesRequest {
        execute: "qmp_capabilities",
        id: "capabilities",
    };
    send_message(writer, &capabilities, phase)
        .map_err(|err| Error::msg(format!("enable QMP capabilities: {err}")))?;
    decode_response(reader, "capabilities", None, phase)
}

fn decode_response(
    reader: &mut BufReader<UnixStream>,
    id: &str,
    result: Option<&mut JsonValue>,
    phase: &Phase,
) -> Result<(), Error> {
    loop {
        phase
            .check()
            .map_err(|err| Error::msg(format!("read QMP response {id}: {err}")))?;
        let raw = read_message(reader, phase)
            .map_err(|err| Error::msg(format!("read QMP response {id}: {err}")))?;
        let response: QmpEnvelope = serde_json::from_str(raw.get())
            .map_err(|_| Error::msg(format!("read QMP response {id}: invalid QMP message")))?;
        phase
            .check()
            .map_err(|err| Error::msg(format!("read QMP response {id}: {err}")))?;
        let response_id = response
            .id
            .as_ref()
            .and_then(|value| serde_json::from_str::<String>(value.get()).ok());
        if response_id.as_deref() != Some(id) {
            continue;
        }
        if let Some(error) = response.error.as_ref() {
            let error = JsonValue::from_raw(error)
                .map_err(|_| Error::msg(format!("read QMP response {id}: invalid QMP message")))?;
            phase
                .check()
                .map_err(|err| Error::msg(format!("read QMP response {id}: {err}")))?;
            if !matches!(error, JsonValue::Null) {
                let class = error.get("class").and_then(|v| v.as_str());
                let desc = error.get("desc").and_then(|v| v.as_str());
                match (class, desc) {
                    (Some(class), Some(desc)) => {
                        return Err(Error::msg(format!("QMP {id} failed: {class}: {desc}")))
                    }
                    _ => {
                        return Err(Error::msg(format!(
                            "read QMP response {id}: invalid QMP message"
                        )))
                    }
                }
            }
        }
        match response.result {
            None => return Err(Error::msg("QMP response has neither result nor error")),
            Some(payload) => {
                let payload = JsonValue::from_raw(&payload).map_err(|_| {
                    Error::msg(format!("read QMP response {id}: invalid QMP message"))
                })?;
                phase
                    .check()
                    .map_err(|err| Error::msg(format!("read QMP response {id}: {err}")))?;
                if let Some(out) = result {
                    *out = payload;
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

    fn fixture_phase() -> Phase {
        Phase::timeout(Duration::from_secs(30))
    }

    #[test]
    fn typed_envelope_uses_final_raw_duplicate_before_conversion() {
        let (mut client, server) = UnixStream::pair().unwrap();
        client
            .write_all(b"{\"id\":false,\"id\":\"status\",\"return\":false,\"return\":{\"n\":1e400,\"n\":-0}}\n")
            .unwrap();
        let mut reader = BufReader::new(server);
        let mut result = JsonValue::Null;
        decode_response(&mut reader, "status", Some(&mut result), &fixture_phase()).unwrap();
        assert_eq!(result.get("n"), Some(&JsonValue::Number("-0".to_string())));
    }

    #[test]
    fn request_envelope_order_keeps_arbitrary_argument_order_and_numbers() {
        let arguments = JsonValue::Object(vec![
            ("z".into(), JsonValue::Number("1e2".into())),
            ("a".into(), JsonValue::Number("-0".into())),
        ]);
        let request = ExecuteRequest {
            execute: "query-status",
            arguments: Some(&arguments),
            id: "status",
        };
        let mut output = String::new();
        crate::jsonio::write_compact(&mut output, &request);
        assert_eq!(
            output,
            r#"{"execute":"query-status","arguments":{"z":1e2,"a":-0},"id":"status"}"#
        );
    }

    fn execute_bounded(client: QmpClient, phase: Phase, outer: Duration) -> Result<(), String> {
        let (tx, rx) = std::sync::mpsc::channel();
        std::thread::spawn(move || {
            let result = client
                .execute("query-status", "status", None, None, &phase)
                .map_err(|err| err.to_string());
            let _ = tx.send(result);
        });
        rx.recv_timeout(outer)
            .expect("QMP operation exceeded outer test timeout")
    }

    fn serve_fixture(listener: UnixListener, payload: &str) -> std::thread::JoinHandle<()> {
        let payload = payload.to_string();
        std::thread::spawn(move || {
            let (connection, _) = listener.accept().unwrap();
            let mut reader = BufReader::new(connection.try_clone().unwrap());
            let mut writer = connection;
            send_message(
                &mut writer,
                &JsonValue::Object(vec![("QMP".to_string(), JsonValue::Object(vec![]))]),
                &fixture_phase(),
            )
            .unwrap();
            for _ in 0..2 {
                let request = read_dynamic_message(&mut reader, &fixture_phase()).unwrap();
                let id = request
                    .get("id")
                    .and_then(|v| v.as_str())
                    .unwrap()
                    .to_string();
                let body = if request.get("execute").and_then(|v| v.as_str())
                    == Some("qmp_capabilities")
                {
                    "{}".to_string()
                } else {
                    payload.to_string()
                };
                let response = JsonValue::Object(vec![
                    ("return".to_string(), JsonValue::parse(&body).unwrap()),
                    ("id".to_string(), JsonValue::Str(id)),
                ]);
                send_message(&mut writer, &response, &fixture_phase()).unwrap();
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
            .execute(
                "query-status",
                "status",
                None,
                Some(&mut result),
                &Phase::timeout(Duration::from_secs(30)),
            )
            .unwrap();
        assert_eq!(
            result.get("status").and_then(|v| v.as_str()),
            Some("running")
        );
        server.join().unwrap();
    }

    static PAIR_SERVER: OnceLock<Mutex<Option<UnixStream>>> = OnceLock::new();

    fn pair_dial(_: &str) -> std::io::Result<UnixStream> {
        let mut guard = PAIR_SERVER.get().unwrap().lock().unwrap();
        guard.take().ok_or_else(|| {
            std::io::Error::new(std::io::ErrorKind::NotConnected, "no paired QMP server")
        })
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
            send_message(
                &mut writer,
                &JsonValue::Object(vec![("QMP".to_string(), JsonValue::Object(vec![]))]),
                &fixture_phase(),
            )
            .unwrap();
            let request = read_dynamic_message(&mut reader, &fixture_phase()).unwrap();
            let id = request
                .get("id")
                .and_then(|v| v.as_str())
                .unwrap()
                .to_string();
            let ok = JsonValue::Object(vec![
                ("return".to_string(), JsonValue::parse("{}").unwrap()),
                ("id".to_string(), JsonValue::Str(id)),
            ]);
            send_message(&mut writer, &ok, &fixture_phase()).unwrap();
            let request = read_dynamic_message(&mut reader, &fixture_phase()).unwrap();
            let id = request
                .get("id")
                .and_then(|v| v.as_str())
                .unwrap()
                .to_string();
            let failure = JsonValue::Object(vec![
                (
                    "error".to_string(),
                    JsonValue::Object(vec![
                        (
                            "class".to_string(),
                            JsonValue::Str("GenericError".to_string()),
                        ),
                        ("desc".to_string(), JsonValue::Str("rejected".to_string())),
                    ]),
                ),
                ("id".to_string(), JsonValue::Str(id)),
            ]);
            send_message(&mut writer, &failure, &fixture_phase()).unwrap();
        });
        let client = QmpClient {
            socket: "ignored".to_string(),
            dial: Some(pair_dial),
        };
        let err = client
            .execute(
                "system_powerdown",
                "powerdown",
                None,
                None,
                &Phase::timeout(Duration::from_secs(30)),
            )
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
            send_message(
                &mut connection,
                &JsonValue::Object(vec![]),
                &fixture_phase(),
            )
            .unwrap();
        });
        let client = QmpClient { socket, dial: None };
        let err = client
            .execute(
                "query-status",
                "status",
                None,
                None,
                &Phase::timeout(Duration::from_secs(30)),
            )
            .unwrap_err();
        assert_eq!(err.to_string(), "QMP greeting is missing capabilities");
        server.join().unwrap();
    }

    #[test]
    fn expired_deadline_does_not_connect() {
        let dir = TempDir::new("qmp-expired").unwrap();
        let socket = dir.join("absent.sock").to_string_lossy().into_owned();
        let err = execute_bounded(
            QmpClient { socket, dial: None },
            Phase::timeout(Duration::ZERO),
            Duration::from_secs(1),
        )
        .unwrap_err();
        assert!(err.contains("deadline exceeded"), "{err}");
    }

    #[test]
    fn trickled_greeting_obeys_one_absolute_deadline() {
        let dir = TempDir::new("qmp-trickle").unwrap();
        let socket = dir.join("qmp.sock").to_string_lossy().into_owned();
        let listener = UnixListener::bind(&socket).unwrap();
        let server = std::thread::spawn(move || {
            if let Ok((mut connection, _)) = listener.accept() {
                for byte in br#"{"QMP":{}}"#.iter().copied().chain(std::iter::once(b'\n')) {
                    if connection.write_all(&[byte]).is_err() {
                        break;
                    }
                    std::thread::sleep(Duration::from_millis(25));
                }
            }
        });
        let started = Instant::now();
        let err = execute_bounded(
            QmpClient { socket, dial: None },
            Phase::timeout(Duration::from_millis(100)),
            Duration::from_secs(1),
        )
        .unwrap_err();
        assert!(err.contains("deadline exceeded"), "{err}");
        assert!(started.elapsed() < Duration::from_millis(500));
        server.join().unwrap();
    }

    #[test]
    fn cancellation_interrupts_stalled_greeting() {
        let dir = TempDir::new("qmp-cancel").unwrap();
        let socket = dir.join("qmp.sock").to_string_lossy().into_owned();
        let listener = UnixListener::bind(&socket).unwrap();
        let server = std::thread::spawn(move || {
            if let Ok((connection, _)) = listener.accept() {
                std::thread::sleep(Duration::from_millis(300));
                drop(connection);
            }
        });
        let phase = Phase::timeout(Duration::from_secs(5));
        let client_phase = phase.clone();
        let client = QmpClient { socket, dial: None };
        let (tx, rx) = std::sync::mpsc::channel();
        std::thread::spawn(move || {
            let result = client
                .execute("query-status", "status", None, None, &client_phase)
                .map_err(|err| err.to_string());
            let _ = tx.send(result);
        });
        std::thread::sleep(Duration::from_millis(50));
        let started = Instant::now();
        phase.cancel();
        let err = rx
            .recv_timeout(Duration::from_millis(250))
            .expect("QMP cancellation was not responsive")
            .unwrap_err();
        assert!(err.contains("cancel"), "{err}");
        assert!(started.elapsed() < Duration::from_millis(200));
        server.join().unwrap();
    }

    #[test]
    fn expired_phase_rejects_buffered_matching_response() {
        let (mut peer, connection) = UnixStream::pair().unwrap();
        peer.write_all(b"{\"event\":\"STOP\"}\n{\"return\":{},\"id\":\"status\"}\n")
            .unwrap();
        let mut reader = BufReader::new(connection);
        assert!(!reader.fill_buf().unwrap().is_empty());
        let phase = Phase::timeout(Duration::ZERO);
        let err = decode_response(&mut reader, "status", None, &phase).unwrap_err();
        assert!(err.to_string().contains("deadline exceeded"), "{err}");
    }

    #[test]
    fn connect_retries_when_unix_listener_backlog_is_full() {
        use std::mem::{size_of, zeroed};
        use std::os::fd::{FromRawFd, IntoRawFd, OwnedFd};

        let dir = TempDir::new("qmp-backlog").unwrap();
        let socket = dir.join("qmp.sock").to_string_lossy().into_owned();
        let path = std::ffi::OsStr::new(&socket).as_bytes();
        let mut address: libc::sockaddr_un = unsafe { zeroed() };
        address.sun_family = libc::AF_UNIX as libc::sa_family_t;
        for (dst, src) in address.sun_path.iter_mut().zip(path.iter().copied()) {
            *dst = src as libc::c_char;
        }
        let listener_fd =
            unsafe { libc::socket(libc::AF_UNIX, libc::SOCK_STREAM | libc::SOCK_CLOEXEC, 0) };
        assert!(listener_fd >= 0);
        let listener_owned = unsafe { OwnedFd::from_raw_fd(listener_fd) };
        let address_len = (size_of::<libc::sa_family_t>() + path.len() + 1) as libc::socklen_t;
        assert_eq!(
            unsafe {
                libc::bind(
                    listener_fd,
                    (&address as *const libc::sockaddr_un).cast(),
                    address_len,
                )
            },
            0
        );
        assert_eq!(unsafe { libc::listen(listener_fd, 1) }, 0);
        let listener = unsafe { UnixListener::from_raw_fd(listener_owned.into_raw_fd()) };

        let mut queued = Vec::new();
        let mut observed_full = false;
        for _ in 0..8 {
            let fd = unsafe {
                libc::socket(
                    libc::AF_UNIX,
                    libc::SOCK_STREAM | libc::SOCK_NONBLOCK | libc::SOCK_CLOEXEC,
                    0,
                )
            };
            assert!(fd >= 0);
            let owned = unsafe { OwnedFd::from_raw_fd(fd) };
            let rc = unsafe {
                libc::connect(
                    fd,
                    (&address as *const libc::sockaddr_un).cast(),
                    address_len,
                )
            };
            if rc == 0 {
                queued.push(owned);
                continue;
            }
            let err = std::io::Error::last_os_error();
            if err.raw_os_error() == Some(libc::EAGAIN) {
                observed_full = true;
                break;
            }
            panic!("pre-fill listener backlog: {err}");
        }
        assert!(
            observed_full,
            "test failed to fill the one-connection backlog"
        );

        let connect_path = socket.clone();
        let (timeout_tx, timeout_rx) = std::sync::mpsc::channel();
        std::thread::spawn(move || {
            let phase = Phase::timeout(Duration::from_millis(60));
            let result = connect_until(&connect_path, &phase).map_err(|err| err.to_string());
            let _ = timeout_tx.send(result);
        });
        let timeout = timeout_rx
            .recv_timeout(Duration::from_secs(1))
            .expect("full-backlog connect exceeded outer timeout")
            .unwrap_err();
        assert!(timeout.contains("deadline exceeded"), "{timeout}");

        let queued_count = queued.len();
        let acceptor = std::thread::spawn(move || {
            listener.set_nonblocking(true).unwrap();
            let accept_before = Instant::now() + Duration::from_secs(2);
            for _ in 0..queued_count {
                let prefilled = loop {
                    match listener.accept() {
                        Ok((connection, _)) => break connection,
                        Err(err)
                            if err.kind() == std::io::ErrorKind::WouldBlock
                                && Instant::now() < accept_before =>
                        {
                            std::thread::sleep(Duration::from_millis(5))
                        }
                        Err(err) => panic!("accept prefilled connection: {err}"),
                    }
                };
                drop(prefilled);
            }
            let retried = loop {
                match listener.accept() {
                    Ok((connection, _)) => break connection,
                    Err(err)
                        if err.kind() == std::io::ErrorKind::WouldBlock
                            && Instant::now() < accept_before =>
                    {
                        std::thread::sleep(Duration::from_millis(5))
                    }
                    Err(err) => panic!("accept retried connection: {err}"),
                }
            };
            drop(retried);
        });
        let connect_path = socket.clone();
        let phase = Phase::timeout(Duration::from_secs(2));
        let (tx, rx) = std::sync::mpsc::channel();
        std::thread::spawn(move || {
            let result = connect_until(&connect_path, &phase).map_err(|err| err.to_string());
            let _ = tx.send(result);
        });
        let connected = rx
            .recv_timeout(Duration::from_secs(3))
            .expect("backlog retry exceeded outer timeout")
            .unwrap();
        drop(connected);
        drop(queued);
        acceptor.join().unwrap();
    }

    #[test]
    fn negotiation_and_response_share_one_phase_budget() {
        let dir = TempDir::new("qmp-total-deadline").unwrap();
        let socket = dir.join("qmp.sock").to_string_lossy().into_owned();
        let listener = UnixListener::bind(&socket).unwrap();
        let server = std::thread::spawn(move || {
            let (mut connection, _) = listener.accept().unwrap();
            let mut reader = BufReader::new(connection.try_clone().unwrap());
            send_message(
                &mut connection,
                &JsonValue::Object(vec![("QMP".into(), JsonValue::Object(vec![]))]),
                &fixture_phase(),
            )
            .unwrap();
            let capability = read_dynamic_message(&mut reader, &fixture_phase()).unwrap();
            std::thread::sleep(Duration::from_millis(60));
            let cap_id = capability
                .get("id")
                .and_then(|value| value.as_str())
                .unwrap()
                .to_owned();
            send_message(
                &mut connection,
                &JsonValue::Object(vec![
                    ("return".into(), JsonValue::Object(vec![])),
                    ("id".into(), JsonValue::Str(cap_id)),
                ]),
                &fixture_phase(),
            )
            .unwrap();
            let command = read_dynamic_message(&mut reader, &fixture_phase()).unwrap();
            let command_id = command
                .get("id")
                .and_then(|value| value.as_str())
                .unwrap()
                .to_owned();
            std::thread::sleep(Duration::from_millis(60));
            let _ = send_message(
                &mut connection,
                &JsonValue::Object(vec![
                    ("return".into(), JsonValue::Object(vec![])),
                    ("id".into(), JsonValue::Str(command_id)),
                ]),
                &fixture_phase(),
            );
        });
        let started = Instant::now();
        let err = execute_bounded(
            QmpClient { socket, dial: None },
            Phase::timeout(Duration::from_millis(100)),
            Duration::from_secs(1),
        )
        .unwrap_err();
        assert!(err.contains("deadline exceeded"), "{err}");
        assert!(started.elapsed() < Duration::from_millis(500));
        server.join().unwrap();
    }

    #[test]
    fn matching_response_after_buffered_event_storm_is_found() {
        let dir = TempDir::new("qmp-events").unwrap();
        let socket = dir.join("qmp.sock").to_string_lossy().into_owned();
        let listener = UnixListener::bind(&socket).unwrap();
        let server = std::thread::spawn(move || {
            let (mut connection, _) = listener.accept().unwrap();
            let mut reader = BufReader::new(connection.try_clone().unwrap());
            send_message(
                &mut connection,
                &JsonValue::Object(vec![("QMP".into(), JsonValue::Object(vec![]))]),
                &fixture_phase(),
            )
            .unwrap();
            let request = read_dynamic_message(&mut reader, &fixture_phase()).unwrap();
            let id = request
                .get("id")
                .and_then(|v| v.as_str())
                .unwrap()
                .to_owned();
            send_message(
                &mut connection,
                &JsonValue::Object(vec![
                    ("return".into(), JsonValue::Object(vec![])),
                    ("id".into(), JsonValue::Str(id)),
                ]),
                &fixture_phase(),
            )
            .unwrap();
            let _ = read_dynamic_message(&mut reader, &fixture_phase()).unwrap();
            let mut stream = String::new();
            for _ in 0..2000 {
                stream.push_str("{\"event\":\"STOP\"}\n");
            }
            stream.push_str("{\"return\":{\"status\":\"running\"},\"id\":\"status\"}\n");
            connection.write_all(stream.as_bytes()).unwrap();
        });
        let mut result = JsonValue::Null;
        QmpClient { socket, dial: None }
            .execute(
                "query-status",
                "status",
                None,
                Some(&mut result),
                &Phase::timeout(Duration::from_secs(2)),
            )
            .unwrap();
        assert_eq!(
            result.get("status").and_then(|v| v.as_str()),
            Some("running")
        );
        server.join().unwrap();
    }

    #[test]
    fn blocked_partial_request_write_obeys_deadline() {
        let dir = TempDir::new("qmp-write").unwrap();
        let socket = dir.join("qmp.sock").to_string_lossy().into_owned();
        let listener = UnixListener::bind(&socket).unwrap();
        let server = std::thread::spawn(move || {
            let (mut connection, _) = listener.accept().unwrap();
            let mut reader = BufReader::new(connection.try_clone().unwrap());
            send_message(
                &mut connection,
                &JsonValue::Object(vec![("QMP".into(), JsonValue::Object(vec![]))]),
                &fixture_phase(),
            )
            .unwrap();
            let request = read_dynamic_message(&mut reader, &fixture_phase()).unwrap();
            let id = request
                .get("id")
                .and_then(|v| v.as_str())
                .unwrap()
                .to_owned();
            send_message(
                &mut connection,
                &JsonValue::Object(vec![
                    ("return".into(), JsonValue::Object(vec![])),
                    ("id".into(), JsonValue::Str(id)),
                ]),
                &fixture_phase(),
            )
            .unwrap();
            std::thread::sleep(Duration::from_millis(250));
        });
        let client_socket = socket.clone();
        let (tx, rx) = std::sync::mpsc::channel();
        let phase = Phase::timeout(Duration::from_millis(100));
        std::thread::spawn(move || {
            let client = QmpClient {
                socket: client_socket,
                dial: None,
            };
            let large =
                JsonValue::Object(vec![("data".into(), JsonValue::Str("x".repeat(2 << 20)))]);
            let result = client
                .execute("query-status", "status", Some(&large), None, &phase)
                .map_err(|e| e.to_string());
            let _ = tx.send(result);
        });
        let err = rx
            .recv_timeout(Duration::from_secs(2))
            .expect("QMP write exceeded outer test timeout")
            .unwrap_err();
        assert!(err.contains("deadline exceeded"), "{err}");
        server.join().unwrap();
    }
}
