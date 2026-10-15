use super::super::*;
use crate::sha256::{sha256_hex, Sha256};
use sha2::Digest;

#[test]
fn cli_accepts_boot_and_exact_ensure_forms_only() {
    use super::super::context::{parse as parse_for_test, Invocation};
    assert!(matches!(
        parse_for_test(&["soda-image-import".to_string()]),
        Ok(Invocation::Boot)
    ));
    let ensure = |payload: &str, set: &str| {
        parse_for_test(&[
            "soda-image-import".to_string(),
            "ensure".to_string(),
            "--payload".to_string(),
            payload.to_string(),
            "--set".to_string(),
            set.to_string(),
        ])
    };
    assert!(matches!(
        ensure("/usr/share/soda/release.json", "required"),
        Ok(Invocation::Ensure { .. })
    ));
    assert!(matches!(
        ensure("/tmp/release.json", "tailnet"),
        Ok(Invocation::Ensure { .. })
    ));
    assert!(ensure("relative/release.json", "required").is_err());
    assert!(ensure("/tmp/release.json", "all").is_err());
    assert!(parse_for_test(&["soda-image-import".to_string(), "extra".to_string()]).is_err());
    assert!(parse_for_test(&["soda-image-import".to_string(), "ensure".to_string()]).is_err());
    assert!(parse_for_test(&[]).is_err());
}

#[test]
fn sha256_matches_fips_vectors_streamed_and_oneshot() {
    let vectors: &[(&[u8], &str)] = &[
        (
            b"",
            "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
        ),
        (
            b"abc",
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
        ),
        (
            b"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq",
            "248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1",
        ),
    ];
    for (input, want) in vectors {
        assert_eq!(&sha256_hex(input), want);
        // Feed byte-by-byte plus odd chunks to exercise block splits.
        let mut h = Sha256::new();
        for byte in input.iter() {
            h.update(std::slice::from_ref(byte));
        }
        assert_eq!(&format!("{:x}", h.finalize()), want);
    }
    let million = vec![b'a'; 1_000_000];
    let mut h = Sha256::new();
    for chunk in million.chunks(333) {
        h.update(chunk);
    }
    assert_eq!(
        format!("{:x}", h.finalize()),
        "cdc76e5c9914fb9281a1c7e284d73e67f1809a48a497200e046d39ccc7112cd0"
    );
}

#[test]
fn native_platform_matches_go_checks() {
    assert_eq!(oci_architecture("x86_64").unwrap(), "amd64");
    assert_eq!(oci_architecture("aarch64").unwrap_err(), "expected x86_64");
    assert_eq!(oci_architecture("").unwrap_err(), "expected x86_64");
    assert!(require_native("x86_64").is_ok());
    assert_eq!(require_native("armv7").unwrap_err(), "expected x86_64");
}
