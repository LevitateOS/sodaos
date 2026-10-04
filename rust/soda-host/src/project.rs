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
use crate::ssh;

pub const PROFILE_INSPECT_FORMAT: &str =
    "{\"Id\":{{json .ID}},\"Architecture\":{{json .Architecture}},\"Os\":{{json .Os}},\"Labels\":{{json .Labels}}}";

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
        let outcome = std::thread::scope(|scope| {
            let writer = scope.spawn(|| {
                if let Some(mut w) = input.take() {
                    let _ = w.write_all(stdin);
                }
            });
            let status = loop {
                match child.try_wait().map_err(|e| format!("{cmd} failed: {e}"))? {
                    Some(status) => break status,
                    None => {
                        if Instant::now() >= deadline {
                            let _ = child.kill();
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
            Ok(status)
        })?;
        let status: std::process::ExitStatus = outcome;
        let mut stdout = Vec::new();
        let mut stderr = Vec::new();
        use std::io::Read;
        if let Some(mut o) = child.stdout.take() {
            let _ = o.read_to_end(&mut stdout);
        }
        if let Some(mut e) = child.stderr.take() {
            let _ = e.read_to_end(&mut stderr);
        }
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

    /// `ResolveProfile`: inspect only the configured installed image.
    pub fn resolve_profile(&self, deadline: Instant) -> Result<domain::Profile, String> {
        let raw = self.podman(
            &[],
            &[
                "image",
                "inspect",
                "--format",
                PROFILE_INSPECT_FORMAT,
                &self.config.image,
            ],
            deadline,
        )?;
        if raw.len() > 65536 {
            return Err("invalid installed image inspection".to_string());
        }
        let v = json::decode_tolerant(&raw)
            .map_err(|_| "invalid installed image inspection".to_string())?;
        let id = json::tolerant_get(&v, "Id")
            .and_then(Value::as_str)
            .unwrap_or("")
            .to_string();
        let architecture = json::tolerant_get(&v, "Architecture")
            .and_then(Value::as_str)
            .unwrap_or("")
            .to_string();
        let os = json::tolerant_get(&v, "Os")
            .and_then(Value::as_str)
            .unwrap_or("")
            .to_string();
        // encoding/json would fail the whole unmarshal on mistyped fields.
        for key in ["Id", "Architecture", "Os"] {
            if let Some(got) = json::tolerant_get(&v, key) {
                if got.as_str().is_none() {
                    return Err("invalid installed image inspection".to_string());
                }
            }
        }
        let labels = match json::tolerant_get(&v, "Labels") {
            None | Some(Value::Null) => HashMap::new(),
            Some(Value::Object(fields)) => {
                let mut map = HashMap::new();
                for (k, val) in fields {
                    match val {
                        Value::Str(s) => {
                            map.insert(k.clone(), s.clone());
                        }
                        Value::Null => {}
                        _ => return Err("invalid installed image inspection".to_string()),
                    }
                }
                map
            }
            Some(_) => return Err("invalid installed image inspection".to_string()),
        };
        let mut image_id = id;
        if !image_id.starts_with("sha256:") {
            image_id = format!("sha256:{image_id}");
        }
        let get = |k: &str| labels.get(k).cloned().unwrap_or_default();
        let profile = domain::Profile {
            id: get("org.soda.profile"),
            distribution: get("org.soda.distribution"),
            version: get("org.soda.distribution.version"),
            interface: get("org.soda.interface"),
            architecture: architecture.clone(),
            image: image_id,
            revision: get("org.opencontainers.image.revision"),
        };
        if os != "linux" || architecture != go_arch() {
            return Err("project image is not native Linux".to_string());
        }
        profile.validate()?;
        Ok(profile)
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
        apply_creation_profile(&mut env, &labels, &image)?;
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

    /// `Create`: provision and start a project environment. Callers hold
    /// the mutation gate before invoking this method.
    pub fn create(
        &self,
        input: &domain::Create,
        deadline: Instant,
    ) -> Result<domain::Environment, String> {
        input.validate()?;
        self.create_container(input, deadline)?;
        self.start_created(input, deadline)
    }

    fn create_container(&self, input: &domain::Create, deadline: Instant) -> Result<(), String> {
        let profile = self.resolve_profile(deadline)?;
        let want = input
            .profile
            .as_ref()
            .ok_or_else(|| "invalid creation identity".to_string())?;
        if profile != *want {
            return Err("installed profile changed; reservation retained".to_string());
        }
        let encoded = profile.encode();
        if self
            .podman(&[], &["network", "exists", &self.config.network], deadline)
            .is_err()
        {
            self.podman(
                &[],
                &[
                    "network",
                    "create",
                    "--driver",
                    "bridge",
                    "--subnet",
                    &self.config.subnet,
                    "--interface-name",
                    &self.config.bridge,
                    &self.config.network,
                ],
                deadline,
            )?;
        }
        let name = format!("soda-{}", input.id);
        let mut args: Vec<String> = vec![
            "create".to_string(),
            "--name".to_string(),
            name,
            "--label".to_string(),
            format!("org.soda.project={}", input.id),
            "--label".to_string(),
            format!("org.soda.owner={}", input.owner),
            "--network".to_string(),
            self.config.network.clone(),
            "--userns=auto:size=262144".to_string(),
            "--systemd=always".to_string(),
            "--cgroupns=private".to_string(),
            "--cap-add=SYS_ADMIN,MKNOD,NET_ADMIN,SYS_PTRACE".to_string(),
            "--device=/dev/fuse".to_string(),
            "--security-opt=label=disable".to_string(),
            "--label".to_string(),
            format!("org.soda.profile={}", profile.id),
            "--label".to_string(),
            format!("org.soda.creation-profile={encoded}"),
            "--pull=never".to_string(),
        ];
        if !self.config.muse_socket.is_empty() {
            args.push("--volume".to_string());
            args.push(format!(
                "{}:/run/soda-muse-interface:ro",
                go_dir(&self.config.muse_socket)
            ));
        }
        args.push(profile.image.clone());
        let refs: Vec<&str> = args.iter().map(String::as_str).collect();
        self.podman(&[], &refs, deadline)?;
        Ok(())
    }

    fn start_created(
        &self,
        input: &domain::Create,
        deadline: Instant,
    ) -> Result<domain::Environment, String> {
        let name = format!("soda-{}", input.id);
        let unit = format!("soda-project@{}.service", input.id);
        self.exec.run(
            &[],
            "/usr/bin/systemctl",
            &["enable", "--now", &unit],
            deadline,
        )?;
        self.wait_project_ready(&name, deadline)?;
        let (env, _) = self.inspect(&input.id, deadline)?;
        let want = input
            .profile
            .as_ref()
            .ok_or_else(|| "invalid creation identity".to_string())?;
        match &env.profile {
            Some(got) if env.running && !env.ip.is_empty() && got == want => Ok(env),
            _ => {
                Err("project did not report the expected profile and running endpoint".to_string())
            }
        }
    }

    fn wait_project_ready(&self, name: &str, deadline: Instant) -> Result<(), String> {
        loop {
            if self
                .podman(
                    &[],
                    &[
                        "exec",
                        name,
                        "/usr/bin/test",
                        "-f",
                        "/run/soda-project-ready",
                    ],
                    deadline,
                )
                .is_ok()
            {
                return Ok(());
            }
            if Instant::now() >= deadline {
                return Err("deadline exceeded".to_string());
            }
            std::thread::sleep(Duration::from_secs(1));
            if Instant::now() >= deadline {
                return Err("deadline exceeded".to_string());
            }
        }
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

    /// `ObserveOS`: read-only OS release observation, never starting a
    /// stopped container. Unparseable or missing data reports unavailable.
    pub fn observe_os(&self, id: &str, deadline: Instant) -> Result<domain::OsObservation, String> {
        let (env, _) = self.inspect(id, deadline)?;
        let mut result = domain::OsObservation {
            environment: env.clone(),
            release: None,
            unavailable: true,
        };
        if !env.running {
            return Ok(result);
        }
        // Fast regular-file refusal (the Python helper opened O_NONBLOCK and
        // required S_ISREG); the read below still bounds the window.
        let target = format!("soda-{id}");
        if self
            .podman(
                &[],
                &["exec", &target, "/usr/bin/test", "-f", "/etc/os-release"],
                deadline,
            )
            .is_err()
        {
            return Ok(result);
        }
        let raw = match self.podman(
            &[],
            &[
                "exec",
                &target,
                "/usr/bin/head",
                "-c",
                "4097",
                "/etc/os-release",
            ],
            deadline,
        ) {
            Ok(raw) if raw.len() <= 4096 => raw,
            _ => return Ok(result),
        };
        let release = match parse_os_release(&raw) {
            Ok(r) => r,
            Err(_) => return Ok(result),
        };
        if !domain::valid_os_release(&release) {
            return Ok(result);
        }
        result.release = Some(release);
        result.unavailable = false;
        Ok(result)
    }

    /// `Connection`: environment plus the fixed public Ed25519 host key.
    pub fn connection(&self, id: &str, deadline: Instant) -> Result<domain::Connection, String> {
        let (env, _) = self.inspect(id, deadline)?;
        let mut result = domain::Connection {
            environment: env,
            host_key: String::new(),
            fingerprint: String::new(),
        };
        if !result.environment.running {
            return Ok(result);
        }
        let data = match self.podman(
            &[],
            &[
                "exec",
                &format!("soda-{id}"),
                "/usr/bin/head",
                "-c",
                "16385",
                "/etc/ssh/ssh_host_ed25519_key.pub",
            ],
            deadline,
        ) {
            Ok(data) if data.len() <= 16384 => data,
            _ => return Err("public host key unavailable".to_string()),
        };
        let (key, _) =
            ssh::parse_authorized_key(&data).map_err(|_| "invalid public host key".to_string())?;
        if key.key_type != ssh::ALGO_ED25519 {
            return Err("invalid public host key".to_string());
        }
        result.host_key = ssh::marshal_authorized_key(&key.key_type, &key.blob);
        result.fingerprint = ssh::fingerprint_sha256(&key.blob);
        Ok(result)
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

/// `path/filepath.Dir` for socket paths.
fn go_dir(path: &str) -> &str {
    let trimmed = path.trim_end_matches('/');
    if trimmed.is_empty() {
        return "/";
    }
    match trimmed.rfind('/') {
        None => ".",
        Some(0) => "/",
        Some(i) => {
            let dir = trimmed[..i].trim_end_matches('/');
            if dir.is_empty() {
                "/"
            } else {
                dir
            }
        }
    }
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

fn apply_creation_profile(
    env: &mut domain::Environment,
    labels: &HashMap<String, String>,
    image: &str,
) -> Result<(), String> {
    let raw = labels
        .get("org.soda.creation-profile")
        .map(String::as_str)
        .unwrap_or("");
    if raw.is_empty() {
        return Ok(());
    }
    let profile =
        domain::decode_profile(raw).map_err(|_| "native creation profile mismatch".to_string())?;
    let mut expected = image.to_string();
    if !expected.starts_with("sha256:") {
        expected = format!("sha256:{expected}");
    }
    if profile.id
        != labels
            .get("org.soda.profile")
            .map(String::as_str)
            .unwrap_or("")
        || profile.image != expected
    {
        return Err("native creation profile mismatch".to_string());
    }
    env.profile = Some(profile);
    Ok(())
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

// ---------- os-release parsing (project_os.py rules) ----------

fn is_python_space(c: char) -> bool {
    matches!(
        c,
        '\u{0009}'..='\u{000D}'
            | ' '
            | '\u{0085}'
            | '\u{00A0}'
            | '\u{1680}'
            | '\u{2000}'..='\u{200A}'
            | '\u{2028}'
            | '\u{2029}'
            | '\u{202F}'
            | '\u{205F}'
            | '\u{3000}'
    )
}

fn split_lines(text: &str) -> Vec<&str> {
    // Python str.splitlines boundaries.
    let mut lines = Vec::new();
    let mut start = 0;
    let bytes = text.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        let c = text[i..].chars().next().unwrap();
        let len = c.len_utf8();
        let boundary = matches!(
            c,
            '\n' | '\r'
                | '\u{000B}'
                | '\u{000C}'
                | '\u{001C}'
                | '\u{001D}'
                | '\u{001E}'
                | '\u{0085}'
                | '\u{2028}'
                | '\u{2029}'
        );
        if boundary {
            lines.push(&text[start..i]);
            if c == '\r' && bytes.get(i + 1) == Some(&b'\n') {
                i += 1;
            }
            i += len;
            start = i;
        } else {
            i += len;
        }
    }
    lines.push(&text[start..]);
    // Python drops the final empty element after a trailing break.
    if text.is_empty() {
        return Vec::new();
    }
    if lines.last() == Some(&"") && start == text.len() && !text.is_empty() {
        lines.pop();
    }
    lines
}

/// POSIX shell word parsing for one os-release value: `shlex.split(value,
/// comments=False, posix=True)` semantics with `#` ordinary.
fn shlex_posix(value: &str) -> Result<Vec<String>, ()> {
    let mut words = Vec::new();
    let mut word = String::new();
    let mut started = false;
    let mut chars = value.chars().peekable();
    while let Some(c) = chars.next() {
        if matches!(c, ' ' | '\t' | '\r' | '\n') {
            if started {
                words.push(std::mem::take(&mut word));
                started = false;
            }
            continue;
        }
        started = true;
        match c {
            '\'' => {
                let mut closed = false;
                for c in chars.by_ref() {
                    if c == '\'' {
                        closed = true;
                        break;
                    }
                    word.push(c);
                }
                if !closed {
                    return Err(());
                }
            }
            '"' => {
                let mut closed = false;
                while let Some(c) = chars.next() {
                    if c == '"' {
                        closed = true;
                        break;
                    }
                    if c == '\\' {
                        match chars.next() {
                            Some(n @ ('$' | '`' | '"' | '\\')) => word.push(n),
                            Some('\n') => {}
                            Some(other) => {
                                word.push('\\');
                                word.push(other);
                            }
                            None => return Err(()),
                        }
                    } else {
                        word.push(c);
                    }
                }
                if !closed {
                    return Err(());
                }
            }
            '\\' => match chars.next() {
                Some(n) => word.push(n),
                None => return Err(()),
            },
            c => word.push(c),
        }
    }
    if started {
        words.push(word);
    }
    Ok(words)
}

fn valid_release_id(s: &str) -> bool {
    let b = s.as_bytes();
    !b.is_empty()
        && b.len() <= 64
        && (b[0].is_ascii_lowercase() || b[0].is_ascii_digit())
        && b[1..].iter().all(|c| {
            c.is_ascii_lowercase() || c.is_ascii_digit() || *c == b'.' || *c == b'_' || *c == b'-'
        })
}

fn valid_release_version(s: &str) -> bool {
    let b = s.as_bytes();
    !b.is_empty()
        && b.len() <= 64
        && b[0].is_ascii_alphanumeric()
        && b[1..]
            .iter()
            .all(|c| c.is_ascii_alphanumeric() || *c == b'.' || *c == b'_' || *c == b'-')
}

pub fn parse_os_release(raw: &[u8]) -> Result<domain::OsRelease, String> {
    let text = std::str::from_utf8(raw).map_err(|_| "invalid OS release text".to_string())?;
    let mut values: HashMap<&str, String> = HashMap::new();
    for line in split_lines(text) {
        if line.trim_matches(is_python_space).is_empty() {
            continue;
        }
        if line.trim_start_matches(is_python_space).starts_with('#') {
            continue;
        }
        let (key, sep, value) = match line.find('=') {
            Some(i) => (&line[..i], "=", &line[i + 1..]),
            None => (line, "", ""),
        };
        if key != "ID" && key != "VERSION_ID" && key != "PRETTY_NAME" {
            continue;
        }
        if sep.is_empty() || values.contains_key(key) {
            return Err("ambiguous OS release field".to_string());
        }
        let mut words = shlex_posix(value).map_err(|_| "invalid OS release field".to_string())?;
        if words.len() != 1 || words[0].is_empty() || words[0].len() > 256 {
            return Err("invalid OS release field".to_string());
        }
        if words[0].chars().any(|c| (c as u32) < 32 || c as u32 == 127) {
            return Err("invalid OS release text".to_string());
        }
        values.insert(key, words.remove(0));
    }
    let id = values.get("ID").map(String::as_str).unwrap_or("");
    let version = values.get("VERSION_ID").map(String::as_str).unwrap_or("");
    if !valid_release_id(id) {
        return Err("missing OS identity".to_string());
    }
    if !valid_release_version(version) {
        return Err("missing OS version".to_string());
    }
    let name = values
        .get("PRETTY_NAME")
        .cloned()
        .unwrap_or_else(|| id.to_string());
    Ok(domain::OsRelease {
        id: id.to_string(),
        version: version.to_string(),
        name,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::RefCell;
    use std::collections::VecDeque;

    struct Mock {
        calls: RefCell<Vec<(String, Vec<String>)>>,
        script: RefCell<VecDeque<Result<Vec<u8>, String>>>,
    }

    impl Mock {
        fn new(responses: Vec<Result<Vec<u8>, String>>) -> Self {
            Mock {
                calls: RefCell::new(Vec::new()),
                script: RefCell::new(responses.into()),
            }
        }
    }

    impl Executor for Mock {
        fn run(
            &self,
            _stdin: &[u8],
            cmd: &str,
            args: &[&str],
            _deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            self.calls.borrow_mut().push((
                cmd.to_string(),
                args.iter().map(|s| s.to_string()).collect(),
            ));
            self.script
                .borrow_mut()
                .pop_front()
                .unwrap_or(Err("no scripted response".to_string()))
        }
    }

    fn deadline() -> Instant {
        Instant::now() + Duration::from_secs(30)
    }

    fn test_config() -> Config {
        Config {
            muse_socket: String::new(),
            image: "img".to_string(),
            network: "sodanet".to_string(),
            subnet: "10.0.0.0/24".to_string(),
            bridge: "sodabr".to_string(),
        }
    }

    fn sample_profile() -> domain::Profile {
        domain::Profile {
            id: domain::ROCKY_HEADLESS.to_string(),
            distribution: "rocky".to_string(),
            version: "9.7".to_string(),
            interface: "headless".to_string(),
            architecture: "amd64".to_string(),
            image: format!("sha256:{}", "a".repeat(64)),
            revision: "b".repeat(40),
        }
    }

    fn image_inspect(profile: &domain::Profile, arch: &str, os: &str) -> Vec<u8> {
        format!(
            "{{\"Id\":{:?},\"Architecture\":{:?},\"Os\":{:?},\"Labels\":{{\"org.soda.profile\":{:?},\"org.soda.distribution\":\"rocky\",\"org.soda.distribution.version\":\"9.7\",\"org.soda.interface\":\"headless\",\"org.opencontainers.image.revision\":{:?}}}}}",
            profile.image, arch, os, profile.id, profile.revision
        )
        .into_bytes()
    }

    #[test]
    fn resolve_profile_accepts_native_image() {
        let p = sample_profile();
        let mock = Mock::new(vec![Ok(image_inspect(&p, "amd64", "linux"))]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(rt.resolve_profile(deadline()).unwrap(), p);
        let calls = mock.calls.borrow();
        assert_eq!(calls[0].0, "/usr/bin/podman");
        assert!(calls[0].1.contains(&"img".to_string()));
    }

    #[test]
    fn resolve_profile_refuses_foreign_or_invalid() {
        let p = sample_profile();
        for (arch, os) in [("arm64", "linux"), ("amd64", "windows")] {
            let mock = Mock::new(vec![Ok(image_inspect(&p, arch, os))]);
            let rt = Runtime {
                exec: &mock,
                config: test_config(),
            };
            assert_eq!(
                rt.resolve_profile(deadline()).unwrap_err(),
                "project image is not native Linux"
            );
        }
        let mock = Mock::new(vec![Ok(b"{}".to_vec())]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert!(rt.resolve_profile(deadline()).is_err());
        let mock = Mock::new(vec![Err("boom".to_string())]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(rt.resolve_profile(deadline()).unwrap_err(), "boom");
    }

    fn container_inspect(
        id: &str,
        owner: &str,
        profile: Option<&domain::Profile>,
        running: bool,
        ip: &str,
    ) -> Vec<u8> {
        let mut labels = format!("\"org.soda.project\":{id:?},\"org.soda.owner\":{owner:?}");
        if let Some(p) = profile {
            labels.push_str(&format!(
                ",\"org.soda.creation-profile\":{:?},\"org.soda.profile\":{:?}",
                p.encode(),
                p.id
            ));
        }
        let image = profile.map(|p| p.image.clone()).unwrap_or_default();
        format!(
            "[{{\"Image\":{image:?},\"Config\":{{\"Labels\":{{{labels}}}}},\"State\":{{\"Running\":{running}}},\"NetworkSettings\":{{\"Networks\":{{\"sodanet\":{{\"IPAddress\":{ip:?}}}}}}}}}]"
        )
        .into_bytes()
    }

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
    fn create_validates_before_exec() {
        let mock = Mock::new(vec![]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let bad = domain::Create {
            profile: None,
            id: "bogus".to_string(),
            owner: 0,
        };
        assert!(rt.create(&bad, deadline()).is_err());
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

    #[test]
    fn os_release_parsing_mirrors_python_cases() {
        // Selected fields; unknown keys ignored.
        let r = parse_os_release(
            b"ID=\"rocky\"\nVERSION_ID=\"9.7\"\nPRETTY_NAME=\"Rocky Linux 9.7\"\nOTHER=\"x\"\n",
        )
        .unwrap();
        assert_eq!(
            (r.id, r.version, r.name),
            (
                "rocky".to_string(),
                "9.7".to_string(),
                "Rocky Linux 9.7".to_string()
            )
        );
        // Shell syntax never executes.
        let r = parse_os_release(b"ID=rocky\nVERSION_ID=9.7\nPRETTY_NAME=\"$(touch /tmp/x)\"\n")
            .unwrap();
        assert!(r.name.contains("$(touch "));
        // PRETTY_NAME defaults to ID.
        let r = parse_os_release(b"ID=rocky\nVERSION_ID=9\n").unwrap();
        assert_eq!(r.name, "rocky");
        // Comments and blanks skipped.
        let r = parse_os_release(b"# c\n\n  \nID=rocky\nVERSION_ID=9\n").unwrap();
        assert_eq!(r.id, "rocky");
        // Malformed inputs fail.
        assert!(parse_os_release(b"ID=rocky\nID=fedora\nVERSION_ID=9").is_err());
        assert!(parse_os_release(b"ID=rocky").is_err());
        assert!(parse_os_release(b"ID=rocky\nVERSION_ID=9\nPRETTY_NAME=\"oops").is_err());
        assert!(parse_os_release(b"ID=rocky\nVERSION_ID=9\nPRETTY_NAME=\"a\x00b\"").is_err());
        assert!(
            parse_os_release(b"ID=rocky\nVERSION_ID=9\nPRETTY_NAME=\"two words\" extra").is_err()
        );
        assert!(parse_os_release(&[b'x'; 257]).is_err()); // no valid identity anyway
        let mut big = b"ID=rocky\nVERSION_ID=9\nPRETTY_NAME=\"".to_vec();
        big.extend(vec![b'x'; 257]);
        big.extend(b"\"");
        assert!(parse_os_release(&big).is_err());
        assert!(parse_os_release(b"ID=rocky\nVERSION_ID=9\nPRETTY_NAME=\"\xff\"").is_err());
        assert!(parse_os_release(b"ID=rocky\n").is_err()); // missing version
        assert!(parse_os_release(b"ID\nVERSION_ID=9\n").is_err()); // bare key with no '='
    }

    #[test]
    fn observe_os_reports_unavailable_without_starting() {
        let id = format!("p{}", "e".repeat(24));
        let p = sample_profile();
        // Stopped container: single inspect, no exec into the root.
        let mock = Mock::new(vec![Ok(container_inspect(&id, "42", Some(&p), false, ""))]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let obs = rt.observe_os(&id, deadline()).unwrap();
        assert!(obs.unavailable && obs.release.is_none());
        assert_eq!(mock.calls.borrow().len(), 1);
        // Running with a valid release.
        let mock = Mock::new(vec![
            Ok(container_inspect(&id, "42", Some(&p), true, "10.0.0.5")),
            Ok(Vec::new()),
            Ok(b"ID=rocky\nVERSION_ID=\"9.7\"\nPRETTY_NAME=\"Rocky Linux\"\n".to_vec()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let obs = rt.observe_os(&id, deadline()).unwrap();
        assert!(!obs.unavailable);
        assert_eq!(obs.release.unwrap().name, "Rocky Linux");
        // Unparsable content degrades to unavailable, not an error.
        let mock = Mock::new(vec![
            Ok(container_inspect(&id, "42", Some(&p), true, "10.0.0.5")),
            Ok(Vec::new()),
            Ok(b"garbage".to_vec()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert!(rt.observe_os(&id, deadline()).unwrap().unavailable);
    }

    #[test]
    fn connection_returns_ed25519_material() {
        let id = format!("p{}", "e".repeat(24));
        let p = sample_profile();
        let mut blob = vec![0, 0, 0, 11];
        blob.extend_from_slice(b"ssh-ed25519");
        blob.extend_from_slice(&[0, 0, 0, 32]);
        blob.extend_from_slice(&[0x77; 32]);
        let line = ssh::marshal_authorized_key(ssh::ALGO_ED25519, &blob);
        let mock = Mock::new(vec![
            Ok(container_inspect(&id, "42", Some(&p), true, "10.0.0.5")),
            Ok(line.clone().into_bytes()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let conn = rt.connection(&id, deadline()).unwrap();
        assert_eq!(conn.host_key, line);
        assert_eq!(conn.fingerprint, ssh::fingerprint_sha256(&blob));
        // Stopped containers stay stopped with empty key material.
        let mock = Mock::new(vec![Ok(container_inspect(&id, "42", Some(&p), false, ""))]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let conn = rt.connection(&id, deadline()).unwrap();
        assert!(conn.host_key.is_empty() && conn.fingerprint.is_empty());
    }
}
