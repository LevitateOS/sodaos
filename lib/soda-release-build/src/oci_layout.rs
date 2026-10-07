//! Shared-blob OCI layout inspection (`oci_layout.go`): verifies the
//! exact named image set and its local blobs without running image code.
//! References are immutable config IDs, as in our OCI exports.

use crate::confined_files::Root;
use crate::files::{is_digest, is_revision, oci_architecture};
use crate::oci::archive::read_oci_blob;
use crate::oci::manifest::{inspect_oci_image, read_oci_index};
use crate::oci::{Blob, Image, LoadBlobs};
use crate::{io_error, Error};
use std::collections::HashMap;
use std::path::Path;

/// Verified shared-blob directory: images by config reference, file hashes
/// (index/layout included), blobs counted exactly once.
#[derive(Debug, Clone, Default)]
pub struct OciLayout {
    pub images: HashMap<String, Image>,
    pub files: HashMap<String, String>,
    pub bytes: u64,
}

fn same_opened(st: &crate::confined_files::FileMeta, actual: &std::fs::Metadata) -> bool {
    use std::os::unix::fs::MetadataExt;
    st.dev == actual.dev() && st.ino == actual.ino()
}

fn validate_oci_layout_inputs(
    arch: &str,
    revisions: &HashMap<String, String>,
) -> Result<String, Error> {
    let want = oci_architecture(arch)?.to_string();
    if revisions.is_empty() {
        return Err(Error::msg("explicit OCI image set required"));
    }
    for (reference, revision) in revisions {
        let hex = reference.strip_prefix("sha256:").unwrap_or("");
        if !reference.starts_with("sha256:")
            || !is_digest(hex)
            || (!revision.is_empty() && !is_revision(revision))
        {
            return Err(Error::msg("invalid OCI image selection"));
        }
    }
    Ok(want)
}

fn open_oci_layout_root(dir: &Path) -> Result<Root, Error> {
    let info = std::fs::symlink_metadata(dir)
        .map_err(|_| Error::msg("real OCI layout directory required"))?;
    if !info.is_dir() || info.file_type().is_symlink() {
        return Err(Error::msg("real OCI layout directory required"));
    }
    Root::open(dir).map_err(|_| Error::msg("real OCI layout directory required"))
}

/// Verifies the exact named image set and its local blobs.
pub fn inspect_oci_layout(
    dir: &Path,
    arch: &str,
    revisions: &HashMap<String, String>,
) -> Result<OciLayout, Error> {
    let want = validate_oci_layout_inputs(arch, revisions)?;
    let root = open_oci_layout_root(dir)?;
    let mut entries: HashMap<String, Blob> = HashMap::new();
    let mut json_bytes = 0usize;
    for name in ["index.json", "oci-layout"] {
        load_layout_entry(&root, &mut entries, &mut json_bytes, name)?;
    }
    let index = read_oci_index(&entries)?;
    if index.len() != revisions.len() {
        return Err(Error::msg("exact OCI image set required"));
    }
    let mut images: HashMap<String, Image> = HashMap::with_capacity(index.len());
    for descriptor in &index {
        let reference = descriptor
            .annotation("org.opencontainers.image.ref.name")
            .unwrap_or("");
        let revision = revisions.get(reference);
        let duplicate = images
            .get(reference)
            .map(|image: &Image| !image.config.is_empty())
            .unwrap_or(false);
        if revision.is_none()
            || duplicate
            || descriptor.media_type != "application/vnd.oci.image.manifest.v1+json"
        {
            return Err(Error::msg("unexpected or duplicate OCI image reference"));
        }
        let image = {
            let mut load = |entries: &mut HashMap<String, Blob>, name: &str| {
                load_layout_entry(&root, entries, &mut json_bytes, name)
            };
            let loader: LoadBlobs<'_> = &mut load;
            inspect_oci_image(&mut entries, descriptor, &want, revision.unwrap(), loader)?
        };
        if image.config != reference {
            return Err(Error::msg("OCI reference differs from config identity"));
        }
        images.insert(reference.to_string(), image);
    }
    let mut files = HashMap::with_capacity(entries.len());
    let mut total_bytes = 0u64;
    for (name, entry) in &entries {
        files.insert(name.clone(), entry.hash.clone());
        total_bytes += entry.size as u64;
    }
    Ok(OciLayout {
        images,
        files,
        bytes: total_bytes,
    })
}

