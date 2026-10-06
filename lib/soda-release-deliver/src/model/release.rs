use std::collections::BTreeMap;

use soda_json::JsonValue;

use crate::buildx::is_digest;
use crate::jsonx::{as_i64, parse_lenient, parse_strict, Binder, Emit, Emitter, Soft};
use crate::payload::{
    decode_opt_i64, decode_opt_string, decode_opt_u64, decode_string_map, Payload, NAMES,
};
use crate::{is_channel, is_digest_ref, Error};

use super::{Candidate, Channel, Highwater, Trust};

// ---------------------------------------------------------------------------
// Media binding (lenient decode: extra fields ignored)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MediaFile {
    pub path: String,
    pub sha256: String,
    pub bytes: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MediaBinding {
    pub revision: String,
    pub architecture: String,
    pub host_manifest: String,
    pub payload_sha256: String,
    pub rootfs_url: String,
    pub iso: MediaFile,
    pub rootfs: MediaFile,
}

fn decode_media_file(value: &JsonValue) -> Result<MediaFile, crate::jsonx::DecodeError> {
    let soft = Soft::new(value)?;
    Ok(MediaFile {
        path: soft.string("Path")?.unwrap_or_default(),
        sha256: soft.string("SHA256")?.unwrap_or_default(),
        bytes: as_i64(soft.integer("Bytes")?.unwrap_or(0))?,
    })
}

impl MediaBinding {
    pub fn decode_lenient(data: &[u8]) -> Result<MediaBinding, crate::jsonx::DecodeError> {
        let value = parse_lenient(data)?;
        let soft = Soft::new(&value)?;
        let iso = match soft.field("ISO") {
            Some(v) => decode_media_file(v)?,
            None => MediaFile::default(),
        };
        let rootfs = match soft.field("Rootfs") {
            Some(v) => decode_media_file(v)?,
            None => MediaFile::default(),
        };
        Ok(MediaBinding {
            revision: soft.string("Revision")?.unwrap_or_default(),
            architecture: soft.string("Architecture")?.unwrap_or_default(),
            host_manifest: soft.string("HostManifest")?.unwrap_or_default(),
            payload_sha256: soft.string("PayloadSHA256")?.unwrap_or_default(),
            rootfs_url: soft.string("RootfsURL")?.unwrap_or_default(),
            iso,
            rootfs,
        })
    }
}

pub(crate) fn valid_media_file(f: &MediaFile) -> bool {
    !f.path.is_empty()
        && !f.path.contains("..")
        && !f.path.bytes().any(|b| matches!(b, b'\n' | b'\r' | 0))
        && is_digest(&f.sha256)
        && f.bytes > 0
}

pub(crate) fn valid_media_url(url: &str) -> bool {
    !url.is_empty()
        && !url.bytes().any(|b| matches!(b, b'\n' | b'\r' | 0 | b' '))
        && (url.starts_with("https://") || url.starts_with("http://"))
}

pub(crate) fn valid_media_binding(m: &MediaBinding, p: &Payload, c: &Candidate) -> bool {
    if m.revision != p.revision
        || m.architecture != p.architecture
        || m.host_manifest != c.host.manifest
        || m.payload_sha256 != c.payload_sha256
    {
        return false;
    }
    valid_media_file(&m.iso) && valid_media_file(&m.rootfs) && valid_media_url(&m.rootfs_url)
}

// ---------------------------------------------------------------------------
// Release
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Release {
    pub format: i64,
    pub serial: u64,
    pub class: String,
    pub payload: Vec<u8>,
    pub candidate: Vec<u8>,
    pub media: Vec<u8>,
    pub provenance: BTreeMap<String, String>,
    pub qualification: String,
    pub evidence: BTreeMap<String, String>,
    pub notes: String,
}

fn valid_release_identity(r: &Release) -> bool {
    r.format == 1
        && r.serial != 0
        && (r.class == "normal" || r.class == "emergency")
        && (r.qualification == "local-only" || r.qualification == "native-install-upgrade-recovery")
}

fn valid_release_notes_and_evidence(r: &Release) -> bool {
    !r.evidence.is_empty()
        && r.evidence.len() <= 64
        && !r.notes.is_empty()
        && r.notes.len() <= 16384
}

fn decode_release_payloads(r: &Release) -> Result<(Payload, Candidate), Error> {
    let payload_value = parse_strict(&r.payload)?;
    let candidate_value = parse_strict(&r.candidate)?;
    let p = Payload::decode(&payload_value).map_err(|_| Error::refused())?;
    let c = Candidate::decode(&candidate_value).map_err(|_| Error::refused())?;
    Ok((p, c))
}

fn decode_release_media(r: &Release, p: &Payload, c: &Candidate) -> Result<(), Error> {
    if r.media.is_empty() || r.media.len() > 1 << 20 {
        return Err(Error::refused());
    }
    let binding = MediaBinding::decode_lenient(&r.media).map_err(|_| Error::refused())?;
    if !valid_media_binding(&binding, p, c) {
        return Err(Error::refused());
    }
    Ok(())
}

