use std::fs;

use crate::origin::{valid_listen, valid_origin};

/// The embedded Python block: read only public origins and the dashboard
/// port, never credentials or the whole document.
pub(crate) fn render_config(config: &str, tunnel_host: &str) {
    let host = if tunnel_host == "HOST" {
        ""
    } else {
        tunnel_host
    };
    let raw = fs::read(config).unwrap_or_default();
    let text = String::from_utf8_lossy(&raw);
    let parsed = parse_top_object(&text);
    let rendered = parsed.as_ref().and_then(|fields| {
        // `config.get('listen', default)`: absent takes the default, but a
        // present non-string is invalid, never defaulted.
        let listen = match fields.get("listen") {
            None => "127.0.0.1:8080".to_string(),
            Some(Some(value)) => value.clone(),
            Some(None) => return None,
        };
        let (listen_host, listen_port) = valid_listen(&listen)?;
        let url = match fields.get("forgejo_url") {
            Some(Some(url)) => url,
            _ => return None,
        };
        let display = valid_origin(url)?;
        Some((listen_host, listen_port, display))
    });
    match rendered {
        None => {
            println!(
                "\nBrowser origins are not configured or cannot be read; complete operator setup."
            );
            println!("\nForgejo installer is loopback-first: http://127.0.0.1:3000");
            if !host.is_empty() {
                println!("From your client, tunnel over SSH:");
                println!("  ssh -N -L 33000:127.0.0.1:3000 root@{host}");
                println!("Then open http://localhost:33000 in that client browser.");
            } else {
                println!(
                    "No local uplink address is available for an SSH tunnel; configure networking first."
                );
            }
        }
        Some((listen_host, listen_port, forgejo)) => {
            println!("\nDashboard is loopback-first: http://{listen_host}:{listen_port}");
            if !host.is_empty() {
                println!("From your client, tunnel over SSH:");
                println!("  ssh -N -L {listen_port}:{listen_host}:{listen_port} root@{host}");
                println!("Then open http://{listen_host}:{listen_port} in that client browser.");
            } else {
                println!(
                    "No local uplink address is available for an SSH tunnel; configure networking first."
                );
            }
            println!("\nConfigured browser origins (not a listener or reachability check):");
            println!("  Forgejo / Sodaspaces: {forgejo}");
        }
    }
}

/// Top-level JSON object as string-field map (last duplicate wins, like
/// Python's dict). None = parse failure or non-object. Values decode to
/// Some(string) for JSON strings, None for any other JSON value.
pub(crate) fn parse_top_object(
    text: &str,
) -> Option<std::collections::HashMap<String, Option<String>>> {
    let bytes = text.as_bytes();
    let mut parser = JsonParser {
        bytes,
        pos: 0,
        depth: 0,
    };
    parser.skip_ws();
    if parser.peek() != Some(b'{') {
        return None;
    }
    let fields = parser.parse_object_top()?;
    parser.skip_ws();
    if parser.pos != bytes.len() {
        return None; // json.loads rejects trailing data ("Extra data").
    }
    Some(fields)
}

struct JsonParser<'a> {
    bytes: &'a [u8],
    pos: usize,
    depth: u32,
}

impl<'a> JsonParser<'a> {
    fn peek(&self) -> Option<u8> {
        self.bytes.get(self.pos).copied()
    }

    fn skip_ws(&mut self) {
        while matches!(self.peek(), Some(b' ' | b'\t' | b'\n' | b'\r')) {
            self.pos += 1;
        }
    }

    fn literal(&mut self, word: &[u8]) -> bool {
        if self.bytes[self.pos..].starts_with(word) {
            self.pos += word.len();
            true
        } else {
            false
        }
    }

    fn parse_object_top(&mut self) -> Option<std::collections::HashMap<String, Option<String>>> {
        // pos at '{'.
        self.pos += 1;
        let mut fields = std::collections::HashMap::new();
        self.skip_ws();
        if self.peek() == Some(b'}') {
            self.pos += 1;
            return Some(fields);
        }
        loop {
            self.skip_ws();
            let key = self.parse_string()?;
            self.skip_ws();
            if self.peek() != Some(b':') {
                return None;
            }
            self.pos += 1;
            self.skip_ws();
            let value = self.parse_value_top()?;
            fields.insert(key, value);
            self.skip_ws();
            match self.peek() {
                Some(b',') => {
                    self.pos += 1;
                }
                Some(b'}') => {
                    self.pos += 1;
                    return Some(fields);
                }
                _ => return None,
            }
        }
    }

    /// Top-level member value: decoded for strings, validated-and-discarded
    /// otherwise (objects/arrays recurse for syntax validation, as json.loads
    /// parses the whole document).
    fn parse_value_top(&mut self) -> Option<Option<String>> {
        match self.peek()? {
            b'"' => Some(Some(self.parse_string()?)),
            _ => {
                self.parse_any()?;
                Some(None)
            }
        }
    }

    fn parse_any(&mut self) -> Option<()> {
        // json.loads has no depth limit (deep input raises RecursionError,
        // which the script does NOT catch); cap here and treat overflow as
        // a parse failure instead of a traceback.
        if self.depth > 1000 {
            return None;
        }
        self.depth += 1;
        let result = self.parse_any_inner();
        self.depth -= 1;
        result
    }

