//! Native tailscaled observation and host actions.
//!
//! PR26 port of the `control.go` LocalAPI flow (`request`, `observe`,
//! `execute*`, `readbackHostAction`, `decodeUpNotifications`), the native
//! document decoders (`nativeObject`, `peerView` binding), and the `tailnet.go`
//! CLI client (`Status`, `Endpoint`).
//!
//! LocalAPI transport is plain HTTP/1.1 over the tailscaled unix socket,
//! spoken with `std` only. Provider HTTPS is not here (see the provider
//! module); command execution goes through the crate's `Executor`.

use std::io::{Read, Write};
use std::os::unix::net::UnixStream;
use std::path::Path;
use std::time::{Duration, Instant};

use crate::json::Value;
use crate::tcontrol_wire as wire;

pub const HOST_SOCKET: &str = "/var/run/tailscale/tailscaled.sock";
pub const DEFAULT_CLI: &str = "/usr/bin/tailscale";
pub const DEFAULT_LIBEXEC: &str = "/usr/libexec/soda";
/// Native LocalAPI body cap (`responseLimit` in control.go).
pub const RESPONSE_LIMIT: usize = 65536;
/// Per-request LocalAPI timeout, mirroring Go's 8s client timeout.
const LOCAL_TIMEOUT: Duration = Duration::from_secs(8);
/// Fresh-login `tailscale up` sub-deadline, mirroring Go's 8s timeout.
const UP_TIMEOUT: Duration = Duration::from_secs(8);

/// LocalAPI round trip: `(method, path, body, deadline)` returning the raw
/// `(status, body)`; [`finish_local`] applies Go's `request` rules. The
/// production transport is [`local_request`]; tests inject stubs.
pub type Transport =
    dyn Fn(&str, &str, Option<&[u8]>, Instant) -> Result<(u16, Vec<u8>), String> + Send + Sync;

/// Borrowed LocalAPI round trip for function parameters: any closure or
/// stub, with no auto-trait bounds and no `'static` requirement.
pub type RoundTrip<'a> =
    &'a (dyn Fn(&str, &str, Option<&[u8]>, Instant) -> Result<(u16, Vec<u8>), String> + 'a);

/// Apply Go's `request` rules: status must be 200/204 and the body capped.
pub fn finish_local(status: u16, body: &[u8]) -> Result<Vec<u8>, String> {
    if body.len() > RESPONSE_LIMIT || (status != 200 && status != 204) {
        return Err(wire::err_unavailable());
    }
    Ok(body.to_vec())
}

/// Raw HTTP/1.1 round trip over the tailscaled unix socket.
pub fn local_request(
    socket: &Path,
    method: &str,
    path: &str,
    body: Option<&[u8]>,
    deadline: Instant,
) -> Result<(u16, Vec<u8>), String> {
    let now = Instant::now();
    if now >= deadline {
        return Err(wire::err_unavailable());
    }
    let timeout = (deadline - now).min(LOCAL_TIMEOUT);
    let mut stream = UnixStream::connect(socket).map_err(|_| wire::err_unavailable())?;
    stream
        .set_read_timeout(Some(timeout))
        .map_err(|_| wire::err_unavailable())?;
    stream
        .set_write_timeout(Some(timeout))
        .map_err(|_| wire::err_unavailable())?;
    let payload = body.unwrap_or_default();
    let head = format!(
        "{method} /localapi/v0/{path} HTTP/1.1\r\nHost: local-tailscaled.sock\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        payload.len()
    );
    stream
        .write_all(head.as_bytes())
        .map_err(|_| wire::err_unavailable())?;
    stream
        .write_all(payload)
        .map_err(|_| wire::err_unavailable())?;
    // Close-delimited read with a total cap (headers plus the 64 KiB body).
    let mut raw = Vec::new();
    let mut chunk = [0u8; 8192];
    loop {
        match stream.read(&mut chunk) {
            Ok(0) => break,
            Ok(n) => {
                raw.extend_from_slice(&chunk[..n]);
                if raw.len() > RESPONSE_LIMIT + 8192 {
                    return Err(wire::err_unavailable());
                }
            }
            Err(_) => return Err(wire::err_unavailable()),
        }
    }
    let text = std::str::from_utf8(&raw).map_err(|_| wire::err_unavailable())?;
    let (head, body) = match text.split_once("\r\n\r\n") {
        Some(split) => split,
        None => return Err(wire::err_unavailable()),
    };
    let status = head
        .split_whitespace()
        .nth(1)
        .and_then(|code| code.parse::<u16>().ok())
        .filter(|code| (100..600).contains(code));
    match status {
        Some(code) => Ok((code, body.as_bytes().to_vec())),
        None => Err(wire::err_unavailable()),
    }
}

