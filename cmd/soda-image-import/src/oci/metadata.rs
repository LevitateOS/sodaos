use crate::json::{parse_json, raw_int, raw_string};
use serde::de::{MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;
use std::collections::HashMap;

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
    pub(crate) data: Option<Vec<u8>>,
}

macro_rules! raw_record {
    ($name:ident { $($field:ident => [$($alias:literal),+]),+ $(,)? } unknown $unknown:ident) => {
        struct $name { $($field: Option<Box<RawValue>>),+ }
        impl<'de> Deserialize<'de> for $name {
            fn deserialize<D:Deserializer<'de>>(d:D)->Result<Self,D::Error>{
                struct V; impl<'de> Visitor<'de> for V { type Value=$name;
                    fn expecting(&self,f:&mut std::fmt::Formatter)->std::fmt::Result{f.write_str("an object")}
                    fn visit_map<M:MapAccess<'de>>(self,mut m:M)->Result<$name,M::Error>{
                        $(let mut $field: Option<Box<RawValue>> = None;)+
                        let _ = stringify!($unknown);
                        while let Some(k)=m.next_key::<String>()? { match k.to_ascii_lowercase().as_str() {
                            $( $( $alias => { let v=m.next_value::<Box<RawValue>>()?; if v.get()!="null" { $field=Some(v); } }, )+ )+
                            _ => { let _:serde::de::IgnoredAny=m.next_value()?; }
                        }} Ok($name{$($field),+})
                    }
                } d.deserialize_map(V)
            }
        }
    }
}
raw_record!(DescriptorWire { digest=>["digest"], size=>["size"], media_type=>["mediatype"], urls=>["urls"], annotations=>["annotations"] } unknown ignored);
raw_record!(ManifestWire { schema=>["schemaversion"], media=>["mediatype"], config=>["config"], layers=>["layers"] } unknown ignored);
raw_record!(IndexWire { schema=>["schemaversion"], media=>["mediatype"], manifests=>["manifests"] } unknown ignored);
raw_record!(LayoutWire { version=>["imagelayoutversion"] } unknown ignored);
raw_record!(ConfigWire { os=>["os"], architecture=>["architecture"], rootfs=>["rootfs"], config=>["config"] } unknown ignored);
raw_record!(RootfsWire { kind=>["type"], diff_ids=>["diff_ids"] } unknown ignored);
raw_record!(LabelsWire { labels=>["labels"] } unknown ignored);

fn string_list(raw: Option<&RawValue>, name: &str) -> Result<Vec<String>, String> {
    match raw {
        None => Ok(Vec::new()),
        Some(v) => serde_json::from_str::<Vec<Option<String>>>(v.get())
            .map(|a| a.into_iter().map(Option::unwrap_or_default).collect())
            .map_err(|_| format!("field {name} must be a string list")),
    }
}
fn string_map(raw: Option<&RawValue>, name: &str) -> Result<HashMap<String, String>, String> {
    match raw {
        None => Ok(HashMap::new()),
        Some(v) => serde_json::from_str::<HashMap<String, Option<String>>>(v.get())
            .map(|m| {
                m.into_iter()
                    .map(|(k, v)| (k, v.unwrap_or_default()))
                    .collect()
            })
            .map_err(|_| format!("field {name} must be a string map")),
    }
}
fn descriptor(raw: &RawValue) -> Result<OciDescriptor, String> {
    let w: DescriptorWire = parse_json(raw.get().as_bytes())
        .map_err(|_| "OCI descriptor must be an object".to_string())?;
    Ok(OciDescriptor {
        digest: raw_string(w.digest.as_deref(), "digest")?,
        size: raw_int(w.size.as_deref(), "size")?,
        media_type: raw_string(w.media_type.as_deref(), "mediaType")?,
        urls: string_list(w.urls.as_deref(), "urls")?,
        annotations: string_map(w.annotations.as_deref(), "annotations")?,
    })
}
fn descriptors(raw: Option<&RawValue>) -> Result<Vec<OciDescriptor>, String> {
    match raw {
        None => Ok(Vec::new()),
        Some(v) => {
            let a: Vec<Option<Box<RawValue>>> = serde_json::from_str(v.get())
                .map_err(|_| "field manifests must be a list".to_string())?;
            a.into_iter()
                .map(|x| match x {
                    None => Ok(OciDescriptor::default()),
                    Some(x) => descriptor(&x),
                })
                .collect()
        }
    }
}
pub(crate) struct OciManifestData {
    pub(crate) config: OciDescriptor,
    pub(crate) layers: Vec<OciDescriptor>,
}
pub(crate) fn parse_oci_manifest(data: &[u8]) -> Result<OciManifestData, String> {
    let w: ManifestWire = parse_json(data)?;
    let schema = raw_int(w.schema.as_deref(), "schemaVersion")?;
    let media = raw_string(w.media.as_deref(), "mediaType")?;
    let config = match w.config.as_deref() {
        None => OciDescriptor::default(),
        Some(v) if v.get() == "null" => OciDescriptor::default(),
        Some(v) => descriptor(v)?,
    };
    let layers = descriptors(w.layers.as_deref())?;
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
    let w: ConfigWire = parse_json(data)?;
    let os = raw_string(w.os.as_deref(), "os")?;
    let arch = raw_string(w.architecture.as_deref(), "architecture")?;
    let (rootfs_type, diff_ids) = match w.rootfs.as_deref() {
        None => (String::new(), Vec::new()),
        Some(v) if v.get() == "null" => (String::new(), Vec::new()),
        Some(v) => {
            let x: RootfsWire = parse_json(v.get().as_bytes())
                .map_err(|_| "field rootfs must be an object".to_string())?;
            (
                raw_string(x.kind.as_deref(), "type")?,
                string_list(x.diff_ids.as_deref(), "diff_ids")?,
            )
        }
    };
    let labels = match w.config.as_deref() {
        None => HashMap::new(),
        Some(v) if v.get() == "null" => HashMap::new(),
        Some(v) => {
            let x: LabelsWire = parse_json(v.get().as_bytes())
                .map_err(|_| "field config must be an object".to_string())?;
            string_map(x.labels.as_deref(), "Labels")?
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
    let layout = entries
        .get("oci-layout")
        .and_then(|b| b.data.as_ref())
        .ok_or_else(|| "missing OCI layout".to_string())?;
    let l: LayoutWire = parse_json(layout).map_err(|_| "missing OCI layout".to_string())?;
    let version = raw_string(l.version.as_deref(), "imageLayoutVersion")
        .map_err(|_| "missing OCI layout".to_string())?;
    if version != "1.0.0" {
        return Err("missing OCI layout".to_string());
    }
    let data = entries
        .get("index.json")
        .and_then(|b| b.data.as_ref())
        .ok_or_else(|| "valid OCI index required".to_string())?;
    let w: IndexWire = parse_json(data).map_err(|_| "valid OCI index required".to_string())?;
    let schema = raw_int(w.schema.as_deref(), "schemaVersion")
        .map_err(|_| "valid OCI index required".to_string())?;
    let media = raw_string(w.media.as_deref(), "mediaType")
        .map_err(|_| "valid OCI index required".to_string())?;
    let manifests =
        descriptors(w.manifests.as_deref()).map_err(|_| "valid OCI index required".to_string())?;
    if schema != 2 || (!media.is_empty() && media != "application/vnd.oci.image.index.v1+json") {
        return Err("valid OCI index required".to_string());
    }
    Ok(manifests)
}
