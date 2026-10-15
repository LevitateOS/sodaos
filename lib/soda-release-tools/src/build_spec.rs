//! Build request validation (Go `internal/release/image` `build.go` and
//! `assemble.go` request subset) plus `soda-build` dispatch admission.

use url::{Host, Url};

/// Build request populated by the `soda-build` command schema.
#[derive(Debug, Clone, Default)]
pub struct Request {
    pub source: String,
    pub out: String,
    pub arch: String,
    pub repository_prefix: String,
    pub revision: String,
    pub rootfs_base_url: String,
    pub media_authority: String,
    pub live_inputs: String,
    pub forgejo_source: String,
    pub forgejo_revision: String,
    pub development: bool,
    pub target: String,
    pub media_compression: String,
    pub upgrade_from: String,
    pub distribution: String,
    pub release_tag: String,
}

#[derive(Debug, Clone, Default)]
pub struct ImageResult {
    pub revision: String,
    pub architecture: String,
    pub candidate: String,
    pub candidate_sha256: String,
    pub host_manifest: String,
    pub payload_sha256: String,
    pub scope: String,
    pub media: String,
    pub purpose: String,
    pub requested_target: String,
    pub completed_target: String,
    pub media_compression: String,
    pub distribution: String,
    pub release_tag: String,
    pub docker: String,
}

impl Request {
    pub fn wants_media(&self) -> bool {
        crate::pipeline::image_request(self).wants_media()
    }

    pub fn purpose(&self) -> &'static str {
        if self.development {
            "development"
        } else {
            "production"
        }
    }

    pub fn requested_target(&self) -> String {
        if self.development {
            self.target.clone()
        } else {
            "release".to_owned()
        }
    }

    pub fn validate_target(&self) -> Result<(), String> {
        crate::pipeline::image_request(self)
            .validate_target()
            .map_err(|e| e.0)
    }

    pub fn wants_docker(&self) -> bool {
        crate::pipeline::image_request(self).wants_docker()
    }

    pub fn distribution(&self) -> &str {
        if self.distribution.is_empty() {
            "appliance"
        } else {
            &self.distribution
        }
    }

    pub fn completed_target(&self) -> &'static str {
        crate::pipeline::image_request(self).completed_target()
    }
}

