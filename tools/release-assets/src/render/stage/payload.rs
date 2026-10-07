//! Payload-manifest and locked-terminal-asset resolution.

use std::collections::HashMap;
use std::fmt;
use std::path::{Path, PathBuf};

use serde::de::{MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;

use super::files::read_text;
use super::StageError;

pub(super) struct PayloadEntries(pub(super) Vec<(String, String)>);

impl<'de> Deserialize<'de> for PayloadEntries {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct EntriesVisitor;

        impl<'de> Visitor<'de> for EntriesVisitor {
            type Value = PayloadEntries;

            fn expecting(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
                formatter.write_str("a payload object whose values are strings")
            }

            fn visit_map<M>(self, mut map: M) -> Result<Self::Value, M::Error>
            where
                M: MapAccess<'de>,
            {
                let mut entries = Vec::with_capacity(map.size_hint().unwrap_or(0));
                while let Some((key, value)) = map.next_entry::<String, String>()? {
                    entries.push((key, value));
                }
                Ok(PayloadEntries(entries))
            }
        }

        deserializer.deserialize_map(EntriesVisitor)
    }
}

pub(crate) fn payload_entries(source: &Path) -> Result<Vec<(String, String)>, StageError> {
    let path = source.join("frontend/forgejo/payload.json");
    let text = read_text(&path)?;
    let value: PayloadEntries = serde_json::from_str(&text)
        .map_err(|_| StageError::failure(format!("cannot parse {}", path.display())))?;
    Ok(value.0)
}

pub(crate) fn locked_terminal_assets(source: &Path) -> Result<HashMap<String, String>, StageError> {
    let path = source.join("tools/release-assets/terminal-assets.lock.json");
    let text = read_text(&path)?;
    let root: Box<RawValue> = serde_json::from_str(&text)
        .map_err(|_| StageError::failure(format!("cannot parse {}", path.display())))?;
    if root.get().as_bytes()[0] != b'[' {
        return Err(StageError::failure(format!(
            "cannot parse {}",
            path.display()
        )));
    }
    let items: Vec<Box<RawValue>> = serde_json::from_str(root.get())
        .map_err(|_| StageError::failure(format!("cannot parse {}", path.display())))?;
    let mut locked = HashMap::new();
    for item in items {
        let item: LockedItem = serde_json::from_str(item.get())
            .map_err(|_| StageError::failure(format!("cannot parse {}", path.display())))?;
        let files = item
            .files
            .ok_or_else(|| StageError::failure(format!("cannot parse {}", path.display())))?;
        let files: Vec<Box<RawValue>> = serde_json::from_str(files.get())
            .map_err(|_| StageError::failure(format!("cannot parse {}", path.display())))?;
        for asset in files {
            let asset: LockedAsset = serde_json::from_str(asset.get())
                .map_err(|_| StageError::failure(format!("cannot parse {}", path.display())))?;
            let file = asset
                .file
                .and_then(|raw| serde_json::from_str::<String>(raw.get()).ok())
                .ok_or_else(|| StageError::failure(format!("cannot parse {}", path.display())))?;
            let sha = asset
                .sha256
                .and_then(|raw| serde_json::from_str::<String>(raw.get()).ok())
                .ok_or_else(|| StageError::failure(format!("cannot parse {}", path.display())))?;
            locked.insert(file, sha);
        }
    }
    Ok(locked)
}

pub(super) struct LockedItem {
    pub(super) files: Option<Box<RawValue>>,
}

impl<'de> Deserialize<'de> for LockedItem {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct ItemVisitor;
        impl<'de> Visitor<'de> for ItemVisitor {
            type Value = LockedItem;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a lock item")
            }
            fn visit_map<M>(self, mut map: M) -> Result<Self::Value, M::Error>
            where
                M: MapAccess<'de>,
            {
                let mut files = None;
                while let Some(key) = map.next_key::<String>()? {
                    if key == "files" {
                        files = Some(map.next_value()?);
                    } else {
                        let _: Box<RawValue> = map.next_value()?;
                    }
                }
                Ok(LockedItem { files })
            }
        }
        deserializer.deserialize_map(ItemVisitor)
    }
}

pub(super) struct LockedAsset {
    pub(super) file: Option<Box<RawValue>>,
    pub(super) sha256: Option<Box<RawValue>>,
}

impl<'de> Deserialize<'de> for LockedAsset {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct AssetVisitor;
        impl<'de> Visitor<'de> for AssetVisitor {
            type Value = LockedAsset;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a locked asset")
            }
            fn visit_map<M>(self, mut map: M) -> Result<Self::Value, M::Error>
            where
                M: MapAccess<'de>,
            {
                let mut file = None;
                let mut sha256 = None;
                while let Some(key) = map.next_key::<String>()? {
                    match key.as_str() {
                        "file" => file = Some(map.next_value()?),
                        "sha256" => sha256 = Some(map.next_value()?),
                        _ => {
                            let _: Box<RawValue> = map.next_value()?;
                        }
                    }
                }
                Ok(LockedAsset { file, sha256 })
            }
        }
        deserializer.deserialize_map(AssetVisitor)
    }
}

pub(crate) fn payload_source(source: &Path, build: &Path, origin: &str) -> PathBuf {
    match origin.strip_prefix("@build/") {
        Some(rest) => build.join(rest),
        None => source.join(origin),
    }
}
