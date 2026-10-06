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

// ---------- terminal service ----------

/// Broker custody callback behind `end` reconciliation.
pub type EndIdentityHook<'a> = &'a dyn Fn(i64, &str) -> Result<(), String>;

/// Privileged project-terminal executor (`terminal.Service`).
/// `EndIdentity` is passed per call so the service stays dependency-free.
pub struct Service<E> {
    pub exec: E,
    pub codex_harness: String,
    pub codex_harness_sha256: String,
    pub codex_harness_version: String,
    pub muse_harness: String,
    pub muse_harness_sha256: String,
    pub muse_harness_version: String,
}

/// `podman --remote=false inspect --format <terminalInspect> <name>`.
pub fn inspect_argv(name: &str) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "inspect".to_string(),
        "--format".to_string(),
        TERMINAL_INSPECT.to_string(),
        name.to_string(),
    ]
}

/// `podman --remote=false container exists <target>`.
pub fn container_exists_argv(target: &str) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "container".to_string(),
        "exists".to_string(),
        target.to_string(),
    ]
}

/// Fixed `podman exec` of the project terminal agent (`AgentExec` argv).
/// Python is gone: the agent is the native `project-terminal` binary.
pub fn agent_argv(container: &str, args: &[&str]) -> Vec<String> {
    let mut argv = vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
        container.to_string(),
        AGENT_PROGRAM.to_string(),
    ];
    argv.extend(args.iter().map(|s| s.to_string()));
    argv
}

impl<E: Executor> Service<E> {
    fn podman(&self, stdin: &[u8], args: &[&str], deadline: Instant) -> Result<Vec<u8>, String> {
        self.exec.run(stdin, "/usr/bin/podman", args, deadline)
    }

    fn podman_owned(
        &self,
        stdin: &[u8],
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let refs: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
        self.podman(stdin, &refs, deadline)
    }

    /// `Service.projectContainer`: fixed-name inspect with the full
    /// isolation gate. Native lifecycle may inspect stopped containers.
    pub fn project_container(
        &self,
        id: &str,
        require_running: bool,
        deadline: Instant,
    ) -> Result<String, String> {
        if !domain::valid_id(id) {
            return Err("invalid project".to_string());
        }
        let argv = inspect_argv(&format!("soda-{id}"));
        let data = match self.podman_owned(&[], &argv, deadline) {
            Ok(data) if data.len() <= 4096 => data,
            _ => return Err("terminal inspection unavailable".to_string()),
        };
        let v = TerminalInspection::decode(&data)
            .map_err(|_| "invalid terminal inspection".to_string())?;
        if !terminal_target_ready(&v, id, require_running) {
            return Err("terminal target not ready or isolated".to_string());
        }
        Ok(v.id)
    }

    /// `Service.factoryProjectContainer`: supervised factory runs bind the
    /// exact container incarnation instead of the 262144-mapping profile.
    pub fn factory_project_container(
        &self,
        id: &str,
        require_running: bool,
        deadline: Instant,
    ) -> Result<String, String> {
        if !domain::valid_id(id) {
            return Err(err_denied());
        }
        let argv = inspect_argv(&format!("soda-{id}"));
        let data = match self.podman_owned(&[], &argv, deadline) {
            Ok(data) if !data.is_empty() && data.len() <= 16384 => data,
            _ => return Err(err_stale()),
        };
        let v = TerminalInspection::decode(&data).map_err(|_| err_stale())?;
        if !domain::valid_container_id(&v.id)
            || v.project != id
            || v.privileged
            || v.userns != "private"
        {
            return Err(err_denied());
        }
        let Some(owner) = parse_go_int(&v.owner) else {
            return Err(err_denied());
        };
        if owner <= 0 {
            return Err(err_denied());
        }
        if require_running && !v.running {
            return Err(err_stale());
        }
        Ok(v.id)
    }
}

// ---------- identity broker calls (identity.go) ----------

/// Fixed agent request envelope (`identityRequest`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct IdentityRequest {
    pub action: String,
    pub delivery: Delivery,
    pub login: String,
    pub scope: String,
    pub cols: i64,
    pub rows: i64,
    pub source_hash: String,
    pub container: String,
    pub harness_sha256: String,
}

