//! Host-side answers for the kept installed shell skeletons, mirroring
//! the embedded Python of `tests/installed/host.sh`,
//! `tests/installed/operator.sh`, and
//! `tests/installed/forgejo-advertisement.sh`.
//!
//! The shells keep every gate, package, unit, ownership, and label
//! check; only the `python3` blocks move here, called by absolute
//! path. Verdict messages (`SystemExit` and assertion texts)
//! print verbatim; operational failures (I/O, JSON, subprocess)
//! collapse Python tracebacks to one `host probe failed:` line.
//! Success output is byte-identical.

use std::collections::{BTreeSet, HashMap};
use std::io::Read;
use std::net::{IpAddr, TcpStream};
use std::process::{Command, Stdio};
use std::time::Duration;

use soda_json::JsonValue;

use crate::sha256::{self, Sha256};

/// Host-probe failure: a verdict message or an operational failure.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ProbeFailure {
    /// Verdict message, printed verbatim like `SystemExit`.
    Exit(String),
    /// Operational failure; the fixed operation name only.
    Failed(&'static str),
}

impl ProbeFailure {
    fn exit(message: impl Into<String>) -> ProbeFailure {
        ProbeFailure::Exit(message.into())
    }

    fn failed(op: &'static str) -> ProbeFailure {
        ProbeFailure::Failed(op)
    }
}

impl std::fmt::Display for ProbeFailure {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ProbeFailure::Exit(message) => f.write_str(message),
            ProbeFailure::Failed(op) => write!(f, "host probe failed: {op}"),
        }
    }
}

impl std::error::Error for ProbeFailure {}

/// Python `json.dumps` rendering: insertion order, `(', ', ': ')`
/// separators, ASCII-only output with lowercase `\u` escapes.
pub fn dumps(value: &JsonValue) -> String {
    fn escape(text: &str, out: &mut String) {
        out.push('"');
        for ch in text.chars() {
            match ch {
                '"' => out.push_str("\\\""),
                '\\' => out.push_str("\\\\"),
                '\n' => out.push_str("\\n"),
                '\r' => out.push_str("\\r"),
                '\t' => out.push_str("\\t"),
                '\u{08}' => out.push_str("\\b"),
                '\u{0c}' => out.push_str("\\f"),
                c if (c as u32) < 0x20 => {
                    out.push_str(&format!("\\u{:04x}", c as u32));
                }
                c if (c as u32) < 0x7f => out.push(c),
                c if (c as u32) < 0x10000 => {
                    out.push_str(&format!("\\u{:04x}", c as u32));
                }
                c => {
                    let code = c as u32 - 0x10000;
                    out.push_str(&format!("\\u{:04x}\\u{:04x}", 0xd800 + (code >> 10), 0xdc00 + (code & 0x3ff)));
                }
            }
        }
        out.push('"');
    }

    fn render(value: &JsonValue, out: &mut String) {
        match value {
            JsonValue::Null => out.push_str("null"),
            JsonValue::Bool(true) => out.push_str("true"),
            JsonValue::Bool(false) => out.push_str("false"),
            JsonValue::Number(raw) => out.push_str(raw),
            JsonValue::Str(text) => escape(text, out),
            JsonValue::Array(items) => {
                out.push('[');
                for (index, item) in items.iter().enumerate() {
                    if index > 0 {
                        out.push_str(", ");
                    }
                    render(item, out);
                }
                out.push(']');
            }
            JsonValue::Object(entries) => {
                out.push('{');
                for (index, (key, item)) in entries.iter().enumerate() {
                    if index > 0 {
                        out.push_str(", ");
                    }
                    escape(key, out);
                    out.push_str(": ");
                    render(item, out);
                }
                out.push('}');
            }
        }
    }

    let mut out = String::new();
    render(value, &mut out);
    out
}

fn read_file(path: &str, op: &'static str) -> Result<Vec<u8>, ProbeFailure> {
    std::fs::read(path).map_err(|_| ProbeFailure::failed(op))
}

fn read_text(path: &str, op: &'static str) -> Result<String, ProbeFailure> {
    let bytes = read_file(path, op)?;
    String::from_utf8(bytes).map_err(|_| ProbeFailure::failed(op))
}

