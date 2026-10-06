use super::super::*;
use super::fixtures::{
    ai_element, cert_from_tbs_parts, concat, dsa_params, dsa_spki, ec_spki, err_text, int_raw,
    int_small, null, octet, seq, spki_with_key, std_parts, OID_DSA, OID_EC_PUB, OID_ED25519,
    OID_P256, OID_RSA_ENC, OID_SHA256_RSA,
};

fn key_err(spki: &[u8]) -> String {
    let mut parts = std_parts();
    parts[5] = spki.to_vec();
    err_text(parse_certificate(&cert_from_tbs_parts(
        &parts,
        &ai_element(OID_SHA256_RSA, Some(&null())),
        &[1],
    )))
}

#[test]
fn parse_rsa_key_errors() {
    let key = |n: &[u8], e: &[u8]| seq(&concat(&[int_raw(n), int_raw(e)]));
    let rsa = ai_element(OID_RSA_ENC, Some(&null()));
    // Missing NULL parameters (absent and wrong).
    assert_eq!(
        key_err(&spki_with_key(
            &ai_element(OID_RSA_ENC, None),
            &key(&[1], &[1])
        )),
        "x509: RSA key missing NULL parameters"
    );
    assert_eq!(
        key_err(&spki_with_key(
            &ai_element(OID_RSA_ENC, Some(&octet(&[]))),
            &key(&[1], &[1])
        )),
        "x509: RSA key missing NULL parameters"
    );
    // Key not a SEQUENCE.
    assert_eq!(
        key_err(&spki_with_key(&rsa, &int_small(1))),
        "x509: invalid RSA public key"
    );
    // Modulus not an INTEGER.
    assert_eq!(
        key_err(&spki_with_key(
            &rsa,
            &seq(&concat(&[octet(&[1]), int_small(1)]))
        )),
        "x509: invalid RSA modulus"
    );
    // Exponent missing.
    assert_eq!(
        key_err(&spki_with_key(&rsa, &seq(&int_small(17)))),
        "x509: invalid RSA public exponent"
    );
    // Non-positive modulus.
    assert_eq!(
        key_err(&spki_with_key(&rsa, &key(&[0xFF], &[1]))),
        "x509: RSA modulus is not a positive number"
    );
    assert_eq!(
        key_err(&spki_with_key(&rsa, &key(&[0x00], &[1]))),
        "x509: RSA modulus is not a positive number"
    );
    // Non-positive exponent.
    assert_eq!(
        key_err(&spki_with_key(&rsa, &key(&[0x11], &[0x00]))),
        "x509: RSA public exponent is not a positive number"
    );
    assert_eq!(
        key_err(&spki_with_key(&rsa, &key(&[0x11], &[0xFF]))),
        "x509: RSA public exponent is not a positive number"
    );
    // Exponent too large for int64 is still an invalid exponent.
    assert_eq!(
        key_err(&spki_with_key(&rsa, &key(&[0x11], &[0x01; 9]))),
        "x509: invalid RSA public exponent"
    );
}

#[test]
fn parse_dsa_key() {
    // Happy path stores magnitudes.
    let mut parts = std_parts();
    parts[5] = dsa_spki(&[0x00, 0x80], &dsa_params(&[0x11], &[0x13], &[0x17]));
    let cert = parse_certificate(&cert_from_tbs_parts(
        &parts,
        &ai_element(OID_SHA256_RSA, Some(&null())),
        &[1],
    ))
    .expect("dsa parses");
    assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Dsa);
    match cert.public_key {
        Some(PublicKeyData::Dsa { y, p, q, g }) => {
            assert_eq!(y, vec![0x80]);
            assert_eq!(p, vec![0x11]);
            assert_eq!(q, vec![0x13]);
            assert_eq!(g, vec![0x17]);
        }
        _ => panic!("expected DSA key"),
    }
    // Y not an INTEGER.
    assert_eq!(
        key_err(&spki_with_key(
            &ai_element(OID_DSA, Some(&dsa_params(&[0x11], &[0x13], &[0x17]))),
            &octet(&[1])
        )),
        "x509: invalid DSA public key"
    );
    // Params not a SEQUENCE.
    assert_eq!(
        key_err(&dsa_spki(&[0x11], &int_small(1))),
        "x509: invalid DSA parameters"
    );
    // Q missing.
    assert_eq!(
        key_err(&dsa_spki(&[0x11], &seq(&int_raw(&[0x11])))),
        "x509: invalid DSA parameters"
    );
    // Zero and negative parameters.
    assert_eq!(
        key_err(&dsa_spki(&[0x00], &dsa_params(&[0x11], &[0x13], &[0x17]))),
        "x509: zero or negative DSA parameter"
    );
    assert_eq!(
        key_err(&dsa_spki(&[0x11], &dsa_params(&[0x11], &[0xFF], &[0x17]))),
        "x509: zero or negative DSA parameter"
    );
}

