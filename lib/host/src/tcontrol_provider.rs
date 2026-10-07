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

use crate::json;
use crate::tcontrol_wire as wire;
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};

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

#[derive(Debug, Clone)]
enum ExpirationWire {
    Null,
    Number(String),
    Text(String),
}

impl<'de> Deserialize<'de> for ExpirationWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let raw = Box::<serde_json::value::RawValue>::deserialize(deserializer)?;
        if raw.get() == "null" {
            return Ok(Self::Null);
        }
        if raw.get().starts_with('"') {
            return serde_json::from_str::<String>(raw.get())
                .map(Self::Text)
                .map_err(serde::de::Error::custom);
        }
        Ok(Self::Number(raw.get().to_string()))
    }
}

impl ExpirationWire {
    fn seconds(&self) -> Option<i64> {
        let lit = match self {
            Self::Null => return Some(0),
            Self::Number(v) | Self::Text(v) => v,
        };
        let digits = lit.strip_prefix('-').unwrap_or(lit);
        if digits.is_empty() || !digits.bytes().all(|byte| byte.is_ascii_digit()) {
            return None;
        }
        let value = lit.parse::<i64>().ok()?;
        Some((value as i32) as i64)
    }
}

#[derive(Default)]
struct TokenWire {
    access_token: Option<String>,
    token_type: Option<String>,
    expires_in: Option<ExpirationWire>,
}

impl<'de> Deserialize<'de> for TokenWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = TokenWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("OAuth token response")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = TokenWire::default();
                while let Some(key) = map.next_key::<String>()? {
                    match key.as_str() {
                        "access_token" => out.access_token = map.next_value::<Option<String>>()?,
                        "token_type" => out.token_type = map.next_value::<Option<String>>()?,
                        "expires_in" => {
                            out.expires_in = map.next_value::<Option<ExpirationWire>>()?
                        }
                        _ => {
                            map.next_value::<de::IgnoredAny>()?;
                        }
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(V)
    }
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
    let value: TokenWire = json::decode_tolerant_as(body).map_err(|_| failure())?;
    let token = value.access_token.unwrap_or_default();
    if token.is_empty() {
        return Err(failure());
    }
    if strict_bearer && token.contains(['\r', '\n', '\0']) {
        return Err(failure());
    }
    let token_type = value.token_type.unwrap_or_default();
    if !token_type.eq_ignore_ascii_case("bearer") {
        return Err(failure());
    }
    let skew = value
        .expires_in
        .as_ref()
        .and_then(ExpirationWire::seconds)
        .ok_or_else(failure)?;
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

#[derive(Deserialize, Default)]
struct KeyBindingWire {
    #[serde(rename = "reusable")]
    reusable: Option<bool>,
    ephemeral: Option<bool>,
    preauthorized: Option<bool>,
    tags: Option<Vec<Option<String>>>,
}
#[derive(Deserialize, Default)]
struct KeyCreateDeviceWire {
    create: Option<KeyBindingWire>,
}
#[derive(Deserialize, Default)]
struct KeyCreateCapabilitiesWire {
    devices: Option<KeyCreateDeviceWire>,
}
#[derive(Default)]
struct KeyResponseWire {
    id: Option<String>,
    key: Option<String>,
    created: Option<String>,
    expires: Option<String>,
    capabilities: Option<KeyCreateCapabilitiesWire>,
    invalid: Option<Option<bool>>,
    revoked: Option<Option<String>>,
}

fn fold_ascii_go(a: &str, b: &str) -> bool {
    if a.eq_ignore_ascii_case(b) {
        return true;
    }
    let fold = |c: char| match c {
        'ſ' => 's',
        '\u{212a}' => 'k',
        x if x.is_ascii_alphabetic() => x.to_ascii_lowercase(),
        x => x,
    };
    let mut x = a.chars();
    let mut y = b.chars();
    loop {
        match (x.next(), y.next()) {
            (None, None) => return true,
            (Some(a), Some(b)) if fold(a) == fold(b) => {}
            (Some(_), Some(_)) | (None, Some(_)) | (Some(_), None) => return false,
        }
    }
}
impl<'de> Deserialize<'de> for KeyResponseWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = KeyResponseWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("auth key response")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = KeyResponseWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    match k.as_str() {
                        "id" => o.id = map.next_value()?,
                        "key" => o.key = map.next_value()?,
                        "created" => o.created = map.next_value()?,
                        "expires" => o.expires = map.next_value()?,
                        "capabilities" => o.capabilities = map.next_value()?,
                        _ if fold_ascii_go(&k, "invalid") => {
                            o.invalid = Some(map.next_value::<Option<bool>>()?)
                        }
                        _ if fold_ascii_go(&k, "revoked") => {
                            o.revoked = Some(map.next_value::<Option<String>>()?)
                        }
                        _ => {
                            map.next_value::<de::IgnoredAny>()?;
                        }
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
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
    let value: KeyResponseWire =
        json::decode_strict_as(body).map_err(|_| wire::err_unconfirmed())?;
    let id = value.id.ok_or_else(wire::err_unconfirmed)?;
    let key = value.key.ok_or_else(wire::err_unconfirmed)?;
    let created = value.created.ok_or_else(wire::err_unconfirmed)?;
    let expires = value.expires.ok_or_else(wire::err_unconfirmed)?;
    let create = value
        .capabilities
        .and_then(|v| v.devices)
        .and_then(|v| v.create)
        .ok_or_else(wire::err_unconfirmed)?;
    let reusable = create.reusable.ok_or_else(wire::err_unconfirmed)?;
    let ephemeral = create.ephemeral.ok_or_else(wire::err_unconfirmed)?;
    let got_preauth = create.preauthorized.ok_or_else(wire::err_unconfirmed)?;
    let got_tags = create
        .tags
        .ok_or_else(wire::err_unconfirmed)?
        .into_iter()
        .map(Option::unwrap_or_default)
        .collect::<Vec<_>>();
    if id.is_empty() || id.len() > 128 || !wire::valid_auth_key(&key) {
        return Err(wire::err_unconfirmed());
    }
    if value.invalid.flatten().unwrap_or_default() {
        return Err(wire::err_unconfirmed());
    }
    // `revoked` must be the zero time (missing/null decodes to zero, like Go).
    const ZERO_NANOS: i128 = -(719162i128 * 86_400 * 1_000_000_000);
    if let Some(s) = value.revoked.flatten() {
        if parse_rfc3339_nanos(&s) != Some(ZERO_NANOS) {
            return Err(wire::err_unconfirmed());
        }
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
