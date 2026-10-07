//! OCI archive inspection (`oci.go`): identity checks over real OCI
//! archives without extracting layers. Streams the tar, verifies blob
//! digests, and resolves exact member hashes through the layer overlay.
//!
//! Identity checks adapted from soda-os bc1d3e0 release/inspection.go.

use crate::json_emit::marshal_compact;
use crate::{io_error, Error};
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};
use std::collections::{BTreeMap, HashMap};
use std::path::Path;

pub(crate) mod archive;
mod content;
pub(crate) mod manifest;

#[cfg(test)]
mod tests;

pub(crate) const MANIFEST_MEDIA_TYPE: &str = "application/vnd.oci.image.manifest.v1+json";
const INDEX_MEDIA_TYPE: &str = "application/vnd.oci.image.index.v1+json";
pub(crate) const CONFIG_MEDIA_TYPE: &str = "application/vnd.oci.image.config.v1+json";
pub(crate) const LAYER_TAR: &str = "application/vnd.oci.image.layer.v1.tar";
const LAYER_TAR_GZIP: &str = "application/vnd.oci.image.layer.v1.tar+gzip";
const LAYER_TAR_ZSTD: &str = "application/vnd.oci.image.layer.v1.tar+zstd";

/// Verified image identity.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct Image {
    #[serde(rename = "Manifest")]
    pub manifest: String,
    #[serde(rename = "Config")]
    pub config: String,
    #[serde(rename = "Architecture")]
    pub architecture: String,
    #[serde(rename = "Revision")]
    pub revision: String,
    #[serde(rename = "Source")]
    pub source: String,
    #[serde(rename = "BaseName")]
    pub base_name: String,
    #[serde(rename = "BaseDigest")]
    pub base_digest: String,
}

impl Image {
    /// Go-`Encoder`-compatible compact JSON document: struct field order
    /// (`Manifest` first, `BaseDigest` last), Go string escaping, no
    /// trailing newline (the calling binary's `println!` adds it, matching
    /// `json.Encoder.Encode`).
    pub fn marshal_compact(&self) -> String {
        marshal_compact(self)
    }
}

/// On-demand blob loader for layouts; archives pass a no-op.
pub(crate) type LoadBlobs<'a> =
    &'a mut dyn FnMut(&mut HashMap<String, Blob>, &str) -> Result<(), Error>;

#[derive(Debug, Clone, Default)]
pub(crate) struct Descriptor {
    pub digest: String,
    pub size: i64,
    pub media_type: String,
    pub urls: Vec<String>,
    pub annotations: Vec<(String, String)>,
}

