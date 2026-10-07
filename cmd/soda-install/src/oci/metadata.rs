use std::collections::BTreeMap;
use std::fmt;

use serde::de::{IgnoredAny, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;

use super::layout::{Blob, Descriptor};
use super::{CONFIG_MEDIA_TYPE, INDEX_MEDIA_TYPE, MANIFEST_MEDIA_TYPE};
use crate::errors::Error;

fn raw_document(data: &[u8]) -> Result<Box<RawValue>, Error> {
    let text = String::from_utf8_lossy(data);
    let mut de = serde_json::Deserializer::from_str(&text);
    let raw = Box::<RawValue>::deserialize(&mut de)
        .map_err(|_| Error::msg("unexpected end of JSON input"))?;
    de.end()
        .map_err(|_| Error::msg("unexpected end of JSON input"))?;
    Ok(raw)
}

fn is_null(raw: &RawValue) -> bool {
    raw.get() == "null"
}

fn type_error(raw: &RawValue, go_type: &str) -> Error {
    let token = raw.get().trim_start();
    let kind = match token.as_bytes().first() {
        Some(b'n') => "null".to_string(),
        Some(b't' | b'f') => "bool".to_string(),
        Some(b'"') => "string".to_string(),
        Some(b'[') => "array".to_string(),
        Some(b'{') => "object".to_string(),
        _ => format!("number {token}"),
    };
    Error::msg(format!(
        "json: cannot unmarshal {kind} into Go value of type {go_type}"
    ))
}

fn raw_string(raw: Option<&RawValue>) -> Result<String, Error> {
    match raw {
        None => Ok(String::new()),
        Some(raw) if is_null(raw) => Ok(String::new()),
        Some(raw) => {
            serde_json::from_str::<String>(raw.get()).map_err(|_| type_error(raw, "string"))
        }
    }
}

fn raw_integer(raw: Option<&RawValue>, ty: &str) -> Result<i64, Error> {
    match raw {
        None => Ok(0),
        Some(raw) if is_null(raw) => Ok(0),
        Some(raw) => raw
            .get()
            .parse::<i128>()
            .ok()
            .and_then(|n| i64::try_from(n).ok())
            .ok_or_else(|| type_error(raw, ty)),
    }
}

fn folded(key: &str, name: &str) -> bool {
    key.len() == name.len() && key.eq_ignore_ascii_case(name)
}

#[derive(Default)]
struct RawLayout {
    image_layout_version: Option<Box<RawValue>>,
}
impl<'de> Deserialize<'de> for RawLayout {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawLayout;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI layout object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<RawLayout, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut dto = RawLayout::default();
                while let Some(key) = map.next_key::<String>()? {
                    if folded(&key, "imageLayoutVersion") {
                        dto.image_layout_version = Some(map.next_value()?);
                    } else {
                        let _: IgnoredAny = map.next_value()?;
                    }
                }
                Ok(dto)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct RawIndex {
    schema_version: Option<Box<RawValue>>,
    media_type: Option<Box<RawValue>>,
    manifests: Option<Box<RawValue>>,
}
impl<'de> Deserialize<'de> for RawIndex {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawIndex;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI index object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<RawIndex, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut dto = RawIndex::default();
                while let Some(key) = map.next_key::<String>()? {
                    if folded(&key, "schemaVersion") {
                        dto.schema_version = Some(map.next_value()?);
                    } else if folded(&key, "mediaType") {
                        dto.media_type = Some(map.next_value()?);
                    } else if folded(&key, "manifests") {
                        dto.manifests = Some(map.next_value()?);
                    } else {
                        let _: IgnoredAny = map.next_value()?;
                    }
                }
                Ok(dto)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct RawDescriptor {
    digest: Option<Box<RawValue>>,
    size: Option<Box<RawValue>>,
    media_type: Option<Box<RawValue>>,
    urls: Option<Box<RawValue>>,
    annotations: Option<Box<RawValue>>,
}
impl<'de> Deserialize<'de> for RawDescriptor {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawDescriptor;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI descriptor object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<RawDescriptor, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut dto = RawDescriptor::default();
                while let Some(key) = map.next_key::<String>()? {
                    if folded(&key, "digest") {
                        dto.digest = Some(map.next_value()?);
                    } else if folded(&key, "size") {
                        dto.size = Some(map.next_value()?);
                    } else if folded(&key, "mediaType") {
                        dto.media_type = Some(map.next_value()?);
                    } else if folded(&key, "urls") {
                        dto.urls = Some(map.next_value()?);
                    } else if folded(&key, "annotations") {
                        dto.annotations = Some(map.next_value()?);
                    } else {
                        let _: IgnoredAny = map.next_value()?;
                    }
                }
                Ok(dto)
            }
        }
        d.deserialize_map(V)
    }
}

