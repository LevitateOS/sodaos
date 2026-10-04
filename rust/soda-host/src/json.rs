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

pub const MAXIMUM_REQUEST_BYTES: usize = 1 << 20;

/// Nesting depth limit matching strictjson's per-level duplicate scan: a
/// value nested more than 101 levels below the top-level object is refused.
const MAX_NESTING: u32 = 101;

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
}

impl<'a> Parser<'a> {
    fn new(bytes: &'a [u8], strict: bool) -> Self {
        Parser {
            bytes,
            pos: 0,
            strict_field: strict.then_some(None),
        }
    }

    fn eof(&self) -> bool {
        self.pos >= self.bytes.len()
    }

    fn peek(&self) -> Option<u8> {
        self.bytes.get(self.pos).copied()
    }

    fn skip_ws(&mut self) {
        while matches!(self.peek(), Some(b' ' | b'\t' | b'\n' | b'\r')) {
            self.pos += 1;
        }
    }

    fn expect(&mut self, b: u8) -> Result<(), Error> {
        if self.peek() == Some(b) {
            self.pos += 1;
            Ok(())
        } else {
            Err(err("decode request: expected more input"))
        }
    }

    fn strict(&self) -> bool {
        self.strict_field.is_some()
    }

    fn field_ctx(&self) -> String {
        self.strict_field
            .as_ref()
            .and_then(|f| f.clone())
            .map(|f| format!("decode request field {f:?}: "))
            .unwrap_or_else(|| "decode request: ".to_string())
    }

    fn parse_value(&mut self, depth: u32) -> Result<Value, Error> {
        if depth > MAX_NESTING {
            return Err(err(format!(
                "{}request is nested too deeply",
                self.field_ctx()
            )));
        }
        self.skip_ws();
        match self.peek() {
            Some(b'{') => self.parse_object(depth),
            Some(b'[') => self.parse_array(depth),
            Some(b'"') => Ok(Value::Str(self.parse_string()?)),
            Some(b't') => self.parse_literal("true", Value::Bool(true)),
            Some(b'f') => self.parse_literal("false", Value::Bool(false)),
            Some(b'n') => self.parse_literal("null", Value::Null),
            Some(b'-') | Some(b'0'..=b'9') => Ok(Value::Number(self.parse_number()?)),
            _ => Err(err(format!("{}invalid character", self.field_ctx()))),
        }
    }

    fn parse_literal(&mut self, lit: &str, v: Value) -> Result<Value, Error> {
        if self.bytes[self.pos..].starts_with(lit.as_bytes()) {
            self.pos += lit.len();
            Ok(v)
        } else {
            Err(err(format!("{}invalid character", self.field_ctx())))
        }
    }

    fn parse_object(&mut self, depth: u32) -> Result<Value, Error> {
        self.pos += 1; // {
        let mut fields: Vec<(String, Value)> = Vec::new();
        self.skip_ws();
        if self.peek() == Some(b'}') {
            self.pos += 1;
            return Ok(Value::Object(fields));
        }
        loop {
            self.skip_ws();
            if self.peek() != Some(b'"') {
                return Err(err(format!(
                    "{}request field name must be a string",
                    self.field_ctx()
                )));
            }
            let name = self.parse_string()?;
            if self.strict() && fields.iter().any(|(k, _)| *k == name) {
                return Err(err(format!("duplicate request field {name:?}")));
            }
            self.skip_ws();
            self.expect(b':')?;
            // Track the top-level field name for nested strict messages.
            let saved = self.strict_field.clone();
            if depth == 0 {
                if let Some(slot) = self.strict_field.as_mut() {
                    *slot = Some(name.clone());
                }
            }
            let value = self.parse_value(depth + 1)?;
            self.strict_field = saved;
            if self.strict() {
                fields.push((name, value));
            } else if let Some(slot) = fields.iter_mut().find(|(k, _)| *k == name) {
                slot.1 = value;
            } else {
                fields.push((name, value));
            }
            self.skip_ws();
            match self.peek() {
                Some(b',') => {
                    self.pos += 1;
                }
                Some(b'}') => {
                    self.pos += 1;
                    return Ok(Value::Object(fields));
                }
                _ => return Err(err(format!("{}expected , or }}", self.field_ctx()))),
            }
        }
    }

