use std::fs::File;
use std::io::{BufRead, BufReader, Write};
use std::os::unix::fs::{MetadataExt, PermissionsExt};
use std::os::unix::io::AsRawFd;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

use crate::account::AGENT_PROGRAM;
use crate::domain;
use crate::project::Executor;
use crate::sha256;

use super::{
    agent_argv, harness_pipeline_argv, now_unix, Service, TerminalFrame, TerminalRequest,
    FRAME_LIMIT,
};

const HARNESS_TRANSFER_STDOUT_LIMIT: usize = 64 * 1024;
const HARNESS_TRANSFER_STDERR_LIMIT: usize = 1024 * 1024;

pub(super) fn run_harness_pipeline(
    exec: &dyn Executor,
    tar: &str,
    podman: &str,
    harness: &str,
    container: &str,
    path: &str,
    deadline: Instant,
) -> Result<(), String> {
    let args = harness_pipeline_argv(tar, podman, harness, container, path);
    let refs: Vec<&str> = args.iter().map(String::as_str).collect();
    exec.run_bounded(
        &[],
        "/usr/bin/bash",
        &refs,
        deadline,
        HARNESS_TRANSFER_STDOUT_LIMIT,
        HARNESS_TRANSFER_STDERR_LIMIT,
    )
    .map(|_| ())
}

/// Render path components with Rust, preserving parent components so callers
/// can reject them instead of silently resolving them.
pub fn clean_path(path: &str) -> String {
    let components: PathBuf = Path::new(path).components().collect();
    if components.as_os_str().is_empty() {
        ".".to_string()
    } else {
        components.to_string_lossy().into_owned()
    }
}

pub fn is_clean_absolute_path(path: &str) -> bool {
    if !Path::new(path).is_absolute() {
        return false;
    }
    let mut rebuilt = PathBuf::new();
    for component in Path::new(path).components() {
        match component {
            std::path::Component::RootDir => rebuilt.push("/"),
            std::path::Component::Normal(part) => rebuilt.push(part),
            std::path::Component::CurDir
            | std::path::Component::ParentDir
            | std::path::Component::Prefix(_) => return false,
        }
    }
    rebuilt.to_str() == Some(path)
}

impl<E: Executor> Service<E> {
    /// `Service.streamIdentityHarness`: stage the verified harness bytes
    /// into the guest tmpfs. The producer and consumer share one owned
    /// process group and the caller's absolute deadline.
    pub fn stream_identity_harness(
        &self,
        container: &str,
        path: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        run_harness_pipeline(
            &self.exec,
            "/usr/bin/tar",
            "/usr/bin/podman",
            &self.codex_harness,
            container,
            path,
            deadline,
        )
        .map_err(|_| "codex harness staging failed".to_string())
    }
}

// ---------- agent binary verification (agent.go) ----------

/// Host-side agent path: `SODA_PROJECT_TERMINAL` override or the fixed
/// program path. Ownership/mode/size gates still apply to overrides.
pub fn agent_program_path() -> String {
    std::env::var("SODA_PROJECT_TERMINAL")
        .ok()
        .filter(|v| !v.is_empty())
        .unwrap_or_else(|| AGENT_PROGRAM.to_string())
}

/// `agentProgramHash(uid)`: verify the fixed host-side agent binary and
/// return its lowercase hex SHA-256. Fails closed on missing, non-regular,
/// wrongly owned, writable, or oddly sized files. Production passes 0.
pub fn agent_program_hash(uid: u32) -> Result<String, String> {
    let path = agent_program_path();
    use std::io::Read;
    use std::os::unix::fs::OpenOptionsExt;
    let file = std::fs::OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_CLOEXEC | libc::O_NOFOLLOW | libc::O_NONBLOCK)
        .open(&path)
        .map_err(|e| {
            if e.raw_os_error() == Some(libc::ELOOP) {
                "project terminal agent is not a regular file".to_string()
            } else {
                format!("project terminal agent unavailable: {e}")
            }
        })?;
    let info = file
        .metadata()
        .map_err(|e| format!("project terminal agent unavailable: {e}"))?;
    if !info.file_type().is_file() {
        return Err("project terminal agent is not a regular file".to_string());
    }
    if info.uid() != uid {
        return Err("project terminal agent has unexpected ownership".to_string());
    }
    if info.permissions().mode() & 0o022 != 0 {
        return Err("project terminal agent is group- or world-writable".to_string());
    }
    if info.len() < 1 || info.len() > 32 << 20 {
        return Err("project terminal agent has unexpected size".to_string());
    }
    let mut raw = Vec::new();
    file.take((32 << 20) + 1)
        .read_to_end(&mut raw)
        .map_err(|e| format!("project terminal agent unreadable: {e}"))?;
    if raw.is_empty() || raw.len() > 32 << 20 {
        return Err("project terminal agent changed during verification".to_string());
    }
    Ok(sha256::hex_lower(&sha256::digest(&raw)))
}