struct RawDescriptorList(Vec<Box<RawValue>>);
impl<'de> Deserialize<'de> for RawDescriptorList {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawDescriptorList;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an array of OCI descriptors")
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<RawDescriptorList, A::Error>
            where
                A: serde::de::SeqAccess<'de>,
            {
                let mut values = Vec::new();
                while let Some(value) = seq.next_element::<Box<RawValue>>()? {
                    values.push(value);
                }
                Ok(RawDescriptorList(values))
            }
        }
        d.deserialize_seq(V)
    }
}

#[derive(Default)]
struct RawManifest {
    schema_version: Option<Box<RawValue>>,
    media_type: Option<Box<RawValue>>,
    config: Option<Box<RawValue>>,
    layers: Option<Box<RawValue>>,
}
impl<'de> Deserialize<'de> for RawManifest {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawManifest;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI manifest object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<RawManifest, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut dto = RawManifest::default();
                while let Some(key) = map.next_key::<String>()? {
                    if folded(&key, "schemaVersion") {
                        dto.schema_version = Some(map.next_value()?);
                    } else if folded(&key, "mediaType") {
                        dto.media_type = Some(map.next_value()?);
                    } else if folded(&key, "config") {
                        dto.config = Some(map.next_value()?);
                    } else if folded(&key, "layers") {
                        dto.layers = Some(map.next_value()?);
                    } else {
                        let _: IgnoredAny = map.next_value()?;
                    }
                }
                Ok(dto)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct RawRootfs {
    kind: Option<Box<RawValue>>,
    diff_ids: Option<Box<RawValue>>,
}
impl<'de> Deserialize<'de> for RawRootfs {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawRootfs;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI rootfs object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<RawRootfs, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut dto = RawRootfs::default();
                while let Some(key) = map.next_key::<String>()? {
                    if folded(&key, "type") {
                        dto.kind = Some(map.next_value()?);
                    } else if folded(&key, "diff_ids") {
                        dto.diff_ids = Some(map.next_value()?);
                    } else {
                        let _: IgnoredAny = map.next_value()?;
                    }
                }
                Ok(dto)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct RawImageConfig {
    labels: Option<Box<RawValue>>,
}
impl<'de> Deserialize<'de> for RawImageConfig {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawImageConfig;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI config metadata object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<RawImageConfig, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut dto = RawImageConfig::default();
                while let Some(key) = map.next_key::<String>()? {
                    if folded(&key, "Labels") {
                        dto.labels = Some(map.next_value()?);
                    } else {
                        let _: IgnoredAny = map.next_value()?;
                    }
                }
                Ok(dto)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct RawConfig {
    os: Option<Box<RawValue>>,
    architecture: Option<Box<RawValue>>,
    rootfs: Option<Box<RawValue>>,
    config: Option<Box<RawValue>>,
}
impl<'de> Deserialize<'de> for RawConfig {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawConfig;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI image config object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<RawConfig, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut dto = RawConfig::default();
                while let Some(key) = map.next_key::<String>()? {
                    if folded(&key, "os") {
                        dto.os = Some(map.next_value()?);
                    } else if folded(&key, "architecture") {
                        dto.architecture = Some(map.next_value()?);
                    } else if folded(&key, "rootfs") {
                        dto.rootfs = Some(map.next_value()?);
                    } else if folded(&key, "config") {
                        dto.config = Some(map.next_value()?);
                    } else {
                        let _: IgnoredAny = map.next_value()?;
                    }
                }
                Ok(dto)
            }
        }
        d.deserialize_map(V)
    }
}

