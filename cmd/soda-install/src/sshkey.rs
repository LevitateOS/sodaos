//! OpenSSH public-key parsing mirroring `golang.org/x/crypto/ssh`
//! (`ParseAuthorizedKey`, per-type wire validation, canonical
//! `MarshalAuthorizedKey`, `FingerprintSHA256`), backing `PublicKey`.

use elliptic_curve::sec1::ToEncodedPoint;
use sha2::{Digest, Sha256};

use crate::fmtx::go_trim_space;

// ---------------------------------------------------------------------------
// Base64: exact Go `StdEncoding.Decode` (lenient trailing quantum, strict
// padding placement, ignored `\r\n`) and canonical padded encode.
// ---------------------------------------------------------------------------

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

pub fn b64_decode_go(input: &[u8]) -> Result<Vec<u8>, ()> {
    // Strip newlines exactly as Go's decoder skips them mid-quantum.
    let mut src: Vec<u8> = Vec::with_capacity(input.len());
    for &b in input {
        if b != b'\r' && b != b'\n' {
            src.push(b);
        }
    }
    let mut out = Vec::with_capacity(src.len() / 4 * 3 + 3);
    let mut i = 0;
    while i < src.len() {
        // One quantum: up to 4 significant chars; padding ends the input.
        let mut quantum = [0u8; 4];
        let mut have = 0;
        while have < 4 && i < src.len() {
            let c = src[i];
            if c == b'=' {
                break;
            }
            quantum[have] = b64_value(c).ok_or(())?;
            have += 1;
            i += 1;
        }
        if i < src.len() && src[i] == b'=' {
            // Padding at quantum positions 0/1 is corrupt; at 2 it needs
            // the second `=`; nothing may follow padding.
            if have < 2 {
                return Err(());
            }
            i += 1;
            if have == 2 {
                if i >= src.len() || src[i] != b'=' {
                    return Err(());
                }
                i += 1;
            }
            if i < src.len() {
                return Err(());
            }
        } else if have == 1 {
            // A 1-char tail is corrupt; 2/3-char tails decode short.
            return Err(());
        } else if have == 0 {
            break;
        }
        let n = (u32::from(quantum[0]) << 18)
            | (u32::from(quantum[1]) << 12)
            | (u32::from(quantum[2]) << 6)
            | u32::from(quantum[3]);
        // Non-strict trailing bits are ignored, as in Go.
        out.push((n >> 16) as u8);
        if have >= 3 {
            out.push((n >> 8) as u8);
        }
        if have == 4 {
            out.push(n as u8);
        }
    }
    Ok(out)
}

