use crate::json;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

// ---- PR20: account DTOs (`Account`, `AccessKeys`, `AccessKeyState`) ----

/// `Account`: a project login identity with its authorized keys.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Account {
    pub project: String,
    pub login: String,
    pub identity: i64,
    pub keys: Vec<String>,
}

macro_rules! string_field {
    ($key:expr, $name:literal, $map:expr, $field:expr) => {
        if $key.eq_ignore_ascii_case($name) {
            if let Some(value) = $map.next_value::<Option<String>>()? { $field = value; }
            continue;
        }
    };
}

macro_rules! string_list_field {
    ($key:expr, $name:literal, $map:expr, $field:expr) => {
        if $key.eq_ignore_ascii_case($name) {
            if let Some(value) = $map.next_value::<Option<Vec<Option<String>>>>()? {
                $field = value.into_iter().map(Option::unwrap_or_default).collect();
            }
            continue;
        }
    };
}

impl<'de> Deserialize<'de> for Account {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error> where D: serde::Deserializer<'de> {
        struct AccountVisitor;
        impl<'de> Visitor<'de> for AccountVisitor {
            type Value = Account;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str("an account object") }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error> where A: MapAccess<'de> {
                let mut out = Account { project: String::new(), login: String::new(), identity: 0, keys: Vec::new() };
                while let Some(key) = map.next_key::<String>()? {
                    string_field!(key, "project", map, out.project);
                    string_field!(key, "login", map, out.login);
                    if key.eq_ignore_ascii_case("identity") { if let Some(v) = map.next_value::<Option<i64>>()? { out.identity = v; } continue; }
                    string_list_field!(key, "keys", map, out.keys);
                    return Err(de::Error::unknown_field(&key, &["project", "login", "identity", "keys"]));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(AccountVisitor)
    }
}

impl Account {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
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

impl<'de> Deserialize<'de> for AccessKeys {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error> where D: serde::Deserializer<'de> {
        struct AccessKeysVisitor;
        impl<'de> Visitor<'de> for AccessKeysVisitor {
            type Value = AccessKeys;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str("an access keys object") }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error> where A: MapAccess<'de> {
                let mut out = AccessKeys { project: String::new(), login: String::new(), identity: 0, revision: String::new(), keys: Vec::new(), apply: false };
                while let Some(key) = map.next_key::<String>()? {
                    string_field!(key, "project", map, out.project);
                    string_field!(key, "login", map, out.login);
                    if key.eq_ignore_ascii_case("identity") { if let Some(v) = map.next_value::<Option<i64>>()? { out.identity = v; } continue; }
                    string_field!(key, "revision", map, out.revision);
                    string_list_field!(key, "keys", map, out.keys);
                    if key.eq_ignore_ascii_case("apply") { if let Some(v) = map.next_value::<Option<bool>>()? { out.apply = v; } continue; }
                    return Err(de::Error::unknown_field(&key, &["project", "login", "identity", "revision", "keys", "apply"]));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(AccessKeysVisitor)
    }
}

impl AccessKeys {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
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
            if i > 0 { out.push(','); }
            out.push_str(&json::quote(k));
        }
        out.push_str("]}");
        out
    }
}
