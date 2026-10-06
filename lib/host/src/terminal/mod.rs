//! Host terminal executor: service, native attach, identity broker calls.
//!
//! Port of `internal/host/terminal/service.go` (execution core), `native.go`,
//! `identity.go`, `identity_transfer.go`, the agent-hash helper from
//! `agent.go`, the admission predicates from `types.go`, and the daemon
//! identity dispatch logic from `internal/host/identity.go`.
//!
//! Out of scope (stay in Go): the websocket transport (`Write`, pumps,
//! `Handler`), `AgentExec` command construction for the Go client, and the
//! HTTP route shells (plain `pub` fns are exposed instead).
//!
//! Behavioral notes:
//!
//! * Podman argv, JSON wire bytes, the terminal frame protocol, timeouts and
//!   error strings match the Go implementation. Exit-code checks (`container
//!   exists`, `mountpoint`) parse the `exit status N` text the crate
//!   [`Executor`](crate::project::Executor) surface produces, since the
//!   trait is stringly typed.
//! * `stream_identity_harness` buffers the host tar producer through the
//!   executor instead of streaming pipe-to-pipe; error strings are unchanged.

use std::collections::HashMap;
use std::fs::File;
use std::io::{BufRead, BufReader, Write};
use std::os::unix::fs::{MetadataExt, PermissionsExt};
use std::os::unix::io::AsRawFd;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use crate::account::AGENT_PROGRAM;
use crate::domain;
use crate::json::{self, BoundMap, Kind, Spec, Value};
use crate::project::Executor;
use crate::sha256;

pub mod factory;

// ---------- wire limits ----------

/// `FrameLimit`: bounded 64-row metadata; IO payload bounds stay smaller.
pub const FRAME_LIMIT: usize = 131072;
/// `terminalLimit`: at most 64 terminals per metadata frame.
pub const TERMINAL_LIMIT: usize = 64;
/// Live-stream admission cap: `2*terminalLimit`.
pub const STREAM_LIMIT: usize = 128;
/// Agent stdin delivery cap for broker calls (`384<<10`).
pub const BROKER_RESPONSE_LIMIT: usize = 384 * 1024;
/// Credential byte cap (`256<<10`).
pub const CREDENTIAL_LIMIT: usize = 256 * 1024;

// ---------- identity errors ----------

/// `identity.ErrDenied`.
pub const ERR_DENIED: &str = "identity authority denied";
/// `identity.ErrStale`.
pub const ERR_STALE: &str = "identity generation changed";
/// `identity.ErrUncertain`.
pub const ERR_UNCERTAIN: &str = "subscription requires reconnection";
/// `identity.ErrNotFound`.
pub const ERR_NOT_FOUND: &str = "identity execution missing";

/// Daemon identity-route errors from `internal/host/identity.go`.
pub const ERR_INVALID_IDENTITY_OPERATION: &str = "invalid identity operation";
pub const ERR_IDENTITY_UNCONFIRMED: &str = "identity operation unconfirmed";
/// `POST /identity/launch` path.
pub const IDENTITY_LAUNCH_PATH: &str = "/identity/launch";
/// Provider/kind discriminators from `internal/identity/types.go`.
pub const PROVIDER_CODEX: &str = "codex";
pub const PROVIDER_MUSE: &str = "muse";
pub const KIND_FACTORY: &str = "factory";
pub const KIND_TERMINAL: &str = "terminal";
/// Muse project-execution scope carried in broker bindings.
pub const SCOPE_MUSE_PROJECT: &str = "muse-project";

pub fn err_denied() -> String {
    ERR_DENIED.to_string()
}
pub fn err_stale() -> String {
    ERR_STALE.to_string()
}
pub fn err_uncertain() -> String {
    ERR_UNCERTAIN.to_string()
}

// ---------- small predicates ----------

/// `terminalID = ^[0-9a-f]{32}$`.
pub fn valid_terminal_id(id: &str) -> bool {
    id.len() == 32 && domain::is_hex_lower(id)
}

/// `now` as Unix seconds, saturating on clock failure.
pub fn now_unix() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}

// Exact `unicode.Cc` / `unicode.Cf` ranges from the pinned Go toolchain
// (go1.26.7), mirroring `domain.rs` for `ValidTerminalName`.
const GO_CC: &[(u32, u32)] = &[(0x0, 0x1F), (0x7F, 0x9F)];
const GO_CF: &[(u32, u32)] = &[
    (0xAD, 0xAD),
    (0x600, 0x605),
    (0x61C, 0x61C),
    (0x6DD, 0x6DD),
    (0x70F, 0x70F),
    (0x890, 0x891),
    (0x8E2, 0x8E2),
    (0x180E, 0x180E),
    (0x200B, 0x200F),
    (0x202A, 0x202E),
    (0x2060, 0x2064),
    (0x2066, 0x206F),
    (0xFEFF, 0xFEFF),
    (0xFFF9, 0xFFFB),
    (0x110BD, 0x110BD),
    (0x110CD, 0x110CD),
    (0x13430, 0x1343F),
    (0x1BCA0, 0x1BCA3),
    (0x1D173, 0x1D17A),
    (0xE0001, 0xE0001),
    (0xE0020, 0xE007F),
];

fn in_ranges(table: &[(u32, u32)], c: char) -> bool {
    let v = c as u32;
    table.iter().any(|&(lo, hi)| v >= lo && v <= hi)
}

/// Port of Go `ValidTerminalName`: at most 80 runes, no Cc/Cf characters.
/// (`&str` is always valid UTF-8, so the `utf8.ValidString` check is free.)
pub fn valid_terminal_name(name: &str) -> bool {
    if name.chars().count() > 80 {
        return false;
    }
    for c in name.chars() {
        if in_ranges(GO_CC, c) || in_ranges(GO_CF, c) {
            return false;
        }
    }
    true
}

/// `terminalDimensions`: cols 2..=500, rows 2..=300.
pub fn terminal_dimensions(cols: i64, rows: i64) -> bool {
    (2..=500).contains(&cols) && (2..=300).contains(&rows)
}

// ---------- strict base64 (Go `StdEncoding.Strict` semantics) ----------

fn b64_value(c: u8) -> Option<u8> {
    match c {
        b'A'..=b'Z' => Some(c - b'A'),
        b'a'..=b'z' => Some(c - b'a' + 26),
        b'0'..=b'9' => Some(c - b'0' + 52),
        b'+' => Some(62),
        b'/' => Some(63),
        _ => None,
    }
}

/// One `decodeQuantum` step of Go's `base64.StdEncoding` with `Strict()`.
/// Newlines are skipped (Go `Strict` still skips `\r\n`; the input path
/// rejects them separately via [`contains_crlf`]); trailing padding bits
/// must be zero.
fn strict_quantum(src: &[u8], mut si: usize, out: &mut Vec<u8>) -> Option<usize> {
    let mut dbuf = [0u8; 4];
    let mut dlen = 4usize;
    let mut j = 0usize;
    while j < 4 {
        if si == src.len() {
            if j == 0 {
                return Some(si);
            }
            return None;
        }
        let b = src[si];
        si += 1;
        if let Some(v) = b64_value(b) {
            dbuf[j] = v;
            j += 1;
            continue;
        }
        if b == b'\n' || b == b'\r' {
            continue;
        }
        if b != b'=' {
            return None;
        }
        match j {
            0 | 1 => return None,
            2 => {
                while si < src.len() && (src[si] == b'\n' || src[si] == b'\r') {
                    si += 1;
                }
                if si == src.len() {
                    return None;
                }
                if src[si] != b'=' {
                    return None;
                }
                si += 1;
            }
            _ => {}
        }
        while si < src.len() && (src[si] == b'\n' || src[si] == b'\r') {
            si += 1;
        }
        if si < src.len() {
            return None;
        }
        dlen = j;
        break;
    }
    // Go `Strict` trailing-bit checks: unused low bits must be zero.
    if dlen == 2 && dbuf[1] & 15 != 0 {
        return None;
    }
    if dlen == 3 && dbuf[2] & 3 != 0 {
        return None;
    }
    let val = (u32::from(dbuf[0]) << 18)
        | (u32::from(dbuf[1]) << 12)
        | (u32::from(dbuf[2]) << 6)
        | u32::from(dbuf[3]);
    out.push((val >> 16) as u8);
    if dlen >= 3 {
        out.push((val >> 8) as u8);
    }
    if dlen >= 4 {
        out.push(val as u8);
    }
    Some(si)
}

/// Go `base64.StdEncoding.Strict().DecodeString`, byte for byte.
pub fn strict_b64_decode(src: &str) -> Option<Vec<u8>> {
    let bytes = src.as_bytes();
    let mut out = Vec::with_capacity(bytes.len() / 4 * 3);
    let mut si = 0;
    while si < bytes.len() {
        si = strict_quantum(bytes, si, &mut out)?;
    }
    Some(out)
}

fn contains_crlf(s: &str) -> bool {
    s.bytes().any(|b| b == b'\r' || b == b'\n')
}

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

// ---------- RFC 3339 deadlines (`time.Time` wire form) ----------

fn days_from_civil(year: i64, month: i64, day: i64) -> i64 {
    let y = if month <= 2 { year - 1 } else { year };
    let era = if y >= 0 { y } else { y - 399 } / 400;
    let yoe = y - era * 400;
    let mp = (month + 9) % 12;
    let doy = (153 * mp + 2) / 5 + day - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146097 + doe - 719468
}

fn days_in_month(year: i64, month: i64) -> i64 {
    match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 if year % 4 == 0 && (year % 100 != 0 || year % 400 == 0) => 29,
        2 => 28,
        _ => 0,
    }
}

fn parse_two(digits: &[u8]) -> Option<i64> {
    if digits.len() == 2 && digits.iter().all(|b| b.is_ascii_digit()) {
        Some(((digits[0] - b'0') * 10 + (digits[1] - b'0')) as i64)
    } else {
        None
    }
}

/// Strict RFC 3339 subset matching Go `time.Time` JSON decoding: uppercase
/// `T`, `Z` or numeric offset, optional fractional seconds. Returns Unix
/// seconds and nanoseconds.
pub fn parse_rfc3339(s: &str) -> Option<(i64, u32)> {
    let b = s.as_bytes();
    if b.len() < 20
        || b[4] != b'-'
        || b[7] != b'-'
        || b[10] != b'T'
        || b[13] != b':'
        || b[16] != b':'
    {
        return None;
    }
    if !b[0..4].iter().all(|c| c.is_ascii_digit())
        || !b[5..7].iter().all(|c| c.is_ascii_digit())
        || !b[8..10].iter().all(|c| c.is_ascii_digit())
        || !b[11..13].iter().all(|c| c.is_ascii_digit())
        || !b[14..16].iter().all(|c| c.is_ascii_digit())
        || !b[17..19].iter().all(|c| c.is_ascii_digit())
    {
        return None;
    }
    let year: i64 = s[0..4].parse().ok()?;
    let month = parse_two(&b[5..7])?;
    let day = parse_two(&b[8..10])?;
    let hour = parse_two(&b[11..13])?;
    let minute = parse_two(&b[14..16])?;
    let second = parse_two(&b[17..19])?;
    if !(1..=12).contains(&month) || day < 1 || day > days_in_month(year, month) {
        return None;
    }
    if hour > 23 || minute > 59 || second > 59 {
        return None;
    }
    let mut rest = &b[19..];
    let mut nanos: u32 = 0;
    if rest.first() == Some(&b'.') {
        rest = &rest[1..];
        let mut count = 0usize;
        let mut value: u32 = 0;
        while count < 9 && !rest.is_empty() && rest[0].is_ascii_digit() {
            value = value * 10 + u32::from(rest[0] - b'0');
            rest = &rest[1..];
            count += 1;
        }
        if count == 0 {
            return None;
        }
        // Extra fractional digits beyond nanoseconds: Go rounds; the agent
        // never emits them, so accept-and-truncate would hide corruption.
        // Reject to fail closed.
        if !rest.is_empty() && rest[0].is_ascii_digit() {
            return None;
        }
        for _ in count..9 {
            value *= 10;
        }
        nanos = value;
    }
    let offset: i64 = if rest == b"Z" {
        0
    } else if rest.len() >= 3 && (rest[0] == b'+' || rest[0] == b'-') {
        let sign = if rest[0] == b'-' { -1 } else { 1 };
        let (hours, minutes) = match rest.len() {
            3 => (parse_two(&rest[1..3])?, 0),
            5 => (parse_two(&rest[1..3])?, parse_two(&rest[3..5])?),
            6 if rest[3] == b':' => (parse_two(&rest[1..3])?, parse_two(&rest[4..6])?),
            _ => return None,
        };
        if hours > 23 || minutes > 59 {
            return None;
        }
        sign * (hours * 3600 + minutes * 60)
    } else {
        return None;
    };
    let days = days_from_civil(year, month, day);
    Some((
        days * 86400 + hour * 3600 + minute * 60 + second - offset,
        nanos,
    ))
}

// ---------- identity wire types ----------

/// Native process boundary (`identity.Binding`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Binding {
    pub child_id: String,
    pub uid: i64,
    pub gid: i64,
    pub scope: String,
    pub credential_root: String,
    pub invocation_id: String,
    pub kind: String,
    pub id: String,
    pub project: String,
    pub login: String,
    pub generation: i64,
}

const BINDING_SPECS: &[Spec] = &[
    Spec {
        name: "child_id",
        kind: Kind::Str,
    },
    Spec {
        name: "uid",
        kind: Kind::Int,
    },
    Spec {
        name: "gid",
        kind: Kind::Int,
    },
    Spec {
        name: "scope",
        kind: Kind::Str,
    },
    Spec {
        name: "credential_root",
        kind: Kind::Str,
    },
    Spec {
        name: "invocation_id",
        kind: Kind::Str,
    },
    Spec {
        name: "kind",
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
        name: "generation",
        kind: Kind::I64,
    },
];

/// `,string` int64 decoding, probed against `encoding/json`: the JSON value
/// must be a quoted string starting with `-` or a digit, holding a plain
/// base-10 integer (leading zeros allowed, no fraction/exponent/space).
pub fn parse_string_i64(raw: &str) -> Option<i64> {
    let b = raw.as_bytes();
    let &first = b.first()?;
    if first != b'-' && !first.is_ascii_digit() {
        return None;
    }
    let digits = if first == b'-' { &b[1..] } else { b };
    if digits.is_empty() || !digits.iter().all(|c| c.is_ascii_digit()) {
        return None;
    }
    raw.parse::<i64>().ok()
}

/// Execution lease (`identity.Lease`). The deadline keeps its raw wire text
/// for byte-exact re-encoding plus a parsed instant for comparisons.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Lease {
    pub repository_id: i64,
    pub provider_id: String,
    pub id: String,
    pub connection_id: String,
    pub generation: i64,
    pub actor_id: i64,
    pub project_id: String,
    pub execution_id: String,
    pub kind: String,
    pub role: String,
    pub deadline_raw: String,
    pub deadline: Option<(i64, u32)>,
    pub grant_id: String,
    pub grant_revision: i64,
    pub binding: Option<Binding>,
}

const LEASE_SPECS: &[Spec] = &[
    Spec {
        name: "repository_id",
        kind: Kind::Str,
    },
    Spec {
        name: "provider_id",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "connection_id",
        kind: Kind::Str,
    },
    Spec {
        name: "generation",
        kind: Kind::I64,
    },
    Spec {
        name: "actor_id",
        kind: Kind::Str,
    },
    Spec {
        name: "project_id",
        kind: Kind::Str,
    },
    Spec {
        name: "execution_id",
        kind: Kind::Str,
    },
    Spec {
        name: "kind",
        kind: Kind::Str,
    },
    Spec {
        name: "role",
        kind: Kind::Str,
    },
    Spec {
        name: "deadline",
        kind: Kind::Str,
    },
    Spec {
        name: "grant_id",
        kind: Kind::Str,
    },
    Spec {
        name: "grant_revision",
        kind: Kind::I64,
    },
    Spec {
        name: "binding",
        kind: Kind::OptObject {
            go_type: "*identity.Binding",
            struct_name: "Binding",
            specs: BINDING_SPECS,
        },
    },
];

fn binding_from_map(m: &BoundMap) -> Binding {
    Binding {
        child_id: m.take_string("child_id"),
        uid: m.take_i64("uid"),
        gid: m.take_i64("gid"),
        scope: m.take_string("scope"),
        credential_root: m.take_string("credential_root"),
        invocation_id: m.take_string("invocation_id"),
        kind: m.take_string("kind"),
        id: m.take_string("id"),
        project: m.take_string("project"),
        login: m.take_string("login"),
        generation: m.take_i64("generation"),
    }
}