impl IdentityRequest {
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"action\":");
        out.push_str(&json::quote(&self.action));
        out.push_str(",\"delivery\":");
        out.push_str(&self.delivery.encode());
        if !self.login.is_empty() {
            out.push_str(",\"login\":");
            out.push_str(&json::quote(&self.login));
        }
        if !self.scope.is_empty() {
            out.push_str(",\"scope\":");
            out.push_str(&json::quote(&self.scope));
        }
        if self.cols != 0 {
            out.push_str(",\"cols\":");
            out.push_str(&self.cols.to_string());
        }
        if self.rows != 0 {
            out.push_str(",\"rows\":");
            out.push_str(&self.rows.to_string());
        }
        if !self.source_hash.is_empty() {
            out.push_str(",\"source_hash\":");
            out.push_str(&json::quote(&self.source_hash));
        }
        if !self.container.is_empty() {
            out.push_str(",\"container\":");
            out.push_str(&json::quote(&self.container));
        }
        if !self.harness_sha256.is_empty() {
            out.push_str(",\"harness_sha256\":");
            out.push_str(&json::quote(&self.harness_sha256));
        }
        out.push('}');
        out
    }
}

fn terminal_reservation(lease: &Lease) -> bool {
    lease.kind == KIND_TERMINAL
        && valid_terminal_id(&lease.execution_id)
        && domain::valid_id(&lease.project_id)
        && lease.actor_id > 0
        && !lease.id.is_empty()
        && lease.generation > 0
}

fn terminal_binding(lease: &Lease) -> bool {
    let Some(b) = &lease.binding else {
        return false;
    };
    b.kind == KIND_TERMINAL
        && b.id == lease.execution_id
        && domain::valid_container_id(&b.project)
        && domain::valid_login(&b.login)
        && b.login != "root"
        && b.generation == lease.generation
}

/// `terminalLease`: reservation shape, plus binding (live) or deadline
/// window (preparing).
pub fn terminal_lease(lease: &Lease, preparing: bool, now: i64) -> bool {
    if !terminal_reservation(lease) {
        return false;
    }
    if preparing {
        return lease.binding.is_none()
            && matches!(lease.deadline, Some((secs, _)) if secs > now && secs <= now + 12 * 3600);
    }
    terminal_binding(lease)
}

fn terminal_preparation(
    lease: &Lease,
    login: &str,
    scope: &str,
    cols: i64,
    rows: i64,
    now: i64,
) -> bool {
    terminal_lease(lease, true, now)
        && domain::valid_login(login)
        && login != "root"
        && domain::valid_container_id(scope)
        && terminal_dimensions(cols, rows)
}

fn terminal_prepared(
    result: &Delivery,
    lease: &Lease,
    container: &str,
    login: &str,
    now: i64,
) -> bool {
    if !terminal_lease(&result.lease, false, now) {
        return false;
    }
    result.lease.id == lease.id
        && result
            .lease
            .binding
            .as_ref()
            .map(|b| b.project == container && b.login == login)
            .unwrap_or(false)
        && result
            .credential
            .as_ref()
            .map(|c| c.is_empty())
            .unwrap_or(true)
}

impl<E: Executor> Service<E> {
    /// `Service.identityCall`: fixed `broker` agent invocation over stdin.
    pub fn identity_call(
        &self,
        container: &str,
        request: &IdentityRequest,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        let body = request.encode().into_bytes();
        let argv = agent_argv(container, &["broker"]);
        let out = match self.podman_owned(&body, &argv, deadline) {
            Ok(out) => out,
            Err(_) => return Err("managed Codex operation failed".to_string()),
        };
        if out.len() > BROKER_RESPONSE_LIMIT {
            return Err("invalid managed Codex response".to_string());
        }
        Delivery::decode(&out).map_err(|_| "invalid managed Codex response".to_string())
    }

