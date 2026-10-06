use std::collections::HashMap;
use std::fs;
use std::io::Read;
use std::os::unix::fs::MetadataExt;
use std::path::Path;

use soda_json::JsonValue;

use super::{
    is_coreos_version, is_digest, is_prefixed_digest, is_revision, oci_architecture, parse_json,
    valid_repository_prefix, Binder, NAMES,
};

// ---------- appliance payload (deliver/payload.go) ----------

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

fn decode_image_binding(value: &JsonValue) -> Result<ImageBinding, String> {
    let mut binder = Binder::new(value)?;
    let binding = ImageBinding {
        reference: binder.string("Reference")?,
        config: binder.string("Config")?,
        manifest: binder.string("Manifest")?,
        archive_sha256: binder.string("ArchiveSHA256")?,
    };
    binder.finish()?;
    Ok(binding)
}

pub(super) fn decode_payload(value: &JsonValue) -> Result<Payload, String> {
    let mut binder = Binder::new(value)?;
    let mut payload = Payload {
        format: binder.int("Format")?,
        id: binder.string("ID")?,
        revision: binder.string("Revision")?,
        architecture: binder.string("Architecture")?,
        coreos: binder.string("CoreOS")?,
        base: binder.string("Base")?,
        repository_prefix: binder.string("RepositoryPrefix")?,
        schema: binder.int("Schema")?,
        presentation_sha256: binder.string("PresentationSHA256")?,
        host_packages_sha256: binder.string("HostPackagesSHA256")?,
        images: HashMap::new(),
        upgrade_from: binder.string_list("UpgradeFrom")?,
    };
    if let Some(images) = binder.object("Images")? {
        match images {
            JsonValue::Object(entries) => {
                for (name, item) in entries {
                    payload
                        .images
                        .insert(name.clone(), decode_image_binding(item)?);
                }
            }
            _ => return Err("field Images must be an object".to_string()),
        }
    }
    binder.finish()?;
    Ok(payload)
}

impl Payload {
    fn valid_identity(&self) -> bool {
        if self.format != 3 || !is_revision(&self.revision) {
            return false;
        }
        if self.id != format!("{}.soda-{}", self.coreos, &self.revision[..12]) {
            return false;
        }
        is_coreos_version(&self.coreos)
    }

    fn valid_base(&self) -> bool {
        let (host, digest) = match self.base.split_once("/fedora/fedora-coreos@sha256:") {
            Some(pair) => pair,
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
            (Some(extension), Some(forgejo)) if extension.config != forgejo.config => Ok(()),
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
        if !valid_repository_prefix(&self.repository_prefix) {
            return Err("explicit GHCR repository prefix required".to_string());
        }
        if !self.valid_base() {
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
        let value = parse_json(&data)?;
        let payload = decode_payload(&value)?;
        payload.validate()?;
        Ok(payload)
    }
}

/// Bounded regular-file JSON read with open-time identity check, mirroring
/// `build.ReadJSON`: symlink or oversized input refused, same-file verified.
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
