//! Python JSON output profiles and strict object-shape helpers.

use crate::state_json::StateValue;

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
