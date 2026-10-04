//! Private filesystem inputs: exact ports of the `internal/release/build`
//! file helpers plus an fd-confined directory handle mirroring Go's `os.Root`.
//!
//! Constructed messages match the Go owner byte for byte. Raw OS errors keep
//! their [`std::io::ErrorKind`] but render with Rust's `(os error N)` suffix;
//! tests assert kinds and constructed messages, never raw OS text.

use std::ffi::{CStr, CString};
use std::fs::{File, OpenOptions};
use std::io::{Read, Write};
use std::os::fd::{AsRawFd, FromRawFd, OwnedFd};
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::Path;

use crate::error::Error;
use crate::sha256::Sha256;

/// Maximum private-file input: `PrivateFile` bound.
pub const PRIVATE_FILE_LIMIT: u64 = 1 << 20;
/// Maximum decoded JSON input: `ReadJSONAt` bound.
pub const JSON_LIMIT: u64 = 4 << 20;

/// Lexical path clean, matching Go's `filepath.Clean`: duplicate separators
/// collapse, `.` drops out, `..` pops the previous component without touching
/// the filesystem, and a trailing separator is removed except at the root.
pub fn lexical_clean(path: &str) -> String {
    let absolute = path.starts_with('/');
    let mut parts: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if parts.last().is_some_and(|last| *last != "..") {
                    parts.pop();
                } else if !absolute {
                    parts.push("..");
                }
            }
            _ => parts.push(part),
        }
    }
    let mut out = parts.join("/");
    if absolute {
        out.insert(0, '/');
    }
    if out.is_empty() {
        out.push('.');
    }
    out
}

fn c_string(path: &Path) -> Result<CString, Error> {
    CString::new(path.as_os_str().as_bytes())
        .map_err(|_| Error::from(std::io::Error::new(std::io::ErrorKind::InvalidInput, "path contains NUL")))
}

/// File identity plus the mode bits callers gate on.
#[derive(Debug, Clone, Copy)]
pub struct FileAttr {
    /// Regular file (never a symlink: always read with `AT_SYMLINK_NOFOLLOW`).
    pub is_regular: bool,
    /// Directory.
    pub is_dir: bool,
    /// Symlink.
    pub is_symlink: bool,
    /// Permission bits (`st_mode & 0o7777`).
    pub perm: u32,
    /// Size in bytes.
    pub size: u64,
    /// Owner UID/GID.
    pub uid: u32,
    /// Owner UID/GID.
    pub gid: u32,
    /// Device + inode for same-file comparison.
    pub dev: u64,
    /// Device + inode for same-file comparison.
    pub ino: u64,
}

impl FileAttr {
    fn from_stat(st: &libc::stat) -> FileAttr {
        let mode = st.st_mode;
        FileAttr {
            is_regular: mode & libc::S_IFMT == libc::S_IFREG,
            is_dir: mode & libc::S_IFMT == libc::S_IFDIR,
            is_symlink: mode & libc::S_IFMT == libc::S_IFLNK,
            perm: mode & 0o7777,
            size: st.st_size.max(0) as u64,
            uid: st.st_uid,
            gid: st.st_gid,
            dev: st.st_dev as u64,
            ino: st.st_ino as u64,
        }
    }
}

/// True when both attributes describe the same file, like `os.SameFile`.
pub fn same_file(a: FileAttr, b: FileAttr) -> bool {
    a.dev == b.dev && a.ino == b.ino
}

/// An open directory handle. All operations resolve beneath it, mirroring
/// Go's `os.Root` (`openat2` with `RESOLVE_BENEATH`): intermediate symlinks
/// and escapes fail instead of resolving outside.
#[derive(Debug)]
pub struct OwnedDir {
    fd: OwnedFd,
    /// Original path, for display and disjointness checks only. All I/O uses
    /// the fd, so a renamed root keeps working like the Go owner.
    path: String,
}

impl OwnedDir {
    /// Open an existing directory, like `os.OpenRoot`.
    pub fn open(path: &str) -> Result<OwnedDir, Error> {
        let raw = c_string(Path::new(path))?;
        let fd = unsafe { libc::open(raw.as_ptr(), libc::O_RDONLY | libc::O_DIRECTORY | libc::O_CLOEXEC) };
        if fd < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        Ok(OwnedDir {
            // SAFETY: `open` returned a fresh owned fd.
            fd: unsafe { OwnedFd::from_raw_fd(fd) },
            path: path.to_string(),
        })
    }

