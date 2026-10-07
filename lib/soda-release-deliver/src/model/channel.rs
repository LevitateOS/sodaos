use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

use crate::buildx::oci_architecture;
use crate::{is_channel, is_digest_ref, Error};

use super::Trust;

// ---------------------------------------------------------------------------
// Channel / Highwater
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct Channel {
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub format: i64,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub name: String,
    #[serde(deserialize_with = "crate::json_serde::null_u64")]
    pub sequence: u64,
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub issued: i64,
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub expires: i64,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub withdrawn: bool,
    #[serde(serialize_with = "nil_if_empty")]
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub releases: BTreeMap<String, String>,
}

fn nil_if_empty<S: serde::Serializer>(
    value: &BTreeMap<String, String>,
    serializer: S,
) -> Result<S::Ok, S::Error> {
    if value.is_empty() {
        serializer.serialize_none()
    } else {
        value.serialize(serializer)
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct Seen {
    #[serde(deserialize_with = "crate::json_serde::null_u64")]
    pub sequence: u64,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub digest: String,
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub issued: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct Highwater {
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub format: i64,
    #[serde(deserialize_with = "crate::json_serde::null_u64")]
    pub trust_epoch: u64,
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub checked_at: i64,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub channels: BTreeMap<String, Seen>,
    #[serde(deserialize_with = "crate::json_serde::null_u64_map")]
    pub serials: BTreeMap<String, u64>,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub releases: BTreeMap<String, String>,
}

impl Channel {}

impl Seen {}

/// `EmptyState`: fresh high-water mark.
pub fn empty_state() -> Highwater {
    Highwater {
        format: 1,
        ..Highwater::default()
    }
}

fn valid_highwater_maps(s: &Highwater) -> bool {
    s.channels.len() <= 3 && s.serials.len() <= 2 && s.releases.len() == s.serials.len()
}

fn valid_highwater_channels(s: &Highwater) -> bool {
    for (channel, seen) in &s.channels {
        if !is_channel(channel)
            || seen.sequence == 0
            || !is_digest_ref(&seen.digest)
            || seen.issued <= 0
        {
            return false;
        }
    }
    true
}

fn valid_highwater_serials(s: &Highwater) -> bool {
    for (arch, serial) in &s.serials {
        if oci_architecture(arch).is_err() || *serial == 0 {
            return false;
        }
        match s.releases.get(arch) {
            Some(digest) if is_digest_ref(digest) => {}
            _ => return false,
        }
    }
    true
}

impl Highwater {
    pub fn validate(&self) -> Result<(), Error> {
        if self.format != 1 || !valid_highwater_maps(self) || self.checked_at < 0 {
            return Err(Error::refused());
        }
        if !valid_highwater_channels(self) || !valid_highwater_serials(self) {
            return Err(Error::refused());
        }
        Ok(())
    }
}

fn validate_release_reference(t: &Trust, arch: &str, reference: &str) -> Result<(), Error> {
    if oci_architecture(arch).is_err() {
        return Err(Error::refused());
    }
    let (repo, role) = t.reference(reference)?;
    if role != "artifact" || repo != format!("{}-release", t.prefix) {
        return Err(Error::refused());
    }
    Ok(())
}

fn validate_channel_releases(t: &Trust, c: &Channel) -> Result<(), Error> {
    if c.withdrawn {
        if !c.releases.is_empty() {
            return Err(Error::refused());
        }
        return Ok(());
    }
    if c.releases.is_empty() || c.releases.len() > 2 {
        return Err(Error::refused());
    }
    for (arch, reference) in &c.releases {
        validate_release_reference(t, arch, reference)?;
    }
    Ok(())
}

fn validate_channel_identity(
    t: &Trust,
    c: &Channel,
    digest: &str,
    wanted: &str,
) -> Result<(), Error> {
    if t.validate().is_err() || !is_channel(wanted) || !is_digest_ref(digest) {
        return Err(Error::refused());
    }
    if c.format != 1
        || c.name != wanted
        || c.sequence < t.minimum_sequence.get(wanted).copied().unwrap_or(0)
    {
        return Err(Error::refused());
    }
    Ok(())
}

fn validate_channel_timing(
    t: &Trust,
    s: &Highwater,
    c: &Channel,
    now_unix: i64,
) -> Result<(), Error> {
    if now_unix < t.not_before || now_unix < s.checked_at - t.clock_skew_seconds {
        return Err(Error::refused());
    }
    if c.issued < t.not_before || c.issued > now_unix + t.clock_skew_seconds {
        return Err(Error::refused());
    }
    if c.expires <= now_unix || c.expires <= c.issued || c.expires - c.issued > t.max_age_seconds {
        return Err(Error::refused());
    }
    Ok(())
}

fn validate_channel_progression(old: &Seen, c: &Channel, digest: &str) -> Result<(), Error> {
    if c.sequence < old.sequence || c.issued < old.issued {
        return Err(Error::refused());
    }
    if c.sequence == old.sequence && digest != old.digest {
        return Err(Error::refused());
    }
    Ok(())
}

/// `AdmitChannel`: pure channel admission advancing the high-water mark.
pub fn admit_channel(
    t: &Trust,
    s: &Highwater,
    c: &Channel,
    digest: &str,
    wanted: &str,
    now_unix: i64,
) -> Result<Highwater, Error> {
    if s.validate().is_err() || s.trust_epoch > t.epoch {
        return Err(Error::refused());
    }
    validate_channel_identity(t, c, digest, wanted)?;
    validate_channel_timing(t, s, c, now_unix)?;
    validate_channel_releases(t, c)?;
    let old = s.channels.get(wanted).cloned().unwrap_or_default();
    validate_channel_progression(&old, c, digest)?;
    let mut next = s.clone();
    next.trust_epoch = t.epoch;
    next.checked_at = s.checked_at.max(now_unix);
    next.channels.insert(
        wanted.to_string(),
        Seen {
            sequence: c.sequence,
            digest: digest.to_string(),
            issued: c.issued,
        },
    );
    Ok(next)
}
