//! Root custody and GitHub authenticity for one complete upgrade target.
use std::fs::{self, File, OpenOptions};
use std::io::Read;
use std::os::fd::AsRawFd;
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::time::Instant;

use serde::{Deserialize, Serialize};
use soda_build_tools::project_support::{ProjectSupport, DESTINATION};
use soda_release_deliver::model::{Candidate, MediaBinding};
use soda_release_deliver::payload::Payload;
use soda_release_deliver::upgrade;

const REPOSITORY: &str = "github.com/LevitateOS/sodaos";
const DOWNLOAD_PREFIX: &str = "https://github.com/LevitateOS/sodaos/releases/download/";

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct InstalledSource {
    pub(crate) release_id: String,
    pub(crate) helper_sha256: String,
    pub(crate) deployment: String,
    pub(crate) project_image: String,
}

pub(crate) struct Target {
    pub(crate) payload: Payload,
    pub(crate) candidate: Candidate,
    pub(crate) media: MediaBinding,
    /// Anchored through the held directory descriptor; valid while Target lives.
    pub(crate) rootfs: PathBuf,
    pub(crate) directory: PathBuf,
    pub(crate) source: InstalledSource,
    pub(crate) qualification_fixture: bool,
    _custody: Custody,
}

impl Target {
    /// Recheck immediately before mutation under the caller's admission lock.
    pub(crate) fn confirm_source(&self, deadline: Instant) -> Result<(), String> {
        if installed_source(deadline)? != self.source {
            return Err("installed release changed after target admission".into());
        }
        Ok(())
    }

    pub(crate) fn qualification_scope(&self) -> &'static str {
        if self.qualification_fixture {
            "development native qualification fixture; no supported upgrade claim"
        } else {
            "GitHub authenticated release with exact native upgrade qualification"
        }
    }
}

struct Custody {
    directory: File,
    original: PathBuf,
}

fn admitted_owner(info: &fs::Metadata, uid: u32, directory: bool) -> bool {
    info.uid() == uid
        && info.mode() & 0o022 == 0
        && if directory {
            info.is_dir()
        } else {
            info.is_file()
        }
}

impl Custody {
    fn open(path: &Path, uid: u32) -> Result<Self, String> {
        if !path.is_absolute() {
            return Err("absolute release directory required".into());
        }
        let directory = OpenOptions::new()
            .read(true)
            .custom_flags(libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC)
            .open(path)
            .map_err(|_| "release directory unavailable")?;
        let info = directory
            .metadata()
            .map_err(|_| "release directory unavailable")?;
        if !admitted_owner(&info, uid, true) {
            return Err(
                "release directory must be root-owned and not writable by other users".into(),
            );
        }
        Ok(Self {
            directory,
            original: path.to_path_buf(),
        })
    }

    fn path(&self) -> PathBuf {
        PathBuf::from(format!(
            "/proc/{}/fd/{}",
            std::process::id(),
            self.directory.as_raw_fd()
        ))
    }

    fn file(&self, name: &str, uid: u32, limit: u64) -> Result<File, String> {
        // All names come from fixed metadata paths or the admitted hash filename.
        if name.starts_with('/')
            || name
                .split('/')
                .any(|part| part.is_empty() || part == "." || part == "..")
        {
            return Err("invalid release asset path".into());
        }
        if let Some((parent, _)) = name.rsplit_once('/') {
            let info = fs::symlink_metadata(self.path().join(parent))
                .map_err(|_| "release asset directory unavailable")?;
            if !admitted_owner(&info, uid, true) || info.file_type().is_symlink() {
                return Err("release asset directory must retain root custody".into());
            }
        }
        let file = OpenOptions::new()
            .read(true)
            .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
            .open(self.path().join(name))
            .map_err(|_| "release asset unavailable")?;
        let info = file.metadata().map_err(|_| "release asset unavailable")?;
        if !admitted_owner(&info, uid, false) || info.len() == 0 || info.len() > limit {
            return Err(
                "release asset must be a bounded root-owned nonwritable regular file".into(),
            );
        }
        Ok(file)
    }

    fn read(&self, name: &str, limit: u64) -> Result<Vec<u8>, String> {
        let file = self.file(name, 0, limit)?;
        let mut bytes = Vec::new();
        file.take(limit + 1)
            .read_to_end(&mut bytes)
            .map_err(|_| "release asset unreadable")?;
        if bytes.len() as u64 > limit {
            return Err("release asset exceeds its bound".into());
        }
        Ok(bytes)
    }
}