// ---------- Native document decoding ----------
//
// Mirror of `nativeObject` + `encoding/json` binding: strict single-object
// decode with duplicate rejection at every level, exact-name required-field
// presence with the `HaveNodeKey` omission rule, then case-folded last-wins
// binding with null as a no-op and mistyped values failing the decode.

fn fold_char(c: char) -> char {
    match c {
        'ſ' => 's',
        '\u{212a}' => 'k',
        x if x.is_ascii_alphabetic() => x.to_ascii_lowercase(),
        x => x,
    }
}

fn fold_eq(a: &str, b: &str) -> bool {
    if a == b {
        return true;
    }
    let mut ai = a.chars();
    let mut bi = b.chars();
    loop {
        match (ai.next(), bi.next()) {
            (None, None) => return true,
            (Some(x), Some(y)) => {
                if x != y && fold_char(x) != fold_char(y) {
                    return false;
                }
            }
            (None, Some(_)) | (Some(_), None) => return false,
        }
    }
}

/// Strict validation plus the document-order value for last-wins binding
/// (strict success implies tolerant success: same grammar, fewer bans).
fn native_object(data: &[u8], required: &[&str]) -> Result<Value, String> {
    if crate::json::decode_strict(data).is_err() {
        return Err(wire::err_unavailable());
    }
    let v = match crate::json::decode_tolerant(data) {
        Ok(v) => v,
        Err(_) => return Err(wire::err_unavailable()),
    };
    let fields = match v.as_object() {
        Some(f) => f,
        None => return Err(wire::err_unavailable()),
    };
    for key in required {
        let present = match fields.iter().find(|(k, _)| k == key) {
            None => {
                // `haveNodeKeyOmitted`: an absent field is false on a fresh,
                // unenrolled daemon; case-aliased fields fail closed.
                if *key != "HaveNodeKey" {
                    return Err(wire::err_unavailable());
                }
                if fields.iter().any(|(k, _)| fold_eq(k, key)) {
                    return Err(wire::err_unavailable());
                }
                continue;
            }
            Some((_, v)) => v,
        };
        if present.is_null() && *key != "AdvertiseRoutes" {
            return Err(wire::err_unavailable());
        }
    }
    Ok(v)
}

fn bound_values<'a>(fields: &'a [(String, Value)], name: &str) -> Vec<&'a Value> {
    fields
        .iter()
        .filter(|(k, _)| fold_eq(k, name))
        .map(|(_, v)| v)
        .collect()
}

fn bind_string_into(
    fields: &[(String, Value)],
    name: &str,
    out: &mut String,
) -> Result<(), String> {
    for v in bound_values(fields, name) {
        match v {
            Value::Null => {}
            Value::Str(s) => *out = s.clone(),
            _ => return Err(wire::err_unavailable()),
        }
    }
    Ok(())
}

fn bind_bool_into(fields: &[(String, Value)], name: &str, out: &mut bool) -> Result<(), String> {
    for v in bound_values(fields, name) {
        match v {
            Value::Null => {}
            Value::Bool(b) => *out = *b,
            _ => return Err(wire::err_unavailable()),
        }
    }
    Ok(())
}

fn bind_list_into(
    fields: &[(String, Value)],
    name: &str,
    out: &mut Vec<String>,
) -> Result<(), String> {
    for v in bound_values(fields, name) {
        match v {
            Value::Null => {}
            Value::Array(items) => {
                let mut next = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        Value::Null => next.push(String::new()),
                        Value::Str(s) => next.push(s.clone()),
                        _ => return Err(wire::err_unavailable()),
                    }
                }
                *out = next;
            }
            _ => return Err(wire::err_unavailable()),
        }
    }
    Ok(())
}

#[derive(Debug, Clone, Default)]
pub struct NativePeer {
    pub id: String,
    pub dns_name: String,
    pub ips: Vec<String>,
    pub online: bool,
    pub exit_node_option: bool,
    pub expired: bool,
}

fn bind_native_peer_into(fields: &[(String, Value)], p: &mut NativePeer) -> Result<(), String> {
    bind_string_into(fields, "ID", &mut p.id)?;
    bind_string_into(fields, "DNSName", &mut p.dns_name)?;
    bind_list_into(fields, "TailscaleIPs", &mut p.ips)?;
    bind_bool_into(fields, "Online", &mut p.online)?;
    bind_bool_into(fields, "ExitNodeOption", &mut p.exit_node_option)?;
    bind_bool_into(fields, "Expired", &mut p.expired)?;
    Ok(())
}

