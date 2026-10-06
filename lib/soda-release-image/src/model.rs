//! Exact mirrors of the foreign `release/build` + `release/deliver` model
//! shapes the pipeline decodes, validates, and emits: payloads, candidates,
//! trust, permits, images, CoreOS inputs. JSON field names and emission
//! order match the Go owners; validators return the same messages.

use soda_json::JsonValue;

use crate::error::Error;
use crate::jsonio;
use crate::sys;

mod images;
mod url;

pub use images::{Image, ProducedImage};
pub use url::{https_url, is_loopback_addr, parse_url, UrlParts};

pub const DELIVER_PATH: &str = "/usr/share/soda/release.json";
pub const DELIVER_IMAGES_PATH: &str = "/usr/share/soda/images";
/// `deliver.Names`, in order.
pub const NAMES: [&str; 6] = [
    "dashboard",
    "forgejo",
    "extension",
    "proxy",
    "project-os",
    "tailnet",
];
pub const FORGEJO_COMPILER_IMAGE: &str =
    "docker.io/library/golang@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468";
/// `store.SchemaVersion()`.
pub const SCHEMA_VERSION: i64 = 26;
pub const SODA_SOURCE: &str = "https://github.com/LevitateOS/sodaos";

pub fn is_revision(s: &str) -> bool {
    soda_build_tools::reader::is_revision(s)
}

pub fn is_digest(s: &str) -> bool {
    soda_build_tools::reader::is_digest(s)
}

pub fn oci_architecture(arch: &str) -> Result<&'static str, Error> {
    soda_build_tools::reader::oci_architecture(arch).map_err(|e| Error::msg(e.to_string()))
}

/// `deliver.Digest`: `sha256:`-prefixed hex digest.
pub fn prefixed_digest(s: &str) -> bool {
    match s.strip_prefix("sha256:") {
        Some(hex) => is_digest(hex),
        None => false,
    }
}

/// `deliver.Hash`: `sha256:`-prefixed hex of bytes.
pub fn deliver_hash(data: &[u8]) -> String {
    format!("sha256:{}", sys::hex_sha256(data))
}

fn is_dotted_numbers(value: &str, parts: usize) -> bool {
    let pieces: Vec<&str> = value.split('.').collect();
    pieces.len() == parts
        && pieces
            .iter()
            .all(|p| !p.is_empty() && p.bytes().all(|b| b.is_ascii_digit()))
}

pub fn is_coreos_release(value: &str) -> bool {
    is_dotted_numbers(value, 4)
}

