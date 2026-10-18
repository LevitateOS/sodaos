//! GitHub Releases owns publication state; local artifacts remain immutable inputs.
use crate::Error;
use serde::Deserialize;
use std::path::{Path, PathBuf};
use std::process::Command;
pub mod assets;
mod components;
mod installer;
pub mod qualification;
pub mod registry;
pub(crate) use assets::valid_tag;
use assets::{inspect, inventory, read_json, Asset};
pub use assets::{prepare, Manifest, Qualification};
const REPOSITORY: &str = "github.com/LevitateOS/sodaos";
const API: &str = "repos/LevitateOS/sodaos";

/// Injectable command boundary; production delegates directly to the mature gh CLI.
pub trait GitHub {
    fn run(&self, args: &[String]) -> Result<Vec<u8>, Error>;
}
pub struct Gh;
impl GitHub for Gh {
    fn run(&self, args: &[String]) -> Result<Vec<u8>, Error> {
        let output = Command::new("/usr/bin/gh")
            .args(args)
            .env("GH_HOST", "github.com")
            .output()
            .map_err(|_| Error::msg("GitHub CLI unavailable"))?;
        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            if stderr.contains("(HTTP 404)") || stderr.trim() == "release not found" {
                return Err(Error::msg("GitHub resource not found"));
            }
            return Err(Error::msg(
                "GitHub command failed; inspect the exact release before repeating publication",
            ));
        }
        if output.stdout.len() > 4 << 20 {
            return Err(Error::msg("GitHub response exceeds bound"));
        }
        Ok(output.stdout)
    }
}
fn command(gh: &dyn GitHub, args: &[&str]) -> Result<Vec<u8>, Error> {
    gh.run(&args.iter().map(|s| (*s).to_owned()).collect::<Vec<_>>())
}
fn api<T: serde::de::DeserializeOwned>(gh: &dyn GitHub, path: &str) -> Result<T, Error> {
    let bytes = command(gh, &["api", "--hostname", "github.com", path])?;
    serde_json::from_slice(&bytes).map_err(|_| Error::msg("GitHub response refused"))
}
#[derive(Deserialize)]
struct RemoteAsset {
    name: String,
    size: u64,
    digest: String,
    state: String,
}
#[derive(Deserialize)]
struct Release {
    id: u64,
    tag_name: String,
    draft: bool,
    prerelease: bool,
    immutable: bool,
    #[serde(default)]
    published_at: Option<String>,
    assets: Vec<RemoteAsset>,
}
#[derive(Deserialize)]
struct GitObject {
    #[serde(rename = "type")]
    kind: String,
    sha: String,
}
#[derive(Deserialize)]
struct TagRef {
    #[serde(rename = "ref")]
    reference: String,
    object: GitObject,
}
#[derive(Deserialize)]
struct AnnotatedTag {
    object: GitObject,
}
fn check_tag(gh: &dyn GitHub, manifest: &Manifest, create_missing: bool) -> Result<(), Error> {
    let tag: TagRef = match api(gh, &format!("{API}/git/ref/tags/{}", manifest.tag)) {
        Ok(tag) => tag,
        Err(e) if create_missing && e.0 == "GitHub resource not found" => {
            let bytes = command(
                gh,
                &[
                    "api",
                    "--hostname",
                    "github.com",
                    "--method",
                    "POST",
                    &format!("{API}/git/refs"),
                    "--raw-field",
                    &format!("ref=refs/tags/{}", manifest.tag),
                    "--raw-field",
                    &format!("sha={}", manifest.soda_revision),
                ],
            )?;
            serde_json::from_slice(&bytes).map_err(|_| Error::msg("GitHub tag response refused"))?
        }
        Err(e) => return Err(e),
    };
    if tag.reference != format!("refs/tags/{}", manifest.tag) {
        return Err(Error::msg("exact remote tag required"));
    }
    let object = if tag.object.kind == "tag" {
        let annotated: AnnotatedTag = api(gh, &format!("{API}/git/tags/{}", tag.object.sha))?;
        annotated.object
    } else {
        tag.object
    };
    if object.kind != "commit" || object.sha != manifest.soda_revision {
        return Err(Error::msg(
            "remote tag does not identify the built Soda revision",
        ));
    }
    Ok(())
}

/// Only main's latest push run for these exact sources can authorize delivery.
/// PR merge commits and an older success hidden behind a failed rerun do not qualify.
fn check_ci(gh: &dyn GitHub, manifest: &Manifest) -> Result<u64, Error> {
    check_source_ci(gh, &manifest.soda_revision, &manifest.forgejo_revision)
}

