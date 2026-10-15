//! Native FCOS staging. The caller owns durable admission and operation records.
use std::fs::{self, File, OpenOptions};
use std::io::{Read, Seek};
use std::os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

use crate::command::{run, wait_output};
use crate::release_target::Target;

const OSTREE: &str = "/usr/bin/ostree";
const REPOSITORY: &str = "/sysroot/ostree/repo";
const DEPLOYMENTS: &str = "/sysroot/ostree/deploy/fedora-coreos/deploy";
const STAGE: &str = "/var/lib/soda/project-maintenance/stage";

#[derive(Debug, Clone, Deserialize, Serialize, PartialEq, Eq)]
pub(crate) struct Deployment {
    pub checksum: String,
    pub serial: u32,
    #[serde(default)]
    pub staged: bool,
    #[serde(default, rename = "finalization-locked")]
    pub finalization_locked: bool,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct State {
    pub booted: Deployment,
    pub staged: Option<Deployment>,
}

#[derive(Deserialize)]
struct NativeDeployment {
    #[serde(flatten)]
    deployment: Deployment,
    index: usize,
    #[serde(default)]
    booted: bool,
    #[serde(default)]
    pending: bool,
    stateroot: String,
}

fn digest(value: &str) -> bool {
    value.len() == 64
        && value
            .bytes()
            .all(|c| c.is_ascii_digit() || (b'a'..=b'f').contains(&c))
}

fn decode(bytes: &[u8]) -> Result<Vec<NativeDeployment>, String> {
    #[derive(Deserialize)]
    struct Document {
        deployments: Vec<NativeDeployment>,
    }
    let rows = serde_json::from_slice::<Document>(bytes)
        .map_err(|_| "native deployment observation unavailable")?
        .deployments;
    if rows.is_empty()
        || rows.len() > 16
        || rows.iter().enumerate().any(|(i, row)| {
            row.index != i
                || row.stateroot != "fedora-coreos"
                || !digest(&row.deployment.checksum)
                || row.pending && !row.deployment.staged
                || row.booted && row.deployment.staged
                || row.deployment.finalization_locked && !row.deployment.staged
        })
        || rows.iter().filter(|r| r.booted).count() != 1
        || rows.iter().filter(|r| r.deployment.staged).count() > 1
    {
        return Err("native deployment state is unsupported or uncertain".into());
    }
    Ok(rows)
}

fn deployments(deadline: Instant) -> Result<Vec<NativeDeployment>, String> {
    decode(&run(OSTREE, &["admin", "status", "--json"], deadline)?)
}

pub(crate) fn inspect(deadline: Instant) -> Result<State, String> {
    let rows = deployments(deadline)?;
    Ok(State {
        booted: rows
            .iter()
            .find(|r| r.booted)
            .ok_or("booted deployment unavailable")?
            .deployment
            .clone(),
        staged: rows
            .iter()
            .find(|r| r.deployment.staged)
            .map(|r| r.deployment.clone()),
    })
}

pub(crate) fn deployment_root(deployment: &Deployment) -> Result<PathBuf, String> {
    if !digest(&deployment.checksum) {
        return Err("invalid deployment identity".into());
    }
    Ok(Path::new(DEPLOYMENTS).join(format!("{}.{}", deployment.checksum, deployment.serial)))
}

fn origin(deployment: &Deployment) -> Result<String, String> {
    let path = deployment_root(deployment)?.with_extension(format!("{}.origin", deployment.serial));
    trusted(&path, 65536)?;
    path.into_os_string()
        .into_string()
        .map_err(|_| "native origin path unavailable".into())
}

fn pending(state: &State, commit: &str) -> Result<bool, String> {
    if !digest(commit) {
        return Err("invalid target deployment commit".into());
    }
    match &state.staged {
        None => Ok(false),
        Some(staged) if staged.checksum == commit && staged.finalization_locked => Ok(true),
        Some(_) => Err("another pending deployment blocks Project maintenance".into()),
    }
}

fn stage_commit(commit: &str, source: &Deployment, deadline: Instant) -> Result<(), String> {
    let before = inspect(deadline)?;
    if pending(&before, commit)? {
        return Ok(());
    }
    let origin = origin(source)?;
    run(
        OSTREE,
        &[
            "admin",
            "deploy",
            "--os=fedora-coreos",
            "--stage",
            "--lock-finalization",
            "--retain",
            "--origin-file",
            &origin,
            commit,
        ],
        deadline,
    )?;
    let after = inspect(deadline)?;
    if after.booted != before.booted || !pending(&after, commit)? {
        return Err("native target was not staged with finalization locked".into());
    }
    Ok(())
}

pub(crate) fn stage_retained(commit: &str, deadline: Instant) -> Result<(), String> {
    if !digest(commit) {
        return Err("invalid retained deployment commit".into());
    }
    let rows = deployments(deadline)?;
    let source = rows
        .iter()
        .find(|r| r.deployment.checksum == commit && !r.deployment.staged)
        .ok_or("rollback deployment is not retained")?
        .deployment
        .clone();
    stage_commit(commit, &source, deadline)
}

// Only the coordinator's exact locked target may be discarded. Never clear an
// unrelated pending deployment, even when it prevents this operation proceeding.
pub(crate) fn discard_locked(commit: &str, deadline: Instant) -> Result<(), String> {
    let before = inspect(deadline)?;
    if !pending(&before, commit)? {
        return Ok(());
    }
    let rows = deployments(deadline)?;
    let index = rows
        .iter()
        .find(|r| {
            r.deployment.staged
                && r.deployment.checksum == commit
                && r.deployment.finalization_locked
        })
        .ok_or("locked target changed before discard")?
        .index
        .to_string();
    run(OSTREE, &["admin", "undeploy", &index], deadline)?;
    let after = inspect(deadline)?;
    if after.booted != before.booted || after.staged.is_some() {
        return Err("discarded target could not be confirmed".into());
    }
    Ok(())
}

// This intentionally requests native activation/reboot. The coordinator must
// persist its mutation boundary and keep admission closed before invoking it.
pub(crate) fn activate(commit: &str, deadline: Instant) -> Result<(), String> {
    let before = inspect(deadline)?;
    if !pending(&before, commit)? {
        return Err("exact target is not staged and locked".into());
    }
    let staged = before.staged.as_ref().ok_or("locked target unavailable")?;
    let token = finalization_token(
        &run("/usr/bin/rpm-ostree", &["status", "--json"], deadline)?,
        staged,
    )?;
    if inspect(deadline)? != before {
        return Err("locked target changed before native activation".into());
    }
    run(
        "/usr/bin/rpm-ostree",
        &["finalize-deployment", &token],
        deadline,
    )?;
    Ok(())
}

// rpm-ostree authenticates finalization against the staged base identity:
// an imported container's manifest digest, otherwise the base OSTree commit.
// This also applies to container commits deployed manually from a rootfs repo;
// their origin/refspec does not select which token rpm-ostree expects.
fn finalization_token(bytes: &[u8], staged: &Deployment) -> Result<String, String> {
    #[derive(Deserialize)]
    struct BaseMetadata {
        #[serde(rename = "ostree.manifest-digest")]
        image_digest: Option<String>,
    }
    #[derive(Deserialize)]
    struct Row {
        checksum: String,
        serial: u32,
        osname: String,
        #[serde(default)]
        staged: bool,
        #[serde(default)]
        booted: bool,
        #[serde(rename = "base-checksum")]
        base_checksum: Option<String>,
        #[serde(rename = "container-image-reference-digest")]
        image_digest: Option<String>,
        #[serde(rename = "base-commit-meta")]
        base_metadata: BaseMetadata,
    }
    #[derive(Deserialize)]
    struct Status {
        deployments: Vec<Row>,
    }
    let status: Status =
        serde_json::from_slice(bytes).map_err(|_| "native finalization identity unavailable")?;
    if !staged.staged || !staged.finalization_locked || status.deployments.len() > 16 {
        return Err("exact locked target required for native finalization".into());
    }
    let mut pending = status.deployments.iter().filter(|row| row.staged);
    let row = pending
        .next()
        .ok_or("native finalization target unavailable")?;
    if pending.next().is_some()
        || row.booted
        || row.osname != "fedora-coreos"
        || row.checksum != staged.checksum
        || row.serial != staged.serial
        || !digest(&row.checksum)
        || row.base_checksum.as_ref().is_some_and(|base| !digest(base))
    {
        return Err("native finalization target differs from locked deployment".into());
    }
    let metadata_digest = row.base_metadata.image_digest.as_ref();
    match (row.image_digest.as_ref(), metadata_digest) {
        (Some(image), Some(metadata))
            if image == metadata && image.strip_prefix("sha256:").is_some_and(digest) =>
        {
            Ok(image.clone())
        }
        (None, None) => Ok(row.base_checksum.as_ref().unwrap_or(&row.checksum).clone()),
        _ => Err("native container finalization identity is uncertain".into()),
    }
}

fn trusted(path: &Path, limit: u64) -> Result<File, String> {
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
        .open(path)
        .map_err(|_| "release content unavailable")?;
    let metadata = file.metadata().map_err(|_| "release content unavailable")?;
    if !metadata.is_file()
        || metadata.uid() != 0
        || metadata.mode() & 0o022 != 0
        || metadata.len() > limit
    {
        return Err("release content is not a root-owned admitted file".into());
    }
    Ok(file)
}

fn hash(file: &mut File, deadline: Instant) -> Result<String, String> {
    let mut sha = Sha256::new();
    let mut bytes = [0; 65536];
    loop {
        if Instant::now() >= deadline {
            return Err("release content inspection timed out".into());
        }
        let count = file
            .read(&mut bytes)
            .map_err(|_| "release content unreadable")?;
        if count == 0 {
            break;
        }
        sha.update(&bytes[..count]);
    }
    Ok(format!("{:x}", sha.finalize()))
}

pub(crate) struct StagedContent {
    pub helper_path: String,
    pub root: String,
    commit: String,
    scratch: PathBuf,
    image: PathBuf,
    mountpoint: PathBuf,
    mounted: bool,
}

impl StagedContent {
    pub(crate) fn open(target: &Target, deadline: Instant) -> Result<Self, String> {
        let commit = &target.media.deployment_commit;
        if !digest(commit) || !digest(&target.media.rootfs.sha256) {
            return Err("invalid admitted target content identity".into());
        }
        let size = u64::try_from(target.media.rootfs.bytes).map_err(|_| "invalid rootfs size")?;
        let mut input = trusted(&target.rootfs, size)?;
        if input.metadata().map_err(|_| "rootfs unavailable")?.len() != size
            || hash(&mut input, deadline)? != target.media.rootfs.sha256
        {
            return Err("rootfs differs from admitted release".into());
        }
        input.rewind().map_err(|_| "rootfs rewind failed")?;
        ensure_stage_directory()?;
        let unique = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_err(|_| "staging clock unavailable")?
            .as_nanos();
        let scratch = Path::new(STAGE).join(format!("{}-{}-{unique}", commit, std::process::id()));
        fs::DirBuilder::new()
            .mode(0o700)
            .create(&scratch)
            .map_err(|_| "fresh root-owned staging directory required")?;
        let mountpoint = scratch.join("mount");
        let image = scratch.join("root.erofs");
        let mut content = Self {
            helper_path: String::new(),
            root: String::new(),
            commit: commit.clone(),
            scratch,
            image,
            mountpoint,
            mounted: false,
        };
        fs::DirBuilder::new()
            .mode(0o700)
            .create(&content.mountpoint)
            .map_err(|_| "staging mountpoint unavailable")?;
        // Mature cpio selects this one fixed member into a fresh directory;
        // no source-controlled path or shell is executed during extraction.
        let child = Command::new("/usr/bin/cpio")
            .args([
                "--extract",
                "--quiet",
                "--no-preserve-owner",
                "--no-absolute-filenames",
                "--directory",
                text(&content.scratch)?,
                "root.erofs",
            ])
            .stdin(Stdio::from(input))
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .map_err(|_| "native rootfs extraction failed")?;
        wait_output(child, deadline).map_err(|_| "native rootfs extraction failed")?;
        trusted(&content.image, size)?;
        content.mounted = true;
        run(
            "/usr/bin/mount",
            &[
                "-t",
                "erofs",
                "-o",
                "ro,loop,nodev,nosuid,noexec",
                text(&content.image)?,
                text(&content.mountpoint)?,
            ],
            deadline,
        )?;
        let root = content
            .mountpoint
            .join("ostree/deploy/fedora-coreos/deploy")
            .join(format!("{commit}.0"));
        content.root = text(&root)?.to_owned();
        let payload_path = root.join("usr/share/soda/release.json");
        let mut payload = trusted(&payload_path, 4 << 20)?;
        if hash(&mut payload, deadline)? != target.candidate.payload_sha256 {
            return Err("mounted target payload differs from admitted release".into());
        }
        payload
            .rewind()
            .map_err(|_| "target payload rewind failed")?;
        let mut bytes = Vec::new();
        payload
            .read_to_end(&mut bytes)
            .map_err(|_| "mounted release payload invalid")?;
        let payload: soda_release_deliver::payload::Payload =
            soda_build_tools::payload::PayloadV4::parse(&bytes)
                .map_err(|_| "mounted release payload invalid")?;
        if payload != target.payload {
            return Err("mounted release binding differs".into());
        }
        let support = &target.payload.project_support;
        for relative in [&support.destination, &support.payload_path] {
            let path = root.join(relative.trim_start_matches('/'));
            let mut helper = trusted(&path, 64 << 20)?;
            if helper
                .metadata()
                .map_err(|_| "target helper unavailable")?
                .mode()
                & 0o111
                == 0
                || hash(&mut helper, deadline)? != support.sha256
            {
                return Err("mounted host and maintenance helper bytes differ".into());
            }
        }
        content.helper_path =
            text(&root.join(support.payload_path.trim_start_matches('/')))?.to_owned();
        Ok(content)
    }

