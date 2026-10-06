use super::algorithms::{PublicKeyAlgorithm, SignatureAlgorithm, KEY_USAGE_CERT_SIGN};
use super::der::{Reader, TAG_INTEGER, TAG_SEQ};
use super::types::{Certificate, PublicKeyData};

// ---------------------------------------------------------------------------
// CheckSignatureFrom.
// ---------------------------------------------------------------------------

const ERR_UNSUPPORTED_ALGORITHM: &str = "x509: cannot verify signature: algorithm unimplemented";
const ERR_CONSTRAINT_VIOLATION: &str =
    "x509: invalid signature: parent certificate cannot sign this kind of certificate";
const ERR_RSA_VERIFICATION: &str = "crypto/rsa: verification error";
const ERR_ECDSA_FAILURE: &str = "x509: ECDSA verification failure";
const ERR_ED25519_FAILURE: &str = "x509: Ed25519 verification failure";

fn insecure_algorithm_error(algo: SignatureAlgorithm) -> String {
    format!(
        "x509: cannot verify signature: insecure algorithm {}",
        algo.name()
    )
}

fn mismatch_error(expected: PublicKeyAlgorithm, got: &str) -> String {
    format!(
        "x509: signature algorithm specifies an {} public key, but have public key of type {}",
        expected.name(),
        got
    )
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum Hash {
    Md5,
    Sha1,
    Sha256,
    Sha384,
    Sha512,
    None,
}

/// Go's `signatureAlgorithmDetails` row for verification: hash, expected
/// public key algorithm, and the RSA-PSS flag.
fn algorithm_details(algo: SignatureAlgorithm) -> (Hash, PublicKeyAlgorithm, bool) {
    match algo {
        SignatureAlgorithm::Md5Rsa => (Hash::Md5, PublicKeyAlgorithm::Rsa, false),
        SignatureAlgorithm::Sha1Rsa => (Hash::Sha1, PublicKeyAlgorithm::Rsa, false),
        SignatureAlgorithm::Sha256Rsa => (Hash::Sha256, PublicKeyAlgorithm::Rsa, false),
        SignatureAlgorithm::Sha384Rsa => (Hash::Sha384, PublicKeyAlgorithm::Rsa, false),
        SignatureAlgorithm::Sha512Rsa => (Hash::Sha512, PublicKeyAlgorithm::Rsa, false),
        SignatureAlgorithm::Sha256Pss => (Hash::Sha256, PublicKeyAlgorithm::Rsa, true),
        SignatureAlgorithm::Sha384Pss => (Hash::Sha384, PublicKeyAlgorithm::Rsa, true),
        SignatureAlgorithm::Sha512Pss => (Hash::Sha512, PublicKeyAlgorithm::Rsa, true),
        SignatureAlgorithm::DsaSha1 => (Hash::Sha1, PublicKeyAlgorithm::Dsa, false),
        SignatureAlgorithm::DsaSha256 => (Hash::Sha256, PublicKeyAlgorithm::Dsa, false),
        SignatureAlgorithm::EcdsaSha1 => (Hash::Sha1, PublicKeyAlgorithm::Ecdsa, false),
        SignatureAlgorithm::EcdsaSha256 => (Hash::Sha256, PublicKeyAlgorithm::Ecdsa, false),
        SignatureAlgorithm::EcdsaSha384 => (Hash::Sha384, PublicKeyAlgorithm::Ecdsa, false),
        SignatureAlgorithm::EcdsaSha512 => (Hash::Sha512, PublicKeyAlgorithm::Ecdsa, false),
        SignatureAlgorithm::Ed25519 => (Hash::None, PublicKeyAlgorithm::Ed25519, false),
        SignatureAlgorithm::Unknown => (Hash::None, PublicKeyAlgorithm::Unknown, false),
    }
}

fn digest(hash: Hash, signed: &[u8]) -> Vec<u8> {
    use sha2::Digest as _;
    match hash {
        Hash::Sha256 => sha2::Sha256::digest(signed).to_vec(),
        Hash::Sha384 => sha2::Sha384::digest(signed).to_vec(),
        Hash::Sha512 => sha2::Sha512::digest(signed).to_vec(),
        _ => signed.to_vec(),
    }
}

/// Go `ecdsa.VerifyASN1` lenient signature parse: `SEQUENCE { r, s }` with
/// `encoding/asn1` INTEGER semantics (non-minimal accepted, two's
/// complement, empty rejected, exactly two elements). Returns big-endian
/// magnitudes, or `None` for anything Go rejects (including non-positive
/// values, which fail Go's range check downstream).
pub(super) fn parse_ecdsa_signature(sig: &[u8]) -> Option<(Vec<u8>, Vec<u8>)> {
    let mut outer = Reader::new(sig);
    let contents = outer.read_asn1(TAG_SEQ)?;
    if !outer.is_empty() {
        return None;
    }
    let mut reader = Reader::new(contents);
    let r = reader.read_asn1(TAG_INTEGER)?;
    let s = reader.read_asn1(TAG_INTEGER)?;
    if !reader.is_empty() || r.is_empty() || s.is_empty() {
        return None;
    }
    let to_magnitude = |raw: &[u8]| -> Option<Vec<u8>> {
        if raw[0] & 0x80 != 0 {
            // Negative: Go's range check rejects it downstream.
            return None;
        }
        let mut start = 0;
        while start + 1 < raw.len() && raw[start] == 0 {
            start += 1;
        }
        let mag = &raw[start..];
        if mag.iter().all(|b| *b == 0) {
            return None;
        }
        Some(mag.to_vec())
    };
    Some((to_magnitude(r)?, to_magnitude(s)?))
}

/// Left-pad a magnitude to the field length; `None` when it cannot fit
/// (at least `2^fieldlen`, hence `>= n`, which Go rejects).
fn pad_scalar(mag: &[u8], field: usize) -> Option<Vec<u8>> {
    if mag.len() > field {
        return None;
    }
    let mut out = vec![0u8; field];
    out[field - mag.len()..].copy_from_slice(mag);
    Some(out)
}

fn verify_rsa(
    key: &rsa::RsaPublicKey,
    expected: PublicKeyAlgorithm,
    hash: Hash,
    pss: bool,
    hashed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    if expected != PublicKeyAlgorithm::Rsa {
        return Err(mismatch_error(expected, "*rsa.PublicKey"));
    }
    let ok = match (hash, pss) {
        (Hash::Sha256, false) => {
            key.verify(rsa::Pkcs1v15Sign::new::<sha2::Sha256>(), hashed, signature)
        }
        (Hash::Sha384, false) => {
            key.verify(rsa::Pkcs1v15Sign::new::<sha2::Sha384>(), hashed, signature)
        }
        (Hash::Sha512, false) => {
            key.verify(rsa::Pkcs1v15Sign::new::<sha2::Sha512>(), hashed, signature)
        }
        // Go verifies PSS with salt length equal to the hash length.
        (Hash::Sha256, true) => key.verify(rsa::pss::Pss::new::<sha2::Sha256>(), hashed, signature),
        (Hash::Sha384, true) => key.verify(rsa::pss::Pss::new::<sha2::Sha384>(), hashed, signature),
        (Hash::Sha512, true) => key.verify(rsa::pss::Pss::new::<sha2::Sha512>(), hashed, signature),
        _ => return Err(String::from(ERR_UNSUPPORTED_ALGORITHM)),
    };
    ok.map_err(|_| String::from(ERR_RSA_VERIFICATION))
}

fn verify_ecdsa_256(
    key: &ecdsa::VerifyingKey<p256::NistP256>,
    expected: PublicKeyAlgorithm,
    hashed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    use signature::hazmat::PrehashVerifier as _;
    if expected != PublicKeyAlgorithm::Ecdsa {
        return Err(mismatch_error(expected, "*ecdsa.PublicKey"));
    }
    let (r, s) = parse_ecdsa_signature(signature).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?;
    let mut fixed = Vec::with_capacity(64);
    fixed.extend_from_slice(&pad_scalar(&r, 32).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    fixed.extend_from_slice(&pad_scalar(&s, 32).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    let sig = ecdsa::Signature::<p256::NistP256>::from_slice(&fixed)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))?;
    key.verify_prehash(hashed, &sig)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))
}