pub fn check_source_ci(
    gh: &dyn GitHub,
    soda_revision: &str,
    forgejo_revision: &str,
) -> Result<u64, Error> {
    #[derive(Deserialize)]
    struct FountainSource {
        revision: String,
    }
    let bytes = command(
        gh,
        &[
            "api",
            "--hostname",
            "github.com",
            "--header",
            "Accept: application/vnd.github.raw+json",
            &format!("{API}/contents/fountain-source.json?ref={}", soda_revision),
        ],
    )?;
    let fountain: FountainSource =
        serde_json::from_slice(&bytes).map_err(|_| Error::msg("Fountain source pin refused"))?;
    if fountain.revision != forgejo_revision {
        return Err(Error::msg(
            "built Fountain revision differs from the CI source pin",
        ));
    }
    #[derive(Deserialize)]
    struct Run {
        id: u64,
        head_sha: String,
        head_branch: Option<String>,
        event: String,
        path: String,
        status: String,
        conclusion: Option<String>,
    }
    #[derive(Deserialize)]
    struct Runs {
        workflow_runs: Vec<Run>,
    }
    let runs: Runs = api(
        gh,
        &format!(
            "{API}/actions/workflows/ci.yml/runs?head_sha={}&branch=main&event=push&per_page=1",
            soda_revision
        ),
    )?;
    let Some(run) = runs
        .workflow_runs
        .first()
        .filter(|_| runs.workflow_runs.len() == 1)
    else {
        return Err(Error::msg(
            "CI push run on main required for the built revision",
        ));
    };
    if run.id == 0
        || run.head_sha != soda_revision
        || run.head_branch.as_deref() != Some("main")
        || run.event != "push"
        || run.path != ".github/workflows/ci.yml"
        || run.status != "completed"
        || run.conclusion.as_deref() != Some("success")
    {
        return Err(Error::msg(
            "latest CI push run must pass for the exact built revision",
        ));
    }
    Ok(run.id)
}

fn release(gh: &dyn GitHub, tag: &str) -> Result<Release, Error> {
    #[derive(Deserialize)]
    struct ReleaseId {
        #[serde(rename = "databaseId")]
        id: u64,
    }
    // GitHub's REST lookup by tag excludes drafts; gh resolves both states.
    let bytes = command(
        gh,
        &[
            "release",
            "view",
            tag,
            "--repo",
            REPOSITORY,
            "--json",
            "databaseId",
        ],
    )?;
    let release: ReleaseId =
        serde_json::from_slice(&bytes).map_err(|_| Error::msg("GitHub response refused"))?;
    api(gh, &format!("{API}/releases/{}", release.id))
}
fn check_inventory(release: &Release, manifest: &Manifest, assets: &[Asset]) -> Result<(), Error> {
    if release.id == 0 || release.tag_name != manifest.tag || release.assets.len() != assets.len() {
        return Err(Error::msg("remote release inventory differs"));
    }
    for expected in assets {
        let matches: Vec<_> = release
            .assets
            .iter()
            .filter(|a| a.name == expected.name)
            .collect();
        if matches.len() != 1
            || matches[0].state != "uploaded"
            || matches[0].size != expected.bytes
            || matches[0].digest != format!("sha256:{}", expected.sha256)
        {
            return Err(Error::msg(
                "remote release asset differs from admitted bytes",
            ));
        }
    }
    Ok(())
}
fn verify_release(gh: &dyn GitHub, tag: &str) -> Result<(), Error> {
    command(gh, &["release", "verify", tag, "--repo", REPOSITORY])?;
    Ok(())
}

fn latest_release(gh: &dyn GitHub) -> Result<Option<Release>, Error> {
    match api(gh, &format!("{API}/releases/latest")) {
        Ok(release) => Ok(Some(release)),
        Err(error) if error.0 == "GitHub resource not found" => Ok(None),
        Err(error) => Err(error),
    }
}

// Repeating a verified readback may finish an interrupted local receipt write.
// Existing evidence must match byte-for-byte; this never replaces retained state.
fn record_verified(path: &Path, bytes: &[u8]) -> Result<(), Error> {
    use std::os::unix::fs::MetadataExt;
    match std::fs::symlink_metadata(path) {
        Ok(metadata)
            if metadata.is_file()
                && metadata.mode() & 0o077 == 0
                && metadata.len() == bytes.len() as u64
                && std::fs::read(path).ok().as_deref() == Some(bytes) =>
        {
            Ok(())
        }
        Ok(_) => Err(Error::msg(
            "existing publication evidence differs; inspect retained state",
        )),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
            crate::buildx::write_new(path.to_str().ok_or_else(Error::refused)?, bytes, 0o600)
        }
        Err(_) => Err(Error::msg("publication evidence unavailable")),
    }
}

