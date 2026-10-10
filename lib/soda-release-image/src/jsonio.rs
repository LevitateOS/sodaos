//! Serde adapters for release-image JSON input and ordered object serialization.

use serde::de::{self, DeserializeOwned, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};

/// Preserve JSON object member order for domain maps whose caller observes
/// first-match or duplicate behavior.
pub(crate) struct OrderedMap<T>(pub Vec<(String, T)>);

impl<T> Default for OrderedMap<T> {
    fn default() -> Self {
        Self(Vec::new())
    }
}

/// Serialize a sorted-key view while retaining duplicate source pairs and
/// their stable order for equal keys.
pub(crate) struct SortedPairs<'a, T>(pub &'a [(String, T)]);

impl<T: Serialize> Serialize for SortedPairs<'_, T> {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        use serde::ser::SerializeMap;
        let mut entries: Vec<_> = self.0.iter().collect();
        entries.sort_by(|left, right| left.0.cmp(&right.0));
        let mut map = serializer.serialize_map(Some(entries.len()))?;
        for (key, value) in entries {
            map.serialize_entry(key, value)?;
        }
        map.end()
    }
}

impl<'de, T: Deserialize<'de>> Deserialize<'de> for OrderedMap<T> {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct OrderedMapVisitor<T>(std::marker::PhantomData<T>);
        impl<'de, T: Deserialize<'de>> Visitor<'de> for OrderedMapVisitor<T> {
            type Value = OrderedMap<T>;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("an object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let mut entries = Vec::new();
                while let Some((key, value)) = map.next_entry::<String, T>()? {
                    entries.push((key, value));
                }
                Ok(OrderedMap(entries))
            }
        }
        deserializer.deserialize_map(OrderedMapVisitor(std::marker::PhantomData))
    }
}

use crate::error::Error;

/// Deserialize one complete typed record or arbitrary JSON value.
pub fn parse<T: DeserializeOwned>(text: &str) -> Result<T, Error> {
    let mut decoder = serde_json::Deserializer::from_str(text);
    let value = T::deserialize(&mut decoder).map_err(|_| Error::msg("invalid JSON"))?;
    decoder.end().map_err(|_| Error::msg("invalid JSON"))?;
    Ok(value)
}

pub(crate) fn decode_field<T: DeserializeOwned + Default, E: de::Error>(
    raw: Option<Box<serde_json::value::RawValue>>,
) -> Result<T, E> {
    match raw {
        None => Ok(T::default()),
        // Go's strconv integer readers accept the JSON integer token -0 as
        // zero. Serde's signed/unsigned integer visitors reject that spelling.
        Some(raw) => {
            serde_json::from_str::<Option<T>>(if raw.get() == "-0" { "0" } else { raw.get() })
                .map(Option::unwrap_or_default)
                .map_err(E::custom)
        }
    }
}

/// Generates one concrete strict DTO visitor. Each DTO keeps its own field
/// list and types; raw slots implement the image owner's exact-first, then
/// first-folded lookup before typed conversion.
macro_rules! case_record {
    ($ty:ident, { $($field:ident : $field_type:ty => $name:literal),+ $(,)? }) => {
        $crate::jsonio::case_record!(@impl strict, $ty, { $($field : $field_type => $name),+ });
    };
    ($ty:ident, ignore_unknown, { $($field:ident : $field_type:ty => $name:literal),+ $(,)? }) => {
        $crate::jsonio::case_record!(@impl ignore_unknown, $ty, { $($field : $field_type => $name),+ });
    };
    (@unknown strict, $key:ident, $map:ident) => {
        return Err(serde::de::Error::custom(format!("unknown field {:?}", $key)));
    };
    (@unknown ignore_unknown, $key:ident, $map:ident) => {
        $map.next_value::<serde::de::IgnoredAny>()?;
    };
    (@impl $unknown:ident, $ty:ident, { $($field:ident : $field_type:ty => $name:literal),+ $(,)? }) => {
        impl<'de> serde::Deserialize<'de> for $ty {
            fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
                struct RecordVisitor;
                impl<'de> serde::de::Visitor<'de> for RecordVisitor {
                    type Value = $ty;
                    fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        f.write_str("known Soda JSON record")
                    }
                    fn visit_map<A: serde::de::MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                        $(let mut $field: (Option<Box<serde_json::value::RawValue>>, Option<Box<serde_json::value::RawValue>>) = (None, None);)+
                        while let Some(key) = map.next_key::<String>()? {
                            let mut matched = false;
                            $(if key == $name {
                                let raw = map.next_value::<Box<serde_json::value::RawValue>>()?;
                                if $field.0.is_none() { $field.0 = Some(raw); }
                                matched = true;
                            } else if key.eq_ignore_ascii_case($name) {
                                let raw = map.next_value::<Box<serde_json::value::RawValue>>()?;
                                if $field.1.is_none() { $field.1 = Some(raw); }
                                matched = true;
                            })+
                            if !matched {
                                $crate::jsonio::case_record!(@unknown $unknown, key, map);
                            }
                        }
                        Ok($ty { $($field: $crate::jsonio::decode_field::<$field_type, A::Error>($field.0.or($field.1))?,)+ })
                    }
                }
                deserializer.deserialize_map(RecordVisitor)
            }
        }
    };
}

pub(crate) use case_record;
