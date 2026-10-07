use super::algorithms::{PublicKeyAlgorithm, SignatureAlgorithm};

#[derive(Debug)]
pub enum PublicKeyData {
    Rsa(rsa::RsaPublicKey),
    Ecdsa224(ecdsa::VerifyingKey<p224::NistP224>),
    Ecdsa256(ecdsa::VerifyingKey<p256::NistP256>),
    Ecdsa384(ecdsa::VerifyingKey<p384::NistP384>),
    Ecdsa521(ecdsa::VerifyingKey<p521::NistP521>),
    Ed25519([u8; 32]),
}

#[derive(Debug)]
pub struct Certificate {
    pub raw: Vec<u8>,
    pub raw_tbs: Vec<u8>,
    pub signature_algorithm: SignatureAlgorithm,
    pub public_key_algorithm: PublicKeyAlgorithm,
    pub public_key: Option<PublicKeyData>,
    pub signature: Vec<u8>,
}
