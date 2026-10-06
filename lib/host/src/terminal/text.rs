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

/// One `decodeQuantum` step of Go's `base64.StdEncoding` with `Strict()`.
/// Newlines are skipped (Go `Strict` still skips `\r\n`; the input path
/// rejects them separately via [`contains_crlf`]); trailing padding bits
/// must be zero.
fn strict_quantum(src: &[u8], mut si: usize, out: &mut Vec<u8>) -> Option<usize> {
    let mut dbuf = [0u8; 4];
    let mut dlen = 4usize;
    let mut j = 0usize;
    while j < 4 {
        if si == src.len() {
            if j == 0 {
                return Some(si);
            }
            return None;
        }
        let b = src[si];
        si += 1;
        if let Some(v) = b64_value(b) {
            dbuf[j] = v;
            j += 1;
            continue;
        }
        if b == b'\n' || b == b'\r' {
            continue;
        }
        if b != b'=' {
            return None;
        }
        match j {
            0 | 1 => return None,
            2 => {
                while si < src.len() && (src[si] == b'\n' || src[si] == b'\r') {
                    si += 1;
                }
                if si == src.len() {
                    return None;
                }
                if src[si] != b'=' {
                    return None;
                }
                si += 1;
            }
            _ => {}
        }
        while si < src.len() && (src[si] == b'\n' || src[si] == b'\r') {
            si += 1;
        }
        if si < src.len() {
            return None;
        }
        dlen = j;
        break;
    }
    // Go `Strict` trailing-bit checks: unused low bits must be zero.
    if dlen == 2 && dbuf[1] & 15 != 0 {
        return None;
    }
    if dlen == 3 && dbuf[2] & 3 != 0 {
        return None;
    }
    let val = (u32::from(dbuf[0]) << 18)
        | (u32::from(dbuf[1]) << 12)
        | (u32::from(dbuf[2]) << 6)
        | u32::from(dbuf[3]);
    out.push((val >> 16) as u8);
    if dlen >= 3 {
        out.push((val >> 8) as u8);
    }
    if dlen >= 4 {
        out.push(val as u8);
    }
    Some(si)
}

/// Go `base64.StdEncoding.Strict().DecodeString`, byte for byte.
pub fn strict_b64_decode(src: &str) -> Option<Vec<u8>> {
    let bytes = src.as_bytes();
    let mut out = Vec::with_capacity(bytes.len() / 4 * 3);
    let mut si = 0;
    while si < bytes.len() {
        si = strict_quantum(bytes, si, &mut out)?;
    }
    Some(out)
}

pub(in crate::terminal) fn contains_crlf(s: &str) -> bool {
    s.bytes().any(|b| b == b'\r' || b == b'\n')
}
