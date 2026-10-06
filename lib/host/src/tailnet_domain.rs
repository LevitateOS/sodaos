//! Tailnet domain mirrors: error texts, DTOs, matchers, time/JSON helpers.
//! Lane A owns this file.
//!
//! Pure port of the `internal/tailnet` project-status surface: `ProjectHasNode`
//! and `ProjectStatus` (`project_status.go`), the `nativeObject` / `peerView` /
//! `addresses` helpers (`control.go`), `CanonicalMagicDNSName` (`tailnet.go`),
//! the project DTOs (`control_types.go`), `RunTarget` (`enrollment.go`) and
//! `RunBinding` (`project_runtime.go`). Go stdlib edge semantics (JSON,
//! `netip`, `time`) were verified against the pinned toolchain with probes.

use crate::domain;
use crate::json::{self, Value};

pub const ERR_INVALID: &str = "invalid Tailnet request";
pub const ERR_CONFLICT: &str = "tailnet revision or identity changed";
pub const ERR_UNSUPPORTED: &str = "tailnet runtime is not supported";
pub const ERR_UNCONFIRMED: &str = "tailnet outcome is unconfirmed";
pub const ERR_UNAVAILABLE: &str = "tailscale status is unavailable";

/// Native LocalAPI body cap (`responseLimit` in control.go).
const RESPONSE_LIMIT: usize = 65536;

pub type ReadFile = Box<dyn Fn(&str) -> Result<Vec<u8>, String>>;
pub type LinkFile = Box<dyn Fn(&str) -> Result<String, String>>;
pub type StatFile = Box<dyn Fn(&str) -> Result<std::fs::Metadata, String>>;

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RunTarget {
    pub project: String,
    pub container: String,
    pub run: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RunBinding {
    pub enabled: bool,
    pub admission: bool,
    pub tailnet: String,
    pub tags: Vec<String>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProjectRequest {
    pub project: String,
    pub action: String,
    pub revision: String,
    pub binding: String,
    pub confirm_id: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProjectView {
    pub available_binding: String,
    pub available_network: String,
    pub addresses: Vec<String>,
    pub dns_name: String,
    pub saved: bool,
    pub project: String,
    pub revision: String,
    pub binding: String,
    pub enabled: bool,
    pub state: String,
    pub outcome: String,
}

/// `^p[0-9a-f]{24}$`
pub fn valid_project_id(s: &str) -> bool {
    domain::valid_id(s)
}

/// `^[0-9a-f]{64}$`
pub fn valid_container_id(s: &str) -> bool {
    domain::valid_container_id(s)
}

/// `^(sha256:)?[0-9a-f]{64}$`
pub fn valid_image_id(s: &str) -> bool {
    domain::valid_image_ref(s)
}

fn unavailable<T>() -> Result<T, String> {
    Err(ERR_UNAVAILABLE.to_string())
}

fn digits(b: &[u8], lo: usize, hi: usize) -> Option<i64> {
    if hi > b.len() {
        return None;
    }
    let mut v: i64 = 0;
    for &c in &b[lo..hi] {
        if !c.is_ascii_digit() {
            return None;
        }
        v = v * 10 + (c - b'0') as i64;
    }
    Some(v)
}

/// Proleptic-Gregorian days from 0001-01-01 (negative for year 0 dates).
fn days_since_year_one(year: i64, month: i64, day: i64) -> i64 {
    let y = if month <= 2 { year - 1 } else { year };
    let era = y.div_euclid(400);
    let yoe = y.rem_euclid(400);
    let mp = (month + 9).rem_euclid(12);
    let doy = (153 * mp + 2) / 5 + day - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146097 + doe - 719468 + 719162
}

/// Strict `time.Parse(time.RFC3339Nano)`: `None` for invalid input, else
/// `Some(is_zero)` where zero is the instant 0001-01-01T00:00:00Z.
/// Grammar: `YYYY-MM-DDTHH:MM:SS[.f+](Z|±HH:MM)` with a real calendar
/// date, H/HH 00-23, MM/SS 00-59, 1+ fraction digits (Go keeps nanosecond
/// precision and ignores the rest), offset HH 00-24 and MM 00-60
/// (Go's per-field offset bounds, verified by probe).
pub fn parse_rfc3339_nano(s: &str) -> Option<bool> {
    let b = s.as_bytes();
    if b.len() < 19 {
        return None;
    }
    let year = digits(b, 0, 4)?;
    if b.get(4) != Some(&b'-') {
        return None;
    }
    let month = digits(b, 5, 7)?;
    if b.get(7) != Some(&b'-') {
        return None;
    }
    let day = digits(b, 8, 10)?;
    if b.get(10) != Some(&b'T') {
        return None;
    }
    // Go's hour layout takes 1-2 digits; every other field is fixed width.
    let mut hpos = 11;
    while hpos < b.len() && b[hpos].is_ascii_digit() && hpos - 11 < 2 {
        hpos += 1;
    }
    if hpos == 11 || b.get(hpos) != Some(&b':') {
        return None;
    }
    let hour = digits(b, 11, hpos)?;
    let minute = digits(b, hpos + 1, hpos + 3)?;
    if b.get(hpos + 3) != Some(&b':') {
        return None;
    }
    let second = digits(b, hpos + 4, hpos + 6)?;
    let mut pos = hpos + 6;
    let mut nanos: i64 = 0;
    if b.get(pos) == Some(&b'.') {
        pos += 1;
        let start = pos;
        while pos < b.len() && b[pos].is_ascii_digit() {
            pos += 1;
        }
        let len = pos - start;
        if len < 1 {
            return None;
        }
        let mut scale = 100_000_000i64;
        for &c in &b[start..start + len.min(9)] {
            nanos += (c - b'0') as i64 * scale;
            scale /= 10;
        }
    }
    let rest = b.get(pos..)?;
    let offset_secs: i64 = if rest.len() == 1 && rest[0] == b'Z' {
        0
    } else if rest.len() == 6
        && (rest[0] == b'+' || rest[0] == b'-')
        && rest[3] == b':'
        && rest[1].is_ascii_digit()
        && rest[2].is_ascii_digit()
        && rest[4].is_ascii_digit()
        && rest[5].is_ascii_digit()
    {
        let hh = ((rest[1] - b'0') as i64) * 10 + (rest[2] - b'0') as i64;
        let mm = ((rest[4] - b'0') as i64) * 10 + (rest[5] - b'0') as i64;
        if hh > 24 || mm > 60 {
            return None;
        }
        let total = hh * 3600 + mm * 60;
        if rest[0] == b'-' {
            -total
        } else {
            total
        }
    } else {
        return None;
    };
    if !(1..=12).contains(&month) || hour > 23 || minute > 59 || second > 59 {
        return None;
    }
    let leap = year % 4 == 0 && (year % 100 != 0 || year % 400 == 0);
    let dim = match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        _ => {
            if leap {
                29
            } else {
                28
            }
        }
    };
    if day < 1 || day > dim {
        return None;
    }
    let abs = days_since_year_one(year, month, day) * 86_400 + hour * 3600 + minute * 60 + second
        - offset_secs;
    Some(abs == 0 && nanos == 0)
}

/// `encoding/json` string escaping with surrounding double quotes.
pub fn go_escape(s: &str) -> String {
    json::quote(s)
}

/// Single-rune lowercasing like Go's `unicode.ToLower`: the first rune of the
/// full mapping, never an expansion.
fn lower_char(c: char) -> char {
    c.to_lowercase().next().unwrap_or(c)
}

/// One rune's field-name fold representative. The known-field side is always
/// an ASCII name, so only ASCII fold relations matter: ASCII case plus the
/// two non-ASCII runes Go links to ASCII letters, `ſ` (U+017F, with S/s) and
/// `K` (U+212A Kelvin, with K/k). Notably `İ` (U+0130) does NOT fold with
/// `i` in Go. All other non-ASCII runes compare by identity.
fn fold_char(c: char) -> char {
    match c {
        'ſ' => 's',
        '\u{212a}' => 'k',
        x if x.is_ascii_alphabetic() => x.to_ascii_lowercase(),
        x => x,
    }
}

/// Field-name case folding matching `strings.EqualFold` whenever one side is
/// an ASCII field name (the only use: JSON keys against known ASCII names).
fn fold_eq(a: &str, b: &str) -> bool {
    if a == b {
        return true;
    }
    debug_assert!(a.is_ascii() || b.is_ascii(), "fold needs an ASCII side");
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

/// Mirror of `nativeObject`: strict single-object decode with duplicate
/// rejection at every level (Go's `strictjson.Decode`), exact-name required
/// field presence with the `HaveNodeKey` omission rule, then `encoding/json`
/// binding performed by the caller's typed decoder. Returns the document-order
/// value so case-variant keys bind last-wins exactly like Go.
fn native_object(data: &[u8], required: &[&str]) -> Result<Value, String> {
    if json::decode_strict(data).is_err() {
        return unavailable();
    }
    // A strict success implies tolerant success (same grammar, fewer bans).
    let v = match json::decode_tolerant(data) {
        Ok(v) => v,
        Err(_) => return unavailable(),
    };
    let fields = match v.as_object() {
        Some(f) => f,
        None => return unavailable(),
    };
    for key in required {
        if !required_native_field(fields, key) {
            return unavailable();
        }
    }
    Ok(v)
}

/// Mirror of `requiredNativeField` + `haveNodeKeyOmitted`.
fn required_native_field(fields: &[(String, Value)], key: &str) -> bool {
    match fields.iter().find(|(k, _)| k == key) {
        None => {
            if key != "HaveNodeKey" {
                return false;
            }
            !fields.iter().any(|(k, _)| fold_eq(k, key))
        }
        Some((_, v)) => !v.is_null() || key == "AdvertiseRoutes",
    }
}

/// All values bound to one struct field: exact or fold-equal keys in document
/// order. `encoding/json` applies each in order: the last non-null value wins,
/// `null` is a no-op, and any mistyped value fails the whole decode.
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
            _ => return unavailable(),
        }
    }
    Ok(())
}