#[derive(Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
struct ApplianceEvidence {
    downloaded: std::path::PathBuf,
}

fn appliance_evidence(
    directory: &Path,
    manifest: &Manifest,
) -> Result<(Qualification, std::path::PathBuf), Error> {
    let receipt: Qualification = read_json(&directory.join("appliance-qualification.json"))?;
    let input: ApplianceEvidence = read_json(&directory.join("appliance-evidence.json"))?;
    if !input.downloaded.is_absolute() {
        return Err(Error::msg(
            "absolute downloaded appliance evidence path required",
        ));
    }
    let downloaded = inspect(&input.downloaded, &manifest.iso)?;
    receipt.validate(manifest, &downloaded)?;
    Ok((receipt, input.downloaded))
}

/// Upload a complete draft, verify its inventory, and expose it as an unqualified
/// immutable prerelease. Existing releases are never adopted or overwritten.
fn admit_staging_release(
    gh: &dyn GitHub,
    manifest: &Manifest,
    directory: &Path,
) -> Result<(Vec<Asset>, u64), Error> {
    manifest.validate()?;
    let assets = inventory(manifest, directory)?;
    let ci_run = check_ci(gh, manifest)?;
    #[derive(Deserialize)]
    struct Settings {
        enabled: bool,
    }
    let settings: Settings = api(gh, &format!("{API}/immutable-releases"))?;
    if !settings.enabled {
        return Err(Error::msg(
            "enable GitHub immutable releases before staging",
        ));
    }
    match release(gh, &manifest.tag) {
        Ok(_) => {
            return Err(Error::msg(
                "release already exists; inspect it, do not replay staging",
            ))
        }
        Err(e) if e.0 == "GitHub resource not found" => (),
        Err(e) => return Err(e),
    }
    check_tag(gh, manifest, true)?;
    Ok((assets, ci_run))
}

fn publish_candidate_release(
    gh: &dyn GitHub,
    manifest: &Manifest,
    directory: &Path,
    ci_run: u64,
    assets: &[Asset],
) -> Result<(), Error> {
    let notes_path = directory.join("release-notes.md");
    std::fs::write(
        &notes_path,
        format!(
            "UNQUALIFIED CANDIDATE: native qualification of every selected distribution is pending.\n\n{}\n\nSource CI: https://github.com/LevitateOS/sodaos/actions/runs/{ci_run}\n",
            manifest.notes
        ),
    )
    .map_err(|_| Error::msg("cannot write candidate release notes"))?;
    command(
        gh,
        &[
            "release",
            "create",
            &manifest.tag,
            "--repo",
            REPOSITORY,
            "--verify-tag",
            "--draft",
            "--prerelease",
            "--latest=false",
            "--notes-file",
            notes_path.to_str().ok_or_else(Error::refused)?,
        ],
    )?;
    let mut upload = vec![
        "release".to_owned(),
        "upload".to_owned(),
        manifest.tag.clone(),
        "--repo".to_owned(),
        REPOSITORY.to_owned(),
    ];
    for a in assets {
        upload.push(
            directory
                .join(&a.name)
                .to_str()
                .ok_or_else(Error::refused)?
                .to_owned(),
        );
    }
    gh.run(&upload)?;
    Ok(())
}

fn finalize_candidate_release(
    gh: &dyn GitHub,
    manifest: &Manifest,
    assets: &[Asset],
) -> Result<(), Error> {
    let before = release(gh, &manifest.tag)?;
    if !before.draft || !before.prerelease {
        return Err(Error::msg(
            "candidate release is not the expected draft prerelease",
        ));
    }
    check_inventory(&before, manifest, assets)?;
    command(
        gh,
        &[
            "release",
            "edit",
            &manifest.tag,
            "--repo",
            REPOSITORY,
            "--verify-tag",
            "--draft=false",
            "--prerelease",
            "--latest=false",
        ],
    )?;
    let after = release(gh, &manifest.tag)?;
    if after.draft || !after.prerelease || !after.immutable {
        return Err(Error::msg(
            "candidate publication is not an immutable prerelease",
        ));
    }
    check_inventory(&after, manifest, assets)?;
    verify_release(gh, &manifest.tag)
}

fn stage_github(
    gh: &dyn GitHub,
    manifest: &Manifest,
    directory: &Path,
    before_release: impl FnOnce(u64) -> Result<(), Error>,
) -> Result<(), Error> {
    let (assets, ci_run) = admit_staging_release(gh, manifest, directory)?;
    before_release(ci_run)?;
    publish_candidate_release(gh, manifest, directory, ci_run, &assets)?;
    finalize_candidate_release(gh, manifest, &assets)
}

