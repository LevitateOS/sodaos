//! Upstream Muse release-pin check (`muse.go validMuseArtifact`).
//!
//! Validates the selected release pin before any bytes are staged: the
//! per-architecture filename, the archive digest and size, and the
//! `X.Y.Z-RW.V` release version.

use super::{is_digest, non_empty_digits};

pub struct MuseArtifact {
    pub file: String,
    pub sha256: String,
    pub size: i64,
}

/// `^\d+\.\d+\.\d+-R\d+\.\d+$` release versions.
fn valid_muse_version(version: &str) -> bool {
    let (numbers, suffix) = match version.split_once("-R") {
        Some(pair) => pair,
        None => return false,
    };
    let mut parts = numbers.split('.');
    match (parts.next(), parts.next(), parts.next(), parts.next()) {
        (Some(a), Some(b), Some(c), None) => {
            if !non_empty_digits(a) || !non_empty_digits(b) || !non_empty_digits(c) {
                return false;
            }
        }
        _ => return false,
    }
    let mut parts = suffix.split('.');
    match (parts.next(), parts.next(), parts.next()) {
        (Some(a), Some(b), None) => non_empty_digits(a) && non_empty_digits(b),
        _ => false,
    }
}

pub fn valid_muse_artifact(version: &str, arch: &str, artifact: &MuseArtifact) -> bool {
    let expected = match arch {
        "x86_64" => "muse-x86-linux",
        _ => "",
    };
    artifact.file == expected
        && is_digest(&artifact.sha256)
        && artifact.size > 0
        && valid_muse_version(version)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn artifact() -> MuseArtifact {
        MuseArtifact {
            file: "muse-x86-linux".to_string(),
            sha256: "c".repeat(64),
            size: 1024,
        }
    }

    #[test]
    fn accepts_pinned_release() {
        assert!(valid_muse_artifact("1.2.3-R4.5", "x86_64", &artifact()));
    }

    #[test]
    fn rejects_bad_pins() {
        let good = artifact();
        let wrong_file = MuseArtifact {
            file: "muse-other".to_string(),
            ..artifact()
        };
        assert!(!valid_muse_artifact("1.2.3-R4.5", "x86_64", &wrong_file));
        let bad_digest = MuseArtifact {
            sha256: "z".repeat(64),
            ..artifact()
        };
        assert!(!valid_muse_artifact("1.2.3-R4.5", "x86_64", &bad_digest));
        let empty = MuseArtifact {
            size: 0,
            ..artifact()
        };
        assert!(!valid_muse_artifact("1.2.3-R4.5", "x86_64", &empty));
        assert!(!valid_muse_artifact("1.2-R4.5", "x86_64", &good));
        assert!(!valid_muse_artifact("1.2.3-4.5", "x86_64", &good));
        assert!(!valid_muse_artifact("1.2.3-R4", "x86_64", &good));
        assert!(!valid_muse_artifact("v1.2.3-R4.5", "x86_64", &good));
        // Unknown architectures expect an empty filename, matching Go's
        // map lookup that yields "" for missing keys.
        assert!(!valid_muse_artifact("1.2.3-R4.5", "aarch64", &good));
        let empty_file = MuseArtifact {
            file: String::new(),
            ..artifact()
        };
        assert!(valid_muse_artifact("1.2.3-R4.5", "aarch64", &empty_file));
    }
}
