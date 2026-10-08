//! Live CoreOS stable inputs (`coreos_stream.go`): stream resolution,
//! floating Tailnet toolchain, and the controller-resolved live-inputs
//! file the isolated worker consumes. Pre-release builds resolve the
//! current stable build at build time and record it; no stored version is
//! consulted.

use crate::coreos::https_url;
use crate::files::oci_architecture;
use crate::http::{fetch_capped_json, HttpTransport, UreqTransport};
use crate::Error;
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::collections::BTreeMap;

use crate::coreos_registry::resolve_registry_digests_with;
pub use crate::tailnet_inputs::{resolve_tailnet_inputs, resolve_tailnet_inputs_with};
use soda_build_tools::reader::stream::{CoreOSImage, ResolvedCoreOS};

type RawJson = Box<serde_json::value::RawValue>;

fn decode_raw<T: serde::de::DeserializeOwned>(
    raw: Option<RawJson>,
) -> Result<Option<T>, serde_json::Error> {
    match raw {
        None => Ok(None),
        Some(raw) => serde_json::from_str(raw.get()),
    }
}

struct StreamDocument {
    architectures: Option<RawJson>,
}
struct ArchitectureMap {
    x86_64: Option<RawJson>,
}
struct ArchitectureEntry {
    artifacts: Option<RawJson>,
}
struct ArtifactMap {
    metal: Option<RawJson>,
    qemu: Option<RawJson>,
}
struct Artifact {
    formats: Option<RawJson>,
}
struct FormatMap {
    iso: Option<RawJson>,
    qcow2_xz: Option<RawJson>,
}
struct FormatEntry {
    disk: Option<RawJson>,
}
struct Disk {
    location: Option<RawJson>,
    signature: Option<RawJson>,
    sha256: Option<RawJson>,
    uncompressed_sha256: Option<RawJson>,
}

macro_rules! raw_object {
    ($type:ident, $visitor:ident, $expect:literal, { $($field:ident => $name:literal),+ $(,)? }) => {
        impl<'de> Deserialize<'de> for $type {
            fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
                struct $visitor;
                impl<'de> Visitor<'de> for $visitor {
                    type Value = $type;
                    fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result { f.write_str($expect) }
                    fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                        $(let mut $field: Option<RawJson> = None;)+
                        while let Some(key) = map.next_key::<String>()? {
                            let mut matched = false;
                            $(if key.eq_ignore_ascii_case($name) { $field = Some(map.next_value()?); matched = true; })+
                            if !matched { map.next_value::<de::IgnoredAny>()?; }
                        }
                        Ok($type { $($field),+ })
                    }
                }
                deserializer.deserialize_map($visitor)
            }
        }
    };
}

raw_object!(StreamDocument, StreamDocumentVisitor, "stable stream object", { architectures => "architectures" });
raw_object!(ArchitectureMap, ArchitectureMapVisitor, "architecture map", { x86_64 => "x86_64" });
raw_object!(ArchitectureEntry, ArchitectureEntryVisitor, "architecture entry", { artifacts => "artifacts" });
raw_object!(ArtifactMap, ArtifactMapVisitor, "artifact map", { metal => "metal", qemu => "qemu" });
raw_object!(Artifact, ArtifactVisitor, "artifact object", { formats => "formats" });
raw_object!(FormatMap, FormatMapVisitor, "format map", { iso => "iso", qcow2_xz => "qcow2.xz" });
raw_object!(FormatEntry, FormatEntryVisitor, "format entry", { disk => "disk" });
raw_object!(Disk, DiskVisitor, "disk object", {
    location => "location", signature => "signature", sha256 => "sha256",
    uncompressed_sha256 => "uncompressed-sha256"
});

pub const DEFAULT_COREOS_STREAM_URL: &str =
    "https://builds.coreos.fedoraproject.org/streams/stable.json";
pub const DEFAULT_COREOS_REGISTRY: &str = "https://quay.io";

pub fn coreos_stream_url() -> String {
    let value = std::env::var("SODA_COREOS_STREAM_URL").unwrap_or_default();
    if value.trim().is_empty() {
        DEFAULT_COREOS_STREAM_URL.to_string()
    } else {
        value.trim().to_string()
    }
}

