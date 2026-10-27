use std::time::{Duration, Instant};

/// Typed failure of a bounded run: retained-stdout overflow stays
/// distinct from every other failure so route layers can signal an
/// explicit bounds refusal without matching error text. Anything else —
/// stderr ceiling, timeout/cancellation, cleanup, wait, exit status —
/// fails closed carrying the exact executor diagnostic.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum BoundedError {
    /// Retained stdout outgrew its ceiling with cleanup confirmed.
    StdoutLimit,
    /// Every other failure, carrying the exact executor diagnostic.
    Failed(String),
}

pub trait Executor {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String>;
    /// Run with stdout and stderr capture ceilings enforced before bytes
    /// are retained. Non-native test/custom executors may keep their
    /// existing behavior; the privileged Native implementations enforce
    /// both limits in their pipe readers.
    fn run_bounded(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        _stdout_limit: usize,
        _stderr_limit: usize,
    ) -> Result<Vec<u8>, String> {
        self.run(stdin, cmd, args, deadline)
    }
    /// Typed bounded run: stdout-ceiling overflow with confirmed cleanup
    /// surfaces as [`BoundedError::StdoutLimit`]; every other failure
    /// (stderr ceiling, timeout/cancellation, cleanup, wait, exit status)
    /// surfaces as [`BoundedError::Failed`] with the exact legacy
    /// diagnostic. Non-native test/custom executors keep their existing
    /// behavior through the default: any `run_bounded` error maps to
    /// `Failed`, never to `StdoutLimit`.
    fn run_bounded_typed(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        stdout_limit: usize,
        stderr_limit: usize,
    ) -> Result<Vec<u8>, BoundedError> {
        self.run_bounded(stdin, cmd, args, deadline, stdout_limit, stderr_limit)
            .map_err(BoundedError::Failed)
    }
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

#[derive(Clone, Copy, Default)]
struct CapturePolicy {
    grace: Option<Duration>,
    stdout_limit: Option<usize>,
    stderr_limit: Option<usize>,
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

    fn run_bounded(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        stdout_limit: usize,
        stderr_limit: usize,
    ) -> Result<Vec<u8>, String> {
        (*self).run_bounded(stdin, cmd, args, deadline, stdout_limit, stderr_limit)
    }

    fn run_bounded_typed(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        stdout_limit: usize,
        stderr_limit: usize,
    ) -> Result<Vec<u8>, BoundedError> {
        (*self).run_bounded_typed(stdin, cmd, args, deadline, stdout_limit, stderr_limit)
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
        execute(stdin, cmd, args, deadline, false, CapturePolicy::default())
    }

    fn run_bounded(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        stdout_limit: usize,
        stderr_limit: usize,
    ) -> Result<Vec<u8>, String> {
        execute(
            stdin,
            cmd,
            args,
            deadline,
            false,
            CapturePolicy {
                stdout_limit: Some(stdout_limit),
                stderr_limit: Some(stderr_limit),
                ..CapturePolicy::default()
            },
        )
    }

    fn run_bounded_typed(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        stdout_limit: usize,
        stderr_limit: usize,
    ) -> Result<Vec<u8>, BoundedError> {
        execute_typed(
            stdin,
            cmd,
            args,
            deadline,
            false,
            CapturePolicy {
                stdout_limit: Some(stdout_limit),
                stderr_limit: Some(stderr_limit),
                ..CapturePolicy::default()
            },
        )
    }

    fn run_with_capture_grace(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        grace: Duration,
    ) -> Result<Vec<u8>, String> {
        execute(
            stdin,
            cmd,
            args,
            deadline,
            false,
            CapturePolicy {
                grace: Some(grace),
                ..CapturePolicy::default()
            },
        )
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
        execute(stdin, cmd, args, deadline, true, CapturePolicy::default())
    }

