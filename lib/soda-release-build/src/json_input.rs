//! Strict bounded JSON input: confined regular reads, full-value admission,
//! exact-byte hashing, and typed Serde decoding.

use crate::confined_files::Root;
use crate::{io_error, Error};
use serde::de::DeserializeOwned;
use serde::Deserialize;
use std::io::Read;
use std::path::Path;

/// Parses a Serde-validated JSON number token using Build's original integer
/// grammar. JSON integer `-0` is zero; fractions, exponents and i128 overflow
/// are not integers for the Go-shaped record contracts.
pub(crate) fn parse_integer_token(raw: &serde_json::value::RawValue) -> Option<i128> {
    let token = raw.get();
    let digits = token.strip_prefix('-').unwrap_or(token);
    if digits.is_empty() || !digits.bytes().all(|byte| byte.is_ascii_digit()) {
        return None;
    }
    token.parse::<i128>().ok()
}

/// Strict bounded JSON read with digest of the exact decoded bytes.
pub fn read_json<T: DeserializeOwned + Default>(path: &Path) -> Result<T, Error> {
    let parent = path.parent().unwrap_or(Path::new("/"));
    let root = Root::open(parent)?;
    let name = path
        .file_name()
        .map(|n| n.to_string_lossy().into_owned())
        .unwrap_or_default();
    read_json_at(&root, &name).map(|(value, _)| value)
}

/// Strict bounded JSON read through a confined root; returns the value and
/// the digest of the exact bounded bytes decoded.
pub fn read_json_at<T: DeserializeOwned + Default>(
    root: &Root,
    name: &str,
) -> Result<(T, String), Error> {
    const LIMIT: u64 = 4 << 20;
    let st = root.lstat(name)?;
    if !st.is_regular || st.size > LIMIT {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    let f = root.open_unchanged(name, &st)?;
    let mut data = Vec::new();
    f.take(LIMIT + 1)
        .read_to_end(&mut data)
        .map_err(|e| io_error("read", Path::new(name), e))?;
    if data.len() as u64 > LIMIT {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    std::str::from_utf8(&data).map_err(|_| Error::msg("invalid character in JSON input"))?;

    let mut decoder = serde_json::Deserializer::from_slice(&data);
    let value = Option::<T>::deserialize(&mut decoder)
        .map_err(|_| Error::msg("invalid character in JSON input"))?;
    decoder
        .end()
        .map_err(|_| Error::msg("trailing JSON data"))?;
    Ok((value.unwrap_or_default(), crate::sha256_hex(&data)))
}