#[derive(Debug, Clone, Default)]
pub struct NativeStatus {
    pub backend_state: String,
    pub have_node_key: bool,
    pub tailnet_name: String,
    pub tailnet_magic_dns: bool,
    pub has_tailnet: bool,
    pub self_peer: Option<NativePeer>,
    pub peers: Vec<NativePeer>,
    pub health_count: usize,
    pub auth_url: String,
}

pub fn decode_native_status(data: &[u8]) -> Result<NativeStatus, String> {
    let v = native_object(data, &["BackendState", "HaveNodeKey"])?;
    let fields = v.as_object().ok_or_else(wire::err_unavailable)?;
    let mut s = NativeStatus::default();
    bind_string_into(fields, "BackendState", &mut s.backend_state)?;
    bind_bool_into(fields, "HaveNodeKey", &mut s.have_node_key)?;
    bind_string_into(fields, "AuthURL", &mut s.auth_url)?;
    // Pointer-struct and map bindings merge across case-variant keys into
    // one value, like `encoding/json`; null is a no-op, mistyped fails.
    for v in bound_values(fields, "CurrentTailnet") {
        match v {
            Value::Null => {}
            Value::Object(inner) => {
                s.has_tailnet = true;
                bind_string_into(inner, "Name", &mut s.tailnet_name)?;
                bind_bool_into(inner, "MagicDNSEnabled", &mut s.tailnet_magic_dns)?;
            }
            _ => return Err(wire::err_unavailable()),
        }
    }
    for v in bound_values(fields, "Self") {
        match v {
            Value::Null => {}
            Value::Object(inner) => {
                let peer = s.self_peer.get_or_insert_with(NativePeer::default);
                bind_native_peer_into(inner, peer)?;
            }
            _ => return Err(wire::err_unavailable()),
        }
    }
    // `Peer` is a map: entries merge by key across values, keys are
    // otherwise ignored, and null values bind the zero peer.
    let mut peer_keys: Vec<String> = Vec::new();
    for v in bound_values(fields, "Peer") {
        match v {
            Value::Null => {}
            Value::Object(entries) => {
                for (key, peer) in entries {
                    let mut decoded = NativePeer::default();
                    match peer {
                        Value::Null => {}
                        Value::Object(inner) => bind_native_peer_into(inner, &mut decoded)?,
                        _ => return Err(wire::err_unavailable()),
                    }
                    match peer_keys.iter().position(|k| k == key) {
                        Some(at) => s.peers[at] = decoded,
                        None => {
                            peer_keys.push(key.clone());
                            s.peers.push(decoded);
                        }
                    }
                }
            }
            _ => return Err(wire::err_unavailable()),
        }
    }
    // `Health` contributes only its length, but must bind as `[]string`.
    for v in bound_values(fields, "Health") {
        match v {
            Value::Null => {}
            Value::Array(items) => {
                for item in items {
                    match item {
                        Value::Null | Value::Str(_) => {}
                        _ => return Err(wire::err_unavailable()),
                    }
                }
                s.health_count = items.len();
            }
            _ => return Err(wire::err_unavailable()),
        }
    }
    Ok(s)
}

#[derive(Debug, Clone, Default)]
pub struct NativePrefs {
    pub want_running: bool,
    pub exit_node_id: String,
    pub exit_node_ip: String,
    pub allow_lan: bool,
    pub advertise_routes: Option<Vec<String>>,
}

pub fn decode_native_prefs(data: &[u8]) -> Result<NativePrefs, String> {
    let v = native_object(
        data,
        &[
            "WantRunning",
            "ExitNodeID",
            "ExitNodeIP",
            "ExitNodeAllowLANAccess",
            "AdvertiseRoutes",
        ],
    )?;
    let fields = v.as_object().ok_or_else(wire::err_unavailable)?;
    let mut p = NativePrefs::default();
    bind_bool_into(fields, "WantRunning", &mut p.want_running)?;
    bind_string_into(fields, "ExitNodeID", &mut p.exit_node_id)?;
    bind_string_into(fields, "ExitNodeIP", &mut p.exit_node_ip)?;
    bind_bool_into(fields, "ExitNodeAllowLANAccess", &mut p.allow_lan)?;
    for v in bound_values(fields, "AdvertiseRoutes") {
        match v {
            Value::Null => p.advertise_routes = None,
            Value::Array(items) => {
                let mut next = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        Value::Null => next.push(String::new()),
                        Value::Str(s) => next.push(s.clone()),
                        _ => return Err(wire::err_unavailable()),
                    }
                }
                p.advertise_routes = Some(next);
            }
            _ => return Err(wire::err_unavailable()),
        }
    }
    Ok(p)
}

