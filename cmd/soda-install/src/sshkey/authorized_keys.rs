use sha2::{Digest, Sha256};

use super::base64::{b64_decode_go, b64_encode, b64_encode_raw};
use super::wire::parse_public_key;
use super::Key;

use crate::fmtx::go_trim_space;

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
