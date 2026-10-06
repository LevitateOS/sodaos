//! Terminal-logo renderer (ports `scripts/render-terminal-logo.py`).
//!
//! Samples the canonical polygon emblem into a 32-column, 16-row ASCII
//! mark. Geometry, sampling, colorization and gate messages match the
//! script. The script leans on a full XML parser; the port instead runs a
//! strict gate that only accepts the canonical document shape (SVG root,
//! title/desc/path children, comments) and fails closed on anything else,
//! which is exactly what the emblem gate is for: any changed syntax must
//! trip it explicitly.

use std::path::Path;

#[derive(Debug, Clone, PartialEq)]
enum Token {
    Cmd(char),
    Num(String),
}

/// Tokenize path data like `re.findall(r'[A-Za-z]|-?\d+(?:\.\d+)?', data)`:
/// letters are commands, `-?\d+(\.\d+)?` numbers, everything else skipped.
fn tokenize(data: &str) -> Vec<Token> {
    let bytes = data.as_bytes();
    let mut tokens = Vec::new();
    let mut i = 0;
    while i < bytes.len() {
        let b = bytes[i];
        if b.is_ascii_alphabetic() {
            tokens.push(Token::Cmd(b as char));
            i += 1;
        } else if b.is_ascii_digit()
            || (b == b'-' && i + 1 < bytes.len() && bytes[i + 1].is_ascii_digit())
        {
            let start = i;
            if bytes[i] == b'-' {
                i += 1;
            }
            while i < bytes.len() && bytes[i].is_ascii_digit() {
                i += 1;
            }
            if i + 1 < bytes.len() && bytes[i] == b'.' && bytes[i + 1].is_ascii_digit() {
                i += 1;
                while i < bytes.len() && bytes[i].is_ascii_digit() {
                    i += 1;
                }
            }
            tokens.push(Token::Num(data[start..i].to_string()));
        } else {
            i += 1;
        }
    }
    tokens
}

fn operand(token: &Token) -> Result<f64, String> {
    match token {
        Token::Num(raw) => raw
            .parse::<f64>()
            .map_err(|_| format!("could not convert string to float: '{raw}'")),
        Token::Cmd(c) => Err(format!("could not convert string to float: '{c}'")),
    }
}

/// Parse absolute-polygon path data into rings, with the script's exact
/// gates (points accumulate across moves until `Z`, like the owner).
pub fn polygons(data: &str) -> Result<Vec<Vec<(f64, f64)>>, String> {
    let tokens = tokenize(data);
    let mut result: Vec<Vec<(f64, f64)>> = Vec::new();
    let mut points: Vec<(f64, f64)> = Vec::new();
    let (mut x, mut y) = (0.0, 0.0);
    let mut i = 0;
    while i < tokens.len() {
        match &tokens[i] {
            Token::Cmd('M') | Token::Cmd('L') => {
                let command = match &tokens[i] {
                    Token::Cmd(c) => *c,
                    _ => unreachable!(),
                };
                i += 1;
                if i + 2 > tokens.len() {
                    return Err(format!("Truncated emblem command: {command}"));
                }
                x = operand(&tokens[i])?;
                y = operand(&tokens[i + 1])?;
                i += 2;
            }
            Token::Cmd('H') => {
                i += 1;
                if i + 1 > tokens.len() {
                    return Err("Truncated emblem command: H".to_string());
                }
                x = operand(&tokens[i])?;
                i += 1;
            }
            Token::Cmd('V') => {
                i += 1;
                if i + 1 > tokens.len() {
                    return Err("Truncated emblem command: V".to_string());
                }
                y = operand(&tokens[i])?;
                i += 1;
            }
            Token::Cmd('Z') => {
                i += 1;
                if points.len() < 3 {
                    return Err("Emblem polygon has fewer than 3 points".to_string());
                }
                result.push(std::mem::take(&mut points));
                continue;
            }
            Token::Cmd(c) => return Err(format!("Unsupported emblem command: {c}")),
            Token::Num(raw) => return Err(format!("Unsupported emblem command: {raw}")),
        }
        points.push((x, y));
    }
    if !points.is_empty() || result.is_empty() {
        return Err("Unterminated emblem geometry".to_string());
    }
    Ok(result)
}

