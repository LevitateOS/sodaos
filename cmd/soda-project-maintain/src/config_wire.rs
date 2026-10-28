use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::fmt;

use super::config::Config;

pub(crate) fn decode_host_config(data: &[u8]) -> Result<Config, String> {
    let mut stream = serde_json::Deserializer::from_slice(data).into_iter::<Option<Config>>();
    match stream.next() {
        Some(Ok(Some(config))) => Ok(config),
        Some(Ok(None)) => Ok(Config::default()),
        Some(Err(_)) => Err(String::from("invalid host config JSON")),
        None => Err(String::from("invalid host config JSON")),
    }
}

impl<'de> Deserialize<'de> for Config {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct ConfigVisitor;
        impl<'de> Visitor<'de> for ConfigVisitor {
            type Value = Config;

            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a host config object")
            }

            fn visit_map<A>(self, mut map: A) -> Result<Config, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut config = Config::default();
                while let Some(key) = map.next_key::<String>()? {
                    let Some(slot) = host_field_slot(&key) else {
                        return Err(de::Error::unknown_field(&key, HOST_FIELDS));
                    };
                    if set_early_field(&mut map, &mut config, slot)? {
                        continue;
                    }
                    if set_late_field(&mut map, &mut config, slot)? {
                        continue;
                    }
                    return Err(de::Error::unknown_field(&key, HOST_FIELDS));
                }
                Ok(config)
            }

            fn visit_unit<E>(self) -> Result<Config, E>
            where
                E: de::Error,
            {
                Ok(Config::default())
            }
        }
        deserializer.deserialize_any(ConfigVisitor)
    }
}

fn set_string<'de, A>(map: &mut A, slot: &mut String) -> Result<(), A::Error>
where
    A: MapAccess<'de>,
{
    let value = map.next_value::<Option<String>>()?;
    if let Some(value) = value {
        *slot = value;
    }
    Ok(())
}

const HOST_FIELDS: &[&str] = &[
    "muse_sha256",
    "muse_version",
    "muse_socket",
    "identity_socket",
    "codex_harness",
    "codex_harness_sha256",
    "codex_harness_version",
    "muse_harness",
    "muse_harness_sha256",
    "muse_harness_version",
    "image",
    "network",
    "subnet",
    "bridge",
];

fn set_early_field<'de, A>(map: &mut A, config: &mut Config, slot: usize) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    let target = match slot {
        0 => &mut config.muse_sha256,
        1 => &mut config.muse_version,
        2 => &mut config.muse_socket,
        3 => &mut config.identity_socket,
        4 => &mut config.codex_harness,
        5 => &mut config.codex_harness_sha256,
        6 => &mut config.codex_harness_version,
        _ => return Ok(false),
    };
    set_string(map, target)?;
    Ok(true)
}

fn set_late_field<'de, A>(map: &mut A, config: &mut Config, slot: usize) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    let target = match slot {
        7 => &mut config.muse_harness,
        8 => &mut config.muse_harness_sha256,
        9 => &mut config.muse_harness_version,
        10 => &mut config.image,
        11 => &mut config.network,
        12 => &mut config.subnet,
        13 => &mut config.bridge,
        _ => return Ok(false),
    };
    set_string(map, target)?;
    Ok(true)
}

fn host_field_slot(key: &str) -> Option<usize> {
    if let Some(i) = HOST_FIELDS.iter().position(|field| *field == key) {
        return Some(i);
    }
    let mut found = None;
    for (i, field) in HOST_FIELDS.iter().enumerate() {
        if field.eq_ignore_ascii_case(key) {
            if found.is_some() {
                return None;
            }
            found = Some(i);
        }
    }
    found
}

pub(crate) fn go_quoted(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    out.push('"');
    for c in s.chars() {
        match c {
            '\u{07}' => out.push_str("\\a"),
            '\u{08}' => out.push_str("\\b"),
            '\u{0c}' => out.push_str("\\f"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\u{0b}' => out.push_str("\\v"),
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            c if (c as u32) < 0x80 => push_go_ascii(&mut out, c),
            c => push_go_unicode(&mut out, c),
        }
    }
    out.push('"');
    out
}

fn push_go_ascii(out: &mut String, c: char) {
    if c.is_ascii_graphic() || c == ' ' {
        out.push(c);
    } else {
        out.push_str(&format!("\\x{:02x}", c as u32));
    }
}

fn push_go_unicode(out: &mut String, c: char) {
    if c.is_control() || (c.is_whitespace() && c != ' ') || is_go_nonprint(c) {
        let n = c as u32;
        if n <= 0xffff {
            out.push_str(&format!("\\u{n:04x}"));
        } else {
            out.push_str(&format!("\\U{n:08x}"));
        }
    } else {
        out.push(c);
    }
}

fn is_go_nonprint(c: char) -> bool {
    matches!(c as u32,
        0x00AD | 0x061C | 0x06DD | 0x070F | 0x08E2 | 0x180E | 0xFEFF
        | 0x110BD | 0x110CD | 0xE0001
        | 0x0600..=0x0605 | 0x200B..=0x200F | 0x202A..=0x202E
        | 0x2060..=0x2064 | 0x2066..=0x206F | 0xFFF9..=0xFFFB
        | 0x13430..=0x13438 | 0x1BCA0..=0x1BCA3 | 0x1D173..=0x1D17A
        | 0xE0020..=0xE007F | 0xE0100..=0xE01EF
        | 0xE000..=0xF8FF | 0xF0000..=0xFFFFD | 0x100000..=0x10FFFD)
}
