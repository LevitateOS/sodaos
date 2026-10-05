// Scalar codecs and shared validation helpers, extracted from wire.rs (A05.M).
use crate::wire_time::UnixTime;
use serde::{Deserialize, Deserializer, Serializer};
/// Go `,string` integer form: encoded as a JSON string of digits.
/// A JSON null decodes to zero like encoding/json's scalar null no-op.
pub mod i64_string {
    use super::*;

    pub fn serialize<S: Serializer>(value: &i64, ser: S) -> Result<S::Ok, S::Error> {
        ser.serialize_str(&value.to_string())
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(de: D) -> Result<i64, D::Error> {
        let text = Option::<String>::deserialize(de)?;
        match text {
            None => Ok(0),
            Some(text) => text.parse::<i64>().map_err(serde::de::Error::custom),
        }
    }
}

/// Go `,string,omitempty` integer form: omitted when zero.
pub mod i64_string_omitted {
    use super::*;

    pub fn serialize<S: Serializer>(value: &i64, ser: S) -> Result<S::Ok, S::Error> {
        ser.serialize_str(&value.to_string())
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(de: D) -> Result<i64, D::Error> {
        i64_string::deserialize(de)
    }
}

/// Null-tolerant scalar decoding: encoding/json leaves a scalar field
/// unchanged on JSON null instead of failing, so every scalar below
/// accepts null as its zero value.
pub mod null_tolerant {
    use super::*;

    pub fn string<'de, D: Deserializer<'de>>(de: D) -> Result<String, D::Error> {
        Ok(Option::<String>::deserialize(de)?.unwrap_or_default())
    }

    pub fn boolean<'de, D: Deserializer<'de>>(de: D) -> Result<bool, D::Error> {
        Ok(Option::<bool>::deserialize(de)?.unwrap_or_default())
    }

    pub fn integer<'de, D: Deserializer<'de>>(de: D) -> Result<i64, D::Error> {
        Ok(Option::<i64>::deserialize(de)?.unwrap_or_default())
    }

    pub fn integer32<'de, D: Deserializer<'de>>(de: D) -> Result<i32, D::Error> {
        Ok(Option::<i32>::deserialize(de)?.unwrap_or_default())
    }

    pub fn time<'de, D: Deserializer<'de>>(de: D) -> Result<UnixTime, D::Error> {
        Ok(Option::<UnixTime>::deserialize(de)?.unwrap_or(UnixTime { sec: 0, nanos: 0 }))
    }
}

pub fn is_zero(value: &i64) -> bool {
    *value == 0
}

/// Go []byte JSON form: standard padded base64.
pub mod base64_bytes {
    use super::*;

    const ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

    pub fn encode(data: &[u8]) -> String {
        let mut out = String::with_capacity(data.len().div_ceil(3) * 4);
        for chunk in data.chunks(3) {
            let n = chunk.len();
            let mut v: u32 = 0;
            for &b in chunk {
                v = (v << 8) | b as u32;
            }
            v <<= 8 * (3 - n);
            for i in 0..4 {
                if i <= n {
                    out.push(ALPHABET[((v >> (18 - i * 6)) & 63) as usize] as char);
                } else {
                    out.push('=');
                }
            }
        }
        out
    }

    fn decode_value(b: u8) -> Result<u8, String> {
        match b {
            b'A'..=b'Z' => Ok(b - b'A'),
            b'a'..=b'z' => Ok(b - b'a' + 26),
            b'0'..=b'9' => Ok(b - b'0' + 52),
            b'+' => Ok(62),
            b'/' => Ok(63),
            _ => Err(format!("invalid base64 byte {b:?}")),
        }
    }

    pub fn decode(text: &str) -> Result<Vec<u8>, String> {
        let bytes = text.as_bytes();
        if !bytes.len().is_multiple_of(4) {
            return Err("invalid base64 length".to_string());
        }
        let mut out = Vec::with_capacity(bytes.len() / 4 * 3);
        for chunk in bytes.chunks(4) {
            // Padding rules mirror encoding/base64 StdEncoding strictly:
            // at most two trailing `=`, and no data after padding.
            let pad = chunk.iter().rev().take_while(|b| **b == b'=').count();
            if pad > 2 || chunk[..4 - pad].contains(&b'=') {
                return Err("invalid base64 padding".to_string());
            }
            let mut v: u32 = 0;
            for &b in &chunk[..4 - pad] {
                v = (v << 6) | decode_value(b)? as u32;
            }
            v <<= 6 * pad;
            let word = v.to_be_bytes();
            out.extend_from_slice(&word[1..4 - pad]);
        }
        Ok(out)
    }

    #[allow(clippy::ptr_arg)]
    pub fn serialize<S: Serializer>(value: &Vec<u8>, ser: S) -> Result<S::Ok, S::Error> {
        ser.serialize_str(&encode(value))
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(de: D) -> Result<Vec<u8>, D::Error> {
        // encoding/json emits null for a nil slice; accept it as empty.
        let text = Option::<String>::deserialize(de)?.unwrap_or_default();
        decode(&text).map_err(serde::de::Error::custom)
    }
}

pub fn is_zero_i32(value: &i32) -> bool {
    *value == 0
}

/// `[]byte` with `omitempty`: absent when empty, base64 when present. Go
/// omits both nil and zero-length; `Some(vec![])` never serializes because
/// the broker only sets credentials from non-empty input.
pub mod base64_bytes_option {
    use super::*;

    pub fn serialize<S: Serializer>(value: &Option<Vec<u8>>, ser: S) -> Result<S::Ok, S::Error> {
        match value {
            Some(data) => ser.serialize_str(&base64_bytes::encode(data)),
            None => ser.serialize_none(),
        }
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(de: D) -> Result<Option<Vec<u8>>, D::Error> {
        let text = String::deserialize(de)?;
        base64_bytes::decode(&text)
            .map(Some)
            .map_err(serde::de::Error::custom)
    }
}
