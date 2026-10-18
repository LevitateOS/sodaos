use std::collections::{BTreeMap, BTreeSet};
use std::io::Read;

use crate::buildx::{is_digest, is_revision, oci_architecture, Image as BuildImage};
use crate::model::path_clean;
use crate::Error;

use super::schema::{inspect_oci_image, read_oci_blob, read_oci_index, BlobSource};
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

fn is_archive_root_entry(name: &str, is_dir: bool) -> bool {
    name == "." && is_dir
}

fn admit_archive_name(
    seen: &mut BTreeSet<String>,
    raw_name: &str,
    name: &str,
) -> Result<(), Error> {
    if raw_name.starts_with('/') || name == ".." || name.starts_with("../") {
        return Err(Error::msg("unsafe OCI path"));
    }
    if !seen.insert(name.to_owned()) {
        return Err(Error::msg("duplicate OCI entry"));
    }
    if seen.len() > 100000 {
        return Err(Error::msg("too many OCI entries"));
    }
    Ok(())
}

fn ensure_known_archive_dir(name: &str) -> Result<(), Error> {
    if name != "blobs" && name != "blobs/sha256" {
        return Err(Error::msg("unexpected OCI directory"));
    }
    Ok(())
}

fn ensure_regular_archive_slot(
    name: &str,
    is_file: bool,
    entries: &BTreeMap<String, Blob>,
) -> Result<(), Error> {
    if !is_file {
        return Err(Error::msg("non-regular OCI entry"));
    }
    if entries.contains_key(name) {
        return Err(Error::msg("duplicate OCI entry"));
    }
    if entries.len() > 100000 {
        return Err(Error::msg("too many OCI entries"));
    }
    if !is_valid_oci_regular_entry(name) {
        return Err(Error::msg("not an OCI archive"));
    }
    Ok(())
}

fn store_archive_blob<R: Read>(
    entries: &mut BTreeMap<String, Blob>,
    name: &str,
    header: &tar::Header,
    reader: &mut R,
    json_bytes: &mut i64,
) -> Result<(), Error> {
    let size = header
        .size()
        .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
    let length = i64::try_from(size).map_err(|_| Error::msg("invalid OCI blob size"))?;
    read_oci_blob(
        entries,
        name,
        length,
        reader,
        BlobSource::Archive,
        json_bytes,
    )
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
        if is_archive_root_entry(&name, is_dir) {
            continue;
        }
        admit_archive_name(&mut seen, &raw_name, &name)?;
        if is_dir {
            ensure_known_archive_dir(&name)?;
            continue;
        }
        ensure_regular_archive_slot(&name, header.entry_type().is_file(), &entries)?;
        store_archive_blob(&mut entries, &name, &header, &mut item, &mut json_bytes)?;
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

#[cfg(test)]
mod admission_tests {
    use super::{admit_archive_name, read_oci_archive_entries};
    use std::collections::BTreeSet;
    use std::io::Cursor;

    fn file_bytes(name: &str, data: &[u8]) -> Vec<u8> {
        let mut builder = tar::Builder::new(Vec::new());
        let mut header = tar::Header::new_gnu();
        header.set_size(data.len() as u64);
        header.set_mode(0o644);
        header.set_cksum();
        builder.append_data(&mut header, name, data).unwrap();
        builder.into_inner().unwrap()
    }

    fn dir_bytes(name: &str) -> Vec<u8> {
        let mut builder = tar::Builder::new(Vec::new());
        let mut header = tar::Header::new_gnu();
        header.set_entry_type(tar::EntryType::Directory);
        header.set_size(0);
        header.set_mode(0o755);
        header.set_cksum();
        builder.append_data(&mut header, name, &[][..]).unwrap();
        builder.into_inner().unwrap()
    }

    fn two_file_bytes(name: &str, first: &[u8], second: &[u8]) -> Vec<u8> {
        let mut builder = tar::Builder::new(Vec::new());
        for data in [first, second] {
            let mut header = tar::Header::new_gnu();
            header.set_size(data.len() as u64);
            header.set_mode(0o644);
            header.set_cksum();
            builder.append_data(&mut header, name, data).unwrap();
        }
        builder.into_inner().unwrap()
    }

    #[test]
    fn archive_admission_preserves_validation_precedence() {
        // Absolute and parent-escape paths never reach the tar builder below,
        // which only stores relative entries: lock them at the admission gate.
        let mut seen = BTreeSet::new();
        for (raw, name) in [("/abs", "/abs"), ("..", ".."), ("../escape", "../escape")] {
            assert_eq!(
                admit_archive_name(&mut seen, raw, name).unwrap_err(),
                crate::Error::msg("unsafe OCI path"),
                "{raw}"
            );
        }
        assert_eq!(
            read_oci_archive_entries(Cursor::new(dir_bytes(".")))
                .unwrap()
                .len(),
            0
        );
        for (bytes, message) in [
            (
                two_file_bytes("index.json", b"{}", b"{}"),
                "duplicate OCI entry",
            ),
            (dir_bytes("weird"), "unexpected OCI directory"),
            (file_bytes("random.txt", b"{}"), "not an OCI archive"),
        ] {
            assert_eq!(
                read_oci_archive_entries(Cursor::new(bytes)).unwrap_err(),
                crate::Error::msg(message),
                "{message}"
            );
        }
    }
}
