//! Host daemon config: Rust port of `Config`, `loadConfig` (with the
//! appliance release overlay) and the runtime validators from
//! `internal/host/daemon.go`.
//!
//! Field set, strict unknown-field rejection, validation order and error
//! conditions match Go. Two error texts cannot be Go-byte-exact: the
//! subnet failure propagates this crate's `net::parse_prefix` message
//! instead of Go's `netip` stdlib text (same fail/pass conditions), and
//! file-read failures carry the path. Like Go's single `Decode`, trailing
//! data after the first JSON value is ignored, duplicate fields resolve
//! last-wins, and invalid UTF-8 decodes lossy; only the per-byte versus
//! per-sequence replacement nuance differs from `encoding/json`.
//!
//! `apply_release_images` uses the landed crates:
//! `soda_release_deliver::payload::load` plus
//! `soda_release_build::files::require_native` and the payload image table.

use crate::{domain, net, pfactory};
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

#[cfg(test)]
mod tests;

/// Daemon runtime config (`host.Config`); JSON names match Go exactly.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Config {
    pub muse_sha256: String,
    pub muse_version: String,
    pub muse_socket: String,
    pub identity_socket: String,
    pub codex_harness: String,
    pub codex_harness_sha256: String,
    pub codex_harness_version: String,
    pub muse_harness: String,
    pub muse_harness_sha256: String,
    pub muse_harness_version: String,
    pub tailnet_management: bool,
    pub tailnet_image: String,
    pub image: String,
    pub network: String,
    pub subnet: String,
    pub bridge: String,
}

impl<'de> Deserialize<'de> for Config {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ConfigVisitor;

        impl<'de> Visitor<'de> for ConfigVisitor {
            type Value = Config;

            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a host configuration object")
            }

            fn visit_map<A>(self, mut map: A) -> Result<Config, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut config = Config::default();
                while let Some(key) = map.next_key::<String>()? {
                    macro_rules! string_field {
                        ($name:literal, $field:ident) => {
                            if key.eq_ignore_ascii_case($name) {
                                if let Some(value) = map.next_value::<Option<String>>()? {
                                    config.$field = value;
                                }
                                continue;
                            }
                        };
                    }
                    string_field!("muse_sha256", muse_sha256);
                    string_field!("muse_version", muse_version);
                    string_field!("muse_socket", muse_socket);
                    string_field!("identity_socket", identity_socket);
                    string_field!("codex_harness", codex_harness);
                    string_field!("codex_harness_sha256", codex_harness_sha256);
                    string_field!("codex_harness_version", codex_harness_version);
                    string_field!("muse_harness", muse_harness);
                    string_field!("muse_harness_sha256", muse_harness_sha256);
                    string_field!("muse_harness_version", muse_harness_version);
                    string_field!("tailnet_image", tailnet_image);
                    string_field!("image", image);
                    string_field!("network", network);
                    string_field!("subnet", subnet);
                    string_field!("bridge", bridge);
                    if key.eq_ignore_ascii_case("tailnet_management") {
                        if let Some(value) = map.next_value::<Option<bool>>()? {
                            config.tailnet_management = value;
                        }
                        continue;
                    }
                    return Err(de::Error::unknown_field(
                        &key,
                        &[
                            "muse_sha256",
                            "muse_version",
                            "muse_socket",
                            "identity_socket",
                            "codex_harness",
                            "codex_harness_sha256",
                            "codex_harness_version",
                            "muse_harness",
                            "muse_harness_sha256",
                            "muse_harness_version",
                            "tailnet_management",
                            "tailnet_image",
                            "image",
                            "network",
                            "subnet",
                            "bridge",
                        ],
                    ));
                }
                Ok(config)
            }
        }

        deserializer.deserialize_map(ConfigVisitor)
    }
}

/// `applyReleaseImages`: an empty release path is a no-op; otherwise the
/// appliance payload supplies both image defaults, and any saved image
/// choice conflicting with the release fails loudly instead of silently
/// replacing the operator's selection.
fn apply_release_images(c: &mut Config, release_path: &str) -> Result<(), String> {
    if release_path.is_empty() {
        return Ok(());
    }
    const UNAVAILABLE: &str = "immutable appliance image defaults unavailable";
    let payload = match soda_release_deliver::payload::load(release_path) {
        Ok(p) => p,
        Err(_) => return Err(UNAVAILABLE.to_string()),
    };
    if soda_release_build::files::require_native(&payload.architecture).is_err() {
        return Err(UNAVAILABLE.to_string());
    }
    let project_image = payload
        .images
        .get("project-os")
        .map(|i| i.config.as_str())
        .unwrap_or("");
    let companion_image = payload
        .images
        .get("tailnet")
        .map(|i| i.config.as_str())
        .unwrap_or("");
    if (!c.image.is_empty() && c.image != project_image)
        || (!c.tailnet_image.is_empty() && c.tailnet_image != companion_image)
    {
        return Err(
            "saved image selection conflicts with appliance release; explicit migration required"
                .to_string(),
        );
    }
    c.image = project_image.to_string();
    if c.tailnet_management {
        c.tailnet_image = companion_image.to_string();
    }
    Ok(())
}

