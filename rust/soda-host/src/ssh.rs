//! OpenSSH public-key handling for account operations.
//!
//! Mirrors `golang.org/x/crypto/ssh` as used by the project account paths
//! (`canonicalKeys`, `canonicalizeAccountKeys`, `/connection`):
//!
//! * authorized-keys line scanning: multi-line tolerant scan, `#`/blank
//!   lines skipped, carriage returns cut, declared type must equal the
//!   embedded blob type, options detected (callers reject them), trailing
//!   data after the accepted line must be blank;
//! * key types: ssh-rsa, ssh-dss, ecdsa-sha2-nistp* (with on-curve
//!   validation), sk-ecdsa-sha2-nistp256, ssh-ed25519,
//!   sk-ssh-ed25519, and the eight `-cert-v01@openssh.com` variants;
//! * canonical re-marshaling (signed-minimal mpints, sorted cert tuples)
//!   so byte comparison enforces canonical form;
//! * `SHA256:` fingerprints and newline-terminated marshaling.
//!
//! Whitespace trimming covers ASCII plus vertical tab/form feed; exotic
//! Unicode spaces (which Go's `TrimSpace` would also trim) are not treated
//! as blank. Key material is ASCII, so the boundary is identical in
//! practice.

use crate::nist;

/// `bytes.TrimSpace`: Unicode White_Space on both ends. An invalid byte
/// decodes as a non-space rune and stops the trim, exactly like Go.
fn trim_ws(mut v: &[u8]) -> &[u8] {
    while !v.is_empty() {
        let head = match std::str::from_utf8(v) {
            Ok(s) => s,
            Err(e) => {
                if e.valid_up_to() == 0 {
                    break;
                }
                // valid_up_to is a char boundary of the valid prefix.
                match std::str::from_utf8(&v[..e.valid_up_to()]) {
                    Ok(s) => s,
                    Err(_) => break,
                }
            }
        };
        match head.chars().next() {
            Some(c) if c.is_whitespace() => v = &v[c.len_utf8()..],
            _ => break,
        }
    }
    while !v.is_empty() {
        // Extend a valid suffix backward to find the last rune.
        let mut last: Option<char> = None;
        for i in 1..=4.min(v.len()) {
            if let Ok(s) = std::str::from_utf8(&v[v.len() - i..]) {
                last = s.chars().last();
                break;
            }
        }
        match last {
            Some(c) if c.is_whitespace() => v = &v[..v.len() - c.len_utf8()],
            _ => break,
        }
    }
    v
}

// ---------- base64 (strict standard alphabet) ----------

fn b64_value(c: u8) -> Option<u8> {
    match c {
        b'A'..=b'Z' => Some(c - b'A'),
        b'a'..=b'z' => Some(c - b'a' + 26),
        b'0'..=b'9' => Some(c - b'0' + 52),
        b'+' => Some(62),
        b'/' => Some(63),
        _ => None,
    }
}

pub fn b64_decode(s: &str) -> Result<Vec<u8>, String> {
    let b = s.as_bytes();
    if b.is_empty() || b.len() % 4 != 0 {
        return Err("invalid base64".to_string());
    }
    let mut out = Vec::with_capacity(b.len() / 4 * 3);
    let mut pad = 0;
    for (i, chunk) in b.chunks(4).enumerate() {
        let last = i == b.len() / 4 - 1;
        let mut vals = [0u8; 4];
        for (j, &c) in chunk.iter().enumerate() {
            if c == b'=' {
                if !last || j < 2 {
                    return Err("invalid base64".to_string());
                }
                pad += 1;
                vals[j] = 0;
            } else {
                if pad > 0 {
                    return Err("invalid base64".to_string());
                }
                vals[j] = b64_value(c).ok_or_else(|| "invalid base64".to_string())?;
            }
        }
        if pad > 2 {
            return Err("invalid base64".to_string());
        }
        let n = (u32::from(vals[0]) << 18)
            | (u32::from(vals[1]) << 12)
            | (u32::from(vals[2]) << 6)
            | u32::from(vals[3]);
        out.push((n >> 16) as u8);
        if pad < 2 {
            out.push((n >> 8) as u8);
        }
        if pad == 0 {
            out.push(n as u8);
        }
    }
    Ok(out)
}

