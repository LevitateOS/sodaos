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
    pub live_inputs: String,
    pub forgejo_source: String,
    pub forgejo_revision: String,
    pub development: bool,
    pub target: String,
    pub media_compression: String,
    pub upgrade_from: String,
    pub distribution: String,
    pub release_tag: String,
    pub package_base_url: String,
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

fn check_url_prefix(value: &str, failed: &str) -> Result<(), String> {
    if value.bytes().any(|b| b <= 0x20 || b == 0x7f || b == b'\\') || !valid_percent_escapes(value)
    {
        return Err(failed.to_owned());
    }
    if !(value.starts_with("https://") || value.starts_with("http://")) {
        return Err(failed.to_owned());
    }
    Ok(())
}

fn check_url_authority(value: &str, failed: &str) -> Result<(), String> {
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
        return Err(failed.to_owned());
    }
    Ok(())
}

fn check_url_host(value: &str, failed: &str, loopback: &str) -> Result<(), String> {
    let parsed = Url::parse(value).map_err(|_| failed.to_owned())?;
    if !matches!(parsed.scheme(), "http" | "https")
        || parsed.host_str().is_none_or(str::is_empty)
        || parsed.query().is_some_and(|query| !query.is_empty())
    {
        return Err(failed.to_owned());
    }
    let host = parsed.host().ok_or_else(|| failed.to_owned())?;
    let is_loopback = match host {
        Host::Domain(domain) => domain.eq_ignore_ascii_case("localhost"),
        Host::Ipv4(ip) => ip.is_loopback(),
        Host::Ipv6(ip) => ip.is_loopback(),
    };
    if is_loopback {
        return Err(loopback.to_owned());
    }
    Ok(())
}

/// Public rootfs URL admission uses WHATWG host parsing for loopback policy.
pub fn media_base_url(value: &str) -> Result<(), String> {
    let failed = "explicit public HTTP(S) rootfs base URL required";
    let loopback = "rootfs base URL must be reachable from the installing machine, not loopback";
    check_url_prefix(value, failed)?;
    check_url_authority(value, failed)?;
    check_url_host(value, failed, loopback)
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
