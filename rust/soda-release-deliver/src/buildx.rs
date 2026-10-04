//! Port of the `internal/release/build` surface `deliver` consumes:
//! digest/revision/architecture shapes (via `soda-build-tools`), the
//! Forgejo toolchain descriptor, the inspected image identity, and the
//! confined file helpers (`HashAt`, `FreshDirectory`, `PrivateDestination`,
//! `WriteNew`, `ReadJSONAt`).

use std::ffi::CString;
use std::os::fd::{AsRawFd, FromRawFd, OwnedFd};

use sha2::{Digest as _, Sha256};
use soda_json::JsonValue;

use crate::jsonx::{base64_decode, parse_lenient, Binder, Emit, Emitter};
use crate::Error;

pub use soda_build_tools::reader::{is_digest, is_revision, oci_architecture};

pub const FORGEJO_COMPILER_IMAGE: &str =
    "docker.io/library/golang@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468";
pub const FORGEJO_BUN_VERSION: &str = "1.4.2";
pub const FORGEJO_BUN_SHA256: &str =
    "4835eca59d6da70f4674f5642f6e459dcadab773695b2ed9922d131057989742";
pub const FORGEJO_UPSTREAM_BASE: &str = "15.0.9";
pub const FORGEJO_COMPAT_TOKEN: &str = "gitea-1.22.0";

/// `build.Image`: verified OCI image identity.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Image {
    pub manifest: String,
    pub config: String,
    pub architecture: String,
    pub revision: String,
    pub source: String,
    pub base_name: String,
    pub base_digest: String,
}

impl Image {
    pub fn decode(value: &JsonValue) -> Result<Image, String> {
        use crate::payload::decode_opt_string;
        let mut b = Binder::new(value).map_err(|_| "invalid image".to_string())?;
        let image = Image {
            manifest: decode_opt_string(&mut b, "Manifest")?,
            config: decode_opt_string(&mut b, "Config")?,
            architecture: decode_opt_string(&mut b, "Architecture")?,
            revision: decode_opt_string(&mut b, "Revision")?,
            source: decode_opt_string(&mut b, "Source")?,
            base_name: decode_opt_string(&mut b, "BaseName")?,
            base_digest: decode_opt_string(&mut b, "BaseDigest")?,
        };
        b.finish_name()?;
        Ok(image)
    }
}

impl Emit for Image {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Manifest");
        e.string(&self.manifest);
        e.field(false, "Config");
        e.string(&self.config);
        e.field(false, "Architecture");
        e.string(&self.architecture);
        e.field(false, "Revision");
        e.string(&self.revision);
        e.field(false, "Source");
        e.string(&self.source);
        e.field(false, "BaseName");
        e.string(&self.base_name);
        e.field(false, "BaseDigest");
        e.string(&self.base_digest);
        e.end_object(false);
    }
}

/// `build.ForgejoToolchain`: pinned compiler provenance.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ForgejoToolchain {
    pub compiler_image: String,
    pub apk_packages: Vec<String>,
}

impl ForgejoToolchain {
    pub fn validate(&self) -> Result<(), Error> {
        if self.compiler_image != FORGEJO_COMPILER_IMAGE
            || !valid_forgejo_apk_list(&self.apk_packages)
        {
            return Err(Error::msg("invalid Forgejo compiler provenance"));
        }
        if !has_forgejo_native_build_tools(&self.apk_packages) {
            return Err(Error::msg("incomplete Forgejo APK provenance"));
        }
        Ok(())
    }

    pub fn decode(value: &JsonValue) -> Result<ForgejoToolchain, String> {
        use crate::payload::decode_opt_string;
        let mut b = Binder::new(value).map_err(|_| "invalid toolchain".to_string())?;
        let mut toolchain = ForgejoToolchain {
            compiler_image: decode_opt_string(&mut b, "CompilerImage")?,
            apk_packages: Vec::new(),
        };
        if let Some(items) = b
            .array("APKPackages")
            .map_err(|_| "invalid field APKPackages".to_string())?
        {
            for item in items {
                match item {
                    JsonValue::Str(s) => toolchain.apk_packages.push(s.clone()),
                    _ => return Err("invalid field APKPackages".to_string()),
                }
            }
        }
        b.finish_name()?;
        Ok(toolchain)
    }
}

