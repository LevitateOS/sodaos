use crate::config::parse_top_object;
use crate::origin::{valid_listen, valid_origin};

#[test]
fn top_object_extracts_strings_last_wins() {
    let fields =
        parse_top_object(r#"{"listen": "1", "forgejo_url": 2, "listen": "3", "unknown": null}"#)
            .unwrap();
    assert_eq!(fields.get("listen"), Some(&Some("3".to_string())));
    assert_eq!(fields.get("forgejo_url"), Some(&None));
    assert!(!fields.contains_key("unknown"));
    assert!(parse_top_object(r#"{"unclosed""#).is_none());
    assert!(parse_top_object(r#"[1,2]"#).is_none());
    assert!(parse_top_object(r#"{"listen": "1"} trailing"#).is_none());
    assert!(parse_top_object(r#"{"listen": NaN, "b": Infinity, "c": -Infinity}"#).is_none());
}

#[test]
fn string_escapes_decode_like_python() {
    let fields = parse_top_object(r#"{"listen": "x\"y\\z\/\b\f\n\r\té☃"}"#).unwrap();
    assert_eq!(
        fields.get("listen"),
        Some(&Some("x\"y\\z/\u{8}\u{c}\n\r\té☃".to_string()))
    );
    let fields = parse_top_object(r#"{"listen": "A𝄞B"}"#).unwrap();
    assert_eq!(fields.get("listen"), Some(&Some("A𝄞B".to_string())));
    assert!(parse_top_object("{\"listen\": \"\\ud800\"}").is_none());
    assert!(parse_top_object("{\"listen\": \"line\nbreak\"}").is_none());
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
    assert!(valid_origin("https://forgejo.example.test/?").is_some());
    for bad in [
        "https://name:private-value@soda.example.test",
        "https://@soda.example.test",
        "https://soda.example.test:bad-port",
        "https://soda.example.test:65536",
        "http://soda.example.test",
        "https://soda.example.test/path",
        "https://soda.example.test?q=1",
        "https://soda.example.test#frag",
        "https://soda.example.test/pa th",
        "https://",
        "not a url",
        "",
    ] {
        assert!(valid_origin(bad).is_none(), "{bad:?}");
    }
}