impl<'de> Deserialize<'de> for Descriptor {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct DescriptorVisitor;
        impl<'de> Visitor<'de> for DescriptorVisitor {
            type Value = Descriptor;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("OCI descriptor object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let (mut digest, mut size, mut media_type, mut urls, mut annotations) =
                    (None, None, None, None, None);
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("digest") {
                        digest = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("size") {
                        size = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("mediaType") {
                        media_type = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("urls") {
                        urls = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("annotations") {
                        annotations = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(Descriptor {
                    digest: raw_field(digest)?,
                    size: raw_integer_field(size)?,
                    media_type: raw_field(media_type)?,
                    urls: raw_field(urls)?,
                    annotations: raw_field::<StringPairs, A::Error>(annotations)?.0,
                })
            }
        }
        deserializer.deserialize_map(DescriptorVisitor)
    }
}

pub(super) fn raw_field<T: serde::de::DeserializeOwned + Default, E: de::Error>(
    raw: Option<Box<serde_json::value::RawValue>>,
) -> Result<T, E> {
    match raw {
        None => Ok(T::default()),
        Some(raw) => serde_json::from_str::<Option<T>>(raw.get())
            .map(Option::unwrap_or_default)
            .map_err(E::custom),
    }
}

pub(super) fn raw_integer_field<T, E>(raw: Option<Box<serde_json::value::RawValue>>) -> Result<T, E>
where
    T: TryFrom<i128> + Default,
    <T as TryFrom<i128>>::Error: std::fmt::Display,
    E: de::Error,
{
    match raw {
        None => Ok(T::default()),
        Some(raw) if raw.get() == "null" => Ok(T::default()),
        Some(raw) => crate::json_input::parse_integer_token(&raw)
            .ok_or_else(|| E::custom("invalid integer token"))
            .and_then(|value| T::try_from(value).map_err(E::custom)),
    }
}

#[derive(Default)]
pub(super) struct StringPairs(pub(super) Vec<(String, String)>);

impl<'de> Deserialize<'de> for StringPairs {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct PairsVisitor;
        impl<'de> Visitor<'de> for PairsVisitor {
            type Value = StringPairs;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("string map")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let mut pairs = Vec::new();
                while let Some(key) = map.next_key::<String>()? {
                    let value = map.next_value::<String>()?;
                    pairs.push((key, value));
                }
                Ok(StringPairs(pairs))
            }
        }
        deserializer.deserialize_map(PairsVisitor)
    }
}

impl Descriptor {
    pub fn annotation(&self, key: &str) -> Option<&str> {
        self.annotations
            .iter()
            .rev()
            .find(|(k, _)| k == key)
            .map(|(_, v)| v.as_str())
    }
}

pub(crate) struct Blob {
    pub hash: String,
    pub size: i64,
    pub data: Option<Vec<u8>>,
}

fn hex_digest(digest: &[u8]) -> String {
    let mut out = String::with_capacity(digest.len() * 2);
    for byte in digest {
        out.push_str(&format!("{byte:02x}"));
    }
    out
}

fn inspect_archive_index(
    entries: &mut HashMap<String, Blob>,
    want: &str,
    revision: &str,
) -> Result<Image, Error> {
    let index = manifest::read_oci_index(entries)?;
    if index.len() != 1 || index[0].media_type != MANIFEST_MEDIA_TYPE {
        return Err(Error::msg("single-platform OCI index required"));
    }
    let mut no_load = |_: &mut HashMap<String, Blob>, _: &str| -> Result<(), Error> { Ok(()) };
    manifest::inspect_oci_image(entries, &index[0], want, revision, {
        let loader: LoadBlobs<'_> = &mut no_load;
        loader
    })
}

/// Verifies a single-platform OCI archive's identity without running it.
pub fn inspect_oci(file: &Path, arch: &str, revision: &str) -> Result<Image, Error> {
    let (want, f) = archive::open_oci_archive(file, arch, revision)?;
    let mut reader = std::io::BufReader::new(f);
    let mut entries = archive::read_oci_archive_entries(&mut reader)?;
    inspect_archive_index(&mut entries, &want, revision)
}

fn inspect_content_manifest(
    entries: &mut HashMap<String, Blob>,
    want: &str,
    revision: &str,
) -> Result<(Image, manifest::OciManifest), Error> {
    let index = manifest::read_oci_index(entries)
        .map_err(|_| Error::msg("single-platform OCI index required"))?;
    if index.len() != 1 || index[0].media_type != MANIFEST_MEDIA_TYPE {
        return Err(Error::msg("single-platform OCI index required"));
    }
    let mut no_load = |_: &mut HashMap<String, Blob>, _: &str| -> Result<(), Error> { Ok(()) };
    let image = manifest::inspect_oci_image(entries, &index[0], want, revision, {
        let loader: LoadBlobs<'_> = &mut no_load;
        loader
    })?;
    let (_, _, manifest_data) = manifest::fetch_oci_blob(entries, &index[0], {
        let loader: LoadBlobs<'_> = &mut no_load;
        loader
    })?;
    let manifest_data = manifest_data.unwrap_or_default();
    let manifest = manifest::parse_oci_manifest(&manifest_data)?;
    Ok((image, manifest))
}

/// Extends the identity check with hashes of exact regular files as they
/// appear in the verified image rootfs overlay.
pub fn inspect_oci_content(
    file: &Path,
    arch: &str,
    revision: &str,
    paths: &[String],
) -> Result<(Image, BTreeMap<String, String>), Error> {
    let wanted = soda_release_deliver::oci::requested_oci_paths(paths)
        .map_err(|error| Error::msg(error.to_string()))?;
    let (want, mut f) = archive::open_oci_archive(file, arch, revision)?;
    let entries = {
        let mut reader = std::io::BufReader::new(&mut f);
        archive::read_oci_archive_entries(&mut reader)?
    };
    let mut entries = entries;
    let (image, manifest) = inspect_content_manifest(&mut entries, &want, revision)?;
    use std::io::Seek;
    f.seek(std::io::SeekFrom::Start(0))
        .map_err(|e| io_error("seek", file, e))?;
    let mut reader = std::io::BufReader::new(&mut f);
    let (layers, unsupported) =
        content::scan_oci_archive_layers(&mut reader, &manifest.layers, &wanted)?;
    let content = content::resolve_oci_members(&layers, &unsupported, &wanted)?;
    Ok((image, content))
}
