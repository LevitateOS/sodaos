use crate::json::{self, Kind, Spec};

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

const TERMINAL_START_SPECS: &[Spec] = &[
    Spec {
        name: "connection_id",
        kind: Kind::Str,
    },
    Spec {
        name: "project_id",
        kind: Kind::Str,
    },
    Spec {
        name: "actor_id",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "scope",
        kind: Kind::Str,
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

impl TerminalStart {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        let m =
            json::bind_root(&v, "TerminalStart", TERMINAL_START_SPECS, false).map_err(|e| e.0)?;
        let actor_id = if m.contains("actor_id") {
            parse_string_i64(&m.take_string("actor_id"))
                .ok_or_else(|| "invalid actor_id".to_string())?
        } else {
            0
        };
        Ok(TerminalStart {
            connection_id: m.take_string("connection_id"),
            project_id: m.take_string("project_id"),
            actor_id,
            login: m.take_string("login"),
            scope: m.take_string("scope"),
            cols: m.take_i64("cols"),
            rows: m.take_i64("rows"),
        })
    }
}
