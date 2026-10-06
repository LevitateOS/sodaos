use super::{in_ranges, Environment, GO_CC, GO_CF};
use crate::json::{self};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OsRelease {
    pub id: String,
    pub version: String,
    pub name: String,
}

impl OsRelease {
    pub fn encode_into(&self, out: &mut String) {
        out.push_str("{\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"version\":");
        out.push_str(&json::quote(&self.version));
        out.push_str(",\"name\":");
        out.push_str(&json::quote(&self.name));
        out.push('}');
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OsObservation {
    pub environment: Environment,
    pub release: Option<OsRelease>,
    pub unavailable: bool,
}

impl OsObservation {
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"environment\":");
        self.environment.encode_into(&mut out);
        out.push_str(",\"os_release\":");
        match &self.release {
            Some(r) => r.encode_into(&mut out),
            None => out.push_str("null"),
        }
        out.push_str(",\"os_release_unavailable\":");
        out.push_str(if self.unavailable { "true" } else { "false" });
        out.push('}');
        out
    }
}

fn valid_os_id(s: &str) -> bool {
    // ^[a-z0-9][a-z0-9._-]{0,63}$
    let b = s.as_bytes();
    !b.is_empty()
        && b.len() <= 64
        && (b[0].is_ascii_lowercase() || b[0].is_ascii_digit())
        && b[1..].iter().all(|c| {
            c.is_ascii_lowercase() || c.is_ascii_digit() || *c == b'.' || *c == b'_' || *c == b'-'
        })
}

fn valid_os_version(s: &str) -> bool {
    // ^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$
    let b = s.as_bytes();
    !b.is_empty()
        && b.len() <= 64
        && b[0].is_ascii_alphanumeric()
        && b[1..]
            .iter()
            .all(|c| c.is_ascii_alphanumeric() || *c == b'.' || *c == b'_' || *c == b'-')
}

/// Port of `validOSRelease` in `internal/host/project/os.go`.
pub fn valid_os_release(p: &OsRelease) -> bool {
    if !valid_os_id(&p.id)
        || !valid_os_version(&p.version)
        || p.name.is_empty()
        || p.name.len() > 256
    {
        return false;
    }
    for c in p.name.chars() {
        if in_ranges(GO_CC, c) || in_ranges(GO_CF, c) {
            return false;
        }
    }
    true
}
