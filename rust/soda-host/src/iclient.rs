//! Identity broker client: Rust port of the `internal/identity/client`
//! calls the PR26 runtime adapter needs, over the private Unix HTTP socket.
//!
//! Request paths, JSON body shapes and response limits match the Go client
//! byte for byte (request bodies were captured from a live Go probe and are
//! pinned in `tests/iclient_oracle.rs`). Only the HTTP header framing is
//! minimal — `Host`/`Content-Type`/`Content-Length`/`Connection: close`,
//! the same shape as `soda-identity`'s own `HostClient` — instead of Go's
//! `net/http` default headers; the broker accepts any framing, and sending
//! Go's `Accept-Encoding: gzip` would obligate a gzip decoder.
//!
//! Deadlines arrive as `Instant`s and are enforced through socket timeouts.
//! Error mapping mirrors `decodeError`: the broker's `denied`/`busy`/
//! `stale`/`reauth`/`missing` codes map to the Go typed-error strings,
//! except `stale` and `missing` carry an explicit `(stale)`/`(not found)`
//! marker — the Go messages (`identity generation changed`, `identity
//! execution missing`) lack the substrings the adapter matches on.
//!
//! The `/identity/finish` 384 KiB cap lives on the host-daemon callback
//! client (`internal/host/identity.go`), not on any broker-client call:
//! every method here uses the Go client's 512 KiB response limit.

use std::io::{Read, Write};
use std::os::unix::net::UnixStream;
use std::time::Instant;

use crate::json::{self, Kind, Spec, Value};
use crate::muse::MuseConnection;
use crate::texec::{parse_string_i64, AcquireRequest, Binding, Delivery, Lease};

/// `decodeResponse` limit: `512<<10`, probed with one byte of slack.
pub const RESPONSE_LIMIT: usize = 512 * 1024;
/// `decodeError` reads the first 64 body bytes.
const ERROR_PREFIX_LIMIT: usize = 64;
/// Response head cap, matching the sibling broker callback client.
const HEAD_LIMIT: usize = 65536;

/// `identity.ErrBusy`: no `texec` constant exists (nothing else maps it).
const ERR_BUSY: &str = "subscription is in use";

/// Private Unix HTTP broker client (`identity/client.Client`).
pub struct BrokerClient {
    socket_path: String,
}

/// Broker execution identity (`identity.Execution`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Execution {
    pub binding: Option<Binding>,
    pub kind: String,
    pub execution_id: String,
    pub digest: String,
    pub state: String,
    pub lease_id: String,
}

/// `identity.ExecutionTerminal`: the factory treats a terminal execution as
/// terminally settled — custody converged without further broker calls
/// (`internal/host/project/factory.go`: a `Return` failure with a terminal
/// execution still reports the stop as returned).
pub fn execution_is_terminal(e: &Execution) -> bool {
    e.state == "terminal"
}

fn unavailable() -> String {
    "identity broker unavailable".to_string()
}

/// `time.Time` JSON encoding for a UTC instant: RFC 3339 with trailing
/// zero nanoseconds trimmed, exactly like Go's `RFC3339Nano` layout.
fn format_go_time(secs: i64, nanos: u32) -> String {
    let secs = secs + i64::from(nanos / 1_000_000_000);
    let nanos = nanos % 1_000_000_000;
    let days = secs.div_euclid(86400);
    let clock = secs.rem_euclid(86400);
    let (year, month, day) = civil_from_days(days);
    let mut out = if (0..10000).contains(&year) {
        format!(
            "{year:04}-{month:02}-{day:02}T{:02}:{:02}:{:02}",
            clock / 3600,
            (clock % 3600) / 60,
            clock % 60
        )
    } else {
        format!(
            "{year:+05}-{month:02}-{day:02}T{:02}:{:02}:{:02}",
            clock / 3600,
            (clock % 3600) / 60,
            clock % 60
        )
    };
    if nanos != 0 {
        let mut frac = format!("{nanos:09}");
        while frac.ends_with('0') {
            frac.pop();
        }
        out.push('.');
        out.push_str(&frac);
    }
    out.push('Z');
    out
}

