//! Fixed Soda application repositories and their separately released inputs.
use super::registry::{authenticated_tag_absent, path, run, Registry};
use super::{assets, record_verified, GitHub, Manifest};
use crate::payload::{Image, Payload, Role};
use crate::{hash_bytes, Error};
use serde::Deserialize;
use std::os::unix::fs::{DirBuilderExt, MetadataExt};
use std::path::{Path, PathBuf};

const LOCAL: [Role; 4] = [
    Role::Dashboard,
    Role::Forgejo,
    Role::ProjectOs,
    Role::CanlangRuntime,
];

fn payload(directory: &Path, manifest: &Manifest) -> Result<Payload, Error> {
    let payload = crate::payload::load(path(&directory.join("payload.json"))?)?;
    if payload.revision != manifest.soda_revision
        || payload.fountain.source_revision != manifest.forgejo_revision
        || payload.release_tag != manifest.tag
    {
        return Err(Error::msg(
            "prepared component source/tag differs from release",
        ));
    }
    for role in LOCAL {
        let image = role.binding(&payload.images);
        if image.reference
            != format!(
                "{}-{}@{}",
                crate::model::DOCKER_REPOSITORY,
                role.name(),
                image.manifest
            )
        {
            return Err(Error::msg(
                "production application repository differs from fixed destination",
            ));
        }
    }
    Ok(payload)
}

fn observe_platform(registry: &dyn Registry, reference: &str, expected: &str) -> Result<(), Error> {
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
    struct Platform {
        digest: String,
        architecture: String,
        os: String,
    }
    let observed: Platform = serde_json::from_slice(&bytes)
        .map_err(|_| Error::msg("public component platform refused"))?;
    if observed.digest != expected || observed.architecture != "amd64" || observed.os != "linux" {
        return Err(Error::msg(
            "public component digest/platform differs from payload",
        ));
    }
    Ok(())
}

fn observe(
    registry: &dyn Registry,
    reference: &str,
    manifest: &str,
    config: &str,
) -> Result<(), Error> {
    let raw = run(
        registry,
        &[
            "inspect",
            "--raw",
            "--no-creds",
            &format!("docker://{reference}"),
        ],
    )?;
    if hash_bytes(&raw) != manifest {
        return Err(Error::msg(
            "public component manifest/config differs from payload",
        ));
    }
    #[derive(Deserialize)]
    struct Descriptor {
        digest: String,
    }
    #[derive(Deserialize)]
    struct ImageManifest {
        config: Descriptor,
    }
    // Single-arch payloads (local builds, Fountain) bind config directly.
    if let Ok(parsed) = serde_json::from_slice::<ImageManifest>(&raw) {
        if parsed.config.digest != config {
            return Err(Error::msg(
                "public component manifest/config differs from payload",
            ));
        }
        return observe_platform(registry, reference, manifest);
    }
    // Upstream pins (proxy, postgres) are multi-arch indexes: select the
    // bound linux/amd64 child from the verified index and check it instead.
    #[derive(Deserialize)]
    struct ChildPlatform {
        architecture: String,
        os: String,
    }
    #[derive(Deserialize)]
    struct IndexEntry {
        digest: String,
        platform: Option<ChildPlatform>,
    }
    #[derive(Deserialize)]
    struct ImageIndex {
        manifests: Vec<IndexEntry>,
    }
    let index: ImageIndex = serde_json::from_slice(&raw)
        .map_err(|_| Error::msg("public component manifest refused"))?;
    let child = index
        .manifests
        .iter()
        .find(|entry| {
            entry
                .platform
                .as_ref()
                .is_some_and(|platform| platform.architecture == "amd64" && platform.os == "linux")
        })
        .ok_or_else(|| Error::msg("public component manifest refused"))?;
    let (repository, _) = reference.rsplit_once('@').ok_or_else(Error::refused)?;
    let child_reference = format!("{repository}@{}", child.digest);
    let child_raw = run(
        registry,
        &[
            "inspect",
            "--raw",
            "--no-creds",
            &format!("docker://{child_reference}"),
        ],
    )?;
    let parsed: ImageManifest = serde_json::from_slice(&child_raw)
        .map_err(|_| Error::msg("public component manifest refused"))?;
    if parsed.config.digest != config {
        return Err(Error::msg(
            "public component manifest/config differs from payload",
        ));
    }
    observe_platform(registry, &child_reference, &child.digest)
}

fn observe_binding(registry: &dyn Registry, image: &Image) -> Result<(), Error> {
    observe(registry, &image.reference, &image.manifest, &image.config)
}

fn observe_payload(registry: &dyn Registry, payload: &Payload) -> Result<(), Error> {
    for role in Role::ALL {
        let image = role.binding(&payload.images);
        observe_binding(registry, image)?;
        if LOCAL.contains(&role) {
            let tag = format!(
                "{}:{}",
                image
                    .reference
                    .split('@')
                    .next()
                    .ok_or_else(Error::refused)?,
                payload.release_tag
            );
            observe(registry, &tag, &image.manifest, &image.config)?;
        }
    }
    let fountain = &payload.fountain;
    let (_, digest) = fountain.image.rsplit_once('@').ok_or_else(Error::refused)?;
    observe(registry, &fountain.image, digest, &fountain.image_config)?;
    observe_binding(
        registry,
        &payload.optional_extensions.tailnet.images.tailscale,
    )
}

