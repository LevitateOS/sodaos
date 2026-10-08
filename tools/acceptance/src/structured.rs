//! Owner-local representation for the acceptance callers that intentionally
//! inspect arbitrary JSON trees. Serde owns tokenization and string grammar;
//! ordered pairs and raw number tokens keep the observable dynamic-data rules.

use serde::de::{MapAccess, SeqAccess, Visitor};
use serde::ser::{SerializeMap, SerializeSeq};
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use serde_json::value::RawValue;

/// Maximum number of nested containers admitted by the pinned Serde parser.
/// The root container counts as one; scalar roots have depth zero.
const MAX_CONTAINERS: usize = 127;

struct RawEntries<'a>(Vec<(String, &'a RawValue)>);

impl<'de> Deserialize<'de> for RawEntries<'de> {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct EntriesVisitor;
        impl<'de> Visitor<'de> for EntriesVisitor {
            type Value = RawEntries<'de>;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("a JSON object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let mut entries = Vec::new();
                while let Some(key) = map.next_key::<String>()? {
                    entries.push((key, map.next_value::<&'de RawValue>()?));
                }
                Ok(RawEntries(entries))
            }
        }
        deserializer.deserialize_map(EntriesVisitor)
    }
}

struct RawItems<'a>(Vec<&'a RawValue>);

impl<'de> Deserialize<'de> for RawItems<'de> {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct ItemsVisitor;
        impl<'de> Visitor<'de> for ItemsVisitor {
            type Value = RawItems<'de>;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("a JSON array")
            }
            fn visit_seq<A: SeqAccess<'de>>(self, mut seq: A) -> Result<Self::Value, A::Error> {
                let mut items = Vec::new();
                while let Some(item) = seq.next_element::<&'de RawValue>()? {
                    items.push(item);
                }
                Ok(RawItems(items))
            }
        }
        deserializer.deserialize_seq(ItemsVisitor)
    }
}

/// A dynamic JSON value retaining object order, duplicate members, and number
/// spelling. Known wire records should continue to use typed DTOs instead.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Value {
    Null,
    Bool(bool),
    Number(String),
    Str(String),
    Array(Vec<Value>),
    Object(Vec<(String, Value)>),
}

impl Value {
    /// Parse one complete JSON value and convert it under the ordinary
    /// serde_json container-depth limit.
    pub fn parse(text: &str) -> Result<Self, serde_json::Error> {
        let raw: Box<RawValue> = serde_json::from_str(text)?;
        Self::from_raw(&raw)
    }

    /// Convert a validated raw token while preserving duplicate object names
    /// and lexical number spelling.
    pub fn from_raw(raw: &RawValue) -> Result<Self, serde_json::Error> {
        Self::from_raw_at(raw, 0)
    }

    fn from_raw_at(raw: &RawValue, depth: usize) -> Result<Self, serde_json::Error> {
        let text = raw.get();
        match text.as_bytes().first().copied() {
            Some(b'{') => {
                let next = depth + 1;
                if next > MAX_CONTAINERS {
                    return Err(<serde_json::Error as serde::de::Error>::custom(
                        "JSON nesting limit exceeded",
                    ));
                }
                let RawEntries(raw_entries): RawEntries<'_> = serde_json::from_str(text)?;
                let mut entries = Vec::with_capacity(raw_entries.len());
                for (key, value) in raw_entries {
                    entries.push((key, Self::from_raw_at(value, next)?));
                }
                Ok(Self::Object(entries))
            }
            Some(b'[') => {
                let next = depth + 1;
                if next > MAX_CONTAINERS {
                    return Err(<serde_json::Error as serde::de::Error>::custom(
                        "JSON nesting limit exceeded",
                    ));
                }
                let RawItems(raw_items): RawItems<'_> = serde_json::from_str(text)?;
                let mut items = Vec::with_capacity(raw_items.len());
                for value in raw_items {
                    items.push(Self::from_raw_at(value, next)?);
                }
                Ok(Self::Array(items))
            }
            Some(b'"') => serde_json::from_str::<String>(text).map(Self::Str),
            Some(b't') => serde_json::from_str::<bool>(text).map(Self::Bool),
            Some(b'f') => serde_json::from_str::<bool>(text).map(Self::Bool),
            Some(b'n') => {
                let _: () = serde_json::from_str(text)?;
                Ok(Self::Null)
            }
            Some(b'-' | b'0'..=b'9') => Ok(Self::Number(text.to_owned())),
            _ => Err(<serde_json::Error as serde::de::Error>::custom(
                "invalid JSON value",
            )),
        }
    }

