//! Exact-byte resolved input snapshots; no journey business-state schema here.
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::{
    fs,
    io::Write,
    os::unix::fs::{MetadataExt, PermissionsExt},
    path::{Path, PathBuf},
};

#[derive(Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct FileReference {
    pub path: PathBuf,
    pub sha256: String,
}

pub(crate) fn write_new(path: &Path, bytes: &[u8], mode: u32) -> Result<(), String> {
    let parent = path.parent().ok_or("parent required")?;
    let mut file = tempfile::NamedTempFile::new_in(parent).map_err(|e| e.to_string())?;
    file.as_file()
        .set_permissions(fs::Permissions::from_mode(mode))
        .map_err(|e| e.to_string())?;
    file.write_all(bytes)
        .and_then(|_| file.as_file().sync_all())
        .map_err(|e| e.to_string())?;
    file.persist_noclobber(path).map_err(|e| e.to_string())?;
    fs::File::open(parent)
        .and_then(|f| f.sync_all())
        .map_err(|e| e.to_string())
}

fn bytes(reference: &FileReference) -> Result<Vec<u8>, String> {
    let meta = fs::symlink_metadata(&reference.path).map_err(|e| e.to_string())?;
    if !meta.is_file() || meta.file_type().is_symlink() || meta.nlink() != 1 {
        return Err("unshared regular resolved input required".into());
    }
    let bytes = fs::read(&reference.path).map_err(|e| e.to_string())?;
    if format!("{:x}", Sha256::digest(&bytes)) != reference.sha256 {
        return Err("resolved input identity changed".into());
    }
    Ok(bytes)
}

/// Source protection is checked by the collector. Copy only the admitted small
/// document, never its referenced credentials, historical attempts or runtime.
pub(crate) fn snapshot(source: &FileReference, path: &Path) -> Result<FileReference, String> {
    let bytes = bytes(source)?;
    let value: serde_json::Value = serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    if !value.is_object() {
        return Err("resolved input object required".into());
    }
    write_new(path, &bytes, 0o444)?;
    Ok(FileReference {
        path: path.to_owned(),
        sha256: source.sha256.clone(),
    })
}

pub(crate) fn verify(snapshot: &FileReference) -> Result<(), String> {
    let meta = fs::symlink_metadata(&snapshot.path).map_err(|e| e.to_string())?;
    if meta.mode() & 0o7777 != 0o444 {
        return Err("read-only resolved input snapshot required".into());
    }
    bytes(snapshot).map(|_| ())
}
