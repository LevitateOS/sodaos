use super::{go_arch, Executor};
use std::collections::HashMap;

use crate::domain;
use crate::json::{self, Value};
use std::time::Instant;

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
        let v = json::decode_tolerant(&raw)
            .map_err(|_| "invalid installed image inspection".to_string())?;
        let id = json::tolerant_get(&v, "Id")
            .and_then(Value::as_str)
            .unwrap_or("")
            .to_string();
        let architecture = json::tolerant_get(&v, "Architecture")
            .and_then(Value::as_str)
            .unwrap_or("")
            .to_string();
        let os = json::tolerant_get(&v, "Os")
            .and_then(Value::as_str)
            .unwrap_or("")
            .to_string();
        // encoding/json would fail the whole unmarshal on mistyped fields.
        for key in ["Id", "Architecture", "Os"] {
            if let Some(got) = json::tolerant_get(&v, key) {
                if got.as_str().is_none() {
                    return Err("invalid installed image inspection".to_string());
                }
            }
        }
        let labels = match json::tolerant_get(&v, "Labels") {
            None | Some(Value::Null) => HashMap::new(),
            Some(Value::Object(fields)) => {
                let mut map = HashMap::new();
                for (k, val) in fields {
                    match val {
                        Value::Str(s) => {
                            map.insert(k.clone(), s.clone());
                        }
                        Value::Null => {}
                        _ => return Err("invalid installed image inspection".to_string()),
                    }
                }
                map
            }
            Some(_) => return Err("invalid installed image inspection".to_string()),
        };
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
