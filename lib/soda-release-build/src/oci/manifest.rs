//! OCI index/manifest/config identity: layout parsing, blob binding,
//! and rootfs/attribution validation.

use super::{
    Blob, Descriptor, Image, LoadBlobs, CONFIG_MEDIA_TYPE, INDEX_MEDIA_TYPE, LAYER_TAR,
    LAYER_TAR_GZIP, LAYER_TAR_ZSTD, MANIFEST_MEDIA_TYPE,
};
use crate::files::is_digest;
use crate::json_go::Fields;
use crate::Error;
use soda_json::JsonValue;
use std::collections::HashMap;

/// Parses the index; both single-image archives and multi-image layouts
/// share this gate.
pub(crate) fn read_oci_index(entries: &HashMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
    let layout_data = entries.get("oci-layout").and_then(|b| b.data.as_ref());
    let layout_ok = layout_data
        .and_then(|data| std::str::from_utf8(data).ok())
        .and_then(|text| JsonValue::parse(text).ok())
        .and_then(|v| Fields::of(&v).map(|f| f.string("imageLayoutVersion").unwrap_or_default()))
        == Some("1.0.0".to_string());
    if !layout_ok {
        return Err(Error::msg("missing OCI layout"));
    }
    let index_data = entries.get("index.json").and_then(|b| b.data.as_ref());
    let index_text = index_data
        .and_then(|data| std::str::from_utf8(data).ok())
        .ok_or_else(|| Error::msg("valid OCI index required"))?;
    let index_value =
        JsonValue::parse(index_text).map_err(|_| Error::msg("valid OCI index required"))?;
    let index = Fields::of(&index_value).ok_or_else(|| Error::msg("valid OCI index required"))?;
    let schema = index
        .int("schemaVersion")
        .map_err(|_| Error::msg("valid OCI index required"))?;
    let media = index.media_type();
    if schema != 2 || (!media.is_empty() && media != INDEX_MEDIA_TYPE) {
        return Err(Error::msg("valid OCI index required"));
    }
    let manifests = index
        .object_list("manifests")
        .map_err(|_| Error::msg("valid OCI index required"))?;
    manifests
        .iter()
        .map(Descriptor::decode)
        .collect::<Result<Vec<_>, _>>()
        .map_err(|_| Error::msg("valid OCI index required"))
}

trait MediaType {
    fn media_type(&self) -> String;
}

impl MediaType for Fields<'_> {
    fn media_type(&self) -> String {
        self.string("mediaType").unwrap_or_default()
    }
}

pub(super) struct OciManifest {
    pub config: Descriptor,
    pub layers: Vec<Descriptor>,
}

pub(super) fn parse_oci_manifest(data: &[u8]) -> Result<OciManifest, Error> {
    let text = std::str::from_utf8(data).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let value = JsonValue::parse(text).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let fields = Fields::of(&value).ok_or_else(|| Error::msg("invalid OCI image manifest"))?;
    let schema = fields
        .int("schemaVersion")
        .map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let media = fields.media_type();
    let config = fields
        .object("config")
        .map_err(|_| Error::msg("invalid OCI image manifest"))?
        .ok_or_else(|| Error::msg("invalid OCI image manifest"))?;
    let config =
        Descriptor::decode(&config).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    if schema != 2
        || (!media.is_empty() && media != MANIFEST_MEDIA_TYPE)
        || config.media_type != CONFIG_MEDIA_TYPE
    {
        return Err(Error::msg("invalid OCI image manifest"));
    }
    let layers = fields
        .object_list("layers")
        .map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let layers = layers
        .iter()
        .map(Descriptor::decode)
        .collect::<Result<Vec<_>, _>>()
        .map_err(|_| Error::msg("invalid OCI image manifest"))?;
    Ok(OciManifest { config, layers })
}

/// Fetches a blob by descriptor, loading it on demand for layouts.
pub(super) fn fetch_oci_blob(
    entries: &mut HashMap<String, Blob>,
    desc: &Descriptor,
    load: LoadBlobs,
) -> Result<(String, i64, Option<Vec<u8>>), Error> {
    if desc.size < 0 || !desc.urls.is_empty() {
        return Err(Error::msg("local bounded OCI descriptor required"));
    }
    let hex = match desc.digest.strip_prefix("sha256:") {
        Some(hex) if is_digest(hex) => hex,
        _ => return Err(Error::msg("invalid OCI digest")),
    };
    let name = format!("blobs/sha256/{hex}");
    load(entries, &name)?;
    match entries.get(&name) {
        Some(blob) if blob.size == desc.size => {
            Ok((blob.hash.clone(), blob.size, blob.data.clone()))
        }
        _ => Err(Error::msg("missing or wrong-size OCI blob")),
    }
}

