use super::algorithms::{PublicKeyAlgorithm, SignatureAlgorithm};

/// Parsed public key. ECDSA keys are validated on-curve at parse time, as in
/// Go; Ed25519 keys are length-checked only (validity surfaces at verify).
#[derive(Debug)]
pub enum PublicKeyData {
    Rsa(rsa::RsaPublicKey),
    // Parsed for structural fidelity like Go; verification refuses DSA later.
    #[allow(dead_code)]
    Dsa {
        y: Vec<u8>,
        p: Vec<u8>,
        q: Vec<u8>,
        g: Vec<u8>,
    },
    Ecdsa224(ecdsa::VerifyingKey<p224::NistP224>),
    Ecdsa256(ecdsa::VerifyingKey<p256::NistP256>),
    Ecdsa384(ecdsa::VerifyingKey<p384::NistP384>),
    Ecdsa521(ecdsa::VerifyingKey<p521::NistP521>),
    Ed25519([u8; 32]),
}

/// Parsed certificate: the fields the installer reads, plus raw blobs.
/// Times are absolute Unix seconds (UTCTime/GeneralizedTime carry no nanos).
/// Raw blobs mirror Go's `x509.Certificate` even where uncompared.
#[derive(Debug)]
#[allow(dead_code)]
pub struct Certificate {
    pub raw: Vec<u8>,
    pub raw_tbs: Vec<u8>,
    pub raw_subject_public_key_info: Vec<u8>,
    pub raw_subject: Vec<u8>,
    pub raw_issuer: Vec<u8>,
    pub version: u8,
    pub serial: Vec<u8>,
    pub signature_algorithm: SignatureAlgorithm,
    pub public_key_algorithm: PublicKeyAlgorithm,
    pub public_key: Option<PublicKeyData>,
    pub signature: Vec<u8>,
    pub is_ca: bool,
    pub basic_constraints_valid: bool,
    pub max_path_len: i64,
    pub max_path_len_zero: bool,
    pub key_usage: u16,
    pub not_before: i64,
    pub not_after: i64,
    pub unhandled_critical: Vec<String>,
}
