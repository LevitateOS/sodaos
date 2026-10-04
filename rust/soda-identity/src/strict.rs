// Bounded strict JSON admission, mirroring internal/strictjson: exactly one
// object, valid UTF-8, no duplicate fields at any depth (depth cap 100),
// no unknown fields (with Go's case-insensitive fallback), and null
// tolerated for scalar fields. Every rejection maps to the same client
// response (`invalid request`), so only the accept/reject direction must
// match; messages stay close for operators.
pub const MAX_DOCUMENT: usize = 1 << 20;

pub fn decode<T: serde::de::DeserializeOwned>(
    input: &[u8],
    max: usize,
    fields: &[&str],
    nested: &[(&str, &[&str])],
) -> Result<T, String> {
    if input.len() > max {
        return Err("request exceeds size limit".to_string());
    }
    let text =
        std::str::from_utf8(input).map_err(|_| "request must contain valid UTF-8".to_string())?;
    check_unique_keys(text)?;
    let mut value: serde_json::Value =
        serde_json::from_str(text).map_err(|e| format!("decode request: {e}"))?;
    let object = value
        .as_object_mut()
        .ok_or_else(|| "request must be one JSON object".to_string())?;
    remap_case(object, fields);
    for (name, sub) in nested {
        if let Some(inner) = object.get_mut(*name).and_then(|v| v.as_object_mut()) {
            remap_case(inner, sub);
        }
    }
    check_known_fields(object, fields, nested)?;
    serde_json::from_value(value).map_err(|e| format!("decode request: {e}"))
}

fn remap_case(object: &mut serde_json::Map<String, serde_json::Value>, fields: &[&str]) {
    // Go prefers an exact field match but also accepts a case-insensitive
    // one. Keys that would collide after folding are left alone so the
    // unknown-field check rejects them.
    let mut renames = Vec::new();
    for key in object.keys() {
        if fields.contains(&key.as_str()) {
            continue;
        }
        let mut folded = None;
        for field in fields {
            if field.eq_ignore_ascii_case(key) {
                folded = Some(*field);
                break;
            }
        }
        if let Some(field) = folded {
            if !object.contains_key(field) {
                renames.push((key.clone(), field.to_string()));
            }
        }
    }
    for (from, to) in renames {
        if let Some(value) = object.remove(&from) {
            object.insert(to, value);
        }
    }
}

fn check_known_fields(
    object: &serde_json::Map<String, serde_json::Value>,
    fields: &[&str],
    nested: &[(&str, &[&str])],
) -> Result<(), String> {
    for key in object.keys() {
        if !fields.contains(&key.as_str()) {
            return Err(format!("unknown field {key:?}"));
        }
    }
    for (name, sub) in nested {
        if let Some(inner) = object.get(*name).and_then(|v| v.as_object()) {
            for key in inner.keys() {
                if !sub.contains(&key.as_str()) {
                    return Err(format!("unknown field {key:?}"));
                }
            }
        }
    }
    Ok(())
}

struct Scanner<'a> {
    bytes: &'a [u8],
    pos: usize,
}

