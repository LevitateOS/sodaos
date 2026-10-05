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

use crate::json::{self, Kind, Spec};
use crate::{domain, net, pfactory};

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

const CONFIG_SPECS: &[Spec] = &[
    Spec {
        name: "muse_sha256",
        kind: Kind::Str,
    },
    Spec {
        name: "muse_version",
        kind: Kind::Str,
    },
    Spec {
        name: "muse_socket",
        kind: Kind::Str,
    },
    Spec {
        name: "identity_socket",
        kind: Kind::Str,
    },
    Spec {
        name: "codex_harness",
        kind: Kind::Str,
    },
    Spec {
        name: "codex_harness_sha256",
        kind: Kind::Str,
    },
    Spec {
        name: "codex_harness_version",
        kind: Kind::Str,
    },
    Spec {
        name: "muse_harness",
        kind: Kind::Str,
    },
    Spec {
        name: "muse_harness_sha256",
        kind: Kind::Str,
    },
    Spec {
        name: "muse_harness_version",
        kind: Kind::Str,
    },
    Spec {
        name: "tailnet_management",
        kind: Kind::Bool,
    },
    Spec {
        name: "tailnet_image",
        kind: Kind::Str,
    },
    Spec {
        name: "image",
        kind: Kind::Str,
    },
    Spec {
        name: "network",
        kind: Kind::Str,
    },
    Spec {
        name: "subnet",
        kind: Kind::Str,
    },
    Spec {
        name: "bridge",
        kind: Kind::Str,
    },
];

/// End offset of the first JSON value: Go's `loadConfig` performs one
/// `Decode` and never examines trailing data. Empty input fails like Go's
/// `io.EOF`. The slice is re-validated by the tolerant decoder, so this
/// only finds the boundary (strings/escapes/nesting aware).
fn first_value_end(text: &[u8]) -> Result<usize, String> {
    let mut i = 0;
    while i < text.len() && matches!(text[i], b' ' | b'\t' | b'\n' | b'\r') {
        i += 1;
    }
    if i == text.len() {
        return Err("EOF".to_string());
    }
    let mut j = i;
    let mut depth = 0usize;
    let mut in_string = false;
    let mut escape = false;
    let mut value_seen = false;
    while j < text.len() {
        let b = text[j];
        if in_string {
            if escape {
                escape = false;
            } else if b == b'\\' {
                escape = true;
            } else if b == b'"' {
                in_string = false;
                if depth == 0 {
                    return Ok(j + 1);
                }
            }
            j += 1;
            continue;
        }
        match b {
            b'"' => {
                in_string = true;
                value_seen = true;
            }
            b'{' | b'[' => {
                depth += 1;
                value_seen = true;
            }
            b'}' | b']' => {
                if depth == 0 {
                    break;
                }
                depth -= 1;
                if depth == 0 {
                    return Ok(j + 1);
                }
            }
            b' ' | b'\t' | b'\n' | b'\r' if depth == 0 && value_seen => {
                return Ok(j);
            }
            b',' | b':' if depth == 0 => break,
            _ => {
                value_seen = true;
            }
        }
        j += 1;
    }
    Ok(j)
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
    if !c.muse_socket.starts_with('/')
        || !c.identity_socket.starts_with('/')
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
        if !c.identity_socket.starts_with('/')
            || !c.codex_harness.starts_with('/')
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
        if !c.identity_socket.starts_with('/')
            || !c.muse_harness.starts_with('/')
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
    let end = first_value_end(text.as_bytes())?;
    let first = &text.as_bytes()[..end];
    // A literal `null` decodes into the struct as a no-op in Go.
    if first
        .iter()
        .filter(|b| !matches!(b, b' ' | b'\t' | b'\n' | b'\r'))
        .copied()
        .collect::<Vec<u8>>()
        == b"null"
    {
        let mut c = Config::default();
        apply_release_images(&mut c, release_path)?;
        return validate_runtime_config(&c).map(|()| c);
    }
    let v = json::decode_tolerant(first).map_err(|e| e.0)?;
    let m = json::bind_root(&v, "Config", CONFIG_SPECS, false).map_err(|e| e.0)?;
    let mut c = Config {
        muse_sha256: m.take_string("muse_sha256"),
        muse_version: m.take_string("muse_version"),
        muse_socket: m.take_string("muse_socket"),
        identity_socket: m.take_string("identity_socket"),
        codex_harness: m.take_string("codex_harness"),
        codex_harness_sha256: m.take_string("codex_harness_sha256"),
        codex_harness_version: m.take_string("codex_harness_version"),
        muse_harness: m.take_string("muse_harness"),
        muse_harness_sha256: m.take_string("muse_harness_sha256"),
        muse_harness_version: m.take_string("muse_harness_version"),
        tailnet_management: m.take_bool("tailnet_management"),
        tailnet_image: m.take_string("tailnet_image"),
        image: m.take_string("image"),
        network: m.take_string("network"),
        subnet: m.take_string("subnet"),
        bridge: m.take_string("bridge"),
    };
    apply_release_images(&mut c, release_path)?;
    validate_runtime_config(&c)?;
    Ok(c)
}