/// Days since the Unix epoch to civil date (Howard Hinnant's algorithm).
fn civil_from_days(days: i64) -> (i64, u32, u32) {
    let z = days + 719468;
    let era = z.div_euclid(146097);
    let doe = (z - era * 146097) as u32;
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let year = yoe as i64 + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let day = doy - (153 * mp + 2) / 5 + 1;
    let month = if mp < 10 { mp + 3 } else { mp - 9 };
    (if month <= 2 { year + 1 } else { year }, month, day)
}

/// `identity.AcquireRequest` JSON, field order included.
fn encode_acquire(req: &AcquireRequest) -> String {
    let mut out = String::from("{");
    let mut first = true;
    let mut field = |out: &mut String, name: &str, value: &str| {
        if !first {
            out.push(',');
        }
        first = false;
        out.push_str(&json::quote(name));
        out.push(':');
        out.push_str(value);
    };
    if req.repository_id != 0 {
        field(
            &mut out,
            "repository_id",
            &json::quote(&req.repository_id.to_string()),
        );
    }
    field(&mut out, "provider_id", &json::quote(&req.provider_id));
    field(&mut out, "execution_id", &json::quote(&req.execution_id));
    field(
        &mut out,
        "actor_id",
        &json::quote(&req.actor_id.to_string()),
    );
    field(&mut out, "connection_id", &json::quote(&req.connection_id));
    field(&mut out, "project_id", &json::quote(&req.project_id));
    field(&mut out, "kind", &json::quote(&req.kind));
    field(
        &mut out,
        "deadline",
        &json::quote(&format_go_time(req.deadline_secs, req.deadline_nanos)),
    );
    if !req.role.is_empty() {
        field(&mut out, "role", &json::quote(&req.role));
    }
    out.push('}');
    out
}

/// `identity.Request` envelope: the five always-present fields in struct
/// order, then at most one `omitempty` payload. `kind`/`execution_id`
/// carry the execution calls; `nested` carries `acquire`/`binding`.
fn encode_request(
    owner_id: i64,
    id: &str,
    project_id: &str,
    kind: &str,
    execution_id: &str,
    nested: Option<(&str, &str)>,
    credential: Option<&str>,
) -> String {
    let mut out = String::from("{\"provider_id\":\"\",\"owner_id\":\"");
    out.push_str(&owner_id.to_string());
    out.push_str("\",\"id\":");
    out.push_str(&json::quote(id));
    out.push_str(",\"label\":\"\",\"project_id\":");
    out.push_str(&json::quote(project_id));
    if !kind.is_empty() {
        out.push_str(",\"kind\":");
        out.push_str(&json::quote(kind));
    }
    if !execution_id.is_empty() {
        out.push_str(",\"execution_id\":");
        out.push_str(&json::quote(execution_id));
    }
    if let Some((name, value)) = nested {
        out.push(',');
        out.push_str(&json::quote(name));
        out.push(':');
        out.push_str(value);
    }
    if let Some(data) = credential {
        out.push_str(",\"credential\":");
        out.push_str(&json::quote(data));
    }
    out.push_str("}\n");
    out
}

fn map_error(body: &[u8]) -> String {
    let prefix = &body[..body.len().min(ERROR_PREFIX_LIMIT)];
    let code = std::str::from_utf8(prefix)
        .unwrap_or("")
        .trim_matches(|c: char| c.is_ascii_whitespace());
    match code {
        "denied" => crate::texec::ERR_DENIED.to_string(),
        "busy" => ERR_BUSY.to_string(),
        "stale" => format!("{} (stale)", crate::texec::ERR_STALE),
        "reauth" => crate::texec::ERR_UNCERTAIN.to_string(),
        "missing" => format!("{} (not found)", crate::texec::ERR_NOT_FOUND),
        _ => "identity operation failed".to_string(),
    }
}

/// One response line (`\r\n`-terminated), without the terminator.
fn read_line(stream: &mut UnixStream, cap: &mut usize) -> Result<Vec<u8>, String> {
    let mut line = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err(unavailable()),
            Ok(_) => {
                if *cap == 0 {
                    return Err(unavailable());
                }
                *cap -= 1;
                line.push(byte[0]);
                if byte[0] == b'\n' {
                    if line.ends_with(b"\r\n") {
                        line.truncate(line.len() - 2);
                    } else {
                        // Go's textproto tolerates a bare line feed.
                        line.pop();
                    }
                    return Ok(line);
                }
            }
            Err(_) => return Err(unavailable()),
        }
    }
}