fn bind_bool_into(fields: &[(String, Value)], name: &str, out: &mut bool) -> Result<(), String> {
    for v in bound_values(fields, name) {
        match v {
            Value::Null => {}
            Value::Bool(b) => *out = *b,
            _ => return unavailable(),
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
                        _ => return unavailable(),
                    }
                }
                *out = next;
            }
            _ => return unavailable(),
        }
    }
    Ok(())
}

#[derive(Debug, Clone, Default)]
struct SelfPeer {
    id: String,
    dns_name: String,
    ips: Vec<String>,
    tags: Vec<String>,
    online: bool,
    expired: bool,
}

/// Pointer-struct binding merges across case-variant keys into one struct.
fn bind_self(fields: &[(String, Value)]) -> Result<Option<SelfPeer>, String> {
    let mut out: Option<SelfPeer> = None;
    for v in bound_values(fields, "Self") {
        match v {
            Value::Null => {}
            Value::Object(inner) => {
                let p = out.get_or_insert_with(SelfPeer::default);
                bind_string_into(inner, "ID", &mut p.id)?;
                bind_string_into(inner, "DNSName", &mut p.dns_name)?;
                bind_list_into(inner, "TailscaleIPs", &mut p.ips)?;
                bind_list_into(inner, "Tags", &mut p.tags)?;
                bind_bool_into(inner, "Online", &mut p.online)?;
                bind_bool_into(inner, "Expired", &mut p.expired)?;
            }
            _ => return unavailable(),
        }
    }
    Ok(out)
}

fn bind_tailnet(fields: &[(String, Value)]) -> Result<Option<String>, String> {
    let mut out: Option<String> = None;
    for v in bound_values(fields, "CurrentTailnet") {
        match v {
            Value::Null => {}
            Value::Object(inner) => {
                let slot = out.get_or_insert_with(String::new);
                bind_string_into(inner, "Name", slot)?;
            }
            _ => return unavailable(),
        }
    }
    Ok(out)
}

#[derive(Debug, Clone, Default)]
struct NativeStatus {
    backend_state: String,
    have_node_key: bool,
    tailnet: Option<String>,
    peer: Option<SelfPeer>,
}

fn decode_native_status(v: &Value) -> Result<NativeStatus, String> {
    let fields = match v.as_object() {
        Some(f) => f,
        None => return unavailable(),
    };
    let mut s = NativeStatus::default();
    bind_string_into(fields, "BackendState", &mut s.backend_state)?;
    bind_bool_into(fields, "HaveNodeKey", &mut s.have_node_key)?;
    s.tailnet = bind_tailnet(fields)?;
    s.peer = bind_self(fields)?;
    Ok(s)
}

#[derive(Debug, Clone, Default)]
struct NativePrefs {
    want_running: bool,
    corp_dns: bool,
    route_all: bool,
    run_ssh: bool,
    exit_node_id: String,
    exit_node_ip: String,
    advertise_routes: Vec<String>,
}

