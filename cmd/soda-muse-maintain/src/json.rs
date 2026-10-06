use super::config_wire::invalid_character;

pub(crate) struct JsonParser<'a> {
    pub(crate) bytes: &'a [u8],
    pub(crate) pos: usize,
}

impl<'a> JsonParser<'a> {
    pub(crate) fn new(bytes: &'a [u8]) -> Self {
        JsonParser { bytes, pos: 0 }
    }

    pub(crate) fn eof(&self) -> bool {
        self.pos >= self.bytes.len()
    }

    pub(crate) fn peek(&self) -> Option<u8> {
        self.bytes.get(self.pos).copied()
    }

    pub(crate) fn peek_char(&self) -> String {
        match self.peek() {
            Some(b'\'') => String::from("\\'"),
            Some(b) if (b as char).is_ascii_graphic() || b == b' ' => format!("{}", b as char),
            Some(b'\n') => String::from("\\n"),
            Some(b'\r') => String::from("\\r"),
            Some(b'\t') => String::from("\\t"),
            Some(b) => format!("\\x{b:02x}"),
            None => String::from("EOF"),
        }
    }

    pub(crate) fn bump(&mut self) {
        self.pos += 1;
    }

    pub(crate) fn skip_ws(&mut self) {
        while matches!(self.peek(), Some(b' ' | b'\t' | b'\n' | b'\r')) {
            self.pos += 1;
        }
    }