fn record_appliance_outcome(
    gh: &dyn GitHub,
    directory: &Path,
    tag: &str,
    downloaded_iso: std::path::PathBuf,
    receipt: &Qualification,
) -> Result<(), Error> {
    command(
        gh,
        &[
            "release",
            "verify-asset",
            tag,
            downloaded_iso.to_str().ok_or_else(Error::refused)?,
            "--repo",
            REPOSITORY,
        ],
    )?;
    let bytes = serde_json::to_vec(&ApplianceEvidence {
        downloaded: downloaded_iso,
    })
    .map_err(|_| Error::refused())?;
    record_verified(&directory.join("appliance-evidence.json"), &bytes)?;
    let bytes = serde_json::to_vec_pretty(receipt).map_err(|_| Error::refused())?;
    record_verified(&directory.join("appliance-qualification.json"), &bytes)?;
    Ok(())
}

/// Promote only the unchanged immutable candidate after exact install/boot smoke
/// checks and verification of the ISO actually downloaded from GitHub.
pub fn qualify_appliance(
    gh: &dyn GitHub,
    directory: &Path,
    downloaded_iso: &Path,
    observed_revision: &str,
    smoke_passed: bool,
) -> Result<(), Error> {
    if !smoke_passed {
        return Err(Error::msg(
            "explicit native install/boot/image smoke confirmation required",
        ));
    }
    let manifest = assets::load(directory)?;
    let downloaded = inspect(downloaded_iso, &manifest.iso)?;
    let downloaded_iso = std::fs::canonicalize(downloaded_iso)
        .map_err(|_| Error::msg("downloaded ISO unavailable"))?;
    let receipt = Qualification {
        format: 1,
        soda_revision: manifest.soda_revision.clone(),
        forgejo_revision: manifest.forgejo_revision.clone(),
        iso_sha256: manifest.asset(&manifest.iso)?.sha256.clone(),
        rootfs_sha256: manifest.asset(&manifest.rootfs)?.sha256.clone(),
        downloaded_iso_sha256: downloaded.sha256.clone(),
        observed_soda_revision: observed_revision.to_owned(),
        install: "passed".to_owned(),
        boot: "passed".to_owned(),
        applications: "passed".to_owned(),
    };
    receipt.validate(&manifest, &downloaded)?;
    check_tag(gh, &manifest, false)?;
    check_ci(gh, &manifest)?;
    let current = release(gh, &manifest.tag)?;
    if current.draft || !current.prerelease || !current.immutable {
        return Err(Error::msg(
            "only the exact immutable prerelease can be promoted",
        ));
    }
    let assets = inventory(&manifest, directory)?;
    check_inventory(&current, &manifest, &assets)?;
    verify_release(gh, &manifest.tag)?;
    record_appliance_outcome(gh, directory, &manifest.tag, downloaded_iso, &receipt)?;
    Ok(())
}

fn admit_image_staging<'a>(
    directory: &Path,
    auth: Option<&'a Path>,
) -> Result<(Manifest, Option<&'a Path>), Error> {
    let manifest = assets::load(directory)?;
    if !directory.join("components/public.json").exists()
        || (manifest.docker.is_some() && !directory.join("registry/registry.json").exists())
    {
        let auth =
            auth.ok_or_else(|| Error::msg("registry auth file required before image staging"))?;
        use std::os::unix::fs::MetadataExt;
        let st = std::fs::symlink_metadata(auth).map_err(|_| Error::refused())?;
        if !st.is_file() || st.mode() & 0o077 != 0 || st.len() == 0 {
            return Err(Error::msg(
                "restricted registry auth file required before staging",
            ));
        }
        if (directory.join("registry").exists()
            && !directory.join("registry/registry.json").exists())
            || (directory.join("components").exists()
                && !directory.join("components/public.json").exists())
        {
            return Err(Error::msg("registry upload was already attempted; run verify-images and inspect remote state instead of replaying it"));
        }
    }
    Ok((manifest, auth))
}

fn admit_staging_candidate(
    directory: &Path,
    manifest: &Manifest,
) -> Result<Option<PathBuf>, Error> {
    if manifest.docker.is_some() {
        Ok(Some(assets::candidate(directory, manifest)?))
    } else {
        Ok(None)
    }
}

