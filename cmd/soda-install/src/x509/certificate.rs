use std::collections::HashSet;

use x509_cert::{
    der::{oid::AssociatedOid, Decode, Encode, Reader, SliceReader},
    ext::pkix::{BasicConstraints, KeyUsage},
    Certificate as TypedCertificate,
};

use super::{algorithms::PublicKeyAlgorithm, types::Certificate};

/// Decode the certificate with the upstream strict DER/X.509 implementation,
/// then retain only the values used for local CA admission and self-signature
/// verification. `raw_tbs` is borrowed from the admitted input, never encoded
/// again from the typed value.
pub(crate) fn parse_certificate(der: &[u8]) -> Result<Certificate, String> {
    let typed =
        TypedCertificate::from_der(der).map_err(|_| String::from("invalid certificate DER"))?;
    if typed
        .to_der()
        .map_err(|_| String::from("invalid certificate DER"))?
        != der
    {
        return Err(String::from("noncanonical certificate DER"));
    }
    let has_extensions = typed.tbs_certificate.extensions.is_some();
    if has_extensions && typed.tbs_certificate.version != x509_cert::certificate::Version::V3 {
        return Err(String::from("extensions require certificate version 3"));
    }
    if (typed.tbs_certificate.issuer_unique_id.is_some()
        || typed.tbs_certificate.subject_unique_id.is_some())
        && typed.tbs_certificate.version == x509_cert::certificate::Version::V1
    {
        return Err(String::from(
            "unique identifiers require certificate version 2 or 3",
        ));
    }
    if typed.tbs_certificate.signature != typed.signature_algorithm {
        return Err(String::from(
            "certificate signature algorithms do not match",
        ));
    }

    let serial = typed.tbs_certificate.serial_number.as_bytes();
    if serial.is_empty()
        || serial.len() > 20
        || serial[0] & 0x80 != 0
        || serial.iter().all(|byte| *byte == 0)
    {
        return Err(String::from("invalid certificate serial number"));
    }

    let mut reader = SliceReader::new(der).map_err(|_| String::from("invalid certificate DER"))?;
    let raw_tbs = reader
        .sequence(|sequence| {
            let tbs = sequence.tlv_bytes()?;
            let _ = sequence.tlv_bytes()?;
            let _ = sequence.tlv_bytes()?;
            Ok(tbs)
        })
        .map_err(|_| String::from("invalid certificate DER"))?;
    reader
        .finish(())
        .map_err(|_| String::from("trailing certificate DER"))?;

    let mut seen = HashSet::new();
    for extension in typed
        .tbs_certificate
        .extensions
        .as_deref()
        .unwrap_or_default()
    {
        if !seen.insert(extension.extn_id) {
            return Err(String::from("duplicate certificate extension"));
        }
        if extension.critical
            && extension.extn_id != BasicConstraints::OID
            && extension.extn_id != KeyUsage::OID
        {
            return Err(String::from("unsupported critical certificate extension"));
        }
    }

    let (_basic_constraints_critical, basic_constraints) = typed
        .tbs_certificate
        .get::<BasicConstraints>()
        .map_err(|_| String::from("invalid basic constraints"))?
        .ok_or_else(|| String::from("missing basic constraints"))?;
    if !basic_constraints.ca {
        return Err(String::from("certificate is not a CA"));
    }
    let key_usage = typed
        .tbs_certificate
        .get::<KeyUsage>()
        .map_err(|_| String::from("invalid key usage"))?;
    if let Some((_, usage)) = &key_usage {
        if !usage.key_cert_sign() {
            return Err(String::from(
                "CA key usage does not permit certificate signing",
            ));
        }
    }

    let spki = &typed.tbs_certificate.subject_public_key_info;
    let key_bytes = spki
        .subject_public_key
        .as_bytes()
        .ok_or_else(|| String::from("unaligned public key bits"))?;
    let (key_algorithm, public_key) = parse_public_key(spki, key_bytes)?;
    let signature_algorithm = super::algorithms::signature_algorithm(&typed.signature_algorithm);
    let signature = typed
        .signature
        .as_bytes()
        .ok_or_else(|| String::from("unaligned certificate signature"))?
        .to_vec();

    Ok(Certificate {
        raw: der.to_vec(),
        raw_tbs: raw_tbs.to_vec(),
        signature_algorithm,
        public_key_algorithm: key_algorithm,
        public_key: Some(public_key),
        signature,
    })
}

