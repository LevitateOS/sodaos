use std::collections::{BTreeSet, HashMap};
use std::net::{IpAddr, TcpStream};
use std::time::Duration;

use super::{get_str, parse_json, podman, read_text, run_output, ProbeFailure};

/// One observed listener.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord)]
pub(super) struct Listener {
    address: String,
    port: u16,
}

/// Parse `ss -H -ltn` rows into listeners.
pub(super) fn parse_listeners(output: &str) -> Result<BTreeSet<Listener>, ProbeFailure> {
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
        let port: u16 = port
            .parse()
            .map_err(|_| ProbeFailure::failed("parse listeners"))?;
        listeners.insert(Listener {
            address: address.trim_matches(|c| c == '[' || c == ']').to_string(),
            port,
        });
    }
    Ok(listeners)
}

pub(super) fn bound(
    listeners: &BTreeSet<Listener>,
    address: &str,
    port: u16,
) -> Result<(), ProbeFailure> {
    if !listeners.contains(&Listener {
        address: address.to_string(),
        port,
    }) {
        return Err(ProbeFailure::exit("required service binding missing"));
    }
    for listener in listeners {
        if listener.port != port {
            continue;
        }
        let allowed =
            listener.address == address || (address == "127.0.0.1" && listener.address == "::1");
        if !allowed {
            return Err(ProbeFailure::exit("unexpected additional service binding"));
        }
    }
    Ok(())
}

/// Stock Cockpit's socket listens on every interface.
pub(super) fn cockpit_bound(listeners: &BTreeSet<Listener>) -> Result<(), ProbeFailure> {
    let mut found = false;
    for listener in listeners {
        if listener.port != 9090 {
            continue;
        }
        if !matches!(listener.address.as_str(), "*" | "::" | "0.0.0.0") {
            return Err(ProbeFailure::exit("unexpected additional service binding"));
        }
        found = true;
    }
    if !found {
        return Err(ProbeFailure::exit("required service binding missing"));
    }
    Ok(())
}

fn published(
    listeners: &BTreeSet<Listener>,
    address: &str,
    host_port: u16,
    container_port: u16,
) -> Result<(), ProbeFailure> {
    let mapping = podman(
        &["port", "soda-forgejo", &format!("{container_port}/tcp")],
        "read published ports",
    )?;
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
            return Err(ProbeFailure::exit(
                "unexpected host listener on published port",
            ));
        }
    }
    let ip: IpAddr = address
        .parse()
        .map_err(|_| ProbeFailure::failed("connect published endpoint"))?;
    TcpStream::connect_timeout(
        &std::net::SocketAddr::new(ip, host_port),
        Duration::from_secs(5),
    )
    .map_err(|_| ProbeFailure::failed("connect published endpoint"))?;
    Ok(())
}

/// RFC 1918 plus shared `100.64/10` for v4, ULA for v6: the bind
/// addresses operators actually configure. Python's `is_private`
/// agrees on all of these; exotic ranges are out of scope.
pub(super) fn is_private_bind(ip: IpAddr) -> bool {
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
pub(super) fn split_origin(origin: &str) -> Result<(String, u16), ProbeFailure> {
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
            Some(digits) => digits
                .parse::<u16>()
                .map_err(|_| ProbeFailure::failed("parse origin"))?,
            None if after.is_empty() => 443,
            None => return Err(err()),
        };
        // `urlsplit` lowercases every hostname, bracketed or not.
        (host.to_lowercase(), port)
    } else if let Some((host, digits)) = host_port.rsplit_once(':') {
        if host.is_empty() || digits.is_empty() || !digits.bytes().all(|b| b.is_ascii_digit()) {
            return Err(err());
        }
        (
            host.to_lowercase(),
            digits
                .parse::<u16>()
                .map_err(|_| ProbeFailure::failed("parse origin"))?,
        )
    } else {
        (host_port.to_lowercase(), 443)
    };
    // Python's `origin.port or 443`: an explicit zero still means 443.
    let port = if port == 0 { 443 } else { port };
    Ok((host, port))
}