pub fn stage(
    gh: &dyn GitHub,
    registry: &dyn registry::Registry,
    directory: &Path,
    auth: Option<&Path>,
) -> Result<(), Error> {
    let (manifest, auth) = admit_image_staging(directory, auth)?;
    let candidate = admit_staging_candidate(directory, &manifest)?;
    stage_github(gh, &manifest, directory, |ci_run| {
        components::stage(gh, registry, directory, &manifest, auth)?;
        if let Some(metadata) = &manifest.docker {
            let output = directory.join("registry");
            if output.join("registry.json").exists() {
                registry::observe_public(registry, metadata, ci_run)?;
            } else {
                registry::stage_admitted(
                    gh,
                    registry,
                    candidate.as_deref().ok_or_else(Error::refused)?,
                    auth.ok_or_else(|| {
                        Error::msg("registry auth file required for Docker staging")
                    })?,
                    &output,
                    metadata,
                )?;
            }
        }
        Ok(())
    })
}

/// Recheck every selected target before moving either recommendation pointer.
pub fn promote(
    gh: &dyn GitHub,
    registry: &dyn registry::Registry,
    directory: &Path,
    auth: Option<&Path>,
) -> Result<(), Error> {
    let manifest = assets::load(directory)?;
    if directory.join("promotion-started.json").exists() {
        return Err(Error::msg("promotion already attempted; inspect status and remote pointers, do not replay or roll back"));
    }
    let mut summaries = serde_json::Map::new();
    let mut downloaded_iso = None;
    if manifest.has_appliance() {
        let (receipt, downloaded) = appliance_evidence(directory, &manifest)?;
        summaries.insert(
            "appliance".into(),
            serde_json::to_value(&receipt).map_err(|_| Error::refused())?,
        );
        downloaded_iso = Some(downloaded);
    }
    if manifest.docker.is_some() {
        let receipt = qualification::verify(&manifest, directory)?;
        summaries.insert(
            "docker".into(),
            serde_json::to_value(&receipt).map_err(|_| Error::refused())?,
        );
        let file =
            auth.ok_or_else(|| Error::msg("registry auth file required for Docker promotion"))?;
        let st = std::fs::symlink_metadata(file).map_err(|_| Error::refused())?;
        use std::os::unix::fs::MetadataExt;
        if !st.is_file() || st.mode() & 0o077 != 0 || st.len() == 0 {
            return Err(Error::msg("restricted registry auth file required"));
        }
    }
    check_tag(gh, &manifest, false)?;
    let ci_run = check_ci(gh, &manifest)?;
    let current = release(gh, &manifest.tag)?;
    if current.draft || !current.prerelease || !current.immutable {
        return Err(Error::msg(
            "only the exact immutable prerelease can be promoted",
        ));
    }
    let assets = inventory(&manifest, directory)?;
    check_inventory(&current, &manifest, &assets)?;
    verify_release(gh, &manifest.tag)?;
    if let Some(downloaded) = &downloaded_iso {
        command(
            gh,
            &[
                "release",
                "verify-asset",
                &manifest.tag,
                downloaded.to_str().ok_or_else(Error::refused)?,
                "--repo",
                REPOSITORY,
            ],
        )?;
    }
    if let Some(metadata) = &manifest.docker {
        qualification::validate_publication_time(
            &manifest,
            directory,
            &qualification::evidence(directory)?,
            current
                .published_at
                .as_deref()
                .ok_or_else(|| Error::msg("immutable publication time unavailable"))?,
        )?;
        registry::observe_public(registry, metadata, ci_run)?;
    }
    components::observe_public(registry, directory, &manifest)?;
    let prior_appliance_latest = if !manifest.has_appliance() {
        latest_release(gh)?.map(|release| release.id)
    } else {
        None
    };
    let notes = directory.join("qualified-release-notes.md");
    let summary = serde_json::to_string_pretty(&summaries).map_err(|_| Error::refused())?;
    let docker_qualification = manifest
        .docker
        .as_ref()
        .map(|metadata| {
            format!(
                "\n<!-- soda-docker-qualified: {} {} {} -->\n",
                manifest.tag, manifest.soda_revision, metadata.image.manifest
            )
        })
        .unwrap_or_default();
    std::fs::write(&notes, format!("{}\n\nSource CI: https://github.com/LevitateOS/sodaos/actions/runs/{ci_run}\n\nNative qualification passed for every selected distribution and these exact release bytes.\n{docker_qualification}\n\x60\x60\x60json\n{summary}\n\x60\x60\x60\n", manifest.notes)).map_err(|_| Error::refused())?;
    crate::buildx::write_new(directory.join("promotion-started.json").to_str().ok_or_else(Error::refused)?, summary.as_bytes(), 0o600)
        .map_err(|_| Error::msg("promotion already attempted; inspect status and remote pointers, do not replay or roll back"))?;
    if let Some(metadata) = &manifest.docker {
        registry::promote_digest(registry, metadata, auth.ok_or_else(Error::refused)?, directory)
            .map_err(|_| Error::msg("Docker recommendation may have changed; inspect partial promotion with status before any further mutation"))?;
    }
    command(gh, &["release", "edit", &manifest.tag, "--repo", REPOSITORY, "--verify-tag", "--prerelease=false", if manifest.has_appliance() { "--latest=true" } else { "--latest=false" }, "--notes-file", notes.to_str().ok_or_else(Error::refused)?])
        .map_err(|_| Error::msg("GitHub promotion failed after recommendation update; inspect status, do not replay or roll back"))?;
    let after = release(gh, &manifest.tag)?;
    if after.id != current.id || after.draft || after.prerelease || !after.immutable {
        return Err(Error::msg(
            "qualified release state differs; inspect partial promotion",
        ));
    }
    check_inventory(&after, &manifest, &assets)?;
    if manifest.has_appliance() {
        let latest: Release = api(gh, &format!("{API}/releases/latest"))?;
        if latest.id != current.id {
            return Err(Error::msg(
                "appliance latest pointer differs; inspect partial promotion",
            ));
        }
    } else if latest_release(gh)?.map(|release| release.id) != prior_appliance_latest {
        return Err(Error::msg("Docker promotion changed the appliance latest pointer; inspect partial promotion without rollback"));
    }
    crate::buildx::write_new(
        directory
            .join("promotion-complete.json")
            .to_str()
            .ok_or_else(Error::refused)?,
        summary.as_bytes(),
        0o600,
    )?;
    Ok(())
}

