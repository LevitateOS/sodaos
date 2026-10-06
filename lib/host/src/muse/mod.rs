//! Native Muse execution runtime and launch socket.
//!
//! Port of `internal/host/terminal/muse_types.go`,
//! `muse_linux.go`, `muse_execution_linux.go`, `muse_socket_linux.go`,
//! `muse_state_linux.go`, `muse_binary_linux.go`, the launch policy from
//! `internal/identity/launch.go` + `selection.go`, and the daemon Muse
//! wiring from `internal/host/muse.go` (authorization/selection over
//! broker connections plus launch-listener setup).
//!
//! Broker custody callbacks arrive through [`MuseHooks`]; unlike Go's
//! nullable func fields the hooks are mandatory, which collapses the
//! nil-hook denials into the type system (the daemon always wires all
//! six). `MuseRuntime::prepare_execution` returns a plain
//! [`MuseExecution`] record; the argv, delivery, control and finish steps
//! are separate methods so routes can stage them around process spawn.

use std::collections::HashMap;
use std::os::unix::io::RawFd;
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::{Duration, Instant};

use crate::json::{self, Value};
use crate::project::Executor;
use crate::terminal;
#[cfg(test)]
use crate::terminal::{AcquireRequest, Binding, Delivery, Lease};

mod wire;

pub use self::wire::{
    LaunchControl, LaunchExit, LaunchRequest, NestedRegistration, MUSE_CONNECTION_SETTING,
    MUSE_LAUNCH_SOCKET,
};

mod args;

pub use self::args::muse_arguments;

mod connection;

pub use self::connection::{muse_connection_authorized, select_muse_connection, MuseConnection};

mod runtime_types;

pub use self::runtime_types::{
    MuseCaller, MuseExecution, MuseHooks, MuseNested, MusePeer, MuseRuntime,
};

mod validate;

pub use self::validate::{
    host_go_arch, muse_account_modes, muse_account_node, muse_child_pid, muse_delivery_valid,
    muse_elf, muse_host_environment, muse_mapped_uid, muse_passwd_valid, muse_project_cgroup,
    muse_project_credential_root, muse_readonly_mount, muse_registration_valid, muse_signal,
    MUSE_CHILD_INSPECT, MUSE_INSPECT,
};

mod argv;

pub use self::argv::{muse_command_argv, unit_active_argv, unit_invocation_argv};

// ---------- runtime ----------

mod execution;
mod inspect;
mod nested;
mod observe;
mod operate;
mod ops;

pub use self::ops::{muse_resize, prepare_muse_listener_dir, state_container};

mod request;

pub use self::request::{muse_descriptors_valid, muse_request_from_fd, MuseRequest};

mod resolve;
mod socket;
mod stage;
mod stop;

pub(in crate::muse) use self::socket::close_fds;
pub use self::socket::{muse_peer_from_fd, parse_unix_rights};

pub(in crate::muse) use self::inspect::{
    muse_peer_alive, sleep_until, MuseInspection, MUSE_INSPECTION_SPECS,
};

/// `museCommandExit`: shell wait status to launch outcome.
pub fn muse_command_exit(status: std::io::Result<std::process::ExitStatus>) -> LaunchExit {
    use std::os::unix::process::ExitStatusExt;
    match status {
        Ok(status) => match status.code() {
            Some(code) => LaunchExit {
                code,
                error: String::new(),
            },
            None => LaunchExit {
                code: 128 + status.signal().unwrap_or(0),
                error: String::new(),
            },
        },
        Err(_) => LaunchExit {
            code: 1,
            error: String::new(),
        },
    }
}

