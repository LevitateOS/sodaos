//! Live CoreOS stable inputs (`coreos_stream.go`): stream resolution,
//! floating Tailnet toolchain, and the controller-resolved live-inputs
//! file the isolated worker consumes. Pre-release builds resolve the
//! current stable build at build time and record it; no stored version is
//! consulted.

use crate::coreos::{https_url, CoreOSImage};
use crate::files::{is_digest, oci_architecture, write_new};
use crate::http::{get_follow, HttpTransport, UreqTransport};
use crate::json_go::{marshal_indent, Emit, Fields, Strict};
use crate::json_input::read_json;
use crate::Error;
use soda_json::JsonValue;
use std::collections::HashMap;
use std::io::Read;
use std::path::Path;
use std::time::Duration;

pub const DEFAULT_COREOS_STREAM_URL: &str =
    "https://builds.coreos.fedoraproject.org/streams/stable.json";
pub const DEFAULT_COREOS_REGISTRY: &str = "https://quay.io";
const COREOS_CONTAINER_REPO: &str = "fedora/fedora-coreos";
const COREOS_CONTAINER_TAG: &str = "stable";

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

/// Floating Tailnet toolchain for one attempt.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TailnetInputs {
    pub version: String,
    pub sha256: String,
    pub base: String,
}

impl TailnetInputs {
    pub fn emit(&self) -> Emit {
        Emit::Object(vec![
            ("Version".to_string(), Emit::Str(self.version.clone())),
            ("SHA256".to_string(), Emit::Str(self.sha256.clone())),
            ("Base".to_string(), Emit::Str(self.base.clone())),
        ])
    }

    pub fn decode(binder: &mut Strict<'_>) -> Result<TailnetInputs, String> {
        Ok(TailnetInputs {
            version: binder.string("Version")?,
            sha256: binder.string("SHA256")?,
            base: binder.string("Base")?,
        })
    }
}

/// One stable build as found right now.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ResolvedCoreOS {
    pub release: String,
    pub metadata_url: String,
    pub container: HashMap<String, String>,
    pub iso: HashMap<String, CoreOSImage>,
    pub qemu: HashMap<String, CoreOSImage>,
}

impl ResolvedCoreOS {
    pub fn emit(&self) -> Emit {
        let mut container: Vec<(String, Emit)> = self
            .container
            .iter()
            .map(|(k, v)| (k.clone(), Emit::Str(v.clone())))
            .collect();
        container.sort_by(|a, b| a.0.cmp(&b.0));
        let mut iso: Vec<(String, Emit)> = self
            .iso
            .iter()
            .map(|(k, v)| (k.clone(), v.emit()))
            .collect();
        iso.sort_by(|a, b| a.0.cmp(&b.0));
        let mut qemu: Vec<(String, Emit)> = self
            .qemu
            .iter()
            .map(|(k, v)| (k.clone(), v.emit()))
            .collect();
        qemu.sort_by(|a, b| a.0.cmp(&b.0));
        Emit::Object(vec![
            ("Release".to_string(), Emit::Str(self.release.clone())),
            (
                "MetadataURL".to_string(),
                Emit::Str(self.metadata_url.clone()),
            ),
            ("Container".to_string(), Emit::Object(container)),
            ("ISO".to_string(), Emit::Object(iso)),
            ("QEMU".to_string(), Emit::Object(qemu)),
        ])
    }

    pub fn decode(binder: &mut Strict<'_>) -> Result<ResolvedCoreOS, String> {
        let container: Vec<(String, String)> = binder.string_map("Container")?;
        let iso = binder.object_map("ISO", "build.CoreOSImage", decode_coreos_image)?;
        let qemu = binder.object_map("QEMU", "build.CoreOSImage", decode_coreos_image)?;
        Ok(ResolvedCoreOS {
            release: binder.string("Release")?,
            metadata_url: binder.string("MetadataURL")?,
            container: container.into_iter().collect(),
            iso: iso.into_iter().collect(),
            qemu: qemu.into_iter().collect(),
        })
    }
}