    pub(crate) fn value_kind(&mut self) -> Result<&'static str, String> {
        match self.peek() {
            Some(b'"') => {
                self.parse_string()?;
                Ok("string")
            }
            Some(b't') | Some(b'f') | Some(b'n') => {
                let lit = self.parse_literal()?;
                Ok(if lit == "null" { "null" } else { "bool" })
            }
            Some(b'-') | Some(b'0'..=b'9') => {
                self.scan_number()?;
                Ok("number")
            }
            Some(b'[') => {
                self.skip_value()?;
                Ok("array")
            }
            Some(b'{') => {
                self.skip_value()?;
                Ok("object")
            }
            _ => Err(invalid_character(self, "looking for beginning of value")),
        }
    }

    pub(crate) fn parse_literal(&mut self) -> Result<&'static str, String> {
        for lit in ["true", "false", "null"] {
            if self.bytes[self.pos..].starts_with(lit.as_bytes()) {
                self.pos += lit.len();
                return Ok(lit);
            }
        }
        let full: &[u8] = match self.peek() {
            Some(b't') => b"true",
            Some(b'f') => b"false",
            Some(b'n') => b"null",
            _ => return Err(invalid_character(self, "looking for beginning of value")),
        };
        // Walk the literal to find the offending character.
        let mut i = 1;
        while i < full.len()
            && self.pos + i < self.bytes.len()
            && self.bytes[self.pos + i] == full[i]
        {
            i += 1;
        }
        self.pos += i;
        if self.eof() {
            return Err(String::from("unexpected EOF"));
        }
        let c = self.peek_char();
        let want = std::str::from_utf8(full).unwrap();
        let exp = full
            .get(i)
            .map(|b| format!("'{}'", *b as char))
            .unwrap_or_default();
        Err(format!(
            "invalid character '{c}' in literal {want} (expecting {exp})"
        ))
    }

    pub(crate) fn scan_number(&mut self) -> Result<(), String> {
        if self.peek() == Some(b'-') {
            self.bump();
            if self.eof() {
                return Err(String::from("unexpected EOF"));
            }
        }
        match self.peek() {
            Some(b'0') => {
                self.bump();
            }
            Some(b'1'..=b'9') => {
                while matches!(self.peek(), Some(b'0'..=b'9')) {
                    self.bump();
                }
            }
            _ => return Err(self.numeric_error()),
        }
        if self.peek() == Some(b'.') {
            self.bump();
            if !matches!(self.peek(), Some(b'0'..=b'9')) {
                return Err(self.numeric_error());
            }
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.bump();
            }
        }
        if matches!(self.peek(), Some(b'e' | b'E')) {
            self.bump();
            if matches!(self.peek(), Some(b'+' | b'-')) {
                self.bump();
            }
            if !matches!(self.peek(), Some(b'0'..=b'9')) {
                return Err(self.numeric_error());
            }
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.bump();
            }
        }
        Ok(())
    }

    fn numeric_error(&self) -> String {
        if self.eof() {
            return String::from("unexpected EOF");
        }
        format!(
            "invalid character '{}' in numeric literal",
            self.peek_char()
        )
    }

    pub(crate) fn skip_value(&mut self) -> Result<(), String> {
        match self.peek() {
            Some(b'"') => {
                self.parse_string()?;
                Ok(())
            }
            Some(b't') | Some(b'f') | Some(b'n') => {
                self.parse_literal()?;
                Ok(())
            }
            Some(b'-') | Some(b'0'..=b'9') => self.scan_number(),
            Some(b'{') | Some(b'[') => {
                let open = self.peek().unwrap();
                let close = if open == b'{' { b'}' } else { b']' };
                // Go rejects trailing commas and names the separator by
                // container: "after array element" or "after object
                // key:value pair".
                let after = if open == b'{' {
                    "after object key:value pair"
                } else {
                    "after array element"
                };
                self.bump();
                self.skip_ws();
                if self.eof() {
                    return Err(String::from("unexpected EOF"));
                }
                if self.peek() == Some(close) {
                    self.bump();
                    return Ok(());
                }
                loop {
                    if open == b'{' {
                        if self.peek() != Some(b'"') {
                            return Err(invalid_character(
                                self,
                                "looking for beginning of object key string",
                            ));
                        }
                        self.parse_string()?;
                        self.skip_ws();
                        if self.peek() != Some(b':') {
                            if self.eof() {
                                return Err(String::from("unexpected EOF"));
                            }
                            return Err(invalid_character(self, "after object key"));
                        }
                        self.bump();
                        self.skip_ws();
                    }
                    self.skip_value()?;
                    self.skip_ws();
                    if self.eof() {
                        return Err(String::from("unexpected EOF"));
                    }
                    if self.peek() == Some(b',') {
                        self.bump();
                        self.skip_ws();
                        if self.eof() {
                            return Err(String::from("unexpected EOF"));
                        }
                        // No close check here: a closer after a comma is a
                        // trailing comma, rejected by element parsing above.
                        continue;
                    }
                    if self.peek() == Some(close) {
                        self.bump();
                        return Ok(());
                    }
                    return Err(invalid_character(self, after));
                }
            }
            _ => {
                if self.eof() {
                    return Err(String::from("unexpected EOF"));
                }
                Err(invalid_character(self, "looking for beginning of value"))
            }
        }
    }
}

// decode_first_char decodes one UTF-8 unit, substituting U+FFFD for bytes
// encoding/json would also refuse to fail on.
pub(crate) fn decode_first_char(rest: &[u8]) -> (char, usize) {
    if rest.is_empty() {
        return ('\u{FFFD}', 0);
    }
    let b0 = rest[0];
    let width = if b0 < 0x80 {
        return (b0 as char, 1);
    } else if (0xc2..=0xdf).contains(&b0) {
        2
    } else if (0xe0..=0xef).contains(&b0) {
        3
    } else if (0xf0..=0xf4).contains(&b0) {
        4
    } else {
        return ('\u{FFFD}', 1);
    };
    if rest.len() < width || !rest[1..width].iter().all(|b| (0x80..=0xbf).contains(b)) {
        return ('\u{FFFD}', 1);
    }
    match std::str::from_utf8(&rest[..width]) {
        Ok(s) => (s.chars().next().unwrap(), width),
        Err(_) => ('\u{FFFD}', 1),
    }
}
