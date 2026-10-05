//! Go-exact JSON decoding helpers and emission.
//!
//! Decoding mirrors `encoding/json` struct rules used by the pipeline:
//! exact key match first with an ASCII case-insensitive fallback, `null`
//! leaving a field at its default, strict mode rejecting unknown fields,
//! and trailing data rejected. Emission mirrors `json.Marshal` (compact)
//! and `json.MarshalIndent(v, "", "  ")`; object entries keep caller order
//! and callers sort decoded maps first, matching Go's sorted map keys.

use soda_json::{escape_into, JsonValue};

use crate::error::Error;

pub fn parse(text: &str) -> Result<JsonValue, Error> {
    JsonValue::parse(text).map_err(|_| Error::msg("invalid JSON"))
}

/// Reject unknown object keys the way `DisallowUnknownFields` does. Matching
/// is exact-first with an ASCII case-insensitive fallback, like Go.
pub fn check_no_unknown(value: &JsonValue, known: &[&str]) -> Result<(), Error> {
    if let JsonValue::Object(entries) = value {
        for (key, _) in entries {
            let mut ok = false;
            for want in known {
                if key == want || key.eq_ignore_ascii_case(want) {
                    ok = true;
                    break;
                }
            }
            if !ok {
                return Err(Error::msg(format!("unknown field {key:?}")));
            }
        }
    }
    Ok(())
}

fn field<'a>(value: &'a JsonValue, name: &str) -> Option<&'a JsonValue> {
    if let JsonValue::Object(entries) = value {
        if let Some(found) = entries.iter().find(|(k, _)| k == name).map(|(_, v)| v) {
            return Some(found);
        }
        return entries
            .iter()
            .find(|(k, _)| k.eq_ignore_ascii_case(name))
            .map(|(_, v)| v);
    }
    None
}

fn is_null(value: Option<&JsonValue>) -> bool {
    matches!(value, None | Some(JsonValue::Null))
}

/// Required string field. Missing key decodes to the zero value only for
/// optional fields; required fields error, matching decode-then-validate.
pub fn require_string(value: &JsonValue, field_name: &str) -> Result<String, Error> {
    match field(value, field_name) {
        Some(JsonValue::Str(s)) => Ok(s.clone()),
        Some(JsonValue::Null) | None => Ok(String::new()),
        Some(_) => Err(Error::msg(format!("invalid {field_name}"))),
    }
}

/// Optional string with presence bit (for `json.RawMessage != nil` checks):
/// present means the key exists with a non-null value.
pub fn raw_present(value: &JsonValue, field_name: &str) -> bool {
    !is_null(field(value, field_name))
}

pub fn require_bool(value: &JsonValue, field_name: &str) -> Result<bool, Error> {
    match field(value, field_name) {
        Some(JsonValue::Bool(b)) => Ok(*b),
        Some(JsonValue::Null) | None => Ok(false),
        Some(_) => Err(Error::msg(format!("invalid {field_name}"))),
    }
}

fn number_raw(value: &JsonValue, field_name: &str) -> Result<Option<String>, Error> {
    match value {
        JsonValue::Number(raw) => Ok(Some(raw.clone())),
        JsonValue::Null => Ok(None),
        _ => Err(Error::msg(format!("invalid {field_name}"))),
    }
}

/// Go integer decode: whole JSON numbers only, erroring on fractions,
/// exponents, overflow, or wrong types.
pub fn require_i64(value: &JsonValue, field_name: &str) -> Result<i64, Error> {
    match field(value, field_name) {
        None | Some(JsonValue::Null) => Ok(0),
        Some(v) => match number_raw(v, field_name)? {
            None => Ok(0),
            Some(raw) => raw
                .parse::<i64>()
                .map_err(|_| Error::msg(format!("invalid {field_name}"))),
        },
    }
}

pub fn require_u64(value: &JsonValue, field_name: &str) -> Result<u64, Error> {
    match field(value, field_name) {
        None | Some(JsonValue::Null) => Ok(0),
        Some(v) => match number_raw(v, field_name)? {
            None => Ok(0),
            Some(raw) => raw
                .parse::<u64>()
                .map_err(|_| Error::msg(format!("invalid {field_name}"))),
        },
    }
}

pub fn require_object<'a>(value: &'a JsonValue, field_name: &str) -> Result<&'a JsonValue, Error> {
    match field(value, field_name) {
        None | Some(JsonValue::Null) => Ok(&JsonValue::Null),
        Some(v @ JsonValue::Object(_)) => Ok(v),
        Some(_) => Err(Error::msg(format!("invalid {field_name}"))),
    }
}

pub fn require_array<'a>(
    value: &'a JsonValue,
    field_name: &str,
) -> Result<Vec<&'a JsonValue>, Error> {
    match field(value, field_name) {
        None | Some(JsonValue::Null) => Ok(Vec::new()),
        Some(JsonValue::Array(items)) => Ok(items.iter().collect()),
        Some(_) => Err(Error::msg(format!("invalid {field_name}"))),
    }
}

