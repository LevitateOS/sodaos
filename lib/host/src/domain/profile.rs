use super::{is_hex_lower, valid_id, valid_version, ROCKY_HEADLESS};
use crate::json::{self};
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

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

macro_rules! string_field {
    ($key:expr, $name:literal, $map:expr, $field:expr) => {
        if $key.eq_ignore_ascii_case($name) {
            if let Some(value) = $map.next_value::<Option<String>>()? { $field = value; }
            continue;
        }
    };
}

impl<'de> Deserialize<'de> for Profile {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error> where D: serde::Deserializer<'de> {
        struct ProfileVisitor;
        impl<'de> Visitor<'de> for ProfileVisitor {
            type Value = Profile;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str("a creation profile object") }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error> where A: MapAccess<'de> {
                let mut out = Profile { id: String::new(), distribution: String::new(), version: String::new(), interface: String::new(), architecture: String::new(), image: String::new(), revision: String::new() };
                while let Some(key) = map.next_key::<String>()? {
                    string_field!(key, "id", map, out.id);
                    string_field!(key, "distribution", map, out.distribution);
                    string_field!(key, "version", map, out.version);
                    string_field!(key, "interface", map, out.interface);
                    string_field!(key, "architecture", map, out.architecture);
                    string_field!(key, "image", map, out.image);
                    string_field!(key, "revision", map, out.revision);
                    return Err(de::Error::unknown_field(&key, &["id", "distribution", "version", "interface", "architecture", "image", "revision"]));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(ProfileVisitor)
    }
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

    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
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
    let profile = Profile::decode(raw.as_bytes())?;
    profile.validate()?;
    Ok(profile)
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Create {
    pub profile: Option<Profile>,
    pub id: String,
    pub owner: i64,
}

impl<'de> Deserialize<'de> for Create {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error> where D: serde::Deserializer<'de> {
        struct CreateVisitor;
        impl<'de> Visitor<'de> for CreateVisitor {
            type Value = Create;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str("a project creation object") }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error> where A: MapAccess<'de> {
                let mut out = Create { profile: None, id: String::new(), owner: 0 };
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("profile") { if let Some(v) = map.next_value::<Option<Profile>>()? { out.profile = Some(v); } continue; }
                    string_field!(key, "id", map, out.id);
                    if key.eq_ignore_ascii_case("owner") { if let Some(v) = map.next_value::<Option<i64>>()? { out.owner = v; } continue; }
                    return Err(de::Error::unknown_field(&key, &["profile", "id", "owner"]));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(CreateVisitor)
    }
}

impl Create {
    pub fn validate(&self) -> Result<(), String> {
        match &self.profile {
            Some(p) if valid_id(&self.id) && self.owner > 0 && p.validate().is_ok() => Ok(()),
            _ => Err("invalid creation identity".to_string()),
        }
    }

    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}
