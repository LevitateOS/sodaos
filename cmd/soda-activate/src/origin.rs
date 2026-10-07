use std::net::IpAddr;
use url::Url;

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

fn split_origin(value: &str) -> Result<Url, ActivateError> {
    if value.bytes().any(|b| b <= 0x20 || b == 0x7f || b == b'\\')
        || !value.contains("://")
        || !valid_percent_escapes(value)
    {
        return Err(usage("invalid HTTPS browser origin"));
    }
    let (_, rest) = value
        .split_once("://")
        .ok_or_else(|| usage("invalid HTTPS browser origin"))?;
    let authority = rest.split(['/', '?', '#']).next().unwrap_or("");
    if authority.is_empty()
        || authority.starts_with('[')
            && authority
                .split_once(']')
                .is_some_and(|(inside, _)| inside.contains('%'))
    {
        return Err(usage("invalid HTTPS browser origin"));
    }
    let path = rest
        .find('/')
        .map(|at| rest[at..].split(['?', '#']).next().unwrap_or(""))
        .unwrap_or("");
    if !matches!(path, "" | "/") {
        return Err(usage("invalid HTTPS browser origin"));
    }
    Url::parse(value).map_err(|_| usage("invalid HTTPS browser origin"))
}

pub(crate) fn check_browser_origin(value: &str) -> Result<(), ActivateError> {
    let parsed = split_origin(value)?;
    if parsed.scheme() != "https" {
        return Err(usage("invalid HTTPS browser origin"));
    }
    let Some(hostname) = parsed.host().map(host_to_string) else {
        return Err(usage("invalid HTTPS browser origin"));
    };
    if hostname.is_empty() {
        return Err(usage("invalid HTTPS browser origin"));
    }
    let authority = raw_authority(value);
    if authority.is_some_and(|authority| authority.contains('@')) {
        let userinfo = authority.unwrap().rsplit('@').nth(1).unwrap_or("");
        let (username, password) = match userinfo.split_once(':') {
            Some((u, p)) => (u, Some(p)),
            None => (userinfo, None),
        };
        if !username.is_empty() || matches!(password, Some(p) if !p.is_empty()) {
            return Err(usage("invalid HTTPS browser origin"));
        }
    }
    if parsed.query().is_some_and(|query| !query.is_empty())
        || parsed
            .fragment()
            .is_some_and(|fragment| !fragment.is_empty())
    {
        return Err(usage("invalid HTTPS browser origin"));
    }
    if !matches!(parsed.path(), "" | "/") {
        return Err(usage("invalid HTTPS browser origin"));
    }
    if value.contains(['\r', '\n']) {
        return Err(usage("invalid HTTPS browser origin"));
    }
    Ok(())
}

pub(crate) fn origin_host_port(value: &str) -> Result<OriginParts, ActivateError> {
    let parsed = split_origin(value)?;
    let hostname = parsed.host().map(host_to_string).unwrap_or_default();
    let authority = raw_authority(value).unwrap_or("");
    let hostport = authority.rsplit('@').next().unwrap_or("");
    let raw_port = if hostport.starts_with('[') {
        hostport
            .split_once(']')
            .and_then(|(_, rest)| rest.strip_prefix(':'))
    } else {
        hostport.rsplit_once(':').map(|(_, port)| port)
    };
    let port = raw_port
        .map(|raw| {
            raw.parse::<u16>()
                .map_err(|_| runtime(format!("invalid port in {value:?}")))
        })
        .transpose()?;
    Ok(OriginParts {
        hostname,
        port_present: raw_port.is_some(),
        port,
    })
}

fn raw_authority(value: &str) -> Option<&str> {
    let (_, rest) = value.split_once("://")?;
    Some(rest.split(['/', '?', '#']).next().unwrap_or(""))
}

fn host_to_string(host: url::Host<&str>) -> String {
    match host {
        url::Host::Domain(domain) => domain.to_owned(),
        url::Host::Ipv4(address) => address.to_string(),
        url::Host::Ipv6(address) => address.to_string(),
    }
}

fn valid_percent_escapes(value: &str) -> bool {
    let bytes = value.as_bytes();
    let mut at = 0;
    while at < bytes.len() {
        if bytes[at] == b'%' {
            if at + 2 >= bytes.len()
                || !bytes[at + 1].is_ascii_hexdigit()
                || !bytes[at + 2].is_ascii_hexdigit()
            {
                return false;
            }
            at += 3;
        } else {
            at += 1;
        }
    }
    true
}
