use crate::json::{self, Kind, Spec, Value};
use crate::terminal;

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

const NESTED_REGISTRATION_SPECS: &[Spec] = &[
    Spec {
        name: "child_id",
        kind: Kind::Str,
    },
    Spec {
        name: "actor_id",
        kind: Kind::Str,
    },
    Spec {
        name: "registration_id",
        kind: Kind::Str,
    },
    Spec {
        name: "muse",
        kind: Kind::Bool,
    },
];

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

const LAUNCH_REQUEST_SPECS: &[Spec] = &[
    Spec {
        name: "home",
        kind: Kind::Str,
    },
    Spec {
        name: "register",
        kind: Kind::OptObject {
            go_type: "*identity.NestedRegistration",
            struct_name: "NestedRegistration",
            specs: NESTED_REGISTRATION_SPECS,
        },
    },
    Spec {
        name: "config_home",
        kind: Kind::Str,
    },
    Spec {
        name: "term",
        kind: Kind::Str,
    },
    Spec {
        name: "connection_id",
        kind: Kind::Str,
    },
    Spec {
        name: "cwd",
        kind: Kind::Str,
    },
    Spec {
        name: "args",
        kind: Kind::StrList,
    },
    Spec {
        name: "tty",
        kind: Kind::Bool,
    },
    Spec {
        name: "cols",
        kind: Kind::Int,
    },
    Spec {
        name: "rows",
        kind: Kind::Int,
    },
];

/// Bounded live shell control (`identity.LaunchControl`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LaunchControl {
    pub signal: i64,
    pub cols: u16,
    pub rows: u16,
}

const LAUNCH_CONTROL_SPECS: &[Spec] = &[
    Spec {
        name: "signal",
        kind: Kind::Int,
    },
    Spec {
        name: "cols",
        kind: Kind::Int,
    },
    Spec {
        name: "rows",
        kind: Kind::Int,
    },
];

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
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        Self::decode_value(&v)
    }

    pub fn decode_value(v: &Value) -> Result<Self, String> {
        let m =
            json::bind_root(v, "LaunchRequest", LAUNCH_REQUEST_SPECS, false).map_err(|e| e.0)?;
        let register = match m.take_opt_map("register") {
            None => None,
            Some(nested) => {
                let actor_id = if nested.contains("actor_id") {
                    terminal::parse_string_i64(&nested.take_string("actor_id"))
                        .ok_or_else(|| "invalid actor_id".to_string())?
                } else {
                    0
                };
                Some(NestedRegistration {
                    child_id: nested.take_string("child_id"),
                    actor_id,
                    registration_id: nested.take_string("registration_id"),
                    muse: nested.take_bool("muse"),
                })
            }
        };
        let cols = m.take_i64("cols");
        let rows = m.take_i64("rows");
        if !(0..=u16::MAX as i64).contains(&cols) || !(0..=u16::MAX as i64).contains(&rows) {
            return Err("decode request: cols/rows out of range".to_string());
        }
        Ok(LaunchRequest {
            home: m.take_string("home"),
            register,
            config_home: m.take_string("config_home"),
            term: m.take_string("term"),
            connection_id: m.take_string("connection_id"),
            cwd: m.take_string("cwd"),
            args: m.take_str_list("args"),
            tty: m.take_bool("tty"),
            cols: cols as u16,
            rows: rows as u16,
        })
    }
}

impl LaunchControl {
    /// Strict decode of one control message (`DisallowUnknownFields`).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        let m =
            json::bind_root(&v, "LaunchControl", LAUNCH_CONTROL_SPECS, false).map_err(|e| e.0)?;
        let cols = m.take_i64("cols");
        let rows = m.take_i64("rows");
        if !(0..=u16::MAX as i64).contains(&cols) || !(0..=u16::MAX as i64).contains(&rows) {
            return Err("decode request: cols/rows out of range".to_string());
        }
        Ok(LaunchControl {
            signal: m.take_i64("signal"),
            cols: cols as u16,
            rows: rows as u16,
        })
    }
}
