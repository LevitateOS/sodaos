use std::net::IpAddr;

use crate::origin::{activate_rejects_ip, check_browser_origin};

#[test]
fn ip_classifier_matches_cpython_oracle() {
    // (address, accepted?) from the interpreter's own verdicts.
    for (addr, accepted) in [
        ("192.168.1.5", true),
        ("10.0.0.1", true),
        ("172.16.0.1", true),
        ("8.8.8.8", false),
        ("127.0.0.1", false),
        ("0.0.0.0", false),
        ("224.0.0.1", false),
        ("255.255.255.255", true),
        ("169.254.1.1", true),
        ("100.64.0.1", true),
        ("192.0.2.1", true),
        ("198.51.100.1", true),
        ("203.0.113.1", true),
        ("198.18.0.1", true),
        ("192.0.0.1", true),
        ("192.0.0.9", false),
        ("192.0.0.10", false),
        ("240.0.0.1", true),
        ("0.0.0.1", true),
        ("::1", false),
        ("::", false),
        ("fe80::1", true),
        ("fc00::1", true),
        ("fd00::5", true),
        ("ff02::1", false),
        ("2001:db8::1", true),
        ("2001:4860:4860::8888", false),
        ("::ffff:192.168.1.1", true),
        ("::ffff:8.8.8.8", false),
        ("::ffff:0.0.0.0", true),
        ("::ffff:127.0.0.1", true),
        ("::ffff:10.0.0.1", true),
        ("::ffff:255.255.255.255", true),
        ("::ffff:192.0.2.1", true),
        ("::ffff:169.254.1.1", true),
        ("::ffff:100.64.0.1", true),
        ("100::1", true),
        ("64:ff9b::808:808", false),
        ("2001::1", true),
        ("2001:30::1", false),
        ("2002:c000:0200::1", true),
        ("64:ff9b:1::1", true),
        ("::ffff:0:101:101", false),
    ] {
        let ip: IpAddr = addr.parse().expect("fixture parses");
        assert_eq!(!activate_rejects_ip(&ip), accepted, "{addr}");
    }
}

#[test]
fn browser_origin_checks_match_activate_rules() {
    for url in [
        "https://forgejo.test",
        "https://forgejo.test/",
        "https://192.168.2.100",
        "https://[fd00::5]:8443/",
        "HTTPS://Forgejo.Test",
    ] {
        assert!(check_browser_origin(url).is_ok(), "{url}");
    }
    for url in [
        "http://forgejo.test",
        "https://forgejo.test/x",
        "https://user@forgejo.test",
        "https://u:p@forgejo.test",
        "https://forgejo.test?x",
        "https://forgejo.test#x",
        "https://",
        "https://forgejo.test\n",
        "not a url",
    ] {
        assert!(check_browser_origin(url).is_err(), "{url}");
    }
    // Empty userinfo fields are falsy in the Python check, so they pass.
    assert!(check_browser_origin("https://@forgejo.test").is_ok());
    assert!(check_browser_origin("https://:@forgejo.test").is_ok());
}
