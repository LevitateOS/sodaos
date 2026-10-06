use super::super::*;
use super::fixtures::{
    ai_element, bitstring, cert_from_tbs_parts, concat, err_text, int_raw, int_small, null, octet,
    seq, std_cert, std_parts, std_spki, tlv, utctime, OID_RSA_ENC, OID_SHA1_RSA, OID_SHA256_RSA,
};

#[test]
fn parse_minimal_v1_rsa_cert() {
    let bytes = std_cert();
    let cert = parse_certificate(&bytes).expect("parse std cert");
    assert_eq!(cert.version, 1);
    assert_eq!(cert.serial, vec![0x01]);
    assert_eq!(cert.signature_algorithm, SignatureAlgorithm::Sha256Rsa);
    assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Rsa);
    assert!(matches!(cert.public_key, Some(PublicKeyData::Rsa(_))));
    assert_eq!(cert.signature, vec![0xAB, 0xCD]);
    assert_eq!(cert.key_usage, 0);
    assert!(!cert.is_ca);
    assert!(!cert.basic_constraints_valid);
    assert_eq!(cert.max_path_len, 0);
    assert!(!cert.max_path_len_zero);
    assert_eq!(cert.not_before, 0);
    assert_eq!(cert.not_after, 86400);
    assert!(cert.unhandled_critical.is_empty());
    assert_eq!(cert.raw, bytes);
    assert_eq!(cert.raw_tbs, seq(&concat(&std_parts())));
    assert_eq!(cert.raw_subject, vec![0x30, 0x00]);
    assert_eq!(cert.raw_issuer, vec![0x30, 0x00]);
    assert_eq!(cert.raw_subject_public_key_info, std_spki());
}

#[test]
fn parse_ignores_trailing_bytes() {
    let mut bytes = std_cert();
    bytes.extend_from_slice(&[0x00, 0xFF, 0x04]);
    let cert = parse_certificate(&bytes).expect("trailing ignored");
    assert_eq!(cert.version, 1);
}

#[test]
fn parse_structural_error_texts() {
    // Empty and wrong outer tag.
    assert_eq!(
        err_text(parse_certificate(&[])),
        "x509: malformed certificate"
    );
    assert_eq!(
        err_text(parse_certificate(&[0x31, 0x00])),
        "x509: malformed certificate"
    );
    // Truncated outer length.
    assert_eq!(
        err_text(parse_certificate(&[0x30, 0x05, 0x30, 0x00])),
        "x509: malformed certificate"
    );
    // TBS not a SEQUENCE.
    let bad_tbs = seq(&concat(&[int_small(1), int_small(2), bitstring(&[1])]));
    assert_eq!(
        err_text(parse_certificate(&bad_tbs)),
        "x509: malformed tbs certificate"
    );

    let outer = ai_element(OID_SHA256_RSA, Some(&null()));
    let mutant =
        |parts: &[Vec<u8>]| err_text(parse_certificate(&cert_from_tbs_parts(parts, &outer, &[1])));

    // Version element with non-INTEGER contents.
    let mut parts = std_parts();
    parts.insert(0, tlv(0xA0, &[0x04, 0x01, 0x00]));
    assert_eq!(mutant(&parts), "x509: malformed version");
    // Version 4 (field value 3).
    let mut parts = std_parts();
    parts.insert(0, tlv(0xA0, &int_small(3)));
    assert_eq!(mutant(&parts), "x509: invalid version");
    // Negative version.
    let mut parts = std_parts();
    parts.insert(0, tlv(0xA0, &int_raw(&[0xFF])));
    assert_eq!(mutant(&parts), "x509: malformed version");

    // Serial not an INTEGER.
    let mut parts = std_parts();
    parts[0] = seq(&[]);
    assert_eq!(mutant(&parts), "x509: malformed serial number");
    // Negative serial.
    let mut parts = std_parts();
    parts[0] = int_raw(&[0xFF]);
    assert_eq!(mutant(&parts), "x509: negative serial number");

    // Inner AI not a SEQUENCE.
    let mut parts = std_parts();
    parts[1] = int_small(1);
    assert_eq!(
        mutant(&parts),
        "x509: malformed signature algorithm identifier"
    );

    // Issuer not a SEQUENCE.
    let mut parts = std_parts();
    parts[2] = octet(&[1]);
    assert_eq!(mutant(&parts), "x509: malformed issuer");

    // Validity not a SEQUENCE.
    let mut parts = std_parts();
    parts[3] = int_small(1);
    assert_eq!(mutant(&parts), "x509: malformed validity");
    // Malformed notBefore (UTCTime too short).
    let mut parts = std_parts();
    parts[3] = seq(&concat(&[
        utctime("70010100000Z"),
        utctime("700102000000Z"),
    ]));
    assert_eq!(mutant(&parts), "x509: malformed UTCTime");
    // Malformed notAfter (bad zone).
    let mut parts = std_parts();
    parts[3] = seq(&concat(&[
        utctime("700101000000Z"),
        utctime("700102000000X"),
    ]));
    assert_eq!(mutant(&parts), "x509: malformed UTCTime");

    // Subject uses the issuer's error text (Go quirk).
    let mut parts = std_parts();
    parts[4] = int_small(1);
    assert_eq!(mutant(&parts), "x509: malformed issuer");

    // SPKI not a SEQUENCE.
    let mut parts = std_parts();
    parts[5] = int_small(1);
    assert_eq!(mutant(&parts), "x509: malformed spki");
    // SPKI algorithm identifier not a SEQUENCE.
    let mut parts = std_parts();
    parts[5] = seq(&concat(&[int_small(1), bitstring(&[1, 2, 3])]));
    assert_eq!(
        mutant(&parts),
        "x509: malformed public key algorithm identifier"
    );
    // SPKI key not a BIT STRING.
    let mut parts = std_parts();
    parts[5] = seq(&concat(&[
        ai_element(OID_RSA_ENC, Some(&null())),
        octet(&[1, 2, 3]),
    ]));
    assert_eq!(mutant(&parts), "x509: malformed subjectPublicKey");

    // Outer AI not a SEQUENCE.
    let bad_outer = seq(&concat(&[
        seq(&concat(&std_parts())),
        int_small(1),
        bitstring(&[1]),
    ]));
    assert_eq!(
        err_text(parse_certificate(&bad_outer)),
        "x509: malformed algorithm identifier"
    );

    // Inner/outer AI mismatch.
    let mismatch =
        cert_from_tbs_parts(&std_parts(), &ai_element(OID_SHA1_RSA, Some(&null())), &[1]);
    assert_eq!(
        err_text(parse_certificate(&mismatch)),
        "x509: inner and outer signature algorithm identifiers don't match"
    );

    // Signature not a BIT STRING.
    let bad_sig = seq(&concat(&[
        seq(&concat(&std_parts())),
        ai_element(OID_SHA256_RSA, Some(&null())),
        octet(&[1]),
    ]));
    assert_eq!(
        err_text(parse_certificate(&bad_sig)),
        "x509: malformed signature"
    );
}

