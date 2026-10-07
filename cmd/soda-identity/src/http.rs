// Unix HTTP listener, implemented with Hyper's HTTP/1 server.
use crate::control::Controller;
use crate::http_routes::dispatch;
use crate::http_wire::{error_response, HttpRequest, MAX_BODY, MAX_HEADER};
use bytes::Bytes;
use http_body_util::{BodyExt, Full};
use hyper::body::Incoming;
use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{Request, Response};
use hyper_util::rt::{TokioIo, TokioTimer};
use std::collections::HashMap;
use std::io;
use std::os::unix::net::UnixListener;
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Duration;
use tokio::net::UnixListener as TokioUnixListener;
use tokio::sync::Semaphore;
use tokio::task::{AbortHandle, Id, JoinSet};

const MAX_CONNECTIONS: usize = 64;
const MAX_BACKENDS: usize = 8;
const BODY_TIMEOUT: Duration = Duration::from_secs(30);

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
        if listener.set_nonblocking(true).is_err() {
            return;
        }
        let runtime = match tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
        {
            Ok(runtime) => runtime,
            Err(_) => return,
        };
        let listener = match listener.try_clone() {
            Ok(listener) => listener,
            Err(_) => return,
        };
        let controller = Arc::clone(&self.controller);
        let shutdown = Arc::clone(&self.shutdown);
        let inflight = Arc::clone(&self.inflight);
        let runtime_allowed = self.runtime_allowed;
        runtime.block_on(async move {
            let listener = match TokioUnixListener::from_std(listener) {
                Ok(listener) => listener,
                Err(_) => return,
            };
            serve_runtime(listener, controller, runtime_allowed, shutdown, inflight).await;
        });
    }
}

async fn serve_runtime(
    listener: TokioUnixListener,
    controller: Arc<Controller>,
    runtime_allowed: bool,
    shutdown: Arc<AtomicBool>,
    inflight: Arc<AtomicUsize>,
) {
    let connections = Arc::new(Semaphore::new(MAX_CONNECTIONS));
    let backends = Arc::new(Semaphore::new(MAX_BACKENDS));
    let mut drivers = JoinSet::new();
    let mut running: HashMap<Id, (AbortHandle, Arc<AtomicBool>)> = HashMap::new();
    let mut fatal_errors = 0u32;
    loop {
        if shutdown.load(Ordering::SeqCst) {
            break;
        }
        tokio::select! {
            _ = tokio::time::sleep(Duration::from_millis(50)) => {},
            accepted = listener.accept() => match accepted {
                Ok((stream, _)) => {
                    fatal_errors = 0;
                    // Backpressure the accept loop; at most one extra accepted stream is held.
                    let permit = tokio::select! {
                        permit = Arc::clone(&connections).acquire_owned() => match permit {
                            Ok(permit) => permit,
                            Err(_) => break,
                        },
                        _ = wait_shutdown(&shutdown) => break,
                    };
                    let backend_active = Arc::new(AtomicBool::new(false));
                    let task_active = Arc::clone(&backend_active);
                    let controller = Arc::clone(&controller);
                    let backends = Arc::clone(&backends);
                    let shutdown = Arc::clone(&shutdown);
                    let inflight = Arc::clone(&inflight);
                    inflight.fetch_add(1, Ordering::SeqCst);
                    let abort = drivers.spawn(async move {
                        let _guard = InflightGuard(inflight);
                        let _connection_permit = permit;
                        serve_connection(stream, controller, runtime_allowed, backends, shutdown, task_active).await;
                    });
                    running.insert(abort.id(), (abort, backend_active));
                }
                Err(e) if e.kind() == io::ErrorKind::WouldBlock || e.kind() == io::ErrorKind::Interrupted || e.kind() == io::ErrorKind::ConnectionAborted => {},
                Err(_) => {
                    fatal_errors += 1;
                    if fatal_errors >= 20 { break; }
                    tokio::time::sleep(Duration::from_millis(50)).await;
                }
            }
        }
        while let Some(result) = drivers.try_join_next_with_id() {
            if let Ok((id, _)) = result {
                running.remove(&id);
            } else if let Err(error) = result {
                running.remove(&error.id());
            }
        }
    }

    // Prevent any new backend admission. Active synchronous callbacks are
    // intentionally allowed to finish and are joined before serve returns.
    backends.close();
    for (abort, active) in running.values() {
        if !active.load(Ordering::SeqCst) {
            abort.abort();
        }
    }
    while !drivers.is_empty() {
        tokio::select! {
            result = drivers.join_next_with_id() => if let Some(result) = result {
                match result {
                    Ok((id, _)) => { running.remove(&id); }
                    Err(error) => { running.remove(&error.id()); }
                }
            },
            _ = tokio::time::sleep(Duration::from_millis(25)) => {
                // A backend may finish after the initial scan and leave the
                // request waiting on a socket write. Cancel that now-idle
                // driver while retaining any still-running backend task.
                for (abort, active) in running.values() {
                    if !active.load(Ordering::SeqCst) { abort.abort(); }
                }
            }
        }
    }
}

