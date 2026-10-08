//! OpenSSH public-key handling for account operations.

mod base64;

#[cfg(test)]
mod tests;

pub use self::base64::{b64_corrupt, b64_decode, b64_decode_go, b64_encode, b64_encode_raw};

use ssh_key::public::KeyData;
use ssh_key::EcdsaCurve;
use ssh_key::{Certificate, PublicKey};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ParsedKey {
    pub key_type: String,
    pub blob: Vec<u8>,
}

pub const ALGO_RSA: &str = "ssh-rsa";
pub const ALGO_DSS: &str = "ssh-dss";
pub const ALGO_ECDSA256: &str = "ecdsa-sha2-nistp256";
pub const ALGO_ECDSA384: &str = "ecdsa-sha2-nistp384";
pub const ALGO_ECDSA521: &str = "ecdsa-sha2-nistp521";
pub const ALGO_SKECDSA: &str = "sk-ecdsa-sha2-nistp256@openssh.com";
pub const ALGO_ED25519: &str = "ssh-ed25519";
pub const ALGO_SKED25519: &str = "sk-ssh-ed25519@openssh.com";

// Enforce SSH's exact-width uncompressed SEC1 profile before validating the
// point with the selected typed curve implementation. ssh-key identifies the
// curve; this gate retains the host's encoding policy.
fn valid_ecdsa_point(curve: EcdsaCurve, point: &[u8]) -> bool {
    let coord_len = match curve {
        EcdsaCurve::NistP256 => 32,
        EcdsaCurve::NistP384 => 48,
        EcdsaCurve::NistP521 => 66,
    };
    if point.len() != 1 + 2 * coord_len || point.first() != Some(&0x04) {
        return false;
    }

    match curve {
        EcdsaCurve::NistP256 => p256::PublicKey::from_sec1_bytes(point).is_ok(),
        EcdsaCurve::NistP384 => p384::PublicKey::from_sec1_bytes(point).is_ok(),
        EcdsaCurve::NistP521 => p521::PublicKey::from_sec1_bytes(point).is_ok(),
    }
}

fn check_key_data(data: &KeyData) -> Result<(), String> {
    match data {
        KeyData::Rsa(key) => {
            let e = key.e().as_positive_bytes().ok_or("invalid public key")?;
            let n = key.n().as_positive_bytes().ok_or("invalid public key")?;
            let e_bits = bit_len(e);
            if e_bits > 24
                || e_bits < 2
                || e.last().is_none_or(|byte| byte & 1 == 0)
                || bit_len(n) > 16_384
            {
                return Err("invalid public key".to_string());
            }
        }
        KeyData::Dsa(key) => {
            let p = key.p().as_positive_bytes().ok_or("invalid public key")?;
            let q = key.q().as_positive_bytes().ok_or("invalid public key")?;
            let g = key.g().as_positive_bytes().ok_or("invalid public key")?;
            let y = key.y().as_positive_bytes().ok_or("invalid public key")?;
            if bit_len(p) != 1024
                || bit_len(q) != 160
                || g.is_empty()
                || y.is_empty()
                || !less_than(g, p)
                || !less_than(y, p)
            {
                return Err("invalid public key".to_string());
            }
        }
        KeyData::Ecdsa(key) => {
            if !valid_ecdsa_point(key.curve(), key.as_sec1_bytes()) {
                return Err("invalid public key".to_string());
            }
        }
        KeyData::SkEcdsaSha2NistP256(key) => {
            if !valid_ecdsa_point(EcdsaCurve::NistP256, key.ec_point().as_bytes()) {
                return Err("invalid public key".to_string());
            }
        }
        KeyData::Certificate(_) | KeyData::Other(_) => {
            return Err("invalid public key".to_string());
        }
        _ => {}
    }
    Ok(())
}

fn bit_len(value: &[u8]) -> usize {
    value.iter().position(|byte| *byte != 0).map_or(0, |i| {
        (value.len() - i - 1) * 8 + (8 - value[i].leading_zeros() as usize)
    })
}

fn less_than(left: &[u8], right: &[u8]) -> bool {
    let left = left
        .iter()
        .position(|byte| *byte != 0)
        .map_or(&[][..], |i| &left[i..]);
    let right = right
        .iter()
        .position(|byte| *byte != 0)
        .map_or(&[][..], |i| &right[i..]);
    left.len() < right.len() || (left.len() == right.len() && left < right)
}

fn parse_wire(blob: &[u8]) -> Result<ParsedKey, String> {
    let name_len = u32::from_be_bytes(
        blob.get(..4)
            .ok_or("invalid public key")?
            .try_into()
            .map_err(|_| "invalid public key")?,
    ) as usize;
    let name_end = 4usize.checked_add(name_len).ok_or("invalid public key")?;
    let name = std::str::from_utf8(blob.get(4..name_end).ok_or("invalid public key")?)
        .map_err(|_| "invalid public key")?;
    if name.ends_with("-cert-v01@openssh.com") {
        let cert = Certificate::from_bytes(blob).map_err(|_| "invalid public key")?;
        check_key_data(cert.public_key())?;
        check_key_data(cert.signature_key())?;
        let key_type = cert.algorithm().to_certificate_type();
        if key_type != name {
            return Err("invalid public key".to_string());
        }
        let serialized = cert.to_bytes().map_err(|_| "invalid public key")?;
        if serialized.as_slice() != blob {
            return Err("invalid public key".to_string());
        }
        return Ok(ParsedKey {
            key_type,
            blob: blob.to_vec(),
        });
    }
    let key = PublicKey::from_bytes(blob).map_err(|_| "invalid public key")?;
    check_key_data(key.key_data())?;
    let key_type = key.algorithm().as_str().to_owned();
    let blob = key.to_bytes().map_err(|_| "invalid public key")?;
    Ok(ParsedKey { key_type, blob })
}

pub fn parse_public_key(blob: &[u8]) -> Result<ParsedKey, String> {
    parse_wire(blob)
}

/// Parse an authorized-key record. Key fields are decoded by ssh-key; this
/// adapter retains only the host's header, line, and options policy.
pub fn parse_authorized_key(input: &[u8]) -> Result<(ParsedKey, bool), String> {
    let text = std::str::from_utf8(input).map_err(|_| "invalid public key")?;
    let mut last_error = "no key found";
    let mut lines = text.lines().peekable();
    while let Some(line) = lines.next() {
        let line = line.split('\r').next().unwrap_or(line).trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<_> = line.split_whitespace().collect();
        let offset = usize::from(
            fields
                .first()
                .is_some_and(|field| !field.is_empty() && field.bytes().all(|byte| byte == b',')),
        );
        let Some((declared, encoded)) = fields.get(offset).zip(fields.get(offset + 1)) else {
            last_error = "invalid public key";
            continue;
        };
        let candidate = match b64_decode(encoded).and_then(|blob| parse_wire(&blob)) {
            Ok(key) if key.key_type == *declared => key,
            _ => {
                last_error = "invalid public key";
                continue;
            }
        };
        if lines.any(|rest| !rest.trim().is_empty()) {
            return Err("invalid public key".to_string());
        }
        return Ok((candidate, false));
    }
    Err(last_error.to_string())
}

pub fn marshal_authorized_key(key_type: &str, blob: &[u8]) -> String {
    format!("{key_type} {}\n", b64_encode(blob))
}

pub fn fingerprint_sha256(blob: &[u8]) -> String {
    let digest = crate::sha256::digest(blob);
    format!("SHA256:{}", b64_encode_raw(&digest))
}