/// Decode a `map[string]string` field; missing/null decodes to empty.
pub fn string_map(value: &JsonValue, field_name: &str) -> Result<Vec<(String, String)>, Error> {
    let mut out = Vec::new();
    if let JsonValue::Object(entries) = require_object(value, field_name)? {
        for (k, v) in entries {
            match v {
                JsonValue::Str(s) => out.push((k.clone(), s.clone())),
                _ => return Err(Error::msg(format!("invalid {field_name}"))),
            }
        }
    }
    Ok(out)
}

/// Compact emission, like `json.Marshal`.
pub fn write_compact(out: &mut String, value: &JsonValue) {
    match value {
        JsonValue::Null => out.push_str("null"),
        JsonValue::Bool(true) => out.push_str("true"),
        JsonValue::Bool(false) => out.push_str("false"),
        JsonValue::Number(raw) => out.push_str(raw),
        JsonValue::Str(s) => escape_into(out, s),
        JsonValue::Array(items) => {
            out.push('[');
            for (i, item) in items.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                write_compact(out, item);
            }
            out.push(']');
        }
        JsonValue::Object(entries) => {
            out.push('{');
            for (i, (key, item)) in entries.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                escape_into(out, key);
                out.push(':');
                write_compact(out, item);
            }
            out.push('}');
        }
    }
}

/// Two-space indented emission, like `json.MarshalIndent(v, "", "  ")`.
/// No trailing newline; callers append it like the Go owner does.
pub fn write_indent(out: &mut String, value: &JsonValue) {
    write_indent_at(out, value, 0);
}

fn write_indent_at(out: &mut String, value: &JsonValue, depth: usize) {
    match value {
        JsonValue::Array(items) if !items.is_empty() => {
            out.push_str("[\n");
            for (i, item) in items.iter().enumerate() {
                if i > 0 {
                    out.push_str(",\n");
                }
                write_padding(out, depth + 1);
                write_indent_at(out, item, depth + 1);
            }
            out.push('\n');
            write_padding(out, depth);
            out.push(']');
        }
        JsonValue::Object(entries) if !entries.is_empty() => {
            out.push_str("{\n");
            for (i, (key, item)) in entries.iter().enumerate() {
                if i > 0 {
                    out.push_str(",\n");
                }
                write_padding(out, depth + 1);
                escape_into(out, key);
                out.push_str(": ");
                write_indent_at(out, item, depth + 1);
            }
            out.push('\n');
            write_padding(out, depth);
            out.push('}');
        }
        _ => write_compact(out, value),
    }
}

fn write_padding(out: &mut String, depth: usize) {
    for _ in 0..depth {
        out.push_str("  ");
    }
}

pub fn to_compact(value: &JsonValue) -> String {
    let mut out = String::new();
    write_compact(&mut out, value);
    out
}

pub fn to_indent(value: &JsonValue) -> String {
    let mut out = String::new();
    write_indent(&mut out, value);
    out
}

/// Format a float64 the way Go's `encoding/json` does for small magnitudes:
/// shortest round-trip, no exponent.
pub fn format_float_go(value: f64) -> String {
    if value == value.trunc() && value.abs() < 1e15 {
        format!("{}", value.trunc() as i64)
    } else {
        format!("{value}")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_indent_matches_go_marshal_indent() {
        // Oracle: Go json.MarshalIndent(map[string]string{"b":"1","a":"x <&>"}, "", "  ").
        let value = JsonValue::Object(vec![
            ("a".to_string(), JsonValue::Str("x <&>".to_string())),
            ("b".to_string(), JsonValue::Str("1".to_string())),
        ]);
        assert_eq!(
            to_indent(&value),
            "{\n  \"a\": \"x \\u003c\\u0026\\u003e\",\n  \"b\": \"1\"\n}"
        );
    }

    #[test]
    fn oracle_empty_containers_emit_bare() {
        assert_eq!(to_indent(&JsonValue::Array(vec![])), "[]");
        assert_eq!(to_indent(&JsonValue::Object(vec![])), "{}");
    }

    #[test]
    fn oracle_strict_decode_rejects_unknown_and_trailing() {
        let value = parse("{\"Format\":3,\"Bogus\":1}").unwrap();
        assert!(check_no_unknown(&value, &["Format"]).is_err());
        assert!(parse("{\"Format\":3} trailing").is_err());
        // Null leaves the default, like Go struct decode.
        let value = parse("{\"Format\":null}").unwrap();
        assert_eq!(require_i64(&value, "Format").unwrap(), 0);
        assert_eq!(require_string(&value, "Missing").unwrap(), "");
    }
}
