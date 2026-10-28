use base64::Engine;

use crate::domain::{in_ranges, GO_CC, GO_CF};

/// Port of Go `ValidTerminalName`: at most 80 runes, no Cc/Cf characters.
/// (`&str` is always valid UTF-8, so the `utf8.ValidString` check is free.)
pub fn valid_terminal_name(name: &str) -> bool {
    if name.chars().count() > 80 {
        return false;
    }
    for c in name.chars() {
        if in_ranges(GO_CC, c) || in_ranges(GO_CF, c) {
            return false;
        }
    }
    true
}

/// `terminalDimensions`: cols 2..=500, rows 2..=300.
pub fn terminal_dimensions(cols: i64, rows: i64) -> bool {
    (2..=500).contains(&cols) && (2..=300).contains(&rows)
}

// ---------- strict base64 (Go `StdEncoding.Strict` semantics) ----------

/// Go `base64.StdEncoding.Strict().DecodeString`. This helper retains Go's
/// CR/LF skipping; the terminal wire admission gate rejects CR/LF separately.
pub fn strict_b64_decode(src: &str) -> Option<Vec<u8>> {
    let compact: Vec<u8> = src
        .bytes()
        .filter(|byte| *byte != b'\r' && *byte != b'\n')
        .collect();
    base64::engine::general_purpose::STANDARD
        .decode(compact)
        .ok()
}

pub(in crate::terminal) fn contains_crlf(s: &str) -> bool {
    s.bytes().any(|b| b == b'\r' || b == b'\n')
}
