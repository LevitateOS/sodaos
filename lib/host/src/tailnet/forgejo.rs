//! Private Rust port of the `soda-forgejo-tailnet` refresh helper
//! (`cmd/soda-forgejo-tailnet/main.go` + `internal/forgejo/tailnet.go`):
//! root gate, 90s deadline, native Tailnet endpoint, `podman inspect`
//! listener admission, SSH_DOMAIN-only env rewrite, conditional restart.
//! Included by the thin `soda-forgejo-tailnet` bin; not part of the
//! `soda_host` library API, so only `soda_host::` paths are used here.

use std::os::unix::fs::OpenOptionsExt;
use std::os::unix::io::{IntoRawFd, RawFd};
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, Instant};

use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use soda_host::project::{Executor, NativeStatusOnly};
use soda_host::tcontrol_native;

const DEADLINE_SECS: u64 = 90;
const PODMAN: &str = "/usr/bin/podman";
const SYSTEMCTL: &str = "/usr/bin/systemctl";
const FORGEJO_CONTAINER: &str = "soda-forgejo";
const FORGEJO_ENV: &str = "/etc/soda/forgejo.env";
const SSH_DOMAIN_KEY: &str = "FORGEJO__server__SSH_DOMAIN=";

/// Full refresh flow; mirrors Go `run`.
pub fn run(exec: &dyn Executor) -> Result<(), String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err(String::from("host operator required"));
    }
    let deadline = Instant::now() + Duration::from_secs(DEADLINE_SECS);
    // CLI diagnostics are status-only by design: the shared runner with
    // the status-only error policy reports Go's documented exit/signal/
    // spawn head shape, but intentionally suppresses the trimmed-stderr
    // detail the retired Go Status appended. Byte parity against the
    // retired helper is unproved; further diagnostics come from the shared
    // observer, never subprocess stderr.
    let status_exec = NativeStatusOnly;
    let runtime = soda_host::tailnet_host::HostRuntime::new(&status_exec);
    let endpoint = tcontrol_native::cli_endpoint(&|args, end| runtime.cli(args, end), deadline)?;
    // Do not advertise a Tailnet address while the native port only binds a LAN IP.
    // Raw inspection may contain credentials; never print it or include it in errors.
    let live = inspect_forgejo(exec, deadline)?;
    let (domain, running) = published_state(&live, &endpoint.ipv4)?;
    let changed = update_ssh_domain(FORGEJO_ENV, &endpoint.identity)?;
    // The restart shares the status-only runner: no subprocess stderr by
    // design. Go's Run also discarded stderr, but byte parity against the
    // retired helper is unproved; only the tested status shapes are pinned.
    restart_forgejo_if_needed(
        &status_exec,
        changed,
        &domain,
        &endpoint.identity,
        running,
        deadline,
    )?;
    println!("Forgejo SSH address refreshed; configured browser/OAuth origins preserved.");
    Ok(())
}

fn inspect_forgejo(exec: &dyn Executor, deadline: Instant) -> Result<Vec<u8>, String> {
    exec.run(&[], PODMAN, &["inspect", FORGEJO_CONTAINER], deadline)
        .map_err(|_| {
            String::from("cannot inspect native Forgejo; inspect its operator service journal")
        })
}

#[derive(Clone, Default)]
struct PodBinding {
    host_ip: Option<String>,
    host_port: Option<String>,
}

impl<'de> Deserialize<'de> for PodBinding {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = PodBinding;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("port binding")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = PodBinding::default();
                while let Some(k) = map.next_key::<String>()? {
                    if k.eq_ignore_ascii_case("HostIp") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            o.host_ip = Some(v)
                        }
                    } else if k.eq_ignore_ascii_case("HostPort") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            o.host_port = Some(v)
                        }
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct HostConfig {
    port_bindings: Option<std::collections::BTreeMap<String, Option<Vec<PodBinding>>>>,
}

