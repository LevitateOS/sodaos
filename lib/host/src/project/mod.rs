//! Privileged project environment operations (PR19: init slice).
//!
//! Port of `internal/host/project/{config,runtime,profiles,create,
//! container,os}.go` except the account tail of `create.go` (PR20), plus
//! the `project_os.py` observation logic. The daemon routes served from
//! this module are `/profile`, `/create`, `/inspect`, `/os` and
//! `/connection`.
//!
//! Behavioral notes:
//!
//! * OS observation runs `test -f` + `head -c 4097` in the container and
//!   parses `/etc/os-release` on the host with the exact `project_os.py`
//!   rules instead of executing Python in the container. The `test -f`
//!   pre-check keeps the FIFO refusal fast; anything unparsable still
//!   reports `os_release_unavailable`.
//! * Exact podman/argv shapes, output caps and error strings are preserved;
//!   only process-spawn failure text (never wire-visible: the daemon maps
//!   it to a generic 500) follows Rust formatting.

use std::collections::HashMap;
use std::time::{Duration, Instant};

use crate::domain;
use crate::json::{self, Value};
use crate::net;

mod connection;
mod create;
mod os;
mod profile;

#[cfg(test)]
mod tests;

pub(crate) const INSPECTION_SPECS: &[json::Spec] = &[
    json::Spec {
        name: "id",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "running",
        kind: json::Kind::Bool,
    },
    json::Spec {
        name: "project",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "owner",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "privileged",
        kind: json::Kind::Bool,
    },
    json::Spec {
        name: "userns",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "mappings",
        kind: json::Kind::Object {
            go_type: "struct",
            struct_name: "struct",
            specs: &[
                json::Spec {
                    name: "UidMap",
                    kind: json::Kind::StrList,
                },
                json::Spec {
                    name: "GidMap",
                    kind: json::Kind::StrList,
                },
            ],
        },
    },
];

pub const PROJECT_INSPECT_FORMAT: &str = "{\"id\":{{json .ID}},\"running\":{{json .State.Running}},\"project\":{{json (index .Config.Labels \"org.soda.project\")}},\"owner\":{{json (index .Config.Labels \"org.soda.owner\")}},\"privileged\":{{json .HostConfig.Privileged}},\"userns\":{{json .HostConfig.UsernsMode}},\"mappings\":{{json .HostConfig.IDMappings}}}";

#[derive(Debug, Clone, Default)]
pub struct Config {
    pub muse_socket: String,
    pub image: String,
    pub network: String,
    pub subnet: String,
    pub bridge: String,
}

/// `project.Lifecycle`: a start/stop/inspect request for one project.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Lifecycle {
    pub project: String,
    pub action: String,
}

const LIFECYCLE_SPECS: &[json::Spec] = &[
    json::Spec {
        name: "project",
        kind: json::Kind::Str,
    },
    json::Spec {
        name: "action",
        kind: json::Kind::Str,
    },
];

impl Lifecycle {
    /// Strict decode of one lifecycle request (`strictjson.Decode` parity:
    /// unknown fields rejected).
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "Lifecycle", LIFECYCLE_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &json::BoundMap) -> Self {
        Lifecycle {
            project: m.take_string("project"),
            action: m.take_string("action"),
        }
    }
}

/// `project.LifecycleState`: the observed lifecycle outcome.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct LifecycleState {
    pub environment: domain::Environment,
    pub boot_enabled: bool,
}

impl LifecycleState {
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"environment\":");
        self.environment.encode_into(&mut out);
        out.push_str(",\"boot_enabled\":");
        out.push_str(if self.boot_enabled { "true" } else { "false" });
        out.push('}');
        out
    }
}

/// Selected project unit template (`platform.ProjectUnit`).
pub const PROJECT_UNIT_PATH: &str = "/usr/lib/systemd/system/soda-project@.service";

pub trait Executor {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String>;
    /// Mirrors Go's `HostNative()` marker assertion: the privileged host
    /// executor whose native protocols must never leak stderr text.
    fn is_host_native(&self) -> bool {
        false
    }
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
        use std::io::Write;
        use std::process::Stdio;
        let mut child = std::process::Command::new(cmd)
            .args(args)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()
            .map_err(|e| format!("{cmd} failed: {e}"))?;
        let mut input = child.stdin.take();
        let mut out_pipe = child.stdout.take();
        let mut err_pipe = child.stderr.take();
        let outcome = std::thread::scope(|scope| {
            let writer = scope.spawn(|| {
                if let Some(mut w) = input.take() {
                    let _ = w.write_all(stdin);
                }
            });
            // Drain stdout/stderr concurrently: a child emitting beyond
            // pipe capacity would otherwise block forever while the poll
            // loop below waits for exit (H01-F2).
            let out_drain = scope.spawn(|| {
                use std::io::Read;
                let mut buf = Vec::new();
                if let Some(mut o) = out_pipe.take() {
                    let _ = o.read_to_end(&mut buf);
                }
                buf
            });
            let err_drain = scope.spawn(|| {
                use std::io::Read;
                let mut buf = Vec::new();
                if let Some(mut e) = err_pipe.take() {
                    let _ = e.read_to_end(&mut buf);
                }
                buf
            });
            let status = loop {
                match child.try_wait().map_err(|e| format!("{cmd} failed: {e}"))? {
                    Some(status) => break status,
                    None => {
                        if Instant::now() >= deadline {
                            let _ = child.kill();
                            let _ = child.wait();
                            let _ = writer.join();
                            return Err(format!("{cmd} failed: deadline exceeded"));
                        }
                        std::thread::sleep(Duration::from_millis(5));
                    }
                }
            };
            writer
                .join()
                .map_err(|_| format!("{cmd} failed: stdin writer panicked"))?;
            let stdout = out_drain.join().unwrap_or_default();
            let stderr = err_drain.join().unwrap_or_default();
            Ok((status, stdout, stderr))
        })?;
        let (status, stdout, stderr): (std::process::ExitStatus, Vec<u8>, Vec<u8>) = outcome;
        if status.success() {
            return Ok(stdout);
        }
        Err(format!(
            "{cmd} failed: {}: {}",
            exit_text(status),
            String::from_utf8_lossy(&stderr)
        ))
    }

    fn is_host_native(&self) -> bool {
        true
    }
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

