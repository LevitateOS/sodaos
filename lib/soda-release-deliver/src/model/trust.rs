use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet};

use crate::payload::{valid_repository_prefix, NAMES};
use crate::{hash_bytes, is_digest_ref, Error};

// ---------------------------------------------------------------------------
// Trust
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct Trust {
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub format: i64,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub prefix: String,
    #[serde(deserialize_with = "crate::json_serde::null_u64")]
    pub epoch: u64,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub keys: BTreeMap<String, Vec<String>>,
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub not_before: i64,
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub max_age_seconds: i64,
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub clock_skew_seconds: i64,
    #[serde(deserialize_with = "crate::json_serde::null_u64_map")]
    pub minimum_sequence: BTreeMap<String, u64>,
}

fn valid_trust_timing(t: &Trust) -> bool {
    t.not_before > 0
        && t.max_age_seconds >= 60
        && t.max_age_seconds <= 7 * 86400
        && t.clock_skew_seconds >= 0
        && t.clock_skew_seconds <= 300
}

fn valid_trust_envelope(t: &Trust) -> bool {
    t.format == 1
        && valid_repository_prefix(&t.prefix)
        && t.epoch != 0
        && t.keys.len() == 4
        && t.minimum_sequence.len() == 3
        && valid_trust_timing(t)
}

/// Parse one admitted P-256 PUBLIC KEY PEM and return its original DER.
fn parse_trust_public_key(key: &str) -> Result<Vec<u8>, Error> {
    soda_build_tools::trust_key::parse_p256_public_key(key)
        .map_err(|_| Error::msg("native P-256 Sigstore public key required"))
}

fn admit_trust_role_keys(keys: &[String], seen: &mut BTreeSet<String>) -> Result<(), Error> {
    if keys.is_empty() || keys.len() > 4 {
        return Err(Error::refused());
    }
    for key in keys {
        let der = parse_trust_public_key(key)?;
        let fingerprint = hash_bytes(&der);
        if !seen.insert(fingerprint) {
            return Err(Error::msg("signer roles must not share keys"));
        }
    }
    Ok(())
}

impl Trust {
    pub fn validate(&self) -> Result<(), Error> {
        if !valid_trust_envelope(self) {
            return Err(Error::refused());
        }
        let mut seen = BTreeSet::new();
        for role in ["artifact", "candidate", "preview", "stable"] {
            let empty = Vec::new();
            let keys = self.keys.get(role).unwrap_or(&empty);
            admit_trust_role_keys(keys, &mut seen)?;
            if role != "artifact" && self.minimum_sequence.get(role).copied().unwrap_or(0) == 0 {
                return Err(Error::refused());
            }
        }
        Ok(())
    }

    pub fn role(&self, repository: &str) -> Result<String, Error> {
        for name in ["host", "release", "media"].into_iter().chain(NAMES) {
            if repository == format!("{}-{name}", self.prefix) {
                return Ok("artifact".to_string());
            }
        }
        for channel in ["candidate", "preview", "stable"] {
            if repository == format!("{}-channel-{channel}", self.prefix) {
                return Ok(channel.to_string());
            }
        }
        Err(Error::refused())
    }

    pub fn reference(&self, reference: &str) -> Result<(String, String), Error> {
        let (repo, digest) = reference.split_once('@').ok_or_else(Error::refused)?;
        if !is_digest_ref(digest) {
            return Err(Error::refused());
        }
        let role = self.role(repo)?;
        Ok((repo.to_string(), role))
    }
}