    fn run_bounded(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        stdout_limit: usize,
        stderr_limit: usize,
    ) -> Result<Vec<u8>, String> {
        execute(
            stdin,
            cmd,
            args,
            deadline,
            true,
            CapturePolicy {
                stdout_limit: Some(stdout_limit),
                stderr_limit: Some(stderr_limit),
                ..CapturePolicy::default()
            },
        )
    }

    fn run_bounded_typed(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        stdout_limit: usize,
        stderr_limit: usize,
    ) -> Result<Vec<u8>, BoundedError> {
        execute_typed(
            stdin,
            cmd,
            args,
            deadline,
            true,
            CapturePolicy {
                stdout_limit: Some(stdout_limit),
                stderr_limit: Some(stderr_limit),
                ..CapturePolicy::default()
            },
        )
    }

    fn run_with_capture_grace(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        grace: Duration,
    ) -> Result<Vec<u8>, String> {
        execute(
            stdin,
            cmd,
            args,
            deadline,
            true,
            CapturePolicy {
                grace: Some(grace),
                ..CapturePolicy::default()
            },
        )
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
    capture: CapturePolicy,
) -> Result<Vec<u8>, String> {
    execute_with_waiter(stdin, cmd, args, deadline, status_only, capture, wait_exit)
        .map_err(|error| error.into_legacy(cmd, status_only))
}

fn execute_typed(
    stdin: &[u8],
    cmd: &str,
    args: &[&str],
    deadline: Instant,
    status_only: bool,
    capture: CapturePolicy,
) -> Result<Vec<u8>, BoundedError> {
    execute_with_waiter(stdin, cmd, args, deadline, status_only, capture, wait_exit).map_err(
        |error| match error {
            RunError::StdoutLimit {
                cleanup_confirmed: true,
            } => BoundedError::StdoutLimit,
            RunError::StdoutLimit {
                cleanup_confirmed: false,
            } => BoundedError::Failed(capture_limit_error(cmd, status_only, false)),
            RunError::Msg(message) => BoundedError::Failed(message),
        },
    )
}

/// Structured failure of one bounded execution. A stdout-ceiling
/// overflow with confirmed cleanup stays distinct from every other
/// failure so typed callers can signal it without matching error text.
enum RunError {
    /// Pure retained-stdout overflow: every transfer completed, the
    /// reaped exit was clean, and the flag records whether the bounded
    /// cleanup confirmed the child reaped. Anything mixed (incomplete
    /// transfer, stderr involvement, nonzero exit, lapsed deadline,
    /// unconfirmed cleanup) is `Msg`, never this.
    StdoutLimit { cleanup_confirmed: bool },
    /// Every other failure, carrying the exact legacy diagnostic.
    Msg(String),
}

impl RunError {
    fn into_legacy(self, cmd: &str, status_only: bool) -> String {
        match self {
            RunError::Msg(message) => message,
            RunError::StdoutLimit { cleanup_confirmed } => {
                capture_limit_error(cmd, status_only, cleanup_confirmed)
            }
        }
    }
}

/// Which retained pipe outgrew its ceiling while the run was in flight.
/// Stderr involvement always fails closed as `Msg`; only a pure stdout
/// overflow with confirmed cleanup is a typed bounds refusal.
#[derive(Clone, Copy, PartialEq, Eq)]
enum LimitCause {
    Stdout,
    Stderr,
}

/// Shared runner with an injectable exit probe. Exclusive-waiter design:
/// this loop is the only waiter (production probes never reap; reaps
/// happen only on the three exclusive paths below), the default
/// disposition reaps nothing early, and no waitpid(-1) stealing exists
/// anywhere. Test waiters must preserve these assumptions.
fn spawn_execution_child(
    cmd: &str,
    args: &[&str],
    status_only: bool,
) -> Result<std::process::Child, String> {
    use std::os::unix::process::CommandExt;
    use std::process::Stdio;
    std::process::Command::new(cmd)
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
        })
}

