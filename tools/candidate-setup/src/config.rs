use std::io::{self, Write};

use serde::Serialize;
use serde_json::ser::Formatter;

#[derive(Default)]
struct EnsureAsciiPretty {
    indent: usize,
    has_value: bool,
}

impl Formatter for EnsureAsciiPretty {
    fn write_string_fragment<W>(&mut self, writer: &mut W, fragment: &str) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        for ch in fragment.chars() {
            if ch <= '\u{7e}' {
                let mut bytes = [0; 4];
                writer.write_all(ch.encode_utf8(&mut bytes).as_bytes())?;
            } else {
                let mut units = [0u16; 2];
                for unit in ch.encode_utf16(&mut units) {
                    write!(writer, "\\u{unit:04x}")?;
                }
            }
        }
        Ok(())
    }

    fn begin_array<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        self.indent += 1;
        self.has_value = false;
        writer.write_all(b"[")
    }

    fn end_array<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        self.indent -= 1;
        if self.has_value {
            writer.write_all(b"\n")?;
            self.write_indent(writer)?;
        }
        writer.write_all(b"]")
    }

    fn begin_array_value<W>(&mut self, writer: &mut W, first: bool) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        writer.write_all(if first { b"\n" } else { b",\n" })?;
        self.write_indent(writer)
    }

    fn end_array_value<W>(&mut self, _: &mut W) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        self.has_value = true;
        Ok(())
    }

    fn begin_object<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        self.indent += 1;
        self.has_value = false;
        writer.write_all(b"{")
    }

    fn end_object<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        self.indent -= 1;
        if self.has_value {
            writer.write_all(b"\n")?;
            self.write_indent(writer)?;
        }
        writer.write_all(b"}")
    }

    fn begin_object_key<W>(&mut self, writer: &mut W, first: bool) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        writer.write_all(if first { b"\n" } else { b",\n" })?;
        self.write_indent(writer)
    }

    fn begin_object_value<W>(&mut self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        writer.write_all(b": ")
    }

    fn end_object_value<W>(&mut self, _: &mut W) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        self.has_value = true;
        Ok(())
    }
}

impl EnsureAsciiPretty {
    fn write_indent<W>(&self, writer: &mut W) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        for _ in 0..self.indent {
            writer.write_all(b"  ")?;
        }
        Ok(())
    }
}

fn pretty_json(value: &impl Serialize) -> String {
    let mut output = Vec::new();
    {
        let mut serializer =
            serde_json::Serializer::with_formatter(&mut output, EnsureAsciiPretty::default());
        value
            .serialize(&mut serializer)
            .expect("serializing a JSON record to Vec cannot fail");
    }
    String::from_utf8(output).expect("JSON serialization emits UTF-8")
}

#[derive(Serialize)]
#[allow(non_snake_case)]
struct WorkerRecord<'a> {
    Executable: &'a str,
    Source: &'a str,
    ForgejoSource: &'a str,
    OutputParent: &'a str,
    StorageRoot: &'a str,
    BuildHome: &'a str,
    Runtime: &'a str,
    Tools: &'a str,
    MediaAuthorityDirectory: &'a str,
}

#[derive(Serialize)]
#[allow(non_snake_case)]
struct TrustRecord<'a> {
    Format: u8,
    Prefix: &'a str,
    Epoch: u8,
    Keys: TrustKeys<'a>,
    NotBefore: u64,
    MaxAgeSeconds: u16,
    ClockSkewSeconds: u8,
    MinimumSequence: MinimumSequence,
}

#[derive(Serialize)]
struct TrustKeys<'a> {
    artifact: [&'a str; 1],
    candidate: [&'a str; 1],
    preview: [&'a str; 1],
    stable: [&'a str; 1],
}

#[derive(Serialize)]
struct MinimumSequence {
    candidate: u8,
    preview: u8,
    stable: u8,
}

#[derive(Serialize)]
#[allow(non_snake_case)]
struct ConfigRecord {
    Trust: &'static str,
    Keys: ConfigKeys,
}

#[derive(Serialize)]
#[allow(non_snake_case)]
struct ConfigKeys {
    Key: &'static str,
    Passphrase: &'static str,
}

/// `worker_json` mirrors Python's `json.dump(..., indent=2)` without a final LF.
#[allow(clippy::too_many_arguments)]
pub(super) fn worker_json(
    executable: &str,
    source: &str,
    forgejo_source: &str,
    output_parent: &str,
    storage_root: &str,
    build_home: &str,
    runtime: &str,
    tools: &str,
    authority: &str,
) -> String {
    pretty_json(&WorkerRecord {
        Executable: executable,
        Source: source,
        ForgejoSource: forgejo_source,
        OutputParent: output_parent,
        StorageRoot: storage_root,
        BuildHome: build_home,
        Runtime: runtime,
        Tools: tools,
        MediaAuthorityDirectory: authority,
    })
}

/// `trust_json` mirrors Python's `json.dumps(..., indent=2) + "\n"` trust file.
pub(super) fn trust_json(prefix: &str, now: u64, pubs: [&str; 4]) -> String {
    let record = TrustRecord {
        Format: 1,
        Prefix: prefix,
        Epoch: 1,
        Keys: TrustKeys {
            artifact: [pubs[0]],
            candidate: [pubs[1]],
            preview: [pubs[2]],
            stable: [pubs[3]],
        },
        NotBefore: now - 600,
        MaxAgeSeconds: 3600,
        ClockSkewSeconds: 10,
        MinimumSequence: MinimumSequence {
            candidate: 1,
            preview: 1,
            stable: 1,
        },
    };
    pretty_json(&record) + "\n"
}

/// `config_json` mirrors Python's `json.dumps(..., indent=2) + "\n"` config file.
pub(super) fn config_json() -> String {
    pretty_json(&ConfigRecord {
        Trust: "/run/soda-media-authority/trust.json",
        Keys: ConfigKeys {
            Key: "/run/soda-media-authority/artifact.private",
            Passphrase: "/run/soda-media-authority/passphrase",
        },
    }) + "\n"
}
