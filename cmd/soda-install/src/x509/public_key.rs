use super::algorithms::{PublicKeyAlgorithm, SignatureAlgorithm};
use super::der::{
    asn1_signed, Reader, CTX0_CONS, CTX1_CONS, CTX2_CONS, CTX3_CONS, OID_CURVE_P224,
    OID_CURVE_P256, OID_CURVE_P384, OID_CURVE_P521, OID_DSA_SHA1, OID_DSA_SHA256, OID_ECDSA_SHA1,
    OID_ECDSA_SHA256, OID_ECDSA_SHA384, OID_ECDSA_SHA512, OID_ED25519_SIG, OID_ISO_SHA1_RSA,
    OID_MD5_RSA, OID_MGF1, OID_PUB_DSA, OID_PUB_ECDSA, OID_PUB_ED25519, OID_PUB_RSA, OID_RSAPSS,
    OID_SHA1_RSA, OID_SHA256, OID_SHA256_RSA, OID_SHA384, OID_SHA384_RSA, OID_SHA512,
    OID_SHA512_RSA, TAG_INTEGER, TAG_SEQ,
};
use super::names::{parse_ai, AlgorithmIdentifier};
use super::types::PublicKeyData;
use super::NULL_BYTES;

// ---------------------------------------------------------------------------
// Signature algorithm mapping.
// ---------------------------------------------------------------------------

/// Lenient `encoding/asn1` INTEGER into `i64`: two's complement, 1-8 bytes.
fn lenient_asn1_int(bytes: &[u8]) -> Option<i64> {
    if bytes.is_empty() || bytes.len() > 8 {
        return None;
    }
    asn1_signed(bytes)
}

/// Explicitly-tagged inner `AlgorithmIdentifier` (PSS hash/MGF buckets).
fn parse_inner_ai(bytes: &[u8]) -> Option<AlgorithmIdentifier<'_>> {
    let mut inner = Reader::new(bytes);
    let seq = inner.read_asn1(TAG_SEQ)?;
    if !inner.is_empty() {
        return None;
    }
    parse_ai(seq).ok()
}

/// Go `getSignatureAlgorithmFromAI` for RSA-PSS parameters: the parameters
/// must unmarshal as `{hash, mgf, saltLength, trailer}` with MGF1 matching
/// the message hash, salt length equal to the hash length, and default
/// trailer. Anything else yields `Unknown`.
fn pss_algorithm(params: &[u8]) -> SignatureAlgorithm {
    let mut outer = Reader::new(params);
    let contents = match outer.read_asn1(TAG_SEQ) {
        Some(c) if outer.is_empty() => c,
        _ => return SignatureAlgorithm::Unknown,
    };
    let mut reader = Reader::new(contents);
    let hash = match reader.read_asn1(CTX0_CONS) {
        Some(c) => c,
        None => return SignatureAlgorithm::Unknown,
    };
    let mgf = match reader.read_asn1(CTX1_CONS) {
        Some(c) => c,
        None => return SignatureAlgorithm::Unknown,
    };
    let salt = match reader.read_asn1(CTX2_CONS) {
        Some(c) => c,
        None => return SignatureAlgorithm::Unknown,
    };
    let mut trailer: i64 = 1;
    if reader.peek_tag(CTX3_CONS) {
        let bytes = match reader.read_asn1(CTX3_CONS) {
            Some(c) => c,
            None => return SignatureAlgorithm::Unknown,
        };
        let mut inner = Reader::new(bytes);
        let raw = match inner.read_asn1(TAG_INTEGER) {
            Some(c) if inner.is_empty() => c,
            _ => return SignatureAlgorithm::Unknown,
        };
        trailer = match lenient_asn1_int(raw) {
            Some(v) => v,
            None => return SignatureAlgorithm::Unknown,
        };
    }
    if !reader.is_empty() {
        return SignatureAlgorithm::Unknown;
    }
    let hash_ai = match parse_inner_ai(hash) {
        Some(ai) => ai,
        None => return SignatureAlgorithm::Unknown,
    };
    let mgf_ai = match parse_inner_ai(mgf) {
        Some(ai) => ai,
        None => return SignatureAlgorithm::Unknown,
    };
    let mut salt_reader = Reader::new(salt);
    let salt_raw = match salt_reader.read_asn1(TAG_INTEGER) {
        Some(c) if salt_reader.is_empty() => c,
        _ => return SignatureAlgorithm::Unknown,
    };
    let salt_len = match lenient_asn1_int(salt_raw) {
        Some(v) => v,
        None => return SignatureAlgorithm::Unknown,
    };
    let mgf_hash = match parse_inner_ai(mgf_ai.params) {
        Some(ai) => ai,
        None => return SignatureAlgorithm::Unknown,
    };
    let null_or_absent = |params: &[u8]| params.is_empty() || params == NULL_BYTES;
    if !null_or_absent(hash_ai.params)
        || mgf_ai.oid != OID_MGF1
        || mgf_hash.oid != hash_ai.oid
        || !null_or_absent(mgf_hash.params)
        || trailer != 1
    {
        return SignatureAlgorithm::Unknown;
    }
    if hash_ai.oid == OID_SHA256 && salt_len == 32 {
        SignatureAlgorithm::Sha256Pss
    } else if hash_ai.oid == OID_SHA384 && salt_len == 48 {
        SignatureAlgorithm::Sha384Pss
    } else if hash_ai.oid == OID_SHA512 && salt_len == 64 {
        SignatureAlgorithm::Sha512Pss
    } else {
        SignatureAlgorithm::Unknown
    }
}

