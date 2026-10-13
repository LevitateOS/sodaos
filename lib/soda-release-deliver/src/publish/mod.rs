//! GitHub Releases owns publication state; local artifacts remain immutable inputs.
use crate::Error;
use serde::Deserialize;
use std::path::Path;
use std::process::Command;
pub mod assets;
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
            if String::from_utf8_lossy(&output.stderr).contains("(HTTP 404)") {
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
fn check_tag(gh: &dyn GitHub, manifest: &Manifest) -> Result<(), Error> {
    let tag: TagRef = api(gh, &format!("{API}/git/ref/tags/{}", manifest.tag))?;
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
fn release(gh: &dyn GitHub, tag: &str) -> Result<Release, Error> {
    api(gh, &format!("{API}/releases/tags/{tag}"))
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
    check_tag(gh, manifest)?;
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
    let notes_path = directory.join("release-notes.md");
    std::fs::write(
        &notes_path,
        format!(
            "UNQUALIFIED CANDIDATE: install and first-boot checks pending.\n\n{}",
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
    qualification_path: &Path,
    downloaded_iso: &Path,
) -> Result<(), Error> {
    let manifest: Manifest = read_json(&directory.join("release.json"))?;
    manifest.validate()?;
    let receipt: Qualification = read_json(qualification_path)?;
    let downloaded = inspect(downloaded_iso, &manifest.iso)?;
    receipt.validate(&manifest, &downloaded)?;
    check_tag(gh, &manifest)?;
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
    let notes = directory.join("qualified-release-notes.md");
    let receipt_text = serde_json::to_string_pretty(&receipt).map_err(|_| Error::refused())?;
    std::fs::write(&notes, format!("{}\n\nNative install and first-boot smoke checks passed for the unchanged release bytes.\n\n```json\n{receipt_text}\n```\n", manifest.notes)).map_err(|_| Error::msg("cannot write qualified notes"))?;
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
