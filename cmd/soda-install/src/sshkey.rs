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
mod tests {
    use super::*;

    const ED25519: &str =
        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKQ0MsA1tWa7risZNVfq58qNB9BByJfSJUQpWI9KvglV";
    const RSA: &str = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCoDZAt7ewKwwXBa7tCvC9/+p//wMupqSjGVnTOvoYOUt1UNbromMK5hBZGq2xIlqJQ0rRZoTEtsE7w5BUgkNzvsioTnnjD48UeqMvJhBfSrJYTVMZeM/ttm1jmlNhbjs6nt98R/KmdsyK+2a+z5BQ+KRlr4kbKfd/UDOPj8XuA2/vW4K2301UUdDk9Jh2r/bcjRnrIyHUX1Rmga608tAWRZtJQRo+8/28JqnjQM5s4qcu1d1N2Y823P4YGaLYhRLoKhV1/gCMRRhD9ZTZpn58sVmiEGQ90YeHE/8vETm+Q2IkjZvx2vobzhvdsA3LGs3B1EN2kdbCGJRqaltrJLK+t";
    const ECDSA256: &str = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBLtQL7Kq7o6tng5YEyRN3ICkkd2BErzxr+p3zkns1Apc0BJ7BDcTZqzWuusUsWLZxRtnOVtz2FT2vd0GlCz20RI=";
    const ECDSA384: &str = "ecdsa-sha2-nistp384 AAAAE2VjZHNhLXNoYTItbmlzdHAzODQAAAAIbmlzdHAzODQAAABhBNuz0aERM3G5ldqem61DeWNH6TPAuvKqwTrF2yCFg4tphB+K2icoJtn+oELywJfa5vRZmifM9zYAviqIcFjRIWZw2eeZQBdu8563omqOcS8x0JMVFFRTuPhpASQwnaK2OA==";
    const ECDSA521: &str = "ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBAB6wXlDnMlkFpqHCbvCbzARmcMPfFnBWBe9lNPrdal/1+t6X2tGURdQ2CGo7D15m/c5gN9fLlECNwYumO8Zo/9ctQFkH+iiqkuOJvaivvK56hHpb8ytWB9FPouFtXfzjfeuwqkpix6MR54fQbFSXgHEfqZPmf11b3+n1yJzi7nyV8EYMQ==";

    fn sk_ed25519_wire(extra: &[u8]) -> String {
        let mut wire = marshal_string(b"sk-ssh-ed25519@openssh.com");
        wire.extend(marshal_string(&[0x42; 32]));
        wire.extend(marshal_string(b"ssh:"));
        wire.extend_from_slice(extra);
        format!("sk-ssh-ed25519@openssh.com {}", b64_encode(&wire))
    }

    #[test]
    fn real_keys_parse_and_normalize() {
        for key in [ED25519, RSA, ECDSA256, ECDSA384, ECDSA521] {
            assert_eq!(public_key(key).unwrap(), key, "round trip");
            assert_eq!(public_key(&format!("{key} test-comment")).unwrap(), key);
            assert_eq!(public_key(&format!("  {key}  ")).unwrap(), key);
            assert_eq!(public_key(&format!("{key}\tcomment")).unwrap(), key);
        }
        let key = parse_authorized_key(ED25519).unwrap().key;
        assert_eq!(
            fingerprint_sha256_wire(&key.marshal()),
            "SHA256:8zz4BxDGZ75OGjyLd0V7cLMngT+KWeelWxFYt6+fKtc"
        );
        // Raw tool output with a trailing newline parses like Go.
        let key = parse_authorized_key_bytes(format!("{ED25519}\n").as_bytes())
            .unwrap()
            .key;
        assert_eq!(key.key_type(), "ssh-ed25519");
    }

    #[test]
    fn rejects_match_go_taxonomy() {
        // "one SSH public key required": size/CTL pre-checks.
        assert_eq!(
            public_key(&"x".repeat(16385)).unwrap_err(),
            "one SSH public key required"
        );
        assert_eq!(
            public_key(&format!("{ED25519}\n")).unwrap_err(),
            "one SSH public key required"
        );
        assert_eq!(
            public_key(&format!("{ED25519}\r")).unwrap_err(),
            "one SSH public key required"
        );
        assert_eq!(
            public_key(&format!("{ED25519}\0")).unwrap_err(),
            "one SSH public key required"
        );
        // "valid SSH public key ...": unparsable, options, or key material.
        let opt_cases = [
            format!("command=\"x\" {ED25519}"),
            format!("no-pty {ED25519}"),
            format!("restrict {ED25519}"),
            format!("\"\" {ED25519}"),
            ",,,ssh-ed25519 AAAA".to_string(),
        ];
        for bad in [
            "AAA",
            "ssh-ed25519",
            "",
            "   ",
            "# just a comment",
            "ssh-ed25519 AAAA",
            "ssh-ed25519 !!!",
            "ssh-rsa AAAA",
            "ssh-ed25519-cert-v01@openssh.com AAAA",
        ] {
            assert_eq!(
                public_key(bad).unwrap_err(),
                "valid SSH public key without authorized_keys options required",
                "input {bad:?}"
            );
        }
        for bad in &opt_cases {
            assert_eq!(
                public_key(bad).unwrap_err(),
                "valid SSH public key without authorized_keys options required",
                "input {bad:?}"
            );
        }
        // Empty options (commas only) parse with no options, as in Go.
        assert_eq!(public_key(&format!(",,, {ED25519}")).unwrap(), ED25519);
        assert_eq!(public_key(&format!(" , {ED25519}")).unwrap(), ED25519);
        // Declared/embedded type mismatch falls through to options parsing.
        let swapped = format!("ssh-rsa {}", ED25519.split(' ').nth(1).unwrap());
        assert_eq!(
            public_key(&swapped).unwrap_err(),
            "valid SSH public key without authorized_keys options required"
        );
    }

