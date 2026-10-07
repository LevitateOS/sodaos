use crate::domain;
use crate::json::{self, SignedInteger};
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};

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

#[derive(Default)]
struct TerminalRequestWire(TerminalRequest);

impl<'de> Deserialize<'de> for TerminalRequestWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = TerminalRequestWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("terminal request")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = TerminalRequest::default();
                while let Some(k) = map.next_key::<String>()? {
                    match k.to_ascii_lowercase().as_str() {
                        "action" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.action = v;
                            }
                        }
                        "id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.id = v;
                            }
                        }
                        "project" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.project = v;
                            }
                        }
                        "login" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.login = v;
                            }
                        }
                        "identity" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                out.identity = v.0;
                            }
                        }
                        "cols" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                out.cols = v.0;
                            }
                        }
                        "rows" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                out.rows = v.0;
                            }
                        }
                        "expires" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                out.expires = v.0;
                            }
                        }
                        "name" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.name = v;
                            }
                        }
                        "scope" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.scope = v;
                            }
                        }
                        _ => {
                            return Err(de::Error::unknown_field(
                                &k,
                                &[
                                    "action", "id", "project", "login", "identity", "cols", "rows",
                                    "expires", "name", "scope",
                                ],
                            ))
                        }
                    }
                }
                Ok(TerminalRequestWire(out))
            }
        }
        deserializer.deserialize_map(V)
    }
}

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

#[derive(Default)]
struct TerminalStateWire(TerminalState);

impl<'de> Deserialize<'de> for TerminalStateWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = TerminalStateWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("terminal state")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = TerminalState::default();
                while let Some(k) = map.next_key::<String>()? {
                    match k.to_ascii_lowercase().as_str() {
                        "id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.id = v;
                            }
                        }
                        "name" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.name = v;
                            }
                        }
                        "created_at" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                out.created_at = v.0;
                            }
                        }
                        "ready" => {
                            if let Some(v) = map.next_value::<Option<bool>>()? {
                                out.ready = v;
                            }
                        }
                        "attached" => {
                            if let Some(v) = map.next_value::<Option<bool>>()? {
                                out.attached = v;
                            }
                        }
                        "state" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.state = v;
                            }
                        }
                        _ => {
                            return Err(de::Error::unknown_field(
                                &k,
                                &["id", "name", "created_at", "ready", "attached", "state"],
                            ))
                        }
                    }
                }
                Ok(TerminalStateWire(out))
            }
        }
        deserializer.deserialize_map(V)
    }
}

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

#[derive(Default)]
struct TerminalFrameWire(TerminalFrame);

impl<'de> Deserialize<'de> for TerminalFrameWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = TerminalFrameWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("terminal frame")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = TerminalFrame::default();
                let mut terminal_count = 0usize;
                let mut terminal_raw: Option<Box<serde_json::value::RawValue>> = None;
                while let Some(k) = map.next_key::<String>()? {
                    if k.eq_ignore_ascii_case("terminals") {
                        terminal_count += 1;
                        let raw = map.next_value::<Box<serde_json::value::RawValue>>()?;
                        if terminal_count == 1 {
                            terminal_raw = Some(raw);
                        }
                        continue;
                    }
                    match k.to_ascii_lowercase().as_str() {
                        "type" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.frame_type = v;
                            }
                        }
                        "data" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.data = v;
                            }
                        }
                        "cols" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                out.cols = v.0;
                            }
                        }
                        "rows" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                out.rows = v.0;
                            }
                        }
                        "reason" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                out.reason = v;
                            }
                        }
                        _ => {
                            return Err(de::Error::unknown_field(
                                &k,
                                &["type", "data", "cols", "rows", "reason", "terminals"],
                            ))
                        }
                    }
                }
                if terminal_count > 1 {
                    return Err(de::Error::custom("folded duplicate terminals field"));
                }
                if let Some(raw) = terminal_raw {
                    if raw.get() != "null" {
                        let states: Vec<Option<TerminalStateWire>> =
                            serde_json::from_str(raw.get()).map_err(de::Error::custom)?;
                        out.terminals = Some(
                            states
                                .into_iter()
                                .map(|v| v.map(|s| s.0).unwrap_or_default())
                                .collect(),
                        );
                    }
                }
                Ok(TerminalFrameWire(out))
            }
        }
        deserializer.deserialize_map(V)
    }
}

impl TerminalState {
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

impl TerminalFrame {
    /// Strict decode of one frame object (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as::<TerminalFrameWire>(body)
            .map(|wire| wire.0)
            .map_err(|error| error.0)
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
        json::decode_strict_as::<TerminalRequestWire>(body)
            .map(|wire| wire.0)
            .map_err(|error| error.0)
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
    let mut input = serde_json::Deserializer::from_slice(body);
    serde::de::IgnoredAny::deserialize(&mut input).is_ok() && input.end().is_ok()
}

/// Port of Go `identity.CredentialValid`.
pub fn credential_valid(data: &[u8]) -> bool {
    !data.is_empty() && data.len() <= CREDENTIAL_LIMIT && json_valid(data)
}
