//! Canonical Payload v4 wire types and validation (RCX-2 packet R01).
//!
//! Single Rust owner for the appliance payload document. Every producer and
//! reader imports this definition; there are no parallel DTOs and no v3
//! branch. The envelope keeps PascalCase; the Fountain catalog mirror keeps
//! the exact lower-case field names and values of the frozen F01 SDK JSON
//! contract (authoritative Go type stays in Fountain).
//!
//! Validation implements release-design.md rules 1-7: identity with
//! `Format=4` and the fixed release tag, six fixed roles with
//! Soda/upstream provenance, digest-pinned runtime references on
//! `linux/amd64`, frozen Fountain release binding, `ExtensionCatalog`
//! version 1 with exactly `soda` then `tailnet`, the optional Tailnet
//! official-image binding, and no build-cycle checksums.

use serde::{Deserialize, Serialize};

use crate::project_support::ProjectSupport;
use crate::reader::Error;

/// Accepted payload format. Old formats are rejected, never migrated.
pub const FORMAT: u32 = 4;
/// Frozen Fountain catalog version declared by Soda releases.
pub const CATALOG_VERSION: u32 = 1;
/// Frozen Fountain extension protocol and browser contract in this cutover.
pub const EXTENSION_PROTOCOL: u32 = 1;
pub const BROWSER_CONTRACT: u32 = 1;
/// Fountain archive cap mirrored from the frozen SDK: 512 MiB compressed.
pub const MAX_PACKAGE_BYTES: u64 = 512 << 20;
/// ID of the required native package and its lifecycle subject.
pub const SODA_PACKAGE_ID: &str = "soda";
/// ID of the optional native package controlled by the required one.
pub const TAILNET_PACKAGE_ID: &str = "tailnet";
/// Approved required-package capabilities (Fountain/Tailnet refinement).
pub const SODA_CAPABILITIES: [&str; 9] = [
    "native.actor.read",
    "native.repository.read",
    "native.repositories.owned.search",
    "native.organization.ownership",
    "native.user.public_keys.read",
    "native.contribution.authorize",
    "native.service.bridge",
    "native.background.operations",
    "native.extension.lifecycle",
];
/// Approved optional-package capabilities (Tailnet refinement).
pub const TAILNET_CAPABILITIES: [&str; 2] = ["native.actor.read", "native.service.bridge"];

fn invalid(message: impl Into<String>) -> Error {
    Error(message.into())
}

fn is_hex_digest(s: &str) -> bool {
    s.len() == 64 && s.bytes().all(|b| matches!(b, b'0'..=b'9' | b'a'..=b'f'))
}

fn is_prefixed_digest(s: &str) -> bool {
    s.strip_prefix("sha256:")
        .is_some_and(|hex| is_hex_digest(hex))
}

/// Frozen SDK package version shape: `[A-Za-z0-9][A-Za-z0-9._+-]{0,63}`.
fn valid_package_version(s: &str) -> bool {
    let mut bytes = s.bytes();
    if !bytes.next().is_some_and(|b| b.is_ascii_alphanumeric()) || s.len() > 64 {
        return false;
    }
    bytes.all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'+' | b'-'))
}

fn is_coreos_release(s: &str) -> bool {
    let parts: Vec<&str> = s.split('.').collect();
    parts.len() == 4
        && parts
            .iter()
            .all(|p| !p.is_empty() && p.bytes().all(|b| b.is_ascii_digit()))
}

/// Fixed HTTPS release-asset URL: no credentials, query or fragment.
fn valid_asset_url(s: &str) -> bool {
    crate::reader::url::https_url(s)
}