    fn parse_any_inner(&mut self) -> Option<()> {
        match self.peek()? {
            b'"' => {
                self.parse_string()?;
                Some(())
            }
            b'{' => {
                self.pos += 1;
                self.skip_ws();
                if self.peek() == Some(b'}') {
                    self.pos += 1;
                    return Some(());
                }
                loop {
                    self.skip_ws();
                    self.parse_string()?;
                    self.skip_ws();
                    if self.peek() != Some(b':') {
                        return None;
                    }
                    self.pos += 1;
                    self.skip_ws();
                    self.parse_any()?;
                    self.skip_ws();
                    match self.peek() {
                        Some(b',') => self.pos += 1,
                        Some(b'}') => {
                            self.pos += 1;
                            return Some(());
                        }
                        _ => return None,
                    }
                }
            }
            b'[' => {
                self.pos += 1;
                self.skip_ws();
                if self.peek() == Some(b']') {
                    self.pos += 1;
                    return Some(());
                }
                loop {
                    self.skip_ws();
                    self.parse_any()?;
                    self.skip_ws();
                    match self.peek() {
                        Some(b',') => self.pos += 1,
                        Some(b']') => {
                            self.pos += 1;
                            return Some(());
                        }
                        _ => return None,
                    }
                }
            }
            b't' => self.literal(b"true").then_some(()),
            b'f' => self.literal(b"false").then_some(()),
            b'n' => self.literal(b"null").then_some(()),
            // json.loads accepts NaN/Infinity/-Infinity; all land in the
            // except branch later as non-strings.
            b'N' => self.literal(b"NaN").then_some(()),
            b'I' => self.literal(b"Infinity").then_some(()),
            b'-' => {
                if self.bytes[self.pos..].starts_with(b"-Infinity") {
                    self.pos += "-Infinity".len();
                    Some(())
                } else {
                    self.parse_number()
                }
            }
            b'0'..=b'9' => self.parse_number(),
            _ => None,
        }
    }

    fn parse_number(&mut self) -> Option<()> {
        let start = self.pos;
        if self.peek() == Some(b'-') {
            self.pos += 1;
        }
        match self.peek() {
            Some(b'0') => self.pos += 1,
            Some(b'1'..=b'9') => {
                while matches!(self.peek(), Some(b'0'..=b'9')) {
                    self.pos += 1;
                }
            }
            _ => return None,
        }
        if self.peek() == Some(b'.') {
            self.pos += 1;
            if !matches!(self.peek(), Some(b'0'..=b'9')) {
                return None;
            }
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
        }
        if matches!(self.peek(), Some(b'e' | b'E')) {
            self.pos += 1;
            if matches!(self.peek(), Some(b'+' | b'-')) {
                self.pos += 1;
            }
            if !matches!(self.peek(), Some(b'0'..=b'9')) {
                return None;
            }
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
        }
        if self.pos == start {
            return None;
        }
        Some(())
    }

    /// JSON string with full escape decoding, including \uXXXX surrogate
    /// pairs. Lone surrogates fail (Python keeps them, then crashes printing
    /// a validating URL — a traceback, not a contract).
    fn parse_string(&mut self) -> Option<String> {
        if self.peek() != Some(b'"') {
            return None;
        }
        self.pos += 1;
        let mut out = String::new();
        loop {
            let byte = self.peek()?;
            match byte {
                b'"' => {
                    self.pos += 1;
                    return Some(out);
                }
                b'\\' => {
                    self.pos += 1;
                    match self.peek()? {
                        b'"' => {
                            out.push('"');
                            self.pos += 1;
                        }
                        b'\\' => {
                            out.push('\\');
                            self.pos += 1;
                        }
                        b'/' => {
                            out.push('/');
                            self.pos += 1;
                        }
                        b'b' => {
                            out.push('\u{8}');
                            self.pos += 1;
                        }
                        b'f' => {
                            out.push('\u{c}');
                            self.pos += 1;
                        }
                        b'n' => {
                            out.push('\n');
                            self.pos += 1;
                        }
                        b'r' => {
                            out.push('\r');
                            self.pos += 1;
                        }
                        b't' => {
                            out.push('\t');
                            self.pos += 1;
                        }
                        b'u' => {
                            self.pos += 1;
                            let high = self.parse_hex4()?;
                            if (0xD800..0xDC00).contains(&high) {
                                if self.bytes.get(self.pos..self.pos + 2) == Some(b"\\u") {
                                    self.pos += 2;
                                    let low = self.parse_hex4()?;
                                    if (0xDC00..0xE000).contains(&low) {
                                        let code =
                                            0x10000 + ((high - 0xD800) << 10) + (low - 0xDC00);
                                        out.push(char::from_u32(code)?);
                                    } else {
                                        return None;
                                    }
                                } else {
                                    return None;
                                }
                            } else if (0xDC00..0xE000).contains(&high) {
                                return None;
                            } else {
                                out.push(char::from_u32(high)?);
                            }
                        }
                        _ => return None,
                    }
                }
                0x00..=0x1F => return None, // json.loads rejects literal controls.
                _ => {
                    // Regular UTF-8 scalar (input is already valid UTF-8).
                    let rest = &self.bytes[self.pos..];
                    let text = std::str::from_utf8(rest).ok()?;
                    let ch = text.chars().next()?;
                    out.push(ch);
                    self.pos += ch.len_utf8();
                }
            }
        }
    }

    fn parse_hex4(&mut self) -> Option<u32> {
        if self.pos + 4 > self.bytes.len() {
            return None;
        }
        let mut value = 0u32;
        for i in 0..4 {
            value = value * 16 + hex_value(self.bytes[self.pos + i])?;
        }
        self.pos += 4;
        Some(value)
    }
}

fn hex_value(byte: u8) -> Option<u32> {
    match byte {
        b'0'..=b'9' => Some((byte - b'0') as u32),
        b'a'..=b'f' => Some((byte - b'a' + 10) as u32),
        b'A'..=b'F' => Some((byte - b'A' + 10) as u32),
        _ => None,
    }
}
