//! Minimal JSON parsing and Go-compatible string escaping shared by the
//! Rust operator ports. The parser validates one complete JSON value;
//! object lookup keeps duplicate-key last-wins semantics.

#[derive(Debug, Clone, PartialEq)]
pub enum JsonValue {
    Null,
    Bool(bool),
    Number(String),
    Str(String),
    Array(Vec<JsonValue>),
    Object(Vec<(String, JsonValue)>),
}

#[derive(Debug, Clone, PartialEq)]
pub struct ParseError;

impl JsonValue {
    pub fn parse(text: &str) -> Result<JsonValue, ParseError> {
        let mut parser = Parser {
            bytes: text.as_bytes(),
            pos: 0,
            depth: 0,
        };
        parser.skip_ws();
        let value = parser.parse_value()?;
        parser.skip_ws();
        if parser.pos != parser.bytes.len() {
            return Err(ParseError);
        }
        Ok(value)
    }

    /// Last value wins on duplicate keys, matching Go and Python decoders.
    pub fn get(&self, key: &str) -> Option<&JsonValue> {
        match self {
            JsonValue::Object(entries) => {
                entries.iter().rev().find(|(k, _)| k == key).map(|(_, v)| v)
            }
            _ => None,
        }
    }

    pub fn as_bool(&self) -> Option<bool> {
        match self {
            JsonValue::Bool(b) => Some(*b),
            _ => None,
        }
    }

    pub fn as_str(&self) -> Option<&str> {
        match self {
            JsonValue::Str(s) => Some(s),
            _ => None,
        }
    }

    pub fn is_object(&self) -> bool {
        matches!(self, JsonValue::Object(_))
    }

    /// Strict JSON integer literal (no fraction or exponent), as i128 so
    /// callers can range-check huge values the way Python's unbounded int
    /// comparison does.
    pub fn as_integer(&self) -> Option<i128> {
        match self {
            JsonValue::Number(raw) => {
                let digits = raw.strip_prefix('-').unwrap_or(raw);
                if digits.is_empty() || !digits.bytes().all(|b| b.is_ascii_digit()) {
                    return None;
                }
                raw.parse().ok()
            }
            _ => None,
        }
    }
}

/// Escape a string the way Go's encoding/json does: short escapes for the
/// common controls, `\u00xx` for the rest, `<`, `>`, `&` escaped.
pub fn escape_into(out: &mut String, value: &str) {
    out.push('"');
    for ch in value.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '<' => out.push_str("\\u003c"),
            '>' => out.push_str("\\u003e"),
            '&' => out.push_str("\\u0026"),
            '\u{2028}' => out.push_str("\\u2028"),
            '\u{2029}' => out.push_str("\\u2029"),
            c if (c as u32) < 0x20 => {
                out.push_str(&format!("\\u{:04x}", c as u32));
            }
            c => out.push(c),
        }
    }
    out.push('"');
}

struct Parser<'a> {
    bytes: &'a [u8],
    pos: usize,
    depth: u32,
}

/// Go caps nesting at 10000; refuse deeper input instead of overflowing the
/// stack on hostile blobs.
const MAX_DEPTH: u32 = 10000;

impl<'a> Parser<'a> {
    fn skip_ws(&mut self) {
        while self.pos < self.bytes.len()
            && matches!(self.bytes[self.pos], b' ' | b'\t' | b'\n' | b'\r')
        {
            self.pos += 1;
        }
    }

    fn parse_value(&mut self) -> Result<JsonValue, ParseError> {
        match self.bytes.get(self.pos) {
            Some(b'{') => {
                if self.depth >= MAX_DEPTH {
                    return Err(ParseError);
                }
                self.depth += 1;
                let value = self.parse_object();
                self.depth -= 1;
                value
            }
            Some(b'[') => {
                if self.depth >= MAX_DEPTH {
                    return Err(ParseError);
                }
                self.depth += 1;
                let value = self.parse_array();
                self.depth -= 1;
                value
            }
            Some(b'"') => Ok(JsonValue::Str(self.parse_string()?)),
            Some(b't') => self.parse_literal("true", JsonValue::Bool(true)),
            Some(b'f') => self.parse_literal("false", JsonValue::Bool(false)),
            Some(b'n') => self.parse_literal("null", JsonValue::Null),
            Some(b'-') | Some(b'0'..=b'9') => Ok(JsonValue::Number(self.parse_number()?)),
            _ => Err(ParseError),
        }
    }