/// `registry/repository@sha256:<manifest>` with a digest suffix.
fn valid_digest_reference(s: &str) -> bool {
    match s.rsplit_once('@') {
        Some((repo, digest)) => !repo.is_empty() && is_prefixed_digest(digest),
        None => false,
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum Role {
    Dashboard,
    Forgejo,
    Proxy,
    Postgres,
    ProjectOs,
    CanlangRuntime,
}

impl Role {
    pub const ALL: [Role; 6] = [
        Role::Dashboard,
        Role::Forgejo,
        Role::Proxy,
        Role::Postgres,
        Role::ProjectOs,
        Role::CanlangRuntime,
    ];

    pub fn name(&self) -> &'static str {
        match self {
            Role::Dashboard => "dashboard",
            Role::Forgejo => "forgejo",
            Role::Proxy => "proxy",
            Role::Postgres => "postgres",
            Role::ProjectOs => "project-os",
            Role::CanlangRuntime => "canlang-runtime",
        }
    }

    pub fn from_name(name: &str) -> Option<Role> {
        Role::ALL.into_iter().find(|role| role.name() == name)
    }

    pub fn iter() -> impl Iterator<Item = Role> {
        Role::ALL.into_iter()
    }

    pub fn binding<'a>(&self, images: &'a RequiredImages) -> &'a ImageBinding {
        match self {
            Role::Dashboard => &images.dashboard,
            Role::Forgejo => &images.forgejo,
            Role::Proxy => &images.proxy,
            Role::Postgres => &images.postgres,
            Role::ProjectOs => &images.project_os,
            Role::CanlangRuntime => &images.canlang_runtime,
        }
    }

    pub fn binding_mut<'a>(&self, images: &'a mut RequiredImages) -> &'a mut ImageBinding {
        match self {
            Role::Dashboard => &mut images.dashboard,
            Role::Forgejo => &mut images.forgejo,
            Role::Proxy => &mut images.proxy,
            Role::Postgres => &mut images.postgres,
            Role::ProjectOs => &mut images.project_os,
            Role::CanlangRuntime => &mut images.canlang_runtime,
        }
    }
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct PayloadV4 {
    pub format: u32,
    #[serde(rename = "ID")]
    pub id: String,
    pub release_tag: String,
    pub revision: String,
    pub architecture: String,
    #[serde(rename = "CoreOS")]
    pub core_os: String,
    pub base: String,
    pub schema: u32,
    #[serde(rename = "PresentationSHA256")]
    pub presentation_sha256: String,
    #[serde(rename = "HostPackagesSHA256")]
    pub host_packages_sha256: String,
    pub project_support: ProjectSupport,
    pub fountain: FountainRelease,
    pub images: RequiredImages,
    pub extension_catalog: FountainCatalog,
    pub optional_extensions: OptionalExtensions,
    pub upgrade_from: Vec<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct RequiredImages {
    pub dashboard: ImageBinding,
    pub forgejo: ImageBinding,
    pub proxy: ImageBinding,
    pub postgres: ImageBinding,
    #[serde(rename = "project-os")]
    pub project_os: ImageBinding,
    #[serde(rename = "canlang-runtime")]
    pub canlang_runtime: ImageBinding,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct ImageBinding {
    pub reference: String,
    pub manifest: String,
    pub config: String,
    pub platform: String,
    pub provenance: ImageProvenance,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(tag = "Kind", deny_unknown_fields)]
pub enum ImageProvenance {
    Soda {
        #[serde(rename = "Revision")]
        revision: String,
        #[serde(rename = "Base")]
        base: String,
    },
    Upstream {
        #[serde(rename = "Requested")]
        requested: String,
    },
}

impl Default for ImageProvenance {
    fn default() -> Self {
        ImageProvenance::Upstream {
            requested: String::new(),
        }
    }
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct ReleaseFile {
    #[serde(rename = "URL")]
    pub url: String,
    pub bytes: u64,
    #[serde(rename = "SHA256")]
    pub sha256: String,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct FountainRelease {
    pub version: String,
    pub metadata: ReleaseFile,
    pub image: String,
    pub image_config: String,
    pub source_revision: String,
    pub source: ReleaseFile,
    pub prepared_source: ReleaseFile,
    pub extension_protocol: u32,
    pub browser_contract: u32,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct FountainCatalog {
    pub version: u32,
    pub packages: Vec<FountainPackage>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct FountainPackage {
    pub id: String,
    pub version: String,
    pub url: String,
    pub size: u64,
    pub sha256: String,
    pub required: bool,
    pub fountain_protocol: u32,
    pub browser_contract: u32,
    pub capabilities: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub lifecycle_controller: Option<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct OptionalExtensions {
    pub tailnet: TailnetDependency,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct TailnetDependency {
    #[serde(rename = "PackageID")]
    pub package_id: String,
    pub images: TailnetImages,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct TailnetImages {
    pub tailscale: ImageBinding,
}

fn validate_binding(binding: &ImageBinding, role: &str) -> Result<(), Error> {
    if !is_prefixed_digest(&binding.manifest)
        || !is_prefixed_digest(&binding.config)
        || binding.platform != "linux/amd64"
    {
        return Err(invalid(format!("invalid {role} image binding")));
    }
    match binding.reference.rsplit_once('@') {
        Some((repo, digest)) if !repo.is_empty() && digest == binding.manifest => {}
        _ => return Err(invalid(format!("invalid {role} image binding"))),
    }
    Ok(())
}

fn validate_release_file(file: &ReleaseFile, what: &str) -> Result<(), Error> {
    if !valid_asset_url(&file.url) || file.bytes == 0 || !is_hex_digest(&file.sha256) {
        return Err(invalid(format!("invalid {what} release file")));
    }
    Ok(())
}

fn validate_package(
    package: &FountainPackage,
    id: &str,
    required: bool,
    controller: Option<&str>,
    capabilities: &[&str],
    version: &str,
) -> Result<(), Error> {
    if package.id != id
        || package.required != required
        || package.lifecycle_controller.as_deref() != controller
        || !valid_package_version(&package.version)
        || package.version != version
        || package.size == 0
        || package.size > MAX_PACKAGE_BYTES
        || !is_hex_digest(&package.sha256)
        || package.fountain_protocol != EXTENSION_PROTOCOL
        || package.browser_contract != BROWSER_CONTRACT
        || package.capabilities != capabilities
    {
        return Err(invalid(format!("invalid {id} catalog package")));
    }
    let filename = format!("{id}-extension-{}-linux-x86_64.tar.gz", package.version);
    match package.url.rsplit_once('/') {
        Some((_, name)) if name == filename && valid_asset_url(&package.url) => {}
        _ => return Err(invalid(format!("invalid {id} catalog package"))),
    }
    if controller.is_some_and(|c| c == package.id) {
        return Err(invalid(format!("invalid {id} catalog package")));
    }
    Ok(())
}

impl PayloadV4 {
    /// Strictly parse one payload document: unknown fields, duplicate keys
    /// and trailing data fail; every field must be present and validate.
    pub fn parse(data: &[u8]) -> Result<PayloadV4, Error> {
        let payload: PayloadV4 =
            serde_json::from_slice(data).map_err(|_| invalid("invalid payload document"))?;
        payload.validate()?;
        Ok(payload)
    }

    pub fn validate(&self) -> Result<(), Error> {
        if self.format != FORMAT
            || !crate::reader::is_revision(&self.revision)
            || self.id != format!("{}.soda-{}", self.core_os, &self.revision[..12])
            || !is_coreos_release(&self.core_os)
        {
            return Err(invalid("invalid appliance payload identity"));
        }
        if self.architecture != "x86_64" {
            return Err(invalid("expected x86_64"));
        }
        let Some(version) = self.release_tag.strip_prefix('v') else {
            return Err(invalid("invalid appliance payload identity"));
        };
        if version.is_empty() {
            return Err(invalid("invalid appliance payload identity"));
        }
        match self.base.split_once("/fedora/fedora-coreos@sha256:") {
            Some((host, digest))
                if !host.is_empty()
                    && !host.contains('/')
                    && is_hex_digest(digest)
                    && self.schema >= 1
                    && is_hex_digest(&self.presentation_sha256)
                    && is_hex_digest(&self.host_packages_sha256) => {}
            _ => return Err(invalid("incomplete appliance payload")),
        }
        for role in Role::iter() {
            validate_binding(role.binding(&self.images), role.name())?;
        }
        for role in Role::iter() {
            let binding = role.binding(&self.images);
            match (&role, &binding.provenance) {
                (
                    Role::Dashboard | Role::Forgejo | Role::ProjectOs | Role::CanlangRuntime,
                    ImageProvenance::Soda { revision, base },
                ) if revision == &self.revision && valid_digest_reference(base) => {}
                (Role::Proxy | Role::Postgres, ImageProvenance::Upstream { requested })
                    if valid_digest_reference(requested) => {}
                _ => return Err(invalid(format!("invalid {} image provenance", role.name()))),
            }
        }
        if let ImageProvenance::Soda { base, .. } = &self.images.forgejo.provenance {
            if base != &self.fountain.image {
                return Err(invalid("invalid forgejo image provenance"));
            }
        } else {
            return Err(invalid("invalid forgejo image provenance"));
        }
        self.project_support
            .validate(&self.images.project_os.config)
            .map_err(invalid)?;
        validate_release_file(&self.fountain.metadata, "fountain metadata")?;
        validate_release_file(&self.fountain.source, "fountain source")?;
        validate_release_file(&self.fountain.prepared_source, "fountain prepared source")?;
        if self.fountain.version.is_empty()
            || !valid_digest_reference(&self.fountain.image)
            || !is_prefixed_digest(&self.fountain.image_config)
            || !crate::reader::is_revision(&self.fountain.source_revision)
            || self.fountain.extension_protocol != EXTENSION_PROTOCOL
            || self.fountain.browser_contract != BROWSER_CONTRACT
        {
            return Err(invalid("invalid fountain release binding"));
        }
        if self.extension_catalog.version != CATALOG_VERSION
            || self.extension_catalog.packages.len() != 2
        {
            return Err(invalid("invalid extension catalog"));
        }
        validate_package(
            &self.extension_catalog.packages[0],
            SODA_PACKAGE_ID,
            true,
            None,
            &SODA_CAPABILITIES,
            version,
        )?;
        validate_package(
            &self.extension_catalog.packages[1],
            TAILNET_PACKAGE_ID,
            false,
            Some(SODA_PACKAGE_ID),
            &TAILNET_CAPABILITIES,
            version,
        )?;
        if self.optional_extensions.tailnet.package_id != TAILNET_PACKAGE_ID {
            return Err(invalid("invalid tailnet extension binding"));
        }
        validate_binding(
            &self.optional_extensions.tailnet.images.tailscale,
            "tailscale",
        )?;
        if !matches!(
            self.optional_extensions.tailnet.images.tailscale.provenance,
            ImageProvenance::Upstream { .. }
        ) {
            return Err(invalid("invalid tailscale image provenance"));
        }
        crate::project_support::validate_upgrade_sources(&self.upgrade_from, &self.id)
            .map_err(invalid)?;
        Ok(())
    }

    /// Project the exact catalog bytes bound into Fountain: pretty JSON plus
    /// one final newline. Hashing and mounts reuse these same bytes.
    pub fn projected_catalog_bytes(&self) -> Result<Vec<u8>, Error> {
        let mut bytes = serde_json::to_vec_pretty(&self.extension_catalog)
            .map_err(|_| invalid("invalid extension catalog"))?;
        bytes.push(b'\n');
        Ok(bytes)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    pub(crate) const FIXTURE: &str = include_str!("../fixtures/payload-v4.json");

    fn fixture() -> PayloadV4 {
        PayloadV4::parse(FIXTURE.as_bytes()).expect("shared v4 fixture must validate")
    }

    fn mutate(key: &str, value: &str) -> Vec<u8> {
        let mut doc: serde_json::Value =
            serde_json::from_str(FIXTURE).expect("fixture must be JSON");
        let slot = doc
            .get_mut(key)
            .unwrap_or_else(|| panic!("fixture lacks {key}"));
        *slot = serde_json::from_str(value).expect("replacement must be JSON");
        serde_json::to_vec(&doc).expect("fixture must serialize")
    }

    #[test]
    fn shared_fixture_validates_and_projects_stable_catalog() {
        let payload = fixture();
        assert_eq!(payload.format, FORMAT);
        assert_eq!(Role::ALL.len(), 6);
        let bytes = payload.projected_catalog_bytes().unwrap();
        assert!(bytes.ends_with(b"\n}\n") || bytes.ends_with(b"\n"));
        assert_eq!(bytes, payload.projected_catalog_bytes().unwrap());
        let catalog: FountainCatalog =
            serde_json::from_slice(&bytes).expect("projected catalog must decode");
        assert_eq!(catalog.version, CATALOG_VERSION);
        assert_eq!(catalog.packages.len(), 2);
    }

    #[test]
    fn roles_cover_the_fixed_set_with_accessors() {
        let payload = fixture();
        let names: Vec<_> = Role::iter().map(|role| role.name()).collect();
        assert_eq!(
            names,
            [
                "dashboard",
                "forgejo",
                "proxy",
                "postgres",
                "project-os",
                "canlang-runtime"
            ]
        );
        for role in Role::iter() {
            assert_eq!(Role::from_name(role.name()), Some(role));
            assert!(!role.binding(&payload.images).reference.is_empty());
        }
        assert_eq!(Role::from_name("extension"), None);
        assert_eq!(Role::from_name("tailnet"), None);
    }

    #[test]
    fn old_format_and_envelope_violations_fail() {
        assert!(PayloadV4::parse(&mutate("Format", "3")).is_err());
        assert!(PayloadV4::parse(&mutate("ReleaseTag", "\"\"")).is_err());
        assert!(PayloadV4::parse(&mutate("ReleaseTag", "\"0.0.0-no-v\"")).is_err());
        assert!(PayloadV4::parse(&mutate("Architecture", "\"amd64\"")).is_err());
        assert!(PayloadV4::parse(&mutate("UpgradeFrom", "null")).is_err());
        let mut extended = FIXTURE.as_bytes().to_vec();
        extended.extend_from_slice(b" ");
        assert!(PayloadV4::parse(&extended).is_ok());
        let mut trailing = FIXTURE.as_bytes().to_vec();
        trailing.extend_from_slice(b"{}");
        assert!(PayloadV4::parse(&trailing).is_err());
        let unknown =
            FIXTURE.replacen("\"Schema\":1", "\"Schema\":1,\"RepositoryPrefix\":\"x\"", 1);
        assert!(PayloadV4::parse(unknown.as_bytes()).is_err());
        let duplicated = FIXTURE.replacen("\"dashboard\":{", "\"dashboard\":{},\"dashboard\":{", 1);
        assert!(PayloadV4::parse(duplicated.as_bytes()).is_err());
    }

    #[test]
    fn image_and_catalog_violations_fail() {
        let mut doc: serde_json::Value = serde_json::from_str(FIXTURE).unwrap();
        doc["Images"]["proxy"]["Provenance"] = serde_json::json!({"Kind": "Soda", "Revision": doc["Revision"].clone(), "Base": "quay.io/x@sha256:0000000000000000000000000000000000000000000000000000000000000000"});
        assert!(PayloadV4::parse(&serde_json::to_vec(&doc).unwrap()).is_err());
        let mut doc: serde_json::Value = serde_json::from_str(FIXTURE).unwrap();
        doc["ExtensionCatalog"]["packages"][1]["lifecycle_controller"] =
            serde_json::json!("tailnet");
        assert!(PayloadV4::parse(&serde_json::to_vec(&doc).unwrap()).is_err());
        let mut doc: serde_json::Value = serde_json::from_str(FIXTURE).unwrap();
        doc["ExtensionCatalog"]["packages"][0]["capabilities"] = serde_json::json!([]);
        assert!(PayloadV4::parse(&serde_json::to_vec(&doc).unwrap()).is_err());
        let mut doc: serde_json::Value = serde_json::from_str(FIXTURE).unwrap();
        doc["OptionalExtensions"]["tailnet"]["PackageID"] = serde_json::json!("soda");
        assert!(PayloadV4::parse(&serde_json::to_vec(&doc).unwrap()).is_err());
    }
}
