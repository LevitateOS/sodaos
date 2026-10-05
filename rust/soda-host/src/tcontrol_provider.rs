//! Tailscale provider HTTPS via the host curl binary.
//!
//! PR26 port of the `control.go` credential check (`checkCredential`), the
//! `enrollment.go` OAuth token fetch (`tokenFromEnrollment`) and auth-key
//! creation (`createProjectAuthKey`), and the key-response validators.
//!
//! TLS is delegated to the host `curl` invoked through the crate `Executor`
//! (no hand-rolled crypto): fixed argv recipes, secrets on curl's stdin
//! config (never argv), no redirects, stdout capped like Go's bounded
//! provider bodies. The status code rides back via `--write-out` so error
//! responses stay classifiable without curl's diagnostics.

use std::time::{Instant, SystemTime};

use crate::json::Value;
use crate::tcontrol_wire as wire;

pub const TOKEN_URL: &str = "https://api.tailscale.com/api/v2/oauth/token";
pub const API_HOST: &str = "api.tailscale.com";
pub const PROJECT_KEY_LIFETIME_SECS: i64 = 300;
pub const KEY_DESCRIPTION: &str = "Soda ephemeral project run";
pub const DEFAULT_CURL: &str = "/usr/bin/curl";

/// Provider call description for stubs and tests. `Debug` redacts secrets.
pub enum ProviderRequest {
    Token {
        client_id: String,
        client_secret: String,
        tags: Vec<String>,
    },
    KeyCreate {
        tailnet: String,
        tags: Vec<String>,
        preauthorized: bool,
        token: String,
    },
}

impl std::fmt::Debug for ProviderRequest {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ProviderRequest::Token {
                client_id, tags, ..
            } => f
                .debug_struct("Token")
                .field("client_id", client_id)
                .field("client_secret", &"[redacted]")
                .field("tags", tags)
                .finish(),
            ProviderRequest::KeyCreate {
                tailnet,
                tags,
                preauthorized,
                ..
            } => f
                .debug_struct("KeyCreate")
                .field("tailnet", tailnet)
                .field("tags", tags)
                .field("preauthorized", preauthorized)
                .field("token", &"[redacted]")
                .finish(),
        }
    }
}

/// Provider round trip returning the raw `(status, body)`; validators below
/// apply Go's rules. The production transport shells out to curl; tests
/// inject stubs.
pub type ProviderTransport =
    dyn Fn(ProviderRequest, Instant) -> Result<(u16, Vec<u8>), String> + Send + Sync;

fn provider_failed() -> String {
    // Neutral error: no secret, no diagnostics, no adapter keyword (502).
    // Callers map it to unavailable (check path) or unconfirmed (key path).
    "tailscale provider call failed".to_string()
}

// ---------- curl transport ----------

fn config_escape(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    for c in s.chars() {
        match c {
            '\\' => out.push_str("\\\\"),
            '"' => out.push_str("\\\""),
            _ => out.push(c),
        }
    }
    out
}

/// Mirror of Go's `url.QueryEscape` for the OAuth form body: alphanumerics
/// plus `-_.~` verbatim, space to `+`, the rest `%XX` (uppercase hex).
fn query_escape(s: &str) -> String {
    let mut out = String::new();
    for b in s.bytes() {
        if b.is_ascii_alphanumeric() || matches!(b, b'-' | b'_' | b'.' | b'~') {
            out.push(b as char);
        } else if b == b' ' {
            out.push('+');
        } else {
            out.push_str(&format!("%{b:02X}"));
        }
    }
    out
}

/// OAuth token form body, mirroring `url.Values.Encode` over
/// `grant_type`/`scope`/`tags` (Go map encoding sorts the keys).
pub fn token_form_body(tags: &[String]) -> String {
    format!(
        "grant_type=client_credentials&scope=auth_keys&tags={}",
        query_escape(&tags.join(" "))
    )
}

/// Auth-key creation body, matching the SDK's `CreateKeyRequest` encoding
/// (field order included).
pub fn key_create_body(tags: &[String], preauthorized: bool) -> String {
    let q = crate::json::quote;
    let mut list = String::from("[");
    for (i, t) in tags.iter().enumerate() {
        if i > 0 {
            list.push(',');
        }
        list.push_str(&q(t));
    }
    list.push(']');
    format!(
        "{{\"capabilities\":{{\"devices\":{{\"create\":{{\"reusable\":false,\"ephemeral\":true,\"tags\":{},\"preauthorized\":{}}}}}}},\"expirySeconds\":{},\"description\":{}}}",
        list, preauthorized, PROJECT_KEY_LIFETIME_SECS, q(KEY_DESCRIPTION),
    )
}

