use base64::Engine;

// Exact `unicode.Cc` / `unicode.Cf` ranges from the pinned Go toolchain
// (go1.26.7), mirroring `domain.rs` for `ValidTerminalName`.
const GO_CC: &[(u32, u32)] = &[(0x0, 0x1F), (0x7F, 0x9F)];
const GO_CF: &[(u32, u32)] = &[
    (0xAD, 0xAD),
    (0x600, 0x605),
    (0x61C, 0x61C),
    (0x6DD, 0x6DD),
    (0x70F, 0x70F),
    (0x890, 0x891),
    (0x8E2, 0x8E2),
    (0x180E, 0x180E),
    (0x200B, 0x200F),
    (0x202A, 0x202E),
    (0x2060, 0x2064),
    (0x2066, 0x206F),
    (0xFEFF, 0xFEFF),
    (0xFFF9, 0xFFFB),
    (0x110BD, 0x110BD),
    (0x110CD, 0x110CD),
    (0x13430, 0x1343F),
    (0x1BCA0, 0x1BCA3),
    (0x1D173, 0x1D17A),
    (0xE0001, 0xE0001),
    (0xE0020, 0xE007F),
];

fn in_ranges(table: &[(u32, u32)], c: char) -> bool {
    let v = c as u32;
    table.iter().any(|&(lo, hi)| v >= lo && v <= hi)
}

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