    fn parse_array(&mut self, depth: u32) -> Result<Value, Error> {
        self.pos += 1; // [
        let mut items = Vec::new();
        self.skip_ws();
        if self.peek() == Some(b']') {
            self.pos += 1;
            return Ok(Value::Array(items));
        }
        loop {
            items.push(self.parse_value(depth + 1)?);
            self.skip_ws();
            match self.peek() {
                Some(b',') => {
                    self.pos += 1;
                }
                Some(b']') => {
                    self.pos += 1;
                    return Ok(Value::Array(items));
                }
                _ => return Err(err(format!("{}expected , or ]", self.field_ctx()))),
            }
        }
    }

    fn hex4(&mut self) -> Result<u32, Error> {
        if self.pos + 4 > self.bytes.len() {
            return Err(err(format!("{}invalid string escape", self.field_ctx())));
        }
        let mut v: u32 = 0;
        for i in 0..4 {
            let c = self.bytes[self.pos + i];
            let d = match c {
                b'0'..=b'9' => (c - b'0') as u32,
                b'a'..=b'f' => (c - b'a' + 10) as u32,
                b'A'..=b'F' => (c - b'A' + 10) as u32,
                _ => return Err(err(format!("{}invalid string escape", self.field_ctx()))),
            };
            v = v * 16 + d;
        }
        self.pos += 4;
        Ok(v)
    }

    fn parse_string(&mut self) -> Result<String, Error> {
        self.pos += 1; // opening quote
        let mut out = String::new();
        loop {
            let c = match self.peek() {
                Some(c) => c,
                None => return Err(err(format!("{}unterminated string", self.field_ctx()))),
            };
            match c {
                b'"' => {
                    self.pos += 1;
                    return Ok(out);
                }
                b'\\' => {
                    self.pos += 1;
                    let e = match self.peek() {
                        Some(e) => e,
                        None => {
                            return Err(err(format!("{}unterminated string", self.field_ctx())))
                        }
                    };
                    self.pos += 1;
                    match e {
                        b'"' => out.push('"'),
                        b'\\' => out.push('\\'),
                        b'/' => out.push('/'),
                        b'b' => out.push('\u{0008}'),
                        b'f' => out.push('\u{000C}'),
                        b'n' => out.push('\n'),
                        b'r' => out.push('\r'),
                        b't' => out.push('\t'),
                        b'u' => {
                            let n = self.hex4()?;
                            if (0xD800..0xDC00).contains(&n) {
                                // High surrogate: must be followed by \uDC00-\uDFFF,
                                // else encoding/json substitutes U+FFFD.
                                if self.bytes.get(self.pos..self.pos + 2) == Some(b"\\u".as_slice())
                                {
                                    let save = self.pos;
                                    self.pos += 2;
                                    let lo = self.hex4()?;
                                    if (0xDC00..0xE000).contains(&lo) {
                                        let c = 0x10000 + ((n - 0xD800) << 10) + (lo - 0xDC00);
                                        out.push(char::from_u32(c).unwrap_or('\u{FFFD}'));
                                    } else {
                                        out.push('\u{FFFD}');
                                        self.pos = save;
                                        // Re-parse the second escape normally below
                                        // by rewinding to its backslash.
                                        continue;
                                    }
                                } else {
                                    out.push('\u{FFFD}');
                                }
                            } else if (0xDC00..0xE000).contains(&n) {
                                out.push('\u{FFFD}');
                            } else {
                                out.push(char::from_u32(n).unwrap_or('\u{FFFD}'));
                            }
                        }
                        _ => return Err(err(format!("{}invalid string escape", self.field_ctx()))),
                    }
                }
                0x00..=0x1F => {
                    return Err(err(format!(
                        "{}invalid character in string",
                        self.field_ctx()
                    )));
                }
                _ => {
                    // Regular UTF-8 scalar (input is already valid UTF-8).
                    let rest = &self.bytes[self.pos..];
                    let s = std::str::from_utf8(rest)
                        .map_err(|_| err(format!("{}invalid UTF-8", self.field_ctx())))?;
                    let ch = s.chars().next().unwrap();
                    out.push(ch);
                    self.pos += ch.len_utf8();
                }
            }
        }
    }

