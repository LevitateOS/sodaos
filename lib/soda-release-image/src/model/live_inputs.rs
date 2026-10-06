use soda_json::JsonValue;

use crate::error::Error;
use crate::jsonio;

use super::{content_get, https_url, is_coreos_release, is_digest, is_dotted_numbers};

// ---------------------------------------------------------------------------
// build CoreOS inputs
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct CoreOSImage {
    pub url: String,
    pub signature_url: String,
    pub sha256: String,
    pub uncompressed_sha256: String,
}

impl CoreOSImage {
    pub fn parse(value: &JsonValue) -> Result<CoreOSImage, Error> {
        jsonio::check_no_unknown(
            value,
            &["URL", "SignatureURL", "SHA256", "UncompressedSHA256"],
        )?;
        Ok(CoreOSImage {
            url: jsonio::require_string(value, "URL")?,
            signature_url: jsonio::require_string(value, "SignatureURL")?,
            sha256: jsonio::require_string(value, "SHA256")?,
            uncompressed_sha256: jsonio::require_string(value, "UncompressedSHA256")?,
        })
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ResolvedCoreOS {
    pub release: String,
    pub metadata_url: String,
    pub container: Vec<(String, String)>,
    pub iso: Vec<(String, CoreOSImage)>,
    pub qemu: Vec<(String, CoreOSImage)>,
}

impl ResolvedCoreOS {
    pub fn parse(value: &JsonValue) -> Result<ResolvedCoreOS, Error> {
        jsonio::check_no_unknown(
            value,
            &["Release", "MetadataURL", "Container", "ISO", "QEMU"],
        )?;
        let images = |field: &str| -> Result<Vec<(String, CoreOSImage)>, Error> {
            let mut out = Vec::new();
            if let JsonValue::Object(entries) = jsonio::require_object(value, field)? {
                for (arch, image) in entries {
                    out.push((arch.clone(), CoreOSImage::parse(image)?));
                }
            }
            Ok(out)
        };
        Ok(ResolvedCoreOS {
            release: jsonio::require_string(value, "Release")?,
            metadata_url: jsonio::require_string(value, "MetadataURL")?,
            container: jsonio::string_map(value, "Container")?,
            iso: images("ISO")?,
            qemu: images("QEMU")?,
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
    pub fn parse(value: &JsonValue) -> Result<TailnetInputs, Error> {
        if matches!(value, JsonValue::Null) {
            return Ok(TailnetInputs::default());
        }
        jsonio::check_no_unknown(value, &["Version", "SHA256", "Base"])?;
        Ok(TailnetInputs {
            version: jsonio::require_string(value, "Version")?,
            sha256: jsonio::require_string(value, "SHA256")?,
            base: jsonio::require_string(value, "Base")?,
        })
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LiveInputs {
    pub core_os: ResolvedCoreOS,
    pub tailnet: TailnetInputs,
}

impl LiveInputs {
    pub fn parse(value: &JsonValue) -> Result<LiveInputs, Error> {
        jsonio::check_no_unknown(value, &["CoreOS", "Tailnet"])?;
        let core_os_value = jsonio::require_object(value, "CoreOS")?;
        let core_os = if matches!(core_os_value, JsonValue::Null) {
            ResolvedCoreOS::default()
        } else {
            ResolvedCoreOS::parse(core_os_value)?
        };
        let tailnet_value = jsonio::require_object(value, "Tailnet")?;
        Ok(LiveInputs {
            core_os,
            tailnet: TailnetInputs::parse(tailnet_value)?,
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
