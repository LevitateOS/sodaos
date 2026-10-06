use super::{close_frame, err, go_quote, Error, Frame, Parser, Value};

const MAX_NESTING: u32 = 101;

impl<'a> Parser<'a> {
    pub(super) fn new(bytes: &'a [u8], strict: bool) -> Self {
        Parser {
            bytes,
            pos: 0,
            strict_field: strict.then_some(None),
            deferred: None,
        }
    }

    pub(super) fn eof(&self) -> bool {
        self.pos >= self.bytes.len()
    }

    pub(super) fn peek(&self) -> Option<u8> {
        self.bytes.get(self.pos).copied()
    }

    pub(super) fn skip_ws(&mut self) {
        while matches!(self.peek(), Some(b' ' | b'\t' | b'\n' | b'\r')) {
            self.pos += 1;
        }
    }

    fn strict(&self) -> bool {
        self.strict_field.is_some()
    }

    pub(super) fn field_ctx(&self) -> String {
        self.strict_field
            .as_ref()
            .and_then(|f| f.clone())
            .map(|f| format!("decode request field {}: ", go_quote(&f)))
            .unwrap_or_else(|| "decode request: ".to_string())
    }

    pub(super) fn parse_value(&mut self, depth: u32) -> Result<Value, Error> {
        debug_assert_eq!(depth, 0);
        self.run_machine()
    }

    pub(super) fn parse_object(&mut self, depth: u32) -> Result<Value, Error> {
        debug_assert_eq!(depth, 0);
        match self.run_machine()? {
            v @ Value::Object(_) => Ok(v),
            _ => Err(err("decode request: expected object")),
        }
    }

    pub(super) fn parse_literal(&mut self, lit: &str, v: Value) -> Result<Value, Error> {
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
    pub(super) fn quote_byte(c: u8) -> String {
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
}