#[test]
fn parse_unknown_public_key_algorithm_yields_none() {
    let mut parts = std_parts();
    parts[5] = seq(&concat(&[
        ai_element(&[0x2A, 0x03], None),
        bitstring(&[1, 2, 3]),
    ]));
    let cert = parse_certificate(&cert_from_tbs_parts(
        &parts,
        &ai_element(OID_SHA256_RSA, Some(&null())),
        &[1],
    ))
    .expect("unknown pk algorithm parses");
    assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Unknown);
    assert!(cert.public_key.is_none());
}

#[test]
fn parse_unknown_signature_algorithm_yields_unknown() {
    let ai = ai_element(&[0x2A, 0x03], None);
    let cert = parse_certificate(&cert_from_tbs_parts(
        &{
            let mut parts = std_parts();
            parts[1] = ai.clone();
            parts
        },
        &ai,
        &[1],
    ))
    .expect("unknown sig algorithm parses");
    assert_eq!(cert.signature_algorithm, SignatureAlgorithm::Unknown);
}

#[test]
fn parse_versions_and_unique_ids() {
    let outer = ai_element(OID_SHA256_RSA, Some(&null()));
    // v2 without unique IDs.
    let mut parts = std_parts();
    parts.insert(0, tlv(0xA0, &int_small(1)));
    let cert = parse_certificate(&cert_from_tbs_parts(&parts, &outer, &[1])).expect("v2 parses");
    assert_eq!(cert.version, 2);
    // v2 with well-formed unique IDs (skipped).
    let mut parts = parts.clone();
    parts.push(tlv(0x81, &[0x00, 0xAA]));
    parts.push(tlv(0x82, &[0x00, 0xBB]));
    let cert = parse_certificate(&cert_from_tbs_parts(&parts, &outer, &[1]))
        .expect("v2 with unique IDs parses");
    assert_eq!(cert.version, 2);
    // v3 without extensions.
    let mut parts = std_parts();
    parts.insert(0, tlv(0xA0, &int_small(2)));
    let cert = parse_certificate(&cert_from_tbs_parts(&parts, &outer, &[1]))
        .expect("v3 without extensions parses");
    assert_eq!(cert.version, 3);
    assert!(!cert.basic_constraints_valid);
    // v3 with malformed extensions wrapper.
    let mut parts = std_parts();
    parts.insert(0, tlv(0xA0, &int_small(2)));
    parts.push(tlv(0xA3, &[0x04, 0x01, 0x00]));
    assert_eq!(
        err_text(parse_certificate(&cert_from_tbs_parts(
            &parts,
            &outer,
            &[1]
        ))),
        "x509: malformed extensions"
    );
}