    fn parse_literal(&mut self, word: &str, value: JsonValue) -> Result<JsonValue, ParseError> {
        if self.bytes[self.pos..].starts_with(word.as_bytes()) {
            self.pos += word.len();
            Ok(value)
        } else {
            Err(ParseError)
        }
    }

    fn parse_object(&mut self) -> Result<JsonValue, ParseError> {
        self.pos += 1;
        let mut entries = Vec::new();
        self.skip_ws();
        if self.bytes.get(self.pos) == Some(&b'}') {
            self.pos += 1;
            return Ok(JsonValue::Object(entries));
        }
        loop {
            self.skip_ws();
            if self.bytes.get(self.pos) != Some(&b'"') {
                return Err(ParseError);
            }
            let key = self.parse_string()?;
            self.skip_ws();
            if self.bytes.get(self.pos) != Some(&b':') {
                return Err(ParseError);
            }
            self.pos += 1;
            self.skip_ws();
            entries.push((key, self.parse_value()?));
            self.skip_ws();
            match self.bytes.get(self.pos) {
                Some(b',') => self.pos += 1,
                Some(b'}') => {
                    self.pos += 1;
                    return Ok(JsonValue::Object(entries));
                }
                _ => return Err(ParseError),
            }
        }
    }

    fn parse_array(&mut self) -> Result<JsonValue, ParseError> {
        self.pos += 1;
        let mut items = Vec::new();
        self.skip_ws();
        if self.bytes.get(self.pos) == Some(&b']') {
            self.pos += 1;
            return Ok(JsonValue::Array(items));
        }
        loop {
            self.skip_ws();
            items.push(self.parse_value()?);
            self.skip_ws();
            match self.bytes.get(self.pos) {
                Some(b',') => self.pos += 1,
                Some(b']') => {
                    self.pos += 1;
                    return Ok(JsonValue::Array(items));
                }
                _ => return Err(ParseError),
            }
        }
    }

    fn parse_string(&mut self) -> Result<String, ParseError> {
        self.pos += 1;
        let mut out: Vec<u8> = Vec::new();
        loop {
            let byte = *self.bytes.get(self.pos).ok_or(ParseError)?;
            self.pos += 1;
            match byte {
                b'"' => break,
                b'\\' => {
                    let esc = *self.bytes.get(self.pos).ok_or(ParseError)?;
                    self.pos += 1;
                    match esc {
                        b'"' => out.push(b'"'),
                        b'\\' => out.push(b'\\'),
                        b'/' => out.push(b'/'),
                        b'b' => out.push(0x08),
                        b'f' => out.push(0x0c),
                        b'n' => out.push(b'\n'),
                        b'r' => out.push(b'\r'),
                        b't' => out.push(b'\t'),
                        b'u' => {
                            if self.pos + 4 > self.bytes.len() {
                                return Err(ParseError);
                            }
                            let hex = std::str::from_utf8(&self.bytes[self.pos..self.pos + 4])
                                .map_err(|_| ParseError)?;
                            let mut code = u32::from_str_radix(hex, 16).map_err(|_| ParseError)?;
                            self.pos += 4;
                            if (0xd800..0xdc00).contains(&code)
                                && self.bytes.get(self.pos..self.pos + 2) == Some(b"\\u".as_slice())
                            {
                                // H03-F1: the low-surrogate tail needs 4 hex
                                // bytes after \u; a truncated tail is malformed
                                // input, not a panic.
                                let tail = self
                                    .bytes
                                    .get(self.pos + 2..self.pos + 6)
                                    .ok_or(ParseError)?;
                                let low_hex = std::str::from_utf8(tail).map_err(|_| ParseError)?;
                                let low =
                                    u32::from_str_radix(low_hex, 16).map_err(|_| ParseError)?;
                                if (0xdc00..0xe000).contains(&low) {
                                    code = 0x10000 + ((code - 0xd800) << 10) + (low - 0xdc00);
                                    self.pos += 6;
                                }
                            }
                            // Go replaces lone surrogates with U+FFFD instead of failing.
                            let ch = char::from_u32(code).unwrap_or('\u{FFFD}');
                            let mut buf = [0u8; 4];
                            out.extend_from_slice(ch.encode_utf8(&mut buf).as_bytes());
                        }
                        _ => return Err(ParseError),
                    }
                }
                0x00..=0x1f => return Err(ParseError),
                _ => out.push(byte),
            }
        }
        String::from_utf8(out).map_err(|_| ParseError)
    }

