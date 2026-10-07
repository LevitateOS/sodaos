use super::{
    algorithms::{PublicKeyAlgorithm, SignatureAlgorithm},
    parse_certificate,
    types::{Certificate, PublicKeyData},
    verify_self_signature,
};

struct DeterministicTestRng(u64);

impl rsa::rand_core::RngCore for DeterministicTestRng {
    fn next_u32(&mut self) -> u32 {
        self.next_u64() as u32
    }
    fn next_u64(&mut self) -> u64 {
        self.0 ^= self.0 << 13;
        self.0 ^= self.0 >> 7;
        self.0 ^= self.0 << 17;
        self.0
    }
    fn fill_bytes(&mut self, output: &mut [u8]) {
        for chunk in output.chunks_mut(8) {
            let bytes = self.next_u64().to_le_bytes();
            chunk.copy_from_slice(&bytes[..chunk.len()]);
        }
    }
    fn try_fill_bytes(&mut self, output: &mut [u8]) -> Result<(), rsa::rand_core::Error> {
        self.fill_bytes(output);
        Ok(())
    }
}

impl rsa::rand_core::CryptoRng for DeterministicTestRng {}

const CADDY_ROOT: &[u8] = include_bytes!("fixtures/caddy-2.10.2-root.pem");
const CADDY_DER_SHA256: &str = "b2f9010ee242c635c1e50be5faeb0ec51049d13c2731b68f4d9624a3d1940b61";

#[test]
fn pinned_caddy_root_admits_and_verifies_original_tbs() {
    let der = crate::pemx::decode_certificate(CADDY_ROOT).expect("one public Caddy certificate");
    let certificate = parse_certificate(&der).expect("typed CA admission");
    verify_self_signature(&certificate).expect("original TBS self-signature");
    assert_eq!(
        crate::buildx::sha256_hex(&certificate.raw),
        CADDY_DER_SHA256
    );
}

#[test]
fn pinned_root_refuses_signature_tampering_and_trailing_der() {
    let mut der = crate::pemx::decode_certificate(CADDY_ROOT).unwrap();
    let last = der.last_mut().unwrap();
    *last ^= 1;
    let tampered = parse_certificate(&der).expect("signature bytes remain well formed");
    assert!(verify_self_signature(&tampered).is_err());

    let mut der = crate::pemx::decode_certificate(CADDY_ROOT).unwrap();
    der.push(0);
    assert!(parse_certificate(&der).is_err());
}

#[test]
fn ecdsa_signature_der_stays_strict_for_every_supported_curve() {
    use ecdsa::Signature;

    let canonical = [0x30, 0x07, 0x02, 0x02, 0x00, 0x80, 0x02, 0x01, 0x02];
    assert!(Signature::<p224::NistP224>::from_der(&canonical).is_ok());
    assert!(Signature::<p256::NistP256>::from_der(&canonical).is_ok());
    assert!(Signature::<p384::NistP384>::from_der(&canonical).is_ok());
    assert!(Signature::<p521::NistP521>::from_der(&canonical).is_ok());

    for invalid in [
        &[0x30, 0x08, 0x02, 0x03, 0x00, 0x00, 0x01, 0x02, 0x01, 0x02][..],
        &[0x30, 0x06, 0x02, 0x01, 0xff, 0x02, 0x01, 0x02][..],
        &[0x30, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x00][..],
        &[0x30, 0x80, 0x00, 0x00][..],
    ] {
        assert!(Signature::<p224::NistP224>::from_der(invalid).is_err());
        assert!(Signature::<p256::NistP256>::from_der(invalid).is_err());
        assert!(Signature::<p384::NistP384>::from_der(invalid).is_err());
        assert!(Signature::<p521::NistP521>::from_der(invalid).is_err());
    }
}

#[test]
fn signature_policy_refuses_weak_and_unknown_algorithms() {
    let key = ed25519_dalek::SigningKey::from_bytes(&[3; 32]);
    let certificate = |signature_algorithm| Certificate {
        raw: Vec::new(),
        raw_tbs: b"signature policy".to_vec(),
        signature_algorithm,
        public_key_algorithm: PublicKeyAlgorithm::Ed25519,
        public_key: Some(PublicKeyData::Ed25519(key.verifying_key().to_bytes())),
        signature: vec![0; 64],
    };
    for algorithm in [
        SignatureAlgorithm::Md5Rsa,
        SignatureAlgorithm::Sha1Rsa,
        SignatureAlgorithm::EcdsaSha1,
        SignatureAlgorithm::Unknown,
    ] {
        assert!(verify_self_signature(&certificate(algorithm)).is_err());
    }
}