fn classify_publication(
    remote: &Result<Release, Error>,
    manifest: &Manifest,
    directory: &Path,
    gh: &dyn GitHub,
    blockers: &mut Vec<String>,
) -> Result<&'static str, Error> {
    let publication = match remote {
        Ok(ref remote) if remote.draft => {
            blockers
                .push("partial staging: inspect the existing draft before further mutation".into());
            if check_inventory(remote, manifest, &inventory(manifest, directory)?).is_err() {
                blockers.push("draft release inventory is incomplete or differs".into());
            }
            "draft"
        }
        Ok(ref remote)
            if check_inventory(remote, manifest, &inventory(manifest, directory)?).is_ok() =>
        {
            if !remote.immutable {
                blockers.push("remote release is not immutable".into());
                "mismatched"
            } else if let Err(error) =
                check_tag(gh, manifest, false).and_then(|_| verify_release(gh, &manifest.tag))
            {
                blockers.push(error.0);
                "unverified"
            } else if remote.prerelease {
                "unqualified prerelease"
            } else {
                "published release"
            }
        }
        Ok(_) => {
            blockers.push("remote release inventory differs".into());
            "mismatched"
        }
        Err(ref error) if error.0 == "GitHub resource not found" => "not observed",
        Err(ref error) => {
            blockers.push(error.0.clone());
            "unavailable"
        }
    };
    Ok(publication)
}

fn validate_qualifications(
    manifest: &Manifest,
    directory: &Path,
    remote: &Result<Release, Error>,
    blockers: &mut Vec<String>,
) -> Result<(bool, bool), Error> {
    let appliance = !manifest.has_appliance() || appliance_evidence(directory, manifest).is_ok();
    let docker_qualified = manifest.docker.is_none()
        || (|| -> Result<(), Error> {
            qualification::verify(manifest, directory)?;
            let published = remote.as_ref().map_err(|_| Error::refused())?;
            if published.draft || !published.immutable {
                return Err(Error::refused());
            }
            qualification::validate_publication_time(
                manifest,
                directory,
                &qualification::evidence(directory)?,
                published
                    .published_at
                    .as_deref()
                    .ok_or_else(Error::refused)?,
            )
        })()
        .is_ok();
    if !appliance {
        blockers.push("appliance qualification missing or invalid".into());
    }
    if !docker_qualified {
        blockers.push("Docker qualification missing or invalid".into());
    }
    Ok((appliance, docker_qualified))
}

