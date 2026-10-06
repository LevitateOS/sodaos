use super::super::*;
use super::fixtures::{
    boolean, concat, err_text, extension, int_small, ku_value, octet, oid, seq, v3_cert,
    OID_BASIC_CONSTRAINTS, OID_KEY_USAGE,
};

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
