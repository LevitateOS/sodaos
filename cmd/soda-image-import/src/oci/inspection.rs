use std::collections::HashMap;
use std::path::Path;

use super::super::{is_digest, is_revision, oci_architecture};
use super::layout::{
    open_layout_root, validate_oci_attribution, validate_oci_layers, validate_oci_rootfs,
    LayoutLoader,
};
use super::metadata::{
    parse_oci_config, parse_oci_manifest, read_oci_index, OciDescriptor, OciImage,
};

fn inspect_oci_image(
    loader: &mut LayoutLoader,
    image: &OciDescriptor,
    want: &str,
    revision: &str,
) -> Result<OciImage, String> {
    let manifest_blob = loader.fetch(image)?;
    let manifest = parse_oci_manifest(manifest_blob.data.as_deref().unwrap_or(b""))?;
    validate_oci_layers(loader, &manifest.layers)?;
    let config_blob = loader.fetch(&manifest.config)?;
    let config = parse_oci_config(config_blob.data.as_deref().unwrap_or(b""))?;
    if config.rootfs_type != "layers" {
        return Err("OCI rootfs/layer count mismatch".to_string());
    }
    validate_oci_rootfs(&config.diff_ids, &manifest.layers)?;
    if config.os != "linux" || config.arch != want {
        return Err(format!("OCI must be linux/{want}"));
    }
    let found_revision = validate_oci_attribution(&config.labels, revision)?;
    Ok(OciImage {
        manifest: image.digest.clone(),
        config: manifest.config.digest.clone(),
        architecture: config.arch,
        revision: found_revision,
        source: config
            .labels
            .get("org.opencontainers.image.source")
            .cloned()
            .unwrap_or_default(),
        base_name: config
            .labels
            .get("org.opencontainers.image.base.name")
            .cloned()
            .unwrap_or_default(),
        base_digest: config
            .labels
            .get("org.opencontainers.image.base.digest")
            .cloned()
            .unwrap_or_default(),
    })
}

/// Verify the exact named image set and its local blobs without running
/// image code. References are immutable config IDs, as in our OCI exports.
pub(crate) fn inspect_oci_layout(
    dir: &Path,
    arch: &str,
    revisions: &HashMap<String, String>,
) -> Result<HashMap<String, OciImage>, String> {
    let want = oci_architecture(arch)?;
    if revisions.is_empty() {
        return Err("explicit OCI image set required".to_string());
    }
    for (reference, revision) in revisions {
        let hex = reference.strip_prefix("sha256:").unwrap_or("");
        if !reference.starts_with("sha256:")
            || !is_digest(hex)
            || (!revision.is_empty() && !is_revision(revision))
        {
            return Err("invalid OCI image selection".to_string());
        }
    }
    open_layout_root(dir)?;
    let mut loader = LayoutLoader {
        dir: dir.to_path_buf(),
        entries: HashMap::new(),
        json_bytes: 0,
    };
    loader.load("index.json")?;
    loader.load("oci-layout")?;
    let index = read_oci_index(&loader.entries)?;
    if index.len() != revisions.len() {
        return Err("exact OCI image set required".to_string());
    }
    let mut images: HashMap<String, OciImage> = HashMap::with_capacity(index.len());
    for descriptor in &index {
        let reference = descriptor
            .annotations
            .get("org.opencontainers.image.ref.name")
            .cloned()
            .unwrap_or_default();
        let revision = match revisions.get(&reference) {
            Some(revision) => revision.clone(),
            None => {
                return Err("unexpected or duplicate OCI image reference".to_string());
            }
        };
        if images.contains_key(&reference)
            || descriptor.media_type != "application/vnd.oci.image.manifest.v1+json"
        {
            return Err("unexpected or duplicate OCI image reference".to_string());
        }
        let image = inspect_oci_image(&mut loader, descriptor, want, &revision)?;
        if image.config != reference {
            return Err("OCI reference differs from config identity".to_string());
        }
        images.insert(reference, image);
    }
    Ok(images)
}