fn decode_native_prefs(v: &Value) -> Result<NativePrefs, String> {
    let fields = match v.as_object() {
        Some(f) => f,
        None => return unavailable(),
    };
    let mut p = NativePrefs::default();
    bind_bool_into(fields, "WantRunning", &mut p.want_running)?;
    bind_bool_into(fields, "CorpDNS", &mut p.corp_dns)?;
    bind_bool_into(fields, "RouteAll", &mut p.route_all)?;
    bind_bool_into(fields, "RunSSH", &mut p.run_ssh)?;
    bind_string_into(fields, "ExitNodeID", &mut p.exit_node_id)?;
    bind_string_into(fields, "ExitNodeIP", &mut p.exit_node_ip)?;
    bind_list_into(fields, "AdvertiseRoutes", &mut p.advertise_routes)?;
    Ok(p)
}

fn valid_label(label: &str) -> bool {
    let b = label.as_bytes();
    if b.is_empty() || b.len() > 63 || b[0] == b'-' || b[b.len() - 1] == b'-' {
        return false;
    }
    b.iter()
        .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || *c == b'-')
}

/// Mirror of `CanonicalMagicDNSName`.
fn canonical_magic_dns_name(value: &str) -> Result<String, String> {
    // `strings.TrimSpace` is Unicode White_Space plus U+0085 (NEL); Rust's
    // `char::is_whitespace` omits the Latin-1 special case.
    let trimmed = value.trim_matches(|c: char| c.is_whitespace() || c == '\u{85}');
    let dotted = trimmed.strip_suffix('.').unwrap_or(trimmed);
    let name: String = dotted.chars().map(lower_char).collect();
    if name.len() > 253 || !name.contains('.') || name.ends_with(".local") {
        return unavailable();
    }
    for label in name.split('.') {
        if !valid_label(label) {
            return unavailable();
        }
    }
    Ok(name)
}

enum Addr {
    V4([u8; 4]),
    V6([u16; 8]),
}

struct ParsedAddr {
    addr: Addr,
    zone: Option<String>,
}

/// Mirror of `netip.ParseAddr` for the forms `addresses` can accept.
fn parse_addr(s: &str) -> Option<ParsedAddr> {
    // Zone splits at the first `%`; any non-empty zone is kept verbatim.
    let (head, zone) = match s.find('%') {
        Some(i) => {
            let z = s.get(i + 1..)?;
            if z.is_empty() {
                return None;
            }
            (s.get(..i)?, Some(z.to_string()))
        }
        None => (s, None),
    };
    if head.contains(':') {
        Some(ParsedAddr {
            addr: Addr::V6(parse_ipv6(head)?),
            zone,
        })
    } else {
        if zone.is_some() {
            return None;
        }
        Some(ParsedAddr {
            addr: Addr::V4(parse_ipv4(head)?),
            zone: None,
        })
    }
}

fn parse_ipv4(s: &str) -> Option<[u8; 4]> {
    let mut out = [0u8; 4];
    let mut parts = s.split('.');
    for slot in out.iter_mut() {
        let p = parts.next()?;
        if p.is_empty() || p.len() > 3 || !p.bytes().all(|c| c.is_ascii_digit()) {
            return None;
        }
        if p.len() > 1 && p.as_bytes()[0] == b'0' {
            return None;
        }
        let v: u32 = p.parse().ok()?;
        if v > 255 {
            return None;
        }
        *slot = v as u8;
    }
    if parts.next().is_some() {
        return None;
    }
    Some(out)
}

fn parse_h16(g: &str) -> Option<u16> {
    if g.is_empty() || g.len() > 4 || !g.bytes().all(|c| c.is_ascii_hexdigit()) {
        return None;
    }
    u16::from_str_radix(g, 16).ok()
}

fn parse_ipv6(s: &str) -> Option<[u16; 8]> {
    // An embedded dotted quad is allowed only as the final 32 bits.
    let (head, tail): (&str, Option<[u16; 2]>) = if s.contains('.') {
        let (h, dotted) = s.rsplit_once(':')?;
        let v4 = parse_ipv4(dotted)?;
        let hi = ((v4[0] as u16) << 8) | v4[1] as u16;
        let lo = ((v4[2] as u16) << 8) | v4[3] as u16;
        (h, Some([hi, lo]))
    } else {
        (s, None)
    };
    let extra = if tail.is_some() { 2 } else { 0 };
    let mut groups: Vec<u16> = Vec::with_capacity(8);
    match head.find("::") {
        None => {
            if head.is_empty() {
                return None;
            }
            for g in head.split(':') {
                groups.push(parse_h16(g)?);
            }
            if groups.len() + extra != 8 {
                return None;
            }
        }
        Some(at) => {
            let left = head.get(..at)?;
            let right = head.get(at + 2..)?;
            if right.contains("::") {
                return None;
            }
            if !left.is_empty() {
                for g in left.split(':') {
                    groups.push(parse_h16(g)?);
                }
            }
            let left_len = groups.len();
            let mut right_groups: Vec<u16> = Vec::new();
            if !right.is_empty() {
                for g in right.split(':') {
                    right_groups.push(parse_h16(g)?);
                }
            }
            if left_len + right_groups.len() + extra >= 8 {
                return None;
            }
            groups.extend(std::iter::repeat_n(
                0,
                8 - extra - left_len - right_groups.len(),
            ));
            groups.extend(right_groups);
        }
    }
    if let Some(t) = tail {
        groups.push(t[0]);
        groups.push(t[1]);
    }
    if groups.len() != 8 {
        return None;
    }
    let mut out = [0u16; 8];
    out.copy_from_slice(&groups);
    Some(out)
}

fn is_4in6(g: &[u16; 8]) -> bool {
    g[0] == 0 && g[1] == 0 && g[2] == 0 && g[3] == 0 && g[4] == 0 && g[5] == 0xffff
}

/// RFC 5952 §4.2.3 longest-run compression: runs under length 2 are left in
/// place and the first run wins ties, exactly like `netip.Addr.String`.
fn compress_v6(g: &[u16; 8]) -> String {
    let mut best_at = 0usize;
    let mut best_len = 0usize;
    let mut i = 0;
    while i < 8 {
        if g[i] == 0 {
            let mut j = i;
            while j < 8 && g[j] == 0 {
                j += 1;
            }
            if j - i > best_len {
                best_at = i;
                best_len = j - i;
            }
            i = j;
        } else {
            i += 1;
        }
    }
    if best_len < 2 {
        return g
            .iter()
            .map(|x| format!("{x:x}"))
            .collect::<Vec<_>>()
            .join(":");
    }
    let left = g[..best_at]
        .iter()
        .map(|x| format!("{x:x}"))
        .collect::<Vec<_>>()
        .join(":");
    let right = g[best_at + best_len..]
        .iter()
        .map(|x| format!("{x:x}"))
        .collect::<Vec<_>>()
        .join(":");
    if left.is_empty() && right.is_empty() {
        return "::".to_string();
    }
    if left.is_empty() {
        return format!("::{right}");
    }
    if right.is_empty() {
        return format!("{left}::");
    }
    format!("{left}::{right}")
}

