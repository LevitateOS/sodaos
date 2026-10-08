//! Project domain DTOs and pure validation.
//!
//! Port of the `internal/project` surface used by the host daemon:
//! `profile.go`, `project.go` and the PR19 slice of `types.go`.
//! JSON shapes are the frozen Unix-socket wire contract; validation here
//! is pure (no I/O), exactly like the Go package.

use crate::json::{self};

mod account;
mod os;
mod profile;

#[cfg(test)]
mod tests;

pub use self::account::{
    AccessKeyState, AccessKeys, Account, ProjectAccessRequest, ProjectAccessStatus,
};
pub use self::os::{valid_os_release, OsObservation, OsRelease};
pub use self::profile::{decode_profile, Create, Profile};

pub const ROCKY_HEADLESS: &str = "rocky-headless";

// Exact `unicode.Cc` / `unicode.Cf` ranges from the pinned Go toolchain
// (go1.26.7), used by the OS-release name check.
const GO_CC: &[(u32, u32)] = &[(0x0, 0x1F), (0x7F, 0x9F)];
const GO_CF: &[(u32, u32)] = &[
    (0xAD, 0xAD),
    (0x600, 0x605),
    (0x61C, 0x61C),
    (0x6DD, 0x6DD),
    (0x70F, 0x70F),
    (0x890, 0x891),
    (0x8E2, 0x8E2),
    (0x180E, 0x180E),
    (0x200B, 0x200F),
    (0x202A, 0x202E),
    (0x2060, 0x2064),
    (0x2066, 0x206F),
    (0xFEFF, 0xFEFF),
    (0xFFF9, 0xFFFB),
    (0x110BD, 0x110BD),
    (0x110CD, 0x110CD),
    (0x13430, 0x1343F),
    (0x1BCA0, 0x1BCA3),
    (0x1D173, 0x1D17A),
    (0xE0001, 0xE0001),
    (0xE0020, 0xE007F),
];

fn in_ranges(table: &[(u32, u32)], c: char) -> bool {
    let v = c as u32;
    table.iter().any(|&(lo, hi)| v >= lo && v <= hi)
}

pub(crate) fn is_hex_lower(s: &str) -> bool {
    !s.is_empty()
        && s.bytes()
            .all(|b| b.is_ascii_hexdigit() && (b.is_ascii_digit() || b.is_ascii_lowercase()))
}

fn valid_version(v: &str) -> bool {
    // ^[0-9]{1,3}(\.[0-9]{1,3}){0,2}$
    let mut parts = v.split('.');
    let first = match parts.next() {
        Some(p) if !p.is_empty() && p.len() <= 3 && p.bytes().all(|b| b.is_ascii_digit()) => p,
        _ => return false,
    };
    let _ = first;
    let mut count = 1;
    for p in parts {
        count += 1;
        if count > 3 || p.is_empty() || p.len() > 3 || !p.bytes().all(|b| b.is_ascii_digit()) {
            return false;
        }
    }
    true
}

/// `^p[0-9a-f]{24}$`
pub fn valid_id(id: &str) -> bool {
    id.len() == 25 && id.as_bytes()[0] == b'p' && is_hex_lower(&id[1..])
}

/// `^[a-z][a-z0-9_-]{0,30}$`
pub fn valid_login(login: &str) -> bool {
    let b = login.as_bytes();
    !b.is_empty()
        && b.len() <= 31
        && b[0].is_ascii_lowercase()
        && b[1..]
            .iter()
            .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || *c == b'_' || *c == b'-')
}

/// `^(?:sha256:)?[0-9a-f]{64}$`
pub fn valid_image_ref(reference: &str) -> bool {
    let hex = reference.strip_prefix("sha256:").unwrap_or(reference);
    hex.len() == 64 && is_hex_lower(hex)
}

/// `^[0-9a-f]{64}$`
pub fn valid_container_id(id: &str) -> bool {
    id.len() == 64 && is_hex_lower(id)
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Environment {
    pub image: String,
    pub profile: Option<Profile>,
    pub id: String,
    pub ip: String,
    pub running: bool,
}

impl Environment {
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        if !self.image.is_empty() {
            first = false;
            out.push_str("\"image\":");
            out.push_str(&json::quote(&self.image));
        }
        if let Some(p) = &self.profile {
            if !first {
                out.push(',');
            }
            first = false;
            out.push_str("\"profile\":");
            p.encode_into(out);
        }
        if !first {
            out.push(',');
        }
        out.push_str("\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"ip\":");
        out.push_str(&json::quote(&self.ip));
        out.push_str(",\"running\":");
        out.push_str(if self.running { "true" } else { "false" });
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Connection {
    pub environment: Environment,
    pub host_key: String,
    pub fingerprint: String,
}

impl Connection {
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"environment\":");
        self.environment.encode_into(&mut out);
        out.push_str(",\"host_key\":");
        out.push_str(&json::quote(&self.host_key));
        out.push_str(",\"fingerprint\":");
        out.push_str(&json::quote(&self.fingerprint));
        out.push('}');
        out
    }
}
