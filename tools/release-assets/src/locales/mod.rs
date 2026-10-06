//! Locked Forgejo locale catalog builder (ex `soda-forgejo-locales`).
//!
//! Ports `scripts/forgejo-locales.py` to one binary. Every
//! success output (the merged catalog bytes) and every validation message
//! matches the owning script; only the argparse envelope (usage preface,
//! `prog: error:` prefix) carries the new binary name, and unpinned
//! failure text (usage errors, malformed-input tracebacks, transport
//! errors) differs.
//!
//! Exit codes mirror the script: `0` on success, `2` on argument and lock
//! refusals (`parser.error`), `1` on anything else (traceback equivalents).

pub mod merge;

/// Tokenize one argv element: `--name=value` gives `("name", Some(value))`,
/// `--name` gives `("name", None)`. Returns `None` for non-flag tokens
/// (including a bare `--`).
pub fn split_flag(arg: &str) -> Option<(String, Option<String>)> {
    if arg == "--" || !arg.starts_with("--") {
        return None;
    }
    let body = &arg[2..];
    if body.is_empty() {
        return None;
    }
    match body.split_once('=') {
        Some((name, value)) => {
            if name.is_empty() {
                return None;
            }
            Some((name.to_string(), Some(value.to_string())))
        }
        None => Some((body.to_string(), None)),
    }
}

/// Lowercase-hex SHA-256, like `hashlib.sha256().hexdigest()`.
pub fn sha256_hex(data: &[u8]) -> String {
    use sha2::Digest;
    let mut hasher = sha2::Sha256::new();
    hasher.update(data);
    let mut out = String::with_capacity(64);
    for byte in hasher.finalize() {
        out.push_str(&format!("{byte:02x}"));
    }
    out
}