impl<'a> Scanner<'a> {
    fn new(text: &'a str) -> Scanner<'a> {
        Scanner {
            bytes: text.as_bytes(),
            pos: 0,
        }
    }

    fn skip_ws(&mut self) {
        while self.pos < self.bytes.len()
            && matches!(self.bytes[self.pos], b' ' | b'\t' | b'\n' | b'\r')
        {
            self.pos += 1;
        }
    }

    fn peek(&mut self) -> Option<u8> {
        self.skip_ws();
        self.bytes.get(self.pos).copied()
    }

    fn eat(&mut self, want: u8) -> bool {
        self.skip_ws();
        if self.bytes.get(self.pos) == Some(&want) {
            self.pos += 1;
            true
        } else {
            false
        }
    }

    // Decode one JSON string starting at the opening quote. Returns the
    // decoded key and consumes the closing quote. Liberal on malformed
    // escapes: the later serde parse rejects malformed documents anyway.
    fn string(&mut self) -> Option<String> {
        if !self.eat(b'"') {
            return None;
        }
        let mut out = String::new();
        loop {
            let b = *self.bytes.get(self.pos)?;
            self.pos += 1;
            match b {
                b'"' => return Some(out),
                b'\\' => {
                    let e = *self.bytes.get(self.pos)?;
                    self.pos += 1;
                    match e {
                        b'"' => out.push('"'),
                        b'\\' => out.push('\\'),
                        b'/' => out.push('/'),
                        b'b' => out.push('\x08'),
                        b'f' => out.push('\x0c'),
                        b'n' => out.push('\n'),
                        b'r' => out.push('\r'),
                        b't' => out.push('\t'),
                        b'u' => {
                            let hex = self.bytes.get(self.pos..self.pos + 4)?;
                            self.pos += 4;
                            let code =
                                u32::from_str_radix(std::str::from_utf8(hex).ok()?, 16).ok()?;
                            if (0xd800..0xdc00).contains(&code) {
                                // Surrogate pair; consume the low half.
                                if self.bytes.get(self.pos..self.pos + 2) != Some(b"\\u") {
                                    return None;
                                }
                                self.pos += 2;
                                let low_hex = self.bytes.get(self.pos..self.pos + 4)?;
                                self.pos += 4;
                                let low =
                                    u32::from_str_radix(std::str::from_utf8(low_hex).ok()?, 16)
                                        .ok()?;
                                if !(0xdc00..0xe000).contains(&low) {
                                    return None;
                                }
                                let ch = char::from_u32(
                                    0x10000 + (code - 0xd800) * 0x400 + (low - 0xdc00),
                                )?;
                                out.push(ch);
                            } else {
                                out.push(char::from_u32(code)?);
                            }
                        }
                        // Not a valid escape; keep the byte so scanning
                        // terminates. Serde rejects the document afterwards.
                        _ => out.push(e as char),
                    }
                }
                _ if b < 0x80 => out.push(b as char),
                _ => {
                    // Validated UTF-8 above; copy the whole sequence.
                    let width = if b >= 0xf0 {
                        4
                    } else if b >= 0xe0 {
                        3
                    } else if b >= 0xc0 {
                        2
                    } else {
                        return None;
                    };
                    let bytes = self.bytes.get(self.pos - 1..self.pos - 1 + width)?;
                    out.push_str(std::str::from_utf8(bytes).ok()?);
                    self.pos += width - 1;
                }
            }
        }
    }

    fn skip_string(&mut self) -> bool {
        self.string().is_some()
    }

    fn skip_scalar(&mut self) {
        while let Some(b) = self.bytes.get(self.pos) {
            if matches!(b, b',' | b']' | b'}') || b.is_ascii_whitespace() {
                break;
            }
            self.pos += 1;
        }
    }
}

// Exactly-one-object with unique decoded keys at every depth. Returns Ok
// on any structural confusion so the later serde parse (which rejects
// malformed documents) decides; only certain duplicates and over-depth
// fail here.
fn check_unique_keys(text: &str) -> Result<(), String> {
    use std::collections::HashSet;
    let mut s = Scanner::new(text);
    if !s.eat(b'{') {
        return Ok(());
    }
    let mut seen = HashSet::new();
    loop {
        s.skip_ws();
        if s.eat(b'}') {
            return Ok(());
        }
        let Some(key) = s.string() else { return Ok(()) };
        if !seen.insert(key.clone()) {
            return Err(format!("duplicate request field {key:?}"));
        }
        if !s.eat(b':') {
            return Ok(());
        }
        check_value(&mut s, &key, 0)?;
        s.skip_ws();
        if s.eat(b',') {
            continue;
        }
        if s.eat(b'}') {
            return Ok(());
        }
        return Ok(());
    }
}

fn check_value(s: &mut Scanner<'_>, top: &str, depth: u32) -> Result<(), String> {
    use std::collections::HashSet;
    if depth > 100 {
        return Err(format!(
            "decode request field {top:?}: request is nested too deeply"
        ));
    }
    match s.peek() {
        Some(b'{') => {
            s.eat(b'{');
            let mut seen = HashSet::new();
            loop {
                s.skip_ws();
                if s.eat(b'}') {
                    return Ok(());
                }
                let Some(key) = s.string() else { return Ok(()) };
                if !seen.insert(key.clone()) {
                    return Err(format!("duplicate request field {key:?}"));
                }
                if !s.eat(b':') {
                    return Ok(());
                }
                check_value(s, top, depth + 1)?;
                s.skip_ws();
                if s.eat(b',') {
                    continue;
                }
                if s.eat(b'}') {
                    return Ok(());
                }
                return Ok(());
            }
        }
        Some(b'[') => {
            s.eat(b'[');
            loop {
                s.skip_ws();
                if s.eat(b']') {
                    return Ok(());
                }
                check_value(s, top, depth + 1)?;
                s.skip_ws();
                if s.eat(b',') {
                    continue;
                }
                if s.eat(b']') {
                    return Ok(());
                }
                return Ok(());
            }
        }
        Some(b'"') => {
            if !s.skip_string() {
                return Ok(());
            }
            Ok(())
        }
        Some(_) => {
            s.skip_scalar();
            Ok(())
        }
        None => Ok(()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde::Deserialize;

    #[derive(Debug, PartialEq, Deserialize)]
    struct Envelope {
        #[serde(default)]
        command_id: String,
        #[serde(default, rename = "type")]
        wire_type: String,
        #[serde(default)]
        target: String,
    }

    const FIELDS: &[&str] = &["command_id", "type", "target"];

    fn decode_envelope(input: &str) -> Result<Envelope, String> {
        decode(input.as_bytes(), MAX_DOCUMENT, FIELDS, &[])
    }

    #[test]
    fn strict_vectors_match_go() {
        // Mirrors scripts/fixtures/portcontracts/strictjson_vectors.json.
        for (input, ok) in [
            (r#"{"type":"status"}"#, true),
            (r#"{}"#, true),
            (r#"  {"type":"status","target":"r"}  "#, true),
            ("{\"type\":\"a\\\"b \\u00e9\"}", true),
            (r#"{"type":"status","project_id":"site"}"#, false),
            (r#"{"type":"status","type":"stop"}"#, false),
            (r#"{"type":{"a":1,"a":2}}"#, false),
            (r#"{"type":{"outer":{"key":"1","key":"2"}}}"#, false),
            (
                r#"{"type":"x","target":[{"k":"1"},{"k":"1","k":"2"}]}"#,
                false,
            ),
            (r#"[]"#, false),
            (r#"{"type":"one"}{"type":"two"}"#, false),
            (r#"{"type":"one""#, false),
        ] {
            let result = decode_envelope(input);
            assert_eq!(result.is_ok(), ok, "input {input}");
        }
        assert!(decode_envelope(r#"{"type":"status","project_id":"site"}"#)
            .unwrap_err()
            .contains("unknown field"));
        assert!(decode_envelope(r#"{"type":"status","type":"stop"}"#)
            .unwrap_err()
            .contains("duplicate request field"));
    }

    #[test]
    fn limits_match_go() {
        let big = format!("{{\"type\":\"{}\"}}", "a".repeat(1 << 20));
        assert!(decode_envelope(&big).unwrap_err().contains("size limit"));
        assert!(
            decode::<Envelope>(&[b'{', b'"', 0xff], MAX_DOCUMENT, FIELDS, &[])
                .unwrap_err()
                .contains("valid UTF-8")
        );
        let mut nested = String::from("{\"type\":");
        for _ in 0..150 {
            nested.push_str("{\"k\":");
        }
        nested.push('1');
        for _ in 0..150 {
            nested.push('}');
        }
        nested.push('}');
        assert!(decode_envelope(&nested)
            .unwrap_err()
            .contains("nested too deeply"));
    }

    #[test]
    fn case_fold_matches_go_fallback() {
        let envelope = decode_envelope(r#"{"TYPE":"status"}"#).unwrap();
        assert_eq!(envelope.wire_type, "status");
        // Exact wins; a colliding fold is rejected as unknown.
        assert!(decode_envelope(r#"{"TYPE":"a","type":"b"}"#).is_err());
    }

    #[test]
    fn unicode_keys_compare_decoded() {
        assert!(decode_envelope("{\"type\":\"a\",\"\\u0074ype\":\"b\"}").is_err());
    }
}
