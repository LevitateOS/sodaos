use super::execution::muse_environment_with;
use super::launch_wire::{base64_encode, parse_launch_exit, shell_request_json};
use super::paths::{go_base, go_join, go_quote_rune, is_clean_absolute_path};
use super::shell::validate_shell;
use super::shell::ShellRequest;

fn shell_fixture() -> ShellRequest {
    ShellRequest {
        cwd: String::from("/work"),
        args: vec![String::from("a b"), String::from("c<d")],
        home: String::from("/home/u"),
        connection_id: String::from("conn1"),
        config_home: String::from("/home/u/.config"),
        term: String::from("xterm"),
        tty: true,
        cols: 80,
        rows: 24,
    }
}

#[test]
fn shell_request_wire_preserves_field_order_and_optional_fields() {
    assert_eq!(
        shell_request_json(&shell_fixture()),
        "{\"home\":\"/home/u\",\"config_home\":\"/home/u/.config\",\"term\":\"xterm\",\"connection_id\":\"conn1\",\"cwd\":\"/work\",\"args\":[\"a b\",\"c<d\"],\"tty\":true,\"cols\":80,\"rows\":24}"
    );
    let empty = ShellRequest {
        cwd: String::from("/w"),
        args: Vec::new(),
        home: String::new(),
        connection_id: String::new(),
        config_home: String::new(),
        term: String::new(),
        tty: false,
        cols: 0,
        rows: 0,
    };
    assert_eq!(
        shell_request_json(&empty),
        "{\"connection_id\":\"\",\"cwd\":\"/w\",\"args\":[],\"tty\":false,\"cols\":0,\"rows\":0}"
    );
}

#[test]
fn shell_request_roundtrips_html_and_control_characters() {
    let mut request = shell_fixture();
    request.cwd = String::from("/work/<>&\u{2028}\u{2029}");
    let encoded = shell_request_json(&request);
    let decoded: serde_json::Value = serde_json::from_str(&encoded).unwrap();
    assert_eq!(decoded["cwd"], request.cwd);
    assert_eq!(decoded["args"][1], "c<d");
}

#[test]
fn shell_validation_matches_go() {
    validate_shell(&shell_fixture()).unwrap();
    let mut bad = shell_fixture();
    bad.cwd = String::from("relative");
    assert!(validate_shell(&bad).is_err());
    let mut bad = shell_fixture();
    bad.term = "x".repeat(129);
    assert!(validate_shell(&bad).is_err());
    let mut bad = shell_fixture();
    bad.cols = 0;
    assert!(validate_shell(&bad).is_err());
    let mut bad = shell_fixture();
    bad.args = vec![String::from("x"); 257];
    assert!(validate_shell(&bad).is_err());
    let mut bad = shell_fixture();
    bad.args = vec![String::from("a\0b")];
    assert!(validate_shell(&bad).is_err());
    // Empty home/config_home stay optional.
    let mut ok = shell_fixture();
    ok.home.clear();
    ok.config_home.clear();
    validate_shell(&ok).unwrap();
}

#[test]
fn launch_exit_parsing_matches_go_unmarshal() {
    assert_eq!(
        parse_launch_exit(b"{\"code\":0}").unwrap(),
        (0, String::new())
    );
    assert_eq!(parse_launch_exit(b"{}").unwrap(), (0, String::new()));
    assert_eq!(
        parse_launch_exit(b"{\"code\":3}").unwrap(),
        (3, String::new())
    );
    assert_eq!(
        parse_launch_exit(b"{\"error\":\"denied\",\"code\":1}\n").unwrap(),
        (1, String::from("denied"))
    );
    assert_eq!(
        parse_launch_exit(b"{\"code\":1,\"code\":2,\"error\":\"first\",\"error\":\"last\"}")
            .unwrap(),
        (2, String::from("last"))
    );
    assert_eq!(
        parse_launch_exit(b"{\"future\":{\"array\":[1,{\"nested\":true}]}}").unwrap(),
        (0, String::new())
    );
    let mut deeply_nested = String::from("{\"future\":");
    deeply_nested.push_str(&"[".repeat(130));
    deeply_nested.push('0');
    deeply_nested.push_str(&"]".repeat(130));
    deeply_nested.push('}');
    assert!(deeply_nested.len() < 4096);
    assert_eq!(
        parse_launch_exit(deeply_nested.as_bytes()).unwrap(),
        (0, String::new())
    );
    for bad in [
        "",
        "{",
        "{\"code\":}",
        "{\"code\":\"0\"}",
        "{\"code\":0,}",
        "{\"code\":null}",
        "{\"error\":null}",
        "{\"code\":1.0}",
        "{}{}",
        "[]",
    ] {
        assert!(
            parse_launch_exit(bad.as_bytes()).is_err(),
            "admitted {bad:?}"
        );
    }
}

