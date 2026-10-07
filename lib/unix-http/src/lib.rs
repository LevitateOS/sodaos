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

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::{Read, Write};
    use std::os::unix::net::UnixListener;
    use std::sync::atomic::{AtomicU64, Ordering};
    use std::thread;
    use std::time::Instant as StdInstant;

    static NEXT_SOCKET: AtomicU64 = AtomicU64::new(0);

    fn socket_path() -> std::path::PathBuf {
        let dir = Path::new(env!("CARGO_MANIFEST_DIR"))
            .ancestors()
            .nth(2)
            .expect("workspace root")
            .join(".artifacts/l08-l09/u");
        std::fs::create_dir_all(&dir).expect("create scoped test artifact directory");
        dir.join(format!(
            "s{}-{}.sock",
            std::process::id(),
            NEXT_SOCKET.fetch_add(1, Ordering::Relaxed)
        ))
    }

    fn serve(reply: &'static [u8]) -> (std::path::PathBuf, thread::JoinHandle<()>) {
        serve_owned(reply.to_vec())
    }

    fn serve_owned(reply: Vec<u8>) -> (std::path::PathBuf, thread::JoinHandle<()>) {
        let path = socket_path();
        let _ = std::fs::remove_file(&path);
        let listener = UnixListener::bind(&path).expect("bind Unix socket");
        let worker = thread::spawn(move || {
            let (mut stream, _) = listener.accept().expect("accept request");
            let mut request = Vec::new();
            let mut byte = [0];
            while !request.ends_with(b"\r\n\r\n") {
                stream.read_exact(&mut byte).expect("read request head");
                request.push(byte[0]);
            }
            let content_length = request
                .split(|byte| *byte == b'\n')
                .filter_map(|line| {
                    let split = line.iter().position(|byte| *byte == b':')?;
                    line[..split]
                        .eq_ignore_ascii_case(b"content-length")
                        .then(|| std::str::from_utf8(&line[split + 1..]).ok())
                        .flatten()?
                        .trim()
                        .parse::<usize>()
                        .ok()
                })
                .next()
                .expect("request content length");
            let mut body = vec![0; content_length];
            stream.read_exact(&mut body).expect("read request body");
            stream.write_all(&reply).expect("write response");
        });
        (path, worker)
    }

    fn call(path: &Path, cap: usize, timeout: Duration) -> Result<Response, Error> {
        request(
            path,
            "POST",
            "/test",
            "soda-test",
            b"{}",
            timeout,
            Limits {
                header_bytes: 8192,
                body_bytes: cap,
            },
        )
    }

    #[test]
    fn accepts_chunked_body_and_trailers() {
        let (path, worker) = serve(
            b"HTTP/1.1 200 OK\r\nContent-Length: 200\r\nTransfer-Encoding: chunked\r\nTrailer: x-done\r\n\r\n2\r\nok\r\n0\r\nx-done: yes\r\n\r\n",
        );
        assert_eq!(call(&path, 4, Duration::from_secs(1)).unwrap().body, b"ok");
        worker.join().unwrap();
        let _ = std::fs::remove_file(path);
    }

    #[test]
    fn accepts_close_delimited_body() {
        let (path, worker) = serve(b"HTTP/1.1 200 OK\r\nConnection: close\r\n\r\nok");
        assert_eq!(call(&path, 4, Duration::from_secs(1)).unwrap().body, b"ok");
        worker.join().unwrap();
        let _ = std::fs::remove_file(path);
    }

    #[test]
    fn rejects_conflicting_framing_and_body_over_cap() {
        for response in [
            &b"HTTP/1.1 200 OK\r\nContent-Length: 2\r\nContent-Length: 3\r\n\r\n{}"[..],
            &b"HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nhello"[..],
        ] {
            let (path, worker) = serve(response);
            let result = call(&path, 4, Duration::from_secs(1));
            if response.starts_with(b"HTTP/1.1 200 OK\r\nContent-Length: 2") {
                assert_eq!(result, Err(Error::Protocol));
            } else {
                assert_eq!(result, Err(Error::BodyTooLarge));
            }
            worker.join().unwrap();
            let _ = std::fs::remove_file(path);
        }
    }

    #[test]
    fn enforces_response_header_byte_cap() {
        let mut response = b"HTTP/1.1 200 OK\r\nX-Fill: ".to_vec();
        response.extend(std::iter::repeat_n(b'x', 9000));
        response.extend_from_slice(b"\r\nContent-Length: 0\r\n\r\n");
        let (path, worker) = serve_owned(response);
        assert_eq!(call(&path, 4, Duration::from_secs(1)), Err(Error::Protocol));
        worker.join().unwrap();
        let _ = std::fs::remove_file(path);
    }

    #[test]
    fn deadline_covers_trickling_response() {
        let path = socket_path();
        let _ = std::fs::remove_file(&path);
        let listener = UnixListener::bind(&path).unwrap();
        let worker = thread::spawn(move || {
            let (mut stream, _) = listener.accept().unwrap();
            let mut request = Vec::new();
            let mut byte = [0];
            while !request.ends_with(b"\r\n\r\n") {
                if stream.read_exact(&mut byte).is_err() {
                    return;
                }
                request.push(byte[0]);
            }
            stream
                .write_all(b"HTTP/1.1 200 OK\r\nContent-Length: 4\r\n\r\n")
                .unwrap();
            thread::sleep(Duration::from_millis(300));
            let _ = stream.write_all(b"done");
        });
        let started = StdInstant::now();
        assert_eq!(
            call(&path, 8, Duration::from_millis(20)),
            Err(Error::Deadline)
        );
        assert!(started.elapsed() < Duration::from_millis(200));
        worker.join().unwrap();
        let _ = std::fs::remove_file(path);
    }
}
