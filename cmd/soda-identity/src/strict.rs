// Bounded strict JSON admission, mirroring internal/strictjson: exactly one
// object, valid UTF-8, no duplicate fields at any depth (depth cap 100).
use serde::de::{self, DeserializeSeed, MapAccess, SeqAccess, Visitor};
use std::collections::HashSet;
use std::fmt;

pub const MAX_DOCUMENT: usize = 1 << 20;

/// Decode one bounded object without rewriting its JSON keys. The owning DTO
/// enforces exact field names with `deny_unknown_fields`.
pub fn decode_typed<T: serde::de::DeserializeOwned>(input: &[u8], max: usize) -> Result<T, String> {
    if input.len() > max {
        return Err("request exceeds size limit".to_string());
    }
    let text =
        std::str::from_utf8(input).map_err(|_| "request must contain valid UTF-8".to_string())?;
    check_unique_object(text)?;
    serde_json::from_str(text).map_err(|e| format!("decode request: {e}"))
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

fn check_unique_object(text: &str) -> Result<(), String> {
    let mut deserializer = serde_json::Deserializer::from_str(text);
    serde::Deserializer::deserialize_map(
        &mut deserializer,
        UniqueVisitor {
            // Root keys are checked here, while the root object itself does
            // not consume one of the existing 100 nested-value levels.
            depth: -1,
            top: None,
        },
    )
    .map_err(|error| format!("decode request: {error}"))?;
    deserializer
        .end()
        .map_err(|error| format!("decode request: {error}"))
}

#[cfg(test)]
#[path = "strict_tests.rs"]
mod strict_tests;
