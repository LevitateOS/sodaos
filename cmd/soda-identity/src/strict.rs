// Bounded strict JSON admission, mirroring internal/strictjson: exactly one
// object, valid UTF-8, no duplicate fields at any depth (depth cap 100).
// Settings uses `decode` with its Go-compatible case-folding projection;
// private HTTP wire DTOs use `decode_typed` and their exact field names.
use serde::de::{self, DeserializeSeed, MapAccess, SeqAccess, Visitor};
use std::collections::HashSet;
use std::fmt;

pub const MAX_DOCUMENT: usize = 1 << 20;

/// Decode one bounded HTTP protocol DTO without rewriting its JSON keys.
/// The owning wire structs enforce their exact field names with
/// `deny_unknown_fields`; Settings continues to use the profile-aware
/// `decode` path below.
pub fn decode_typed<T: serde::de::DeserializeOwned>(input: &[u8], max: usize) -> Result<T, String> {
    if input.len() > max {
        return Err("request exceeds size limit".to_string());
    }
    let text =
        std::str::from_utf8(input).map_err(|_| "request must contain valid UTF-8".to_string())?;
    check_unique_keys(text)?;
    serde_json::from_str(text).map_err(|e| format!("decode request: {e}"))
}

pub fn decode<T: serde::de::DeserializeOwned>(
    input: &[u8],
    max: usize,
    fields: &[&str],
    nested: &[(&str, &[&str])],
) -> Result<T, String> {
    if input.len() > max {
        return Err("request exceeds size limit".to_string());
    }
    let text =
        std::str::from_utf8(input).map_err(|_| "request must contain valid UTF-8".to_string())?;
    check_unique_keys(text)?;
    let mut value: serde_json::Value =
        serde_json::from_str(text).map_err(|e| format!("decode request: {e}"))?;
    let object = value
        .as_object_mut()
        .ok_or_else(|| "request must be one JSON object".to_string())?;
    remap_case(object, fields);
    for (name, sub) in nested {
        if let Some(inner) = object.get_mut(*name).and_then(|v| v.as_object_mut()) {
            remap_case(inner, sub);
        }
    }
    check_known_fields(object, fields, nested)?;
    serde_json::from_value(value).map_err(|e| format!("decode request: {e}"))
}

fn remap_case(object: &mut serde_json::Map<String, serde_json::Value>, fields: &[&str]) {
    // Go prefers an exact field match but also accepts a case-insensitive
    // one. Keys that would collide after folding are left alone so the
    // unknown-field check rejects them.
    let mut renames = Vec::new();
    for key in object.keys() {
        if fields.contains(&key.as_str()) {
            continue;
        }
        let mut folded = None;
        for field in fields {
            if field.eq_ignore_ascii_case(key) {
                folded = Some(*field);
                break;
            }
        }
        if let Some(field) = folded {
            if !object.contains_key(field) {
                renames.push((key.clone(), field.to_string()));
            }
        }
    }
    for (from, to) in renames {
        if let Some(value) = object.remove(&from) {
            object.insert(to, value);
        }
    }
}

fn check_known_fields(
    object: &serde_json::Map<String, serde_json::Value>,
    fields: &[&str],
    nested: &[(&str, &[&str])],
) -> Result<(), String> {
    for key in object.keys() {
        if !fields.contains(&key.as_str()) {
            return Err(format!("unknown field {key:?}"));
        }
    }
    for (name, sub) in nested {
        if let Some(inner) = object.get(*name).and_then(|v| v.as_object()) {
            for key in inner.keys() {
                if !sub.contains(&key.as_str()) {
                    return Err(format!("unknown field {key:?}"));
                }
            }
        }
    }
    Ok(())
}

struct UniqueSeed {
    depth: i16,
    top: Option<String>,
}

impl<'de> DeserializeSeed<'de> for UniqueSeed {
    type Value = ();

    fn deserialize<D>(self, deserializer: D) -> Result<Self::Value, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        if self.depth > 100 {
            let message = match self.top {
                Some(top) => format!("decode request field {top:?}: request is nested too deeply"),
                None => "request is nested too deeply".to_string(),
            };
            return Err(de::Error::custom(message));
        }
        deserializer.deserialize_any(UniqueVisitor {
            depth: self.depth,
            top: self.top,
        })
    }
}

struct UniqueVisitor {
    depth: i16,
    top: Option<String>,
}

impl<'de> Visitor<'de> for UniqueVisitor {
    type Value = ();

    fn expecting(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str("a JSON value without duplicate object keys")
    }

    fn visit_bool<E>(self, _: bool) -> Result<(), E> {
        Ok(())
    }

    fn visit_i64<E>(self, _: i64) -> Result<(), E> {
        Ok(())
    }

    fn visit_u64<E>(self, _: u64) -> Result<(), E> {
        Ok(())
    }

    fn visit_f64<E>(self, _: f64) -> Result<(), E> {
        Ok(())
    }

    fn visit_str<E>(self, _: &str) -> Result<(), E> {
        Ok(())
    }

    fn visit_string<E>(self, _: String) -> Result<(), E> {
        Ok(())
    }

    fn visit_unit<E>(self) -> Result<(), E> {
        Ok(())
    }

    fn visit_none<E>(self) -> Result<(), E> {
        Ok(())
    }

    fn visit_seq<A>(self, mut seq: A) -> Result<(), A::Error>
    where
        A: SeqAccess<'de>,
    {
        while seq
            .next_element_seed(UniqueSeed {
                depth: self.depth + 1,
                top: self.top.clone(),
            })?
            .is_some()
        {}
        Ok(())
    }

    fn visit_map<A>(self, mut map: A) -> Result<(), A::Error>
    where
        A: MapAccess<'de>,
    {
        let mut keys = HashSet::new();
        while let Some(key) = map.next_key::<String>()? {
            if !keys.insert(key.clone()) {
                return Err(de::Error::custom(format!(
                    "duplicate request field {key:?}"
                )));
            }
            let top = self
                .top
                .clone()
                .or_else(|| (self.depth == -1).then_some(key));
            map.next_value_seed(UniqueSeed {
                depth: self.depth + 1,
                top,
            })?;
        }
        Ok(())
    }
}

fn check_unique_keys(text: &str) -> Result<(), String> {
    let mut deserializer = serde_json::Deserializer::from_str(text);
    UniqueSeed {
        // Root keys are checked here, while the root object itself does not
        // consume one of the existing 100 nested-value levels.
        depth: -1,
        top: None,
    }
    .deserialize(&mut deserializer)
    .map_err(|error| format!("decode request: {error}"))?;
    deserializer
        .end()
        .map_err(|error| format!("decode request: {error}"))
}

#[cfg(test)]
#[path = "strict_tests.rs"]
mod strict_tests;
