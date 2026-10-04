//! Recipe and unit-file readers (`production.go`).
//!
//! `single_setting` returns the single `prefix` line of a manifest, and
//! `soda_commands` inventories the appliance command directory. Both match
//! the Go owners, including the empty-value quirk below.

use super::Error;
use std::path::Path;

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

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU64, Ordering};

    static NEXT_ID: AtomicU64 = AtomicU64::new(0);

    fn scratch_dir(case: &str) -> std::path::PathBuf {
        let dir = std::env::temp_dir().join(format!(
            "soda-build-tools-{}-{}-{}",
            std::process::id(),
            NEXT_ID.fetch_add(1, Ordering::SeqCst),
            case
        ));
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    #[test]
    fn single_setting_reads_missing_and_duplicate() {
        let dir = scratch_dir("single-setting");
        let file = dir.join("Containerfile");
        std::fs::write(&file, "FROM base\nARG BASE_IMAGE=rocky:9\n").unwrap();
        assert_eq!(recipe_base(&file).unwrap(), "rocky:9");
        std::fs::write(&file, "FROM base\n").unwrap();
        assert_eq!(
            recipe_base(&file).unwrap_err(),
            Error("missing ARG BASE_IMAGE= input".to_string())
        );
        std::fs::write(&file, "ARG BASE_IMAGE=a\nARG BASE_IMAGE=b\n").unwrap();
        assert_eq!(
            recipe_base(&file).unwrap_err(),
            Error("duplicate ARG BASE_IMAGE= input".to_string())
        );
        // Empty values never trip the duplicate detector, matching Go.
        std::fs::write(&file, "ARG BASE_IMAGE=\nARG BASE_IMAGE=\n").unwrap();
        assert_eq!(
            recipe_base(&file).unwrap_err(),
            Error("missing ARG BASE_IMAGE= input".to_string())
        );
        assert!(recipe_base(&dir.join("absent")).is_err());
        let unit = dir.join("svc.container");
        std::fs::write(&unit, "[Container]\nImage=registry.test/svc:latest\n").unwrap();
        assert_eq!(unit_image(&unit).unwrap(), "registry.test/svc:latest");
        std::fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn soda_commands_inventories_sorted_commands() {
        let dir = scratch_dir("soda-commands");
        let cmd = dir.join("cmd");
        std::fs::create_dir_all(cmd.join("soda-host")).unwrap();
        std::fs::create_dir_all(cmd.join("soda-factory")).unwrap();
        std::fs::write(cmd.join("README.md"), "not a command\n").unwrap();
        assert_eq!(
            soda_commands(&cmd).unwrap(),
            vec!["soda-factory".to_string(), "soda-host".to_string()]
        );
        std::fs::create_dir_all(cmd.join("soda-artifacts")).unwrap();
        assert_eq!(
            soda_commands(&cmd).unwrap_err(),
            Error("support tools must remain outside appliance commands".to_string())
        );
        std::fs::remove_dir_all(cmd.join("soda-artifacts")).unwrap();
        std::fs::create_dir_all(cmd.join("Not-A-Command")).unwrap();
        assert_eq!(
            soda_commands(&cmd).unwrap_err(),
            Error("support tools must remain outside appliance commands".to_string())
        );
        let empty = dir.join("empty");
        std::fs::create_dir_all(&empty).unwrap();
        assert_eq!(
            soda_commands(&empty).unwrap_err(),
            Error("missing Soda commands".to_string())
        );
        assert!(soda_commands(&dir.join("absent")).is_err());
        std::fs::remove_dir_all(&dir).ok();
    }
}
