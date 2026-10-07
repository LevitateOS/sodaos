//! JSON wire compatibility with the Go host daemon.
//!
//! Port of `internal/strictjson` (request decode) plus the output encoding
//! rules of `encoding/json` used for daemon responses and the
//! `org.soda.creation-profile` container label:
//!
//! * requests: exactly one top-level object, valid UTF-8, at most 1 MiB,
//!   duplicate fields rejected at every nesting level, unknown fields
//!   rejected by the typed decoders;
//! * output: struct field order, `omitempty` elision, HTML escaping
//!   (`<`, `>`, `&`), `\u2028`/`\u2029` escaping, trailing newline for the
//!   HTTP encoder only.

use std::collections::HashMap;
use std::collections::{BTreeMap, HashSet};

use serde::de::{self, DeserializeOwned, DeserializeSeed, MapAccess, SeqAccess, Visitor};
use serde::{Deserialize, Deserializer};

mod bind;
mod number;
mod scan;
mod specs;
mod string;

#[cfg(test)]
mod binding_tests;
#[cfg(test)]
mod strict_tests;

pub use self::bind::{bind_root, bind_struct};
// `muse_serve_oracle` compiles this root through a private `#[path]` copy
// that never touches some re-exported names; they serve the real library.
#[allow(unused_imports)]
pub use self::number::{parse_go_int64, parse_go_uint32};
#[allow(unused_imports)]
pub use self::specs::{Bound, BoundMap, Kind, Spec};

pub const MAXIMUM_REQUEST_BYTES: usize = 1 << 20;

/// Strictly admit one host request object, then decode directly into its
/// owner DTO. Root names are sorted before typed decoding to preserve the
/// host's established root alias precedence; nested raw values retain their
/// source order.
pub fn decode_strict_as<T: DeserializeOwned>(body: &[u8]) -> Result<T, Error> {
    if body.len() > MAXIMUM_REQUEST_BYTES {
        return Err(err("request exceeds 1 MiB"));
    }
    std::str::from_utf8(body).map_err(|_| err("request must contain valid UTF-8"))?;

    struct Root(BTreeMap<String, Box<serde_json::value::RawValue>>);
    impl<'de> Deserialize<'de> for Root {
        fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
        where
            D: Deserializer<'de>,
        {
            struct RootVisitor;
            impl<'de> Visitor<'de> for RootVisitor {
                type Value = Root;
                fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                    f.write_str("a JSON object")
                }
                fn visit_map<A>(self, mut map: A) -> Result<Root, A::Error>
                where
                    A: MapAccess<'de>,
                {
                    let mut members = BTreeMap::new();
                    while let Some(key) = map.next_key::<String>()? {
                        if members.contains_key(&key) {
                            return Err(de::Error::custom("duplicate object key"));
                        }
                        let value = map.next_value::<Box<serde_json::value::RawValue>>()?;
                        members.insert(key, value);
                    }
                    Ok(Root(members))
                }
            }
            deserializer.deserialize_map(RootVisitor)
        }
    }

    struct UniqueSeed(usize);
    impl<'de> DeserializeSeed<'de> for UniqueSeed {
        type Value = ();
        fn deserialize<D>(self, deserializer: D) -> Result<(), D::Error>
        where
            D: Deserializer<'de>,
        {
            if self.0 > 100 {
                return Err(de::Error::custom("maximum host JSON depth exceeded"));
            }
            deserializer.deserialize_any(UniqueVisitor(self.0))
        }
    }
    struct UniqueVisitor(usize);
    impl<'de> Visitor<'de> for UniqueVisitor {
        type Value = ();
        fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
            f.write_str("a JSON value")
        }
        fn visit_bool<E>(self, _: bool) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_i64<E>(self, _: i64) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_u64<E>(self, _: u64) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_f64<E>(self, _: f64) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_str<E>(self, _: &str) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_borrowed_str<E>(self, _: &'de str) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_string<E>(self, _: String) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_unit<E>(self) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_none<E>(self) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_seq<A>(self, mut seq: A) -> Result<(), A::Error>
        where
            A: SeqAccess<'de>,
        {
            while seq.next_element_seed(UniqueSeed(self.0 + 1))?.is_some() {}
            Ok(())
        }
        fn visit_map<A>(self, mut map: A) -> Result<(), A::Error>
        where
            A: MapAccess<'de>,
        {
            let mut names = HashSet::new();
            while let Some(key) = map.next_key::<String>()? {
                if !names.insert(key) {
                    return Err(de::Error::custom("duplicate object key"));
                }
                map.next_value_seed(UniqueSeed(self.0 + 1))?;
            }
            Ok(())
        }
    }

    let mut deserializer = serde_json::Deserializer::from_slice(body);
    let Root(root) = Root::deserialize(&mut deserializer).map_err(|e| err(e.to_string()))?;
    deserializer.end().map_err(|e| err(e.to_string()))?;
    for raw in root.values() {
        let mut value = serde_json::Deserializer::from_str(raw.get());
        UniqueSeed(0)
            .deserialize(&mut value)
            .map_err(|e| err(e.to_string()))?;
        value.end().map_err(|e| err(e.to_string()))?;
    }
    let ordered = serde_json::to_vec(&root).map_err(|e| err(e.to_string()))?;
    let mut input = serde_json::Deserializer::from_slice(&ordered);
    let value = T::deserialize(&mut input).map_err(|e| err(e.to_string()))?;
    input.end().map_err(|e| err(e.to_string()))?;
    Ok(value)
}

