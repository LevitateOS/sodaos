use soda_json::JsonValue;

use super::config::{valid_signer, VmConfig};
use super::Phase;
use crate::coreos;
use crate::error::Error;
use crate::evidence::Evidence;
use crate::files;
use crate::jsonio;

/// Verified base receipt, mirroring Go's `VerifiedBase`.
pub struct VerifiedBase {
    /// Base image path.
    pub path: String,
    /// Expected uncompressed SHA-256.
    pub sha256: String,
    /// Base architecture.
    pub architecture: String,
    /// Base release.
    pub release: String,
    /// Signer fingerprint.
    pub signer: String,
}

fn decode_verified_base(value: &JsonValue) -> Result<VerifiedBase, Error> {
    jsonio::check_no_unknown(
        value,
        &["Path", "SHA256", "Architecture", "Release", "Signer"],
    )?;
    Ok(VerifiedBase {
        path: jsonio::opt_string(value, "Path")?,
        sha256: jsonio::opt_string(value, "SHA256")?,
        architecture: jsonio::opt_string(value, "Architecture")?,
        release: jsonio::opt_string(value, "Release")?,
        signer: jsonio::opt_string(value, "Signer")?,
    })
}

pub(super) fn verify_launch_base_image(
    phase: &Phase,
    config: &VmConfig,
) -> Result<VerifiedBase, Error> {
    let (value, _) = jsonio::read_json_file(&config.base_receipt)?;
    let base = decode_verified_base(&value)?;
    let (release, image) = coreos::resolve_qemu(phase, &config.architecture)?;
    if base.architecture != config.architecture
        || base.release != release
        || base.sha256 != image.uncompressed_sha256
    {
        return Err(Error::msg("base does not match resolved CoreOS input"));
    }
    if !base.path.starts_with('/') || base.path.contains([',', '\n', '\r']) {
        return Err(Error::msg("unsafe base path"));
    }
    let meta = std::fs::symlink_metadata(&base.path).map_err(Error::from)?;
    use std::os::unix::fs::MetadataExt;
    if meta.mode() & 0o222 != 0 {
        return Err(Error::msg(
            "verified base must be read-only; fetch a fresh cache, never chmod a live base",
        ));
    }
    if !valid_signer(&base.signer) {
        return Err(Error::msg("verified base receipt lacks selected signer"));
    }
    match files::hash_file(&base.path) {
        Ok(sum) if sum == base.sha256 => Ok(base),
        Ok(_) => Err(Error::msg("base checksum mismatch")),
        Err(err) => {
            Err(Error::join(vec![Some(err), Some(Error::msg("base checksum mismatch"))]).unwrap())
        }
    }
}

pub(super) fn publish_launch_fixture(
    config: &VmConfig,
    evidence: &Evidence,
    base: &VerifiedBase,
) -> Result<(), Error> {
    let firmware_hash = files::hash_file(&config.firmware)?;
    let vars_hash = files::hash_file(&config.variables)?;
    let description = JsonValue::Object(vec![
        ("Name".to_string(), JsonValue::Str(config.name.clone())),
        (
            "Architecture".to_string(),
            JsonValue::Str(config.architecture.clone()),
        ),
        (
            "BaseSHA256".to_string(),
            JsonValue::Str(base.sha256.clone()),
        ),
        ("Release".to_string(), JsonValue::Str(base.release.clone())),
        ("FirmwareSHA256".to_string(), JsonValue::Str(firmware_hash)),
        ("VariablesSHA256".to_string(), JsonValue::Str(vars_hash)),
        ("Work".to_string(), JsonValue::Str(config.work.clone())),
    ]);
    evidence.write_json("fixture.json", &description)
}