    pub(crate) fn stage(&self, commit: &str, deadline: Instant) -> Result<(), String> {
        if commit != self.commit {
            return Err("staging commit differs from admitted mounted target".into());
        }
        let before = inspect(deadline)?;
        let already_staged = pending(&before, commit)?;
        let repository = self.mountpoint.join("ostree/repo");
        let repository = repository
            .to_str()
            .ok_or("target repository path unavailable")?;
        // rpm-ostree resolves container base metadata through digest-named blob
        // refs; pulling only the deployment commit leaves those refs missing.
        // Never import the source repository's mutable image/refspec pointers.
        let refs = run(OSTREE, &[&format!("--repo={repository}"), "refs"], deadline)?;
        let mut args = vec![
            format!("--repo={REPOSITORY}"),
            "pull-local".into(),
            "--untrusted".into(),
            repository.into(),
            commit.into(),
        ];
        args.extend(container_blob_refs(&refs)?);
        let borrowed: Vec<_> = args.iter().map(String::as_str).collect();
        run(OSTREE, &borrowed, deadline)?;
        if already_staged {
            if inspect(deadline)? != before {
                return Err("locked target changed during native metadata import".into());
            }
            return Ok(());
        }
        stage_commit(commit, &before.booted, deadline)
    }

    pub(crate) fn close(mut self, deadline: Instant) -> Result<(), String> {
        self.cleanup(deadline)
    }

