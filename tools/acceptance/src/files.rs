//! Private filesystem inputs: exact ports of the `internal/release/build`
//! file helpers plus an fd-confined directory handle mirroring Go's `os.Root`.
//!
//! Constructed messages match the Go owner byte for byte. Raw OS errors keep
//! their [`std::io::ErrorKind`] but render with Rust's `(os error N)` suffix;
//! tests assert kinds and constructed messages, never raw OS text.

use std::ffi::CString;
#[cfg(test)]
use std::io::Write;
use std::os::unix::ffi::OsStrExt;
use std::path::Path;

use crate::error::Error;

mod inputs;
mod owned_directory;
mod temporary;

pub use inputs::{hash_at, hash_file, private_file};
pub use owned_directory::OwnedDir;
pub use temporary::{fresh_directory, private_destination, write_new, TempDir};

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
    CString::new(path.as_os_str().as_bytes()).map_err(|_| {
        Error::from(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "path contains NUL",
        ))
    })
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
            dev: st.st_dev,
            ino: st.st_ino,
        }
    }
}

/// True when both attributes describe the same file, like `os.SameFile`.
pub fn same_file(a: FileAttr, b: FileAttr) -> bool {
    a.dev == b.dev && a.ino == b.ino
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
        assert_eq!(
            hash_at(&root, "sub/file").unwrap(),
            crate::sha256::hex_lower(&crate::sha256::digest(b"data"))
        );
        root.link_at("sub/file", "linked").unwrap();
        assert!(root.link_at("sub/file", "linked").is_err());
        assert!(
            root.create_new_at("sub/file", 0o600).unwrap_err().io_kind()
                == Some(std::io::ErrorKind::AlreadyExists)
        );

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
            private_destination(out.to_str().unwrap())
                .unwrap_err()
                .to_string(),
            "output already exists or cannot be inspected"
        );
        std::fs::set_permissions(&parent, std::fs::Permissions::from_mode(0o755)).unwrap();
        assert_eq!(
            private_destination(parent.join("other").to_str().unwrap())
                .unwrap_err()
                .to_string(),
            "real private output parent required"
        );
        std::fs::remove_dir_all(&dir).unwrap();
    }
}