/// Parse `KEY=value` environment files. Empty lines are skipped; a
/// line without `=` fails, like the owner's unpacking split.
pub(super) fn parse_env_file(text: &str) -> Result<HashMap<String, String>, ProbeFailure> {
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

pub(super) fn tls_material_paths(
    mode: Option<&str>,
) -> Result<&'static [&'static str], ProbeFailure> {
    match mode {
        Some("internal") => Ok(&[
            "/var/lib/soda/proxy/caddy/pki/authorities/local/root.crt",
            "/var/lib/soda/proxy/caddy/pki/authorities/local/root.key",
            "/var/lib/soda/proxy/caddy/pki/authorities/local/intermediate.crt",
            "/var/lib/soda/proxy/caddy/pki/authorities/local/intermediate.key",
        ]),
        Some("/etc/soda/tls/cert.pem /etc/soda/tls/key.pem") => {
            Ok(&["/etc/soda/tls/cert.pem", "/etc/soda/tls/key.pem"])
        }
        _ => Err(ProbeFailure::exit("unsupported configured TLS mode")),
    }
}

/// Listener, publication, credential-mode, and origin boundaries: the
/// second `host.sh` Python block.
pub fn host_listeners(phase: &str) -> Result<String, ProbeFailure> {
    let rows = run_output(&["ss", "-H", "-ltn"], "read listeners")?;
    let listeners = parse_listeners(&rows)?;
    cockpit_bound(&listeners)?;
    published(&listeners, "127.0.0.1", 3000, 3000)?;
    if phase == "activated" {
        let cfg = parse_json(
            &read_text("/etc/soda/dashboard.json", "read dashboard.json")?,
            "parse dashboard.json",
        )?;
        // Retired bootstrap files are not dashboard credentials and need not exist.
        let grant = get_str(&cfg, "grant_key_file", "parse dashboard.json")?;
        let path = std::path::Path::new(grant);
        let info = std::fs::symlink_metadata(path).map_err(|_| {
            ProbeFailure::exit("configured dashboard credential permissions invalid")
        })?;
        use std::os::unix::fs::MetadataExt;
        if !grant.starts_with('/')
            || !info.is_file()
            || info.uid() != 0
            || info.gid() != 2000
            || info.mode() & 0o7777 != 0o640
        {
            return Err(ProbeFailure::exit(
                "configured dashboard credential permissions invalid",
            ));
        }
        // These are the variables actually consumed by the packaged Caddy config,
        // not ports reconstructed from a client's unrelated browser tunnel.
        let env = parse_env_file(&read_text("/etc/soda/proxy.env", "read proxy.env")?)?;
        for path in tls_material_paths(env.get("SODA_TLS").map(String::as_str))? {
            let info = std::fs::symlink_metadata(path)
                .map_err(|_| ProbeFailure::exit("TLS material permissions invalid"))?;
            if !info.is_file() || info.uid() != 0 || info.mode() & 0o7777 != 0o600 {
                return Err(ProbeFailure::exit("TLS material permissions invalid"));
            }
        }
        let bind_raw = env
            .get("SODA_BIND")
            .ok_or_else(|| ProbeFailure::failed("parse proxy.env"))?;
        let bind: IpAddr = bind_raw
            .parse()
            .map_err(|_| ProbeFailure::exit("private explicit proxy bind required"))?;
        if !is_private_bind(bind) || bind.is_unspecified() {
            return Err(ProbeFailure::exit("private explicit proxy bind required"));
        }
        let address = bind.to_string();
        if cfg.get("public_url").is_some() || env.contains_key("SODA_ORIGIN") {
            return Err(ProbeFailure::exit(
                "legacy separate-origin configuration requires rehearsed maintenance",
            ));
        }
        let forgejo_origin = env
            .get("FORGEJO_ORIGIN")
            .ok_or_else(|| ProbeFailure::failed("parse proxy.env"))?;
        if forgejo_origin.trim_end_matches('/')
            != get_str(&cfg, "forgejo_url", "parse dashboard.json")?.trim_end_matches('/')
        {
            return Err(ProbeFailure::exit("proxy and API browser origin mismatch"));
        }
        let (host, port) = split_origin(forgejo_origin)?;
        bound(&listeners, &host, port)?;
        let listen = get_str(&cfg, "listen", "parse dashboard.json")?;
        let Some((host, port)) = listen.rsplit_once(':') else {
            return Err(ProbeFailure::failed("parse dashboard.json"));
        };
        let port: u16 = port
            .trim()
            .parse()
            .map_err(|_| ProbeFailure::failed("parse dashboard.json"))?;
        bound(
            &listeners,
            host.trim_matches(|c| c == '[' || c == ']'),
            port,
        )?;
        published(&listeners, &address, 2222, 22)?;
    } else {
        published(&listeners, "127.0.0.1", 2222, 22)?;
    }
    Ok("Configured native listener and credential-mode boundaries observed; not a login or client-route proof.\n".to_string())
}