    /// Original path this handle was opened from.
    pub fn path(&self) -> &str {
        &self.path
    }

    fn single_component(name: &str) -> Result<CString, Error> {
        if name.is_empty() || name == "." || name == ".." || name.contains('/') {
            return Err(Error::msg("invalid evidence name"));
        }
        c_string(Path::new(name))
    }

    /// Open one child directory, refusing symlinks (`O_NOFOLLOW`).
    fn open_child_dir(fd: &OwnedFd, name: &CStr) -> Result<OwnedFd, Error> {
        let child = unsafe {
            libc::openat(
                fd.as_raw_fd(),
                name.as_ptr(),
                libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            )
        };
        if child < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        // SAFETY: `openat` returned a fresh owned fd.
        Ok(unsafe { OwnedFd::from_raw_fd(child) })
    }

    /// Traverse `parent` (possibly empty) component by component beneath this
    /// root, refusing symlinks and `..`, like Go's beneath-resolving root.
    fn traverse(&self, parent: &str) -> Result<OwnedFd, Error> {
        let mut fd = self.duplicate()?;
        if parent.is_empty() {
            return Ok(fd);
        }
        for component in parent.split('/') {
            let raw = OwnedDir::single_component(component)?;
            fd = OwnedDir::open_child_dir(&fd, &raw)?;
        }
        Ok(fd)
    }

    fn split_parent(name: &str) -> Result<(&str, CString), Error> {
        let (parent, base) = match name.rsplit_once('/') {
            Some((parent, base)) => (parent, base),
            None => ("", name),
        };
        Ok((parent, OwnedDir::single_component(base)?))
    }

    /// `lstat` a path beneath this root without following symlinks.
    pub fn lstat_at(&self, name: &str) -> Result<FileAttr, Error> {
        let (parent, base) = OwnedDir::split_parent(name)?;
        let dir = self.traverse(parent)?;
        let mut st: libc::stat = unsafe { std::mem::zeroed() };
        let rc = unsafe { libc::fstatat(dir.as_raw_fd(), base.as_ptr(), &mut st, libc::AT_SYMLINK_NOFOLLOW) };
        if rc != 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        Ok(FileAttr::from_stat(&st))
    }

    /// Open a file beneath this root for reading, refusing escapes. The
    /// final component is opened `O_NOFOLLOW`; callers re-stat, like the Go
    /// owner's unchanged-regular check.
    pub fn open_file_at(&self, name: &str) -> Result<File, Error> {
        let (parent, base) = OwnedDir::split_parent(name)?;
        let dir = self.traverse(parent)?;
        let fd = unsafe {
            libc::openat(
                dir.as_raw_fd(),
                base.as_ptr(),
                libc::O_RDONLY | libc::O_NONBLOCK | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            )
        };
        if fd < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        // SAFETY: `openat` returned a fresh owned fd.
        Ok(File::from(unsafe { OwnedFd::from_raw_fd(fd) }))
    }

    /// Open a subdirectory beneath this root, refusing escapes.
    pub fn sub_dir(&self, name: &str) -> Result<OwnedDir, Error> {
        let mut fd = self.duplicate()?;
        for component in name.split('/') {
            let raw = OwnedDir::single_component(component)?;
            fd = OwnedDir::open_child_dir(&fd, &raw)?;
        }
        Ok(OwnedDir {
            fd,
            path: format!("{}/{}", self.path, name),
        })
    }

    /// `mkdirat` beneath this root with exact mode bits, traversing
    /// intermediate components like Go's `Root.Mkdir` on nested names.
    pub fn mkdir_at(&self, name: &str, mode: u32) -> Result<(), Error> {
        let (parent, base) = OwnedDir::split_parent(name)?;
        let dir = self.traverse(parent)?;
        let rc = unsafe { libc::mkdirat(dir.as_raw_fd(), base.as_ptr(), mode) };
        if rc != 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        Ok(())
    }

