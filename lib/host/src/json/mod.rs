//! Host JSON admission and Go-compatible output string formatting.
//!
//! Concrete request and response shapes live in their owning modules as
//! Serde DTOs. This module supplies only strict recursive admission, the
//! signed JSON integer token adapter, byte-field adapter, and output quoting.

use std::collections::{BTreeMap, HashSet};

use serde::de::{self, DeserializeOwned, DeserializeSeed, MapAccess, SeqAccess, Visitor};
use serde::{Deserialize, Deserializer};

mod number;
#[cfg(test)]
mod strict_tests;

pub use self::number::{parse_go_int64, parse_go_uint32};

pub const MAXIMUM_REQUEST_BYTES: usize = 1 << 20;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Error(pub String);

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.0)
    }
}
impl std::error::Error for Error {}
fn err(message: impl Into<String>) -> Error {
    Error(message.into())
}

/// Signed integer spelling adapter. Go's integer binder accepts JSON `-0`;
/// native integer visitors in serde_json classify that token as floating point.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SignedInteger(pub i64);
impl From<SignedInteger> for i64 {
    fn from(value: SignedInteger) -> Self {
        value.0
    }
}
impl<'de> Deserialize<'de> for SignedInteger {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        let raw = Box::<serde_json::value::RawValue>::deserialize(deserializer)?;
        let token = raw.get();
        let bytes = token.as_bytes();
        let digits = if bytes.first() == Some(&b'-') {
            &bytes[1..]
        } else {
            bytes
        };
        let integer = match digits.first() {
            Some(b'0') => digits.len() == 1,
            Some(b'1'..=b'9') => digits.iter().all(u8::is_ascii_digit),
            _ => false,
        };
        if !integer {
            return Err(de::Error::custom("expected a signed integer"));
        }
        token
            .parse::<i64>()
            .map(SignedInteger)
            .map_err(de::Error::custom)
    }
}

/// Go `[]uint8` JSON binding: base64 strings, numeric arrays, null bytes as
/// zero, and null/missing slices as empty at the containing DTO.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct BytesField(pub Vec<u8>);
impl<'de> Deserialize<'de> for BytesField {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = BytesField;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("base64 text or an unsigned byte array")
            }
            fn visit_str<E>(self, value: &str) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                crate::ssh::b64_decode_go(value.as_bytes())
                    .map(BytesField)
                    .map_err(E::custom)
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
                Ok(BytesField(Vec::new()))
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                let mut bytes = Vec::new();
                while let Some(value) = seq.next_element::<Option<u64>>()? {
                    let byte = value.unwrap_or(0);
                    if byte > u8::MAX as u64 {
                        return Err(de::Error::custom("byte out of range"));
                    }
                    bytes.push(byte as u8);
                }
                Ok(BytesField(bytes))
            }
        }
        deserializer.deserialize_any(V)
    }
}

