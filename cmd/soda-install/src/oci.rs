//! Shared-blob OCI layout verification ported from
//! `internal/release/build` (`oci_layout.go` plus the reachable half of
//! `oci.go`). Only the installer's `InspectOCILayout` path is ported: exact
//! image-set identity, blob hashing, and byte counts. Archive and layer
//! member inspection stay in Go with their callers.

use std::collections::BTreeMap;

use soda_json::JsonValue;

use self::layout::{load_layout_file, open_root, Blob, Descriptor, Loader};
pub use self::layout::{OciImage, OciLayout};
use crate::buildx;
use crate::errors::Error;
use crate::jsongo::parse;

const MANIFEST_MEDIA_TYPE: &str = "application/vnd.oci.image.manifest.v1+json";
const CONFIG_MEDIA_TYPE: &str = "application/vnd.oci.image.config.v1+json";
const INDEX_MEDIA_TYPE: &str = "application/vnd.oci.image.index.v1+json";
const LAYER_TAR: &str = "application/vnd.oci.image.layer.v1.tar";
const LAYER_GZIP: &str = "application/vnd.oci.image.layer.v1.tar+gzip";
const LAYER_ZSTD: &str = "application/vnd.oci.image.layer.v1.tar+zstd";

// ---------------------------------------------------------------------------
// Tolerant decode (index/layout): unknown fields ignored, only success or
// failure escapes through mapped errors.
// ---------------------------------------------------------------------------

fn soft_string(value: &JsonValue) -> Result<String, ()> {
    match value {
        JsonValue::Str(s) => Ok(s.clone()),
        JsonValue::Null => Ok(String::new()),
        _ => Err(()),
    }
}

fn soft_field<'a>(entries: &'a [(String, JsonValue)], name: &str) -> Option<&'a JsonValue> {
    // Go matches struct fields exactly first, then by ASCII/Unicode fold.
    // Index and descriptor keys are fixed camelCase; exact match wins and a
    // folded duplicate loses to it, per key order.
    let mut found: Option<&JsonValue> = None;
    for (key, value) in entries {
        if key == name || (key.len() == name.len() && key.eq_ignore_ascii_case(name)) {
            found = Some(value);
        }
    }
    found
}

fn soft_entries(value: &JsonValue) -> Result<&[(String, JsonValue)], ()> {
    match value {
        JsonValue::Object(entries) => Ok(entries),
        _ => Err(()),
    }
}

fn decode_descriptor_soft(value: &JsonValue) -> Result<Descriptor, ()> {
    let mut desc = Descriptor::default();
    if matches!(value, JsonValue::Null) {
        return Ok(desc);
    }
    let entries = soft_entries(value)?;
    if let Some(v) = soft_field(entries, "digest") {
        desc.digest = soft_string(v)?;
    }
    if let Some(v) = soft_field(entries, "size") {
        match v {
            JsonValue::Null => {}
            JsonValue::Number(_) => {
                desc.size = v
                    .as_integer()
                    .and_then(|n| i64::try_from(n).ok())
                    .ok_or(())?;
            }
            _ => return Err(()),
        }
    }
    if let Some(v) = soft_field(entries, "mediaType") {
        desc.media_type = soft_string(v)?;
    }
    if let Some(v) = soft_field(entries, "urls") {
        match v {
            JsonValue::Null => {}
            JsonValue::Array(items) => {
                for item in items {
                    desc.urls.push(soft_string(item)?);
                }
            }
            _ => return Err(()),
        }
    }
    if let Some(v) = soft_field(entries, "annotations") {
        match v {
            JsonValue::Null => {}
            JsonValue::Object(map) => {
                for (key, val) in map {
                    desc.annotations.insert(key.clone(), soft_string(val)?);
                }
            }
            _ => return Err(()),
        }
    }
    Ok(desc)
}

