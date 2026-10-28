use std::collections::HashMap;

use crate::error::Error;
use crate::request;
use crate::sys;

pub fn collect_media_inventory(
    root: &str,
    artifacts: &str,
    out_dir: &str,
) -> Result<HashMap<String, String>, Error> {
    let mut inventory: HashMap<String, String> = HashMap::new();
    for dir in [
        sys::join(&[root, "config"]),
        sys::join(&[artifacts, "tools"]),
    ] {
        sys::walk(&dir, |path, is_dir, is_link| {
            if is_dir || is_link {
                return Ok(());
            }
            let hash = sys::hash_file(path)?;
            let rel = sys::rel_path(out_dir, path).unwrap_or_else(|_| path.to_string());
            inventory.insert(rel, hash);
            Ok(())
        })?;
    }
    for path in [
        sys::join(&[root, "assembler-root.oci"]),
        sys::join(&[root, "config.tar"]),
        sys::join(&[artifacts, "live.ign"]),
        sys::join(&[artifacts, "destination.ign"]),
        sys::join(&[artifacts, "candidate.json"]),
        sys::join(&[artifacts, "payload.json"]),
        sys::join(&[artifacts, "image-config.json"]),
    ] {
        let hash = sys::hash_file(&path)?;
        let rel = sys::rel_path(out_dir, &path).unwrap_or_else(|_| path.clone());
        inventory.insert(rel, hash);
    }
    Ok(inventory)
}

pub fn verify_media_inventory(
    out_dir: &str,
    inventory: &HashMap<String, String>,
) -> Result<(), Error> {
    let mut paths: Vec<&String> = inventory.keys().collect();
    paths.sort();
    for path in paths {
        let want = &inventory[path];
        match sys::hash_file(&sys::join(&[out_dir, path])) {
            Ok(got) if &got == want => {}
            _ => return Err(Error::msg("packaging input changed after admission")),
        }
    }
    Ok(())
}

/// Hash the auxiliary packaging inputs and check they are unchanged before use.
pub fn authenticate_packaging_inputs(
    request: &request::Request,
    root: &str,
    artifacts: &str,
) -> Result<(), Error> {
    let inventory = collect_media_inventory(root, artifacts, &request.out)?;
    verify_media_inventory(&request.out, &inventory)
}
