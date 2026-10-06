use super::*;

fn fields(url: &Url) -> (String, bool, String, String, String, String, String) {
    (
        url.scheme.clone(),
        url.user_present,
        String::from_utf8_lossy(&url.host).into_owned(),
        String::from_utf8_lossy(&url.hostname()).into_owned(),
        String::from_utf8_lossy(&url.path).into_owned(),
        url.raw_query.clone(),
        String::from_utf8_lossy(&url.fragment).into_owned(),
    )
}

fn parse_err(raw: &str) -> String {
    parse(raw).unwrap_err().text().to_string()
}

#[test]
fn oracle_url_vectors() {
    // Oracle: TestZZOracleURL `URL` lines (errors only; fields spot-checked).
    // Go rejects a leading space before `https:`: the first path segment
    // then contains a colon (`net/url` sources, verified by probe).
    for bad in [
        "https://192.168.1.5 ",
        "https://a b",
        " https://192.168.1.5",
    ] {
        assert!(parse(bad).is_err(), "accepted {bad:?}");
    }
    assert_eq!(
        parse(" https://192.168.1.5").unwrap_err().text,
        "parse \" https://192.168.1.5\": first path segment in URL cannot contain colon"
    );
    // "https://192.168.1.5:99999" parses (port unchecked); hostname strips.
    let url = parse("https://192.168.1.5:99999").unwrap();
    assert_eq!(url.hostname(), b"192.168.1.5");
    type UrlVector<'a> = (
        &'a str,
        &'a str,
        bool,
        &'a str,
        &'a str,
        &'a str,
        &'a str,
        &'a str,
    );
    let cases: &[UrlVector<'_>] = &[
        (
            "https://192.168.1.5",
            "https",
            false,
            "192.168.1.5",
            "192.168.1.5",
            "",
            "",
            "",
        ),
        (
            "https://192.168.1.5/",
            "https",
            false,
            "192.168.1.5",
            "192.168.1.5",
            "/",
            "",
            "",
        ),
        (
            "HTTPS://192.168.1.5",
            "https",
            false,
            "192.168.1.5",
            "192.168.1.5",
            "",
            "",
            "",
        ),
        (
            "https://192.168.1.2:444",
            "https",
            false,
            "192.168.1.2:444",
            "192.168.1.2",
            "",
            "",
            "",
        ),
        (
            "https://192.168.1.2/path",
            "https",
            false,
            "192.168.1.2",
            "192.168.1.2",
            "/path",
            "",
            "",
        ),
        (
            "https://root@192.168.1.2",
            "https",
            true,
            "192.168.1.2",
            "192.168.1.2",
            "",
            "",
            "",
        ),
        (
            "https://user:pass@192.168.1.5",
            "https",
            true,
            "192.168.1.5",
            "192.168.1.5",
            "",
            "",
            "",
        ),
        (
            "https://192.168.1.2?argument",
            "https",
            false,
            "192.168.1.2",
            "192.168.1.2",
            "",
            "argument",
            "",
        ),
        (
            "https://192.168.1.2#frag",
            "https",
            false,
            "192.168.1.2",
            "192.168.1.2",
            "",
            "",
            "frag",
        ),
        (
            "https://[fd00::123]",
            "https",
            false,
            "[fd00::123]",
            "fd00::123",
            "",
            "",
            "",
        ),
        (
            "https://[FD00::123]/",
            "https",
            false,
            "[FD00::123]",
            "FD00::123",
            "/",
            "",
            "",
        ),
        (
            "https://soda.example.test",
            "https",
            false,
            "soda.example.test",
            "soda.example.test",
            "",
            "",
            "",
        ),
        (
            "http://192.168.1.5",
            "http",
            false,
            "192.168.1.5",
            "192.168.1.5",
            "",
            "",
            "",
        ),
        (
            "https://192.168.1.5:443",
            "https",
            false,
            "192.168.1.5:443",
            "192.168.1.5",
            "",
            "",
            "",
        ),
        (
            "https://192.168.1.5//",
            "https",
            false,
            "192.168.1.5",
            "192.168.1.5",
            "//",
            "",
            "",
        ),
        (
            "//192.168.1.5",
            "",
            false,
            "192.168.1.5",
            "192.168.1.5",
            "",
            "",
            "",
        ),
        ("https:///path", "https", false, "", "", "/path", "", ""),
        (
            "https://[::1]:8080/x?y#z",
            "https",
            false,
            "[::1]:8080",
            "::1",
            "/x",
            "y",
            "z",
        ),
        ("https://", "https", false, "", "", "", "", ""),
        ("https://?x", "https", false, "", "", "", "x", ""),
        ("https:foo", "https", false, "", "", "", "", ""),
        ("//", "", false, "", "", "", "", ""),
    ];
    for (raw, scheme, user, host, hostname, path, query, fragment) in cases {
        let url = parse(raw).unwrap_or_else(|err| panic!("parse {raw:?}: {err}"));
        assert_eq!(
            fields(&url),
            (
                (*scheme).to_string(),
                *user,
                (*host).to_string(),
                (*hostname).to_string(),
                (*path).to_string(),
                (*query).to_string(),
                (*fragment).to_string()
            ),
            "{raw:?}"
        );
    }
    // Bare paths parse without a scheme (and fail origin checks later).
    let url = parse("192.168.1.5").unwrap();
    assert_eq!((url.scheme.as_str(), text(&url.path)), ("", "192.168.1.5"));
    // Rootless `scheme:rest` is opaque, never an error.
    let url = parse("a:b/c").unwrap();
    assert_eq!((url.scheme.as_str(), text(&url.path)), ("a", ""));
}