/// `filepath.Base`: trailing slashes stripped, then the final element.
fn base_name(path: &str) -> &str {
    let trimmed = path.trim_end_matches('/');
    if trimmed.is_empty() {
        return "/";
    }
    match trimmed.rsplit('/').next() {
        Some(base) if !base.is_empty() => base,
        _ => "/",
    }
}

/// `validateMuseRuntime`.
fn validate_muse_runtime(c: &Config) -> Result<(), String> {
    if c.muse_sha256.is_empty() {
        return Ok(());
    }
    if !crate::terminal::is_clean_absolute_path(&c.muse_socket)
        || !crate::terminal::is_clean_absolute_path(&c.identity_socket)
        || !domain::valid_container_id(&c.muse_sha256)
        || c.muse_version.is_empty()
        || base_name(&c.muse_socket) != "launch.sock"
    {
        return Err("explicit muse socket, broker socket and release digest required".to_string());
    }
    Ok(())
}

/// `validateIdentityRuntime`, extended for the muse harness
/// family: each configured family needs the runtime socket and a
/// verified pin, exactly like codex.
fn validate_identity_runtime(c: &Config) -> Result<(), String> {
    if !c.codex_harness.is_empty() {
        if !crate::terminal::is_clean_absolute_path(&c.identity_socket)
            || !crate::terminal::is_clean_absolute_path(&c.codex_harness)
            || c.codex_harness_sha256.len() != 64
        {
            return Err(
                "explicit identity runtime socket and verified harness required".to_string(),
            );
        }
        if !c.codex_harness_version.is_empty()
            && !pfactory::valid_harness_version(&c.codex_harness_version)
        {
            return Err("invalid staged harness version".to_string());
        }
    }
    if !c.muse_harness.is_empty() {
        if !crate::terminal::is_clean_absolute_path(&c.identity_socket)
            || !crate::terminal::is_clean_absolute_path(&c.muse_harness)
            || c.muse_harness_sha256.len() != 64
        {
            return Err(
                "explicit identity runtime socket and verified harness required".to_string(),
            );
        }
        if !c.muse_harness_version.is_empty()
            && !pfactory::valid_harness_version(&c.muse_harness_version)
        {
            return Err("invalid staged harness version".to_string());
        }
    }
    Ok(())
}

/// `validTailnetImage`.
fn valid_tailnet_image(c: &Config) -> bool {
    if c.tailnet_image.is_empty() {
        return true;
    }
    c.tailnet_management
        && c.tailnet_image.starts_with("sha256:")
        && domain::valid_image_ref(&c.tailnet_image)
}

/// `validNetworkNames` (`networkName` is `domain::valid_login`'s regex).
fn valid_network_names(c: &Config) -> bool {
    !c.image.is_empty()
        && !c.image.starts_with('-')
        && domain::valid_login(&c.network)
        && domain::valid_login(&c.bridge)
}

/// `validateRuntimeConfig`: muse, identity, subnet prefix, tailnet image,
/// network names — in that order.
fn validate_runtime_config(c: &Config) -> Result<(), String> {
    validate_muse_runtime(c)?;
    validate_identity_runtime(c)?;
    net::parse_prefix(&c.subnet).map(|_| ())?;
    if !valid_tailnet_image(c) {
        return Err("invalid immutable Tailnet companion configuration".to_string());
    }
    if !valid_network_names(c) {
        return Err("invalid native runtime configuration".to_string());
    }
    Ok(())
}

/// `loadConfig`: strict-decode one config value (unknown fields rejected,
/// trailing data ignored like Go), overlay appliance release images, then
/// validate. `release_path` is the platform release file; empty disables
/// the overlay.
pub fn load_config(path: &str, release_path: &str) -> Result<Config, String> {
    let bytes = std::fs::read(path).map_err(|e| format!("read {path}: {e}"))?;
    let text = String::from_utf8_lossy(&bytes);
    let mut deserializer = serde_json::Deserializer::from_str(&text);
    // Like Go Decoder.Decode, consume one value and ignore any suffix. Config's
    // visitor keeps its historical case folding, null no-op, and last-wins rules.
    let mut c = Option::<Config>::deserialize(&mut deserializer)
        .map_err(|e| e.to_string())?
        .unwrap_or_default();
    apply_release_images(&mut c, release_path)?;
    validate_runtime_config(&c)?;
    Ok(c)
}
