//! Build-time staging and render tools (ex `soda-stage-render`).
//!
//! Ports `scripts/stage.py`, `scripts/render-provisioning.py` and
//! `scripts/render-terminal-logo.py` to three binaries.
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
/// branding payload manifest the stage renderer consumes.
pub fn source_root() -> Result<PathBuf, String> {
    let cwd = std::env::current_dir().map_err(|e| e.to_string())?;
    let mut dir: &Path = &cwd;
    loop {
        if dir
            .join("frontend/forgejo/payload.json")
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
mod tests;