fn read_oci_index(entries: &BTreeMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
    let layout_data = entries
        .get("oci-layout")
        .and_then(|b| b.data.as_deref())
        .unwrap_or(b"");
    let layout_value = parse(layout_data).map_err(|_| Error::msg("missing OCI layout"))?;
    let version = soft_entries(&layout_value)
        .ok()
        .and_then(|fields| soft_field(fields, "imageLayoutVersion"))
        .and_then(|v| soft_string(v).ok())
        .unwrap_or_default();
    if version != "1.0.0" {
        return Err(Error::msg("missing OCI layout"));
    }
    let index_data = entries
        .get("index.json")
        .and_then(|b| b.data.as_deref())
        .unwrap_or(b"");
    let index_value = parse(index_data).map_err(|_| Error::msg("valid OCI index required"))?;
    let fields = soft_entries(&index_value).map_err(|_| Error::msg("valid OCI index required"))?;
    let schema = match soft_field(fields, "schemaVersion") {
        None | Some(JsonValue::Null) => 0i64,
        Some(v @ JsonValue::Number(_)) => v
            .as_integer()
            .and_then(|n| i64::try_from(n).ok())
            .ok_or_else(|| Error::msg("valid OCI index required"))?,
        Some(_) => return Err(Error::msg("valid OCI index required")),
    };
    let media = match soft_field(fields, "mediaType") {
        None | Some(JsonValue::Null) => String::new(),
        Some(v) => soft_string(v).map_err(|_| Error::msg("valid OCI index required"))?,
    };
    if schema != 2 || (!media.is_empty() && media != INDEX_MEDIA_TYPE) {
        return Err(Error::msg("valid OCI index required"));
    }
    let manifests = match soft_field(fields, "manifests") {
        None | Some(JsonValue::Null) => Vec::new(),
        Some(JsonValue::Array(items)) => items
            .iter()
            .map(decode_descriptor_soft)
            .collect::<Result<Vec<_>, _>>()
            .map_err(|_| Error::msg("valid OCI index required"))?,
        Some(_) => return Err(Error::msg("valid OCI index required")),
    };
    Ok(manifests)
}

// ---------------------------------------------------------------------------
// Manifest/config decode: Go `UnmarshalTypeError` strings escape raw, so the
// decoder reproduces them exactly.
// ---------------------------------------------------------------------------

fn json_type_name(value: &JsonValue) -> String {
    match value {
        JsonValue::Null => "null".to_string(),
        JsonValue::Bool(_) => "bool".to_string(),
        JsonValue::Str(_) => "string".to_string(),
        JsonValue::Number(raw) => format!("number {raw}"),
        JsonValue::Array(_) => "array".to_string(),
        JsonValue::Object(_) => "object".to_string(),
    }
}

fn type_error(value: &JsonValue, go_type: &str) -> Error {
    Error::msg(format!(
        "json: cannot unmarshal {} into Go value of type {go_type}",
        json_type_name(value)
    ))
}

fn hard_string(value: &JsonValue) -> Result<String, Error> {
    match value {
        JsonValue::Str(s) => Ok(s.clone()),
        JsonValue::Null => Ok(String::new()),
        _ => Err(type_error(value, "string")),
    }
}

fn hard_int(value: &JsonValue) -> Result<i64, Error> {
    match value {
        JsonValue::Null => Ok(0),
        v @ JsonValue::Number(_) => v
            .as_integer()
            .and_then(|n| i64::try_from(n).ok())
            .ok_or_else(|| type_error(value, "int")),
        _ => Err(type_error(value, "int")),
    }
}

fn hard_int64(value: &JsonValue) -> Result<i64, Error> {
    match value {
        JsonValue::Null => Ok(0),
        v @ JsonValue::Number(_) => v
            .as_integer()
            .and_then(|n| i64::try_from(n).ok())
            .ok_or_else(|| type_error(value, "int64")),
        _ => Err(type_error(value, "int64")),
    }
}

fn hard_field<'a>(entries: &'a [(String, JsonValue)], name: &str) -> Option<&'a JsonValue> {
    soft_field(entries, name)
}

