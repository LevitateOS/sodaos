use super::{is_hex_lower, valid_id, valid_version, ROCKY_HEADLESS};
use crate::json::{self, BoundMap, Kind, Spec, Value};

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
