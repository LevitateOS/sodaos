// Unix-socket HTTP server for the Rust daemon mux (PR26).
//
// Hyper owns HTTP/1 parsing, framing and response serialization. This owner
// keeps bounded connection/backend admission, route policy and cancellation.
//
// Served from the root systemd socket like cmd/soda-host/main.go: the
// binary main obtains the listener from fd 3 (see `systemd_listener`),
// mounts one `Server`, and stops it through the shutdown flag. Terminal
// upgrades retain a joined pump task for the full admitted session.
use crate::gmux_admission::{AdmissionGate, TerminalGate};
use crate::gmux_backend::ExecBackend;
use crate::gmux_routes::{dispatch, error_response, DaemonConfig, HttpResponse, RouteOutcome};
use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::Request;
use hyper_util::rt::{TokioIo, TokioTimer};
use std::io;
use std::os::unix::fs::PermissionsExt;
use std::os::unix::net::UnixListener;
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::sync::{Arc, Mutex};
use std::time::Duration;

#[path = "daemon/http.rs"]
mod http;

use self::http::{read_body, request_head, HEADER_TIMEOUT, MAX_HEADER};

/// The mux server. Generic over the executor so tests mount the stub and
/// the integrator mounts the real adapter without touching this file.
pub struct Server<B> {
    backend: Arc<B>,
    config: DaemonConfig,
    gate: Arc<AdmissionGate>,
    terminal_gate: Arc<TerminalGate>,
    shutdown: Arc<AtomicBool>,
    inflight: Arc<AtomicUsize>,
}