// ---------- Observation ----------

fn round_trip(
    transport: RoundTrip<'_>,
    method: &str,
    path: &str,
    body: Option<&[u8]>,
    deadline: Instant,
) -> Result<Vec<u8>, String> {
    let (status, data) = transport(method, path, body, deadline)?;
    finish_local(status, &data)
}

pub fn fetch_native_status(
    transport: RoundTrip<'_>,
    deadline: Instant,
) -> Result<NativeStatus, String> {
    let data = round_trip(transport, "GET", "status", None, deadline)?;
    let s = decode_native_status(&data)?;
    if s.backend_state.is_empty() || s.peers.len() > 128 || s.health_count > 128 {
        return Err(wire::err_unavailable());
    }
    match s.backend_state.as_str() {
        "NoState" | "InUseOtherUser" | "NeedsLogin" | "NeedsMachineAuth" | "Stopped"
        | "Starting" => {}
        "Running" => match &s.self_peer {
            Some(peer) if s.have_node_key && !peer.id.is_empty() && !peer.ips.is_empty() => {}
            _ => return Err(wire::err_unavailable()),
        },
        _ => return Err(wire::err_unavailable()),
    }
    Ok(s)
}

pub fn fetch_native_prefs(
    transport: RoundTrip<'_>,
    deadline: Instant,
) -> Result<NativePrefs, String> {
    let data = round_trip(transport, "GET", "prefs", None, deadline)?;
    let p = decode_native_prefs(&data)?;
    if p.advertise_routes.as_ref().map(Vec::len).unwrap_or(0) > 128 || p.exit_node_id.len() > 128 {
        return Err(wire::err_unavailable());
    }
    if !p.exit_node_ip.is_empty() && !wire::parseable_addr(&p.exit_node_ip) {
        return Err(wire::err_unavailable());
    }
    Ok(p)
}

fn populate_host_preferences(p: &NativePrefs) -> Result<wire::HostPreferences, String> {
    let mut prefs = wire::HostPreferences {
        want_running: p.want_running,
        exit_node_id: p.exit_node_id.clone(),
        exit_node_ip: p.exit_node_ip.clone(),
        allow_lan: p.allow_lan,
        advertise_exit_node: false,
    };
    if let Some(routes) = &p.advertise_routes {
        for route in routes {
            if !wire::valid_prefix(route) {
                return Err(wire::err_unavailable());
            }
            if route == "0.0.0.0/0" || route == "::/0" {
                prefs.advertise_exit_node = true;
            }
        }
    }
    Ok(prefs)
}

/// Host revision: sha256 over Go's exact revision-document encoding.
/// Revision covers identity/state/preferences, not volatile peer or health
/// polling.
pub fn compute_host_revision(s: &NativeStatus, view: &wire::HostView, p: &NativePrefs) -> String {
    let q = crate::json::quote;
    let mut addrs = String::from("[");
    for (i, a) in view.addresses.iter().enumerate() {
        if i > 0 {
            addrs.push(',');
        }
        addrs.push_str(&q(a));
    }
    addrs.push(']');
    let mut routes = String::from("[");
    let mut routes_null = true;
    if let Some(list) = &p.advertise_routes {
        routes_null = false;
        for (i, r) in list.iter().enumerate() {
            if i > 0 {
                routes.push(',');
            }
            routes.push_str(&q(r));
        }
    }
    routes.push(']');
    let doc = format!(
        "{{\"State\":{},\"HaveNodeKey\":{},\"Expired\":{},\"DNSName\":{},\"ID\":{},\"Addresses\":{},\"Prefs\":{{\"WantRunning\":{},\"ExitNodeID\":{},\"ExitNodeIP\":{},\"ExitNodeAllowLANAccess\":{},\"AdvertiseRoutes\":{}}}}}",
        q(&s.backend_state),
        s.have_node_key,
        view.expired,
        q(&view.dns_name),
        q(s.self_peer.as_ref().map(|p| p.id.as_str()).unwrap_or("")),
        addrs,
        p.want_running,
        q(&p.exit_node_id),
        q(&p.exit_node_ip),
        p.allow_lan,
        if routes_null { "null".to_string() } else { routes },
    );
    let digest = crate::sha256::digest(doc.as_bytes());
    crate::sha256::hex_lower(&digest)
}

