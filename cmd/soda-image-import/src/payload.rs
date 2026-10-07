use std::collections::HashMap;
use std::fs;
use std::io::Read;
use std::os::unix::fs::MetadataExt;
use std::path::Path;

use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;

use super::json::{parse_json, raw_int, raw_string};
use super::{
    is_coreos_version, is_digest, is_prefixed_digest, is_revision, oci_architecture,
    valid_repository_prefix, NAMES,
};

#[derive(Debug, Clone, Default)]
pub(super) struct ImageBinding {
    pub(super) reference: String,
    pub(super) config: String,
    pub(super) manifest: String,
    pub(super) archive_sha256: String,
}

#[derive(Debug, Clone, Default)]
pub(super) struct Payload {
    pub(super) format: i64,
    pub(super) id: String,
    pub(super) revision: String,
    pub(super) architecture: String,
    pub(super) coreos: String,
    pub(super) base: String,
    pub(super) repository_prefix: String,
    pub(super) schema: i64,
    pub(super) presentation_sha256: String,
    pub(super) host_packages_sha256: String,
    pub(super) images: HashMap<String, ImageBinding>,
    pub(super) upgrade_from: Vec<String>,
}

struct ImageBindingWire {
    reference: Option<Box<RawValue>>,
    config: Option<Box<RawValue>>,
    manifest: Option<Box<RawValue>>,
    archive: Option<Box<RawValue>>,
}
impl<'de> Deserialize<'de> for ImageBindingWire {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ImageBindingWire;
            fn expecting(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
                f.write_str("an image binding object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<Self::Value, M::Error> {
                let (mut reference, mut config, mut manifest, mut archive) =
                    (None, None, None, None);
                while let Some(k) = m.next_key::<String>()? {
                    let v = m.next_value::<Box<RawValue>>()?;
                    let slot = if k.eq_ignore_ascii_case("Reference") {
                        &mut reference
                    } else if k.eq_ignore_ascii_case("Config") {
                        &mut config
                    } else if k.eq_ignore_ascii_case("Manifest") {
                        &mut manifest
                    } else if k.eq_ignore_ascii_case("ArchiveSHA256") {
                        &mut archive
                    } else {
                        return Err(de::Error::custom(format!("unknown field {k:?}")));
                    };
                    if v.get() != "null" {
                        *slot = Some(v);
                    }
                }
                Ok(ImageBindingWire {
                    reference,
                    config,
                    manifest,
                    archive,
                })
            }
        }
        d.deserialize_map(V)
    }
}
struct PayloadWire {
    format: Option<Box<RawValue>>,
    id: Option<Box<RawValue>>,
    revision: Option<Box<RawValue>>,
    architecture: Option<Box<RawValue>>,
    coreos: Option<Box<RawValue>>,
    base: Option<Box<RawValue>>,
    prefix: Option<Box<RawValue>>,
    schema: Option<Box<RawValue>>,
    presentation: Option<Box<RawValue>>,
    host: Option<Box<RawValue>>,
    images: Option<Box<RawValue>>,
    upgrade: Option<Box<RawValue>>,
}
impl<'de> Deserialize<'de> for PayloadWire {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = PayloadWire;
            fn expecting(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
                f.write_str("a payload object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<Self::Value, M::Error> {
                let (
                    mut format,
                    mut id,
                    mut revision,
                    mut architecture,
                    mut coreos,
                    mut base,
                    mut prefix,
                    mut schema,
                    mut presentation,
                    mut host,
                    mut images,
                    mut upgrade,
                ) = (
                    None, None, None, None, None, None, None, None, None, None, None, None,
                );
                while let Some(k) = m.next_key::<String>()? {
                    let v = m.next_value::<Box<RawValue>>()?;
                    let slot = if k.eq_ignore_ascii_case("Format") {
                        &mut format
                    } else if k.eq_ignore_ascii_case("ID") {
                        &mut id
                    } else if k.eq_ignore_ascii_case("Revision") {
                        &mut revision
                    } else if k.eq_ignore_ascii_case("Architecture") {
                        &mut architecture
                    } else if k.eq_ignore_ascii_case("CoreOS") {
                        &mut coreos
                    } else if k.eq_ignore_ascii_case("Base") {
                        &mut base
                    } else if k.eq_ignore_ascii_case("RepositoryPrefix") {
                        &mut prefix
                    } else if k.eq_ignore_ascii_case("Schema") {
                        &mut schema
                    } else if k.eq_ignore_ascii_case("PresentationSHA256") {
                        &mut presentation
                    } else if k.eq_ignore_ascii_case("HostPackagesSHA256") {
                        &mut host
                    } else if k.eq_ignore_ascii_case("Images") {
                        &mut images
                    } else if k.eq_ignore_ascii_case("UpgradeFrom") {
                        &mut upgrade
                    } else {
                        return Err(de::Error::custom(format!("unknown field {k:?}")));
                    };
                    if v.get() != "null" {
                        *slot = Some(v);
                    }
                }
                Ok(PayloadWire {
                    format,
                    id,
                    revision,
                    architecture,
                    coreos,
                    base,
                    prefix,
                    schema,
                    presentation,
                    host,
                    images,
                    upgrade,
                })
            }
        }
        d.deserialize_map(V)
    }
}

