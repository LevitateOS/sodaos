//! Go-exact JSON decode/emit shared by the port.
//!
//! `decode_strict` mirrors `strictjson.Decode`: exactly one JSON object,
//! at most 1 MiB of valid UTF-8, duplicate keys rejected at every level,
//! unknown fields rejected at every level by the typed binders below.
//! `decode_lenient` mirrors `encoding/json.Unmarshal` for the few
//! non-strict sites (media bindings, tool locks, registry observations).
//! `Emitter` mirrors `json.MarshalIndent(v, "", "  ")` plus trailing `\n`.

pub const MAX_STRICT_BYTES: usize = 1 << 20;

/// Untyped JSON shape failure; callers map it to the owner's error text.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DecodeError;

/// Standard base64 (Go `encoding/base64.StdEncoding`), used for `[]byte`
/// fields. Implemented locally so padding and alphabet stay exact.
pub fn base64_encode(data: &[u8]) -> String {
    const ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity(data.len().div_ceil(3) * 4);
    for chunk in data.chunks(3) {
        let mut n: u32 = 0;
        for (i, &b) in chunk.iter().enumerate() {
            n |= (b as u32) << (16 - 8 * i);
        }
        out.push(ALPHABET[(n >> 18) as usize & 63] as char);
        out.push(ALPHABET[(n >> 12) as usize & 63] as char);
        if chunk.len() > 1 {
            out.push(ALPHABET[(n >> 6) as usize & 63] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(ALPHABET[n as usize & 63] as char);
        } else {
            out.push('=');
        }
    }
    out
}

pub fn base64_decode(text: &str) -> Result<Vec<u8>, DecodeError> {
    if !text.len().is_multiple_of(4) {
        return Err(DecodeError);
    }
    let value = |c: u8| -> Result<u32, DecodeError> {
        match c {
            b'A'..=b'Z' => Ok((c - b'A') as u32),
            b'a'..=b'z' => Ok((c - b'a' + 26) as u32),
            b'0'..=b'9' => Ok((c - b'0' + 52) as u32),
            b'+' => Ok(62),
            b'/' => Ok(63),
            _ => Err(DecodeError),
        }
    };
    let bytes = text.as_bytes();
    let mut out = Vec::with_capacity(bytes.len() / 4 * 3);
    for chunk in bytes.chunks(4) {
        let mut n: u32 = 0;
        let mut pad = 0;
        for (i, &c) in chunk.iter().enumerate() {
            if c == b'=' {
                if i < 2 {
                    return Err(DecodeError);
                }
                pad += 1;
                n <<= 6;
            } else {
                if pad > 0 {
                    return Err(DecodeError);
                }
                n = (n << 6) | value(c)?;
            }
        }
        if pad > 2 {
            return Err(DecodeError);
        }
        out.push((n >> 16) as u8);
        if pad < 2 {
            out.push((n >> 8) as u8);
        }
        if pad < 1 {
            out.push(n as u8);
        }
    }
    // Reject non-canonical trailing bits the way Go's decoder does.
    if base64_encode(&out) != text {
        return Err(DecodeError);
    }
    Ok(out)
}

mod decode;
pub use decode::{dedupe_last_wins, parse_lenient, parse_strict, Binder, Soft};

mod emit;
pub use emit::{Emit, Emitter};

/// Go `int`/`int64` field range check.
pub fn as_i64(value: i128) -> Result<i64, DecodeError> {
    i64::try_from(value).map_err(|_| DecodeError)
}

/// Go `uint64` field range check.
pub fn as_u64(value: i128) -> Result<u64, DecodeError> {
    u64::try_from(value).map_err(|_| DecodeError)
}

pub fn marshal<T: Emit + ?Sized>(value: &T) -> Vec<u8> {
    let mut e = Emitter::new();
    value.emit(&mut e);
    e.finish().into_bytes()
}

#[cfg(test)]
mod tests;
