//! Outer OCI archive custody: input admission, tar entry gates, and
//! bounded blob collection with digest verification.

use super::Blob;
use crate::files::{is_digest, oci_architecture};
use crate::{io_error, path_clean, Error};
use soda_json::JsonValue;
use std::collections::HashMap;
use std::fs::File as FsFile;
use std::io::Read;
use std::path::Path;

pub(super) fn open_oci_archive(
    file: &Path,
    arch: &str,
    revision: &str,
) -> Result<(String, FsFile), Error> {
    let want = oci_architecture(arch)?.to_string();
    if !revision.is_empty() && !crate::files::is_revision(revision) {
        return Err(Error::msg("full source revision required"));
    }
    let st = std::fs::symlink_metadata(file).map_err(|e| io_error("lstat", file, e))?;
    if !st.is_file() || st.file_type().is_symlink() {
        return Err(Error::msg("OCI archive must be regular"));
    }
    let f = FsFile::open(file).map_err(|e| io_error("open", file, e))?;
    Ok((want, f))
}

fn check_tar_entry_name(name: &str, is_dir: bool) -> Result<bool, Error> {
    if name == "." && is_dir {
        return Ok(true);
    }
    if name.starts_with('/') || name == ".." || name.starts_with("../") {
        return Err(Error::msg("unsafe OCI path"));
    }
    Ok(false)
}

fn check_tar_directory(name: &str, is_dir: bool) -> Result<bool, Error> {
    if !is_dir {
        return Ok(false);
    }
    if name != "blobs" && name != "blobs/sha256" {
        return Err(Error::msg("unexpected OCI directory"));
    }
    Ok(true)
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

fn record_tar_entry(seen: &mut HashMap<String, ()>, name: &str) -> Result<(), Error> {
    if seen.contains_key(name) {
        return Err(Error::msg("duplicate OCI entry"));
    }
    seen.insert(name.to_string(), ());
    if seen.len() > 100_000 {
        return Err(Error::msg("too many OCI entries"));
    }
    Ok(())
}

/// Reads exactly `length` bytes, hashing throughout and retaining the body
/// only for small valid-JSON blobs.
fn copy_oci_blob(
    name: &str,
    length: i64,
    reader: &mut dyn Read,
) -> Result<(String, i64, Option<Vec<u8>>), Error> {
    use sha2::Digest;
    let mut hasher = sha2::Sha256::new();
    let mut data = Vec::new();
    let retain = length <= 4 << 20;
    let mut remaining = length;
    let mut buf = [0u8; 32 << 10];
    loop {
        if remaining <= 0 {
            break;
        }
        // Read one byte past the declared size to detect growth, like Go's
        // LimitReader(length+1) + size comparison.
        let want = (remaining + 1).min(buf.len() as i64) as usize;
        let n = reader
            .read(&mut buf[..want])
            .map_err(|e| Error::msg(e.to_string()))?;
        if n == 0 {
            break;
        }
        hasher.update(&buf[..n]);
        if retain {
            data.extend_from_slice(&buf[..n]);
        }
        remaining -= n as i64;
    }
    let size = length - remaining;
    if size != length {
        return Err(Error::msg("OCI blob size changed"));
    }
    let digest = {
        let mut out = String::with_capacity(64);
        for byte in hasher.finalize() {
            out.push_str(&format!("{byte:02x}"));
        }
        out
    };
    if name.starts_with("blobs/") && name != format!("blobs/sha256/{digest}").as_str() {
        return Err(Error::msg("OCI blob checksum mismatch"));
    }
    let body = if is_json_bytes(&data) {
        Some(data)
    } else {
        None
    };
    Ok((digest, size, body))
}

fn is_json_bytes(data: &[u8]) -> bool {
    if data.is_empty() {
        return false;
    }
    match std::str::from_utf8(data) {
        Ok(text) => JsonValue::parse(text).is_ok(),
        Err(_) => false,
    }
}

pub(crate) fn read_oci_blob(
    entries: &mut HashMap<String, Blob>,
    name: &str,
    length: i64,
    reader: &mut dyn Read,
    json_bytes: &mut usize,
) -> Result<(), Error> {
    if length < 0 || length == i64::MAX {
        return Err(Error::msg("invalid OCI blob size"));
    }
    let (sum, size, body) = copy_oci_blob(name, length, reader)?;
    *json_bytes += body.as_ref().map(Vec::len).unwrap_or(0);
    if *json_bytes > 32 << 20 {
        return Err(Error::msg("OCI JSON metadata limit exceeded"));
    }
    entries.insert(
        name.to_string(),
        Blob {
            hash: sum,
            size,
            data: body,
        },
    );
    Ok(())
}

pub(super) fn entry_raw_name(entry: &tar::Entry<'_, impl Read>) -> String {
    match entry.path() {
        Ok(path) => path.to_string_lossy().into_owned(),
        Err(_) => String::from_utf8_lossy(&entry.path_bytes()).into_owned(),
    }
}

fn read_archive_blob_entry(
    entries: &mut HashMap<String, Blob>,
    name: &str,
    size: i64,
    is_regular: bool,
    reader: &mut dyn Read,
    json_bytes: &mut usize,
) -> Result<(), Error> {
    if !is_regular {
        return Err(Error::msg("non-regular OCI entry"));
    }
    if entries.contains_key(name) {
        return Err(Error::msg("duplicate OCI entry"));
    }
    if entries.len() > 100_000 {
        return Err(Error::msg("too many OCI entries"));
    }
    if !is_valid_oci_regular_entry(name) {
        return Err(Error::msg("not an OCI archive"));
    }
    read_oci_blob(entries, name, size, reader, json_bytes)
}

pub(super) fn read_oci_archive_entries(
    reader: &mut dyn Read,
) -> Result<HashMap<String, Blob>, Error> {
    let mut archive = tar::Archive::new(reader);
    let mut entries: HashMap<String, Blob> = HashMap::new();
    let mut seen: HashMap<String, ()> = HashMap::new();
    let mut json_bytes = 0usize;
    let listed = archive.entries().map_err(|e| Error::msg(e.to_string()))?;
    for entry in listed {
        let mut entry = entry.map_err(|e| Error::msg(e.to_string()))?;
        let raw = entry_raw_name(&entry);
        let name = path_clean(&raw);
        let header = entry.header().clone();
        let entry_type = header.entry_type();
        let is_dir = entry_type.is_dir();
        let is_regular = entry_type.is_file();
        if check_tar_entry_name(&name, is_dir)? {
            continue;
        }
        record_tar_entry(&mut seen, &name)?;
        if check_tar_directory(&name, is_dir)? {
            continue;
        }
        let size = header.size().map_err(|e| Error::msg(e.to_string()))? as i64;
        read_archive_blob_entry(
            &mut entries,
            &name,
            size,
            is_regular,
            &mut entry,
            &mut json_bytes,
        )?;
    }
    Ok(entries)
}
