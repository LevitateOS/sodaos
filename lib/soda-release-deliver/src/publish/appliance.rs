//! Native Apple Silicon host and UTM guest readback for ARM appliance admission.
use super::{assets, Manifest};
use crate::{buildx::is_digest, model::FileIdentity, Error};
use serde::Deserialize;
use std::path::Path;

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct NativeAppliance {
    target: String,
    host_platform: String,
    runtime: String,
    accelerator: String,
    guest_platform: String,
    virtual_machine_id: String,
    soda_revision: String,
    iso_sha256: String,
    host_readback_sha256: String,
    guest_readback_sha256: String,
}

pub(super) fn observe(
    manifest: &Manifest,
    path: &Path,
    published_at: Option<&str>,
) -> Result<FileIdentity, Error> {
    let refused = || {
        Error::msg("ARM appliance qualification requires finalized Apple Silicon host and native UTM guest readback")
    };
    let _: serde_json::Value = assets::read_json(path)?;
    let (observation, hash) =
        soda_acceptance::report::read_observation(path.to_str().ok_or_else(refused)?)
            .map_err(|_| refused())?;
    let native_path = path.parent().ok_or_else(refused)?.join("native.json");
    let native: NativeAppliance = assets::read_json(&native_path)?;
    let native_file = assets::inspect(&native_path, "native.json")?;
    let started = soda_wire_time::parse_nanos(&observation.started).ok_or_else(refused)?;
    let finished = soda_wire_time::parse_nanos(&observation.finished).ok_or_else(refused)?;
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map_err(|_| refused())?
        .as_nanos() as i128;
    if manifest.architecture != "aarch64"
        || observation.outcome != "completed"
        || observation.execution != "completed"
        || observation.evidence != "completed"
        || observation.exit_code != Some(0)
        || observation.tool_dirty
        || !crate::buildx::is_revision(&observation.tool_revision)
        || observation.requested_revision != manifest.soda_revision
        || observation.requested_architecture != manifest.architecture
        || !super::qualification::control_platform(
            &observation.client_platform,
            &manifest.architecture,
        )
        || !soda_acceptance::report::valid_owner(&observation.owner)
        || observation.target != native.target
        || native.target.is_empty()
        || observation.action != "exec"
        || observation
            .invocation
            .as_ref()
            .is_none_or(|args| args.is_empty() || args[0].is_empty())
        || started >= finished
        || finished > now
        || native.host_platform != "darwin/arm64"
        || native.runtime != "utm"
        || !matches!(native.accelerator.as_str(), "hvf" | "apple-virtualization")
        || native.guest_platform != "linux/arm64"
        || native.virtual_machine_id.trim().is_empty()
        || native.soda_revision != manifest.soda_revision
        || native.iso_sha256 != manifest.asset(&manifest.iso)?.sha256
        || !is_digest(&native.host_readback_sha256)
        || !is_digest(&native.guest_readback_sha256)
        || native.host_readback_sha256 == native.guest_readback_sha256
        || observation
            .files
            .as_ref()
            .and_then(|files| files.get("native.json"))
            != Some(&native_file.sha256)
        || ![&native.host_readback_sha256, &native.guest_readback_sha256]
            .iter()
            .all(|expected| {
                observation.files.as_ref().is_some_and(|files| {
                    files
                        .iter()
                        .any(|(name, hash)| name != "native.json" && hash == *expected)
                })
            })
        || !observation.artifacts.as_ref().is_some_and(|artifacts| {
            artifacts.iter().any(|(name, hash)| {
                Path::new(name).file_name().and_then(|name| name.to_str())
                    == Some(manifest.iso.as_str())
                    && hash == &native.iso_sha256
            })
        })
    {
        return Err(refused());
    }
    if let Some(published) = published_at {
        if started < soda_wire_time::parse_nanos(published).ok_or_else(refused)? {
            return Err(Error::msg(
                "ARM appliance qualification must follow immutable publication",
            ));
        }
    }
    Ok(FileIdentity {
        sha256: hash,
        bytes: std::fs::metadata(path).map_err(|_| refused())?.len(),
    })
}
