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
use crate::gmux_admission::{body_limit_for, AdmissionGate, RequestHead, TerminalGate};
use crate::gmux_backend::ExecBackend;
use crate::gmux_routes::{dispatch, error_response, DaemonConfig, RouteOutcome};
use std::io;
use std::io::{Read, Write};
use std::os::unix::fs::PermissionsExt;
use std::os::unix::net::{UnixListener, UnixStream};
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Duration;

/// Max request head in bytes (Go: MaxHeaderBytes 8192).
const MAX_HEADER: usize = 8192;
/// Head read deadline (Go: ReadHeaderTimeout 5s).
const HEADER_TIMEOUT: Duration = Duration::from_secs(5);
/// Body read deadline (Rust http.rs precedent: 30s).
const BODY_TIMEOUT: Duration = Duration::from_secs(30);

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

enum ParseFailure {
    Invalid,
    HeadersTooLarge,
}

/// Parse the head, then read the body under the route's own limit.
/// Returns the head, the body bytes kept, and whether the body arrived
/// complete (a `false` is a decode failure for every subsystem, exactly
/// like Go's `MaxBytesReader` tripping before strict decode).
fn read_request(stream: &mut UnixStream) -> Result<(RequestHead, Vec<u8>, bool), ParseFailure> {
    stream
        .set_read_timeout(Some(HEADER_TIMEOUT))
        .map_err(|_| ParseFailure::Invalid)?;
    let mut head = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err(ParseFailure::Invalid),
            Ok(_) => {
                head.push(byte[0]);
                if head.len() > MAX_HEADER {
                    return Err(ParseFailure::HeadersTooLarge);
                }
                if head.ends_with(b"\r\n\r\n") {
                    break;
                }
            }
            Err(_) => return Err(ParseFailure::Invalid),
        }
    }
    let mut lines = head.split(|b| *b == b'\n');
    let request_line = lines.next().ok_or(ParseFailure::Invalid)?;
    let request_line = request_line.strip_suffix(b"\r").unwrap_or(request_line);
    let mut parts = request_line.split(|b| *b == b' ');
    let method = parts.next().ok_or(ParseFailure::Invalid)?;
    let target = parts.next().ok_or(ParseFailure::Invalid)?;
    if parts.next().is_none() {
        return Err(ParseFailure::Invalid);
    }
    let method = std::str::from_utf8(method)
        .map_err(|_| ParseFailure::Invalid)?
        .to_string();
    let target = std::str::from_utf8(target).map_err(|_| ParseFailure::Invalid)?;
    let (raw_path, has_query) = match target.find('?') {
        Some(i) => (&target[..i], true),
        None => (target, false),
    };
    let escaped = raw_path.contains('%');
    let path = percent_decode(raw_path).map_err(|_| ParseFailure::Invalid)?;
    let mut origin_present = false;
    let mut upgrade_websocket = false;
    let mut ws_key = None;
    let mut content_lengths = Vec::new();
    let mut chunked = false;
    for line in lines {
        let line = line.strip_suffix(b"\r").unwrap_or(line);
        if line.is_empty() {
            continue;
        }
        let Some(colon) = line.iter().position(|b| *b == b':') else {
            return Err(ParseFailure::Invalid);
        };
        let (name, value) = (&line[..colon], &line[colon + 1..]);
        if name.eq_ignore_ascii_case(b"origin") {
            // Any Origin line counts, even empty (Go: Header.Values != 0).
            origin_present = true;
        } else if name.eq_ignore_ascii_case(b"upgrade") {
            if value
                .split(|b| *b == b',')
                .any(|token| token.trim_ascii().eq_ignore_ascii_case(b"websocket"))
            {
                upgrade_websocket = true;
            }
        } else if name.eq_ignore_ascii_case(b"sec-websocket-key") {
            if ws_key.is_none() {
                ws_key = std::str::from_utf8(value.trim_ascii())
                    .map_err(|_| ParseFailure::Invalid)?
                    .to_string()
                    .into();
            }
        } else if name.eq_ignore_ascii_case(b"content-length") {
            content_lengths.push(
                std::str::from_utf8(value.trim_ascii())
                    .map_err(|_| ParseFailure::Invalid)?
                    .to_string(),
            );
        } else if name.eq_ignore_ascii_case(b"transfer-encoding")
            && value.trim_ascii().eq_ignore_ascii_case(b"chunked")
        {
            chunked = true;
        }
    }
    if content_lengths.len() > 1 || chunked {
        return Err(ParseFailure::Invalid);
    }
    let content_length = content_lengths
        .first()
        .map(|v| v.parse::<usize>().map_err(|_| ParseFailure::Invalid))
        .transpose()?
        .unwrap_or(0);
    let request = RequestHead {
        method,
        path: path.clone(),
        has_query,
        escaped,
        origin_present,
        upgrade_websocket,
        ws_key,
    };
    // Terminal upgrades hijack after the head like Go: no body is consumed.
    if request.method == "GET" && request.path == "/terminal" {
        return Ok((request, Vec::new(), true));
    }
    let limit = body_limit_for(&path);
    stream
        .set_read_timeout(Some(BODY_TIMEOUT))
        .map_err(|_| ParseFailure::Invalid)?;
    let want = content_length.min(limit + 1);
    let mut body = vec![0u8; want];
    stream
        .read_exact(&mut body)
        .map_err(|_| ParseFailure::Invalid)?;
    Ok((request, body, content_length <= limit))
}

/// Discard the remainder of an over-limit head: until the blank line,
/// a 1 MiB cap, or any read failure. Bounded so a hostile client
/// cannot park the connection thread past the read deadline.
fn drain_head(stream: &mut UnixStream) {
    const DRAIN_CAP: usize = 1 << 20;
    stream.set_read_timeout(Some(HEADER_TIMEOUT)).ok();
    let mut tail = [0u8; 4];
    let mut discarded = 0usize;
    let mut chunk = [0u8; 1024];
    loop {
        match stream.read(&mut chunk) {
            Ok(0) => return,
            Ok(n) => {
                discarded += n;
                for &b in &chunk[..n] {
                    tail.copy_within(1.., 0);
                    tail[3] = b;
                    if tail == *b"\r\n\r\n" {
                        return;
                    }
                }
                if discarded >= DRAIN_CAP {
                    return;
                }
            }
            Err(_) => return,
        }
    }
}

fn percent_decode(input: &str) -> Result<String, ()> {
    let bytes = input.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len() {
                return Err(());
            }
            let hex = std::str::from_utf8(&bytes[i + 1..i + 3]).map_err(|_| ())?;
            out.push(u8::from_str_radix(hex, 16).map_err(|_| ())?);
            i += 3;
        } else {
            out.push(bytes[i]);
            i += 1;
        }
    }
    String::from_utf8(out).map_err(|_| ())
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