fn inside(x: f64, y: f64, polygon: &[(f64, f64)]) -> bool {
    let mut hit = false;
    for (i, (ax, ay)) in polygon.iter().enumerate() {
        let (bx, by) = polygon[(i + 1) % polygon.len()];
        if (*ay > y) != (by > y) && x < (bx - ax) * (y - ay) / (by - ay) + ax {
            hit = !hit;
        }
    }
    hit
}

fn render_layers(layers: &[Vec<Vec<(f64, f64)>>]) -> (String, String) {
    let mut rows = Vec::new();
    for row in 0..16 {
        let mut cells = String::new();
        for column in 0..32 {
            let mut char = ' ';
            for (shape, glyph) in layers.iter().zip(['#', '@']) {
                let mut hits = 0;
                for polygon in shape {
                    if inside(
                        (column as f64 + 0.5) * 4.0,
                        (row as f64 + 0.5) * 8.0,
                        polygon,
                    ) {
                        hits += 1;
                    }
                }
                if hits % 2 == 1 {
                    char = glyph;
                }
            }
            cells.push(char);
        }
        rows.push(cells.trim_end().to_string());
    }
    let mut colored = Vec::new();
    for row in &rows {
        let mut line = String::new();
        let mut previous: Option<char> = None;
        for char in row.chars() {
            let color = if char == '#' { '1' } else { '2' };
            if char != ' ' && Some(color) != previous {
                line.push('$');
                line.push(color);
                previous = Some(color);
            }
            line.push(char);
        }
        line.push_str("$2");
        colored.push(line);
    }
    (colored.join("\n") + "\n", rows.join("\n") + "\n\nSODA OS\n")
}

type Attrs = Vec<(String, String)>;

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
fn parse_svg(text: &str) -> Result<(Attrs, Vec<Attrs>), String> {
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

fn attr<'a>(attrs: &'a Attrs, name: &str) -> Option<&'a str> {
    attrs
        .iter()
        .find(|(key, _)| key == name)
        .map(|(_, value)| value.as_str())
}

/// Render one SVG document to `(sodaos.txt, motd.txt)`, with the script's
/// gate order: viewBox, then layers, then fill rule, then geometry.
pub fn render_svg(text: &str) -> Result<(String, String), String> {
    let (root, paths) = parse_svg(text)?;
    if attr(&root, "viewBox") != Some("0 0 128 128") {
        return Err("Unexpected emblem viewBox".to_string());
    }
    // `findall` only sees the SVG namespace: without the default binding
    // there are no layers, and the layers gate fires first, like the owner.
    let paths = if attr(&root, "xmlns") == Some("http://www.w3.org/2000/svg") {
        paths
    } else {
        Vec::new()
    };
    let fills: Vec<Option<&str>> = paths.iter().map(|attrs| attr(attrs, "fill")).collect();
    if fills.as_slice() != [Some("#df001b"), Some("#101010")] {
        return Err("Unexpected emblem layers".to_string());
    }
    if paths.is_empty()
        || paths
            .iter()
            .any(|attrs| attr(attrs, "fill-rule") != Some("evenodd"))
    {
        return Err("Unexpected emblem fill rule".to_string());
    }
    let mut layers = Vec::new();
    for attrs in &paths {
        let data = attr(attrs, "d").ok_or("Emblem path has no geometry".to_string())?;
        layers.push(polygons(data)?);
    }
    Ok(render_layers(&layers))
}

/// Render the canonical emblem, or `--check` the committed outputs.
pub fn run(root: &Path, check: bool) -> Result<(), String> {
    let source = root.join("assets/branding/source/soda-symbol-brutalist.svg");
    let out = root.join("assets/branding/terminal");
    let svg = std::fs::read_to_string(&source)
        .map_err(|e| format!("cannot read {}: {e}", source.display()))?;
    let (colored, plain) = render_svg(&svg)?;
    for (name, text) in [("sodaos.txt", colored), ("motd.txt", plain)] {
        let path = out.join(name);
        if check {
            let current = std::fs::read_to_string(&path)
                .map_err(|e| format!("cannot read {}: {e}", path.display()))?;
            if current != text {
                return Err(format!("Stale terminal branding: {}", path.display()));
            }
        } else {
            std::fs::write(&path, text)
                .map_err(|e| format!("cannot write {}: {e}", path.display()))?;
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests;
