//! Stopped, complete Canlang storage snapshots. The native writer lock excludes
//! both serving and bootstrap; restore always creates a separate directory.
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs::{self, File, OpenOptions};
use std::io::{Read, Write};
use std::os::fd::AsRawFd;
use std::os::unix::fs::{MetadataExt, OpenOptionsExt, PermissionsExt};
use std::path::{Component, Path};

#[derive(Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Entry {
    path: String,
    bytes: u64,
    sha256: String,
}

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Snapshot {
    schema: String,
    directories: Vec<String>,
    files: Vec<Entry>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    application: Option<SnapshotInfo>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SnapshotInfo {
    pub id: String,
    pub created_at: u64,
    pub application: super::Spec,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct StorageIdentity {
    schema: String,
    app: String,
    #[serde(rename = "storageFingerprint")]
    storage_fingerprint: String,
    resources: Vec<(String, String)>,
}

fn storage_identity(source: &Path) -> Result<StorageIdentity, String> {
    let identity: StorageIdentity = serde_json::from_reader(regular(&source.join("storage.json"))?)
        .map_err(|_| "Canlang storage identity is invalid")?;
    let mut bindings = std::collections::BTreeSet::new();
    if identity.schema != "canlang.soda.storage.v2"
        || identity.app.is_empty()
        || !super::wire::hex(&identity.storage_fingerprint, 64)
        || identity.resources.iter().any(|(binding, resource)| {
            binding.is_empty() || resource.is_empty() || !bindings.insert(binding)
        })
    {
        return Err("Canlang storage identity is invalid".into());
    }
    Ok(identity)
}

fn directory(path: &Path) -> Result<(), String> {
    // CoreOS exposes its trusted /var/home mount through /home. Resolve only
    // that platform alias; application-controlled path links remain forbidden.
    super::home_path(path)?;
    let meta = fs::symlink_metadata(path).map_err(|_| "snapshot directory is unavailable")?;
    if !meta.is_dir() || meta.file_type().is_symlink() || meta.mode() & 0o077 != 0 {
        return Err("snapshot requires a private canonical directory under /home".into());
    }
    Ok(())
}

fn new_directory(path: &Path, source: &Path) -> Result<(), String> {
    let parent = path.parent().ok_or("snapshot output requires a parent")?;
    directory(parent)?;
    if path.starts_with(source)
        || path.file_name().is_none()
        || path
            .components()
            .any(|c| !matches!(c, Component::RootDir | Component::Normal(_)))
    {
        return Err("snapshot output must have a fresh separate identity".into());
    }
    fs::create_dir(path).map_err(|_| "snapshot output is occupied or unavailable")?;
    fs::set_permissions(path, fs::Permissions::from_mode(0o700)).map_err(|e| e.to_string())
}

fn regular(path: &Path) -> Result<File, String> {
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW)
        .open(path)
        .map_err(|_| "snapshot file is unavailable")?;
    if !file.metadata().map_err(|e| e.to_string())?.is_file() {
        return Err("snapshot contains a special file".into());
    }
    Ok(file)
}

fn digest(file: &mut File) -> Result<(u64, String), String> {
    let mut hash = Sha256::new();
    let mut bytes = 0;
    let mut buffer = [0u8; 65536];
    loop {
        let count = file.read(&mut buffer).map_err(|e| e.to_string())?;
        if count == 0 {
            break;
        }
        hash.update(&buffer[..count]);
        bytes += count as u64;
    }
    Ok((bytes, crate::sha256::hex_lower(&hash.finalize())))
}

fn inventory(root: &Path, relative: &Path, result: &mut Snapshot) -> Result<(), String> {
    let mut children = fs::read_dir(root.join(relative))
        .map_err(|e| e.to_string())?
        .collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())?;
    children.sort_by_key(|entry| entry.file_name());
    for child in children {
        let path = relative.join(child.file_name());
        if relative.as_os_str().is_empty()
            && ["tmp", ".writer-lock", "snapshot.json"]
                .iter()
                .any(|name| path == Path::new(name))
        {
            continue;
        }
        let text = path
            .to_str()
            .ok_or("snapshot requires UTF-8 storage paths")?
            .to_string();
        let kind = child.file_type().map_err(|e| e.to_string())?;
        if kind.is_dir() {
            result.directories.push(text);
            inventory(root, &path, result)?;
        } else if kind.is_file() {
            let (bytes, sha256) = digest(&mut regular(&child.path())?)?;
            result.files.push(Entry {
                path: text,
                bytes,
                sha256,
            });
        } else {
            return Err("snapshot storage links and special files are unsupported".into());
        }
    }
    Ok(())
}

