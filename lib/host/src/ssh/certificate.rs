use super::material::{parse_key_fields, KeyMaterial};
use super::mpint::{put_string, read_string, read_u32, read_u64};
use super::{
    parse_public_key, ParsedKey, ALGO_DSS, ALGO_ECDSA256, ALGO_ECDSA384, ALGO_ECDSA521,
    ALGO_ED25519, ALGO_RSA, ALGO_SKECDSA, ALGO_SKED25519,
};

pub fn cert_inner(algo: &str) -> Option<&'static str> {
    match algo {
        "ssh-rsa-cert-v01@openssh.com" => Some(ALGO_RSA),
        "ssh-dss-cert-v01@openssh.com" => Some(ALGO_DSS),
        "ecdsa-sha2-nistp256-cert-v01@openssh.com" => Some(ALGO_ECDSA256),
        "ecdsa-sha2-nistp384-cert-v01@openssh.com" => Some(ALGO_ECDSA384),
        "ecdsa-sha2-nistp521-cert-v01@openssh.com" => Some(ALGO_ECDSA521),
        "sk-ecdsa-sha2-nistp256-cert-v01@openssh.com" => Some(ALGO_SKECDSA),
        "ssh-ed25519-cert-v01@openssh.com" => Some(ALGO_ED25519),
        "sk-ssh-ed25519-cert-v01@openssh.com" => Some(ALGO_SKED25519),
        _ => None,
    }
}

fn is_cert_algo(algo: &[u8]) -> bool {
    std::str::from_utf8(algo)
        .map(cert_inner)
        .unwrap_or(None)
        .is_some()
}

#[derive(Debug, Clone)]
pub struct CertMaterial {
    nonce: Vec<u8>,
    inner: KeyMaterial,
    serial: u64,
    cert_type_num: u32,
    key_id: Vec<u8>,
    principals: Vec<Vec<u8>>,
    valid_after: u64,
    valid_before: u64,
    critical_options: Vec<(Vec<u8>, Vec<u8>)>,
    extensions: Vec<(Vec<u8>, Vec<u8>)>,
    reserved: Vec<u8>,
    sig_key: ParsedKey,
    sig_format: Vec<u8>,
    sig_blob: Vec<u8>,
}

type ParsedTuples = (Vec<(Vec<u8>, Vec<u8>)>, ());

fn parse_tuples(mut buf: &[u8]) -> Result<ParsedTuples, ()> {
    let mut out = Vec::new();
    let mut last: Option<Vec<u8>> = None;
    while !buf.is_empty() {
        let (key, rest) = read_string(buf)?;
        if let Some(prev) = &last {
            if key <= prev.as_slice() {
                return Err(());
            }
        }
        last = Some(key.to_vec());
        let (val, rest) = read_string(rest)?;
        buf = rest;
        if val.is_empty() {
            out.push((key.to_vec(), Vec::new()));
        } else {
            let (inner, extra) = read_string(val)?;
            if !extra.is_empty() {
                return Err(());
            }
            out.push((key.to_vec(), inner.to_vec()));
        }
    }
    Ok((out, ()))
}

pub fn parse_cert(
    _algo: &str,
    inner_algo: &'static str,
    buf: &[u8],
) -> Result<(KeyMaterial, Vec<u8>), ()> {
    let (nonce, rest) = read_string(buf)?;
    let (inner, rest_after_inner) = parse_key_fields(inner_algo, rest)?;
    let (serial, rest) = read_u64(&rest_after_inner)?;
    let (cert_type_num, rest) = read_u32(rest)?;
    let (key_id, rest) = read_string(rest)?;
    let (principals_raw, rest) = read_string(rest)?;
    let (valid_after, rest) = read_u64(rest)?;
    let (valid_before, rest) = read_u64(rest)?;
    let (crit_raw, rest) = read_string(rest)?;
    let (ext_raw, rest) = read_string(rest)?;
    let (reserved, rest) = read_string(rest)?;
    let (sig_key_raw, rest) = read_string(rest)?;
    let (sig_raw, rest) = read_string(rest)?;
    let mut principals = Vec::new();
    let mut p = principals_raw;
    while !p.is_empty() {
        let (one, next) = read_string(p)?;
        principals.push(one.to_vec());
        p = next;
    }
    let (critical_options, _) = parse_tuples(crit_raw)?;
    let (extensions, _) = parse_tuples(ext_raw)?;
    let (sig_algo, _) = read_string(sig_key_raw)?;
    if is_cert_algo(sig_algo) {
        return Err(());
    }
    let sig_key = parse_public_key(sig_key_raw).map_err(|_| ())?;
    let (sig_format, sig_rest) = read_string(sig_raw)?;
    let (sig_blob, sig_rest) = read_string(sig_rest)?;
    if !sig_rest.is_empty() {
        return Err(());
    }
    Ok((
        KeyMaterial::Cert(Box::new(CertMaterial {
            nonce: nonce.to_vec(),
            inner,
            serial,
            cert_type_num,
            key_id: key_id.to_vec(),
            principals,
            valid_after,
            valid_before,
            critical_options,
            extensions,
            reserved: reserved.to_vec(),
            sig_key,
            sig_format: sig_format.to_vec(),
            sig_blob: sig_blob.to_vec(),
        })),
        rest.to_vec(),
    ))
}

fn marshal_tuples(out: &mut Vec<u8>, tuples: &[(Vec<u8>, Vec<u8>)]) {
    let mut sorted = tuples.to_vec();
    sorted.sort_by(|a, b| a.0.cmp(&b.0));
    let mut body = Vec::new();
    for (key, val) in &sorted {
        put_string(&mut body, key);
        if val.is_empty() {
            put_string(&mut body, &[]);
        } else {
            let mut inner = Vec::new();
            put_string(&mut inner, val);
            put_string(&mut body, &inner);
        }
    }
    put_string(out, &body);
}

/// Certificate body serialization (the `Cert` arm of field marshaling).
pub fn marshal_cert_fields(out: &mut Vec<u8>, c: &CertMaterial) {
    put_string(out, &c.nonce);
    super::material::marshal_fields(out, &c.inner);
    out.extend_from_slice(&c.serial.to_be_bytes());
    out.extend_from_slice(&c.cert_type_num.to_be_bytes());
    put_string(out, &c.key_id);
    let mut names = Vec::new();
    for p in &c.principals {
        put_string(&mut names, p);
    }
    put_string(out, &names);
    out.extend_from_slice(&c.valid_after.to_be_bytes());
    out.extend_from_slice(&c.valid_before.to_be_bytes());
    marshal_tuples(out, &c.critical_options);
    marshal_tuples(out, &c.extensions);
    put_string(out, &c.reserved);
    put_string(out, &c.sig_key.blob);
    let mut sig = Vec::new();
    put_string(&mut sig, &c.sig_format);
    put_string(&mut sig, &c.sig_blob);
    put_string(out, &sig);
}
