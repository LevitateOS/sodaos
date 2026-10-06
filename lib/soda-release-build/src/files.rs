//! Native artifact support (`files.go`): validators, hashing, and
//! private-output admission. No product policy or release qualification.

use crate::json_go::{marshal_indent, Emit, Strict};
use crate::{io_error, sha256_hex_stream, Error};
use std::fs::{File as FsFile, OpenOptions};
use std::io::Write;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};

#[cfg(test)]
mod tests;

/// One inventoried byte, link, or directory for producer records and
/// context inventories.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct File {
    pub sha256: String,
    pub mode: u32,
    pub link: String,
    pub directory: bool,
}

impl File {
    pub fn emit(&self) -> Emit {
        let mut fields = Vec::new();
        if !self.sha256.is_empty() {
            fields.push(("sha256".to_string(), Emit::Str(self.sha256.clone())));
        }
        fields.push(("mode".to_string(), Emit::UInt(self.mode)));
        if !self.link.is_empty() {
            fields.push(("link".to_string(), Emit::Str(self.link.clone())));
        }
        if self.directory {
            fields.push(("directory".to_string(), Emit::Bool(true)));
        }
        Emit::Object(fields)
    }

    pub fn marshal(&self) -> String {
        marshal_indent(&self.emit())
    }

    pub fn decode(binder: &mut Strict<'_>) -> Result<File, String> {
        Ok(File {
            sha256: binder.string("SHA256")?,
            mode: binder.uint32("Mode")?,
            link: binder.string("Link")?,
            directory: binder.boolean("Directory")?,
        })
    }
}

/// Maps a Soda architecture name to its OCI form.
pub fn oci_architecture(arch: &str) -> Result<&'static str, Error> {
    if arch == "x86_64" {
        Ok("amd64")
    } else {
        Err(Error::msg("expected x86_64"))
    }
}

/// Lowercase hex SHA-256 digest shape.
pub fn is_digest(s: &str) -> bool {
    soda_build_tools::reader::is_digest(s)
}

/// Lowercase hex 40-char revision shape.
pub fn is_revision(s: &str) -> bool {
    soda_build_tools::reader::is_revision(s)
}

/// Requires the matching native Linux runtime.
pub fn require_native(arch: &str) -> Result<(), Error> {
    let want = oci_architecture(arch)?;
    if !cfg!(target_os = "linux") || std::env::consts::ARCH != "x86_64" || want != "amd64" {
        return Err(Error::msg("matching-native Linux required"));
    }
    Ok(())
}

/// SHA-256 of a regular non-symlink file.
pub fn hash_file(path: &Path) -> Result<String, Error> {
    let st = std::fs::symlink_metadata(path).map_err(|e| io_error("lstat", path, e))?;
    if !st.is_file() || st.file_type().is_symlink() {
        return Err(Error::msg("regular non-symlink file required"));
    }
    let mut f = FsFile::open(path).map_err(|e| io_error("open", path, e))?;
    sha256_hex_stream(&mut f).map_err(|e| io_error("read", path, e))
}

/// Creates a new private directory whose parent is real and unlinked.
pub fn fresh_directory(path: &Path) -> Result<(), Error> {
    if !path.is_absolute() {
        return Err(Error::msg("absolute new directory required"));
    }
    let parent = path.parent().unwrap_or(Path::new("/"));
    let resolved = std::fs::canonicalize(parent).map_err(|e| io_error("lstat", parent, e))?;
    let clean = PathBuf::from(crate::path_clean(&parent.to_string_lossy()));
    if resolved != clean {
        return Err(Error::msg("symlinked parent refused"));
    }
    std::fs::create_dir(path).map_err(|e| io_error("mkdir", path, e))?;
    chmod(path, 0o700)
}

/// Admits a private output path: absolute, under a real 0700-or-tighter
/// parent, not yet existing.
pub fn private_destination(path: &Path) -> Result<(), Error> {
    if !path.is_absolute() {
        return Err(Error::msg("absolute private output required"));
    }
    let parent = path.parent().unwrap_or(Path::new("/"));
    let resolved = std::fs::canonicalize(parent).map_err(|e| io_error("lstat", parent, e))?;
    let st = std::fs::metadata(parent).map_err(|e| io_error("stat", parent, e))?;
    if resolved != *parent || st.permissions().mode() & 0o077 != 0 {
        return Err(Error::msg("real private output parent required"));
    }
    match std::fs::symlink_metadata(path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        _ => Err(Error::msg("output already exists or cannot be inspected")),
    }
}

/// Creates a file exclusively and writes it fully.
pub fn write_new(path: &Path, data: &[u8], mode: u32) -> Result<(), Error> {
    let mut f = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path)
        .map_err(|e| io_error("open", path, e))?;
    let write_err = f.write_all(data).err();
    let close_err = f.sync_all().err();
    if let Some(e) = write_err {
        return Err(io_error("write", path, e));
    }
    if let Some(e) = close_err {
        return Err(io_error("write", path, e));
    }
    chmod(path, mode)
}

pub fn chmod(path: &Path, mode: u32) -> Result<(), Error> {
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(mode))
        .map_err(|e| io_error("chmod", path, e))
}

/// Absolute path helper mirroring `filepath.Abs` for the fetch paths.
pub fn abs_path(path: &Path) -> Result<PathBuf, Error> {
    if path.is_absolute() {
        return Ok(path.to_path_buf());
    }
    let cwd = std::env::current_dir().map_err(|e| io_error("getcwd", Path::new("."), e))?;
    Ok(cwd.join(path))
}
