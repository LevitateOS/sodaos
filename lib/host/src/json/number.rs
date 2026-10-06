use super::{err, Error, Parser};

impl<'a> Parser<'a> {
    pub(super) fn parse_number(&mut self) -> Result<String, Error> {
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

/// `strconv.ParseInt(s, 10, 64)`: Rust's parse rejects the leading `+`
/// that Go accepts, so strip one first.
pub fn parse_go_int64(s: &str) -> Option<i64> {
    s.strip_prefix('+').unwrap_or(s).parse::<i64>().ok()
}

/// `strconv.ParseUint(s, 10, 32)`: same leading-`+` rule as above.
pub fn parse_go_uint32(s: &str) -> Option<u32> {
    s.strip_prefix('+').unwrap_or(s).parse::<u32>().ok()
}
