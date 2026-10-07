use serde::de::{DeserializeOwned, DeserializeSeed, MapAccess, SeqAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::collections::HashSet;
use std::fmt;

use crate::Error;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct DecodeError;

pub(crate) const MAX_STRICT_BYTES: usize = 1 << 20;

/// Validate the strict root and decoded duplicate names before Serde can
/// collapse object members, then deserialize the owner DTO from the bytes.
pub(crate) fn strict<T: DeserializeOwned>(data: &[u8]) -> Result<T, Error> {
    if data.len() > MAX_STRICT_BYTES || std::str::from_utf8(data).is_err() {
        return Err(Error::refused());
    }
    let mut scan = serde_json::Deserializer::from_slice(data);
    RootSeed
        .deserialize(&mut scan)
        .map_err(|_| Error::refused())?;
    scan.end().map_err(|_| Error::refused())?;

    let mut decode = serde_json::Deserializer::from_slice(data);
    let value = T::deserialize(&mut decode).map_err(|_| Error::refused())?;
    decode.end().map_err(|_| Error::refused())?;
    Ok(value)
}

struct RootSeed;
impl<'de> DeserializeSeed<'de> for RootSeed {
    type Value = ();
    fn deserialize<D: Deserializer<'de>>(self, d: D) -> Result<(), D::Error> {
        struct RootVisitor;
        impl<'de> Visitor<'de> for RootVisitor {
            type Value = ();
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a JSON object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut map: M) -> Result<(), M::Error> {
                let mut names = HashSet::new();
                while let Some(name) = map.next_key::<String>()? {
                    if !names.insert(name) {
                        return Err(serde::de::Error::custom("duplicate object member"));
                    }
                    let raw = map.next_value::<Box<serde_json::value::RawValue>>()?;
                    check_raw_value::<M::Error>(&raw, 1)?;
                }
                Ok(())
            }
        }
        d.deserialize_map(RootVisitor)
    }
}

struct RawObject(Vec<(String, Box<serde_json::value::RawValue>)>);
impl<'de> Deserialize<'de> for RawObject {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawObject;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a JSON object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut map: M) -> Result<RawObject, M::Error> {
                let mut entries = Vec::new();
                while let Some(name) = map.next_key::<String>()? {
                    entries.push((name, map.next_value::<Box<serde_json::value::RawValue>>()?));
                }
                Ok(RawObject(entries))
            }
        }
        d.deserialize_map(V)
    }
}

struct RawArray(Vec<Box<serde_json::value::RawValue>>);
impl<'de> Deserialize<'de> for RawArray {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawArray;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a JSON array")
            }
            fn visit_seq<S: SeqAccess<'de>>(self, mut seq: S) -> Result<RawArray, S::Error> {
                let mut values = Vec::new();
                while let Some(value) = seq.next_element::<Box<serde_json::value::RawValue>>()? {
                    values.push(value);
                }
                Ok(RawArray(values))
            }
        }
        d.deserialize_seq(V)
    }
}

fn check_raw_value<E: serde::de::Error>(
    raw: &serde_json::value::RawValue,
    depth: u32,
) -> Result<(), E> {
    if depth > 100 {
        return Err(E::custom("maximum JSON depth exceeded"));
    }
    // RawValue already validated the whole token; only containers need a
    // structural walk here, so numeric leaves never pass through f64.
    let first = raw.get().as_bytes().first().copied();
    match first {
        Some(b'{') => {
            let RawObject(entries) = serde_json::from_str(raw.get()).map_err(E::custom)?;
            let mut names = HashSet::new();
            for (name, value) in entries {
                if !names.insert(name) {
                    return Err(E::custom("duplicate object member"));
                }
                check_raw_value::<E>(&value, depth + 1)?;
            }
        }
        Some(b'[') => {
            let RawArray(values) = serde_json::from_str(raw.get()).map_err(E::custom)?;
            for value in values {
                check_raw_value::<E>(&value, depth + 1)?;
            }
        }
        _ => {}
    }
    Ok(())
}

