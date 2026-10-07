//! Serde adapters for release-image JSON input and Go-compatible output.

use serde::de::{self, DeserializeOwned, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};
use serde_json::ser::{CharEscape, Formatter, PrettyFormatter, Serializer as JsonSerializer};
use std::io;

/// Preserve JSON object member order for domain maps whose caller observes
/// first-match or duplicate behavior.
pub(crate) struct OrderedMap<T>(pub Vec<(String, T)>);

impl<T> Default for OrderedMap<T> {
    fn default() -> Self {
        Self(Vec::new())
    }
}

/// Serialize a Go-style map view sorted by key while retaining duplicate
/// source pairs and their stable order for equal keys.
pub(crate) struct SortedPairs<'a, T>(pub &'a [(String, T)]);

impl<T: Serialize> Serialize for SortedPairs<'_, T> {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        use serde::ser::SerializeMap;
        let mut entries: Vec<_> = self.0.iter().collect();
        entries.sort_by(|left, right| left.0.cmp(&right.0));
        let mut map = serializer.serialize_map(Some(entries.len()))?;
        for (key, value) in entries {
            map.serialize_entry(key, value)?;
        }
        map.end()
    }
}

impl<'de, T: Deserialize<'de>> Deserialize<'de> for OrderedMap<T> {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct OrderedMapVisitor<T>(std::marker::PhantomData<T>);
        impl<'de, T: Deserialize<'de>> Visitor<'de> for OrderedMapVisitor<T> {
            type Value = OrderedMap<T>;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("an object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let mut entries = Vec::new();
                while let Some((key, value)) = map.next_entry::<String, T>()? {
                    entries.push((key, value));
                }
                Ok(OrderedMap(entries))
            }
        }
        deserializer.deserialize_map(OrderedMapVisitor(std::marker::PhantomData))
    }
}

use crate::error::Error;

/// Deserialize one complete typed record or arbitrary JSON value.
pub fn parse<T: DeserializeOwned>(text: &str) -> Result<T, Error> {
    let mut decoder = serde_json::Deserializer::from_str(text);
    let value = T::deserialize(&mut decoder).map_err(|_| Error::msg("invalid JSON"))?;
    decoder.end().map_err(|_| Error::msg("invalid JSON"))?;
    Ok(value)
}

pub(crate) fn decode_field<T: DeserializeOwned + Default, E: de::Error>(
    raw: Option<Box<serde_json::value::RawValue>>,
) -> Result<T, E> {
    match raw {
        None => Ok(T::default()),
        // Go's strconv integer readers accept the JSON integer token -0 as
        // zero. Serde's signed/unsigned integer visitors reject that spelling.
        Some(raw) => {
            serde_json::from_str::<Option<T>>(if raw.get() == "-0" { "0" } else { raw.get() })
                .map(Option::unwrap_or_default)
                .map_err(E::custom)
        }
    }
}

/// Generates one concrete strict DTO visitor. Each DTO keeps its own field
/// list and types; raw slots implement the image owner's exact-first, then
/// first-folded lookup before typed conversion.
macro_rules! case_record {
    ($ty:ident, { $($field:ident : $field_type:ty => $name:literal),+ $(,)? }) => {
        impl<'de> serde::Deserialize<'de> for $ty {
            fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
                struct RecordVisitor;
                impl<'de> serde::de::Visitor<'de> for RecordVisitor {
                    type Value = $ty;
                    fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        f.write_str("known Soda JSON record")
                    }
                    fn visit_map<A: serde::de::MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                        $(let mut $field: (Option<Box<serde_json::value::RawValue>>, Option<Box<serde_json::value::RawValue>>) = (None, None);)+
                        while let Some(key) = map.next_key::<String>()? {
                            let mut matched = false;
                            $(if key == $name {
                                let raw = map.next_value::<Box<serde_json::value::RawValue>>()?;
                                if $field.0.is_none() { $field.0 = Some(raw); }
                                matched = true;
                            } else if key.eq_ignore_ascii_case($name) {
                                let raw = map.next_value::<Box<serde_json::value::RawValue>>()?;
                                if $field.1.is_none() { $field.1 = Some(raw); }
                                matched = true;
                            })+
                            if !matched {
                                return Err(serde::de::Error::custom(format!("unknown field {key:?}")));
                            }
                        }
                        Ok($ty { $($field: $crate::jsonio::decode_field::<$field_type, A::Error>($field.0.or($field.1))?,)+ })
                    }
                }
                deserializer.deserialize_map(RecordVisitor)
            }
        }
    };
}

pub(crate) use case_record;

struct GoFormatter<'a> {
    pretty: Option<PrettyFormatter<'a>>,
}

