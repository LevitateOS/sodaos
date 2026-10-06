//! Payload-manifest and locked-terminal-asset resolution.

use std::collections::HashMap;
use std::path::{Path, PathBuf};

use soda_json::JsonValue;

use super::files::read_text;
use super::StageError;

pub(crate) fn payload_entries(source: &Path) -> Result<Vec<(String, String)>, StageError> {
    let path = source.join("assets/branding/forgejo/forgejo-payload.json");
    let text = read_text(&path)?;
    let value = JsonValue::parse(&text)
        .map_err(|_| StageError::failure(format!("cannot parse {}", path.display())))?;
    match value {
        JsonValue::Object(entries) => {
            let mut out = Vec::with_capacity(entries.len());
            for (dest, origin) in entries {
                match origin.as_str() {
                    Some(origin) => out.push((dest, origin.to_string())),
                    None => {
                        return Err(StageError::failure(format!(
                            "cannot parse {}",
                            path.display()
                        )))
                    }
                }
            }
            Ok(out)
        }
        _ => Err(StageError::failure(format!(
            "cannot parse {}",
            path.display()
        ))),
    }
}

pub(crate) fn locked_terminal_assets(source: &Path) -> Result<HashMap<String, String>, StageError> {
    let path = source.join("tools/release-assets/terminal-assets.lock.json");
    let text = read_text(&path)?;
    let value = JsonValue::parse(&text)
        .map_err(|_| StageError::failure(format!("cannot parse {}", path.display())))?;
    let mut locked = HashMap::new();
    let items = match &value {
        JsonValue::Array(items) => items,
        _ => {
            return Err(StageError::failure(format!(
                "cannot parse {}",
                path.display()
            )))
        }
    };
    for item in items {
        let files = match item.get("files") {
            Some(JsonValue::Array(files)) => files,
            _ => {
                return Err(StageError::failure(format!(
                    "cannot parse {}",
                    path.display()
                )))
            }
        };
        for asset in files {
            let (Some(file), Some(sha)) = (
                asset.get("file").and_then(|v| v.as_str()),
                asset.get("sha256").and_then(|v| v.as_str()),
            ) else {
                return Err(StageError::failure(format!(
                    "cannot parse {}",
                    path.display()
                )));
            };
            locked.insert(file.to_string(), sha.to_string());
        }
    }
    Ok(locked)
}

pub(crate) fn payload_source(source: &Path, build: &Path, origin: &str) -> PathBuf {
    match origin.strip_prefix("@build/") {
        Some(rest) => build.join(rest),
        None => source.join(origin),
    }
}