pub fn b64_encode(raw: &[u8]) -> String {
    const ALPHA: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity(raw.len().div_ceil(3) * 4);
    for chunk in raw.chunks(3) {
        let mut n: u32 = 0;
        for &b in chunk {
            n = (n << 8) | u32::from(b);
        }
        n <<= 8 * (3 - chunk.len());
        for i in 0..4 {
            if i <= chunk.len() {
                out.push(ALPHA[((n >> (18 - 6 * i)) & 63) as usize] as char);
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

// ---------------------------------------------------------------------------
// SSH wire primitives.
// ---------------------------------------------------------------------------

fn parse_string(input: &[u8]) -> Result<(&[u8], &[u8]), ()> {
    if input.len() < 4 {
        return Err(());
    }
    let len = u32::from_be_bytes([input[0], input[1], input[2], input[3]]) as usize;
    if input.len() < 4 + len {
        return Err(());
    }
    Ok((&input[4..4 + len], &input[4 + len..]))
}

fn parse_u32(input: &[u8]) -> Result<(u32, &[u8]), ()> {
    if input.len() < 4 {
        return Err(());
    }
    Ok((
        u32::from_be_bytes([input[0], input[1], input[2], input[3]]),
        &input[4..],
    ))
}

fn parse_u64(input: &[u8]) -> Result<(u64, &[u8]), ()> {
    if input.len() < 8 {
        return Err(());
    }
    Ok((
        u64::from_be_bytes([
            input[0], input[1], input[2], input[3], input[4], input[5], input[6], input[7],
        ]),
        &input[8..],
    ))
}

/// SSH mpint as (negative, minimal magnitude bytes).
fn parse_mpint(input: &[u8]) -> Result<(bool, Vec<u8>, &[u8]), ()> {
    let (contents, rest) = parse_string(input)?;
    if contents.is_empty() {
        return Ok((false, Vec::new(), rest));
    }
    if contents[0] & 0x80 == 0 {
        let start = contents
            .iter()
            .position(|b| *b != 0)
            .unwrap_or(contents.len());
        return Ok((false, contents[start..].to_vec(), rest));
    }
    // Negative: two's complement magnitude.
    let mut mag: Vec<u8> = contents.iter().map(|b| !b).collect();
    let mut carry = 1u16;
    for b in mag.iter_mut().rev() {
        let sum = u16::from(*b) + carry;
        *b = sum as u8;
        carry = sum >> 8;
    }
    let start = mag.iter().position(|b| *b != 0).unwrap_or(mag.len());
    Ok((true, mag[start..].to_vec(), rest))
}

fn bit_len(mag: &[u8]) -> usize {
    if mag.is_empty() {
        return 0;
    }
    mag.len() * 8 - mag[0].leading_zeros() as usize
}

/// Minimal signed mpint encoding, mirroring x/crypto `marshalInt`.
fn marshal_mpint(negative: bool, mag: &[u8]) -> Vec<u8> {
    let body = if !negative {
        if mag.is_empty() {
            Vec::new()
        } else if mag[0] & 0x80 != 0 {
            let mut out = vec![0u8];
            out.extend_from_slice(mag);
            out
        } else {
            mag.to_vec()
        }
    } else {
        // Two's complement of the magnitude: invert ((mag - 1)).
        let mut minus: Vec<u8> = mag.to_vec();
        let mut borrow = 1i16;
        for b in minus.iter_mut().rev() {
            let diff = i16::from(*b) - borrow;
            *b = diff as u8;
            borrow = if diff < 0 { 1 } else { 0 };
        }
        let start = minus.iter().position(|b| *b != 0).unwrap_or(minus.len());
        let mut out: Vec<u8> = minus[start..].iter().map(|b| !b).collect();
        if out.is_empty() || out[0] & 0x80 == 0 {
            out.insert(0, 0xff);
        }
        out
    };
    let mut out = Vec::with_capacity(4 + body.len());
    out.extend_from_slice(&(body.len() as u32).to_be_bytes());
    out.extend_from_slice(&body);
    out
}

fn marshal_string(bytes: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(4 + bytes.len());
    out.extend_from_slice(&(bytes.len() as u32).to_be_bytes());
    out.extend_from_slice(bytes);
    out
}

fn cmp_mag(a: &[u8], b: &[u8]) -> std::cmp::Ordering {
    let sa = a.iter().position(|x| *x != 0).unwrap_or(a.len());
    let sb = b.iter().position(|x| *x != 0).unwrap_or(b.len());
    let (a, b) = (&a[sa..], &b[sb..]);
    a.len().cmp(&b.len()).then_with(|| a.cmp(b))
}

// ---------------------------------------------------------------------------
// Parsed public keys.
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Key {
    Rsa {
        e: u32,
        n_negative: bool,
        n_mag: Vec<u8>,
    },
    Dss,
    Ecdsa {
        curve: &'static str,
        point: Vec<u8>,
    },
    Ed25519 {
        key: [u8; 32],
    },
    SkEcdsa {
        point: Vec<u8>,
        application: Vec<u8>,
    },
    SkEd25519 {
        key: [u8; 32],
        application: Vec<u8>,
    },
    Cert {
        algo: String,
    },
}

impl Key {
    pub fn key_type(&self) -> String {
        match self {
            Key::Rsa { .. } => "ssh-rsa".to_string(),
            Key::Dss => "ssh-dss".to_string(),
            Key::Ecdsa { curve, .. } => format!("ecdsa-sha2-{curve}"),
            Key::Ed25519 { .. } => "ssh-ed25519".to_string(),
            Key::SkEcdsa { .. } => "sk-ecdsa-sha2-nistp256@openssh.com".to_string(),
            Key::SkEd25519 { .. } => "sk-ssh-ed25519@openssh.com".to_string(),
            Key::Cert { algo } => algo.clone(),
        }
    }
    /// Canonical wire encoding, mirroring each type's `Marshal`.
    pub fn marshal(&self) -> Vec<u8> {
        let mut out = marshal_string(self.key_type().as_bytes());
        match self {
            Key::Rsa {
                e,
                n_negative,
                n_mag,
            } => {
                let mut mag = e.to_be_bytes().to_vec();
                let start = mag.iter().position(|b| *b != 0).unwrap_or(mag.len());
                mag = mag[start..].to_vec();
                out.extend(marshal_mpint(false, &mag));
                out.extend(marshal_mpint(*n_negative, n_mag));
            }
            Key::Dss => unreachable!("DSA keys never marshal in this port"),
            Key::Ecdsa { curve, point } => {
                out.extend(marshal_string(curve.as_bytes()));
                out.extend(marshal_string(point));
            }
            Key::Ed25519 { key } => out.extend(marshal_string(key)),
            Key::SkEcdsa { point, application } => {
                out.extend(marshal_string(b"nistp256"));
                out.extend(marshal_string(point));
                out.extend(marshal_string(application));
            }
            Key::SkEd25519 { key, application } => {
                out.extend(marshal_string(key));
                out.extend(marshal_string(application));
            }
            Key::Cert { .. } => unreachable!("certificates never marshal in this port"),
        }
        out
    }
}

fn ecdsa_point(curve: &str, bytes: &[u8]) -> Result<Vec<u8>, ()> {
    // Go `elliptic.Unmarshal` accepts uncompressed points on the curve only;
    // re-encode canonically like `elliptic.Marshal`. Compressed encodings
    // are refused even though the curve backend would parse them.
    if bytes.first() != Some(&0x04) {
        return Err(());
    }
    let canonical = match curve {
        "nistp256" => {
            let key = p256::PublicKey::from_sec1_bytes(bytes).map_err(|_| ())?;
            key.to_encoded_point(false).as_bytes().to_vec()
        }
        "nistp384" => {
            let key = p384::PublicKey::from_sec1_bytes(bytes).map_err(|_| ())?;
            key.to_encoded_point(false).as_bytes().to_vec()
        }
        "nistp521" => {
            let key = p521::PublicKey::from_sec1_bytes(bytes).map_err(|_| ())?;
            key.to_encoded_point(false).as_bytes().to_vec()
        }
        _ => return Err(()),
    };
    Ok(canonical)
}

fn parse_rsa(input: &[u8]) -> Result<(Key, &[u8]), ()> {
    let (e_neg, e_mag, rest) = parse_mpint(input)?;
    let (n_neg, n_mag, rest) = parse_mpint(rest)?;
    if bit_len(&n_mag) > 16384 || bit_len(&e_mag) > 24 {
        return Err(());
    }
    if e_neg {
        return Err(());
    }
    let mut e: u32 = 0;
    for &b in &e_mag {
        e = e.checked_shl(8).ok_or(())? | u32::from(b);
    }
    if e < 3 || e & 1 == 0 {
        return Err(());
    }
    Ok((
        Key::Rsa {
            e,
            n_negative: n_neg,
            n_mag,
        },
        rest,
    ))
}

fn parse_dsa(input: &[u8]) -> Result<(Key, &[u8]), ()> {
    let (p_neg, p, rest) = parse_mpint(input)?;
    let (q_neg, q, rest) = parse_mpint(rest)?;
    let (g_neg, g, rest) = parse_mpint(rest)?;
    let (y_neg, y, rest) = parse_mpint(rest)?;
    // Go checks only bit lengths of P and Q (sign ignored); Y and G must
    // be positive and below P.
    let _ = (p_neg, q_neg);
    if bit_len(&p) != 1024 || bit_len(&q) != 160 {
        return Err(());
    }
    if g_neg || g.is_empty() || cmp_mag(&g, &p) != std::cmp::Ordering::Less {
        return Err(());
    }
    if y_neg || y.is_empty() || cmp_mag(&y, &p) != std::cmp::Ordering::Less {
        return Err(());
    }
    Ok((Key::Dss, rest))
}

fn parse_ecdsa<'a>(input: &'a [u8], expected: &str) -> Result<(Key, &'a [u8]), ()> {
    let (curve, rest) = parse_string(input)?;
    let (point, rest) = parse_string(rest)?;
    let curve = std::str::from_utf8(curve).map_err(|_| ())?;
    if format!("ecdsa-sha2-{curve}") != expected {
        return Err(());
    }
    let point = ecdsa_point(curve, point)?;
    let curve: &'static str = match curve {
        "nistp256" => "nistp256",
        "nistp384" => "nistp384",
        "nistp521" => "nistp521",
        _ => return Err(()),
    };
    Ok((Key::Ecdsa { curve, point }, rest))
}

fn parse_ed25519(input: &[u8]) -> Result<(Key, &[u8]), ()> {
    let (key, rest) = parse_string(input)?;
    if key.len() != 32 {
        return Err(());
    }
    let mut fixed = [0u8; 32];
    fixed.copy_from_slice(key);
    Ok((Key::Ed25519 { key: fixed }, rest))
}

fn parse_sk_ecdsa(input: &[u8]) -> Result<(Key, &[u8]), ()> {
    let (curve, rest) = parse_string(input)?;
    let (point, rest) = parse_string(rest)?;
    let (application, rest) = parse_string(rest)?;
    if curve != b"nistp256" {
        return Err(());
    }
    let point = ecdsa_point("nistp256", point)?;
    Ok((
        Key::SkEcdsa {
            point,
            application: application.to_vec(),
        },
        rest,
    ))
}

fn parse_sk_ed25519(input: &[u8]) -> Result<(Key, &[u8]), ()> {
    let (key, rest) = parse_string(input)?;
    let (application, rest) = parse_string(rest)?;
    if key.len() != 32 {
        return Err(());
    }
    let mut fixed = [0u8; 32];
    fixed.copy_from_slice(key);
    Ok((
        Key::SkEd25519 {
            key: fixed,
            application: application.to_vec(),
        },
        rest,
    ))
}

fn parse_tuples(mut input: &[u8]) -> Result<(), ()> {
    let mut last: Option<Vec<u8>> = None;
    while !input.is_empty() {
        let (key, rest) = parse_string(input)?;
        if let Some(prev) = &last {
            if key <= prev.as_slice() {
                return Err(());
            }
        }
        last = Some(key.to_vec());
        let (value, rest) = parse_string(rest)?;
        if !value.is_empty() {
            let (_, extra) = parse_string(value)?;
            if !extra.is_empty() {
                return Err(());
            }
        }
        input = rest;
    }
    Ok(())
}

fn cert_inner_algo(algo: &str) -> Option<&'static str> {
    match algo {
        "ssh-rsa-cert-v01@openssh.com" => Some("ssh-rsa"),
        "ssh-dss-cert-v01@openssh.com" => Some("ssh-dss"),
        "ecdsa-sha2-nistp256-cert-v01@openssh.com" => Some("ecdsa-sha2-nistp256"),
        "ecdsa-sha2-nistp384-cert-v01@openssh.com" => Some("ecdsa-sha2-nistp384"),
        "ecdsa-sha2-nistp521-cert-v01@openssh.com" => Some("ecdsa-sha2-nistp521"),
        "sk-ecdsa-sha2-nistp256-cert-v01@openssh.com" => Some("sk-ecdsa-sha2-nistp256@openssh.com"),
        "ssh-ed25519-cert-v01@openssh.com" => Some("ssh-ed25519"),
        "sk-ssh-ed25519-cert-v01@openssh.com" => Some("sk-ssh-ed25519@openssh.com"),
        _ => None,
    }
}

fn is_cert_algo(algo: &str) -> bool {
    cert_inner_algo(algo).is_some()
        || algo == "rsa-sha2-256-cert-v01@openssh.com"
        || algo == "rsa-sha2-512-cert-v01@openssh.com"
}

fn parse_cert(input: &[u8], algo: &str) -> Result<Key, ()> {
    let inner = cert_inner_algo(algo).ok_or(())?;
    let (_, mut rest) = parse_string(input)?; // nonce
    let (_, inner_rest) = parse_pub_key(rest, inner)?; // inner key, no trailing check
    rest = inner_rest;
    let (_, r) = parse_u64(rest)?; // serial
    let (_, r) = parse_u32(r)?; // cert type
    let (_, r) = parse_string(r)?; // key id
    let (principals, r) = parse_string(r)?;
    let (_, r) = parse_u64(r)?; // valid after
    let (_, r) = parse_u64(r)?; // valid before
    let (critical, r) = parse_string(r)?;
    let (extensions, r) = parse_string(r)?;
    let (_, r) = parse_string(r)?; // reserved
    let (sigkey, r) = parse_string(r)?;
    let (signature, r) = parse_string(r)?;
    if !r.is_empty() {
        return Err(());
    }
    let mut principals = principals;
    while !principals.is_empty() {
        let (_, p) = parse_string(principals)?;
        principals = p;
    }
    parse_tuples(critical)?;
    parse_tuples(extensions)?;
    let (sig_algo, _) = parse_string(sigkey)?;
    let sig_algo = std::str::from_utf8(sig_algo).map_err(|_| ())?;
    if is_cert_algo(sig_algo) {
        return Err(());
    }
    parse_public_key(sigkey)?;
    let (format, sig_rest) = parse_string(signature)?; // format
    let (_, sig_rest) = parse_string(sig_rest)?; // blob
                                                 // Go stashes trailing bytes for SK signature formats and rejects them
                                                 // for every other format.
                                                 // The four SK signature formats share one shape: "sk-" + key +
                                                 // optional "-cert-v01" + "@openssh.com", with two possible keys.
    let sk_trailing = match format.strip_suffix(b"@openssh.com") {
        Some(head) => {
            let head = head.strip_suffix(b"-cert-v01").unwrap_or(head);
            matches!(head.strip_prefix(b"sk-"), Some(m) if m == b"ecdsa-sha2-nistp256" || m == b"ssh-ed25519")
        }
        None => false,
    };
    if !sig_rest.is_empty() && !sk_trailing {
        return Err(());
    }
    Ok(Key::Cert {
        algo: algo.to_string(),
    })
}

/// x/crypto `parsePubKey`: structural validation, trailing bytes returned.
fn parse_pub_key<'a>(input: &'a [u8], algo: &str) -> Result<(Key, &'a [u8]), ()> {
    match algo {
        "ssh-rsa" => parse_rsa(input),
        "ssh-dss" => parse_dsa(input),
        "ecdsa-sha2-nistp256" | "ecdsa-sha2-nistp384" | "ecdsa-sha2-nistp521" => {
            parse_ecdsa(input, algo)
        }
        "sk-ecdsa-sha2-nistp256@openssh.com" => parse_sk_ecdsa(input),
        "ssh-ed25519" => parse_ed25519(input),
        "sk-ssh-ed25519@openssh.com" => parse_sk_ed25519(input),
        _ if cert_inner_algo(algo).is_some() => {
            let key = parse_cert(input, algo)?;
            Ok((key, &[]))
        }
        _ => Err(()),
    }
}

