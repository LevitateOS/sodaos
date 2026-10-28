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
use std::time::Instant;

use crate::domain;
use crate::json;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

pub mod admission;
mod connection;
mod create;
mod executor;
mod inspect;
mod os;
mod profile;

// `muse_serve_oracle` compiles this root through a private `#[path]` copy
// that never runs a restart; the re-export serves the real library
// (`tailnet/forgejo.rs`).
#[allow(unused_imports)]
pub use self::executor::{BoundedError, Executor, Native, NativeStatusOnly};
// `muse_serve_oracle` compiles this root through a private `#[path]` copy
// that never touches the crate-visible specs; the re-export serves the
// real library (`prepare/helper.rs`).
pub use self::inspect::PROJECT_INSPECT_FORMAT;
#[allow(unused_imports)]
pub(crate) use self::inspect::{
    decode_project_container_inspection, GoStringList, ProjectContainerInspection,
};

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

impl<'de> Deserialize<'de> for Lifecycle {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct LifecycleVisitor;
        impl<'de> Visitor<'de> for LifecycleVisitor {
            type Value = Lifecycle;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a lifecycle object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = Lifecycle::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("project") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.project = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("action") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.action = v;
                        }
                        continue;
                    }
                    return Err(de::Error::unknown_field(&key, &["project", "action"]));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(LifecycleVisitor)
    }
}

impl Lifecycle {
    /// Strict decode of one lifecycle request; fields are owned by this DTO.
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
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
        if input.action == "start" {
            match std::fs::symlink_metadata(format!(
                "/var/lib/soda/dashboard/member-access/{}",
                input.project
            )) {
                Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
                _ => {
                    return Err(
                        "member access operation pending; reconcile before Project start".into(),
                    )
                }
            }
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
