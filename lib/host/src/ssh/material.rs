use super::certificate::{cert_inner, parse_cert, CertMaterial};
use super::mpint::{
    marshal_mpint, mpint_bitlen, mpint_cmp, mpint_to_i64, parse_mpint, put_string, read_string,
    Mpint,
};
use super::{
    ALGO_DSS, ALGO_ECDSA256, ALGO_ECDSA384, ALGO_ECDSA521, ALGO_ED25519, ALGO_RSA, ALGO_SKECDSA,
    ALGO_SKED25519,
};
use crate::nist;

/// Parsed public key with its canonical wire bytes.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ParsedKey {
    pub key_type: String,
    pub blob: Vec<u8>,
}

#[derive(Debug, Clone)]
pub enum KeyMaterial {
    Rsa {
        e: Mpint,
        n: Mpint,
    },
    Dss {
        p: Mpint,
        q: Mpint,
        g: Mpint,
        y: Mpint,
    },
    Ecdsa {
        curve: &'static str,
        point: Vec<u8>,
    },
    SkEcdsa {
        point: Vec<u8>,
        app: Vec<u8>,
    },
    Ed25519 {
        key: Vec<u8>,
    },
    SkEd25519 {
        key: Vec<u8>,
        app: Vec<u8>,
    },
    Cert(Box<CertMaterial>),
}

fn curve_for(algo: &str) -> Option<(&'static nist::Curve, &'static str)> {
    match algo {
        ALGO_ECDSA256 => Some((&nist::P256, "nistp256")),
        ALGO_ECDSA384 => Some((&nist::P384, "nistp384")),
        ALGO_ECDSA521 => Some((&nist::P521, "nistp521")),
        _ => None,
    }
}

pub fn parse_key_fields(algo: &str, buf: &[u8]) -> Result<(KeyMaterial, Vec<u8>), ()> {
    match algo {
        ALGO_RSA => {
            let (e, rest) = parse_mpint(buf)?;
            let (n, rest) = parse_mpint(rest)?;
            if mpint_bitlen(&n) > 16384 || mpint_bitlen(&e) > 24 {
                return Err(());
            }
            let ev = mpint_to_i64(&e).ok_or(())?;
            if ev < 3 || ev & 1 == 0 {
                return Err(());
            }
            Ok((KeyMaterial::Rsa { e, n }, rest.to_vec()))
        }
        ALGO_DSS => {
            let (p, rest) = parse_mpint(buf)?;
            let (q, rest) = parse_mpint(rest)?;
            let (g, rest) = parse_mpint(rest)?;
            let (y, rest) = parse_mpint(rest)?;
            if mpint_bitlen(&p) != 1024 || mpint_bitlen(&q) != 160 {
                return Err(());
            }
            if mpint_cmp(&g, &p) != std::cmp::Ordering::Less {
                return Err(());
            }
            let zero = Mpint {
                negative: false,
                mag: Vec::new(),
            };
            if mpint_cmp(&y, &zero) != std::cmp::Ordering::Greater
                || mpint_cmp(&y, &p) != std::cmp::Ordering::Less
            {
                return Err(());
            }
            Ok((KeyMaterial::Dss { p, q, g, y }, rest.to_vec()))
        }
        ALGO_ECDSA256 | ALGO_ECDSA384 | ALGO_ECDSA521 => {
            let (curve_id, rest) = read_string(buf)?;
            let (point, rest) = read_string(rest)?;
            let (curve, expect) = curve_for(algo).ok_or(())?;
            if curve_id != expect.as_bytes() {
                return Err(());
            }
            let point = nist::decode_point(curve, point).ok_or(())?;
            Ok((
                KeyMaterial::Ecdsa {
                    curve: expect,
                    point,
                },
                rest.to_vec(),
            ))
        }
        ALGO_SKECDSA => {
            let (curve_id, rest) = read_string(buf)?;
            let (point, rest) = read_string(rest)?;
            let (app, rest) = read_string(rest)?;
            if curve_id != b"nistp256" {
                return Err(());
            }
            let point = nist::decode_point(&nist::P256, point).ok_or(())?;
            Ok((
                KeyMaterial::SkEcdsa {
                    point,
                    app: app.to_vec(),
                },
                rest.to_vec(),
            ))
        }
        ALGO_ED25519 => {
            let (key, rest) = read_string(buf)?;
            if key.len() != 32 {
                return Err(());
            }
            Ok((KeyMaterial::Ed25519 { key: key.to_vec() }, rest.to_vec()))
        }
        ALGO_SKED25519 => {
            let (key, rest) = read_string(buf)?;
            let (app, rest) = read_string(rest)?;
            if key.len() != 32 {
                return Err(());
            }
            Ok((
                KeyMaterial::SkEd25519 {
                    key: key.to_vec(),
                    app: app.to_vec(),
                },
                rest.to_vec(),
            ))
        }
        _ => {
            let inner = cert_inner(algo).ok_or(())?;
            parse_cert(algo, inner, buf)
        }
    }
}

pub fn marshal_material(out: &mut Vec<u8>, algo: &str, m: &KeyMaterial) {
    put_string(out, algo.as_bytes());
    marshal_fields(out, m);
}

pub fn marshal_fields(out: &mut Vec<u8>, m: &KeyMaterial) {
    match m {
        KeyMaterial::Rsa { e, n } => {
            marshal_mpint(out, e);
            marshal_mpint(out, n);
        }
        KeyMaterial::Dss { p, q, g, y } => {
            marshal_mpint(out, p);
            marshal_mpint(out, q);
            marshal_mpint(out, g);
            marshal_mpint(out, y);
        }
        KeyMaterial::Ecdsa { curve, point } => {
            put_string(out, curve.as_bytes());
            put_string(out, point);
        }
        KeyMaterial::SkEcdsa { point, app } => {
            put_string(out, b"nistp256");
            put_string(out, point);
            put_string(out, app);
        }
        KeyMaterial::Ed25519 { key } => put_string(out, key),
        KeyMaterial::SkEd25519 { key, app } => {
            put_string(out, key);
            put_string(out, app);
        }
        KeyMaterial::Cert(c) => super::certificate::marshal_cert_fields(out, c),
    }
}
