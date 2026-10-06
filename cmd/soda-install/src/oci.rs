//! Shared-blob OCI layout verification ported from
//! `internal/release/build` (`oci_layout.go` plus the reachable half of
//! `oci.go`). Only the installer's `InspectOCILayout` path is ported: exact
//! image-set identity, blob hashing, and byte counts. Archive and layer
//! member inspection stay in Go with their callers.

use std::collections::BTreeMap;

use soda_json::JsonValue;

use self::layout::{load_layout_file, open_root, Blob, Descriptor, Loader};
pub use self::layout::{OciImage, OciLayout};
use self::metadata::{parse_oci_config, parse_oci_manifest, read_oci_index};
use crate::buildx;
use crate::errors::Error;
use crate::jsongo::parse;

const MANIFEST_MEDIA_TYPE: &str = "application/vnd.oci.image.manifest.v1+json";
const CONFIG_MEDIA_TYPE: &str = "application/vnd.oci.image.config.v1+json";
const INDEX_MEDIA_TYPE: &str = "application/vnd.oci.image.index.v1+json";
const LAYER_TAR: &str = "application/vnd.oci.image.layer.v1.tar";
const LAYER_GZIP: &str = "application/vnd.oci.image.layer.v1.tar+gzip";
const LAYER_ZSTD: &str = "application/vnd.oci.image.layer.v1.tar+zstd";

// ---------------------------------------------------------------------------
// Image inspection.
// ---------------------------------------------------------------------------

fn fetch_oci_blob(loader: &mut Loader, desc: &Descriptor) -> Result<Blob, Error> {
    if desc.size < 0 || !desc.urls.is_empty() {
        return Err(Error::msg("local bounded OCI descriptor required"));
    }
    let hex = match desc.digest.strip_prefix("sha256:") {
        Some(hex) if buildx::digest(hex) => hex,
        _ => return Err(Error::msg("invalid OCI digest")),
    };
    let name = format!("blobs/sha256/{hex}");
    load_layout_file(loader, &name)?;
    match loader.entries.get(&name) {
        Some(blob) if blob.size == desc.size => Ok(blob.clone()),
        _ => Err(Error::msg("missing or wrong-size OCI blob")),
    }
}

fn validate_oci_layers(loader: &mut Loader, layers: &[Descriptor]) -> Result<(), Error> {
    for layer in layers {
        match layer.media_type.as_str() {
            LAYER_TAR | LAYER_GZIP | LAYER_ZSTD => {}
            _ => return Err(Error::msg("unsupported OCI layer media type")),
        }
        fetch_oci_blob(loader, layer)?;
    }
    Ok(())
}