fn token_config(client_id: &str, client_secret: &str, tags: &[String]) -> String {
    format!(
        "url = \"{TOKEN_URL}\"\nrequest = \"POST\"\nheader = \"Content-Type: application/x-www-form-urlencoded\"\nuser = \"{}:{}\"\ndata-binary = \"{}\"\n",
        config_escape(client_id),
        config_escape(client_secret),
        config_escape(&token_form_body(tags)),
    )
}

fn key_config(tailnet: &str, tags: &[String], preauthorized: bool, token: &str) -> String {
    format!(
        "url = \"https://{API_HOST}/api/v2/tailnet/{tailnet}/keys\"\nrequest = \"POST\"\nheader = \"Content-Type: application/json\"\nheader = \"Authorization: Bearer {}\"\ndata-binary = \"{}\"\n",
        config_escape(token),
        config_escape(&key_create_body(tags, preauthorized)),
    )
}

/// Run curl with a stdin config; split the `--write-out` status suffix.
/// Deadline enforcement is the executor's kill, like Go's context.
pub fn run_curl(
    exec: &dyn crate::project::Executor,
    curl: &str,
    config: &str,
    deadline: Instant,
) -> Result<(u16, Vec<u8>), String> {
    let out = exec
        .run(
            config.as_bytes(),
            curl,
            &[
                "--silent",
                "--config",
                "-",
                "--write-out",
                "\\n%{http_code}",
            ],
            deadline,
        )
        .map_err(|_| provider_failed())?;
    if out.len() > crate::tcontrol_native::RESPONSE_LIMIT + 4 {
        return Err(provider_failed());
    }
    let split = out
        .iter()
        .rposition(|b| *b == b'\n')
        .ok_or_else(provider_failed)?;
    let (body, tail) = out.split_at(split);
    let code = tail.get(1..).ok_or_else(provider_failed)?;
    if code.len() != 3 || !code.iter().all(|b| b.is_ascii_digit()) {
        return Err(provider_failed());
    }
    let status: u16 = std::str::from_utf8(code)
        .map_err(|_| provider_failed())?
        .parse()
        .map_err(|_| provider_failed())?;
    Ok((status, body.to_vec()))
}

/// Fetch an OAuth token, returning the raw `(status, body)`.
pub fn fetch_token(
    exec: &dyn crate::project::Executor,
    curl: &str,
    client_id: &str,
    client_secret: &str,
    tags: &[String],
    deadline: Instant,
) -> Result<(u16, Vec<u8>), String> {
    let mut config = token_config(client_id, client_secret, tags);
    let out = run_curl(exec, curl, &config, deadline);
    wire::zero_string(&mut config);
    out
}

/// Create an ephemeral auth key, returning the raw `(status, body)`.
pub fn create_key(
    exec: &dyn crate::project::Executor,
    curl: &str,
    tailnet: &str,
    tags: &[String],
    preauthorized: bool,
    token: &str,
    deadline: Instant,
) -> Result<(u16, Vec<u8>), String> {
    let mut config = key_config(tailnet, tags, preauthorized, token);
    let out = run_curl(exec, curl, &config, deadline);
    wire::zero_string(&mut config);
    out
}

// ---------- Token responses ----------

fn last_str(fields: &[(String, Value)], name: &str) -> Option<String> {
    fields
        .iter()
        .rev()
        .find(|(k, _)| k == name)
        .and_then(|(_, v)| match v {
            Value::Null => None,
            Value::Str(s) => Some(s.clone()),
            _ => Some(String::new()),
        })
}

/// Mirror of `expirationTime.UnmarshalJSON`: a JSON number or a numeric
/// string, integer only, wrapping to `i32` exactly like Go's conversion.
fn expires_in(fields: &[(String, Value)]) -> Option<i64> {
    let v = fields
        .iter()
        .rev()
        .find(|(k, _)| k == "expires_in")?
        .1
        .clone();
    let lit = match v {
        Value::Null => return Some(0),
        Value::Number(lit) => lit,
        Value::Str(s) => s,
        _ => return None,
    };
    let neg = lit.strip_prefix('-').unwrap_or(&lit);
    if neg.is_empty() || !neg.bytes().all(|b| b.is_ascii_digit()) {
        // Go's `json.Number.Int64` also rejects `+`, fractions, exponents.
        return None;
    }
    let n: i64 = lit.parse().ok()?;
    Some((n as i32) as i64)
}

