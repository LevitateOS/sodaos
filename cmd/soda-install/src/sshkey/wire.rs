use elliptic_curve::sec1::ToEncodedPoint;

// ---------------------------------------------------------------------------
// SSH wire primitives.
// ---------------------------------------------------------------------------

pub(super) fn parse_string(input: &[u8]) -> Result<(&[u8], &[u8]), ()> {
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
pub(super) fn parse_mpint(input: &[u8]) -> Result<(bool, Vec<u8>, &[u8]), ()> {
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
pub(super) fn marshal_mpint(negative: bool, mag: &[u8]) -> Vec<u8> {
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

pub(super) fn marshal_string(bytes: &[u8]) -> Vec<u8> {
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
pub(super) fn parse_public_key(wire: &[u8]) -> Result<Key, ()> {
    let (algo, rest) = parse_string(wire)?;
    let algo = std::str::from_utf8(algo).map_err(|_| ())?;
    let (key, rest) = parse_pub_key(rest, algo)?;
    if !rest.is_empty() {
        return Err(());
    }
    Ok(key)
}
