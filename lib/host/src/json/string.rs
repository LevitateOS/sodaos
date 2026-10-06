use super::{err, Error, Parser};

impl<'a> Parser<'a> {
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

    pub(super) fn parse_string(&mut self) -> Result<String, Error> {
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
}