fn parse_json(text: &str, op: &'static str) -> Result<JsonValue, ProbeFailure> {
    JsonValue::parse(text).map_err(|_| ProbeFailure::failed(op))
}

fn get_str<'a>(value: &'a JsonValue, key: &str, op: &'static str) -> Result<&'a str, ProbeFailure> {
    value.get(key).and_then(|v| v.as_str()).ok_or_else(|| ProbeFailure::failed(op))
}

/// Run a probe subprocess to completion, returning stdout. Failures
/// carry the fixed operation name, never argv or output.
fn run_output(argv: &[&str], op: &'static str) -> Result<String, ProbeFailure> {
    let output = Command::new(argv[0])
        .args(&argv[1..])
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::null())
        .output()
        .map_err(|_| ProbeFailure::failed(op))?;
    if !output.status.success() {
        return Err(ProbeFailure::failed(op));
    }
    String::from_utf8(output.stdout).map_err(|_| ProbeFailure::failed(op))
}

fn sha256_file(path: &std::path::Path, op: &'static str) -> Result<String, ProbeFailure> {
    let mut file = std::fs::File::open(path).map_err(|_| ProbeFailure::failed(op))?;
    let mut digest = Sha256::new();
    let mut chunk = vec![0u8; 1024 * 1024];
    loop {
        let n = file.read(&mut chunk).map_err(|_| ProbeFailure::failed(op))?;
        if n == 0 {
            break;
        }
        digest.update(&chunk[..n]);
    }
    Ok(sha256::hex_lower(&digest.finalize()))
}

fn podman(args: &[&str], op: &'static str) -> Result<String, ProbeFailure> {
    let mut argv = vec!["podman", "--remote=false"];
    argv.extend(args.iter().copied());
    run_output(&argv, op)
}

