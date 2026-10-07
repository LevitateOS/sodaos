//! Standard-alphabet base64 (`validate=True` decode) for factory files.

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
        assert!(decode("YW=J").is_none());
        assert!(decode("====").is_none());
        assert!(decode("abc").is_none());
        assert!(decode("ab=d").is_none());
        assert!(decode("a===").is_none());
        assert!(decode("!!!").is_none());
        assert!(decode("!!!!").is_none());
        assert!(decode("TW\nFu").is_none());
        assert!(decode("TWFu\n").is_none());
    }
}
