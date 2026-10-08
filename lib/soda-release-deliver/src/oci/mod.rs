//! Port of `internal/release/build` OCI inspection (`oci.go`,
//! `oci_layout.go`) consumed by `deliver`: single-image archive identity,
//! rootfs content hashes, and shared-layout verification. Streams archives
//! without extracting layers or importing release code.

use std::collections::BTreeMap;
use std::io::Read;

use crate::buildx::{
    is_digest, is_revision, oci_architecture, open_layout_entry, Image as BuildImage, Root,
};
use crate::model::path_clean;
use crate::Error;

const MANIFEST_TYPE: &str = "application/vnd.oci.image.manifest.v1+json";
const CONFIG_TYPE: &str = "application/vnd.oci.image.config.v1+json";
const INDEX_TYPE: &str = "application/vnd.oci.image.index.v1+json";
const LAYER_TAR: &str = "application/vnd.oci.image.layer.v1.tar";
const LAYER_GZIP: &str = "application/vnd.oci.image.layer.v1.tar+gzip";
const LAYER_ZSTD: &str = "application/vnd.oci.image.layer.v1.tar+zstd";

/// `build.OCILayout`: verified shared-blob directory.
#[derive(Debug, Clone, Default)]
pub struct OciLayout {
    pub images: BTreeMap<String, BuildImage>,
    pub files: BTreeMap<String, String>,
    pub bytes: u64,
}

#[derive(Debug, Clone, Default)]
struct Blob {
    hash: String,
    size: i64,
    data: Option<Vec<u8>>,
}

mod schema;
use schema::{
    fetch_oci_blob, inspect_oci_config, inspect_oci_image, parse_oci_manifest, read_oci_blob,
    read_oci_index, BlobSource, Descriptor, OciManifest,
};

mod archive;
use archive::{inspect_archive_index, open_oci_archive, read_oci_archive_entries};

/// `build.InspectOCI`: verify single-image archive identity.
pub fn inspect_oci(file: &str, arch: &str, revision: &str) -> Result<BuildImage, Error> {
    let (want, f) = open_oci_archive(file, arch, revision)?;
    let entries = read_oci_archive_entries(f)?;
    inspect_archive_index(&entries, &want, revision)
}

mod layers;
use layers::{layer_archive_indexes, resolve_oci_members, scan_archive_layer_with_budget};
pub use layers::{
    requested_oci_paths, scan_oci_layer, scan_oci_layer_with_budget, LayerMember,
    MAX_OCI_IMAGE_COMPRESSED_BYTES, MAX_OCI_IMAGE_LAYER_BYTES, MAX_OCI_LAYER_BYTES,
    MAX_OCI_LAYER_COMPRESSED_BYTES, MAX_OCI_MEMBER_BYTES,
};

fn inspect_content_manifest(
    entries: &BTreeMap<String, Blob>,
    want: &str,
    revision: &str,
) -> Result<(BuildImage, OciManifest), Error> {
    let index = read_oci_index(entries)?;
    if index.len() != 1 || index[0].media_type != MANIFEST_TYPE {
        return Err(Error::msg("single-platform OCI index required"));
    }
    let image = inspect_oci_image(entries, &index[0], want, revision)?;
    let manifest_blob = fetch_oci_blob(entries, &index[0])?;
    let empty = Vec::new();
    let manifest = parse_oci_manifest(manifest_blob.data.as_ref().unwrap_or(&empty))?;
    Ok((image, manifest))
}

/// `build.InspectOCIContent`: identity plus exact rootfs member hashes.
pub fn inspect_oci_content(
    file: &str,
    arch: &str,
    revision: &str,
    paths: &[String],
) -> Result<(BuildImage, BTreeMap<String, String>), Error> {
    let wanted = requested_oci_paths(paths)?;
    let (want, mut f) = open_oci_archive(file, arch, revision)?;
    let entries = read_oci_archive_entries(&f)?;
    let (image, manifest) = inspect_content_manifest(&entries, &want, revision)?;
    use std::io::Seek;
    f.seek(std::io::SeekFrom::Start(0))
        .map_err(|e| Error::msg(format!("seek {file}: {e}")))?;
    let (layers, unsupported) = scan_oci_archive_layers(&mut f, &manifest.layers, &wanted)?;
    let content = resolve_oci_members(&layers, &unsupported, &wanted)?;
    Ok((image, content))
}

type LayerScan = (Vec<BTreeMap<String, LayerMember>>, Vec<bool>);

