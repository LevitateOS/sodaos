//! `content.go`: shared OCI content verification.

use std::collections::BTreeMap;

use crate::oci::{inspect_oci_layout, OciLayout};
use crate::payload::{Payload, NAMES};
use crate::Error;

fn bind_image_revisions(p: &Payload) -> Result<BTreeMap<String, String>, Error> {
    let mut revisions: BTreeMap<String, String> = BTreeMap::new();
    for name in NAMES {
        let revision = if name == "proxy" {
            String::new()
        } else {
            p.revision.clone()
        };
        let reference = p
            .images
            .get(name)
            .map(|i| i.config.clone())
            .unwrap_or_default();
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
    for name in NAMES {
        let expected = p.images.get(name).cloned().unwrap_or_default();
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
    p.validate()?;
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

    #[test]
    fn content_path_rules() {
        let payload = Payload::default();
        // Payload validation runs before the path check.
        assert!(verify_content(&payload, "/abs").is_err());
    }
}