/// x/crypto `ParsePublicKey`: trailing bytes are refused.
fn parse_public_key(wire: &[u8]) -> Result<Key, ()> {
    let (algo, rest) = parse_string(wire)?;
    let algo = std::str::from_utf8(algo).map_err(|_| ())?;
    let (key, rest) = parse_pub_key(rest, algo)?;
    if !rest.is_empty() {
        return Err(());
    }
    Ok(key)
}

// ---------------------------------------------------------------------------
// Authorized-line parsing: exact port of x/crypto `ParseAuthorizedKey` for
// single-line input (callers pre-reject `\r\n\x00`).
// ---------------------------------------------------------------------------

fn parse_key_field(line: &[u8]) -> Result<(Key, String), ()> {
    let line = go_trim_space(std::str::from_utf8(line).map_err(|_| ())?);
    let split = line.find([' ', '\t']).unwrap_or(line.len());
    let (encoded, comment) = (&line.as_bytes()[..split], go_trim_space(&line[split..]));
    let wire = b64_decode_go(encoded)?;
    let key = parse_public_key(&wire)?;
    Ok((key, comment.to_string()))
}

fn scan_options(line: &str) -> Option<(Vec<String>, &str)> {
    // Exact port of the x/crypto options scan: commas split outside quotes,
    // the first unquoted space/tab ends the options field.
    let bytes = line.as_bytes();
    let mut in_quote = false;
    let mut options = Vec::new();
    let mut start = 0;
    let mut i = 0;
    let mut ended = false;
    while i < bytes.len() {
        let b = bytes[i];
        let is_end = !in_quote && (b == b' ' || b == b'\t');
        if (b == b',' && !in_quote) || is_end {
            if i > start {
                options.push(line[start..i].to_string());
            }
            start = i + 1;
        }
        if is_end {
            ended = true;
            break;
        }
        // Go toggles on `"` unless backslash-escaped (first byte counts).
        if b == b'"' && (i == 0 || bytes[i - 1] != b'\\') {
            in_quote = !in_quote;
        }
        i += 1;
    }
    if !ended {
        // Go's range loop leaves `i` at the last index; unmatched quotes or
        // a missing key field both fail below.
        i = bytes.len().wrapping_sub(1);
    }
    while i < bytes.len() && (bytes[i] == b' ' || bytes[i] == b'\t') {
        i += 1;
    }
    if i >= bytes.len() {
        return None;
    }
    Some((options, &line[i..]))
}

