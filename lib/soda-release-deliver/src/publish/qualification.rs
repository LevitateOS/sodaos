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
        let configs = payload
            .images
            .iter()
            .map(|(name, image)| (name.clone(), image.config.clone()))
            .collect::<BTreeMap<_, _>>();
        if self.format != 1
            || native.tag != manifest.tag
            || native.registry_manifest != docker.image.manifest
            || native.registry_config != docker.image.config
            || native.compose_sha256 != docker.compose.sha256
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
    for name in [
        "docker.json",
        "sodaos-docker-compose.yaml",
        "candidate.json",
        "payload.json",
    ] {
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
        for name in [
            "docker.json",
            "sodaos-docker-compose.yaml",
            "candidate.json",
            "payload.json",
        ] {
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
