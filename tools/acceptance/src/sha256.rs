//! SHA-256 helpers for acceptance evidence and file fingerprints.

pub use sha2::{Digest, Sha256};

pub fn digest(data: &[u8]) -> [u8; 32] {
    Sha256::digest(data).into()
}

/// Lowercase hex, matching Go's `hex.EncodeToString`.
pub fn hex_lower(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(bytes.len() * 2);
    for &byte in bytes {
        out.push(HEX[(byte >> 4) as usize] as char);
        out.push(HEX[(byte & 0x0f) as usize] as char);
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn nist_vectors_and_streaming_partitions() {
        let cases = [
            (
                b"".as_slice(),
                "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
            ),
            (
                b"abc".as_slice(),
                "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
            ),
            (
                b"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq".as_slice(),
                "248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1",
            ),
        ];
        for (input, want) in cases {
            assert_eq!(hex_lower(&digest(input)), want);
        }

        let data = vec![7u8; 100_000];
        let expected = digest(&data);
        for split in [
            0, 1, 55, 56, 57, 63, 64, 65, 119, 120, 128, 50_000, 99_999, 100_000,
        ] {
            let mut hasher = Sha256::new();
            hasher.update(&data[..split]);
            hasher.update(&data[split..]);
            assert_eq!(&hasher.finalize()[..], expected);
        }
    }
}
