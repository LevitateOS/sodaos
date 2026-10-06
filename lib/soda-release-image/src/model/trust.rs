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

/// Parse a PKIX `PUBLIC KEY` PEM and require a P-256 EC key, structurally.
/// (The deliver owner additionally verifies the point lies on the curve;
/// fixtures here use real P-256 keys so both agree.)
fn parse_trust_public_key(pem_text: &str) -> Result<Vec<u8>, Error> {
    let refused = || Error::msg(sys::refused());
    let begin = "-----BEGIN PUBLIC KEY-----";
    let end = "-----END PUBLIC KEY-----";
    let start = pem_text.find(begin).ok_or_else(refused)? + begin.len();
    let stop = pem_text[start..].find(end).ok_or_else(refused)? + start;
    let body: String = pem_text[start..stop]
        .chars()
        .filter(|c| !c.is_whitespace())
        .collect();
    let rest = pem_text[stop + end.len()..].trim();
    if !rest.is_empty() {
        return Err(refused());
    }
    use base64::Engine;
    let der = base64::engine::general_purpose::STANDARD
        .decode(body.as_bytes())
        .map_err(|_| refused())?;
    if !is_p256_spki(&der) {
        // Go distinguishes a non-P-256 parse here; keep the owner message.
        return Err(Error::msg("native P-256 Sigstore public key required"));
    }
    Ok(der)
}

fn read_der_length(bytes: &[u8], pos: &mut usize) -> Option<usize> {
    if *pos >= bytes.len() {
        return None;
    }
    let first = bytes[*pos];
    *pos += 1;
    if first & 0x80 == 0 {
        return Some(first as usize);
    }
    let count = (first & 0x7f) as usize;
    if count == 0 || count > 4 || *pos + count > bytes.len() {
        return None;
    }
    let mut length = 0usize;
    for _ in 0..count {
        length = (length << 8) | bytes[*pos] as usize;
        *pos += 1;
    }
    Some(length)
}

fn is_p256_spki(der: &[u8]) -> bool {
    let mut pos = 0;
    // SEQUENCE
    if der.get(pos) != Some(&0x30) {
        return false;
    }
    pos += 1;
    let outer = match read_der_length(der, &mut pos) {
        Some(length) => length,
        None => return false,
    };
    if pos + outer != der.len() {
        return false;
    }
    // algorithm SEQUENCE { ecPublicKey OID, secp256r1 OID }
    if der.get(pos) != Some(&0x30) {
        return false;
    }
    pos += 1;
    let alg_len = match read_der_length(der, &mut pos) {
        Some(length) => length,
        None => return false,
    };
    let alg_end = pos + alg_len;
    // OID 1.2.840.10045.2.1
    let ec_oid: [u8; 9] = [0x06, 0x07, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x02, 0x01];
    if der.get(pos..pos + ec_oid.len()) != Some(&ec_oid[..]) {
        return false;
    }
    pos += ec_oid.len();
    // OID 1.2.840.10045.3.1.7 (secp256r1)
    let curve_oid: [u8; 10] = [0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07];
    if der.get(pos..pos + curve_oid.len()) != Some(&curve_oid[..]) {
        return false;
    }
    pos += curve_oid.len();
    if pos != alg_end {
        return false;
    }
    // BIT STRING, 0 unused bits, uncompressed 65-byte point.
    if der.get(pos) != Some(&0x03) {
        return false;
    }
    pos += 1;
    let bit_len = match read_der_length(der, &mut pos) {
        Some(length) => length,
        None => return false,
    };
    if bit_len != 66 || pos + bit_len != der.len() {
        return false;
    }
    der[pos] == 0x00 && der[pos + 1] == 0x04
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
