// Enrollment wire shapes shared by the providers, byte-identical to
// internal/identity: field names, order and the `owner_id,string`
// string-encoded integer form.
use serde::{Deserialize, Deserializer, Serialize, Serializer};

pub const MAX_CREDENTIAL_BYTES: usize = 256 << 10;

/// A credential is non-empty, bounded and well-formed JSON.
pub fn credential_valid(data: &[u8]) -> bool {
    !data.is_empty()
        && data.len() <= MAX_CREDENTIAL_BYTES
        && serde_json::from_slice::<serde_json::Value>(data).is_ok()
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Connection {
    pub provider_id: String,
    pub id: String,
    #[serde(with = "i64_string")]
    pub owner_id: i64,
    pub label: String,
    pub email: String,
    pub plan: String,
    pub generation: i64,
    pub state: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Enrollment {
    pub provider_id: String,
    pub id: String,
    pub verification_url: String,
    pub user_code: String,
    pub state: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub error: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub connection: Option<Connection>,
}

/// Go `,string` integer form: encoded as a JSON string of digits, decoded
/// from a JSON string only.
mod i64_string {
    use super::*;

    pub fn serialize<S: Serializer>(value: &i64, ser: S) -> Result<S::Ok, S::Error> {
        ser.serialize_str(&value.to_string())
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(de: D) -> Result<i64, D::Error> {
        let text = String::deserialize(de)?;
        text.parse::<i64>().map_err(serde::de::Error::custom)
    }
}