fn scan_oci_archive_layers<R: Read>(
    reader: R,
    layers: &[Descriptor],
    wanted: &BTreeMap<String, String>,
) -> Result<LayerScan, Error> {
    let indexes = layer_archive_indexes(layers)?;
    let mut found: Vec<BTreeMap<String, LayerMember>> = Vec::new();
    found.resize_with(layers.len(), BTreeMap::new);
    let mut unsupported = vec![false; layers.len()];
    let mut archive = tar::Archive::new(reader);
    let mut decoded_total = 0u64;
    let items = archive
        .entries()
        .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
    for item in items {
        let mut item = item.map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        let raw_name = String::from_utf8_lossy(&item.path_bytes()).into_owned();
        let positions = indexes
            .get(&path_clean(&raw_name))
            .cloned()
            .unwrap_or_default();
        if positions.is_empty() {
            continue;
        }
        let (members, blocked) = scan_archive_layer_with_budget(
            &mut item,
            &layers[positions[0]],
            wanted,
            &mut decoded_total,
        )?;
        for position in positions {
            found[position] = members.clone();
            unsupported[position] = blocked;
        }
    }
    Ok((found, unsupported))
}

// ---------------------------------------------------------------------------
// Shared layout
// ---------------------------------------------------------------------------

fn validate_oci_layout_inputs(
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<String, Error> {
    let want = oci_architecture(arch).map_err(|e| Error::msg(e.0))?;
    if revisions.is_empty() {
        return Err(Error::msg("explicit OCI image set required"));
    }
    for (reference, revision) in revisions {
        let hex = reference.strip_prefix("sha256:").unwrap_or("");
        if !reference.starts_with("sha256:")
            || !is_digest(hex)
            || (!revision.is_empty() && !is_revision(revision))
        {
            return Err(Error::msg("invalid OCI image selection"));
        }
    }
    Ok(want.to_string())
}

/// `build.InspectOCILayout`: verify the exact named image set and blobs.
pub fn inspect_oci_layout(
    dir: &str,
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<OciLayout, Error> {
    let want = validate_oci_layout_inputs(arch, revisions)?;
    let info = std::fs::symlink_metadata(dir)
        .map_err(|_| Error::msg("real OCI layout directory required"))?;
    if !info.is_dir() {
        return Err(Error::msg("real OCI layout directory required"));
    }
    let root = Root::open(dir).map_err(|_| Error::msg("real OCI layout directory required"))?;
    let mut loader = LayoutLoader {
        root: &root,
        entries: BTreeMap::new(),
        json_bytes: 0,
    };
    for name in ["index.json", "oci-layout"] {
        loader.load(name)?;
    }
    let index = read_oci_index(&loader.entries)?;
    let images = inspect_layout_images(&mut loader, &index, &want, revisions)?;
    let mut files: BTreeMap<String, String> = BTreeMap::new();
    let mut total: u64 = 0;
    for (name, entry) in &loader.entries {
        files.insert(name.clone(), entry.hash.clone());
        total += entry.size as u64;
    }
    Ok(OciLayout {
        images,
        files,
        bytes: total,
    })
}

struct LayoutLoader<'a> {
    root: &'a Root,
    entries: BTreeMap<String, Blob>,
    json_bytes: i64,
}

impl LayoutLoader<'_> {
    fn load(&mut self, name: &str) -> Result<(), Error> {
        if self.entries.contains_key(name) {
            return Ok(());
        }
        let (file, size) = open_layout_entry(self.root, name)?;
        read_oci_blob(
            &mut self.entries,
            name,
            size,
            file,
            BlobSource::Layout,
            &mut self.json_bytes,
        )
    }

    fn fetch(&mut self, d: &Descriptor) -> Result<Blob, Error> {
        if d.size < 0 || !d.urls.is_empty() {
            return Err(Error::msg("local bounded OCI descriptor required"));
        }
        let hex = d.digest.strip_prefix("sha256:").unwrap_or("");
        if !d.digest.starts_with("sha256:") || !is_digest(hex) {
            return Err(Error::msg("invalid OCI digest"));
        }
        let name = format!("blobs/sha256/{hex}");
        self.load(&name)?;
        match self.entries.get(&name) {
            Some(blob) if blob.size == d.size => Ok(blob.clone()),
            _ => Err(Error::msg("missing or wrong-size OCI blob")),
        }
    }

    fn inspect_image(
        &mut self,
        image: &Descriptor,
        want: &str,
        revision: &str,
    ) -> Result<BuildImage, Error> {
        let manifest_blob = self.fetch(image)?;
        let empty = Vec::new();
        let manifest = parse_oci_manifest(manifest_blob.data.as_ref().unwrap_or(&empty))?;
        for layer in &manifest.layers {
            match layer.media_type.as_str() {
                LAYER_TAR | LAYER_GZIP | LAYER_ZSTD => {}
                _ => return Err(Error::msg("unsupported OCI layer media type")),
            }
            self.fetch(layer)?;
        }
        let config_blob = self.fetch(&manifest.config)?;
        inspect_oci_config(
            &config_blob,
            &manifest.layers,
            want,
            revision,
            &image.digest,
            &manifest.config.digest,
        )
    }
}

