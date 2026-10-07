//! OCI index/manifest/config identity: layout parsing, blob binding,
//! and rootfs/attribution validation.

use super::{
    Blob, Descriptor, Image, LoadBlobs, CONFIG_MEDIA_TYPE, INDEX_MEDIA_TYPE, LAYER_TAR,
    LAYER_TAR_GZIP, LAYER_TAR_ZSTD, MANIFEST_MEDIA_TYPE,
};
use crate::files::is_digest;
use crate::Error;
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::collections::HashMap;

type RawJson = Box<serde_json::value::RawValue>;

#[derive(Default)]
struct IntegerToken(i64);

impl<'de> Deserialize<'de> for IntegerToken {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let raw = RawJson::deserialize(deserializer)?;
        let value = crate::json_input::parse_integer_token(&raw)
            .ok_or_else(|| de::Error::custom("invalid integer token"))?;
        i64::try_from(value)
            .map(IntegerToken)
            .map_err(de::Error::custom)
    }
}

struct LayoutRecord {
    image_layout_version: String,
}
struct IndexRecord {
    schema_version: IntegerToken,
    media_type: String,
    manifests: Vec<Descriptor>,
}
struct ManifestRecord {
    schema_version: IntegerToken,
    media_type: String,
    config: Option<Descriptor>,
    layers: Vec<Descriptor>,
}
struct ConfigRecord {
    rootfs: Option<RootfsRecord>,
    os: String,
    architecture: String,
    config: Option<ConfigSection>,
}
struct RootfsRecord {
    kind: String,
    diff_ids: Vec<String>,
}
struct ConfigSection {
    labels: super::StringPairs,
}

macro_rules! serde_record {
    ($ty:ident, $visitor:ident, $expect:literal, { $($field:ident : $field_type:ty => $name:literal),+ $(,)? }) => {
        impl<'de> Deserialize<'de> for $ty {
            fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
                struct $visitor;
                impl<'de> Visitor<'de> for $visitor {
                    type Value = $ty;
                    fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result { f.write_str($expect) }
                    fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                        $(let mut $field: Option<RawJson> = None;)+
                        while let Some(key) = map.next_key::<String>()? {
                            let mut matched = false;
                            $(if key.eq_ignore_ascii_case($name) { $field = Some(map.next_value()?); matched = true; })+
                            if !matched { map.next_value::<de::IgnoredAny>()?; }
                        }
                        Ok($ty { $($field: super::raw_field::<$field_type, A::Error>($field)?,)+ })
                    }
                }
                deserializer.deserialize_map($visitor)
            }
        }
    };
}

serde_record!(LayoutRecord, LayoutVisitor, "OCI layout object", { image_layout_version: String => "imageLayoutVersion" });
serde_record!(IndexRecord, IndexVisitor, "OCI index object", {
    schema_version: IntegerToken => "schemaVersion", media_type: String => "mediaType", manifests: Vec<Descriptor> => "manifests"
});
serde_record!(ManifestRecord, ManifestVisitor, "OCI manifest object", {
    schema_version: IntegerToken => "schemaVersion", media_type: String => "mediaType", config: Option<Descriptor> => "config", layers: Vec<Descriptor> => "layers"
});
serde_record!(ConfigRecord, ConfigVisitor, "OCI config object", {
    rootfs: Option<RootfsRecord> => "rootfs", os: String => "os", architecture: String => "architecture", config: Option<ConfigSection> => "config"
});
serde_record!(RootfsRecord, RootfsVisitor, "OCI rootfs object", { kind: String => "type", diff_ids: Vec<String> => "diff_ids" });
serde_record!(ConfigSection, ConfigSectionVisitor, "OCI config section", { labels: super::StringPairs => "Labels" });

/// Parses the index; both single-image archives and multi-image layouts
/// share this gate.
pub(crate) fn read_oci_index(entries: &HashMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
    let layout_data = entries.get("oci-layout").and_then(|b| b.data.as_ref());
    let layout_ok = layout_data
        .and_then(|data| std::str::from_utf8(data).ok())
        .and_then(|text| serde_json::from_str::<LayoutRecord>(text).ok())
        .map(|layout| layout.image_layout_version == "1.0.0")
        .unwrap_or(false);
    if !layout_ok {
        return Err(Error::msg("missing OCI layout"));
    }
    let index_data = entries.get("index.json").and_then(|b| b.data.as_ref());
    let index_text = index_data
        .and_then(|data| std::str::from_utf8(data).ok())
        .ok_or_else(|| Error::msg("valid OCI index required"))?;
    let index: IndexRecord =
        serde_json::from_str(index_text).map_err(|_| Error::msg("valid OCI index required"))?;
    if index.schema_version.0 != 2
        || (!index.media_type.is_empty() && index.media_type != INDEX_MEDIA_TYPE)
    {
        return Err(Error::msg("valid OCI index required"));
    }
    Ok(index.manifests)
}

pub(super) struct OciManifest {
    pub config: Descriptor,
    pub layers: Vec<Descriptor>,
}

pub(super) fn parse_oci_manifest(data: &[u8]) -> Result<OciManifest, Error> {
    let text = std::str::from_utf8(data).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let record: ManifestRecord =
        serde_json::from_str(text).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let config = record
        .config
        .ok_or_else(|| Error::msg("invalid OCI image manifest"))?;
    if record.schema_version.0 != 2
        || (!record.media_type.is_empty() && record.media_type != MANIFEST_MEDIA_TYPE)
        || config.media_type != CONFIG_MEDIA_TYPE
    {
        return Err(Error::msg("invalid OCI image manifest"));
    }
    Ok(OciManifest {
        config,
        layers: record.layers,
    })
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
    let cfg: ConfigRecord =
        serde_json::from_str(text).map_err(|_| Error::msg("invalid OCI image config"))?;
    let rootfs = cfg
        .rootfs
        .ok_or_else(|| Error::msg("invalid OCI image config"))?;
    if rootfs.kind != "layers" {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    validate_oci_rootfs(&rootfs.diff_ids, layers)?;
    let os = cfg.os;
    let arch = cfg.architecture;
    // Compressed layer contents are not extracted here. Their blob
    // identities are checked; native import remains the proof of
    // decompression/rootfs use.
    if os != "linux" || arch != want {
        return Err(Error::msg(format!("OCI must be linux/{want}")));
    }
    let labels = cfg
        .config
        .map(|section| section.labels.0)
        .unwrap_or_default();
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

#[cfg(test)]
mod tests {
    use super::IndexRecord;

    #[test]
    fn schema_version_keeps_go_integer_tokens() {
        let record: IndexRecord = serde_json::from_str(r#"{"schemaVersion":-0}"#).unwrap();
        assert_eq!(record.schema_version.0, 0);
        for token in ["0.0", "0e0", "9223372036854775808", "-9223372036854775809"] {
            let body = format!(r#"{{"schemaVersion":{token}}}"#);
            assert!(
                serde_json::from_str::<IndexRecord>(&body).is_err(),
                "{token} must fail"
            );
        }
    }
}
