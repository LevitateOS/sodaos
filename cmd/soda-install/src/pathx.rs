//! Native filesystem path operations used by installer device discovery.

use std::path::{Path, PathBuf};

/// Join a trusted sysfs root with admitted relative components.
pub fn join(first: &str, rest: &[&str]) -> String {
    let mut path = PathBuf::from(first);
    for part in rest {
        path.push(part);
    }
    path.to_string_lossy().into_owned()
}

/// Parent directory using Rust's native path components.
pub fn dir(path: &str) -> String {
    let path = Path::new(path);
    path.parent()
        .filter(|parent| !parent.as_os_str().is_empty())
        .unwrap_or_else(|| {
            if path.is_absolute() {
                Path::new("/")
            } else {
                Path::new(".")
            }
        })
        .to_string_lossy()
        .into_owned()
}

/// Final normal component; empty becomes `"."`, root stays `/`.
pub fn base(path: &str) -> String {
    let path = Path::new(path);
    path.file_name()
        .map(|name| name.to_string_lossy().into_owned())
        .unwrap_or_else(|| if path.is_absolute() { "/" } else { "." }.to_string())
}

/// Resolve an existing path and its symlink aliases using the OS filesystem.
pub fn eval_symlinks(path: &str) -> Result<String, crate::errors::Error> {
    if path.is_empty() {
        return Err(crate::errors::path_error(
            "lstat",
            "",
            std::io::Error::from_raw_os_error(libc::ENOENT),
        ));
    }
    std::fs::canonicalize(path)
        .map(|resolved| resolved.to_string_lossy().into_owned())
        .map_err(|error| crate::errors::path_error("lstat", path, error))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn join_dir_base_vectors() {
        assert_eq!(join("/a", &["b", "c"]), "/a/b/c");
        assert_eq!(join("a", &["../b"]), "a/../b");
        assert_eq!(dir("/a/b/c"), "/a/b");
        assert_eq!(dir("/a"), "/");
        assert_eq!(dir("a"), ".");
        assert_eq!(dir("/"), "/");
        assert_eq!(dir(""), ".");
        assert_eq!(dir("/a/b/"), "/a");
        assert_eq!(dir("a/b/"), "a");
        assert_eq!(base("/a/b/c"), "c");
        assert_eq!(base("/a/b/"), "b");
        assert_eq!(base("/"), "/");
        assert_eq!(base(""), ".");
        assert_eq!(base("a"), "a");
    }

    #[test]
    fn symlinks_resolve() {
        let root =
            std::env::temp_dir().join(format!("soda-install-pathtest-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&root);
        std::fs::create_dir_all(root.join("a/b")).unwrap();
        std::fs::write(root.join("a/b/file"), b"x").unwrap();
        std::os::unix::fs::symlink(root.join("a"), root.join("link")).unwrap();
        std::os::unix::fs::symlink("b/file", root.join("a/rel")).unwrap();
        let got = eval_symlinks(root.join("link/b/file").to_str().unwrap()).unwrap();
        assert_eq!(got, root.join("a/b/file").to_str().unwrap());
        let got = eval_symlinks(root.join("a/rel").to_str().unwrap()).unwrap();
        assert_eq!(got, root.join("a/b/file").to_str().unwrap());
        assert!(eval_symlinks(root.join("missing").to_str().unwrap()).is_err());
        std::os::unix::fs::symlink("loop", root.join("loop")).unwrap();
        assert!(eval_symlinks(root.join("loop").to_str().unwrap()).is_err());
        let _ = std::fs::remove_dir_all(&root);
    }
}
