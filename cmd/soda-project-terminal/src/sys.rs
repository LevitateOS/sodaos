//! Process, time, fd, and lock primitives for the project terminal.
//!
//! Conventions: typed rustix file operations and scoped libc calls; opened
//! fds are owned [`std::fs::File`]s with `O_CLOEXEC` set at open time.
//!
//! Name rule for the dir-fd APIs ([`open_child_dir`], [`open_at`]): `name`
//! must be a single final component — empty, `"."`, `".."`, names containing
//! `'/'`, and names with NUL bytes are rejected with `InvalidInput`. Callers
//! walk multi-component paths one component at a time.

use std::io::Read;
use std::os::unix::io::AsRawFd;
use std::process::{Command, Output, Stdio};
use std::sync::OnceLock;
use std::time::Instant;

pub use rustix::fs::OFlags;

/// Process-local monotonic elapsed seconds, used for relative deadlines.
pub fn monotonic() -> f64 {
    static ORIGIN: OnceLock<Instant> = OnceLock::new();
    ORIGIN.get_or_init(Instant::now).elapsed().as_secs_f64()
}

/// System wall clock as whole seconds since the Unix epoch.
pub fn now_secs() -> i64 {
    match std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH) {
        Ok(d) => d.as_secs() as i64,
        Err(e) => -(e.duration().as_secs() as i64),
    }
}

fn spawn_argv(argv: &[String], piped: bool) -> std::io::Result<std::process::Child> {
    let (head, rest) = argv
        .split_first()
        .ok_or_else(|| std::io::Error::new(std::io::ErrorKind::InvalidInput, "empty argv"))?;
    let (out, err) = if piped {
        (Stdio::piped(), Stdio::piped())
    } else {
        (Stdio::null(), Stdio::null())
    };
    Command::new(head)
        .args(rest)
        .stdin(Stdio::null())
        .stdout(out)
        .stderr(err)
        .spawn()
}

/// Poll-wait for `child` until `timeout_secs` elapse (measured on
/// [`monotonic`]); on timeout the child is killed, reaped, and a `TimedOut`
/// error is returned.
fn wait_timeout(
    child: &mut std::process::Child,
    timeout_secs: u64,
) -> std::io::Result<std::process::ExitStatus> {
    let deadline = Instant::now() + std::time::Duration::from_secs(timeout_secs);
    loop {
        if Instant::now() >= deadline {
            let _ = child.kill();
            let _ = child.wait();
            return Err(std::io::Error::new(
                std::io::ErrorKind::TimedOut,
                "command timed out",
            ));
        }
        match child.try_wait() {
            Ok(Some(status)) => return Ok(status),
            Ok(None) => {}
            Err(error) => {
                let _ = child.kill();
                let _ = child.wait();
                return Err(error);
            }
        }
        std::thread::sleep(std::time::Duration::from_millis(5));
    }
}

/// Run `argv` with stdin null and stdout/stderr discarded; `Ok(())` only when
/// the child exits with status zero before `timeout_secs` elapse. Non-zero
/// exit is `Err`; on timeout the child is killed and `Err(TimedOut)` returned.
pub fn run_checked(argv: &[String], timeout_secs: u64) -> std::io::Result<()> {
    let mut child = spawn_argv(argv, false)?;
    let status = wait_timeout(&mut child, timeout_secs)?;
    if status.success() {
        Ok(())
    } else {
        Err(std::io::Error::other(format!("command failed: {status}")))
    }
}

const OUTPUT_LIMIT: usize = 8 << 20;
const DRAIN_GRACE: std::time::Duration = std::time::Duration::from_secs(2);

/// Kill `child`, reap it, and hand `error` back for return.
fn abort_child(child: &mut std::process::Child, error: std::io::Error) -> std::io::Error {
    let _ = child.kill();
    let _ = child.wait();
    error
}