/// One `decodeQuantum` step of `base64.StdEncoding.Decode` (pinned Go
/// toolchain): `\r`/`\n` skipped, `CorruptInputError` offsets exact.
/// Returns the new read index; partial output on error is discarded by
/// every caller, exactly like `encoding/json` discards it.
fn decode_quantum_go(src: &[u8], mut si: usize, out: &mut Vec<u8>) -> Result<usize, usize> {
    let mut dbuf = [0u8; 4];
    let mut dlen = 4usize;
    let mut j = 0usize;
    while j < 4 {
        if src.len() == si {
            if j == 0 {
                return Ok(si);
            }
            return Err(si - j);
        }
        let b = src[si];
        si += 1;
        if let Some(v) = b64_value(b) {
            dbuf[j] = v;
            j += 1;
            continue;
        }
        if b == b'\n' || b == b'\r' {
            continue;
        }
        if b != b'=' {
            return Err(si - 1);
        }
        match j {
            0 | 1 => return Err(si - 1),
            2 => {
                while si < src.len() && (src[si] == b'\n' || src[si] == b'\r') {
                    si += 1;
                }
                if si == src.len() {
                    return Err(src.len());
                }
                if src[si] != b'=' {
                    return Err(si - 1);
                }
                si += 1;
            }
            _ => {}
        }
        while si < src.len() && (src[si] == b'\n' || src[si] == b'\r') {
            si += 1;
        }
        if si < src.len() {
            return Err(si);
        }
        dlen = j;
        break;
    }
    let val = (u32::from(dbuf[0]) << 18)
        | (u32::from(dbuf[1]) << 12)
        | (u32::from(dbuf[2]) << 6)
        | u32::from(dbuf[3]);
    out.push((val >> 16) as u8);
    if dlen >= 3 {
        out.push((val >> 8) as u8);
    }
    if dlen >= 4 {
        out.push(val as u8);
    }
    Ok(si)
}

/// `base64.StdEncoding.Decode` for JSON `[]byte` fields: decoded bytes on
/// success, the `CorruptInputError` byte offset on failure.
pub fn b64_decode_go(src: &[u8]) -> Result<Vec<u8>, usize> {
    if src.is_empty() {
        return Ok(Vec::new());
    }
    let mut out = Vec::with_capacity(src.len().div_ceil(4) * 3);
    let mut si = 0;
    while si < src.len() {
        si = decode_quantum_go(src, si, &mut out)?;
    }
    Ok(out)
}

/// `CorruptInputError.Error()`: `illegal base64 data at input byte N`.
pub fn b64_corrupt(offset: usize) -> String {
    format!("illegal base64 data at input byte {offset}")
}

const B64_STD: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

pub fn b64_encode(raw: &[u8]) -> String {
    let mut out = String::with_capacity(raw.len().div_ceil(3) * 4);
    for chunk in raw.chunks(3) {
        let mut n: u32 = 0;
        for &b in chunk {
            n = (n << 8) | u32::from(b);
        }
        n <<= 8 * (3 - chunk.len());
        for i in 0..4 {
            if i <= chunk.len() {
                out.push(B64_STD[((n >> (18 - 6 * i)) & 63) as usize] as char);
            } else {
                out.push('=');
            }
        }
    }
    out
}

pub fn b64_encode_raw(raw: &[u8]) -> String {
    b64_encode(raw).trim_end_matches('=').to_string()
}

// ---------- wire primitives ----------

fn read_string(buf: &[u8]) -> Result<(&[u8], &[u8]), ()> {
    if buf.len() < 4 {
        return Err(());
    }
    let len = u32::from_be_bytes([buf[0], buf[1], buf[2], buf[3]]) as usize;
    if buf.len() < 4 + len {
        return Err(());
    }
    Ok((&buf[4..4 + len], &buf[4 + len..]))
}

fn read_u32(buf: &[u8]) -> Result<(u32, &[u8]), ()> {
    if buf.len() < 4 {
        return Err(());
    }
    Ok((
        u32::from_be_bytes([buf[0], buf[1], buf[2], buf[3]]),
        &buf[4..],
    ))
}