pub fn coreos_registry() -> String {
    let value = std::env::var("SODA_COREOS_REGISTRY").unwrap_or_default();
    let trimmed = value.trim();
    if trimmed.is_empty() {
        DEFAULT_COREOS_REGISTRY.to_string()
    } else {
        trimmed.strip_suffix('/').unwrap_or(trimmed).to_string()
    }
}

pub const DEFAULT_TAILNET_INDEX_URL: &str = "https://pkgs.tailscale.com/stable/";
pub const DEFAULT_TAILNET_BASE_TAGS_URL: &str =
    "https://hub.docker.com/v2/repositories/tailscale/alpine-base/tags";

pub fn tailnet_index_url() -> String {
    let value = std::env::var("SODA_TAILSCALE_INDEX_URL").unwrap_or_default();
    if value.trim().is_empty() {
        DEFAULT_TAILNET_INDEX_URL.to_string()
    } else {
        value.trim().to_string()
    }
}

pub fn tailnet_base_tags_url() -> String {
    let value = std::env::var("SODA_TAILSCALE_BASE_TAGS_URL").unwrap_or_default();
    if value.trim().is_empty() {
        DEFAULT_TAILNET_BASE_TAGS_URL.to_string()
    } else {
        value.trim().to_string()
    }
}

/// Parses one stable-stream document into release + x86_64 ISO/QEMU triples.
fn resolve_stream_build(data: &[u8]) -> Result<(String, CoreOSImage, CoreOSImage), Error> {
    let text = String::from_utf8_lossy(data);
    let doc: StreamDocument =
        serde_json::from_str(&text).map_err(|_| Error::msg("invalid stable stream document"))?;
    let archs = decode_raw::<ArchitectureMap>(doc.architectures)
        .map_err(|_| Error::msg("invalid stable stream document"))?
        .ok_or_else(|| Error::msg("stable stream lacks architecture x86_64"))?;
    let entry = decode_raw::<ArchitectureEntry>(archs.x86_64)
        .map_err(|_| Error::msg("invalid stable stream document"))?
        .ok_or_else(|| Error::msg("stable stream lacks architecture x86_64"))?;
    let artifacts = decode_raw::<ArtifactMap>(entry.artifacts)
        .map_err(|_| Error::msg("invalid stable stream document"))?;
    let disk = |artifact: &str, format: &str| -> Result<CoreOSImage, Error> {
        let missing = if artifact == "metal" {
            "stable stream lacks x86_64 live ISO".to_string()
        } else {
            format!("stable stream lacks x86_64 {artifact} image")
        };
        let no_artifact = || Error::msg(missing.clone());
        let art_raw = artifacts
            .as_ref()
            .and_then(|a| match artifact {
                "metal" => a.metal.as_deref(),
                "qemu" => a.qemu.as_deref(),
                _ => None,
            })
            .ok_or_else(no_artifact)?;
        let art = decode_raw::<Artifact>(Some(art_raw.to_owned()))
            .ok()
            .flatten()
            .ok_or_else(no_artifact)?;
        let formats_raw = art.formats.as_deref().ok_or_else(no_artifact)?;
        let formats = decode_raw::<FormatMap>(Some(formats_raw.to_owned()))
            .ok()
            .flatten()
            .ok_or_else(no_artifact)?;
        let entry_raw = match format {
            "iso" => formats.iso.as_deref(),
            "qcow2.xz" => formats.qcow2_xz.as_deref(),
            _ => None,
        }
        .ok_or_else(no_artifact)?;
        let entry = decode_raw::<FormatEntry>(Some(entry_raw.to_owned()))
            .ok()
            .flatten()
            .ok_or_else(no_artifact)?;
        // A missing disk decodes to the zero triple, which the shape rules
        // refuse downstream, exactly as the Go owner flows.
        let disk = entry
            .disk
            .as_deref()
            .and_then(|raw| decode_raw::<Disk>(Some(raw.to_owned())).ok().flatten());
        let get = |raw: Option<&RawJson>| -> Result<String, Error> {
            match raw {
                None => Ok(String::new()),
                Some(raw) => serde_json::from_str::<Option<String>>(raw.get())
                    .map(Option::unwrap_or_default)
                    .map_err(|_| Error::msg("invalid stable stream document")),
            }
        };
        Ok(CoreOSImage {
            url: get(disk.as_ref().and_then(|d| d.location.as_ref()))?,
            signature_url: get(disk.as_ref().and_then(|d| d.signature.as_ref()))?,
            sha256: get(disk.as_ref().and_then(|d| d.sha256.as_ref()))?,
            uncompressed_sha256: get(disk.as_ref().and_then(|d| d.uncompressed_sha256.as_ref()))?,
        })
    };
    let iso = disk("metal", "iso")?;
    let release = stream_release_from_location(&iso.url)
        .ok_or_else(|| Error::msg("stable stream x86_64 ISO location names no release"))?;
    let qemu = disk("qemu", "qcow2.xz")?;
    soda_build_tools::reader::stream::valid_stream_images(&release, &iso, &qemu)?;
    Ok((release, iso, qemu))
}