    fn cleanup(&mut self, deadline: Instant) -> Result<(), String> {
        if self.mounted {
            run("/usr/bin/umount", &[text(&self.mountpoint)?], deadline)?;
            self.mounted = false;
        }
        // Remove only this invocation's exact known files and empty directories.
        // Any unexpected entry or still-mounted path is retained for inspection.
        for result in [
            fs::remove_file(&self.image),
            fs::remove_dir(&self.mountpoint),
            fs::remove_dir(&self.scratch),
        ] {
            if let Err(error) = result {
                if error.kind() != std::io::ErrorKind::NotFound {
                    return Err("staging cleanup requires inspection".into());
                }
            }
        }
        Ok(())
    }
}

fn container_blob_refs(bytes: &[u8]) -> Result<Vec<String>, String> {
    const PREFIX: &str = "ostree/container/blob/";
    let text = std::str::from_utf8(bytes).map_err(|_| "native repository refs unavailable")?;
    if text.lines().count() > 256 {
        return Err("native repository ref inventory exceeds its bound".into());
    }
    text.lines()
        .filter(|reference| reference.starts_with(PREFIX))
        .map(|reference| {
            if !reference
                .strip_prefix("ostree/container/blob/sha256_3A_")
                .is_some_and(digest)
            {
                return Err("invalid native container blob ref".into());
            }
            Ok(reference.to_owned())
        })
        .collect()
}

fn text(path: &Path) -> Result<&str, String> {
    path.to_str()
        .ok_or_else(|| "native staging path unavailable".into())
}

fn ensure_stage_directory() -> Result<(), String> {
    match fs::DirBuilder::new().mode(0o700).create(STAGE) {
        Ok(()) => {}
        Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {}
        Err(_) => return Err("native staging directory unavailable".into()),
    }
    let directory = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC)
        .open(STAGE)
        .map_err(|_| "unsafe native staging directory")?;
    let metadata = directory
        .metadata()
        .map_err(|_| "native staging directory unavailable")?;
    if metadata.uid() != 0 || metadata.mode() & 0o022 != 0 {
        return Err("native staging directory must retain root custody".into());
    }
    Ok(())
}

