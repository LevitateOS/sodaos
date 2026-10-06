use super::config_wire::invalid_character;
use super::json::{decode_first_char, JsonParser};

impl<'a> JsonParser<'a> {
    // parse_string decodes with Go's leniency: lone surrogates become U+FFFD.
    pub(crate) fn parse_string(&mut self) -> Result<String, String> {
        if self.peek() != Some(b'"') {
            return Err(invalid_character(self, "looking for beginning of value"));
        }
        self.bump();
        let mut out = String::new();
        loop {
            if self.eof() {
                return Err(String::from("unexpected EOF"));
            }
            match self.bytes[self.pos] {
                b'"' => {
                    self.bump();
                    return Ok(out);
                }
                b'\\' => {
                    self.bump();
                    if self.eof() {
                        return Err(String::from("unexpected EOF"));
                    }
                    match self.bytes[self.pos] {
                        b'"' => out.push('"'),
                        b'\\' => out.push('\\'),
                        b'/' => out.push('/'),
                        b'b' => out.push('\u{0008}'),
                        b'f' => out.push('\u{000c}'),
                        b'n' => out.push('\n'),
                        b'r' => out.push('\r'),
                        b't' => out.push('\t'),
                        b'u' => {
                            // The escape consumes itself; continue below
                            // without the shared bump.
                            if self.pos + 4 >= self.bytes.len() {
                                return Err(String::from("unexpected EOF"));
                            }
                            let hex = &self.bytes[self.pos + 1..self.pos + 5];
                            if !hex.iter().all(|b| b.is_ascii_hexdigit()) {
                                let seq = String::from_utf8_lossy(
                                    &self.bytes[self.pos - 1..self.pos + 5],
                                )
                                .into_owned();
                                return Err(format!("invalid escape sequence `{seq}` in string"));
                            }
                            let cp =
                                u32::from_str_radix(std::str::from_utf8(hex).unwrap(), 16).unwrap();
                            self.pos += 5;
                            // Combine valid surrogate pairs; lone halves
                            // become U+FFFD exactly like encoding/json.
                            if (0xd800..0xdc00).contains(&cp) {
                                self.surrogate_tail(cp, &mut out)?;
                                continue;
                            } else if (0xd800..0xe000).contains(&cp) {
                                out.push('\u{FFFD}');
                            } else {
                                out.push(char::from_u32(cp).unwrap_or('\u{FFFD}'));
                            }
                            continue;
                        }
                        _ => {
                            let seq =
                                String::from_utf8_lossy(&self.bytes[self.pos - 1..self.pos + 1])
                                    .into_owned();
                            return Err(format!("invalid escape sequence `{seq}` in string"));
                        }
                    }
                    self.bump();
                }
                0x00..=0x1f => {
                    return Err(format!(
                        "invalid character '{}' in string",
                        self.peek_char()
                    ));
                }
                _ => {
                    // Invalid UTF-8 becomes U+FFFD like encoding/json.
                    let (c, n) = decode_first_char(&self.bytes[self.pos..]);
                    out.push(c);
                    self.pos += n;
                }
            }
        }
    }

    fn surrogate_tail(&mut self, high: u32, out: &mut String) -> Result<(), String> {
        if self.pos + 1 >= self.bytes.len()
            || self.bytes[self.pos] != b'\\'
            || self.bytes[self.pos + 1] != b'u'
        {
            out.push('\u{FFFD}');
            return Ok(());
        }
        if self.pos + 5 >= self.bytes.len() {
            return Err(String::from("unexpected EOF"));
        }
        let hex = &self.bytes[self.pos + 2..self.pos + 6];
        if !hex.iter().all(|b| b.is_ascii_hexdigit()) {
            let seq = String::from_utf8_lossy(&self.bytes[self.pos..self.pos + 6]).into_owned();
            return Err(format!("invalid escape sequence `{seq}` in string"));
        }
        let lo = u32::from_str_radix(std::str::from_utf8(hex).unwrap(), 16).unwrap();
        self.pos += 6;
        if (0xdc00..0xe000).contains(&lo) {
            let pair = 0x10000 + ((high - 0xd800) << 10) + (lo - 0xdc00);
            out.push(char::from_u32(pair).unwrap());
        } else {
            out.push('\u{FFFD}');
            out.push(char::from_u32(lo).unwrap_or('\u{FFFD}'));
        }
        Ok(())
    }
}
