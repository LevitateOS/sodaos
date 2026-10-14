//! Carry the public, locally admitted tools input into the immutable build
//! snapshot. Git archives deliberately exclude the generated native archive.
use crate::error::Error;
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::fs::{self, File, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::OpenOptionsExt;
use std::path::Path;

use soda_build_tools::reader::canlang::{archive_filename, release_path};

/// Existing admitted reference/license assets copied by the runtime recipe.
pub(crate) fn runtime_content(native: &str) -> Result<Vec<(String, String)>, Error> {
    let root = Path::new(native).join("project-tools/canlang");
    let mut files = vec![(
        "canlang-runtime:/opt/canlang/tools.json".into(),
        root.join("tools.json").to_string_lossy().into_owned(),
    )];
    let licenses = root.join("licenses");
    crate::sys::walk(&licenses.to_string_lossy(), |path, is_dir, _| {
        if is_dir {
            return Ok(());
        }
        if !fs::symlink_metadata(path)?.file_type().is_file() {
            return Err(Error::msg("Canlang attribution must be regular"));
        }
        let rel = crate::sys::rel_path(&licenses.to_string_lossy(), path)?;
        files.push((
            format!(
                "canlang-runtime:/usr/share/licenses/canlang/{}",
                crate::sys::to_slash(&rel)
            ),
            path.to_string(),
        ));
        Ok(())
    })?;
    Ok(files)
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct ArchiveReference {
    archive_filename: String,
    sha256: String,
}

pub(crate) fn carry_tools(source: &Path, snapshot: &Path, arch: &str) -> Result<(), Error> {
    let archive_name = archive_filename(arch).map_err(|error| Error::msg(error.0))?;
    let manifest_path = release_path(arch).map_err(|error| Error::msg(error.0))?;
    let manifest = snapshot.join(manifest_path);
    // Narrow snapshots used by unit callers need no Project tool input. The
    // production asset owner independently requires the complete manifest.
    if !manifest.exists() {
        if arch == "x86_64" && !snapshot.join("system/project/Containerfile").exists() {
            return Ok(());
        }
        return Err(Error::msg(format!(
            "verified Canlang release lock unavailable for {arch}: {manifest_path}"
        )));
    }
    let input: ArchiveReference = serde_json::from_reader(File::open(manifest)?)
        .map_err(|_| Error::msg("invalid Canlang archive reference"))?;
    if input.archive_filename != archive_name
        || input.sha256.len() != 64
        || !input
            .sha256
            .bytes()
            .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
    {
        return Err(Error::msg("invalid Canlang archive reference"));
    }
    let mut archive = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW)
        .open(source.join(".artifacts/canlang").join(archive_name))
        .map_err(|_| {
            Error::msg(
                "stage the exact Canlang tools archive under .artifacts/canlang before building",
            )
        })?;
    let metadata = archive.metadata()?;
    if !metadata.is_file() || metadata.len() > 1_000_000_000 {
        return Err(Error::msg("invalid Canlang archive input"));
    }
    let directory = snapshot.join(".artifacts/canlang");
    fs::create_dir_all(&directory)?;
    let out = directory.join(archive_name);
    let mut target = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o644)
        .open(&out)?;
    let mut digest = Sha256::new();
    let mut buffer = [0u8; 65536];
    loop {
        let bytes = archive.read(&mut buffer)?;
        if bytes == 0 {
            break;
        }
        digest.update(&buffer[..bytes]);
        target.write_all(&buffer[..bytes])?;
    }
    if crate::sys::hex_bytes(&digest.finalize()) != input.sha256 {
        drop(target);
        fs::remove_file(out)?;
        return Err(Error::msg(
            "Canlang tools input checksum differs from the source release manifest",
        ));
    }
    target.sync_all()?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn carries_only_exact_locked_public_bytes_into_snapshot() {
        for arch in ["x86_64", "aarch64"] {
            let archive_name = archive_filename(arch).unwrap();
            let stamp = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos();
            let root =
                std::env::temp_dir().join(format!("canlang-input-{}-{stamp}", std::process::id()));
            let source = root.join("source");
            let snapshot = root.join("snapshot");
            fs::create_dir_all(source.join(".artifacts/canlang")).unwrap();
            fs::create_dir_all(snapshot.join("system/project")).unwrap();
            if arch == "aarch64" {
                assert!(carry_tools(&source, &snapshot, arch)
                    .unwrap_err()
                    .to_string()
                    .contains("release lock unavailable for aarch64"));
            }
            let bytes = b"locked generated public tools input";
            let path = source.join(".artifacts/canlang").join(archive_name);
            fs::write(&path, bytes).unwrap();
            let manifest = serde_json::json!({"archiveFilename":archive_name, "sha256":crate::sys::hex_bytes(&Sha256::digest(bytes))});
            fs::write(
                snapshot.join(release_path(arch).unwrap()),
                manifest.to_string(),
            )
            .unwrap();
            carry_tools(&source, &snapshot, arch).unwrap();
            assert_eq!(
                fs::read(snapshot.join(".artifacts/canlang").join(archive_name)).unwrap(),
                bytes
            );
            assert!(carry_tools(&source, &snapshot, arch).is_err());
            fs::remove_file(snapshot.join(".artifacts/canlang").join(archive_name)).unwrap();
            fs::write(&path, b"different input").unwrap();
            assert!(carry_tools(&source, &snapshot, arch)
                .unwrap_err()
                .to_string()
                .contains("checksum"));
            assert!(!snapshot
                .join(".artifacts/canlang")
                .join(archive_name)
                .exists());
            fs::remove_dir_all(root).unwrap();
        }
    }
}
