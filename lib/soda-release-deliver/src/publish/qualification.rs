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

/// Controller-owned completion of an admitted collector, distinct from build output.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ProtectedEvidence {
    pub format: u32,
    pub purpose: String,
    pub source_revision: String,
    pub architecture: String,
    pub candidate_sha256: String,
    pub payload_sha256: String,
    pub build_sha256: String,
    pub target: String,
    pub controller_sha256: String,
    pub collector_sha256: String,
    pub authority_sha256: String,
    pub unit: String,
    pub exit_code: i32,
    pub native_retirement: String,
    pub files: BTreeMap<String, FileIdentity>,
}

/// Every ancestor must be controlled by root; a root-owned leaf in a user's
/// writable directory does not establish release authority.
pub fn protected_path(path: &Path, directory: bool) -> Result<(), Error> {
    use std::os::unix::fs::MetadataExt;
    if !path.is_absolute()
        || path.components().any(|p| {
            matches!(
                p,
                std::path::Component::ParentDir | std::path::Component::CurDir
            )
        })
    {
        return Err(Error::msg("absolute protected release path required"));
    }
    for (index, ancestor) in path.ancestors().enumerate() {
        let meta = std::fs::symlink_metadata(ancestor).map_err(|_| refused())?;
        if meta.uid() != 0
            || meta.mode() & 0o022 != 0
            || meta.file_type().is_symlink()
            || (index > 0 && !meta.is_dir())
            || (index == 0
                && if directory {
                    !meta.is_dir() || meta.mode() & 0o777 != 0o700
                } else {
                    !meta.is_file() || meta.mode() & 0o777 != 0o600 || meta.nlink() != 1
                })
        {
            return Err(Error::msg(
                "root-owned private release path and trusted ancestors required",
            ));
        }
    }
    Ok(())
}

pub fn protected_files(directory: &Path) -> Result<BTreeMap<String, FileIdentity>, Error> {
    fn walk(
        root: &Path,
        directory: &Path,
        files: &mut BTreeMap<String, FileIdentity>,
        total: &mut u64,
    ) -> Result<(), Error> {
        protected_path(directory, true)?;
        for entry in std::fs::read_dir(directory).map_err(|_| refused())? {
            let path = entry.map_err(|_| refused())?.path();
            let meta = std::fs::symlink_metadata(&path).map_err(|_| refused())?;
            if meta.is_dir() {
                walk(root, &path, files, total)?;
                continue;
            }
            protected_path(&path, false)?;
            let name = path
                .strip_prefix(root)
                .ok()
                .and_then(Path::to_str)
                .ok_or_else(refused)?;
            if name == "release-evidence.json" {
                continue;
            }
            *total = total.checked_add(meta.len()).ok_or_else(refused)?;
            if files.len() >= 1024 || meta.len() > 16 << 20 || *total > 64 << 20 {
                return Err(refused());
            }
            let asset = assets::inspect(
                &path,
                path.file_name()
                    .and_then(|s| s.to_str())
                    .ok_or_else(refused)?,
            )?;
            files.insert(
                name.to_owned(),
                FileIdentity {
                    sha256: asset.sha256,
                    bytes: asset.bytes,
                },
            );
        }
        Ok(())
    }
    let mut files = BTreeMap::new();
    walk(directory, directory, &mut files, &mut 0)?;
    if files.is_empty() {
        return Err(Error::msg("collector evidence is incomplete"));
    }
    Ok(files)
}

pub fn read_protected(directory: &Path) -> Result<ProtectedEvidence, Error> {
    protected_path(&directory.join("release-evidence.json"), false)?;
    let record: ProtectedEvidence = assets::read_json(&directory.join("release-evidence.json"))?;
    if record.format != 1
        || !matches!(record.purpose.as_str(), "development" | "production")
        || !crate::buildx::is_revision(&record.source_revision)
        || record.architecture != "x86_64"
        || record.target.is_empty()
        || record.target.len() > 512
        || record.target.contains(['\0', '\n', '\r'])
        || record.exit_code != 0
        || record.native_retirement != "confirmed"
        || record
            .unit
            .strip_prefix("soda-qualify-")
            .and_then(|s| s.strip_suffix(".service"))
            .is_none_or(|s| {
                s.len() != 32
                    || !s
                        .bytes()
                        .all(|c| c.is_ascii_hexdigit() && !c.is_ascii_uppercase())
            })
        || [
            &record.candidate_sha256,
            &record.payload_sha256,
            &record.build_sha256,
            &record.controller_sha256,
            &record.collector_sha256,
            &record.authority_sha256,
        ]
        .iter()
        .any(|s| !is_digest(s))
        || protected_files(directory)? != record.files
    {
        return Err(Error::msg(
            "protected collector completion or exact evidence binding refused",
        ));
    }
    Ok(record)
}

pub fn protected_for_manifest(
    manifest: &Manifest,
    evidence: &Path,
) -> Result<ProtectedEvidence, Error> {
    let record = read_protected(evidence)?;
    if record.purpose != "production"
        || record.source_revision != manifest.soda_revision
        || record.candidate_sha256 != manifest.asset("candidate.json")?.sha256
        || record.payload_sha256 != manifest.asset("payload.json")?.sha256
    {
        return Err(Error::msg(
            "production-bound protected collector evidence required",
        ));
    }
    Ok(record)
}

