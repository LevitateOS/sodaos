use crate::errors::Error;

pub(super) fn local_ca_fingerprint(data: &[u8]) -> Result<String, Error> {
    let (block, rest) = crate::pemx::decode(data);
    let block = match block {
        Some(block) if block.der_type == "CERTIFICATE" => block,
        _ => return Err(Error::msg("expected one public CA certificate")),
    };
    if !String::from_utf8_lossy(rest).trim().is_empty() {
        return Err(Error::msg("expected one public CA certificate"));
    }
    let certificate = crate::x509::parse_certificate(&block.bytes)
        .map_err(|_| Error::msg("invalid local CA certificate"))?;
    if !certificate.is_ca
        || !certificate.basic_constraints_valid
        || crate::x509::check_signature_from(&certificate, &certificate).is_err()
    {
        return Err(Error::msg("invalid local CA certificate"));
    }
    Ok(crate::buildx::sha256_hex(&certificate.raw))
}