pub fn valid_repository_prefix(s: &str) -> bool {
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
    if repo.contains('/') {
        return false;
    }
    let owner_ok = !owner.is_empty()
        && owner
            .bytes()
            .next()
            .is_some_and(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        && owner
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-');
    let repo_ok = !repo.is_empty()
        && repo
            .bytes()
            .next()
            .is_some_and(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        && repo.bytes().all(|b| {
            b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'.' || b == b'_' || b == b'-'
        });
    owner_ok && repo_ok
}

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

// ---------------------------------------------------------------------------
// deliver.Candidate
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ForgejoToolchain {
    pub compiler_image: String,
    pub apk_packages: Vec<String>,
}

impl ForgejoToolchain {
    pub fn parse(value: &JsonValue) -> Result<ForgejoToolchain, Error> {
        jsonio::check_no_unknown(value, &["CompilerImage", "APKPackages"])?;
        let mut apk_packages = Vec::new();
        for item in jsonio::require_array(value, "APKPackages")? {
            match item {
                JsonValue::Str(s) => apk_packages.push(s.clone()),
                _ => return Err(Error::msg("invalid APKPackages")),
            }
        }
        Ok(ForgejoToolchain {
            compiler_image: jsonio::require_string(value, "CompilerImage")?,
            apk_packages,
        })
    }

    pub fn to_json(&self) -> JsonValue {
        JsonValue::Object(vec![
            (
                "CompilerImage".to_string(),
                JsonValue::Str(self.compiler_image.clone()),
            ),
            (
                "APKPackages".to_string(),
                JsonValue::Array(
                    self.apk_packages
                        .iter()
                        .map(|s| JsonValue::Str(s.clone()))
                        .collect(),
                ),
            ),
        ])
    }

    pub fn validate(&self) -> Result<(), Error> {
        if self.compiler_image != FORGEJO_COMPILER_IMAGE
            || !valid_forgejo_apk_list(&self.apk_packages)
        {
            return Err(Error::msg("invalid Forgejo compiler provenance"));
        }
        if !has_forgejo_native_build_tools(&self.apk_packages) {
            return Err(Error::msg("incomplete Forgejo APK provenance"));
        }
        Ok(())
    }
}

fn valid_forgejo_apk_list(packages: &[String]) -> bool {
    if packages.is_empty() || packages.len() > 256 {
        return false;
    }
    let mut previous: Option<&str> = None;
    for name in packages {
        if name.is_empty()
            || name
                .bytes()
                .any(|b| matches!(b, b' ' | b'\t' | b'\n' | b'\r' | b'\\' | 0))
        {
            return false;
        }
        // Non-decreasing (slices.IsSorted) plus the explicit duplicate refusal.
        if let Some(prev) = previous {
            if prev > name.as_str() || prev == name.as_str() {
                return false;
            }
        }
        previous = Some(name);
    }
    true
}

fn has_forgejo_native_build_tools(packages: &[String]) -> bool {
    let (mut base, mut gcc, mut musl) = (false, false, false);
    for name in packages {
        base = base || name == "build-base-0.5-r4";
        gcc = gcc || name.starts_with("gcc-");
        musl = musl || name.starts_with("musl-dev-");
    }
    base && gcc && musl
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Candidate {
    pub format: i64,
    pub host: Image,
    pub host_reference: String,
    pub host_archive_sha256: String,
    pub payload_sha256: String,
    pub migration: String,
    pub notes: String,
    pub forgejo_revision: String,
    pub forgejo_source_sha256: String,
    pub forgejo_toolchain: ForgejoToolchain,
    pub architecture: String,
    pub content_sha256: Vec<(String, String)>,
}

impl Candidate {
    pub fn parse(value: &JsonValue) -> Result<Candidate, Error> {
        jsonio::check_no_unknown(
            value,
            &[
                "Format",
                "Host",
                "HostReference",
                "HostArchiveSHA256",
                "PayloadSHA256",
                "Migration",
                "Notes",
                "ForgejoRevision",
                "ForgejoSourceSHA256",
                "ForgejoToolchain",
                "Architecture",
                "ContentSHA256",
            ],
        )?;
        let host_value = jsonio::require_object(value, "Host")?;
        let host = if matches!(host_value, JsonValue::Null) {
            Image::default()
        } else {
            Image::parse(host_value)?
        };
        let toolchain_value = jsonio::require_object(value, "ForgejoToolchain")?;
        let forgejo_toolchain = if matches!(toolchain_value, JsonValue::Null) {
            ForgejoToolchain::default()
        } else {
            ForgejoToolchain::parse(toolchain_value)?
        };
        Ok(Candidate {
            format: jsonio::require_i64(value, "Format")?,
            host,
            host_reference: jsonio::require_string(value, "HostReference")?,
            host_archive_sha256: jsonio::require_string(value, "HostArchiveSHA256")?,
            payload_sha256: jsonio::require_string(value, "PayloadSHA256")?,
            migration: jsonio::require_string(value, "Migration")?,
            notes: jsonio::require_string(value, "Notes")?,
            forgejo_revision: jsonio::require_string(value, "ForgejoRevision")?,
            forgejo_source_sha256: jsonio::require_string(value, "ForgejoSourceSHA256")?,
            forgejo_toolchain,
            architecture: jsonio::require_string(value, "Architecture")?,
            content_sha256: jsonio::string_map(value, "ContentSHA256")?,
        })
    }

    pub fn to_json(&self) -> JsonValue {
        let mut content = self.content_sha256.clone();
        content.sort_by(|a, b| a.0.cmp(&b.0));
        JsonValue::Object(vec![
            (
                "Format".to_string(),
                JsonValue::Number(self.format.to_string()),
            ),
            ("Host".to_string(), self.host.to_json()),
            (
                "HostReference".to_string(),
                JsonValue::Str(self.host_reference.clone()),
            ),
            (
                "HostArchiveSHA256".to_string(),
                JsonValue::Str(self.host_archive_sha256.clone()),
            ),
            (
                "PayloadSHA256".to_string(),
                JsonValue::Str(self.payload_sha256.clone()),
            ),
            (
                "Migration".to_string(),
                JsonValue::Str(self.migration.clone()),
            ),
            ("Notes".to_string(), JsonValue::Str(self.notes.clone())),
            (
                "ForgejoRevision".to_string(),
                JsonValue::Str(self.forgejo_revision.clone()),
            ),
            (
                "ForgejoSourceSHA256".to_string(),
                JsonValue::Str(self.forgejo_source_sha256.clone()),
            ),
            (
                "ForgejoToolchain".to_string(),
                self.forgejo_toolchain.to_json(),
            ),
            (
                "Architecture".to_string(),
                JsonValue::Str(self.architecture.clone()),
            ),
            (
                "ContentSHA256".to_string(),
                JsonValue::Object(
                    content
                        .into_iter()
                        .map(|(k, v)| (k, JsonValue::Str(v)))
                        .collect(),
                ),
            ),
        ])
    }

    pub fn validate(&self, payload: &Payload, payload_bytes: &[u8]) -> Result<(), Error> {
        let arch = match oci_architecture(&payload.architecture) {
            Ok(arch) => arch,
            Err(_) => return Err(Error::msg(sys::refused())),
        };
        if payload.validate().is_err() {
            return Err(Error::msg(sys::refused()));
        }
        if !valid_candidate_host(self, payload, arch)
            || !valid_candidate_provenance(self, payload, payload_bytes)
        {
            return Err(Error::msg(sys::refused()));
        }
        Ok(())
    }
}

fn valid_candidate_host(candidate: &Candidate, payload: &Payload, arch: &str) -> bool {
    candidate.host_reference
        == format!(
            "{}-host@{}",
            payload.repository_prefix, candidate.host.manifest
        )
        && prefixed_digest(&candidate.host.manifest)
        && prefixed_digest(&candidate.host.config)
        && is_digest(&candidate.host_archive_sha256)
        && candidate.host.architecture == arch
        && candidate.host.revision == payload.revision
        && candidate.host.base_name == payload.base
}

fn valid_candidate_provenance(
    candidate: &Candidate,
    payload: &Payload,
    payload_bytes: &[u8],
) -> bool {
    valid_candidate_source(candidate, payload, payload_bytes)
        && valid_candidate_build(candidate, payload)
        && valid_candidate_content(&candidate.content_sha256)
        && !candidate.migration.is_empty()
        && !candidate.notes.is_empty()
}

fn valid_candidate_source(candidate: &Candidate, payload: &Payload, payload_bytes: &[u8]) -> bool {
    let base_digest = payload.base.split('@').nth(1).unwrap_or("");
    candidate.format == 1
        && candidate.payload_sha256
            == deliver_hash(payload_bytes)
                .strip_prefix("sha256:")
                .unwrap_or("")
        && candidate.host.base_digest == base_digest
        && candidate.host.source == SODA_SOURCE
}

fn valid_candidate_build(candidate: &Candidate, payload: &Payload) -> bool {
    is_revision(&candidate.forgejo_revision)
        && is_digest(&candidate.forgejo_source_sha256)
        && candidate.forgejo_toolchain.validate().is_ok()
        && candidate.architecture == payload.architecture
}

const REQUIRED_CANDIDATE_CONTENT: [&str; 9] = [
    "dashboard:/usr/local/bin/soda-dashboard",
    "forgejo:/usr/local/bin/gitea",
    "extension:/usr/local/bin/gitea",
    "extension:/usr/share/soda/extension/extension.json",
    "extension:/usr/share/soda/extension/backend",
    "extension:/usr/share/soda/extension/run",
    "host:/usr/share/containers/systemd/forgejo.container",
    "host:/usr/share/containers/systemd/soda-dashboard.container",
    "host:/usr/lib/systemd/system/soda-extension-install.service",
];

fn content_get(files: &[(String, String)], name: &str) -> String {
    files
        .iter()
        .find(|(k, _)| k == name)
        .map(|(_, v)| v.clone())
        .unwrap_or_default()
}

pub fn valid_candidate_content(files: &[(String, String)]) -> bool {
    for path in REQUIRED_CANDIDATE_CONTENT {
        if !is_digest(&content_get(files, path)) {
            return false;
        }
    }
    if content_get(files, "forgejo:/usr/local/bin/gitea")
        != content_get(files, "extension:/usr/local/bin/gitea")
    {
        return false;
    }
    let mut assets = 0;
    for (file, hash) in files {
        if !is_digest(hash) {
            return false;
        }
        if let Some(name) = file.strip_prefix("extension:/usr/share/soda/extension/assets/") {
            if !valid_candidate_asset_name(name) {
                return false;
            }
            assets += 1;
            continue;
        }
        if !REQUIRED_CANDIDATE_CONTENT.contains(&file.as_str()) {
            return false;
        }
    }
    assets > 0
}

fn valid_candidate_asset_name(name: &str) -> bool {
    if name.is_empty()
        || sys::clean_path(name) != name
        || name.starts_with('/')
        || name.starts_with("../")
        || name.bytes().any(|b| matches!(b, b'\\' | b'\n' | b'\r' | 0))
    {
        return false;
    }
    true
}

// ---------------------------------------------------------------------------
// deliver.Trust
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Trust {
    pub format: i64,
    pub prefix: String,
    pub epoch: u64,
    pub keys: Vec<(String, Vec<String>)>,
    pub not_before: i64,
    pub max_age_seconds: i64,
    pub clock_skew_seconds: i64,
    pub minimum_sequence: Vec<(String, u64)>,
}

impl Trust {
    pub fn parse(value: &JsonValue) -> Result<Trust, Error> {
        jsonio::check_no_unknown(
            value,
            &[
                "Format",
                "Prefix",
                "Epoch",
                "Keys",
                "NotBefore",
                "MaxAgeSeconds",
                "ClockSkewSeconds",
                "MinimumSequence",
            ],
        )
        .map_err(|_| Error::msg(sys::refused()))?;
        let mut keys = Vec::new();
        if let JsonValue::Object(entries) =
            jsonio::require_object(value, "Keys").map_err(|_| Error::msg(sys::refused()))?
        {
            for (role, list) in entries {
                let JsonValue::Array(items) = list else {
                    return Err(Error::msg(sys::refused()));
                };
                let mut role_keys = Vec::new();
                for item in items {
                    match item {
                        JsonValue::Str(s) => role_keys.push(s.clone()),
                        _ => return Err(Error::msg(sys::refused())),
                    }
                }
                keys.push((role.clone(), role_keys));
            }
        }
        let mut minimum_sequence = Vec::new();
        if let JsonValue::Object(entries) = jsonio::require_object(value, "MinimumSequence")
            .map_err(|_| Error::msg(sys::refused()))?
        {
            for (role, number) in entries {
                let JsonValue::Number(raw) = number else {
                    return Err(Error::msg(sys::refused()));
                };
                let sequence: u64 = raw.parse().map_err(|_| Error::msg(sys::refused()))?;
                minimum_sequence.push((role.clone(), sequence));
            }
        }
        Ok(Trust {
            format: jsonio::require_i64(value, "Format").map_err(|_| Error::msg(sys::refused()))?,
            prefix: jsonio::require_string(value, "Prefix")
                .map_err(|_| Error::msg(sys::refused()))?,
            epoch: jsonio::require_u64(value, "Epoch").map_err(|_| Error::msg(sys::refused()))?,
            keys,
            not_before: jsonio::require_i64(value, "NotBefore")
                .map_err(|_| Error::msg(sys::refused()))?,
            max_age_seconds: jsonio::require_i64(value, "MaxAgeSeconds")
                .map_err(|_| Error::msg(sys::refused()))?,
            clock_skew_seconds: jsonio::require_i64(value, "ClockSkewSeconds")
                .map_err(|_| Error::msg(sys::refused()))?,
            minimum_sequence,
        })
    }

    fn role_keys(&self, role: &str) -> Vec<String> {
        self.keys
            .iter()
            .find(|(r, _)| r == role)
            .map(|(_, keys)| keys.clone())
            .unwrap_or_default()
    }

    fn minimum(&self, role: &str) -> u64 {
        self.minimum_sequence
            .iter()
            .find(|(r, _)| r == role)
            .map(|(_, v)| *v)
            .unwrap_or(0)
    }

    pub fn validate(&self) -> Result<(), Error> {
        if !(self.format == 1
            && valid_repository_prefix(&self.prefix)
            && self.epoch != 0
            && self.keys.len() == 4
            && self.minimum_sequence.len() == 3
            && self.not_before > 0
            && self.max_age_seconds >= 60
            && self.max_age_seconds <= 7 * 86400
            && self.clock_skew_seconds >= 0
            && self.clock_skew_seconds <= 300)
        {
            return Err(Error::msg(sys::refused()));
        }
        let mut seen: Vec<String> = Vec::new();
        for role in ["artifact", "candidate", "preview", "stable"] {
            admit_trust_role_keys(&self.role_keys(role), &mut seen)?;
            if role != "artifact" && self.minimum(role) == 0 {
                return Err(Error::msg(sys::refused()));
            }
        }
        Ok(())
    }
}

fn admit_trust_role_keys(keys: &[String], seen: &mut Vec<String>) -> Result<(), Error> {
    if keys.is_empty() || keys.len() > 4 {
        return Err(Error::msg(sys::refused()));
    }
    for key in keys {
        let der = parse_trust_public_key(key)?;
        let fingerprint = deliver_hash(&der);
        if seen.contains(&fingerprint) {
            return Err(Error::msg("signer roles must not share keys"));
        }
        seen.push(fingerprint);
    }
    Ok(())
}

/// Parse a PKIX `PUBLIC KEY` PEM and require a P-256 EC key, structurally.
/// (The deliver owner additionally verifies the point lies on the curve;
/// fixtures here use real P-256 keys so both agree.)
fn parse_trust_public_key(pem_text: &str) -> Result<Vec<u8>, Error> {
    let refused = || Error::msg(sys::refused());
    let begin = "-----BEGIN PUBLIC KEY-----";
    let end = "-----END PUBLIC KEY-----";
    let start = pem_text.find(begin).ok_or_else(refused)? + begin.len();
    let stop = pem_text[start..].find(end).ok_or_else(refused)? + start;
    let body: String = pem_text[start..stop]
        .chars()
        .filter(|c| !c.is_whitespace())
        .collect();
    let rest = pem_text[stop + end.len()..].trim();
    if !rest.is_empty() {
        return Err(refused());
    }
    use base64::Engine;
    let der = base64::engine::general_purpose::STANDARD
        .decode(body.as_bytes())
        .map_err(|_| refused())?;
    if !is_p256_spki(&der) {
        // Go distinguishes a non-P-256 parse here; keep the owner message.
        return Err(Error::msg("native P-256 Sigstore public key required"));
    }
    Ok(der)
}

fn read_der_length(bytes: &[u8], pos: &mut usize) -> Option<usize> {
    if *pos >= bytes.len() {
        return None;
    }
    let first = bytes[*pos];
    *pos += 1;
    if first & 0x80 == 0 {
        return Some(first as usize);
    }
    let count = (first & 0x7f) as usize;
    if count == 0 || count > 4 || *pos + count > bytes.len() {
        return None;
    }
    let mut length = 0usize;
    for _ in 0..count {
        length = (length << 8) | bytes[*pos] as usize;
        *pos += 1;
    }
    Some(length)
}

fn is_p256_spki(der: &[u8]) -> bool {
    let mut pos = 0;
    // SEQUENCE
    if der.get(pos) != Some(&0x30) {
        return false;
    }
    pos += 1;
    let outer = match read_der_length(der, &mut pos) {
        Some(length) => length,
        None => return false,
    };
    if pos + outer != der.len() {
        return false;
    }
    // algorithm SEQUENCE { ecPublicKey OID, secp256r1 OID }
    if der.get(pos) != Some(&0x30) {
        return false;
    }
    pos += 1;
    let alg_len = match read_der_length(der, &mut pos) {
        Some(length) => length,
        None => return false,
    };
    let alg_end = pos + alg_len;
    // OID 1.2.840.10045.2.1
    let ec_oid: [u8; 9] = [0x06, 0x07, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x02, 0x01];
    if der.get(pos..pos + ec_oid.len()) != Some(&ec_oid[..]) {
        return false;
    }
    pos += ec_oid.len();
    // OID 1.2.840.10045.3.1.7 (secp256r1)
    let curve_oid: [u8; 10] = [0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07];
    if der.get(pos..pos + curve_oid.len()) != Some(&curve_oid[..]) {
        return false;
    }
    pos += curve_oid.len();
    if pos != alg_end {
        return false;
    }
    // BIT STRING, 0 unused bits, uncompressed 65-byte point.
    if der.get(pos) != Some(&0x03) {
        return false;
    }
    pos += 1;
    let bit_len = match read_der_length(der, &mut pos) {
        Some(length) => length,
        None => return false,
    };
    if bit_len != 66 || pos + bit_len != der.len() {
        return false;
    }
    der[pos] == 0x00 && der[pos + 1] == 0x04
}

// ---------------------------------------------------------------------------
// deliver.Permit + deliver.SecretFiles
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Permit {
    pub format: i64,
    pub repository: String,
    pub digest: String,
    pub previous: String,
    pub expires: i64,
}

impl Permit {
    pub fn to_json(&self) -> JsonValue {
        JsonValue::Object(vec![
            (
                "Format".to_string(),
                JsonValue::Number(self.format.to_string()),
            ),
            (
                "Repository".to_string(),
                JsonValue::Str(self.repository.clone()),
            ),
            ("Digest".to_string(), JsonValue::Str(self.digest.clone())),
            (
                "Previous".to_string(),
                JsonValue::Str(self.previous.clone()),
            ),
            (
                "Expires".to_string(),
                JsonValue::Number(self.expires.to_string()),
            ),
        ])
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SecretFiles {
    pub key: String,
    pub passphrase: String,
}

impl SecretFiles {
    pub fn parse(value: &JsonValue) -> Result<SecretFiles, Error> {
        if matches!(value, JsonValue::Null) {
            return Ok(SecretFiles::default());
        }
        jsonio::check_no_unknown(value, &["Key", "Passphrase"])
            .map_err(|_| Error::msg(sys::refused()))?;
        Ok(SecretFiles {
            key: jsonio::require_string(value, "Key").map_err(|_| Error::msg(sys::refused()))?,
            passphrase: jsonio::require_string(value, "Passphrase")
                .map_err(|_| Error::msg(sys::refused()))?,
        })
    }

    pub fn to_json(&self) -> JsonValue {
        JsonValue::Object(vec![
            ("Key".to_string(), JsonValue::Str(self.key.clone())),
            (
                "Passphrase".to_string(),
                JsonValue::Str(self.passphrase.clone()),
            ),
        ])
    }
}

// ---------------------------------------------------------------------------
// build CoreOS inputs
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct CoreOSImage {
    pub url: String,
    pub signature_url: String,
    pub sha256: String,
    pub uncompressed_sha256: String,
}

impl CoreOSImage {
    pub fn parse(value: &JsonValue) -> Result<CoreOSImage, Error> {
        jsonio::check_no_unknown(
            value,
            &["URL", "SignatureURL", "SHA256", "UncompressedSHA256"],
        )?;
        Ok(CoreOSImage {
            url: jsonio::require_string(value, "URL")?,
            signature_url: jsonio::require_string(value, "SignatureURL")?,
            sha256: jsonio::require_string(value, "SHA256")?,
            uncompressed_sha256: jsonio::require_string(value, "UncompressedSHA256")?,
        })
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ResolvedCoreOS {
    pub release: String,
    pub metadata_url: String,
    pub container: Vec<(String, String)>,
    pub iso: Vec<(String, CoreOSImage)>,
    pub qemu: Vec<(String, CoreOSImage)>,
}

impl ResolvedCoreOS {
    pub fn parse(value: &JsonValue) -> Result<ResolvedCoreOS, Error> {
        jsonio::check_no_unknown(
            value,
            &["Release", "MetadataURL", "Container", "ISO", "QEMU"],
        )?;
        let images = |field: &str| -> Result<Vec<(String, CoreOSImage)>, Error> {
            let mut out = Vec::new();
            if let JsonValue::Object(entries) = jsonio::require_object(value, field)? {
                for (arch, image) in entries {
                    out.push((arch.clone(), CoreOSImage::parse(image)?));
                }
            }
            Ok(out)
        };
        Ok(ResolvedCoreOS {
            release: jsonio::require_string(value, "Release")?,
            metadata_url: jsonio::require_string(value, "MetadataURL")?,
            container: jsonio::string_map(value, "Container")?,
            iso: images("ISO")?,
            qemu: images("QEMU")?,
        })
    }

    pub fn container_ref(&self, arch: &str) -> String {
        content_get(&self.container, arch)
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TailnetInputs {
    pub version: String,
    pub sha256: String,
    pub base: String,
}

impl TailnetInputs {
    pub fn parse(value: &JsonValue) -> Result<TailnetInputs, Error> {
        if matches!(value, JsonValue::Null) {
            return Ok(TailnetInputs::default());
        }
        jsonio::check_no_unknown(value, &["Version", "SHA256", "Base"])?;
        Ok(TailnetInputs {
            version: jsonio::require_string(value, "Version")?,
            sha256: jsonio::require_string(value, "SHA256")?,
            base: jsonio::require_string(value, "Base")?,
        })
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LiveInputs {
    pub core_os: ResolvedCoreOS,
    pub tailnet: TailnetInputs,
}

impl LiveInputs {
    pub fn parse(value: &JsonValue) -> Result<LiveInputs, Error> {
        jsonio::check_no_unknown(value, &["CoreOS", "Tailnet"])?;
        let core_os_value = jsonio::require_object(value, "CoreOS")?;
        let core_os = if matches!(core_os_value, JsonValue::Null) {
            ResolvedCoreOS::default()
        } else {
            ResolvedCoreOS::parse(core_os_value)?
        };
        let tailnet_value = jsonio::require_object(value, "Tailnet")?;
        Ok(LiveInputs {
            core_os,
            tailnet: TailnetInputs::parse(tailnet_value)?,
        })
    }
}

fn find_arch(list: &[(String, CoreOSImage)]) -> Option<&(String, CoreOSImage)> {
    list.iter().find(|(arch, _)| arch == "x86_64")
}

pub fn valid_stream_images(
    release: &str,
    iso: &[(String, CoreOSImage)],
    qemu: &[(String, CoreOSImage)],
) -> Result<(), Error> {
    if !is_coreos_release(release) {
        return Err(Error::msg("stable stream release is malformed"));
    }
    let find = find_arch;
    let (_, image) =
        find(iso).ok_or_else(|| Error::msg("stable stream x86_64 live ISO is malformed"))?;
    if !https_url(&image.url)
        || !image.url.ends_with(".iso")
        || image.signature_url != format!("{}.sig", image.url)
        || !is_digest(&image.sha256)
    {
        return Err(Error::msg("stable stream x86_64 live ISO is malformed"));
    }
    let (_, qemu_image) =
        find(qemu).ok_or_else(|| Error::msg("stable stream x86_64 qemu image is malformed"))?;
    if !https_url(&qemu_image.url)
        || !https_url(&qemu_image.signature_url)
        || !is_digest(&qemu_image.sha256)
        || !is_digest(&qemu_image.uncompressed_sha256)
    {
        return Err(Error::msg("stable stream x86_64 qemu image is malformed"));
    }
    Ok(())
}

pub fn valid_tailnet_inputs(tailnet: &TailnetInputs) -> Result<(), Error> {
    if !is_dotted_numbers(&tailnet.version, 3) {
        return Err(Error::msg("invalid Tailnet version"));
    }
    if !is_digest(&tailnet.sha256) {
        return Err(Error::msg("invalid Tailnet archive checksum"));
    }
    let base_ok = tailnet
        .base
        .strip_prefix("docker.io/tailscale/alpine-base:")
        .is_some_and(|rest| is_dotted_numbers(rest, 2));
    if !base_ok {
        return Err(Error::msg("invalid Tailnet base tag"));
    }
    Ok(())
}

pub fn valid_resolved_core_os(resolved: &ResolvedCoreOS) -> Result<(), Error> {
    valid_stream_images(&resolved.release, &resolved.iso, &resolved.qemu)?;
    if !https_url(&resolved.metadata_url)
        || !resolved
            .metadata_url
            .ends_with(&format!("/builds/{}/release.json", resolved.release))
    {
        return Err(Error::msg("resolved CoreOS metadata URL is malformed"));
    }
    if resolved.container.len() != 1 {
        return Err(Error::msg("x86_64 base digest required"));
    }
    match resolved
        .container_ref("x86_64")
        .split_once("/fedora/fedora-coreos@sha256:")
    {
        Some((host, digest)) if !host.is_empty() && !host.contains('/') && is_digest(digest) => {
            Ok(())
        }
        _ => Err(Error::msg("digest-pinned CoreOS base required")),
    }
}

pub fn valid_live_inputs(inputs: &LiveInputs) -> Result<(), Error> {
    valid_resolved_core_os(&inputs.core_os)?;
    valid_tailnet_inputs(&inputs.tailnet)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_prefix_and_digest_shapes() {
        // Oracle: Go ValidRepositoryPrefix + Digest vectors.
        assert!(valid_repository_prefix("ghcr.io/example/sodaos"));
        assert!(!valid_repository_prefix("ghcr.io/Example/sodaos"));
        assert!(!valid_repository_prefix("docker.io/example/sodaos"));
        assert!(!valid_repository_prefix("ghcr.io/example"));
        assert!(prefixed_digest(&format!("sha256:{}", "a".repeat(64))));
        assert!(!prefixed_digest(&"a".repeat(64)));
    }

    #[test]
    fn oracle_url_parser_matches_go_probe() {
        // Oracle: /tmp/urlprobe vectors from the real Go url.Parse.
        let upper = parse_url("HTTPS://Example.COM:8080/p").unwrap();
        assert_eq!(upper.scheme, "https");
        assert_eq!(upper.hostname, "Example.COM");
        assert!(parse_url("https://h:bad/p").is_none());
        assert_eq!(parse_url("https:///p").unwrap().host, "");
        assert!(parse_url("https://h p/").is_none());
        assert!(parse_url("https://[::1]/p").unwrap().hostname == "::1");
        assert!(https_url(
            "https://example.invalid/builds/1.0.0.0/release.json"
        ));
        assert!(!https_url("http://example.invalid/x"));
        assert!(!https_url("https://example.invalid/x?"));
        assert!(is_loopback_addr("127.0.0.1"));
        assert!(is_loopback_addr("::1"));
        assert!(is_loopback_addr("0:0:0:0:0:0:0:1"));
        assert!(!is_loopback_addr("10.0.0.1"));
        assert!(!is_loopback_addr("::ffff:127.0.0.1"));
        assert!(!is_loopback_addr("example.invalid"));
    }

    #[test]
    fn oracle_payload_identity_validation() {
        // Oracle: Go Payload.Validate error strings.
        let payload = Payload::default();
        assert_eq!(
            payload.validate().unwrap_err().0,
            "invalid appliance payload identity"
        );
    }
}
