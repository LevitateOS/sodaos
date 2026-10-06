use soda_json::JsonValue;

use crate::error::Error;
use crate::jsonio;
use crate::sys;

use super::{deliver_hash, valid_repository_prefix};

// ---------------------------------------------------------------------------
// deliver.Trust
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Trust {
    pub format: i64,
    pub prefix: String,
    pub epoch: u64,
    pub keys: Vec<(String, Vec<String>)>,
    pub not_before: i64,
    pub max_age_seconds: i64,
    pub clock_skew_seconds: i64,
    pub minimum_sequence: Vec<(String, u64)>,
}

impl Trust {
    pub fn parse(value: &JsonValue) -> Result<Trust, Error> {
        jsonio::check_no_unknown(
            value,
            &[
                "Format",
                "Prefix",
                "Epoch",
                "Keys",
                "NotBefore",
                "MaxAgeSeconds",
                "ClockSkewSeconds",
                "MinimumSequence",
            ],
        )
        .map_err(|_| Error::msg(sys::refused()))?;
        let mut keys = Vec::new();
        if let JsonValue::Object(entries) =
            jsonio::require_object(value, "Keys").map_err(|_| Error::msg(sys::refused()))?
        {
            for (role, list) in entries {
                let JsonValue::Array(items) = list else {
                    return Err(Error::msg(sys::refused()));
                };
                let mut role_keys = Vec::new();
                for item in items {
                    match item {
                        JsonValue::Str(s) => role_keys.push(s.clone()),
                        _ => return Err(Error::msg(sys::refused())),
                    }
                }
                keys.push((role.clone(), role_keys));
            }
        }
        let mut minimum_sequence = Vec::new();
        if let JsonValue::Object(entries) = jsonio::require_object(value, "MinimumSequence")
            .map_err(|_| Error::msg(sys::refused()))?
        {
            for (role, number) in entries {
                let JsonValue::Number(raw) = number else {
                    return Err(Error::msg(sys::refused()));
                };
                let sequence: u64 = raw.parse().map_err(|_| Error::msg(sys::refused()))?;
                minimum_sequence.push((role.clone(), sequence));
            }
        }
        Ok(Trust {
            format: jsonio::require_i64(value, "Format").map_err(|_| Error::msg(sys::refused()))?,
            prefix: jsonio::require_string(value, "Prefix")
                .map_err(|_| Error::msg(sys::refused()))?,
            epoch: jsonio::require_u64(value, "Epoch").map_err(|_| Error::msg(sys::refused()))?,
            keys,
            not_before: jsonio::require_i64(value, "NotBefore")
                .map_err(|_| Error::msg(sys::refused()))?,
            max_age_seconds: jsonio::require_i64(value, "MaxAgeSeconds")
                .map_err(|_| Error::msg(sys::refused()))?,
            clock_skew_seconds: jsonio::require_i64(value, "ClockSkewSeconds")
                .map_err(|_| Error::msg(sys::refused()))?,
            minimum_sequence,
        })
    }

    fn role_keys(&self, role: &str) -> Vec<String> {
        self.keys
            .iter()
            .find(|(r, _)| r == role)
            .map(|(_, keys)| keys.clone())
            .unwrap_or_default()
    }

    fn minimum(&self, role: &str) -> u64 {
        self.minimum_sequence
            .iter()
            .find(|(r, _)| r == role)
            .map(|(_, v)| *v)
            .unwrap_or(0)
    }

    pub fn validate(&self) -> Result<(), Error> {
        if !(self.format == 1
            && valid_repository_prefix(&self.prefix)
            && self.epoch != 0
            && self.keys.len() == 4
            && self.minimum_sequence.len() == 3
            && self.not_before > 0
            && self.max_age_seconds >= 60
            && self.max_age_seconds <= 7 * 86400
            && self.clock_skew_seconds >= 0
            && self.clock_skew_seconds <= 300)
        {
            return Err(Error::msg(sys::refused()));
        }
        let mut seen: Vec<String> = Vec::new();
        for role in ["artifact", "candidate", "preview", "stable"] {
            admit_trust_role_keys(&self.role_keys(role), &mut seen)?;
            if role != "artifact" && self.minimum(role) == 0 {
                return Err(Error::msg(sys::refused()));
            }
        }
        Ok(())
    }
}

fn admit_trust_role_keys(keys: &[String], seen: &mut Vec<String>) -> Result<(), Error> {
    if keys.is_empty() || keys.len() > 4 {
        return Err(Error::msg(sys::refused()));
    }
    for key in keys {
        let der = parse_trust_public_key(key)?;
        let fingerprint = deliver_hash(&der);
        if seen.contains(&fingerprint) {
            return Err(Error::msg("signer roles must not share keys"));
        }
        seen.push(fingerprint);
    }
    Ok(())
}

/// Parse one admitted P-256 PUBLIC KEY PEM and return its original DER.
fn parse_trust_public_key(pem_text: &str) -> Result<Vec<u8>, Error> {
    soda_build_tools::trust_key::parse_p256_public_key(pem_text)
        .map_err(|_| Error::msg(sys::refused()))
}

// ---------------------------------------------------------------------------
// deliver.Permit + deliver.SecretFiles
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Permit {
    pub format: i64,
    pub repository: String,
    pub digest: String,
    pub previous: String,
    pub expires: i64,
}

impl Permit {
    pub fn to_json(&self) -> JsonValue {
        JsonValue::Object(vec![
            (
                "Format".to_string(),
                JsonValue::Number(self.format.to_string()),
            ),
            (
                "Repository".to_string(),
                JsonValue::Str(self.repository.clone()),
            ),
            ("Digest".to_string(), JsonValue::Str(self.digest.clone())),
            (
                "Previous".to_string(),
                JsonValue::Str(self.previous.clone()),
            ),
            (
                "Expires".to_string(),
                JsonValue::Number(self.expires.to_string()),
            ),
        ])
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SecretFiles {
    pub key: String,
    pub passphrase: String,
}

impl SecretFiles {
    pub fn parse(value: &JsonValue) -> Result<SecretFiles, Error> {
        if matches!(value, JsonValue::Null) {
            return Ok(SecretFiles::default());
        }
        jsonio::check_no_unknown(value, &["Key", "Passphrase"])
            .map_err(|_| Error::msg(sys::refused()))?;
        Ok(SecretFiles {
            key: jsonio::require_string(value, "Key").map_err(|_| Error::msg(sys::refused()))?,
            passphrase: jsonio::require_string(value, "Passphrase")
                .map_err(|_| Error::msg(sys::refused()))?,
        })
    }

    pub fn to_json(&self) -> JsonValue {
        JsonValue::Object(vec![
            ("Key".to_string(), JsonValue::Str(self.key.clone())),
            (
                "Passphrase".to_string(),
                JsonValue::Str(self.passphrase.clone()),
            ),
        ])
    }
}
