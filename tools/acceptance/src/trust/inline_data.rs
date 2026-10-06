use soda_json::JsonValue;

use crate::error::Error;

/// Inline gzip bound: 1 MiB of decoded bytes.
pub(super) const INLINE_GZIP_LIMIT: u64 = 1 << 20;

const B64_ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

fn b64_value(byte: u8) -> Option<u8> {
    B64_ALPHABET
        .iter()
        .position(|b| *b == byte)
        .map(|i| i as u8)
}

/// Strict standard-alphabet base64 decode, like Go's `StdEncoding`.
pub fn decode_base64(input: &str) -> Result<Vec<u8>, Error> {
    let bytes = input.as_bytes();
    if !bytes.len().is_multiple_of(4) {
        return Err(Error::msg("invalid base64"));
    }
    let mut out = Vec::with_capacity(bytes.len() / 4 * 3);
    let (chunks, _) = bytes.as_chunks::<4>();
    let total = chunks.len();
    for (index, chunk) in chunks.iter().enumerate() {
        let last = index + 1 == total;
        let mut values = [0u8; 4];
        let mut padding = 0;
        for (i, byte) in chunk.iter().enumerate() {
            if *byte == b'=' {
                if !last || i < 2 {
                    return Err(Error::msg("invalid base64"));
                }
                padding += 1;
                values[i] = 0;
            } else {
                if padding > 0 {
                    return Err(Error::msg("invalid base64"));
                }
                values[i] = b64_value(*byte).ok_or_else(|| Error::msg("invalid base64"))?;
            }
        }
        let triple = ((values[0] as u32) << 18)
            | ((values[1] as u32) << 12)
            | ((values[2] as u32) << 6)
            | values[3] as u32;
        out.push((triple >> 16) as u8);
        if padding < 2 {
            out.push((triple >> 8) as u8);
        }
        if padding == 0 {
            out.push(triple as u8);
        }
    }
    Ok(out)
}

/// Standard base64 encode, for tests only. Production never emits secrets.
#[cfg(test)]
pub fn encode_base64(input: &[u8]) -> String {
    let mut out = String::new();
    for chunk in input.chunks(3) {
        let mut block = [0u8; 3];
        block[..chunk.len()].copy_from_slice(chunk);
        let triple = ((block[0] as u32) << 16) | ((block[1] as u32) << 8) | block[2] as u32;
        out.push(B64_ALPHABET[(triple >> 18) as usize] as char);
        out.push(B64_ALPHABET[((triple >> 12) & 63) as usize] as char);
        if chunk.len() > 1 {
            out.push(B64_ALPHABET[((triple >> 6) & 63) as usize] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(B64_ALPHABET[(triple & 63) as usize] as char);
        } else {
            out.push('=');
        }
    }
    out
}

/// Percent-decode bytes, like Go's `url.PathUnescape`: `+` stays literal,
/// malformed escapes fail, and non-UTF-8 results are kept as bytes.
fn percent_decode(input: &str) -> Result<Vec<u8>, Error> {
    let bytes = input.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len() {
                return Err(Error::msg("invalid inline encoding"));
            }
            let hex = std::str::from_utf8(&bytes[i + 1..i + 3])
                .map_err(|_| Error::msg("invalid inline encoding"))?;
            let byte =
                u8::from_str_radix(hex, 16).map_err(|_| Error::msg("invalid inline encoding"))?;
            out.push(byte);
            i += 3;
        } else {
            out.push(bytes[i]);
            i += 1;
        }
    }
    Ok(out)
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
    if data.len() < 2 || data[0] != 0x1f || data[1] != 0x8b {
        return Err(Error::msg("invalid compressed inline data"));
    }
    let mut decoder = flate2::read::GzDecoder::new(data);
    let mut decoded = Vec::new();
    let mut bounded = std::io::Read::take(&mut decoder, INLINE_GZIP_LIMIT + 1);
    match std::io::Read::read_to_end(&mut bounded, &mut decoded) {
        Ok(_) if decoded.len() as u64 <= INLINE_GZIP_LIMIT => Ok(decoded),
        _ => Err(Error::msg("invalid or oversized compressed inline data")),
    }
}

/// Decode one file's inline bytes: plain or bounded gzip. Mirrors
/// `inlineData`, including raw error passthrough.
pub fn inline_data(source: &str, compression: &str) -> Result<Vec<u8>, Error> {
    let data = decode_data_uri(source)?;
    if compression.is_empty() {
        return Ok(data);
    }
    if compression != "gzip" {
        return Err(Error::msg("unsupported inline compression"));
    }
    gunzip_bounded(&data)
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

/// Decode the storage files of an Ignition document, ignoring unknown
/// fields like the Go owner does. Present-but-mistyped shapes fail like
/// Go's `Unmarshal` type errors; callers map them to their own message.
pub fn decode_ignition_files(value: &JsonValue) -> Result<Vec<IgnitionFile>, Error> {
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
        let field = |name: &str| -> Result<String, Error> {
            match contents.and_then(|c| c.get(name)) {
                None | Some(JsonValue::Null) => Ok(String::new()),
                Some(JsonValue::Str(s)) => Ok(s.clone()),
                Some(_) => Err(Error::msg(format!("invalid {name}: string required"))),
            }
        };
        files.push(IgnitionFile {
            path: crate::jsonio::opt_string(item, "path")?,
            source: field("source")?,
            compression: field("compression")?,
        });
    }
    Ok(files)
}
