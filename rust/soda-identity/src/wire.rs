// Identity wire records, byte-identical to internal/identity: field names,
// order, string-encoded integers and RFC3339Nano timestamps. Connection and
// Enrollment are reused from the providers crate so the shapes cannot drift.
use serde::{Deserialize, Deserializer, Serialize, Serializer};

pub use identity_providers::types::{credential_valid, Connection, Enrollment};

pub const CODEX: &str = "codex";
pub const CLAUDE: &str = "claude";
pub const READY: &str = "ready";
pub const REAUTH: &str = "reauth";
pub const REVOKED: &str = "revoked";
pub const FACTORY: &str = "factory";
pub const TERMINAL: &str = "terminal";

pub const EXECUTION_PENDING: &str = "pending";
pub const EXECUTION_LIVE: &str = "live";
pub const EXECUTION_TERMINAL: &str = "terminal";

pub fn provider_valid(id: &str) -> bool {
    id == CODEX || id == CLAUDE
}

/// Unix time with nanoseconds, serialized exactly like Go time.Time
/// (RFC3339Nano, UTC `Z`, trailing fractional zeros trimmed).
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub struct UnixTime {
    pub sec: i64,
    pub nanos: u32,
}

impl UnixTime {
    pub fn now() -> UnixTime {
        let duration = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default();
        UnixTime { sec: duration.as_secs() as i64, nanos: duration.subsec_nanos() }
    }

    pub fn add_hours(self, hours: i64) -> UnixTime {
        UnixTime { sec: self.sec + hours * 3600, nanos: self.nanos }
    }

    pub fn as_system_time(self) -> std::time::SystemTime {
        if self.sec >= 0 {
            std::time::UNIX_EPOCH + std::time::Duration::new(self.sec as u64, self.nanos)
        } else {
            std::time::UNIX_EPOCH - std::time::Duration::new((-self.sec) as u64, 0)
                + std::time::Duration::from_nanos(self.nanos as u64)
        }
    }
}

impl Serialize for UnixTime {
    fn serialize<S: Serializer>(&self, ser: S) -> Result<S::Ok, S::Error> {
        ser.serialize_str(&format_rfc3339_nano(self.sec, self.nanos))
    }
}

impl<'de> Deserialize<'de> for UnixTime {
    fn deserialize<D: Deserializer<'de>>(de: D) -> Result<UnixTime, D::Error> {
        let text = String::deserialize(de)?;
        let (sec, nanos) = parse_rfc3339_nano(&text).map_err(serde::de::Error::custom)?;
        Ok(UnixTime { sec, nanos })
    }
}

fn is_leap(year: i64) -> bool {
    year % 4 == 0 && (year % 100 != 0 || year % 400 == 0)
}

fn days_in_month(year: i64, month: u32) -> u32 {
    match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 if is_leap(year) => 29,
        2 => 28,
        _ => 0,
    }
}

// Howard Hinnant's days-from-civil algorithm; valid for all civil dates.
fn days_from_civil(year: i64, month: u32, day: u32) -> i64 {
    let y = if month <= 2 { year - 1 } else { year };
    let era = y.div_euclid(400);
    let yoe = y.rem_euclid(400) as u64;
    let mp = ((month as u64 + 9) % 12) as u64;
    let doy = (153 * mp + 2) / 5 + day as u64 - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    (era * 146097 + doe as i64) - 719468
}

fn civil_from_days(days: i64) -> (i64, u32, u32) {
    let z = days + 719468;
    let era = z.div_euclid(146097);
    let doe = z.rem_euclid(146097) as u64;
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let mut year = yoe as i64 + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let day = (doy - (153 * mp + 2) / 5 + 1) as u32;
    let month = (if mp < 10 { mp + 3 } else { mp - 9 }) as u32;
    if month <= 2 {
        year += 1;
    }
    (year, month, day)
}