/// Validate an OAuth token response, returning the bearer token. Mirrors
/// `checkCredential` (with `strict_bearer = false`) and
/// `tokenFromEnrollment` (`strict_bearer = true`, which additionally rejects
/// control characters). Failures carry the caller's typed error.
pub fn validate_token(
    status: u16,
    body: &[u8],
    strict_bearer: bool,
    failure: &dyn Fn() -> String,
) -> Result<String, String> {
    if status != 200 {
        return Err(failure());
    }
    // Expiry is anchored at decode time, like Go's `tokenJSON.expiry()`.
    let decoded_at = system_nanos(SystemTime::now());
    // Go decodes with `encoding/json`: unknown fields ignored, last
    // duplicate wins, mistyped values fail.
    let v = crate::json::decode_tolerant(body).map_err(|_| failure())?;
    let fields = match v.as_object() {
        Some(f) => f,
        None => return Err(failure()),
    };
    let token = last_str(fields, "access_token").unwrap_or_default();
    if token.is_empty() {
        return Err(failure());
    }
    if strict_bearer && token.contains(['\r', '\n', '\0']) {
        return Err(failure());
    }
    let token_type = last_str(fields, "token_type").unwrap_or_default();
    if !token_type.eq_ignore_ascii_case("bearer") {
        return Err(failure());
    }
    let skew = expires_in(fields).ok_or_else(failure)?;
    if decoded_at + skew as i128 * 1_000_000_000 <= system_nanos(SystemTime::now()) {
        return Err(failure());
    }
    Ok(token)
}

// ---------- RFC 3339 instants ----------

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

/// Strict `time.Parse(time.RFC3339Nano)` returning nanoseconds since the Unix
/// epoch, or `None` for invalid input. Same grammar as Go: 1-2 digit hour,
/// fixed minute/second, optional fraction (kept to nanoseconds), `Z` or
/// `±HH:MM` with Go's per-field offset bounds (HH ≤ 24, MM ≤ 60).
pub fn parse_rfc3339_nanos(s: &str) -> Option<i128> {
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
    let mut nanos: i128 = 0;
    if b.get(pos) == Some(&b'.') {
        pos += 1;
        let start = pos;
        while pos < b.len() && b[pos].is_ascii_digit() {
            pos += 1;
        }
        if pos - start < 1 {
            return None;
        }
        let mut scale = 100_000_000i128;
        for &c in &b[start..start + (pos - start).min(9)] {
            nanos += (c - b'0') as i128 * scale;
            scale /= 10;
        }
    }
    let rest = b.get(pos..)?;
    let offset_secs: i128 = if rest.len() == 1 && rest[0] == b'Z' {
        0
    } else if rest.len() == 6
        && (rest[0] == b'+' || rest[0] == b'-')
        && rest[3] == b':'
        && rest[1].is_ascii_digit()
        && rest[2].is_ascii_digit()
        && rest[4].is_ascii_digit()
        && rest[5].is_ascii_digit()
    {
        let hh = ((rest[1] - b'0') as i128) * 10 + (rest[2] - b'0') as i128;
        let mm = ((rest[4] - b'0') as i128) * 10 + (rest[5] - b'0') as i128;
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
    let days = days_since_year_one(year, month, day) as i128 - 719162;
    Some(
        (days * 86_400 + hour as i128 * 3600 + minute as i128 * 60 + second as i128 - offset_secs)
            * 1_000_000_000
            + nanos,
    )
}

/// Nanoseconds since the Unix epoch for a `SystemTime` (negative before it).
pub fn system_nanos(t: SystemTime) -> i128 {
    match t.duration_since(SystemTime::UNIX_EPOCH) {
        Ok(d) => d.as_secs() as i128 * 1_000_000_000 + d.subsec_nanos() as i128,
        Err(e) => {
            let d = e.duration();
            -(d.as_secs() as i128 * 1_000_000_000 + d.subsec_nanos() as i128)
        }
    }
}

// ---------- Key responses ----------

fn exact_field<'a>(fields: &'a [(String, Value)], name: &str) -> Option<&'a Value> {
    fields.iter().find(|(k, _)| k == name).map(|(_, v)| v)
}

// `strings.EqualFold` for ASCII field names (see the native module): ASCII
// case plus ſ (U+017F, with S/s) and K (U+212A, with K/k).
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

/// Last fold-matching value, for SDK-bound fields without a required-name
/// check (`invalid`, `revoked`).
fn fold_field<'a>(fields: &'a [(String, Value)], name: &str) -> Option<&'a Value> {
    fields
        .iter()
        .rev()
        .find(|(k, _)| fold_eq(k, name))
        .map(|(_, v)| v)
}

fn require_str(fields: &[(String, Value)], name: &str) -> Result<String, String> {
    match exact_field(fields, name) {
        Some(Value::Str(s)) => Ok(s.clone()),
        _ => Err(wire::err_unconfirmed()),
    }
}

