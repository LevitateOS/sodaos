//! Operator-only activation after native Forgejo setup, with explicit private TLS.
use std::ffi::CString;
use std::fs::{self, OpenOptions};
use std::io::{self, Write};
use std::net::IpAddr;
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};

use soda_json::JsonValue;

#[link(name = "c")]
extern "C" {
    fn geteuid() -> u32;
    fn chown(path: *const std::os::raw::c_char, owner: u32, group: u32) -> std::os::raw::c_int;
    fn getpwnam(name: *const std::os::raw::c_char) -> *const Passwd;
}

#[repr(C)]
struct Passwd {
    pw_name: *const std::os::raw::c_char,
    pw_passwd: *const std::os::raw::c_char,
    pw_uid: u32,
    pw_gid: u32,
}

const USAGE: &str = "usage: soda-activate [-h] --bind-ip BIND_IP [--certificate CERTIFICATE] [--private-key PRIVATE_KEY] [--local-tls]";

fn help_text(prog: &str) -> String {
    format!(
        "usage: {prog} [-h] --bind-ip BIND_IP [--certificate CERTIFICATE] [--private-key PRIVATE_KEY] [--local-tls]\n\noptions:\n  -h, --help            show this help message and exit\n  --bind-ip BIND_IP     explicit private appliance address\n  --certificate CERTIFICATE\n                        PEM certificate covering the Forgejo/Sodaspaces browser origin\n  --private-key PRIVATE_KEY\n                        PEM private key; never committed\n  --local-tls           use Caddy local certificates for the selected private IP; client trust is explicit\n"
    )
}

fn main() {
    let argv: Vec<String> = std::env::args().collect();
    let prog = argv
        .first()
        .map(|a| a.rsplit('/').next().unwrap_or("soda-activate"))
        .unwrap_or("soda-activate");
    let prog = if prog.is_empty() {
        "soda-activate"
    } else {
        prog
    };
    let args = match parse_args(prog, &argv[1..]) {
        Ok(None) => {
            print!("{}", help_text(prog));
            let _ = io::stdout().flush();
            std::process::exit(0);
        }
        Ok(Some(args)) => args,
        Err(msg) => {
            eprintln!("{USAGE}\n{prog}: error: {msg}");
            std::process::exit(2);
        }
    };
    let paths = Paths::production();
    let mut sys = RealSys;
    let mut stdout = io::stdout();
    match activate(&args, &paths, &mut sys, &mut stdout) {
        Ok(()) => {}
        Err(ActivateError::Usage(msg)) => {
            eprintln!("{USAGE}\n{prog}: error: {msg}");
            std::process::exit(2);
        }
        Err(ActivateError::Runtime(msg)) => {
            eprintln!("{prog}: {msg}");
            std::process::exit(1);
        }
    }
}

#[derive(Debug, PartialEq)]
struct CliArgs {
    bind_ip: String,
    certificate: Option<String>,
    private_key: Option<String>,
    local_tls: bool,
}

/// Parse argv argparse-style: exact long options, unambiguous prefixes,
/// `--opt=value`, `-h/--help`. Errors carry argparse's messages.
fn parse_args(prog: &str, args: &[String]) -> Result<Option<CliArgs>, String> {
    let mut bind_ip: Option<String> = None;
    let mut certificate: Option<String> = None;
    let mut private_key: Option<String> = None;
    let mut local_tls = false;
    let mut i = 0;
    let mut positionals: Vec<String> = Vec::new();
    let long_opts = ["bind-ip", "certificate", "private-key", "local-tls", "help"];
    while i < args.len() {
        let arg = &args[i];
        if arg == "--" {
            positionals.extend(args[i + 1..].iter().cloned());
            break;
        }
        if arg == "-" || !arg.starts_with('-') {
            positionals.push(arg.clone());
            i += 1;
            continue;
        }
        if arg == "-h" {
            return Ok(None);
        }
        if !arg.starts_with("--") {
            positionals.push(arg.clone());
            i += 1;
            continue;
        }
        let body = &arg[2..];
        let (name, inline) = match body.split_once('=') {
            Some((n, v)) => (n, Some(v)),
            None => (body, None),
        };
        if name.is_empty() {
            positionals.push(arg.clone());
            i += 1;
            continue;
        }
        let mut matches: Vec<&&str> = long_opts.iter().filter(|o| o.starts_with(name)).collect();
        if long_opts.contains(&name) {
            matches = long_opts.iter().filter(|o| **o == name).collect();
        }
        if matches.is_empty() {
            // argparse reports the whole token for unrecognized long options.
            positionals.push(arg.clone());
            i += 1;
            continue;
        }
        if matches.len() > 1 {
            let options: Vec<String> = matches.iter().map(|o| format!("--{o}")).collect();
            return Err(format!(
                "argument --{name}: ambiguous option: {arg} could match {}",
                options.join(", ")
            ));
        }
        let opt = matches[0].to_string();
        if opt == "help" {
            if let Some(v) = inline {
                return Err(format!("argument --help: ignored explicit argument '{v}'"));
            }
            let _ = prog;
            return Ok(None);
        }
        if opt == "local-tls" {
            if let Some(v) = inline {
                return Err(format!(
                    "argument --local-tls: ignored explicit argument '{v}'"
                ));
            }
            local_tls = true;
            i += 1;
            continue;
        }
        let value = match inline {
            Some(v) => v.to_string(),
            None => {
                if i + 1 >= args.len() {
                    return Err(format!("argument --{opt}: expected one argument"));
                }
                i += 1;
                args[i].clone()
            }
        };
        match opt.as_str() {
            "bind-ip" => bind_ip = Some(value),
            "certificate" => certificate = Some(value),
            "private-key" => private_key = Some(value),
            _ => {}
        }
        i += 1;
    }
    if !positionals.is_empty() {
        return Err(format!("unrecognized arguments: {}", positionals.join(" ")));
    }
    match bind_ip {
        Some(bind_ip) => Ok(Some(CliArgs {
            bind_ip,
            certificate,
            private_key,
            local_tls,
        })),
        None => Err("the following arguments are required: --bind-ip".to_string()),
    }
}

