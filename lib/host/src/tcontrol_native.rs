//! Native tailscaled observation and host actions.
//!
//! PR26 port of the `control.go` LocalAPI flow (`request`, `observe`,
//! `execute*`, `readbackHostAction`, `decodeUpNotifications`), the native
//! document decoders (`nativeObject`, `peerView` binding), and the `tailnet.go`
//! CLI client (`Status`, `Endpoint`).
//!
//! LocalAPI uses the shared bounded Hyper HTTP/1 client over the tailscaled
//! Unix socket. Provider HTTPS is not here (see the provider module); command
//! execution goes through the crate's `Executor`.

use std::path::Path;
use std::time::{Duration, Instant};

use crate::json;
use crate::tcontrol_wire as wire;
use serde::de::{self, MapAccess, SeqAccess, Visitor};
use serde::{Deserialize, Deserializer};

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

/// HTTP/1.1 round trip over the tailscaled Unix socket.
pub fn local_request(
    socket: &Path,
    method: &str,
    path: &str,
    body: Option<&[u8]>,
    deadline: Instant,
) -> Result<(u16, Vec<u8>), String> {
    let payload = body.unwrap_or_default();
    let timeout = deadline
        .checked_duration_since(Instant::now())
        .filter(|remaining| !remaining.is_zero())
        .map(|remaining| remaining.min(LOCAL_TIMEOUT))
        .ok_or_else(wire::err_unavailable)?;
    let response = soda_unix_http::request(
        socket,
        method,
        &format!("/localapi/v0/{path}"),
        "local-tailscaled.sock",
        payload,
        timeout,
        soda_unix_http::Limits {
            header_bytes: 8192,
            body_bytes: RESPONSE_LIMIT,
        },
    )
    .map_err(|_| wire::err_unavailable())?;
    Ok((response.status, response.body))
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

#[derive(Debug, Clone, Default)]
pub struct NativePeer {
    pub id: String,
    pub dns_name: String,
    pub ips: Vec<String>,
    pub online: bool,
    pub exit_node_option: bool,
    pub expired: bool,
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

#[derive(Default)]
struct GoStringList(Vec<String>);
impl<'de> Deserialize<'de> for GoStringList {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = GoStringList;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("string list")
            }
            fn visit_unit<E>(self) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(GoStringList::default())
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                let mut values = Vec::new();
                while let Some(v) = seq.next_element::<Option<String>>()? {
                    values.push(v.unwrap_or_default());
                }
                Ok(GoStringList(values))
            }
        }
        d.deserialize_any(V)
    }
}

#[derive(Default)]
struct PeerWire {
    id: Option<String>,
    dns_name: Option<String>,
    ips: Option<GoStringList>,
    online: Option<bool>,
    exit_node_option: Option<bool>,
    expired: Option<bool>,
}
impl<'de> Deserialize<'de> for PeerWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = PeerWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("native peer")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = PeerWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    if fold_eq(&k, "ID") {
                        out.id = map.next_value()?;
                    } else if fold_eq(&k, "DNSName") {
                        out.dns_name = map.next_value()?;
                    } else if fold_eq(&k, "TailscaleIPs") {
                        out.ips = map.next_value()?;
                    } else if fold_eq(&k, "Online") {
                        out.online = map.next_value()?;
                    } else if fold_eq(&k, "ExitNodeOption") {
                        out.exit_node_option = map.next_value()?;
                    } else if fold_eq(&k, "Expired") {
                        out.expired = map.next_value()?;
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(out)
            }
        }
        d.deserialize_map(V)
    }
}
impl From<PeerWire> for NativePeer {
    fn from(v: PeerWire) -> Self {
        Self {
            id: v.id.unwrap_or_default(),
            dns_name: v.dns_name.unwrap_or_default(),
            ips: v.ips.map(|x| x.0).unwrap_or_default(),
            online: v.online.unwrap_or_default(),
            exit_node_option: v.exit_node_option.unwrap_or_default(),
            expired: v.expired.unwrap_or_default(),
        }
    }
}
fn merge_peer(out: &mut NativePeer, v: PeerWire) {
    if let Some(x) = v.id {
        out.id = x;
    }
    if let Some(x) = v.dns_name {
        out.dns_name = x;
    }
    if let Some(x) = v.ips {
        out.ips = x.0;
    }
    if let Some(x) = v.online {
        out.online = x;
    }
    if let Some(x) = v.exit_node_option {
        out.exit_node_option = x;
    }
    if let Some(x) = v.expired {
        out.expired = x;
    }
}

