use serde::de::{DeserializeSeed, IgnoredAny, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};
use std::fmt;

use crate::buildx::is_digest;
use crate::payload::Payload;

use super::Candidate;

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
    pub deployment_commit: String,
    #[serde(rename = "RootfsURL")]
    pub rootfs_url: String,
    #[serde(rename = "ISO")]
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
                    mut deployment_commit,
                    mut rootfs_url,
                    mut iso,
                    mut rootfs,
                ) = (None, None, None, None, None, None, None, None);
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
                        "DeploymentCommit" => {
                            deployment_commit =
                                Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
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
                    deployment_commit: decode_slot(deployment_commit)?,
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

    #[test]
    fn media_binding_rejects_rootfs_above_kernel_margin() {
        let payload = super::Payload::default();
        let candidate = super::Candidate::default();
        let mut media = MediaBinding::default();
        media.iso = super::MediaFile {
            path: "installer.iso".into(),
            sha256: "a".repeat(64),
            bytes: 1,
        };
        media.rootfs = super::MediaFile {
            path: "rootfs.img".into(),
            sha256: "b".repeat(64),
            bytes: soda_build_tools::MAX_ROOTFS_BYTES as i64,
        };
        media.rootfs_url = "https://example.test/rootfs.img".into();
        assert!(super::valid_media_binding(&media, &payload, &candidate));
        media.rootfs.bytes += 1;
        assert!(!super::valid_media_binding(&media, &payload, &candidate));
    }
    #[test]
    fn deployment_binding_is_distinct_from_assembler_import() {
        let parsed = MediaBinding::decode_lenient(
            br#"{"DeploymentCommit":"actual-rootfs","AssemblerImportCommit":"assembler-import"}"#,
        )
        .unwrap();
        assert_eq!(parsed.deployment_commit, "actual-rootfs");
        let import_only =
            MediaBinding::decode_lenient(br#"{"AssemblerImportCommit":"assembler-import"}"#)
                .unwrap();
        assert!(import_only.deployment_commit.is_empty());
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
    valid_media_file(&m.iso)
        && valid_media_file(&m.rootfs)
        && soda_build_tools::valid_rootfs_size(m.rootfs.bytes as u64)
        && valid_media_url(&m.rootfs_url)
}
