//! Go-exact JSON decode/emit shared by the port.
//!
//! `decode_strict` mirrors `strictjson.Decode`: exactly one JSON object,
//! at most 1 MiB of valid UTF-8, duplicate keys rejected at every level,
//! unknown fields rejected at every level by the typed binders below.
//! `decode_lenient` mirrors `encoding/json.Unmarshal` for the few
//! non-strict sites (media bindings, tool locks, registry observations).
//! `Emitter` mirrors `json.MarshalIndent(v, "", "  ")` plus trailing `\n`.

use soda_json::{escape_into, JsonValue};

use crate::Error;

pub const MAX_STRICT_BYTES: usize = 1 << 20;

/// Untyped JSON shape failure; callers map it to the owner's error text.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DecodeError;

/// Standard base64 (Go `encoding/base64.StdEncoding`), used for `[]byte`
/// fields. Implemented locally so padding and alphabet stay exact.
pub fn base64_encode(data: &[u8]) -> String {
    const ALPHABET: &[u8; 64] =
        b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
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
    if text.len() % 4 != 0 {
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

fn reject_duplicates(value: &JsonValue, depth: u32) -> Result<(), DecodeError> {
    if depth > 100 {
        return Err(DecodeError);
    }
    match value {
        JsonValue::Object(entries) => {
            for i in 0..entries.len() {
                for j in 0..i {
                    if entries[i].0 == entries[j].0 {
                        return Err(DecodeError);
                    }
                }
                reject_duplicates(&entries[i].1, depth + 1)?;
            }
            Ok(())
        }
        JsonValue::Array(items) => {
            for item in items {
                reject_duplicates(item, depth + 1)?;
            }
            Ok(())
        }
        _ => Ok(()),
    }
}

/// Strict object binder: every field must be consumed exactly once;
/// `finish` rejects unknown fields.
pub struct Binder<'a> {
    entries: &'a [(String, JsonValue)],
    seen: Vec<bool>,
}

impl<'a> Binder<'a> {
    pub fn new(value: &'a JsonValue) -> Result<Binder<'a>, DecodeError> {
        match value {
            JsonValue::Object(entries) => Ok(Binder {
                entries,
                seen: vec![false; entries.len()],
            }),
            _ => Err(DecodeError),
        }
    }

    fn find(&mut self, name: &str) -> Result<Option<&'a JsonValue>, DecodeError> {
        let mut found = None;
        for (i, (key, value)) in self.entries.iter().enumerate() {
            if key == name {
                if found.is_some() {
                    return Err(DecodeError);
                }
                self.seen[i] = true;
                found = Some(value);
            }
        }
        Ok(found)
    }

    fn optional(&mut self, name: &str) -> Result<Option<&'a JsonValue>, DecodeError> {
        match self.find(name)? {
            None | Some(JsonValue::Null) => Ok(None),
            Some(value) => Ok(Some(value)),
        }
    }

    pub fn string(&mut self, name: &str) -> Result<Option<String>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(JsonValue::Str(s)) => Ok(Some(s.clone())),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn boolean(&mut self, name: &str) -> Result<Option<bool>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(JsonValue::Bool(b)) => Ok(Some(*b)),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn integer(&mut self, name: &str) -> Result<Option<i128>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(v) => v.as_integer().map(Some).ok_or(DecodeError),
        }
    }

    pub fn object(&mut self, name: &str) -> Result<Option<Binder<'a>>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(v) => Ok(Some(Binder::new(v)?)),
        }
    }

    pub fn array(&mut self, name: &str) -> Result<Option<&'a [JsonValue]>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(JsonValue::Array(items)) => Ok(Some(items)),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn bytes(&mut self, name: &str) -> Result<Option<Vec<u8>>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(JsonValue::Str(s)) => Ok(Some(base64_decode(s)?)),
            Some(_) => Err(DecodeError),
        }
    }

    /// Raw value of a field, kept verbatim (null included).
    pub fn raw(&mut self, name: &str) -> Result<Option<&'a JsonValue>, DecodeError> {
        self.find(name)
    }

    /// Raw entries of a nested object for strict map decoding.
    pub fn entries(&mut self, name: &str) -> Result<Option<&'a [(String, JsonValue)]>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(JsonValue::Object(entries)) => Ok(Some(entries)),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn finish(self) -> Result<(), DecodeError> {
        self.finish_name().map_err(|_| DecodeError)
    }

    /// Unknown-field name in document order, for Go `encoding/json`
    /// error text at the lenient `build.ReadJSON` sites.
    pub fn finish_name(self) -> Result<(), String> {
        for (i, (key, _)) in self.entries.iter().enumerate() {
            if !self.seen[i] {
                return Err(format!("json: unknown field \"{key}\""));
            }
        }
        Ok(())
    }
}

/// Lenient value reader mirroring `encoding/json.Unmarshal` into the small
/// anonymous shapes the Go side decodes non-strictly. Unknown fields are
/// ignored; duplicate keys keep the last value.
pub struct Soft<'a> {
    value: &'a JsonValue,
}

impl<'a> Soft<'a> {
    pub fn new(value: &'a JsonValue) -> Result<Soft<'a>, DecodeError> {
        match value {
            JsonValue::Object(_) => Ok(Soft { value }),
            _ => Err(DecodeError),
        }
    }

    pub fn field(&self, name: &str) -> Option<&'a JsonValue> {
        match self.value.get(name) {
            None | Some(JsonValue::Null) => None,
            Some(value) => Some(value),
        }
    }

