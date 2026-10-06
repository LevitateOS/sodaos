//! SHA-256 helpers for pinned binary checks and provider identifiers.

use sha2::{Digest, Sha256};
use std::io::{self, Read};

pub fn digest(data: &[u8]) -> [u8; 32] {
    Sha256::digest(data).into()
}

pub fn digest_reader(reader: &mut dyn Read) -> io::Result<[u8; 32]> {
    let mut hasher = Sha256::new();
    let mut buf = [0u8; 8192];
    loop {
        let n = reader.read(&mut buf)?;
        if n == 0 {
            return Ok(hasher.finalize().into());
        }
        hasher.update(&buf[..n]);
    }
}

pub fn hex(bytes: &[u8]) -> String {
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
    fn standard_vectors() {
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
            assert_eq!(hex(&digest(input)), want);
        }

        let mut hasher = Sha256::new();
        for _ in 0..1_000_000 {
            hasher.update(b"a");
        }
        assert_eq!(
            hex(&hasher.finalize()),
            "cdc76e5c9914fb9281a1c7e284d73e67f1809a48a497200e046d39ccc7112cd0"
        );
    }

    #[test]
    fn reader_streams_and_propagates_errors() {
        let data = vec![7u8; 100_000];
        let mut cursor = io::Cursor::new(&data);
        assert_eq!(digest_reader(&mut cursor).unwrap(), digest(&data));

        struct FailingReader;
        impl Read for FailingReader {
            fn read(&mut self, _: &mut [u8]) -> io::Result<usize> {
                Err(io::Error::other("read failed"))
            }
        }
        assert_eq!(
            digest_reader(&mut FailingReader).unwrap_err().to_string(),
            "read failed"
        );
    }
}
