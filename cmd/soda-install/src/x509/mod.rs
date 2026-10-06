//! Go `crypto/x509` certificate subset used by the installer: DER parsing
//! with Go's exact accept/reject behavior and error text, plus
//! `CheckSignatureFrom` for self-signed local CA certificates.
//!
//! Design: a hand-rolled DER reader mirroring `golang.org/x/crypto/cryptobyte`
//! read strictness, with the parse/verify flow ported line for line from Go
//! 1.26.7 (`crypto/x509/parser.go`, `x509.go`). `x509-cert` was evaluated and
//! rejected: its `Rfc5280` profile enforces a 21-byte serial cap and a 1970
//! UTCTime floor that Go does not, and every parse error string would need
//! hand-mapping. Signature math delegates to `rsa`, `p224`/`p256`/`p384`/
//! `p521`, `ed25519-dalek`, and `sha2`.
//!
//! Scope: `setup` needs `IsCA`, `BasicConstraintsValid`, `KeyUsage`,
//! `Version`, the public key, the signature algorithm, `RawTBSCertificate`,
//! `Signature`, and `Raw`. Every other extension is still parsed and
//! validated (malformed values fail exactly as in Go) but its decoded value
//! is dropped. `Verify`, CRLs, CSRs, and creation APIs are not ported:
//! nothing in the installer uses them.
//!
//! `GODEBUG` escape hatches (`x509negativeserial`) are not honored: they are
//! Go-toolchain rollout controls, not installer behavior. Defaults match.
//!
//! Ed25519 verification uses `ed25519-dalek`'s `verify` (canonical `S`,
//! cofactored equation, no small-order checks), which is equivalent by
//! construction to Go's `verifyWithDom`. RSA verification failures map to
//! Go's `crypto/rsa: verification error`; Go returns that same error for
//! every RSA verification failure.

pub use self::algorithms::{PublicKeyAlgorithm, SignatureAlgorithm, KEY_USAGE_CERT_SIGN};
#[cfg(test)]
pub use self::algorithms::{
    KEY_USAGE_DECIPHER_ONLY, KEY_USAGE_DIGITAL_SIGNATURE, KEY_USAGE_KEY_ENCIPHERMENT,
};
pub use self::certificate::parse_certificate;
use self::der::*;
use self::name_constraints::parse_rfc2821_mailbox;
use self::names::parse_ai;
use self::public_key::{signature_algorithm_from_ai, P256_ELEMENT_ERROR, P256_PRIME};
pub use self::types::{Certificate, PublicKeyData};
pub use self::verify::check_signature_from;
use self::verify::parse_ecdsa_signature;
const NULL_BYTES: &[u8] = &[0x05, 0x00];
mod algorithms;
mod certificate;
mod der;
mod extensions;
mod name_constraints;
mod names;
mod public_key;
mod time;
mod types;
mod verify;

#[cfg(test)]
mod tests;
