//! Project domain DTOs and pure validation.
//!
//! Port of the `internal/project` surface used by the host daemon:
//! `profile.go`, `project.go` and the PR19 slice of `types.go`.
//! JSON shapes are the frozen Unix-socket wire contract; validation here
//! is pure (no I/O), exactly like the Go package.

use crate::json::{self, BoundMap, Kind, Spec, Value};

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

fn is_hex_lower(s: &str) -> bool {
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

const PROFILE_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "distribution",
        kind: Kind::Str,
    },
    Spec {
        name: "version",
        kind: Kind::Str,
    },
    Spec {
        name: "interface",
        kind: Kind::Str,
    },
    Spec {
        name: "architecture",
        kind: Kind::Str,
    },
    Spec {
        name: "image",
        kind: Kind::Str,
    },
    Spec {
        name: "revision",
        kind: Kind::Str,
    },
];

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Profile {
    pub id: String,
    pub distribution: String,
    pub version: String,
    pub interface: String,
    pub architecture: String,
    pub image: String,
    pub revision: String,
}

impl Profile {
    pub fn validate(&self) -> Result<(), String> {
        let digest_ok = self.image.len() == 71
            && self.image.starts_with("sha256:")
            && is_hex_lower(&self.image[7..]);
        let revision_ok = self.revision.len() == 40 && is_hex_lower(&self.revision);
        if self.id != ROCKY_HEADLESS
            || self.distribution != "rocky"
            || self.interface != "headless"
            || !valid_version(&self.version)
            || (self.architecture != "amd64" && self.architecture != "arm64")
            || !digest_ok
            || !revision_ok
        {
            return Err("unsupported or incomplete project creation profile".to_string());
        }
        Ok(())
    }

    /// Strict decode of one profile object (unknown fields rejected).
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "Profile", PROFILE_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        Profile {
            id: m.take_string("id"),
            distribution: m.take_string("distribution"),
            version: m.take_string("version"),
            interface: m.take_string("interface"),
            architecture: m.take_string("architecture"),
            image: m.take_string("image"),
            revision: m.take_string("revision"),
        }
    }

    /// `encoding/json` field order and escaping, no trailing newline
    /// (matches both the daemon response body and the creation label).
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        out.push_str("\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"distribution\":");
        out.push_str(&json::quote(&self.distribution));
        out.push_str(",\"version\":");
        out.push_str(&json::quote(&self.version));
        out.push_str(",\"interface\":");
        out.push_str(&json::quote(&self.interface));
        out.push_str(",\"architecture\":");
        out.push_str(&json::quote(&self.architecture));
        out.push_str(",\"image\":");
        out.push_str(&json::quote(&self.image));
        out.push_str(",\"revision\":");
        out.push_str(&json::quote(&self.revision));
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

/// `domain.Decode`: strict decode of a creation-profile label plus validation.
pub fn decode_profile(raw: &str) -> Result<Profile, String> {
    if raw.len() > 1024 {
        return Err("oversized project profile".to_string());
    }
    let v = json::decode_strict(raw.as_bytes()).map_err(|e| e.0)?;
    let p = Profile::from_value(&v)?;
    p.validate()?;
    Ok(p)
}

const CREATE_SPECS: &[Spec] = &[
    Spec {
        name: "profile",
        kind: Kind::OptObject {
            go_type: "project.Profile",
            struct_name: "Profile",
            specs: PROFILE_SPECS,
        },
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "owner",
        kind: Kind::I64,
    },
];

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Create {
    pub profile: Option<Profile>,
    pub id: String,
    pub owner: i64,
}

impl Create {
    pub fn validate(&self) -> Result<(), String> {
        match &self.profile {
            Some(p) if valid_id(&self.id) && self.owner > 0 && p.validate().is_ok() => Ok(()),
            _ => Err("invalid creation identity".to_string()),
        }
    }

    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "Create", CREATE_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        Create {
            profile: m.take_opt_map("profile").map(|c| Profile::from_map(&c)),
            id: m.take_string("id"),
            owner: m.take_i64("owner"),
        }
    }
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

// ---- PR20: account DTOs (`Account`, `AccessKeys`, `AccessKeyState`) ----

/// `Account`: a project login identity with its authorized keys.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Account {
    pub project: String,
    pub login: String,
    pub identity: i64,
    pub keys: Vec<String>,
}

