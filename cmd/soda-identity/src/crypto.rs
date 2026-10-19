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
