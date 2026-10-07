//! Admitted live-input records and validation: TailnetInputs,
//! ResolvedCoreOS and LiveInputs emit/decode, exact write/read and
//! admitted-input validation with the round-trip/refusal case.

use crate::coreos::{https_url, CoreOSImage};
use crate::files::{is_digest, write_new};
use crate::json_input::read_json;
use crate::Error;
use serde::de::{self, MapAccess, Visitor};
use serde::ser::SerializeStruct;
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use std::collections::{BTreeMap, HashMap};
use std::path::Path;

/// Floating Tailnet toolchain for one attempt.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct TailnetInputs {
    #[serde(rename = "Version")]
    pub version: String,
    #[serde(rename = "SHA256")]
    pub sha256: String,
    #[serde(rename = "Base")]
    pub base: String,
}

impl<'de> Deserialize<'de> for TailnetInputs {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct TailnetVisitor;
        impl<'de> Visitor<'de> for TailnetVisitor {
            type Value = TailnetInputs;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("Tailnet input object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let (mut version, mut sha256, mut base) = (None, None, None);
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("Version") {
                        version = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("SHA256") {
                        sha256 = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("Base") {
                        base = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &["Version", "SHA256", "Base"],
                        ));
                    }
                }
                let decode =
                    |raw: Option<Box<serde_json::value::RawValue>>| -> Result<String, A::Error> {
                        match raw {
                            None => Ok(String::new()),
                            Some(raw) => serde_json::from_str::<Option<String>>(raw.get())
                                .map(Option::unwrap_or_default)
                                .map_err(de::Error::custom),
                        }
                    };
                Ok(TailnetInputs {
                    version: decode(version)?,
                    sha256: decode(sha256)?,
                    base: decode(base)?,
                })
            }
        }
        deserializer.deserialize_map(TailnetVisitor)
    }
}

/// One stable build as found right now.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ResolvedCoreOS {
    pub release: String,
    pub metadata_url: String,
    pub container: HashMap<String, String>,
    pub iso: HashMap<String, CoreOSImage>,
    pub qemu: HashMap<String, CoreOSImage>,
}

impl Serialize for ResolvedCoreOS {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        let container: BTreeMap<_, _> = self
            .container
            .iter()
            .map(|(key, value)| (key, value))
            .collect();
        let iso: BTreeMap<_, _> = self.iso.iter().map(|(key, value)| (key, value)).collect();
        let qemu: BTreeMap<_, _> = self.qemu.iter().map(|(key, value)| (key, value)).collect();
        let mut state = serializer.serialize_struct("ResolvedCoreOS", 5)?;
        state.serialize_field("Release", &self.release)?;
        state.serialize_field("MetadataURL", &self.metadata_url)?;
        state.serialize_field("Container", &container)?;
        state.serialize_field("ISO", &iso)?;
        state.serialize_field("QEMU", &qemu)?;
        state.end()
    }
}

impl<'de> Deserialize<'de> for ResolvedCoreOS {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct ResolvedVisitor;
        impl<'de> Visitor<'de> for ResolvedVisitor {
            type Value = ResolvedCoreOS;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("resolved CoreOS object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let (mut release, mut metadata_url, mut container, mut iso, mut qemu) =
                    (None, None, None, None, None);
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("Release") {
                        release = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("MetadataURL") {
                        metadata_url = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("Container") {
                        container = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("ISO") {
                        iso = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("QEMU") {
                        qemu = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &["Release", "MetadataURL", "Container", "ISO", "QEMU"],
                        ));
                    }
                }
                fn decode<T: serde::de::DeserializeOwned + Default, E: de::Error>(
                    raw: Option<Box<serde_json::value::RawValue>>,
                ) -> Result<T, E> {
                    match raw {
                        None => Ok(T::default()),
                        Some(raw) => serde_json::from_str::<Option<T>>(raw.get())
                            .map(Option::unwrap_or_default)
                            .map_err(E::custom),
                    }
                }
                Ok(ResolvedCoreOS {
                    release: decode(release)?,
                    metadata_url: decode(metadata_url)?,
                    container: decode(container)?,
                    iso: decode(iso)?,
                    qemu: decode(qemu)?,
                })
            }
        }
        deserializer.deserialize_map(ResolvedVisitor)
    }
}

/// Controller-resolved live inputs for one isolated worker attempt.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct LiveInputs {
    #[serde(rename = "CoreOS")]
    pub coreos: ResolvedCoreOS,
    #[serde(rename = "Tailnet")]
    pub tailnet: TailnetInputs,
}

impl<'de> Deserialize<'de> for LiveInputs {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct LiveVisitor;
        impl<'de> Visitor<'de> for LiveVisitor {
            type Value = LiveInputs;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("live inputs object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let (mut coreos, mut tailnet) = (None, None);
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("CoreOS") {
                        coreos = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("Tailnet") {
                        tailnet = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        return Err(de::Error::unknown_field(&key, &["CoreOS", "Tailnet"]));
                    }
                }
                Ok(LiveInputs {
                    coreos: match coreos {
                        None => ResolvedCoreOS::default(),
                        Some(raw) => serde_json::from_str::<Option<ResolvedCoreOS>>(raw.get())
                            .map(Option::unwrap_or_default)
                            .map_err(de::Error::custom)?,
                    },
                    tailnet: match tailnet {
                        None => TailnetInputs::default(),
                        Some(raw) => serde_json::from_str::<Option<TailnetInputs>>(raw.get())
                            .map(Option::unwrap_or_default)
                            .map_err(de::Error::custom)?,
                    },
                })
            }
        }
        deserializer.deserialize_map(LiveVisitor)
    }
}

