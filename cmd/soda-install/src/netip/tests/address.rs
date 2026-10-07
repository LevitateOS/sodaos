use super::super::*;

#[test]
fn standard_ip_grammar_and_display() {
    for (input, expected, v4, mapped) in [
        ("192.168.1.1", "192.168.1.1", true, false),
        ("2001:db8::1", "2001:db8::1", false, false),
        ("2001:db8::192.0.2.1", "2001:db8::c000:201", false, false),
        ("::192.0.2.1", "::c000:201", false, false),
        ("::ffff:192.168.1.2", "::ffff:192.168.1.2", false, true),
    ] {
        let addr = parse_addr(input).unwrap();
        assert_eq!(addr.to_string_go(), expected.as_bytes());
        assert_eq!(addr.is4(), v4);
        assert_eq!(addr.is4_in6(), mapped);
    }
    for invalid in [
        "192.168.01.1",
        "192.168.1",
        "1::2::3",
        "fe80::1%",
        "1.2.3.4%eth0",
    ] {
        assert!(parse_addr(invalid).is_err(), "accepted {invalid:?}");
    }
}

#[test]
fn zone_adapter_splits_first_percent_and_requires_nonempty_ipv6_zone() {
    let addr = parse_addr("fe80::1%eth0%extra").unwrap();
    assert_eq!(addr.zone(), b"eth0%extra");
    assert_eq!(addr.to_string_go(), b"fe80::1%eth0%extra");
    assert!(parse_addr("fe80::1%").is_err());
}

#[test]
fn private_policy_unmaps_ipv4_mapped_values() {
    for value in [
        "10.1.2.3",
        "172.16.1.2",
        "192.168.1.2",
        "::ffff:10.1.2.3",
        "fd00::1",
    ] {
        assert!(parse_addr(value).unwrap().is_private(), "{value}");
    }
    for value in ["100.64.0.1", "fe00::1", "::ffff:100.64.0.1"] {
        assert!(!parse_addr(value).unwrap().is_private(), "{value}");
    }
}
