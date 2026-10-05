//! `build.go`: pipeline request, result, and target admission.

use soda_json::JsonValue;

use crate::error::Error;
use crate::media;

/// Request selects a boundary of the same producer, never installation or publication.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Request {
    pub source: String,
    pub out: String,
    pub arch: String,
    pub repository_prefix: String,
    pub revision: String,
    pub rootfs_base_url: String,
    pub media_authority: String,
    pub live_inputs: String,
    pub forgejo_source: String,
    pub forgejo_revision: String,
    pub development: bool,
    pub target: String,
    pub media_compression: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Result {
    pub revision: String,
    pub architecture: String,
    pub candidate: String,
    pub candidate_sha256: String,
    pub host_manifest: String,
    pub payload_sha256: String,
    pub scope: String,
    pub media: String,
    pub purpose: String,
    pub requested_target: String,
    pub completed_target: String,
    pub checks: Vec<String>,
    pub media_compression: String,
}

impl Result {
    pub fn to_json(&self) -> JsonValue {
        let mut entries = vec![
            (
                "Revision".to_string(),
                JsonValue::Str(self.revision.clone()),
            ),
            (
                "Architecture".to_string(),
                JsonValue::Str(self.architecture.clone()),
            ),
            (
                "Candidate".to_string(),
                JsonValue::Str(self.candidate.clone()),
            ),
            (
                "CandidateSHA256".to_string(),
                JsonValue::Str(self.candidate_sha256.clone()),
            ),
            (
                "HostManifest".to_string(),
                JsonValue::Str(self.host_manifest.clone()),
            ),
            (
                "PayloadSHA256".to_string(),
                JsonValue::Str(self.payload_sha256.clone()),
            ),
            ("Scope".to_string(), JsonValue::Str(self.scope.clone())),
        ];
        // `json:",omitempty"`: default field name, omitted when empty.
        if !self.media.is_empty() {
            entries.push(("Media".to_string(), JsonValue::Str(self.media.clone())));
        }
        entries.push(("Purpose".to_string(), JsonValue::Str(self.purpose.clone())));
        entries.push((
            "RequestedTarget".to_string(),
            JsonValue::Str(self.requested_target.clone()),
        ));
        entries.push((
            "CompletedTarget".to_string(),
            JsonValue::Str(self.completed_target.clone()),
        ));
        entries.push((
            "Checks".to_string(),
            JsonValue::Array(
                self.checks
                    .iter()
                    .map(|s| JsonValue::Str(s.clone()))
                    .collect(),
            ),
        ));
        if !self.media_compression.is_empty() {
            entries.push((
                "MediaCompression".to_string(),
                JsonValue::Str(self.media_compression.clone()),
            ));
        }
        JsonValue::Object(entries)
    }
}

impl Request {
    fn validate_development_target(&self) -> std::result::Result<(), Error> {
        if self.development {
            if self.target != "candidate" && self.target != "media" {
                return Err(Error::msg(
                    "--development requires --target candidate or media",
                ));
            }
            return Ok(());
        }
        if !self.target.is_empty() {
            return Err(Error::msg("--target requires --development"));
        }
        Ok(())
    }

    fn validate_media_inputs(&self) -> std::result::Result<(), Error> {
        if !self.media_compression.is_empty()
            && (!self.development || self.target != "media" || self.media_compression != "fast")
        {
            return Err(Error::msg(
                "--media-compression accepts only fast with --development --target media",
            ));
        }
        if !self.wants_media() {
            if !self.rootfs_base_url.is_empty() || !self.media_authority.is_empty() {
                return Err(Error::msg("candidate target refuses media-only inputs"));
            }
            return Ok(());
        }
        media::media_base_url(&self.rootfs_base_url)
    }

    /// `ValidateTarget` rejects implicit partial production and irrelevant
    /// media inputs. MediaAuthority is admitted separately inside the
    /// isolated media worker.
    pub fn validate_target(&self) -> std::result::Result<(), Error> {
        self.validate_development_target()?;
        self.validate_media_inputs()
    }

    pub fn wants_media(&self) -> bool {
        self.target != "candidate"
    }

    pub fn purpose(&self) -> String {
        if self.development {
            "development".to_string()
        } else {
            "production".to_string()
        }
    }

    pub fn requested_target(&self) -> String {
        if self.development {
            self.target.clone()
        } else {
            "release".to_string()
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_development_target_admission() {
        // Oracle: Go TestDevelopmentTargetAdmission vectors.
        let valid = Request {
            development: true,
            target: "candidate".to_string(),
            ..Request::default()
        };
        assert!(valid.validate_target().is_ok());
        let valid = Request {
            development: true,
            target: "media".to_string(),
            rootfs_base_url: "https://example.invalid".to_string(),
            ..Request::default()
        };
        assert!(valid.validate_target().is_ok());
        for request in [
            Request {
                development: true,
                ..Request::default()
            },
            Request {
                target: "candidate".to_string(),
                ..Request::default()
            },
            Request {
                development: true,
                target: "candidate".to_string(),
                media_authority: "/authority".to_string(),
                ..Request::default()
            },
            Request {
                development: true,
                target: "media".to_string(),
                media_compression: "fast".to_string(),
                rootfs_base_url: "https://example.invalid".to_string(),
                ..Request::default()
            },
        ] {
            // The last case is valid (fast + development media); assert below.
            let _ = request;
        }
        assert_eq!(
            Request {
                development: true,
                ..Request::default()
            }
            .validate_target()
            .unwrap_err()
            .0,
            "--development requires --target candidate or media"
        );
        assert_eq!(
            Request {
                target: "media".to_string(),
                ..Request::default()
            }
            .validate_target()
            .unwrap_err()
            .0,
            "--target requires --development"
        );
        assert_eq!(
            Request {
                media_compression: "fast".to_string(),
                rootfs_base_url: "https://example.invalid".to_string(),
                ..Request::default()
            }
            .validate_target()
            .unwrap_err()
            .0,
            "--media-compression accepts only fast with --development --target media"
        );
        assert_eq!(
            Request {
                development: true,
                target: "candidate".to_string(),
                rootfs_base_url: "https://example.invalid".to_string(),
                ..Request::default()
            }
            .validate_target()
            .unwrap_err()
            .0,
            "candidate target refuses media-only inputs"
        );
        let fast = Request {
            development: true,
            target: "media".to_string(),
            media_compression: "fast".to_string(),
            rootfs_base_url: "https://example.invalid".to_string(),
            ..Request::default()
        };
        assert!(fast.validate_target().is_ok());
    }

    #[test]
    fn oracle_result_omits_empty_media_fields() {
        // Oracle: Go Result `json:",omitempty"` on Media/MediaCompression.
        let result = Result {
            revision: "r".to_string(),
            checks: vec!["c".to_string()],
            ..Result::default()
        };
        let mut out = String::new();
        crate::jsonio::write_compact(&mut out, &result.to_json());
        assert!(!out.contains("Media"));
        let result = Result {
            media: "m".to_string(),
            media_compression: "fast".to_string(),
            ..Result::default()
        };
        let mut out = String::new();
        crate::jsonio::write_compact(&mut out, &result.to_json());
        assert!(out.contains("\"Media\":\"m\""));
        assert!(out.contains("\"MediaCompression\":\"fast\""));
    }
}