#[test]
fn launch_exit_rejects_malformed_unicode_boundary() {
    // H03-F4: a \u window ending inside a multibyte char must reject
    // through the established Err path, not panic the decoder.
    assert!(parse_launch_exit("{\"error\":\"\\u€é\"}".as_bytes()).is_err());
    assert!(parse_launch_exit(b"{\xff}").is_err());
    // Valid escapes still decode.
    assert_eq!(
        parse_launch_exit(b"{\"error\":\"A\\u0041\"}").unwrap(),
        (0, String::from("AA"))
    );
}

#[test]
fn base64_matches_standard_vectors() {
    for (raw, want) in [
        ("", ""),
        ("f", "Zg=="),
        ("fo", "Zm8="),
        ("foo", "Zm9v"),
        ("foob", "Zm9vYg=="),
        ("fooba", "Zm9vYmE="),
        ("foobar", "Zm9vYmFy"),
        ("{\"a\":1}", "eyJhIjoxfQ=="),
    ] {
        assert_eq!(base64_encode(raw.as_bytes()), want, "input {raw:?}");
    }
}

#[test]
fn native_path_join_and_root_admission() {
    assert_eq!(go_join("/home/u/", ".config"), "/home/u/.config");
    assert_eq!(go_join("/", ".config"), "/.config");
    assert!(is_clean_absolute_path("/run/soda-muse/0123456789abcdef"));
    for input in [
        "/run/soda-muse//0123456789abcdef",
        "/run/soda-muse/./0123456789abcdef",
        "/run/soda-muse/../soda-muse/0123456789abcdef",
        "/run/soda-muse/0123456789abcdef/",
    ] {
        assert!(!is_clean_absolute_path(input), "{input:?}");
    }
}

#[test]
fn go_quote_matches_invalid_byte_cases() {
    assert_eq!(go_quote_rune(b'X'), "'X'");
    assert_eq!(go_quote_rune(b'\''), "'\\''");
    assert_eq!(go_quote_rune(0x01), "'\\x01'");
}

#[test]
fn execution_id_rules_match_go() {
    assert_eq!(go_base("/run/soda-muse/abc"), "abc");
    // 32-hex passes the shape check (mkdir may fail without privilege).
    assert!("a".repeat(32).bytes().all(|b| b.is_ascii_hexdigit()));
    assert!("g".repeat(32).bytes().any(|b| !b.is_ascii_hexdigit()));
}

#[test]
fn muse_environment_orders_fixed_then_passthrough() {
    let env = muse_environment_with("/run/r", "/tmp/s", &|n| match n {
        "HOME" => Some(String::from("/home/u")),
        "TERM" => Some(String::new()),
        _ => None,
    });
    assert_eq!(
        env,
        vec![
            "PATH=/usr/local/bin:/usr/bin:/bin",
            "LANG=C.UTF-8",
            "TBH_CREDENTIAL_BACKEND=file",
            "XDG_CONFIG_HOME=/run/r/config",
            "XDG_STATE_HOME=/tmp/s/state",
            "XDG_CACHE_HOME=/tmp/s/cache",
            "XDG_DATA_HOME=/tmp/s/data",
            "TMPDIR=/tmp/s/tmp",
            "HOME=/home/u",
        ]
    );
}