fn verify_ecdsa_384(
    key: &ecdsa::VerifyingKey<p384::NistP384>,
    expected: PublicKeyAlgorithm,
    hashed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    use signature::hazmat::PrehashVerifier as _;
    if expected != PublicKeyAlgorithm::Ecdsa {
        return Err(mismatch_error(expected, "*ecdsa.PublicKey"));
    }
    let (r, s) = parse_ecdsa_signature(signature).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?;
    let mut fixed = Vec::with_capacity(96);
    fixed.extend_from_slice(&pad_scalar(&r, 48).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    fixed.extend_from_slice(&pad_scalar(&s, 48).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    let sig = ecdsa::Signature::<p384::NistP384>::from_slice(&fixed)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))?;
    key.verify_prehash(hashed, &sig)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))
}

fn verify_ecdsa_521(
    key: &ecdsa::VerifyingKey<p521::NistP521>,
    expected: PublicKeyAlgorithm,
    hashed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    use signature::hazmat::PrehashVerifier as _;
    if expected != PublicKeyAlgorithm::Ecdsa {
        return Err(mismatch_error(expected, "*ecdsa.PublicKey"));
    }
    let (r, s) = parse_ecdsa_signature(signature).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?;
    let mut fixed = Vec::with_capacity(132);
    fixed.extend_from_slice(&pad_scalar(&r, 66).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    fixed.extend_from_slice(&pad_scalar(&s, 66).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    let sig = ecdsa::Signature::<p521::NistP521>::from_slice(&fixed)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))?;
    key.verify_prehash(hashed, &sig)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))
}

