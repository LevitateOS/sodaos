//! Restricted canonical-SVG cursor and document gate.

pub(crate) type Attrs = Vec<(String, String)>;

struct Cursor<'a> {
    bytes: &'a [u8],
    pos: usize,
}

impl<'a> Cursor<'a> {
    fn new(text: &'a str) -> Cursor<'a> {
        Cursor {
            bytes: text.as_bytes(),
            pos: 0,
        }
    }

    fn eof(&self) -> bool {
        self.pos >= self.bytes.len()
    }

    fn starts_with(&self, needle: &str) -> bool {
        self.bytes[self.pos..].starts_with(needle.as_bytes())
    }

    fn consume(&mut self, needle: &str) -> bool {
        if self.starts_with(needle) {
            self.pos += needle.len();
            true
        } else {
            false
        }
    }

    fn skip_ws(&mut self) {
        while !self.eof() && matches!(self.bytes[self.pos], b' ' | b'\t' | b'\n' | b'\r') {
            self.pos += 1;
        }
    }

    fn skip_comment(&mut self) -> Result<(), String> {
        debug_assert!(self.starts_with("<!--"));
        self.pos += 4;
        while !self.eof() && !self.starts_with("-->") {
            self.pos += 1;
        }
        if !self.consume("-->") {
            return Err("Unterminated emblem comment".to_string());
        }
        Ok(())
    }

    fn skip_pi(&mut self) -> Result<(), String> {
        debug_assert!(self.starts_with("<?"));
        self.pos += 2;
        while !self.eof() && !self.starts_with("?>") {
            self.pos += 1;
        }
        if !self.consume("?>") {
            return Err("Unterminated emblem instruction".to_string());
        }
        Ok(())
    }

    fn skip_prolog(&mut self) -> Result<(), String> {
        loop {
            self.skip_ws();
            if self.starts_with("<!--") {
                self.skip_comment()?;
            } else if self.starts_with("<?") {
                self.skip_pi()?;
            } else {
                return Ok(());
            }
        }
    }

    fn parse_name(&mut self) -> Result<String, String> {
        let start = self.pos;
        if self.eof()
            || !(self.bytes[self.pos].is_ascii_alphabetic()
                || matches!(self.bytes[self.pos], b'_' | b':'))
        {
            return Err("Malformed emblem document".to_string());
        }
        self.pos += 1;
        while !self.eof()
            && (self.bytes[self.pos].is_ascii_alphanumeric()
                || matches!(self.bytes[self.pos], b'_' | b':' | b'-' | b'.'))
        {
            self.pos += 1;
        }
        Ok(String::from_utf8_lossy(&self.bytes[start..self.pos]).into_owned())
    }

    fn decode_entities(raw: &[u8]) -> Result<String, String> {
        let text = std::str::from_utf8(raw).map_err(|_| "Malformed emblem document".to_string())?;
        let mut out = String::new();
        let mut rest = text;
        while let Some(amp) = rest.find('&') {
            out.push_str(&rest[..amp]);
            rest = &rest[amp + 1..];
            let end = rest
                .find(';')
                .ok_or("Malformed emblem entity".to_string())?;
            let entity = &rest[..end];
            rest = &rest[end + 1..];
            match entity {
                "amp" => out.push('&'),
                "lt" => out.push('<'),
                "gt" => out.push('>'),
                "quot" => out.push('"'),
                "apos" => out.push('\''),
                _ if entity.starts_with("#x") || entity.starts_with("#X") => {
                    let code = u32::from_str_radix(&entity[2..], 16)
                        .map_err(|_| "Malformed emblem entity".to_string())?;
                    out.push(char::from_u32(code).ok_or("Malformed emblem entity".to_string())?);
                }
                _ if entity.starts_with('#') => {
                    let code = entity[1..]
                        .parse::<u32>()
                        .map_err(|_| "Malformed emblem entity".to_string())?;
                    out.push(char::from_u32(code).ok_or("Malformed emblem entity".to_string())?);
                }
                _ => return Err("Malformed emblem entity".to_string()),
            }
        }
        out.push_str(rest);
        Ok(out)
    }

    /// Parse attributes up to `>` or `/>`; returns whether self-closed.
    fn parse_attrs(&mut self) -> Result<(Attrs, bool), String> {
        let mut attrs = Vec::new();
        loop {
            self.skip_ws();
            if self.eof() {
                return Err("Unterminated emblem document".to_string());
            }
            if self.consume("/>") {
                return Ok((attrs, true));
            }
            if self.consume(">") {
                return Ok((attrs, false));
            }
            let name = self.parse_name()?;
            self.skip_ws();
            if !self.consume("=") {
                return Err("Malformed emblem document".to_string());
            }
            self.skip_ws();
            if self.eof() || !matches!(self.bytes[self.pos], b'\'' | b'"') {
                return Err("Malformed emblem document".to_string());
            }
            let quote = self.bytes[self.pos];
            self.pos += 1;
            let start = self.pos;
            while !self.eof() && self.bytes[self.pos] != quote {
                if self.bytes[self.pos] == b'<' {
                    return Err("Malformed emblem document".to_string());
                }
                self.pos += 1;
            }
            if self.eof() {
                return Err("Unterminated emblem document".to_string());
            }
            let value = Self::decode_entities(&self.bytes[start..self.pos])?;
            self.pos += 1;
            if attrs.iter().any(|(known, _)| known == &name) {
                return Err("Malformed emblem document".to_string());
            }
            attrs.push((name, value));
        }
    }

    /// After `<title ...>` / `<desc ...>`: skip text up to the matching
    /// close tag; element content inside metadata trips the gate.
    fn skip_text_element(&mut self, name: &str, self_closed: bool) -> Result<(), String> {
        if self_closed {
            return Ok(());
        }
        let close = format!("</{name}");
        while !self.eof() && !self.starts_with(&close) {
            if self.bytes[self.pos] == b'<' {
                return Err(format!("Unexpected emblem element in {name}"));
            }
            self.pos += 1;
        }
        if !self.consume(&close) {
            return Err("Unterminated emblem document".to_string());
        }
        self.skip_ws();
        if !self.consume(">") {
            return Err("Malformed emblem document".to_string());
        }
        Ok(())
    }
}

fn tag_boundary(cursor: &Cursor) -> bool {
    cursor.eof()
        || matches!(
            cursor.bytes[cursor.pos],
            b' ' | b'\t' | b'\n' | b'\r' | b'/' | b'>'
        )
}

/// Parse the canonical document shape: one SVG root holding only
/// title/desc/path children and comments. Anything else trips the gate.
pub(crate) fn parse_svg(text: &str) -> Result<(Attrs, Vec<Attrs>), String> {
    let mut cursor = Cursor::new(text);
    cursor.skip_prolog()?;
    if !cursor.consume("<svg") || !tag_boundary(&cursor) {
        return Err("Emblem root is not an SVG document".to_string());
    }
    let (root, self_closed) = cursor.parse_attrs()?;
    let mut paths = Vec::new();
    if !self_closed {
        loop {
            cursor.skip_ws();
            if cursor.starts_with("<!--") {
                cursor.skip_comment()?;
            } else if cursor.starts_with("<?") {
                cursor.skip_pi()?;
            } else if cursor.starts_with("</") {
                cursor.consume("</");
                let name = cursor.parse_name()?;
                cursor.skip_ws();
                if !cursor.consume(">") {
                    return Err("Malformed emblem document".to_string());
                }
                if name != "svg" {
                    return Err(format!("Unexpected emblem element: {name}"));
                }
                break;
            } else if cursor.starts_with("<path") {
                cursor.consume("<path");
                if !tag_boundary(&cursor) {
                    return Err("Malformed emblem document".to_string());
                }
                let (attrs, closed) = cursor.parse_attrs()?;
                if !closed {
                    cursor.skip_ws();
                    if !cursor.consume("</path") {
                        return Err("Unexpected emblem path content".to_string());
                    }
                    cursor.skip_ws();
                    if !cursor.consume(">") {
                        return Err("Malformed emblem document".to_string());
                    }
                }
                paths.push(attrs);
            } else if cursor.starts_with("<title") || cursor.starts_with("<desc") {
                let name = if cursor.starts_with("<title") {
                    cursor.consume("<title");
                    "title"
                } else {
                    cursor.consume("<desc");
                    "desc"
                };
                if !tag_boundary(&cursor) {
                    return Err("Malformed emblem document".to_string());
                }
                let (_, closed) = cursor.parse_attrs()?;
                cursor.skip_text_element(name, closed)?;
            } else if cursor.eof() {
                return Err("Unterminated emblem document".to_string());
            } else {
                cursor.consume("<");
                let name = cursor.parse_name().unwrap_or_else(|_| "?".to_string());
                return Err(format!("Unexpected emblem element: {name}"));
            }
        }
    }
    cursor.skip_prolog()?;
    if !cursor.eof() {
        return Err("Malformed emblem document".to_string());
    }
    Ok((root, paths))
}

pub(crate) fn attr<'a>(attrs: &'a Attrs, name: &str) -> Option<&'a str> {
    attrs
        .iter()
        .find(|(key, _)| key == name)
        .map(|(_, value)| value.as_str())
}
