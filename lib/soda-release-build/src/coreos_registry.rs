//! CoreOS container-registry resolution: image-index request,
//! bounded body decode and matching x86_64 digest selection.

use crate::coreos::https_url;
use crate::files::{is_digest, oci_architecture};
use crate::http::{get_follow, HttpTransport};
use crate::Error;
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::collections::HashMap;
use std::io::Read;
use std::time::Duration;

const COREOS_CONTAINER_REPO: &str = "fedora/fedora-coreos";
const COREOS_CONTAINER_TAG: &str = "stable";

pub(crate) fn resolve_registry_digests_with<T: HttpTransport>(
    transport: &T,
    registry: &str,
) -> Result<HashMap<String, String>, Error> {
    if !https_url(registry) {
        return Err(Error::msg("container registry URL must be HTTPS"));
    }
    let host = registry
        .split_once("://")
        .map(|(_, rest)| rest.split('/').next().unwrap_or(""))
        .unwrap_or("");
    if host.is_empty() || host.contains('/') {
        return Err(Error::msg("container registry URL is malformed"));
    }
    let endpoint =
        format!("{registry}/v2/{COREOS_CONTAINER_REPO}/manifests/{COREOS_CONTAINER_TAG}");
    let response = get_follow(
        transport,
        &endpoint,
        Some("application/vnd.oci.image.index.v1+json"),
        Duration::from_secs(60),
        "unsafe metadata redirect",
    )
    .map_err(|_| Error::msg("container registry fetch failed"))?;
    if response.status == 401 {
        return Err(Error::msg(
            "container registry refused anonymous manifest access",
        ));
    }
    if response.status != 200 {
        return Err(Error::msg(format!(
            "container registry HTTP failure: {}",
            response.status_line()
        )));
    }
    let mut data = Vec::new();
    response
        .body
        .take((1 << 20) + 1)
        .read_to_end(&mut data)
        .map_err(|e| Error::msg(e.to_string()))?;
    if data.len() > 1 << 20 {
        return Err(Error::msg("container index exceeds size limit"));
    }
    let text = String::from_utf8_lossy(&data);
    let index: ManifestIndex =
        serde_json::from_str(&text).map_err(|_| Error::msg("invalid container index"))?;
    let oci_arch = oci_architecture("x86_64")?;
    let mut found = String::new();
    for manifest in &index.manifests {
        if manifest.platform.architecture != oci_arch {
            continue;
        }
        let digest = &manifest.digest;
        match digest.strip_prefix("sha256:") {
            Some(hex) if is_digest(hex) => {}
            _ => return Err(Error::msg("container index x86_64 digest is malformed")),
        }
        found = digest.clone();
    }
    if found.is_empty() {
        return Err(Error::msg("container index lacks architecture x86_64"));
    }
    let mut digests = HashMap::new();
    digests.insert(
        "x86_64".to_string(),
        format!("{host}/{COREOS_CONTAINER_REPO}@{found}"),
    );
    Ok(digests)
}

struct ManifestIndex {
    manifests: Vec<ManifestRecord>,
}
struct ManifestRecord {
    platform: ManifestPlatform,
    digest: String,
}
#[derive(Default)]
struct ManifestPlatform {
    architecture: String,
}

impl<'de> Deserialize<'de> for ManifestIndex {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ManifestIndex;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("container index object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let mut manifests = None;
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("manifests") {
                        manifests = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                let manifests = match manifests {
                    None => Vec::new(),
                    Some(raw) => serde_json::from_str::<Option<Vec<ManifestRecord>>>(raw.get())
                        .map_err(de::Error::custom)?
                        .unwrap_or_default(),
                };
                Ok(ManifestIndex { manifests })
            }
        }
        deserializer.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for ManifestRecord {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ManifestRecord;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("container manifest object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let (mut platform, mut digest) = (None, None);
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("platform") {
                        platform = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("digest") {
                        digest = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                let platform = match platform {
                    None => ManifestPlatform::default(),
                    Some(raw) => serde_json::from_str::<Option<ManifestPlatform>>(raw.get())
                        .map_err(de::Error::custom)?
                        .unwrap_or_default(),
                };
                let digest = match digest {
                    None => String::new(),
                    Some(raw) => serde_json::from_str::<Option<String>>(raw.get())
                        .map_err(de::Error::custom)?
                        .unwrap_or_default(),
                };
                Ok(ManifestRecord { platform, digest })
            }
        }
        deserializer.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for ManifestPlatform {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ManifestPlatform;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("platform object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let mut architecture = None;
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("architecture") {
                        architecture = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                let architecture = match architecture {
                    None => String::new(),
                    Some(raw) => serde_json::from_str::<Option<String>>(raw.get())
                        .map_err(de::Error::custom)?
                        .unwrap_or_default(),
                };
                Ok(ManifestPlatform { architecture })
            }
        }
        deserializer.deserialize_map(V)
    }
}
