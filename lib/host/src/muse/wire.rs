use crate::json;
use crate::terminal;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

// ---------- launch wire types (internal/identity/launch.go) ----------

/// Kernel launch-socket path.
pub const MUSE_LAUNCH_SOCKET: &str = "/run/soda-muse-interface/launch.sock";
/// Setting named when several Muse connections need an explicit choice.
pub const MUSE_CONNECTION_SETTING: &str = "SODA_MUSE_CONNECTION";

/// Nested-container registration (`identity.NestedRegistration`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NestedRegistration {
    pub child_id: String,
    pub actor_id: i64,
    pub registration_id: String,
    pub muse: bool,
}

impl<'de> Deserialize<'de> for NestedRegistration {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct RegistrationVisitor;
        impl<'de> Visitor<'de> for RegistrationVisitor {
            type Value = NestedRegistration;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a nested registration object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = NestedRegistration::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("child_id") {
                        if let Some(value) = map.next_value::<Option<String>>()? {
                            out.child_id = value;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("actor_id") {
                        if let Some(value) = map.next_value::<Option<String>>()? {
                            out.actor_id = terminal::parse_string_i64(&value)
                                .ok_or_else(|| de::Error::custom("invalid actor_id"))?;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("registration_id") {
                        if let Some(value) = map.next_value::<Option<String>>()? {
                            out.registration_id = value;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("muse") {
                        if let Some(value) = map.next_value::<Option<bool>>()? {
                            out.muse = value;
                        }
                        continue;
                    }
                    return Err(de::Error::unknown_field(
                        &key,
                        &["child_id", "actor_id", "registration_id", "muse"],
                    ));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(RegistrationVisitor)
    }
}

/// Invocation preferences, never caller authority (`identity.LaunchRequest`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LaunchRequest {
    pub home: String,
    pub register: Option<NestedRegistration>,
    pub config_home: String,
    pub term: String,
    pub connection_id: String,
    pub cwd: String,
    pub args: Vec<String>,
    pub tty: bool,
    pub cols: u16,
    pub rows: u16,
}

impl<'de> Deserialize<'de> for LaunchRequest {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct RequestVisitor;
        impl<'de> Visitor<'de> for RequestVisitor {
            type Value = LaunchRequest;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a launch request object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = LaunchRequest::default();
                let mut cols = 0i64;
                let mut rows = 0i64;
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("home") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.home = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("register") {
                        if let Some(v) = map.next_value::<Option<NestedRegistration>>()? {
                            out.register = Some(v);
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("config_home") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.config_home = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("term") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.term = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("connection_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.connection_id = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("cwd") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.cwd = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("args") {
                        if let Some(v) = map.next_value::<Option<Vec<Option<String>>>>()? {
                            out.args = v.into_iter().map(Option::unwrap_or_default).collect();
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("tty") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.tty = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("cols") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            cols = v.0;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("rows") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            rows = v.0;
                        }
                        continue;
                    }
                    return Err(de::Error::unknown_field(
                        &key,
                        &[
                            "home",
                            "register",
                            "config_home",
                            "term",
                            "connection_id",
                            "cwd",
                            "args",
                            "tty",
                            "cols",
                            "rows",
                        ],
                    ));
                }
                if !(0..=u16::MAX as i64).contains(&cols) || !(0..=u16::MAX as i64).contains(&rows)
                {
                    return Err(de::Error::custom("cols/rows out of range"));
                }
                out.cols = cols as u16;
                out.rows = rows as u16;
                Ok(out)
            }
        }
        deserializer.deserialize_map(RequestVisitor)
    }
}

/// Bounded live shell control (`identity.LaunchControl`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LaunchControl {
    pub signal: i64,
    pub cols: u16,
    pub rows: u16,
}

impl<'de> Deserialize<'de> for LaunchControl {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ControlVisitor;
        impl<'de> Visitor<'de> for ControlVisitor {
            type Value = LaunchControl;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a launch control object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = LaunchControl::default();
                let mut cols = 0i64;
                let mut rows = 0i64;
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("signal") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            out.signal = v.0;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("cols") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            cols = v.0;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("rows") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            rows = v.0;
                        }
                        continue;
                    }
                    return Err(de::Error::unknown_field(&key, &["signal", "cols", "rows"]));
                }
                if !(0..=u16::MAX as i64).contains(&cols) || !(0..=u16::MAX as i64).contains(&rows)
                {
                    return Err(de::Error::custom("cols/rows out of range"));
                }
                out.cols = cols as u16;
                out.rows = rows as u16;
                Ok(out)
            }
        }
        deserializer.deserialize_map(ControlVisitor)
    }
}

/// Shell outcome (`identity.LaunchExit`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LaunchExit {
    pub code: i32,
    pub error: String,
}

impl LaunchExit {
    pub fn denied() -> Self {
        LaunchExit {
            code: 1,
            error: "Muse launch denied".to_string(),
        }
    }

    pub fn cleanup_unconfirmed() -> Self {
        LaunchExit {
            code: 1,
            error: "Muse cleanup unconfirmed".to_string(),
        }
    }

    pub fn encode(&self) -> String {
        let mut out = format!("{{\"code\":{}", self.code);
        if !self.error.is_empty() {
            out.push_str(",\"error\":");
            out.push_str(&json::quote(&self.error));
        }
        out.push('}');
        out
    }

    /// Socket framing: `json.Encoder` appends one newline.
    pub fn encode_line(&self) -> String {
        format!("{}\n", self.encode())
    }
}

fn launch_text(value: &str, limit: usize) -> bool {
    value.len() <= limit && !value.contains('\0')
}

fn launch_absolute_path(value: &str, optional: bool) -> bool {
    if optional && value.is_empty() {
        return true;
    }
    value.starts_with('/') && launch_text(value, 4096)
}

fn launch_arguments_valid(args: &[String]) -> bool {
    let mut size = 0usize;
    for arg in args {
        size += arg.len();
        if !launch_text(arg, 32768) {
            return false;
        }
    }
    size <= 32768
}

impl LaunchRequest {
    /// `LaunchRequest.Validate()`.
    pub fn validate(&self) -> Result<(), String> {
        if self.register.is_some() {
            return self.registration_valid();
        }
        if !launch_absolute_path(&self.cwd, false)
            || !launch_absolute_path(&self.config_home, true)
            || !launch_absolute_path(&self.home, true)
        {
            return Err(terminal::err_denied());
        }
        if !launch_text(&self.term, 128) || self.connection_id.len() > 128 || self.args.len() > 256
        {
            return Err(terminal::err_denied());
        }
        if self.tty && (self.cols == 0 || self.rows == 0) {
            return Err(terminal::err_denied());
        }
        if !launch_arguments_valid(&self.args) {
            return Err(terminal::err_denied());
        }
        Ok(())
    }

    fn registration_valid(&self) -> Result<(), String> {
        let Some(register) = &self.register else {
            return Err(terminal::err_denied());
        };
        if !self.cwd.is_empty()
            || !self.args.is_empty()
            || !self.connection_id.is_empty()
            || self.tty
            || register.actor_id <= 0
            || !register.muse
        {
            return Err(terminal::err_denied());
        }
        Ok(())
    }

    /// Strict decode of one launch request.
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

impl LaunchControl {
    /// Strict decode of one control message (`DisallowUnknownFields`).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}
