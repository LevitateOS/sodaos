use super::super::*;
use super::fixtures::*;
use crate::sha256::{hex_lower, sha256_hex, Sha256};
use sha2::Digest;

#[test]
fn admission_requires_root_and_no_arguments() {
    assert!(admit(0, 1).is_ok());
    assert_eq!(admit(1, 1).unwrap_err(), "root and no arguments required");
    assert_eq!(admit(0, 2).unwrap_err(), "root and no arguments required");
    assert_eq!(admit(0, 0).unwrap_err(), "root and no arguments required");
}

#[test]
fn identifier_shapes_match_go_regexps() {
    assert!(is_digest(&repeat('a', 64)));
    assert!(is_digest(&repeat('9', 64)));
    assert!(!is_digest(&repeat('a', 63)));
    assert!(!is_digest(&repeat('a', 65)));
    assert!(!is_digest(&repeat('A', 64)));
    assert!(!is_digest(&repeat('g', 64)));
    assert!(is_revision(&repeat('f', 40)));
    assert!(!is_revision(&repeat('f', 39)));
    assert!(!is_revision(&repeat('F', 40)));
    assert!(is_prefixed_digest(&format!("sha256:{}", repeat('0', 64))));
    assert!(!is_prefixed_digest(&repeat('0', 64)));
    assert!(!is_prefixed_digest(&format!("sha256:{}", repeat('0', 63))));
    assert!(is_coreos_version("44.20260817.3.2"));
    assert!(!is_coreos_version("44.20260817.3"));
    assert!(!is_coreos_version("44.20260817.3.2.1"));
    assert!(!is_coreos_version("44.20260817.3.x"));
    assert!(!is_coreos_version(""));
    assert!(valid_repository_prefix("ghcr.io/example/sodaos"));
    assert!(valid_repository_prefix("ghcr.io/a-b/c_d.e-f"));
    assert!(!valid_repository_prefix(
        "ghcr.io/example/soda\nImage=untrusted"
    ));
    assert!(!valid_repository_prefix("quay.io/example/sodaos"));
    assert!(!valid_repository_prefix("ghcr.io/example"));
    assert!(!valid_repository_prefix("ghcr.io/example/a/b"));
    assert!(!valid_repository_prefix("ghcr.io/-bad/repo"));
    assert!(!valid_repository_prefix(&format!(
        "ghcr.io/example/{}",
        repeat('a', 200)
    )));
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
        assert_eq!(&hex_lower(&h.finalize()), want);
    }
    let million = vec![b'a'; 1_000_000];
    let mut h = Sha256::new();
    for chunk in million.chunks(333) {
        h.update(chunk);
    }
    assert_eq!(
        hex_lower(&h.finalize()),
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

#[test]
fn json_validity_matches_single_value_rule() {
    assert!(json_valid(br#"{"a":1}"#));
    assert!(json_valid(b"[1,2]"));
    assert!(json_valid(b"  null  "));
    assert!(!json_valid(b""));
    assert!(!json_valid(br#"{"a":1} {"b":2}"#));
    assert!(!json_valid(br#"{"a":}"#));
    assert!(!json_valid(b"\xff\xfe"));
}
