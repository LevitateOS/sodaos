//! Live-input metadata shapes (`coreos_stream.go`).
//!
//! Owns the Rust wire records and validation for resolved CoreOS and Fountain
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
    #[serde(rename = "Fountain")]
    pub fountain: crate::payload::FountainRelease,
}

pub fn valid_live_inputs(inputs: &LiveInputs) -> Result<(), Error> {
    valid_resolved_coreos(&inputs.coreos)?;
    inputs
        .fountain
        .validate()
        .map_err(|error| Error(error.to_string()))
}
