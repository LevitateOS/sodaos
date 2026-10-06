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

#[cfg(test)]
mod tests {
    use super::parse_p256_public_key;
    use p256::pkcs8::{der::pem::LineEnding, Document};

    const GENERATOR: &str = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEaxfR8uEsQkf4vOblY6RA8ncDfYEt\n6zOg9KE5RdiYwpZP40Li/hp/m47n60p8D54WK84zV2sxXs7LtkBoN79R9Q==\n-----END PUBLIC KEY-----";

    #[test]
    fn admits_p256_and_returns_the_decoded_spki() {
        let der = parse_p256_public_key(GENERATOR).expect("valid P-256 SPKI");
        assert_eq!(&der[..2], &[0x30, 0x59]);
        assert_eq!(der.len(), 91);
    }

    #[test]
    fn rejects_off_curve_and_non_single_pem_envelopes() {
        let (_, document) = Document::from_pem(GENERATOR).expect("fixture PEM");
        let mut off_curve = document.as_bytes().to_vec();
        off_curve[27..59].fill(0);
        off_curve[58] = 1;
        off_curve[59..91].fill(0);
        off_curve[90] = 1;
        assert_eq!(&off_curve[27..59], &off_curve[59..91]);
        assert_eq!(off_curve[58], 1);
        assert_eq!(off_curve[90], 1);
        let off_curve_pem = p256::pkcs8::der::pem::encode_string(
            "PUBLIC KEY",
            p256::pkcs8::der::pem::LineEnding::LF,
            &off_curve,
        )
        .expect("encode off-curve test key");
        assert!(parse_p256_public_key(&off_curve_pem).is_err());
        assert!(parse_p256_public_key(&format!("preamble\n{GENERATOR}")).is_err());
        assert!(parse_p256_public_key(&format!("{GENERATOR}\ntrailing")).is_err());
        assert!(parse_p256_public_key(&format!("{GENERATOR}\n{GENERATOR}")).is_err());
    }

    #[test]
    fn rejects_non_p256_or_noncanonical_spki_encodings() {
        let (_, document) = Document::from_pem(GENERATOR).expect("fixture PEM");
        let der = document.as_bytes();
        let pem = |bytes: &[u8]| {
            p256::pkcs8::der::pem::encode_string("PUBLIC KEY", LineEnding::LF, bytes)
                .expect("PEM encoding")
        };

        let mut wrong_curve = der.to_vec();
        wrong_curve[22] = 8;
        assert!(parse_p256_public_key(&pem(&wrong_curve)).is_err());

        let mut compressed = der[..23].to_vec();
        compressed[1] = 57;
        compressed.extend_from_slice(&[0x03, 34, 0, 0x02]);
        compressed.extend_from_slice(&der[27..59]);
        assert!(parse_p256_public_key(&pem(&compressed)).is_err());

        let mut nonzero_unused_bits = der.to_vec();
        nonzero_unused_bits[25] = 1;
        let last = nonzero_unused_bits.len() - 1;
        nonzero_unused_bits[last] &= !1;
        assert!(parse_p256_public_key(&pem(&nonzero_unused_bits)).is_err());

        let mut nonminimal_length = vec![0x30, 0x81, der[1]];
        nonminimal_length.extend_from_slice(&der[2..]);
        assert!(parse_p256_public_key(&pem(&nonminimal_length)).is_err());

        let mut trailing_der = der.to_vec();
        trailing_der.push(0);
        assert!(parse_p256_public_key(&pem(&trailing_der)).is_err());
    }
}
