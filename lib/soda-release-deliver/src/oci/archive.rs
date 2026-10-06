use std::collections::{BTreeMap, BTreeSet};
use std::io::Read;

use crate::buildx::{is_digest, is_revision, oci_architecture, Image as BuildImage};
use crate::model::path_clean;
use crate::Error;

use super::schema::{inspect_oci_image, read_oci_blob, read_oci_index};
use super::{Blob, MANIFEST_TYPE};

// ---------------------------------------------------------------------------
// Archive ingestion
// ---------------------------------------------------------------------------

pub(super) fn open_oci_archive(
    file: &str,
    arch: &str,
    revision: &str,
) -> Result<(String, std::fs::File), Error> {
    let want = oci_architecture(arch).map_err(|e| Error::msg(e.0))?;
    if !revision.is_empty() && !is_revision(revision) {
        return Err(Error::msg("full source revision required"));
    }
    let st =
        std::fs::symlink_metadata(file).map_err(|e| Error::msg(format!("lstat {file}: {e}")))?;
    if !st.is_file() {
        return Err(Error::msg("OCI archive must be regular"));
    }
    let f = std::fs::File::open(file).map_err(|e| Error::msg(format!("open {file}: {e}")))?;
    Ok((want.to_string(), f))
}

fn is_valid_oci_regular_entry(name: &str) -> bool {
    if name == "index.json" || name == "oci-layout" {
        return true;
    }
    match name.strip_prefix("blobs/sha256/") {
        Some(hex) => is_digest(hex),
        None => false,
    }
}

pub(super) fn read_oci_archive_entries<R: Read>(
    reader: R,
) -> Result<BTreeMap<String, Blob>, Error> {
    let mut archive = tar::Archive::new(reader);
    let mut entries: BTreeMap<String, Blob> = BTreeMap::new();
    let mut seen: BTreeSet<String> = BTreeSet::new();
    let mut json_bytes: i64 = 0;
    let items = archive
        .entries()
        .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
    for item in items {
        let mut item = item.map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        let raw_name = String::from_utf8_lossy(&item.path_bytes()).into_owned();
        let name = path_clean(&raw_name);
        let header = item.header().clone();
        let is_dir = header.entry_type().is_dir();
        if name == "." && is_dir {
            continue;
        }
        if raw_name.starts_with('/') || name == ".." || name.starts_with("../") {
            return Err(Error::msg("unsafe OCI path"));
        }
        if !seen.insert(name.clone()) {
            return Err(Error::msg("duplicate OCI entry"));
        }
        if seen.len() > 100000 {
            return Err(Error::msg("too many OCI entries"));
        }
        if is_dir {
            if name != "blobs" && name != "blobs/sha256" {
                return Err(Error::msg("unexpected OCI directory"));
            }
            continue;
        }
        if !header.entry_type().is_file() {
            return Err(Error::msg("non-regular OCI entry"));
        }
        if entries.contains_key(&name) {
            return Err(Error::msg("duplicate OCI entry"));
        }
        if entries.len() > 100000 {
            return Err(Error::msg("too many OCI entries"));
        }
        if !is_valid_oci_regular_entry(&name) {
            return Err(Error::msg("not an OCI archive"));
        }
        let size = header
            .size()
            .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        let length = i64::try_from(size).map_err(|_| Error::msg("invalid OCI blob size"))?;
        let mut data = Vec::new();
        item.read_to_end(&mut data)
            .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        read_oci_blob(&mut entries, &name, length, data, &mut json_bytes)?;
    }
    Ok(entries)
}

pub(super) fn inspect_archive_index(
    entries: &BTreeMap<String, Blob>,
    want: &str,
    revision: &str,
) -> Result<BuildImage, Error> {
    let index = read_oci_index(entries)?;
    if index.len() != 1 || index[0].media_type != MANIFEST_TYPE {
        return Err(Error::msg("single-platform OCI index required"));
    }
    inspect_oci_image(entries, &index[0], want, revision)
}
