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
use std::time::{Duration, Instant};

use crate::domain;
use crate::json::{self, Kind, Spec, Value};
use crate::project::Executor;
#[cfg(test)]
use crate::terminal::Lease;
use crate::terminal::{self, AcquireRequest, Binding, Delivery};

mod wire;

pub use self::wire::{
    LaunchControl, LaunchExit, LaunchRequest, NestedRegistration, MUSE_CONNECTION_SETTING,
    MUSE_LAUNCH_SOCKET,
};

mod args;

pub use self::args::muse_arguments;

mod connection;

pub use self::connection::{muse_connection_authorized, select_muse_connection, MuseConnection};

mod runtime_types;

pub use self::runtime_types::{
    MuseCaller, MuseExecution, MuseHooks, MuseNested, MusePeer, MuseRuntime,
};

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
    Err(terminal::err_denied())
}

/// `museMappedUID`: inside-UID for a host UID from a uid/gid map.
pub fn muse_mapped_uid(data: &str, uid: u32) -> Result<i64, String> {
    for line in data.split('\n') {
        let p: Vec<&str> = line.split_whitespace().collect();
        if p.len() != 3 {
            continue;
        }
        let (Some(inside), Some(outside), Some(count)) = (
            terminal::parse_go_uint(p[0], 32),
            terminal::parse_go_uint(p[1], 32),
            terminal::parse_go_uint(p[2], 32),
        ) else {
            continue;
        };
        if outside > 0 && u64::from(uid) >= outside && u64::from(uid) < outside + count {
            return Ok((inside + u64::from(uid) - outside) as i64);
        }
    }
    Err(terminal::err_denied())
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
        && terminal::valid_terminal_id(&input.registration_id)
        && input.muse
}

