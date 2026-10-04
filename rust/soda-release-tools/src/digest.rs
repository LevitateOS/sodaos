//! Revision/digest predicates and file hashing (Go `internal/release/build`
//! `files.go` subset used by the release CLIs).

use sha2::{Digest, Sha256};
use std::fs::File;
use std::io::Read;
use std::path::Path;

pub fn is_revision(s: &str) -> bool {
    s.len() == 40 && s.bytes().all(|b| b.is_ascii_hexdigit())
}

pub fn is_digest(s: &str) -> bool {
    s.len() == 64 && s.bytes().all(|b| b.is_ascii_hexdigit())
}

pub fn is_signer(s: &str) -> bool {
    (s.len() == 40 || s.len() == 64) && s.bytes().all(|b| b.is_ascii_hexdigit())
}

/// anywhere
pub fn oci_architecture(arch: &str) -> Result<&'static str, String> {
    if arch == "x86_64" {
        return Ok("amd64");
    }
    Err("expected x86_64".to_owned())
}

pub fn require_native(arch: &str) -> Result<(), String> {
    let native = oci_architecture(arch)?;
    if cfg!(target_os = "linux") && native == std::env::consts::ARCH {
        return Ok(());
    }
    Err("matching-native Linux required".to_owned())
}

/// Hash a regular non-symlink file, refusing anything else like the Go owner.
pub fn hash_file(path: &str) -> Result<String, String> {
    let st = std::fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    if !st.file_type().is_file() || st.file_type().is_symlink() {
        return Err("regular non-symlink file required".to_owned());
    }
    // symlink_metadata on a symlink reports the link itself, so is_file()
    // already excludes symlinks; the extra check above is belt and braces.
    let mut file = File::open(path).map_err(|e| e.to_string())?;
    let mut hasher = Sha256::new();
    let mut buf = [0u8; 8192];
    loop {
        let n = file.read(&mut buf).map_err(|e| e.to_string())?;
        if n == 0 {
            break;
        }
        hasher.update(&buf[..n]);
    }
    Ok(format!("{:x}", hasher.finalize()))
}

/// Resolve symlinks lexically-free like `filepath.EvalSymlinks`: every
/// existing component must resolve and the result is the canonical path.
pub fn eval_symlinks(path: &Path) -> std::io::Result<std::path::PathBuf> {
    std::fs::canonicalize(path)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn revision_and_digest_shapes() {
        assert!(is_revision(&"a".repeat(40)));
        assert!(!is_revision(&"a".repeat(39)));
        assert!(!is_revision(&"g".repeat(40)));
        assert!(is_digest(&"b".repeat(64)));
        assert!(!is_digest(&"b".repeat(63)));
        assert!(is_signer(&"A".repeat(40)));
        assert!(is_signer(&"f".repeat(64)));
        assert!(!is_signer("deadbeef"));
    }

    #[test]
    fn native_arch_admission() {
        assert_eq!(oci_architecture("x86_64").unwrap(), "amd64");
        assert_eq!(
            oci_architecture("aarch64").unwrap_err(),
            "expected x86_64"
        );
        if cfg!(target_arch = "x86_64") && cfg!(target_os = "linux") {
            assert!(require_native("x86_64").is_ok());
        }
        assert!(require_native("aarch64").is_err());
    }
}