fn read_chunked(stream: &mut UnixStream) -> Result<Vec<u8>, String> {
    let mut body = Vec::new();
    let mut cap = HEAD_LIMIT;
    loop {
        let line = read_line(stream, &mut cap)?;
        let size_text = line.split(|b| *b == b';').next().unwrap_or(&[]);
        let size_text = std::str::from_utf8(size_text).map_err(|_| unavailable())?;
        let size = usize::from_str_radix(size_text.trim(), 16).map_err(|_| unavailable())?;
        if size == 0 {
            let _ = read_line(stream, &mut cap)?;
            return Ok(body);
        }
        if body.len() + size > RESPONSE_LIMIT + 1 {
            // Enough to report the limit violation; the socket is dropped.
            let want = RESPONSE_LIMIT + 1 - body.len();
            body.resize(RESPONSE_LIMIT + 1, 0);
            let start = body.len() - want;
            stream
                .read_exact(&mut body[start..])
                .map_err(|_| unavailable())?;
            return Ok(body);
        }
        let start = body.len();
        body.resize(start + size, 0);
        stream
            .read_exact(&mut body[start..])
            .map_err(|_| unavailable())?;
        let crlf = read_line(stream, &mut cap)?;
        if !crlf.is_empty() {
            return Err(unavailable());
        }
    }
}

fn read_response(stream: &mut UnixStream) -> Result<(u16, Vec<u8>), String> {
    let mut cap = HEAD_LIMIT;
    let status_line = read_line(stream, &mut cap)?;
    let status_text = std::str::from_utf8(&status_line).map_err(|_| unavailable())?;
    let status: u16 = status_text
        .split(' ')
        .nth(1)
        .and_then(|code| code.parse::<u16>().ok())
        .ok_or_else(unavailable)?;
    let mut content_length: Option<usize> = None;
    let mut chunked = false;
    loop {
        let line = read_line(stream, &mut cap)?;
        if line.is_empty() {
            break;
        }
        let (name, mut value) = match line.iter().position(|b| *b == b':') {
            Some(i) => (&line[..i], &line[i + 1..]),
            None => continue,
        };
        while value.first() == Some(&b' ') || value.first() == Some(&b'\t') {
            value = &value[1..];
        }
        while value.last() == Some(&b' ') || value.last() == Some(&b'\t') {
            value = &value[..value.len() - 1];
        }
        if name.eq_ignore_ascii_case(b"content-length") {
            if content_length.is_some() {
                return Err(unavailable());
            }
            let text = std::str::from_utf8(value).map_err(|_| unavailable())?;
            content_length = Some(text.parse::<usize>().map_err(|_| unavailable())?);
        } else if name.eq_ignore_ascii_case(b"transfer-encoding")
            && value.split(|b| *b == b',').any(|token| {
                token
                    .iter()
                    .filter(|b| **b != b' ' && **b != b'\t')
                    .copied()
                    .collect::<Vec<u8>>()
                    .eq_ignore_ascii_case(b"chunked")
            })
        {
            chunked = true;
        }
    }
    if chunked {
        return Ok((status, read_chunked(stream)?));
    }
    if let Some(len) = content_length {
        // Mirror Go's `LimitReader(body, 512<<10+1)`: never allocate a
        // hostile length, but read enough to report the violation.
        let want = len.min(RESPONSE_LIMIT + 1);
        let mut body = vec![0u8; want];
        stream.read_exact(&mut body).map_err(|_| unavailable())?;
        return Ok((status, body));
    }
    let mut body = Vec::new();
    stream
        .take((RESPONSE_LIMIT + 1) as u64)
        .read_to_end(&mut body)
        .map_err(|_| unavailable())?;
    Ok((status, body))
}

/// `decodeResponse` framing: the 512 KiB limit, then one strict JSON value.
fn decode_limited(body: &[u8]) -> Result<Value, String> {
    if body.len() > RESPONSE_LIMIT {
        return Err("identity response exceeds limit".to_string());
    }
    json::decode_tolerant(body).map_err(|e| e.0)
}

