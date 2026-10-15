//! Candidate archive and installed-content integrity checks.
use crate::buildx::{hash_at, Image as BuildImage, Root};
use crate::model::Candidate;
use crate::oci::{inspect_oci, inspect_oci_content};
use crate::payload::{Payload, Role};
use crate::{hash_bytes, Error};
use std::collections::BTreeMap;

fn candidate_image_content(
    content: &BTreeMap<String, String>,
    image: &str,
) -> (BTreeMap<String, String>, Vec<String>) {
    let prefix = format!("{image}:");
    let mut expected = BTreeMap::new();
    let mut paths = Vec::new();
    for (name, hash) in content {
        if let Some(member) = name.strip_prefix(&prefix) {
            expected.insert(member.to_string(), hash.clone());
            paths.push(member.to_string());
        }
    }
    paths.sort();
    (expected, paths)
}

pub(crate) fn candidate_content_with_embedded_inventory(
    content: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, String>, Error> {
    let mut data = serde_json::to_vec_pretty(content).map_err(|_| Error::refused())?;
    data.push(b'\n');
    let mut with_inventory = content.clone();
    with_inventory.insert(
        "host:/usr/share/soda/host-image/content.json".to_string(),
        hash_bytes(&data).trim_start_matches("sha256:").to_string(),
    );
    Ok(with_inventory)
}

fn inspect_candidate_image(
    path: &str,
    name: &str,
    arch: &str,
    revision: &str,
    content: &BTreeMap<String, String>,
) -> Result<BuildImage, Error> {
    let (expected, members) = candidate_image_content(content, name);
    if members.is_empty() {
        return inspect_oci(path, arch, revision)
            .map_err(|_| Error::at(&format!("{name} identity")));
    }
    let member_paths: Vec<String> = members;
    let (image, observed) = inspect_oci_content(path, arch, revision, &member_paths)
        .map_err(|_| Error::at(&format!("{name} content")))?;
    for (member, digest) in &expected {
        if observed.get(member) != Some(digest) {
            return Err(Error::at(&format!("{name} content")));
        }
    }
    Ok(image)
}

fn verify_candidate_image(
    root: &Root,
    candidate: &str,
    name: &str,
    arch: &str,
    expected_manifest: &str,
    expected_config: &str,
    expected_archive: &str,
    revision: &str,
    content: &BTreeMap<String, String>,
) -> Result<(String, String), Error> {
    let path = if name == "host" {
        "host.oci".to_string()
    } else {
        format!("images/{name}.oci")
    };
    let hash = match hash_at(root, &path) {
        Ok(hash) if hash == expected_archive => hash,
        _ => return Err(Error::at(&format!("{name} archive"))),
    };
    let full = format!("{candidate}/{path}");
    let image = inspect_candidate_image(&full, name, arch, revision, content)?;
    if image.manifest != expected_manifest || image.config != expected_config {
        return Err(Error::at(&format!("{name} identity")));
    }
    Ok((path, hash))
}

fn candidate_image_inputs(
    name: &str,
    p: &Payload,
    c: &Candidate,
) -> Result<(String, BTreeMap<String, String>), Error> {
    use soda_build_tools::payload::ImageProvenance;
    let revision = match name {
        "host" => p.revision.clone(),
        _ => match Role::from_name(name) {
            Some(role) => match role.binding(&p.images).provenance {
                ImageProvenance::Soda { .. } => p.revision.clone(),
                ImageProvenance::Upstream { .. } => String::new(),
            },
            None => String::new(),
        },
    };
    match name {
        "host" => {
            let mut content = candidate_content_with_embedded_inventory(&c.content_sha256)?;
            for path in [
                &p.project_support.destination,
                &p.project_support.payload_path,
            ] {
                content.insert(format!("host:{path}"), p.project_support.sha256.clone());
            }
            Ok((revision, content))
        }
        "project-os" => Ok((
            revision,
            BTreeMap::from([(
                format!("project-os:{}", p.project_support.destination),
                p.project_support.sha256.clone(),
            )]),
        )),
        "dashboard" | "forgejo" | "canlang-runtime" => Ok((revision, c.content_sha256.clone())),
        _ => Ok((revision, BTreeMap::new())),
    }
}

/// `VerifyCandidateImages`: shared native archive/identity check.
pub fn verify_candidate_images(
    root: &Root,
    candidate: &str,
    p: &Payload,
    c: &Candidate,
) -> Result<BTreeMap<String, String>, Error> {
    let mut files: BTreeMap<String, String> = BTreeMap::new();
    // Expected archive hashes live in candidate transport metadata; the
    // runtime payload carries manifest/config identity only.
    let mut archives: BTreeMap<String, String> = BTreeMap::new();
    archives.insert("host".to_string(), c.host_archive_sha256.clone());
    let local_roles = [
        Role::Dashboard,
        Role::Forgejo,
        Role::ProjectOs,
        Role::CanlangRuntime,
    ];
    for role in local_roles {
        let archive = c
            .image_archives
            .get(role.name())
            .ok_or_else(|| Error::at(&format!("{} archive", role.name())))?;
        if !crate::buildx::is_digest(archive) {
            return Err(Error::at(&format!("{} archive", role.name())));
        }
        archives.insert(role.name().to_string(), archive.clone());
    }
    let mut names = vec!["host".to_string()];
    names.extend(local_roles.into_iter().map(|role| role.name().to_string()));
    for name in names {
        let (expected_manifest, expected_config) = match name.as_str() {
            "host" => (c.host.manifest.clone(), c.host.config.clone()),
            _ => match Role::from_name(&name) {
                Some(role) => {
                    let binding = role.binding(&p.images);
                    (binding.manifest.clone(), binding.config.clone())
                }
                None => return Err(Error::at(&format!("{name} identity"))),
            },
        };
        let expected_archive = archives.get(&name).cloned().unwrap_or_default();
        let (revision, content) = candidate_image_inputs(&name, p, c)?;
        let (path, hash) = verify_candidate_image(
            root,
            candidate,
            &name,
            &p.architecture,
            &expected_manifest,
            &expected_config,
            &expected_archive,
            &revision,
            &content,
        )?;
        files.insert(path, hash);
    }
    for (name, expected) in &c.content_sha256 {
        let path = if let Some(name) = name.strip_prefix("package:") {
            format!("packages/{name}")
        } else if let Some(name) = name.strip_prefix("release:") {
            name.to_string()
        } else {
            continue;
        };
        if hash_at(root, &path)? != *expected {
            return Err(Error::at("candidate package/release content"));
        }
        files.insert(path, expected.clone());
    }
    Ok(files)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::BTreeMap;

    #[test]
    fn embedded_inventory_hash_uses_standard_json_for_hostile_filenames() {
        let content: BTreeMap<String, String> =
            soda_build_tools::reader::candidate::REQUIRED_CONTENT
                .iter()
                .copied()
                .chain([
                    "package:soda-extension-0.1.0-linux-x86_64.tar.gz",
                    "package:tailnet-extension-0.1.0-linux-x86_64.tar.gz",
                    "release:quote\"<&> café.svg",
                ])
                .map(|path| (path.to_string(), "a".repeat(64)))
                .collect();
        assert!(crate::model::valid_candidate_content(&content));
        let with_inventory = candidate_content_with_embedded_inventory(&content).unwrap();
        let files: Vec<(String, String)> = content
            .iter()
            .map(|(path, digest)| (path.clone(), digest.clone()))
            .collect();
        assert!(soda_release_image::model::valid_candidate_content(&files));
        let bytes = soda_release_image::record::content_inventory_bytes(&files).unwrap();
        assert!(std::str::from_utf8(&bytes)
            .unwrap()
            .contains("quote\\\"<&> café.svg"));
        let expected_hash = hash_bytes(&bytes)
            .strip_prefix("sha256:")
            .unwrap()
            .to_string();
        assert_eq!(
            with_inventory.get("host:/usr/share/soda/host-image/content.json"),
            Some(&expected_hash)
        );
    }
}