struct Paths {
    root: PathBuf,
    var_lib: PathBuf,
    containers_systemd: PathBuf,
}

impl Paths {
    fn production() -> Paths {
        Paths {
            root: PathBuf::from("/etc/soda"),
            var_lib: PathBuf::from("/var/lib/soda"),
            containers_systemd: PathBuf::from("/etc/containers/systemd"),
        }
    }
}

#[derive(Debug, PartialEq)]
enum ActivateError {
    /// argparse-style validation failure: usage + error, exit 2.
    Usage(String),
    /// Runtime failure (IO, missing user, failing unit): message, exit 1.
    Runtime(String),
}

fn usage(msg: impl Into<String>) -> ActivateError {
    ActivateError::Usage(msg.into())
}

fn runtime(msg: impl Into<String>) -> ActivateError {
    ActivateError::Runtime(msg.into())
}

/// Privileged operations, injectable for tests. Plain file reads/writes go
/// through std directly; only identity, ownership, processes, and time vary.
trait Sys {
    fn euid(&self) -> u32;
    fn lookup_user(&self, name: &str) -> Result<(u32, u32), String>;
    fn chown(&mut self, path: &Path, uid: u32, gid: u32) -> io::Result<()>;
    fn run(&mut self, argv: &[&str]) -> io::Result<i32>;
    fn now(&mut self) -> f64;
    fn sleep(&mut self, secs: u64);
}

struct RealSys;

impl Sys for RealSys {
    fn euid(&self) -> u32 {
        unsafe { geteuid() }
    }

    fn lookup_user(&self, name: &str) -> Result<(u32, u32), String> {
        let cname =
            CString::new(name).map_err(|_| format!("getpwnam(): name not found: {name:?}"))?;
        let entry = unsafe { getpwnam(cname.as_ptr()) };
        if entry.is_null() {
            return Err(format!("getpwnam(): name not found: {name:?}"));
        }
        let (uid, gid) = unsafe { ((*entry).pw_uid, (*entry).pw_gid) };
        Ok((uid, gid))
    }

    fn chown(&mut self, path: &Path, uid: u32, gid: u32) -> io::Result<()> {
        let bytes = path.as_os_str().as_bytes();
        let mut nul = Vec::with_capacity(bytes.len() + 1);
        nul.extend_from_slice(bytes);
        nul.push(0);
        let ret = unsafe { chown(nul.as_ptr() as *const std::os::raw::c_char, uid, gid) };
        if ret == 0 {
            Ok(())
        } else {
            Err(io::Error::last_os_error())
        }
    }

    fn run(&mut self, argv: &[&str]) -> io::Result<i32> {
        let status = std::process::Command::new(argv[0])
            .args(&argv[1..])
            .status()?;
        Ok(status.code().unwrap_or(1))
    }

    fn now(&mut self) -> f64 {
        use std::time::{SystemTime, UNIX_EPOCH};
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs_f64())
            .unwrap_or(0.0)
    }

    fn sleep(&mut self, secs: u64) {
        std::thread::sleep(std::time::Duration::from_secs(secs));
    }
}

// --- IP classification, mirroring CPython's ipaddress module ---
//
// Accept rule (from soda-activate): reject when unspecified, multicast,
// global, or loopback. Global/private tables below are CPython 3.12's
// iana-special-registry tables (_private_networks + exceptions), verified
// against the interpreter's own verdicts.

fn v4_in_net(ip: u32, base: u32, prefix: u32) -> bool {
    if prefix == 0 {
        return true;
    }
    (ip >> (32 - prefix)) == (base >> (32 - prefix))
}

fn v4_is_private(ip: u32) -> bool {
    const NETS: &[(u32, u32)] = &[
        (0x00000000, 8),
        (0x0A000000, 8),
        (0x7F000000, 8),
        (0xA9FE0000, 16),
        (0xAC100000, 12),
        (0xC0000000, 24),
        (0xC00000AA, 31),
        (0xC0000200, 24),
        (0xC0A80000, 16),
        (0xC6120000, 15),
        (0xC6336400, 24),
        (0xCB007100, 24),
        (0xF0000000, 4),
        (0xFFFFFFFF, 32),
    ];
    if ip == 0xC0000009 || ip == 0xC000000A {
        return false;
    }
    NETS.iter()
        .any(|(base, prefix)| v4_in_net(ip, *base, *prefix))
}

fn v4_is_global(ip: u32) -> bool {
    !v4_in_net(ip, 0x64400000, 10) && !v4_is_private(ip)
}

fn v6_in_net(ip: u128, base: u128, prefix: u32) -> bool {
    if prefix == 0 {
        return true;
    }
    (ip >> (128 - prefix)) == (base >> (128 - prefix))
}

fn v6_is_private(ip: u128) -> bool {
    const NETS: &[(u128, u32)] = &[
        (1, 128),
        (0, 128),
        (0x00000000000000000000FFFF00000000, 96),
        (0x0064FF9B000100000000000000000000, 48),
        (0x01000000000000000000000000000000, 64),
        (0x20010000000000000000000000000000, 23),
        (0x20010DB8000000000000000000000000, 32),
        (0x20020000000000000000000000000000, 16),
        (0x3FFF0000000000000000000000000000, 20),
        (0xFC000000000000000000000000000000, 7),
        (0xFE800000000000000000000000000000, 10),
    ];
    const EXCEPTIONS: &[(u128, u32)] = &[
        (0x20010001000000000000000000000001, 128),
        (0x20010001000000000000000000000002, 128),
        (0x20010003000000000000000000000000, 32),
        (0x20010004011200000000000000000000, 48),
        (0x20010020000000000000000000000000, 28),
        (0x20010030000000000000000000000000, 28),
    ];
    NETS.iter()
        .any(|(base, prefix)| v6_in_net(ip, *base, *prefix))
        && !EXCEPTIONS
            .iter()
            .any(|(base, prefix)| v6_in_net(ip, *base, *prefix))
}

