use url::{Host, Url};

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

/// Parse the URL forms used by release metadata while keeping raw delimiter
/// presence available to the purpose-specific HTTPS admission below.
pub fn parse_url(raw: &str) -> Option<UrlParts> {
    if raw.bytes().any(|b| b <= 0x20 || b == 0x7f || b == b'\\') || !valid_escapes(raw) {
        return None;
    }
    let (scheme, rest) = raw.split_once("://")?;
    if scheme.is_empty() {
        return None;
    }
    let authority = rest.split(['/', '?', '#']).next()?;
    if authority.is_empty()
        || authority.starts_with('[')
            && authority
                .split_once(']')
                .is_some_and(|(inside, _)| inside.contains('%'))
    {
        return None;
    }
    let parsed = Url::parse(raw).ok()?;
    let host = match parsed.host()? {
        Host::Domain(domain) => domain.to_owned(),
        Host::Ipv4(address) => address.to_string(),
        Host::Ipv6(address) => address.to_string(),
    };
    let query = raw
        .split_once('?')
        .map(|(_, q)| q.split('#').next().unwrap_or(""));
    let fragment = raw.split_once('#').map(|(_, f)| f);
    Some(UrlParts {
        scheme: parsed.scheme().to_string(),
        host: host.clone(),
        hostname: host,
        user: authority.contains('@'),
        raw_query: query.unwrap_or("").to_string(),
        force_query: query == Some(""),
        fragment: fragment.unwrap_or("").to_string(),
    })
}

fn valid_escapes(raw: &str) -> bool {
    let bytes = raw.as_bytes();
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

/// Strict HTTPS metadata shape. The parsed host is used for admission only;
/// signed and retained URL strings stay in their original DTO fields.
pub fn https_url(raw: &str) -> bool {
    let Some(parts) = parse_url(raw) else {
        return false;
    };
    parts.scheme == "https"
        && !parts.hostname.is_empty()
        && !parts.user
        && parts.raw_query.is_empty()
        && !parts.force_query
        && parts.fragment.is_empty()
        && !raw.contains('#')
}

/// WHATWG parses numeric IPv4 aliases before exposing the typed host.
pub fn is_loopback_addr(host: &str) -> bool {
    if host.contains('%') {
        return false;
    }
    let authority = if host.contains(':') && !host.starts_with('[') {
        format!("[{host}]")
    } else {
        host.to_string()
    };
    let Ok(parsed) = Url::parse(&format!("http://{authority}/")) else {
        return false;
    };
    matches!(parsed.host(), Some(Host::Ipv4(ip)) if ip.is_loopback())
        || matches!(parsed.host(), Some(Host::Ipv6(ip)) if ip.is_loopback())
}
