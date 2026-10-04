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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn connection_wire_shape_matches_go() {
        let conn = Connection {
            provider_id: "codex".to_string(),
            id: "conn-1".to_string(),
            owner_id: 7,
            label: "private subscription".to_string(),
            email: "soda-tester@example.invalid".to_string(),
            plan: "plus".to_string(),
            generation: 1,
            state: "ready".to_string(),
        };
        assert_eq!(
            serde_json::to_string(&conn).unwrap(),
            r#"{"provider_id":"codex","id":"conn-1","owner_id":"7","label":"private subscription","email":"soda-tester@example.invalid","plan":"plus","generation":1,"state":"ready"}"#
        );
    }

    #[test]
    fn enrollment_omits_empty_error() {
        let enrollment = Enrollment {
            provider_id: "muse".to_string(),
            id: "enrollment-1".to_string(),
            verification_url: "https://auth.meta.com/oauth/device/?code=ABCD-1234".to_string(),
            user_code: "ABCD-1234".to_string(),
            state: "pending".to_string(),
            error: String::new(),
            connection: None,
        };
        let wire = serde_json::to_string(&enrollment).unwrap();
        assert!(!wire.contains("error"), "unexpected error field: {wire}");
        assert!(
            !wire.contains("connection"),
            "unexpected connection: {wire}"
        );
        assert!(wire.starts_with(r#"{"provider_id":"muse","id":"enrollment-1""#));
    }

    #[test]
    fn credential_bounds_match_go() {
        assert!(credential_valid(br#"{"tokens":{}}"#));
        assert!(!credential_valid(b""));
        assert!(!credential_valid(b"not json"));
        assert!(!credential_valid(&vec![b'a'; MAX_CREDENTIAL_BYTES + 1]));
    }
}