fn fixed_root_file(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
        .open(path)
        .map_err(|_| "installed release unavailable")?;
    let info = file
        .metadata()
        .map_err(|_| "installed release unavailable")?;
    if !admitted_owner(&info, 0, false) || info.len() == 0 || info.len() > limit {
        return Err("installed release is not an admitted root-owned file".into());
    }
    let mut bytes = Vec::new();
    file.take(limit + 1)
        .read_to_end(&mut bytes)
        .map_err(|_| "installed release unreadable")?;
    if bytes.len() as u64 > limit {
        return Err("installed release exceeds its bound".into());
    }
    Ok(bytes)
}

fn source_support_binding(
    support: Option<&serde_json::Value>,
    project_image: &str,
    helper: &str,
) -> Result<(), String> {
    if let Some(support) = support {
        let support: ProjectSupport = serde_json::from_value(support.clone())
            .map_err(|_| "invalid installed Project support provenance")?;
        support.validate(project_image).map_err(str::to_string)?;
        if support.sha256 != helper {
            return Err("installed host helper differs from its release support binding".into());
        }
    }
    Ok(())
}

fn require_native_architecture(architecture: &str) -> Result<(), String> {
    if soda_build_tools::reader::oci_architecture(architecture).is_err()
        || !cfg!(target_os = "linux")
        || architecture != std::env::consts::ARCH
    {
        return Err("release maintenance requires matching-native Linux architecture".into());
    }
    Ok(())
}

pub(crate) fn installed_source(deadline: Instant) -> Result<InstalledSource, String> {
    require_native_architecture(std::env::consts::ARCH)?;
    if soda_host::terminal::agent_program_path() != DESTINATION {
        return Err("release maintenance refuses a terminal helper environment override".into());
    }
    let bytes = fixed_root_file(Path::new(crate::RELEASE_PATH), 4 << 20)?;
    let admitted = crate::release::load_release_payload(crate::RELEASE_PATH)
        .map_err(|_| "installed release provenance unavailable")?;
    require_native_architecture(&admitted.architecture)?;
    let payload: serde_json::Value =
        serde_json::from_slice(&bytes).map_err(|_| "installed release provenance unavailable")?;
    let release_id = payload
        .get("ID")
        .and_then(|value| value.as_str())
        .ok_or("installed release identity unavailable")?
        .to_owned();
    let helper_sha256 = soda_host::terminal::agent_program_hash(0)?;
    let project_image = admitted
        .image_config("project-os")
        .ok_or("installed Project image unavailable")?;
    source_support_binding(
        payload.get("ProjectSupport"),
        &project_image,
        &helper_sha256,
    )?;
    let status = crate::command::run("/usr/bin/ostree", &["admin", "status", "--json"], deadline)?;
    let status: serde_json::Value =
        serde_json::from_slice(&status).map_err(|_| "native booted deployment unavailable")?;
    let booted: Vec<_> = status
        .get("deployments")
        .and_then(|value| value.as_array())
        .ok_or("native booted deployment unavailable")?
        .iter()
        .filter(|deployment| {
            deployment.get("booted").and_then(|value| value.as_bool()) == Some(true)
        })
        .collect();
    if booted.len() != 1 {
        return Err("one exact native booted deployment required".into());
    }
    let deployment = booted[0]
        .get("checksum")
        .and_then(|value| value.as_str())
        .ok_or("native booted deployment unavailable")?
        .to_owned();
    if !crate::release_validation::is_hex_string(&deployment, 64) {
        return Err("invalid native booted deployment".into());
    }
    Ok(InstalledSource {
        release_id,
        helper_sha256,
        deployment,
        project_image,
    })
}

fn release_tag(url: &str, rootfs: &str) -> Result<String, String> {
    let rest = url
        .strip_prefix(DOWNLOAD_PREFIX)
        .ok_or("exact Soda GitHub release URL required")?;
    let (tag, asset) = rest
        .split_once('/')
        .ok_or("exact Soda GitHub release URL required")?;
    if asset != rootfs
        || tag.is_empty()
        || tag.len() > 128
        || tag.contains("..")
        || !tag.as_bytes()[0].is_ascii_alphanumeric()
        || !tag
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || b"._-".contains(&byte))
    {
        return Err("exact Soda GitHub release tag and rootfs asset required".into());
    }
    Ok(tag.to_owned())
}

