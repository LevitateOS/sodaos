use std::time::{Duration, Instant};

pub trait Executor {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String>;
    /// Run with a bounded post-exit capture grace (Go WaitDelay analog):
    /// after the leader exits, capture must settle within `grace`
    /// (clamped to `deadline`) or the run fails. Default: full-deadline
    /// capture, preserving existing behavior for unselected callers.
    fn run_with_capture_grace(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        _grace: Duration,
    ) -> Result<Vec<u8>, String> {
        self.run(stdin, cmd, args, deadline)
    }
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

    fn run_with_capture_grace(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        grace: Duration,
    ) -> Result<Vec<u8>, String> {
        (*self).run_with_capture_grace(stdin, cmd, args, deadline, grace)
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
        execute(stdin, cmd, args, deadline, false, None)
    }

    fn run_with_capture_grace(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        grace: Duration,
    ) -> Result<Vec<u8>, String> {
        execute(stdin, cmd, args, deadline, false, Some(grace))
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
        execute(stdin, cmd, args, deadline, true, None)
    }

    fn run_with_capture_grace(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        grace: Duration,
    ) -> Result<Vec<u8>, String> {
        execute(stdin, cmd, args, deadline, true, Some(grace))
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
    capture_grace: Option<Duration>,
) -> Result<Vec<u8>, String> {
    execute_with_waiter(
        stdin,
        cmd,
        args,
        deadline,
        status_only,
        capture_grace,
        wait_exit,
    )
}

/// Shared runner with an injectable exit probe. Exclusive-waiter design:
/// this loop is the only waiter (production probes never reap; reaps
/// happen only on the three exclusive paths below), the default
/// disposition reaps nothing early, and no waitpid(-1) stealing exists
/// anywhere. Test waiters must preserve these assumptions.
fn execute_with_waiter(
    stdin: &[u8],
    cmd: &str,
    args: &[&str],
    deadline: Instant,
    status_only: bool,
    capture_grace: Option<Duration>,
    waiter: impl Fn(libc::pid_t) -> Result<bool, std::io::Error>,
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
    let pid = child.id() as libc::pid_t;
    // Post-exit capture bound, shared with the transfers. Unselected
    // callers keep the full caller deadline; selected callers tighten it
    // to the capture grace once exit is observed.
    let capture = std::sync::Mutex::new(deadline);
    let cap = capture_grace.map(|_| &capture);
    let outcome = std::thread::scope(|scope| {
        // Stdin/stdout/stderr transfer concurrently: a child emitting
        // beyond pipe capacity would otherwise block forever while the
        // poll loop below waits for exit (H01-F2). Every transfer honors
        // the capture bound, so every join below is bounded and no pump
        // is ever detached.
        let writer = scope.spawn(|| match input.take() {
            Some(w) => pump_stdin(w, stdin, deadline, cap),
            None => Transfer::Done(()),
        });
        let out_drain = scope.spawn(|| match out_pipe.take() {
            Some(o) => drain_pipe(o, deadline, cap),
            None => Transfer::Done(Vec::new()),
        });
        let err_drain = scope.spawn(|| match err_pipe.take() {
            Some(e) => drain_pipe(e, deadline, cap),
            None => Transfer::Done(Vec::new()),
        });
        loop {
            match waiter(pid) {
                // Exited but UN-REAPED: the zombie pins the PID/PGID
                // against reuse until capture and group retirement
                // settle below; the single reap follows the joins.
                Ok(true) => {
                    if let Some(grace) = capture_grace {
                        if let Ok(mut bound) = capture.lock() {
                            *bound = deadline.min(Instant::now() + grace);
                        }
                    }
                    break;
                }
                Ok(false) => {
                    if Instant::now() >= deadline {
                        // Timeout: bounded cleanup of the owned child
                        // and group, then join the already-due
                        // transfers (prompt, never detached) and report
                        // the timeout; the join results are discarded.
                        let cleaned = cleanup_child(&mut child);
                        let _ = writer.join();
                        let _ = out_drain.join();
                        let _ = err_drain.join();
                        if status_only {
                            // A completed kill reports the wait status
                            // like Go's deadline kill; an uncertain one
                            // claims no kill at all.
                            return match cleaned {
                                Cleanup::Clean(status) => Err(exit_text(status)),
                                Cleanup::Uncertain => Err(completion_text(cmd, status_only)),
                            };
                        }
                        return Err(format!("{cmd} failed: deadline exceeded"));
                    }
                    std::thread::sleep(Duration::from_millis(5));
                }
                Err(e) => {
                    if e.raw_os_error() == Some(libc::ECHILD) {
                        // Ownership lost: the numeric PID/PGID is unpinned
                        // and may already be reused — never signal it.
                        // Join the bounded transfers (slow but signal-free;
                        // a live child cannot be hurried), then report the
                        // wait failure as uncertainty, never as cleanup.
                        let _ = writer.join();
                        let _ = out_drain.join();
                        let _ = err_drain.join();
                        return Err(wait_text(cmd, status_only, &e));
                    }
                    // Sole-owner unreaped custody intact (only this loop
                    // waits; production probes never reap): run the bounded
                    // cleanup first so nothing is abandoned, then report
                    // the wait failure. The cleanup outcome stays
                    // best-effort by design; a killed status here would
                    // misdescribe the run.
                    let _ = cleanup_child(&mut child);
                    let _ = writer.join();
                    let _ = out_drain.join();
                    let _ = err_drain.join();
                    return Err(wait_text(cmd, status_only, &e));
                }
            }
        }
        // Capture phase: the leader is a pinned zombie. Joins are bounded
        // by the caller deadline, so they always return. Limit: a panicked
        // pump returns early below, skipping group retirement and the
        // single reap (zombie and group survive to process exit); the
        // never-orphan claims cover the joined paths only, not unwinds.
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
        let (stdout, stderr) = match (wrote, stdout, stderr) {
            (Transfer::Done(()), Transfer::Done(out), Transfer::Done(err)) => (out, err),
            _ => {
                // Incomplete with the leader still pinned: retire the
                // original owned group BEFORE the single reap (the zombie
                // holds the PGID against reuse), then report the bounded
                // capture failure. Never signal after the reap.
                let _ = retire_group(pid);
                let _ = child.wait();
                return Err(completion_text(cmd, status_only));
            }
        };
        // Single reap now that capture settled; the exit status decides.
        let status = child.wait().map_err(|e| wait_text(cmd, status_only, &e))?;
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

/// Wait-failure text: reports the wait error itself, never a cleanup
/// claim and never a status the run did not observe.
fn wait_text(cmd: &str, status_only: bool, e: &std::io::Error) -> String {
    if status_only {
        format!("wait {cmd}: {e}")
    } else {
        format!("{cmd} failed: {e}")
    }
}

/// Bounded transfer outcome: Done carries the complete bytes (unit for
/// stdin); Incomplete means the caller deadline passed or a pipe error
/// struck with the transfer unfinished — never usable as success.
enum Transfer<T> {
    Done(T),
    Incomplete,
}

/// Bounded cleanup outcome. Clean carries the reaped status; Uncertain
/// preserves caller uncertainty (reap missed the grace, or group
/// retirement unverified) instead of claiming a completed kill. An
/// Uncertain outcome may leave a live, unreaped child; callers must
/// assume nothing about its fate.
enum Cleanup {
    Clean(std::process::ExitStatus),
    Uncertain,
}

/// Post-kill reap grace: SIGKILL lands in milliseconds, so one second
/// distinguishes a completed kill from an unkillable child without
/// letting cleanup run open-ended.
const CLEANUP_GRACE: Duration = Duration::from_secs(1);

/// Retire the owned group: true when the termination was requested of
/// a live group, or the group was already gone (ESRCH). A delivered
/// SIGKILL is a request, not proof every descendant vanished; false
/// preserves uncertainty.
fn retire_group(pgid: libc::pid_t) -> bool {
    if unsafe { libc::killpg(pgid, libc::SIGKILL) } == 0 {
        return true;
    }
    std::io::Error::last_os_error().raw_os_error() == Some(libc::ESRCH)
}

/// Bounded synchronous cleanup of an owned, un-reaped child and its group
/// (group id == child pid via process_group(0), unreusable until we reap,
/// so no foreign PID/PGID is signaled). Retires the group, kills the
/// leader, then reaps within CLEANUP_GRACE.
fn cleanup_child(child: &mut std::process::Child) -> Cleanup {
    let retired = retire_group(child.id() as libc::pid_t);
    let _ = child.kill();
    let grace = Instant::now() + CLEANUP_GRACE;
    loop {
        match child.try_wait() {
            Ok(Some(status)) => {
                return if retired {
                    Cleanup::Clean(status)
                } else {
                    Cleanup::Uncertain
                };
            }
            Ok(None) => {
                if Instant::now() >= grace {
                    return Cleanup::Uncertain;
                }
                std::thread::sleep(Duration::from_millis(5));
            }
            Err(_) => return Cleanup::Uncertain,
        }
    }
}

/// Non-reaping exit probe: true when the leader exited (it stays a pinned
/// zombie until the single reap), false while running. EINTR retries.
/// waitid is Linux's non-reaping wait; waitpid rejects WNOWAIT with EINVAL.
fn wait_exit(pid: libc::pid_t) -> Result<bool, std::io::Error> {
    loop {
        let mut info: libc::siginfo_t = unsafe { std::mem::zeroed() };
        let r = unsafe {
            libc::waitid(
                libc::P_PID,
                pid as libc::id_t,
                &mut info,
                libc::WEXITED | libc::WNOHANG | libc::WNOWAIT,
            )
        };
        if r == 0 {
            // WNOHANG with no state change leaves si_pid zeroed.
            return Ok(unsafe { info.si_pid() } != 0);
        }
        let e = std::io::Error::last_os_error();
        if e.kind() == std::io::ErrorKind::Interrupted {
            continue;
        }
        return Err(e);
    }
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
        match poll_once(fd, events, timeout) {
            Poll::Ready => return true,
            Poll::Timeout => return false,
            Poll::Failed => return false,
            // Interruption returns to this loop so the retry recomputes
            // the remaining budget from the absolute deadline instead of
            // restarting the original timeout.
            Poll::Interrupted => continue,
        }
    }
}

/// Single poll outcome.
enum Poll {
    Ready,
    Timeout,
    Failed,
    Interrupted,
}

/// One poll with a millisecond timeout. EINTR is reported, never
/// retried here: retrying with the original relative timeout would let
/// repeated caught signals extend the wait past the absolute bound.
fn poll_once(fd: std::os::unix::io::RawFd, events: libc::c_short, timeout_ms: libc::c_int) -> Poll {
    let mut pfd = libc::pollfd {
        fd,
        events,
        revents: 0,
    };
    match unsafe { libc::poll(&mut pfd, 1, timeout_ms) } {
        n if n > 0 => return Poll::Ready,
        0 => return Poll::Timeout,
        _ => {
            if std::io::Error::last_os_error().kind() == std::io::ErrorKind::Interrupted {
                return Poll::Interrupted;
            }
            return Poll::Failed;
        }
    }
}

/// Wait for readiness until the effective capture bound: the caller
/// deadline, tightened by the shared post-exit cap when selected. False
/// only when the bound passed (or the fd failed).
fn wait_ready(
    fd: std::os::unix::io::RawFd,
    events: libc::c_short,
    deadline: Instant,
    capture: Option<&std::sync::Mutex<Instant>>,
) -> bool {
    let Some(cap) = capture else {
        return poll_ready(fd, events, deadline);
    };
    loop {
        let bound = cap
            .lock()
            .ok()
            .map(|c| *c)
            .unwrap_or(deadline)
            .min(deadline);
        let remaining = bound.saturating_duration_since(Instant::now());
        if remaining.is_zero() {
            return false;
        }
        // Re-check the shared cap every 100 ms so post-exit tightening
        // lands promptly without busy-waiting.
        let chunk = remaining.as_millis().min(100) as libc::c_int;
        match poll_once(fd, events, chunk) {
            Poll::Ready => return true,
            // Timeout and interruption both re-read the shared cap and
            // recompute from the absolute bound; retries never reset it.
            Poll::Timeout | Poll::Interrupted => continue,
            Poll::Failed => return false,
        }
    }
}

/// Read one pipe to EOF or the caller deadline. Never blocks past the
/// deadline; partial bytes after an error are Incomplete, never Done.
fn drain_pipe(
    pipe: impl std::os::unix::io::AsRawFd,
    deadline: Instant,
    capture: Option<&std::sync::Mutex<Instant>>,
) -> Transfer<Vec<u8>> {
    let fd = pipe.as_raw_fd();
    let mut buf = Vec::new();
    let mut chunk = [0u8; 8192];
    loop {
        if !wait_ready(
            fd,
            libc::POLLIN | libc::POLLHUP | libc::POLLERR,
            deadline,
            capture,
        ) {
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
    capture: Option<&std::sync::Mutex<Instant>>,
) -> Transfer<()> {
    let fd = pipe.as_raw_fd();
    while !stdin.is_empty() {
        if !wait_ready(
            fd,
            libc::POLLOUT | libc::POLLHUP | libc::POLLERR,
            deadline,
            capture,
        ) {
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

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::io::RawFd;

    extern "C" fn sigusr1_noop(_: libc::c_int) {}

    /// Install the process-lifetime SIGUSR1 noop (no SA_RESTART, so polls
    /// surface EINTR). Idempotent; never restored: restoring to SIG_DFL
    /// while a sibling test's joined spammer runs would terminate the
    /// process, and no test sends SIGUSR1 except through joined spammers.
    fn arm_sigusr1() {
        unsafe {
            let mut sa: libc::sigaction = std::mem::zeroed();
            sa.sa_sigaction = sigusr1_noop as *const () as usize;
            sa.sa_flags = 0;
            libc::sigemptyset(&mut sa.sa_mask);
            assert_eq!(libc::sigaction(libc::SIGUSR1, &sa, std::ptr::null_mut()), 0);
        }
    }

    /// Finite self-signal spam: pthread_kill targets ONLY the calling
    /// thread (process-directed kill would starve behind the harness
    /// main thread), so sibling tests never observe a signal. Joined
    /// before the test ends; the thread is gone before any assertion on
    /// timing completes.
    fn spam_self(tid: libc::pthread_t, rounds: u32) -> std::thread::JoinHandle<()> {
        std::thread::spawn(move || {
            for _ in 0..rounds {
                unsafe {
                    libc::pthread_kill(tid, libc::SIGUSR1);
                }
                std::thread::sleep(Duration::from_millis(1));
            }
        })
    }

    fn never_ready_pipe() -> (RawFd, RawFd) {
        let mut fds = [0; 2];
        assert_eq!(unsafe { libc::pipe(fds.as_mut_ptr()) }, 0);
        (fds[0], fds[1])
    }

    fn close_fd(fd: RawFd) {
        unsafe {
            libc::close(fd);
        }
    }

    #[test]
    fn poll_once_reports_interruption() {
        arm_sigusr1();
        let (read, write) = never_ready_pipe();
        let spam = spam_self(unsafe { libc::pthread_self() }, 1000);
        // A 5s poll interrupted within milliseconds reports Interrupted;
        // restarting the original timeout would sleep through the spam.
        let start = Instant::now();
        let out = poll_once(read, libc::POLLIN, 5000);
        let dt = start.elapsed();
        spam.join().unwrap();
        close_fd(read);
        close_fd(write);
        assert!(
            dt < Duration::from_millis(1000),
            "EINTR restarted the full timeout: {dt:?}"
        );
        match out {
            Poll::Timeout => panic!("EINTR was retried with the original timeout"),
            _ => {}
        }
    }

    #[test]
    fn wait_ready_recomputes_budget_on_interruption() {
        arm_sigusr1();
        let (read, write) = never_ready_pipe();
        let spam = spam_self(unsafe { libc::pthread_self() }, 1000);
        // A 200ms bound fails at ~200ms despite constant interruption;
        // restarting timeouts would push the failure out with the spam.
        let deadline = Instant::now() + Duration::from_millis(200);
        let start = Instant::now();
        assert!(!wait_ready(read, libc::POLLIN, deadline, None));
        let dt = start.elapsed();
        spam.join().unwrap();
        close_fd(read);
        close_fd(write);
        assert!(
            dt < Duration::from_millis(500),
            "retries reset the budget: {dt:?}"
        );
    }

    struct Fixture {
        dir: std::path::PathBuf,
    }

    static FIXTURE_SEQ: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);

    impl Fixture {
        fn fresh() -> Fixture {
            let id = FIXTURE_SEQ.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
            let dir = std::env::temp_dir().join(format!("executor-{}-{id}", std::process::id()));
            let _ = std::fs::remove_dir_all(&dir);
            std::fs::create_dir_all(&dir).unwrap();
            Fixture { dir }
        }

        fn script(&self, name: &str, body: &str) -> String {
            use std::os::unix::fs::PermissionsExt;
            let path = self.dir.join(name);
            std::fs::write(&path, format!("#!/bin/sh\n{body}\n")).unwrap();
            std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o755)).unwrap();
            path.to_string_lossy().into_owned()
        }

        fn path(&self, name: &str) -> String {
            self.dir.join(name).to_string_lossy().into_owned()
        }
    }

    impl Drop for Fixture {
        fn drop(&mut self) {
            let _ = std::fs::remove_dir_all(&self.dir);
        }
    }

    #[test]
    fn echild_never_signals_unpinned_group() {
        // Simulated ownership loss with a real owned child: the waiter
        // reports ECHILD while the leader and a heartbeat grandchild
        // live on. Nothing may be signaled (the numeric PGID is
        // unpinned); the beats prove survival past our bounded failure.
        let fixture = Fixture::fresh();
        let beat = fixture.path("beat");
        let script = fixture.script(
            "live.sh",
            &format!(
                "(for i in $(seq 1 30); do echo x >> \"{beat}\"; sleep 0.05; done) &\nsleep 3"
            ),
        );
        for (status_only, want) in [
            (
                false,
                format!("{script} failed: No child processes (os error 10)"),
            ),
            (
                true,
                format!("wait {script}: No child processes (os error 10)"),
            ),
        ] {
            let deadline = Instant::now() + Duration::from_secs(1);
            let start = Instant::now();
            let err = execute_with_waiter(&[], &script, &[], deadline, status_only, None, |_| {
                Err(std::io::Error::from_raw_os_error(libc::ECHILD))
            })
            .unwrap_err();
            assert_eq!(err, want);
            assert!(start.elapsed() >= Duration::from_millis(500), "too fast");
            assert!(start.elapsed() < Duration::from_secs(10), "wedged");
            let at_return = std::fs::read(&beat).unwrap().len();
            std::thread::sleep(Duration::from_millis(1500));
            assert!(
                std::fs::read(&beat).unwrap().len() > at_return,
                "group was signaled despite ECHILD"
            );
        }
        // Finite fixture: the loops exit alone; nothing leaks.
        std::thread::sleep(Duration::from_millis(1000));
        let end = std::fs::read(&beat).unwrap().len();
        std::thread::sleep(Duration::from_millis(500));
        assert_eq!(
            std::fs::read(&beat).unwrap().len(),
            end,
            "fixture never settled"
        );
    }

    #[test]
    fn intact_custody_wait_error_cleans_up() {
        // Simulated non-ECHILD wait failure with sole-owner custody
        // intact: the owned child and group are cleaned up promptly
        // (not abandoned), and the wait failure is reported.
        let fixture = Fixture::fresh();
        let beat = fixture.path("beat");
        let script = fixture.script(
            "live.sh",
            &format!(
                "(for i in $(seq 1 30); do echo x >> \"{beat}\"; sleep 0.05; done) &\nsleep 3"
            ),
        );
        for (status_only, want) in [
            (
                false,
                format!("{script} failed: Invalid argument (os error 22)"),
            ),
            (
                true,
                format!("wait {script}: Invalid argument (os error 22)"),
            ),
        ] {
            let deadline = Instant::now() + Duration::from_secs(1);
            let start = Instant::now();
            let err = execute_with_waiter(&[], &script, &[], deadline, status_only, None, |_| {
                Err(std::io::Error::from_raw_os_error(libc::EINVAL))
            })
            .unwrap_err();
            assert_eq!(err, want);
            // Cleanup-first fails fast; abandoning would wait the deadline.
            assert!(
                start.elapsed() < Duration::from_millis(500),
                "not cleaned up"
            );
            // The group died with the cleanup: beats freeze at any count
            // (a live loop would write within the window).
            std::thread::sleep(Duration::from_millis(300));
            let frozen = std::fs::read(&beat).unwrap_or_default().len();
            std::thread::sleep(Duration::from_millis(500));
            assert_eq!(
                std::fs::read(&beat).unwrap_or_default().len(),
                frozen,
                "group survived intact-custody cleanup"
            );
        }
    }
}
