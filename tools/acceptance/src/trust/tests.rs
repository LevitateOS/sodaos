use super::inline_data::INLINE_GZIP_LIMIT;
use super::*;
use crate::command::Remote;

#[test]
fn base64_round_trip_and_rejections() {
    for raw in [
        b"f".as_slice(),
        b"fo",
        b"foo",
        b"foob",
        b"fooba",
        b"foobar",
        b"\x00\xff binary \x01",
    ] {
        assert_eq!(decode_base64(&encode_base64(raw)).unwrap(), raw);
    }
    assert_eq!(decode_base64("").unwrap(), b"");
    assert_eq!(decode_base64("YR==").unwrap(), b"a"); // Go ignores unused trailing bits.
    assert!(decode_base64("Zm9v\n").is_err());
    for bad in ["a", "abc", "ab=c", "a===", "ab!d", "abcd=", "===="] {
        assert!(decode_base64(bad).is_err(), "{bad}");
    }
    assert_eq!(decode_base64("Zm9v").unwrap(), b"foo");
}

#[test]
fn data_uri_matrix() {
    assert_eq!(
        decode_data_uri("data:,soda-native-fixture%0A").unwrap(),
        b"soda-native-fixture\n"
    );
    assert_eq!(decode_data_uri("data:,a+b%41").unwrap(), b"a+bA");
    assert_eq!(decode_data_uri("data:;base64,Zm9v").unwrap(), b"foo");
    assert_eq!(
        decode_data_uri("data:text/plain;base64,Zm9v").unwrap(),
        b"foo"
    );
    assert_eq!(
        decode_data_uri("data:text/plain;charset=utf-8;base64,Zm9v").unwrap(),
        b"foo"
    );
    assert_eq!(decode_data_uri("data:,plain").unwrap(), b"plain");
    assert_eq!(
        decode_data_uri("plain").unwrap_err().to_string(),
        "inline data URI required"
    );
    assert_eq!(
        decode_data_uri("https://example.test/x")
            .unwrap_err()
            .to_string(),
        "inline data URI required"
    );
    assert_eq!(
        decode_data_uri("data:,a%zz").unwrap_err().to_string(),
        "invalid inline encoding"
    );
    assert_eq!(
        decode_data_uri("data:;base64,!!!").unwrap_err().to_string(),
        "invalid inline base64"
    );
    assert_eq!(decode_data_uri("data:,not-gzip").unwrap(), b"not-gzip");
}

#[test]
fn inline_data_compression_gates() {
    assert!(inline_data("data:,not-gzip", "gzip").is_err());
    assert_eq!(
        inline_data("data:,plain", "unknown")
            .unwrap_err()
            .to_string(),
        "unsupported inline compression"
    );
    assert_eq!(inline_data("data:,plain", "").unwrap(), b"plain");
}

#[test]
fn gzip_bounds_match_go() {
    let mut encoder = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
    std::io::Write::write_all(&mut encoder, b"payload").unwrap();
    let compressed = encoder.finish().unwrap();
    assert_eq!(gunzip_bounded(&compressed).unwrap(), b"payload");
    let mut second = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
    std::io::Write::write_all(&mut second, b" second").unwrap();
    let mut concatenated = compressed.clone();
    concatenated.extend(second.finish().unwrap());
    assert_eq!(gunzip_bounded(&concatenated).unwrap(), b"payload second");
    let mut trailing = compressed.clone();
    trailing.push(0);
    assert!(gunzip_bounded(&trailing).is_err());
    let mut bad_trailer = compressed.clone();
    *bad_trailer.last_mut().unwrap() ^= 1;
    assert!(gunzip_bounded(&bad_trailer).is_err());
    assert_eq!(
        gunzip_bounded(b"not gzip").unwrap_err().to_string(),
        "invalid compressed inline data"
    );
    assert_eq!(
        gunzip_bounded(&[0x1f, 0x8b, 0x00]).unwrap_err().to_string(),
        "invalid or oversized compressed inline data"
    );
    let big = vec![b'x'; (INLINE_GZIP_LIMIT + 1) as usize];
    let mut encoder = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
    std::io::Write::write_all(&mut encoder, &big).unwrap();
    let compressed = encoder.finish().unwrap();
    assert_eq!(
        gunzip_bounded(&compressed).unwrap_err().to_string(),
        "invalid or oversized compressed inline data"
    );
}

fn key_fixture(dir: &std::path::Path) -> (Vec<u8>, String) {
    let key = dir.join("id_ed25519");
    let status = std::process::Command::new("ssh-keygen")
        .args(["-t", "ed25519", "-f", key.to_str().unwrap(), "-N", "", "-q"])
        .stdin(std::process::Stdio::null())
        .status()
        .unwrap();
    assert!(status.success(), "ssh-keygen is required for trust tests");
    let private = std::fs::read(&key).unwrap();
    let public = std::fs::read_to_string(format!("{}.pub", key.display())).unwrap();
    (private, public.trim().to_string())
}

fn trust_fixture(dir: &std::path::Path) -> (String, Remote) {
    let (private, public) = key_fixture(dir);
    let hosts = dir.join("known_hosts");
    std::fs::write(&hosts, format!("[127.0.0.1]:22222 {public}\n")).unwrap();
    let input = dir.join("instance.ign");
    let body = format!(
        "{{\"ignition\":{{\"version\":\"3.5.0\"}},\"storage\":{{\"files\":[{{\"path\":\"/etc/hostname\",\"contents\":{{\"source\":\"data:,soda-native-fixture%0A\"}}}},{{\"path\":\"/etc/ssh/ssh_host_ed25519_key\",\"contents\":{{\"source\":\"data:;base64,{}\"}}}}]}}}}",
        encode_base64(&private)
    );
    std::fs::write(&input, body).unwrap();
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(&input, std::fs::Permissions::from_mode(0o600)).unwrap();
    let remote = Remote {
        user: "root".to_string(),
        host: "127.0.0.1".to_string(),
        key: String::new(),
        known_hosts: hosts.to_string_lossy().into_owned(),
        port: 22222,
        timeout: std::time::Duration::ZERO,
    };
    (input.to_string_lossy().into_owned(), remote)
}

#[test]
fn fixture_trust_happy_and_mismatch() {
    let dir = std::env::temp_dir().join(format!("soda-trust-test-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    let (input, remote) = trust_fixture(&dir);
    verify_fixture_trust(&input, "soda-native-fixture", &remote).unwrap();
    assert!(verify_fixture_trust(&input, "soda-native-other", &remote).is_err());
    let mut remote = remote;
    remote.port = 22223;
    assert!(verify_fixture_trust(&input, "soda-native-fixture", &remote).is_err());
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn fixture_trust_rejects_shape_violations() {
    let dir = std::env::temp_dir().join(format!("soda-trust-shape-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    let (input, remote) = trust_fixture(&dir);
    std::fs::write(&input, b"not json").unwrap();
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(&input, std::fs::Permissions::from_mode(0o600)).unwrap();
    assert_eq!(
        verify_fixture_trust(&input, "soda-native-fixture", &remote)
            .unwrap_err()
            .to_string(),
        "invalid private Ignition"
    );
    std::fs::write(&input, b"{\"storage\":{\"files\":[]}}").unwrap();
    std::fs::set_permissions(&input, std::fs::Permissions::from_mode(0o600)).unwrap();
    assert_eq!(
        verify_fixture_trust(&input, "soda-native-fixture", &remote)
            .unwrap_err()
            .to_string(),
        "matching fixture hostname and pinned Ed25519 host key required in Ignition"
    );
    std::fs::remove_dir_all(&dir).unwrap();
}