fn verify_asset(tag: &str, path: &Path, deadline: Instant) -> Result<(), String> {
    let mut command = Command::new("/usr/bin/gh");
    command
        .args(["release", "verify-asset", tag])
        .arg(path)
        .args(["--repo", REPOSITORY])
        .env("GH_HOST", "github.com")
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::null());
    let child = command
        .spawn()
        .map_err(|_| "GitHub release asset verification unavailable; gh is required")?;
    crate::command::wait_output(child, deadline)
        .map(|_| ())
        .map_err(|_| "GitHub release asset authentication failed".into())
}

pub(crate) fn admit(
    directory: &Path,
    qualification_fixture: bool,
    deadline: Instant,
) -> Result<Target, String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err("release target admission requires root".into());
    }
    let source = installed_source(deadline)?;
    let custody = Custody::open(directory, 0)?;
    let anchored = custody.path();
    let media_name = if qualification_fixture && anchored.join("media/media.json").is_file() {
        "media/media.json"
    } else {
        "media.json"
    };
    for name in ["payload.json", "candidate.json", media_name] {
        custody.file(name, 0, 4 << 20)?;
    }
    let body = custody.read(media_name, 4 << 20)?;
    let media: MediaBinding =
        serde_json::from_slice(&body).map_err(|_| "invalid release media metadata")?;
    let rootfs_name = format!("{}-rootfs.img", media.rootfs.sha256);
    if media.rootfs.path != rootfs_name
        || !crate::release_validation::is_hex_string(&media.rootfs.sha256, 64)
    {
        return Err("exact hash-bound release rootfs required".into());
    }
    let rootfs_relative = if media_name.starts_with("media/") {
        format!("media/{rootfs_name}")
    } else {
        rootfs_name.clone()
    };
    custody.file(
        &rootfs_relative,
        0,
        soda_release_deliver::publish::assets::MAX_ASSET_BYTES - 1,
    )?;
    if !qualification_fixture {
        custody.file(upgrade::QUALIFICATION_FILE, 0, 1 << 20)?;
        let tag = release_tag(&media.rootfs_url, &rootfs_name)?;
        for name in [
            "payload.json",
            "candidate.json",
            "media.json",
            upgrade::QUALIFICATION_FILE,
            rootfs_name.as_str(),
        ] {
            verify_asset(&tag, &anchored.join(name), deadline)?;
        }
    }
    let (payload, candidate, media, rootfs) = if qualification_fixture {
        let target =
            upgrade::admit_candidate_directory(&anchored).map_err(|error| error.to_string())?;
        if target.payload.upgrade_from.as_deref().unwrap_or_default() != [source.release_id.clone()]
            || target.payload.id == source.release_id
            || target.media.deployment_commit == source.deployment
        {
            return Err("qualification fixture must declare this exact different installed source and native target".into());
        }
        (
            target.payload,
            target.candidate,
            target.media,
            target.rootfs,
        )
    } else {
        let target = upgrade::admit_directory(&anchored, &source.release_id, &source.helper_sha256)
            .map_err(|error| error.to_string())?;
        if target.media.deployment_commit == source.deployment {
            return Err("qualified upgrade requires a different native target deployment".into());
        }
        (
            target.payload,
            target.candidate,
            target.media,
            target.rootfs,
        )
    };
    require_native_architecture(&payload.architecture)?;
    Ok(Target {
        payload,
        candidate,
        media,
        rootfs,
        directory: custody.original.clone(),
        source,
        qualification_fixture,
        _custody: custody,
    })
}

