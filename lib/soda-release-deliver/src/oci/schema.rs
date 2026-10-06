use std::collections::BTreeMap;

use sha2::{Digest as _, Sha256};
use soda_json::JsonValue;

use crate::buildx::{is_digest, Image as BuildImage};
use crate::jsonx::{as_i64, parse_lenient, Soft};
use crate::Error;

use super::{Blob, CONFIG_TYPE, INDEX_TYPE, LAYER_GZIP, LAYER_TAR, LAYER_ZSTD, MANIFEST_TYPE};

#[derive(Debug, Clone, Default)]
pub(super) struct Descriptor {
    pub(super) digest: String,
    pub(super) size: i64,
    pub(super) media_type: String,
    pub(super) urls: Vec<String>,
    pub(super) annotations: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Default)]
pub(super) struct OciManifest {
    pub(super) config: Descriptor,
    pub(super) layers: Vec<Descriptor>,
}

// ---------------------------------------------------------------------------
// Small decoders (encoding/json semantics: lenient, last-wins)
// ---------------------------------------------------------------------------

fn decode_descriptor(value: &JsonValue) -> Result<Descriptor, crate::jsonx::DecodeError> {
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
                _ => return Err(crate::jsonx::DecodeError),
            }
        }
    }
    if let Some(entries) = soft
        .object("annotations")?
        .and_then(|o| o.entries().map(|e| e.to_vec()))
    {
        for (key, item) in &entries {
            match item {
                JsonValue::Str(s) => {
                    descriptor.annotations.insert(key.clone(), s.clone());
                }
                _ => return Err(crate::jsonx::DecodeError),
            }
        }
    }
    Ok(descriptor)
}

pub(super) fn read_oci_index(entries: &BTreeMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
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
    let version = as_i64(
        soft.integer("schemaVersion")
            .map_err(|_| Error::msg("valid OCI index required"))?
            .unwrap_or(0),
    )
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
            manifests
                .push(decode_descriptor(item).map_err(|_| Error::msg("valid OCI index required"))?);
        }
    }
    Ok(manifests)
}

pub(super) fn parse_oci_manifest(data: &[u8]) -> Result<OciManifest, Error> {
    let value = parse_lenient(data).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let soft = Soft::new(&value).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let invalid = || Error::msg("invalid OCI image manifest");
    let version = soft
        .integer("schemaVersion")
        .map_err(|_| invalid())?
        .unwrap_or(0);
    let media = soft
        .string("mediaType")
        .map_err(|_| invalid())?
        .unwrap_or_default();
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
        rootfs_type = rootfs
            .string("type")
            .map_err(|_| invalid())?
            .unwrap_or_default();
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
        if let Some(entries) = config
            .object("Labels")
            .map_err(|_| invalid())?
            .and_then(|o| o.entries().map(|e| e.to_vec()))
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
        os: soft
            .string("os")
            .map_err(|_| invalid())?
            .unwrap_or_default(),
        arch: soft
            .string("architecture")
            .map_err(|_| invalid())?
            .unwrap_or_default(),
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

pub(super) fn read_oci_blob(
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

pub(super) fn fetch_oci_blob(
    entries: &BTreeMap<String, Blob>,
    d: &Descriptor,
) -> Result<Blob, Error> {
    if d.size < 0 || !d.urls.is_empty() {
        return Err(Error::msg("local bounded OCI descriptor required"));
    }
    let hex = d.digest.strip_prefix("sha256:").unwrap_or("");
    if !d.digest.starts_with("sha256:") || !is_digest(hex) {
        return Err(Error::msg("invalid OCI digest"));
    }
    let name = format!("blobs/sha256/{hex}");
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
) -> Result<(), Error> {
    for layer in layers {
        match layer.media_type.as_str() {
            LAYER_TAR | LAYER_GZIP | LAYER_ZSTD => {}
            _ => return Err(Error::msg("unsupported OCI layer media type")),
        }
        fetch_oci_blob(entries, layer)?;
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

pub(super) fn inspect_oci_config(
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

pub(super) fn inspect_oci_image(
    entries: &BTreeMap<String, Blob>,
    image: &Descriptor,
    want: &str,
    revision: &str,
) -> Result<BuildImage, Error> {
    let manifest_blob = fetch_oci_blob(entries, image)?;
    let empty = Vec::new();
    let manifest = parse_oci_manifest(manifest_blob.data.as_ref().unwrap_or(&empty))?;
    validate_oci_layers(entries, &manifest.layers)?;
    let config_blob = fetch_oci_blob(entries, &manifest.config)?;
    inspect_oci_config(
        &config_blob,
        &manifest.layers,
        want,
        revision,
        &image.digest,
        &manifest.config.digest,
    )
}