pub struct Runtime<E> {
    pub exec: E,
    pub config: Config,
}

impl<E: Executor> Runtime<E> {
    pub(crate) fn podman(
        &self,
        stdin: &[u8],
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.exec.run(stdin, "/usr/bin/podman", args, deadline)
    }

    /// `Inspect`: observe one project container by identity.
    pub fn inspect(
        &self,
        id: &str,
        deadline: Instant,
    ) -> Result<(domain::Environment, i64), String> {
        let mut env = domain::Environment {
            image: String::new(),
            profile: None,
            id: id.to_string(),
            ip: String::new(),
            running: false,
        };
        if !domain::valid_id(id) {
            return Err("invalid project id".to_string());
        }
        let out = self.podman(&[], &["inspect", &format!("soda-{id}")], deadline)?;
        let v = json::decode_tolerant(&out).map_err(|_| "invalid native inspection".to_string())?;
        let items = v
            .as_array()
            .filter(|a| a.len() == 1)
            .ok_or_else(|| "invalid native inspection".to_string())?;
        let item = &items[0];
        if item.as_object().is_none() {
            return Err("invalid native inspection".to_string());
        }
        let image = tolerant_str(item, "Image")?;
        if domain::valid_image_ref(&image) {
            env.image = format!("sha256:{}", image.trim_start_matches("sha256:"));
        }
        let empty = Value::Object(Vec::new());
        let config = json::tolerant_get(item, "Config").unwrap_or(&empty);
        let labels = labels_of(config)?;
        if labels
            .get("org.soda.project")
            .map(String::as_str)
            .unwrap_or("")
            != id
        {
            return Err("container is not owned by this project".to_string());
        }
        let owner: i64 = json::parse_go_int64(
            labels
                .get("org.soda.owner")
                .map(String::as_str)
                .unwrap_or(""),
        )
        .ok_or_else(|| "invalid native project owner".to_string())?;
        if owner <= 0 {
            return Err("invalid native project owner".to_string());
        }
        profile::apply_creation_profile(&mut env, &labels, &image)?;
        env.running = match json::tolerant_get(item, "State") {
            None => false,
            Some(state) => match json::tolerant_get(state, "Running") {
                None => false,
                Some(Value::Bool(b)) => *b,
                Some(_) => return Err("invalid native inspection".to_string()),
            },
        };
        let mut ip = String::new();
        if let Some(settings) = json::tolerant_get(item, "NetworkSettings") {
            if settings.as_object().is_none() {
                return Err("invalid native inspection".to_string());
            }
            let empty = Value::Object(Vec::new());
            let networks = json::tolerant_get(settings, "Networks").unwrap_or(&empty);
            if networks.as_object().is_none() {
                return Err("invalid native inspection".to_string());
            }
            if let Some(entry) = networks
                .as_object()
                .unwrap()
                .iter()
                .find(|(k, _)| *k == self.config.network)
                .map(|(_, v)| v)
            {
                if entry.is_null() {
                    // Null decodes as the zero struct: no address.
                } else if entry.as_object().is_none() {
                    return Err("invalid native inspection".to_string());
                }
                match json::tolerant_get(entry, "IPAddress") {
                    None => {}
                    Some(Value::Str(s)) => ip = s.clone(),
                    Some(_) => return Err("invalid native inspection".to_string()),
                }
            }
        }
        env.ip = ip.clone();
        net::admit_ip(&ip, &self.config.subnet).map_err(|e| {
            if e == "project IP outside configured network" {
                e
            } else {
                format!("invalid IP address {ip:?}")
            }
        })?;
        Ok((env, owner))
    }

    /// `ProjectContainer`: bind the exact isolated container identity.
    pub fn project_container(
        &self,
        id: &str,
        require_running: bool,
        deadline: Instant,
    ) -> Result<String, String> {
        if !domain::valid_id(id) {
            return Err("invalid project".to_string());
        }
        let data = self
            .podman(
                &[],
                &[
                    "--remote=false",
                    "inspect",
                    "--format",
                    PROJECT_INSPECT_FORMAT,
                    &format!("soda-{id}"),
                ],
                deadline,
            )
            .map_err(|_| "terminal inspection unavailable".to_string())?;
        if data.len() > 4096 {
            return Err("terminal inspection unavailable".to_string());
        }
        let v =
            json::decode_strict(&data).map_err(|_| "invalid terminal inspection".to_string())?;
        let m = json::bind_root(&v, "projectInspection", INSPECTION_SPECS, false)
            .map_err(|_| "invalid terminal inspection".to_string())?;
        let cid = m.take_string("id");
        let running = m.take_bool("running");
        let project = m.take_string("project");
        let owner = m.take_string("owner");
        let privileged = m.take_bool("privileged");
        let userns = m.take_string("userns");
        let mappings = m.take_map("mappings");
        let uid_map = mappings.take_str_list("UidMap");
        let gid_map = mappings.take_str_list("GidMap");
        let owner_num: i64 = json::parse_go_int64(&owner)
            .ok_or_else(|| "terminal target not ready or isolated".to_string())?;
        if owner_num <= 0 || (require_running && !running) {
            return Err("terminal target not ready or isolated".to_string());
        }
        if !domain::valid_container_id(&cid) || project != id || privileged || userns != "private" {
            return Err("terminal target not ready or isolated".to_string());
        }
        if !project_id_map(&uid_map) || !project_id_map(&gid_map) {
            return Err("terminal target not ready or isolated".to_string());
        }
        Ok(cid)
    }

