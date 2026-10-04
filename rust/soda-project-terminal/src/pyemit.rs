//! Python `json.dumps(value, separators=(',', ':'), ensure_ascii=True)`
//! over [`soda_json::JsonValue`], plus strict object-shape validation with
//! duplicate-key rejection (the `.py` decoders' `object_pairs_hook=unique`).

use soda_json::JsonValue;

/// Emit one compact ASCII JSON document, byte-identical to CPython.
pub fn dumps(value: &JsonValue) -> String {
    let mut out = String::new();
    emit(&mut out, value);
    out
}

/// `line()`: compact document plus `\n`.
pub fn line(value: &JsonValue) -> Vec<u8> {
    let mut doc = dumps(value);
    doc.push('\n');
    doc.into_bytes()
}

fn emit(out: &mut String, value: &JsonValue) {
    match value {
        JsonValue::Null => out.push_str("null"),
        JsonValue::Bool(true) => out.push_str("true"),
        JsonValue::Bool(false) => out.push_str("false"),
        JsonValue::Number(raw) => out.push_str(raw),
        JsonValue::Str(text) => escape_py(out, text),
        JsonValue::Array(items) => {
            out.push('[');
            for (i, item) in items.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                emit(out, item);
            }
            out.push(']');
        }
        JsonValue::Object(entries) => {
            out.push('{');
            for (i, (key, item)) in entries.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                escape_py(out, key);
                out.push(':');
                emit(out, item);
            }
            out.push('}');
        }
    }
}

/// CPython `ensure_ascii` string quoting: short escapes, `\uXXXX`
/// (lowercase hex) for other controls/DEL/non-ASCII with surrogate pairs
/// for astral characters. `/`, `<`, `>`, `&` stay raw.
pub fn escape_py(out: &mut String, text: &str) {
    out.push('"');
    for ch in text.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\x08' => out.push_str("\\b"),
            '\x0c' => out.push_str("\\f"),
            c if (c as u32) < 0x20 || c as u32 == 0x7f => {
                out.push_str(&format!("\\u{:04x}", c as u32));
            }
            c if (c as u32) < 0x80 => out.push(c),
            c if (c as u32) < 0x10000 => {
                out.push_str(&format!("\\u{:04x}", c as u32));
            }
            c => {
                let v = c as u32 - 0x10000;
                out.push_str(&format!("\\u{:04x}\\u{:04x}", 0xd800 + (v >> 10), 0xdc00 + (v & 0x3ff)));
            }
        }
    }
    out.push('"');
}

/// Strict object entries: duplicate keys rejected like the `.py` hook.
pub fn unique_entries(value: &JsonValue) -> Option<&Vec<(String, JsonValue)>> {
    match value {
        JsonValue::Object(entries) => {
            for i in 0..entries.len() {
                for other in entries.iter().skip(i + 1) {
                    if other.0 == entries[i].0 {
                        return None;
                    }
                }
            }
            Some(entries)
        }
        _ => None,
    }
}

/// Exact key set plus lookup over unique entries.
pub fn shape<'a>(entries: &'a [(String, JsonValue)], keys: &[&str]) -> Option<Vec<(String, &'a JsonValue)>> {
    if entries.len() != keys.len() {
        return None;
    }
    let mut out = Vec::with_capacity(keys.len());
    for key in keys {
        let found = entries.iter().find(|(k, _)| k == key)?;
        out.push((found.0.clone(), &found.1));
    }
    // Exact set: every entry key must be wanted (lengths match, so a miss
    // means an unwanted key).
    if entries.iter().any(|(k, _)| !keys.contains(&k.as_str())) {
        return None;
    }
    Some(out)
}

/// Strict JSON integer as i64 (no fraction/exponent, like `type(x) is int`
/// after `json.loads` for in-range values).
pub fn as_int(value: &JsonValue) -> Option<i64> {
    value.as_integer().and_then(|n| i64::try_from(n).ok())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn obj(pairs: Vec<(&str, JsonValue)>) -> JsonValue {
        JsonValue::Object(pairs.into_iter().map(|(k, v)| (k.to_string(), v)).collect())
    }

    #[test]
    fn emit_matches_cpython() {
        // Baked against CPython json.dumps(separators=(',',':'), ensure_ascii=True).
        let value = obj(vec![
            ("type", JsonValue::Str("output".to_string())),
            ("data", JsonValue::Str("a+b/c<d>&\"q\"\\é\0\x1f\x7f😀".to_string())),
            ("n", JsonValue::Number("-12".to_string())),
            ("t", JsonValue::Bool(true)),
            ("f", JsonValue::Bool(false)),
            ("z", JsonValue::Null),
            ("a", JsonValue::Array(vec![JsonValue::Number("1".to_string())])),
        ]);
        assert_eq!(
            dumps(&value),
            "{\"type\":\"output\",\"data\":\"a+b/c<d>&\\\"q\\\"\\\\\\u00e9\\u0000\\u001f\\u007f\\ud83d\\ude00\",\"n\":-12,\"t\":true,\"f\":false,\"z\":null,\"a\":[1]}"
        );
    }

    #[test]
    fn emit_short_escapes() {
        assert_eq!(dumps(&JsonValue::Str("\x08\x0c\n\r\t".to_string())), "\"\\b\\f\\n\\r\\t\"");
        assert_eq!(dumps(&JsonValue::Str("/".to_string())), "\"/\"");
    }

    #[test]
    fn duplicates_rejected() {
        let dup = JsonValue::parse("{\"a\":1,\"a\":2}").unwrap();
        assert!(unique_entries(&dup).is_none());
        let ok = JsonValue::parse("{\"a\":1,\"b\":2}").unwrap();
        assert!(unique_entries(&ok).is_some());
        assert!(unique_entries(&JsonValue::Number("1".to_string())).is_none());
    }

    #[test]
    fn exact_shape() {
        let value = JsonValue::parse("{\"type\":\"close\"}").unwrap();
        let entries = unique_entries(&value).unwrap();
        assert!(shape(entries, &["type"]).is_some());
        assert!(shape(entries, &["type", "data"]).is_none());
        assert!(shape(entries, &["other"]).is_none());
        let value = JsonValue::parse("{\"type\":\"x\",\"z\":1}").unwrap();
        let entries = unique_entries(&value).unwrap();
        assert!(shape(entries, &["type", "data"]).is_none());
    }

    #[test]
    fn int_shapes() {
        assert_eq!(as_int(&JsonValue::Number("42".to_string())), Some(42));
        assert_eq!(as_int(&JsonValue::Number("-1".to_string())), Some(-1));
        assert_eq!(as_int(&JsonValue::Number("4.0".to_string())), None);
        assert_eq!(as_int(&JsonValue::Number("1e3".to_string())), None);
        assert_eq!(as_int(&JsonValue::Bool(true)), None);
        assert_eq!(as_int(&JsonValue::Number("9223372036854775808".to_string())), None);
    }
}