/// `museChildPID`: running nested-child PID from its inspect output.
pub fn muse_child_pid(body: &[u8], failed: bool, id: &str) -> Result<i32, String> {
    if failed || body.len() > 4096 {
        return Err(terminal::err_denied());
    }
    let v = json::decode_strict(body).map_err(|_| terminal::err_denied())?;
    let Some(fields) = v.as_object() else {
        return Err(terminal::err_denied());
    };
    let get = |name: &str| fields.iter().find(|(k, _)| k == name).map(|(_, v)| v);
    let (Some(Value::Str(got)), Some(pid), Some(Value::Bool(running))) =
        (get("id"), get("pid"), get("running"))
    else {
        return Err(terminal::err_denied());
    };
    let pid = match pid {
        Value::Number(lit) => terminal::parse_go_int(lit).unwrap_or(0),
        _ => 0,
    };
    if got != id || !running || pid <= 0 || pid > i32::MAX as i64 {
        return Err(terminal::err_denied());
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
        return Err(terminal::err_denied());
    }
    let v = json::decode_tolerant(body).map_err(|_| terminal::err_denied())?;
    let Some(mounts) = v.as_array() else {
        return Err(terminal::err_denied());
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
    Err(terminal::err_denied())
}

/// `museELF`: 64-bit little-endian ELF for the host architecture.
pub fn muse_elf(header: &[u8], arch: &str) -> Result<(), String> {
    if header.len() != 64
        || header[0..4] != [0x7f, b'E', b'L', b'F']
        || header[4] != 2
        || header[5] != 1
    {
        return Err(terminal::err_denied());
    }
    let machine = u16::from_le_bytes([header[18], header[19]]);
    if arch == "amd64" && machine == 62 {
        return Ok(());
    }
    if arch == "arm64" && machine == 183 {
        return Ok(());
    }
    Err(terminal::err_denied())
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
        || terminal::clean_path(&binding.credential_root) != binding.credential_root
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
        && terminal::valid_terminal_id(parts[1])
        && parts[2] == binding.id
}

/// `museDeliveryValid`: well-formed Muse delivery for an operation.
pub fn muse_delivery_valid(delivery: &Delivery) -> bool {
    let Some(binding) = &delivery.lease.binding else {
        return false;
    };
    delivery.lease.provider_id == terminal::PROVIDER_MUSE
        && terminal::valid_terminal_id(&delivery.lease.execution_id)
        && binding.id == delivery.lease.execution_id
        && terminal::valid_terminal_id(&binding.invocation_id)
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
            return Err(terminal::err_denied());
        };
        if body.len() > 4096 {
            return Err(terminal::err_denied());
        }
        let v = json::decode_strict(&body).map_err(|_| terminal::err_denied())?;
        let m = json::bind_root(&v, "museInspection", MUSE_INSPECTION_SPECS, false)
            .map_err(|_| terminal::err_denied())?;
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
            return Err(terminal::err_denied());
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
        let svc = terminal::Service {
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
            _ => return Err(terminal::err_denied()),
        }
        let project_ns = std::fs::read_link(format!("/proc/{}/ns/pid", caller.project_pid))
            .map_err(|_| terminal::err_denied())?;
        if caller.namespace != project_ns.to_string_lossy().into_owned() {
            return self.registered_caller(caller, peer, deadline);
        }
        if !muse_peer_alive(peer) {
            return Err(terminal::err_denied());
        }
        Ok(caller)
    }

    fn kernel_caller(&self, peer: &MusePeer) -> Result<MuseCaller, String> {
        let mut caller = MuseCaller::default();
        if !muse_peer_alive(peer) {
            return Err(terminal::err_denied());
        }
        let proc = format!("/proc/{}", peer.pid);
        let cg = std::fs::read_to_string(format!("{proc}/cgroup"))
            .map_err(|_| terminal::err_denied())?;
        caller.container = muse_project_cgroup(&cg)?;
        let mappings = std::fs::read_to_string(format!("{proc}/uid_map"))
            .map_err(|_| terminal::err_denied())?;
        caller.uid = muse_mapped_uid(&mappings, peer.uid)?;
        caller.namespace = std::fs::read_link(format!("{proc}/ns/pid"))
            .map_err(|_| terminal::err_denied())?
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
            return Err(terminal::err_denied());
        };
        if registered.parent != caller.container || registered.project != caller.project {
            return Err(terminal::err_denied());
        }
        self.validate_nested(&registered, deadline)?;
        caller.actor = registered.actor;
        caller.child = registered.child;
        caller.registration = registered.registration;
        caller.nested_pid = registered.pid;
        caller.muse_allowed = registered.muse;
        if !muse_peer_alive(peer) {
            return Err(terminal::err_denied());
        }
        Ok(caller)
    }

    /// `MuseRuntime.resolve`: full caller authority resolution.
    pub fn resolve(&self, peer: &MusePeer, deadline: Instant) -> Result<MuseCaller, String> {
        match self.resolve_inner(peer, deadline) {
            Ok(caller) => Ok(caller),
            Err(_) => Err(terminal::err_denied()),
        }
    }

    fn resolve_inner(&self, peer: &MusePeer, deadline: Instant) -> Result<MuseCaller, String> {
        let caller = self.resolve_project(peer, deadline)?;
        if !caller.child.is_empty() {
            if !caller.muse_allowed {
                return Err(terminal::err_denied());
            }
            return self.nested_caller(caller, peer, deadline);
        }
        if caller.uid == 0 {
            return Err(terminal::err_denied());
        }
        let caller = self.project_account(caller, deadline)?;
        let mut caller = caller;
        caller.actor = self.project_actor(&caller, deadline)?;
        if !self.authorized_caller(&caller, peer, deadline) {
            return Err(terminal::err_denied());
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
            return Err(terminal::err_denied());
        };
        if body.len() > 4096 {
            return Err(terminal::err_denied());
        }
        let account: Vec<String> = String::from_utf8_lossy(&body)
            .trim()
            .split(':')
            .map(|s| s.to_string())
            .collect();
        if !muse_passwd_valid(&account, caller.uid) {
            return Err(terminal::err_denied());
        }
        caller.login = account[0].clone();
        caller.home = account[5].clone();
        caller.gid = terminal::parse_go_int(&account[3]).ok_or_else(terminal::err_denied)?;
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
            return Err(terminal::err_denied());
        };
        if !muse_account_modes(&String::from_utf8_lossy(&body)) {
            return Err(terminal::err_denied());
        }
        let body = self.guest_refs(&caller.container, &[], &["/usr/bin/cat", &marker], deadline);
        let Ok(body) = body else {
            return Err(terminal::err_denied());
        };
        if body.len() > 64 {
            return Err(terminal::err_denied());
        }
        match terminal::parse_go_int(String::from_utf8_lossy(&body).trim()) {
            Some(actor) if actor > 0 => Ok(actor),
            _ => Err(terminal::err_denied()),
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
            return Err(terminal::err_denied());
        }
        let caller = self.resolve_project(peer, deadline)?;
        if !self.registration_authority(&caller, input.actor_id, deadline) {
            return Err(terminal::err_denied());
        }
        let record = self.registered_child(&caller, input, deadline)?;
        self.validate_nested(&record, deadline)?;
        if !muse_peer_alive(peer) {
            return Err(terminal::err_denied());
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
            .map_err(|_| terminal::err_denied())?;
        let ns = String::from_utf8_lossy(&body).trim().to_string();
        if !ns.starts_with("pid:[") || ns == caller.namespace {
            return Err(terminal::err_denied());
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
            return Err(terminal::err_denied());
        };
        if String::from_utf8_lossy(&body).trim() != format!("{} {} true", record.child, record.pid)
        {
            return Err(terminal::err_denied());
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
            return Err(terminal::err_denied());
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
            .map_err(|_| terminal::err_denied())?;
        if String::from_utf8_lossy(&body).trim() != record.namespace {
            return Err(terminal::err_denied());
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
            .map_err(|_| terminal::err_denied())?;
        let text = String::from_utf8_lossy(&body);
        let names: Vec<&str> = text.split_whitespace().collect();
        if names.len() != 2 || names[0] != names[1] {
            return Err(terminal::err_denied());
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
            return Err(terminal::err_denied());
        };
        if body.len() > 4096 {
            return Err(terminal::err_denied());
        }
        let v = json::decode_tolerant(&body).map_err(|_| terminal::err_denied())?;
        let Some(fields) = v.as_object() else {
            return Err(terminal::err_denied());
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
            return Err(terminal::err_denied());
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
            .map_err(|_| terminal::err_denied())?;
        caller.gid = muse_mapped_uid(&mappings, peer.gid)?;
        if !self.authorized_caller(&caller, peer, deadline) {
            return Err(terminal::err_denied());
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
            return Err(terminal::err_denied());
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
            Err(_) => return Err(terminal::err_denied()),
        };
        let text = String::from_utf8_lossy(&hash);
        let fields: Vec<&str> = text.split_whitespace().collect();
        if fields.len() != 2 || fields[0] != self.binary_sha256 {
            return Err(terminal::err_denied());
        }
        let mut head_argv = prefix.clone();
        head_argv.extend([
            "/usr/bin/head".to_string(),
            "--bytes=64".to_string(),
            "/usr/local/libexec/soda/muse".to_string(),
        ]);
        let header = self
            .guest(container, &[], &head_argv, deadline)
            .map_err(|_| terminal::err_denied())?;
        muse_elf(&header, host_go_arch())?;
        if self.binary_version.is_empty() {
            return Err(terminal::err_denied());
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
            return Err(terminal::err_denied());
        }
        request.args = muse_arguments(&request.args)?;
        let caller = self.resolve(peer, budget)?;
        self.verify_guest_binary(&caller.container, &caller.child, budget)?;
        let execution = self.reserve_execution(&caller, &request, budget)?;
        if !muse_peer_alive(peer) {
            let cleanup = Instant::now() + Duration::from_secs(30);
            let _ = self.hooks.end(caller.actor, &execution.lease.id, cleanup);
            return Err(terminal::err_denied());
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
        let id = terminal::rand_id().map_err(|_| terminal::err_denied())?;
        let lease = self.hooks.acquire(
            &AcquireRequest {
                execution_id: id.clone(),
                actor_id: caller.actor,
                connection_id: connection,
                project_id: caller.project.clone(),
                kind: terminal::KIND_TERMINAL.to_string(),
                provider_id: terminal::PROVIDER_MUSE.to_string(),
                deadline_secs: terminal::now_unix() + 12 * 3600,
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
            kind: terminal::KIND_TERMINAL.to_string(),
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
        if !terminal::valid_terminal_id(&execution.binding.invocation_id) {
            return Err(terminal::err_denied());
        }
        let mut delivery = self
            .hooks
            .attach(&execution.lease.id, &execution.binding, budget)?;
        let mut credential = delivery.credential.take().unwrap_or_default();
        if !terminal::credential_valid(&credential) {
            return Err(terminal::err_denied());
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
                return Err(terminal::err_denied());
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
            return Err(terminal::err_denied());
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
            return Err(terminal::err_denied());
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
            .map_err(|_| terminal::err_uncertain())?;
        if String::from_utf8_lossy(&body).trim() != "inactive" {
            return Err(terminal::err_uncertain());
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
            .map_err(|_| terminal::err_uncertain())?;
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
            || !terminal::valid_terminal_id(&binding.id)
            || !domain::valid_container_id(&binding.project)
            || !domain::valid_login(&binding.login)
        {
            return Err(terminal::err_denied());
        }
        let body = self
            .guest(
                &binding.project,
                &[],
                &unit_active_argv(&format!("soda-muse-{}.service", binding.id)),
                deadline,
            )
            .map_err(|_| terminal::err_stale())?;
        if String::from_utf8_lossy(&body).trim() != "active" {
            return Err(terminal::err_stale());
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
                return Err(terminal::err_denied());
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
            return Err(terminal::err_denied());
        }
        let binding = delivery
            .lease
            .binding
            .as_ref()
            .ok_or_else(terminal::err_denied)?;
        match binding.scope.as_str() {
            "muse-project" => self.project_operation(action, delivery, deadline)?,
            _ => return Err(terminal::err_denied()),
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
            .ok_or_else(terminal::err_denied)?;
        let unit = format!("soda-muse-{}.service", binding.id);
        if !muse_project_credential_root(binding) {
            return Err(terminal::err_denied());
        }
        if let Ok(body) = self.guest(
            &binding.project,
            &[],
            &unit_invocation_argv(&unit),
            deadline,
        ) {
            let current = String::from_utf8_lossy(&body).trim().to_string();
            if !current.is_empty() && current != binding.invocation_id {
                return Err(terminal::err_stale());
            }
        }
        if action == "validate" {
            return self.validate_muse_binding(binding, deadline);
        }
        if action != "stop" {
            return Err(terminal::err_denied());
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
            return Err(terminal::err_denied());
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
            Err(err) if terminal::exit_code_of(&err) == Some(1) => return Ok(()),
            Err(_) => return Err(terminal::err_uncertain()),
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
        .map_err(|_| terminal::err_uncertain())?;
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
        .map_err(|_| terminal::err_uncertain())
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
            Err(err) if terminal::exit_code_of(&err) == Some(32) => Ok(()),
            _ => Err(terminal::err_uncertain()),
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
        .map_err(|_| terminal::err_uncertain())
    }
}

/// `museResize`: PTY resize over the stdin fd.
pub fn muse_resize(stdin_fd: RawFd, control: &LaunchControl) -> Result<(), String> {
    if control.signal != 0 || control.cols == 0 || control.rows == 0 {
        return Err(terminal::err_denied());
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
        return Err(terminal::err_denied());
    }
    Ok(())
}

/// `stateContainer`: execution state owner (child for nested runs).
pub fn state_container(binding: &Binding) -> Result<String, String> {
    if !terminal::valid_terminal_id(&binding.id) || binding.uid < 0 || binding.gid < 0 {
        return Err(terminal::err_denied());
    }
    let mut container = binding.project.clone();
    if binding.scope == "muse-project" && !binding.child_id.is_empty() {
        container = binding.child_id.clone();
    }
    if !domain::valid_container_id(&container) {
        return Err(terminal::err_denied());
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
    let v = json::decode_tolerant(body).map_err(|_| terminal::err_denied())?;
    let Some(fields) = v.as_object() else {
        return Err(terminal::err_denied());
    };
    let mut out = HashMap::with_capacity(fields.len());
    for (key, value) in fields {
        let bytes = match value {
            Value::Null => Vec::new(),
            Value::Str(s) => {
                crate::ssh::b64_decode_go(s.as_bytes()).map_err(|_| terminal::err_denied())?
            }
            Value::Array(items) => {
                let mut bytes = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        Value::Number(lit) => {
                            let n: i64 = lit.parse().map_err(|_| terminal::err_denied())?;
                            if !(0..=255).contains(&n) {
                                return Err(terminal::err_denied());
                            }
                            bytes.push(n as u8);
                        }
                        _ => return Err(terminal::err_denied()),
                    }
                }
                bytes
            }
            _ => return Err(terminal::err_denied()),
        };
        out.insert(key.clone(), bytes);
    }
    Ok(out)
}

#[cfg(test)]
mod tests;
