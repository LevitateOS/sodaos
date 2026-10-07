use crate::config::encode_dashboard_config;
use crate::forgejo::decode_user;
use crate::secrets::{base64_encode, hex_encode};
use crate::system::FORGEJO_RESPONSE_LIMIT;

#[test]
fn dashboard_config_bytes_match_go_encoder() {
    // Byte-exact reference captured from Go's json.Encoder over
    // config.Config for the same logical record (see PR07 notes).
    let golden = "{\"listen\":\"127.0.0.1:8080\",\"forgejo_url\":\"https://forgejo.test\",\"forgejo_internal_url\":\"http://127.0.0.1:3000\",\"database_dsn_file\":\"/etc/soda/postgres/soda.dsn\",\"host_socket\":\"/run/soda/host.sock\",\"identity_socket\":\"\",\"grant_key_file\":\"/etc/soda/grant-key\",\"operator_id\":42,\"forgejo_background_socket\":\"\",\"forgejo_background_host_uid\":null,\"forgejo_background_credential_file\":\"\",\"forgejo_review_credential_file\":\"\",\"forgejo_merge_credential_file\":\"\",\"factory_intake_secret_file\":\"\",\"factory_publication_root\":\"\"}\n";
    let encoded = encode_dashboard_config(
        "127.0.0.1:8080",
        "https://forgejo.test",
        "http://127.0.0.1:3000",
        "/etc/soda/postgres/soda.dsn",
        "/run/soda/host.sock",
        "",
        "/etc/soda/grant-key",
        42,
    );
    assert_eq!(String::from_utf8(encoded).expect("utf8"), golden);
}

#[test]
fn forgejo_user_decode_matches_client_rules() {
    let user = decode_user(
        b"{\"id\": 42, \"login\": \"soda-tester\", \"is_admin\": true, \"extra\": [1]}",
    )
    .expect("valid");
    assert_eq!(user.id, 42);
    assert!(user.admin);
    let missing = decode_user(b"{}").expect("omitted scalar defaults");
    assert_eq!((missing.id, missing.admin), (0, false));
    let last_wins = decode_user(
        br#"{"id":42,"id":43,"is_admin":true,"is_admin":false,"future":{"version":1}}"#,
    )
    .expect("duplicates retain last recognized values and unknown fields are allowed");
    assert_eq!((last_wins.id, last_wins.admin), (43, false));
    assert!(decode_user(b"null").is_err());
    assert!(decode_user(b"{\"id\":null,\"is_admin\":false}").is_err());
    assert!(decode_user(b"{\"id\": 1} garbage").is_err());
    assert!(decode_user(b"{\"id\": \"42\", \"is_admin\": true}").is_err());
    assert!(decode_user(b"{\"id\": 42.5, \"is_admin\": true}").is_err());
    assert!(decode_user(b"{\"id\": 42, \"is_admin\": \"yes\"}").is_err());
    assert!(decode_user(b"{\"id\": 9223372036854775808}").is_err());
    assert!(decode_user(b"{\"id\": 42e0}").is_err());
    assert_eq!(
        decode_user(b"\xff").unwrap_err(),
        "invalid Forgejo response"
    );
    assert!(decode_user(b"[1,2]").is_err());
    let big = vec![b'x'; FORGEJO_RESPONSE_LIMIT + 1];
    assert_eq!(
        decode_user(&big).expect_err("oversize"),
        "forgejo response exceeds the supported size"
    );
}

#[test]
fn config_strings_keep_go_html_and_line_separator_escapes() {
    let encoded = encode_dashboard_config("<>&\u{2028}\u{2029}", "", "", "", "", "", "", 0);
    assert!(String::from_utf8(encoded)
        .unwrap()
        .starts_with(r#"{"listen":"\u003c\u003e\u0026\u2028\u2029","forgejo_url":"""#));
}

#[test]
fn base64_and_hex_vectors() {
    assert_eq!(hex_encode(b"\x00\xff\x10abc"), "00ff10616263");
    assert_eq!(base64_encode(b""), "");
    assert_eq!(base64_encode(b"f"), "Zg==");
    assert_eq!(base64_encode(b"fo"), "Zm8=");
    assert_eq!(base64_encode(b"foo"), "Zm9v");
    assert_eq!(base64_encode(b"foob"), "Zm9vYg==");
    assert_eq!(base64_encode(&[0u8; 32]).len(), 44);
}