/// Observe the host view plus the raw native auth URL, mirroring `observe`.
pub fn observe(
    transport: RoundTrip<'_>,
    deadline: Instant,
) -> Result<(wire::HostView, String), String> {
    let s = fetch_native_status(transport, deadline)?;
    let p = fetch_native_prefs(transport, deadline)?;
    let prefs = populate_host_preferences(&p)?;
    let mut view = wire::HostView {
        state: s.backend_state.clone(),
        have_node_key: s.have_node_key,
        peers: Vec::new(),
        addresses: Vec::new(),
        health_issues: s.health_count as i64,
        preferences: prefs,
        ..Default::default()
    };
    if s.has_tailnet {
        view.tailnet = s.tailnet_name.clone();
        view.magic_dns_enabled = s.tailnet_magic_dns;
    }
    if let Some(peer) = &s.self_peer {
        let projected = wire::peer_view(
            &peer.id,
            &peer.dns_name,
            &peer.ips,
            peer.online,
            peer.exit_node_option,
            peer.expired,
        )?;
        view.dns_name = projected.dns_name;
        view.addresses = projected.addresses;
        view.expired = projected.expired;
    }
    let mut peers = Vec::with_capacity(s.peers.len());
    for peer in &s.peers {
        peers.push(wire::peer_view(
            &peer.id,
            &peer.dns_name,
            &peer.ips,
            peer.online,
            peer.exit_node_option,
            peer.expired,
        )?);
    }
    peers.sort_by(|a, b| a.id.cmp(&b.id));
    view.peers = peers;
    view.revision = compute_host_revision(&s, &view, &p);
    view.validate()?;
    Ok((view, s.auth_url))
}

// ---------- Command execution ----------

/// Post-exit capture grace for `RunNative`, mirroring Go's WaitDelay=1s:
/// after the leader exits, pipes must settle within a second or the
/// action reports unconfirmed (never truncated success).
const COMPLETION_GRACE: Duration = Duration::from_secs(1);

/// Bounded native command run, mirroring `RunNative`: stdout is capped at
/// 64 KiB, post-exit capture settles within the completion grace, and
/// every failure (spawn, deadline, exit status, overflow) is an
/// unconfirmed outcome with no native diagnostics attached.
pub fn run_command(
    exec: &dyn crate::project::Executor,
    path: &str,
    args: &[&str],
    deadline: Instant,
) -> Result<Vec<u8>, String> {
    match exec.run_with_capture_grace(&[], path, args, deadline, COMPLETION_GRACE) {
        Ok(out) if out.len() <= RESPONSE_LIMIT => Ok(out),
        _ => Err(wire::err_unconfirmed()),
    }
}

/// Decode complete bounded `tailscale up` notifications; never emit native
/// error text.
pub fn decode_up_notifications(data: &[u8]) -> Result<(), String> {
    for extent in split_json_values(data)? {
        let v = crate::json::decode_tolerant(&extent).map_err(|_| wire::err_unconfirmed())?;
        let fields = match v.as_object() {
            Some(f) => f,
            None => return Err(wire::err_unconfirmed()),
        };
        // Struct binding, like Go: every occurrence binds in order, any
        // mistyped occurrence fails, and any error text fails the decode.
        let mut error = String::new();
        for (k, v) in fields {
            if fold_eq(k, "Error") {
                match v {
                    Value::Null => {}
                    Value::Str(s) => error = s.clone(),
                    _ => return Err(wire::err_unconfirmed()),
                }
            } else if fold_eq(k, "AuthURL") {
                match v {
                    Value::Null | Value::Str(_) => {}
                    _ => return Err(wire::err_unconfirmed()),
                }
            }
        }
        if !error.is_empty() {
            return Err(wire::err_unconfirmed());
        }
    }
    Ok(())
}

