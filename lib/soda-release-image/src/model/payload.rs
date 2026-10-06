use soda_json::JsonValue;

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

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct PayloadImage {
    pub reference: String,
    pub config: String,
    pub manifest: String,
    pub archive_sha256: String,
}

impl PayloadImage {
    pub fn parse(value: &JsonValue) -> Result<PayloadImage, Error> {
        jsonio::check_no_unknown(value, &["Reference", "Config", "Manifest", "ArchiveSHA256"])?;
        Ok(PayloadImage {
            reference: jsonio::require_string(value, "Reference")?,
            config: jsonio::require_string(value, "Config")?,
            manifest: jsonio::require_string(value, "Manifest")?,
            archive_sha256: jsonio::require_string(value, "ArchiveSHA256")?,
        })
    }

    pub fn to_json(&self) -> JsonValue {
        JsonValue::Object(vec![
            (
                "Reference".to_string(),
                JsonValue::Str(self.reference.clone()),
            ),
            ("Config".to_string(), JsonValue::Str(self.config.clone())),
            (
                "Manifest".to_string(),
                JsonValue::Str(self.manifest.clone()),
            ),
            (
                "ArchiveSHA256".to_string(),
                JsonValue::Str(self.archive_sha256.clone()),
            ),
        ])
    }
}

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

impl Payload {
    pub fn parse(value: &JsonValue) -> Result<Payload, Error> {
        jsonio::check_no_unknown(
            value,
            &[
                "Format",
                "ID",
                "Revision",
                "Architecture",
                "CoreOS",
                "Base",
                "RepositoryPrefix",
                "Schema",
                "PresentationSHA256",
                "HostPackagesSHA256",
                "Images",
                "UpgradeFrom",
            ],
        )?;
        let mut images = Vec::new();
        if let JsonValue::Object(entries) = jsonio::require_object(value, "Images")? {
            for (name, image) in entries {
                images.push((name.clone(), PayloadImage::parse(image)?));
            }
        }
        let mut upgrade_from = Vec::new();
        for item in jsonio::require_array(value, "UpgradeFrom")? {
            match item {
                JsonValue::Str(s) => upgrade_from.push(s.clone()),
                _ => return Err(Error::msg("invalid UpgradeFrom")),
            }
        }
        Ok(Payload {
            format: jsonio::require_i64(value, "Format")?,
            id: jsonio::require_string(value, "ID")?,
            revision: jsonio::require_string(value, "Revision")?,
            architecture: jsonio::require_string(value, "Architecture")?,
            core_os: jsonio::require_string(value, "CoreOS")?,
            base: jsonio::require_string(value, "Base")?,
            repository_prefix: jsonio::require_string(value, "RepositoryPrefix")?,
            schema: jsonio::require_i64(value, "Schema")?,
            presentation_sha256: jsonio::require_string(value, "PresentationSHA256")?,
            host_packages_sha256: jsonio::require_string(value, "HostPackagesSHA256")?,
            images,
            upgrade_from,
        })
    }

    pub fn image(&self, name: &str) -> PayloadImage {
        self.images
            .iter()
            .find(|(n, _)| n == name)
            .map(|(_, image)| image.clone())
            .unwrap_or_default()
    }

    pub fn to_json(&self) -> JsonValue {
        let mut images: Vec<(String, PayloadImage)> = self.images.clone();
        images.sort_by(|a, b| a.0.cmp(&b.0));
        JsonValue::Object(vec![
            (
                "Format".to_string(),
                JsonValue::Number(self.format.to_string()),
            ),
            ("ID".to_string(), JsonValue::Str(self.id.clone())),
            (
                "Revision".to_string(),
                JsonValue::Str(self.revision.clone()),
            ),
            (
                "Architecture".to_string(),
                JsonValue::Str(self.architecture.clone()),
            ),
            ("CoreOS".to_string(), JsonValue::Str(self.core_os.clone())),
            ("Base".to_string(), JsonValue::Str(self.base.clone())),
            (
                "RepositoryPrefix".to_string(),
                JsonValue::Str(self.repository_prefix.clone()),
            ),
            (
                "Schema".to_string(),
                JsonValue::Number(self.schema.to_string()),
            ),
            (
                "PresentationSHA256".to_string(),
                JsonValue::Str(self.presentation_sha256.clone()),
            ),
            (
                "HostPackagesSHA256".to_string(),
                JsonValue::Str(self.host_packages_sha256.clone()),
            ),
            (
                "Images".to_string(),
                JsonValue::Object(
                    images
                        .into_iter()
                        .map(|(name, image)| (name, image.to_json()))
                        .collect(),
                ),
            ),
            (
                "UpgradeFrom".to_string(),
                JsonValue::Array(
                    self.upgrade_from
                        .iter()
                        .map(|s| JsonValue::Str(s.clone()))
                        .collect(),
                ),
            ),
        ])
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
        let value = sys::read_json_build(path)?;
        let payload = Payload::parse(&value)?;
        payload.validate()?;
        Ok(payload)
    }
}
