use super::*;

fn put_string(out: &mut Vec<u8>, value: &[u8]) {
    out.extend_from_slice(&(value.len() as u32).to_be_bytes());
    out.extend_from_slice(value);
}

fn read_string(input: &[u8]) -> (&[u8], &[u8]) {
    let len = u32::from_be_bytes(input[..4].try_into().unwrap()) as usize;
    (&input[4..4 + len], &input[4 + len..])
}

fn push_u32(out: &mut Vec<u8>, value: u32) {
    out.extend_from_slice(&value.to_be_bytes());
}
fn push_u64(out: &mut Vec<u8>, value: u64) {
    out.extend_from_slice(&value.to_be_bytes());
}

fn certificate_blob(cert_type: &str, fields: &[Vec<u8>]) -> Vec<u8> {
    let mut out = Vec::new();
    put_string(&mut out, cert_type.as_bytes());
    put_string(&mut out, b"nonce");
    for field in fields {
        out.extend_from_slice(field);
    }
    push_u64(&mut out, 0);
    push_u32(&mut out, 1);
    put_string(&mut out, b"expired-test-certificate");
    put_string(&mut out, b"");
    // Expired but representable: parsing must not decide certificate trust.
    push_u64(&mut out, 0);
    push_u64(&mut out, 1);
    put_string(&mut out, b"");
    put_string(&mut out, b"");
    put_string(&mut out, b"");
    let mut ca = Vec::new();
    put_string(&mut ca, b"ssh-ed25519");
    put_string(&mut ca, &[0x37; 32]);
    put_string(&mut out, &ca);
    let mut sig = Vec::new();
    put_string(&mut sig, b"ssh-ed25519");
    put_string(&mut sig, &[0; 64]);
    put_string(&mut out, &sig);
    out
}

fn string_field(value: &[u8]) -> Vec<u8> {
    let mut out = Vec::new();
    put_string(&mut out, value);
    out
}

fn mpint_field(value: &[u8]) -> Vec<u8> {
    let mut encoded = Vec::new();
    if value.first().is_some_and(|byte| byte & 0x80 != 0) {
        encoded.push(0);
    }
    encoded.extend_from_slice(value);
    string_field(&encoded)
}

fn rsa_blob(exponent: &[u8], modulus: &[u8]) -> Vec<u8> {
    let mut out = Vec::new();
    put_string(&mut out, b"ssh-rsa");
    out.extend(
        [mpint_field(exponent), mpint_field(modulus)]
            .into_iter()
            .flatten(),
    );
    out
}

fn dsa_blob(p: &[u8], q: &[u8], g: &[u8], y: &[u8]) -> Vec<u8> {
    let mut out = Vec::new();
    put_string(&mut out, b"ssh-dss");
    out.extend(
        [
            mpint_field(p),
            mpint_field(q),
            mpint_field(g),
            mpint_field(y),
        ]
        .into_iter()
        .flatten(),
    );
    out
}

fn ecdsa_fields(curve: &[u8], point: &[u8]) -> Vec<Vec<u8>> {
    vec![string_field(curve), string_field(point)]
}

fn ordinary_ecdsa_point(line: &str, algorithm: &str, curve: &[u8]) -> Vec<u8> {
    let raw = b64_decode(line.split_once(' ').unwrap().1).unwrap();
    let (actual_algorithm, fields) = read_string(&raw);
    assert_eq!(actual_algorithm, algorithm.as_bytes());
    let (actual_curve, fields) = read_string(fields);
    assert_eq!(actual_curve, curve);
    let (point, rest) = read_string(fields);
    assert!(rest.is_empty());
    point.to_vec()
}