fn decode_coreos_image(binder: &mut Strict<'_>) -> Result<CoreOSImage, String> {
    Ok(CoreOSImage {
        url: binder.string("URL")?,
        signature_url: binder.string("SignatureURL")?,
        sha256: binder.string("SHA256")?,
        uncompressed_sha256: binder.string("UncompressedSHA256")?,
    })
}

/// Controller-resolved live inputs for one isolated worker attempt.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LiveInputs {
    pub coreos: ResolvedCoreOS,
    pub tailnet: TailnetInputs,
}

impl LiveInputs {
    pub fn marshal(&self) -> String {
        marshal_indent(&Emit::Object(vec![
            ("CoreOS".to_string(), self.coreos.emit()),
            ("Tailnet".to_string(), self.tailnet.emit()),
        ]))
    }

    pub fn decode(binder: &mut Strict<'_>) -> Result<LiveInputs, String> {
        let coreos = binder.nested("CoreOS", "build.ResolvedCoreOS", ResolvedCoreOS::decode)?;
        let tailnet = binder.nested("Tailnet", "build.TailnetInputs", TailnetInputs::decode)?;
        Ok(LiveInputs { coreos, tailnet })
    }
}

/// Records one attempt's live inputs. Validation mirrors the live path.
pub fn write_live_inputs(path: &Path, inputs: &LiveInputs) -> Result<(), Error> {
    valid_live_inputs(inputs)?;
    write_new(path, (inputs.marshal() + "\n").as_bytes(), 0o644)
}

/// Admits controller-resolved inputs for the isolated worker.
pub fn read_live_inputs(path: &Path) -> Result<LiveInputs, Error> {
    let inputs: LiveInputs = read_json(path, "build.LiveInputs", LiveInputs::decode)?;
    valid_live_inputs(&inputs)?;
    Ok(inputs)
}

/// Live resolution shape rules over admitted inputs.
pub fn valid_live_inputs(inputs: &LiveInputs) -> Result<(), Error> {
    valid_resolved_coreos(&inputs.coreos)?;
    valid_tailnet_inputs(&inputs.tailnet)
}

pub fn valid_tailnet_inputs(tailnet: &TailnetInputs) -> Result<(), Error> {
    let reader = soda_build_tools::reader::stream::TailnetInputs {
        version: tailnet.version.clone(),
        sha256: tailnet.sha256.clone(),
        base: tailnet.base.clone(),
    };
    soda_build_tools::reader::stream::valid_tailnet_inputs(&reader)?;
    Ok(())
}

pub fn valid_resolved_coreos(resolved: &ResolvedCoreOS) -> Result<(), Error> {
    let iso = match resolved.iso.get("x86_64") {
        Some(img) => img,
        None => return Err(Error::msg("stable stream x86_64 live ISO is malformed")),
    };
    let qemu = match resolved.qemu.get("x86_64") {
        Some(img) => img,
        None => return Err(Error::msg("stable stream x86_64 qemu image is malformed")),
    };
    let reader_iso = soda_build_tools::reader::stream::CoreOSImage {
        url: iso.url.clone(),
        signature_url: iso.signature_url.clone(),
        sha256: iso.sha256.clone(),
        uncompressed_sha256: iso.uncompressed_sha256.clone(),
    };
    let reader_qemu = soda_build_tools::reader::stream::CoreOSImage {
        url: qemu.url.clone(),
        signature_url: qemu.signature_url.clone(),
        sha256: qemu.sha256.clone(),
        uncompressed_sha256: qemu.uncompressed_sha256.clone(),
    };
    soda_build_tools::reader::stream::valid_stream_images(
        &resolved.release,
        &reader_iso,
        &reader_qemu,
    )?;
    // The shared reader takes single triples; container + metadata rules
    // apply here over the admitted maps.
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
    let entry = resolved
        .container
        .get("x86_64")
        .map(String::as_str)
        .unwrap_or("");
    match entry.split_once("/fedora/fedora-coreos@sha256:") {
        Some((host, digest)) if !host.is_empty() && !host.contains('/') && is_digest(digest) => {
            Ok(())
        }
        _ => Err(Error::msg("digest-pinned CoreOS base required")),
    }
}

