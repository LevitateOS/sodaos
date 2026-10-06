//! Single-layer member resolution: requested-path gates, whiteout and
//! ancestor rules, per-member hashing, and one-layer tar scans.

use super::archive::entry_raw_name;
use super::hex_digest;
use crate::{path_clean, Error};
use std::collections::{BTreeMap, HashMap};
use std::io::Read;

#[derive(Debug, Clone, Default)]
pub(super) struct LayerMember {
    pub hash: String,
    pub present: bool,
    pub blocked: bool,
}

/// Absolute clean member paths requested out of the verified rootfs.
pub(super) fn requested_oci_paths(paths: &[String]) -> Result<BTreeMap<String, String>, Error> {
    let mut wanted = BTreeMap::new();
    for requested in paths {
        let name = requested
            .strip_prefix('/')
            .ok_or_else(|| Error::msg("absolute clean OCI member paths required"))?;
        if name.is_empty() || path_clean(name) != name || name.contains(['\\', '\n', '\r', '\0']) {
            return Err(Error::msg("absolute clean OCI member paths required"));
        }
        if wanted.contains_key(name) {
            return Err(Error::msg("duplicate OCI member request"));
        }
        wanted.insert(name.to_string(), requested.clone());
    }
    if wanted.is_empty() {
        return Err(Error::msg("explicit OCI members required"));
    }
    Ok(wanted)
}

/// Admits real Linux layer entries. Backslashes stay permitted: base layers
/// ship systemd escaped unit names, and member requests reject backslashes,
/// so such entries can never match a request.
fn clean_layer_name(name: &str) -> Result<String, Error> {
    let name = name.strip_prefix("./").unwrap_or(name);
    let clean = path_clean(name);
    let segments: Vec<&str> = name.split('/').collect();
    if clean == "."
        || clean.starts_with('/')
        || segments.contains(&"..")
        || clean.contains(['\n', '\r', '\0'])
    {
        return Err(Error::msg("unsafe OCI layer path"));
    }
    Ok(clean)
}

fn whiteout_target(name: &str) -> Option<String> {
    let base = name.rsplit('/').next().unwrap_or(name);
    if !base.starts_with(".wh.") || base == ".wh..wh..opq" {
        return None;
    }
    let dir = name.rfind('/').map(|i| &name[..i]).unwrap_or("");
    let target = base.strip_prefix(".wh.").unwrap_or("");
    if dir.is_empty() {
        Some(target.to_string())
    } else {
        Some(format!("{dir}/{target}"))
    }
}

fn record_layer_deletion(
    found: &mut HashMap<String, LayerMember>,
    wanted: &BTreeMap<String, String>,
    removed: &str,
) {
    for name in wanted.keys() {
        if name != removed && !name.starts_with(&format!("{removed}/")) {
            continue;
        }
        if let Some(existing) = found.get(name) {
            if existing.present || existing.blocked {
                continue;
            }
        }
        found.insert(name.clone(), LayerMember::default());
    }
}

fn record_ancestor_replacement(
    found: &mut HashMap<String, LayerMember>,
    wanted: &BTreeMap<String, String>,
    name: &str,
    is_dir: bool,
) {
    if is_dir {
        return;
    }
    for requested in wanted.keys() {
        if requested.starts_with(&format!("{name}/")) {
            found.insert(
                requested.clone(),
                LayerMember {
                    blocked: true,
                    ..LayerMember::default()
                },
            );
        }
    }
}

fn record_opaque_directory(
    found: &mut HashMap<String, LayerMember>,
    wanted: &BTreeMap<String, String>,
    dir: &str,
) {
    for requested in wanted.keys() {
        if dir != "." && !requested.starts_with(&format!("{dir}/")) {
            continue;
        }
        if let Some(existing) = found.get(requested) {
            if existing.present || existing.blocked {
                continue;
            }
        }
        found.insert(requested.clone(), LayerMember::default());
    }
}

fn record_layer_entry(
    reader: &mut dyn Read,
    size: u64,
    name: &str,
    found: &mut HashMap<String, LayerMember>,
) -> Result<(), Error> {
    let mut hasher = sha2::Sha256::new();
    let mut remaining = size;
    let mut buf = [0u8; 32 << 10];
    let mut copied = 0u64;
    use sha2::Digest;
    while remaining > 0 {
        let want = (remaining as usize).min(buf.len());
        let n = reader
            .read(&mut buf[..want])
            .map_err(|_| Error::msg("OCI layer member size changed"))?;
        if n == 0 {
            return Err(Error::msg("OCI layer member size changed"));
        }
        hasher.update(&buf[..n]);
        copied += n as u64;
        remaining -= n as u64;
    }
    if copied != size {
        return Err(Error::msg("OCI layer member size changed"));
    }
    found.insert(
        name.to_string(),
        LayerMember {
            hash: hex_digest(&hasher.finalize()),
            present: true,
            blocked: false,
        },
    );
    Ok(())
}

pub(super) fn scan_oci_layer(
    reader: &mut dyn Read,
    wanted: &BTreeMap<String, String>,
) -> Result<HashMap<String, LayerMember>, Error> {
    let mut found: HashMap<String, LayerMember> = HashMap::new();
    let mut seen: HashMap<String, ()> = HashMap::new();
    let mut archive = tar::Archive::new(reader);
    let listed = archive.entries().map_err(|e| Error::msg(e.to_string()))?;
    for entry in listed {
        let mut entry = entry.map_err(|e| Error::msg(e.to_string()))?;
        let raw = entry_raw_name(&entry);
        // Base layers include the tar root directory. It can never match
        // a requested member, so skip it instead of refusing the archive.
        if raw == "." || raw == "./" {
            continue;
        }
        let name = clean_layer_name(&raw)?;
        if seen.contains_key(&name) {
            return Err(Error::msg("duplicate OCI layer entry"));
        }
        seen.insert(name.clone(), ());
        if seen.len() > 1_000_000 {
            return Err(Error::msg("too many OCI layer entries"));
        }
        let header = entry.header().clone();
        let entry_type = header.entry_type();
        let is_regular = entry_type.is_file();
        let is_dir = entry_type.is_dir();
        if let Some(removed) = whiteout_target(&name) {
            record_layer_deletion(&mut found, wanted, &removed);
            continue;
        }
        let base = name.rsplit('/').next().unwrap_or(&name);
        if base == ".wh..wh..opq" {
            let dir = if name.contains('/') {
                name.rfind('/').map(|i| &name[..i]).unwrap_or(".")
            } else {
                "."
            };
            record_opaque_directory(&mut found, wanted, dir);
            continue;
        }
        record_ancestor_replacement(&mut found, wanted, &name, is_dir);
        if !wanted.contains_key(&name) || found.get(&name).map(|m| m.blocked).unwrap_or(false) {
            continue;
        }
        if !is_regular {
            return Err(Error::msg("requested OCI member is non-regular"));
        }
        let size = header.size().map_err(|e| Error::msg(e.to_string()))?;
        record_layer_entry(&mut entry, size, &name, &mut found)?;
    }
    Ok(found)
}
