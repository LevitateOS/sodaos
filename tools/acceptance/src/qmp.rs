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
    let mut out = serde_json::to_string(value).expect("QMP message serializes");
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
