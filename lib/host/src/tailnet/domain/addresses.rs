use super::{lower_char, unavailable};

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

enum Addr {
    V4([u8; 4]),
    V6([u16; 8]),
}

struct ParsedAddr {
    addr: Addr,
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
            addr: Addr::V6(parse_ipv6(head)?),
            zone,
        })
    } else {
        if zone.is_some() {
            return None;
        }
        Some(ParsedAddr {
            addr: Addr::V4(parse_ipv4(head)?),
            zone: None,
        })
    }
}

fn parse_ipv4(s: &str) -> Option<[u8; 4]> {
    let mut out = [0u8; 4];
    let mut parts = s.split('.');
    for slot in out.iter_mut() {
        let p = parts.next()?;
        if p.is_empty() || p.len() > 3 || !p.bytes().all(|c| c.is_ascii_digit()) {
            return None;
        }
        if p.len() > 1 && p.as_bytes()[0] == b'0' {
            return None;
        }
        let v: u32 = p.parse().ok()?;
        if v > 255 {
            return None;
        }
        *slot = v as u8;
    }
    if parts.next().is_some() {
        return None;
    }
    Some(out)
}

fn parse_h16(g: &str) -> Option<u16> {
    if g.is_empty() || g.len() > 4 || !g.bytes().all(|c| c.is_ascii_hexdigit()) {
        return None;
    }
    u16::from_str_radix(g, 16).ok()
}

fn parse_ipv6(s: &str) -> Option<[u16; 8]> {
    // An embedded dotted quad is allowed only as the final 32 bits.
    let (head, tail): (&str, Option<[u16; 2]>) = if s.contains('.') {
        let (h, dotted) = s.rsplit_once(':')?;
        let v4 = parse_ipv4(dotted)?;
        let hi = ((v4[0] as u16) << 8) | v4[1] as u16;
        let lo = ((v4[2] as u16) << 8) | v4[3] as u16;
        (h, Some([hi, lo]))
    } else {
        (s, None)
    };
    let extra = if tail.is_some() { 2 } else { 0 };
    let mut groups: Vec<u16> = Vec::with_capacity(8);
    match head.find("::") {
        None => {
            if head.is_empty() {
                return None;
            }
            for g in head.split(':') {
                groups.push(parse_h16(g)?);
            }
            if groups.len() + extra != 8 {
                return None;
            }
        }
        Some(at) => {
            let left = head.get(..at)?;
            let right = head.get(at + 2..)?;
            if right.contains("::") {
                return None;
            }
            if !left.is_empty() {
                for g in left.split(':') {
                    groups.push(parse_h16(g)?);
                }
            }
            let left_len = groups.len();
            let mut right_groups: Vec<u16> = Vec::new();
            if !right.is_empty() {
                for g in right.split(':') {
                    right_groups.push(parse_h16(g)?);
                }
            }
            if left_len + right_groups.len() + extra >= 8 {
                return None;
            }
            groups.extend(std::iter::repeat_n(
                0,
                8 - extra - left_len - right_groups.len(),
            ));
            groups.extend(right_groups);
        }
    }
    if let Some(t) = tail {
        groups.push(t[0]);
        groups.push(t[1]);
    }
    if groups.len() != 8 {
        return None;
    }
    let mut out = [0u16; 8];
    out.copy_from_slice(&groups);
    Some(out)
}

fn is_4in6(g: &[u16; 8]) -> bool {
    g[0] == 0 && g[1] == 0 && g[2] == 0 && g[3] == 0 && g[4] == 0 && g[5] == 0xffff
}

/// RFC 5952 §4.2.3 longest-run compression: runs under length 2 are left in
/// place and the first run wins ties, exactly like `netip.Addr.String`.
fn compress_v6(g: &[u16; 8]) -> String {
    let mut best_at = 0usize;
    let mut best_len = 0usize;
    let mut i = 0;
    while i < 8 {
        if g[i] == 0 {
            let mut j = i;
            while j < 8 && g[j] == 0 {
                j += 1;
            }
            if j - i > best_len {
                best_at = i;
                best_len = j - i;
            }
            i = j;
        } else {
            i += 1;
        }
    }
    if best_len < 2 {
        return g
            .iter()
            .map(|x| format!("{x:x}"))
            .collect::<Vec<_>>()
            .join(":");
    }
    let left = g[..best_at]
        .iter()
        .map(|x| format!("{x:x}"))
        .collect::<Vec<_>>()
        .join(":");
    let right = g[best_at + best_len..]
        .iter()
        .map(|x| format!("{x:x}"))
        .collect::<Vec<_>>()
        .join(":");
    if left.is_empty() && right.is_empty() {
        return "::".to_string();
    }
    if left.is_empty() {
        return format!("::{right}");
    }
    if right.is_empty() {
        return format!("{left}::");
    }
    format!("{left}::{right}")
}

impl ParsedAddr {
    /// Mirror of `netip.Addr.String` (zone kept verbatim).
    fn canonical(&self) -> String {
        let mut s = match &self.addr {
            Addr::V4(v) => format!("{}.{}.{}.{}", v[0], v[1], v[2], v[3]),
            Addr::V6(g) => {
                if is_4in6(g) {
                    format!(
                        "::ffff:{}.{}.{}.{}",
                        (g[6] >> 8) as u8,
                        g[6] as u8,
                        (g[7] >> 8) as u8,
                        g[7] as u8
                    )
                } else {
                    compress_v6(g)
                }
            }
        };
        if let Some(z) = &self.zone {
            s.push('%');
            s.push_str(z);
        }
        s
    }
}

fn v4_global_unicast(v: &[u8; 4]) -> bool {
    if *v == [0, 0, 0, 0] || *v == [255, 255, 255, 255] {
        return false;
    }
    if v[0] == 127 || v[0] & 0xf0 == 0xe0 {
        return false;
    }
    if v[0] == 169 && v[1] == 254 {
        return false;
    }
    true
}

/// Mirror of `netip.Addr.IsGlobalUnicast` (probed: private/CGNAT/reserved
/// pass; only unspecified, broadcast, loopback, link-local and multicast
/// fail; 4-in-6 follows the mapped v4 address).
fn is_global_unicast(a: &ParsedAddr) -> bool {
    match &a.addr {
        Addr::V4(v) => v4_global_unicast(v),
        Addr::V6(g) => {
            if is_4in6(g) {
                let v = [(g[6] >> 8) as u8, g[6] as u8, (g[7] >> 8) as u8, g[7] as u8];
                return v4_global_unicast(&v);
            }
            if g.iter().all(|&x| x == 0) {
                return false;
            }
            if g[7] == 1 && g[..7].iter().all(|&x| x == 0) {
                return false;
            }
            if g[0] & 0xff00 == 0xff00 || g[0] & 0xffc0 == 0xfe80 {
                return false;
            }
            true
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
