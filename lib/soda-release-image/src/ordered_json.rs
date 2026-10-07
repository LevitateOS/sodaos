//! Ordered raw JSON values for the few image metadata boundaries that observe
//! object order, duplicate members, or the original spelling of numbers.

use serde::de::{Error as DeError, MapAccess, SeqAccess, Visitor};
use serde::ser::{Error as _, SerializeMap, SerializeSeq};
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use serde_json::value::RawValue;

use crate::error::Error;

#[derive(Debug, Clone, PartialEq)]
pub(crate) enum OrderedValue {
    Null,
    Bool(bool),
    Number(String),
    String(String),
    Array(Vec<OrderedValue>),
    Object(Vec<(String, OrderedValue)>),
}

impl OrderedValue {
    pub(crate) fn parse(text: &str) -> Result<Self, Error> {
        let mut decoder = serde_json::Deserializer::from_str(text);
        let raw =
            Box::<RawValue>::deserialize(&mut decoder).map_err(|_| Error::msg("invalid JSON"))?;
        decoder.end().map_err(|_| Error::msg("invalid JSON"))?;
        Self::from_raw(raw.get(), 0)
    }

    fn from_raw(raw: &str, parent_depth: usize) -> Result<Self, Error> {
        match raw.as_bytes().first() {
            Some(b'{') => {
                let depth = parent_depth + 1;
                if depth > 127 {
                    return Err(Error::msg("invalid JSON"));
                }
                struct ObjectVisitor {
                    depth: usize,
                }
                impl<'de> Visitor<'de> for ObjectVisitor {
                    type Value = OrderedValue;
                    fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        f.write_str("a JSON object")
                    }
                    fn visit_map<A: MapAccess<'de>>(
                        self,
                        mut map: A,
                    ) -> Result<Self::Value, A::Error> {
                        let mut entries = Vec::new();
                        while let Some(key) = map.next_key::<String>()? {
                            let raw = map.next_value::<Box<RawValue>>()?;
                            let value = OrderedValue::from_raw(raw.get(), self.depth)
                                .map_err(A::Error::custom)?;
                            entries.push((key, value));
                        }
                        Ok(OrderedValue::Object(entries))
                    }
                }
                let mut decoder = serde_json::Deserializer::from_str(raw);
                let value = decoder
                    .deserialize_map(ObjectVisitor { depth })
                    .map_err(|_| Error::msg("invalid JSON"))?;
                decoder.end().map_err(|_| Error::msg("invalid JSON"))?;
                Ok(value)
            }
            Some(b'[') => {
                let depth = parent_depth + 1;
                if depth > 127 {
                    return Err(Error::msg("invalid JSON"));
                }
                struct ArrayVisitor {
                    depth: usize,
                }
                impl<'de> Visitor<'de> for ArrayVisitor {
                    type Value = OrderedValue;
                    fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        f.write_str("a JSON array")
                    }
                    fn visit_seq<A: SeqAccess<'de>>(
                        self,
                        mut seq: A,
                    ) -> Result<Self::Value, A::Error> {
                        let mut entries = Vec::new();
                        while let Some(raw) = seq.next_element::<Box<RawValue>>()? {
                            let value = OrderedValue::from_raw(raw.get(), self.depth)
                                .map_err(A::Error::custom)?;
                            entries.push(value);
                        }
                        Ok(OrderedValue::Array(entries))
                    }
                }
                let mut decoder = serde_json::Deserializer::from_str(raw);
                let value = decoder
                    .deserialize_seq(ArrayVisitor { depth })
                    .map_err(|_| Error::msg("invalid JSON"))?;
                decoder.end().map_err(|_| Error::msg("invalid JSON"))?;
                Ok(value)
            }
            Some(b'"') => serde_json::from_str(raw)
                .map(OrderedValue::String)
                .map_err(|_| Error::msg("invalid JSON")),
            Some(b't' | b'f') => serde_json::from_str(raw)
                .map(OrderedValue::Bool)
                .map_err(|_| Error::msg("invalid JSON")),
            Some(b'n') if raw == "null" => Ok(OrderedValue::Null),
            Some(b'-' | b'0'..=b'9') => {
                // The RawValue was validated by serde_json; retain its exact
                // token so 1, 1.0 and 1e0 remain distinct.
                RawValue::from_string(raw.to_owned())
                    .map(|_| OrderedValue::Number(raw.to_owned()))
                    .map_err(|_| Error::msg("invalid JSON"))
            }
            _ => Err(Error::msg("invalid JSON")),
        }
    }

    pub(crate) fn object(&self) -> Option<&[(String, OrderedValue)]> {
        match self {
            Self::Object(entries) => Some(entries),
            _ => None,
        }
    }

    pub(crate) fn object_mut(&mut self) -> Option<&mut Vec<(String, OrderedValue)>> {
        match self {
            Self::Object(entries) => Some(entries),
            _ => None,
        }
    }

    pub(crate) fn set_all_exact(&mut self, name: &str, value: Self) {
        if let Some(entries) = self.object_mut() {
            for (key, current) in entries.iter_mut() {
                if key == name {
                    *current = value.clone();
                }
            }
        }
    }

    pub(crate) fn last_exact(&self, name: &str) -> Option<&Self> {
        self.object()?
            .iter()
            .rev()
            .find(|(key, _)| key == name)
            .map(|(_, value)| value)
    }

    pub(crate) fn first_exact_then_folded(&self, name: &str) -> Option<&Self> {
        let entries = self.object()?;
        entries
            .iter()
            .find(|(key, _)| key == name)
            .or_else(|| {
                entries
                    .iter()
                    .find(|(key, _)| key.eq_ignore_ascii_case(name))
            })
            .map(|(_, value)| value)
    }

    pub(crate) fn string_or_empty(&self, name: &str) -> Result<String, Error> {
        match self.first_exact_then_folded(name) {
            None | Some(Self::Null) => Ok(String::new()),
            Some(Self::String(value)) => Ok(value.clone()),
            _ => Err(Error::msg(format!("missing or invalid {name}"))),
        }
    }

    pub(crate) fn prune_null_members(&mut self) {
        match self {
            Self::Object(entries) => {
                entries.retain(|(_, value)| !matches!(value, Self::Null));
                for (_, value) in entries {
                    value.prune_null_members();
                }
            }
            Self::Array(entries) => {
                for value in entries {
                    value.prune_null_members();
                }
            }
            _ => {}
        }
    }
}

