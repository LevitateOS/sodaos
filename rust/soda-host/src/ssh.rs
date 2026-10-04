//! OpenSSH public-key handling for the `/connection` observation.
//!
//! Mirrors the `golang.org/x/crypto/ssh` calls in
//! `internal/host/project/create.go`: strict authorized-key parsing with no
//! options and no trailing comment, Ed25519-only admission,
//! newline-terminated marshaling, and `SHA256:` fingerprints.

use crate::sha256;

const ED25519: &str = "ssh-ed25519";

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

/// Strict standard-alphabet base64 decode (no whitespace, correct padding).
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

fn read_string(buf: &[u8]) -> Result<(&[u8], &[u8]), String> {
    if buf.len() < 4 {
        return Err("short key blob".to_string());
    }
    let len = u32::from_be_bytes([buf[0], buf[1], buf[2], buf[3]]) as usize;
    if buf.len() < 4 + len {
        return Err("short key blob".to_string());
    }
    Ok((&buf[4..4 + len], &buf[4 + len..]))
}

/// Parse one authorized-key line and return the raw public-key blob.
/// Exactly two fields (type, base64) with optional surrounding whitespace;
/// anything else (options, comments, extra fields) is refused, matching the
/// Go call sites that reject non-empty options and non-blank trailing data.
pub fn parse_authorized_key(line: &[u8]) -> Result<Vec<u8>, String> {
    let text = std::str::from_utf8(line).map_err(|_| "invalid public key".to_string())?;
    let mut fields = text.split_whitespace();
    let (Some(kind), Some(data), None) = (fields.next(), fields.next(), fields.next()) else {
        return Err("invalid public key".to_string());
    };
    if kind != ED25519 {
        return Err("invalid public key".to_string());
    }
    let blob = b64_decode(data).map_err(|_| "invalid public key".to_string())?;
    let (name, rest) = read_string(&blob).map_err(|_| "invalid public key".to_string())?;
    let (key, rest) = read_string(rest).map_err(|_| "invalid public key".to_string())?;
    if name != ED25519.as_bytes() || key.len() != 32 || !rest.is_empty() {
        return Err("invalid public key".to_string());
    }
    Ok(blob)
}

/// `ssh.MarshalAuthorizedKey`: `type base64` plus the trailing newline.
pub fn marshal_authorized_key(blob: &[u8]) -> String {
    format!("{ED25519} {}\n", b64_encode(blob))
}

/// `ssh.FingerprintSHA256`: `SHA256:` plus unpadded base64 of the digest.
pub fn fingerprint_sha256(blob: &[u8]) -> String {
    let digest = sha256::digest(blob);
    format!("SHA256:{}", b64_encode_raw(&digest))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn fixture_line() -> String {
        let mut blob = vec![0, 0, 0, 11];
        blob.extend_from_slice(b"ssh-ed25519");
        blob.extend_from_slice(&[0, 0, 0, 32]);
        blob.extend_from_slice(&[0x42; 32]);
        format!("ssh-ed25519 {}", b64_encode(&blob))
    }

    #[test]
    fn key_round_trip_matches_go() {
        // 32-byte key body of 0x61 ('a').
        let mut blob = vec![0, 0, 0, 11];
        blob.extend_from_slice(b"ssh-ed25519");
        blob.extend_from_slice(&[0, 0, 0, 32]);
        blob.extend_from_slice(&[0x61; 32]);
        let line = marshal_authorized_key(&blob);
        assert!(line.ends_with('\n'));
        assert_eq!(parse_authorized_key(line.as_bytes()).unwrap(), blob);
        let fp = fingerprint_sha256(&blob);
        assert!(fp.starts_with("SHA256:") && !fp.ends_with('='));
        assert_eq!(fp.len(), 7 + 43); // 32-byte digest, unpadded
    }

    #[test]
    fn base64_vectors() {
        assert_eq!(b64_decode("").unwrap_err(), "invalid base64".to_string());
        assert_eq!(b64_decode("QUI=").unwrap(), b"AB");
        assert_eq!(b64_decode("QUI"), Err("invalid base64".to_string()));
        assert_eq!(b64_decode("AB=C"), Err("invalid base64".to_string()));
        assert_eq!(b64_encode(b"AB"), "QUI=");
        assert_eq!(b64_encode_raw(b"AB"), "QUI");
        // A well-formed ed25519 line parses to its 51-byte blob.
        assert_eq!(
            parse_authorized_key(fixture_line().as_bytes())
                .unwrap()
                .len(),
            51
        );
    }

    #[test]
    fn rejects_options_comments_and_wrong_types() {
        assert!(parse_authorized_key(b"no-pty ssh-ed25519 AAAA").is_err());
        assert!(parse_authorized_key(format!("{} comment", fixture_line()).as_bytes()).is_err());
        assert!(parse_authorized_key(b"ssh-rsa AAAA").is_err());
        assert!(parse_authorized_key(b"ssh-ed25519 !!!").is_err());
        assert!(parse_authorized_key(b"ssh-ed25519").is_err());
        assert!(parse_authorized_key(b"").is_err());
    }
}
