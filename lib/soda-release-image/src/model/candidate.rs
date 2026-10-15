use serde::ser::SerializeStruct;
use serde::{Serialize, Serializer};

use crate::error::Error;
use crate::jsonio;
use crate::sys;

use super::{
    deliver_hash, is_digest, is_revision, oci_architecture, prefixed_digest, Image, Payload,
    FORGEJO_COMPILER_IMAGE, SODA_SOURCE,
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
    repository_prefix: String,
    image_archives: crate::jsonio::OrderedMap<String>,
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
    repository_prefix: String => "RepositoryPrefix",
    image_archives: crate::jsonio::OrderedMap<String> => "ImageArchives",
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
    pub repository_prefix: String,
    pub image_archives: Vec<(String, String)>,
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
        let mut record = serializer.serialize_struct("Candidate", 14)?;
        record.serialize_field("Format", &self.format)?;
        record.serialize_field("Host", &self.host)?;
        record.serialize_field("HostReference", &self.host_reference)?;
        record.serialize_field("HostArchiveSHA256", &self.host_archive_sha256)?;
        record.serialize_field("RepositoryPrefix", &self.repository_prefix)?;
        record.serialize_field(
            "ImageArchives",
            &crate::jsonio::SortedPairs(&self.image_archives),
        )?;
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
            repository_prefix: wire.repository_prefix,
            image_archives: wire.image_archives.0,
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
            candidate.repository_prefix, candidate.host.manifest
        )
        && super::valid_repository_prefix(&candidate.repository_prefix)
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
    // The selected Fountain release/source proof replaces the local fork
    // toolchain proof: an explicitly empty toolchain is the release-proof
    // mode, never a forged one. A non-empty toolchain must still validate
    // exactly; revision and proof digests always bind the selected release.
    let toolchain = &candidate.forgejo_toolchain;
    let release_proof_mode =
        toolchain.compiler_image.is_empty() && toolchain.apk_packages.is_empty();
    is_revision(&candidate.forgejo_revision)
        && is_digest(&candidate.forgejo_source_sha256)
        && (toolchain.validate().is_ok() || release_proof_mode)
        && candidate.architecture == payload.architecture
}

pub fn valid_candidate_content(files: &[(String, String)]) -> bool {
    let inventory: std::collections::BTreeMap<String, String> = files.iter().cloned().collect();
    inventory.len() == files.len() && soda_build_tools::reader::candidate::valid_content(&inventory)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn proof_candidate() -> (Candidate, Payload) {
        let payload = Payload {
            architecture: "x86_64".to_string(),
            ..Payload::default()
        };
        let candidate = Candidate {
            forgejo_revision: "a".repeat(40),
            forgejo_source_sha256: "b".repeat(64),
            architecture: "x86_64".to_string(),
            ..Candidate::default()
        };
        (candidate, payload)
    }

    #[test]
    fn release_proof_mode_accepts_empty_toolchain_only() {
        // Empty toolchain = selected release/source proof mode, never forged.
        let (candidate, payload) = proof_candidate();
        assert!(valid_candidate_build(&candidate, &payload));
        let mut forged = candidate.clone();
        forged.forgejo_toolchain.compiler_image = "example.invalid/toolchain:latest".to_string();
        assert!(!valid_candidate_build(&forged, &payload));
        let mut exact = candidate.clone();
        exact.forgejo_toolchain.compiler_image = FORGEJO_COMPILER_IMAGE.to_string();
        exact.forgejo_toolchain.apk_packages = vec![
            "build-base-0.5-r4".to_string(),
            "gcc-14.2.0-r6".to_string(),
            "musl-dev-1.2.5-r10".to_string(),
        ];
        assert!(valid_candidate_build(&exact, &payload));
        let mut bad_revision = candidate.clone();
        bad_revision.forgejo_revision = "short".to_string();
        assert!(!valid_candidate_build(&bad_revision, &payload));
    }

    #[test]
    fn thin_content_inventory_requires_new_layout() {
        let digest = "c".repeat(64);
        let mut files: Vec<(String, String)> =
            soda_build_tools::reader::candidate::REQUIRED_CONTENT
                .iter()
                .map(|name| (name.to_string(), digest.clone()))
                .collect();
        files.push((
            "package:soda-extension-0.1.0-linux-x86_64.tar.gz".to_string(),
            digest.clone(),
        ));
        files.push((
            "package:tailnet-extension-0.1.0-linux-x86_64.tar.gz".to_string(),
            digest.clone(),
        ));
        assert!(valid_candidate_content(&files));
        files.push((
            "extension:/usr/share/soda/extension/backend".to_string(),
            digest.clone(),
        ));
        assert!(!valid_candidate_content(&files), "no extension OCI entries");
        files.pop();
        files.retain(|(name, _)| name != "host:/usr/share/soda/extensions/catalog.json");
        assert!(!valid_candidate_content(&files), "catalog required");
    }
}
