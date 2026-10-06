// ---------------------------------------------------------------------------
// Minimal URL parser (Go url.Parse subset used by the pipeline checks)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default)]
pub struct UrlParts {
    pub scheme: String,
    pub host: String,
    pub hostname: String,
    pub user: bool,
    pub raw_query: String,
    pub force_query: bool,
    pub fragment: String,
}

/// Parse `scheme://authority/path?query#fragment`. Returns `None` where Go
/// reports a parse error for inputs the pipeline can produce.
pub fn parse_url(raw: &str) -> Option<UrlParts> {
    let (scheme, rest) = raw.split_once(':')?;
    if scheme.is_empty()
        || !scheme
            .bytes()
            .next()
            .is_some_and(|b| b.is_ascii_alphabetic())
        || !scheme
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || b == b'+' || b == b'-' || b == b'.')
    {
        return None;
    }
    let rest = rest.strip_prefix("//")?;
    let (authority, after) = match rest.find(['/', '?', '#']) {
        Some(i) => (&rest[..i], &rest[i..]),
        None => (rest, ""),
    };
    if authority.bytes().any(|b| b <= 0x20 || b == 0x7f) {
        return None;
    }
    let (userinfo, hostport) = match authority.rfind('@') {
        Some(i) => (true, &authority[i + 1..]),
        None => (false, authority),
    };
    let hostname = if let Some(bracketed) = hostport.strip_prefix('[') {
        let (inside, after_bracket) = bracketed.split_once(']')?;
        if !after_bracket.is_empty() {
            let port = after_bracket.strip_prefix(':')?;
            if !port.bytes().all(|b| b.is_ascii_digit()) {
                return None;
            }
        }
        inside.to_string()
    } else {
        match hostport.rfind(':') {
            Some(i) if !hostport[..i].contains(':') => {
                let port = &hostport[i + 1..];
                if !port.bytes().all(|b| b.is_ascii_digit()) {
                    return None;
                }
                hostport[..i].to_string()
            }
            _ => {
                if hostport.contains(':') {
                    return None;
                }
                hostport.to_string()
            }
        }
    };
    let (before_frag, fragment) = match after.split_once('#') {
        Some((before, frag)) => (before, frag.to_string()),
        None => (after, String::new()),
    };
    let (raw_query, force_query) = match before_frag.split_once('?') {
        Some((_, query)) => (query.to_string(), query.is_empty()),
        None => (String::new(), false),
    };
    Some(UrlParts {
        scheme: scheme.to_ascii_lowercase(),
        host: hostport.to_string(),
        hostname,
        user: userinfo,
        raw_query,
        force_query,
        fragment,
    })
}

/// `build.httpsURL`: strict metadata-URL shape.
pub fn https_url(raw: &str) -> bool {
    match parse_url(raw) {
        Some(u) => {
            u.scheme == "https"
                && !u.hostname.is_empty()
                && !u.user
                && u.raw_query.is_empty()
                && !u.force_query
                && u.fragment.is_empty()
                && !raw.contains('#')
        }
        None => false,
    }
}

/// Go `netip.ParseAddr(host).IsLoopback()` for the shapes URLs carry:
/// IPv4 `127/8` and IPv6 `::1`.
pub fn is_loopback_addr(host: &str) -> bool {
    if let Some(v4) = parse_ipv4(host) {
        return v4[0] == 127;
    }
    if let Some(v6) = parse_ipv6(host) {
        return v6 == [0, 0, 0, 0, 0, 0, 0, 1];
    }
    false
}

fn parse_ipv4(host: &str) -> Option<[u8; 4]> {
    let parts: Vec<&str> = host.split('.').collect();
    if parts.len() != 4 {
        return None;
    }
    let mut out = [0u8; 4];
    for (i, part) in parts.iter().enumerate() {
        if part.is_empty() || part.len() > 3 || !part.bytes().all(|b| b.is_ascii_digit()) {
            return None;
        }
        out[i] = part.parse::<u8>().ok()?;
    }
    Some(out)
}

fn parse_ipv6(host: &str) -> Option<[u16; 8]> {
    if !host.contains(':')
        || host
            .bytes()
            .any(|b| !(b.is_ascii_hexdigit() || b == b':' || b == b'.'))
    {
        return None;
    }
    // Expand a possible :: gap; tolerate one embedded IPv4 tail.
    let (head, tail) = match host.split_once("::") {
        Some((head, tail)) => {
            if tail.contains("::") {
                return None;
            }
            (head, Some(tail))
        }
        None => (host, None),
    };
    let mut groups: Vec<u16> = Vec::new();
    let mut tail_groups: Vec<u16> = Vec::new();
    parse_v6_side(head, &mut groups)?;
    if let Some(tail) = tail {
        parse_v6_side(tail, &mut tail_groups)?;
        if groups.len() + tail_groups.len() > 7 {
            return None;
        }
        let zeros = 8 - groups.len() - tail_groups.len();
        groups.extend(std::iter::repeat_n(0, zeros));
        groups.extend(tail_groups);
    } else if groups.len() != 8 {
        return None;
    }
    let mut out = [0u16; 8];
    out.copy_from_slice(&groups);
    Some(out)
}

fn parse_v6_side(side: &str, out: &mut Vec<u16>) -> Option<()> {
    if side.is_empty() {
        return Some(());
    }
    for part in side.split(':') {
        if part.contains('.') {
            let v4 = parse_ipv4(part)?;
            out.push(u16::from_be_bytes([v4[0], v4[1]]));
            out.push(u16::from_be_bytes([v4[2], v4[3]]));
        } else {
            if part.is_empty() || part.len() > 4 {
                return None;
            }
            out.push(u16::from_str_radix(part, 16).ok()?);
        }
    }
    Some(())
}
