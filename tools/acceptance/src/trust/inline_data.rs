use crate::structured::Value as JsonValue;
use base64::engine::general_purpose::{GeneralPurpose, GeneralPurposeConfig};
use base64::engine::DecodePaddingMode;
use base64::{alphabet, Engine};
use percent_encoding::percent_decode_str;

use crate::error::Error;

/// Inline gzip bound: 1 MiB of decoded bytes.
pub(crate) const INLINE_GZIP_LIMIT: u64 = 1 << 20;
const INLINE_GZIP_COMPRESSED_LIMIT: usize = 2 << 20;

/// Standard-alphabet Go `StdEncoding` decode: canonical padding with unused
/// trailing bits ignored.
pub fn decode_base64(input: &str) -> Result<Vec<u8>, Error> {
    let engine = GeneralPurpose::new(
        &alphabet::STANDARD,
        GeneralPurposeConfig::new()
            .with_decode_padding_mode(DecodePaddingMode::RequireCanonical)
            .with_decode_allow_trailing_bits(true),
    );
    engine
        .decode(input)
        .map_err(|_| Error::msg("invalid base64"))
}

/// Standard base64 encode, for tests only. Production never emits secrets.
#[cfg(test)]
pub fn encode_base64(input: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD.encode(input)
}

/// Percent-decode bytes, like Go's `url.PathUnescape`: `+` stays literal,
/// malformed escapes fail, and non-UTF-8 results are kept as bytes.
fn percent_decode(input: &str) -> Result<Vec<u8>, Error> {
    if !valid_percent_escapes(input) {
        return Err(Error::msg("invalid inline encoding"));
    }
    Ok(percent_decode_str(input).collect())
}

fn valid_percent_escapes(input: &str) -> bool {
    let bytes = input.as_bytes();
    let mut at = 0;
    while at < bytes.len() {
        if bytes[at] == b'%' {
            if at + 2 >= bytes.len()
                || !bytes[at + 1].is_ascii_hexdigit()
                || !bytes[at + 2].is_ascii_hexdigit()
            {
                return false;
            }
            at += 3;
        } else {
            at += 1;
        }
    }
    true
}

/// Decode an inline `data:` URI: unescape first, then base64-decode when
/// the mediatype ends with `;base64`. Mirrors `decodeDataURI`.
pub fn decode_data_uri(source: &str) -> Result<Vec<u8>, Error> {
    let (head, data) = source
        .split_once(',')
        .ok_or_else(|| Error::msg("inline data URI required"))?;
    if !head.starts_with("data:") {
        return Err(Error::msg("inline data URI required"));
    }
    let unescaped = percent_decode(data)?;
    if !head.ends_with(";base64") {
        return Ok(unescaped);
    }
    let text = String::from_utf8(unescaped).map_err(|_| Error::msg("invalid inline base64"))?;
    decode_base64(&text).map_err(|_| Error::msg("invalid inline base64"))
}

/// Bounded gzip decode with the Go owner's error taxonomy.
pub fn gunzip_bounded(data: &[u8]) -> Result<Vec<u8>, Error> {
    gunzip_with_limit(data, INLINE_GZIP_LIMIT)
}

fn gunzip_with_limit(data: &[u8], limit: u64) -> Result<Vec<u8>, Error> {
    if data.len() > INLINE_GZIP_COMPRESSED_LIMIT {
        return Err(Error::msg("invalid or oversized compressed inline data"));
    }
    if data.len() < 2 || data[0] != 0x1f || data[1] != 0x8b {
        return Err(Error::msg("invalid compressed inline data"));
    }
    let mut decoder = flate2::read::MultiGzDecoder::new(data);
    let mut decoded = Vec::new();
    let mut bounded = std::io::Read::take(&mut decoder, limit.saturating_add(1));
    match std::io::Read::read_to_end(&mut bounded, &mut decoded) {
        Ok(_) if decoded.len() as u64 <= limit => Ok(decoded),
        _ => Err(Error::msg("invalid or oversized compressed inline data")),
    }
}

