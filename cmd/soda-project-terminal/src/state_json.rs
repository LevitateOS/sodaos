//! Ordered, RawValue-backed application data for factory and subscription
//! state. Serde owns JSON grammar; this node only retains decoded member order
//! and validated number tokens where those application paths observe them.

use serde::de::{Error as DeError, MapAccess, SeqAccess, Visitor};
use serde::ser::{Error as SerError, SerializeMap, SerializeSeq};
use serde::{Deserializer, Serialize, Serializer};
use serde_json::value::RawValue;

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum StateValue {
    Null,
    Bool(bool),
    Number(String),
    Str(String),
    Array(Vec<StateValue>),
    Object(Vec<(String, StateValue)>),
}

impl StateValue {
    pub(crate) fn parse(text: &str) -> Result<Self, String> {
        let raw: &RawValue = serde_json::from_str(text).map_err(|_| "invalid JSON".to_string())?;
        let mut value = Self::from_raw(raw, 0)?;
        value.collapse_dictionaries();
        Ok(value)
    }

    fn from_raw(raw: &RawValue, parent_depth: usize) -> Result<Self, String> {
        let token = raw.get();
        match token.as_bytes().first() {
            Some(b'{') => {
                let depth = parent_depth + 1;
                if depth > 127 {
                    return Err("invalid JSON".to_string());
                }
                struct ObjectVisitor {
                    depth: usize,
                }
                impl<'de> Visitor<'de> for ObjectVisitor {
                    type Value = StateValue;
                    fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        f.write_str("a JSON object")
                    }
                    fn visit_map<A: MapAccess<'de>>(
                        self,
                        mut map: A,
                    ) -> Result<Self::Value, A::Error> {
                        let mut entries = Vec::new();
                        while let Some(key) = map.next_key::<String>()? {
                            let raw = map.next_value::<&'de RawValue>()?;
                            let value =
                                StateValue::from_raw(raw, self.depth).map_err(A::Error::custom)?;
                            entries.push((key, value));
                        }
                        Ok(StateValue::Object(entries))
                    }
                }
                let mut decoder = serde_json::Deserializer::from_str(raw.get());
                let value = decoder
                    .deserialize_map(ObjectVisitor { depth })
                    .map_err(|_| "invalid JSON".to_string())?;
                decoder.end().map_err(|_| "invalid JSON".to_string())?;
                Ok(value)
            }
            Some(b'[') => {
                let depth = parent_depth + 1;
                if depth > 127 {
                    return Err("invalid JSON".to_string());
                }
                struct ArrayVisitor {
                    depth: usize,
                }
                impl<'de> Visitor<'de> for ArrayVisitor {
                    type Value = StateValue;
                    fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        f.write_str("a JSON array")
                    }
                    fn visit_seq<A: SeqAccess<'de>>(
                        self,
                        mut seq: A,
                    ) -> Result<Self::Value, A::Error> {
                        let mut values = Vec::new();
                        while let Some(raw) = seq.next_element::<&'de RawValue>()? {
                            let value =
                                StateValue::from_raw(raw, self.depth).map_err(A::Error::custom)?;
                            values.push(value);
                        }
                        Ok(StateValue::Array(values))
                    }
                }
                let mut decoder = serde_json::Deserializer::from_str(raw.get());
                let value = decoder
                    .deserialize_seq(ArrayVisitor { depth })
                    .map_err(|_| "invalid JSON".to_string())?;
                decoder.end().map_err(|_| "invalid JSON".to_string())?;
                Ok(value)
            }
            Some(b'"') => serde_json::from_str(token)
                .map(Self::Str)
                .map_err(|_| "invalid JSON".to_string()),
            Some(b't' | b'f') => serde_json::from_str(token)
                .map(Self::Bool)
                .map_err(|_| "invalid JSON".to_string()),
            Some(b'n') if token == "null" => Ok(Self::Null),
            Some(b'-' | b'0'..=b'9') => RawValue::from_string(token.to_owned())
                .map(|_| Self::Number(token.to_owned()))
                .map_err(|_| "invalid JSON".to_string()),
            _ => Err("invalid JSON".to_string()),
        }
    }

    fn collapse_dictionaries(&mut self) {
        match self {
            Self::Object(entries) => {
                for (_, value) in entries.iter_mut() {
                    value.collapse_dictionaries();
                }
                let mut collapsed: Vec<(String, StateValue)> = Vec::with_capacity(entries.len());
                let mut positions: std::collections::HashMap<String, usize> =
                    std::collections::HashMap::with_capacity(entries.len());
                for (key, value) in std::mem::take(entries) {
                    if let Some(index) = positions.get(&key).copied() {
                        collapsed[index].1 = value;
                    } else {
                        positions.insert(key.clone(), collapsed.len());
                        collapsed.push((key, value));
                    }
                }
                *entries = collapsed;
            }
            Self::Array(items) => items.iter_mut().for_each(Self::collapse_dictionaries),
            _ => {}
        }
    }

    pub(crate) fn get(&self, key: &str) -> Option<&Self> {
        self.object()?
            .iter()
            .rev()
            .find(|(name, _)| name == key)
            .map(|(_, value)| value)
    }

    pub(crate) fn object(&self) -> Option<&[(String, StateValue)]> {
        match self {
            Self::Object(entries) => Some(entries),
            _ => None,
        }
    }

    pub(crate) fn as_str(&self) -> Option<&str> {
        match self {
            Self::Str(value) => Some(value),
            _ => None,
        }
    }

    // The factory binary uses this accessor; the terminal binary shares this module.
    #[allow(dead_code)]
    pub(crate) fn as_bool(&self) -> Option<bool> {
        match self {
            Self::Bool(value) => Some(*value),
            _ => None,
        }
    }

    pub(crate) fn as_integer(&self) -> Option<i128> {
        let Self::Number(raw) = self else { return None };
        let digits = raw.strip_prefix('-').unwrap_or(raw);
        if digits.is_empty() || !digits.bytes().all(|byte| byte.is_ascii_digit()) {
            return None;
        }
        raw.parse().ok()
    }
}

