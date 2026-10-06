use std::collections::BTreeMap;

use soda_json::JsonValue;

use super::layout::{Blob, Descriptor};
use super::{CONFIG_MEDIA_TYPE, INDEX_MEDIA_TYPE, MANIFEST_MEDIA_TYPE};
use crate::errors::Error;
use crate::jsongo::parse;

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

pub(super) fn read_oci_index(entries: &BTreeMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
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

pub(super) struct OciManifestDoc {
    schema_version: i64,
    media_type: String,
    pub(super) config: Descriptor,
    pub(super) layers: Vec<Descriptor>,
}

pub(super) fn parse_oci_manifest(data: &[u8]) -> Result<OciManifestDoc, Error> {
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

pub(super) struct OciConfigDoc {
    pub(super) os: String,
    pub(super) arch: String,
    pub(super) rootfs_type: String,
    pub(super) diff_ids: Vec<String>,
    pub(super) labels: BTreeMap<String, String>,
}

pub(super) fn parse_oci_config(data: &[u8]) -> Result<OciConfigDoc, Error> {
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
