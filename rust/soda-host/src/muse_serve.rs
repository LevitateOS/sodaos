//! Muse launch/serve loop: daemon entry points over the native runtime.
//!
//! Port of the `MuseRuntime.Start` supervision path and the `MuseLaunch.Serve`
//! accept loop (`internal/host/terminal/muse_socket_linux.go`) plus the
//! listener setup from `internal/host/muse.go` (`Daemon.OpenMuseListener`).
//!
//! The runtime core (resolve/register/reserve/deliver/validators), request
//! decoding and exit encoding live in [`muse`](crate::muse) and peer
//! attestation in [`gmux_admission`](crate::gmux_admission); this module only
//! adds the daemon-called surface and reuses those helpers throughout.
//!
//! Shaping notes (contract signatures pin these):
//!
//! * [`MuseRuntime::start`] fuses Go `Start` with shell supervision but runs
//!   detached: the signature carries no stdio or control channel, so the
//!   child gets null stdio and no resize/signal supervisor. The interactive
//!   path stays [`muse::MuseLaunch::shell`](crate::muse::MuseLaunch::shell).
//! * [`MuseLaunch::serve`] validates launch descriptors (3 fds, Go parity)
//!   but the `Start` callable takes peer + request only, so decoded stdio is
//!   closed before dispatch. Wiring SCM_RIGHTS stdio through needs a
//!   signature change and stays a remainder.
//! * [`MuseRuntime::muse`] is the Go-name entry over `muse_operation`
//!   (validate/stop served, start/finish denied exactly like Go
//!   `projectOperation`).
//! * [`open_muse_listener`] reuses `prepare_muse_listener_dir` for the
//!   mkdir/empty/occupied gates, then binds unixpacket and chmods 0666. Go
//!   passes 0755 to `MkdirAll`; the reused helper follows umask (0755 under
//!   the standard 022). The daemon-nil case (`d.Muse == nil`) has no free
//!   form here; the daemon simply does not call this without a runtime.

use std::os::unix::io::{AsRawFd, FromRawFd, RawFd};
use std::os::unix::net::{UnixListener, UnixStream};
use std::time::{Duration, Instant};

use crate::gmux_admission::{close_pidfd, muse_peer};
use crate::muse::{
    muse_command_argv, muse_command_exit, muse_host_environment, muse_request_from_fd,
    prepare_muse_listener_dir, LaunchExit, LaunchRequest, MuseExecution, MuseHooks, MusePeer,
    MuseRuntime, NestedRegistration,
};
use crate::project::Executor;
use crate::texec::Delivery;

/// Supervised session horizon: Go `shell` runs under a 12h context and the
/// guest unit carries `RuntimeMaxSec=43200`.
const SESSION_SECS: u64 = 12 * 3600;
/// Post-spawn custody-return horizon: Go's 30s `Finish` context.
const CLEANUP_SECS: u64 = 30;
/// Accept-poll slice while watching for shutdown.
const ACCEPT_POLL_MS: i32 = 100;
/// Child wait-poll slice while watching the session deadline.
const WAIT_POLL_MS: u64 = 50;
/// Listen backlog for the launch socket.
const LISTEN_BACKLOG: i32 = 128;

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
    /// Attested launch to supervised execution to exit encoding.
    ///
    /// Mirrors Go `Start` + `shell`: prepare (validate, sanitize, resolve,
    /// verify, reserve, stage), spawn the podman boundary, deliver the
    /// credential once the unit is active, wait out the session, then finish
    /// (stop the unit, retire files, return custody). A failed finish
    /// overrides every outcome with cleanup-unconfirmed, exactly like Go's
    /// deferred `Finish`.
    ///
    /// Pre-completion failures surface as `Err` (daemon-observable, never
    /// wire-visible: the serve loop encodes those as denied); a supervised
    /// run returns `Ok` with the encoded exit. The wait is bounded by
    /// `deadline` like Go's cancelling 12h context: expiry kills the child
    /// and the kill signal folds into the exit code via `muse_command_exit`.
    pub fn start(
        &self,
        peer: &MusePeer,
        req: &LaunchRequest,
        deadline: Instant,
    ) -> Result<LaunchExit, String> {
        let mut execution = self.prepare_execution(peer, req, deadline)?;
        let mut child = match spawn_detached(&execution) {
            Ok(child) => child,
            Err(err) => return self.finish_start(&execution, Err(err)),
        };
        if let Err(err) = self.deliver_execution(&mut execution, deadline) {
            let _ = child.kill();
            let _ = child.wait();
            return self.finish_start(&execution, Err(err));
        }
        let result = muse_command_exit(wait_bounded(&mut child, deadline));
        self.finish_start(&execution, Ok(result))
    }

    /// Broker-selected operations on recorded `muse-project` boundaries.
    ///
    /// Go-name entry over `muse_operation`: `validate` re-attests the exact
    /// incarnation, `stop` tears the unit down without custody return, and
    /// `start`/`finish`/unknown actions deny (Go `projectOperation` serves
    /// only validate/stop). Success echoes the input delivery like Go's
    /// `(delivery, err)` return.
    pub fn muse(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        self.muse_operation(action, delivery, deadline)
    }

    /// Shell `defer Finish`: stop, retire, return custody; a failed return
    /// overrides the outcome with cleanup-unconfirmed.
    fn finish_start(
        &self,
        execution: &MuseExecution,
        outcome: Result<LaunchExit, String>,
    ) -> Result<LaunchExit, String> {
        let cleanup = Instant::now() + Duration::from_secs(CLEANUP_SECS);
        if self
            .stop_execution(
                &execution.binding,
                execution.caller.actor,
                &execution.lease.id,
                true,
                cleanup,
            )
            .is_err()
        {
            return Ok(LaunchExit::cleanup_unconfirmed());
        }
        outcome
    }
}

