//! Confined file/JSON readers.

use serde::de::DeserializeOwned;

use crate::buildx::{read_at, Root};
use crate::Error;

/// `ReadFile`: bounded regular read through a confined directory handle.
pub fn read_file(path: &str, maximum: i64) -> Result<Vec<u8>, Error> {
    let (parent, base) = match path.rfind('/') {
        Some(0) => ("/", &path[1..]),
        Some(i) => (&path[..i], &path[i + 1..]),
        None => (".", path),
    };
    let root = Root::open(parent)?;
    read_at(&root, base, maximum)
}

/// `ReadJSON`: strict-decode a bounded JSON file.
pub fn read_json<T: DeserializeOwned>(path: &str) -> Result<T, Error> {
    let data = read_file(path, 1 << 20)?;
    crate::json_serde::strict(&data)
}