struct InflightGuard(Arc<AtomicUsize>);
impl Drop for InflightGuard {
    fn drop(&mut self) {
        self.0.fetch_sub(1, Ordering::SeqCst);
    }
}

async fn serve_connection(
    stream: tokio::net::UnixStream,
    controller: Arc<Controller>,
    runtime_allowed: bool,
    backends: Arc<Semaphore>,
    shutdown: Arc<AtomicBool>,
    backend_active: Arc<AtomicBool>,
) {
    let service = service_fn(move |request: Request<Incoming>| {
        let controller = Arc::clone(&controller);
        let backends = Arc::clone(&backends);
        let shutdown = Arc::clone(&shutdown);
        let active = Arc::clone(&backend_active);
        async move {
            let response = respond(
                request,
                controller,
                runtime_allowed,
                backends,
                shutdown,
                active,
            )
            .await;
            Ok::<_, std::convert::Infallible>(response)
        }
    });
    let mut builder = http1::Builder::new();
    builder
        .timer(TokioTimer::new())
        .header_read_timeout(crate::http_routes::HEADER_TIMEOUT)
        .max_header_size(MAX_HEADER)
        .max_buf_size(MAX_HEADER)
        .keep_alive(false);
    let _ = builder
        .serve_connection(TokioIo::new(stream), service)
        .await;
}

async fn respond(
    request: Request<Incoming>,
    controller: Arc<Controller>,
    runtime_allowed: bool,
    backends: Arc<Semaphore>,
    shutdown: Arc<AtomicBool>,
    active: Arc<AtomicBool>,
) -> Response<Full<Bytes>> {
    if shutdown.load(Ordering::SeqCst) {
        return error_response(503, "unavailable");
    }
    let (parts, body) = request.into_parts();
    let content_lengths = parts.headers.get_all(hyper::header::CONTENT_LENGTH);
    if content_lengths.iter().count() > 1
        || parts
            .headers
            .get(hyper::header::TRANSFER_ENCODING)
            .is_some_and(|value| {
                value
                    .as_bytes()
                    .trim_ascii()
                    .eq_ignore_ascii_case(b"chunked")
            })
    {
        return error_response(400, "invalid request");
    }
    let origin = parts
        .headers
        .get_all(hyper::header::ORIGIN)
        .iter()
        .filter_map(|value| std::str::from_utf8(value.as_bytes()).ok())
        .map(str::trim_ascii)
        .find(|value| !value.is_empty())
        .unwrap_or("")
        .to_owned();
    let path = match crate::http_wire::percent_decode(parts.uri.path()) {
        Ok(path) => path,
        Err(_) => return error_response(400, "invalid request"),
    };
    let request = HttpRequest {
        method: parts.method.as_str().to_owned(),
        path,
        query: parts.uri.query().unwrap_or("").to_owned(),
        origin,
    };

    let mut body = body;
    let mut bytes = Vec::new();
    let body_deadline = tokio::time::Instant::now() + BODY_TIMEOUT;
    loop {
        let frame = match tokio::time::timeout_at(body_deadline, body.frame()).await {
            Ok(frame) => frame,
            Err(_) => return error_response(400, "invalid request"),
        };
        match frame {
            None => break,
            Some(Ok(frame)) => {
                if let Ok(data) = frame.into_data() {
                    if bytes.len().saturating_add(data.len()) > MAX_BODY {
                        return error_response(400, "invalid request");
                    }
                    bytes.extend_from_slice(&data);
                }
            }
            Some(Err(_)) => return error_response(400, "invalid request"),
        }
    }

    if shutdown.load(Ordering::SeqCst) {
        return error_response(503, "unavailable");
    }
    let permit = match Arc::clone(&backends).try_acquire_owned() {
        Ok(permit) => permit,
        Err(_) => return error_response(503, "unavailable"),
    };
    // Publish the admitted blocking call before checking shutdown so shutdown
    // either aborts this request or observes and joins the backend operation.
    active.store(true, Ordering::SeqCst);
    if shutdown.load(Ordering::SeqCst) {
        active.store(false, Ordering::SeqCst);
        drop(permit);
        return error_response(503, "unavailable");
    }
    let join = tokio::task::spawn_blocking(move || {
        let _permit = permit;
        dispatch(&controller, request, bytes, runtime_allowed)
    });
    let response = join
        .await
        .unwrap_or_else(|_| error_response(500, "unavailable"));
    active.store(false, Ordering::SeqCst);
    response
}

async fn wait_shutdown(shutdown: &AtomicBool) {
    while !shutdown.load(Ordering::SeqCst) {
        tokio::time::sleep(Duration::from_millis(25)).await;
    }
}
