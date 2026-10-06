// ---------------------------------------------------------------------------
// Base64: exact Go `StdEncoding.Decode` (lenient trailing quantum, strict
// padding placement, ignored `\r\n`) and canonical padded encode.
// ---------------------------------------------------------------------------

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

pub fn b64_decode_go(input: &[u8]) -> Result<Vec<u8>, ()> {
    // Strip newlines exactly as Go's decoder skips them mid-quantum.
    let mut src: Vec<u8> = Vec::with_capacity(input.len());
    for &b in input {
        if b != b'\r' && b != b'\n' {
            src.push(b);
        }
    }
    let mut out = Vec::with_capacity(src.len() / 4 * 3 + 3);
    let mut i = 0;
    while i < src.len() {
        // One quantum: up to 4 significant chars; padding ends the input.
        let mut quantum = [0u8; 4];
        let mut have = 0;
        while have < 4 && i < src.len() {
            let c = src[i];
            if c == b'=' {
                break;
            }
            quantum[have] = b64_value(c).ok_or(())?;
            have += 1;
            i += 1;
        }
        if i < src.len() && src[i] == b'=' {
            // Padding at quantum positions 0/1 is corrupt; at 2 it needs
            // the second `=`; nothing may follow padding.
            if have < 2 {
                return Err(());
            }
            i += 1;
            if have == 2 {
                if i >= src.len() || src[i] != b'=' {
                    return Err(());
                }
                i += 1;
            }
            if i < src.len() {
                return Err(());
            }
        } else if have == 1 {
            // A 1-char tail is corrupt; 2/3-char tails decode short.
            return Err(());
        } else if have == 0 {
            break;
        }
        let n = (u32::from(quantum[0]) << 18)
            | (u32::from(quantum[1]) << 12)
            | (u32::from(quantum[2]) << 6)
            | u32::from(quantum[3]);
        // Non-strict trailing bits are ignored, as in Go.
        out.push((n >> 16) as u8);
        if have >= 3 {
            out.push((n >> 8) as u8);
        }
        if have == 4 {
            out.push(n as u8);
        }
    }
    Ok(out)
}

pub fn b64_encode(raw: &[u8]) -> String {
    const ALPHA: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity(raw.len().div_ceil(3) * 4);
    for chunk in raw.chunks(3) {
        let mut n: u32 = 0;
        for &b in chunk {
            n = (n << 8) | u32::from(b);
        }
        n <<= 8 * (3 - chunk.len());
        for i in 0..4 {
            if i <= chunk.len() {
                out.push(ALPHA[((n >> (18 - 6 * i)) & 63) as usize] as char);
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