impl Serialize for OrderedValue {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        match self {
            Self::Null => serializer.serialize_none(),
            Self::Bool(value) => serializer.serialize_bool(*value),
            Self::Number(raw) => {
                let value = RawValue::from_string(raw.clone()).map_err(S::Error::custom)?;
                value.serialize(serializer)
            }
            Self::String(value) => serializer.serialize_str(value),
            Self::Array(values) => {
                let mut seq = serializer.serialize_seq(Some(values.len()))?;
                for value in values {
                    seq.serialize_element(value)?;
                }
                seq.end()
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
mod depth_tests {
    use super::OrderedValue;

    #[test]
    fn ordered_values_match_serde_container_depth_limit() {
        let scalar = OrderedValue::parse("1e400").expect("scalar root");
        assert_eq!(serde_json::to_string(&scalar).unwrap(), "1e400");
        let nested = |depth: usize| format!("{}1e400{}", "[".repeat(depth), "]".repeat(depth));
        let accepted = nested(127);
        let value = OrderedValue::parse(&accepted).expect("127 containers");
        assert_eq!(serde_json::to_string(&value).unwrap(), accepted);
        let duplicate_object = format!(
            "{}{{\"first\":1e2,\"second\":-0,\"first\":1.00}}{}",
            "[".repeat(126),
            "]".repeat(126)
        );
        assert_eq!(
            serde_json::to_string(&OrderedValue::parse(&duplicate_object).unwrap()).unwrap(),
            duplicate_object
        );
        assert_eq!(
            OrderedValue::parse(&nested(128)).unwrap_err().to_string(),
            "invalid JSON"
        );
    }
}
