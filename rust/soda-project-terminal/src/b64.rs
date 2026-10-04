//! Strict standard-alphabet base64 (encode + `validate=True` decode).

const ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

fn value(byte: u8) -> Option<u8> {
    match byte {
        b'A'..=b'Z' => Some(byte - b'A'),
        b'a'..=b'z' => Some(byte - b'a' + 26),
        b'0'..=b'9' => Some(byte - b'0' + 52),
        b'+' => Some(62),
        b'/' => Some(63),
        _ => None,
    }
}

/// Decode with `validate=True` semantics: length multiple of 4 (empty
/// accepted), padding only at the end (max 2, never leading, never a lone
/// data char). Trailing-bit canonicity is NOT checked, like binascii.
pub fn decode(input: &str) -> Option<Vec<u8>> {
    let bytes = input.as_bytes();
    if bytes.len() % 4 != 0 {
        return None;
    }
    if bytes.is_empty() {
        return Some(Vec::new());
    }
    let mut out = Vec::with_capacity(bytes.len() / 4 * 3);
    let chunks = bytes.chunks_exact(4);
    let total = chunks.len();
    for (index, chunk) in chunks.enumerate() {
        let last = index + 1 == total;
        let mut values = [0u8; 4];
        let mut padding = 0;
        for (i, byte) in chunk.iter().enumerate() {
            if *byte == b'=' {
                if !last || i < 2 {
                    return None;
                }
                padding += 1;
                values[i] = 0;
            } else {
                if padding > 0 {
                    return None;
                }
                values[i] = value(*byte)?;
            }
        }
        let triple = (u32::from(values[0]) << 18)
            | (u32::from(values[1]) << 12)
            | (u32::from(values[2]) << 6)
            | u32::from(values[3]);
        out.push((triple >> 16) as u8);
        if padding < 2 {
            out.push((triple >> 8) as u8);
        }
        if padding == 0 {
            out.push(triple as u8);
        }
    }
    Some(out)
}

pub fn encode(input: &[u8]) -> String {
    let mut out = String::with_capacity(input.len().div_ceil(3) * 4);
    for chunk in input.chunks(3) {
        let mut block = [0u8; 3];
        block[..chunk.len()].copy_from_slice(chunk);
        let triple = (u32::from(block[0]) << 16) | (u32::from(block[1]) << 8) | u32::from(block[2]);
        out.push(ALPHABET[(triple >> 18) as usize] as char);
        out.push(ALPHABET[((triple >> 12) & 63) as usize] as char);
        if chunk.len() > 1 {
            out.push(ALPHABET[((triple >> 6) & 63) as usize] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(ALPHABET[(triple & 63) as usize] as char);
        } else {
            out.push('=');
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn vectors() {
        assert_eq!(decode("YWI=").unwrap(), b"ab");
        assert_eq!(decode("YQ==").unwrap(), b"a");
        assert_eq!(decode("TWFu").unwrap(), b"Man");
        assert_eq!(decode("").unwrap(), b"");
        // Non-canonical trailing bits accepted like binascii.
        assert_eq!(decode("YR==").unwrap(), b"a");
        assert_eq!(decode("YWE=").unwrap(), b"aa");
        // Padding placement.
        assert_eq!(decode("YW=J").is_none(), true);
        assert_eq!(decode("====").is_none(), true);
        assert_eq!(decode("abc").is_none(), true);
        assert_eq!(decode("ab=d").is_none(), true);
        assert_eq!(decode("a===").is_none(), true);
        // Round trip incl. credential sizes.
        for len in [0, 1, 2, 3, 55, 256] {
            let data: Vec<u8> = (0..len).map(|i| (i * 37 + 11) as u8).collect();
            if len == 0 {
                assert_eq!(encode(&data), "");
            } else {
                assert_eq!(decode(&encode(&data)).unwrap(), data);
            }
        }
    }
}