/// Public rootfs URL admission uses WHATWG host parsing for loopback policy.
pub fn media_base_url(value: &str) -> Result<(), String> {
    let failed = "explicit public HTTP(S) rootfs base URL required".to_owned();
    let loopback =
        "rootfs base URL must be reachable from the installing machine, not loopback".to_owned();
    if value.bytes().any(|b| b <= 0x20 || b == 0x7f || b == b'\\') || !valid_percent_escapes(value)
    {
        return Err(failed);
    }
    if !(value.starts_with("https://") || value.starts_with("http://")) {
        return Err(failed);
    }
    let rest = value.split_once("://").map(|(_, rest)| rest).unwrap_or("");
    let authority = rest.split(['/', '?', '#']).next().unwrap_or("");
    if authority.is_empty()
        || authority.contains('@')
        || value.contains('#')
        || authority.starts_with('[')
            && authority
                .split_once(']')
                .is_some_and(|(inside, _)| inside.contains('%'))
    {
        return Err(failed);
    }
    let parsed = Url::parse(value).map_err(|_| failed.clone())?;
    if !matches!(parsed.scheme(), "http" | "https")
        || parsed.host_str().is_none_or(str::is_empty)
        || parsed.query().is_some_and(|query| !query.is_empty())
    {
        return Err(failed);
    }
    let host = parsed.host().ok_or_else(|| failed.clone())?;
    let is_loopback = match host {
        Host::Domain(domain) => domain.eq_ignore_ascii_case("localhost"),
        Host::Ipv4(ip) => ip.is_loopback(),
        Host::Ipv6(ip) => ip.is_loopback(),
    };
    if is_loopback {
        return Err(loopback);
    }
    Ok(())
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

#[cfg(test)]
mod tests {
    use super::*;

    fn dev_request(target: &str) -> Request {
        Request {
            development: true,
            target: target.to_owned(),
            rootfs_base_url: if target == "media" {
                "https://example.invalid/rootfs".to_owned()
            } else {
                String::new()
            },
            ..Request::default()
        }
    }

    #[test]
    fn target_validation_matrix() {
        assert!(dev_request("candidate").validate_target().is_ok());
        let mut r = dev_request("candidate");
        r.rootfs_base_url.clear();
        assert!(r.validate_target().is_ok());
        r.rootfs_base_url = "https://example.invalid".to_owned();
        r.media_authority = "x".to_owned();
        assert_eq!(
            r.validate_target().unwrap_err(),
            "candidate target refuses media-only inputs"
        );
        let mut r = dev_request("bogus");
        assert_eq!(
            r.validate_target().unwrap_err(),
            "--development requires --target candidate, media, docker or all"
        );
        // The narrow combined development target: one shared candidate for
        // local media and Docker fixtures, with the v4 tag and a fixture
        // rootfs base. It stays purpose=development.
        let mut r = Request {
            development: true,
            target: "all".to_owned(),
            distribution: "all".to_owned(),
            release_tag: "v9.9.9-dev".to_owned(),
            rootfs_base_url: "https://example.invalid/rootfs".to_owned(),
            ..Request::default()
        };
        assert!(r.validate_target().is_ok());
        assert_eq!(r.purpose(), "development");
        assert_eq!(r.requested_target(), "all");
        assert_eq!(r.completed_target(), "all");
        r.distribution = "appliance".to_owned();
        assert!(r.validate_target().is_err());
        r = Request::default();
        assert_eq!(
            r.validate_target().unwrap_err(),
            "explicit public HTTP(S) rootfs base URL required"
        );
        r.target = "candidate".to_owned();
        assert_eq!(
            r.validate_target().unwrap_err(),
            "--target requires --development"
        );
        let mut r = dev_request("media");
        r.media_compression = "fast".to_owned();
        assert!(r.validate_target().is_ok());
        r.media_compression = "slow".to_owned();
        assert_eq!(
            r.validate_target().unwrap_err(),
            "--media-compression accepts only fast with --development --target media or all"
        );
    }

    #[test]
    fn media_url_boundaries() {
        assert!(media_base_url("https://example.invalid/rootfs").is_ok());
        assert!(media_base_url("https://example.invalid/rootfs?").is_ok());
        assert!(media_base_url("http://192.168.122.1:8080/").is_ok());
        for bad in [
            "",
            "ftp://example.invalid/x",
            "https://",
            "https://host/x?y=1",
            "https://host/x#f",
            "https://user@host/x",
            "https://host:abc/x",
            "HTTPS://host/x",
            "https://ho st/x",
            "https://host\\x",
            "https://host:65536/x",
            "https://host/x?token=x",
            "https://host/x#f",
            "not-a-url",
        ] {
            assert_eq!(
                media_base_url(bad).unwrap_err(),
                "explicit public HTTP(S) rootfs base URL required",
                "{bad}"
            );
        }
        for loopback in [
            "http://localhost:8080/",
            "http://127.0.0.1:8080/x",
            "http://[::1]:8080/x",
            "https://LOCALHOST/",
            "https://0177.0.0.1/",
            "https://2130706433/",
        ] {
            assert_eq!(
                media_base_url(loopback).unwrap_err(),
                "rootfs base URL must be reachable from the installing machine, not loopback",
                "{loopback}"
            );
        }
    }

    #[test]
    fn purpose_and_requested_target() {
        let r = dev_request("media");
        assert_eq!(r.purpose(), "development");
        assert_eq!(r.requested_target(), "media");
        assert!(r.wants_media());
        let plain = Request::default();
        assert_eq!(plain.purpose(), "production");
        assert_eq!(plain.requested_target(), "release");
    }
}
