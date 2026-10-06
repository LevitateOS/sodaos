//! Go `net/url` `Parse` with Go's exact accept/reject behavior, decoded
//! fields, and error text (`parse %q: %s`).
//!
//! Byte fidelity: Go stores the UNESCAPED host/path/fragment, which may be
//! invalid UTF-8 after `%`-decoding, so those fields are `Vec<u8>`.
//! Bracketed hosts are validated as IP literals through [`crate::netip`]
//! (error: `invalid host: ...`); non-`http(s)` schemes keep Go's lenient
//! last-colon port split while `http`/`https` enforce strict colons.
//! `GODEBUG=urlstrictcolons=0` (rollout escape hatch) is not honored.

use crate::fmtx::go_quote_into;
use crate::netip;

/// Parsed URL: the fields the installer and the x509 SAN parser read.
/// `host`/`path`/`fragment` hold Go's decoded bytes.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Url {
    pub scheme: String,
    pub user_present: bool,
    pub host: Vec<u8>,
    pub path: Vec<u8>,
    pub raw_query: String,
    pub fragment: Vec<u8>,
}

/// Go `*url.Error` with `Op == "parse"`, preformatted exactly as Go renders
/// it: `parse "url": <inner>`. The quoted URL is the pre-fragment input,
/// except fragment errors quote the full raw URL.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ParseError {
    text: String,
}

impl ParseError {
    fn wrap(url: &str, inner: &str) -> ParseError {
        let mut text = String::from("parse ");
        go_quote_into(&mut text, url.as_bytes());
        text.push_str(": ");
        text.push_str(inner);
        ParseError { text }
    }

    /// The exact Go error text.
    pub fn text(&self) -> &str {
        &self.text
    }
}

impl std::fmt::Display for ParseError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.text)
    }
}

impl std::error::Error for ParseError {}

fn has_ctl(bytes: &[u8]) -> bool {
    bytes.iter().any(|b| *b < 0x20 || *b == 0x7f)
}

fn is_hex(b: u8) -> bool {
    b.is_ascii_hexdigit()
}

fn unhex(b: u8) -> u8 {
    9 * (b >> 6) + (b & 15)
}

// Bytes Go's `shouldEscape` keeps raw in host/zone mode (extracted from the
// generated table): everything else literal below 0x80 is refused in hosts.
const HOST_OK: &[u8] =
    b"!$&'()*+,-.0123456789:;<=>ABCDEFGHIJKLMNOPQRSTUVWXYZ[]_abcdefghijklmnopqrstuvwxyz~\"";

fn should_escape_host(b: u8) -> bool {
    !HOST_OK.contains(&b)
}

fn escape_error(bytes: &[u8]) -> String {
    let mut quoted = String::new();
    go_quote_into(&mut quoted, bytes);
    format!("invalid URL escape {quoted}")
}

fn invalid_host_error(byte: &[u8]) -> String {
    let mut quoted = String::new();
    go_quote_into(&mut quoted, byte);
    format!("invalid character {quoted} in host name")
}