const ACCOUNT_SPECS: &[Spec] = &[
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "identity",
        kind: Kind::I64,
    },
    Spec {
        name: "keys",
        kind: Kind::StrList,
    },
];

impl Account {
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "Account", ACCOUNT_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        Account {
            project: m.take_string("project"),
            login: m.take_string("login"),
            identity: m.take_i64("identity"),
            keys: m.take_str_list("keys"),
        }
    }
}

/// `AccessKeys`: replace or observe a login's authorized key set.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccessKeys {
    pub project: String,
    pub login: String,
    pub identity: i64,
    pub revision: String,
    pub keys: Vec<String>,
    pub apply: bool,
}

const ACCESS_KEYS_SPECS: &[Spec] = &[
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "identity",
        kind: Kind::I64,
    },
    Spec {
        name: "revision",
        kind: Kind::Str,
    },
    Spec {
        name: "keys",
        kind: Kind::StrList,
    },
    Spec {
        name: "apply",
        kind: Kind::Bool,
    },
];

impl AccessKeys {
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "AccessKeys", ACCESS_KEYS_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        AccessKeys {
            project: m.take_string("project"),
            login: m.take_string("login"),
            identity: m.take_i64("identity"),
            revision: m.take_string("revision"),
            keys: m.take_str_list("keys"),
            apply: m.take_bool("apply"),
        }
    }
}

/// `AccessKeyState`: the observed key set and its revision.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccessKeyState {
    pub revision: String,
    pub keys: Vec<String>,
}

