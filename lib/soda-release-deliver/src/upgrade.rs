//! Exact native upgrade qualification, separate from fresh-install admission.
use std::collections::BTreeMap;
use std::fs;
use std::io::Write;
use std::path::{Path, PathBuf};

use serde::{Deserialize, Serialize};

use crate::buildx::{hash_at, is_digest, Root};
use crate::model::{Candidate, MediaBinding};
use crate::payload::Payload;
use crate::Error;

pub const QUALIFICATION_FILE: &str = "upgrade-qualification.json";
const KINDS: [&str; 3] = ["transition", "interrupted_recovery", "stopped_project"];

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct EvidenceFile {
    pub sha256: String,
    pub bytes: u64,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Qualification {
    pub format: u32,
    pub source_release: String,
    pub source_support_sha256: String,
    pub target_payload_sha256: String,
    pub target_host_manifest: String,
    pub target_support_sha256: String,
    pub target_project_image: String,
    pub target_deployment_commit: String,
    pub rootfs_sha256: String,
    pub evidence: BTreeMap<String, EvidenceFile>,
}

/// Concrete readbacks emitted by the native journey, never an operator's
/// unchecked "passed" assertion. Each fixed evidence file covers one boundary.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct NativeEvidence {
    pub format: u32,
    pub kind: String,
    pub native_architecture: String,
    pub source_release: String,
    pub source_support_sha256: String,
    pub target_payload_sha256: String,
    pub target_host_manifest: String,
    pub target_support_sha256: String,
    pub target_project_image: String,
    pub target_deployment_commit: String,
    pub rootfs_sha256: String,
    pub source_deployment: String,
    pub observed_deployment: String,
    pub staged_deployment: String,
    pub staged_finalization_locked: bool,
    pub admission: String,
    pub container_before: String,
    pub container_after: String,
    pub protected_state_before_sha256: String,
    pub protected_state_after_sha256: String,
    pub observed_support_sha256: String,
    pub project_state_before: String,
    pub project_state_after: String,
    pub interruption_checkpoint: String,
}

fn refused() -> Error {
    Error::msg("native upgrade qualification does not bind this exact transition")
}

impl Qualification {
    pub fn validate(
        &self,
        payload: &Payload,
        candidate: &Candidate,
        media: &MediaBinding,
        source_release: &str,
        source_support_sha256: &str,
    ) -> Result<(), Error> {
        payload.validate().map_err(|e| Error::msg(e.to_string()))?;
        let sources = payload.upgrade_from.as_slice();
        if self.format != 1
            || sources != [source_release.to_owned()]
            || source_release == payload.id
            || self.source_release != source_release
            || self.source_support_sha256 != source_support_sha256
            || !is_digest(&self.source_support_sha256)
            || self.target_payload_sha256 != candidate.payload_sha256
            || self.target_host_manifest != candidate.host.manifest
            || self.target_support_sha256 != payload.project_support.sha256
            || self.target_project_image != payload.project_support.project_image
            || !is_digest(&self.target_deployment_commit)
            || self.target_deployment_commit != media.deployment_commit
            || self.rootfs_sha256 != media.rootfs.sha256
            || !crate::model::valid_media_binding(media, payload, candidate)
            || self.evidence.len() != KINDS.len()
            || KINDS.iter().any(|kind| {
                !self.evidence.get(*kind).is_some_and(|file| {
                    is_digest(&file.sha256) && file.bytes > 0 && file.bytes <= 1 << 20
                })
            })
        {
            return Err(refused());
        }
        Ok(())
    }
}

impl NativeEvidence {
    fn validate(&self, kind: &str, receipt: &Qualification) -> Result<(), Error> {
        self.validate_receipt_identity(kind)?;
        self.validate_receipt_binding(receipt)?;
        self.validate_deployment_admission()?;
        self.validate_protected_observations()?;
        self.validate_kind_specific(kind)?;
        Ok(())
    }

    fn validate_receipt_identity(&self, kind: &str) -> Result<(), Error> {
        if self.format != 1 || self.kind != kind || self.native_architecture != "x86_64" {
            return Err(refused());
        }
        Ok(())
    }

    fn validate_receipt_binding(&self, receipt: &Qualification) -> Result<(), Error> {
        if self.source_release != receipt.source_release
            || self.source_support_sha256 != receipt.source_support_sha256
            || self.target_payload_sha256 != receipt.target_payload_sha256
            || self.target_host_manifest != receipt.target_host_manifest
            || self.target_support_sha256 != receipt.target_support_sha256
            || self.target_project_image != receipt.target_project_image
            || self.target_deployment_commit != receipt.target_deployment_commit
            || self.rootfs_sha256 != receipt.rootfs_sha256
        {
            return Err(refused());
        }
        Ok(())
    }