fn text(bytes: &[u8]) -> &str {
    std::str::from_utf8(bytes).unwrap()
}

#[test]
fn url_error_text_matches_go() {
    assert_eq!(
        parse_err("https://192.168.1.5 "),
        "parse \"https://192.168.1.5 \": invalid character \" \" in host name"
    );
    assert_eq!(
        parse_err("https://a b"),
        "parse \"https://a b\": invalid character \" \" in host name"
    );
    assert_eq!(
        parse_err("https://h/%zz"),
        "parse \"https://h/%zz\": invalid URL escape \"%zz\""
    );
    assert_eq!(
        parse_err("https://h/%"),
        "parse \"https://h/%\": invalid URL escape \"%\""
    );
    assert_eq!(
        parse_err("https://h/a%2"),
        "parse \"https://h/a%2\": invalid URL escape \"%2\""
    );
    assert_eq!(
        parse_err("https://h#%zz"),
        "parse \"https://h#%zz\": invalid URL escape \"%zz\""
    );
    assert_eq!(
        parse_err("https://h:ab/"),
        "parse \"https://h:ab/\": invalid port \":ab\" after host"
    );
    assert_eq!(
        parse_err("https://[::1"),
        "parse \"https://[::1\": missing ']' in host"
    );
    assert_eq!(
        parse_err("https://a[::1]/"),
        "parse \"https://a[::1]/\": invalid IP-literal"
    );
    assert_eq!(
        parse_err("https://[999]/"),
        "parse \"https://[999]/\": invalid host: ParseAddr(\"999\"): unable to parse IP"
    );
    assert_eq!(
        parse_err("https://[1.2.3.4]/"),
        "parse \"https://[1.2.3.4]/\": invalid IP-literal"
    );
    assert_eq!(
        parse_err("https://a:b:c/"),
        "parse \"https://a:b:c/\": invalid port \":b:c\" after host"
    );
    assert_eq!(
        parse_err("foo://a:b:c/"),
        "parse \"foo://a:b:c/\": invalid port \":c\" after host"
    );
    assert_eq!(
        parse_err("https://a b@c/"),
        "parse \"https://a b@c/\": net/url: invalid userinfo"
    );
    assert_eq!(
        parse_err("https://u%zz@h/"),
        "parse \"https://u%zz@h/\": invalid URL escape \"%zz\""
    );
    // Host errors win over userinfo errors.
    assert_eq!(
        parse_err("https://u%zz@[::1/"),
        "parse \"https://u%zz@[::1/\": missing ']' in host"
    );
    assert_eq!(
        parse_err("cache_object:foo/bar"),
        "parse \"cache_object:foo/bar\": first path segment in URL cannot contain colon"
    );
    assert_eq!(parse_err(":foo"), "parse \":foo\": missing protocol scheme");
    assert_eq!(
        parse_err("https://h/\x7f"),
        "parse \"https://h/\\x7f\": net/url: invalid control character in URL"
    );
}

#[test]
fn url_edge_behavior_matches_go() {
    // Control bytes in the fragment are accepted: Go screens only the
    // pre-fragment input.
    let url = parse("https://h#\x7f").unwrap();
    assert_eq!(url.fragment, b"\x7f");
    // `///` without a scheme is a path, not an authority.
    let url = parse("///x").unwrap();
    assert_eq!((text(&url.host), text(&url.path)), ("", "///x"));
    // Fields store Go's decoded bytes.
    let url = parse("https://m%C3%BCnchen.de/a%2fb#x%41").unwrap();
    assert_eq!(text(&url.host), "münchen.de");
    assert_eq!(text(&url.path), "/a/b");
    assert_eq!(text(&url.fragment), "xA");
    // Zones unescape `%25` to a literal `%`.
    let url = parse("https://[fe80::1%25en0]/").unwrap();
    assert_eq!(text(&url.host), "[fe80::1%en0]");
    assert_eq!(text(&url.hostname()), "fe80::1%en0");
    // A bare `:` is a valid empty port.
    let url = parse("https://h:/").unwrap();
    assert_eq!((text(&url.host), text(&url.hostname())), ("h:", "h"));
    // Non-http schemes split ports at the last colon.
    let url = parse("foo://a:b:80/").unwrap();
    assert_eq!((text(&url.host), text(&url.hostname())), ("a:b:80", "a:b"));
}
