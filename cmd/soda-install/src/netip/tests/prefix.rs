use super::super::*;

#[test]
fn prefix_masks_and_contains_by_family() {
    for (input, masked) in [
        ("10.89.0.0/24", true),
        ("10.89.0.1/24", false),
        ("0.0.0.0/0", true),
        ("fd00::/64", true),
        ("fd00::1/64", false),
        ("::ffff:1.2.3.4/96", false),
    ] {
        let prefix = parse_prefix(input).unwrap();
        assert_eq!(prefix == prefix.masked(), masked, "{input}");
    }
    let cgnat = parse_prefix("100.64.0.0/10").unwrap();
    assert!(cgnat.contains(&parse_addr("100.90.1.2").unwrap()));
    assert!(!cgnat.contains(&parse_addr("192.168.1.1").unwrap()));
    assert!(!cgnat.contains(&parse_addr("fd00::1").unwrap()));
    let v6 = parse_prefix("fd00::/64").unwrap();
    assert!(v6.contains(&parse_addr("fd00::1").unwrap()));
    assert!(!v6.contains(&parse_addr("fd00::1%eth0").unwrap()));
}

#[test]
fn prefix_admission_checks_length_family_zone_and_mask() {
    for invalid in [
        "10.0.0.0/33",
        "10.0.0.0/-1",
        "10.0.0.0/",
        "10.0.0.0",
        "010.0.0.0/8",
        "10.0.0.0/08",
        "fe80::1%eth0/64",
        "fd00::/129",
    ] {
        assert!(parse_prefix(invalid).is_err(), "accepted {invalid:?}");
    }
    assert!(
        parse_prefix("10.89.0.0/24").unwrap() == parse_prefix("10.89.0.0/24").unwrap().masked()
    );
    assert!(
        parse_prefix("10.89.0.1/24").unwrap() != parse_prefix("10.89.0.1/24").unwrap().masked()
    );
}
