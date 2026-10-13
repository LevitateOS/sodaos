//! Exact release files and the bounded install/boot promotion receipt.
use crate::{model::MediaBinding, Error};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs::{self, File};
use std::io::{Read, Write};
use std::path::Path;

pub const MAX_ASSET_BYTES: u64 = 2 * 1024 * 1024 * 1024;
pub const ISO_NAME: &str = "sodaos-x86_64.iso";
const REQUIRED_METADATA: [&str; 8] = [
    "sodaos-source.tar",
    "fountain-source.tar",
    "payload.json",
    "candidate.json",
    "media.json",
    "app-inputs.json",
    "packages.txt",
    "presentation.json",
];

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub struct Asset {
    pub name: String,
    pub bytes: u64,
    pub sha256: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Manifest {
    pub format: u32,
    pub tag: String,
    pub soda_revision: String,
    pub forgejo_revision: String,
    pub iso: String,
    pub rootfs: String,
    pub notes: String,
    pub assets: Vec<Asset>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Qualification {
    pub format: u32,
    pub soda_revision: String,
    pub forgejo_revision: String,
    pub iso_sha256: String,
    pub rootfs_sha256: String,
    pub downloaded_iso_sha256: String,
    pub observed_soda_revision: String,
    pub install: String,
    pub boot: String,
    pub applications: String,
}

fn failure(message: &str) -> Error {
    Error::msg(message)
}
pub(super) fn read_json<T: serde::de::DeserializeOwned>(path: &Path) -> Result<T, Error> {
    let bytes = crate::document::read_file(
        path.to_str().ok_or_else(|| failure("invalid path"))?,
        1 << 20,
    )?;
    serde_json::from_slice(&bytes).map_err(|_| failure("release metadata refused"))
}
pub(super) fn write_json<T: Serialize>(path: &Path, value: &T) -> Result<(), Error> {
    let bytes =
        serde_json::to_vec_pretty(value).map_err(|_| failure("release metadata refused"))?;
    fs::write(path, bytes).map_err(|_| failure("cannot write release metadata"))
}
pub(super) fn valid_tag(tag: &str) -> bool {
    !tag.is_empty()
        && tag.len() <= 128
        && tag.as_bytes()[0].is_ascii_alphanumeric()
        && tag
            .bytes()
            .all(|c| c.is_ascii_alphanumeric() || b"._-".contains(&c))
        && !tag.contains("..")
}
fn basename(name: &str) -> bool {
    !name.is_empty()
        && name != "."
        && name != ".."
        && name
            .bytes()
            .all(|c| c.is_ascii_alphanumeric() || b"._-".contains(&c))
}
fn checked_file(path: &Path) -> Result<File, Error> {
    let meta = fs::symlink_metadata(path).map_err(|_| failure("release asset unavailable"))?;
    if !meta.is_file() || meta.len() == 0 || meta.len() >= MAX_ASSET_BYTES {
        return Err(failure(
            "release asset must be a nonempty regular file under 2 GiB",
        ));
    }
    File::open(path).map_err(|_| failure("release asset unavailable"))
}
fn hash_reader(mut source: File, mut target: Option<&mut File>) -> Result<(u64, String), Error> {
    let mut hash = Sha256::new();
    let mut bytes = 0;
    let mut buffer = [0u8; 64 * 1024];
    loop {
        let n = source
            .read(&mut buffer)
            .map_err(|_| failure("release asset read failed"))?;
        if n == 0 {
            break;
        }
        bytes += n as u64;
        if bytes >= MAX_ASSET_BYTES {
            return Err(failure("release asset exceeds GitHub limit"));
        }
        hash.update(&buffer[..n]);
        if let Some(file) = target.as_mut() {
            file.write_all(&buffer[..n])
                .map_err(|_| failure("release asset copy failed"))?;
        }
    }
    Ok((bytes, format!("{:x}", hash.finalize())))
}
pub(super) fn inspect(path: &Path, name: &str) -> Result<Asset, Error> {
    if !basename(name) {
        return Err(failure("invalid release asset name"));
    }
    let (bytes, sha256) = hash_reader(checked_file(path)?, None)?;
    Ok(Asset {
        name: name.to_owned(),
        bytes,
        sha256,
    })
}
fn copy_asset(source: &Path, out: &Path, name: &str) -> Result<Asset, Error> {
    if !basename(name) {
        return Err(failure("invalid release asset name"));
    }
    let input = checked_file(source)?;
    let mut output = File::options()
        .create_new(true)
        .write(true)
        .open(out.join(name))
        .map_err(|_| failure("release destination occupied"))?;
    let (bytes, sha256) = hash_reader(input, Some(&mut output))?;
    Ok(Asset {
        name: name.to_owned(),
        bytes,
        sha256,
    })
}
impl Manifest {
    pub(super) fn asset(&self, name: &str) -> Result<&Asset, Error> {
        self.assets
            .iter()
            .find(|a| a.name == name)
            .ok_or_else(|| failure("required release asset missing"))
    }
    pub(super) fn validate(&self) -> Result<(), Error> {
        let mut names = std::collections::BTreeSet::new();
        if self.format != 1
            || !valid_tag(&self.tag)
            || !crate::buildx::is_revision(&self.soda_revision)
            || !crate::buildx::is_revision(&self.forgejo_revision)
            || self.iso != ISO_NAME
            || self.notes.trim().is_empty()
            || self.assets.len() != REQUIRED_METADATA.len() + 2
        {
            return Err(failure("release manifest refused"));
        }
        for a in &self.assets {
            if !basename(&a.name)
                || !names.insert(&a.name)
                || a.bytes == 0
                || a.bytes >= MAX_ASSET_BYTES
                || !crate::buildx::is_digest(&a.sha256)
            {
                return Err(failure("release asset inventory refused"));
            }
        }
        self.asset(&self.iso)?;
        let rootfs = self.asset(&self.rootfs)?;
        if rootfs.name != format!("{}-rootfs.img", rootfs.sha256) {
            return Err(failure("rootfs asset must be hash-named"));
        }
        for name in REQUIRED_METADATA {
            self.asset(name)?;
        }
        Ok(())
    }
}
impl Qualification {
    pub(super) fn validate(&self, manifest: &Manifest, downloaded: &Asset) -> Result<(), Error> {
        let iso = manifest.asset(&manifest.iso)?;
        let rootfs = manifest.asset(&manifest.rootfs)?;
        if self.format != 1
            || self.soda_revision != manifest.soda_revision
            || self.forgejo_revision != manifest.forgejo_revision
            || self.observed_soda_revision != manifest.soda_revision
            || self.iso_sha256 != iso.sha256
            || self.rootfs_sha256 != rootfs.sha256
            || self.downloaded_iso_sha256 != iso.sha256
            || downloaded.sha256 != iso.sha256
            || downloaded.bytes != iso.bytes
            || self.install != "passed"
            || self.boot != "passed"
            || self.applications != "passed"
        {
            return Err(failure(
                "native install/boot receipt does not qualify these exact release bytes",
            ));
        }
        Ok(())
    }
}

/// Stage exact bytes locally; this does not qualify or publish a release.
pub fn prepare(
    candidate: &Path,
    media_path: &Path,
    tag: &str,
    notes_path: &Path,
    out: &Path,
) -> Result<Manifest, Error> {
    if !valid_tag(tag) {
        return Err(failure("exact release tag required"));
    }
    let candidate_path = candidate
        .to_str()
        .ok_or_else(|| failure("invalid candidate path"))?;
    let (payload, c, _) = crate::admission::load_admitted_candidate(candidate_path)?;
    let media = MediaBinding::decode_lenient(&crate::document::read_file(
        media_path
            .to_str()
            .ok_or_else(|| failure("invalid media path"))?,
        1 << 20,
    )?)
    .map_err(|_| failure("media metadata refused"))?;
    if !crate::model::valid_media_binding(&media, &payload, &c) || payload.architecture != "x86_64"
    {
        return Err(failure("media does not bind the x86_64 candidate"));
    }
    let rootfs = Path::new(&media.rootfs.path)
        .file_name()
        .and_then(|s| s.to_str())
        .ok_or_else(|| failure("invalid rootfs path"))?
        .to_owned();
    if media.iso.path != "installer.iso"
        || media.rootfs.path != rootfs
        || rootfs != format!("{}-rootfs.img", media.rootfs.sha256)
    {
        return Err(failure(
            "media must name the producer ISO and hash-bound rootfs",
        ));
    }
    if media.rootfs_url
        != format!("https://github.com/LevitateOS/sodaos/releases/download/{tag}/{rootfs}")
    {
        return Err(failure(
            "media rootfs URL must name this exact GitHub tag and asset",
        ));
    }
    let notes = String::from_utf8(crate::document::read_file(
        notes_path
            .to_str()
            .ok_or_else(|| failure("invalid notes path"))?,
        64 << 10,
    )?)
    .map_err(|_| failure("release notes must be UTF-8"))?;
    if notes.trim().is_empty() {
        return Err(failure("release notes required"));
    }
    fs::create_dir(out).map_err(|_| failure("new release output directory required"))?;
    let parent = media_path
        .parent()
        .ok_or_else(|| failure("media parent required"))?;
    let mut assets = vec![
        copy_asset(&parent.join(&media.iso.path), out, ISO_NAME)?,
        copy_asset(&parent.join(&media.rootfs.path), out, &rootfs)?,
    ];
    if assets[0].sha256 != media.iso.sha256
        || assets[0].bytes != media.iso.bytes as u64
        || assets[1].sha256 != media.rootfs.sha256
        || assets[1].bytes != media.rootfs.bytes as u64
    {
        return Err(failure("media bytes differ from their recorded hashes"));
    }
    for (source, name) in [
        ("source.tar", "sodaos-source.tar"),
        ("forgejo-source.tar", "fountain-source.tar"),
        ("payload.json", "payload.json"),
        ("candidate.json", "candidate.json"),
        ("app-inputs.json", "app-inputs.json"),
        ("packages.txt", "packages.txt"),
        ("forgejo-context/presentation.json", "presentation.json"),
    ] {
        assets.push(copy_asset(&candidate.join(source), out, name)?);
    }
    for (name, expected) in [
        ("sodaos-source.tar", payload.revision.as_str()),
        ("fountain-source.tar", c.forgejo_revision.as_str()),
    ] {
        let source =
            File::open(out.join(name)).map_err(|_| failure("release source unavailable"))?;
        let observed = std::process::Command::new("git")
            .arg("get-tar-commit-id")
            .stdin(source)
            .output()
            .map_err(|_| failure("git archive revision inspection unavailable"))?;
        if !observed.status.success()
            || String::from_utf8_lossy(&observed.stdout).trim() != expected
        {
            return Err(failure(
                "source archive does not identify the compiled revision",
            ));
        }
    }
    if assets
        .iter()
        .find(|a| a.name == "fountain-source.tar")
        .unwrap()
        .sha256
        != c.forgejo_source_sha256
    {
        return Err(failure(
            "Fountain source differs from the compiled candidate",
        ));
    }
    assets.push(copy_asset(media_path, out, "media.json")?);
    let manifest = Manifest {
        format: 1,
        tag: tag.to_owned(),
        soda_revision: payload.revision,
        forgejo_revision: c.forgejo_revision,
        iso: ISO_NAME.to_owned(),
        rootfs,
        notes,
        assets,
    };
    manifest.validate()?;
    write_json(&out.join("release.json"), &manifest)?;
    let mut sums: Vec<Asset> = manifest.assets.clone();
    sums.push(inspect(&out.join("release.json"), "release.json")?);
    let checksums = sums
        .iter()
        .map(|a| format!("{}  {}\n", a.sha256, a.name))
        .collect::<String>();
    fs::write(out.join("SHA256SUMS"), checksums).map_err(|_| failure("cannot write checksums"))?;
    Ok(manifest)
}
pub(super) fn inventory(manifest: &Manifest, directory: &Path) -> Result<Vec<Asset>, Error> {
    let mut assets = manifest.assets.clone();
    for name in ["release.json", "SHA256SUMS"] {
        assets.push(inspect(&directory.join(name), name)?);
    }
    Ok(assets)
}
