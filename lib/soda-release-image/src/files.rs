//! Owned file writes, hashing, public inventories, media files
//! (`complete.go` + `assemble.go` shared leaves).

use std::fs;
use std::os::unix::fs::MetadataExt;
use std::os::unix::fs::PermissionsExt;

use serde::Serialize;

use crate::error::Error;
use crate::sys;

pub fn owned_write(path: &str, data: &[u8], mode: u32) -> Result<(), Error> {
    let dir = sys::dir_name(path);
    fs::create_dir_all(&dir)?;
    fs::write(path, data)?;
    fs::set_permissions(path, fs::Permissions::from_mode(mode))?;
    Ok(())
}

pub fn hash_bytes(data: &[u8]) -> String {
    sys::hex_sha256(data)
}

/// publicFiles verifies directly emitted assets without another layout copy.
/// Public modes must not depend on the builder's umask.
pub fn public_files(source: &str) -> Result<Vec<(String, String)>, Error> {
    let meta = fs::symlink_metadata(source)
        .map_err(|_| Error::msg("real generated public directory required"))?;
    if !meta.file_type().is_dir() {
        return Err(Error::msg("real generated public directory required"));
    }
    let mut files: Vec<(String, String)> = Vec::new();
    sys::walk(source, |path, is_dir, _| {
        let rel = sys::rel_path(source, path)?;
        let info = fs::symlink_metadata(path)?;
        if is_dir {
            if info.mode() != 0o40755 {
                return Err(Error::msg("public presentation directory must be 0755"));
            }
            return Ok(());
        }
        if !info.file_type().is_file() {
            return Err(Error::msg("public stage symlink/special file refused"));
        }
        if info.mode() != 0o100644 {
            return Err(Error::msg("public presentation file must be 0644"));
        }
        let hash = sys::hash_file(path)?;
        files.push((sys::to_slash(&rel), hash));
        Ok(())
    })?;
    Ok(files)
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct MediaFile {
    #[serde(rename = "Path")]
    pub path: String,
    #[serde(rename = "SHA256")]
    pub sha256: String,
    #[serde(rename = "Bytes")]
    pub bytes: i64,
}

pub fn media_file(path: &str) -> Result<MediaFile, Error> {
    let meta = fs::metadata(path)?;
    let hash = sys::hash_file(path)?;
    Ok(MediaFile {
        path: sys::base_name(path),
        sha256: hash,
        bytes: meta.len() as i64,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_public_files_enforce_modes() {
        // Oracle: Go publicFiles mode errors.
        let dir = std::env::temp_dir().join(format!("sri-pf-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(dir.join("sub")).unwrap();
        fs::write(dir.join("sub/a.txt"), b"a").unwrap();
        fs::set_permissions(dir.join("sub/a.txt"), fs::Permissions::from_mode(0o644)).unwrap();
        fs::set_permissions(dir.join("sub"), fs::Permissions::from_mode(0o755)).unwrap();
        fs::set_permissions(&dir, fs::Permissions::from_mode(0o755)).unwrap();
        let files = public_files(dir.to_str().unwrap()).unwrap();
        assert_eq!(files.len(), 1);
        assert_eq!(files[0].0, "sub/a.txt");
        fs::set_permissions(dir.join("sub/a.txt"), fs::Permissions::from_mode(0o600)).unwrap();
        assert_eq!(
            public_files(dir.to_str().unwrap()).unwrap_err().0,
            "public presentation file must be 0644"
        );
        let _ = fs::remove_dir_all(&dir);
    }
}
