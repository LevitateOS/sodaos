// Unix-socket HTTP server for the Rust daemon mux (PR26).
//
// Follows the established Rust Unix-HTTP pattern (soda-identity/src/http.rs):
// a nonblocking accept loop, one thread per connection, connections closed
// after each response, fixed header/body caps with read timeouts. Request
// parsing mirrors Go's net/http surface the daemon depends on (method,
// target, query/escape/Origin admission inputs, single Content-Length, no
// chunked); dispatch and envelopes live in gmux_routes.
//
// Served from the root systemd socket like cmd/soda-host/main.go: the
// binary main obtains the listener from fd 3 (see `systemd_listener`),
// mounts one `Server`, and stops it through the shutdown flag. Terminal
// upgrades keep their connection thread as the pump thread, mirroring Go
// where the handler goroutine pumps until the session ends.
use crate::gmux_admission::{AdmissionGate, TerminalGate};
use crate::gmux_backend::ExecBackend;
use crate::gmux_routes::{dispatch, error_response, DaemonConfig, RouteOutcome};
use std::io;
use std::io::Write;
use std::os::unix::fs::PermissionsExt;
use std::os::unix::net::{UnixListener, UnixStream};
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Duration;

#[path = "daemon/http.rs"]
mod http;

use self::http::{drain_head, read_request, ParseFailure};

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

    /// Connections currently being handled (binary main drains on this).
    pub fn inflight(&self) -> usize {
        self.inflight.load(Ordering::SeqCst)
    }

    /// Serve until `shutdown`. Listener errors fail fast after a short
    /// burst so the supervisor restarts a deaf daemon (http.rs precedent,
    /// mirroring Go's Serve returning on fatal listener errors).
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
                    let backend = Arc::clone(&self.backend);
                    let config = self.config.clone();
                    let gate = Arc::clone(&self.gate);
                    let terminal_gate = Arc::clone(&self.terminal_gate);
                    let inflight = Arc::clone(&self.inflight);
                    std::thread::spawn(move || {
                        handle_connection(stream, backend.as_ref(), &config, &gate, &terminal_gate);
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
                    fatal_errors = 0;
                    std::thread::sleep(Duration::from_millis(50));
                }
                Err(_) => {
                    if self.shutdown.load(Ordering::SeqCst) {
                        return;
                    }
                    fatal_errors += 1;
                    if fatal_errors >= 20 {
                        return;
                    }
                    std::thread::sleep(Duration::from_millis(50));
                }
            }
        }
    }
}

fn handle_connection<B: ExecBackend>(
    mut stream: UnixStream,
    backend: &B,
    config: &DaemonConfig,
    gate: &AdmissionGate,
    terminal_gate: &TerminalGate,
) {
    match read_request(&mut stream) {
        Ok((head, body, body_complete)) => {
            match dispatch(
                backend,
                config,
                gate,
                terminal_gate,
                &head,
                &body,
                body_complete,
            ) {
                RouteOutcome::Respond(response) => {
                    let _ = stream.write_all(&response);
                }
                RouteOutcome::TerminalUpgrade {
                    response,
                    session,
                    slot,
                } => {
                    if stream.write_all(&response).is_err() {
                        return;
                    }
                    // The slot lives in this thread until the pump returns,
                    // mirroring Go's deferred unregister after pumpIO.
                    let _slot = slot;
                    let _ = backend.pump_terminal(stream, session);
                }
            }
        }
        Err(ParseFailure::Invalid) => {
            let _ = stream.write_all(&error_response(400, "invalid request"));
        }
        Err(ParseFailure::HeadersTooLarge) => {
            // Drain the rest of the head so the client observes the 431
            // instead of a reset (the response must not race in-flight
            // request bytes on a connection we are about to close).
            drain_head(&mut stream);
            let _ = stream.write_all(&error_response(431, "header too large"));
        }
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
