//! `payload.go`: immutable appliance payload metadata.

use serde::de::{DeserializeOwned, Error as DeError, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};
use std::collections::BTreeMap;
use std::fmt;

use crate::buildx::{is_digest, is_revision, oci_architecture, Root};
use crate::Error;

pub const PATH: &str = "/usr/share/soda/release.json";
pub const IMAGES_PATH: &str = "/usr/share/soda/images";

pub const NAMES: [&str; 6] = [
    "dashboard",
    "forgejo",
    "extension",
    "proxy",
    "project-os",
    "tailnet",
];

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct Image {
    pub reference: String,
    pub config: String,
    pub manifest: String,
    #[serde(rename = "ArchiveSHA256")]
    pub archive_sha256: String,
}

fn decode_raw_default<T: DeserializeOwned + Default, E: DeError>(
    raw: Option<Box<serde_json::value::RawValue>>,
) -> Result<T, E> {
    let Some(raw) = raw else {
        return Ok(T::default());
    };
    serde_json::from_str::<Option<T>>(raw.get())
        .map_err(E::custom)
        .map(Option::unwrap_or_default)
}

fn decode_images<E: DeError>(
    raw: Option<Box<serde_json::value::RawValue>>,
) -> Result<BTreeMap<String, Image>, E> {
    let Some(raw) = raw else {
        return Ok(BTreeMap::new());
    };
    let Some(entries) = serde_json::from_str::<
        Option<BTreeMap<String, Box<serde_json::value::RawValue>>>,
    >(raw.get())
    .map_err(E::custom)?
    else {
        return Ok(BTreeMap::new());
    };
    entries
        .into_iter()
        .map(|(name, value)| {
            serde_json::from_str::<Image>(value.get())
                .map(|image| (name, image))
                .map_err(E::custom)
        })
        .collect()
}

#[derive(Default)]
struct RawI64(i64);
impl<'de> Deserialize<'de> for RawI64 {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        crate::json_serde::null_i64(d).map(RawI64)
    }
}