fn certificate_families() -> Vec<Vec<u8>> {
    let p256 = ordinary_p256_point();
    let p384 = ordinary_ecdsa_point(ECDSA384, ALGO_ECDSA384, b"nistp384");
    let p521 = ordinary_ecdsa_point(ECDSA521, ALGO_ECDSA521, b"nistp521");
    let rsa = vec![mpint_field(&[3]), mpint_field(&[0x80, 1])];
    let dsa = vec![
        mpint_field(&[0x80; 128]),
        mpint_field(&[0x80; 20]),
        mpint_field(&[2]),
        mpint_field(&[3]),
    ];
    let cases = vec![
        ("ssh-rsa-cert-v01@openssh.com", rsa),
        ("ssh-dss-cert-v01@openssh.com", dsa),
        (
            "ecdsa-sha2-nistp256-cert-v01@openssh.com",
            ecdsa_fields(b"nistp256", &p256),
        ),
        (
            "ecdsa-sha2-nistp384-cert-v01@openssh.com",
            ecdsa_fields(b"nistp384", &p384),
        ),
        (
            "ecdsa-sha2-nistp521-cert-v01@openssh.com",
            ecdsa_fields(b"nistp521", &p521),
        ),
        (
            "ssh-ed25519-cert-v01@openssh.com",
            vec![string_field(&[0x41; 32])],
        ),
        (
            "sk-ecdsa-sha2-nistp256-cert-v01@openssh.com",
            vec![
                string_field(b"nistp256"),
                string_field(&p256),
                string_field(b"ssh:"),
            ],
        ),
        (
            "sk-ssh-ed25519-cert-v01@openssh.com",
            vec![string_field(&[0x42; 32]), string_field(b"ssh:")],
        ),
    ];
    cases
        .into_iter()
        .map(|(cert, fields)| certificate_blob(cert, &fields))
        .collect()
}

