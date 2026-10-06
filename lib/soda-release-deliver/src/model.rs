//! `model.go`: trust, candidate, release, channel, high-water mark, permit.

use std::collections::BTreeMap;

use soda_json::JsonValue;

use crate::buildx::oci_architecture;
use crate::jsonx::{as_u64, Binder, Emit, Emitter};
use crate::payload::{
    decode_opt_bool, decode_opt_i64, decode_opt_string, decode_opt_u64, decode_string_map,
};
use crate::{is_channel, is_digest_ref, Error};

mod trust;
pub use trust::Trust;

mod candidate;
pub(crate) use candidate::path_clean;
pub use candidate::{valid_candidate_content, Candidate};

mod release;
pub use release::{admit_release, MediaBinding, MediaFile, Release};
pub(crate) use release::{valid_media_binding, valid_media_file};

// ---------------------------------------------------------------------------
// Channel / Highwater
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Channel {
    pub format: i64,
    pub name: String,
    pub sequence: u64,
    pub issued: i64,
    pub expires: i64,
    pub withdrawn: bool,
    pub releases: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Seen {
    pub sequence: u64,
    pub digest: String,
    pub issued: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Highwater {
    pub format: i64,
    pub trust_epoch: u64,
    pub checked_at: i64,
    pub channels: BTreeMap<String, Seen>,
    pub serials: BTreeMap<String, u64>,
    pub releases: BTreeMap<String, String>,
}

impl Channel {
    pub fn decode(value: &JsonValue) -> Result<Channel, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid channel".to_string())?;
        let channel = Channel {
            format: decode_opt_i64(&mut b, "Format")?,
            name: decode_opt_string(&mut b, "Name")?,
            sequence: decode_opt_u64(&mut b, "Sequence")?,
            issued: decode_opt_i64(&mut b, "Issued")?,
            expires: decode_opt_i64(&mut b, "Expires")?,
            withdrawn: decode_opt_bool(&mut b, "Withdrawn")?,
            releases: decode_string_map(&mut b, "Releases")?,
        };
        b.finish_name()?;
        Ok(channel)
    }
}

impl Emit for Channel {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Name");
        e.string(&self.name);
        e.field(false, "Sequence");
        e.uint(self.sequence);
        e.field(false, "Issued");
        e.int(self.issued);
        e.field(false, "Expires");
        e.int(self.expires);
        e.field(false, "Withdrawn");
        e.boolean(self.withdrawn);
        e.field(false, "Releases");
        // Withdrawn channels carry a nil map in Go; an empty map only ever
        // arises from withdrawal, so empty marshals as `null` like the owner.
        if self.releases.is_empty() {
            e.null();
        } else {
            e.begin_object(false);
            for (i, (arch, reference)) in self.releases.iter().enumerate() {
                e.field(i == 0, arch);
                e.string(reference);
            }
            e.end_object(false);
        }
        e.end_object(false);
    }
}

impl Seen {
    pub fn decode(value: &JsonValue) -> Result<Seen, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid seen".to_string())?;
        let seen = Seen {
            sequence: decode_opt_u64(&mut b, "Sequence")?,
            digest: decode_opt_string(&mut b, "Digest")?,
            issued: decode_opt_i64(&mut b, "Issued")?,
        };
        b.finish_name()?;
        Ok(seen)
    }
}

impl Emit for Seen {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Sequence");
        e.uint(self.sequence);
        e.field(false, "Digest");
        e.string(&self.digest);
        e.field(false, "Issued");
        e.int(self.issued);
        e.end_object(false);
    }
}

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

    pub fn decode(value: &JsonValue) -> Result<Highwater, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid highwater".to_string())?;
        let mut state = Highwater {
            format: decode_opt_i64(&mut b, "Format")?,
            trust_epoch: decode_opt_u64(&mut b, "TrustEpoch")?,
            checked_at: decode_opt_i64(&mut b, "CheckedAt")?,
            channels: BTreeMap::new(),
            serials: BTreeMap::new(),
            releases: BTreeMap::new(),
        };
        if let Some(entries) = b
            .entries("Channels")
            .map_err(|_| "invalid field Channels".to_string())?
        {
            for (channel, item) in entries {
                state.channels.insert(channel.clone(), Seen::decode(item)?);
            }
        }
        if let Some(entries) = b
            .entries("Serials")
            .map_err(|_| "invalid field Serials".to_string())?
        {
            for (arch, item) in entries {
                let raw = item
                    .as_integer()
                    .ok_or_else(|| "invalid field Serials".to_string())?;
                state.serials.insert(
                    arch.clone(),
                    as_u64(raw).map_err(|_| "invalid field Serials".to_string())?,
                );
            }
        }
        state.releases = decode_string_map(&mut b, "Releases")?;
        b.finish_name()?;
        Ok(state)
    }
}

impl Emit for Highwater {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "TrustEpoch");
        e.uint(self.trust_epoch);
        e.field(false, "CheckedAt");
        e.int(self.checked_at);
        e.field(false, "Channels");
        e.begin_object(self.channels.is_empty());
        for (i, (channel, seen)) in self.channels.iter().enumerate() {
            e.field(i == 0, channel);
            seen.emit(e);
        }
        e.end_object(self.channels.is_empty());
        e.field(false, "Serials");
        e.begin_object(self.serials.is_empty());
        for (i, (arch, serial)) in self.serials.iter().enumerate() {
            e.field(i == 0, arch);
            e.uint(*serial);
        }
        e.end_object(self.serials.is_empty());
        e.field(false, "Releases");
        e.begin_object(self.releases.is_empty());
        for (i, (arch, digest)) in self.releases.iter().enumerate() {
            e.field(i == 0, arch);
            e.string(digest);
        }
        e.end_object(self.releases.is_empty());
        e.end_object(false);
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

// ---------------------------------------------------------------------------
// Permit
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
    pub fn validate(&self, t: &Trust, now_unix: i64) -> Result<(), Error> {
        if t.role(&self.repository).is_err()
            || self.format != 1
            || !is_digest_ref(&self.digest)
            || self.expires <= now_unix
            || self.expires - now_unix > 86400
        {
            return Err(Error::refused());
        }
        Ok(())
    }

    pub fn decode(value: &JsonValue) -> Result<Permit, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid permit".to_string())?;
        let permit = Permit {
            format: decode_opt_i64(&mut b, "Format")?,
            repository: decode_opt_string(&mut b, "Repository")?,
            digest: decode_opt_string(&mut b, "Digest")?,
            previous: decode_opt_string(&mut b, "Previous")?,
            expires: decode_opt_i64(&mut b, "Expires")?,
        };
        b.finish_name()?;
        Ok(permit)
    }
}

impl Emit for Permit {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Repository");
        e.string(&self.repository);
        e.field(false, "Digest");
        e.string(&self.digest);
        e.field(false, "Previous");
        e.string(&self.previous);
        e.field(false, "Expires");
        e.int(self.expires);
        e.end_object(false);
    }
}

#[cfg(test)]
mod tests;