pub struct AuthorizedKey {
    pub key: Key,
    pub options_empty: bool,
}

/// Parse one authorized-key line: `(key, options)` or an error. The trailing
/// `rest` of multi-line input cannot occur (callers pre-reject newlines).
pub fn parse_authorized_key(line: &str) -> Result<AuthorizedKey, ()> {
    let trimmed = go_trim_space(line);
    if trimmed.is_empty() || trimmed.starts_with('#') {
        return Err(());
    }
    let split = match trimmed.find([' ', '\t']) {
        Some(i) => i,
        None => return Err(()),
    };
    let (declared, after) = (&trimmed[..split], &trimmed[split..]);
    if let Ok((key, _)) = parse_key_field(after.as_bytes()) {
        if declared == key.key_type() {
            return Ok(AuthorizedKey {
                key,
                options_empty: true,
            });
        }
    }
    // Options field at the beginning.
    let (options, rest) = scan_options(trimmed).ok_or(())?;
    let split = match rest.find([' ', '\t']) {
        Some(i) => i,
        None => return Err(()),
    };
    let (declared, after) = (&rest[..split], &rest[split..]);
    match parse_key_field(after.as_bytes()) {
        Ok((key, _)) if declared == key.key_type() => Ok(AuthorizedKey {
            key,
            options_empty: options.is_empty(),
        }),
        _ => Err(()),
    }
}

