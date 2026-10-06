use super::super::*;

pub(super) fn tlv(tag: u8, contents: &[u8]) -> Vec<u8> {
    let mut out = vec![tag];
    if contents.len() < 128 {
        out.push(contents.len() as u8);
    } else {
        let mut len = contents.len();
        let mut bytes = Vec::new();
        while len > 0 {
            bytes.push((len & 0xff) as u8);
            len >>= 8;
        }
        out.push(0x80 | bytes.len() as u8);
        bytes.reverse();
        out.extend_from_slice(&bytes);
    }
    out.extend_from_slice(contents);
    out
}

pub(super) fn seq(contents: &[u8]) -> Vec<u8> {
    tlv(0x30, contents)
}

pub(super) fn concat(parts: &[Vec<u8>]) -> Vec<u8> {
    let mut out = Vec::new();
    for part in parts {
        out.extend_from_slice(part);
    }
    out
}

pub(super) fn int_raw(bytes: &[u8]) -> Vec<u8> {
    tlv(0x02, bytes)
}

pub(super) fn int_small(v: u64) -> Vec<u8> {
    if v == 0 {
        return tlv(0x02, &[0]);
    }
    let mut bytes = Vec::new();
    let mut x = v;
    while x > 0 {
        bytes.push((x & 0xff) as u8);
        x >>= 8;
    }
    bytes.reverse();
    if bytes[0] & 0x80 != 0 {
        bytes.insert(0, 0);
    }
    tlv(0x02, &bytes)
}

pub(super) fn oid(body: &[u8]) -> Vec<u8> {
    tlv(0x06, body)
}

pub(super) fn null() -> Vec<u8> {
    vec![0x05, 0x00]
}

pub(super) fn bitstring(payload: &[u8]) -> Vec<u8> {
    let mut contents = vec![0x00];
    contents.extend_from_slice(payload);
    tlv(0x03, &contents)
}

pub(super) fn utctime(s: &str) -> Vec<u8> {
    tlv(0x17, s.as_bytes())
}

pub(super) fn gentime(s: &str) -> Vec<u8> {
    tlv(0x18, s.as_bytes())
}

pub(super) fn octet(contents: &[u8]) -> Vec<u8> {
    tlv(0x04, contents)
}

pub(super) fn boolean(v: bool) -> Vec<u8> {
    tlv(0x01, &[if v { 0xff } else { 0 }])
}

pub(super) const OID_SHA256_RSA: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x0B];
pub(super) const OID_SHA1_RSA: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x05];
pub(super) const OID_MD5_RSA: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x04];
pub(super) const OID_RSA_ENC: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x01];
pub(super) const OID_PSS: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x0A];
pub(super) const OID_MGF1: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x08];
pub(super) const OID_SHA256: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01];
pub(super) const OID_SHA384: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x02];
pub(super) const OID_SHA512: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x03];
pub(super) const OID_ECDSA_SHA256: &[u8] = &[0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x04, 0x03, 0x02];
pub(super) const OID_ECDSA_SHA1: &[u8] = &[0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x04, 0x01];
pub(super) const OID_EC_PUB: &[u8] = &[0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x02, 0x01];
pub(super) const OID_P256: &[u8] = &[0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x03, 0x01, 0x07];
pub(super) const OID_ED25519: &[u8] = &[0x2B, 0x65, 0x70];
pub(super) const OID_DSA: &[u8] = &[0x2A, 0x86, 0x48, 0xCE, 0x38, 0x04, 0x01];
pub(super) const OID_DSA_SHA256: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x03, 0x02];
pub(super) const OID_ISO_SHA1_RSA: &[u8] = &[0x2B, 0x0E, 0x03, 0x02, 0x1D];
pub(super) const OID_KEY_USAGE: &[u8] = &[0x55, 0x1D, 0x0F];
pub(super) const OID_BASIC_CONSTRAINTS: &[u8] = &[0x55, 0x1D, 0x13];