    fn parse_number(&mut self) -> Result<String, Error> {
        let start = self.pos;
        if self.peek() == Some(b'-') {
            self.pos += 1;
        }
        match self.peek() {
            Some(b'0') => {
                self.pos += 1;
            }
            Some(b'1'..=b'9') => {
                while matches!(self.peek(), Some(b'0'..=b'9')) {
                    self.pos += 1;
                }
            }
            _ => return Err(err(format!("{}invalid number", self.field_ctx()))),
        }
        if self.peek() == Some(b'.') {
            self.pos += 1;
            if !matches!(self.peek(), Some(b'0'..=b'9')) {
                return Err(err(format!("{}invalid number", self.field_ctx())));
            }
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
        }
        if matches!(self.peek(), Some(b'e' | b'E')) {
            self.pos += 1;
            if matches!(self.peek(), Some(b'+' | b'-')) {
                self.pos += 1;
            }
            if !matches!(self.peek(), Some(b'0'..=b'9')) {
                return Err(err(format!("{}invalid number", self.field_ctx())));
            }
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
        }
        Ok(String::from_utf8_lossy(&self.bytes[start..self.pos]).into_owned())
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
    p.skip_ws();
    if p.peek() != Some(b'{') {
        return Err(err("request must be one JSON object"));
    }
    let v = p.parse_object(0)?;
    p.skip_ws();
    if !p.eof() {
        // Distinguish trailing garbage from a second value for parity with
        // finishObject: any trailing token is an error either way.
        return Err(err("request must contain exactly one JSON object"));
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

/// Typed strict object binding with Go field matching: exact name first,
/// then a unique ASCII case-insensitive match; anything else is unknown.
pub struct Binder<'a> {
    fields: &'a Vec<(String, Value)>,
    seen: Vec<bool>,
}

impl<'a> Binder<'a> {
    pub fn new(v: &'a Value) -> Result<Self, Error> {
        match v {
            Value::Object(fields) => Ok(Binder {
                fields,
                seen: vec![false; fields.len()],
            }),
            _ => Err(err("decode request: expected object")),
        }
    }

    fn lookup(&mut self, name: &str) -> Result<Option<&'a Value>, Error> {
        if let Some(i) = self.fields.iter().position(|(k, _)| k == name) {
            self.seen[i] = true;
            return Ok(Some(&self.fields[i].1));
        }
        let mut found = None;
        for (i, (k, _)) in self.fields.iter().enumerate() {
            if k.eq_ignore_ascii_case(name) {
                if found.is_some() {
                    return Err(err(format!("decode request: ambiguous field {name:?}")));
                }
                found = Some(i);
            }
        }
        if let Some(i) = found {
            self.seen[i] = true;
            Ok(Some(&self.fields[i].1))
        } else {
            Ok(None)
        }
    }

    /// Required string field; missing or null decodes as empty (Go leaves
    /// the zero value for null), wrong types are an error.
    pub fn string(&mut self, name: &str) -> Result<String, Error> {
        match self.lookup(name)? {
            None | Some(Value::Null) => Ok(String::new()),
            Some(Value::Str(s)) => Ok(s.clone()),
            Some(_) => Err(err(format!(
                "decode request: cannot unmarshal field {name:?} as string"
            ))),
        }
    }

    pub fn boolean(&mut self, name: &str) -> Result<bool, Error> {
        match self.lookup(name)? {
            None | Some(Value::Null) => Ok(false),
            Some(Value::Bool(b)) => Ok(*b),
            Some(_) => Err(err(format!(
                "decode request: cannot unmarshal field {name:?} as bool"
            ))),
        }
    }

    pub fn int64(&mut self, name: &str) -> Result<i64, Error> {
        match self.lookup(name)? {
            None | Some(Value::Null) => Ok(0),
            Some(v @ Value::Number(_)) => v.as_i64().ok_or_else(|| {
                err(format!(
                    "decode request: cannot unmarshal field {name:?} as int"
                ))
            }),
            Some(_) => Err(err(format!(
                "decode request: cannot unmarshal field {name:?} as int"
            ))),
        }
    }

    /// Optional nested object; missing or null yields None.
    pub fn object(&mut self, name: &str) -> Result<Option<&'a Value>, Error> {
        match self.lookup(name)? {
            None | Some(Value::Null) => Ok(None),
            Some(v @ Value::Object(_)) => Ok(Some(v)),
            Some(_) => Err(err(format!(
                "decode request: cannot unmarshal field {name:?} as object"
            ))),
        }
    }