pub fn protected_for_candidate(
    candidate: &Path,
    evidence: &Path,
) -> Result<ProtectedEvidence, Error> {
    let record = read_protected(evidence)?;
    let (payload, _, _) =
        crate::admission::load_admitted_candidate(candidate.to_str().ok_or_else(refused)?)?;
    if record.purpose != "production"
        || record.source_revision != payload.revision
        || record.candidate_sha256
            != assets::inspect(&candidate.join("candidate.json"), "candidate.json")?.sha256
        || record.payload_sha256
            != assets::inspect(&candidate.join("payload.json"), "payload.json")?.sha256
    {
        return Err(Error::msg(
            "production-bound protected collector evidence required",
        ));
    }
    Ok(record)
}

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
        let payload: crate::payload::Payload = assets::read_json(&directory.join("payload.json"))?;
        let configs = soda_build_tools::payload::Role::iter()
            .map(|role| {
                (
                    role.name().to_string(),
                    role.binding(&payload.images).config.clone(),
                )
            })
            .collect::<BTreeMap<_, _>>();
        self.validate_native_binding(docker, manifest, &configs)?;
        self.validate_environment()?;
        self.validate_persistence()?;
        if self.observations.len() != KINDS.len() {
            return Err(refused());
        }
        self.validate_factory()?;
        if KINDS
            .iter()
            .any(|kind| !self.observations.contains_key(*kind))
        {
            return Err(refused());
        }
        Ok(())
    }

    fn validate_native_binding(
        &self,
        docker: &crate::model::DockerMetadata,
        manifest: &Manifest,
        configs: &BTreeMap<String, String>,
    ) -> Result<(), Error> {
        let native = &self.native;
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
            || native.application_configs != *configs
        {
            return Err(refused());
        }
        Ok(())
    }

    fn validate_environment(&self) -> Result<(), Error> {
        let native = &self.native;
        if native.target.is_empty()
            || native.os != "linux"
            || native.architecture != "x86_64"
            || !native.rootful
            || native.cgroup_version != 2
            || !native.host_network
            || !native.fresh_volumes
            || !is_digest(&native.outer_id)
        {
            return Err(refused());
        }
        Ok(())
    }

    fn validate_persistence(&self) -> Result<(), Error> {
        let native = &self.native;
        let persistence = &self.persistence;
        if persistence.target != native.target
            || persistence.registry_manifest != native.registry_manifest
            || persistence.registry_config != native.registry_config
            || persistence.outer_before != native.outer_id
            || !is_digest(&persistence.outer_after)
            || persistence.outer_before == persistence.outer_after
            || !is_digest(&persistence.project_before)
            || persistence.project_before != persistence.project_after
            || !is_digest(&persistence.protected_before_sha256)
            || persistence.protected_before_sha256 != persistence.protected_after_sha256
        {
            return Err(refused());
        }
        Ok(())
    }

    fn validate_factory(&self) -> Result<(), Error> {
        let factory = &self.factory;
        if !is_digest(&factory.provider_execution_sha256)
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
        {
            return Err(refused());
        }
        Ok(())
    }
}
fn validate_collected_downloads(evidence: &Path, manifest: &Manifest) -> Result<(), Error> {
    for name in DOWNLOADS {
        if &assets::inspect(&evidence.join("downloads").join(name), name)?
            != manifest.asset(name)?
        {
            return Err(Error::msg("downloaded qualification bytes changed"));
        }
    }
    Ok(())
}

fn validate_observation_outcome(
    observation: &soda_acceptance::report::Observation,
    manifest: &Manifest,
    native: &NativeDocker,
) -> Result<(), Error> {
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
    Ok(())
}

fn check_observation_time(
    observation: &soda_acceptance::report::Observation,
    now: i128,
    previous_finish: &mut Option<i128>,
) -> Result<(), Error> {
    let started = soda_wire_time::parse_nanos(&observation.started).ok_or_else(refused)?;
    let finished = soda_wire_time::parse_nanos(&observation.finished).ok_or_else(refused)?;
    if started >= finished || finished > now || previous_finish.is_some_and(|time| started < time) {
        return Err(Error::msg("native qualification observations must be finalized in installation, browser/Project, factory and persistence order"));
    }
    *previous_finish = Some(finished);
    Ok(())
}

fn validate_observation_downloads(
    observation: &soda_acceptance::report::Observation,
    manifest: &Manifest,
) -> Result<(), Error> {
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
    Ok(())
}

fn validate_observation_evidence(
    kind: &str,
    evidence: &Path,
    observation: &soda_acceptance::report::Observation,
    factory: &NativeFactory,
    native: &NativeDocker,
) -> Result<(), Error> {
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
    Ok(())
}

fn collect(
    manifest: &Manifest,
    release: &Path,
    evidence: &Path,
) -> Result<DockerQualification, Error> {
    let protected = protected_for_manifest(manifest, evidence)?;
    let native = read_native(evidence)?;
    if native.target != protected.target {
        return Err(refused());
    }
    let root = Root::open(evidence.to_str().ok_or_else(refused)?)?;
    let factory: NativeFactory =
        crate::json_serde::strict(&read_at(&root, "factory/native.json", 1 << 20)?)
            .map_err(|_| refused())?;
    let persistence: NativePersistence =
        crate::json_serde::strict(&read_at(&root, "persistence/native.json", 1 << 20)?)
            .map_err(|_| refused())?;
    validate_collected_downloads(evidence, manifest)?;
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
        validate_observation_outcome(&observation, manifest, &native)?;
        check_observation_time(&observation, now, &mut previous_finish)?;
        validate_observation_downloads(&observation, manifest)?;
        validate_observation_evidence(kind, evidence, &observation, &factory, &native)?;
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