/// Installed content inventory, images, and (when activated) the
/// extension package: the first `host.sh` Python block.
pub fn host_content(phase: &str) -> Result<String, ProbeFailure> {
    let payload = parse_json(&read_text("/usr/share/soda/release.json", "read release.json")?, "parse release.json")?;
    let content = parse_json(
        &read_text("/usr/share/soda/host-image/content.json", "read content inventory")?,
        "parse content inventory",
    )?;
    let fixed = BTreeSet::from([
        "dashboard:/usr/local/bin/soda-dashboard",
        "forgejo:/usr/local/bin/gitea",
        "extension:/usr/local/bin/gitea",
        "extension:/usr/share/soda/extension/extension.json",
        "extension:/usr/share/soda/extension/backend",
        "extension:/usr/share/soda/extension/run",
        "host:/usr/share/containers/systemd/forgejo.container",
        "host:/usr/share/containers/systemd/soda-dashboard.container",
        "host:/usr/lib/systemd/system/soda-extension-install.service",
    ]);
    let asset_prefix = "extension:/usr/share/soda/extension/assets/";
    let JsonValue::Object(entries) = &content else {
        return Err(ProbeFailure::failed("parse content inventory"));
    };
    let names: BTreeSet<&str> = entries.iter().map(|(k, _)| k.as_str()).collect();
    let assets: BTreeSet<&str> = names.iter().copied().filter(|name| name.starts_with(asset_prefix)).collect();
    let mut want = fixed.clone();
    want.extend(assets.iter().copied());
    if assets.is_empty() || names != want {
        return Err(ProbeFailure::exit("incomplete installed content inventory"));
    }
    for name in &assets {
        let relative = name.strip_prefix(asset_prefix).unwrap_or("");
        if relative.is_empty() || relative.split('/').any(|part| part == "..") || relative.starts_with('/') {
            return Err(ProbeFailure::exit("unsafe installed extension asset inventory"));
        }
    }
    if get_str(&payload, "Architecture", "parse release.json")? != std::env::consts::ARCH {
        return Err(ProbeFailure::exit("installed release architecture differs from native host"));
    }
    let packages = read_file("/usr/share/soda/host-image/packages.txt", "read package inventory")?;
    if sha256::hex_lower(&sha256::digest(&packages)) != get_str(&payload, "HostPackagesSHA256", "parse release.json")? {
        return Err(ProbeFailure::exit("installed RPM inventory differs from release metadata"));
    }
    let images = payload.get("Images").ok_or_else(|| ProbeFailure::failed("parse release.json"))?;
    let config = |name: &str| -> Result<&str, ProbeFailure> {
        images.get(name).and_then(|image| image.get("Config")).and_then(|v| v.as_str()).ok_or_else(|| ProbeFailure::failed("parse release.json"))
    };
    if config("forgejo")? == config("extension")? {
        return Err(ProbeFailure::exit("independent extension image required"));
    }
    let content_digest = |name: &str| -> Result<&str, ProbeFailure> {
        content.get(name).and_then(|v| v.as_str()).ok_or_else(|| ProbeFailure::failed("parse content inventory"))
    };
    if content_digest("forgejo:/usr/local/bin/gitea")? != content_digest("extension:/usr/local/bin/gitea")? {
        return Err(ProbeFailure::exit("extension installer CLI differs from patched Forgejo"));
    }
    for name in ["dashboard", "forgejo", "extension"] {
        let image = config(name)?;
        let observed = podman(&["image", "inspect", "--format", "{{.Id}}", image], "inspect installed image")?;
        if observed.trim() != image {
            return Err(ProbeFailure::exit(format!("{name} image identity differs from installed release")));
        }
    }
    for (name, expected) in entries {
        let expected = expected.as_str().ok_or_else(|| ProbeFailure::failed("parse content inventory"))?;
        let Some((component, path)) = name.split_once(':') else {
            return Err(ProbeFailure::failed("parse content inventory"));
        };
        let actual = if component == "host" {
            sha256_file(std::path::Path::new(path), "read installed content")?
        } else {
            let image = config(component)?;
            let output = podman(
                &[
                    "run",
                    "--rm",
                    "--network=none",
                    "--read-only",
                    "--cap-drop=all",
                    "--security-opt=no-new-privileges",
                    "--entrypoint=/usr/bin/sha256sum",
                    image,
                    path,
                ],
                "read installed image content",
            )?;
            let output = output.trim();
            match output.strip_suffix(&format!("  {path}")) {
                Some(hash) => hash.to_string(),
                None => String::new(),
            }
        };
        if actual != expected {
            return Err(ProbeFailure::exit(format!("installed content differs from release inventory: {name}")));
        }
    }
    let checked = if phase == "activated" {
        check_extension_package(&content)?;
        "installed package replacement"
    } else {
        "extension image package"
    };
    Ok(format!("Installed fork, {checked}, Soda service bytes and native architecture match the release inventory.\n"))
}

fn check_extension_package(content: &JsonValue) -> Result<(), ProbeFailure> {
    let package_prefix = "extension:/usr/share/soda/extension/";
    let JsonValue::Object(entries) = content else {
        return Err(ProbeFailure::failed("parse content inventory"));
    };
    let mut expected = HashMap::new();
    let mut ordered: Vec<String> = Vec::new();
    for (name, digest) in entries {
        if let Some(relative) = name.strip_prefix(package_prefix) {
            expected.insert(relative.to_string(), digest.as_str().ok_or_else(|| ProbeFailure::failed("parse content inventory"))?);
            ordered.push(relative.to_string());
        }
    }
    let root = std::path::Path::new("/var/lib/soda/forgejo/gitea/extensions/soda");
    let meta = std::fs::symlink_metadata(root).map_err(|_| ProbeFailure::exit("installed Soda extension package is not a regular directory"))?;
    if !meta.is_dir() || meta.is_symlink() {
        return Err(ProbeFailure::exit("installed Soda extension package is not a regular directory"));
    }
    let mut observed = BTreeSet::new();
    let mut stack = vec![root.to_path_buf()];
    while let Some(dir) = stack.pop() {
        let mut children: Vec<std::path::PathBuf> = Vec::new();
        for entry in std::fs::read_dir(&dir).map_err(|_| ProbeFailure::failed("walk extension package"))? {
            children.push(entry.map_err(|_| ProbeFailure::failed("walk extension package"))?.path());
        }
        children.sort();
        for path in children {
            let file_type = std::fs::symlink_metadata(&path).map_err(|_| ProbeFailure::failed("walk extension package"))?.file_type();
            if file_type.is_symlink() {
                return Err(ProbeFailure::exit("installed Soda extension package contains a symlink"));
            }
            if file_type.is_dir() {
                stack.push(path);
            } else if file_type.is_file() {
                let relative = path.strip_prefix(root).map_err(|_| ProbeFailure::failed("walk extension package"))?;
                let relative = relative.to_string_lossy().replace('\\', "/");
                if relative != ".disabled" {
                    observed.insert(relative);
                }
            }
        }
    }
    let want: BTreeSet<String> = expected.keys().cloned().collect();
    if observed != want {
        return Err(ProbeFailure::exit("installed Soda extension package differs from candidate inventory"));
    }
    for relative in &ordered {
        let actual = sha256_file(&root.join(relative), "read extension package")?;
        if actual != expected[relative.as_str()] {
            return Err(ProbeFailure::exit(format!("installed package replacement differs from candidate inventory: {relative}")));
        }
    }
    Ok(())
}