/// Spawn the prepared podman boundary with detached stdio.
///
/// Same argv/env construction as the interactive spawner (`muse_command_argv`
/// + `muse_host_environment`); stdio is null because [`MuseRuntime::start`]
/// carries no descriptors.
fn spawn_detached(execution: &MuseExecution) -> Result<std::process::Child, String> {
    let argv = muse_command_argv(
        &execution.caller,
        &execution.request,
        &execution.unit,
        &execution.path,
    );
    std::process::Command::new("/usr/bin/podman")
        .args(&argv)
        .env_clear()
        .envs(muse_host_environment())
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .spawn()
        .map_err(|e| format!("/usr/bin/podman failed: {e}"))
}

/// Wait for the child, killing it at the session deadline like Go's
/// cancelling command context.
fn wait_bounded(
    child: &mut std::process::Child,
    deadline: Instant,
) -> std::io::Result<std::process::ExitStatus> {
    loop {
        match child.try_wait()? {
            Some(status) => return Ok(status),
            None => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    return child.wait();
                }
                std::thread::sleep(Duration::from_millis(WAIT_POLL_MS));
            }
        }
    }
}

/// Dedicated unixpacket launch service: `S` starts executions, `R` registers
/// nested containers. Mirrors Go's `MuseLaunch{Start, Register}` func fields;
/// unlike Go the callables are mandatory, which collapses the nil checks into
/// the type system (same rationale as `MuseHooks`).
pub struct MuseLaunch<S, R> {
    start: S,
    register: R,
}

impl<S, R> MuseLaunch<S, R> {
    pub fn new(start: S, register: R) -> Self {
        MuseLaunch { start, register }
    }
}

/// Owned pidfd pin; Go defers `unix.Close(peer.PIDFD)`.
struct PidfdGuard(crate::gmux_admission::MusePeer);

impl Drop for PidfdGuard {
    fn drop(&mut self) {
        close_pidfd(&self.0);
    }
}

impl<S, R> MuseLaunch<S, R> {
    /// Accept loop until `is_shutdown` flips, then join every live
    /// connection (Go's `Serve` + `wg.Wait` via scoped threads).
    ///
    /// Per connection: attest the peer with `gmux_admission::muse_peer`,
    /// decode one request, dispatch register vs start, encode the outcome.
    /// Attest/decode/dispatch failures encode as denied; only accept-loop
    /// failures surface as `Err`, and those carry errno text only, never
    /// request or credential bytes. An accept error with shutdown set exits
    /// `Ok`, like Go's cancelled `AcceptUnix`.
    pub fn serve(
        &self,
        listener: UnixListener,
        is_shutdown: &dyn Fn() -> bool,
    ) -> Result<(), String>
    where
        S: Fn(&MusePeer, &LaunchRequest, Instant) -> Result<LaunchExit, String> + Sync,
        R: Fn(&MusePeer, &NestedRegistration, Instant) -> Result<(), String> + Sync,
    {
        listener
            .set_nonblocking(true)
            .map_err(|e| format!("muse accept failed: {e}"))?;
        let fd = listener.as_raw_fd();
        std::thread::scope(|scope| loop {
            if is_shutdown() {
                return Ok(());
            }
            let mut pfd = libc::pollfd {
                fd,
                events: libc::POLLIN,
                revents: 0,
            };
            // SAFETY: valid one-element pollfd array.
            let ready = unsafe { libc::poll(&mut pfd, 1, ACCEPT_POLL_MS) };
            if ready < 0 {
                let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
                if errno == libc::EINTR {
                    continue;
                }
                if is_shutdown() {
                    return Ok(());
                }
                return Err(format!("muse accept failed: errno {errno}"));
            }
            if ready == 0 {
                continue;
            }
            match listener.accept() {
                Ok((stream, _)) => {
                    scope.spawn(|| self.serve_one(stream));
                }
                Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => continue,
                Err(e) if e.kind() == std::io::ErrorKind::Interrupted => continue,
                Err(e) => {
                    if is_shutdown() {
                        return Ok(());
                    }
                    return Err(format!("muse accept failed: {e}"));
                }
            }
        })
    }

    /// One connection: encode the outcome, then close (Go's deferred
    /// `Encode` + `Close`).
    fn serve_one(&self, mut stream: UnixStream)
    where
        S: Fn(&MusePeer, &LaunchRequest, Instant) -> Result<LaunchExit, String> + Sync,
        R: Fn(&MusePeer, &NestedRegistration, Instant) -> Result<(), String> + Sync,
    {
        let result = self.serve_connection(stream.as_raw_fd());
        let _ = std::io::Write::write_all(&mut stream, result.encode_line().as_bytes());
    }