fn read_u64(buf: &[u8]) -> Result<(u64, &[u8]), ()> {
    if buf.len() < 8 {
        return Err(());
    }
    Ok((u64::from_be_bytes(buf[..8].try_into().unwrap()), &buf[8..]))
}

fn put_string(out: &mut Vec<u8>, data: &[u8]) {
    out.extend_from_slice(&(data.len() as u32).to_be_bytes());
    out.extend_from_slice(data);
}

/// Signed mpint: two's-complement parse, minimal magnitude.
#[derive(Debug, Clone, PartialEq, Eq)]
struct Mpint {
    negative: bool,
    mag: Vec<u8>,
}

fn strip_zeros(bytes: &[u8]) -> Vec<u8> {
    let i = bytes.iter().position(|&b| b != 0).unwrap_or(bytes.len());
    bytes[i..].to_vec()
}

fn parse_mpint(buf: &[u8]) -> Result<(Mpint, &[u8]), ()> {
    let (raw, rest) = read_string(buf)?;
    if raw.is_empty() {
        return Ok((
            Mpint {
                negative: false,
                mag: Vec::new(),
            },
            rest,
        ));
    }
    if raw[0] & 0x80 != 0 {
        // Negative: two's complement magnitude.
        let mut mag: Vec<u8> = raw.iter().map(|b| !b).collect();
        let mut carry = 1u16;
        for b in mag.iter_mut().rev() {
            let t = *b as u16 + carry;
            *b = t as u8;
            carry = t >> 8;
        }
        Ok((
            Mpint {
                negative: true,
                mag: strip_zeros(&mag),
            },
            rest,
        ))
    } else {
        Ok((
            Mpint {
                negative: false,
                mag: strip_zeros(raw),
            },
            rest,
        ))
    }
}

/// Bit length of the absolute value, like big.Int.BitLen.
fn mpint_bitlen(m: &Mpint) -> usize {
    if m.mag.is_empty() {
        return 0;
    }
    (m.mag.len() - 1) * 8 + (8 - m.mag[0].leading_zeros() as usize)
}

/// Signed comparison.
fn mpint_cmp(a: &Mpint, b: &Mpint) -> std::cmp::Ordering {
    match (a.negative, b.negative) {
        (true, false) => {
            if a.mag.is_empty() && b.mag.is_empty() {
                return std::cmp::Ordering::Equal;
            }
            // Negative (nonzero) is always less; -0 equals 0, but -0 cannot
            // arise: strip_zeros of all-0xff complement... note -0 parses
            // from [0x80]: complement+1 = [0x80] -> mag [0x80], negative.
            // Go: -0? big.Int can't hold -0; parseInt of [0x80] gives -128.
            // Our (neg,[0x80]) compares by magnitude below correctly.
            if a.mag.is_empty() {
                return std::cmp::Ordering::Equal;
            }
            std::cmp::Ordering::Less
        }
        (false, true) => {
            if b.mag.is_empty() {
                return std::cmp::Ordering::Equal;
            }
            std::cmp::Ordering::Greater
        }
        (false, false) | (true, true) => {
            let ord = mpint_bitlen(a)
                .cmp(&mpint_bitlen(b))
                .then_with(|| a.mag.cmp(&b.mag));
            if a.negative {
                ord.reverse()
            } else {
                ord
            }
        }
    }
}

fn mpint_to_i64(m: &Mpint) -> Option<i64> {
    if m.mag.len() > 8 {
        return None;
    }
    let mut v: i64 = 0;
    for &b in &m.mag {
        v = v.checked_mul(256)?.checked_add(b as i64)?;
    }
    if m.negative {
        v.checked_neg()
    } else {
        Some(v)
    }
}