#[test]
fn parse_ecdsa_key_errors() {
    // Params not an OID.
    assert_eq!(
        key_err(&spki_with_key(
            &ai_element(OID_EC_PUB, Some(&null())),
            &[0x04, 0x01, 0x02]
        )),
        "x509: invalid ECDSA parameters"
    );
    // Unsupported curve.
    assert_eq!(
        key_err(&ec_spki(&[0x2A, 0x03], &[0x04, 0x01, 0x02])),
        "x509: unsupported elliptic curve"
    );
    // Empty and compressed points.
    assert_eq!(
        key_err(&ec_spki(OID_P256, &[])),
        "ecdsa: invalid uncompressed public key"
    );
    let mut compressed = vec![0x02];
    compressed.extend_from_slice(&[0x11; 32]);
    assert_eq!(
        key_err(&ec_spki(OID_P256, &compressed)),
        "ecdsa: invalid uncompressed public key"
    );
    // Wrong length.
    let mut short = vec![0x04];
    short.extend_from_slice(&[0x11; 63]);
    assert_eq!(
        key_err(&ec_spki(OID_P256, &short)),
        "invalid P256 point encoding"
    );
    // X equal to the field prime is out of range.
    let mut point = vec![0x04];
    point.extend_from_slice(&P256_PRIME);
    point.extend_from_slice(&[0x01; 32]);
    assert_eq!(key_err(&ec_spki(OID_P256, &point)), P256_ELEMENT_ERROR);
    // Y equal to the field prime is out of range.
    let mut point = vec![0x04];
    point.extend_from_slice(&[0x01; 32]);
    point.extend_from_slice(&P256_PRIME);
    assert_eq!(key_err(&ec_spki(OID_P256, &point)), P256_ELEMENT_ERROR);
    // In-range but off-curve.
    let mut point = vec![0x04];
    point.extend_from_slice(&[0x00; 31]);
    point.push(0x01);
    point.extend_from_slice(&[0x00; 31]);
    point.push(0x01);
    assert_eq!(
        key_err(&ec_spki(OID_P256, &point)),
        "P256 point not on curve"
    );
}

#[test]
fn parse_ecdsa_p256_generator() {
    // The P-256 base point is the standard on-curve vector.
    let gx = [
        0x6B, 0x17, 0xD1, 0xF2, 0xE1, 0x2C, 0x42, 0x47, 0xF8, 0xBC, 0xE6, 0xE5, 0x63, 0xA4, 0x40,
        0xF2, 0x77, 0x03, 0x7D, 0x81, 0x2D, 0xEB, 0x33, 0xA0, 0xF4, 0xA1, 0x39, 0x45, 0xD8, 0x98,
        0xC2, 0x96,
    ];
    let gy = [
        0x4F, 0xE3, 0x42, 0xE2, 0xFE, 0x1A, 0x7F, 0x9B, 0x8E, 0xE7, 0xEB, 0x4A, 0x7C, 0x0F, 0x9E,
        0x16, 0x2B, 0xCE, 0x33, 0x57, 0x6B, 0x31, 0x5E, 0xCE, 0xCB, 0xB6, 0x40, 0x68, 0x37, 0xBF,
        0x51, 0xF5,
    ];
    let mut point = vec![0x04];
    point.extend_from_slice(&gx);
    point.extend_from_slice(&gy);
    let mut parts = std_parts();
    parts[5] = ec_spki(OID_P256, &point);
    let cert = parse_certificate(&cert_from_tbs_parts(
        &parts,
        &ai_element(OID_SHA256_RSA, Some(&null())),
        &[1],
    ))
    .expect("p256 generator parses");
    assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Ecdsa);
    assert!(matches!(cert.public_key, Some(PublicKeyData::Ecdsa256(_))));
}

#[test]
fn parse_ed25519_key() {
    // Happy path.
    let mut parts = std_parts();
    parts[5] = spki_with_key(&ai_element(OID_ED25519, None), &[0x42; 32]);
    let cert = parse_certificate(&cert_from_tbs_parts(
        &parts,
        &ai_element(OID_SHA256_RSA, Some(&null())),
        &[1],
    ))
    .expect("ed25519 parses");
    assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Ed25519);
    assert!(matches!(
        cert.public_key,
        Some(PublicKeyData::Ed25519(k)) if k == [0x42; 32]
    ));
    // Parameters must be absent (even NULL is illegal).
    assert_eq!(
        key_err(&spki_with_key(
            &ai_element(OID_ED25519, Some(&null())),
            &[0x42; 32]
        )),
        "x509: Ed25519 key encoded with illegal parameters"
    );
    // Wrong size.
    assert_eq!(
        key_err(&spki_with_key(&ai_element(OID_ED25519, None), &[0x42; 31])),
        "x509: wrong Ed25519 public key size"
    );
    assert_eq!(
        key_err(&spki_with_key(&ai_element(OID_ED25519, None), &[0x42; 33])),
        "x509: wrong Ed25519 public key size"
    );
}