// Public-key fixtures only (comments stripped). RSA/ECDSA/Ed25519 from
// ssh-keygen, DSA and the OpenSSH certificate from Go's x/crypto.
const RSA: &str = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7sz+R+V5I0foZSFl3AOjfvN9eOpNqb/n1QFuiO6XzEo4kKnOs5/nIp/XOFC0/SzBl4S0xhmBw+bbNgpPQ+rreuLrsA01wqEoNmtE7VsEXNysYV58cVaGaGI4E/hAjKuyRzdQkI99uGiHXnEN5iwojR493ZJ0wvN8mWhsjU2U+ctjQneR4sphYRUgDet3mPqHMYucH7eZgfPVxHNUTxTTUXinZPxhRODHz0QRCezBoMDY1nwpFy/gC8hXusrKomlWkGg3mEZTMTbmMsWcpz/xVT48S3M+z9LYptTXTS6bPhiUh4xHsQWOo6Iy1LIiSMCPcBkhgCebx23OGjpFbjuNJ";
const ECDSA256: &str = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHj7ciCLbVeoJXxkjiotLyfyRA8p9CyO0TMNFOqLt/TPEElUQ2eJTL9+Wi9hhFVDjaWWCzm/dCcAyU94xFHSe2c=";
const ECDSA384: &str = "ecdsa-sha2-nistp384 AAAAE2VjZHNhLXNoYTItbmlzdHAzODQAAAAIbmlzdHAzODQAAABhBGhjWK/kZlOIFnt++QFazBBAxQ+/qsF0ZiCqRuMVeVPjPsdw1ZI6idf5gBpPNBRiscAM201TmUdse8NTzPh4cMSN0tHx+8CIKlACp+4CyEnITV81ZiRozztvIXlABAe9Aw==";
const ECDSA521: &str = "ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBAC7PT6SUkkAyi1W3YCqPFMDBYOHD3W7s5e78EObzfBPnJU1TmNvk1oIYttpaPEv8+v7VmgE0KWkLOYAVro5gH3tMQA/oQYdzKuD1i6RvifNpRIt2lfRV2NKZ2IvuVV/O4TdRoW4IoJ/Z6I0xA8TcbACkei5u2g2xu3fZWVHq9hOCaFmJw==";
const ED: &str = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre";
const DSA: &str = "ssh-dss AAAAB3NzaC1kc3MAAACBAIz7Di7wJQy0oy8m6z/XMjyzC4nIWYnEbwJfa+7BTVCYF1Ft5zzz6Zz9rjdmRcgSbLy5ImfS5I4dDFO7RaGu8vWTUX4Bt+WuZPPXuP7gfoWKJS+QhYabM/AbFpqF/Y+RqKf78fmlJf93vx8N4KJHtXpKOscR8+f5o2hjOZ3qdA5NAAAAFQDQwFThs2jiwKMm5WGMWT/yhMLDxwAAAIBvtk8aJSrq2eFwwpTRpvj/J4goZiirxksn87GpqCT+LVrcGTWNClQaBRA+K7Vr2L7QvB7YxeBWAo71Rq2xTPwBa9UXlzVOCaZeg5DXwtC0pAVcHSNlsVa7id1W2qV4fjCaF6clNUUjTsOxqwKRhYvHWAXPS0H9Xopj94672Oq4cAAAAIBqBT9SM/sIyejm2HcefUtARGMKOg6qkvkwNV4XMSADgdu/0vNeUxPXChtm/wmhZ6HygBOq+aV/o/athTEpDdteskAxdgPUgpDwKiScG8XhraCOCGnzNFAgeMAS4mKZilKJ0ewhMu3yCF1taiQqqG2iIosyaszZD8Zk/BhXN7IOeQ==";
const CERT: &str = "ssh-ed25519-cert-v01@openssh.com AAAAIHNzaC1lZDI1NTE5LWNlcnQtdjAxQG9wZW5zc2guY29tAAAAIPVDs6sltHdos8K2q8bZnSJvBVHhzMSZwnC4EUO5JMbHAAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcreAAAAAAAAAAAAAAABAAAABnRlc3RpZAAAAAkAAAAFYWxpY2UAAAAAAAAAAP//////////AAAAAAAAAIIAAAAVcGVybWl0LVgxMS1mb3J3YXJkaW5nAAAAAAAAABdwZXJtaXQtYWdlbnQtZm9yd2FyZGluZwAAAAAAAAAWcGVybWl0LXBvcnQtZm9yd2FyZGluZwAAAAAAAAAKcGVybWl0LXB0eQAAAAAAAAAOcGVybWl0LXVzZXItcmMAAAAAAAAAAAAAARcAAAAHc3NoLXJzYQAAAAMBAAEAAAEBALuzP5H5XkjR+hlIWXcA6N+83146k2pv+fVAW6I7pfMSjiQqc6zn+cin9c4ULT9LMGXhLTGGYHD5ts2Ck9D6ut64uuwDTXCoSg2a0TtWwRc3KxhXnxxVoZoYjgT+ECMq7JHN1CQj324aIdecQ3mLCiNHj3dknTC83yZaGyNTZT5y2NCd5HiymFhFSAN63eY+ocxi5wft5mB89XEc1RPFNNReKdk/GFE4MfPRBEJ7MGgwNjWfCkXL+ALyFe6ysqiaVaQaDeYRlMxNuYyxZynP/FVPjxLcz7P0tim1NdNLps+GJSHjEexBY6jojLUsiJIwI9wGSGAJ5vHbc4aOkVuO40kAAAEUAAAADHJzYS1zaGEyLTUxMgAAAQBrguk2IhdEGpwyqtiWaL87YzDzY7lFiKLeXDNF3WzcYlgx8zLF3Ujy33WH8MSvzIgNJTY/QJseouCU9iYrTrTlUjLht5DWeKf622TaOwUw7B3R4/ugjdDk49wrxjoO86Wvb8WosM8NYuNTddom2D/CVuOXB39LxUIP5kzaK3Z6p59xlGyamdHXaxs3M5yzU67imcs0G3pnf3G2N+KhcPOIzThd8nyyp2hPUgJqwfNxjxboHSr6JbOXxa0Iud5gRilfsoREaA4Fh4Qf/2OvqvhWLqhjPISqozAR3CZJ1v9RQaqr5ROaGLrQFudRFacxi3HgVa9GjCP2ZEwC4Q4kT0d3";

