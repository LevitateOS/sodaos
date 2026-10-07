//! Tailnet control-plane wire layer: errors, patterns, DTOs, strict decoders,
//! validators and Go-exact JSON encoders.
//!
//! PR26 port of `internal/tailnet/control_types.go` (DTO shapes plus the
//! `Validate` methods), `internal/tailnet/control_validation.go` (view
//! validators), the address/DNS/peer/URL primitives used by `control.go`
//! (`authenticationURL`, `addresses`, `peerView`, `CanonicalMagicDNSName`),
//! and the daemon error-to-status mapping in `internal/host/tailnet.go`
//! (`tailnetErrorStatus`).
//!
//! Wire JSON field order, `omitempty` rules and string escaping match
//! `encoding/json` exactly; see the oracle tests for differential vectors.
//!
//! This module is dependency-free (`std` plus the sibling crate modules) and
//! performs no I/O.

use crate::json;
use crate::tailnet_domain::{self, ERR_CONFLICT, ERR_INVALID, ERR_UNAVAILABLE, ERR_UNCONFIRMED};
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

// ---------- Adapter error strings ----------
//
// The daemon adapter maps errors to HTTP statuses by substring, mirroring
// `tailnetErrorStatus`: `invalid request` (400), `conflict` (409),
// `unsupported` (422), `unavailable` (503); anything else is 502. Each
// constructor below embeds both the adapter keyword and the matching Go
// `tailnet.Err*` text so oracle runs can compare against Go behaviour.

/// Mirror of `tailnet.ErrInvalid` (adapter: 400).
pub fn err_invalid() -> String {
    format!("invalid request: {ERR_INVALID}")
}

/// Mirror of `tailnet.ErrConflict` (adapter: 409).
pub fn err_conflict() -> String {
    format!("conflict: {ERR_CONFLICT}")
}

/// Mirror of `tailnet.ErrUnsupported` (adapter: 422).
pub fn err_unsupported() -> String {
    format!("unsupported: {}", tailnet_domain::ERR_UNSUPPORTED)
}

/// Mirror of `tailnet.ErrUnavailable` (adapter: 503).
pub fn err_unavailable() -> String {
    format!("unavailable: {ERR_UNAVAILABLE}")
}

/// Mirror of `tailnet.ErrUnconfirmed` (adapter: 502 via the default branch).
pub fn err_unconfirmed() -> String {
    ERR_UNCONFIRMED.to_string()
}

/// Mirror of `tailnet.ErrNotEnrolled`, for the CLI client surface only.
pub fn err_not_enrolled() -> String {
    "tailscale is not enrolled".to_string()
}

/// Mirror of `tailnet.ErrIPv4Unavailable`, for the CLI client surface only.
pub fn err_ipv4_unavailable() -> String {
    "tailscale did not report an IPv4 address".to_string()
}

/// Best-effort in-place zeroing of a secret string, mirroring the explicit
/// `Secret = ""` assignments in Go. Clears this allocation only; other copies
/// (JSON buffers, curl stdin) are cleared at their own use sites.
pub fn zero_string(s: &mut String) {
    let bytes = unsafe { s.as_bytes_mut() };
    for b in bytes.iter_mut() {
        *b = 0;
    }
    s.clear();
}

/// Wire request cap. The Go daemon wraps route bodies in a 64 KiB
/// `http.MaxBytesReader` before `strictjson.Decode`, so anything larger is an
/// invalid request regardless of the 1 MiB strictjson cap.
pub const WIRE_BODY_LIMIT: usize = 65536;

/// Wire response cap, mirroring `writeTailnetResponse` (oversize is a 502).
pub const WIRE_RESPONSE_LIMIT: usize = 65536;

// ---------- Validation patterns ----------

fn is_lower_hex(s: &str) -> bool {
    s.bytes().all(|b| matches!(b, b'0'..=b'9' | b'a'..=b'f'))
}

fn is_token_char(b: u8) -> bool {
    b.is_ascii_alphanumeric() || b == b'_' || b == b'-'
}

/// `^[0-9a-f]{32}$`
pub fn is_hex32(s: &str) -> bool {
    s.len() == 32 && is_lower_hex(s)
}

/// `validRevision`: `"0"` or `^[0-9a-f]{32}$`.
pub fn valid_revision(s: &str) -> bool {
    s == "0" || is_hex32(s)
}

/// `^tag:[a-zA-Z][a-zA-Z0-9-]{0,62}$`
pub fn valid_tag(s: &str) -> bool {
    let rest = match s.strip_prefix("tag:") {
        Some(r) => r,
        None => return false,
    };
    let b = rest.as_bytes();
    if b.is_empty() || b.len() > 63 || !b[0].is_ascii_alphabetic() {
        return false;
    }
    b[1..]
        .iter()
        .all(|c| c.is_ascii_alphanumeric() || *c == b'-')
}

fn valid_tskey(prefix: &str, s: &str) -> bool {
    match s.strip_prefix(prefix) {
        Some(r) => (8..=512).contains(&r.len()) && r.bytes().all(is_token_char),
        None => false,
    }
}

