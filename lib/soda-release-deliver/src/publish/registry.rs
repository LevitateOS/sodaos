//! GHCR candidate upload and independent anonymous identity verification.
use super::{api, check_source_ci, GitHub};
use crate::buildx::Image;
use crate::model::{DockerMetadata, DOCKER_REPOSITORY};
use crate::{docker, hash_bytes, Error};
use serde::{Deserialize, Serialize};
use std::os::unix::fs::{DirBuilderExt, MetadataExt};
use std::path::Path;
use std::process::Command;

const PACKAGE: &str = "orgs/LevitateOS/packages/container/sodaos";

pub trait Registry {
    fn run(&self, args: &[String]) -> Result<Vec<u8>, Error>;
}
pub struct Skopeo;
impl Registry for Skopeo {
    fn run(&self, args: &[String]) -> Result<Vec<u8>, Error> {
        let result = Command::new("/usr/bin/skopeo")
            .args(args)
            .output()
            .map_err(|_| Error::msg("Skopeo unavailable"))?;
        if !result.status.success() {
            // Never return credential-bearing transport diagnostics.
            if String::from_utf8_lossy(&result.stderr).contains("manifest unknown") {
                return Err(Error::msg("registry version not found"));
            }
            return Err(Error::msg("registry operation failed; inspect the version before retrying; anonymous readback requires public package visibility"));
        }
        if result.stdout.len() > 4 << 20 {
            return Err(Error::msg("registry response exceeds bound"));
        }
        Ok(result.stdout)
    }
}
fn run(registry: &dyn Registry, args: &[&str]) -> Result<Vec<u8>, Error> {
    registry.run(
        &args
            .iter()
            .map(|value| (*value).to_owned())
            .collect::<Vec<_>>(),
    )
}
fn path(path: &Path) -> Result<&str, Error> {
    path.to_str().ok_or_else(Error::refused)
}

#[derive(Debug, Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub struct Publication {
    pub format: u32,
    pub tag: String,
    pub reference: String,
    pub soda_revision: String,
    pub forgejo_revision: String,
    pub candidate_sha256: String,
    pub image_archive_sha256: String,
    pub ci_run: u64,
    pub anonymous: bool,
    pub qualified: bool,
    pub image: Image,
}

fn authenticated_tag_absent(
    gh: &dyn GitHub,
    registry: &dyn Registry,
    tag: &str,
    auth_file: &Path,
) -> Result<(), Error> {
    // A first-time package has no registry read namespace yet. GitHub's
    // authenticated 404 is distinct from an ambiguous registry auth failure.
    match api::<serde_json::Value>(gh, PACKAGE) {
        Err(error) if error.0 == "GitHub resource not found" => return Ok(()),
        Err(error) => return Err(error),
        Ok(_) => (),
    }
    match run(
        registry,
        &["inspect", "--raw", "--authfile", path(auth_file)?, tag],
    ) {
        Err(error) if error.0 == "registry version not found" => Ok(()),
        Err(error) => Err(error),
        Ok(_) => Err(Error::msg(
            "Docker version already exists; verify it instead of overwriting it",
        )),
    }
}

/// Upload only verified production output with successful exact-source main CI.
/// Never move latest or change GitHub releases. Retain partial output on failure.
pub fn stage(
    gh: &dyn GitHub,
    registry: &dyn Registry,
    candidate: &Path,
    auth_file: &Path,
    out: &Path,
) -> Result<Publication, Error> {
    let metadata = docker::admit_publication(path(candidate)?)?;
    stage_admitted(gh, registry, candidate, auth_file, out, &metadata)
}

pub(super) fn stage_admitted(
    gh: &dyn GitHub,
    registry: &dyn Registry,
    candidate: &Path,
    auth_file: &Path,
    out: &Path,
    metadata: &DockerMetadata,
) -> Result<Publication, Error> {
    metadata.require_production()?;
    let ci_run = check_source_ci(gh, &metadata.soda_revision, &metadata.forgejo_revision)?;
    let auth = std::fs::symlink_metadata(auth_file)
        .map_err(|_| Error::msg("restricted registry auth file required"))?;
    if !auth.is_file() || auth.mode() & 0o077 != 0 || auth.len() == 0 {
        return Err(Error::msg("restricted regular registry auth file required"));
    }
    let tag = format!("docker://{DOCKER_REPOSITORY}:{}", metadata.release_tag);
    authenticated_tag_absent(gh, registry, &tag, auth_file)?;
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(out)
        .map_err(|_| Error::msg("fresh publication output directory required"))?;
    let image = format!("oci-archive:{}", path(&candidate.join(docker::IMAGE_PATH))?);
    let digest_file = out.join("pushed.digest");
    run(
        registry,
        &[
            "copy",
            "--preserve-digests",
            "--dest-authfile",
            path(auth_file)?,
            "--digestfile",
            path(&digest_file)?,
            &image,
            &tag,
        ],
    )?;
    let digest = std::fs::read_to_string(&digest_file)
        .map_err(|_| Error::msg("registry push digest unavailable"))?;
    if digest.trim() != metadata.image.manifest {
        return Err(Error::msg(
            "registry push changed the admitted manifest digest",
        ));
    }
    record_public(registry, &metadata, ci_run, out)
}

/// Verify an already-uploaded version without replaying any registry write.
/// This is also the next action after fixing first-publication visibility.
pub fn verify(
    gh: &dyn GitHub,
    registry: &dyn Registry,
    candidate: &Path,
    out: &Path,
) -> Result<Publication, Error> {
    let metadata = docker::admit_publication(path(candidate)?)?;
    let ci_run = check_source_ci(gh, &metadata.soda_revision, &metadata.forgejo_revision)?;
    let directory = std::fs::symlink_metadata(out).map_err(|_| Error::refused())?;
    if !directory.is_dir() || directory.mode() & 0o077 != 0 {
        return Err(Error::msg("private publication output directory required"));
    }
    record_public(registry, &metadata, ci_run, out)
}