impl<'de> Deserialize<'de> for HostConfig {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = HostConfig;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("podman host config")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = HostConfig::default();
                while let Some(k) = map.next_key::<String>()? {
                    if k.eq_ignore_ascii_case("PortBindings") {
                        o.port_bindings = map.next_value()?;
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct PodConfig {
    env: Option<Vec<Option<String>>>,
}

impl<'de> Deserialize<'de> for PodConfig {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = PodConfig;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("podman config")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = PodConfig::default();
                while let Some(k) = map.next_key::<String>()? {
                    if k.eq_ignore_ascii_case("Env") {
                        o.env = map.next_value()?;
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct PodState {
    running: Option<bool>,
}

impl<'de> Deserialize<'de> for PodState {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = PodState;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("podman state")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = PodState::default();
                while let Some(k) = map.next_key::<String>()? {
                    if k.eq_ignore_ascii_case("Running") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            o.running = Some(v)
                        }
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct PodInspect {
    host_config: Option<HostConfig>,
    config: Option<PodConfig>,
    state: Option<PodState>,
}
impl<'de> Deserialize<'de> for PodInspect {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = PodInspect;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("podman inspection")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = PodInspect::default();
                while let Some(k) = map.next_key::<String>()? {
                    if k.eq_ignore_ascii_case("HostConfig") {
                        o.host_config = map.next_value()?;
                    } else if k.eq_ignore_ascii_case("Config") {
                        o.config = map.next_value()?;
                    } else if k.eq_ignore_ascii_case("State") {
                        o.state = map.next_value()?;
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

fn published_state(data: &[u8], ip: &str) -> Result<(String, bool), String> {
    let invalid = || String::from("cannot read native Forgejo network state");
    let items: Vec<Option<PodInspect>> =
        soda_host::json::decode_tolerant_as(data).map_err(|_| invalid())?;
    if items.len() != 1 {
        return Err(invalid());
    }
    let root = items
        .into_iter()
        .next()
        .flatten()
        .ok_or_else(|| invalid())?;
    let bindings = root
        .host_config
        .and_then(|host| host.port_bindings)
        .and_then(|map| map.get("22/tcp").cloned().flatten())
        .unwrap_or_default()
        .into_iter()
        .map(|entry| {
            (
                entry.host_ip.unwrap_or_default(),
                entry.host_port.unwrap_or_default(),
            )
        })
        .collect::<Vec<_>>();
    let domain = root
        .config
        .and_then(|cfg| cfg.env)
        .unwrap_or_default()
        .into_iter()
        .flatten()
        .find_map(|line| line.strip_prefix(SSH_DOMAIN_KEY).map(str::to_owned))
        .unwrap_or_default();
    let running = root
        .state
        .and_then(|state| state.running)
        .unwrap_or_default();
    if !ssh_bound_to_ip(&bindings, ip) {
        return Err(format!(
            "forgejo Git SSH is not bound to Tailnet IP {ip}:2222; configure the intended private native listener before refreshing its advertised address"
        ));
    }
    Ok((domain, running))
}

fn ssh_bound_to_ip(bindings: &[(String, String)], ip: &str) -> bool {
    bindings.iter().any(|(host_ip, host_port)| {
        host_port == "2222" && (host_ip == ip || host_ip == "0.0.0.0" || host_ip.is_empty())
    })
}

fn valid_endpoint_name(endpoint: &str) -> bool {
    // ^[A-Za-z0-9][A-Za-z0-9.:-]*$ without a regex dependency.
    let mut chars = endpoint.chars();
    match chars.next() {
        Some(c) if c.is_ascii_alphanumeric() => {}
        _ => return false,
    }
    chars.all(|c| c.is_ascii_alphanumeric() || matches!(c, '.' | ':' | '-'))
}

fn update_ssh_domain(path: &str, endpoint: &str) -> Result<bool, String> {
    if !valid_endpoint_name(endpoint) {
        return Err(String::from("invalid native Tailnet endpoint"));
    }
    let before = std::fs::read(path).map_err(|e| format!("open {path}: {e}"))?;
    let updated = rewrite_ssh_domain(&before, endpoint);
    if updated == before {
        return Ok(false);
    }
    write_forgejo_env(path, &updated)?;
    Ok(true)
}

/// Byte-level port of Go `rewriteSSHDomain`, so non-UTF8 files round-trip
/// exactly like the Go string rewrite: first key line replaced, later key
/// lines dropped, key appended when missing, single trailing newline.
fn rewrite_ssh_domain(content: &[u8], endpoint: &str) -> Vec<u8> {
    let key = SSH_DOMAIN_KEY.as_bytes();
    let value = [key, endpoint.as_bytes()].concat();
    let mut trimmed = content;
    while trimmed.last() == Some(&b'\n') {
        trimmed = &trimmed[..trimmed.len() - 1];
    }
    let mut found = false;
    let mut out: Vec<&[u8]> = Vec::new();
    for line in trimmed.split(|b| *b == b'\n') {
        if line.starts_with(key) {
            if !found {
                out.push(&value);
                found = true;
            }
            continue;
        }
        out.push(line);
    }
    if !found {
        out.push(&value);
    }
    let mut joined = Vec::new();
    for (i, line) in out.iter().enumerate() {
        if i > 0 {
            joined.push(b'\n');
        }
        joined.extend_from_slice(line);
    }
    joined.push(b'\n');
    joined
}

/// 0600 private temp in the target directory, chmod/write/close checks,
/// atomic rename; the temp is removed on every failure path.
fn write_forgejo_env(path: &str, updated: &[u8]) -> Result<(), String> {
    static TEMP_SEQ: AtomicU64 = AtomicU64::new(0);
    let dir = match std::path::Path::new(path).parent() {
        Some(p) if !p.as_os_str().is_empty() => p.to_path_buf(),
        _ => std::path::PathBuf::from("."),
    };
    let pid = std::process::id();
    let mut temp_path = std::path::PathBuf::new();
    let mut file = None;
    let mut last_err = String::new();
    for _ in 0..100 {
        let id = TEMP_SEQ.fetch_add(1, Ordering::Relaxed);
        temp_path = dir.join(format!(".forgejo-env-{pid}-{id}"));
        match std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(&temp_path)
        {
            Ok(f) => {
                file = Some(f);
                break;
            }
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(e) => {
                last_err = format!("open {}: {e}", temp_path.display());
                break;
            }
        }
    }
    let file = match file {
        Some(f) => f,
        None if last_err.is_empty() => {
            return Err(format!(
                "create temp file in {}: too many collisions",
                dir.display()
            ));
        }
        None => return Err(last_err),
    };
    // Owned-descriptor staging with a checked close, mirroring Go's
    // explicit close check that File's Drop cannot report. The fd closes
    // exactly once on every path below; rename runs only after a
    // successful chmod, write, and close.
    let fd = file.into_raw_fd();
    let staged = stage_forgejo_env(fd, &temp_path, updated);
    let closed = close_owned(fd).map_err(|e| format!("close {}: {e}", temp_path.display()));
    if let Err(e) = staged {
        let _ = std::fs::remove_file(&temp_path);
        return Err(e);
    }
    if let Err(e) = closed {
        let _ = std::fs::remove_file(&temp_path);
        return Err(e);
    }
    if let Err(e) = std::fs::rename(&temp_path, path) {
        let _ = std::fs::remove_file(&temp_path);
        return Err(format!("rename {} {path}: {e}", temp_path.display()));
    }
    Ok(())
}

/// chmod + full write on an owned staged descriptor; the caller owns the
/// checked close. First error wins, like Go's staged checks.
fn stage_forgejo_env(fd: RawFd, temp: &std::path::Path, updated: &[u8]) -> Result<(), String> {
    if unsafe { libc::fchmod(fd, 0o600) } != 0 {
        return Err(format!(
            "chmod {}: {}",
            temp.display(),
            std::io::Error::last_os_error()
        ));
    }
    write_full(temp, updated, |buf| {
        let n = unsafe { libc::write(fd, buf.as_ptr() as *const libc::c_void, buf.len()) };
        if n < 0 {
            Err(std::io::Error::last_os_error())
        } else {
            Ok(n as usize)
        }
    })
}

/// Complete write with EINTR retry. A zero write fails like
/// `File::write_all`'s WriteZero instead of spinning forever.
fn write_full(
    temp: &std::path::Path,
    mut remaining: &[u8],
    mut write: impl FnMut(&[u8]) -> Result<usize, std::io::Error>,
) -> Result<(), String> {
    while !remaining.is_empty() {
        match write(remaining) {
            Ok(0) => {
                return Err(format!(
                    "write {}: {}",
                    temp.display(),
                    std::io::Error::new(
                        std::io::ErrorKind::WriteZero,
                        "failed to write whole buffer"
                    )
                ));
            }
            Ok(n) => remaining = &remaining[n.min(remaining.len())..],
            Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
            Err(e) => return Err(format!("write {}: {e}", temp.display())),
        }
    }
    Ok(())
}

/// Checked close for an owned descriptor.
fn close_owned(fd: RawFd) -> Result<(), std::io::Error> {
    if unsafe { libc::close(fd) } == 0 {
        Ok(())
    } else {
        Err(std::io::Error::last_os_error())
    }
}

fn restart_forgejo_if_needed(
    exec: &dyn Executor,
    changed: bool,
    domain: &str,
    identity: &str,
    running: bool,
    deadline: Instant,
) -> Result<(), String> {
    if changed || domain != identity || !running {
        exec.run(&[], SYSTEMCTL, &["restart", "forgejo.service"], deadline)
            .map_err(|e| format!("native Forgejo configuration saved, restart failed: {e}"))?;
    }
    Ok(())
}
