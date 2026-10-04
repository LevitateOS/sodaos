// Unix HTTP service, mirroring internal/identity/control/http.go over the
// admin and runtime listeners: POST-only admission, strict bodies, path
// dispatch and the fixed error codes. Connections close after each
// response; the Go client transparently redials.
use crate::control::Controller;
use crate::strict;
use crate::wire::{DeliveryWire, Error, ErrorKind, Request};
use std::io::{Read, Write};
use std::os::unix::net::{UnixListener, UnixStream};
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Duration;

const MAX_BODY: usize = 512 << 10;
const MAX_HEADER: usize = 8192;
const HEADER_TIMEOUT: Duration = Duration::from_secs(5);
const BODY_TIMEOUT: Duration = Duration::from_secs(30);

const REQUEST_FIELDS: &[&str] = &[
    "provider_id",
    "owner_id",
    "id",
    "label",
    "project_id",
    "kind",
    "execution_id",
    "grant",
    "acquire",
    "binding",
    "credential",
];

const GRANT_FIELDS: &[&str] = &[
    "connection_id",
    "user_id",
    "project_id",
    "confirm_subscription",
    "confirm_credential_exposure",
];

const ACQUIRE_FIELDS: &[&str] = &[
    "repository_id",
    "provider_id",
    "execution_id",
    "actor_id",
    "connection_id",
    "project_id",
    "kind",
    "deadline",
    "role",
];

const BINDING_FIELDS: &[&str] = &[
    "child_id",
    "uid",
    "gid",
    "scope",
    "credential_root",
    "invocation_id",
    "kind",
    "id",
    "project",
    "login",
    "generation",
];

const NESTED: &[(&str, &[&str])] = &[
    ("grant", GRANT_FIELDS),
    ("acquire", ACQUIRE_FIELDS),
    ("binding", BINDING_FIELDS),
];

pub struct Server {
    controller: Arc<Controller>,
    runtime_allowed: bool,
    shutdown: Arc<AtomicBool>,
    inflight: Arc<AtomicUsize>,
}

impl Server {
    pub fn new(
        controller: Arc<Controller>,
        runtime_allowed: bool,
        shutdown: Arc<AtomicBool>,
        inflight: Arc<AtomicUsize>,
    ) -> Server {
        Server {
            controller,
            runtime_allowed,
            shutdown,
            inflight,
        }
    }

    pub fn serve(&self, listener: &UnixListener) {
        listener.set_nonblocking(true).ok();
        let mut fatal_errors = 0u32;
        loop {
            if self.shutdown.load(Ordering::SeqCst) {
                return;
            }
            match listener.accept() {
                Ok((stream, _)) => {
                    fatal_errors = 0;
                    self.inflight.fetch_add(1, Ordering::SeqCst);
                    let controller = Arc::clone(&self.controller);
                    let inflight = Arc::clone(&self.inflight);
                    let runtime_allowed = self.runtime_allowed;
                    std::thread::spawn(move || {
                        handle_connection(stream, &controller, runtime_allowed);
                        inflight.fetch_sub(1, Ordering::SeqCst);
                    });
                }
                Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {
                    fatal_errors = 0;
                    std::thread::sleep(Duration::from_millis(50));
                }
                Err(e)
                    if e.kind() == std::io::ErrorKind::Interrupted
                        || e.kind() == std::io::ErrorKind::ConnectionAborted =>
                {
                    // Transient or per-connection failure: keep serving.
                    fatal_errors = 0;
                    std::thread::sleep(Duration::from_millis(50));
                }
                Err(_) => {
                    if self.shutdown.load(Ordering::SeqCst) {
                        return;
                    }
                    fatal_errors += 1;
                    // Fail fast for supervisor restart, like Go's Serve on
                    // a fatal listener error, instead of spinning deaf.
                    if fatal_errors >= 20 {
                        return;
                    }
                    std::thread::sleep(Duration::from_millis(50));
                }
            }
        }
    }
}

fn handle_connection(mut stream: UnixStream, controller: &Controller, runtime_allowed: bool) {
    let response = match read_request(&mut stream) {
        Ok((request, body)) => dispatch(controller, request, body, runtime_allowed),
        Err(Admission::Invalid) => error_response(400, "invalid request"),
        Err(Admission::HeadersTooLarge) => error_response(431, "header too large"),
    };
    let _ = stream.write_all(&response);
}

enum Admission {
    Invalid,
    HeadersTooLarge,
}

struct HttpRequest {
    method: String,
    path: String,
    query: String,
    origin: String,
}

