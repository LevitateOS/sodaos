use crate::errors::Error;

pub(super) fn local_ca_fingerprint(data: &[u8]) -> Result<String, Error> {
    let der = crate::pemx::decode_certificate(data)
        .map_err(|_| Error::msg("expected one public CA certificate"))?;
    let certificate = crate::x509::parse_certificate(&der)
        .map_err(|_| Error::msg("invalid local CA certificate"))?;
    if crate::x509::verify_self_signature(&certificate).is_err() {
        return Err(Error::msg("invalid local CA certificate"));
    }
    Ok(crate::buildx::sha256_hex(&certificate.raw))
}
