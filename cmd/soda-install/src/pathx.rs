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
