use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

use crate::buildx::{
    is_digest, is_revision, oci_architecture, ForgejoToolchain, Image as BuildImage,
};
use crate::payload::{valid_repository_prefix, Payload};
use crate::{hash_bytes, is_digest_ref, Error};

// ---------------------------------------------------------------------------
// Candidate
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct Candidate {
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub format: i64,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub host: BuildImage,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub host_reference: String,
    #[serde(rename = "HostArchiveSHA256")]
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub host_archive_sha256: String,
    /// Repository policy for this candidate's image references. The v4
    /// payload no longer carries it; build/publication metadata does.
    #[serde(rename = "RepositoryPrefix")]
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub repository_prefix: String,
    /// Expected per-image OCI archive hashes of the delivered candidate
    /// transport, keyed by fixed role name. The runtime payload carries no
    /// per-image archive checksums by design.
    #[serde(rename = "ImageArchives")]
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub image_archives: BTreeMap<String, String>,
    #[serde(rename = "PayloadSHA256")]
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub payload_sha256: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub migration: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub notes: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub forgejo_revision: String,
    #[serde(rename = "ForgejoSourceSHA256")]
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub forgejo_source_sha256: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub forgejo_toolchain: ForgejoToolchain,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub architecture: String,
    #[serde(rename = "ContentSHA256")]
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub content_sha256: BTreeMap<String, String>,
}

fn valid_candidate_host(c: &Candidate, p: &Payload, arch: &str) -> bool {
    c.host_reference == format!("{}-host@{}", c.repository_prefix, c.host.manifest)
        && is_digest_ref(&c.host.manifest)
        && is_digest_ref(&c.host.config)
        && is_digest(&c.host_archive_sha256)
        && c.host.architecture == arch
        && c.host.revision == p.revision
        && c.host.base_name == p.base
}

fn valid_candidate_provenance(c: &Candidate, p: &Payload, payload: &[u8]) -> bool {
    valid_candidate_source(c, p, payload)
        && valid_candidate_build(c, p)
        && valid_candidate_content(&c.content_sha256)
        && !c.migration.is_empty()
        && !c.notes.is_empty()
}

fn valid_candidate_source(c: &Candidate, p: &Payload, payload: &[u8]) -> bool {
    let base_digest = p.base.split('@').nth(1).unwrap_or("");
    c.format == 1
        && c.payload_sha256 == hash_bytes(payload).trim_start_matches("sha256:")
        && c.host.base_digest == base_digest
        && c.host.source == "https://github.com/LevitateOS/sodaos"
}

fn valid_candidate_build(c: &Candidate, p: &Payload) -> bool {
    is_revision(&c.forgejo_revision)
        && c.forgejo_revision == p.fountain.source_revision
        && is_digest(&c.forgejo_source_sha256)
        && ((c.forgejo_toolchain.compiler_image.is_empty()
            && c.forgejo_toolchain.apk_packages.is_empty())
            || c.forgejo_toolchain.validate().is_ok())
        && c.architecture == p.architecture
}

pub fn valid_candidate_content(files: &BTreeMap<String, String>) -> bool {
    soda_build_tools::reader::candidate::valid_content(files)
}

pub(crate) fn path_clean(name: &str) -> String {
    if name.is_empty() {
        return ".".to_string();
    }
    let rooted = name.starts_with('/');
    let mut parts: Vec<&str> = Vec::new();
    for segment in name.split('/') {
        match segment {
            "" | "." => {}
            ".." => {
                if parts.pop().is_none() && !rooted {
                    parts.push("..");
                }
            }
            _ => parts.push(segment),
        }
    }
    // Go keeps leading ".." for relative paths; the pop/push above models it.
    let mut out = if rooted {
        "/".to_string()
    } else {
        String::new()
    };
    out.push_str(&parts.join("/"));
    if out.is_empty() {
        ".".to_string()
    } else {
        out
    }
}

impl Candidate {
    pub fn validate(&self, p: &Payload, payload: &[u8]) -> Result<(), Error> {
        let arch = oci_architecture(&p.architecture).map_err(|e| Error::msg(e.0))?;
        if p.validate().is_err() || !valid_repository_prefix(&self.repository_prefix) {
            return Err(Error::refused());
        }
        if !valid_candidate_host(self, p, arch) || !valid_candidate_provenance(self, p, payload) {
            return Err(Error::refused());
        }
        Ok(())
    }
}
