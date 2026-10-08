use serde::de::{IgnoredAny, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};
use std::fmt;

#[derive(Serialize)]
pub(crate) struct NestedRegistration {
    pub(crate) child_id: String,
    pub(crate) actor_id: String,
    pub(crate) registration_id: String,
    pub(crate) muse: bool,
}

#[derive(Serialize)]
struct LaunchRequestDto<'a> {
    register: &'a NestedRegistration,
    connection_id: &'static str,
    cwd: &'static str,
    args: Option<&'a [String]>,
    tty: bool,
    cols: u32,
    rows: u32,
}

pub(crate) fn launch_request_json(request: &NestedRegistration) -> String {
    let request = LaunchRequestDto {
        register: request,
        connection_id: "",
        cwd: "",
        args: None,
        tty: false,
        cols: 0,
        rows: 0,
    };
    serde_json::to_string(&request).expect("serializing a launch request cannot fail")
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
