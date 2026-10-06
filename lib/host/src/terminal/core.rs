use std::time::{SystemTime, UNIX_EPOCH};

use crate::domain;

// ---------- wire limits ----------

/// `FrameLimit`: bounded 64-row metadata; IO payload bounds stay smaller.
pub const FRAME_LIMIT: usize = 131072;
/// `terminalLimit`: at most 64 terminals per metadata frame.
pub const TERMINAL_LIMIT: usize = 64;
/// Live-stream admission cap: `2*terminalLimit`.
pub const STREAM_LIMIT: usize = 128;
/// Agent stdin delivery cap for broker calls (`384<<10`).
pub const BROKER_RESPONSE_LIMIT: usize = 384 * 1024;
/// Credential byte cap (`256<<10`).
pub const CREDENTIAL_LIMIT: usize = 256 * 1024;

// ---------- identity errors ----------

/// `identity.ErrDenied`.
pub const ERR_DENIED: &str = "identity authority denied";
/// `identity.ErrStale`.
pub const ERR_STALE: &str = "identity generation changed";
/// `identity.ErrUncertain`.
pub const ERR_UNCERTAIN: &str = "subscription requires reconnection";
/// `identity.ErrNotFound`.
pub const ERR_NOT_FOUND: &str = "identity execution missing";

/// Daemon identity-route errors from `internal/host/identity.go`.
pub const ERR_INVALID_IDENTITY_OPERATION: &str = "invalid identity operation";
pub const ERR_IDENTITY_UNCONFIRMED: &str = "identity operation unconfirmed";
/// `POST /identity/launch` path.
pub const IDENTITY_LAUNCH_PATH: &str = "/identity/launch";
/// Provider/kind discriminators from `internal/identity/types.go`.
pub const PROVIDER_CODEX: &str = "codex";
pub const PROVIDER_MUSE: &str = "muse";
pub const KIND_FACTORY: &str = "factory";
pub const KIND_TERMINAL: &str = "terminal";
/// Muse project-execution scope carried in broker bindings.
pub const SCOPE_MUSE_PROJECT: &str = "muse-project";

pub fn err_denied() -> String {
    ERR_DENIED.to_string()
}
pub fn err_stale() -> String {
    ERR_STALE.to_string()
}
pub fn err_uncertain() -> String {
    ERR_UNCERTAIN.to_string()
}

// ---------- small predicates ----------

/// `terminalID = ^[0-9a-f]{32}$`.
pub fn valid_terminal_id(id: &str) -> bool {
    id.len() == 32 && domain::is_hex_lower(id)
}

/// `now` as Unix seconds, saturating on clock failure.
pub fn now_unix() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}