/// Enforce the run deadline, then record a fresh child exit.
fn advance_status(
    child: &mut std::process::Child,
    status: &mut Option<std::process::ExitStatus>,
    drain_deadline: &mut Option<Instant>,
    now: Instant,
    deadline: Instant,
) -> std::io::Result<()> {
    if status.is_none() && now >= deadline {
        return Err(abort_child(
            child,
            std::io::Error::new(std::io::ErrorKind::TimedOut, "command timed out"),
        ));
    }
    if status.is_none() {
        match child.try_wait() {
            Ok(Some(done)) => {
                *status = Some(done);
                *drain_deadline = Some(now + DRAIN_GRACE);
            }
            Ok(None) => {}
            Err(error) => return Err(abort_child(child, error)),
        }
    }
    Ok(())
}

/// Read one nonblocking pipe pass; returns `(progressed, closed)`.
fn drain_pipe<R: Read>(
    pipe: &mut Option<R>,
    buf: &mut [u8],
    acc: &mut Vec<u8>,
    over: &'static str,
    child: &mut std::process::Child,
) -> std::io::Result<(bool, bool)> {
    let Some(pipe) = pipe else {
        return Ok((false, false));
    };
    match pipe.read(buf) {
        Ok(0) => Ok((true, true)),
        Ok(n) => {
            if acc.len().saturating_add(n) > OUTPUT_LIMIT {
                return Err(abort_child(
                    child,
                    std::io::Error::new(std::io::ErrorKind::InvalidData, over),
                ));
            }
            acc.extend_from_slice(&buf[..n]);
            Ok((true, false))
        }
        Err(error) if error.kind() == std::io::ErrorKind::WouldBlock => Ok((false, false)),
        Err(error) if error.kind() == std::io::ErrorKind::Interrupted => Ok((false, false)),
        Err(error) => Err(abort_child(child, error)),
    }
}

/// Build the finished output once both pipes closed, or fail an overrun drain.
fn take_finished(
    status: Option<std::process::ExitStatus>,
    stdout: &mut Option<std::process::ChildStdout>,
    stderr: &mut Option<std::process::ChildStderr>,
    drain_deadline: Option<Instant>,
    out: &mut Vec<u8>,
    err: &mut Vec<u8>,
) -> Option<std::io::Result<Output>> {
    let done = status?;
    if stdout.is_none() && stderr.is_none() {
        return Some(Ok(Output {
            status: done,
            stdout: std::mem::take(out),
            stderr: std::mem::take(err),
        }));
    }
    if drain_deadline.is_some_and(|at| Instant::now() >= at) {
        stdout.take();
        stderr.take();
        return Some(Err(std::io::Error::new(
            std::io::ErrorKind::TimedOut,
            "command output pipes did not close",
        )));
    }
    None
}

/// Run `argv` with stdin null and stdout/stderr piped; capture each stream up
/// to 8 MiB. Timeout, read failure, overflow, or pipes retained past the
/// post-exit grace are errors. The caller checks `Output.status`.
pub fn run_output(argv: &[String], timeout_secs: u64) -> std::io::Result<Output> {
    let mut child = spawn_argv(argv, true)?;
    let mut stdout = child.stdout.take();
    let mut stderr = child.stderr.take();
    if let Err(error) =
        set_pipe_nonblocking(stdout.as_ref()).and_then(|()| set_pipe_nonblocking(stderr.as_ref()))
    {
        return Err(abort_child(&mut child, error));
    }
    let deadline = Instant::now() + std::time::Duration::from_secs(timeout_secs);
    let mut status = None;
    let mut drain_deadline = None;
    let mut out = Vec::new();
    let mut err = Vec::new();
    let mut out_buf = [0u8; 65536];
    let mut err_buf = [0u8; 65536];
    while status.is_none() || stdout.is_some() || stderr.is_some() {
        let now = Instant::now();
        advance_status(&mut child, &mut status, &mut drain_deadline, now, deadline)?;
        let mut progressed = false;
        let (step, closed) = drain_pipe(
            &mut stdout,
            &mut out_buf,
            &mut out,
            "command stdout exceeded capture bound",
            &mut child,
        )?;
        progressed |= step;
        if closed {
            stdout.take();
        }
        let (step, closed) = drain_pipe(
            &mut stderr,
            &mut err_buf,
            &mut err,
            "command stderr exceeded capture bound",
            &mut child,
        )?;
        progressed |= step;
        if closed {
            stderr.take();
        }
        if let Some(result) = take_finished(
            status,
            &mut stdout,
            &mut stderr,
            drain_deadline,
            &mut out,
            &mut err,
        ) {
            return result;
        }
        if !progressed {
            std::thread::sleep(std::time::Duration::from_millis(5));
        }
    }
    Err(std::io::Error::other("command status unavailable"))
}

