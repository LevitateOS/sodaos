//! The fixed CoreOS /home alias is trusted; application-controlled links are not.
use std::fs;
use std::os::unix::fs::MetadataExt;
use std::path::{Component, Path, PathBuf};

fn owned_directory(path: &Path, owner: u32) -> Result<(), String> {
    let meta = fs::symlink_metadata(path).map_err(|_| "application home is unavailable")?;
    if !meta.is_dir()
        || meta.file_type().is_symlink()
        || meta.uid() != owner
        || meta.mode() & 0o022 != 0
        || fs::canonicalize(path).map_err(|_| "application home is unavailable")? != path
    {
        return Err("application home must be an operator-owned canonical directory".into());
    }
    Ok(())
}

fn resolve_at(path: &Path, home: &Path, coreos_home: &Path, owner: u32) -> Result<PathBuf, String> {
    if !path.is_absolute()
        || path
            .components()
            .any(|part| !matches!(part, Component::RootDir | Component::Normal(_)))
    {
        return Err("application storage requires a lexical path under /home".into());
    }
    let relative = path
        .strip_prefix(home)
        .map_err(|_| "application storage must remain under /home")?;
    let meta = fs::symlink_metadata(home).map_err(|_| "application home is unavailable")?;
    let canonical_home = fs::canonicalize(home).map_err(|_| "application home is unavailable")?;
    if meta.file_type().is_symlink() {
        let link = fs::read_link(home).map_err(|_| "application home alias is unavailable")?;
        if meta.uid() != owner
            || canonical_home != coreos_home
            || (link != Path::new("var/home") && link != coreos_home)
        {
            return Err("application home has an unsupported alias".into());
        }
        owned_directory(
            coreos_home
                .parent()
                .ok_or("application home has no parent")?,
            owner,
        )?;
        owned_directory(coreos_home, owner)?;
    } else {
        owned_directory(home, owner)?;
        if canonical_home != home {
            return Err("application home has an unsupported alias".into());
        }
    }
    let expected = canonical_home.join(relative);
    if fs::canonicalize(path).map_err(|_| "application storage is unavailable")? != expected {
        return Err("application storage links below /home are unsupported".into());
    }
    Ok(expected)
}

pub(super) fn home_path(path: &Path) -> Result<PathBuf, String> {
    let home = Path::new("/home");
    let canonical = Path::new("/var/home");
    if let Ok(relative) = path.strip_prefix(canonical) {
        // Internal callers retain the resolved path. Admit that result only
        // when resolving its lexical alias proves the same trusted CoreOS tree.
        let resolved = resolve_at(&home.join(relative), home, canonical, 0)?;
        if resolved != path {
            return Err("application storage has an unsupported home alias".into());
        }
        return Ok(resolved);
    }
    resolve_at(path, home, canonical, 0)
}
