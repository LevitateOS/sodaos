//! Build request validation (Go `internal/release/image` `build.go` and
//! `assemble.go` request subset) plus `soda-build` dispatch admission.

/// Build request: the same fields the Go flag set fills.
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
}

impl Request {
    pub fn wants_media(&self) -> bool {
        self.target != "candidate"
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

    fn validate_development_target(&self) -> Result<(), String> {
        if self.development {
            if self.target != "candidate" && self.target != "media" {
                return Err("--development requires --target candidate or media".to_owned());
            }
            return Ok(());
        }
        if !self.target.is_empty() {
            return Err("--target requires --development".to_owned());
        }
        Ok(())
    }

    fn validate_media_inputs(&self) -> Result<(), String> {
        if !self.media_compression.is_empty()
            && (!self.development || self.target != "media" || self.media_compression != "fast")
        {
            return Err(
                "--media-compression accepts only fast with --development --target media"
                    .to_owned(),
            );
        }
        if !self.wants_media() {
            if !self.rootfs_base_url.is_empty() || !self.media_authority.is_empty() {
                return Err("candidate target refuses media-only inputs".to_owned());
            }
            return Ok(());
        }
        media_base_url(&self.rootfs_base_url)
    }

    pub fn validate_target(&self) -> Result<(), String> {
        self.validate_development_target()?;
        self.validate_media_inputs()
    }
}

/// Minimal `url.Parse` equivalent for the rootfs admission rule: the Go
/// owner rejects parse failures, missing hosts, non-HTTP(S) schemes,
/// userinfo, queries, fragments, whitespace, and loopback hosts.
pub fn media_base_url(value: &str) -> Result<(), String> {
    let failed = "explicit public HTTP(S) rootfs base URL required".to_owned();
    let loopback =
        "rootfs base URL must be reachable from the installing machine, not loopback".to_owned();
    if value.chars().any(|c| c == '\r' || c == '\n' || c == ' ') {
        return Err(failed);
    }
    let rest = value
        .strip_prefix("https://")
        .or_else(|| value.strip_prefix("http://"))
        .ok_or_else(|| failed.clone())?;
    if rest.is_empty() {
        return Err(failed);
    }
    let authority = rest.split('/').next().unwrap_or_default();
    let authority = authority.split(['?', '#']).next().unwrap_or_default();
    if authority.is_empty() || rest.contains(['?', '#']) {
        return Err(failed);
    }
    if authority.contains('@') {
        return Err(failed);
    }
    let (host, port, bracketed) = if let Some(rest) = authority.strip_prefix('[') {
        let Some((inside, after)) = rest.split_once(']') else {
            return Err(failed);
        };
        if !after.is_empty() && !after.starts_with(':') {
            return Err(failed);
        }
        (inside, after.strip_prefix(':').unwrap_or(""), true)
    } else {
        if authority.contains(']') {
            return Err(failed);
        }
        match authority.rsplit_once(':') {
            Some((h, p)) => (h, p, false),
            None => (authority, "", false),
        }
    };
    if host.is_empty() || host.contains('@') || (!bracketed && host.contains(':')) {
        return Err(failed);
    }
    if !port.is_empty() && !port.bytes().all(|b| b.is_ascii_digit()) {
        return Err(failed);
    }
    if host.eq_ignore_ascii_case("localhost") || is_loopback_ip(host) {
        return Err(loopback);
    }
    Ok(())
}

fn is_loopback_ip(host: &str) -> bool {
    if host == "::1" {
        return true;
    }
    let mut parts = host.split('.');
    let first = parts.next();
    if first != Some("127") {
        return false;
    }
    let mut count = 1;
    for part in parts {
        count += 1;
        if part.is_empty() || part.len() > 3 || !part.bytes().all(|b| b.is_ascii_digit()) {
            return false;
        }
    }
    count == 4
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
            "--development requires --target candidate or media"
        );
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
            "--media-compression accepts only fast with --development --target media"
        );
    }

    #[test]
    fn media_url_boundaries() {
        assert!(media_base_url("https://example.invalid/rootfs").is_ok());
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
