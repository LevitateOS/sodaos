//! `prepare.go`: release document preparation and candidate verification.

use std::collections::BTreeMap;

use crate::buildx::{hash_at, read_at, Image as BuildImage, Root};
use crate::document::{read_file, write_document};
use crate::jsonx::marshal;
use crate::model::{Candidate, MediaBinding, Release, Trust};
use crate::oci::{inspect_oci, inspect_oci_content};
use crate::payload::{Payload, NAMES};
use crate::{hash_bytes, is_channel, is_digest_ref, Error};

/// `Qualification`: admitted operator release identity plus evidence.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Qualification {
    pub serial: u64,
    pub class: String,
    pub scope: String,
    pub notes: String,
    pub evidence: BTreeMap<String, String>,
}

fn hash_release_provenance(root: &Root, release: &mut Release) -> Result<(), Error> {
    for (name, path) in [
        ("source.tar", "source.tar"),
        ("forgejo-source.tar", "forgejo-source.tar"),
        ("app-inputs.json", "app-inputs.json"),
        ("packages.txt", "packages.txt"),
        (
            "presentation.json",
            "forgejo-context/presentation.json",
        ),
    ] {
        let hash = hash_at(root, path)?;
        release
            .provenance
            .insert(name.to_string(), format!("sha256:{hash}"));
    }
    Ok(())
}

pub(crate) fn read_media_binding(path: &str) -> Result<Vec<u8>, Error> {
    let data = read_file(path, 1 << 20)?;
    let binding = MediaBinding::decode_lenient(&data).map_err(|_| Error::refused())?;
    if !crate::model::valid_media_file(&binding.iso)
        || !crate::model::valid_media_file(&binding.rootfs)
    {
        return Err(Error::refused());
    }
    Ok(data)
}

fn build_release_document(
    t: &Trust,
    candidate: &str,
    media: Vec<u8>,
    q: &Qualification,
) -> Result<Release, Error> {
    let root = Root::open(candidate)?;
    let payload_bytes = read_at(&root, "payload.json", 1 << 20)?;
    let candidate_bytes = read_at(&root, "candidate.json", 1 << 20)?;
    let mut release = Release {
        format: 1,
        serial: q.serial,
        class: q.class.clone(),
        payload: payload_bytes,
        candidate: candidate_bytes,
        media,
        provenance: BTreeMap::new(),
        qualification: q.scope.clone(),
        evidence: q.evidence.clone(),
        notes: q.notes.clone(),
    };
    hash_release_provenance(&root, &mut release)?;
    let (p, c) = release.validate(t)?;
    verify_candidate_images(&root, candidate, &p, &c)?;
    Ok(release)
}

/// `Prepare`: build and package the release document from unchanged inputs.
pub fn prepare(
    t: &Trust,
    candidate: &str,
    media: &str,
    q: &Qualification,
    out: &str,
) -> Result<String, Error> {
    t.validate()?;
    let media_bytes = read_media_binding(media)?;
    let release = build_release_document(t, candidate, media_bytes, q)?;
    write_document(out, &release)
}

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