fn activate_rejects_ip(addr: &IpAddr) -> bool {
    match addr {
        IpAddr::V4(v4) => {
            let ip: u32 = (*v4).into();
            ip == 0
                || v4_in_net(ip, 0xE0000000, 4)
                || v4_is_global(ip)
                || v4_in_net(ip, 0x7F000000, 8)
        }
        IpAddr::V6(v6) => {
            // Mapped addresses delegate to the inner IPv4 verdict: the outer
            // address is never unspecified, multicast, or loopback, so only
            // the inner global bit can reject (CPython ipv4_mapped rule).
            if let Some(inner) = v6.to_ipv4_mapped() {
                return v4_is_global(inner.into());
            }
            let ip: u128 = (*v6).into();
            ip == 0
                || v6_in_net(ip, 0xFF000000000000000000000000000000, 8)
                || !v6_is_private(ip)
                || ip == 1
        }
    }
}

// --- URL origin checks, mirroring the urlsplit subset soda-activate uses ---

struct OriginParts {
    hostname: String,
    port: Option<u16>,
    port_present: bool,
}

fn split_origin(value: &str) -> Result<(String, String, String, String, String), ActivateError> {
    // Split into (scheme, authority, path, query, fragment). Bracket errors
    // raise in urlsplit, so they are runtime failures here too.
    let (scheme, rest) = value
        .split_once("://")
        .ok_or_else(|| usage("invalid HTTPS browser origin"))?;
    let frag_at = rest.find('#');
    let query_at = rest.find('?');
    let (before_frag, fragment) = match frag_at {
        Some(at) => (&rest[..at], &rest[at + 1..]),
        None => (rest, ""),
    };
    let (authority_path, query) = match query_at {
        Some(at) if frag_at.is_none_or(|fat| at < fat) => {
            (&before_frag[..at], &before_frag[at + 1..])
        }
        _ => (before_frag, ""),
    };
    let split_at = authority_path.find('/').unwrap_or(authority_path.len());
    let (authority, path) = authority_path.split_at(split_at);
    if authority.contains('[') || authority.contains(']') {
        if !authority.starts_with('[') {
            return Err(runtime(format!("invalid IPv6 URL in {value:?}")));
        }
        let end = authority
            .find(']')
            .ok_or_else(|| runtime(format!("invalid IPv6 URL in {value:?}")))?;
        let after = &authority[end + 1..];
        if !after.is_empty() && !after.starts_with(':') {
            return Err(runtime(format!("invalid IPv6 URL in {value:?}")));
        }
    }
    Ok((
        scheme.to_string(),
        authority.to_string(),
        path.to_string(),
        query.to_string(),
        fragment.to_string(),
    ))
}

fn check_browser_origin(value: &str) -> Result<(), ActivateError> {
    let (scheme, authority, path, query, fragment) = split_origin(value)?;
    if !scheme.eq_ignore_ascii_case("https") {
        return Err(usage("invalid HTTPS browser origin"));
    }
    let hostname = authority.rsplit('@').next().unwrap_or("");
    let hostname = hostname.strip_suffix(':').unwrap_or(hostname);
    let hostname = if hostname.starts_with('[') {
        hostname
            .find(']')
            .map(|end| &hostname[1..end])
            .unwrap_or("")
    } else {
        hostname.split(':').next().unwrap_or("")
    };
    if hostname.is_empty() {
        return Err(usage("invalid HTTPS browser origin"));
    }
    if authority.contains('@') {
        let userinfo = authority.rsplit('@').nth(1).unwrap_or("");
        let (username, password) = match userinfo.split_once(':') {
            Some((u, p)) => (u, Some(p)),
            None => (userinfo, None),
        };
        if !username.is_empty() || matches!(password, Some(p) if !p.is_empty()) {
            return Err(usage("invalid HTTPS browser origin"));
        }
    }
    if !query.is_empty() || !fragment.is_empty() {
        return Err(usage("invalid HTTPS browser origin"));
    }
    if !path.is_empty() && path != "/" {
        return Err(usage("invalid HTTPS browser origin"));
    }
    if value.contains(['\r', '\n']) {
        return Err(usage("invalid HTTPS browser origin"));
    }
    Ok(())
}

fn origin_host_port(value: &str) -> Result<OriginParts, ActivateError> {
    let (_, authority, _, _, _) = split_origin(value)?;
    let hostport = authority.rsplit('@').next().unwrap_or("");
    if hostport.starts_with('[') {
        let end = hostport
            .find(']')
            .ok_or_else(|| usage("invalid HTTPS browser origin"))?;
        let hostname = hostport[1..end].to_ascii_lowercase();
        let rest = &hostport[end + 1..];
        let port = if let Some(number) = rest.strip_prefix(':') {
            Some(
                number
                    .parse::<u16>()
                    .map_err(|_| runtime(format!("invalid port in {value:?}")))?,
            )
        } else {
            None
        };
        return Ok(OriginParts {
            hostname,
            port_present: port.is_some(),
            port,
        });
    }
    let (hostname, port) = match hostport.split_once(':') {
        Some((h, p)) => (
            h,
            Some(
                p.parse::<u16>()
                    .map_err(|_| runtime(format!("invalid port in {value:?}")))?,
            ),
        ),
        None => (hostport, None),
    };
    Ok(OriginParts {
        hostname: hostname.to_ascii_lowercase(),
        port_present: port.is_some(),
        port,
    })
}

