use self::fixtures::{
    ai_element, bitstring, boolean, cert_from_tbs_parts, concat, dsa_params, dsa_spki, ec_spki,
    err_text, extension, gentime, int_raw, int_small, ku_value, null, octet, oid, seq,
    spki_with_key, std_parts, std_spki, tlv, utctime, v3_cert, v3_parts, OID_BASIC_CONSTRAINTS,
    OID_DSA, OID_DSA_SHA256, OID_ECDSA_SHA1, OID_ECDSA_SHA256, OID_EC_PUB, OID_ED25519,
    OID_ISO_SHA1_RSA, OID_KEY_USAGE, OID_MD5_RSA, OID_MGF1, OID_P256, OID_PSS, OID_RSA_ENC,
    OID_SHA1_RSA, OID_SHA256, OID_SHA256_RSA, OID_SHA384, OID_SHA512,
};
use super::*;

mod fixtures;
mod structure;

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

fn sig_alg(oid_body: &[u8], params: Option<&[u8]>) -> SignatureAlgorithm {
    let ai = ai_element(oid_body, params);
    let mut reader = Reader::new(&ai);
    let contents = reader.read_asn1(0x30).expect("ai seq");
    let parsed = parse_ai(contents).expect("parse ai");
    signature_algorithm_from_ai(&parsed)
}

#[test]
fn signature_algorithm_mapping() {
    // RSA/ECDSA OIDs map regardless of parameters.
    assert_eq!(
        sig_alg(OID_SHA256_RSA, Some(&null())),
        SignatureAlgorithm::Sha256Rsa
    );
    assert_eq!(sig_alg(OID_SHA256_RSA, None), SignatureAlgorithm::Sha256Rsa);
    assert_eq!(sig_alg(OID_SHA1_RSA, None), SignatureAlgorithm::Sha1Rsa);
    assert_eq!(
        sig_alg(OID_ISO_SHA1_RSA, Some(&null())),
        SignatureAlgorithm::Sha1Rsa
    );
    assert_eq!(sig_alg(OID_MD5_RSA, None), SignatureAlgorithm::Md5Rsa);
    assert_eq!(
        sig_alg(OID_ECDSA_SHA256, None),
        SignatureAlgorithm::EcdsaSha256
    );
    assert_eq!(sig_alg(OID_ECDSA_SHA1, None), SignatureAlgorithm::EcdsaSha1);
    assert_eq!(sig_alg(OID_DSA_SHA256, None), SignatureAlgorithm::DsaSha256);
    // Ed25519 requires absent parameters.
    assert_eq!(sig_alg(OID_ED25519, None), SignatureAlgorithm::Ed25519);
    assert_eq!(
        sig_alg(OID_ED25519, Some(&null())),
        SignatureAlgorithm::Unknown
    );
    // Unknown OIDs map to Unknown.
    assert_eq!(sig_alg(&[0x2A, 0x03], None), SignatureAlgorithm::Unknown);
}

fn pss_params(hash: &[u8], mgf_hash: &[u8], salt: u64, trailer: Option<u64>) -> Vec<u8> {
    let hash_ai = ai_element(hash, None);
    let mgf_inner = ai_element(mgf_hash, None);
    let mgf_ai = ai_element(OID_MGF1, Some(&mgf_inner));
    let mut contents = tlv(0xA0, &hash_ai);
    contents.extend_from_slice(&tlv(0xA1, &mgf_ai));
    contents.extend_from_slice(&tlv(0xA2, &int_small(salt)));
    if let Some(t) = trailer {
        contents.extend_from_slice(&tlv(0xA3, &int_small(t)));
    }
    seq(&contents)
}

#[test]
fn pss_algorithm_buckets() {
    // The three standard buckets.
    assert_eq!(
        sig_alg(OID_PSS, Some(&pss_params(OID_SHA256, OID_SHA256, 32, None))),
        SignatureAlgorithm::Sha256Pss
    );
    assert_eq!(
        sig_alg(OID_PSS, Some(&pss_params(OID_SHA384, OID_SHA384, 48, None))),
        SignatureAlgorithm::Sha384Pss
    );
    assert_eq!(
        sig_alg(OID_PSS, Some(&pss_params(OID_SHA512, OID_SHA512, 64, None))),
        SignatureAlgorithm::Sha512Pss
    );
    // Explicit default trailer is fine.
    assert_eq!(
        sig_alg(
            OID_PSS,
            Some(&pss_params(OID_SHA256, OID_SHA256, 32, Some(1)))
        ),
        SignatureAlgorithm::Sha256Pss
    );
    // Everything else collapses to Unknown.
    assert_eq!(
        sig_alg(OID_PSS, Some(&pss_params(OID_SHA256, OID_SHA256, 20, None))),
        SignatureAlgorithm::Unknown
    );
    assert_eq!(
        sig_alg(OID_PSS, Some(&pss_params(OID_SHA256, OID_SHA384, 32, None))),
        SignatureAlgorithm::Unknown
    );
    assert_eq!(
        sig_alg(
            OID_PSS,
            Some(&pss_params(OID_SHA256, OID_SHA256, 32, Some(2)))
        ),
        SignatureAlgorithm::Unknown
    );
    assert_eq!(
        sig_alg(OID_PSS, Some(&pss_params(OID_SHA384, OID_SHA384, 32, None))),
        SignatureAlgorithm::Unknown
    );
    // Hash parameters may be explicit NULL.
    let hash_ai = ai_element(OID_SHA256, Some(&null()));
    let mgf_inner = ai_element(OID_SHA256, None);
    let mgf_ai = ai_element(OID_MGF1, Some(&mgf_inner));
    let mut contents = tlv(0xA0, &hash_ai);
    contents.extend_from_slice(&tlv(0xA1, &mgf_ai));
    contents.extend_from_slice(&tlv(0xA2, &int_small(32)));
    assert_eq!(
        sig_alg(OID_PSS, Some(&seq(&contents))),
        SignatureAlgorithm::Sha256Pss
    );
    // Garbage parameters.
    assert_eq!(
        sig_alg(OID_PSS, Some(&octet(&[1, 2, 3]))),
        SignatureAlgorithm::Unknown
    );
    assert_eq!(sig_alg(OID_PSS, None), SignatureAlgorithm::Unknown);
}