/// One observed listener.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord)]
struct Listener {
    address: String,
    port: u16,
}

/// Parse `ss -H -ltn` rows into listeners.
fn parse_listeners(output: &str) -> Result<BTreeSet<Listener>, ProbeFailure> {
    let mut listeners = BTreeSet::new();
    for row in output.lines() {
        let fields: Vec<&str> = row.split_whitespace().collect();
        if fields.len() < 5 {
            return Err(ProbeFailure::exit("malformed listener observation"));
        }
        let Some((address, port)) = fields[3].rsplit_once(':') else {
            return Err(ProbeFailure::failed("parse listeners"));
        };
        let port = port.trim().strip_prefix('+').unwrap_or(port.trim());
        if port.is_empty() || !port.bytes().all(|b| b.is_ascii_digit()) {
            return Err(ProbeFailure::failed("parse listeners"));
        }
        let port: u16 = port.parse().map_err(|_| ProbeFailure::failed("parse listeners"))?;
        listeners.insert(Listener {
            address: address.trim_matches(|c| c == '[' || c == ']').to_string(),
            port,
        });
    }
    Ok(listeners)
}

fn bound(listeners: &BTreeSet<Listener>, address: &str, port: u16) -> Result<(), ProbeFailure> {
    if !listeners.contains(&Listener { address: address.to_string(), port }) {
        return Err(ProbeFailure::exit("required service binding missing"));
    }
    for listener in listeners {
        if listener.port != port {
            continue;
        }
        let allowed = listener.address == address || (address == "127.0.0.1" && listener.address == "::1");
        if !allowed {
            return Err(ProbeFailure::exit("unexpected additional service binding"));
        }
    }
    Ok(())
}

fn published(listeners: &BTreeSet<Listener>, address: &str, host_port: u16, container_port: u16) -> Result<(), ProbeFailure> {
    let mapping = podman(&["port", "soda-forgejo", &format!("{container_port}/tcp")], "read published ports")?;
    let expected = if address.contains(':') {
        format!("[{address}]:{host_port}")
    } else {
        format!("{address}:{host_port}")
    };
    if mapping.trim() != expected {
        return Err(ProbeFailure::exit("unexpected published Git/HTTP mapping"));
    }
    // Rootful bridge publication may use DNAT, not a host listening process.
    for listener in listeners {
        if listener.port == host_port && listener.address != address {
            return Err(ProbeFailure::exit("unexpected host listener on published port"));
        }
    }
    let ip: IpAddr = address.parse().map_err(|_| ProbeFailure::failed("connect published endpoint"))?;
    TcpStream::connect_timeout(&std::net::SocketAddr::new(ip, host_port), Duration::from_secs(5))
        .map_err(|_| ProbeFailure::failed("connect published endpoint"))?;
    Ok(())
}