    #[test]
    fn unsupported_types_reported() {
        // Valid DSA wire (P 1024-bit, Q 160-bit) parses but is unsupported.
        let p = vec![0x81u8]
            .into_iter()
            .chain(vec![0u8; 127])
            .collect::<Vec<u8>>();
        let mut wire = marshal_string(b"ssh-dss");
        wire.extend(marshal_mpint(false, &p));
        wire.extend(marshal_mpint(false, &[0x81; 20]));
        wire.extend(marshal_mpint(false, &[0x02]));
        wire.extend(marshal_mpint(false, &[0x03]));
        let dss = format!("ssh-dss {}", b64_encode(&wire));
        assert_eq!(
            public_key(&dss).unwrap_err(),
            "unsupported operator SSH key type"
        );
    }

    #[test]
    fn rsa_canonicalizes_padded_mpints() {
        // Oracle: RSA-PADDED accepted with marshal_clean=true.
        let wire = b64_decode_go(RSA.split(' ').nth(1).unwrap().as_bytes()).unwrap();
        let (_, rest) = parse_string(&wire).unwrap();
        let (_, e_mag, rest) = parse_mpint(rest).unwrap();
        let (_, n_mag, _) = parse_mpint(rest).unwrap();
        let mut padded = marshal_string(b"ssh-rsa");
        padded.extend(marshal_string(&[vec![0, 0, 0], e_mag].concat()));
        padded.extend(marshal_string(&[vec![0, 0], n_mag].concat()));
        let line = format!("ssh-rsa {}", b64_encode(&padded));
        assert_eq!(public_key(&line).unwrap(), RSA);
        // Even exponent and oversized exponent are refused.
        let mut bad = marshal_string(b"ssh-rsa");
        bad.extend(marshal_string(&[4]));
        bad.extend(marshal_string(&[9u8; 64]));
        assert!(public_key(&format!("ssh-rsa {}", b64_encode(&bad))).is_err());
    }

    #[test]
    fn sk_keys_match_go_truncation_rule() {
        // x/crypto rejects trailing flags/handle bytes ("trailing junk");
        // only the truncated key+application form parses.
        assert!(public_key(&sk_ed25519_wire(&[])).is_ok());
        assert!(public_key(&sk_ed25519_wire(&[0x01])).is_err());
        let mut full = vec![0x01u8];
        full.extend(marshal_string(b"handle"));
        full.extend(marshal_string(b""));
        assert!(public_key(&sk_ed25519_wire(&full)).is_err());
    }

    #[test]
    fn base64_matches_go_decode() {
        assert_eq!(b64_decode_go(b"").unwrap(), b"");
        assert_eq!(b64_decode_go(b"QUI=").unwrap(), b"AB");
        assert_eq!(b64_decode_go(b"QUI").unwrap(), b"AB"); // lenient tail
        assert_eq!(b64_decode_go(b"QUJD").unwrap(), b"ABC");
        assert!(b64_decode_go(b"Q").is_err());
        assert!(b64_decode_go(b"AB=C").is_err());
        assert!(b64_decode_go(b"A===").is_err());
        assert!(b64_decode_go(b"AB==CD").is_err());
        assert_eq!(b64_encode(b"AB"), "QUI=");
        assert_eq!(b64_encode_raw(b"AB"), "QUI");
    }

    #[test]
    fn mpint_round_trip() {
        for (neg, mag) in [
            (false, vec![]),
            (false, vec![1]),
            (false, vec![0x80]),
            (true, vec![1]),
            (true, vec![0x80]),
        ] {
            let encoded = marshal_mpint(neg, &mag);
            let (n, m, rest) = parse_mpint(&encoded).unwrap();
            assert!(rest.is_empty());
            assert_eq!((n, m.clone()), (neg, mag));
            assert_eq!(marshal_mpint(n, &m), encoded);
        }
    }
}