fn validity_time(nb: &[u8], na: &[u8]) -> (i64, i64) {
    let mut parts = std_parts();
    parts[3] = seq(&concat(&[nb.to_vec(), na.to_vec()]));
    let cert = parse_certificate(&cert_from_tbs_parts(
        &parts,
        &ai_element(OID_SHA256_RSA, Some(&null())),
        &[1],
    ))
    .expect("validity parses");
    (cert.not_before, cert.not_after)
}

#[test]
fn validity_time_values() {
    // UTCTime year mapping and epoch.
    assert_eq!(
        validity_time(&utctime("700101000000Z"), &utctime("700102000000Z")),
        (0, 86400)
    );
    assert_eq!(
        validity_time(&utctime("500101000000Z"), &utctime("700101000000Z")).0,
        -631152000
    );
    assert_eq!(
        validity_time(&utctime("490101000000Z"), &utctime("700101000000Z")).0,
        2493072000
    );
    // GeneralizedTime.
    assert_eq!(
        validity_time(&gentime("19700101000000Z"), &utctime("700101000000Z")).0,
        0
    );
    // Numeric zones shift to UTC.
    assert_eq!(
        validity_time(&utctime("700101000000+0130"), &utctime("700101000000Z")).0,
        -5400
    );
    assert_eq!(
        validity_time(&utctime("700101030000-0130"), &utctime("700101000000Z")).0,
        16200
    );
    // Leap day.
    assert_eq!(
        validity_time(&utctime("000229120000Z"), &utctime("700101000000Z")).0,
        951825600
    );
}

#[test]
fn validity_time_rejections() {
    // Month 13.
    let mut parts = std_parts();
    parts[3] = seq(&concat(&[
        utctime("701301000000Z"),
        utctime("700102000000Z"),
    ]));
    assert_eq!(
        err_text(parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1]
        ))),
        "x509: malformed UTCTime"
    );
    // Hour 24.
    parts[3] = seq(&concat(&[
        utctime("700101240000Z"),
        utctime("700102000000Z"),
    ]));
    assert_eq!(
        err_text(parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1]
        ))),
        "x509: malformed UTCTime"
    );
    // GeneralizedTime wrong length.
    parts[3] = seq(&concat(&[
        gentime("1970010100000Z"),
        utctime("700102000000Z"),
    ]));
    assert_eq!(
        err_text(parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1]
        ))),
        "x509: malformed GeneralizedTime"
    );
    // Non-UTCTime/GeneralizedTime tag.
    parts[3] = seq(&concat(&[
        octet(b"700101000000Z"),
        utctime("700102000000Z"),
    ]));
    assert_eq!(
        err_text(parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1]
        ))),
        "x509: unsupported time format"
    );
}

#[test]
fn mailbox_acceptance() {
    assert!(parse_rfc2821_mailbox(b"user@example.com"));
    assert!(parse_rfc2821_mailbox(b"a@b"));
    assert!(parse_rfc2821_mailbox(b"\"a b\"@c"));
    assert!(parse_rfc2821_mailbox(b"a\\ b@c"));
    assert!(!parse_rfc2821_mailbox(b""));
    assert!(!parse_rfc2821_mailbox(b"a"));
    assert!(!parse_rfc2821_mailbox(b"@b"));
    // Go quirk: an empty domain passes domainToReverseLabels, so "a@" is valid.
    assert!(parse_rfc2821_mailbox(b"a@"));
    // Go quirk: '@' (64) is a legal domain char, so "a@b@c" is valid.
    assert!(parse_rfc2821_mailbox(b"a@b@c"));
    assert!(!parse_rfc2821_mailbox(b"\"a@c"));
    assert!(!parse_rfc2821_mailbox(b"\"a\"b@c"));
    assert!(!parse_rfc2821_mailbox(b".a@b"));
    assert!(!parse_rfc2821_mailbox(b"a.@b"));
    assert!(!parse_rfc2821_mailbox(b"a..b@c"));
    assert!(!parse_rfc2821_mailbox(b"a b@c"));
    assert!(!parse_rfc2821_mailbox(b"a@b "));
}

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

