//! Appliance payload metadata ported from `internal/release/deliver`:
//! immutable release binding plus the shared-OCI content verification the
//! installer runs before any disk write. Only the installer-reachable
//! surface (`Payload`, `Load`, `VerifyContent`) is ported.

use std::collections::BTreeMap;

use crate::buildx;
use crate::errors::Error;
use crate::oci;
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;
use std::fmt;

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

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Image {
    pub reference: String,
    pub config: String,
    pub manifest: String,
    pub archive_sha256: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Payload {
    pub format: i64,
    pub id: String,
    pub revision: String,
    pub architecture: String,
    pub core_os: String,
    pub base: String,
    pub repository_prefix: String,
    pub schema: i64,
    pub presentation_sha256: String,
    pub host_packages_sha256: String,
    pub images: BTreeMap<String, Image>,
    pub upgrade_from: Vec<String>,
}

fn core_os_shape(core_os: &str) -> bool {
    // `^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`
    let parts: Vec<&str> = core_os.split('.').collect();
    parts.len() == 4
        && parts
            .iter()
            .all(|p| !p.is_empty() && p.bytes().all(|b| b.is_ascii_digit()))
}

fn valid_repository_prefix(s: &str) -> bool {
    // `^ghcr\.io/[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9._-]*$`, length under 200.
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
    if owner.is_empty() || repo.is_empty() || rest.contains("//") {
        // `split_once` already rejects missing separators; a second slash
        // anywhere (including a trailing one) fails the anchored pattern.
        if rest.matches('/').count() != 1 {
            return false;
        }
    }
    let owner_ok = owner
        .bytes()
        .next()
        .map(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        .unwrap_or(false)
        && owner
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-');
    let repo_ok = repo
        .bytes()
        .next()
        .map(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        .unwrap_or(false)
        && repo.bytes().all(|b| {
            b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'.' || b == b'_' || b == b'-'
        });
    owner_ok && repo_ok
}

fn digest_prefixed(s: &str) -> bool {
    match s.strip_prefix("sha256:") {
        Some(hex) => buildx::digest(hex),
        None => false,
    }
}

impl Payload {
    fn valid_identity(&self) -> bool {
        self.format == 3
            && buildx::revision(&self.revision)
            && self.id == format!("{}.soda-{}", self.core_os, &self.revision[..12])
            && core_os_shape(&self.core_os)
    }

    fn valid_base(&self) -> bool {
        match self.base.split_once("/fedora/fedora-coreos@sha256:") {
            Some((host, digest)) => {
                !host.is_empty()
                    && !host.contains('/')
                    && buildx::digest(digest)
                    && self.schema >= 1
                    && buildx::digest(&self.presentation_sha256)
                    && buildx::digest(&self.host_packages_sha256)
            }
            None => false,
        }
    }

    fn valid_images(&self) -> Result<(), Error> {
        if self.images.len() != NAMES.len() {
            return Err(Error::msg("complete image set required"));
        }
        for name in NAMES {
            let valid = match self.images.get(name) {
                Some(image) => {
                    digest_prefixed(&image.config)
                        && digest_prefixed(&image.manifest)
                        && buildx::digest(&image.archive_sha256)
                        && image.reference
                            == format!("{}-{name}@{}", self.repository_prefix, image.manifest)
                }
                None => false,
            };
            if !valid {
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
        buildx::oci_architecture(&self.architecture)?;
        if !valid_repository_prefix(&self.repository_prefix) {
            return Err(Error::msg("explicit GHCR repository prefix required"));
        }
        if !self.valid_base() {
            return Err(Error::msg("incomplete appliance payload"));
        }
        self.valid_images()?;
        // Native admission is not implemented in this milestone. Do not
        // publish a guessed compatibility promise simply because schemas
        // happen to match.
        if !self.upgrade_from.is_empty() {
            return Err(Error::msg("candidate has no qualified upgrade paths"));
        }
        Ok(())
    }
}

struct ImageFields([Option<Box<RawValue>>; 4]);
impl<'de> Deserialize<'de> for ImageFields {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ImageFields;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an image object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<ImageFields, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut fields: [Option<Box<RawValue>>; 4] = [None, None, None, None];
                while let Some(key) = map.next_key::<String>()? {
                    let slot = image_slot(&key)
                        .ok_or_else(|| de::Error::unknown_field(&key, IMAGE_FIELDS))?;
                    fields[slot] = Some(map.next_value()?);
                }
                Ok(ImageFields(fields))
            }
        }
        d.deserialize_map(V)
    }
}

const IMAGE_FIELDS: &[&str] = &["Reference", "Config", "Manifest", "ArchiveSHA256"];
fn image_slot(key: &str) -> Option<usize> {
    field_slot(key, IMAGE_FIELDS)
}
fn field_slot(key: &str, fields: &[&str]) -> Option<usize> {
    if let Some(i) = fields.iter().position(|field| *field == key) {
        return Some(i);
    }
    fields
        .iter()
        .position(|field| key.len() == field.len() && key.eq_ignore_ascii_case(field))
}
fn raw_string(raw: &Option<Box<RawValue>>) -> Result<String, ()> {
    match raw {
        None => Ok(String::new()),
        Some(raw) => serde_json::from_str::<Option<String>>(raw.get())
            .map(|v| v.unwrap_or_default())
            .map_err(|_| ()),
    }
}
fn decode_image(raw: &RawValue) -> Result<Image, ()> {
    if raw.get() == "null" {
        return Ok(Image::default());
    }
    let fields: ImageFields = serde_json::from_str(raw.get()).map_err(|_| ())?;
    Ok(Image {
        reference: raw_string(&fields.0[0])?,
        config: raw_string(&fields.0[1])?,
        manifest: raw_string(&fields.0[2])?,
        archive_sha256: raw_string(&fields.0[3])?,
    })
}

struct PayloadFields([Option<Box<RawValue>>; 12]);
impl<'de> Deserialize<'de> for PayloadFields {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = PayloadFields;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a payload object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<PayloadFields, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut fields: [Option<Box<RawValue>>; 12] = std::array::from_fn(|_| None);
                while let Some(key) = map.next_key::<String>()? {
                    let slot = payload_slot(&key)
                        .ok_or_else(|| de::Error::unknown_field(&key, PAYLOAD_FIELDS))?;
                    fields[slot] = Some(map.next_value()?);
                }
                Ok(PayloadFields(fields))
            }
        }
        d.deserialize_map(V)
    }
}
const PAYLOAD_FIELDS: &[&str] = &[
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
];
fn payload_slot(key: &str) -> Option<usize> {
    field_slot(key, PAYLOAD_FIELDS)
}
fn raw_i64(raw: &Option<Box<RawValue>>) -> Result<i64, ()> {
    match raw {
        None => Ok(0),
        Some(raw) if raw.get() == "null" => Ok(0),
        Some(raw) => raw.get().parse::<i64>().map_err(|_| ()),
    }
}
fn raw_string_list(raw: &Option<Box<RawValue>>) -> Result<Vec<String>, ()> {
    match raw {
        None => Ok(Vec::new()),
        Some(raw) => serde_json::from_str::<Option<Vec<Option<String>>>>(raw.get())
            .map(|v| {
                v.unwrap_or_default()
                    .into_iter()
                    .map(|s| s.unwrap_or_default())
                    .collect()
            })
            .map_err(|_| ()),
    }
}