impl<'de> Deserialize<'de> for Image {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = Image;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an image object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<Image, M::Error> {
                let (mut reference, mut config, mut manifest, mut archive_sha256) =
                    (None, None, None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "Reference" => {
                            reference = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "Config" => {
                            config = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "Manifest" => {
                            manifest = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "ArchiveSHA256" => {
                            archive_sha256 =
                                Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            return Err(M::Error::unknown_field(
                                &key,
                                &["Reference", "Config", "Manifest", "ArchiveSHA256"],
                            ))
                        }
                    }
                }
                Ok(Image {
                    reference: decode_raw_default(reference)?,
                    config: decode_raw_default(config)?,
                    manifest: decode_raw_default(manifest)?,
                    archive_sha256: decode_raw_default(archive_sha256)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl Image {}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct Payload {
    pub format: i64,
    #[serde(rename = "ID")]
    pub id: String,
    pub revision: String,
    pub architecture: String,
    #[serde(rename = "CoreOS")]
    pub core_os: String,
    pub base: String,
    pub repository_prefix: String,
    pub schema: i64,
    #[serde(rename = "PresentationSHA256")]
    pub presentation_sha256: String,
    #[serde(rename = "HostPackagesSHA256")]
    pub host_packages_sha256: String,
    pub images: BTreeMap<String, Image>,
    // `nil` vs empty is observable in JSON (`null` vs `[]`); Go leaves this
    // unset, so `None` marshals as `null` exactly like the owner.
    pub upgrade_from: Option<Vec<String>>,
}

impl<'de> Deserialize<'de> for Payload {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = Payload;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an appliance payload object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<Payload, M::Error> {
                let (
                    mut format,
                    mut id,
                    mut revision,
                    mut architecture,
                    mut core_os,
                    mut base,
                    mut repository_prefix,
                    mut schema,
                    mut presentation_sha256,
                    mut host_packages_sha256,
                    mut images,
                    mut upgrade_from,
                ) = (
                    None, None, None, None, None, None, None, None, None, None, None, None,
                );
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "Format" => {
                            format = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "ID" => id = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "Revision" => {
                            revision = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "Architecture" => {
                            architecture = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "CoreOS" => {
                            core_os = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "Base" => base = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "RepositoryPrefix" => {
                            repository_prefix =
                                Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "Schema" => {
                            schema = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "PresentationSHA256" => {
                            presentation_sha256 =
                                Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "HostPackagesSHA256" => {
                            host_packages_sha256 =
                                Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "Images" => {
                            images = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "UpgradeFrom" => {
                            upgrade_from = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            return Err(M::Error::unknown_field(
                                &key,
                                &[
                                    "Format",
                                    "ID",
                                    "Revision",
                                    "Architecture",
                                    "CoreOS",
                                    "Base",
                                    "RepositoryPrefix",
                                    "Schema",
                                    "PresentationSHA256",
                                    "HostPackagesSHA256",
                                    "Images",
                                    "UpgradeFrom",
                                ],
                            ))
                        }
                    }
                }
                let format = decode_raw_default::<RawI64, M::Error>(format)?.0;
                let schema = decode_raw_default::<RawI64, M::Error>(schema)?.0;
                Ok(Payload {
                    format,
                    id: decode_raw_default(id)?,
                    revision: decode_raw_default(revision)?,
                    architecture: decode_raw_default(architecture)?,
                    core_os: decode_raw_default(core_os)?,
                    base: decode_raw_default(base)?,
                    repository_prefix: decode_raw_default(repository_prefix)?,
                    schema,
                    presentation_sha256: decode_raw_default(presentation_sha256)?,
                    host_packages_sha256: decode_raw_default(host_packages_sha256)?,
                    images: decode_images(images)?,
                    upgrade_from: decode_raw_default(upgrade_from)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

fn is_core_os_version(s: &str) -> bool {
    let parts: Vec<&str> = s.split('.').collect();
    if parts.len() != 4 {
        return false;
    }
    parts
        .iter()
        .all(|p| !p.is_empty() && p.bytes().all(|b| b.is_ascii_digit()))
}

fn is_digest_ref(s: &str) -> bool {
    s.starts_with("sha256:") && is_digest(s.strip_prefix("sha256:").unwrap_or(""))
}

impl Payload {
    fn valid_identity(&self) -> bool {
        if self.format != 3 || !is_revision(&self.revision) {
            return false;
        }
        if self.id != format!("{}.soda-{}", self.core_os, &self.revision[..12]) {
            return false;
        }
        is_core_os_version(&self.core_os)
    }

    fn valid_base(&self) -> bool {
        let (_, rest) = match self.base.split_once("/fedora/fedora-coreos@sha256:") {
            Some((host, digest)) => (host, digest),
            None => return false,
        };
        let host = self
            .base
            .split_once("/fedora/fedora-coreos@sha256:")
            .map(|(h, _)| h)
            .unwrap_or("");
        if host.is_empty() || host.contains('/') {
            return false;
        }
        is_digest(rest)
            && self.schema >= 1
            && is_digest(&self.presentation_sha256)
            && is_digest(&self.host_packages_sha256)
    }

    fn valid_images(&self) -> Result<(), Error> {
        if self.images.len() != NAMES.len() {
            return Err(Error::msg("complete image set required"));
        }
        for name in NAMES {
            let image = match self.images.get(name) {
                Some(image) => image,
                None => return Err(Error::msg(format!("invalid {name} image binding"))),
            };
            let reference = format!("{}-{name}@{}", self.repository_prefix, image.manifest);
            if !is_digest_ref(&image.config)
                || !is_digest_ref(&image.manifest)
                || !is_digest(&image.archive_sha256)
                || image.reference != reference
            {
                return Err(Error::msg(format!("invalid {name} image binding")));
            }
        }
        if self.images["extension"].config == self.images["forgejo"].config {
            return Err(Error::msg(
                "extension requires an independent image identity",
            ));
        }
        Ok(())
    }

    pub fn validate(&self) -> Result<(), Error> {
        if !self.valid_identity() {
            return Err(Error::msg("invalid appliance payload identity"));
        }
        if let Err(e) = oci_architecture(&self.architecture) {
            return Err(Error::msg(e.0));
        }
        if !valid_repository_prefix(&self.repository_prefix) {
            return Err(Error::msg("explicit GHCR repository prefix required"));
        }
        if !self.valid_base() {
            return Err(Error::msg("incomplete appliance payload"));
        }
        self.valid_images()?;
        if matches!(&self.upgrade_from, Some(entries) if !entries.is_empty()) {
            return Err(Error::msg("candidate has no qualified upgrade paths"));
        }
        Ok(())
    }
}

/// `ValidRepositoryPrefix`: `^ghcr\.io/[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9._-]*$`
/// with length below 200.
pub fn valid_repository_prefix(s: &str) -> bool {
    if s.len() >= 200 {
        return false;
    }
    let rest = match s.strip_prefix("ghcr.io/") {
        Some(rest) => rest,
        None => return false,
    };
    let (owner, repo) = match rest.split_once('/') {
        Some(pair) => pair,
        None => return false,
    };
    if repo.contains('/') {
        return false;
    }
    let mut owner_chars = owner.bytes();
    match owner_chars.next() {
        Some(b @ (b'a'..=b'z' | b'0'..=b'9')) => {
            let _ = b;
        }
        _ => return false,
    }
    if !owner_chars.all(|b| matches!(b, b'a'..=b'z' | b'0'..=b'9' | b'-')) {
        return false;
    }
    let mut repo_chars = repo.bytes();
    match repo_chars.next() {
        Some(b @ (b'a'..=b'z' | b'0'..=b'9')) => {
            let _ = b;
        }
        _ => return false,
    }
    repo_chars.all(|b| matches!(b, b'a'..=b'z' | b'0'..=b'9' | b'.' | b'_' | b'-'))
}

/// `Load`: read and validate a payload file.
pub fn load(path: &str) -> Result<Payload, Error> {
    let parent = match path.rfind('/') {
        Some(0) | None => "/".to_string(),
        Some(i) => path[..i].to_string(),
    };
    let base = match path.rfind('/') {
        Some(i) => &path[i + 1..],
        None => path,
    };
    let root = Root::open(&parent)?;
    let (payload, _) = crate::buildx::read_json_at(&root, base)?;
    payload.validate()?;
    Ok(payload)
}

#[cfg(test)]
mod tests;