/// Go `getSignatureAlgorithmFromAI`. RSA/ECDSA OIDs map regardless of
/// parameters; Ed25519 requires absent parameters; PSS parses its bucket.
pub(super) fn signature_algorithm_from_ai(ai: &AlgorithmIdentifier<'_>) -> SignatureAlgorithm {
    if ai.oid == OID_ED25519_SIG {
        if !ai.params.is_empty() {
            return SignatureAlgorithm::Unknown;
        }
        return SignatureAlgorithm::Ed25519;
    }
    if ai.oid == OID_RSAPSS {
        return pss_algorithm(ai.params);
    }
    if ai.oid == OID_MD5_RSA {
        SignatureAlgorithm::Md5Rsa
    } else if ai.oid == OID_SHA1_RSA || ai.oid == OID_ISO_SHA1_RSA {
        SignatureAlgorithm::Sha1Rsa
    } else if ai.oid == OID_SHA256_RSA {
        SignatureAlgorithm::Sha256Rsa
    } else if ai.oid == OID_SHA384_RSA {
        SignatureAlgorithm::Sha384Rsa
    } else if ai.oid == OID_SHA512_RSA {
        SignatureAlgorithm::Sha512Rsa
    } else if ai.oid == OID_DSA_SHA1 {
        SignatureAlgorithm::DsaSha1
    } else if ai.oid == OID_DSA_SHA256 {
        SignatureAlgorithm::DsaSha256
    } else if ai.oid == OID_ECDSA_SHA1 {
        SignatureAlgorithm::EcdsaSha1
    } else if ai.oid == OID_ECDSA_SHA256 {
        SignatureAlgorithm::EcdsaSha256
    } else if ai.oid == OID_ECDSA_SHA384 {
        SignatureAlgorithm::EcdsaSha384
    } else if ai.oid == OID_ECDSA_SHA512 {
        SignatureAlgorithm::EcdsaSha512
    } else {
        SignatureAlgorithm::Unknown
    }
}

