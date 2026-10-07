//! Pinned terminal distribution extraction (`scripts/fetch-terminal.py`).

use std::io::Read;
use std::os::unix::fs::OpenOptionsExt;
use std::path::{Path, PathBuf};
use std::time::Duration;

use serde::de::{MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;

pub const USER_AGENT: &str = "SodaOS-build";
const TIMEOUT: Duration = Duration::from_secs(30);
const ARCHIVE_LIMIT: u64 = 10_000_000;
const MEMBER_LIMIT: u64 = 2_000_000;
const DECOMPRESSED_LIMIT: u64 = 256 << 20;
const MEMBER_COUNT_LIMIT: usize = 100_000;

#[derive(Clone)]
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
            let asset = Asset {
                member: lock_string(&file, "member")?,
                file: lock_string(&file, "file")?,
                sha256: lock_string(&file, "sha256")?,
            };
            if asset.file.is_empty()
                || asset.file.contains(['/', '\0'])
                || matches!(asset.file.as_str(), "." | "..")
            {
                return Err("invalid terminal distribution member".to_string());
            }
            assets.push(asset);
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
        let file = match std::fs::OpenOptions::new()
            .read(true)
            .custom_flags(libc::O_NONBLOCK | libc::O_CLOEXEC)
            .open(&path)
        {
            Ok(file) => file,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(false),
            Err(error) => return Err(error.to_string()),
        };
        if !file.metadata().map_err(|e| e.to_string())?.is_file() {
            return Ok(false);
        }
        let mut data = Vec::new();
        file.take(MEMBER_LIMIT + 1)
            .read_to_end(&mut data)
            .map_err(|e| e.to_string())?;
        if data.len() as u64 > MEMBER_LIMIT {
            return Ok(false);
        }
        if crate::fetch::sha256_hex(&data) != asset.sha256 {
            return Ok(false);
        }
    }
    Ok(true)
}

struct BudgetReader<'a, R> {
    inner: &'a mut R,
    read: u64,
    limit: u64,
}

impl<R: Read> Read for BudgetReader<'_, R> {
    fn read(&mut self, output: &mut [u8]) -> std::io::Result<usize> {
        if output.is_empty() {
            return Ok(0);
        }
        if self.read == self.limit {
            let mut probe = [0; 1];
            return match self.inner.read(&mut probe)? {
                0 => Ok(0),
                _ => Err(std::io::Error::new(
                    std::io::ErrorKind::InvalidData,
                    "terminal archive decoded limit exceeded",
                )),
            };
        }
        let cap = (self.limit - self.read).min(output.len() as u64) as usize;
        let n = self.inner.read(&mut output[..cap])?;
        self.read += n as u64;
        Ok(n)
    }
}

fn read_members(
    body: &[u8],
    assets: &[Asset],
) -> Result<std::collections::HashMap<String, Vec<u8>>, String> {
    read_members_with_limit(body, assets, DECOMPRESSED_LIMIT)
}

fn read_members_with_limit(
    body: &[u8],
    assets: &[Asset],
    decoded_limit: u64,
) -> Result<std::collections::HashMap<String, Vec<u8>>, String> {
    let mut wanted = std::collections::HashMap::new();
    for asset in assets {
        if wanted.insert(asset.member.as_str(), ()).is_some() {
            return Err("duplicate terminal distribution member request".to_string());
        }
    }
    // Consume every gzip member and require only zero tar padding after the
    // first archive terminator; concatenated archives and junk are rejected.
    let mut decoder = flate2::read::MultiGzDecoder::new(body);
    let mut bounded = BudgetReader {
        inner: &mut decoder,
        read: 0,
        limit: decoded_limit,
    };
    let mut found = {
        let mut archive = tar::Archive::new(&mut bounded);
        let entries = archive
            .entries()
            .map_err(|e| format!("terminal archive is not readable: {e}"))?;
        let mut found = std::collections::HashMap::new();
        let mut count = 0;
        for entry in entries {
            let mut entry = entry.map_err(|e| format!("terminal archive is not readable: {e}"))?;
            count += 1;
            if count > MEMBER_COUNT_LIMIT {
                return Err("terminal archive has too many members".to_string());
            }
            let name = entry
                .path()
                .map_err(|e| format!("terminal archive is not readable: {e}"))?
                .to_str()
                .unwrap_or("")
                .to_string();
            if !wanted.contains_key(name.as_str()) {
                continue;
            }
            if found.contains_key(&name) {
                return Err("duplicate terminal distribution member".to_string());
            }
            let size = entry
                .header()
                .size()
                .map_err(|e| format!("terminal archive is not readable: {e}"))?;
            if !entry.header().entry_type().is_file() || size > MEMBER_LIMIT {
                return Err("invalid terminal distribution member".to_string());
            }
            let capacity = usize::try_from(size)
                .map_err(|_| "invalid terminal distribution member".to_string())?;
            let mut data = Vec::new();
            data.try_reserve_exact(capacity)
                .map_err(|_| "invalid terminal distribution member".to_string())?;
            entry
                .read_to_end(&mut data)
                .map_err(|e| format!("terminal archive is not readable: {e}"))?;
            if data.len() as u64 != size {
                return Err("terminal archive is not readable: member size changed".to_string());
            }
            found.insert(name, data);
        }
        found
    };
    let mut tail = [0u8; 65536];
    loop {
        let n = bounded
            .read(&mut tail)
            .map_err(|e| format!("terminal archive is not readable: {e}"))?;
        if n == 0 {
            break;
        }
        if tail[..n].iter().any(|byte| *byte != 0) {
            return Err("terminal archive has trailing data".to_string());
        }
    }
    for asset in assets {
        if !found.contains_key(&asset.member) {
            return Err("invalid terminal distribution member".to_string());
        }
    }
    Ok(found)
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
        let members = read_members(&body, &item.files)?;
        for asset in &item.files {
            let member = members
                .get(&asset.member)
                .ok_or_else(|| "invalid terminal distribution member".to_string())?;
            if asset.file.contains('/') {
                return Err("invalid terminal distribution member".to_string());
            }
            if crate::fetch::sha256_hex(member) != asset.sha256 {
                return Err("terminal asset integrity mismatch".to_string());
            }
        }
        for asset in &item.files {
            std::fs::write(out.join(&asset.file), &members[&asset.member])
                .map_err(|e| e.to_string())?;
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests;