impl ParsedAddr {
    /// Mirror of `netip.Addr.String` (zone kept verbatim).
    fn canonical(&self) -> String {
        let mut s = match &self.addr {
            Addr::V4(v) => format!("{}.{}.{}.{}", v[0], v[1], v[2], v[3]),
            Addr::V6(g) => {
                if is_4in6(g) {
                    format!(
                        "::ffff:{}.{}.{}.{}",
                        (g[6] >> 8) as u8,
                        g[6] as u8,
                        (g[7] >> 8) as u8,
                        g[7] as u8
                    )
                } else {
                    compress_v6(g)
                }
            }
        };
        if let Some(z) = &self.zone {
            s.push('%');
            s.push_str(z);
        }
        s
    }
}

fn v4_global_unicast(v: &[u8; 4]) -> bool {
    if *v == [0, 0, 0, 0] || *v == [255, 255, 255, 255] {
        return false;
    }
    if v[0] == 127 || v[0] & 0xf0 == 0xe0 {
        return false;
    }
    if v[0] == 169 && v[1] == 254 {
        return false;
    }
    true
}

/// Mirror of `netip.Addr.IsGlobalUnicast` (probed: private/CGNAT/reserved
/// pass; only unspecified, broadcast, loopback, link-local and multicast
/// fail; 4-in-6 follows the mapped v4 address).
fn is_global_unicast(a: &ParsedAddr) -> bool {
    match &a.addr {
        Addr::V4(v) => v4_global_unicast(v),
        Addr::V6(g) => {
            if is_4in6(g) {
                let v = [(g[6] >> 8) as u8, g[6] as u8, (g[7] >> 8) as u8, g[7] as u8];
                return v4_global_unicast(&v);
            }
            if g.iter().all(|&x| x == 0) {
                return false;
            }
            if g[7] == 1 && g[..7].iter().all(|&x| x == 0) {
                return false;
            }
            if g[0] & 0xff00 == 0xff00 || g[0] & 0xffc0 == 0xfe80 {
                return false;
            }
            true
        }
    }
}

/// Mirror of `addresses` in control.go.
fn check_addresses(v: &[String]) -> Result<Vec<String>, String> {
    if v.len() > 16 {
        return unavailable();
    }
    for s in v {
        let a = match parse_addr(s) {
            Some(a) => a,
            None => return unavailable(),
        };
        if !is_global_unicast(&a) || a.canonical() != *s {
            return unavailable();
        }
    }
    Ok(v.to_vec())
}

/// Mirror of `peerView` (ID empty on the project path; kept for parity).
fn peer_view(id: &str, dns_name: &str, ips: &[String]) -> Result<(Vec<String>, String), String> {
    if id.len() > 128 || id.contains(['\r', '\n', '\0']) {
        return unavailable();
    }
    let mut name = String::new();
    if !dns_name.is_empty() {
        name = canonical_magic_dns_name(dns_name)?;
    }
    let addrs = check_addresses(ips)?;
    Ok((addrs, name))
}

/// Mirror of `resolveProjectPeer`.
fn resolve_project_peer(dns_name: &str, ips: &[String]) -> Result<(Vec<String>, String), String> {
    let (addrs, name) = peer_view("", dns_name, ips)?;
    if addrs.is_empty() {
        return unavailable();
    }
    Ok((addrs, name))
}

/// Mirror of `parseProjectStatus`: the decoded status plus a terminal outcome
/// (`""` means Running, continue to preferences and binding checks).
fn parse_project_status(data: &[u8]) -> Result<(NativeStatus, String), String> {
    if data.len() > RESPONSE_LIMIT {
        return unavailable();
    }
    let v = native_object(data, &["BackendState", "HaveNodeKey"])?;
    let s = decode_native_status(&v)?;
    let outcome = match s.backend_state.as_str() {
        "NeedsLogin" => "needs-login",
        "NeedsMachineAuth" => "approval-required",
        "Starting" | "NoState" | "Stopped" => "pending",
        "Running" => "",
        _ => "unconfirmed",
    };
    Ok((s, outcome.to_string()))
}

