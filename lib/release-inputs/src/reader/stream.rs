//! Live-input metadata shapes (`coreos_stream.go`).
//!
//! Owns the Rust wire records and validation for resolved CoreOS and Tailnet
//! inputs. Upstream stream-document parsing and network resolution stay in
//! the build crate; these records define its validated handoff to consumers.

use super::url::https_url;
use super::{is_digest, non_empty_digits, Error};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct CoreOSImage {
    #[serde(rename = "URL")]
    pub url: String,
    #[serde(rename = "SignatureURL")]
    pub signature_url: String,
    #[serde(rename = "SHA256")]
    pub sha256: String,
    #[serde(rename = "UncompressedSHA256")]
    pub uncompressed_sha256: String,
}

/// `^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$` stream releases.
fn valid_release(release: &str) -> bool {
    let mut parts = release.split('.');
    match (
        parts.next(),
        parts.next(),
        parts.next(),
        parts.next(),
        parts.next(),
    ) {
        (Some(a), Some(b), Some(c), Some(d), None) => {
            non_empty_digits(a) && non_empty_digits(b) && non_empty_digits(c) && non_empty_digits(d)
        }
        _ => false,
    }
}

/// Shape rules for the stream-resolved x86_64 ISO/QEMU triples.
pub fn valid_stream_images(
    release: &str,
    iso: &CoreOSImage,
    qemu: &CoreOSImage,
) -> Result<(), Error> {
    if !valid_release(release) {
        return Err(Error("stable stream release is malformed".to_string()));
    }
    if !https_url(&iso.url)
        || !iso.url.ends_with(".iso")
        || iso.signature_url != format!("{}.sig", iso.url)
        || !is_digest(&iso.sha256)
    {
        return Err(Error(
            "stable stream x86_64 live ISO is malformed".to_string(),
        ));
    }
    if !https_url(&qemu.url)
        || !https_url(&qemu.signature_url)
        || !is_digest(&qemu.sha256)
        || !is_digest(&qemu.uncompressed_sha256)
    {
        return Err(Error(
            "stable stream x86_64 qemu image is malformed".to_string(),
        ));
    }
    Ok(())
}