fn scan(root: &Path) -> Result<Snapshot, String> {
    let mut snapshot = Snapshot {
        schema: "canlang.soda.snapshot.v1".into(),
        directories: vec![],
        files: vec![],
        application: None,
    };
    inventory(root, Path::new(""), &mut snapshot)?;
    Ok(snapshot)
}

fn copy(source: &Path, out: &Path, snapshot: &Snapshot) -> Result<(), String> {
    for name in &snapshot.directories {
        let path = out.join(name);
        fs::create_dir(&path).map_err(|e| e.to_string())?;
        fs::set_permissions(path, fs::Permissions::from_mode(0o700)).map_err(|e| e.to_string())?;
    }
    for entry in &snapshot.files {
        let mut source = regular(&source.join(&entry.path))?;
        let mut file = OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(out.join(&entry.path))
            .map_err(|e| e.to_string())?;
        std::io::copy(&mut source, &mut file).map_err(|e| e.to_string())?;
        file.sync_all().map_err(|e| e.to_string())?;
    }
    let actual = scan(out)?;
    if actual.files != snapshot.files || actual.directories != snapshot.directories {
        return Err("snapshot byte verification failed".into());
    }
    Ok(())
}

fn seal(root: &Path, snapshot: &Snapshot) -> Result<(), String> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(root.join("snapshot.json"))
        .map_err(|e| e.to_string())?;
    file.write_all(&serde_json::to_vec(snapshot).map_err(|e| e.to_string())?)
        .map_err(|e| e.to_string())?;
    file.sync_all().map_err(|e| e.to_string())?;
    for dir in snapshot.directories.iter().rev() {
        File::open(root.join(dir))
            .and_then(|file| file.sync_all())
            .map_err(|e| e.to_string())?;
    }
    File::open(root)
        .and_then(|file| file.sync_all())
        .map_err(|e| e.to_string())?;
    File::open(root.parent().ok_or("snapshot parent is unavailable")?)
        .and_then(|file| file.sync_all())
        .map_err(|e| e.to_string())
}

pub fn backup(source: &Path, out: &Path) -> Result<(), String> {
    backup_with_identity(source, out, None, None)
}

pub fn backup_application(
    source: &Path,
    out: &Path,
    info: SnapshotInfo,
    expected_storage: &str,
) -> Result<(), String> {
    info.application.validate()?;
    if !super::wire::id(&info.id) || info.created_at == 0 {
        return Err("invalid application snapshot identity".into());
    }
    backup_with_identity(source, out, Some(info), Some(expected_storage))
}

fn backup_with_identity(
    source: &Path,
    out: &Path,
    info: Option<SnapshotInfo>,
    expected_storage: Option<&str>,
) -> Result<(), String> {
    directory(source)?;
    let lock = OpenOptions::new()
        .read(true)
        .write(true)
        .custom_flags(libc::O_NOFOLLOW)
        .open(source.join(".writer-lock"))
        .map_err(|_| "Canlang writer lock is unavailable")?;
    if !lock.metadata().map_err(|e| e.to_string())?.is_file()
        || unsafe { libc::flock(lock.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) } != 0
    {
        return Err("stop the Canlang writer before backing up".into());
    }
    let storage = storage_identity(source)?;
    if let Some(info) = &info {
        if storage.app != info.application.name
            || Some(storage.storage_fingerprint.as_str()) != expected_storage
        {
            return Err("snapshot storage belongs to another application".into());
        }
    }
    let mut snapshot = scan(source)?;
    snapshot.application = info;
    new_directory(out, source)?;
    // A failed output is retained for diagnosis but has no successful seal;
    // callers must select another fresh output rather than overwrite it.
    copy(source, out, &snapshot)?;
    seal(out, &snapshot)
}