fn record_public(
    registry: &dyn Registry,
    metadata: &DockerMetadata,
    ci_run: u64,
    out: &Path,
) -> Result<Publication, Error> {
    let (mut receipt, raw) = observe_public(registry, metadata, ci_run)?;
    if out.join("registry.json").exists() {
        let previous: Publication = super::read_json(&out.join("registry.json"))?;
        // A later successful CI rerun need not replace the original publication
        // receipt; identity is freshly observed before retaining it.
        if previous.ci_run == 0 {
            return Err(Error::msg(
                "existing publication receipt has no successful source CI",
            ));
        }
        receipt.ci_run = previous.ci_run;
        if previous != receipt {
            return Err(Error::msg(
                "existing publication receipt differs from public image",
            ));
        }
    }
    let mut bytes = serde_json::to_vec_pretty(&receipt).map_err(|_| Error::refused())?;
    bytes.push(b'\n');
    super::record_verified(&out.join("public-manifest.json"), &raw)?;
    super::record_verified(&out.join("registry.json"), &bytes)?;
    Ok(receipt)
}

pub(super) fn observe_public(
    registry: &dyn Registry,
    metadata: &DockerMetadata,
    ci_run: u64,
) -> Result<(Publication, Vec<u8>), Error> {
    metadata.require_production()?;
    let tag = format!("docker://{DOCKER_REPOSITORY}:{}", metadata.release_tag);
    let raw = run(registry, &["inspect", "--raw", "--no-creds", &tag])?;
    if hash_bytes(&raw) != metadata.image.manifest {
        return Err(Error::msg(
            "public version manifest differs from admitted image",
        ));
    }
    #[derive(Deserialize)]
    struct Descriptor {
        digest: String,
    }
    #[derive(Deserialize)]
    struct Manifest {
        config: Descriptor,
    }
    let manifest: Manifest =
        serde_json::from_slice(&raw).map_err(|_| Error::msg("public image manifest refused"))?;
    let reference = metadata.reference()?;
    let bytes = run(
        registry,
        &[
            "inspect",
            "--no-tags",
            "--no-creds",
            &format!("docker://{reference}"),
        ],
    )?;
    #[derive(Deserialize)]
    #[serde(rename_all = "PascalCase")]
    struct Observation {
        digest: String,
        architecture: String,
        os: String,
        labels: std::collections::BTreeMap<String, String>,
    }
    let observed: Observation = serde_json::from_slice(&bytes)
        .map_err(|_| Error::msg("anonymous registry observation refused"))?;
    let label = |name: &str| observed.labels.get(name).cloned().unwrap_or_default();
    let image = Image {
        config: manifest.config.digest,
        manifest: observed.digest,
        architecture: observed.architecture,
        revision: label("org.opencontainers.image.revision"),
        source: label("org.opencontainers.image.source"),
        base_name: label("org.opencontainers.image.base.name"),
        base_digest: label("org.opencontainers.image.base.digest"),
    };
    if observed.os != "linux" {
        return Err(Error::msg("public Docker image is not Linux"));
    }
    metadata.verify_registry(&image)?;
    let receipt = Publication {
        format: 1,
        tag: metadata.release_tag.clone(),
        reference,
        soda_revision: metadata.soda_revision.clone(),
        forgejo_revision: metadata.forgejo_revision.clone(),
        candidate_sha256: metadata.candidate_sha256.clone(),
        image_archive_sha256: metadata.image_archive.sha256.clone(),
        ci_run,
        anonymous: true,
        qualified: false,
        image,
    };
    Ok((receipt, raw))
}

pub(super) fn promote_digest(
    registry: &dyn Registry,
    metadata: &DockerMetadata,
    auth_file: &Path,
    directory: &Path,
) -> Result<(), Error> {
    let auth = std::fs::symlink_metadata(auth_file)
        .map_err(|_| Error::msg("restricted registry auth file required"))?;
    if !auth.is_file() || auth.mode() & 0o077 != 0 || auth.len() == 0 {
        return Err(Error::msg("restricted regular registry auth file required"));
    }
    let digest = directory.join("recommended.digest");
    run(
        registry,
        &[
            "copy",
            "--preserve-digests",
            "--src-no-creds",
            "--dest-authfile",
            path(auth_file)?,
            "--digestfile",
            path(&digest)?,
            &format!("docker://{}", metadata.reference()?),
            &format!("docker://{DOCKER_REPOSITORY}:latest"),
        ],
    )?;
    if std::fs::read_to_string(&digest)
        .map_err(|_| Error::msg("promotion digest unavailable"))?
        .trim()
        != metadata.image.manifest
    {
        return Err(Error::msg(
            "recommendation digest changed; inspect partial promotion",
        ));
    }
    let raw = run(
        registry,
        &[
            "inspect",
            "--raw",
            "--no-creds",
            &format!("docker://{DOCKER_REPOSITORY}:latest"),
        ],
    )?;
    if hash_bytes(&raw) != metadata.image.manifest {
        return Err(Error::msg(
            "latest does not identify qualified digest; inspect partial promotion",
        ));
    }
    Ok(())
}

pub(super) fn latest_digest(registry: &dyn Registry) -> Result<String, Error> {
    let raw = run(
        registry,
        &[
            "inspect",
            "--raw",
            "--no-creds",
            &format!("docker://{DOCKER_REPOSITORY}:latest"),
        ],
    )?;
    Ok(hash_bytes(&raw))
}
