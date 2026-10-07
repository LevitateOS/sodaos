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
        assert!(https_url("HTTPS://éxample.test/"));
        assert!(https_url("https://example.test:0/base"));
    }

    #[test]
    fn rejects_unsafe_public_inputs() {
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
            "https://example.test:65536/",
            "https://exa mple.test/",
            "https://example.test\\path",
            "https://example.test/%zz",
            "",
        ] {
            assert!(!https_url(raw), "accepted {raw:?}");
        }
    }

    #[test]
    fn numeric_hosts_use_whatwg_ip_classification_for_admission() {
        assert!(https_url("https://0177.0.0.1/"));
        assert!(https_url("https://2130706433/"));
    }
}
