use crate::state_json::StateValue;
use serde::de::{MapAccess, SeqAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;

use crate::key_lines::canonical_lines;

/// Python truthiness over JSON values (only used for the `revision` preview
/// gate: `if data['revision'] or keys`).
#[derive(Debug, Clone, PartialEq)]
pub struct RevisionValue(String);

impl RevisionValue {
    pub(crate) fn from_raw(raw: &str) -> Self {
        Self(raw.to_string())
    }

    pub fn as_string(&self) -> Option<String> {
        serde_json::from_str(&self.0).ok()
    }

    pub fn is_truthy(&self) -> bool {
        let raw = self.0.trim();
        match raw.as_bytes().first() {
            Some(b'n' | b'f') => false,
            Some(b't') => true,
            Some(b'"') => serde_json::from_str::<String>(raw)
                .map(|value| !value.is_empty())
                .unwrap_or(true),
            Some(b'[' | b'{') => {
                let open = raw.find(|ch| ch == '[' || ch == '{').unwrap_or(0);
                let close = raw.rfind(|ch| ch == ']' || ch == '}').unwrap_or(raw.len());
                !raw[open + 1..close].trim().is_empty()
            }
            Some(b'-' | b'0'..=b'9') => raw.parse::<f64>().map(|n| n != 0.0).unwrap_or(true),
            _ => true,
        }
    }
}

/// Deserialize one JSON value while rejecting duplicate decoded object names
/// recursively, matching Python's `object_pairs_hook=unique` boundary.
fn check_unique_raw(raw: &str, parent_depth: usize) -> Result<(), String> {
    match raw.as_bytes().first() {
        Some(b'{') => {
            let depth = parent_depth + 1;
            if depth > 127 {
                return Err("invalid key operation".to_string());
            }
            struct ObjectVisitor;
            impl<'de> Visitor<'de> for ObjectVisitor {
                type Value = Vec<(String, Box<RawValue>)>;
                fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                    f.write_str("a JSON object")
                }
                fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                    let mut entries = Vec::new();
                    while let Some(key) = map.next_key::<String>()? {
                        entries.push((key, map.next_value::<Box<RawValue>>()?));
                    }
                    Ok(entries)
                }
            }
            let mut decoder = serde_json::Deserializer::from_str(raw);
            let entries = decoder
                .deserialize_map(ObjectVisitor)
                .map_err(|_| "invalid key operation".to_string())?;
            decoder
                .end()
                .map_err(|_| "invalid key operation".to_string())?;
            let mut names = std::collections::HashSet::new();
            for (key, value) in entries {
                if !names.insert(key) {
                    return Err("duplicate field".to_string());
                }
                check_unique_raw(value.get(), depth)?;
            }
            Ok(())
        }
        Some(b'[') => {
            let depth = parent_depth + 1;
            if depth > 127 {
                return Err("invalid key operation".to_string());
            }
            struct ArrayVisitor;
            impl<'de> Visitor<'de> for ArrayVisitor {
                type Value = Vec<Box<RawValue>>;
                fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                    f.write_str("a JSON array")
                }
                fn visit_seq<A: SeqAccess<'de>>(self, mut seq: A) -> Result<Self::Value, A::Error> {
                    let mut values = Vec::new();
                    while let Some(value) = seq.next_element::<Box<RawValue>>()? {
                        values.push(value);
                    }
                    Ok(values)
                }
            }
            let mut decoder = serde_json::Deserializer::from_str(raw);
            let values = decoder
                .deserialize_seq(ArrayVisitor)
                .map_err(|_| "invalid key operation".to_string())?;
            decoder
                .end()
                .map_err(|_| "invalid key operation".to_string())?;
            for value in values {
                check_unique_raw(value.get(), depth)?;
            }
            Ok(())
        }
        _ => Ok(()),
    }
}

/// Validated key request: exact shape, strict `apply`/`identity` types,
/// canonical desired bytes.
#[derive(Debug, Clone, PartialEq)]
pub struct KeyRequest {
    pub login: String,
    pub identity: i64,
    pub apply: bool,
    pub revision: RevisionValue,
    pub keys: Vec<String>,
    pub desired: Vec<u8>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct KeyRequestWire {
    login: String,
    identity: Box<RawValue>,
    apply: bool,
    revision: Box<RawValue>,
    keys: Vec<String>,
}

/// Pure request decode: duplicate validation followed by typed Serde binding.
pub fn decode_key_request(body: &[u8]) -> Result<KeyRequest, String> {
    let text = std::str::from_utf8(body).map_err(|_| "invalid key operation".to_string())?;
    let mut checker = serde_json::Deserializer::from_str(text);
    let raw = Box::<RawValue>::deserialize(&mut checker)
        .map_err(|_| "invalid key operation".to_string())?;
    checker
        .end()
        .map_err(|_| "invalid key operation".to_string())?;
    check_unique_raw(raw.get(), 0)?;
    let wire: KeyRequestWire =
        serde_json::from_str(text).map_err(|_| "invalid key operation".to_string())?;
    let identity: i64 = wire
        .identity
        .get()
        .parse()
        .map_err(|_| "invalid key operation".to_string())?;
    let mut desired = wire.keys.join("\n").into_bytes();
    if !wire.keys.is_empty() {
        desired.push(b'\n');
    }
    if !desired.is_ascii() || canonical_lines(&desired)? != wire.keys {
        return Err("invalid keys".to_string());
    }
    Ok(KeyRequest {
        login: wire.login,
        identity,
        apply: wire.apply,
        revision: RevisionValue::from_raw(wire.revision.get()),
        keys: wire.keys,
        desired,
    })
}

/// `{"revision","keys"}` result object in `.py` key order.
pub fn state_object(revision: &str, keys: &[String]) -> StateValue {
    StateValue::Object(vec![
        (
            "revision".to_string(),
            StateValue::Str(revision.to_string()),
        ),
        (
            "keys".to_string(),
            StateValue::Array(keys.iter().map(|k| StateValue::Str(k.clone())).collect()),
        ),
    ])
}
