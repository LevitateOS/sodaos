use super::config::Config;
use super::JsonParser;

// go_quoted mirrors strconv.Quote: ASCII graphic bytes and U+0020 pass
// through, C0/DEL take short or \x escapes, and every other character
// Go's IsPrint rejects (controls, other separators, format,
// private-use) takes \u or \U escapes.
pub(crate) fn go_quoted(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    out.push('"');
    for c in s.chars() {
        match c {
            '\u{07}' => out.push_str("\\a"),
            '\u{08}' => out.push_str("\\b"),
            '\u{0c}' => out.push_str("\\f"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\u{0b}' => out.push_str("\\v"),
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            c if (c as u32) < 0x80 => {
                if c.is_ascii_graphic() || c == ' ' {
                    out.push(c);
                } else {
                    out.push_str(&format!("\\x{:02x}", c as u32));
                }
            }
            c if c.is_control() || (c.is_whitespace() && c != ' ') || is_go_nonprint(c) => {
                if (c as u32) <= 0xffff {
                    out.push_str(&format!("\\u{:04x}", c as u32));
                } else {
                    out.push_str(&format!("\\U{:08x}", c as u32));
                }
            }
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

// is_go_nonprint covers the Unicode format (Cf) and private-use (Co)
// ranges, which strconv.Quote escapes but char::is_control misses.
fn is_go_nonprint(c: char) -> bool {
    matches!(c as u32,
        0x00AD | 0x061C | 0x06DD | 0x070F | 0x08E2 | 0x180E | 0xFEFF
        | 0x110BD | 0x110CD | 0xE0001
        | 0x0600..=0x0605 | 0x200B..=0x200F | 0x202A..=0x202E
        | 0x2060..=0x2064 | 0x2066..=0x206F | 0xFFF9..=0xFFFB
        | 0x13430..=0x13438 | 0x1BCA0..=0x1BCA3 | 0x1D173..=0x1D17A
        | 0xE0020..=0xE007F | 0xE0100..=0xE01EF
        | 0xE000..=0xF8FF | 0xF0000..=0xFFFFD | 0x100000..=0x10FFFD)
}

// decode_host_config mirrors encoding/json Decode with DisallowUnknownFields
// into host.Config: exact-then-case-insensitive keys, last duplicate wins,
// null is a no-op, trailing data ignored, Go-compatible error strings.
pub(crate) fn decode_host_config(data: &[u8]) -> Result<Config, String> {
    let mut p = JsonParser::new(data);
    p.skip_ws();
    if p.eof() {
        return Err(String::from("EOF"));
    }
    match p.peek() {
        Some(b'{') => {
            p.bump();
        }
        Some(b'n') => {
            p.parse_literal()?;
            return Ok(Config::default());
        }
        Some(_) => {
            let kind = p.value_kind()?;
            return Err(format!(
                "json: cannot unmarshal {kind} into Go value of type host.Config"
            ));
        }
        None => return Err(String::from("EOF")),
    }
    let mut c = Config::default();
    // Go saves the first unknown-field/type error but keeps parsing: a
    // later syntax error overwrites it, later save-errors do not.
    let mut saved: Option<String> = None;
    let mut first = true;
    loop {
        p.skip_ws();
        if p.eof() {
            return Err(String::from("unexpected EOF"));
        }
        if first && p.peek() == Some(b'}') {
            p.bump();
            break;
        }
        if p.peek() != Some(b'"') {
            return Err(invalid_character(
                &p,
                "looking for beginning of object key string",
            ));
        }
        let key = p.parse_string()?;
        p.skip_ws();
        if p.eof() {
            return Err(String::from("unexpected EOF"));
        }
        if p.peek() != Some(b':') {
            return Err(invalid_character(&p, "after object key"));
        }
        p.bump();
        p.skip_ws();
        if p.eof() {
            return Err(String::from("unexpected EOF"));
        }
        if let Some(err) = decode_host_field(&mut p, &key, &mut c)? {
            if saved.is_none() {
                saved = Some(err);
            }
        }
        p.skip_ws();
        if p.eof() {
            return Err(String::from("unexpected EOF"));
        }
        match p.peek() {
            Some(b',') => {
                p.bump();
            }
            Some(b'}') => {
                p.bump();
                break;
            }
            _ => return Err(invalid_character(&p, "after object key:value pair")),
        }
        first = false;
    }
    if let Some(err) = saved {
        return Err(err);
    }
    Ok(c)
}

pub(crate) fn invalid_character(p: &JsonParser, context: &str) -> String {
    let c = p.peek_char();
    format!("invalid character '{c}' {context}")
}

fn decode_host_field(
    p: &mut JsonParser,
    key: &str,
    c: &mut Config,
) -> Result<Option<String>, String> {
    // Value syntax validates before field assignment: Go reports a broken
    // value even for unknown fields. The returned save-error (unknown field
    // or type mismatch) lets the caller keep parsing.
    enum Value {
        Str(String),
        Bool(bool),
        Null,
        Other(&'static str),
    }
    let value = match p.peek() {
        Some(b'"') => Value::Str(p.parse_string()?),
        Some(b't') | Some(b'f') | Some(b'n') => match p.parse_literal()? {
            "true" => Value::Bool(true),
            "false" => Value::Bool(false),
            _ => Value::Null,
        },
        Some(b'-') | Some(b'0'..=b'9') => {
            p.scan_number()?;
            Value::Other("number")
        }
        Some(b'[') => {
            p.skip_value()?;
            Value::Other("array")
        }
        Some(b'{') => {
            p.skip_value()?;
            Value::Other("object")
        }
        _ => return Err(invalid_character(p, "looking for beginning of value")),
    };
    let slot = match host_field_slot(key) {
        Some(s) => s,
        None => return Ok(Some(format!("json: unknown field {}", go_quoted(key)))),
    };
    if matches!(value, Value::Null) {
        return Ok(None);
    }
    let is_bool = slot == 7;
    match value {
        Value::Str(s) => {
            if is_bool {
                return Ok(Some(type_error("string", key, "bool")));
            }
            set_host_string(c, slot, s);
            Ok(None)
        }
        Value::Bool(b) => {
            if !is_bool {
                return Ok(Some(type_error("bool", key, "string")));
            }
            c.tailnet_management = b;
            Ok(None)
        }
        Value::Other(kind) => Ok(Some(type_error(
            kind,
            key,
            if is_bool { "bool" } else { "string" },
        ))),
        Value::Null => Ok(None),
    }
}

fn type_error(kind: &str, key: &str, ty: &str) -> String {
    format!("json: cannot unmarshal {kind} into Go struct field Config.{key} of type {ty}")
}

// host_field_slot resolves exact keys first, then one case-insensitive
// fallback, mirroring encoding/json field matching.
fn host_field_slot(key: &str) -> Option<usize> {
    const KEYS: [&str; 13] = [
        "muse_sha256",
        "muse_version",
        "muse_socket",
        "identity_socket",
        "codex_harness",
        "codex_harness_sha256",
        "codex_harness_version",
        "tailnet_management",
        "tailnet_image",
        "image",
        "network",
        "subnet",
        "bridge",
    ];
    if let Some(i) = KEYS.iter().position(|k| *k == key) {
        return Some(i);
    }
    let mut found = None;
    for (i, k) in KEYS.iter().enumerate() {
        if k.eq_ignore_ascii_case(key) {
            if found.is_some() {
                return None;
            }
            found = Some(i);
        }
    }
    found
}

fn set_host_string(c: &mut Config, slot: usize, value: String) {
    match slot {
        0 => c.muse_sha256 = value,
        1 => c.muse_version = value,
        2 => c.muse_socket = value,
        3 => c.identity_socket = value,
        4 => c.codex_harness = value,
        5 => c.codex_harness_sha256 = value,
        6 => c.codex_harness_version = value,
        8 => c.tailnet_image = value,
        9 => c.image = value,
        10 => c.network = value,
        11 => c.subnet = value,
        12 => c.bridge = value,
        _ => {}
    }
}