// ---------- native attach (native.go) ----------

/// `AttachNative` argv: the fixed podman/agent attachment bridge.
/// `seconds` is the attachment deadline (`expires - now`), already checked
/// positive by the caller path below.
#[allow(clippy::too_many_arguments)] // one parameter per fixed argv word, in order
pub fn native_argv(
    container: &str,
    action: &str,
    id: &str,
    login: &str,
    identity: i64,
    cols: i64,
    rows: i64,
    seconds: i64,
    hash: &str,
    name: &str,
    scope: &str,
) -> Vec<String> {
    agent_argv(
        container,
        &[
            action,
            id,
            login,
            &identity.to_string(),
            &cols.to_string(),
            &rows.to_string(),
            &seconds.to_string(),
            hash,
            name,
            scope,
        ],
    )
}

/// Parse one agent stdout line: strict JSON frame plus `OutputValid`.
/// Scanner-failure (overlong line, EOF) is `None`; decode/validity failure
/// is the `invalid terminal response` error.
pub fn parse_output_line(line: &[u8]) -> Result<TerminalFrame, String> {
    TerminalFrame::decode(line)
        .map_err(|_| "invalid terminal response".to_string())
        .and_then(|f| {
            if f.output_valid() {
                Ok(f)
            } else {
                Err("invalid terminal response".to_string())
            }
        })
}

/// Streaming native attachment (`nativeTerminal`). Terminal bytes never
/// enter diagnostics; stdin close requests launcher EOF.
#[derive(Debug)]
pub struct NativeAttach {
    pub(in crate::terminal) child: Option<std::process::Child>,
    pub(in crate::terminal) stdin: Option<File>,
    pub(in crate::terminal) reader: Option<BufReader<File>>,
    pub(in crate::terminal) closed: bool,
    pub(in crate::terminal) close_failure: Option<String>,
    pub(in crate::terminal) shutdown: Arc<AtomicBool>,
}

fn reap_failed_attach(
    mut child: std::process::Child,
    reason: &str,
    shutdown: Arc<AtomicBool>,
) -> String {
    drop(child.stdin.take());
    drop(child.stdout.take());
    let grace_deadline = Instant::now() + Duration::from_secs(3);
    let mut may_kill = true;
    let mut kill_attempted = false;
    let mut reap_deadline = None;
    loop {
        match child.try_wait() {
            Ok(Some(_)) => return reason.to_string(),
            Ok(None) if may_kill && !kill_attempted && Instant::now() >= grace_deadline => {
                kill_attempted = true;
                reap_deadline = Some(Instant::now() + Duration::from_secs(3));
                if child.kill().is_err() {
                    shutdown.store(true, Ordering::SeqCst);
                }
            }
            Ok(None) => {
                if reap_deadline.is_some_and(|deadline| Instant::now() >= deadline) {
                    shutdown.store(true, Ordering::SeqCst);
                }
            }
            Err(_) => {
                may_kill = false;
                shutdown.store(true, Ordering::SeqCst);
            }
        }
        std::thread::sleep(Duration::from_millis(10));
    }
}

impl NativeAttach {
    #[cfg(test)]
    pub(crate) fn from_child_for_test(
        mut child: std::process::Child,
        shutdown: Arc<AtomicBool>,
    ) -> Result<Self, String> {
        use std::os::fd::FromRawFd;
        let stdin = match child.stdin.take() {
            Some(stdin) => stdin,
            None => {
                return Err(reap_failed_attach(
                    child,
                    "test stdin unavailable",
                    shutdown,
                ))
            }
        };
        let stdout = match child.stdout.take() {
            Some(stdout) => stdout,
            None => {
                drop(stdin);
                return Err(reap_failed_attach(
                    child,
                    "test stdout unavailable",
                    shutdown,
                ));
            }
        };
        let stdin_fd = stdin.as_raw_fd();
        let stdout_fd = stdout.as_raw_fd();
        std::mem::forget(stdin);
        std::mem::forget(stdout);
        let stdin = unsafe { File::from_raw_fd(stdin_fd) };
        let stdout = unsafe { File::from_raw_fd(stdout_fd) };
        let flags = unsafe { libc::fcntl(stdin.as_raw_fd(), libc::F_GETFL) };
        if flags < 0
            || unsafe { libc::fcntl(stdin.as_raw_fd(), libc::F_SETFL, flags | libc::O_NONBLOCK) }
                < 0
        {
            drop(stdin);
            drop(stdout);
            return Err(reap_failed_attach(
                child,
                "test terminal input unavailable",
                shutdown,
            ));
        }
        Ok(NativeAttach {
            child: Some(child),
            stdin: Some(stdin),
            reader: Some(BufReader::new(stdout)),
            closed: false,
            close_failure: None,
            shutdown,
        })
    }