pub(super) fn public_key_algorithm_from_oid(oid: &[u8]) -> PublicKeyAlgorithm {
    if oid == OID_PUB_RSA {
        PublicKeyAlgorithm::Rsa
    } else if oid == OID_PUB_DSA {
        PublicKeyAlgorithm::Dsa
    } else if oid == OID_PUB_ECDSA {
        PublicKeyAlgorithm::Ecdsa
    } else if oid == OID_PUB_ED25519 {
        PublicKeyAlgorithm::Ed25519
    } else {
        PublicKeyAlgorithm::Unknown
    }
}

// ---------------------------------------------------------------------------
// Public keys.
// ---------------------------------------------------------------------------

/// Field primes, big-endian, for Go's coordinate-range error. Boundary
/// behavior is covered by differential tests against the Go oracle.
const P224_PRIME: [u8; 28] = [
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
];
pub(super) const P256_PRIME: [u8; 32] = [
    0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
];
const P384_PRIME: [u8; 48] = [
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe, 0xff, 0xff, 0xff, 0xff,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
];
const P521_PRIME: [u8; 66] = [
    0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff,
];

/// Go's P-256 coordinate error is platform dependent (assembly wording on
/// amd64/arm64/ppc64le/s390x, fiat wording elsewhere, `purego` excluded).
#[cfg(any(
    target_arch = "x86_64",
    target_arch = "aarch64",
    target_arch = "powerpc64",
    target_arch = "s390x"
))]
pub(super) const P256_ELEMENT_ERROR: &str = "invalid P256 element encoding";
#[cfg(not(any(
    target_arch = "x86_64",
    target_arch = "aarch64",
    target_arch = "powerpc64",
    target_arch = "s390x"
)))]
pub(super) const P256_ELEMENT_ERROR: &str = "invalid P256Element encoding";

pub(super) fn magnitude(bytes: &[u8]) -> &[u8] {
    if bytes.len() > 1 && bytes[0] == 0 {
        &bytes[1..]
    } else {
        bytes
    }
}

pub(super) fn is_negative(bytes: &[u8]) -> bool {
    !bytes.is_empty() && bytes[0] & 0x80 != 0
}

fn is_zero(bytes: &[u8]) -> bool {
    bytes.iter().all(|b| *b == 0)
}

fn parse_rsa_key(params: &[u8], data: &[u8]) -> Result<PublicKeyData, String> {
    if params != NULL_BYTES {
        return Err(String::from("x509: RSA key missing NULL parameters"));
    }
    let mut reader = Reader::new(data);
    let contents = reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid RSA public key"))?;
    let mut reader = Reader::new(contents);
    let n = reader
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid RSA modulus"))?;
    let e = reader
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid RSA public exponent"))?;
    let e = asn1_signed(e).ok_or_else(|| String::from("x509: invalid RSA public exponent"))?;
    if is_negative(n) || is_zero(n) {
        return Err(String::from("x509: RSA modulus is not a positive number"));
    }
    if e <= 0 {
        return Err(String::from(
            "x509: RSA public exponent is not a positive number",
        ));
    }
    let key = rsa::RsaPublicKey::new_unchecked(
        rsa::BigUint::from_bytes_be(magnitude(n)),
        rsa::BigUint::from(e as u64),
    );
    Ok(PublicKeyData::Rsa(key))
}

pub(super) fn parse_dsa_key(params: &[u8], data: &[u8]) -> Result<PublicKeyData, String> {
    let mut reader = Reader::new(data);
    let y = reader
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid DSA public key"))?;
    let mut params = Reader::new(params);
    let contents = params
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid DSA parameters"))?;
    let mut params = Reader::new(contents);
    let p = params
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid DSA parameters"))?;
    let q = params
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid DSA parameters"))?;
    let g = params
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid DSA parameters"))?;
    for value in [y, p, q, g] {
        if is_negative(value) || is_zero(value) {
            return Err(String::from("x509: zero or negative DSA parameter"));
        }
    }
    Ok(PublicKeyData::Dsa {
        y: magnitude(y).to_vec(),
        p: magnitude(p).to_vec(),
        q: magnitude(q).to_vec(),
        g: magnitude(g).to_vec(),
    })
}

