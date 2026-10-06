use std::time::{Duration, Instant};

pub trait Executor {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String>;
    /// Mirrors Go's `HostNative()` marker assertion: the privileged host
    /// executor whose native protocols must never leak stderr text.
    fn is_host_native(&self) -> bool {
        false
    }
}

impl<E: Executor> Executor for &E {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        (*self).run(stdin, cmd, args, deadline)
    }

    fn is_host_native(&self) -> bool {
        (*self).is_host_native()
    }
}

/// Native process execution with deadline kill, mirroring
/// `exec.CommandContext` + `Output` (piped stdin, combined stderr in the
/// failure text).
pub struct Native;

impl Executor for Native {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        execute(stdin, cmd, args, deadline, false)
    }

    fn is_host_native(&self) -> bool {
        true
    }
}

/// Native process execution reporting status-only failures (Go `Run`
/// shape: `exit status N` / `signal: name`), for CLI diagnostics that
/// must not emit subprocess stderr. Shares Native's launch, concurrent
/// drain, and deadline kill/group-cleanup/wait lifecycle; only the error
/// format differs.
pub struct NativeStatusOnly;

impl Executor for NativeStatusOnly {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        execute(stdin, cmd, args, deadline, true)
    }

    fn is_host_native(&self) -> bool {
        true
    }
}

fn execute(
    stdin: &[u8],
    cmd: &str,
    args: &[&str],
    deadline: Instant,
    status_only: bool,
) -> Result<Vec<u8>, String> {
    use std::os::unix::io::AsRawFd;
    use std::os::unix::process::CommandExt;
    use std::process::Stdio;
    // Refuse before launch: an expired deadline must never start an
    // effectful command. The error describes the refusal, never a kill
    // (no child exists), like Go's expired-context Start refusal.
    if Instant::now() >= deadline {
        return Err(refusal_text(cmd, status_only));
    }
    let mut child = std::process::Command::new(cmd)
        .args(args)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        // Own process group (group id == child pid) so timeout cleanup
        // reaches descendants holding the pipes. Signaled only while the
        // child is un-reaped, when the id cannot be reused.
        .process_group(0)
        .spawn()
        .map_err(|e| {
            if status_only {
                format!("fork/exec {cmd}: {e}")
            } else {
                format!("{cmd} failed: {e}")
            }
        })?;
    let mut input = child.stdin.take();
    let mut out_pipe = child.stdout.take();
    let mut err_pipe = child.stderr.take();
    // Deadline-bounded transfer needs non-blocking pipes. A pipe that
    // cannot be armed fails the run with the owned child cleaned up,
    // never orphaned and never detached.
    for fd in [
        input.as_ref().map(AsRawFd::as_raw_fd),
        out_pipe.as_ref().map(AsRawFd::as_raw_fd),
        err_pipe.as_ref().map(AsRawFd::as_raw_fd),
    ]
    .into_iter()
    .flatten()
    {
        if let Err(e) = set_nonblocking(fd) {
            let _ = cleanup_child(&mut child);
            return Err(format!("{cmd} failed: {e}"));
        }
    }
    let outcome = std::thread::scope(|scope| {
        // Stdin/stdout/stderr transfer concurrently: a child emitting
        // beyond pipe capacity would otherwise block forever while the
        // poll loop below waits for exit (H01-F2). Every transfer honors
        // the same caller deadline, so every join below is bounded and no
        // pump is ever detached.
        let writer = scope.spawn(|| match input.take() {
            Some(w) => pump_stdin(w, stdin, deadline),
            None => Transfer::Done(()),
        });
        let out_drain = scope.spawn(|| match out_pipe.take() {
            Some(o) => drain_pipe(o, deadline),
            None => Transfer::Done(Vec::new()),
        });
        let err_drain = scope.spawn(|| match err_pipe.take() {
            Some(e) => drain_pipe(e, deadline),
            None => Transfer::Done(Vec::new()),
        });
        let status = loop {
            match child.try_wait() {
                Ok(Some(status)) => break status,
                Ok(None) => {
                    if Instant::now() >= deadline {
                        // Timeout: stop the owned child and its group,
                        // reap it, then join the already-due transfers
                        // (prompt, never detached) and report the
                        // timeout; the join results are discarded.
                        let waited = cleanup_child(&mut child);
                        let _ = writer.join();
                        let _ = out_drain.join();
                        let _ = err_drain.join();
                        if status_only {
                            // A real kill happened here, so the wait
                            // status is reported like Go's deadline kill.
                            return match waited {
                                Ok(status) => Err(exit_text(status)),
                                Err(e) => Err(format!("wait {cmd}: {e}")),
                            };
                        }
                        return Err(format!("{cmd} failed: deadline exceeded"));
                    }
                    std::thread::sleep(Duration::from_millis(5));
                }
                Err(e) => {
                    // try_wait failure leaves no reaped child to track;
                    // join the bounded transfers and fail without
                    // signaling anything (never orphan, never detach).
                    let _ = writer.join();
                    let _ = out_drain.join();
                    let _ = err_drain.join();
                    return Err(format!("{cmd} failed: {e}"));
                }
            }
        };
        // The child is reaped: ownership ends here. No further signals —
        // the PID/PGID may be reused from this point on.
        let wrote = writer
            .join()
            .map_err(|_| format!("{cmd} failed: stdin writer panicked"))?;
        let stdout = out_drain
            .join()
            .map_err(|_| completion_text(cmd, status_only))?;
        let stderr = err_drain
            .join()
            .map_err(|_| completion_text(cmd, status_only))?;
        // No successful truncated output: any transfer that missed the
        // deadline fails the run even when the exit status is zero, like
        // Go's WaitDelay expiry (ErrWaitDelay instead of nil).
        if !matches!(wrote, Transfer::Done(())) {
            return Err(completion_text(cmd, status_only));
        }
        let stdout = match stdout {
            Transfer::Done(buf) => buf,
            Transfer::Incomplete => return Err(completion_text(cmd, status_only)),
        };
        let stderr = match stderr {
            Transfer::Done(buf) => buf,
            Transfer::Incomplete => return Err(completion_text(cmd, status_only)),
        };
        Ok((status, stdout, stderr))
    })?;
    let (status, stdout, stderr): (std::process::ExitStatus, Vec<u8>, Vec<u8>) = outcome;
    if status.success() {
        return Ok(stdout);
    }
    if status_only {
        return Err(exit_text(status));
    }
    Err(format!(
        "{cmd} failed: {}: {}",
        exit_text(status),
        String::from_utf8_lossy(&stderr)
    ))
}