/// RFC 1918 plus shared `100.64/10` for v4, ULA for v6: the bind
/// addresses operators actually configure. Python's `is_private`
/// agrees on all of these; exotic ranges are out of scope.
fn is_private_bind(ip: IpAddr) -> bool {
    match ip {
        IpAddr::V4(v4) => {
            let octets = v4.octets();
            octets[0] == 10
                || (octets[0] == 100 && octets[1] & 0xc0 == 64)
                || (octets[0] == 172 && octets[1] & 0xf0 == 16)
                || (octets[0] == 192 && octets[1] == 168)
        }
        IpAddr::V6(v6) => v6.octets()[0] & 0xfe == 0xfc,
    }
}

/// Split an `https://host[:port][/...]` origin into hostname and port.
/// Userinfo is stripped; IPv6 stays bracketed for binding comparison.
fn split_origin(origin: &str) -> Result<(String, u16), ProbeFailure> {
    let err = || ProbeFailure::exit("invalid configured HTTPS origin");
    let Some(rest) = origin.strip_prefix("https://") else {
        return Err(err());
    };
    let authority = rest.split(['/', '?', '#']).next().unwrap_or("");
    let host_port = authority.rsplit('@').next().unwrap_or("");
    if host_port.is_empty() {
        return Err(err());
    }
    let (host, port) = if let Some(bracketed) = host_port.strip_prefix('[') {
        let Some((host, after)) = bracketed.split_once(']') else {
            return Err(err());
        };
        if host.is_empty() {
            return Err(err());
        }
        let port = match after.strip_prefix(':') {
            Some(digits) => digits.parse::<u16>().map_err(|_| ProbeFailure::failed("parse origin"))?,
            None if after.is_empty() => 443,
            None => return Err(err()),
        };
        (host.to_string(), port)
    } else if let Some((host, digits)) = host_port.rsplit_once(':') {
        if host.is_empty() || digits.is_empty() || !digits.bytes().all(|b| b.is_ascii_digit()) {
            return Err(err());
        }
        (host.to_lowercase(), digits.parse::<u16>().map_err(|_| ProbeFailure::failed("parse origin"))?)
    } else {
        (host_port.to_lowercase(), 443)
    };
    Ok((host, port))
}

/// Python truthiness for JSON values, for the `(d.get("Self") or {})`
/// shapes.
fn is_truthy(value: &JsonValue) -> bool {
    match value {
        JsonValue::Null => false,
        JsonValue::Bool(b) => *b,
        JsonValue::Number(raw) => raw.parse::<f64>().map(|n| n != 0.0).unwrap_or(true),
        JsonValue::Str(s) => !s.is_empty(),
        JsonValue::Array(items) => !items.is_empty(),
        JsonValue::Object(entries) => !entries.is_empty(),
    }
}

/// Parse `KEY=value` environment files. Empty lines are skipped; a
/// line without `=` fails, like the owner's unpacking split.
fn parse_env_file(text: &str) -> Result<HashMap<String, String>, ProbeFailure> {
    let mut env = HashMap::new();
    for line in text.lines() {
        if line.is_empty() {
            continue;
        }
        let Some((key, value)) = line.split_once('=') else {
            return Err(ProbeFailure::failed("parse proxy.env"));
        };
        env.insert(key.to_string(), value.to_string());
    }
    Ok(env)
}

