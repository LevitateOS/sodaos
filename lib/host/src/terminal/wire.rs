use crate::domain;
use crate::json::{self, BoundMap, Kind, Spec, Value};

use super::{
    contains_crlf, strict_b64_decode, terminal_dimensions, valid_terminal_id, valid_terminal_name,
    CREDENTIAL_LIMIT, TERMINAL_LIMIT,
};

// ---------- terminal request/frame wire types ----------

/// Private root:soda helper input (`TerminalRequest`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TerminalRequest {
    pub action: String,
    pub id: String,
    pub project: String,
    pub login: String,
    pub identity: i64,
    pub cols: i64,
    pub rows: i64,
    pub expires: i64,
    pub name: String,
    pub scope: String,
}

const TERMINAL_REQUEST_SPECS: &[Spec] = &[
    Spec {
        name: "action",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "identity",
        kind: Kind::I64,
    },
    Spec {
        name: "cols",
        kind: Kind::Int,
    },
    Spec {
        name: "rows",
        kind: Kind::Int,
    },
    Spec {
        name: "expires",
        kind: Kind::I64,
    },
    Spec {
        name: "name",
        kind: Kind::Str,
    },
    Spec {
        name: "scope",
        kind: Kind::Str,
    },
];

/// One terminal listing row (`TerminalState`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TerminalState {
    pub id: String,
    pub name: String,
    pub created_at: i64,
    pub ready: bool,
    pub attached: bool,
    pub state: String,
}

const TERMINAL_STATE_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "name",
        kind: Kind::Str,
    },
    Spec {
        name: "created_at",
        kind: Kind::I64,
    },
    Spec {
        name: "ready",
        kind: Kind::Bool,
    },
    Spec {
        name: "attached",
        kind: Kind::Bool,
    },
    Spec {
        name: "state",
        kind: Kind::Str,
    },
];

/// One terminal stream frame (`TerminalFrame`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TerminalFrame {
    pub frame_type: String,
    pub data: String,
    pub cols: i64,
    pub rows: i64,
    pub reason: String,
    pub terminals: Option<Vec<TerminalState>>,
}

const TERMINAL_FRAME_SPECS: &[Spec] = &[
    Spec {
        name: "type",
        kind: Kind::Str,
    },
    Spec {
        name: "data",
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
    Spec {
        name: "reason",
        kind: Kind::Str,
    },
];

impl TerminalState {
    fn from_map(m: &BoundMap) -> Self {
        TerminalState {
            id: m.take_string("id"),
            name: m.take_string("name"),
            created_at: m.take_i64("created_at"),
            ready: m.take_bool("ready"),
            attached: m.take_bool("attached"),
            state: m.take_string("state"),
        }
    }

    pub fn encode_into(&self, out: &mut String) {
        out.push_str("{\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"name\":");
        out.push_str(&json::quote(&self.name));
        out.push_str(",\"created_at\":");
        out.push_str(&self.created_at.to_string());
        out.push_str(",\"ready\":");
        out.push_str(if self.ready { "true" } else { "false" });
        out.push_str(",\"attached\":");
        out.push_str(if self.attached { "true" } else { "false" });
        out.push_str(",\"state\":");
        out.push_str(&json::quote(&self.state));
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

/// Fold-match one object field like `encoding/json`: exact match wins,
/// otherwise a unique case-insensitive match binds.
fn find_field<'a>(fields: &'a [(String, Value)], name: &str) -> Vec<&'a Value> {
    let exact: Vec<&Value> = fields
        .iter()
        .filter(|(k, _)| k == name)
        .map(|(_, v)| v)
        .collect();
    if !exact.is_empty() {
        return exact;
    }
    fields
        .iter()
        .filter(|(k, _)| k.eq_ignore_ascii_case(name))
        .map(|(_, v)| v)
        .collect()
}

fn decode_terminals(fields: &[(String, Value)]) -> Result<Option<Vec<TerminalState>>, String> {
    let hits = find_field(fields, "terminals");
    if hits.len() > 1 && hits.iter().any(|v| !v.is_null()) {
        // Several fields fold to this key: Go hides all of them, so the
        // frame carries an unknown field.
        return Err("decode request: json: unknown field \"terminals\"".to_string());
    }
    let Some(value) = hits.into_iter().find(|v| !v.is_null()) else {
        return Ok(None);
    };
    let Value::Array(items) = value else {
        return Err("decode request: terminals must be an array".to_string());
    };
    let mut out = Vec::with_capacity(items.len());
    for item in items {
        if item.is_null() {
            out.push(TerminalState::default());
            continue;
        }
        let Value::Object(_) = item else {
            return Err("decode request: terminals must be objects".to_string());
        };
        let m = json::bind_struct(
            item,
            "TerminalState",
            &["terminals".to_string()],
            TERMINAL_STATE_SPECS,
            false,
        )
        .map_err(|e| e.0)?;
        out.push(TerminalState::from_map(&m));
    }
    Ok(Some(out))
}

impl TerminalFrame {
    /// Strict decode of one frame object (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        let Value::Object(fields) = &v else {
            return Err("decode request: expected object".to_string());
        };
        let terminals = decode_terminals(fields)?;
        // Bind the scalar fields with the shared strict binder, hiding the
        // already-consumed `terminals` member.
        let rest: Vec<(String, Value)> = fields
            .iter()
            .filter(|(k, _)| k != "terminals" && !k.eq_ignore_ascii_case("terminals"))
            .cloned()
            .collect();
        // If several keys folded to `terminals`, the frame is invalid; that
        // was already rejected above when a non-null value was present. A
        // folded all-null pair is still an unknown field in Go.
        let folded = fields
            .iter()
            .filter(|(k, _)| k.eq_ignore_ascii_case("terminals"))
            .count();
        if folded > 1 {
            return Err("decode request: json: unknown field \"terminals\"".to_string());
        }
        let m = json::bind_root(
            &Value::Object(rest),
            "TerminalFrame",
            TERMINAL_FRAME_SPECS,
            false,
        )
        .map_err(|e| e.0)?;
        Ok(TerminalFrame {
            frame_type: m.take_string("type"),
            data: m.take_string("data"),
            cols: m.take_i64("cols"),
            rows: m.take_i64("rows"),
            reason: m.take_string("reason"),
            terminals,
        })
    }

    /// `encoding/json` field order with `omitempty`, no trailing newline.
    pub fn encode_into(&self, out: &mut String) {
        out.push_str("{\"type\":");
        out.push_str(&json::quote(&self.frame_type));
        if !self.data.is_empty() {
            out.push_str(",\"data\":");
            out.push_str(&json::quote(&self.data));
        }
        if self.cols != 0 {
            out.push_str(",\"cols\":");
            out.push_str(&self.cols.to_string());
        }
        if self.rows != 0 {
            out.push_str(",\"rows\":");
            out.push_str(&self.rows.to_string());
        }
        if !self.reason.is_empty() {
            out.push_str(",\"reason\":");
            out.push_str(&json::quote(&self.reason));
        }
        if let Some(list) = &self.terminals {
            out.push_str(",\"terminals\":[");
            for (i, item) in list.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                item.encode_into(out);
            }
            out.push(']');
        }
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