fn read_request(stream: &mut UnixStream) -> Result<(HttpRequest, Vec<u8>), Admission> {
    stream
        .set_read_timeout(Some(HEADER_TIMEOUT))
        .map_err(|_| Admission::Invalid)?;
    let mut head = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err(Admission::Invalid),
            Ok(_) => {
                head.push(byte[0]);
                if head.len() > MAX_HEADER {
                    return Err(Admission::HeadersTooLarge);
                }
                if head.ends_with(b"\r\n\r\n") {
                    break;
                }
            }
            Err(_) => return Err(Admission::Invalid),
        }
    }
    let mut lines = head.split(|b| *b == b'\n');
    let request_line = lines.next().ok_or(Admission::Invalid)?;
    let mut parts = request_line.split(|b| *b == b' ');
    let method = parts.next().ok_or(Admission::Invalid)?;
    let target = parts.next().ok_or(Admission::Invalid)?;
    if parts.next().is_none() {
        return Err(Admission::Invalid);
    }
    let method = std::str::from_utf8(method)
        .map_err(|_| Admission::Invalid)?
        .to_string();
    let target = std::str::from_utf8(target).map_err(|_| Admission::Invalid)?;
    let (path, query) = match target.find('?') {
        Some(i) => (&target[..i], target[i + 1..].to_string()),
        None => (target, String::new()),
    };
    let path = percent_decode(path).map_err(|_| Admission::Invalid)?;
    let mut origin = String::new();
    let mut content_lengths = Vec::new();
    let mut chunked = false;
    for line in lines {
        let line = line.strip_suffix(b"\r").unwrap_or(line);
        if line.is_empty() {
            continue;
        }
        let Some(colon) = line.iter().position(|b| *b == b':') else {
            return Err(Admission::Invalid);
        };
        let (name, value) = (&line[..colon], &line[colon + 1..]);
        if name.eq_ignore_ascii_case(b"origin") {
            if let Ok(text) = std::str::from_utf8(value.trim_ascii()) {
                if !text.is_empty() {
                    origin = text.to_string();
                }
            }
        } else if name.eq_ignore_ascii_case(b"content-length") {
            content_lengths.push(
                std::str::from_utf8(value.trim_ascii())
                    .map_err(|_| Admission::Invalid)?
                    .to_string(),
            );
        } else if name.eq_ignore_ascii_case(b"transfer-encoding")
            && value.trim_ascii().eq_ignore_ascii_case(b"chunked")
        {
            chunked = true;
        }
    }
    if content_lengths.len() > 1 || chunked {
        return Err(Admission::Invalid);
    }
    let content_length = content_lengths
        .first()
        .map(|v| v.parse::<usize>().map_err(|_| Admission::Invalid))
        .transpose()?
        .unwrap_or(0);
    stream
        .set_read_timeout(Some(BODY_TIMEOUT))
        .map_err(|_| Admission::Invalid)?;
    let mut body = vec![0u8; content_length.min(MAX_BODY + 1)];
    stream
        .read_exact(&mut body)
        .map_err(|_| Admission::Invalid)?;
    if content_length > MAX_BODY {
        return Err(Admission::Invalid);
    }
    Ok((
        HttpRequest {
            method,
            path,
            query,
            origin,
        },
        body,
    ))
}

fn percent_decode(input: &str) -> Result<String, String> {
    let bytes = input.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len() {
                return Err("invalid percent encoding".to_string());
            }
            let hex = std::str::from_utf8(&bytes[i + 1..i + 3])
                .map_err(|_| "invalid percent encoding".to_string())?;
            out.push(
                u8::from_str_radix(hex, 16).map_err(|_| "invalid percent encoding".to_string())?,
            );
            i += 3;
        } else {
            out.push(bytes[i]);
            i += 1;
        }
    }
    String::from_utf8(out).map_err(|_| "invalid percent encoding".to_string())
}

fn dispatch(
    controller: &Controller,
    request: HttpRequest,
    body: Vec<u8>,
    runtime_allowed: bool,
) -> Vec<u8> {
    if request.method != "POST" || !request.query.is_empty() || !request.origin.is_empty() {
        return error_response(403, "denied");
    }
    let input: Request = match strict::decode(&body, MAX_BODY, REQUEST_FIELDS, NESTED) {
        Ok(input) => input,
        Err(_) => return error_response(400, "invalid request"),
    };
    let result = route(controller, &request.path, &input, runtime_allowed);
    match result {
        Ok(output) => success_response(&output),
        Err(err) => {
            let (status, code) = match err.kind() {
                ErrorKind::Denied => (403, "denied"),
                ErrorKind::NotFound => (404, "missing"),
                ErrorKind::Busy => (409, "busy"),
                ErrorKind::Stale => (409, "stale"),
                ErrorKind::Uncertain => (409, "reauth"),
                ErrorKind::Internal => (500, "unavailable"),
            };
            error_response(status, code)
        }
    }
}

