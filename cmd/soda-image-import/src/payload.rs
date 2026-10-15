//! Canonical Payload v4 admission for the importer.
//!
//! The importer owns only bounded file I/O and trust admission (regular
//! file, 4 MiB cap, same-device recheck). Parsing and validation are the
//! shared canonical implementation; there is no importer-local mirror.

use std::fs;
use std::io::Read;
use std::os::unix::fs::MetadataExt;
use std::path::Path;

pub(super) use soda_build_tools::payload::{ImageBinding, PayloadV4 as Payload, Role};

pub(super) fn load(path: &Path) -> Result<Payload, String> {
    let data = read_bounded_json(path, 4 << 20)?;
    Payload::parse(&data).map_err(|e| e.to_string())
}

fn read_bounded_json(path: &Path, max: u64) -> Result<Vec<u8>, String> {
    let before = fs::symlink_metadata(path).map_err(|e| format!("cannot stat JSON input: {e}"))?;
    if !before.file_type().is_file() || before.len() > max {
        return Err("bounded regular JSON input required".to_string());
    }
    let file = fs::File::open(path).map_err(|e| format!("cannot open JSON input: {e}"))?;
    let after = file
        .metadata()
        .map_err(|e| format!("cannot stat JSON input: {e}"))?;
    if !after.is_file() || after.dev() != before.dev() || after.ino() != before.ino() {
        return Err("JSON input changed before reading".to_string());
    }
    let mut data = Vec::new();
    file.take(max + 1)
        .read_to_end(&mut data)
        .map_err(|e| format!("cannot read JSON input: {e}"))?;
    if data.len() as u64 > max {
        return Err("JSON input exceeds limit".to_string());
    }
    Ok(data)
}
