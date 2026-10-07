//! Python-compatible JSON emission and bootstrap document construction.

use std::fmt;
use std::io;
use std::path::Path;

use serde::de::{MapAccess, SeqAccess, Visitor};
use serde::ser::{SerializeMap, SerializeSeq};
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use serde_json::ser::{CharEscape, Formatter, PrettyFormatter};
use serde_json::value::RawValue;

use super::private_files::read_text;
use super::{ProvError, ProvKind};

/// Application tree for the one dynamic Butane document. Object pairs retain
/// source order and duplicates; number leaves retain their validated token.
#[derive(Debug)]
pub(crate) enum Node {
    Null,
    Bool(bool),
    Number(Box<RawValue>),
    String(String),
    Array(Vec<Node>),
    Object(Vec<(String, Node)>),
}

enum RawChildren {
    Array(Vec<Box<RawValue>>),
    Object(Vec<(String, Box<RawValue>)>),
}

impl<'de> Deserialize<'de> for RawChildren {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct ChildrenVisitor;

        impl<'de> Visitor<'de> for ChildrenVisitor {
            type Value = RawChildren;

            fn expecting(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
                formatter.write_str("an ordered JSON object or array")
            }

            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                let mut values = Vec::with_capacity(seq.size_hint().unwrap_or(0));
                while let Some(value) = seq.next_element::<Box<RawValue>>()? {
                    values.push(value);
                }
                Ok(RawChildren::Array(values))
            }

            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut values = Vec::with_capacity(map.size_hint().unwrap_or(0));
                while let Some((key, value)) = map.next_entry::<String, Box<RawValue>>()? {
                    values.push((key, value));
                }
                Ok(RawChildren::Object(values))
            }
        }

        deserializer.deserialize_any(ChildrenVisitor)
    }
}

impl Node {
    fn parse(raw: &RawValue) -> Result<Node, serde_json::Error> {
        // RawValue has already admitted the token under Serde's JSON grammar;
        // inspect only its leading structural byte, never parse numbers into
        // machine numeric types.
        match raw.get().as_bytes()[0] {
            b'n' => Ok(Node::Null),
            b't' => Ok(Node::Bool(true)),
            b'f' => Ok(Node::Bool(false)),
            b'"' => Ok(Node::String(serde_json::from_str(raw.get())?)),
            b'-' | b'0'..=b'9' => Ok(Node::Number(serde_json::from_str(raw.get())?)),
            b'[' => {
                let RawChildren::Array(values) = serde_json::from_str(raw.get())? else {
                    unreachable!("shape was an array")
                };
                Ok(Node::Array(
                    values
                        .iter()
                        .map(|value| Node::parse(value))
                        .collect::<Result<_, _>>()?,
                ))
            }
            b'{' => {
                let RawChildren::Object(values) = serde_json::from_str(raw.get())? else {
                    unreachable!("shape was an object")
                };
                Ok(Node::Object(
                    values
                        .iter()
                        .map(|(key, value)| Ok((key.clone(), Node::parse(value)?)))
                        .collect::<Result<_, serde_json::Error>>()?,
                ))
            }
            _ => unreachable!("RawValue is a valid JSON token"),
        }
    }

    pub(crate) fn parse_document(text: &str) -> Result<Node, serde_json::Error> {
        let raw: Box<RawValue> = serde_json::from_str(text)?;
        Node::parse(&raw)
    }
}

impl Serialize for Node {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match self {
            Node::Null => serializer.serialize_unit(),
            Node::Bool(value) => serializer.serialize_bool(*value),
            Node::Number(raw) => raw.serialize(serializer),
            Node::String(value) => serializer.serialize_str(value),
            Node::Array(values) => {
                let mut seq = serializer.serialize_seq(Some(values.len()))?;
                for value in values {
                    seq.serialize_element(value)?;
                }
                seq.end()
            }
            Node::Object(values) => {
                let mut map = serializer.serialize_map(Some(values.len()))?;
                for (key, value) in values {
                    map.serialize_entry(key, value)?;
                }
                map.end()
            }
        }
    }
}

/// Python's `json.dump(indent=2)` layout with ensure_ascii enabled. Serde
/// owns JSON token emission; this formatter supplies the caller's exact layout
/// and lowercase ASCII escapes.
struct PythonFormatter(PrettyFormatter<'static>);

impl Default for PythonFormatter {
    fn default() -> Self {
        Self(PrettyFormatter::with_indent(b"  "))
    }
}

impl Formatter for PythonFormatter {
    fn write_string_fragment<W>(&mut self, writer: &mut W, fragment: &str) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        for ch in fragment.chars() {
            if (ch as u32) < 0x7f {
                let mut bytes = [0; 4];
                writer.write_all(ch.encode_utf8(&mut bytes).as_bytes())?;
            } else if ch.is_ascii() {
                write_unicode_escape(writer, ch as u32)?;
            } else if (ch as u32) <= 0xffff {
                write_unicode_escape(writer, ch as u32)?;
            } else {
                let value = ch as u32 - 0x10000;
                write_unicode_escape(writer, 0xd800 + (value >> 10))?;
                write_unicode_escape(writer, 0xdc00 + (value & 0x3ff))?;
            }
        }
        Ok(())
    }

    fn begin_array<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.0.begin_array(writer)
    }
    fn end_array<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.0.end_array(writer)
    }
    fn begin_array_value<W>(&mut self, writer: &mut W, first: bool) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.0.begin_array_value(writer, first)
    }
    fn end_array_value<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.0.end_array_value(writer)
    }
    fn begin_object<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.0.begin_object(writer)
    }
    fn end_object<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.0.end_object(writer)
    }
    fn begin_object_key<W>(&mut self, writer: &mut W, first: bool) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.0.begin_object_key(writer, first)
    }
    fn end_object_key<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.0.end_object_key(writer)
    }
    fn begin_object_value<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.0.begin_object_value(writer)
    }
    fn end_object_value<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        self.0.end_object_value(writer)
    }

    fn write_char_escape<W>(&mut self, writer: &mut W, escape: CharEscape) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        match escape {
            CharEscape::AsciiControl(byte) => write_unicode_escape(writer, byte as u32),
            other => self.0.write_char_escape(writer, other),
        }
    }
}