fn fetch_capped_json<T: HttpTransport>(
    transport: &T,
    url: &str,
    max_bytes: i64,
) -> Result<Vec<u8>, Error> {
    if !https_url(url) || max_bytes <= 0 {
        return Err(Error::msg("bounded HTTPS fetch required"));
    }
    let response = get_follow(
        transport,
        url,
        None,
        Duration::from_secs(60),
        "unsafe metadata redirect",
    )
    .map_err(|e| Error::msg(format!("live input fetch failed: {}", e.message())))?;
    if response.status != 200 {
        return Err(Error::msg(format!(
            "live input HTTP failure: {}",
            response.status_line()
        )));
    }
    let mut data = Vec::new();
    response
        .body
        .take((max_bytes + 1) as u64)
        .read_to_end(&mut data)
        .map_err(|e| Error::msg(e.to_string()))?;
    if data.len() as i64 > max_bytes {
        return Err(Error::msg("live input exceeds size limit"));
    }
    Ok(data)
}

fn fetch_capped_text<T: HttpTransport>(
    transport: &T,
    url: &str,
    max_bytes: i64,
) -> Result<String, Error> {
    Ok(String::from_utf8_lossy(&fetch_capped_json(transport, url, max_bytes)?).into_owned())
}

/// Resolves the floating Tailnet toolchain: newest stable release, its
/// archive checksum, and the newest upstream alpine-base tag.
pub fn resolve_tailnet_inputs(arch: &str) -> Result<TailnetInputs, Error> {
    resolve_tailnet_inputs_with(&UreqTransport, arch)
}

pub fn resolve_tailnet_inputs_with<T: HttpTransport>(
    transport: &T,
    arch: &str,
) -> Result<TailnetInputs, Error> {
    let platform = oci_architecture(arch)?.to_string();
    let version = latest_tailnet_release_with(transport)?;
    let checksum_url = format!(
        "{index}tailscale_{version}_{platform}.tgz.sha256",
        index = tailnet_index_url()
    );
    let raw = fetch_capped_text(transport, &checksum_url, 1 << 20)?;
    let sha = raw.trim().split(' ').next().unwrap_or("").to_string();
    let base = latest_tailnet_base_tag_with(transport)?;
    let inputs = TailnetInputs {
        version,
        sha256: sha,
        base,
    };
    valid_tailnet_inputs(&inputs)?;
    Ok(inputs)
}

/// Scans for `tailscale_<major>.<minor>.<patch>_amd64.tgz` releases and
/// returns the newest version (original digit strings).
fn latest_tailnet_release(text: &str) -> Result<String, Error> {
    let bytes = text.as_bytes();
    let mut best: [i64; 3] = [0, 0, 0];
    let mut version = String::new();
    let mut i = 0;
    while i < bytes.len() {
        match parse_tailnet_release_at(bytes, i) {
            Some((end, parts)) => {
                let mut nums = [0i64; 3];
                for (slot, part) in parts.iter().enumerate() {
                    nums[slot] = part
                        .parse::<i64>()
                        .map_err(|_| Error::msg("unparseable Tailnet release"))?;
                }
                if version.is_empty()
                    || nums[0] > best[0]
                    || (nums[0] == best[0]
                        && (nums[1] > best[1] || (nums[1] == best[1] && nums[2] > best[2])))
                {
                    best = nums;
                    version = format!("{}.{}.{}", parts[0], parts[1], parts[2]);
                }
                i = end;
            }
            None => i += 1,
        }
    }
    if version.is_empty() {
        return Err(Error::msg("no Tailnet release in upstream index"));
    }
    Ok(version)
}

fn parse_tailnet_release_at(bytes: &[u8], start: usize) -> Option<(usize, [String; 3])> {
    let mut i = start;
    if bytes.get(i..i + 10)? != b"tailscale_" {
        return None;
    }
    i += 10;
    let mut parts = [String::new(), String::new(), String::new()];
    for (slot, part) in parts.iter_mut().enumerate() {
        let begin = i;
        while i < bytes.len() && bytes[i].is_ascii_digit() {
            i += 1;
        }
        if i == begin {
            return None;
        }
        *part = String::from_utf8_lossy(&bytes[begin..i]).into_owned();
        if slot < 2 {
            if bytes.get(i) != Some(&b'.') {
                return None;
            }
            i += 1;
        }
    }
    if bytes.get(i..i + 10)? != b"_amd64.tgz" {
        return None;
    }
    i += 10;
    Some((i, parts))
}

