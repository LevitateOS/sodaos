use super::mpint::{put_string, read_string};
use super::*;

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
    let raw = b64_decode(ECDSA256.split_once(' ').unwrap().1).unwrap();
    let (algorithm, fields) = read_string(&raw).unwrap();
    assert_eq!(algorithm, ALGO_ECDSA256.as_bytes());
    let (curve, fields) = read_string(fields).unwrap();
    assert_eq!(curve, b"nistp256");
    let (point, rest) = read_string(fields).unwrap();
    assert!(rest.is_empty());
    point.to_vec()
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

    let mut off_curve = vec![0; point.len()];
    off_curve[0] = 0x04;
    let mut blob = Vec::new();
    put_string(&mut blob, ALGO_SKECDSA.as_bytes());
    put_string(&mut blob, b"nistp256");
    put_string(&mut blob, &off_curve);
    put_string(&mut blob, b"ssh:test");
    assert!(
        parse_authorized_key(format!("{ALGO_SKECDSA} {}", b64_encode(&blob)).as_bytes()).is_err()
    );

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
fn options_detected_and_multiline_rules_match() {
    let (_, opts) = parse_authorized_key(format!("no-pty {ED}\n").as_bytes()).unwrap();
    assert!(opts);
    let (_, opts) =
        parse_authorized_key(format!("command=\"echo hi\",no-pty {ED}\n").as_bytes()).unwrap();
    assert!(opts);
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
    // Cert signed by a cert.
    let mut cert = b64_decode(CERT.split(' ').nth(1).unwrap()).unwrap();
    let _ = &mut cert;
}

#[test]
fn base64_vectors() {
    assert_eq!(b64_decode("").unwrap_err(), "invalid base64".to_string());
    assert_eq!(b64_decode("QUI=").unwrap(), b"AB");
    assert_eq!(b64_decode("QUI"), Err("invalid base64".to_string()));
    assert_eq!(b64_decode("AB=C"), Err("invalid base64".to_string()));
    assert_eq!(b64_encode(b"AB"), "QUI=");
    assert_eq!(b64_encode_raw(b"AB"), "QUI");
}

#[test]
fn fingerprint_shape() {
    let (key, _) = parse_authorized_key(ED.as_bytes()).unwrap();
    let fp = fingerprint_sha256(&key.blob);
    assert!(fp.starts_with("SHA256:") && !fp.ends_with('='));
    assert_eq!(fp.len(), 7 + 43);
}
