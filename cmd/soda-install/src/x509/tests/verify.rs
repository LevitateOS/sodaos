use super::super::*;
use super::fixtures::{
    ai_element, bitstring, boolean, cert_from_tbs_parts, concat, dsa_params, dsa_spki, ec_spki,
    extension, int_raw, int_small, ku_value, null, seq, spki_with_key, std_parts, std_spki, tlv,
    v3_cert, v3_parts, OID_BASIC_CONSTRAINTS, OID_DSA_SHA256, OID_ECDSA_SHA1, OID_ECDSA_SHA256,
    OID_ED25519, OID_KEY_USAGE, OID_MD5_RSA, OID_P256, OID_SHA1_RSA, OID_SHA256_RSA,
};

#[test]
fn ecdsa_signature_der_is_strict_on_all_curves() {
    use ecdsa::Signature;

    let valid = seq(&concat(&[int_raw(&[0x00, 0x80]), int_small(2)]));
    assert!(Signature::<p224::NistP224>::from_der(&valid).is_ok());
    assert!(Signature::<p256::NistP256>::from_der(&valid).is_ok());
    assert!(Signature::<p384::NistP384>::from_der(&valid).is_ok());
    assert!(Signature::<p521::NistP521>::from_der(&valid).is_ok());

    // Redundant sign bytes, negative/zero values, empty integers, and bad
    // arity are rejected by the typed DER parser.
    for invalid in [
        seq(&concat(&[int_raw(&[0x00, 0x00, 0x01]), int_small(2)])),
        seq(&concat(&[int_raw(&[0xff]), int_small(2)])),
        seq(&concat(&[int_small(1), int_raw(&[0x00])])),
        seq(&concat(&[tlv(0x02, &[]), int_small(2)])),
        seq(&concat(&[int_small(1)])),
        seq(&concat(&[int_small(1), int_small(2), int_small(3)])),
        vec![0x30, 0x81, 0x06, 0x02, 1, 1, 0x02, 1, 2],
        vec![0x04, 0x02, 0x01, 0x02],
    ] {
        assert!(Signature::<p224::NistP224>::from_der(&invalid).is_err());
        assert!(Signature::<p256::NistP256>::from_der(&invalid).is_err());
        assert!(Signature::<p384::NistP384>::from_der(&invalid).is_err());
        assert!(Signature::<p521::NistP521>::from_der(&invalid).is_err());
    }

    for (width, curve) in [(28, 224), (32, 256), (48, 384), (66, 521)] {
        let mut oversized = vec![0; width + 1];
        oversized[1..].fill(0xff);
        let signature = seq(&concat(&[tlv(0x02, &oversized), int_small(1)]));
        match curve {
            224 => assert!(Signature::<p224::NistP224>::from_der(&signature).is_err()),
            256 => assert!(Signature::<p256::NistP256>::from_der(&signature).is_err()),
            384 => assert!(Signature::<p384::NistP384>::from_der(&signature).is_err()),
            _ => assert!(Signature::<p521::NistP521>::from_der(&signature).is_err()),
        }
    }

    let mut trailing = valid.clone();
    trailing.push(0);
    assert!(Signature::<p224::NistP224>::from_der(&trailing).is_err());
    assert!(Signature::<p256::NistP256>::from_der(&trailing).is_err());
    assert!(Signature::<p384::NistP384>::from_der(&trailing).is_err());
    assert!(Signature::<p521::NistP521>::from_der(&trailing).is_err());
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

fn signed_p256_ca(scalar: [u8; 32]) -> Certificate {
    use signature::Signer as _;

    let signing = p256::ecdsa::SigningKey::from_bytes(&scalar.into()).expect("synthetic key");
    let algorithm = ai_element(OID_ECDSA_SHA256, None);
    let point = signing.verifying_key().to_encoded_point(false);
    let mut parts = v3_parts(&[extension(OID_BASIC_CONSTRAINTS, true, &seq(&boolean(true)))]);
    parts[2] = algorithm.clone();
    parts[6] = ec_spki(OID_P256, point.as_bytes());
    let tbs = seq(&concat(&parts));
    let signature: p256::ecdsa::Signature = signing.sign(&tbs);
    let signature_der = signature.to_der();
    parse_certificate(&seq(&concat(&[
        tbs,
        algorithm,
        bitstring(signature_der.as_bytes()),
    ])))
    .expect("synthetic signed CA parses")
}

const OID_P224: &[u8] = &[0x2b, 0x81, 0x04, 0x00, 0x21];
const OID_P384: &[u8] = &[0x2b, 0x81, 0x04, 0x00, 0x22];
const OID_P521: &[u8] = &[0x2b, 0x81, 0x04, 0x00, 0x23];
const OID_ECDSA_SHA512: &[u8] = &[0x2a, 0x86, 0x48, 0xce, 0x3d, 0x04, 0x03, 0x04];

fn signed_ecdsa_ca(
    curve_oid: &[u8],
    signature_oid: &[u8],
    point: &[u8],
    sign_prehash: impl FnOnce(&[u8]) -> Vec<u8>,
) -> Certificate {
    let algorithm = ai_element(signature_oid, None);
    let mut parts = v3_parts(&[extension(OID_BASIC_CONSTRAINTS, true, &seq(&boolean(true)))]);
    parts[2] = algorithm.clone();
    parts[6] = ec_spki(curve_oid, point);
    let tbs = seq(&concat(&parts));
    let signature_der = sign_prehash(&tbs);
    parse_certificate(&seq(&concat(&[tbs, algorithm, bitstring(&signature_der)])))
        .expect("synthetic signed CA parses")
}

macro_rules! signed_ecdsa_ca_case {
    ($name:ident, $curve:ty, $signing_key:ty, $verifying_key:ty, $curve_oid:expr, $width:expr, $signature_oid:expr, $digest:ty) => {
        fn $name() -> Certificate {
            use sha2::Digest as _;
            use signature::hazmat::PrehashSigner as _;

            let mut scalar = [0u8; $width];
            scalar[$width - 1] = 1;
            let signing = <$signing_key>::from_slice(&scalar).expect("synthetic ECDSA key");
            let point = <$verifying_key>::from(&signing).to_encoded_point(false);
            signed_ecdsa_ca($curve_oid, $signature_oid, point.as_bytes(), |tbs| {
                let prehash = <$digest>::digest(tbs);
                let signature: ecdsa::Signature<$curve> = signing
                    .sign_prehash(&prehash)
                    .expect("synthetic ECDSA signature");
                signature.to_der().as_bytes().to_vec()
            })
        }
    };
}

signed_ecdsa_ca_case!(
    signed_p224_ca,
    p224::NistP224,
    ecdsa::SigningKey<p224::NistP224>,
    ecdsa::VerifyingKey<p224::NistP224>,
    OID_P224,
    28,
    OID_ECDSA_SHA256,
    sha2::Sha256
);
signed_ecdsa_ca_case!(
    signed_p384_ca,
    p384::NistP384,
    ecdsa::SigningKey<p384::NistP384>,
    ecdsa::VerifyingKey<p384::NistP384>,
    OID_P384,
    48,
    OID_ECDSA_SHA256,
    sha2::Sha256
);
signed_ecdsa_ca_case!(
    signed_p521_ca,
    p521::NistP521,
    p521::ecdsa::SigningKey,
    p521::ecdsa::VerifyingKey,
    OID_P521,
    66,
    OID_ECDSA_SHA512,
    sha2::Sha512
);

#[test]
fn ecdsa_raw_tbs_verification_round_trips_all_curves() {
    let cases = [
        ("P-224", signed_p224_ca()),
        (
            "P-256",
            signed_p256_ca({
                let mut scalar = [0u8; 32];
                scalar[31] = 1;
                scalar
            }),
        ),
        ("P-384", signed_p384_ca()),
        ("P-521", signed_p521_ca()),
    ];
    for (name, certificate) in cases {
        check_signature_from(&certificate, &certificate)
            .unwrap_or_else(|error| panic!("{name} raw-TBS signature failed: {error}"));
    }
}

#[test]
fn ecdsa_uses_original_tbs_and_rejects_wrong_key_or_tampering() {
    let child = signed_p256_ca({
        let mut scalar = [0u8; 32];
        scalar[31] = 1;
        scalar
    });
    check_signature_from(&child, &child).expect("self-signed P-256 CA verifies");

    let wrong_parent = signed_p256_ca({
        let mut scalar = [0u8; 32];
        scalar[31] = 2;
        scalar
    });
    assert_eq!(
        check_signature_from(&child, &wrong_parent),
        Err(String::from("x509: ECDSA verification failure"))
    );

    let mut tampered = signed_p256_ca({
        let mut scalar = [0u8; 32];
        scalar[31] = 1;
        scalar
    });
    let last = tampered.raw_tbs.len() - 1;
    tampered.raw_tbs[last] ^= 1;
    assert_eq!(
        check_signature_from(&tampered, &child),
        Err(String::from("x509: ECDSA verification failure"))
    );
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
