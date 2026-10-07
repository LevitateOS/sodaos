// Shared provider support: errors, private enrollment roots and
// captured child execution. Folded from identity-providers (A06.M).
// Rust port of the Soda identity subscription providers (PR23).
// Mirrors internal/identity/{codex,muse}: pinned native enrollment sessions,
// credential custody checks and the shared enrollment wire shapes. The Go
// broker keeps calling the Go providers until the PR28 broker cutover; this

pub mod codex;
pub mod muse;
pub mod sha256;
pub mod types;

use std::fmt;
use std::io::{self, Read};
use std::os::fd::AsRawFd;
use std::os::unix::process::CommandExt;
use std::path::Path;
use std::process::{Child, Command};
use std::time::Duration;

/// Failure class. Provider error text stays out of client responses (the
/// broker maps unknown failures to `unavailable`), except for the fixed
/// `Enrollment.error` strings each session records, which are preserved
/// byte-identically in the provider modules.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Kind {
    /// Identity authority denied: invalid configuration, request or file.
    Denied,
    /// Subscription requires reconnection: teardown could not confirm.
    Uncertain,
    /// Any other enrollment failure.
    Failed,
}

#[derive(Debug)]
pub struct Error {
    kind: Kind,
    message: String,
}

impl Error {
    pub fn denied(message: impl Into<String>) -> Error {
        Error {
            kind: Kind::Denied,
            message: message.into(),
        }
    }
    pub fn uncertain(message: impl Into<String>) -> Error {
        Error {
            kind: Kind::Uncertain,
            message: message.into(),
        }
    }
    pub fn failed(message: impl Into<String>) -> Error {
        Error {
            kind: Kind::Failed,
            message: message.into(),
        }
    }
    pub fn kind(&self) -> Kind {
        self.kind
    }
    pub fn is_denied(&self) -> bool {
        self.kind == Kind::Denied
    }
    pub fn is_uncertain(&self) -> bool {
        self.kind == Kind::Uncertain
    }
}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.message)
    }
}

impl std::error::Error for Error {}

impl From<std::io::Error> for Error {
    fn from(err: std::io::Error) -> Error {
        Error::failed(err.to_string())
    }
}

/// Unique `enrollment-*` directory under a provider root, mirroring
/// `os.MkdirTemp(root, "enrollment-")`.
pub(crate) fn enrollment_tempdir(root: &Path) -> std::io::Result<tempfile::TempDir> {
    enrollment_tempdir_with(root, |out| {
        getrandom::fill(out).map_err(std::io::Error::other)
    })
}

fn enrollment_tempdir_with(
    root: &Path,
    mut fill: impl FnMut(&mut [u8]) -> std::io::Result<()>,
) -> std::io::Result<tempfile::TempDir> {
    // Keep the provider's fail-closed entropy boundary explicit; tempfile's
    // own collision-safe suffix is used only after this required source passes.
    let mut prefix = [0u8; 16];
    fill(&mut prefix).map_err(|_| std::io::Error::other("enrollment randomness unavailable"))?;
    use std::os::unix::fs::PermissionsExt;
    tempfile::Builder::new()
        .prefix(&format!("enrollment-{}-", sha256::hex(&prefix)))
        .rand_bytes(8)
        .permissions(std::fs::Permissions::from_mode(0o700))
        .tempdir_in(root)
}