    fn validate_deployment_admission(&self) -> Result<(), Error> {
        if !is_digest(&self.source_deployment)
            || self.source_deployment == self.target_deployment_commit
            || self.observed_deployment != self.target_deployment_commit
            || self.staged_deployment != self.target_deployment_commit
            || !self.staged_finalization_locked
            || self.admission != "closed_until_verified"
        {
            return Err(refused());
        }
        Ok(())
    }

    fn validate_protected_observations(&self) -> Result<(), Error> {
        if !is_digest(&self.container_before)
            || self.container_before != self.container_after
            || !is_digest(&self.protected_state_before_sha256)
            || self.protected_state_before_sha256 != self.protected_state_after_sha256
            || self.observed_support_sha256 != self.target_support_sha256
            || self.project_state_before != self.project_state_after
            || !matches!(self.project_state_before.as_str(), "running" | "stopped")
        {
            return Err(refused());
        }
        Ok(())
    }

    fn validate_kind_specific(&self, kind: &str) -> Result<(), Error> {
        if (kind == "stopped_project" && self.project_state_before != "stopped")
            || (kind == "interrupted_recovery"
                && self.interruption_checkpoint != "support_verified_host_locked")
        {
            return Err(refused());
        }
        Ok(())
    }
}

pub fn load(path: &Path) -> Result<Qualification, Error> {
    let body = crate::document::read_file(path.to_str().ok_or_else(refused)?, 1 << 20)?;
    crate::json_serde::strict(&body).map_err(|_| refused())
}

fn target(directory: &Path) -> Result<(Payload, Candidate, MediaBinding), Error> {
    let directory_text = directory.to_str().ok_or_else(refused)?;
    let (payload, candidate, _) = crate::admission::load_admitted_candidate(directory_text)?;
    let media_path = if directory.join("media.json").is_file() {
        directory.join("media.json")
    } else {
        directory.join("media/media.json")
    };
    let body = crate::document::read_file(media_path.to_str().ok_or_else(refused)?, 1 << 20)?;
    let media: MediaBinding = serde_json::from_slice(&body).map_err(|_| refused())?;
    if !crate::model::valid_media_binding(&media, &payload, &candidate)
        || !is_digest(&media.deployment_commit)
    {
        return Err(refused());
    }
    Ok((payload, candidate, media))
}

/// Stamp only after all fixed native evidence files exist and their concrete
/// observations agree. The receipt does not alter any shipping host bytes.
pub fn stamp(
    candidate_directory: &Path,
    evidence_directory: &Path,
) -> Result<Qualification, Error> {
    let AdmittedCandidate {
        payload,
        candidate,
        media,
        ..
    } = admit_candidate_directory(candidate_directory)?;
    let evidence_root = Root::open(evidence_directory.to_str().ok_or_else(refused)?)?;
    let first = crate::buildx::read_at(&evidence_root, "transition.json", 1 << 20)?;
    let first: NativeEvidence = crate::json_serde::strict(&first).map_err(|_| refused())?;
    let mut receipt = Qualification {
        format: 1,
        source_release: first.source_release.clone(),
        source_support_sha256: first.source_support_sha256.clone(),
        target_payload_sha256: candidate.payload_sha256.clone(),
        target_host_manifest: candidate.host.manifest.clone(),
        target_support_sha256: payload.project_support.sha256.clone(),
        target_project_image: payload.project_support.project_image.clone(),
        target_deployment_commit: media.deployment_commit.clone(),
        rootfs_sha256: media.rootfs.sha256.clone(),
        evidence: BTreeMap::new(),
    };
    for kind in KINDS {
        let body = crate::buildx::read_at(&evidence_root, &format!("{kind}.json"), 1 << 20)?;
        let observed: NativeEvidence = crate::json_serde::strict(&body).map_err(|_| refused())?;
        observed.validate(kind, &receipt)?;
        receipt.evidence.insert(
            kind.to_owned(),
            EvidenceFile {
                sha256: crate::hash_bytes(&body)
                    .trim_start_matches("sha256:")
                    .to_owned(),
                bytes: body.len() as u64,
            },
        );
    }
    receipt.validate(
        &payload,
        &candidate,
        &media,
        &first.source_release,
        &first.source_support_sha256,
    )?;
    let bytes = serde_json::to_vec_pretty(&receipt).map_err(|_| refused())?;
    let mut output = fs::OpenOptions::new()
        .create_new(true)
        .write(true)
        .open(candidate_directory.join(QUALIFICATION_FILE))
        .map_err(|_| Error::msg("new upgrade qualification output required"))?;
    output
        .write_all(&bytes)
        .and_then(|_| output.write_all(b"\n"))
        .and_then(|_| output.sync_all())
        .map_err(|_| Error::msg("cannot write upgrade qualification"))?;
    Ok(receipt)
}