fn verify_ecdsa_224(
    key: &ecdsa::VerifyingKey<p224::NistP224>,
    expected: PublicKeyAlgorithm,
    hashed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    use signature::hazmat::PrehashVerifier as _;
    if expected != PublicKeyAlgorithm::Ecdsa {
        return Err(mismatch_error(expected, "*ecdsa.PublicKey"));
    }
    let (r, s) = parse_ecdsa_signature(signature).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?;
    let mut fixed = Vec::with_capacity(56);
    fixed.extend_from_slice(&pad_scalar(&r, 28).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    fixed.extend_from_slice(&pad_scalar(&s, 28).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    let sig = ecdsa::Signature::<p224::NistP224>::from_slice(&fixed)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))?;
    key.verify_prehash(hashed, &sig)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))
}

fn verify_ed25519(
    key: &[u8; 32],
    expected: PublicKeyAlgorithm,
    signed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    use ed25519_dalek::Verifier as _;
    if expected != PublicKeyAlgorithm::Ed25519 {
        return Err(mismatch_error(expected, "ed25519.PublicKey"));
    }
    let vk = ed25519_dalek::VerifyingKey::from_bytes(key)
        .map_err(|_| String::from(ERR_ED25519_FAILURE))?;
    let sig = ed25519_dalek::Signature::try_from(signature)
        .map_err(|_| String::from(ERR_ED25519_FAILURE))?;
    vk.verify(signed, &sig)
        .map_err(|_| String::from(ERR_ED25519_FAILURE))
}

/// Go `(*Certificate).CheckSignatureFrom`: parent constraint checks, then
/// `checkSignature` with SHA-1 disallowed.
pub fn check_signature_from(child: &Certificate, parent: &Certificate) -> Result<(), String> {
    if parent.version == 3 && !parent.basic_constraints_valid
        || parent.basic_constraints_valid && !parent.is_ca
    {
        return Err(String::from(ERR_CONSTRAINT_VIOLATION));
    }
    if parent.key_usage != 0 && parent.key_usage & KEY_USAGE_CERT_SIGN == 0 {
        return Err(String::from(ERR_CONSTRAINT_VIOLATION));
    }
    if parent.public_key_algorithm == PublicKeyAlgorithm::Unknown {
        return Err(String::from(ERR_UNSUPPORTED_ALGORITHM));
    }
    let (hash, expected, pss) = algorithm_details(child.signature_algorithm);
    match hash {
        Hash::Md5 => return Err(insecure_algorithm_error(child.signature_algorithm)),
        Hash::Sha1 => return Err(insecure_algorithm_error(child.signature_algorithm)),
        Hash::None if expected != PublicKeyAlgorithm::Ed25519 => {
            return Err(String::from(ERR_UNSUPPORTED_ALGORITHM))
        }
        _ => {}
    }
    let hashed;
    let signed: &[u8] = if hash == Hash::None {
        &child.raw_tbs
    } else {
        hashed = digest(hash, &child.raw_tbs);
        &hashed
    };
    let key = match parent.public_key.as_ref() {
        Some(key) => key,
        None => return Err(String::from(ERR_UNSUPPORTED_ALGORITHM)),
    };
    match key {
        PublicKeyData::Rsa(key) => verify_rsa(key, expected, hash, pss, signed, &child.signature),
        PublicKeyData::Ecdsa224(key) => verify_ecdsa_224(key, expected, signed, &child.signature),
        PublicKeyData::Ecdsa256(key) => verify_ecdsa_256(key, expected, signed, &child.signature),
        PublicKeyData::Ecdsa384(key) => verify_ecdsa_384(key, expected, signed, &child.signature),
        PublicKeyData::Ecdsa521(key) => verify_ecdsa_521(key, expected, signed, &child.signature),
        PublicKeyData::Ed25519(key) => verify_ed25519(key, expected, signed, &child.signature),
        // Go's key-type switch has no DSA arm: DSA keys always fail here
        // (after the hash gate above).
        PublicKeyData::Dsa { .. } => Err(String::from(ERR_UNSUPPORTED_ALGORITHM)),
    }
}
