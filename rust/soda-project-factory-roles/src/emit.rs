//! Plain `json.dumps(value)` (default `(', ', ': ')` separators,
//! `ensure_ascii=True`): every response line and every state file the helper
//! writes is emitted exactly so. Plus Python `==` over parsed values for the
//! idempotency comparisons (`saved != request`).

use soda_json::JsonValue;

/// Build an object preserving insertion order (response key order matters).
pub fn obj(pairs: Vec<(&str, JsonValue)>) -> JsonValue {
    JsonValue::Object(pairs.into_iter().map(|(k, v)| (k.to_string(), v)).collect())
}

pub fn str_value(text: &str) -> JsonValue {
    JsonValue::Str(text.to_string())
}

/// Emit one compact document with default separators, byte-identical to
/// CPython `json.dumps`.
pub fn dumps_default(value: &JsonValue) -> String {
    let mut out = String::new();
    emit(&mut out, value);
    out
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
                    out.push_str(", ");
                }
                emit(out, item);
            }
            out.push(']');
        }
        JsonValue::Object(entries) => {
            out.push('{');
            for (i, (key, item)) in entries.iter().enumerate() {
                if i > 0 {
                    out.push_str(", ");
                }
                escape_py(out, key);
                out.push_str(": ");
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
                out.push_str(&format!(
                    "\\u{:04x}\\u{:04x}",
                    0xd800 + (v >> 10),
                    0xdc00 + (v & 0x3ff)
                ));
            }
        }
    }
    out.push('"');
}

/// Python `==` over two parsed values: objects compare order-insensitive
/// with last-wins duplicates, and numbers compare numerically (`1 == 1.0`,
/// `True == 1`, unbounded ints exact).
pub fn json_equal(left: &JsonValue, right: &JsonValue) -> bool {
    match (left, right) {
        (JsonValue::Null, JsonValue::Null) => true,
        (JsonValue::Bool(a), JsonValue::Bool(b)) => a == b,
        (JsonValue::Str(a), JsonValue::Str(b)) => a == b,
        (JsonValue::Array(a), JsonValue::Array(b)) => {
            a.len() == b.len() && a.iter().zip(b.iter()).all(|(x, y)| json_equal(x, y))
        }
        (JsonValue::Object(a), JsonValue::Object(b)) => {
            let keys_a = unique_keys(a);
            let keys_b = unique_keys(b);
            if keys_a != keys_b {
                return false;
            }
            keys_a.iter().all(|key| {
                let Some(x) = left.get(key) else {
                    return false;
                };
                let Some(y) = right.get(key) else {
                    return false;
                };
                json_equal(x, y)
            })
        }
        (JsonValue::Number(a), JsonValue::Number(b)) => numbers_equal(a, b),
        (JsonValue::Bool(a), JsonValue::Number(b)) => numbers_equal(if *a { "1" } else { "0" }, b),
        (JsonValue::Number(a), JsonValue::Bool(b)) => numbers_equal(a, if *b { "1" } else { "0" }),
        _ => false,
    }
}

fn unique_keys(entries: &[(String, JsonValue)]) -> Vec<String> {
    let mut keys: Vec<String> = Vec::new();
    for (key, _) in entries {
        if !keys.contains(key) {
            keys.push(key.clone());
        }
    }
    keys.sort();
    keys
}

fn is_int_literal(raw: &str) -> bool {
    let digits = raw.strip_prefix('-').unwrap_or(raw);
    !digits.is_empty() && digits.bytes().all(|b| b.is_ascii_digit())
}

/// Normalize an integer literal for exact comparison (`-0` is `0`;
/// leading zeros cannot occur: the parser rejects them).
fn norm_int(raw: &str) -> (bool, &str) {
    let raw = if raw == "-0" { "0" } else { raw };
    match raw.strip_prefix('-') {
        Some(digits) => (true, digits),
        None => (false, raw),
    }
}

