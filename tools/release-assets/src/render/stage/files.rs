//! Confined file copying, tree normalization and platform gates.

use std::path::{Path, PathBuf};

use super::StageError;

use crate::render::chmod;

/// Native-Linux gate, like the script's `platform` check. The architecture
/// is the compile-time target: a native binary only runs where it was
/// built for, so this is the runtime check's honest equivalent.
pub fn check_platform(arch: &str) -> Result<(), String> {
    if std::env::consts::OS != "linux" || std::env::consts::ARCH != arch {
        return Err("matching native Linux required".to_string());
    }
    Ok(())
}

pub(crate) fn is_real_dir(path: &Path) -> bool {
    if !path.is_absolute() || !path.is_dir() {
        return false;
    }
    match std::fs::canonicalize(path) {
        Ok(real) => real == path,
        Err(_) => false,
    }
}

pub(crate) fn copy(
    root: &Path,
    src: &Path,
    dest: &str,
    mode: Option<u32>,
) -> Result<PathBuf, StageError> {
    let target = root.join(dest.trim_start_matches('/'));
    let fresh: Vec<PathBuf> = target
        .ancestors()
        .skip(1)
        .filter(|parent| !parent.exists())
        .map(|parent| parent.to_path_buf())
        .collect();
    if let Some(parent) = target.parent() {
        std::fs::create_dir_all(parent).map_err(|e| {
            StageError::failure(format!("cannot prepare {}: {e}", parent.display()))
        })?;
    }
    for parent in target.ancestors().skip(1) {
        if parent == root {
            break;
        }
        if fresh.iter().any(|known| known == parent) {
            chmod(parent, 0o755).map_err(StageError::failure)?;
        }
    }
    if std::fs::symlink_metadata(&target).is_ok() {
        return Err(StageError::refusal("occupied staging file refused"));
    }
    std::fs::copy(src, &target)
        .map_err(|e| StageError::failure(format!("cannot copy {}: {e}", src.display())))?;
    if let Some(mode) = mode {
        chmod(&target, mode).map_err(StageError::failure)?;
    }
    Ok(target)
}

/// Like `shutil.copytree` without `dirs_exist_ok`: the destination must not
/// exist, symlinks are materialized, and directory modes are left for the
/// caller's normalization pass.
pub(crate) fn copy_tree(src: &Path, dst: &Path) -> Result<(), StageError> {
    if std::fs::symlink_metadata(dst).is_ok() {
        return Err(StageError::failure(format!(
            "cannot copy tree to occupied {}",
            dst.display()
        )));
    }
    std::fs::create_dir_all(dst)
        .map_err(|e| StageError::failure(format!("cannot prepare {}: {e}", dst.display())))?;
    let entries = std::fs::read_dir(src)
        .map_err(|e| StageError::failure(format!("cannot list {}: {e}", src.display())))?;
    for entry in entries {
        let entry = entry.map_err(|e| StageError::failure(e.to_string()))?;
        let from = entry.path();
        let to = dst.join(entry.file_name());
        let file_type = entry
            .file_type()
            .map_err(|e| StageError::failure(e.to_string()))?;
        if file_type.is_dir() || (file_type.is_symlink() && from.is_dir()) {
            copy_tree(&from, &to)?;
        } else {
            std::fs::copy(&from, &to)
                .map_err(|e| StageError::failure(format!("cannot copy {}: {e}", from.display())))?;
        }
    }
    Ok(())
}

/// Like the script's `rglob` mode passes: directories become 0755, files
/// 0644. Symlinked directories are not descended into; setting a symlink's
/// own mode follows the link, like `os.chmod`.
pub(crate) fn normalize_tree(root: &Path) -> Result<(), StageError> {
    let entries = std::fs::read_dir(root)
        .map_err(|e| StageError::failure(format!("cannot list {}: {e}", root.display())))?;
    for entry in entries {
        let entry = entry.map_err(|e| StageError::failure(e.to_string()))?;
        let path = entry.path();
        let file_type = entry
            .file_type()
            .map_err(|e| StageError::failure(e.to_string()))?;
        if file_type.is_dir() {
            chmod(&path, 0o755).map_err(StageError::failure)?;
            normalize_tree(&path)?;
        } else {
            chmod(&path, 0o644).map_err(StageError::failure)?;
        }
    }
    Ok(())
}

/// Like bare `Path.mkdir()`: the parent must exist and the leaf must be
/// fresh.
pub(crate) fn mkdir_leaf(path: &Path) -> Result<(), StageError> {
    std::fs::create_dir(path)
        .map_err(|e| StageError::failure(format!("cannot create {}: {e}", path.display())))
}

/// Like `Path.mkdir(parents=True)` without `exist_ok`: missing ancestors
/// are created, but the leaf itself must be fresh.
pub(crate) fn mkdir_fresh(path: &Path) -> Result<(), StageError> {
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent).map_err(|e| {
            StageError::failure(format!("cannot prepare {}: {e}", parent.display()))
        })?;
    }
    std::fs::create_dir(path)
        .map_err(|e| StageError::failure(format!("cannot create {}: {e}", path.display())))
}

pub(crate) fn read_text(path: &Path) -> Result<String, StageError> {
    std::fs::read_to_string(path)
        .map_err(|e| StageError::failure(format!("cannot read {}: {e}", path.display())))
}
