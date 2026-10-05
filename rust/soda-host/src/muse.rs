//! Native Muse execution runtime and launch socket.
//!
//! Port of `internal/host/terminal/muse_types.go`,
//! `muse_linux.go`, `muse_execution_linux.go`, `muse_socket_linux.go`,
//! `muse_state_linux.go`, `muse_binary_linux.go`, the launch policy from
//! `internal/identity/launch.go` + `selection.go`, and the daemon Muse
//! wiring from `internal/host/muse.go` (authorization/selection over
//! broker connections plus launch-listener setup).
//!
//! Broker custody callbacks arrive through [`MuseHooks`]; unlike Go's
//! nullable func fields the hooks are mandatory, which collapses the
//! nil-hook denials into the type system (the daemon always wires all
//! six). `MuseRuntime::prepare_execution` returns a plain
//! [`MuseExecution`] record; the argv, delivery, control and finish steps
//! are separate methods so routes can stage them around process spawn.

use std::collections::HashMap;
use std::ffi::OsString;
use std::os::unix::io::{FromRawFd, RawFd};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Mutex;
use std::time::{Duration, Instant};

use crate::domain;
use crate::json::{self, Kind, Spec, Value};
use crate::project::Executor;
use crate::texec::{self, AcquireRequest, Binding, Delivery, Lease};

// ---------- launch wire types (internal/identity/launch.go) ----------

/// Kernel launch-socket path.
pub const MUSE_LAUNCH_SOCKET: &str = "/run/soda-muse-interface/launch.sock";
/// Setting named when several Muse connections need an explicit choice.
pub const MUSE_CONNECTION_SETTING: &str = "SODA_MUSE_CONNECTION";

/// Nested-container registration (`identity.NestedRegistration`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NestedRegistration {
    pub child_id: String,
    pub actor_id: i64,
    pub registration_id: String,
    pub muse: bool,
}

const NESTED_REGISTRATION_SPECS: &[Spec] = &[
    Spec {
        name: "child_id",
        kind: Kind::Str,
    },
    Spec {
        name: "actor_id",
        kind: Kind::Str,
    },
    Spec {
        name: "registration_id",
        kind: Kind::Str,
    },
    Spec {
        name: "muse",
        kind: Kind::Bool,
    },
];

/// Invocation preferences, never caller authority (`identity.LaunchRequest`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LaunchRequest {
    pub home: String,
    pub register: Option<NestedRegistration>,
    pub config_home: String,
    pub term: String,
    pub connection_id: String,
    pub cwd: String,
    pub args: Vec<String>,
    pub tty: bool,
    pub cols: u16,
    pub rows: u16,
}

const LAUNCH_REQUEST_SPECS: &[Spec] = &[
    Spec {
        name: "home",
        kind: Kind::Str,
    },
    Spec {
        name: "register",
        kind: Kind::OptObject {
            go_type: "*identity.NestedRegistration",
            struct_name: "NestedRegistration",
            specs: NESTED_REGISTRATION_SPECS,
        },
    },
    Spec {
        name: "config_home",
        kind: Kind::Str,
    },
    Spec {
        name: "term",
        kind: Kind::Str,
    },
    Spec {
        name: "connection_id",
        kind: Kind::Str,
    },
    Spec {
        name: "cwd",
        kind: Kind::Str,
    },
    Spec {
        name: "args",
        kind: Kind::StrList,
    },
    Spec {
        name: "tty",
        kind: Kind::Bool,
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

/// Bounded live shell control (`identity.LaunchControl`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LaunchControl {
    pub signal: i64,
    pub cols: u16,
    pub rows: u16,
}

const LAUNCH_CONTROL_SPECS: &[Spec] = &[
    Spec {
        name: "signal",
        kind: Kind::Int,
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

/// Shell outcome (`identity.LaunchExit`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LaunchExit {
    pub code: i32,
    pub error: String,
}

impl LaunchExit {
    pub fn denied() -> Self {
        LaunchExit {
            code: 1,
            error: "Muse launch denied".to_string(),
        }
    }

    pub fn cleanup_unconfirmed() -> Self {
        LaunchExit {
            code: 1,
            error: "Muse cleanup unconfirmed".to_string(),
        }
    }

    pub fn encode(&self) -> String {
        let mut out = format!("{{\"code\":{}", self.code);
        if !self.error.is_empty() {
            out.push_str(",\"error\":");
            out.push_str(&json::quote(&self.error));
        }
        out.push('}');
        out
    }

    /// Socket framing: `json.Encoder` appends one newline.
    pub fn encode_line(&self) -> String {
        format!("{}\n", self.encode())
    }
}

fn launch_text(value: &str, limit: usize) -> bool {
    value.len() <= limit && !value.contains('\0')
}

fn launch_absolute_path(value: &str, optional: bool) -> bool {
    if optional && value.is_empty() {
        return true;
    }
    value.starts_with('/') && launch_text(value, 4096)
}

fn launch_arguments_valid(args: &[String]) -> bool {
    let mut size = 0usize;
    for arg in args {
        size += arg.len();
        if !launch_text(arg, 32768) {
            return false;
        }
    }
    size <= 32768
}

impl LaunchRequest {
    /// `LaunchRequest.Validate()`.
    pub fn validate(&self) -> Result<(), String> {
        if self.register.is_some() {
            return self.registration_valid();
        }
        if !launch_absolute_path(&self.cwd, false)
            || !launch_absolute_path(&self.config_home, true)
            || !launch_absolute_path(&self.home, true)
        {
            return Err(texec::err_denied());
        }
        if !launch_text(&self.term, 128) || self.connection_id.len() > 128 || self.args.len() > 256
        {
            return Err(texec::err_denied());
        }
        if self.tty && (self.cols == 0 || self.rows == 0) {
            return Err(texec::err_denied());
        }
        if !launch_arguments_valid(&self.args) {
            return Err(texec::err_denied());
        }
        Ok(())
    }

    fn registration_valid(&self) -> Result<(), String> {
        let Some(register) = &self.register else {
            return Err(texec::err_denied());
        };
        if !self.cwd.is_empty()
            || !self.args.is_empty()
            || !self.connection_id.is_empty()
            || self.tty
            || register.actor_id <= 0
            || !register.muse
        {
            return Err(texec::err_denied());
        }
        Ok(())
    }

    /// Strict decode of one launch request.
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        Self::decode_value(&v)
    }

    pub fn decode_value(v: &Value) -> Result<Self, String> {
        let m =
            json::bind_root(v, "LaunchRequest", LAUNCH_REQUEST_SPECS, false).map_err(|e| e.0)?;
        let register = match m.take_opt_map("register") {
            None => None,
            Some(nested) => {
                let actor_id = if nested.contains("actor_id") {
                    texec::parse_string_i64(&nested.take_string("actor_id"))
                        .ok_or_else(|| "invalid actor_id".to_string())?
                } else {
                    0
                };
                Some(NestedRegistration {
                    child_id: nested.take_string("child_id"),
                    actor_id,
                    registration_id: nested.take_string("registration_id"),
                    muse: nested.take_bool("muse"),
                })
            }
        };
        let cols = m.take_i64("cols");
        let rows = m.take_i64("rows");
        if !(0..=u16::MAX as i64).contains(&cols) || !(0..=u16::MAX as i64).contains(&rows) {
            return Err("decode request: cols/rows out of range".to_string());
        }
        Ok(LaunchRequest {
            home: m.take_string("home"),
            register,
            config_home: m.take_string("config_home"),
            term: m.take_string("term"),
            connection_id: m.take_string("connection_id"),
            cwd: m.take_string("cwd"),
            args: m.take_str_list("args"),
            tty: m.take_bool("tty"),
            cols: cols as u16,
            rows: rows as u16,
        })
    }
}

impl LaunchControl {
    /// Strict decode of one control message (`DisallowUnknownFields`).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        let m =
            json::bind_root(&v, "LaunchControl", LAUNCH_CONTROL_SPECS, false).map_err(|e| e.0)?;
        let cols = m.take_i64("cols");
        let rows = m.take_i64("rows");
        if !(0..=u16::MAX as i64).contains(&cols) || !(0..=u16::MAX as i64).contains(&rows) {
            return Err("decode request: cols/rows out of range".to_string());
        }
        Ok(LaunchControl {
            signal: m.take_i64("signal"),
            cols: cols as u16,
            rows: rows as u16,
        })
    }
}

// ---------- Muse argument policy ----------

fn muse_auth_override(arg: &str) -> bool {
    matches!(
        arg.split('=').next(),
        Some("--provider") | Some("--base-url")
    )
}

fn muse_value_flag(flag: &str) -> bool {
    matches!(
        flag,
        "--model"
            | "--reasoning-effort"
            | "--agents"
            | "--preset"
            | "--image"
            | "--workspace"
            | "--worktree-base"
            | "--worktree-existing"
            | "--approval-mode"
            | "--permission-profile"
            | "--approval-judge"
            | "--sandbox-network"
            | "--echo-delay-ms"
    )
}

fn muse_positional(args: &[String]) -> &str {
    let mut skip = false;
    for arg in args {
        if skip {
            skip = false;
            continue;
        }
        if arg.starts_with('-') {
            skip = muse_value_flag(arg);
            continue;
        }
        return arg;
    }
    ""
}

fn muse_provider_arguments(args: &[String]) -> Vec<String> {
    if args.is_empty() {
        return vec!["--provider".to_string(), "meta".to_string()];
    }
    match args[0].as_str() {
        "exec" | "resume" | "serve" => {
            let mut out = vec![
                args[0].clone(),
                "--provider".to_string(),
                "meta".to_string(),
            ];
            out.extend(args[1..].iter().cloned());
            out
        }
        "config" | "export" | "trace" | "skills" | "sandbox" | "schema" | "session-message"
        | "mcp" | "init" => args.to_vec(),
        _ => {
            let mut out = vec!["--provider".to_string(), "meta".to_string()];
            out.extend(args.iter().cloned());
            out
        }
    }
}

/// `identity.MuseArguments`: ordinary invocation bytes under the
/// subscription provider; auth subcommands and overrides denied.
pub fn muse_arguments(args: &[String]) -> Result<Vec<String>, String> {
    for arg in args {
        if muse_auth_override(arg) {
            return Err(texec::err_denied());
        }
    }
    match muse_positional(args) {
        "auth" | "login" | "logout" => Err(texec::err_denied()),
        _ => Ok(muse_provider_arguments(args)),
    }
}

// ---------- connection directory (internal/host/muse.go + selection.go) ----------

/// Minimal broker connection record the Muse directory needs.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MuseConnection {
    pub id: String,
    pub provider_id: String,
    pub state: String,
}

/// Daemon `authorizeNested`/muse-authorize closure over `Available`:
/// any ready Muse connection authorizes.
pub fn muse_connection_authorized(connections: &[MuseConnection]) -> Result<(), String> {
    for connection in connections {
        if connection.provider_id == texec::PROVIDER_MUSE && connection.state == "ready" {
            return Ok(());
        }
    }
    Err(texec::err_denied())
}

/// `identity.SelectMuseConnection`: the single ready Muse connection,
/// or the explicit choice; several require `SODA_MUSE_CONNECTION`.
pub fn select_muse_connection(
    connections: &[MuseConnection],
    selected: &str,
) -> Result<String, String> {
    let mut matches = Vec::new();
    for connection in connections {
        if connection.provider_id != texec::PROVIDER_MUSE || connection.state != "ready" {
            continue;
        }
        if !selected.is_empty() && connection.id != selected {
            continue;
        }
        matches.push(connection.id.clone());
    }
    if matches.len() == 1 {
        return Ok(matches.pop().unwrap());
    }
    if matches.len() > 1 {
        return Err(format!(
            "choose an authorized {} connection with {MUSE_CONNECTION_SETTING}",
            texec::PROVIDER_MUSE
        ));
    }
    Err(texec::err_denied())
}

// ---------- runtime types ----------

/// Kernel-attested socket identity. PIDFD pins the original caller.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct MusePeer {
    pub pid: i32,
    pub uid: u32,
    pub gid: u32,
    pub pidfd: RawFd,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MuseCaller {
    pub project: String,
    pub container: String,
    pub login: String,
    pub home: String,
    pub namespace: String,
    pub child: String,
    pub registration: String,
    pub actor: i64,
    pub uid: i64,
    pub gid: i64,
    pub project_pid: i32,
    pub nested_pid: i32,
    pub muse_allowed: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MuseNested {
    pub parent: String,
    pub project: String,
    pub child: String,
    pub namespace: String,
    pub registration: String,
    pub actor: i64,
    pub pid: i32,
    pub muse: bool,
}

/// One prepared native execution: resolved caller, sanitized request,
/// broker lease/binding, credential path and unit name.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MuseExecution {
    pub caller: MuseCaller,
    pub request: LaunchRequest,
    pub lease: Lease,
    pub binding: Binding,
    pub path: String,
    pub unit: String,
}

/// Broker custody surface the Muse runtime needs.
pub trait MuseHooks {
    fn acquire(&self, req: &AcquireRequest, deadline: Instant) -> Result<Lease, String>;
    fn attach(
        &self,
        lease_id: &str,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<Delivery, String>;
    fn end(&self, actor: i64, lease_id: &str, deadline: Instant) -> Result<(), String>;
    fn authorize(&self, actor: i64, project: &str, deadline: Instant) -> Result<(), String>;
    fn nested_authorize(&self, actor: i64, project: &str, deadline: Instant) -> Result<(), String>;
    fn select(
        &self,
        actor: i64,
        project: &str,
        selected: &str,
        deadline: Instant,
    ) -> Result<String, String>;
}

/// Native caller resolution plus one supervised execution per launch.
pub struct MuseRuntime<E, H> {
    pub exec: E,
    pub hooks: H,
    pub binary_version: String,
    pub binary_sha256: String,
    nested: Mutex<HashMap<String, MuseNested>>,
}

impl<E, H> MuseRuntime<E, H> {
    pub fn new(exec: E, hooks: H, binary_version: String, binary_sha256: String) -> Self {
        MuseRuntime {
            exec,
            hooks,
            binary_version,
            binary_sha256,
            nested: Mutex::new(HashMap::new()),
        }
    }
}

// ---------- pure resolution helpers (muse_linux.go) ----------

/// `museInspect`: exact podman `--format` template.
pub const MUSE_INSPECT: &str = "{\"id\":{{json .ID}},\"pid\":{{json .State.Pid}},\"project\":{{json (index .Config.Labels \"org.soda.project\")}},\"running\":{{json .State.Running}},\"privileged\":{{json .HostConfig.Privileged}},\"userns\":{{json .HostConfig.UsernsMode}}}";
/// Nested-child inspect template.
pub const MUSE_CHILD_INSPECT: &str =
    "{\"id\":{{json .ID}},\"pid\":{{json .State.Pid}},\"running\":{{json .State.Running}}}";

/// `museProjectCgroup`: the complete OCI scope component from
/// `/proc/<pid>/cgroup`, never a substring guess.
pub fn muse_project_cgroup(value: &str) -> Result<String, String> {
    for part in value.trim().split('/') {
        if let Some(id) = part
            .strip_prefix("libpod-")
            .and_then(|s| s.strip_suffix(".scope"))
        {
            if domain::valid_container_id(id) {
                return Ok(id.to_string());
            }
        }
    }
    Err(texec::err_denied())
}

/// `museMappedUID`: inside-UID for a host UID from a uid/gid map.
pub fn muse_mapped_uid(data: &str, uid: u32) -> Result<i64, String> {
    for line in data.split('\n') {
        let p: Vec<&str> = line.split_whitespace().collect();
        if p.len() != 3 {
            continue;
        }
        let (Some(inside), Some(outside), Some(count)) = (
            texec::parse_go_uint(p[0], 32),
            texec::parse_go_uint(p[1], 32),
            texec::parse_go_uint(p[2], 32),
        ) else {
            continue;
        };
        if outside > 0 && u64::from(uid) >= outside && u64::from(uid) < outside + count {
            return Ok((inside + u64::from(uid) - outside) as i64);
        }
    }
    Err(texec::err_denied())
}

/// `musePasswdValid`: seven-field passwd row for the mapped UID.
pub fn muse_passwd_valid(account: &[String], uid: i64) -> bool {
    if account.len() != 7 {
        return false;
    }
    domain::valid_login(&account[0])
        && account[0] != "root"
        && account[2] == uid.to_string()
        && account[5].starts_with('/')
}

/// `museAccountNode`: one root-owned account-path node.
pub fn muse_account_node(line: &str, marker: bool) -> bool {
    let parts: Vec<&str> = line.splitn(4, ':').collect();
    if parts.len() != 4 || parts[0] != "0" || parts[1] != "0" {
        return false;
    }
    let Ok(mode) = u32::from_str_radix(parts[2], 8) else {
        return false;
    };
    if mode & 0o022 != 0 {
        return false;
    }
    if marker {
        return parts[3] == "regular file" && mode == 0o600;
    }
    parts[3] == "directory"
}

/// `museAccountModes`: five account-path nodes, marker last.
pub fn muse_account_modes(body: &str) -> bool {
    let lines: Vec<&str> = body.trim().split('\n').collect();
    if lines.len() != 5 {
        return false;
    }
    for (i, line) in lines.iter().enumerate() {
        if !muse_account_node(line, i == 4) {
            return false;
        }
    }
    true
}

/// `museRegistrationValid`: registration shape (Muse launches only).
pub fn muse_registration_valid(input: &NestedRegistration) -> bool {
    domain::valid_container_id(&input.child_id)
        && input.actor_id > 0
        && texec::valid_terminal_id(&input.registration_id)
        && input.muse
}

/// `museChildPID`: running nested-child PID from its inspect output.
pub fn muse_child_pid(body: &[u8], failed: bool, id: &str) -> Result<i32, String> {
    if failed || body.len() > 4096 {
        return Err(texec::err_denied());
    }
    let v = json::decode_strict(body).map_err(|_| texec::err_denied())?;
    let Some(fields) = v.as_object() else {
        return Err(texec::err_denied());
    };
    let get = |name: &str| fields.iter().find(|(k, _)| k == name).map(|(_, v)| v);
    let (Some(Value::Str(got)), Some(pid), Some(Value::Bool(running))) =
        (get("id"), get("pid"), get("running"))
    else {
        return Err(texec::err_denied());
    };
    let pid = match pid {
        Value::Number(lit) => texec::parse_go_int(lit).unwrap_or(0),
        _ => 0,
    };
    if got != id || !running || pid <= 0 || pid > i32::MAX as i64 {
        return Err(texec::err_denied());
    }
    Ok(pid as i32)
}

/// `museReadonlyMount`: the required read-only nested credential mount.
/// Decoded tolerantly (`encoding/json`, not strict) like Go.
pub fn muse_readonly_mount(
    body: &[u8],
    failed: bool,
    source: &str,
    destination: &str,
) -> Result<(), String> {
    if failed || body.len() > 32768 {
        return Err(texec::err_denied());
    }
    let v = json::decode_tolerant(body).map_err(|_| texec::err_denied())?;
    let Some(mounts) = v.as_array() else {
        return Err(texec::err_denied());
    };
    for mount in mounts {
        let Some(fields) = mount.as_object() else {
            continue;
        };
        let get = |name: &str| fields.iter().find(|(k, _)| k == name).map(|(_, v)| v);
        match (get("Source"), get("Destination"), get("RW")) {
            (Some(Value::Str(s)), Some(Value::Str(d)), Some(Value::Bool(rw)))
                if s == source && d == destination && !rw =>
            {
                return Ok(())
            }
            _ => {}
        }
    }
    Err(texec::err_denied())
}

/// `museELF`: 64-bit little-endian ELF for the host architecture.
pub fn muse_elf(header: &[u8], arch: &str) -> Result<(), String> {
    if header.len() != 64
        || header[0..4] != [0x7f, b'E', b'L', b'F']
        || header[4] != 2
        || header[5] != 1
    {
        return Err(texec::err_denied());
    }
    let machine = u16::from_le_bytes([header[18], header[19]]);
    if arch == "amd64" && machine == 62 {
        return Ok(());
    }
    if arch == "arm64" && machine == 183 {
        return Ok(());
    }
    Err(texec::err_denied())
}

/// Host `GOARCH` spelled the Go way.
pub fn host_go_arch() -> &'static str {
    match std::env::consts::ARCH {
        "x86_64" => "amd64",
        "aarch64" => "arm64",
        _ => "unknown",
    }
}

