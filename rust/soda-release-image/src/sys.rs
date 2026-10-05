//! Small exact mirrors of the foreign `release/build` + `release/deliver`
//! helpers the pipeline calls: path lexics, file hashing, exclusive writes,
//! fresh directories, bounded strict JSON reads, native admission, command
//! inventory, and the private-file check. Heavy foreign operations (OCI,
//! signing, live resolution) live behind [`crate::foreign::Production`].

use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Component, Path, PathBuf};

use sha2::{Digest, Sha256};
use soda_json::JsonValue;

use crate::error::Error;
use crate::jsonio;

/// Shared inventory primitive (`build.File`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct File {
    pub sha256: String,
    pub mode: u32,
    pub link: String,
    pub directory: bool,
}

impl File {
    pub fn to_json(&self) -> JsonValue {
        let mut entries = Vec::new();
        if !self.sha256.is_empty() {
            entries.push(("sha256".to_string(), JsonValue::Str(self.sha256.clone())));
        }
        entries.push(("mode".to_string(), JsonValue::Number(self.mode.to_string())));
        if !self.link.is_empty() {
            entries.push(("link".to_string(), JsonValue::Str(self.link.clone())));
        }
        if self.directory {
            entries.push(("directory".to_string(), JsonValue::Bool(true)));
        }
        JsonValue::Object(entries)
    }

    pub fn parse(value: &JsonValue) -> Result<File, Error> {
        jsonio::check_no_unknown(value, &["sha256", "mode", "link", "directory"])?;
        Ok(File {
            sha256: jsonio::require_string(value, "sha256")?,
            mode: jsonio::require_u64(value, "mode")? as u32,
            link: jsonio::require_string(value, "link")?,
            directory: jsonio::require_bool(value, "directory")?,
        })
    }
}

/// Go `path/filepath.Clean` lexics.
pub fn clean_path(path: &str) -> String {
    if path.is_empty() {
        return ".".to_string();
    }
    let rooted = path.starts_with('/');
    let mut out: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if out.pop().is_none() && !rooted {
                    out.push("..");
                }
            }
            _ => out.push(part),
        }
    }
    let mut joined = out.join("/");
    if rooted {
        joined.insert(0, '/');
    }
    if joined.is_empty() {
        return if rooted {
            "/".to_string()
        } else {
            ".".to_string()
        };
    }
    joined
}

pub fn is_abs(path: &str) -> bool {
    path.starts_with('/')
}

pub fn join<S: AsRef<str>>(parts: &[S]) -> String {
    let mut buf = PathBuf::new();
    for part in parts {
        if part.as_ref().is_empty() {
            continue;
        }
        buf.push(part.as_ref());
    }
    clean_path(&buf.to_string_lossy())
}

pub fn dir_name(path: &str) -> String {
    let cleaned = clean_path(path);
    match cleaned.rfind('/') {
        None => ".".to_string(),
        Some(0) => "/".to_string(),
        Some(i) => cleaned[..i].to_string(),
    }
}

pub fn base_name(path: &str) -> String {
    let cleaned = clean_path(path);
    if cleaned == "/" {
        return "/".to_string();
    }
    cleaned.rsplit('/').next().unwrap_or("").to_string()
}

/// Go `filepath.Rel`: lexical relative path or an error.
pub fn rel_path(base: &str, target: &str) -> Result<String, Error> {
    let b = clean_path(base);
    let t = clean_path(target);
    let b_parts: Vec<&str> = b.split('/').filter(|p| !p.is_empty()).collect();
    let t_parts: Vec<&str> = t.split('/').filter(|p| !p.is_empty()).collect();
    let mut common = 0;
    while common < b_parts.len() && common < t_parts.len() && b_parts[common] == t_parts[common] {
        common += 1;
    }
    if common == 0 && (b.starts_with('/') || t.starts_with('/')) {
        return Err(Error::msg("cannot make relative path"));
    }
    let mut out = vec![".."; b_parts.len() - common];
    out.extend_from_slice(&t_parts[common..]);
    if out.is_empty() {
        return Ok(".".to_string());
    }
    Ok(out.join("/"))
}

pub fn to_slash(path: &str) -> String {
    path.to_string()
}

/// Lexical path components without `.`/`..` surprises, for walk inventories.
pub fn components(path: &Path) -> Vec<String> {
    path.components()
        .filter_map(|c| match c {
            Component::Normal(part) => Some(part.to_string_lossy().into_owned()),
            _ => None,
        })
        .collect()
}