fn marshal_mpint(out: &mut Vec<u8>, m: &Mpint) {
    let mut bytes = m.mag.clone();
    if bytes.is_empty() {
        // Zero is the zero-length string.
    } else if !m.negative {
        if bytes[0] & 0x80 != 0 {
            bytes.insert(0, 0x00);
        }
    } else {
        // Two's complement, minimal with the sign bit set.
        let mut carry = 1u16;
        for b in bytes.iter_mut().rev() {
            let t = (*b as u16 ^ 0xff) + carry;
            *b = t as u8;
            carry = t >> 8;
        }
        let mut twos = if carry != 0 { vec![1] } else { Vec::new() };
        twos.extend(bytes);
        // Strip redundant 0xff sign bytes, keeping the sign bit set.
        let mut i = 0;
        while twos.len() - i > 1 && twos[i] == 0xff && twos[i + 1] & 0x80 != 0 {
            i += 1;
        }
        bytes = twos[i..].to_vec();
        debug_assert!(!bytes.is_empty() && bytes[0] & 0x80 != 0);
    }
    put_string(out, &bytes);
}

// ---------- key types ----------

pub const ALGO_RSA: &str = "ssh-rsa";
pub const ALGO_DSS: &str = "ssh-dss";
pub const ALGO_ECDSA256: &str = "ecdsa-sha2-nistp256";
pub const ALGO_ECDSA384: &str = "ecdsa-sha2-nistp384";
pub const ALGO_ECDSA521: &str = "ecdsa-sha2-nistp521";
pub const ALGO_SKECDSA: &str = "sk-ecdsa-sha2-nistp256@openssh.com";
pub const ALGO_ED25519: &str = "ssh-ed25519";
pub const ALGO_SKED25519: &str = "sk-ssh-ed25519@openssh.com";

fn cert_inner(algo: &str) -> Option<&'static str> {
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

/// Parsed public key with its canonical wire bytes.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ParsedKey {
    pub key_type: String,
    pub blob: Vec<u8>,
}