fn take_and_arm_pipes(
    child: &mut std::process::Child,
    cmd: &str,
) -> Result<
    (
        Option<std::process::ChildStdin>,
        Option<std::process::ChildStdout>,
        Option<std::process::ChildStderr>,
    ),
    String,
> {
    use std::os::unix::io::AsRawFd;
    let input = child.stdin.take();
    let out_pipe = child.stdout.take();
    let err_pipe = child.stderr.take();
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
            let _ = cleanup_child(child);
            return Err(format!("{cmd} failed: {e}"));
        }
    }
    Ok((input, out_pipe, err_pipe))
}

fn pump_stdin_opt(
    pipe: Option<std::process::ChildStdin>,
    stdin: &[u8],
    deadline: Instant,
    capture: Option<&std::sync::Mutex<Instant>>,
) -> Transfer<()> {
    match pipe {
        Some(writer) => pump_stdin(writer, stdin, deadline, capture),
        None => Transfer::Done(()),
    }
}

fn drain_stdout_opt(
    pipe: Option<std::process::ChildStdout>,
    deadline: Instant,
    capture: Option<&std::sync::Mutex<Instant>>,
    limit: Option<usize>,
    exceeded: &std::sync::atomic::AtomicBool,
) -> Transfer<Vec<u8>> {
    match pipe {
        Some(output) => drain_pipe(output, deadline, capture, limit, exceeded),
        None => Transfer::Done(Vec::new()),
    }
}

fn drain_stderr_opt(
    pipe: Option<std::process::ChildStderr>,
    deadline: Instant,
    capture: Option<&std::sync::Mutex<Instant>>,
    limit: Option<usize>,
    exceeded: &std::sync::atomic::AtomicBool,
) -> Transfer<Vec<u8>> {
    match pipe {
        Some(output) => drain_pipe(output, deadline, capture, limit, exceeded),
        None => Transfer::Done(Vec::new()),
    }
}

fn tighten_capture_grace(
    grace: Option<Duration>,
    bound: &std::sync::Mutex<Instant>,
    deadline: Instant,
) {
    if let Some(grace) = grace {
        if let Ok(mut cap) = bound.lock() {
            *cap = deadline.min(Instant::now() + grace);
        }
    }
}

fn timeout_execution_error(cmd: &str, status_only: bool, cleaned: Cleanup) -> String {
    if status_only {
        match cleaned {
            Cleanup::Clean(status) => exit_text(status),
            Cleanup::Uncertain => completion_text(cmd, status_only),
        }
    } else {
        format!("{cmd} failed: deadline exceeded")
    }
}

fn cleanup_unless_lost(child: &mut std::process::Child, cause: &std::io::Error) {
    if cause.raw_os_error() != Some(libc::ECHILD) {
        let _ = cleanup_child(child);
    }
}

fn limit_capture_error(
    pid: libc::pid_t,
    child: &mut std::process::Child,
    cmd: &str,
    status_only: bool,
    cause: LimitCause,
    retained_stderr: Vec<u8>,
) -> RunError {
    let retired = retire_group(pid);
    let status = match child.wait() {
        Ok(status) => status,
        Err(e) => return RunError::Msg(wait_text(cmd, status_only, &e)),
    };
    match cause {
        LimitCause::Stderr => RunError::Msg(capture_limit_error(cmd, status_only, retired)),
        LimitCause::Stdout => {
            // Fail-closed precedence: the reaped exit status decides, not
            // the overflow flag. A nonzero exit is an execution failure
            // (reported with the retained stderr, like the normal path),
            // never a bounds refusal. Only a clean exit with confirmed
            // cleanup stays a typed bounds refusal; an unconfirmed
            // cleanup fails closed with the limit diagnostic.
            if !status.success() {
                return match finalize_execution(
                    status,
                    Vec::new(),
                    retained_stderr,
                    cmd,
                    status_only,
                ) {
                    Err(message) => RunError::Msg(message),
                    Ok(_) => RunError::Msg(capture_limit_error(cmd, status_only, retired)),
                };
            }
            if retired {
                RunError::StdoutLimit {
                    cleanup_confirmed: true,
                }
            } else {
                RunError::Msg(capture_limit_error(cmd, status_only, false))
            }
        }
    }
}

