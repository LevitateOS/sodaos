use serde::de::{IgnoredAny, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};
use serde_json::ser::Formatter;
use std::collections::BTreeMap;
use std::fmt;
use std::io;

#[derive(Serialize)]
struct RegistrationDto<'a> {
    child_id: &'a str,
    actor_id: &'a str,
    registration_id: &'a str,
    muse: bool,
}

#[derive(Serialize)]
struct LaunchRequestDto<'a> {
    register: RegistrationDto<'a>,
    connection_id: &'static str,
    cwd: &'static str,
    args: Option<&'a [String]>,
    tty: bool,
    cols: u32,
    rows: u32,
}

#[derive(Serialize)]
struct ComposeOverrideDto {
    services: BTreeMap<String, ComposeServiceDto>,
}

#[derive(Serialize)]
struct ComposeServiceDto {
    volumes: Vec<String>,
}

/// Go's encoding/json escapes HTML-sensitive characters and the two Unicode
/// line separators even though JSON itself does not require those escapes.
#[derive(Default)]
struct GoHtmlFormatter;

impl Formatter for GoHtmlFormatter {
    fn write_string_fragment<W>(&mut self, writer: &mut W, fragment: &str) -> io::Result<()>
    where
        W: ?Sized + io::Write,
    {
        let mut start = 0;
        for (index, character) in fragment.char_indices() {
            let escape = match character {
                '<' => Some(b"\\u003c".as_slice()),
                '>' => Some(b"\\u003e".as_slice()),
                '&' => Some(b"\\u0026".as_slice()),
                '\u{2028}' => Some(b"\\u2028".as_slice()),
                '\u{2029}' => Some(b"\\u2029".as_slice()),
                _ => None,
            };
            if let Some(escape) = escape {
                writer.write_all(&fragment.as_bytes()[start..index])?;
                writer.write_all(escape)?;
                start = index + character.len_utf8();
            }
        }
        writer.write_all(&fragment.as_bytes()[start..])
    }
}

fn to_go_json<T: Serialize + ?Sized>(value: &T) -> String {
    let mut bytes = Vec::new();
    let mut serializer = serde_json::Serializer::with_formatter(&mut bytes, GoHtmlFormatter);
    value
        .serialize(&mut serializer)
        .expect("serializing to a Vec cannot fail");
    String::from_utf8(bytes).expect("serde_json emits UTF-8")
}

pub(crate) fn json_string(value: &str) -> String {
    to_go_json(value)
}

pub(crate) fn launch_request_json(
    child_id: &str,
    actor_id: &str,
    registration_id: &str,
    muse: bool,
) -> String {
    let request = LaunchRequestDto {
        register: RegistrationDto {
            child_id,
            actor_id,
            registration_id,
            muse,
        },
        connection_id: "",
        cwd: "",
        args: None,
        tty: false,
        cols: 0,
        rows: 0,
    };
    to_go_json(&request)
}

pub(crate) fn compose_override_json(service: &str, volumes: Vec<String>) -> String {
    let mut services = BTreeMap::new();
    services.insert(service.to_string(), ComposeServiceDto { volumes });
    to_go_json(&ComposeOverrideDto { services })
}

#[derive(Default)]
struct LaunchExit {
    code: i64,
    error: String,
}

impl<'de> Deserialize<'de> for LaunchExit {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct ExitVisitor;

        impl<'de> Visitor<'de> for ExitVisitor {
            type Value = LaunchExit;

            fn expecting(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
                formatter.write_str("an object containing optional launch exit fields")
            }

            fn visit_map<M>(self, mut map: M) -> Result<Self::Value, M::Error>
            where
                M: MapAccess<'de>,
            {
                let mut exit = LaunchExit::default();
                while let Some(key) = map.next_key::<String>()? {
                    match key.as_str() {
                        "code" => {
                            exit.code = map.next_value::<i64>()?;
                        }
                        "error" => {
                            exit.error = map.next_value::<String>()?;
                        }
                        _ => {
                            map.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(exit)
            }
        }

        deserializer.deserialize_map(ExitVisitor)
    }
}

pub(crate) fn parse_launch_exit(body: &[u8]) -> Result<(i64, String), ()> {
    let mut deserializer = serde_json::Deserializer::from_slice(body);
    let exit = LaunchExit::deserialize(&mut deserializer).map_err(|_| ())?;
    deserializer.end().map_err(|_| ())?;
    Ok((exit.code, exit.error))
}