impl Formatter for GoFormatter<'_> {
    fn write_string_fragment<W>(&mut self, writer: &mut W, fragment: &str) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        let mut plain = 0;
        for (index, ch) in fragment.char_indices() {
            let escaped = match ch {
                '<' => Some(b"\\u003c".as_slice()),
                '>' => Some(b"\\u003e".as_slice()),
                '&' => Some(b"\\u0026".as_slice()),
                '\u{2028}' => Some(b"\\u2028".as_slice()),
                '\u{2029}' => Some(b"\\u2029".as_slice()),
                _ => None,
            };
            if let Some(escaped) = escaped {
                writer.write_all(fragment[plain..index].as_bytes())?;
                writer.write_all(escaped)?;
                plain = index + ch.len_utf8();
            }
        }
        writer.write_all(fragment[plain..].as_bytes())
    }

    fn write_char_escape<W>(&mut self, writer: &mut W, escape: CharEscape) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        match escape {
            CharEscape::Quote => writer.write_all(b"\\\""),
            CharEscape::ReverseSolidus => writer.write_all(b"\\\\"),
            CharEscape::Solidus => writer.write_all(b"\\/"),
            CharEscape::Backspace => writer.write_all(b"\\b"),
            CharEscape::FormFeed => writer.write_all(b"\\f"),
            CharEscape::LineFeed => writer.write_all(b"\\n"),
            CharEscape::CarriageReturn => writer.write_all(b"\\r"),
            CharEscape::Tab => writer.write_all(b"\\t"),
            CharEscape::AsciiControl(byte) => write!(writer, "\\u{byte:04x}"),
        }
    }

    fn begin_array<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        if let Some(pretty) = &mut self.pretty {
            pretty.begin_array(writer)
        } else {
            writer.write_all(b"[")
        }
    }
    fn end_array<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        if let Some(pretty) = &mut self.pretty {
            pretty.end_array(writer)
        } else {
            writer.write_all(b"]")
        }
    }
    fn begin_array_value<W>(&mut self, writer: &mut W, first: bool) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        if let Some(pretty) = &mut self.pretty {
            pretty.begin_array_value(writer, first)
        } else if !first {
            writer.write_all(b",")
        } else {
            Ok(())
        }
    }
    fn end_array_value<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        if let Some(pretty) = &mut self.pretty {
            pretty.end_array_value(writer)
        } else {
            Ok(())
        }
    }
    fn begin_object<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        if let Some(pretty) = &mut self.pretty {
            pretty.begin_object(writer)
        } else {
            writer.write_all(b"{")
        }
    }
    fn end_object<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        if let Some(pretty) = &mut self.pretty {
            pretty.end_object(writer)
        } else {
            writer.write_all(b"}")
        }
    }
    fn begin_object_key<W>(&mut self, writer: &mut W, first: bool) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        if let Some(pretty) = &mut self.pretty {
            pretty.begin_object_key(writer, first)
        } else if !first {
            writer.write_all(b",")
        } else {
            Ok(())
        }
    }
    fn begin_object_value<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        if let Some(pretty) = &mut self.pretty {
            pretty.begin_object_value(writer)
        } else {
            writer.write_all(b":")
        }
    }
    fn end_object_value<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        if let Some(pretty) = &mut self.pretty {
            pretty.end_object_value(writer)
        } else {
            Ok(())
        }
    }
}

fn serialize<T: Serialize + ?Sized>(value: &T, pretty: bool) -> String {
    let mut bytes = Vec::new();
    let formatter = GoFormatter {
        pretty: pretty.then(|| PrettyFormatter::with_indent(b"  ")),
    };
    let mut serializer = JsonSerializer::with_formatter(&mut bytes, formatter);
    value
        .serialize(&mut serializer)
        .expect("serialization to Vec cannot fail");
    String::from_utf8(bytes).expect("serde_json emits UTF-8")
}

pub fn to_compact<T: Serialize + ?Sized>(value: &T) -> String {
    serialize(value, false)
}
pub fn to_indent<T: Serialize + ?Sized>(value: &T) -> String {
    serialize(value, true)
}

/// Format a float64 the way Go's `encoding/json` does for small magnitudes.
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
    fn formatter_matches_go_indent_and_compact_escapes() {
        let value = serde_json::json!({"a": "x <&>", "b": "1"});
        assert_eq!(
            to_indent(&value),
            "{\n  \"a\": \"x \\u003c\\u0026\\u003e\",\n  \"b\": \"1\"\n}"
        );
        assert_eq!(to_indent(&serde_json::json!([])), "[]");
        assert_eq!(to_indent(&serde_json::json!({})), "{}");
        assert_eq!(
            to_compact("<>&\u{2028}\u{2029}"),
            "\"\\u003c\\u003e\\u0026\\u2028\\u2029\""
        );
    }
}