fn observe_recommendation(
    gh: &dyn GitHub,
    registry: &dyn registry::Registry,
    directory: &Path,
    manifest: &Manifest,
    blockers: &mut Vec<String>,
) -> (bool, Option<String>, Option<String>, bool) {
    let partial = directory.join("promotion-started.json").exists()
        && !directory.join("promotion-complete.json").exists();
    if partial {
        blockers
            .push("partial promotion: inspect both remote pointers before further mutation".into());
    }
    let github_latest = match latest_release(gh) {
        Ok(release) => release.map(|release| release.tag_name),
        Err(error) => {
            blockers.push(format!("appliance recommendation unavailable: {}", error.0));
            None
        }
    };
    let docker_latest = if manifest.docker.is_some() {
        registry::latest_digest(registry).ok()
    } else {
        None
    };
    let promotion_complete = directory.join("promotion-complete.json").exists();
    if promotion_complete
        && manifest.has_appliance()
        && github_latest.as_deref() != Some(manifest.tag.as_str())
    {
        blockers.push("appliance recommendation no longer matches this release".into());
    }
    if promotion_complete
        && manifest.docker.as_ref().is_some_and(|metadata| {
            docker_latest.as_deref() != Some(metadata.image.manifest.as_str())
        })
    {
        blockers.push("Docker recommendation no longer matches this release".into());
    }
    (partial, github_latest, docker_latest, promotion_complete)
}

fn next_action(
    recommendation: &(bool, Option<String>, Option<String>, bool),
    ci_ok: bool,
    publication: &str,
    docker_available: bool,
    components_ok: bool,
    appliance: bool,
    docker_qualified: bool,
    directory: &Path,
) -> &'static str {
    let partial = recommendation.0;
    let github_latest = recommendation.1.as_deref();
    let docker_latest = recommendation.2.as_deref();
    let promotion_complete = recommendation.3;
    if partial {
        "inspect partial promotion"
    } else if !ci_ok {
        "obtain successful exact-source main CI"
    } else if publication == "mismatched" {
        "inspect immutable remote inventory"
    } else if publication == "draft" {
        "inspect partial staging"
    } else if matches!(publication, "unavailable" | "unverified") {
        "inspect remote publication"
    } else if promotion_complete {
        "inspect recommendation state"
    } else if publication == "published release" {
        "inspect existing promotion"
    } else if (directory.join("registry").exists()
        && !directory.join("registry/registry.json").exists())
        || (directory.join("components").exists()
            && !directory.join("components/public.json").exists())
    {
        "verify-images"
    } else if publication == "not observed" {
        "stage"
    } else if !docker_available {
        "inspect public Docker image"
    } else if !components_ok {
        "inspect public component images"
    } else if !appliance || !docker_qualified {
        "qualify"
    } else {
        "promote"
    }
}

pub fn status(
    gh: &dyn GitHub,
    registry: &dyn registry::Registry,
    directory: &Path,
) -> Result<serde_json::Value, Error> {
    let manifest = assets::load(directory)?;
    let ci = check_ci(gh, &manifest);
    let remote = release(gh, &manifest.tag);
    let docker = manifest.docker.as_ref().map(|metadata| {
        registry::observe_public(registry, metadata, ci.as_ref().copied().unwrap_or(0))
    });
    let mut blockers = Vec::new();
    let components = components::observe_public(registry, directory, &manifest);
    if let Err(error) = &components {
        blockers.push(error.0.clone());
    }
    if let Err(error) = &ci {
        blockers.push(error.0.clone());
    }
    let publication = classify_publication(&remote, &manifest, directory, gh, &mut blockers)?;
    let docker_available = !matches!(docker, Some(Err(_)));
    if let Some(Err(error)) = &docker {
        blockers.push(error.0.clone());
    }
    let (appliance, docker_qualified) =
        validate_qualifications(&manifest, directory, &remote, &mut blockers)?;
    let recommendation = observe_recommendation(gh, registry, directory, &manifest, &mut blockers);
    let (_, github_latest, docker_latest, promotion_complete) = &recommendation;
    let next = next_action(
        &recommendation,
        ci.is_ok(),
        publication,
        docker_available,
        components.is_ok(),
        appliance,
        docker_qualified,
        directory,
    );
    Ok(
        serde_json::json!({"tag":manifest.tag,"distribution":manifest.distribution,"soda_revision":manifest.soda_revision,"publication":publication,"github_latest":github_latest,"docker_latest_digest":docker_latest,"appliance_qualified":if manifest.has_appliance(){Some(appliance)}else{None},"docker_qualified":if manifest.docker.is_some(){Some(docker_qualified)}else{None},"promotion_complete":promotion_complete,"blockers":blockers,"next_action":next}),
    )
}
#[cfg(test)]
mod tests;

pub fn verify_registry(
    gh: &dyn GitHub,
    registry: &dyn registry::Registry,
    directory: &Path,
) -> Result<(), Error> {
    let manifest = assets::load(directory)?;
    check_ci(gh, &manifest)?;
    components::verify(registry, directory, &manifest)?;
    if manifest.docker.is_some() {
        let candidate = assets::candidate(directory, &manifest)?;
        registry::verify(gh, registry, &candidate, &directory.join("registry"))?;
    }
    Ok(())
}

