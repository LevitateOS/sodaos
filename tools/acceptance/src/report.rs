//! Observation records and the native support handoff, mirroring
//! `internal/acceptance/report.go`.
//!
//! An [`Observation`] is an ordinary log index, not a
//! scenario/qualification registry. [`handoff`] cites exact observations
//! and explicitly leaves missing scopes open; it can complete without
//! core product evidence and never certifies readiness.

use crate::error::Error;

mod handoff;
mod observation;

pub use handoff::{handoff, hashes, read_observation};
pub use observation::{observation_from_json, observation_json, Observation};

/// Candidate architecture gate: only `x86_64` is admitted, reported as
/// its OCI name. Mirrors `OCIArchitecture`.
pub fn oci_architecture(arch: &str) -> Result<String, Error> {
    soda_build_tools::reader::oci_architecture(arch)
        .map(|name| name.to_string())
        .map_err(|err| Error::msg(err.to_string()))
}

/// Matching-native Linux gate. Mirrors `RequireNative`.
pub fn require_native(arch: &str) -> Result<(), Error> {
    let oci = oci_architecture(arch)?;
    let native = std::env::consts::OS == "linux"
        && match std::env::consts::ARCH {
            "x86_64" => "amd64",
            "x86" => "386",
            "aarch64" => "arm64",
            other => other,
        } == oci;
    if !native {
        return Err(Error::msg("matching-native Linux required"));
    }
    Ok(())
}

/// Lowercase SHA-256 hex shape. Mirrors `Digest`.
pub fn digest(text: &str) -> bool {
    soda_build_tools::reader::is_digest(text)
}

/// Full lowercase source-revision shape. Mirrors `Revision`.
pub fn revision(text: &str) -> bool {
    soda_build_tools::reader::is_revision(text)
}

/// Owner label check. It never grants execution permission.
pub fn valid_owner(owner: &str) -> bool {
    match owner {
        "P02" | "P03" | "P04" | "P05" | "P06" | "P11" => return true,
        _ => {}
    }
    // U01-U20: `^U(?:0[1-9]|1[0-9]|20)$`.
    let bytes = owner.as_bytes();
    if bytes.len() != 3
        || bytes[0] != b'U'
        || !bytes[1].is_ascii_digit()
        || !bytes[2].is_ascii_digit()
    {
        return false;
    }
    matches!(
        (bytes[1] - b'0', bytes[2] - b'0'),
        (0, 1..=9) | (1, 0..=9) | (2, 0)
    )
}

#[cfg(test)]
mod tests;