    fn parse_number(&mut self) -> Result<String, ParseError> {
        let start = self.pos;
        if self.bytes.get(self.pos) == Some(&b'-') {
            self.pos += 1;
        }
        match self.bytes.get(self.pos) {
            Some(b'0') => self.pos += 1,
            Some(b'1'..=b'9') => {
                while matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                    self.pos += 1;
                }
            }
            _ => return Err(ParseError),
        }
        if self.bytes.get(self.pos) == Some(&b'.') {
            self.pos += 1;
            if !matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                return Err(ParseError);
            }
            while matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
        }
        if matches!(self.bytes.get(self.pos), Some(b'e' | b'E')) {
            self.pos += 1;
            if matches!(self.bytes.get(self.pos), Some(b'+' | b'-')) {
                self.pos += 1;
            }
            if !matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                return Err(ParseError);
            }
            while matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
        }
        std::str::from_utf8(&self.bytes[start..self.pos])
            .map(|s| s.to_string())
            .map_err(|_| ParseError)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn round_trip_shapes() {
        let value = JsonValue::parse("{\"a\": [1, true, null, \"x\\u0041\"], \"n\": -12.5e2}")
            .expect("parse");
        assert!(value.is_object());
        assert_eq!(value.get("missing"), None);
        match value.get("a").expect("a") {
            JsonValue::Array(items) => assert_eq!(items.len(), 4),
            _ => panic!("array lost"),
        }
        assert_eq!(value.get("n").expect("n").as_integer(), None);
    }

    #[test]
    fn integers_and_duplicates() {
        let value = JsonValue::parse("{\"id\": 42, \"id\": 7, \"big\": 9223372036854775807}")
            .expect("parse");
        assert_eq!(value.get("id").and_then(|v| v.as_integer()), Some(7));
        assert_eq!(
            value.get("big").and_then(|v| v.as_integer()),
            Some(9223372036854775807)
        );
        assert_eq!(
            JsonValue::parse("{\"id\": 7.0}")
                .expect("p")
                .get("id")
                .and_then(|v| v.as_integer()),
            None
        );
        assert_eq!(
            JsonValue::parse("{\"id\": true}")
                .expect("p")
                .get("id")
                .and_then(|v| v.as_integer()),
            None
        );
    }

    #[test]
    fn rejects_trailing_garbage() {
        assert!(JsonValue::parse("{\"a\":1} x").is_err());
        assert!(JsonValue::parse("").is_err());
        assert!(JsonValue::parse("{\"a\":}").is_err());
    }

    #[test]
    fn truncated_surrogate_tail_is_error_not_panic() {
        // H03-F1: a high surrogate followed by a truncated \u tail must
        // return ParseError, never panic on the unchecked tail slice.
        assert_eq!(JsonValue::parse("\"\\uD800\\u12\""), Err(ParseError));
        assert_eq!(JsonValue::parse("\"\\uD800\\u\""), Err(ParseError));
        // Established behavior is preserved: valid pairs decode, lone
        // surrogates map to U+FFFD like Go, and a non-low \u tail falls
        // through to lone-surrogate handling plus a fresh escape.
        assert_eq!(
            JsonValue::parse("\"\\uD834\\uDD1E\"")
                .expect("pair")
                .as_str(),
            Some("\u{1D11E}")
        );
        assert_eq!(
            JsonValue::parse("\"\\uD800\"").expect("lone").as_str(),
            Some("\u{FFFD}")
        );
        assert_eq!(
            JsonValue::parse("\"\\uD800\\u0041\"")
                .expect("fallthrough")
                .as_str(),
            Some("\u{FFFD}A")
        );
    }

    #[test]
    fn escapes_match_go() {
        let mut out = String::new();
        escape_into(&mut out, "a<b>&\"c\"\n");
        assert_eq!(out, "\"a\\u003cb\\u003e\\u0026\\\"c\\\"\\n\"");
    }
}