/// Mirror of `validateProjectPreferences`.
fn validate_project_preferences(preferences: &[u8]) -> Result<(), String> {
    if preferences.len() > RESPONSE_LIMIT {
        return unavailable();
    }
    let v = native_object(
        preferences,
        &[
            "WantRunning",
            "CorpDNS",
            "RouteAll",
            "RunSSH",
            "ExitNodeID",
            "ExitNodeIP",
            "AdvertiseRoutes",
        ],
    )?;
    let p = decode_native_prefs(&v)?;
    if !p.want_running || !p.corp_dns || p.route_all || p.run_ssh {
        return Err(ERR_CONFLICT.to_string());
    }
    if !p.exit_node_id.is_empty() || !p.exit_node_ip.is_empty() || !p.advertise_routes.is_empty() {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(())
}

/// Mirror of `matchProjectSelf` (binding tags compared as-is, like Go).
fn match_project_self(peer: Option<&SelfPeer>, tags: &[String]) -> bool {
    let p = match peer {
        Some(p) => p,
        None => return false,
    };
    if p.id.is_empty() || !p.online || p.expired {
        return false;
    }
    let mut cloned = p.tags.clone();
    cloned.sort();
    cloned == tags
}

/// Mirror of `matchProjectBinding`.
fn match_project_binding(s: &NativeStatus, binding: &RunBinding) -> bool {
    if !binding.enabled || !s.have_node_key {
        return false;
    }
    match &s.tailnet {
        Some(name) if name == &binding.tailnet => {}
        _ => return false,
    }
    match_project_self(s.peer.as_ref(), &binding.tags)
}

/// Mirror of `ProjectStatus`: `(outcome, addresses, dns_name)`.
pub fn project_status(
    data: &[u8],
    prefs: &[u8],
    binding: &RunBinding,
) -> Result<(String, Vec<String>, String), String> {
    let (s, outcome) = parse_project_status(data)?;
    if !outcome.is_empty() {
        return Ok((outcome, Vec::new(), String::new()));
    }
    validate_project_preferences(prefs)?;
    if !match_project_binding(&s, binding) {
        return Ok(("unconfirmed".to_string(), Vec::new(), String::new()));
    }
    let peer = match &s.peer {
        Some(p) => p,
        None => return Ok(("unconfirmed".to_string(), Vec::new(), String::new())),
    };
    let (addrs, dns_name) = resolve_project_peer(&peer.dns_name, &peer.ips)?;
    Ok(("connected".to_string(), addrs, dns_name))
}

/// Mirror of `ProjectHasNode`.
pub fn project_has_node(data: &[u8]) -> Result<bool, String> {
    if data.len() > RESPONSE_LIMIT {
        return unavailable();
    }
    let v = native_object(data, &["BackendState", "HaveNodeKey"])?;
    let fields = match v.as_object() {
        Some(f) => f,
        None => return unavailable(),
    };
    let mut backend = String::new();
    let mut key = false;
    bind_string_into(fields, "BackendState", &mut backend)?;
    bind_bool_into(fields, "HaveNodeKey", &mut key)?;
    match backend.as_str() {
        "NoState" | "NeedsLogin" | "NeedsMachineAuth" | "Stopped" | "Starting" => {}
        "Running" => {
            if !key {
                return unavailable();
            }
        }
        _ => return unavailable(),
    }
    Ok(key)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn binding() -> RunBinding {
        RunBinding {
            enabled: true,
            admission: true,
            tailnet: "soda.example.test".to_string(),
            tags: vec!["tag:soda-project".to_string()],
        }
    }

    fn status_doc(state: &str) -> String {
        format!(
            concat!(
                r#"{{"Version":"1.102.4","BackendState":"{state}","HaveNodeKey":true,"#,
                r#""CurrentTailnet":{{"Name":"soda.example.test"}},"#,
                r#""Self":{{"ID":"node-project-a","Online":true,"DNSName":"project.soda.ts.net.","#,
                r#""TailscaleIPs":["100.64.0.2"],"Tags":["tag:soda-project"]}},"#,
                r#""AuthURL":"private","Health":["private"]}}"#
            ),
            state = state
        )
    }

    fn prefs_doc() -> String {
        r#"{"WantRunning":true,"CorpDNS":true,"RouteAll":false,"RunSSH":false,"ExitNodeID":"","ExitNodeIP":"","AdvertiseRoutes":null,"Persist":{"PrivateNodeKey":"private"}}"#
            .to_string()
    }

    #[test]
    fn has_node_vectors() {
        // Ported from TestProjectHasNodeUsesCurrentStateAndOptionalNodeKey.
        for (body, node, invalid) in [
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":false}"#,
                false,
                false,
            ),
            (
                r#"{"Version":"other","BackendState":"Running","HaveNodeKey":true}"#,
                true,
                false,
            ),
            (
                r#"{"BackendState":"NeedsMachineAuth","HaveNodeKey":true}"#,
                true,
                false,
            ),
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":true}"#,
                true,
                false,
            ),
            (
                r#"{"BackendState":"Running","HaveNodeKey":false}"#,
                false,
                true,
            ),
            (
                r#"{"BackendState":"Unknown","HaveNodeKey":false}"#,
                false,
                true,
            ),
            (r#"{"BackendState":"NeedsLogin"}"#, false, false),
            (r#"{"BackendState":"Running"}"#, false, true),
            (r#"{"HaveNodeKey":false}"#, false, true),
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":null}"#,
                false,
                true,
            ),
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":"false"}"#,
                false,
                true,
            ),
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":0}"#,
                false,
                true,
            ),
            (
                r#"{"BackendState":"NeedsLogin","haveNodeKey":false}"#,
                false,
                true,
            ),
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":false,"HaveNodeKey":true}"#,
                false,
                true,
            ),
            ("null", false, true),
        ] {
            let r = project_has_node(body.as_bytes());
            match r {
                Ok(v) => assert!(
                    !invalid && v == node,
                    "{body}: got node={v}, want node={node} invalid={invalid}"
                ),
                Err(e) => assert!(
                    invalid && e == ERR_UNAVAILABLE,
                    "{body}: got err={e:?}, want node={node} invalid={invalid}"
                ),
            }
        }
    }

    #[test]
    fn has_node_state_table() {
        for (state, key, ok, node) in [
            ("NoState", false, true, false),
            ("NeedsLogin", false, true, false),
            ("NeedsMachineAuth", true, true, true),
            ("Stopped", true, true, true),
            ("Starting", false, true, false),
            ("Running", true, true, true),
            ("Running", false, false, false),
            ("InUseOtherUser", true, false, false),
            ("", true, false, false),
            ("Bogus", true, false, false),
        ] {
            let body = format!(r#"{{"BackendState":"{state}","HaveNodeKey":{key}}}"#);
            let r = project_has_node(body.as_bytes());
            if ok {
                assert_eq!(r.unwrap(), node, "{body}");
            } else {
                assert_eq!(r.unwrap_err(), ERR_UNAVAILABLE, "{body}");
            }
        }
    }

    #[test]
    fn status_fresh_daemon_omits_node_key() {
        // Ported from TestProjectStatusFreshDaemonOmitsFalseNodeKey.
        let (state, ips, dns) = project_status(
            br#"{"Version":"1.102.4","BackendState":"NeedsLogin"}"#,
            &[],
            &RunBinding::default(),
        )
        .unwrap();
        assert_eq!(state, "needs-login");
        assert!(ips.is_empty());
        assert!(dns.is_empty());
        for value in ["null", r#""false""#, "0"] {
            let body = format!(
                r#"{{"Version":"1.102.4","BackendState":"NeedsLogin","HaveNodeKey":{value}}}"#
            );
            assert_eq!(
                project_status(body.as_bytes(), &[], &RunBinding::default()).unwrap_err(),
                ERR_UNAVAILABLE,
                "{value}"
            );
        }
    }

    #[test]
    fn status_connected_and_unsafe_prefs() {
        // Ported from TestProjectStatusRequiresExactNetworkTagsAndNativePreferences.
        let (state, ips, dns) = project_status(
            status_doc("Running").as_bytes(),
            prefs_doc().as_bytes(),
            &binding(),
        )
        .unwrap();
        assert_eq!(state, "connected");
        assert_eq!(ips, vec!["100.64.0.2".to_string()]);
        assert_eq!(dns, "project.soda.ts.net");
        for prefs in [
            prefs_doc().replace(r#""CorpDNS":true"#, r#""CorpDNS":false"#),
            prefs_doc().replace(r#""RouteAll":false"#, r#""RouteAll":true"#),
            prefs_doc().replace(r#""RunSSH":false"#, r#""RunSSH":true"#),
            prefs_doc().replace(r#""ExitNodeID":"""#, r#""ExitNodeID":"foreign""#),
            prefs_doc().replace(r#""ExitNodeIP":"""#, r#""ExitNodeIP":"1.2.3.4""#),
            prefs_doc().replace(
                r#""AdvertiseRoutes":null"#,
                r#""AdvertiseRoutes":["0.0.0.0/0"]"#,
            ),
            prefs_doc().replace(r#""WantRunning":true"#, r#""WantRunning":false"#),
        ] {
            assert_eq!(
                project_status(
                    status_doc("Running").as_bytes(),
                    prefs.as_bytes(),
                    &binding()
                )
                .unwrap_err(),
                ERR_CONFLICT,
                "{prefs}"
            );
        }
    }

    #[test]
    fn status_binding_mismatch_is_unconfirmed() {
        let mut bad = binding();
        bad.tags = vec!["tag:other".to_string()];
        let (state, ips, _) = project_status(
            status_doc("Running").as_bytes(),
            prefs_doc().as_bytes(),
            &bad,
        )
        .unwrap();
        assert_eq!(state, "unconfirmed");
        assert!(ips.is_empty());
        // Unsorted binding tags never match (Go compares the raw slice).
        let mut unsorted = binding();
        unsorted.tags = vec!["tag:soda-project".to_string(), "tag:aaa".to_string()];
        let body = status_doc("Running").replace(
            r#""Tags":["tag:soda-project"]"#,
            r#""Tags":["tag:aaa","tag:soda-project"]"#,
        );
        let (state, _, _) =
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &unsorted).unwrap();
        assert_eq!(state, "unconfirmed");
        // Disabled binding, wrong tailnet, missing tailnet, offline/expired/empty self.
        for body in [
            status_doc("Running").replace(r#""Online":true"#, r#""Online":false"#),
            status_doc("Running").replace(
                r#""ID":"node-project-a""#,
                r#""ID":"node-project-a","Expired":true"#,
            ),
            status_doc("Running").replace(r#""ID":"node-project-a""#, r#""ID":"""#),
            status_doc("Running").replace("soda.example.test", "other.example.test"),
            status_doc("Running").replace(
                r#""CurrentTailnet":{"Name":"soda.example.test"}"#,
                r#""CurrentTailnet":null"#,
            ),
        ] {
            let (state, _, _) =
                project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap();
            assert_eq!(state, "unconfirmed", "{body}");
        }
        let (state, _, _) = project_status(
            status_doc("Running").as_bytes(),
            prefs_doc().as_bytes(),
            &RunBinding::default(),
        )
        .unwrap();
        assert_eq!(state, "unconfirmed");
    }

    #[test]
    fn status_non_running_outcomes() {
        for (native, want) in [
            ("NeedsMachineAuth", "approval-required"),
            ("NeedsLogin", "needs-login"),
            ("Starting", "pending"),
            ("NoState", "pending"),
            ("Stopped", "pending"),
            ("Bogus", "unconfirmed"),
            ("InUseOtherUser", "unconfirmed"),
        ] {
            let (state, ips, dns) =
                project_status(status_doc(native).as_bytes(), &[], &RunBinding::default()).unwrap();
            assert_eq!(state, want, "{native}");
            assert!(ips.is_empty());
            assert!(dns.is_empty());
        }
        // Release numbers are not a runtime veto.
        let body = status_doc("NeedsLogin").replace("1.102.4", "a different release");
        let (state, _, _) = project_status(body.as_bytes(), &[], &RunBinding::default()).unwrap();
        assert_eq!(state, "needs-login");
    }

    #[test]
    fn status_malformed_inputs() {
        // Running with missing prefs body is unavailable, not unconfirmed.
        assert_eq!(
            project_status(status_doc("Running").as_bytes(), &[], &binding()).unwrap_err(),
            ERR_UNAVAILABLE
        );
        // Prefs missing a required key, or mistyped, is unavailable.
        for prefs in [
            prefs_doc().replace(r#""CorpDNS":true,"#, ""),
            prefs_doc().replace(r#""WantRunning":true"#, r#""WantRunning":"yes""#),
            prefs_doc().replace(r#""AdvertiseRoutes":null"#, r#""AdvertiseRoutes":"x""#),
            prefs_doc().replace(
                r#""AdvertiseRoutes":null"#,
                r#""AdvertiseRoutes":null,"AdvertiseRoutes":[]"#,
            ),
        ] {
            assert_eq!(
                project_status(
                    status_doc("Running").as_bytes(),
                    prefs.as_bytes(),
                    &binding()
                )
                .unwrap_err(),
                ERR_UNAVAILABLE,
                "{prefs}"
            );
        }
        // `[null]` routes decode to `[""]`, which is unsafe -> conflict.
        let prefs = prefs_doc().replace(r#""AdvertiseRoutes":null"#, r#""AdvertiseRoutes":[null]"#);
        assert_eq!(
            project_status(
                status_doc("Running").as_bytes(),
                prefs.as_bytes(),
                &binding()
            )
            .unwrap_err(),
            ERR_CONFLICT
        );
        // Status mistyped optionals fail closed.
        for body in [
            status_doc("Running").replace(r#""Self":{"#, r#""Self":"x","Self2":{"#),
            status_doc("Running").replace(r#""Online":true"#, r#""Online":"yes""#),
            status_doc("Running").replace(
                r#""TailscaleIPs":["100.64.0.2"]"#,
                r#""TailscaleIPs":[true]"#,
            ),
            status_doc("Running").replace(
                r#""CurrentTailnet":{"Name":"soda.example.test"}"#,
                r#""CurrentTailnet":[]"#,
            ),
            status_doc("Running").replace(r#""BackendState":"Running""#, r#""BackendState":123"#),
        ] {
            assert_eq!(
                project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap_err(),
                ERR_UNAVAILABLE,
                "{body}"
            );
        }
        // Nested exact-duplicate keys are rejected by the strict gate.
        let body = status_doc("Running").replace(r#""ID":"#, r#""ID":"x","ID":"#);
        assert_eq!(
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap_err(),
            ERR_UNAVAILABLE
        );
        // Trailing data and non-objects are rejected.
        for raw in ["null", "[]", "42", "", "{},", "{\"BackendState\""] {
            assert_eq!(
                project_status(raw.as_bytes(), &[], &RunBinding::default()).unwrap_err(),
                ERR_UNAVAILABLE,
                "{raw}"
            );
        }
        let big_ok = format!(
            r#"{{"BackendState":"NeedsLogin","HaveNodeKey":false{}}}"#,
            ""
        );
        assert!(project_status(big_ok.as_bytes(), &[], &RunBinding::default()).is_ok());
    }

    #[test]
    fn status_case_fold_binding() {
        // Case-variant keys bind with last-in-document-wins; pointer structs merge.
        let body = status_doc("Running").replace(
            r#""ID":"node-project-a","#,
            r#""id":"lower","ID":"node-project-a","#,
        );
        let (state, _, _) =
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap();
        assert_eq!(state, "connected");
        // Reversed order: the fold-equal key comes last and wins (empty ID).
        let body = status_doc("Running").replace(
            r#""ID":"node-project-a","#,
            r#""ID":"node-project-a","id":"","#,
        );
        let (state, _, _) =
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap();
        assert_eq!(state, "unconfirmed");
        // Pointer structs merge across case-variant keys (Go reuses the struct).
        let body = r#"{"Version":"1.102.4","BackendState":"Running","HaveNodeKey":true,"CurrentTailnet":{"Name":"soda.example.test"},"Self":{"ID":"node-project-a","Online":true,"TailscaleIPs":["100.64.0.2"],"Tags":["tag:soda-project"]},"self":{"DNSName":"project.soda.ts.net."},"AuthURL":"private","Health":["private"]}"#;
        let (state, ips, dns) =
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap();
        assert_eq!(state, "connected");
        assert_eq!(ips, vec!["100.64.0.2".to_string()]);
        assert_eq!(dns, "project.soda.ts.net");
        // A case-aliased HaveNodeKey blocks the omission rule.
        let body = r#"{"BackendState":"NeedsLogin","havenodekey":false}"#;
        assert_eq!(
            project_status(body.as_bytes(), &[], &RunBinding::default()).unwrap_err(),
            ERR_UNAVAILABLE
        );
    }

    #[test]
    fn status_response_limit() {
        let mut big = format!(
            r#"{{"BackendState":"NeedsLogin","Pad":"{}"}}"#,
            "x".repeat(65536)
        );
        assert!(big.len() > RESPONSE_LIMIT);
        assert_eq!(
            project_status(big.as_bytes(), &[], &RunBinding::default()).unwrap_err(),
            ERR_UNAVAILABLE
        );
        assert_eq!(
            project_has_node(big.as_bytes()).unwrap_err(),
            ERR_UNAVAILABLE
        );
        big = format!(
            r#"{{"BackendState":"NeedsLogin","Pad":"{}"}}"#,
            "x".repeat(1000)
        );
        assert!(project_status(big.as_bytes(), &[], &RunBinding::default()).is_ok());
    }

    #[test]
    fn status_peer_address_rules() {
        // Each address list replaces the fixture's single IPv4; all must be
        // canonical global-unicast forms or the whole status is unavailable.
        for (ips, ok) in [
            (r#"["100.64.0.2"]"#, true),
            (r#"["10.0.0.1","192.168.1.1"]"#, true),
            (r#"["fd7a:115c:a1e0::1"]"#, true),
            (r#"["2001:db8::1"]"#, true),
            (r#"["::ffff:1.2.3.4"]"#, true),
            (r#"["1:2:3:4:5:6:7:8"]"#, true),
            (r#"["fd7a::1%ETH0"]"#, true),
            (r#"["127.0.0.1"]"#, false),
            (r#"["0.0.0.0"]"#, false),
            (r#"["169.254.1.1"]"#, false),
            (r#"["224.0.0.1"]"#, false),
            (r#"["255.255.255.255"]"#, false),
            (r#"["::1"]"#, false),
            (r#"["::"]"#, false),
            (r#"["fe80::1"]"#, false),
            (r#"["ff02::1"]"#, false),
            (r#"["::ffff:127.0.0.1"]"#, false),
            (r#"["01.2.3.4"]"#, false),
            (r#"["FD7A::1"]"#, false),
            (r#"["fd7a:115c:a1e0:0:0:0:0:1"]"#, false),
            (r#"["1.2.3.4%eth0"]"#, false),
            (r#"["100.64.0.2%"]"#, false),
            (r#"["1.2.3.4 "]"#, false),
            (r#"["abc"]"#, false),
            (r#"[]"#, false),
            (r#"null"#, false),
        ] {
            let body = status_doc("Running").replace(r#"["100.64.0.2"]"#, ips);
            let r = project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding());
            if ok {
                assert_eq!(r.unwrap().0, "connected", "{ips}");
            } else {
                assert_eq!(r.unwrap_err(), ERR_UNAVAILABLE, "{ips}");
            }
        }
        // More than 16 addresses is unavailable.
        let many = "[".to_string() + &vec!["\"100.64.0.2\""; 17].join(",") + "]";
        let body = status_doc("Running").replace(r#"["100.64.0.2"]"#, &many);
        assert_eq!(
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap_err(),
            ERR_UNAVAILABLE
        );
    }

    #[test]
    fn status_dns_name_rules() {
        let long_label = "a".repeat(64);
        let long_name = format!("{}.example.ts.net", "a".repeat(240));
        for (dns, ok) in [
            ("project.soda.ts.net.", true),
            ("Atlas.Example.ts.net.", true),
            ("  atlas.example.ts.net  ", true),
            ("a.b", true),
            ("a.b..", false),
            ("atlas.local", false),
            ("atlas.LOCAL.", false),
            ("nodot", false),
            ("", true),
            ("-a.b", false),
            ("a-.b", false),
            ("a.B_c", false),
            ("a..b", false),
            (".a.b", false),
            ("exa mple.ts.net", false),
            (&long_label, false),
            (&long_name, false),
        ] {
            let body = status_doc("Running").replace("project.soda.ts.net.", dns);
            let r = project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding());
            if ok {
                let (state, _, name) = r.unwrap();
                assert_eq!(state, "connected", "{dns:?}");
                assert_eq!(name, dns.trim().trim_end_matches('.').to_lowercase());
            } else {
                assert_eq!(r.unwrap_err(), ERR_UNAVAILABLE, "{dns:?}");
            }
        }
        assert_eq!(
            canonical_magic_dns_name("Atlas.Example.ts.net.").unwrap(),
            "atlas.example.ts.net"
        );
    }

    #[test]
    fn dns_trims_nel_and_fold_covers_simple_fold_orbits() {
        // Go's strings.TrimSpace also trims U+0085 (NEL).
        assert_eq!(
            canonical_magic_dns_name("atlas.example.ts.net").unwrap(),
            "atlas.example.ts.net"
        );
        // Go's non-ASCII mates of ASCII letters: exactly ſ (S/s) and
        // Kelvin K (K/k). İ (U+0130) does NOT fold with i.
        assert!(fold_eq("ſ", "s"));
        assert!(fold_eq("S", "ſ"));
        assert!(fold_eq("Backendſtate", "BackendState"));
        assert!(fold_eq("\u{212a}", "k"));
        assert!(fold_eq("K", "\u{212a}"));
        assert!(!fold_eq("İ", "i"));
        assert!(!fold_eq("BackendState", "BackendStates"));
        assert!(!fold_eq("ß", "ss"));
    }

    #[test]
    fn id_matchers() {
        assert!(valid_project_id(&("p".to_string() + &"a".repeat(24))));
        for bad in [
            "".to_string(),
            "p".to_string(),
            "P".to_string() + &"a".repeat(24),
            "p".to_string() + &"a".repeat(23),
            "p".to_string() + &"a".repeat(25),
            "p".to_string() + &"b".repeat(23) + "q",
            "p".to_string() + &"B".repeat(24),
        ] {
            assert!(!valid_project_id(&bad), "{bad}");
        }
        assert!(valid_container_id(&"c".repeat(64)));
        for bad in [
            "",
            &"c".repeat(63),
            &("c".repeat(63) + "C"),
            &"c".repeat(65),
        ] {
            assert!(!valid_container_id(bad), "{bad}");
        }
        assert!(valid_image_id(&"d".repeat(64)));
        assert!(valid_image_id(&("sha256:".to_string() + &"d".repeat(64))));
        for bad in [
            "",
            "sha256:",
            &("sha256:".to_string() + &"d".repeat(63)),
            &("SHA256:".to_string() + &"d".repeat(64)),
            &("e".repeat(65)),
        ] {
            assert!(!valid_image_id(bad), "{bad}");
        }
        assert_eq!(ERR_INVALID, "invalid Tailnet request");
        assert_eq!(ERR_CONFLICT, "tailnet revision or identity changed");
        assert_eq!(ERR_UNSUPPORTED, "tailnet runtime is not supported");
        assert_eq!(ERR_UNCONFIRMED, "tailnet outcome is unconfirmed");
        assert_eq!(ERR_UNAVAILABLE, "tailscale status is unavailable");
    }

    #[test]
    fn rfc3339_vectors() {
        for (s, want) in [
            ("0001-01-01T00:00:00Z", Some(true)),
            ("0001-01-01T00:00:00.000000000Z", Some(true)),
            ("0001-01-01T00:00:00+00:00", Some(true)),
            ("0001-01-01T01:00:00+01:00", Some(true)),
            ("0000-12-31T23:00:00-01:00", Some(true)),
            ("0001-01-01T00:00:00.000000001Z", Some(false)),
            ("0001-01-01T00:30:00+01:00", Some(false)),
            ("2024-02-29T00:00:00Z", Some(false)),
            ("2000-02-29T12:30:45.123Z", Some(false)),
            ("1900-02-28T00:00:00Z", Some(false)),
            ("0000-01-01T00:00:00Z", Some(false)),
            ("9999-12-31T23:59:59Z", Some(false)),
            ("2024-01-01T00:00:00.5Z", Some(false)),
            // Go keeps nanosecond precision and ignores further digits.
            ("2024-01-01T00:00:00.1234567891Z", Some(false)),
            ("2024-01-01T00:00:00.0000000001Z", Some(false)),
            (
                "2024-01-01T00:00:00.123456789012345678901234567890Z",
                Some(false),
            ),
            (
                "0001-01-01T00:00:00.000000000000000000000000000000Z",
                Some(true),
            ),
            ("2024-01-01T00:00:00+24:00", Some(false)),
            ("2024-01-01T00:00:00+24:60", Some(false)),
            ("2024-01-01T00:00:00-24:60", Some(false)),
            ("2024-01-01T00:00:00+00:60", Some(false)),
            ("2024-01-01T00:00:00-00:00", Some(false)),
            ("2023-02-29T00:00:00Z", None),
            ("1900-02-29T00:00:00Z", None),
            ("2024-13-01T00:00:00Z", None),
            ("2024-00-10T00:00:00Z", None),
            ("2024-01-00T00:00:00Z", None),
            ("2024-01-32T00:00:00Z", None),
            ("2024-04-31T00:00:00Z", None),
            ("2024-01-01T24:00:00Z", None),
            ("2024-01-01T00:00:60Z", None),
            ("2024-01-01T00:00:00", None),
            ("2024-01-01T00:00:00.Z", None),
            ("2024-01-01T00:00:00.", None),
            ("2024-01-01T00:00:00+07", None),
            ("2024-01-01T00:00:00+0700", None),
            ("2024-01-01T00:00:00+07:00:00", None),
            ("2024-01-01T00:00:00+25:00", None),
            ("2024-01-01T00:00:00+24:61", None),
            ("2024-01-01T00:00:00+00:61", None),
            ("2024-01-01T00:00:00+99:99", None),
            ("2024-01-01t00:00:00z", None),
            ("2024-01-01T00:00:00z", None),
            ("2024-1-1T00:00:00Z", None),
            ("2024-01-01T1:00:00Z", Some(false)),
            ("2024-06-15T0:30:45.123456789+05:30", Some(false)),
            ("2024-01-01T123:00:00Z", None),
            ("2024-01-01T01:2:03Z", None),
            ("2024-01-01T01:02:3Z", None),
            ("24-01-01T00:00:00Z", None),
            ("20240-01-01T00:00:00Z", None),
            ("2024-01-01", None),
            ("2024-01-01T00:00", None),
            ("2024-01-01T00:00:00Z ", None),
            (" 2024-01-01T00:00:00Z", None),
            ("", None),
        ] {
            assert_eq!(parse_rfc3339_nano(s), want, "{s}");
        }
        // Go accepts 10+ fraction digits and keeps nanosecond precision.
        assert_eq!(
            parse_rfc3339_nano("2024-01-01T00:00:00.1234567890Z"),
            Some(false)
        );
    }

    #[test]
    fn escape_vectors() {
        for (raw, want) in [
            ("", r#""""#),
            ("abc", r#""abc""#),
            ("a\"b\\c", r#""a\"b\\c""#),
            ("a\nb\rc\td", r#""a\nb\rc\td""#),
            ("\u{8}\u{c}", r#""\b\f""#),
            ("\u{1}\u{1f}", r#""\u0001\u001f""#),
            ("<>&", r#""\u003c\u003e\u0026""#),
            ("é☃", r#""é☃""#),
            ("\u{7f}", "\"\u{7f}\""),
        ] {
            assert_eq!(go_escape(raw), want, "{raw:?}");
        }
        assert_eq!(go_escape("\u{2028}"), r#""\u2028""#);
        assert_eq!(go_escape("\u{2029}"), r#""\u2029""#);
    }
}
