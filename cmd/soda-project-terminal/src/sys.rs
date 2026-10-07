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

/// Run `argv` with stdin null and stdout/stderr piped; capture each stream up
/// to 8 MiB. Timeout, read failure, overflow, or pipes retained past the
/// post-exit grace are errors. The caller checks `Output.status`.
pub fn run_output(argv: &[String], timeout_secs: u64) -> std::io::Result<Output> {
    const OUTPUT_LIMIT: usize = 8 << 20;
    const DRAIN_GRACE: std::time::Duration = std::time::Duration::from_secs(2);
    let mut child = spawn_argv(argv, true)?;
    let mut stdout = child.stdout.take();
    let mut stderr = child.stderr.take();
    if let Err(error) =
        set_pipe_nonblocking(stdout.as_ref()).and_then(|()| set_pipe_nonblocking(stderr.as_ref()))
    {
        let _ = child.kill();
        let _ = child.wait();
        return Err(error);
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
        if status.is_none() && now >= deadline {
            let _ = child.kill();
            let _ = child.wait();
            return Err(std::io::Error::new(
                std::io::ErrorKind::TimedOut,
                "command timed out",
            ));
        }
        if status.is_none() {
            match child.try_wait() {
                Ok(Some(done)) => {
                    status = Some(done);
                    drain_deadline = Some(now + DRAIN_GRACE);
                }
                Ok(None) => {}
                Err(error) => {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(error);
                }
            }
        }
        let mut progressed = false;
        let mut close_stdout = false;
        if let Some(pipe) = stdout.as_mut() {
            match pipe.read(&mut out_buf) {
                Ok(0) => {
                    close_stdout = true;
                    progressed = true;
                }
                Ok(n) => {
                    if out.len().saturating_add(n) > OUTPUT_LIMIT {
                        let _ = child.kill();
                        let _ = child.wait();
                        return Err(std::io::Error::new(
                            std::io::ErrorKind::InvalidData,
                            "command stdout exceeded capture bound",
                        ));
                    }
                    out.extend_from_slice(&out_buf[..n]);
                    progressed = true;
                }
                Err(error) if error.kind() == std::io::ErrorKind::WouldBlock => {}
                Err(error) if error.kind() == std::io::ErrorKind::Interrupted => {}
                Err(error) => {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(error);
                }
            }
        }
        if close_stdout {
            stdout.take();
        }
        let mut close_stderr = false;
        if let Some(pipe) = stderr.as_mut() {
            match pipe.read(&mut err_buf) {
                Ok(0) => {
                    close_stderr = true;
                    progressed = true;
                }
                Ok(n) => {
                    if err.len().saturating_add(n) > OUTPUT_LIMIT {
                        let _ = child.kill();
                        let _ = child.wait();
                        return Err(std::io::Error::new(
                            std::io::ErrorKind::InvalidData,
                            "command stderr exceeded capture bound",
                        ));
                    }
                    err.extend_from_slice(&err_buf[..n]);
                    progressed = true;
                }
                Err(error) if error.kind() == std::io::ErrorKind::WouldBlock => {}
                Err(error) if error.kind() == std::io::ErrorKind::Interrupted => {}
                Err(error) => {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(error);
                }
            }
        }
        if close_stderr {
            stderr.take();
        }
        if let Some(done) = status {
            if stdout.is_none() && stderr.is_none() {
                return Ok(Output {
                    status: done,
                    stdout: out,
                    stderr: err,
                });
            }
            if drain_deadline.is_some_and(|at| Instant::now() >= at) {
                stdout.take();
                stderr.take();
                return Err(std::io::Error::new(
                    std::io::ErrorKind::TimedOut,
                    "command output pipes did not close",
                ));
            }
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

#[cfg(test)]
mod tests {
    use super::*;

    static TEST_SEQ: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);

    /// Unique per-test scratch dir (tests run in parallel threads).
    fn test_dir(tag: &str) -> std::path::PathBuf {
        let n = TEST_SEQ.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        let dir =
            std::env::temp_dir().join(format!("soda-pt-sys-{tag}-{}-{n}", std::process::id()));
        std::fs::create_dir_all(&dir).expect("test dir");
        dir
    }

    fn argv(words: &[&str]) -> Vec<String> {
        words.iter().map(|w| w.to_string()).collect()
    }

    #[test]
    fn clocks() {
        let a = monotonic();
        let b = monotonic();
        assert!(a > 0.0 && b >= a);
        let sys_now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;
        assert!((now_secs() - sys_now).abs() <= 2);
    }

    #[test]
    fn run_checked_matrix() {
        assert!(run_checked(&argv(&["/bin/true"]), 10).is_ok());
        let err = run_checked(&argv(&["/bin/false"]), 10).unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::Other);
        let err = run_checked(&argv(&["/no/such/binary-soda-pt"]), 10).unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::NotFound);
        let err = run_checked(&[], 10).unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::InvalidInput);
    }

    #[test]
    fn run_checked_timeout_kills() {
        let start = monotonic();
        let err = run_checked(&argv(&["/bin/sleep", "30"]), 1).unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::TimedOut);
        assert!(monotonic() - start < 10.0, "timeout must fire promptly");
        // Zero timeout on a slow child also times out instead of hanging.
        let err = run_checked(&argv(&["/bin/sleep", "30"]), 0).unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::TimedOut);
        // The killed children were reaped; the runner still works.
        assert!(run_checked(&argv(&["/bin/true"]), 10).is_ok());
    }

    #[test]
    fn run_output_matrix() {
        let out = run_output(&argv(&["/bin/echo", "hello"]), 10).unwrap();
        assert!(out.status.success());
        assert_eq!(out.stdout, b"hello\n");
        let out = run_output(&argv(&["/bin/false"]), 10).unwrap();
        assert!(!out.status.success());
        let err = run_output(&argv(&["/bin/sleep", "30"]), 1).unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::TimedOut);
        let err = run_output(&[], 10).unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::InvalidInput);
    }

    #[test]
    fn run_output_large_no_deadlock() {
        // ~1.9MB through the pipe: far beyond the 64KiB pipe buffer, so a
        // wait-then-read implementation would deadlock here.
        let out = run_output(&argv(&["/bin/sh", "-c", "seq 1 200000"]), 60).unwrap();
        assert!(out.status.success());
        assert!(out.stdout.len() > 1_000_000);
        assert!(out.stdout.ends_with(b"200000\n"));
        assert!(out.stdout.starts_with(b"1\n2\n3\n"));
    }

    #[test]
    fn run_output_drains_both_pipes_in_bounded_passes() {
        let out = run_output(
            &argv(&[
                "/bin/sh",
                "-c",
                "head -c 1000000 /dev/zero; head -c 1000000 /dev/zero >&2",
            ]),
            10,
        )
        .unwrap();
        assert!(out.status.success());
        assert_eq!(out.stdout.len(), 1_000_000);
        assert_eq!(out.stderr.len(), 1_000_000);
    }

    #[test]
    fn run_output_fails_promptly_at_per_stream_bound() {
        let err = run_output(&argv(&["/usr/bin/yes"]), 10).unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::InvalidData);
        let err = run_output(&argv(&["/bin/sh", "-c", "exec yes >&2"]), 10).unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::InvalidData);
    }

    #[test]
    fn run_output_requires_pipe_eof_after_child_exit() {
        let err = run_output(&argv(&["/bin/sh", "-c", "sleep 2.5 & exit 0"]), 10).unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::TimedOut);
    }

    #[test]
    fn lock_contention_two_fds() {
        let dir = test_dir("lock");
        let path = dir.join("lockfile");
        std::fs::write(&path, b"x").unwrap();
        let first = std::fs::File::open(&path).unwrap();
        let second = std::fs::File::open(&path).unwrap();
        assert!(flock_exclusive_nb(&first).unwrap());
        assert!(!flock_exclusive_nb(&second).unwrap());
        drop(first);
        assert!(flock_exclusive_nb(&second).unwrap());
        // Blocking shared/exclusive on an uncontended fd succeed.
        assert!(flock_shared(&second).is_ok());
        assert!(flock_exclusive(&second).is_ok());
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn open_walk_and_component_rule() {
        let root = open_root().unwrap();
        let usr = open_child_dir(&root, "usr").unwrap();
        let _ = open_child_dir(&usr, "bin").unwrap();
        for bad in ["", ".", "..", "a/b", "/bin", "a\0b"] {
            assert!(open_child_dir(&root, bad).is_err(), "component {bad:?}");
            assert!(
                open_at(&root, bad, OFlags::RDONLY, rustix::fs::Mode::empty()).is_err(),
                "open_at {bad:?}"
            );
        }
        assert!(open_child_dir(&root, "no-such-dir-soda-pt").is_err());
        // O_NOFOLLOW: a symlink component fails with ELOOP.
        let dir = test_dir("symlink");
        std::os::unix::fs::symlink("target", dir.join("link")).unwrap();
        let tmp_fd = std::fs::File::open(&dir).unwrap();
        let err = open_at(&tmp_fd, "link", OFlags::RDONLY, rustix::fs::Mode::empty()).unwrap_err();
        assert_eq!(err.raw_os_error(), Some(libc::ELOOP));
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn opened_fds_are_cloexec() {
        let root = open_root().unwrap();
        for file in [&root, &open_child_dir(&root, "usr").unwrap()] {
            let flags = unsafe { libc::fcntl(file.as_raw_fd(), libc::F_GETFD) };
            assert!(flags >= 0 && flags & libc::FD_CLOEXEC != 0);
        }
    }

    #[test]
    fn uuid_shape() {
        let uuid = read_uuid().unwrap();
        assert_eq!(uuid.len(), 36);
        assert_eq!(uuid.bytes().filter(|b| *b == b'-').count(), 4);
        assert!(uuid.bytes().all(|b| b.is_ascii_hexdigit() || b == b'-'));
        assert_ne!(uuid, read_uuid().unwrap(), "uuids must differ per read");
    }
}
