//! Bounded synchronous HTTP/1 client calls over Unix sockets.
//!
//! This crate owns wire framing and the HTTP driver lifetime. Callers retain
//! their socket authority, request policy, response status and domain errors.

use std::path::Path;
use std::time::Duration;

use bytes::Bytes;
use http_body_util::{BodyExt, Full};
use hyper::body::Incoming;
use hyper::client::conn::http1;
use hyper::{Request, Response as HttpResponse};
use hyper_util::rt::TokioIo;
use tokio::net::UnixStream;
use tokio::time::{timeout_at, Instant};

/// Per-call admission limits. `header_bytes` is enforced by Hyper's HTTP/1
/// parser; `body_bytes` is checked before extending the returned allocation.
#[derive(Clone, Copy, Debug)]
pub struct Limits {
    pub header_bytes: usize,
    pub body_bytes: usize,
}

#[derive(Debug, PartialEq, Eq)]
pub struct Response {
    pub status: u16,
    pub body: Vec<u8>,
}

#[derive(Debug, PartialEq, Eq)]
pub enum Error {
    Connect,
    InvalidRequest,
    Protocol,
    Deadline,
    BodyTooLarge,
}

/// Perform one non-retried HTTP/1 request. The timeout covers Unix connect,
/// request writing, response headers and the complete bounded response body.
pub fn request(
    socket: &Path,
    method: &str,
    target: &str,
    host: &str,
    body: &[u8],
    timeout: Duration,
    limits: Limits,
) -> Result<Response, Error> {
    let deadline = Instant::now() + timeout;
    let work_deadline = deadline - Duration::from_millis(10).min(timeout / 4);
    let runtime = tokio::runtime::Builder::new_current_thread()
        .enable_io()
        .enable_time()
        .build()
        .map_err(|_| Error::Connect)?;
    runtime.block_on(request_inner(
        socket,
        method,
        target,
        host,
        body,
        work_deadline,
        deadline,
        limits,
    ))
}

async fn request_inner(
    socket: &Path,
    method: &str,
    target: &str,
    host: &str,
    body: &[u8],
    work_deadline: Instant,
    deadline: Instant,
    limits: Limits,
) -> Result<Response, Error> {
    let stream = timeout_at(work_deadline, UnixStream::connect(socket))
        .await
        .map_err(|_| Error::Deadline)?
        .map_err(|_| Error::Connect)?;

    let mut builder = http1::Builder::new();
    builder.max_headers(100);
    builder.max_header_size(limits.header_bytes);
    builder.max_buf_size(limits.header_bytes.max(8192));
    let (mut sender, connection) =
        timeout_at(work_deadline, builder.handshake(TokioIo::new(stream)))
            .await
            .map_err(|_| Error::Deadline)?
            .map_err(|error| map_protocol_error(&error))?;
    let mut driver = tokio::spawn(connection);

    let operation = async {
        let request = Request::builder()
            .method(method)
            .uri(target)
            .header(hyper::header::HOST, host)
            .header(hyper::header::CONTENT_TYPE, "application/json")
            .header(hyper::header::CONTENT_LENGTH, body.len().to_string())
            .header(hyper::header::CONNECTION, "close")
            .body(Full::new(Bytes::copy_from_slice(body)))
            .map_err(|_| Error::InvalidRequest)?;
        let response = sender
            .send_request(request)
            .await
            .map_err(|error| map_protocol_error(&error))?;
        collect_response(response, limits.body_bytes, work_deadline).await
    };
    let outcome = match timeout_at(work_deadline, operation).await {
        Ok(result) => result,
        Err(_) => Err(Error::Deadline),
    };

    // A connection driver belongs to this call, including on a body-cap or
    // protocol error. Abort first so no pending read can outlive the facade.
    driver.abort();
    let joined = match timeout_at(deadline, &mut driver).await {
        Ok(joined) => joined,
        Err(_) => return Err(Error::Deadline),
    };
    if outcome.is_ok() {
        match joined {
            Ok(Ok(())) => {}
            Ok(Err(error)) => return Err(map_protocol_error(&error)),
            Err(error) if error.is_cancelled() => {}
            Err(_) => return Err(Error::Connect),
        }
    }
    outcome
}

async fn collect_response(
    response: HttpResponse<Incoming>,
    cap: usize,
    deadline: Instant,
) -> Result<Response, Error> {
    let status = response.status().as_u16();
    let mut body = Vec::new();
    let mut incoming = response.into_body();
    loop {
        let frame = timeout_at(deadline, incoming.frame())
            .await
            .map_err(|_| Error::Deadline)?;
        let Some(frame) = frame else { break };
        let frame = frame.map_err(|error| map_protocol_error(&error))?;
        if let Ok(data) = frame.into_data() {
            if data.len() > cap.saturating_add(1).saturating_sub(body.len()) {
                return Err(Error::BodyTooLarge);
            }
            body.extend_from_slice(&data);
            if body.len() > cap {
                return Err(Error::BodyTooLarge);
            }
        }
    }
    Ok(Response { status, body })
}

fn map_protocol_error(error: &hyper::Error) -> Error {
    if error.is_parse() {
        Error::Protocol
    } else {
        Error::Connect
    }
}