fn canonical(line: &str) -> (String, bool) {
    let (key, opts) = parse_authorized_key(line.as_bytes()).unwrap();
    (marshal_authorized_key(&key.key_type, &key.blob), opts)
}

fn ordinary_p256_point() -> Vec<u8> {
    ordinary_ecdsa_point(ECDSA256, ALGO_ECDSA256, b"nistp256")
}

#[test]
fn all_types_round_trip_canonically() {
    for line in [RSA, ECDSA256, ECDSA384, ECDSA521, ED, DSA, CERT] {
        let (back, opts) = canonical(line);
        assert_eq!(back, format!("{line}\n"));
        assert!(!opts);
    }
}

#[test]
fn real_forever_certificate_preserves_raw_validity_and_signed_wire() {
    let encoded = CERT.split_once(' ').unwrap().1;
    let wire = b64_decode(encoded).unwrap();
    let cert = Certificate::from_bytes(&wire).expect("forever sentinel is valid wire data");
    assert_eq!(cert.valid_before(), u64::MAX);
    assert_eq!(cert.to_bytes().unwrap(), wire);
    let parsed = parse_public_key(&wire).unwrap();
    assert_eq!(parsed.blob, wire);
}

#[test]
fn redundant_leading_zero_mpint_is_rejected() {
    let modulus = [0x80, 0x01];
    let mut noncanonical = Vec::new();
    put_string(&mut noncanonical, b"ssh-rsa");
    put_string(&mut noncanonical, &[0x00, 0x03]);
    noncanonical.extend(mpint_field(&modulus));

    assert!(
        parse_public_key(&noncanonical).is_err(),
        "the pinned decoder rejects a redundant MPINT sign byte"
    );
}

#[test]
fn all_certificate_families_parse_without_trust_or_validity_checks() {
    let families = certificate_families();
    assert_eq!(families.len(), 8);
    let names = [
        "RSA",
        "DSA",
        "ECDSA P-256",
        "ECDSA P-384",
        "ECDSA P-521",
        "Ed25519",
        "SK-ECDSA P-256",
        "SK-Ed25519",
    ];
    for (name, cert) in names.into_iter().zip(families) {
        let parsed = parse_public_key(&cert).unwrap_or_else(|error| panic!("{name}: {error}"));
        assert!(parsed.key_type.ends_with("-cert-v01@openssh.com"));
        assert_eq!(parse_public_key(&parsed.blob).unwrap(), parsed, "{name}");
    }
}

#[test]
fn rsa_sha2_certificate_alias_is_refused_when_library_normalizes_it() {
    let mut families = certificate_families();
    let legacy = families.remove(0);
    let (_, body) = read_string(&legacy);
    let mut alias = Vec::new();
    put_string(&mut alias, b"rsa-sha2-256-cert-v01@openssh.com");
    alias.extend_from_slice(body);

    let parsed = Certificate::from_bytes(&alias).expect("library recognizes RSA SHA-2 alias");
    assert_eq!(parsed.to_bytes().unwrap(), legacy);
    assert_ne!(
        parsed.algorithm().to_certificate_type(),
        "rsa-sha2-256-cert-v01@openssh.com"
    );
    assert!(parse_public_key(&alias).is_err());
}

#[test]
fn sk_types_parse() {
    // Synthetic SK blobs with the exact wire shape (verified against
    // Go's parser in the differential check, not just here).
    let mut ed = vec![0u8; 32];
    ed.copy_from_slice(b"0123456789abcdef0123456789abcdef");
    let mut blob = Vec::new();
    put_string(&mut blob, ALGO_SKED25519.as_bytes());
    put_string(&mut blob, &ed);
    put_string(&mut blob, b"ssh:test");
    let line = format!("{ALGO_SKED25519} {}", b64_encode(&blob));
    assert_eq!(canonical(&line).0, format!("{line}\n"));
}