impl Emit for ForgejoToolchain {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "CompilerImage");
        e.string(&self.compiler_image);
        e.field(false, "APKPackages");
        e.begin_array(self.apk_packages.is_empty());
        for (i, package) in self.apk_packages.iter().enumerate() {
            e.item(i == 0);
            e.string(package);
        }
        e.end_array(self.apk_packages.is_empty());
        e.end_object(false);
    }
}

fn valid_forgejo_apk_list(packages: &[String]) -> bool {
    if packages.is_empty() || packages.len() > 256 {
        return false;
    }
    for (i, name) in packages.iter().enumerate() {
        if i > 0 && packages[i - 1] >= *name {
            return false;
        }
        if name.is_empty() || name.bytes().any(|c| matches!(c, b' ' | b'\t' | b'\n' | b'\r' | b'\\' | 0)) {
            return false;
        }
    }
    true
}

fn has_forgejo_native_build_tools(packages: &[String]) -> bool {
    let mut base = false;
    let mut gcc = false;
    let mut musl = false;
    for name in packages {
        base = base || name == "build-base-0.5-r4";
        gcc = gcc || name.starts_with("gcc-");
        musl = musl || name.starts_with("musl-dev-");
    }
    base && gcc && musl
}

fn os_error(op: &str, path: &str, err: std::io::Error) -> Error {
    Error::msg(format!("{op} {path}: {err}"))
}

/// Directory file descriptor confining resolutions the way `os.Root` does.
pub struct Root {
    fd: OwnedFd,
}

impl Root {
    pub fn open(path: &str) -> Result<Root, Error> {
        let fd = open_dir_fd(path).map_err(|e| os_error("open", path, e))?;
        Ok(Root { fd })
    }

    fn fd(&self) -> i32 {
        self.fd.as_raw_fd()
    }
}

fn open_dir_fd(path: &str) -> std::io::Result<OwnedFd> {
    let cpath = CString::new(path).map_err(|_| {
        std::io::Error::new(std::io::ErrorKind::InvalidInput, "invalid path")
    })?;
    // SAFETY: open(2) with a NUL-terminated path; result checked.
    let fd = unsafe {
        libc::open(
            cpath.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(std::io::Error::last_os_error());
    }
    // SAFETY: fd is a fresh owned descriptor.
    Ok(unsafe { OwnedFd::from_raw_fd(fd) })
}

fn check_confined(name: &str) -> Result<(), Error> {
    if name.is_empty() || name.starts_with('/') {
        return Err(Error::refused());
    }
    let mut depth = 0i32;
    for segment in name.split('/') {
        match segment {
            "" | "." => {}
            ".." => {
                depth -= 1;
                if depth < 0 {
                    return Err(Error::refused());
                }
            }
            _ => depth += 1,
        }
    }
    Ok(())
}

fn c_string(name: &str) -> Result<CString, Error> {
    CString::new(name).map_err(|_| Error::refused())
}

fn fstatat_no_follow(dirfd: i32, name: &CString) -> std::io::Result<libc::stat> {
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    // SAFETY: fstatat on a live dirfd with a valid path and output struct.
    let rc = unsafe {
        libc::fstatat(
            dirfd,
            name.as_ptr(),
            &mut st,
            libc::AT_SYMLINK_NOFOLLOW,
        )
    };
    if rc != 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(st)
}

fn fstat_fd(fd: i32) -> std::io::Result<libc::stat> {
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    // SAFETY: fstat on a live descriptor with an output struct.
    if unsafe { libc::fstat(fd, &mut st) } != 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(st)
}

fn is_regular(mode: u32) -> bool {
    (mode & libc::S_IFMT as u32) == libc::S_IFREG as u32
}

/// `build.HashAt`: hash a regular file confined to an open directory.
pub fn hash_at(root: &Root, name: &str) -> Result<String, Error> {
    check_confined(name)?;
    let cname = c_string(name)?;
    let st = fstatat_no_follow(root.fd(), &cname)
        .map_err(|e| Error::msg(format!("lstat {name}: {e}")))?;
    if !is_regular(st.st_mode) {
        return Err(Error::msg("regular non-symlink file required"));
    }
    // SAFETY: openat on a live dirfd; flags mirror Go's O_RDONLY|O_NONBLOCK.
    let fd = unsafe {
        libc::openat(
            root.fd(),
            cname.as_ptr(),
            libc::O_RDONLY | libc::O_NONBLOCK | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(Error::msg(format!(
            "open {name}: {}",
            std::io::Error::last_os_error()
        )));
    }
    // SAFETY: fd is a fresh owned descriptor.
    let owned = unsafe { OwnedFd::from_raw_fd(fd) };
    let actual = fstat_fd(owned.as_raw_fd())
        .map_err(|e| Error::msg(format!("stat {name}: {e}")))?;
    if !is_regular(actual.st_mode) || actual.st_dev != st.st_dev || actual.st_ino != st.st_ino {
        return Err(Error::msg("file changed before hashing"));
    }
    let mut file = std::fs::File::from(owned);
    let mut hasher = Sha256::new();
    std::io::copy(&mut file, &mut HashWriter(&mut hasher))
        .map_err(|e| Error::msg(format!("read {name}: {e}")))?;
    Ok(format!("{:x}", hasher.finalize()))
}

struct HashWriter<'a>(&'a mut Sha256);

impl std::io::Write for HashWriter<'_> {
    fn write(&mut self, buf: &[u8]) -> std::io::Result<usize> {
        self.0.update(buf);
        Ok(buf.len())
    }
    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

fn clean_path(path: &str) -> String {
    // Lexical clean mirroring filepath.Clean for absolute paths.
    let mut parts: Vec<&str> = Vec::new();
    for segment in path.split('/') {
        match segment {
            "" | "." => {}
            ".." => {
                parts.pop();
            }
            _ => parts.push(segment),
        }
    }
    format!("/{}", parts.join("/"))
}

/// `build.FreshDirectory`: create a new private directory under a real parent.
pub fn fresh_directory(path: &str) -> Result<(), Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute new directory required"));
    }
    let parent = match path.rfind('/') {
        Some(0) | None => "/".to_string(),
        Some(i) => path[..i].to_string(),
    };
    let resolved = std::fs::canonicalize(&parent)
        .map_err(|e| Error::msg(format!("readlink {parent}: {e}")))?;
    if resolved.to_string_lossy() != clean_path(&parent) {
        return Err(Error::msg("symlinked parent refused"));
    }
    use std::os::unix::fs::DirBuilderExt;
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(path)
        .map_err(|e| os_error("mkdir", path, e))
}