/// Verify the complete embedded layout, then inspect only the exact future
/// Project image. This imports image content without changing retained roots.
pub(crate) fn verify_project_image(
    root: &str,
    target: &Target,
    deadline: Instant,
) -> Result<(), String> {
    if Instant::now() >= deadline {
        return Err("target image inspection timed out".into());
    }
    let images = Path::new(root).join("usr/share/soda/images");
    let images = images.to_str().ok_or("embedded image path unavailable")?;
    soda_release_deliver::content::verify_content(&target.payload, images)
        .map_err(|_| "embedded OCI content differs from the selected release")?;
    if Instant::now() >= deadline {
        return Err("target image inspection timed out".into());
    }
    let support = &target.payload.project_support;
    let source = format!("oci:{images}:{}", support.project_image);
    crate::command::run(
        "/usr/bin/podman",
        &["--remote=false", "pull", "--retry=0", &source],
        deadline,
    )
    .map_err(|_| "exact future Project image import unconfirmed")?;
    let observed = crate::command::run(
        "/usr/bin/podman",
        &[
            "--remote=false",
            "run",
            "--rm",
            "--pull=never",
            "--network=none",
            "--read-only",
            "--cap-drop=all",
            "--security-opt=no-new-privileges",
            "--entrypoint=/usr/bin/sha256sum",
            &support.project_image,
            DESTINATION,
        ],
        deadline,
    )
    .map_err(|_| "future Project image support inspection unavailable")?;
    if std::str::from_utf8(&observed)
        .ok()
        .map(str::split_whitespace)
        .map(Iterator::collect::<Vec<_>>)
        != Some(vec![support.sha256.as_str(), DESTINATION])
    {
        return Err("future Project image helper differs from the selected release".into());
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::{symlink, PermissionsExt};

    #[test]
    fn release_source_and_target_must_match_the_native_architecture() {
        require_native_architecture(std::env::consts::ARCH).unwrap();
        let foreign = if std::env::consts::ARCH == "x86_64" {
            "aarch64"
        } else {
            "x86_64"
        };
        for architecture in [foreign, "riscv64", ""] {
            assert_eq!(
                require_native_architecture(architecture).unwrap_err(),
                "release maintenance requires matching-native Linux architecture"
            );
        }
    }

    #[test]
    fn github_tag_is_bound_to_the_fixed_repository_and_exact_rootfs() {
        let asset = format!("{}-rootfs.img", "a".repeat(64));
        assert_eq!(
            release_tag(&format!("{DOWNLOAD_PREFIX}v0.2.0/{asset}"), &asset).unwrap(),
            "v0.2.0"
        );
        for url in [
            format!("https://example.invalid/releases/v0.2.0/{asset}"),
            format!("{DOWNLOAD_PREFIX}../{asset}"),
            format!("{DOWNLOAD_PREFIX}v0.2.0/other.img"),
            format!("{DOWNLOAD_PREFIX}v0.2.0/{asset}?token=private"),
        ] {
            assert!(release_tag(&url, &asset).is_err());
        }
    }

    #[test]
    fn held_release_directory_prevents_ancestor_rename_substitution() {
        let dir = crate::test_support::TestDir::make("release-custody");
        let selected = dir.0.join("selected");
        fs::create_dir(&selected).unwrap();
        fs::write(selected.join("payload.json"), b"selected-original").unwrap();
        let uid = unsafe { libc::geteuid() };
        let custody = Custody::open(&selected, uid).unwrap();
        fs::rename(&selected, dir.0.join("retained")).unwrap();
        fs::create_dir(&selected).unwrap();
        fs::write(selected.join("payload.json"), b"substituted").unwrap();
        let mut file = custody.file("payload.json", uid, 1024).unwrap();
        let mut bytes = Vec::new();
        file.read_to_end(&mut bytes).unwrap();
        assert_eq!(bytes, b"selected-original");
        symlink("payload.json", custody.path().join("symlink.json")).unwrap();
        assert!(custody.file("symlink.json", uid, 1024).is_err());
        fs::set_permissions(
            custody.path().join("payload.json"),
            fs::Permissions::from_mode(0o666),
        )
        .unwrap();
        assert!(custody.file("payload.json", uid, 1024).is_err());
    }

    #[test]
    fn installed_helper_must_match_its_source_release_binding() {
        let image = format!("sha256:{}", "a".repeat(64));
        let helper = "b".repeat(64);
        let support = serde_json::json!({
            "SHA256":helper, "Destination":DESTINATION,
            "PayloadPath":soda_build_tools::project_support::PAYLOAD_PATH,
            "ProjectImage":image,
        });
        source_support_binding(Some(&support), &image, &helper).unwrap();
        assert!(source_support_binding(Some(&support), &image, &"c".repeat(64)).is_err());
        assert!(source_support_binding(
            Some(&support),
            &format!("sha256:{}", "d".repeat(64)),
            &helper
        )
        .is_err());
    }
}
