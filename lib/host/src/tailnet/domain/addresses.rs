use super::{lower_char, unavailable};
use std::net::{IpAddr, Ipv4Addr, Ipv6Addr};

fn valid_label(label: &str) -> bool {
    let b = label.as_bytes();
    if b.is_empty() || b.len() > 63 || b[0] == b'-' || b[b.len() - 1] == b'-' {
        return false;
    }
    b.iter()
        .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || *c == b'-')
}

/// Mirror of `CanonicalMagicDNSName`.
pub(crate) fn canonical_magic_dns_name(value: &str) -> Result<String, String> {
    // `strings.TrimSpace` is Unicode White_Space plus U+0085 (NEL); Rust's
    // `char::is_whitespace` omits the Latin-1 special case.
    let trimmed = value.trim_matches(|c: char| c.is_whitespace() || c == '\u{85}');
    let dotted = trimmed.strip_suffix('.').unwrap_or(trimmed);
    let name: String = dotted.chars().map(lower_char).collect();
    if name.len() > 253 || !name.contains('.') || name.ends_with(".local") {
        return unavailable();
    }
    for label in name.split('.') {
        if !valid_label(label) {
            return unavailable();
        }
    }
    Ok(name)
}

struct ParsedAddr {
    addr: IpAddr,
    zone: Option<String>,
}

/// Mirror of `netip.ParseAddr` for the forms `addresses` can accept.
fn parse_addr(s: &str) -> Option<ParsedAddr> {
    // Zone splits at the first `%`; any non-empty zone is kept verbatim.
    let (head, zone) = match s.find('%') {
        Some(i) => {
            let z = s.get(i + 1..)?;
            if z.is_empty() {
                return None;
            }
            (s.get(..i)?, Some(z.to_string()))
        }
        None => (s, None),
    };
    if head.contains(':') {
        Some(ParsedAddr {
            addr: head.parse::<Ipv6Addr>().ok()?.into(),
            zone,
        })
    } else {
        if zone.is_some() {
            return None;
        }
        Some(ParsedAddr {
            addr: head.parse::<Ipv4Addr>().ok()?.into(),
            zone: None,
        })
    }
}

impl ParsedAddr {
    fn canonical(&self) -> String {
        let mut s = self.addr.to_string();
        if let Some(z) = &self.zone {
            s.push('%');
            s.push_str(z);
        }
        s
    }
}

/// Tailnet's global-unicast rule excludes special-use classes while allowing
/// private, CGNAT and reserved unicast ranges.
fn is_global_unicast(a: &ParsedAddr) -> bool {
    match a.addr {
        IpAddr::V4(ip) => {
            let b = ip.octets();
            !ip.is_unspecified()
                && !ip.is_loopback()
                && !ip.is_multicast()
                && ip != Ipv4Addr::BROADCAST
                && !(b[0] == 169 && b[1] == 254)
        }
        IpAddr::V6(ip) => {
            let ip = ip
                .to_ipv4_mapped()
                .map(IpAddr::V4)
                .unwrap_or(IpAddr::V6(ip));
            match ip {
                IpAddr::V4(v4) => {
                    !v4.is_unspecified()
                        && !v4.is_loopback()
                        && !v4.is_multicast()
                        && v4 != Ipv4Addr::BROADCAST
                        && !v4.is_link_local()
                }
                IpAddr::V6(v6) => {
                    !v6.is_unspecified()
                        && !v6.is_loopback()
                        && !v6.is_multicast()
                        && v6.segments()[0] & 0xffc0 != 0xfe80
                }
            }
        }
    }
}

/// Mirror of `addresses` in control.go.
fn check_addresses(v: &[String]) -> Result<Vec<String>, String> {
    if v.len() > 16 {
        return unavailable();
    }
    for s in v {
        let a = match parse_addr(s) {
            Some(a) => a,
            None => return unavailable(),
        };
        if !is_global_unicast(&a) || a.canonical() != *s {
            return unavailable();
        }
    }
    Ok(v.to_vec())
}

/// Mirror of `peerView` (ID empty on the project path; kept for parity).
fn peer_view(id: &str, dns_name: &str, ips: &[String]) -> Result<(Vec<String>, String), String> {
    if id.len() > 128 || id.contains(['\r', '\n', '\0']) {
        return unavailable();
    }
    let mut name = String::new();
    if !dns_name.is_empty() {
        name = canonical_magic_dns_name(dns_name)?;
    }
    let addrs = check_addresses(ips)?;
    Ok((addrs, name))
}

/// Mirror of `resolveProjectPeer`.
pub(crate) fn resolve_project_peer(
    dns_name: &str,
    ips: &[String],
) -> Result<(Vec<String>, String), String> {
    let (addrs, name) = peer_view("", dns_name, ips)?;
    if addrs.is_empty() {
        return unavailable();
    }
    Ok((addrs, name))
}
