use serde::de::{DeserializeOwned, Error as DeError, IgnoredAny, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::collections::BTreeMap;
use std::fmt;

use sha2::{Digest as _, Sha256};

use crate::buildx::{is_digest, Image as BuildImage};
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
struct IndexDocument {
    schema_version: i64,
    media_type: String,
    manifests: Vec<Descriptor>,
}

#[derive(Debug, Clone, Default)]
struct ManifestDocument {
    schema_version: i64,
    media_type: String,
    config: Descriptor,
    layers: Vec<Descriptor>,
}

#[derive(Debug, Clone, Default)]
struct RootfsDocument {
    rootfs_type: String,
    diff_ids: Vec<String>,
}

#[derive(Debug, Clone, Default)]
struct ImageConfigDocument {
    labels: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Default)]
struct ConfigDocument {
    os: String,
    architecture: String,
    rootfs: RootfsDocument,
    config: ImageConfigDocument,
}

#[derive(Default)]
struct LayoutDocument {
    image_layout_version: String,
}

fn decode_raw<T: DeserializeOwned + Default, E: DeError>(
    raw: Option<Box<serde_json::value::RawValue>>,
) -> Result<T, E> {
    let Some(raw) = raw else {
        return Ok(T::default());
    };
    serde_json::from_str::<Option<T>>(raw.get())
        .map_err(E::custom)
        .map(Option::unwrap_or_default)
}

#[derive(Default)]
struct RawI64(i64);
impl<'de> Deserialize<'de> for RawI64 {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        crate::json_serde::null_i64(d).map(RawI64)
    }
}