    pub fn string(&self, name: &str) -> Result<Option<String>, DecodeError> {
        match self.field(name) {
            None => Ok(None),
            Some(JsonValue::Str(s)) => Ok(Some(s.clone())),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn integer(&self, name: &str) -> Result<Option<i128>, DecodeError> {
        match self.field(name) {
            None => Ok(None),
            Some(v) => v.as_integer().map(Some).ok_or(DecodeError),
        }
    }

    pub fn boolean(&self, name: &str) -> Result<Option<bool>, DecodeError> {
        match self.field(name) {
            None => Ok(None),
            Some(JsonValue::Bool(b)) => Ok(Some(*b)),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn object(&self, name: &str) -> Result<Option<Soft<'a>>, DecodeError> {
        match self.field(name) {
            None => Ok(None),
            Some(v @ JsonValue::Object(_)) => Ok(Some(Soft { value: v })),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn array(&self, name: &str) -> Result<Option<&'a [JsonValue]>, DecodeError> {
        match self.field(name) {
            None => Ok(None),
            Some(JsonValue::Array(items)) => Ok(Some(items)),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn entries(&self) -> Option<&'a [(String, JsonValue)]> {
        match self.value {
            JsonValue::Object(entries) => Some(entries),
            _ => None,
        }
    }
}

pub fn parse_strict(data: &[u8]) -> Result<JsonValue, Error> {
    if data.len() > MAX_STRICT_BYTES {
        return Err(Error::refused());
    }
    let text = std::str::from_utf8(data).map_err(|_| Error::refused())?;
    let value = JsonValue::parse(text).map_err(|_| Error::refused())?;
    if !value.is_object() {
        return Err(Error::refused());
    }
    reject_duplicates(&value, 0).map_err(|_| Error::refused())?;
    Ok(value)
}

pub fn parse_lenient(data: &[u8]) -> Result<JsonValue, DecodeError> {
    let text = std::str::from_utf8(data).map_err(|_| DecodeError)?;
    JsonValue::parse(text).map_err(|_| DecodeError)
}

/// Last-wins deduplication mirroring `encoding/json` object semantics at
/// the `build.ReadJSON` sites (unlike `strictjson`, duplicates are kept).
pub fn dedupe_last_wins(value: JsonValue) -> JsonValue {
    match value {
        JsonValue::Object(entries) => {
            let mut kept: Vec<(String, JsonValue)> = Vec::with_capacity(entries.len());
            for (key, val) in entries {
                let val = dedupe_last_wins(val);
                if let Some(slot) = kept.iter_mut().find(|(k, _)| *k == key) {
                    slot.1 = val;
                } else {
                    kept.push((key, val));
                }
            }
            JsonValue::Object(kept)
        }
        JsonValue::Array(items) => {
            JsonValue::Array(items.into_iter().map(dedupe_last_wins).collect())
        }
        scalar => scalar,
    }
}

/// Indented emitter mirroring `json.MarshalIndent(v, "", "  ")`.
pub struct Emitter {
    out: String,
    level: usize,
}

impl Emitter {
    pub fn new() -> Emitter {
        Emitter {
            out: String::new(),
            level: 0,
        }
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
mod tests {
    use super::*;

    #[test]
    fn base64_round_trip_matches_go_vectors() {
        assert_eq!(base64_encode(b""), "");
        assert_eq!(base64_encode(b"f"), "Zg==");
        assert_eq!(base64_encode(b"fo"), "Zm8=");
        assert_eq!(base64_encode(b"foo"), "Zm9v");
        assert_eq!(base64_encode(b"foob"), "Zm9vYg==");
        assert_eq!(base64_encode(b"fooba"), "Zm9vYmE=");
        assert_eq!(base64_encode(b"foobar"), "Zm9vYmFy");
        for raw in [b"".as_slice(), b"f", b"fo", b"foo", b"record-bytes-12345"] {
            assert_eq!(base64_decode(&base64_encode(raw)).unwrap(), raw);
        }
        assert!(base64_decode("Zg=").is_err());
        assert!(base64_decode("Zg*=").is_err());
        // Non-canonical trailing bits rejected like Go's decoder.
        assert!(base64_decode("Zh==").is_err());
    }

    #[test]
    fn strict_decode_rejects_duplicates_and_unknowns() {
        let value = parse_strict(br#"{"A": 1, "B": [1, 2]}"#).unwrap();
        let mut binder = Binder::new(&value).unwrap();
        assert_eq!(binder.integer("A").unwrap(), Some(1));
        assert_eq!(binder.array("B").unwrap().unwrap().len(), 2);
        assert!(binder.finish().is_ok());
        assert!(parse_strict(br#"{"A": 1, "A": 2}"#).is_err());
        assert!(parse_strict(br#"{"A": {"B": 1, "B": 2}}"#).is_err());
        assert!(parse_strict(br#"[1, 2]"#).is_err());
        let value = parse_strict(br#"{"A": 1, "Extra": true}"#).unwrap();
        let mut binder = Binder::new(&value).unwrap();
        assert_eq!(binder.integer("A").unwrap(), Some(1));
        assert!(binder.finish().is_err());
        // Go type rules: null means absent, wrong types refuse.
        let value = parse_strict(br#"{"A": null, "B": "x"}"#).unwrap();
        let mut binder = Binder::new(&value).unwrap();
        assert_eq!(binder.integer("A").unwrap(), None);
        assert_eq!(binder.string("B").unwrap(), Some("x".to_string()));
        assert!(binder.finish().is_ok());
        let value = parse_strict(br#"{"A": "1"}"#).unwrap();
        let mut binder = Binder::new(&value).unwrap();
        assert!(binder.integer("A").is_err());
        let value = parse_strict(br#"{"A": 1.5}"#).unwrap();
        let mut binder = Binder::new(&value).unwrap();
        assert!(binder.integer("A").is_err());
    }
}