fn lease_from_map(m: &BoundMap) -> Result<Lease, String> {
    let repository_id = if m.contains("repository_id") {
        parse_string_i64(&m.take_string("repository_id"))
            .ok_or_else(|| "invalid repository_id".to_string())?
    } else {
        0
    };
    let actor_id = if m.contains("actor_id") {
        parse_string_i64(&m.take_string("actor_id"))
            .ok_or_else(|| "invalid actor_id".to_string())?
    } else {
        0
    };
    let deadline_raw = m.take_string("deadline");
    let deadline = if m.contains("deadline") {
        Some(parse_rfc3339(&deadline_raw).ok_or_else(|| "invalid deadline".to_string())?)
    } else {
        None
    };
    Ok(Lease {
        repository_id,
        provider_id: m.take_string("provider_id"),
        id: m.take_string("id"),
        connection_id: m.take_string("connection_id"),
        generation: m.take_i64("generation"),
        actor_id,
        project_id: m.take_string("project_id"),
        execution_id: m.take_string("execution_id"),
        kind: m.take_string("kind"),
        role: m.take_string("role"),
        deadline_raw,
        deadline,
        grant_id: m.take_string("grant_id"),
        grant_revision: m.take_i64("grant_revision"),
        binding: m.take_opt_map("binding").map(|b| binding_from_map(&b)),
    })
}

impl Binding {
    /// `Binding.Validate()`.
    pub fn validate(&self) -> Result<(), String> {
        if self.id.is_empty()
            || self.generation <= 0
            || (self.kind != KIND_FACTORY && self.kind != KIND_TERMINAL)
        {
            return Err(err_denied());
        }
        if self.kind == KIND_TERMINAL && (self.project.is_empty() || self.login.trim().is_empty()) {
            return Err(err_denied());
        }
        Ok(())
    }

    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        let mut field = |out: &mut String, name: &str, value: &str| {
            if !first {
                out.push(',');
            }
            first = false;
            out.push_str(&json::quote(name));
            out.push(':');
            out.push_str(value);
        };
        if !self.child_id.is_empty() {
            field(out, "child_id", &json::quote(&self.child_id));
        }
        if self.uid != 0 {
            field(out, "uid", &self.uid.to_string());
        }
        if self.gid != 0 {
            field(out, "gid", &self.gid.to_string());
        }
        if !self.scope.is_empty() {
            field(out, "scope", &json::quote(&self.scope));
        }
        if !self.credential_root.is_empty() {
            field(out, "credential_root", &json::quote(&self.credential_root));
        }
        if !self.invocation_id.is_empty() {
            field(out, "invocation_id", &json::quote(&self.invocation_id));
        }
        field(out, "kind", &json::quote(&self.kind));
        field(out, "id", &json::quote(&self.id));
        field(out, "project", &json::quote(&self.project));
        field(out, "login", &json::quote(&self.login));
        field(out, "generation", &self.generation.to_string());
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

impl Lease {
    /// Strict decode of one lease object.
    pub fn decode_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_struct(v, "Lease", &[], LEASE_SPECS, false).map_err(|e| e.0)?;
        lease_from_map(&m)
    }

    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        let mut field = |out: &mut String, name: &str, value: &str| {
            if !first {
                out.push(',');
            }
            first = false;
            out.push_str(&json::quote(name));
            out.push(':');
            out.push_str(value);
        };
        if self.repository_id != 0 {
            field(
                out,
                "repository_id",
                &json::quote(&self.repository_id.to_string()),
            );
        }
        field(out, "provider_id", &json::quote(&self.provider_id));
        field(out, "id", &json::quote(&self.id));
        field(out, "connection_id", &json::quote(&self.connection_id));
        field(out, "generation", &self.generation.to_string());
        field(out, "actor_id", &json::quote(&self.actor_id.to_string()));
        field(out, "project_id", &json::quote(&self.project_id));
        field(out, "execution_id", &json::quote(&self.execution_id));
        field(out, "kind", &json::quote(&self.kind));
        if !self.role.is_empty() {
            field(out, "role", &json::quote(&self.role));
        }
        // Go marshals the zero time as `0001-01-01T00:00:00Z` (probed); an
        // empty raw deadline is that zero time, never an empty string.
        let deadline = if self.deadline_raw.is_empty() {
            "0001-01-01T00:00:00Z"
        } else {
            &self.deadline_raw
        };
        field(out, "deadline", &json::quote(deadline));
        if !self.grant_id.is_empty() {
            field(out, "grant_id", &json::quote(&self.grant_id));
        }
        if self.grant_revision != 0 {
            field(out, "grant_revision", &self.grant_revision.to_string());
        }
        if let Some(binding) = &self.binding {
            let mut nested = String::new();
            binding.encode_into(&mut nested);
            field(out, "binding", &nested);
        }
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

/// Trusted-process result (`identity.Delivery` / `DeliveryWire`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Delivery {
    pub lease: Lease,
    pub credential: Option<Vec<u8>>,
}

const DELIVERY_SPECS: &[Spec] = &[
    Spec {
        name: "lease",
        kind: Kind::Object {
            go_type: "identity.Lease",
            struct_name: "Lease",
            specs: LEASE_SPECS,
        },
    },
    Spec {
        name: "credential",
        kind: Kind::Bytes,
    },
];

impl Delivery {
    /// Strict decode of one delivery object. `,string` and deadline fields
    /// are re-validated after the shared binder runs.
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        Self::decode_value(&v)
    }

    pub fn decode_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "DeliveryWire", DELIVERY_SPECS, false).map_err(|e| e.0)?;
        let lease = lease_from_map(&m.take_map("lease"))?;
        let credential = if m.contains("credential") {
            Some(m.take_bytes("credential"))
        } else {
            None
        };
        Ok(Delivery { lease, credential })
    }

    pub fn encode_into(&self, out: &mut String) {
        out.push_str("{\"lease\":");
        self.lease.encode_into(out);
        out.push_str(",\"credential\":");
        match &self.credential {
            None => out.push_str("null"),
            Some(bytes) => out.push_str(&json::quote(&crate::ssh::b64_encode(bytes))),
        }
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

/// Lease acquisition input (`identity.AcquireRequest`); in-process only,
/// never on the JSON wire. Deadline is Unix seconds plus nanoseconds.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct AcquireRequest {
    pub repository_id: i64,
    pub provider_id: String,
    pub execution_id: String,
    pub actor_id: i64,
    pub connection_id: String,
    pub project_id: String,
    pub kind: String,
    pub deadline_secs: i64,
    pub deadline_nanos: u32,
    pub role: String,
}

// ---------- container inspection ----------

/// `terminalInspect`: the exact podman `--format` template.
pub const TERMINAL_INSPECT: &str = "{\"id\":{{json .ID}},\"running\":{{json .State.Running}},\"project\":{{json (index .Config.Labels \"org.soda.project\")}},\"owner\":{{json (index .Config.Labels \"org.soda.owner\")}},\"privileged\":{{json .HostConfig.Privileged}},\"userns\":{{json .HostConfig.UsernsMode}},\"mappings\":{{json .HostConfig.IDMappings}}}";

/// `terminalInspection`: strict-decoded podman inspect output.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TerminalInspection {
    pub id: String,
    pub running: bool,
    pub project: String,
    pub owner: String,
    pub privileged: bool,
    pub userns: String,
    pub uid_map: Vec<String>,
    pub gid_map: Vec<String>,
}

const MAPPINGS_SPECS: &[Spec] = &[
    Spec {
        name: "UidMap",
        kind: Kind::StrList,
    },
    Spec {
        name: "GidMap",
        kind: Kind::StrList,
    },
];

const INSPECTION_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "running",
        kind: Kind::Bool,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "owner",
        kind: Kind::Str,
    },
    Spec {
        name: "privileged",
        kind: Kind::Bool,
    },
    Spec {
        name: "userns",
        kind: Kind::Str,
    },
    Spec {
        name: "mappings",
        kind: Kind::Object {
            go_type: "struct",
            struct_name: "struct",
            specs: MAPPINGS_SPECS,
        },
    },
];

impl TerminalInspection {
    /// Strict decode (`strictjson.Decode` in Go).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        let m =
            json::bind_root(&v, "terminalInspection", INSPECTION_SPECS, false).map_err(|e| e.0)?;
        let mappings = m.take_map("mappings");
        Ok(TerminalInspection {
            id: m.take_string("id"),
            running: m.take_bool("running"),
            project: m.take_string("project"),
            owner: m.take_string("owner"),
            privileged: m.take_bool("privileged"),
            userns: m.take_string("userns"),
            uid_map: mappings.take_str_list("UidMap"),
            gid_map: mappings.take_str_list("GidMap"),
        })
    }
}

/// `terminalIDMap`: exactly one `0:<base>:262144` mapping with host root
/// shifted away from container root.
pub fn terminal_id_map(values: &[String]) -> bool {
    if values.len() != 1 {
        return false;
    }
    let parts: Vec<&str> = values[0].split(':').collect();
    if parts.len() != 3 || parts[0] != "0" || parts[2] != "262144" {
        return false;
    }
    let Some(base) = parse_go_uint(parts[1], 32) else {
        return false;
    };
    // Canonical re-format check rejects leading zeros, as in Go.
    base.to_string() == parts[1] && base > 0 && base + 262144 <= 4294967295
}

/// `terminalIsolation`: exact container identity, project label,
/// unprivileged private userns, production ID mappings.
pub fn terminal_isolation(v: &TerminalInspection, id: &str) -> bool {
    if !domain::valid_container_id(&v.id)
        || v.project != id
        || v.privileged
        || v.userns != "private"
    {
        return false;
    }
    terminal_id_map(&v.uid_map) && terminal_id_map(&v.gid_map)
}

/// Go `strconv.ParseInt(s, 10, 64)`, probed: one optional sign, ASCII
/// digits, range-checked. Rust's `FromStr` matches exactly (leading `+`
/// and zeros accepted, underscores/spaces rejected).
pub(crate) fn parse_go_int(s: &str) -> Option<i64> {
    s.parse::<i64>().ok()
}

/// Go `strconv.ParseUint(s, 10, bits)`, probed: ASCII digits only (no
/// sign), range-checked.
pub(crate) fn parse_go_uint(s: &str, bits: u32) -> Option<u64> {
    if s.is_empty() || !s.bytes().all(|b| b.is_ascii_digit()) {
        return None;
    }
    let value: u64 = s.parse().ok()?;
    let max = if bits >= 64 {
        u64::MAX
    } else {
        (1u64 << bits) - 1
    };
    if value > max {
        None
    } else {
        Some(value)
    }
}

/// `terminalTargetReady`: positive owner plus isolation, running when required.
pub fn terminal_target_ready(v: &TerminalInspection, id: &str, require_running: bool) -> bool {
    let Some(owner) = parse_go_int(&v.owner) else {
        return false;
    };
    if owner <= 0 {
        return false;
    }
    if require_running && !v.running {
        return false;
    }
    terminal_isolation(v, id)
}

/// Parse the exit code out of a crate-executor error string. Both the real
/// executor (`{cmd} failed: exit status N: {stderr}`) and test fakes (bare
/// `exit status N`) put the true code at the first `exit status ` marker;
/// anything else fails closed to `None`.
pub fn exit_code_of(err: &str) -> Option<i32> {
    let marker = "exit status ";
    let start = err.find(marker)? + marker.len();
    let digits: String = err[start..]
        .chars()
        .take_while(|c| c.is_ascii_digit())
        .collect();
    if digits.is_empty() || digits.len() > 10 {
        return None;
    }
    digits.parse::<i32>().ok()
}

// ---------- terminal service ----------

/// Broker custody callback behind `end` reconciliation.
pub type EndIdentityHook<'a> = &'a dyn Fn(i64, &str) -> Result<(), String>;

/// Privileged project-terminal executor (`terminal.Service`).
/// `EndIdentity` is passed per call so the service stays dependency-free.
pub struct Service<E> {
    pub exec: E,
    pub codex_harness: String,
    pub codex_harness_sha256: String,
    pub codex_harness_version: String,
    pub muse_harness: String,
    pub muse_harness_sha256: String,
    pub muse_harness_version: String,
}

/// `podman --remote=false inspect --format <terminalInspect> <name>`.
pub fn inspect_argv(name: &str) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "inspect".to_string(),
        "--format".to_string(),
        TERMINAL_INSPECT.to_string(),
        name.to_string(),
    ]
}

/// `podman --remote=false container exists <target>`.
pub fn container_exists_argv(target: &str) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "container".to_string(),
        "exists".to_string(),
        target.to_string(),
    ]
}

/// Fixed `podman exec` of the project terminal agent (`AgentExec` argv).
/// Python is gone: the agent is the native `project-terminal` binary.
pub fn agent_argv(container: &str, args: &[&str]) -> Vec<String> {
    let mut argv = vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
        container.to_string(),
        AGENT_PROGRAM.to_string(),
    ];
    argv.extend(args.iter().map(|s| s.to_string()));
    argv
}

impl<E: Executor> Service<E> {
    fn podman(&self, stdin: &[u8], args: &[&str], deadline: Instant) -> Result<Vec<u8>, String> {
        self.exec.run(stdin, "/usr/bin/podman", args, deadline)
    }

    fn podman_owned(
        &self,
        stdin: &[u8],
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let refs: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
        self.podman(stdin, &refs, deadline)
    }

    /// `Service.projectContainer`: fixed-name inspect with the full
    /// isolation gate. Native lifecycle may inspect stopped containers.
    pub fn project_container(
        &self,
        id: &str,
        require_running: bool,
        deadline: Instant,
    ) -> Result<String, String> {
        if !domain::valid_id(id) {
            return Err("invalid project".to_string());
        }
        let argv = inspect_argv(&format!("soda-{id}"));
        let data = match self.podman_owned(&[], &argv, deadline) {
            Ok(data) if data.len() <= 4096 => data,
            _ => return Err("terminal inspection unavailable".to_string()),
        };
        let v = TerminalInspection::decode(&data)
            .map_err(|_| "invalid terminal inspection".to_string())?;
        if !terminal_target_ready(&v, id, require_running) {
            return Err("terminal target not ready or isolated".to_string());
        }
        Ok(v.id)
    }

    /// `Service.factoryProjectContainer`: supervised factory runs bind the
    /// exact container incarnation instead of the 262144-mapping profile.
    pub fn factory_project_container(
        &self,
        id: &str,
        require_running: bool,
        deadline: Instant,
    ) -> Result<String, String> {
        if !domain::valid_id(id) {
            return Err(err_denied());
        }
        let argv = inspect_argv(&format!("soda-{id}"));
        let data = match self.podman_owned(&[], &argv, deadline) {
            Ok(data) if !data.is_empty() && data.len() <= 16384 => data,
            _ => return Err(err_stale()),
        };
        let v = TerminalInspection::decode(&data).map_err(|_| err_stale())?;
        if !domain::valid_container_id(&v.id)
            || v.project != id
            || v.privileged
            || v.userns != "private"
        {
            return Err(err_denied());
        }
        let Some(owner) = parse_go_int(&v.owner) else {
            return Err(err_denied());
        };
        if owner <= 0 {
            return Err(err_denied());
        }
        if require_running && !v.running {
            return Err(err_stale());
        }
        Ok(v.id)
    }
}

// ---------- identity broker calls (identity.go) ----------

/// Fixed agent request envelope (`identityRequest`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct IdentityRequest {
    pub action: String,
    pub delivery: Delivery,
    pub login: String,
    pub scope: String,
    pub cols: i64,
    pub rows: i64,
    pub source_hash: String,
    pub container: String,
    pub harness_sha256: String,
}

impl IdentityRequest {
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"action\":");
        out.push_str(&json::quote(&self.action));
        out.push_str(",\"delivery\":");
        out.push_str(&self.delivery.encode());
        if !self.login.is_empty() {
            out.push_str(",\"login\":");
            out.push_str(&json::quote(&self.login));
        }
        if !self.scope.is_empty() {
            out.push_str(",\"scope\":");
            out.push_str(&json::quote(&self.scope));
        }
        if self.cols != 0 {
            out.push_str(",\"cols\":");
            out.push_str(&self.cols.to_string());
        }
        if self.rows != 0 {
            out.push_str(",\"rows\":");
            out.push_str(&self.rows.to_string());
        }
        if !self.source_hash.is_empty() {
            out.push_str(",\"source_hash\":");
            out.push_str(&json::quote(&self.source_hash));
        }
        if !self.container.is_empty() {
            out.push_str(",\"container\":");
            out.push_str(&json::quote(&self.container));
        }
        if !self.harness_sha256.is_empty() {
            out.push_str(",\"harness_sha256\":");
            out.push_str(&json::quote(&self.harness_sha256));
        }
        out.push('}');
        out
    }
}