    /// `Service.PrepareIdentity`: reserve one native terminal before the
    /// broker releases bytes.
    pub fn prepare_identity(
        &self,
        lease: &Lease,
        login: &str,
        scope: &str,
        cols: i64,
        rows: i64,
        deadline: Instant,
    ) -> Result<Binding, String> {
        let now = now_unix();
        if !terminal_preparation(lease, login, scope, cols, rows, now) {
            return Err(err_denied());
        }
        let container = self.project_container(&lease.project_id, true, deadline)?;
        let hash = agent_program_hash(0)?;
        let request = IdentityRequest {
            action: "prepare".to_string(),
            delivery: Delivery {
                lease: lease.clone(),
                credential: None,
            },
            login: login.to_string(),
            scope: scope.to_string(),
            cols,
            rows,
            source_hash: hash,
            container: container.clone(),
            ..Default::default()
        };
        let result = self.identity_call(&container, &request, deadline)?;
        if !terminal_prepared(&result, lease, &container, login, now_unix()) {
            return Err("managed terminal reservation differs".to_string());
        }
        result
            .lease
            .binding
            .clone()
            .ok_or_else(|| "managed terminal reservation differs".to_string())
    }

    fn identity_action(&self, action: &str, delivery: &Delivery) -> bool {
        match action {
            "start" => {
                delivery
                    .credential
                    .as_ref()
                    .map(|c| credential_valid(c))
                    .unwrap_or(false)
                    && self.codex_harness.starts_with('/')
                    && domain::valid_container_id(&self.codex_harness_sha256)
            }
            "validate" | "finish" | "stop" => delivery
                .credential
                .as_ref()
                .map(|c| c.is_empty())
                .unwrap_or(true),
            _ => false,
        }
    }

    /// `Service.Identity`: fixed model-session operations.
    pub fn identity(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        if !terminal_lease(&delivery.lease, false, now_unix()) {
            return Err(err_denied());
        }
        if !self.identity_action(action, delivery) {
            return Err(err_denied());
        }
        let (container, done) = self.identity_target(action, &delivery.lease, deadline)?;
        if done {
            return Ok(delivery.clone());
        }
        if action == "start" {
            self.identity_stage(&container, delivery, deadline)?;
        }
        let request = IdentityRequest {
            action: action.to_string(),
            delivery: delivery.clone(),
            harness_sha256: self.codex_harness_sha256.clone(),
            ..Default::default()
        };
        let result = self.identity_call(&container, &request, deadline)?;
        identity_result(action, delivery, &result)
    }

    /// `Service.managedEnd`: reconcile the managed lease behind an `end`
    /// before the launcher tears the terminal down.
    pub fn managed_end(
        &self,
        container: &str,
        request: &TerminalRequest,
        end_identity: Option<EndIdentityHook<'_>>,
        deadline: Instant,
    ) -> Result<(), String> {
        if request.action != "end" {
            return Ok(());
        }
        let lookup = IdentityRequest {
            action: "lookup".to_string(),
            delivery: Delivery {
                lease: Lease {
                    execution_id: request.id.clone(),
                    actor_id: request.identity,
                    ..Default::default()
                },
                credential: None,
            },
            login: request.login.clone(),
            ..Default::default()
        };
        let result = self.identity_call(container, &lookup, deadline)?;
        if result.lease.id.is_empty() {
            return Ok(());
        }
        let bound = result
            .lease
            .binding
            .as_ref()
            .map(|b| b.project == container && b.id == request.id)
            .unwrap_or(false);
        if !bound || result.lease.actor_id != request.identity {
            return Err(err_stale());
        }
        match end_identity {
            None => Err(err_denied()),
            Some(end) => end(request.identity, &result.lease.id),
        }
    }

    /// `Service.verifyIdentityHarness`: the staged host Codex bytes must
    /// match the pinned digest.
    pub fn verify_identity_harness(&self) -> Result<(), String> {
        let path = format!("{}/bin/codex", self.codex_harness.trim_end_matches('/'));
        // Go gates on `Lstat` regularity plus any exec bit before opening.
        let lstat = std::fs::symlink_metadata(&path)
            .map_err(|_| "verified Codex executable required".to_string())?;
        if !lstat.file_type().is_file() || lstat.permissions().mode() & 0o111 == 0 {
            return Err("verified Codex executable required".to_string());
        }
        let mut data = Vec::new();
        use std::io::Read;
        std::fs::File::open(&path)
            .map_err(|_| "verified Codex executable required".to_string())?
            .read_to_end(&mut data)
            .map_err(|e| format!("codex harness unreadable: {e}"))?;
        if sha256::hex_lower(&sha256::digest(&data)) != self.codex_harness_sha256 {
            return Err("codex harness digest differs".to_string());
        }
        Ok(())
    }