/// Records one attempt's live inputs. Validation mirrors the live path.
pub fn write_live_inputs(path: &Path, inputs: &LiveInputs) -> Result<(), Error> {
    valid_live_inputs(inputs)?;
    let body = serde_json::to_string_pretty(inputs)
        .expect("serializing validated live inputs cannot fail")
        + "\n";
    write_new(path, body.as_bytes(), 0o644)
}

/// Admits controller-resolved inputs for the isolated worker.
pub fn read_live_inputs(path: &Path) -> Result<LiveInputs, Error> {
    let inputs: LiveInputs = read_json(path)?;
    valid_live_inputs(&inputs)?;
    Ok(inputs)
}

/// Live resolution shape rules over admitted inputs.
pub fn valid_live_inputs(inputs: &LiveInputs) -> Result<(), Error> {
    valid_resolved_coreos(&inputs.coreos)?;
    valid_tailnet_inputs(&inputs.tailnet)
}

pub fn valid_tailnet_inputs(tailnet: &TailnetInputs) -> Result<(), Error> {
    let reader = soda_build_tools::reader::stream::TailnetInputs {
        version: tailnet.version.clone(),
        sha256: tailnet.sha256.clone(),
        base: tailnet.base.clone(),
    };
    soda_build_tools::reader::stream::valid_tailnet_inputs(&reader)?;
    Ok(())
}

pub fn valid_resolved_coreos(resolved: &ResolvedCoreOS) -> Result<(), Error> {
    let iso = match resolved.iso.get("x86_64") {
        Some(img) => img,
        None => return Err(Error::msg("stable stream x86_64 live ISO is malformed")),
    };
    let qemu = match resolved.qemu.get("x86_64") {
        Some(img) => img,
        None => return Err(Error::msg("stable stream x86_64 qemu image is malformed")),
    };
    let reader_iso = soda_build_tools::reader::stream::CoreOSImage {
        url: iso.url.clone(),
        signature_url: iso.signature_url.clone(),
        sha256: iso.sha256.clone(),
        uncompressed_sha256: iso.uncompressed_sha256.clone(),
    };
    let reader_qemu = soda_build_tools::reader::stream::CoreOSImage {
        url: qemu.url.clone(),
        signature_url: qemu.signature_url.clone(),
        sha256: qemu.sha256.clone(),
        uncompressed_sha256: qemu.uncompressed_sha256.clone(),
    };
    soda_build_tools::reader::stream::valid_stream_images(
        &resolved.release,
        &reader_iso,
        &reader_qemu,
    )?;
    // The shared reader takes single triples; container + metadata rules
    // apply here over the admitted maps.
    if !https_url(&resolved.metadata_url)
        || !resolved
            .metadata_url
            .ends_with(&format!("/builds/{}/release.json", resolved.release))
    {
        return Err(Error::msg("resolved CoreOS metadata URL is malformed"));
    }
    if resolved.container.len() != 1 {
        return Err(Error::msg("x86_64 base digest required"));
    }
    let entry = resolved
        .container
        .get("x86_64")
        .map(String::as_str)
        .unwrap_or("");
    match entry.split_once("/fedora/fedora-coreos@sha256:") {
        Some((host, digest)) if !host.is_empty() && !host.contains('/') && is_digest(digest) => {
            Ok(())
        }
        _ => Err(Error::msg("digest-pinned CoreOS base required")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_support::fixture_live_inputs;

    #[test]
    fn live_inputs_nested_aliases_keep_source_order_before_typed_decode() {
        let input = r#"{"CoreOS":{"ISO":{"x86_64":{"url":false,"URL":"ok"}}}}"#;
        let parsed: LiveInputs = serde_json::from_str(input).unwrap();
        assert_eq!(parsed.coreos.iso["x86_64"].url, "ok");

        let input = r#"{"CoreOS":{"ISO":{"x86_64":{"URL":"ok","url":false}}}}"#;
        assert!(serde_json::from_str::<LiveInputs>(input).is_err());

        let input = r#"{"CoreOS":{"ISO":{"x86_64":{"url":false,"URL":null}}}}"#;
        let parsed: LiveInputs = serde_json::from_str(input).unwrap();
        assert_eq!(parsed.coreos.iso["x86_64"].url, "");

        let input = r#"{"CoreOS":{"ISO":{"x86_64":{"URL":null,"url":"ok"}}}}"#;
        let parsed: LiveInputs = serde_json::from_str(input).unwrap();
        assert_eq!(parsed.coreos.iso["x86_64"].url, "ok");
    }

    #[test]
    fn oracle_live_inputs_round_trip() {
        // Oracle: WriteLiveInputs/ReadLiveInputs keep the resolved release.
        let dir = std::env::temp_dir().join(format!(
            "soda-live-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("live-inputs.json");
        let inputs = fixture_live_inputs();
        write_live_inputs(&path, &inputs).unwrap();
        let back = read_live_inputs(&path).unwrap();
        assert_eq!(back.coreos.release, "44.20260901.1.0");
        assert_eq!(back.tailnet.version, "1.98.2");
        // Tampered digests and versions are refused on read.
        let mut bad = inputs.clone();
        bad.tailnet.version = "yesterday".to_string();
        assert!(write_live_inputs(&dir.join("bad.json"), &bad).is_err());
        let mut bad = inputs.clone();
        bad.coreos.release = "tomorrow".to_string();
        assert!(write_live_inputs(&dir.join("bad2.json"), &bad).is_err());
    }
}
