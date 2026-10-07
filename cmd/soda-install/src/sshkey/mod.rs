//! Typed OpenSSH formats with installer key, line and fingerprint policy.

pub use self::base64::b64_encode;
#[cfg(test)]
pub use self::base64::{b64_decode_go, b64_encode_raw};
pub use self::wire::Key;
// AuthorizedKey stays reachable as sshkey::AuthorizedKey; no in-crate namer exists yet.
#[allow(unused_imports)]
pub use self::authorized_keys::{
    fingerprint_sha256_wire, parse_authorized_key, parse_authorized_key_bytes, public_key,
    AuthorizedKey,
};
mod authorized_keys;
mod base64;
mod wire;

#[cfg(test)]
mod tests;