// read_request returns the head plus body separately to keep header limits
// distinct from body limits.
fn route(
    controller: &Controller,
    path: &str,
    input: &Request,
    runtime_allowed: bool,
) -> Result<Option<Vec<u8>>, Error> {
    match path {
        "/connections" => Ok(Some(serde_json::to_vec(
            &controller.connections(input.owner_id)?,
        )?)),
        "/available" => Ok(Some(serde_json::to_vec(
            &controller.available(input.owner_id, &input.project_id)?,
        )?)),
        "/revoke" => {
            controller.revoke(input.owner_id, &input.id)?;
            Ok(None)
        }
        "/enrollment/start" => {
            let enrollment =
                controller.start_enrollment(input.owner_id, &input.provider_id, &input.label)?;
            Ok(Some(serde_json::to_vec(&enrollment)?))
        }
        "/enrollment/read" => Ok(Some(serde_json::to_vec(
            &controller.enrollment(input.owner_id, &input.id)?,
        )?)),
        "/enrollment/cancel" => {
            controller.cancel_enrollment(input.owner_id, &input.id)?;
            Ok(None)
        }
        "/grants" => Ok(Some(serde_json::to_vec(
            &controller.grants(input.owner_id, &input.id)?,
        )?)),
        "/grant/create" => {
            let Some(grant) = &input.grant else {
                return Err(Error::denied("identity authority denied"));
            };
            Ok(Some(serde_json::to_vec(
                &controller.create_grant(input.owner_id, grant)?,
            )?))
        }
        "/grant/revoke" => {
            controller.revoke_grant(input.owner_id, &input.id)?;
            Ok(None)
        }
        "/leases" => Ok(Some(serde_json::to_vec(
            &controller.leases(input.owner_id, &input.id)?,
        )?)),
        "/lease/end" => {
            controller.end_lease(input.owner_id, &input.id)?;
            Ok(None)
        }
        _ => {
            if !runtime_allowed {
                return Err(Error::denied("identity authority denied"));
            }
            route_runtime(controller, path, input)
        }
    }
}

fn route_runtime(
    controller: &Controller,
    path: &str,
    input: &Request,
) -> Result<Option<Vec<u8>>, Error> {
    match path {
        "/acquire" => {
            let Some(acquire) = &input.acquire else {
                return Err(Error::denied("identity authority denied"));
            };
            Ok(Some(serde_json::to_vec(&controller.acquire(acquire)?)?))
        }
        "/register" => {
            let Some(binding) = &input.binding else {
                return Err(Error::denied("identity authority denied"));
            };
            let (lease, credential) = controller.register(&input.id, binding)?;
            Ok(Some(serde_json::to_vec(&DeliveryWire {
                lease,
                credential,
            })?))
        }
        "/reject" => {
            let Some(binding) = &input.binding else {
                return Err(Error::denied("identity authority denied"));
            };
            controller.reject(&input.id, binding)?;
            Ok(None)
        }
        "/return" => {
            let Some(binding) = &input.binding else {
                return Err(Error::denied("identity authority denied"));
            };
            controller.return_lease(
                &input.id,
                binding,
                input.credential.as_deref().unwrap_or(b""),
            )?;
            Ok(None)
        }
        "/reconcile-lease" => {
            controller.reconcile_lease(&input.id)?;
            Ok(None)
        }
        "/execution/get" => Ok(Some(serde_json::to_vec(
            &controller.get_execution(&input.kind, &input.execution_id)?,
        )?)),
        "/execution/close" => {
            controller.close_execution(&input.kind, &input.execution_id)?;
            Ok(None)
        }
        _ => Err(Error::denied("identity authority denied")),
    }
}

fn success_response(output: &Option<Vec<u8>>) -> Vec<u8> {
    let mut body = match output {
        Some(bytes) => bytes.clone(),
        None => b"{}".to_vec(),
    };
    body.push(b'\n');
    let mut response = format!(
        "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nCache-Control: no-store\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        body.len()
    )
    .into_bytes();
    response.extend_from_slice(&body);
    response
}

fn error_response(status: u16, code: &str) -> Vec<u8> {
    let reason = match status {
        400 => "Bad Request",
        403 => "Forbidden",
        404 => "Not Found",
        409 => "Conflict",
        431 => "Request Header Fields Too Large",
        _ => "Internal Server Error",
    };
    let body = format!("{code}\n");
    let mut response = format!(
        "HTTP/1.1 {status} {reason}\r\nContent-Type: text/plain; charset=utf-8\r\nX-Content-Type-Options: nosniff\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        body.len()
    )
    .into_bytes();
    response.extend_from_slice(body.as_bytes());
    response
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn percent_paths_decode() {
        assert_eq!(percent_decode("/acquire").unwrap(), "/acquire");
        assert_eq!(percent_decode("/%61cquire").unwrap(), "/acquire");
        assert!(percent_decode("/%zz").is_err());
        assert!(percent_decode("/%4").is_err());
    }

    #[test]
    fn error_bodies_match_go() {
        let response = error_response(403, "denied");
        let text = String::from_utf8(response).unwrap();
        assert!(text.starts_with("HTTP/1.1 403 Forbidden\r\n"));
        assert!(text.ends_with("\r\n\r\ndenied\n"));
        assert!(text.contains("Content-Type: text/plain; charset=utf-8"));
    }

    #[test]
    fn success_envelope_matches_go() {
        let response = success_response(&None);
        let text = String::from_utf8(response).unwrap();
        assert!(text.contains("Content-Type: application/json"));
        assert!(text.contains("Cache-Control: no-store"));
        assert!(text.ends_with("\r\n\r\n{}\n"));
    }
}
