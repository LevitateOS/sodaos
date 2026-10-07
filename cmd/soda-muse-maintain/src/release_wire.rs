use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::fmt;

use super::release::{ReleaseImage, ReleasePayload};

pub(crate) fn decode_release_payload(body: &[u8]) -> Result<ReleasePayload, ()> {
    let mut deserializer = serde_json::Deserializer::from_slice(body);
    let payload = PayloadDto::deserialize(&mut deserializer).map_err(|_| ())?;
    deserializer.end().map_err(|_| ())?;
    super::release_validation::validate_release_payload(
        payload.format,
        &payload.id,
        &payload.revision,
        &payload.architecture,
        &payload.coreos,
        &payload.base,
        &payload.repository_prefix,
        payload.schema,
        &payload.presentation,
        &payload.host_packages,
        &payload.images,
        &payload.upgrade_from,
    )?;
    Ok(ReleasePayload {
        architecture: payload.architecture,
        images: payload.images,
    })
}

#[derive(Default)]
struct PayloadDto {
    format: i64,
    id: String,
    revision: String,
    architecture: String,
    coreos: String,
    base: String,
    repository_prefix: String,
    schema: i64,
    presentation: String,
    host_packages: String,
    images: Vec<(String, ReleaseImage)>,
    upgrade_from: Vec<String>,
}

impl<'de> Deserialize<'de> for PayloadDto {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct PayloadVisitor;
        impl<'de> Visitor<'de> for PayloadVisitor {
            type Value = PayloadDto;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a release payload object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<PayloadDto, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut dto = PayloadDto {
                    format: -1,
                    schema: -1,
                    ..Default::default()
                };
                while let Some(key) = map.next_key::<String>()? {
                    match payload_slot(&key) {
                        Some(0) => dto.format = map.next_value::<Nullable<i64>>()?.0,
                        Some(1) => dto.id = map.next_value::<Nullable<String>>()?.0,
                        Some(2) => dto.revision = map.next_value::<Nullable<String>>()?.0,
                        Some(3) => dto.architecture = map.next_value::<Nullable<String>>()?.0,
                        Some(4) => dto.coreos = map.next_value::<Nullable<String>>()?.0,
                        Some(5) => dto.base = map.next_value::<Nullable<String>>()?.0,
                        Some(6) => dto.repository_prefix = map.next_value::<Nullable<String>>()?.0,
                        Some(7) => dto.schema = map.next_value::<Nullable<i64>>()?.0,
                        Some(8) => dto.presentation = map.next_value::<Nullable<String>>()?.0,
                        Some(9) => dto.host_packages = map.next_value::<Nullable<String>>()?.0,
                        Some(10) => dto.images = map.next_value::<Images>()?.0,
                        Some(11) => dto.upgrade_from = map.next_value::<Nullable<Vec<String>>>()?.0,
                        _ => return Err(de::Error::unknown_field(&key, PAYLOAD_FIELDS)),
                    }
                }
                Ok(dto)
            }
        }
        deserializer.deserialize_map(PayloadVisitor)
    }
}

struct Nullable<T>(T);
impl<'de, T> Deserialize<'de> for Nullable<T>
where
    T: Deserialize<'de> + Default,
{
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct NullableVisitor<T>(std::marker::PhantomData<T>);
        impl<'de, T> Visitor<'de> for NullableVisitor<T>
        where
            T: Deserialize<'de> + Default,
        {
            type Value = Nullable<T>;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a value or null")
            }
            fn visit_none<E>(self) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(Nullable(T::default()))
            }
            fn visit_unit<E>(self) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(Nullable(T::default()))
            }
            fn visit_some<D2>(self, d: D2) -> Result<Self::Value, D2::Error>
            where
                D2: Deserializer<'de>,
            {
                T::deserialize(d).map(Nullable)
            }
        }
        deserializer.deserialize_option(NullableVisitor(std::marker::PhantomData))
    }
}

struct Images(Vec<(String, ReleaseImage)>);
impl<'de> Deserialize<'de> for Images {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct ImagesVisitor;
        impl<'de> Visitor<'de> for ImagesVisitor {
            type Value = Images;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an image object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Images, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut images: Vec<(String, ReleaseImage)> = Vec::new();
                while let Some(name) = map.next_key::<String>()? {
                    let image = map.next_value::<ImageDto>()?.0;
                    if let Some(entry) = images.iter_mut().find(|(existing, _)| existing == &name) {
                        entry.1 = image;
                    } else {
                        images.push((name, image));
                    }
                }
                Ok(Images(images))
            }
        }
        deserializer.deserialize_map(ImagesVisitor)
    }
}

struct ImageDto(ReleaseImage);
impl<'de> Deserialize<'de> for ImageDto {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct ImageVisitor;
        impl<'de> Visitor<'de> for ImageVisitor {
            type Value = ImageDto;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a release image object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<ImageDto, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut image = ReleaseImage {
                    reference: String::new(),
                    config: String::new(),
                    manifest: String::new(),
                    archive_sha256: String::new(),
                };
                while let Some(key) = map.next_key::<String>()? {
                    match image_slot(&key) {
                        Some(0) => image.reference = map.next_value::<Nullable<String>>()?.0,
                        Some(1) => image.config = map.next_value::<Nullable<String>>()?.0,
                        Some(2) => image.manifest = map.next_value::<Nullable<String>>()?.0,
                        Some(3) => image.archive_sha256 = map.next_value::<Nullable<String>>()?.0,
                        _ => return Err(de::Error::unknown_field(&key, IMAGE_FIELDS)),
                    }
                }
                Ok(ImageDto(image))
            }
        }
        deserializer.deserialize_map(ImageVisitor)
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
const IMAGE_FIELDS: &[&str] = &["Reference", "Config", "Manifest", "ArchiveSHA256"];

fn slot(key: &str, fields: &[&str]) -> Option<usize> {
    if let Some(i) = fields.iter().position(|f| *f == key) {
        return Some(i);
    }
    let mut found = None;
    for (i, field) in fields.iter().enumerate() {
        if field.eq_ignore_ascii_case(key) {
            if found.is_some() {
                return None;
            }
            found = Some(i);
        }
    }
    found
}
fn payload_slot(key: &str) -> Option<usize> {
    slot(key, PAYLOAD_FIELDS)
}
fn image_slot(key: &str) -> Option<usize> {
    slot(key, IMAGE_FIELDS)
}