fn unhex(hex: &str) -> Vec<u8> {
    let bytes = hex.as_bytes();
    assert!(bytes.len().is_multiple_of(2));
    let value = |c: u8| match c {
        b'0'..=b'9' => c - b'0',
        b'a'..=b'f' => c - b'a' + 10,
        b'A'..=b'F' => c - b'A' + 10,
        _ => panic!("bad hex"),
    };
    bytes
        .chunks_exact(2)
        .map(|pair| value(pair[0]) << 4 | value(pair[1]))
        .collect()
}

#[test]
fn generator_points_parse_and_remain_uncompressed() {
    let vectors = [
        (
            EcdsaCurve::NistP256,
            32,
            "6b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296",
            "4fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5",
            "ffffffff00000001000000000000000000000000ffffffffffffffffffffffff",
        ),
        (
            EcdsaCurve::NistP384,
            48,
            "aa87ca22be8b05378eb1c71ef320ad746e1d3b628ba79b9859f741e082542a385502f25dbf55296c3a545e3872760ab7",
            "3617de4a96262c6f5d9e98bf9292dc29f8f41dbd289a147ce9da3113b5f0b8c00a60b1ce1d7e819d7a431d7c90ea0e5f",
            "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffeffffffff0000000000000000ffffffff",
        ),
        (
            EcdsaCurve::NistP521,
            66,
            "c6858e06b70404e9cd9e3ecb662395b4429c648139053fb521f828af606b4d3dbaa14b5e77efe75928fe1dc127a2ffa8de3348b3c1856a429bf97e7e31c2e5bd66",
            "011839296a789a3bc0045c8a5fb42c7d1bd998f54449579b446817afbd17273e662c97ee72995ef42640c550b9013fad0761353c7086a272c24088be94769fd16650",
            "01ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
        ),
    ];

    for (curve, coord_len, x, y, prime_hex) in vectors {
        let mut point = vec![0x04];
        for coordinate in [x, y] {
            let raw = unhex(coordinate);
            point.extend(vec![0; coord_len - raw.len()]);
            point.extend(raw);
        }
        assert!(valid_ecdsa_point(curve, &point));

        // Each named curve rejects a point from either other
        // curve, and both compressed and hybrid encodings stay refused.
        for other in [
            EcdsaCurve::NistP256,
            EcdsaCurve::NistP384,
            EcdsaCurve::NistP521,
        ] {
            if other != curve {
                assert!(!valid_ecdsa_point(other, &point));
            }
        }

        let mut compressed = vec![0x02];
        compressed.extend_from_slice(&point[1..1 + coord_len]);
        assert!(!valid_ecdsa_point(curve, &compressed));

        let mut hybrid = point.clone();
        hybrid[0] = 0x06;
        assert!(!valid_ecdsa_point(curve, &hybrid));
        assert!(!valid_ecdsa_point(curve, &point[..point.len() - 1]));

        let mut overlong = point.clone();
        overlong.push(0);
        assert!(!valid_ecdsa_point(curve, &overlong));
        assert!(!valid_ecdsa_point(curve, &[0x00]));

        let mut off_curve = point.clone();
        off_curve[1] ^= 1;
        assert!(!valid_ecdsa_point(curve, &off_curve));

        let prime = unhex(prime_hex);
        assert_eq!(prime.len(), coord_len);
        for coordinate_start in [1, 1 + coord_len] {
            let mut at_prime = vec![0; 1 + 2 * coord_len];
            at_prime[0] = 0x04;
            at_prime[coordinate_start..coordinate_start + coord_len].copy_from_slice(&prime);
            assert!(!valid_ecdsa_point(curve, &at_prime));
            at_prime[coordinate_start..coordinate_start + coord_len].fill(0xff);
            assert!(!valid_ecdsa_point(curve, &at_prime));
        }
    }
}

