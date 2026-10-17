//! `check.go`: delivered-candidate binding check.

use crate::admission::load_admitted_candidate;
use crate::buildx::Root;
use crate::prepare::verify_candidate_images;
use crate::Error;

/// `CheckCandidate`: bind one delivered candidate directory to the requested
/// architecture and revisions, then verify its archives.
pub fn check_candidate(
    candidate: &str,
    arch: &str,
    soda_revision: &str,
    forgejo_revision: &str,
) -> Result<(), Error> {
    let (p, c, _) = load_admitted_candidate(candidate)?;
    if p.architecture != arch {
        return Err(Error::at("candidate architecture"));
    }
    if p.revision != soda_revision {
        return Err(Error::at("candidate soda revision"));
    }
    if c.forgejo_revision != forgejo_revision {
        return Err(Error::at("candidate forgejo revision"));
    }
    let root = Root::open(candidate)?;
    verify_candidate_images(&root, candidate, &p, &c)?;
    if crate::buildx::entry_exists(&root, "docker")? {
        crate::docker::verify(candidate)?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn missing_candidate_dir_errors() {
        assert!(check_candidate("/nonexistent-candidate-xyz", "x86_64", "r", "f").is_err());
    }

    /// Stage one admitted pair as a candidate directory: the shared valid
    /// payload bytes plus its bound candidate document.
    fn stage_pair(tag: &str) -> (String, crate::model::Candidate) {
        let dir = std::env::temp_dir().join(format!("soda-check-{tag}-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        let (bytes, _, c) = crate::model::Candidate::fixture_pair();
        std::fs::write(dir.join("payload.json"), &bytes).unwrap();
        std::fs::write(dir.join("candidate.json"), serde_json::to_vec(&c).unwrap()).unwrap();
        (dir.to_string_lossy().into_owned(), c)
    }

    /// R50 binding: mismatched architecture and revisions refuse before
    /// any archive verification runs.
    #[test]
    fn check_binding_mismatches_refuse() {
        let (dir, c) = stage_pair("bind");
        let result = check_candidate(&dir, "aarch64", &c.host.revision, &c.forgejo_revision);
        assert!(result.is_err());
        let result = check_candidate(&dir, "x86_64", &"f".repeat(40), &c.forgejo_revision);
        assert!(result.is_err());
        let result = check_candidate(&dir, "x86_64", &c.host.revision, &"f".repeat(40));
        assert!(result.is_err());
        let _ = std::fs::remove_dir_all(&dir);
    }
}