fn latest_tailnet_release_with<T: HttpTransport>(transport: &T) -> Result<String, Error> {
    let raw = fetch_capped_text(transport, &tailnet_index_url(), 1 << 20)?;
    latest_tailnet_release(&raw)
}

fn latest_tailnet_base_tag_with<T: HttpTransport>(transport: &T) -> Result<String, Error> {
    let data = fetch_capped_json(transport, &tailnet_base_tags_url(), 1 << 20)?;
    let text = String::from_utf8_lossy(&data);
    let value = JsonValue::parse(&text).map_err(|_| Error::msg("invalid Tailnet base tags"))?;
    let doc = Fields::of(&value).ok_or_else(|| Error::msg("invalid Tailnet base tags"))?;
    let results = doc
        .object_list("Results")
        .map_err(|_| Error::msg("invalid Tailnet base tags"))?;
    let mut best = String::new();
    let mut best_v = [0i64; 2];
    for tag in &results {
        let name = tag
            .string("Name")
            .map_err(|_| Error::msg("invalid Tailnet base tags"))?;
        let (major_s, minor_s) = match name.split_once('.') {
            Some(pair) if !pair.0.is_empty() && !pair.1.is_empty() => pair,
            _ => continue,
        };
        if major_s.bytes().any(|b| !b.is_ascii_digit())
            || minor_s.bytes().any(|b| !b.is_ascii_digit())
        {
            continue;
        }
        if major_s.contains('.') || minor_s.contains('.') {
            continue;
        }
        // Go ignores Atoi errors here (overflow pins to 0).
        let major = major_s.parse::<i64>().unwrap_or(0);
        let minor = minor_s.parse::<i64>().unwrap_or(0);
        if best.is_empty() || major > best_v[0] || (major == best_v[0] && minor > best_v[1]) {
            best_v = [major, minor];
            best = format!("{major_s}.{minor_s}");
        }
    }
    if best.is_empty() {
        return Err(Error::msg("no Tailnet base tag upstream"));
    }
    Ok(format!("docker.io/tailscale/alpine-base:{best}"))
}

/// Parses one stable-stream document into release + x86_64 ISO/QEMU triples.
fn resolve_stream_build(data: &[u8]) -> Result<(String, CoreOSImage, CoreOSImage), Error> {
    let text = String::from_utf8_lossy(data);
    let value =
        JsonValue::parse(&text).map_err(|_| Error::msg("invalid stable stream document"))?;
    let doc = Fields::of(&value).ok_or_else(|| Error::msg("invalid stable stream document"))?;
    let archs = doc
        .object("architectures")
        .map_err(|_| Error::msg("invalid stable stream document"))?
        .ok_or_else(|| Error::msg("stable stream lacks architecture x86_64"))?;
    let entry = archs
        .object("x86_64")
        .map_err(|_| Error::msg("invalid stable stream document"))?
        .ok_or_else(|| Error::msg("stable stream lacks architecture x86_64"))?;
    let artifacts = entry
        .object("artifacts")
        .map_err(|_| Error::msg("invalid stable stream document"))?;
    let disk = |artifact: &str, format: &str| -> Result<CoreOSImage, Error> {
        let missing = if artifact == "metal" {
            "stable stream lacks x86_64 live ISO".to_string()
        } else {
            format!("stable stream lacks x86_64 {artifact} image")
        };
        let no_artifact = || Error::msg(missing.clone());
        let art = artifacts
            .as_ref()
            .and_then(|a| a.object(artifact).ok().flatten())
            .ok_or_else(no_artifact)?;
        let formats = art
            .object("formats")
            .ok()
            .flatten()
            .ok_or_else(no_artifact)?;
        let entry = formats
            .object(format)
            .ok()
            .flatten()
            .ok_or_else(no_artifact)?;
        // A missing disk decodes to the zero triple, which the shape rules
        // refuse downstream, exactly as the Go owner flows.
        let disk = entry.object("disk").ok().flatten();
        let get = |disk: &Option<Fields<'_>>, name: &str| -> Result<String, Error> {
            match disk {
                None => Ok(String::new()),
                Some(d) => d
                    .string(name)
                    .map_err(|_| Error::msg("invalid stable stream document")),
            }
        };
        Ok(CoreOSImage {
            url: get(&disk, "location")?,
            signature_url: get(&disk, "signature")?,
            sha256: get(&disk, "sha256")?,
            uncompressed_sha256: get(&disk, "uncompressed-sha256")?,
        })
    };
    let iso = disk("metal", "iso")?;
    let release = stream_release_from_location(&iso.url)
        .ok_or_else(|| Error::msg("stable stream x86_64 ISO location names no release"))?;
    let qemu = disk("qemu", "qcow2.xz")?;
    let mut iso_map = HashMap::new();
    iso_map.insert("x86_64".to_string(), iso.clone());
    let mut qemu_map = HashMap::new();
    qemu_map.insert("x86_64".to_string(), qemu.clone());
    valid_stream_images_local(&release, &iso_map, &qemu_map)?;
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

