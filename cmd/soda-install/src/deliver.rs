//! Appliance payload metadata consumed from the canonical release-image
//! producer, plus shared-OCI content verification before installer writes.
//!
//! The wire types and validation are owned by soda-build-tools; this module
//! keeps only the installer's file-read admission and its local candidate
//! layout binding. There is no v3 branch: old documents fail validation
//! instead of being migrated.

use std::collections::BTreeMap;

use crate::buildx;
use crate::errors::Error;
use crate::oci;
use serde::Deserialize;

pub use soda_build_tools::payload::{PayloadV4 as Payload, Role};

pub const PATH: &str = "/usr/share/soda/release.json";
pub const IMAGES_PATH: &str = "/usr/share/soda/images";

fn bind_image_revisions(payload: &Payload) -> Result<BTreeMap<String, String>, Error> {
    use soda_build_tools::payload::ImageProvenance;
    let mut revisions = BTreeMap::new();
    for role in Role::iter() {
        let binding = role.binding(&payload.images);
        let revision = match binding.provenance {
            ImageProvenance::Soda { .. } => payload.revision.clone(),
            ImageProvenance::Upstream { .. } => String::new(),
        };
        let reference = binding.config.clone();
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
    for role in Role::iter() {
        let name = role.name();
        let expected = role.binding(&payload.images);
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
    payload.validate().map_err(|e| Error::msg(e.to_string()))?;
    if !images.starts_with('/') || images.contains([':', '\r', '\n']) {
        return Err(Error::msg("absolute local OCI directory required"));
    }
    let revisions = bind_image_revisions(payload)?;
    let layout = oci::inspect_oci_layout(images, &payload.architecture, &revisions)?;
    match_layout_identities(payload, &layout)?;
    Ok((layout.files, layout.bytes))
}

pub fn load(path: &str) -> Result<Payload, Error> {
    let data = buildx::read_json_bytes(path)?;
    let text = std::str::from_utf8(&data).map_err(|_| Error::msg("invalid payload document"))?;
    let mut deserializer = serde_json::Deserializer::from_str(text);
    let payload = Payload::deserialize(&mut deserializer)
        .map_err(|_| Error::msg("invalid payload document"))?;
    deserializer
        .end()
        .map_err(|_| Error::msg("invalid payload document"))?;
    payload.validate().map_err(|e| Error::msg(e.to_string()))?;
    Ok(payload)
}

#[cfg(test)]
mod tests;
