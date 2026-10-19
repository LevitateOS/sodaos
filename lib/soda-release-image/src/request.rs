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