/// Validated curve point length/prefix/range shared by all four curves.
/// Returns the coordinate field length on success.
fn check_ec_point(
    data: &[u8],
    prime: &[u8],
    point_error: &str,
    element_error: &str,
) -> Result<usize, String> {
    if data.is_empty() || data[0] != 4 {
        return Err(String::from("ecdsa: invalid uncompressed public key"));
    }
    let field = prime.len();
    if data.len() != 1 + 2 * field {
        return Err(String::from(point_error));
    }
    let (x, y) = (&data[1..1 + field], &data[1 + field..]);
    if x >= prime || y >= prime {
        return Err(String::from(element_error));
    }
    Ok(field)
}

fn parse_ecdsa_key(params: &[u8], data: &[u8]) -> Result<PublicKeyData, String> {
    let mut reader = Reader::new(params);
    let curve = reader
        .read_oid()
        .ok_or_else(|| String::from("x509: invalid ECDSA parameters"))?;
    if curve == OID_CURVE_P224 {
        check_ec_point(
            data,
            &P224_PRIME,
            "invalid P224 point encoding",
            "invalid P224Element encoding",
        )?;
        let key = ecdsa::VerifyingKey::<p224::NistP224>::from_sec1_bytes(data)
            .map_err(|_| String::from("P224 point not on curve"))?;
        Ok(PublicKeyData::Ecdsa224(key))
    } else if curve == OID_CURVE_P256 {
        check_ec_point(
            data,
            &P256_PRIME,
            "invalid P256 point encoding",
            P256_ELEMENT_ERROR,
        )?;
        let key = ecdsa::VerifyingKey::<p256::NistP256>::from_sec1_bytes(data)
            .map_err(|_| String::from("P256 point not on curve"))?;
        Ok(PublicKeyData::Ecdsa256(key))
    } else if curve == OID_CURVE_P384 {
        check_ec_point(
            data,
            &P384_PRIME,
            "invalid P384 point encoding",
            "invalid P384Element encoding",
        )?;
        let key = ecdsa::VerifyingKey::<p384::NistP384>::from_sec1_bytes(data)
            .map_err(|_| String::from("P384 point not on curve"))?;
        Ok(PublicKeyData::Ecdsa384(key))
    } else if curve == OID_CURVE_P521 {
        check_ec_point(
            data,
            &P521_PRIME,
            "invalid P521 point encoding",
            "invalid P521Element encoding",
        )?;
        let key = ecdsa::VerifyingKey::<p521::NistP521>::from_sec1_bytes(data)
            .map_err(|_| String::from("P521 point not on curve"))?;
        Ok(PublicKeyData::Ecdsa521(key))
    } else {
        Err(String::from("x509: unsupported elliptic curve"))
    }
}

pub(super) fn parse_ed25519_key(params: &[u8], data: &[u8]) -> Result<PublicKeyData, String> {
    if !params.is_empty() {
        return Err(String::from(
            "x509: Ed25519 key encoded with illegal parameters",
        ));
    }
    if data.len() != 32 {
        return Err(String::from("x509: wrong Ed25519 public key size"));
    }
    let mut key = [0u8; 32];
    key.copy_from_slice(data);
    Ok(PublicKeyData::Ed25519(key))
}

pub(super) fn parse_public_key(
    algorithm: PublicKeyAlgorithm,
    params: &[u8],
    data: &[u8],
) -> Result<PublicKeyData, String> {
    match algorithm {
        PublicKeyAlgorithm::Rsa => parse_rsa_key(params, data),
        PublicKeyAlgorithm::Dsa => parse_dsa_key(params, data),
        PublicKeyAlgorithm::Ecdsa => parse_ecdsa_key(params, data),
        PublicKeyAlgorithm::Ed25519 => parse_ed25519_key(params, data),
        PublicKeyAlgorithm::Unknown => Err(String::from("x509: unknown public key algorithm")),
    }
}