impl TerminalRequest {
    /// Strict decode of one request object (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        let m = json::bind_root(&v, "TerminalRequest", TERMINAL_REQUEST_SPECS, false)
            .map_err(|e| e.0)?;
        Ok(TerminalRequest {
            action: m.take_string("action"),
            id: m.take_string("id"),
            project: m.take_string("project"),
            login: m.take_string("login"),
            identity: m.take_i64("identity"),
            cols: m.take_i64("cols"),
            rows: m.take_i64("rows"),
            expires: m.take_i64("expires"),
            name: m.take_string("name"),
            scope: m.take_string("scope"),
        })
    }

    pub fn encode(&self) -> String {
        let mut out = String::from("{\"action\":");
        out.push_str(&json::quote(&self.action));
        out.push_str(",\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"login\":");
        out.push_str(&json::quote(&self.login));
        out.push_str(",\"identity\":");
        out.push_str(&self.identity.to_string());
        out.push_str(",\"cols\":");
        out.push_str(&self.cols.to_string());
        out.push_str(",\"rows\":");
        out.push_str(&self.rows.to_string());
        out.push_str(",\"expires\":");
        out.push_str(&self.expires.to_string());
        out.push_str(",\"name\":");
        out.push_str(&json::quote(&self.name));
        out.push_str(",\"scope\":");
        out.push_str(&json::quote(&self.scope));
        out.push('}');
        out
    }
}

// ---------- admission predicates (types.go) ----------

fn valid_terminal_actor(input: &TerminalRequest) -> bool {
    domain::valid_id(&input.project)
        && domain::valid_login(&input.login)
        && input.login != "root"
        && input.identity > 0
        && valid_terminal_name(&input.name)
}

fn valid_terminal_window(input: &TerminalRequest, now: i64) -> bool {
    input.expires > now && input.expires <= now + 12 * 3600
}

fn valid_terminal_scope(action: &str, scope: &str) -> bool {
    let creating = action == "reserve" || action == "create";
    if creating {
        return scope.len() == 64 && domain::valid_container_id(scope);
    }
    scope.is_empty()
}

fn valid_list_request(input: &TerminalRequest) -> bool {
    input.id.is_empty() && input.cols == 0 && input.rows == 0 && input.name.is_empty()
}

fn valid_sized_terminal_action(input: &TerminalRequest) -> bool {
    terminal_dimensions(input.cols, input.rows)
        && (input.action != "attach" || input.name.is_empty())
}