    /// Exclusive `O_CREAT|O_EXCL` file creation beneath this root.
    pub fn create_new_at(&self, name: &str, mode: u32) -> Result<File, Error> {
        let (parent, base) = OwnedDir::split_parent(name)?;
        let dir = self.traverse(parent)?;
        let fd = unsafe {
            libc::openat(
                dir.as_raw_fd(),
                base.as_ptr(),
                libc::O_WRONLY | libc::O_CREAT | libc::O_EXCL | libc::O_CLOEXEC,
                mode,
            )
        };
        if fd < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        // SAFETY: `openat` returned a fresh owned fd.
        Ok(File::from(unsafe { OwnedFd::from_raw_fd(fd) }))
    }

    /// Exclusive hard link beneath this root, like `Root.Link`.
    pub fn link_at(&self, old: &str, new: &str) -> Result<(), Error> {
        let (old_parent, old_base) = OwnedDir::split_parent(old)?;
        let (new_parent, new_base) = OwnedDir::split_parent(new)?;
        let old_dir = self.traverse(old_parent)?;
        let new_dir = self.traverse(new_parent)?;
        let rc = unsafe {
            libc::linkat(
                old_dir.as_raw_fd(),
                old_base.as_ptr(),
                new_dir.as_raw_fd(),
                new_base.as_ptr(),
                0,
            )
        };
        if rc != 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        Ok(())
    }

    fn duplicate(&self) -> Result<OwnedFd, Error> {
        let fd = unsafe { libc::fcntl(self.fd.as_raw_fd(), libc::F_DUPFD_CLOEXEC, 0) };
        if fd < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        // SAFETY: `fcntl` returned a fresh owned fd.
        Ok(unsafe { OwnedFd::from_raw_fd(fd) })
    }

    /// Walk entries beneath this root, yielding root-relative paths with a
    /// regularity flag. Directories recurse; callers report their own error
    /// for non-regular entries like the Go owner does.
    pub fn walk_files(&self, visit: &mut dyn FnMut(&str, bool) -> Result<(), Error>) -> Result<(), Error> {
        self.walk_recursion("", visit)
    }

    fn walk_recursion(&self, prefix: &str, visit: &mut dyn FnMut(&str, bool) -> Result<(), Error>) -> Result<(), Error> {
        let dir = if prefix.is_empty() {
            OwnedDir {
                fd: self.duplicate()?,
                path: self.path.clone(),
            }
        } else {
            self.sub_dir(prefix)?
        };
        // `fdopendir` takes ownership, so hand it a duplicate.
        let raw = unsafe { libc::fcntl(dir.fd.as_raw_fd(), libc::F_DUPFD_CLOEXEC, 0) };
        if raw < 0 {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        let stream = unsafe { libc::fdopendir(raw) };
        if stream.is_null() {
            unsafe { libc::close(raw) };
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        let mut result = Ok(());
        loop {
            unsafe { *libc::__errno_location() = 0 };
            let entry = unsafe { libc::readdir(stream) };
            if entry.is_null() {
                let errno = unsafe { *libc::__errno_location() };
                if errno != 0 {
                    result = Err(Error::from(std::io::Error::from_raw_os_error(errno)));
                }
                break;
            }
            let name = unsafe { CStr::from_ptr((*entry).d_name.as_ptr()) }.to_string_lossy();
            if name == "." || name == ".." {
                continue;
            }
            let rel = if prefix.is_empty() {
                name.to_string()
            } else {
                format!("{prefix}/{name}")
            };
            let attr = dir.lstat_at(&name)?;
            if attr.is_dir {
                if let Err(e) = self.walk_recursion(&rel, visit) {
                    result = Err(e);
                    break;
                }
            } else if let Err(e) = visit(&rel, attr.is_regular) {
                result = Err(e);
                break;
            }
        }
        unsafe { libc::closedir(stream) };
        result
    }
}

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
        return Err(Error::msg(format!("restricted regular input required: {base}")));
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

/// Create a fresh private directory whose parent has no symlinks.
/// Mirrors `FreshDirectory`.
pub fn fresh_directory(path: &str) -> Result<(), Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute new directory required"));
    }
    let parent = Path::new(path).parent().unwrap_or(Path::new("/"));
    let resolved = std::fs::canonicalize(parent)?;
    if resolved.to_string_lossy() != lexical_clean(&parent.to_string_lossy()) {
        return Err(Error::msg("symlinked parent refused"));
    }
    let raw = c_string(Path::new(path))?;
    let rc = unsafe { libc::mkdir(raw.as_ptr(), 0o700) };
    if rc != 0 {
        return Err(Error::from(std::io::Error::last_os_error()));
    }
    Ok(())
}