/// Split one complete JSON object off the front of `buffer`, tracking
/// strings and escapes like a streaming decoder. Returns the object
/// length when balanced.
pub fn split_json_object(buffer: &[u8]) -> Option<usize> {
    let mut depth = 0i32;
    let mut in_string = false;
    let mut escape = false;
    let mut started = false;
    for (i, &b) in buffer.iter().enumerate() {
        if in_string {
            if escape {
                escape = false;
            } else if b == b'\\' {
                escape = true;
            } else if b == b'"' {
                in_string = false;
            }
            continue;
        }
        match b {
            b'"' => in_string = true,
            b'{' => {
                depth += 1;
                started = true;
            }
            b'}' => {
                depth -= 1;
                if started && depth == 0 {
                    return Some(i + 1);
                }
                if depth < 0 {
                    return None;
                }
            }
            _ => {}
        }
    }
    None
}

/// `MuseLaunch`: dedicated SOCK_SEQPACKET launch listener service.
/// Served behind an [`Arc`] so each connection runs detached, like Go's
/// per-connection goroutine.
pub struct MuseLaunch<E, H> {
    pub runtime: MuseRuntime<E, H>,
}

impl<E, H> MuseLaunch<E, H> {
    pub fn new(runtime: MuseRuntime<E, H>) -> Self {
        MuseLaunch { runtime }
    }
}

impl<E: Executor + Send + Sync + 'static, H: MuseHooks + Send + Sync + 'static> MuseLaunch<E, H> {
    /// `MuseLaunch.Serve`: accept loop until `shutdown` is set, then join
    /// every live connection (Go's `wg.Wait`).
    /// `listen_fd` is a bound SOCK_SEQPACKET listener (owned by the caller).
    pub fn serve(
        self: &std::sync::Arc<Self>,
        listen_fd: RawFd,
        shutdown: &AtomicBool,
    ) -> Result<(), String> {
        // Non-blocking accept with a stop poll, mirroring Go's
        // context-cancelled `AcceptUnix`.
        unsafe {
            let flags = libc::fcntl(listen_fd, libc::F_GETFL);
            if flags >= 0 {
                libc::fcntl(listen_fd, libc::F_SETFL, flags | libc::O_NONBLOCK);
            }
        }
        let mut handles = Vec::new();
        let outcome = loop {
            if shutdown.load(Ordering::SeqCst) {
                break Ok(());
            }
            let mut pfd = libc::pollfd {
                fd: listen_fd,
                events: libc::POLLIN,
                revents: 0,
            };
            // SAFETY: valid one-element pollfd array.
            let ready = unsafe { libc::poll(&mut pfd, 1, 100) };
            if ready < 0 {
                let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
                if errno == libc::EINTR {
                    continue;
                }
                break Err(format!("muse accept failed: errno {errno}"));
            }
            if ready == 0 {
                continue;
            }
            // SAFETY: accept on a listening unix socket.
            let conn =
                unsafe { libc::accept(listen_fd, std::ptr::null_mut(), std::ptr::null_mut()) };
            if conn < 0 {
                let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
                if errno == libc::EAGAIN || errno == libc::EWOULDBLOCK || errno == libc::EINTR {
                    continue;
                }
                break Err(format!("muse accept failed: errno {errno}"));
            }
            let service = self.clone();
            handles.push(std::thread::spawn(move || {
                service.serve_one(conn, &service)
            }));
        };
        for handle in handles {
            let _ = handle.join();
        }
        outcome
    }

    fn serve_one(&self, conn: RawFd, service: &std::sync::Arc<Self>) {
        let result = self.serve_connection(conn, service);
        // `json.Encoder.Encode(result)`: framed with one newline.
        let line = result.encode_line();
        let bytes = line.as_bytes();
        let mut written = 0;
        while written < bytes.len() {
            // SAFETY: send on the connected fd.
            let n = unsafe {
                libc::send(
                    conn,
                    bytes[written..].as_ptr() as *const libc::c_void,
                    bytes.len() - written,
                    libc::MSG_NOSIGNAL,
                )
            };
            if n <= 0 {
                break;
            }
            written += n as usize;
        }
        // SAFETY: close the accepted fd exactly once.
        unsafe {
            libc::close(conn);
        }
    }

