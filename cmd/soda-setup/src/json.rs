//! Go-compatible string escaping for the setup-owned config producer.

use serde_json::ser::Formatter;
use std::io::{self, Write};

pub(crate) struct GoFormatter;

impl Formatter for GoFormatter {
    fn write_string_fragment<W>(&mut self, writer: &mut W, fragment: &str) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        let mut start = 0;
        for (index, ch) in fragment.char_indices() {
            let escape: Option<&[u8]> = match ch {
                '<' => Some(b"\\u003c"),
                '>' => Some(b"\\u003e"),
                '&' => Some(b"\\u0026"),
                '\u{2028}' => Some(b"\\u2028"),
                '\u{2029}' => Some(b"\\u2029"),
                _ => None,
            };
            if let Some(escape) = escape {
                writer.write_all(fragment[start..index].as_bytes())?;
                writer.write_all(escape)?;
                start = index + ch.len_utf8();
            }
        }
        writer.write_all(fragment[start..].as_bytes())
    }
}
