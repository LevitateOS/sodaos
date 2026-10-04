//! Port of `internal/release/build` OCI inspection (`oci.go`,
//! `oci_layout.go`) consumed by `deliver`: single-image archive identity,
//! rootfs content hashes, and shared-layout verification. Streams archives
//! without extracting layers or importing release code.

use std::collections::{BTreeMap, BTreeSet};
use std::io::Read;

use flate2::read::GzDecoder;
use sha2::{Digest as _, Sha256};
use soda_json::JsonValue;

use crate::buildx::{is_digest, is_revision, oci_architecture, read_layout_entry, Image as BuildImage, Root};
use crate::jsonx::{as_i64, parse_lenient, Soft};
use crate::model::path_clean;
use crate::Error;

const MANIFEST_TYPE: &str = "application/vnd.oci.image.manifest.v1+json";
const CONFIG_TYPE: &str = "application/vnd.oci.image.config.v1+json";
const INDEX_TYPE: &str = "application/vnd.oci.image.index.v1+json";
const LAYER_TAR: &str = "application/vnd.oci.image.layer.v1.tar";
const LAYER_GZIP: &str = "application/vnd.oci.image.layer.v1.tar+gzip";
const LAYER_ZSTD: &str = "application/vnd.oci.image.layer.v1.tar+zstd";

/// `build.OCILayout`: verified shared-blob directory.
#[derive(Debug, Clone, Default)]
pub struct OciLayout {
    pub images: BTreeMap<String, BuildImage>,
    pub files: BTreeMap<String, String>,
    pub bytes: u64,
}

#[derive(Debug, Clone, Default)]
struct Blob {
    hash: String,
    size: i64,
    data: Option<Vec<u8>>,
}

#[derive(Debug, Clone, Default)]
struct Descriptor {
    digest: String,
    size: i64,
    media_type: String,
    urls: Vec<String>,
    annotations: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Default)]
struct OciManifest {
    config: Descriptor,
    layers: Vec<Descriptor>,
}

#[derive(Debug, Clone, Default)]
struct LayerMember {
    hash: String,
    present: bool,
    blocked: bool,
}

// ---------------------------------------------------------------------------
// Small decoders (encoding/json semantics: lenient, last-wins)
// ---------------------------------------------------------------------------

fn decode_descriptor(value: &JsonValue) -> Result<Descriptor, ()> {
    let soft = Soft::new(value)?;
    let mut descriptor = Descriptor {
        digest: soft.string("digest")?.unwrap_or_default(),
        size: as_i64(soft.integer("size")?.unwrap_or(0))?,
        media_type: soft.string("mediaType")?.unwrap_or_default(),
        urls: Vec::new(),
        annotations: BTreeMap::new(),
    };
    if let Some(items) = soft.array("urls")? {
        for item in items {
            match item {
                JsonValue::Str(s) => descriptor.urls.push(s.clone()),
                _ => return Err(()),
            }
        }
    }
    if let Some(entries) = soft.object("annotations")?.and_then(|o| o.entries().map(|e| e.to_vec())) {
        for (key, item) in &entries {
            match item {
                JsonValue::Str(s) => {
                    descriptor.annotations.insert(key.clone(), s.clone());
                }
                _ => return Err(()),
            }
        }
    }
    Ok(descriptor)
}

fn read_oci_index(entries: &BTreeMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
    let layout_data = entries.get("oci-layout").and_then(|b| b.data.as_deref());
    let layout_value = layout_data.and_then(|data| parse_lenient(data).ok());
    let layout_ok = layout_value
        .as_ref()
        .and_then(|v| Soft::new(v).ok())
        .and_then(|s| s.string("imageLayoutVersion").ok().flatten())
        == Some("1.0.0".to_string());
    if !layout_ok {
        return Err(Error::msg("missing OCI layout"));
    }
    let index_data = entries.get("index.json").and_then(|b| b.data.as_deref());
    let value = index_data
        .and_then(|data| parse_lenient(data).ok())
        .ok_or_else(|| Error::msg("valid OCI index required"))?;
    let soft = Soft::new(&value).map_err(|_| Error::msg("valid OCI index required"))?;
    let version = as_i64(soft.integer("schemaVersion").map_err(|_| Error::msg("valid OCI index required"))?.unwrap_or(0))
        .map_err(|_| Error::msg("valid OCI index required"))?;
    let media = soft
        .string("mediaType")
        .map_err(|_| Error::msg("valid OCI index required"))?
        .unwrap_or_default();
    if version != 2 || (!media.is_empty() && media != INDEX_TYPE) {
        return Err(Error::msg("valid OCI index required"));
    }
    let mut manifests = Vec::new();
    if let Some(items) = soft
        .array("manifests")
        .map_err(|_| Error::msg("valid OCI index required"))?
    {
        for item in items {
            manifests.push(decode_descriptor(item).map_err(|_| Error::msg("valid OCI index required"))?);
        }
    }
    Ok(manifests)
}