/// x/crypto `ParseAuthorizedKey` over raw bytes: blank and comment lines
/// are skipped and the first parseable line wins. Carriage returns
/// truncate the line, as in Go. Lossy decoding is outcome-equivalent
/// here: only ASCII bytes take part in trimming, splitting, option
/// scanning, and base64, and the comment is discarded by every caller.
pub fn parse_authorized_key_bytes(mut input: &[u8]) -> Result<AuthorizedKey, ()> {
    loop {
        let line;
        match input.iter().position(|b| *b == b'\n') {
            Some(i) => {
                line = &input[..i];
                input = &input[i + 1..];
            }
            None => {
                line = input;
                input = &[];
            }
        };
        let line = match line.iter().position(|b| *b == b'\r') {
            Some(i) => &line[..i],
            None => line,
        };
        match parse_authorized_key(&String::from_utf8_lossy(line)) {
            Ok(key) => return Ok(key),
            Err(_) if input.is_empty() => return Err(()),
            Err(_) => {}
        }
    }
}

/// Installer `PublicKey`: validated, comment-stripped, canonical `type b64`.
pub fn public_key(value: &str) -> Result<String, &'static str> {
    if value.len() > 16384 || value.contains(['\r', '\n', '\0']) {
        return Err("one SSH public key required");
    }
    let parsed = parse_authorized_key(value)
        .map_err(|_| "valid SSH public key without authorized_keys options required")?;
    if !parsed.options_empty {
        return Err("valid SSH public key without authorized_keys options required");
    }
    match parsed.key {
        Key::Rsa { .. }
        | Key::Ecdsa { .. }
        | Key::Ed25519 { .. }
        | Key::SkEcdsa { .. }
        | Key::SkEd25519 { .. } => {}
        Key::Dss | Key::Cert { .. } => return Err("unsupported operator SSH key type"),
    }
    Ok(format!(
        "{} {}",
        parsed.key.key_type(),
        b64_encode(&parsed.key.marshal())
    ))
}