#[derive(Default)]
struct TailnetWire {
    name: Option<String>,
    magic: Option<bool>,
}
impl<'de> Deserialize<'de> for TailnetWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = TailnetWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("current tailnet")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = TailnetWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    if fold_eq(&k, "Name") {
                        o.name = map.next_value()?;
                    } else if fold_eq(&k, "MagicDNSEnabled") {
                        o.magic = map.next_value()?;
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct NativeStatusWire {
    value: NativeStatus,
    exact_backend: bool,
    exact_key_seen: bool,
    exact_key: bool,
    folded_key_seen: bool,
}
impl<'de> Deserialize<'de> for NativeStatusWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = NativeStatusWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("native status")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = NativeStatusWire::default();
                let mut peer_keys: Vec<String> = Vec::new();
                while let Some(k) = map.next_key::<String>()? {
                    if fold_eq(&k, "BackendState") {
                        let v: Option<String> = map.next_value()?;
                        if k == "BackendState" {
                            o.exact_backend = v.is_some();
                        }
                        if let Some(v) = v {
                            o.value.backend_state = v;
                        }
                    } else if fold_eq(&k, "HaveNodeKey") {
                        let v: Option<bool> = map.next_value()?;
                        o.folded_key_seen = true;
                        if k == "HaveNodeKey" {
                            o.exact_key_seen = true;
                            o.exact_key = v.is_some();
                        }
                        if let Some(v) = v {
                            o.value.have_node_key = v;
                        }
                    } else if fold_eq(&k, "AuthURL") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            o.value.auth_url = v;
                        }
                    } else if fold_eq(&k, "CurrentTailnet") {
                        if let Some(v) = map.next_value::<Option<TailnetWire>>()? {
                            o.value.has_tailnet = true;
                            if let Some(x) = v.name {
                                o.value.tailnet_name = x;
                            }
                            if let Some(x) = v.magic {
                                o.value.tailnet_magic_dns = x;
                            }
                        }
                    } else if fold_eq(&k, "Self") {
                        if let Some(v) = map.next_value::<Option<PeerWire>>()? {
                            let p = o.value.self_peer.get_or_insert_with(NativePeer::default);
                            merge_peer(p, v);
                        }
                    } else if fold_eq(&k, "Peer") {
                        if let Some(entries)=map.next_value::<Option<std::collections::BTreeMap<String,Option<PeerWire>>>>()?{for (key,v) in entries{let decoded=v.map(NativePeer::from).unwrap_or_default();match peer_keys.iter().position(|old|old==&key){Some(i)=>o.value.peers[i]=decoded,None=>{peer_keys.push(key);o.value.peers.push(decoded);}}}}
                    } else if fold_eq(&k, "Health") {
                        if let Some(v) = map.next_value::<Option<GoStringList>>()? {
                            o.value.health_count = v.0.len();
                        }
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                if !o.exact_backend
                    || (o.exact_key_seen && !o.exact_key)
                    || (o.folded_key_seen && !o.exact_key_seen)
                {
                    return Err(de::Error::custom(
                        "required exact native status field missing",
                    ));
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

pub fn decode_native_status(data: &[u8]) -> Result<NativeStatus, String> {
    let wire: NativeStatusWire =
        json::decode_strict_as(data).map_err(|_| wire::err_unavailable())?;
    Ok(wire.value)
}

#[derive(Debug, Clone, Default)]
pub struct NativePrefs {
    pub want_running: bool,
    pub exit_node_id: String,
    pub exit_node_ip: String,
    pub allow_lan: bool,
    pub advertise_routes: Option<Vec<String>>,
}

#[derive(Default)]
struct NativePrefsWire {
    value: NativePrefs,
    required: [bool; 5],
}
impl<'de> Deserialize<'de> for NativePrefsWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = NativePrefsWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("native preferences")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = NativePrefsWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    if fold_eq(&k, "WantRunning") {
                        let v: Option<bool> = map.next_value()?;
                        if k == "WantRunning" {
                            o.required[0] = v.is_some();
                        }
                        if let Some(v) = v {
                            o.value.want_running = v;
                        }
                    } else if fold_eq(&k, "ExitNodeID") {
                        let v: Option<String> = map.next_value()?;
                        if k == "ExitNodeID" {
                            o.required[1] = v.is_some();
                        }
                        if let Some(v) = v {
                            o.value.exit_node_id = v;
                        }
                    } else if fold_eq(&k, "ExitNodeIP") {
                        let v: Option<String> = map.next_value()?;
                        if k == "ExitNodeIP" {
                            o.required[2] = v.is_some();
                        }
                        if let Some(v) = v {
                            o.value.exit_node_ip = v;
                        }
                    } else if fold_eq(&k, "ExitNodeAllowLANAccess") {
                        let v: Option<bool> = map.next_value()?;
                        if k == "ExitNodeAllowLANAccess" {
                            o.required[3] = v.is_some();
                        }
                        if let Some(v) = v {
                            o.value.allow_lan = v;
                        }
                    } else if fold_eq(&k, "AdvertiseRoutes") {
                        let v: Option<GoStringList> = map.next_value()?;
                        if k == "AdvertiseRoutes" {
                            o.required[4] = true;
                        }
                        o.value.advertise_routes = v.map(|x| x.0);
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                if !o.required.iter().all(|v| *v) {
                    return Err(de::Error::custom(
                        "missing required exact native preference field",
                    ));
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}
pub fn decode_native_prefs(data: &[u8]) -> Result<NativePrefs, String> {
    let wire: NativePrefsWire =
        json::decode_strict_as(data).map_err(|_| wire::err_unavailable())?;
    Ok(wire.value)
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
        #[derive(Default)]
        struct NotificationWire {
            error: String,
        }
        impl<'de> Deserialize<'de> for NotificationWire {
            fn deserialize<D>(d: D) -> Result<Self, D::Error>
            where
                D: Deserializer<'de>,
            {
                struct V;
                impl<'de> Visitor<'de> for V {
                    type Value = NotificationWire;
                    fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        f.write_str("native notification")
                    }
                    fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
                    where
                        A: MapAccess<'de>,
                    {
                        let mut out = NotificationWire::default();
                        while let Some(k) = map.next_key::<String>()? {
                            if fold_eq(&k, "Error") {
                                if let Some(v) = map.next_value::<Option<String>>()? {
                                    out.error = v;
                                }
                            } else if fold_eq(&k, "AuthURL") {
                                map.next_value::<Option<String>>()?;
                            } else {
                                map.next_value::<de::IgnoredAny>()?;
                            }
                        }
                        Ok(out)
                    }
                }
                d.deserialize_map(V)
            }
        }
        let value: NotificationWire =
            json::decode_tolerant_as(&extent).map_err(|_| wire::err_unconfirmed())?;
        if !value.error.is_empty() {
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

#[derive(Default)]
struct CliTailnetWire {
    magic: bool,
}
impl<'de> Deserialize<'de> for CliTailnetWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = CliTailnetWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("tailnet status")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = CliTailnetWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    if fold_eq(&k, "MagicDNSEnabled") {
                        o.magic = map.next_value::<Option<bool>>()?.unwrap_or_default();
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}
#[derive(Default)]
struct CliPeerWire {
    dns: String,
    expired: bool,
    ips: Vec<String>,
}
impl<'de> Deserialize<'de> for CliPeerWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = CliPeerWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("self peer status")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = CliPeerWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    if fold_eq(&k, "DNSName") {
                        o.dns = map.next_value::<Option<String>>()?.unwrap_or_default();
                    } else if fold_eq(&k, "Expired") {
                        o.expired = map.next_value::<Option<bool>>()?.unwrap_or_default();
                    } else if fold_eq(&k, "TailscaleIPs") {
                        o.ips = map
                            .next_value::<Option<GoStringList>>()?
                            .map(|v| v.0)
                            .unwrap_or_default();
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}
#[derive(Default)]
struct CliStatusWire {
    backend: String,
    auth: String,
    tailnet: CliTailnetWire,
    peer: CliPeerWire,
}
impl<'de> Deserialize<'de> for CliStatusWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = CliStatusWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("CLI status")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = CliStatusWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    if fold_eq(&k, "BackendState") {
                        o.backend = map.next_value::<Option<String>>()?.unwrap_or_default();
                    } else if fold_eq(&k, "AuthURL") {
                        o.auth = map.next_value::<Option<String>>()?.unwrap_or_default();
                    } else if fold_eq(&k, "CurrentTailnet") {
                        o.tailnet = map
                            .next_value::<Option<CliTailnetWire>>()?
                            .unwrap_or_default();
                    } else if fold_eq(&k, "Self") {
                        o.peer = map.next_value::<Option<CliPeerWire>>()?.unwrap_or_default();
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

fn parse_cli_status(data: &[u8]) -> Result<CliStatus, String> {
    // Go uses plain `encoding/json.Unmarshal`: unknown fields ignored, last
    // duplicate wins, mistyped values fail.
    let decoded: CliStatusWire =
        json::decode_tolerant_as(data).map_err(|_| wire::err_unavailable())?;
    let backend = decoded.backend;
    if backend.is_empty() {
        return Err(wire::err_unavailable());
    }
    let auth_pending = !decoded.auth.is_empty();
    let magic = decoded.tailnet.magic;
    let dns = decoded.peer.dns;
    let expired = decoded.peer.expired;
    let ips = decoded.peer.ips;
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