/// Decode one complete machine response into its owner DTO. Unknown-field
/// behavior is chosen by that DTO's Serde implementation.
pub fn decode_tolerant_as<T: DeserializeOwned>(body: &[u8]) -> Result<T, Error> {
    let mut input = serde_json::Deserializer::from_slice(body);
    let value = T::deserialize(&mut input).map_err(|e| err(e.to_string()))?;
    input.end().map_err(|e| err(e.to_string()))?;
    Ok(value)
}

/// Nesting depth limit matching strictjson's per-level duplicate scan: a
/// value nested more than 101 levels below the top-level object is refused.
#[derive(Debug, Clone, PartialEq)]
pub enum Value {
    Null,
    Bool(bool),
    Number(String),
    Str(String),
    Array(Vec<Value>),
    Object(Vec<(String, Value)>),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Error(pub String);

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.0)
    }
}

impl std::error::Error for Error {}

fn err(msg: impl Into<String>) -> Error {
    Error(msg.into())
}

struct Parser<'a> {
    bytes: &'a [u8],
    pos: usize,
    /// None = tolerant (podman output): last duplicate wins. Some = strict
    /// duplicate rejection with the top-level field name for messages.
    strict_field: Option<Option<String>>,
    /// First nested duplicate/depth finding in walk order, flushed when the
    /// enclosing top-level value completes (strict only).
    deferred: Option<Error>,
}

enum Frame {
    Object {
        fields: Vec<(String, Value)>,
        pending: Option<String>,
    },
    Array {
        items: Vec<Value>,
    },
}

fn close_frame(frame: Frame) -> Value {
    match frame {
        Frame::Object { fields, .. } => Value::Object(fields),
        Frame::Array { items } => Value::Array(items),
    }
}

/// Strict request decode: exactly one JSON object, at most 1 MiB, valid
/// UTF-8, no duplicate fields at any depth. Mirrors `strictjson.Decode`
/// up to the typed struct binding, which each DTO decoder performs with
/// `Object::get_*` so unknown fields are rejected there.
pub fn decode_strict(body: &[u8]) -> Result<Value, Error> {
    if body.len() > MAXIMUM_REQUEST_BYTES {
        return Err(err("request exceeds 1 MiB"));
    }
    let text = std::str::from_utf8(body).map_err(|_| err("request must contain valid UTF-8"))?;
    let mut p = Parser::new(text.as_bytes(), true);
    let mut v = p.parse_object(0)?;
    p.skip_ws();
    if !p.eof() {
        // Trailing `Token`: a second complete token is excess data, but a
        // token scan error surfaces instead.
        match p.peek() {
            Some(b'{') | Some(b'[') => {
                return Err(err("request must contain exactly one JSON object"));
            }
            Some(b'"') => {
                p.parse_string()?;
                return Err(err("request must contain exactly one JSON object"));
            }
            Some(b't') => {
                p.parse_literal("true", Value::Bool(true))?;
                return Err(err("request must contain exactly one JSON object"));
            }
            Some(b'f') => {
                p.parse_literal("false", Value::Bool(false))?;
                return Err(err("request must contain exactly one JSON object"));
            }
            Some(b'n') => {
                p.parse_literal("null", Value::Null)?;
                return Err(err("request must contain exactly one JSON object"));
            }
            Some(b'-') | Some(b'0'..=b'9') => {
                let lit = p.parse_number()?;
                // `finishObject` takes one trailing `Token`, which converts
                // the number: overflow fails before the excess-data verdict.
                if !lit.parse::<f64>().is_ok_and(|v| v.is_finite()) {
                    return Err(err(format!(
                        "decode request: json: cannot unmarshal number {lit} into Go value of type float64"
                    )));
                }
                return Err(err("request must contain exactly one JSON object"));
            }
            Some(c) => {
                return Err(err(format!(
                    "decode request: invalid character {} looking for beginning of value",
                    Parser::quote_byte(c)
                )));
            }
            None => {}
        }
    }
    // strictjson re-marshals the top-level map before struct binding, which
    // sorts top-level keys; nested raw values keep document order. The
    // binding driver relies on this for exact first-error ordering.
    if let Value::Object(fields) = &mut v {
        fields.sort_by(|a, b| a.0.cmp(&b.0));
    }
    Ok(v)
}

