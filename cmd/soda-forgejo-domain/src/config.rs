use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;

use crate::system::{Paths, MARKER_NAME};

// --- configparser-compatible INI subset ---
//
// Matches CPython configparser defaults for the shapes app_data_path uses:
// case-sensitive sections, lowercased keys, `=`/`:` separators, full-line
// `#`/`;` comments, indented continuations joined with `\n`, strict
// duplicates, DEFAULT fallback, and BasicInterpolation on read.

struct IniFile {
    defaults: HashMap<String, String>,
    sections: HashMap<String, HashMap<String, String>>,
}

fn parse_ini(text: &str) -> Result<IniFile, String> {
    let mut defaults: HashMap<String, String> = HashMap::new();
    let mut sections: HashMap<String, HashMap<String, String>> = HashMap::new();
    let mut current: Option<String> = None;
    let mut current_key: Option<String> = None;
    for raw_line in text.split('\n') {
        let line = raw_line.strip_suffix('\r').unwrap_or(raw_line);
        if line.trim().is_empty() {
            // Blank lines join an open value; otherwise they are skipped.
            if let (Some(section), Some(key)) = (current.as_ref(), current_key.as_ref()) {
                let map = if section == "DEFAULT" {
                    &mut defaults
                } else {
                    sections.get_mut(section).unwrap()
                };
                if let Some(value) = map.get_mut(key) {
                    value.push('\n');
                }
            }
            continue;
        }
        let stripped = line.trim_start();
        if stripped.starts_with('#') || stripped.starts_with(';') {
            continue;
        }
        if line.starts_with([' ', '\t']) {
            if let (Some(section), Some(key)) = (current.as_ref(), current_key.as_ref()) {
                let map = if section == "DEFAULT" {
                    &mut defaults
                } else {
                    sections.get_mut(section).unwrap()
                };
                match map.get_mut(key) {
                    Some(value) => {
                        value.push('\n');
                        value.push_str(stripped);
                    }
                    None => return Err("continuation without an option".to_string()),
                }
                continue;
            }
            return Err("continuation without a section".to_string());
        }
        if stripped.starts_with('[') {
            let end = stripped
                .find(']')
                .ok_or_else(|| "missing section header".to_string())?;
            let header = stripped[1..end].to_string();
            if header.is_empty() {
                if current.is_none() {
                    return Err("missing section header".to_string());
                }
                return Err("missing delimiter".to_string());
            }
            if header == "DEFAULT" {
                current = Some(header);
            } else {
                if sections.contains_key(&header) {
                    return Err(format!("duplicate section {header}"));
                }
                sections.insert(header.clone(), HashMap::new());
                current = Some(header);
            }
            current_key = None;
            continue;
        }
        let section = current
            .clone()
            .ok_or_else(|| "missing section header".to_string())?;
        let sep = line
            .find(['=', ':'])
            .ok_or_else(|| "missing delimiter".to_string())?;
        let key = line[..sep].trim().to_ascii_lowercase();
        let value = line[sep + 1..].trim().to_string();
        let map = if section == "DEFAULT" {
            &mut defaults
        } else {
            sections.get_mut(&section).unwrap()
        };
        if map.contains_key(&key) {
            return Err(format!("duplicate option {key}"));
        }
        map.insert(key.clone(), value);
        current_key = Some(key);
    }
    Ok(IniFile { defaults, sections })
}

fn interpolate(
    value: &str,
    section: &HashMap<String, String>,
    defaults: &HashMap<String, String>,
) -> Result<String, String> {
    interpolate_depth(value, section, defaults, 0)
}

fn interpolate_depth(
    value: &str,
    section: &HashMap<String, String>,
    defaults: &HashMap<String, String>,
    depth: u32,
) -> Result<String, String> {
    if depth > 10 {
        return Err("interpolation depth exceeded".to_string());
    }
    let mut out = String::new();
    let mut rest = value;
    while let Some(at) = rest.find('%') {
        out.push_str(&rest[..at]);
        rest = &rest[at + 1..];
        if rest.starts_with('%') {
            out.push('%');
            rest = &rest[1..];
        } else if rest.starts_with('(') {
            let end = rest
                .find(')')
                .ok_or_else(|| "interpolation syntax error".to_string())?;
            let name = rest[1..end].to_ascii_lowercase();
            let after = &rest[end + 1..];
            if !after.starts_with('s') {
                return Err("interpolation syntax error".to_string());
            }
            let raw = section
                .get(&name)
                .or_else(|| defaults.get(&name))
                .ok_or_else(|| "interpolation option missing".to_string())?;
            out.push_str(&interpolate_depth(raw, section, defaults, depth + 1)?);
            rest = &after[1..];
        } else {
            return Err("interpolation syntax error".to_string());
        }
    }
    out.push_str(rest);
    Ok(out)
}

