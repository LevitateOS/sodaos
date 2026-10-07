use super::wire::parse_public_key;
use super::*;

fn put_string(out: &mut Vec<u8>, value: &[u8]) {
    out.extend_from_slice(&(value.len() as u32).to_be_bytes());
    out.extend_from_slice(value);
}

fn marshal_string(value: &[u8]) -> Vec<u8> {
    let mut out = Vec::new();
    put_string(&mut out, value);
    out
}

fn mpint(value: &[u8]) -> Vec<u8> {
    let mut bytes = Vec::with_capacity(value.len() + 1);
    if value.first().is_some_and(|byte| byte & 0x80 != 0) {
        bytes.push(0);
    }
    bytes.extend_from_slice(value);
    marshal_string(&bytes)
}

fn rsa_line(exponent: &[u8], modulus: &[u8]) -> String {
    let mut wire = marshal_string(b"ssh-rsa");
    wire.extend(mpint(exponent));
    wire.extend(mpint(modulus));
    format!("ssh-rsa {}", b64_encode(&wire))
}

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

fn sk_ecdsa_wire(curve: &[u8], point: &[u8]) -> String {
    let mut wire = marshal_string(b"sk-ecdsa-sha2-nistp256@openssh.com");
    wire.extend(marshal_string(curve));
    wire.extend(marshal_string(point));
    wire.extend(marshal_string(b"ssh:"));
    format!("sk-ecdsa-sha2-nistp256@openssh.com {}", b64_encode(&wire))
}

fn ordinary_p256_point() -> Vec<u8> {
    let wire = b64_decode_go(ECDSA256.split(' ').nth(1).unwrap().as_bytes()).unwrap();
    let key = ssh_key::PublicKey::from_bytes(&wire).unwrap();
    match key.key_data() {
        ssh_key::public::KeyData::Ecdsa(key) => key.as_sec1_bytes().to_vec(),
        _ => panic!("ECDSA fixture decoded to another key family"),
    }
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
fn rsa_exponent_and_modulus_bounds_remain_installer_policy() {
    let modulus = vec![0x80; 256];
    assert!(public_key(&rsa_line(&[3], &modulus)).is_ok());
    assert!(public_key(&rsa_line(&[4], &modulus)).is_err());
    assert!(public_key(&rsa_line(&[1, 0, 0, 1], &modulus)).is_err());
    assert!(public_key(&rsa_line(&[3], &vec![0x80; 2049])).is_err());
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
    wire.extend(mpint(&p));
    wire.extend(mpint(&[0x81; 20]));
    wire.extend(mpint(&[0x02]));
    wire.extend(mpint(&[0x03]));
    let dss = format!("ssh-dss {}", b64_encode(&wire));
    assert_eq!(
        public_key(&dss).unwrap_err(),
        "unsupported operator SSH key type"
    );
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
fn sk_ecdsa_keeps_uncompressed_valid_point_policy() {
    let point = ordinary_p256_point();
    let expected = sk_ecdsa_wire(b"nistp256", &point);
    assert_eq!(public_key(&expected).unwrap(), expected);

    let mut compressed = vec![2; 33];
    compressed[0] = 2;
    assert!(public_key(&sk_ecdsa_wire(b"nistp256", &compressed)).is_err());
    assert!(public_key(&sk_ecdsa_wire(b"nistp384", &point)).is_err());

    let mut off_curve = vec![0; 65];
    off_curve[0] = 4;
    assert!(public_key(&sk_ecdsa_wire(b"nistp256", &off_curve)).is_err());
}

#[test]
fn base64_matches_go_decode() {
    assert_eq!(b64_decode_go(b"").unwrap(), b"");
    assert_eq!(b64_decode_go(b"QUI=").unwrap(), b"AB");
    assert_eq!(b64_decode_go(b"YR==").unwrap(), b"a"); // unused trailing bits ignored
    assert_eq!(b64_decode_go(b"Zg==\r\n").unwrap(), b"f");
    assert_eq!(b64_decode_go(b"QUJD").unwrap(), b"ABC");
    assert!(b64_decode_go(b"QUI").is_err()); // standard padding is required
    assert!(b64_decode_go(b"Q").is_err());
    assert!(b64_decode_go(b"AB=C").is_err());
    assert!(b64_decode_go(b"A===").is_err());
    assert!(b64_decode_go(b"AB==CD").is_err());
    assert!(b64_decode_go(b"Zg== ").is_err());
    assert_eq!(b64_encode(b"AB"), "QUI=");
    assert_eq!(b64_encode_raw(b"AB"), "QUI");
}

#[test]
fn malformed_mpint_and_trailing_wire_bytes_are_rejected() {
    let mut truncated = marshal_string(b"ssh-rsa");
    truncated.extend_from_slice(&[0, 0, 0, 4, 1]);
    assert!(parse_public_key(&truncated).is_err());
    let mut trailing = b64_decode_go(RSA.split(' ').nth(1).unwrap().as_bytes()).unwrap();
    trailing.push(0);
    assert!(parse_public_key(&trailing).is_err());
}