/// Listener, publication, credential-mode, and origin boundaries: the
/// second `host.sh` Python block.
pub fn host_listeners(phase: &str) -> Result<String, ProbeFailure> {
    let rows = run_output(&["ss", "-H", "-ltn"], "read listeners")?;
    let listeners = parse_listeners(&rows)?;
    bound(&listeners, "127.0.0.1", 9090)?;
    published(&listeners, "127.0.0.1", 3000, 3000)?;
    if phase == "activated" {
        let cfg = parse_json(&read_text("/etc/soda/dashboard.json", "read dashboard.json")?, "parse dashboard.json")?;
        // Retired bootstrap files are not dashboard credentials and need not exist.
        let grant = get_str(&cfg, "grant_key_file", "parse dashboard.json")?;
        let path = std::path::Path::new(grant);
        let info = std::fs::symlink_metadata(path).map_err(|_| ProbeFailure::exit("configured dashboard credential permissions invalid"))?;
        use std::os::unix::fs::MetadataExt;
        if !grant.starts_with('/') || !info.is_file() || info.uid() != 0 || info.gid() != 2000 || info.mode() & 0o7777 != 0o640 {
            return Err(ProbeFailure::exit("configured dashboard credential permissions invalid"));
        }
        for name in ["cert.pem", "key.pem"] {
            let info =
                std::fs::symlink_metadata(std::path::Path::new("/etc/soda/tls").join(name)).map_err(|_| ProbeFailure::exit("TLS material permissions invalid"))?;
            if !info.is_file() || info.uid() != 0 || info.mode() & 0o7777 != 0o600 {
                return Err(ProbeFailure::exit("TLS material permissions invalid"));
            }
        }
        // These are the variables actually consumed by the packaged Caddy config,
        // not ports reconstructed from a client's unrelated browser tunnel.
        let env = parse_env_file(&read_text("/etc/soda/proxy.env", "read proxy.env")?)?;
        let bind_raw = env.get("SODA_BIND").ok_or_else(|| ProbeFailure::failed("parse proxy.env"))?;
        let bind: IpAddr = bind_raw.parse().map_err(|_| ProbeFailure::exit("private explicit proxy bind required"))?;
        if !is_private_bind(bind) || bind.is_unspecified() {
            return Err(ProbeFailure::exit("private explicit proxy bind required"));
        }
        let address = bind.to_string();
        if cfg.get("public_url").is_some() || env.contains_key("SODA_ORIGIN") {
            return Err(ProbeFailure::exit("legacy separate-origin configuration requires rehearsed maintenance"));
        }
        let forgejo_origin = env.get("FORGEJO_ORIGIN").ok_or_else(|| ProbeFailure::failed("parse proxy.env"))?;
        if forgejo_origin.trim_end_matches('/') != get_str(&cfg, "forgejo_url", "parse dashboard.json")?.trim_end_matches('/') {
            return Err(ProbeFailure::exit("proxy and API browser origin mismatch"));
        }
        let (host, port) = split_origin(forgejo_origin)?;
        bound(&listeners, &host, port)?;
        let listen = get_str(&cfg, "listen", "parse dashboard.json")?;
        let Some((host, port)) = listen.rsplit_once(':') else {
            return Err(ProbeFailure::failed("parse dashboard.json"));
        };
        let port: u16 = port.trim().parse().map_err(|_| ProbeFailure::failed("parse dashboard.json"))?;
        bound(&listeners, host.trim_matches(|c| c == '[' || c == ']'), port)?;
        published(&listeners, &address, 2222, 22)?;
    } else {
        published(&listeners, "127.0.0.1", 2222, 22)?;
    }
    Ok("Configured native listener and credential-mode boundaries observed; not a login or client-route proof.\n".to_string())
}

/// `rpm-ostree status --json` deployments summary: the `host.sh`
/// inline `python3 -c` over stdin.
pub fn host_deployments(stdin: &str) -> Result<String, ProbeFailure> {
    let doc = parse_json(stdin, "parse deployments")?;
    let deployments = doc.get("deployments").and_then(|v| match v {
        JsonValue::Array(items) => Some(items),
        _ => None,
    });
    let Some(deployments) = deployments else {
        return Err(ProbeFailure::failed("parse deployments"));
    };
    let mut out = Vec::new();
    for deployment in deployments {
        let JsonValue::Object(_) = deployment else {
            return Err(ProbeFailure::failed("parse deployments"));
        };
        let mut summary = Vec::new();
        for key in ["booted", "version", "checksum", "requested-packages"] {
            summary.push((key.to_string(), deployment.get(key).cloned().unwrap_or(JsonValue::Null)));
        }
        out.push(JsonValue::Object(summary));
    }
    Ok(dumps(&JsonValue::Array(out)) + "\n")
}

/// Retained Tailnet state summary: the `operator.sh` inline
/// `python3 -c` over stdin.
pub fn operator_tailscale(stdin: &str) -> Result<String, ProbeFailure> {
    let doc = parse_json(stdin, "parse tailscale status")?;
    let Some(JsonValue::Str(state)) = doc.get("BackendState") else {
        return Err(ProbeFailure::failed("validate tailscale status"));
    };
    // `(d.get("Self") or {})`: falsy shapes yield no expiry, truthy
    // non-objects fail like the owner's attribute error.
    let expired = match doc.get("Self") {
        None => JsonValue::Null,
        Some(value) if !is_truthy(value) => JsonValue::Null,
        Some(JsonValue::Object(_)) => doc.get("Self").and_then(|v| v.get("Expired")).cloned().unwrap_or(JsonValue::Null),
        Some(_) => return Err(ProbeFailure::failed("validate tailscale status")),
    };
    Ok(dumps(&JsonValue::Object(vec![
        ("BackendState".to_string(), JsonValue::Str(state.clone())),
        ("Expired".to_string(), expired),
    ])) + "\n")
}

