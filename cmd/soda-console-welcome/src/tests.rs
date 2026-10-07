use crate::config::read_config;
use crate::origin::{valid_listen, valid_origin};
use std::io::Cursor;

#[test]
fn config_value_keeps_last_duplicate_and_unknown_values() {
    let value = read_config(Cursor::new(
        br#"{"listen": "1", "forgejo_url": 2, "listen": "3", "unknown": null}"#,
    ))
    .unwrap();
    let fields = value.as_object().unwrap();
    assert_eq!(
        fields.get("listen").and_then(serde_json::Value::as_str),
        Some("3")
    );
    assert!(fields.get("forgejo_url").unwrap().as_str().is_none());
    assert!(fields.get("unknown").unwrap().is_null());
    assert!(read_config(Cursor::new(br#"{"unclosed""#)).is_none());
    assert!(read_config(Cursor::new(br#"{"listen": "1"} trailing"#)).is_none());
    assert!(read_config(Cursor::new(
        br#"{"listen": NaN, "b": Infinity, "c": -Infinity}"#
    ))
    .is_none());
    let non_object = read_config(Cursor::new(b"[1,2]")).unwrap();
    assert!(non_object.as_object().is_none());
}

#[test]
fn string_escapes_decode_like_python() {
    let fields = read_config(Cursor::new(
        r#"{"listen": "x\"y\\z\/\b\f\n\r\té☃"}"#.as_bytes(),
    ))
    .unwrap();
    let fields = fields.as_object().unwrap();
    assert_eq!(
        fields.get("listen").and_then(serde_json::Value::as_str),
        Some("x\"y\\z/\u{8}\u{c}\n\r\té☃")
    );
    let fields = read_config(Cursor::new(r#"{"listen": "A𝄞B"}"#.as_bytes())).unwrap();
    assert_eq!(
        fields
            .as_object()
            .and_then(|fields| fields.get("listen"))
            .and_then(serde_json::Value::as_str),
        Some("A𝄞B")
    );
    assert!(read_config(Cursor::new(b"{\"listen\": \"\\ud800\"}")).is_none());
    assert!(read_config(Cursor::new(b"{\"listen\": \"line\nbreak\"}")).is_none());
}

#[test]
fn config_reader_accepts_exact_cap_and_rejects_cap_plus_one() {
    const LIMIT: usize = 64 * 1024;
    let base = br#"{"listen":"127.0.0.1:8080","forgejo_url":"https://forgejo.example.test"}"#;
    let mut exact = base.to_vec();
    exact.resize(LIMIT, b' ');
    assert!(read_config(Cursor::new(exact)).is_some());

    let mut overflow = base.to_vec();
    overflow.resize(LIMIT + 1, b' ');
    overflow[LIMIT] = 0xff;
    assert!(read_config(Cursor::new(overflow)).is_none());
}

#[test]
fn listen_shapes() {
    assert_eq!(
        valid_listen("127.0.0.1:8080"),
        Some(("127.0.0.1".to_string(), "8080".to_string()))
    );
    assert!(valid_listen("127.1.2.3:1").is_some());
    for bad in [
        "192.168.1.10:8080",
        "127.0.0.1:notaport",
        "127.0.0.1:0",
        "127.0.0.1:65536",
        "127.0.0.1:8080:extra",
        "42",
        "localhost:8080",
        "127.0.0.01:8080",
        "::1:8080",
        "",
    ] {
        assert!(valid_listen(bad).is_none(), "{bad:?}");
    }
}

#[test]
fn origin_shapes() {
    assert_eq!(
        valid_origin("https://forgejo.example.test"),
        Some("https://forgejo.example.test".to_string())
    );
    assert_eq!(
        valid_origin("https://forgejo.example.test/"),
        Some("https://forgejo.example.test".to_string())
    );
    assert!(valid_origin("HTTPS://forgejo.example.test").is_some());
    assert!(valid_origin("https://forgejo.example.test:8443").is_some());
    assert!(valid_origin("https://forgejo.example.test:0").is_some());
    assert!(valid_origin("https://forgejo.example.test/?").is_some());
    for bad in [
        "https://name:private-value@soda.example.test",
        "https://@soda.example.test",
        "https://soda.example.test:bad-port",
        "https://soda.example.test:65536",
        "https://soda.example.test:65536/..",
        "http://soda.example.test",
        "https://soda.example.test/path",
        "https://soda.example.test?q=1",
        "https://soda.example.test#frag",
        "https://soda.example.test/pa th",
        "https://",
        "https:///soda.example.test",
        "https://soda.example.test/%2e%2e",
        "not a url",
        "",
    ] {
        assert!(valid_origin(bad).is_none(), "{bad:?}");
    }
}