    fn serve_connection(&self, conn: RawFd, service: &std::sync::Arc<Self>) -> LaunchExit {
        let peer = match muse_peer_from_fd(conn) {
            Ok(peer) => peer,
            Err(_) => return LaunchExit::denied(),
        };
        // SAFETY: close the pidfd exactly once at the end.
        struct Closer(RawFd);
        impl Drop for Closer {
            fn drop(&mut self) {
                unsafe {
                    libc::close(self.0);
                }
            }
        }
        let _pidfd = Closer(peer.pidfd);
        let received = match muse_request_from_fd(conn) {
            Ok(received) => received,
            Err(_) => return LaunchExit::denied(),
        };
        if let Some(register) = &received.request.register {
            let deadline = Instant::now() + Duration::from_secs(30);
            return match self.runtime.register_nested(&peer, register, deadline) {
                Ok(()) => LaunchExit {
                    code: 0,
                    error: String::new(),
                },
                Err(_) => LaunchExit::denied(),
            };
        }
        let [stdin, stdout, stderr] = received.files;
        let (Some(stdin), Some(stdout), Some(stderr)) = (stdin, stdout, stderr) else {
            return LaunchExit::denied();
        };
        self.shell(
            conn,
            &peer,
            &received.request,
            [stdin, stdout, stderr],
            service,
        )
    }

    /// `MuseLaunch.shell`: spawn, deliver, supervise, finish.
    pub fn shell(
        &self,
        conn: RawFd,
        peer: &MusePeer,
        request: &LaunchRequest,
        stdio: [std::fs::File; 3],
        service: &std::sync::Arc<Self>,
    ) -> LaunchExit {
        self.shell_inner(conn, peer, request, stdio, Some(service.clone()))
    }

    fn shell_inner(
        &self,
        conn: RawFd,
        peer: &MusePeer,
        request: &LaunchRequest,
        stdio: [std::fs::File; 3],
        supervisor_owner: Option<std::sync::Arc<Self>>,
    ) -> LaunchExit {
        use std::os::unix::io::AsRawFd;
        let session_end = Instant::now() + Duration::from_secs(12 * 3600);
        let mut execution = match self.runtime.prepare_execution(peer, request, session_end) {
            Ok(execution) => execution,
            Err(_) => return LaunchExit::denied(),
        };
        // The supervisor resizes through the parent's stdin handle.
        // SAFETY: dup before spawn transfers ownership of stdio to the child.
        let stdin_fd = unsafe { libc::dup(stdio[0].as_raw_fd()) };
        let mut child = match spawn_execution(&execution, stdio) {
            Ok(child) => child,
            Err(_) => {
                unsafe {
                    libc::close(stdin_fd);
                }
                self.finish_unconfirmed(&execution);
                return LaunchExit::denied();
            }
        };
        if self
            .runtime
            .deliver_execution(&mut execution, session_end)
            .is_err()
        {
            let _ = child.kill();
            let _ = child.wait();
            unsafe {
                libc::close(stdin_fd);
            }
            self.finish_unconfirmed(&execution);
            return LaunchExit::denied();
        }
        // Control supervisor: any channel failure kills the session, like
        // Go's `museControls` cancel.
        if let Some(owner) = supervisor_owner {
            // SAFETY: dup the conn for the supervisor; it closes its copy.
            let conn_dup = unsafe { libc::dup(conn) };
            let child_pid = child.id() as i32;
            let snapshot = execution.clone();
            std::thread::spawn(move || {
                owner.control_loop(conn_dup, stdin_fd, child_pid, &snapshot, session_end);
                unsafe {
                    libc::close(conn_dup);
                    libc::close(stdin_fd);
                    libc::kill(child_pid, libc::SIGKILL);
                }
            });
        } else {
            unsafe {
                libc::close(stdin_fd);
            }
        }
        let mut result = muse_command_exit(child.wait());
        let cleanup = Instant::now() + Duration::from_secs(30);
        if self
            .runtime
            .stop_execution(
                &execution.binding,
                execution.caller.actor,
                &execution.lease.id,
                true,
                cleanup,
            )
            .is_err()
        {
            result = LaunchExit::cleanup_unconfirmed();
        }
        result
    }

