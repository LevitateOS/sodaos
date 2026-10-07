use serde::ser::SerializeStruct;
use serde::{Serialize, Serializer};

use crate::error::Error;
use crate::jsonio;
use crate::sys;

use super::{
    content_get, deliver_hash, is_digest, is_revision, oci_architecture, prefixed_digest, Image,
    Payload, FORGEJO_COMPILER_IMAGE, SODA_SOURCE,
};

// ---------------------------------------------------------------------------
// deliver.Candidate
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct ForgejoToolchain {
    #[serde(rename = "CompilerImage")]
    pub compiler_image: String,
    #[serde(rename = "APKPackages")]
    pub apk_packages: Vec<String>,
}

impl ForgejoToolchain {
    pub fn parse(text: &str) -> Result<ForgejoToolchain, Error> {
        jsonio::parse(text)
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

crate::jsonio::case_record!(ForgejoToolchain, {
    compiler_image: String => "CompilerImage",
    apk_packages: Vec<String> => "APKPackages",
});

#[derive(Default)]
struct CandidateWire {
    format: i64,
    host: Image,
    host_reference: String,
    host_archive_sha256: String,
    payload_sha256: String,
    migration: String,
    notes: String,
    forgejo_revision: String,
    forgejo_source_sha256: String,
    forgejo_toolchain: ForgejoToolchain,
    architecture: String,
    content_sha256: crate::jsonio::OrderedMap<String>,
}

crate::jsonio::case_record!(CandidateWire, {
    format: i64 => "Format",
    host: Image => "Host",
    host_reference: String => "HostReference",
    host_archive_sha256: String => "HostArchiveSHA256",
    payload_sha256: String => "PayloadSHA256",
    migration: String => "Migration",
    notes: String => "Notes",
    forgejo_revision: String => "ForgejoRevision",
    forgejo_source_sha256: String => "ForgejoSourceSHA256",
    forgejo_toolchain: ForgejoToolchain => "ForgejoToolchain",
    architecture: String => "Architecture",
    content_sha256: crate::jsonio::OrderedMap<String> => "ContentSHA256",
});

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

impl Serialize for Candidate {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        let mut record = serializer.serialize_struct("Candidate", 12)?;
        record.serialize_field("Format", &self.format)?;
        record.serialize_field("Host", &self.host)?;
        record.serialize_field("HostReference", &self.host_reference)?;
        record.serialize_field("HostArchiveSHA256", &self.host_archive_sha256)?;
        record.serialize_field("PayloadSHA256", &self.payload_sha256)?;
        record.serialize_field("Migration", &self.migration)?;
        record.serialize_field("Notes", &self.notes)?;
        record.serialize_field("ForgejoRevision", &self.forgejo_revision)?;
        record.serialize_field("ForgejoSourceSHA256", &self.forgejo_source_sha256)?;
        record.serialize_field("ForgejoToolchain", &self.forgejo_toolchain)?;
        record.serialize_field("Architecture", &self.architecture)?;
        record.serialize_field(
            "ContentSHA256",
            &crate::jsonio::SortedPairs(&self.content_sha256),
        )?;
        record.end()
    }
}

impl Candidate {
    pub fn parse(text: &str) -> Result<Candidate, Error> {
        let wire: CandidateWire = jsonio::parse(text)?;
        Ok(Candidate {
            format: wire.format,
            host: wire.host,
            host_reference: wire.host_reference,
            host_archive_sha256: wire.host_archive_sha256,
            payload_sha256: wire.payload_sha256,
            migration: wire.migration,
            notes: wire.notes,
            forgejo_revision: wire.forgejo_revision,
            forgejo_source_sha256: wire.forgejo_source_sha256,
            forgejo_toolchain: wire.forgejo_toolchain,
            architecture: wire.architecture,
            content_sha256: wire.content_sha256.0,
        })
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
