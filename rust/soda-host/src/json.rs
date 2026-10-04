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

impl<'a> Parser<'a> {
    fn new(bytes: &'a [u8], strict: bool) -> Self {
        Parser {
            bytes,
            pos: 0,
            strict_field: strict.then_some(None),
            deferred: None,
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

    fn strict(&self) -> bool {
        self.strict_field.is_some()
    }

    fn field_ctx(&self) -> String {
        self.strict_field
            .as_ref()
            .and_then(|f| f.clone())
            .map(|f| format!("decode request field {}: ", go_quote(&f)))
            .unwrap_or_else(|| "decode request: ".to_string())
    }

    fn parse_value(&mut self, depth: u32) -> Result<Value, Error> {
        debug_assert_eq!(depth, 0);
        self.run_machine()
    }

    fn parse_object(&mut self, depth: u32) -> Result<Value, Error> {
        debug_assert_eq!(depth, 0);
        match self.run_machine()? {
            v @ Value::Object(_) => Ok(v),
            _ => Err(err("decode request: expected object")),
        }
    }

    fn parse_literal(&mut self, lit: &str, v: Value) -> Result<Value, Error> {
        debug_assert_eq!(self.peek(), Some(lit.as_bytes()[0]));
        self.pos += 1;
        for &want in &lit.as_bytes()[1..] {
            match self.peek() {
                Some(c) if c == want => {
                    self.pos += 1;
                }
                None => return Err(err(format!("{}unexpected EOF", self.field_ctx()))),
                Some(c) => {
                    return Err(err(format!(
                        "{}invalid character {} in literal {lit} (expecting '{}')",
                        self.field_ctx(),
                        Self::quote_byte(c),
                        want as char
                    )));
                }
            }
        }
        Ok(v)
    }

    /// Iterative JSON reader with an explicit heap stack: 10000-deep machine
    /// output must parse without overflowing the thread stack, exactly like
    /// `encoding/json`. Depth checks sit at the same positions as the former
    /// recursion: strict values (after the field name, like strictjson's
    /// depth-first scan) and tolerant container opens.
    /// Single-quoted scanner character, mirroring `quoteChar`: `'''` and
    /// `'"'` special-cased, `strconv.Quote`-style escapes inside, printable
    /// ASCII raw, Latin-1 printables raw, everything else `\xnn`/`\u00nn`.
    fn quote_byte(c: u8) -> String {
        if c == b'\'' {
            return "'\\''".to_string();
        }
        if c == b'"' {
            return "'\"'".to_string();
        }
        let inner = match c {
            b'\n' => "\\n".to_string(),
            b'\r' => "\\r".to_string(),
            b'\t' => "\\t".to_string(),
            0x07 => "\\a".to_string(),
            0x08 => "\\b".to_string(),
            0x0C => "\\f".to_string(),
            0x0B => "\\v".to_string(),
            b'\\' => "\\\\".to_string(),
            0x20..=0x7E => (c as char).to_string(),
            0xA1..=0xFF => char::from_u32(c as u32).unwrap().to_string(),
            _ => {
                if c < 0xA0 {
                    return format!("'\\x{c:02x}'");
                }
                return format!("'\\u00{c:02x}'");
            }
        };
        format!("'{inner}'")
    }

    /// Iterative JSON reader with an explicit heap stack: 10000-deep machine
    /// output must parse without overflowing the thread stack, exactly like
    /// `encoding/json`. Message shapes mirror the scanner (`invalid
    /// character`, `in string literal`, ...) wrapped the way
    /// `strictjson.Decode` wraps `Token`/`Decode` errors: top-level framing
    /// errors are bare, value errors carry the top field, and nested
    /// duplicate/depth findings are deferred until the top-level value they
    /// sit in has scanned clean (Go walks a `RawMessage` copy after the
    /// scan, so any scan error beats them).
    fn run_machine(&mut self) -> Result<Value, Error> {
        // Strict root: `requireObject` takes one `Token`. A scan error
        // surfaces; any complete non-object token is rejected.
        if self.strict() {
            self.skip_ws();
            match self.peek() {
                None => return Err(err("decode request: EOF")),
                Some(b'{') => {}
                Some(b'[') => return Err(err("request must be one JSON object")),
                Some(b'"') => {
                    self.parse_string()?;
                    return Err(err("request must be one JSON object"));
                }
                Some(b't') => {
                    self.parse_literal("true", Value::Bool(true))?;
                    return Err(err("request must be one JSON object"));
                }
                Some(b'f') => {
                    self.parse_literal("false", Value::Bool(false))?;
                    return Err(err("request must be one JSON object"));
                }
                Some(b'n') => {
                    self.parse_literal("null", Value::Null)?;
                    return Err(err("request must be one JSON object"));
                }
                Some(b'-') | Some(b'0'..=b'9') => {
                    let lit = self.parse_number()?;
                    // `requireObject` takes one `Token`, which converts the
                    // number: overflow fails before the shape is judged.
                    if !lit.parse::<f64>().is_ok_and(|v| v.is_finite()) {
                        return Err(err(format!(
                            "decode request: json: cannot unmarshal number {lit} into Go value of type float64"
                        )));
                    }
                    return Err(err("request must be one JSON object"));
                }
                Some(c) => {
                    return Err(err(format!(
                        "decode request: invalid character {} looking for beginning of value",
                        Self::quote_byte(c)
                    )))
                }
            }
        }
        let mut stack: Vec<Frame> = Vec::new();
        let mut pending: Option<Value> = None;
        loop {
            if let Some(v) = pending.take() {
                match stack.last_mut() {
                    None => return Ok(v),
                    Some(Frame::Array { items }) => items.push(v),
                    Some(Frame::Object {
                        fields,
                        pending: name,
                    }) => {
                        let name = name.take().expect("object value without field name");
                        if self.strict() {
                            fields.push((name, v));
                        } else if let Some(slot) = fields.iter_mut().find(|(k, _)| *k == name) {
                            slot.1 = v;
                        } else {
                            fields.push((name, v));
                        }
                    }
                }
                // A root-object field value just completed: flush the first
                // deferred nested duplicate/depth finding (Go's post-scan
                // walk reports it before the next field is read), then clear
                // the top-level field context.
                if stack.len() == 1 {
                    if let Some(e) = self.deferred.take() {
                        return Err(e);
                    }
                    if let Some(slot) = self.strict_field.as_mut() {
                        *slot = None;
                    }
                }
                self.skip_ws();
                let is_object = matches!(stack.last(), Some(Frame::Object { .. }));
                match self.peek() {
                    Some(b',') => {
                        self.pos += 1;
                    }
                    Some(b'}') if is_object => {
                        self.pos += 1;
                        pending = Some(close_frame(stack.pop().expect("object frame")));
                        continue;
                    }
                    Some(b']') if !is_object => {
                        self.pos += 1;
                        pending = Some(close_frame(stack.pop().expect("array frame")));
                        continue;
                    }
                    None if stack.len() <= 1 => return Err(err("decode request: EOF")),
                    None => {
                        return Err(err(format!("{}unexpected EOF", self.field_ctx())));
                    }
                    Some(c) => {
                        return Err(err(format!(
                            "{}invalid character {} after {}",
                            self.field_ctx(),
                            Self::quote_byte(c),
                            if is_object {
                                "object key:value pair"
                            } else {
                                "array element"
                            }
                        )));
                    }
                }
            }
            // Object field-name step comes before value scanning, like the
            // streaming `Token` name read: a bad name reports before the
            // value is touched.
            if matches!(stack.last(), Some(Frame::Object { .. })) {
                self.skip_ws();
                let nested = stack.len() > 1;
                match self.peek() {
                    None if !nested => return Err(err("decode request: EOF")),
                    None => {
                        return Err(err(format!("{}unexpected EOF", self.field_ctx())));
                    }
                    Some(b'"') => {}
                    Some(c) => {
                        // A first field name reports the bare character: the
                        // streaming error has no context in object-start
                        // state, only once a comma was consumed.
                        let first = match stack.last() {
                            Some(Frame::Object { fields, .. }) => fields.is_empty(),
                            _ => false,
                        };
                        if !nested && first {
                            return Err(err(format!(
                                "decode request: invalid character {}",
                                Self::quote_byte(c)
                            )));
                        }
                        return Err(err(format!(
                            "{}invalid character {} looking for beginning of object key string",
                            self.field_ctx(),
                            Self::quote_byte(c)
                        )));
                    }
                }
                let name = self.parse_string()?;
                let duplicate = match stack.last() {
                    Some(Frame::Object { fields, .. }) => fields.iter().any(|(k, _)| *k == name),
                    _ => false,
                };
                if self.strict() && duplicate {
                    if nested {
                        // Deferred: the enclosing value must scan clean
                        // first; a later scan error beats this.
                        if self.deferred.is_none() {
                            self.deferred =
                                Some(err(format!("duplicate request field {}", go_quote(&name))));
                        }
                    } else {
                        return Err(err(format!("duplicate request field {}", go_quote(&name))));
                    }
                }
                if let Some(Frame::Object { pending: slot, .. }) = stack.last_mut() {
                    *slot = Some(name.clone());
                }
                // Top-level field context for nested strict messages, with a
                // fresh deferred slot for this field's value.
                if stack.len() == 1 {
                    if let Some(slot) = self.strict_field.as_mut() {
                        *slot = Some(name);
                    }
                    self.deferred = None;
                }
                self.skip_ws();
                match self.peek() {
                    Some(b':') => {
                        self.pos += 1;
                    }
                    None if !nested => {
                        return Err(err(format!("{}EOF", self.field_ctx())));
                    }
                    None => {
                        return Err(err(format!("{}unexpected EOF", self.field_ctx())));
                    }
                    Some(c) if nested => {
                        return Err(err(format!(
                            "{}invalid character {} after object key",
                            self.field_ctx(),
                            Self::quote_byte(c)
                        )));
                    }
                    Some(_) => {
                        return Err(err(format!(
                            "{}expected colon after object key",
                            self.field_ctx()
                        )));
                    }
                }
            }
            // Go checks nesting depth per value in the post-scan walk, so a
            // value that scans clean but sits too deep reports only once its
            // top-level value completes; anything scanning past the scanner's
            // own 10000-deep cap fails immediately like `readValue`.
            if self.strict() && (stack.len() as u32) > MAX_NESTING && self.deferred.is_none() {
                self.deferred = Some(err(format!(
                    "{}request is nested too deeply",
                    self.field_ctx()
                )));
            }
            self.skip_ws();
            match self.peek() {
                Some(b'{') | Some(b'[') => {
                    let bracket = self.peek().unwrap();
                    if self.strict() && stack.len() > 10_000 {
                        return Err(err(format!(
                            "{}invalid character {} exceeded max depth",
                            self.field_ctx(),
                            Self::quote_byte(bracket)
                        )));
                    }
                    if !self.strict() && stack.len() >= 10_000 {
                        return Err(err("decode request: request is nested too deeply"));
                    }
                    let is_object = bracket == b'{';
                    self.pos += 1;
                    if is_object {
                        stack.push(Frame::Object {
                            fields: Vec::new(),
                            pending: None,
                        });
                    } else {
                        stack.push(Frame::Array { items: Vec::new() });
                    }
                    self.skip_ws();
                    let closed = (is_object && self.peek() == Some(b'}'))
                        || (!is_object && self.peek() == Some(b']'));
                    if closed {
                        self.pos += 1;
                        pending = Some(close_frame(stack.pop().expect("new frame")));
                    }
                }
                Some(b'"') => pending = Some(Value::Str(self.parse_string()?)),
                Some(b't') => pending = Some(self.parse_literal("true", Value::Bool(true))?),
                Some(b'f') => pending = Some(self.parse_literal("false", Value::Bool(false))?),
                Some(b'n') => pending = Some(self.parse_literal("null", Value::Null)?),
                Some(b'-') | Some(b'0'..=b'9') => {
                    let lit = self.parse_number()?;
                    // strictjson walks each top-level value with `Token`,
                    // which converts numbers: an overflowing literal joins
                    // the deferred findings in walk order, after any scan
                    // error but beside duplicate/depth findings. Underflow
                    // to zero stays accepted, exactly like Go.
                    if self.strict()
                        && self.deferred.is_none()
                        && !lit.parse::<f64>().is_ok_and(|v| v.is_finite())
                    {
                        self.deferred = Some(err(format!(
                            "{}json: cannot unmarshal number {} into Go value of type float64",
                            self.field_ctx(),
                            lit
                        )));
                    }
                    pending = Some(Value::Number(lit))
                }
                None if stack.len() <= 1 => return Err(err(format!("{}EOF", self.field_ctx()))),
                None => return Err(err(format!("{}unexpected EOF", self.field_ctx()))),
                Some(c) => {
                    return Err(err(format!(
                        "{}invalid character {} looking for beginning of value",
                        self.field_ctx(),
                        Self::quote_byte(c)
                    )))
                }
            }
        }
    }

    fn hex_val(c: u8) -> Option<u32> {
        match c {
            b'0'..=b'9' => Some((c - b'0') as u32),
            b'a'..=b'f' => Some((c - b'a' + 10) as u32),
            b'A'..=b'F' => Some((c - b'A' + 10) as u32),
            _ => None,
        }
    }

    fn hex4(&mut self) -> Result<u32, Error> {
        let mut v: u32 = 0;
        for _ in 0..4 {
            match self.peek() {
                None => return Err(err(format!("{}unexpected EOF", self.field_ctx()))),
                Some(c) => match Self::hex_val(c) {
                    Some(d) => {
                        v = v * 16 + d;
                        self.pos += 1;
                    }
                    None => {
                        return Err(err(format!(
                            "{}invalid character {} in \\u hexadecimal character escape",
                            self.field_ctx(),
                            Self::quote_byte(c)
                        )))
                    }
                },
            }
        }
        Ok(v)
    }

    fn parse_string(&mut self) -> Result<String, Error> {
        self.pos += 1; // opening quote
        let mut out = String::new();
        loop {
            let c = match self.peek() {
                Some(c) => c,
                None => return Err(err(format!("{}unexpected EOF", self.field_ctx()))),
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
                        None => return Err(err(format!("{}unexpected EOF", self.field_ctx()))),
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
                        _ => {
                            return Err(err(format!(
                                "{}invalid character {} in string escape code",
                                self.field_ctx(),
                                Self::quote_byte(e)
                            )))
                        }
                    }
                }
                0x00..=0x1F => {
                    return Err(err(format!(
                        "{}invalid character {} in string literal",
                        self.field_ctx(),
                        Self::quote_byte(c)
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
            match self.peek() {
                Some(b'0'..=b'9') => {}
                None => return Err(err(format!("{}unexpected EOF", self.field_ctx()))),
                Some(c) => {
                    return Err(err(format!(
                        "{}invalid character {} in numeric literal",
                        self.field_ctx(),
                        Self::quote_byte(c)
                    )))
                }
            }
        }
        match self.peek() {
            Some(b'0') => {
                // A leading zero ends the integer part even before another
                // digit: the scanner stops the value there.
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
            match self.peek() {
                Some(b'0'..=b'9') => {
                    while matches!(self.peek(), Some(b'0'..=b'9')) {
                        self.pos += 1;
                    }
                }
                None => return Err(err(format!("{}unexpected EOF", self.field_ctx()))),
                Some(c) => {
                    return Err(err(format!(
                        "{}invalid character {} after decimal point in numeric literal",
                        self.field_ctx(),
                        Self::quote_byte(c)
                    )))
                }
            }
        }
        if matches!(self.peek(), Some(b'e' | b'E')) {
            self.pos += 1;
            if matches!(self.peek(), Some(b'+' | b'-')) {
                self.pos += 1;
            }
            match self.peek() {
                Some(b'0'..=b'9') => {
                    while matches!(self.peek(), Some(b'0'..=b'9')) {
                        self.pos += 1;
                    }
                }
                None => return Err(err(format!("{}unexpected EOF", self.field_ctx()))),
                Some(c) => {
                    return Err(err(format!(
                        "{}invalid character {} in exponent of numeric literal",
                        self.field_ctx(),
                        Self::quote_byte(c)
                    )))
                }
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

/// `strconv.ParseInt(s, 10, 64)`: Rust's parse rejects the leading `+`
/// that Go accepts, so strip one first.
pub fn parse_go_int64(s: &str) -> Option<i64> {
    s.strip_prefix('+').unwrap_or(s).parse::<i64>().ok()
}

/// `strconv.ParseUint(s, 10, 32)`: same leading-`+` rule as above.
pub fn parse_go_uint32(s: &str) -> Option<u32> {
    s.strip_prefix('+').unwrap_or(s).parse::<u32>().ok()
}

/// One struct field's binding rule. `go_type` is the exact
/// `encoding/json` type word used in mismatch messages.
pub struct Spec {
    pub name: &'static str,
    pub kind: Kind,
}

pub enum Kind {
    Str,
    Bool,
    I64,
    /// Go `int` (64-bit): same range as I64, `int` in messages.
    Int,
    /// Go `uint32`.
    U32,
    /// `*int`: missing/null is None.
    OptInt,
    StrList,
    /// `[]byte`: base64 string or numeric array.
    Bytes,
    /// `map[string][]byte`.
    BytesMap,
    /// Required nested struct: null/missing binds the zero value.
    Object {
        go_type: &'static str,
        struct_name: &'static str,
        specs: &'static [Spec],
    },
    /// `*struct`: null/missing binds None.
    OptObject {
        go_type: &'static str,
        struct_name: &'static str,
        specs: &'static [Spec],
    },
    /// `[]struct`: null/missing binds empty.
    StructList {
        go_type: &'static str,
        struct_name: &'static str,
        specs: &'static [Spec],
    },
}

/// A bound field value, keyed by spec name in [`BoundMap`].
#[derive(Debug, Clone)]
pub enum Bound {
    Str(String),
    Bool(bool),
    I64(i64),
    U32(u32),
    OptInt(Option<i64>),
    StrList(Vec<String>),
    Bytes(Vec<u8>),
    BytesMap(HashMap<String, Vec<u8>>),
    Map(BoundMap),
    OptMap(Option<BoundMap>),
    StructList(Vec<BoundMap>),
}

#[derive(Debug, Clone, Default)]
pub struct BoundMap(HashMap<String, Bound>);

impl BoundMap {
    fn get(&self, name: &str) -> Option<&Bound> {
        self.0.get(name)
    }
    fn insert(&mut self, name: String, bound: Bound) {
        self.0.insert(name, bound);
    }
    pub fn take_string(&self, name: &str) -> String {
        match self.get(name) {
            Some(Bound::Str(s)) => s.clone(),
            _ => String::new(),
        }
    }
    pub fn take_bool(&self, name: &str) -> bool {
        match self.get(name) {
            Some(Bound::Bool(b)) => *b,
            _ => false,
        }
    }
    pub fn take_i64(&self, name: &str) -> i64 {
        match self.get(name) {
            Some(Bound::I64(n)) => *n,
            _ => 0,
        }
    }
    pub fn take_u32(&self, name: &str) -> u32 {
        match self.get(name) {
            Some(Bound::U32(n)) => *n,
            _ => 0,
        }
    }
    pub fn take_opt_i64(&self, name: &str) -> Option<i64> {
        match self.get(name) {
            Some(Bound::OptInt(n)) => *n,
            _ => None,
        }
    }
    pub fn take_str_list(&self, name: &str) -> Vec<String> {
        match self.get(name) {
            Some(Bound::StrList(v)) => v.clone(),
            _ => Vec::new(),
        }
    }
    pub fn take_bytes(&self, name: &str) -> Vec<u8> {
        match self.get(name) {
            Some(Bound::Bytes(v)) => v.clone(),
            _ => Vec::new(),
        }
    }
    pub fn take_bytes_map(&self, name: &str) -> HashMap<String, Vec<u8>> {
        match self.get(name) {
            Some(Bound::BytesMap(m)) => m.clone(),
            _ => HashMap::new(),
        }
    }
    pub fn contains(&self, name: &str) -> bool {
        self.0.contains_key(name)
    }
    pub fn take_map(&self, name: &str) -> BoundMap {
        match self.get(name) {
            Some(Bound::Map(m)) => m.clone(),
            _ => BoundMap::default(),
        }
    }
    pub fn take_opt_map(&self, name: &str) -> Option<BoundMap> {
        match self.get(name) {
            Some(Bound::OptMap(m)) => m.clone(),
            _ => None,
        }
    }
    pub fn take_struct_list(&self, name: &str) -> Vec<BoundMap> {
        match self.get(name) {
            Some(Bound::StructList(v)) => v.clone(),
            _ => Vec::new(),
        }
    }
}

fn value_word(v: &Value) -> &'static str {
    match v {
        Value::Null => "null",
        Value::Bool(_) => "bool",
        Value::Number(_) => "number",
        Value::Str(_) => "string",
        Value::Array(_) => "array",
        Value::Object(_) => "object",
    }
}

/// `UnmarshalTypeError` with struct-field context:
/// `json: cannot unmarshal {value} into Go struct field {S}.{path}.{field}
/// of type {type}`. `number_word` carries the literal for int targets.
fn type_error(
    struct_name: &str,
    path: &[String],
    field: &str,
    number_word: Option<&str>,
    v: &Value,
    go_type: &str,
) -> Error {
    let value = match (v, number_word) {
        (Value::Number(_), Some(lit)) => format!("number {lit}"),
        _ => value_word(v).to_string(),
    };
    let mut full = String::from(struct_name);
    for p in path {
        full.push('.');
        full.push_str(p);
    }
    full.push('.');
    full.push_str(field);
    err(format!(
        "decode request: json: cannot unmarshal {value} into Go struct field {full} of type {go_type}"
    ))
}

fn bind_uint8_element(
    v: &Value,
    struct_name: &str,
    path: &[String],
    field: &str,
) -> Result<u8, Error> {
    match v {
        Value::Null => Ok(0),
        Value::Number(lit) => lit
            .parse::<u8>()
            .map_err(|_| type_error(struct_name, path, field, Some(lit), v, "uint8")),
        _ => Err(type_error(struct_name, path, field, None, v, "uint8")),
    }
}

fn bind_bytes_value(
    v: &Value,
    struct_name: &str,
    path: &[String],
    field: &str,
    go_type: &str,
) -> Result<Vec<u8>, Error> {
    match v {
        Value::Null => Ok(Vec::new()),
        Value::Str(s) => crate::ssh::b64_decode_go(s.as_bytes())
            .map_err(|off| err(format!("decode request: {}", crate::ssh::b64_corrupt(off)))),
        Value::Array(items) => {
            let mut out = Vec::with_capacity(items.len());
            for item in items {
                out.push(bind_uint8_element(item, struct_name, path, field)?);
            }
            Ok(out)
        }
        _ => Err(type_error(struct_name, path, field, None, v, go_type)),
    }
}

fn bind_value(
    v: &Value,
    spec: &Spec,
    struct_name: &str,
    path: &[String],
    tolerant: bool,
) -> Result<Bound, Error> {
    // Error paths use the spec (struct field) name; values bind from any
    // fold-matching key.
    let field = spec.name;
    match &spec.kind {
        Kind::Str => match v {
            Value::Null => Ok(Bound::Str(String::new())),
            Value::Str(s) => Ok(Bound::Str(s.clone())),
            _ => Err(type_error(struct_name, path, field, None, v, "string")),
        },
        Kind::Bool => match v {
            Value::Null => Ok(Bound::Bool(false)),
            Value::Bool(b) => Ok(Bound::Bool(*b)),
            _ => Err(type_error(struct_name, path, field, None, v, "bool")),
        },
        Kind::I64 => match v {
            Value::Null => Ok(Bound::I64(0)),
            Value::Number(lit) => lit
                .parse::<i64>()
                .map(Bound::I64)
                .map_err(|_| type_error(struct_name, path, field, Some(lit), v, "int64")),
            _ => Err(type_error(struct_name, path, field, None, v, "int64")),
        },
        Kind::Int => match v {
            Value::Null => Ok(Bound::I64(0)),
            Value::Number(lit) => parse_go_int64(lit)
                .map(Bound::I64)
                .ok_or_else(|| type_error(struct_name, path, field, Some(lit), v, "int")),
            _ => Err(type_error(struct_name, path, field, None, v, "int")),
        },
        Kind::U32 => match v {
            Value::Null => Ok(Bound::U32(0)),
            Value::Number(lit) => parse_go_uint32(lit)
                .map(Bound::U32)
                .ok_or_else(|| type_error(struct_name, path, field, Some(lit), v, "uint32")),
            _ => Err(type_error(struct_name, path, field, None, v, "uint32")),
        },
        Kind::OptInt => match v {
            Value::Null => Ok(Bound::OptInt(None)),
            Value::Number(lit) => lit
                .parse::<i64>()
                .map(|n| Bound::OptInt(Some(n)))
                .map_err(|_| type_error(struct_name, path, field, Some(lit), v, "int")),
            _ => Err(type_error(struct_name, path, field, None, v, "int")),
        },
        Kind::StrList => match v {
            Value::Null => Ok(Bound::StrList(Vec::new())),
            Value::Array(items) => {
                let mut out = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        Value::Null => out.push(String::new()),
                        Value::Str(s) => out.push(s.clone()),
                        _ => {
                            return Err(type_error(struct_name, path, field, None, item, "string"))
                        }
                    }
                }
                Ok(Bound::StrList(out))
            }
            _ => Err(type_error(struct_name, path, field, None, v, "[]string")),
        },
        Kind::Bytes => bind_bytes_value(v, struct_name, path, field, "[]uint8").map(Bound::Bytes),
        Kind::BytesMap => match v {
            Value::Null => Ok(Bound::BytesMap(HashMap::new())),
            Value::Object(entries) => {
                let mut out = HashMap::with_capacity(entries.len());
                for (k, val) in entries {
                    out.insert(
                        k.clone(),
                        bind_bytes_value(val, struct_name, path, field, "[]uint8")?,
                    );
                }
                Ok(Bound::BytesMap(out))
            }
            _ => Err(type_error(
                struct_name,
                path,
                field,
                None,
                v,
                "map[string][]uint8",
            )),
        },
        Kind::Object {
            go_type,
            struct_name: nested,
            specs,
        } => match v {
            Value::Null => Ok(Bound::Map(BoundMap::default())),
            Value::Object(_) => {
                let mut child = path.to_vec();
                child.push(field.to_string());
                bind_struct(v, nested, &child, specs, tolerant).map(Bound::Map)
            }
            _ => Err(type_error(struct_name, path, field, None, v, go_type)),
        },
        Kind::OptObject {
            go_type,
            struct_name: nested,
            specs,
        } => match v {
            Value::Null => Ok(Bound::OptMap(None)),
            Value::Object(_) => {
                let mut child = path.to_vec();
                child.push(field.to_string());
                bind_struct(v, nested, &child, specs, tolerant).map(|m| Bound::OptMap(Some(m)))
            }
            _ => Err(type_error(struct_name, path, field, None, v, go_type)),
        },
        Kind::StructList {
            go_type,
            struct_name: nested,
            specs,
        } => match v {
            Value::Null => Ok(Bound::StructList(Vec::new())),
            Value::Array(items) => {
                let mut out = Vec::with_capacity(items.len());
                let mut child = path.to_vec();
                child.push(field.to_string());
                for item in items {
                    match item {
                        Value::Object(_) => {
                            out.push(bind_struct(item, nested, &child, specs, tolerant)?)
                        }
                        _ => return Err(type_error(struct_name, path, field, None, item, go_type)),
                    }
                }
                Ok(Bound::StructList(out))
            }
            _ => Err(type_error(struct_name, path, field, None, v, go_type)),
        },
    }
}

fn match_spec<'s>(specs: &'s [Spec], key: &str) -> Option<&'s Spec> {
    if let Some(spec) = specs.iter().find(|s| s.name == key) {
        return Some(spec);
    }
    let mut fold = specs.iter().filter(|s| s.name.eq_ignore_ascii_case(key));
    let first = fold.next()?;
    if fold.next().is_some() {
        // Several fields fold to this key: Go hides all of them.
        return None;
    }
    Some(first)
}

/// Ordered struct binding with exact `encoding/json` semantics: fields are
/// processed in value order (top level is pre-sorted like strictjson's
/// re-marshal, nested levels keep document order), the first error wins,
/// fold-matching keys all bind with last-wins, and unknown fields error
/// inline. `path` is the JSON field path from the root for messages.
/// Ordered struct binding with exact `encoding/json` semantics: fields are
/// processed in value order (top level is pre-sorted like strictjson's
/// re-marshal, nested levels keep document order), the first error wins,
/// fold-matching keys all bind with last-wins, and unknown fields error
/// inline (or are ignored when `tolerant`, like plain `Unmarshal`).
/// `path` is the JSON field path from the root for messages.
pub fn bind_struct(
    v: &Value,
    struct_name: &'static str,
    path: &[String],
    specs: &[Spec],
    tolerant: bool,
) -> Result<BoundMap, Error> {
    let fields = match v {
        Value::Object(fields) => fields,
        _ => return Err(err("decode request: expected object")),
    };
    let mut out = BoundMap::default();
    for (key, val) in fields.iter() {
        let Some(spec) = match_spec(specs, key) else {
            if tolerant {
                continue;
            }
            return Err(err(format!(
                "decode request: json: unknown field {}",
                go_quote(key)
            )));
        };
        // Null leaves the zero value, exactly like a missing field (unknown
        // nulls still error above). Array elements and map values handle
        // their own nulls inside their kinds.
        if val.is_null() {
            continue;
        }
        out.insert(
            spec.name.to_string(),
            bind_value(val, spec, struct_name, path, tolerant)?,
        );
    }
    Ok(out)
}

/// Strict binding entry point: top-level struct with an empty path.
pub fn bind_root(
    v: &Value,
    struct_name: &'static str,
    specs: &[Spec],
    tolerant: bool,
) -> Result<BoundMap, Error> {
    bind_struct(v, struct_name, &[], specs, tolerant)
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
    fn strict_rejects_float64_overflowing_numbers() {
        // Go converts every scanned number to float64: overflow fails the
        // gate with the top-level field name, at any nesting depth.
        for (input, field, lit) in [
            (r#"{"z":1e999}"#, "z", "1e999"),
            (r#"{"a":{"b":-2e999}}"#, "a", "-2e999"),
            (r#"{"a":[1.8e308]}"#, "a", "1.8e308"),
            (r#"{"m":1,"z":1e309,"a":1}"#, "z", "1e309"),
            (r#"{"z":1e999,"y":2e999}"#, "z", "1e999"),
            (
                r#"{"z":1.7976931348623159e308}"#,
                "z",
                "1.7976931348623159e308",
            ),
        ] {
            let e = decode_strict(input.as_bytes()).unwrap_err();
            assert_eq!(
                e.0,
                format!(
                    "decode request field \"{field}\": json: cannot unmarshal number {lit} into Go value of type float64"
                ),
                "{input}"
            );
        }
        // Underflow to zero and the largest finite double stay accepted.
        for input in [
            r#"{"z":1e-999}"#,
            r#"{"z":-1e-999}"#,
            r#"{"z":4e-324}"#,
            r#"{"z":0e999}"#,
            r#"{"z":1.7976931348623157e308}"#,
            r#"{"z":1e308}"#,
            r#"{"z":42}"#,
        ] {
            assert!(decode_strict(input.as_bytes()).is_ok(), "{input}");
        }
        // Walk order: a duplicate earlier in the same value beats the
        // float, but a float in an earlier field beats a later duplicate.
        let e = decode_strict(br#"{"a":{"x":1,"x":2,"y":1e999}}"#).unwrap_err();
        assert!(e.0.contains("duplicate"), "{e}");
        let e = decode_strict(br#"{"a":{"x":1,"x":2},"b":1e999}"#).unwrap_err();
        assert!(e.0.contains("duplicate"), "{e}");
        let e = decode_strict(br#"{"b":1e999,"a":{"x":1,"x":2}}"#).unwrap_err();
        assert!(e.0.contains("cannot unmarshal number 1e999"), "{e}");
        // Root and trailing `Token`s convert too: bare overflow errors.
        for (input, want) in [
            ("1e999", "decode request: json: cannot unmarshal number 1e999 into Go value of type float64"),
            ("{} 1e999", "decode request: json: cannot unmarshal number 1e999 into Go value of type float64"),
            (r#"{"a":1} -2e999"#, "decode request: json: cannot unmarshal number -2e999 into Go value of type float64"),
            ("42", "request must be one JSON object"),
            ("{} 42", "request must contain exactly one JSON object"),
        ] {
            assert_eq!(decode_strict(input.as_bytes()).unwrap_err().0, want, "{input}");
        }
        // Tolerant decode keeps accepting: machine parsers skip values.
        assert!(decode_tolerant(br#"{"z":1e999}"#.as_ref()).is_ok());
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
    fn strict_scanner_messages_match_go_toolchain() {
        // Exact `strictjson.Decode` message shapes, pinned against the
        // pinned Go toolchain (see the corpus4 differential).
        for (input, want) in [
            ("", "decode request: EOF"),
            ("  ", "decode request: EOF"),
            ("[", "request must be one JSON object"),
            ("[1]", "request must be one JSON object"),
            ("1", "request must be one JSON object"),
            ("\"a", "decode request: unexpected EOF"),
            ("\"a\"", "request must be one JSON object"),
            ("tru", "decode request: unexpected EOF"),
            ("true", "request must be one JSON object"),
            ("-", "decode request: unexpected EOF"),
            ("-x", "decode request: invalid character 'x' in numeric literal"),
            ("x", "decode request: invalid character 'x' looking for beginning of value"),
            (",", "decode request: invalid character ',' looking for beginning of value"),
            ("]", "decode request: invalid character ']' looking for beginning of value"),
            ("{", "decode request: EOF"),
            ("{]", "decode request: invalid character ']'"),
            ("{1", "decode request: invalid character '1'"),
            ("{,", "decode request: invalid character ','"),
            ("{\"a\":1,}", "decode request: invalid character '}' looking for beginning of object key string"),
            ("{\"a\":1,2", "decode request: invalid character '2' looking for beginning of object key string"),
            ("{\"a\":1,", "decode request: EOF"),
            ("{\"a\"", "decode request field \"a\": EOF"),
            ("{\"a\" ", "decode request field \"a\": EOF"),
            ("{\"a\" 1}", "decode request field \"a\": expected colon after object key"),
            ("{\"a\":", "decode request field \"a\": EOF"),
            ("{\"a\":x}", "decode request field \"a\": invalid character 'x' looking for beginning of value"),
            ("{\"a\":tru}", "decode request field \"a\": invalid character '}' in literal true (expecting 'e')"),
            ("{\"a\":1", "decode request: EOF"),
            ("{\"a\":1 ", "decode request: EOF"),
            ("{\"a\":1]", "decode request: invalid character ']' after object key:value pair"),
            ("{\"a\":1x}", "decode request: invalid character 'x' after object key:value pair"),
            ("{\"a\":01}", "decode request: invalid character '1' after object key:value pair"),
            ("{\"a\":1.5e+}", "decode request field \"a\": invalid character '}' in exponent of numeric literal"),
            ("{\"a\":1.5e+", "decode request field \"a\": unexpected EOF"),
            ("{\"a\":1.5ex}", "decode request field \"a\": invalid character 'x' in exponent of numeric literal"),
            ("{\"a\":1.x}", "decode request field \"a\": invalid character 'x' after decimal point in numeric literal"),
            ("{\"a\":\"b\\x}", "decode request field \"a\": invalid character 'x' in string escape code"),
            ("{\"a\":\"b\\u12}", "decode request field \"a\": invalid character '}' in \\u hexadecimal character escape"),
            ("{\"a\":1}{", "request must contain exactly one JSON object"),
            ("{\"a\":1}[", "request must contain exactly one JSON object"),
            ("{\"a\":1}1", "request must contain exactly one JSON object"),
            ("{\"a\":1}1x", "request must contain exactly one JSON object"),
            ("{\"a\":1}}", "decode request: invalid character '}' looking for beginning of value"),
            ("{\"a\":1},", "decode request: invalid character ',' looking for beginning of value"),
            ("{\"a\":1}\"a", "decode request: unexpected EOF"),
            ("{\"a\":1}tru", "decode request: unexpected EOF"),
            ("{\"a\":1}-x", "decode request: invalid character 'x' in numeric literal"),
            ("{\"f\":{\"g\":1", "decode request field \"f\": unexpected EOF"),
            ("{\"f\":{\"g\":1,", "decode request field \"f\": unexpected EOF"),
            ("{\"f\":{\"g\"", "decode request field \"f\": unexpected EOF"),
            ("{\"f\":{\"g\" 1}", "decode request field \"f\": invalid character '1' after object key"),
            ("{\"f\":[1 2]}", "decode request field \"f\": invalid character '2' after array element"),
            ("{\"f\":[01]}", "decode request field \"f\": invalid character '1' after array element"),
            ("{\"f\":[x]}", "decode request field \"f\": invalid character 'x' looking for beginning of value"),
            ("{\"f\":{1}}", "decode request field \"f\": invalid character '1' looking for beginning of object key string"),
            ("{\"a\":1,\"a\":2}", "duplicate request field \"a\""),
            ("{\"a\":{\"x\":1,\"x\":2}}", "duplicate request field \"x\""),
            // A scan error anywhere in the value beats a deferred nested dup.
            ("{\"a\":{\"x\":1,\"x\":truX}}", "decode request field \"a\": invalid character 'X' in literal true (expecting 'e')"),
            // A top duplicate reports before the later value is scanned.
            ("{\"a\":1,\"a\":{\"x\":truX}}", "duplicate request field \"a\""),
        ] {
            let got = decode_strict(input.as_bytes()).unwrap_err().0;
            assert_eq!(got, want, "input={input:?}");
        }
    }

    #[test]
    fn strict_depth_bounds_match_reject_duplicate_keys() {
        // strictjson rejects values nested past depth 100 (102 opens) once
        // the top-level value completes; the scanner itself fails past 10000.
        for (n, ok) in [(101, true), (102, false), (103, false)] {
            let doc = format!("{{\"n\":{}}}", "[".repeat(n) + &"]".repeat(n));
            assert_eq!(
                decode_strict(doc.as_bytes()).is_ok(),
                ok,
                "empty arrays n={n}"
            );
        }
        // A scalar counts one deeper than its containers.
        for (n, ok) in [(100, true), (101, false), (102, false)] {
            let doc = format!("{{\"n\":{}1{}}}", "[".repeat(n), "]".repeat(n));
            assert_eq!(
                decode_strict(doc.as_bytes()).is_ok(),
                ok,
                "scalar arrays n={n}"
            );
            let doc = format!("{{\"n\":{}}}", "{\"a\":".repeat(n) + "1" + &"}".repeat(n));
            assert_eq!(decode_strict(doc.as_bytes()).is_ok(), ok, "objects n={n}");
        }
        let doc = format!("{{\"n\":{}}}", "[".repeat(102) + &"]".repeat(102));
        assert_eq!(
            decode_strict(doc.as_bytes()).unwrap_err().0,
            "decode request field \"n\": request is nested too deeply"
        );
        for (n, want) in [
            (
                10000,
                "decode request field \"n\": request is nested too deeply",
            ),
            (
                10001,
                "decode request field \"n\": invalid character '[' exceeded max depth",
            ),
        ] {
            let doc = format!("{{\"n\":{}}}", "[".repeat(n) + &"]".repeat(n));
            assert_eq!(decode_strict(doc.as_bytes()).unwrap_err().0, want, "n={n}");
        }
    }

    const WIDGET_SPECS: &[Spec] = &[
        Spec {
            name: "id",
            kind: Kind::Str,
        },
        Spec {
            name: "count",
            kind: Kind::I64,
        },
        Spec {
            name: "flag",
            kind: Kind::Bool,
        },
    ];

    fn bind_widget(raw: &[u8]) -> Result<BoundMap, Error> {
        let v = decode_strict(raw)?;
        bind_root(&v, "Widget", WIDGET_SPECS, false)
    }

    #[test]
    fn binder_matches_go_field_rules() {
        let m = bind_widget(br#"{"ID":"x","Count":3,"Flag":true}"#).unwrap();
        assert_eq!(m.take_string("id"), "x"); // case-insensitive fallback
        assert_eq!(m.take_i64("count"), 3);
        assert!(m.take_bool("flag"));

        // Unknown field, Go-quoted.
        let err = bind_widget(br#"{"id":"x","bogus":1}"#).unwrap_err();
        assert_eq!(err.0, "decode request: json: unknown field \"bogus\"");

        // Null leaves the zero value.
        let m = bind_widget(br#"{"id":null,"count":null}"#).unwrap();
        assert_eq!(m.take_string("id"), "");
        assert_eq!(m.take_i64("count"), 0);

        // Exact Go mismatch messages, literal echoed for int targets only.
        for (bad, want) in [
            (
                br#"{"count":1.5}"#.as_slice(),
                "decode request: json: cannot unmarshal number 1.5 into Go struct field Widget.count of type int64",
            ),
            (
                br#"{"count":"3"}"#,
                "decode request: json: cannot unmarshal string into Go struct field Widget.count of type int64",
            ),
            (
                br#"{"count":true}"#,
                "decode request: json: cannot unmarshal bool into Go struct field Widget.count of type int64",
            ),
            (
                br#"{"count":1e3}"#,
                "decode request: json: cannot unmarshal number 1e3 into Go struct field Widget.count of type int64",
            ),
            (
                br#"{"id":7}"#,
                "decode request: json: cannot unmarshal number into Go struct field Widget.id of type string",
            ),
        ] {
            let err = bind_widget(bad).unwrap_err();
            assert_eq!(err.0, want);
        }

        // Fold-matching keys all bind, last in sorted order wins; no error.
        let m = bind_widget(br#"{"ID":"a","id":"b"}"#).unwrap();
        assert_eq!(m.take_string("id"), "b");

        // Sorted top-level order decides between two faults.
        let err = bind_widget(br#"{"id":1,"count":"x"}"#).unwrap_err();
        assert!(err.0.contains("Widget.count"), "{err:?}");
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

    #[test]
    fn tolerant_decode_matches_go_nesting_limit() {
        // encoding/json fails opening the 10001st container; scalars at
        // depth are fine. Pinned against the pinned Go toolchain.
        for (n, ok) in [(9999, true), (10000, false)] {
            let doc = format!("{{\"n\":{}}}", "[".repeat(n) + &"]".repeat(n));
            assert_eq!(decode_tolerant(doc.as_bytes()).is_ok(), ok, "n={n}");
            let doc = format!("{{\"n\":{}1{}}}", "[".repeat(n), "]".repeat(n));
            assert_eq!(decode_tolerant(doc.as_bytes()).is_ok(), ok, "scalar n={n}");
        }
        // Strict requests keep strictjson's much smaller depth-100 scan.
        let doc = format!("{{\"n\":{}}}", "[".repeat(102) + &"]".repeat(102));
        assert!(decode_strict(doc.as_bytes()).is_err());
    }
}
