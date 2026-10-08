//! Appliance payload metadata consumed from the canonical release-image
//! producer, plus shared-OCI content verification before installer writes.

use std::collections::BTreeMap;

use crate::buildx;
use crate::errors::Error;
use crate::oci;
use serde::Deserialize;

pub const PATH: &str = "/usr/share/soda/release.json";
pub const IMAGES_PATH: &str = "/usr/share/soda/images";

pub const NAMES: [&str; 6] = [
    "dashboard",
    "forgejo",
    "extension",
    "proxy",
    "project-os",
    "tailnet",
];

#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Image {
    #[serde(rename = "Reference")]
    pub reference: String,
    #[serde(rename = "Config")]
    pub config: String,
    #[serde(rename = "Manifest")]
    pub manifest: String,
    #[serde(rename = "ArchiveSHA256")]
    pub archive_sha256: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Payload {
    #[serde(rename = "Format")]
    pub format: i64,
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Revision")]
    pub revision: String,
    #[serde(rename = "Architecture")]
    pub architecture: String,
    #[serde(rename = "CoreOS")]
    pub core_os: String,
    #[serde(rename = "Base")]
    pub base: String,
    #[serde(rename = "RepositoryPrefix")]
    pub repository_prefix: String,
    #[serde(rename = "Schema")]
    pub schema: i64,
    #[serde(rename = "PresentationSHA256")]
    pub presentation_sha256: String,
    #[serde(rename = "HostPackagesSHA256")]
    pub host_packages_sha256: String,
    #[serde(rename = "Images")]
    pub images: BTreeMap<String, Image>,
    #[serde(rename = "UpgradeFrom")]
    pub upgrade_from: Vec<String>,
}

fn core_os_shape(core_os: &str) -> bool {
    // `^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`
    let parts: Vec<&str> = core_os.split('.').collect();
    parts.len() == 4
        && parts
            .iter()
            .all(|p| !p.is_empty() && p.bytes().all(|b| b.is_ascii_digit()))
}

fn valid_repository_prefix(s: &str) -> bool {
    // `^ghcr\.io/[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9._-]*$`, length under 200.
    if s.len() >= 200 {
        return false;
    }
    let rest = match s.strip_prefix("ghcr.io/") {
        Some(rest) => rest,
        None => return false,
    };
    let (owner, repo) = match rest.split_once('/') {
        Some(pair) => pair,
        None => return false,
    };
    if owner.is_empty() || repo.is_empty() || rest.contains("//") {
        // `split_once` already rejects missing separators; a second slash
        // anywhere (including a trailing one) fails the anchored pattern.
        if rest.matches('/').count() != 1 {
            return false;
        }
    }
    let owner_ok = owner
        .bytes()
        .next()
        .map(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        .unwrap_or(false)
        && owner
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-');
    let repo_ok = repo
        .bytes()
        .next()
        .map(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        .unwrap_or(false)
        && repo.bytes().all(|b| {
            b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'.' || b == b'_' || b == b'-'
        });
    owner_ok && repo_ok
}

fn digest_prefixed(s: &str) -> bool {
    match s.strip_prefix("sha256:") {
        Some(hex) => buildx::digest(hex),
        None => false,
    }
}

impl Payload {
    fn valid_identity(&self) -> bool {
        self.format == 3
            && buildx::revision(&self.revision)
            && self.id == format!("{}.soda-{}", self.core_os, &self.revision[..12])
            && core_os_shape(&self.core_os)
    }

    fn valid_base(&self) -> bool {
        match self.base.split_once("/fedora/fedora-coreos@sha256:") {
            Some((host, digest)) => {
                !host.is_empty()
                    && !host.contains('/')
                    && buildx::digest(digest)
                    && self.schema >= 1
                    && buildx::digest(&self.presentation_sha256)
                    && buildx::digest(&self.host_packages_sha256)
            }
            None => false,
        }
    }

    fn valid_images(&self) -> Result<(), Error> {
        if self.images.len() != NAMES.len() {
            return Err(Error::msg("complete image set required"));
        }
        for name in NAMES {
            let valid = match self.images.get(name) {
                Some(image) => {
                    digest_prefixed(&image.config)
                        && digest_prefixed(&image.manifest)
                        && buildx::digest(&image.archive_sha256)
                        && image.reference
                            == format!("{}-{name}@{}", self.repository_prefix, image.manifest)
                }
                None => false,
            };
            if !valid {
                return Err(Error::msg(format!("invalid {name} image binding")));
            }
        }
        if self.images["extension"].config == self.images["forgejo"].config {
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
        buildx::oci_architecture(&self.architecture)?;
        if !valid_repository_prefix(&self.repository_prefix) {
            return Err(Error::msg("explicit GHCR repository prefix required"));
        }
        if !self.valid_base() {
            return Err(Error::msg("incomplete appliance payload"));
        }
        self.valid_images()?;
        // Native admission is not implemented in this milestone. Do not
        // publish a guessed compatibility promise simply because schemas
        // happen to match.
        if !self.upgrade_from.is_empty() {
            return Err(Error::msg("candidate has no qualified upgrade paths"));
        }
        Ok(())
    }
}

pub fn load(path: &str) -> Result<Payload, Error> {
    let data = buildx::read_json_bytes(path)?;
    let text = std::str::from_utf8(&data).map_err(|_| Error::msg("invalid payload document"))?;
    let mut deserializer = serde_json::Deserializer::from_str(text);
    let payload = Payload::deserialize(&mut deserializer)
        .map_err(|_| Error::msg("invalid payload document"))?;
    deserializer
        .end()
        .map_err(|_| Error::msg("invalid payload document"))?;
    payload.validate()?;
    Ok(payload)
}

fn bind_image_revisions(payload: &Payload) -> Result<BTreeMap<String, String>, Error> {
    let mut revisions = BTreeMap::new();
    for name in NAMES {
        let revision = if name == "proxy" {
            String::new()
        } else {
            payload.revision.clone()
        };
        let reference = payload.images[name].config.clone();
        if let Some(previous) = revisions.get(&reference) {
            if *previous != revision {
                return Err(Error::msg("conflicting OCI source bindings"));
            }
        }
        revisions.insert(reference, revision);
    }
    Ok(revisions)
}

fn match_layout_identities(payload: &Payload, layout: &oci::OciLayout) -> Result<(), Error> {
    for name in NAMES {
        let expected = &payload.images[name];
        match layout.images.get(&expected.config) {
            Some(got) if got.config == expected.config && got.manifest == expected.manifest => {}
            _ => return Err(Error::msg(format!("{name} OCI identity mismatch"))),
        }
    }
    Ok(())
}

/// Binds the shared OCI content to the payload before import or disk
/// writes. Each shared blob is hashed and counted once.
pub fn verify_content(
    payload: &Payload,
    images: &str,
) -> Result<(BTreeMap<String, String>, u64), Error> {
    payload.validate()?;
    if !images.starts_with('/') || images.contains([':', '\r', '\n']) {
        return Err(Error::msg("absolute local OCI directory required"));
    }
    let revisions = bind_image_revisions(payload)?;
    let layout = oci::inspect_oci_layout(images, &payload.architecture, &revisions)?;
    match_layout_identities(payload, &layout)?;
    Ok((layout.files, layout.bytes))
}

#[cfg(test)]
mod tests;