fn decode_image_binding(raw: &RawValue) -> Result<ImageBinding, String> {
    let wire: ImageBindingWire = parse_json(raw.get().as_bytes())?;
    Ok(ImageBinding {
        reference: raw_string(wire.reference.as_deref(), "Reference")?,
        config: raw_string(wire.config.as_deref(), "Config")?,
        manifest: raw_string(wire.manifest.as_deref(), "Manifest")?,
        archive_sha256: raw_string(wire.archive.as_deref(), "ArchiveSHA256")?,
    })
}
fn decode_images(raw: Option<&RawValue>) -> Result<HashMap<String, ImageBinding>, String> {
    let Some(raw) = raw else {
        return Ok(HashMap::new());
    };
    struct ImageMap(Vec<(String, Box<RawValue>)>);
    impl<'de> Deserialize<'de> for ImageMap {
        fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
            struct V;
            impl<'de> Visitor<'de> for V {
                type Value = ImageMap;
                fn expecting(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
                    f.write_str("an image map")
                }
                fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<ImageMap, M::Error> {
                    let mut v = Vec::new();
                    while let Some(k) = m.next_key::<String>()? {
                        v.push((k, m.next_value::<Box<RawValue>>()?));
                    }
                    Ok(ImageMap(v))
                }
            }
            d.deserialize_map(V)
        }
    }
    let ImageMap(entries) = parse_json(raw.get().as_bytes())
        .map_err(|_| "field Images must be an object".to_string())?;
    let mut out = HashMap::new();
    for (name, value) in entries {
        let binding = decode_image_binding(&value)?;
        out.insert(name, binding);
    }
    Ok(out)
}

pub(super) fn decode_payload(data: &[u8]) -> Result<Payload, String> {
    let wire: PayloadWire = parse_json(data)?;
    let upgrade_from = match wire.upgrade.as_deref() {
        None => Vec::new(),
        Some(v) => serde_json::from_str::<Vec<Option<String>>>(v.get())
            .map_err(|_| "field UpgradeFrom must be a string list".to_string())?
            .into_iter()
            .map(Option::unwrap_or_default)
            .collect(),
    };
    Ok(Payload {
        format: raw_int(wire.format.as_deref(), "Format")?,
        id: raw_string(wire.id.as_deref(), "ID")?,
        revision: raw_string(wire.revision.as_deref(), "Revision")?,
        architecture: raw_string(wire.architecture.as_deref(), "Architecture")?,
        coreos: raw_string(wire.coreos.as_deref(), "CoreOS")?,
        base: raw_string(wire.base.as_deref(), "Base")?,
        repository_prefix: raw_string(wire.prefix.as_deref(), "RepositoryPrefix")?,
        schema: raw_int(wire.schema.as_deref(), "Schema")?,
        presentation_sha256: raw_string(wire.presentation.as_deref(), "PresentationSHA256")?,
        host_packages_sha256: raw_string(wire.host.as_deref(), "HostPackagesSHA256")?,
        images: decode_images(wire.images.as_deref())?,
        upgrade_from,
    })
}

impl Payload {
    fn valid_identity(&self) -> bool {
        self.format == 3
            && is_revision(&self.revision)
            && self.id == format!("{}.soda-{}", self.coreos, &self.revision[..12])
            && is_coreos_version(&self.coreos)
    }
    fn valid_base(&self) -> bool {
        let (host, digest) = match self.base.split_once("/fedora/fedora-coreos@sha256:") {
            Some(v) => v,
            None => return false,
        };
        !host.is_empty()
            && !host.contains('/')
            && is_digest(digest)
            && self.schema >= 1
            && is_digest(&self.presentation_sha256)
            && is_digest(&self.host_packages_sha256)
    }
    fn valid_images(&self) -> Result<(), String> {
        if self.images.len() != NAMES.len() {
            return Err("complete image set required".to_string());
        }
        for name in NAMES {
            match self.images.get(name) {
                Some(image)
                    if is_prefixed_digest(&image.config)
                        && is_prefixed_digest(&image.manifest)
                        && is_digest(&image.archive_sha256)
                        && image.reference
                            == format!("{}-{name}@{}", self.repository_prefix, image.manifest) => {}
                _ => return Err(format!("invalid {name} image binding")),
            }
        }
        match (self.images.get("extension"), self.images.get("forgejo")) {
            (Some(a), Some(b)) if a.config != b.config => Ok(()),
            (Some(_), Some(_)) => {
                Err("extension requires an independent image identity".to_string())
            }
            _ => Err("complete image set required".to_string()),
        }
    }
    pub(super) fn validate(&self) -> Result<(), String> {
        if !self.valid_identity() {
            return Err("invalid appliance payload identity".to_string());
        }
        oci_architecture(&self.architecture)?;
        if !valid_repository_prefix(&self.repository_prefix) || !self.valid_base() {
            return Err("incomplete appliance payload".to_string());
        }
        self.valid_images()?;
        if !self.upgrade_from.is_empty() {
            return Err("candidate has no qualified upgrade paths".to_string());
        }
        Ok(())
    }
    pub(super) fn load(path: &Path) -> Result<Self, String> {
        let data = read_bounded_json(path, 4 << 20)?;
        let payload = decode_payload(&data)?;
        payload.validate()?;
        Ok(payload)
    }
}
fn read_bounded_json(path: &Path, max: u64) -> Result<Vec<u8>, String> {
    let before = fs::symlink_metadata(path).map_err(|e| format!("cannot stat JSON input: {e}"))?;
    if !before.file_type().is_file() || before.len() > max {
        return Err("bounded regular JSON input required".to_string());
    }
    let file = fs::File::open(path).map_err(|e| format!("cannot open JSON input: {e}"))?;
    let after = file
        .metadata()
        .map_err(|e| format!("cannot stat JSON input: {e}"))?;
    if !after.is_file() || after.dev() != before.dev() || after.ino() != before.ino() {
        return Err("JSON input changed before reading".to_string());
    }
    let mut data = Vec::new();
    file.take(max + 1)
        .read_to_end(&mut data)
        .map_err(|e| format!("cannot read JSON input: {e}"))?;
    if data.len() as u64 > max {
        return Err("JSON input exceeds limit".to_string());
    }
    Ok(data)
}
