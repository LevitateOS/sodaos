//! Release-bound terminal support for the retained Project root.

use serde::{Deserialize, Serialize};

pub const DESTINATION: &str = "/usr/libexec/soda/project-terminal";
pub const PAYLOAD_PATH: &str = "/usr/share/soda/project-support/project-terminal";

/// Prospective sources describe candidate scope; they never qualify an upgrade.
pub fn validate_upgrade_sources(sources: &[String], target: &str) -> Result<(), &'static str> {
    if sources.len() > 1
        || sources.iter().any(|source| {
            let Some((coreos, revision)) = source.split_once(".soda-") else {
                return true;
            };
            let parts: Vec<_> = coreos.split('.').collect();
            source == target
                || parts.len() != 4
                || parts
                    .iter()
                    .any(|part| part.is_empty() || !part.bytes().all(|b| b.is_ascii_digit()))
                || revision.len() != 12
                || !revision
                    .bytes()
                    .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
        })
    {
        return Err("invalid prospective upgrade source");
    }
    Ok(())
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct ProjectSupport {
    #[serde(rename = "SHA256")]
    pub sha256: String,
    pub destination: String,
    pub payload_path: String,
    pub project_image: String,
}

impl ProjectSupport {
    pub fn validate(&self, project_image: &str) -> Result<(), &'static str> {
        if !crate::reader::is_digest(&self.sha256)
            || self.destination != DESTINATION
            || self.payload_path != PAYLOAD_PATH
            || self.project_image != project_image
            || !project_image
                .strip_prefix("sha256:")
                .is_some_and(crate::reader::is_digest)
        {
            return Err("invalid Project support binding");
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn support_refuses_unbound_images_and_arbitrary_install_paths() {
        let image = format!("sha256:{}", "a".repeat(64));
        let support = ProjectSupport {
            sha256: "b".repeat(64),
            destination: DESTINATION.into(),
            payload_path: PAYLOAD_PATH.into(),
            project_image: image.clone(),
        };
        support.validate(&image).unwrap();
        assert!(support
            .validate(&format!("sha256:{}", "c".repeat(64)))
            .is_err());
        assert!(ProjectSupport {
            destination: "/bin/sh".into(),
            ..support.clone()
        }
        .validate(&image)
        .is_err());
        assert!(ProjectSupport {
            payload_path: "../../helper".into(),
            ..support.clone()
        }
        .validate(&image)
        .is_err());
        assert!(ProjectSupport {
            sha256: String::new(),
            ..support
        }
        .validate(&image)
        .is_err());
    }
}
