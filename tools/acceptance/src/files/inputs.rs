use std::fs::File;
use std::io::Read;
use std::os::fd::AsRawFd;
use std::os::unix::fs::MetadataExt;
use std::path::Path;

use crate::error::Error;
use crate::sha256::Sha256;

use super::{same_file, FileAttr, OwnedDir, PRIVATE_FILE_LIMIT};

/// Read a restricted private input: absolute, regular, group/other-
/// inaccessible, at most 1 MiB. Mirrors `PrivateFile`.
pub fn private_file(path: &str) -> Result<Vec<u8>, Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute private input required"));
    }
    let meta = std::fs::symlink_metadata(path)?;
    if !meta.is_file() || meta.mode() & 0o077 != 0 || meta.size() > PRIVATE_FILE_LIMIT {
        let base = Path::new(path)
            .file_name()
            .map(|s| s.to_string_lossy().into_owned())
            .unwrap_or_default();
        return Err(Error::msg(format!(
            "restricted regular input required: {base}"
        )));
    }
    Ok(std::fs::read(path)?)
}

/// Stream a regular file's lowercase SHA-256 hex. Mirrors `HashFile`.
pub fn hash_file(path: &str) -> Result<String, Error> {
    let meta = std::fs::symlink_metadata(path)?;
    if !meta.is_file() {
        return Err(Error::msg("regular non-symlink file required"));
    }
    let mut file = File::open(path)?;
    hash_reader(&mut file)
}

/// Hash a root-relative file through an open directory, refusing changes
/// between stat and read. Mirrors `HashAt`.
pub fn hash_at(root: &OwnedDir, name: &str) -> Result<String, Error> {
    let before = root.lstat_at(name)?;
    if !before.is_regular {
        return Err(Error::msg("regular non-symlink file required"));
    }
    let mut file = root.open_file_at(name)?;
    let after = FileAttr::from_stat(&fstat_of(&file)?);
    if !after.is_regular || !same_file(before, after) {
        return Err(Error::msg("file changed before hashing"));
    }
    hash_reader(&mut file)
}

fn fstat_of(file: &File) -> Result<libc::stat, Error> {
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::fstat(file.as_raw_fd(), &mut st) };
    if rc != 0 {
        return Err(Error::from(std::io::Error::last_os_error()));
    }
    Ok(st)
}

fn hash_reader(reader: &mut dyn Read) -> Result<String, Error> {
    let mut hasher = Sha256::new();
    let mut buf = [0u8; 32768];
    loop {
        let n = reader.read(&mut buf)?;
        if n == 0 {
            break;
        }
        hasher.update(&buf[..n]);
    }
    Ok(crate::sha256::hex_lower(&hasher.finalize()))
}