/// Core browser origins: the `forgejo-advertisement.sh` `origins()`
/// helper.
pub fn forgejo_origins() -> Result<String, ProbeFailure> {
    let cfg = parse_json(&read_text("/etc/soda/dashboard.json", "read dashboard.json")?, "parse dashboard.json")?;
    Ok(dumps(&JsonValue::Object(vec![
        ("forgejo_internal_url".to_string(), JsonValue::Str(get_str(&cfg, "forgejo_internal_url", "parse dashboard.json")?.to_string())),
        ("forgejo_url".to_string(), JsonValue::Str(get_str(&cfg, "forgejo_url", "parse dashboard.json")?.to_string())),
    ])) + "\n")
}

/// Approved running Tailnet gate: the `forgejo-advertisement.sh`
/// inline `python3 -c` over stdin. Silent on success.
pub fn forgejo_tailnet(stdin: &str) -> Result<String, ProbeFailure> {
    let doc = parse_json(stdin, "parse tailscale status")?;
    let running = doc.get("BackendState").and_then(|v| v.as_str()) == Some("Running");
    let expired = match doc.get("Self") {
        None => false,
        Some(value) if !is_truthy(value) => false,
        Some(JsonValue::Object(_)) => {
            doc.get("Self").and_then(|v| v.get("Expired")).is_some_and(is_truthy)
        }
        Some(_) => return Err(ProbeFailure::failed("validate tailscale status")),
    };
    if !running || expired {
        return Err(ProbeFailure::exit("approved running Tailnet required; no enrollment is performed"));
    }
    Ok(String::new())
}