fn set_pipe_nonblocking<T: AsRawFd>(pipe: Option<&T>) -> std::io::Result<()> {
    let Some(pipe) = pipe else { return Ok(()) };
    let fd = pipe.as_raw_fd();
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(())
}

/// Single-component rule shared by the dir-fd opens.
fn check_component(name: &str) -> std::io::Result<()> {
    if name.is_empty() || name == "." || name == ".." || name.contains('/') || name.contains('\0') {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "invalid name component",
        ));
    }
    Ok(())
}

/// `openat(dir, name, O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC)`.
pub fn open_child_dir(dir: &std::fs::File, name: &str) -> std::io::Result<std::fs::File> {
    open_at(
        dir,
        name,
        OFlags::RDONLY | OFlags::DIRECTORY,
        rustix::fs::Mode::empty(),
    )
}

/// `open("/", O_RDONLY|O_DIRECTORY|O_CLOEXEC)`.
pub fn open_root() -> std::io::Result<std::fs::File> {
    rustix::fs::open(
        "/",
        rustix::fs::OFlags::RDONLY | rustix::fs::OFlags::DIRECTORY | rustix::fs::OFlags::CLOEXEC,
        rustix::fs::Mode::empty(),
    )
    .map(std::fs::File::from)
    .map_err(Into::into)
}

/// Open exactly one caller-admitted component below the held directory.
pub fn open_at(
    dir: &std::fs::File,
    name: &str,
    flags: OFlags,
    mode: rustix::fs::Mode,
) -> std::io::Result<std::fs::File> {
    check_component(name)?;
    rustix::fs::openat(dir, name, flags | OFlags::CLOEXEC | OFlags::NOFOLLOW, mode)
        .map(std::fs::File::from)
        .map_err(Into::into)
}

pub fn flock_exclusive_nb(file: &std::fs::File) -> std::io::Result<bool> {
    match rustix::fs::flock(file, rustix::fs::FlockOperation::NonBlockingLockExclusive) {
        Ok(()) => Ok(true),
        Err(rustix::io::Errno::WOULDBLOCK) => Ok(false),
        Err(error) => Err(error.into()),
    }
}

fn flock_blocking(
    file: &std::fs::File,
    operation: rustix::fs::FlockOperation,
) -> std::io::Result<()> {
    loop {
        match rustix::fs::flock(file, operation) {
            Err(rustix::io::Errno::INTR) => continue,
            result => return result.map_err(Into::into),
        }
    }
}

pub fn flock_shared(file: &std::fs::File) -> std::io::Result<()> {
    flock_blocking(file, rustix::fs::FlockOperation::LockShared)
}

pub fn flock_exclusive(file: &std::fs::File) -> std::io::Result<()> {
    flock_blocking(file, rustix::fs::FlockOperation::LockExclusive)
}

/// Trimmed contents of `/proc/sys/kernel/random/uuid`.
pub fn read_uuid() -> std::io::Result<String> {
    let text = std::fs::read_to_string("/proc/sys/kernel/random/uuid")?;
    Ok(text.trim().to_string())
}
