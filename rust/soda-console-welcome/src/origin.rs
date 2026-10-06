/// `listen` of "host:port" with a loopback IP literal, exactly as the
/// Python validates it: one colon, numeric port 1..=65535, IPv4 127/8.
/// (IPv6 can never pass: any IPv6 form holds more than one colon.)
pub(crate) fn valid_listen(listen: &str) -> Option<(String, String)> {
    if listen.bytes().filter(|b| *b == b':').count() != 1 {
        return None;
    }
    let (host, port) = listen.rsplit_once(':')?;
    // Python's isdigit()+int() also accept non-ASCII decimal digits; only
    // ASCII is reachable in real configs, and only ASCII is matched here.
    if port.is_empty() || !port.bytes().all(|b| b.is_ascii_digit()) {
        return None;
    }
    let number: u32 = port.parse().ok()?;
    if !(1..=65535).contains(&number) {
        return None;
    }
    if !is_loopback_ipv4(host) {
        return None;
    }
    Some((host.to_string(), port.to_string()))
}

/// `ipaddress.ip_address(host).is_loopback` for the reachable shape: modern
/// Python requires 4 dot-decimal parts with no leading zeros.
fn is_loopback_ipv4(host: &str) -> bool {
    let parts: Vec<&str> = host.split('.').collect();
    if parts.len() != 4 {
        return false;
    }
    for (i, part) in parts.iter().enumerate() {
        if part.is_empty() || part.len() > 3 || !part.bytes().all(|b| b.is_ascii_digit()) {
            return false;
        }
        if part.len() > 1 && part.starts_with('0') {
            return false;
        }
        let octet: u32 = part.parse().unwrap_or(256);
        if octet > 255 {
            return false;
        }
        if i == 0 && octet != 127 {
            return false;
        }
    }
    true
}

/// The `forgejo_url` checks, replicating urlsplit semantics: https scheme
/// (case-insensitive, as urlsplit lowercases it), non-empty host, no
/// userinfo, empty query/fragment, path "" or "/". Returns the display form
/// (`value.rstrip('/')`).
pub(crate) fn valid_origin(value: &str) -> Option<String> {
    if value
        .chars()
        .any(|c| c.is_whitespace() || c < '\u{20}' || c == '\u{7f}')
    {
        return None;
    }
    let (scheme, rest) = value.split_once(':')?;
    if !scheme.eq_ignore_ascii_case("https") {
        return None;
    }
    let after = rest.strip_prefix("//")?;
    // Authority runs to the first '/', '?' or '#'.
    let end = after.find(['/', '?', '#']).unwrap_or(after.len());
    let (authority, remainder) = after.split_at(end);
    // Any '@' means username (possibly empty) is not None.
    if authority.contains('@') {
        return None;
    }
    let (host, port) = split_host_port(authority)?;
    if host.is_empty() {
        return None;
    }
    // `_ = url.port`: malformed or out-of-range ports raise.
    if let Some(port) = port {
        if !valid_url_port(port) {
            return None;
        }
    }
    // Query and fragment must be empty (a bare '?' or '#' still passes).
    let (before_frag, fragment) = match remainder.split_once('#') {
        Some((before, after)) => (before, after),
        None => (remainder, ""),
    };
    if !fragment.is_empty() {
        return None;
    }
    let (path, query) = match before_frag.split_once('?') {
        Some((before, after)) => (before, after),
        None => (before_frag, ""),
    };
    if !query.is_empty() || (!path.is_empty() && path != "/") {
        return None;
    }
    Some(value.trim_end_matches('/').to_string())
}

fn split_host_port(authority: &str) -> Option<(&str, Option<&str>)> {
    if let Some(bracketed) = authority.strip_prefix('[') {
        let (host, rest) = bracketed.split_once(']')?;
        match rest.strip_prefix(':') {
            Some(port) => Some((host, Some(port))),
            None if rest.is_empty() => Some((host, None)),
            _ => None,
        }
    } else {
        match authority.split_once(':') {
            Some((host, port)) => {
                if port.contains(':') {
                    return None;
                }
                Some((host, Some(port)))
            }
            None => Some((authority, None)),
        }
    }
}

/// `url.port`: ASCII digits only (Python rejects signs, spaces and
/// non-ASCII digits with "Port could not be cast"), 0..=65535. Empty port
/// (from "host:") is None upstream and never reaches here.
fn valid_url_port(port: &str) -> bool {
    if port.is_empty() || !port.bytes().all(|b| b.is_ascii_digit()) {
        return false;
    }
    port.parse::<u32>().is_ok_and(|number| number <= 65535)
}