/// Strictly admit a complete root object and deserialize it into its owner
/// DTO. Root names are sorted before typed decoding; nested RawValue bytes keep
/// source order for alias selection.
pub fn decode_strict_as<T: DeserializeOwned>(body: &[u8]) -> Result<T, Error> {
    if body.len() > MAXIMUM_REQUEST_BYTES {
        return Err(err("request exceeds 1 MiB"));
    }
    std::str::from_utf8(body).map_err(|_| err("request must contain valid UTF-8"))?;

    struct Root(BTreeMap<String, Box<serde_json::value::RawValue>>);
    impl<'de> Deserialize<'de> for Root {
        fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
        where
            D: Deserializer<'de>,
        {
            struct V;
            impl<'de> Visitor<'de> for V {
                type Value = Root;
                fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                    f.write_str("a JSON object")
                }
                fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
                where
                    A: MapAccess<'de>,
                {
                    let mut members = BTreeMap::new();
                    while let Some(key) = map.next_key::<String>()? {
                        if members.contains_key(&key) {
                            return Err(de::Error::custom("duplicate object key"));
                        }
                        members.insert(key, map.next_value::<Box<serde_json::value::RawValue>>()?);
                    }
                    Ok(Root(members))
                }
            }
            deserializer.deserialize_map(V)
        }
    }

    struct UniqueSeed(usize);
    impl<'de> DeserializeSeed<'de> for UniqueSeed {
        type Value = ();
        fn deserialize<D>(self, deserializer: D) -> Result<(), D::Error>
        where
            D: Deserializer<'de>,
        {
            if self.0 > 100 {
                return Err(de::Error::custom("maximum host JSON depth exceeded"));
            }
            deserializer.deserialize_any(UniqueVisitor(self.0))
        }
    }
    struct UniqueVisitor(usize);
    impl<'de> Visitor<'de> for UniqueVisitor {
        type Value = ();
        fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
            f.write_str("a JSON value")
        }
        fn visit_bool<E>(self, _: bool) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_i64<E>(self, _: i64) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_u64<E>(self, _: u64) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_f64<E>(self, _: f64) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_str<E>(self, _: &str) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_borrowed_str<E>(self, _: &'de str) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_string<E>(self, _: String) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_unit<E>(self) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_none<E>(self) -> Result<(), E>
        where
            E: de::Error,
        {
            Ok(())
        }
        fn visit_seq<A>(self, mut seq: A) -> Result<(), A::Error>
        where
            A: SeqAccess<'de>,
        {
            while seq.next_element_seed(UniqueSeed(self.0 + 1))?.is_some() {}
            Ok(())
        }
        fn visit_map<A>(self, mut map: A) -> Result<(), A::Error>
        where
            A: MapAccess<'de>,
        {
            let mut keys = HashSet::new();
            while let Some(key) = map.next_key::<String>()? {
                if !keys.insert(key) {
                    return Err(de::Error::custom("duplicate object key"));
                }
                map.next_value_seed(UniqueSeed(self.0 + 1))?;
            }
            Ok(())
        }
    }

    let mut original = serde_json::Deserializer::from_slice(body);
    let Root(root) = Root::deserialize(&mut original).map_err(|e| err(e.to_string()))?;
    original.end().map_err(|e| err(e.to_string()))?;
    for raw in root.values() {
        let mut value = serde_json::Deserializer::from_str(raw.get());
        UniqueSeed(0)
            .deserialize(&mut value)
            .map_err(|e| err(e.to_string()))?;
        value.end().map_err(|e| err(e.to_string()))?;
    }
    let ordered = serde_json::to_vec(&root).map_err(|e| err(e.to_string()))?;
    let mut input = serde_json::Deserializer::from_slice(&ordered);
    let decoded = T::deserialize(&mut input).map_err(|e| err(e.to_string()))?;
    input.end().map_err(|e| err(e.to_string()))?;
    Ok(decoded)
}

/// Decode one complete machine response into its owner DTO.
pub fn decode_tolerant_as<T: DeserializeOwned>(body: &[u8]) -> Result<T, Error> {
    let mut input = serde_json::Deserializer::from_slice(body);
    let value = T::deserialize(&mut input).map_err(|e| err(e.to_string()))?;
    input.end().map_err(|e| err(e.to_string()))?;
    Ok(value)
}

/// Go `encoding/json` string encoding, including HTML and JS-separator
/// escapes. Field order and omission remain the responsibility of each DTO.
pub fn quote(s: &str) -> String {
    use serde::Serialize;
    let mut out = Vec::with_capacity(s.len() + 2);
    let mut serializer = serde_json::Serializer::with_formatter(&mut out, GoFormatter);
    s.serialize(&mut serializer)
        .expect("writing to Vec cannot fail");
    String::from_utf8(out).expect("JSON string is UTF-8")
}

struct GoFormatter;
impl serde_json::ser::Formatter for GoFormatter {
    fn write_string_fragment<W>(&mut self, writer: &mut W, fragment: &str) -> std::io::Result<()>
    where
        W: ?Sized + std::io::Write,
    {
        let mut start = 0;
        for (index, ch) in fragment.char_indices() {
            let escaped = match ch {
                '<' => Some(b"\\u003c".as_slice()),
                '>' => Some(b"\\u003e".as_slice()),
                '&' => Some(b"\\u0026".as_slice()),
                '\u{2028}' => Some(b"\\u2028".as_slice()),
                '\u{2029}' => Some(b"\\u2029".as_slice()),
                _ => None,
            };
            if let Some(escaped) = escaped {
                writer.write_all(&fragment.as_bytes()[start..index])?;
                writer.write_all(escaped)?;
                start = index + ch.len_utf8();
            }
        }
        writer.write_all(&fragment.as_bytes()[start..])
    }
}