pub fn parse_rfc3339_nano(text: &str) -> Result<(i64, u32), String> {
    // `2006-01-02T15:04:05(.999999999)?(Z|±07:00|±0700|±07)` like Go's
    // RFC3339Nano. Exactly four year digits and two for every other field.
    let err = || format!("invalid timestamp {text:?}");
    if text.len() < 20 {
        return Err(err());
    }
    let date = text.as_bytes();
    if date[4] != b'-' || date[7] != b'-' || date[10] != b'T' || date[13] != b':'
        || date[16] != b':'
    {
        return Err(err());
    }
    let num = |from: usize, to: usize| -> Result<i64, String> {
        text[from..to].parse::<i64>().map_err(|_| err())
    };
    let year = num(0, 4)?;
    let month = num(5, 7)?;
    let day = num(8, 10)?;
    let hour = num(11, 13)?;
    let minute = num(14, 16)?;
    let second = num(17, 19)?;
    if !(1..=12).contains(&month) || day < 1 || day > days_in_month(year, month as u32) as i64
        || hour > 23 || minute > 59 || second > 59
    {
        return Err(err());
    }
    let mut rest = &text[19..];
    let mut nanos: u32 = 0;
    if let Some(fraction) = rest.strip_prefix('.') {
        let digits = fraction.bytes().take_while(u8::is_ascii_digit).count();
        if digits == 0 || digits > 9 {
            return Err(err());
        }
        let mut scale = 100_000_000u32;
        for b in fraction.bytes().take(digits) {
            nanos += (b - b'0') as u32 * scale;
            scale /= 10;
        }
        rest = &fraction[digits..];
    }
    let mut offset: i64 = 0;
    if rest == "Z" {
        rest = "";
    } else if rest.len() >= 3 && (rest.as_bytes()[0] == b'+' || rest.as_bytes()[0] == b'-') {
        let sign = if rest.as_bytes()[0] == b'-' { -1 } else { 1 };
        let hour: i64 = rest[1..3].parse().map_err(|_| err())?;
        let mut minute: i64 = 0;
        rest = &rest[3..];
        if let Some(tail) = rest.strip_prefix(':') {
            if tail.len() != 2 {
                return Err(err());
            }
            minute = tail.parse().map_err(|_| err())?;
            rest = "";
        } else if rest.len() == 2 {
            minute = rest.parse().map_err(|_| err())?;
            rest = "";
        } else if !rest.is_empty() {
            return Err(err());
        }
        if hour > 23 || minute > 59 {
            return Err(err());
        }
        offset = sign * (hour * 3600 + minute * 60);
    }
    if !rest.is_empty() {
        return Err(err());
    }
    let days = days_from_civil(year, month as u32, day as u32);
    let sec = days * 86400 + hour * 3600 + minute * 60 + second - offset;
    Ok((sec, nanos))
}

pub fn format_rfc3339_nano(sec: i64, nanos: u32) -> String {
    let days = sec.div_euclid(86400);
    let clock = sec.rem_euclid(86400);
    let (year, month, day) = civil_from_days(days);
    let mut out = format!(
        "{:04}-{:02}-{:02}T{:02}:{:02}:{:02}",
        year,
        month,
        day,
        clock / 3600,
        clock % 3600 / 60,
        clock % 60
    );
    if nanos != 0 {
        let mut fraction = format!("{nanos:09}");
        while fraction.ends_with('0') {
            fraction.pop();
        }
        out.push('.');
        out.push_str(&fraction);
    }
    out.push('Z');
    out
}

/// Go `,string` integer form: encoded as a JSON string of digits.
/// A JSON null decodes to zero like encoding/json's scalar null no-op.
pub mod i64_string {
    use super::*;