/// Split consecutive whitespace-separated JSON values (Go `json.Decoder`
/// streaming). Trailing garbage is an error; empty input yields nothing.
fn split_json_values(data: &[u8]) -> Result<Vec<Vec<u8>>, String> {
    let text = std::str::from_utf8(data).map_err(|_| wire::err_unconfirmed())?;
    let bytes = text.as_bytes();
    let mut out = Vec::new();
    let mut i = 0;
    while i < bytes.len() {
        while i < bytes.len() && bytes[i].is_ascii_whitespace() {
            i += 1;
        }
        if i >= bytes.len() {
            break;
        }
        let start = i;
        match bytes[i] {
            b'{' | b'[' => {
                let open = bytes[i];
                let close = if open == b'{' { b'}' } else { b']' };
                let mut depth = 0;
                let mut in_string = false;
                let mut escape = false;
                while i < bytes.len() {
                    let c = bytes[i];
                    if in_string {
                        if escape {
                            escape = false;
                        } else if c == b'\\' {
                            escape = true;
                        } else if c == b'"' {
                            in_string = false;
                        }
                    } else if c == b'"' {
                        in_string = true;
                    } else if c == open {
                        depth += 1;
                    } else if c == close {
                        depth -= 1;
                        if depth == 0 {
                            i += 1;
                            break;
                        }
                    }
                    i += 1;
                }
                if depth != 0 || in_string {
                    return Err(wire::err_unconfirmed());
                }
            }
            b'"' => {
                i += 1;
                let mut escape = false;
                let mut closed = false;
                while i < bytes.len() {
                    let c = bytes[i];
                    if escape {
                        escape = false;
                    } else if c == b'\\' {
                        escape = true;
                    } else if c == b'"' {
                        closed = true;
                        i += 1;
                        break;
                    }
                    i += 1;
                }
                if !closed {
                    return Err(wire::err_unconfirmed());
                }
            }
            _ => {
                while i < bytes.len() && !bytes[i].is_ascii_whitespace() {
                    if matches!(bytes[i], b'{' | b'}' | b'[' | b']' | b'"' | b',') {
                        break;
                    }
                    i += 1;
                }
                if i == start {
                    return Err(wire::err_unconfirmed());
                }
            }
        }
        out.push(bytes[start..i].to_vec());
    }
    Ok(out)
}

// ---------- Host actions ----------

fn socket_arg(socket: &Path) -> String {
    format!("--socket={}", socket.display())
}

