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
