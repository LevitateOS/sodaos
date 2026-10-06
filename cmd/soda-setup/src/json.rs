#[derive(Debug)]
#[allow(dead_code)]
pub(crate) enum JsonValue {
    Null,
    Bool(bool),
    Number(String),
    Str(String),
    Array(Vec<JsonValue>),
    Object(Vec<(String, JsonValue)>),
}

impl JsonValue {
    pub(crate) fn parse(text: &str) -> Result<JsonValue, String> {
        let mut parser = JsonParser {
            bytes: text.as_bytes(),
            pos: 0,
        };
        parser.skip_ws();
        let value = parser.parse_value()?;
        parser.skip_ws();
        if parser.pos != parser.bytes.len() {
            return Err("invalid Forgejo response".to_string());
        }
        Ok(value)
    }

    pub(crate) fn as_object(&self) -> Option<&[(String, JsonValue)]> {
        match self {
            JsonValue::Object(entries) => Some(entries),
            _ => None,
        }
    }

    pub(crate) fn as_bool(&self) -> Option<bool> {
        match self {
            JsonValue::Bool(b) => Some(*b),
            _ => None,
        }
    }

    pub(crate) fn as_i64(&self) -> Option<i64> {
        match self {
            JsonValue::Number(raw) => {
                let digits = raw.strip_prefix('-').unwrap_or(raw);
                digits
                    .bytes()
                    .all(|b| b.is_ascii_digit())
                    .then(|| raw.parse().ok())?
            }
            _ => None,
        }
    }
}

struct JsonParser<'a> {
    bytes: &'a [u8],
    pos: usize,
}

impl<'a> JsonParser<'a> {
    fn skip_ws(&mut self) {
        while self.pos < self.bytes.len()
            && matches!(self.bytes[self.pos], b' ' | b'\t' | b'\n' | b'\r')
        {
            self.pos += 1;
        }
    }

    fn parse_value(&mut self) -> Result<JsonValue, String> {
        let invalid = || "invalid Forgejo response".to_string();
        let byte = *self.bytes.get(self.pos).ok_or_else(invalid)?;
        match byte {
            b'{' => self.parse_object(),
            b'[' => self.parse_array(),
            b'"' => Ok(JsonValue::Str(self.parse_string()?)),
            b't' => self.parse_literal("true", JsonValue::Bool(true)),
            b'f' => self.parse_literal("false", JsonValue::Bool(false)),
            b'n' => self.parse_literal("null", JsonValue::Null),
            b'-' | b'0'..=b'9' => Ok(JsonValue::Number(self.parse_number()?)),
            _ => Err(invalid()),
        }
    }

    fn parse_literal(&mut self, word: &str, value: JsonValue) -> Result<JsonValue, String> {
        if self.bytes[self.pos..].starts_with(word.as_bytes()) {
            self.pos += word.len();
            Ok(value)
        } else {
            Err("invalid Forgejo response".to_string())
        }
    }

    fn parse_object(&mut self) -> Result<JsonValue, String> {
        let invalid = || "invalid Forgejo response".to_string();
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
                return Err(invalid());
            }
            let key = self.parse_string()?;
            self.skip_ws();
            if self.bytes.get(self.pos) != Some(&b':') {
                return Err(invalid());
            }
            self.pos += 1;
            self.skip_ws();
            let value = self.parse_value()?;
            entries.push((key, value));
            self.skip_ws();
            match self.bytes.get(self.pos) {
                Some(b',') => {
                    self.pos += 1;
                }
                Some(b'}') => {
                    self.pos += 1;
                    return Ok(JsonValue::Object(entries));
                }
                _ => return Err(invalid()),
            }
        }
    }

    fn parse_array(&mut self) -> Result<JsonValue, String> {
        let invalid = || "invalid Forgejo response".to_string();
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
                Some(b',') => {
                    self.pos += 1;
                }
                Some(b']') => {
                    self.pos += 1;
                    return Ok(JsonValue::Array(items));
                }
                _ => return Err(invalid()),
            }
        }
    }

    fn parse_string(&mut self) -> Result<String, String> {
        let invalid = || "invalid Forgejo response".to_string();
        self.pos += 1;
        let mut out: Vec<u8> = Vec::new();
        loop {
            let byte = *self.bytes.get(self.pos).ok_or_else(invalid)?;
            self.pos += 1;
            match byte {
                b'"' => break,
                b'\\' => {
                    let esc = *self.bytes.get(self.pos).ok_or_else(invalid)?;
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
                                return Err(invalid());
                            }
                            let hex = std::str::from_utf8(&self.bytes[self.pos..self.pos + 4])
                                .map_err(|_| invalid())?;
                            let mut code = u32::from_str_radix(hex, 16).map_err(|_| invalid())?;
                            self.pos += 4;
                            if (0xd800..0xdc00).contains(&code)
                                && self.bytes.get(self.pos..self.pos + 2) == Some(b"\\u".as_slice())
                            {
                                let low_hex =
                                    std::str::from_utf8(&self.bytes[self.pos + 2..self.pos + 6])
                                        .map_err(|_| invalid())?;
                                let low =
                                    u32::from_str_radix(low_hex, 16).map_err(|_| invalid())?;
                                if (0xdc00..0xe000).contains(&low) {
                                    code = 0x10000 + ((code - 0xd800) << 10) + (low - 0xdc00);
                                    self.pos += 6;
                                }
                            }
                            let ch = char::from_u32(code).ok_or_else(invalid)?;
                            let mut buf = [0u8; 4];
                            out.extend_from_slice(ch.encode_utf8(&mut buf).as_bytes());
                        }
                        _ => return Err(invalid()),
                    }
                }
                0x00..=0x1f => return Err(invalid()),
                _ => out.push(byte),
            }
        }
        String::from_utf8(out).map_err(|_| invalid())
    }

    fn parse_number(&mut self) -> Result<String, String> {
        let invalid = || "invalid Forgejo response".to_string();
        let start = self.pos;
        if self.bytes.get(self.pos) == Some(&b'-') {
            self.pos += 1;
        }
        match self.bytes.get(self.pos) {
            Some(b'0') => {
                self.pos += 1;
            }
            Some(b'1'..=b'9') => {
                while matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                    self.pos += 1;
                }
            }
            _ => return Err(invalid()),
        }
        if self.bytes.get(self.pos) == Some(&b'.') {
            self.pos += 1;
            if !matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                return Err(invalid());
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
                return Err(invalid());
            }
            while matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
        }
        std::str::from_utf8(&self.bytes[start..self.pos])
            .map(|s| s.to_string())
            .map_err(|_| invalid())
    }
}

pub(crate) fn push_json_string(out: &mut String, value: &str) {
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
            c if (c as u32) < 0x20 => {
                out.push_str(&format!("\\u{:04x}", c as u32));
            }
            c => out.push(c),
        }
    }
    out.push('"');
}