    /// Optional string list; missing or null yields None, `[]` yields empty.
    pub fn string_list(&mut self, name: &str) -> Result<Option<Vec<String>>, Error> {
        match self.lookup(name)? {
            None | Some(Value::Null) => Ok(None),
            Some(Value::Array(items)) => {
                let mut out = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        // Null leaves the zero value, like encoding/json.
                        Value::Null => out.push(String::new()),
                        Value::Str(s) => out.push(s.clone()),
                        _ => {
                            return Err(err(format!(
                                "decode request: cannot unmarshal field {name:?} item as string"
                            )))
                        }
                    }
                }
                Ok(Some(out))
            }
            Some(_) => Err(err(format!(
                "decode request: cannot unmarshal field {name:?} as list"
            ))),
        }
    }

    /// Raw field access for custom nested decoders (maps, DTO lists).
    pub fn raw(&mut self, name: &str) -> Result<Option<&'a Value>, Error> {
        self.lookup(name)
    }

    /// Reject any fields the DTO did not consume (DisallowUnknownFields).
    pub fn finish(&self) -> Result<(), Error> {
        if let Some(i) = self.seen.iter().position(|s| !s) {
            return Err(err(format!(
                "decode request: unknown field {:?}",
                self.fields[i].0
            )));
        }
        Ok(())
    }
}

/// Tolerant field lookup for podman output: exact name first, then unique
/// case-insensitive; missing and null are both None.
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
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\u{0008}' => out.push_str("\\b"),
            '\u{000C}' => out.push_str("\\f"),
            '<' => out.push_str("\\u003c"),
            '>' => out.push_str("\\u003e"),
            '&' => out.push_str("\\u0026"),
            '\u{2028}' => out.push_str("\\u2028"),
            '\u{2029}' => out.push_str("\\u2029"),
            c if (c as u32) < 0x20 => {
                use std::fmt::Write;
                write!(out, "\\u{:04x}", c as u32).unwrap();
            }
            c => out.push(c),
        }
    }
}

