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
                    match host_field_slot(&key) {
                        Some(0) => set_string::<A>(&mut map, &mut config.muse_sha256)?,
                        Some(1) => set_string::<A>(&mut map, &mut config.muse_version)?,
                        Some(2) => set_string::<A>(&mut map, &mut config.muse_socket)?,
                        Some(3) => set_string::<A>(&mut map, &mut config.identity_socket)?,
                        Some(4) => set_string::<A>(&mut map, &mut config.codex_harness)?,
                        Some(5) => set_string::<A>(&mut map, &mut config.codex_harness_sha256)?,
                        Some(6) => set_string::<A>(&mut map, &mut config.codex_harness_version)?,
                        Some(7) => set_bool::<A>(&mut map, &mut config.tailnet_management)?,
                        Some(8) => set_string::<A>(&mut map, &mut config.tailnet_image)?,
                        Some(9) => set_string::<A>(&mut map, &mut config.image)?,
                        Some(10) => set_string::<A>(&mut map, &mut config.network)?,
                        Some(11) => set_string::<A>(&mut map, &mut config.subnet)?,
                        Some(12) => set_string::<A>(&mut map, &mut config.bridge)?,
                        _ => return Err(de::Error::unknown_field(&key, HOST_FIELDS)),
                    }
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

fn set_bool<'de, A>(map: &mut A, slot: &mut bool) -> Result<(), A::Error>
where
    A: MapAccess<'de>,
{
    let value = map.next_value::<Option<bool>>()?;
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
    "tailnet_management",
    "tailnet_image",
    "image",
    "network",
    "subnet",
    "bridge",
];

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
            c if (c as u32) < 0x80 => {
                if c.is_ascii_graphic() || c == ' ' {
                    out.push(c);
                } else {
                    out.push_str(&format!("\\x{:02x}", c as u32));
                }
            }
            c if c.is_control() || (c.is_whitespace() && c != ' ') || is_go_nonprint(c) => {
                let n = c as u32;
                if n <= 0xffff {
                    out.push_str(&format!("\\u{n:04x}"));
                } else {
                    out.push_str(&format!("\\U{n:08x}"));
                }
            }
            c => out.push(c),
        }
    }
    out.push('"');
    out
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