fn load_layout_entry(
    root: &Root,
    entries: &mut HashMap<String, Blob>,
    json_bytes: &mut usize,
    name: &str,
) -> Result<(), Error> {
    if entries.contains_key(name) {
        return Ok(());
    }
    let st = root.lstat(name)?;
    if !st.is_regular {
        return Err(Error::msg("non-regular OCI layout entry"));
    }
    let mut f = root.open_file(name)?;
    let actual = f
        .metadata()
        .map_err(|e| io_error("stat", Path::new(name), e))?;
    if !same_opened(&st, &actual) {
        return Err(Error::msg("OCI layout entry changed"));
    }
    read_oci_blob(entries, name, st.size as i64, &mut f, json_bytes)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sha256_hex;
    use crate::test_support::{fixture_oci_bytes, FIXTURE_REVISION};

    fn layout_fixture() -> (std::path::PathBuf, HashMap<String, String>) {
        // Unpacks two archives into one shared layout.
        let dir = std::env::temp_dir().join(format!(
            "soda-layout-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        let layout = dir.join("layout");
        std::fs::create_dir_all(layout.join("blobs/sha256")).unwrap();
        for _ in 0..2 {
            let bytes = fixture_oci_bytes("amd64");
            let mut archive = tar::Archive::new(&bytes[..]);
            for entry in archive.entries().unwrap() {
                let mut entry = entry.unwrap();
                let path = entry.path().unwrap().into_owned();
                if entry.header().entry_type().is_dir() {
                    continue;
                }
                let mut data = Vec::new();
                std::io::Read::read_to_end(&mut entry, &mut data).unwrap();
                let dest = layout.join(&path);
                if let Some(parent) = dest.parent() {
                    std::fs::create_dir_all(parent).unwrap();
                }
                // The two fixtures share every blob; skip re-writes.
                if dest.exists() && path.starts_with("blobs/") {
                    continue;
                }
                std::fs::write(&dest, &data).unwrap();
            }
        }
        // Rebuild the multi-image index from the unpacked blobs.
        let blobs_dir = layout.join("blobs/sha256");
        let mut index_manifests: Vec<String> = Vec::new();
        let mut revisions = HashMap::new();
        // Find configs: blobs whose JSON has an architecture field.
        let mut names: Vec<String> = std::fs::read_dir(&blobs_dir)
            .unwrap()
            .map(|e| e.unwrap().file_name().to_string_lossy().into_owned())
            .collect();
        names.sort();
        for hex in &names {
            let data = std::fs::read(blobs_dir.join(hex)).unwrap();
            let text = String::from_utf8_lossy(&data);
            if text.contains("\"architecture\"") {
                // Config blob: find the manifest pointing at it.
                for mhex in &names {
                    let mdata = std::fs::read(blobs_dir.join(mhex)).unwrap();
                    let mtext = String::from_utf8_lossy(&mdata);
                    if mtext.contains(&format!("sha256:{hex}")) && mtext.contains("\"layers\"") {
                        let reference = format!("sha256:{hex}");
                        index_manifests.push(format!(
                            "{{\"digest\":\"sha256:{mhex}\",\"size\":{},\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"annotations\":{{\"org.opencontainers.image.ref.name\":\"{reference}\"}}}}",
                            mdata.len()
                        ));
                        revisions.insert(reference, FIXTURE_REVISION.to_string());
                        break;
                    }
                }
            }
        }
        // Two identical fixtures share every blob: duplicate the image with
        // a distinct config so the layout holds two images.
        if revisions.len() < 2 {
            let first_ref = revisions.keys().next().cloned().unwrap_or_default();
            let first_hex = first_ref.trim_start_matches("sha256:");
            let mut config = std::fs::read(blobs_dir.join(first_hex)).unwrap();
            // Flip the revision label to force a distinct config blob.
            let needle = FIXTURE_REVISION.as_bytes();
            if let Some(pos) = config.windows(needle.len()).position(|w| w == needle) {
                config[pos..pos + 3].copy_from_slice(b"bbb");
            }
            let config_hex = sha256_hex(&config);
            // Update the manifest copy to point at the new config.
            let mut manifest_hex = String::new();
            let mut manifest_size = 0usize;
            for mhex in &names {
                let mdata = std::fs::read(blobs_dir.join(mhex)).unwrap();
                let mtext = String::from_utf8_lossy(&mdata);
                if mtext.contains(&format!("sha256:{first_hex}")) && mtext.contains("\"layers\"") {
                    let patched = mtext.replacen(first_hex, &config_hex, 1);
                    manifest_size = patched.len();
                    manifest_hex = sha256_hex(patched.as_bytes());
                    std::fs::write(blobs_dir.join(&manifest_hex), patched.as_bytes()).unwrap();
                    break;
                }
            }
            std::fs::write(blobs_dir.join(&config_hex), &config).unwrap();
            let reference = format!("sha256:{config_hex}");
            index_manifests.push(format!(
                "{{\"digest\":\"sha256:{manifest_hex}\",\"size\":{manifest_size},\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"annotations\":{{\"org.opencontainers.image.ref.name\":\"{reference}\"}}}}",
            ));
            // The flipped config no longer matches the fixture revision, so
            // admit it with an empty revision (attribution unchecked).
            revisions.insert(reference, String::new());
        }
        let index = format!(
            "{{\"schemaVersion\":2,\"manifests\":[{}]}}",
            index_manifests.join(",")
        );
        std::fs::write(layout.join("index.json"), index.as_bytes()).unwrap();
        std::fs::write(
            layout.join("oci-layout"),
            br#"{"imageLayoutVersion":"1.0.0"}"#,
        )
        .unwrap();
        (layout, revisions)
    }

    #[test]
    fn oracle_layout_identities_and_counts() {
        // Oracle: TestSharedOCILayoutPreservesIdentitiesAndCountsBlobsOnce.
        let (dir, revisions) = layout_fixture();
        let got = inspect_oci_layout(&dir, "x86_64", &revisions).unwrap();
        assert_eq!(got.images.len(), 2);
        let mut total = 0u64;
        for (name, hash) in &got.files {
            let bytes = std::fs::read(dir.join(name)).unwrap();
            assert_eq!(&sha256_hex(&bytes), hash);
            total += bytes.len() as u64;
        }
        assert_eq!(total, got.bytes);
        for (reference, image) in &got.images {
            assert_eq!(reference, &image.config);
            assert_eq!(image.architecture, "amd64");
        }
        assert_eq!(
            inspect_oci_layout(&dir, "aarch64", &revisions)
                .unwrap_err()
                .message(),
            "expected x86_64"
        );
    }

    #[test]
    fn oracle_layout_refuses_substitution() {
        // Oracle: TestSharedOCILayoutRefusesSubstitution (representative kinds).
        for kind in [
            "missing",
            "corrupt",
            "symlink-file",
            "symlink-root",
            "wrong-ref",
            "wrong-size",
            "external-url",
            "empty-index",
            "nested-index",
        ] {
            let (dir, revisions) = layout_fixture();
            let before = inspect_oci_layout(&dir, "x86_64", &revisions).unwrap();
            let mut blob = String::new();
            for name in before.files.keys() {
                if name.starts_with("blobs/") {
                    blob = name.clone();
                    break;
                }
            }
            match kind {
                "missing" => std::fs::remove_file(dir.join(&blob)).unwrap(),
                "corrupt" => std::fs::write(dir.join(&blob), b"corrupted").unwrap(),
                "symlink-file" => {
                    let outside = dir.parent().unwrap().join("outside-blob");
                    std::fs::rename(dir.join(&blob), &outside).unwrap();
                    std::os::unix::fs::symlink(&outside, dir.join(&blob)).unwrap();
                }
                "symlink-root" => {
                    let link = dir.with_extension("link");
                    std::os::unix::fs::symlink(&dir, &link).unwrap();
                    assert!(
                        inspect_oci_layout(&link, "x86_64", &revisions).is_err(),
                        "{kind}"
                    );
                    continue;
                }
                other => {
                    let path = dir.join("index.json");
                    let text = std::fs::read_to_string(&path).unwrap();
                    // Rewrite the first manifest descriptor per kind.
                    let patched = match other {
                        "wrong-ref" => text.replacen(
                            "\"org.opencontainers.image.ref.name\":\"sha256:",
                            "\"org.opencontainers.image.ref.name\":\"latest",
                            1,
                        ).replacen("latest", "latest\"", 1),
                        "wrong-size" => {
                            // First size occurrence belongs to the first manifest.
                            text.replacen("\"size\":", "\"size\":1,\"x_size_ignored\":", 1)
                        }
                        "external-url" => text.replacen(
                            "\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\"",
                            "\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"urls\":[\"https://example.invalid/layer\"]",
                            1,
                        ),
                        "empty-index" => {
                            let start = text.find('[').unwrap();
                            let end = text.rfind(']').unwrap();
                            format!("{}{}", &text[..start + 1], &text[end..])
                        }
                        "nested-index" => text.replacen(
                            "application/vnd.oci.image.manifest.v1+json",
                            "application/vnd.oci.image.index.v1+json",
                            1,
                        ),
                        _ => unreachable!(),
                    };
                    std::fs::write(&path, patched).unwrap();
                }
            }
            assert!(
                inspect_oci_layout(&dir, "x86_64", &revisions).is_err(),
                "{kind}"
            );
        }
    }
}