struct PayloadImages(BTreeMap<String, Image>);

impl<'de> Deserialize<'de> for PayloadImages {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct ImagesVisitor;
        impl<'de> Visitor<'de> for ImagesVisitor {
            type Value = PayloadImages;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("payload image bindings")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let mut images = BTreeMap::new();
                while let Some(name) = map.next_key::<String>()? {
                    let raw = map.next_value::<Box<RawValue>>()?;
                    let image = decode_image(&raw)
                        .map_err(|_| de::Error::custom("invalid payload image binding"))?;
                    images.insert(name, image);
                }
                Ok(PayloadImages(images))
            }
        }
        deserializer.deserialize_map(ImagesVisitor)
    }
}

fn decode_payload(raw: &RawValue) -> Result<Payload, ()> {
    let fields: PayloadFields = serde_json::from_str(raw.get()).map_err(|_| ())?;
    let strings = [
        &fields.0[1],
        &fields.0[2],
        &fields.0[3],
        &fields.0[4],
        &fields.0[5],
        &fields.0[6],
        &fields.0[8],
        &fields.0[9],
    ];
    let mut values = Vec::with_capacity(strings.len());
    for value in strings {
        values.push(raw_string(value)?);
    }
    let mut images = BTreeMap::new();
    if let Some(raw_images) = &fields.0[10] {
        if raw_images.get() != "null" {
            images = serde_json::from_str::<PayloadImages>(raw_images.get())
                .map_err(|_| ())?
                .0;
        }
    }
    Ok(Payload {
        format: raw_i64(&fields.0[0])?,
        id: values[0].clone(),
        revision: values[1].clone(),
        architecture: values[2].clone(),
        core_os: values[3].clone(),
        base: values[4].clone(),
        repository_prefix: values[5].clone(),
        schema: raw_i64(&fields.0[7])?,
        presentation_sha256: values[6].clone(),
        host_packages_sha256: values[7].clone(),
        images,
        upgrade_from: raw_string_list(&fields.0[11])?,
    })
}

