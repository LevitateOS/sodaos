//! `extension_assets.go`: Soda extension asset staging.

use std::fs;

use crate::error::Error;
use crate::jsonio;
use crate::sys;

/// stageExtensionAssets copies only the files emitted by the separate Soda
/// browser build. Bun resolves the JavaScript/CSS graph and records its
/// output; this stage never guesses source dependencies from the old public
/// payload.
pub fn stage_extension_assets(
    native: &str,
    package_dir: &str,
    package_id: &str,
) -> Result<(), Error> {
    let asset_dir = match package_id {
        "soda" => "soda-extension-assets",
        "tailnet" => "tailnet-extension-assets",
        _ => return Err(Error::msg("unknown extension package")),
    };
    let root = sys::join(&[native, asset_dir]);
    let files = extension_asset_inventory(&root)?;
    for (i, name) in files.iter().enumerate() {
        if !safe_extension_asset_name(name) || i > 0 && files[i - 1] == *name {
            return Err(Error::msg("unsafe Soda extension asset inventory"));
        }
        link_extension_asset(&root, package_dir, name)?;
    }
    Ok(())
}

pub fn extension_asset_inventory(root: &str) -> Result<Vec<String>, Error> {
    let data = fs::read(sys::join(&[root, "files.json"]))?;
    let text = std::str::from_utf8(&data).map_err(|_| Error::msg("invalid JSON"))?;
    let files: Vec<String> = jsonio::parse(text)
        .map_err(|_| Error::msg("sorted Soda extension asset inventory required"))?;
    // slices.IsSorted: non-decreasing; adjacent duplicates are refused by the stager.
    if files.is_empty() || files.windows(2).any(|w| w[0] > w[1]) {
        return Err(Error::msg("sorted Soda extension asset inventory required"));
    }
    Ok(files)
}

pub fn safe_extension_asset_name(name: &str) -> bool {
    if name.is_empty() || name.starts_with('/') || name.starts_with("../") || name.contains('\\') {
        return false;
    }
    sys::is_clean_path(name)
}

pub fn link_extension_asset(root: &str, package_dir: &str, name: &str) -> Result<(), Error> {
    let from = sys::join(&[root, "assets", name]);
    let info = fs::symlink_metadata(&from)
        .map_err(|_| Error::msg("regular Soda extension asset required"))?;
    if !info.file_type().is_file() {
        return Err(Error::msg("regular Soda extension asset required"));
    }
    let to = sys::join(&[package_dir, "assets", name]);
    fs::create_dir_all(sys::dir_name(&to))?;
    fs::hard_link(&from, &to).map_err(|e| Error::msg(e.to_string()))?;
    Ok(())
}
