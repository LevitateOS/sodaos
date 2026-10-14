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
    let marker: serde_json::Value = serde_json::from_reader(regular(&source.join("storage.json"))?)
        .map_err(|_| "Canlang storage identity is invalid")?;
    if marker["schema"] != "canlang.soda.storage.v1" {
        return Err("Canlang storage identity is invalid".into());
    }
    let snapshot = scan(source)?;
    new_directory(out, source)?;
    // A failed output is retained for diagnosis but has no successful seal;
    // callers must select another fresh output rather than overwrite it.
    copy(source, out, &snapshot)?;
    seal(out, &snapshot)
}

pub fn restore(source: &Path, out: &Path) -> Result<(), String> {
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

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;

    struct Fixture(PathBuf);
    impl Drop for Fixture {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.0);
        }
    }
    fn fixture() -> Fixture {
        let timestamp = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let root =
            std::env::temp_dir().join(format!("can-snapshot-{}-{timestamp}", std::process::id()));
        fs::create_dir(&root).unwrap();
        fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).unwrap();
        let source = root.join("data");
        fs::create_dir(&source).unwrap();
        fs::set_permissions(&source, fs::Permissions::from_mode(0o700)).unwrap();
        fs::write(
            source.join("storage.json"),
            r#"{"schema":"canlang.soda.storage.v1","app":"TaskBoard"}"#,
        )
        .unwrap();
        fs::write(source.join(".writer-lock"), "").unwrap();
        fs::create_dir(source.join("d1")).unwrap();
        fs::create_dir(source.join("objects")).unwrap();
        fs::write(source.join("d1/retained-bytes"), b"acknowledged state").unwrap();
        fs::write(
            source.join("objects/retained-file"),
            b"canonical file bytes",
        )
        .unwrap();
        Fixture(root)
    }

    #[test]
    fn snapshots_complete_stopped_tree_and_restores_only_into_fresh_identity() {
        let root = fixture();
        let data = root.0.join("data");
        let saved = root.0.join("saved");
        let restored = root.0.join("restored");
        backup(&data, &saved).unwrap();
        restore(&saved, &restored).unwrap();
        assert_eq!(
            fs::read(restored.join("d1/retained-bytes")).unwrap(),
            b"acknowledged state"
        );
        assert_eq!(
            fs::read(restored.join("objects/retained-file")).unwrap(),
            b"canonical file bytes"
        );
        fs::write(restored.join("later-write"), "protected").unwrap();
        assert!(restore(&saved, &restored).unwrap_err().contains("occupied"));
        assert_eq!(
            fs::read_to_string(restored.join("later-write")).unwrap(),
            "protected"
        );
    }

    #[test]
    fn live_writer_prevents_any_backup_output() {
        let root = fixture();
        let data = root.0.join("data");
        let saved = root.0.join("saved");
        let writer = OpenOptions::new()
            .read(true)
            .write(true)
            .open(data.join(".writer-lock"))
            .unwrap();
        assert_eq!(
            unsafe { libc::flock(writer.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) },
            0
        );
        assert!(backup(&data, &saved).unwrap_err().contains("stop"));
        assert!(!saved.exists());
    }

    #[test]
    fn damaged_or_foreign_snapshot_refuses_before_restore() {
        let root = fixture();
        let data = root.0.join("data");
        let saved = root.0.join("saved");
        let restored = root.0.join("restored");
        backup(&data, &saved).unwrap();
        fs::write(saved.join("objects/retained-file"), "changed").unwrap();
        assert!(restore(&saved, &restored).unwrap_err().contains("checksum"));
        assert!(!restored.exists());
        std::os::unix::fs::symlink(
            data.join("objects/retained-file"),
            data.join("foreign-link"),
        )
        .unwrap();
        assert!(backup(&data, &root.0.join("second"))
            .unwrap_err()
            .contains("links"));
        assert!(!root.0.join("second").exists());
    }
}