/// First `/builds/<a>.<b>.<c>.<d>/` release in the ISO location.
fn stream_release_from_location(location: &str) -> Option<String> {
    let mut search = location;
    while let Some(idx) = search.find("/builds/") {
        let rest = &search[idx + 8..];
        let end = rest.find('/')?;
        let candidate = &rest[..end];
        let mut parts = candidate.split('.');
        let valid = matches!(
            (parts.next(), parts.next(), parts.next(), parts.next(), parts.next()),
            (Some(a), Some(b), Some(c), Some(d), None)
                if !a.is_empty() && !b.is_empty() && !c.is_empty() && !d.is_empty()
                    && a.bytes().all(|b| b.is_ascii_digit())
                    && b.bytes().all(|b| b.is_ascii_digit())
                    && c.bytes().all(|b| b.is_ascii_digit())
                    && d.bytes().all(|b| b.is_ascii_digit())
        );
        if valid {
            return Some(candidate.to_string());
        }
        search = &search[idx + 1..];
    }
    None
}

/// Current stable live ISO for one architecture.
pub fn resolve_coreos_iso(arch: &str) -> Result<(String, CoreOSImage), Error> {
    resolve_coreos_iso_with(&UreqTransport, arch)
}

pub fn resolve_coreos_iso_with<T: HttpTransport>(
    transport: &T,
    arch: &str,
) -> Result<(String, CoreOSImage), Error> {
    oci_architecture(arch)?;
    let resolved = resolve_coreos_with(transport)?;
    match resolved.iso.get(arch) {
        Some(img) => Ok((resolved.release.clone(), img.clone())),
        None => Err(Error::msg(format!("stable stream lacks {arch} live ISO"))),
    }
}

/// Current stable qemu image for one architecture.
pub fn resolve_coreos_qemu(arch: &str) -> Result<(String, CoreOSImage), Error> {
    resolve_coreos_qemu_with(&UreqTransport, arch)
}

pub fn resolve_coreos_qemu_with<T: HttpTransport>(
    transport: &T,
    arch: &str,
) -> Result<(String, CoreOSImage), Error> {
    oci_architecture(arch)?;
    let resolved = resolve_coreos_with(transport)?;
    match resolved.qemu.get(arch) {
        Some(img) => Ok((resolved.release.clone(), img.clone())),
        None => Err(Error::msg(format!("stable stream lacks {arch} qemu image"))),
    }
}

/// Current stable build: release, live-ISO, and qemu locations from the
/// stream, container digests from the registry.
pub fn resolve_coreos() -> Result<ResolvedCoreOS, Error> {
    resolve_coreos_with(&UreqTransport)
}

