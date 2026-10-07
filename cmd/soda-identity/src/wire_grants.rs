// Grant and acquisition DTOs, extracted from wire.rs (A05.M).
use crate::wire::{provider_valid, FACTORY, TERMINAL};
use crate::wire_errors::Error;
use crate::wire_time::UnixTime;
use serde::{Deserialize, Serialize};
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Grant {
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
    #[serde(default, with = "crate::wire_scalars::i64_string")]
    pub user_id: i64,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub project_id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::integer"
    )]
    pub revision: i64,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::boolean"
    )]
    pub revoked: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct GrantRequest {
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub connection_id: String,
    #[serde(default, with = "crate::wire_scalars::i64_string")]
    pub user_id: i64,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub project_id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::boolean"
    )]
    pub confirm_subscription: bool,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::boolean"
    )]
    pub confirm_credential_exposure: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct AcquireRequest {
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
    pub execution_id: String,
    #[serde(default, with = "crate::wire_scalars::i64_string")]
    pub actor_id: i64,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub connection_id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub project_id: String,
    #[serde(
        default,
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    pub kind: String,
    #[serde(default, deserialize_with = "crate::wire_scalars::null_tolerant::time")]
    pub deadline: UnixTime,
    #[serde(
        deserialize_with = "crate::wire_scalars::null_tolerant::string",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub role: String,
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
            || now.add_hours(24).is_none_or(|limit| self.deadline > limit)
        {
            return Err(Error::denied("identity authority denied"));
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
    crate::providers::sha256::hex(&crate::providers::sha256::digest(canonical.as_bytes()))
}
