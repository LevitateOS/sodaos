//! OCI archive inspection (`oci.go`): identity checks over real OCI
//! archives without extracting layers. Streams the tar, verifies blob
//! digests, and resolves exact member hashes through the layer overlay.
//!
//! Identity checks adapted from soda-os bc1d3e0 release/inspection.go.

use crate::json_go::{FieldError, Fields};
use crate::{io_error, Error};
use std::collections::{BTreeMap, HashMap};
use std::path::Path;

pub(crate) mod archive;
mod content;
mod layers;
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Image {
    pub manifest: String,
    pub config: String,
    pub architecture: String,
    pub revision: String,
    pub source: String,
    pub base_name: String,
    pub base_digest: String,
}

impl Image {
    /// Go-`Encoder`-compatible compact JSON document: struct field order
    /// (`Manifest` first, `BaseDigest` last), Go string escaping, no
    /// trailing newline (the calling binary's `println!` adds it, matching
    /// `json.Encoder.Encode`).
    pub fn marshal_compact(&self) -> String {
        let mut out = String::new();
        out.push('{');
        let fields = [
            ("Manifest", &self.manifest),
            ("Config", &self.config),
            ("Architecture", &self.architecture),
            ("Revision", &self.revision),
            ("Source", &self.source),
            ("BaseName", &self.base_name),
            ("BaseDigest", &self.base_digest),
        ];
        for (i, (name, value)) in fields.iter().enumerate() {
            if i > 0 {
                out.push(',');
            }
            out.push('"');
            out.push_str(name);
            out.push_str("\":");
            soda_json::escape_into(&mut out, value);
        }
        out.push('}');
        out
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

impl Descriptor {
    fn decode(fields: &Fields<'_>) -> Result<Descriptor, FieldError> {
        Ok(Descriptor {
            digest: fields.string("digest")?,
            size: fields.int("size")?,
            media_type: fields.string("mediaType")?,
            urls: fields.string_list("urls")?,
            annotations: fields.string_map("annotations")?,
        })
    }

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
    let wanted = layers::requested_oci_paths(paths)?;
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