impl AccessKeyState {
    /// `encoding/json` struct order (`revision`, `keys`), no trailing newline.
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"revision\":");
        out.push_str(&json::quote(&self.revision));
        out.push_str(",\"keys\":[");
        for (i, k) in self.keys.iter().enumerate() {
            if i > 0 {
                out.push(',');
            }
            out.push_str(&json::quote(k));
        }
        out.push_str("]}");
        out
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sample_profile() -> Profile {
        Profile {
            id: ROCKY_HEADLESS.to_string(),
            distribution: "rocky".to_string(),
            version: "9.7".to_string(),
            interface: "headless".to_string(),
            architecture: "amd64".to_string(),
            image: format!("sha256:{}", "a".repeat(64)),
            revision: "b".repeat(40),
        }
    }

    #[test]
    fn validators_match_go_regexps() {
        assert!(valid_id(&format!("p{}", "a".repeat(24))));
        assert!(!valid_id("p123"));
        assert!(!valid_id(&format!("P{}", "a".repeat(24))));
        assert!(!valid_id(&format!("p{}", "A".repeat(24))));
        assert!(valid_login("soda-tester"));
        assert!(valid_login("root-1")); // exact "root" is rejected at the op layer, not here
        assert!(!valid_login("1abc"));
        assert!(!valid_login("Root"));
        assert!(valid_image_ref(&"c".repeat(64)));
        assert!(valid_image_ref(&format!("sha256:{}", "c".repeat(64))));
        assert!(!valid_image_ref("c"));
        assert!(valid_container_id(&"d".repeat(64)));
        assert!(!valid_container_id("xyz"));
    }

    #[test]
    fn profile_validation_matches() {
        assert!(sample_profile().validate().is_ok());
        let mut p = sample_profile();
        p.architecture = "x86_64".to_string();
        assert!(p.validate().is_err());
        let mut p = sample_profile();
        p.version = "9.7.1.2".to_string();
        assert!(p.validate().is_err());
        let mut p = sample_profile();
        p.version = "9".to_string();
        assert!(p.validate().is_ok());
    }

    #[test]
    fn profile_wire_round_trip() {
        let p = sample_profile();
        let enc = p.encode();
        assert_eq!(decode_profile(&enc).unwrap(), p);
        // Unknown fields rejected, like strictjson.
        assert!(decode_profile(&enc.replace('}', ",\"x\":1}")).is_err());
        // Oversized labels rejected before parsing.
        assert!(decode_profile(&"x".repeat(1025)).is_err());
    }

    #[test]
    fn create_decode_and_validate() {
        let raw = format!(
            "{{\"profile\":{},\"id\":\"p{}\",\"owner\":42}}",
            sample_profile().encode(),
            "e".repeat(24)
        );
        let v = json::decode_strict(raw.as_bytes()).unwrap();
        let c = Create::from_value(&v).unwrap();
        assert!(c.validate().is_ok());
        let v = json::decode_strict(br#"{"id":"pAAAAAAAAAAAAAAAAAAAAAAAA"}"#).unwrap();
        let c = Create::from_value(&v).unwrap();
        assert!(c.validate().is_err()); // missing profile + bad id
    }

    #[test]
    fn environment_omits_empty_like_go() {
        let env = Environment {
            image: String::new(),
            profile: None,
            id: "x".to_string(),
            ip: String::new(),
            running: false,
        };
        assert_eq!(env.encode(), r#"{"id":"x","ip":"","running":false}"#);
        let env = Environment {
            image: "sha256:1".to_string(),
            profile: Some(sample_profile()),
            id: "x".to_string(),
            ip: "10.0.0.2".to_string(),
            running: true,
        };
        let enc = env.encode();
        assert!(enc.starts_with(r#"{"image":"sha256:1","profile":{"id":"rocky-headless""#));
        assert!(enc.ends_with(r#""ip":"10.0.0.2","running":true}"#));
    }

    #[test]
    fn os_release_validation_matches_go() {
        let good = OsRelease {
            id: "rocky".to_string(),
            version: "9.7".to_string(),
            name: "Rocky Linux 9.7".to_string(),
        };
        assert!(valid_os_release(&good));
        assert!(!valid_os_release(&OsRelease {
            id: "Rocky".to_string(),
            ..good.clone()
        }));
        assert!(!valid_os_release(&OsRelease {
            name: "".to_string(),
            ..good.clone()
        }));
        assert!(!valid_os_release(&OsRelease {
            name: "a\u{200b}b".to_string(),
            ..good.clone()
        })); // Cf ZWSP
        assert!(!valid_os_release(&OsRelease {
            name: "a\u{7f}b".to_string(),
            ..good.clone()
        })); // Cc DEL
        assert!(!valid_os_release(&OsRelease {
            name: "a\u{ad}b".to_string(),
            ..good.clone()
        })); // Cf soft hyphen
        assert!(valid_os_release(&OsRelease {
            name: "Rocky Linux \u{e9}".to_string(),
            ..good.clone()
        }));
    }

    #[test]
    fn account_dto_strict_shape() {
        let v = json::decode_strict(
            br#"{"project":"p0123456789abcdef01234567","login":"alice","identity":1,"keys":["a","b"]}"#,
        )
        .unwrap();
        let a = Account::from_value(&v).unwrap();
        assert_eq!(a.login, "alice");
        assert_eq!(a.identity, 1);
        assert_eq!(a.keys, vec!["a".to_string(), "b".to_string()]);
        // Missing keys decodes as empty (browser-only account).
        let v = json::decode_strict(
            br#"{"project":"p0123456789abcdef01234567","login":"alice","identity":1}"#,
        )
        .unwrap();
        assert!(Account::from_value(&v).unwrap().keys.is_empty());
        // Unknown fields rejected.
        let v = json::decode_strict(br#"{"project":"p","login":"a","identity":1,"admin":true}"#)
            .unwrap();
        assert!(Account::from_value(&v).is_err());
        // Non-integer identity rejected.
        let v = json::decode_strict(br#"{"project":"p","login":"a","identity":1.5}"#).unwrap();
        assert!(Account::from_value(&v).is_err());
    }

    #[test]
    fn access_keys_dto_strict_shape() {
        let v = json::decode_strict(
            br#"{"project":"p","login":"a","identity":7,"revision":"r","keys":["k"],"apply":true}"#,
        )
        .unwrap();
        let a = AccessKeys::from_value(&v).unwrap();
        assert_eq!(a.identity, 7);
        assert!(a.apply);
        assert_eq!(a.keys, vec!["k".to_string()]);
        // Duplicates rejected at the strict layer.
        assert!(json::decode_strict(br#"{"project":"p","project":"q"}"#).is_err());
    }

    #[test]
    fn access_key_state_encodes_struct_order() {
        let s = AccessKeyState {
            revision: "r".to_string(),
            keys: vec!["a\"b".to_string()],
        };
        assert_eq!(s.encode(), "{\"revision\":\"r\",\"keys\":[\"a\\\"b\"]}");
    }
}
