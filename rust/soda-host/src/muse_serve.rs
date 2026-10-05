//! Muse launch entries: supervised start, broker dispatch, listener setup.
//!
//! Daemon-called surface over the already-ported runtime in
//! [`muse`](crate::muse): [`MuseRuntime::start`] fuses Go `Start` with shell
//! supervision, [`MuseRuntime::muse`] is the Go-name broker-dispatch entry,
//! and [`open_muse_listener`] completes `prepare_muse_listener_dir` with the
//! unixpacket bind + chmod from Go `Daemon.OpenMuseListener`
//! (`internal/host/muse.go`). The accept loop itself (`MuseLaunch::serve`),
//! caller resolution, registration, reserve/deliver and all validators
//! already exist in `muse` and are reused, never redefined here.
//!
//! Shaping notes (contract signatures pin these):
//!
//! * `start` runs detached: the signature carries no stdio or control
//!   channel, so the child gets null stdio and no resize/signal supervisor.
//!   The interactive path stays
//!   [`muse::MuseLaunch::shell`](crate::muse::MuseLaunch::shell).
//! * `muse` delegates to `muse_operation` (validate/stop served,
//!   start/finish denied exactly like Go `projectOperation`).
//! * `open_muse_listener` reuses `prepare_muse_listener_dir` for the
//!   mkdir/empty/occupied gates. Go passes 0755 to `MkdirAll`; the reused
//!   helper follows umask (0755 under the standard 022). The daemon-nil
//!   case (`d.Muse == nil`) has no free form here; the daemon simply does
//!   not call this without a runtime.

use std::os::unix::io::FromRawFd;
use std::os::unix::net::UnixListener;
use std::time::{Duration, Instant};

use crate::muse::{
    muse_command_argv, muse_command_exit, muse_host_environment, prepare_muse_listener_dir,
    LaunchExit, LaunchRequest, MuseExecution, MuseHooks, MusePeer, MuseRuntime,
};
use crate::project::Executor;
use crate::texec::Delivery;

/// Post-spawn custody-return horizon: Go's 30s `Finish` context.
const CLEANUP_SECS: u64 = 30;
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
/// Same argv/env construction as the interactive spawner
/// (`muse_command_argv` with `muse_host_environment`); stdio is null because
/// [`MuseRuntime::start`] carries no descriptors.
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
    let bound = unsafe { libc::bind(fd, &addr as *const _ as *const libc::sockaddr, addr_len) };
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
    // Go's os.Chmod(path, 0666), closing the listener on failure. Path-based
    // on purpose: fchmod on a unix socket fd is a silent no-op on Linux.
    let c_path = std::ffi::CString::new(socket_path)
        .map_err(|_| "muse launch socket path invalid".to_string())?;
    // SAFETY: chmod on a valid NUL-terminated path.
    if unsafe { libc::chmod(c_path.as_ptr(), 0o666) } != 0 {
        let err = std::io::Error::last_os_error();
        unsafe {
            libc::close(fd);
        }
        return Err(format!("muse launch socket unavailable: {err}"));
    }
    // SAFETY: fd is an open listening socket; ownership moves to the listener.
    Ok(unsafe { UnixListener::from_raw_fd(fd) })
}