fn validate_oci_rootfs(diff_ids: &[String], layers: &[Descriptor]) -> Result<(), Error> {
    if diff_ids.len() != layers.len() {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    for (i, id) in diff_ids.iter().enumerate() {
        let hex = match id.strip_prefix("sha256:") {
            Some(hex) if buildx::digest(hex) => hex,
            _ => return Err(Error::msg("invalid OCI diff ID")),
        };
        let _ = hex;
        if layers[i].media_type == LAYER_TAR && *id != layers[i].digest {
            return Err(Error::msg("uncompressed OCI layer identity mismatch"));
        }
    }
    Ok(())
}

fn validate_oci_attribution(
    labels: &BTreeMap<String, String>,
    want_revision: &str,
) -> Result<(), Error> {
    let rev = labels
        .get("org.opencontainers.image.revision")
        .cloned()
        .unwrap_or_default();
    if !want_revision.is_empty() && rev != want_revision {
        return Err(Error::msg("OCI source revision mismatch"));
    }
    if want_revision.is_empty() {
        return Ok(());
    }
    let base_digest = labels
        .get("org.opencontainers.image.base.digest")
        .cloned()
        .unwrap_or_default();
    let source_ok = labels
        .get("org.opencontainers.image.source")
        .map(|s| s.as_str())
        == Some("https://github.com/LevitateOS/sodaos");
    let base_name_ok = labels
        .get("org.opencontainers.image.base.name")
        .map(|s| !s.is_empty())
        .unwrap_or(false);
    let base_digest_ok = match base_digest.strip_prefix("sha256:") {
        Some(hex) => buildx::digest(hex),
        None => false,
    };
    if !source_ok || !base_name_ok || !base_digest_ok {
        return Err(Error::msg("soda image lacks source/base attribution"));
    }
    Ok(())
}

fn inspect_oci_config(
    config_blob: &Blob,
    layers: &[Descriptor],
    want: &str,
    revision: &str,
    image_digest: &str,
    config_digest: &str,
) -> Result<OciImage, Error> {
    let data = config_blob.data.as_deref().unwrap_or(b"");
    let cfg = parse_oci_config(data)?;
    if cfg.rootfs_type != "layers" {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    validate_oci_rootfs(&cfg.diff_ids, layers)?;
    // Compressed layer contents are not extracted here. Their blob identities
    // are checked; native import remains the proof of decompression/rootfs use.
    if cfg.os != "linux" || cfg.arch != want {
        return Err(Error::msg(format!("OCI must be linux/{want}")));
    }
    validate_oci_attribution(&cfg.labels, revision)?;
    Ok(OciImage {
        manifest: image_digest.to_string(),
        config: config_digest.to_string(),
        architecture: cfg.arch,
        revision: cfg
            .labels
            .get("org.opencontainers.image.revision")
            .cloned()
            .unwrap_or_default(),
        source: cfg
            .labels
            .get("org.opencontainers.image.source")
            .cloned()
            .unwrap_or_default(),
        base_name: cfg
            .labels
            .get("org.opencontainers.image.base.name")
            .cloned()
            .unwrap_or_default(),
        base_digest: cfg
            .labels
            .get("org.opencontainers.image.base.digest")
            .cloned()
            .unwrap_or_default(),
    })
}

fn inspect_oci_image(
    loader: &mut Loader,
    image: &Descriptor,
    want: &str,
    revision: &str,
) -> Result<OciImage, Error> {
    let manifest_blob = fetch_oci_blob(loader, image)?;
    let manifest = parse_oci_manifest(manifest_blob.data.as_deref().unwrap_or(b""))?;
    validate_oci_layers(loader, &manifest.layers)?;
    let config_blob = fetch_oci_blob(loader, &manifest.config)?;
    inspect_oci_config(
        &config_blob,
        &manifest.layers,
        want,
        revision,
        &image.digest,
        &manifest.config.digest,
    )
}

fn validate_oci_layout_inputs(
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<String, Error> {
    let want = buildx::oci_architecture(arch)?;
    if revisions.is_empty() {
        return Err(Error::msg("explicit OCI image set required"));
    }
    for (reference, revision) in revisions {
        let digest_ok = match reference.strip_prefix("sha256:") {
            Some(hex) => buildx::digest(hex),
            None => false,
        };
        if !digest_ok || (!revision.is_empty() && !buildx::revision(revision)) {
            return Err(Error::msg("invalid OCI image selection"));
        }
    }
    Ok(want)
}

fn inspect_layout_descriptor(
    loader: &mut Loader,
    desc: &Descriptor,
    want: &str,
    revisions: &BTreeMap<String, String>,
    images: &BTreeMap<String, OciImage>,
) -> Result<(String, OciImage), Error> {
    let reference = desc
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
        .map(|image| !image.config.is_empty())
        .unwrap_or(false)
        || desc.media_type != MANIFEST_MEDIA_TYPE
    {
        return Err(Error::msg("unexpected or duplicate OCI image reference"));
    }
    let image = inspect_oci_image(loader, desc, want, &revision)?;
    if image.config != reference {
        return Err(Error::msg("OCI reference differs from config identity"));
    }
    Ok((reference, image))
}

fn inspect_layout_images(
    loader: &mut Loader,
    index: &[Descriptor],
    want: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, OciImage>, Error> {
    if index.len() != revisions.len() {
        return Err(Error::msg("exact OCI image set required"));
    }
    let mut images = BTreeMap::new();
    for desc in index {
        let (reference, image) = inspect_layout_descriptor(loader, desc, want, revisions, &images)?;
        images.insert(reference, image);
    }
    Ok(images)
}

/// Verifies the exact named image set and its local blobs without running
/// image code. References are immutable config IDs, as in our OCI exports.
pub fn inspect_oci_layout(
    dir: &str,
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<OciLayout, Error> {
    let want = validate_oci_layout_inputs(arch, revisions)?;
    let root = open_root(dir)?;
    let mut loader = Loader {
        root,
        entries: BTreeMap::new(),
        json_bytes: 0,
    };
    for name in ["index.json", "oci-layout"] {
        load_layout_file(&mut loader, name)?;
    }
    let index = read_oci_index(&loader.entries)?;
    let images = inspect_layout_images(&mut loader, &index, &want, revisions)?;
    let mut files = BTreeMap::new();
    let mut bytes: u64 = 0;
    for (name, entry) in &loader.entries {
        files.insert(name.clone(), entry.hash.clone());
        bytes += entry.size as u64;
    }
    Ok(OciLayout {
        images,
        files,
        bytes,
    })
}

mod layout;
mod metadata;

#[cfg(test)]
pub mod test_support;

#[cfg(test)]
mod tests;