fn candidate_content_with_embedded_inventory(
    content: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, String>, Error> {
    let data = marshal(content);
    let mut with_inventory = content.clone();
    with_inventory.insert(
        "host:/usr/share/soda/host-image/content.json".to_string(),
        hash_bytes(&data)
            .trim_start_matches("sha256:")
            .to_string(),
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
    if content.is_empty() {
        return inspect_oci(path, arch, revision).map_err(|_| Error::at(&format!("{name} identity")));
    }
    let (expected, members) = candidate_image_content(content, name);
    let member_paths: Vec<String> = members;
    let (image, observed) =
        inspect_oci_content(path, arch, revision, &member_paths).map_err(|_| Error::at(&format!("{name} content")))?;
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
    expected: &crate::payload::Image,
    revision: &str,
    content: &BTreeMap<String, String>,
) -> Result<(String, String), Error> {
    let path = if name == "host" {
        "host.oci".to_string()
    } else {
        format!("images/{name}.oci")
    };
    let hash = match hash_at(root, &path) {
        Ok(hash) if hash == expected.archive_sha256 => hash,
        _ => return Err(Error::at(&format!("{name} archive"))),
    };
    let full = format!("{candidate}/{path}");
    let image = inspect_candidate_image(&full, name, arch, revision, content)?;
    if image.manifest != expected.manifest || image.config != expected.config {
        return Err(Error::at(&format!("{name} identity")));
    }
    Ok((path, hash))
}

fn candidate_image_inputs(
    name: &str,
    p: &Payload,
    c: &Candidate,
) -> Result<(String, BTreeMap<String, String>), Error> {
    let revision = if name == "proxy" {
        String::new()
    } else {
        p.revision.clone()
    };
    match name {
        "host" => Ok((
            revision,
            candidate_content_with_embedded_inventory(&c.content_sha256)?,
        )),
        "dashboard" | "forgejo" | "extension" => Ok((revision, c.content_sha256.clone())),
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
    let mut inputs: BTreeMap<String, crate::payload::Image> = BTreeMap::new();
    inputs.insert(
        "host".to_string(),
        crate::payload::Image {
            config: c.host.config.clone(),
            manifest: c.host.manifest.clone(),
            archive_sha256: c.host_archive_sha256.clone(),
            ..Default::default()
        },
    );
    for (name, image) in &p.images {
        inputs.insert(name.clone(), image.clone());
    }
    let mut names = vec!["host".to_string()];
    names.extend(NAMES.iter().map(|s| s.to_string()));
    for name in names {
        let expected = inputs.get(&name).cloned().unwrap_or_default();
        let (revision, content) = candidate_image_inputs(&name, p, c)?;
        let (path, hash) = verify_candidate_image(
            root,
            candidate,
            &name,
            &p.architecture,
            &expected,
            &revision,
            &content,
        )?;
        files.insert(path, hash);
    }
    Ok(files)
}

/// `ReferenceForDocument`: immutable reference for a prepared document.
pub fn reference_for_document(t: &Trust, kind: &str, digest: &str) -> Result<String, Error> {
    if !is_digest_ref(digest) {
        return Err(Error::refused());
    }
    let repo = if is_channel(kind) {
        format!("{}-channel-{kind}", t.prefix)
    } else if kind == "release" {
        format!("{}-release", t.prefix)
    } else {
        return Err(Error::refused());
    };
    Ok(format!("{repo}@{digest}"))
}

pub(crate) fn immutable_tag(reference: &str) -> String {
    let (repo, digest) = reference.split_once('@').unwrap_or((reference, ""));
    format!(
        "{repo}:sha256-{}",
        digest.trim_start_matches("sha256:")
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn document_references_and_tags() {
        let trust = Trust {
            prefix: "ghcr.io/example/sodaos".to_string(),
            ..Trust::default()
        };
        let digest = format!("sha256:{}", "a".repeat(64));
        assert_eq!(
            reference_for_document(&trust, "release", &digest).unwrap(),
            format!("ghcr.io/example/sodaos-release@{digest}")
        );
        assert_eq!(
            reference_for_document(&trust, "stable", &digest).unwrap(),
            format!("ghcr.io/example/sodaos-channel-stable@{digest}")
        );
        assert!(reference_for_document(&trust, "bogus", &digest).is_err());
        assert!(reference_for_document(&trust, "release", "nope").is_err());
        assert_eq!(
            immutable_tag(&format!("repo@{}", digest)),
            format!("repo:sha256-{}", "a".repeat(64))
        );
    }

    #[test]
    fn prepare_refuses_bad_trust_first() {
        let trust = Trust::default();
        let err = prepare(
            &trust,
            "/nonexistent",
            "/nonexistent-media",
            &Qualification::default(),
            "/tmp/srd-nope",
        )
        .unwrap_err();
        assert_eq!(err, Error::refused());
    }
}
