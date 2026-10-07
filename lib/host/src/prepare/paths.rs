// Path, ID-map and output-line helpers for preparation operations.
use crate::preparation::FACTORY_PREPARATIONS_DIR;
use std::path::{Component, Path, PathBuf};

/// Join the audited relative components under their first path.
pub fn path_join(parts: &[&str]) -> String {
    let mut path = PathBuf::new();
    for part in parts {
        if !part.is_empty() {
            path.push(part);
        }
    }
    path.to_string_lossy().into_owned()
}

/// `prepareIDMap`: every mapping keeps host root out of the container and
/// some mapping shifts container root onto a usable range.
pub fn prepare_id_map(values: &[String]) -> bool {
    if values.is_empty() {
        return false;
    }
    let mut shifted = false;
    for value in values {
        let parts: Vec<&str> = value.split(':').collect();
        if parts.len() != 3 {
            return false;
        }
        let container: u32 = match parts[0].parse::<u32>() {
            Ok(v) if v.to_string() == parts[0] => v,
            _ => return false,
        };
        let base: u32 = match parts[1].parse::<u32>() {
            Ok(v) if v.to_string() == parts[1] && v > 0 => v,
            _ => return false,
        };
        let size: u32 = match parts[2].parse::<u32>() {
            Ok(v) if v.to_string() == parts[2] => v,
            _ => return false,
        };
        if container == 0 && size >= 65536 && u64::from(base) + u64::from(size) <= 4294967295 {
            shifted = true;
        }
    }
    shifted
}

/// `preparationPaths`: fixed role/checkout/snapshot/bundle/home layout.
pub fn preparation_paths(role: &str, id: &str) -> (String, String, String, String) {
    let checkout = format!("/home/{role}/checkouts/{id}");
    let snapshot = format!("{}/{id}/snapshot", FACTORY_PREPARATIONS_DIR);
    let bundle = format!("{snapshot}/source.bundle");
    let home = format!("{checkout}/.soda-home");
    (checkout, snapshot, bundle, home)
}

/// `singleLine`: one trimmed line, bounded, with no inner newline.
pub fn single_line(out: &[u8], limit: usize) -> Option<String> {
    if out.is_empty() || out.len() > limit {
        return None;
    }
    let line = std::str::from_utf8(out).ok()?.trim().to_string();
    if line.is_empty() || line.contains('\n') {
        return None;
    }
    Some(line)
}

/// `validResolvedToolPath`: absolute, clean, under a fixed tool prefix.
pub fn valid_resolved_tool_path(value: &str) -> bool {
    if value.is_empty() || value.len() > 256 || !is_clean_absolute_path(value) {
        return false;
    }
    value.starts_with("/usr/bin/") || value.starts_with("/usr/local/bin/")
}

fn is_clean_absolute_path(value: &str) -> bool {
    if !Path::new(value).is_absolute() {
        return false;
    }
    let mut rebuilt = PathBuf::new();
    for component in Path::new(value).components() {
        match component {
            Component::RootDir => rebuilt.push("/"),
            Component::Normal(part) => rebuilt.push(part),
            Component::CurDir | Component::ParentDir | Component::Prefix(_) => return false,
        }
    }
    rebuilt.to_str() == Some(value)
}