/// `build.HashFile`: SHA-256 of a regular non-symlink file.
pub fn hash_file(path: &str) -> Result<String, Error> {
    let meta = fs::symlink_metadata(path)?;
    if !meta.file_type().is_file() {
        return Err(Error::msg("regular non-symlink file required"));
    }
    let data = fs::read(path)?;
    Ok(hex_sha256(&data))
}

pub fn hex_sha256(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    hex_bytes(&hasher.finalize())
}

pub fn hex_bytes(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        out.push_str(&format!("{b:02x}"));
    }
    out
}

/// `build.FreshDirectory`: create a new mode-700 directory below a real parent.
pub fn fresh_directory(path: &str) -> Result<(), Error> {
    if !is_abs(path) {
        return Err(Error::msg("absolute new directory required"));
    }
    let parent = dir_name(path);
    let resolved = fs::canonicalize(&parent).map_err(|e| Error::msg(e.to_string()))?;
    if resolved.to_string_lossy() != clean_path(&parent) {
        return Err(Error::msg("symlinked parent refused"));
    }
    fs::create_dir(path).map_err(|e| Error::msg(e.to_string()))?;
    fs::set_permissions(path, fs::Permissions::from_mode(0o700))?;
    Ok(())
}

/// `build.WriteNew`: exclusive-create write; refuses to overwrite.
pub fn write_new(path: &str, data: &[u8], mode: u32) -> Result<(), Error> {
    use std::fs::OpenOptions;
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path)
        .map_err(|e| Error::msg(e.to_string()))?;
    use std::io::Write;
    let write_result = file.write_all(data).map_err(Error::from);
    drop(file);
    write_result?;
    fs::set_permissions(path, fs::Permissions::from_mode(mode))?;
    Ok(())
}

