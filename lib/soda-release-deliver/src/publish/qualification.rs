//! Docker qualification consumes finalized native observations, never a pass flag.
use super::{assets, Manifest};
use crate::{
    buildx::{is_digest, read_at, Root},
    model::FileIdentity,
    Error,
};
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    io::Write,
    path::{Path, PathBuf},
};

const KINDS: [&str; 4] = ["installation", "browser_project", "factory", "persistence"];
pub(super) const DOWNLOADS: [&str; 5] = [
    "docker.json",
    "sodaos-docker-compose.yaml",
    super::installer::NAME,
    "candidate.json",
    "payload.json",
];
#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct RetainedQualification {
    qualification: DockerQualification,
    evidence_directory: PathBuf,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct NativeDocker {
    pub target: String,
    pub tag: String,
    pub registry_manifest: String,
    pub registry_config: String,
    pub compose_sha256: String,
    pub installer_sha256: String,
    pub installer_execution_sha256: String,
    pub candidate_sha256: String,
    pub payload_sha256: String,
    pub soda_revision: String,
    pub forgejo_revision: String,
    pub application_configs: BTreeMap<String, String>,
    pub os: String,
    pub architecture: String,
    pub rootful: bool,
    pub cgroup_version: u32,
    pub host_network: bool,
    pub fresh_volumes: bool,
    pub outer_id: String,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct NativePersistence {
    pub target: String,
    pub registry_manifest: String,
    pub registry_config: String,
    pub outer_before: String,
    pub outer_after: String,
    pub project_before: String,
    pub project_after: String,
    pub protected_before_sha256: String,
    pub protected_after_sha256: String,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DockerQualification {
    pub format: u32,
    pub native: NativeDocker,
    pub factory: NativeFactory,
    pub persistence: NativePersistence,
    pub observations: BTreeMap<String, FileIdentity>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct NativeFactory {
    pub provider_execution_sha256: String,
    pub repository_id: u64,
    pub issue_id: u64,
    pub pull_request_id: u64,
    pub coder_actor: u64,
    pub reviewer_actor: u64,
    pub reviewed_head: String,
    pub merged_candidate_head: String,
    pub required_checks: BTreeMap<String, String>,
    pub elapsed_ms: u64,
    pub funding_limit_ms: u64,
}
fn refused() -> Error {
    Error::msg("Docker qualification requires matching finalized native installation, browser/Project, funded factory and persistence evidence")
}
fn read_native(directory: &Path) -> Result<NativeDocker, Error> {
    let root = Root::open(directory.to_str().ok_or_else(refused)?)?;
    let bytes = read_at(&root, "installation/native.json", 1 << 20)?;
    crate::json_serde::strict(&bytes).map_err(|_| refused())
}
impl DockerQualification {
    fn validate(&self, manifest: &Manifest, directory: &Path) -> Result<(), Error> {
        let docker = manifest.docker.as_ref().ok_or_else(refused)?;
        let native = &self.native;
        let factory = &self.factory;
        let persistence = &self.persistence;
        let payload: crate::payload::Payload = assets::read_json(&directory.join("payload.json"))?;
        let configs = soda_build_tools::payload::Role::iter()
            .map(|role| {
                (
                    role.name().to_string(),
                    role.binding(&payload.images).config.clone(),
                )
            })
            .collect::<BTreeMap<_, _>>();
        if self.format != 1
            || native.tag != manifest.tag
            || native.registry_manifest != docker.image.manifest
            || native.registry_config != docker.image.config
            || native.compose_sha256 != docker.compose.sha256
            || native.installer_sha256 != manifest.asset(super::installer::NAME)?.sha256
            || !is_digest(&native.installer_execution_sha256)
            || native.candidate_sha256 != docker.candidate_sha256
            || native.payload_sha256 != docker.payload_sha256
            || native.soda_revision != manifest.soda_revision
            || native.forgejo_revision != manifest.forgejo_revision
            || native.application_configs != configs
            || native.target.is_empty()
            || native.os != "linux"
            || native.architecture != "x86_64"
            || !native.rootful
            || native.cgroup_version != 2
            || !native.host_network
            || !native.fresh_volumes
            || !is_digest(&native.outer_id)
            || persistence.target != native.target
            || persistence.registry_manifest != native.registry_manifest
            || persistence.registry_config != native.registry_config
            || persistence.outer_before != native.outer_id
            || !is_digest(&persistence.outer_after)
            || persistence.outer_before == persistence.outer_after
            || !is_digest(&persistence.project_before)
            || persistence.project_before != persistence.project_after
            || !is_digest(&persistence.protected_before_sha256)
            || persistence.protected_before_sha256 != persistence.protected_after_sha256
            || self.observations.len() != KINDS.len()
            || !is_digest(&factory.provider_execution_sha256)
            || factory.repository_id == 0
            || factory.issue_id == 0
            || factory.pull_request_id == 0
            || factory.coder_actor == 0
            || factory.reviewer_actor == 0
            || factory.coder_actor == factory.reviewer_actor
            || !crate::buildx::is_revision(&factory.reviewed_head)
            || factory.reviewed_head != factory.merged_candidate_head
            || factory.required_checks.is_empty()
            || factory
                .required_checks
                .values()
                .any(|value| value != "success")
            || factory.elapsed_ms == 0
            || factory.funding_limit_ms == 0
            || factory.elapsed_ms > factory.funding_limit_ms
            || KINDS
                .iter()
                .any(|kind| !self.observations.contains_key(*kind))
        {
            return Err(refused());
        }
        Ok(())
    }
}
fn collect(
    manifest: &Manifest,
    release: &Path,
    evidence: &Path,
) -> Result<DockerQualification, Error> {
    let native = read_native(evidence)?;
    let root = Root::open(evidence.to_str().ok_or_else(refused)?)?;
    let factory: NativeFactory =
        crate::json_serde::strict(&read_at(&root, "factory/native.json", 1 << 20)?)
            .map_err(|_| refused())?;
    let persistence: NativePersistence =
        crate::json_serde::strict(&read_at(&root, "persistence/native.json", 1 << 20)?)
            .map_err(|_| refused())?;
    for name in DOWNLOADS {
        if &assets::inspect(&evidence.join("downloads").join(name), name)?
            != manifest.asset(name)?
        {
            return Err(Error::msg("downloaded qualification bytes changed"));
        }
    }
    let mut observations = BTreeMap::new();
    let mut previous_finish = None;
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map_err(|_| refused())?
        .as_nanos() as i128;
    for kind in KINDS {
        let file = evidence.join(kind).join("observation.json");
        // The general observation reader permits duplicate keys for its historical
        // log contract. Qualification admits one unambiguous current record.
        let _: serde_json::Value = assets::read_json(&file)?;
        let (observation, hash) =
            soda_acceptance::report::read_observation(file.to_str().ok_or_else(refused)?)
                .map_err(|_| refused())?;
        if observation.outcome != "completed"
            || observation.execution != "completed"
            || observation.evidence != "completed"
            || observation.exit_code != Some(0)
            || observation.tool_dirty
            || !crate::buildx::is_revision(&observation.tool_revision)
            || observation.requested_revision != manifest.soda_revision
            || observation.requested_architecture != "x86_64"
            || !soda_acceptance::report::valid_owner(&observation.owner)
            || observation.target != native.target
            || observation.client_platform != "linux/amd64"
            || observation.action != "exec"
            || observation
                .invocation
                .as_ref()
                .is_none_or(|args| args.is_empty() || args[0].is_empty())
            || observation.files.as_ref().is_none_or(BTreeMap::is_empty)
        {
            return Err(refused());
        }
        let started = soda_wire_time::parse_nanos(&observation.started).ok_or_else(refused)?;
        let finished = soda_wire_time::parse_nanos(&observation.finished).ok_or_else(refused)?;
        if started >= finished
            || finished > now
            || previous_finish.is_some_and(|time| started < time)
        {
            return Err(Error::msg("native qualification observations must be finalized in installation, browser/Project, factory and persistence order"));
        }
        previous_finish = Some(finished);
        for name in DOWNLOADS {
            let expected = &manifest.asset(name)?.sha256;
            if !observation.artifacts.as_ref().is_some_and(|artifacts| {
                artifacts.iter().any(|(path, hash)| {
                    Path::new(path).file_name().and_then(|value| value.to_str()) == Some(name)
                        && hash == expected
                })
            }) {
                return Err(refused());
            }
        }
        if matches!(kind, "installation" | "factory" | "persistence") {
            let bytes = crate::document::read_file(
                evidence
                    .join(format!("{kind}/native.json"))
                    .to_str()
                    .ok_or_else(refused)?,
                1 << 20,
            )?;
            if observation
                .files
                .as_ref()
                .and_then(|files| files.get("native.json"))
                != Some(&crate::hash_bytes(&bytes)[7..].to_owned())
            {
                return Err(refused());
            }
        }
        if kind == "factory"
            && !observation.files.as_ref().is_some_and(|files| {
                files.iter().any(|(path, hash)| {
                    path != "native.json" && hash == &factory.provider_execution_sha256
                })
            })
        {
            return Err(Error::msg(
                "factory execution digest must identify retained execution evidence",
            ));
        }
        if kind == "installation"
            && !observation.files.as_ref().is_some_and(|files| {
                files.iter().any(|(path, hash)| {
                    path != "native.json" && hash == &native.installer_execution_sha256
                })
            })
        {
            return Err(Error::msg(
                "installer execution digest must identify retained execution evidence",
            ));
        }
        observations.insert(
            kind.into(),
            FileIdentity {
                sha256: hash,
                bytes: std::fs::metadata(&file).map_err(|_| refused())?.len(),
            },
        );
    }
    let receipt = DockerQualification {
        format: 1,
        native,
        factory,
        persistence,
        observations,
    };
    receipt.validate(manifest, release)?;
    Ok(receipt)
}
pub fn stamp(
    manifest: &Manifest,
    release: &Path,
    evidence: &Path,
) -> Result<DockerQualification, Error> {
    let receipt = collect(manifest, release, evidence)?;
    let destination = release.join("docker-qualification.json");
    if destination.exists() {
        if verify(manifest, release)? != receipt {
            return Err(refused());
        }
        return Ok(receipt);
    }
    // Receipt and private locator commit together. An interrupted write cannot
    // leave a success receipt without its original evidence location.
    let retained = RetainedQualification {
        qualification: receipt.clone(),
        evidence_directory: std::fs::canonicalize(evidence).map_err(|_| refused())?,
    };
    let bytes = serde_json::to_vec_pretty(&retained).map_err(|_| refused())?;
    let mut file = tempfile::NamedTempFile::new_in(release).map_err(|_| refused())?;
    file.write_all(&bytes).map_err(|_| refused())?;
    file.as_file().sync_all().map_err(|_| refused())?;
    file.persist_noclobber(&destination)
        .map_err(|_| refused())?;
    Ok(receipt)
}
pub(super) fn verify(manifest: &Manifest, release: &Path) -> Result<DockerQualification, Error> {
    let expected: RetainedQualification =
        assets::read_json(&release.join("docker-qualification.json"))?;
    let observed = collect(manifest, release, &expected.evidence_directory)?;
    if expected.qualification != observed {
        return Err(refused());
    }
    Ok(observed)
}

pub(super) fn evidence(release: &Path) -> Result<PathBuf, Error> {
    let retained: RetainedQualification =
        assets::read_json(&release.join("docker-qualification.json"))?;
    Ok(retained.evidence_directory)
}

pub(super) fn validate_publication_time(
    manifest: &Manifest,
    release: &Path,
    evidence: &Path,
    published_at: &str,
) -> Result<(), Error> {
    collect(manifest, release, evidence)?;
    let published = soda_wire_time::parse_nanos(published_at).ok_or_else(refused)?;
    let file = evidence.join("installation/observation.json");
    let (observation, _) =
        soda_acceptance::report::read_observation(file.to_str().ok_or_else(refused)?)
            .map_err(|_| refused())?;
    let started = soda_wire_time::parse_nanos(&observation.started).ok_or_else(refused)?;
    if started < published {
        return Err(Error::msg(
            "Docker qualification must start after immutable release publication",
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::super::{assets, Manifest};
    use super::*;
    use crate::model::{Distribution, DockerMetadata, FileIdentity};
    use soda_acceptance::report::{observation_json, Observation};
    use std::collections::BTreeMap;

    const KIND_ORDER: [&str; 4] = KINDS;
    const REVISIONS: (&str, &str) = (
        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    );

    struct Staged {
        _evidence: tempfile::TempDir,
        _release: tempfile::TempDir,
        manifest: Manifest,
    }

    impl Staged {
        fn evidence(&self) -> &Path {
            self._evidence.path()
        }
        fn release(&self) -> &Path {
            self._release.path()
        }
    }

    fn file_identity(digest: &str, bytes: u64) -> FileIdentity {
        FileIdentity {
            sha256: digest.to_string(),
            bytes,
        }
    }

    fn observation(kind_index: usize, files: BTreeMap<String, String>) -> Observation {
        let mut artifacts = BTreeMap::new();
        for asset in Staged::download_assets() {
            artifacts.insert(asset.0, asset.1);
        }
        Observation {
            owner: "P06".to_string(),
            requested_revision: REVISIONS.0.to_string(),
            tool_revision: "d".repeat(40),
            requested_architecture: "x86_64".to_string(),
            target: "synthetic-qual".to_string(),
            client_platform: "linux/amd64".to_string(),
            action: "exec".to_string(),
            outcome: "completed".to_string(),
            execution: "completed".to_string(),
            evidence: "completed".to_string(),
            cleanup: "synthetic".to_string(),
            topology: "synthetic".to_string(),
            exit_code: Some(0),
            tool_dirty: false,
            invocation: Some(vec!["/bin/true".to_string()]),
            files: Some(files),
            artifacts: Some(artifacts),
            started: format!("2026-10-04T00:{:02}:00Z", kind_index * 10),
            finished: format!("2026-10-04T00:{:02}:05Z", kind_index * 10),
            ..Default::default()
        }
    }

    impl Staged {
        fn download_assets() -> Vec<(String, String)> {
            let mut out = Vec::new();
            for name in DOWNLOADS {
                let digest = crate::hash_bytes(format!("qualification-{name}\n").as_bytes());
                out.push((name.to_string(), digest[7..].to_string()));
            }
            out
        }

        fn stage() -> Self {
            let evidence =
                tempfile::TempDir::new().expect("isolated evidence directory must stage");
            let release = tempfile::TempDir::new().expect("isolated release directory must stage");
            let downloads = evidence.path().join("downloads");
            std::fs::create_dir_all(&downloads).unwrap();
            let mut manifest_assets = Vec::new();
            for (name, _) in Self::download_assets() {
                let bytes = format!("qualification-{name}\n");
                std::fs::write(downloads.join(&name), &bytes).unwrap();
                manifest_assets.push(assets::inspect(&downloads.join(&name), &name).unwrap());
            }
            let asset = |assets: &[crate::publish::assets::Asset], name: &str| {
                assets
                    .iter()
                    .find(|entry| entry.name == name)
                    .unwrap()
                    .clone()
            };
            let compose_asset = asset(&manifest_assets, "sodaos-docker-compose.yaml");
            let installer_asset = asset(&manifest_assets, super::super::installer::NAME);
            let fedora = "registry.fedoraproject.org/fedora@sha256:".to_string() + &"c".repeat(64);
            let image = crate::buildx::Image {
                manifest: format!("sha256:{}", "b".repeat(64)),
                config: format!("sha256:{}", "c".repeat(64)),
                architecture: "amd64".to_string(),
                revision: REVISIONS.0.to_string(),
                source: "https://github.com/LevitateOS/sodaos".to_string(),
                base_name: fedora.clone(),
                base_digest: "c".repeat(64),
            };
            let manifest = Manifest {
                format: 1,
                tag: "v0.1.0".to_string(),
                distribution: Distribution::Docker,
                docker: Some(DockerMetadata {
                    format: 1,
                    distribution: Distribution::Docker,
                    purpose: crate::model::BuildPurpose::Production,
                    release_tag: "v0.1.0".to_string(),
                    architecture: "x86_64".to_string(),
                    platform: "linux/amd64".to_string(),
                    soda_revision: REVISIONS.0.to_string(),
                    forgejo_revision: REVISIONS.1.to_string(),
                    soda_source: file_identity(&"d".repeat(64), 7),
                    forgejo_source: file_identity(&"d".repeat(64), 7),
                    candidate_sha256: "e".repeat(64),
                    payload_sha256: "f".repeat(64),
                    host_manifest: format!("sha256:{}", "b".repeat(64)),
                    host_archive_sha256: "a".repeat(64),
                    fedora_base: fedora,
                    recipe: file_identity(&"d".repeat(64), 7),
                    image,
                    image_archive: file_identity(&"d".repeat(64), 7),
                    packages: file_identity(&"d".repeat(64), 7),
                    compose: file_identity(&compose_asset.sha256, compose_asset.bytes),
                }),
                soda_revision: REVISIONS.0.to_string(),
                forgejo_revision: REVISIONS.1.to_string(),
                iso: assets::ISO_NAME.to_string(),
                rootfs: "synthetic-rootfs.img".to_string(),
                notes: "development proof fixture, never production evidence".to_string(),
                assets: manifest_assets,
            };
            let (_, payload, _) = crate::model::Candidate::fixture_pair();
            let configs = soda_build_tools::payload::Role::iter()
                .map(|role| {
                    (
                        role.name().to_string(),
                        role.binding(&payload.images).config.clone(),
                    )
                })
                .collect::<BTreeMap<_, _>>();
            let installer_log = b"synthetic installer execution log\n";
            let provider_log = b"synthetic provider execution log\n";
            let probe_log = b"synthetic probe log\n";
            let log_hash = |bytes: &[u8]| crate::hash_bytes(bytes)[7..].to_string();
            let native = NativeDocker {
                target: "synthetic-qual".to_string(),
                tag: manifest.tag.clone(),
                registry_manifest: manifest.docker.as_ref().unwrap().image.manifest.clone(),
                registry_config: manifest.docker.as_ref().unwrap().image.config.clone(),
                compose_sha256: compose_asset.sha256.clone(),
                installer_sha256: installer_asset.sha256.clone(),
                installer_execution_sha256: log_hash(installer_log),
                candidate_sha256: manifest.docker.as_ref().unwrap().candidate_sha256.clone(),
                payload_sha256: manifest.docker.as_ref().unwrap().payload_sha256.clone(),
                soda_revision: REVISIONS.0.to_string(),
                forgejo_revision: REVISIONS.1.to_string(),
                application_configs: configs,
                os: "linux".to_string(),
                architecture: "x86_64".to_string(),
                rootful: true,
                cgroup_version: 2,
                host_network: true,
                fresh_volumes: true,
                outer_id: "2".repeat(64),
            };
            let factory = NativeFactory {
                provider_execution_sha256: log_hash(provider_log),
                repository_id: 7,
                issue_id: 8,
                pull_request_id: 9,
                coder_actor: 10,
                reviewer_actor: 11,
                reviewed_head: "c".repeat(40),
                merged_candidate_head: "c".repeat(40),
                required_checks: BTreeMap::from([("smoke".to_string(), "success".to_string())]),
                elapsed_ms: 1000,
                funding_limit_ms: 60000,
            };
            let persistence = NativePersistence {
                target: native.target.clone(),
                registry_manifest: native.registry_manifest.clone(),
                registry_config: native.registry_config.clone(),
                outer_before: native.outer_id.clone(),
                outer_after: "4".repeat(64),
                project_before: "5".repeat(64),
                project_after: "5".repeat(64),
                protected_before_sha256: "6".repeat(64),
                protected_after_sha256: "6".repeat(64),
            };
            let native_bytes = serde_json::to_vec(&native).unwrap();
            let factory_bytes = serde_json::to_vec(&factory).unwrap();
            let persistence_bytes = serde_json::to_vec(&persistence).unwrap();
            let hash_of = |bytes: &[u8]| crate::hash_bytes(bytes)[7..].to_string();
            for kind in KIND_ORDER {
                std::fs::create_dir_all(evidence.path().join(kind)).unwrap();
            }
            std::fs::write(
                evidence.path().join("installation/native.json"),
                &native_bytes,
            )
            .unwrap();
            std::fs::write(evidence.path().join("factory/native.json"), &factory_bytes).unwrap();
            std::fs::write(
                evidence.path().join("persistence/native.json"),
                &persistence_bytes,
            )
            .unwrap();
            for (index, kind) in KIND_ORDER.iter().enumerate() {
                let mut files = BTreeMap::new();
                if *kind == "installation" {
                    files.insert("native.json".to_string(), hash_of(&native_bytes));
                } else if *kind == "factory" {
                    files.insert("native.json".to_string(), hash_of(&factory_bytes));
                } else if *kind == "persistence" {
                    files.insert("native.json".to_string(), hash_of(&persistence_bytes));
                } else {
                    std::fs::write(evidence.path().join(kind).join("probe.log"), probe_log)
                        .unwrap();
                    files.insert("probe.log".to_string(), log_hash(probe_log));
                }
                if *kind == "factory" {
                    std::fs::write(
                        evidence.path().join(kind).join("provider.log"),
                        provider_log,
                    )
                    .unwrap();
                    files.insert(
                        "provider.log".to_string(),
                        factory.provider_execution_sha256.clone(),
                    );
                }
                if *kind == "installation" {
                    std::fs::write(
                        evidence.path().join(kind).join("installer.log"),
                        installer_log,
                    )
                    .unwrap();
                    files.insert(
                        "installer.log".to_string(),
                        native.installer_execution_sha256.clone(),
                    );
                }
                let completed = observation(index, files);
                let record = observation_json(&completed);
                std::fs::create_dir_all(evidence.path().join(kind)).unwrap();
                std::fs::write(
                    evidence.path().join(kind).join("observation.json"),
                    serde_json::to_string(&record).unwrap(),
                )
                .unwrap();
            }
            let (payload_bytes, _, _) = crate::model::Candidate::fixture_pair();
            std::fs::write(release.path().join("payload.json"), &payload_bytes).unwrap();
            Self {
                _evidence: evidence,
                _release: release,
                manifest,
            }
        }
    }

    /// rewrite_observation mutates one staged observation file as the
    /// corresponding failed, cancelled, incomplete, mismatched or forged
    /// record would appear on disk: same JSON shape, altered content.
    fn rewrite_observation(
        staged: &Staged,
        kind: &str,
        mutate: impl FnOnce(&mut serde_json::Value),
    ) {
        let path = staged.evidence().join(kind).join("observation.json");
        let mut value: serde_json::Value =
            serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
        mutate(&mut value);
        std::fs::write(&path, serde_json::to_vec(&value).unwrap()).unwrap();
    }

    /// R50 development proof: completed native evidence across all four
    /// observation kinds reaches finalization — stamp persists the
    /// receipt, verify re-collects it from the retained directory, and
    /// re-stamping is idempotent. Isolated temp directories only; no
    /// signing, publication or production bytes.
    #[test]
    fn valid_evidence_stamps_finalization_and_reverifies() {
        let staged = Staged::stage();
        let receipt = super::stamp(&staged.manifest, staged.release(), staged.evidence())
            .expect("completed evidence must finalize");
        assert!(staged.release().join("docker-qualification.json").is_file());
        let reverified = super::verify(&staged.manifest, staged.release())
            .expect("retained evidence must re-verify");
        assert_eq!(reverified, receipt);
        let restamped = super::stamp(&staged.manifest, staged.release(), staged.evidence())
            .expect("identical re-collection must re-stamp");
        assert_eq!(restamped, receipt);
    }

    /// R50 failed/cancelled observations cannot finalize, even when every
    /// other binding matches.
    #[test]
    fn failed_and_cancelled_observations_cannot_finalize() {
        let staged = Staged::stage();
        rewrite_observation(&staged, "factory", |value| {
            value["Outcome"] = serde_json::Value::String("failed".to_string());
            value["Execution"] = serde_json::Value::String("failed".to_string());
        });
        assert!(super::stamp(&staged.manifest, staged.release(), staged.evidence()).is_err());

        let staged = Staged::stage();
        rewrite_observation(&staged, "installation", |value| {
            value["Outcome"] = serde_json::Value::String("cancelled".to_string());
        });
        assert!(super::stamp(&staged.manifest, staged.release(), staged.evidence()).is_err());
    }

    /// R50 incomplete observations cannot finalize: a missing record and
    /// an empty retained-files map both refuse.
    #[test]
    fn incomplete_observations_cannot_finalize() {
        let staged = Staged::stage();
        std::fs::remove_file(staged.evidence().join("browser_project/observation.json")).unwrap();
        assert!(super::stamp(&staged.manifest, staged.release(), staged.evidence()).is_err());

        let staged = Staged::stage();
        rewrite_observation(&staged, "persistence", |value| {
            value["Files"] = serde_json::Value::Object(serde_json::Map::new());
        });
        assert!(super::stamp(&staged.manifest, staged.release(), staged.evidence()).is_err());
    }

    /// R50 mismatched observations cannot finalize: a foreign requested
    /// revision and a download whose bytes no longer match the manifest
    /// both refuse.
    #[test]
    fn mismatched_observations_cannot_finalize() {
        let staged = Staged::stage();
        rewrite_observation(&staged, "factory", |value| {
            value["RequestedRevision"] = serde_json::Value::String("0".repeat(40));
        });
        assert!(super::stamp(&staged.manifest, staged.release(), staged.evidence()).is_err());

        let staged = Staged::stage();
        std::fs::write(
            staged.evidence().join("downloads/docker.json"),
            b"tampered download bytes\n",
        )
        .unwrap();
        assert!(super::stamp(&staged.manifest, staged.release(), staged.evidence()).is_err());
    }

    /// R50 forged native evidence cannot finalize: native bytes that no
    /// longer match the observation's retained file hash refuse.
    #[test]
    fn forged_native_evidence_cannot_finalize() {
        let staged = Staged::stage();
        let path = staged.evidence().join("factory/native.json");
        let mut bytes = std::fs::read(&path).unwrap();
        bytes.extend_from_slice(b"forged");
        std::fs::write(&path, &bytes).unwrap();
        assert!(super::stamp(&staged.manifest, staged.release(), staged.evidence()).is_err());
    }

    /// R50 preservation and recovery: evidence tampered after stamping no
    /// longer verifies against the retained receipt, and a fresh stamp
    /// after receipt removal re-collects (refusing the tampered
    /// evidence) rather than restoring the old receipt.
    #[test]
    fn tampered_evidence_breaks_preservation_and_restamp_refuses() {
        let staged = Staged::stage();
        super::stamp(&staged.manifest, staged.release(), staged.evidence())
            .expect("completed evidence must finalize");
        let path = staged.evidence().join("installation/native.json");
        let mut bytes = std::fs::read(&path).unwrap();
        bytes.extend_from_slice(b"tampered");
        std::fs::write(&path, &bytes).unwrap();
        assert!(super::verify(&staged.manifest, staged.release()).is_err());
        std::fs::remove_file(staged.release().join("docker-qualification.json")).unwrap();
        // Recovery re-collects the tampered evidence, so it must refuse
        // rather than restore the old receipt.
        assert!(super::stamp(&staged.manifest, staged.release(), staged.evidence()).is_err());
    }
}