/// Derives `<stream-prefix>/prod/streams/stable/builds/<release>/release.json`.
pub fn stream_release_url(stream_url: &str, release: &str) -> Result<String, Error> {
    let idx = match stream_url.find("/streams/") {
        Some(i) => i,
        None => return Err(Error("CoreOS stream URL names no stream".to_string())),
    };
    let meta = format!(
        "{}/prod/streams/stable/builds/{release}/release.json",
        &stream_url[..idx]
    );
    if !https_url(&meta) {
        return Err(Error(
            "CoreOS release metadata URL is malformed".to_string(),
        ));
    }
    Ok(meta)
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct TailnetInputs {
    #[serde(rename = "Version")]
    pub version: String,
    #[serde(rename = "SHA256")]
    pub sha256: String,
    #[serde(rename = "Base")]
    pub base: String,
}

/// `^[0-9]+\.[0-9]+\.[0-9]+$` Tailnet releases.
fn valid_tailnet_version(version: &str) -> bool {
    let mut parts = version.split('.');
    match (parts.next(), parts.next(), parts.next(), parts.next()) {
        (Some(a), Some(b), Some(c), None) => {
            non_empty_digits(a) && non_empty_digits(b) && non_empty_digits(c)
        }
        _ => false,
    }
}

/// `^docker\.io/tailscale/alpine-base:[0-9]+\.[0-9]+$` base tags.
fn valid_tailnet_base(base: &str) -> bool {
    let rest = match base.strip_prefix("docker.io/tailscale/alpine-base:") {
        Some(rest) => rest,
        None => return false,
    };
    let mut parts = rest.split('.');
    match (parts.next(), parts.next(), parts.next()) {
        (Some(a), Some(b), None) => non_empty_digits(a) && non_empty_digits(b),
        _ => false,
    }
}

pub fn valid_tailnet_inputs(tailnet: &TailnetInputs) -> Result<(), Error> {
    if !valid_tailnet_version(&tailnet.version) {
        return Err(Error("invalid Tailnet version".to_string()));
    }
    if !is_digest(&tailnet.sha256) {
        return Err(Error("invalid Tailnet archive checksum".to_string()));
    }
    if !valid_tailnet_base(&tailnet.base) {
        return Err(Error("invalid Tailnet base tag".to_string()));
    }
    Ok(())
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ResolvedCoreOS {
    #[serde(rename = "Release")]
    pub release: String,
    #[serde(rename = "MetadataURL")]
    pub metadata_url: String,
    #[serde(rename = "Container")]
    pub container: BTreeMap<String, String>,
    #[serde(rename = "ISO")]
    pub iso: BTreeMap<String, CoreOSImage>,
    #[serde(rename = "QEMU")]
    pub qemu: BTreeMap<String, CoreOSImage>,
}

pub fn valid_resolved_coreos(resolved: &ResolvedCoreOS) -> Result<(), Error> {
    let iso = resolved
        .iso
        .get("x86_64")
        .ok_or_else(|| Error("stable stream x86_64 live ISO is malformed".to_string()))?;
    let qemu = resolved
        .qemu
        .get("x86_64")
        .ok_or_else(|| Error("stable stream x86_64 qemu image is malformed".to_string()))?;
    valid_stream_images(&resolved.release, iso, qemu)?;
    if !https_url(&resolved.metadata_url)
        || !resolved
            .metadata_url
            .ends_with(&format!("/builds/{}/release.json", resolved.release))
    {
        return Err(Error(
            "resolved CoreOS metadata URL is malformed".to_string(),
        ));
    }
    if resolved.container.len() != 1 {
        return Err(Error("x86_64 base digest required".to_string()));
    }
    // Go indexes Container["x86_64"]; a lone entry under another key yields
    // "" and fails the cut below.
    let entry = resolved
        .container
        .get("x86_64")
        .map(String::as_str)
        .unwrap_or("");
    let (host, digest) = match entry.split_once("/fedora/fedora-coreos@sha256:") {
        Some(pair) => pair,
        None => {
            return Err(Error("digest-pinned CoreOS base required".to_string()));
        }
    };
    if host.is_empty() || host.contains('/') || !is_digest(digest) {
        return Err(Error("digest-pinned CoreOS base required".to_string()));
    }
    Ok(())
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct LiveInputs {
    #[serde(rename = "CoreOS")]
    pub coreos: ResolvedCoreOS,
    #[serde(rename = "Tailnet")]
    pub tailnet: TailnetInputs,
}

pub fn valid_live_inputs(inputs: &LiveInputs) -> Result<(), Error> {
    valid_resolved_coreos(&inputs.coreos)?;
    valid_tailnet_inputs(&inputs.tailnet)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn iso() -> CoreOSImage {
        CoreOSImage {
            url: "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/x86_64/fedora-coreos-41.20250101.3.0-live.x86_64.iso".to_string(),
            signature_url: "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/x86_64/fedora-coreos-41.20250101.3.0-live.x86_64.iso.sig".to_string(),
            sha256: "d".repeat(64),
            uncompressed_sha256: String::new(),
        }
    }

    fn qemu() -> CoreOSImage {
        CoreOSImage {
            url: "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/x86_64/fedora-coreos-41.20250101.3.0-qemu.x86_64.qcow2.xz".to_string(),
            signature_url: "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/x86_64/fedora-coreos-41.20250101.3.0-qemu.x86_64.qcow2.xz.sig".to_string(),
            sha256: "e".repeat(64),
            uncompressed_sha256: "f".repeat(64),
        }
    }

    fn resolved() -> ResolvedCoreOS {
        ResolvedCoreOS {
            release: "41.20250101.3.0".to_string(),
            metadata_url: "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/release.json".to_string(),
            container: BTreeMap::from([(
                "x86_64".to_string(),
                format!("quay.io/fedora/fedora-coreos@sha256:{}", "a".repeat(64)),
            )]),
            iso: BTreeMap::from([("x86_64".to_string(), iso())]),
            qemu: BTreeMap::from([("x86_64".to_string(), qemu())]),
        }
    }

    #[test]
    fn stream_images_require_https_triples() {
        assert!(valid_stream_images("41.20250101.3.0", &iso(), &qemu()).is_ok());
        assert_eq!(
            valid_stream_images("41.20250101.3", &iso(), &qemu()).unwrap_err(),
            Error("stable stream release is malformed".to_string())
        );
        let http_iso = CoreOSImage {
            url: "http://example.test/live.iso".to_string(),
            signature_url: "http://example.test/live.iso.sig".to_string(),
            ..iso()
        };
        assert_eq!(
            valid_stream_images("41.20250101.3.0", &http_iso, &qemu()).unwrap_err(),
            Error("stable stream x86_64 live ISO is malformed".to_string())
        );
        let bad_sig = CoreOSImage {
            signature_url: "https://example.test/other.sig".to_string(),
            ..iso()
        };
        assert_eq!(
            valid_stream_images("41.20250101.3.0", &bad_sig, &qemu()).unwrap_err(),
            Error("stable stream x86_64 live ISO is malformed".to_string())
        );
        let bad_qemu = CoreOSImage {
            uncompressed_sha256: "short".to_string(),
            ..qemu()
        };
        assert_eq!(
            valid_stream_images("41.20250101.3.0", &iso(), &bad_qemu).unwrap_err(),
            Error("stable stream x86_64 qemu image is malformed".to_string())
        );
    }

    #[test]
    fn release_url_derives_from_stream_endpoint() {
        assert_eq!(
            stream_release_url(
                "https://builds.coreos.fedoraproject.org/streams/stable.json",
                "41.20250101.3.0"
            )
            .unwrap(),
            "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/release.json"
        );
        assert_eq!(
            stream_release_url("https://example.test/no-stream", "41.20250101.3.0").unwrap_err(),
            Error("CoreOS stream URL names no stream".to_string())
        );
    }

    #[test]
    fn tailnet_pins_require_version_checksum_and_base() {
        let good = TailnetInputs {
            version: "1.86.2".to_string(),
            sha256: "b".repeat(64),
            base: "docker.io/tailscale/alpine-base:3.21".to_string(),
        };
        assert!(valid_tailnet_inputs(&good).is_ok());
        let bad_version = TailnetInputs {
            version: "1.86".to_string(),
            sha256: good.sha256.clone(),
            base: good.base.clone(),
        };
        assert_eq!(
            valid_tailnet_inputs(&bad_version).unwrap_err(),
            Error("invalid Tailnet version".to_string())
        );
        let bad_base = TailnetInputs {
            version: good.version.clone(),
            sha256: good.sha256.clone(),
            base: "docker.io/tailscale/alpine-base:latest".to_string(),
        };
        assert_eq!(
            valid_tailnet_inputs(&bad_base).unwrap_err(),
            Error("invalid Tailnet base tag".to_string())
        );
    }

    #[test]
    fn resolved_coreos_requires_metadata_and_pinned_base() {
        assert!(valid_resolved_coreos(&resolved()).is_ok());
        let mut extra_arch = resolved();
        extra_arch.iso.insert("aarch64".to_string(), iso());
        extra_arch.qemu.insert("aarch64".to_string(), qemu());
        assert!(valid_resolved_coreos(&extra_arch).is_ok());
        let mut bad_meta = resolved();
        bad_meta.metadata_url = "https://example.test/other.json".to_string();
        assert_eq!(
            valid_resolved_coreos(&bad_meta).unwrap_err(),
            Error("resolved CoreOS metadata URL is malformed".to_string())
        );
        let mut two_entries = resolved();
        two_entries.container.insert(
            "aarch64".to_string(),
            "quay.io/fedora/fedora-coreos@sha256:".to_string() + &"a".repeat(64),
        );
        assert_eq!(
            valid_resolved_coreos(&two_entries).unwrap_err(),
            Error("x86_64 base digest required".to_string())
        );
        let mut wrong_key = resolved();
        wrong_key.container.remove("x86_64");
        wrong_key.container.insert(
            "aarch64".to_string(),
            "quay.io/fedora/fedora-coreos@sha256:".to_string() + &"a".repeat(64),
        );
        assert_eq!(
            valid_resolved_coreos(&wrong_key).unwrap_err(),
            Error("digest-pinned CoreOS base required".to_string())
        );
        let mut bad_digest = resolved();
        *bad_digest.container.get_mut("x86_64").unwrap() =
            "quay.io/fedora/fedora-coreos@sha256:short".to_string();
        assert_eq!(
            valid_resolved_coreos(&bad_digest).unwrap_err(),
            Error("digest-pinned CoreOS base required".to_string())
        );
    }
}
