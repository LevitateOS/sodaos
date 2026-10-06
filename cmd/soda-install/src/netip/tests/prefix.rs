use super::super::*;

#[test]
fn oracle_prefix_vectors() {
    // Oracle: `PREFIX` + `MP` lines.
    for bad in [
        "10.0.0.0/33",
        "10.0.0.0/-1",
        "10.0.0.0/",
        "10.0.0.0",
        "010.0.0.0/8",
        "10.0.0.0/08",
        "10.0.0.0/8 ",
        " 10.0.0.0/8",
        "10.0.0.0/+8",
        "10.0.0.0/8/8",
        "fe80::1%eth0/64",
        "10.0.0.0/00",
        "10.0.0.0/000",
        "10.0.0.0/0008",
        "fd00::/129",
        "fd00::/00",
    ] {
        assert!(parse_prefix(bad).is_err(), "accepted {bad:?}");
    }
    for (value, masked) in [
        ("10.89.0.0/24", true),
        ("10.89.0.1/24", false),
        ("0.0.0.0/0", true),
        ("fd00::/64", true),
        ("fd00::1/64", false),
        ("10.0.0.0/0", false),
        ("::/0", true),
        ("1.2.3.4/32", true),
        ("1.2.3.4/31", true),
        ("fd00::/128", true),
        ("::ffff:1.2.3.4/128", true),
        ("::ffff:1.2.3.4/96", false),
    ] {
        let prefix = parse_prefix(value).unwrap_or_else(|err| panic!("parse {value:?}: {err}"));
        assert_eq!(prefix == prefix.masked(), masked, "{value:?} masked");
    }
    let tailscale = parse_prefix("100.64.0.0/10").unwrap();
    assert!(tailscale.contains(&parse_addr("100.90.1.2").unwrap()));
    assert!(!tailscale.contains(&parse_addr("192.168.1.1").unwrap()));
    assert!(!tailscale.contains(&parse_addr("fd00::1").unwrap()));
    // Zoned addresses never match: prefixes strip zones.
    let v6 = parse_prefix("fd00::/64").unwrap();
    assert!(v6.contains(&parse_addr("fd00::1").unwrap()));
    assert!(!v6.contains(&parse_addr("fd00::1%eth0").unwrap()));
}

#[test]
fn prefix_error_text_matches_go() {
    let err = parse_prefix("10.0.0.0").unwrap_err();
    assert_eq!(err.text(), "netip.ParsePrefix(\"10.0.0.0\"): no '/'");
    let err = parse_prefix("10.0.0.0/33").unwrap_err();
    assert_eq!(
        err.text(),
        "netip.ParsePrefix(\"10.0.0.0/33\"): prefix length out of range"
    );
    let err = parse_prefix("10.0.0.0/08").unwrap_err();
    assert_eq!(
        err.text(),
        "netip.ParsePrefix(\"10.0.0.0/08\"): bad bits after slash: \"08\""
    );
    let err = parse_prefix("fe80::1%eth0/64").unwrap_err();
    assert_eq!(
        err.text(),
        "netip.ParsePrefix(\"fe80::1%eth0/64\"): IPv6 zones cannot be present in a prefix"
    );
    let err = parse_prefix("999/8").unwrap_err();
    assert_eq!(
        err.text(),
        "netip.ParsePrefix(\"999/8\"): ParseAddr(\"999\"): unable to parse IP"
    );
}