pub fn load(path: &str) -> Result<Payload, Error> {
    let data = buildx::read_json_bytes(path)?;
    let text = String::from_utf8_lossy(&data);
    let mut deserializer = serde_json::Deserializer::from_str(&text);
    let raw = Box::<RawValue>::deserialize(&mut deserializer)
        .map_err(|_| Error::msg("invalid payload document"))?;
    deserializer
        .end()
        .map_err(|_| Error::msg("invalid payload document"))?;
    let payload = decode_payload(&raw).map_err(|_| Error::msg("invalid payload document"))?;
    payload.validate()?;
    Ok(payload)
}

fn bind_image_revisions(payload: &Payload) -> Result<BTreeMap<String, String>, Error> {
    let mut revisions = BTreeMap::new();
    for name in NAMES {
        let revision = if name == "proxy" {
            String::new()
        } else {
            payload.revision.clone()
        };
        let reference = payload.images[name].config.clone();
        if let Some(previous) = revisions.get(&reference) {
            if *previous != revision {
                return Err(Error::msg("conflicting OCI source bindings"));
            }
        }
        revisions.insert(reference, revision);
    }
    Ok(revisions)
}

fn match_layout_identities(payload: &Payload, layout: &oci::OciLayout) -> Result<(), Error> {
    for name in NAMES {
        let expected = &payload.images[name];
        match layout.images.get(&expected.config) {
            Some(got) if got.config == expected.config && got.manifest == expected.manifest => {}
            _ => return Err(Error::msg(format!("{name} OCI identity mismatch"))),
        }
    }
    Ok(())
}

/// Binds the shared OCI content to the payload before import or disk
/// writes. Each shared blob is hashed and counted once.
pub fn verify_content(
    payload: &Payload,
    images: &str,
) -> Result<(BTreeMap<String, String>, u64), Error> {
    payload.validate()?;
    if !images.starts_with('/') || images.contains([':', '\r', '\n']) {
        return Err(Error::msg("absolute local OCI directory required"));
    }
    let revisions = bind_image_revisions(payload)?;
    let layout = oci::inspect_oci_layout(images, &payload.architecture, &revisions)?;
    match_layout_identities(payload, &layout)?;
    Ok((layout.files, layout.bytes))
}

#[cfg(test)]
mod tests;
