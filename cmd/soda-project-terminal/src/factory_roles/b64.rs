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