fn valid_release_provenance(r: &Release, p: &Payload, c: &Candidate) -> bool {
    if r.provenance.len() != 5 {
        return false;
    }
    for name in [
        "source.tar",
        "forgejo-source.tar",
        "app-inputs.json",
        "packages.txt",
        "presentation.json",
    ] {
        match r.provenance.get(name) {
            Some(hash) if is_digest_ref(hash) => {}
            _ => return false,
        }
    }
    r.provenance.get("packages.txt").map(String::as_str)
        == Some(format!("sha256:{}", p.host_packages_sha256).as_str())
        && r.provenance.get("presentation.json").map(String::as_str)
            == Some(format!("sha256:{}", p.presentation_sha256).as_str())
        && r.provenance.get("forgejo-source.tar").map(String::as_str)
            == Some(format!("sha256:{}", c.forgejo_source_sha256).as_str())
}

fn valid_release_evidence(evidence: &BTreeMap<String, String>) -> bool {
    for (name, hash) in evidence {
        if name.is_empty()
            || name.len() > 128
            || name.bytes().any(|b| matches!(b, b'\n' | b'\r' | 0))
            || !is_digest_ref(hash)
        {
            return false;
        }
    }
    true
}

impl Release {
    pub fn validate(&self, t: &Trust) -> Result<(Payload, Candidate), Error> {
        if !valid_release_identity(self) || !valid_release_notes_and_evidence(self) {
            return Err(Error::refused());
        }
        let (p, c) = decode_release_payloads(self)?;
        if p.repository_prefix != t.prefix || c.validate(&p, &self.payload).is_err() {
            return Err(Error::refused());
        }
        if !valid_release_provenance(self, &p, &c)
            || !valid_release_evidence(&self.evidence)
            || decode_release_media(self, &p, &c).is_err()
        {
            return Err(Error::refused());
        }
        Ok((p, c))
    }

    pub fn references(&self, t: &Trust) -> Result<Vec<String>, Error> {
        let (p, c) = self.validate(t)?;
        let mut refs = vec![c.host_reference.clone()];
        for name in NAMES {
            match p.images.get(name) {
                Some(image) => refs.push(image.reference.clone()),
                None => return Err(Error::refused()),
            }
        }
        Ok(refs)
    }

    pub fn decode(value: &JsonValue) -> Result<Release, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid release".to_string())?;
        let release = Release {
            format: decode_opt_i64(&mut b, "Format")?,
            serial: decode_opt_u64(&mut b, "Serial")?,
            class: decode_opt_string(&mut b, "Class")?,
            payload: b
                .bytes("Payload")
                .map_err(|_| "invalid field Payload".to_string())?
                .unwrap_or_default(),
            candidate: b
                .bytes("Candidate")
                .map_err(|_| "invalid field Candidate".to_string())?
                .unwrap_or_default(),
            media: b
                .bytes("Media")
                .map_err(|_| "invalid field Media".to_string())?
                .unwrap_or_default(),
            provenance: decode_string_map(&mut b, "Provenance")?,
            qualification: decode_opt_string(&mut b, "Qualification")?,
            evidence: decode_string_map(&mut b, "Evidence")?,
            notes: decode_opt_string(&mut b, "Notes")?,
        };
        b.finish_name()?;
        Ok(release)
    }
}

impl Emit for Release {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Serial");
        e.uint(self.serial);
        e.field(false, "Class");
        e.string(&self.class);
        e.field(false, "Payload");
        e.bytes(&self.payload);
        e.field(false, "Candidate");
        e.bytes(&self.candidate);
        e.field(false, "Media");
        e.bytes(&self.media);
        e.field(false, "Provenance");
        e.begin_object(self.provenance.is_empty());
        for (i, (name, hash)) in self.provenance.iter().enumerate() {
            e.field(i == 0, name);
            e.string(hash);
        }
        e.end_object(self.provenance.is_empty());
        e.field(false, "Qualification");
        e.string(&self.qualification);
        e.field(false, "Evidence");
        e.begin_object(self.evidence.is_empty());
        for (i, (name, hash)) in self.evidence.iter().enumerate() {
            e.field(i == 0, name);
            e.string(hash);
        }
        e.end_object(self.evidence.is_empty());
        e.field(false, "Notes");
        e.string(&self.notes);
        e.end_object(false);
    }
}

fn admit_channel_ref(c: &Channel, arch: &str, reference: &str) -> bool {
    is_channel(&c.name)
        && c.format == 1
        && !c.withdrawn
        && c.releases.get(arch).map(String::as_str) == Some(reference)
}

fn admit_release_digest(
    s: &Highwater,
    arch: &str,
    reference: &str,
    r: &Release,
) -> Result<String, Error> {
    let digest = reference.split('@').nth(1).unwrap_or("").to_string();
    let serial = s.serials.get(arch).copied().unwrap_or(0);
    if r.serial < serial
        || (r.serial == serial && s.releases.get(arch).map(String::as_str) != Some(digest.as_str()))
    {
        return Err(Error::refused());
    }
    Ok(digest)
}

/// `AdmitRelease`: pure release admission advancing serials.
pub fn admit_release(
    t: &Trust,
    s: &Highwater,
    c: &Channel,
    arch: &str,
    reference: &str,
    r: &Release,
) -> Result<Highwater, Error> {
    let (p, _) = r.validate(t)?;
    if s.validate().is_err() || !admit_channel_ref(c, arch, reference) {
        return Err(Error::refused());
    }
    if p.architecture != arch {
        return Err(Error::refused());
    }
    if c.name != "candidate" && r.qualification != "native-install-upgrade-recovery" {
        return Err(Error::refused());
    }
    let digest = admit_release_digest(s, arch, reference, r)?;
    let mut next = s.clone();
    next.serials.insert(arch.to_string(), r.serial);
    next.releases.insert(arch.to_string(), digest);
    Ok(next)
}
