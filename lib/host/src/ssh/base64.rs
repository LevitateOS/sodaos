// ---------- base64 (strict standard alphabet) ----------

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

pub fn b64_decode(s: &str) -> Result<Vec<u8>, String> {
    let b = s.as_bytes();
    if b.is_empty() || !b.len().is_multiple_of(4) {
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

/// One `decodeQuantum` step of `base64.StdEncoding.Decode` (pinned Go
/// toolchain): `\r`/`\n` skipped, `CorruptInputError` offsets exact.
/// Returns the new read index; partial output on error is discarded by
/// every caller, exactly like `encoding/json` discards it.
fn decode_quantum_go(src: &[u8], mut si: usize, out: &mut Vec<u8>) -> Result<usize, usize> {
    let mut dbuf = [0u8; 4];
    let mut dlen = 4usize;
    let mut j = 0usize;
    while j < 4 {
        if src.len() == si {
            if j == 0 {
                return Ok(si);
            }
            return Err(si - j);
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
            return Err(si - 1);
        }
        match j {
            0 | 1 => return Err(si - 1),
            2 => {
                while si < src.len() && (src[si] == b'\n' || src[si] == b'\r') {
                    si += 1;
                }
                if si == src.len() {
                    return Err(src.len());
                }
                if src[si] != b'=' {
                    return Err(si - 1);
                }
                si += 1;
            }
            _ => {}
        }
        while si < src.len() && (src[si] == b'\n' || src[si] == b'\r') {
            si += 1;
        }
        if si < src.len() {
            return Err(si);
        }
        dlen = j;
        break;
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
    Ok(si)
}

/// `base64.StdEncoding.Decode` for JSON `[]byte` fields: decoded bytes on
/// success, the `CorruptInputError` byte offset on failure.
pub fn b64_decode_go(src: &[u8]) -> Result<Vec<u8>, usize> {
    if src.is_empty() {
        return Ok(Vec::new());
    }
    let mut out = Vec::with_capacity(src.len().div_ceil(4) * 3);
    let mut si = 0;
    while si < src.len() {
        si = decode_quantum_go(src, si, &mut out)?;
    }
    Ok(out)
}

/// `CorruptInputError.Error()`: `illegal base64 data at input byte N`.
pub fn b64_corrupt(offset: usize) -> String {
    format!("illegal base64 data at input byte {offset}")
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