/// Listener-checked Git SSH advertisement: the final
/// `forgejo-advertisement.sh` Python block.
pub fn forgejo_advertisement() -> Result<String, ProbeFailure> {
    let mut values = HashMap::new();
    for line in read_text("/etc/soda/forgejo.env", "read forgejo.env")?.lines() {
        if line.contains('=') && !line.starts_with('#') {
            let (key, value) = line.split_once('=').unwrap();
            values.insert(key.to_string(), value.to_string());
        }
    }
    let host = values.get("FORGEJO__server__SSH_DOMAIN").ok_or_else(|| ProbeFailure::failed("parse forgejo.env"))?;
    let port = values.get("FORGEJO__server__SSH_PORT").ok_or_else(|| ProbeFailure::failed("parse forgejo.env"))?;
    let status = parse_json(
        &run_output(&["/usr/bin/tailscale", "status", "--json"], "read tailscale status")?,
        "parse tailscale status",
    )?;
    let on_tailnet = match status.get("TailscaleIPs") {
        Some(JsonValue::Array(ips)) => ips.iter().any(|ip| ip.as_str() == Some(host.as_str())),
        _ => false,
    };
    if !on_tailnet {
        return Err(ProbeFailure::exit("advertisement does not match a current Tailnet address"));
    }
    let listeners = podman(&["port", "soda-forgejo", "22/tcp"], "read published ports")?;
    if !listeners.lines().any(|line| line == format!("{host}:{port}")) {
        return Err(ProbeFailure::exit("advertised endpoint is not a real selected private listener"));
    }
    Ok(format!(
        "Listener-checked Git SSH advertisement: {host} {port}\nCore browser origins unchanged. Probe the pinned endpoint from the intended client; this is not routed Git authentication proof.\n"
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dumps_keeps_order_and_separators() {
        let value = JsonValue::Object(vec![
            ("b".to_string(), JsonValue::Number("1".to_string())),
            ("a".to_string(), JsonValue::Array(vec![JsonValue::Bool(true), JsonValue::Null])),
        ]);
        assert_eq!(dumps(&value), r#"{"b": 1, "a": [true, null]}"#);
        let value = JsonValue::Object(vec![("e".to_string(), JsonValue::Str("é\t\"q\"".to_string()))]);
        assert_eq!(dumps(&value), "{\"e\": \"\\u00e9\\t\\\"q\\\"\"}");
    }

    #[test]
    fn origins_split_hosts_and_ports() {
        assert_eq!(split_origin("https://example.test").unwrap(), ("example.test".to_string(), 443));
        assert_eq!(split_origin("https://Example.Test:8443/x").unwrap(), ("example.test".to_string(), 8443));
        assert_eq!(split_origin("https://user@[::1]:2222").unwrap(), ("::1".to_string(), 2222));
        assert!(split_origin("http://example.test").is_err());
        assert!(split_origin("https://").is_err());
        assert!(split_origin("https://host:notaport").is_err());
    }

    #[test]
    fn private_binds_match_plain_lan_shapes() {
        for ip in ["10.1.2.3", "100.100.0.1", "172.20.0.1", "192.168.0.1", "fc00::1"] {
            assert!(is_private_bind(ip.parse().unwrap()), "{ip}");
        }
        for ip in ["8.8.8.8", "0.0.0.0", "::", "127.0.0.1", "fe80::1"] {
            assert!(!is_private_bind(ip.parse().unwrap()), "{ip}");
        }
    }

    #[test]
    fn listeners_parse_and_bind() {
        let rows = "LISTEN 0 128 127.0.0.1:9090 0.0.0.0:*\nLISTEN 0 128 [::1]:9090 [::]:*\n";
        let listeners = parse_listeners(rows).unwrap();
        assert_eq!(listeners.len(), 2);
        bound(&listeners, "127.0.0.1", 9090).unwrap();
        assert_eq!(bound(&listeners, "127.0.0.1", 3000).unwrap_err(), ProbeFailure::exit("required service binding missing"));
        let rows = "LISTEN 0 128 0.0.0.0:9090 0.0.0.0:*\n";
        let listeners = parse_listeners(rows).unwrap();
        assert_eq!(bound(&listeners, "127.0.0.1", 9090).unwrap_err(), ProbeFailure::exit("required service binding missing"));
        assert!(parse_listeners("short row\n").is_err());
    }

    #[test]
    fn env_files_skip_empty_and_reject_bare_words() {
        let env = parse_env_file("A=1\n\nB=x=y\n").unwrap();
        assert_eq!(env.get("A").unwrap(), "1");
        assert_eq!(env.get("B").unwrap(), "x=y");
        assert_eq!(parse_env_file("BARE\n").unwrap_err(), ProbeFailure::failed("parse proxy.env"));
    }

    #[test]
    fn tailscale_summaries_shape() {
        let out = operator_tailscale(r#"{"BackendState": "Running", "Self": {"Expired": false}}"#).unwrap();
        assert_eq!(out, "{\"BackendState\": \"Running\", \"Expired\": false}\n");
        let out = operator_tailscale(r#"{"BackendState": "Stopped"}"#).unwrap();
        assert_eq!(out, "{\"BackendState\": \"Stopped\", \"Expired\": null}\n");
        assert!(operator_tailscale(r#"{"BackendState": 7}"#).is_err());
        assert_eq!(forgejo_tailnet(r#"{"BackendState": "Running", "Self": {"Expired": false}}"#).unwrap(), "");
        assert_eq!(
            forgejo_tailnet(r#"{"BackendState": "NoState"}"#).unwrap_err(),
            ProbeFailure::exit("approved running Tailnet required; no enrollment is performed")
        );
    }

    #[test]
    fn deployments_summarize_in_order() {
        let out = host_deployments(r#"{"deployments": [{"booted": true, "version": "41", "checksum": "abc", "requested-packages": ["x"], "extra": 1}]}"#)
            .unwrap();
        assert_eq!(out, "[{\"booted\": true, \"version\": \"41\", \"checksum\": \"abc\", \"requested-packages\": [\"x\"]}]\n");
        assert!(host_deployments(r#"{"deployments": [7]}"#).is_err());
    }
}
