use std::net::Ipv4Addr;
use url::Url;

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
    host.parse::<Ipv4Addr>()
        .is_ok_and(|address| address.is_loopback())
}

/// The `forgejo_url` checks: https scheme
/// (case-insensitive, as urlsplit lowercases it), non-empty host, no
/// userinfo, empty query/fragment, path "" or "/". Returns the display form
/// (`value.rstrip('/')`).
pub(crate) fn valid_origin(value: &str) -> Option<String> {
    if value.bytes().any(|b| b <= 0x20 || b == 0x7f || b == b'\\') {
        return None;
    }
    if !value.contains("://") || !valid_percent_escapes(value) {
        return None;
    }
    let (_, rest) = value.split_once("://")?;
    let authority = rest.split(['/', '?', '#']).next()?;
    if authority.is_empty()
        || authority.starts_with('[')
            && authority
                .split_once(']')
                .is_some_and(|(inside, _)| inside.contains('%'))
    {
        return None;
    }
    let raw_path = rest
        .find('/')
        .map(|at| rest[at..].split(['?', '#']).next().unwrap_or(""))
        .unwrap_or("");
    if !matches!(raw_path, "" | "/") {
        return None;
    }
    let parsed = Url::parse(value).ok()?;
    if parsed.scheme() != "https" || parsed.host_str().is_none_or(str::is_empty) {
        return None;
    }
    // Any '@' means username (possibly empty) is not None.
    if authority.contains('@') {
        return None;
    }
    // Empty query and fragment delimiters are accepted by this caller.
    if parsed.query().is_some_and(|q| !q.is_empty())
        || parsed.fragment().is_some_and(|f| !f.is_empty())
        || !matches!(parsed.path(), "" | "/")
    {
        return None;
    }
    Some(value.trim_end_matches('/').to_string())
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