/// `build.PrivateDestination`: admit a not-yet-existing private output path.
pub fn private_destination(path: &str) -> Result<(), Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute private output required"));
    }
    let parent = match path.rfind('/') {
        Some(0) | None => "/".to_string(),
        Some(i) => path[..i].to_string(),
    };
    let resolved = std::fs::canonicalize(&parent)
        .map_err(|e| Error::msg(format!("readlink {parent}: {e}")))?;
    let st = std::fs::metadata(&parent).map_err(|e| os_error("stat", &parent, e))?;
    use std::os::unix::fs::MetadataExt;
    if resolved.to_string_lossy() != parent || st.mode() & 0o077 != 0 {
        return Err(Error::msg("real private output parent required"));
    }
    match std::fs::symlink_metadata(path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        _ => Err(Error::msg("output already exists or cannot be inspected")),
    }
}

/// `build.WriteNew`: create a file that must not already exist.
pub fn write_new(path: &str, data: &[u8], mode: u32) -> Result<(), Error> {
    use std::os::unix::fs::OpenOptionsExt;
    let mut file = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(mode)
        .open(path)
        .map_err(|e| os_error("open", path, e))?;
    use std::io::Write;
    let write_err = file.write_all(data).err();
    let close_err = file.sync_all().err();
    match (write_err, close_err) {
        (None, None) => Ok(()),
        (Some(e), None) => Err(Error::msg(format!("write {path}: {e}"))),
        (None, Some(e)) => Err(Error::msg(format!("sync {path}: {e}"))),
        (Some(e1), Some(e2)) => Err(Error::msg(format!("write {path}: {e1}; sync: {e2}"))),
    }
}

