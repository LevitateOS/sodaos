//! Native artifact support (`files.go`): validators, hashing, and
//! private-output admission. No product policy or release qualification.

use crate::json_emit::marshal_indent;
use crate::{io_error, sha256_hex_stream, Error};
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};
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
    pub fn marshal(&self) -> String {
        marshal_indent(&FileOutput {
            sha256: &self.sha256,
            mode: self.mode,
            link: &self.link,
            directory: self.directory,
        })
    }
}

#[derive(Serialize)]
struct FileOutput<'a> {
    #[serde(rename = "sha256", skip_serializing_if = "str::is_empty")]
    sha256: &'a str,
    mode: u32,
    #[serde(rename = "link", skip_serializing_if = "str::is_empty")]
    link: &'a str,
    #[serde(rename = "directory", skip_serializing_if = "is_false")]
    directory: bool,
}

fn is_false(value: &bool) -> bool {
    !value
}

impl<'de> Deserialize<'de> for File {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct FileVisitor;

        impl<'de> Visitor<'de> for FileVisitor {
            type Value = File;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("a file record object")
            }

            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<File, A::Error> {
                // Keep only the last spelling/value for each field before
                // applying its typed decode, matching encoding/json structs.
                let (mut sha256, mut mode, mut link, mut directory) = (None, None, None, None);
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("SHA256") {
                        sha256 = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("Mode") {
                        mode = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("Link") {
                        link = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("Directory") {
                        directory = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &["SHA256", "Mode", "Link", "Directory"],
                        ));
                    }
                }
                fn field<T: serde::de::DeserializeOwned + Default, E: de::Error>(
                    raw: Option<Box<serde_json::value::RawValue>>,
                ) -> Result<T, E> {
                    match raw {
                        None => Ok(T::default()),
                        Some(value) => serde_json::from_str::<Option<T>>(value.get())
                            .map(Option::unwrap_or_default)
                            .map_err(E::custom),
                    }
                }
                fn integer_field<T, E>(
                    raw: Option<Box<serde_json::value::RawValue>>,
                ) -> Result<T, E>
                where
                    T: TryFrom<i128> + Default,
                    <T as TryFrom<i128>>::Error: std::fmt::Display,
                    E: de::Error,
                {
                    match raw {
                        None => Ok(T::default()),
                        Some(raw) if raw.get() == "null" => Ok(T::default()),
                        Some(raw) => crate::json_input::parse_integer_token(&raw)
                            .ok_or_else(|| E::custom("invalid integer token"))
                            .and_then(|value| T::try_from(value).map_err(E::custom)),
                    }
                }
                Ok(File {
                    sha256: field(sha256)?,
                    mode: integer_field(mode)?,
                    link: field(link)?,
                    directory: field(directory)?,
                })
            }
        }

        deserializer.deserialize_map(FileVisitor)
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