/// `^tskey-client-[A-Za-z0-9_-]{8,512}$`
pub fn valid_client_secret(s: &str) -> bool {
    valid_tskey("tskey-client-", s)
}

/// `^tskey-auth-[A-Za-z0-9_-]{8,512}$`
pub fn valid_auth_key(s: &str) -> bool {
    valid_tskey("tskey-auth-", s)
}

/// `^[A-Za-z0-9_-]{1,128}$`
pub fn valid_client_id(s: &str) -> bool {
    (1..=128).contains(&s.len()) && s.bytes().all(is_token_char)
}

/// `^[a-zA-Z0-9][a-zA-Z0-9.@_-]{0,252}$`
pub fn valid_network(s: &str) -> bool {
    let b = s.as_bytes();
    if b.is_empty() || b.len() > 253 || !b[0].is_ascii_alphanumeric() {
        return false;
    }
    b[1..]
        .iter()
        .all(|c| c.is_ascii_alphanumeric() || matches!(c, b'.' | b'@' | b'_' | b'-'))
}

// ---------- MagicDNS names ----------

fn lower_char(c: char) -> char {
    c.to_lowercase().next().unwrap_or(c)
}

fn valid_label(label: &str) -> bool {
    let b = label.as_bytes();
    if b.is_empty() || b.len() > 63 || b[0] == b'-' || b[b.len() - 1] == b'-' {
        return false;
    }
    b.iter()
        .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || *c == b'-')
}

/// Mirror of `CanonicalMagicDNSName` (`tailnet.go`).
pub fn canonical_magic_dns_name(value: &str) -> Result<String, String> {
    // `strings.TrimSpace` is Unicode White_Space plus U+0085 (NEL); the extra
    // arm keeps parity however Rust classifies the Latin-1 special case.
    let trimmed = value.trim_matches(|c: char| c.is_whitespace() || c == '\u{85}');
    let dotted = trimmed.strip_suffix('.').unwrap_or(trimmed);
    let name: String = dotted.chars().map(lower_char).collect();
    if name.len() > 253 || !name.contains('.') || name.ends_with(".local") {
        return Err(err_unavailable());
    }
    for label in name.split('.') {
        if !valid_label(label) {
            return Err(err_unavailable());
        }
    }
    Ok(name)
}

// ---------- IP addresses ----------
//
// Mirror of the `netip` subset `control.go` relies on: `ParseAddr` for the
// forms `addresses` can accept (including zones on IPv6 only), `Addr.String`
// canonicalisation, `IsGlobalUnicast`, `Is4`, and `ParsePrefix` (host bits
// accepted, zones rejected, bits range-checked; verified against the pinned
// toolchain with probes).

use std::net::{IpAddr, Ipv4Addr, Ipv6Addr};

struct ParsedAddr {
    addr: IpAddr,
    zone: Option<String>,
}

fn parse_addr(s: &str) -> Option<ParsedAddr> {
    let (head, zone) = match s.find('%') {
        Some(i) => {
            let zone = s.get(i + 1..)?;
            if zone.is_empty() {
                return None;
            }
            (s.get(..i)?, Some(zone.to_owned()))
        }
        None => (s, None),
    };
    if head.contains(':') {
        Some(ParsedAddr {
            addr: head.parse::<Ipv6Addr>().ok()?.into(),
            zone,
        })
    } else if zone.is_none() {
        Some(ParsedAddr {
            addr: head.parse::<Ipv4Addr>().ok()?.into(),
            zone: None,
        })
    } else {
        None
    }
}

fn v4_global_unicast(ip: Ipv4Addr) -> bool {
    let octets = ip.octets();
    !ip.is_unspecified()
        && !ip.is_loopback()
        && !ip.is_multicast()
        && ip != Ipv4Addr::BROADCAST
        && !(octets[0] == 169 && octets[1] == 254)
}

fn is_global_unicast(a: &ParsedAddr) -> bool {
    match a.addr {
        IpAddr::V4(ip) => v4_global_unicast(ip),
        IpAddr::V6(ip) => match ip.to_ipv4_mapped() {
            Some(v4) => v4_global_unicast(v4),
            None => {
                !ip.is_unspecified()
                    && !ip.is_loopback()
                    && !ip.is_multicast()
                    && ip.segments()[0] & 0xffc0 != 0xfe80
            }
        },
    }
}

/// Mirror of `addresses` in control.go.
pub fn check_addresses(v: &[String]) -> Result<Vec<String>, String> {
    if v.len() > 16 {
        return Err(err_unavailable());
    }
    for s in v {
        let a = parse_addr(s).ok_or_else(err_unavailable)?;
        if !is_global_unicast(&a) || a.canonical() != *s {
            return Err(err_unavailable());
        }
    }
    Ok(v.to_vec())
}

impl ParsedAddr {
    fn canonical(&self) -> String {
        let mut value = self.addr.to_string();
        if let Some(zone) = &self.zone {
            value.push('%');
            value.push_str(zone);
        }
        value
    }
}