fn decode_descriptor_hard(value: &JsonValue) -> Result<Descriptor, Error> {
    let mut desc = Descriptor::default();
    if matches!(value, JsonValue::Null) {
        return Ok(desc);
    }
    let entries = match value {
        JsonValue::Object(entries) => entries,
        _ => return Err(type_error(value, "build.descriptor")),
    };
    if let Some(v) = hard_field(entries, "digest") {
        desc.digest = hard_string(v)?;
    }
    if let Some(v) = hard_field(entries, "size") {
        desc.size = hard_int64(v)?;
    }
    if let Some(v) = hard_field(entries, "mediaType") {
        desc.media_type = hard_string(v)?;
    }
    if let Some(v) = hard_field(entries, "urls") {
        match v {
            JsonValue::Null => {}
            JsonValue::Array(items) => {
                for item in items {
                    desc.urls.push(hard_string(item)?);
                }
            }
            _ => return Err(type_error(v, "[]string")),
        }
    }
    if let Some(v) = hard_field(entries, "annotations") {
        match v {
            JsonValue::Null => {}
            JsonValue::Object(map) => {
                for (key, val) in map {
                    desc.annotations.insert(key.clone(), hard_string(val)?);
                }
            }
            _ => return Err(type_error(v, "map[string]string")),
        }
    }
    Ok(desc)
}

struct OciManifestDoc {
    schema_version: i64,
    media_type: String,
    config: Descriptor,
    layers: Vec<Descriptor>,
}

fn parse_oci_manifest(data: &[u8]) -> Result<OciManifestDoc, Error> {
    let value = parse(data).map_err(|_| Error::msg("unexpected end of JSON input"))?;
    if matches!(value, JsonValue::Null) {
        return Ok(OciManifestDoc {
            schema_version: 0,
            media_type: String::new(),
            config: Descriptor::default(),
            layers: Vec::new(),
        });
    }
    let entries = match &value {
        JsonValue::Object(entries) => entries,
        _ => return Err(type_error(&value, "build.ociManifest")),
    };
    let mut doc = OciManifestDoc {
        schema_version: 0,
        media_type: String::new(),
        config: Descriptor::default(),
        layers: Vec::new(),
    };
    if let Some(v) = hard_field(entries, "schemaVersion") {
        doc.schema_version = hard_int(v)?;
    }
    if let Some(v) = hard_field(entries, "mediaType") {
        doc.media_type = hard_string(v)?;
    }
    if let Some(v) = hard_field(entries, "config") {
        doc.config = decode_descriptor_hard(v)?;
    }
    if let Some(v) = hard_field(entries, "layers") {
        match v {
            JsonValue::Null => {}
            JsonValue::Array(items) => {
                for item in items {
                    doc.layers.push(decode_descriptor_hard(item)?);
                }
            }
            _ => return Err(type_error(v, "[]build.descriptor")),
        }
    }
    if doc.schema_version != 2
        || (!doc.media_type.is_empty() && doc.media_type != MANIFEST_MEDIA_TYPE)
        || doc.config.media_type != CONFIG_MEDIA_TYPE
    {
        return Err(Error::msg("invalid OCI image manifest"));
    }
    Ok(doc)
}

struct OciConfigDoc {
    os: String,
    arch: String,
    rootfs_type: String,
    diff_ids: Vec<String>,
    labels: BTreeMap<String, String>,
}