    /// Attest, decode, dispatch. Any failure is denied, never detailed.
    fn serve_connection(&self, conn: RawFd) -> LaunchExit
    where
        S: Fn(&MusePeer, &LaunchRequest, Instant) -> Result<LaunchExit, String> + Sync,
        R: Fn(&MusePeer, &NestedRegistration, Instant) -> Result<(), String> + Sync,
    {
        let attested = match muse_peer(conn) {
            Ok(peer) => peer,
            Err(_) => return LaunchExit::denied(),
        };
        let peer = MusePeer {
            pid: attested.pid,
            uid: attested.uid,
            gid: attested.gid,
            pidfd: attested.pidfd,
        };
        let _pin = PidfdGuard(attested);
        let received = match muse_request_from_fd(conn) {
            Ok(received) => received,
            Err(_) => return LaunchExit::denied(),
        };
        // Go serves register under the parent context and launches under a
        // 12h shell context; one session horizon covers both dispatches.
        let session = Instant::now() + Duration::from_secs(SESSION_SECS);
        if let Some(register) = &received.request.register {
            return match (self.register)(&peer, register, session) {
                Ok(()) => LaunchExit {
                    code: 0,
                    error: String::new(),
                },
                Err(_) => LaunchExit::denied(),
            };
        }
        // Decoded stdio validated the launch shape, then drops here: the
        // `Start` callable takes peer + request only (see module notes).
        match (self.start)(&peer, &received.request, session) {
            Ok(exit) => exit,
            Err(_) => LaunchExit::denied(),
        }
    }
}

/// Unixpacket launch listener over an empty interface directory.
///
/// Mirrors Go `OpenMuseListener` check order exactly: mkdir -p, empty-dir
/// gate, occupied-socket refusal (via `prepare_muse_listener_dir`), bind,
/// listen, chmod 0666 (closing the socket on chmod failure like Go).
pub fn open_muse_listener(socket_path: &str) -> Result<UnixListener, String> {
    prepare_muse_listener_dir(socket_path)?;
    #[cfg(target_os = "linux")]
    let socktype = libc::SOCK_SEQPACKET | libc::SOCK_CLOEXEC;
    #[cfg(not(target_os = "linux"))]
    let socktype = libc::SOCK_SEQPACKET;
    // SAFETY: socket with a valid type.
    let fd = unsafe { libc::socket(libc::AF_UNIX, socktype, 0) };
    if fd < 0 {
        return Err(format!(
            "muse launch socket unavailable: {}",
            std::io::Error::last_os_error()
        ));
    }
    #[cfg(not(target_os = "linux"))]
    // SAFETY: fcntl on the owned fd.
    unsafe {
        let flags = libc::fcntl(fd, libc::F_GETFD);
        if flags >= 0 {
            libc::fcntl(fd, libc::F_SETFD, flags | libc::FD_CLOEXEC);
        }
    }
    let mut addr: libc::sockaddr_un = unsafe { std::mem::zeroed() };
    addr.sun_family = libc::AF_UNIX as libc::sa_family_t;
    let bytes = socket_path.as_bytes();
    if bytes.contains(&0) {
        unsafe {
            libc::close(fd);
        }
        return Err("muse launch socket path invalid".to_string());
    }
    if bytes.len() >= addr.sun_path.len() {
        unsafe {
            libc::close(fd);
        }
        return Err("muse launch socket path too long".to_string());
    }
    for (i, b) in bytes.iter().enumerate() {
        addr.sun_path[i] = *b as libc::c_char;
    }
    let addr_len = (std::mem::size_of::<libc::sa_family_t>() + bytes.len() + 1) as libc::socklen_t;
    // SAFETY: bind + listen on the owned socket fd.
    let bound = unsafe {
        libc::bind(
            fd,
            &addr as *const _ as *const libc::sockaddr,
            addr_len,
        )
    };
    if bound != 0 {
        let err = std::io::Error::last_os_error();
        unsafe {
            libc::close(fd);
        }
        return Err(format!("muse launch socket bind failed: {err}"));
    }
    // SAFETY: listen on the bound fd.
    if unsafe { libc::listen(fd, LISTEN_BACKLOG) } != 0 {
        let err = std::io::Error::last_os_error();
        unsafe {
            libc::close(fd);
        }
        return Err(format!("muse launch socket listen failed: {err}"));
    }
    // Go's os.Chmod(path, 0666) via the pinned fd; close on failure like Go.
    // SAFETY: fchmod on the owned fd.
    if unsafe { libc::fchmod(fd, 0o666) } != 0 {
        let err = std::io::Error::last_os_error();
        unsafe {
            libc::close(fd);
        }
        return Err(format!("muse launch socket unavailable: {err}"));
    }
    // SAFETY: fd is an open listening socket; ownership moves to the listener.
    Ok(unsafe { UnixListener::from_raw_fd(fd) })
}