    /// Return the last exact-name object member, matching these callers'
    /// existing duplicate lookup policy.
    pub fn get(&self, name: &str) -> Option<&Value> {
        match self {
            Self::Object(entries) => entries
                .iter()
                .rev()
                .find(|(key, _)| key == name)
                .map(|(_, value)| value),
            _ => None,
        }
    }

    pub fn as_str(&self) -> Option<&str> {
        match self {
            Self::Str(value) => Some(value),
            _ => None,
        }
    }

    pub fn as_integer(&self) -> Option<i128> {
        match self {
            Self::Number(raw) => raw.parse().ok(),
            _ => None,
        }
    }

    /// Refuse constructed trees deeper than the dynamic parser would admit.
    pub fn validate_depth(&self) -> Result<(), serde_json::Error> {
        let mut pending = vec![(self, 0usize)];
        while let Some((value, depth)) = pending.pop() {
            match value {
                Self::Array(items) => {
                    let next = depth + 1;
                    if next > MAX_CONTAINERS {
                        return Err(<serde_json::Error as serde::de::Error>::custom(
                            "JSON nesting limit exceeded",
                        ));
                    }
                    pending.extend(items.iter().map(|item| (item, next)));
                }
                Self::Object(entries) => {
                    let next = depth + 1;
                    if next > MAX_CONTAINERS {
                        return Err(<serde_json::Error as serde::de::Error>::custom(
                            "JSON nesting limit exceeded",
                        ));
                    }
                    pending.extend(entries.iter().map(|(_, item)| (item, next)));
                }
                _ => {}
            }
        }
        Ok(())
    }
}

impl Serialize for Value {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        self.validate_depth().map_err(serde::ser::Error::custom)?;
        serialize_value(self, serializer)
    }
}

struct NestedValue<'a>(&'a Value);

impl Serialize for NestedValue<'_> {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serialize_value(self.0, serializer)
    }
}

fn serialize_value<S: Serializer>(value: &Value, serializer: S) -> Result<S::Ok, S::Error> {
    match value {
        Value::Null => serializer.serialize_none(),
        Value::Bool(value) => serializer.serialize_bool(*value),
        Value::Number(raw) => {
            let number = RawValue::from_string(raw.clone()).map_err(serde::ser::Error::custom)?;
            number.serialize(serializer)
        }
        Value::Str(value) => serializer.serialize_str(value),
        Value::Array(items) => {
            let mut seq = serializer.serialize_seq(Some(items.len()))?;
            for item in items {
                seq.serialize_element(&NestedValue(item))?;
            }
            seq.end()
        }
        Value::Object(entries) => {
            let mut map = serializer.serialize_map(Some(entries.len()))?;
            for (key, value) in entries {
                map.serialize_entry(key, &NestedValue(value))?;
            }
            map.end()
        }
    }
}

#[cfg(test)]
mod tests {
    use super::Value;

    #[test]
    fn dynamic_tree_keeps_duplicates_order_and_raw_numbers() {
        let value = Value::parse(r#"{"z":1e2,"a":-0,"z":1e400}"#).unwrap();
        let Value::Object(entries) = value else {
            panic!()
        };
        assert_eq!(entries.len(), 3);
        assert_eq!(entries[0].0, "z");
        assert_eq!(entries[1].1, Value::Number("-0".into()));
        assert_eq!(entries[2].1, Value::Number("1e400".into()));
    }

    #[test]
    fn dynamic_tree_uses_pinned_serde_container_depth() {
        let nested = |count: usize| format!("{}0{}", "[".repeat(count), "]".repeat(count));
        assert!(Value::parse(&nested(127)).is_ok());
        assert!(Value::parse(&nested(128)).is_err());
        assert_eq!(Value::parse("0").unwrap(), Value::Number("0".into()));
    }
}
