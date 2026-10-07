// Binding, lease, execution and event DTOs, extracted from wire.rs (A05.M).
use crate::wire::{EXECUTION_LIVE, EXECUTION_PENDING, EXECUTION_TERMINAL, FACTORY, TERMINAL};
use crate::wire_errors::Error;
use crate::wire_time::UnixTime;
use serde::{Deserialize, Serialize};
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Binding {
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub child_id: String,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::integer32",
        default,
        skip_serializing_if = "crate::wire_scalars::is_zero_i32"
    )]
    pub uid: i32,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::integer32",
        default,
        skip_serializing_if = "crate::wire_scalars::is_zero_i32"
    )]
    pub gid: i32,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub scope: String,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub credential_root: String,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub invocation_id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub kind: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub project: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub login: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::integer"
    )]
    pub generation: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Lease {
    #[serde(
        default,
        skip_serializing_if = "crate::wire_scalars::is_zero",
        with = "crate::wire_scalars::i64_string_omitted"
    )]
    pub repository_id: i64,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub provider_id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub connection_id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::integer"
    )]
    pub generation: i64,
    #[serde(default, with = "crate::wire_scalars::i64_string")]
    pub actor_id: i64,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub project_id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub execution_id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub kind: String,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub role: String,
    #[serde(default, deserialize_with = "crate::wire_scalars::null_tolerant::time")]
    pub deadline: UnixTime,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub grant_id: String,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::integer",
        default,
        skip_serializing_if = "crate::wire_scalars::is_zero"
    )]
    pub grant_revision: i64,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub binding: Option<Binding>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Execution {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub binding: Option<Binding>,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub kind: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub execution_id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub digest: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub state: String,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub lease_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Event {
    #[serde(default, with = "crate::wire_scalars::i64_string")]
    pub id: i64,
    #[serde(default, deserialize_with = "crate::wire_scalars::null_tolerant::time")]
    pub time: UnixTime,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub action: String,
    #[serde(default, with = "crate::wire_scalars::i64_string")]
    pub owner_id: i64,
    #[serde(default, with = "crate::wire_scalars::i64_string")]
    pub actor_id: i64,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub connection_id: String,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub lease_id: String,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub project_id: String,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub grant_id: String,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub execution_id: String,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub kind: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::integer"
    )]
    pub generation: i64,
}

impl Binding {
    pub fn validate(&self) -> Result<(), Error> {
        if self.id.is_empty()
            || self.generation <= 0
            || (self.kind != FACTORY && self.kind != TERMINAL)
        {
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
