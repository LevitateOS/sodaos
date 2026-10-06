use super::super::*;
use super::fixtures::{
    ai_element, int_small, null, octet, seq, tlv, OID_DSA_SHA256, OID_ECDSA_SHA1, OID_ECDSA_SHA256,
    OID_ED25519, OID_ISO_SHA1_RSA, OID_MD5_RSA, OID_MGF1, OID_PSS, OID_SHA1_RSA, OID_SHA256,
    OID_SHA256_RSA, OID_SHA384, OID_SHA512,
};

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
