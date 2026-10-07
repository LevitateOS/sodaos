use serde::de::{DeserializeSeed, IgnoredAny, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};
use std::collections::BTreeMap;
use std::fmt;

use crate::buildx::is_digest;
use crate::payload::{Payload, NAMES};
use crate::{is_channel, is_digest_ref, Error};

use super::{Candidate, Channel, Highwater, Trust};

// ---------------------------------------------------------------------------
// Media binding (lenient decode: extra fields ignored)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct MediaFile {
    pub path: String,
    #[serde(rename = "SHA256")]
    pub sha256: String,
    pub bytes: i64,
}

struct I64Seed;
impl<'de> serde::de::DeserializeSeed<'de> for I64Seed {
    type Value = i64;
    fn deserialize<D: Deserializer<'de>>(self, d: D) -> Result<i64, D::Error> {
        crate::json_serde::null_i64(d)
    }
}

#[derive(Default)]
struct RawI64(i64);
impl<'de> Deserialize<'de> for RawI64 {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        I64Seed.deserialize(d).map(RawI64)
    }
}

fn decode_slot<T: for<'de> Deserialize<'de> + Default, E: serde::de::Error>(
    raw: Option<Box<serde_json::value::RawValue>>,
) -> Result<T, E> {
    let Some(raw) = raw else {
        return Ok(T::default());
    };
    serde_json::from_str::<Option<T>>(raw.get())
        .map_err(E::custom)
        .map(Option::unwrap_or_default)
}

impl<'de> Deserialize<'de> for MediaFile {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = MediaFile;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a media file object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<MediaFile, M::Error> {
                let (mut path, mut sha256, mut bytes) = (None, None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "Path" => path = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "SHA256" => {
                            sha256 = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "Bytes" => {
                            bytes = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                let path = decode_slot(path)?;
                let sha256 = decode_slot(sha256)?;
                let bytes = decode_slot::<RawI64, M::Error>(bytes)?.0;
                Ok(MediaFile {
                    path,
                    sha256,
                    bytes,
                })
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct MediaBinding {
    pub revision: String,
    pub architecture: String,
    pub host_manifest: String,
    #[serde(rename = "PayloadSHA256")]
    pub payload_sha256: String,
    #[serde(rename = "RootfsURL")]
    pub rootfs_url: String,
    pub iso: MediaFile,
    pub rootfs: MediaFile,
}

impl<'de> Deserialize<'de> for MediaBinding {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = MediaBinding;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a media binding object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<MediaBinding, M::Error> {
                let (
                    mut revision,
                    mut architecture,
                    mut host_manifest,
                    mut payload_sha256,
                    mut rootfs_url,
                    mut iso,
                    mut rootfs,
                ) = (None, None, None, None, None, None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "Revision" => {
                            revision = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "Architecture" => {
                            architecture = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "HostManifest" => {
                            host_manifest =
                                Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "PayloadSHA256" => {
                            payload_sha256 =
                                Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "RootfsURL" => {
                            rootfs_url = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "ISO" => iso = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "Rootfs" => {
                            rootfs = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(MediaBinding {
                    revision: decode_slot(revision)?,
                    architecture: decode_slot(architecture)?,
                    host_manifest: decode_slot(host_manifest)?,
                    payload_sha256: decode_slot(payload_sha256)?,
                    rootfs_url: decode_slot(rootfs_url)?,
                    iso: decode_slot(iso)?,
                    rootfs: decode_slot(rootfs)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl MediaBinding {
    pub(crate) fn decode_lenient(
        data: &[u8],
    ) -> Result<MediaBinding, crate::json_serde::DecodeError> {
        serde_json::from_slice(data).map_err(|_| crate::json_serde::DecodeError)
    }
}

#[cfg(test)]
mod media_json_tests {
    use super::MediaBinding;

    #[test]
    fn media_binding_defers_validation_until_the_last_exact_raw_value() {
        let value = MediaBinding::decode_lenient(br#"{"Revision":7,"Revision":"final","ISO":{"Path":false,"Path":"disk.iso","Bytes":1.5,"Bytes":-0,"Unknown":{"nested":[1]}},"Rootfs":null,"ignored":true}"#).unwrap();
        assert_eq!(value.revision, "final");
        assert_eq!(value.iso.path, "disk.iso");
        assert_eq!(value.iso.bytes, 0);
        assert_eq!(value.rootfs, Default::default());
        assert!(MediaBinding::decode_lenient(br#"{"ISO":{"Bytes":-0.0}}"#).is_err());
        assert!(MediaBinding::decode_lenient(br#"{"ISO":{"Bytes":-0,"Bytes":1e0}}"#).is_err());
        let reset = MediaBinding::decode_lenient(br#"{"Revision":"old","Revision":null}"#).unwrap();
        assert!(reset.revision.is_empty());
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

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct Release {
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub format: i64,
    #[serde(deserialize_with = "crate::json_serde::null_u64")]
    pub serial: u64,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub class: String,
    #[serde(serialize_with = "serialize_base64")]
    #[serde(deserialize_with = "deserialize_base64")]
    pub payload: Vec<u8>,
    #[serde(serialize_with = "serialize_base64")]
    #[serde(deserialize_with = "deserialize_base64")]
    pub candidate: Vec<u8>,
    #[serde(serialize_with = "serialize_base64")]
    #[serde(deserialize_with = "deserialize_base64")]
    pub media: Vec<u8>,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub provenance: BTreeMap<String, String>,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub qualification: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub evidence: BTreeMap<String, String>,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub notes: String,
}

fn serialize_base64<S: serde::Serializer>(
    bytes: &Vec<u8>,
    serializer: S,
) -> Result<S::Ok, S::Error> {
    use base64::Engine;
    serializer.serialize_str(&base64::engine::general_purpose::STANDARD.encode(bytes))
}

fn deserialize_base64<'de, D: Deserializer<'de>>(deserializer: D) -> Result<Vec<u8>, D::Error> {
    use base64::Engine;
    use serde::de::Error as _;
    let value = Option::<String>::deserialize(deserializer)?.unwrap_or_default();
    let decoded = base64::engine::general_purpose::STANDARD
        .decode(&value)
        .map_err(D::Error::custom)?;
    if base64::engine::general_purpose::STANDARD.encode(&decoded) != value {
        return Err(D::Error::custom("non-canonical base64"));
    }
    Ok(decoded)
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
    let p = crate::json_serde::strict(&r.payload)?;
    let c = crate::json_serde::strict(&r.candidate)?;
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