/// Single-address check for exit-node requests: canonical form and global
/// unicast, mirroring `validateHostExitNode`.
pub fn valid_exit_node_ip(s: &str) -> bool {
    parse_addr(s).is_some_and(|a| is_global_unicast(&a) && a.canonical() == s)
}

/// Parse-only check for native `ExitNodeIP` preferences, mirroring
/// `fetchNativePrefs`.
pub fn parseable_addr(s: &str) -> bool {
    parse_addr(s).is_some()
}

/// Mirror of `netip.ParsePrefix` acceptance (host bits accepted, zones
/// rejected, bits range-checked), for native `AdvertiseRoutes`.
pub fn valid_prefix(s: &str) -> bool {
    let Some((head, bits)) = s.rsplit_once('/') else {
        return false;
    };
    if head.is_empty()
        || head.contains('%')
        || bits.is_empty()
        || !bits.bytes().all(|c| c.is_ascii_digit())
    {
        return false;
    }
    let Ok(bits) = bits.parse::<u32>() else {
        return false;
    };
    match head.parse::<IpAddr>() {
        Ok(IpAddr::V4(_)) => bits <= 32,
        Ok(IpAddr::V6(_)) => bits <= 128,
        Err(_) => false,
    }
}

/// First true IPv4 address in canonical form; mapped IPv6 remains IPv6.
pub fn first_ipv4(ips: &[String]) -> Option<String> {
    ips.iter().find_map(|s| match parse_addr(s) {
        Some(ParsedAddr {
            addr: IpAddr::V4(ip),
            ..
        }) => Some(ip.to_string()),
        _ => None,
    })
}