fn write_unicode_escape<W: ?Sized + io::Write>(writer: &mut W, value: u32) -> io::Result<()> {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let bytes = [
        b'\\',
        b'u',
        HEX[((value >> 12) & 15) as usize],
        HEX[((value >> 8) & 15) as usize],
        HEX[((value >> 4) & 15) as usize],
        HEX[(value & 15) as usize],
    ];
    writer.write_all(&bytes)
}

pub(crate) fn dump_python(value: &Node) -> String {
    let mut output = Vec::new();
    let mut serializer =
        serde_json::Serializer::with_formatter(&mut output, PythonFormatter::default());
    value
        .serialize(&mut serializer)
        .expect("serializing the Butane application tree is infallible");
    String::from_utf8(output).expect("Serde emits valid UTF-8")
}

pub(crate) fn object_mut(value: &mut Node) -> Option<&mut Vec<(String, Node)>> {
    match value {
        Node::Object(entries) => Some(entries),
        _ => None,
    }
}

fn get_mut<'a>(entries: &'a mut [(String, Node)], key: &str) -> Option<&'a mut Node> {
    entries
        .iter_mut()
        .rev()
        .find(|(k, _)| k == key)
        .map(|(_, v)| v)
}

/// Navigate to the storage file list, reporting `KeyError`/`TypeError`
/// like the script's bare subscripts. A non-list `files` entry reports
/// `AttributeError`, like the script's failed `.append`.
fn files_mut(config: &mut Node) -> Result<&mut Node, ProvError> {
    let entries = object_mut(config)
        .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap must be an object"))?;
    let storage = get_mut(entries, "storage")
        .ok_or_else(|| ProvError::new(ProvKind::Key, "bootstrap has no storage"))?;
    let storage_entries = object_mut(storage)
        .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap storage must be an object"))?;
    let files = get_mut(storage_entries, "files")
        .ok_or_else(|| ProvError::new(ProvKind::Key, "bootstrap storage has no files"))?;
    if matches!(files, Node::Array(_)) {
        Ok(files)
    } else {
        Err(ProvError::new(
            ProvKind::Attribute,
            "bootstrap files have no append",
        ))
    }
}

fn not_a_list() -> ProvError {
    ProvError::new(ProvKind::Attribute, "bootstrap files have no append")
}

pub(crate) fn push_file(config: &mut Node, entry: Node) -> Result<(), ProvError> {
    match files_mut(config)? {
        Node::Array(items) => {
            items.push(entry);
            Ok(())
        }
        _ => Err(not_a_list()),
    }
}

pub(crate) fn clear_files(config: &mut Node) -> Result<(), ProvError> {
    match files_mut(config)? {
        Node::Array(items) => {
            items.clear();
            Ok(())
        }
        _ => Err(not_a_list()),
    }
}

pub(crate) fn str_value(text: &str) -> Node {
    Node::String(text.to_string())
}

pub(crate) fn file_entry(path: &str, mode: u32, inline: &str) -> Node {
    Node::Object(vec![
        ("path".to_string(), str_value(path)),
        (
            "mode".to_string(),
            Node::Number(RawValue::from_string(mode.to_string()).expect("integer token")),
        ),
        (
            "contents".to_string(),
            Node::Object(vec![("inline".to_string(), str_value(inline))]),
        ),
    ])
}

/// One public bootstrap for private provisioning and installer-media
/// conversion: the base document plus the shared branding file.
pub(crate) fn public_config(source: &Path) -> Result<Node, ProvError> {
    let path = source.join("system/host/provisioning/base.json");
    let text = read_text(&path)?;
    let mut config = Node::parse_document(&text).map_err(|_| {
        ProvError::new(
            ProvKind::JsonDecode,
            format!("cannot parse {}", path.display()),
        )
    })?;
    let svg = read_text(&source.join("assets/branding/source/soda-symbol.svg"))?;
    push_file(
        &mut config,
        file_entry(
            "/var/usrlocal/share/icons/hicolor/scalable/apps/sodaos-icon.svg",
            0o644,
            &svg,
        ),
    )?;
    Ok(config)
}
