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
    use base64::engine::general_purpose::{GeneralPurpose, GeneralPurposeConfig};
    use base64::engine::DecodePaddingMode;
    use base64::{alphabet, DecodeError, Engine};

    fn go_std() -> GeneralPurpose {
        GeneralPurpose::new(
            &alphabet::STANDARD,
            GeneralPurposeConfig::new()
                .with_decode_padding_mode(DecodePaddingMode::RequireCanonical)
                .with_decode_allow_trailing_bits(true),
        )
    }

    pub fn encode(data: &[u8]) -> String {
        base64::engine::general_purpose::STANDARD.encode(data)
    }

    pub fn decode(text: &str) -> Result<Vec<u8>, String> {
        if !text.len().is_multiple_of(4) {
            return Err("invalid base64 length".to_string());
        }
        go_std().decode(text).map_err(|error| match error {
            DecodeError::InvalidByte(_, b'=') | DecodeError::InvalidPadding => {
                "invalid base64 padding".to_string()
            }
            DecodeError::InvalidByte(_, byte) => format!("invalid base64 byte {byte:?}"),
            DecodeError::InvalidLength(_) | DecodeError::InvalidLastSymbol(_, _) => {
                "invalid base64 length".to_string()
            }
        })
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