#[test]
fn sk_ecdsa_p256_validates_point_and_curve_id() {
    let point = ordinary_p256_point();
    let mut blob = Vec::new();
    put_string(&mut blob, ALGO_SKECDSA.as_bytes());
    put_string(&mut blob, b"nistp256");
    put_string(&mut blob, &point);
    put_string(&mut blob, b"ssh:test");
    let line = format!("{ALGO_SKECDSA} {}", b64_encode(&blob));
    assert_eq!(canonical(&line).0, format!("{line}\n"));

    let mut infinity = vec![0; point.len()];
    infinity[0] = 0x04;
    let mut off_curve = infinity.clone();
    off_curve[1] = 1;
    for invalid_point in [infinity, off_curve] {
        let mut blob = Vec::new();
        put_string(&mut blob, ALGO_SKECDSA.as_bytes());
        put_string(&mut blob, b"nistp256");
        put_string(&mut blob, &invalid_point);
        put_string(&mut blob, b"ssh:test");
        assert!(
            parse_authorized_key(format!("{ALGO_SKECDSA} {}", b64_encode(&blob)).as_bytes())
                .is_err()
        );
    }

    let mut blob = Vec::new();
    put_string(&mut blob, ALGO_SKECDSA.as_bytes());
    put_string(&mut blob, b"nistp384");
    put_string(&mut blob, &point);
    put_string(&mut blob, b"ssh:test");
    assert!(
        parse_authorized_key(format!("{ALGO_SKECDSA} {}", b64_encode(&blob)).as_bytes()).is_err()
    );
}

#[test]
fn ecdsa_algorithm_rejects_wrong_curve_id() {
    let point = ordinary_p256_point();
    let mut blob = Vec::new();
    put_string(&mut blob, ALGO_ECDSA256.as_bytes());
    put_string(&mut blob, b"nistp384");
    put_string(&mut blob, &point);
    assert!(
        parse_authorized_key(format!("{ALGO_ECDSA256} {}", b64_encode(&blob)).as_bytes()).is_err()
    );
}

#[test]
fn comments_and_whitespace_tolerated_but_not_canonical() {
    // Parse accepts (comment captured, rest blank); the canonical form
    // drops them, so canonicalKeys' byte comparison rejects them while
    // the account path normalizes them.
    let (key, opts) = parse_authorized_key(format!("{ED} alice@host\n").as_bytes()).unwrap();
    assert_eq!(key.key_type, ALGO_ED25519);
    assert!(!opts);
    let (key, _) = parse_authorized_key(format!("   {ED}  \n").as_bytes()).unwrap();
    assert_eq!(
        marshal_authorized_key(&key.key_type, &key.blob),
        format!("{ED}\n")
    );
    // Declared type must equal the embedded type.
    assert!(parse_authorized_key(format!("ssh-rsa {}", &ED[12..]).as_bytes()).is_err());
}

#[test]
fn options_are_refused_and_multiline_key_selection_stays_bounded() {
    assert!(parse_authorized_key(format!("no-pty {ED}\n").as_bytes()).is_err());
    assert!(parse_authorized_key(format!("command=\"echo hi\",no-pty {ED}\n").as_bytes()).is_err());
    // Garbage lines are skipped; trailing data after the key rejects.
    assert!(parse_authorized_key(format!("garbage\n{ED}\n").as_bytes()).is_ok());
    assert!(parse_authorized_key(format!("{ED}\n{ED}\n").as_bytes()).is_err());
    assert!(parse_authorized_key(format!("{ED}\n  \n").as_bytes()).is_ok());
    assert!(parse_authorized_key(format!("{ED}\r\n").as_bytes()).is_ok());
    assert!(parse_authorized_key(b"").is_err());
    assert!(parse_authorized_key(b"ssh-ed25519").is_err());
    assert!(parse_authorized_key(b"ssh-ed25519 YWJj\n").is_err());
}

