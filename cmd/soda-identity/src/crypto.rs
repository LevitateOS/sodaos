// AES-256-GCM credential custody, mirroring internal/store/grants.go:
// random 12-byte nonce prepended to the ciphertext, connection binding
// as additional authenticated data.
use crate::wire::Error;
use aes_gcm::aead::{Aead, Payload};
use aes_gcm::{Aes256Gcm, KeyInit};

pub const KEY_BINDING: &str = "soda/session-grants/key-check/v1";
const GRANT_KEY_ERROR: &str = "identity credential encryption key missing or incorrect";

pub struct GrantCipher {
    cipher: Aes256Gcm,
}

impl GrantCipher {
    pub fn new(key: &[u8]) -> Result<GrantCipher, Error> {
        if key.len() != 32 {
            return Err(Error::internal(GRANT_KEY_ERROR));
        }
        let cipher =
            Aes256Gcm::new_from_slice(key).map_err(|_| Error::internal(GRANT_KEY_ERROR))?;
        Ok(GrantCipher { cipher })
    }

    pub fn seal(&self, data: &[u8], binding: &str) -> Vec<u8> {
        self.seal_with_random(data, binding, |out| {
            getrandom::fill(out).map_err(std::io::Error::other)
        })
    }

    fn seal_with_random(
        &self,
        data: &[u8],
        binding: &str,
        fill: impl FnOnce(&mut [u8]) -> std::io::Result<()>,
    ) -> Vec<u8> {
        let mut raw = [0u8; 12];
        // crypto/rand fills the buffer or terminates the process; match that.
        fill(&mut raw).expect("credential randomness unavailable");
        let nonce = aes_gcm::Nonce::from(raw);
        let body = self
            .cipher
            .encrypt(
                &nonce,
                Payload {
                    msg: data,
                    aad: binding.as_bytes(),
                },
            )
            .expect("credential seal failed");
        let mut out = Vec::with_capacity(12 + body.len());
        out.extend_from_slice(&raw);
        out.extend_from_slice(&body);
        out
    }

    pub fn open(&self, data: &[u8], binding: &str) -> Result<Vec<u8>, Error> {
        if data.len() < 12 {
            return Err(Error::internal(GRANT_KEY_ERROR));
        }
        let mut raw = [0u8; 12];
        raw.copy_from_slice(&data[..12]);
        self.cipher
            .decrypt(
                &aes_gcm::Nonce::from(raw),
                Payload {
                    msg: &data[12..],
                    aad: binding.as_bytes(),
                },
            )
            .map_err(|_| Error::internal(GRANT_KEY_ERROR))
    }

    pub fn nonce_size() -> usize {
        12
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_short_keys() {
        assert!(GrantCipher::new(&[0u8; 31]).is_err());
        assert!(GrantCipher::new(&[0u8; 32]).is_ok());
    }

    #[test]
    fn round_trip_with_binding() {
        let cipher = GrantCipher::new(&[7u8; 32]).unwrap();
        let sealed = cipher.seal(br#"{"tokens":{}}"#, "soda/identity/conn/1");
        assert_eq!(sealed.len(), 12 + 13 + 16);
        assert_eq!(
            cipher.open(&sealed, "soda/identity/conn/1").unwrap(),
            br#"{"tokens":{}}"#
        );
        assert!(cipher.open(&sealed, "soda/identity/conn/2").is_err());
        assert!(cipher.open(&sealed[..11], "soda/identity/conn/1").is_err());
        let mut tampered = sealed.clone();
        let last = tampered.len() - 1;
        tampered[last] ^= 1;
        assert!(cipher.open(&tampered, "soda/identity/conn/1").is_err());
    }

    #[test]
    fn nist_vector() {
        // NIST SP 800-38D test case 2: empty plaintext authenticates the AAD.
        let cipher = GrantCipher::new(&[0u8; 32]).unwrap();
        let tag = cipher
            .cipher
            .encrypt(
                &aes_gcm::Nonce::from([0u8; 12]),
                Payload { msg: b"", aad: b"" },
            )
            .unwrap();
        assert_eq!(hex(&tag), "530f8afbc74536b9a963b4f1c4cb738b");
    }

    #[test]
    fn seal_fails_closed_after_partial_entropy_write() {
        let cipher = GrantCipher::new(&[7u8; 32]).unwrap();
        let result = std::panic::catch_unwind(|| {
            cipher.seal_with_random(b"secret", "binding", |out| {
                out[0] = 1;
                Err(std::io::Error::other("injected entropy failure"))
            });
        });
        assert!(result.is_err());
    }

    fn hex(bytes: &[u8]) -> String {
        bytes.iter().map(|b| format!("{b:02x}")).collect()
    }
}