impl<B: ExecBackend + 'static> Server<B> {
    pub fn new(backend: Arc<B>, config: DaemonConfig) -> Server<B> {
        Server {
            backend,
            config,
            gate: Arc::new(AdmissionGate::new()),
            terminal_gate: Arc::new(TerminalGate::new()),
            shutdown: Arc::new(AtomicBool::new(false)),
            inflight: Arc::new(AtomicUsize::new(0)),
        }
    }

    /// Signal the accept loop to stop (binary main calls this on SIGTERM).
    pub fn shutdown(&self) {
        self.shutdown.store(true, Ordering::SeqCst);
    }

    /// Whether a request or operator has asked the host to retire this server.
    pub fn is_shutdown(&self) -> bool {
        self.shutdown.load(Ordering::Acquire)
    }

    /// Connections currently being handled (binary main drains on this).
    pub fn inflight(&self) -> usize {
        self.inflight.load(Ordering::SeqCst)
    }

    /// Serve until `shutdown`. Listener errors fail fast after a short
    /// burst so the supervisor restarts a deaf daemon (http.rs precedent,
    /// mirroring Go's Serve returning on fatal listener errors).
    pub fn serve(&self, listener: &UnixListener) -> Result<(), String> {
        listener
            .set_nonblocking(true)
            .map_err(|_| "server listener setup failed".to_string())?;
        let runtime = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
            .map_err(|_| "server runtime setup failed".to_string())?;
        let listener = listener
            .try_clone()
            .map_err(|_| "server listener clone failed".to_string())?;
        let shutdown = Arc::clone(&self.shutdown);
        let backend = Arc::clone(&self.backend);
        let config = self.config.clone();
        let gate = Arc::clone(&self.gate);
        let terminal_gate = Arc::clone(&self.terminal_gate);
        let inflight = Arc::clone(&self.inflight);
        let task_failure = Arc::new(Mutex::new(None::<String>));
        let failures = Arc::clone(&task_failure);
        let backend_permits = Arc::new(tokio::sync::Semaphore::new(16));
        let backend_jobs = Arc::new(std::sync::Mutex::new(
            Vec::<tokio::task::JoinHandle<()>>::new(),
        ));
        let result = runtime.block_on(async move {
            let listener = tokio::net::UnixListener::from_std(listener)
                .map_err(|_| "server listener conversion failed".to_string())?;
            let permits = Arc::new(tokio::sync::Semaphore::new(128));
            let upgrades = Arc::new(std::sync::Mutex::new(Vec::<tokio::task::JoinHandle<()>>::new()));
            let mut connections = tokio::task::JoinSet::new();
            let mut fatal_errors = 0u32;
            loop {
                if shutdown.load(Ordering::SeqCst) { break; }
                let finished = {
                    let mut jobs = backend_jobs.lock().unwrap();
                    let all = std::mem::take(&mut *jobs);
                    let (finished, pending): (Vec<_>, Vec<_>) = all.into_iter().partition(|job| job.is_finished());
                    *jobs = pending;
                    finished
                };
                for job in finished {
                    if job.await.is_err() {
                        record_task_failure(&failures, "backend worker panicked");
                    }
                }
                let finished_upgrades = {
                    let mut tasks = upgrades.lock().unwrap();
                    let all = std::mem::take(&mut *tasks);
                    let (finished, pending): (Vec<_>, Vec<_>) = all.into_iter().partition(|task| task.is_finished());
                    *tasks = pending;
                    finished
                };
                for task in finished_upgrades {
                    if task.await.is_err() {
                        record_task_failure(&failures, "terminal upgrade worker panicked");
                    }
                }
                tokio::select! {
                    accepted = listener.accept() => match accepted {
                        Ok((stream, _)) => {
                            fatal_errors = 0;
                            let permit = match Arc::clone(&permits).try_acquire_owned() {
                                Ok(permit) => permit,
                                Err(_) => { drop(stream); continue; }
                            };
                            inflight.fetch_add(1, Ordering::SeqCst);
                            connections.spawn(handle_connection(
                                stream, Arc::clone(&backend), config.clone(), Arc::clone(&gate),
                                Arc::clone(&terminal_gate), Arc::clone(&shutdown), Arc::clone(&upgrades),
                                Arc::clone(&backend_permits), Arc::clone(&backend_jobs), permit,
                                Arc::clone(&inflight), Arc::clone(&failures),
                            ));
                        }
                        Err(e) if e.kind() == std::io::ErrorKind::Interrupted || e.kind() == std::io::ErrorKind::ConnectionAborted => {}
                        Err(_) => {
                            fatal_errors += 1;
                            if fatal_errors >= 20 {
                                record_task_failure(
                                    &failures,
                                    "HTTP listener failed after repeated accept errors",
                                );
                                break;
                            }
                        }
                    },
                    joined = connections.join_next(), if !connections.is_empty() => {
                        if matches!(joined, Some(Err(_))) {
                            record_task_failure(&failures, "HTTP connection worker panicked");
                        }
                    }
                    _ = tokio::time::sleep(Duration::from_millis(50)) => {}
                }
            }
            shutdown.store(true, Ordering::SeqCst);
            while let Some(joined) = connections.join_next().await {
                if joined.is_err() {
                    record_task_failure(&failures, "HTTP connection worker panicked");
                }
            }
            let tasks = upgrades.lock().map(|mut tasks| std::mem::take(&mut *tasks)).unwrap_or_default();
            for task in tasks {
                if task.await.is_err() {
                    record_task_failure(&failures, "terminal upgrade worker panicked");
                }
            }
            let jobs = backend_jobs.lock().map(|mut jobs| std::mem::take(&mut *jobs)).unwrap_or_default();
            for job in jobs {
                if job.await.is_err() {
                    record_task_failure(&failures, "backend worker panicked");
                }
            }
            Ok::<(), String>(())
        });
        result?;
        let failure_result = match task_failure.lock() {
            Ok(failure) => match failure.clone() {
                Some(error) => Err(error),
                None => Ok(()),
            },
            Err(_) => Err("server failure state unavailable".to_string()),
        };
        failure_result
    }
}

