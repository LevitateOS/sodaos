//! Read only the Docker recipe/support inputs from the archived Soda revision.
use std::collections::BTreeMap;
use std::io::Read;
use std::process::{Command, Stdio};

use crate::buildx::{open_layout_entry, Root};
use crate::Error;

pub(super) const SOURCE_FILES: [&str; 7] = [
    "system/docker/Dockerfile",
    "system/docker/compose.yaml",
    "system/docker/soda-docker-init.service",
    "system/docker/image-import.conf",
    "LICENSE",
    "NOTICE",
    "fountain-source.json",
];

pub(super) fn verify_revision(root: &Root, archive: &str, revision: &str) -> Result<(), Error> {
    let (file, _) = open_layout_entry(root, archive)?;
    let result = Command::new("git")
        .arg("get-tar-commit-id")
        .stdin(Stdio::from(file))
        .output()
        .map_err(|_| Error::msg("source archive revision inspection unavailable"))?;
    if !result.status.success() || String::from_utf8_lossy(&result.stdout).trim() != revision {
        return Err(Error::msg("Docker source archive revision mismatch"));
    }
    Ok(())
}

pub(super) fn read_inputs(root: &Root) -> Result<BTreeMap<String, Vec<u8>>, Error> {
    let (file, _) = open_layout_entry(root, "source.tar")?;
    let mut archive = tar::Archive::new(file);
    let mut found = BTreeMap::new();
    for entry in archive.entries().map_err(|_| Error::refused())? {
        let mut entry = entry.map_err(|_| Error::refused())?;
        let name = entry
            .path()
            .map_err(|_| Error::refused())?
            .to_string_lossy()
            .into_owned();
        if !SOURCE_FILES.contains(&name.as_str()) {
            continue;
        }
        if !entry.header().entry_type().is_file() || found.contains_key(&name) {
            return Err(Error::msg("non-regular or duplicate Docker source input"));
        }
        let mut bytes = Vec::new();
        entry
            .by_ref()
            .take((1 << 20) + 1)
            .read_to_end(&mut bytes)
            .map_err(|_| Error::refused())?;
        if bytes.is_empty() || bytes.len() > 1 << 20 {
            return Err(Error::msg("missing or oversized Docker source input"));
        }
        found.insert(name, bytes);
    }
    if found.len() != SOURCE_FILES.len() {
        return Err(Error::msg("incomplete Docker source inputs"));
    }
    Ok(found)
}
