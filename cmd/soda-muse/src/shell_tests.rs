use super::env::{go_base, go_clean, go_join, go_quote_rune};
use super::home_cfg::base64_encode;
use super::launch_json::parse_launch_exit;
use super::shell::{shell_request_json, validate_shell};
use super::terminal::muse_environment_with;
use super::ShellRequest;

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
fn shell_request_wire_matches_go() {
    assert_eq!(
        shell_request_json(&shell_fixture()),
        "{\"home\":\"/home/u\",\"config_home\":\"/home/u/.config\",\"term\":\"xterm\",\"connection_id\":\"conn1\",\"cwd\":\"/work\",\"args\":[\"a b\",\"c\\u003cd\"],\"tty\":true,\"cols\":80,\"rows\":24}"
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
    for bad in [
        "",
        "{",
        "{\"code\":}",
        "{\"code\":\"0\"}",
        "{\"code\":0,}",
        "[]",
    ] {
        assert!(
            parse_launch_exit(bad.as_bytes()).is_err(),
            "admitted {bad:?}"
        );
    }
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
fn go_clean_matches_filepath_cases() {
    for (input, want) in [
        ("", "."),
        ("/", "/"),
        ("//", "/"),
        ("/a/b/../c", "/a/c"),
        ("/../a", "/a"),
        ("a/./b", "a/b"),
        ("a/../../b", "../b"),
        ("/run/soda-muse/", "/run/soda-muse"),
        ("/home/u/", "/home/u"),
    ] {
        assert_eq!(go_clean(input), want, "input {input:?}");
    }
    assert_eq!(go_join("/home/u/", ".config"), "/home/u/.config");
    assert_eq!(go_join("/", ".config"), "/.config");
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