/// `build.ReadJSONAt` decode step: bounded strict-shape JSON with Go's
/// unknown-field error text. Duplicate keys keep Go's last-wins rule here
/// (unlike `strictjson`), matching `encoding/json` exactly.
pub fn decode_build_json(data: &[u8]) -> Result<JsonValue, Error> {
    if data.len() > 4 << 20 {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    parse_lenient(data).map_err(|_| Error::msg("invalid JSON input"))
}

/// Read and decode bounded JSON confined to an open directory, returning
/// the digest of the exact bytes decoded.
pub fn read_json_at(
    root: &Root,
    name: &str,
    decode: &mut dyn FnMut(&JsonValue) -> Result<(), String>,
) -> Result<String, Error> {
    check_confined(name)?;
    let cname = c_string(name)?;
    let st = fstatat_no_follow(root.fd(), &cname)
        .map_err(|e| Error::msg(format!("lstat {name}: {e}")))?;
    if !is_regular(st.st_mode) || st.st_size > 4 << 20 {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    // SAFETY: openat on a live dirfd; flags mirror Go's O_RDONLY|O_NONBLOCK.
    let fd = unsafe {
        libc::openat(
            root.fd(),
            cname.as_ptr(),
            libc::O_RDONLY | libc::O_NONBLOCK | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(Error::msg(format!(
            "open {name}: {}",
            std::io::Error::last_os_error()
        )));
    }
    // SAFETY: fd is a fresh owned descriptor.
    let owned = unsafe { OwnedFd::from_raw_fd(fd) };
    let actual = fstat_fd(owned.as_raw_fd())
        .map_err(|e| Error::msg(format!("stat {name}: {e}")))?;
    if !is_regular(actual.st_mode) || actual.st_dev != st.st_dev || actual.st_ino != st.st_ino
    {
        return Err(Error::msg("JSON input changed before reading"));
    }
    let file = std::fs::File::from(owned);
    let mut data = Vec::new();
    use std::io::Read;
    file.take((4 << 20) + 1)
        .read_to_end(&mut data)
        .map_err(|e| Error::msg(format!("read {name}: {e}")))?;
    if data.len() > 4 << 20 {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    let value = decode_build_json(&data)?;
    decode(&value).map_err(Error::msg)?;
    let mut hasher = Sha256::new();
    hasher.update(&data);
    Ok(format!("{:x}", hasher.finalize()))
}

/// Unknown-field error text matching Go's `encoding/json`.
pub fn unknown_field(name: &str) -> String {
    format!("json: unknown field \"{name}\"")
}

pub fn decode_bytes_value(value: &JsonValue) -> Result<Vec<u8>, ()> {
    match value {
        JsonValue::Str(s) => base64_decode(s),
        _ => Err(()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn toolchain_validation_matches_go() {
        let good = ForgejoToolchain {
            compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
            apk_packages: [
                "build-base-0.5-r4",
                "gcc-14.2.0-r6",
                "musl-dev-1.2.5-r10",
            ]
            .iter()
            .map(|s| s.to_string())
            .collect(),
        };
        assert!(good.validate().is_ok());
        let bad = ForgejoToolchain {
            compiler_image: "other".to_string(),
            apk_packages: good.apk_packages.clone(),
        };
        assert_eq!(
            bad.validate().unwrap_err(),
            Error::msg("invalid Forgejo compiler provenance")
        );
        let unsorted = ForgejoToolchain {
            compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
            apk_packages: ["gcc-14.2.0-r6", "build-base-0.5-r4"]
                .iter()
                .map(|s| s.to_string())
                .collect(),
        };
        assert!(unsorted.validate().is_err());
        let missing = ForgejoToolchain {
            compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
            apk_packages: ["build-base-0.5-r4".to_string()].to_vec(),
        };
        assert_eq!(
            missing.validate().unwrap_err(),
            Error::msg("incomplete Forgejo APK provenance")
        );
    }

    #[test]
    fn file_helpers_match_go_errors() {
        assert_eq!(
            fresh_directory("relative/path").unwrap_err(),
            Error::msg("absolute new directory required")
        );
        assert_eq!(
            private_destination("relative").unwrap_err(),
            Error::msg("absolute private output required")
        );
        let dir = std::env::temp_dir().join(format!("srd-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let target = dir.join("fresh");
        fresh_directory(target.to_str().unwrap()).unwrap();
        assert!(fresh_directory(target.to_str().unwrap()).is_err());
        std::fs::remove_dir_all(&dir).unwrap();
    }
}
