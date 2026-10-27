//! Store root and policy configuration (T00 frozen).
//!
//! The root is explicit, never a hardcoded stale release: `SODA_PROVIDERS_ROOT`
//! overrides, otherwise the installed default `/var/lib/soda/providers`.
//! The channel selects the upstream metadata source, never an obsolete pin:
//! `SODA_PROVIDERS_CHANNEL` overrides, otherwise `stable`.

use std::path::PathBuf;
use std::time::Duration;

/// Installed-system default. Integration tests and local runs set
/// `SODA_PROVIDERS_ROOT` to an isolated directory instead.
pub const DEFAULT_ROOT: &str = "/var/lib/soda/providers";
/// Default upstream channel for both providers.
pub const DEFAULT_CHANNEL: &str = "stable";
/// Default maximum age of a successful check: five minutes (RQ05).
pub const DEFAULT_CHECK_TTL_SECS: u64 = 300;

pub fn store_root() -> PathBuf {
    std::env::var("SODA_PROVIDERS_ROOT")
        .map(PathBuf::from)
        .unwrap_or_else(|_| PathBuf::from(DEFAULT_ROOT))
}

pub fn channel() -> String {
    std::env::var("SODA_PROVIDERS_CHANNEL").unwrap_or_else(|_| DEFAULT_CHANNEL.to_string())
}

/// Maximum age of a successful upstream check before a new launch refreshes.
/// Never zero in production paths; integration fixtures may set
/// `SODA_PROVIDERS_CHECK_TTL_SECS=0` to force a due check.
pub fn check_ttl() -> Duration {
    let secs: u64 = std::env::var("SODA_PROVIDERS_CHECK_TTL_SECS")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(DEFAULT_CHECK_TTL_SECS);
    Duration::from_secs(secs)
}