fn valid_stream_images_local(
    release: &str,
    iso: &HashMap<String, CoreOSImage>,
    qemu: &HashMap<String, CoreOSImage>,
) -> Result<(), Error> {
    let iso_img = iso.get("x86_64");
    let qemu_img = qemu.get("x86_64");
    if iso_img.is_none() {
        return Err(Error::msg("stable stream x86_64 live ISO is malformed"));
    }
    if qemu_img.is_none() {
        return Err(Error::msg("stable stream x86_64 qemu image is malformed"));
    }
    let reader_iso = soda_build_tools::reader::stream::CoreOSImage {
        url: iso_img.unwrap().url.clone(),
        signature_url: iso_img.unwrap().signature_url.clone(),
        sha256: iso_img.unwrap().sha256.clone(),
        uncompressed_sha256: iso_img.unwrap().uncompressed_sha256.clone(),
    };
    let reader_qemu = soda_build_tools::reader::stream::CoreOSImage {
        url: qemu_img.unwrap().url.clone(),
        signature_url: qemu_img.unwrap().signature_url.clone(),
        sha256: qemu_img.unwrap().sha256.clone(),
        uncompressed_sha256: qemu_img.unwrap().uncompressed_sha256.clone(),
    };
    soda_build_tools::reader::stream::valid_stream_images(release, &reader_iso, &reader_qemu)?;
    Ok(())
}

