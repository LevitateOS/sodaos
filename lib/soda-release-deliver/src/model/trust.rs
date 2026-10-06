use std::collections::{BTreeMap, BTreeSet};

use soda_json::JsonValue;

use crate::jsonx::{as_u64, Binder, Emit, Emitter};
use crate::payload::{
    decode_opt_i64, decode_opt_string, decode_opt_u64, valid_repository_prefix, NAMES,
};
use crate::{hash_bytes, is_digest_ref, Error};

// ---------------------------------------------------------------------------
// Trust
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Trust {
    pub format: i64,
    pub prefix: String,
    pub epoch: u64,
    pub keys: BTreeMap<String, Vec<String>>,
    pub not_before: i64,
    pub max_age_seconds: i64,
    pub clock_skew_seconds: i64,
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

    pub fn decode(value: &JsonValue) -> Result<Trust, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid trust".to_string())?;
        let mut trust = Trust {
            format: decode_opt_i64(&mut b, "Format")?,
            prefix: decode_opt_string(&mut b, "Prefix")?,
            epoch: decode_opt_u64(&mut b, "Epoch")?,
            keys: BTreeMap::new(),
            not_before: decode_opt_i64(&mut b, "NotBefore")?,
            max_age_seconds: decode_opt_i64(&mut b, "MaxAgeSeconds")?,
            clock_skew_seconds: decode_opt_i64(&mut b, "ClockSkewSeconds")?,
            minimum_sequence: BTreeMap::new(),
        };
        if let Some(entries) = b
            .entries("Keys")
            .map_err(|_| "invalid field Keys".to_string())?
        {
            for (role, item) in entries {
                match item {
                    JsonValue::Array(items) => {
                        let mut keys = Vec::new();
                        for key in items {
                            match key {
                                JsonValue::Str(s) => keys.push(s.clone()),
                                _ => return Err("invalid field Keys".to_string()),
                            }
                        }
                        trust.keys.insert(role.clone(), keys);
                    }
                    _ => return Err("invalid field Keys".to_string()),
                }
            }
        }
        if let Some(entries) = b
            .entries("MinimumSequence")
            .map_err(|_| "invalid field MinimumSequence".to_string())?
        {
            for (channel, item) in entries {
                let raw = item
                    .as_integer()
                    .ok_or_else(|| "invalid field MinimumSequence".to_string())?;
                trust.minimum_sequence.insert(
                    channel.clone(),
                    as_u64(raw).map_err(|_| "invalid field MinimumSequence")?,
                );
            }
        }
        b.finish_name()?;
        Ok(trust)
    }
}

impl Emit for Trust {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Prefix");
        e.string(&self.prefix);
        e.field(false, "Epoch");
        e.uint(self.epoch);
        e.field(false, "Keys");
        e.begin_object(self.keys.is_empty());
        for (i, (role, keys)) in self.keys.iter().enumerate() {
            e.field(i == 0, role);
            e.begin_array(keys.is_empty());
            for (j, key) in keys.iter().enumerate() {
                e.item(j == 0);
                e.string(key);
            }
            e.end_array(keys.is_empty());
        }
        e.end_object(self.keys.is_empty());
        e.field(false, "NotBefore");
        e.int(self.not_before);
        e.field(false, "MaxAgeSeconds");
        e.int(self.max_age_seconds);
        e.field(false, "ClockSkewSeconds");
        e.int(self.clock_skew_seconds);
        e.field(false, "MinimumSequence");
        e.begin_object(self.minimum_sequence.is_empty());
        for (i, (channel, seq)) in self.minimum_sequence.iter().enumerate() {
            e.field(i == 0, channel);
            e.uint(*seq);
        }
        e.end_object(self.minimum_sequence.is_empty());
        e.end_object(false);
    }
}