/// `museSignal`: forwardable unit signals.
pub fn muse_signal(sig: i64) -> bool {
    if !(1..=64).contains(&sig) {
        return false;
    }
    matches!(
        sig as i32,
        libc::SIGINT
            | libc::SIGTERM
            | libc::SIGHUP
            | libc::SIGQUIT
            | libc::SIGTSTP
            | libc::SIGCONT
            | libc::SIGUSR1
            | libc::SIGUSR2
    )
}

/// `museProjectCredentialRoot`: exact `/run/soda-muse/` execution path.
pub fn muse_project_credential_root(binding: &Binding) -> bool {
    if !domain::valid_container_id(&binding.project)
        || texec::clean_path(&binding.credential_root) != binding.credential_root
        || !binding.credential_root.starts_with("/run/soda-muse/")
    {
        return false;
    }
    let parts: Vec<&str> = binding.credential_root["/run/soda-muse/".len()..]
        .split('/')
        .collect();
    if parts.len() == 1 {
        return parts[0] == binding.id;
    }
    parts.len() == 3
        && parts[0] == "nested"
        && texec::valid_terminal_id(parts[1])
        && parts[2] == binding.id
}

/// `museDeliveryValid`: well-formed Muse delivery for an operation.
pub fn muse_delivery_valid(delivery: &Delivery) -> bool {
    let Some(binding) = &delivery.lease.binding else {
        return false;
    };
    delivery.lease.provider_id == texec::PROVIDER_MUSE
        && texec::valid_terminal_id(&delivery.lease.execution_id)
        && binding.id == delivery.lease.execution_id
        && texec::valid_terminal_id(&binding.invocation_id)
}

/// `museHostEnvironment`: operator environment minus the META key.
pub fn muse_host_environment() -> Vec<(OsString, OsString)> {
    std::env::vars_os()
        .filter(|(name, _)| name.as_os_str() != std::ffi::OsStr::new("META_API_KEY"))
        .collect()
}

// ---------- argv builders ----------

/// `museCommand` argv (after the `/usr/bin/podman` argv0): the fixed
/// systemd-run boundary plus the pinned dispatcher. Golden-pinned.
pub fn muse_command_argv(
    caller: &MuseCaller,
    request: &LaunchRequest,
    unit: &str,
    path: &str,
) -> Vec<String> {
    let home = if request.home.is_empty() {
        caller.home.clone()
    } else {
        request.home.clone()
    };
    let mut config_path = path.to_string();
    if !caller.child.is_empty() {
        let prefix = format!("/run/soda-muse/nested/{}/", caller.registration);
        config_path = format!(
            "/run/soda-muse/credentials/{}",
            path.strip_prefix(&prefix).unwrap_or(path)
        );
    }
    let mut args = vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
    ];
    if request.tty {
        args.push("--tty".to_string());
    }
    args.extend([
        caller.container.clone(),
        "/usr/bin/systemd-run".to_string(),
        "--quiet".to_string(),
        "--wait".to_string(),
        "--collect".to_string(),
        "--service-type=exec".to_string(),
        format!("--unit={unit}"),
        "--property=KillMode=control-group".to_string(),
        "--property=TimeoutStopSec=10".to_string(),
        "--property=RuntimeMaxSec=43200".to_string(),
        "--property=UMask=0077".to_string(),
        format!("--setenv=HOME={home}"),
        format!("--setenv=USER={}", caller.login),
        format!("--setenv=LOGNAME={}", caller.login),
        format!("--setenv=TERM={}", request.term),
        format!("--setenv=XDG_CONFIG_HOME={path}/config"),
        format!("--setenv=XDG_STATE_HOME={path}/state"),
        format!("--setenv=XDG_CACHE_HOME={path}/cache"),
        "--setenv=TBH_CREDENTIAL_BACKEND=file".to_string(),
        "--setenv=PATH=/usr/local/bin:/usr/bin:/bin".to_string(),
    ]);
    if caller.child.is_empty() {
        args.extend([
            format!("--uid={}", caller.login),
            format!("--gid={}", caller.gid),
            format!("--working-directory={}", request.cwd),
            format!("--property=ReadOnlyPaths={path}/auth.json"),
            format!("--property=BindReadOnlyPaths={path}/auth.json:{path}/config/muse/auth.json"),
        ]);
    }
    args.push(if request.tty {
        "--pty".to_string()
    } else {
        "--pipe".to_string()
    });
    args.push("--".to_string());
    if !caller.child.is_empty() {
        args.extend([
            "/usr/bin/nsenter".to_string(),
            format!("--target={}", caller.nested_pid),
            "--mount".to_string(),
            "--pid".to_string(),
            "--uts".to_string(),
            "--ipc".to_string(),
            "--net".to_string(),
            "--root".to_string(),
            format!("--setuid={}", caller.uid),
            format!("--setgid={}", caller.gid),
            "--".to_string(),
        ]);
    }
    args.extend([
        "/usr/local/bin/muse".to_string(),
        "--soda-exec".to_string(),
        config_path,
        request.cwd.clone(),
    ]);
    args.extend(request.args.iter().cloned());
    args
}

/// Guest `systemctl show ActiveState` argv tail (after container).
pub fn unit_active_argv(unit: &str) -> Vec<String> {
    vec![
        "/usr/bin/systemctl".to_string(),
        "show".to_string(),
        "--property=ActiveState".to_string(),
        "--value".to_string(),
        unit.to_string(),
    ]
}

/// Guest `systemctl show InvocationID` argv tail.
pub fn unit_invocation_argv(unit: &str) -> Vec<String> {
    vec![
        "/usr/bin/systemctl".to_string(),
        "show".to_string(),
        "--property=InvocationID".to_string(),
        "--value".to_string(),
        unit.to_string(),
    ]
}

// ---------- runtime ----------

/// Strict-decoded Muse container inspection.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
struct MuseInspection {
    id: String,
    project: String,
    running: bool,
    privileged: bool,
    userns: String,
}

const MUSE_INSPECTION_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "pid",
        kind: Kind::Int,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "running",
        kind: Kind::Bool,
    },
    Spec {
        name: "privileged",
        kind: Kind::Bool,
    },
    Spec {
        name: "userns",
        kind: Kind::Str,
    },
];

fn muse_peer_alive(peer: &MusePeer) -> bool {
    let mut fds = [libc::pollfd {
        fd: peer.pidfd,
        events: libc::POLLIN,
        revents: 0,
    }];
    // SAFETY: valid one-element pollfd array.
    let n = unsafe { libc::poll(fds.as_mut_ptr(), 1, 0) };
    n == 0
}