async fn handle_connection<B: ExecBackend + 'static>(
    stream: tokio::net::UnixStream,
    backend: Arc<B>,
    config: DaemonConfig,
    gate: Arc<AdmissionGate>,
    terminal_gate: Arc<TerminalGate>,
    shutdown: Arc<AtomicBool>,
    upgrades: Arc<std::sync::Mutex<Vec<tokio::task::JoinHandle<()>>>>,
    backend_permits: Arc<tokio::sync::Semaphore>,
    backend_jobs: Arc<std::sync::Mutex<Vec<tokio::task::JoinHandle<()>>>>,
    _permit: tokio::sync::OwnedSemaphorePermit,
    inflight: Arc<AtomicUsize>,
    task_failure: Arc<Mutex<Option<String>>>,
) {
    struct Inflight(Arc<AtomicUsize>);
    impl Drop for Inflight {
        fn drop(&mut self) {
            self.0.fetch_sub(1, Ordering::SeqCst);
        }
    }
    let _inflight = Inflight(inflight);
    let connection_shutdown = Arc::clone(&shutdown);
    let connection_count = Arc::clone(&_inflight.0);
    let service = service_fn(move |mut request: Request<hyper::body::Incoming>| {
        let backend = Arc::clone(&backend);
        let config = config.clone();
        let gate = Arc::clone(&gate);
        let terminal_gate = Arc::clone(&terminal_gate);
        let shutdown = Arc::clone(&connection_shutdown);
        let upgrades = Arc::clone(&upgrades);
        let backend_permits = Arc::clone(&backend_permits);
        let backend_jobs = Arc::clone(&backend_jobs);
        let task_failure = Arc::clone(&task_failure);
        let connection_count = Arc::clone(&connection_count);
        async move {
            let on_upgrade = hyper::upgrade::on(&mut request);
            let head = match request_head(&request) {
                Ok(head) => head,
                Err(_) => {
                    return Ok::<HttpResponse, std::convert::Infallible>(error_response(
                        400,
                        "invalid request",
                    ))
                }
            };
            let body_result = tokio::select! {
                result = read_body(request.into_body(), &head.path) => result,
                _ = wait_shutdown(Arc::clone(&shutdown)) => return Ok(error_response(503, "service shutting down")),
            };
            let (body, complete) = match body_result {
                Ok(body) => body,
                Err(_) => (Vec::new(), false),
            };
            if shutdown.load(Ordering::Acquire) {
                return Ok(error_response(503, "service shutting down"));
            }
            let permit = match backend_permits.try_acquire_owned() {
                Ok(permit) => permit,
                Err(_) => return Ok(error_response(503, "service busy")),
            };
            let route_backend = Arc::clone(&backend);
            let route_shutdown = Arc::clone(&shutdown);
            let (send_outcome, receive_outcome) = tokio::sync::oneshot::channel();
            let job = tokio::task::spawn_blocking(move || {
                let _permit = permit;
                if route_shutdown.load(Ordering::Acquire) {
                    let _ = send_outcome.send(RouteOutcome::Respond(error_response(
                        503,
                        "service shutting down",
                    )));
                    return;
                }
                let outcome = dispatch(
                    route_backend.as_ref(),
                    &config,
                    &gate,
                    &terminal_gate,
                    &head,
                    &body,
                    complete,
                );
                let _ = send_outcome.send(outcome);
            });
            if let Ok(mut jobs) = backend_jobs.lock() {
                jobs.push(job);
            }
            let outcome = tokio::select! {
                result = receive_outcome => match result {
                    Ok(outcome) => outcome,
                    Err(_) => return Ok(error_response(500, "native operation failed; inspect operator journal")),
                },
                _ = wait_shutdown(Arc::clone(&shutdown)) => return Ok(error_response(503, "service shutting down")),
            };
            match outcome {
                RouteOutcome::Respond(response) => Ok(response),
                RouteOutcome::TerminalUpgrade {
                    response,
                    session,
                    slot,
                } => {
                    let backend = Arc::clone(&backend);
                    connection_count.fetch_add(1, Ordering::SeqCst);
                    let task = tokio::spawn(async move {
                        struct PumpInflight(Arc<AtomicUsize>);
                        impl Drop for PumpInflight {
                            fn drop(&mut self) {
                                self.0.fetch_sub(1, Ordering::SeqCst);
                            }
                        }
                        let _pump_inflight = PumpInflight(connection_count);
                        let _slot = slot;
                        let task_failure = Arc::clone(&task_failure);
                        let upgraded = tokio::select! {
                            upgraded = on_upgrade => upgraded,
                            _ = wait_shutdown(Arc::clone(&shutdown)) => return,
                        };
                        if let Ok(upgraded) = upgraded {
                            if let Ok(io) = upgraded.downcast::<TokioIo<tokio::net::UnixStream>>() {
                                let read_buf = io.read_buf.to_vec();
                                if let Ok(stream) = io.io.into_inner().into_std() {
                                    let _ = stream.set_nonblocking(false);
                                    let mut websocket_config =
                                        tungstenite::protocol::WebSocketConfig::default();
                                    websocket_config.read_buffer_size = 8192;
                                    websocket_config.write_buffer_size = 8192;
                                    websocket_config.max_write_buffer_size = 262_144;
                                    websocket_config.max_message_size =
                                        Some(crate::gmux_admission::TERMINAL_FRAME_LIMIT);
                                    websocket_config.max_frame_size =
                                        Some(crate::gmux_admission::TERMINAL_FRAME_LIMIT);
                                    let ws = tungstenite::protocol::WebSocket::from_partially_read(
                                        stream,
                                        read_buf,
                                        tungstenite::protocol::Role::Server,
                                        Some(websocket_config),
                                    );
                                    let job_shutdown = Arc::clone(&shutdown);
                                    match tokio::task::spawn_blocking(move || {
                                        backend.pump_terminal(ws, session, job_shutdown)
                                    })
                                    .await
                                    {
                                        Ok(Ok(())) => {}
                                        Ok(Err(error)) => {
                                            record_task_failure(
                                                &task_failure,
                                                &format!("terminal pump failed: {error:?}"),
                                            );
                                            shutdown.store(true, Ordering::SeqCst);
                                        }
                                        Err(_) => {
                                            record_task_failure(
                                                &task_failure,
                                                "terminal pump worker panicked",
                                            );
                                            shutdown.store(true, Ordering::SeqCst);
                                        }
                                    }
                                }
                            }
                        }
                    });
                    if let Ok(mut tasks) = upgrades.lock() {
                        tasks.push(task);
                    }
                    Ok(response)
                }
            }
        }
    });
    let mut builder = http1::Builder::new();
    builder
        .keep_alive(false)
        .max_buf_size(MAX_HEADER)
        .max_header_size(MAX_HEADER)
        .header_read_timeout(HEADER_TIMEOUT)
        .timer(TokioTimer::new());
    tokio::select! {
        _ = builder.serve_connection(TokioIo::new(stream), service).with_upgrades() => {},
        _ = wait_shutdown(Arc::clone(&shutdown)) => {},
    }
}

