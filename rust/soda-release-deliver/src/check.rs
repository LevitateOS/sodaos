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
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn missing_candidate_dir_errors() {
        assert!(check_candidate("/nonexistent-candidate-xyz", "x86_64", "r", "f").is_err());
    }
}
