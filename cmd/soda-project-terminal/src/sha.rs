//! SHA-256 helpers for key revisions and approved setup inputs.

pub use sha2::{Digest, Sha256};

pub fn digest(data: &[u8]) -> [u8; 32] {
    Sha256::digest(data).into()
}

pub fn hex(data: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(data.len() * 2);
    for &byte in data {
        out.push(HEX[(byte >> 4) as usize] as char);
        out.push(HEX[(byte & 0x0f) as usize] as char);
    }
    out
}

pub fn hex_digest(data: &[u8]) -> String {
    hex(&digest(data))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn nist_vectors_and_streaming_partitions() {
        assert_eq!(
            hex_digest(b""),
            "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
        );
        assert_eq!(
            hex_digest(b"abc"),
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
        assert_eq!(
            hex_digest(b"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq"),
            "248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1"
        );

        let input = vec![b'a'; 1_000];
        let expected = digest(&input);
        for split in [0, 55, 56, 57, 63, 64, 65, 119, 120, 129, 999, 1_000] {
            let mut hasher = Sha256::new();
            hasher.update(&input[..split]);
            hasher.update(&input[split..]);
            assert_eq!(&hasher.finalize()[..], expected);
        }
    }
}
