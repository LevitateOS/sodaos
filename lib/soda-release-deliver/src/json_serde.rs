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
                    map.next_value_seed(UniqueSeed { depth: 0 })?;
                }
                Ok(())
            }
        }
        d.deserialize_map(RootVisitor)
    }
}

struct UniqueSeed {
    depth: u32,
}
impl<'de> DeserializeSeed<'de> for UniqueSeed {
    type Value = ();
    fn deserialize<D: Deserializer<'de>>(self, d: D) -> Result<(), D::Error> {
        if self.depth > 100 {
            return Err(serde::de::Error::custom("maximum JSON depth exceeded"));
        }
        struct AnyVisitor {
            depth: u32,
        }
        impl<'de> Visitor<'de> for AnyVisitor {
            type Value = ();
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a JSON value")
            }
            fn visit_bool<E: serde::de::Error>(self, _: bool) -> Result<(), E> {
                Ok(())
            }
            fn visit_i64<E: serde::de::Error>(self, _: i64) -> Result<(), E> {
                Ok(())
            }
            fn visit_u64<E: serde::de::Error>(self, _: u64) -> Result<(), E> {
                Ok(())
            }
            fn visit_i128<E: serde::de::Error>(self, _: i128) -> Result<(), E> {
                Ok(())
            }
            fn visit_u128<E: serde::de::Error>(self, _: u128) -> Result<(), E> {
                Ok(())
            }
            fn visit_f64<E: serde::de::Error>(self, _: f64) -> Result<(), E> {
                Ok(())
            }
            fn visit_str<E: serde::de::Error>(self, _: &str) -> Result<(), E> {
                Ok(())
            }
            fn visit_string<E: serde::de::Error>(self, _: String) -> Result<(), E> {
                Ok(())
            }
            fn visit_unit<E: serde::de::Error>(self) -> Result<(), E> {
                Ok(())
            }
            fn visit_none<E: serde::de::Error>(self) -> Result<(), E> {
                Ok(())
            }
            fn visit_seq<S: SeqAccess<'de>>(self, mut seq: S) -> Result<(), S::Error> {
                while seq
                    .next_element_seed(UniqueSeed {
                        depth: self.depth + 1,
                    })?
                    .is_some()
                {}
                Ok(())
            }
            fn visit_map<M: MapAccess<'de>>(self, mut map: M) -> Result<(), M::Error> {
                let mut names = HashSet::new();
                while let Some(name) = map.next_key::<String>()? {
                    if !names.insert(name) {
                        return Err(serde::de::Error::custom("duplicate object member"));
                    }
                    map.next_value_seed(UniqueSeed {
                        depth: self.depth + 1,
                    })?;
                }
                Ok(())
            }
        }
        d.deserialize_any(AnyVisitor { depth: self.depth })
    }
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

    #[test]
    fn strict_admission_checks_decoded_duplicates_depth_and_full_input() {
        assert!(strict::<serde_json::Value>(br#"{"unknown":[{"\u0061":1,"a":2}]}"#).is_err());
        assert!(strict::<serde_json::Value>(br#"{"a":1} {"b":2}"#).is_err());
        let at_limit = format!("{{\"v\":{}}}", "[".repeat(100) + "0" + &"]".repeat(100));
        let over_limit = format!("{{\"v\":{}}}", "[".repeat(101) + "0" + &"]".repeat(101));
        assert!(strict::<serde_json::Value>(at_limit.as_bytes()).is_ok());
        assert!(strict::<serde_json::Value>(over_limit.as_bytes()).is_err());
    }
}
