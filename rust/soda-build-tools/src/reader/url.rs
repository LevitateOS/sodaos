//! Strict HTTPS metadata-URL shape (`coreos.go HTTPSURL`).
//!
//! Mirrors Go `net/url` on the inputs this gate admits: an `https` scheme,
//! a non-empty host, no userinfo, no query string, and no fragment. Go
//! lowercases the parsed scheme, so the prefix match is ASCII
//! case-insensitive; everything else is case-sensitive.

/// Splits an authority into its host half, refusing the malformed ports Go's
/// parser would refuse. Returns `None` when the port is non-numeric or the
/// host shape is invalid.
fn split_host_port(authority: &str) -> Option<&str> {
    if let Some(rest) = authority.strip_prefix('[') {
        let end = rest.find(']')?;
        let tail = &rest[end + 1..];
        if tail.is_empty() {
            return Some(&rest[..end]);
        }
        let port = tail.strip_prefix(':')?;
        if !port.is_empty() && !port.bytes().all(|b| b.is_ascii_digit()) {
            return None;
        }
        return Some(&rest[..end]);
    }
    if authority.contains(':') {
        let mut parts = authority.rsplitn(2, ':');
        let port = parts.next().unwrap_or("");
        let host = parts.next().unwrap_or("");
        if host.contains(':') {
            return None;
        }
        if !port.is_empty() && !port.bytes().all(|b| b.is_ascii_digit()) {
            return None;
        }
        return Some(host);
    }
    Some(authority)
}

/// Every `%` escape must carry two hex digits, as Go's parser requires.
fn valid_escapes(s: &str) -> bool {
    let bytes = s.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len()
                || !bytes[i + 1].is_ascii_hexdigit()
                || !bytes[i + 2].is_ascii_hexdigit()
            {
                return false;
            }
            i += 3;
        } else {
            i += 1;
        }
    }
    true
}

pub fn https_url(raw: &str) -> bool {
    // Byte-safe prefix check (D02-F1): `get(..8)` refuses short inputs and
    // `eq_ignore_ascii_case` only matches ASCII bytes, so byte 8 is always
    // a character boundary below. Admissibility is unchanged.
    match raw.as_bytes().get(..8) {
        Some(prefix) if prefix.eq_ignore_ascii_case(b"https://") => {}
        _ => return false,
    }
    let rest = &raw[8..];
    if rest.is_empty() || raw.contains('#') || rest.contains('?') {
        return false;
    }
    if !valid_escapes(raw) {
        return false;
    }
    let authority = rest.split('/').next().unwrap_or("");
    if authority.is_empty() || authority.contains('@') {
        return false;
    }
    if authority
        .bytes()
        .any(|b| b <= 0x20 || b == 0x7f || b == b'/' || b == b'?' || b == b'#')
    {
        return false;
    }
    matches!(split_host_port(authority), Some(host) if !host.is_empty())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn admits_production_metadata_urls() {
        assert!(https_url(
            "https://builds.coreos.fedoraproject.org/streams/stable.json"
        ));
        assert!(https_url("https://quay.io"));
        assert!(https_url(
            "https://quay.io/v2/fedora/fedora-coreos/manifests/stable"
        ));
        assert!(https_url("https://pkgs.tailscale.com/stable/"));
        assert!(https_url(
            "https://hub.docker.com/v2/repositories/tailscale/alpine-base/tags"
        ));
        assert!(https_url("https://example.test:8443/base"));
        assert!(https_url("https://example.test/base%20path"));
    }

    #[test]
    fn rejects_unsafe_public_inputs() {
        // Mirrors TestCoreOSLockAndDownloadBoundaries plus the parser edges.
        for raw in [
            "http://example.test/base",
            "https://user@example.test/base",
            "https://user:pass@example.test/base",
            "https://example.test/base?",
            "https://example.test/base#",
            "https://example.test/base?token=x",
            "https://example.test/base#frag",
            "https://",
            "https:///path",
            "https:example.test",
            "https://example.test:abc/",
            "https://exa mple.test/",
            "https://example.test/%zz",
            "",
        ] {
            assert!(!https_url(raw), "accepted {raw:?}");
        }
    }

    #[test]
    fn scheme_matches_go_case_folding() {
        // Go lowercases the parsed scheme before the comparison.
        assert!(https_url("HTTPS://example.test/base"));
    }

    #[test]
    fn multibyte_prefix_never_panics() {
        // D02-F1: 7 ASCII bytes + a multibyte char pass the byte-length
        // guard while byte 8 lands mid-char; the predicate must refuse,
        // not panic.
        assert!(!https_url("https:/éxample.test/"));
        assert!(!https_url("https:/é"));
        assert!(!https_url("http://é/"));
        // Admissibility unchanged past a valid prefix: unicode hosts keep
        // the verdict the downstream checks already gave them.
        assert!(https_url("https://éxample.test/"));
    }
}
