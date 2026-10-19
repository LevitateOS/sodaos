//! Validate Muse artifact integrity and target architecture.

use super::is_digest;

pub struct MuseArtifact {
    pub file: String,
    pub sha256: String,
    pub size: i64,
}

pub fn valid_muse_artifact(arch: &str, artifact: &MuseArtifact) -> bool {
    let expected = match arch {
        "x86_64" => "muse-x86-linux",
        _ => return false,
    };
    artifact.file == expected && is_digest(&artifact.sha256) && artifact.size > 0
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
    fn accepts_verified_artifact() {
        assert!(valid_muse_artifact("x86_64", &artifact()));
    }

    #[test]
    fn rejects_bad_artifacts() {
        let good = artifact();
        let wrong_file = MuseArtifact {
            file: "muse-other".to_string(),
            ..artifact()
        };
        assert!(!valid_muse_artifact("x86_64", &wrong_file));
        let bad_digest = MuseArtifact {
            sha256: "z".repeat(64),
            ..artifact()
        };
        assert!(!valid_muse_artifact("x86_64", &bad_digest));
        let empty = MuseArtifact {
            size: 0,
            ..artifact()
        };
        assert!(!valid_muse_artifact("x86_64", &empty));
        assert!(!valid_muse_artifact("aarch64", &good));
        let empty_file = MuseArtifact {
            file: String::new(),
            ..artifact()
        };
        assert!(!valid_muse_artifact("aarch64", &empty_file));
    }
}
