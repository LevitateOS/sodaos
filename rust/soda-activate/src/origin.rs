use std::net::IpAddr;

use crate::system::{runtime, usage, ActivateError};

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

pub(crate) fn activate_rejects_ip(addr: &IpAddr) -> bool {
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

pub(crate) struct OriginParts {
    pub(crate) hostname: String,
    pub(crate) port: Option<u16>,
    pub(crate) port_present: bool,
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

pub(crate) fn check_browser_origin(value: &str) -> Result<(), ActivateError> {
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

pub(crate) fn origin_host_port(value: &str) -> Result<OriginParts, ActivateError> {
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