fn sleep_until(target: Instant, deadline: Instant) -> Result<(), ()> {
    loop {
        let now = Instant::now();
        if now >= deadline {
            return Err(());
        }
        if now >= target {
            return Ok(());
        }
        let slice = (target - now)
            .min(deadline - now)
            .min(Duration::from_millis(10));
        std::thread::sleep(slice);
    }
}

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
    fn podman(&self, stdin: &[u8], args: &[String], deadline: Instant) -> Result<Vec<u8>, String> {
        let mut full = vec!["--remote=false".to_string()];
        full.extend(args.iter().cloned());
        let refs: Vec<&str> = full.iter().map(|s| s.as_str()).collect();
        self.exec.run(stdin, "/usr/bin/podman", &refs, deadline)
    }

    fn guest(
        &self,
        container: &str,
        stdin: &[u8],
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let mut full = vec![
            "exec".to_string(),
            "--interactive".to_string(),
            container.to_string(),
        ];
        full.extend(args.iter().cloned());
        self.podman(stdin, &full, deadline)
    }

    fn guest_refs(
        &self,
        container: &str,
        stdin: &[u8],
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.guest(
            container,
            stdin,
            &args.iter().map(|s| s.to_string()).collect::<Vec<_>>(),
            deadline,
        )
    }

    /// `MuseRuntime.inspect`: running, labeled, unprivileged container.
    pub fn inspect(&self, container: &str, deadline: Instant) -> Result<MuseCaller, String> {
        let body = self.podman(
            &[],
            &[
                "inspect".to_string(),
                "--format".to_string(),
                MUSE_INSPECT.to_string(),
                container.to_string(),
            ],
            deadline,
        );
        let mut out = MuseCaller::default();
        let Ok(body) = body else {
            return Err(texec::err_denied());
        };
        if body.len() > 4096 {
            return Err(texec::err_denied());
        }
        let v = json::decode_strict(&body).map_err(|_| texec::err_denied())?;
        let m = json::bind_root(&v, "museInspection", MUSE_INSPECTION_SPECS, false)
            .map_err(|_| texec::err_denied())?;
        let raw_pid = m.take_i64("pid");
        let inspection = MuseInspection {
            id: m.take_string("id"),
            project: m.take_string("project"),
            running: m.take_bool("running"),
            privileged: m.take_bool("privileged"),
            userns: m.take_string("userns"),
        };
        if inspection.id != container
            || !inspection.running
            || raw_pid <= 0
            || raw_pid > i32::MAX as i64
            || !domain::valid_id(&inspection.project)
            || inspection.privileged
        {
            return Err(texec::err_denied());
        }
        out.container = container.to_string();
        out.project = inspection.project;
        out.project_pid = raw_pid as i32;
        let _ = inspection.userns;
        Ok(out)
    }

    /// `museKernelCaller` + project binding + registration switch.
    pub fn resolve_project(
        &self,
        peer: &MusePeer,
        deadline: Instant,
    ) -> Result<MuseCaller, String> {
        let mut caller = self.kernel_caller(peer)?;
        let inspected = self.inspect(&caller.container, deadline)?;
        caller.project = inspected.project;
        caller.project_pid = inspected.project_pid;
        // Re-verify through the terminal isolation gate.
        let svc = texec::Service {
            exec: &self.exec,
            codex_harness: String::new(),
            codex_harness_sha256: String::new(),
            codex_harness_version: String::new(),
            muse_harness: String::new(),
            muse_harness_sha256: String::new(),
            muse_harness_version: String::new(),
        };
        match svc.project_container(&caller.project, true, deadline) {
            Ok(verified) if verified == caller.container => {}
            _ => return Err(texec::err_denied()),
        }
        let project_ns = std::fs::read_link(format!("/proc/{}/ns/pid", caller.project_pid))
            .map_err(|_| texec::err_denied())?;
        if caller.namespace != project_ns.to_string_lossy().into_owned() {
            return self.registered_caller(caller, peer, deadline);
        }
        if !muse_peer_alive(peer) {
            return Err(texec::err_denied());
        }
        Ok(caller)
    }

    fn kernel_caller(&self, peer: &MusePeer) -> Result<MuseCaller, String> {
        let mut caller = MuseCaller::default();
        if !muse_peer_alive(peer) {
            return Err(texec::err_denied());
        }
        let proc = format!("/proc/{}", peer.pid);
        let cg =
            std::fs::read_to_string(format!("{proc}/cgroup")).map_err(|_| texec::err_denied())?;
        caller.container = muse_project_cgroup(&cg)?;
        let mappings =
            std::fs::read_to_string(format!("{proc}/uid_map")).map_err(|_| texec::err_denied())?;
        caller.uid = muse_mapped_uid(&mappings, peer.uid)?;
        caller.namespace = std::fs::read_link(format!("{proc}/ns/pid"))
            .map_err(|_| texec::err_denied())?
            .to_string_lossy()
            .into_owned();
        Ok(caller)
    }

    fn registered_caller(
        &self,
        mut caller: MuseCaller,
        peer: &MusePeer,
        deadline: Instant,
    ) -> Result<MuseCaller, String> {
        let registered = self.nested.lock().unwrap().get(&caller.namespace).cloned();
        let Some(registered) = registered else {
            return Err(texec::err_denied());
        };
        if registered.parent != caller.container || registered.project != caller.project {
            return Err(texec::err_denied());
        }
        self.validate_nested(&registered, deadline)?;
        caller.actor = registered.actor;
        caller.child = registered.child;
        caller.registration = registered.registration;
        caller.nested_pid = registered.pid;
        caller.muse_allowed = registered.muse;
        if !muse_peer_alive(peer) {
            return Err(texec::err_denied());
        }
        Ok(caller)
    }

    /// `MuseRuntime.resolve`: full caller authority resolution.
    pub fn resolve(&self, peer: &MusePeer, deadline: Instant) -> Result<MuseCaller, String> {
        match self.resolve_inner(peer, deadline) {
            Ok(caller) => Ok(caller),
            Err(_) => Err(texec::err_denied()),
        }
    }

    fn resolve_inner(&self, peer: &MusePeer, deadline: Instant) -> Result<MuseCaller, String> {
        let caller = self.resolve_project(peer, deadline)?;
        if !caller.child.is_empty() {
            if !caller.muse_allowed {
                return Err(texec::err_denied());
            }
            return self.nested_caller(caller, peer, deadline);
        }
        if caller.uid == 0 {
            return Err(texec::err_denied());
        }
        let caller = self.project_account(caller, deadline)?;
        let mut caller = caller;
        caller.actor = self.project_actor(&caller, deadline)?;
        if !self.authorized_caller(&caller, peer, deadline) {
            return Err(texec::err_denied());
        }
        Ok(caller)
    }

    fn project_account(
        &self,
        mut caller: MuseCaller,
        deadline: Instant,
    ) -> Result<MuseCaller, String> {
        let body = self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/getent", "passwd", &caller.uid.to_string()],
            deadline,
        );
        let Ok(body) = body else {
            return Err(texec::err_denied());
        };
        if body.len() > 4096 {
            return Err(texec::err_denied());
        }
        let account: Vec<String> = String::from_utf8_lossy(&body)
            .trim()
            .split(':')
            .map(|s| s.to_string())
            .collect();
        if !muse_passwd_valid(&account, caller.uid) {
            return Err(texec::err_denied());
        }
        caller.login = account[0].clone();
        caller.home = account[5].clone();
        caller.gid = texec::parse_go_int(&account[3]).ok_or_else(texec::err_denied)?;
        Ok(caller)
    }

    fn project_actor(&self, caller: &MuseCaller, deadline: Instant) -> Result<i64, String> {
        let marker = format!("/var/lib/soda/accounts/{}", caller.login);
        let body = self.guest_refs(
            &caller.container,
            &[],
            &[
                "/usr/bin/stat",
                "--format=%u:%g:%a:%F",
                "/var",
                "/var/lib",
                "/var/lib/soda",
                "/var/lib/soda/accounts",
                &marker,
            ],
            deadline,
        );
        let Ok(body) = body else {
            return Err(texec::err_denied());
        };
        if !muse_account_modes(&String::from_utf8_lossy(&body)) {
            return Err(texec::err_denied());
        }
        let body = self.guest_refs(&caller.container, &[], &["/usr/bin/cat", &marker], deadline);
        let Ok(body) = body else {
            return Err(texec::err_denied());
        };
        if body.len() > 64 {
            return Err(texec::err_denied());
        }
        match texec::parse_go_int(String::from_utf8_lossy(&body).trim()) {
            Some(actor) if actor > 0 => Ok(actor),
            _ => Err(texec::err_denied()),
        }
    }

    fn authorized_caller(&self, caller: &MuseCaller, peer: &MusePeer, deadline: Instant) -> bool {
        self.hooks
            .authorize(caller.actor, &caller.project, deadline)
            .is_ok()
            && muse_peer_alive(peer)
    }

    /// `MuseRuntime.RegisterNested`: bind a current project-root registration.
    pub fn register_nested(
        &self,
        peer: &MusePeer,
        input: &NestedRegistration,
        deadline: Instant,
    ) -> Result<(), String> {
        if !muse_registration_valid(input) {
            return Err(texec::err_denied());
        }
        let caller = self.resolve_project(peer, deadline)?;
        if !self.registration_authority(&caller, input.actor_id, deadline) {
            return Err(texec::err_denied());
        }
        let record = self.registered_child(&caller, input, deadline)?;
        self.validate_nested(&record, deadline)?;
        if !muse_peer_alive(peer) {
            return Err(texec::err_denied());
        }
        self.nested
            .lock()
            .unwrap()
            .insert(record.namespace.clone(), record);
        Ok(())
    }

    fn registration_authority(&self, caller: &MuseCaller, actor: i64, deadline: Instant) -> bool {
        if caller.uid != 0 {
            return false;
        }
        if self
            .hooks
            .nested_authorize(actor, &caller.project, deadline)
            .is_err()
        {
            return false;
        }
        let ns = match std::fs::read_link(format!("/proc/{}/ns/pid", caller.project_pid)) {
            Ok(ns) => ns.to_string_lossy().into_owned(),
            Err(_) => return false,
        };
        if caller.namespace != ns {
            return false;
        }
        self.actor_account(&caller.container, actor, deadline)
            .is_ok()
    }

    fn registered_child(
        &self,
        caller: &MuseCaller,
        input: &NestedRegistration,
        deadline: Instant,
    ) -> Result<MuseNested, String> {
        let body = self.guest_refs(
            &caller.container,
            &[],
            &[
                "/usr/bin/podman",
                "--remote=false",
                "inspect",
                "--format",
                MUSE_CHILD_INSPECT,
                &input.child_id,
            ],
            deadline,
        );
        let (body, failed) = match body {
            Ok(body) => (body, false),
            Err(_) => (Vec::new(), true),
        };
        let pid = muse_child_pid(&body, failed, &input.child_id)?;
        let body = self
            .guest_refs(
                &caller.container,
                &[],
                &["/usr/bin/readlink", &format!("/proc/{pid}/ns/pid")],
                deadline,
            )
            .map_err(|_| texec::err_denied())?;
        let ns = String::from_utf8_lossy(&body).trim().to_string();
        if !ns.starts_with("pid:[") || ns == caller.namespace {
            return Err(texec::err_denied());
        }
        Ok(MuseNested {
            parent: caller.container.clone(),
            project: caller.project.clone(),
            child: input.child_id.clone(),
            namespace: ns,
            registration: input.registration_id.clone(),
            actor: input.actor_id,
            pid,
            muse: input.muse,
        })
    }

    fn validate_nested(&self, record: &MuseNested, deadline: Instant) -> Result<(), String> {
        let body = self.guest_refs(
            &record.parent,
            &[],
            &[
                "/usr/bin/podman",
                "--remote=false",
                "inspect",
                "--format",
                "{{.ID}} {{.State.Pid}} {{.State.Running}}",
                &record.child,
            ],
            deadline,
        );
        let Ok(body) = body else {
            return Err(texec::err_denied());
        };
        if String::from_utf8_lossy(&body).trim() != format!("{} {} true", record.child, record.pid)
        {
            return Err(texec::err_denied());
        }
        self.nested_namespace(record, deadline)?;
        let body = self.guest_refs(
            &record.parent,
            &[],
            &[
                "/usr/bin/podman",
                "--remote=false",
                "inspect",
                "--format",
                "{{json .Mounts}}",
                &record.child,
            ],
            deadline,
        );
        let (body, failed) = match body {
            Ok(body) => (body, false),
            Err(_) => (Vec::new(), true),
        };
        if record.muse
            && muse_readonly_mount(
                &body,
                failed,
                &format!("/run/soda-muse/nested/{}", record.registration),
                "/run/soda-muse/credentials",
            )
            .is_err()
        {
            return Err(texec::err_denied());
        }
        Ok(())
    }

    fn nested_namespace(&self, record: &MuseNested, deadline: Instant) -> Result<(), String> {
        let body = self
            .guest_refs(
                &record.parent,
                &[],
                &["/usr/bin/readlink", &format!("/proc/{}/ns/pid", record.pid)],
                deadline,
            )
            .map_err(|_| texec::err_denied())?;
        if String::from_utf8_lossy(&body).trim() != record.namespace {
            return Err(texec::err_denied());
        }
        let body = self
            .guest_refs(
                &record.parent,
                &[],
                &[
                    "/usr/bin/readlink",
                    &format!("/proc/{}/ns/user", record.pid),
                    "/proc/1/ns/user",
                ],
                deadline,
            )
            .map_err(|_| texec::err_denied())?;
        let text = String::from_utf8_lossy(&body);
        let names: Vec<&str> = text.split_whitespace().collect();
        if names.len() != 2 || names[0] != names[1] {
            return Err(texec::err_denied());
        }
        Ok(())
    }

    /// `actorAccount`: guest username/home for a broker actor.
    /// Decoded tolerantly (`encoding/json`) like Go.
    pub fn actor_account(
        &self,
        container: &str,
        actor: i64,
        deadline: Instant,
    ) -> Result<(String, String), String> {
        let body = self.guest_refs(
            container,
            &[],
            &["/usr/local/bin/muse", "--soda-account", &actor.to_string()],
            deadline,
        );
        let Ok(body) = body else {
            return Err(texec::err_denied());
        };
        if body.len() > 4096 {
            return Err(texec::err_denied());
        }
        let v = json::decode_tolerant(&body).map_err(|_| texec::err_denied())?;
        let Some(fields) = v.as_object() else {
            return Err(texec::err_denied());
        };
        let get = |name: &str| {
            fields
                .iter()
                .find(|(k, _)| k == name)
                .and_then(|(_, v)| v.as_str())
                .unwrap_or("")
                .to_string()
        };
        let username = get("Username");
        if !domain::valid_login(&username) || username == "root" {
            return Err(texec::err_denied());
        }
        Ok((username, get("HomeDir")))
    }

    fn nested_caller(
        &self,
        mut caller: MuseCaller,
        peer: &MusePeer,
        deadline: Instant,
    ) -> Result<MuseCaller, String> {
        let (username, home) = self.actor_account(&caller.container, caller.actor, deadline)?;
        caller.login = username;
        caller.home = home;
        let mappings = std::fs::read_to_string(format!("/proc/{}/gid_map", peer.pid))
            .map_err(|_| texec::err_denied())?;
        caller.gid = muse_mapped_uid(&mappings, peer.gid)?;
        if !self.authorized_caller(&caller, peer, deadline) {
            return Err(texec::err_denied());
        }
        Ok(caller)
    }

    /// `MuseRuntime.verifyGuestBinary`: pinned dispatcher digest, ELF
    /// header and version handshake.
    pub fn verify_guest_binary(
        &self,
        container: &str,
        child: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        if !domain::valid_container_id(&self.binary_sha256) {
            return Err(texec::err_denied());
        }
        let mut prefix = Vec::new();
        if !child.is_empty() {
            prefix.extend([
                "/usr/bin/podman".to_string(),
                "--remote=false".to_string(),
                "exec".to_string(),
                child.to_string(),
            ]);
        }
        let mut hash_argv = prefix.clone();
        hash_argv.extend([
            "/usr/bin/sha256sum".to_string(),
            "/usr/local/libexec/soda/muse".to_string(),
        ]);
        let hash = match self.guest(container, &[], &hash_argv, deadline) {
            Ok(hash) => hash,
            Err(_) => return Err(texec::err_denied()),
        };
        let text = String::from_utf8_lossy(&hash);
        let fields: Vec<&str> = text.split_whitespace().collect();
        if fields.len() != 2 || fields[0] != self.binary_sha256 {
            return Err(texec::err_denied());
        }
        let mut head_argv = prefix.clone();
        head_argv.extend([
            "/usr/bin/head".to_string(),
            "--bytes=64".to_string(),
            "/usr/local/libexec/soda/muse".to_string(),
        ]);
        let header = self
            .guest(container, &[], &head_argv, deadline)
            .map_err(|_| texec::err_denied())?;
        muse_elf(&header, host_go_arch())?;
        if self.binary_version.is_empty() {
            return Err(texec::err_denied());
        }
        let mut check_argv = prefix;
        check_argv.extend([
            "/usr/local/bin/muse".to_string(),
            "--soda-check".to_string(),
            self.binary_version.clone(),
        ]);
        self.guest(container, &[], &check_argv, deadline)
            .map(|_| ())
    }

    // ---------- execution lifecycle (muse_execution_linux.go) ----------

    /// `MuseRuntime.Start` preparation: validate, sanitize, resolve,
    /// verify, reserve, stage. Returns the staged execution; the caller
    /// spawns [`muse_command_argv`], then calls [`deliver_execution`].
    pub fn prepare_execution(
        &self,
        peer: &MusePeer,
        request: &LaunchRequest,
        deadline: Instant,
    ) -> Result<MuseExecution, String> {
        let budget = deadline.min(Instant::now() + Duration::from_secs(30));
        let mut request = request.clone();
        if request.validate().is_err() {
            return Err(texec::err_denied());
        }
        request.args = muse_arguments(&request.args)?;
        let caller = self.resolve(peer, budget)?;
        self.verify_guest_binary(&caller.container, &caller.child, budget)?;
        let execution = self.reserve_execution(&caller, &request, budget)?;
        if !muse_peer_alive(peer) {
            let cleanup = Instant::now() + Duration::from_secs(30);
            let _ = self.hooks.end(caller.actor, &execution.lease.id, cleanup);
            return Err(texec::err_denied());
        }
        if let Err(err) = self.stage(&caller, &execution.path, &request.config_home, budget) {
            let cleanup = Instant::now() + Duration::from_secs(30);
            let _ = self.stop_execution(
                &execution.binding,
                caller.actor,
                &execution.lease.id,
                true,
                cleanup,
            );
            return Err(err);
        }
        Ok(execution)
    }

    /// `MuseRuntime.reserveExecution`: broker connection, lease, binding.
    pub fn reserve_execution(
        &self,
        caller: &MuseCaller,
        request: &LaunchRequest,
        deadline: Instant,
    ) -> Result<MuseExecution, String> {
        let connection = self.hooks.select(
            caller.actor,
            &caller.project,
            &request.connection_id,
            deadline,
        )?;
        let id = texec::rand_id().map_err(|_| texec::err_denied())?;
        let lease = self.hooks.acquire(
            &AcquireRequest {
                execution_id: id.clone(),
                actor_id: caller.actor,
                connection_id: connection,
                project_id: caller.project.clone(),
                kind: texec::KIND_TERMINAL.to_string(),
                provider_id: texec::PROVIDER_MUSE.to_string(),
                deadline_secs: texec::now_unix() + 12 * 3600,
                ..Default::default()
            },
            deadline,
        )?;
        let path = format!("/run/soda-muse/{id}");
        let path = if caller.child.is_empty() {
            path
        } else {
            format!("/run/soda-muse/nested/{}/{id}", caller.registration)
        };
        let binding = Binding {
            kind: texec::KIND_TERMINAL.to_string(),
            id: id.clone(),
            project: caller.container.clone(),
            child_id: caller.child.clone(),
            uid: caller.uid,
            gid: caller.gid,
            login: caller.login.clone(),
            generation: lease.generation,
            scope: "muse-project".to_string(),
            credential_root: path.clone(),
            ..Default::default()
        };
        Ok(MuseExecution {
            caller: caller.clone(),
            request: request.clone(),
            lease,
            binding,
            path: path.clone(),
            unit: format!("soda-muse-{id}.service"),
        })
    }

    /// `MuseRuntime.deliverExecution`: await the unit, bind the invocation,
    /// attach the lease, deliver the credential, plant the ready marker.
    pub fn deliver_execution(
        &self,
        execution: &mut MuseExecution,
        deadline: Instant,
    ) -> Result<(), String> {
        let budget = deadline.min(Instant::now() + Duration::from_secs(30));
        self.await_unit(&execution.caller.container, &execution.unit, budget)?;
        let body = self.guest(
            &execution.caller.container,
            &[],
            &unit_invocation_argv(&execution.unit),
            budget,
        )?;
        execution.binding.invocation_id = String::from_utf8_lossy(&body).trim().to_string();
        if !texec::valid_terminal_id(&execution.binding.invocation_id) {
            return Err(texec::err_denied());
        }
        let mut delivery = self
            .hooks
            .attach(&execution.lease.id, &execution.binding, budget)?;
        let mut credential = delivery.credential.take().unwrap_or_default();
        if !texec::credential_valid(&credential) {
            return Err(texec::err_denied());
        }
        let target = format!("{}/auth.json", execution.path);
        let dd = [
            "/usr/bin/dd".to_string(),
            format!("of={target}"),
            "status=none".to_string(),
        ];
        let dd_result = self.guest(&execution.caller.container, &credential, &dd, budget);
        for byte in credential.iter_mut() {
            *byte = 0;
        }
        dd_result.map(|_| ())?;
        self.guest_refs(
            &execution.caller.container,
            &[],
            &["/usr/bin/touch", &format!("{}/ready", execution.path)],
            budget,
        )
        .map(|_| ())
    }

    /// `MuseRuntime.controlExecution`: unit signal or PTY resize.
    pub fn control_execution(
        &self,
        execution: &MuseExecution,
        stdin_fd: RawFd,
        child_pid: Option<i32>,
        control: &LaunchControl,
        deadline: Instant,
    ) -> Result<(), String> {
        if control.signal != 0 {
            if control.cols != 0 || control.rows != 0 || !muse_signal(control.signal) {
                return Err(texec::err_denied());
            }
            return self
                .guest_refs(
                    &execution.caller.container,
                    &[],
                    &[
                        "/usr/bin/systemctl",
                        "kill",
                        "--kill-whom=all",
                        &format!("--signal={}", control.signal),
                        &execution.unit,
                    ],
                    deadline,
                )
                .map(|_| ());
        }
        if !execution.request.tty {
            return Err(texec::err_denied());
        }
        muse_resize(stdin_fd, control)?;
        if let Some(pid) = child_pid {
            // SAFETY: kill with a valid signal number; ESRCH is ignored.
            unsafe {
                libc::kill(pid, libc::SIGWINCH);
            }
        }
        Ok(())
    }

    /// `MuseRuntime.stage`: credential directory plus config population.
    pub fn stage(
        &self,
        caller: &MuseCaller,
        path: &str,
        config_home: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/install", "--directory", "--mode=0700", path],
            deadline,
        )?;
        if caller.child.is_empty() {
            self.guest_refs(
                &caller.container,
                &[],
                &[
                    "/usr/bin/mount",
                    "--types=tmpfs",
                    "--options=mode=0700,size=4M",
                    "tmpfs",
                    path,
                ],
                deadline,
            )?;
        }
        self.stage_files(caller, path, deadline)?;
        self.stage_config(caller, path, config_home, deadline)
    }

    fn stage_files(
        &self,
        caller: &MuseCaller,
        path: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let dir = format!("{path}/config/muse");
        self.guest_refs(
            &caller.container,
            &[],
            &[
                "/usr/bin/install",
                "--directory",
                "--mode=0700",
                &format!("--owner={}", caller.uid),
                &format!("--group={}", caller.gid),
                &dir,
            ],
            deadline,
        )?;
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chmod", "0711", path],
            deadline,
        )?;
        let auth = format!("{path}/auth.json");
        self.guest(
            &caller.container,
            b"{}",
            &[
                "/usr/bin/dd".to_string(),
                format!("of={auth}"),
                "status=none".to_string(),
            ],
            deadline,
        )?;
        let ownership = format!("{}:{}", caller.uid, caller.gid);
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chown", &ownership, &auth],
            deadline,
        )?;
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chmod", "0600", &auth],
            deadline,
        )
        .map(|_| ())
    }

    fn stage_config(
        &self,
        caller: &MuseCaller,
        path: &str,
        config_home: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let source = if config_home.is_empty() {
            format!("{}/.config", caller.home)
        } else {
            config_home.to_string()
        };
        self.populate_config(caller, path, &source, deadline)?;
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chmod", "0711", &format!("{path}/config")],
            deadline,
        )?;
        self.auth_mount_target(caller, path, deadline)
    }

    fn populate_config(
        &self,
        caller: &MuseCaller,
        path: &str,
        source: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        if !caller.child.is_empty() {
            return self.copy_nested_config(caller, path, source, deadline);
        }
        self.podman(
            &[],
            &[
                "exec".to_string(),
                format!("--user={}:{}", caller.uid, caller.gid),
                caller.container.clone(),
                "/usr/local/bin/muse".to_string(),
                "--soda-copy-config".to_string(),
                format!("{source}/muse"),
                format!("{path}/config/muse"),
            ],
            deadline,
        )
        .map(|_| ())
    }

    fn auth_mount_target(
        &self,
        caller: &MuseCaller,
        path: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        if !caller.child.is_empty() {
            return self
                .guest_refs(
                    &caller.container,
                    &[],
                    &[
                        "/usr/bin/ln",
                        "--symbolic",
                        "../../auth.json",
                        &format!("{path}/config/muse/auth.json"),
                    ],
                    deadline,
                )
                .map(|_| ());
        }
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/touch", &format!("{path}/config/muse/auth.json")],
            deadline,
        )
        .map(|_| ())
    }

    fn copy_nested_config(
        &self,
        caller: &MuseCaller,
        path: &str,
        source: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let body = self.guest_refs(
            &caller.container,
            &[],
            &[
                "/usr/bin/nsenter",
                &format!("--target={}", caller.nested_pid),
                "--mount",
                "--pid",
                "--uts",
                "--ipc",
                "--net",
                "--root",
                &format!("--setuid={}", caller.uid),
                &format!("--setgid={}", caller.gid),
                "--",
                "/usr/local/bin/muse",
                "--soda-read-config",
                &format!("{source}/muse"),
            ],
            deadline,
        )?;
        if body.len() > 3 << 20 {
            return Err(texec::err_denied());
        }
        let mut view = decode_config_view(&body)?;
        for name in ["settings.json", "trust.json"] {
            let Some(value) = view.remove(name) else {
                continue;
            };
            let target = format!("{path}/config/muse/{name}");
            let dd = [
                "/usr/bin/dd".to_string(),
                format!("of={target}"),
                "status=none".to_string(),
            ];
            self.guest(&caller.container, &value, &dd, deadline)?;
            let mut value = value;
            for byte in value.iter_mut() {
                *byte = 0;
            }
            let ownership = format!("{}:{}", caller.uid, caller.gid);
            self.guest_refs(
                &caller.container,
                &[],
                &["/usr/bin/chown", &ownership, &target],
                deadline,
            )?;
            self.guest_refs(
                &caller.container,
                &[],
                &["/usr/bin/chmod", "0600", &target],
                deadline,
            )?;
        }
        Ok(())
    }

    /// `MuseRuntime.stopExecution`: stop the unit, verify its cgroup is
    /// gone, retire files, return custody.
    pub fn stop_execution(
        &self,
        binding: &Binding,
        actor: i64,
        lease_id: &str,
        return_custody: bool,
        deadline: Instant,
    ) -> Result<(), String> {
        let container = binding.project.clone();
        let unit = format!("soda-muse-{}.service", binding.id);
        let stop_result = self.guest_refs(
            &container,
            &[],
            &["/usr/bin/systemctl", "stop", &unit],
            deadline,
        );
        let body = self
            .guest(&container, &[], &unit_active_argv(&unit), deadline)
            .map_err(|_| texec::err_uncertain())?;
        if String::from_utf8_lossy(&body).trim() != "inactive" {
            return Err(texec::err_uncertain());
        }
        if stop_result.is_err() {
            self.guest_refs(
                &container,
                &[],
                &[
                    "/usr/bin/test",
                    "!",
                    "-d",
                    &format!("/sys/fs/cgroup/system.slice/{unit}"),
                ],
                deadline,
            )
            .map_err(|_| texec::err_uncertain())?;
        }
        self.retire_execution_files(binding, deadline)?;
        if return_custody {
            return self.hooks.end(actor, lease_id, deadline);
        }
        Ok(())
    }

    /// `MuseRuntime.ValidateMuseBinding`: exact incarnation + active unit.
    pub fn validate_muse_binding(
        &self,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<(), String> {
        if binding.validate().is_err()
            || !texec::valid_terminal_id(&binding.id)
            || !domain::valid_container_id(&binding.project)
            || !domain::valid_login(&binding.login)
        {
            return Err(texec::err_denied());
        }
        let body = self
            .guest(
                &binding.project,
                &[],
                &unit_active_argv(&format!("soda-muse-{}.service", binding.id)),
                deadline,
            )
            .map_err(|_| texec::err_stale())?;
        if String::from_utf8_lossy(&body).trim() != "active" {
            return Err(texec::err_stale());
        }
        Ok(())
    }

    /// `MuseRuntime.awaitUnit`: active state within 5s.
    pub fn await_unit(&self, container: &str, unit: &str, deadline: Instant) -> Result<(), String> {
        let end = Instant::now() + Duration::from_secs(5);
        loop {
            if let Ok(body) = self.guest(container, &[], &unit_active_argv(unit), deadline) {
                if String::from_utf8_lossy(&body).trim() == "active" {
                    return Ok(());
                }
            }
            let now = Instant::now();
            if now >= deadline {
                return Err("context deadline exceeded".to_string());
            }
            if now >= end {
                return Err(texec::err_denied());
            }
            if sleep_until(now + Duration::from_millis(50), deadline).is_err() {
                return Err("context deadline exceeded".to_string());
            }
        }
    }

    /// `MuseRuntime.Muse`: broker-selected operations on recorded boundaries.
    pub fn muse_operation(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        if !muse_delivery_valid(delivery) {
            return Err(texec::err_denied());
        }
        let binding = delivery
            .lease
            .binding
            .as_ref()
            .ok_or_else(texec::err_denied)?;
        match binding.scope.as_str() {
            "muse-project" => self.project_operation(action, delivery, deadline)?,
            _ => return Err(texec::err_denied()),
        }
        Ok(delivery.clone())
    }

    fn project_operation(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<(), String> {
        let binding = delivery
            .lease
            .binding
            .as_ref()
            .ok_or_else(texec::err_denied)?;
        let unit = format!("soda-muse-{}.service", binding.id);
        if !muse_project_credential_root(binding) {
            return Err(texec::err_denied());
        }
        if let Ok(body) = self.guest(
            &binding.project,
            &[],
            &unit_invocation_argv(&unit),
            deadline,
        ) {
            let current = String::from_utf8_lossy(&body).trim().to_string();
            if !current.is_empty() && current != binding.invocation_id {
                return Err(texec::err_stale());
            }
        }
        if action == "validate" {
            return self.validate_muse_binding(binding, deadline);
        }
        if action != "stop" {
            return Err(texec::err_denied());
        }
        self.stop_execution(
            binding,
            delivery.lease.actor_id,
            &delivery.lease.id,
            false,
            deadline,
        )
    }

    // ---------- state cleanup (muse_state_linux.go) ----------

    /// `MuseRuntime.cleanupExecutionState`: remove the guest state dir.
    pub fn cleanup_execution_state(
        &self,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<(), String> {
        let container = state_container(binding)?;
        if container.is_empty() {
            return Err(texec::err_denied());
        }
        match self.invoke_state(
            binding,
            &[
                "container".to_string(),
                "exists".to_string(),
                container.clone(),
            ],
            deadline,
        ) {
            Ok(_) => {}
            Err(err) if texec::exit_code_of(&err) == Some(1) => return Ok(()),
            Err(_) => return Err(texec::err_uncertain()),
        }
        let state = format!("/tmp/soda-muse-state-{}", binding.id);
        let owner = format!("--user={}:{}", binding.uid, binding.gid);
        self.invoke_state(
            binding,
            &[
                "exec".to_string(),
                owner.clone(),
                container.clone(),
                "/usr/bin/rm".to_string(),
                "--recursive".to_string(),
                "--force".to_string(),
                "--".to_string(),
                state.clone(),
            ],
            deadline,
        )
        .map_err(|_| texec::err_uncertain())?;
        self.invoke_state(
            binding,
            &[
                "exec".to_string(),
                owner,
                container,
                "/usr/bin/test".to_string(),
                "!".to_string(),
                "-e".to_string(),
                state,
            ],
            deadline,
        )
        .map(|_| ())
        .map_err(|_| texec::err_uncertain())
    }

    fn invoke_state(
        &self,
        binding: &Binding,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        if binding.scope == "muse-project" && !binding.child_id.is_empty() {
            let mut full = vec!["/usr/bin/podman".to_string(), "--remote=false".to_string()];
            full.extend(args.iter().cloned());
            return self.guest(&binding.project, &[], &full, deadline);
        }
        self.podman(&[], args, deadline)
    }

    fn retire_mount(&self, container: &str, path: &str, deadline: Instant) -> Result<(), String> {
        if self
            .guest_refs(container, &[], &["/usr/bin/umount", path], deadline)
            .is_ok()
        {
            return Ok(());
        }
        if self
            .guest_refs(
                container,
                &[],
                &["/usr/bin/test", "!", "-e", path],
                deadline,
            )
            .is_ok()
        {
            return Ok(());
        }
        match self.guest_refs(
            container,
            &[],
            &["/usr/bin/mountpoint", "--quiet", path],
            deadline,
        ) {
            Err(err) if texec::exit_code_of(&err) == Some(32) => Ok(()),
            _ => Err(texec::err_uncertain()),
        }
    }

    fn retire_execution_files(&self, binding: &Binding, deadline: Instant) -> Result<(), String> {
        self.cleanup_execution_state(binding, deadline)?;
        if !binding.credential_root.contains("/nested/") {
            self.retire_mount(&binding.project, &binding.credential_root, deadline)?;
        }
        self.guest_refs(
            &binding.project,
            &[],
            &[
                "/usr/bin/rm",
                "--recursive",
                "--force",
                "--",
                &binding.credential_root,
            ],
            deadline,
        )
        .map(|_| ())
        .map_err(|_| texec::err_uncertain())
    }
}

/// `museResize`: PTY resize over the stdin fd.
pub fn muse_resize(stdin_fd: RawFd, control: &LaunchControl) -> Result<(), String> {
    if control.signal != 0 || control.cols == 0 || control.rows == 0 {
        return Err(texec::err_denied());
    }
    let ws = libc::winsize {
        ws_row: control.rows,
        ws_col: control.cols,
        ws_xpixel: 0,
        ws_ypixel: 0,
    };
    // SAFETY: ioctl with a valid winsize pointer.
    let result = unsafe { libc::ioctl(stdin_fd, libc::TIOCSWINSZ, &ws) };
    if result < 0 {
        return Err(texec::err_denied());
    }
    Ok(())
}

/// `stateContainer`: execution state owner (child for nested runs).
pub fn state_container(binding: &Binding) -> Result<String, String> {
    if !texec::valid_terminal_id(&binding.id) || binding.uid < 0 || binding.gid < 0 {
        return Err(texec::err_denied());
    }
    let mut container = binding.project.clone();
    if binding.scope == "muse-project" && !binding.child_id.is_empty() {
        container = binding.child_id.clone();
    }
    if !domain::valid_container_id(&container) {
        return Err(texec::err_denied());
    }
    Ok(container)
}

/// Daemon `OpenMuseListener` filesystem setup: create the interface
/// directory, require it empty, require the socket path absent. Returns
/// the directory path for the caller to bind.
pub fn prepare_muse_listener_dir(socket_path: &str) -> Result<(), String> {
    let dir = match socket_path.rfind('/') {
        Some(0) => "/",
        Some(i) => &socket_path[..i],
        None => ".",
    };
    std::fs::create_dir_all(dir)
        .map_err(|e| format!("muse interface directory unavailable: {e}"))?;
    // Go checks `ReadDir` success plus emptiness with one error.
    let empty = std::fs::read_dir(dir)
        .map(|mut entries| entries.next().is_none())
        .unwrap_or(false);
    if !empty {
        return Err(
            "muse interface directory must be empty before launch service startup".to_string(),
        );
    }
    match std::fs::symlink_metadata(socket_path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        _ => Err("muse launch socket is occupied".to_string()),
    }
}

// ---------- launch socket (muse_socket_linux.go) ----------

/// `SO_PEERPIDFD` (Linux 6.5+), absent from libc: Go uses the literal too.
const SO_PEERPIDFD: i32 = 77;

/// `musePeer`: kernel-attested pid/uid/gid plus a pinning pidfd.
/// Fails closed when the kernel lacks `SO_PEERPIDFD`.
pub fn muse_peer_from_fd(fd: RawFd) -> Result<MusePeer, String> {
    // SAFETY: getsockopt with correctly sized outputs.
    unsafe {
        let mut cred: libc::ucred = std::mem::zeroed();
        let mut cred_len = std::mem::size_of::<libc::ucred>() as libc::socklen_t;
        if libc::getsockopt(
            fd,
            libc::SOL_SOCKET,
            libc::SO_PEERCRED,
            &mut cred as *mut _ as *mut libc::c_void,
            &mut cred_len,
        ) != 0
        {
            return Err("muse peer credentials unavailable".to_string());
        }
        let mut pidfd: libc::c_int = -1;
        let mut pidfd_len = std::mem::size_of::<libc::c_int>() as libc::socklen_t;
        if libc::getsockopt(
            fd,
            libc::SOL_SOCKET,
            SO_PEERPIDFD,
            &mut pidfd as *mut _ as *mut libc::c_void,
            &mut pidfd_len,
        ) != 0
        {
            return Err("muse peer pidfd unavailable".to_string());
        }
        Ok(MusePeer {
            pid: cred.pid,
            uid: cred.uid,
            gid: cred.gid,
            pidfd,
        })
    }
}

fn close_fds(fds: &[RawFd]) {
    for fd in fds {
        // SAFETY: fds came from SCM_RIGHTS; double close is avoided by
        // construction (each fd closed exactly once).
        unsafe {
            libc::close(*fd);
        }
    }
}

/// Parse `SCM_RIGHTS` fds out of one control buffer (glibc `CMSG_NXTHDR`
/// iteration, alignment included).
pub fn parse_unix_rights(control: &[u8], controllen: usize) -> Result<Vec<RawFd>, String> {
    let mut fds = Vec::new();
    if controllen == 0 {
        return Ok(fds);
    }
    // SAFETY: manual cmsg iteration over the kernel-filled prefix.
    unsafe {
        let mhdr = libc::msghdr {
            msg_name: std::ptr::null_mut(),
            msg_namelen: 0,
            msg_iov: std::ptr::null_mut(),
            msg_iovlen: 0,
            msg_control: control.as_ptr() as *mut libc::c_void,
            msg_controllen: controllen as _,
            msg_flags: 0,
        };
        let align = std::mem::size_of::<usize>();
        let mut cmsg = libc::CMSG_FIRSTHDR(&mhdr);
        while !cmsg.is_null() {
            if (*cmsg).cmsg_level == libc::SOL_SOCKET && (*cmsg).cmsg_type == libc::SCM_RIGHTS {
                let data = libc::CMSG_DATA(cmsg);
                let end = (cmsg as *const u8).add((*cmsg).cmsg_len as usize);
                let mut cursor = data;
                while (cursor as *const u8).add(std::mem::size_of::<libc::c_int>()) <= end {
                    fds.push(*(cursor as *const libc::c_int));
                    cursor = cursor.add(std::mem::size_of::<libc::c_int>());
                }
            }
            let next = (cmsg as usize + ((*cmsg).cmsg_len as usize).div_ceil(align) * align)
                as *const libc::cmsghdr;
            let limit = (mhdr.msg_control as usize + mhdr.msg_controllen as usize) as *const u8;
            if (next as *const u8).add(std::mem::size_of::<libc::cmsghdr>()) > limit {
                break;
            }
            cmsg = next as *mut libc::cmsghdr;
        }
    }
    Ok(fds)
}

/// One launch request plus its stdio files (`museRequest`).
/// `files` holds `None`s when the caller passed no descriptors (register).
#[derive(Debug)]
pub struct MuseRequest {
    pub request: LaunchRequest,
    pub files: [Option<std::fs::File>; 3],
}

/// `museRequest` over an already-accepted fd: one 64KB datagram plus up
/// to 3 SCM_RIGHTS descriptors, 5s read budget.
pub fn muse_request_from_fd(fd: RawFd) -> Result<MuseRequest, String> {
    let mut body = vec![0u8; 65536];
    // SAFETY: CMSG_SPACE for 3 ints.
    let cmsg_len = unsafe { libc::CMSG_SPACE(3 * 4) } as usize;
    let mut control = vec![0u8; cmsg_len];
    let (n, controllen, flags) = unsafe {
        let mut iov = libc::iovec {
            iov_base: body.as_mut_ptr() as *mut libc::c_void,
            iov_len: body.len(),
        };
        let mut hdr: libc::msghdr = std::mem::zeroed();
        hdr.msg_iov = &mut iov;
        hdr.msg_iovlen = 1;
        hdr.msg_control = control.as_mut_ptr() as *mut libc::c_void;
        hdr.msg_controllen = control.len() as _;
        // 5s read budget like Go's `SetReadDeadline`.
        let mut pfd = libc::pollfd {
            fd,
            events: libc::POLLIN,
            revents: 0,
        };
        if libc::poll(&mut pfd, 1, 5000) <= 0 {
            return Err("muse launch request unreadable".to_string());
        }
        let n = libc::recvmsg(fd, &mut hdr, 0);
        if n < 0 {
            return Err("muse launch request unreadable".to_string());
        }
        (n as usize, hdr.msg_controllen as usize, hdr.msg_flags)
    };
    let fds = parse_unix_rights(&control, controllen)?;
    if flags & (libc::MSG_TRUNC | libc::MSG_CTRUNC) != 0 || (!fds.is_empty() && fds.len() != 3) {
        close_fds(&fds);
        return Err("invalid launch descriptors".to_string());
    }
    let mut files: [Option<std::fs::File>; 3] = [None, None, None];
    if fds.len() == 3 {
        for (i, fd) in fds.iter().enumerate() {
            // SAFETY: set CLOEXEC on the received fd, then adopt it.
            unsafe {
                let flags = libc::fcntl(*fd, libc::F_GETFD);
                if flags >= 0 {
                    libc::fcntl(*fd, libc::F_SETFD, flags | libc::FD_CLOEXEC);
                }
                files[i] = Some(std::fs::File::from_raw_fd(*fd));
            }
        }
    }
    let request = LaunchRequest::decode(&body[..n])?;
    if !muse_descriptors_valid(&request, &files) {
        return Err("invalid launch descriptors".to_string());
    }
    request.validate()?;
    Ok(MuseRequest { request, files })
}

/// `museDescriptorsValid`: register carries no stdio, launches carry all three.
pub fn muse_descriptors_valid(request: &LaunchRequest, files: &[Option<std::fs::File>; 3]) -> bool {
    if request.register.is_some() {
        return files[0].is_none();
    }
    files[0].is_some() && files[1].is_some() && files[2].is_some()
}

/// `museCommandExit`: shell wait status to launch outcome.
pub fn muse_command_exit(status: std::io::Result<std::process::ExitStatus>) -> LaunchExit {
    use std::os::unix::process::ExitStatusExt;
    match status {
        Ok(status) => match status.code() {
            Some(code) => LaunchExit {
                code,
                error: String::new(),
            },
            None => LaunchExit {
                code: 128 + status.signal().unwrap_or(0),
                error: String::new(),
            },
        },
        Err(_) => LaunchExit {
            code: 1,
            error: String::new(),
        },
    }
}

/// Split one complete JSON object off the front of `buffer`, tracking
/// strings and escapes like a streaming decoder. Returns the object
/// length when balanced.
pub fn split_json_object(buffer: &[u8]) -> Option<usize> {
    let mut depth = 0i32;
    let mut in_string = false;
    let mut escape = false;
    let mut started = false;
    for (i, &b) in buffer.iter().enumerate() {
        if in_string {
            if escape {
                escape = false;
            } else if b == b'\\' {
                escape = true;
            } else if b == b'"' {
                in_string = false;
            }
            continue;
        }
        match b {
            b'"' => in_string = true,
            b'{' => {
                depth += 1;
                started = true;
            }
            b'}' => {
                depth -= 1;
                if started && depth == 0 {
                    return Some(i + 1);
                }
                if depth < 0 {
                    return None;
                }
            }
            _ => {}
        }
    }
    None
}

/// `MuseLaunch`: dedicated SOCK_SEQPACKET launch listener service.
/// Served behind an [`Arc`] so each connection runs detached, like Go's
/// per-connection goroutine.
pub struct MuseLaunch<E, H> {
    pub runtime: MuseRuntime<E, H>,
}

impl<E, H> MuseLaunch<E, H> {
    pub fn new(runtime: MuseRuntime<E, H>) -> Self {
        MuseLaunch { runtime }
    }
}

impl<E: Executor + Send + Sync + 'static, H: MuseHooks + Send + Sync + 'static> MuseLaunch<E, H> {
    /// `MuseLaunch.Serve`: accept loop until `shutdown` is set, then join
    /// every live connection (Go's `wg.Wait`).
    /// `listen_fd` is a bound SOCK_SEQPACKET listener (owned by the caller).
    pub fn serve(
        self: &std::sync::Arc<Self>,
        listen_fd: RawFd,
        shutdown: &AtomicBool,
    ) -> Result<(), String> {
        // Non-blocking accept with a stop poll, mirroring Go's
        // context-cancelled `AcceptUnix`.
        unsafe {
            let flags = libc::fcntl(listen_fd, libc::F_GETFL);
            if flags >= 0 {
                libc::fcntl(listen_fd, libc::F_SETFL, flags | libc::O_NONBLOCK);
            }
        }
        let mut handles = Vec::new();
        let outcome = loop {
            if shutdown.load(Ordering::SeqCst) {
                break Ok(());
            }
            let mut pfd = libc::pollfd {
                fd: listen_fd,
                events: libc::POLLIN,
                revents: 0,
            };
            // SAFETY: valid one-element pollfd array.
            let ready = unsafe { libc::poll(&mut pfd, 1, 100) };
            if ready < 0 {
                let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
                if errno == libc::EINTR {
                    continue;
                }
                break Err(format!("muse accept failed: errno {errno}"));
            }
            if ready == 0 {
                continue;
            }
            // SAFETY: accept on a listening unix socket.
            let conn =
                unsafe { libc::accept(listen_fd, std::ptr::null_mut(), std::ptr::null_mut()) };
            if conn < 0 {
                let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
                if errno == libc::EAGAIN || errno == libc::EWOULDBLOCK || errno == libc::EINTR {
                    continue;
                }
                break Err(format!("muse accept failed: errno {errno}"));
            }
            let service = self.clone();
            handles.push(std::thread::spawn(move || {
                service.serve_one(conn, &service)
            }));
        };
        for handle in handles {
            let _ = handle.join();
        }
        outcome
    }

    fn serve_one(&self, conn: RawFd, service: &std::sync::Arc<Self>) {
        let result = self.serve_connection(conn, service);
        // `json.Encoder.Encode(result)`: framed with one newline.
        let line = result.encode_line();
        let bytes = line.as_bytes();
        let mut written = 0;
        while written < bytes.len() {
            // SAFETY: send on the connected fd.
            let n = unsafe {
                libc::send(
                    conn,
                    bytes[written..].as_ptr() as *const libc::c_void,
                    bytes.len() - written,
                    libc::MSG_NOSIGNAL,
                )
            };
            if n <= 0 {
                break;
            }
            written += n as usize;
        }
        // SAFETY: close the accepted fd exactly once.
        unsafe {
            libc::close(conn);
        }
    }

    fn serve_connection(&self, conn: RawFd, service: &std::sync::Arc<Self>) -> LaunchExit {
        let peer = match muse_peer_from_fd(conn) {
            Ok(peer) => peer,
            Err(_) => return LaunchExit::denied(),
        };
        // SAFETY: close the pidfd exactly once at the end.
        struct Closer(RawFd);
        impl Drop for Closer {
            fn drop(&mut self) {
                unsafe {
                    libc::close(self.0);
                }
            }
        }
        let _pidfd = Closer(peer.pidfd);
        let received = match muse_request_from_fd(conn) {
            Ok(received) => received,
            Err(_) => return LaunchExit::denied(),
        };
        if let Some(register) = &received.request.register {
            let deadline = Instant::now() + Duration::from_secs(30);
            return match self.runtime.register_nested(&peer, register, deadline) {
                Ok(()) => LaunchExit {
                    code: 0,
                    error: String::new(),
                },
                Err(_) => LaunchExit::denied(),
            };
        }
        let [stdin, stdout, stderr] = received.files;
        let (Some(stdin), Some(stdout), Some(stderr)) = (stdin, stdout, stderr) else {
            return LaunchExit::denied();
        };
        self.shell(
            conn,
            &peer,
            &received.request,
            [stdin, stdout, stderr],
            service,
        )
    }

    /// `MuseLaunch.shell`: spawn, deliver, supervise, finish.
    pub fn shell(
        &self,
        conn: RawFd,
        peer: &MusePeer,
        request: &LaunchRequest,
        stdio: [std::fs::File; 3],
        service: &std::sync::Arc<Self>,
    ) -> LaunchExit {
        self.shell_inner(conn, peer, request, stdio, Some(service.clone()))
    }

    fn shell_inner(
        &self,
        conn: RawFd,
        peer: &MusePeer,
        request: &LaunchRequest,
        stdio: [std::fs::File; 3],
        supervisor_owner: Option<std::sync::Arc<Self>>,
    ) -> LaunchExit {
        use std::os::unix::io::AsRawFd;
        let session_end = Instant::now() + Duration::from_secs(12 * 3600);
        let mut execution = match self.runtime.prepare_execution(peer, request, session_end) {
            Ok(execution) => execution,
            Err(_) => return LaunchExit::denied(),
        };
        // The supervisor resizes through the parent's stdin handle.
        // SAFETY: dup before spawn transfers ownership of stdio to the child.
        let stdin_fd = unsafe { libc::dup(stdio[0].as_raw_fd()) };
        let mut child = match spawn_execution(&execution, stdio) {
            Ok(child) => child,
            Err(_) => {
                unsafe {
                    libc::close(stdin_fd);
                }
                self.finish_unconfirmed(&execution);
                return LaunchExit::denied();
            }
        };
        if self
            .runtime
            .deliver_execution(&mut execution, session_end)
            .is_err()
        {
            let _ = child.kill();
            let _ = child.wait();
            unsafe {
                libc::close(stdin_fd);
            }
            self.finish_unconfirmed(&execution);
            return LaunchExit::denied();
        }
        // Control supervisor: any channel failure kills the session, like
        // Go's `museControls` cancel.
        if let Some(owner) = supervisor_owner {
            // SAFETY: dup the conn for the supervisor; it closes its copy.
            let conn_dup = unsafe { libc::dup(conn) };
            let child_pid = child.id() as i32;
            let snapshot = execution.clone();
            std::thread::spawn(move || {
                owner.control_loop(conn_dup, stdin_fd, child_pid, &snapshot, session_end);
                unsafe {
                    libc::close(conn_dup);
                    libc::close(stdin_fd);
                    libc::kill(child_pid, libc::SIGKILL);
                }
            });
        } else {
            unsafe {
                libc::close(stdin_fd);
            }
        }
        let mut result = muse_command_exit(child.wait());
        let cleanup = Instant::now() + Duration::from_secs(30);
        if self
            .runtime
            .stop_execution(
                &execution.binding,
                execution.caller.actor,
                &execution.lease.id,
                true,
                cleanup,
            )
            .is_err()
        {
            result = LaunchExit::cleanup_unconfirmed();
        }
        result
    }

    fn finish_unconfirmed(&self, execution: &MuseExecution) {
        let cleanup = Instant::now() + Duration::from_secs(30);
        let _ = self.runtime.stop_execution(
            &execution.binding,
            execution.caller.actor,
            &execution.lease.id,
            true,
            cleanup,
        );
    }

    /// Control supervisor (`museControls`): streaming strict control
    /// messages, 1MB total, unknown fields rejected. Any failure ends the
    /// loop; the spawner kills the session afterwards.
    pub fn control_loop(
        &self,
        conn: RawFd,
        stdin_fd: RawFd,
        child_pid: i32,
        execution: &MuseExecution,
        session_end: Instant,
    ) {
        let mut buffer = Vec::new();
        let mut chunk = [0u8; 4096];
        loop {
            if Instant::now() >= session_end {
                return;
            }
            // SAFETY: recv into the scratch chunk.
            let n = unsafe {
                libc::recv(
                    conn,
                    chunk.as_mut_ptr() as *mut libc::c_void,
                    chunk.len(),
                    0,
                )
            };
            if n <= 0 {
                return;
            }
            buffer.extend_from_slice(&chunk[..n as usize]);
            if buffer.len() > 1 << 20 {
                return;
            }
            loop {
                let trim: Vec<u8> = buffer
                    .iter()
                    .skip_while(|b| b.is_ascii_whitespace())
                    .cloned()
                    .collect();
                if trim.is_empty() {
                    buffer.clear();
                    break;
                }
                if trim[0] != b'{' {
                    return;
                }
                let Some(len) = split_json_object(&trim) else {
                    buffer = trim;
                    break;
                };
                let control = match LaunchControl::decode(&trim[..len]) {
                    Ok(control) => control,
                    Err(_) => return,
                };
                if self
                    .runtime
                    .control_execution(execution, stdin_fd, Some(child_pid), &control, session_end)
                    .is_err()
                {
                    return;
                }
                buffer = trim[len..].to_vec();
            }
        }
    }
}

/// Spawn the prepared execution with owned stdio files.
fn spawn_execution(
    execution: &MuseExecution,
    stdio: [std::fs::File; 3],
) -> Result<std::process::Child, String> {
    let [stdin, stdout, stderr] = stdio;
    let argv = muse_command_argv(
        &execution.caller,
        &execution.request,
        &execution.unit,
        &execution.path,
    );
    std::process::Command::new("/usr/bin/podman")
        .args(&argv)
        .env_clear()
        .envs(muse_host_environment())
        .stdin(std::process::Stdio::from(stdin))
        .stdout(std::process::Stdio::from(stdout))
        .stderr(std::process::Stdio::from(stderr))
        .spawn()
        .map_err(|e| format!("/usr/bin/podman failed: {e}"))
}

/// Tolerant `map[string][]byte` decode for nested config views
/// (`encoding/json` into `map[string][]byte`: base64 strings or numeric
/// arrays, like [`Kind::Bytes`](crate::json::Kind::Bytes)).
pub fn decode_config_view(body: &[u8]) -> Result<HashMap<String, Vec<u8>>, String> {
    let v = json::decode_tolerant(body).map_err(|_| texec::err_denied())?;
    let Some(fields) = v.as_object() else {
        return Err(texec::err_denied());
    };
    let mut out = HashMap::with_capacity(fields.len());
    for (key, value) in fields {
        let bytes = match value {
            Value::Null => Vec::new(),
            Value::Str(s) => {
                crate::ssh::b64_decode_go(s.as_bytes()).map_err(|_| texec::err_denied())?
            }
            Value::Array(items) => {
                let mut bytes = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        Value::Number(lit) => {
                            let n: i64 = lit.parse().map_err(|_| texec::err_denied())?;
                            if !(0..=255).contains(&n) {
                                return Err(texec::err_denied());
                            }
                            bytes.push(n as u8);
                        }
                        _ => return Err(texec::err_denied()),
                    }
                }
                bytes
            }
            _ => return Err(texec::err_denied()),
        };
        out.insert(key.clone(), bytes);
    }
    Ok(out)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::AtomicU64;
    use std::sync::Mutex;

    const PID: &str = "p0123456789abcdef01234567";
    const TID: &str = "0123456789abcdef0123456789abcdef";
    const IID: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    const CID: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
    const PIN: &str = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";

    static TMP_COUNTER: AtomicU64 = AtomicU64::new(0);
    static ENV_LOCK: Mutex<()> = Mutex::new(());

    fn deadline() -> Instant {
        Instant::now() + Duration::from_secs(30)
    }

    fn test_tmp(slug: &str) -> std::path::PathBuf {
        let n = TMP_COUNTER.fetch_add(1, Ordering::SeqCst);
        let dir = std::env::temp_dir().join(format!("t26m-{}-{n}-{slug}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    type RecordedCall = (Vec<u8>, String, Vec<String>);

    struct FakeExec {
        calls: Mutex<Vec<RecordedCall>>,
        script: Mutex<Vec<Result<Vec<u8>, String>>>,
    }

    impl FakeExec {
        fn new(script: Vec<Result<Vec<u8>, String>>) -> Self {
            FakeExec {
                calls: Mutex::new(Vec::new()),
                script: Mutex::new(script),
            }
        }

        fn calls(&self) -> Vec<RecordedCall> {
            self.calls.lock().unwrap().clone()
        }
    }

    impl Executor for FakeExec {
        fn run(
            &self,
            stdin: &[u8],
            cmd: &str,
            args: &[&str],
            _deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            self.calls.lock().unwrap().push((
                stdin.to_vec(),
                cmd.to_string(),
                args.iter().map(|s| s.to_string()).collect(),
            ));
            let mut script = self.script.lock().unwrap();
            if script.is_empty() {
                return Err("unexpected call".to_string());
            }
            script.remove(0)
        }
    }

    fn ok(body: &str) -> Result<Vec<u8>, String> {
        Ok(body.as_bytes().to_vec())
    }

    fn err(body: &str) -> Result<Vec<u8>, String> {
        Err(body.to_string())
    }

    struct FakeHooks {
        select_out: Mutex<Result<String, String>>,
        acquire_out: Mutex<Result<Lease, String>>,
        attach_out: Mutex<Result<Delivery, String>>,
        end_calls: Mutex<Vec<(i64, String)>>,
        end_out: Mutex<Result<(), String>>,
        authorize_out: Mutex<Result<(), String>>,
        nested_out: Mutex<Result<(), String>>,
        selects: Mutex<Vec<(i64, String, String)>>,
        acquires: Mutex<Vec<AcquireRequest>>,
        attaches: Mutex<Vec<(String, Binding)>>,
    }

    impl FakeHooks {
        fn new() -> Self {
            FakeHooks {
                select_out: Mutex::new(Ok("conn-1".to_string())),
                acquire_out: Mutex::new(Err("no acquire".to_string())),
                attach_out: Mutex::new(Err("no attach".to_string())),
                end_calls: Mutex::new(Vec::new()),
                end_out: Mutex::new(Ok(())),
                authorize_out: Mutex::new(Ok(())),
                nested_out: Mutex::new(Ok(())),
                selects: Mutex::new(Vec::new()),
                acquires: Mutex::new(Vec::new()),
                attaches: Mutex::new(Vec::new()),
            }
        }
    }

    impl MuseHooks for FakeHooks {
        fn acquire(&self, req: &AcquireRequest, _deadline: Instant) -> Result<Lease, String> {
            self.acquires.lock().unwrap().push(req.clone());
            self.acquire_out.lock().unwrap().clone()
        }
        fn attach(
            &self,
            lease_id: &str,
            binding: &Binding,
            _deadline: Instant,
        ) -> Result<Delivery, String> {
            self.attaches
                .lock()
                .unwrap()
                .push((lease_id.to_string(), binding.clone()));
            self.attach_out.lock().unwrap().clone()
        }
        fn end(&self, actor: i64, lease_id: &str, _deadline: Instant) -> Result<(), String> {
            self.end_calls
                .lock()
                .unwrap()
                .push((actor, lease_id.to_string()));
            self.end_out.lock().unwrap().clone()
        }
        fn authorize(&self, _actor: i64, _project: &str, _deadline: Instant) -> Result<(), String> {
            self.authorize_out.lock().unwrap().clone()
        }
        fn nested_authorize(
            &self,
            _actor: i64,
            _project: &str,
            _deadline: Instant,
        ) -> Result<(), String> {
            self.nested_out.lock().unwrap().clone()
        }
        fn select(
            &self,
            actor: i64,
            project: &str,
            selected: &str,
            _deadline: Instant,
        ) -> Result<String, String> {
            self.selects
                .lock()
                .unwrap()
                .push((actor, project.to_string(), selected.to_string()));
            self.select_out.lock().unwrap().clone()
        }
    }

    fn runtime(exec: FakeExec) -> MuseRuntime<FakeExec, FakeHooks> {
        MuseRuntime::new(exec, FakeHooks::new(), "1.0".to_string(), PIN.to_string())
    }

    fn caller() -> MuseCaller {
        MuseCaller {
            project: PID.to_string(),
            container: CID.to_string(),
            login: "dev".to_string(),
            home: "/home/dev".to_string(),
            actor: 7,
            uid: 1000,
            gid: 1000,
            ..Default::default()
        }
    }

    fn lease_fixture() -> Lease {
        Lease {
            provider_id: texec::PROVIDER_MUSE.to_string(),
            id: "lease-m".to_string(),
            connection_id: "conn-1".to_string(),
            generation: 4,
            actor_id: 7,
            project_id: PID.to_string(),
            execution_id: TID.to_string(),
            kind: texec::KIND_TERMINAL.to_string(),
            binding: Some(Binding {
                kind: texec::KIND_TERMINAL.to_string(),
                id: TID.to_string(),
                project: CID.to_string(),
                login: "dev".to_string(),
                uid: 1000,
                gid: 1000,
                scope: texec::SCOPE_MUSE_PROJECT.to_string(),
                invocation_id: IID.to_string(),
                credential_root: format!("/run/soda-muse/{TID}"),
                generation: 4,
                ..Default::default()
            }),
            ..Default::default()
        }
    }

    // ----- launch protocol -----

    #[test]
    fn launch_request_validate_matrix() {
        // Vectors from Go `TestLaunchRequestHasNoAuthority`.
        for req in [
            LaunchRequest {
                cwd: "relative".to_string(),
                ..Default::default()
            },
            LaunchRequest {
                cwd: "/workspace".to_string(),
                args: vec!["a\0b".to_string()],
                ..Default::default()
            },
            LaunchRequest {
                cwd: "/workspace".to_string(),
                tty: true,
                ..Default::default()
            },
            LaunchRequest {
                cwd: "/workspace".to_string(),
                config_home: "relative".to_string(),
                ..Default::default()
            },
            LaunchRequest {
                cwd: "/workspace".to_string(),
                register: Some(NestedRegistration {
                    actor_id: 1,
                    ..Default::default()
                }),
                ..Default::default()
            },
        ] {
            assert!(req.validate().is_err(), "{req:?}");
        }
        assert!(LaunchRequest {
            cwd: "/workspace".to_string(),
            args: vec![
                "--model".to_string(),
                "meta/test".to_string(),
                "prompt with spaces".to_string()
            ],
            ..Default::default()
        }
        .validate()
        .is_ok());
        // Limits.
        assert!(LaunchRequest {
            cwd: "/w".to_string(),
            args: vec!["x".repeat(32769)],
            ..Default::default()
        }
        .validate()
        .is_err());
        assert!(LaunchRequest {
            cwd: "/w".to_string(),
            args: vec!["x".repeat(200); 200],
            ..Default::default()
        }
        .validate()
        .is_err());
        assert!(LaunchRequest {
            cwd: "/w".to_string(),
            args: vec!["x".to_string(); 257],
            ..Default::default()
        }
        .validate()
        .is_err());
        assert!(LaunchRequest {
            cwd: "/w".to_string(),
            term: "t".repeat(129),
            ..Default::default()
        }
        .validate()
        .is_err());
        assert!(LaunchRequest {
            cwd: "/w".to_string(),
            connection_id: "c".repeat(129),
            ..Default::default()
        }
        .validate()
        .is_err());
        assert!(LaunchRequest {
            cwd: "/w".to_string(),
            home: "relative".to_string(),
            ..Default::default()
        }
        .validate()
        .is_err());
        assert!(LaunchRequest {
            cwd: "/w".to_string(),
            tty: true,
            cols: 80,
            rows: 0,
            ..Default::default()
        }
        .validate()
        .is_err());
        assert!(LaunchRequest {
            cwd: "/w".to_string(),
            tty: true,
            cols: 80,
            rows: 24,
            ..Default::default()
        }
        .validate()
        .is_ok());
        // Registration variant ignores CWD/args/connection/TTY presence rules.
        let reg = NestedRegistration {
            child_id: CID.to_string(),
            actor_id: 7,
            registration_id: TID.to_string(),
            muse: true,
        };
        assert!(LaunchRequest {
            register: Some(reg.clone()),
            ..Default::default()
        }
        .validate()
        .is_ok());
        assert!(LaunchRequest {
            register: Some(reg.clone()),
            tty: true,
            ..Default::default()
        }
        .validate()
        .is_err());
        assert!(LaunchRequest {
            register: Some(reg.clone()),
            args: vec!["x".to_string()],
            ..Default::default()
        }
        .validate()
        .is_err());
        assert!(LaunchRequest {
            register: Some(NestedRegistration {
                muse: false,
                ..reg.clone()
            }),
            ..Default::default()
        }
        .validate()
        .is_err());
        assert!(LaunchRequest {
            register: Some(NestedRegistration { actor_id: 0, ..reg }),
            ..Default::default()
        }
        .validate()
        .is_err());
    }

    #[test]
    fn launch_decode_matrix() {
        let body = format!(
            "{{\"home\":\"/h\",\"register\":{{\"child_id\":{CID:?},\"actor_id\":\"7\",\"registration_id\":{TID:?},\"muse\":true}},\"config_home\":\"/c\",\"term\":\"xterm\",\"connection_id\":\"conn\",\"cwd\":\"/w\",\"args\":[\"a\",\"b\"],\"tty\":true,\"cols\":80,\"rows\":24}}"
        );
        let req = LaunchRequest::decode(body.as_bytes()).unwrap();
        assert_eq!(req.register.as_ref().unwrap().actor_id, 7);
        assert_eq!((req.cols, req.rows), (80, 24));
        assert!(req.tty);
        assert!(LaunchRequest::decode(br#"{"cols":70000}"#).is_err());
        assert!(LaunchRequest::decode(br#"{"cols":-1}"#).is_err());
        assert!(LaunchRequest::decode(br#"{"bogus":1}"#).is_err());
        assert!(LaunchRequest::decode(br#"{"register":{"actor_id":""}}"#).is_err());
        assert!(LaunchRequest::decode(br#"{"register":null}"#)
            .unwrap()
            .register
            .is_none());
        assert_eq!(
            LaunchRequest::decode(br#"{}"#).unwrap(),
            LaunchRequest::default()
        );
        let control = LaunchControl::decode(br#"{"signal":15}"#).unwrap();
        assert_eq!(
            control,
            LaunchControl {
                signal: 15,
                cols: 0,
                rows: 0
            }
        );
        assert!(LaunchControl::decode(br#"{"cols":80,"rows":24,"bogus":1}"#).is_err());
        assert!(LaunchControl::decode(br#"{"rows":65536}"#).is_err());
        assert_eq!(
            LaunchExit::denied().encode(),
            r#"{"code":1,"error":"Muse launch denied"}"#
        );
        assert_eq!(
            LaunchExit::cleanup_unconfirmed().encode(),
            r#"{"code":1,"error":"Muse cleanup unconfirmed"}"#
        );
        assert_eq!(
            LaunchExit {
                code: 0,
                error: String::new()
            }
            .encode(),
            r#"{"code":0}"#
        );
        assert!(LaunchExit::denied().encode_line().ends_with('\n'));
    }

    #[test]
    fn muse_arguments_matrix() {
        let args = |list: &[&str]| list.iter().map(|s| s.to_string()).collect::<Vec<_>>();
        // Vectors from Go `TestMuseNativeCommandProviderPlacement`.
        assert_eq!(
            muse_arguments(&args(&[
                "exec",
                "--model",
                "native/model",
                "prompt with spaces"
            ]))
            .unwrap(),
            args(&[
                "exec",
                "--provider",
                "meta",
                "--model",
                "native/model",
                "prompt with spaces"
            ])
        );
        assert_eq!(
            muse_arguments(&args(&["prompt"])).unwrap(),
            args(&["--provider", "meta", "prompt"])
        );
        assert_eq!(
            muse_arguments(&args(&["config", "--help"])).unwrap(),
            args(&["config", "--help"])
        );
        assert_eq!(
            muse_arguments(&args(&[])).unwrap(),
            args(&["--provider", "meta"])
        );
        // Auth surface denied.
        for denied in [
            vec!["auth"],
            vec!["login"],
            vec!["logout"],
            vec!["--model", "m", "auth"],
            vec!["--provider", "evil"],
            vec!["exec", "--base-url=https://evil.example"],
        ] {
            assert_eq!(
                muse_arguments(&args(&denied)).unwrap_err(),
                texec::ERR_DENIED,
                "{denied:?}"
            );
        }
        // Value flags hide their operand from the positional scan.
        assert!(muse_arguments(&args(&["--model", "auth"])).is_ok());
        assert!(muse_arguments(&args(&["--model"])).is_ok());
        assert_eq!(
            muse_arguments(&args(&["resume", "abc"])).unwrap(),
            args(&["resume", "--provider", "meta", "abc"])
        );
    }

    #[test]
    fn connection_directory_matrix() {
        let ready = MuseConnection {
            id: "a".to_string(),
            provider_id: "muse".to_string(),
            state: "ready".to_string(),
        };
        let reauth = MuseConnection {
            id: "b".to_string(),
            provider_id: "muse".to_string(),
            state: "reauth".to_string(),
        };
        let codex = MuseConnection {
            id: "c".to_string(),
            provider_id: "codex".to_string(),
            state: "ready".to_string(),
        };
        assert!(
            muse_connection_authorized(&[reauth.clone(), codex.clone(), ready.clone()]).is_ok()
        );
        assert_eq!(
            muse_connection_authorized(&[reauth, codex]).unwrap_err(),
            texec::ERR_DENIED
        );
        assert_eq!(
            muse_connection_authorized(&[]).unwrap_err(),
            texec::ERR_DENIED
        );
        assert_eq!(
            select_muse_connection(std::slice::from_ref(&ready), "").unwrap(),
            "a"
        );
        let two = vec![
            ready.clone(),
            MuseConnection {
                id: "z".to_string(),
                provider_id: "muse".to_string(),
                state: "ready".to_string(),
            },
        ];
        assert_eq!(select_muse_connection(&two, "z").unwrap(), "z");
        assert_eq!(
            select_muse_connection(&two, "").unwrap_err(),
            "choose an authorized muse connection with SODA_MUSE_CONNECTION"
        );
        assert_eq!(
            select_muse_connection(&two, "absent").unwrap_err(),
            texec::ERR_DENIED
        );
        assert_eq!(
            select_muse_connection(&[], "").unwrap_err(),
            texec::ERR_DENIED
        );
    }

    // ----- pure resolution helpers -----

    #[test]
    fn cgroup_matrix() {
        assert_eq!(
            muse_project_cgroup(&format!(
                "0::/user.slice/user-1000.slice/session-1.scope/libpod-{CID}.scope"
            ))
            .unwrap(),
            CID
        );
        assert_eq!(
            muse_project_cgroup(&format!("12:memory:/kubepods/libpod-{CID}.scope")).unwrap(),
            CID
        );
        assert!(muse_project_cgroup("0::/init.scope").is_err());
        assert!(muse_project_cgroup(&format!("0::/libpod-{CID}.scope.extra")).is_err());
        assert!(muse_project_cgroup(
            "0::/libpod-0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF.scope"
        )
        .is_err());
        assert!(muse_project_cgroup("").is_err());
    }

    #[test]
    fn uidmap_matrix() {
        assert_eq!(
            muse_mapped_uid("         0          1      65536\n", 1).unwrap(),
            0
        );
        assert_eq!(muse_mapped_uid("0 100000 262144\n", 100500).unwrap(), 500);
        assert_eq!(
            muse_mapped_uid("0 1 1\n1 100000 65536\n", 100000).unwrap(),
            1
        );
        assert!(muse_mapped_uid("0 100000 262144\n", 500000).is_err());
        assert!(muse_mapped_uid("0 0 10\n", 5).is_err()); // outside 0 skipped
        assert!(muse_mapped_uid("garbage\n", 1).is_err());
        assert!(muse_mapped_uid("", 1).is_err());
    }

    #[test]
    fn passwd_and_modes_matrix() {
        let row = |parts: &[&str]| parts.iter().map(|s| s.to_string()).collect::<Vec<_>>();
        assert!(muse_passwd_valid(
            &row(&["dev", "x", "1000", "1000", "", "/home/dev", "/bin/sh"]),
            1000
        ));
        assert!(!muse_passwd_valid(
            &row(&["root", "x", "0", "0", "", "/root", "/bin/sh"]),
            0
        ));
        assert!(!muse_passwd_valid(
            &row(&["dev", "x", "1001", "1000", "", "/home/dev", "/bin/sh"]),
            1000
        ));
        assert!(!muse_passwd_valid(
            &row(&["dev", "x", "1000", "1000", "", "home/dev", "/bin/sh"]),
            1000
        ));
        assert!(!muse_passwd_valid(&row(&["dev", "x", "1000"]), 1000));
        assert!(!muse_passwd_valid(
            &row(&["Dev", "x", "1000", "1000", "", "/home/dev", "/bin/sh"]),
            1000
        ));
        let good = "0:0:755:directory\n0:0:755:directory\n0:0:755:directory\n0:0:755:directory\n0:0:600:regular file\n";
        assert!(muse_account_modes(good));
        assert!(!muse_account_modes(
            "0:0:755:directory\n0:0:600:regular file\n"
        ));
        assert!(!muse_account_modes(
            &good.replace("0:0:600:regular file", "0:0:644:regular file")
        ));
        assert!(!muse_account_modes(
            &good.replace("0:0:600:regular file", "0:0:600:directory")
        ));
        assert!(!muse_account_modes(
            &good.replace("0:0:755:directory", "1:0:755:directory")
        ));
        assert!(!muse_account_modes(
            &good.replace("0:0:755:directory", "0:0:775:directory")
        ));
    }

    #[test]
    fn registration_and_child_matrix() {
        let reg = NestedRegistration {
            child_id: CID.to_string(),
            actor_id: 7,
            registration_id: TID.to_string(),
            muse: true,
        };
        assert!(muse_registration_valid(&reg));
        assert!(!muse_registration_valid(&NestedRegistration {
            muse: false,
            ..reg.clone()
        }));
        assert!(!muse_registration_valid(&NestedRegistration {
            actor_id: 0,
            ..reg.clone()
        }));
        assert!(!muse_registration_valid(&NestedRegistration {
            child_id: "short".to_string(),
            ..reg
        }));
        assert_eq!(
            muse_child_pid(
                format!("{{\"id\":{CID:?},\"pid\":4242,\"running\":true}}").as_bytes(),
                false,
                CID
            )
            .unwrap(),
            4242
        );
        assert!(muse_child_pid(b"{}", true, CID).is_err());
        assert!(muse_child_pid(&vec![b'x'; 4097], false, CID).is_err());
        assert!(muse_child_pid(
            format!("{{\"id\":{CID:?},\"pid\":0,\"running\":true}}").as_bytes(),
            false,
            CID
        )
        .is_err());
        assert!(muse_child_pid(
            format!("{{\"id\":{CID:?},\"pid\":42,\"running\":false}}").as_bytes(),
            false,
            CID
        )
        .is_err());
        assert!(muse_child_pid(br#"{"id":"other","pid":42,"running":true}"#, false, CID).is_err());
        assert!(muse_child_pid(b"nope", false, CID).is_err());
    }

    #[test]
    fn readonly_mount_matrix() {
        let src = "/run/soda-muse/nested/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
        let good = format!(
            r#"[{{"Source":{src:?},"Destination":"/run/soda-muse/credentials","RW":false}}]"#
        );
        assert!(
            muse_readonly_mount(good.as_bytes(), false, src, "/run/soda-muse/credentials").is_ok()
        );
        // Decoded tolerantly: unknown fields and ordering do not matter.
        let extra = format!(
            r#"[{{"RW":false,"Extra":1,"Source":{src:?},"Destination":"/run/soda-muse/credentials"}}]"#
        );
        assert!(
            muse_readonly_mount(extra.as_bytes(), false, src, "/run/soda-muse/credentials").is_ok()
        );
        let rw = format!(
            r#"[{{"Source":{src:?},"Destination":"/run/soda-muse/credentials","RW":true}}]"#
        );
        assert!(
            muse_readonly_mount(rw.as_bytes(), false, src, "/run/soda-muse/credentials").is_err()
        );
        assert!(muse_readonly_mount(br#"[]"#, false, src, "/run/soda-muse/credentials").is_err());
        assert!(
            muse_readonly_mount(good.as_bytes(), true, src, "/run/soda-muse/credentials").is_err()
        );
        assert!(muse_readonly_mount(b"nope", false, src, "/run/soda-muse/credentials").is_err());
    }

    fn elf_header(machine: u16) -> Vec<u8> {
        let mut header = vec![0u8; 64];
        header[0..4].copy_from_slice(&[0x7f, b'E', b'L', b'F']);
        header[4] = 2;
        header[5] = 1;
        header[18..20].copy_from_slice(&machine.to_le_bytes());
        header
    }

    #[test]
    fn elf_matrix() {
        assert!(muse_elf(&elf_header(62), "amd64").is_ok());
        assert!(muse_elf(&elf_header(183), "arm64").is_ok());
        assert!(muse_elf(&elf_header(183), "amd64").is_err());
        assert!(muse_elf(&elf_header(62), "arm64").is_err());
        assert!(muse_elf(&elf_header(62)[..63], "amd64").is_err());
        assert!(muse_elf(&[0u8; 64], "amd64").is_err());
        let mut bad = elf_header(62);
        bad[4] = 1;
        assert!(muse_elf(&bad, "amd64").is_err());
        assert!(["amd64", "arm64", "unknown"].contains(&host_go_arch()));
    }

    #[test]
    fn signal_and_root_matrix() {
        for sig in [1, 2, 3, 10, 12, 15, 18, 20] {
            assert!(muse_signal(sig), "{sig}");
        }
        for sig in [0, 9, 19, 64, 65, -1, 1 << 40] {
            assert!(!muse_signal(sig), "{sig}");
        }
        let lease = lease_fixture();
        let binding = lease.binding.clone().unwrap();
        assert!(muse_project_credential_root(&binding));
        let mut nested = binding.clone();
        nested.child_id = CID.to_string();
        nested.credential_root = format!("/run/soda-muse/nested/{IID}/{TID}");
        assert!(muse_project_credential_root(&nested));
        for root in [
            format!("/run/soda-muse/{TID}/../escape"),
            "/run/other/x".to_string(),
            "/run/soda-muse/other".to_string(),
            format!("/run/soda-muse/nested/{IID}"),
        ] {
            let mut bad = binding.clone();
            bad.credential_root = root;
            assert!(
                !muse_project_credential_root(&bad),
                "{:?}",
                bad.credential_root
            );
        }
        assert!(muse_delivery_valid(&Delivery {
            lease: lease.clone(),
            credential: None
        }));
        assert!(!muse_delivery_valid(&Delivery::default()));
        let mut bad = lease.clone();
        bad.provider_id = "codex".to_string();
        assert!(!muse_delivery_valid(&Delivery {
            lease: bad,
            credential: None
        }));
    }

    // ----- argv goldens (byte-exact vs Go `museCommand`) -----

    #[test]
    fn muse_command_goldens() {
        let request = LaunchRequest {
            term: "xterm-256color".to_string(),
            cwd: "/workspace".to_string(),
            args: vec![
                "--provider".to_string(),
                "meta".to_string(),
                "prompt".to_string(),
            ],
            tty: true,
            cols: 80,
            rows: 24,
            ..Default::default()
        };
        let unit = "soda-muse-0123456789abcdef0123456789abcdef.service";
        let path = "/run/soda-muse/0123456789abcdef0123456789abcdef";
        let mut want = vec!["/usr/bin/podman".to_string()];
        want.extend(muse_command_argv(&caller(), &request, unit, path));
        assert_eq!(
            want,
            [
                "/usr/bin/podman", "--remote=false", "exec", "--interactive", "--tty", CID,
                "/usr/bin/systemd-run", "--quiet", "--wait", "--collect", "--service-type=exec",
                "--unit=soda-muse-0123456789abcdef0123456789abcdef.service",
                "--property=KillMode=control-group", "--property=TimeoutStopSec=10",
                "--property=RuntimeMaxSec=43200", "--property=UMask=0077",
                "--setenv=HOME=/home/dev", "--setenv=USER=dev", "--setenv=LOGNAME=dev",
                "--setenv=TERM=xterm-256color",
                "--setenv=XDG_CONFIG_HOME=/run/soda-muse/0123456789abcdef0123456789abcdef/config",
                "--setenv=XDG_STATE_HOME=/run/soda-muse/0123456789abcdef0123456789abcdef/state",
                "--setenv=XDG_CACHE_HOME=/run/soda-muse/0123456789abcdef0123456789abcdef/cache",
                "--setenv=TBH_CREDENTIAL_BACKEND=file", "--setenv=PATH=/usr/local/bin:/usr/bin:/bin",
                "--uid=dev", "--gid=1000", "--working-directory=/workspace",
                "--property=ReadOnlyPaths=/run/soda-muse/0123456789abcdef0123456789abcdef/auth.json",
                "--property=BindReadOnlyPaths=/run/soda-muse/0123456789abcdef0123456789abcdef/auth.json:/run/soda-muse/0123456789abcdef0123456789abcdef/config/muse/auth.json",
                "--pty", "--", "/usr/local/bin/muse", "--soda-exec",
                "/run/soda-muse/0123456789abcdef0123456789abcdef", "/workspace",
                "--provider", "meta", "prompt",
            ]
            .iter()
            .map(|s| s.to_string())
            .collect::<Vec<_>>()
        );
        // Nested variant: no uid/gid/workdir/mounts, nsenter chain, remapped config path.
        let mut nested = caller();
        nested.child =
            "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210".to_string();
        nested.registration = IID.to_string();
        nested.nested_pid = 4242;
        let nested_request = LaunchRequest {
            term: "dumb".to_string(),
            cwd: "/work".to_string(),
            args: vec![
                "exec".to_string(),
                "--provider".to_string(),
                "meta".to_string(),
            ],
            ..Default::default()
        };
        let nested_argv = muse_command_argv(
            &nested,
            &nested_request,
            "soda-muse-bbbb.service",
            "/run/soda-muse/nested/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bbbb",
        );
        let mut want_nested = vec!["/usr/bin/podman".to_string()];
        want_nested.extend(nested_argv.clone());
        assert_eq!(
            want_nested,
            [
                "/usr/bin/podman", "--remote=false", "exec", "--interactive", CID,
                "/usr/bin/systemd-run", "--quiet", "--wait", "--collect", "--service-type=exec",
                "--unit=soda-muse-bbbb.service", "--property=KillMode=control-group",
                "--property=TimeoutStopSec=10", "--property=RuntimeMaxSec=43200",
                "--property=UMask=0077", "--setenv=HOME=/home/dev", "--setenv=USER=dev",
                "--setenv=LOGNAME=dev", "--setenv=TERM=dumb",
                "--setenv=XDG_CONFIG_HOME=/run/soda-muse/nested/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bbbb/config",
                "--setenv=XDG_STATE_HOME=/run/soda-muse/nested/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bbbb/state",
                "--setenv=XDG_CACHE_HOME=/run/soda-muse/nested/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bbbb/cache",
                "--setenv=TBH_CREDENTIAL_BACKEND=file", "--setenv=PATH=/usr/local/bin:/usr/bin:/bin",
                "--pipe", "--", "/usr/bin/nsenter", "--target=4242", "--mount", "--pid", "--uts",
                "--ipc", "--net", "--root", "--setuid=1000", "--setgid=1000", "--",
                "/usr/local/bin/muse", "--soda-exec", "/run/soda-muse/credentials/bbbb", "/work",
                "exec", "--provider", "meta",
            ]
            .iter()
            .map(|s| s.to_string())
            .collect::<Vec<_>>()
        );
        // Explicit HOME wins over the caller home.
        let home_request = LaunchRequest {
            home: "/custom/home".to_string(),
            ..request.clone()
        };
        let home_argv = muse_command_argv(
            &caller(),
            &home_request,
            "soda-muse-cccc.service",
            "/run/soda-muse/cccc",
        );
        assert!(home_argv.contains(&"--setenv=HOME=/custom/home".to_string()));
        assert!(!home_argv.contains(&"--setenv=HOME=/home/dev".to_string()));
    }

    // ----- exec flows -----

    #[test]
    fn inspect_flows() {
        let good = format!(
            "{{\"id\":{CID:?},\"pid\":1234,\"project\":{PID:?},\"running\":true,\"privileged\":false,\"userns\":\"private\"}}"
        );
        let rt = runtime(FakeExec::new(vec![ok(&good)]));
        let out = rt.inspect(CID, deadline()).unwrap();
        assert_eq!(
            (out.project, out.project_pid, out.container),
            (PID.to_string(), 1234, CID.to_string())
        );
        assert_eq!(rt.exec.calls()[0].2[3], MUSE_INSPECT);
        // userns is NOT gated here (Go checks only labels/privilege).
        let host_userns = good.replace("\"userns\":\"private\"", "\"userns\":\"host\"");
        let rt = runtime(FakeExec::new(vec![ok(&host_userns)]));
        assert!(rt.inspect(CID, deadline()).is_ok());
        for body in [
            good.replace("\"running\":true", "\"running\":false"),
            good.replace("\"pid\":1234", "\"pid\":0"),
            good.replace(PID, "nope"),
            good.replace("\"privileged\":false", "\"privileged\":true"),
            good.replace(CID, &"f".repeat(64)),
            "{nope".to_string(),
        ] {
            let rt = runtime(FakeExec::new(vec![ok(&body)]));
            assert_eq!(
                rt.inspect(CID, deadline()).unwrap_err(),
                texec::ERR_DENIED,
                "{body}"
            );
        }
        let rt = runtime(FakeExec::new(vec![err("boom")]));
        assert_eq!(rt.inspect(CID, deadline()).unwrap_err(), texec::ERR_DENIED);
        let rt = runtime(FakeExec::new(vec![Ok(vec![b'x'; 4097])]));
        assert_eq!(rt.inspect(CID, deadline()).unwrap_err(), texec::ERR_DENIED);
    }

    #[test]
    fn actor_account_flows() {
        let rt = runtime(FakeExec::new(vec![ok(
            r#"{"Username":"dev","Uid":1000,"Gid":1000,"HomeDir":"/home/dev"}"#,
        )]));
        assert_eq!(
            rt.actor_account(CID, 7, deadline()).unwrap(),
            ("dev".to_string(), "/home/dev".to_string())
        );
        let calls = rt.exec.calls();
        assert_eq!(
            calls[0].2[4..],
            [
                "/usr/local/bin/muse".to_string(),
                "--soda-account".to_string(),
                "7".to_string()
            ]
        );
        let rt = runtime(FakeExec::new(vec![ok(
            r#"{"Username":"root","HomeDir":"/root"}"#,
        )]));
        assert_eq!(
            rt.actor_account(CID, 7, deadline()).unwrap_err(),
            texec::ERR_DENIED
        );
        let rt = runtime(FakeExec::new(vec![ok("nope")]));
        assert_eq!(
            rt.actor_account(CID, 7, deadline()).unwrap_err(),
            texec::ERR_DENIED
        );
        let rt = runtime(FakeExec::new(vec![err("boom")]));
        assert_eq!(
            rt.actor_account(CID, 7, deadline()).unwrap_err(),
            texec::ERR_DENIED
        );
    }

    #[test]
    fn reserve_execution_flows() {
        let rt = runtime(FakeExec::new(vec![]));
        *rt.hooks.acquire_out.lock().unwrap() = Ok(Lease {
            id: "lease-new".to_string(),
            generation: 4,
            ..Default::default()
        });
        let request = LaunchRequest {
            cwd: "/workspace".to_string(),
            connection_id: "conn-9".to_string(),
            ..Default::default()
        };
        let execution = rt
            .reserve_execution(&caller(), &request, deadline())
            .unwrap();
        assert_eq!(execution.lease.id, "lease-new");
        assert_eq!(execution.binding.generation, 4);
        assert_eq!(execution.binding.project, CID);
        assert_eq!(execution.binding.scope, texec::SCOPE_MUSE_PROJECT);
        assert_eq!(execution.binding.credential_root, execution.path);
        assert!(execution.path.starts_with("/run/soda-muse/"));
        assert_eq!(
            execution.unit,
            format!("soda-muse-{}.service", execution.binding.id)
        );
        assert_eq!(execution.binding.id.len(), 32);
        {
            // Scoped: the nested reserve below re-locks `acquires`.
            let acquires = rt.hooks.acquires.lock().unwrap();
            assert_eq!(acquires.len(), 1);
            assert_eq!(acquires[0].provider_id, "muse");
            assert_eq!(acquires[0].actor_id, 7);
            assert_eq!(acquires[0].connection_id, "conn-1");
            assert_eq!(acquires[0].project_id, PID);
            assert_eq!(acquires[0].kind, "terminal");
            assert!((acquires[0].deadline_secs - (texec::now_unix() + 12 * 3600)).abs() <= 5);
        }
        assert_eq!(
            rt.hooks.selects.lock().unwrap()[0],
            (7, PID.to_string(), "conn-9".to_string())
        );
        // Nested path nests the credential root.
        let mut nested = caller();
        nested.child = CID.to_string();
        nested.registration = IID.to_string();
        let execution = rt.reserve_execution(&nested, &request, deadline()).unwrap();
        assert!(execution
            .path
            .starts_with(&format!("/run/soda-muse/nested/{IID}/")));
        assert_eq!(execution.binding.child_id, CID);
        // Select failure propagates.
        *rt.hooks.select_out.lock().unwrap() = Err(texec::ERR_DENIED.to_string());
        assert_eq!(
            rt.reserve_execution(&caller(), &request, deadline())
                .unwrap_err(),
            texec::ERR_DENIED
        );
    }

    #[test]
    fn stage_flows() {
        let path = format!("/run/soda-muse/{TID}");
        let rt = runtime(FakeExec::new(vec![ok(""); 10]));
        rt.stage(&caller(), &path, "", deadline()).unwrap();
        let calls = rt.exec.calls();
        assert_eq!(calls.len(), 10);
        let guest_tail = |call: &(Vec<u8>, String, Vec<String>)| call.2[4..].to_vec();
        assert_eq!(
            guest_tail(&calls[0]),
            ["/usr/bin/install", "--directory", "--mode=0700", &path]
                .iter()
                .map(|s| s.to_string())
                .collect::<Vec<_>>()
        );
        assert_eq!(
            guest_tail(&calls[1]),
            [
                "/usr/bin/mount",
                "--types=tmpfs",
                "--options=mode=0700,size=4M",
                "tmpfs",
                &path
            ]
            .iter()
            .map(|s| s.to_string())
            .collect::<Vec<_>>()
        );
        assert!(calls[2].2[9].ends_with("/config/muse"));
        assert_eq!(
            &calls[2].2[7..9],
            &["--owner=1000".to_string(), "--group=1000".to_string()]
        );
        assert_eq!(
            calls[3].2[4..],
            [
                "/usr/bin/chmod".to_string(),
                "0711".to_string(),
                path.clone()
            ]
        );
        assert_eq!(calls[4].0, b"{}");
        assert!(calls[4].2[5].ends_with("/auth.json"));
        assert_eq!(
            calls[5].2[4..],
            [
                "/usr/bin/chown".to_string(),
                "1000:1000".to_string(),
                format!("{path}/auth.json")
            ]
        );
        assert_eq!(
            calls[6].2[4..],
            [
                "/usr/bin/chmod".to_string(),
                "0600".to_string(),
                format!("{path}/auth.json")
            ]
        );
        // Config copy runs without --interactive via plain podman exec.
        assert_eq!(calls[7].1, "/usr/bin/podman");
        assert_eq!(
            calls[7].2,
            [
                "--remote=false",
                "exec",
                "--user=1000:1000",
                CID,
                "/usr/local/bin/muse",
                "--soda-copy-config",
                "/home/dev/.config/muse",
                &format!("{path}/config/muse"),
            ]
            .iter()
            .map(|s| s.to_string())
            .collect::<Vec<_>>()
        );
        assert_eq!(
            calls[8].2[4..],
            [
                "/usr/bin/chmod".to_string(),
                "0711".to_string(),
                format!("{path}/config")
            ]
        );
        assert_eq!(
            calls[9].2[4..],
            [
                "/usr/bin/touch".to_string(),
                format!("{path}/config/muse/auth.json")
            ]
        );
        // Explicit config home wins.
        let rt = runtime(FakeExec::new(vec![ok(""); 10]));
        rt.stage(&caller(), &path, "/custom/config", deadline())
            .unwrap();
        assert_eq!(rt.exec.calls()[7].2[6], "/custom/config/muse");
        // Nested stage skips the tmpfs mount and reads config through nsenter.
        let mut nested = caller();
        nested.child = CID.to_string();
        nested.registration = IID.to_string();
        nested.nested_pid = 4242;
        let nested_path = format!("/run/soda-muse/nested/{IID}/{TID}");
        let rt = runtime(FakeExec::new(vec![
            ok(""),                                           // install dir
            ok(""),                                           // install config dir
            ok(""),                                           // chmod path
            ok(""),                                           // dd auth
            ok(""),                                           // chown auth
            ok(""),                                           // chmod auth
            ok(r#"{"settings.json":"e30=","other":"e30="}"#), // nsenter read-config
            ok(""),                                           // dd settings
            ok(""),                                           // chown settings
            ok(""),                                           // chmod settings
            ok(""),                                           // chmod config
            ok(""),                                           // ln auth target
        ]));
        rt.stage(&nested, &nested_path, "", deadline()).unwrap();
        let calls = rt.exec.calls();
        assert_eq!(calls.len(), 12);
        assert_eq!(calls[6].2[4], "/usr/bin/nsenter");
        assert!(calls[6].2.contains(&"--target=4242".to_string()));
        assert_eq!(calls[7].0, b"{}");
        assert!(calls[7].2[5].ends_with("/config/muse/settings.json"));
        assert_eq!(
            calls[10].2[4..],
            [
                "/usr/bin/chmod".to_string(),
                "0711".to_string(),
                format!("{nested_path}/config")
            ]
        );
        assert_eq!(
            calls[11].2[4..],
            [
                "/usr/bin/ln".to_string(),
                "--symbolic".to_string(),
                "../../auth.json".to_string(),
                format!("{nested_path}/config/muse/auth.json")
            ]
        );
        // Mount failure propagates raw.
        let rt = runtime(FakeExec::new(vec![ok(""), err("exit status 32")]));
        assert_eq!(
            rt.stage(&caller(), &path, "", deadline()).unwrap_err(),
            "exit status 32"
        );
    }

    #[test]
    fn deliver_execution_flows() {
        let lease = lease_fixture();
        let mut execution = MuseExecution {
            caller: caller(),
            request: LaunchRequest::default(),
            lease: lease.clone(),
            binding: lease.binding.clone().unwrap(),
            path: format!("/run/soda-muse/{TID}"),
            unit: format!("soda-muse-{TID}.service"),
        };
        execution.binding.invocation_id = String::new();
        let rt = runtime(FakeExec::new(vec![
            ok("active\n"),
            ok(&format!("{IID}\n")),
            ok(""),
            ok(""),
        ]));
        *rt.hooks.attach_out.lock().unwrap() = Ok(Delivery {
            lease: lease.clone(),
            credential: Some(b"{\"k\":1}".to_vec()),
        });
        rt.deliver_execution(&mut execution, deadline()).unwrap();
        assert_eq!(execution.binding.invocation_id, IID);
        let calls = rt.exec.calls();
        assert_eq!(calls.len(), 4);
        assert_eq!(calls[2].0, b"{\"k\":1}");
        assert!(calls[2].2[5].ends_with("/auth.json"));
        assert!(calls[3].2[5].ends_with("/ready"));
        let attaches = rt.hooks.attaches.lock().unwrap();
        assert_eq!(attaches[0].0, "lease-m");
        assert_eq!(attaches[0].1.invocation_id, IID);
        // Malformed invocation denies before attach.
        let mut execution = execution.clone();
        execution.binding.invocation_id = String::new();
        let rt = runtime(FakeExec::new(vec![ok("active\n"), ok("short\n")]));
        assert_eq!(
            rt.deliver_execution(&mut execution, deadline())
                .unwrap_err(),
            texec::ERR_DENIED
        );
        assert!(rt.hooks.attaches.lock().unwrap().is_empty());
        // Invalid credential denies after attach.
        let mut execution = execution.clone();
        execution.binding.invocation_id = String::new();
        let rt = runtime(FakeExec::new(vec![ok("active\n"), ok(&format!("{IID}\n"))]));
        *rt.hooks.attach_out.lock().unwrap() = Ok(Delivery {
            lease: lease.clone(),
            credential: Some(b"nope".to_vec()),
        });
        assert_eq!(
            rt.deliver_execution(&mut execution, deadline())
                .unwrap_err(),
            texec::ERR_DENIED
        );
        // dd failure propagates raw.
        let mut execution = execution.clone();
        execution.binding.invocation_id = String::new();
        let rt = runtime(FakeExec::new(vec![
            ok("active\n"),
            ok(&format!("{IID}\n")),
            err("exit status 1"),
        ]));
        *rt.hooks.attach_out.lock().unwrap() = Ok(Delivery {
            lease: lease.clone(),
            credential: Some(b"{}".to_vec()),
        });
        assert_eq!(
            rt.deliver_execution(&mut execution, deadline())
                .unwrap_err(),
            "exit status 1"
        );
        // A unit that never activates denies (tight deadline, no 5s wait).
        let mut execution = execution.clone();
        execution.binding.invocation_id = String::new();
        let rt = runtime(FakeExec::new(vec![ok("inactive\n")]));
        let past = Instant::now() - Duration::from_secs(1);
        assert!(rt.deliver_execution(&mut execution, past).is_err());
    }

    #[test]
    fn stop_execution_flows() {
        let lease = lease_fixture();
        let binding = lease.binding.clone().unwrap();
        // Full stop with custody: stop, inactive, exists, rm state,
        // test state, umount, rm path.
        let rt = runtime(FakeExec::new(vec![
            ok(""),
            ok("inactive\n"),
            ok(""),
            ok(""),
            ok(""),
            ok(""),
            ok(""),
        ]));
        rt.stop_execution(&binding, 7, "lease-m", true, deadline())
            .unwrap();
        let calls = rt.exec.calls();
        assert_eq!(calls.len(), 7);
        assert_eq!(
            calls[0].2[4..],
            [
                "/usr/bin/systemctl".to_string(),
                "stop".to_string(),
                format!("soda-muse-{TID}.service")
            ]
        );
        assert_eq!(
            calls[1].2[4..8],
            [
                "/usr/bin/systemctl".to_string(),
                "show".to_string(),
                "--property=ActiveState".to_string(),
                "--value".to_string()
            ]
        );
        assert_eq!(
            calls[2].2[1..4],
            [
                "container".to_string(),
                "exists".to_string(),
                CID.to_string()
            ]
        );
        assert!(calls[3].2.contains(&"--user=1000:1000".to_string()));
        assert_eq!(
            calls[5].2[4..],
            [
                "/usr/bin/umount".to_string(),
                format!("/run/soda-muse/{TID}")
            ]
        );
        assert_eq!(
            rt.hooks.end_calls.lock().unwrap().as_slice(),
            &[(7, "lease-m".to_string())]
        );
        // Unit still active is uncertain.
        let rt = runtime(FakeExec::new(vec![ok(""), ok("active\n")]));
        assert_eq!(
            rt.stop_execution(&binding, 7, "lease-m", true, deadline())
                .unwrap_err(),
            texec::ERR_UNCERTAIN
        );
        // Failed stop with a gone cgroup still retires.
        let rt = runtime(FakeExec::new(vec![
            err("exit status 1"),
            ok("inactive\n"),
            ok(""),
            ok(""),
            ok(""),
            ok(""),
            ok(""),
            ok(""),
        ]));
        rt.stop_execution(&binding, 7, "lease-m", false, deadline())
            .unwrap();
        assert_eq!(rt.exec.calls().len(), 8);
        assert!(rt.hooks.end_calls.lock().unwrap().is_empty());
        // Failed stop with a live cgroup is uncertain.
        let rt = runtime(FakeExec::new(vec![
            err("exit status 1"),
            ok("inactive\n"),
            err("exit status 1"),
        ]));
        assert_eq!(
            rt.stop_execution(&binding, 7, "lease-m", false, deadline())
                .unwrap_err(),
            texec::ERR_UNCERTAIN
        );
        // Removed container skips the rm/test pair.
        let rt = runtime(FakeExec::new(vec![
            ok(""),
            ok("inactive\n"),
            err("exit status 1"),
            ok(""),
            ok(""),
        ]));
        rt.stop_execution(&binding, 7, "lease-m", false, deadline())
            .unwrap();
        assert_eq!(rt.exec.calls().len(), 5);
        // Retire failure is uncertain.
        let rt = runtime(FakeExec::new(vec![
            ok(""),
            ok("inactive\n"),
            ok(""),
            ok(""),
            err("exit status 1"),
        ]));
        assert_eq!(
            rt.stop_execution(&binding, 7, "lease-m", false, deadline())
                .unwrap_err(),
            texec::ERR_UNCERTAIN
        );
        // Nested bindings skip the tmpfs unmount and wrap state podman.
        let mut nested = binding.clone();
        nested.child_id = "f".repeat(64);
        nested.credential_root = format!("/run/soda-muse/nested/{IID}/{TID}");
        let rt = runtime(FakeExec::new(vec![
            ok(""),
            ok("inactive\n"),
            ok(""),
            ok(""),
            ok(""),
            ok(""),
        ]));
        rt.stop_execution(&nested, 7, "lease-m", false, deadline())
            .unwrap();
        let calls = rt.exec.calls();
        assert_eq!(calls.len(), 6);
        assert_eq!(
            calls[2].2[4..8],
            [
                "/usr/bin/podman".to_string(),
                "--remote=false".to_string(),
                "container".to_string(),
                "exists".to_string()
            ]
        );
        assert_eq!(calls[2].2[8], "f".repeat(64));
        assert!(!calls
            .iter()
            .any(|c| c.2.contains(&"/usr/bin/umount".to_string())));
    }

    #[test]
    fn retire_mount_matrix() {
        let lease = lease_fixture();
        let binding = lease.binding.clone().unwrap();
        // umount failure + vanished path retires.
        let rt = runtime(FakeExec::new(vec![
            ok(""),
            ok("inactive\n"),
            ok(""),
            ok(""),
            ok(""),
            err("exit status 32"),
            ok(""),
            ok(""),
        ]));
        rt.stop_execution(&binding, 7, "lease-m", false, deadline())
            .unwrap();
        assert_eq!(rt.exec.calls().len(), 8);
        // umount + test failures + mountpoint exit 32 retires.
        let rt = runtime(FakeExec::new(vec![
            ok(""),
            ok("inactive\n"),
            ok(""),
            ok(""),
            ok(""),
            err("exit status 1"),
            err("exit status 1"),
            err("exit status 32"),
            ok(""),
        ]));
        rt.stop_execution(&binding, 7, "lease-m", false, deadline())
            .unwrap();
        assert_eq!(rt.exec.calls().len(), 9);
        // A live mountpoint (exit 0) is uncertain.
        let rt = runtime(FakeExec::new(vec![
            ok(""),
            ok("inactive\n"),
            ok(""),
            ok(""),
            ok(""),
            err("exit status 1"),
            err("exit status 1"),
            ok(""),
        ]));
        assert_eq!(
            rt.stop_execution(&binding, 7, "lease-m", false, deadline())
                .unwrap_err(),
            texec::ERR_UNCERTAIN
        );
        assert_eq!(rt.exec.calls().len(), 8);
    }

    #[test]
    fn validate_and_ops_matrix() {
        let lease = lease_fixture();
        let binding = lease.binding.clone().unwrap();
        let rt = runtime(FakeExec::new(vec![ok("active\n")]));
        assert!(rt.validate_muse_binding(&binding, deadline()).is_ok());
        let rt = runtime(FakeExec::new(vec![ok("inactive\n")]));
        assert_eq!(
            rt.validate_muse_binding(&binding, deadline()).unwrap_err(),
            texec::ERR_STALE
        );
        let rt = runtime(FakeExec::new(vec![err("boom")]));
        assert_eq!(
            rt.validate_muse_binding(&binding, deadline()).unwrap_err(),
            texec::ERR_STALE
        );
        let rt = runtime(FakeExec::new(vec![]));
        assert_eq!(
            rt.validate_muse_binding(&Binding::default(), deadline())
                .unwrap_err(),
            texec::ERR_DENIED
        );
        assert!(rt.exec.calls().is_empty());
        // Ops dispatch.
        let delivery = Delivery {
            lease: lease.clone(),
            credential: None,
        };
        let rt = runtime(FakeExec::new(vec![ok(&format!("{IID}\n")), ok("active\n")]));
        assert_eq!(
            rt.muse_operation("validate", &delivery, deadline())
                .unwrap(),
            delivery
        );
        let rt = runtime(FakeExec::new(vec![
            ok(&format!("{IID}\n")),
            ok(""),
            ok("inactive\n"),
            err("exit status 1"),
            ok(""),
            ok(""),
        ]));
        assert_eq!(
            rt.muse_operation("stop", &delivery, deadline()).unwrap(),
            delivery
        );
        assert!(rt.hooks.end_calls.lock().unwrap().is_empty()); // no custody on broker stop
                                                                // Invocation mismatch is stale, checked before the action.
        let rt = runtime(FakeExec::new(vec![ok(&format!("{}\n", "b".repeat(32)))]));
        assert_eq!(
            rt.muse_operation("validate", &delivery, deadline())
                .unwrap_err(),
            texec::ERR_STALE
        );
        // Empty observation is tolerated (unit may be gone).
        let rt = runtime(FakeExec::new(vec![ok("\n"), ok("active\n")]));
        assert!(rt.muse_operation("validate", &delivery, deadline()).is_ok());
        // Unknown actions deny after the invocation check.
        let rt = runtime(FakeExec::new(vec![ok(&format!("{IID}\n"))]));
        assert_eq!(
            rt.muse_operation("launch", &delivery, deadline())
                .unwrap_err(),
            texec::ERR_DENIED
        );
        assert_eq!(rt.exec.calls().len(), 1);
        // Malformed deliveries never call out.
        let rt = runtime(FakeExec::new(vec![]));
        assert_eq!(
            rt.muse_operation("validate", &Delivery::default(), deadline())
                .unwrap_err(),
            texec::ERR_DENIED
        );
        let mut other = lease.clone();
        other.binding.as_mut().unwrap().scope = "other".to_string();
        assert_eq!(
            rt.muse_operation(
                "validate",
                &Delivery {
                    lease: other,
                    credential: None
                },
                deadline()
            )
            .unwrap_err(),
            texec::ERR_DENIED
        );
        assert!(rt.exec.calls().is_empty());
    }

    #[test]
    fn state_container_matrix() {
        let lease = lease_fixture();
        let binding = lease.binding.clone().unwrap();
        assert_eq!(state_container(&binding).unwrap(), CID);
        let mut nested = binding.clone();
        nested.child_id = "f".repeat(64);
        assert_eq!(state_container(&nested).unwrap(), "f".repeat(64));
        let mut bad = binding.clone();
        bad.id = "short".to_string();
        assert!(state_container(&bad).is_err());
        bad = binding.clone();
        bad.uid = -1;
        assert!(state_container(&bad).is_err());
        bad = binding.clone();
        bad.project = "short".to_string();
        assert!(state_container(&bad).is_err());
    }

    #[test]
    fn config_view_matrix() {
        let view =
            decode_config_view(br#"{"settings.json":"e30=","trust.json":[123,125]}"#).unwrap();
        assert_eq!(view["settings.json"], b"{}");
        assert_eq!(view["trust.json"], b"{}");
        assert!(decode_config_view(b"[]").is_err());
        assert!(decode_config_view(br#"{"a":7}"#).is_err());
        assert!(decode_config_view(br#"{"a":"!!!"}"#).is_err());
        assert!(decode_config_view(br#"{"a":[256]}"#).is_err());
        assert!(decode_config_view(b"nope").is_err());
    }

    #[test]
    fn control_execution_flows() {
        let lease = lease_fixture();
        let execution = MuseExecution {
            caller: caller(),
            request: LaunchRequest::default(),
            lease,
            binding: lease_fixture().binding.clone().unwrap(),
            path: format!("/run/soda-muse/{TID}"),
            unit: format!("soda-muse-{TID}.service"),
        };
        // Signal path pins the guest kill argv.
        let rt = runtime(FakeExec::new(vec![ok("")]));
        rt.control_execution(
            &execution,
            -1,
            None,
            &LaunchControl {
                signal: 15,
                cols: 0,
                rows: 0,
            },
            deadline(),
        )
        .unwrap();
        let calls = rt.exec.calls();
        assert_eq!(
            calls[0].2[4..],
            [
                "/usr/bin/systemctl".to_string(),
                "kill".to_string(),
                "--kill-whom=all".to_string(),
                "--signal=15".to_string(),
                format!("soda-muse-{TID}.service")
            ]
        );
        // Mixed signal+resize and bad signals deny without calling out.
        let rt = runtime(FakeExec::new(vec![]));
        assert!(rt
            .control_execution(
                &execution,
                -1,
                None,
                &LaunchControl {
                    signal: 15,
                    cols: 80,
                    rows: 24
                },
                deadline()
            )
            .is_err());
        assert!(rt
            .control_execution(
                &execution,
                -1,
                None,
                &LaunchControl {
                    signal: 9,
                    cols: 0,
                    rows: 0
                },
                deadline()
            )
            .is_err());
        assert!(rt.exec.calls().is_empty());
        // Resize requires a TTY session.
        assert!(rt
            .control_execution(
                &execution,
                -1,
                None,
                &LaunchControl {
                    signal: 0,
                    cols: 80,
                    rows: 24
                },
                deadline()
            )
            .is_err());
        // Resize over a real PTY applies the window size.
        let tty = MuseExecution {
            request: LaunchRequest {
                tty: true,
                cols: 80,
                rows: 24,
                ..Default::default()
            },
            ..execution.clone()
        };
        let (master, slave) = open_pty_pair();
        let rt = runtime(FakeExec::new(vec![]));
        rt.control_execution(
            &tty,
            master,
            None,
            &LaunchControl {
                signal: 0,
                cols: 100,
                rows: 40,
            },
            deadline(),
        )
        .unwrap();
        assert_eq!(pty_size(master), (40, 100));
        unsafe {
            libc::close(master);
            libc::close(slave);
        }
        // Resize on a non-TTY denies.
        let fds = open_pipe_pair();
        let rt = runtime(FakeExec::new(vec![]));
        assert!(rt
            .control_execution(
                &tty,
                fds.0,
                None,
                &LaunchControl {
                    signal: 0,
                    cols: 80,
                    rows: 24
                },
                deadline()
            )
            .is_err());
        unsafe {
            libc::close(fds.0);
            libc::close(fds.1);
        }
    }

    fn open_pty_pair() -> (RawFd, RawFd) {
        let mut master = -1;
        let mut slave = -1;
        // SAFETY: openpty with default termios/winsize.
        let result = unsafe {
            libc::openpty(
                &mut master,
                &mut slave,
                std::ptr::null_mut(),
                std::ptr::null_mut(),
                std::ptr::null_mut(),
            )
        };
        assert_eq!(result, 0, "openpty unavailable");
        (master, slave)
    }

    fn pty_size(fd: RawFd) -> (u16, u16) {
        let mut ws: libc::winsize = unsafe { std::mem::zeroed() };
        // SAFETY: TIOCGWINSZ with a valid winsize pointer.
        let result = unsafe { libc::ioctl(fd, libc::TIOCGWINSZ, &mut ws) };
        assert_eq!(result, 0);
        (ws.ws_row, ws.ws_col)
    }

    fn open_pipe_pair() -> (RawFd, RawFd) {
        let mut fds = [-1, -1];
        // SAFETY: pipe with a valid fd pair.
        assert_eq!(unsafe { libc::pipe(fds.as_mut_ptr()) }, 0);
        (fds[0], fds[1])
    }

    #[test]
    fn verify_guest_binary_flows() {
        let machine = if host_go_arch() == "arm64" {
            183u16
        } else {
            62u16
        };
        let header = elf_header(machine);
        let rt = runtime(FakeExec::new(vec![
            ok(&format!("{PIN}  /usr/local/libexec/soda/muse\n")),
            Ok(header.clone()),
            ok(""),
        ]));
        rt.verify_guest_binary(CID, "", deadline()).unwrap();
        let calls = rt.exec.calls();
        assert_eq!(
            calls[0].2[4..],
            [
                "/usr/bin/sha256sum".to_string(),
                "/usr/local/libexec/soda/muse".to_string()
            ]
        );
        assert_eq!(
            calls[1].2[4..],
            [
                "/usr/bin/head".to_string(),
                "--bytes=64".to_string(),
                "/usr/local/libexec/soda/muse".to_string()
            ]
        );
        assert_eq!(
            calls[2].2[4..],
            [
                "/usr/local/bin/muse".to_string(),
                "--soda-check".to_string(),
                "1.0".to_string()
            ]
        );
        // Unpinned digest denies before any exec.
        let rt = MuseRuntime::new(
            FakeExec::new(vec![]),
            FakeHooks::new(),
            "1.0".to_string(),
            "short".to_string(),
        );
        assert_eq!(
            rt.verify_guest_binary(CID, "", deadline()).unwrap_err(),
            texec::ERR_DENIED
        );
        // Digest mismatch denies.
        let rt = runtime(FakeExec::new(vec![ok(&format!(
            "{}  /usr/local/libexec/soda/muse\n",
            "f".repeat(64)
        ))]));
        assert_eq!(
            rt.verify_guest_binary(CID, "", deadline()).unwrap_err(),
            texec::ERR_DENIED
        );
        // Bad ELF header denies.
        let rt = runtime(FakeExec::new(vec![
            ok(&format!("{PIN}  /usr/local/libexec/soda/muse\n")),
            Ok(vec![0u8; 64]),
        ]));
        assert_eq!(
            rt.verify_guest_binary(CID, "", deadline()).unwrap_err(),
            texec::ERR_DENIED
        );
        // Empty version denies after the header check.
        let rt = MuseRuntime::new(
            FakeExec::new(vec![
                ok(&format!("{PIN}  /usr/local/libexec/soda/muse\n")),
                Ok(header),
            ]),
            FakeHooks::new(),
            String::new(),
            PIN.to_string(),
        );
        assert_eq!(
            rt.verify_guest_binary(CID, "", deadline()).unwrap_err(),
            texec::ERR_DENIED
        );
        // Version-handshake failure propagates raw.
        let rt = runtime(FakeExec::new(vec![
            ok(&format!("{PIN}  /usr/local/libexec/soda/muse\n")),
            Ok(elf_header(machine)),
            err("exit status 1"),
        ]));
        assert_eq!(
            rt.verify_guest_binary(CID, "", deadline()).unwrap_err(),
            "exit status 1"
        );
        // Nested children route through the parent podman.
        let child = "f".repeat(64);
        let rt = runtime(FakeExec::new(vec![
            ok(&format!("{PIN}  /usr/local/libexec/soda/muse\n")),
            Ok(elf_header(machine)),
            ok(""),
        ]));
        rt.verify_guest_binary(CID, &child, deadline()).unwrap();
        let calls = rt.exec.calls();
        assert_eq!(
            calls[0].2[4..8],
            [
                "/usr/bin/podman".to_string(),
                "--remote=false".to_string(),
                "exec".to_string(),
                child
            ]
        );
    }

    // ----- socket protocol -----

    fn seqpacket_pair() -> (RawFd, RawFd) {
        let mut fds = [-1, -1];
        // SAFETY: socketpair with a valid fd pair.
        let result =
            unsafe { libc::socketpair(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0, fds.as_mut_ptr()) };
        assert_eq!(result, 0, "socketpair unavailable");
        (fds[0], fds[1])
    }

    fn send_with_fds(fd: RawFd, body: &[u8], fds: &[RawFd]) {
        unsafe {
            let mut iov = libc::iovec {
                iov_base: body.as_ptr() as *mut libc::c_void,
                iov_len: body.len(),
            };
            let mut hdr: libc::msghdr = std::mem::zeroed();
            hdr.msg_iov = &mut iov;
            hdr.msg_iovlen = 1;
            let cmsg_len = libc::CMSG_SPACE((fds.len() * 4) as _) as usize;
            let mut control = vec![0u8; cmsg_len.max(1)];
            if !fds.is_empty() {
                hdr.msg_control = control.as_mut_ptr() as *mut libc::c_void;
                hdr.msg_controllen = control.len() as _;
                let cmsg = libc::CMSG_FIRSTHDR(&hdr);
                assert!(!cmsg.is_null());
                (*cmsg).cmsg_level = libc::SOL_SOCKET;
                (*cmsg).cmsg_type = libc::SCM_RIGHTS;
                (*cmsg).cmsg_len = libc::CMSG_LEN((fds.len() * 4) as _) as _;
                let data = libc::CMSG_DATA(cmsg) as *mut RawFd;
                for (i, fd) in fds.iter().enumerate() {
                    *data.add(i) = *fd;
                }
            }
            let n = libc::sendmsg(fd, &hdr, 0);
            assert_eq!(n as usize, body.len(), "sendmsg short");
        }
    }

    #[test]
    fn peer_attestation() {
        let (a, b) = seqpacket_pair();
        let peer = muse_peer_from_fd(a).expect("kernel must offer SO_PEERPIDFD on 6.5+");
        assert_eq!(peer.pid, std::process::id() as i32);
        assert_eq!(peer.uid, unsafe { libc::geteuid() });
        assert_eq!(peer.gid, unsafe { libc::getegid() });
        assert!(peer.pidfd >= 0);
        assert!(muse_peer_alive(&peer));
        unsafe {
            libc::close(peer.pidfd);
            libc::close(a);
            libc::close(b);
        }
        // A stale pidfd (POLLNVAL/POLLERR) reads non-alive.
        let (c, d) = seqpacket_pair();
        let stale = muse_peer_from_fd(c).unwrap();
        let stale_pidfd = stale.pidfd;
        unsafe {
            libc::close(c);
            libc::close(d);
            libc::close(stale_pidfd);
        }
        // No fd allocation happens between the close and the poll, so the
        // number still refers to nothing.
        assert!(!muse_peer_alive(&stale));
    }

    #[test]
    fn request_parsing_matrix() {
        // Register without descriptors.
        let (a, b) = seqpacket_pair();
        let body = format!(
            "{{\"register\":{{\"child_id\":{CID:?},\"actor_id\":\"7\",\"registration_id\":{TID:?},\"muse\":true}}}}"
        );
        send_with_fds(b, body.as_bytes(), &[]);
        let received = muse_request_from_fd(a).unwrap();
        assert!(received.request.register.is_some());
        assert!(received.files.iter().all(|f| f.is_none()));
        unsafe {
            libc::close(a);
            libc::close(b);
        }
        // Launch with three descriptors.
        let (a, b) = seqpacket_pair();
        let (p1, p2) = open_pipe_pair();
        let (p3, p4) = open_pipe_pair();
        let (p5, p6) = open_pipe_pair();
        send_with_fds(b, br#"{"cwd":"/w","tty":false}"#, &[p1, p3, p5]);
        let received = muse_request_from_fd(a).unwrap();
        assert_eq!(received.request.cwd, "/w");
        assert!(received.files.iter().all(|f| f.is_some()));
        drop(received);
        unsafe {
            libc::close(a);
            libc::close(b);
            libc::close(p1);
            libc::close(p2);
            libc::close(p3);
            libc::close(p4);
            libc::close(p5);
            libc::close(p6);
        }
        // Launch without descriptors is invalid.
        let (a, b) = seqpacket_pair();
        send_with_fds(b, br#"{"cwd":"/w"}"#, &[]);
        assert_eq!(
            muse_request_from_fd(a).unwrap_err(),
            "invalid launch descriptors"
        );
        unsafe {
            libc::close(a);
            libc::close(b);
        }
        // Bad JSON and invalid requests fail.
        let (a, b) = seqpacket_pair();
        send_with_fds(b, b"nope", &[]);
        assert!(muse_request_from_fd(a).is_err());
        unsafe {
            libc::close(a);
            libc::close(b);
        }
        let (a, b) = seqpacket_pair();
        send_with_fds(b, br#"{"cwd":"relative"}"#, &[]);
        assert!(muse_request_from_fd(a).is_err());
        unsafe {
            libc::close(a);
            libc::close(b);
        }
    }

    #[test]
    fn command_exit_matrix() {
        use std::os::unix::process::ExitStatusExt;
        let exited = |code| {
            muse_command_exit(Ok(std::process::Command::new("/bin/sh")
                .arg("-c")
                .arg(format!("exit {code}"))
                .status()
                .unwrap()))
        };
        assert_eq!(
            exited(0),
            LaunchExit {
                code: 0,
                error: String::new()
            }
        );
        assert_eq!(
            exited(3),
            LaunchExit {
                code: 3,
                error: String::new()
            }
        );
        let signaled = std::process::Command::new("/bin/sh")
            .arg("-c")
            .arg("kill -TERM $$")
            .status()
            .unwrap();
        assert!(signaled.signal() == Some(15));
        assert_eq!(
            muse_command_exit(Ok(signaled)),
            LaunchExit {
                code: 143,
                error: String::new()
            }
        );
        assert_eq!(
            muse_command_exit(Err(std::io::Error::new(
                std::io::ErrorKind::BrokenPipe,
                "gone"
            ))),
            LaunchExit {
                code: 1,
                error: String::new()
            }
        );
    }

    #[test]
    fn json_split_matrix() {
        assert_eq!(split_json_object(br#"{"a":1}"#), Some(7));
        assert_eq!(split_json_object(br#"{"a":{"b":[1,2]}}{"c":3}"#), Some(17));
        assert_eq!(split_json_object(br#"{"a":"}{"}"#), Some(10)); // braces inside strings ignored
        assert_eq!(split_json_object(br#"{"a":"}"}"#), Some(9));
        assert_eq!(split_json_object(br#"{"a":"\"}"}"#), Some(11));
        assert_eq!(split_json_object(br#"{"a":"}"#,), None); // truncated
        assert_eq!(split_json_object(b"{]"), None);
        assert_eq!(split_json_object(b""), None);
        assert_eq!(split_json_object(b"   "), None);
        assert_eq!(split_json_object(br#"{"signal":15} {"cols":1}"#), Some(13));
    }

    #[test]
    fn control_loop_matrix() {
        use std::sync::Arc;
        let lease = lease_fixture();
        let execution = MuseExecution {
            caller: caller(),
            request: LaunchRequest {
                tty: true,
                cols: 80,
                rows: 24,
                ..Default::default()
            },
            lease,
            binding: lease_fixture().binding.clone().unwrap(),
            path: format!("/run/soda-muse/{TID}"),
            unit: format!("soda-muse-{TID}.service"),
        };
        // One signal message reaches the guest, then EOF ends the loop.
        let rt = runtime(FakeExec::new(vec![ok("")]));
        let service = Arc::new(MuseLaunch::new(rt));
        let (a, b) = seqpacket_pair();
        let (master, slave) = open_pty_pair();
        let peer_service = service.clone();
        let execution_clone = execution.clone();
        let worker = std::thread::spawn(move || {
            peer_service.control_loop(
                b,
                master,
                424242,
                &execution_clone,
                Instant::now() + Duration::from_secs(30),
            );
            unsafe {
                libc::close(b);
                libc::close(master);
            }
        });
        unsafe {
            let body = br#"{"signal":15}"#;
            assert_eq!(
                libc::send(
                    a,
                    body.as_ptr() as *const libc::c_void,
                    body.len(),
                    libc::MSG_NOSIGNAL
                ) as usize,
                body.len()
            );
        }
        // Wait for the guest call, then close to end the loop.
        for _ in 0..100 {
            if !service.runtime.exec.calls().is_empty() {
                break;
            }
            std::thread::sleep(Duration::from_millis(10));
        }
        unsafe {
            libc::close(a);
            libc::close(slave);
        }
        worker.join().unwrap();
        let calls = service.runtime.exec.calls();
        assert_eq!(calls.len(), 1);
        assert!(calls[0].2.contains(&"--signal=15".to_string()));
        // Garbage ends the loop without guest calls.
        let rt = runtime(FakeExec::new(vec![]));
        let service = Arc::new(MuseLaunch::new(rt));
        let (a, b) = seqpacket_pair();
        let peer_service = service.clone();
        let worker = std::thread::spawn(move || {
            peer_service.control_loop(
                b,
                -1,
                424242,
                &execution,
                Instant::now() + Duration::from_secs(30),
            );
            unsafe {
                libc::close(b);
            }
        });
        unsafe {
            let body = b"nope";
            libc::send(
                a,
                body.as_ptr() as *const libc::c_void,
                body.len(),
                libc::MSG_NOSIGNAL,
            );
            libc::close(a);
        }
        worker.join().unwrap();
        assert!(service.runtime.exec.calls().is_empty());
    }

    #[test]
    fn serve_shutdown_and_listener_setup() {
        use std::sync::Arc;
        // Listener dir setup pins.
        let dir = test_tmp("listener");
        let socket = dir.join("launch.sock");
        prepare_muse_listener_dir(socket.to_str().unwrap()).unwrap();
        // Missing parents are created.
        let fresh = test_tmp("fresh");
        let nested = fresh.join("a").join("b").join("launch.sock");
        prepare_muse_listener_dir(nested.to_str().unwrap()).unwrap();
        assert!(fresh.join("a").join("b").is_dir());
        // Non-empty dir fails even when the socket path itself is absent;
        // an occupied socket inside a non-empty dir reports emptiness
        // first, since Go checks ReadDir before Lstat.
        std::fs::write(dir.join("junk"), b"x").unwrap();
        assert_eq!(
            prepare_muse_listener_dir(socket.to_str().unwrap()).unwrap_err(),
            "muse interface directory must be empty before launch service startup"
        );
        std::fs::write(&socket, b"x").unwrap();
        assert_eq!(
            prepare_muse_listener_dir(socket.to_str().unwrap()).unwrap_err(),
            "muse interface directory must be empty before launch service startup"
        );
        std::fs::remove_file(dir.join("junk")).unwrap();
        std::fs::remove_file(&socket).unwrap();
        // Accept loop starts and stops cleanly on a short socket path.
        let path = dir.join("s").to_str().unwrap().to_string();
        let listen_fd = unsafe { libc::socket(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0) };
        assert!(listen_fd >= 0);
        let mut addr: libc::sockaddr_un = unsafe { std::mem::zeroed() };
        addr.sun_family = libc::AF_UNIX as _;
        let bytes = path.as_bytes();
        addr.sun_path[..bytes.len()]
            .copy_from_slice(unsafe { std::mem::transmute::<&[u8], &[libc::c_char]>(bytes) });
        let addr_len =
            (std::mem::size_of::<libc::sa_family_t>() + bytes.len() + 1) as libc::socklen_t;
        unsafe {
            assert_eq!(
                libc::bind(
                    listen_fd,
                    &addr as *const _ as *const libc::sockaddr,
                    addr_len
                ),
                0
            );
            assert_eq!(libc::listen(listen_fd, 8), 0);
        }
        let service = Arc::new(MuseLaunch::new(runtime(FakeExec::new(vec![]))));
        let shutdown = Arc::new(AtomicBool::new(false));
        let peer = service.clone();
        let flag = shutdown.clone();
        let worker = std::thread::spawn(move || peer.serve(listen_fd, &flag));
        std::thread::sleep(Duration::from_millis(50));
        shutdown.store(true, Ordering::SeqCst);
        assert!(worker.join().unwrap().is_ok());
        unsafe {
            libc::close(listen_fd);
        }
        std::fs::remove_file(&path).ok();
    }

    #[test]
    fn muse_host_environment_filters_meta_key() {
        let _guard = ENV_LOCK.lock().unwrap();
        unsafe {
            std::env::set_var("META_API_KEY", "secret");
            std::env::set_var("T26_MUSE_PROBE", "kept");
        }
        let env = muse_host_environment();
        assert!(!env.iter().any(|(k, _)| k == "META_API_KEY"));
        assert!(env
            .iter()
            .any(|(k, v)| k == "T26_MUSE_PROBE" && v == "kept"));
        // No secrets leak into error strings: the key never appears.
        unsafe {
            std::env::remove_var("META_API_KEY");
            std::env::remove_var("T26_MUSE_PROBE");
        }
    }
}