fn inspect_layout_images(
    loader: &mut LayoutLoader<'_>,
    index: &[Descriptor],
    want: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, BuildImage>, Error> {
    if index.len() != revisions.len() {
        return Err(Error::msg("exact OCI image set required"));
    }
    let mut images: BTreeMap<String, BuildImage> = BTreeMap::new();
    for descriptor in index {
        let reference = descriptor
            .annotations
            .get("org.opencontainers.image.ref.name")
            .cloned()
            .unwrap_or_default();
        let revision = match revisions.get(&reference) {
            Some(revision) => revision.clone(),
            None => return Err(Error::msg("unexpected or duplicate OCI image reference")),
        };
        if images
            .get(&reference)
            .map(|image: &BuildImage| !image.config.is_empty())
            .unwrap_or(false)
            || descriptor.media_type != MANIFEST_TYPE
        {
            return Err(Error::msg("unexpected or duplicate OCI image reference"));
        }
        let image = loader.inspect_image(descriptor, want, &revision)?;
        if image.config != reference {
            return Err(Error::msg("OCI reference differs from config identity"));
        }
        images.insert(reference, image);
    }
    Ok(images)
}

#[cfg(test)]
mod tests {
    use super::*;
    use sha2::{Digest as _, Sha256};

    #[test]
    fn member_path_rules() {
        // Path validation runs before archive opening.
        let err = inspect_oci_content("/nonexistent.oci", "x86_64", "", &[]).unwrap_err();
        assert_eq!(err, Error::msg("explicit OCI members required"));
        let err = inspect_oci_content("/nonexistent.oci", "x86_64", "", &["relative".to_string()])
            .unwrap_err();
        assert_eq!(err, Error::msg("absolute clean OCI member paths required"));
        let err = inspect_oci("/nonexistent.oci", "aarch64", "").unwrap_err();
        assert_eq!(err, Error::msg("expected x86_64"));
        let err = inspect_oci("/nonexistent.oci", "x86_64", "short").unwrap_err();
        assert_eq!(err, Error::msg("full source revision required"));
    }

    #[test]
    fn non_archive_refused() {
        let dir = std::env::temp_dir().join(format!("srd-oci-unit-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("junk.oci").to_string_lossy().into_owned();
        std::fs::write(&path, b"definitely not a tar archive ....................").unwrap();
        let err = inspect_oci(&path, "x86_64", "").unwrap_err();
        assert!(err.0.contains("read OCI archive"), "{}", err.0);
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn layout_and_tar_blob_callers_stream_large_bodies_without_retention() {
        let temp = tempfile::tempdir().unwrap();
        let root_path = temp.path().join("layout");
        let blob_dir = root_path.join("blobs/sha256");
        std::fs::create_dir_all(&blob_dir).unwrap();
        let body = vec![b'x'; (6 << 20) + 19];
        let digest = format!("{:x}", Sha256::digest(&body));
        let name = format!("blobs/sha256/{digest}");
        std::fs::write(root_path.join(&name), &body).unwrap();

        let root = Root::open(root_path.to_str().unwrap()).unwrap();
        let mut loader = LayoutLoader {
            root: &root,
            entries: BTreeMap::new(),
            json_bytes: 0,
        };
        loader.load(&name).unwrap();
        let layout_blob = &loader.entries[&name];
        assert_eq!(layout_blob.hash, digest);
        assert_eq!(layout_blob.size, body.len() as i64);
        assert!(layout_blob.data.is_none());

        let archive_path = temp.path().join("large.oci");
        let archive_file = std::fs::File::create(&archive_path).unwrap();
        let mut archive = tar::Builder::new(archive_file);
        let mut header = tar::Header::new_gnu();
        header.set_size(body.len() as u64);
        header.set_mode(0o644);
        header.set_cksum();
        archive
            .append_data(&mut header, &name, body.as_slice())
            .unwrap();
        let layout = br#"{"imageLayoutVersion":"1.0.0"}"#;
        let mut layout_header = tar::Header::new_gnu();
        layout_header.set_size(layout.len() as u64);
        layout_header.set_mode(0o644);
        layout_header.set_cksum();
        archive
            .append_data(&mut layout_header, "oci-layout", layout.as_slice())
            .unwrap();
        archive.finish().unwrap();

        let archive_entries =
            read_oci_archive_entries(std::fs::File::open(archive_path).unwrap()).unwrap();
        let tar_blob = &archive_entries[&name];
        assert_eq!(tar_blob.hash, digest);
        assert_eq!(tar_blob.size, body.len() as i64);
        assert!(tar_blob.data.is_none());
        assert_eq!(
            archive_entries["oci-layout"].data.as_deref(),
            Some(layout.as_slice())
        );
    }
}
