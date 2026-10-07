//! Standard-alphabet base64 (encode + Python `validate=True` decode).

use base64::engine::general_purpose::{GeneralPurpose, GeneralPurposeConfig};
use base64::engine::DecodePaddingMode;
use base64::{alphabet, Engine};

fn python_validate() -> GeneralPurpose {
    GeneralPurpose::new(
        &alphabet::STANDARD,
        GeneralPurposeConfig::new()
            .with_decode_padding_mode(DecodePaddingMode::RequireCanonical)
            .with_decode_allow_trailing_bits(true),
    )
}

pub fn decode(input: &str) -> Option<Vec<u8>> {
    python_validate().decode(input).ok()
}

pub fn encode(input: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD.encode(input)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn vectors() {
        assert_eq!(decode("YWI=").unwrap(), b"ab");
        assert_eq!(decode("YQ==").unwrap(), b"a");
        assert_eq!(decode("TWFu").unwrap(), b"Man");
        assert_eq!(decode("").unwrap(), b"");
        // Non-canonical trailing bits accepted like binascii.
        assert_eq!(decode("YR==").unwrap(), b"a");
        assert_eq!(decode("YWE=").unwrap(), b"aa");
        // Padding placement.
        assert!(decode("YW=J").is_none());
        assert!(decode("====").is_none());
        assert!(decode("abc").is_none());
        assert!(decode("ab=d").is_none());
        assert!(decode("a===").is_none());
        // Whitespace rejected under validate=True.
        assert!(decode("TW\nFu").is_none());
        assert!(decode("TWFu\n").is_none());
        // Round trip incl. credential sizes.
        for len in [0, 1, 2, 3, 55, 256] {
            let data: Vec<u8> = (0..len).map(|i| (i * 37 + 11) as u8).collect();
            assert_eq!(decode(&encode(&data)).unwrap(), data);
        }
    }
}