const CONNECTION_SPECS: &[Spec] = &[
    Spec {
        name: "provider_id",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "owner_id",
        kind: Kind::Str,
    },
    Spec {
        name: "label",
        kind: Kind::Str,
    },
    Spec {
        name: "email",
        kind: Kind::Str,
    },
    Spec {
        name: "plan",
        kind: Kind::Str,
    },
    Spec {
        name: "generation",
        kind: Kind::I64,
    },
    Spec {
        name: "state",
        kind: Kind::Str,
    },
];

/// Strict `identity.Connection` decode, projected to the minimal
/// `MuseConnection` the daemon directory needs. Unknown fields and
/// malformed `,string`/integer fields fail like Go's `DisallowUnknownFields`
/// binding into the full struct.
fn decode_connections(body: &[u8]) -> Result<Vec<MuseConnection>, String> {
    let v = decode_limited(body)?;
    let items = match &v {
        Value::Array(items) => items,
        _ => {
            let word = match &v {
                Value::Null => "null",
                Value::Bool(_) => "bool",
                Value::Number(_) => "number",
                Value::Str(_) => "string",
                Value::Object(_) => "object",
                Value::Array(_) => "array",
            };
            return Err(format!(
                "json: cannot unmarshal {word} into Go value of type []identity.Connection"
            ));
        }
    };
    let mut out = Vec::with_capacity(items.len());
    for item in items {
        let m =
            json::bind_struct(item, "Connection", &[], CONNECTION_SPECS, false).map_err(|e| e.0)?;
        if m.contains("owner_id") {
            parse_string_i64(&m.take_string("owner_id"))
                .ok_or_else(|| "invalid owner_id".to_string())?;
        }
        out.push(MuseConnection {
            id: m.take_string("id"),
            provider_id: m.take_string("provider_id"),
            state: m.take_string("state"),
        });
    }
    Ok(out)
}

const EXECUTION_BINDING_SPECS: &[Spec] = &[
    Spec {
        name: "child_id",
        kind: Kind::Str,
    },
    Spec {
        name: "uid",
        kind: Kind::Int,
    },
    Spec {
        name: "gid",
        kind: Kind::Int,
    },
    Spec {
        name: "scope",
        kind: Kind::Str,
    },
    Spec {
        name: "credential_root",
        kind: Kind::Str,
    },
    Spec {
        name: "invocation_id",
        kind: Kind::Str,
    },
    Spec {
        name: "kind",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "generation",
        kind: Kind::I64,
    },
];

const EXECUTION_SPECS: &[Spec] = &[
    Spec {
        name: "binding",
        kind: Kind::OptObject {
            go_type: "*identity.Binding",
            struct_name: "Binding",
            specs: EXECUTION_BINDING_SPECS,
        },
    },
    Spec {
        name: "kind",
        kind: Kind::Str,
    },
    Spec {
        name: "execution_id",
        kind: Kind::Str,
    },
    Spec {
        name: "digest",
        kind: Kind::Str,
    },
    Spec {
        name: "state",
        kind: Kind::Str,
    },
    Spec {
        name: "lease_id",
        kind: Kind::Str,
    },
];

fn decode_execution(body: &[u8]) -> Result<Execution, String> {
    let v = decode_limited(body)?;
    let m = json::bind_root(&v, "Execution", EXECUTION_SPECS, false).map_err(|e| e.0)?;
    let binding = m.take_opt_map("binding").map(|b| Binding {
        child_id: b.take_string("child_id"),
        uid: b.take_i64("uid"),
        gid: b.take_i64("gid"),
        scope: b.take_string("scope"),
        credential_root: b.take_string("credential_root"),
        invocation_id: b.take_string("invocation_id"),
        kind: b.take_string("kind"),
        id: b.take_string("id"),
        project: b.take_string("project"),
        login: b.take_string("login"),
        generation: b.take_i64("generation"),
    });
    Ok(Execution {
        binding,
        kind: m.take_string("kind"),
        execution_id: m.take_string("execution_id"),
        digest: m.take_string("digest"),
        state: m.take_string("state"),
        lease_id: m.take_string("lease_id"),
    })
}