fn ini_get(ini: &IniFile, section: &str, key: &str) -> Option<String> {
    let key = key.to_ascii_lowercase();
    let map = ini.sections.get(section)?;
    let raw = map.get(&key).or_else(|| ini.defaults.get(&key))?;
    Some(raw.clone())
}

/// Resolve the deployment's Forgejo AppDataPath, container-side.
///
/// Reads [server] APP_DATA_PATH from the deployment app.ini with the
/// /etc/soda/forgejo.env FORGEJO__server__APP_DATA_PATH override winning,
/// matching Forgejo's own precedence. Refuses to guess: an absent or
/// relative value is an explicit error, never a default.
pub(crate) fn app_data_path(paths: &Paths) -> Result<String, String> {
    let mut value: Option<String> = None;
    if paths.env_file.exists() {
        let text = fs::read_to_string(&paths.env_file).map_err(|e| e.to_string())?;
        for line in text.split('\n') {
            let line = line.strip_suffix('\r').unwrap_or(line).trim();
            if line.starts_with("FORGEJO__server__APP_DATA_PATH=") {
                value = Some(
                    line.split_once('=')
                        .map(|(_, v)| v)
                        .unwrap_or("")
                        .trim()
                        .trim_matches(|c| c == '\'' || c == '"')
                        .to_string(),
                );
            }
        }
    }
    if value.is_none() {
        if !paths.app_ini.is_file() {
            return Err(format!(
                "deployment app.ini not found at {}",
                paths.app_ini.display()
            ));
        }
        // configparser.read swallows OS errors (missing file reads as
        // empty); only decode and syntax failures surface.
        let raw = fs::read(&paths.app_ini).unwrap_or_default();
        let text = String::from_utf8(raw).map_err(|e| {
            format!(
                "cannot parse deployment app.ini at {}: {e}",
                paths.app_ini.display()
            )
        })?;
        let ini = parse_ini(&text).map_err(|e| {
            format!(
                "cannot parse deployment app.ini at {}: {e}",
                paths.app_ini.display()
            )
        })?;
        // has_option consults DEFAULT too; get interpolates before strip.
        // A missing [server] section raises NoSectionError outside the
        // parse guard, so it surfaces as a bare failure here too.
        let section = ini
            .sections
            .get("server")
            .ok_or_else(|| "No section: 'server'".to_string())?;
        let key = "app_data_path".to_string();
        if section.contains_key(&key) || ini.defaults.contains_key(&key) {
            let raw = ini_get(&ini, "server", "APP_DATA_PATH").unwrap_or_default();
            let interpolated = interpolate(&raw, section, &ini.defaults).map_err(|e| {
                format!(
                    "cannot parse deployment app.ini at {}: {e}",
                    paths.app_ini.display()
                )
            })?;
            value = Some(interpolated.trim().to_string());
        }
    }
    let value = value.unwrap_or_default();
    if value.is_empty() {
        return Err(format!(
            "APP_DATA_PATH is not set in {} nor {}; refusing to guess",
            paths.app_ini.display(),
            paths.env_file.display()
        ));
    }
    if !value.starts_with('/') {
        return Err(format!(
            "APP_DATA_PATH {value:?} is relative; refusing to resolve against an unknown work path"
        ));
    }
    Ok(value)
}

/// Map the container-side marker path onto the host data volume.
pub(crate) fn marker_path(paths: &Paths) -> Result<PathBuf, String> {
    let app_data = app_data_path(paths)?;
    let container = format!("{}/{MARKER_NAME}", app_data.trim_end_matches('/'));
    if container != "/data" && !container.starts_with("/data/") {
        return Err(format!(
            "AppDataPath {container:?} is outside the /data volume; cannot map to the host"
        ));
    }
    let relative = if container == "/data" {
        ".".to_string()
    } else {
        container["/data/".len()..].to_string()
    };
    let host = paths.data_root.join(relative);
    for parent in [&host, &host.parent().unwrap_or(&host).to_path_buf()] {
        if fs::symlink_metadata(parent)
            .map(|st| st.file_type().is_symlink())
            .unwrap_or(false)
        {
            return Err(format!(
                "marker path {} must not contain symlinks",
                parent.display()
            ));
        }
    }
    Ok(host)
}