fn read_bounded(path: &str, maximum: usize, too_big: &str) -> Result<Vec<u8>, Error> {
    let meta = fs::symlink_metadata(path)?;
    if !meta.file_type().is_file() || meta.len() > maximum as u64 {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    let data = fs::read(path)?;
    if data.len() > maximum {
        return Err(Error::msg(too_big));
    }
    Ok(data)
}

/// `build.ReadJSON`: bounded (4 MiB) strict read of a regular file.
/// (Go additionally pins the file through an `os.Root`; same bytes here.)
pub fn read_json_build(path: &str) -> Result<JsonValue, Error> {
    let data = read_bounded(path, 4 << 20, "JSON input exceeds limit")?;
    let text = std::str::from_utf8(&data).map_err(|_| Error::msg("invalid JSON"))?;
    jsonio::parse(text)
}

/// `deliver.ReadJSON`: bounded (1 MiB) strict read.
pub fn read_json_deliver(path: &str) -> Result<JsonValue, Error> {
    let meta = fs::symlink_metadata(path).map_err(|_| Error::msg(refused()))?;
    if !meta.file_type().is_file() || meta.len() > (1 << 20) as u64 {
        return Err(Error::msg(refused()));
    }
    let data = fs::read(path).map_err(|_| Error::msg(refused()))?;
    if data.len() > 1 << 20 {
        return Err(Error::msg(refused()));
    }
    let text = std::str::from_utf8(&data).map_err(|_| Error::msg(refused()))?;
    jsonio::parse(text).map_err(|_| Error::msg(refused()))
}

pub fn refused() -> String {
    "release authority or completeness refused".to_string()
}

/// `deliver.PrivateFile`: absolute, symlink-free, owned private regular file.
pub fn private_file(path: &str) -> Result<(), Error> {
    if !is_abs(path) {
        return Err(Error::msg(refused()));
    }
    let resolved = fs::canonicalize(path).map_err(|_| Error::msg(refused()))?;
    if resolved.to_string_lossy() != clean_path(path) {
        return Err(Error::msg(refused()));
    }
    let meta = fs::symlink_metadata(path).map_err(|_| Error::msg(refused()))?;
    if !meta.file_type().is_file() || meta.permissions().mode() & 0o077 != 0 {
        return Err(Error::msg(refused()));
    }
    Ok(())
}

/// `build.RequireNative`: x86_64 Linux only.
pub fn require_native(arch: &str) -> Result<(), Error> {
    let oci =
        soda_build_tools::reader::oci_architecture(arch).map_err(|e| Error::msg(e.to_string()))?;
    // D01-F1: Rust target arch and OCI arch are separate namespaces; each
    // is validated against its own supported value.
    if std::env::consts::OS != "linux" || std::env::consts::ARCH != "x86_64" || oci != "amd64" {
        return Err(Error::msg("matching-native Linux required"));
    }
    Ok(())
}

/// `build.SodaCommands`: sorted `cmd/soda-*` directories, tools excluded.
pub fn soda_commands(source: &str) -> Result<Vec<String>, Error> {
    let mut names = Vec::new();
    let entries = fs::read_dir(join(&[source, "cmd"]))?;
    let mut dirs: Vec<String> = entries
        .filter_map(|entry| entry.ok())
        .filter(|entry| entry.file_type().map(|t| t.is_dir()).unwrap_or(false))
        .map(|entry| entry.file_name().to_string_lossy().into_owned())
        .collect();
    dirs.sort();
    for name in dirs {
        if !is_soda_command(&name) || name == "soda-artifacts" || name == "soda-acceptance" {
            return Err(Error::msg(
                "support tools must remain outside appliance commands",
            ));
        }
        names.push(name);
    }
    if names.is_empty() {
        return Err(Error::msg("missing Soda commands"));
    }
    Ok(names)
}

fn is_soda_command(name: &str) -> bool {
    let rest = match name.strip_prefix("soda-") {
        Some(rest) => rest,
        None => return false,
    };
    !rest.is_empty()
        && rest
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
}

/// Mount mode bits for a freshly created directory (`os.MkdirAll` parity).
pub fn mkdir_all(path: &str, mode: u32) -> Result<(), Error> {
    fs::create_dir_all(path)?;
    fs::set_permissions(path, fs::Permissions::from_mode(mode))?;
    Ok(())
}

pub fn create_dir(path: &str, mode: u32) -> Result<(), Error> {
    fs::create_dir(path).map_err(|e| Error::msg(e.to_string()))?;
    fs::set_permissions(path, fs::Permissions::from_mode(mode))?;
    Ok(())
}

/// Recursive directory walk yielding (path, is_dir, is_symlink).
pub fn walk<F>(root: &str, mut visit: F) -> Result<(), Error>
where
    F: FnMut(&str, bool, bool) -> Result<(), Error>,
{
    let mut stack = vec![root.to_string()];
    while let Some(path) = stack.pop() {
        let meta = fs::symlink_metadata(&path)?;
        let is_link = meta.file_type().is_symlink();
        let is_dir = meta.file_type().is_dir();
        visit(&path, is_dir, is_link)?;
        if is_dir && !is_link {
            let mut entries: Vec<String> = fs::read_dir(&path)?
                .filter_map(|entry| entry.ok())
                .map(|entry| entry.path().to_string_lossy().into_owned())
                .collect();
            entries.sort();
            entries.reverse();
            stack.extend(entries);
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_clean_path_matches_go() {
        // Oracle: Go filepath.Clean vectors.
        assert_eq!(clean_path(""), ".");
        assert_eq!(clean_path("/a//b/./c/../d"), "/a/b/d");
        assert_eq!(clean_path("a/../../b"), "../b");
        assert_eq!(join(&["/a", "b", "c"]), "/a/b/c");
        assert_eq!(dir_name("/a/b"), "/a");
        assert_eq!(base_name("/a/b"), "b");
        assert_eq!(rel_path("/a/b", "/a/b/c/d").unwrap(), "c/d");
        assert_eq!(rel_path("/a/b/c", "/a/d").unwrap(), "../../d");
    }

    #[test]
    fn oracle_hash_file_refuses_symlink() {
        let dir = std::env::temp_dir().join(format!("sri-sys-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        let target = dir.join("f");
        fs::write(&target, b"bytes").unwrap();
        let link = dir.join("l");
        std::os::unix::fs::symlink(&target, &link).unwrap();
        assert_eq!(
            hash_file(target.to_str().unwrap()).unwrap(),
            hex_sha256(b"bytes")
        );
        assert_eq!(
            hash_file(link.to_str().unwrap()).unwrap_err().0,
            "regular non-symlink file required"
        );
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn require_native_accepts_matching_x86_64_linux() {
        // D01-F1: Rust target arch and OCI arch are separate namespaces;
        // matching-native x86_64 Linux must pass, anything else must fail.
        assert!(require_native("x86_64").is_ok());
        assert!(require_native("amd64").is_err());
        assert!(require_native("arm64").is_err());
        assert!(require_native("").is_err());
    }

    #[test]
    fn oracle_write_new_refuses_overwrite() {
        let dir = std::env::temp_dir().join(format!("sri-wn-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        let path = dir.join("n");
        write_new(path.to_str().unwrap(), b"one", 0o600).unwrap();
        assert!(write_new(path.to_str().unwrap(), b"two", 0o600).is_err());
        assert_eq!(fs::read(&path).unwrap(), b"one");
        let _ = fs::remove_dir_all(&dir);
    }
}
