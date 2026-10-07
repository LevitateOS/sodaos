//! Strict X.509 admission for the installer's local CA fingerprint guidance.
//!
//! Certificate structure, DER framing, typed extensions, and SPKI fields are
//! decoded by `x509-cert`. The adapter verifies the original TBS bytes and
//! preserves the raw DER fingerprint input. It does not perform chain or
//! validity-window verification.

mod algorithms;
mod certificate;
mod types;
mod verify;

#[cfg(test)]
mod tests;

pub(super) use certificate::parse_certificate;
pub(super) use verify::verify_self_signature;
