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

/// Standard padded base64, used for `[]byte` fields.
pub fn base64_encode(data: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD.encode(data)
}

pub fn base64_decode(text: &str) -> Result<Vec<u8>, DecodeError> {
    let out = base64::engine::general_purpose::STANDARD
        .decode(text)
        .map_err(|_| DecodeError)?;
    // Retain the caller's canonical representation gate as part of the
    // release JSON byte contract.
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
use base64::Engine;
