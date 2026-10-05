// Identity wire records, byte-identical to internal/identity: field names,
// order, string-encoded integers and RFC3339Nano timestamps. Connection and
// Enrollment are reused from the providers crate so the shapes cannot drift.
use serde::{Deserialize, Serialize};

pub use crate::providers::types::{credential_valid, Connection, Enrollment};

pub const CODEX: &str = "codex";
pub const MUSE: &str = "muse";
pub const READY: &str = "ready";
pub const REAUTH: &str = "reauth";
pub const REVOKED: &str = "revoked";
pub const FACTORY: &str = "factory";
pub const TERMINAL: &str = "terminal";

pub const EXECUTION_PENDING: &str = "pending";
pub const EXECUTION_LIVE: &str = "live";
pub const EXECUTION_TERMINAL: &str = "terminal";

pub fn provider_valid(id: &str) -> bool {
    id == CODEX || id == MUSE
}

pub use crate::wire_errors::{Error, ErrorKind};
pub use crate::wire_execution::{Binding, Event, Execution, Lease};
pub use crate::wire_grants::{acquisition_digest, AcquireRequest, Grant, GrantRequest};
pub use crate::wire_scalars::{
    base64_bytes, base64_bytes_option, i64_string, i64_string_omitted, is_zero, is_zero_i32,
    null_tolerant,
};
pub use crate::wire_time::{format_rfc3339_nano, parse_rfc3339_nano, UnixTime};

/// The private Unix HTTP protocol. Browser handlers never accept it.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Request {
    #[serde(
        deserialize_with = "null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub provider_id: String,
    #[serde(default, with = "i64_string")]
    pub owner_id: i64,
    #[serde(
        deserialize_with = "null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub id: String,
    #[serde(
        deserialize_with = "null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub label: String,
    #[serde(
        deserialize_with = "null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub project_id: String,
    #[serde(
        deserialize_with = "null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub kind: String,
    #[serde(
        deserialize_with = "null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub execution_id: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub grant: Option<GrantRequest>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub acquire: Option<AcquireRequest>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub binding: Option<Binding>,
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        with = "base64_bytes_option"
    )]
    pub credential: Option<Vec<u8>>,
}

/// DeliveryWire exists only on the runtime socket; never on the admin socket.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeliveryWire {
    pub lease: Lease,
    #[serde(default, with = "base64_bytes")]
    pub credential: Vec<u8>,
}

#[cfg(test)]
#[path = "wire_tests.rs"]
mod tests;