#[test]
fn rsa_spki_parameters_allow_only_absent_or_null() {
    use x509_cert::der::{asn1::Any, Tag};

    assert!(super::certificate::rsa_parameters_supported(&None));
    let null = Some(Any::new(Tag::Null, Vec::new()).unwrap());
    assert!(super::certificate::rsa_parameters_supported(&null));
    let integer = Some(Any::new(Tag::Integer, vec![1]).unwrap());
    assert!(!super::certificate::rsa_parameters_supported(&integer));
}

#[test]
fn rsa_pss_requires_matching_hash_mgf_and_hash_length_salt() {
    use x509_cert::{
        der::asn1::{Any, ObjectIdentifier},
        spki::AlgorithmIdentifierOwned,
    };

    fn identifier(salt_len: u8) -> AlgorithmIdentifierOwned {
        let parameters = rsa::pkcs1::RsaPssParams::new::<sha2::Sha256>(salt_len);
        AlgorithmIdentifierOwned {
            oid: ObjectIdentifier::new("1.2.840.113549.1.1.10").unwrap(),
            parameters: Some(Any::encode_from(&parameters).unwrap()),
        }
    }
    assert_eq!(
        super::algorithms::signature_algorithm(&identifier(32)),
        SignatureAlgorithm::Sha256Pss
    );
    assert_eq!(
        super::algorithms::signature_algorithm(&identifier(31)),
        SignatureAlgorithm::Unknown
    );
}

#[test]
fn signature_algorithm_parameters_follow_the_typed_der_profile() {
    use x509_cert::{
        der::{
            asn1::{Any, ObjectIdentifier},
            Tag,
        },
        spki::AlgorithmIdentifierOwned,
    };

    fn algorithm(oid: &str, parameters: Option<Any>) -> AlgorithmIdentifierOwned {
        AlgorithmIdentifierOwned {
            oid: ObjectIdentifier::new(oid).unwrap(),
            parameters,
        }
    }
    let null = || Any::new(Tag::Null, Vec::new()).unwrap();
    let integer = || Any::new(Tag::Integer, vec![1]).unwrap();

    for parameters in [None, Some(null())] {
        assert_eq!(
            super::algorithms::signature_algorithm(&algorithm("1.2.840.113549.1.1.11", parameters)),
            SignatureAlgorithm::Sha256Rsa
        );
    }
    assert_eq!(
        super::algorithms::signature_algorithm(&algorithm(
            "1.2.840.113549.1.1.11",
            Some(integer())
        )),
        SignatureAlgorithm::Unknown
    );
    assert_eq!(
        super::algorithms::signature_algorithm(&algorithm("1.2.840.10045.4.3.2", None)),
        SignatureAlgorithm::EcdsaSha256
    );
    assert_eq!(
        super::algorithms::signature_algorithm(&algorithm("1.2.840.10045.4.3.2", Some(null()))),
        SignatureAlgorithm::Unknown
    );
    assert_eq!(
        super::algorithms::signature_algorithm(&algorithm("1.3.101.112", None)),
        SignatureAlgorithm::Ed25519
    );
    assert_eq!(
        super::algorithms::signature_algorithm(&algorithm("1.3.101.112", Some(null()))),
        SignatureAlgorithm::Unknown
    );
}

#[test]
fn duplicate_and_malformed_used_extensions_are_refused() {
    use x509_cert::der::oid::AssociatedOid;
    use x509_cert::{
        der::{Decode, Encode},
        ext::pkix::KeyUsage,
    };

    let der = crate::pemx::decode_certificate(CADDY_ROOT).unwrap();
    let mut typed = x509_cert::Certificate::from_der(&der).unwrap();
    let extensions = typed.tbs_certificate.extensions.as_mut().unwrap();
    extensions.push(extensions[0].clone());
    let duplicate = typed.to_der().unwrap();
    assert!(parse_certificate(&duplicate).is_err());

    let mut typed = x509_cert::Certificate::from_der(&der).unwrap();
    let extensions = typed.tbs_certificate.extensions.as_mut().unwrap();
    let mut unknown = extensions.last().unwrap().clone();
    unknown.extn_id = x509_cert::der::asn1::ObjectIdentifier::new("1.2.3.4").unwrap();
    unknown.critical = true;
    extensions.push(unknown);
    let unsupported_critical = typed.to_der().unwrap();
    assert!(parse_certificate(&unsupported_critical).is_err());

    let mut typed = x509_cert::Certificate::from_der(&der).unwrap();
    let extension = typed
        .tbs_certificate
        .extensions
        .as_mut()
        .unwrap()
        .iter_mut()
        .find(|extension| extension.extn_id == KeyUsage::OID)
        .unwrap();
    extension.extn_value = x509_cert::der::asn1::OctetString::new(vec![0x30, 0]).unwrap();
    let malformed = typed.to_der().unwrap();
    assert!(parse_certificate(&malformed).is_err());
}