    /// `AttachNative`: start the fixed podman/agent attachment bridge.
    pub fn attach(
        container: &str,
        input: &TerminalRequest,
        shutdown: Arc<AtomicBool>,
    ) -> Result<Self, String> {
        let seconds = input.expires - now_unix();
        if !domain::valid_container_id(container) || !input.valid(now_unix()) || seconds < 1 {
            return Err("invalid terminal target".to_string());
        }
        let hash = agent_program_hash(0)?;
        let argv = native_argv(
            container,
            &input.action,
            &input.id,
            &input.login,
            input.identity,
            input.cols,
            input.rows,
            seconds,
            &hash,
            &input.name,
            &input.scope,
        );
        let mut child = std::process::Command::new("/usr/bin/podman")
            .args(&argv)
            .stdin(std::process::Stdio::piped())
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::null())
            .spawn()
            .map_err(|e| format!("/usr/bin/podman failed: {e}"))?;
        let stdin = match child.stdin.take() {
            Some(stdin) => stdin,
            None => {
                return Err(reap_failed_attach(
                    child,
                    "terminal stdin unavailable",
                    shutdown,
                ))
            }
        };
        let stdout = match child.stdout.take() {
            Some(stdout) => stdout,
            None => {
                drop(stdin);
                return Err(reap_failed_attach(
                    child,
                    "terminal stdout unavailable",
                    shutdown,
                ));
            }
        };
        use std::os::unix::io::FromRawFd;
        // `ChildStdin` has no timeout API; convert to `File` for deadlines.
        // SAFETY: the stdio handles are owned by us exactly once.
        let stdin_fd = stdin.as_raw_fd();
        let stdout_fd = stdout.as_raw_fd();
        std::mem::forget(stdin);
        std::mem::forget(stdout);
        let stdin = unsafe { File::from_raw_fd(stdin_fd) };
        let stdout = unsafe { File::from_raw_fd(stdout_fd) };
        let flags = unsafe { libc::fcntl(stdin.as_raw_fd(), libc::F_GETFL) };
        if flags < 0
            || unsafe { libc::fcntl(stdin.as_raw_fd(), libc::F_SETFL, flags | libc::O_NONBLOCK) }
                < 0
        {
            drop(stdin);
            drop(stdout);
            return Err(reap_failed_attach(
                child,
                "terminal input unavailable",
                shutdown,
            ));
        }
        Ok(NativeAttach {
            child: Some(child),
            stdin: Some(stdin),
            reader: Some(BufReader::new(stdout)),
            closed: false,
            close_failure: None,
            shutdown,
        })
    }

    /// `nativeTerminal.Input`: one validated frame over stdin (2s deadline).
    pub fn input_frame(&mut self, f: &TerminalFrame) -> Result<(), String> {
        if !f.input_valid() {
            return Err("invalid terminal control".to_string());
        }
        let stdin = self
            .stdin
            .as_mut()
            .ok_or_else(|| "terminal input ended".to_string())?;
        let mut body = f.encode().into_bytes();
        body.push(b'\n');
        let deadline = Instant::now() + Duration::from_secs(2);
        let mut offset = 0;
        while offset < body.len() {
            let remaining = deadline.saturating_duration_since(Instant::now());
            if remaining.is_zero() {
                return Err("terminal input deadline exceeded".to_string());
            }
            let mut pfd = libc::pollfd {
                fd: stdin.as_raw_fd(),
                events: libc::POLLOUT,
                revents: 0,
            };
            let wait = remaining.as_millis().min(50) as i32;
            let ready = unsafe { libc::poll(&mut pfd, 1, wait.max(1)) };
            if ready < 0 {
                if std::io::Error::last_os_error().kind() == std::io::ErrorKind::Interrupted {
                    continue;
                }
                return Err("terminal input ended".to_string());
            }
            if ready == 0 {
                continue;
            }
            if pfd.revents & (libc::POLLERR | libc::POLLHUP | libc::POLLNVAL) != 0 {
                return Err("terminal input ended".to_string());
            }
            match stdin.write(&body[offset..]) {
                Ok(0) => return Err("terminal input ended".to_string()),
                Ok(written) => offset += written,
                Err(e)
                    if e.kind() == std::io::ErrorKind::WouldBlock
                        || e.kind() == std::io::ErrorKind::Interrupted =>
                {
                    continue
                }
                Err(_) => return Err("terminal input ended".to_string()),
            }
        }
        Ok(())
    }

    /// Detach the stdout reader for lock-free output pumps (H01-F3).
    /// Teardown still funnels through [`Self::close`].
    pub fn take_reader(&mut self) -> Option<BufReader<File>> {
        let reader = self.reader.take()?;
        let fd = reader.get_ref().as_raw_fd();
        let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
        if flags < 0 || unsafe { libc::fcntl(fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
            self.reader = Some(reader);
            return None;
        }
        Some(reader)
    }

    /// `nativeTerminal.Output`: one validated agent stdout line.
    pub fn output_frame(reader: &mut BufReader<File>) -> Result<TerminalFrame, String> {
        let mut line = Vec::new();
        // `bufio.Scanner` with a 131072-byte token cap: overlong lines and
        // EOF both end the stream.
        let mut total = 0usize;
        loop {
            let chunk = reader
                .fill_buf()
                .map_err(|_| "terminal output ended".to_string())?;
            if chunk.is_empty() {
                return Err("terminal output ended".to_string());
            }
            let end = chunk.iter().position(|&b| b == b'\n');
            let take = match end {
                Some(i) => i + 1,
                None => chunk.len(),
            };
            total += take;
            if total > FRAME_LIMIT {
                return Err("terminal output ended".to_string());
            }
            line.extend_from_slice(&chunk[..take]);
            reader.consume(take);
            if end.is_some() {
                break;
            }
        }
        line.pop();
        // `ScanLines` strips one trailing `\r`.
        if line.last() == Some(&b'\r') {
            line.pop();
        }
        parse_output_line(&line)
    }

    /// `nativeTerminal.Close`: stdin EOF, 3s grace, then kill and bounded reap.
    /// An unconfirmed child remains owned here; callers must retain this attach
    /// until termination is confirmed or the host's shutdown cutoff is reached.
    pub fn close(&mut self) -> Result<(), String> {
        if let Some(error) = &self.close_failure {
            return Err(error.clone());
        }
        if self.closed {
            return Ok(());
        }
        self.closed = true;
        drop(self.stdin.take());
        drop(self.reader.take());
        if self.child.is_some() {
            let deadline = Instant::now() + Duration::from_secs(3);
            loop {
                let status = match self.child.as_mut().expect("child checked").try_wait() {
                    Ok(status) => status,
                    Err(_) => return self.fail_close("terminal child status unconfirmed"),
                };
                match status {
                    Some(_) => {
                        self.child.take();
                        return Ok(());
                    }
                    None => {
                        if Instant::now() >= deadline {
                            if self.child.as_mut().expect("child checked").kill().is_err() {
                                return self.fail_close("terminal child termination unconfirmed");
                            }
                            let reap_deadline = Instant::now() + Duration::from_secs(3);
                            loop {
                                match self.child.as_mut().expect("child checked").try_wait() {
                                    Ok(Some(_)) => {
                                        self.child.take();
                                        return Ok(());
                                    }
                                    Ok(None) if Instant::now() < reap_deadline => {
                                        std::thread::sleep(Duration::from_millis(10));
                                    }
                                    Ok(None) => {
                                        return self.fail_close("terminal child reap unconfirmed");
                                    }
                                    Err(_) => {
                                        return self.fail_close("terminal child reap unconfirmed");
                                    }
                                }
                            }
                        }
                        std::thread::sleep(Duration::from_millis(10));
                    }
                }
            }
        }
        Ok(())
    }

    fn fail_close(&mut self, message: &str) -> Result<(), String> {
        self.shutdown.store(true, Ordering::SeqCst);
        let error = message.to_string();
        self.close_failure = Some(error.clone());
        Err(error)
    }

    /// Keep the child handle in its current pump owner until wait confirms
    /// terminal state. The host shutdown deadline bounds how long this owner
    /// may remain in the server task tree.
    pub(crate) fn retain_child_until_exit(&mut self) {
        while let Some(child) = self.child.as_mut() {
            match child.try_wait() {
                Ok(Some(_)) => {
                    self.child.take();
                    return;
                }
                Ok(None) | Err(_) => std::thread::sleep(Duration::from_millis(25)),
            }
        }
    }
}

impl Drop for NativeAttach {
    fn drop(&mut self) {
        if self.close().is_err() {
            self.retain_child_until_exit();
        }
    }
}