/// Decode one file's inline bytes: plain or bounded gzip. Mirrors
/// `inlineData`, including raw error passthrough.
pub fn inline_data(source: &str, compression: &str) -> Result<Vec<u8>, Error> {
    let limit = if compression == "gzip" {
        INLINE_GZIP_LIMIT as usize
    } else {
        usize::MAX
    };
    inline_data_limited(source, compression, limit)
}

/// Decode through the normal inline path with an additional caller budget.
/// Gzip output stops at the limit plus one byte; plain URI decoding is bounded
/// by its source text and checked against the caller limit before returning.
pub fn inline_data_limited(
    source: &str,
    compression: &str,
    limit: usize,
) -> Result<Vec<u8>, Error> {
    if compression == "gzip" && source.len() > INLINE_GZIP_COMPRESSED_LIMIT * 3 + 64 {
        return Err(Error::msg("invalid or oversized compressed inline data"));
    }
    let data = decode_data_uri(source)?;
    if compression.is_empty() {
        if data.len() > limit {
            return Err(Error::msg("invalid or oversized inline data"));
        }
        return Ok(data);
    }
    if compression != "gzip" {
        return Err(Error::msg("unsupported inline compression"));
    }
    gunzip_with_limit(&data, limit as u64)
}

/// Ignition file entry (lenient shape: unknown fields ignored, like Go's
/// plain `Unmarshal`).
pub struct IgnitionFile {
    /// Absolute target path.
    pub path: String,
    /// Inline source URI.
    pub source: String,
    /// Compression name, when present.
    pub compression: String,
}

/// Borrowed Ignition file entry for callers that must account source bytes
/// before retaining copies. Unknown fields remain ignored.
pub struct IgnitionFileRef<'a> {
    pub path: &'a str,
    pub source: &'a str,
    pub compression: &'a str,
}

fn borrowed_string_field<'a>(
    contents: Option<&'a JsonValue>,
    name: &str,
) -> Result<&'a str, Error> {
    match contents.and_then(|c| c.get(name)) {
        None | Some(JsonValue::Null) => Ok(""),
        Some(JsonValue::Str(s)) => Ok(s),
        Some(_) => Err(Error::msg(format!("invalid {name}: string required"))),
    }
}

pub fn decode_ignition_file_refs(value: &JsonValue) -> Result<Vec<IgnitionFileRef<'_>>, Error> {
    let mut files = Vec::new();
    let storage = match value.get("storage") {
        None | Some(JsonValue::Null) => return Ok(files),
        Some(JsonValue::Object(_)) => value.get("storage"),
        Some(_) => return Err(Error::msg("invalid storage: object required")),
    };
    let items = match storage.and_then(|s| s.get("files")) {
        None | Some(JsonValue::Null) => return Ok(files),
        Some(JsonValue::Array(items)) => items,
        Some(_) => return Err(Error::msg("invalid files: array required")),
    };
    for item in items {
        let JsonValue::Object(_) = item else {
            return Err(Error::msg("invalid file: object required"));
        };
        let contents = match item.get("contents") {
            None | Some(JsonValue::Null) => None,
            Some(JsonValue::Object(_)) => item.get("contents"),
            Some(_) => return Err(Error::msg("invalid contents: object required")),
        };
        let path = match item.get("path") {
            None | Some(JsonValue::Null) => "",
            Some(JsonValue::Str(path)) => path,
            Some(_) => return Err(Error::msg("invalid path: string required")),
        };
        files.push(IgnitionFileRef {
            path,
            source: borrowed_string_field(contents, "source")?,
            compression: borrowed_string_field(contents, "compression")?,
        });
    }
    Ok(files)
}

/// Decode the storage files of an Ignition document, ignoring unknown
/// fields like the Go owner does. Present-but-mistyped shapes fail like
/// Go's `Unmarshal` type errors; callers map them to their own message.
pub fn decode_ignition_files(value: &JsonValue) -> Result<Vec<IgnitionFile>, Error> {
    Ok(decode_ignition_file_refs(value)?
        .into_iter()
        .map(|file| IgnitionFile {
            path: file.path.to_string(),
            source: file.source.to_string(),
            compression: file.compression.to_string(),
        })
        .collect())
}