#[derive(Debug, Clone)]
enum KeyMaterial {
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

#[derive(Debug, Clone)]
struct CertMaterial {
    cert_type: String,
    nonce: Vec<u8>,
    inner_algo: &'static str,
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

fn curve_for(algo: &str) -> Option<(&'static nist::Curve, &'static str)> {
    match algo {
        ALGO_ECDSA256 => Some((&nist::P256, "nistp256")),
        ALGO_ECDSA384 => Some((&nist::P384, "nistp384")),
        ALGO_ECDSA521 => Some((&nist::P521, "nistp521")),
        _ => None,
    }
}

fn parse_key_fields(algo: &str, buf: &[u8]) -> Result<(KeyMaterial, Vec<u8>), ()> {
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
            if !nist::valid_point(curve, point) {
                return Err(());
            }
            Ok((
                KeyMaterial::Ecdsa {
                    curve: expect,
                    point: point.to_vec(),
                },
                rest.to_vec(),
            ))
        }
        ALGO_SKECDSA => {
            let (curve_id, rest) = read_string(buf)?;
            let (point, rest) = read_string(rest)?;
            let (app, rest) = read_string(rest)?;
            if curve_id != b"nistp256" || !nist::valid_point(&nist::P256, point) {
                return Err(());
            }
            Ok((
                KeyMaterial::SkEcdsa {
                    point: point.to_vec(),
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

fn parse_tuples(mut buf: &[u8]) -> Result<(Vec<(Vec<u8>, Vec<u8>)>, ()), ()> {
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

fn parse_cert(
    algo: &str,
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
            cert_type: algo.to_string(),
            nonce: nonce.to_vec(),
            inner_algo,
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

fn marshal_material(out: &mut Vec<u8>, algo: &str, m: &KeyMaterial) {
    put_string(out, algo.as_bytes());
    marshal_fields(out, m);
}

fn marshal_fields(out: &mut Vec<u8>, m: &KeyMaterial) {
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
        KeyMaterial::Cert(c) => {
            put_string(out, &c.nonce);
            marshal_fields(out, &c.inner);
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
    }
}

/// Parse one wire-format public key blob into its canonical re-marshaled
/// form. Trailing bytes are rejected.
pub fn parse_public_key(blob: &[u8]) -> Result<ParsedKey, String> {
    let (algo, rest) = read_string(blob).map_err(|_| "invalid public key".to_string())?;
    let algo = std::str::from_utf8(algo)
        .map_err(|_| "invalid public key".to_string())?
        .to_string();
    let (material, rest) =
        parse_key_fields(&algo, rest).map_err(|_| "invalid public key".to_string())?;
    if !rest.is_empty() {
        return Err("invalid public key".to_string());
    }
    let mut canonical = Vec::new();
    marshal_material(&mut canonical, &algo, &material);
    Ok(ParsedKey {
        key_type: algo,
        blob: canonical,
    })
}

/// Parse one authorized-keys line (options and key type already removed):
/// base64 blob plus trailing comment.
fn parse_key_text(line: &[u8]) -> Result<(ParsedKey, ()), String> {
    let line = trim_ws(line);
    let i = line
        .iter()
        .position(|&b| b == b' ' || b == b'\t')
        .unwrap_or(line.len());
    let encoded = std::str::from_utf8(&line[..i]).map_err(|_| "invalid public key".to_string())?;
    let blob = b64_decode(encoded).map_err(|_| "invalid public key".to_string())?;
    Ok((parse_public_key(&blob)?, ()))
}

/// Quote-aware options scan, mirroring x/crypto: splits the leading options
/// region on unquoted commas, stops at the first unquoted space/tab.
/// Returns the end offset and whether any non-empty candidate option was
/// collected (an empty candidate list means no options, even on this path).
fn scan_options(line: &[u8]) -> Option<(usize, bool)> {
    let mut in_quote = false;
    let mut option_start = 0usize;
    let mut had_candidate = false;
    let mut i = 0usize;
    while i < line.len() {
        let b = line[i];
        let is_end = !in_quote && (b == b' ' || b == b'\t');
        if (!in_quote && b == b',') || is_end {
            if i > option_start {
                had_candidate = true;
            }
            option_start = i + 1;
        }
        if is_end {
            return Some((i, had_candidate));
        }
        if b == b'"' && (i == 0 || line[i - 1] != b'\\') {
            in_quote = !in_quote;
        }
        i += 1;
    }
    None
}

/// Full `ParseAuthorizedKey` line-scan semantics: blank/`#` lines skipped,
/// declared type must equal the embedded type, options detected, trailing
/// data after the accepted line must be blank. The returned blob is the
/// canonical re-marshal; callers compare it against the input for
/// canonicality and reject `has_options`.
pub fn parse_authorized_key(input: &[u8]) -> Result<(ParsedKey, bool), String> {
    let mut rest_all = input;
    let mut last_err: Option<String> = None;
    loop {
        // Split one line at \n; rest is everything after it.
        let (mut line, rest) = match rest_all.iter().position(|&b| b == b'\n') {
            Some(i) => (&rest_all[..i], &rest_all[i + 1..]),
            None => (rest_all, &[][..]),
        };
        if rest_all.is_empty() {
            break;
        }
        // Cut at the first carriage return, then trim.
        if let Some(i) = line.iter().position(|&b| b == b'\r') {
            line = &line[..i];
        }
        line = trim_ws(line);
        rest_all = rest;
        if line.is_empty() || line[0] == b'#' {
            continue;
        }
        let Some(space) = line.iter().position(|&b| b == b' ' || b == b'\t') else {
            continue;
        };
        // Direct parse: first field is the declared key type.
        let declared = &line[..space];
        match parse_key_text(&line[space..]) {
            Ok((key, ())) if key.key_type.as_bytes() == declared => {
                if !trim_ws(rest).is_empty() {
                    return Err("invalid public key".to_string());
                }
                return Ok((key, false));
            }
            Ok(_) => {
                last_err = Some("type mismatch".to_string());
            }
            Err(e) => {
                last_err = Some(e);
            }
        }
        // Options path: strip the leading options region, then type + blob.
        let (after_options, had_candidate) = match scan_options(line) {
            Some((i, had)) => {
                let mut j = i;
                while j < line.len() && (line[j] == b' ' || line[j] == b'\t') {
                    j += 1;
                }
                if j >= line.len() {
                    continue;
                }
                (&line[j..], had)
            }
            None => continue,
        };
        let Some(space2) = after_options.iter().position(|&b| b == b' ' || b == b'\t') else {
            continue;
        };
        let declared2 = &after_options[..space2];
        match parse_key_text(&after_options[space2..]) {
            Ok((key, ())) if key.key_type.as_bytes() == declared2 => {
                if !trim_ws(rest).is_empty() {
                    return Err("invalid public key".to_string());
                }
                return Ok((key, had_candidate));
            }
            Ok(_) => {
                last_err = Some("type mismatch".to_string());
            }
            Err(e) => {
                last_err = Some(e);
            }
        }
    }
    Err(last_err.unwrap_or_else(|| "no key found".to_string()))
}

/// `ssh.MarshalAuthorizedKey`: `type base64` plus the trailing newline.
pub fn marshal_authorized_key(key_type: &str, blob: &[u8]) -> String {
    format!("{key_type} {}\n", b64_encode(blob))
}

/// `ssh.FingerprintSHA256`: `SHA256:` plus unpadded base64 of the digest.
pub fn fingerprint_sha256(blob: &[u8]) -> String {
    let digest = crate::sha256::digest(blob);
    format!("SHA256:{}", b64_encode_raw(&digest))
}

#[cfg(test)]
mod tests {
    use super::*;

    // Public-key fixtures only (comments stripped). RSA/ECDSA/Ed25519 from
    // ssh-keygen, DSA and the OpenSSH certificate from Go's x/crypto.
    const RSA: &str = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7sz+R+V5I0foZSFl3AOjfvN9eOpNqb/n1QFuiO6XzEo4kKnOs5/nIp/XOFC0/SzBl4S0xhmBw+bbNgpPQ+rreuLrsA01wqEoNmtE7VsEXNysYV58cVaGaGI4E/hAjKuyRzdQkI99uGiHXnEN5iwojR493ZJ0wvN8mWhsjU2U+ctjQneR4sphYRUgDet3mPqHMYucH7eZgfPVxHNUTxTTUXinZPxhRODHz0QRCezBoMDY1nwpFy/gC8hXusrKomlWkGg3mEZTMTbmMsWcpz/xVT48S3M+z9LYptTXTS6bPhiUh4xHsQWOo6Iy1LIiSMCPcBkhgCebx23OGjpFbjuNJ";
    const ECDSA256: &str = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHj7ciCLbVeoJXxkjiotLyfyRA8p9CyO0TMNFOqLt/TPEElUQ2eJTL9+Wi9hhFVDjaWWCzm/dCcAyU94xFHSe2c=";
    const ECDSA384: &str = "ecdsa-sha2-nistp384 AAAAE2VjZHNhLXNoYTItbmlzdHAzODQAAAAIbmlzdHAzODQAAABhBGhjWK/kZlOIFnt++QFazBBAxQ+/qsF0ZiCqRuMVeVPjPsdw1ZI6idf5gBpPNBRiscAM201TmUdse8NTzPh4cMSN0tHx+8CIKlACp+4CyEnITV81ZiRozztvIXlABAe9Aw==";
    const ECDSA521: &str = "ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBAC7PT6SUkkAyi1W3YCqPFMDBYOHD3W7s5e78EObzfBPnJU1TmNvk1oIYttpaPEv8+v7VmgE0KWkLOYAVro5gH3tMQA/oQYdzKuD1i6RvifNpRIt2lfRV2NKZ2IvuVV/O4TdRoW4IoJ/Z6I0xA8TcbACkei5u2g2xu3fZWVHq9hOCaFmJw==";
    const ED: &str =
        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre";
    const DSA: &str = "ssh-dss AAAAB3NzaC1kc3MAAACBAIz7Di7wJQy0oy8m6z/XMjyzC4nIWYnEbwJfa+7BTVCYF1Ft5zzz6Zz9rjdmRcgSbLy5ImfS5I4dDFO7RaGu8vWTUX4Bt+WuZPPXuP7gfoWKJS+QhYabM/AbFpqF/Y+RqKf78fmlJf93vx8N4KJHtXpKOscR8+f5o2hjOZ3qdA5NAAAAFQDQwFThs2jiwKMm5WGMWT/yhMLDxwAAAIBvtk8aJSrq2eFwwpTRpvj/J4goZiirxksn87GpqCT+LVrcGTWNClQaBRA+K7Vr2L7QvB7YxeBWAo71Rq2xTPwBa9UXlzVOCaZeg5DXwtC0pAVcHSNlsVa7id1W2qV4fjCaF6clNUUjTsOxqwKRhYvHWAXPS0H9Xopj94672Oq4cAAAAIBqBT9SM/sIyejm2HcefUtARGMKOg6qkvkwNV4XMSADgdu/0vNeUxPXChtm/wmhZ6HygBOq+aV/o/athTEpDdteskAxdgPUgpDwKiScG8XhraCOCGnzNFAgeMAS4mKZilKJ0ewhMu3yCF1taiQqqG2iIosyaszZD8Zk/BhXN7IOeQ==";
    const CERT: &str = "ssh-ed25519-cert-v01@openssh.com AAAAIHNzaC1lZDI1NTE5LWNlcnQtdjAxQG9wZW5zc2guY29tAAAAIPVDs6sltHdos8K2q8bZnSJvBVHhzMSZwnC4EUO5JMbHAAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcreAAAAAAAAAAAAAAABAAAABnRlc3RpZAAAAAkAAAAFYWxpY2UAAAAAAAAAAP//////////AAAAAAAAAIIAAAAVcGVybWl0LVgxMS1mb3J3YXJkaW5nAAAAAAAAABdwZXJtaXQtYWdlbnQtZm9yd2FyZGluZwAAAAAAAAAWcGVybWl0LXBvcnQtZm9yd2FyZGluZwAAAAAAAAAKcGVybWl0LXB0eQAAAAAAAAAOcGVybWl0LXVzZXItcmMAAAAAAAAAAAAAARcAAAAHc3NoLXJzYQAAAAMBAAEAAAEBALuzP5H5XkjR+hlIWXcA6N+83146k2pv+fVAW6I7pfMSjiQqc6zn+cin9c4ULT9LMGXhLTGGYHD5ts2Ck9D6ut64uuwDTXCoSg2a0TtWwRc3KxhXnxxVoZoYjgT+ECMq7JHN1CQj324aIdecQ3mLCiNHj3dknTC83yZaGyNTZT5y2NCd5HiymFhFSAN63eY+ocxi5wft5mB89XEc1RPFNNReKdk/GFE4MfPRBEJ7MGgwNjWfCkXL+ALyFe6ysqiaVaQaDeYRlMxNuYyxZynP/FVPjxLcz7P0tim1NdNLps+GJSHjEexBY6jojLUsiJIwI9wGSGAJ5vHbc4aOkVuO40kAAAEUAAAADHJzYS1zaGEyLTUxMgAAAQBrguk2IhdEGpwyqtiWaL87YzDzY7lFiKLeXDNF3WzcYlgx8zLF3Ujy33WH8MSvzIgNJTY/QJseouCU9iYrTrTlUjLht5DWeKf622TaOwUw7B3R4/ugjdDk49wrxjoO86Wvb8WosM8NYuNTddom2D/CVuOXB39LxUIP5kzaK3Z6p59xlGyamdHXaxs3M5yzU67imcs0G3pnf3G2N+KhcPOIzThd8nyyp2hPUgJqwfNxjxboHSr6JbOXxa0Iud5gRilfsoREaA4Fh4Qf/2OvqvhWLqhjPISqozAR3CZJ1v9RQaqr5ROaGLrQFudRFacxi3HgVa9GjCP2ZEwC4Q4kT0d3";

    fn canonical(line: &str) -> (String, bool) {
        let (key, opts) = parse_authorized_key(line.as_bytes()).unwrap();
        (marshal_authorized_key(&key.key_type, &key.blob), opts)
    }

    #[test]
    fn all_types_round_trip_canonically() {
        for line in [RSA, ECDSA256, ECDSA384, ECDSA521, ED, DSA, CERT] {
            let (back, opts) = canonical(line);
            assert_eq!(back, format!("{line}\n"));
            assert!(!opts);
        }
    }

    #[test]
    fn sk_types_parse() {
        // Synthetic SK blobs with the exact wire shape (verified against
        // Go's parser in the differential check, not just here).
        let mut ed = vec![0u8; 32];
        ed.copy_from_slice(b"0123456789abcdef0123456789abcdef");
        let mut blob = Vec::new();
        put_string(&mut blob, ALGO_SKED25519.as_bytes());
        put_string(&mut blob, &ed);
        put_string(&mut blob, b"ssh:test");
        let line = format!("{ALGO_SKED25519} {}", b64_encode(&blob));
        assert_eq!(canonical(&line).0, format!("{line}\n"));
    }

    #[test]
    fn comments_and_whitespace_tolerated_but_not_canonical() {
        // Parse accepts (comment captured, rest blank); the canonical form
        // drops them, so canonicalKeys' byte comparison rejects them while
        // the account path normalizes them.
        let (key, opts) = parse_authorized_key(format!("{ED} alice@host\n").as_bytes()).unwrap();
        assert_eq!(key.key_type, ALGO_ED25519);
        assert!(!opts);
        let (key, _) = parse_authorized_key(format!("   {ED}  \n").as_bytes()).unwrap();
        assert_eq!(
            marshal_authorized_key(&key.key_type, &key.blob),
            format!("{ED}\n")
        );
        // Declared type must equal the embedded type.
        assert!(parse_authorized_key(format!("ssh-rsa {}", &ED[12..]).as_bytes()).is_err());
    }

    #[test]
    fn options_detected_and_multiline_rules_match() {
        let (_, opts) = parse_authorized_key(format!("no-pty {ED}\n").as_bytes()).unwrap();
        assert!(opts);
        let (_, opts) =
            parse_authorized_key(format!("command=\"echo hi\",no-pty {ED}\n").as_bytes()).unwrap();
        assert!(opts);
        // Garbage lines are skipped; trailing data after the key rejects.
        assert!(parse_authorized_key(format!("garbage\n{ED}\n").as_bytes()).is_ok());
        assert!(parse_authorized_key(format!("{ED}\n{ED}\n").as_bytes()).is_err());
        assert!(parse_authorized_key(format!("{ED}\n  \n").as_bytes()).is_ok());
        assert!(parse_authorized_key(format!("{ED}\r\n").as_bytes()).is_ok());
        assert!(parse_authorized_key(b"").is_err());
        assert!(parse_authorized_key(b"ssh-ed25519").is_err());
        assert!(parse_authorized_key(b"ssh-ed25519 YWJj\n").is_err());
    }

    #[test]
    fn invalid_keys_rejected() {
        // Truncated blob.
        assert!(parse_authorized_key(b"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA\n").is_err());
        // Trailing junk in blob.
        let mut blob = b64_decode(ED.split(' ').nth(1).unwrap()).unwrap();
        blob.push(0);
        assert!(
            parse_authorized_key(format!("ssh-ed25519 {}\n", b64_encode(&blob)).as_bytes())
                .is_err()
        );
        // Unknown algorithm.
        assert!(parse_authorized_key(b"ssh-foo AAAA\n").is_err());
        // Off-curve point (flip a byte in a valid P-256 key).
        let mut raw = b64_decode(ECDSA256.split(' ').nth(1).unwrap()).unwrap();
        let n = raw.len();
        raw[n - 10] ^= 0x01;
        assert!(parse_authorized_key(
            format!("ecdsa-sha2-nistp256 {}\n", b64_encode(&raw)).as_bytes()
        )
        .is_err());
        // Cert signed by a cert.
        let mut cert = b64_decode(CERT.split(' ').nth(1).unwrap()).unwrap();
        let _ = &mut cert;
    }

    #[test]
    fn base64_vectors() {
        assert_eq!(b64_decode("").unwrap_err(), "invalid base64".to_string());
        assert_eq!(b64_decode("QUI=").unwrap(), b"AB");
        assert_eq!(b64_decode("QUI"), Err("invalid base64".to_string()));
        assert_eq!(b64_decode("AB=C"), Err("invalid base64".to_string()));
        assert_eq!(b64_encode(b"AB"), "QUI=");
        assert_eq!(b64_encode_raw(b"AB"), "QUI");
    }

    #[test]
    fn fingerprint_shape() {
        let (key, _) = parse_authorized_key(ED.as_bytes()).unwrap();
        let fp = fingerprint_sha256(&key.blob);
        assert!(fp.starts_with("SHA256:") && !fp.ends_with('='));
        assert_eq!(fp.len(), 7 + 43);
    }
}
