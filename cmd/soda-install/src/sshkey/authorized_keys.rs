use super::base64::{b64_decode_go, b64_encode, b64_encode_raw};
use super::wire::{parse_public_key, Kind};
use super::Key;
use sha2::{Digest, Sha256};

pub struct AuthorizedKey {
    pub key: Key,
    pub options_empty: bool,
}

/// Parse one authorized-key line: `(key, options)` or an error. The trailing
/// `rest` of multi-line input cannot occur (callers pre-reject newlines).
pub fn parse_authorized_key(line: &str) -> Result<AuthorizedKey, ()> {
    let trimmed = line.trim();
    if trimmed.is_empty() || trimmed.starts_with('#') {
        return Err(());
    }
    let fields: Vec<_> = trimmed.split_whitespace().collect();
    let offset = usize::from(
        fields
            .first()
            .is_some_and(|field| !field.is_empty() && field.bytes().all(|byte| byte == b',')),
    );
    let (declared, encoded) = fields.get(offset).zip(fields.get(offset + 1)).ok_or(())?;
    let wire = b64_decode_go(encoded.as_bytes())?;
    let key = parse_public_key(&wire)?;
    if key.key_type() != *declared {
        return Err(());
    }
    Ok(AuthorizedKey {
        key,
        options_empty: true,
    })
}

/// Parse raw tool output line by line, skipping blank/comment/malformed lines
/// and truncating each line at its first carriage return. Comments are unused.
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
    match parsed.key.kind() {
        Kind::Rsa | Kind::Ecdsa | Kind::Ed25519 | Kind::SkEcdsa | Kind::SkEd25519 => {}
        Kind::Dsa | Kind::Certificate => return Err("unsupported operator SSH key type"),
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