fn parse_oci_config(data: &[u8]) -> Result<OciConfigDoc, Error> {
    let value = parse(data).map_err(|_| Error::msg("unexpected end of JSON input"))?;
    let mut doc = OciConfigDoc {
        os: String::new(),
        arch: String::new(),
        rootfs_type: String::new(),
        diff_ids: Vec::new(),
        labels: BTreeMap::new(),
    };
    if matches!(value, JsonValue::Null) {
        return Ok(doc);
    }
    let entries = match &value {
        JsonValue::Object(entries) => entries,
        _ => return Err(type_error(&value, "build.ociConfig")),
    };
    if let Some(v) = hard_field(entries, "os") {
        doc.os = hard_string(v)?;
    }
    if let Some(v) = hard_field(entries, "architecture") {
        doc.arch = hard_string(v)?;
    }
    if let Some(v) = hard_field(entries, "rootfs") {
        match v {
            JsonValue::Null => {}
            JsonValue::Object(rootfs) => {
                if let Some(t) = hard_field(rootfs, "type") {
                    doc.rootfs_type = hard_string(t)?;
                }
                if let Some(d) = hard_field(rootfs, "diff_ids") {
                    match d {
                        JsonValue::Null => {}
                        JsonValue::Array(items) => {
                            for item in items {
                                doc.diff_ids.push(hard_string(item)?);
                            }
                        }
                        _ => return Err(type_error(d, "[]string")),
                    }
                }
            }
            _ => return Err(type_error(v, "struct { Type string; DiffIDs []string }")),
        }
    }
    if let Some(v) = hard_field(entries, "config") {
        match v {
            JsonValue::Null => {}
            JsonValue::Object(config) => {
                if let Some(l) = hard_field(config, "Labels") {
                    match l {
                        JsonValue::Null => {}
                        JsonValue::Object(map) => {
                            for (key, val) in map {
                                doc.labels.insert(key.clone(), hard_string(val)?);
                            }
                        }
                        _ => return Err(type_error(l, "map[string]string")),
                    }
                }
            }
            _ => return Err(type_error(v, "struct { Labels map[string]string }")),
        }
    }
    Ok(doc)
}

// ---------------------------------------------------------------------------
// Image inspection.
// ---------------------------------------------------------------------------

fn fetch_oci_blob(loader: &mut Loader, desc: &Descriptor) -> Result<Blob, Error> {
    if desc.size < 0 || !desc.urls.is_empty() {
        return Err(Error::msg("local bounded OCI descriptor required"));
    }
    let hex = match desc.digest.strip_prefix("sha256:") {
        Some(hex) if buildx::digest(hex) => hex,
        _ => return Err(Error::msg("invalid OCI digest")),
    };
    let name = format!("blobs/sha256/{hex}");
    load_layout_file(loader, &name)?;
    match loader.entries.get(&name) {
        Some(blob) if blob.size == desc.size => Ok(blob.clone()),
        _ => Err(Error::msg("missing or wrong-size OCI blob")),
    }
}

fn validate_oci_layers(loader: &mut Loader, layers: &[Descriptor]) -> Result<(), Error> {
    for layer in layers {
        match layer.media_type.as_str() {
            LAYER_TAR | LAYER_GZIP | LAYER_ZSTD => {}
            _ => return Err(Error::msg("unsupported OCI layer media type")),
        }
        fetch_oci_blob(loader, layer)?;
    }
    Ok(())
}