fn require_object<'a>(
    fields: &'a [(String, Value)],
    name: &str,
) -> Result<&'a [(String, Value)], String> {
    match exact_field(fields, name) {
        Some(Value::Object(inner)) => Ok(inner),
        _ => Err(wire::err_unconfirmed()),
    }
}

fn require_bool(fields: &[(String, Value)], name: &str) -> Result<bool, String> {
    match exact_field(fields, name) {
        Some(Value::Bool(b)) => Ok(*b),
        _ => Err(wire::err_unconfirmed()),
    }
}

fn optional_bool(fields: &[(String, Value)], name: &str) -> Result<bool, String> {
    // Fold-matched, like the SDK binding (no required-name check governs).
    match fold_field(fields, name) {
        None | Some(Value::Null) => Ok(false),
        Some(Value::Bool(b)) => Ok(*b),
        Some(_) => Err(wire::err_unconfirmed()),
    }
}

/// Validate a key-creation response, returning the single-use auth key.
/// Mirrors `validKeyCreateCapabilities` plus the SDK binding and
/// `validCreatedAuthKey`. `before` is the instant just before the create
/// call; `now` the validation instant (both injectable for tests).
pub fn validate_key(
    status: u16,
    body: &[u8],
    tags: &[String],
    preauthorized: bool,
    before: SystemTime,
    now: SystemTime,
) -> Result<String, String> {
    if status != 200 || body.len() > crate::tcontrol_native::RESPONSE_LIMIT {
        return Err(wire::err_unconfirmed());
    }
    // Strict validation at every level, then document-order binding, like
    // `nativeObject` (strict success implies tolerant success).
    if crate::json::decode_strict(body).is_err() {
        return Err(wire::err_unconfirmed());
    }
    let v = crate::json::decode_tolerant(body).map_err(|_| wire::err_unconfirmed())?;
    let fields = match v.as_object() {
        Some(f) => f,
        None => return Err(wire::err_unconfirmed()),
    };
    let id = require_str(fields, "id")?;
    let key = require_str(fields, "key")?;
    let created = require_str(fields, "created")?;
    let expires = require_str(fields, "expires")?;
    let caps = require_object(fields, "capabilities")?;
    let devices = require_object(caps, "devices")?;
    let create = require_object(devices, "create")?;
    let reusable = require_bool(create, "reusable")?;
    let ephemeral = require_bool(create, "ephemeral")?;
    let got_preauth = require_bool(create, "preauthorized")?;
    let got_tags = match exact_field(create, "tags") {
        // The SDK binds `[]string`: null elements become empty strings
        // (which then fail the equality check), mistyped fails.
        Some(Value::Array(items)) => {
            let mut out = Vec::with_capacity(items.len());
            for item in items {
                match item {
                    Value::Null => out.push(String::new()),
                    Value::Str(s) => out.push(s.clone()),
                    _ => return Err(wire::err_unconfirmed()),
                }
            }
            out
        }
        _ => return Err(wire::err_unconfirmed()),
    };
    if id.is_empty() || id.len() > 128 || !wire::valid_auth_key(&key) {
        return Err(wire::err_unconfirmed());
    }
    if optional_bool(fields, "invalid")? {
        return Err(wire::err_unconfirmed());
    }
    // `revoked` must be the zero time (missing/null decodes to zero, like Go).
    const ZERO_NANOS: i128 = -(719162i128 * 86_400 * 1_000_000_000);
    match fold_field(fields, "revoked") {
        None | Some(Value::Null) => {}
        Some(Value::Str(s)) => {
            if parse_rfc3339_nanos(s) != Some(ZERO_NANOS) {
                return Err(wire::err_unconfirmed());
            }
        }
        Some(_) => return Err(wire::err_unconfirmed()),
    }
    if reusable || !ephemeral || got_preauth != preauthorized || got_tags != tags {
        return Err(wire::err_unconfirmed());
    }
    let created_at = parse_rfc3339_nanos(&created).ok_or_else(wire::err_unconfirmed)?;
    let expires_at = parse_rfc3339_nanos(&expires).ok_or_else(wire::err_unconfirmed)?;
    let before_nanos = system_nanos(before);
    let now_nanos = system_nanos(now);
    if created_at < before_nanos - 60_000_000_000
        || created_at > now_nanos + 60_000_000_000
        || expires_at <= now_nanos
        || expires_at > before_nanos + (PROJECT_KEY_LIFETIME_SECS as i128 + 60) * 1_000_000_000
    {
        return Err(wire::err_unconfirmed());
    }
    Ok(key)
}
