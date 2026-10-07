// Approved setup inputs and their digest.
use std::collections::HashMap;

use super::{
    valid_approved_name, FACTORY_CHECK_ENTRY, FACTORY_SETUP_ENTRY, MAX_APPROVED_FILES,
    MAX_APPROVED_FILE_SIZE, MAX_APPROVED_TOTAL, MAX_SOURCE_BUNDLE,
};
use crate::sha256;
use serde::de::{self, MapAccess, SeqAccess, Visitor};
use serde::Deserialize;
use std::fmt;

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct ApprovedSetup {
    pub files: HashMap<String, Vec<u8>>,
    pub bundle: Vec<u8>,
}

pub(super) struct GoBytes(pub(super) Vec<u8>);

impl<'de> Deserialize<'de> for GoBytes {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct BytesVisitor;
        impl<'de> Visitor<'de> for BytesVisitor {
            type Value = GoBytes;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("base64 text or a byte array")
            }
            fn visit_str<E>(self, value: &str) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                crate::ssh::b64_decode_go(value.as_bytes())
                    .map(GoBytes)
                    .map_err(|_| E::custom("invalid base64 bytes"))
            }
            fn visit_string<E>(self, value: String) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                self.visit_str(&value)
            }
            fn visit_unit<E>(self) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(GoBytes(Vec::new()))
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                let mut bytes = Vec::new();
                while let Some(value) = seq.next_element::<Option<u64>>()? {
                    let byte = value.unwrap_or(0);
                    if byte > u8::MAX as u64 {
                        return Err(de::Error::custom(format!("byte out of range: {byte}")));
                    }
                    bytes.push(byte as u8);
                }
                Ok(GoBytes(bytes))
            }
        }
        deserializer.deserialize_any(BytesVisitor)
    }
}

impl ApprovedSetup {
    pub fn validate(&self) -> Result<(), String> {
        if self.files.is_empty() || self.files.len() > MAX_APPROVED_FILES {
            return Err("invalid approved file set".to_string());
        }
        if !self.files.contains_key(FACTORY_SETUP_ENTRY) {
            return Err("approved setup entrypoint is required".to_string());
        }
        if !self.files.contains_key(FACTORY_CHECK_ENTRY) {
            return Err("approved check entrypoint is required".to_string());
        }
        // Go iterates the map in random order; sort so multi-fault reports
        // are deterministic (single faults agree either way).
        let mut names: Vec<&String> = self.files.keys().collect();
        names.sort();
        let mut total = 0usize;
        for name in names {
            let contents = &self.files[name];
            if !valid_approved_name(name) {
                return Err("invalid approved file name".to_string());
            }
            if contents.is_empty() || contents.len() > MAX_APPROVED_FILE_SIZE {
                return Err("invalid approved file size".to_string());
            }
            total += contents.len();
        }
        if total > MAX_APPROVED_TOTAL {
            return Err("approved inputs exceed the bounded size".to_string());
        }
        if self.bundle.is_empty() || self.bundle.len() > MAX_SOURCE_BUNDLE {
            return Err("invalid source bundle size".to_string());
        }
        Ok(())
    }
}

impl<'de> Deserialize<'de> for ApprovedSetup {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct SetupVisitor;
        impl<'de> Visitor<'de> for SetupVisitor {
            type Value = ApprovedSetup;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an approved setup object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut files = None;
                let mut bundle = None;
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("files") {
                        if let Some(value) = map.next_value::<Option<HashMap<String, GoBytes>>>()? {
                            files = Some(value.into_iter().map(|(k, v)| (k, v.0)).collect());
                        }
                    } else if key.eq_ignore_ascii_case("bundle") {
                        if let Some(value) = map.next_value::<Option<GoBytes>>()? {
                            bundle = Some(value.0);
                        }
                    } else {
                        return Err(de::Error::unknown_field(&key, &["files", "bundle"]));
                    }
                }
                Ok(ApprovedSetup {
                    files: files.unwrap_or_default(),
                    bundle: bundle.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_map(SetupVisitor)
    }
}

/// `SetupDigestOf`: sha256 over each file name, a zero byte, then contents,
/// names sorted; lowercase hex.
pub fn setup_digest_of(files: &HashMap<String, Vec<u8>>) -> String {
    let mut names: Vec<&String> = files.keys().collect();
    names.sort();
    let mut input = Vec::new();
    for name in names {
        input.extend_from_slice(name.as_bytes());
        input.push(0);
        input.extend_from_slice(&files[name]);
    }
    sha256::hex_lower(&sha256::digest(&input))
}
