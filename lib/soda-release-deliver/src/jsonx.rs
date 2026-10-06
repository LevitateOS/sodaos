//! Go-exact JSON decode/emit shared by the port.
//!
//! `decode_strict` mirrors `strictjson.Decode`: exactly one JSON object,
//! at most 1 MiB of valid UTF-8, duplicate keys rejected at every level,
//! unknown fields rejected at every level by the typed binders below.
//! `decode_lenient` mirrors `encoding/json.Unmarshal` for the few
//! non-strict sites (media bindings, tool locks, registry observations).
//! `Emitter` mirrors `json.MarshalIndent(v, "", "  ")` plus trailing `\n`.

use soda_json::{escape_into, JsonValue};

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

/// Indented emitter mirroring `json.MarshalIndent(v, "", "  ")`.
#[derive(Default)]
pub struct Emitter {
    out: String,
    level: usize,
}

impl Emitter {
    pub fn new() -> Emitter {
        Emitter::default()
    }

    pub fn finish(mut self) -> String {
        self.out.push('\n');
        self.out
    }

    fn indent(&mut self) {
        for _ in 0..self.level {
            self.out.push_str("  ");
        }
    }

    pub fn null(&mut self) {
        self.out.push_str("null");
    }

    pub fn boolean(&mut self, value: bool) {
        self.out.push_str(if value { "true" } else { "false" });
    }

    pub fn int(&mut self, value: i64) {
        self.out.push_str(&value.to_string());
    }

    pub fn uint(&mut self, value: u64) {
        self.out.push_str(&value.to_string());
    }

    pub fn string(&mut self, value: &str) {
        escape_into(&mut self.out, value);
    }

    pub fn bytes(&mut self, value: &[u8]) {
        escape_into(&mut self.out, &base64_encode(value));
    }

    pub fn begin_object(&mut self, empty: bool) {
        if empty {
            self.out.push_str("{}");
        } else {
            self.out.push_str("{\n");
            self.level += 1;
        }
    }

    pub fn field(&mut self, first: bool, name: &str) {
        if !first {
            self.out.push_str(",\n");
        }
        self.indent();
        escape_into(&mut self.out, name);
        self.out.push_str(": ");
    }

    pub fn end_object(&mut self, empty: bool) {
        if !empty {
            self.out.push('\n');
            self.level -= 1;
            self.indent();
            self.out.push('}');
        }
    }

    pub fn begin_array(&mut self, empty: bool) {
        if empty {
            self.out.push_str("[]");
        } else {
            self.out.push_str("[\n");
            self.level += 1;
        }
    }

    pub fn item(&mut self, first: bool) {
        if !first {
            self.out.push_str(",\n");
        }
        self.indent();
    }

    pub fn end_array(&mut self, empty: bool) {
        if !empty {
            self.out.push('\n');
            self.level -= 1;
            self.indent();
            self.out.push(']');
        }
    }
}

/// Values the port marshals with Go field order and shape.
pub trait Emit {
    fn emit(&self, e: &mut Emitter);
}

impl Emit for String {
    fn emit(&self, e: &mut Emitter) {
        e.string(self);
    }
}

impl Emit for str {
    fn emit(&self, e: &mut Emitter) {
        e.string(self);
    }
}

impl Emit for bool {
    fn emit(&self, e: &mut Emitter) {
        e.boolean(*self);
    }
}

impl Emit for i64 {
    fn emit(&self, e: &mut Emitter) {
        e.int(*self);
    }
}

impl Emit for u64 {
    fn emit(&self, e: &mut Emitter) {
        e.uint(*self);
    }
}

impl Emit for i32 {
    fn emit(&self, e: &mut Emitter) {
        e.int(*self as i64);
    }
}

impl Emit for Vec<String> {
    fn emit(&self, e: &mut Emitter) {
        e.begin_array(self.is_empty());
        for (i, item) in self.iter().enumerate() {
            e.item(i == 0);
            e.string(item);
        }
        e.end_array(self.is_empty());
    }
}

impl Emit for std::collections::BTreeMap<String, String> {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(self.is_empty());
        for (i, (key, value)) in self.iter().enumerate() {
            e.field(i == 0, key);
            e.string(value);
        }
        e.end_object(self.is_empty());
    }
}

impl Emit for std::collections::BTreeMap<String, u64> {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(self.is_empty());
        for (i, (key, value)) in self.iter().enumerate() {
            e.field(i == 0, key);
            e.uint(*value);
        }
        e.end_object(self.is_empty());
    }
}

/// Indented emission of a dynamic value. Object entries keep document
/// order: callers modeling Go structs build struct order, callers modeling
/// Go maps pre-sort keys. Numbers are emitted verbatim.
impl Emit for JsonValue {
    fn emit(&self, e: &mut Emitter) {
        match self {
            JsonValue::Null => e.null(),
            JsonValue::Bool(b) => e.boolean(*b),
            JsonValue::Number(raw) => {
                // Numbers in this port are always canonical literals.
                e.out.push_str(raw);
            }
            JsonValue::Str(s) => e.string(s),
            JsonValue::Array(items) => {
                e.begin_array(items.is_empty());
                for (i, item) in items.iter().enumerate() {
                    e.item(i == 0);
                    item.emit(e);
                }
                e.end_array(items.is_empty());
            }
            JsonValue::Object(entries) => {
                e.begin_object(entries.is_empty());
                for (i, (key, value)) in entries.iter().enumerate() {
                    e.field(i == 0, key);
                    value.emit(e);
                }
                e.end_object(entries.is_empty());
            }
        }
    }
}

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