pub(crate) fn null_default<'de, D, T>(d: D) -> Result<T, D::Error>
where
    D: Deserializer<'de>,
    T: Deserialize<'de> + Default,
{
    Ok(Option::<T>::deserialize(d)?.unwrap_or_default())
}

pub(crate) fn null_i64<'de, D: Deserializer<'de>>(d: D) -> Result<i64, D::Error> {
    use serde::de::Error as _;
    let raw = Option::<Box<serde_json::value::RawValue>>::deserialize(d)?;
    match raw {
        None => Ok(0),
        Some(raw) => parse_integer(raw.get())
            .and_then(|n| n.parse::<i64>().ok())
            .ok_or_else(|| D::Error::custom("invalid signed integer")),
    }
}

pub(crate) fn null_u64<'de, D: Deserializer<'de>>(d: D) -> Result<u64, D::Error> {
    use serde::de::Error as _;
    let raw = Option::<Box<serde_json::value::RawValue>>::deserialize(d)?;
    match raw {
        None => Ok(0),
        Some(raw) => {
            let lexeme = parse_integer(raw.get())
                .ok_or_else(|| D::Error::custom("invalid unsigned integer"))?;
            if lexeme == "-0" {
                return Ok(0);
            }
            lexeme
                .parse::<u64>()
                .map_err(|_| D::Error::custom("invalid unsigned integer"))
        }
    }
}

pub(crate) fn null_u64_map<'de, D: Deserializer<'de>>(
    d: D,
) -> Result<std::collections::BTreeMap<String, u64>, D::Error> {
    use serde::de::Error as _;
    let raw = Option::<std::collections::BTreeMap<String, Box<serde_json::value::RawValue>>>::deserialize(d)?;
    let mut out = std::collections::BTreeMap::new();
    for (key, value) in raw.unwrap_or_default() {
        let number = if value.get() == "null" {
            0
        } else {
            let lexeme = parse_integer(value.get())
                .ok_or_else(|| D::Error::custom("invalid unsigned integer"))?;
            if lexeme == "-0" {
                0
            } else {
                lexeme
                    .parse::<u64>()
                    .map_err(|_| D::Error::custom("invalid unsigned integer"))?
            }
        };
        out.insert(key, number);
    }
    Ok(out)
}

fn parse_integer(token: &str) -> Option<&str> {
    let digits = token.strip_prefix('-').unwrap_or(token);
    if digits.is_empty() || !digits.bytes().all(|b| b.is_ascii_digit()) {
        return None;
    }
    Some(token)
}

#[cfg(test)]
mod tests {
    use super::strict;
    use serde::Deserialize;

    #[derive(Deserialize)]
    struct RawDocument {
        value: Box<serde_json::value::RawValue>,
    }

    #[test]
    fn strict_raw_values_preserve_numbers_and_reject_duplicates_and_depth() {
        let doc: RawDocument =
            strict(br#"{"value":{"z":1e2,"negative_zero":-0,"wide":1e400}}"#).unwrap();
        assert_eq!(
            doc.value.get(),
            r#"{"z":1e2,"negative_zero":-0,"wide":1e400}"#
        );
        assert!(strict::<RawDocument>(br#"{"value":{"nested":{"\u0061":1,"a":2}}}"#).is_err());
        assert!(strict::<RawDocument>(br#"{"value":1} {"value":2}"#).is_err());

        // Root object is depth 0; its value starts at depth 1. A scalar at
        // depth 100 is accepted and one at depth 101 is refused.
        let at_limit = format!("{{\"value\":{}}}", "[".repeat(99) + "0" + &"]".repeat(99));
        let over_limit = format!("{{\"value\":{}}}", "[".repeat(100) + "0" + &"]".repeat(100));
        assert!(strict::<RawDocument>(at_limit.as_bytes()).is_ok());
        assert!(strict::<RawDocument>(over_limit.as_bytes()).is_err());
    }
}
