use self::fixtures::{
    ai_element, bitstring, boolean, cert_from_tbs_parts, concat, dsa_params, dsa_spki, ec_spki,
    extension, int_raw, int_small, ku_value, null, seq, spki_with_key, std_parts, std_spki, tlv,
    v3_cert, v3_parts, OID_BASIC_CONSTRAINTS, OID_DSA_SHA256, OID_ECDSA_SHA1, OID_ECDSA_SHA256,
    OID_ED25519, OID_KEY_USAGE, OID_MD5_RSA, OID_P256, OID_SHA1_RSA, OID_SHA256_RSA,
};
use super::*;

mod algorithms;
mod extensions;
mod fixtures;
mod public_key;
mod structure;
mod time;

#[test]
fn ecdsa_signature_lenient_parse() {
    // Minimal valid.
    let sig = seq(&concat(&[int_small(1), int_small(2)]));
    assert_eq!(parse_ecdsa_signature(&sig), Some((vec![0x01], vec![0x02])));
    // Non-minimal integers are accepted (encoding/asn1 semantics).
    let sig = seq(&concat(&[int_raw(&[0x00, 0x00, 0x01]), int_small(2)]));
    assert_eq!(parse_ecdsa_signature(&sig), Some((vec![0x01], vec![0x02])));
    // Negative, zero, and empty values are rejected.
    let sig = seq(&concat(&[int_raw(&[0xFF]), int_small(2)]));
    assert_eq!(parse_ecdsa_signature(&sig), None);
    let sig = seq(&concat(&[int_small(1), int_raw(&[0x00])]));
    assert_eq!(parse_ecdsa_signature(&sig), None);
    let sig = seq(&concat(&[tlv(0x02, &[]), int_small(2)]));
    assert_eq!(parse_ecdsa_signature(&sig), None);
    // Wrong arity, trailing bytes, wrong tag.
    let sig = seq(&concat(&[int_small(1), int_small(2), int_small(3)]));
    assert_eq!(parse_ecdsa_signature(&sig), None);
    let mut sig = seq(&concat(&[int_small(1), int_small(2)]));
    sig.push(0x00);
    assert_eq!(parse_ecdsa_signature(&sig), None);
    assert_eq!(parse_ecdsa_signature(&[0x04, 0x02, 0x01, 0x02]), None);
}

/// v3 CA parent with the given SPKI and extra extensions.
fn ca_cert(spki: &[u8], extra: &[Vec<u8>]) -> Certificate {
    let mut exts = vec![extension(OID_BASIC_CONSTRAINTS, true, &seq(&boolean(true)))];
    exts.extend_from_slice(extra);
    let mut parts = v3_parts(&exts);
    parts[6] = spki.to_vec();
    parse_certificate(&cert_from_tbs_parts(
        &parts,
        &ai_element(OID_SHA256_RSA, Some(&null())),
        &[1],
    ))
    .expect("ca parses")
}

fn child_with_alg(sig_oid: &[u8], params: Option<&[u8]>) -> Certificate {
    let ai = ai_element(sig_oid, params);
    let mut parts = std_parts();
    parts[1] = ai.clone();
    parse_certificate(&cert_from_tbs_parts(&parts, &ai, &[0xDE, 0xAD])).expect("child parses")
}

#[test]
fn check_signature_parent_constraints() {
    let child = child_with_alg(OID_SHA256_RSA, Some(&null()));
    // v3 parent without basic constraints.
    let parent = parse_certificate(&v3_cert(&[])).expect("parent parses");
    assert_eq!(
        check_signature_from(&child, &parent),
        Err(String::from(
            "x509: invalid signature: parent certificate cannot sign this kind of certificate"
        ))
    );
    // Basic constraints present but not a CA.
    let parent = parse_certificate(&v3_cert(&[extension(
        OID_BASIC_CONSTRAINTS,
        true,
        &seq(&boolean(false)),
    )]))
    .expect("parent parses");
    assert_eq!(
        check_signature_from(&child, &parent),
        Err(String::from(
            "x509: invalid signature: parent certificate cannot sign this kind of certificate"
        ))
    );
    // Key usage without certSign.
    let parent = ca_cert(
        &std_spki(),
        &[extension(OID_KEY_USAGE, true, &ku_value(&[0x80], 1))],
    );
    assert_eq!(
        check_signature_from(&child, &parent),
        Err(String::from(
            "x509: invalid signature: parent certificate cannot sign this kind of certificate"
        ))
    );
    // Unknown parent public key algorithm.
    let mut parts = v3_parts(&[extension(OID_BASIC_CONSTRAINTS, true, &seq(&boolean(true)))]);
    parts[6] = seq(&concat(&[ai_element(&[0x2A, 0x03], None), bitstring(&[1])]));
    let parent = parse_certificate(&cert_from_tbs_parts(
        &parts,
        &ai_element(OID_SHA256_RSA, Some(&null())),
        &[1],
    ))
    .expect("parent parses");
    assert_eq!(
        check_signature_from(&child, &parent),
        Err(String::from(
            "x509: cannot verify signature: algorithm unimplemented"
        ))
    );
}

#[test]
fn check_signature_algorithm_gates() {
    let parent = ca_cert(&std_spki(), &[]);
    // Insecure hashes name Go's algorithm string.
    for (oid_body, name) in [
        (OID_MD5_RSA, "MD5-RSA"),
        (OID_SHA1_RSA, "SHA1-RSA"),
        (OID_ECDSA_SHA1, "ECDSA-SHA1"),
    ] {
        let child = child_with_alg(oid_body, Some(&null()));
        assert_eq!(
            check_signature_from(&child, &parent),
            Err(format!(
                "x509: cannot verify signature: insecure algorithm {name}"
            )),
            "{name} must be rejected"
        );
    }
    // Unknown child algorithm.
    let child = child_with_alg(&[0x2A, 0x03], None);
    assert_eq!(
        check_signature_from(&child, &parent),
        Err(String::from(
            "x509: cannot verify signature: algorithm unimplemented"
        ))
    );
}