    /// `Lifecycle`: inspect, start or stop one project environment through
    /// its systemd unit, fencing on container identity across the action.
    pub fn lifecycle(
        &self,
        input: &Lifecycle,
        deadline: Instant,
    ) -> Result<LifecycleState, String> {
        if !valid_lifecycle_action(&input.action) {
            return Err("invalid lifecycle operation".to_string());
        }
        let cid = self.project_container(&input.project, false, deadline)?;
        let unit = format!("soda-project@{}.service", input.project);
        self.read_project_unit(&unit, deadline)?;
        self.apply_lifecycle_action(&input.action, &unit, deadline)?;
        let after = self.project_container(&input.project, false, deadline)?;
        if after != cid {
            return Err("project identity changed during operation".to_string());
        }
        let boot_enabled = self.read_project_unit(&unit, deadline)?;
        let (environment, _) = self.inspect(&input.project, deadline)?;
        let result = LifecycleState {
            environment,
            boot_enabled,
        };
        verify_lifecycle_outcome(&input.action, &result)?;
        Ok(result)
    }

    /// `readProjectUnit`: observe the selected unit's boot-enable flag.
    fn read_project_unit(&self, unit: &str, deadline: Instant) -> Result<bool, String> {
        let out = self
            .exec
            .run(
                &[],
                "/usr/bin/systemctl",
                &[
                    "show",
                    unit,
                    "--property=LoadState,FragmentPath,DropInPaths,UnitFileState",
                ],
                deadline,
            )
            .map_err(|_| "native unit unavailable".to_string())?;
        let fields = parse_unit_show_properties(&out)?;
        validate_unit_properties(&fields)
    }

    /// `applyLifecycleAction`: enable (start) or disable (stop) the unit.
    fn apply_lifecycle_action(
        &self,
        action: &str,
        unit: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        if action == "inspect" {
            return Ok(());
        }
        let verb = if action == "stop" {
            "disable"
        } else {
            "enable"
        };
        // Stop also disables next-boot start. Start restores it. No
        // persistent desired-state copy or direct Podman stop competing
        // with systemd Restart.
        self.exec
            .run(&[], "/usr/bin/systemctl", &[verb, "--now", unit], deadline)
            .map(|_| ())
            .map_err(|_| "native lifecycle outcome unconfirmed".to_string())
    }
}

/// Go `runtime.GOARCH` naming for the local architecture.
fn go_arch() -> &'static str {
    match std::env::consts::ARCH {
        "x86_64" => "amd64",
        "aarch64" => "arm64",
        // SodaOS targets x86_64 only; anything else compares unequal to the
        // podman architecture exactly like the Go build would.
        _ => std::env::consts::ARCH,
    }
}

fn valid_lifecycle_action(action: &str) -> bool {
    action == "inspect" || action == "start" || action == "stop"
}

/// `parseUnitShowProperties`: strict `key=value` lines for the four
/// selected unit properties.
fn parse_unit_show_properties(out: &[u8]) -> Result<HashMap<String, String>, String> {
    if out.len() > 4096 {
        return Err("native unit unavailable".to_string());
    }
    let text = String::from_utf8_lossy(out);
    let mut fields = HashMap::new();
    for line in text.trim().split('\n') {
        let (key, value) = line
            .split_once('=')
            .ok_or_else(|| "invalid native unit observation".to_string())?;
        if fields.contains_key(key) {
            return Err("ambiguous native unit".to_string());
        }
        match key {
            "LoadState" | "FragmentPath" | "DropInPaths" | "UnitFileState" => {
                fields.insert(key.to_string(), value.to_string());
            }
            _ => return Err("unexpected unit property".to_string()),
        }
    }
    Ok(fields)
}

/// `validateUnitProperties`: the unit must be the selected project unit;
/// reports its boot-enable flag.
fn validate_unit_properties(fields: &HashMap<String, String>) -> Result<bool, String> {
    const ERR: &str = "native unit is not the selected project unit";
    let get = |k: &str| fields.get(k).map(String::as_str).unwrap_or("");
    if fields.len() != 4 || get("LoadState") != "loaded" || get("FragmentPath") != PROJECT_UNIT_PATH
    {
        return Err(ERR.to_string());
    }
    let drop_ins = get("DropInPaths");
    if !drop_ins.is_empty() && drop_ins != "/usr/lib/systemd/system/service.d/10-timeout-abort.conf"
    {
        return Err(ERR.to_string());
    }
    match get("UnitFileState") {
        "enabled" => Ok(true),
        "disabled" => Ok(false),
        _ => Err(ERR.to_string()),
    }
}

