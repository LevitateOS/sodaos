//! Go-compatible JSON serialization for release-build producer records.

use serde::Serialize;
use serde_json::ser::{CharEscape, Formatter, PrettyFormatter, Serializer as JsonSerializer};
use std::io;

struct CompactGoFormatter;

impl Formatter for CompactGoFormatter {
    fn write_string_fragment<W>(&mut self, writer: &mut W, fragment: &str) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        write_go_string_fragment(writer, fragment)
    }

    fn write_char_escape<W>(&mut self, writer: &mut W, escape: CharEscape) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        write_go_char_escape(writer, escape)
    }
}

struct PrettyGoFormatter<'a> {
    pretty: PrettyFormatter<'a>,
}

impl Formatter for PrettyGoFormatter<'_> {
    fn write_string_fragment<W>(&mut self, writer: &mut W, fragment: &str) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        write_go_string_fragment(writer, fragment)
    }

    fn write_char_escape<W>(&mut self, writer: &mut W, escape: CharEscape) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        write_go_char_escape(writer, escape)
    }

    fn begin_array<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.pretty.begin_array(writer)
    }

    fn end_array<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.pretty.end_array(writer)
    }

    fn begin_array_value<W>(&mut self, writer: &mut W, first: bool) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.pretty.begin_array_value(writer, first)
    }

    fn end_array_value<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.pretty.end_array_value(writer)
    }

    fn begin_object<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.pretty.begin_object(writer)
    }

    fn end_object<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.pretty.end_object(writer)
    }

    fn begin_object_key<W>(&mut self, writer: &mut W, first: bool) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.pretty.begin_object_key(writer, first)
    }

    fn begin_object_value<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.pretty.begin_object_value(writer)
    }

    fn end_object_value<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.pretty.end_object_value(writer)
    }
}

fn write_go_string_fragment<W>(writer: &mut W, fragment: &str) -> io::Result<()>
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

fn write_go_char_escape<W>(writer: &mut W, escape: CharEscape) -> io::Result<()>
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

fn serialize<T: Serialize + ?Sized, F: Formatter>(value: &T, formatter: F) -> String {
    let mut bytes = Vec::new();
    let mut serializer = JsonSerializer::with_formatter(&mut bytes, formatter);
    value
        .serialize(&mut serializer)
        .expect("JSON serialization to Vec");
    String::from_utf8(bytes).expect("serde_json emits UTF-8")
}

/// Go `json.MarshalIndent(value, "", "  ")`, without a trailing newline.
pub fn marshal_indent<T: Serialize + ?Sized>(value: &T) -> String {
    serialize(
        value,
        PrettyGoFormatter {
            pretty: PrettyFormatter::with_indent(b"  "),
        },
    )
}

/// Go `json.Marshal(value)` byte layout, without a trailing newline.
pub fn marshal_compact<T: Serialize + ?Sized>(value: &T) -> String {
    serialize(value, CompactGoFormatter)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pretty_go_json_vectors() {
        #[derive(Serialize)]
        struct Vector<'a> {
            #[serde(rename = "URL")]
            url: &'a str,
            n: i32,
            ok: bool,
            list: Vec<serde_json::Value>,
            empty: Vec<String>,
            nested: Nested<'a>,
        }
        #[derive(Serialize)]
        struct Nested<'a> {
            k: &'a str,
        }
        let value = Vector {
            url: "https://x.test/a.iso",
            n: -3,
            ok: true,
            list: vec![serde_json::json!("a<b"), serde_json::json!(7)],
            empty: vec![],
            nested: Nested { k: "v&v" },
        };
        assert_eq!(
            marshal_indent(&value),
            "{\n  \"URL\": \"https://x.test/a.iso\",\n  \"n\": -3,\n  \"ok\": true,\n  \"list\": [\n    \"a\\u003cb\",\n    7\n  ],\n  \"empty\": [],\n  \"nested\": {\n    \"k\": \"v\\u0026v\"\n  }\n}"
        );
        assert_eq!(marshal_indent(&serde_json::json!({})), "{}");
        assert_eq!(
            marshal_compact(&"<>&\u{2028}\u{2029}"),
            "\"\\u003c\\u003e\\u0026\\u2028\\u2029\""
        );
    }
}