#[test]
fn check_signature_key_mismatches() {
    let rsa_parent = ca_cert(&std_spki(), &[]);
    // RSA key with an ECDSA signature algorithm.
    let child = child_with_alg(OID_ECDSA_SHA256, None);
    assert_eq!(
        check_signature_from(&child, &rsa_parent),
        Err(String::from(
            "x509: signature algorithm specifies an ECDSA public key, but have public key of type *rsa.PublicKey"
        ))
    );
    // RSA key with a DSA signature algorithm (Go's "an DSA" quirk: the
    // DSA-SHA256 hash is secure, so dispatch reaches the key check).
    let child = child_with_alg(OID_DSA_SHA256, None);
    assert_eq!(
        check_signature_from(&child, &rsa_parent),
        Err(String::from(
            "x509: signature algorithm specifies an DSA public key, but have public key of type *rsa.PublicKey"
        ))
    );
    // ECDSA (P-256 generator) key with an RSA signature algorithm.
    let mut point = vec![0x04];
    point.extend_from_slice(&[
        0x6B, 0x17, 0xD1, 0xF2, 0xE1, 0x2C, 0x42, 0x47, 0xF8, 0xBC, 0xE6, 0xE5, 0x63, 0xA4, 0x40,
        0xF2, 0x77, 0x03, 0x7D, 0x81, 0x2D, 0xEB, 0x33, 0xA0, 0xF4, 0xA1, 0x39, 0x45, 0xD8, 0x98,
        0xC2, 0x96,
    ]);
    point.extend_from_slice(&[
        0x4F, 0xE3, 0x42, 0xE2, 0xFE, 0x1A, 0x7F, 0x9B, 0x8E, 0xE7, 0xEB, 0x4A, 0x7C, 0x0F, 0x9E,
        0x16, 0x2B, 0xCE, 0x33, 0x57, 0x6B, 0x31, 0x5E, 0xCE, 0xCB, 0xB6, 0x40, 0x68, 0x37, 0xBF,
        0x51, 0xF5,
    ]);
    let ec_parent = ca_cert(&ec_spki(OID_P256, &point), &[]);
    let child = child_with_alg(OID_SHA256_RSA, Some(&null()));
    assert_eq!(
        check_signature_from(&child, &ec_parent),
        Err(String::from(
            "x509: signature algorithm specifies an RSA public key, but have public key of type *ecdsa.PublicKey"
        ))
    );
    // DSA keys have no verify arm in Go: always unimplemented.
    let dsa_parent = ca_cert(
        &dsa_spki(&[0x11], &dsa_params(&[0x17], &[0x13], &[0x05])),
        &[],
    );
    assert_eq!(
        check_signature_from(&child, &dsa_parent),
        Err(String::from(
            "x509: cannot verify signature: algorithm unimplemented"
        ))
    );
    // RSA key with garbage signature fails verification.
    assert_eq!(
        check_signature_from(&child, &rsa_parent),
        Err(String::from("crypto/rsa: verification error"))
    );
}

#[test]
fn check_signature_ed25519_self_signed_round_trip() {
    use ed25519_dalek::Signer as _;
    let signing = ed25519_dalek::SigningKey::from_bytes(&[
        0x9D, 0x61, 0xB1, 0x9D, 0xEF, 0xFD, 0x5A, 0x60, 0xBA, 0x84, 0x4A, 0xF4, 0x92, 0x2E, 0xC4,
        0x44, 0x48, 0xC8, 0x58, 0x07, 0x31, 0x11, 0xED, 0xD3, 0xAD, 0x45, 0x8B, 0x22, 0x7E, 0x4E,
        0x4B, 0x63,
    ]);
    let verifying = signing.verifying_key();
    let spki = spki_with_key(&ai_element(OID_ED25519, None), verifying.as_bytes());
    let ai = ai_element(OID_ED25519, None);
    let mut parts = v3_parts(&[extension(OID_BASIC_CONSTRAINTS, true, &seq(&boolean(true)))]);
    parts[2] = ai.clone();
    parts[6] = spki;
    let tbs = seq(&concat(&parts));
    let sig = signing.sign(&tbs);
    let cert = parse_certificate(&seq(&concat(&[tbs, ai, bitstring(&sig.to_bytes())])))
        .expect("self-signed parses");
    assert_eq!(cert.signature_algorithm, SignatureAlgorithm::Ed25519);
    assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Ed25519);
    assert!(cert.is_ca);
    check_signature_from(&cert, &cert).expect("self-signed verifies");
    // A tampered signature fails.
    let mut bad = sig.to_bytes();
    bad[0] ^= 0x01;
    let mut parts = v3_parts(&[extension(OID_BASIC_CONSTRAINTS, true, &seq(&boolean(true)))]);
    parts[2] = ai_element(OID_ED25519, None);
    parts[6] = spki_with_key(&ai_element(OID_ED25519, None), verifying.as_bytes());
    let tbs = seq(&concat(&parts));
    let tampered = parse_certificate(&seq(&concat(&[
        tbs,
        ai_element(OID_ED25519, None),
        bitstring(&bad),
    ])))
    .expect("tampered parses");
    assert_eq!(
        check_signature_from(&tampered, &cert),
        Err(String::from("x509: Ed25519 verification failure"))
    );
}
