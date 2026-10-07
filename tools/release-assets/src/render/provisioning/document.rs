//! JSON bootstrap document construction.

use std::path::Path;

use serde_json::Value;

use super::private_files::read_text;
use super::{ProvError, ProvKind};

fn files_mut(config: &mut Value) -> Result<&mut Vec<Value>, ProvError> {
    let object = config
        .as_object_mut()
        .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap must be an object"))?;
    let storage = object
        .get_mut("storage")
        .ok_or_else(|| ProvError::new(ProvKind::Key, "bootstrap has no storage"))?;
    let storage = storage
        .as_object_mut()
        .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap storage must be an object"))?;
    let files = storage
        .get_mut("files")
        .ok_or_else(|| ProvError::new(ProvKind::Key, "bootstrap storage has no files"))?;
    files
        .as_array_mut()
        .ok_or_else(|| ProvError::new(ProvKind::Attribute, "bootstrap files have no append"))
}

pub(crate) fn push_file(config: &mut Value, entry: Value) -> Result<(), ProvError> {
    files_mut(config)?.push(entry);
    Ok(())
}

pub(crate) fn clear_files(config: &mut Value) -> Result<(), ProvError> {
    files_mut(config)?.clear();
    Ok(())
}

pub(crate) fn file_entry(path: &str, mode: u32, inline: &str) -> Value {
    serde_json::json!({
        "path": path,
        "mode": mode,
        "contents": {"inline": inline},
    })
}

/// One public bootstrap for private provisioning and installer-media
/// conversion: the base document plus the shared branding file.
pub(crate) fn public_config(source: &Path) -> Result<Value, ProvError> {
    let path = source.join("system/host/provisioning/base.json");
    let text = read_text(&path)?;
    let mut config: Value = serde_json::from_str(&text).map_err(|_| {
        ProvError::new(
            ProvKind::JsonDecode,
            format!("cannot parse {}", path.display()),
        )
    })?;
    let svg = read_text(&source.join("assets/branding/source/soda-symbol.svg"))?;
    push_file(
        &mut config,
        file_entry(
            "/var/usrlocal/share/icons/hicolor/scalable/apps/sodaos-icon.svg",
            0o644,
            &svg,
        ),
    )?;
    Ok(config)
}