fn invalid_port_error(colon_port: &str) -> String {
    let mut quoted = String::new();
    go_quote_into(&mut quoted, colon_port.as_bytes());
    format!("invalid port {quoted} after host")
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum Mode {
    Host,
    Zone,
    Other,
}

/// Go `unescape`: validates `%` sequences (host/zone literal rules included)
/// and returns the decoded bytes. Non-query modes keep `+` literal.
fn unescape(input: &[u8], mode: Mode) -> Result<Vec<u8>, String> {
    let mut saw_percent = false;
    let mut i = 0;
    while i < input.len() {
        match input[i] {
            b'%' => {
                saw_percent = true;
                if i + 2 >= input.len() || !is_hex(input[i + 1]) || !is_hex(input[i + 2]) {
                    let end = (i + 3).min(input.len());
                    return Err(escape_error(&input[i..end]));
                }
                if mode == Mode::Host && unhex(input[i + 1]) < 8 && &input[i..i + 3] != b"%25" {
                    return Err(escape_error(&input[i..i + 3]));
                }
                if mode == Mode::Zone {
                    let v = unhex(input[i + 1]) << 4 | unhex(input[i + 2]);
                    if &input[i..i + 3] != b"%25" && v != b' ' && should_escape_host(v) {
                        return Err(escape_error(&input[i..i + 3]));
                    }
                }
                i += 3;
            }
            b => {
                if (mode == Mode::Host || mode == Mode::Zone) && b < 0x80 && should_escape_host(b) {
                    return Err(invalid_host_error(&input[i..i + 1]));
                }
                i += 1;
            }
        }
    }
    if !saw_percent {
        return Ok(input.to_vec());
    }
    let mut out = Vec::with_capacity(input.len());
    let mut i = 0;
    while i < input.len() {
        match input[i] {
            b'%' => {
                out.push(unhex(input[i + 1]) << 4 | unhex(input[i + 2]));
                i += 3;
            }
            b => {
                out.push(b);
                i += 1;
            }
        }
    }
    Ok(out)
}

/// Go `validOptionalPort`: empty, or `:` followed by zero or more digits
/// (a bare `:` is a valid empty port).
fn valid_optional_port(port: &[u8]) -> bool {
    if port.is_empty() {
        return true;
    }
    if port[0] != b':' {
        return false;
    }
    port[1..].iter().all(|b| b.is_ascii_digit())
}

/// Go `parseHost`: bracketed IP literals (with `%25` zone handling and
/// `netip` validation, rejecting plain IPv4) or plain hosts with scheme
/// dependent colon strictness. Returns the UNESCAPED host.
fn parse_host(scheme: &str, host: &str) -> Result<Vec<u8>, String> {
    if let Some(open) = host.rfind('[') {
        if open > 0 {
            return Err(String::from("invalid IP-literal"));
        }
        let close = host
            .rfind(']')
            .ok_or_else(|| String::from("missing ']' in host"))?;
        let colon_port = &host[close + 1..];
        if !valid_optional_port(colon_port.as_bytes()) {
            return Err(invalid_port_error(colon_port));
        }
        let unescaped_port = unescape(colon_port.as_bytes(), Mode::Host)?;
        let hostname = &host[1..close];
        let unescaped_host = match hostname.find("%25") {
            Some(zone_idx) => {
                let host_part = unescape(&hostname.as_bytes()[..zone_idx], Mode::Host)?;
                let zone_part = unescape(&hostname.as_bytes()[zone_idx..], Mode::Zone)?;
                [host_part, zone_part].concat()
            }
            None => unescape(hostname.as_bytes(), Mode::Host)?,
        };
        let addr = netip::parse_addr_bytes(&unescaped_host)
            .map_err(|err| format!("invalid host: {}", err.text()))?;
        if addr.is4() {
            return Err(String::from("invalid IP-literal"));
        }
        let mut out = Vec::with_capacity(unescaped_host.len() + 2);
        out.push(b'[');
        out.extend_from_slice(&unescaped_host);
        out.push(b']');
        out.extend_from_slice(&unescaped_port);
        return Ok(out);
    }
    if let Some(first) = host.find(':') {
        let last = host.rfind(':').unwrap_or(first);
        let mut cut = first;
        if last != first && scheme != "http" && scheme != "https" {
            cut = last;
        }
        let colon_port = &host[cut..];
        if !valid_optional_port(colon_port.as_bytes()) {
            return Err(invalid_port_error(colon_port));
        }
    }
    unescape(host.as_bytes(), Mode::Host)
}

/// Go `validUserinfo`: alphanumerics plus `-._:~!$&'()*+,;=%@` (runes;
/// non-ASCII fails).
fn valid_userinfo(userinfo: &str) -> bool {
    userinfo.chars().all(|c| {
        c.is_ascii_alphanumeric()
            || matches!(
                c,
                '-' | '.'
                    | '_'
                    | ':'
                    | '~'
                    | '!'
                    | '$'
                    | '&'
                    | '\''
                    | '('
                    | ')'
                    | '*'
                    | '+'
                    | ','
                    | ';'
                    | '='
                    | '%'
                    | '@'
            )
    })
}

/// Go `parseAuthority`: the host parses FIRST (its errors win), then the
/// userinfo charset and escapes validate. Only userinfo presence is stored.
fn parse_authority(scheme: &str, authority: &str) -> Result<(bool, Vec<u8>), String> {
    let at = authority.rfind('@');
    let host_part = match at {
        Some(i) => &authority[i + 1..],
        None => authority,
    };
    let host = parse_host(scheme, host_part)?;
    let Some(i) = at else {
        return Ok((false, host));
    };
    let userinfo = &authority[..i];
    if !valid_userinfo(userinfo) {
        return Err(String::from("net/url: invalid userinfo"));
    }
    match userinfo.find(':') {
        None => {
            unescape(userinfo.as_bytes(), Mode::Other)?;
        }
        Some(c) => {
            unescape(userinfo.as_bytes()[..c].as_ref(), Mode::Other)?;
            unescape(userinfo.as_bytes()[c + 1..].as_ref(), Mode::Other)?;
        }
    }
    Ok((true, host))
}

/// Go `getScheme`, byte-based: a leading `:` is `missing protocol scheme`;
/// anything else unscannable means no scheme.
fn get_scheme(rest: &str) -> Result<(String, &str), String> {
    let bytes = rest.as_bytes();
    for (i, b) in bytes.iter().enumerate() {
        match b {
            b'a'..=b'z' | b'A'..=b'Z' => {}
            b'0'..=b'9' | b'+' | b'-' | b'.' => {
                if i == 0 {
                    return Ok((String::new(), rest));
                }
            }
            b':' => {
                if i == 0 {
                    return Err(String::from("missing protocol scheme"));
                }
                return Ok((rest[..i].to_lowercase(), &rest[i + 1..]));
            }
            _ => break,
        }
    }
    Ok((String::new(), rest))
}

/// Go `url.Parse`: fragments cut at the first `#` (fragment bytes are NOT
/// control-screened); inner errors quote the pre-fragment input while
/// fragment errors quote the full raw URL.
pub fn parse(raw: &str) -> Result<Url, ParseError> {
    let (u, fragment) = match raw.find('#') {
        Some(i) => (&raw[..i], &raw[i + 1..]),
        None => (raw, ""),
    };
    let mut url = parse_inner(u).map_err(|inner| ParseError::wrap(u, &inner))?;
    if !fragment.is_empty() {
        url.fragment = unescape(fragment.as_bytes(), Mode::Other)
            .map_err(|inner| ParseError::wrap(raw, &inner))?;
    }
    Ok(url)
}

fn parse_inner(u: &str) -> Result<Url, String> {
    if has_ctl(u.as_bytes()) {
        return Err(String::from("net/url: invalid control character in URL"));
    }
    let (scheme, rest) = get_scheme(u)?;
    let (rest, query) = match rest.find('?') {
        Some(i) => (&rest[..i], &rest[i + 1..]),
        None => (rest, ""),
    };
    let mut url = Url {
        scheme,
        user_present: false,
        host: Vec::new(),
        path: Vec::new(),
        raw_query: query.to_string(),
        fragment: Vec::new(),
    };
    if !rest.starts_with('/') {
        if !url.scheme.is_empty() {
            // Rootless paths with a scheme are opaque: Path stays empty.
            return Ok(url);
        }
        let first = rest.split('/').next().unwrap_or("");
        if first.contains(':') {
            return Err(String::from(
                "first path segment in URL cannot contain colon",
            ));
        }
    }
    if (!url.scheme.is_empty() || !rest.starts_with("///")) && rest.starts_with("//") {
        let after = &rest[2..];
        let end = after.find('/').unwrap_or(after.len());
        let (user_present, host) = parse_authority(&url.scheme, &after[..end])?;
        url.user_present = user_present;
        url.host = host;
        url.path = unescape(&after.as_bytes()[end..], Mode::Other)?;
        return Ok(url);
    }
    url.path = unescape(rest.as_bytes(), Mode::Other)?;
    Ok(url)
}

impl Url {
    /// Go `URL.Hostname` via `splitHostPort`: strip a valid optional port at
    /// the last colon, then one pair of surrounding brackets.
    pub fn hostname(&self) -> Vec<u8> {
        let mut host = self.host.as_slice();
        if let Some(colon) = host.iter().rposition(|b| *b == b':') {
            if valid_optional_port(&host[colon..]) {
                host = &host[..colon];
            }
        }
        if host.len() >= 2 && host.starts_with(b"[") && host.ends_with(b"]") {
            host = &host[1..host.len() - 1];
        }
        host.to_vec()
    }
}

#[cfg(test)]
mod tests;