fn numbers_equal(left: &str, right: &str) -> bool {
    if is_int_literal(left) && is_int_literal(right) {
        let (neg_a, a) = norm_int(left);
        let (neg_b, b) = norm_int(right);
        return neg_a == neg_b && a.len() == b.len() && a == b;
    }
    // At least one side is a float literal: compare as f64, like CPython
    // for values in float range. (Out-of-range literals such as `1e400`
    // parse to infinity on both sides here; CPython can distinguish them,
    // but the protocol only carries small ints and strings.)
    match (left.parse::<f64>(), right.parse::<f64>()) {
        (Ok(a), Ok(b)) => a == b,
        _ => left == right,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn parse(text: &str) -> JsonValue {
        JsonValue::parse(text).expect("parse")
    }

    #[test]
    fn emit_matches_reference() {
        // Baked against CPython json.dumps (default separators, ensure_ascii).
        let value = obj(vec![
            ("roles", JsonValue::Array(vec![str_value("soda-coder")])),
            ("n", JsonValue::Number("-12".to_string())),
            ("t", JsonValue::Bool(true)),
            ("f", JsonValue::Bool(false)),
            ("z", JsonValue::Null),
            ("s", str_value("a+b/c<d>&\"q\"\\é\0\x1f\x7f😀")),
        ]);
        assert_eq!(
            dumps_default(&value),
            "{\"roles\": [\"soda-coder\"], \"n\": -12, \"t\": true, \"f\": false, \"z\": null, \
             \"s\": \"a+b/c<d>&\\\"q\\\"\\\\\\u00e9\\u0000\\u001f\\u007f\\ud83d\\ude00\"}"
        );
    }

    #[test]
    fn emit_state_files() {
        assert_eq!(
            dumps_default(&obj(vec![
                ("pid", JsonValue::Number("999".to_string())),
                ("pgid", JsonValue::Number("999".to_string())),
            ])),
            "{\"pid\": 999, \"pgid\": 999}"
        );
        assert_eq!(
            dumps_default(&obj(vec![
                ("setup_exit", JsonValue::Number("0".to_string())),
                ("check_exit", JsonValue::Null),
            ])),
            "{\"setup_exit\": 0, \"check_exit\": null}"
        );
        assert_eq!(
            dumps_default(&obj(vec![("stopped", JsonValue::Bool(true))])),
            "{\"stopped\": true}"
        );
        assert_eq!(dumps_default(&obj(vec![])), "{}");
    }

    #[test]
    fn equality_matches_reference() {
        // Order-insensitive, last-wins.
        assert!(json_equal(
            &parse("{\"a\": 1, \"b\": [1, {\"x\": null}]}"),
            &parse("{\"b\": [1, {\"x\": null}], \"a\": 1}")
        ));
        assert!(json_equal(
            &parse("{\"a\": 1, \"a\": 2}"),
            &parse("{\"a\": 2}")
        ));
        assert!(!json_equal(
            &parse("{\"a\": 1}"),
            &parse("{\"a\": 1, \"b\": 2}")
        ));
        // Numeric coercions.
        assert!(json_equal(&parse("{\"a\": 1}"), &parse("{\"a\": 1.0}")));
        assert!(json_equal(&parse("{\"a\": true}"), &parse("{\"a\": 1}")));
        assert!(json_equal(&parse("{\"a\": false}"), &parse("{\"a\": 0.0}")));
        assert!(!json_equal(&parse("{\"a\": true}"), &parse("{\"a\": 2}")));
        assert!(!json_equal(&parse("{\"a\": \"1\"}"), &parse("{\"a\": 1}")));
        assert!(!json_equal(&parse("{\"a\": 1}"), &parse("{\"a\": 2}")));
        assert!(!json_equal(
            &parse("{\"a\": null}"),
            &parse("{\"a\": false}")
        ));
        // Unbounded ints.
        let big = "9".repeat(100);
        assert!(json_equal(
            &parse(&format!("{{\"a\": {big}}}")),
            &parse(&format!("{{\"a\": {big}}}"))
        ));
        assert!(!json_equal(
            &parse(&format!("{{\"a\": {big}}}")),
            &parse(&format!("{{\"a\": {big}0}}"))
        ));
        assert!(json_equal(&parse("{\"a\": -0}"), &parse("{\"a\": 0}")));
    }
}