/// Full SEQ element for an AlgorithmIdentifier with optional params.
pub(super) fn ai_element(oid_body: &[u8], params: Option<&[u8]>) -> Vec<u8> {
    let mut contents = oid(oid_body);
    if let Some(p) = params {
        contents.extend_from_slice(p);
    }
    seq(&contents)
}

pub(super) fn rsa_spki(n: &[u8], e: &[u8], params: Option<&[u8]>) -> Vec<u8> {
    let pk_alg = ai_element(OID_RSA_ENC, params);
    let key = seq(&concat(&[int_raw(n), int_raw(e)]));
    seq(&concat(&[pk_alg, bitstring(&key)]))
}

pub(super) fn std_spki() -> Vec<u8> {
    rsa_spki(
        &[0x00, 0xC0, 0xFF, 0xEE],
        &[0x01, 0x00, 0x01],
        Some(&null()),
    )
}

pub(super) fn std_validity() -> Vec<u8> {
    seq(&concat(&[
        utctime("700101000000Z"),
        utctime("700102000000Z"),
    ]))
}

/// The six standard TBS parts: serial, inner AI, issuer, validity,
/// subject, SPKI. Callers may prefix a version or suffix unique IDs.
pub(super) fn std_parts() -> Vec<Vec<u8>> {
    vec![
        int_small(1),
        ai_element(OID_SHA256_RSA, Some(&null())),
        seq(&[]),
        std_validity(),
        seq(&[]),
        std_spki(),
    ]
}

pub(super) fn cert_from_tbs_parts(parts: &[Vec<u8>], outer_ai: &[u8], sig: &[u8]) -> Vec<u8> {
    let tbs = seq(&concat(parts));
    seq(&concat(&[tbs, outer_ai.to_vec(), bitstring(sig)]))
}

pub(super) fn std_cert() -> Vec<u8> {
    cert_from_tbs_parts(
        &std_parts(),
        &ai_element(OID_SHA256_RSA, Some(&null())),
        &[0xAB, 0xCD],
    )
}

pub(super) fn err_text(result: Result<Certificate, String>) -> String {
    result.expect_err("expected parse failure")
}

pub(super) fn spki_with_key(pk_alg: &[u8], key_der: &[u8]) -> Vec<u8> {
    seq(&concat(&[pk_alg.to_vec(), bitstring(key_der)]))
}

pub(super) fn dsa_spki(y: &[u8], params: &[u8]) -> Vec<u8> {
    spki_with_key(&ai_element(OID_DSA, Some(params)), &int_raw(y))
}

pub(super) fn dsa_params(p: &[u8], q: &[u8], g: &[u8]) -> Vec<u8> {
    seq(&concat(&[int_raw(p), int_raw(q), int_raw(g)]))
}

pub(super) fn ec_spki(curve_body: &[u8], point: &[u8]) -> Vec<u8> {
    spki_with_key(&ai_element(OID_EC_PUB, Some(&oid(curve_body))), point)
}

pub(super) fn extension(oid_body: &[u8], critical: bool, value: &[u8]) -> Vec<u8> {
    let mut contents = oid(oid_body);
    if critical {
        contents.extend_from_slice(&boolean(true));
    }
    contents.extend_from_slice(&octet(value));
    seq(&contents)
}

pub(super) fn v3_parts(exts: &[Vec<u8>]) -> Vec<Vec<u8>> {
    let mut parts = std_parts();
    parts.insert(0, tlv(0xA0, &int_small(2)));
    parts.push(tlv(0xA3, &seq(&concat(exts))));
    parts
}

pub(super) fn v3_cert(exts: &[Vec<u8>]) -> Vec<u8> {
    cert_from_tbs_parts(
        &v3_parts(exts),
        &ai_element(OID_SHA256_RSA, Some(&null())),
        &[1],
    )
}

pub(super) fn ku_value(bits: &[u8], unused: u8) -> Vec<u8> {
    let mut contents = vec![unused];
    contents.extend_from_slice(bits);
    tlv(0x03, &contents)
}