fn check_release_absence(gh: &dyn GitHub, tag: &str) -> Result<(), Error> {
    match release(gh, tag) {
        Ok(_) => return Err(Error::msg("release already exists; choose a fresh version")),
        Err(error) if error.0 == "GitHub resource not found" => (),
        Err(error) => return Err(error),
    }
    Ok(())
}

fn check_tag_reference(gh: &dyn GitHub, tag: &str, soda: &str) -> Result<(), Error> {
    match api::<TagRef>(gh, &format!("{API}/git/ref/tags/{tag}")) {
        Ok(reference) if reference.object.kind == "commit" && reference.object.sha == soda => (),
        Ok(_) => return Err(Error::msg("existing tag differs from selected source")),
        Err(error) if error.0 == "GitHub resource not found" => (),
        Err(error) => return Err(error),
    }
    Ok(())
}

fn check_package_visibility(
    gh: &dyn GitHub,
    distribution: crate::model::Distribution,
) -> Result<(), Error> {
    let mut packages = vec![
        "sodaos-dashboard",
        "sodaos-forgejo",
        "sodaos-project-os",
        "sodaos-canlang-runtime",
    ];
    if distribution != crate::model::Distribution::Appliance {
        packages.push("sodaos");
    }
    for name in packages {
        let package =
            api::<serde_json::Value>(gh, &format!("orgs/LevitateOS/packages/container/{name}"));
        match package {
            Ok(value) if value.get("visibility").and_then(|v|v.as_str())!=Some("public")=>return Err(Error::msg("GHCR package must be public before staging; change its package visibility")),
            Ok(_) =>(),Err(error) if error.0=="GitHub resource not found"=>eprintln!("GHCR package does not exist; first upload requires public visibility before anonymous verification."),Err(error)=>return Err(error),
        }
    }
    Ok(())
}

pub fn preflight(
    gh: &dyn GitHub,
    soda: &str,
    fountain: &str,
    tag: &str,
    distribution: crate::model::Distribution,
) -> Result<(), Error> {
    if !valid_tag(tag) {
        return Err(Error::msg("exact release tag required"));
    }
    check_source_ci(gh, soda, fountain)?;
    #[derive(Deserialize)]
    struct Settings {
        enabled: bool,
    }
    if !api::<Settings>(gh, &format!("{API}/immutable-releases"))?.enabled {
        return Err(Error::msg(
            "enable immutable GitHub releases before building",
        ));
    }
    check_release_absence(gh, tag)?;
    check_tag_reference(gh, tag, soda)?;
    check_package_visibility(gh, distribution)
}

fn verify_downloaded_assets(
    gh: &dyn GitHub,
    manifest: &Manifest,
    evidence: &Path,
) -> Result<(), Error> {
    for name in qualification::DOWNLOADS {
        let downloaded = evidence.join("downloads").join(name);
        if &inspect(&downloaded, name)? != manifest.asset(name)? {
            return Err(Error::msg(
                "downloaded installation bytes differ from staged release",
            ));
        }
        command(
            gh,
            &[
                "release",
                "verify-asset",
                &manifest.tag,
                downloaded.to_str().ok_or_else(Error::refused)?,
                "--repo",
                REPOSITORY,
            ],
        )?;
    }
    Ok(())
}

pub fn qualify_docker(
    gh: &dyn GitHub,
    registry: &dyn registry::Registry,
    manifest: &Manifest,
    directory: &Path,
    evidence: &Path,
) -> Result<(), Error> {
    check_tag(gh, manifest, false)?;
    let ci = check_ci(gh, manifest)?;
    let remote = release(gh, &manifest.tag)?;
    if remote.draft || !remote.prerelease || !remote.immutable {
        return Err(Error::msg(
            "exact immutable unqualified prerelease required for qualification",
        ));
    }
    check_inventory(&remote, manifest, &inventory(manifest, directory)?)?;
    verify_release(gh, &manifest.tag)?;
    qualification::validate_publication_time(
        manifest,
        directory,
        evidence,
        remote
            .published_at
            .as_deref()
            .ok_or_else(|| Error::msg("immutable publication time unavailable"))?,
    )?;
    registry::observe_public(
        registry,
        manifest.docker.as_ref().ok_or_else(Error::refused)?,
        ci,
    )?;
    verify_downloaded_assets(gh, manifest, evidence)?;
    if directory.join("docker-qualification.json").exists() {
        qualification::verify(manifest, directory)?;
    } else {
        qualification::stamp(manifest, directory, evidence)?;
    }
    Ok(())
}
