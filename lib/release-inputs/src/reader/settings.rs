//! Recipe and unit-file readers (`production.go`).
//!
//! `single_setting` returns the single `prefix` line of a manifest, and
//! `soda_commands` inventories the appliance command directory. Both match
//! the Go owners, including the empty-value quirk below.

use super::Error;
use std::path::Path;

/// Version identity accepted by the GitHub and Docker release producers.
pub fn release_tag(tag: &str) -> bool {
    !tag.is_empty()
        && tag.len() <= 128
        && tag.as_bytes()[0].is_ascii_alphanumeric()
        && tag
            .bytes()
            .all(|c| c.is_ascii_alphanumeric() || b"._-".contains(&c))
        && !tag.contains("..")
}

/// Reads the one `prefix` line of a manifest. Like the Go owner, the
/// duplicate detector keys on a non-empty accumulated value, so two lines
/// that both yield an empty value report `missing` rather than `duplicate`.
pub fn single_setting(path: &Path, prefix: &str) -> Result<String, Error> {
    let data = std::fs::read(path).map_err(|e| Error(e.to_string()))?;
    let text = String::from_utf8_lossy(&data);
    let mut value = String::new();
    for line in text.split('\n') {
        if let Some(rest) = line.strip_prefix(prefix) {
            if !value.is_empty() {
                return Err(Error(format!("duplicate {prefix} input")));
            }
            value = rest.to_string();
        }
    }
    if value.is_empty() {
        return Err(Error(format!("missing {prefix} input")));
    }
    Ok(value)
}

/// Reads the base-image setting of a Containerfile recipe.
pub fn recipe_base(path: &Path) -> Result<String, Error> {
    single_setting(path, "ARG BASE_IMAGE=")
}

/// Reads the image setting of a `.container` unit file.
pub fn unit_image(path: &Path) -> Result<String, Error> {
    single_setting(path, "Image=")
}

/// `^soda-[a-z0-9-]+$` command-directory names.
fn valid_command_name(name: &str) -> bool {
    match name.strip_prefix("soda-") {
        Some(rest) => {
            !rest.is_empty()
                && rest
                    .bytes()
                    .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
        }
        None => false,
    }
}

/// Inventories the appliance command directory in sorted order, as Go's
/// sorted `ReadDir` does. Support tools stay outside `cmd/`.
pub fn soda_commands(cmd_dir: &Path) -> Result<Vec<String>, Error> {
    let entries = std::fs::read_dir(cmd_dir).map_err(|e| Error(e.to_string()))?;
    let mut names = Vec::new();
    for entry in entries {
        let entry = entry.map_err(|e| Error(e.to_string()))?;
        let file_type = entry.file_type().map_err(|e| Error(e.to_string()))?;
        if !file_type.is_dir() {
            continue;
        }
        let name = entry.file_name().to_string_lossy().into_owned();
        if !valid_command_name(&name) || name == "soda-artifacts" || name == "soda-acceptance" {
            return Err(Error(
                "support tools must remain outside appliance commands".to_string(),
            ));
        }
        names.push(name);
    }
    if names.is_empty() {
        return Err(Error("missing Soda commands".to_string()));
    }
    names.sort();
    Ok(names)
}