pub struct AdmittedUpgrade {
    pub payload: Payload,
    pub candidate: Candidate,
    pub media: MediaBinding,
    pub qualification: Qualification,
    pub rootfs: PathBuf,
}

pub struct AdmittedCandidate {
    pub payload: Payload,
    pub candidate: Candidate,
    pub media: MediaBinding,
    pub rootfs: PathBuf,
}

/// Development qualification uses exact target bytes before a receipt exists.
/// This read-only admission makes no supported-upgrade assertion.
pub fn admit_candidate_directory(directory: &Path) -> Result<AdmittedCandidate, Error> {
    let (payload, candidate, media) = target(directory)?;
    let expected = format!("{}-rootfs.img", media.rootfs.sha256);
    if media.rootfs.path != expected {
        return Err(refused());
    }
    let asset_directory = if directory.join("media.json").is_file() {
        directory.to_path_buf()
    } else {
        directory.join("media")
    };
    let root = Root::open(asset_directory.to_str().ok_or_else(refused)?)?;
    let rootfs = asset_directory.join(expected);
    if media.rootfs.bytes as u64 >= crate::publish::assets::MAX_ASSET_BYTES
        || fs::symlink_metadata(&rootfs).map_err(|_| refused())?.len() != media.rootfs.bytes as u64
    {
        return Err(refused());
    }
    if hash_at(&root, &media.rootfs.path)? != media.rootfs.sha256 {
        return Err(Error::msg(
            "downloaded upgrade rootfs hash differs from release",
        ));
    }
    Ok(AdmittedCandidate {
        payload,
        candidate,
        media,
        rootfs,
    })
}