fn parse_public_key(
    spki: &x509_cert::spki::SubjectPublicKeyInfoOwned,
    key: &[u8],
) -> Result<(PublicKeyAlgorithm, super::types::PublicKeyData), String> {
    use rsa::pkcs1::DecodeRsaPublicKey;
    let algorithm = spki.algorithm.oid.to_string();
    match algorithm.as_str() {
        "1.2.840.113549.1.1.1" => {
            if !rsa_parameters_supported(&spki.algorithm.parameters) {
                return Err(String::from("invalid RSA public key parameters"));
            }
            let public = rsa::RsaPublicKey::from_pkcs1_der(key)
                .map_err(|_| String::from("invalid RSA public key"))?;
            Ok((
                PublicKeyAlgorithm::Rsa,
                super::types::PublicKeyData::Rsa(public),
            ))
        }
        "1.2.840.10045.2.1" => {
            let curve = spki
                .algorithm
                .parameters
                .as_ref()
                .ok_or_else(|| String::from("missing ECDSA curve"))?
                .decode_as::<x509_cert::der::asn1::ObjectIdentifier>()
                .map_err(|_| String::from("invalid ECDSA curve"))?
                .to_string();
            let parsed = match curve.as_str() {
                "1.3.132.0.33" if key.first() == Some(&4) && key.len() == 57 => {
                    super::types::PublicKeyData::Ecdsa224(
                        ecdsa::VerifyingKey::<p224::NistP224>::from_sec1_bytes(key)
                            .map_err(|_| String::from("invalid ECDSA public key"))?,
                    )
                }
                "1.2.840.10045.3.1.7" if key.first() == Some(&4) && key.len() == 65 => {
                    super::types::PublicKeyData::Ecdsa256(
                        ecdsa::VerifyingKey::<p256::NistP256>::from_sec1_bytes(key)
                            .map_err(|_| String::from("invalid ECDSA public key"))?,
                    )
                }
                "1.3.132.0.34" if key.first() == Some(&4) && key.len() == 97 => {
                    super::types::PublicKeyData::Ecdsa384(
                        ecdsa::VerifyingKey::<p384::NistP384>::from_sec1_bytes(key)
                            .map_err(|_| String::from("invalid ECDSA public key"))?,
                    )
                }
                "1.3.132.0.35" if key.first() == Some(&4) && key.len() == 133 => {
                    super::types::PublicKeyData::Ecdsa521(
                        ecdsa::VerifyingKey::<p521::NistP521>::from_sec1_bytes(key)
                            .map_err(|_| String::from("invalid ECDSA public key"))?,
                    )
                }
                _ => return Err(String::from("unsupported ECDSA curve")),
            };
            Ok((PublicKeyAlgorithm::Ecdsa, parsed))
        }
        "1.3.101.112" => {
            if spki.algorithm.parameters.is_some() || key.len() != 32 {
                return Err(String::from("invalid Ed25519 public key"));
            }
            let mut bytes = [0; 32];
            bytes.copy_from_slice(key);
            Ok((
                PublicKeyAlgorithm::Ed25519,
                super::types::PublicKeyData::Ed25519(bytes),
            ))
        }
        _ => Err(String::from("unsupported public key algorithm")),
    }
}

pub(super) fn rsa_parameters_supported(parameters: &Option<x509_cert::der::asn1::Any>) -> bool {
    parameters
        .as_ref()
        .is_none_or(|value| x509_cert::der::asn1::AnyRef::from(value).is_null())
}