/// `ssh.FingerprintSHA256` over canonical key wire bytes.
pub fn fingerprint_sha256_wire(wire: &[u8]) -> String {
    let digest = Sha256::digest(wire);
    format!("SHA256:{}", b64_encode_raw(&digest))
}

#[cfg(test)]
mod tests {
    use super::*;

    const ED25519: &str =
        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKQ0MsA1tWa7risZNVfq58qNB9BByJfSJUQpWI9KvglV";
    const RSA: &str = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCoDZAt7ewKwwXBa7tCvC9/+p//wMupqSjGVnTOvoYOUt1UNbromMK5hBZGq2xIlqJQ0rRZoTEtsE7w5BUgkNzvsioTnnjD48UeqMvJhBfSrJYTVMZeM/ttm1jmlNhbjs6nt98R/KmdsyK+2a+z5BQ+KRlr4kbKfd/UDOPj8XuA2/vW4K2301UUdDk9Jh2r/bcjRnrIyHUX1Rmga608tAWRZtJQRo+8/28JqnjQM5s4qcu1d1N2Y823P4YGaLYhRLoKhV1/gCMRRhD9ZTZpn58sVmiEGQ90YeHE/8vETm+Q2IkjZvx2vobzhvdsA3LGs3B1EN2kdbCGJRqaltrJLK+t";
    const ECDSA256: &str = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBLtQL7Kq7o6tng5YEyRN3ICkkd2BErzxr+p3zkns1Apc0BJ7BDcTZqzWuusUsWLZxRtnOVtz2FT2vd0GlCz20RI=";
    const ECDSA384: &str = "ecdsa-sha2-nistp384 AAAAE2VjZHNhLXNoYTItbmlzdHAzODQAAAAIbmlzdHAzODQAAABhBNuz0aERM3G5ldqem61DeWNH6TPAuvKqwTrF2yCFg4tphB+K2icoJtn+oELywJfa5vRZmifM9zYAviqIcFjRIWZw2eeZQBdu8563omqOcS8x0JMVFFRTuPhpASQwnaK2OA==";
    const ECDSA521: &str = "ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBAB6wXlDnMlkFpqHCbvCbzARmcMPfFnBWBe9lNPrdal/1+t6X2tGURdQ2CGo7D15m/c5gN9fLlECNwYumO8Zo/9ctQFkH+iiqkuOJvaivvK56hHpb8ytWB9FPouFtXfzjfeuwqkpix6MR54fQbFSXgHEfqZPmf11b3+n1yJzi7nyV8EYMQ==";