/// Enrollment roots must be private tmpfs, mirroring `privateTmpfs`.
pub(crate) fn private_tmpfs(root: &Path) -> Result<(), Error> {
    #[cfg(target_os = "linux")]
    {
        let path = root.as_os_str().as_encoded_bytes();
        let mut buf = vec![0u8; path.len() + 1];
        buf[..path.len()].copy_from_slice(path);
        let mut fs: libc::statfs64 = unsafe { std::mem::zeroed() };
        // `statfs64` matches the kernel ABI on every supported x86_64 target.
        let ok = unsafe { libc::statfs64(buf.as_ptr() as *const libc::c_char, &mut fs) == 0 };
        if !ok {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        if fs.f_type as u32 != 0x01021994 {
            return Err(Error::denied("enrollment root must be tmpfs"));
        }
        Ok(())
    }
    #[cfg(not(target_os = "linux"))]
    {
        let _ = root;
        Err(Error::denied("identity enrollment requires Linux tmpfs"))
    }
}

#[cfg(test)]
pub(crate) struct TestDir {
    dir: tempfile::TempDir,
}

#[cfg(test)]
impl TestDir {
    pub(crate) fn new() -> TestDir {
        TestDir {
            dir: tempfile::Builder::new()
                .prefix("soda-idp-test-")
                .tempdir()
                .expect("test temporary directory unavailable"),
        }
    }

    pub(crate) fn path(&self) -> &Path {
        self.dir.path()
    }
}

#[cfg(test)]
mod entropy_tests {
    use super::*;

    #[test]
    fn enrollment_tempdir_propagates_partial_entropy_failure() {
        let parent = tempfile::tempdir().unwrap();
        let root = parent.path();
        let result = enrollment_tempdir_with(root, |out| {
            out[0] = 1;
            Err(std::io::Error::other("injected entropy failure"))
        });
        assert!(result.is_err());
        assert_eq!(std::fs::read_dir(root).unwrap().count(), 0);
        let made = enrollment_tempdir_with(root, |out| {
            out.fill(1);
            Ok(())
        })
        .unwrap();
        use std::os::unix::fs::PermissionsExt;
        assert_eq!(
            std::fs::metadata(made.path()).unwrap().permissions().mode() & 0o777,
            0o700
        );
        made.close().unwrap();
    }

    #[test]
    fn version_probe_accepts_output_at_the_limit_and_rejects_one_byte_over() {
        let exact = run_capture(
            "/bin/sh",
            &["-c", "dd if=/dev/zero bs=65536 count=1 2>/dev/null"],
            &[],
            None,
            Duration::from_secs(2),
        )
        .unwrap();
        assert_eq!(exact.len(), VERSION_PROBE_OUTPUT_LIMIT);

        let over = run_capture(
            "/bin/sh",
            &["-c", "dd if=/dev/zero bs=65537 count=1 2>/dev/null"],
            &[],
            None,
            Duration::from_secs(2),
        )
        .unwrap_err();
        assert_eq!(over.to_string(), "version probe failed");
    }

    #[test]
    fn version_probe_closes_descendant_held_stdout_without_reader_join() {
        let started = std::time::Instant::now();
        let output = run_capture(
            "/bin/sh",
            &["-c", "printf 'pinned version'; (sleep 2) & exit 0"],
            &[],
            None,
            Duration::from_secs(2),
        )
        .unwrap();
        assert_eq!(output, "pinned version");
        assert!(started.elapsed() < Duration::from_secs(1));
    }

    #[test]
    fn version_probe_timeout_has_bounded_cleanup() {
        let started = std::time::Instant::now();
        let error = run_capture(
            "/bin/sh",
            &["-c", "sleep 2"],
            &[],
            None,
            Duration::from_millis(25),
        )
        .unwrap_err();
        assert_eq!(error.to_string(), "version probe timed out");
        assert!(started.elapsed() < Duration::from_secs(1));
    }
}

const VERSION_PROBE_OUTPUT_LIMIT: usize = 64 * 1024;
const VERSION_PROBE_CLEANUP: Duration = Duration::from_millis(250);

/// Run a version probe with bounded output and time. A complete, successful
/// child is required before any captured bytes reach the version comparison.
pub(crate) fn run_capture(
    binary: &str,
    args: &[&str],
    env: &[String],
    dir: Option<&Path>,
    timeout: Duration,
) -> Result<String, Error> {
    let mut command = Command::new(binary);
    command
        .args(args)
        .env_clear()
        .envs(env.iter().map(|e| {
            let (k, v) = e.split_once('=').unwrap_or((e.as_str(), ""));
            (k, v)
        }))
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::null());
    command.process_group(0);
    if let Some(dir) = dir {
        command.current_dir(dir);
    }
    let deadline = std::time::Instant::now() + timeout;
    let mut child: Child = command
        .spawn()
        .map_err(|_| Error::failed("version probe failed"))?;
    let mut stdout = match child.stdout.take() {
        Some(stdout) => stdout,
        None => {
            return Err(
                if terminate_probe_child(&mut child, VERSION_PROBE_CLEANUP) {
                    Error::failed("version probe failed")
                } else {
                    Error::failed("version probe cleanup unconfirmed")
                },
            );
        }
    };
    let flags = unsafe { libc::fcntl(stdout.as_raw_fd(), libc::F_GETFL) };
    if flags < 0
        || unsafe { libc::fcntl(stdout.as_raw_fd(), libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0
    {
        return Err(
            if terminate_probe_child(&mut child, VERSION_PROBE_CLEANUP) {
                Error::failed("version probe failed")
            } else {
                Error::failed("version probe cleanup unconfirmed")
            },
        );
    }
    let mut text = Vec::with_capacity(VERSION_PROBE_OUTPUT_LIMIT);
    let mut eof = false;
    let mut overflow = false;
    let mut read_failed = false;
    let mut child_status = None;
    let mut timed_out = false;
    let mut completion_before_deadline = false;
    let mut cleanup_unconfirmed = false;
    let mut buffer = [0u8; 4096];
    'probe: loop {
        loop {
            let remaining = VERSION_PROBE_OUTPUT_LIMIT
                .saturating_add(1)
                .saturating_sub(text.len());
            if remaining == 0 {
                overflow = true;
                break;
            }
            let cap = buffer.len().min(remaining);
            match stdout.read(&mut buffer[..cap]) {
                Ok(0) => {
                    eof = true;
                    break;
                }
                Ok(n) => {
                    let admitted = n.min(VERSION_PROBE_OUTPUT_LIMIT.saturating_sub(text.len()));
                    text.extend_from_slice(&buffer[..admitted]);
                    if admitted != n {
                        overflow = true;
                        break;
                    }
                }
                Err(err) if err.kind() == io::ErrorKind::WouldBlock => break,
                Err(err) if err.kind() == io::ErrorKind::Interrupted => {
                    if std::time::Instant::now() >= deadline {
                        timed_out = true;
                        break;
                    }
                    continue;
                }
                Err(_) => {
                    read_failed = true;
                    break;
                }
            }
        }

        let mut info: libc::siginfo_t = unsafe { std::mem::zeroed() };
        let wait_result = unsafe {
            libc::waitid(
                libc::P_PID,
                child.id() as libc::id_t,
                &mut info,
                libc::WEXITED | libc::WNOHANG | libc::WNOWAIT,
            )
        };
        if wait_result < 0 {
            let wait_error = io::Error::last_os_error();
            if wait_error.kind() == io::ErrorKind::Interrupted {
                if std::time::Instant::now() < deadline {
                    continue 'probe;
                }
                timed_out = true;
            } else {
                read_failed = true;
            }
            // No numeric group signal is safe unless waitid successfully
            // confirmed this unreaped child in the current iteration.
            cleanup_unconfirmed = true;
            break;
        }
        if unsafe { info.si_pid() } != 0 {
            child_status =
                Some(info.si_code == libc::CLD_EXITED && unsafe { info.si_status() } == 0);
            // Keep the leader unreaped while killing descendants in its group,
            // so the group ID cannot be reused before this cleanup signal.
            cleanup_unconfirmed |= !terminate_probe_group(child.id());
            completion_before_deadline = eof && std::time::Instant::now() < deadline;
            break;
        }
        if overflow || read_failed {
            cleanup_unconfirmed |= !terminate_probe_group(child.id());
            break;
        }
        if std::time::Instant::now() >= deadline {
            timed_out = true;
            cleanup_unconfirmed |= !terminate_probe_group(child.id());
            break;
        }
        std::thread::sleep(Duration::from_millis(2));
    }

    let cleanup_deadline = std::time::Instant::now() + VERSION_PROBE_CLEANUP;
    if child_status.is_none() {
        while std::time::Instant::now() < cleanup_deadline {
            let mut info: libc::siginfo_t = unsafe { std::mem::zeroed() };
            let result = unsafe {
                libc::waitid(
                    libc::P_PID,
                    child.id() as libc::id_t,
                    &mut info,
                    libc::WEXITED | libc::WNOHANG | libc::WNOWAIT,
                )
            };
            if result < 0 && io::Error::last_os_error().kind() == io::ErrorKind::Interrupted {
                continue;
            }
            if result < 0 {
                read_failed = true;
                break;
            }
            if unsafe { info.si_pid() } != 0 {
                child_status =
                    Some(info.si_code == libc::CLD_EXITED && unsafe { info.si_status() } == 0);
                break;
            }
            std::thread::sleep(Duration::from_millis(2));
        }
    }
    let mut reaped = false;
    if child_status.is_some() {
        reaped = child.wait().is_ok();
        cleanup_unconfirmed |= !reaped;
    } else {
        cleanup_unconfirmed = true;
    }
    // The descriptor is nonblocking, so draining after group termination is
    // bounded by the same cleanup window and never needs a reader thread.
    loop {
        match stdout.read(&mut buffer) {
            Ok(0) => {
                eof = true;
                completion_before_deadline =
                    child_status.is_some() && std::time::Instant::now() < deadline;
                break;
            }
            Ok(n) => {
                let admitted = n.min(VERSION_PROBE_OUTPUT_LIMIT.saturating_sub(text.len()));
                text.extend_from_slice(&buffer[..admitted]);
                if admitted != n {
                    overflow = true;
                    break;
                }
            }
            Err(err) if err.kind() == io::ErrorKind::WouldBlock => {
                if std::time::Instant::now() >= cleanup_deadline {
                    cleanup_unconfirmed = true;
                    break;
                }
                std::thread::sleep(Duration::from_millis(2));
            }
            Err(err) if err.kind() == io::ErrorKind::Interrupted => {
                if std::time::Instant::now() >= cleanup_deadline {
                    cleanup_unconfirmed = true;
                    break;
                }
                continue;
            }
            Err(_) => {
                read_failed = true;
                break;
            }
        }
    }
    if !reaped || cleanup_unconfirmed || !eof {
        return Err(Error::failed("version probe cleanup unconfirmed"));
    }
    if timed_out || !completion_before_deadline {
        return Err(Error::failed("version probe timed out"));
    }
    if child_status != Some(true) || overflow || read_failed {
        return Err(Error::failed("version probe failed"));
    }
    Ok(String::from_utf8_lossy(&text).into_owned())
}

fn terminate_probe_group(pid: u32) -> bool {
    // SAFETY: the command created a new process group whose ID is its PID.
    let result = unsafe { libc::kill(-(pid as libc::pid_t), libc::SIGKILL) };
    result == 0 || io::Error::last_os_error().raw_os_error() == Some(libc::ESRCH)
}

fn wait_probe_child(child: &mut Child, allowance: Duration) -> bool {
    let deadline = std::time::Instant::now() + allowance;
    loop {
        let mut info: libc::siginfo_t = unsafe { std::mem::zeroed() };
        let result = unsafe {
            libc::waitid(
                libc::P_PID,
                child.id() as libc::id_t,
                &mut info,
                libc::WEXITED | libc::WNOHANG | libc::WNOWAIT,
            )
        };
        if result < 0 && io::Error::last_os_error().kind() == io::ErrorKind::Interrupted {
            if std::time::Instant::now() < deadline {
                continue;
            }
            return false;
        }
        if result < 0 {
            return false;
        }
        if unsafe { info.si_pid() } != 0 {
            return child.wait().is_ok();
        }
        if std::time::Instant::now() >= deadline {
            return false;
        }
        std::thread::sleep(Duration::from_millis(2));
    }
}

fn terminate_probe_child(child: &mut Child, allowance: Duration) -> bool {
    let deadline = std::time::Instant::now() + allowance;
    loop {
        let mut info: libc::siginfo_t = unsafe { std::mem::zeroed() };
        let result = unsafe {
            libc::waitid(
                libc::P_PID,
                child.id() as libc::id_t,
                &mut info,
                libc::WEXITED | libc::WNOHANG | libc::WNOWAIT,
            )
        };
        if result == 0 {
            break;
        }
        if io::Error::last_os_error().kind() != io::ErrorKind::Interrupted
            || std::time::Instant::now() >= deadline
        {
            return false;
        }
        std::thread::sleep(Duration::from_millis(2));
    }
    // A successful waitid either confirms this child is still live (si_pid=0)
    // or terminal-but-unreaped. In both cases its PID/process-group ID cannot
    // have been reused before the following signal.
    if !terminate_probe_group(child.id()) {
        return false;
    }
    wait_probe_child(
        child,
        deadline.saturating_duration_since(std::time::Instant::now()),
    )
}
