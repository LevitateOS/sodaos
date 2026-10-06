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

mod base64;
mod certificate;
mod material;
mod mpint;

#[cfg(test)]
mod tests;

pub use self::base64::{b64_corrupt, b64_decode, b64_decode_go, b64_encode, b64_encode_raw};
pub use self::material::ParsedKey;

use self::material::{marshal_material, parse_key_fields};
use self::mpint::read_string;

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

// ---------- key types ----------

pub const ALGO_RSA: &str = "ssh-rsa";
pub const ALGO_DSS: &str = "ssh-dss";
pub const ALGO_ECDSA256: &str = "ecdsa-sha2-nistp256";
pub const ALGO_ECDSA384: &str = "ecdsa-sha2-nistp384";
pub const ALGO_ECDSA521: &str = "ecdsa-sha2-nistp521";
pub const ALGO_SKECDSA: &str = "sk-ecdsa-sha2-nistp256@openssh.com";
pub const ALGO_ED25519: &str = "ssh-ed25519";
pub const ALGO_SKED25519: &str = "sk-ssh-ed25519@openssh.com";

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