    fn sk_ed25519_wire(extra: &[u8]) -> String {
        let mut wire = marshal_string(b"sk-ssh-ed25519@openssh.com");
        wire.extend(marshal_string(&[0x42; 32]));
        wire.extend(marshal_string(b"ssh:"));
        wire.extend_from_slice(extra);
        format!("sk-ssh-ed25519@openssh.com {}", b64_encode(&wire))
    }

    #[test]
    fn real_keys_parse_and_normalize() {
        for key in [ED25519, RSA, ECDSA256, ECDSA384, ECDSA521] {
            assert_eq!(public_key(key).unwrap(), key, "round trip");
            assert_eq!(public_key(&format!("{key} test-comment")).unwrap(), key);
            assert_eq!(public_key(&format!("  {key}  ")).unwrap(), key);
            assert_eq!(public_key(&format!("{key}\tcomment")).unwrap(), key);
        }
        let key = parse_authorized_key(ED25519).unwrap().key;
        assert_eq!(
            fingerprint_sha256_wire(&key.marshal()),
            "SHA256:8zz4BxDGZ75OGjyLd0V7cLMngT+KWeelWxFYt6+fKtc"
        );
        // Raw tool output with a trailing newline parses like Go.
        let key = parse_authorized_key_bytes(format!("{ED25519}\n").as_bytes())
            .unwrap()
            .key;
        assert_eq!(key.key_type(), "ssh-ed25519");
    }

    #[test]
    fn rejects_match_go_taxonomy() {
        // "one SSH public key required": size/CTL pre-checks.
        assert_eq!(
            public_key(&"x".repeat(16385)).unwrap_err(),
            "one SSH public key required"
        );
        assert_eq!(
            public_key(&format!("{ED25519}\n")).unwrap_err(),
            "one SSH public key required"
        );
        assert_eq!(
            public_key(&format!("{ED25519}\r")).unwrap_err(),
            "one SSH public key required"
        );
        assert_eq!(
            public_key(&format!("{ED25519}\0")).unwrap_err(),
            "one SSH public key required"
        );
        // "valid SSH public key ...": unparsable, options, or key material.
        let opt_cases = [
            format!("command=\"x\" {ED25519}"),
            format!("no-pty {ED25519}"),
            format!("restrict {ED25519}"),
            format!("\"\" {ED25519}"),
            ",,,ssh-ed25519 AAAA".to_string(),
        ];
        for bad in [
            "AAA",
            "ssh-ed25519",
            "",
            "   ",
            "# just a comment",
            "ssh-ed25519 AAAA",
            "ssh-ed25519 !!!",
            "ssh-rsa AAAA",
            "ssh-ed25519-cert-v01@openssh.com AAAA",
        ] {
            assert_eq!(
                public_key(bad).unwrap_err(),
                "valid SSH public key without authorized_keys options required",
                "input {bad:?}"
            );
        }
        for bad in &opt_cases {
            assert_eq!(
                public_key(bad).unwrap_err(),
                "valid SSH public key without authorized_keys options required",
                "input {bad:?}"
            );
        }
        // Empty options (commas only) parse with no options, as in Go.
        assert_eq!(public_key(&format!(",,, {ED25519}")).unwrap(), ED25519);
        assert_eq!(public_key(&format!(" , {ED25519}")).unwrap(), ED25519);
        // Declared/embedded type mismatch falls through to options parsing.
        let swapped = format!("ssh-rsa {}", ED25519.split(' ').nth(1).unwrap());
        assert_eq!(
            public_key(&swapped).unwrap_err(),
            "valid SSH public key without authorized_keys options required"
        );
    }