pub fn quote(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    out.push('"');
    escape_into(&mut out, s);
    out.push('"');
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn strict_accepts_one_known_object() {
        let v = decode_strict(br#"{"id":"one"}"#).unwrap();
        assert_eq!(
            v.as_object().unwrap(),
            &vec![("id".to_string(), Value::Str("one".to_string()))]
        );
    }

    #[test]
    fn strict_rejects_bad_shapes() {
        for input in [
            "{\"id\":\"one\",\"id\":\"two\"}",
            "[]",
            "{\"id\":\"one\"}{\"id\":\"two\"}",
            "{\"id\":\"one\"",
            "null",
            "\"str\"",
            "42",
            "{\"id\":}",
            "{,}",
        ] {
            assert!(decode_strict(input.as_bytes()).is_err(), "{input}");
        }
        // Trailing whitespace is fine.
        assert!(decode_strict(b"{\"id\":\"one\"}\n").is_ok());
    }

    #[test]
    fn strict_rejects_nested_duplicates() {
        for input in [
            "{\"id\":\"one\",\"meta\":{\"key\":\"1\",\"key\":\"2\"}}",
            "{\"id\":\"one\",\"meta\":{\"outer\":{\"key\":\"1\",\"key\":\"2\"}}}",
            "{\"id\":\"one\",\"items\":[{\"key\":\"1\"},{\"key\":\"1\",\"key\":\"2\"}]}",
        ] {
            let e = decode_strict(input.as_bytes()).unwrap_err();
            assert!(e.0.contains("duplicate"), "{input}: {e}");
        }
        assert!(decode_strict(br#"{"id":"one","meta":{"first":"1","second":"2"}}"#).is_ok());
    }

    #[test]
    fn strict_rejects_invalid_utf8_and_oversized() {
        let mut bad = b"{\"id\":\"".to_vec();
        bad.push(0xff);
        bad.extend_from_slice(b"\"}");
        assert!(decode_strict(&bad).unwrap_err().0.contains("UTF-8"));
        let big = format!("{{\"id\":{:?}}}", "a".repeat(MAXIMUM_REQUEST_BYTES));
        assert!(decode_strict(big.as_bytes())
            .unwrap_err()
            .0
            .contains("1 MiB"));
    }

    #[test]
    fn binder_matches_go_field_rules() {
        let v = decode_strict(br#"{"ID":"x","Count":3,"Flag":true}"#).unwrap();
        let mut b = Binder::new(&v).unwrap();
        assert_eq!(b.string("id").unwrap(), "x"); // case-insensitive fallback
        assert_eq!(b.int64("count").unwrap(), 3);
        assert!(b.boolean("flag").unwrap());
        b.finish().unwrap();

        let v = decode_strict(br#"{"id":"x","bogus":1}"#).unwrap();
        let mut b = Binder::new(&v).unwrap();
        b.string("id").unwrap();
        assert!(b.finish().is_err()); // unknown field

        let v = decode_strict(br#"{"id":null,"count":null}"#).unwrap();
        let mut b = Binder::new(&v).unwrap();
        assert_eq!(b.string("id").unwrap(), "");
        assert_eq!(b.int64("count").unwrap(), 0);

        for bad in [
            br#"{"count":1.5}"#.as_slice(),
            br#"{"count":"3"}"#,
            br#"{"count":true}"#,
            br#"{"count":1e3}"#,
        ] {
            let v = decode_strict(bad).unwrap();
            assert!(Binder::new(&v).unwrap().int64("count").is_err());
        }
    }

    #[test]
    fn surrogate_pairs_match_go_substitution() {
        // Lone surrogates become U+FFFD, like encoding/json.
        let v = decode_strict(b"{\"a\":\"\\ud800\"}").unwrap();
        assert_eq!(tolerant_get(&v, "a").unwrap().as_str().unwrap(), "\u{FFFD}");
        let v = decode_strict(b"{\"a\":\"\\udc00\"}").unwrap();
        assert_eq!(tolerant_get(&v, "a").unwrap().as_str().unwrap(), "\u{FFFD}");
        // Valid pair decodes to the astral character.
        let v = decode_strict(b"{\"a\":\"\\ud800\\udc00\"}").unwrap();
        assert_eq!(
            tolerant_get(&v, "a").unwrap().as_str().unwrap(),
            "\u{10000}"
        );
        // Bad hex in an escape is a hard error.
        assert!(decode_strict(b"{\"a\":\"\\ud80x\"}").is_err());
        // Raw C0 controls inside strings are rejected.
        assert!(decode_strict(b"{\"a\":\"a\x01b\"}").is_err());
    }

    #[test]
    fn go_string_escaping_matches() {
        assert_eq!(
            quote("a<b>&\"c\"\n\t\x01\u{2028}é"),
            "\"a\\u003cb\\u003e\\u0026\\\"c\\\"\\n\\t\\u0001\\u2028é\""
        );
    }

    #[test]
    fn tolerant_decode_keeps_last_duplicate() {
        let v = decode_tolerant(br#"{"a":"1","a":"2"}"#).unwrap();
        assert_eq!(tolerant_get(&v, "a").unwrap().as_str().unwrap(), "2");
        let v = decode_tolerant(br#"[{"a":1}]"#).unwrap();
        assert!(v.as_array().is_some());
    }
}