fn resolve_registry_digests_with<T: HttpTransport>(
    transport: &T,
    registry: &str,
) -> Result<HashMap<String, String>, Error> {
    if !https_url(registry) {
        return Err(Error::msg("container registry URL must be HTTPS"));
    }
    let host = registry
        .split_once("://")
        .map(|(_, rest)| rest.split('/').next().unwrap_or(""))
        .unwrap_or("");
    if host.is_empty() || host.contains('/') {
        return Err(Error::msg("container registry URL is malformed"));
    }
    let endpoint =
        format!("{registry}/v2/{COREOS_CONTAINER_REPO}/manifests/{COREOS_CONTAINER_TAG}");
    let response = get_follow(
        transport,
        &endpoint,
        Some("application/vnd.oci.image.index.v1+json"),
        Duration::from_secs(60),
        "unsafe metadata redirect",
    )
    .map_err(|_| Error::msg("container registry fetch failed"))?;
    if response.status == 401 {
        return Err(Error::msg(
            "container registry refused anonymous manifest access",
        ));
    }
    if response.status != 200 {
        return Err(Error::msg(format!(
            "container registry HTTP failure: {}",
            response.status_line()
        )));
    }
    let mut data = Vec::new();
    response
        .body
        .take((1 << 20) + 1)
        .read_to_end(&mut data)
        .map_err(|e| Error::msg(e.to_string()))?;
    if data.len() > 1 << 20 {
        return Err(Error::msg("container index exceeds size limit"));
    }
    let text = String::from_utf8_lossy(&data);
    let value = JsonValue::parse(&text).map_err(|_| Error::msg("invalid container index"))?;
    let index = Fields::of(&value).ok_or_else(|| Error::msg("invalid container index"))?;
    let manifests = index
        .object_list("manifests")
        .map_err(|_| Error::msg("invalid container index"))?;
    let oci_arch = oci_architecture("x86_64")?;
    let mut found = String::new();
    for manifest in &manifests {
        let platform = manifest
            .object("platform")
            .map_err(|_| Error::msg("invalid container index"))?;
        let architecture = match platform {
            Some(p) => p
                .string("architecture")
                .map_err(|_| Error::msg("invalid container index"))?,
            None => String::new(),
        };
        if architecture != oci_arch {
            continue;
        }
        let digest = manifest
            .string("digest")
            .map_err(|_| Error::msg("invalid container index"))?;
        match digest.strip_prefix("sha256:") {
            Some(hex) if is_digest(hex) => {}
            _ => return Err(Error::msg("container index x86_64 digest is malformed")),
        }
        found = digest;
    }
    if found.is_empty() {
        return Err(Error::msg("container index lacks architecture x86_64"));
    }
    let mut digests = HashMap::new();
    digests.insert(
        "x86_64".to_string(),
        format!("{host}/{COREOS_CONTAINER_REPO}@{found}"),
    );
    Ok(digests)
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
    let mut iso_map = HashMap::new();
    iso_map.insert("x86_64".to_string(), iso);
    let mut qemu_map = HashMap::new();
    qemu_map.insert("x86_64".to_string(), qemu);
    Ok(ResolvedCoreOS {
        release,
        metadata_url: meta,
        container: digests,
        iso: iso_map,
        qemu: qemu_map,
    })
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use crate::http::tests::Stub;

    pub fn fixture_live_inputs() -> LiveInputs {
        let img = CoreOSImage {
            url: "https://builds.test/fedora-coreos-44.20260901.1.0-live.x86_64.iso".to_string(),
            signature_url: "https://builds.test/fedora-coreos-44.20260901.1.0-live.x86_64.iso.sig"
                .to_string(),
            sha256: "a".repeat(64),
            uncompressed_sha256: "b".repeat(64),
        };
        let mut container = HashMap::new();
        container.insert(
            "x86_64".to_string(),
            format!("quay.io/fedora/fedora-coreos@sha256:{}", "c".repeat(64)),
        );
        let mut iso = HashMap::new();
        iso.insert("x86_64".to_string(), img.clone());
        let mut qemu = HashMap::new();
        qemu.insert("x86_64".to_string(), img);
        LiveInputs {
            coreos: ResolvedCoreOS {
                release: "44.20260901.1.0".to_string(),
                metadata_url:
                    "https://builds.test/prod/streams/stable/builds/44.20260901.1.0/release.json"
                        .to_string(),
                container,
                iso,
                qemu,
            },
            tailnet: TailnetInputs {
                version: "1.98.2".to_string(),
                sha256: "e".repeat(64),
                base: "docker.io/tailscale/alpine-base:3.22".to_string(),
            },
        }
    }

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
    fn oracle_tailnet_release_selection() {
        // Oracle: latestTailnetRelease newest-stable selection.
        let index =
            "tailscale_1.98.1_amd64.tgz\ntailscale_1.98.2_amd64.tgz\ntailscale_1.98.10_amd64.tgz\n";
        assert_eq!(latest_tailnet_release(index).unwrap(), "1.98.10");
        assert_eq!(
            latest_tailnet_release("nothing here")
                .unwrap_err()
                .message(),
            "no Tailnet release in upstream index"
        );
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
    fn oracle_live_inputs_round_trip() {
        // Oracle: WriteLiveInputs/ReadLiveInputs keep the resolved release.
        let dir = std::env::temp_dir().join(format!(
            "soda-live-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("live-inputs.json");
        let inputs = fixture_live_inputs();
        write_live_inputs(&path, &inputs).unwrap();
        let back = read_live_inputs(&path).unwrap();
        assert_eq!(back.coreos.release, "44.20260901.1.0");
        assert_eq!(back.tailnet.version, "1.98.2");
        // Tampered digests and versions are refused on read.
        let mut bad = inputs.clone();
        bad.tailnet.version = "yesterday".to_string();
        assert!(write_live_inputs(&dir.join("bad.json"), &bad).is_err());
        let mut bad = inputs.clone();
        bad.coreos.release = "tomorrow".to_string();
        assert!(write_live_inputs(&dir.join("bad2.json"), &bad).is_err());
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
