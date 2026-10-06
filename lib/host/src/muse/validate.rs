use std::ffi::OsString;

use super::NestedRegistration;
use crate::domain;
use crate::json::{self, Value};
use crate::terminal::{self, Binding, Delivery};

// ---------- pure resolution helpers (muse_linux.go) ----------

/// `museInspect`: exact podman `--format` template.
pub const MUSE_INSPECT: &str = "{\"id\":{{json .ID}},\"pid\":{{json .State.Pid}},\"project\":{{json (index .Config.Labels \"org.soda.project\")}},\"running\":{{json .State.Running}},\"privileged\":{{json .HostConfig.Privileged}},\"userns\":{{json .HostConfig.UsernsMode}}}";
/// Nested-child inspect template.
pub const MUSE_CHILD_INSPECT: &str =
    "{\"id\":{{json .ID}},\"pid\":{{json .State.Pid}},\"running\":{{json .State.Running}}}";

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