/// Admit a new private output path: absolute, unoccupied, below a real
/// private directory. Mirrors `PrivateDestination`.
pub fn private_destination(path: &str) -> Result<(), Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute private output required"));
    }
    let parent = Path::new(path).parent().unwrap_or(Path::new("/"));
    let parent_text = parent.to_string_lossy();
    // Like Go, resolution and stat failures propagate raw; only the
    // shape violations below carry the fixed message.
    let resolved = std::fs::canonicalize(parent)?;
    let meta = std::fs::metadata(parent)?;
    if resolved.to_string_lossy() != parent_text || meta.mode() & 0o077 != 0 {
        return Err(Error::msg("real private output parent required"));
    }
    match std::fs::symlink_metadata(path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        _ => Err(Error::msg("output already exists or cannot be inspected")),
    }
}

/// Private scratch directory, removed on drop.
#[derive(Debug)]
pub struct TempDir {
    path: std::path::PathBuf,
}

impl TempDir {
    /// Create a fresh private scratch directory.
    pub fn new(prefix: &str) -> Result<TempDir, Error> {
        static NEXT: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
        let mut path = std::env::temp_dir();
        path.push(format!(
            "{prefix}-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, std::sync::atomic::Ordering::SeqCst)
        ));
        fresh_directory(&path.to_string_lossy())?;
        Ok(TempDir { path })
    }

    /// Join a name below the directory.
    pub fn join(&self, name: &str) -> std::path::PathBuf {
        self.path.join(name)
    }

    /// Directory path.
    pub fn path(&self) -> &std::path::Path {
        &self.path
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.path);
    }
}

