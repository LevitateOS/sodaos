use base64::engine::DecodePaddingMode;
use base64::Engine;

fn go_std_decode(input: &[u8]) -> Result<Vec<u8>, ()> {
    let compact: Vec<u8> = input
        .iter()
        .copied()
        .filter(|byte| *byte != b'\r' && *byte != b'\n')
        .collect();
    base64::engine::general_purpose::GeneralPurpose::new(
        &base64::alphabet::STANDARD,
        base64::engine::general_purpose::GeneralPurposeConfig::new()
            .with_decode_padding_mode(DecodePaddingMode::RequireCanonical)
            .with_decode_allow_trailing_bits(true),
    )
    .decode(compact)
    .map_err(|_| ())
}

pub fn b64_decode_go(input: &[u8]) -> Result<Vec<u8>, ()> {
    go_std_decode(input)
}

pub fn b64_encode(raw: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD.encode(raw)
}

pub fn b64_encode_raw(raw: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD_NO_PAD.encode(raw)
}
