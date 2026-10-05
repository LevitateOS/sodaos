//! Native artifact support (`files.go`): validators, hashing, confined
//! reads, and private-output admission. No product policy or release
//! qualification.

use crate::json_go::{marshal_indent, Emit, Strict};
use crate::{io_error, sha256_hex_stream, Error};
use soda_json::JsonValue;
use std::fs::{File as FsFile, OpenOptions};
use std::io::{Read, Write};
use std::os::fd::{AsFd, AsRawFd, FromRawFd, IntoRawFd, OwnedFd};
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};

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

/// SHA-256 through a confined root, refusing mid-read swaps.
pub fn hash_at(root: &Root, name: &str) -> Result<String, Error> {
    let st = root.lstat(name)?;
    if !st.is_regular {
        return Err(Error::msg("regular non-symlink file required"));
    }
    let mut f = root.open_file(name)?;
    let actual = f
        .metadata()
        .map_err(|e| io_error("stat", Path::new(name), e))?;
    if !is_regular_meta(&actual) || !same_file_meta(&st, &actual) {
        return Err(Error::msg("file changed before hashing"));
    }
    sha256_hex_stream(&mut f).map_err(|e| io_error("read", Path::new(name), e))
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

/// Strict bounded JSON read with digest of the exact decoded bytes.
pub fn read_json<T>(
    path: &Path,
    target: &'static str,
    decode: impl FnOnce(&mut Strict<'_>) -> Result<T, String>,
) -> Result<T, Error> {
    let parent = path.parent().unwrap_or(Path::new("/"));
    let root = Root::open(parent)?;
    let name = path
        .file_name()
        .map(|n| n.to_string_lossy().into_owned())
        .unwrap_or_default();
    read_json_at(&root, &name, target, decode).map(|(value, _)| value)
}

/// Strict bounded JSON read through a confined root; returns the value and
/// the digest of the exact bounded bytes decoded.
pub fn read_json_at<T>(
    root: &Root,
    name: &str,
    target: &'static str,
    decode: impl FnOnce(&mut Strict<'_>) -> Result<T, String>,
) -> Result<(T, String), Error> {
    const LIMIT: u64 = 4 << 20;
    let st = root.lstat(name)?;
    if !st.is_regular || st.size > LIMIT {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    let f = root.open_unchanged(name, &st)?;
    let mut data = Vec::new();
    f.take(LIMIT + 1)
        .read_to_end(&mut data)
        .map_err(|e| io_error("read", Path::new(name), e))?;
    if data.len() as u64 > LIMIT {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    let text =
        std::str::from_utf8(&data).map_err(|_| Error::msg("invalid character in JSON input"))?;
    let value = match JsonValue::parse(text) {
        Ok(value) => value,
        Err(_) => match first_json_end(text) {
            // A complete value followed by more data: Go's decoder
            // reports trailing data rather than a syntax failure.
            Some(end) if !text[end..].trim().is_empty() => {
                return Err(Error::msg("trailing JSON data"))
            }
            _ => return Err(Error::msg("invalid character in JSON input")),
        },
    };
    let mut binder = Strict::bind(&value, target).map_err(Error::msg)?;
    let decoded = decode(&mut binder).map_err(Error::msg)?;
    binder.finish().map_err(Error::msg)?;
    Ok((decoded, crate::sha256_hex(&data)))
}

/// Confined directory handle emulating Go's `os.Root`: every resolution
/// stays inside the opened directory; intermediate symlinks are refused.
pub struct Root {
    fd: OwnedFd,
}

#[derive(Debug, Clone)]
pub struct FileMeta {
    pub is_regular: bool,
    pub size: u64,
    pub dev: u64,
    pub ino: u64,
}

#[allow(clippy::unnecessary_cast)] // dev_t/ino_t widths are platform-dependent
fn stat_to_meta(st: &libc::stat) -> FileMeta {
    FileMeta {
        is_regular: (st.st_mode & libc::S_IFMT) == libc::S_IFREG,
        size: st.st_size.max(0) as u64,
        dev: st.st_dev as u64,
        ino: st.st_ino as u64,
    }
}

impl Root {
    /// Opens a real directory without following a final symlink.
    pub fn open(dir: &Path) -> Result<Root, Error> {
        let st = std::fs::symlink_metadata(dir).map_err(|e| io_error("lstat", dir, e))?;
        if !st.is_dir() || st.file_type().is_symlink() {
            return Err(Error::msg(format!(
                "open {}: not a directory",
                dir.display()
            )));
        }
        let fd = open_at(
            libc::AT_FDCWD,
            dir,
            libc::O_PATH | libc::O_DIRECTORY | libc::O_NOFOLLOW,
        )?;
        Ok(Root { fd })
    }

    fn resolve_parent(&self, name: &str) -> Result<(OwnedFd, String), Error> {
        use std::os::fd::BorrowedFd;
        if name.is_empty() || name.starts_with('/') {
            return Err(Error::msg("confined path escapes root"));
        }
        let parts: Vec<&str> = name.split('/').collect();
        if parts
            .iter()
            .any(|p| p.is_empty() || *p == "." || *p == "..")
        {
            return Err(Error::msg("confined path escapes root"));
        }
        let mut dir_fd: BorrowedFd<'_> = self.fd.as_fd();
        let mut owned: Vec<OwnedFd> = Vec::new();
        for component in &parts[..parts.len() - 1] {
            let child = open_at_fd(
                dir_fd,
                component,
                libc::O_PATH | libc::O_DIRECTORY | libc::O_NOFOLLOW,
            )
            .map_err(|e| Error::msg(format!("open {component}: {e}")))?;
            owned.push(child);
            dir_fd = owned.last().unwrap().as_fd();
        }
        // Duplicate the final dirfd so the returned fd is independent.
        let raw = dir_fd.as_raw_fd();
        let duped = unsafe { libc::fcntl(raw, libc::F_DUPFD_CLOEXEC, 0) };
        if duped < 0 {
            return Err(Error::msg(format!(
                "open {name}: {}",
                std::io::Error::last_os_error()
            )));
        }
        let out = unsafe { OwnedFd::from_raw_fd(duped) };
        Ok((out, parts[parts.len() - 1].to_string()))
    }

    /// `lstat` confined to the root.
    pub fn lstat(&self, name: &str) -> Result<FileMeta, Error> {
        let (dir, leaf) = self.resolve_parent(name)?;
        let leaf_c = std::ffi::CString::new(leaf.as_str()).unwrap();
        let mut st: libc::stat = unsafe { std::mem::zeroed() };
        let rc = unsafe {
            libc::fstatat(
                dir.as_raw_fd(),
                leaf_c.as_ptr(),
                &mut st,
                libc::AT_SYMLINK_NOFOLLOW,
            )
        };
        if rc != 0 {
            return Err(Error::msg(format!(
                "lstat {name}: {}",
                std::io::Error::last_os_error()
            )));
        }
        Ok(stat_to_meta(&st))
    }

    /// Opens a regular file without following a final symlink.
    pub fn open_file(&self, name: &str) -> Result<FsFile, Error> {
        let (dir, leaf) = self.resolve_parent(name)?;
        let fd = open_at_fd(
            dir.as_fd(),
            &leaf,
            libc::O_RDONLY | libc::O_NONBLOCK | libc::O_NOFOLLOW,
        )
        .map_err(|e| Error::msg(format!("open {name}: {e}")))?;
        Ok(unsafe { FsFile::from(OwnedFd::from_raw_fd(fd.into_raw_fd())) })
    }

    /// Opens `name` only if it is still the same regular file.
    pub fn open_unchanged(&self, name: &str, st: &FileMeta) -> Result<FsFile, Error> {
        let f = self.open_file(name)?;
        let actual = f
            .metadata()
            .map_err(|e| io_error("stat", Path::new(name), e))?;
        if !is_regular_meta(&actual) || !same_file_meta(st, &actual) {
            return Err(Error::msg("JSON input changed before reading"));
        }
        Ok(f)
    }
}

fn open_at(dirfd: libc::c_int, path: &Path, flags: libc::c_int) -> Result<OwnedFd, Error> {
    let c = std::ffi::CString::new(path.as_os_str().as_bytes())
        .map_err(|_| Error::msg(format!("open {}: invalid path", path.display())))?;
    let raw = unsafe { libc::openat(dirfd, c.as_ptr(), flags | libc::O_CLOEXEC, 0) };
    if raw < 0 {
        return Err(Error::msg(format!(
            "open {}: {}",
            path.display(),
            std::io::Error::last_os_error()
        )));
    }
    Ok(unsafe { OwnedFd::from_raw_fd(raw) })
}

fn open_at_fd(dirfd: impl AsFd, leaf: &str, flags: libc::c_int) -> Result<OwnedFd, std::io::Error> {
    let c = std::ffi::CString::new(leaf)
        .map_err(|e| std::io::Error::new(std::io::ErrorKind::InvalidInput, e.to_string()))?;
    let raw = unsafe {
        libc::openat(
            dirfd.as_fd().as_raw_fd(),
            c.as_ptr(),
            flags | libc::O_CLOEXEC,
            0,
        )
    };
    if raw < 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(unsafe { OwnedFd::from_raw_fd(raw) })
}

fn is_regular_meta(meta: &std::fs::Metadata) -> bool {
    meta.is_file() && !meta.file_type().is_symlink()
}

fn same_file_meta(st: &FileMeta, meta: &std::fs::Metadata) -> bool {
    use std::os::unix::fs::MetadataExt;
    st.dev == meta.dev() && st.ino == meta.ino()
}

/// End offset of the first JSON value in `text`, for trailing-data
/// classification. Lenient where the strict parser is not: only the
/// value/trailing split matters here.
fn first_json_end(text: &str) -> Option<usize> {
    skip_value(text.as_bytes(), skip_ws(text.as_bytes(), 0))
}

fn skip_ws(bytes: &[u8], mut i: usize) -> usize {
    while i < bytes.len() && matches!(bytes[i], b' ' | b'\t' | b'\n' | b'\r') {
        i += 1;
    }
    i
}

fn skip_string(bytes: &[u8], mut i: usize) -> Option<usize> {
    if bytes.get(i) != Some(&b'"') {
        return None;
    }
    i += 1;
    while i < bytes.len() {
        match bytes[i] {
            b'"' => return Some(i + 1),
            b'\\' => i += 2,
            _ => i += 1,
        }
    }
    None
}

fn skip_number(bytes: &[u8], mut i: usize) -> Option<usize> {
    if bytes.get(i) == Some(&b'-') {
        i += 1;
    }
    let start = i;
    while i < bytes.len()
        && (bytes[i].is_ascii_digit() || matches!(bytes[i], b'.' | b'e' | b'E' | b'+' | b'-'))
    {
        i += 1;
    }
    if i == start {
        return None;
    }
    Some(i)
}

fn skip_literal(bytes: &[u8], i: usize, word: &[u8]) -> Option<usize> {
    if bytes.get(i..i + word.len()) == Some(word) {
        Some(i + word.len())
    } else {
        None
    }
}

fn skip_value(bytes: &[u8], i: usize) -> Option<usize> {
    let open = *bytes.get(i)?;
    match open {
        b'{' | b'[' => {
            let close = if open == b'{' { b'}' } else { b']' };
            let mut i = skip_ws(bytes, i + 1);
            if bytes.get(i) == Some(&close) {
                return Some(i + 1);
            }
            loop {
                if open == b'{' {
                    i = skip_string(bytes, i)?;
                    i = skip_ws(bytes, i);
                    if bytes.get(i) != Some(&b':') {
                        return None;
                    }
                    i = skip_ws(bytes, i + 1);
                }
                i = skip_value(bytes, i)?;
                i = skip_ws(bytes, i);
                match bytes.get(i) {
                    Some(b',') => i = skip_ws(bytes, i + 1),
                    Some(c) if *c == close => return Some(i + 1),
                    _ => return None,
                }
            }
        }
        b'"' => skip_string(bytes, i),
        b't' => skip_literal(bytes, i, b"true"),
        b'f' => skip_literal(bytes, i, b"false"),
        b'n' => skip_literal(bytes, i, b"null"),
        b'-' | b'0'..=b'9' => skip_number(bytes, i),
        _ => None,
    }
}

/// Absolute path helper mirroring `filepath.Abs` for the fetch paths.
pub fn abs_path(path: &Path) -> Result<PathBuf, Error> {
    if path.is_absolute() {
        return Ok(path.to_path_buf());
    }
    let cwd = std::env::current_dir().map_err(|e| io_error("getcwd", Path::new("."), e))?;
    Ok(cwd.join(path))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;

    fn scratch() -> PathBuf {
        let dir =
            std::env::temp_dir().join(format!("soda-relbuild-{}-{}", std::process::id(), unique()));
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    static COUNT: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
    fn unique() -> u64 {
        COUNT.fetch_add(1, std::sync::atomic::Ordering::SeqCst)
    }

    #[test]
    fn validators_match_go() {
        assert_eq!(oci_architecture("x86_64").unwrap(), "amd64");
        assert_eq!(
            oci_architecture("aarch64").unwrap_err().message(),
            "expected x86_64"
        );
        assert!(is_digest(&"a".repeat(64)));
        assert!(!is_digest(&"A".repeat(64)));
        assert!(is_revision(&"b".repeat(40)));
        assert!(!is_revision("dirty"));
        require_native("x86_64").unwrap();
        assert!(require_native("other").is_err());
    }

    #[test]
    fn hash_file_refuses_symlinks() {
        let dir = scratch();
        let real = dir.join("real");
        std::fs::write(&real, b"data").unwrap();
        assert_eq!(hash_file(&real).unwrap(), crate::sha256_hex(b"data"));
        let link = dir.join("link");
        std::os::unix::fs::symlink(&real, &link).unwrap();
        assert_eq!(
            hash_file(&link).unwrap_err().message(),
            "regular non-symlink file required"
        );
        let root = Root::open(&dir).unwrap();
        assert_eq!(hash_at(&root, "real").unwrap(), crate::sha256_hex(b"data"));
        assert_eq!(
            hash_at(&root, "link").unwrap_err().message(),
            "regular non-symlink file required"
        );
        assert!(hash_at(&root, "../escape").is_err());
    }

    #[test]
    fn fresh_and_private_directories() {
        let dir = scratch();
        assert!(fresh_directory(Path::new("relative")).is_err());
        let fresh = dir.join("fresh");
        fresh_directory(&fresh).unwrap();
        assert_eq!(
            std::fs::metadata(&fresh).unwrap().permissions().mode() & 0o777,
            0o700
        );
        assert!(fresh_directory(&fresh).is_err());
        let parent_link = dir.join("plink");
        std::os::unix::fs::symlink(&dir, &parent_link).unwrap();
        assert_eq!(
            fresh_directory(&parent_link.join("x"))
                .unwrap_err()
                .message(),
            "symlinked parent refused"
        );
        let out = fresh.join("out");
        private_destination(&out).unwrap();
        std::fs::write(&out, b"x").unwrap();
        assert_eq!(
            private_destination(&out).unwrap_err().message(),
            "output already exists or cannot be inspected"
        );
        let loose = dir.join("loose");
        std::fs::create_dir(&loose).unwrap();
        chmod(&loose, 0o755).unwrap();
        assert_eq!(
            private_destination(&loose.join("o")).unwrap_err().message(),
            "real private output parent required"
        );
    }

    #[test]
    fn write_new_is_exclusive() {
        let dir = scratch();
        let path = dir.join("new");
        write_new(&path, b"bytes", 0o600).unwrap();
        assert_eq!(
            std::fs::metadata(&path).unwrap().permissions().mode() & 0o777,
            0o600
        );
        assert!(write_new(&path, b"bytes", 0o600).is_err());
    }

    #[test]
    fn read_json_strict_round_trip() {
        let dir = scratch();
        let path = dir.join("file.json");
        let file = File {
            sha256: "a".repeat(64),
            mode: 0o644,
            link: String::new(),
            directory: true,
        };
        let body = file.marshal() + "\n";
        std::fs::write(&path, &body).unwrap();
        let root = Root::open(&dir).unwrap();
        let (back, digest) = read_json_at(&root, "file.json", "build.File", File::decode).unwrap();
        assert_eq!(back, file);
        assert_eq!(digest, crate::sha256_hex(body.as_bytes()));
        // Unknown fields and trailing data are refused.
        std::fs::write(&path, "{\"Mode\":420,\"bogus\":1}").unwrap();
        assert!(read_json(&path, "build.File", File::decode).is_err());
        std::fs::write(&path, "{\"Mode\":420} {}").unwrap();
        assert!(read_json(&path, "build.File", File::decode).is_err());
    }

    #[test]
    fn root_rejects_symlink_escape() {
        let dir = scratch();
        let outside = dir.join("outside");
        std::fs::write(&outside, b"sensitive").unwrap();
        let linkroot = dir.join("linkroot");
        std::os::unix::fs::symlink(&dir, &linkroot).unwrap();
        assert!(Root::open(&linkroot).is_err());
        let sub = dir.join("sub");
        std::fs::create_dir(&sub).unwrap();
        std::os::unix::fs::symlink(&dir, sub.join("evil")).unwrap();
        let root = Root::open(&sub).unwrap();
        assert!(root.lstat("evil/outside").is_err());
    }
}