/// Exclusive file creation with exact mode bits. Mirrors `WriteNew`.
pub fn write_new(path: &str, data: &[u8], mode: u32) -> Result<(), Error> {
    let mut file = OpenOptions::new().write(true).create_new(true).mode(mode).open(path)?;
    file.write_all(data)?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;

    fn temp_dir(prefix: &str) -> std::path::PathBuf {
        let mut dir = std::env::temp_dir();
        dir.push(format!("{prefix}-{}-{}", std::process::id(), rand_suffix()));
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    fn rand_suffix() -> u64 {
        use std::collections::hash_map::DefaultHasher;
        use std::hash::{Hash, Hasher};
        let mut hasher = DefaultHasher::new();
        std::time::SystemTime::now().hash(&mut hasher);
        std::thread::current().id().hash(&mut hasher);
        hasher.finish()
    }

    #[test]
    fn lexical_clean_matches_go() {
        for (input, want) in [
            ("", "."),
            (".", "."),
            ("a/../b", "b"),
            ("../escape", "../escape"),
            ("/a//b/", "/a/b"),
            ("/..", "/"),
            ("/a/./b", "/a/b"),
            ("a/b", "a/b"),
        ] {
            assert_eq!(lexical_clean(input), want, "clean {input:?}");
        }
    }

    #[test]
    fn private_file_matrix() {
        let dir = temp_dir("soda-files-private");
        let good = dir.join("good");
        std::fs::write(&good, b"secret").unwrap();
        std::fs::set_permissions(&good, std::fs::Permissions::from_mode(0o600)).unwrap();
        assert_eq!(private_file(good.to_str().unwrap()).unwrap(), b"secret");

        let open = dir.join("open");
        std::fs::write(&open, b"secret").unwrap();
        std::fs::set_permissions(&open, std::fs::Permissions::from_mode(0o644)).unwrap();
        let err = private_file(open.to_str().unwrap()).unwrap_err();
        assert_eq!(err.to_string(), "restricted regular input required: open");

        let link = dir.join("link");
        std::os::unix::fs::symlink(&good, &link).unwrap();
        assert!(private_file(link.to_str().unwrap()).is_err());
        assert!(private_file(dir.to_str().unwrap()).is_err());
        assert_eq!(
            private_file("relative/path").unwrap_err().to_string(),
            "absolute private input required"
        );
        assert!(private_file(dir.join("missing").to_str().unwrap()).is_err());

        let big = dir.join("big");
        std::fs::write(&big, vec![0u8; (PRIVATE_FILE_LIMIT + 1) as usize]).unwrap();
        std::fs::set_permissions(&big, std::fs::Permissions::from_mode(0o600)).unwrap();
        assert!(private_file(big.to_str().unwrap()).is_err());
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn hash_file_vectors() {
        let dir = temp_dir("soda-files-hash");
        let path = dir.join("data");
        std::fs::write(&path, b"abc").unwrap();
        assert_eq!(
            hash_file(path.to_str().unwrap()).unwrap(),
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
        let link = dir.join("link");
        std::os::unix::fs::symlink(&path, &link).unwrap();
        assert_eq!(
            hash_file(link.to_str().unwrap()).unwrap_err().to_string(),
            "regular non-symlink file required"
        );
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn owned_dir_confines_and_links() {
        let dir = temp_dir("soda-files-root");
        let root = OwnedDir::open(dir.to_str().unwrap()).unwrap();
        root.mkdir_at("sub", 0o700).unwrap();
        assert!(root.mkdir_at("sub", 0o700).unwrap_err().io_kind().is_some());
        {
            let mut file = root.create_new_at("sub/file", 0o600).unwrap();
            file.write_all(b"data").unwrap();
        }
        assert_eq!(hash_at(&root, "sub/file").unwrap(), crate::sha256::hex_lower(&crate::sha256::digest(b"data")));
        root.link_at("sub/file", "linked").unwrap();
        assert!(root.link_at("sub/file", "linked").is_err());
        assert!(root.create_new_at("sub/file", 0o600).unwrap_err().io_kind() == Some(std::io::ErrorKind::AlreadyExists));

        let outside = temp_dir("soda-files-outside");
        let escape = dir.join("escape");
        std::os::unix::fs::symlink(&outside, &escape).unwrap();
        assert!(root.open_file_at("escape/x").is_err());
        assert!(root.sub_dir("escape").is_err());
        assert!(root.lstat_at("escape").unwrap().is_symlink);

        let mut found = Vec::new();
        root.walk_files(&mut |rel: &str, regular: bool| {
            found.push((rel.to_string(), regular));
            Ok(())
        })
        .unwrap();
        found.sort();
        assert_eq!(
            found,
            vec![
                ("escape".to_string(), false),
                ("linked".to_string(), true),
                ("sub/file".to_string(), true),
            ]
        );
        std::fs::remove_dir_all(&dir).unwrap();
        std::fs::remove_dir_all(&outside).unwrap();
    }

    #[test]
    fn fresh_directory_and_destination_gates() {
        let dir = temp_dir("soda-files-fresh");
        let fresh = dir.join("fresh");
        fresh_directory(fresh.to_str().unwrap()).unwrap();
        assert_eq!(
            std::fs::metadata(&fresh).unwrap().permissions().mode() & 0o777,
            0o700
        );
        let err = fresh_directory(fresh.to_str().unwrap()).unwrap_err();
        assert_eq!(err.io_kind(), Some(std::io::ErrorKind::AlreadyExists));
        assert!(fresh_directory("relative").is_err());

        let link_parent = dir.join("linkparent");
        std::os::unix::fs::symlink(&dir, &link_parent).unwrap();
        let err = fresh_directory(link_parent.join("x").to_str().unwrap()).unwrap_err();
        assert_eq!(err.to_string(), "symlinked parent refused");

        let parent = dir.join("parent");
        std::fs::create_dir(&parent).unwrap();
        std::fs::set_permissions(&parent, std::fs::Permissions::from_mode(0o700)).unwrap();
        let out = parent.join("out.md");
        private_destination(out.to_str().unwrap()).unwrap();
        write_new(out.to_str().unwrap(), b"bytes", 0o600).unwrap();
        assert_eq!(
            private_destination(out.to_str().unwrap()).unwrap_err().to_string(),
            "output already exists or cannot be inspected"
        );
        std::fs::set_permissions(&parent, std::fs::Permissions::from_mode(0o755)).unwrap();
        assert_eq!(
            private_destination(parent.join("other").to_str().unwrap()).unwrap_err().to_string(),
            "real private output parent required"
        );
        std::fs::remove_dir_all(&dir).unwrap();
    }
}
