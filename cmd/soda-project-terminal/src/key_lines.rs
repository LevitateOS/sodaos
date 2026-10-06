/// Stdin read cap: `sys.stdin.buffer.read(65537)` reads at most this many
/// bytes (longer input is silently truncated, never an explicit error).
pub const STDIN_LIMIT: usize = 65537;
/// Managed key file bound mirrored in `canonical_lines`.
pub const KEY_FILE_LIMIT: usize = 65536;
/// Maximum managed keys per file.
pub const KEY_COUNT_LIMIT: usize = 32;

fn is_word(byte: u8) -> bool {
    byte.is_ascii_alphanumeric() || byte == b'_'
}

/// `[\w-]+` over ASCII (`\w` is ASCII-only here: the file decoded as ASCII).
fn word_dash_plus(text: &str) -> bool {
    !text.is_empty() && text.bytes().all(|b| is_word(b) || b == b'-')
}

/// `[\w@.-]+` over ASCII.
fn sk_plus(text: &str) -> bool {
    !text.is_empty()
        && text
            .bytes()
            .all(|b| is_word(b) || b == b'@' || b == b'.' || b == b'-')
}

/// One managed key line:
/// `(ssh-[\w-]+|ecdsa-[\w-]+|sk-[\w@.-]+) [A-Za-z0-9+/=]+` fullmatch.
/// Exactly one space; the blob is charset-only (padding position unchecked,
/// like the regex).
pub fn canonical_key_ok(key: &str) -> bool {
    let space = match key.find(' ') {
        Some(at) => at,
        None => return false,
    };
    if key.as_bytes()[space + 1..].contains(&b' ') {
        return false;
    }
    let (kind, blob) = (&key[..space], &key[space + 1..]);
    let kind_ok = kind
        .strip_prefix("ssh-")
        .map(word_dash_plus)
        .or_else(|| kind.strip_prefix("ecdsa-").map(word_dash_plus))
        .or_else(|| kind.strip_prefix("sk-").map(sk_plus))
        .unwrap_or(false);
    kind_ok
        && !blob.is_empty()
        && blob
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || b == b'+' || b == b'/' || b == b'=')
}

/// `canonical_lines`: empty file → no keys; otherwise at most 32 unique
/// ASCII lines with a trailing newline, each passing [`canonical_key_ok`].
pub fn canonical_lines(raw: &[u8]) -> Result<Vec<String>, String> {
    if raw.is_empty() {
        return Ok(Vec::new());
    }
    if raw.len() > KEY_FILE_LIMIT || !raw.ends_with(b"\n") {
        return Err("not a managed key file".to_string());
    }
    if !raw.is_ascii() {
        return Err("not a managed key file".to_string());
    }
    // SAFETY: ASCII checked above.
    let text = std::str::from_utf8(raw).map_err(|_| "not a managed key file".to_string())?;
    let keys: Vec<&str> = text[..text.len() - 1].split('\n').collect();
    if keys.len() > KEY_COUNT_LIMIT {
        return Err("not a managed key file".to_string());
    }
    for (i, key) in keys.iter().enumerate() {
        if keys[..i].contains(key) {
            return Err("not a managed key file".to_string());
        }
        if !canonical_key_ok(key) {
            return Err("not a managed key file".to_string());
        }
    }
    Ok(keys.into_iter().map(|k| k.to_string()).collect())
}
