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

#[derive(Default)]
struct TrustWire {
    format: i64,
    prefix: String,
    epoch: u64,
    keys: crate::jsonio::OrderedMap<Vec<String>>,
    not_before: i64,
    max_age_seconds: i64,
    clock_skew_seconds: i64,
    minimum_sequence: crate::jsonio::OrderedMap<u64>,
}

crate::jsonio::case_record!(TrustWire, {
    format: i64 => "Format",
    prefix: String => "Prefix",
    epoch: u64 => "Epoch",
    keys: crate::jsonio::OrderedMap<Vec<String>> => "Keys",
    not_before: i64 => "NotBefore",
    max_age_seconds: i64 => "MaxAgeSeconds",
    clock_skew_seconds: i64 => "ClockSkewSeconds",
    minimum_sequence: crate::jsonio::OrderedMap<u64> => "MinimumSequence",
});

impl Trust {
    pub fn parse(text: &str) -> Result<Trust, Error> {
        let wire: TrustWire = jsonio::parse(text).map_err(|_| Error::msg(sys::refused()))?;
        Ok(Trust {
            format: wire.format,
            prefix: wire.prefix,
            epoch: wire.epoch,
            keys: wire.keys.0,
            not_before: wire.not_before,
            max_age_seconds: wire.max_age_seconds,
            clock_skew_seconds: wire.clock_skew_seconds,
            minimum_sequence: wire.minimum_sequence.0,
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

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SecretFiles {
    pub key: String,
    pub passphrase: String,
}

impl SecretFiles {
    pub fn parse(text: &str) -> Result<SecretFiles, Error> {
        if text.trim() == "null" {
            return Ok(SecretFiles::default());
        }
        jsonio::parse(text).map_err(|_| Error::msg(sys::refused()))
    }
}

crate::jsonio::case_record!(SecretFiles, {
    key: String => "Key",
    passphrase: String => "Passphrase",
});
