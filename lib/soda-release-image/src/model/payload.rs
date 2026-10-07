use serde::ser::SerializeStruct;
use serde::{Serialize, Serializer};

use crate::error::Error;
use crate::jsonio;
use crate::sys;

use super::{
    is_coreos_release, is_digest, is_revision, oci_architecture, prefixed_digest,
    valid_repository_prefix, NAMES,
};

// ---------------------------------------------------------------------------
// deliver.Payload
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct PayloadImage {
    #[serde(rename = "Reference")]
    pub reference: String,
    #[serde(rename = "Config")]
    pub config: String,
    #[serde(rename = "Manifest")]
    pub manifest: String,
    #[serde(rename = "ArchiveSHA256")]
    pub archive_sha256: String,
}

crate::jsonio::case_record!(PayloadImage, {
    reference: String => "Reference",
    config: String => "Config",
    manifest: String => "Manifest",
    archive_sha256: String => "ArchiveSHA256",
});

#[derive(Default)]
struct PayloadWire {
    format: i64,
    id: String,
    revision: String,
    architecture: String,
    core_os: String,
    base: String,
    repository_prefix: String,
    schema: i64,
    presentation_sha256: String,
    host_packages_sha256: String,
    images: crate::jsonio::OrderedMap<PayloadImage>,
    upgrade_from: Vec<String>,
}

crate::jsonio::case_record!(PayloadWire, {
    format: i64 => "Format",
    id: String => "ID",
    revision: String => "Revision",
    architecture: String => "Architecture",
    core_os: String => "CoreOS",
    base: String => "Base",
    repository_prefix: String => "RepositoryPrefix",
    schema: i64 => "Schema",
    presentation_sha256: String => "PresentationSHA256",
    host_packages_sha256: String => "HostPackagesSHA256",
    images: crate::jsonio::OrderedMap<PayloadImage> => "Images",
    upgrade_from: Vec<String> => "UpgradeFrom",
});

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Payload {
    pub format: i64,
    pub id: String,
    pub revision: String,
    pub architecture: String,
    pub core_os: String,
    pub base: String,
    pub repository_prefix: String,
    pub schema: i64,
    pub presentation_sha256: String,
    pub host_packages_sha256: String,
    pub images: Vec<(String, PayloadImage)>,
    pub upgrade_from: Vec<String>,
}

impl Serialize for Payload {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        let mut record = serializer.serialize_struct("Payload", 12)?;
        record.serialize_field("Format", &self.format)?;
        record.serialize_field("ID", &self.id)?;
        record.serialize_field("Revision", &self.revision)?;
        record.serialize_field("Architecture", &self.architecture)?;
        record.serialize_field("CoreOS", &self.core_os)?;
        record.serialize_field("Base", &self.base)?;
        record.serialize_field("RepositoryPrefix", &self.repository_prefix)?;
        record.serialize_field("Schema", &self.schema)?;
        record.serialize_field("PresentationSHA256", &self.presentation_sha256)?;
        record.serialize_field("HostPackagesSHA256", &self.host_packages_sha256)?;
        record.serialize_field("Images", &crate::jsonio::SortedPairs(&self.images))?;
        record.serialize_field("UpgradeFrom", &self.upgrade_from)?;
        record.end()
    }
}

impl Payload {
    pub fn parse(text: &str) -> Result<Payload, Error> {
        let wire: PayloadWire = jsonio::parse(text)?;
        Ok(Payload {
            format: wire.format,
            id: wire.id,
            revision: wire.revision,
            architecture: wire.architecture,
            core_os: wire.core_os,
            base: wire.base,
            repository_prefix: wire.repository_prefix,
            schema: wire.schema,
            presentation_sha256: wire.presentation_sha256,
            host_packages_sha256: wire.host_packages_sha256,
            images: wire.images.0,
            upgrade_from: wire.upgrade_from,
        })
    }

    pub fn image(&self, name: &str) -> PayloadImage {
        self.images
            .iter()
            .find(|(n, _)| n == name)
            .map(|(_, image)| image.clone())
            .unwrap_or_default()
    }

    fn valid_identity(&self) -> bool {
        self.format == 3
            && is_revision(&self.revision)
            && self.id == format!("{}.soda-{}", self.core_os, &self.revision[..12])
            && is_coreos_release(&self.core_os)
    }

    fn valid_base(&self) -> bool {
        match self.base.split_once("/fedora/fedora-coreos@sha256:") {
            Some((host, digest)) => {
                !host.is_empty()
                    && !host.contains('/')
                    && is_digest(digest)
                    && self.schema >= 1
                    && is_digest(&self.presentation_sha256)
                    && is_digest(&self.host_packages_sha256)
            }
            None => false,
        }
    }

    fn valid_images(&self) -> Result<(), Error> {
        if self.images.len() != NAMES.len() {
            return Err(Error::msg("complete image set required"));
        }
        for name in NAMES {
            let image = self.image(name);
            let found = self.images.iter().any(|(n, _)| n == name);
            if !found
                || !prefixed_digest(&image.config)
                || !prefixed_digest(&image.manifest)
                || !is_digest(&image.archive_sha256)
                || image.reference
                    != format!("{}-{name}@{}", self.repository_prefix, image.manifest)
            {
                return Err(Error::msg(format!("invalid {name} image binding")));
            }
        }
        if self.image("extension").config == self.image("forgejo").config {
            return Err(Error::msg(
                "extension requires an independent image identity",
            ));
        }
        Ok(())
    }

    pub fn validate(&self) -> Result<(), Error> {
        if !self.valid_identity() {
            return Err(Error::msg("invalid appliance payload identity"));
        }
        oci_architecture(&self.architecture)?;
        if !valid_repository_prefix(&self.repository_prefix) {
            return Err(Error::msg("explicit GHCR repository prefix required"));
        }
        if !self.valid_base() {
            return Err(Error::msg("incomplete appliance payload"));
        }
        self.valid_images()?;
        if !self.upgrade_from.is_empty() {
            return Err(Error::msg("candidate has no qualified upgrade paths"));
        }
        Ok(())
    }

    /// `deliver.Load`: strict build-JSON read plus validation.
    pub fn load(path: &str) -> Result<Payload, Error> {
        let text = sys::read_json_build_text(path)?;
        let payload = Payload::parse(&text)?;
        payload.validate()?;
        Ok(payload)
    }
}
