use base64::Engine;
use serde::de::{IgnoredAny, MapAccess, Visitor};
use serde::{Deserialize, Serialize};
use std::fmt;

use super::launch_json::serialize_go;
use super::shell::ShellRequest;

#[derive(Serialize)]
struct ShellLaunchRequest<'a> {
    #[serde(skip_serializing_if = "Option::is_none")]
    home: Option<&'a str>,
    #[serde(skip_serializing_if = "Option::is_none")]
    config_home: Option<&'a str>,
    #[serde(skip_serializing_if = "Option::is_none")]
    term: Option<&'a str>,
    connection_id: &'a str,
    cwd: &'a str,
    args: &'a [String],
    tty: bool,
    cols: u16,
    rows: u16,
}

// shell_request_json emits the exact Go LaunchRequest field order for a
// shell: empty home/config_home/term omitted, argv always an array.
pub(crate) fn shell_request_json(r: &ShellRequest) -> String {
    serialize_go(&ShellLaunchRequest {
        home: (!r.home.is_empty()).then_some(r.home.as_str()),
        config_home: (!r.config_home.is_empty()).then_some(r.config_home.as_str()),
        term: (!r.term.is_empty()).then_some(r.term.as_str()),
        connection_id: &r.connection_id,
        cwd: &r.cwd,
        args: &r.args,
        tty: r.tty,
        cols: r.cols,
        rows: r.rows,
    })
}

pub(crate) fn base64_encode(data: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD.encode(data)
}

#[derive(Default)]
struct LaunchExit {
    code: i64,
    error: String,
}

impl<'de> Deserialize<'de> for LaunchExit {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ExitVisitor;

        impl<'de> Visitor<'de> for ExitVisitor {
            type Value = LaunchExit;

            fn expecting(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
                formatter.write_str("a Muse launch-exit object")
            }

            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut exit = LaunchExit::default();
                while let Some(key) = map.next_key::<String>()? {
                    match key.as_str() {
                        "code" => exit.code = map.next_value()?,
                        "error" => exit.error = map.next_value()?,
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

// Missing fields keep Go's zero values, repeated recognized fields apply in
// input order, and unknown fields (including nested values) are ignored.
pub(crate) fn parse_launch_exit(body: &[u8]) -> Result<(i64, String), ()> {
    let text = std::str::from_utf8(body).map_err(|_| ())?;
    let mut deserializer = serde_json::Deserializer::from_str(text);
    let exit = LaunchExit::deserialize(&mut deserializer).map_err(|_| ())?;
    deserializer.end().map_err(|_| ())?;
    Ok((exit.code, exit.error))
}