fn terminal_reservation(lease: &Lease) -> bool {
    lease.kind == KIND_TERMINAL
        && valid_terminal_id(&lease.execution_id)
        && domain::valid_id(&lease.project_id)
        && lease.actor_id > 0
        && !lease.id.is_empty()
        && lease.generation > 0
}

fn terminal_binding(lease: &Lease) -> bool {
    let Some(b) = &lease.binding else {
        return false;
    };
    b.kind == KIND_TERMINAL
        && b.id == lease.execution_id
        && domain::valid_container_id(&b.project)
        && domain::valid_login(&b.login)
        && b.login != "root"
        && b.generation == lease.generation
}

/// `terminalLease`: reservation shape, plus binding (live) or deadline
/// window (preparing).
pub fn terminal_lease(lease: &Lease, preparing: bool, now: i64) -> bool {
    if !terminal_reservation(lease) {
        return false;
    }
    if preparing {
        return lease.binding.is_none()
            && matches!(lease.deadline, Some((secs, _)) if secs > now && secs <= now + 12 * 3600);
    }
    terminal_binding(lease)
}

fn terminal_preparation(
    lease: &Lease,
    login: &str,
    scope: &str,
    cols: i64,
    rows: i64,
    now: i64,
) -> bool {
    terminal_lease(lease, true, now)
        && domain::valid_login(login)
        && login != "root"
        && domain::valid_container_id(scope)
        && terminal_dimensions(cols, rows)
}

fn terminal_prepared(
    result: &Delivery,
    lease: &Lease,
    container: &str,
    login: &str,
    now: i64,
) -> bool {
    if !terminal_lease(&result.lease, false, now) {
        return false;
    }
    result.lease.id == lease.id
        && result
            .lease
            .binding
            .as_ref()
            .map(|b| b.project == container && b.login == login)
            .unwrap_or(false)
        && result
            .credential
            .as_ref()
            .map(|c| c.is_empty())
            .unwrap_or(true)
}

impl<E: Executor> Service<E> {
    /// `Service.identityCall`: fixed `broker` agent invocation over stdin.
    pub fn identity_call(
        &self,
        container: &str,
        request: &IdentityRequest,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        let body = request.encode().into_bytes();
        let argv = agent_argv(container, &["broker"]);
        let out = match self.podman_owned(&body, &argv, deadline) {
            Ok(out) => out,
            Err(_) => return Err("managed Codex operation failed".to_string()),
        };
        if out.len() > BROKER_RESPONSE_LIMIT {
            return Err("invalid managed Codex response".to_string());
        }
        Delivery::decode(&out).map_err(|_| "invalid managed Codex response".to_string())
    }

    /// `Service.PrepareIdentity`: reserve one native terminal before the
    /// broker releases bytes.
    pub fn prepare_identity(
        &self,
        lease: &Lease,
        login: &str,
        scope: &str,
        cols: i64,
        rows: i64,
        deadline: Instant,
    ) -> Result<Binding, String> {
        let now = now_unix();
        if !terminal_preparation(lease, login, scope, cols, rows, now) {
            return Err(err_denied());
        }
        let container = self.project_container(&lease.project_id, true, deadline)?;
        let hash = agent_program_hash(0)?;
        let request = IdentityRequest {
            action: "prepare".to_string(),
            delivery: Delivery {
                lease: lease.clone(),
                credential: None,
            },
            login: login.to_string(),
            scope: scope.to_string(),
            cols,
            rows,
            source_hash: hash,
            container: container.clone(),
            ..Default::default()
        };
        let result = self.identity_call(&container, &request, deadline)?;
        if !terminal_prepared(&result, lease, &container, login, now_unix()) {
            return Err("managed terminal reservation differs".to_string());
        }
        result
            .lease
            .binding
            .clone()
            .ok_or_else(|| "managed terminal reservation differs".to_string())
    }

    fn identity_action(&self, action: &str, delivery: &Delivery) -> bool {
        match action {
            "start" => {
                delivery
                    .credential
                    .as_ref()
                    .map(|c| credential_valid(c))
                    .unwrap_or(false)
                    && self.codex_harness.starts_with('/')
                    && domain::valid_container_id(&self.codex_harness_sha256)
            }
            "validate" | "finish" | "stop" => delivery
                .credential
                .as_ref()
                .map(|c| c.is_empty())
                .unwrap_or(true),
            _ => false,
        }
    }

    /// `Service.Identity`: fixed model-session operations.
    pub fn identity(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        if !terminal_lease(&delivery.lease, false, now_unix()) {
            return Err(err_denied());
        }
        if !self.identity_action(action, delivery) {
            return Err(err_denied());
        }
        let (container, done) = self.identity_target(action, &delivery.lease, deadline)?;
        if done {
            return Ok(delivery.clone());
        }
        if action == "start" {
            self.identity_stage(&container, delivery, deadline)?;
        }
        let request = IdentityRequest {
            action: action.to_string(),
            delivery: delivery.clone(),
            harness_sha256: self.codex_harness_sha256.clone(),
            ..Default::default()
        };
        let result = self.identity_call(&container, &request, deadline)?;
        identity_result(action, delivery, &result)
    }

    /// `Service.managedEnd`: reconcile the managed lease behind an `end`
    /// before the launcher tears the terminal down.
    pub fn managed_end(
        &self,
        container: &str,
        request: &TerminalRequest,
        end_identity: Option<EndIdentityHook<'_>>,
        deadline: Instant,
    ) -> Result<(), String> {
        if request.action != "end" {
            return Ok(());
        }
        let lookup = IdentityRequest {
            action: "lookup".to_string(),
            delivery: Delivery {
                lease: Lease {
                    execution_id: request.id.clone(),
                    actor_id: request.identity,
                    ..Default::default()
                },
                credential: None,
            },
            login: request.login.clone(),
            ..Default::default()
        };
        let result = self.identity_call(container, &lookup, deadline)?;
        if result.lease.id.is_empty() {
            return Ok(());
        }
        let bound = result
            .lease
            .binding
            .as_ref()
            .map(|b| b.project == container && b.id == request.id)
            .unwrap_or(false);
        if !bound || result.lease.actor_id != request.identity {
            return Err(err_stale());
        }
        match end_identity {
            None => Err(err_denied()),
            Some(end) => end(request.identity, &result.lease.id),
        }
    }

    /// `Service.verifyIdentityHarness`: the staged host Codex bytes must
    /// match the pinned digest.
    pub fn verify_identity_harness(&self) -> Result<(), String> {
        let path = format!("{}/bin/codex", self.codex_harness.trim_end_matches('/'));
        // Go gates on `Lstat` regularity plus any exec bit before opening.
        let lstat = std::fs::symlink_metadata(&path)
            .map_err(|_| "verified Codex executable required".to_string())?;
        if !lstat.file_type().is_file() || lstat.permissions().mode() & 0o111 == 0 {
            return Err("verified Codex executable required".to_string());
        }
        let mut data = Vec::new();
        use std::io::Read;
        std::fs::File::open(&path)
            .map_err(|_| "verified Codex executable required".to_string())?
            .read_to_end(&mut data)
            .map_err(|e| format!("codex harness unreadable: {e}"))?;
        if sha256::hex_lower(&sha256::digest(&data)) != self.codex_harness_sha256 {
            return Err("codex harness digest differs".to_string());
        }
        Ok(())
    }

    fn identity_target(
        &self,
        action: &str,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<(String, bool), String> {
        if action != "stop" {
            let target = self.project_container(&lease.project_id, true, deadline)?;
            let bound = lease
                .binding
                .as_ref()
                .map(|b| b.project.clone())
                .unwrap_or_default();
            if target != bound {
                return Err(err_stale());
            }
            return Ok((target, false));
        }
        self.identity_stop_target(lease, deadline)
    }

    fn identity_stop_target(
        &self,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<(String, bool), String> {
        let target = lease
            .binding
            .as_ref()
            .map(|b| b.project.clone())
            .unwrap_or_default();
        let argv = container_exists_argv(&target);
        if let Err(err) = self.podman_owned(&[], &argv, deadline) {
            if exit_code_of(&err) == Some(1) {
                return Ok((target, true));
            }
            return Err(err_uncertain());
        }
        let argv = inspect_argv(&target);
        let data = match self.podman_owned(&[], &argv, deadline) {
            Ok(data) if data.len() <= 4096 => data,
            _ => return Err(err_uncertain()),
        };
        let v = TerminalInspection::decode(&data).map_err(|_| err_uncertain())?;
        if v.id != target || !terminal_target_ready(&v, &lease.project_id, false) {
            return Err(err_stale());
        }
        Ok((target, !v.running))
    }

    fn identity_stage(
        &self,
        container: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<(), String> {
        self.verify_identity_harness()?;
        let request = IdentityRequest {
            action: "stage".to_string(),
            delivery: delivery.clone(),
            ..Default::default()
        };
        self.identity_call(container, &request, deadline)?;
        let path = format!(
            "/run/soda-terminals/{}/model/harness",
            delivery.lease.execution_id
        );
        self.stream_identity_harness(container, &path, deadline)
    }
}

/// `identityResult`: the agent must echo the binding; only `finish`
/// returns credential bytes.
pub fn identity_result(
    action: &str,
    delivery: &Delivery,
    result: &Delivery,
) -> Result<Delivery, String> {
    // Go returns the partial result alongside the error; the daemon drops
    // it on every error path, so only the verdict is preserved here.
    if result.lease.id != delivery.lease.id || result.lease.binding.is_none() {
        return Err(err_stale());
    }
    if result.lease.binding != delivery.lease.binding {
        return Err(err_stale());
    }
    if action == "finish" {
        match &result.credential {
            Some(c) if credential_valid(c) => {}
            _ => return Err(err_uncertain()),
        }
    } else if result
        .credential
        .as_ref()
        .map(|c| !c.is_empty())
        .unwrap_or(false)
    {
        return Err("unexpected credential response".to_string());
    }
    Ok(result.clone())
}

// ---------- identity harness transfer (identity_transfer.go) ----------

/// Host tar producer argv: `tar --create --file=- --directory <harness> .`.
/// The harness path is cleaned exactly like Go's `filepath.Clean`.
pub fn tar_producer_argv(harness: &str) -> Vec<String> {
    vec![
        "--create".to_string(),
        "--file=-".to_string(),
        "--directory".to_string(),
        clean_path(harness),
        ".".to_string(),
    ]
}

/// Guest tar consumer argv: `podman ... exec --interactive <container> tar
/// --extract --file=- --directory <path> --no-same-owner --same-permissions`.
pub fn tar_consumer_argv(container: &str, path: &str) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
        container.to_string(),
        "/usr/bin/tar".to_string(),
        "--extract".to_string(),
        "--file=-".to_string(),
        "--directory".to_string(),
        path.to_string(),
        "--no-same-owner".to_string(),
        "--same-permissions".to_string(),
    ]
}

/// Lexical path cleaning matching Go `path/filepath.Clean` (Linux).
pub fn clean_path(path: &str) -> String {
    let rooted = path.starts_with('/');
    let mut parts: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if parts.pop().is_none() && !rooted {
                    parts.push("..");
                }
            }
            _ => parts.push(part),
        }
    }
    let mut out = parts.join("/");
    if rooted {
        out.insert(0, '/');
    }
    if out.is_empty() {
        out.push('.');
    }
    out
}

impl<E: Executor> Service<E> {
    /// `Service.streamIdentityHarness`: stage the verified harness bytes
    /// into the guest tmpfs. The Go implementation streams producer to
    /// consumer over a pipe; this port buffers the producer output through
    /// the executor (error strings unchanged).
    pub fn stream_identity_harness(
        &self,
        container: &str,
        path: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let producer = tar_producer_argv(&self.codex_harness);
        let refs: Vec<&str> = producer.iter().map(|s| s.as_str()).collect();
        let stream = match self.exec.run(&[], "/usr/bin/tar", &refs, deadline) {
            Ok(stream) => stream,
            Err(_) => return Err("codex harness stream unavailable".to_string()),
        };
        let consumer = tar_consumer_argv(container, path);
        let refs: Vec<&str> = consumer.iter().map(|s| s.as_str()).collect();
        match self.exec.run(&stream, "/usr/bin/podman", &refs, deadline) {
            Ok(_) => Ok(()),
            Err(_) => Err("codex harness staging failed".to_string()),
        }
    }
}

// ---------- agent binary verification (agent.go) ----------

/// Host-side agent path: `SODA_PROJECT_TERMINAL` override or the fixed
/// program path. Ownership/mode/size gates still apply to overrides.
pub fn agent_program_path() -> String {
    std::env::var("SODA_PROJECT_TERMINAL")
        .ok()
        .filter(|v| !v.is_empty())
        .unwrap_or_else(|| AGENT_PROGRAM.to_string())
}

/// `agentProgramHash(uid)`: verify the fixed host-side agent binary and
/// return its lowercase hex SHA-256. Fails closed on missing, non-regular,
/// wrongly owned, writable, or oddly sized files. Production passes 0.
pub fn agent_program_hash(uid: u32) -> Result<String, String> {
    let path = agent_program_path();
    let info = std::fs::symlink_metadata(&path)
        .map_err(|e| format!("project terminal agent unavailable: {e}"))?;
    if !info.file_type().is_file() {
        return Err("project terminal agent is not a regular file".to_string());
    }
    if info.uid() != uid {
        return Err("project terminal agent has unexpected ownership".to_string());
    }
    if info.permissions().mode() & 0o022 != 0 {
        return Err("project terminal agent is group- or world-writable".to_string());
    }
    if info.len() < 1 || info.len() > 32 << 20 {
        return Err("project terminal agent has unexpected size".to_string());
    }
    let raw =
        std::fs::read(&path).map_err(|e| format!("project terminal agent unreadable: {e}"))?;
    if raw.is_empty() || raw.len() > 32 << 20 {
        return Err("project terminal agent changed during verification".to_string());
    }
    Ok(sha256::hex_lower(&sha256::digest(&raw)))
}

// ---------- native attach (native.go) ----------

/// `AttachNative` argv: the fixed podman/agent attachment bridge.
/// `seconds` is the attachment deadline (`expires - now`), already checked
/// positive by the caller path below.
#[allow(clippy::too_many_arguments)] // one parameter per fixed argv word, in order
pub fn native_argv(
    container: &str,
    action: &str,
    id: &str,
    login: &str,
    identity: i64,
    cols: i64,
    rows: i64,
    seconds: i64,
    hash: &str,
    name: &str,
    scope: &str,
) -> Vec<String> {
    agent_argv(
        container,
        &[
            action,
            id,
            login,
            &identity.to_string(),
            &cols.to_string(),
            &rows.to_string(),
            &seconds.to_string(),
            hash,
            name,
            scope,
        ],
    )
}

/// Parse one agent stdout line: strict JSON frame plus `OutputValid`.
/// Scanner-failure (overlong line, EOF) is `None`; decode/validity failure
/// is the `invalid terminal response` error.
pub fn parse_output_line(line: &[u8]) -> Result<TerminalFrame, String> {
    TerminalFrame::decode(line)
        .map_err(|_| "invalid terminal response".to_string())
        .and_then(|f| {
            if f.output_valid() {
                Ok(f)
            } else {
                Err("invalid terminal response".to_string())
            }
        })
}

/// Streaming native attachment (`nativeTerminal`). Terminal bytes never
/// enter diagnostics; stdin close requests launcher EOF.
#[derive(Debug)]
pub struct NativeAttach {
    child: Option<std::process::Child>,
    stdin: Option<File>,
    reader: Option<BufReader<File>>,
    closed: bool,
}

