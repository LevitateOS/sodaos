use serde::Serialize;
use serde_json::value::RawValue;
use std::collections::BTreeMap;

use super::config::{valid_signer, VmConfig};
use crate::coreos;
use crate::error::Error;
use crate::evidence::Evidence;
use crate::files;
use crate::jsonio;
use crate::process::Phase;

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

fn decode_verified_base(value: &RawValue) -> Result<VerifiedBase, Error> {
    let fields = match serde_json::from_str::<BTreeMap<String, Box<RawValue>>>(value.get()) {
        Ok(fields) => fields,
        Err(_) if !value.get().trim_start().starts_with('{') => BTreeMap::new(),
        Err(_) => return Err(Error::msg("invalid base receipt")),
    };
    for key in fields.keys() {
        if !["Path", "SHA256", "Architecture", "Release", "Signer"].contains(&key.as_str()) {
            return Err(Error::msg("unknown JSON field"));
        }
    }
    let string_field = |name: &str| -> Result<String, Error> {
        match fields.get(name) {
            None => Ok(String::new()),
            Some(raw) => serde_json::from_str::<Option<String>>(raw.get())
                .map(|v| v.unwrap_or_default())
                .map_err(|_| Error::msg("invalid JSON string field")),
        }
    };
    Ok(VerifiedBase {
        path: string_field("Path")?,
        sha256: string_field("SHA256")?,
        architecture: string_field("Architecture")?,
        release: string_field("Release")?,
        signer: string_field("Signer")?,
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
    #[derive(Serialize)]
    struct Fixture<'a> {
        #[serde(rename = "Name")]
        name: &'a str,
        #[serde(rename = "Architecture")]
        architecture: &'a str,
        #[serde(rename = "BaseSHA256")]
        base_sha256: &'a str,
        #[serde(rename = "Release")]
        release: &'a str,
        #[serde(rename = "FirmwareSHA256")]
        firmware_sha256: &'a str,
        #[serde(rename = "VariablesSHA256")]
        variables_sha256: &'a str,
        #[serde(rename = "Work")]
        work: &'a str,
    }
    let description = Fixture {
        name: &config.name,
        architecture: &config.architecture,
        base_sha256: &base.sha256,
        release: &base.release,
        firmware_sha256: &firmware_hash,
        variables_sha256: &vars_hash,
        work: &config.work,
    };
    evidence.write_json("fixture.json", &description)
}