#[test]
fn invalid_keys_rejected() {
    // Truncated blob.
    assert!(parse_authorized_key(b"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA\n").is_err());
    // Trailing junk in blob.
    let mut blob = b64_decode(ED.split(' ').nth(1).unwrap()).unwrap();
    blob.push(0);
    assert!(
        parse_authorized_key(format!("ssh-ed25519 {}\n", b64_encode(&blob)).as_bytes()).is_err()
    );
    // Unknown algorithm.
    assert!(parse_authorized_key(b"ssh-foo AAAA\n").is_err());
    // Off-curve point (flip a byte in a valid P-256 key).
    let mut raw = b64_decode(ECDSA256.split(' ').nth(1).unwrap()).unwrap();
    let n = raw.len();
    raw[n - 10] ^= 0x01;
    assert!(
        parse_authorized_key(format!("ecdsa-sha2-nistp256 {}\n", b64_encode(&raw)).as_bytes())
            .is_err()
    );
}

#[test]
fn rsa_exponent_and_modulus_bounds_remain_host_policy() {
    let modulus = vec![0x80; 256];
    assert!(parse_public_key(&rsa_blob(&[3], &modulus)).is_ok());
    assert!(parse_public_key(&rsa_blob(&[4], &modulus)).is_err());
    assert!(parse_public_key(&rsa_blob(&[1, 0, 0, 1], &modulus)).is_err());
    assert!(parse_public_key(&rsa_blob(&[3], &vec![0x80; 2049])).is_err());
}

#[test]
fn dsa_width_and_positive_in_range_parameters_remain_host_policy() {
    let p = vec![0x80; 128];
    let q = vec![0x80; 20];
    assert!(parse_public_key(&dsa_blob(&p, &q, &[2], &[3])).is_ok());
    assert!(parse_public_key(&dsa_blob(&p, &q, &[], &[3])).is_err());
    assert!(parse_public_key(&dsa_blob(&p, &q, &p, &[3])).is_err());
    assert!(parse_public_key(&dsa_blob(&p, &q, &[2], &p)).is_err());
    assert!(parse_public_key(&dsa_blob(&p, &[0x40; 20], &[2], &[3])).is_err());
}

#[test]
fn base64_vectors() {
    assert_eq!(b64_decode("").unwrap_err(), "invalid base64".to_string());
    assert_eq!(b64_decode("QUI=").unwrap(), b"AB");
    assert_eq!(b64_decode("YR==").unwrap(), b"a");
    assert!(b64_decode("YQ==\r\n").is_err());
    assert_eq!(b64_decode("QUI"), Err("invalid base64".to_string()));
    assert_eq!(b64_decode("AB=C"), Err("invalid base64".to_string()));
    assert_eq!(b64_encode(b"AB"), "QUI=");
    assert_eq!(b64_encode_raw(b"AB"), "QUI");
    assert_eq!(b64_decode_go(b"Y\r\nQ==").unwrap(), b"a");
    assert_eq!(b64_decode_go(b"Zg\r\n").unwrap_err(), 2);
    assert_eq!(b64_decode_go(b"Y\r\n!").unwrap_err(), 3);
    assert_eq!(b64_decode_go(b"AA=").unwrap_err(), 3);
    assert_eq!(b64_decode_go(b"AB=\r\nC").unwrap_err(), 4);
    assert_eq!(b64_decode_go(b"AB==CD").unwrap_err(), 4);
}

#[test]
fn fingerprint_shape() {
    let (key, _) = parse_authorized_key(ED.as_bytes()).unwrap();
    let fp = fingerprint_sha256(&key.blob);
    assert!(fp.starts_with("SHA256:") && !fp.ends_with('='));
    assert_eq!(fp.len(), 7 + 43);
}
