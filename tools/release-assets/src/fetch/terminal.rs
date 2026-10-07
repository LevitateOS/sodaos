//! Pinned terminal distribution extraction (`scripts/fetch-terminal.py`).

use std::io::Read;
use std::path::{Path, PathBuf};
use std::time::Duration;

use serde::de::{MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;

pub const USER_AGENT: &str = "SodaOS-build";
const TIMEOUT: Duration = Duration::from_secs(30);
const ARCHIVE_LIMIT: u64 = 10_000_000;
const MEMBER_LIMIT: u64 = 2_000_000;

struct Asset {
    member: String,
    file: String,
    sha256: String,
}

struct Item {
    url: String,
    integrity: String,
    files: Vec<Asset>,
}

struct RawObject(Vec<(String, Box<RawValue>)>);

impl<'de> Deserialize<'de> for RawObject {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct ObjectVisitor;
        impl<'de> Visitor<'de> for ObjectVisitor {
            type Value = RawObject;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("a JSON object")
            }
            fn visit_map<M>(self, mut map: M) -> Result<Self::Value, M::Error>
            where
                M: MapAccess<'de>,
            {
                let mut fields = Vec::with_capacity(map.size_hint().unwrap_or(0));
                while let Some((key, value)) = map.next_entry::<String, Box<RawValue>>()? {
                    fields.push((key, value));
                }
                Ok(RawObject(fields))
            }
        }
        deserializer.deserialize_map(ObjectVisitor)
    }
}

fn last<'a>(item: &'a RawObject, key: &str) -> Option<&'a RawValue> {
    item.0
        .iter()
        .rev()
        .find(|(name, _)| name == key)
        .map(|(_, value)| value.as_ref())
}

fn lock_string(item: &RawObject, key: &str) -> Result<String, String> {
    let raw = last(item, key).ok_or_else(|| "terminal lock entry is malformed".to_string())?;
    serde_json::from_str(raw.get()).map_err(|_| "terminal lock entry is malformed".to_string())
}

fn parse_lock(text: &str) -> Result<Vec<Item>, String> {
    let root: Box<RawValue> =
        serde_json::from_str(text).map_err(|_| "terminal lock is not valid JSON".to_string())?;
    if root.get().as_bytes()[0] != b'[' {
        return Err("terminal lock is not valid JSON".to_string());
    }
    let raw_items: Vec<Box<RawValue>> = serde_json::from_str(root.get())
        .map_err(|_| "terminal lock is not valid JSON".to_string())?;
    let mut items = Vec::with_capacity(raw_items.len());
    for raw in raw_items {
        let raw: RawObject = serde_json::from_str(raw.get())
            .map_err(|_| "terminal lock entry is malformed".to_string())?;
        let file_values =
            last(&raw, "files").ok_or_else(|| "terminal lock entry is malformed".to_string())?;
        let files: Vec<Box<RawValue>> = serde_json::from_str(file_values.get())
            .map_err(|_| "terminal lock entry is malformed".to_string())?;
        let mut assets = Vec::with_capacity(files.len());
        for file in files {
            let file: RawObject = serde_json::from_str(file.get())
                .map_err(|_| "terminal lock entry is malformed".to_string())?;
            assets.push(Asset {
                member: lock_string(&file, "member")?,
                file: lock_string(&file, "file")?,
                sha256: lock_string(&file, "sha256")?,
            });
        }
        items.push(Item {
            url: lock_string(&raw, "url")?,
            integrity: lock_string(&raw, "integrity")?,
            files: assets,
        });
    }
    Ok(items)
}

/// The owner reads `tools/release-assets/terminal-assets.lock.json` under the repo
/// root; every caller runs fetchers from the source root, so the working
/// directory resolves the same file.
pub fn default_lock() -> Result<PathBuf, String> {
    let cwd = std::env::current_dir().map_err(|e| e.to_string())?;
    Ok(cwd.join("tools/release-assets/terminal-assets.lock.json"))
}

/// Every listed file already staged with a matching digest. An item with
/// no files is trivially satisfied, like the owner's `all()`.
fn cached(out: &Path, files: &[Asset]) -> Result<bool, String> {
    for asset in files {
        let path = out.join(&asset.file);
        if !path.is_file() {
            return Ok(false);
        }
        let data = std::fs::read(&path).map_err(|e| e.to_string())?;
        if crate::fetch::sha256_hex(&data) != asset.sha256 {
            return Ok(false);
        }
    }
    Ok(true)
}

struct Member {
    name: String,
    is_file: bool,
    size: u64,
    data: Vec<u8>,
}

fn read_members(body: &[u8]) -> Result<Vec<Member>, String> {
    let gz = flate2::read::GzDecoder::new(body);
    let mut archive = tar::Archive::new(gz);
    let mut members = Vec::new();
    let entries = archive
        .entries()
        .map_err(|e| format!("terminal archive is not readable: {e}"))?;
    for entry in entries {
        let mut entry = entry.map_err(|e| format!("terminal archive is not readable: {e}"))?;
        let name = entry
            .path()
            .map_err(|e| format!("terminal archive is not readable: {e}"))?
            .to_str()
            .unwrap_or("")
            .to_string();
        let is_file = entry.header().entry_type().is_file();
        let size = entry.header().size().unwrap_or(u64::MAX);
        let mut data = Vec::new();
        entry
            .read_to_end(&mut data)
            .map_err(|e| format!("terminal archive is not readable: {e}"))?;
        members.push(Member {
            name,
            is_file,
            size,
            data,
        });
    }
    Ok(members)
}

fn extract(members: &[Member], asset: &Asset, out: &Path) -> Result<(), String> {
    // First exact-name match, like `getmember` (a missing member surfaces
    // as the invalid-member error rather than a bare key error).
    let member = members
        .iter()
        .find(|m| m.name == asset.member)
        .ok_or_else(|| "invalid terminal distribution member".to_string())?;
    if !member.is_file || member.size > MEMBER_LIMIT || asset.file.contains('/') {
        return Err("invalid terminal distribution member".to_string());
    }
    if crate::fetch::sha256_hex(&member.data) != asset.sha256 {
        return Err("terminal asset integrity mismatch".to_string());
    }
    std::fs::write(out.join(&asset.file), &member.data).map_err(|e| e.to_string())
}

pub fn fetch(lock_path: &Path, out: &Path) -> Result<(), String> {
    std::fs::create_dir_all(out).map_err(|e| e.to_string())?;
    let text = std::fs::read_to_string(lock_path)
        .map_err(|e| format!("cannot read terminal lock {}: {e}", lock_path.display()))?;
    let items = parse_lock(&text)?;
    for item in &items {
        if cached(out, &item.files)? {
            continue;
        }
        let mut response = crate::fetch::http_get(&item.url, USER_AGENT, TIMEOUT)
            .map_err(|e| format!("fetch {}: {e}", item.url))?;
        if !(200..300).contains(&response.status) {
            return Err(format!(
                "fetch {}: unexpected HTTP status {}",
                item.url, response.status
            ));
        }
        let body = crate::fetch::read_capped(&mut response.reader, ARCHIVE_LIMIT)?;
        if body.len() as u64 > ARCHIVE_LIMIT
            || format!("sha512-{}", crate::fetch::sha512_base64(&body)) != item.integrity
        {
            return Err("terminal archive integrity mismatch".to_string());
        }
        let members = read_members(&body)?;
        for asset in &item.files {
            extract(&members, asset, out)?;
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests;