fn validate_oci_layers(
    entries: &mut HashMap<String, Blob>,
    layers: &[Descriptor],
    load: LoadBlobs,
) -> Result<(), Error> {
    for layer in layers {
        match layer.media_type.as_str() {
            LAYER_TAR | LAYER_TAR_GZIP | LAYER_TAR_ZSTD => {}
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
    for (id, layer) in diff_ids.iter().zip(layers.iter()) {
        match id.strip_prefix("sha256:") {
            Some(hex) if is_digest(hex) => {}
            _ => return Err(Error::msg("invalid OCI diff ID")),
        }
        if layer.media_type == LAYER_TAR && id != &layer.digest {
            return Err(Error::msg("uncompressed OCI layer identity mismatch"));
        }
    }
    Ok(())
}

fn validate_oci_attribution(labels: &[(String, String)], want_revision: &str) -> Result<(), Error> {
    let label = |key: &str| {
        labels
            .iter()
            .rev()
            .find(|(k, _)| k == key)
            .map(|(_, v)| v.as_str())
            .unwrap_or("")
    };
    let rev = label("org.opencontainers.image.revision");
    if !want_revision.is_empty() && rev != want_revision {
        return Err(Error::msg("OCI source revision mismatch"));
    }
    if want_revision.is_empty() {
        return Ok(());
    }
    let base_digest = label("org.opencontainers.image.base.digest");
    let base_hex = base_digest.strip_prefix("sha256:").unwrap_or("");
    if label("org.opencontainers.image.source") != "https://github.com/LevitateOS/sodaos"
        || label("org.opencontainers.image.base.name").is_empty()
        || !base_digest.starts_with("sha256:")
        || !is_digest(base_hex)
    {
        return Err(Error::msg("soda image lacks source/base attribution"));
    }
    Ok(())
}

fn inspect_oci_config(
    config_data: &[u8],
    layers: &[Descriptor],
    want: &str,
    revision: &str,
    image_digest: &str,
    config_digest: &str,
) -> Result<Image, Error> {
    let text =
        std::str::from_utf8(config_data).map_err(|_| Error::msg("invalid OCI image config"))?;
    let value = JsonValue::parse(text).map_err(|_| Error::msg("invalid OCI image config"))?;
    let cfg = Fields::of(&value).ok_or_else(|| Error::msg("invalid OCI image config"))?;
    let rootfs = cfg
        .object("rootfs")
        .map_err(|_| Error::msg("invalid OCI image config"))?
        .ok_or_else(|| Error::msg("invalid OCI image config"))?;
    if rootfs
        .string("type")
        .map_err(|_| Error::msg("invalid OCI image config"))?
        != "layers"
    {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    let diff_ids = rootfs
        .string_list("diff_ids")
        .map_err(|_| Error::msg("invalid OCI image config"))?;
    validate_oci_rootfs(&diff_ids, layers)?;
    let os = cfg
        .string("os")
        .map_err(|_| Error::msg("invalid OCI image config"))?;
    let arch = cfg
        .string("architecture")
        .map_err(|_| Error::msg("invalid OCI image config"))?;
    // Compressed layer contents are not extracted here. Their blob
    // identities are checked; native import remains the proof of
    // decompression/rootfs use.
    if os != "linux" || arch != want {
        return Err(Error::msg(format!("OCI must be linux/{want}")));
    }
    let config_section = cfg
        .object("config")
        .map_err(|_| Error::msg("invalid OCI image config"))?;
    let labels = match config_section {
        Some(section) => section
            .string_map("Labels")
            .map_err(|_| Error::msg("invalid OCI image config"))?,
        None => Vec::new(),
    };
    validate_oci_attribution(&labels, revision)?;
    let label = |key: &str| {
        labels
            .iter()
            .rev()
            .find(|(k, _)| k == key)
            .map(|(_, v)| v.clone())
            .unwrap_or_default()
    };
    Ok(Image {
        manifest: image_digest.to_string(),
        config: config_digest.to_string(),
        architecture: arch,
        revision: label("org.opencontainers.image.revision"),
        source: label("org.opencontainers.image.source"),
        base_name: label("org.opencontainers.image.base.name"),
        base_digest: label("org.opencontainers.image.base.digest"),
    })
}

/// Verifies one image out of loaded blobs; `load` reads more for layouts.
pub(crate) fn inspect_oci_image(
    entries: &mut HashMap<String, Blob>,
    image: &Descriptor,
    want: &str,
    revision: &str,
    load: LoadBlobs,
) -> Result<Image, Error> {
    let (_, _, manifest_data) = fetch_oci_blob(entries, image, load)?;
    let manifest_data = manifest_data.unwrap_or_default();
    let manifest = parse_oci_manifest(&manifest_data)?;
    validate_oci_layers(entries, &manifest.layers, load)?;
    let (_, _, config_data) = fetch_oci_blob(entries, &manifest.config, load)?;
    let config_data = config_data.unwrap_or_default();
    inspect_oci_config(
        &config_data,
        &manifest.layers,
        want,
        revision,
        &image.digest,
        &manifest.config.digest,
    )
}