fn verify_lifecycle_outcome(action: &str, result: &LifecycleState) -> Result<(), String> {
    if action == "start" && (!result.environment.running || !result.boot_enabled) {
        return Err("native lifecycle outcome unconfirmed".to_string());
    }
    if action == "stop" && (result.environment.running || result.boot_enabled) {
        return Err("native lifecycle outcome unconfirmed".to_string());
    }
    Ok(())
}

// ---------- facade confirmations ----------
//
// Pure client-side logic from `internal/host/{lifecycle,profiles,os,
// access_keys}.go`: transport (`c.call`) arrives with the route layer,
// but the confirmation predicates are exact here.

/// `validAddress`: a usable endpoint address, never unspecified,
/// multicast or loopback. Go's `netip.ParseAddr` also accepts scoped
/// (`%zone`) addresses, which this daemon never reports; those are
/// rejected here.
pub fn valid_address(value: &str) -> bool {
    match value.parse::<std::net::IpAddr>() {
        Ok(ip) => !ip.is_unspecified() && !ip.is_multicast() && !ip.is_loopback(),
        Err(_) => false,
    }
}

fn lifecycle_action_confirmed(action: &str, out: &LifecycleState) -> bool {
    match action {
        "start" => out.environment.running && out.boot_enabled,
        "stop" => !out.environment.running && !out.boot_enabled,
        _ => true,
    }
}

/// `Client.Lifecycle` outcome confirmation.
pub fn confirm_lifecycle(input: &Lifecycle, out: &LifecycleState) -> Result<(), String> {
    if out.environment.id != input.project {
        return Err("native lifecycle outcome not confirmed".to_string());
    }
    if !out.environment.ip.is_empty() && !valid_address(&out.environment.ip) {
        return Err("native lifecycle outcome not confirmed".to_string());
    }
    if !lifecycle_action_confirmed(&input.action, out) {
        return Err("native lifecycle outcome not confirmed".to_string());
    }
    Ok(())
}

/// `Client.ResolveProfile` confirmation: the profile validates and the
/// image is native to this backend. Like Go, a foreign architecture
/// overwrites a validation failure.
pub fn confirm_resolve_profile(p: &domain::Profile) -> Result<(), String> {
    let mut err = p.validate().err();
    if p.architecture != go_arch() {
        err = Some("project image is not native to this backend".to_string());
    }
    err.map_or(Ok(()), Err)
}

fn valid_os_environment(env: &domain::Environment) -> bool {
    if !env.image.is_empty()
        && (!env.image.starts_with("sha256:") || !domain::valid_image_ref(&env.image))
    {
        return false;
    }
    if !env.ip.is_empty() && !valid_address(&env.ip) {
        return false;
    }
    env.profile
        .as_ref()
        .map(|p| p.validate().is_ok())
        .unwrap_or(true)
}

fn valid_os_observation(id: &str, out: &domain::OsObservation) -> bool {
    if out.environment.id != id || out.release.is_none() != out.unavailable {
        return false;
    }
    if !valid_os_environment(&out.environment) {
        return false;
    }
    match &out.release {
        Some(release) => out.environment.running && domain::valid_os_release(release),
        None => true,
    }
}

/// `Client.ObserveOS` confirmation.
pub fn confirm_observe_os(id: &str, out: &domain::OsObservation) -> Result<(), String> {
    if valid_os_observation(id, out) {
        Ok(())
    } else {
        Err("invalid native OS observation".to_string())
    }
}

/// `Client.AccessKeys` confirmation. Go additionally rejects a null key
/// list, which the decoded struct cannot distinguish from empty; that
/// single check is not mirrored.
pub fn confirm_access_keys(
    input: &domain::AccessKeys,
    out: &domain::AccessKeyState,
) -> Result<(), String> {
    let mut err = crate::account::canonical_keys(&out.keys).err();
    if !crate::account::valid_key_revision(&out.revision) {
        err = Some("invalid native key revision".to_string());
    }
    if input.apply && out.keys.join("\n") != input.keys.join("\n") {
        err = Some("native key result differs from request".to_string());
    }
    err.map_or(Ok(()), Err)
}

fn tolerant_str(item: &Value, key: &str) -> Result<String, String> {
    match json::tolerant_get(item, key) {
        None => Ok(String::new()),
        Some(Value::Str(s)) => Ok(s.clone()),
        Some(_) => Err("invalid native inspection".to_string()),
    }
}

fn labels_of(config: &Value) -> Result<HashMap<String, String>, String> {
    if config.as_object().is_none() {
        return Err("invalid native inspection".to_string());
    }
    match json::tolerant_get(config, "Labels") {
        None => Ok(HashMap::new()),
        Some(Value::Object(fields)) => {
            let mut map = HashMap::new();
            for (k, val) in fields {
                match val {
                    Value::Str(s) => {
                        map.insert(k.clone(), s.clone());
                    }
                    Value::Null => {}
                    _ => return Err("invalid native inspection".to_string()),
                }
            }
            Ok(map)
        }
        Some(_) => Err("invalid native inspection".to_string()),
    }
}

/// Single 262144-ID mapping with container root shifted off host root.
/// (The Go `base+262144 <= 4294967295` bound is vacuous under uint32
/// wraparound: every canonical non-zero base passes it, so the check is
/// exactly `base > 0`.)
fn project_id_map(values: &[String]) -> bool {
    if values.len() != 1 {
        return false;
    }
    let parts: Vec<&str> = values[0].split(':').collect();
    if parts.len() != 3 || parts[0] != "0" || parts[2] != "262144" {
        return false;
    }
    matches!(parts[1].parse::<u32>(), Ok(base) if base.to_string() == parts[1] && base > 0 && u64::from(base) + 262144 <= 4294967295)
}

