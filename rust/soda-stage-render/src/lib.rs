//! Build-time staging and render tools (lane R, PR31 stage-render-rust).
//!
//! Ports `scripts/stage.py`, `scripts/render-provisioning.py` and
//! `scripts/render-terminal-logo.py` to one crate with three binaries.
//! Every success output (staged file bytes, modes, stdout lines, rendered
//! JSON documents, branding text) and every validation message matches the
//! owning script; only the argparse envelope (usage preface, `prog: error:`
//! prefix) carries the new binary name, and unpinned failure text (usage
//! errors, malformed-input tracebacks, transport errors) differs.
//!
//! The scripts locate the source tree from their own path; the binaries
//! walk up from the working directory instead (see [`source_root`]), so
//! the documented invocations run from the checkout root, while the
//! branding check also keeps working from `scripts/`.

pub mod provisioning;
pub mod stage;
pub mod terminal_logo;

use std::path::{Path, PathBuf};

/// Locate the checkout root: the nearest ancestor-or-self holding the
/// build payload manifest the stage renderer consumes. The manifest is
/// build-owned data, not one of the ported scripts, so it survives the
/// cutover and keeps identifying the tree.
pub fn source_root() -> Result<PathBuf, String> {
    let cwd = std::env::current_dir().map_err(|e| e.to_string())?;
    let mut dir: &Path = &cwd;
    loop {
        if dir
            .join("internal/release/build/forgejo-payload.json")
            .is_file()
        {
            return Ok(dir.to_path_buf());
        }
        match dir.parent() {
            Some(parent) => dir = parent,
            None => return Err(format!("cannot find checkout root above {}", cwd.display())),
        }
    }
}

pub(crate) fn chmod(path: &Path, mode: u32) -> Result<(), String> {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(path, std::fs::Permissions::from_mode(mode))
            .map_err(|e| e.to_string())
    }
    #[cfg(not(unix))]
    {
        let _ = (path, mode);
        Err("staging requires unix file modes".to_string())
    }
}

/// Tokenize one argv element: `--name=value` gives `("name", Some(value))`,
/// `--name` gives `("name", None)`. Only the double-dash long form is
/// accepted, like the argparse owners (single-dash and abbreviated flags
/// are rejected); returns `None` for anything else, including `--`.
pub fn split_flag(arg: &str) -> Option<(String, Option<String>)> {
    if arg == "--" || !arg.starts_with("--") {
        return None;
    }
    let body = &arg[2..];
    if body.is_empty() {
        return None;
    }
    match body.split_once('=') {
        Some((name, value)) => {
            if name.is_empty() {
                return None;
            }
            Some((name.to_string(), Some(value.to_string())))
        }
        None => Some((body.to_string(), None)),
    }
}

/// Lowercase-hex SHA-256, like `hashlib.sha256().hexdigest()`.
pub fn sha256_hex(data: &[u8]) -> String {
    use sha2::Digest;
    let mut hasher = sha2::Sha256::new();
    hasher.update(data);
    let mut out = String::new();
    for byte in hasher.finalize() {
        out.push_str(&format!("{byte:02x}"));
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn flags_match_argparse_long_forms() {
        assert_eq!(
            split_flag("--arch=x86_64"),
            Some(("arch".to_string(), Some("x86_64".to_string())))
        );
        assert_eq!(split_flag("--out"), Some(("out".to_string(), None)));
        assert_eq!(split_flag("--"), None);
        assert_eq!(split_flag("-arch"), None);
        assert_eq!(split_flag("positional"), None);
        assert_eq!(split_flag("--=x"), None);
        assert_eq!(split_flag("--"), None);
    }

    #[test]
    fn sha256_matches_hashlib() {
        assert_eq!(
            sha256_hex(b"abc"),
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
    }
}