impl NativeAttach {
    /// `AttachNative`: start the fixed podman/agent attachment bridge.
    pub fn attach(container: &str, input: &TerminalRequest) -> Result<Self, String> {
        let seconds = input.expires - now_unix();
        if !domain::valid_container_id(container) || !input.valid(now_unix()) || seconds < 1 {
            return Err("invalid terminal target".to_string());
        }
        let hash = agent_program_hash(0)?;
        let argv = native_argv(
            container,
            &input.action,
            &input.id,
            &input.login,
            input.identity,
            input.cols,
            input.rows,
            seconds,
            &hash,
            &input.name,
            &input.scope,
        );
        let mut child = std::process::Command::new("/usr/bin/podman")
            .args(&argv)
            .stdin(std::process::Stdio::piped())
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::null())
            .spawn()
            .map_err(|e| format!("/usr/bin/podman failed: {e}"))?;
        let stdin = child
            .stdin
            .take()
            .ok_or_else(|| "terminal stdin unavailable".to_string())?;
        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| "terminal stdout unavailable".to_string())?;
        use std::os::unix::io::FromRawFd;
        // `ChildStdin` has no timeout API; convert to `File` for deadlines.
        // SAFETY: the stdio handles are owned by us exactly once.
        let stdin_fd = stdin.as_raw_fd();
        let stdout_fd = stdout.as_raw_fd();
        std::mem::forget(stdin);
        std::mem::forget(stdout);
        let stdin = unsafe { File::from_raw_fd(stdin_fd) };
        let stdout = unsafe { File::from_raw_fd(stdout_fd) };
        Ok(NativeAttach {
            child: Some(child),
            stdin: Some(stdin),
            reader: Some(BufReader::new(stdout)),
            closed: false,
        })
    }

    /// `nativeTerminal.Input`: one validated frame over stdin (2s deadline).
    pub fn input_frame(&mut self, f: &TerminalFrame) -> Result<(), String> {
        if !f.input_valid() {
            return Err("invalid terminal control".to_string());
        }
        let stdin = self
            .stdin
            .as_mut()
            .ok_or_else(|| "terminal input ended".to_string())?;
        // `File` has no write deadline; poll for writability like Go's
        // `SetWriteDeadline` on the stdin pipe.
        let mut pfd = libc::pollfd {
            fd: stdin.as_raw_fd(),
            events: libc::POLLOUT,
            revents: 0,
        };
        let ready = unsafe { libc::poll(&mut pfd, 1, 2000) };
        if ready == 0 {
            return Err("terminal input deadline exceeded".to_string());
        }
        if ready < 0 || pfd.revents & (libc::POLLERR | libc::POLLHUP | libc::POLLNVAL) != 0 {
            return Err("terminal input ended".to_string());
        }
        let mut body = f.encode().into_bytes();
        body.push(b'\n');
        stdin
            .write_all(&body)
            .map_err(|e| format!("terminal input ended: {e}"))?;
        Ok(())
    }

    /// Detach the stdout reader for lock-free output pumps (H01-F3).
    /// Teardown still funnels through [`Self::close`].
    pub fn take_reader(&mut self) -> Option<BufReader<File>> {
        self.reader.take()
    }

    /// `nativeTerminal.Output`: one validated agent stdout line.
    pub fn output_frame(reader: &mut BufReader<File>) -> Result<TerminalFrame, String> {
        let mut line = Vec::new();
        // `bufio.Scanner` with a 131072-byte token cap: overlong lines and
        // EOF both end the stream.
        let mut total = 0usize;
        loop {
            let chunk = reader
                .fill_buf()
                .map_err(|_| "terminal output ended".to_string())?;
            if chunk.is_empty() {
                return Err("terminal output ended".to_string());
            }
            let end = chunk.iter().position(|&b| b == b'\n');
            let take = match end {
                Some(i) => i + 1,
                None => chunk.len(),
            };
            total += take;
            if total > FRAME_LIMIT {
                return Err("terminal output ended".to_string());
            }
            line.extend_from_slice(&chunk[..take]);
            reader.consume(take);
            if end.is_some() {
                break;
            }
        }
        line.pop();
        // `ScanLines` strips one trailing `\r`.
        if line.last() == Some(&b'\r') {
            line.pop();
        }
        parse_output_line(&line)
    }

    /// `nativeTerminal.Close`: stdin EOF, 3s grace, then kill. Idempotent.
    pub fn close(&mut self) {
        if self.closed {
            return;
        }
        self.closed = true;
        drop(self.stdin.take());
        drop(self.reader.take());
        if let Some(mut child) = self.child.take() {
            let deadline = Instant::now() + Duration::from_secs(3);
            loop {
                match child.try_wait() {
                    Ok(Some(_)) | Err(_) => break,
                    Ok(None) => {
                        if Instant::now() >= deadline {
                            let _ = child.kill();
                            break;
                        }
                        std::thread::sleep(Duration::from_millis(10));
                    }
                }
            }
        }
    }
}

impl Drop for NativeAttach {
    fn drop(&mut self) {
        self.close();
    }
}

// ---------- private request admission + stream table (service.go) ----------

/// `validPrivateTerminalRequest` over pre-parsed request fields.
pub fn valid_private_terminal_request(
    method: &str,
    raw_query: &str,
    force_query: bool,
    raw_path: &str,
    origin_count: usize,
) -> bool {
    method == "GET"
        && raw_query.is_empty()
        && !force_query
        && raw_path.is_empty()
        && origin_count == 0
}

/// `Service` stream registry: at most [`STREAM_LIMIT`] live streams;
/// `Close` cancels pending and live streams alike.
pub struct StreamTable {
    streams: HashMap<u64, Arc<AtomicBool>>,
    closed: bool,
    next: u64,
}

impl StreamTable {
    pub fn new() -> Self {
        StreamTable {
            streams: HashMap::new(),
            closed: false,
            next: 0,
        }
    }

    /// `Service.register`: admit one stream unless closed or full.
    pub fn register(&mut self) -> Option<(u64, Arc<AtomicBool>)> {
        if self.closed || self.streams.len() >= STREAM_LIMIT {
            return None;
        }
        let id = self.next;
        self.next += 1;
        let cancel = Arc::new(AtomicBool::new(false));
        self.streams.insert(id, cancel.clone());
        Some((id, cancel))
    }

    pub fn unregister(&mut self, id: u64) {
        self.streams.remove(&id);
    }

    /// `Service.Close`: cancel every registered stream.
    pub fn close(&mut self) {
        self.closed = true;
        for cancel in self.streams.values() {
            cancel.store(true, Ordering::SeqCst);
        }
    }

    pub fn len(&self) -> usize {
        self.streams.len()
    }

    pub fn is_empty(&self) -> bool {
        self.streams.is_empty()
    }

    pub fn is_closed(&self) -> bool {
        self.closed
    }
}

impl Default for StreamTable {
    fn default() -> Self {
        Self::new()
    }
}

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

/// `validIdentityRequest` over pre-parsed request fields.
pub fn valid_identity_request(
    method: &str,
    raw_query: &str,
    force_query: bool,
    raw_path: &str,
    origin_count: usize,
) -> bool {
    method == "POST"
        && raw_query.is_empty()
        && !force_query
        && raw_path.is_empty()
        && origin_count == 0
}

/// Broker custody surface the identity routes need.
pub trait IdentityBroker {
    fn acquire(&self, req: &AcquireRequest, deadline: Instant) -> Result<Lease, String>;
    fn register(
        &self,
        lease_id: &str,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<Delivery, String>;
    fn reconcile_lease(&self, lease_id: &str) -> Result<(), String>;
}

/// 16 bytes of kernel randomness, hex-encoded (`museID` / launch IDs).
pub fn rand_id() -> Result<String, String> {
    let mut bytes = [0u8; 16];
    let mut filled = 0;
    while filled < bytes.len() {
        let n = unsafe {
            libc::getrandom(
                bytes[filled..].as_mut_ptr() as *mut libc::c_void,
                bytes.len() - filled,
                0,
            )
        };
        if n < 0 {
            let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
            if errno == libc::EINTR {
                continue;
            }
            return Err(format!("kernel randomness unavailable: errno {errno}"));
        }
        filled += n as usize;
    }
    Ok(sha256::hex_lower(&bytes))
}

/// `Daemon.identityLaunch`: acquire, prepare, register, start. Every
/// failure after acquire reconciles the lease (reconcile errors ignored).
pub fn identity_launch<B: IdentityBroker, E: Executor>(
    broker: &B,
    service: &Service<E>,
    input: &TerminalStart,
    codex_harness_configured: bool,
    deadline: Instant,
) -> Result<Lease, String> {
    if !codex_harness_configured {
        return Err(err_denied());
    }
    let execution_id = rand_id()?;
    let now = now_unix();
    let lease = broker.acquire(
        &AcquireRequest {
            provider_id: PROVIDER_CODEX.to_string(),
            actor_id: input.actor_id,
            connection_id: input.connection_id.clone(),
            project_id: input.project_id.clone(),
            execution_id,
            kind: KIND_TERMINAL.to_string(),
            deadline_secs: now + 12 * 3600,
            ..Default::default()
        },
        deadline,
    )?;
    let binding = match service.prepare_identity(
        &lease,
        &input.login,
        &input.scope,
        input.cols,
        input.rows,
        deadline,
    ) {
        Ok(binding) => binding,
        Err(err) => {
            let _ = broker.reconcile_lease(&lease.id);
            return Err(err);
        }
    };
    let mut delivery = match broker.register(&lease.id, &binding, deadline) {
        Ok(delivery) => delivery,
        Err(err) => {
            let _ = broker.reconcile_lease(&lease.id);
            return Err(err);
        }
    };
    let outcome = service.identity("start", &delivery, deadline);
    // Zero the credential copy however the call ends.
    if let Some(credential) = delivery.credential.as_mut() {
        for byte in credential.iter_mut() {
            *byte = 0;
        }
    }
    if let Err(err) = outcome {
        let _ = broker.reconcile_lease(&lease.id);
        return Err(err);
    }
    Ok(delivery.lease)
}

/// `Daemon.identityOperation` dispatch target.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum IdentityRoute {
    Launch,
    Factory,
    Muse,
    Terminal,
}

/// Classify one identity operation: launch path, factory kind, live
/// muse-project binding, or plain terminal operation.
pub fn identity_route(path: &str, lease: &Lease, muse_available: bool) -> IdentityRoute {
    if path == IDENTITY_LAUNCH_PATH {
        return IdentityRoute::Launch;
    }
    if lease.kind == KIND_FACTORY {
        return IdentityRoute::Factory;
    }
    let muse_bound = lease.provider_id == PROVIDER_MUSE
        && lease
            .binding
            .as_ref()
            .map(|b| b.scope == SCOPE_MUSE_PROJECT)
            .unwrap_or(false);
    if muse_bound && muse_available {
        IdentityRoute::Muse
    } else {
        IdentityRoute::Terminal
    }
}