/// Admit one complete downloaded GitHub release. Authentication of the download
/// remains with GitHub's existing attestation boundary; this verifies its content.
pub fn admit_directory(
    directory: &Path,
    source_release: &str,
    source_support_sha256: &str,
) -> Result<AdmittedUpgrade, Error> {
    let admitted = admit_candidate_directory(directory)?;
    let AdmittedCandidate {
        payload,
        candidate,
        media,
        rootfs,
    } = admitted;
    let qualification = load(&directory.join(QUALIFICATION_FILE))?;
    qualification.validate(
        &payload,
        &candidate,
        &media,
        source_release,
        source_support_sha256,
    )?;
    Ok(AdmittedUpgrade {
        payload,
        candidate,
        media,
        qualification,
        rootfs,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    fn fixture() -> tempfile::TempDir {
        let dir = tempfile::TempDir::new().unwrap();
        let golden: serde_json::Value =
            serde_json::from_str(include_str!("../tests/goldens/deliver.json")).unwrap();
        let mut payload: Payload =
            serde_json::from_str(golden["payload"].as_str().unwrap()).unwrap();
        payload.upgrade_from = vec!["43.20260101.3.0.soda-cccccccccccc".into()];
        let bytes = serde_json::to_vec_pretty(&payload).unwrap();
        let mut candidate: Candidate =
            serde_json::from_str(golden["candidate"].as_str().unwrap()).unwrap();
        candidate.payload_sha256 = crate::hash_bytes(&bytes)
            .trim_start_matches("sha256:")
            .into();
        fs::write(dir.path().join("payload.json"), &bytes).unwrap();
        fs::write(
            dir.path().join("candidate.json"),
            serde_json::to_vec_pretty(&candidate).unwrap(),
        )
        .unwrap();
        let rootfs = b"complete-native-fixture-rootfs";
        let digest = crate::hash_bytes(rootfs)
            .trim_start_matches("sha256:")
            .to_owned();
        let media = MediaBinding {
            revision: payload.revision.clone(),
            architecture: "x86_64".into(),
            host_manifest: candidate.host.manifest.clone(),
            payload_sha256: candidate.payload_sha256.clone(),
            deployment_commit: "9".repeat(64),
            rootfs_url: format!("https://github.com/LevitateOS/sodaos/releases/download/fixture/{digest}-rootfs.img"),
            iso: crate::model::MediaFile { path: "installer.iso".into(), sha256: "1".repeat(64), bytes: 3 },
            rootfs: crate::model::MediaFile { path: format!("{digest}-rootfs.img"), sha256: digest, bytes: rootfs.len() as i64 },
        };
        fs::write(dir.path().join(&media.rootfs.path), rootfs).unwrap();
        fs::write(
            dir.path().join("media.json"),
            serde_json::to_vec_pretty(&media).unwrap(),
        )
        .unwrap();
        fs::create_dir(dir.path().join("evidence")).unwrap();
        for kind in KINDS {
            let evidence = NativeEvidence {
                format: 1,
                kind: kind.into(),
                native_architecture: "x86_64".into(),
                source_release: payload.upgrade_from.first().cloned().unwrap_or_default(),
                source_support_sha256: "2".repeat(64),
                target_payload_sha256: candidate.payload_sha256.clone(),
                target_host_manifest: candidate.host.manifest.clone(),
                target_support_sha256: payload.project_support.sha256.clone(),
                target_project_image: payload.project_support.project_image.clone(),
                target_deployment_commit: media.deployment_commit.clone(),
                rootfs_sha256: media.rootfs.sha256.clone(),
                source_deployment: "3".repeat(64),
                observed_deployment: media.deployment_commit.clone(),
                staged_deployment: media.deployment_commit.clone(),
                staged_finalization_locked: true,
                admission: "closed_until_verified".into(),
                container_before: "4".repeat(64),
                container_after: "4".repeat(64),
                protected_state_before_sha256: "5".repeat(64),
                protected_state_after_sha256: "5".repeat(64),
                observed_support_sha256: payload.project_support.sha256.clone(),
                project_state_before: if kind == "stopped_project" {
                    "stopped"
                } else {
                    "running"
                }
                .into(),
                project_state_after: if kind == "stopped_project" {
                    "stopped"
                } else {
                    "running"
                }
                .into(),
                interruption_checkpoint: if kind == "interrupted_recovery" {
                    "support_verified_host_locked"
                } else {
                    ""
                }
                .into(),
            };
            fs::write(
                dir.path().join("evidence").join(format!("{kind}.json")),
                serde_json::to_vec_pretty(&evidence).unwrap(),
            )
            .unwrap();
        }
        dir
    }

    #[test]
    fn prospective_sources_require_native_receipt_for_upgrade_admission() {
        let dir = fixture();
        let target = admit_candidate_directory(dir.path()).unwrap();
        let source = target
            .payload
            .upgrade_from
            .first()
            .cloned()
            .unwrap_or_default();
        let helper = "2".repeat(64);
        assert!(admit_directory(dir.path(), &source, &helper).is_err());
        let receipt = stamp(dir.path(), &dir.path().join("evidence")).unwrap();
        let admitted = admit_directory(dir.path(), &source, &helper).unwrap();
        assert_eq!(admitted.qualification, receipt);
        assert!(admit_directory(dir.path(), &source, &"6".repeat(64)).is_err());
        fs::write(&admitted.rootfs, b"changed-public-rootfs").unwrap();
        assert!(admit_directory(dir.path(), &source, &helper).is_err());
    }

    #[test]
    fn matching_support_records_verification_without_claiming_replacement() {
        let dir = fixture();
        for kind in KINDS {
            let path = dir.path().join("evidence").join(format!("{kind}.json"));
            let mut evidence: NativeEvidence =
                serde_json::from_slice(&fs::read(&path).unwrap()).unwrap();
            evidence.source_support_sha256 = evidence.target_support_sha256.clone();
            fs::write(path, serde_json::to_vec_pretty(&evidence).unwrap()).unwrap();
        }
        let receipt = stamp(dir.path(), &dir.path().join("evidence")).unwrap();
        assert_eq!(receipt.source_support_sha256, receipt.target_support_sha256);
        admit_directory(
            dir.path(),
            &receipt.source_release,
            &receipt.source_support_sha256,
        )
        .unwrap();
    }

    #[test]
    fn stamping_requires_real_bound_observations_for_every_native_boundary() {
        for change in [
            "missing_recovery",
            "same_commit",
            "container_changed",
            "state_changed",
            "unlocked",
            "wrong_helper",
            "stopped_started",
            "wrong_checkpoint",
        ] {
            let dir = fixture();
            let kind = if change == "stopped_started" {
                "stopped_project"
            } else {
                "interrupted_recovery"
            };
            let path = dir.path().join("evidence").join(format!("{kind}.json"));
            if change == "missing_recovery" {
                fs::remove_file(path).unwrap();
            } else {
                let mut evidence: NativeEvidence =
                    serde_json::from_slice(&fs::read(&path).unwrap()).unwrap();
                match change {
                    "same_commit" => {
                        evidence.source_deployment = evidence.target_deployment_commit.clone()
                    }
                    "container_changed" => evidence.container_after = "6".repeat(64),
                    "state_changed" => evidence.protected_state_after_sha256 = "6".repeat(64),
                    "unlocked" => evidence.staged_finalization_locked = false,
                    "wrong_helper" => evidence.observed_support_sha256 = "6".repeat(64),
                    "stopped_started" => evidence.project_state_after = "running".into(),
                    "wrong_checkpoint" => evidence.interruption_checkpoint = "host_booted".into(),
                    _ => unreachable!(),
                }
                fs::write(path, serde_json::to_vec_pretty(&evidence).unwrap()).unwrap();
            }
            assert!(
                stamp(dir.path(), &dir.path().join("evidence")).is_err(),
                "{change}"
            );
            assert!(!dir.path().join(QUALIFICATION_FILE).exists(), "{change}");
        }
    }
}