    fn finish_unconfirmed(&self, execution: &MuseExecution) {
        let cleanup = Instant::now() + Duration::from_secs(30);
        let _ = self.runtime.stop_execution(
            &execution.binding,
            execution.caller.actor,
            &execution.lease.id,
            true,
            cleanup,
        );
    }

    /// Control supervisor (`museControls`): streaming strict control
    /// messages, 1MB total, unknown fields rejected. Any failure ends the
    /// loop; the spawner kills the session afterwards.
    pub fn control_loop(
        &self,
        conn: RawFd,
        stdin_fd: RawFd,
        child_pid: i32,
        execution: &MuseExecution,
        session_end: Instant,
    ) {
        let mut buffer = Vec::new();
        let mut chunk = [0u8; 4096];
        loop {
            if Instant::now() >= session_end {
                return;
            }
            // SAFETY: recv into the scratch chunk.
            let n = unsafe {
                libc::recv(
                    conn,
                    chunk.as_mut_ptr() as *mut libc::c_void,
                    chunk.len(),
                    0,
                )
            };
            if n <= 0 {
                return;
            }
            buffer.extend_from_slice(&chunk[..n as usize]);
            if buffer.len() > 1 << 20 {
                return;
            }
            loop {
                let trim: Vec<u8> = buffer
                    .iter()
                    .skip_while(|b| b.is_ascii_whitespace())
                    .cloned()
                    .collect();
                if trim.is_empty() {
                    buffer.clear();
                    break;
                }
                if trim[0] != b'{' {
                    return;
                }
                let Some(len) = split_json_object(&trim) else {
                    buffer = trim;
                    break;
                };
                let control = match LaunchControl::decode(&trim[..len]) {
                    Ok(control) => control,
                    Err(_) => return,
                };
                if self
                    .runtime
                    .control_execution(execution, stdin_fd, Some(child_pid), &control, session_end)
                    .is_err()
                {
                    return;
                }
                buffer = trim[len..].to_vec();
            }
        }
    }
}

/// Spawn the prepared execution with owned stdio files.
fn spawn_execution(
    execution: &MuseExecution,
    stdio: [std::fs::File; 3],
) -> Result<std::process::Child, String> {
    let [stdin, stdout, stderr] = stdio;
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
        .stdin(std::process::Stdio::from(stdin))
        .stdout(std::process::Stdio::from(stdout))
        .stderr(std::process::Stdio::from(stderr))
        .spawn()
        .map_err(|e| format!("/usr/bin/podman failed: {e}"))
}

/// Tolerant `map[string][]byte` decode for nested config views
/// (`encoding/json` into `map[string][]byte`: base64 strings or numeric
/// arrays, like [`Kind::Bytes`](crate::json::Kind::Bytes)).
pub fn decode_config_view(body: &[u8]) -> Result<HashMap<String, Vec<u8>>, String> {
    let v = json::decode_tolerant(body).map_err(|_| terminal::err_denied())?;
    let Some(fields) = v.as_object() else {
        return Err(terminal::err_denied());
    };
    let mut out = HashMap::with_capacity(fields.len());
    for (key, value) in fields {
        let bytes = match value {
            Value::Null => Vec::new(),
            Value::Str(s) => {
                crate::ssh::b64_decode_go(s.as_bytes()).map_err(|_| terminal::err_denied())?
            }
            Value::Array(items) => {
                let mut bytes = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        Value::Number(lit) => {
                            let n: i64 = lit.parse().map_err(|_| terminal::err_denied())?;
                            if !(0..=255).contains(&n) {
                                return Err(terminal::err_denied());
                            }
                            bytes.push(n as u8);
                        }
                        _ => return Err(terminal::err_denied()),
                    }
                }
                bytes
            }
            _ => return Err(terminal::err_denied()),
        };
        out.insert(key.clone(), bytes);
    }
    Ok(out)
}

#[cfg(test)]
mod tests;
