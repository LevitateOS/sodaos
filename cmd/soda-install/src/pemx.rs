//! Strict one-certificate PEM framing for the installer's local CA input.

const BEGIN: &[u8] = b"-----BEGIN CERTIFICATE-----";
const MAX_INPUT: usize = 16 * 1024;

pub fn decode_certificate(input: &[u8]) -> Result<Vec<u8>, ()> {
    if input.len() > MAX_INPUT {
        return Err(());
    }
    let start = input
        .iter()
        .position(|byte| !byte.is_ascii_whitespace())
        .ok_or(())?;
    let end = input
        .iter()
        .rposition(|byte| !byte.is_ascii_whitespace())
        .ok_or(())?
        + 1;
    let input = &input[start..end];
    if !input.starts_with(BEGIN) {
        return Err(());
    }
    let (label, der) = x509_cert::der::pem::decode_vec(input).map_err(|_| ())?;
    if label != "CERTIFICATE" || der.is_empty() {
        return Err(());
    }
    Ok(der)
}