/// Expired-deadline refusal text: no child was started, so the wording
/// reports the missed deadline and never claims a killed child.
fn refusal_text(cmd: &str, status_only: bool) -> String {
    if status_only {
        String::from("deadline exceeded")
    } else {
        format!("{cmd} failed: deadline exceeded")
    }
}

/// Incomplete-capture text: the caller deadline passed before the pipes
/// finished. No kill is claimed; the child may have exited cleanly while
/// descendants held the pipes open.
fn completion_text(cmd: &str, status_only: bool) -> String {
    refusal_text(cmd, status_only)
}

/// Bounded transfer outcome: Done carries the complete bytes (unit for
/// stdin); Incomplete means the caller deadline passed or a pipe error
/// struck with the transfer unfinished — never usable as success.
enum Transfer<T> {
    Done(T),
    Incomplete,
}

/// Best-effort synchronous cleanup of an owned, un-reaped child and its
/// process group. Call only while holding the un-reaped Child: the group
/// id (== child pid via process_group(0)) cannot be reused until we reap,
/// so no foreign PID/PGID is ever signaled. Returns the reap result.
fn cleanup_child(child: &mut std::process::Child) -> std::io::Result<std::process::ExitStatus> {
    let pgid = child.id() as libc::pid_t;
    // Descendants first: SIGKILL the owned group so pipes held past the
    // direct child's death release promptly. Errors ignored: an empty or
    // already-dead group is a fine outcome.
    unsafe {
        libc::killpg(pgid, libc::SIGKILL);
    }
    let _ = child.kill();
    child.wait()
}