pub(super) fn verify(
    registry: &dyn Registry,
    directory: &Path,
    manifest: &Manifest,
) -> Result<(), Error> {
    let payload = payload(directory, manifest)?;
    observe_payload(registry, &payload)?;
    let out = directory.join("components");
    let st = std::fs::symlink_metadata(&out)
        .map_err(|_| Error::msg("component staging attempt required before readback"))?;
    if !st.is_dir() || st.mode() & 0o077 != 0 {
        return Err(Error::refused());
    }
    let bytes = serde_json::to_vec(&payload).map_err(|_| Error::refused())?;
    record_verified(&out.join("public.json"), &bytes)
}

pub(super) fn observe_public(
    registry: &dyn Registry,
    directory: &Path,
    manifest: &Manifest,
) -> Result<(), Error> {
    observe_payload(registry, &payload(directory, manifest)?)
}

fn admit_stage_attempt<'a>(out: &Path, auth: Option<&'a Path>) -> Result<&'a Path, Error> {
    if out.exists() {
        return Err(Error::msg("component staging already attempted; inspect remote state and run verify-images instead of replaying uploads"));
    }
    let auth = auth.ok_or_else(|| {
        Error::msg("restricted registry auth file required for application staging")
    })?;
    let st = std::fs::symlink_metadata(auth).map_err(|_| Error::refused())?;
    if !st.is_file() || st.mode() & 0o077 != 0 || st.len() == 0 {
        return Err(Error::refused());
    }
    Ok(auth)
}

fn admit_stage_candidate(
    directory: &Path,
    manifest: &Manifest,
    payload: &Payload,
) -> Result<PathBuf, Error> {
    let candidate = assets::candidate(directory, manifest)?;
    let (actual, transport, _) = crate::admission::load_admitted_candidate(path(&candidate)?)?;
    if actual != *payload {
        return Err(Error::msg(
            "local candidate differs from prepared component payload",
        ));
    }
    assets::admit_build(
        &candidate,
        &actual,
        &transport,
        manifest.distribution,
        &manifest.tag,
    )?;
    crate::check::check_candidate(
        path(&candidate)?,
        &payload.architecture,
        &manifest.soda_revision,
        &manifest.forgejo_revision,
    )?;
    Ok(candidate)
}

fn observe_stage_inputs(
    gh: &dyn GitHub,
    registry: &dyn Registry,
    manifest: &Manifest,
    payload: &Payload,
    auth: &Path,
) -> Result<(), Error> {
    // Check every fixed destination and upstream input before the first upload.
    for role in LOCAL {
        authenticated_tag_absent(
            gh,
            registry,
            &format!(
                "docker://{}-{}:{}",
                crate::model::DOCKER_REPOSITORY,
                role.name(),
                manifest.tag
            ),
            auth,
            &format!("orgs/LevitateOS/packages/container/sodaos-{}", role.name()),
        )?;
    }
    for role in [Role::Proxy, Role::Postgres] {
        observe_binding(registry, role.binding(&payload.images))?;
    }
    let (_, digest) = payload
        .fountain
        .image
        .rsplit_once('@')
        .ok_or_else(Error::refused)?;
    observe(
        registry,
        &payload.fountain.image,
        digest,
        &payload.fountain.image_config,
    )?;
    observe_binding(
        registry,
        &payload.optional_extensions.tailnet.images.tailscale,
    )?;
    Ok(())
}

fn upload_role_copy(
    registry: &dyn Registry,
    out: &Path,
    candidate: &Path,
    manifest: &Manifest,
    payload: &Payload,
    auth: &Path,
    role: Role,
) -> Result<(), Error> {
    let digest_file = out.join(format!("{}.digest", role.name()));
    run(
        registry,
        &[
            "copy",
            "--preserve-digests",
            "--dest-authfile",
            path(auth)?,
            "--digestfile",
            path(&digest_file)?,
            &format!(
                "oci-archive:{}",
                path(&candidate.join(format!("images/{}.oci", role.name())))?
            ),
            &format!(
                "docker://{}-{}:{}",
                crate::model::DOCKER_REPOSITORY,
                role.name(),
                manifest.tag
            ),
        ],
    )?;
    if std::fs::read_to_string(&digest_file)
        .map_err(|_| Error::refused())?
        .trim()
        != role.binding(&payload.images).manifest
    {
        return Err(Error::msg(
            "application upload changed the admitted manifest digest",
        ));
    }
    Ok(())
}

pub(super) fn stage(
    gh: &dyn GitHub,
    registry: &dyn Registry,
    directory: &Path,
    manifest: &Manifest,
    auth: Option<&Path>,
) -> Result<(), Error> {
    let payload = payload(directory, manifest)?;
    let out = directory.join("components");
    if out.join("public.json").exists() {
        return verify(registry, directory, manifest);
    }
    let auth = admit_stage_attempt(&out, auth)?;
    let candidate = admit_stage_candidate(directory, manifest, &payload)?;
    observe_stage_inputs(gh, registry, manifest, &payload, auth)?;
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&out)
        .map_err(|_| Error::refused())?;
    for role in LOCAL {
        upload_role_copy(registry, &out, &candidate, manifest, &payload, auth, role)?;
    }
    verify(registry, directory, manifest)
}