/// Tolerant decode for machine-generated podman output: mirrors
/// `encoding/json.Unmarshal` (unknown fields ignored, last duplicate wins).
pub fn decode_tolerant(body: &[u8]) -> Result<Value, Error> {
    let text = std::str::from_utf8(body).map_err(|_| err("invalid UTF-8"))?;
    let mut p = Parser::new(text.as_bytes(), false);
    let v = p.parse_value(0)?;
    p.skip_ws();
    if !p.eof() {
        return Err(err("trailing data"));
    }
    Ok(v)
}

impl Value {
    pub fn as_object(&self) -> Option<&Vec<(String, Value)>> {
        match self {
            Value::Object(fields) => Some(fields),
            _ => None,
        }
    }

    pub fn as_array(&self) -> Option<&Vec<Value>> {
        match self {
            Value::Array(items) => Some(items),
            _ => None,
        }
    }

    pub fn as_str(&self) -> Option<&str> {
        match self {
            Value::Str(s) => Some(s),
            _ => None,
        }
    }

    pub fn as_bool(&self) -> Option<bool> {
        match self {
            Value::Bool(b) => Some(*b),
            _ => None,
        }
    }

    /// Go `encoding/json` number-to-int64 binding: the raw literal must
    /// parse as a base-10 64-bit integer (`1e3`, `1.5` are rejected).
    pub fn as_i64(&self) -> Option<i64> {
        match self {
            Value::Number(lit) => lit.parse::<i64>().ok(),
            _ => None,
        }
    }

    pub fn is_null(&self) -> bool {
        matches!(self, Value::Null)
    }
}

/// `strconv.Quote` for ASCII input: exact for all ASCII bytes, UTF-8 passed
/// through. Used for `unknown field` names (ASCII in practice).
pub(crate) fn go_quote(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    out.push('"');
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\u{07}' => out.push_str("\\a"),
            '\u{08}' => out.push_str("\\b"),
            '\u{0C}' => out.push_str("\\f"),
            '\u{0B}' => out.push_str("\\v"),
            c if (c as u32) < 0x20 || (c as u32) == 0x7f => {
                out.push_str(&format!("\\x{:02x}", c as u32));
            }
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

pub fn tolerant_get<'a>(v: &'a Value, name: &str) -> Option<&'a Value> {
    let fields = v.as_object()?;
    if let Some((_, v)) = fields.iter().find(|(k, _)| k == name) {
        return if v.is_null() { None } else { Some(v) };
    }
    let mut found = None;
    for (k, v) in fields {
        if k.eq_ignore_ascii_case(name) {
            if found.is_some() {
                return None;
            }
            found = Some(v);
        }
    }
    found.filter(|v| !v.is_null())
}

pub fn tolerant_string_map(v: &Value) -> HashMap<String, String> {
    let mut out = HashMap::new();
    if let Some(fields) = v.as_object() {
        for (k, v) in fields {
            if let Value::Str(s) = v {
                out.insert(k.clone(), s.clone());
            }
        }
    }
    out
}

// ---------- Go-compatible output encoding ----------

/// Escape a string exactly like `encoding/json`: short escapes, HTML
/// escaping for `<`, `>` and `&`, `\u2028`/`\u2029` escaping, and
/// `\u00xx` for other C0 controls. Lowercase hex, like Go.
pub fn escape_into(out: &mut String, s: &str) {
    let quoted = quote(s);
    out.push_str(&quoted[1..quoted.len() - 1]);
}

pub fn quote(s: &str) -> String {
    use serde::Serialize;
    let mut out = Vec::with_capacity(s.len() + 2);
    let mut serializer = serde_json::Serializer::with_formatter(&mut out, GoFormatter);
    s.serialize(&mut serializer)
        .expect("writing to Vec cannot fail");
    String::from_utf8(out).expect("JSON string is UTF-8")
}

/// Go `encoding/json`'s string policy layered onto serde_json's compact
/// formatter. Controls are escaped by the upstream formatter; this method
/// adds Go's HTML and JavaScript-separator escapes.
struct GoFormatter;

impl serde_json::ser::Formatter for GoFormatter {
    fn write_string_fragment<W>(&mut self, writer: &mut W, fragment: &str) -> std::io::Result<()>
    where
        W: ?Sized + std::io::Write,
    {
        let mut start = 0;
        for (index, ch) in fragment.char_indices() {
            let escaped = match ch {
                '<' => Some(b"\\u003c".as_slice()),
                '>' => Some(b"\\u003e".as_slice()),
                '&' => Some(b"\\u0026".as_slice()),
                '\u{2028}' => Some(b"\\u2028".as_slice()),
                '\u{2029}' => Some(b"\\u2029".as_slice()),
                _ => None,
            };
            if let Some(escaped) = escaped {
                writer.write_all(&fragment.as_bytes()[start..index])?;
                writer.write_all(escaped)?;
                start = index + ch.len_utf8();
            }
        }
        writer.write_all(&fragment.as_bytes()[start..])
    }
}