fn validate_oci_rootfs(diff_ids: &[String], layers: &[Descriptor]) -> Result<(), Error> {
    if diff_ids.len() != layers.len() {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    for (i, id) in diff_ids.iter().enumerate() {
        let hex = match id.strip_prefix("sha256:") {
            Some(hex) if buildx::digest(hex) => hex,
            _ => return Err(Error::msg("invalid OCI diff ID")),
        };
        let _ = hex;
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
    let source_ok = labels
        .get("org.opencontainers.image.source")
        .map(|s| s.as_str())
        == Some("https://github.com/LevitateOS/sodaos");
    let base_name_ok = labels
        .get("org.opencontainers.image.base.name")
        .map(|s| !s.is_empty())
        .unwrap_or(false);
    let base_digest_ok = match base_digest.strip_prefix("sha256:") {
        Some(hex) => buildx::digest(hex),
        None => false,
    };
    if !source_ok || !base_name_ok || !base_digest_ok {
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
) -> Result<OciImage, Error> {
    let data = config_blob.data.as_deref().unwrap_or(b"");
    let cfg = parse_oci_config(data)?;
    if cfg.rootfs_type != "layers" {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    validate_oci_rootfs(&cfg.diff_ids, layers)?;
    // Compressed layer contents are not extracted here. Their blob identities
    // are checked; native import remains the proof of decompression/rootfs use.
    if cfg.os != "linux" || cfg.arch != want {
        return Err(Error::msg(format!("OCI must be linux/{want}")));
    }
    validate_oci_attribution(&cfg.labels, revision)?;
    Ok(OciImage {
        manifest: image_digest.to_string(),
        config: config_digest.to_string(),
        architecture: cfg.arch,
        revision: cfg
            .labels
            .get("org.opencontainers.image.revision")
            .cloned()
            .unwrap_or_default(),
        source: cfg
            .labels
            .get("org.opencontainers.image.source")
            .cloned()
            .unwrap_or_default(),
        base_name: cfg
            .labels
            .get("org.opencontainers.image.base.name")
            .cloned()
            .unwrap_or_default(),
        base_digest: cfg
            .labels
            .get("org.opencontainers.image.base.digest")
            .cloned()
            .unwrap_or_default(),
    })
}

fn inspect_oci_image(
    loader: &mut Loader,
    image: &Descriptor,
    want: &str,
    revision: &str,
) -> Result<OciImage, Error> {
    let manifest_blob = fetch_oci_blob(loader, image)?;
    let manifest = parse_oci_manifest(manifest_blob.data.as_deref().unwrap_or(b""))?;
    validate_oci_layers(loader, &manifest.layers)?;
    let config_blob = fetch_oci_blob(loader, &manifest.config)?;
    inspect_oci_config(
        &config_blob,
        &manifest.layers,
        want,
        revision,
        &image.digest,
        &manifest.config.digest,
    )
}

fn validate_oci_layout_inputs(
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<String, Error> {
    let want = buildx::oci_architecture(arch)?;
    if revisions.is_empty() {
        return Err(Error::msg("explicit OCI image set required"));
    }
    for (reference, revision) in revisions {
        let digest_ok = match reference.strip_prefix("sha256:") {
            Some(hex) => buildx::digest(hex),
            None => false,
        };
        if !digest_ok || (!revision.is_empty() && !buildx::revision(revision)) {
            return Err(Error::msg("invalid OCI image selection"));
        }
    }
    Ok(want)
}

fn inspect_layout_descriptor(
    loader: &mut Loader,
    desc: &Descriptor,
    want: &str,
    revisions: &BTreeMap<String, String>,
    images: &BTreeMap<String, OciImage>,
) -> Result<(String, OciImage), Error> {
    let reference = desc
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
        .map(|image| !image.config.is_empty())
        .unwrap_or(false)
        || desc.media_type != MANIFEST_MEDIA_TYPE
    {
        return Err(Error::msg("unexpected or duplicate OCI image reference"));
    }
    let image = inspect_oci_image(loader, desc, want, &revision)?;
    if image.config != reference {
        return Err(Error::msg("OCI reference differs from config identity"));
    }
    Ok((reference, image))
}

fn inspect_layout_images(
    loader: &mut Loader,
    index: &[Descriptor],
    want: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, OciImage>, Error> {
    if index.len() != revisions.len() {
        return Err(Error::msg("exact OCI image set required"));
    }
    let mut images = BTreeMap::new();
    for desc in index {
        let (reference, image) = inspect_layout_descriptor(loader, desc, want, revisions, &images)?;
        images.insert(reference, image);
    }
    Ok(images)
}

/// Verifies the exact named image set and its local blobs without running
/// image code. References are immutable config IDs, as in our OCI exports.
pub fn inspect_oci_layout(
    dir: &str,
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<OciLayout, Error> {
    let want = validate_oci_layout_inputs(arch, revisions)?;
    let root = open_root(dir)?;
    let mut loader = Loader {
        root,
        entries: BTreeMap::new(),
        json_bytes: 0,
    };
    for name in ["index.json", "oci-layout"] {
        load_layout_file(&mut loader, name)?;
    }
    let index = read_oci_index(&loader.entries)?;
    let images = inspect_layout_images(&mut loader, &index, &want, revisions)?;
    let mut files = BTreeMap::new();
    let mut bytes: u64 = 0;
    for (name, entry) in &loader.entries {
        files.insert(name.clone(), entry.hash.clone());
        bytes += entry.size as u64;
    }
    Ok(OciLayout {
        images,
        files,
        bytes,
    })
}

mod layout;

#[cfg(test)]
pub mod test_support;

#[cfg(test)]
mod tests;