// ---------- DTOs ----------
//
// Field order matches the Go structs; encoders below preserve it.

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct HostPreferences {
    pub want_running: bool,
    pub exit_node_id: String,
    pub exit_node_ip: String,
    pub allow_lan: bool,
    pub advertise_exit_node: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Peer {
    pub id: String,
    pub dns_name: String,
    pub addresses: Vec<String>,
    pub online: bool,
    pub exit_node: bool,
    pub expired: bool,
}

/// Mirror of `peerView` in control.go.
pub fn peer_view(
    id: &str,
    dns_name: &str,
    ips: &[String],
    online: bool,
    exit_node: bool,
    expired: bool,
) -> Result<Peer, String> {
    if id.len() > 128 || id.contains(['\r', '\n', '\0']) {
        return Err(err_unavailable());
    }
    let mut name = String::new();
    if !dns_name.is_empty() {
        name = canonical_magic_dns_name(dns_name)?;
    }
    let addresses = check_addresses(ips)?;
    Ok(Peer {
        id: id.to_string(),
        dns_name: name,
        addresses,
        online,
        exit_node,
        expired,
    })
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct HostView {
    pub tailnet: String,
    pub magic_dns_enabled: bool,
    pub revision: String,
    pub state: String,
    pub have_node_key: bool,
    pub expired: bool,
    pub dns_name: String,
    pub addresses: Vec<String>,
    pub peers: Vec<Peer>,
    pub health_issues: i64,
    pub preferences: HostPreferences,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct EnrollmentView {
    pub revision: String,
    pub binding: String,
    pub tailnet: String,
    pub tags: Vec<String>,
    pub configured: bool,
    pub admission: bool,
    pub default: bool,
    pub preauthorized: bool,
    pub credential_checked: bool,
    pub enrollment_verified: bool,
    pub runtime_supported: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SettingsView {
    pub host: Option<HostView>,
    pub host_unavailable: bool,
    pub enrollment: EnrollmentView,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct HostRequest {
    pub action: String,
    pub revision: String,
    pub confirm: String,
    pub exit_node: Option<String>,
    pub allow_lan: Option<bool>,
    pub advertise: Option<bool>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct HostResult {
    pub outcome: String,
    pub host: Option<HostView>,
    pub readback_unavailable: bool,
    pub auth_url: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct EnrollmentRequest {
    pub action: String,
    pub revision: String,
    pub tailnet: String,
    pub tags: Option<Vec<String>>,
    pub preauthorized: Option<bool>,
    pub client_id: String,
    pub client_secret: String,
    pub default: Option<bool>,
}

impl EnrollmentRequest {
    /// Zero the transient client secret, mirroring Go's `Secret = ""`
    /// assignments after credential use.
    pub fn zero_secret(&mut self) {
        zero_string(&mut self.client_secret);
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct EnrollmentResult {
    pub outcome: String,
    pub saved: bool,
    pub credential_checked: bool,
    pub enrollment: EnrollmentView,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProjectOptions {
    pub revision: String,
    pub binding: String,
    pub tailnet: String,
    pub available: bool,
    pub default: bool,
}

/// Mirror of `ProjectSelection` (`project_runtime.go`): the human-reviewed
/// create-time selection decoded by the web API.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProjectSelection {
    pub enabled: bool,
    pub revision: String,
    pub binding: String,
}

macro_rules! decode_folded_field {
    ($key:expr, $label:literal, $map:expr, $field:expr) => {
        if $key.eq_ignore_ascii_case($label) {
            if let Some(value) = $map.next_value::<Option<_>>()? {
                $field = value;
            }
            continue;
        }
    };
}

macro_rules! decode_folded_optional {
    ($key:expr, $label:literal, $map:expr, $field:expr) => {
        if $key.eq_ignore_ascii_case($label) {
            if let Some(value) = $map.next_value::<Option<_>>()? {
                $field = Some(value);
            }
            continue;
        }
    };
}

impl<'de> Deserialize<'de> for HostRequest {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct RequestVisitor;
        impl<'de> Visitor<'de> for RequestVisitor {
            type Value = HostRequest;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a host request object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut request = HostRequest::default();
                while let Some(key) = map.next_key::<String>()? {
                    decode_folded_field!(key, "action", map, request.action);
                    decode_folded_field!(key, "revision", map, request.revision);
                    decode_folded_field!(key, "confirm", map, request.confirm);
                    decode_folded_optional!(key, "exit_node", map, request.exit_node);
                    decode_folded_optional!(key, "allow_lan", map, request.allow_lan);
                    decode_folded_optional!(key, "advertise", map, request.advertise);
                    return Err(de::Error::unknown_field(
                        &key,
                        &[
                            "action",
                            "revision",
                            "confirm",
                            "exit_node",
                            "allow_lan",
                            "advertise",
                        ],
                    ));
                }
                Ok(request)
            }
        }
        deserializer.deserialize_map(RequestVisitor)
    }
}

impl<'de> Deserialize<'de> for EnrollmentRequest {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct RequestVisitor;
        impl<'de> Visitor<'de> for RequestVisitor {
            type Value = EnrollmentRequest;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an enrollment request object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut request = EnrollmentRequest::default();
                while let Some(key) = map.next_key::<String>()? {
                    decode_folded_field!(key, "action", map, request.action);
                    decode_folded_field!(key, "revision", map, request.revision);
                    decode_folded_field!(key, "tailnet", map, request.tailnet);
                    decode_folded_optional!(key, "tags", map, request.tags);
                    decode_folded_optional!(key, "preauthorized", map, request.preauthorized);
                    decode_folded_field!(key, "client_id", map, request.client_id);
                    decode_folded_field!(key, "client_secret", map, request.client_secret);
                    decode_folded_optional!(key, "default", map, request.default);
                    return Err(de::Error::unknown_field(
                        &key,
                        &[
                            "action",
                            "revision",
                            "tailnet",
                            "tags",
                            "preauthorized",
                            "client_id",
                            "client_secret",
                            "default",
                        ],
                    ));
                }
                Ok(request)
            }
        }
        deserializer.deserialize_map(RequestVisitor)
    }
}

impl<'de> Deserialize<'de> for ProjectSelection {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct SelectionVisitor;
        impl<'de> Visitor<'de> for SelectionVisitor {
            type Value = ProjectSelection;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a project selection object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut selection = ProjectSelection::default();
                while let Some(key) = map.next_key::<String>()? {
                    decode_folded_field!(key, "enabled", map, selection.enabled);
                    decode_folded_field!(key, "revision", map, selection.revision);
                    decode_folded_field!(key, "binding", map, selection.binding);
                    return Err(de::Error::unknown_field(
                        &key,
                        &["enabled", "revision", "binding"],
                    ));
                }
                Ok(selection)
            }
        }
        deserializer.deserialize_map(SelectionVisitor)
    }
}

// ---------- Strict wire decoders ----------

fn strict_body<T: serde::de::DeserializeOwned>(body: &[u8]) -> Result<T, String> {
    if body.len() > WIRE_BODY_LIMIT {
        return Err(err_invalid());
    }
    crate::json::decode_strict_as(body).map_err(|_| err_invalid())
}

/// Strict-decode a `HostRequest` body, mirroring `decodeTailnetBody` plus the
/// `HostRequest` struct binding (unknown fields rejected).
pub fn decode_host_request(body: &[u8]) -> Result<HostRequest, String> {
    strict_body(body)
}

/// Strict-decode an `EnrollmentRequest` body.
pub fn decode_enrollment_request(body: &[u8]) -> Result<EnrollmentRequest, String> {
    strict_body(body)
}

/// Strict-decode a `ProjectSelection` body.
pub fn decode_project_selection(body: &[u8]) -> Result<ProjectSelection, String> {
    strict_body(body)
}

// ---------- Request validators ----------

impl HostRequest {
    fn has_extra_fields(&self) -> bool {
        self.exit_node.is_some() || self.allow_lan.is_some() || self.advertise.is_some()
    }

    fn validate_signin(&self) -> Result<(), String> {
        if !self.confirm.is_empty() || self.has_extra_fields() {
            return Err(err_invalid());
        }
        Ok(())
    }

    fn validate_confirmed(&self) -> Result<(), String> {
        if self.confirm != self.action || self.has_extra_fields() {
            return Err(err_invalid());
        }
        Ok(())
    }

    fn validate_exit_node(&self) -> Result<(), String> {
        if self.confirm != self.action
            || self.exit_node.is_none()
            || self.allow_lan.is_none()
            || self.advertise.is_some()
        {
            return Err(err_invalid());
        }
        let exit = self.exit_node.as_deref().unwrap_or_default();
        let lan = self.allow_lan.unwrap_or(false);
        if exit.is_empty() {
            if lan {
                return Err(err_invalid());
            }
            return Ok(());
        }
        if !valid_exit_node_ip(exit) {
            return Err(err_invalid());
        }
        Ok(())
    }

    fn validate_advertise(&self) -> Result<(), String> {
        if self.confirm != self.action
            || self.advertise.is_none()
            || self.exit_node.is_some()
            || self.allow_lan.is_some()
        {
            return Err(err_invalid());
        }
        Ok(())
    }

    /// Mirror of `HostRequest.Validate`.
    pub fn validate(&self) -> Result<(), String> {
        if !tailnet_domain::valid_container_id(&self.revision) {
            return Err(err_invalid());
        }
        match self.action.as_str() {
            "signin" | "authentication" => self.validate_signin(),
            "logout" | "refresh-forgejo" => self.validate_confirmed(),
            "exit-node" => self.validate_exit_node(),
            "advertise-exit-node" => self.validate_advertise(),
            _ => Err(err_invalid()),
        }
    }
}

pub(crate) fn validate_enrollment_tags(tags: &[String]) -> Result<(), String> {
    if tags.is_empty() || tags.len() > 8 {
        return Err(err_invalid());
    }
    let mut prev: Option<&str> = None;
    for t in tags {
        if !valid_tag(t) {
            return Err(err_invalid());
        }
        // Go: `slices.IsSorted` plus an adjacent-duplicate ban. Together the
        // tags must be strictly increasing.
        if let Some(p) = prev {
            if p >= t.as_str() {
                return Err(err_invalid());
            }
        }
        prev = Some(t);
    }
    Ok(())
}

fn validate_enrollment_policy(
    tailnet: &str,
    tags: &[String],
    preauthorized: Option<bool>,
) -> Result<(), String> {
    if !valid_network(tailnet) || tailnet.contains("..") || preauthorized.is_none() {
        return Err(err_invalid());
    }
    validate_enrollment_tags(tags)
}

impl EnrollmentRequest {
    fn validate_mutation(&self) -> Result<(), String> {
        if !valid_client_id(&self.client_id)
            || !valid_client_secret(&self.client_secret)
            || self.default.is_some()
        {
            return Err(err_invalid());
        }
        validate_enrollment_policy(
            &self.tailnet,
            self.tags.as_deref().unwrap_or(&[]),
            self.preauthorized,
        )
    }

    fn has_payload(&self) -> bool {
        !self.client_id.is_empty()
            || !self.client_secret.is_empty()
            || !self.tailnet.is_empty()
            || self.tags.is_some()
            || self.preauthorized.is_some()
    }

    fn validate_toggle(&self) -> Result<(), String> {
        if self.has_payload() {
            return Err(err_invalid());
        }
        if self.action == "default" && self.default.is_none() {
            return Err(err_invalid());
        }
        if self.action == "disable" && self.default.is_some() {
            return Err(err_invalid());
        }
        Ok(())
    }

    /// Mirror of `EnrollmentRequest.Validate`.
    pub fn validate(&self) -> Result<(), String> {
        if !valid_revision(&self.revision) {
            return Err(err_invalid());
        }
        match self.action.as_str() {
            "save" | "check" | "rotate" => self.validate_mutation(),
            "default" | "disable" => self.validate_toggle(),
            _ => Err(err_invalid()),
        }
    }
}

/// Mirror of `ProjectRequest.Validate` for `tailnet_domain::ProjectRequest`.
///
/// Internal only: the daemon adapter owns the adapter-facing project wire
/// shape; this enforces Go's `Control.Project` argument checks.
pub(crate) fn check_project_request(r: &tailnet_domain::ProjectRequest) -> Result<(), String> {
    if !tailnet_domain::valid_project_id(&r.project) {
        return Err(err_invalid());
    }
    if r.action == "inspect" {
        if !r.revision.is_empty() || !r.binding.is_empty() || !r.confirm_id.is_empty() {
            return Err(err_invalid());
        }
        return Ok(());
    }
    if !valid_revision(&r.revision) || r.confirm_id != r.project {
        return Err(err_invalid());
    }
    match r.action.as_str() {
        "disable" => {
            if !r.binding.is_empty() {
                return Err(err_invalid());
            }
        }
        "enable" | "retry" => {
            if !is_hex32(&r.binding) {
                return Err(err_invalid());
            }
        }
        _ => return Err(err_invalid()),
    }
    Ok(())
}

impl ProjectSelection {
    /// Mirror of `ProjectSelection.Validate`.
    pub fn validate(&self) -> Result<(), String> {
        if !self.enabled {
            if !self.revision.is_empty() || !self.binding.is_empty() {
                return Err(err_invalid());
            }
        } else if !is_hex32(&self.revision) || !is_hex32(&self.binding) {
            return Err(err_invalid());
        }
        Ok(())
    }
}

// ---------- Auth-URL filter ----------

/// Mirror of `authenticationURL` in control.go: admit only
/// `https://login.tailscale.com/a/<client-pattern>` with no userinfo, port,
/// query, fragment or escapes. Returns the normalized form (`url.String()`,
/// which lowercases the scheme) or `""`.
///
/// Probe-verified `url.Parse` edges: an uppercase scheme parses and
/// normalizes to lowercase, the host is case-sensitive, a port/userinfo/space
/// either fails the field checks or sets `RawPath`.
pub fn authentication_url(raw: &str) -> String {
    if raw.len() > 2048 || raw.bytes().any(|b| b < 0x20 || b == 0x7f) {
        return String::new();
    }
    let after_scheme = match raw.split_once("://") {
        Some((scheme, rest)) if scheme.eq_ignore_ascii_case("https") => rest,
        _ => return String::new(),
    };
    let rest = match after_scheme.strip_prefix("login.tailscale.com") {
        Some(r) => r,
        None => return String::new(),
    };
    // Authority must end exactly at the host: no userinfo (`@` would precede
    // it, but the literal host match already excludes that), no port.
    let path = match rest.strip_prefix("/a/") {
        Some(p) => p,
        None => return String::new(),
    };
    if path.is_empty() || !valid_client_id(path) {
        return String::new();
    }
    // The client-charset path needs no escaping, so `RawPath` is empty and
    // `u.String()` is the input with a lowercased scheme.
    format!("https://login.tailscale.com/a/{path}")
}

// ---------- View validators ----------

impl EnrollmentView {
    fn valid_identity(&self) -> bool {
        if !valid_revision(&self.revision) || self.enrollment_verified {
            return false;
        }
        // Rust has no nil slice: an empty tag list on a fresh view is the
        // `Tags: []string{}` fixture, which Go accepts.
        !self.default || (self.configured && self.admission)
    }

    /// Mirror of `EnrollmentView.Validate`.
    pub fn validate(&self) -> Result<(), String> {
        if !self.valid_identity() {
            return Err(err_unavailable());
        }
        if !self.configured {
            let empty_binding =
                self.revision == "0" && self.binding.is_empty() && self.tailnet.is_empty();
            let idle = !self.admission && !self.credential_checked && !self.preauthorized;
            if !(empty_binding && self.tags.is_empty() && idle) {
                return Err(err_unavailable());
            }
            return Ok(());
        }
        if !is_hex32(&self.revision) || !is_hex32(&self.binding) || !self.credential_checked {
            return Err(err_unavailable());
        }
        let probe = EnrollmentRequest {
            action: "save".to_string(),
            revision: self.revision.clone(),
            tailnet: self.tailnet.clone(),
            tags: Some(self.tags.clone()),
            preauthorized: Some(self.preauthorized),
            client_id: "validation".to_string(),
            client_secret: "tskey-client-validation-only".to_string(),
            default: None,
        };
        if probe.validate().is_err() {
            return Err(err_unavailable());
        }
        Ok(())
    }
}

impl HostView {
    fn valid_tailnet(&self) -> bool {
        if !self.tailnet.is_empty() {
            return valid_network(&self.tailnet);
        }
        !self.magic_dns_enabled && self.state != "Running"
    }

    fn valid_inventory(&self) -> bool {
        tailnet_domain::valid_container_id(&self.revision)
            && self.peers.len() <= 128
            && self.health_issues >= 0
            && self.health_issues <= 128
    }

    fn valid_backend_state(&self) -> bool {
        matches!(
            self.state.as_str(),
            "NoState"
                | "InUseOtherUser"
                | "NeedsLogin"
                | "NeedsMachineAuth"
                | "Stopped"
                | "Starting"
                | "Running"
        )
    }

    fn valid_peers(&self) -> Result<(), String> {
        peer_view("", &self.dns_name, &self.addresses, false, false, false)?;
        let mut seen = std::collections::HashSet::new();
        for p in &self.peers {
            if p.id.is_empty() || !seen.insert(p.id.clone()) {
                return Err(err_unavailable());
            }
            peer_view(&p.id, &p.dns_name, &p.addresses, false, false, false)?;
        }
        Ok(())
    }

    fn valid_exit_node(&self) -> Result<(), String> {
        if self.preferences.exit_node_id.len() > 128
            || self.preferences.exit_node_id.contains(['\r', '\n', '\0'])
        {
            return Err(err_unavailable());
        }
        if !self.preferences.exit_node_ip.is_empty() {
            check_addresses(std::slice::from_ref(&self.preferences.exit_node_ip))?;
        }
        Ok(())
    }

    /// Mirror of `HostView.Validate`.
    pub fn validate(&self) -> Result<(), String> {
        if !self.valid_tailnet() || !self.valid_inventory() || !self.valid_backend_state() {
            return Err(err_unavailable());
        }
        self.valid_peers()?;
        self.valid_exit_node()
    }
}

impl SettingsView {
    /// Mirror of `SettingsView.Validate`.
    pub fn validate(&self) -> Result<(), String> {
        if self.enrollment.validate().is_err() || self.host.is_none() != self.host_unavailable {
            return Err(err_unavailable());
        }
        if let Some(h) = &self.host {
            return h.validate();
        }
        Ok(())
    }
}

impl HostResult {
    /// Mirror of `HostResult.Validate`.
    pub fn validate(&self) -> Result<(), String> {
        match self.outcome.as_str() {
            "confirmed" | "unconfirmed" | "pending" | "observed" => {}
            _ => return Err(err_unavailable()),
        }
        if self.host.is_none() != self.readback_unavailable {
            return Err(err_unavailable());
        }
        if !self.auth_url.is_empty() && authentication_url(&self.auth_url) != self.auth_url {
            return Err(err_unavailable());
        }
        if self.outcome == "pending" && self.auth_url.is_empty() {
            return Err(err_unavailable());
        }
        if let Some(h) = &self.host {
            return h.validate();
        }
        Ok(())
    }
}

impl EnrollmentResult {
    /// Mirror of `EnrollmentResult.Validate`.
    pub fn validate(&self) -> Result<(), String> {
        if self.outcome != "confirmed" || (!self.credential_checked && self.saved) {
            return Err(err_unavailable());
        }
        self.enrollment.validate()
    }
}

impl ProjectOptions {
    /// Mirror of `ProjectOptions.Validate`.
    pub fn validate(&self) -> Result<(), String> {
        if !valid_revision(&self.revision)
            || (self.default && !self.available)
            || (self.available && self.revision == "0")
        {
            return Err(err_unavailable());
        }
        if self.revision == "0" {
            if !self.binding.is_empty() || !self.tailnet.is_empty() {
                return Err(err_unavailable());
            }
        } else if !is_hex32(&self.binding) || !valid_network(&self.tailnet) {
            return Err(err_unavailable());
        }
        Ok(())
    }
}

fn valid_project_availability(binding: &str, network: &str) -> bool {
    if binding.is_empty() != network.is_empty() {
        return false;
    }
    if binding.is_empty() {
        return true;
    }
    is_hex32(binding) && valid_network(network)
}

fn valid_project_identity(v: &tailnet_domain::ProjectView) -> bool {
    tailnet_domain::valid_project_id(&v.project)
        && valid_revision(&v.revision)
        && (v.binding.is_empty() || is_hex32(&v.binding))
        && (!v.enabled || !v.binding.is_empty())
}

fn valid_project_persistence(v: &tailnet_domain::ProjectView) -> bool {
    if v.saved {
        return v.revision != "0"
            && matches!(
                v.outcome.as_str(),
                "disconnect-unconfirmed" | "runtime-unconfirmed" | "queued"
            );
    }
    v.outcome == "observed"
}

fn valid_project_runtime(v: &tailnet_domain::ProjectView) -> Result<(), String> {
    match v.state.as_str() {
        "runtime-unsupported"
        | "off"
        | "stopped"
        | "pending"
        | "needs-login"
        | "approval-required"
        | "unconfirmed" => {
            if !v.addresses.is_empty() || !v.dns_name.is_empty() {
                return Err(err_unavailable());
            }
        }
        "connected" => {
            if !v.enabled || v.addresses.is_empty() {
                return Err(err_unavailable());
            }
            peer_view("", &v.dns_name, &v.addresses, false, false, false)?;
        }
        _ => return Err(err_unavailable()),
    }
    Ok(())
}

/// Mirror of `ProjectView.Validate` for `tailnet_domain::ProjectView`.
pub fn validate_project_view(v: &tailnet_domain::ProjectView) -> Result<(), String> {
    if !valid_project_availability(&v.available_binding, &v.available_network)
        || !valid_project_identity(v)
        || !valid_project_persistence(v)
    {
        return Err(err_unavailable());
    }
    valid_project_runtime(v)
}

// ---------- Go-exact JSON encoders ----------
//
// Field order follows the Go structs; `omitempty` drops empty strings/lists;
// strings use `encoding/json` escaping (`json::quote`).

fn push_comma(out: &mut String, first: &mut bool) {
    if *first {
        *first = false;
    } else {
        out.push(',');
    }
}

fn push_str(out: &mut String, first: &mut bool, name: &str, val: &str) {
    push_comma(out, first);
    out.push_str(&json::quote(name));
    out.push(':');
    out.push_str(&json::quote(val));
}

fn push_bool(out: &mut String, first: &mut bool, name: &str, val: bool) {
    push_comma(out, first);
    out.push_str(&json::quote(name));
    out.push(':');
    out.push_str(if val { "true" } else { "false" });
}

fn push_int(out: &mut String, first: &mut bool, name: &str, val: i64) {
    push_comma(out, first);
    out.push_str(&json::quote(name));
    out.push(':');
    out.push_str(&val.to_string());
}

fn push_str_list(out: &mut String, first: &mut bool, name: &str, vals: &[String]) {
    push_comma(out, first);
    out.push_str(&json::quote(name));
    out.push_str(":[");
    for (i, v) in vals.iter().enumerate() {
        if i > 0 {
            out.push(',');
        }
        out.push_str(&json::quote(v));
    }
    out.push(']');
}

impl HostPreferences {
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        push_bool(out, &mut first, "want_running", self.want_running);
        push_str(out, &mut first, "exit_node_id", &self.exit_node_id);
        push_str(out, &mut first, "exit_node_ip", &self.exit_node_ip);
        push_bool(out, &mut first, "allow_lan", self.allow_lan);
        push_bool(
            out,
            &mut first,
            "advertise_exit_node",
            self.advertise_exit_node,
        );
        out.push('}');
    }
}

impl Peer {
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        push_str(out, &mut first, "id", &self.id);
        push_str(out, &mut first, "dns_name", &self.dns_name);
        push_str_list(out, &mut first, "addresses", &self.addresses);
        push_bool(out, &mut first, "online", self.online);
        push_bool(out, &mut first, "exit_node", self.exit_node);
        push_bool(out, &mut first, "expired", self.expired);
        out.push('}');
    }
}

impl HostView {
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        push_str(out, &mut first, "tailnet", &self.tailnet);
        push_bool(out, &mut first, "magic_dns_enabled", self.magic_dns_enabled);
        push_str(out, &mut first, "revision", &self.revision);
        push_str(out, &mut first, "state", &self.state);
        push_bool(out, &mut first, "have_node_key", self.have_node_key);
        push_bool(out, &mut first, "expired", self.expired);
        push_str(out, &mut first, "dns_name", &self.dns_name);
        push_str_list(out, &mut first, "addresses", &self.addresses);
        push_comma(out, &mut first);
        out.push_str("\"peers\":[");
        for (i, p) in self.peers.iter().enumerate() {
            if i > 0 {
                out.push(',');
            }
            p.encode_into(out);
        }
        out.push(']');
        push_int(out, &mut first, "health_issues", self.health_issues);
        push_comma(out, &mut first);
        out.push_str("\"preferences\":");
        self.preferences.encode_into(out);
        out.push('}');
    }
}

