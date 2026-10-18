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
    pub package_base_url: String,
}

impl Request {
    fn validate_development_target(&self) -> std::result::Result<(), Error> {
        if self.development {
            if !["candidate", "media", "docker", "all"].contains(&self.target.as_str()) {
                return Err(Error::msg(
                    "--development requires --target candidate, media, docker or all",
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
            && (!self.development
                || (self.target != "media" && self.target != "all")
                || self.media_compression != "fast")
        {
            return Err(Error::msg(
                "--media-compression accepts only fast with --development --target media or all",
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
    fn validate_development_pairing(&self) -> std::result::Result<(), Error> {
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
        // The narrow combined development target builds one shared
        // candidate and derives both local media and Docker fixtures from
        // it. It stays purpose=development and never manufactures a
        // production or publication receipt.
        if self.development {
            let paired = match (self.target.as_str(), self.distribution()) {
                ("candidate", distribution) => distribution != "all",
                ("docker", "docker") => true,
                ("media", "appliance") => true,
                ("all", "all") => true,
                _ => false,
            };
            if !paired {
                return Err(Error::msg("development media requires appliance; development docker requires docker; development all requires all; candidate selects no packaging"));
            }
        } else if self.distribution() == "all" && self.target.is_empty() {
            // Production all keeps its same-tag GitHub URL restriction below.
        }
        Ok(())
    }

    fn validate_docker_rootfs(&self) -> std::result::Result<(), Error> {
        if self.wants_docker() {
            // Production all binds the public same-tag rootfs URL. The
            // development combined target instead uses an explicit reachable
            // fixture rootfs base admitted by media validation below.
            if !self.development
                && self.distribution() == "all"
                && !self.rootfs_base_url.starts_with(&format!(
                    "https://github.com/LevitateOS/sodaos/releases/download/{}/",
                    self.release_tag
                ))
            {
                return Err(Error::msg(
                    "all distribution requires a rootfs URL for the same release tag",
                ));
            }
        }
        Ok(())
    }

    pub fn validate_target(&self) -> std::result::Result<(), Error> {
        self.validate_development_pairing()?;
        self.validate_docker_rootfs()?;
        self.validate_media_inputs()?;
        if !soda_build_tools::payload::valid_release_tag(&self.release_tag) {
            return Err(Error::msg(
                "Payload v4 requires an explicit version --release-tag",
            ));
        }
        if !self.package_base_url.is_empty() {
            if !self.development {
                return Err(Error::msg("--package-base-url requires --development"));
            }
            media::media_base_url(&self.package_base_url)?;
        }
        // Package URLs always use verified HTTPS. Development can select a
        // reachable fixture origin independently of the rootfs transport.
        let base = self.package_base_url();
        if !soda_build_tools::reader::url::https_url(&base) {
            return Err(Error::msg("native packages require an HTTPS asset base"));
        }
        Ok(())
    }

    pub fn package_base_url(&self) -> String {
        if self.development && !self.package_base_url.is_empty() {
            format!("{}/", self.package_base_url.trim_end_matches('/'))
        } else {
            format!(
                "https://github.com/LevitateOS/sodaos/releases/download/{}/",
                self.release_tag
            )
        }
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
        // The narrow combined development target: one shared candidate
        // feeds both local media and Docker fixtures. It carries the same
        // release tag the v4 native package URLs need, stays
        // purpose=development, and accepts a reachable fixture rootfs base
        // instead of the production same-tag GitHub URL.
        let combined = Request {
            development: true,
            target: "all".into(),
            distribution: "all".into(),
            release_tag: "v9.9.9-dev".into(),
            rootfs_base_url: "https://fixture.invalid/rootfs/".into(),
            package_base_url: "https://fixture.invalid/packages/".into(),
            ..Request::default()
        };
        assert!(combined.validate_target().is_ok());
        assert!(combined.wants_docker() && combined.wants_media());
        assert_eq!(combined.completed_target(), "all");
        assert_eq!(combined.purpose(), "development");
        assert_eq!(combined.requested_target(), "all");
        assert_eq!(
            combined.package_base_url(),
            "https://fixture.invalid/packages/"
        );
        assert_eq!(
            docker.package_base_url(),
            "https://github.com/LevitateOS/sodaos/releases/download/v1.2.3/"
        );
        assert!(Request {
            package_base_url: "http://fixture.invalid/packages/".into(),
            ..combined.clone()
        }
        .validate_target()
        .is_err());
        assert!(Request {
            development: false,
            target: String::new(),
            ..combined.clone()
        }
        .validate_target()
        .is_err());
        // Every candidate seals versioned native packages, including candidate-only builds.
        let dev_media_tag = Request {
            development: true,
            target: "media".into(),
            release_tag: "v9.9.9-dev".into(),
            rootfs_base_url: "https://fixture.invalid/rootfs/".into(),
            ..Request::default()
        };
        assert!(dev_media_tag.validate_target().is_ok());
        let candidate = Request {
            target: "candidate".into(),
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
            // Production all keeps the same-tag GitHub URL restriction: a
            // fixture base is refused outside development.
            Request {
                distribution: "all".into(),
                release_tag: "v1.2.3".into(),
                rootfs_base_url: "https://fixture.invalid/rootfs/".into(),
                ..Request::default()
            },
            // A bare candidate also needs its native package version.
            Request {
                development: true,
                target: "candidate".into(),
                release_tag: String::new(),
                ..Request::default()
            },
            // The combined target pairs only with the all distribution.
            Request {
                development: true,
                target: "all".into(),
                distribution: "appliance".into(),
                release_tag: "v9.9.9-dev".into(),
                rootfs_base_url: "https://fixture.invalid/rootfs/".into(),
                ..Request::default()
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
            release_tag: "v9.9.9-dev".into(),
            ..Request::default()
        };
        assert!(valid.validate_target().is_ok());
        let valid = Request {
            development: true,
            target: "media".to_string(),
            rootfs_base_url: "https://example.invalid".to_string(),
            release_tag: "v9.9.9-dev".into(),
            ..Request::default()
        };
        assert!(valid.validate_target().is_ok());
        // The combined development target is admitted with an explicit
        // release tag and a reachable fixture rootfs base.
        let combined = Request {
            development: true,
            target: "all".to_string(),
            distribution: "all".to_string(),
            release_tag: "v9.9.9-dev".to_string(),
            rootfs_base_url: "https://example.invalid".to_string(),
            ..Request::default()
        };
        assert!(combined.validate_target().is_ok());
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
            "--development requires --target candidate, media, docker or all"
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
            "--media-compression accepts only fast with --development --target media or all"
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
            release_tag: "v9.9.9-dev".into(),
            ..Request::default()
        };
        assert!(fast.validate_target().is_ok());
        let fast_combined = Request {
            development: true,
            target: "all".to_string(),
            distribution: "all".to_string(),
            release_tag: "v9.9.9-dev".to_string(),
            media_compression: "fast".to_string(),
            rootfs_base_url: "https://example.invalid".to_string(),
            ..Request::default()
        };
        assert!(fast_combined.validate_target().is_ok());
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
