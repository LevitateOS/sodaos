//! GitHub Releases owns publication state; local artifacts remain immutable inputs.
use crate::Error;
use serde::Deserialize;
use std::path::Path;
use std::process::Command;
pub mod assets;
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
            &format!(
                "{API}/contents/fountain-source.json?ref={}",
                manifest.soda_revision
            ),
        ],
    )?;
    let fountain: FountainSource =
        serde_json::from_slice(&bytes).map_err(|_| Error::msg("Fountain source pin refused"))?;
    if fountain.revision != manifest.forgejo_revision {
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
            manifest.soda_revision
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
        || run.head_sha != manifest.soda_revision
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

/// Upload a complete draft, verify its inventory, and expose it as an unqualified
/// immutable prerelease. Existing releases are never adopted or overwritten.
pub fn stage(gh: &dyn GitHub, manifest: &Manifest, directory: &Path) -> Result<(), Error> {
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
    let notes_path = directory.join("release-notes.md");
    std::fs::write(
        &notes_path,
        format!(
            "UNQUALIFIED CANDIDATE: install and first-boot checks pending.\n\n{}\n\nSource CI: https://github.com/LevitateOS/sodaos/actions/runs/{ci_run}\n",
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
    for a in &assets {
        upload.push(
            directory
                .join(&a.name)
                .to_str()
                .ok_or_else(Error::refused)?
                .to_owned(),
        );
    }
    gh.run(&upload)?;
    let before = release(gh, &manifest.tag)?;
    if !before.draft || !before.prerelease {
        return Err(Error::msg(
            "candidate release is not the expected draft prerelease",
        ));
    }
    check_inventory(&before, manifest, &assets)?;
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
    check_inventory(&after, manifest, &assets)?;
    verify_release(gh, &manifest.tag)
}

/// Promote only the unchanged immutable candidate after exact install/boot smoke
/// checks and verification of the ISO actually downloaded from GitHub.
pub fn promote(
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
    let manifest: Manifest = read_json(&directory.join("release.json"))?;
    manifest.validate()?;
    let downloaded = inspect(downloaded_iso, &manifest.iso)?;
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
    let ci_run = check_ci(gh, &manifest)?;
    let current = release(gh, &manifest.tag)?;
    if current.draft || !current.prerelease || !current.immutable {
        return Err(Error::msg(
            "only the exact immutable prerelease can be promoted",
        ));
    }
    let assets = inventory(&manifest, directory)?;
    check_inventory(&current, &manifest, &assets)?;
    command(
        gh,
        &[
            "release",
            "verify-asset",
            &manifest.tag,
            downloaded_iso.to_str().ok_or_else(Error::refused)?,
            "--repo",
            REPOSITORY,
        ],
    )?;
    assets::write_json(&directory.join("qualification.json"), &receipt)?;
    let notes = directory.join("qualified-release-notes.md");
    let receipt_text = serde_json::to_string_pretty(&receipt).map_err(|_| Error::refused())?;
    std::fs::write(&notes, format!("{}\n\nSource CI: https://github.com/LevitateOS/sodaos/actions/runs/{ci_run}\n\nNative install and first-boot smoke checks passed for the unchanged release bytes.\n\n```json\n{receipt_text}\n```\n", manifest.notes)).map_err(|_| Error::msg("cannot write qualified notes"))?;
    command(
        gh,
        &[
            "release",
            "edit",
            &manifest.tag,
            "--repo",
            REPOSITORY,
            "--verify-tag",
            "--prerelease=false",
            "--latest",
            "--notes-file",
            notes.to_str().ok_or_else(Error::refused)?,
        ],
    )?;
    let latest: Release = api(gh, &format!("{API}/releases/latest"))?;
    if latest.id != current.id || latest.draft || latest.prerelease || !latest.immutable {
        return Err(Error::msg(
            "qualified release is not the expected latest release",
        ));
    }
    check_inventory(&latest, &manifest, &assets)?;
    Ok(())
}
#[cfg(test)]
mod tests;
