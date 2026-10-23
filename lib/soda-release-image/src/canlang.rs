//! Carry the public, locally admitted tools input into the immutable build
//! snapshot. Git archives deliberately exclude the generated native archive.
use crate::error::Error;
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::fs::{self, File, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::OpenOptionsExt;
use std::path::Path;

const ARCHIVE: &str = "canlang-tools-linux-x86_64.tar.gz";

/// Admitted production runtime attributions copied by the runtime recipe.
pub(crate) fn runtime_content(native: &str) -> Result<Vec<(String, String)>, Error> {
    let root = Path::new(native).join("project-tools/canlang");
    let mut files = vec![(
        "canlang-runtime:/opt/canlang/tools.json".into(),
        root.join("tools.json").to_string_lossy().into_owned(),
    )];
    let licenses = root.join("runtime/licenses");
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

pub(crate) fn carry_tools(source: &Path, snapshot: &Path) -> Result<(), Error> {
    let manifest = snapshot.join("system/project/canlang-release.json");
    // Narrow snapshots used by unit callers need no Project tool input. The
    // production asset owner independently requires the complete manifest.
    if !manifest.exists() {
        return Ok(());
    }
    let input: ArchiveReference = serde_json::from_reader(File::open(manifest)?)
        .map_err(|_| Error::msg("invalid Canlang archive reference"))?;
    if input.archive_filename != ARCHIVE
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
        .open(source.join(".artifacts/canlang").join(ARCHIVE))
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
    let out = directory.join(ARCHIVE);
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
