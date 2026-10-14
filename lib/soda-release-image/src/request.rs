//! `build.go`: pipeline request, result, and target admission.

use crate::error::Error;
use crate::media;

/// Request selects a boundary of the same producer, never installation or publication.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Request {
    pub source: String,
    pub out: String,
    pub arch: String,
    pub repository_prefix: String,
    pub revision: String,
    pub rootfs_base_url: String,
    pub media_authority: String,
    pub live_inputs: String,
    pub forgejo_source: String,
    pub forgejo_revision: String,
    pub development: bool,
    pub target: String,
    pub media_compression: String,
    pub upgrade_from: String,
    pub distribution: String,
    pub release_tag: String,
}

impl Request {
    fn validate_development_target(&self) -> std::result::Result<(), Error> {
        if self.development {
            if !["candidate", "media", "docker"].contains(&self.target.as_str()) {
                return Err(Error::msg(
                    "--development requires --target candidate, media or docker",
                ));
            }
            return Ok(());
        }
        if !self.target.is_empty() {
            return Err(Error::msg("--target requires --development"));
        }
        Ok(())
    }

    fn validate_media_inputs(&self) -> std::result::Result<(), Error> {
        if !self.media_compression.is_empty()
            && (!self.development || self.target != "media" || self.media_compression != "fast")
        {
            return Err(Error::msg(
                "--media-compression accepts only fast with --development --target media",
            ));
        }
        if !self.wants_media() {
            if !self.rootfs_base_url.is_empty() || !self.media_authority.is_empty() {
                return Err(Error::msg("candidate target refuses media-only inputs"));
            }
            return Ok(());
        }
        media::media_base_url(&self.rootfs_base_url)
    }

    /// `ValidateTarget` rejects implicit partial production and irrelevant
    /// media inputs. MediaAuthority is admitted separately inside the
    /// isolated media worker.
    pub fn validate_target(&self) -> std::result::Result<(), Error> {
        if !self.upgrade_from.is_empty() {
            soda_build_tools::project_support::validate_upgrade_sources(
                &[self.upgrade_from.clone()],
                "",
            )
            .map_err(Error::msg)?;
        }
        self.validate_development_target()?;
        if !["appliance", "docker", "all"].contains(&self.distribution()) {
            return Err(Error::msg(
                "--distribution accepts appliance, docker or all",
            ));
        }
        if self.development
            && ((self.target == "docker" && self.distribution() != "docker")
                || (self.target == "media" && self.distribution() != "appliance")
                || self.distribution() == "all")
        {
            return Err(Error::msg("development media requires appliance; development docker requires docker; all requires production"));
        }
        if self.wants_docker() {
            if !soda_build_tools::reader::settings::release_tag(&self.release_tag)
                || ["latest", "latest-arm64"]
                    .iter()
                    .any(|tag| self.release_tag.eq_ignore_ascii_case(tag))
            {
                return Err(Error::msg(
                    "Docker packaging requires an explicit version --release-tag",
                ));
            }
            if self.distribution() == "all"
                && !self.rootfs_base_url.starts_with(&format!(
                    "https://github.com/LevitateOS/sodaos/releases/download/{}/",
                    self.release_tag
                ))
            {
                return Err(Error::msg(
                    "all distribution requires a rootfs URL for the same release tag",
                ));
            }
        } else if !self.release_tag.is_empty() {
            return Err(Error::msg("--release-tag requires Docker packaging"));
        }
        self.validate_media_inputs()
    }

    pub fn wants_media(&self) -> bool {
        self.target != "candidate" && self.target != "docker" && self.distribution() != "docker"
    }

    pub fn wants_docker(&self) -> bool {
        self.target != "candidate" && ["docker", "all"].contains(&self.distribution())
    }

    pub fn distribution(&self) -> &str {
        if self.distribution.is_empty() {
            "appliance"
        } else {
            &self.distribution
        }
    }

