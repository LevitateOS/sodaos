//! Fixture trust and inline data, mirroring `trust.go` with OpenSSH
//! delegation.
//!
//! The Go owner parses SSH keys in-process (`x/crypto/ssh`). Hand-rolling
//! SSH key formats and `known_hosts` matching (including hashed entries)
//! would risk subtle verification gaps, so this port delegates those exact
//! operations to the host's OpenSSH: `ssh-keygen -y` parses the embedded
//! private key, and `ssh-keygen -F` matches the pinned entry. Gzip stays
//! in-process via `flate2` so the byte bounds and error taxonomy match
//! exactly.

mod host_key;
mod inline_data;

pub use host_key::{parse_ignition, verify_fixture_trust, IgnitionError};
#[cfg(test)]
pub use inline_data::encode_base64;
pub use inline_data::{
    decode_base64, decode_data_uri, decode_ignition_file_refs, decode_ignition_files,
    gunzip_bounded, inline_data, inline_data_limited, IgnitionFile, IgnitionFileRef,
};
pub(crate) use inline_data::INLINE_GZIP_LIMIT;

#[cfg(test)]
mod tests;