fn record_task_failure(failure: &Mutex<Option<String>>, message: &str) {
    if let Ok(mut failure) = failure.lock() {
        if failure.is_none() {
            *failure = Some(message.to_string());
        }
    }
}

async fn wait_shutdown(shutdown: Arc<AtomicBool>) {
    while !shutdown.load(Ordering::Acquire) {
        tokio::time::sleep(Duration::from_millis(50)).await;
    }
}

// -- socket activation (cmd/soda-host/main.go serveHostSocket) --

/// Take the systemd-activated listener from fd 3.
///
/// Mirrors `serveHostSocket`'s gate: root only, exactly one passed fd owned
/// by this process. MUST be called exactly once: ownership of fd 3 moves
/// into the returned listener.
pub fn systemd_listener() -> io::Result<UnixListener> {
    use std::os::unix::io::FromRawFd;
    if unsafe { libc::geteuid() } != 0 {
        return Err(io::Error::new(
            io::ErrorKind::PermissionDenied,
            "requires root and the soda-host systemd Unix socket",
        ));
    }
    let pid = std::process::id().to_string();
    if std::env::var("LISTEN_PID").as_deref() != Ok(pid.as_str())
        || std::env::var("LISTEN_FDS").as_deref() != Ok("1")
    {
        return Err(io::Error::new(
            io::ErrorKind::NotFound,
            "requires root and the soda-host systemd Unix socket",
        ));
    }
    Ok(unsafe { UnixListener::from_raw_fd(3) })
}

/// Bind a Unix listener at an explicit path (fixture/debug mode).
///
/// Same root gate as activation; the caller pre-cleans the path. Used
/// with `--listen-path` when no supervisor passes a listener. Unlike
/// the systemd unit (0660 root:soda), the bound socket is world
/// accessible: connecting needs write permission and the fixture
/// driver runs as an unprivileged user, so confine access through the
/// socket directory's own permissions instead.
pub fn bind_listener(path: &str) -> io::Result<UnixListener> {
    // SAFETY: `geteuid` is async-signal-safe and infallible.
    if unsafe { libc::geteuid() } != 0 {
        return Err(io::Error::new(
            io::ErrorKind::PermissionDenied,
            "requires root and the soda-host systemd Unix socket",
        ));
    }
    let listener = UnixListener::bind(path)?;
    let mut perms = std::fs::metadata(path)?.permissions();
    perms.set_mode(0o777);
    std::fs::set_permissions(path, perms)?;
    Ok(listener)
}
