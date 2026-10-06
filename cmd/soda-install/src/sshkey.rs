//! OpenSSH public-key parsing mirroring `golang.org/x/crypto/ssh`
//! (`ParseAuthorizedKey`, per-type wire validation, canonical
//! `MarshalAuthorizedKey`, `FingerprintSHA256`), backing `PublicKey`.

pub use self::base64::{b64_decode_go, b64_encode, b64_encode_raw};
pub use self::wire::Key;
// AuthorizedKey stays reachable as sshkey::AuthorizedKey; no in-crate namer exists yet.
#[allow(unused_imports)]
pub use self::authorized_keys::{
    fingerprint_sha256_wire, parse_authorized_key, parse_authorized_key_bytes, public_key,
    AuthorizedKey,
};
use self::wire::{marshal_mpint, marshal_string, parse_mpint, parse_string};

mod authorized_keys;
mod base64;
mod wire;

#[cfg(test)]
mod tests;