fn valid_idle_terminal_action(input: &TerminalRequest) -> bool {
    input.cols == 0 && input.rows == 0 && (input.action == "rename" || input.name.is_empty())
}

fn valid_terminal_action(input: &TerminalRequest) -> bool {
    if input.action == "list" {
        return valid_list_request(input);
    }
    if !valid_terminal_id(&input.id) {
        return false;
    }
    match input.action.as_str() {
        "reserve" | "create" | "attach" => valid_sized_terminal_action(input),
        "inspect" | "end" | "rename" => valid_idle_terminal_action(input),
        _ => false,
    }
}

impl TerminalRequest {
    /// `TerminalRequest.Valid(now)`.
    pub fn valid(&self, now: i64) -> bool {
        valid_terminal_actor(self)
            && valid_terminal_window(self, now)
            && valid_terminal_scope(&self.action, &self.scope)
            && valid_terminal_action(self)
    }
}

fn valid_typed_input(f: &TerminalFrame) -> bool {
    let Some(data) = strict_b64_decode(&f.data) else {
        return false;
    };
    !contains_crlf(&f.data) && !data.is_empty() && data.len() <= 16384 && f.cols == 0 && f.rows == 0
}

fn valid_resize_input(f: &TerminalFrame) -> bool {
    f.data.is_empty() && terminal_dimensions(f.cols, f.rows)
}

fn valid_idle_input(f: &TerminalFrame) -> bool {
    f.data.is_empty() && f.cols == 0 && f.rows == 0
}

impl TerminalFrame {
    /// `TerminalFrame.InputValid()`.
    pub fn input_valid(&self) -> bool {
        if !self.reason.is_empty() || self.terminals.is_some() {
            return false;
        }
        match self.frame_type.as_str() {
            "input" => valid_typed_input(self),
            "resize" => valid_resize_input(self),
            "heartbeat" | "close" => valid_idle_input(self),
            _ => false,
        }
    }

    /// `TerminalFrame.OutputValid()`.
    pub fn output_valid(&self) -> bool {
        if self.cols != 0 || self.rows != 0 {
            return false;
        }
        if self.frame_type != "metadata" && self.terminals.is_some() {
            return false;
        }
        match self.frame_type.as_str() {
            "metadata" => valid_metadata_output(self),
            "ready" => self.data.is_empty() && self.reason.is_empty(),
            "output" => valid_output_data(self),
            "closed" => valid_closed_output(self),
            _ => false,
        }
    }
}

fn valid_terminal_state_value(state: &str) -> bool {
    matches!(state, "ready" | "opening" | "ending" | "ended")
}

fn valid_terminal_item_flags(ready: bool, attached: bool, state: &str) -> bool {
    if ready != (state == "ready") {
        return false;
    }
    !attached || ready
}

fn valid_terminal_item(item: &TerminalState, seen: &mut std::collections::HashSet<String>) -> bool {
    if !valid_terminal_id(&item.id) || seen.contains(&item.id) || !valid_terminal_name(&item.name) {
        return false;
    }
    if item.created_at <= 0 || item.created_at > 9007199254740991 {
        return false;
    }
    seen.insert(item.id.clone());
    valid_terminal_item_flags(item.ready, item.attached, &item.state)
        && valid_terminal_state_value(&item.state)
}

fn valid_metadata_output(f: &TerminalFrame) -> bool {
    let Some(list) = &f.terminals else {
        return false;
    };
    if !f.data.is_empty() || !f.reason.is_empty() || list.len() > TERMINAL_LIMIT {
        return false;
    }
    let mut seen = std::collections::HashSet::with_capacity(list.len());
    for item in list {
        if !valid_terminal_item(item, &mut seen) {
            return false;
        }
    }
    true
}

fn valid_output_data(f: &TerminalFrame) -> bool {
    if !f.reason.is_empty() {
        return false;
    }
    // No CRLF rejection here: Go `Strict()` skips newlines, and unlike the
    // input path the output path accepts them.
    match strict_b64_decode(&f.data) {
        Some(data) => !data.is_empty() && data.len() <= 4096,
        None => false,
    }
}

fn valid_closed_reason(reason: &str) -> bool {
    matches!(
        reason,
        "disconnected"
            | "expired"
            | "exited"
            | "launch_failed"
            | "stream_failed"
            | "cleanup_unconfirmed"
    )
}

fn valid_closed_output(f: &TerminalFrame) -> bool {
    f.data.is_empty() && valid_closed_reason(&f.reason)
}

// ---------- JSON validity (`json.Valid`) ----------

/// Port of Go `json.Valid`: exactly one JSON value with only surrounding
/// whitespace.
pub fn json_valid(body: &[u8]) -> bool {
    json::decode_tolerant(body).is_ok()
}

/// Port of Go `identity.CredentialValid`.
pub fn credential_valid(data: &[u8]) -> bool {
    !data.is_empty() && data.len() <= CREDENTIAL_LIMIT && json_valid(data)
}