fn set_nonblocking(fd: std::os::unix::io::RawFd) -> std::io::Result<()> {
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
    if flags < 0 {
        return Err(std::io::Error::last_os_error());
    }
    if unsafe { libc::fcntl(fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(())
}

/// Poll one fd for readiness until the caller deadline. True when the fd
/// is ready; false when the deadline passed (or poll itself failed).
fn poll_ready(fd: std::os::unix::io::RawFd, events: libc::c_short, deadline: Instant) -> bool {
    loop {
        let remaining = deadline.saturating_duration_since(Instant::now());
        if remaining.is_zero() {
            return false;
        }
        let timeout = remaining.as_millis().min(i32::MAX as u128) as libc::c_int;
        let mut pfd = libc::pollfd {
            fd,
            events,
            revents: 0,
        };
        match unsafe { libc::poll(&mut pfd, 1, timeout) } {
            0 => return false,
            n if n > 0 => return true,
            _ => {
                if std::io::Error::last_os_error().kind() == std::io::ErrorKind::Interrupted {
                    continue;
                }
                return false;
            }
        }
    }
}

/// Read one pipe to EOF or the caller deadline. Never blocks past the
/// deadline; partial bytes after an error are Incomplete, never Done.
fn drain_pipe(pipe: impl std::os::unix::io::AsRawFd, deadline: Instant) -> Transfer<Vec<u8>> {
    let fd = pipe.as_raw_fd();
    let mut buf = Vec::new();
    let mut chunk = [0u8; 8192];
    loop {
        if !poll_ready(fd, libc::POLLIN | libc::POLLHUP | libc::POLLERR, deadline) {
            return Transfer::Incomplete;
        }
        match unsafe { libc::read(fd, chunk.as_mut_ptr() as *mut libc::c_void, chunk.len()) } {
            0 => return Transfer::Done(buf),
            n if n > 0 => buf.extend_from_slice(&chunk[..n as usize]),
            _ => match std::io::Error::last_os_error().kind() {
                std::io::ErrorKind::Interrupted => continue,
                std::io::ErrorKind::WouldBlock => continue,
                _ => return Transfer::Incomplete,
            },
        }
    }
}

/// Write stdin fully or to the caller deadline; dropping the pipe then
/// lets the child observe EOF. A dead reader ends the transfer as Done,
/// like the previous write_all write: the child declined the input.
/// Missing the deadline (a descendant holding the read end) is Incomplete.
fn pump_stdin(
    pipe: impl std::os::unix::io::AsRawFd,
    mut stdin: &[u8],
    deadline: Instant,
) -> Transfer<()> {
    let fd = pipe.as_raw_fd();
    while !stdin.is_empty() {
        if !poll_ready(fd, libc::POLLOUT | libc::POLLHUP | libc::POLLERR, deadline) {
            return Transfer::Incomplete;
        }
        match unsafe { libc::write(fd, stdin.as_ptr() as *const libc::c_void, stdin.len()) } {
            n if n > 0 => stdin = &stdin[(n as usize).min(stdin.len())..],
            0 => continue,
            _ => match std::io::Error::last_os_error().kind() {
                std::io::ErrorKind::Interrupted => continue,
                std::io::ErrorKind::WouldBlock => continue,
                _ => return Transfer::Done(()),
            },
        }
    }
    Transfer::Done(())
}

#[cfg(unix)]
fn exit_text(status: std::process::ExitStatus) -> String {
    use std::os::unix::process::ExitStatusExt;
    match status.code() {
        Some(code) => format!("exit status {code}"),
        None => match status.signal() {
            Some(1) => "signal: hangup".to_string(),
            Some(2) => "signal: interrupt".to_string(),
            Some(3) => "signal: quit".to_string(),
            Some(6) => "signal: aborted".to_string(),
            Some(9) => "signal: killed".to_string(),
            Some(15) => "signal: terminated".to_string(),
            Some(n) => format!("signal: {n}"),
            None => "signal: unknown".to_string(),
        },
    }
}

#[cfg(not(unix))]
fn exit_text(status: std::process::ExitStatus) -> String {
    match status.code() {
        Some(code) => format!("exit status {code}"),
        None => "signal: unknown".to_string(),
    }
}
