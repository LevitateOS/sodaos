//! Bounded JSON file custody for owner-defined JSON records.
//!
//! Schema decoding belongs to each owner. Readers return Serde's validated
//! raw value together with a digest of the exact original bytes.

use std::fs::File;
use std::io::Read;
use std::path::Path;

use serde_json::value::RawValue;

use crate::error::Error;
use crate::files::{self, FileAttr, OwnedDir};
use crate::sha256;

pub use crate::timestamps::{now_rfc3339_nano, validate_rfc3339};

/// Read one bounded JSON value, returning its raw token and the SHA-256 of
/// the exact bytes read. The caller owns the schema and root policy.
pub fn read_json_file(path: &str) -> Result<(Box<RawValue>, String), Error> {
    let file = Path::new(path);
    let parent = file.parent().unwrap_or(Path::new("/"));
    let base = file
        .file_name()
        .ok_or_else(|| Error::msg("bounded regular JSON input required"))?;
    let root = OwnedDir::open(&parent.to_string_lossy())?;
    let name = base.to_string_lossy();
    let before = root.lstat_at(&name)?;
    if !before.is_regular || before.size > files::JSON_LIMIT {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    let handle = root.open_file_at(&name)?;
    let after = fstat_attr(&handle)?;
    if !after.is_regular || !files::same_file(before, after) {
        return Err(Error::msg("JSON input changed before reading"));
    }
    let mut data = Vec::new();
    handle.take(files::JSON_LIMIT + 1).read_to_end(&mut data)?;
    if data.len() as u64 > files::JSON_LIMIT {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    let value = serde_json::from_slice(&data).map_err(|_| Error::msg("invalid JSON input"))?;
    Ok((value, sha256::hex_lower(&sha256::digest(&data))))
}

/// Read a bounded JSON value through an already-open directory, retaining
/// exact input-byte hashing separately from the parsed token.
pub fn read_json_at(root: &OwnedDir, name: &str) -> Result<(Box<RawValue>, String), Error> {
    let before = root.lstat_at(name)?;
    if !before.is_regular || before.size > files::JSON_LIMIT {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    let handle = root.open_file_at(name)?;
    let after = fstat_attr(&handle)?;
    if !after.is_regular || !files::same_file(before, after) {
        return Err(Error::msg("JSON input changed before reading"));
    }
    let mut data = Vec::new();
    handle.take(files::JSON_LIMIT + 1).read_to_end(&mut data)?;
    if data.len() as u64 > files::JSON_LIMIT {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    let value = serde_json::from_slice(&data).map_err(|_| Error::msg("invalid JSON input"))?;
    Ok((value, sha256::hex_lower(&sha256::digest(&data))))
}

fn fstat_attr(file: &File) -> Result<FileAttr, Error> {
    use std::os::fd::AsRawFd;
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::fstat(file.as_raw_fd(), &mut st) };
    if rc != 0 {
        return Err(Error::from(std::io::Error::last_os_error()));
    }
    let mode = st.st_mode;
    Ok(FileAttr {
        is_regular: mode & libc::S_IFMT == libc::S_IFREG,
        is_dir: mode & libc::S_IFMT == libc::S_IFDIR,
        is_symlink: mode & libc::S_IFMT == libc::S_IFLNK,
        perm: mode & 0o7777,
        size: st.st_size.max(0) as u64,
        uid: st.st_uid,
        gid: st.st_gid,
        dev: st.st_dev as u64,
        ino: st.st_ino as u64,
    })
}

/// Go JSON quote escaping for credential bytes included in a secret set.
pub fn escape_go(out: &mut String, value: &str) {
    out.push('"');
    for ch in value.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '<' => out.push_str("\\u003c"),
            '>' => out.push_str("\\u003e"),
            '&' => out.push_str("\\u0026"),
            '\u{2028}' => out.push_str("\\u2028"),
            '\u{2029}' => out.push_str("\\u2029"),
            c if (c as u32) < 0x20 => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => out.push(c),
        }
    }
    out.push('"');
}

#[cfg(test)]
mod tests;
