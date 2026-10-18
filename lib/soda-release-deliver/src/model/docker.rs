//! Docker's detached binding to a common candidate and exact shipping bytes.
use serde::{Deserialize, Serialize};

use crate::buildx::{is_digest, is_revision, Image};
use crate::{is_digest_ref, Error};

pub const DOCKER_REPOSITORY: &str = "ghcr.io/levitateos/sodaos";

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Distribution {
    Appliance,
    Docker,
    All,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum BuildPurpose {
    Development,
    Production,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields, rename_all = "PascalCase")]
pub struct FileIdentity {
    #[serde(rename = "SHA256")]
    pub sha256: String,
    pub bytes: u64,
}

impl FileIdentity {
    pub fn validate(&self) -> Result<(), Error> {
        if !is_digest(&self.sha256) || self.bytes == 0 || self.bytes > i64::MAX as u64 {
            return Err(Error::msg("invalid Docker file identity"));
        }
        Ok(())
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields, rename_all = "PascalCase")]
pub struct DockerMetadata {
    pub format: u32,
    pub distribution: Distribution,
    pub purpose: BuildPurpose,
    pub release_tag: String,
    pub architecture: String,
    pub platform: String,
    pub soda_revision: String,
    pub forgejo_revision: String,
    pub soda_source: FileIdentity,
    pub forgejo_source: FileIdentity,
    #[serde(rename = "CandidateSHA256")]
    pub candidate_sha256: String,
    #[serde(rename = "PayloadSHA256")]
    pub payload_sha256: String,
    pub host_manifest: String,
    #[serde(rename = "HostArchiveSHA256")]
    pub host_archive_sha256: String,
    pub fedora_base: String,
    pub recipe: FileIdentity,
    pub image: Image,
    pub image_archive: FileIdentity,
    pub packages: FileIdentity,
    pub compose: FileIdentity,
}

impl DockerMetadata {
    pub fn validate(&self) -> Result<(), Error> {
        self.validate_release_identity()?;
        self.validate_candidate_binding()?;
        self.validate_image_binding()?;
        Ok(())
    }

    fn validate_release_identity(&self) -> Result<(), Error> {
        if self.format != 1
            || self.distribution == Distribution::Appliance
            || self.architecture != "x86_64"
            || self.platform != "linux/amd64"
            || !valid_docker_tag(&self.release_tag)
            || !is_revision(&self.soda_revision)
            || !is_revision(&self.forgejo_revision)
        {
            return Err(Error::msg("invalid Docker release identity"));
        }
        Ok(())
    }

    fn validate_candidate_binding(&self) -> Result<(), Error> {
        for hash in [
            &self.candidate_sha256,
            &self.payload_sha256,
            &self.host_archive_sha256,
        ] {
            if !is_digest(hash) {
                return Err(Error::msg("invalid Docker candidate binding"));
            }
        }
        for file in [
            &self.soda_source,
            &self.forgejo_source,
            &self.recipe,
            &self.image_archive,
            &self.packages,
            &self.compose,
        ] {
            file.validate()?;
        }
        Ok(())
    }

    fn validate_image_binding(&self) -> Result<(), Error> {
        let base_digest = fedora_digest(&self.fedora_base)?;
        if !is_digest_ref(&self.host_manifest)
            || !is_digest_ref(&self.image.manifest)
            || !is_digest_ref(&self.image.config)
            || self.image.architecture != "amd64"
            || self.image.revision != self.soda_revision
            || self.image.source != "https://github.com/LevitateOS/sodaos"
            || self.image.base_name != self.fedora_base
            || self.image.base_digest != base_digest
        {
            return Err(Error::msg("invalid Docker image/base binding"));
        }
        Ok(())
    }

    pub fn require_production(&self) -> Result<(), Error> {
        self.validate()?;
        if self.purpose != BuildPurpose::Production {
            return Err(Error::msg("development Docker output cannot be published"));
        }
        Ok(())
    }

    pub fn reference(&self) -> Result<String, Error> {
        self.validate()?;
        Ok(format!("{DOCKER_REPOSITORY}@{}", self.image.manifest))
    }

    /// Match an independently inspected registry image after upload/download.
    /// The registry transport must obtain this observation anonymously.
    pub fn verify_registry(&self, observed: &Image) -> Result<(), Error> {
        self.require_production()?;
        if observed != &self.image {
            return Err(Error::msg(
                "registry Docker image differs from recorded image",
            ));
        }
        Ok(())
    }
}

pub(crate) fn valid_docker_tag(tag: &str) -> bool {
    crate::publish::valid_tag(tag) && !tag.eq_ignore_ascii_case("latest")
}

pub(crate) fn fedora_digest(reference: &str) -> Result<&str, Error> {
    let digest = reference
        .strip_prefix("registry.fedoraproject.org/fedora@")
        .filter(|digest| is_digest_ref(digest))
        .ok_or_else(|| Error::msg("digest-pinned Fedora Docker base required"))?;
    Ok(digest)
}
