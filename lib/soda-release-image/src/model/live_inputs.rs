use crate::error::Error;
use crate::jsonio;
use serde::Serialize;

use super::{content_get, https_url, is_coreos_release, is_digest, is_dotted_numbers};

// ---------------------------------------------------------------------------
// build CoreOS inputs
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
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

impl CoreOSImage {
    pub fn parse(text: &str) -> Result<CoreOSImage, Error> {
        jsonio::parse(text)
    }
}

crate::jsonio::case_record!(CoreOSImage, {
    url: String => "URL",
    signature_url: String => "SignatureURL",
    sha256: String => "SHA256",
    uncompressed_sha256: String => "UncompressedSHA256",
});

#[derive(Default)]
struct ResolvedCoreOSWire {
    release: String,
    metadata_url: String,
    container: crate::jsonio::OrderedMap<String>,
    iso: crate::jsonio::OrderedMap<CoreOSImage>,
    qemu: crate::jsonio::OrderedMap<CoreOSImage>,
}

crate::jsonio::case_record!(ResolvedCoreOSWire, {
    release: String => "Release",
    metadata_url: String => "MetadataURL",
    container: crate::jsonio::OrderedMap<String> => "Container",
    iso: crate::jsonio::OrderedMap<CoreOSImage> => "ISO",
    qemu: crate::jsonio::OrderedMap<CoreOSImage> => "QEMU",
});

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ResolvedCoreOS {
    pub release: String,
    pub metadata_url: String,
    pub container: Vec<(String, String)>,
    pub iso: Vec<(String, CoreOSImage)>,
    pub qemu: Vec<(String, CoreOSImage)>,
}

impl ResolvedCoreOS {
    pub fn parse(text: &str) -> Result<ResolvedCoreOS, Error> {
        let wire: ResolvedCoreOSWire = jsonio::parse(text)?;
        Ok(ResolvedCoreOS {
            release: wire.release,
            metadata_url: wire.metadata_url,
            container: wire.container.0,
            iso: wire.iso.0,
            qemu: wire.qemu.0,
        })
    }

    pub fn container_ref(&self, arch: &str) -> String {
        content_get(&self.container, arch)
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TailnetInputs {
    pub version: String,
    pub sha256: String,
    pub base: String,
}

impl TailnetInputs {
    pub fn parse(text: &str) -> Result<TailnetInputs, Error> {
        if text.trim() == "null" {
            return Ok(TailnetInputs::default());
        }
        jsonio::parse(text)
    }
}

crate::jsonio::case_record!(TailnetInputs, {
    version: String => "Version",
    sha256: String => "SHA256",
    base: String => "Base",
});

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LiveInputs {
    pub core_os: ResolvedCoreOS,
    pub tailnet: TailnetInputs,
}

impl LiveInputs {
    pub fn parse(text: &str) -> Result<LiveInputs, Error> {
        jsonio::parse(text)
    }
}

crate::jsonio::case_record!(LiveInputs, {
    core_os: ResolvedCoreOS => "CoreOS",
    tailnet: TailnetInputs => "Tailnet",
});

impl<'de> serde::Deserialize<'de> for ResolvedCoreOS {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let wire = ResolvedCoreOSWire::deserialize(deserializer)?;
        Ok(Self {
            release: wire.release,
            metadata_url: wire.metadata_url,
            container: wire.container.0,
            iso: wire.iso.0,
            qemu: wire.qemu.0,
        })
    }
}

fn find_arch(list: &[(String, CoreOSImage)]) -> Option<&(String, CoreOSImage)> {
    list.iter().find(|(arch, _)| arch == "x86_64")
}

pub fn valid_stream_images(
    release: &str,
    iso: &[(String, CoreOSImage)],
    qemu: &[(String, CoreOSImage)],
) -> Result<(), Error> {
    if !is_coreos_release(release) {
        return Err(Error::msg("stable stream release is malformed"));
    }
    let find = find_arch;
    let (_, image) =
        find(iso).ok_or_else(|| Error::msg("stable stream x86_64 live ISO is malformed"))?;
    if !https_url(&image.url)
        || !image.url.ends_with(".iso")
        || image.signature_url != format!("{}.sig", image.url)
        || !is_digest(&image.sha256)
    {
        return Err(Error::msg("stable stream x86_64 live ISO is malformed"));
    }
    let (_, qemu_image) =
        find(qemu).ok_or_else(|| Error::msg("stable stream x86_64 qemu image is malformed"))?;
    if !https_url(&qemu_image.url)
        || !https_url(&qemu_image.signature_url)
        || !is_digest(&qemu_image.sha256)
        || !is_digest(&qemu_image.uncompressed_sha256)
    {
        return Err(Error::msg("stable stream x86_64 qemu image is malformed"));
    }
    Ok(())
}

pub fn valid_tailnet_inputs(tailnet: &TailnetInputs) -> Result<(), Error> {
    if !is_dotted_numbers(&tailnet.version, 3) {
        return Err(Error::msg("invalid Tailnet version"));
    }
    if !is_digest(&tailnet.sha256) {
        return Err(Error::msg("invalid Tailnet archive checksum"));
    }
    let base_ok = tailnet
        .base
        .strip_prefix("docker.io/tailscale/alpine-base:")
        .is_some_and(|rest| is_dotted_numbers(rest, 2));
    if !base_ok {
        return Err(Error::msg("invalid Tailnet base tag"));
    }
    Ok(())
}

pub fn valid_resolved_core_os(resolved: &ResolvedCoreOS) -> Result<(), Error> {
    valid_stream_images(&resolved.release, &resolved.iso, &resolved.qemu)?;
    if !https_url(&resolved.metadata_url)
        || !resolved
            .metadata_url
            .ends_with(&format!("/builds/{}/release.json", resolved.release))
    {
        return Err(Error::msg("resolved CoreOS metadata URL is malformed"));
    }
    if resolved.container.len() != 1 {
        return Err(Error::msg("x86_64 base digest required"));
    }
    match resolved
        .container_ref("x86_64")
        .split_once("/fedora/fedora-coreos@sha256:")
    {
        Some((host, digest)) if !host.is_empty() && !host.contains('/') && is_digest(digest) => {
            Ok(())
        }
        _ => Err(Error::msg("digest-pinned CoreOS base required")),
    }
}

pub fn valid_live_inputs(inputs: &LiveInputs) -> Result<(), Error> {
    valid_resolved_core_os(&inputs.core_os)?;
    valid_tailnet_inputs(&inputs.tailnet)
}