/// Operation word carried after `/identity/` (`validate`, `stop`, ...).
pub fn identity_action(path: &str) -> &str {
    path.strip_prefix("/identity/").unwrap_or(path)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::AtomicU64;
    use std::sync::{Mutex, MutexGuard};

    const PID: &str = "p0123456789abcdef01234567";
    const TID: &str = "0123456789abcdef0123456789abcdef";
    const CID: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";

    static TMP_COUNTER: AtomicU64 = AtomicU64::new(0);
    static ENV_LOCK: Mutex<()> = Mutex::new(());

    fn deadline() -> Instant {
        Instant::now() + Duration::from_secs(30)
    }

    /// Unique short-lived scratch dir (no fixed `/tmp` paths).
    fn test_tmp(slug: &str) -> std::path::PathBuf {
        let n = TMP_COUNTER.fetch_add(1, Ordering::SeqCst);
        let dir = std::env::temp_dir().join(format!("t26-{}-{}-{slug}", std::process::id(), n));
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    type RecordedCall = (Vec<u8>, String, Vec<String>);

    struct FakeExec {
        calls: Mutex<Vec<RecordedCall>>,
        script: Mutex<Vec<Result<Vec<u8>, String>>>,
    }

    impl FakeExec {
        fn new(script: Vec<Result<Vec<u8>, String>>) -> Self {
            FakeExec {
                calls: Mutex::new(Vec::new()),
                script: Mutex::new(script),
            }
        }

        fn calls(&self) -> Vec<RecordedCall> {
            self.calls.lock().unwrap().clone()
        }

        fn argvs(&self) -> Vec<Vec<String>> {
            self.calls()
                .into_iter()
                .map(|(_, cmd, args)| {
                    let mut full = vec![cmd];
                    full.extend(args);
                    full
                })
                .collect()
        }
    }

    impl Executor for FakeExec {
        fn run(
            &self,
            stdin: &[u8],
            cmd: &str,
            args: &[&str],
            _deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            self.calls.lock().unwrap().push((
                stdin.to_vec(),
                cmd.to_string(),
                args.iter().map(|s| s.to_string()).collect(),
            ));
            let mut script = self.script.lock().unwrap();
            if script.is_empty() {
                return Err("unexpected call".to_string());
            }
            script.remove(0)
        }
    }

    fn ok(body: &str) -> Result<Vec<u8>, String> {
        Ok(body.as_bytes().to_vec())
    }

    fn inspect_json(
        id: &str,
        running: bool,
        project: &str,
        owner: &str,
        privileged: bool,
        userns: &str,
    ) -> String {
        format!(
            "{{\"id\":{id:?},\"running\":{running},\"project\":{project:?},\"owner\":{owner:?},\"privileged\":{privileged},\"userns\":{userns:?},\"mappings\":{{\"UidMap\":[\"0:100000:262144\"],\"GidMap\":[\"0:100000:262144\"]}}}}"
        )
    }

    // ----- predicates -----

    #[test]
    fn terminal_name_matrix() {
        assert!(valid_terminal_name(""));
        assert!(valid_terminal_name("plain name-1_2"));
        assert!(valid_terminal_name(&"x".repeat(80)));
        assert!(!valid_terminal_name(&"x".repeat(81)));
        assert!(valid_terminal_name(&"é".repeat(80)));
        assert!(!valid_terminal_name(&"é".repeat(81)));
        assert!(!valid_terminal_name("a\nb"));
        assert!(!valid_terminal_name("a\tb"));
        assert!(!valid_terminal_name("\u{0}"));
        assert!(!valid_terminal_name("\u{7f}"));
        // Cf format characters rejected.
        for c in [
            "\u{ad}",
            "\u{200b}",
            "\u{200f}",
            "\u{202a}",
            "\u{2060}",
            "\u{feff}",
            "\u{61c}",
            "\u{1d173}",
            "\u{e0001}",
        ] {
            assert!(!valid_terminal_name(c), "Cf {c:?} must be rejected");
        }
        // Nearby non-Cf characters accepted.
        for c in ["\u{a0}", "\u{2000}", "\u{2028}", "é", "中"] {
            assert!(valid_terminal_name(c), "{c:?} must be accepted");
        }
    }

    #[test]
    fn id_predicates() {
        assert!(valid_terminal_id(TID));
        assert!(!valid_terminal_id(&TID[..31]));
        assert!(!valid_terminal_id(&format!("{TID}0")));
        assert!(!valid_terminal_id("0123456789ABCDEF0123456789ABCDEF"));
        assert!(terminal_dimensions(2, 2));
        assert!(terminal_dimensions(500, 300));
        assert!(!terminal_dimensions(1, 2));
        assert!(!terminal_dimensions(501, 2));
        assert!(!terminal_dimensions(2, 301));
    }

    fn base_request() -> TerminalRequest {
        TerminalRequest {
            action: "create".to_string(),
            id: TID.to_string(),
            project: PID.to_string(),
            login: "dev".to_string(),
            identity: 7,
            cols: 80,
            rows: 24,
            expires: now_unix() + 60,
            name: "term".to_string(),
            scope: CID.to_string(),
        }
    }

    #[test]
    fn request_valid_matrix() {
        let now = now_unix();
        assert!(base_request().valid(now));
        // Actor faults.
        for broke in [
            TerminalRequest {
                project: "nope".to_string(),
                ..base_request()
            },
            TerminalRequest {
                login: "Root".to_string(),
                ..base_request()
            },
            TerminalRequest {
                login: "root".to_string(),
                ..base_request()
            },
            TerminalRequest {
                identity: 0,
                ..base_request()
            },
            TerminalRequest {
                name: "a\nb".to_string(),
                ..base_request()
            },
        ] {
            assert!(!broke.valid(now), "{broke:?}");
        }
        // Window edges.
        assert!(!TerminalRequest {
            expires: now,
            ..base_request()
        }
        .valid(now));
        assert!(TerminalRequest {
            expires: now + 12 * 3600,
            ..base_request()
        }
        .valid(now));
        assert!(!TerminalRequest {
            expires: now + 12 * 3600 + 1,
            ..base_request()
        }
        .valid(now));
        // Scope rules.
        assert!(!TerminalRequest {
            scope: String::new(),
            ..base_request()
        }
        .valid(now));
        assert!(!TerminalRequest {
            scope: TID.to_string(),
            ..base_request()
        }
        .valid(now));
        let mut attach = base_request();
        attach.action = "attach".to_string();
        attach.scope = String::new();
        attach.name = String::new();
        assert!(attach.valid(now));
        attach.scope = CID.to_string();
        assert!(!attach.valid(now));
        // List shape.
        let list = TerminalRequest {
            action: "list".to_string(),
            id: String::new(),
            project: PID.to_string(),
            login: "dev".to_string(),
            identity: 7,
            cols: 0,
            rows: 0,
            expires: now + 60,
            name: String::new(),
            scope: String::new(),
        };
        assert!(list.valid(now));
        assert!(!TerminalRequest {
            id: TID.to_string(),
            ..list.clone()
        }
        .valid(now));
        assert!(!TerminalRequest {
            cols: 80,
            ..list.clone()
        }
        .valid(now));
        assert!(!TerminalRequest {
            scope: CID.to_string(),
            ..list.clone()
        }
        .valid(now));
        // Idle actions.
        for action in ["inspect", "end", "rename"] {
            let mut idle = base_request();
            idle.action = action.to_string();
            idle.cols = 0;
            idle.rows = 0;
            idle.scope = String::new();
            if action != "rename" {
                idle.name = String::new();
            }
            assert!(idle.valid(now), "{action}");
            idle.cols = 80;
            assert!(!idle.valid(now), "{action} sized");
        }
        let mut rename = base_request();
        rename.action = "rename".to_string();
        rename.cols = 0;
        rename.rows = 0;
        rename.scope = String::new();
        assert!(rename.valid(now));
        // Unknown action and bad id.
        assert!(!TerminalRequest {
            action: "kill".to_string(),
            ..base_request()
        }
        .valid(now));
        assert!(!TerminalRequest {
            id: "short".to_string(),
            ..base_request()
        }
        .valid(now));
    }

    #[test]
    fn strict_b64_vectors() {
        // Probed against Go `Strict()`: trailing bits checked, newlines skipped.
        assert_eq!(strict_b64_decode("AB/C").unwrap(), vec![0x00, 0x1f, 0xc2]);
        assert_eq!(strict_b64_decode("AB/D").unwrap(), vec![0x00, 0x1f, 0xc3]);
        assert_eq!(strict_b64_decode("AQ==").unwrap(), vec![0x01]);
        assert!(strict_b64_decode("AR==").is_none());
        assert!(strict_b64_decode("AZ==").is_none());
        assert!(strict_b64_decode("A/==").is_none());
        assert!(strict_b64_decode("A+/=").is_none());
        assert!(strict_b64_decode("AB=C").is_none());
        assert!(strict_b64_decode("ABC").is_none());
        assert!(strict_b64_decode("ABCD====").is_none());
        assert!(strict_b64_decode("AB=CDEF").is_none());
        assert_eq!(strict_b64_decode("AB\nCD").unwrap(), vec![0x00, 0x10, 0x83]);
        assert_eq!(strict_b64_decode("").unwrap(), Vec::<u8>::new());
        assert!(strict_b64_decode("====").is_none());
        assert!(strict_b64_decode("AB").is_none());
    }

    #[test]
    fn frame_input_matrix() {
        let input = |data: &str| TerminalFrame {
            frame_type: "input".to_string(),
            data: data.to_string(),
            ..Default::default()
        };
        assert!(input("aGk=").input_valid());
        assert!(!input("").input_valid());
        assert!(!input("AR==").input_valid());
        assert!(!input("AB\nCD").input_valid());
        assert!(!input("AB\rCD").input_valid());
        assert!(input(&"QUFB".repeat(5461)).input_valid()); // 16383 bytes
        assert!(!input(&"QUFB".repeat(5462)).input_valid()); // 16386 bytes
        assert!(!TerminalFrame {
            cols: 1,
            ..input("aGk=")
        }
        .input_valid());
        assert!(!TerminalFrame {
            reason: "x".to_string(),
            ..input("aGk=")
        }
        .input_valid());
        assert!(!TerminalFrame {
            terminals: Some(vec![]),
            ..input("aGk=")
        }
        .input_valid());
        let resize = |cols, rows| TerminalFrame {
            frame_type: "resize".to_string(),
            cols,
            rows,
            ..Default::default()
        };
        assert!(resize(80, 24).input_valid());
        assert!(!resize(1, 24).input_valid());
        assert!(!TerminalFrame {
            data: "aGk=".to_string(),
            ..resize(80, 24)
        }
        .input_valid());
        for t in ["heartbeat", "close"] {
            assert!(TerminalFrame {
                frame_type: t.to_string(),
                ..Default::default()
            }
            .input_valid());
            assert!(!TerminalFrame {
                frame_type: t.to_string(),
                data: "aGk=".to_string(),
                ..Default::default()
            }
            .input_valid());
        }
        assert!(!TerminalFrame {
            frame_type: "output".to_string(),
            ..Default::default()
        }
        .input_valid());
        assert!(!TerminalFrame::default().input_valid());
    }

    #[test]
    fn frame_output_matrix() {
        let output = |data: &str| TerminalFrame {
            frame_type: "output".to_string(),
            data: data.to_string(),
            ..Default::default()
        };
        assert!(output("aGk=").output_valid());
        // Newlines accepted on the output path (Go `Strict` skips them).
        assert!(output("AB\nCD").output_valid());
        assert!(!output("").output_valid());
        assert!(!output("AR==").output_valid());
        assert!(output(&"QUFB".repeat(1365)).output_valid()); // 4095 bytes
        assert!(!output(&"QUFB".repeat(1366)).output_valid()); // 4098 bytes
        assert!(!TerminalFrame {
            reason: "x".to_string(),
            ..output("aGk=")
        }
        .output_valid());
        assert!(!TerminalFrame {
            cols: 1,
            ..output("aGk=")
        }
        .output_valid());
        assert!(TerminalFrame {
            frame_type: "ready".to_string(),
            ..Default::default()
        }
        .output_valid());
        assert!(!TerminalFrame {
            frame_type: "ready".to_string(),
            data: "aGk=".to_string(),
            ..Default::default()
        }
        .output_valid());
        for reason in [
            "disconnected",
            "expired",
            "exited",
            "launch_failed",
            "stream_failed",
            "cleanup_unconfirmed",
        ] {
            assert!(
                TerminalFrame {
                    frame_type: "closed".to_string(),
                    reason: reason.to_string(),
                    ..Default::default()
                }
                .output_valid(),
                "{reason}"
            );
        }
        assert!(!TerminalFrame {
            frame_type: "closed".to_string(),
            reason: "nope".to_string(),
            ..Default::default()
        }
        .output_valid());
        assert!(!TerminalFrame {
            frame_type: "closed".to_string(),
            ..Default::default()
        }
        .output_valid());
        assert!(!TerminalFrame {
            frame_type: "input".to_string(),
            ..Default::default()
        }
        .output_valid());
    }

    fn meta_state(id: &str, ready: bool, attached: bool, state: &str) -> TerminalState {
        TerminalState {
            id: id.to_string(),
            name: "t".to_string(),
            created_at: 1700000000,
            ready,
            attached,
            state: state.to_string(),
        }
    }

    #[test]
    fn metadata_output_matrix() {
        let meta = |items: Vec<TerminalState>| TerminalFrame {
            frame_type: "metadata".to_string(),
            terminals: Some(items),
            ..Default::default()
        };
        assert!(meta(vec![meta_state(TID, true, true, "ready")]).output_valid());
        assert!(meta(vec![meta_state(TID, false, false, "opening")]).output_valid());
        assert!(meta(vec![]).output_valid());
        assert!(!TerminalFrame {
            frame_type: "metadata".to_string(),
            ..Default::default()
        }
        .output_valid());
        assert!(!TerminalFrame {
            frame_type: "metadata".to_string(),
            data: "aGk=".to_string(),
            ..Default::default()
        }
        .output_valid());
        // Shape: terminals forbidden off metadata.
        assert!(!TerminalFrame {
            frame_type: "ready".to_string(),
            terminals: Some(vec![]),
            ..Default::default()
        }
        .output_valid());
        // Item faults.
        assert!(!meta(vec![meta_state("short", true, true, "ready")]).output_valid());
        assert!(!meta(vec![
            meta_state(TID, true, true, "ready"),
            meta_state(TID, true, true, "ready")
        ])
        .output_valid());
        assert!(!meta(vec![TerminalState {
            name: "x\ny".to_string(),
            ..meta_state(TID, true, true, "ready")
        }])
        .output_valid());
        assert!(!meta(vec![TerminalState {
            created_at: 0,
            ..meta_state(TID, true, true, "ready")
        }])
        .output_valid());
        assert!(!meta(vec![TerminalState {
            created_at: 9007199254740992,
            ..meta_state(TID, true, true, "ready")
        }])
        .output_valid());
        assert!(meta(vec![TerminalState {
            created_at: 9007199254740991,
            ..meta_state(TID, true, true, "ready")
        }])
        .output_valid());
        assert!(!meta(vec![meta_state(TID, false, true, "ready")]).output_valid());
        assert!(!meta(vec![meta_state(TID, true, false, "opening")]).output_valid());
        assert!(!meta(vec![meta_state(TID, false, false, "bogus")]).output_valid());
        // 64-row cap.
        let many: Vec<TerminalState> = (0..65)
            .map(|i| meta_state(&format!("{i:032x}"), true, false, "ready"))
            .collect();
        assert!(!meta(many).output_valid());
        let edge: Vec<TerminalState> = (0..64)
            .map(|i| meta_state(&format!("{i:032x}"), true, false, "ready"))
            .collect();
        assert!(meta(edge).output_valid());
    }

    // ----- wire codec goldens (byte-exact vs Go `encoding/json`) -----

    #[test]
    fn frame_encode_goldens() {
        let cases: Vec<(TerminalFrame, &str)> = vec![
            (
                TerminalFrame {
                    frame_type: "input".to_string(),
                    data: "aGk=".to_string(),
                    ..Default::default()
                },
                r#"{"type":"input","data":"aGk="}"#,
            ),
            (
                TerminalFrame {
                    frame_type: "resize".to_string(),
                    cols: 80,
                    rows: 24,
                    ..Default::default()
                },
                r#"{"type":"resize","cols":80,"rows":24}"#,
            ),
            (
                TerminalFrame {
                    frame_type: "close".to_string(),
                    ..Default::default()
                },
                r#"{"type":"close"}"#,
            ),
            (
                TerminalFrame {
                    frame_type: "output".to_string(),
                    data: "aGkK".to_string(),
                    ..Default::default()
                },
                r#"{"type":"output","data":"aGkK"}"#,
            ),
            (
                TerminalFrame {
                    frame_type: "closed".to_string(),
                    reason: "exited".to_string(),
                    ..Default::default()
                },
                r#"{"type":"closed","reason":"exited"}"#,
            ),
            (
                TerminalFrame {
                    frame_type: "ready".to_string(),
                    ..Default::default()
                },
                r#"{"type":"ready"}"#,
            ),
            (
                TerminalFrame {
                    frame_type: "metadata".to_string(),
                    terminals: Some(vec![
                        TerminalState {
                            id: TID.to_string(),
                            name: "a<b>&\"c".to_string(),
                            created_at: 1700000000,
                            ready: true,
                            attached: true,
                            state: "ready".to_string(),
                        },
                        TerminalState {
                            id: "fedcba9876543210fedcba9876543210".to_string(),
                            created_at: 1,
                            state: "opening".to_string(),
                            ..Default::default()
                        },
                    ]),
                    ..Default::default()
                },
                r#"{"type":"metadata","terminals":[{"id":"0123456789abcdef0123456789abcdef","name":"a\u003cb\u003e\u0026\"c","created_at":1700000000,"ready":true,"attached":true,"state":"ready"},{"id":"fedcba9876543210fedcba9876543210","name":"","created_at":1,"ready":false,"attached":false,"state":"opening"}]}"#,
            ),
        ];
        for (frame, want) in cases {
            assert_eq!(frame.encode(), want, "{frame:?}");
            assert_eq!(
                TerminalFrame::decode(frame.encode().as_bytes()).unwrap(),
                frame
            );
        }
    }

    #[test]
    fn request_encode_golden() {
        let req = TerminalRequest {
            action: "create".to_string(),
            id: TID.to_string(),
            project: PID.to_string(),
            login: "dev".to_string(),
            identity: 7,
            cols: 80,
            rows: 24,
            expires: 1900000000,
            name: "n".to_string(),
            scope: CID.to_string(),
        };
        assert_eq!(
            req.encode(),
            r#"{"action":"create","id":"0123456789abcdef0123456789abcdef","project":"p0123456789abcdef01234567","login":"dev","identity":7,"cols":80,"rows":24,"expires":1900000000,"name":"n","scope":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}"#
        );
        assert_eq!(
            TerminalRequest::decode(req.encode().as_bytes()).unwrap(),
            req
        );
    }

    fn golden_binding() -> Binding {
        Binding {
            child_id: "f0123456789abcdef01234567".to_string(),
            uid: 1000,
            gid: 1000,
            scope: "muse-project".to_string(),
            credential_root: "/run/soda-muse/abcd".to_string(),
            invocation_id: TID.to_string(),
            kind: "terminal".to_string(),
            id: "abcd".to_string(),
            project: CID.to_string(),
            login: "dev".to_string(),
            generation: 3,
        }
    }

    fn golden_lease() -> Lease {
        Lease {
            repository_id: 9,
            provider_id: "muse".to_string(),
            id: "lease-1".to_string(),
            connection_id: "conn-1".to_string(),
            generation: 3,
            actor_id: 7,
            project_id: PID.to_string(),
            execution_id: TID.to_string(),
            kind: "terminal".to_string(),
            role: "r".to_string(),
            deadline_raw: "2026-10-04T12:00:00.123456789Z".to_string(),
            deadline: parse_rfc3339("2026-10-04T12:00:00.123456789Z"),
            grant_id: "g".to_string(),
            grant_revision: 2,
            binding: Some(golden_binding()),
        }
    }

    #[test]
    fn identity_encode_goldens() {
        assert_eq!(
            golden_binding().encode(),
            r#"{"child_id":"f0123456789abcdef01234567","uid":1000,"gid":1000,"scope":"muse-project","credential_root":"/run/soda-muse/abcd","invocation_id":"0123456789abcdef0123456789abcdef","kind":"terminal","id":"abcd","project":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","login":"dev","generation":3}"#
        );
        let lease_golden = r#"{"repository_id":"9","provider_id":"muse","id":"lease-1","connection_id":"conn-1","generation":3,"actor_id":"7","project_id":"p0123456789abcdef01234567","execution_id":"0123456789abcdef0123456789abcdef","kind":"terminal","role":"r","deadline":"2026-10-04T12:00:00.123456789Z","grant_id":"g","grant_revision":2,"binding":{"child_id":"f0123456789abcdef01234567","uid":1000,"gid":1000,"scope":"muse-project","credential_root":"/run/soda-muse/abcd","invocation_id":"0123456789abcdef0123456789abcdef","kind":"terminal","id":"abcd","project":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","login":"dev","generation":3}}"#;
        assert_eq!(golden_lease().encode(), lease_golden);
        for (credential, suffix) in [
            (None, "null"),
            (Some(vec![]), r#""""#),
            (Some(b"{\"a\":1}".to_vec()), r#""eyJhIjoxfQ==""#),
        ] {
            let delivery = Delivery {
                lease: golden_lease(),
                credential,
            };
            assert_eq!(
                delivery.encode(),
                format!("{{\"lease\":{lease_golden},\"credential\":{suffix}}}")
            );
            assert_eq!(
                Delivery::decode(delivery.encode().as_bytes()).unwrap(),
                delivery
            );
        }
        // Lean lease: omitempty drops zero fields.
        let lean = Lease {
            provider_id: "codex".to_string(),
            id: "l".to_string(),
            connection_id: "c".to_string(),
            generation: 1,
            actor_id: 1,
            project_id: PID.to_string(),
            execution_id: TID.to_string(),
            kind: "terminal".to_string(),
            deadline_raw: "2026-01-01T00:00:00Z".to_string(),
            deadline: parse_rfc3339("2026-01-01T00:00:00Z"),
            ..Default::default()
        };
        assert_eq!(
            lean.encode(),
            r#"{"provider_id":"codex","id":"l","connection_id":"c","generation":1,"actor_id":"1","project_id":"p0123456789abcdef01234567","execution_id":"0123456789abcdef0123456789abcdef","kind":"terminal","deadline":"2026-01-01T00:00:00Z"}"#
        );
        assert_eq!(
            Delivery::decode(
                format!("{{\"lease\":{},\"credential\":null}}", lean.encode()).as_bytes()
            )
            .unwrap()
            .lease,
            lean
        );
    }

    #[test]
    fn strict_decode_matrix() {
        assert!(TerminalRequest::decode(br#"{"action":"list","bogus":1}"#).is_err());
        assert!(TerminalFrame::decode(br#"{"type":"close","bogus":1}"#).is_err());
        assert!(TerminalFrame::decode(br#"{"type":"close","terminals":"x"}"#).is_err());
        assert!(TerminalFrame::decode(br#"{"type":"close","terminals":[7]}"#).is_err());
        // terminals tri-state: missing/null -> None, [] -> Some(empty).
        assert_eq!(
            TerminalFrame::decode(br#"{"type":"close"}"#)
                .unwrap()
                .terminals,
            None
        );
        assert_eq!(
            TerminalFrame::decode(br#"{"type":"close","terminals":null}"#)
                .unwrap()
                .terminals,
            None
        );
        assert_eq!(
            TerminalFrame::decode(br#"{"type":"metadata","terminals":[]}"#)
                .unwrap()
                .terminals,
            Some(vec![])
        );
        // Folded duplicate terminals keys are an unknown field.
        assert!(
            TerminalFrame::decode(br#"{"type":"close","terminals":null,"Terminals":null}"#)
                .is_err()
        );
        // Null items decode to zero states.
        let f = TerminalFrame::decode(br#"{"type":"metadata","terminals":[null]}"#).unwrap();
        assert_eq!(f.terminals, Some(vec![TerminalState::default()]));
        // Trailing data rejected.
        assert!(TerminalFrame::decode(br#"{"type":"close"} {}"#).is_err());
        // Delivery credential shapes.
        assert_eq!(
            Delivery::decode(br#"{"lease":{"provider_id":"x"},"credential":"aGk="}"#)
                .unwrap()
                .credential,
            Some(b"hi".to_vec())
        );
        assert_eq!(
            Delivery::decode(br#"{"lease":{"provider_id":"x"},"credential":[104,105]}"#)
                .unwrap()
                .credential,
            Some(b"hi".to_vec())
        );
        assert_eq!(
            Delivery::decode(br#"{"lease":{"provider_id":"x"},"credential":null}"#)
                .unwrap()
                .credential,
            None
        );
        assert_eq!(
            Delivery::decode(br#"{"lease":{"provider_id":"x"},"credential":""}"#)
                .unwrap()
                .credential,
            Some(vec![])
        );
        assert_eq!(
            Delivery::decode(br#"{"lease":{"provider_id":"x"}}"#)
                .unwrap()
                .credential,
            None
        );
        assert!(Delivery::decode(br#"{"lease":{},"credential":"!!!"}"#).is_err());
        // ,string edges.
        assert!(Delivery::decode(br#"{"lease":{"actor_id":5}}"#).is_err());
        assert!(Delivery::decode(br#"{"lease":{"actor_id":""}}"#).is_err());
        assert!(Delivery::decode(br#"{"lease":{"deadline":"not-a-time"}}"#).is_err());
        let d =
            Delivery::decode(br#"{"lease":{"actor_id":"007","deadline":"2026-01-01T00:00:00Z"}}"#)
                .unwrap();
        assert_eq!(d.lease.actor_id, 7);
    }

    #[test]
    fn credential_valid_vectors() {
        // Probed against Go `json.Valid`.
        for (body, want) in [
            ("01", false),
            (" 1 ", true),
            ("", false),
            ("null", true),
            ("{},", false),
            ("{}", true),
            ("[1,2]", true),
            ("\"a\"", true),
            ("1e3", true),
            ("-0", true),
            ("0.5", true),
            ("--1", false),
            ("{\"a\":1}", true),
            ("[", false),
        ] {
            assert_eq!(credential_valid(body.as_bytes()), want, "{body:?}");
        }
        assert!(!credential_valid(&vec![b'{'; CREDENTIAL_LIMIT + 1]));
        assert!(!credential_valid(&vec![b' '; CREDENTIAL_LIMIT - 2])); // whitespace is not a value
        assert!(json_valid(b"{}"));
    }

    #[test]
    fn string_i64_vectors() {
        for (raw, want) in [
            ("5", Some(5)),
            ("-5", Some(-5)),
            ("05", Some(5)),
            ("00", Some(0)),
            ("-0", Some(0)),
            ("0", Some(0)),
            ("9223372036854775807", Some(i64::MAX)),
            ("-9223372036854775808", Some(i64::MIN)),
        ] {
            assert_eq!(parse_string_i64(raw), want, "{raw:?}");
        }
        for raw in [
            "",
            " 5",
            "+5",
            "5 ",
            "5.0",
            "5e1",
            "0x5",
            "9223372036854775808",
            "-9223372036854775809",
            "-",
            "５",
        ] {
            assert!(parse_string_i64(raw).is_none(), "{raw:?}");
        }
    }

    #[test]
    fn rfc3339_vectors() {
        assert_eq!(parse_rfc3339("1970-01-01T00:00:00Z"), Some((0, 0)));
        assert_eq!(parse_rfc3339("2026-01-01T00:00:00Z"), Some((1767225600, 0)));
        assert_eq!(
            parse_rfc3339("2026-10-04T12:00:00.123456789Z").unwrap().1,
            123456789
        );
        assert_eq!(
            parse_rfc3339("2026-10-04T12:00:00.5Z").unwrap().1,
            500000000
        );
        let a = parse_rfc3339("2026-10-04T12:00:00Z").unwrap();
        assert_eq!(
            parse_rfc3339("2026-10-04T14:00:00+02:00").unwrap(),
            (a.0, 0)
        );
        assert_eq!(parse_rfc3339("2026-10-04T07:00:00-0500").unwrap(), (a.0, 0));
        assert_eq!(
            parse_rfc3339("2026-10-04T07:00:00-05:00").unwrap(),
            (a.0, 0)
        );
        assert_eq!(parse_rfc3339("2026-10-04T07:00:00-05").unwrap(), (a.0, 0));
        assert_eq!(parse_rfc3339("2024-02-29T00:00:00Z").unwrap().0, 1709164800);
        for bad in [
            "",
            "not-a-time",
            "2026-13-01T00:00:00Z",
            "2026-00-01T00:00:00Z",
            "2026-02-30T00:00:00Z",
            "2025-02-29T00:00:00Z",
            "2026-01-01T24:00:00Z",
            "2026-01-01T00:60:00Z",
            "2026-01-01T00:00:61Z",
            "2026-01-01T00:00:00",
            "2026-01-01T00:00:00.",
            "2026-01-01T00:00:00.Z",
            "2026-01-01t00:00:00Z",
            "2026-01-01T00:00:00z",
            "2026-01-01T00:00:00+25:00",
            "2026-01-01T00:00:00+02:60",
            "2026-01-01T00:00:00.1234567890Z",
        ] {
            assert_eq!(parse_rfc3339(bad), None, "{bad:?}");
        }
    }

    // ----- inspection + argv -----

    #[test]
    fn id_map_matrix() {
        let yes = |s: &str| terminal_id_map(&[s.to_string()]);
        assert!(yes("0:100000:262144"));
        assert!(yes("0:1:262144"));
        assert!(yes("0:4294705151:262144")); // max base: 4294967295-262144
        assert!(!yes("0:4294705152:262144"));
        assert!(!yes("0:0:262144"));
        assert!(!yes("0:0100000:262144"));
        assert!(!yes("1:100000:262144"));
        assert!(!yes("0:100000:262145"));
        assert!(!yes("0:100000"));
        assert!(!yes("0:100000:262144:extra"));
        assert!(!yes("0:+100000:262144"));
        assert!(!yes("0:4294967296:262144"));
        assert!(!terminal_id_map(&[]));
        assert!(!terminal_id_map(&[
            "0:1:262144".to_string(),
            "0:1:262144".to_string()
        ]));
    }

    fn inspection() -> TerminalInspection {
        TerminalInspection {
            id: CID.to_string(),
            running: true,
            project: PID.to_string(),
            owner: "7".to_string(),
            privileged: false,
            userns: "private".to_string(),
            uid_map: vec!["0:100000:262144".to_string()],
            gid_map: vec!["0:100000:262144".to_string()],
        }
    }

    #[test]
    fn isolation_matrix() {
        assert!(terminal_isolation(&inspection(), PID));
        assert!(!terminal_isolation(
            &TerminalInspection {
                id: TID.to_string(),
                ..inspection()
            },
            PID
        ));
        assert!(!terminal_isolation(
            &TerminalInspection {
                project: "pffffffffffffffffffffffff".to_string(),
                ..inspection()
            },
            PID
        ));
        assert!(!terminal_isolation(
            &TerminalInspection {
                privileged: true,
                ..inspection()
            },
            PID
        ));
        assert!(!terminal_isolation(
            &TerminalInspection {
                userns: "host".to_string(),
                ..inspection()
            },
            PID
        ));
        assert!(!terminal_isolation(
            &TerminalInspection {
                gid_map: vec![],
                ..inspection()
            },
            PID
        ));
        assert!(terminal_target_ready(&inspection(), PID, true));
        assert!(terminal_target_ready(
            &TerminalInspection {
                running: false,
                ..inspection()
            },
            PID,
            false
        ));
        assert!(!terminal_target_ready(
            &TerminalInspection {
                running: false,
                ..inspection()
            },
            PID,
            true
        ));
        assert!(!terminal_target_ready(
            &TerminalInspection {
                owner: "0".to_string(),
                ..inspection()
            },
            PID,
            true
        ));
        assert!(!terminal_target_ready(
            &TerminalInspection {
                owner: "-3".to_string(),
                ..inspection()
            },
            PID,
            true
        ));
        assert!(!terminal_target_ready(
            &TerminalInspection {
                owner: "no".to_string(),
                ..inspection()
            },
            PID,
            true
        ));
        assert!(terminal_target_ready(
            &TerminalInspection {
                owner: "+7".to_string(),
                ..inspection()
            },
            PID,
            true
        ));
        assert!(terminal_target_ready(
            &TerminalInspection {
                owner: "007".to_string(),
                ..inspection()
            },
            PID,
            true
        ));
        // Strict decode pins.
        assert!(TerminalInspection::decode(br#"{"id":"x","unknown":1}"#).is_err());
        let v = TerminalInspection::decode(
            inspect_json(CID, true, PID, "7", false, "private").as_bytes(),
        )
        .unwrap();
        assert_eq!(v, inspection());
    }

    #[test]
    fn exit_code_pins() {
        assert_eq!(exit_code_of("exit status 1"), Some(1));
        assert_eq!(
            exit_code_of("/usr/bin/podman failed: exit status 32: boom"),
            Some(32)
        );
        assert_eq!(exit_code_of("exit status 0"), Some(0));
        assert_eq!(exit_code_of("signal: killed"), None);
        assert_eq!(exit_code_of("exit status "), None);
        assert_eq!(exit_code_of(""), None);
    }

    #[test]
    fn argv_vectors() {
        assert_eq!(
            inspect_argv("soda-abc"),
            vec![
                "--remote=false",
                "inspect",
                "--format",
                TERMINAL_INSPECT,
                "soda-abc"
            ]
        );
        assert_eq!(
            container_exists_argv(CID),
            vec!["--remote=false", "container", "exists", CID]
        );
        assert_eq!(
            agent_argv(CID, &["broker"]),
            vec![
                "--remote=false",
                "exec",
                "--interactive",
                CID,
                "/usr/libexec/soda/project-terminal",
                "broker"
            ]
        );
        // Current native attach argv: python is gone, fixed agent binary.
        assert_eq!(
            native_argv(CID, "create", TID, "dev", 7, 80, 24, 3600, "hash", "n", "scope"),
            vec![
                "--remote=false",
                "exec",
                "--interactive",
                CID,
                "/usr/libexec/soda/project-terminal",
                "create",
                TID,
                "dev",
                "7",
                "80",
                "24",
                "3600",
                "hash",
                "n",
                "scope",
            ]
        );
        assert_eq!(
            tar_producer_argv("/opt/harness//x/../"),
            vec!["--create", "--file=-", "--directory", "/opt/harness", "."]
        );
        assert_eq!(
            tar_consumer_argv(CID, "/run/x"),
            vec![
                "--remote=false",
                "exec",
                "--interactive",
                CID,
                "/usr/bin/tar",
                "--extract",
                "--file=-",
                "--directory",
                "/run/x",
                "--no-same-owner",
                "--same-permissions",
            ]
        );
    }

    #[test]
    fn clean_path_vectors() {
        for (input, want) in [
            ("/a/b/c", "/a/b/c"),
            ("/a//b/./c/", "/a/b/c"),
            ("/a/b/../c", "/a/c"),
            ("/../a", "/a"),
            ("", "."),
            (".", "."),
            ("a/../../b", "../b"),
            ("/", "/"),
            ("a/b/", "a/b"),
            ("/opt/harness//x/../", "/opt/harness"),
        ] {
            assert_eq!(clean_path(input), want, "{input:?}");
        }
    }

    // ----- service exec flows -----

    fn make_service(exec: FakeExec) -> Service<FakeExec> {
        Service {
            exec,
            codex_harness: "/opt/harness".to_string(),
            codex_harness_sha256:
                "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855".to_string(),
            codex_harness_version: "1.0".to_string(),
            muse_harness: String::new(),
            muse_harness_sha256: String::new(),
            muse_harness_version: String::new(),
        }
    }

    #[test]
    fn project_container_flows() {
        // Invalid project: no exec call.
        let exec = FakeExec::new(vec![]);
        let svc = make_service(exec);
        assert_eq!(
            svc.project_container("nope", true, deadline()).unwrap_err(),
            "invalid project"
        );
        // Exec failure and oversize.
        let exec = FakeExec::new(vec![Err("boom".to_string())]);
        let svc = make_service(exec);
        assert_eq!(
            svc.project_container(PID, true, deadline()).unwrap_err(),
            "terminal inspection unavailable"
        );
        let exec = FakeExec::new(vec![Ok(vec![b'x'; 4097])]);
        let svc = make_service(exec);
        assert_eq!(
            svc.project_container(PID, true, deadline()).unwrap_err(),
            "terminal inspection unavailable"
        );
        // Bad JSON and not-ready target.
        let exec = FakeExec::new(vec![ok("{nope")]);
        let svc = make_service(exec);
        assert_eq!(
            svc.project_container(PID, true, deadline()).unwrap_err(),
            "invalid terminal inspection"
        );
        let exec = FakeExec::new(vec![ok(&inspect_json(
            CID, false, PID, "7", false, "private",
        ))]);
        let svc = make_service(exec);
        assert_eq!(
            svc.project_container(PID, true, deadline()).unwrap_err(),
            "terminal target not ready or isolated"
        );
        // Success pins argv and container id.
        let exec = FakeExec::new(vec![ok(&inspect_json(
            CID, true, PID, "7", false, "private",
        ))]);
        let svc = make_service(exec);
        assert_eq!(svc.project_container(PID, true, deadline()).unwrap(), CID);
        assert_eq!(
            svc.exec.argvs(),
            vec![{
                let mut v = vec!["/usr/bin/podman".to_string()];
                v.extend(inspect_argv(&format!("soda-{PID}")));
                v
            }]
        );
        assert!(svc.exec.calls()[0].0.is_empty());
        // Stopped container admitted when running is not required.
        let exec = FakeExec::new(vec![ok(&inspect_json(
            CID, false, PID, "7", false, "private",
        ))]);
        let svc = make_service(exec);
        assert_eq!(svc.project_container(PID, false, deadline()).unwrap(), CID);
    }

    #[test]
    fn factory_project_container_flows() {
        let exec = FakeExec::new(vec![]);
        let svc = make_service(exec);
        assert_eq!(
            svc.factory_project_container("nope", true, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        let exec = FakeExec::new(vec![Err("boom".to_string())]);
        let svc = make_service(exec);
        assert_eq!(
            svc.factory_project_container(PID, true, deadline())
                .unwrap_err(),
            ERR_STALE
        );
        let exec = FakeExec::new(vec![ok("")]);
        let svc = make_service(exec);
        assert_eq!(
            svc.factory_project_container(PID, true, deadline())
                .unwrap_err(),
            ERR_STALE
        );
        let exec = FakeExec::new(vec![ok("{nope")]);
        let svc = make_service(exec);
        assert_eq!(
            svc.factory_project_container(PID, true, deadline())
                .unwrap_err(),
            ERR_STALE
        );
        let exec = FakeExec::new(vec![ok(&inspect_json(
            CID, true, PID, "7", true, "private",
        ))]);
        let svc = make_service(exec);
        assert_eq!(
            svc.factory_project_container(PID, true, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        let exec = FakeExec::new(vec![ok(&inspect_json(
            CID, true, PID, "0", false, "private",
        ))]);
        let svc = make_service(exec);
        assert_eq!(
            svc.factory_project_container(PID, true, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        let exec = FakeExec::new(vec![ok(&inspect_json(
            CID, false, PID, "7", false, "private",
        ))]);
        let svc = make_service(exec);
        assert_eq!(
            svc.factory_project_container(PID, true, deadline())
                .unwrap_err(),
            ERR_STALE
        );
        let exec = FakeExec::new(vec![ok(&inspect_json(
            CID, true, PID, "7", false, "private",
        ))]);
        let svc = make_service(exec);
        assert_eq!(
            svc.factory_project_container(PID, true, deadline())
                .unwrap(),
            CID
        );
    }

    fn live_lease() -> Lease {
        Lease {
            provider_id: "codex".to_string(),
            id: "lease-1".to_string(),
            connection_id: "conn".to_string(),
            generation: 2,
            actor_id: 7,
            project_id: PID.to_string(),
            execution_id: TID.to_string(),
            kind: KIND_TERMINAL.to_string(),
            deadline_raw: "2026-10-05T00:00:00Z".to_string(),
            deadline: parse_rfc3339("2026-10-05T00:00:00Z"),
            binding: Some(Binding {
                kind: KIND_TERMINAL.to_string(),
                id: TID.to_string(),
                project: CID.to_string(),
                login: "dev".to_string(),
                generation: 2,
                ..Default::default()
            }),
            ..Default::default()
        }
    }

    #[test]
    fn identity_call_flows() {
        let req = IdentityRequest {
            action: "validate".to_string(),
            ..Default::default()
        };
        let exec = FakeExec::new(vec![Err("boom".to_string())]);
        let svc = make_service(exec);
        assert_eq!(
            svc.identity_call(CID, &req, deadline()).unwrap_err(),
            "managed Codex operation failed"
        );
        let exec = FakeExec::new(vec![Ok(vec![b'x'; BROKER_RESPONSE_LIMIT + 1])]);
        let svc = make_service(exec);
        assert_eq!(
            svc.identity_call(CID, &req, deadline()).unwrap_err(),
            "invalid managed Codex response"
        );
        let exec = FakeExec::new(vec![ok("{nope")]);
        let svc = make_service(exec);
        assert_eq!(
            svc.identity_call(CID, &req, deadline()).unwrap_err(),
            "invalid managed Codex response"
        );
        // Success pins stdin bytes and broker argv.
        let body = Delivery {
            lease: live_lease(),
            credential: None,
        }
        .encode();
        let exec = FakeExec::new(vec![ok(&body)]);
        let svc = make_service(exec);
        let out = svc.identity_call(CID, &req, deadline()).unwrap();
        assert_eq!(out.lease, live_lease());
        let calls = svc.exec.calls();
        assert_eq!(calls[0].0, req.encode().as_bytes());
        let mut want = vec!["/usr/bin/podman".to_string()];
        want.extend(agent_argv(CID, &["broker"]));
        assert_eq!(svc.exec.argvs(), vec![want]);
    }

    #[test]
    fn terminal_lease_matrix() {
        let now = now_unix();
        let mut lease = live_lease();
        assert!(terminal_lease(&lease, false, now));
        lease.binding = None;
        assert!(!terminal_lease(&lease, false, now));
        lease.deadline_raw = "2020-01-01T00:00:00Z".to_string();
        lease.deadline = parse_rfc3339(&lease.deadline_raw);
        assert!(!terminal_lease(&lease, true, now)); // not after now
        lease.deadline = Some((now + 3600, 0));
        assert!(terminal_lease(&lease, true, now)); // live window
        lease.deadline = Some((now + 12 * 3600 + 1, 0));
        assert!(!terminal_lease(&lease, true, now)); // beyond 12h
        lease.deadline = Some((now, 0));
        assert!(!terminal_lease(&lease, true, now)); // not strictly after now
        lease = live_lease();
        lease.execution_id = "short".to_string();
        assert!(!terminal_lease(&lease, false, now));
        lease = live_lease();
        lease.binding.as_mut().unwrap().generation = 99;
        assert!(!terminal_lease(&lease, false, now));
    }

    #[test]
    fn identity_result_matrix() {
        let delivery = Delivery {
            lease: live_lease(),
            credential: None,
        };
        assert_eq!(
            identity_result("validate", &delivery, &delivery).unwrap(),
            delivery
        );
        let mut other = delivery.clone();
        other.lease.id = "other".to_string();
        assert_eq!(
            identity_result("validate", &delivery, &other).unwrap_err(),
            ERR_STALE
        );
        other = delivery.clone();
        other.lease.binding = None;
        assert_eq!(
            identity_result("validate", &delivery, &other).unwrap_err(),
            ERR_STALE
        );
        other = delivery.clone();
        other.lease.binding.as_mut().unwrap().login = "mallory".to_string();
        assert_eq!(
            identity_result("validate", &delivery, &other).unwrap_err(),
            ERR_STALE
        );
        other = delivery.clone();
        other.credential = Some(b"{}".to_vec());
        assert_eq!(
            identity_result("validate", &delivery, &other).unwrap_err(),
            "unexpected credential response"
        );
        assert_eq!(identity_result("finish", &delivery, &other).unwrap(), other);
        other = delivery.clone();
        other.credential = Some(b"{}".to_vec());
        let mut bad = other.clone();
        bad.credential = Some(b"nope".to_vec());
        assert_eq!(
            identity_result("finish", &delivery, &bad).unwrap_err(),
            ERR_UNCERTAIN
        );
        assert_eq!(
            identity_result("finish", &delivery, &delivery).unwrap_err(),
            ERR_UNCERTAIN
        );
    }

    #[test]
    fn identity_dispatch_matrix() {
        // Denied lease and denied action: no exec calls.
        let exec = FakeExec::new(vec![]);
        let svc = make_service(exec);
        let bad = Delivery::default();
        assert_eq!(
            svc.identity("validate", &bad, deadline()).unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.identity(
                "bogus",
                &Delivery {
                    lease: live_lease(),
                    credential: None
                },
                deadline()
            )
            .unwrap_err(),
            ERR_DENIED
        );
        let with_cred = Delivery {
            lease: live_lease(),
            credential: Some(b"{}".to_vec()),
        };
        assert_eq!(
            svc.identity("validate", &with_cred, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
        // Validate round-trip: inspect + broker call.
        let body = Delivery {
            lease: live_lease(),
            credential: None,
        }
        .encode();
        let exec = FakeExec::new(vec![
            ok(&inspect_json(CID, true, PID, "7", false, "private")),
            ok(&body),
        ]);
        let svc = make_service(exec);
        let out = svc
            .identity(
                "validate",
                &Delivery {
                    lease: live_lease(),
                    credential: None,
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(out.lease, live_lease());
        assert_eq!(svc.exec.calls().len(), 2);
        // Stale container incarnation.
        let exec = FakeExec::new(vec![ok(&inspect_json(
            "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
            true,
            PID,
            "7",
            false,
            "private",
        ))]);
        let svc = make_service(exec);
        assert_eq!(
            svc.identity(
                "validate",
                &Delivery {
                    lease: live_lease(),
                    credential: None
                },
                deadline()
            )
            .unwrap_err(),
            ERR_STALE
        );
        // Stop on a removed container short-circuits without a broker call.
        let exec = FakeExec::new(vec![Err("exit status 1".to_string())]);
        let svc = make_service(exec);
        let out = svc
            .identity(
                "stop",
                &Delivery {
                    lease: live_lease(),
                    credential: None,
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(out.lease, live_lease());
        assert_eq!(svc.exec.calls().len(), 1);
        // Stop on a stopped container also short-circuits.
        let exec = FakeExec::new(vec![
            ok(""),
            ok(&inspect_json(CID, false, PID, "7", false, "private")),
        ]);
        let svc = make_service(exec);
        let out = svc
            .identity(
                "stop",
                &Delivery {
                    lease: live_lease(),
                    credential: None,
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(out.lease, live_lease());
        // Uncertain stop target.
        let exec = FakeExec::new(vec![Err("exit status 2".to_string())]);
        let svc = make_service(exec);
        assert_eq!(
            svc.identity(
                "stop",
                &Delivery {
                    lease: live_lease(),
                    credential: None
                },
                deadline()
            )
            .unwrap_err(),
            ERR_UNCERTAIN
        );
    }

    #[test]
    fn managed_end_matrix() {
        let end = TerminalRequest {
            action: "end".to_string(),
            id: TID.to_string(),
            project: PID.to_string(),
            login: "dev".to_string(),
            identity: 7,
            ..Default::default()
        };
        // Non-end requests never call out.
        let exec = FakeExec::new(vec![]);
        let svc = make_service(exec);
        let mut other = end.clone();
        other.action = "attach".to_string();
        assert!(svc.managed_end(CID, &other, None, deadline()).is_ok());
        assert!(svc.exec.calls().is_empty());
        // Lookup failure propagates.
        let exec = FakeExec::new(vec![Err("boom".to_string())]);
        let svc = make_service(exec);
        assert_eq!(
            svc.managed_end(CID, &end, None, deadline()).unwrap_err(),
            "managed Codex operation failed"
        );
        // No lease recorded: success without a callback.
        let body = Delivery::default().encode();
        let exec = FakeExec::new(vec![ok(&body)]);
        let svc = make_service(exec);
        assert!(svc.managed_end(CID, &end, None, deadline()).is_ok());
        // Binding mismatch is stale.
        let mut lease = live_lease();
        lease.binding.as_mut().unwrap().id = "ffffffffffffffffffffffffffffffff".to_string();
        let body = Delivery {
            lease,
            credential: None,
        }
        .encode();
        let exec = FakeExec::new(vec![ok(&body)]);
        let svc = make_service(exec);
        assert_eq!(
            svc.managed_end(CID, &end, None, deadline()).unwrap_err(),
            ERR_STALE
        );
        // Missing callback denies; present callback runs.
        let body = Delivery {
            lease: live_lease(),
            credential: None,
        }
        .encode();
        let exec = FakeExec::new(vec![ok(&body)]);
        let svc = make_service(exec);
        assert_eq!(
            svc.managed_end(CID, &end, None, deadline()).unwrap_err(),
            ERR_DENIED
        );
        let body = Delivery {
            lease: live_lease(),
            credential: None,
        }
        .encode();
        let exec = FakeExec::new(vec![ok(&body)]);
        let svc = make_service(exec);
        let seen = Mutex::new(Vec::new());
        let end_identity = |actor: i64, lease_id: &str| {
            seen.lock().unwrap().push((actor, lease_id.to_string()));
            Ok(())
        };
        assert!(svc
            .managed_end(CID, &end, Some(&end_identity), deadline())
            .is_ok());
        assert_eq!(
            seen.lock().unwrap().as_slice(),
            &[(7, "lease-1".to_string())]
        );
        // Lookup stdin pins the request envelope.
        let calls = svc.exec.calls();
        let body_json = String::from_utf8(calls[0].0.clone()).unwrap();
        assert!(body_json.contains(r#""action":"lookup""#), "{body_json}");
        assert!(body_json.contains(r#""login":"dev""#), "{body_json}");
    }

    // ----- agent + harness files -----

    fn euid() -> u32 {
        unsafe { libc::geteuid() }
    }

    fn with_agent_env(path: &std::path::Path) -> MutexGuard<'static, ()> {
        let guard = ENV_LOCK.lock().unwrap();
        unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", path) };
        guard
    }

    #[test]
    fn agent_program_hash_matrix() {
        let dir = test_tmp("agent");
        let path = dir.join("project-terminal");
        std::fs::write(&path, b"agent-bytes").unwrap();
        let _guard = with_agent_env(&path);
        assert_eq!(agent_program_path(), path.to_str().unwrap().to_string());
        // Happy path under the test uid.
        assert_eq!(
            agent_program_hash(euid()).unwrap(),
            sha256::hex_lower(&sha256::digest(b"agent-bytes"))
        );
        // Ownership gate.
        assert_eq!(
            agent_program_hash(euid().wrapping_add(1)).unwrap_err(),
            "project terminal agent has unexpected ownership"
        );
        // Writable gate: 0o644 passes (no write bits outside owner).
        use std::os::unix::fs::PermissionsExt;
        assert!(agent_program_hash(euid()).is_ok());
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o664)).unwrap();
        assert_eq!(
            agent_program_hash(euid()).unwrap_err(),
            "project terminal agent is group- or world-writable"
        );
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o755)).unwrap();
        assert!(agent_program_hash(euid()).is_ok());
        // Symlinks and directories are not regular files.
        let link = dir.join("link");
        std::os::unix::fs::symlink(&path, &link).unwrap();
        unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &link) };
        assert_eq!(
            agent_program_hash(euid()).unwrap_err(),
            "project terminal agent is not a regular file"
        );
        unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &dir) };
        assert_eq!(
            agent_program_hash(euid()).unwrap_err(),
            "project terminal agent is not a regular file"
        );
        // Missing file.
        unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", dir.join("absent")) };
        assert!(
            agent_program_hash(euid())
                .unwrap_err()
                .starts_with("project terminal agent unavailable: "),
            "missing file must fail with unavailable"
        );
        // Empty file.
        let empty = dir.join("empty");
        std::fs::write(&empty, b"").unwrap();
        unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &empty) };
        assert_eq!(
            agent_program_hash(euid()).unwrap_err(),
            "project terminal agent has unexpected size"
        );
        // Oversize file (sparse).
        let big = dir.join("big");
        let f = std::fs::File::create(&big).unwrap();
        f.set_len((32 << 20) + 1).unwrap();
        unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &big) };
        assert_eq!(
            agent_program_hash(euid()).unwrap_err(),
            "project terminal agent has unexpected size"
        );
        unsafe { std::env::remove_var("SODA_PROJECT_TERMINAL") };
    }

    fn write_harness(dir: &std::path::Path, bytes: &[u8], mode: u32) -> String {
        let bin = dir.join("bin");
        std::fs::create_dir_all(&bin).unwrap();
        let path = bin.join("codex");
        std::fs::write(&path, bytes).unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(mode)).unwrap();
        sha256::hex_lower(&sha256::digest(bytes))
    }

    #[test]
    fn verify_identity_harness_matrix() {
        let dir = test_tmp("harness");
        let digest = write_harness(&dir, b"codex-bytes", 0o755);
        let mut svc = make_service(FakeExec::new(vec![]));
        svc.codex_harness = dir.to_str().unwrap().to_string();
        svc.codex_harness_sha256 = digest;
        assert!(svc.verify_identity_harness().is_ok());
        svc.codex_harness_sha256 = "0".repeat(64);
        assert_eq!(
            svc.verify_identity_harness().unwrap_err(),
            "codex harness digest differs"
        );
        svc.codex_harness = dir.join("absent").to_str().unwrap().to_string();
        assert_eq!(
            svc.verify_identity_harness().unwrap_err(),
            "verified Codex executable required"
        );
        let dir2 = test_tmp("harness-noexec");
        write_harness(&dir2, b"codex-bytes", 0o644);
        svc.codex_harness = dir2.to_str().unwrap().to_string();
        assert_eq!(
            svc.verify_identity_harness().unwrap_err(),
            "verified Codex executable required"
        );
    }

    #[test]
    fn stream_identity_harness_flows() {
        let svc = make_service(FakeExec::new(vec![Err("no tar".to_string())]));
        assert_eq!(
            svc.stream_identity_harness(CID, "/run/x", deadline())
                .unwrap_err(),
            "codex harness stream unavailable"
        );
        let svc = make_service(FakeExec::new(vec![
            ok("tar-bytes"),
            Err("no podman".to_string()),
        ]));
        assert_eq!(
            svc.stream_identity_harness(CID, "/run/x", deadline())
                .unwrap_err(),
            "codex harness staging failed"
        );
        let svc = make_service(FakeExec::new(vec![ok("tar-bytes"), ok("")]));
        assert!(svc
            .stream_identity_harness(CID, "/run/x", deadline())
            .is_ok());
        let calls = svc.exec.calls();
        assert_eq!(calls[0].1, "/usr/bin/tar");
        assert_eq!(calls[0].2, tar_producer_argv("/opt/harness"));
        assert_eq!(calls[1].1, "/usr/bin/podman");
        assert_eq!(calls[1].2, tar_consumer_argv(CID, "/run/x"));
        assert_eq!(calls[1].0, b"tar-bytes");
    }

    fn preparing_lease() -> Lease {
        // Binding-free lease with a live 12h deadline window.
        Lease {
            provider_id: "codex".to_string(),
            id: "lease-9".to_string(),
            connection_id: "conn".to_string(),
            generation: 1,
            actor_id: 7,
            project_id: PID.to_string(),
            execution_id: TID.to_string(),
            kind: KIND_TERMINAL.to_string(),
            deadline_raw: "live-window".to_string(),
            deadline: Some((now_unix() + 3600, 0)),
            ..Default::default()
        }
    }

    #[test]
    fn prepare_identity_flows() {
        // Denied input never calls out.
        let exec = FakeExec::new(vec![]);
        let svc = make_service(exec);
        assert_eq!(
            svc.prepare_identity(&Lease::default(), "dev", CID, 80, 24, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
        // The prepare path verifies the production agent binary (uid 0):
        // as non-root the ownership gate fires after the inspect call.
        let _guard = ENV_LOCK.lock().unwrap();
        let dir = test_tmp("prep-agent");
        let path = dir.join("project-terminal");
        std::fs::write(&path, b"agent-bytes").unwrap();
        unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &path) };
        let exec = FakeExec::new(vec![ok(&inspect_json(
            CID, true, PID, "7", false, "private",
        ))]);
        let svc = make_service(exec);
        if euid() == 0 {
            let prepared = Delivery {
                lease: live_lease(),
                credential: None,
            }
            .encode();
            let exec = FakeExec::new(vec![
                ok(&inspect_json(CID, true, PID, "7", false, "private")),
                ok(&prepared),
            ]);
            let svc = make_service(exec);
            let mut lease = preparing_lease();
            lease.id = "lease-1".to_string();
            let binding = svc
                .prepare_identity(&lease, "dev", CID, 80, 24, deadline())
                .unwrap();
            assert_eq!(binding, live_lease().binding.unwrap());
            let calls = svc.exec.calls();
            assert_eq!(calls.len(), 2);
            let stdin = String::from_utf8(calls[1].0.clone()).unwrap();
            assert!(stdin.contains(r#""action":"prepare""#), "{stdin}");
            assert!(stdin.contains(&format!("\"container\":{CID:?}")), "{stdin}");
        } else {
            assert_eq!(
                svc.prepare_identity(&preparing_lease(), "dev", CID, 80, 24, deadline())
                    .unwrap_err(),
                "project terminal agent has unexpected ownership"
            );
            assert_eq!(svc.exec.calls().len(), 1);
        }
        unsafe { std::env::remove_var("SODA_PROJECT_TERMINAL") };
    }

    #[test]
    fn identity_start_flow() {
        // Full start sequence: inspect, stage broker call, tar producer,
        // tar consumer, start broker call.
        let dir = test_tmp("start-harness");
        let digest = write_harness(&dir, b"codex-bytes", 0o755);
        let lease = live_lease();
        let delivery = Delivery {
            lease: lease.clone(),
            credential: Some(b"{}".to_vec()),
        };
        let stage_out = Delivery {
            lease: lease.clone(),
            credential: None,
        }
        .encode();
        let start_out = Delivery {
            lease: lease.clone(),
            credential: None,
        }
        .encode();
        let exec = FakeExec::new(vec![
            ok(&inspect_json(CID, true, PID, "7", false, "private")),
            ok(&stage_out),
            ok("tar-bytes"),
            ok(""),
            ok(&start_out),
        ]);
        let mut svc = make_service(exec);
        svc.codex_harness = dir.to_str().unwrap().to_string();
        svc.codex_harness_sha256 = digest;
        let out = svc.identity("start", &delivery, deadline()).unwrap();
        assert_eq!(out.lease, lease);
        let calls = svc.exec.calls();
        assert_eq!(calls.len(), 5);
        assert_eq!(calls[0].1, "/usr/bin/podman");
        assert_eq!(calls[1].1, "/usr/bin/podman");
        assert_eq!(calls[2].1, "/usr/bin/tar");
        assert_eq!(calls[3].1, "/usr/bin/podman");
        assert_eq!(
            calls[3].2,
            tar_consumer_argv(CID, &format!("/run/soda-terminals/{TID}/model/harness"))
        );
        let stage_stdin = String::from_utf8(calls[1].0.clone()).unwrap();
        assert!(stage_stdin.contains(r#""action":"stage""#), "{stage_stdin}");
        let start_stdin = String::from_utf8(calls[4].0.clone()).unwrap();
        assert!(start_stdin.contains(r#""action":"start""#), "{start_stdin}");
        assert!(
            start_stdin.contains(&format!(
                "\"harness_sha256\":{:?}",
                svc.codex_harness_sha256
            )),
            "{start_stdin}"
        );
    }

    // ----- stream table + admission -----

    #[test]
    fn stream_table_flows() {
        let mut table = StreamTable::new();
        let mut flags = Vec::new();
        for _ in 0..STREAM_LIMIT {
            let (id, cancel) = table.register().expect("room");
            flags.push((id, cancel));
        }
        assert!(table.register().is_none());
        assert_eq!(table.len(), STREAM_LIMIT);
        table.unregister(flags[0].0);
        assert_eq!(table.len(), STREAM_LIMIT - 1);
        let (id, _) = table.register().expect("room after unregister");
        table.unregister(id);
        table.close();
        assert!(table.is_closed());
        assert!(table.register().is_none());
        // Only still-registered streams are cancelled.
        assert!(!flags[0].1.load(Ordering::SeqCst));
        for (_, cancel) in &flags[1..] {
            assert!(cancel.load(Ordering::SeqCst));
        }
    }

    #[test]
    fn private_request_matrix() {
        assert!(valid_private_terminal_request("GET", "", false, "", 0));
        assert!(!valid_private_terminal_request("POST", "", false, "", 0));
        assert!(!valid_private_terminal_request("GET", "x=1", false, "", 0));
        assert!(!valid_private_terminal_request("GET", "", true, "", 0));
        assert!(!valid_private_terminal_request("GET", "", false, "/x", 0));
        assert!(!valid_private_terminal_request("GET", "", false, "", 1));
        assert!(valid_identity_request("POST", "", false, "", 0));
        assert!(!valid_identity_request("GET", "", false, "", 0));
        assert!(!valid_identity_request("POST", "x", false, "", 0));
        assert_eq!(identity_action("/identity/validate"), "validate");
        assert_eq!(identity_action("/identity/launch"), "launch");
        assert_eq!(identity_action("/other"), "/other");
    }

    #[test]
    fn terminal_start_decode() {
        let body = br#"{"connection_id":"c","project_id":"p0123456789abcdef01234567","actor_id":"7","login":"dev","scope":"s","cols":80,"rows":24}"#;
        let start = TerminalStart::decode(body).unwrap();
        assert_eq!(start.actor_id, 7);
        assert_eq!(start.cols, 80);
        assert!(TerminalStart::decode(br#"{"actor_id":""}"#).is_err());
        assert!(TerminalStart::decode(br#"{"actor_id":7}"#).is_err());
        assert_eq!(TerminalStart::decode(br#"{}"#).unwrap().actor_id, 0);
        assert!(TerminalStart::decode(br#"{"bogus":1}"#).is_err());
    }

    #[test]
    fn zero_deadline_encode_pin() {
        // Go marshals the zero time, never an empty string (probed).
        let encoded = Lease::default().encode();
        assert!(
            encoded.contains(r#""deadline":"0001-01-01T00:00:00Z""#),
            "{encoded}"
        );
        let round =
            Delivery::decode(format!("{{\"lease\":{encoded},\"credential\":null}}").as_bytes())
                .unwrap();
        assert_eq!(round.lease.deadline_raw, "0001-01-01T00:00:00Z");
        assert_eq!(round.lease.deadline, parse_rfc3339("0001-01-01T00:00:00Z"));
    }

    #[test]
    fn output_line_and_attach_pins() {
        let f = parse_output_line(br#"{"type":"output","data":"aGk="}"#).unwrap();
        assert_eq!(f.frame_type, "output");
        assert_eq!(
            parse_output_line(br#"{"type":"input","data":"aGk="}"#).unwrap_err(),
            "invalid terminal response"
        );
        assert_eq!(
            parse_output_line(b"nope").unwrap_err(),
            "invalid terminal response"
        );
        // Attach pre-checks fire before any spawn.
        assert_eq!(
            NativeAttach::attach("short", &base_request()).unwrap_err(),
            "invalid terminal target"
        );
        let mut expired = base_request();
        expired.expires = now_unix() - 1;
        assert_eq!(
            NativeAttach::attach(CID, &expired).unwrap_err(),
            "invalid terminal target"
        );
    }

    #[test]
    fn rand_id_shape() {
        let a = rand_id().unwrap();
        let b = rand_id().unwrap();
        assert_eq!(a.len(), 32);
        assert!(a.bytes().all(|c| c.is_ascii_hexdigit()));
        assert_ne!(a, b);
    }

    // ----- daemon dispatch -----

    struct FakeBroker {
        acquire_result: Mutex<Result<Lease, String>>,
        register_result: Mutex<Result<Delivery, String>>,
        reconciled: Mutex<Vec<String>>,
        acquires: Mutex<Vec<AcquireRequest>>,
    }

    impl IdentityBroker for FakeBroker {
        fn acquire(&self, req: &AcquireRequest, _deadline: Instant) -> Result<Lease, String> {
            self.acquires.lock().unwrap().push(req.clone());
            self.acquire_result.lock().unwrap().clone()
        }
        fn register(
            &self,
            _lease_id: &str,
            _binding: &Binding,
            _deadline: Instant,
        ) -> Result<Delivery, String> {
            self.register_result.lock().unwrap().clone()
        }
        fn reconcile_lease(&self, lease_id: &str) -> Result<(), String> {
            self.reconciled.lock().unwrap().push(lease_id.to_string());
            Ok(())
        }
    }

    fn launch_input() -> TerminalStart {
        TerminalStart {
            connection_id: "conn".to_string(),
            project_id: PID.to_string(),
            actor_id: 7,
            login: "dev".to_string(),
            scope: CID.to_string(),
            cols: 80,
            rows: 24,
        }
    }

    #[test]
    fn identity_launch_flows() {
        let input = launch_input();
        // Unconfigured harness denies before acquiring.
        let broker = FakeBroker {
            acquire_result: Mutex::new(Err("unreachable".to_string())),
            register_result: Mutex::new(Err("unreachable".to_string())),
            reconciled: Mutex::new(Vec::new()),
            acquires: Mutex::new(Vec::new()),
        };
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            identity_launch(&broker, &svc, &input, false, deadline()).unwrap_err(),
            ERR_DENIED
        );
        assert!(broker.acquires.lock().unwrap().is_empty());
        // Acquire failure: no reconcile.
        let broker = FakeBroker {
            acquire_result: Mutex::new(Err("broker down".to_string())),
            register_result: Mutex::new(Err("unreachable".to_string())),
            reconciled: Mutex::new(Vec::new()),
            acquires: Mutex::new(Vec::new()),
        };
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            identity_launch(&broker, &svc, &input, true, deadline()).unwrap_err(),
            "broker down"
        );
        assert!(broker.reconciled.lock().unwrap().is_empty());
        let acquire = broker.acquires.lock().unwrap()[0].clone();
        assert_eq!(acquire.provider_id, "codex");
        assert_eq!(acquire.actor_id, 7);
        assert_eq!(acquire.kind, "terminal");
        assert_eq!(acquire.execution_id.len(), 32);
        assert!((acquire.deadline_secs - (now_unix() + 12 * 3600)).abs() <= 5);
        // Prepare failure reconciles.
        let broker = FakeBroker {
            acquire_result: Mutex::new(Ok(preparing_lease())),
            register_result: Mutex::new(Err("unreachable".to_string())),
            reconciled: Mutex::new(Vec::new()),
            acquires: Mutex::new(Vec::new()),
        };
        let mut bad = preparing_lease();
        bad.project_id = "nope".to_string();
        *broker.acquire_result.lock().unwrap() = Ok(bad);
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            identity_launch(&broker, &svc, &input, true, deadline()).unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            broker.reconciled.lock().unwrap().as_slice(),
            &["lease-9".to_string()]
        );
        // Register failure reconciles (prepare succeeds only as root; as
        // non-root the agent ownership gate fires first and still reconciles).
        let _guard = ENV_LOCK.lock().unwrap();
        let dir = test_tmp("launch-agent");
        let path = dir.join("project-terminal");
        std::fs::write(&path, b"agent-bytes").unwrap();
        unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &path) };
        let broker = FakeBroker {
            acquire_result: Mutex::new(Ok(preparing_lease())),
            register_result: Mutex::new(Err("register down".to_string())),
            reconciled: Mutex::new(Vec::new()),
            acquires: Mutex::new(Vec::new()),
        };
        let prepared = Delivery {
            lease: live_lease(),
            credential: None,
        }
        .encode();
        let exec = FakeExec::new(vec![
            ok(&inspect_json(CID, true, PID, "7", false, "private")),
            ok(&prepared),
        ]);
        let svc = make_service(exec);
        if euid() == 0 {
            // prepare would still fail: lease id differs from prepared id.
            assert_eq!(
                identity_launch(&broker, &svc, &input, true, deadline()).unwrap_err(),
                "managed terminal reservation differs"
            );
        } else {
            assert_eq!(
                identity_launch(&broker, &svc, &input, true, deadline()).unwrap_err(),
                "project terminal agent has unexpected ownership"
            );
        }
        assert_eq!(
            broker.reconciled.lock().unwrap().as_slice(),
            &["lease-9".to_string()]
        );
        unsafe { std::env::remove_var("SODA_PROJECT_TERMINAL") };
    }

    #[test]
    fn identity_route_matrix() {
        let factory = Lease {
            kind: KIND_FACTORY.to_string(),
            ..Default::default()
        };
        assert_eq!(
            identity_route("/identity/launch", &factory, true),
            IdentityRoute::Launch
        );
        assert_eq!(
            identity_route("/identity/validate", &factory, true),
            IdentityRoute::Factory
        );
        let mut muse = live_lease();
        muse.provider_id = "muse".to_string();
        muse.binding.as_mut().unwrap().scope = SCOPE_MUSE_PROJECT.to_string();
        assert_eq!(
            identity_route("/identity/stop", &muse, true),
            IdentityRoute::Muse
        );
        assert_eq!(
            identity_route("/identity/stop", &muse, false),
            IdentityRoute::Terminal
        );
        muse.binding.as_mut().unwrap().scope = "other".to_string();
        assert_eq!(
            identity_route("/identity/stop", &muse, true),
            IdentityRoute::Terminal
        );
        assert_eq!(
            identity_route("/identity/finish", &live_lease(), true),
            IdentityRoute::Terminal
        );
    }

    fn piped_file() -> (std::os::unix::net::UnixStream, File) {
        use std::os::unix::io::{FromRawFd, IntoRawFd};
        use std::os::unix::net::UnixStream;
        let (peer, end) = UnixStream::pair().unwrap();
        // SAFETY: the fd is owned by the new File exactly once.
        let file = unsafe { File::from_raw_fd(end.into_raw_fd()) };
        (peer, file)
    }

    #[test]
    fn take_reader_detaches_output() {
        let (out_peer, out_file) = piped_file();
        let (_in_peer, in_file) = piped_file();
        let mut attach = NativeAttach {
            child: None,
            stdin: Some(in_file),
            reader: Some(BufReader::new(out_file)),
            closed: false,
        };
        let mut reader = attach.take_reader().expect("reader detached");
        assert!(attach.take_reader().is_none());
        std::io::Write::write_all(&mut &out_peer, b"{\"type\":\"ready\"}\n").unwrap();
        let frame = NativeAttach::output_frame(&mut reader).unwrap();
        assert_eq!(frame.frame_type, "ready");
        drop(out_peer);
        assert_eq!(
            NativeAttach::output_frame(&mut reader).unwrap_err(),
            "terminal output ended"
        );
        attach.close();
        assert!(attach.closed);
    }

    #[test]
    fn quiet_output_never_blocks_input() {
        let (out_peer, out_file) = piped_file();
        let (_in_peer, in_file) = piped_file();
        let mut attach = NativeAttach {
            child: None,
            stdin: Some(in_file),
            reader: Some(BufReader::new(out_file)),
            closed: false,
        };
        let mut reader = attach.take_reader().unwrap();
        let out = std::thread::spawn(move || NativeAttach::output_frame(&mut reader));
        // Child quiet (peer open, no data): input still flows.
        let frame = TerminalFrame {
            frame_type: "input".to_string(),
            data: "eA==".to_string(),
            cols: 0,
            rows: 0,
            reason: String::new(),
            terminals: None,
        };
        attach.input_frame(&frame).unwrap();
        drop(out_peer);
        assert_eq!(out.join().unwrap().unwrap_err(), "terminal output ended");
        attach.close();
    }
}