impl BrokerClient {
    pub fn new(socket_path: &str) -> Self {
        BrokerClient {
            socket_path: socket_path.to_string(),
        }
    }

    fn call(&self, path: &str, body: &str, deadline: Instant) -> Result<Vec<u8>, String> {
        let timeout = deadline
            .checked_duration_since(Instant::now())
            .filter(|d| !d.is_zero())
            .ok_or_else(unavailable)?;
        let mut stream = UnixStream::connect(&self.socket_path).map_err(|_| unavailable())?;
        stream
            .set_read_timeout(Some(timeout))
            .map_err(|_| unavailable())?;
        stream
            .set_write_timeout(Some(timeout))
            .map_err(|_| unavailable())?;
        let head = format!(
            "POST {path} HTTP/1.1\r\nHost: soda-identity\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
            body.len()
        );
        stream
            .write_all(head.as_bytes())
            .map_err(|_| unavailable())?;
        stream
            .write_all(body.as_bytes())
            .map_err(|_| unavailable())?;
        let (status, response) = read_response(&mut stream)?;
        if status != 200 {
            return Err(map_error(&response));
        }
        Ok(response)
    }

    /// `Client.Acquire`: `POST /acquire`.
    pub fn acquire(&self, req: &AcquireRequest, deadline: Instant) -> Result<Lease, String> {
        let nested = encode_acquire(req);
        let body = encode_request(0, "", "", "", "", Some(("acquire", &nested)), None);
        let raw = self.call("/acquire", &body, deadline)?;
        let v = decode_limited(&raw)?;
        Lease::decode_value(&v)
    }

    /// `Client.Register`: `POST /register`.
    pub fn register(
        &self,
        lease_id: &str,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        let nested = binding.encode();
        let body = encode_request(0, lease_id, "", "", "", Some(("binding", &nested)), None);
        let raw = self.call("/register", &body, deadline)?;
        let v = decode_limited(&raw)?;
        Delivery::decode_value(&v)
    }

    /// `Client.ReconcileLease`: `POST /reconcile-lease`. The body is
    /// unread like Go's nil `out`, beyond the bounded transport read.
    pub fn reconcile_lease(&self, lease_id: &str, deadline: Instant) -> Result<(), String> {
        let body = encode_request(0, lease_id, "", "", "", None, None);
        self.call("/reconcile-lease", &body, deadline)?;
        Ok(())
    }

    /// `Client.EndLease`: `POST /lease/end`.
    pub fn end_lease(&self, actor: i64, lease_id: &str, deadline: Instant) -> Result<(), String> {
        let body = encode_request(actor, lease_id, "", "", "", None, None);
        self.call("/lease/end", &body, deadline)?;
        Ok(())
    }

    /// `Client.Available`: `POST /available`.
    pub fn available(
        &self,
        actor: i64,
        project: &str,
        deadline: Instant,
    ) -> Result<Vec<MuseConnection>, String> {
        let body = encode_request(actor, "", project, "", "", None, None);
        let raw = self.call("/available", &body, deadline)?;
        decode_connections(&raw)
    }

    /// `Client.Return`: `POST /return`. An empty credential is omitted,
    /// like Go's `omitempty` on the byte slice.
    pub fn return_lease(
        &self,
        lease_id: &str,
        binding: &Binding,
        credential: &[u8],
        deadline: Instant,
    ) -> Result<(), String> {
        let nested = binding.encode();
        let encoded = (!credential.is_empty()).then(|| crate::ssh::b64_encode(credential));
        let body = encode_request(
            0,
            lease_id,
            "",
            "",
            "",
            Some(("binding", &nested)),
            encoded.as_deref(),
        );
        self.call("/return", &body, deadline)?;
        Ok(())
    }

    /// `Client.GetExecution`: `POST /execution/get`.
    pub fn get_execution(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<Execution, String> {
        let body = encode_request(0, "", "", kind, execution_id, None, None);
        let raw = self.call("/execution/get", &body, deadline)?;
        decode_execution(&raw)
    }

    /// `Client.CloseExecution`: `POST /execution/close`.
    pub fn close_execution(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let body = encode_request(0, "", "", kind, execution_id, None, None);
        self.call("/execution/close", &body, deadline)?;
        Ok(())
    }
}
