use crate::media::MediaLock;
use std::collections::HashMap;

use crate::error::Error;
use crate::foreign::{PackagingInputs, Production};
use crate::media::MediaAuthority;
use crate::model;
use crate::request;
use crate::sys;

pub fn sign_media_input(
    production: &dyn Production,
    authority: &MediaAuthority,
    trust: &model::Trust,
    input: &str,
    transport: &str,
    repository: &str,
    digest: &str,
    out: &str,
) -> Result<(), Error> {
    let trust_home = sys::dir_name(&authority.trust);
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0);
    let permit = model::Permit {
        format: 1,
        repository: repository.to_string(),
        digest: digest.to_string(),
        expires: now + 3600,
        ..model::Permit::default()
    };
    production.sign_media(
        trust,
        &permit,
        transport,
        input,
        out,
        &authority.keys,
        &trust_home,
    )?;
    production.verify_copy(
        trust,
        &format!("{repository}@{digest}"),
        &format!("dir:{}", sys::join(&[out, "signed"])),
        &format!("{out}-admitted"),
        &trust_home,
    )
}

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

pub fn authenticate_packaging_inputs(
    production: &dyn Production,
    request: &request::Request,
    root: &str,
    artifacts: &str,
    archive: &str,
    manifest: &str,
    authority: &MediaAuthority,
    trust: &model::Trust,
    lock: &MediaLock,
) -> Result<(), Error> {
    production.check_native(&sys::dir_name(&authority.trust))?;
    sign_media_input(
        production,
        authority,
        trust,
        archive,
        "oci-archive",
        &format!("{}-host", request.repository_prefix),
        manifest,
        &sys::join(&[root, "host-signature"]),
    )?;
    // The signed public inventory authenticates exactly the auxiliary inputs
    // used by the native packager. No source script runs in this signing
    // operation.
    let inventory = collect_media_inventory(root, artifacts, &request.out)?;
    let document_value = PackagingInputs {
        tools: lock.clone(),
        files: inventory.clone().into_iter().collect(),
    };
    let document = sys::join(&[root, "input-document"]);
    let digest = production.write_document(&document, &document_value)?;
    sign_media_input(
        production,
        authority,
        trust,
        &document,
        "oci",
        &format!("{}-media", request.repository_prefix),
        &digest,
        &sys::join(&[root, "input-signature"]),
    )?;
    verify_media_inventory(&request.out, &inventory)
}
