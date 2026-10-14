//! Exact release files and the bounded install/boot promotion receipt.
use crate::{
    model::{Distribution, DockerMetadata, MediaBinding},
    Error,
};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs::{self, File};
use std::io::{Read, Write};
use std::path::Path;

pub const MAX_ASSET_BYTES: u64 = 2 * 1024 * 1024 * 1024;
pub const ISO_NAME: &str = "sodaos-x86_64.iso";
const REQUIRED_METADATA: [&str; 7] = [
    "sodaos-source.tar",
    "fountain-source.tar",
    "payload.json",
    "candidate.json",
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
    pub distribution: Distribution,
    pub docker: Option<DockerMetadata>,
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
    crate::json_serde::strict(&bytes).map_err(|_| failure("release metadata refused"))
}
pub(super) fn write_json<T: Serialize>(path: &Path, value: &T) -> Result<(), Error> {
    let bytes =
        serde_json::to_vec_pretty(value).map_err(|_| failure("release metadata refused"))?;
    fs::write(path, bytes).map_err(|_| failure("cannot write release metadata"))
}
pub(crate) fn valid_tag(tag: &str) -> bool {
    soda_build_tools::reader::settings::release_tag(tag)
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
        .read(true)
        .write(true)
        .open(out.join(name))
        .map_err(|_| failure("release destination occupied"))?;
    // Independent copy-on-write files avoid writing the complete media again.
    // Hash the destination after cloning; ordinary filesystems keep one-pass copy.
    let (bytes, sha256) = if rustix::fs::ioctl_ficlone(&output, &input).is_ok() {
        hash_reader(output, None)?
    } else {
        hash_reader(input, Some(&mut output))?
    };
    Ok(Asset {
        name: name.to_owned(),
        bytes,
        sha256,
    })
}
impl Manifest {
    pub fn has_appliance(&self) -> bool {
        self.distribution != Distribution::Docker
    }
    pub(super) fn asset(&self, name: &str) -> Result<&Asset, Error> {
        self.assets
            .iter()
            .find(|a| a.name == name)
            .ok_or_else(|| failure("required release asset missing"))
    }
    pub(super) fn validate(&self) -> Result<(), Error> {
        let mut names = std::collections::BTreeSet::new();
        let has_upgrade = self
            .assets
            .iter()
            .any(|asset| asset.name == crate::upgrade::QUALIFICATION_FILE);
        if self.format != 1
            || !valid_tag(&self.tag)
            || !crate::buildx::is_revision(&self.soda_revision)
            || !crate::buildx::is_revision(&self.forgejo_revision)
            || self.notes.trim().is_empty()
            || self.assets.len()
                != REQUIRED_METADATA.len()
                    + 3 * (usize::from(self.has_appliance()) + usize::from(self.docker.is_some()))
                    + usize::from(has_upgrade)
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
        if self.has_appliance() {
            if self.iso != ISO_NAME {
                return Err(failure("installer asset required"));
            }
            self.asset(&self.iso)?;
            self.asset("media.json")?;
            let rootfs = self.asset(&self.rootfs)?;
            if rootfs.name != format!("{}-rootfs.img", rootfs.sha256) {
                return Err(failure("rootfs asset must be hash-named"));
            }
        } else if !self.iso.is_empty() || !self.rootfs.is_empty() || has_upgrade {
            return Err(failure("Docker-only release cannot carry appliance assets"));
        }
        if self.docker.is_some() != (self.distribution != Distribution::Appliance) {
            return Err(failure("selected Docker binding required"));
        }
        if let Some(docker) = &self.docker {
            docker.require_production()?;
            if docker.distribution != self.distribution
                || docker.release_tag != self.tag
                || docker.soda_revision != self.soda_revision
                || docker.forgejo_revision != self.forgejo_revision
            {
                return Err(failure("Docker binding differs from prepared release"));
            }
            for (name, identity) in [
                ("sodaos-docker-compose.yaml", &docker.compose),
                ("docker-packages.txt", &docker.packages),
                ("sodaos-source.tar", &docker.soda_source),
                ("fountain-source.tar", &docker.forgejo_source),
            ] {
                let asset = self.asset(name)?;
                if asset.sha256 != identity.sha256 || asset.bytes != identity.bytes {
                    return Err(failure("Docker release asset differs from binding"));
                }
            }
            self.asset("docker.json")?;
            if self.asset("candidate.json")?.sha256 != docker.candidate_sha256
                || self.asset("payload.json")?.sha256 != docker.payload_sha256
            {
                return Err(failure(
                    "Docker candidate/payload assets differ from binding",
                ));
            }
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
    tag: &str,
    distribution: Distribution,
    notes_path: &Path,
    out: &Path,
) -> Result<Manifest, Error> {
    let media_path = candidate.join("media/media.json");
    if !valid_tag(tag) {
        return Err(failure("exact release tag required"));
    }
    let candidate_path = candidate
        .to_str()
        .ok_or_else(|| failure("invalid candidate path"))?;
    let (payload, c, _) = crate::admission::load_admitted_candidate(candidate_path)?;
    admit_build(candidate, &payload, &c, distribution, tag)?;
    let media = if distribution != Distribution::Docker {
        Some(admit_media(&media_path, &payload, &c, tag)?)
    } else {
        None
    };
    let docker = if distribution != Distribution::Appliance {
        let value = crate::docker::admit_publication(candidate_path)?;
        if value.release_tag != tag || value.distribution != distribution {
            return Err(failure("Docker build selection differs from release"));
        }
        Some(value)
    } else {
        None
    };
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
    let mut assets = Vec::new();
    if let Some((media, rootfs)) = &media {
        let parent = media_path
            .parent()
            .ok_or_else(|| failure("media parent required"))?;
        assets.push(copy_asset(&parent.join(&media.iso.path), out, ISO_NAME)?);
        assets.push(copy_asset(&parent.join(&media.rootfs.path), out, rootfs)?);
        if assets[0].sha256 != media.iso.sha256
            || assets[0].bytes != media.iso.bytes as u64
            || assets[1].sha256 != media.rootfs.sha256
            || assets[1].bytes != media.rootfs.bytes as u64
        {
            return Err(failure("media bytes differ from recorded hashes"));
        }
        assets.push(copy_asset(&media_path, out, "media.json")?);
    }
    if docker.is_some() {
        for (source, name) in [
            (crate::docker::METADATA_PATH, "docker.json"),
            (crate::docker::PACKAGES_PATH, "docker-packages.txt"),
            (crate::docker::COMPOSE_PATH, "sodaos-docker-compose.yaml"),
        ] {
            assets.push(copy_asset(&candidate.join(source), out, name)?);
        }
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
    if !payload
        .upgrade_from
        .as_deref()
        .unwrap_or_default()
        .is_empty()
    {
        let path = candidate.join(crate::upgrade::QUALIFICATION_FILE);
        let receipt = crate::upgrade::load(&path)?;
        receipt.validate(
            &payload,
            &c,
            &media
                .as_ref()
                .ok_or_else(|| failure("upgrade requires appliance media"))?
                .0,
            &receipt.source_release,
            &receipt.source_support_sha256,
        )?;
        assets.push(copy_asset(&path, out, crate::upgrade::QUALIFICATION_FILE)?);
    } else if candidate.join(crate::upgrade::QUALIFICATION_FILE).exists() {
        return Err(failure(
            "upgrade qualification requires a declared prospective source",
        ));
    }
    let manifest = Manifest {
        format: 1,
        tag: tag.to_owned(),
        soda_revision: payload.revision,
        forgejo_revision: c.forgejo_revision,
        distribution,
        docker,
        iso: if media.is_some() {
            ISO_NAME.to_owned()
        } else {
            String::new()
        },
        rootfs: media.map(|(_, rootfs)| rootfs).unwrap_or_default(),
        notes,
        assets,
    };
    manifest.validate()?;
    write_json(&out.join("release.json"), &manifest)?;
    #[derive(Serialize)]
    struct Operator {
        candidate: String,
    }
    let bytes = serde_json::to_vec(&Operator {
        candidate: fs::canonicalize(candidate)
            .map_err(|_| failure("candidate path unavailable"))?
            .to_string_lossy()
            .into_owned(),
    })
    .map_err(|_| failure("operator request refused"))?;
    crate::buildx::write_new(
        out.join("operator.json")
            .to_str()
            .ok_or_else(Error::refused)?,
        &bytes,
        0o600,
    )?;
    write_checksums(&manifest, out)?;
    Ok(manifest)
}
pub(super) fn inventory(manifest: &Manifest, directory: &Path) -> Result<Vec<Asset>, Error> {
    manifest.validate()?;
    let mut assets = Vec::new();
    for expected in &manifest.assets {
        let actual = inspect(&directory.join(&expected.name), &expected.name)?;
        if &actual != expected {
            return Err(failure("prepared release asset changed"));
        }
        assets.push(actual);
    }
    if let Some(expected) = &manifest.docker {
        let actual: DockerMetadata = read_json(&directory.join("docker.json"))?;
        if &actual != expected {
            return Err(failure("prepared Docker metadata changed"));
        }
    }
    assets.push(inspect(&directory.join("release.json"), "release.json")?);
    let checksums = crate::document::read_file(
        directory
            .join("SHA256SUMS")
            .to_str()
            .ok_or_else(Error::refused)?,
        64 << 10,
    )?;
    if checksums != checksums_text(&assets).as_bytes() {
        return Err(failure(
            "prepared release checksums differ from actual assets",
        ));
    }
    assets.push(inspect(&directory.join("SHA256SUMS"), "SHA256SUMS")?);
    Ok(assets)
}

fn checksums_text(assets: &[Asset]) -> String {
    assets
        .iter()
        .map(|asset| format!("{}  {}\n", asset.sha256, asset.name))
        .collect()
}

pub(super) fn write_checksums(manifest: &Manifest, directory: &Path) -> Result<(), Error> {
    let mut assets = manifest.assets.clone();
    assets.push(inspect(&directory.join("release.json"), "release.json")?);
    fs::write(directory.join("SHA256SUMS"), checksums_text(&assets))
        .map_err(|_| failure("cannot write checksums"))
}

fn admit_build(
    directory: &Path,
    payload: &crate::payload::Payload,
    candidate: &crate::model::Candidate,
    distribution: Distribution,
    tag: &str,
) -> Result<(), Error> {
    let evidence = directory.join("../evidence");
    let bytes = crate::document::read_file(
        evidence
            .join("build.json")
            .to_str()
            .ok_or_else(Error::refused)?,
        1 << 20,
    )?;
    let completion: soda_build_tools::reader::build::BuildCompletion =
        read_json(&evidence.join("completed.json"))?;
    if completion.build_sha256 != crate::hash_bytes(&bytes)[7..] {
        return Err(failure(
            "successful controller completion for the unchanged producer result required",
        ));
    }
    let result: soda_build_tools::reader::build::BuildResult = crate::json_serde::strict(&bytes)?;
    let (selected, completed) = match distribution {
        Distribution::Appliance => ("appliance", "media"),
        Distribution::Docker => ("docker", "docker"),
        Distribution::All => ("all", "all"),
    };
    let actual = inspect(&directory.join("candidate.json"), "candidate.json")?;
    if result.purpose != "production"
        || result.requested_target != "release"
        || result.completed_target != completed
        || result.distribution != selected
        || result.revision != payload.revision
        || result.architecture != "x86_64"
        || result.host_manifest != candidate.host.manifest
        || result.payload_sha256 != candidate.payload_sha256
        || result.candidate_sha256 != actual.sha256
        || result.checks.is_empty()
        || (distribution != Distribution::Appliance && result.release_tag != tag)
        || (distribution == Distribution::Appliance
            && !result.release_tag.is_empty()
            && result.release_tag != tag)
    {
        return Err(failure(
            "completed production build for this exact distribution and candidate required",
        ));
    }
    Ok(())
}

fn admit_media(
    media_path: &Path,
    payload: &crate::payload::Payload,
    c: &crate::model::Candidate,
    tag: &str,
) -> Result<(MediaBinding, String), Error> {
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
    Ok((media, rootfs))
}

pub fn load(directory: &Path) -> Result<Manifest, Error> {
    let manifest: Manifest = read_json(&directory.join("release.json"))?;
    inventory(&manifest, directory)?;
    Ok(manifest)
}
pub(super) fn candidate(
    directory: &Path,
    manifest: &Manifest,
) -> Result<std::path::PathBuf, Error> {
    #[derive(Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Operator {
        candidate: String,
    }
    let operator: Operator = read_json(&directory.join("operator.json"))?;
    let path = std::path::PathBuf::from(operator.candidate);
    if let Some(expected) = &manifest.docker {
        let actual = crate::docker::admit_publication(path.to_str().ok_or_else(Error::refused)?)?;
        if &actual != expected {
            return Err(failure(
                "local candidate differs from prepared Docker release",
            ));
        }
    }
    Ok(path)
}
