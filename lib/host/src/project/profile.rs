use super::{go_arch, Executor};
use std::collections::HashMap;

use crate::domain;
use crate::json;
use serde::de::{MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;
use std::time::Instant;

#[derive(Default)]
struct ImageInspection {
    id: String,
    architecture: String,
    os: String,
    labels: HashMap<String, Option<String>>,
}

impl<'de> Deserialize<'de> for ImageInspection {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ImageVisitor;
        impl<'de> Visitor<'de> for ImageVisitor {
            type Value = ImageInspection;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an image inspection object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = ImageInspection::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("Id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.id = v;
                        }
                    } else if key.eq_ignore_ascii_case("Architecture") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.architecture = v;
                        }
                    } else if key.eq_ignore_ascii_case("Os") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.os = v;
                        }
                    } else if key.eq_ignore_ascii_case("Labels") {
                        if let Some(v) =
                            map.next_value::<Option<HashMap<String, Option<String>>>>()?
                        {
                            out.labels = v;
                        }
                    } else {
                        map.next_value::<serde::de::IgnoredAny>()?;
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(ImageVisitor)
    }
}

pub const PROFILE_INSPECT_FORMAT: &str =
    "{\"Id\":{{json .ID}},\"Architecture\":{{json .Architecture}},\"Os\":{{json .Os}},\"Labels\":{{json .Labels}}}";

impl<E: Executor> super::Runtime<E> {
    /// `ResolveProfile`: inspect only the configured installed image.
    pub fn resolve_profile(&self, deadline: Instant) -> Result<domain::Profile, String> {
        let raw = self.podman(
            &[],
            &[
                "image",
                "inspect",
                "--format",
                PROFILE_INSPECT_FORMAT,
                &self.config.image,
            ],
            deadline,
        )?;
        if raw.len() > 65536 {
            return Err("invalid installed image inspection".to_string());
        }
        let inspect: ImageInspection = json::decode_tolerant_as(&raw)
            .map_err(|_| "invalid installed image inspection".to_string())?;
        let id = inspect.id;
        let architecture = inspect.architecture;
        let os = inspect.os;
        let labels: HashMap<String, String> = inspect
            .labels
            .into_iter()
            .filter_map(|(k, value)| value.map(|v| (k, v)))
            .collect();
        let mut image_id = id;
        if !image_id.starts_with("sha256:") {
            image_id = format!("sha256:{image_id}");
        }
        let get = |k: &str| labels.get(k).cloned().unwrap_or_default();
        let profile = domain::Profile {
            id: get("org.soda.profile"),
            distribution: get("org.soda.distribution"),
            version: get("org.soda.distribution.version"),
            interface: get("org.soda.interface"),
            architecture: architecture.clone(),
            image: image_id,
            revision: get("org.opencontainers.image.revision"),
        };
        if os != "linux" || architecture != go_arch() {
            return Err("project image is not native Linux".to_string());
        }
        profile.validate()?;
        Ok(profile)
    }
}

pub fn apply_creation_profile(
    env: &mut domain::Environment,
    labels: &HashMap<String, String>,
    image: &str,
) -> Result<(), String> {
    let raw = labels
        .get("org.soda.creation-profile")
        .map(String::as_str)
        .unwrap_or("");
    if raw.is_empty() {
        return Ok(());
    }
    let profile =
        domain::decode_profile(raw).map_err(|_| "native creation profile mismatch".to_string())?;
    let mut expected = image.to_string();
    if !expected.starts_with("sha256:") {
        expected = format!("sha256:{expected}");
    }
    if profile.id
        != labels
            .get("org.soda.profile")
            .map(String::as_str)
            .unwrap_or("")
        || profile.image != expected
    {
        return Err("native creation profile mismatch".to_string());
    }
    env.profile = Some(profile);
    Ok(())
}