    fn identity_target(
        &self,
        action: &str,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<(String, bool), String> {
        if action != "stop" {
            let target = self.project_container(&lease.project_id, true, deadline)?;
            let bound = lease
                .binding
                .as_ref()
                .map(|b| b.project.clone())
                .unwrap_or_default();
            if target != bound {
                return Err(err_stale());
            }
            return Ok((target, false));
        }
        self.identity_stop_target(lease, deadline)
    }

    fn identity_stop_target(
        &self,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<(String, bool), String> {
        let target = lease
            .binding
            .as_ref()
            .map(|b| b.project.clone())
            .unwrap_or_default();
        let argv = container_exists_argv(&target);
        if let Err(err) = self.podman_owned(&[], &argv, deadline) {
            if exit_code_of(&err) == Some(1) {
                return Ok((target, true));
            }
            return Err(err_uncertain());
        }
        let argv = inspect_argv(&target);
        let data = match self.podman_owned(&[], &argv, deadline) {
            Ok(data) if data.len() <= 4096 => data,
            _ => return Err(err_uncertain()),
        };
        let v = TerminalInspection::decode(&data).map_err(|_| err_uncertain())?;
        if v.id != target || !terminal_target_ready(&v, &lease.project_id, false) {
            return Err(err_stale());
        }
        Ok((target, !v.running))
    }

    fn identity_stage(
        &self,
        container: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<(), String> {
        self.verify_identity_harness()?;
        let request = IdentityRequest {
            action: "stage".to_string(),
            delivery: delivery.clone(),
            ..Default::default()
        };
        self.identity_call(container, &request, deadline)?;
        let path = format!(
            "/run/soda-terminals/{}/model/harness",
            delivery.lease.execution_id
        );
        self.stream_identity_harness(container, &path, deadline)
    }
}

/// `identityResult`: the agent must echo the binding; only `finish`
/// returns credential bytes.
pub fn identity_result(
    action: &str,
    delivery: &Delivery,
    result: &Delivery,
) -> Result<Delivery, String> {
    // Go returns the partial result alongside the error; the daemon drops
    // it on every error path, so only the verdict is preserved here.
    if result.lease.id != delivery.lease.id || result.lease.binding.is_none() {
        return Err(err_stale());
    }
    if result.lease.binding != delivery.lease.binding {
        return Err(err_stale());
    }
    if action == "finish" {
        match &result.credential {
            Some(c) if credential_valid(c) => {}
            _ => return Err(err_uncertain()),
        }
    } else if result
        .credential
        .as_ref()
        .map(|c| !c.is_empty())
        .unwrap_or(false)
    {
        return Err("unexpected credential response".to_string());
    }
    Ok(result.clone())
}

// ---------- identity harness transfer (identity_transfer.go) ----------

/// Host tar producer argv: `tar --create --file=- --directory <harness> .`.
/// The harness path is cleaned exactly like Go's `filepath.Clean`.
pub fn tar_producer_argv(harness: &str) -> Vec<String> {
    vec![
        "--create".to_string(),
        "--file=-".to_string(),
        "--directory".to_string(),
        clean_path(harness),
        ".".to_string(),
    ]
}

/// Guest tar consumer argv: `podman ... exec --interactive <container> tar
/// --extract --file=- --directory <path> --no-same-owner --same-permissions`.
pub fn tar_consumer_argv(container: &str, path: &str) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
        container.to_string(),
        "/usr/bin/tar".to_string(),
        "--extract".to_string(),
        "--file=-".to_string(),
        "--directory".to_string(),
        path.to_string(),
        "--no-same-owner".to_string(),
        "--same-permissions".to_string(),
    ]
}

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
