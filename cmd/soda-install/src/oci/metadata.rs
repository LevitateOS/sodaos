use std::collections::BTreeMap;

use serde::de::DeserializeOwned;
use serde::Deserialize;

use super::layout::{Blob, Descriptor};
use super::{CONFIG_MEDIA_TYPE, INDEX_MEDIA_TYPE, MANIFEST_MEDIA_TYPE};
use crate::errors::Error;

fn deserialize_utf8<T: DeserializeOwned>(data: &[u8], message: &str) -> Result<T, Error> {
    let text = std::str::from_utf8(data).map_err(|_| Error::msg(message))?;
    let mut deserializer = serde_json::Deserializer::from_str(text);
    let value = T::deserialize(&mut deserializer).map_err(|_| Error::msg(message))?;
    deserializer.end().map_err(|_| Error::msg(message))?;
    Ok(value)
}

#[derive(Deserialize)]
struct Layout {
    #[serde(rename = "imageLayoutVersion")]
    image_layout_version: String,
}

#[derive(Deserialize)]
struct Index {
    #[serde(rename = "schemaVersion")]
    schema_version: i64,
    #[serde(rename = "mediaType", default = "index_media_type")]
    media_type: String,
    manifests: Vec<Descriptor>,
}

#[derive(Deserialize)]
pub(super) struct OciManifestDoc {
    #[serde(rename = "schemaVersion")]
    schema_version: i64,
    #[serde(rename = "mediaType", default = "manifest_media_type")]
    media_type: String,
    pub(super) config: Descriptor,
    pub(super) layers: Vec<Descriptor>,
}

#[derive(Deserialize)]
pub(super) struct Rootfs {
    #[serde(rename = "type")]
    pub(super) kind: String,
    pub(super) diff_ids: Vec<String>,
}

#[derive(Deserialize)]
pub(super) struct ExecutionConfig {
    #[serde(rename = "Labels")]
    pub(super) labels: Option<std::collections::BTreeMap<String, String>>,
}

#[derive(Deserialize)]
pub(super) struct OciConfig {
    pub(super) os: String,
    pub(super) architecture: String,
    pub(super) rootfs: Rootfs,
    #[serde(default)]
    pub(super) config: Option<ExecutionConfig>,
}

fn index_media_type() -> String {
    INDEX_MEDIA_TYPE.to_string()
}

fn manifest_media_type() -> String {
    MANIFEST_MEDIA_TYPE.to_string()
}

pub(super) fn read_oci_index(entries: &BTreeMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
    let layout_data = entries
        .get("oci-layout")
        .and_then(|blob| blob.data.as_deref())
        .unwrap_or(b"");
    let layout: Layout = deserialize_utf8(layout_data, "missing OCI layout")?;
    if layout.image_layout_version != "1.0.0" {
        return Err(Error::msg("missing OCI layout"));
    }

    let index_data = entries
        .get("index.json")
        .and_then(|blob| blob.data.as_deref())
        .unwrap_or(b"");
    let index: Index = deserialize_utf8(index_data, "valid OCI index required")?;
    if index.schema_version != 2 || index.media_type != INDEX_MEDIA_TYPE {
        return Err(Error::msg("valid OCI index required"));
    }
    Ok(index.manifests)
}

pub(super) fn parse_oci_manifest(data: &[u8]) -> Result<OciManifestDoc, Error> {
    let manifest: OciManifestDoc = deserialize_utf8(data, "invalid OCI image manifest")?;
    if manifest.schema_version != 2
        || manifest.media_type != MANIFEST_MEDIA_TYPE
        || manifest.config.media_type != CONFIG_MEDIA_TYPE
    {
        return Err(Error::msg("invalid OCI image manifest"));
    }
    Ok(manifest)
}

pub(super) fn parse_oci_config(data: &[u8]) -> Result<OciConfig, Error> {
    deserialize_utf8(data, "invalid OCI image config")
}
