use std::os::fd::{AsRawFd, BorrowedFd, FromRawFd, OwnedFd, RawFd};
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::{Duration, Instant};

use crate::project::Executor;

use super::{
    muse_command_exit, muse_peer_from_fd, muse_request_from_fd, spawn_execution, split_json_object,
    JsonPacket, LaunchControl, LaunchExit, LaunchRequest, MuseExecution, MuseHooks, MusePeer,
    MuseRuntime,
};

// Keep the launch listener's custody finite even before request admission.
const MAX_SERVE_WORKERS: usize = 128;
const CONTROL_POLL_MS: i32 = 100;

/// `MuseLaunch`: dedicated SOCK_SEQPACKET launch listener service.
/// Served behind an [`Arc`] so each admitted connection has an owned
/// worker handle, like Go's per-connection goroutine.
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
        let mut handles: Vec<std::thread::JoinHandle<()>> = Vec::new();
        let worker_cancel = std::sync::Arc::new(AtomicBool::new(false));
        let mut worker_panicked = false;
        let outcome = loop {
            // Completed workers are joined while the listener is live, so
            // churn does not accumulate completed JoinHandles.
            let mut index = 0;
            while index < handles.len() {
                if handles[index].is_finished() {
                    let handle = handles.swap_remove(index);
                    if handle.join().is_err() {
                        worker_panicked = true;
                    }
                } else {
                    index += 1;
                }
            }
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
            if handles.len() >= MAX_SERVE_WORKERS {
                // Admission is full. Dropping the accepted descriptor gives
                // this peer EOF without creating another worker.
                drop(conn);
                continue;
            }
            let service = self.clone();
            let cancel = worker_cancel.clone();
            match std::thread::Builder::new()
                .spawn(move || service.serve_one(conn, &service, cancel))
            {
                Ok(handle) => handles.push(handle),
                Err(error) => break Err(format!("muse worker start failed: {error}")),
            }
        };
        worker_cancel.store(true, Ordering::SeqCst);
        for handle in handles {
            worker_panicked |= handle.join().is_err();
        }
        if worker_panicked && outcome.is_ok() {
            Err("muse launch worker panicked".to_string())
        } else {
            outcome
        }
    }

    fn serve_one(
        &self,
        conn: OwnedFd,
        service: &std::sync::Arc<Self>,
        cancel: std::sync::Arc<AtomicBool>,
    ) {
        let conn_fd = conn.as_raw_fd();
        let result = self.serve_connection(conn_fd, service, cancel);
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

    fn serve_connection(
        &self,
        conn: RawFd,
        service: &std::sync::Arc<Self>,
        cancel: std::sync::Arc<AtomicBool>,
    ) -> LaunchExit {
        let peer = match muse_peer_from_fd(conn) {
            Ok(peer) => peer,
            Err(_) => return LaunchExit::denied(),
        };
        let received = match muse_request_from_fd(conn) {
            Ok(received) => received,
            Err(_) => return LaunchExit::denied(),
        };
        if cancel.load(Ordering::SeqCst) {
            return LaunchExit::denied();
        }
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
            service.clone(),
            cancel,
        )
    }

    /// `MuseLaunch.shell`: spawn, deliver, supervise, finish.
    fn shell(
        &self,
        conn: RawFd,
        peer: &MusePeer,
        request: &LaunchRequest,
        stdio: [std::fs::File; 3],
        supervisor_owner: std::sync::Arc<Self>,
        shutdown: std::sync::Arc<AtomicBool>,
    ) -> LaunchExit {
        use std::os::unix::io::AsRawFd;
        if shutdown.load(Ordering::SeqCst) {
            return LaunchExit::denied();
        }
        let session_end = Instant::now() + Duration::from_secs(12 * 3600);
        let mut execution = match self.runtime.prepare_execution(peer, request, session_end) {
            Ok(execution) => execution,
            Err(_) => return LaunchExit::denied(),
        };
        if shutdown.load(Ordering::SeqCst) {
            return self.denied_after_cleanup(&execution);
        }
        // Keep a close-on-exec duplicate of stdin for the supervisor while
        // the original stdio files move into the child command.
        let stdin_control = match stdio[0].try_clone() {
            Ok(file) => file,
            Err(_) => return self.denied_after_cleanup(&execution),
        };
        let mut child = match spawn_execution(&execution, stdio) {
            Ok(child) => child,
            Err(_) => return self.denied_after_cleanup(&execution),
        };
        if shutdown.load(Ordering::SeqCst) {
            let _ = child.kill();
            let _ = child.wait();
            return self.denied_after_cleanup(&execution);
        }
        if self
            .runtime
            .deliver_execution(&mut execution, session_end)
            .is_err()
        {
            let _ = child.kill();
            let _ = child.wait();
            return self.denied_after_cleanup(&execution);
        }
        if shutdown.load(Ordering::SeqCst) {
            let _ = child.kill();
            let _ = child.wait();
            return self.denied_after_cleanup(&execution);
        }
        // Open an identity-bound handle before delegating control. Numeric PIDs
        // are never used by a thread that can outlive Child::wait.
        let pidfd = match pidfd_open(child.id()) {
            Ok(fd) => fd,
            Err(_) => {
                let _ = child.kill();
                let _ = child.wait();
                return self.denied_after_cleanup(&execution);
            }
        };
        // `conn` remains owned by serve_one until this shell returns.
        let conn_dup = match unsafe { BorrowedFd::borrow_raw(conn) }.try_clone_to_owned() {
            Ok(fd) => fd,
            Err(_) => {
                let _ = child.kill();
                let _ = child.wait();
                return self.denied_after_cleanup(&execution);
            }
        };
        let control_cancel = std::sync::Arc::new(AtomicBool::new(false));
        let supervisor_cancel = control_cancel.clone();
        let snapshot = execution.clone();
        let supervisor = std::thread::Builder::new().spawn(move || {
            supervisor_owner.control_loop(
                conn_dup.as_raw_fd(),
                stdin_control.as_raw_fd(),
                pidfd.as_raw_fd(),
                &supervisor_cancel,
                &shutdown,
                &snapshot,
                session_end,
            );
            // The pidfd remains the same task identity after wait/reap; this
            // cannot signal a newly reused numeric PID.
            let _ = pidfd_send_signal(pidfd.as_raw_fd(), libc::SIGKILL);
            drop(conn_dup);
            drop(stdin_control);
            drop(pidfd);
        });
        let supervisor = match supervisor {
            Ok(supervisor) => supervisor,
            Err(_) => {
                let _ = child.kill();
                let _ = child.wait();
                // Captured descriptors are dropped when the failed spawn
                // closure is discarded; keep cleanup under this owner.
                return self.denied_after_cleanup(&execution);
            }
        };
        let mut result = muse_command_exit(child.wait());
        control_cancel.store(true, Ordering::SeqCst);
        if supervisor.join().is_err() {
            result = LaunchExit::cleanup_unconfirmed();
        }
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

    fn denied_after_cleanup(&self, execution: &MuseExecution) -> LaunchExit {
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
            LaunchExit::cleanup_unconfirmed()
        } else {
            LaunchExit::denied()
        }
    }

    /// Control supervisor (`museControls`): streaming strict control
    /// messages, 1MB total, unknown fields rejected. Any failure ends the
    /// loop; the spawner kills the session afterwards.
    pub fn control_loop(
        &self,
        conn: RawFd,
        stdin_fd: RawFd,
        child_pidfd: RawFd,
        control_cancel: &AtomicBool,
        shutdown: &AtomicBool,
        execution: &MuseExecution,
        session_end: Instant,
    ) {
        let mut buffer = Vec::new();
        let mut chunk = [0u8; 4096];
        loop {
            if Instant::now() >= session_end {
                return;
            }
            if control_cancel.load(Ordering::SeqCst) || shutdown.load(Ordering::SeqCst) {
                return;
            }
            let mut pfd = libc::pollfd {
                fd: conn,
                events: libc::POLLIN,
                revents: 0,
            };
            // SAFETY: one valid pollfd; the bounded wait also observes both
            // cancellation owners without detaching a helper thread.
            let ready = unsafe { libc::poll(&mut pfd, 1, CONTROL_POLL_MS) };
            if ready == 0 {
                continue;
            }
            if ready < 0 {
                if std::io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
                    continue;
                }
                return;
            }
            // SAFETY: recv into the scratch chunk after poll reports readiness.
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
                let control_deadline = session_end.min(Instant::now() + Duration::from_secs(30));
                if self
                    .runtime
                    .control_execution(execution, stdin_fd, &control, control_deadline)
                    .is_err()
                {
                    return;
                }
                if control.signal == 0 && execution.request.tty {
                    // Resize has already been applied through the owned stdin
                    // handle; signal only through the identity-bound pidfd.
                    if let Err(error) = pidfd_send_signal(child_pidfd, libc::SIGWINCH) {
                        if error.raw_os_error() != Some(libc::ESRCH) {
                            return;
                        }
                    }
                }
                buffer = trim[len..].to_vec();
            }
        }
    }
}

pub(super) fn pidfd_open(pid: u32) -> std::io::Result<OwnedFd> {
    // SAFETY: Linux pidfd_open takes a pid and flags; result ownership is
    // transferred to OwnedFd below.
    let fd = unsafe { libc::syscall(libc::SYS_pidfd_open, pid as libc::pid_t, 0) as RawFd };
    if fd < 0 {
        return Err(std::io::Error::last_os_error());
    }
    // SAFETY: syscall returned a fresh owned descriptor.
    Ok(unsafe { OwnedFd::from_raw_fd(fd) })
}

pub(super) fn pidfd_send_signal(pidfd: RawFd, signal: i32) -> std::io::Result<()> {
    // SAFETY: pidfd identifies the task; null siginfo and zero flags are the
    // ordinary pidfd_send_signal form.
    let result = unsafe {
        libc::syscall(
            libc::SYS_pidfd_send_signal,
            pidfd,
            signal,
            std::ptr::null::<libc::siginfo_t>(),
            0,
        ) as i32
    };
    if result < 0 {
        Err(std::io::Error::last_os_error())
    } else {
        Ok(())
    }
}
