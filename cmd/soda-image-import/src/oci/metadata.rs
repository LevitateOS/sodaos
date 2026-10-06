use std::collections::HashMap;

use soda_json::JsonValue;

use super::super::{obj_fields, parse_json, t_field, t_int, t_string, t_string_list, t_string_map};

// ---------- OCI inspection (build/oci.go layout path) ----------

/// Mirrors Go's `build.Image`; identity fields are asserted by the tests.
#[allow(dead_code)]
#[derive(Debug, Clone, Default)]
pub(crate) struct OciImage {
    pub(crate) manifest: String,
    pub(crate) config: String,
    pub(crate) architecture: String,
    pub(crate) revision: String,
    pub(crate) source: String,
    pub(crate) base_name: String,
    pub(crate) base_digest: String,
}

#[derive(Debug, Clone, Default)]
pub(crate) struct OciDescriptor {
    pub(crate) digest: String,
    pub(crate) size: i64,
    pub(crate) media_type: String,
    pub(crate) urls: Vec<String>,
    pub(crate) annotations: HashMap<String, String>,
}

#[derive(Debug, Clone)]
pub(crate) struct OciBlobData {
    pub(crate) size: i64,
    /// Buffered only for small JSON blobs, like Go's `copyOCIBlob`.
    pub(crate) data: Option<Vec<u8>>,
}

fn decode_descriptor(value: &JsonValue) -> Result<OciDescriptor, String> {
    if *value == JsonValue::Null {
        return Ok(OciDescriptor::default());
    }
    if obj_fields(value).is_none() {
        return Err("OCI descriptor must be an object".to_string());
    }
    Ok(OciDescriptor {
        digest: t_string(value, "digest")?,
        size: t_int(value, "size")?,
        media_type: t_string(value, "mediaType")?,
        urls: t_string_list(value, "urls")?,
        annotations: t_string_map(value, "annotations")?,
    })
}

fn decode_descriptor_list(value: &JsonValue, name: &str) -> Result<Vec<OciDescriptor>, String> {
    match t_field(value, name) {
        None => Ok(Vec::new()),
        Some(JsonValue::Array(items)) => {
            let mut out = Vec::with_capacity(items.len());
            for item in items {
                out.push(decode_descriptor(item)?);
            }
            Ok(out)
        }
        Some(_) => Err(format!("field {name} must be a list")),
    }
}

pub(crate) struct OciManifestData {
    pub(crate) config: OciDescriptor,
    pub(crate) layers: Vec<OciDescriptor>,
}

pub(crate) fn parse_oci_manifest(data: &[u8]) -> Result<OciManifestData, String> {
    let value = parse_json(data)?;
    let schema = t_int(&value, "schemaVersion")?;
    let media = t_string(&value, "mediaType")?;
    let config = match t_field(&value, "config") {
        None => OciDescriptor::default(),
        Some(c) => decode_descriptor(c)?,
    };
    let layers = decode_descriptor_list(&value, "layers")?;
    if schema != 2
        || (!media.is_empty() && media != "application/vnd.oci.image.manifest.v1+json")
        || config.media_type != "application/vnd.oci.image.config.v1+json"
    {
        return Err("invalid OCI image manifest".to_string());
    }
    Ok(OciManifestData { config, layers })
}

pub(crate) struct OciConfigData {
    pub(crate) os: String,
    pub(crate) arch: String,
    pub(crate) rootfs_type: String,
    pub(crate) diff_ids: Vec<String>,
    pub(crate) labels: HashMap<String, String>,
}

pub(crate) fn parse_oci_config(data: &[u8]) -> Result<OciConfigData, String> {
    let value = parse_json(data)?;
    let os = t_string(&value, "os")?;
    let arch = t_string(&value, "architecture")?;
    let (rootfs_type, diff_ids) = match t_field(&value, "rootfs") {
        None => (String::new(), Vec::new()),
        Some(rootfs) => {
            if obj_fields(rootfs).is_none() {
                return Err("field rootfs must be an object".to_string());
            }
            (
                t_string(rootfs, "type")?,
                t_string_list(rootfs, "diff_ids")?,
            )
        }
    };
    let labels = match t_field(&value, "config") {
        None => HashMap::new(),
        Some(config) => {
            if obj_fields(config).is_none() {
                return Err("field config must be an object".to_string());
            }
            t_string_map(config, "Labels")?
        }
    };
    Ok(OciConfigData {
        os,
        arch,
        rootfs_type,
        diff_ids,
        labels,
    })
}

pub(crate) fn read_oci_index(
    entries: &HashMap<String, OciBlobData>,
) -> Result<Vec<OciDescriptor>, String> {
    let layout_data = entries.get("oci-layout").and_then(|b| b.data.as_ref());
    let version = match layout_data {
        Some(data) => {
            let value = parse_json(data).map_err(|_| "missing OCI layout".to_string())?;
            t_string(&value, "imageLayoutVersion").map_err(|_| "missing OCI layout".to_string())?
        }
        None => return Err("missing OCI layout".to_string()),
    };
    if version != "1.0.0" {
        return Err("missing OCI layout".to_string());
    }
    let index_data = entries.get("index.json").and_then(|b| b.data.as_ref());
    let value = match index_data {
        Some(data) => parse_json(data).map_err(|_| "valid OCI index required".to_string())?,
        None => return Err("valid OCI index required".to_string()),
    };
    let schema =
        t_int(&value, "schemaVersion").map_err(|_| "valid OCI index required".to_string())?;
    let media =
        t_string(&value, "mediaType").map_err(|_| "valid OCI index required".to_string())?;
    let manifests = decode_descriptor_list(&value, "manifests")
        .map_err(|_| "valid OCI index required".to_string())?;
    if schema != 2 || (!media.is_empty() && media != "application/vnd.oci.image.index.v1+json") {
        return Err("valid OCI index required".to_string());
    }
    Ok(manifests)
}