fn descriptor_fields(raw: &RawValue, go_type: &str) -> Result<RawDescriptor, Error> {
    if is_null(raw) {
        return Ok(RawDescriptor::default());
    }
    serde_json::from_str(raw.get()).map_err(|_| type_error(raw, go_type))
}

fn descriptor_strings(dto: &RawDescriptor) -> Result<Descriptor, Error> {
    let mut desc = Descriptor::default();
    desc.digest = raw_string(dto.digest.as_deref())?;
    desc.media_type = raw_string(dto.media_type.as_deref())?;
    desc.size = raw_integer(dto.size.as_deref(), "int64")?;
    if let Some(raw) = dto.urls.as_deref() {
        if !is_null(raw) {
            let values: Vec<Option<String>> =
                serde_json::from_str(raw.get()).map_err(|_| type_error(raw, "[]string"))?;
            desc.urls
                .extend(values.into_iter().map(Option::unwrap_or_default));
        }
    }
    if let Some(raw) = dto.annotations.as_deref() {
        if !is_null(raw) {
            let annotations: BTreeMap<String, Option<String>> = serde_json::from_str(raw.get())
                .map_err(|_| type_error(raw, "map[string]string"))?;
            desc.annotations = annotations
                .into_iter()
                .map(|(key, value)| (key, value.unwrap_or_default()))
                .collect();
        }
    }
    Ok(desc)
}

fn decode_descriptor_soft(raw: &RawValue) -> Result<Descriptor, ()> {
    let dto = descriptor_fields(raw, "build.descriptor").map_err(|_| ())?;
    descriptor_strings(&dto).map_err(|_| ())
}

fn decode_descriptor_hard(raw: &RawValue) -> Result<Descriptor, Error> {
    let dto = descriptor_fields(raw, "build.descriptor")?;
    descriptor_strings(&dto)
}

pub(super) fn read_oci_index(entries: &BTreeMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
    let layout_data = entries
        .get("oci-layout")
        .and_then(|blob| blob.data.as_deref())
        .unwrap_or(b"");
    let layout_raw = raw_document(layout_data).map_err(|_| Error::msg("missing OCI layout"))?;
    let layout: RawLayout =
        serde_json::from_str(layout_raw.get()).map_err(|_| Error::msg("missing OCI layout"))?;
    if raw_string(layout.image_layout_version.as_deref()).unwrap_or_default() != "1.0.0" {
        return Err(Error::msg("missing OCI layout"));
    }

    let index_data = entries
        .get("index.json")
        .and_then(|blob| blob.data.as_deref())
        .unwrap_or(b"");
    let index_raw = raw_document(index_data).map_err(|_| Error::msg("valid OCI index required"))?;
    let index: RawIndex = serde_json::from_str(index_raw.get())
        .map_err(|_| Error::msg("valid OCI index required"))?;
    if raw_integer(index.schema_version.as_deref(), "int64")
        .map_err(|_| Error::msg("valid OCI index required"))?
        != 2
    {
        return Err(Error::msg("valid OCI index required"));
    }
    let media = raw_string(index.media_type.as_deref())
        .map_err(|_| Error::msg("valid OCI index required"))?;
    if !media.is_empty() && media != INDEX_MEDIA_TYPE {
        return Err(Error::msg("valid OCI index required"));
    }
    let mut manifests = Vec::new();
    if let Some(raw) = index.manifests.as_deref() {
        if !is_null(raw) {
            let items: RawDescriptorList = serde_json::from_str(raw.get())
                .map_err(|_| type_error(raw, "[]build.descriptor"))
                .map_err(|_| Error::msg("valid OCI index required"))?;
            for item in items.0 {
                manifests.push(
                    decode_descriptor_soft(&item)
                        .map_err(|_| Error::msg("valid OCI index required"))?,
                );
            }
        }
    }
    Ok(manifests)
}