impl Serialize for StateValue {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        match self {
            Self::Null => serializer.serialize_none(),
            Self::Bool(value) => serializer.serialize_bool(*value),
            Self::Number(raw) => RawValue::from_string(raw.clone())
                .map_err(S::Error::custom)?
                .serialize(serializer),
            Self::Str(value) => serializer.serialize_str(value),
            Self::Array(values) => {
                let mut sequence = serializer.serialize_seq(Some(values.len()))?;
                for value in values {
                    sequence.serialize_element(value)?;
                }
                sequence.end()
            }
            Self::Object(entries) => {
                let mut map = serializer.serialize_map(Some(entries.len()))?;
                for (key, value) in entries {
                    map.serialize_entry(key, value)?;
                }
                map.end()
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::StateValue;
    use crate::pyemit;

    #[test]
    fn python_dictionary_order_and_raw_numbers_survive_decode_and_emit() {
        let value = StateValue::parse(r#"{"first":1e2,"second":-0,"first":1.00}"#).unwrap();
        assert_eq!(pyemit::dumps(&value), r#"{"first":1.00,"second":-0}"#);
    }

    #[test]
    fn dynamic_tree_matches_serde_container_depth_limit() {
        assert_eq!(pyemit::dumps(&StateValue::parse("1e400").unwrap()), "1e400");
        let nested = |depth: usize| format!("{}1e400{}", "[".repeat(depth), "]".repeat(depth));
        let accepted = nested(127);
        assert_eq!(
            pyemit::dumps(&StateValue::parse(&accepted).unwrap()),
            accepted
        );
        let duplicate_object = format!(
            "{}{{\"first\":1e2,\"second\":-0,\"first\":1.00}}{}",
            "[".repeat(126),
            "]".repeat(126)
        );
        let expected = format!(
            "{}{{\"first\":1.00,\"second\":-0}}{}",
            "[".repeat(126),
            "]".repeat(126)
        );
        assert_eq!(
            pyemit::dumps(&StateValue::parse(&duplicate_object).unwrap()),
            expected
        );
        assert_eq!(StateValue::parse(&nested(128)).unwrap_err(), "invalid JSON");
    }
}