fn finalize_execution(
    status: std::process::ExitStatus,
    stdout: Vec<u8>,
    stderr: Vec<u8>,
    cmd: &str,
    status_only: bool,
) -> Result<Vec<u8>, String> {
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

fn execute_with_waiter(
    stdin: &[u8],
    cmd: &str,
    args: &[&str],
    deadline: Instant,
    status_only: bool,
    policy: CapturePolicy,
    waiter: impl Fn(libc::pid_t) -> Result<bool, std::io::Error>,
) -> Result<Vec<u8>, RunError> {
    // Refuse before launch: an expired deadline must never start an
    // effectful command. The error describes the refusal, never a kill
    // (no child exists), like Go's expired-context Start refusal.
    if Instant::now() >= deadline {
        return Err(RunError::Msg(refusal_text(cmd, status_only)));
    }
    let mut child = spawn_execution_child(cmd, args, status_only).map_err(RunError::Msg)?;
    let (mut input, mut out_pipe, mut err_pipe) =
        take_and_arm_pipes(&mut child, cmd).map_err(RunError::Msg)?;
    let pid = child.id() as libc::pid_t;
    // Post-exit capture bound, shared with the transfers. Unselected
    // callers keep the full caller deadline; selected callers tighten it
    // to the capture grace once exit is observed.
    let capture_deadline = std::sync::Mutex::new(deadline);
    let cap = Some(&capture_deadline);
    // One overflow flag per retained pipe: only a pure stdout overflow
    // with confirmed cleanup is a typed bounds refusal. Stderr
    // involvement always fails closed as `Msg`.
    let stdout_exceeded = std::sync::atomic::AtomicBool::new(false);
    let stderr_exceeded = std::sync::atomic::AtomicBool::new(false);
    let outcome = std::thread::scope(|scope| {
        // Stdin/stdout/stderr transfer concurrently: a child emitting
        // beyond pipe capacity would otherwise block forever while the
        // poll loop below waits for exit (H01-F2). Every transfer honors
        // the capture bound, so every join below is bounded and no pump
        // is ever detached.
        let writer = scope.spawn(|| pump_stdin_opt(input.take(), stdin, deadline, cap));
        let out_drain = scope.spawn(|| {
            drain_stdout_opt(
                out_pipe.take(),
                deadline,
                cap,
                policy.stdout_limit,
                &stdout_exceeded,
            )
        });
        let err_drain = scope.spawn(|| {
            drain_stderr_opt(
                err_pipe.take(),
                deadline,
                cap,
                policy.stderr_limit,
                &stderr_exceeded,
            )
        });
        loop {
            if stdout_exceeded.load(std::sync::atomic::Ordering::Acquire)
                || stderr_exceeded.load(std::sync::atomic::Ordering::Acquire)
            {
                if let Ok(mut bound) = capture_deadline.lock() {
                    *bound = (*bound).min(Instant::now() + CLEANUP_GRACE);
                }
                let cleaned = cleanup_child(&mut child);
                let wrote = writer.join();
                let stdout = out_drain.join();
                let stderr = err_drain.join();
                // Fail-closed precedence: a transfer that stopped WITHOUT
                // its pipe's overflow flag is genuinely incomplete (the
                // caller deadline passed or the pipe broke), never bounds.
                // A flagged pipe's Incomplete IS the overflow, so only an
                // unflagged Incomplete fails the run here.
                if !matches!(wrote, Ok(Transfer::Done(())))
                    || (!stdout_exceeded.load(std::sync::atomic::Ordering::Acquire)
                        && !matches!(stdout, Ok(Transfer::Done(_))))
                    || (!stderr_exceeded.load(std::sync::atomic::Ordering::Acquire)
                        && !matches!(stderr, Ok(Transfer::Done(_))))
                {
                    return Err(RunError::Msg(completion_text(cmd, status_only)));
                }
                let cleanup_confirmed = matches!(cleaned, Cleanup::Clean(_));
                if stderr_exceeded.load(std::sync::atomic::Ordering::Acquire) {
                    return Err(RunError::Msg(capture_limit_error(
                        cmd,
                        status_only,
                        cleanup_confirmed,
                    )));
                }
                // Fail-closed precedence: a reaped nonzero EXIT is a
                // natural `exit N` that landed before the cleanup reap
                // (the flag causally precedes it: the overflow drain
                // drops the read end, the writer SIGPIPEs, the shell
                // exits N). Report the execution failure, never
                // bounds — exactly like the capture-phase path. Signal
                // deaths (including this branch's own group kill of a
                // still-live leader) keep the bounds refusal, so a pure
                // overflow never leaks its kill status.
                if let Cleanup::Clean(status) = &cleaned {
                    if matches!(status.code(), Some(code) if code != 0) {
                        let retained_stderr = match &stderr {
                            Ok(Transfer::Done(retained)) => retained.clone(),
                            _ => Vec::new(),
                        };
                        return Err(
                            match finalize_execution(
                                *status,
                                Vec::new(),
                                retained_stderr,
                                cmd,
                                status_only,
                            ) {
                                Err(message) => RunError::Msg(message),
                                Ok(_) => RunError::Msg(capture_limit_error(
                                    cmd,
                                    status_only,
                                    cleanup_confirmed,
                                )),
                            },
                        );
                    }
                }
                // A deadline that lapsed around the overflow is a
                // cancellation, never a bounds refusal.
                if Instant::now() >= deadline {
                    return Err(RunError::Msg(timeout_execution_error(
                        cmd,
                        status_only,
                        cleaned,
                    )));
                }
                return Err(RunError::StdoutLimit { cleanup_confirmed });
            }
            match waiter(pid) {
                // Exited but UN-REAPED: the zombie pins the PID/PGID
                // against reuse until capture and group retirement
                // settle below; the single reap follows the joins.
                Ok(true) => {
                    tighten_capture_grace(policy.grace, &capture_deadline, deadline);
                    break;
                }
                Ok(false) => {
                    if Instant::now() >= deadline {
                        // Timeout: bounded cleanup of the owned child
                        // and group, then join the already-due
                        // transfers (prompt, never detached) and report
                        // the timeout; the join results are discarded.
                        // A completed kill reports the wait status
                        // like Go's deadline kill; an uncertain one
                        // claims no kill at all.
                        let cleaned = cleanup_child(&mut child);
                        let _ = writer.join();
                        let _ = out_drain.join();
                        let _ = err_drain.join();
                        return Err(RunError::Msg(timeout_execution_error(
                            cmd,
                            status_only,
                            cleaned,
                        )));
                    }
                    std::thread::sleep(Duration::from_millis(5));
                }
                Err(e) => {
                    // Ownership lost (ECHILD): the numeric PID/PGID is unpinned
                    // and may already be reused — never signal it. Join the
                    // bounded transfers (slow but signal-free; a live child
                    // cannot be hurried), then report the wait failure as
                    // uncertainty, never as cleanup. Sole-owner unreaped
                    // custody intact runs the bounded cleanup first so nothing
                    // is abandoned; the cleanup outcome stays best-effort by
                    // design, since a killed status here would misdescribe
                    // the run.
                    cleanup_unless_lost(&mut child, &e);
                    let _ = writer.join();
                    let _ = out_drain.join();
                    let _ = err_drain.join();
                    return Err(RunError::Msg(wait_text(cmd, status_only, &e)));
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
            .map_err(|_| RunError::Msg(format!("{cmd} failed: stdin writer panicked")))?;
        let stdout = out_drain
            .join()
            .map_err(|_| RunError::Msg(completion_text(cmd, status_only)))?;
        let stderr = err_drain
            .join()
            .map_err(|_| RunError::Msg(completion_text(cmd, status_only)))?;
        // Fail-closed precedence: genuine incompleteness (an Incomplete
        // transfer with no overflow flag for its pipe) beats every
        // overflow flag, exactly like the exclusive-waiter path above.
        // The check runs before the flags because a flagged drain always
        // reports Incomplete itself.
        let wrote_incomplete = matches!(wrote, Transfer::Incomplete);
        let stdout_genuine = matches!(stdout, Transfer::Incomplete)
            && !stdout_exceeded.load(std::sync::atomic::Ordering::Acquire);
        let stderr_genuine = matches!(stderr, Transfer::Incomplete)
            && !stderr_exceeded.load(std::sync::atomic::Ordering::Acquire);
        if wrote_incomplete || stdout_genuine || stderr_genuine {
            // Incomplete with the leader still pinned: retire the
            // original owned group BEFORE the single reap (the zombie
            // holds the PGID against reuse), then report the bounded
            // capture failure. Never signal after the reap.
            let _ = retire_group(pid);
            let _ = child.wait();
            return Err(RunError::Msg(completion_text(cmd, status_only)));
        }
        if stderr_exceeded.load(std::sync::atomic::Ordering::Acquire) {
            return Err(limit_capture_error(
                pid,
                &mut child,
                cmd,
                status_only,
                LimitCause::Stderr,
                Vec::new(),
            ));
        }
        if stdout_exceeded.load(std::sync::atomic::Ordering::Acquire) {
            let retained_stderr = match &stderr {
                Transfer::Done(retained) => retained.clone(),
                Transfer::Incomplete => Vec::new(),
            };
            return Err(limit_capture_error(
                pid,
                &mut child,
                cmd,
                status_only,
                LimitCause::Stdout,
                retained_stderr,
            ));
        }
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
                return Err(RunError::Msg(completion_text(cmd, status_only)));
            }
        };
        // Single reap now that capture settled; the exit status decides.
        let status = child
            .wait()
            .map_err(|e| RunError::Msg(wait_text(cmd, status_only, &e)))?;
        Ok((status, stdout, stderr))
    })?;
    let (status, stdout, stderr): (std::process::ExitStatus, Vec<u8>, Vec<u8>) = outcome;
    finalize_execution(status, stdout, stderr, cmd, status_only).map_err(RunError::Msg)
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

fn capture_limit_error(cmd: &str, status_only: bool, cleanup_confirmed: bool) -> String {
    let reason = if cleanup_confirmed {
        "output capture limit exceeded"
    } else {
        "output capture limit exceeded; child cleanup unconfirmed"
    };
    if status_only {
        reason.to_string()
    } else {
        format!("{cmd} failed: {reason}")
    }
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
    limit: Option<usize>,
    exceeded: &std::sync::atomic::AtomicBool,
) -> Transfer<Vec<u8>> {
    let fd = pipe.as_raw_fd();
    let mut chunk = [0u8; 8192];
    let initial_capacity = limit.map(|limit| limit.min(chunk.len())).unwrap_or(0);
    let mut buf = Vec::with_capacity(initial_capacity);
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
            n if n > 0 => {
                let n = n as usize;
                if let Some(limit) = limit {
                    if buf.len().saturating_add(n) > limit {
                        if let Some(capture) = capture {
                            if let Ok(mut bound) = capture.lock() {
                                *bound = (*bound).min(Instant::now() + CLEANUP_GRACE);
                            }
                        }
                        exceeded.store(true, std::sync::atomic::Ordering::Release);
                        return Transfer::Incomplete;
                    }
                }
                buf.extend_from_slice(&chunk[..n]);
            }
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
