//! Host terminal executor: service, native attach, identity broker calls.
//!
//! Port of `internal/host/terminal/service.go` (execution core), `native.go`,
//! `identity.go`, `identity_transfer.go`, the agent-hash helper from
//! `agent.go`, the admission predicates from `types.go`, and the daemon
//! identity dispatch logic from `internal/host/identity.go`.
//!
//! Out of scope (stay in Go): the websocket transport (`Write`, pumps,
//! `Handler`), `AgentExec` command construction for the Go client, and the
//! HTTP route shells (plain `pub` fns are exposed instead).
//!
//! Behavioral notes:
//!
//! * Podman argv, JSON wire bytes, the terminal frame protocol, timeouts and
//!   error strings match the Go implementation. Exit-code checks (`container
//!   exists`, `mountpoint`) parse the `exit status N` text the crate
//!   [`Executor`](crate::project::Executor) surface produces, since the
//!   trait is stringly typed.
//! * `stream_identity_harness` buffers the host tar producer through the
//!   executor instead of streaming pipe-to-pipe; error strings are unchanged.

use std::collections::HashMap;
use std::fs::File;
use std::io::{BufRead, BufReader, Write};
use std::os::unix::fs::{MetadataExt, PermissionsExt};
use std::os::unix::io::AsRawFd;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

use crate::account::AGENT_PROGRAM;
use crate::domain;
use crate::json::{self, Kind, Spec};
use crate::project::Executor;
use crate::sha256;

pub mod factory;

mod core;

pub use self::core::{
    err_denied, err_stale, err_uncertain, now_unix, valid_terminal_id, BROKER_RESPONSE_LIMIT,
    CREDENTIAL_LIMIT, ERR_DENIED, ERR_IDENTITY_UNCONFIRMED, ERR_INVALID_IDENTITY_OPERATION,
    ERR_NOT_FOUND, ERR_STALE, ERR_UNCERTAIN, FRAME_LIMIT, IDENTITY_LAUNCH_PATH, KIND_FACTORY,
    KIND_TERMINAL, PROVIDER_CODEX, PROVIDER_MUSE, SCOPE_MUSE_PROJECT, STREAM_LIMIT, TERMINAL_LIMIT,
};

mod text;

pub(in crate::terminal) use self::text::contains_crlf;
pub use self::text::{strict_b64_decode, terminal_dimensions, valid_terminal_name};

mod wire;

pub use self::wire::{credential_valid, json_valid, TerminalFrame, TerminalRequest, TerminalState};

mod time;

pub use self::time::parse_rfc3339;

mod lease;

pub use self::lease::{parse_string_i64, AcquireRequest, Binding, Delivery, Lease};

mod inspect;

pub use self::inspect::{
    exit_code_of, terminal_id_map, terminal_isolation, terminal_target_ready, TerminalInspection,
    TERMINAL_INSPECT,
};
pub(crate) use self::inspect::{parse_go_int, parse_go_uint};

mod agent;

pub use self::agent::{agent_argv, container_exists_argv, inspect_argv, EndIdentityHook, Service};

mod identity;

pub use self::identity::{identity_result, terminal_lease, IdentityRequest};

mod transfer;

pub use self::transfer::{tar_consumer_argv, tar_producer_argv};

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
    child: Option<std::process::Child>,
    stdin: Option<File>,
    reader: Option<BufReader<File>>,
    closed: bool,
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

/// `Service` stream registry: at most [`STREAM_LIMIT`] live streams;
/// `Close` cancels pending and live streams alike.
pub struct StreamTable {
    streams: HashMap<u64, Arc<AtomicBool>>,
    closed: bool,
    next: u64,
}

impl StreamTable {
    pub fn new() -> Self {
        StreamTable {
            streams: HashMap::new(),
            closed: false,
            next: 0,
        }
    }

    /// `Service.register`: admit one stream unless closed or full.
    pub fn register(&mut self) -> Option<(u64, Arc<AtomicBool>)> {
        if self.closed || self.streams.len() >= STREAM_LIMIT {
            return None;
        }
        let id = self.next;
        self.next += 1;
        let cancel = Arc::new(AtomicBool::new(false));
        self.streams.insert(id, cancel.clone());
        Some((id, cancel))
    }

    pub fn unregister(&mut self, id: u64) {
        self.streams.remove(&id);
    }

    /// `Service.Close`: cancel every registered stream.
    pub fn close(&mut self) {
        self.closed = true;
        for cancel in self.streams.values() {
            cancel.store(true, Ordering::SeqCst);
        }
    }

    pub fn len(&self) -> usize {
        self.streams.len()
    }

    pub fn is_empty(&self) -> bool {
        self.streams.is_empty()
    }

    pub fn is_closed(&self) -> bool {
        self.closed
    }
}

impl Default for StreamTable {
    fn default() -> Self {
        Self::new()
    }
}

// ---------- daemon identity dispatch (internal/host/identity.go) ----------

/// Trusted web-to-host launch input (`identity.TerminalStart`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TerminalStart {
    pub connection_id: String,
    pub project_id: String,
    pub actor_id: i64,
    pub login: String,
    pub scope: String,
    pub cols: i64,
    pub rows: i64,
}