fn parse_oci_manifest(data: &[u8]) -> Result<OciManifest, Error> {
    let value = parse_lenient(data).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let soft = Soft::new(&value).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let invalid = || Error::msg("invalid OCI image manifest");
    let version = soft.integer("schemaVersion").map_err(|_| invalid())?.unwrap_or(0);
    let media = soft.string("mediaType").map_err(|_| invalid())?.unwrap_or_default();
    if version != 2 || (!media.is_empty() && media != MANIFEST_TYPE) {
        return Err(invalid());
    }
    let config_value = soft
        .field("config")
        .cloned()
        .unwrap_or(JsonValue::Object(Vec::new()));
    let config = decode_descriptor(&config_value).map_err(|_| invalid())?;
    if config.media_type != CONFIG_TYPE {
        return Err(invalid());
    }
    let mut layers = Vec::new();
    if let Some(items) = soft.array("layers").map_err(|_| invalid())? {
        for item in items {
            layers.push(decode_descriptor(item).map_err(|_| invalid())?);
        }
    }
    Ok(OciManifest { config, layers })
}

struct OciConfig {
    os: String,
    arch: String,
    rootfs_type: String,
    diff_ids: Vec<String>,
    labels: BTreeMap<String, String>,
}

fn parse_oci_config(data: &[u8]) -> Result<OciConfig, Error> {
    let invalid = || Error::msg("invalid OCI image config");
    let value = parse_lenient(data).map_err(|_| invalid())?;
    let soft = Soft::new(&value).map_err(|_| invalid())?;
    let rootfs = soft.object("rootfs").map_err(|_| invalid())?;
    let mut diff_ids = Vec::new();
    let mut rootfs_type = String::new();
    if let Some(rootfs) = rootfs {
        rootfs_type = rootfs.string("type").map_err(|_| invalid())?.unwrap_or_default();
        if let Some(items) = rootfs.array("diff_ids").map_err(|_| invalid())? {
            for item in items {
                match item {
                    JsonValue::Str(s) => diff_ids.push(s.clone()),
                    _ => return Err(invalid()),
                }
            }
        }
    }
    let mut labels = BTreeMap::new();
    if let Some(config) = soft.object("config").map_err(|_| invalid())? {
        if let Some(entries) = config.object("Labels").map_err(|_| invalid())?.and_then(|o| o.entries().map(|e| e.to_vec()))
        {
            for (key, item) in &entries {
                match item {
                    JsonValue::Str(s) => {
                        labels.insert(key.clone(), s.clone());
                    }
                    _ => return Err(invalid()),
                }
            }
        }
    }
    Ok(OciConfig {
        os: soft.string("os").map_err(|_| invalid())?.unwrap_or_default(),
        arch: soft.string("architecture").map_err(|_| invalid())?.unwrap_or_default(),
        rootfs_type,
        diff_ids,
        labels,
    })
}

// ---------------------------------------------------------------------------
// Blob ingestion
// ---------------------------------------------------------------------------

fn is_json(data: &[u8]) -> bool {
    parse_lenient(data).is_ok()
}

fn read_oci_blob(
    entries: &mut BTreeMap<String, Blob>,
    name: &str,
    length: i64,
    data: Vec<u8>,
    json_bytes: &mut i64,
) -> Result<(), Error> {
    if length < 0 || length == i64::MAX {
        return Err(Error::msg("invalid OCI blob size"));
    }
    if data.len() as i64 != length {
        return Err(Error::msg("OCI blob size changed"));
    }
    let mut hasher = Sha256::new();
    hasher.update(&data);
    let sum = format!("{:x}", hasher.finalize());
    if name.starts_with("blobs/") && *name != format!("blobs/sha256/{sum}") {
        return Err(Error::msg("OCI blob checksum mismatch"));
    }
    let body = if (length as usize) <= (4 << 20) && is_json(&data) {
        Some(data)
    } else {
        None
    };
    *json_bytes += body.as_ref().map(|b| b.len()).unwrap_or(0) as i64;
    if *json_bytes > 32 << 20 {
        return Err(Error::msg("OCI JSON metadata limit exceeded"));
    }
    entries.insert(
        name.to_string(),
        Blob {
            hash: sum,
            size: length,
            data: body,
        },
    );
    Ok(())
}