fn activate(
    args: &CliArgs,
    paths: &Paths,
    sys: &mut dyn Sys,
    stdout: &mut dyn Write,
) -> Result<(), ActivateError> {
    if sys.euid() != 0 {
        return Err(usage("native host operator/root required"));
    }
    if args.local_tls {
        if args.certificate.is_some() || args.private_key.is_some() {
            return Err(usage(
                "local TLS and supplied certificates are separate choices",
            ));
        }
    } else if args.certificate.is_none() || args.private_key.is_none() {
        return Err(usage(
            "select --local-tls or supply both --certificate and --private-key",
        ));
    }
    let address: IpAddr = args
        .bind_ip
        .parse()
        .map_err(|e| runtime(format!("invalid bind IP {:?}: {e}", args.bind_ip)))?;
    if activate_rejects_ip(&address) {
        return Err(usage(
            "select an explicit private address, not loopback or a public administration exposure",
        ));
    }
    let root = &paths.root;
    if root.join("activated").exists() {
        return Err(usage("already activated; use explicit native configuration maintenance instead of blind reactivation"));
    }
    let raw = fs::read_to_string(root.join("dashboard.json"))
        .map_err(|e| runtime(format!("cannot read dashboard.json: {e}")))?;
    let cfg = JsonValue::parse(&raw).map_err(|_| runtime("dashboard.json is not valid JSON"))?;
    if !cfg.is_object() {
        return Err(runtime("dashboard configuration must be a JSON object"));
    }
    if cfg.get("public_url").is_some() {
        return Err(usage(
            "legacy separate-origin configuration; use rehearsed configuration maintenance",
        ));
    }
    let operator_id = match cfg.get("operator_id").and_then(|v| v.as_integer()) {
        Some(id) if 1 <= id && id <= i64::MAX as i128 => id as i64,
        _ => return Err(usage("dashboard operator identity is invalid")),
    };
    let forgejo_url = cfg
        .get("forgejo_url")
        .and_then(|v| v.as_str())
        .ok_or_else(|| runtime("dashboard forgejo_url must be a string"))?;
    check_browser_origin(forgejo_url)?;
    if args.local_tls {
        let origin = origin_host_port(forgejo_url)?;
        if origin.hostname != address.to_string()
            || (origin.port_present && origin.port != Some(443))
        {
            return Err(usage(
                "local TLS uses the exact selected private IP on HTTPS port 443",
            ));
        }
    }
    let internal = cfg
        .get("forgejo_internal_url")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let listen = cfg.get("listen").and_then(|v| v.as_str()).unwrap_or("");
    if internal != "http://127.0.0.1:3000" || listen != "127.0.0.1:8080" {
        return Err(usage(
            "bundled service recipes require the documented internal listeners",
        ));
    }
    let grant_key = cfg
        .get("grant_key_file")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let host_socket = cfg
        .get("host_socket")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let identity_socket = cfg
        .get("identity_socket")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let identity_socket = if identity_socket.is_empty() {
        "/run/soda/identity/admin.sock"
    } else {
        identity_socket
    };
    if grant_key != "/etc/soda/grant-key"
        || host_socket != "/run/soda/host.sock"
        || identity_socket != "/run/soda/identity/admin.sock"
    {
        return Err(usage(
            "bundled isolated service mounts require the standard private file and socket paths",
        ));
    }
    let (soda_uid, soda_gid) = sys.lookup_user("soda").map_err(ActivateError::Runtime)?;
    if soda_uid != 2000 || soda_gid != 2000 {
        return Err(usage("unexpected native soda service identity"));
    }
    // Mutations below run only after every validation above passed.
    fs::set_permissions(root, fs::Permissions::from_mode(0o750))
        .map_err(|e| runtime(e.to_string()))?;
    sys.chown(root, 0, soda_gid)
        .map_err(|e| runtime(e.to_string()))?;
    for name in ["dashboard.json", "grant-key"] {
        let path = root.join(name);
        if path.parent() != Some(root.as_path()) || is_symlink(&path) {
            return Err(usage(
                "credentials must be regular operator-created files under /etc/soda",
            ));
        }
        sys.chown(&path, 0, soda_gid)
            .map_err(|e| runtime(e.to_string()))?;
        fs::set_permissions(&path, fs::Permissions::from_mode(0o640))
            .map_err(|e| runtime(e.to_string()))?;
    }
    // The extension package needs only the stable operator ID for its navigation
    // guard. Keep it in the package's private data directory, outside the image.
    let extension_root = paths.var_lib.join("forgejo/gitea/extensions");
    for path in [
        &paths.var_lib,
        &paths.var_lib.join("forgejo"),
        &paths.var_lib.join("forgejo/gitea"),
        &extension_root,
    ] {
        if is_symlink(path) {
            return Err(usage(
                "Forgejo extension data path must not contain symlinks",
            ));
        }
    }
    let extension_data = extension_root.join(".data");
    fs::create_dir(&extension_root).map_err(|e| runtime(e.to_string()))?;
    sys.chown(&extension_root, 1000, 1000)
        .map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(&extension_root, fs::Permissions::from_mode(0o700))
        .map_err(|e| runtime(e.to_string()))?;
    if is_symlink(&extension_data) {
        return Err(usage(
            "Forgejo extension data path must not contain symlinks",
        ));
    }
    fs::create_dir(&extension_data).map_err(|e| runtime(e.to_string()))?;
    sys.chown(&extension_data, 1000, 1000)
        .map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(&extension_data, fs::Permissions::from_mode(0o700))
        .map_err(|e| runtime(e.to_string()))?;
    let operator_data = extension_data.join("soda");
    if is_symlink(&operator_data) {
        return Err(usage("Soda extension data path must not contain symlinks"));
    }
    fs::create_dir(&operator_data).map_err(|e| runtime(e.to_string()))?;
    sys.chown(&operator_data, 1000, 1000)
        .map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(&operator_data, fs::Permissions::from_mode(0o700))
        .map_err(|e| runtime(e.to_string()))?;
    let operator_file = operator_data.join("operator-id");
    if operator_file.exists() || is_symlink(&operator_file) {
        let st = fs::symlink_metadata(&operator_file).map_err(|e| runtime(e.to_string()))?;
        if !st.file_type().is_file() || st.nlink() != 1 {
            return Err(usage(
                "Soda extension operator identity must be a private regular file",
            ));
        }
    }
    fs::write(&operator_file, format!("{operator_id}\n")).map_err(|e| runtime(e.to_string()))?;
    sys.chown(&operator_file, 1000, 1000)
        .map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(&operator_file, fs::Permissions::from_mode(0o600))
        .map_err(|e| runtime(e.to_string()))?;
    let tls = root.join("tls");
    match fs::DirBuilder::new().mode(0o700).create(&tls) {
        Ok(()) => {}
        Err(e) if e.kind() == io::ErrorKind::AlreadyExists => {}
        Err(e) => return Err(runtime(e.to_string())),
    }
    if !args.local_tls {
        for (source, dest) in [
            (&args.certificate, "cert.pem"),
            (&args.private_key, "key.pem"),
        ] {
            let data =
                fs::read(source.as_deref().unwrap_or("")).map_err(|e| runtime(e.to_string()))?;
            let dest_path = tls.join(dest);
            fs::write(&dest_path, data).map_err(|e| runtime(e.to_string()))?;
            fs::set_permissions(&dest_path, fs::Permissions::from_mode(0o600))
                .map_err(|e| runtime(e.to_string()))?;
        }
    }
    let tls_input = if args.local_tls {
        "internal".to_string()
    } else {
        "/etc/soda/tls/cert.pem /etc/soda/tls/key.pem".to_string()
    };
    fs::write(
        root.join("proxy.env"),
        format!("FORGEJO_ORIGIN={forgejo_url}\nSODA_BIND={address}\nSODA_TLS={tls_input}\n"),
    )
    .map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(root.join("proxy.env"), fs::Permissions::from_mode(0o600))
        .map_err(|e| runtime(e.to_string()))?;
    rewrite_forgejo_env(&root.join("forgejo.env"), forgejo_url, &address.to_string())?;
    let network = paths.containers_systemd.join("forgejo.container.d");
    fs::create_dir_all(&network).map_err(|e| runtime(e.to_string()))?;
    let bind = match address {
        IpAddr::V6(_) => format!("[{address}]"),
        IpAddr::V4(_) => address.to_string(),
    };
    fs::write(network.join("10-private-network.conf"), format!("[Container]\nPublishPort=\nPublishPort=127.0.0.1:3000:3000\nPublishPort={bind}:2222:22\n"))
        .map_err(|e| runtime(e.to_string()))?;
    OpenOptions::new()
        .create(true)
        .write(true)
        .truncate(false)
        .mode(0o600)
        .open(root.join("activated"))
        .map_err(|e| runtime(e.to_string()))?;
    run_checked(sys, &["systemctl", "daemon-reload"])?;
    // No enable: Quadlet-generated units refuse it. Boot-time binding lives in
    // soda-console.service; activation only (re)starts the configured listeners.
    run_checked(sys, &["systemctl", "restart", "forgejo.service"])?;
    run_checked(
        sys,
        &[
            "systemctl",
            "start",
            "soda-dashboard.service",
            "soda-proxy.service",
        ],
    )?;
    let mut pending = vec![
        "forgejo.service",
        "soda-dashboard.service",
        "soda-proxy.service",
    ];
    let deadline = sys.now() + 60.0;
    while !pending.is_empty() && sys.now() < deadline {
        pending.retain(|unit| {
            sys.run(&["systemctl", "is-active", "--quiet", unit])
                .unwrap_or(1)
                != 0
        });
        if !pending.is_empty() {
            sys.sleep(5);
        }
    }
    if !pending.is_empty() {
        return Err(usage(format!(
            "activation started but not active: {}; inspect journalctl -u {}",
            pending.join(", "),
            pending[0]
        )));
    }
    let _ = writeln!(stdout, "Activation requested. Services report active; perform the later login/SSH validation before trusting browser setup.");
    if args.local_tls {
        let _ = writeln!(stdout, "Caddy manages local HTTPS certificates. Trust its public root certificate on each intended client before opening Soda; no domain or public certificate service is required.");
    }
    Ok(())
}

