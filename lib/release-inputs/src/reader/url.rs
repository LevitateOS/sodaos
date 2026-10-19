//! Strict HTTPS metadata URL admission. Callers retain the original literal
//! in signed and locked records; parsing here is only for connection policy.

use url::Url;

pub fn https_url(raw: &str) -> bool {
    if !raw
        .as_bytes()
        .get(..8)
        .is_some_and(|prefix| prefix.eq_ignore_ascii_case(b"https://"))
        || raw.bytes().any(|b| b <= 0x20 || b == 0x7f || b == b'\\')
        || raw.contains(['?', '#'])
        || !valid_escapes(raw)
    {
        return false;
    }
    let authority = raw[8..].split('/').next().unwrap_or("");
    if authority.is_empty()
        || authority.contains('@')
        || authority.starts_with('[')
            && authority
                .split_once(']')
                .is_some_and(|(inside, _)| inside.contains('%'))
    {
        return false;
    }
    let Ok(parsed) = Url::parse(raw) else {
        return false;
    };
    parsed.scheme() == "https" && parsed.host_str().is_some_and(|host| !host.is_empty())
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