#[cfg(test)]
// Narrow A01 split: tests for duties staying in this root
// (executor/inspect/lifecycle/confirmations); moved-duty tests live in tests.rs.
mod retained_tests {
    use super::tests::{container_inspect, deadline, sample_profile, test_config, Mock};
    use super::*;

    #[test]
    fn inspect_observes_identity_and_profile() {
        let id = format!("p{}", "e".repeat(24));
        let p = sample_profile();
        let mock = Mock::new(vec![Ok(container_inspect(
            &id,
            "42",
            Some(&p),
            true,
            "10.0.0.5",
        ))]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let (env, owner) = rt.inspect(&id, deadline()).unwrap();
        assert_eq!(owner, 42);
        assert_eq!(env.id, id);
        assert!(env.running);
        assert_eq!(env.ip, "10.0.0.5");
        assert_eq!(env.profile.as_ref().unwrap(), &p);
        assert_eq!(env.image, p.image);
    }

    #[test]
    fn inspect_rejects_mismatch_and_bad_network() {
        let id = format!("p{}", "e".repeat(24));
        let p = sample_profile();
        // Wrong project label.
        let mock = Mock::new(vec![Ok(container_inspect(
            "p000000000000000000000000",
            "42",
            Some(&p),
            true,
            "10.0.0.5",
        ))]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.inspect(&id, deadline()).unwrap_err(),
            "container is not owned by this project"
        );
        // Creation profile for a different image than the live container.
        let mut other = p.clone();
        other.image = format!("sha256:{}", "f".repeat(64));
        let body = format!(
            "[{{\"Image\":{:?},\"Config\":{{\"Labels\":{{\"org.soda.project\":{id:?},\"org.soda.owner\":\"42\",\"org.soda.creation-profile\":{:?},\"org.soda.profile\":{:?}}}}},\"State\":{{\"Running\":true}},\"NetworkSettings\":{{\"Networks\":{{}}}}}}]",
            p.image,
            other.encode(),
            p.id,
        );
        let mock = Mock::new(vec![Ok(body.into_bytes())]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.inspect(&id, deadline()).unwrap_err(),
            "native creation profile mismatch"
        );
        // IP outside the subnet.
        let mock = Mock::new(vec![Ok(container_inspect(
            &id,
            "42",
            Some(&p),
            true,
            "10.0.9.9",
        ))]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.inspect(&id, deadline()).unwrap_err(),
            "project IP outside configured network"
        );
        // Bad project id never execs.
        let mock = Mock::new(vec![]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.inspect("bogus", deadline()).unwrap_err(),
            "invalid project id"
        );
        assert!(mock.calls.borrow().is_empty());
    }

    #[test]
    fn project_container_binds_isolated_identity() {
        let id = format!("p{}", "e".repeat(24));
        let cid = "f".repeat(64);
        let body = format!(
            "{{\"id\":{cid:?},\"running\":true,\"project\":{id:?},\"owner\":\"42\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:1000000:262144\"],\"GidMap\":[\"0:1000000:262144\"]}}}}"
        );
        let mock = Mock::new(vec![Ok(body.into_bytes())]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(rt.project_container(&id, true, deadline()).unwrap(), cid);
        // Privileged or shared userns is refused.
        let body = format!(
            "{{\"id\":{cid:?},\"running\":true,\"project\":{id:?},\"owner\":\"42\",\"privileged\":true,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:1:262144\"],\"GidMap\":[\"0:1:262144\"]}}}}"
        );
        let mock = Mock::new(vec![Ok(body.into_bytes())]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.project_container(&id, false, deadline()).unwrap_err(),
            "terminal target not ready or isolated"
        );
        // Unknown fields are rejected (strict podman-format decode).
        let body = format!(
            "{{\"id\":{cid:?},\"running\":true,\"project\":{id:?},\"owner\":\"42\",\"privileged\":false,\"userns\":\"private\",\"extra\":1,\"mappings\":{{\"UidMap\":[],\"GidMap\":[]}}}}"
        );
        let mock = Mock::new(vec![Ok(body.into_bytes())]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.project_container(&id, false, deadline()).unwrap_err(),
            "invalid terminal inspection"
        );
    }

    fn format_inspect(id: &str, cid: &str) -> Vec<u8> {
        format!(
            "{{\"id\":{cid:?},\"running\":true,\"project\":{id:?},\"owner\":\"42\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:1000000:262144\"],\"GidMap\":[\"0:1000000:262144\"]}}}}"
        )
        .into_bytes()
    }

    fn unit_show(state: &str) -> Vec<u8> {
        format!(
            "LoadState=loaded\nFragmentPath={PROJECT_UNIT_PATH}\nDropInPaths=\nUnitFileState={state}\n"
        )
        .into_bytes()
    }

    #[test]
    fn unit_show_parsing_matrix() {
        let fields = parse_unit_show_properties(&unit_show("enabled")).unwrap();
        assert_eq!(fields.len(), 4);
        assert!(validate_unit_properties(&fields).unwrap());
        let fields = parse_unit_show_properties(&unit_show("disabled")).unwrap();
        assert!(!validate_unit_properties(&fields).unwrap());
        // Allowed drop-in.
        let with_dropin = format!(
            "LoadState=loaded\nFragmentPath={PROJECT_UNIT_PATH}\nDropInPaths=/usr/lib/systemd/system/service.d/10-timeout-abort.conf\nUnitFileState=enabled\n"
        );
        let fields = parse_unit_show_properties(with_dropin.as_bytes()).unwrap();
        assert!(validate_unit_properties(&fields).unwrap());
        // Oversized.
        assert_eq!(
            parse_unit_show_properties(&vec![b'x'; 4097]).unwrap_err(),
            "native unit unavailable"
        );
        // Line without '='.
        assert_eq!(
            parse_unit_show_properties(b"LoadState").unwrap_err(),
            "invalid native unit observation"
        );
        // Empty output has one empty line without '='.
        assert_eq!(
            parse_unit_show_properties(b"").unwrap_err(),
            "invalid native unit observation"
        );
        // Duplicate key.
        assert_eq!(
            parse_unit_show_properties(b"LoadState=loaded\nLoadState=loaded\n").unwrap_err(),
            "ambiguous native unit"
        );
        // Unexpected property.
        assert_eq!(
            parse_unit_show_properties(b"LoadState=loaded\nActiveState=active\n").unwrap_err(),
            "unexpected unit property"
        );
        // Not the selected unit.
        for body in [
            "LoadState=loaded\nFragmentPath=/other.service\nDropInPaths=\nUnitFileState=enabled\n",
            "LoadState=failed\nFragmentPath=/x\nDropInPaths=\nUnitFileState=enabled\n",
            &format!("LoadState=loaded\nFragmentPath={PROJECT_UNIT_PATH}\nDropInPaths=/other.conf\nUnitFileState=enabled\n"),
            &format!("LoadState=loaded\nFragmentPath={PROJECT_UNIT_PATH}\nDropInPaths=\nUnitFileState=masked\n"),
            &format!("LoadState=loaded\nFragmentPath={PROJECT_UNIT_PATH}\nUnitFileState=enabled\n"),
        ] {
            let fields = parse_unit_show_properties(body.as_bytes()).unwrap();
            assert_eq!(
                validate_unit_properties(&fields).unwrap_err(),
                "native unit is not the selected project unit",
                "{body:?}"
            );
        }
    }

    #[test]
    fn lifecycle_start_stop_inspect_flows() {
        let id = format!("p{}", "e".repeat(24));
        let cid = "f".repeat(64);
        let p = sample_profile();
        for (action, verb, running, boot) in [
            ("start", "enable", true, "enabled"),
            ("stop", "disable", false, "disabled"),
        ] {
            let mock = Mock::new(vec![
                Ok(format_inspect(&id, &cid)),
                Ok(unit_show("enabled")),
                Ok(Vec::new()),
                Ok(format_inspect(&id, &cid)),
                Ok(unit_show(boot)),
                Ok(container_inspect(&id, "42", Some(&p), running, "10.0.0.5")),
            ]);
            let rt = Runtime {
                exec: &mock,
                config: test_config(),
            };
            let out = rt
                .lifecycle(
                    &Lifecycle {
                        project: id.clone(),
                        action: action.to_string(),
                    },
                    deadline(),
                )
                .unwrap();
            assert_eq!(out.environment.id, id);
            assert_eq!(out.environment.running, running);
            assert_eq!(out.boot_enabled, boot == "enabled");
            let calls = mock.calls.borrow();
            assert_eq!(calls.len(), 6);
            // Container identity fenced both sides of the action.
            assert_eq!(calls[0].0, "/usr/bin/podman");
            assert_eq!(
                calls[0].1,
                vec![
                    "--remote=false".to_string(),
                    "inspect".to_string(),
                    "--format".to_string(),
                    PROJECT_INSPECT_FORMAT.to_string(),
                    format!("soda-{id}"),
                ]
            );
            assert_eq!(calls[3], calls[0]);
            // Unit observed before and after; action applied between.
            let unit = format!("soda-project@{id}.service");
            assert_eq!(calls[1].0, "/usr/bin/systemctl");
            assert_eq!(
                calls[1].1,
                vec![
                    "show".to_string(),
                    unit.clone(),
                    "--property=LoadState,FragmentPath,DropInPaths,UnitFileState".to_string(),
                ]
            );
            assert_eq!(calls[2].0, "/usr/bin/systemctl");
            assert_eq!(
                calls[2].1,
                vec![verb.to_string(), "--now".to_string(), unit.clone()]
            );
            assert_eq!(calls[4].0, "/usr/bin/systemctl");
            assert_eq!(calls[5].0, "/usr/bin/podman");
            assert_eq!(
                calls[5].1,
                vec!["inspect".to_string(), format!("soda-{id}")]
            );
        }
        // Inspect issues no systemctl action.
        let mock = Mock::new(vec![
            Ok(format_inspect(&id, &cid)),
            Ok(unit_show("disabled")),
            Ok(format_inspect(&id, &cid)),
            Ok(unit_show("disabled")),
            Ok(container_inspect(&id, "42", Some(&p), false, "10.0.0.5")),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let out = rt
            .lifecycle(
                &Lifecycle {
                    project: id.clone(),
                    action: "inspect".to_string(),
                },
                deadline(),
            )
            .unwrap();
        assert!(!out.environment.running && !out.boot_enabled);
        let calls = mock.calls.borrow();
        assert_eq!(calls.len(), 5);
        assert!(calls.iter().all(|(cmd, args)| cmd != "/usr/bin/systemctl"
            || args.first().map(String::as_str) == Some("show")));
    }

    #[test]
    fn lifecycle_rejects_identity_change_and_bad_outcome() {
        let id = format!("p{}", "e".repeat(24));
        let p = sample_profile();
        // Invalid action.
        let mock = Mock::new(Vec::new());
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.lifecycle(
                &Lifecycle {
                    project: id.clone(),
                    action: "restart".to_string(),
                },
                deadline()
            )
            .unwrap_err(),
            "invalid lifecycle operation"
        );
        // Container identity changed across the action.
        let mock = Mock::new(vec![
            Ok(format_inspect(&id, &"f".repeat(64))),
            Ok(unit_show("enabled")),
            Ok(Vec::new()),
            Ok(format_inspect(&id, &"e".repeat(64))),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.lifecycle(
                &Lifecycle {
                    project: id.clone(),
                    action: "start".to_string(),
                },
                deadline()
            )
            .unwrap_err(),
            "project identity changed during operation"
        );
        // Start that reports stopped is unconfirmed.
        let mock = Mock::new(vec![
            Ok(format_inspect(&id, &"f".repeat(64))),
            Ok(unit_show("enabled")),
            Ok(Vec::new()),
            Ok(format_inspect(&id, &"f".repeat(64))),
            Ok(unit_show("enabled")),
            Ok(container_inspect(&id, "42", Some(&p), false, "")),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.lifecycle(
                &Lifecycle {
                    project: id.clone(),
                    action: "start".to_string(),
                },
                deadline()
            )
            .unwrap_err(),
            "native lifecycle outcome unconfirmed"
        );
        // Unit observation failure.
        let mock = Mock::new(vec![
            Ok(format_inspect(&id, &"f".repeat(64))),
            Err("systemctl blew up".to_string()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.lifecycle(
                &Lifecycle {
                    project: id.clone(),
                    action: "inspect".to_string(),
                },
                deadline()
            )
            .unwrap_err(),
            "native unit unavailable"
        );
        // Action failure.
        let mock = Mock::new(vec![
            Ok(format_inspect(&id, &"f".repeat(64))),
            Ok(unit_show("enabled")),
            Err("systemctl blew up".to_string()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.lifecycle(
                &Lifecycle {
                    project: id.clone(),
                    action: "stop".to_string(),
                },
                deadline()
            )
            .unwrap_err(),
            "native lifecycle outcome unconfirmed"
        );
        // State encoding.
        let state = LifecycleState {
            environment: domain::Environment {
                image: String::new(),
                profile: None,
                id: id.clone(),
                ip: "10.0.0.5".to_string(),
                running: true,
            },
            boot_enabled: true,
        };
        assert_eq!(
            state.encode(),
            format!("{{\"environment\":{{\"id\":{id:?},\"ip\":\"10.0.0.5\",\"running\":true}},\"boot_enabled\":true}}")
        );
    }

    #[test]
    fn lifecycle_facade_confirmations() {
        let id = format!("p{}", "e".repeat(24));
        let env = domain::Environment {
            image: format!("sha256:{}", "a".repeat(64)),
            profile: Some(sample_profile()),
            id: id.clone(),
            ip: "10.0.0.5".to_string(),
            running: true,
        };
        let running = LifecycleState {
            environment: env.clone(),
            boot_enabled: true,
        };
        let start = Lifecycle {
            project: id.clone(),
            action: "start".to_string(),
        };
        assert!(confirm_lifecycle(&start, &running).is_ok());
        assert_eq!(
            confirm_lifecycle(
                &start,
                &LifecycleState {
                    environment: domain::Environment {
                        running: false,
                        ..env.clone()
                    },
                    boot_enabled: true,
                }
            )
            .unwrap_err(),
            "native lifecycle outcome not confirmed"
        );
        let stop = Lifecycle {
            project: id.clone(),
            action: "stop".to_string(),
        };
        let stopped = LifecycleState {
            environment: domain::Environment {
                running: false,
                ..env.clone()
            },
            boot_enabled: false,
        };
        assert!(confirm_lifecycle(&stop, &stopped).is_ok());
        // Wrong project.
        assert_eq!(
            confirm_lifecycle(
                &stop,
                &LifecycleState {
                    environment: domain::Environment {
                        id: "p000000000000000000000000".to_string(),
                        ..env.clone()
                    },
                    boot_enabled: false,
                }
            )
            .unwrap_err(),
            "native lifecycle outcome not confirmed"
        );
        // Bad endpoint address.
        assert_eq!(
            confirm_lifecycle(
                &stop,
                &LifecycleState {
                    environment: domain::Environment {
                        ip: "127.0.0.1".to_string(),
                        running: false,
                        ..env.clone()
                    },
                    boot_enabled: false,
                }
            )
            .unwrap_err(),
            "native lifecycle outcome not confirmed"
        );
        // Inspect confirms any settled state.
        let inspect = Lifecycle {
            project: id.clone(),
            action: "inspect".to_string(),
        };
        assert!(confirm_lifecycle(&inspect, &running).is_ok());
        assert!(confirm_lifecycle(&inspect, &stopped).is_ok());
    }

    #[test]
    fn address_profile_os_key_confirmations() {
        assert!(valid_address("10.0.0.5"));
        assert!(valid_address("2001:db8::1"));
        assert!(!valid_address(""));
        assert!(!valid_address("not-an-ip"));
        assert!(!valid_address("0.0.0.0"));
        assert!(!valid_address("127.0.0.1"));
        assert!(!valid_address("::1"));
        assert!(!valid_address("224.0.0.1"));
        assert!(!valid_address("fe80::1%eth0"));

        // Profiles confirm native images; foreign arches overwrite validation.
        let native = domain::Profile {
            architecture: match std::env::consts::ARCH {
                "x86_64" => "amd64".to_string(),
                "aarch64" => "arm64".to_string(),
                other => other.to_string(),
            },
            ..sample_profile()
        };
        assert!(confirm_resolve_profile(&native).is_ok());
        let mut foreign = native.clone();
        foreign.architecture = if foreign.architecture == "amd64" {
            "arm64".to_string()
        } else {
            "amd64".to_string()
        };
        assert_eq!(
            confirm_resolve_profile(&foreign).unwrap_err(),
            "project image is not native to this backend"
        );
        let mut broken = native.clone();
        broken.version = String::new();
        assert!(confirm_resolve_profile(&broken).is_err());

        // OS observations confirm shape and liveness.
        let id = format!("p{}", "e".repeat(24));
        let env = domain::Environment {
            image: String::new(),
            profile: None,
            id: id.clone(),
            ip: "10.0.0.5".to_string(),
            running: true,
        };
        let release = domain::OsRelease {
            id: "rocky".to_string(),
            version: "9.7".to_string(),
            name: "Rocky Linux 9.7".to_string(),
        };
        let observed = domain::OsObservation {
            environment: env.clone(),
            release: Some(release.clone()),
            unavailable: false,
        };
        assert!(confirm_observe_os(&id, &observed).is_ok());
        let stopped = domain::OsObservation {
            environment: domain::Environment {
                running: false,
                ..env.clone()
            },
            release: None,
            unavailable: true,
        };
        assert!(confirm_observe_os(&id, &stopped).is_ok());
        // Release on a stopped environment is inconsistent.
        assert_eq!(
            confirm_observe_os(
                &id,
                &domain::OsObservation {
                    environment: domain::Environment {
                        running: false,
                        ..env.clone()
                    },
                    release: Some(release),
                    unavailable: false,
                }
            )
            .unwrap_err(),
            "invalid native OS observation"
        );
        // Mismatched availability flag.
        assert_eq!(
            confirm_observe_os(
                &id,
                &domain::OsObservation {
                    environment: env.clone(),
                    release: None,
                    unavailable: false,
                }
            )
            .unwrap_err(),
            "invalid native OS observation"
        );

        // Key results confirm canonical sets and revisions.
        let key =
            "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre";
        let input = domain::AccessKeys {
            project: id.clone(),
            login: "alice".to_string(),
            identity: 1,
            revision: "b".repeat(64),
            keys: vec![key.to_string()],
            apply: true,
        };
        let out = domain::AccessKeyState {
            revision: "b".repeat(64),
            keys: vec![key.to_string()],
        };
        assert!(confirm_access_keys(&input, &out).is_ok());
        assert_eq!(
            confirm_access_keys(
                &input,
                &domain::AccessKeyState {
                    revision: "short".to_string(),
                    keys: vec![key.to_string()],
                }
            )
            .unwrap_err(),
            "invalid native key revision"
        );
        assert_eq!(
            confirm_access_keys(
                &input,
                &domain::AccessKeyState {
                    revision: "b".repeat(64),
                    keys: Vec::new(),
                }
            )
            .unwrap_err(),
            "native key result differs from request"
        );
        // Noncanonical member keys surface the canonical error on preview.
        let preview = domain::AccessKeys {
            apply: false,
            revision: String::new(),
            keys: Vec::new(),
            ..input.clone()
        };
        assert_eq!(
            confirm_access_keys(
                &preview,
                &domain::AccessKeyState {
                    revision: "b".repeat(64),
                    keys: vec!["not-a-key".to_string()],
                }
            )
            .unwrap_err(),
            "invalid development key"
        );
    }

    #[test]
    fn native_run_drains_large_dual_streams_intact() {
        // >64 KiB on stdout with a live stderr: the pre-H01-F2
        // reap-before-drain order deadlocked here.
        let out = Native
            .run(
                b"",
                "sh",
                &["-c", "head -c 70000 /dev/zero | tr '\\0' 'A'"],
                Instant::now() + Duration::from_secs(5),
            )
            .unwrap();
        assert_eq!(out, vec![b'A'; 70000]);
    }

    #[test]
    fn native_run_failure_shape_carries_full_stderr() {
        let err = Native
            .run(
                b"",
                "sh",
                &[
                    "-c",
                    "head -c 70000 /dev/zero | tr '\\0' 'A'; head -c 70000 /dev/zero | tr '\\0' 'B' >&2; exit 3",
                ],
                Instant::now() + Duration::from_secs(5),
            )
            .unwrap_err();
        assert_eq!(
            err,
            format!("sh failed: exit status 3: {}", "B".repeat(70000))
        );
    }

    #[test]
    fn native_run_deadline_kills_promptly() {
        let start = Instant::now();
        let err = Native
            .run(
                b"",
                "sleep",
                &["30"],
                Instant::now() + Duration::from_secs(1),
            )
            .unwrap_err();
        assert_eq!(err, "sleep failed: deadline exceeded");
        assert!(start.elapsed() < Duration::from_secs(10), "kill not prompt");
    }
}