impl Drop for StagedContent {
    fn drop(&mut self) {
        let _ = self.cleanup(Instant::now() + Duration::from_secs(10));
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::{json, Value};

    #[test]
    fn native_import_keeps_digest_blob_refs_without_retargeting_image_pointers() {
        let blob = format!("ostree/container/blob/sha256_3A_{}", "a".repeat(64));
        let refs = format!("ostree/1/1/0\n{blob}\nostree/container/image/current\n");
        assert_eq!(container_blob_refs(refs.as_bytes()).unwrap(), vec![blob]);
        for invalid in [
            b"ostree/container/blob/sha256_3A_../current".as_slice(),
            b"ostree/container/blob/--mirror".as_slice(),
            b"ostree/container/blob/sha256_3A_ABC".as_slice(),
            b"\xff".as_slice(),
        ] {
            assert!(container_blob_refs(invalid).is_err());
        }
        assert!(container_blob_refs("other\n".repeat(257).as_bytes()).is_err());
    }

    fn observed() -> Value {
        json!({"deployments":[
            {"checksum":"a".repeat(64),"serial":1,"index":0,"booted":false,
             "pending":true,"staged":true,"finalization-locked":true,"stateroot":"fedora-coreos"},
            {"checksum":"b".repeat(64),"serial":0,"index":1,"booted":true,
             "stateroot":"fedora-coreos"}
        ]})
    }

    fn state(value: &Value) -> Result<State, String> {
        let rows = decode(&serde_json::to_vec(value).unwrap())?;
        Ok(State {
            booted: rows.iter().find(|r| r.booted).unwrap().deployment.clone(),
            staged: rows
                .iter()
                .find(|r| r.deployment.staged)
                .map(|r| r.deployment.clone()),
        })
    }

    #[test]
    fn native_locked_pending_is_exact_and_unknown_state_is_not_discardable() {
        let mut value = observed();
        let exact = state(&value).unwrap();
        assert!(pending(&exact, &"a".repeat(64)).unwrap());
        assert!(pending(&exact, &"c".repeat(64)).is_err());
        value["deployments"][0]
            .as_object_mut()
            .unwrap()
            .remove("finalization-locked");
        assert!(pending(&state(&value).unwrap(), &"a".repeat(64)).is_err());
        value = observed();
        value["deployments"][0]["staged"] = json!(false);
        assert!(state(&value).is_err()); // A pending ordinary native deployment is not ours.
        value = observed();
        value["deployments"][0]["stateroot"] = json!("other-root");
        assert!(state(&value).is_err());
        value = observed();
        value["deployments"][0]["index"] = json!(1);
        assert!(state(&value).is_err());
    }

    fn finalization_status() -> Value {
        json!({"deployments":[{
            "checksum":"a".repeat(64),"serial":1,"osname":"fedora-coreos",
            "staged":true,"booted":false,
            "base-checksum":null,
            "container-image-reference-digest":format!("sha256:{}", "c".repeat(64)),
            "base-commit-meta":{"ostree.manifest-digest":format!("sha256:{}", "c".repeat(64))}
        }]})
    }

    fn token(value: &Value, staged: &Deployment) -> Result<String, String> {
        finalization_token(&serde_json::to_vec(value).unwrap(), staged)
    }

    #[test]
    fn container_finalization_uses_native_manifest_for_oci_and_manual_deployments() {
        let staged = state(&observed()).unwrap().staged.unwrap();
        let mut value = finalization_status();
        // Native OCI rebase and manual rootfs deploy both retain container
        // commit metadata. The origin is irrelevant to native token selection.
        for origin in ["ostree-unverified-image:oci-archive:/source.oci", ""] {
            value["deployments"][0]["container-image-reference"] = json!(origin);
            assert_eq!(
                token(&value, &staged).unwrap(),
                format!("sha256:{}", "c".repeat(64))
            );
        }
    }

    #[test]
    fn noncontainer_finalization_uses_observed_base_commit() {
        let staged = state(&observed()).unwrap().staged.unwrap();
        let mut value = finalization_status();
        value["deployments"][0]["container-image-reference-digest"] = Value::Null;
        value["deployments"][0]["base-commit-meta"] = json!({});
        assert_eq!(token(&value, &staged).unwrap(), "a".repeat(64));
        value["deployments"][0]["base-checksum"] = json!("b".repeat(64));
        assert_eq!(token(&value, &staged).unwrap(), "b".repeat(64));
    }

    #[test]
    fn finalization_refuses_changed_or_uncertain_native_identity() {
        let staged = state(&observed()).unwrap().staged.unwrap();
        for (field, changed) in [
            ("checksum", json!("b".repeat(64))),
            ("serial", json!(2)),
            ("staged", json!(false)),
            ("booted", json!(true)),
            ("osname", json!("other")),
            ("base-checksum", json!("invalid")),
            ("base-commit-meta", Value::Null),
            ("container-image-reference-digest", Value::Null),
            ("container-image-reference-digest", json!("sha256:invalid")),
            (
                "base-commit-meta",
                json!({"ostree.manifest-digest":format!("sha256:{}", "d".repeat(64))}),
            ),
        ] {
            let mut value = finalization_status();
            value["deployments"][0][field] = changed;
            assert!(token(&value, &staged).is_err(), "accepted changed {field}");
        }
        let mut value = finalization_status();
        let duplicate = value["deployments"][0].clone();
        value["deployments"].as_array_mut().unwrap().push(duplicate);
        assert!(token(&value, &staged).is_err());
        assert!(token(
            &finalization_status(),
            &Deployment {
                finalization_locked: false,
                ..staged
            }
        )
        .is_err());
    }

    #[test]
    fn retained_paths_require_native_commit_identity() {
        let source = state(&observed()).unwrap().booted;
        assert_eq!(
            deployment_root(&source).unwrap(),
            Path::new(DEPLOYMENTS).join(format!("{}.0", "b".repeat(64)))
        );
        assert!(deployment_root(&Deployment {
            checksum: "../wrong".into(),
            ..source
        })
        .is_err());
    }
}