const TERMINAL_START_SPECS: &[Spec] = &[
    Spec {
        name: "connection_id",
        kind: Kind::Str,
    },
    Spec {
        name: "project_id",
        kind: Kind::Str,
    },
    Spec {
        name: "actor_id",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "scope",
        kind: Kind::Str,
    },
    Spec {
        name: "cols",
        kind: Kind::Int,
    },
    Spec {
        name: "rows",
        kind: Kind::Int,
    },
];

impl TerminalStart {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        let m =
            json::bind_root(&v, "TerminalStart", TERMINAL_START_SPECS, false).map_err(|e| e.0)?;
        let actor_id = if m.contains("actor_id") {
            parse_string_i64(&m.take_string("actor_id"))
                .ok_or_else(|| "invalid actor_id".to_string())?
        } else {
            0
        };
        Ok(TerminalStart {
            connection_id: m.take_string("connection_id"),
            project_id: m.take_string("project_id"),
            actor_id,
            login: m.take_string("login"),
            scope: m.take_string("scope"),
            cols: m.take_i64("cols"),
            rows: m.take_i64("rows"),
        })
    }
}

/// `validIdentityRequest` over pre-parsed request fields.
pub fn valid_identity_request(
    method: &str,
    raw_query: &str,
    force_query: bool,
    raw_path: &str,
    origin_count: usize,
) -> bool {
    method == "POST"
        && raw_query.is_empty()
        && !force_query
        && raw_path.is_empty()
        && origin_count == 0
}

/// Broker custody surface the identity routes need.
pub trait IdentityBroker {
    fn acquire(&self, req: &AcquireRequest, deadline: Instant) -> Result<Lease, String>;
    fn register(
        &self,
        lease_id: &str,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<Delivery, String>;
    fn reconcile_lease(&self, lease_id: &str) -> Result<(), String>;
}

/// 16 bytes of kernel randomness, hex-encoded (`museID` / launch IDs).
pub fn rand_id() -> Result<String, String> {
    let mut bytes = [0u8; 16];
    let mut filled = 0;
    while filled < bytes.len() {
        let n = unsafe {
            libc::getrandom(
                bytes[filled..].as_mut_ptr() as *mut libc::c_void,
                bytes.len() - filled,
                0,
            )
        };
        if n < 0 {
            let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
            if errno == libc::EINTR {
                continue;
            }
            return Err(format!("kernel randomness unavailable: errno {errno}"));
        }
        filled += n as usize;
    }
    Ok(sha256::hex_lower(&bytes))
}

/// `Daemon.identityLaunch`: acquire, prepare, register, start. Every
/// failure after acquire reconciles the lease (reconcile errors ignored).
pub fn identity_launch<B: IdentityBroker, E: Executor>(
    broker: &B,
    service: &Service<E>,
    input: &TerminalStart,
    codex_harness_configured: bool,
    deadline: Instant,
) -> Result<Lease, String> {
    if !codex_harness_configured {
        return Err(err_denied());
    }
    let execution_id = rand_id()?;
    let now = now_unix();
    let lease = broker.acquire(
        &AcquireRequest {
            provider_id: PROVIDER_CODEX.to_string(),
            actor_id: input.actor_id,
            connection_id: input.connection_id.clone(),
            project_id: input.project_id.clone(),
            execution_id,
            kind: KIND_TERMINAL.to_string(),
            deadline_secs: now + 12 * 3600,
            ..Default::default()
        },
        deadline,
    )?;
    let binding = match service.prepare_identity(
        &lease,
        &input.login,
        &input.scope,
        input.cols,
        input.rows,
        deadline,
    ) {
        Ok(binding) => binding,
        Err(err) => {
            let _ = broker.reconcile_lease(&lease.id);
            return Err(err);
        }
    };
    let mut delivery = match broker.register(&lease.id, &binding, deadline) {
        Ok(delivery) => delivery,
        Err(err) => {
            let _ = broker.reconcile_lease(&lease.id);
            return Err(err);
        }
    };
    let outcome = service.identity("start", &delivery, deadline);
    // Zero the credential copy however the call ends.
    if let Some(credential) = delivery.credential.as_mut() {
        for byte in credential.iter_mut() {
            *byte = 0;
        }
    }
    if let Err(err) = outcome {
        let _ = broker.reconcile_lease(&lease.id);
        return Err(err);
    }
    Ok(delivery.lease)
}

/// `Daemon.identityOperation` dispatch target.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum IdentityRoute {
    Launch,
    Factory,
    Muse,
    Terminal,
}

/// Classify one identity operation: launch path, factory kind, live
/// muse-project binding, or plain terminal operation.
pub fn identity_route(path: &str, lease: &Lease, muse_available: bool) -> IdentityRoute {
    if path == IDENTITY_LAUNCH_PATH {
        return IdentityRoute::Launch;
    }
    if lease.kind == KIND_FACTORY {
        return IdentityRoute::Factory;
    }
    let muse_bound = lease.provider_id == PROVIDER_MUSE
        && lease
            .binding
            .as_ref()
            .map(|b| b.scope == SCOPE_MUSE_PROJECT)
            .unwrap_or(false);
    if muse_bound && muse_available {
        IdentityRoute::Muse
    } else {
        IdentityRoute::Terminal
    }
}

/// Operation word carried after `/identity/` (`validate`, `stop`, ...).
pub fn identity_action(path: &str) -> &str {
    path.strip_prefix("/identity/").unwrap_or(path)
}

#[cfg(test)]
mod tests;