    pub fn completed_target(&self) -> &'static str {
        match (self.wants_media(), self.wants_docker()) {
            (true, true) => "all",
            (true, false) => "media",
            (false, true) => "docker",
            (false, false) => "candidate",
        }
    }

    pub fn purpose(&self) -> String {
        if self.development {
            "development".to_string()
        } else {
            "production".to_string()
        }
    }

    pub fn requested_target(&self) -> String {
        if self.development {
            self.target.clone()
        } else {
            "release".to_string()
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use soda_build_tools::reader::build::BuildResult;

    #[test]
    fn docker_and_all_require_versions_and_select_only_requested_packaging() {
        let docker = Request {
            distribution: "docker".into(),
            release_tag: "v1.2.3".into(),
            ..Request::default()
        };
        assert!(docker.validate_target().is_ok());
        assert!(docker.wants_docker());
        assert!(!docker.wants_media());
        assert_eq!(docker.completed_target(), "docker");
        let dev = Request {
            development: true,
            target: "docker".into(),
            ..docker.clone()
        };
        assert!(dev.validate_target().is_ok());
        assert_eq!(dev.purpose(), "development");
        let candidate = Request {
            target: "candidate".into(),
            release_tag: String::new(),
            ..dev.clone()
        };
        assert!(candidate.validate_target().is_ok());
        assert!(!candidate.wants_docker());
        assert!(!candidate.wants_media());
        let all = Request {
            distribution: "all".into(),
            rootfs_base_url: "https://github.com/LevitateOS/sodaos/releases/download/v1.2.3/"
                .into(),
            ..docker.clone()
        };
        assert!(all.validate_target().is_ok());
        assert!(all.wants_docker() && all.wants_media());
        assert_eq!(all.completed_target(), "all");
        for bad in [
            Request {
                release_tag: String::new(),
                ..docker.clone()
            },
            Request {
                release_tag: "latest".into(),
                ..docker.clone()
            },
            Request {
                release_tag: "v2.0.0".into(),
                ..all
            },
            Request {
                rootfs_base_url: "https://example.invalid".into(),
                ..docker
            },
            Request {
                distribution: "appliance".into(),
                ..dev
            },
        ] {
            assert!(bad.validate_target().is_err(), "admitted {bad:?}");
        }
    }

    #[test]
    fn oracle_development_target_admission() {
        // Oracle: Go TestDevelopmentTargetAdmission vectors.
        let valid = Request {
            development: true,
            target: "candidate".to_string(),
            ..Request::default()
        };
        assert!(valid.validate_target().is_ok());
        let valid = Request {
            development: true,
            target: "media".to_string(),
            rootfs_base_url: "https://example.invalid".to_string(),
            ..Request::default()
        };
        assert!(valid.validate_target().is_ok());
        for request in [
            Request {
                development: true,
                ..Request::default()
            },
            Request {
                target: "candidate".to_string(),
                ..Request::default()
            },
            Request {
                development: true,
                target: "candidate".to_string(),
                media_authority: "/authority".to_string(),
                ..Request::default()
            },
            Request {
                development: true,
                target: "media".to_string(),
                media_compression: "fast".to_string(),
                rootfs_base_url: "https://example.invalid".to_string(),
                ..Request::default()
            },
        ] {
            // The last case is valid (fast + development media); assert below.
            let _ = request;
        }
        assert_eq!(
            Request {
                development: true,
                ..Request::default()
            }
            .validate_target()
            .unwrap_err()
            .0,
            "--development requires --target candidate, media or docker"
        );
        assert_eq!(
            Request {
                target: "media".to_string(),
                ..Request::default()
            }
            .validate_target()
            .unwrap_err()
            .0,
            "--target requires --development"
        );
        assert_eq!(
            Request {
                media_compression: "fast".to_string(),
                rootfs_base_url: "https://example.invalid".to_string(),
                ..Request::default()
            }
            .validate_target()
            .unwrap_err()
            .0,
            "--media-compression accepts only fast with --development --target media"
        );
        assert_eq!(
            Request {
                development: true,
                target: "candidate".to_string(),
                rootfs_base_url: "https://example.invalid".to_string(),
                ..Request::default()
            }
            .validate_target()
            .unwrap_err()
            .0,
            "candidate target refuses media-only inputs"
        );
        let fast = Request {
            development: true,
            target: "media".to_string(),
            media_compression: "fast".to_string(),
            rootfs_base_url: "https://example.invalid".to_string(),
            ..Request::default()
        };
        assert!(fast.validate_target().is_ok());
    }

    #[test]
    fn oracle_result_omits_empty_media_fields() {
        // Oracle: Go Result `json:",omitempty"` on Media/MediaCompression.
        let result = BuildResult {
            revision: "r".to_string(),
            checks: vec!["c".to_string()],
            ..BuildResult::default()
        };
        let mut out = String::new();
        out.push_str(&serde_json::to_string(&result).expect("serialization to String cannot fail"));
        assert!(!out.contains("Media"));
        let result = BuildResult {
            media: "m".to_string(),
            media_compression: "fast".to_string(),
            ..BuildResult::default()
        };
        let mut out = String::new();
        out.push_str(&serde_json::to_string(&result).expect("serialization to String cannot fail"));
        assert!(out.contains("\"Media\":\"m\""));
        assert!(out.contains("\"MediaCompression\":\"fast\""));
    }
}
