use crate::json;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

use super::parse_string_i64;

// ---------- daemon identity dispatch (internal/host/identity.go) ----------

/// Trusted web-to-host launch input (`identity.TerminalStart`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TerminalStart {
    pub connection_id: String,
    pub project_id: String,
    pub actor_id: i64,
    pub login: String,
    pub scope: String,
    pub cols: i64,
    pub rows: i64,
}

impl<'de> Deserialize<'de> for TerminalStart {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct StartVisitor;
        impl<'de> Visitor<'de> for StartVisitor {
            type Value = TerminalStart;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a terminal start object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = TerminalStart::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("connection_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.connection_id = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("project_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.project_id = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("actor_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.actor_id = parse_string_i64(&v)
                                .ok_or_else(|| de::Error::custom("invalid actor_id"))?;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("login") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.login = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("scope") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.scope = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("cols") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            out.cols = v.0;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("rows") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            out.rows = v.0;
                        }
                        continue;
                    }
                    return Err(de::Error::unknown_field(
                        &key,
                        &[
                            "connection_id",
                            "project_id",
                            "actor_id",
                            "login",
                            "scope",
                            "cols",
                            "rows",
                        ],
                    ));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(StartVisitor)
    }
}

impl TerminalStart {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}