fn is_symlink(path: &Path) -> bool {
    fs::symlink_metadata(path)
        .map(|st| st.file_type().is_symlink())
        .unwrap_or(false)
}

fn run_checked(sys: &mut dyn Sys, argv: &[&str]) -> Result<(), ActivateError> {
    match sys.run(argv) {
        Ok(0) => Ok(()),
        Ok(code) => Err(runtime(format!(
            "Command {argv:?} returned non-zero exit status {code}."
        ))),
        Err(e) => Err(runtime(e.to_string())),
    }
}

fn rewrite_forgejo_env(path: &Path, forgejo_url: &str, address: &str) -> Result<(), ActivateError> {
    let raw = fs::read_to_string(path).map_err(|e| runtime(e.to_string()))?;
    let mut values: Vec<(String, String)> = Vec::new();
    for line in raw.split('\n') {
        let line = line.strip_suffix('\r').unwrap_or(line);
        if line.is_empty() || line.starts_with('#') || !line.contains('=') {
            continue;
        }
        let (key, value) = line.split_once('=').unwrap_or((line, ""));
        match values.iter_mut().find(|(k, _)| k == key) {
            Some(slot) => slot.1 = value.to_string(),
            None => values.push((key.to_string(), value.to_string())),
        }
    }
    let trimmed = forgejo_url.trim_end_matches('/');
    let domain = origin_host_port(forgejo_url)
        .map(|o| o.hostname)
        .unwrap_or_default();
    let set = |values: &mut Vec<(String, String)>, key: &str, value: String| match values
        .iter_mut()
        .find(|(k, _)| k == key)
    {
        Some(slot) => slot.1 = value,
        None => values.push((key.to_string(), value)),
    };
    set(
        &mut values,
        "FORGEJO__server__ROOT_URL",
        format!("{trimmed}/"),
    );
    set(&mut values, "FORGEJO__server__DOMAIN", domain);
    set(&mut values, "FORGEJO__server__SSH_PORT", "2222".to_string());
    set(
        &mut values,
        "FORGEJO__security__INSTALL_LOCK",
        "true".to_string(),
    );
    // Forgejo appends its email hash and size query to this supported provider URL.
    // Dynamic provider/federation settings remain native Forgejo administrator state.
    set(
        &mut values,
        "FORGEJO__picture__GRAVATAR_SOURCE",
        format!("{trimmed}/-/soda/avatars/v1/"),
    );
    set(
        &mut values,
        "FORGEJO__extensions__ENABLED",
        "true".to_string(),
    );
    set(
        &mut values,
        "FORGEJO__extensions__PATH",
        "/data/gitea/extensions".to_string(),
    );
    set(
        &mut values,
        "FORGEJO__extensions__REQUIRED_IDS",
        "soda".to_string(),
    );
    set(
        &mut values,
        "FORGEJO__extensions__SERVICE_CALLBACK_PATH",
        "/ipc/host.sock".to_string(),
    );
    // Service-bridge callers authenticate by Unix peer UID. The dashboard is
    // the only bridge client; it runs as the fixed soda identity (2000).
    set(
        &mut values,
        "FORGEJO__extensions__SERVICE_BRIDGE_PEERS",
        "2000:soda".to_string(),
    );
    if !values
        .iter()
        .any(|(k, _)| k == "FORGEJO__server__SSH_DOMAIN")
    {
        values.push((
            "FORGEJO__server__SSH_DOMAIN".to_string(),
            address.to_string(),
        ));
    }
    let mut out = String::new();
    for (key, value) in &values {
        out.push_str(key);
        out.push('=');
        out.push_str(value);
        out.push('\n');
    }
    fs::write(path, out).map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(path, fs::Permissions::from_mode(0o600))
        .map_err(|e| runtime(e.to_string()))?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;
    use std::sync::atomic::{AtomicU64, Ordering};

    static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

    struct FakeSys {
        euid: u32,
        user: Option<(u32, u32)>,
        calls: Vec<Vec<String>>,
        chowns: Vec<(PathBuf, u32, u32)>,
        probe_code: i32,
        now_values: Vec<f64>,
        sleeps: u32,
    }

    impl FakeSys {
        fn new() -> FakeSys {
            FakeSys {
                euid: 0,
                user: Some((2000, 2000)),
                calls: Vec::new(),
                chowns: Vec::new(),
                probe_code: 0,
                now_values: vec![0.0],
                sleeps: 0,
            }
        }
    }

    impl Sys for FakeSys {
        fn euid(&self) -> u32 {
            self.euid
        }
        fn lookup_user(&self, name: &str) -> Result<(u32, u32), String> {
            assert_eq!(name, "soda");
            self.user
                .ok_or_else(|| "getpwnam(): name not found: 'soda'".to_string())
        }
        fn chown(&mut self, path: &Path, uid: u32, gid: u32) -> io::Result<()> {
            self.chowns.push((path.to_path_buf(), uid, gid));
            Ok(())
        }
        fn run(&mut self, argv: &[&str]) -> io::Result<i32> {
            self.calls
                .push(argv.iter().map(|s| s.to_string()).collect());
            if argv.get(0..2) == Some(&["systemctl", "is-active"]) {
                return Ok(self.probe_code);
            }
            Ok(0)
        }
        fn now(&mut self) -> f64 {
            if self.now_values.len() > 1 {
                self.now_values.remove(0)
            } else {
                self.now_values[0]
            }
        }
        fn sleep(&mut self, secs: u64) {
            assert_eq!(secs, 5);
            self.sleeps += 1;
        }
    }

    struct Fixture {
        temp: PathBuf,
        paths: Paths,
    }

    fn fixture(dashboard: &str, forgejo_env: &str) -> Fixture {
        let seq = TEST_SEQ.fetch_add(1, Ordering::SeqCst);
        let temp =
            std::env::temp_dir().join(format!("soda-activate-test-{}-{seq}", std::process::id()));
        let root = temp.join("etc/soda");
        fs::create_dir_all(&root).expect("root");
        fs::write(root.join("dashboard.json"), dashboard).expect("dashboard");
        fs::write(root.join("grant-key"), "synthetic, not a credential").expect("key");
        fs::set_permissions(root.join("grant-key"), fs::Permissions::from_mode(0o600))
            .expect("key mode");
        fs::write(root.join("forgejo.env"), forgejo_env).expect("env");
        fs::create_dir_all(temp.join("var/lib/soda/forgejo/gitea")).expect("ext parent");
        Fixture {
            paths: Paths {
                root,
                var_lib: temp.join("var/lib/soda"),
                containers_systemd: temp.join("etc/containers/systemd"),
            },
            temp,
        }
    }

    fn dashboard(origin: &str, identity_socket: &str, operator_id: &str) -> String {
        format!(
            "{{\"forgejo_url\":{origin:?},\"forgejo_internal_url\":\"http://127.0.0.1:3000\",\"listen\":\"127.0.0.1:8080\",\"grant_key_file\":\"/etc/soda/grant-key\",\"host_socket\":\"/run/soda/host.sock\",\"identity_socket\":{identity_socket:?},\"operator_id\":{operator_id}}}"
        )
    }

    fn cli(bind_ip: &str, local_tls: bool, temp: &Path) -> CliArgs {
        if local_tls {
            CliArgs {
                bind_ip: bind_ip.to_string(),
                certificate: None,
                private_key: None,
                local_tls: true,
            }
        } else {
            let cert = temp.join("certificate");
            let key = temp.join("private-key");
            fs::write(&cert, "synthetic fixture").expect("cert");
            fs::write(&key, "synthetic fixture").expect("key");
            CliArgs {
                bind_ip: bind_ip.to_string(),
                certificate: Some(cert.to_string_lossy().into_owned()),
                private_key: Some(key.to_string_lossy().into_owned()),
                local_tls: false,
            }
        }
    }

    fn env_map(path: &Path) -> HashMap<String, String> {
        fs::read_to_string(path)
            .expect("env")
            .lines()
            .filter_map(|line| {
                line.split_once('=')
                    .map(|(k, v)| (k.to_string(), v.to_string()))
            })
            .collect()
    }

    #[test]
    fn first_activation_derives_provider_from_forgejo_origin() {
        for (origin, local_tls) in [
            ("https://forge.example.test:8443", false),
            ("https://[fd00::5]:8443/", false),
            ("https://192.168.2.100", true),
            ("https://[fd00::5]", true),
        ] {
            let fx = fixture(
                &dashboard(origin, "/run/soda/identity/admin.sock", "42"),
                "FORGEJO__ui__DEFAULT_THEME=soda-auto\nFORGEJO__server__SSH_DOMAIN=retained.example.test\n",
            );
            let address = if origin == "https://[fd00::5]" {
                "fd00::5"
            } else {
                "192.168.2.100"
            };
            // The non-local origins bind the same private address as the
            // Python suite's mapped run; only local-TLS binds the origin IP.
            let bind = if local_tls && address == "fd00::5" {
                "fd00::5"
            } else {
                "192.168.2.100"
            };
            let args = cli(bind, local_tls, &fx.temp);
            let mut sys = FakeSys::new();
            let mut stdout: Vec<u8> = Vec::new();
            activate(&args, &fx.paths, &mut sys, &mut stdout).expect("activate");
            let values = env_map(&fx.paths.root.join("forgejo.env"));
            assert_eq!(
                values["FORGEJO__picture__GRAVATAR_SOURCE"],
                format!("{}/-/soda/avatars/v1/", origin.trim_end_matches('/'))
            );
            assert_eq!(
                values["FORGEJO__server__SSH_DOMAIN"],
                "retained.example.test"
            );
            assert_eq!(values["FORGEJO__ui__DEFAULT_THEME"], "soda-auto");
            assert_eq!(values["FORGEJO__extensions__REQUIRED_IDS"], "soda");
            assert_eq!(
                values["FORGEJO__extensions__SERVICE_CALLBACK_PATH"],
                "/ipc/host.sock"
            );
            assert!(!values.keys().any(|k| k.contains("DISABLE_GRAVATAR")
                || k.contains("FEDERATED")
                || k.contains("OFFLINE_MODE")));
            // 3 activation phases + 3 is-active health probes.
            assert_eq!(sys.calls.len(), 6, "{origin}");
            let ext = fx.paths.var_lib.join("forgejo/gitea/extensions");
            assert_eq!(
                sys.chowns,
                vec![
                    (fx.paths.root.clone(), 0, 2000),
                    (fx.paths.root.join("dashboard.json"), 0, 2000),
                    (fx.paths.root.join("grant-key"), 0, 2000),
                    (ext.clone(), 1000, 1000),
                    (ext.join(".data"), 1000, 1000),
                    (ext.join(".data/soda"), 1000, 1000),
                    (ext.join(".data/soda/operator-id"), 1000, 1000),
                ]
            );
            for name in ["dashboard.json", "grant-key"] {
                assert_eq!(
                    fs::metadata(fx.paths.root.join(name))
                        .expect("m")
                        .permissions()
                        .mode()
                        & 0o777,
                    0o640
                );
            }
            assert_eq!(
                fs::read_to_string(ext.join(".data/soda/operator-id")).expect("op"),
                "42\n"
            );
            assert_eq!(
                fs::metadata(ext.join(".data/soda/operator-id"))
                    .expect("m")
                    .permissions()
                    .mode()
                    & 0o777,
                0o600
            );
            assert_eq!(
                fs::metadata(&ext).expect("m").permissions().mode() & 0o777,
                0o700
            );
            let proxy = fs::read_to_string(fx.paths.root.join("proxy.env")).expect("proxy");
            assert!(proxy.contains(if local_tls {
                "SODA_TLS=internal\n"
            } else {
                "SODA_TLS=/etc/soda/tls/cert.pem /etc/soda/tls/key.pem\n"
            }));
            assert_eq!(fx.paths.root.join("tls/cert.pem").exists(), !local_tls);
        }
    }

    #[test]
    fn tls_mode_rejects_missing_or_mixed_inputs_before_effects() {
        for (cert, key, local_tls) in [
            (false, false, false),
            (true, false, false),
            (true, false, true),
            (false, true, true),
        ] {
            let fx = fixture(
                &dashboard("https://192.168.1.5", "/run/soda/identity/admin.sock", "42"),
                "",
            );
            let args = CliArgs {
                bind_ip: "192.168.1.5".to_string(),
                certificate: cert.then(|| "/missing".to_string()),
                private_key: key.then(|| "/missing".to_string()),
                local_tls,
            };
            let mut sys = FakeSys::new();
            let mut stdout: Vec<u8> = Vec::new();
            let err = activate(&args, &fx.paths, &mut sys, &mut stdout).expect_err("rejected");
            assert!(matches!(err, ActivateError::Usage(_)), "{err:?}");
            assert!(sys.calls.is_empty(), "effects before validation");
            assert!(!fx.paths.root.join("proxy.env").exists());
        }
    }

    #[test]
    fn empty_identity_socket_uses_standard_admin_socket() {
        let fx = fixture(
            &dashboard("https://192.168.2.100", "", "7"),
            "FORGEJO__ui__DEFAULT_THEME=soda-auto\n",
        );
        let args = cli("192.168.2.100", true, &fx.temp);
        let mut sys = FakeSys::new();
        let mut stdout: Vec<u8> = Vec::new();
        activate(&args, &fx.paths, &mut sys, &mut stdout).expect("activate");
        assert_eq!(
            fs::read_to_string(
                fx.paths
                    .var_lib
                    .join("forgejo/gitea/extensions/.data/soda/operator-id")
            )
            .expect("op"),
            "7\n"
        );
        assert_eq!(
            env_map(&fx.paths.root.join("forgejo.env"))
                ["FORGEJO__extensions__SERVICE_BRIDGE_PEERS"],
            "2000:soda"
        );
    }

    #[test]
    fn reports_units_that_never_become_active() {
        let fx = fixture(
            &dashboard(
                "https://192.168.2.100",
                "/run/soda/identity/admin.sock",
                "7",
            ),
            "",
        );
        let args = cli("192.168.2.100", true, &fx.temp);
        let mut sys = FakeSys::new();
        sys.probe_code = 1;
        sys.now_values = vec![0.0, 61.0, 61.0];
        let mut stdout: Vec<u8> = Vec::new();
        let err = activate(&args, &fx.paths, &mut sys, &mut stdout).expect_err("inactive");
        assert_eq!(
            err,
            ActivateError::Usage(
                "activation started but not active: forgejo.service, soda-dashboard.service, soda-proxy.service; inspect journalctl -u forgejo.service".to_string()
            )
        );
    }

    #[test]
    fn ip_classifier_matches_cpython_oracle() {
        // (address, accepted?) from the interpreter's own verdicts.
        for (addr, accepted) in [
            ("192.168.1.5", true),
            ("10.0.0.1", true),
            ("172.16.0.1", true),
            ("8.8.8.8", false),
            ("127.0.0.1", false),
            ("0.0.0.0", false),
            ("224.0.0.1", false),
            ("255.255.255.255", true),
            ("169.254.1.1", true),
            ("100.64.0.1", true),
            ("192.0.2.1", true),
            ("198.51.100.1", true),
            ("203.0.113.1", true),
            ("198.18.0.1", true),
            ("192.0.0.1", true),
            ("192.0.0.9", false),
            ("192.0.0.10", false),
            ("240.0.0.1", true),
            ("0.0.0.1", true),
            ("::1", false),
            ("::", false),
            ("fe80::1", true),
            ("fc00::1", true),
            ("fd00::5", true),
            ("ff02::1", false),
            ("2001:db8::1", true),
            ("2001:4860:4860::8888", false),
            ("::ffff:192.168.1.1", true),
            ("::ffff:8.8.8.8", false),
            ("::ffff:0.0.0.0", true),
            ("::ffff:127.0.0.1", true),
            ("::ffff:10.0.0.1", true),
            ("::ffff:255.255.255.255", true),
            ("::ffff:192.0.2.1", true),
            ("::ffff:169.254.1.1", true),
            ("::ffff:100.64.0.1", true),
            ("100::1", true),
            ("64:ff9b::808:808", false),
            ("2001::1", true),
            ("2001:30::1", false),
            ("2002:c000:0200::1", true),
            ("64:ff9b:1::1", true),
            ("::ffff:0:101:101", false),
        ] {
            let ip: IpAddr = addr.parse().expect("fixture parses");
            assert_eq!(!activate_rejects_ip(&ip), accepted, "{addr}");
        }
    }

    #[test]
    fn browser_origin_checks_match_activate_rules() {
        for url in [
            "https://forgejo.test",
            "https://forgejo.test/",
            "https://192.168.2.100",
            "https://[fd00::5]:8443/",
            "HTTPS://Forgejo.Test",
        ] {
            assert!(check_browser_origin(url).is_ok(), "{url}");
        }
        for url in [
            "http://forgejo.test",
            "https://forgejo.test/x",
            "https://user@forgejo.test",
            "https://u:p@forgejo.test",
            "https://forgejo.test?x",
            "https://forgejo.test#x",
            "https://",
            "https://forgejo.test\n",
            "not a url",
        ] {
            assert!(check_browser_origin(url).is_err(), "{url}");
        }
        // Empty userinfo fields are falsy in the Python check, so they pass.
        assert!(check_browser_origin("https://@forgejo.test").is_ok());
        assert!(check_browser_origin("https://:@forgejo.test").is_ok());
    }

    #[test]
    fn operator_identity_encodings() {
        for (operator_id, ok) in [
            ("42", true),
            ("1", true),
            ("9223372036854775807", true),
            ("0", false),
            ("-1", false),
            ("9223372036854775808", false),
            ("7.0", false),
            ("true", false),
            ("\"42\"", false),
            ("null", false),
        ] {
            let fx = fixture(
                &dashboard(
                    "https://192.168.2.100",
                    "/run/soda/identity/admin.sock",
                    operator_id,
                ),
                "",
            );
            let args = cli("192.168.2.100", true, &fx.temp);
            let mut sys = FakeSys::new();
            let mut stdout: Vec<u8> = Vec::new();
            assert_eq!(
                activate(&args, &fx.paths, &mut sys, &mut stdout).is_ok(),
                ok,
                "{operator_id}"
            );
        }
        // public_url marks a legacy separate-origin configuration.
        let fx = fixture(
            "{\"forgejo_url\":\"https://192.168.2.100\",\"public_url\":\"https://x\",\"operator_id\":7}",
            "",
        );
        let args = cli("192.168.2.100", true, &fx.temp);
        let mut sys = FakeSys::new();
        let mut stdout: Vec<u8> = Vec::new();
        assert_eq!(
            activate(&args, &fx.paths, &mut sys, &mut stdout),
            Err(usage(
                "legacy separate-origin configuration; use rehearsed configuration maintenance"
            ))
        );
    }

    #[test]
    fn cli_parsing_matches_argparse() {
        let argv = |words: &[&str]| words.iter().map(|w| w.to_string()).collect::<Vec<_>>();
        let args = parse_args(
            "soda-activate",
            &argv(&["--bind-ip", "192.168.1.5", "--local-tls"]),
        )
        .expect("ok")
        .expect("args");
        assert!(args.local_tls);
        assert_eq!(args.bind_ip, "192.168.1.5");
        // Unambiguous prefixes and --opt=value work like argparse.
        let args = parse_args("soda-activate", &argv(&["--bind=192.168.1.5", "--local"]))
            .expect("ok")
            .expect("args");
        assert_eq!(args.bind_ip, "192.168.1.5");
        assert!(parse_args("soda-activate", &argv(&["-h"]))
            .expect("help")
            .is_none());
        assert!(parse_args("soda-activate", &argv(&["--help"]))
            .expect("help")
            .is_none());
        assert_eq!(
            parse_args("soda-activate", &argv(&["--local-tls"])),
            Err("the following arguments are required: --bind-ip".to_string())
        );
        assert!(parse_args("soda-activate", &argv(&["--bind-ip", "x", "--bogus"])).is_err());
        assert!(parse_args("soda-activate", &argv(&["--bind-ip"])).is_err());
        assert!(parse_args("soda-activate", &argv(&["--bind-ip", "x", "positional"])).is_err());
        assert!(parse_args("soda-activate", &argv(&["--local-tls=x", "--bind-ip", "y"])).is_err());
    }

    #[test]
    fn refuses_without_root_before_effects() {
        let fx = fixture(
            &dashboard(
                "https://192.168.2.100",
                "/run/soda/identity/admin.sock",
                "7",
            ),
            "",
        );
        let args = cli("192.168.2.100", true, &fx.temp);
        let mut sys = FakeSys::new();
        sys.euid = 1000;
        let mut stdout: Vec<u8> = Vec::new();
        assert_eq!(
            activate(&args, &fx.paths, &mut sys, &mut stdout),
            Err(usage("native host operator/root required"))
        );
        assert!(sys.calls.is_empty());
    }
}