impl EnrollmentView {
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        push_str(out, &mut first, "revision", &self.revision);
        push_str(out, &mut first, "binding", &self.binding);
        push_str(out, &mut first, "tailnet", &self.tailnet);
        push_str_list(out, &mut first, "tags", &self.tags);
        push_bool(out, &mut first, "configured", self.configured);
        push_bool(out, &mut first, "admission", self.admission);
        push_bool(out, &mut first, "default", self.default);
        push_bool(out, &mut first, "preauthorized", self.preauthorized);
        push_bool(
            out,
            &mut first,
            "credential_checked",
            self.credential_checked,
        );
        push_bool(
            out,
            &mut first,
            "enrollment_verified",
            self.enrollment_verified,
        );
        push_bool(out, &mut first, "runtime_supported", self.runtime_supported);
        out.push('}');
    }
}

impl SettingsView {
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        push_comma(out, &mut first);
        out.push_str("\"host\":");
        match &self.host {
            Some(h) => h.encode_into(out),
            None => out.push_str("null"),
        }
        push_bool(out, &mut first, "host_unavailable", self.host_unavailable);
        push_comma(out, &mut first);
        out.push_str("\"enrollment\":");
        self.enrollment.encode_into(out);
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

impl HostResult {
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        push_str(out, &mut first, "outcome", &self.outcome);
        push_comma(out, &mut first);
        out.push_str("\"host\":");
        match &self.host {
            Some(h) => h.encode_into(out),
            None => out.push_str("null"),
        }
        push_bool(
            out,
            &mut first,
            "readback_unavailable",
            self.readback_unavailable,
        );
        if !self.auth_url.is_empty() {
            push_str(out, &mut first, "auth_url", &self.auth_url);
        }
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

impl EnrollmentResult {
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        push_str(out, &mut first, "outcome", &self.outcome);
        push_bool(out, &mut first, "saved", self.saved);
        push_bool(
            out,
            &mut first,
            "credential_checked",
            self.credential_checked,
        );
        push_comma(out, &mut first);
        out.push_str("\"enrollment\":");
        self.enrollment.encode_into(out);
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

impl ProjectOptions {
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        push_str(out, &mut first, "revision", &self.revision);
        push_str(out, &mut first, "binding", &self.binding);
        push_str(out, &mut first, "tailnet", &self.tailnet);
        push_bool(out, &mut first, "available", self.available);
        push_bool(out, &mut first, "default", self.default);
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}