pub fn resolve_coreos_with<T: HttpTransport>(transport: &T) -> Result<ResolvedCoreOS, Error> {
    let stream_url = coreos_stream_url();
    if !https_url(&stream_url) {
        return Err(Error::msg("CoreOS stream URL must be HTTPS"));
    }
    let data = fetch_capped_json(transport, &stream_url, 8 << 20)?;
    let (release, iso, qemu) = resolve_stream_build(&data)?;
    let digests = resolve_registry_digests_with(transport, &coreos_registry())?;
    let meta: String = soda_build_tools::reader::stream::stream_release_url(&stream_url, &release)?;
    let mut iso_map = BTreeMap::new();
    iso_map.insert("x86_64".to_string(), iso);
    let mut qemu_map = BTreeMap::new();
    qemu_map.insert("x86_64".to_string(), qemu);
    Ok(ResolvedCoreOS {
        release,
        metadata_url: meta,
        container: digests.into_iter().collect(),
        iso: iso_map,
        qemu: qemu_map,
    })
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use crate::http::tests::Stub;

    fn stream_doc() -> String {
        let base = "https://builds.test/prod/streams/stable/builds/44.20260901.1.0/x86_64/fedora-coreos-44.20260901.1.0";
        let a = "a".repeat(64);
        let b = "b".repeat(64);
        let iso_disk = format!(
            "{{\"location\":\"{base}-live.x86_64.iso\",\"sha256\":\"{a}\",\"signature\":\"{base}-live.x86_64.iso.sig\"}}"
        );
        let qemu_disk = format!(
            "{{\"location\":\"{base}-qemu.x86_64.qcow2.xz\",\"sha256\":\"{a}\",\"signature\":\"{base}-qemu.x86_64.qcow2.xz.sig\",\"uncompressed-sha256\":\"{b}\"}}"
        );
        ["{\"architectures\":{\"x86_64\":{\"artifacts\":{\"metal\":{\"formats\":{\"iso\":{\"disk\":",
            &iso_disk,
            "}}},\"qemu\":{\"formats\":{\"qcow2.xz\":{\"disk\":",
            &qemu_disk,
            "}}}}}}}",
        ]
        .concat()
    }

    fn index_doc() -> String {
        format!(
            "{{\"mediaType\":\"application/vnd.oci.image.index.v1+json\",\"manifests\":[{{\"digest\":\"sha256:{}\",\"platform\":{{\"architecture\":\"amd64\"}}}}]}}",
            "c".repeat(64)
        )
    }

    #[test]
    fn oracle_stream_build_parsing() {
        // Oracle: resolveStreamBuild over the Go fixture document.
        let (release, iso, qemu) = resolve_stream_build(stream_doc().as_bytes()).unwrap();
        assert_eq!(release, "44.20260901.1.0");
        assert!(iso.url.ends_with(".iso"));
        assert_eq!(iso.signature_url, format!("{}.sig", iso.url));
        assert_eq!(qemu.uncompressed_sha256, "b".repeat(64));
        assert!(resolve_stream_build(b"{}").is_err());
        assert!(resolve_stream_build(b"not json").is_err());
    }

    #[test]
    fn stream_disk_aliases_validate_only_the_final_source_spelling() {
        let good = stream_doc().replace("\"location\":\"", "\"location\":false,\"Location\":\"");
        assert!(resolve_stream_build(good.as_bytes()).is_ok());

        let bad = stream_doc().replace("\",\"sha256\":", "\",\"LOCATION\":false,\"sha256\":");
        assert_eq!(
            resolve_stream_build(bad.as_bytes()).unwrap_err().message(),
            "invalid stable stream document"
        );
    }

    #[test]
    fn oracle_resolve_coreos_over_stub() {
        // Oracle: TestResolveCoreOSFindsLiveBuild over stub transport.
        let _env = crate::test_env_lock();
        let stub = Stub::new(&[
            ("/streams/stable.json", 200, None, stream_doc().as_bytes()),
            (
                "/v2/fedora/fedora-coreos/manifests/stable",
                200,
                None,
                index_doc().as_bytes(),
            ),
        ]);
        // Point the resolver at the stub via environment (https-shaped).
        std::env::set_var(
            "SODA_COREOS_STREAM_URL",
            "https://stub.test/streams/stable.json",
        );
        std::env::set_var("SODA_COREOS_REGISTRY", "https://stub.test");
        let resolved = resolve_coreos_with(&stub).unwrap();
        std::env::remove_var("SODA_COREOS_STREAM_URL");
        std::env::remove_var("SODA_COREOS_REGISTRY");
        assert_eq!(resolved.release, "44.20260901.1.0");
        assert!(resolved.container["x86_64"].ends_with(&format!("@sha256:{}", "c".repeat(64))));
        assert_eq!(resolved.iso.len(), 1);
        assert_eq!(resolved.qemu.len(), 1);
        // The registry request must ask for an image index.
        let seen = stub.seen_accept.lock().unwrap();
        assert!(seen.contains(&Some("application/vnd.oci.image.index.v1+json".to_string())));
    }
}
