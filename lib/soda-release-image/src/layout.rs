//! `build_layout.go`: shared OCI layout staging for the host image.

use std::collections::HashMap;
use std::fs;
use std::os::unix::fs::PermissionsExt;

use crate::error::Error;
use crate::foreign::Production;
use crate::model;
use crate::sys;

/// Skopeo owns OCI export and shared-blob reuse. Detached per-image archives
/// stay untouched for delivery/provenance; only the shared layout enters the
/// host image.
pub fn valid_stage_layout(
    archives: &str,
    destination: &str,
    payload: &model::Payload,
    runner: Option<&dyn Production>,
) -> bool {
    payload.format == 3
        && runner.is_some()
        && sys::is_abs(archives)
        && sys::is_abs(destination)
        && !(archives.to_string() + destination)
            .chars()
            .any(|c| c == ':' || c == '\r' || c == '\n')
}

pub fn verify_archive_digests(archives: &str, payload: &model::Payload) -> Result<(), Error> {
    for name in model::NAMES {
        let hash = sys::hash_file(&sys::join(&[archives, &format!("{name}.oci")]));
        let want = payload.image(name).archive_sha256;
        match hash {
            Ok(hash) if hash == want => {}
            _ => return Err(Error::msg(format!("{name} archive changed before staging"))),
        }
    }
    Ok(())
}

pub fn copy_staged_archives(
    archives: &str,
    destination: &str,
    payload: &model::Payload,
    production: &dyn Production,
) -> Result<(), Error> {
    for name in model::NAMES {
        production.execute(
            archives,
            "skopeo",
            &[
                "copy".to_string(),
                "--preserve-digests".to_string(),
                "--dest-oci-accept-uncompressed-layers".to_string(),
                format!(
                    "oci-archive:{}",
                    sys::join(&[archives, &format!("{name}.oci")])
                ),
                format!("oci:{destination}:{}", payload.image(name).config),
            ],
        )?;
    }
    Ok(())
}

pub fn chmod_staged_files(destination: &str, files: &HashMap<String, String>) -> Result<(), Error> {
    for name in files.keys() {
        fs::set_permissions(
            sys::join(&[destination, name]),
            fs::Permissions::from_mode(0o644),
        )?;
    }
    Ok(())
}

pub fn stage_images(
    archives: &str,
    destination: &str,
    payload: &model::Payload,
    production: Option<&dyn Production>,
) -> Result<(), Error> {
    payload.validate()?;
    if !valid_stage_layout(archives, destination, payload, production) {
        return Err(Error::msg("explicit local v3 layout staging required"));
    }
    let production = production.unwrap();
    verify_archive_digests(archives, payload)?;
    fs::create_dir_all(sys::dir_name(destination))?;
    sys::create_dir(destination, 0o755)?;
    copy_staged_archives(archives, destination, payload, production)?;
    let (files, _) = production.verify_content(payload, destination)?;
    chmod_staged_files(destination, &files)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_layout_staging_refuses_before_copies() {
        // Oracle: Go TestSharedLayoutStagingRefusesBeforeCopiesAndStopsOnFailure.
        let payload = model::Payload::default();
        assert!(!valid_stage_layout("/a", "/b", &payload, None));
        assert_eq!(
            stage_images("/a", "/b", &payload, None).unwrap_err().0,
            "invalid appliance payload identity"
        );
    }
}