pub fn execute_signin(
    transport: RoundTrip<'_>,
    exec: &dyn crate::project::Executor,
    cli: &str,
    socket: &Path,
    before: &wire::HostView,
    deadline: Instant,
) -> Result<(), String> {
    if before.have_node_key {
        let patched = round_trip(
            transport,
            "PATCH",
            "prefs",
            Some(br#"{"WantRunning":true,"WantRunningSet":true}"#),
            deadline,
        );
        match patched {
            Ok(_) => {
                if before.state == "NeedsLogin" || before.expired {
                    round_trip(transport, "POST", "login-interactive", None, deadline)?;
                }
                Ok(())
            }
            Err(e) => Err(e),
        }
    } else {
        let wait = deadline.min(Instant::now() + UP_TIMEOUT);
        let arg = socket_arg(socket);
        let data = run_command(exec, cli, &[&arg, "up", "--json", "--timeout=5s"], wait)?;
        decode_up_notifications(&data)
    }
}

pub fn execute_logout(transport: RoundTrip<'_>, deadline: Instant) -> Result<(), String> {
    round_trip(transport, "POST", "logout", None, deadline)?;
    Ok(())
}

pub fn find_available_exit_node(peers: &[wire::Peer], target_ip: &str) -> Option<String> {
    for peer in peers {
        if peer.exit_node && peer.online && !peer.expired {
            for ip in &peer.addresses {
                if ip == target_ip {
                    return Some(peer.id.clone());
                }
            }
        }
    }
    None
}

pub fn execute_exit_node(
    exec: &dyn crate::project::Executor,
    cli: &str,
    socket: &Path,
    r: &wire::HostRequest,
    before: &wire::HostView,
    deadline: Instant,
) -> Result<String, String> {
    let want = r.exit_node.as_deref().unwrap_or("");
    let mut selected = String::new();
    if !want.is_empty() {
        match find_available_exit_node(&before.peers, want) {
            Some(id) => selected = id,
            None => return Err(wire::err_conflict()),
        }
    }
    let arg = socket_arg(socket);
    let set_ip = format!("--exit-node={want}");
    let lan = r.allow_lan.unwrap_or(false);
    let set_lan = format!("--exit-node-allow-lan-access={lan}");
    run_command(exec, cli, &[&arg, "set", &set_ip, &set_lan], deadline)?;
    Ok(selected)
}

pub fn execute_advertise_exit_node(
    exec: &dyn crate::project::Executor,
    cli: &str,
    socket: &Path,
    r: &wire::HostRequest,
    deadline: Instant,
) -> Result<(), String> {
    let arg = socket_arg(socket);
    let flag = r.advertise.unwrap_or(false);
    let set = format!("--advertise-exit-node={flag}");
    run_command(exec, cli, &[&arg, "set", &set], deadline)?;
    Ok(())
}

pub fn execute_refresh_forgejo(
    exec: &dyn crate::project::Executor,
    libexec: &str,
    deadline: Instant,
) -> Result<(), String> {
    run_command(
        exec,
        &format!("{libexec}/soda-forgejo-tailnet"),
        &[],
        deadline,
    )?;
    Ok(())
}

fn verify_exit_node(r: &wire::HostRequest, after: &wire::HostView, selected: &str) -> bool {
    // The native daemon upgrades the selected IP to its stable ID and clears
    // ExitNodeIP. Accept either native form; an empty requested IP must clear
    // both, never match a retained ID.
    let want = r.exit_node.as_deref().unwrap_or("");
    let mut matched =
        after.preferences.exit_node_id == selected && after.preferences.exit_node_ip.is_empty();
    if !want.is_empty() && after.preferences.exit_node_ip == want {
        matched = true;
    }
    matched && after.preferences.allow_lan == r.allow_lan.unwrap_or(false)
}

pub fn verify_host_action_outcome(
    r: &wire::HostRequest,
    after: &wire::HostView,
    selected: &str,
) -> bool {
    match r.action.as_str() {
        "signin" => after.preferences.want_running,
        "logout" => after.state == "NeedsLogin",
        "exit-node" => verify_exit_node(r, after, selected),
        "advertise-exit-node" => {
            after.preferences.advertise_exit_node == r.advertise.unwrap_or(false)
        }
        _ => true,
    }
}

/// Re-observe after a host action and classify the outcome, mirroring
/// `readbackHostAction`. `action_err` carries the execution failure, if any.
pub fn readback_host_action(
    transport: RoundTrip<'_>,
    r: &wire::HostRequest,
    selected: &str,
    action_err: Option<String>,
    deadline: Instant,
) -> wire::HostResult {
    let mut result = wire::HostResult {
        outcome: "confirmed".to_string(),
        ..Default::default()
    };
    match observe(transport, deadline) {
        Ok((after, auth)) => {
            if action_err.is_some() || !verify_host_action_outcome(r, &after, selected) {
                result.outcome = "unconfirmed".to_string();
            }
            if r.action == "signin" {
                result.auth_url = wire::authentication_url(&auth);
                if !result.auth_url.is_empty() {
                    result.outcome = "pending".to_string();
                }
            }
            result.host = Some(after);
        }
        Err(_) => {
            result.readback_unavailable = true;
            if action_err.is_some() {
                result.outcome = "unconfirmed".to_string();
            }
        }
    }
    result
}

// ---------- CLI client ----------
//
// Mirror of `tailnet.go`: the `tailscale status --json` reader used by the
// `soda-forgejo-tailnet` helper. Unlike the daemon paths, CLI failures carry
// the native diagnostic, exactly like Go.

/// Stable Soda interpretation of `tailscale status --json`.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct CliStatus {
    pub backend_state: String,
    pub identity: String,
    pub ipv4: String,
    pub magic_dns_enabled: bool,
    pub expired: bool,
    pub auth_pending: bool,
}

/// An enrolled appliance by resolvable name or IPv4 address.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct CliEndpoint {
    pub identity: String,
    pub ipv4: String,
}

fn last_field<'a>(fields: &'a [(String, Value)], name: &str) -> Option<&'a Value> {
    fields
        .iter()
        .rev()
        .find(|(k, _)| fold_eq(k, name))
        .map(|(_, v)| v)
}

fn tolerant_string(fields: &[(String, Value)], name: &str) -> Result<String, String> {
    match last_field(fields, name) {
        None | Some(Value::Null) => Ok(String::new()),
        Some(Value::Str(s)) => Ok(s.clone()),
        Some(_) => Err(wire::err_unavailable()),
    }
}

fn tolerant_bool(fields: &[(String, Value)], name: &str) -> Result<bool, String> {
    match last_field(fields, name) {
        None | Some(Value::Null) => Ok(false),
        Some(Value::Bool(b)) => Ok(*b),
        Some(_) => Err(wire::err_unavailable()),
    }
}

fn tolerant_object(fields: &[(String, Value)], name: &str) -> Result<Vec<(String, Value)>, String> {
    match last_field(fields, name) {
        None | Some(Value::Null) => Ok(Vec::new()),
        Some(Value::Object(inner)) => Ok(inner.clone()),
        Some(_) => Err(wire::err_unavailable()),
    }
}

