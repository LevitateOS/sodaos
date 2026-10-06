//! Strict bounded JSON input: confined regular reads with the 4 MiB
//! bound, exact-byte hashing, and the strict binder with its
//! first-value/trailing-data classifier.

use crate::confined_files::Root;
use crate::json_go::Strict;
use crate::{io_error, Error};
use soda_json::JsonValue;
use std::io::Read;
use std::path::Path;

/// Strict bounded JSON read with digest of the exact decoded bytes.
pub fn read_json<T>(
    path: &Path,
    target: &'static str,
    decode: impl FnOnce(&mut Strict<'_>) -> Result<T, String>,
) -> Result<T, Error> {
    let parent = path.parent().unwrap_or(Path::new("/"));
    let root = Root::open(parent)?;
    let name = path
        .file_name()
        .map(|n| n.to_string_lossy().into_owned())
        .unwrap_or_default();
    read_json_at(&root, &name, target, decode).map(|(value, _)| value)
}

/// Strict bounded JSON read through a confined root; returns the value and
/// the digest of the exact bounded bytes decoded.
pub fn read_json_at<T>(
    root: &Root,
    name: &str,
    target: &'static str,
    decode: impl FnOnce(&mut Strict<'_>) -> Result<T, String>,
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
    let text =
        std::str::from_utf8(&data).map_err(|_| Error::msg("invalid character in JSON input"))?;
    let value = match JsonValue::parse(text) {
        Ok(value) => value,
        Err(_) => match first_json_end(text) {
            // A complete value followed by more data: Go's decoder
            // reports trailing data rather than a syntax failure.
            Some(end) if !text[end..].trim().is_empty() => {
                return Err(Error::msg("trailing JSON data"))
            }
            _ => return Err(Error::msg("invalid character in JSON input")),
        },
    };
    let mut binder = Strict::bind(&value, target).map_err(Error::msg)?;
    let decoded = decode(&mut binder).map_err(Error::msg)?;
    binder.finish().map_err(Error::msg)?;
    Ok((decoded, crate::sha256_hex(&data)))
}

/// End offset of the first JSON value in `text`, for trailing-data
/// classification. Lenient where the strict parser is not: only the
/// value/trailing split matters here.
fn first_json_end(text: &str) -> Option<usize> {
    skip_value(text.as_bytes(), skip_ws(text.as_bytes(), 0))
}

fn skip_ws(bytes: &[u8], mut i: usize) -> usize {
    while i < bytes.len() && matches!(bytes[i], b' ' | b'\t' | b'\n' | b'\r') {
        i += 1;
    }
    i
}

fn skip_string(bytes: &[u8], mut i: usize) -> Option<usize> {
    if bytes.get(i) != Some(&b'"') {
        return None;
    }
    i += 1;
    while i < bytes.len() {
        match bytes[i] {
            b'"' => return Some(i + 1),
            b'\\' => i += 2,
            _ => i += 1,
        }
    }
    None
}

fn skip_number(bytes: &[u8], mut i: usize) -> Option<usize> {
    if bytes.get(i) == Some(&b'-') {
        i += 1;
    }
    let start = i;
    while i < bytes.len()
        && (bytes[i].is_ascii_digit() || matches!(bytes[i], b'.' | b'e' | b'E' | b'+' | b'-'))
    {
        i += 1;
    }
    if i == start {
        return None;
    }
    Some(i)
}

fn skip_literal(bytes: &[u8], i: usize, word: &[u8]) -> Option<usize> {
    if bytes.get(i..i + word.len()) == Some(word) {
        Some(i + word.len())
    } else {
        None
    }
}

fn skip_value(bytes: &[u8], i: usize) -> Option<usize> {
    let open = *bytes.get(i)?;
    match open {
        b'{' | b'[' => {
            let close = if open == b'{' { b'}' } else { b']' };
            let mut i = skip_ws(bytes, i + 1);
            if bytes.get(i) == Some(&close) {
                return Some(i + 1);
            }
            loop {
                if open == b'{' {
                    i = skip_string(bytes, i)?;
                    i = skip_ws(bytes, i);
                    if bytes.get(i) != Some(&b':') {
                        return None;
                    }
                    i = skip_ws(bytes, i + 1);
                }
                i = skip_value(bytes, i)?;
                i = skip_ws(bytes, i);
                match bytes.get(i) {
                    Some(b',') => i = skip_ws(bytes, i + 1),
                    Some(c) if *c == close => return Some(i + 1),
                    _ => return None,
                }
            }
        }
        b'"' => skip_string(bytes, i),
        b't' => skip_literal(bytes, i, b"true"),
        b'f' => skip_literal(bytes, i, b"false"),
        b'n' => skip_literal(bytes, i, b"null"),
        b'-' | b'0'..=b'9' => skip_number(bytes, i),
        _ => None,
    }
}
