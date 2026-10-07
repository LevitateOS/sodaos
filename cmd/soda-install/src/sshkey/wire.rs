use elliptic_curve::sec1::ToEncodedPoint;
use ssh_key::public::KeyData;
use ssh_key::{Certificate, PublicKey};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum Kind {
    Rsa,
    Dsa,
    Ecdsa,
    Ed25519,
    SkEcdsa,
    SkEd25519,
    Certificate,
}

#[derive(Debug, Clone)]
pub struct Key {
    algorithm: String,
    kind: Kind,
    wire: Vec<u8>,
}

impl Key {
    pub fn key_type(&self) -> &str {
        &self.algorithm
    }
    pub(super) fn kind(&self) -> Kind {
        self.kind
    }
    pub fn marshal(&self) -> Vec<u8> {
        self.wire.clone()
    }
}

fn validate_point(data: &KeyData) -> Result<(), ()> {
    match data {
        KeyData::Rsa(key) => {
            let e = key.e().as_positive_bytes().ok_or(())?;
            let n = key.n().as_positive_bytes().ok_or(())?;
            let e_bits = bit_len(e);
            if e_bits > 24
                || e_bits < 2
                || e.last().is_none_or(|byte| byte & 1 == 0)
                || bit_len(n) > 16_384
            {
                return Err(());
            }
        }
        KeyData::Ecdsa(key) => {
            let point = key.as_sec1_bytes();
            let canonical = match key.curve() {
                ssh_key::EcdsaCurve::NistP256 if point.len() == 65 && point[0] == 4 => {
                    p256::PublicKey::from_sec1_bytes(point)
                        .map(|p| p.to_encoded_point(false).as_bytes().to_vec())
                        .map_err(|_| ())?
                }
                ssh_key::EcdsaCurve::NistP384 if point.len() == 97 && point[0] == 4 => {
                    p384::PublicKey::from_sec1_bytes(point)
                        .map(|p| p.to_encoded_point(false).as_bytes().to_vec())
                        .map_err(|_| ())?
                }
                ssh_key::EcdsaCurve::NistP521 if point.len() == 133 && point[0] == 4 => {
                    p521::PublicKey::from_sec1_bytes(point)
                        .map(|p| p.to_encoded_point(false).as_bytes().to_vec())
                        .map_err(|_| ())?
                }
                _ => return Err(()),
            };
            if canonical.as_slice() != point {
                return Err(());
            }
        }
        KeyData::SkEcdsaSha2NistP256(key) => {
            let point = key.ec_point().as_bytes();
            if point.len() != 65 || point[0] != 4 {
                return Err(());
            }
            let parsed = p256::PublicKey::from_sec1_bytes(point).map_err(|_| ())?;
            if parsed.to_encoded_point(false).as_bytes() != point {
                return Err(());
            }
        }
        KeyData::Certificate(_) | KeyData::Other(_) => return Err(()),
        _ => {}
    }
    Ok(())
}

fn bit_len(value: &[u8]) -> usize {
    value.iter().position(|byte| *byte != 0).map_or(0, |i| {
        (value.len() - i - 1) * 8 + (8 - value[i].leading_zeros() as usize)
    })
}

pub(super) fn parse_public_key(wire: &[u8]) -> Result<Key, ()> {
    let len = u32::from_be_bytes(wire.get(..4).ok_or(())?.try_into().map_err(|_| ())?) as usize;
    let end = 4usize.checked_add(len).ok_or(())?;
    let name = std::str::from_utf8(wire.get(4..end).ok_or(())?).map_err(|_| ())?;
    if name.ends_with("-cert-v01@openssh.com") {
        let cert = Certificate::from_bytes(wire).map_err(|_| ())?;
        validate_point(cert.public_key())?;
        validate_point(cert.signature_key())?;
        let algorithm = cert.algorithm().to_certificate_type();
        if algorithm != name {
            return Err(());
        }
        let serialized = cert.to_bytes().map_err(|_| ())?;
        if serialized.as_slice() != wire {
            return Err(());
        }
        return Ok(Key {
            algorithm,
            kind: Kind::Certificate,
            wire: wire.to_vec(),
        });
    }
    let key = PublicKey::from_bytes(wire).map_err(|_| ())?;
    validate_point(key.key_data())?;
    let kind = match key.key_data() {
        KeyData::Rsa(_) => Kind::Rsa,
        KeyData::Dsa(_) => Kind::Dsa,
        KeyData::Ecdsa(_) => Kind::Ecdsa,
        KeyData::Ed25519(_) => Kind::Ed25519,
        KeyData::SkEcdsaSha2NistP256(_) => Kind::SkEcdsa,
        KeyData::SkEd25519(_) => Kind::SkEd25519,
        KeyData::Certificate(_) => return Err(()),
        KeyData::Other(_) => return Err(()),
        _ => return Err(()),
    };
    Ok(Key {
        algorithm: key.algorithm().as_str().to_owned(),
        kind,
        wire: key.to_bytes().map_err(|_| ())?,
    })
}
