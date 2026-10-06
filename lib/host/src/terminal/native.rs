use std::fs::File;
use std::io::{BufRead, BufReader, Write};
use std::os::unix::fs::{MetadataExt, PermissionsExt};
use std::os::unix::io::AsRawFd;
use std::time::{Duration, Instant};

use crate::account::AGENT_PROGRAM;
use crate::domain;
use crate::project::Executor;
use crate::sha256;

use super::{
    agent_argv, now_unix, tar_consumer_argv, tar_producer_argv, Service, TerminalFrame,
    TerminalRequest, FRAME_LIMIT,
};

/// Lexical path cleaning matching Go `path/filepath.Clean` (Linux).
pub fn clean_path(path: &str) -> String {
    let rooted = path.starts_with('/');
    let mut parts: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if parts.pop().is_none() && !rooted {
                    parts.push("..");
                }
            }
            _ => parts.push(part),
        }
    }
    let mut out = parts.join("/");
    if rooted {
        out.insert(0, '/');
    }
    if out.is_empty() {
        out.push('.');
    }
    out
}

impl<E: Executor> Service<E> {
    /// `Service.streamIdentityHarness`: stage the verified harness bytes
    /// into the guest tmpfs. The Go implementation streams producer to
    /// consumer over a pipe; this port buffers the producer output through
    /// the executor (error strings unchanged).
    pub fn stream_identity_harness(
        &self,
        container: &str,
        path: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let producer = tar_producer_argv(&self.codex_harness);
        let refs: Vec<&str> = producer.iter().map(|s| s.as_str()).collect();
        let stream = match self.exec.run(&[], "/usr/bin/tar", &refs, deadline) {
            Ok(stream) => stream,
            Err(_) => return Err("codex harness stream unavailable".to_string()),
        };
        let consumer = tar_consumer_argv(container, path);
        let refs: Vec<&str> = consumer.iter().map(|s| s.as_str()).collect();
        match self.exec.run(&stream, "/usr/bin/podman", &refs, deadline) {
            Ok(_) => Ok(()),
            Err(_) => Err("codex harness staging failed".to_string()),
        }
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
    let info = std::fs::symlink_metadata(&path)
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
    let raw =
        std::fs::read(&path).map_err(|e| format!("project terminal agent unreadable: {e}"))?;
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
}

impl NativeAttach {
    /// `AttachNative`: start the fixed podman/agent attachment bridge.
    pub fn attach(container: &str, input: &TerminalRequest) -> Result<Self, String> {
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
        let stdin = child
            .stdin
            .take()
            .ok_or_else(|| "terminal stdin unavailable".to_string())?;
        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| "terminal stdout unavailable".to_string())?;
        use std::os::unix::io::FromRawFd;
        // `ChildStdin` has no timeout API; convert to `File` for deadlines.
        // SAFETY: the stdio handles are owned by us exactly once.
        let stdin_fd = stdin.as_raw_fd();
        let stdout_fd = stdout.as_raw_fd();
        std::mem::forget(stdin);
        std::mem::forget(stdout);
        let stdin = unsafe { File::from_raw_fd(stdin_fd) };
        let stdout = unsafe { File::from_raw_fd(stdout_fd) };
        Ok(NativeAttach {
            child: Some(child),
            stdin: Some(stdin),
            reader: Some(BufReader::new(stdout)),
            closed: false,
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
        // `File` has no write deadline; poll for writability like Go's
        // `SetWriteDeadline` on the stdin pipe.
        let mut pfd = libc::pollfd {
            fd: stdin.as_raw_fd(),
            events: libc::POLLOUT,
            revents: 0,
        };
        let ready = unsafe { libc::poll(&mut pfd, 1, 2000) };
        if ready == 0 {
            return Err("terminal input deadline exceeded".to_string());
        }
        if ready < 0 || pfd.revents & (libc::POLLERR | libc::POLLHUP | libc::POLLNVAL) != 0 {
            return Err("terminal input ended".to_string());
        }
        let mut body = f.encode().into_bytes();
        body.push(b'\n');
        stdin
            .write_all(&body)
            .map_err(|e| format!("terminal input ended: {e}"))?;
        Ok(())
    }

    /// Detach the stdout reader for lock-free output pumps (H01-F3).
    /// Teardown still funnels through [`Self::close`].
    pub fn take_reader(&mut self) -> Option<BufReader<File>> {
        self.reader.take()
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

    /// `nativeTerminal.Close`: stdin EOF, 3s grace, then kill. Idempotent.
    pub fn close(&mut self) {
        if self.closed {
            return;
        }
        self.closed = true;
        drop(self.stdin.take());
        drop(self.reader.take());
        if let Some(mut child) = self.child.take() {
            let deadline = Instant::now() + Duration::from_secs(3);
            loop {
                match child.try_wait() {
                    Ok(Some(_)) | Err(_) => break,
                    Ok(None) => {
                        if Instant::now() >= deadline {
                            let _ = child.kill();
                            // Reap the killed child (CODEX-H01-REAP-1):
                            // kill leaves a zombie without wait.
                            let _ = child.wait();
                            break;
                        }
                        std::thread::sleep(Duration::from_millis(10));
                    }
                }
            }
        }
    }
}

impl Drop for NativeAttach {
    fn drop(&mut self) {
        self.close();
    }
}

// ---------- private request admission + stream table (service.go) ----------

/// `validPrivateTerminalRequest` over pre-parsed request fields.
pub fn valid_private_terminal_request(
    method: &str,
    raw_query: &str,
    force_query: bool,
    raw_path: &str,
    origin_count: usize,
) -> bool {
    method == "GET"
        && raw_query.is_empty()
        && !force_query
        && raw_path.is_empty()
        && origin_count == 0
}