pub(super) struct OciManifestDoc {
    pub(super) config: Descriptor,
    pub(super) layers: Vec<Descriptor>,
}

pub(super) fn parse_oci_manifest(data: &[u8]) -> Result<OciManifestDoc, Error> {
    let raw = raw_document(data)?;
    if is_null(&raw) {
        return Ok(OciManifestDoc {
            config: Descriptor::default(),
            layers: Vec::new(),
        });
    }
    let dto: RawManifest =
        serde_json::from_str(raw.get()).map_err(|_| type_error(&raw, "build.ociManifest"))?;
    let schema_version = raw_integer(dto.schema_version.as_deref(), "int")?;
    let media_type = raw_string(dto.media_type.as_deref())?;
    let config = match dto.config.as_deref() {
        None => Descriptor::default(),
        Some(raw) => decode_descriptor_hard(raw)?,
    };
    let mut layers = Vec::new();
    if let Some(raw) = dto.layers.as_deref() {
        if !is_null(raw) {
            let items: RawDescriptorList = serde_json::from_str(raw.get())
                .map_err(|_| type_error(raw, "[]build.descriptor"))?;
            for item in items.0 {
                layers.push(decode_descriptor_hard(&item)?);
            }
        }
    }
    if schema_version != 2
        || (!media_type.is_empty() && media_type != MANIFEST_MEDIA_TYPE)
        || config.media_type != CONFIG_MEDIA_TYPE
    {
        return Err(Error::msg("invalid OCI image manifest"));
    }
    Ok(OciManifestDoc { config, layers })
}

pub(super) struct OciConfigDoc {
    pub(super) os: String,
    pub(super) arch: String,
    pub(super) rootfs_type: String,
    pub(super) diff_ids: Vec<String>,
    pub(super) labels: BTreeMap<String, String>,
}

pub(super) fn parse_oci_config(data: &[u8]) -> Result<OciConfigDoc, Error> {
    let raw = raw_document(data)?;
    let mut doc = OciConfigDoc {
        os: String::new(),
        arch: String::new(),
        rootfs_type: String::new(),
        diff_ids: Vec::new(),
        labels: BTreeMap::new(),
    };
    if is_null(&raw) {
        return Ok(doc);
    }
    let dto: RawConfig =
        serde_json::from_str(raw.get()).map_err(|_| type_error(&raw, "build.ociConfig"))?;
    doc.os = raw_string(dto.os.as_deref())?;
    doc.arch = raw_string(dto.architecture.as_deref())?;
    if let Some(raw) = dto.rootfs.as_deref() {
        if !is_null(raw) {
            let rootfs: RawRootfs = serde_json::from_str(raw.get())
                .map_err(|_| type_error(raw, "struct { Type string; DiffIDs []string }"))?;
            doc.rootfs_type = raw_string(rootfs.kind.as_deref())?;
            if let Some(ids) = rootfs.diff_ids.as_deref() {
                if !is_null(ids) {
                    let values: Vec<Option<String>> =
                        serde_json::from_str(ids.get()).map_err(|_| type_error(ids, "[]string"))?;
                    doc.diff_ids
                        .extend(values.into_iter().map(Option::unwrap_or_default));
                }
            }
        }
    }
    if let Some(raw) = dto.config.as_deref() {
        if !is_null(raw) {
            let config: RawImageConfig = serde_json::from_str(raw.get())
                .map_err(|_| type_error(raw, "struct { Labels map[string]string }"))?;
            if let Some(labels) = config.labels.as_deref() {
                if !is_null(labels) {
                    let labels: BTreeMap<String, Option<String>> =
                        serde_json::from_str(labels.get())
                            .map_err(|_| type_error(labels, "map[string]string"))?;
                    doc.labels = labels
                        .into_iter()
                        .map(|(key, value)| (key, value.unwrap_or_default()))
                        .collect();
                }
            }
        }
    }
    Ok(doc)
}
