//! Completed producer receipt shared by build emission and delivery admission.

use serde::{Deserialize, Serialize};

/// Final controller success after worker cleanup and progress finalization.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct BuildCompletion {
    #[serde(rename = "BuildSHA256")]
    pub build_sha256: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct BuildResult {
    #[serde(rename = "Revision")]
    pub revision: String,
    #[serde(rename = "Architecture")]
    pub architecture: String,
    #[serde(rename = "Candidate")]
    pub candidate: String,
    #[serde(rename = "CandidateSHA256")]
    pub candidate_sha256: String,
    #[serde(rename = "HostManifest")]
    pub host_manifest: String,
    #[serde(rename = "PayloadSHA256")]
    pub payload_sha256: String,
    #[serde(rename = "Scope")]
    pub scope: String,
    #[serde(rename = "Media", default, skip_serializing_if = "String::is_empty")]
    pub media: String,
    #[serde(rename = "Purpose")]
    pub purpose: String,
    #[serde(rename = "RequestedTarget")]
    pub requested_target: String,
    #[serde(rename = "CompletedTarget")]
    pub completed_target: String,
    #[serde(rename = "Checks")]
    pub checks: Vec<String>,
    #[serde(
        rename = "MediaCompression",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub media_compression: String,
    #[serde(rename = "Distribution")]
    pub distribution: String,
    #[serde(
        rename = "ReleaseTag",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub release_tag: String,
    #[serde(rename = "Docker", default, skip_serializing_if = "String::is_empty")]
    pub docker: String,
}
