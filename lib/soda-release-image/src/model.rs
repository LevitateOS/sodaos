//! Exact mirrors of the foreign `release/build` + `release/deliver` model
//! shapes the pipeline decodes, validates, and emits: payloads, candidates,
//! trust, permits, images, CoreOS inputs. JSON field names and emission
//! order match the Go owners; validators return the same messages.

use soda_json::JsonValue;

use crate::error::Error;
use crate::jsonio;
use crate::sys;

mod candidate;
mod images;
mod payload;
mod trust;
mod url;

pub use candidate::{valid_candidate_content, Candidate, ForgejoToolchain};
pub use images::{Image, ProducedImage};
pub use payload::{Payload, PayloadImage};
pub use trust::{Permit, SecretFiles, Trust};
pub use url::{https_url, is_loopback_addr, parse_url, UrlParts};

pub const DELIVER_PATH: &str = "/usr/share/soda/release.json";
pub const DELIVER_IMAGES_PATH: &str = "/usr/share/soda/images";
/// `deliver.Names`, in order.
pub const NAMES: [&str; 6] = [
    "dashboard",
    "forgejo",
    "extension",
    "proxy",
    "project-os",
    "tailnet",
];
pub const FORGEJO_COMPILER_IMAGE: &str =
    "docker.io/library/golang@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468";
/// `store.SchemaVersion()`.
pub const SCHEMA_VERSION: i64 = 26;
pub const SODA_SOURCE: &str = "https://github.com/LevitateOS/sodaos";

pub fn is_revision(s: &str) -> bool {
    soda_build_tools::reader::is_revision(s)
}

pub fn is_digest(s: &str) -> bool {
    soda_build_tools::reader::is_digest(s)
}

pub fn oci_architecture(arch: &str) -> Result<&'static str, Error> {
    soda_build_tools::reader::oci_architecture(arch).map_err(|e| Error::msg(e.to_string()))
}

/// `deliver.Digest`: `sha256:`-prefixed hex digest.
pub fn prefixed_digest(s: &str) -> bool {
    match s.strip_prefix("sha256:") {
        Some(hex) => is_digest(hex),
        None => false,
    }
}

/// `deliver.Hash`: `sha256:`-prefixed hex of bytes.
pub fn deliver_hash(data: &[u8]) -> String {
    format!("sha256:{}", sys::hex_sha256(data))
}

fn is_dotted_numbers(value: &str, parts: usize) -> bool {
    let pieces: Vec<&str> = value.split('.').collect();
    pieces.len() == parts
        && pieces
            .iter()
            .all(|p| !p.is_empty() && p.bytes().all(|b| b.is_ascii_digit()))
}

pub fn is_coreos_release(value: &str) -> bool {
    is_dotted_numbers(value, 4)
}

pub fn valid_repository_prefix(s: &str) -> bool {
    if s.len() >= 200 {
        return false;
    }
    let rest = match s.strip_prefix("ghcr.io/") {
        Some(rest) => rest,
        None => return false,
    };
    let (owner, repo) = match rest.split_once('/') {
        Some(pair) => pair,
        None => return false,
    };
    if repo.contains('/') {
        return false;
    }
    let owner_ok = !owner.is_empty()
        && owner
            .bytes()
            .next()
            .is_some_and(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        && owner
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-');
    let repo_ok = !repo.is_empty()
        && repo
            .bytes()
            .next()
            .is_some_and(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        && repo.bytes().all(|b| {
            b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'.' || b == b'_' || b == b'-'
        });
    owner_ok && repo_ok
}

fn content_get(files: &[(String, String)], name: &str) -> String {
    files
        .iter()
        .find(|(k, _)| k == name)
        .map(|(_, v)| v.clone())
        .unwrap_or_default()
}

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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_prefix_and_digest_shapes() {
        // Oracle: Go ValidRepositoryPrefix + Digest vectors.
        assert!(valid_repository_prefix("ghcr.io/example/sodaos"));
        assert!(!valid_repository_prefix("ghcr.io/Example/sodaos"));
        assert!(!valid_repository_prefix("docker.io/example/sodaos"));
        assert!(!valid_repository_prefix("ghcr.io/example"));
        assert!(prefixed_digest(&format!("sha256:{}", "a".repeat(64))));
        assert!(!prefixed_digest(&"a".repeat(64)));
    }

    #[test]
    fn oracle_url_parser_matches_go_probe() {
        // Oracle: /tmp/urlprobe vectors from the real Go url.Parse.
        let upper = parse_url("HTTPS://Example.COM:8080/p").unwrap();
        assert_eq!(upper.scheme, "https");
        assert_eq!(upper.hostname, "Example.COM");
        assert!(parse_url("https://h:bad/p").is_none());
        assert_eq!(parse_url("https:///p").unwrap().host, "");
        assert!(parse_url("https://h p/").is_none());
        assert!(parse_url("https://[::1]/p").unwrap().hostname == "::1");
        assert!(https_url(
            "https://example.invalid/builds/1.0.0.0/release.json"
        ));
        assert!(!https_url("http://example.invalid/x"));
        assert!(!https_url("https://example.invalid/x?"));
        assert!(is_loopback_addr("127.0.0.1"));
        assert!(is_loopback_addr("::1"));
        assert!(is_loopback_addr("0:0:0:0:0:0:0:1"));
        assert!(!is_loopback_addr("10.0.0.1"));
        assert!(!is_loopback_addr("::ffff:127.0.0.1"));
        assert!(!is_loopback_addr("example.invalid"));
    }

    #[test]
    fn oracle_payload_identity_validation() {
        // Oracle: Go Payload.Validate error strings.
        let payload = Payload::default();
        assert_eq!(
            payload.validate().unwrap_err().0,
            "invalid appliance payload identity"
        );
    }
}