pub fn application_info(source: &Path) -> Result<SnapshotInfo, String> {
    directory(source)?;
    let meta = fs::symlink_metadata(source).map_err(|_| "snapshot is unavailable")?;
    if meta.uid() != 0 {
        return Err("application snapshots must belong to the operator".into());
    }
    let file = regular(&source.join("snapshot.json"))?;
    if file.metadata().map_err(|e| e.to_string())?.uid() != 0 {
        return Err("application snapshot metadata must belong to the operator".into());
    }
    let snapshot: Snapshot =
        serde_json::from_reader(file).map_err(|_| "snapshot manifest is invalid")?;
    let info = snapshot
        .application
        .ok_or("snapshot has no admitted application identity")?;
    info.application.validate()?;
    if !super::wire::id(&info.id)
        || info.created_at == 0
        || source.file_name().and_then(|name| name.to_str()) != Some(&info.id)
    {
        return Err("invalid application snapshot identity".into());
    }
    Ok(info)
}

fn verified_snapshot(source: &Path) -> Result<Snapshot, String> {
    directory(source)?;
    let snapshot: Snapshot = serde_json::from_reader(regular(&source.join("snapshot.json"))?)
        .map_err(|_| "snapshot manifest is invalid")?;
    let actual = scan(source)?;
    if snapshot.schema != actual.schema
        || snapshot.files != actual.files
        || snapshot.directories != actual.directories
    {
        return Err("snapshot checksum or inventory verification failed".into());
    }
    storage_identity(source)?;
    Ok(snapshot)
}

pub(super) fn verify_source(source: &Path) -> Result<(), String> {
    verified_snapshot(source).map(|_| ())
}

pub fn restore(source: &Path, out: &Path) -> Result<(), String> {
    let snapshot = verified_snapshot(source)?;
    new_directory(out, source)?;
    copy(source, out, &snapshot)?;
    fs::create_dir(out.join("tmp")).map_err(|e| e.to_string())?;
    fs::set_permissions(out.join("tmp"), fs::Permissions::from_mode(0o700))
        .map_err(|e| e.to_string())?;
    let lock = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(out.join(".writer-lock"))
        .map_err(|e| e.to_string())?;
    lock.sync_all().map_err(|e| e.to_string())?;
    // A privileged restore prepares the dedicated fixed runtime account; an
    // unprivileged local proof keeps the invoking account's ownership.
    if unsafe { libc::geteuid() } == 0 {
        for path in snapshot
            .files
            .iter()
            .map(|f| out.join(&f.path))
            .chain(snapshot.directories.iter().map(|d| out.join(d)))
            .chain([out.to_path_buf(), out.join("tmp"), out.join(".writer-lock")])
        {
            let path = std::ffi::CString::new(path.as_os_str().as_encoded_bytes())
                .map_err(|_| "invalid snapshot path")?;
            if unsafe { libc::chown(path.as_ptr(), 10000, 10000) } != 0 {
                return Err("restored runtime ownership could not be set".into());
            }
        }
    }
    for dir in snapshot.directories.iter().rev() {
        File::open(out.join(dir))
            .and_then(|file| file.sync_all())
            .map_err(|e| e.to_string())?;
    }
    File::open(out)
        .and_then(|file| file.sync_all())
        .map_err(|e| e.to_string())
}

pub(super) fn verify_restored(source: &Path, out: &Path) -> Result<(), String> {
    let snapshot = verified_snapshot(source)?;
    directory(out)?;
    let restored = scan(out)?;
    if snapshot.files != restored.files || snapshot.directories != restored.directories {
        return Err("restored data does not match the selected snapshot".into());
    }
    storage_identity(out)?;
    Ok(())
}
