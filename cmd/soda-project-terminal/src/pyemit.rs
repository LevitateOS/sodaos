//! Python JSON output profiles and strict object-shape helpers.

use crate::state_json::StateValue;

/// Emit one compact ASCII JSON document, byte-identical to CPython.
#[cfg(test)]
pub fn dumps(value: &StateValue) -> String {
    dumps_serde(value)
}

/// Compact document plus `\n`.
pub fn line(value: &StateValue) -> Vec<u8> {
    line_serde(value)
}

/// Plain Python `json.dumps` separators and `ensure_ascii=True`.
pub fn dumps_default(value: &StateValue) -> String {
    dumps_default_serde(value)
}

/// Strict JSON integer as i64 (no fraction/exponent).
pub fn as_int(value: &StateValue) -> Option<i64> {
    value.as_integer().and_then(|n| i64::try_from(n).ok())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn obj(pairs: Vec<(&str, StateValue)>) -> StateValue {
        StateValue::Object(pairs.into_iter().map(|(k, v)| (k.to_string(), v)).collect())
    }

    #[test]
    fn emit_matches_cpython() {
        // Baked against CPython json.dumps(separators=(',',':'), ensure_ascii=True).
        let value = obj(vec![
            ("type", StateValue::Str("output".to_string())),
            (
                "data",
                StateValue::Str("a+b/c<d>&\"q\"\\é\0\x1f\x7f😀".to_string()),
            ),
            ("n", StateValue::Number("-12".to_string())),
            ("t", StateValue::Bool(true)),
            ("f", StateValue::Bool(false)),
            ("z", StateValue::Null),
            (
                "a",
                StateValue::Array(vec![StateValue::Number("1".to_string())]),
            ),
        ]);
        assert_eq!(
            dumps(&value),
            "{\"type\":\"output\",\"data\":\"a+b/c<d>&\\\"q\\\"\\\\\\u00e9\\u0000\\u001f\\u007f\\ud83d\\ude00\",\"n\":-12,\"t\":true,\"f\":false,\"z\":null,\"a\":[1]}"
        );
    }

    #[test]
    fn emit_default_separators() {
        // Baked against CPython json.dumps (default separators, ensure_ascii).
        let value = obj(vec![
            (
                "lease",
                obj(vec![
                    ("a", StateValue::Number("1".to_string())),
                    (
                        "b",
                        StateValue::Array(vec![
                            StateValue::Number("1".to_string()),
                            StateValue::Number("2".to_string()),
                        ]),
                    ),
                ]),
            ),
            ("credential", StateValue::Str(String::new())),
        ]);
        assert_eq!(
            dumps_default(&value),
            "{\"lease\": {\"a\": 1, \"b\": [1, 2]}, \"credential\": \"\"}"
        );
        assert_eq!(dumps_default(&obj(vec![])), "{}");
        assert_eq!(
            dumps_default(&StateValue::Str("é".to_string())),
            "\"\\u00e9\""
        );
    }

    #[test]
    fn emit_short_escapes() {
        assert_eq!(
            dumps(&StateValue::Str("\x08\x0c\n\r\t".to_string())),
            "\"\\b\\f\\n\\r\\t\""
        );
        assert_eq!(dumps(&StateValue::Str("/".to_string())), "\"/\"");
    }

    #[test]
    fn ordinary_dictionary_duplicates_keep_last_value_and_first_position() {
        let value = StateValue::parse(r#"{"a":1,"b":2,"a":3}"#).unwrap();
        assert_eq!(dumps(&value), r#"{"a":3,"b":2}"#);
    }

    #[test]
    fn int_shapes() {
        assert_eq!(as_int(&StateValue::Number("42".to_string())), Some(42));
        assert_eq!(as_int(&StateValue::Number("-1".to_string())), Some(-1));
        assert_eq!(as_int(&StateValue::Number("4.0".to_string())), None);
        assert_eq!(as_int(&StateValue::Number("1e3".to_string())), None);
        assert_eq!(as_int(&StateValue::Bool(true)), None);
        assert_eq!(
            as_int(&StateValue::Number("9223372036854775808".to_string())),
            None
        );
    }
}

/// Serialize a Serde-owned value using Python `ensure_ascii=True` and compact
/// separators. Ordered application values supply their own ordered map view.
pub fn dumps_serde<T: serde::Serialize + ?Sized>(value: &T) -> String {
    dumps_with_profile(value, false)
}

/// Serialize a Serde-owned value using Python `json.dumps` default separators.
pub fn dumps_default_serde<T: serde::Serialize + ?Sized>(value: &T) -> String {
    dumps_with_profile(value, true)
}

pub fn line_serde<T: serde::Serialize + ?Sized>(value: &T) -> Vec<u8> {
    let mut text = dumps_serde(value);
    text.push('\n');
    text.into_bytes()
}

fn dumps_with_profile<T: serde::Serialize + ?Sized>(value: &T, spaced: bool) -> String {
    let mut bytes = Vec::new();
    let mut serializer =
        serde_json::Serializer::with_formatter(&mut bytes, PythonFormatter { spaced });
    value
        .serialize(&mut serializer)
        .expect("serialize JSON output");
    String::from_utf8(bytes).expect("JSON output is UTF-8")
}

struct PythonFormatter {
    spaced: bool,
}

impl serde_json::ser::Formatter for PythonFormatter {
    fn write_string_fragment<W: ?Sized + std::io::Write>(
        &mut self,
        writer: &mut W,
        fragment: &str,
    ) -> std::io::Result<()> {
        for character in fragment.chars() {
            let code = character as u32;
            if code == 0x7f || code >= 0x80 {
                if code <= 0xffff {
                    write!(writer, "\\u{code:04x}")?;
                } else {
                    let value = code - 0x10000;
                    write!(
                        writer,
                        "\\u{:04x}\\u{:04x}",
                        0xd800 + (value >> 10),
                        0xdc00 + (value & 0x3ff)
                    )?;
                }
            } else {
                let mut buffer = [0; 4];
                writer.write_all(character.encode_utf8(&mut buffer).as_bytes())?;
            }
        }
        Ok(())
    }

    fn begin_array_value<W: ?Sized + std::io::Write>(
        &mut self,
        writer: &mut W,
        first: bool,
    ) -> std::io::Result<()> {
        if first {
            Ok(())
        } else if self.spaced {
            writer.write_all(b", ")
        } else {
            writer.write_all(b",")
        }
    }

    fn begin_object_key<W: ?Sized + std::io::Write>(
        &mut self,
        writer: &mut W,
        first: bool,
    ) -> std::io::Result<()> {
        if first {
            Ok(())
        } else if self.spaced {
            writer.write_all(b", ")
        } else {
            writer.write_all(b",")
        }
    }

    fn begin_object_value<W: ?Sized + std::io::Write>(
        &mut self,
        writer: &mut W,
    ) -> std::io::Result<()> {
        if self.spaced {
            writer.write_all(b": ")
        } else {
            writer.write_all(b":")
        }
    }
}

#[cfg(test)]
mod serde_output_tests {
    use super::{dumps_default_serde, dumps_serde};

    #[derive(serde::Serialize)]
    struct Reply<'a> {
        value: &'a str,
        items: [i64; 2],
    }

    #[test]
    fn python_ascii_and_separator_profiles() {
        let value = Reply {
            value: "é😀\x7f",
            items: [1, 2],
        };
        assert_eq!(
            dumps_serde(&value),
            r#"{"value":"\u00e9\ud83d\ude00\u007f","items":[1,2]}"#
        );
        assert_eq!(
            dumps_default_serde(&value),
            r#"{"value": "\u00e9\ud83d\ude00\u007f", "items": [1, 2]}"#
        );
    }
}