impl<'de> Deserialize<'de> for Descriptor {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = Descriptor;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI descriptor")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<Descriptor, M::Error> {
                let (mut digest, mut size, mut media_type, mut urls, mut annotations) =
                    (None, None, None, None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "digest" => {
                            digest = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "size" => size = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "mediaType" => {
                            media_type = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "urls" => urls = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "annotations" => {
                            annotations = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(Descriptor {
                    digest: decode_raw(digest)?,
                    size: decode_raw::<RawI64, M::Error>(size)?.0,
                    media_type: decode_raw(media_type)?,
                    urls: decode_raw(urls)?,
                    annotations: decode_raw(annotations)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for IndexDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = IndexDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI index object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<IndexDocument, M::Error> {
                let (mut version, mut media, mut manifests) = (None, None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "schemaVersion" => {
                            version = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "mediaType" => {
                            media = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "manifests" => {
                            manifests = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(IndexDocument {
                    schema_version: decode_raw::<RawI64, M::Error>(version)?.0,
                    media_type: decode_raw(media)?,
                    manifests: decode_raw(manifests)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for ManifestDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ManifestDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI manifest object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<ManifestDocument, M::Error> {
                let (mut version, mut media, mut config, mut layers) = (None, None, None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "schemaVersion" => {
                            version = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "mediaType" => {
                            media = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "config" => {
                            config = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "layers" => {
                            layers = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(ManifestDocument {
                    schema_version: decode_raw::<RawI64, M::Error>(version)?.0,
                    media_type: decode_raw(media)?,
                    config: decode_raw(config)?,
                    layers: decode_raw(layers)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for RootfsDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RootfsDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI rootfs object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<RootfsDocument, M::Error> {
                let (mut kind, mut ids) = (None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "type" => kind = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "diff_ids" => {
                            ids = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(RootfsDocument {
                    rootfs_type: decode_raw(kind)?,
                    diff_ids: decode_raw(ids)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for ImageConfigDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ImageConfigDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI config object")
            }
            fn visit_map<M: MapAccess<'de>>(
                self,
                mut m: M,
            ) -> Result<ImageConfigDocument, M::Error> {
                let mut labels = None;
                while let Some(key) = m.next_key::<String>()? {
                    if key == "Labels" {
                        labels = Some(m.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        m.next_value::<IgnoredAny>()?;
                    }
                }
                Ok(ImageConfigDocument {
                    labels: decode_raw(labels)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for ConfigDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ConfigDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI image config object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<ConfigDocument, M::Error> {
                let (mut os, mut arch, mut rootfs, mut config) = (None, None, None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "os" => os = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "architecture" => {
                            arch = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "rootfs" => {
                            rootfs = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "config" => {
                            config = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(ConfigDocument {
                    os: decode_raw(os)?,
                    architecture: decode_raw(arch)?,
                    rootfs: decode_raw(rootfs)?,
                    config: decode_raw(config)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for LayoutDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = LayoutDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI layout object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<LayoutDocument, M::Error> {
                let mut version = None;
                while let Some(key) = m.next_key::<String>()? {
                    if key == "imageLayoutVersion" {
                        version = Some(m.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        m.next_value::<IgnoredAny>()?;
                    }
                }
                Ok(LayoutDocument {
                    image_layout_version: decode_raw(version)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Debug, Clone, Default)]
pub(super) struct OciManifest {
    pub(super) config: Descriptor,
    pub(super) layers: Vec<Descriptor>,
}

pub(super) fn read_oci_index(entries: &BTreeMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
    let invalid = || Error::msg("valid OCI index required");
    let layout_data = entries
        .get("oci-layout")
        .and_then(|b| b.data.as_deref())
        .ok_or_else(|| Error::msg("missing OCI layout"))?;
    let layout: LayoutDocument =
        serde_json::from_slice(layout_data).map_err(|_| Error::msg("missing OCI layout"))?;
    if layout.image_layout_version != "1.0.0" {
        return Err(Error::msg("missing OCI layout"));
    }
    let index_data = entries
        .get("index.json")
        .and_then(|b| b.data.as_deref())
        .ok_or_else(invalid)?;
    let index: IndexDocument = serde_json::from_slice(index_data).map_err(|_| invalid())?;
    if index.schema_version != 2 || (!index.media_type.is_empty() && index.media_type != INDEX_TYPE)
    {
        return Err(invalid());
    }
    Ok(index.manifests)
}

pub(super) fn parse_oci_manifest(data: &[u8]) -> Result<OciManifest, Error> {
    let invalid = || Error::msg("invalid OCI image manifest");
    let manifest: ManifestDocument = serde_json::from_slice(data).map_err(|_| invalid())?;
    if manifest.schema_version != 2
        || (!manifest.media_type.is_empty() && manifest.media_type != MANIFEST_TYPE)
    {
        return Err(invalid());
    }
    if manifest.config.media_type != CONFIG_TYPE {
        return Err(invalid());
    }
    Ok(OciManifest {
        config: manifest.config,
        layers: manifest.layers,
    })
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
    let config: ConfigDocument = serde_json::from_slice(data).map_err(|_| invalid())?;
    Ok(OciConfig {
        os: config.os,
        arch: config.architecture,
        rootfs_type: config.rootfs.rootfs_type,
        diff_ids: config.rootfs.diff_ids,
        labels: config.config.labels,
    })
}

fn is_json(data: &[u8]) -> bool {
    let mut decoder = serde_json::Deserializer::from_slice(data);
    serde::de::IgnoredAny::deserialize(&mut decoder).is_ok() && decoder.end().is_ok()
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

#[cfg(test)]
mod json_admission_tests {
    use super::{Descriptor, IndexDocument};

    #[test]
    fn oci_raw_slots_validate_only_the_last_known_value() {
        let descriptor: Descriptor = serde_json::from_str(
            r#"{"size":1.5,"size":-0,"digest":false,"digest":"sha256:abc","urls":2,"urls":[]}"#,
        )
        .unwrap();
        assert_eq!(descriptor.size, 0);
        assert_eq!(descriptor.digest, "sha256:abc");
        assert!(descriptor.urls.is_empty());

        assert!(serde_json::from_str::<Descriptor>(r#"{"size":-0.0}"#).is_err());
        assert!(serde_json::from_str::<Descriptor>(r#"{"size":1e0}"#).is_err());
        assert!(serde_json::from_str::<Descriptor>(r#"{"size":9223372036854775808}"#).is_err());
        assert!(serde_json::from_str::<Descriptor>(r#"{"size":-0,"size":1.25}"#).is_err());

        let index: IndexDocument = serde_json::from_str(
            r#"{"schemaVersion":false,"schemaVersion":2,"manifests":null,"manifests":[{}]}"#,
        )
        .unwrap();
        assert_eq!(index.schema_version, 2);
        assert_eq!(index.manifests.len(), 1);
    }
}