fn fetch_oci_blob(
    entries: &BTreeMap<String, Blob>,
    d: &Descriptor,
    load: &Option<Box<dyn Fn(&str) -> Result<(), Error>>>,
) -> Result<Blob, Error> {
    if d.size < 0 || !d.urls.is_empty() {
        return Err(Error::msg("local bounded OCI descriptor required"));
    }
    let hex = d.digest.strip_prefix("sha256:").unwrap_or("");
    if !d.digest.starts_with("sha256:") || !is_digest(hex) {
        return Err(Error::msg("invalid OCI digest"));
    }
    let name = format!("blobs/sha256/{hex}");
    if let Some(load) = load {
        load(&name)?;
    }
    match entries.get(&name) {
        Some(blob) if blob.size == d.size => Ok(blob.clone()),
        _ => Err(Error::msg("missing or wrong-size OCI blob")),
    }
}

// ---------------------------------------------------------------------------
// Image identity
// ---------------------------------------------------------------------------

fn validate_oci_layers(
    entries: &BTreeMap<String, Blob>,
    layers: &[Descriptor],
    load: &Option<Box<dyn Fn(&str) -> Result<(), Error>>>,
) -> Result<(), Error> {
    for layer in layers {
        match layer.media_type.as_str() {
            LAYER_TAR | LAYER_GZIP | LAYER_ZSTD => {}
            _ => return Err(Error::msg("unsupported OCI layer media type")),
        }
        fetch_oci_blob(entries, layer, load)?;
    }
    Ok(())
}