fn tolerant_str_list(fields: &[(String, Value)], name: &str) -> Result<Vec<String>, String> {
    match last_field(fields, name) {
        None | Some(Value::Null) => Ok(Vec::new()),
        Some(Value::Array(items)) => {
            let mut out = Vec::with_capacity(items.len());
            for item in items {
                match item {
                    Value::Null => out.push(String::new()),
                    Value::Str(s) => out.push(s.clone()),
                    _ => return Err(wire::err_unavailable()),
                }
            }
            Ok(out)
        }
        Some(_) => Err(wire::err_unavailable()),
    }
}

fn parse_cli_status(data: &[u8]) -> Result<CliStatus, String> {
    // Go uses plain `encoding/json.Unmarshal`: unknown fields ignored, last
    // duplicate wins, mistyped values fail.
    let v = crate::json::decode_tolerant(data).map_err(|_| wire::err_unavailable())?;
    let fields = match v.as_object() {
        Some(f) => f,
        None => return Err(wire::err_unavailable()),
    };
    let backend = tolerant_string(fields, "BackendState")?;
    if backend.is_empty() {
        return Err(wire::err_unavailable());
    }
    let auth_pending = !tolerant_string(fields, "AuthURL")?.is_empty();
    let tailnet = tolerant_object(fields, "CurrentTailnet")?;
    let magic = tolerant_bool(&tailnet, "MagicDNSEnabled")?;
    let peer = tolerant_object(fields, "Self")?;
    let dns = tolerant_string(&peer, "DNSName")?;
    let expired = tolerant_bool(&peer, "Expired")?;
    let ips = tolerant_str_list(&peer, "TailscaleIPs")?;
    let mut status = CliStatus {
        backend_state: backend,
        magic_dns_enabled: magic,
        expired,
        auth_pending,
        ..Default::default()
    };
    if !dns.is_empty() {
        status.identity = wire::canonical_magic_dns_name(&dns)?;
    }
    if let Some(ip) = wire::first_ipv4(&ips) {
        status.ipv4 = ip;
    }
    Ok(status)
}

/// Read the local node's authoritative Tailscale status via the CLI.
pub fn cli_status(
    exec: &dyn crate::project::Executor,
    cli: &str,
    deadline: Instant,
) -> Result<CliStatus, String> {
    let output = match exec.run(&[], cli, &["status", "--json"], deadline) {
        Ok(out) => out,
        Err(e) => {
            return Err(format!(
                "{}: {cli} status --json: {e}",
                wire::err_unavailable()
            ))
        }
    };
    parse_cli_status(&output)
}

/// Resolve the advertised host and IPv4 address used by Forgejo.
pub fn cli_endpoint(
    exec: &dyn crate::project::Executor,
    cli: &str,
    deadline: Instant,
) -> Result<CliEndpoint, String> {
    let status = cli_status(exec, cli, deadline)?;
    if status.backend_state != "Running" || status.expired {
        return Err(wire::err_not_enrolled());
    }
    if status.ipv4.is_empty() {
        return Err(wire::err_ipv4_unavailable());
    }
    let identity = if status.magic_dns_enabled && !status.identity.is_empty() {
        status.identity.clone()
    } else {
        status.ipv4.clone()
    };
    Ok(CliEndpoint {
        identity,
        ipv4: status.ipv4,
    })
}

// ---------- Run status ----------
//
// Mirror of `project_runtime.go`'s `RunStatus` (currently without in-repo
// callers, like in Go).

/// Native-only run state projection: public binding metadata plus the
/// run's own connectivity. Never carries credentials or bearer tokens.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RunStatus {
    pub enabled: bool,
    pub admission: bool,
    pub tailnet: String,
    pub tags: Vec<String>,
    pub addresses: Vec<String>,
    pub dns_name: String,
}

impl RunStatus {
    /// Mirror of `RunStatus.Valid`.
    pub fn validate(&self) -> Result<(), String> {
        if !self.enabled && self.admission {
            return Err(wire::err_unavailable());
        }
        if !wire::valid_network(&self.tailnet) {
            return Err(wire::err_unavailable());
        }
        if wire::validate_enrollment_tags(&self.tags).is_err() {
            return Err(wire::err_unavailable());
        }
        wire::check_addresses(&self.addresses)?;
        if !self.dns_name.is_empty() {
            wire::canonical_magic_dns_name(&self.dns_name)?;
        }
        Ok(())
    }
}
