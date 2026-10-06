//! Tailnet domain mirrors: error texts, DTOs, matchers, time/JSON helpers.
//! Lane A owns this file.
//!
//! Pure port of the `internal/tailnet` project-status surface: `ProjectHasNode`
//! and `ProjectStatus` (`project_status.go`), the `nativeObject` / `peerView` /
//! `addresses` helpers (`control.go`), `CanonicalMagicDNSName` (`tailnet.go`),
//! the project DTOs (`control_types.go`), `RunTarget` (`enrollment.go`) and
//! `RunBinding` (`project_runtime.go`). Go stdlib edge semantics (JSON,
//! `netip`, `time`) were verified against the pinned toolchain with probes.

use crate::domain;

pub const ERR_INVALID: &str = "invalid Tailnet request";
pub const ERR_CONFLICT: &str = "tailnet revision or identity changed";
pub const ERR_UNSUPPORTED: &str = "tailnet runtime is not supported";
pub const ERR_UNCONFIRMED: &str = "tailnet outcome is unconfirmed";
pub const ERR_UNAVAILABLE: &str = "tailscale status is unavailable";

/// Native LocalAPI body cap (`responseLimit` in control.go).
const RESPONSE_LIMIT: usize = 65536;

pub type ReadFile = Box<dyn Fn(&str) -> Result<Vec<u8>, String>>;
pub type LinkFile = Box<dyn Fn(&str) -> Result<String, String>>;
pub type StatFile = Box<dyn Fn(&str) -> Result<std::fs::Metadata, String>>;

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RunTarget {
    pub project: String,
    pub container: String,
    pub run: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RunBinding {
    pub enabled: bool,
    pub admission: bool,
    pub tailnet: String,
    pub tags: Vec<String>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProjectRequest {
    pub project: String,
    pub action: String,
    pub revision: String,
    pub binding: String,
    pub confirm_id: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProjectView {
    pub available_binding: String,
    pub available_network: String,
    pub addresses: Vec<String>,
    pub dns_name: String,
    pub saved: bool,
    pub project: String,
    pub revision: String,
    pub binding: String,
    pub enabled: bool,
    pub state: String,
    pub outcome: String,
}

/// `^p[0-9a-f]{24}$`
pub fn valid_project_id(s: &str) -> bool {
    domain::valid_id(s)
}

/// `^[0-9a-f]{64}$`
pub fn valid_container_id(s: &str) -> bool {
    domain::valid_container_id(s)
}

/// `^(sha256:)?[0-9a-f]{64}$`
pub fn valid_image_id(s: &str) -> bool {
    domain::valid_image_ref(s)
}

fn unavailable<T>() -> Result<T, String> {
    Err(ERR_UNAVAILABLE.to_string())
}
#[path = "tailnet/domain/time.rs"]
mod time;

pub use time::{go_escape, parse_rfc3339_nano};

#[path = "tailnet/domain/native.rs"]
mod native;

use native::{
    bind_bool_into, bind_string_into, decode_native_prefs, decode_native_status, fold_eq,
    lower_char, native_object, NativeStatus, SelfPeer,
};

#[path = "tailnet/domain/addresses.rs"]
mod addresses;

use addresses::{canonical_magic_dns_name, resolve_project_peer};

#[path = "tailnet/domain/status.rs"]
mod status;

pub use status::{project_has_node, project_status};

#[cfg(test)]
#[path = "tailnet/domain/node_tests.rs"]
mod node_tests;

#[cfg(test)]
#[path = "tailnet/domain/status_tests.rs"]
mod status_tests;

#[cfg(test)]
#[path = "tailnet/domain/address_tests.rs"]
mod address_tests;