    pub fn serialize<S: Serializer>(value: &i64, ser: S) -> Result<S::Ok, S::Error> {
        ser.serialize_str(&value.to_string())
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(de: D) -> Result<i64, D::Error> {
        let text = Option::<String>::deserialize(de)?;
        match text {
            None => Ok(0),
            Some(text) => text.parse::<i64>().map_err(serde::de::Error::custom),
        }
    }
}

/// Go `,string,omitempty` integer form: omitted when zero.
pub mod i64_string_omitted {
    use super::*;

    pub fn serialize<S: Serializer>(value: &i64, ser: S) -> Result<S::Ok, S::Error> {
        ser.serialize_str(&value.to_string())
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(de: D) -> Result<i64, D::Error> {
        i64_string::deserialize(de)
    }
}

/// Null-tolerant scalar decoding: encoding/json leaves a scalar field
/// unchanged on JSON null instead of failing, so every scalar below
/// accepts null as its zero value.
pub mod null_tolerant {
    use super::*;

    pub fn string<'de, D: Deserializer<'de>>(de: D) -> Result<String, D::Error> {
        Ok(Option::<String>::deserialize(de)?.unwrap_or_default())
    }

    pub fn boolean<'de, D: Deserializer<'de>>(de: D) -> Result<bool, D::Error> {
        Ok(Option::<bool>::deserialize(de)?.unwrap_or_default())
    }

    pub fn integer<'de, D: Deserializer<'de>>(de: D) -> Result<i64, D::Error> {
        Ok(Option::<i64>::deserialize(de)?.unwrap_or_default())
    }

    pub fn integer32<'de, D: Deserializer<'de>>(de: D) -> Result<i32, D::Error> {
        Ok(Option::<i32>::deserialize(de)?.unwrap_or_default())
    }

    pub fn time<'de, D: Deserializer<'de>>(de: D) -> Result<UnixTime, D::Error> {
        Ok(Option::<UnixTime>::deserialize(de)?.unwrap_or(UnixTime { sec: 0, nanos: 0 }))
    }
}

pub fn is_zero(value: &i64) -> bool {
    *value == 0
}

/// Go []byte JSON form: standard padded base64.
pub mod base64_bytes {
    use super::*;

    const ALPHABET: &[u8; 64] =
        b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

    pub fn encode(data: &[u8]) -> String {
        let mut out = String::with_capacity(data.len().div_ceil(3) * 4);
        for chunk in data.chunks(3) {
            let n = chunk.len();
            let mut v: u32 = 0;
            for &b in chunk {
                v = (v << 8) | b as u32;
            }
            v <<= 8 * (3 - n);
            for i in 0..4 {
                if i <= n {
                    out.push(ALPHABET[((v >> (18 - i * 6)) & 63) as usize] as char);
                } else {
                    out.push('=');
                }
            }
        }
        out
    }

    fn decode_value(b: u8) -> Result<u8, String> {
        match b {
            b'A'..=b'Z' => Ok(b - b'A'),
            b'a'..=b'z' => Ok(b - b'a' + 26),
            b'0'..=b'9' => Ok(b - b'0' + 52),
            b'+' => Ok(62),
            b'/' => Ok(63),
            _ => Err(format!("invalid base64 byte {b:?}")),
        }
    }

    pub fn decode(text: &str) -> Result<Vec<u8>, String> {
        let bytes = text.as_bytes();
        if bytes.len() % 4 != 0 {
            return Err("invalid base64 length".to_string());
        }
        let mut out = Vec::with_capacity(bytes.len() / 4 * 3);
        for chunk in bytes.chunks(4) {
            // Padding rules mirror encoding/base64 StdEncoding strictly:
            // at most two trailing `=`, and no data after padding.
            let pad = chunk.iter().rev().take_while(|b| **b == b'=').count();
            if pad > 2 || chunk[..4 - pad].contains(&b'=') {
                return Err("invalid base64 padding".to_string());
            }
            let mut v: u32 = 0;
            for &b in &chunk[..4 - pad] {
                v = (v << 6) | decode_value(b)? as u32;
            }
            v <<= 6 * pad;
            let word = v.to_be_bytes();
            out.extend_from_slice(&word[1..4 - pad]);
        }
        Ok(out)
    }

