//! `content.go`: shared OCI content verification.

use std::collections::BTreeMap;

use crate::oci::{inspect_oci_layout, OciLayout};
use crate::payload::{Payload, Role};
use crate::Error;

fn bind_image_revisions(p: &Payload) -> Result<BTreeMap<String, String>, Error> {
    use soda_build_tools::payload::ImageProvenance;
    let mut revisions: BTreeMap<String, String> = BTreeMap::new();
    for role in Role::iter() {
        let binding = role.binding(&p.images);
        let revision = match binding.provenance {
            ImageProvenance::Soda { .. } => p.revision.clone(),
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

fn match_layout_identities(p: &Payload, layout: &OciLayout) -> Result<(), Error> {
    for role in Role::iter() {
        let name = role.name();
        let expected = role.binding(&p.images).clone();
        let got = layout
            .images
            .get(&expected.config)
            .cloned()
            .unwrap_or_default();
        if got.config != expected.config || got.manifest != expected.manifest {
            return Err(Error::msg(format!("{name} OCI identity mismatch")));
        }
    }
    Ok(())
}

/// `VerifyContent`: bind shared OCI content to the payload.
pub fn verify_content(p: &Payload, images: &str) -> Result<(BTreeMap<String, String>, u64), Error> {
    p.validate().map_err(|e| Error::msg(e.to_string()))?;
    if !images.starts_with('/') || images.bytes().any(|b| matches!(b, b':' | b'\r' | b'\n')) {
        return Err(Error::msg("absolute local OCI directory required"));
    }
    let revisions = bind_image_revisions(p)?;
    let layout = inspect_oci_layout(images, &p.architecture, &revisions)?;
    match_layout_identities(p, &layout)?;
    Ok((layout.files, layout.bytes))
}

#[cfg(test)]
mod tests {
    use super::*;

    const FIXTURE: &str = include_str!("../../release-inputs/fixtures/payload-v4.json");

    #[test]
    fn content_path_rules() {
        let payload: Payload = crate::buildx::decode_build_json(FIXTURE.as_bytes()).unwrap();
        payload.validate().unwrap();
        // The shared layout check runs after payload validation.
        assert!(verify_content(&payload, "/abs").is_err());
        assert!(verify_content(&payload, "relative").is_err());
    }
}
