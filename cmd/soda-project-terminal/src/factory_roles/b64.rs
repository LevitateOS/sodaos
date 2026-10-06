//! Strict standard-alphabet base64 (`validate=True` decode).

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
    if !bytes.len().is_multiple_of(4) {
        return None;
    }
    if bytes.is_empty() {
        return Some(Vec::new());
    }
    let mut out = Vec::with_capacity(bytes.len() / 4 * 3);
    let (chunks, rest) = bytes.as_chunks::<4>();
    debug_assert!(rest.is_empty());
    let total = chunks.len();
    for (index, chunk) in chunks.iter().enumerate() {
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
        assert!(decode("YW=J").is_none());
        assert!(decode("====").is_none());
        assert!(decode("abc").is_none());
        assert!(decode("ab=d").is_none());
        assert!(decode("a===").is_none());
        assert!(decode("!!!").is_none());
        assert!(decode("!!!!").is_none());
        // Whitespace rejected under validate=True.
        assert!(decode("TW\nFu").is_none());
        assert!(decode("TWFu\n").is_none());
    }
}