fn validate_oci_rootfs(diff_ids: &[String], layers: &[Descriptor]) -> Result<(), Error> {
    if diff_ids.len() != layers.len() {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    for (i, id) in diff_ids.iter().enumerate() {
        let hex = id.strip_prefix("sha256:").unwrap_or("");
        if !id.starts_with("sha256:") || !is_digest(hex) {
            return Err(Error::msg("invalid OCI diff ID"));
        }
        if layers[i].media_type == LAYER_TAR && *id != layers[i].digest {
            return Err(Error::msg("uncompressed OCI layer identity mismatch"));
        }
    }
    Ok(())
}

fn validate_oci_attribution(
    labels: &BTreeMap<String, String>,
    want_revision: &str,
) -> Result<(), Error> {
    let rev = labels
        .get("org.opencontainers.image.revision")
        .cloned()
        .unwrap_or_default();
    if !want_revision.is_empty() && rev != want_revision {
        return Err(Error::msg("OCI source revision mismatch"));
    }
    if want_revision.is_empty() {
        return Ok(());
    }
    let base_digest = labels
        .get("org.opencontainers.image.base.digest")
        .cloned()
        .unwrap_or_default();
    let base_hex = base_digest.strip_prefix("sha256:").unwrap_or("");
    if labels
        .get("org.opencontainers.image.source")
        .map(String::as_str)
        != Some("https://github.com/LevitateOS/sodaos")
        || labels
            .get("org.opencontainers.image.base.name")
            .map(String::is_empty)
            .unwrap_or(true)
        || !base_digest.starts_with("sha256:")
        || !is_digest(base_hex)
    {
        return Err(Error::msg("soda image lacks source/base attribution"));
    }
    Ok(())
}

fn inspect_oci_config(
    config_blob: &Blob,
    layers: &[Descriptor],
    want: &str,
    revision: &str,
    image_digest: &str,
    config_digest: &str,
) -> Result<BuildImage, Error> {
    let empty = Vec::new();
    let cfg = parse_oci_config(config_blob.data.as_ref().unwrap_or(&empty))?;
    if cfg.rootfs_type != "layers" {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    validate_oci_rootfs(&cfg.diff_ids, layers)?;
    if cfg.os != "linux" || cfg.arch != want {
        return Err(Error::msg(format!("OCI must be linux/{want}")));
    }
    validate_oci_attribution(&cfg.labels, revision)?;
    let get = |key: &str| cfg.labels.get(key).cloned().unwrap_or_default();
    Ok(BuildImage {
        manifest: image_digest.to_string(),
        config: config_digest.to_string(),
        architecture: cfg.arch,
        revision: get("org.opencontainers.image.revision"),
        source: get("org.opencontainers.image.source"),
        base_name: get("org.opencontainers.image.base.name"),
        base_digest: get("org.opencontainers.image.base.digest"),
    })
}

fn inspect_oci_image(
    entries: &BTreeMap<String, Blob>,
    image: &Descriptor,
    want: &str,
    revision: &str,
    load: &Option<Box<dyn Fn(&str) -> Result<(), Error>>>,
) -> Result<BuildImage, Error> {
    let manifest_blob = fetch_oci_blob(entries, image, load)?;
    let empty = Vec::new();
    let manifest = parse_oci_manifest(manifest_blob.data.as_ref().unwrap_or(&empty))?;
    validate_oci_layers(entries, &manifest.layers, load)?;
    let config_blob = fetch_oci_blob(entries, &manifest.config, load)?;
    inspect_oci_config(
        &config_blob,
        &manifest.layers,
        want,
        revision,
        &image.digest,
        &manifest.config.digest,
    )
}

// ---------------------------------------------------------------------------
// Archive ingestion
// ---------------------------------------------------------------------------

fn open_oci_archive(file: &str, arch: &str, revision: &str) -> Result<(String, std::fs::File), Error> {
    let want = oci_architecture(arch).map_err(|e| Error::msg(e.0))?;
    if !revision.is_empty() && !is_revision(revision) {
        return Err(Error::msg("full source revision required"));
    }
    let st = std::fs::symlink_metadata(file)
        .map_err(|e| Error::msg(format!("lstat {file}: {e}")))?;
    if !st.is_file() {
        return Err(Error::msg("OCI archive must be regular"));
    }
    let f =
        std::fs::File::open(file).map_err(|e| Error::msg(format!("open {file}: {e}")))?;
    Ok((want.to_string(), f))
}

fn is_valid_oci_regular_entry(name: &str) -> bool {
    if name == "index.json" || name == "oci-layout" {
        return true;
    }
    match name.strip_prefix("blobs/sha256/") {
        Some(hex) => is_digest(hex),
        None => false,
    }
}

fn read_oci_archive_entries<R: Read>(
    reader: R,
) -> Result<BTreeMap<String, Blob>, Error> {
    let mut archive = tar::Archive::new(reader);
    let mut entries: BTreeMap<String, Blob> = BTreeMap::new();
    let mut seen: BTreeSet<String> = BTreeSet::new();
    let mut json_bytes: i64 = 0;
    let items = archive
        .entries()
        .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
    for item in items {
        let mut item = item.map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        let raw_name = String::from_utf8_lossy(&item.path_bytes()).into_owned();
        let name = path_clean(&raw_name);
        let header = item.header().clone();
        let is_dir = header.entry_type().is_dir();
        if name == "." && is_dir {
            continue;
        }
        if raw_name.starts_with('/') || name == ".." || name.starts_with("../") {
            return Err(Error::msg("unsafe OCI path"));
        }
        if !seen.insert(name.clone()) {
            return Err(Error::msg("duplicate OCI entry"));
        }
        if seen.len() > 100000 {
            return Err(Error::msg("too many OCI entries"));
        }
        if is_dir {
            if name != "blobs" && name != "blobs/sha256" {
                return Err(Error::msg("unexpected OCI directory"));
            }
            continue;
        }
        if !header.entry_type().is_file() {
            return Err(Error::msg("non-regular OCI entry"));
        }
        if entries.contains_key(&name) {
            return Err(Error::msg("duplicate OCI entry"));
        }
        if entries.len() > 100000 {
            return Err(Error::msg("too many OCI entries"));
        }
        if !is_valid_oci_regular_entry(&name) {
            return Err(Error::msg("not an OCI archive"));
        }
        let size = header
            .size()
            .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        let length =
            i64::try_from(size).map_err(|_| Error::msg("invalid OCI blob size"))?;
        let mut data = Vec::new();
        item.read_to_end(&mut data)
            .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        read_oci_blob(&mut entries, &name, length, data, &mut json_bytes)?;
    }
    Ok(entries)
}

fn inspect_archive_index(
    entries: &BTreeMap<String, Blob>,
    want: &str,
    revision: &str,
) -> Result<BuildImage, Error> {
    let index = read_oci_index(entries)?;
    if index.len() != 1 || index[0].media_type != MANIFEST_TYPE {
        return Err(Error::msg("single-platform OCI index required"));
    }
    inspect_oci_image(entries, &index[0], want, revision, &None)
}

/// `build.InspectOCI`: verify single-image archive identity.
pub fn inspect_oci(file: &str, arch: &str, revision: &str) -> Result<BuildImage, Error> {
    let (want, f) = open_oci_archive(file, arch, revision)?;
    let entries = read_oci_archive_entries(f)?;
    inspect_archive_index(&entries, &want, revision)
}

// ---------------------------------------------------------------------------
// Content verification
// ---------------------------------------------------------------------------

fn requested_oci_paths(paths: &[String]) -> Result<BTreeMap<String, String>, Error> {
    let mut wanted: BTreeMap<String, String> = BTreeMap::new();
    for requested in paths {
        let name = requested.strip_prefix('/').unwrap_or("");
        if !requested.starts_with('/') || name.is_empty() || path_clean(name) != name || name.bytes().any(|b| matches!(b, b'\\' | b'\n' | b'\r' | 0)) {
            return Err(Error::msg("absolute clean OCI member paths required"));
        }
        if wanted.insert(name.to_string(), requested.clone()).is_some() {
            return Err(Error::msg("duplicate OCI member request"));
        }
    }
    if wanted.is_empty() {
        return Err(Error::msg("explicit OCI members required"));
    }
    Ok(wanted)
}

fn clean_layer_name(name: &str) -> Result<String, Error> {
    let trimmed = name.strip_prefix("./").unwrap_or(name);
    let clean = path_clean(trimmed);
    let segments: Vec<&str> = trimmed.split('/').collect();
    if clean == "."
        || clean.starts_with('/')
        || segments.contains(&"..")
        || clean.bytes().any(|b| matches!(b, b'\n' | b'\r' | 0))
    {
        return Err(Error::msg("unsafe OCI layer path"));
    }
    Ok(clean)
}

fn path_dir(name: &str) -> &str {
    match name.rfind('/') {
        Some(i) => &name[..i],
        None => ".",
    }
}

fn path_base(name: &str) -> &str {
    match name.rfind('/') {
        Some(i) => &name[i + 1..],
        None => name,
    }
}

fn path_join(dir: &str, base: &str) -> String {
    if dir == "." || dir.is_empty() {
        base.to_string()
    } else {
        format!("{dir}/{base}")
    }
}

fn whiteout_target(name: &str) -> Option<String> {
    let base = path_base(name);
    if !base.starts_with(".wh.") || base == ".wh..wh..opq" {
        return None;
    }
    Some(path_join(path_dir(name), base.strip_prefix(".wh.").unwrap_or("")))
}

fn record_layer_deletion(
    found: &mut BTreeMap<String, LayerMember>,
    wanted: &BTreeMap<String, String>,
    removed: &str,
) {
    for name in wanted.keys() {
        if name != removed && !name.starts_with(&format!("{removed}/")) {
            continue;
        }
        if let Some(existing) = found.get(name) {
            if existing.present || existing.blocked {
                continue;
            }
        }
        found.insert(name.clone(), LayerMember::default());
    }
}

fn record_ancestor_replacement(
    found: &mut BTreeMap<String, LayerMember>,
    wanted: &BTreeMap<String, String>,
    name: &str,
    is_dir: bool,
) {
    if is_dir {
        return;
    }
    for requested in wanted.keys() {
        if requested.starts_with(&format!("{name}/")) {
            found.insert(
                requested.clone(),
                LayerMember {
                    blocked: true,
                    ..LayerMember::default()
                },
            );
        }
    }
}

fn record_opaque_directory(
    found: &mut BTreeMap<String, LayerMember>,
    wanted: &BTreeMap<String, String>,
    dir: &str,
) {
    for requested in wanted.keys() {
        if dir != "." && !requested.starts_with(&format!("{dir}/")) {
            continue;
        }
        if let Some(existing) = found.get(requested) {
            if existing.present || existing.blocked {
                continue;
            }
        }
        found.insert(requested.clone(), LayerMember::default());
    }
}

fn record_layer_entry<R: Read>(
    reader: &mut R,
    size: u64,
    is_regular: bool,
    is_dir: bool,
    name: &str,
    found: &mut BTreeMap<String, LayerMember>,
    wanted: &BTreeMap<String, String>,
) -> Result<(), Error> {
    if let Some(removed) = whiteout_target(name) {
        return Ok(record_layer_deletion(found, wanted, &removed));
    }
    if path_base(name) == ".wh..wh..opq" {
        record_opaque_directory(found, wanted, path_dir(name));
        return Ok(());
    }
    record_ancestor_replacement(found, wanted, name, is_dir);
    if !wanted.contains_key(name) || found.get(name).map(|m| m.blocked).unwrap_or(false) {
        return Ok(());
    }
    if !is_regular {
        return Err(Error::msg("requested OCI member is non-regular"));
    }
    let mut hasher = Sha256::new();
    let mut copied: u64 = 0;
    let mut chunk = [0u8; 65536];
    loop {
        let n = reader
            .read(&mut chunk)
            .map_err(|_| Error::msg("OCI layer member size changed"))?;
        if n == 0 {
            break;
        }
        hasher.update(&chunk[..n]);
        copied += n as u64;
    }
    if copied != size {
        return Err(Error::msg("OCI layer member size changed"));
    }
    found.insert(
        name.to_string(),
        LayerMember {
            hash: format!("{:x}", hasher.finalize()),
            present: true,
            blocked: false,
        },
    );
    Ok(())
}

fn scan_oci_layer<R: Read>(
    reader: R,
    wanted: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, LayerMember>, Error> {
    let mut found: BTreeMap<String, LayerMember> = BTreeMap::new();
    let mut seen: BTreeSet<String> = BTreeSet::new();
    let mut archive = tar::Archive::new(reader);
    let items = archive
        .entries()
        .map_err(|e| Error::msg(format!("read OCI layer: {e}")))?;
    for item in items {
        let mut item = item.map_err(|e| Error::msg(format!("read OCI layer: {e}")))?;
        let raw_name = String::from_utf8_lossy(&item.path_bytes()).into_owned();
        if raw_name == "." || raw_name == "./" {
            continue;
        }
        let name = clean_layer_name(&raw_name)?;
        if !seen.insert(name.clone()) {
            return Err(Error::msg("duplicate OCI layer entry"));
        }
        if seen.len() > 1000000 {
            return Err(Error::msg("too many OCI layer entries"));
        }
        let header = item.header().clone();
        let size = header
            .size()
            .map_err(|e| Error::msg(format!("read OCI layer: {e}")))?;
        record_layer_entry(
            &mut item,
            size,
            header.entry_type().is_file(),
            header.entry_type().is_dir(),
            &name,
            &mut found,
            wanted,
        )?;
    }
    Ok(found)
}

fn layer_archive_indexes(
    layers: &[Descriptor],
) -> Result<BTreeMap<String, Vec<usize>>, Error> {
    let mut indexes: BTreeMap<String, Vec<usize>> = BTreeMap::new();
    let mut seen: BTreeMap<String, &Descriptor> = BTreeMap::new();
    for (i, layer) in layers.iter().enumerate() {
        let hex = layer.digest.strip_prefix("sha256:").unwrap_or("\x00");
        let name = format!("blobs/sha256/{hex}");
        if let Some(prior) = seen.get(&name) {
            if prior.media_type != layer.media_type || prior.size != layer.size {
                return Err(Error::msg("conflicting OCI layer descriptors"));
            }
        }
        seen.insert(name.clone(), layer);
        indexes.entry(name).or_default().push(i);
    }
    Ok(indexes)
}

struct TeeHasher<R> {
    inner: R,
    hasher: Sha256,
}

impl<R: Read> Read for TeeHasher<R> {
    fn read(&mut self, buf: &mut [u8]) -> std::io::Result<usize> {
        let n = self.inner.read(buf)?;
        self.hasher.update(&buf[..n]);
        Ok(n)
    }
}

fn drain(reader: &mut dyn Read) -> std::io::Result<()> {
    let mut chunk = [0u8; 65536];
    loop {
        if reader.read(&mut chunk)? == 0 {
            return Ok(());
        }
    }
}

fn scan_archive_layer<R: Read>(
    reader: &mut R,
    descriptor: &Descriptor,
    wanted: &BTreeMap<String, String>,
) -> Result<(BTreeMap<String, LayerMember>, bool), Error> {
    let changed = || Error::msg("OCI layer changed during member verification");
    let mut tee = TeeHasher {
        inner: reader,
        hasher: Sha256::new(),
    };
    if descriptor.media_type == LAYER_ZSTD {
        if drain(&mut tee).is_err() {
            return Err(changed());
        }
        let sum = format!("sha256:{:x}", tee.hasher.clone().finalize());
        if sum != descriptor.digest {
            return Err(changed());
        }
        return Ok((BTreeMap::new(), true));
    }
    let members: BTreeMap<String, LayerMember> = match descriptor.media_type.as_str() {
        LAYER_TAR => scan_oci_layer(&mut tee, wanted)?,
        LAYER_GZIP => {
            let gz = GzDecoder::new(&mut tee);
            scan_oci_layer(gz, wanted)?
        }
        _ => return Err(Error::msg("unsupported OCI layer media type")),
    };
    if drain(&mut tee).is_err() {
        return Err(changed());
    }
    let sum = format!("sha256:{:x}", tee.hasher.clone().finalize());
    if sum != descriptor.digest {
        return Err(changed());
    }
    Ok((members, false))
}

fn resolve_oci_members(
    layers: &[BTreeMap<String, LayerMember>],
    unsupported: &[bool],
    wanted: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, String>, Error> {
    let mut resolved: BTreeMap<String, String> = BTreeMap::new();
    for (name, requested) in wanted {
        for i in (0..layers.len()).rev() {
            if unsupported[i] {
                return Err(Error::msg("zstd OCI layer blocks member verification"));
            }
            let member = match layers[i].get(name) {
                Some(member) => member,
                None => continue,
            };
            if member.blocked {
                return Err(Error::msg("requested OCI member has a non-directory ancestor"));
            }
            if !member.present {
                return Err(Error::msg("requested OCI member was removed"));
            }
            resolved.insert(requested.clone(), member.hash.clone());
            break;
        }
        if !resolved.contains_key(requested) {
            return Err(Error::msg("requested OCI member missing"));
        }
    }
    Ok(resolved)
}

fn inspect_content_manifest(
    entries: &BTreeMap<String, Blob>,
    want: &str,
    revision: &str,
) -> Result<(BuildImage, OciManifest), Error> {
    let index = read_oci_index(entries)?;
    if index.len() != 1 || index[0].media_type != MANIFEST_TYPE {
        return Err(Error::msg("single-platform OCI index required"));
    }
    let image = inspect_oci_image(entries, &index[0], want, revision, &None)?;
    let manifest_blob = fetch_oci_blob(entries, &index[0], &None)?;
    let empty = Vec::new();
    let manifest = parse_oci_manifest(manifest_blob.data.as_ref().unwrap_or(&empty))?;
    Ok((image, manifest))
}

/// `build.InspectOCIContent`: identity plus exact rootfs member hashes.
pub fn inspect_oci_content(
    file: &str,
    arch: &str,
    revision: &str,
    paths: &[String],
) -> Result<(BuildImage, BTreeMap<String, String>), Error> {
    let wanted = requested_oci_paths(paths)?;
    let (want, mut f) = open_oci_archive(file, arch, revision)?;
    let entries = read_oci_archive_entries(&f)?;
    let (image, manifest) = inspect_content_manifest(&entries, &want, revision)?;
    use std::io::Seek;
    f.seek(std::io::SeekFrom::Start(0))
        .map_err(|e| Error::msg(format!("seek {file}: {e}")))?;
    let (layers, unsupported) = scan_oci_archive_layers(&mut f, &manifest.layers, &wanted)?;
    let content = resolve_oci_members(&layers, &unsupported, &wanted)?;
    Ok((image, content))
}

fn scan_oci_archive_layers<R: Read>(
    reader: R,
    layers: &[Descriptor],
    wanted: &BTreeMap<String, String>,
) -> Result<(Vec<BTreeMap<String, LayerMember>>, Vec<bool>), Error> {
    let indexes = layer_archive_indexes(layers)?;
    let mut found: Vec<BTreeMap<String, LayerMember>> = Vec::new();
    found.resize_with(layers.len(), BTreeMap::new);
    let mut unsupported = vec![false; layers.len()];
    let mut archive = tar::Archive::new(reader);
    let items = archive
        .entries()
        .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
    for item in items {
        let mut item = item.map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        let raw_name = String::from_utf8_lossy(&item.path_bytes()).into_owned();
        let positions = indexes.get(&path_clean(&raw_name)).cloned().unwrap_or_default();
        if positions.is_empty() {
            continue;
        }
        let (members, blocked) = scan_archive_layer(&mut item, &layers[positions[0]], wanted)?;
        for position in positions {
            found[position] = members.clone();
            unsupported[position] = blocked;
        }
    }
    Ok((found, unsupported))
}

// ---------------------------------------------------------------------------
// Shared layout
// ---------------------------------------------------------------------------

fn validate_oci_layout_inputs(
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<String, Error> {
    let want = oci_architecture(arch).map_err(|e| Error::msg(e.0))?;
    if revisions.is_empty() {
        return Err(Error::msg("explicit OCI image set required"));
    }
    for (reference, revision) in revisions {
        let hex = reference.strip_prefix("sha256:").unwrap_or("");
        if !reference.starts_with("sha256:")
            || !is_digest(hex)
            || (!revision.is_empty() && !is_revision(revision))
        {
            return Err(Error::msg("invalid OCI image selection"));
        }
    }
    Ok(want.to_string())
}

/// `build.InspectOCILayout`: verify the exact named image set and blobs.
pub fn inspect_oci_layout(
    dir: &str,
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<OciLayout, Error> {
    let want = validate_oci_layout_inputs(arch, revisions)?;
    let info =
        std::fs::symlink_metadata(dir).map_err(|_| Error::msg("real OCI layout directory required"))?;
    if !info.is_dir() {
        return Err(Error::msg("real OCI layout directory required"));
    }
    let root = Root::open(dir).map_err(|_| Error::msg("real OCI layout directory required"))?;
    let mut loader = LayoutLoader {
        root: &root,
        entries: BTreeMap::new(),
        json_bytes: 0,
    };
    for name in ["index.json", "oci-layout"] {
        loader.load(name)?;
    }
    let index = read_oci_index(&loader.entries)?;
    let images = inspect_layout_images(&mut loader, &index, &want, revisions)?;
    let mut files: BTreeMap<String, String> = BTreeMap::new();
    let mut total: u64 = 0;
    for (name, entry) in &loader.entries {
        files.insert(name.clone(), entry.hash.clone());
        total += entry.size as u64;
    }
    Ok(OciLayout {
        images,
        files,
        bytes: total,
    })
}

struct LayoutLoader<'a> {
    root: &'a Root,
    entries: BTreeMap<String, Blob>,
    json_bytes: i64,
}

impl LayoutLoader<'_> {
    fn load(&mut self, name: &str) -> Result<(), Error> {
        if self.entries.contains_key(name) {
            return Ok(());
        }
        let (data, size) = read_layout_entry(self.root, name)?;
        read_oci_blob(&mut self.entries, name, size, data, &mut self.json_bytes)
    }

    fn fetch(&mut self, d: &Descriptor) -> Result<Blob, Error> {
        if d.size < 0 || !d.urls.is_empty() {
            return Err(Error::msg("local bounded OCI descriptor required"));
        }
        let hex = d.digest.strip_prefix("sha256:").unwrap_or("");
        if !d.digest.starts_with("sha256:") || !is_digest(hex) {
            return Err(Error::msg("invalid OCI digest"));
        }
        let name = format!("blobs/sha256/{hex}");
        self.load(&name)?;
        match self.entries.get(&name) {
            Some(blob) if blob.size == d.size => Ok(blob.clone()),
            _ => Err(Error::msg("missing or wrong-size OCI blob")),
        }
    }

    fn inspect_image(
        &mut self,
        image: &Descriptor,
        want: &str,
        revision: &str,
    ) -> Result<BuildImage, Error> {
        let manifest_blob = self.fetch(image)?;
        let empty = Vec::new();
        let manifest = parse_oci_manifest(manifest_blob.data.as_ref().unwrap_or(&empty))?;
        for layer in &manifest.layers {
            match layer.media_type.as_str() {
                LAYER_TAR | LAYER_GZIP | LAYER_ZSTD => {}
                _ => return Err(Error::msg("unsupported OCI layer media type")),
            }
            self.fetch(layer)?;
        }
        let config_blob = self.fetch(&manifest.config)?;
        inspect_oci_config(
            &config_blob,
            &manifest.layers,
            want,
            revision,
            &image.digest,
            &manifest.config.digest,
        )
    }
}

fn inspect_layout_images(
    loader: &mut LayoutLoader<'_>,
    index: &[Descriptor],
    want: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, BuildImage>, Error> {
    if index.len() != revisions.len() {
        return Err(Error::msg("exact OCI image set required"));
    }
    let mut images: BTreeMap<String, BuildImage> = BTreeMap::new();
    for descriptor in index {
        let reference = descriptor
            .annotations
            .get("org.opencontainers.image.ref.name")
            .cloned()
            .unwrap_or_default();
        let revision = match revisions.get(&reference) {
            Some(revision) => revision.clone(),
            None => return Err(Error::msg("unexpected or duplicate OCI image reference")),
        };
        if images
            .get(&reference)
            .map(|image: &BuildImage| !image.config.is_empty())
            .unwrap_or(false)
            || descriptor.media_type != MANIFEST_TYPE
        {
            return Err(Error::msg("unexpected or duplicate OCI image reference"));
        }
        let image = loader.inspect_image(descriptor, want, &revision)?;
        if image.config != reference {
            return Err(Error::msg("OCI reference differs from config identity"));
        }
        images.insert(reference, image);
    }
    Ok(images)
}
