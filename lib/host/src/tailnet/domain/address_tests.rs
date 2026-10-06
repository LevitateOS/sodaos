use super::node_tests::{binding, prefs_doc, status_doc};
use super::*;

#[test]
fn status_peer_address_rules() {
    // Each address list replaces the fixture's single IPv4; all must be
    // canonical global-unicast forms or the whole status is unavailable.
    for (ips, ok) in [
        (r#"["100.64.0.2"]"#, true),
        (r#"["10.0.0.1","192.168.1.1"]"#, true),
        (r#"["fd7a:115c:a1e0::1"]"#, true),
        (r#"["2001:db8::1"]"#, true),
        (r#"["::ffff:1.2.3.4"]"#, true),
        (r#"["1:2:3:4:5:6:7:8"]"#, true),
        (r#"["fd7a::1%ETH0"]"#, true),
        (r#"["127.0.0.1"]"#, false),
        (r#"["0.0.0.0"]"#, false),
        (r#"["169.254.1.1"]"#, false),
        (r#"["224.0.0.1"]"#, false),
        (r#"["255.255.255.255"]"#, false),
        (r#"["::1"]"#, false),
        (r#"["::"]"#, false),
        (r#"["fe80::1"]"#, false),
        (r#"["ff02::1"]"#, false),
        (r#"["::ffff:127.0.0.1"]"#, false),
        (r#"["01.2.3.4"]"#, false),
        (r#"["FD7A::1"]"#, false),
        (r#"["fd7a:115c:a1e0:0:0:0:0:1"]"#, false),
        (r#"["1.2.3.4%eth0"]"#, false),
        (r#"["100.64.0.2%"]"#, false),
        (r#"["1.2.3.4 "]"#, false),
        (r#"["abc"]"#, false),
        (r#"[]"#, false),
        (r#"null"#, false),
    ] {
        let body = status_doc("Running").replace(r#"["100.64.0.2"]"#, ips);
        let r = project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding());
        if ok {
            assert_eq!(r.unwrap().0, "connected", "{ips}");
        } else {
            assert_eq!(r.unwrap_err(), ERR_UNAVAILABLE, "{ips}");
        }
    }
    // More than 16 addresses is unavailable.
    let many = "[".to_string() + &vec!["\"100.64.0.2\""; 17].join(",") + "]";
    let body = status_doc("Running").replace(r#"["100.64.0.2"]"#, &many);
    assert_eq!(
        project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap_err(),
        ERR_UNAVAILABLE
    );
}

#[test]
fn status_dns_name_rules() {
    let long_label = "a".repeat(64);
    let long_name = format!("{}.example.ts.net", "a".repeat(240));
    for (dns, ok) in [
        ("project.soda.ts.net.", true),
        ("Atlas.Example.ts.net.", true),
        ("  atlas.example.ts.net  ", true),
        ("a.b", true),
        ("a.b..", false),
        ("atlas.local", false),
        ("atlas.LOCAL.", false),
        ("nodot", false),
        ("", true),
        ("-a.b", false),
        ("a-.b", false),
        ("a.B_c", false),
        ("a..b", false),
        (".a.b", false),
        ("exa mple.ts.net", false),
        (&long_label, false),
        (&long_name, false),
    ] {
        let body = status_doc("Running").replace("project.soda.ts.net.", dns);
        let r = project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding());
        if ok {
            let (state, _, name) = r.unwrap();
            assert_eq!(state, "connected", "{dns:?}");
            assert_eq!(name, dns.trim().trim_end_matches('.').to_lowercase());
        } else {
            assert_eq!(r.unwrap_err(), ERR_UNAVAILABLE, "{dns:?}");
        }
    }
    assert_eq!(
        canonical_magic_dns_name("Atlas.Example.ts.net.").unwrap(),
        "atlas.example.ts.net"
    );
}

#[test]
fn dns_trims_nel_and_fold_covers_simple_fold_orbits() {
    // Go's strings.TrimSpace also trims U+0085 (NEL).
    assert_eq!(
        canonical_magic_dns_name("atlas.example.ts.net").unwrap(),
        "atlas.example.ts.net"
    );
    // Go's non-ASCII mates of ASCII letters: exactly ſ (S/s) and
    // Kelvin K (K/k). İ (U+0130) does NOT fold with i.
    assert!(fold_eq("ſ", "s"));
    assert!(fold_eq("S", "ſ"));
    assert!(fold_eq("Backendſtate", "BackendState"));
    assert!(fold_eq("\u{212a}", "k"));
    assert!(fold_eq("K", "\u{212a}"));
    assert!(!fold_eq("İ", "i"));
    assert!(!fold_eq("BackendState", "BackendStates"));
    assert!(!fold_eq("ß", "ss"));
}

#[test]
fn id_matchers() {
    assert!(valid_project_id(&("p".to_string() + &"a".repeat(24))));
    for bad in [
        "".to_string(),
        "p".to_string(),
        "P".to_string() + &"a".repeat(24),
        "p".to_string() + &"a".repeat(23),
        "p".to_string() + &"a".repeat(25),
        "p".to_string() + &"b".repeat(23) + "q",
        "p".to_string() + &"B".repeat(24),
    ] {
        assert!(!valid_project_id(&bad), "{bad}");
    }
    assert!(valid_container_id(&"c".repeat(64)));
    for bad in [
        "",
        &"c".repeat(63),
        &("c".repeat(63) + "C"),
        &"c".repeat(65),
    ] {
        assert!(!valid_container_id(bad), "{bad}");
    }
    assert!(valid_image_id(&"d".repeat(64)));
    assert!(valid_image_id(&("sha256:".to_string() + &"d".repeat(64))));
    for bad in [
        "",
        "sha256:",
        &("sha256:".to_string() + &"d".repeat(63)),
        &("SHA256:".to_string() + &"d".repeat(64)),
        &("e".repeat(65)),
    ] {
        assert!(!valid_image_id(bad), "{bad}");
    }
    assert_eq!(ERR_INVALID, "invalid Tailnet request");
    assert_eq!(ERR_CONFLICT, "tailnet revision or identity changed");
    assert_eq!(ERR_UNSUPPORTED, "tailnet runtime is not supported");
    assert_eq!(ERR_UNCONFIRMED, "tailnet outcome is unconfirmed");
    assert_eq!(ERR_UNAVAILABLE, "tailscale status is unavailable");
}

#[test]
fn rfc3339_vectors() {
    for (s, want) in [
        ("0001-01-01T00:00:00Z", Some(true)),
        ("0001-01-01T00:00:00.000000000Z", Some(true)),
        ("0001-01-01T00:00:00+00:00", Some(true)),
        ("0001-01-01T01:00:00+01:00", Some(true)),
        ("0000-12-31T23:00:00-01:00", Some(true)),
        ("0001-01-01T00:00:00.000000001Z", Some(false)),
        ("0001-01-01T00:30:00+01:00", Some(false)),
        ("2024-02-29T00:00:00Z", Some(false)),
        ("2000-02-29T12:30:45.123Z", Some(false)),
        ("1900-02-28T00:00:00Z", Some(false)),
        ("0000-01-01T00:00:00Z", Some(false)),
        ("9999-12-31T23:59:59Z", Some(false)),
        ("2024-01-01T00:00:00.5Z", Some(false)),
        // Go keeps nanosecond precision and ignores further digits.
        ("2024-01-01T00:00:00.1234567891Z", Some(false)),
        ("2024-01-01T00:00:00.0000000001Z", Some(false)),
        (
            "2024-01-01T00:00:00.123456789012345678901234567890Z",
            Some(false),
        ),
        (
            "0001-01-01T00:00:00.000000000000000000000000000000Z",
            Some(true),
        ),
        ("2024-01-01T00:00:00+24:00", Some(false)),
        ("2024-01-01T00:00:00+24:60", Some(false)),
        ("2024-01-01T00:00:00-24:60", Some(false)),
        ("2024-01-01T00:00:00+00:60", Some(false)),
        ("2024-01-01T00:00:00-00:00", Some(false)),
        ("2023-02-29T00:00:00Z", None),
        ("1900-02-29T00:00:00Z", None),
        ("2024-13-01T00:00:00Z", None),
        ("2024-00-10T00:00:00Z", None),
        ("2024-01-00T00:00:00Z", None),
        ("2024-01-32T00:00:00Z", None),
        ("2024-04-31T00:00:00Z", None),
        ("2024-01-01T24:00:00Z", None),
        ("2024-01-01T00:00:60Z", None),
        ("2024-01-01T00:00:00", None),
        ("2024-01-01T00:00:00.Z", None),
        ("2024-01-01T00:00:00.", None),
        ("2024-01-01T00:00:00+07", None),
        ("2024-01-01T00:00:00+0700", None),
        ("2024-01-01T00:00:00+07:00:00", None),
        ("2024-01-01T00:00:00+25:00", None),
        ("2024-01-01T00:00:00+24:61", None),
        ("2024-01-01T00:00:00+00:61", None),
        ("2024-01-01T00:00:00+99:99", None),
        ("2024-01-01t00:00:00z", None),
        ("2024-01-01T00:00:00z", None),
        ("2024-1-1T00:00:00Z", None),
        ("2024-01-01T1:00:00Z", Some(false)),
        ("2024-06-15T0:30:45.123456789+05:30", Some(false)),
        ("2024-01-01T123:00:00Z", None),
        ("2024-01-01T01:2:03Z", None),
        ("2024-01-01T01:02:3Z", None),
        ("24-01-01T00:00:00Z", None),
        ("20240-01-01T00:00:00Z", None),
        ("2024-01-01", None),
        ("2024-01-01T00:00", None),
        ("2024-01-01T00:00:00Z ", None),
        (" 2024-01-01T00:00:00Z", None),
        ("", None),
    ] {
        assert_eq!(parse_rfc3339_nano(s), want, "{s}");
    }
    // Go accepts 10+ fraction digits and keeps nanosecond precision.
    assert_eq!(
        parse_rfc3339_nano("2024-01-01T00:00:00.1234567890Z"),
        Some(false)
    );
}

#[test]
fn escape_vectors() {
    for (raw, want) in [
        ("", r#""""#),
        ("abc", r#""abc""#),
        ("a\"b\\c", r#""a\"b\\c""#),
        ("a\nb\rc\td", r#""a\nb\rc\td""#),
        ("\u{8}\u{c}", r#""\b\f""#),
        ("\u{1}\u{1f}", r#""\u0001\u001f""#),
        ("<>&", r#""\u003c\u003e\u0026""#),
        ("é☃", r#""é☃""#),
        ("\u{7f}", "\"\u{7f}\""),
    ] {
        assert_eq!(go_escape(raw), want, "{raw:?}");
    }
    assert_eq!(go_escape("\u{2028}"), r#""\u2028""#);
    assert_eq!(go_escape("\u{2029}"), r#""\u2029""#);
}