#[test]
fn extensions_key_usage_bits() {
    // certSign only: bit 5 of the first byte, one unused bit.
    let cert = parse_certificate(&v3_cert(&[extension(
        OID_KEY_USAGE,
        false,
        &ku_value(&[0x04], 1),
    )]))
    .expect("ku parses");
    assert_eq!(cert.key_usage, KEY_USAGE_CERT_SIGN);
    // digitalSignature + keyEncipherment: bits 0 and 2.
    let cert = parse_certificate(&v3_cert(&[extension(
        OID_KEY_USAGE,
        true,
        &ku_value(&[0xA0], 1),
    )]))
    .expect("ku parses");
    assert_eq!(
        cert.key_usage,
        KEY_USAGE_DIGITAL_SIGNATURE | KEY_USAGE_KEY_ENCIPHERMENT
    );
    // decipherOnly lives in the second byte (bit 8).
    let cert = parse_certificate(&v3_cert(&[extension(
        OID_KEY_USAGE,
        false,
        &ku_value(&[0x00, 0x80], 7),
    )]))
    .expect("ku parses");
    assert_eq!(cert.key_usage, KEY_USAGE_DECIPHER_ONLY);
    // Non-BIT-STRING value is invalid.
    assert_eq!(
        err_text(parse_certificate(&v3_cert(&[extension(
            OID_KEY_USAGE,
            false,
            &octet(&[1])
        )]))),
        "x509: invalid key usage"
    );
}

#[test]
fn extensions_basic_constraints() {
    // CA with no path length: -1, zero flag clear.
    let cert = parse_certificate(&v3_cert(&[extension(
        OID_BASIC_CONSTRAINTS,
        true,
        &seq(&boolean(true)),
    )]))
    .expect("bc parses");
    assert!(cert.is_ca);
    assert!(cert.basic_constraints_valid);
    assert_eq!(cert.max_path_len, -1);
    assert!(!cert.max_path_len_zero);
    // Explicit zero path length sets the zero flag.
    let cert = parse_certificate(&v3_cert(&[extension(
        OID_BASIC_CONSTRAINTS,
        true,
        &seq(&concat(&[boolean(true), int_small(0)])),
    )]))
    .expect("bc parses");
    assert!(cert.is_ca);
    assert_eq!(cert.max_path_len, 0);
    assert!(cert.max_path_len_zero);
    // Non-zero path length.
    let cert = parse_certificate(&v3_cert(&[extension(
        OID_BASIC_CONSTRAINTS,
        true,
        &seq(&concat(&[boolean(true), int_small(3)])),
    )]))
    .expect("bc parses");
    assert_eq!(cert.max_path_len, 3);
    assert!(!cert.max_path_len_zero);
    // Empty constraints: not a CA.
    let cert = parse_certificate(&v3_cert(&[extension(
        OID_BASIC_CONSTRAINTS,
        true,
        &seq(&[]),
    )]))
    .expect("bc parses");
    assert!(!cert.is_ca);
    assert!(cert.basic_constraints_valid);
    // Non-SEQUENCE value is invalid.
    assert_eq!(
        err_text(parse_certificate(&v3_cert(&[extension(
            OID_BASIC_CONSTRAINTS,
            true,
            &octet(&[1])
        )]))),
        "x509: invalid basic constraints"
    );
}

#[test]
fn extensions_unhandled_and_duplicates() {
    // Critical unknown extension is recorded dotted.
    let cert = parse_certificate(&v3_cert(&[extension(&[0x2A, 0x03], true, &[1, 2])]))
        .expect("unknown critical parses");
    assert_eq!(cert.unhandled_critical, vec!["1.2.3".to_string()]);
    // Non-critical unknown extension is ignored.
    let cert = parse_certificate(&v3_cert(&[extension(&[0x2A, 0x03], false, &[1, 2])]))
        .expect("unknown non-critical parses");
    assert!(cert.unhandled_critical.is_empty());
    // Duplicate OID quotes the dotted string (Go %q).
    let ku = extension(OID_KEY_USAGE, false, &ku_value(&[0x04], 1));
    assert_eq!(
        err_text(parse_certificate(&v3_cert(&[ku.clone(), ku]))),
        "x509: certificate contains duplicate extension with OID \"2.5.29.15\""
    );
    // Non-SEQUENCE extension element.
    assert_eq!(
        err_text(parse_certificate(&v3_cert(&[int_small(1)]))),
        "x509: malformed extension"
    );
    // Extension fields.
    assert_eq!(
        err_text(parse_certificate(&v3_cert(&[seq(&concat(&[
            octet(&[1]),
            octet(&[2])
        ]))]))),
        "x509: malformed extension OID field"
    );
    assert_eq!(
        err_text(parse_certificate(&v3_cert(&[seq(&concat(&[
            oid(&[0x2A, 0x03]),
            boolean(true),
            int_small(1)
        ]))]))),
        "x509: malformed extension value field"
    );
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