#[test]
fn retained_ecdsa_curve_verifiers_check_original_tbs() {
    use ecdsa::signature::hazmat::PrehashSigner as _;
    use sha2::Digest as _;

    macro_rules! verify_curve {
        ($curve:ty, $key_variant:ident, $signature_algorithm:expr, $hash:ty, $width:expr) => {{
            let key = ecdsa::SigningKey::<$curve>::from_slice(&vec![1u8; $width]).unwrap();
            let raw_tbs = b"bounded local CA signature vector".to_vec();
            let digest = <$hash>::digest(&raw_tbs);
            let signature: ecdsa::Signature<$curve> = key.sign_prehash(&digest).unwrap();
            let certificate = Certificate {
                raw: Vec::new(),
                raw_tbs,
                signature_algorithm: $signature_algorithm,
                public_key_algorithm: PublicKeyAlgorithm::Ecdsa,
                public_key: Some(PublicKeyData::$key_variant(key.verifying_key().clone())),
                signature: signature.to_der().as_bytes().to_vec(),
            };
            verify_self_signature(&certificate).unwrap();
        }};
    }

    verify_curve!(
        p224::NistP224,
        Ecdsa224,
        SignatureAlgorithm::EcdsaSha256,
        sha2::Sha256,
        28
    );
    verify_curve!(
        p256::NistP256,
        Ecdsa256,
        SignatureAlgorithm::EcdsaSha256,
        sha2::Sha256,
        32
    );
    verify_curve!(
        p384::NistP384,
        Ecdsa384,
        SignatureAlgorithm::EcdsaSha384,
        sha2::Sha384,
        48
    );

    use ecdsa::signature::hazmat::RandomizedPrehashSigner as _;
    let mut scalar = [0; 66];
    scalar[65] = 1;
    let key = p521::ecdsa::SigningKey::from_slice(&scalar).unwrap();
    let raw_tbs = b"bounded P521 local CA signature vector".to_vec();
    let digest = sha2::Sha512::digest(&raw_tbs);
    let signature = key
        .sign_prehash_with_rng(&mut DeterministicTestRng(0x9e37_79b9_7f4a_7c15), &digest)
        .unwrap();
    let point = p521::ecdsa::VerifyingKey::from(&key).to_encoded_point(false);
    let verifying_key =
        ecdsa::VerifyingKey::<p521::NistP521>::from_sec1_bytes(point.as_bytes()).unwrap();
    let certificate = Certificate {
        raw: Vec::new(),
        raw_tbs,
        signature_algorithm: SignatureAlgorithm::EcdsaSha512,
        public_key_algorithm: PublicKeyAlgorithm::Ecdsa,
        public_key: Some(PublicKeyData::Ecdsa521(verifying_key)),
        signature: signature.to_der().as_bytes().to_vec(),
    };
    verify_self_signature(&certificate).unwrap();
}

#[test]
fn retained_ed25519_verifier_checks_original_tbs() {
    use ed25519_dalek::Signer as _;
    let key = ed25519_dalek::SigningKey::from_bytes(&[7; 32]);
    let raw_tbs = b"bounded Ed25519 local CA signature vector".to_vec();
    let signature = key.sign(&raw_tbs);
    let certificate = Certificate {
        raw: Vec::new(),
        raw_tbs,
        signature_algorithm: SignatureAlgorithm::Ed25519,
        public_key_algorithm: PublicKeyAlgorithm::Ed25519,
        public_key: Some(PublicKeyData::Ed25519(key.verifying_key().to_bytes())),
        signature: signature.to_bytes().to_vec(),
    };
    verify_self_signature(&certificate).unwrap();
}

#[test]
fn retained_rsa_pkcs1_and_pss_verifiers_check_original_tbs() {
    use sha2::Digest as _;

    let mut rng = DeterministicTestRng(0x9e37_79b9_7f4a_7c15);
    let private = rsa::RsaPrivateKey::new(&mut rng, 2048).unwrap();
    let public = rsa::RsaPublicKey::from(&private);
    let raw_tbs = b"bounded RSA local CA signature vector".to_vec();
    let digest = sha2::Sha256::digest(&raw_tbs);
    let signatures = [
        (
            SignatureAlgorithm::Sha256Rsa,
            private
                .sign(rsa::Pkcs1v15Sign::new::<sha2::Sha256>(), &digest)
                .unwrap(),
        ),
        (
            SignatureAlgorithm::Sha256Pss,
            private
                .sign_with_rng(&mut rng, rsa::pss::Pss::new::<sha2::Sha256>(), &digest)
                .unwrap(),
        ),
    ];
    for (signature_algorithm, signature) in signatures {
        let certificate = Certificate {
            raw: Vec::new(),
            raw_tbs: raw_tbs.clone(),
            signature_algorithm,
            public_key_algorithm: PublicKeyAlgorithm::Rsa,
            public_key: Some(PublicKeyData::Rsa(public.clone())),
            signature,
        };
        verify_self_signature(&certificate).unwrap();
    }
}