    pub fn serialize<S: Serializer>(value: &Vec<u8>, ser: S) -> Result<S::Ok, S::Error> {
        ser.serialize_str(&encode(value))
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(de: D) -> Result<Vec<u8>, D::Error> {
        // encoding/json emits null for a nil slice; accept it as empty.
        let text = Option::<String>::deserialize(de)?.unwrap_or_default();
        decode(&text).map_err(serde::de::Error::custom)
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Grant {
    #[serde(deserialize_with = "null_tolerant::string")]
    pub id: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub connection_id: String,
    #[serde(with = "i64_string")]
    pub user_id: i64,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub project_id: String,
    #[serde(deserialize_with = "null_tolerant::integer")]
    pub revision: i64,
    #[serde(deserialize_with = "null_tolerant::boolean")]
    pub revoked: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GrantRequest {
    #[serde(deserialize_with = "null_tolerant::string")]
    pub connection_id: String,
    #[serde(with = "i64_string")]
    pub user_id: i64,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub project_id: String,
    #[serde(deserialize_with = "null_tolerant::boolean")]
    pub confirm_subscription: bool,
    #[serde(deserialize_with = "null_tolerant::boolean")]
    pub confirm_credential_exposure: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AcquireRequest {
    #[serde(
        default,
        skip_serializing_if = "is_zero",
        with = "i64_string_omitted"
    )]
    pub repository_id: i64,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub provider_id: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub execution_id: String,
    #[serde(with = "i64_string")]
    pub actor_id: i64,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub connection_id: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub project_id: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub kind: String,
    #[serde(deserialize_with = "null_tolerant::time")]
    pub deadline: UnixTime,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub role: String,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Binding {
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub child_id: String,
    #[serde(deserialize_with = "null_tolerant::integer32", default, skip_serializing_if = "is_zero_i32")]
    pub uid: i32,
    #[serde(deserialize_with = "null_tolerant::integer32", default, skip_serializing_if = "is_zero_i32")]
    pub gid: i32,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub scope: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub credential_root: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub invocation_id: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub kind: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub id: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub project: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub login: String,
    #[serde(deserialize_with = "null_tolerant::integer")]
    pub generation: i64,
}

pub fn is_zero_i32(value: &i32) -> bool {
    *value == 0
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Lease {
    #[serde(
        default,
        skip_serializing_if = "is_zero",
        with = "i64_string_omitted"
    )]
    pub repository_id: i64,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub provider_id: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub id: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub connection_id: String,
    #[serde(deserialize_with = "null_tolerant::integer")]
    pub generation: i64,
    #[serde(with = "i64_string")]
    pub actor_id: i64,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub project_id: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub execution_id: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub kind: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub role: String,
    #[serde(deserialize_with = "null_tolerant::time")]
    pub deadline: UnixTime,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub grant_id: String,
    #[serde(deserialize_with = "null_tolerant::integer", default, skip_serializing_if = "is_zero")]
    pub grant_revision: i64,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub binding: Option<Binding>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Execution {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub binding: Option<Binding>,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub kind: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub execution_id: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub digest: String,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub state: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub lease_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Event {
    #[serde(with = "i64_string")]
    pub id: i64,
    #[serde(deserialize_with = "null_tolerant::time")]
    pub time: UnixTime,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub action: String,
    #[serde(with = "i64_string")]
    pub owner_id: i64,
    #[serde(with = "i64_string")]
    pub actor_id: i64,
    #[serde(deserialize_with = "null_tolerant::string")]
    pub connection_id: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub lease_id: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub project_id: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub grant_id: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub execution_id: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub kind: String,
    #[serde(deserialize_with = "null_tolerant::integer")]
    pub generation: i64,
}

/// The private Unix HTTP protocol. Browser handlers never accept it.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Request {
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub provider_id: String,
    #[serde(default, with = "i64_string")]
    pub owner_id: i64,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub id: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub label: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub project_id: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub kind: String,
    #[serde(deserialize_with = "null_tolerant::string", default, skip_serializing_if = "String::is_empty")]
    pub execution_id: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub grant: Option<GrantRequest>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub acquire: Option<AcquireRequest>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub binding: Option<Binding>,
    #[serde(default, skip_serializing_if = "Option::is_none", with = "base64_bytes_option")]
    pub credential: Option<Vec<u8>>,
}

/// `[]byte` with `omitempty`: absent when empty, base64 when present. Go
/// omits both nil and zero-length; `Some(vec![])` never serializes because
/// the broker only sets credentials from non-empty input.
pub mod base64_bytes_option {
    use super::base64_bytes;
    use super::*;

    pub fn serialize<S: Serializer>(value: &Option<Vec<u8>>, ser: S) -> Result<S::Ok, S::Error> {
        match value {
            Some(data) => ser.serialize_str(&base64_bytes::encode(data)),
            None => ser.serialize_none(),
        }
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(de: D) -> Result<Option<Vec<u8>>, D::Error> {
        let text = String::deserialize(de)?;
        base64_bytes::decode(&text).map(Some).map_err(serde::de::Error::custom)
    }
}

/// DeliveryWire exists only on the runtime socket; never on the admin socket.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeliveryWire {
    pub lease: Lease,
    #[serde(with = "base64_bytes")]
    pub credential: Vec<u8>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ErrorKind {
    Denied,
    Busy,
    Stale,
    Uncertain,
    NotFound,
    Internal,
}

#[derive(Debug)]
pub struct Error {
    kind: ErrorKind,
    message: String,
}

impl Error {
    pub fn denied(message: impl Into<String>) -> Error {
        Error { kind: ErrorKind::Denied, message: message.into() }
    }
    pub fn busy() -> Error {
        Error { kind: ErrorKind::Busy, message: "subscription is in use".to_string() }
    }
    pub fn stale() -> Error {
        Error { kind: ErrorKind::Stale, message: "identity generation changed".to_string() }
    }
    pub fn uncertain() -> Error {
        Error { kind: ErrorKind::Uncertain, message: "subscription requires reconnection".to_string() }
    }
    pub fn not_found() -> Error {
        Error { kind: ErrorKind::NotFound, message: "identity execution missing".to_string() }
    }
    pub fn internal(message: impl Into<String>) -> Error {
        Error { kind: ErrorKind::Internal, message: message.into() }
    }
    pub fn kind(&self) -> ErrorKind {
        self.kind
    }
    pub fn is_denied(&self) -> bool {
        self.kind == ErrorKind::Denied
    }
    pub fn is_not_found(&self) -> bool {
        self.kind == ErrorKind::NotFound
    }
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.message)
    }
}

impl std::error::Error for Error {}

impl From<serde_json::Error> for Error {
    fn from(err: serde_json::Error) -> Error {
        Error::internal(err.to_string())
    }
}

impl From<std::io::Error> for Error {
    fn from(err: std::io::Error) -> Error {
        Error::internal(err.to_string())
    }
}

impl GrantRequest {
    pub fn validate(&self) -> Result<(), Error> {
        if self.connection_id.is_empty()
            || self.user_id <= 0
            || self.project_id.is_empty()
            || !self.confirm_subscription
            || !self.confirm_credential_exposure
        {
            return Err(Error::denied("identity authority denied"));
        }
        Ok(())
    }
}

impl AcquireRequest {
    pub fn validate(&self, now: UnixTime) -> Result<(), Error> {
        if !provider_valid(&self.provider_id)
            || self.actor_id <= 0
            || self.connection_id.is_empty()
            || self.execution_id.is_empty()
            || (self.kind != FACTORY && self.kind != TERMINAL)
            || !(self.deadline > now)
            || self.deadline > now.add_hours(24)
        {
            return Err(Error::denied("identity authority denied"));
        }
        Ok(())
    }
}

impl Binding {
    pub fn validate(&self) -> Result<(), Error> {
        if self.id.is_empty() || self.generation <= 0 || (self.kind != FACTORY && self.kind != TERMINAL) {
            return Err(Error::denied("identity authority denied"));
        }
        if self.kind == TERMINAL && (self.project.is_empty() || self.login.trim().is_empty()) {
            return Err(Error::denied("identity authority denied"));
        }
        Ok(())
    }
}

impl Execution {
    pub fn validate(&self) -> Result<(), Error> {
        if (self.kind != FACTORY && self.kind != TERMINAL)
            || self.execution_id.is_empty()
            || self.execution_id.len() > 128
        {
            return Err(Error::denied("identity authority denied"));
        }
        match self.state.as_str() {
            EXECUTION_PENDING | EXECUTION_LIVE | EXECUTION_TERMINAL => {}
            _ => return Err(Error::denied("identity authority denied")),
        }
        if self.state != EXECUTION_TERMINAL && self.digest.is_empty() {
            return Err(Error::denied("identity authority denied"));
        }
        if let Some(binding) = &self.binding {
            binding.validate()?;
            if binding.kind != self.kind {
                return Err(Error::denied("identity authority denied"));
            }
        }
        Ok(())
    }
}

/// Binds an acquire request to its execution identity. The deadline is a
/// bound, not identity: retries keep the original lease deadline.
pub fn acquisition_digest(input: &AcquireRequest) -> String {
    let canonical = [
        input.kind.as_str(),
        input.execution_id.as_str(),
        input.provider_id.as_str(),
        input.connection_id.as_str(),
        input.role.trim().to_lowercase().as_str(),
        input.project_id.as_str(),
        &input.actor_id.to_string(),
        &input.repository_id.to_string(),
    ]
    .join("\x00");
    identity_providers::sha256::hex(&identity_providers::sha256::digest(canonical.as_bytes()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn go_time_vectors_round_trip() {
        // Captured from encoding/json with the pinned Go toolchain.
        for (text, sec, nanos) in [
            ("2026-10-04T18:30:05Z", 1791138605, 0),
            ("2026-10-04T18:30:05.123Z", 1791138605, 123_000_000),
            ("2026-10-04T18:30:05.123456Z", 1791138605, 123_456_000),
            ("2026-10-04T18:30:05.123456789Z", 1791138605, 123_456_789),
            ("2026-10-04T18:30:05.12Z", 1791138605, 120_000_000),
        ] {
            let (parsed_sec, parsed_nanos) = parse_rfc3339_nano(text).unwrap();
            assert_eq!((parsed_sec, parsed_nanos), (sec, nanos), "{text}");
            assert_eq!(format_rfc3339_nano(sec, nanos), text, "{text}");
        }
        // Offsets normalize to the same instant Go encodes with Z.
        let (sec, nanos) = parse_rfc3339_nano("2026-10-04T20:30:05+02:00").unwrap();
        assert_eq!((sec, nanos), (1791138605, 0));
        assert_eq!(format_rfc3339_nano(sec, nanos), "2026-10-04T18:30:05Z");
    }

    #[test]
    fn timestamps_reject_malformed_input() {
        for text in [
            "",
            "2026-10-04",
            "2026-13-04T18:30:05Z",
            "2026-10-32T18:30:05Z",
            "2026-10-04T25:30:05Z",
            "2026-10-04T18:30:05",
            "2026-10-04T18:30:05.Z",
            "2026-10-04T18:30:05.1234567890Z",
            "2026-10-04T18:30:05+25:00",
            "2026-10-04 18:30:05Z",
        ] {
            assert!(parse_rfc3339_nano(text).is_err(), "admitted {text:?}");
        }
    }

    #[test]
    fn lease_wire_shape_matches_go() {
        let lease = Lease {
            repository_id: 7,
            provider_id: "codex".to_string(),
            id: "lease-1".to_string(),
            connection_id: "conn-1".to_string(),
            generation: 1,
            actor_id: 2,
            project_id: "project".to_string(),
            execution_id: "execution".to_string(),
            kind: "factory".to_string(),
            role: String::new(),
            deadline: UnixTime { sec: 1791138605, nanos: 0 },
            grant_id: String::new(),
            grant_revision: 0,
            binding: None,
        };
        assert_eq!(
            serde_json::to_string(&lease).unwrap(),
            r#"{"repository_id":"7","provider_id":"codex","id":"lease-1","connection_id":"conn-1","generation":1,"actor_id":"2","project_id":"project","execution_id":"execution","kind":"factory","deadline":"2026-10-04T18:30:05Z"}"#
        );
    }

    #[test]
    fn base64_matches_go_byte_form() {
        assert_eq!(base64_bytes::encode(b""), "");
        assert_eq!(base64_bytes::encode(b"f"), "Zg==");
        assert_eq!(base64_bytes::encode(b"fo"), "Zm8=");
        assert_eq!(base64_bytes::encode(b"foo"), "Zm9v");
        assert_eq!(base64_bytes::encode(br#"{"a":1}"#), "eyJhIjoxfQ==");
        for data in [b"".as_slice(), b"f", b"fo", b"foo", b"foob", b"fooba", b"foobar"] {
            assert_eq!(base64_bytes::decode(&base64_bytes::encode(data)).unwrap(), data);
        }
        assert!(base64_bytes::decode("Zg").is_err());
        assert!(base64_bytes::decode("Zg=a").is_err());
        assert!(base64_bytes::decode("====").is_err());
    }

    #[test]
    fn null_scalars_match_go_noop() {
        let request: Request = serde_json::from_str(
            r#"{"provider_id":null,"owner_id":null,"id":null,"label":null,"project_id":null,"kind":null,"execution_id":null,"grant":{"connection_id":null,"user_id":null,"project_id":null,"confirm_subscription":null,"confirm_credential_exposure":null}}"#,
        )
        .unwrap();
        assert_eq!(request.owner_id, 0);
        assert!(request.grant.unwrap().connection_id.is_empty());
        let wire: DeliveryWire = serde_json::from_str(
            r#"{"lease":{"provider_id":"codex","id":"l","connection_id":"c","generation":1,"actor_id":"1","project_id":"p","execution_id":"e","kind":"factory","deadline":"2026-10-04T18:30:05Z"},"credential":null}"#,
        )
        .unwrap();
        assert!(wire.credential.is_empty());
        let lease: Lease = serde_json::from_str(
            r#"{"provider_id":"codex","id":"l","connection_id":"c","generation":null,"actor_id":"1","project_id":"p","execution_id":"e","kind":"factory","deadline":null}"#,
        )
        .unwrap();
        assert_eq!(lease.generation, 0);
        assert_eq!((lease.deadline.sec, lease.deadline.nanos), (0, 0));
    }

    #[test]
    fn digest_matches_go_acquisition() {
        // Reference computed from the Go AcquisitionDigest on the same input.
        let request = AcquireRequest {
            repository_id: 7,
            provider_id: "codex".to_string(),
            execution_id: "execution".to_string(),
            actor_id: 2,
            connection_id: "conn-1".to_string(),
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline: UnixTime { sec: 1791138605, nanos: 0 },
            role: " Soda-Coder ".to_string(),
        };
        // sha256 of the NUL-joined canonical form, verified with sha256sum.
        assert_eq!(
            acquisition_digest(&request),
            "c60bddffb5aa08c8f38f9efd3e8bafc8a633e3923f00baeda96584fea5cbde57"
        );
    }
}