    #[test]
    fn unsupported_types_reported() {
        // Valid DSA wire (P 1024-bit, Q 160-bit) parses but is unsupported.
        let p = vec![0x81u8]
            .into_iter()
            .chain(vec![0u8; 127])
            .collect::<Vec<u8>>();
        let mut wire = marshal_string(b"ssh-dss");
        wire.extend(marshal_mpint(false, &p));
        wire.extend(marshal_mpint(false, &[0x81; 20]));
        wire.extend(marshal_mpint(false, &[0x02]));
        wire.extend(marshal_mpint(false, &[0x03]));
        let dss = format!("ssh-dss {}", b64_encode(&wire));
        assert_eq!(
            public_key(&dss).unwrap_err(),
            "unsupported operator SSH key type"
        );
    }

    #[test]
    fn rsa_canonicalizes_padded_mpints() {
        // Oracle: RSA-PADDED accepted with marshal_clean=true.
        let wire = b64_decode_go(RSA.split(' ').nth(1).unwrap().as_bytes()).unwrap();
        let (_, rest) = parse_string(&wire).unwrap();
        let (_, e_mag, rest) = parse_mpint(rest).unwrap();
        let (_, n_mag, _) = parse_mpint(rest).unwrap();
        let mut padded = marshal_string(b"ssh-rsa");
        padded.extend(marshal_string(&[vec![0, 0, 0], e_mag].concat()));
        padded.extend(marshal_string(&[vec![0, 0], n_mag].concat()));
        let line = format!("ssh-rsa {}", b64_encode(&padded));
        assert_eq!(public_key(&line).unwrap(), RSA);
        // Even exponent and oversized exponent are refused.
        let mut bad = marshal_string(b"ssh-rsa");
        bad.extend(marshal_string(&[4]));
        bad.extend(marshal_string(&[9u8; 64]));
        assert!(public_key(&format!("ssh-rsa {}", b64_encode(&bad))).is_err());
    }

    #[test]
    fn sk_keys_match_go_truncation_rule() {
        // x/crypto rejects trailing flags/handle bytes ("trailing junk");
        // only the truncated key+application form parses.
        assert!(public_key(&sk_ed25519_wire(&[])).is_ok());
        assert!(public_key(&sk_ed25519_wire(&[0x01])).is_err());
        let mut full = vec![0x01u8];
        full.extend(marshal_string(b"handle"));
        full.extend(marshal_string(b""));
        assert!(public_key(&sk_ed25519_wire(&full)).is_err());
    }

    #[test]
    fn base64_matches_go_decode() {
        assert_eq!(b64_decode_go(b"").unwrap(), b"");
        assert_eq!(b64_decode_go(b"QUI=").unwrap(), b"AB");
        assert_eq!(b64_decode_go(b"QUI").unwrap(), b"AB"); // lenient tail
        assert_eq!(b64_decode_go(b"QUJD").unwrap(), b"ABC");
        assert!(b64_decode_go(b"Q").is_err());
        assert!(b64_decode_go(b"AB=C").is_err());
        assert!(b64_decode_go(b"A===").is_err());
        assert!(b64_decode_go(b"AB==CD").is_err());
        assert_eq!(b64_encode(b"AB"), "QUI=");
        assert_eq!(b64_encode_raw(b"AB"), "QUI");
    }

    #[test]
    fn mpint_round_trip() {
        for (neg, mag) in [
            (false, vec![]),
            (false, vec![1]),
            (false, vec![0x80]),
            (true, vec![1]),
            (true, vec![0x80]),
        ] {
            let encoded = marshal_mpint(neg, &mag);
            let (n, m, rest) = parse_mpint(&encoded).unwrap();
            assert!(rest.is_empty());
            assert_eq!((n, m.clone()), (neg, mag));
            assert_eq!(marshal_mpint(n, &m), encoded);
        }
    }
}
