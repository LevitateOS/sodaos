//! Bounded admission for the P-256 public keys in release trust documents.

use p256::pkcs8::{
    der::Decode, DecodePublicKey, Document, ObjectIdentifier, SubjectPublicKeyInfoRef,
};

const MAX_PEM_BYTES: usize = 1 << 20;
const BEGIN: &str = "-----BEGIN PUBLIC KEY-----";
const END: &str = "-----END PUBLIC KEY-----";

/// Decode one bounded PUBLIC KEY PEM and admit an on-curve P-256 SPKI.
///
/// The 1 MiB bound matches the existing trust-document file bound at both
/// callers. The admitted DER is returned intact
/// so callers continue fingerprinting the original bytes.
pub fn parse_p256_public_key(pem: &str) -> Result<Vec<u8>, ()> {
    if pem.len() > MAX_PEM_BYTES {
        return Err(());
    }
    let trimmed = pem.trim_matches(|c: char| c.is_ascii_whitespace());
    if !trimmed.starts_with(BEGIN) || !trimmed.ends_with(END) {
        return Err(());
    }
    let body = &trimmed[BEGIN.len()..trimmed.len() - END.len()];
    if body.contains(BEGIN) || body.contains(END) {
        return Err(());
    }

    let (label, document) = Document::from_pem(trimmed).map_err(|_| ())?;
    if label != "PUBLIC KEY" {
        return Err(());
    }
    let der = document.as_bytes();

    // Parse the full SPKI with the typed DER implementation before inspecting
    // its algorithm and point representation.
    let spki = SubjectPublicKeyInfoRef::from_der(der).map_err(|_| ())?;
    let ec_public_key = ObjectIdentifier::new_unwrap("1.2.840.10045.2.1");
    let prime256v1 = ObjectIdentifier::new_unwrap("1.2.840.10045.3.1.7");
    if spki.algorithm.oid != ec_public_key {
        return Err(());
    }
    let params = spki.algorithm.parameters.ok_or(())?;
    let curve = params.decode_as::<ObjectIdentifier>().map_err(|_| ())?;
    if curve != prime256v1 {
        return Err(());
    }
    let point = spki.subject_public_key;
    if point.unused_bits() != 0 {
        return Err(());
    }
    let encoded_point = point.raw_bytes();
    if encoded_point.len() != 65 || encoded_point[0] != 0x04 {
        return Err(());
    }

    // This typed key constructor checks the actual curve equation and rejects
    // identity and off-curve points (including x=y=1).
    p256::PublicKey::from_public_key_der(der).map_err(|_| ())?;
    Ok(der.to_vec())
}
