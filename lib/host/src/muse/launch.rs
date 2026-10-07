use std::os::fd::{AsRawFd, BorrowedFd, OwnedFd, RawFd};
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::{Duration, Instant};

use crate::project::Executor;

use super::{
    muse_command_exit, muse_peer_from_fd, muse_request_from_fd, spawn_execution, split_json_object,
    JsonPacket, LaunchControl, LaunchExit, LaunchRequest, MuseExecution, MuseHooks, MusePeer,
    MuseRuntime,
};

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
            // SAFETY: `listen_fd` remains owned by the caller for this serve call.
            let listener = unsafe { BorrowedFd::borrow_raw(listen_fd) };
            let conn = rustix::net::accept_with(listener, rustix::net::SocketFlags::CLOEXEC);
            let conn = match conn {
                Ok(conn) => conn,
                Err(errno)
                    if errno == rustix::io::Errno::AGAIN || errno == rustix::io::Errno::INTR =>
                {
                    continue;
                }
                Err(errno) => break Err(format!("muse accept failed: {errno}")),
            };
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

    fn serve_one(&self, conn: OwnedFd, service: &std::sync::Arc<Self>) {
        let conn_fd = conn.as_raw_fd();
        let result = self.serve_connection(conn_fd, service);
        // `json.Encoder.Encode(result)`: framed with one newline.
        let line = result.encode_line();
        let bytes = line.as_bytes();
        let mut written = 0;
        while written < bytes.len() {
            // SAFETY: send on the connected fd.
            let n = unsafe {
                libc::send(
                    conn_fd,
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
    }

    fn serve_connection(&self, conn: RawFd, service: &std::sync::Arc<Self>) -> LaunchExit {
        let peer = match muse_peer_from_fd(conn) {
            Ok(peer) => peer,
            Err(_) => return LaunchExit::denied(),
        };
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
                let len = match split_json_object(&trim) {
                    JsonPacket::Complete(len) => len,
                    JsonPacket::Incomplete => {
                        buffer = trim;
                        break;
                    }
                    JsonPacket::Invalid => return,
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
